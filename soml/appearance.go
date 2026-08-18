package soml

// TextElement는 텍스트 요소 이름이다. 파서가 어휘로 쓰고, core가 외관을 해석할 대상을 이
// 이름으로 찾고, 렌더러가 같은 이름으로 그린다 — 이름 문자열이 세 층에 흩어져 있으면 오타
// 하나가 "파싱은 되는데 그려지지 않는" 조용한 실패가 된다.
const TextElement = "text"

// Appearance는 core가 **해석을 끝낸** 외관이다.
//
// 렌더러는 이 구조체만 읽는다 — **Attrs를 다시 해석하지 않는다.** 값 어휘, 캐스케이드
// (인라인 > variant > 컴포넌트 기본값 > 역할·토큰), 테마 기본값은 core.Resolve가 한 번만
// 적용한다. 이 계약이 깨지면 웹과 PPTX의 외관이 조용히 갈라지고, 그 종류의 버그는 두 타깃을
// 나란히 렌더해 보기 전까지 드러나지 않는다.
type Appearance struct {
	// Text는 텍스트 외관이다. <text>와 (Iteration 2의) 텍스트 서브셋 태그에만 채워진다.
	// 도형 외관(fill/border/radius/shadow)은 Iteration 4에서 여기 형제로 붙는다.
	Text *TextStyle
}

// TextStyle은 해석된 텍스트 외관이다. 해석 전(파싱 직후)이거나 텍스트 요소가 아니면 nil이다 —
// 렌더러가 매번 두 단계 nil 검사를 쓰지 않도록 요소에 직접 둔다.
func (e *Element) TextStyle() *TextStyle {
	if e.Appearance == nil {
		return nil
	}
	return e.Appearance.Text
}

// TextStyle은 확정된 텍스트 외관이다.
type TextStyle struct {
	Role TextRole
	// Font는 폰트 토큰 이름이다 (Theme.Font로 패밀리를 찾는다). 색과 같은 이유로 패밀리
	// 문자열이 아니라 토큰 참조로 남는다 — PPTX fontScheme의 major/minor 참조가 되어야
	// 내보낸 파일이 PowerPoint의 테마 폰트 변경을 따라간다.
	Font FontRole
	// Size는 논리 스테이지 px다 (PRD §13-5). pt·EMU 환산은 타깃 경계에서만 일어난다.
	Size float64
	// Weight는 CSS font-weight 수치다 (400·700). OOXML은 굵기 단계가 없어 렌더러가
	// 임계값으로 rPr@b를 판단한다 — 그 임계값은 렌더러의 몫이고 여기서는 수치를 보존한다.
	Weight int
	// LineHeight는 폰트 크기 배수다 (무단위). CSS line-height와 OOXML lnSpc@spcPct가
	// 둘 다 배수를 받으므로 이 표현이 양 타깃에 그대로 간다.
	LineHeight float64
	Color      Color
	Align      TextAlign
}

// TextRole은 텍스트 역할이다 — 1층의 HTML 서브셋(`<p>`, `<h2>` …)과 2층의 `<text role>`이
// 공유하는 타이포그래피 계약이다 (PRD §8.3). 어휘는 5종으로 닫혀 있다.
type TextRole string

const (
	RoleTitle   TextRole = "title"
	RoleHeading TextRole = "heading"
	RoleBody    TextRole = "body"
	RoleCaption TextRole = "caption"
	RoleCode    TextRole = "code"
)

// DefaultTextRole은 `role`을 생략한 `<text>`의 역할이다. `<p>` ≡ `<text role="body">`라는
// 등가(PRD §7.7)의 다른 쪽 끝이다.
const DefaultTextRole = RoleBody

// TextRoles는 역할 어휘 전체다. 순서가 진단 메시지에 실리는 순서이자 문서 순서다 —
// 크기 큰 것부터 작은 것으로, 마지막에 code.
func TextRoles() []TextRole {
	return []TextRole{RoleTitle, RoleHeading, RoleBody, RoleCaption, RoleCode}
}

// TextAlign은 문단 정렬이다. 값은 `left`/`right`가 아니라 **논리 이름**이다.
//
// 이유: 양 타깃이 논리 이름을 그대로 받는다 — CSS는 `text-align: start|end`, OOXML은
// `a:pPr@algn`이 문단 방향(`rtl`)에 따라 `l`/`r`의 뜻을 뒤집는다. 물리 이름으로 저장하면
// RTL 텍스트에서 "글이 시작하는 쪽"이라는 저자의 의도를 잃고, 그 손실은 저장 시점에
// 일어나 복원할 수 없다.
type TextAlign string

const (
	AlignStart   TextAlign = "start"
	AlignCenter  TextAlign = "center"
	AlignEnd     TextAlign = "end"
	AlignJustify TextAlign = "justify"
)

// TextAligns는 정렬 어휘 전체다 (순서 = 진단 메시지 순서).
func TextAligns() []TextAlign {
	return []TextAlign{AlignStart, AlignCenter, AlignEnd, AlignJustify}
}

// Color는 해석이 끝난 색이다. **토큰 참조**와 **리터럴 RGB** 두 상태를 갖는다.
//
// 토큰으로 선언된 색을 여기서 RGB로 눕히지 않는 것이 이 제품의 핵심 계약이다: PPTX
// 렌더러는 토큰 색을 `schemeClr` 참조로 내보내고, 그래서 내보낸 파일이 PowerPoint 안에서도
// 테마 색 편집을 따라간다 (PRD §9.4). RGB로 눕히면 그 참조를 복원할 방법이 없다.
type Color struct {
	// Token은 테마 색 토큰 이름이다 (Theme.Color로 값을 찾는다). 비어 있으면 리터럴 색이다.
	Token string
	// RGB는 리터럴 색이다 — 대문자 6자리 16진수이고 `#`는 붙지 않는다. 웹은 `#`를,
	// OOXML은 맨몸 16진수를 요구하므로 접두는 각 렌더러가 붙인다.
	RGB string
}

// IsToken은 이 색이 토큰 참조인지 보고한다.
func (c Color) IsToken() bool { return c.Token != "" }

// IsZero는 색이 정해지지 않았는지 보고한다 (해석 후에는 있을 수 없다 — 역할 기본값이 채운다).
func (c Color) IsZero() bool { return c.Token == "" && c.RGB == "" }

// TokenColor는 토큰 참조 색이다.
func TokenColor(name string) Color { return Color{Token: name} }

// LiteralColor는 리터럴 색이다. rgb는 대문자 6자리 16진수여야 한다 (정규화는 core가 한다).
func LiteralColor(rgb string) Color { return Color{RGB: rgb} }

// FontRole은 폰트 토큰 이름이다 (PRD §8.4의 `font` 토큰 3종).
type FontRole string

const (
	FontHeading FontRole = "heading"
	FontBody    FontRole = "body"
	FontMono    FontRole = "mono"
)

// Theme는 덱에 실린 토큰 표다 — 해석된 색·폰트 참조가 가리키는 곳이다.
//
// 표를 덱 모델에 싣는 이유: 웹의 CSS 변수와 PPTX의 테마 파트(clrScheme/fontScheme)가
// **같은 한 표에서** 생성되어야 두 타깃의 색·폰트가 구조적으로 같아진다. 표가 렌더러마다
// 따로 있으면 값이 갈라지는 것을 막을 방법이 없고, 갈라진 것을 알아채는 유일한 방법은
// 사람이 두 산출물을 나란히 보는 것이다.
type Theme struct {
	Name string
	// Colors는 색 토큰이다. 맵이 아니라 순서 있는 목록인 이유는 산출물(CSS 변수 선언,
	// clrScheme)과 진단 메시지가 결정적이어야 하기 때문이다.
	Colors []ColorToken
	Fonts  []FontToken
}

// Color는 이름으로 색 토큰을 찾는다.
func (t Theme) Color(name string) (ColorToken, bool) {
	for _, token := range t.Colors {
		if token.Name == name {
			return token, true
		}
	}
	return ColorToken{}, false
}

// ColorNames는 색 토큰 이름 목록이다 — 진단 메시지가 허용 어휘를 나열하는 근거다.
func (t Theme) ColorNames() []string {
	names := make([]string, len(t.Colors))
	for i, token := range t.Colors {
		names[i] = token.Name
	}
	return names
}

// Font는 이름으로 폰트 토큰을 찾는다.
func (t Theme) Font(name FontRole) (FontToken, bool) {
	for _, token := range t.Fonts {
		if token.Name == name {
			return token, true
		}
	}
	return FontToken{}, false
}

// ColorToken은 색 토큰 하나다.
//
// v1의 **모든** 색 토큰은 clrScheme 슬롯을 갖는다. 슬롯 없는 토큰을 만들면 그 색만
// PowerPoint의 테마 색 편집을 따라가지 못해, 사용자가 색을 바꿨을 때 덱 안에서 색 하나가
// 어긋난 채 남는다 — 그 상태는 파일을 열어 보기 전까지 보이지 않는다.
type ColorToken struct {
	// Name은 표준 토큰 어휘의 이름이다 (PRD §8.4): fg, bg, muted, accent1 ….
	Name string
	// RGB는 대문자 6자리 16진수다 (`#` 없음).
	RGB string
	// Slot은 OOXML 테마 파트의 clrScheme 자식 이름이다 (dk1/lt1/dk2/lt2/accent1…).
	Slot string
	// Ref는 도형이 이 색을 참조할 때 쓰는 `schemeClr@val`이다 (tx1/bg1/tx2/bg2/accent1…).
	// 슬롯 이름과 다른 까닭은 마스터의 clrMap이 둘을 잇기 때문이다 — 슬롯 dk1은 참조 tx1이
	// 된다. 파생 규칙을 렌더러에 두지 않고 표에 적어 두는 편이 검증 가능하다.
	Ref string
}

// FontToken은 폰트 토큰 하나다.
type FontToken struct {
	Name FontRole
	// Latin과 EastAsian은 OOXML fontScheme가 요구하는 구분이다 (`latin`/`ea`). 웹은 이 둘과
	// Generic을 이어 font-family 스택으로 낸다 — 두 타깃의 폰트 순서가 한 표에서 나온다.
	Latin     string
	EastAsian string
	// Generic은 스택 끝의 CSS 제네릭 패밀리다 (sans-serif/monospace). 폰트가 없는 환경의
	// 마지막 폴백이며 OOXML에는 대응 개념이 없다 — 그 자리는 뷰어의 폰트 대체가 맡는다.
	Generic string
	// Scheme은 fontScheme 슬롯이다 ("major"/"minor"). mono는 빈 문자열이다: OOXML
	// fontScheme에는 major/minor 두 자리뿐이어서 세 번째 폰트를 실을 곳이 없고, 렌더러가
	// 리터럴 typeface로 낸다 (테마 폰트 편집을 따라가지 못하는 유일한 예외).
	Scheme string
}

// Family는 웹 font-family 스택이다 (우선순위 순, 중복·빈 항목 제거).
func (f FontToken) Family() []string {
	stack := make([]string, 0, 3)
	for _, name := range []string{f.Latin, f.EastAsian, f.Generic} {
		if name == "" {
			continue
		}
		dup := false
		for _, seen := range stack {
			if seen == name {
				dup = true
				break
			}
		}
		if !dup {
			stack = append(stack, name)
		}
	}
	return stack
}
