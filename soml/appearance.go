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
	// Text는 텍스트 외관이다. <text>와 텍스트 서브셋 태그에만 채워진다.
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
	// Table은 표 프레임 외관이다 (tablestyle.go) — `<table>`에만 채워진다. Shape·Image와
	// 형제로 두고 합치지 않는 이유는 담을 것이 다르다는 것이다: 표 프레임은 변형 이름을 갖고,
	// 셀 외관(채움·테두리·텍스트)은 Cell이 따로 갖는다. 값 어휘(변형 폴백)는 core가 한 번만
	// 본다 — 렌더러는 이 표시자로 "해석 끝난 표"를 확인하고 웹 변형 훅의 근거를 여기서 읽는다.
	Table *TableStyle
	// Cell은 셀 외관이다 (tablestyle.go) — `<th>`/`<td>`에만 채워진다. 표 프레임과 별개로 두는
	// 이유는 소비 단위가 다르다는 것이다: 프레임은 도형 하나, 셀은 격자 안 `a:tc`/`<td>` 하나다.
	// 헤더/본문/줄무늬 기본값 위에 인라인 셀 속성이 얹힌 확정 값이 여기 담긴다.
	Cell *CellStyle
	// Link는 하이퍼링크 목적지다 (<a href>의 값, 원문 그대로). 외관이 아니라 **행동**이라
	// TextStyle 밖에 둔다 — 밑줄·색은 스타일이고, 목적지는 렌더러가 웹의 href와 PPTX의
	// hlinkClick으로 잇는다. 여기 두는 것은 "렌더러가 Attrs를 다시 읽지 않는다"는 계약의
	// 연장이다: 렌더러가 href를 Attrs에서 찾기 시작하면 값 검증(core)을 지나지 않은 값이
	// 산출물에 실릴 길이 생긴다.
	Link string
	// Reveal은 이 리스트 항목이 나타나는 스텝 번호다 (`<li reveal>`의 확정 값).
	// 0은 "순차 등장 없음"이고, 그 0이 reveal을 쓰지 않은 덱의 산출물을 이전과 같게 유지한다
	// (Placement의 nil·Slot의 빈 문자열과 같은 태도).
	//
	// Link와 같은 이유로 TextStyle 밖에 둔다: 이것은 외관이 아니라 **발표 순서**라 캐스케이드·
	// variant·테마 기본값이 없고, 값 층이 슬라이드마다 1부터 센 번호를 렌더러는 읽기만 한다.
	// 이 번호를 소비하는 타깃은 웹 **하나**이고(발표자 뷰의 숨김), PPTX·인쇄·측정은 이 필드를
	// 보지 않는다 — 평탄화가 "무시"로 구현되는 자리다.
	Reveal int
	// Enter는 확정된 등장이다 (motion.go) — `enter`를 가졌거나 `<slide enter>`를 물려받은 요소에만 실린다.
	// Reveal과 같은 이유로 TextStyle 밖이다: 외관이 아니라 발표 순서다.
	Enter *Entrance
	// Connector는 확정된 연결선이다 (connector.go) — `<connector>`에만 채워진다. 렌더러는 잇는 두
	// 상자의 잰 rect와 이 값으로 경로를 계산한다 (render/route).
	Connector *ConnectorStyle
	// Container는 확정된 컨테이너 배치다 (placement.go) — 레이아웃 컨테이너에만 채워진다.
	// 외관과 형제로 두는 이유는 계약이 같기 때문이다: 값 층이 한 번 해석하고 렌더러는
	// 읽기만 한다. 배치가 값 층을 두 번 지나면 CSS와 지오메트리 패스가 갈라진다.
	Container *ContainerLayout
	// Box는 **텍스트 요소 자신의 박스**다 (`<text>`·블록·리스트 컨테이너의 `w`·`min-w`·
	// `max-w`). 크기 속성을 하나도 쓰지 않으면 nil로 남는다.
	//
	// 컨테이너·도형은 이 필드를 쓰지 않는다 — 그쪽은 Container.Box·Shape.Box가 같은 구조체를
	// 임베드로 갖는다. 텍스트에 세 번째 자리를 만든 이유는 TextStyle에 넣지 않은 이유와 같다:
	// 크기는 외관이 아니라 박스의 성질이고(캐스케이드·variant·역할 기본값이 없다), TextStyle에
	// 섞으면 테마 번들이 폭을 가질 수 있다는 뜻이 되어 버린다. 값 문법은 컨테이너·도형과 **같은
	// 헬퍼**가 해석한다 (core.boxAttr) — 열린 속성만 가로 3종으로 좁혀진다.
	Box *Box
	// Placement는 부모 안에서 자기 자리를 정하는 확정 배치다 — 배치 속성이 하나도 없으면
	// nil로 남는다 (그 nil이 배치 없는 요소의 산출물을 이전과 같게 유지한다).
	Placement *Placement
	// Slot은 이 요소가 배정된 레이아웃 슬롯 이름이다 (`slot="title"`).
	// **슬라이드 직계 자식에만** 채워지고, 그 자식이 `slot` 속성을 쓴 경우에만 채워진다 —
	// 슬롯 없는 직계 자식과 중첩 요소는 빈 문자열이다 (core.Resolve가 어휘를 검증하고 채운다).
	// 빈 값이 곧 "슬롯 배치 없음"이라, 슬롯을 쓰지 않은 덱의 산출물이 이 필드 때문에 달라지지
	// 않는다 (Placement의 nil과 같은 태도). 렌더러는 이 이름으로 웹 grid-area와 PPTX `p:ph`를
	// 낸다 — 이름→(phType·idx)·영역 매핑은 soml.LayoutSlot·테마가 소유한다.
	Slot string
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
	// Size는 논리 스테이지 px다. pt·EMU 환산은 타깃 경계에서만 일어난다.
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
	// 같은 150%가 엔진마다 다른 높이로 글줄을 밀었다. 배수 표현을 여기
	// 유지하는 이유: "크기의 몇 배"가 저자와 테마의 의도이고, 절대값 환산은 크기가 확정된
	// 뒤 타깃 경계에서만 가능하다.
	LineHeight float64
	Color      Color
	Align      TextAlign
	// Anchor는 **세로 앵커**다 (`anchor`: top·center·bottom). 정하지 않으면 빈 문자열이다.
	//
	// 다른 필드와 달리 역할 기본값이 채우지 않는 유일한 자리다: 앵커가 뜻을 갖는 곳은
	// 내용보다 큰 박스 안이고, 그런 박스는 도형뿐이다 (슬라이드 직속 텍스트 박스는 잰
	// rect가 곧 내용 크기라 앵커가 움직일 여백이 없다). 기본값을 정하면 모든 텍스트가
	// `bodyPr@anchor`·flex 정렬 선언을 하나 더 싣게 되고, 그 선언은 뜻이 없는 자리에서도
	// 산출물을 흔든다.
	Anchor TextAnchor
	// Tracking은 **자간**이다 — 글자 크기의 배수(em). 0은 "자간 없음"이고 아무 선언도 나가지
	// 않는다. 웹은 `letter-spacing: <n>em`, PPTX는 `a:rPr@spc`(1/100pt)로 환산한다 — 배수로
	// 들고 있는 이유는 LineHeight와 같다: "크기의 몇 배"가 테마의 의도이고, 절대값은 크기가
	// 확정된 뒤 타깃 경계에서만 정해진다. 저자 속성이 아니라 테마 필드다 (역할·variant).
	Tracking float64
	// Case는 **표시 대소문자**다 — 빈 값은 원문 그대로, CaseUpper는 대문자로 보인다. 원문은
	// 바뀌지 않는다: 웹 `text-transform`과 PPTX `a:rPr@cap="all"`이 둘 다 표시 속성이라
	// PowerPoint에서 글을 고치면 저자가 쓴 대소문자로 편집되고, 테마를 갈면 대문자가 사라진다.
	Case TextCase
	// Highlight는 런 뒤에 칠하는 **형광펜 색**이다 (`<mark>`의 런 델타 — 테마의
	// `text.mark.highlight`). nil은 형광펜 없음이다. 웹은 `background-color`, PPTX는
	// `a:rPr/a:highlight`다.
	Highlight *Color
}

// TextCase는 표시 대소문자다. 어휘는 둘이고, 원문 그대로가 빈 값이다 — 테마 파일의 `"none"`은
// 아래 단의 `upper`를 끄는 패치이고 확정 값으로는 빈 값이 된다.
type TextCase string

const (
	// CaseUpper는 대문자 표시다.
	CaseUpper TextCase = "upper"
)

// TextRole은 텍스트 역할이다 — 1층의 HTML 서브셋(`<p>`, `<h2>` …)과 2층의 `<text role>`이
// 공유하는 타이포그래피 계약이다. 어휘는 5종으로 닫혀 있다.
type TextRole string

const (
	RoleTitle   TextRole = "title"
	RoleHeading TextRole = "heading"
	RoleBody    TextRole = "body"
	RoleCaption TextRole = "caption"
	RoleCode    TextRole = "code"
)

// DefaultTextRole은 `role`을 생략한 `<text>`의 역할이다. `<p>` ≡ `<text role="body">`라는
// 등가의 다른 쪽 끝이다.
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
// 테마 색 편집을 따라간다. RGB로 눕히면 그 참조를 복원할 방법이 없다.
type Color struct {
	// Token은 테마 색 토큰 이름이다 (Theme.Color로 값을 찾는다). 비어 있으면 리터럴 색이다.
	Token string
	// RGB는 리터럴 색이다 — 대문자 6자리 16진수이고 `#`는 붙지 않는다. 웹은 `#`를,
	// OOXML은 맨몸 16진수를 요구하므로 접두는 각 렌더러가 붙인다.
	RGB string
	// Mod는 색 변형 함수다 (`shade`/`tint`/`alpha`). nil이면
	// 변형 없는 색이다. **변형은 토큰 참조를 눕히지 않는다**: 변형된 토큰 색은 Token을
	// 그대로 두고 Mod만 얹으므로, PPTX가 `schemeClr` + `lumMod`/`lumOff`/`alpha`로 내어
	// 테마 색 편집을 여전히 따라간다. 그래서 Mod는 Gradient와 달리 Token/RGB와
	// **함께** 산다 — 변형의 밑색이 곧 이 Color의 Token/RGB다.
	Mod *ColorMod
	// Gradient는 그라디언트다 (`gradient(…)`). nil이면 단색이다.
	// 단색과 달리 Token/RGB를 쓰지 않는다: 그라디언트는 **여러 스톱**이라 한 색 참조에
	// 담을 수 없다. 스톱 각각이 토큰·리터럴을 유지하므로 PPTX `gradFill`의 `gs`가 스톱마다
	// `schemeClr`/`srgbClr`를 낸다.
	Gradient *Gradient
}

// ColorMod는 단색에 얹는 색 변형이다 (`shade`/`tint`/`alpha`).
//
// **중첩은 없다** (v1): 변형의 밑색은 토큰·리터럴 단색뿐이고, 변형된 색을 다시 변형하거나
// 그라디언트 스톱으로 쓰지 못한다. 이 제약이 매핑을 1:1로 유지한다 — 웹은 `color-mix()`
// (또는 알파) 한 겹, PPTX는 색 요소의 자식(`lumMod`/`lumOff`/`alpha`) 한 벌이다.
type ColorMod struct {
	// Kind는 변형 종류다 (ColorMods의 3종).
	Kind ColorModKind
	// Amount는 변형 세기다. `shade`/`tint`는 백분율(0~100), `alpha`는 0~1이다 — 단위가
	// 종류마다 다른 이유는 각 함수의 자연 단위이기 때문이고, 타깃 환산(웹 `%`, PPTX
	// 1000분율·100000분율)은 렌더러가 Kind를 보고 한다.
	Amount float64
}

// ColorModKind는 색 변형 종류다 — 닫힌 어휘 3종.
type ColorModKind string

const (
	// ModShade는 색을 어둡게 한다 (검정과 섞음). 웹 `color-mix(… black P%)`, PPTX `lumMod`.
	ModShade ColorModKind = "shade"
	// ModTint는 색을 밝게 한다 (흰색과 섞음). 웹 `color-mix(… white P%)`, PPTX `lumMod`+`lumOff`.
	ModTint ColorModKind = "tint"
	// ModAlpha는 색을 투명하게 한다. 웹 `color-mix(transparent …)`, PPTX `a:alpha`.
	ModAlpha ColorModKind = "alpha"
)

// ColorMods는 변형 어휘 전체다 (순서 = 진단 메시지 순서).
func ColorMods() []ColorModKind { return []ColorModKind{ModShade, ModTint, ModAlpha} }

// Gradient는 확정된 그라디언트다 (`gradient(색… [, 각도deg])`).
//
// 스톱은 **균등 배치**다 (스톱 위치 지정은 v2) — n개 스톱은 0, 1/(n-1), …, 1에 놓인다.
// 각도는 CSS `linear-gradient` 시맨틱(0deg = 아래→위, 시계방향)이고, OOXML `gradFill@ang`
// 변환은 렌더러의 몫이다 (render/pptx.GradientAngle). 스톱은 단색(Token/RGB)만 담으며
// 변형·중첩 그라디언트를 담지 않는다 — 그 제약은 core가 파싱 때 건다.
type Gradient struct {
	// Stops는 그라디언트 색 목록이다 (문서 순서 = 스톱 0→1). 둘 이상이어야 한다. 스톱은 불투명도(alpha 변형)를
	// 가질 수 있다 (`bg/0`).
	Stops []Color
	// Angle은 CSS 각도(도)다 — 생략 시 180(위→아래). 방사형에는 뜻이 없다.
	Angle float64
	// Radial은 방사형인지다 (`radial(…)`). CX · CY는 중심이다 — 박스의 백분율(0~100), 기본은 50 · 50.
	Radial bool
	CX, CY float64
}

// IsToken은 이 색이 토큰 참조인지 보고한다 (변형된 토큰 색도 참조로 센다 — Mod는 밑색을
// 눕히지 않는다).
func (c Color) IsToken() bool { return c.Token != "" }

// IsGradient는 이 색이 그라디언트인지 보고한다.
func (c Color) IsGradient() bool { return c.Gradient != nil }

// IsVariant는 이 색에 변형이 얹혀 있는지 보고한다 (`shade`/`tint`/`alpha`).
func (c Color) IsVariant() bool { return c.Mod != nil }

// IsZero는 색이 정해지지 않았는지 보고한다 (해석 후에는 있을 수 없다 — 역할 기본값이 채운다).
func (c Color) IsZero() bool { return c.Token == "" && c.RGB == "" && c.Gradient == nil }

// TokenColor는 토큰 참조 색이다.
func TokenColor(name string) Color { return Color{Token: name} }

// LiteralColor는 리터럴 색이다. rgb는 대문자 6자리 16진수여야 한다 (정규화는 core가 한다).
func LiteralColor(rgb string) Color { return Color{RGB: rgb} }

// GradientColor는 그라디언트 색이다 (스톱 목록 + CSS 각도).
func GradientColor(stops []Color, angle float64) Color {
	return Color{Gradient: &Gradient{Stops: stops, Angle: angle}}
}

// RadialColor는 방사형 그라디언트 색이다 — 중심 (cx, cy)는 박스의 백분율이다.
func RadialColor(stops []Color, cx, cy float64) Color {
	return Color{Gradient: &Gradient{Stops: stops, Radial: true, CX: cx, CY: cy}}
}

// VariantColor는 단색 base에 변형을 얹은 색이다 — base의 Token/RGB를 유지한다
// (schemeClr 보존). base가 이미 변형·그라디언트면 안 된다 (그 검증은 core).
func VariantColor(base Color, kind ColorModKind, amount float64) Color {
	base.Mod = &ColorMod{Kind: kind, Amount: amount}
	return base
}

// FontRole은 폰트 토큰 이름이다 (테마의 `font` 토큰 3종).
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
	// Spaces는 공간 토큰이다 (테마의 `space` 스케일) — 레이아웃 태그의 `gap`/`pad`
	// 값 어휘가 여기서 온다. 색·폰트와 함께 이 표에 있는 이유: 배치조차 테마의 리듬을
	// 따라가야 두 타깃의 여백이 한 곳에서 나온다.
	Spaces []SpaceToken
	// Radii는 모서리 반경 토큰이다 (테마의 `radius` 토큰) — `radius="lg"`의 값
	// 어휘다.
	Radii []RadiusToken
	// Shadows는 그림자 토큰이다 (테마의 `shadow` 토큰) — `shadow="soft"`의 값
	// 어휘이자 **유일한** 어휘다 (도형 외관은 그림자 수치를 받지 않는다 — ShadowToken).
	Shadows []ShadowToken
	// Motion은 등장의 박자다 (테마의 `motion` 토큰 — motion.go). nil이면 테마가 박자를 정하지 않았고,
	// 그 테마에서 `enter`를 쓴 덱은 값 층의 에러다.
	Motion *MotionTokens
	// Layouts는 레이아웃 이름 → 스테이지에 얹는 **정규화된 배치 선언**이다 (`layouts/*.css`의
	// 실체). 색·폰트처럼 이 표에 있는 이유: 배치도 테마가 소유하고, 웹의 스테이지
	// CSS와 지오메트리 패스가 재는 박스가 **같은 한 표**에서 나와야 두 타깃의 배치가 갈라지지
	// 않는다. 값은 주석을 벗긴 `프로퍼티: 값; …` 문자열이고, 배치 CSS가 없는 레이아웃
	// (content 세로 스택·canvas 자유 배치는 렌더러가 소유)은 **맵에 없다** — 없다는 것이 곧
	// "스테이지에 레이아웃 클래스를 붙이지 않는다"는 신호다 (render/web writeStage).
	Layouts map[string]string
	// SlotLayouts는 레이아웃 이름 → **슬롯 영역 배치 선언**이다 (`layouts/*.slots.css`의 실체).
	// Layouts와 형제이지만 겨누는 선택자가 다르다: Layouts는 흐름 배치
	// (문서 순서)를 `.balpyo-layout-<이름>`에 얹고, 이 표는 **슬롯 영역**(grid-template-areas)을
	// `.balpyo-layout-<이름>.balpyo-slots`에 얹는다 — 슬롯을 실제로 쓴 슬라이드에만 붙는
	// 모디파이어라(render/web writeStage), 슬롯을 쓰지 않은 덱의 흐름 배치는 이 표 때문에
	// 달라지지 않는다 (골든 불변). 슬롯 영역이 없는 레이아웃(canvas·content)은 맵에 없다.
	SlotLayouts map[string]string
	// Assets는 테마 패키지의 그림 가운데 **선언이 가리키는 것**이다 (배경 · 로고, 그리고 슬라이드의
	// `<image src="theme:…">`) — 키는 `themes/<그 파일을 가진 테마>/assets/…`다. 덱의 에셋 표(Deck.Assets)와 같은 항목 모양이고, 에셋
	// 패스가 슬라이드가 실제로 쓴 것만 그 표로 옮긴다 — 렌더러는 덱의 표 하나만 읽는다.
	Assets Assets
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
	// Name은 radius 토큰 어휘다: sm, md, lg.
	Name string
	// Px는 논리 스테이지 px다.
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
// 없다. 즉 "없다"가 아니라 **"있지만 v1은 쓰지 않는다"**다.
type ShadowToken struct {
	// Name은 shadow 토큰 어휘다: soft.
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
	// Alpha는 그림자 색의 불투명도다 (0~1). 색과 분리한 이유: 그림자는 반투명이 기본이므로
	// 색 값의 `alpha()` 변형에 기대지 않고 표가 그 값을 직접 갖는다.
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
	// Name은 표준 토큰 어휘의 이름이다: xs, sm, md, lg, xl, 2xl.
	Name string
	// Px는 논리 스테이지 px다 — pt·EMU 환산은 타깃 경계에서만 일어난다.
	Px float64
}

// ColorToken은 색 토큰 하나다.
//
// v1의 **모든** 색 토큰은 clrScheme 슬롯을 갖는다. 슬롯 없는 토큰을 만들면 그 색만
// PowerPoint의 테마 색 편집을 따라가지 못해, 사용자가 색을 바꿨을 때 덱 안에서 색 하나가
// 어긋난 채 남는다 — 그 상태는 파일을 열어 보기 전까지 보이지 않는다.
type ColorToken struct {
	// Name은 표준 토큰 어휘의 이름이다: fg, bg, muted, accent1 ….
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

// ColorSlots는 OOXML clrScheme의 **열두 자리**다 — 스키마가 요구하는 순서 그대로.
//
// 이 목록이 색 토큰 표의 **상한**이다: 모든 색 토큰이 슬롯을 가져야 하고 슬롯은
// 겹칠 수 없으므로 한 테마의 색 이름은 최대 열둘이다. 그래서 부모를 물려받은 테마는 색을
// 더하는 대신 **갈아 끼운다** — 반경·공간·그림자·폰트에는 이 상한이 없다 (SpaceToken의
// 주석이 그 비대칭의 근거를 적는다).
//
// 값 층(테마 로더)과 PPTX 렌더러가 **같은 목록**을 읽는다: 두 곳에 적으면 "로더는 받는데
// 렌더러는 모르는 슬롯"이 생기고, 그 어긋남은 파일을 조립할 때까지 보이지 않는다.
func ColorSlots() []string {
	return []string{
		"dk1", "lt1", "dk2", "lt2",
		"accent1", "accent2", "accent3", "accent4", "accent5", "accent6",
		"hlink", "folHlink",
	}
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
