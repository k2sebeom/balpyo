package core

import (
	"fmt"
	"sort"
	"strings"

	"github.com/k2sebeom/balpyo/soml"
)

// 이 파일은 연결선(`<connector>`)의 값 층이다.
//
// 연결선은 슬라이드를 다 읽은 **뒤**에 푼다 — `from`·`to`가 뒤에서 선언한 상자를 가리킬 수 있어야
// 하고, 그 표(resolver.idElems)는 도형 · 그림을 해석하면서 채워진다. 외관은 도형과 같은 캐스케이드다:
// 인라인 > variant > 테마의 `connector.default`. 엔진에는 선의 기본값이 없다 — 테마가 정하지 않았고
// 저자도 쓰지 않았으면 에러다.

// connectorPatch는 연결선 외관의 부분 지정이다 (textPatch · shapePatch와 같은 규약 — nil은 "정하지 않음").
type connectorPatch struct {
	line       *soml.Border
	start, end *soml.ConnectorEnd
}

// mergeConnector는 patch가 정한 필드만 style에 얹는다 — 연결선 캐스케이드의 유일한 병합 지점이다.
func mergeConnector(style *soml.ConnectorStyle, patch connectorPatch) {
	if patch.line != nil {
		style.Line = *patch.line
	}
	if patch.start != nil {
		style.Start = *patch.start
	}
	if patch.end != nil {
		style.End = *patch.end
	}
}

// connectorVariant는 테마가 이름 붙인 연결선 번들이다.
type connectorVariant struct {
	name  string
	patch connectorPatch
}

// resolveConnectors는 이 슬라이드의 연결선을 문서 순서로 확정한다.
func (r *resolver) resolveConnectors() {
	for _, elem := range r.connectors {
		r.resolveConnector(elem)
	}
}

func (r *resolver) resolveConnector(elem *soml.Element) {
	style := soml.ConnectorStyle{Route: "straight", Start: soml.EndNone, End: soml.EndNone}
	mergeConnector(&style, r.comps.connectorDefault)
	if attr, ok := r.variantAttr(elem); ok {
		if patch, found := r.connectorVariantPatch(attr.Value); found {
			mergeConnector(&style, patch)
		} else {
			r.warnUnknownVariant(attr, r.connectorVariantNames())
		}
	}
	var inline connectorPatch
	var haveFrom, haveTo bool
	for _, attr := range elem.Attrs {
		switch attr.Name {
		case "from", "to":
			target, ok := r.connectorEnd(elem, attr)
			if !ok {
				continue
			}
			if attr.Name == "from" {
				style.From, haveFrom = target, true
			} else {
				style.To, haveTo = target, true
			}
		case "route":
			if v, ok := r.closedValue(attr, soml.ConnectorRoutes()); ok {
				style.Route = v
			}
		case "from-side", "to-side":
			if v, ok := r.closedValue(attr, soml.ConnectorSides()); ok {
				if v == "auto" {
					v = ""
				}
				if attr.Name == "from-side" {
					style.FromSide = v
				} else {
					style.ToSide = v
				}
			}
		case "line":
			if line, ok := r.connectorLine(attr); ok {
				inline.line = &line
			}
		case "start", "end":
			if v, ok := r.closedValue(attr, soml.ConnectorEnds()); ok {
				end := soml.ConnectorEnd(v)
				if attr.Name == "start" {
					inline.start = &end
				} else {
					inline.end = &end
				}
			}
		case "variant":
			// 위에서 해석했다 — 여기서 또 보면 모르는 variant 경고가 두 번 난다.
		case "flow":
			switch {
			case !attr.HasValue, attr.Value == "true":
				style.Flow = true
			case attr.Value == "false":
			default:
				r.errorf(attr.ValueSpan, "flow must be true or false (%q)", attr.Value)
			}
		case "label":
			if r.hasValue(attr) {
				style.Label = attr.Value
				// 라벨은 caption 역할 위에 테마의 `connector` 태그 기본값이 얹힌다 — 선 옆 글자의
				// 크기 · 색은 테마의 것이다 (저자 속성이 없다).
				label := r.textRoleDefaults(soml.RoleCaption)
				mergeText(&label, r.textComponentDefault(soml.ConnectorElement))
				style.LabelStyle = &label
			}
		}
	}
	mergeConnector(&style, inline)
	if style.Flow && style.Line.Style == soml.BorderSolid {
		// 흐르는 선은 파선이다 — 대시가 있어야 흐름이 보이고, PPTX의 멈춘 모습도 같은 파선이다.
		style.Line.Style = soml.BorderDashed
	}
	if _, ok := elem.Attr("from"); !ok {
		r.errorf(elem.NameSpan, "<connector> needs from — the id of the shape or image the line starts at")
	}
	if _, ok := elem.Attr("to"); !ok {
		r.errorf(elem.NameSpan, "<connector> needs to — the id of the shape or image the line ends at")
	}
	if haveFrom && haveTo && style.From == style.To {
		attr, _ := elem.Attr("to")
		r.errorf(attr.ValueSpan, "<connector> from and to are the same element %q — a connector joins two boxes", attr.Value)
		haveTo = false
	}
	if style.Line.Width <= 0 && !style.Line.None {
		r.errorf(elem.NameSpan, "<connector> has no line — theme %q defines no connector.default line, so write line=%q", r.theme.Name, "2 solid muted")
		return
	}
	if !haveFrom || !haveTo {
		// 끝 하나라도 풀리지 않은 연결선은 확정하지 않는다 — 렌더러가 반쪽 선을 그리지 않게.
		return
	}
	elem.Appearance = &soml.Appearance{Connector: &style}
}

// connectorEnd는 `from`·`to`의 이름을 같은 슬라이드의 상자로 푼다.
func (r *resolver) connectorEnd(elem *soml.Element, attr soml.Attr) (*soml.Element, bool) {
	if !r.hasValue(attr) {
		return nil, false
	}
	target, ok := r.idElems[attr.Value]
	if !ok {
		names := make([]string, 0, len(r.idElems))
		for name := range r.idElems {
			names = append(names, name)
		}
		sort.Strings(names)
		list := "this slide has no ids"
		if len(names) > 0 {
			list = "ids on this slide: " + strings.Join(names, ", ")
		}
		r.errorf(attr.ValueSpan, "<connector> %s=%q is not the id of a shape or image on this slide%s (%s)",
			attr.Name, attr.Value, didYouMean(attr.Value, strings.Join(names, ", ")), list)
		return nil, false
	}
	return target, true
}

// connectorLine은 `line`을 해석한다 — 도형 `border`와 같은 문법이지만 그라디언트와 `none`은 받지 않는다:
// 선 하나에 그라디언트를 걸면 웹의 객체 경계 상자가 가로선에서 높이 0이 되어 그려지지 않고, `none`인
// 연결선은 그릴 것이 없다.
func (r *resolver) connectorLine(attr soml.Attr) (soml.Border, bool) {
	if attr.HasValue && attr.Value == noneValue {
		r.errorf(attr.ValueSpan, "a connector line cannot be none — remove the connector instead")
		return soml.Border{}, false
	}
	line, ok := r.border(attr)
	if !ok {
		return soml.Border{}, false
	}
	if line.Color.IsGradient() {
		r.errorf(attr.ValueSpan, "a connector line is one color — gradients are not supported on connectors")
		return soml.Border{}, false
	}
	return line, true
}

// closedValue는 닫힌 어휘 하나를 본다 — 값 없음 · 빈 값 · 어휘 밖은 진단이다.
func (r *resolver) closedValue(attr soml.Attr, vocab []string) (string, bool) {
	if !r.hasValue(attr) {
		return "", false
	}
	for _, v := range vocab {
		if attr.Value == v {
			return v, true
		}
	}
	list := strings.Join(vocab, ", ")
	r.errorf(attr.ValueSpan, "unknown %s %q%s (available values: %s)", attr.Name, attr.Value, didYouMean(attr.Value, list), list)
	return "", false
}

func (r *resolver) connectorVariantPatch(name string) (connectorPatch, bool) {
	for _, v := range r.comps.connectorVariants {
		if v.name == name {
			return v.patch, true
		}
	}
	return connectorPatch{}, false
}

func (r *resolver) connectorVariantNames() []string {
	names := make([]string, len(r.comps.connectorVariants))
	for i, v := range r.comps.connectorVariants {
		names[i] = v.name
	}
	return names
}

// connectorComponentsFile은 components.json의 `connector` 절이다 — 선 하나와 양 끝 표식의 기본값과
// variant다. 절 전체가 선택이다 (정하지 않은 테마의 연결선은 `line`을 직접 써야 한다).
type connectorComponentsFile struct {
	Comment  string                 `json:"$comment"`
	Default  connectorStyleFile     `json:"default"`
	Variants []connectorVariantFile `json:"variants"`
}

type connectorStyleFile struct {
	Line  string `json:"line"`
	Start string `json:"start"`
	End   string `json:"end"`
}

type connectorVariantFile struct {
	Name  string `json:"name"`
	Line  string `json:"line"`
	Start string `json:"start"`
	End   string `json:"end"`
}

// connectorPatchFromFile은 테마 파일의 선 · 표식을 패치로 눕힌다 (빈 값은 "정하지 않음").
func connectorPatchFromFile(theme soml.Theme, where, line, start, end string) (connectorPatch, error) {
	var patch connectorPatch
	if line != "" {
		r := &resolver{theme: theme, tokensOnly: true}
		b, ok := r.connectorLine(soml.Attr{Name: "line", Value: line, HasValue: true})
		if err := themeValueErr(r, ok); err != nil {
			return patch, declAt(fmt.Errorf("%s line: %w", where, err), atKey("line"))
		}
		patch.line = &b
	}
	for _, e := range []struct {
		key, raw string
		dst      **soml.ConnectorEnd
	}{{"start", start, &patch.start}, {"end", end, &patch.end}} {
		if e.raw == "" {
			continue
		}
		known := false
		for _, v := range soml.ConnectorEnds() {
			known = known || v == e.raw
		}
		if !known {
			return patch, declAt(fmt.Errorf("%s: unknown connector end %q (available values: %s)", where, e.raw, strings.Join(soml.ConnectorEnds(), ", ")), atKey(e.key))
		}
		end := soml.ConnectorEnd(e.raw)
		*e.dst = &end
	}
	return patch, nil
}
