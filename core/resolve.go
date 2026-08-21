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
// 역할 기본값이 완전한 외관을 깔고 위 세 단이 부분 패치로 얹힌다 — 병합은 mergeText
// (cascade.go) 한 곳에서만 일어난다. 도형 외관도 **같은 기계의 복제**를 쓴다 (mergeShape) —
// 도형에는 역할이 없어 컴포넌트 기본값이 가장 아래 단이라는 점만 갈라진다 (resolveShape).
// variant 표는 아직 하드코딩 테마 안이다 (theme.go — 테마 패키지 포맷은 Iteration 10).
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
			// 슬라이드 직속 노드의 부모는 없다 — 배치 해석이 부모를 보는 이유는
			// resolveLayout의 주석에 있다 (span은 <grid>의 자식에서만 뜻이 있다).
			r.node(node, nil)
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
	// ids는 이 슬라이드에서 이미 쓴 도형 `id`와 그 자리다 (중복 진단이 처음 자리를 가리켜야
	// 한다). 상태가 resolver에 있는 것이 곧 검사 범위가 슬라이드라는 뜻이다 — 근거는
	// resolver.shapeID의 주석에 있다.
	ids map[string]soml.Span
}

func (r *resolver) errorf(span soml.Span, format string, args ...any) {
	r.diags = append(r.diags, soml.Errorf(r.file, span, format, args...))
}

func (r *resolver) warnf(span soml.Span, format string, args ...any) {
	r.diags = append(r.diags, soml.Warningf(r.file, span, format, args...))
}

// node는 콘텐츠 트리를 훑는다. 외관을 갖는 요소만 외관을 해석하고, **배치는 요소 종류와
// 무관하게 전부 지난다** (resolveLayout — 자식 배치 속성은 한 벌이므로 종류마다 표를
// 복제하지 않는다). 한 요소가 두 채널을 함께 쓰는 것은 정상이다 (`<p grow="1" size="30">`).
//
// 리스트 컨테이너(<ul>/<ol>)의 Appearance는 **텍스트 외관**을 채우지 않는다: 컨테이너는
// 항목을 담을 뿐이고 타이포그래피는 <li>(body 역할)에 붙는다. 마커·들여쓰기 같은 리스트
// 표현을 렌더러가 컨테이너에서 무엇으로 읽을지는 렌더 매핑(Iteration 2 항목 05·07)이
// 정한다 — 여기서 미리 정하면 그 결정 전에 두 렌더러가 각자 추측하게 된다. 반면 배치는
// 컨테이너에도 붙는다 (`<row><ul basis="360">`은 두 열에 리스트를 놓는 흔한 배치다).
func (r *resolver) node(node soml.Node, parent *soml.Element) {
	elem, ok := node.(*soml.Element)
	if !ok {
		return
	}
	r.resolveLayout(elem, parent)
	if elem.Name == soml.ShapeElement {
		// 도형은 세 번째 갈래다 (외관·배치에 이어) — 자식은 텍스트 요소이므로 아래 순회가
		// 기존 텍스트 경로로 데려간다. 부모가 도형이라는 사실이 텍스트 해석을 바꾸지 않는다:
		// 바꾸는 것은 상속이고, 상속은 지금 구조에 없다.
		r.resolveShape(elem)
	}
	if role, ok := r.textRole(elem); ok {
		r.resolveText(elem, role)
		// 텍스트 역할 요소의 자식은 문자 데이터와 인라인 태그뿐이다 (파서의 modelInline) —
		// 인라인 런 스타일은 블록의 확정 스타일에서 상속하므로 여기서 트리 순회가 끝난다.
		r.resolveInlines(elem, elem.Appearance.Text)
		return
	}
	for _, child := range elem.Children {
		r.node(child, elem)
	}
}

// textRole은 요소의 텍스트 역할이다. `<text>`는 role 속성(생략하면 body)이, 블록·<li>는
// 언어의 등가 표(soml.TagTextRole — `<p>` ≡ `<text role="body">`)가 역할을 정한다.
// 블록은 role 속성을 받지 않으므로 (파서 allowlist — `<p role="title">`은 두 표기의
// 등가를 깬다) 두 출처가 겹치는 요소는 없다.
func (r *resolver) textRole(elem *soml.Element) (soml.TextRole, bool) {
	if elem.Name == soml.TextElement {
		return r.role(elem), true
	}
	return soml.TagTextRole(elem.Name)
}

// resolveText는 텍스트 요소(`<text>`·블록·<li>)의 외관을 4단 캐스케이드로 확정한다
// (PRD §8.3):
//
//	인라인 속성 > variant > 컴포넌트 기본값 > 역할/토큰
//
// 역할 기본값(완전한 외관)을 바닥에 깔고, 위 세 단을 낮은 단부터 부분 패치로 얹는다.
// 컴포넌트 기본값 단은 태그별이다 (textComponentDefault — h3의 크기, blockquote의 색).
// 병합은 mergeText(cascade.go) 한 곳에서만 일어난다 — 우선순위는 아래 세 줄의 순서가
// 전부이고, 다른 어디에도 없다.
func (r *resolver) resolveText(elem *soml.Element, role soml.TextRole) {
	style := textRoleDefaults(role)
	mergeText(&style, textComponentDefault(elem.Name))
	mergeText(&style, r.variantPatch(elem))
	mergeText(&style, r.inlinePatch(elem))
	elemAppearance(elem).Text = &style
}

// elemAppearance는 요소의 확정 외관·배치 자리다 (없으면 만든다).
//
// 구조체를 통째로 대입하지 않는 이유: 한 요소가 외관과 배치를 함께 갖는다
// (`<p grow="1" size="30">`). 대입하는 코드가 둘이면 나중에 해석되는 쪽이 앞의 결과를
// 조용히 지운다 — 그 실패는 산출물에서 배치나 외관 하나가 사라지는 것으로만 드러난다.
func elemAppearance(elem *soml.Element) *soml.Appearance {
	if elem.Appearance == nil {
		elem.Appearance = &soml.Appearance{}
	}
	return elem.Appearance
}

// variantPatch는 `variant` 속성을 테마의 **텍스트** variant 표에서 찾는다.
//
// 모르는 variant는 에러가 아니라 **경고 + 빈 패치(컴포넌트 기본 외관 폴백)**다 (PRD §13-6):
// variant 어휘는 테마의 사유물이라, 다른 테마로 만든 콘텐츠가 이 테마에서 빌드되지 않으면
// 콘텐츠의 테마 간 이식성이 깨진다. 엄격함이 필요한 곳은 --strict가 에러로 승격한다.
func (r *resolver) variantPatch(elem *soml.Element) textPatch {
	attr, ok := r.variantAttr(elem)
	if !ok {
		return textPatch{}
	}
	patch, known := textVariantPatch(attr.Value)
	if !known {
		r.warnUnknownVariant(attr, textVariantNames())
		return textPatch{}
	}
	return patch
}

// variantAttr는 쓸 값이 있는 `variant` 속성이다 — 텍스트와 도형이 공유한다. 값 없음·빈 값의
// 진단은 hasValue가 소유한다 (모르는 variant는 경고지만 값 없는 variant는 에러다: 뜻을 정할
// 수 없는 형태이기 때문이다).
func (r *resolver) variantAttr(elem *soml.Element) (soml.Attr, bool) {
	attr, ok := elem.Attr("variant")
	if !ok || !r.hasValue(attr) {
		return soml.Attr{}, false
	}
	return attr, true
}

// warnUnknownVariant는 모르는 variant 경고다 — **요소 종류에 맞는 어휘를 나열한다.**
//
// variant 어휘가 텍스트용·도형용 두 벌이 된 순간(Iteration 4) 생긴 요구다:
// `<shape variant="kicker">`에 텍스트 목록을 나열하면 저자를 그 테마에 없는 방향으로 보낸다.
// 두 표가 같은 이름을 가져도 무관하고(요소 종류가 어느 표를 볼지 정한다), 그래서 목록은
// 호출자가 준다.
//
// 문구는 여기 한 곳에만 있다: 폴백 규칙이 두 채널에서 같으므로 문구가 갈라질 이유가 없고,
// 갈라지면 저자가 두 경고를 다른 규칙으로 읽는다. 어휘 종류를 문구에 적지 않은 이유는
// 나열된 목록이 이미 그것을 말하기 때문이다 (`variant: kicker, lead`).
func (r *resolver) warnUnknownVariant(attr soml.Attr, names []string) {
	r.warnf(attr.ValueSpan, "variant %q가 테마에 없다 — 기본 외관으로 계속한다 (이 테마의 variant: %s)",
		attr.Value, strings.Join(names, ", "))
}

// inlinePatch는 인라인 속성을 캐스케이드의 마지막 단으로 만든다. 값 어휘 검사와 진단은
// 속성별 해석 함수(size/color/align)가 그대로 소유한다 — 틀린 값은 패치에 실리지 않으므로
// 아래 단의 값이 남는다 (틀린 role이 기본 역할로 계속하는 것과 같은 정책).
func (r *resolver) inlinePatch(elem *soml.Element) textPatch {
	var patch textPatch
	for _, attr := range elem.Attrs {
		switch attr.Name {
		case "size":
			if size, ok := r.size(attr); ok {
				patch.size = &size
			}
		case "color":
			if color, ok := r.color(attr); ok {
				patch.color = &color
			}
		case "align":
			if align, ok := r.align(attr); ok {
				patch.align = &align
			}
		case soml.AnchorAttr:
			if anchor, ok := r.anchor(attr); ok {
				patch.anchor = &anchor
			}
		}
	}
	return patch
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
	if unit := numberSuffix(raw); unit != "" {
		// 수치 뒤에 붙은 것만 "단위"라고 부른다. 수치가 아예 없으면 아래의 수치 진단으로
		// 내려보낸다 — `size="큼"`에 단위를 떼라고 하면 저자가 무엇을 고칠지 알 수 없다
		// (numberSuffix가 그 판정을 소유한다).
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
	return r.colorValue(attr.Name, attr.Value, attr.ValueSpan)
}

// colorValue는 색 값 하나를 해석한다 — 속성이 아니라 (이름, 값, 구간)을 받는 이유는
// lengthValue와 같다: 값 하나가 색을 조각으로 담는다 (`border="1 solid muted"`). 진단은
// 문제가 된 **조각**을 가리킨다.
//
// 도형의 `fill`·`border` 색이 텍스트 `color`와 이 함수를 공유하는 것이 계약이다 — 색 문법이
// 속성마다 따로 있으면 같은 값이 자리에 따라 다르게 읽히고, 그 차이는 두 산출물을 나란히
// 볼 때까지 보이지 않는다.
func (r *resolver) colorValue(name, raw string, span soml.Span) (soml.Color, bool) {
	if i := strings.IndexByte(raw, '('); i >= 0 {
		// 색 함수는 외관 값 문법에 있지만(PRD §7.6) Iteration 8에서 열린다. "알 수 없는 토큰"
		// 으로 뭉개면 저자가 오타를 찾다가 시간을 쓴다 — 문법은 맞고 아직 없다고 말한다.
		r.errorf(span, "색 함수 %s(…)는 아직 쓸 수 없다 (gradient·shade·tint·alpha는 Iteration 8) — 토큰 이름이나 #RRGGBB 리터럴을 쓴다", raw[:i])
		return soml.Color{}, false
	}
	if strings.HasPrefix(raw, "#") {
		rgb, ok := hexRGB(raw[1:])
		if !ok {
			r.errorf(span, "색 리터럴은 #RRGGBB 또는 #RGB 형태다 (%q)", raw)
			return soml.Color{}, false
		}
		return soml.LiteralColor(rgb), true
	}
	if _, known := r.theme.Color(raw); !known {
		r.errorf(span, "%s의 색 토큰 %q가 테마에 없다 (쓸 수 있는 토큰: %s, 또는 #RRGGBB 리터럴)",
			name, raw, strings.Join(r.theme.ColorNames(), ", "))
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
	case "variant":
		return "kicker"
	case "size":
		return "24"
	case "color":
		return "accent1"
	case "align":
		return "center"
	case soml.AnchorAttr:
		return "center"
	// 도형 외관 속성 (PRD §7.6 표의 shape 행 순서).
	case "type":
		return "rect"
	case "id":
		return "db"
	case "fill":
		return "surface"
	case "border":
		return "1 solid muted"
	case "radius":
		return "md"
	case "shadow":
		return "soft"
	case "opacity":
		return "0.8"
	case "rotate":
		return "45"
	case "flip":
		return "h"
	case "weight":
		return "bold"
	case "href":
		return "https://example.com"
	// 배치 속성 (PRD §7.4 표 순서). `wrap`은 값 없이도 뜻이 되므로 여기 없다 —
	// 그 판정은 resolver.wrap이 소유한다.
	case "gap":
		return "md"
	case "justify":
		return "between"
	case "pad":
		return "lg"
	case "cols", "rows":
		return "4"
	case "w", "h":
		return "fill"
	case "min-w", "min-h", "max-w", "max-h":
		return "240"
	case "ratio":
		return "16:9"
	case "grow":
		return "1"
	case "shrink":
		return "0"
	case "basis":
		return "240"
	case "span":
		return "2"
	case "self":
		return "center"
	case "order":
		return "-1"
	case soml.StyleAttr:
		return "align-content: space-evenly"
	}
	return ""
}

// numberSuffix는 수치 뒤에 붙은 꼬리(단위 접미사)다. 수치가 아예 없으면 빈 문자열이다 —
// `size="큼"`·`w="어디"`는 "단위를 떼라"가 아니라 "수치가 아니다"로 진단해야 저자가 무엇을
// 고칠지 안다. 외관(size)과 배치(길이)가 같은 판정을 공유하는 이유는 단위 계약이 하나이기
// 때문이다 (PRD §13-5) — 두 벌이 되면 한쪽만 새 단위를 막게 된다.
func numberSuffix(raw string) string {
	end := 0
	for end < len(raw) {
		c := raw[end]
		if c >= '0' && c <= '9' || c == '.' || (end == 0 && (c == '-' || c == '+')) {
			end++
			continue
		}
		break
	}
	if end == 0 {
		return ""
	}
	return raw[end:]
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
