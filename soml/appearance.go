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
	Text *TextStyle
	// Shape는 도형 외관이다 (shapestyle.go) — `<shape>`에만 채워진다. Text와 형제인 것이
	// 이 계약이 텍스트 밖으로 처음 나간 자리다: 값 어휘·캐스케이드를 core가 한 번만
	// 적용하고 렌더러는 읽기만 한다. 도형 외관이 값 층을 두 번 지나면 웹의 CSS와 PPTX의
	// OOXML이 갈라진다.
	Shape *ShapeStyle
	// Image는 이미지 외관이다 (imagestyle.go) — `<image>`에만 채워진다. Shape와 형제로
	// 두고 **한 구조체로 합치지 않는 이유**는 담을 것이 다르다는 것이다: 도형은
	// `type`·`fill`·`id`를, 그림은 `src`·`fit`·`alt`를 갖는다. 값 문법이 겹치는 부분은
	// 구조가 아니라 **함수**로 공유한다 (core의 border·radius·shadow 해석).
	Image *ImageStyle
	// Link는 하이퍼링크 목적지다 (<a href>의 값, 원문 그대로). 외관이 아니라 **행동**이라
	// TextStyle 밖에 둔다 — 밑줄·색은 스타일이고, 목적지는 렌더러가 웹의 href와 PPTX의
	// hlinkClick으로 잇는다. 여기 두는 것은 "렌더러가 Attrs를 다시 읽지 않는다"는 계약의
	// 연장이다: 렌더러가 href를 Attrs에서 찾기 시작하면 값 검증(core)을 지나지 않은 값이
	// 산출물에 실릴 길이 생긴다.
	Link string
	// Container는 확정된 컨테이너 배치다 (placement.go) — 레이아웃 컨테이너에만 채워진다.
	// 외관과 형제로 두는 이유는 계약이 같기 때문이다: 값 층이 한 번 해석하고 렌더러는
	// 읽기만 한다. 배치가 값 층을 두 번 지나면 CSS와 지오메트리 패스가 갈라진다.
	Container *ContainerLayout
	// Placement는 부모 안에서 자기 자리를 정하는 확정 배치다 — 배치 속성이 하나도 없으면
	// nil로 남는다 (그 nil이 배치 없는 요소의 산출물을 이전과 같게 유지한다).
	Placement *Placement
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
	// Italic은 이탤릭 여부다 (<em>의 런 델타 — core의 인라인 해석이 채운다). 웹 CSS의
	// font-style: italic과 OOXML rPr@i가 둘 다 불리언으로 직접 받는다.
	Italic bool
	// Underline은 밑줄 여부다 (<a>의 런 델타). 웹 text-decoration: underline과 OOXML
	// rPr@u가 직접 받는다. 링크의 목적지는 스타일이 아니므로 Appearance.Link에 있다.
	Underline bool
	// LineHeight는 폰트 크기 배수다 (무단위). 웹은 CSS line-height로 배수를 그대로 받지만
	// PPTX는 **절대 줄간격**(lnSpc@spcPts)으로 환산해 낸다 (render/pptx.LineSpacing) —
	// spcPct(배수)는 엔진마다 기준이 달라(CSS는 폰트 크기, LibreOffice는 폰트 고유 줄 높이)
	// 같은 150%가 글줄을 밀었다 (PRD §13-1의 측정, NOTES.md 01/07). 배수 표현을 여기
	// 유지하는 이유: "크기의 몇 배"가 저자와 테마의 의도이고, 절대값 환산은 크기가 확정된
	// 뒤 타깃 경계에서만 가능하다.
	LineHeight float64
	Color      Color
	Align      TextAlign
	// Anchor는 **세로 앵커**다 (PRD §7.6의 text 행). 정하지 않으면 빈 문자열이다.
	//
	// 다른 필드와 달리 역할 기본값이 채우지 않는 유일한 자리다: 앵커가 뜻을 갖는 곳은
	// 내용보다 큰 박스 안이고, 그런 박스는 도형뿐이다 (슬라이드 직속 텍스트 박스는 잰
	// rect가 곧 내용 크기라 앵커가 움직일 여백이 없다). 기본값을 정하면 모든 텍스트가
	// `bodyPr@anchor`·flex 정렬 선언을 하나 더 싣게 되고, 그 선언은 뜻이 없는 자리에서도
	// 산출물을 흔든다.
	Anchor TextAnchor
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

// TextAnchor는 박스 안 텍스트의 **세로 정렬**이다 (`anchor`).
//
// 값은 TextAlign과 달리 **물리 이름**이다 (`start`/`end`가 아니라 `top`/`bottom`).
// TextAlign이 논리 이름을 고른 근거는 문단 방향(rtl)이 가로축의 뜻을 뒤집는다는 것이고,
// **세로축에는 그 근거가 없다** — v1은 세로쓰기를 지원하지 않으므로 위는 항상 위다. 논리
// 이름을 쓰면 두 축의 어휘가 같아 보이는 대신 "무엇이 뒤집힐 수 있는가"라는 구별을 잃는다.
// 가운데만 TextAlign과 이름을 공유한다 (`center` — 뒤집힘과 무관한 값이다).
type TextAnchor string

const (
	AnchorTop    TextAnchor = "top"
	AnchorCenter TextAnchor = "center"
	AnchorBottom TextAnchor = "bottom"
)

// TextAnchors는 세로 앵커 어휘 전체다 (순서 = 위에서 아래로 = 진단 메시지 순서).
func TextAnchors() []TextAnchor {
	return []TextAnchor{AnchorTop, AnchorCenter, AnchorBottom}
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
	// Spaces는 공간 토큰이다 (PRD §8.4의 `space` 스케일) — 레이아웃 태그의 `gap`/`pad`
	// 값 어휘가 여기서 온다. 색·폰트와 함께 이 표에 있는 이유: 배치조차 테마의 리듬을
	// 따라가야 두 타깃의 여백이 한 곳에서 나온다.
	Spaces []SpaceToken
	// Radii는 모서리 반경 토큰이다 (PRD §8.4의 `shape.radius-*`) — `radius="lg"`의 값
	// 어휘다.
	Radii []RadiusToken
	// Shadows는 그림자 토큰이다 (PRD §8.4의 `shape.shadow-soft`) — `shadow="soft"`의 값
	// 어휘이자 **유일한** 어휘다 (도형 외관은 그림자 수치를 받지 않는다 — ShadowToken).
	Shadows []ShadowToken
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

// Space는 이름으로 공간 토큰을 찾는다.
func (t Theme) Space(name string) (SpaceToken, bool) {
	for _, token := range t.Spaces {
		if token.Name == name {
			return token, true
		}
	}
	return SpaceToken{}, false
}

// SpaceNames는 공간 토큰 이름 목록이다 — 모르는 토큰 진단이 허용 어휘를 나열하는 근거다
// (ColorNames와 같은 태도).
func (t Theme) SpaceNames() []string {
	names := make([]string, len(t.Spaces))
	for i, token := range t.Spaces {
		names[i] = token.Name
	}
	return names
}

// Radius는 이름으로 반경 토큰을 찾는다.
func (t Theme) Radius(name string) (RadiusToken, bool) {
	for _, token := range t.Radii {
		if token.Name == name {
			return token, true
		}
	}
	return RadiusToken{}, false
}

// RadiusNames는 반경 토큰 이름 목록이다 (ColorNames와 같은 태도).
func (t Theme) RadiusNames() []string {
	names := make([]string, len(t.Radii))
	for i, token := range t.Radii {
		names[i] = token.Name
	}
	return names
}

// Shadow는 이름으로 그림자 토큰을 찾는다.
func (t Theme) Shadow(name string) (ShadowToken, bool) {
	for _, token := range t.Shadows {
		if token.Name == name {
			return token, true
		}
	}
	return ShadowToken{}, false
}

// ShadowNames는 그림자 토큰 이름 목록이다 (ColorNames와 같은 태도).
func (t Theme) ShadowNames() []string {
	names := make([]string, len(t.Shadows))
	for i, token := range t.Shadows {
		names[i] = token.Name
	}
	return names
}

// RadiusToken은 모서리 반경 토큰 하나다 — `radius="lg"`가 가리키는 곳.
//
// **clrScheme 슬롯 계약이 여기에는 없다** (SpaceToken과 같은 비대칭이지만 이유가 다르다):
// OOXML에 반경이라는 **테마 개념이 아예 없다.** 반경은 프리셋 지오메트리의 조정값
// (`prstGeom/avLst`의 `adj`)이라 도형마다 구워지고, 테마 파트에서 조정값을 가리키는 자리가
// 없다. PowerPoint에서 모서리를 바꾸는 것은 도형의 노란 핸들을 끄는 일이지 토큰을 바꾸는
// 일이 아니다 — 배치가 좌표로 구워지는 것과 같은 종류의 손실이다.
type RadiusToken struct {
	// Name은 PRD §8.4의 `shape.radius-*` 어휘다: sm, md, lg.
	Name string
	// Px는 논리 스테이지 px다 (PRD §13-5).
	Px float64
}

// ShadowToken은 그림자 토큰 하나다 — `shadow="soft"`가 가리키는 곳.
//
// **도형 외관은 그림자를 토큰으로만 받는다** (수치 축약 없음). 근거는 CSS `box-shadow`의
// 4수치와 OOXML `outerShdw`가 1:1이 아니라는 것이다: 번짐 반경의 정의가 다르고(CSS는
// 가우시안 표준편차의 2배, OOXML `blurRad`는 자체 정의), CSS의 확산(`spread`)에는 대응
// 항목이 아예 없다. 저자가 수치를 쓰면 두 타깃이 다르게 그려지고, 토큰이면 **양쪽 값을
// 테마가 소유한다** — FontToken이 Latin/EastAsian/Generic을 한 표에서 내는 것과 같은 계약을
// 그림자로 넓힌 것이다.
//
// 그래서 필드는 **두 타깃을 다 만들 수 있는 파라미터**다. 오프셋을 극좌표(Dist·Dir)로 담는
// 이유: OOXML `outerShdw@dist/dir`가 극좌표이고 CSS는 직교(dx/dy)이므로 한쪽을 골라야
// 하는데, 극좌표 → 직교는 렌더 시점의 삼각함수 한 번으로 정확히 나오고 반대 방향은
// `atan2`의 결과를 60000분의 1도 정수로 반올림해야 한다. 손실이 있는 방향을 산출 경계에
// 두지 않는다.
//
// **clrScheme 슬롯 계약**: 그림자 자체에는 없지만 색에는 있다 (Color가 토큰 참조를
// 유지하므로 `outerShdw`가 `schemeClr`를 낼 수 있다). 테마 파트에 그림자를 실을 자리가
// **없지는 않다** — `fmtScheme/effectStyleLst`가 그것이다. 그런데 그 목록은 스타일이 세
// 개뿐이고 도형이 `style/effectRef`로 하나를 고르는 구조라, 도형마다 다른 그림자를 실을 수
// 없다. 즉 "없다"가 아니라 **"있지만 v1은 쓰지 않는다"**다 (매핑의 실제 결정은 항목 05).
type ShadowToken struct {
	// Name은 PRD §8.4의 `shape.shadow-*` 어휘다: soft.
	Name string
	// Dist는 그림자가 도형에서 밀려난 거리다 (논리 px).
	Dist float64
	// Dir는 그림자가 밀려난 방향이다 (도, 시계방향, 0 = 오른쪽). 웹과 OOXML이 같은 규약을
	// 쓴다 — y축이 아래로 향하므로 90도가 아래다.
	Dir float64
	// Blur는 번짐 반경이다 (논리 px). 두 엔진의 정의가 정확히 같지 않다는 사실이 값을
	// 토큰에 가둔 이유다 (타입 주석).
	Blur float64
	// Color는 그림자 색이다 — 토큰 참조가 유지된다 (PPTX가 `schemeClr`를 내려면 필요하다).
	Color Color
	// Alpha는 그림자 색의 불투명도다 (0~1). 색과 분리한 이유: 색 변형 함수 `alpha()`는
	// Iteration 8이고, 그때까지 토큰 색에 투명도를 실을 표기가 값 문법에 없다. 그림자는
	// 반투명이 기본이므로 표가 그 값을 직접 갖는다.
	Alpha float64
}

// SpaceToken은 공간 토큰 하나다 — `gap="xl"`이 가리키는 곳.
//
// **ColorToken의 clrScheme 슬롯 계약이 여기에는 없다.** 색 토큰은 전부 PPTX 테마 슬롯을
// 가져서 내보낸 파일이 PowerPoint의 테마 색 편집을 따라가지만, OOXML 테마 파트에는 공간이라는
// 개념이 아예 없다. 공간은 배치이고 **배치는 좌표로 구워져 산출물에 남는다** — PPTX에서
// gap을 편집한다는 것은 도형을 옮긴다는 뜻이지 토큰을 바꾸는 일이 아니다. 이 비대칭은
// 두 채널(외관/배치)이 타깃 경계에서 다르게 살아남는다는 사실의 직접적인 결과다.
type SpaceToken struct {
	// Name은 표준 토큰 어휘의 이름이다 (PRD §8.4): xs, sm, md, lg, xl, 2xl.
	Name string
	// Px는 논리 스테이지 px다 (PRD §13-5) — pt·EMU 환산은 타깃 경계에서만 일어난다.
	Px float64
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
