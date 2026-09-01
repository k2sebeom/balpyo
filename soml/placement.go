package soml

// 이 파일은 **확정된 배치 값**이다 — 카탈로그(layout.go)가 이름을 소유하고 여기가 값의
// 모양을 소유한다.
//
// Appearance와 같은 계약을 진다: **렌더러는 이 구조체만 읽고 Attrs를 다시 해석하지
// 않는다.** 값 어휘(공간 토큰, 트랙 목록, `fill`, `%`)는 core가 한 번만 적용한다. 배치가
// 값 층을 두 번 지나면 웹의 CSS와 지오메트리 패스가 서로 다른 값을 볼 자리가 생기고, 그
// 종류의 버그는 두 산출물을 나란히 볼 때까지 드러나지 않는다 (Appearance의 주석과 같은
// 근거 — 외관에서 배치로 넓힌 것이다).
//
// 컨테이너 배치(ContainerLayout)와 자식 배치(Placement)를 **나눈** 이유: 렌더러가 "이
// 요소가 컨테이너인가"를 다시 판단하지 않아야 한다. Container가 non-nil이면 컨테이너이고
// 그 구조체만으로 컨테이너 CSS 전부(display·direction·gap·정렬·트랙·크기)가 나온다.
// 한 구조체에 합치면 `<p grow="1">`도 컨테이너 필드를 갖게 되어 "gap이 있으면 컨테이너"
// 같은 추론이 렌더러에 생기고, 그 추론이 두 렌더러에서 갈라진다.

// ContainerFlow는 컨테이너의 배치 흐름이다 — 태그에서 파생된다 (FlowOf).
//
// 태그 이름 대신 이 값을 담는 이유: 컨테이너 CSS가 Container 구조체 하나에서 전부 나와야
// 렌더러가 요소 이름을 다시 보지 않는다. flex의 방향과 grid는 CSS에서 서로 다른 축
// (`flex-direction` vs `display:grid`)이므로 값 하나로 합쳐 둔다.
type ContainerFlow int

const (
	// FlowRow는 `<row>`다 — `display:flex; flex-direction:row`.
	FlowRow ContainerFlow = iota
	// FlowColumn은 `<col>`이다 — `display:flex; flex-direction:column`.
	FlowColumn
	// FlowGrid는 `<grid>`다 — `display:grid`.
	FlowGrid
	// FlowStack은 `<stack>`이다 — 단일 셀 `display:grid`로 모든 자식이 같은 칸에 겹친다
	// (PRD §13-3). **KindStack이 아니라 흐름으로 갈리는** 것이 Iteration 6의 결정이다:
	// 파서(콘텐츠 모델·자식 어휘)·core(컨테이너 속성 해석)·PPTX(도형 0개 + 자식 재귀)·
	// 지오메트리(재지만 좌표 미소비)가 `<row>`와 전부 같고, 다른 것은 웹의 CSS 한 줄뿐이며
	// 그 한 줄은 이미 종류가 아니라 흐름으로 갈린다. FlowOf의 존재가 곧 확장 지점이다
	// (ADR — 항목 07).
	FlowStack
)

// FlowOf는 레이아웃 컨테이너 태그의 배치 흐름이다. 컨테이너가 아닌 이름에도 FlowRow를
// 돌려주므로 호출자가 KindOf로 컨테이너임을 먼저 확인한다 — 이름 → 흐름 대응을 soml에
// 두는 것은 카탈로그를 한 곳에 둔 이유와 같다 (core가 자기 표를 들면 두 벌이 된다).
func FlowOf(name string) ContainerFlow {
	switch name {
	case ColElement:
		return FlowColumn
	case GridElement:
		return FlowGrid
	case StackElement:
		return FlowStack
	}
	return FlowRow
}

// ContainerLayout은 레이아웃 컨테이너의 확정 배치다 (PRD §7.4 컨테이너 표).
//
// 필드 순서는 PRD 표 순서다. "정하지 않음"은 포인터의 nil과 닫힌 어휘의 빈 문자열로
// 나타낸다 — `gap="0"`은 저자가 정한 0이고 "gap을 말하지 않음"과 다르다 (textPatch가
// 포인터를 쓰는 것과 같은 근거).
type ContainerLayout struct {
	Flow    ContainerFlow
	Gap     *Length
	Align   BoxAlign
	Justify BoxJustify
	// Wrap은 `flex-wrap: wrap`이다. 값 없는 속성(`<row wrap>`)이 참을 뜻하므로 포인터가
	// 필요 없다 — 거짓은 CSS 기본값(nowrap)과 같고, 끄는 표기는 속성을 지우는 것이다.
	Wrap bool
	Cols *Tracks
	Rows *Tracks
	// Box는 컨테이너 자신의 박스다 — 안쪽 여백·크기·비율. `<shape>`와 **같은 구조체**를
	// 공유한다 (Box의 주석 — 카탈로그를 BoxAttrs로 나눈 것의 자료구조 쪽 절반).
	// 임베드이므로 `container.Pad`·`container.Width`는 그대로 읽힌다.
	Box
}

// Placement는 부모 컨테이너 안에서 자기 자리를 정하는 확정 값이다 (PRD §7.4 자식 표).
//
// 요소 종류와 무관하게 **같은 표**가 채운다 — `<text>`·블록·리스트·컨테이너·`<spacer>`가
// 전부 이 구조체를 받는다. 종류마다 다른 구조체를 두면 자식 속성 하나를 추가할 때 어느
// 종류가 빠졌는지 사람이 세어야 한다.
//
// 필드가 전부 비교 가능한 것은 의도다: core가 "배치 속성이 하나도 없었다"를 제로값 비교로
// 판정하고, 그때 요소의 Placement는 nil로 남는다 (렌더러의 골든 불변이 이 nil에 달려 있다).
type Placement struct {
	Grow   *int
	Shrink *int
	Basis  *Length
	// Span은 `grid-column: span N`이다 — `<grid>`의 자식에서만 뜻이 있고, 그 검증은
	// 부모를 아는 core의 순회가 한다.
	Span  *int
	Self  BoxAlign
	Order *int
	// At은 stack 셀 안 9지점 앵커다 (`place-self`) — `<stack>`의 자식에서만 뜻이 있고, 그
	// 검증은 부모를 아는 core의 순회가 한다 (span과 같은 자리). 정하지 않으면 빈 문자열이고,
	// 그때 자식은 컨테이너의 align/justify 기본 정렬을 따른다.
	At StackAnchor
	// X·Y는 부모 축 기준 **절대 오프셋**이다 — `<stack>`의 자식이거나 canvas 슬라이드의
	// 직속 자식에서만 뜻이 있다 (부모를 아는 core의 순회가 검증한다). 정하지 않은 축은
	// nil로 남는다 (`x`만 준 자식은 Y가 nil). 좌표는 크기가 아니므로 음수를 받는다 (박스
	// 밖으로 걸친 배지) — soml.Length가 이미 담을 수 있고, 그 부호 허용은 값 층이 gate한다.
	X *Length
	Y *Length
	// Style은 정규화된 **배치 CSS 선언 목록**이다 (탈출구 — PRD §7.9). `prop: value`를
	// `; `로 이은 문자열이고, 빈 문자열이면 선언이 없다. 문자열인 이유: 웹 렌더러가 인라인
	// style에 그대로 이어 붙이는 것이 이 값의 유일한 소비처이고, 값 문법은 엔진이 보지
	// 않는다는 것이 탈출구의 계약이다 (core/style.go).
	Style string
}

// Length는 확정된 길이 값이다 (PRD §13-5의 단위 계약).
//
// 수치(논리 스테이지 px)·백분율·**공간 토큰 참조** 세 상태다. `fill`은 값 층에서 **해당
// 축의 100%**로 눕는다 (Percent=true, Value=100) — `fill`을 별도 상태로 남기지 않은 이유:
// 두 상태가 같은 CSS를 내야 하는데 상태가 둘이면 렌더러마다 다르게 낼 자리가 생기고,
// `flex-grow`는 이미 `grow` 속성으로 따로 있다 (ADR 감 — 항목 06).
type Length struct {
	// Value는 Percent가 거짓이면 논리 스테이지 px, 참이면 백분율 수치다. 토큰 참조에도
	// 해석된 px가 함께 담긴다 (Token 주석).
	Value float64
	// Percent는 Value가 부모 축에 대한 백분율인지다.
	Percent bool
	// Token은 이 길이가 온 공간 토큰 이름이다 (`gap="xl"` → "xl"). 비어 있으면 저자가 적은
	// 수치·백분율이다.
	//
	// **Color가 토큰과 리터럴 두 상태를 갖는 것과 같은 이유다**: 웹은 토큰을
	// `var(--balpyo-space-xl)` 참조로 내야 테마를 바꿨을 때 여백이 따라 움직인다. 값 층에서
	// px로 눕히면 그 참조를 복원할 방법이 없고, `gap="24"`(저자가 적은 리터럴)과
	// `gap="md"`(테마를 따라야 하는 참조)가 산출물에서 같은 것이 된다 — 렌더러가 px를
	// 토큰 표에서 거꾸로 찾는 길이 위험한 이유가 그것이다.
	//
	// 해석된 px를 Value에 함께 담아 두는 이유: 토큰 참조가 웹에만 사는 개념이기 때문이다.
	// OOXML 테마 파트에는 공간이 없어(SpaceToken 주석) PPTX는 좌표로 굽고, 진단·테스트도
	// 확정 수치를 본다 — 그 소비자들이 토큰 표를 다시 뒤지지 않아야 한다.
	Token string
}

// Px는 논리 px 길이다.
func Px(v float64) Length { return Length{Value: v} }

// Percent는 백분율 길이다.
func Percent(v float64) Length { return Length{Value: v, Percent: true} }

// SpaceLength는 공간 토큰 참조 길이다 — 이름과 해석된 px를 함께 담는다.
func SpaceLength(token string, px float64) Length {
	return Length{Value: px, Token: token}
}

// IsToken은 이 길이가 공간 토큰 참조인지 보고한다.
func (l Length) IsToken() bool { return l.Token != "" }

// Pad는 확정된 패딩 4변이다. 표기는 CSS `padding`의 1~4값 축약과 같은 규칙으로 여기까지
// 풀린다 — 축약 규칙을 새로 발명하지 않는 이유는 저자가 이미 아는 규칙이기 때문이고,
// 풀어서 담는 이유는 렌더러가 축약을 다시 해석하지 않아야 하기 때문이다.
type Pad struct {
	Top    Length
	Right  Length
	Bottom Length
	Left   Length
}

// Tracks는 `grid-template-columns/rows`의 확정 트랙이다.
//
// 두 상태다: Count > 0이면 균등 분할(`cols="4"` → `repeat(4, 1fr)`), 아니면 명시 트랙
// 목록이다. 균등 분할을 트랙 목록으로 눕히지 않은 이유는 산출 CSS의 크기다 —
// `repeat(12, 1fr)`이 `1fr`을 열두 번 적은 것과 같은 뜻이지만 골든이 읽기 어려워진다.
type Tracks struct {
	Count int
	List  []Track
}

// TrackKind는 트랙 하나의 값 형태다.
type TrackKind int

const (
	// TrackAuto는 `auto`다 — 내용에 맞춘 트랙 (Value는 뜻이 없다).
	TrackAuto TrackKind = iota
	// TrackFr은 `Nfr`이다 — 남은 공간의 비율.
	TrackFr
	// TrackPercent는 `N%`다.
	TrackPercent
	// TrackPx는 무단위 수치 = 논리 스테이지 px다.
	TrackPx
)

// Track은 트랙 하나다.
type Track struct {
	Kind  TrackKind
	Value float64
}

// Ratio는 `aspect-ratio`의 확정 비율이다 — `16:9`의 정수 쌍을 그대로 담는다. 하나의
// 부동소수(1.777…)로 눕히지 않는 이유: 저자가 쓴 비율이 산출물에 그대로 나가야 하고,
// 눕히면 `16:9`와 `1.7778`이 같은 값이 되어 골든이 무엇을 말하는지 흐려진다.
type Ratio struct {
	W int
	H int
}

// BoxAlign은 교차축 정렬이다 (`align-items`와 `align-self`가 같은 어휘를 쓴다).
//
// 값은 CSS의 논리 이름 그대로다 (`flex-start`가 아니라 `start`) — CSS는 두 표기를 모두
// 받지만 `start`가 grid와 flex에서 같은 뜻이고, TextAlign이 논리 이름을 고른 것과 같은
// 근거다 (물리 이름은 방향이 바뀌면 저자의 의도를 잃는다).
type BoxAlign string

const (
	BoxAlignStart    BoxAlign = "start"
	BoxAlignCenter   BoxAlign = "center"
	BoxAlignEnd      BoxAlign = "end"
	BoxAlignStretch  BoxAlign = "stretch"
	BoxAlignBaseline BoxAlign = "baseline"
)

// BoxAligns는 교차축 정렬 어휘 전체다 (순서 = 진단 메시지 순서 — 진단이 결정적이어야
// 한다는 이유는 TextAligns와 같다).
func BoxAligns() []BoxAlign {
	return []BoxAlign{BoxAlignStart, BoxAlignCenter, BoxAlignEnd, BoxAlignStretch, BoxAlignBaseline}
}

// BoxJustify는 주축 분배다 (`justify-content`).
//
// `between`·`around`·`evenly`는 CSS의 `space-*` 접두를 뗀 이름이다 (PRD §7.4 표) —
// 접두를 붙이는 번역은 렌더러의 몫이다.
type BoxJustify string

const (
	BoxJustifyStart   BoxJustify = "start"
	BoxJustifyCenter  BoxJustify = "center"
	BoxJustifyEnd     BoxJustify = "end"
	BoxJustifyBetween BoxJustify = "between"
	BoxJustifyAround  BoxJustify = "around"
	BoxJustifyEvenly  BoxJustify = "evenly"
)

// BoxJustifies는 주축 분배 어휘 전체다 (순서 = 진단 메시지 순서).
func BoxJustifies() []BoxJustify {
	return []BoxJustify{BoxJustifyStart, BoxJustifyCenter, BoxJustifyEnd,
		BoxJustifyBetween, BoxJustifyAround, BoxJustifyEvenly}
}

// StackAnchor는 stack 셀 안 자식의 **9지점 앵커**다 (`at` — PRD §7.4·§7.4의 자식 표).
//
// 어휘는 3×3 격자의 **합성 이름**이다 (`top-left`…`bottom-right`). 짧은 이름(`top`·`left`)과
// 저울질했고 합성을 골랐다: 진단 목록이 3×3으로 읽히고, 저자가 표를 외우지 않아도 "세로 축
// 하나 · 가로 축 하나를 잇는다"는 규칙에서 이름을 유도할 수 있다. `center`만 한 단어인 것이
// 유일한 예외이며 `center-center`는 받지 않는다 — 같은 자리를 두 이름으로 받지 않는다는
// 기준(ADR 0027)이다.
//
// BoxAlign 두 개(align/justify 쌍)로 눕히는 길과도 저울질했고 닫는 쪽을 골랐다: 확정 값에서
// 웹이 `place-self`로 내는 것이 정확히 그 일이지만, 어휘를 align×justify로 열면
// `at="stretch-baseline"` 같은 조합이 문법상 가능해진다. 9지점은 **닫힌 목록**이어야 한다.
type StackAnchor string

const (
	StackTopLeft      StackAnchor = "top-left"
	StackTopCenter    StackAnchor = "top-center"
	StackTopRight     StackAnchor = "top-right"
	StackCenterLeft   StackAnchor = "center-left"
	StackCenter       StackAnchor = "center"
	StackCenterRight  StackAnchor = "center-right"
	StackBottomLeft   StackAnchor = "bottom-left"
	StackBottomCenter StackAnchor = "bottom-center"
	StackBottomRight  StackAnchor = "bottom-right"
)

// StackAnchors는 9지점 앵커 어휘 전체다 — **순서가 곧 3×3 격자 순서이자 진단·문서 순서다**
// (위 행부터, 각 행은 왼쪽부터). 어휘 정의와 진단 목록이 갈라지지 않도록 core가 이 목록을
// 소비한다 (BoxAligns와 같은 태도).
func StackAnchors() []StackAnchor {
	return []StackAnchor{
		StackTopLeft, StackTopCenter, StackTopRight,
		StackCenterLeft, StackCenter, StackCenterRight,
		StackBottomLeft, StackBottomCenter, StackBottomRight,
	}
}

// Container는 이 요소의 확정 컨테이너 배치다. 컨테이너가 아니거나 해석 전이면 nil이다 —
// TextStyle()과 같은 이유로 요소에 직접 둔다 (렌더러가 두 단계 nil 검사를 쓰지 않게).
func (e *Element) Container() *ContainerLayout {
	if e.Appearance == nil {
		return nil
	}
	return e.Appearance.Container
}

// Placement는 이 요소의 확정 자식 배치다. 배치 속성이 없거나 해석 전이면 nil이다.
func (e *Element) Placement() *Placement {
	if e.Appearance == nil {
		return nil
	}
	return e.Appearance.Placement
}
