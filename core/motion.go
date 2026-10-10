package core

import (
	"strings"

	"github.com/k2sebeom/balpyo/soml"
)

// 이 파일은 등장(`enter`)의 값 층이다.
//
// 등장 계획은 슬라이드마다 **한 번**, 콘텐츠 해석이 끝난 뒤 정한다: `enter`를 가진 요소(또는
// `<slide enter>`를 물려받은 직속 자식)를 문서 순서로 세어 순서 · 지연을 붙인다. 지연은 테마의 박자에서
// 나온다 — 첫 등장은 Delay 뒤, 그다음부터 Stagger씩. 두 렌더러는 그 값을 읽기만 한다: 웹은 그 요소의
// 박스를, PPTX는 그 요소 아래의 객체(가장 가까운 등장이 그 요소인 것)를 같은 시각에 나타낸다.

// resolveEntrances는 슬라이드의 등장을 확정한다.
func (r *resolver) resolveEntrances(slide *soml.Slide) {
	inherited := soml.EnterEffect("")
	if slide.Enter != "" {
		if effect, ok := r.enterValue(slide.Enter, slide.EnterSpan); ok {
			inherited = effect
		}
	}
	order := 0
	var walk func(node soml.Node, direct bool)
	walk = func(node soml.Node, direct bool) {
		elem, ok := node.(*soml.Element)
		if !ok || elem.Name == soml.ConnectorElement {
			return
		}
		effect := soml.EnterEffect("")
		if direct {
			effect = inherited
		}
		if attr, ok := elem.Attr(soml.EnterAttr); ok {
			effect = ""
			if r.hasValue(attr) {
				if v, ok := r.enterValue(attr.Value, attr.ValueSpan); ok {
					effect = v
				}
			}
		}
		if effect != "" && effect != soml.EnterNone {
			motion := r.theme.Motion
			if motion == nil {
				r.errorf(elem.NameSpan, "<%s> has enter=%q, but theme %q defines no motion — add \"motion\": { \"duration\": 600, \"stagger\": 90, \"delay\": 150 } to its tokens.json",
					elem.Name, string(effect), r.theme.Name)
			} else {
				elemAppearance(elem).Enter = &soml.Entrance{
					Effect: effect, Order: order,
					Delay: motion.Delay + order*motion.Stagger, Duration: motion.Duration,
				}
				order++
			}
		}
		for _, child := range elem.Children {
			walk(child, false)
		}
	}
	for _, node := range slide.Content {
		walk(node, true)
	}
}

// enterValue는 등장 효과 이름을 본다 (닫힌 어휘).
func (r *resolver) enterValue(raw string, span soml.Span) (soml.EnterEffect, bool) {
	for _, v := range soml.EnterEffects() {
		if raw == v {
			return soml.EnterEffect(v), true
		}
	}
	list := strings.Join(soml.EnterEffects(), ", ")
	r.errorf(span, "unknown enter %q%s (available values: %s)", raw, didYouMean(raw, list), list)
	return "", false
}

// resolveTransition은 이 장에 들어올 때의 전환을 확정한다 — 슬라이드의 값이 테마의 기본값을 덮어쓴다
// (`none`은 끈다). 확정 값은 fade · push · 빈 값이다.
func (r *resolver) resolveTransition(slide *soml.Slide) {
	transition := soml.SlideTransition("")
	duration := 0
	if m := r.theme.Motion; m != nil {
		transition, duration = m.Transition, m.TransitionDuration
	}
	if raw := string(slide.Transition); raw != "" {
		known := false
		for _, v := range soml.SlideTransitions() {
			known = known || raw == v
		}
		if !known {
			list := strings.Join(soml.SlideTransitions(), ", ")
			r.errorf(slide.TransitionSpan, "unknown transition %q%s (available values: %s)", raw, didYouMean(raw, list), list)
		} else {
			transition = soml.SlideTransition(raw)
			if duration == 0 {
				// 테마가 박자를 정하지 않았어도 슬라이드가 고른 전환은 산다 — 길이는 PowerPoint의 기본(빠름)이다.
				duration = 500
			}
		}
	}
	if transition == soml.TransitionNone {
		transition = ""
	}
	slide.Transition = transition
	slide.TransitionDuration = 0
	if transition != "" {
		slide.TransitionDuration = duration
	}
}
