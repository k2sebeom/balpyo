package soml

// 이 파일은 **확정된 도형 외관**이다 — 카탈로그(shape.go)가 이름과 값 어휘를 소유하고
// 여기가 값의 모양을 소유한다 (layout.go ↔ placement.go와 같은 나눔).
//
// Appearance·Placement와 같은 계약을 진다: **렌더러는 이 구조체만 읽고 Attrs를 다시
// 해석하지 않는다.** 값 어휘(색 토큰·리터럴·`none`, border 축약, radius·shadow 토큰,
// 0~1 실수, 각도, 닫힌 어휘)는 core가 한 번만 적용한다. 도형 외관이 값 층을 두 번 지나면
// 웹의 CSS와 PPTX의 OOXML이 갈라지고, 그 종류의 버그는 두 산출물을 나란히 볼 때까지
// 드러나지 않는다.

// ShapeStyle은 확정된 도형 외관이다.
//
// 필드 순서는 속성 목록(shapeOwnAttrs·shapeAppearanceAttrs) 순서다. "정하지 않음"은
// 포인터의 nil과 닫힌 어휘의 빈 문자열로 나타낸다 — `opacity="1"`은 저자가 정한 1이고
// "opacity를 말하지 않음"과 다르다 (ContainerLayout·textPatch가 포인터를 쓰는 것과 같은
// 근거). 인라인 속성이 없는 도형은 값 필드가 비어 남고, 캐스케이드(컴포넌트 기본값·variant)가 그 빈 자리를 채운다.
type ShapeStyle struct {
	// Path는 `type="path"`의 경로다 — 절대 좌표 명령으로 정규화한 목록이다 (H · V는 L이 된다). 다른 종류에서는 nil이다.
	Path []PathCmd
	// Type은 프리셋 이름이다 (ShapeTypes의 12종). 생략할 수 없으므로 포인터가 아니다 —
	// 도형의 종류는 추측할 수 없고, 없으면 값 층이 에러를 낸다 (core).
	Type string
	// ID는 저자가 붙인 도형 이름이다 (`<arrow from="db">`의 참조 대상).
	//
	// 식별자는 외관이 아니다 (Appearance.Link가 목적지를 TextStyle 밖에 둔 것과 같은
	// 구별). 그래도 여기 담는 이유는 "렌더러가 Attrs를 다시 읽지 않는다"는 계약이다:
	// PPTX 도형 이름·웹 식별자에 저자의 id를 실을지는 렌더 매핑의 결정이고, 그 값이
	// 모델에 없으면 렌더러가 Attrs를 뒤지는 길밖에 남지 않는다.
	ID string
	// Fill은 채움이다 — 색(토큰·리터럴) 또는 `none`.
	Fill *Fill
	// Border는 테두리다 — 축약 3조각(굵기·스타일·색) 또는 `none`.
	Border *Border
	// Radius는 모서리 반경이다 — `type="rounded-rect"`에서만 뜻이 있다 (그 검증은 값 층).
	Radius *Radius
	// Shadow는 그림자 토큰이다. 토큰 표의 항목을 **그대로** 담는다 (ShadowToken의 주석).
	Shadow *ShadowToken
	// Opacity는 0~1 실수다 (`0.8`). 웹 `opacity`와 OOXML의 알파가 둘 다 이
	// 범위를 직접 받는다.
	Opacity *float64
	// Rotate는 시계방향 회전 각도다 (도, 음수 허용). 웹 `transform: rotate()`와 OOXML
	// `xfrm@rot`(60000분의 1도)이 같은 방향 규약을 쓴다.
	Rotate *float64
	// Flip은 대칭이다 (닫힌 어휘 — Flips).
	Flip Flip
	// Alt은 도형 **그림**의 대체 텍스트다 (웹은 잉크 `<svg>`의 `role="img" aria-label`,
	// PPTX는 `p:cNvPr@descr`). 빈 값은 "이름 없음"이고 그때 산출물은 이 속성이 없던 때와
	// 한 바이트도 다르지 않다 — 그림의 기본은 장식이다.
	//
	// 외관이 아니라 저자의 내용이므로 캐스케이드를 타지 않는다 (ImageStyle.Alt와 같은 자리).
	Alt string
	// Box는 도형 자신의 박스다 — 안쪽 여백·크기·비율. 레이아웃 컨테이너와 **같은
	// 구조체**를 공유한다 (Box의 주석).
	Box
}

// Fill은 확정된 채움이다.
//
// 색과 `none`을 한 구조체에 담는 이유: 둘은 같은 속성의 두 값이고, "채움 없음"을 Color의
// 제로값으로 표현하면 "해석되지 않은 색"과 구별할 수 없다 (Color.IsZero는 해석 후에는 있을
// 수 없다는 계약이다). 렌더러는 None을 먼저 보고 갈라진다 — 웹은 선언을 내지 않거나
// `transparent`, PPTX는 `a:noFill`이다.
type Fill struct {
	None  bool
	Color Color
	// Pattern은 무늬 채움이다 (`pattern(…)`). nil이 아니면 Color는 쓰지 않는다 — 무늬는 두 색(앞 · 뒤)을 갖는
	// 채움이라 한 색 자리에 담을 수 없다. 도형 채움에만 생긴다 (값 층이 다른 자리를 막는다).
	Pattern *Pattern
}

// Pattern은 확정된 무늬 채움이다 — 프리셋 무늬 하나와 앞색(선 · 점) · 뒷색(바탕)이다. 두 색은 단색이고
// 불투명도(alpha 변형)를 가질 수 있다.
type Pattern struct {
	Kind PatternKind
	Fg   Color
	Bg   Color
}

// PatternKind는 무늬 이름이다 — 닫힌 어휘 (PatternKinds).
type PatternKind string

const (
	PatternHorizontal       PatternKind = "horizontal"
	PatternVertical         PatternKind = "vertical"
	PatternDiagonalUp       PatternKind = "diagonal-up"
	PatternDiagonalDown     PatternKind = "diagonal-down"
	PatternWideDiagonalUp   PatternKind = "wide-diagonal-up"
	PatternWideDiagonalDown PatternKind = "wide-diagonal-down"
	PatternCross            PatternKind = "cross"
	PatternDiagonalCross    PatternKind = "diagonal-cross"
	PatternDots             PatternKind = "dots"
)

// PatternKinds는 무늬 어휘 전체다 (순서 = 진단 메시지 순서).
func PatternKinds() []PatternKind {
	return []PatternKind{PatternHorizontal, PatternVertical, PatternDiagonalUp, PatternDiagonalDown,
		PatternWideDiagonalUp, PatternWideDiagonalDown, PatternCross, PatternDiagonalCross, PatternDots}
}

// ColorFill은 색 채움이다.
func ColorFill(c Color) Fill { return Fill{Color: c} }

// NoFill은 `fill="none"`이다.
func NoFill() Fill { return Fill{None: true} }

// Border는 확정된 테두리다 — `<굵기> <스타일> <색>` 축약이 여기까지 풀린다 (`"1 solid muted"`).
//
// 축약을 문자열로 흘려보내지 않는 이유는 Pad와 같다: 렌더러가 축약을 다시 해석하면 규칙이
// 두 곳에 생기고, 웹의 `border`와 OOXML `a:ln`은 조각을 서로 다른 자리에 싣는다
// (OOXML은 굵기가 `w` 속성, 스타일이 `prstDash` 자식, 색이 `solidFill` 자식이다).
type Border struct {
	None bool
	// Width는 논리 스테이지 px다. 0·음수는 값 층이 막는다.
	Width float64
	Style BorderStyle
	Color Color
}

// NoBorder는 `border="none"`이다.
func NoBorder() Border { return Border{None: true} }

// Radius는 확정된 모서리 반경이다 — 토큰 참조와 수치 두 상태다.
//
// **Length를 재사용하지 않는 이유**: Length.Token은 공간 토큰 이름이고 웹이 그것을
// `var(--balpyo-space-*)`로 낸다 (placement.go). radius 토큰은 다른 표(Theme.Radii)에서
// 오므로 같은 필드에 담으면 렌더러가 어느 변수 이름을 낼지 값만 보고는 알 수 없다.
type Radius struct {
	// Token은 이 반경이 온 radius 토큰 이름이다 (`radius="lg"` → "lg"). 비어 있으면
	// 저자가 적은 수치다.
	Token string
	// Px는 논리 스테이지 px다 — 토큰 참조에도 해석된 값이 함께 담긴다 (Length가 토큰
	// 이름과 px를 함께 담는 것과 같은 근거: PPTX는 `prstGeom` 조정값으로 굽고, OOXML
	// 테마 파트에는 반경 개념이 없다).
	Px float64
}

// TokenRadius는 토큰 참조 반경이다 — 이름과 해석된 px를 함께 담는다.
func TokenRadius(name string, px float64) Radius { return Radius{Token: name, Px: px} }

// PxRadius는 수치 반경이다.
func PxRadius(px float64) Radius { return Radius{Px: px} }

// IsToken은 이 반경이 토큰 참조인지 보고한다.
func (r Radius) IsToken() bool { return r.Token != "" }

// Box는 박스의 안쪽 여백·크기·비율이다 — **레이아웃 컨테이너와 도형이 공유한다**
// (ContainerLayout.Box, ShapeStyle.Box).
//
// ContainerLayout에서 떼어 낸 이유는 카탈로그를 다시 나눈 것(soml.BoxAttrs)의 자료구조 쪽
// 절반이다: 크기와 안쪽 여백은 컨테이너의 성질이 아니라 박스의 성질이고, 두 요소가 같은
// 표기를 같은 뜻으로 받는다. 구조체가 두 벌이면 값 문법이 두 벌이 되고, 그 둘이 갈라지는
// 것은 배치 하나가 웹에서만 먹는 것으로만 드러난다.
//
// **임베드**로 붙이는 이유: 렌더러가 `container.Pad`·`shape.Width`를 그대로 읽어야 한다.
// 이름 있는 필드로 두면 기존 렌더러 전부가 한 겹 더 들어가야 하고, 그 변경은 배치 CSS의
// 선언 순서를 건드릴 위험을 산출물과 무관한 이유로 만든다.
type Box struct {
	Pad *Pad
	// 크기·비율은 요소 자신의 박스다. 자식 배치(Placement)와 나뉘어 있는 이유는 소유자가
	// 다르기 때문이다 — 여기 값은 요소가 자기 크기를 정한 것이고, Placement는 부모 안에서
	// 자기 자리를 정한 것이다.
	Width     *Length
	Height    *Length
	MinWidth  *Length
	MinHeight *Length
	MaxWidth  *Length
	MaxHeight *Length
	Ratio     *Ratio
}

// ShapeStyle은 이 요소의 확정 도형 외관이다. 도형이 아니거나 해석 전이면 nil이다 —
// TextStyle()·Container()와 같은 이유로 요소에 직접 둔다 (렌더러가 두 단계 nil 검사를
// 쓰지 않게).
func (e *Element) ShapeStyle() *ShapeStyle {
	if e.Appearance == nil {
		return nil
	}
	return e.Appearance.Shape
}

// PathCmd는 경로 명령 하나다 — 0~100 상자의 절대 좌표다. Op는 'M'(점 하나) · 'L'(점 하나) · 'C'(점 셋 — 제어점
// 둘과 끝점) · 'Q'(점 둘 — 제어점과 끝점) · 'Z'(점 없음)이고, Pts는 x, y가 번갈아 놓인다.
type PathCmd struct {
	Op  byte
	Pts []float64
}
