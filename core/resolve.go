package core

import (
	"sort"
	"strconv"
	"strings"

	"github.com/k2sebeom/balpyo/soml"
)

// Resolve는 덱 모델의 외관을 해석한다 — 렌더러가 보기 전에 모든 외관 값이 확정되어야
// 한다는 계약을 이 함수가 진다. 결과는 요소의 Appearance와 덱의 Theme(토큰 표)에 담긴다.
//
// 해석 순서는 4단 캐스케이드다 (PRD §8.3):
//
//	인라인 속성 > variant > 컴포넌트 기본값 > 역할/토큰
//
// 이번 iteration에 구현된 것은 **인라인 > 역할 기본값 > 테마 토큰**이다. 가운데 두 단
// (variant, 컴포넌트 기본값)은 Iteration 2에서 resolveText의 표시된 자리에 들어온다.
//
// 이 순서를 렌더러가 아니라 여기서 한 번만 적용하기 때문에 웹과 PPTX가 같은 외관을 본다.
// computed-style 역추론은 없다 — 외관은 스키마에서 양 타깃으로 순방향 컴파일된다.
//
// 값 어휘도 이 층이 소유한다. 파서는 태그·속성 **이름**만 보고 값은 원문으로 남기므로
// (soml/node.go), 역할 5종·정렬 4종·색 토큰 같은 **값 층의 닫힌 어휘**를 여기서 검사한다.
// 진단은 Attr.ValueSpan을 근거로 파서와 같은 정밀도의 line:col을 갖는다.
func Resolve(deck *soml.Deck) soml.Diagnostics {
	if deck == nil {
		return nil
	}
	// 테마는 지금 하드코딩이다 (theme.go). Iteration 10에서 프로젝트 설정의 theme 필드를
	// 읽어 온 테마 패키지가 이 대입을 대체한다.
	deck.Theme = minimalTheme()

	var diags soml.Diagnostics
	for _, slide := range deck.Slides {
		r := resolver{theme: deck.Theme, file: slide.Source}
		for _, node := range slide.Content {
			r.node(node)
		}
		// 진단은 파일 안에서 소스 순서로 낸다 — 저자가 위에서 아래로 고칠 수 있게. 역할을
		// 먼저 해석하고 나머지 속성을 훑기 때문에 발견 순서가 소스 순서와 어긋난다.
		sort.SliceStable(r.diags, func(i, j int) bool {
			return r.diags[i].Span.Start.Offset < r.diags[j].Span.Start.Offset
		})
		diags = append(diags, r.diags...)
	}
	return diags
}

// resolver는 슬라이드 하나의 해석 상태다.
type resolver struct {
	theme soml.Theme
	file  string
	diags soml.Diagnostics
}

func (r *resolver) errorf(span soml.Span, format string, args ...any) {
	r.diags = append(r.diags, soml.Errorf(r.file, span, format, args...))
}

// node는 콘텐츠 트리를 훑는다. 외관을 갖는 요소만 해석하고 나머지는 지나간다.
func (r *resolver) node(node soml.Node) {
	elem, ok := node.(*soml.Element)
	if !ok {
		return
	}
	if elem.Name == soml.TextElement {
		r.resolveText(elem)
	}
	for _, child := range elem.Children {
		r.node(child)
	}
}

// resolveText는 `<text>`의 외관을 확정한다.
func (r *resolver) resolveText(elem *soml.Element) {
	style := textRoleDefaults(r.role(elem))

	// Iteration 2: 여기서 variant(테마가 정의한 외관 번들)와 컴포넌트 기본값이 역할 기본값
	// 위에 얹힌다. 인라인 속성은 그 뒤에 적용되어 항상 마지막에 이긴다.

	for _, attr := range elem.Attrs {
		switch attr.Name {
		case "size":
			if size, ok := r.size(attr); ok {
				style.Size = size
			}
		case "color":
			if color, ok := r.color(attr); ok {
				style.Color = color
			}
		case "align":
			if align, ok := r.align(attr); ok {
				style.Align = align
			}
		}
	}
	elem.Appearance = &soml.Appearance{Text: &style}
}

// role은 텍스트 역할을 해석한다. 값이 없거나 어휘 밖이면 진단하고 기본 역할로 계속한다 —
// 역할 하나가 틀렸다고 그 요소의 다른 속성 진단을 삼키면 저자가 빌드를 여러 번 돌게 된다.
func (r *resolver) role(elem *soml.Element) soml.TextRole {
	attr, ok := elem.Attr("role")
	if !ok {
		return soml.DefaultTextRole
	}
	if !r.hasValue(attr) {
		return soml.DefaultTextRole
	}
	for _, role := range soml.TextRoles() {
		if string(role) == attr.Value {
			return role
		}
	}
	r.errorf(attr.ValueSpan, "알 수 없는 role %q (쓸 수 있는 역할: %s)", attr.Value, roleList())
	return soml.DefaultTextRole
}

// size는 폰트 크기를 해석한다. 값은 **무단위 수치 = 논리 스테이지 px**다 (PRD §13-5).
func (r *resolver) size(attr soml.Attr) (float64, bool) {
	if !r.hasValue(attr) {
		return 0, false
	}
	raw := attr.Value
	if strings.HasSuffix(raw, "%") {
		// %를 이번 iteration에서 막는 이유: 상대 폰트 크기를 PPTX로 번역할 곳이 없다. OOXML
		// `rPr@sz`는 절대값(1/100 pt)이고 "부모 텍스트 크기"라는 상속 기준도 없어서, 웹에서만
		// 동작하고 PPTX에서는 조용히 다른 크기가 되는 속성이 된다. 상대적 위계는 역할과
		// variant(Iteration 2)로 표현한다 — 그 둘은 양 타깃에 모두 대응이 있다.
		r.errorf(attr.ValueSpan, "size에 %%는 쓸 수 없다 (%q) — PPTX 폰트 크기는 절대값이라 상대 크기를 번역할 곳이 없다. 논리 px 수치나 role을 쓴다", raw)
		return 0, false
	}
	end := 0
	for end < len(raw) {
		c := raw[end]
		if c >= '0' && c <= '9' || c == '.' || (end == 0 && (c == '-' || c == '+')) {
			end++
			continue
		}
		break
	}
	if unit := raw[end:]; unit != "" && end > 0 {
		// 수치 뒤에 붙은 것만 "단위"라고 부른다. 수치가 아예 없으면 아래의 수치 진단으로
		// 내려보낸다 — `size="큼"`에 단위를 떼라고 하면 저자가 무엇을 고칠지 알 수 없다.
		r.errorf(attr.ValueSpan, "size는 단위 없는 논리 px 수치다 (%q의 %q를 뗀다) — px·pt·em 접미사는 쓸 수 없다 (PRD §13-5)", raw, unit)
		return 0, false
	}
	px, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		r.errorf(attr.ValueSpan, "size 값이 수치가 아니다 (%q)", raw)
		return 0, false
	}
	if px <= 0 {
		r.errorf(attr.ValueSpan, "size는 0보다 커야 한다 (%q)", raw)
		return 0, false
	}
	return px, true
}

// color는 색을 해석한다. 토큰 이름은 **이름으로 남는다** — RGB로 눕히면 PPTX의 schemeClr
// 참조를 만들 수 없다 (soml.Color의 주석).
func (r *resolver) color(attr soml.Attr) (soml.Color, bool) {
	if !r.hasValue(attr) {
		return soml.Color{}, false
	}
	raw := attr.Value
	if i := strings.IndexByte(raw, '('); i >= 0 {
		// 색 함수는 외관 값 문법에 있지만(PRD §7.6) Iteration 8에서 열린다. "알 수 없는 토큰"
		// 으로 뭉개면 저자가 오타를 찾다가 시간을 쓴다 — 문법은 맞고 아직 없다고 말한다.
		r.errorf(attr.ValueSpan, "색 함수 %s(…)는 아직 쓸 수 없다 (gradient·shade·tint·alpha는 Iteration 8) — 토큰 이름이나 #RRGGBB 리터럴을 쓴다", raw[:i])
		return soml.Color{}, false
	}
	if strings.HasPrefix(raw, "#") {
		rgb, ok := hexRGB(raw[1:])
		if !ok {
			r.errorf(attr.ValueSpan, "색 리터럴은 #RRGGBB 또는 #RGB 형태다 (%q)", raw)
			return soml.Color{}, false
		}
		return soml.LiteralColor(rgb), true
	}
	if _, known := r.theme.Color(raw); !known {
		r.errorf(attr.ValueSpan, "색 토큰 %q가 테마에 없다 (쓸 수 있는 토큰: %s, 또는 #RRGGBB 리터럴)",
			raw, strings.Join(r.theme.ColorNames(), ", "))
		return soml.Color{}, false
	}
	return soml.TokenColor(raw), true
}

// align은 문단 정렬을 해석한다. 어휘는 논리 이름 4종이다 (soml.TextAlign의 주석).
func (r *resolver) align(attr soml.Attr) (soml.TextAlign, bool) {
	if !r.hasValue(attr) {
		return "", false
	}
	raw := attr.Value
	for _, align := range soml.TextAligns() {
		if string(align) == raw {
			return align, true
		}
	}
	// 저자는 CSS 습관대로 left/right를 쓴다. 어휘 목록만 던지는 대신 대응하는 논리 이름을
	// 짚어 준다 — 고치는 방법이 진단 안에 있어야 한 번에 끝난다.
	if physical, ok := map[string]soml.TextAlign{"left": soml.AlignStart, "right": soml.AlignEnd}[raw]; ok {
		r.errorf(attr.ValueSpan, "align은 논리 이름을 쓴다 — %q 대신 %q (RTL에서도 뜻이 유지된다)", raw, string(physical))
		return "", false
	}
	r.errorf(attr.ValueSpan, "알 수 없는 align %q (쓸 수 있는 값: %s)", raw, alignList())
	return "", false
}

// hasValue는 속성에 쓸 값이 있는지 본다. 값 없는 외관 속성은 에러다 — `<slide skip>`처럼
// 있음만으로 뜻이 되는 불리언 속성과 달리, 외관 속성의 "값 없음"은 뜻을 정할 수 없다.
func (r *resolver) hasValue(attr soml.Attr) bool {
	switch {
	case !attr.HasValue:
		r.errorf(attr.NameSpan, "%s 속성에는 값이 필요하다 (%s=%q)", attr.Name, attr.Name, attrExample(attr.Name))
		return false
	case attr.Value == "":
		r.errorf(attr.ValueSpan, "%s 속성이 비어 있다 (지우면 기본값을 쓴다)", attr.Name)
		return false
	}
	return true
}

// attrExample은 진단에 실을 값 예시다.
func attrExample(name string) string {
	switch name {
	case "role":
		return "body"
	case "size":
		return "24"
	case "color":
		return "accent1"
	case "align":
		return "center"
	}
	return ""
}

// hexRGB는 `#` 뒤의 16진수를 대문자 6자리로 정규화한다. #RGB는 각 자리를 두 배로 늘린다
// (CSS와 같은 규칙 — 저자가 두 표기에서 다른 색을 보면 안 된다).
func hexRGB(digits string) (string, bool) {
	if len(digits) != 3 && len(digits) != 6 {
		return "", false
	}
	var b strings.Builder
	b.Grow(6)
	for i := 0; i < len(digits); i++ {
		c := digits[i]
		switch {
		case c >= '0' && c <= '9', c >= 'A' && c <= 'F':
		case c >= 'a' && c <= 'f':
			c -= 'a' - 'A'
		default:
			return "", false
		}
		b.WriteByte(c)
		if len(digits) == 3 {
			b.WriteByte(c)
		}
	}
	return b.String(), true
}

// roleList와 alignList는 진단에 실을 어휘 목록이다 — 어휘 정의(soml)와 메시지가 갈라지지
// 않도록 목록을 손으로 쓰지 않는다.
func roleList() string {
	names := make([]string, 0, len(soml.TextRoles()))
	for _, role := range soml.TextRoles() {
		names = append(names, string(role))
	}
	return strings.Join(names, ", ")
}

func alignList() string {
	names := make([]string, 0, len(soml.TextAligns()))
	for _, align := range soml.TextAligns() {
		names = append(names, string(align))
	}
	return strings.Join(names, ", ")
}
