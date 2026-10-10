package core

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/k2sebeom/balpyo/soml"
)

// 이 파일은 **배치 속성의 값 해석**이다 (컨테이너·자식 배치 속성) — 파서가
// 이름만 본 배치 속성을 확정된 배치 값(soml.ContainerLayout·soml.Placement)으로 눕힌다.
//
// 텍스트 외관 해석(resolveText)과 나란한 자리에 있다. 두 해석이 한 요소에 함께 얹히는 것은
// 정상이다 — `<p grow="1" size="30">`은 외관 채널과 배치 채널을 동시에 쓴 것이고, 두 채널이
// 서로를 모르는 것이 이 언어의 경계다 (배치는 외관이 아니다).
//
// 속성이 19개여도 값 문법은 몇 개의 표다: 길이(수치·`%`·`fill`·공간 토큰), 닫힌 어휘
// (align/justify/self), 불리언(wrap), 목록(cols/rows/pad), 비율(ratio), 정수
// (grow/shrink/span/order), 그리고 탈출구(style — style.go). 헬퍼가 문법을 소유하고
// 속성 표는 헬퍼를 고르기만 한다.
//
// 단위 계약: **무단위 수치는 논리 스테이지 px**이고 `%`만 예외다.
// 그 외 단위(px/pt/em/rem 접미)는 파스 에러다 — `size`가 이미 같은 태도이므로 진단 문구의
// 결을 맞춘다 (numberSuffix를 공유한다).

// resolveLayout은 요소의 배치를 확정한다.
//
// **요소 종류와 무관하게 같은 코드가 해석한다** — `<text>`·블록·리스트·컨테이너·`<spacer>`가
// 같은 표를 지난다. 자식 배치 속성은 한 벌이므로(soml.ChildPlacementAttrs) 종류마다 표를
// 복제하면 속성이 하나 늘 때 어느 종류가 빠졌는지 사람이 세어야 한다.
//
// parent는 트리에서의 부모 요소다 (슬라이드 직속이면 nil). `span`이 `<grid>`의 자식에서만
// 뜻을 갖는다는 규칙을 검증할 수 있는 유일한 자리가 이 순회다.
func (r *resolver) resolveLayout(elem, parent *soml.Element) {
	var container *soml.ContainerLayout
	var containerNames []string
	if soml.KindOf(elem.Name) == soml.KindLayoutContainer {
		// 컨테이너는 속성이 하나도 없어도 Container를 갖는다: 배치 흐름(flex 방향·grid)은
		// 속성이 아니라 태그가 정하는 값이고, 렌더러가 "이 요소가 컨테이너인가"를 다시
		// 판단하지 않아야 컨테이너 CSS 전부가 이 구조체 하나에서 나온다.
		container = &soml.ContainerLayout{Flow: soml.FlowOf(elem.Name)}
		containerNames = soml.ContainerAttrs(elem.Name)
	}
	// 텍스트 요소(`<text>`·블록·리스트 컨테이너)는 **가로 크기**를 자기 박스에 받는다
	// (soml.TakesTextSize가 그 집합을 소유한다). 값 문법은 컨테이너·도형과 같은
	// 헬퍼(boxAttr)가 해석하므로 무단위 px·`%`·`fill`이 같은 뜻이다 (공간 토큰은 크기에
	// 쓸 수 없다 — gap·pad 전용이다).
	var box soml.Box
	takesSize := soml.TakesTextSize(elem.Name)
	var placement soml.Placement
	// at과 x/y는 같은 축의 위치를 정하는 두 채널이다 — 함께 쓰면 어느 쪽이 이기는지 저자가
	// 외워야 하므로 묻는다 (variant/frame을 함께 받지 않은 것과 같은 기준). 충돌을
	// 여기서 한 번 판정해 진단을 한 번만 내고, 충돌한 채널은 확정 배치에 싣지 않는다.
	anchorConflict := r.placementConflict(elem)
	for _, attr := range elem.Attrs {
		// 이름 목록은 카탈로그(soml)에서 온다 — core가 자기 목록을 들면 파서가 연 속성과
		// 값 층이 해석하는 속성이 갈라지고, 갈라진 쪽은 조용히 무시된다.
		switch {
		case container != nil && containsName(containerNames, attr.Name):
			r.containerAttr(elem, container, attr)
		case takesSize && containsName(soml.TextSizeAttrs(), attr.Name):
			// 컨테이너 분기 **뒤**에 온다: 컨테이너는 자기 목록에 크기 7종을 다 갖고 있어
			// 여기까지 오지 않는다 (같은 속성이 두 자리에 담기면 렌더러가 둘 다 읽는다).
			if !r.boxAttr(&box, attr) {
				r.catalogGap(elem, attr)
			}
		case containsName(soml.ChildPlacementAttrs(), attr.Name):
			r.placementAttr(elem, &placement, attr, parent, anchorConflict)
		}
	}
	if container == nil && box == (soml.Box{}) && placement == (soml.Placement{}) {
		// 배치 속성이 없는 요소는 배치 필드가 **nil로 남는다**. 렌더러의 산출물이 배치
		// 해석이 붙었다는 이유로 달라지지 않는다는 뜻이다 (빈 구조체를 채우면 렌더러가 "정하지 않음"과
		// 0을 구별할 수 없다).
		return
	}
	appearance := elemAppearance(elem)
	appearance.Container = container
	if box != (soml.Box{}) {
		appearance.Box = &box
	}
	if placement != (soml.Placement{}) {
		appearance.Placement = &placement
	}
}

// containerAttr은 컨테이너 속성 하나를 해석한다 (표 순서 = 레퍼런스 속성 표 순서).
//
// 박스 속성(`pad`·크기·`ratio`)은 이 switch에 없다 — **도형과 공유하는 헬퍼**(boxAttr)가
// 소유한다. 이어 붙인 카탈로그가 한 벌인 것(soml.BoxAttrs)의 값 층 쪽 절반이고, 여기에 표를
// 다시 적으면 같은 표기가 두 곳에서 해석된다.
//
// 틀린 값은 확정 배치에 실리지 않는다 — 외관 해석과 같은 정책이다 (틀린 size가 역할
// 기본값을 남기는 것처럼, 틀린 gap은 "gap 없음"으로 남는다). 그래야 진단 하나가 다른
// 속성의 진단을 삼키지 않고 저자가 한 번에 다 고친다.
func (r *resolver) containerAttr(elem *soml.Element, container *soml.ContainerLayout, attr soml.Attr) {
	switch attr.Name {
	case "gap":
		if length, ok := r.length(attr, lengthSyntax{tokens: true}); ok {
			container.Gap = &length
		}
	case "align":
		if align, ok := r.boxAlign(attr); ok {
			container.Align = align
		}
	case "justify":
		if justify, ok := r.boxJustify(attr); ok {
			container.Justify = justify
		}
	case "wrap":
		if wrap, ok := r.wrap(attr); ok {
			container.Wrap = wrap
		}
	case "cols":
		if tracks, ok := r.tracks(attr); ok {
			container.Cols = &tracks
		}
	case "rows":
		if tracks, ok := r.tracks(attr); ok {
			container.Rows = &tracks
		}
	default:
		if !r.boxAttr(&container.Box, attr) {
			r.catalogGap(elem, attr)
		}
	}
}

// boxAttr은 박스 속성 하나를 해석한다 — 안쪽 여백·크기·비율 (soml.BoxAttrs, 표 순서 =
// 레퍼런스 속성 표 순서). 목록에 없는 속성이면 거짓을 돌려주고 호출자가 판단한다.
//
// **레이아웃 컨테이너와 `<shape>`가 이 함수를 공유한다.** 크기와 안쪽 여백은 컨테이너의
// 성질이 아니라 박스의 성질이고(soml.Box), 두 요소가 같은 표기를 같은 뜻으로 받는다. 값
// 문법을 도형용으로 다시 쓰면 같은 문자열이 두 곳에서 해석되고, 그 둘이 갈라지는 것은 배치
// 하나가 웹에서만 먹는 것으로만 드러난다. `fill`이 "해당 축의 100%"인 계약도
// 그대로 공유된다.
func (r *resolver) boxAttr(box *soml.Box, attr soml.Attr) bool {
	switch attr.Name {
	case "pad":
		if pad, ok := r.pad(attr); ok {
			box.Pad = &pad
		}
	case "w":
		if length, ok := r.length(attr, lengthSyntax{fill: true}); ok {
			box.Width = &length
		}
	case "h":
		if length, ok := r.length(attr, lengthSyntax{fill: true}); ok {
			box.Height = &length
		}
	case "min-w":
		if length, ok := r.length(attr, lengthSyntax{fill: true}); ok {
			box.MinWidth = &length
		}
	case "min-h":
		if length, ok := r.length(attr, lengthSyntax{fill: true}); ok {
			box.MinHeight = &length
		}
	case "max-w":
		if length, ok := r.length(attr, lengthSyntax{fill: true}); ok {
			box.MaxWidth = &length
		}
	case "max-h":
		if length, ok := r.length(attr, lengthSyntax{fill: true}); ok {
			box.MaxHeight = &length
		}
	case "ratio":
		if ratio, ok := r.ratio(attr); ok {
			box.Ratio = &ratio
		}
	default:
		return false
	}
	return true
}

// placementAttr은 자식 배치 속성 하나를 해석한다 (표 순서 = 레퍼런스 자식 속성 표).
//
// anchorConflict는 at과 x/y를 함께 썼는지다 (resolveLayout이 한 번 판정한다). 참이면 두
// 채널(at·x·y)을 확정 배치에 싣지 않는다 — 진단은 이미 한 번 났고, 충돌한 값을 실으면
// 저자가 고친 뒤의 결과를 예측할 수 없다 (틀린 값을 싣지 않는 값 층의 정책과 같다).
func (r *resolver) placementAttr(elem *soml.Element, placement *soml.Placement, attr soml.Attr, parent *soml.Element, anchorConflict bool) {
	switch attr.Name {
	case "grow":
		if n, ok := r.nonNegativeInt(attr); ok {
			placement.Grow = &n
		}
	case "shrink":
		if n, ok := r.nonNegativeInt(attr); ok {
			placement.Shrink = &n
		}
	case "basis":
		// basis에 `fill`을 열지 않았다: `fill`은 크기 속성(w/h/min-*/max-*)의
		// 값이고, `flex-basis: 100%`를 뜻하는 `fill`은 주축에서만 크기가 되어 교차축에서
		// 저자의 기대와 갈라진다. 백분율이 필요하면 `basis="100%"`로 적는다.
		if length, ok := r.length(attr, lengthSyntax{}); ok {
			placement.Basis = &length
		}
	case "span":
		if n, ok := r.span(attr, parent); ok {
			placement.Span = &n
		}
	case "self":
		if align, ok := r.boxAlign(attr); ok {
			placement.Self = align
		}
	case "order":
		// order만 음수를 받는다 — CSS order의 음수는 유효하고 "먼저 보이게"의 관용 표기다.
		if n, ok := r.integer(attr); ok {
			placement.Order = &n
		}
	case "at":
		if anchor, ok := r.stackAnchor(attr, parent, anchorConflict); ok {
			placement.At = anchor
		}
	case "x", "y":
		if length, ok := r.coord(attr, parent, anchorConflict); ok {
			if attr.Name == "x" {
				placement.X = &length
			} else {
				placement.Y = &length
			}
		}
	case soml.StyleAttr:
		if style, ok := r.placementStyle(attr); ok {
			placement.Style = style
		}
	default:
		r.catalogGap(elem, attr)
	}
}

// placementConflict는 at과 x/y를 같은 자식에 함께 썼는지 판정하고, 그렇다면 진단한다.
//
// 둘 다 자리를 정하는 채널이라(at은 9지점 앵커, x/y는 절대 오프셋) 겹치면 CSS에서
// 절대 배치된 grid 자식의 `place-self`가 정적 위치에만 작용한다는 규칙이 결과를 정하는데,
// 그것은 저자가 외울 이유가 없는 규칙이다. 어느 쪽이 이기는지 정하는 대신 **묻는다** —
// variant와 frame을 함께 받지 않은 것과 같은 기준이다. 진단은 두 속성의 위치를
// 모두 가리켜 저자가 어느 하나를 지울지 고르게 한다.
func (r *resolver) placementConflict(elem *soml.Element) bool {
	at, hasAt := elem.Attr("at")
	x, hasX := elem.Attr("x")
	y, hasY := elem.Attr("y")
	if !hasAt || (!hasX && !hasY) {
		return false
	}
	coord := x
	if !hasX {
		coord = y
	}
	r.errorf(at.NameSpan, "at cannot be combined with %s (%s is at %s) — both set the position, so use only one (at is a 9-point anchor, x/y are absolute offsets)",
		coord.Name, coord.Name, coord.NameSpan.Start)
	return true
}

// catalogGap은 카탈로그(soml)가 연 속성을 값 층이 모르는 상태다 — 엔진 버그다.
// 조용히 지나가면 저자가 쓴 배치·외관이 산출물에서 사라지므로 소리를 낸다 (렌더러의 "렌더하는
// 매핑이 없다" 가드와 같은 태도).
//
// **파서가 거절한 이름에는 소리를 내지 않는다.** 파서는 어휘 밖 속성을 진단한 뒤에도 트리에
// 남겨 두므로(에러 복구) 값 층이 그것을 다시 만난다 — 그때 이 가드가 울면 저자는 오타 하나에
// 진단 둘을 받고 둘째가 "엔진 버그"라고 말한다. 자기 오타를 제품의 결함으로 읽게 만드는
// 진단이므로, 두 경우를 태그의 카탈로그(soml.ElementAttrs)로 가른다. 컨테이너 경로는 카탈로그
// 목록에 없는 이름을 애초에 지나가므로 이 정책이 요소 종류를 넘어 하나가 된다.
func (r *resolver) catalogGap(elem *soml.Element, attr soml.Attr) {
	if !containsName(soml.ElementAttrs(elem.Name), attr.Name) {
		return
	}
	r.errorf(attr.NameSpan, "no value parser for the %s attribute — the catalog (soml) and the value layer (core) disagree (engine bug)", attr.Name)
}

// lengthSyntax는 길이 값이 어떤 형태를 받는지다. 속성마다 받는 형태가 다르다 — 공간 토큰은 `gap`/`pad`의 값이고
// `fill`은 크기 속성의 값이다.
type lengthSyntax struct {
	// tokens는 공간 토큰 이름을 먼저 보는지다 (`gap="xl"`).
	tokens bool
	// fill은 `fill`을 받는지다 (크기 속성).
	fill bool
	// signed는 음수를 받는지다. 좌표(`x`/`y`)에만 참이다 — 크기(폭·gap·basis)의 음수는 CSS가
	// 무효로 버리는 선언이지만, 좌표의 음수는 정당하다 (박스 밖으로 반쯤 걸친 배지는 오버레이의
	// 가장 흔한 장면이다). 한 함수가 두 규칙(크기의 음수 금지 · 좌표의 음수 허용)을 갖게 되므로
	// 회귀가 이 축에서 난다 — 크기 속성의 음수 금지 테스트가 그대로 남아 있어야 한다.
	signed bool
}

// length는 속성 하나를 길이로 해석한다.
func (r *resolver) length(attr soml.Attr, syntax lengthSyntax) (soml.Length, bool) {
	if !r.hasValue(attr) {
		return soml.Length{}, false
	}
	return r.lengthValue(attr.Name, attr.Value, attr.ValueSpan, syntax)
}

// lengthValue는 길이 값 하나를 해석한다 — 속성이 아니라 (값, 구간) 쌍을 받는 이유는
// `pad="sm md"`처럼 값 하나가 길이 여러 개를 담기 때문이다. 진단은 문제가 된 **조각**을
// 가리킨다 (soml.Span.Slice).
func (r *resolver) lengthValue(name, raw string, span soml.Span, syntax lengthSyntax) (soml.Length, bool) {
	if syntax.fill && raw == "fill" {
		// **`fill` = 해당 축의 100%다.** `flex-grow: 1`로 눕히는 길과 저울질했고 버렸다:
		// `grow`가 별도 속성으로 이미 있어서 같은 결과를 내는 표기가 둘이 되고, 두 표기의
		// 시맨틱이 교차축과 grid에서 미묘하게 다르다 (grow는 주축의 남은 공간 분배이고
		// 100%는 축의 크기다).
		return soml.Percent(100), true
	}
	// 토큰 표를 **수치 판정보다 먼저** 본다. `startsNumeric`으로 먼저 갈라 놓으면 이름이
	// 숫자로 시작하는 토큰(`2xl` — 공간 토큰 어휘의 유일한 그런 이름)이 영원히 수치 경로로
	// 가서 "단위를 떼라"는 엉뚱한 진단을 받는다. 표에 있는 이름은 토큰이고, 없으면 수치
	// 문법이 판정한다 — 그래야 진단의 정확도(`gap="16px"`은 단위, `gap="huge"`는 모르는
	// 토큰)가 어휘의 표기와 무관하게 유지된다.
	if syntax.tokens {
		if token, ok := r.theme.Space(raw); ok {
			// 토큰 **이름**이 확정 값에 남는다 — 웹이 `var(--balpyo-space-xl)`로 내야 테마를
			// 바꿨을 때 여백이 따라 움직인다 (색이 schemeClr 참조로 남는 것과 같은 근거,
			// soml.Length.Token). px로만 눕히면 `gap="24"`와 `gap="md"`가 같은 것이 되고,
			// 렌더러가 px를 표에서 거꾸로 찾는 길은 그 둘을 구별할 수 없다.
			return soml.SpaceLength(raw, token.Px), true
		}
		if !startsNumeric(raw) {
			names := strings.Join(r.theme.SpaceNames(), ", ")
			r.errorf(span, "unknown space token %q%s (available tokens: %s, or a number or percentage)",
				raw, didYouMean(raw, names), names)
			return soml.Length{}, false
		}
	} else if !startsNumeric(raw) {
		if _, ok := r.theme.Space(raw); ok {
			// 크기에 공간 토큰을 쓰는 것은 흔한 혼동이다 — 어휘 목록을 던지는 대신 그 값이
			// 어디에 쓰이는지 짚어 준다 (align의 left/right 진단과 같은 태도).
			r.errorf(span, "space token %q is only for gap and pad — %s takes %s", raw, name, lengthForms(syntax))
			return soml.Length{}, false
		}
	}
	value := raw
	percent := strings.HasSuffix(value, "%")
	if percent {
		value = strings.TrimSuffix(value, "%")
	}
	if unit := numberSuffix(value); unit != "" {
		r.errorf(span, "%s is a unitless number in logical px (drop the %[3]q from %[2]q) — px, pt and em suffixes are not allowed",
			name, raw, unit)
		return soml.Length{}, false
	}
	n, err := strconv.ParseFloat(value, 64)
	if err != nil {
		r.errorf(span, "%s value is not a number (%q) — use %s", name, raw, lengthForms(syntax))
		return soml.Length{}, false
	}
	if n < 0 && !syntax.signed {
		// 음수 길이는 CSS가 무효로 버리는 선언이다 — 조용히 사라지므로 값 층에서 막는다.
		// 좌표(signed)는 예외다: 음수 오프셋이 정당하다 (lengthSyntax.signed의 근거).
		r.errorf(span, "%s cannot be negative (%q)", name, raw)
		return soml.Length{}, false
	}
	return soml.Length{Value: n, Percent: percent}, true
}

// lengthForms는 진단에 실을 허용 형태 목록이다 — 문법 정의(lengthSyntax)와 메시지가
// 갈라지지 않도록 손으로 쓰지 않는다.
func lengthForms(syntax lengthSyntax) string {
	forms := "a number, percentage"
	if syntax.tokens {
		forms += ", space token"
	}
	if syntax.fill {
		forms += ", fill"
	}
	return forms
}

// pad는 1~4값 축약을 4변으로 푼다 — 규칙은 CSS `padding`과 같다 (상·우·하·좌). 표기를 새로
// 발명하지 않는 이유는 저자가 이미 아는 규칙이기 때문이고, 여기서 푸는 이유는 렌더러가
// 축약을 다시 해석하지 않아야 하기 때문이다.
func (r *resolver) pad(attr soml.Attr) (soml.Pad, bool) {
	if !r.hasValue(attr) {
		return soml.Pad{}, false
	}
	parts := splitValues(attr.Value)
	switch {
	case len(parts) == 0:
		r.errorf(attr.ValueSpan, "pad has no value (%q)", attr.Value)
		return soml.Pad{}, false
	case len(parts) > 4:
		r.errorf(attr.ValueSpan, "pad takes 1 to 4 values (%q has %d) — the same top/right/bottom/left shorthand rule as CSS padding",
			attr.Value, len(parts))
		return soml.Pad{}, false
	}
	sides := make([]soml.Length, 0, 4)
	for _, part := range parts {
		length, ok := r.lengthValue(attr.Name, part.text, part.span(attr), lengthSyntax{tokens: true})
		if !ok {
			return soml.Pad{}, false
		}
		sides = append(sides, length)
	}
	switch len(sides) {
	case 1:
		return soml.Pad{Top: sides[0], Right: sides[0], Bottom: sides[0], Left: sides[0]}, true
	case 2:
		return soml.Pad{Top: sides[0], Right: sides[1], Bottom: sides[0], Left: sides[1]}, true
	case 3:
		return soml.Pad{Top: sides[0], Right: sides[1], Bottom: sides[2], Left: sides[1]}, true
	default:
		return soml.Pad{Top: sides[0], Right: sides[1], Bottom: sides[2], Left: sides[3]}, true
	}
}

// tracks는 `cols`/`rows`를 해석한다 — 수치 하나는 트랙 **개수**이고 그 밖은 트랙 목록이다.
//
// 목록을 문자열로 흘려보내지 않는 이유: 검증하지 않은 문자열이 CSS로 나가면 그것이 곧
// `style` 탈출구인데, 탈출구는 경고를 동반해야 한다 (style.go). 여기 값은 경고 없이
// 통과하므로 엔진이 문법을 알고 있어야 한다.
func (r *resolver) tracks(attr soml.Attr) (soml.Tracks, bool) {
	if !r.hasValue(attr) {
		return soml.Tracks{}, false
	}
	parts := splitValues(attr.Value)
	if len(parts) == 0 {
		r.errorf(attr.ValueSpan, "%s has no value (%q)", attr.Name, attr.Value)
		return soml.Tracks{}, false
	}
	if len(parts) == 1 && isDigits(parts[0].text) {
		// **무단위 정수 하나는 트랙 개수다** (`cols="4"` → `repeat(4, 1fr)`).
		// 200px 트랙 하나로 읽는 길과 저울질했고 개수를 골랐다: 격자를 균등 분할하는 것이
		// 이 속성의 압도적인 용례다. 트랙 하나를 크기로 적으려는 저자는 `1fr`·`100%`·`auto`처럼
		// 형태를 붙인다 — 그 애매함은 문서가 말한다.
		n, err := strconv.Atoi(parts[0].text)
		if err != nil || n < 1 {
			r.errorf(parts[0].span(attr), "the track count of %s is an integer of at least 1 (%q)", attr.Name, attr.Value)
			return soml.Tracks{}, false
		}
		return soml.Tracks{Count: n}, true
	}
	list := make([]soml.Track, 0, len(parts))
	for _, part := range parts {
		track, ok := r.track(attr.Name, part.text, part.span(attr))
		if !ok {
			return soml.Tracks{}, false
		}
		list = append(list, track)
	}
	return soml.Tracks{List: list}, true
}

// track은 트랙 목록의 항목 하나를 해석한다. 어휘는 수치(px)·`Nfr`·`N%`·`auto` 넷으로 닫는다 —
// `minmax()`·`fit-content()` 같은 CSS 함수는 열지 않았다: 두 타깃 중 웹만 이해하는 문법이
// 아니라(지오메트리 패스가 재므로 좌표는 남는다) 값 문법을 재구현하는 범위의 문제다.
// 필요해지면 `style` 탈출구가 그 자리다.
func (r *resolver) track(name, raw string, span soml.Span) (soml.Track, bool) {
	if raw == "auto" {
		return soml.Track{Kind: soml.TrackAuto}, true
	}
	value, kind := raw, soml.TrackPx
	switch {
	case strings.HasSuffix(raw, "fr"):
		value, kind = strings.TrimSuffix(raw, "fr"), soml.TrackFr
	case strings.HasSuffix(raw, "%"):
		value, kind = strings.TrimSuffix(raw, "%"), soml.TrackPercent
	}
	n, err := strconv.ParseFloat(value, 64)
	if err != nil {
		r.errorf(span, "unknown track value %[2]q in %[1]s (available forms: a number, Nfr, N%%, auto, or a single integer track count)",
			name, raw)
		return soml.Track{}, false
	}
	if n < 0 {
		r.errorf(span, "tracks of %s cannot be negative (%q)", name, raw)
		return soml.Track{}, false
	}
	return soml.Track{Kind: kind, Value: n}, true
}

// ratio는 `W:H` 정수 쌍을 해석한다. 하나의 부동소수로 눕히지 않는 이유는 soml.Ratio의
// 주석에 있다 — 저자가 쓴 비율이 산출물에 그대로 나가야 한다.
func (r *resolver) ratio(attr soml.Attr) (soml.Ratio, bool) {
	if !r.hasValue(attr) {
		return soml.Ratio{}, false
	}
	raw := attr.Value
	w, h, found := strings.Cut(raw, ":")
	if !found {
		r.errorf(attr.ValueSpan, "ratio is a W:H pair of integers (%q) — e.g. 16:9", raw)
		return soml.Ratio{}, false
	}
	width, errW := strconv.Atoi(w)
	height, errH := strconv.Atoi(h)
	if errW != nil || errH != nil {
		r.errorf(attr.ValueSpan, "ratio is a W:H pair of integers (%q) — e.g. 16:9", raw)
		return soml.Ratio{}, false
	}
	if width <= 0 || height <= 0 {
		// 0이나 음수 비율은 CSS가 무효로 버린다 — 조용히 사라지는 선언을 값 층에서 막는다.
		r.errorf(attr.ValueSpan, "W and H of ratio must be greater than 0 (%q)", raw)
		return soml.Ratio{}, false
	}
	return soml.Ratio{W: width, H: height}, true
}

// boxAlign은 교차축 정렬을 해석한다 (`align`·`self`가 같은 어휘를 쓴다 — soml.BoxAligns).
func (r *resolver) boxAlign(attr soml.Attr) (soml.BoxAlign, bool) {
	if !r.hasValue(attr) {
		return "", false
	}
	raw := attr.Value
	for _, align := range soml.BoxAligns() {
		if string(align) == raw {
			return align, true
		}
	}
	// 저자는 CSS 습관대로 flex- 접두를 쓴다 — 대응하는 이름을 짚어 준다 (고치는 방법이
	// 진단 안에 있어야 한 번에 끝난다).
	if logical, ok := map[string]soml.BoxAlign{
		"flex-start": soml.BoxAlignStart,
		"flex-end":   soml.BoxAlignEnd,
	}[raw]; ok {
		r.errorf(attr.ValueSpan, "%s takes unprefixed names — instead of %q, use %q", attr.Name, raw, string(logical))
		return "", false
	}
	r.errorf(attr.ValueSpan, "unknown %s %q%s (available values: %s)", attr.Name, raw, didYouMean(raw, boxAlignList()), boxAlignList())
	return "", false
}

// boxJustify는 주축 분배를 해석한다. 어휘는 CSS의 `space-*` 접두를 뗀 이름이다.
func (r *resolver) boxJustify(attr soml.Attr) (soml.BoxJustify, bool) {
	if !r.hasValue(attr) {
		return "", false
	}
	raw := attr.Value
	for _, justify := range soml.BoxJustifies() {
		if string(justify) == raw {
			return justify, true
		}
	}
	if logical, ok := map[string]soml.BoxJustify{
		"flex-start":    soml.BoxJustifyStart,
		"flex-end":      soml.BoxJustifyEnd,
		"space-between": soml.BoxJustifyBetween,
		"space-around":  soml.BoxJustifyAround,
		"space-evenly":  soml.BoxJustifyEvenly,
	}[raw]; ok {
		r.errorf(attr.ValueSpan, "justify takes unprefixed names — instead of %q, use %q", raw, string(logical))
		return "", false
	}
	r.errorf(attr.ValueSpan, "unknown justify %q%s (available values: %s)", raw, didYouMean(raw, boxJustifyList()), boxJustifyList())
	return "", false
}

// wrap은 줄바꿈 허용을 해석한다.
//
// 두 표기를 받는다: 값 없는 속성(`<row wrap>`)과 `wrap="true"`. 전자는 `<slide skip>`의
// 선례이고(soml.Attr.HasValue) 후자는 레퍼런스 속성 표가 적은 값이다 —
// 한쪽만 받으면 문서와 언어가 어긋난다.
func (r *resolver) wrap(attr soml.Attr) (bool, bool) {
	if !attr.HasValue {
		return true, true
	}
	switch attr.Value {
	case "true":
		return true, true
	case "":
		r.errorf(attr.ValueSpan, "the wrap attribute is empty (remove it for no wrapping — the default)")
		return false, false
	}
	// `wrap="false"`를 받지 않는 이유: 줄바꿈 없음이 CSS 기본값이므로 끄는 표기는 속성을
	// 지우는 것이다 (hasValue 진단의 "지우면 기본값을 쓴다"와 같은 태도). 기본값을 명시하는
	// 표기를 열면 저자가 "줄바꿈이 켜졌는가"를 두 곳에서 확인해야 한다.
	r.errorf(attr.ValueSpan, "wrap is either a valueless attribute (<row wrap>) or wrap=\"true\" (%q) — no wrapping is the default, so remove the attribute to turn it off", attr.Value)
	return false, false
}

// integer는 정수 값을 해석한다 (`order`).
func (r *resolver) integer(attr soml.Attr) (int, bool) {
	if !r.hasValue(attr) {
		return 0, false
	}
	n, err := strconv.Atoi(attr.Value)
	if err != nil {
		r.errorf(attr.ValueSpan, "%s is an integer (%q)", attr.Name, attr.Value)
		return 0, false
	}
	return n, true
}

// nonNegativeInt는 0 이상 정수를 해석한다 (`grow`/`shrink`).
func (r *resolver) nonNegativeInt(attr soml.Attr) (int, bool) {
	n, ok := r.integer(attr)
	if !ok {
		return 0, false
	}
	if n < 0 {
		// CSS는 음수 flex-grow/flex-shrink를 무효로 버린다 — 조용히 사라지는 선언을 막는다.
		r.errorf(attr.ValueSpan, "%s cannot be negative (%q)", attr.Name, attr.Value)
		return 0, false
	}
	return n, true
}

// span은 `grid-column: span N`을 해석한다.
//
// **`<grid>`의 자식에서만 뜻이 있다**. 부모를 아는 자리가 이 순회뿐이라 검증도
// 여기 있다 — 파서는 부모를 모르는 자리에서 속성 표를 만들기 때문에 이름 층에서 막을 수
// 없었다. 검증하지 않고 넘기는 길과 저울질했고 막는 쪽을 골랐다: flex 컨테이너와 슬라이드
// 직속에서 `grid-column`은 CSS가 조용히 버리는 선언이고, 그 침묵이 닫힌 어휘가 막으려던
// 실패다 (파서가 `<row cols>`를 막은 것과 같은 근거, 층만 다르다).
func (r *resolver) span(attr soml.Attr, parent *soml.Element) (int, bool) {
	n, ok := r.integer(attr)
	if !ok {
		return 0, false
	}
	if n < 1 {
		r.errorf(attr.ValueSpan, "span is an integer of at least 1 (%q)", attr.Value)
		return 0, false
	}
	if parent == nil || parent.Name != soml.GridElement {
		r.errorf(attr.NameSpan, "span is only for children of <%s> (%s) — grid-column has no effect in a flex container",
			soml.GridElement, parentDesc(parent))
		return 0, false
	}
	return n, true
}

// parentDesc는 진단에 실을 부모 설명이다.
func parentDesc(parent *soml.Element) string {
	if parent == nil {
		return "this is a direct child of the slide"
	}
	return fmt.Sprintf("the parent is <%s>", parent.Name)
}

// stackAnchor는 `at`의 9지점 앵커를 해석한다.
//
// **`<stack>`의 자식에서만 뜻이 있다**. span과 같은 자리의 부모를 아는 검증이다 —
// 파서는 부모를 모르는 자리에서 속성 표를 만들기 때문에 이름 층에서 막을 수 없었다. flex/grid
// 컨테이너나 슬라이드 직속에서 `place-self`의 9지점은 뜻이 없어(단일 셀 grid가 아니다) 조용히
// 버려지고, 그 침묵이 닫힌 어휘가 막으려던 실패다.
//
// 검사 순서는 span과 같다: 값(어휘) 먼저, 부모 나중. `<row at="center">`은 어휘를 통과하고
// 부모에서 걸리고, `at="top"`(stack 안)은 어휘에서 걸린다 — 두 진단이 겹치지 않는다.
func (r *resolver) stackAnchor(attr soml.Attr, parent *soml.Element, conflict bool) (soml.StackAnchor, bool) {
	if conflict {
		return "", false
	}
	if !r.hasValue(attr) {
		return "", false
	}
	raw := attr.Value
	for _, anchor := range soml.StackAnchors() {
		if string(anchor) == raw {
			if parent == nil || parent.Name != soml.StackElement {
				r.errorf(attr.NameSpan, "at is only for children of <%s> (%s) — wrap it in <%s> if you need overlap",
					soml.StackElement, parentDesc(parent), soml.StackElement)
				return "", false
			}
			return anchor, true
		}
	}
	// 흔한 오타·짧은 이름·축이 뒤바뀐 이름은 올바른 9지점을 짚어 준다 — 고치는 방법이 진단
	// 안에 있어야 한 번에 끝난다 (boxAlign의 flex- 접두 안내와 같은 태도).
	if suggestion, ok := anchorTypo(raw); ok {
		r.errorf(attr.ValueSpan, "unknown at %q — did you mean %q? (9-point anchors: %s)",
			raw, string(suggestion), stackAnchorList())
		return "", false
	}
	r.errorf(attr.ValueSpan, "unknown at %q%s (available 9-point anchors: %s)", raw, didYouMean(raw, stackAnchorList()), stackAnchorList())
	return "", false
}

// coord는 `x`/`y`의 절대 오프셋을 해석한다.
//
// 값 문법은 수치(논리 px)·`%`·**음수**·소수다 (lengthSyntax{signed}). 공간 토큰(`x="md"`)과
// `fill`은 받지 않는다: 토큰은 여백·간격의 어휘라 테마를 따라 움직이는 것이 옳지만 좌표가
// 테마 교체로 움직이면 겹침이 깨지고, `fill`은 크기이지 위치가 아니다 (lengthSyntax{}가 그
// 구분을 표현한다).
//
// **`<stack>`의 자식이거나 canvas 슬라이드의 직속 자식에서만 뜻이 있다.** 후자는 부모가
// 요소가 아니라 슬라이드다 (parent == nil) — 그래서 슬라이드의 확정 layout(r.layout)을 본다.
// content 슬라이드의 직속 자식에서 막는 것이 요점이다: 지금 스테이지는 세로 flex이므로 절대
// 배치가 "되기는" 하지만, 되는 것을 열어 두면 canvas라는 이름이 아무것도 뜻하지 않게 되고
// 테마가 배치를 소유하는 레이아웃의 슬롯 안에 좌표를 박는 길이 열린다 (1층·2층 경계).
func (r *resolver) coord(attr soml.Attr, parent *soml.Element, conflict bool) (soml.Length, bool) {
	if conflict {
		return soml.Length{}, false
	}
	if !r.hasValue(attr) {
		return soml.Length{}, false
	}
	length, ok := r.lengthValue(attr.Name, attr.Value, attr.ValueSpan, lengthSyntax{signed: true})
	if !ok {
		return soml.Length{}, false
	}
	if !r.coordParentOK(parent) {
		r.errorf(attr.NameSpan, "%s is only for children of <%s> or direct children of a layout=\"canvas\" slide (%s) — wrap it in <%s> or give the slide layout=\"canvas\"",
			attr.Name, soml.StackElement, parentDesc(parent), soml.StackElement)
		return soml.Length{}, false
	}
	return length, true
}

// coordParentOK는 `x`/`y`가 유효한 부모에 있는지다 — <stack>의 자식이거나 canvas 슬라이드의
// 직속 자식(parent == nil이고 슬라이드 layout이 canvas)이다.
func (r *resolver) coordParentOK(parent *soml.Element) bool {
	if parent != nil {
		return parent.Name == soml.StackElement
	}
	return r.layout == soml.LayoutCanvas
}

// anchorTypo는 흔한 오타·짧은 이름·축이 뒤바뀐 이름을 올바른 9지점으로 잇는다.
//
// 짧은 이름(`top`)은 축을 하나만 적은 것이므로 나머지 축을 center로 채운 자리를 권한다.
// 축이 뒤바뀐 이름(`right-top`)은 어휘가 세로-가로 순서(`top-right`)라는 규칙으로 되짚어
// 준다. `center-center`는 어휘에 없는 자리라 `center` 한 단어를 권한다 (같은 자리를 두
// 이름으로 받지 않는다는 기준). 공백·붙여쓴 표기(`top left`·`topleft`)도 하이픈 표기로
// 이끈다.
func anchorTypo(raw string) (soml.StackAnchor, bool) {
	anchor, ok := map[string]soml.StackAnchor{
		"top":           soml.StackTopCenter,
		"bottom":        soml.StackBottomCenter,
		"left":          soml.StackCenterLeft,
		"right":         soml.StackCenterRight,
		"middle":        soml.StackCenter,
		"center-center": soml.StackCenter,
		"topleft":       soml.StackTopLeft,
		"topright":      soml.StackTopRight,
		"bottomleft":    soml.StackBottomLeft,
		"bottomright":   soml.StackBottomRight,
		"top left":      soml.StackTopLeft,
		"top center":    soml.StackTopCenter,
		"top right":     soml.StackTopRight,
		"bottom left":   soml.StackBottomLeft,
		"bottom center": soml.StackBottomCenter,
		"bottom right":  soml.StackBottomRight,
		"left-top":      soml.StackTopLeft,
		"right-top":     soml.StackTopRight,
		"left-bottom":   soml.StackBottomLeft,
		"right-bottom":  soml.StackBottomRight,
		"left-center":   soml.StackCenterLeft,
		"right-center":  soml.StackCenterRight,
	}[raw]
	return anchor, ok
}

// stackAnchorList는 진단에 실을 9지점 어휘 목록이다 — 어휘 정의(soml.StackAnchors)와 메시지가
// 갈라지지 않도록 손으로 쓰지 않는다 (boxAlignList와 같은 태도).
func stackAnchorList() string {
	names := make([]string, 0, len(soml.StackAnchors()))
	for _, anchor := range soml.StackAnchors() {
		names = append(names, string(anchor))
	}
	return strings.Join(names, ", ")
}

// valuePart는 값 안의 조각 하나다 — 조각의 원문과 값 안 오프셋을 함께 들고 다니는 이유는
// 진단이 조각을 가리켜야 하기 때문이다 (`pad="sm huge"`의 진단은 huge를 가리킨다).
type valuePart struct {
	text  string
	start int
}

// span은 이 조각의 소스 구간이다.
func (p valuePart) span(attr soml.Attr) soml.Span {
	return attr.ValueSpan.Slice(attr.Value, p.start, p.start+len(p.text))
}

// splitValues는 공백으로 구분된 조각을 오프셋과 함께 나눈다. strings.Fields를 쓰지 않는
// 이유는 오프셋이 필요하기 때문이다.
//
// **괄호 안의 공백은 조각 경계가 아니다.** 색 함수의 인자가 공백 구분이기 때문이다
// (`gradient(accent1 accent4 accent5)`). 괄호를 세지 않으면 그 표기가
// `border`에서 조각 수 에러로 끝나 색 함수 문법의 진단에 도달하지 못한다 (조각 수를 세는
// 진단은 저자를 엉뚱한 곳으로 보낸다).
// 열린 괄호가 닫히지 않으면 값 끝까지 한 조각으로 본다 — 그 조각을 받는 해석기가 함수
// 문법으로 진단한다.
func splitValues(raw string) []valuePart {
	var parts []valuePart
	start, depth := -1, 0
	for i := 0; i <= len(raw); i++ {
		if i < len(raw) {
			switch raw[i] {
			case '(':
				depth++
			case ')':
				if depth > 0 {
					depth--
				}
			}
		}
		if i < len(raw) && (depth > 0 || !isASCIISpace(raw[i])) {
			if start < 0 {
				start = i
			}
			continue
		}
		if start >= 0 {
			parts = append(parts, valuePart{text: raw[start:i], start: start})
			start = -1
		}
	}
	return parts
}

func isASCIISpace(c byte) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == '\r'
}

// startsNumeric은 값이 수치로 시작하는지다 — 토큰 이름과 수치가 같은 자리에서 갈라지는
// 경계다 (`gap="16"` vs `gap="md"`). 이 판정으로 진단이 정확해진다: `gap="huge"`는 "모르는
// 공간 토큰"이고 `gap="16px"`은 "단위를 떼라"다.
func startsNumeric(raw string) bool {
	if raw == "" {
		return false
	}
	c := raw[0]
	return c >= '0' && c <= '9' || c == '.' || c == '-' || c == '+'
}

// isDigits는 값이 부호·소수점 없는 숫자만으로 이루어졌는지다 (트랙 개수 판정).
func isDigits(raw string) bool {
	if raw == "" {
		return false
	}
	for i := 0; i < len(raw); i++ {
		if raw[i] < '0' || raw[i] > '9' {
			return false
		}
	}
	return true
}

// containsName은 이름 목록에 name이 있는지다. 목록이 14개 이하라 선형 탐색이 맵보다 싸고,
// 무엇보다 목록의 **순서**가 계약이므로(진단·문서 순서) 맵으로 바꿀 수 없다.
func containsName(names []string, name string) bool {
	for _, candidate := range names {
		if candidate == name {
			return true
		}
	}
	return false
}

// boxAlignList와 boxJustifyList는 진단에 실을 어휘 목록이다 — 어휘 정의(soml)와 메시지가
// 갈라지지 않도록 목록을 손으로 쓰지 않는다 (roleList·alignList와 같은 태도).
func boxAlignList() string {
	names := make([]string, 0, len(soml.BoxAligns()))
	for _, align := range soml.BoxAligns() {
		names = append(names, string(align))
	}
	return strings.Join(names, ", ")
}

func boxJustifyList() string {
	names := make([]string, 0, len(soml.BoxJustifies()))
	for _, justify := range soml.BoxJustifies() {
		names = append(names, string(justify))
	}
	return strings.Join(names, ", ")
}
