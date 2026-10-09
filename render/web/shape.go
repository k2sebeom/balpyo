package web

import (
	"fmt"
	"math"
	"strings"

	"github.com/k2sebeom/balpyo/soml"
)

// 이 파일은 **`<shape>` → 요소 + SVG 기하 + 외관 CSS**다.
//
// # 도형 하나는 박스 하나와 그림 하나다
//
//	<div class="balpyo-shape" …>      ← 잰 박스. 크기·회전이 여기 걸린다 (pad·앵커는 본문 쪽).
//	  <svg class="balpyo-shape-ink">  ← 그림. 채움·테두리·그림자·불투명도·대칭이 여기 걸린다.
//	  <p class="balpyo-text" …>       ← 도형 안 텍스트 (문서 순서대로 쌓인다).
//
// 프리셋의 **텍스트 영역**이 박스보다 작은 여덟 종은 텍스트가 래퍼 하나를 더 쓴다:
//
//	<div class="balpyo-shape" …>
//	  <svg class="balpyo-shape-ink">
//	  <div class="balpyo-shape-body" style="left: 25%; top: 25%; …">  ← 프리셋의 텍스트 영역
//	    <p class="balpyo-text" …>
//
// 박스와 그림을 나눈 이유는 **잰 rect가 도형의 겉 박스여야** 하기 때문이다: 그 rect가 곧
// PPTX `xfrm`이므로 그림이 레이아웃에 영향을 주면 안 된다 — svg는 절대 배치로
// 흐름에서 빠지고, 박스 크기는 저자가 준 `w`/`h`와 부모의 배치가 정한다.
//
// # 왜 SVG 한 벌인가 (12종 전부)
//
// 세 종(rect·rounded-rect·ellipse)은 CSS로도 그려지지만 나머지 아홉은 아니다. 갈래를 고른
// 근거는 **외관 속성이 두 벌로 구현되는 것을 막는 것**이다:
//
//   - `clip-path: polygon(…)`는 테두리와 그림자를 **함께 자른다**. 컴포넌트 기본값이
//     `border: 1 solid muted`이므로(테마의 shape.default) 테두리는 예외가 아니라 기본이고, 잘린
//     테두리는 "도형 종류에 따라 테두리 굵기가 다르다"로만 드러난다.
//   - 셋만 CSS로 두면 그림자를 `box-shadow`와 SVG 두 벌로 구현하게 되고, 두 벌이 갈라지는
//     것은 도형 종류에 따라 외관이 다른 것으로만 보인다 — 이 저장소가 가장 비싸게 치는 실패다.
//   - SVG는 어차피 들어온다: `border=… gradient(…)`는 엔진이 SVG로 그린다 (CSS
//     `border-image`가 radius와 조합되지 않는다). 그때 채움·선의 자리가 이미 SVG면 그 항목은
//     `linearGradient` 하나를 더하는 일이 된다.
//
// 그래서 12종 전부가 `<svg>` 하나이고, 외관 선언은 **그 svg 하나에** 걸린다. 기하만 두 갈래다
// (아래 shapeSpace) — 그리는 요소가 `<rect>`인지 `<path>`인지의 차이이고, 외관 매핑은 같다.
//
// # 기하의 출처는 OOXML 프리셋 정의다
//
// 12종은 ECMA-376의 `presetShapeDefinitions.xml`(`prstGeom` 프리셋)에서 **기본 조정값으로
// 평가해** 유도했다 — 유도 근거는 각 표 행의 주석에 `gd` 이름과 함께 적어 두었다. 눈대중으로
// 그린 SVG는 LibreOffice·PowerPoint의 `prstGeom`과 다른 모양이 되고, 그 차이를 오차 조정으로
// 덮으면 픽셀 비교가 무력해진다 (몇 %가 아니라 수십 %로 나타난다).
//
// **정규 공간을 늘린다는 결정**: 프리셋 공식의 `ss`는 `min(w, h)`인데, 웹은 도형의 픽셀
// 크기를 알지 못한다 (브라우저가 배치를 계산하고 우리는 그 결과를 잰다 — 그것이 이 엔진의
// 구조다). CSS/SVG에는 자기 박스의 `min(w,h)`를 경로 데이터에 넣을 수단이 없으므로
// (`clip-path`의 `cqmin`은 테두리를 자르는 길로만 갈 수 있다), 프리셋을 **정사각형에서 평가한
// 정규 100×100 공간**에 두고 `preserveAspectRatio="none"`으로 박스에 늘린다. 그 결과
// `ss` 기반 조정값이 걸린 다섯 프리셋(rounded-rect·cylinder·chevron·arrow-right·hexagon·
// parallelogram)은 PowerPoint의 기본 모양과 비율이 다르고, **PPTX 쪽이 측정된 박스로 계산한
// `avLst` 조정값을 내어 같은 그림을 맞춘다** (각 표 행의 Adjust 명세 — ShapeAdjusts). 웹이 기준
// 타깃이므로 맞추는 쪽은 PPTX이고, 조정값을 실은 프리셋도 PowerPoint에서 여전히
// 편집 가능한 진짜 프리셋 도형이다 (`radius` → `prstGeom` 조정값이 이미 같은 수단이다).
//
// # 엔진 CSS와 인라인 style의 경계 (layout.go와 같은 기준)
//
//	엔진 CSS (shapeCSS, 태그마다 고정):
//	  .balpyo-shape:     position, display, flex-direction   — "박스 + 세로로 쌓이는 텍스트"의 정의
//	  .balpyo-shape-ink: position, left/top/width/height, overflow  — 그림이 박스를 덮는다는 정의
//	  .balpyo-shape-body: position, display, flex-direction  — 텍스트 영역이 상자라는 정의
//	  자식의 position: relative                              — 그림 위에 글자가 온다는 정의
//	  회전 선언(transform)과 측정 뷰의 무효화                 — 아래 회전 절
//
//	인라인 style (요소마다 다른 확정 값):
//	  width/height/min-*/max-*, aspect-ratio, --balpyo-shape-rotate  — 박스 쪽
//	  padding, justify-content(anchor)                       — 텍스트가 사는 상자 쪽
//	  left/top/right/bottom (프리셋 텍스트 영역의 백분율 인셋)  — 본문 래퍼 쪽
//	  fill, stroke, stroke-width, stroke-dasharray, filter(shadow), opacity,
//	  transform(flip)                                        — 그림 쪽
//	  자식 배치 선언과 저자의 `style` 탈출구 (맨 끝)
//
// `box-sizing: border-box`는 baseCSS의 `*` 규칙이 이미 준다 — 그래서 `pad`는 잰 박스를
// 키우지 않고 **안쪽으로** 들어간다 (PPTX `bodyPr` 인셋과 같은 뜻).
//
// # 테두리와 잰 박스
//
// SVG 선은 경로 **중앙**에 그려진다 — OOXML `a:ln`과 같은 규약이다. 그래서 굵기 3px이면
// 1.5px이 잰 박스 밖으로 번지고, 웹과 PPTX가 **같은 자리에** 번진다: 두 타깃 사이에 보정할
// 수치가 없다. CSS `border`(박스 안쪽)를 쓰지 않은 이유가 이것이고, 그래서 svg는
// `overflow: visible`이어야 한다 (`callout`의 꼬리도 같은 이유로 박스 밖으로 나간다).
//
// # 측정과 회전
//
// `transform: rotate()`가 걸린 요소의 `getBoundingClientRect`는 **회전된 외곽 bbox**를
// 돌려준다 (45도에서 1.41배). 반면 OOXML `xfrm@rot`은 회전 **전** `ext`를 중심 회전한 것이라,
// 그대로 재면 좌표 대조는 통과하면서(측정값과 산출물이 같으므로) 산출물만 틀린다. 그래서
// 측정 패스는 문서에 측정 뷰를 켜고(web.MeasureView) 그 뷰에서 회전 선언을 무효화한다 —
// 발표자 뷰를 `<html>` 속성 하나로 켜는 것과 같은 수단이다.
//
// 그 무효화가 성립하려면 회전 **선언**이 엔진 CSS에 있어야 한다: 인라인 style은 층 밖에서
// 이기므로 스타일시트가 `transform: none`으로 덮을 수 없다 (`!important`를 쓰지 않는 이유다).
// 그래서 인라인은 각도만 커스텀 프로퍼티로 싣고, 선언은 shapeCSS가 갖는다.
//
// # 도형 안 텍스트 — 웹과 PPTX의 구조가 갈라지는 첫 자리
//
// 웹은 도형 박스 안에 **별도 텍스트 박스**가 중첩되고(자식마다 자기 rect를 갖는다), PPTX는
// 같은 도형의 `txBody` 문단이다. 그래서 도형 안 텍스트의 잰 rect는 **PPTX가
// 소비하지 않는다** — `<li>`와 같은 처지다 (재지만 도형 좌표가 아니다).

// shapeSpace는 기하가 사는 좌표 공간이다 (파일 머리 주석의 기하 절).
type shapeSpace int

const (
	// spacePixel은 스케일 없는 px 공간이다 (viewBox 없음). 백분율 프리미티브로 프리셋을
	// **정확히** 표현할 수 있는 셋이 여기 산다 — 늘림이 없으므로 `rounded-rect`의 반경이
	// 원형으로 유지된다 (OOXML roundRect의 모서리는 반지름 x1의 원호다).
	spacePixel shapeSpace = iota
	// spaceNormalized는 정규 100×100 경로 공간이다 — `preserveAspectRatio="none"`으로 박스에
	// 늘린다. 선 굵기는 `vector-effect="non-scaling-stroke"`로 균일하게 남는다 (늘림이 선까지
	// 늘리면 1px 테두리가 축마다 다른 굵기가 된다 — OOXML `a:ln`은 균일하다).
	spaceNormalized
)

// 도형 요소의 엔진 CSS 클래스다.
const (
	shapeClass = "balpyo-shape"
	// shapeInkClass는 기하를 그리는 svg다 (박스를 덮는 그림).
	shapeInkClass = "balpyo-shape-ink"
	// shapeRotateClass는 회전 선언을 받는 표식이다. 회전이 있는 도형에만 붙는다 — 모든
	// 도형에 `transform: rotate(0deg)`를 걸면 회전 없는 도형까지 스택 문맥과 래스터화 경로가
	// 바뀐다 (그 변화는 픽셀 비교에서만 드러난다).
	shapeRotateClass = "balpyo-shape-rotate"
	// shapeBodyClass는 프리셋의 텍스트 영역을 차지하는 래퍼다 — 백분율 인셋이
	// 있는 프리셋에만 나온다. 0인 프리셋(rect·rounded-rect·chevron·callout)에서 구조를 그대로
	// 두는 이유는 골든의 뜻이다: 결과가 같은데 래퍼를 넣으면 가장 많이 쓰이는 프리셋의 픽셀이
	// 서브픽셀 반올림으로 흔들리고, 그 변화는 텍스트 영역과 무관하다.
	shapeBodyClass = "balpyo-shape-body"
)

// shapeRotateVar는 회전 각도를 싣는 커스텀 프로퍼티다 (선언은 엔진 CSS, 값은 인라인 —
// 파일 머리 주석의 회전 절).
const shapeRotateVar = "--balpyo-shape-rotate"

// shadowVarPrefix는 그림자 토큰의 CSS 커스텀 프로퍼티 이름 규칙이다 (색·폰트·공간과 같은
// 규칙 — style.go의 colorVarPrefix 주석).
//
// 그림자를 변수로 내고 **반경(radius)은 내지 않는** 이유는 참조하는 자리의 성질이다:
// 그림자는 CSS 프로퍼티 값(`filter`)이라 `var()`가 통하지만, 반경은 SVG 기하 속성(`rx`)이라
// 커스텀 프로퍼티를 이식성 있게 참조할 수 없다 (SVG 2의 기하 프로퍼티 지원이 엔진마다
// 다르다). 반경을 확정 수치로 내도 웹만 뒤처지지 않는 이유는 OOXML에도 **반경 테마 개념이
// 없기 때문**이다 (`prstGeom/avLst`의 조정값은 도형마다 구워진다 — soml.Radius의 주석):
// 두 타깃이 같은 방식으로 값을 굽는다.
const shadowVarPrefix = "--balpyo-shadow-"

// shapePath는 도형 기하의 경로 하나다 (정규 100×100 공간).
type shapePath struct {
	// D는 SVG 경로 데이터다.
	D string
	// Overlay면 채움 없이 선만 그린다 — OOXML 프리셋의 `fill="none"` 경로에 대응한다
	// (cylinder의 뚜껑선, cloud의 안쪽 선). 테두리가 `none`인 도형에서는 아무것도 그리지
	// 않으므로 그 프리셋의 세부는 사라진다 (알려진 갈라짐).
	Overlay bool
}

// shapeGeometry는 프리셋 하나의 기하다 — **12종의 유도표는 이 목록 하나뿐이다**.
type shapeGeometry struct {
	// Type은 SOML의 `type` 값이다 (soml.ShapeTypes의 이름).
	Type string
	// Preset은 OOXML `prstGeom@prst` 이름이다 — 이 행의 수치가 어느 프리셋에서 왔는지의
	// 출처이자 PPTX 매핑의 열쇠다 (ShapePreset).
	Preset string
	// Space는 이 기하가 사는 좌표 공간이다.
	Space shapeSpace
	// Element·Attrs는 px 공간의 SVG 프리미티브다 (Space == spacePixel).
	Element string
	Attrs   string
	// Rounded면 `radius`가 이 프리셋의 기하에 실린다 (`rx`) — rounded-rect 하나다.
	Rounded bool
	// Paths는 정규 공간의 경로다 (Space == spaceNormalized).
	Paths []shapePath
	// Adjust는 **늘린 그림을 PPTX에서 재현하는 조정값 명세**다 (비어 있으면 프리셋 기본값
	// 그대로 낸다 — 그 프리셋에는 `ss` 의존이 없다는 뜻이다). 여기 적어 두는 이유는 유도와
	// 보정이 한곳에 있어야 나중에 한쪽만 바뀌지 않기 때문이다 — 평가는 PPTX 렌더러가 잰
	// 박스로 한다 (ShapeAdjusts → render/pptx.AdjustValue).
	Adjust []ShapeAdjust
	// TextRect는 프리셋의 **텍스트 영역**이다 — 비어 있으면 박스 전체다.
	TextRect ShapeInsets
}

// ShapeAdjustAxis는 조정값 비율의 분자다 — 늘린 정규 공간을 되돌리는 값의 출처.
type ShapeAdjustAxis string

const (
	// AdjustAxisWidth는 도형 폭이다.
	AdjustAxisWidth ShapeAdjustAxis = "w"
	// AdjustAxisHeight는 도형 높이다.
	AdjustAxisHeight ShapeAdjustAxis = "h"
	// AdjustAxisRadius는 확정 모서리 반경이다 (rounded-rect 하나). 축이 아니라 값이지만
	// 공식의 자리가 같다 — `ss`로 나눈 비율이 조정값이 되는 구조가 셋 다 같다.
	AdjustAxisRadius ShapeAdjustAxis = "radius"
	// AdjustAxisFixed는 `ss`와 무관한 고정값이다 — 조정값은 Base 그대로다. 보정이 필요 없는
	// 조정값도 avLst에 적어야 하는 프리셋이 있어서 쓴다: PowerPoint는 프리셋 조정값 일부만
	// 적힌 avLst를 받지 않고 파일을 복구하며, 그 복구가 도형을 지운다 (명세로는 덮어쓰기지만
	// 그렇게 구현하지 않았다). 그래서 하나라도 적는 프리셋은 전부를 적는다.
	AdjustAxisFixed ShapeAdjustAxis = "fixed"
)

// ShapeAdjust는 `prstGeom/avLst` 조정값 하나의 명세다 (OOXML `a:gd`).
//
// 값은 `val = Base × Axis ÷ ss`다 (`ss = min(w, h)`, 100000분율). 이 형태로 담는 이유는
// **웹이 정규 공간을 늘리기 때문**이다 (파일 머리 주석): 프리셋 공식의 `ss` 의존을 웹이
// 표현할 수 없으므로 PPTX가 잰 박스로 조정값을 계산해 같은 그림을 맞춘다. 문자열 메모로
// 두면 PPTX 쪽에 공식이 손으로 한 번 더 적히고, 그 두 벌이 갈라지는 것은 웹과 PPTX가 같은
// 프리셋을 **다른 비율로** 그리는 것으로만 드러난다.
//
// 평가는 이 패키지가 하지 않는다 — 웹은 잰 박스를 모른다 (브라우저가 배치를 계산하고
// 렌더러는 그 결과를 소비하지 않는다). 명세의 소유자와 평가의 소유자가 갈라지는 것은
// ListTextIndent(웹이 소유하는 상수를 PPTX가 EMU로 환산한다)와 같은 구조다.
type ShapeAdjust struct {
	// Name은 `a:gd@name`이다 (`adj`·`adj1`·`adj2` — 프리셋이 정한 이름).
	Name string
	// Base는 정사각형(w = h = ss)에서의 조정값이다 — 곧 프리셋의 기본 조정값이거나,
	// 웹이 정규 공간에서 평가한 값의 100000분율이다.
	Base float64
	// Axis는 Base에 곱할 비율의 분자다.
	Axis ShapeAdjustAxis
	// Max는 조정값의 상한이다 (0이면 상한 없음 — 그 프리셋의 `maxAdj`가 같은 축의 비율로
	// 커져 이 공식이 구조적으로 그 안에 있다는 뜻이다).
	Max float64
	// Exceed면 값을 공식보다 **한 단위 넘겨** 낸다 (올림 + 1, 100000분의 1).
	//
	// 프리셋 공식의 조건 분기(`?: x y z` — x > 0이면 y, 아니면 z)가 공식의 값에서 정확히 0이 되는
	// 자리에 쓴다. 0은 명세로는 "아니면" 쪽이지만 엔진마다 경계를 다르게 읽고(LibreOffice는
	// 0을 양수 쪽으로 읽는다), 반올림이 값을 조금 내리면 모든 엔진이 반대쪽으로 간다. 한 단위
	// 넘기면 x < 0이 되어 어느 엔진이든 같은 쪽을 읽는다 — 실루엣은 0.01px도 움직이지 않는다.
	Exceed bool
}

// ShapeInsets는 프리셋의 텍스트 영역이다 — 도형 박스에서 **글자가 쓸 수 있는 사각형**
// (OOXML 프리셋 정의의 `<rect>`).
//
// 왜 필요한가: PowerPoint·LibreOffice는 프리셋마다 이 사각형을 갖고 있고 우리가 끄거나 바꿀
// 수단이 없다. 웹이 박스 전체에 글을 흘리면 같은 SOML에서 줄바꿈 위치와 줄 수가 갈라진다
// (`diamond`는 박스의 절반이 텍스트 영역이다). 그래서 **웹이 이 표를 읽어 맞추고 PPTX는
// 아무것도 하지 않는다** — 두 번 들어가는 것을 막는 유일한 길이고, 대신 프리셋 12종의 잉크를
// 두 타깃에서 견주는 픽셀 테스트가 그 짝을 지킨다 (testkit).
//
// 백분율인 이유는 조정값 보정(ShapeAdjust) 덕분에 실루엣이 어느 종횡비에서도 박스를 같은
// 비율로 나누기 때문이다 — 실측이 400×400·560×320·320×560에서 ±0.3% 안으로 일치한다.
type ShapeInsets struct {
	// L·T·R·B는 도형 박스에 대한 **백분율** 인셋이다 (0이면 그 변이 곧 박스 변이다).
	L, T, R, B float64
	// Radius는 `radius` px에 곱해 네 변에 더하는 비율이다 (rounded-rect 하나).
	//
	// 백분율과 갈라지는 이유는 `radius`가 px이기 때문이고(반경은 늘림 없는 px
	// 공간에 산다), 그 덕분에 rounded-rect는 인셋을 **패딩**으로 받을 수 있다: 내용이 높이를
	// 정하는 카드가 그대로 산다. 백분율 인셋은 자기 박스 높이를 참조해야 하므로 그럴 수 없다.
	Radius float64
}

// HasPercent는 백분율 인셋이 하나라도 있는지다 — 웹이 절대 배치 래퍼를 쓰는 조건이고
// (백분율은 패딩으로 표현할 수 없다: CSS padding 백분율은 네 변 모두 **부모 폭** 기준이다),
// 값 층이 도형에 확정 크기를 요구하는 조건이다.
func (in ShapeInsets) HasPercent() bool {
	return in.L != 0 || in.T != 0 || in.R != 0 || in.B != 0
}

// Px는 잰 박스와 반경으로 네 변의 px 인셋을 구한다 (왼·위·오른·아래 순서).
//
// 여기 두는 이유는 백분율의 **기준**(도형 박스 = 테두리 상자)이 한 곳에만 적혀야 하기
// 때문이다 — CSS는 절대 배치로, 테스트는 잰 rect로 같은 사각형을 구하고, 두 식이 갈라지면
// 테스트가 자기 구현을 확인하게 된다.
func (in ShapeInsets) Px(w, h, radius float64) (left, top, right, bottom float64) {
	edge := in.Radius * radius
	return in.L*w/100 + edge, in.T*h/100 + edge, in.R*w/100 + edge, in.B*h/100 + edge
}

// ShapeTextRect는 프리셋의 텍스트 영역이다 — ok=false는 모르는 `type`이다 (ShapeAdjusts와
// 같은 계약: 빈 인셋(= 박스 전체)과 구별되어야 소비자가 "어휘에 있는데 표에 없는 프리셋"에서
// 멈출 수 있다).
func ShapeTextRect(shapeType string) (ShapeInsets, bool) {
	geom, err := shapeGeometryOf(shapeType)
	if err != nil {
		return ShapeInsets{}, false
	}
	return geom.TextRect, true
}

// shapeGeometries는 12종의 기하 유도표다 (순서 = soml.ShapeTypes의 순서).
//
// 각 행의 수치는 ECMA-376 `presetShapeDefinitions.xml`의 해당 프리셋을 **기본 조정값으로,
// 정사각형(w = h = ss = 100)에서** 평가한 것이다. 주석에 어느 `gd`에서 왔는지 적었다 —
// 그 줄이 없으면 다음 사람이 수치를 눈대중으로 고치게 된다.
//
// TextRect(텍스트 영역)는 같은 프리셋 정의의 `<rect>`에서 오고, 각 행 주석의 **실측**은
// LibreOffice 렌더의 글자 위치에서 잰 값이다 (같은 도형 둘을 나란히 두고 한쪽은 왼·위,
// 다른 쪽은 오른·아래로 글자 하나를 밀어 네 변을 각각 잰다 — 400×400 도형에서 px).
// 유도와 실측을 함께 적는 이유는 이 열이 **우리가 그린 실루엣**과 프리셋 정의 양쪽에 맞아야
// 하기 때문이다: 한쪽만 맞으면 두 타깃의 글자가 갈라진다.
func shapeGeometries() []shapeGeometry {
	return []shapeGeometry{
		{
			// rect: 경로가 곧 박스다 (l,t → r,t → r,b → l,b).
			// 텍스트 영역: 프리셋의 rect가 박스 전체다 (실측 네 변 0).
			Type: "rect", Preset: "rect", Space: spacePixel,
			Element: "rect", Attrs: `width="100%" height="100%"`,
		},
		{
			// roundRect: `adj=16667`이 `x1 = ss·adj/100000`(모서리 반지름)이고 모서리는
			// 반지름 x1의 원호 넷이다. 반경은 저자·테마가 준 **px**이므로 정규 공간에 두면
			// 늘어나 타원 모서리가 된다 — 그래서 px 공간의 `<rect rx>`다.
			// Adjust: PPTX는 adj = 100000·radius/ss로 되돌린다. 상한 50000은 프리셋이
			// adj를 `pin 0 adj 50000`으로 자르는 지점이고(짧은 변의 절반 = 완전한 반원),
			// 브라우저가 `rx`를 박스 절반으로 자르는 것과 같은 규칙이다.
			Type: "rounded-rect", Preset: "roundRect", Space: spacePixel,
			// 텍스트 영역: 모서리 원호에 내접하는 사각형이라 네 변이 `r − r/√2 = 0.29289r` **px**이다
			// (백분율이 아니다 — 반경이 px이므로). 실측 radius=12에서 3~4px(예측 3.51), radius=32에서
			// 9~10px(예측 9.37). px이라 패딩으로 받을 수 있고, 그래서 내용이 높이를 정하는 카드가
			// 그대로 산다 — 백분율 인셋 프리셋은 자기 박스 높이를 참조해야 해서 그럴 수 없다.
			Element: "rect", Attrs: `width="100%" height="100%"`, Rounded: true,
			TextRect: ShapeInsets{Radius: 1 - math.Sqrt2/2},
			Adjust:   []ShapeAdjust{RoundRectAdjust()},
		},
		{
			// ellipse: 프리셋은 wR = wd2, hR = hd2의 원호 넷 — 박스에 내접하는 타원이다.
			// 텍스트 영역: 그 타원에 내접하는 사각형이라 네 변이 `(1 − √2/2)/2` = 14.6447%다
			// (실측 58~59px / 400 = 14.5~14.8%).
			Type: "ellipse", Preset: "ellipse", Space: spacePixel,
			Element: "ellipse", Attrs: `cx="50%" cy="50%" rx="50%" ry="50%"`,
			TextRect: ShapeInsets{L: 14.6447, T: 14.6447, R: 14.6447, B: 14.6447},
		},
		{
			// diamond: (l,vc) → (hc,t) → (r,vc) → (hc,b). 조정값이 없다 (순수 비율).
			// 텍스트 영역: 변의 중점 넷을 이은 내접 사각형 = 안쪽 절반 (실측 100~101px = 25.0%).
			Type: "diamond", Preset: "diamond", Space: spaceNormalized,
			Paths:    []shapePath{{D: "M0,50 L50,0 L100,50 L50,100 Z"}},
			TextRect: ShapeInsets{L: 25, T: 25, R: 25, B: 25},
		},
		{
			// can: `adj=25000` → `y1 = ss·adj/200000` = 12.5가 뚜껑 타원의 세로 반지름이고
			// `y3 = b - y1`이다. 첫 경로는 실루엣(위 반타원 → 오른쪽 변 → 아래 반타원),
			// 둘째는 뚜껑 타원의 선이다 (프리셋의 `fill="none"` 경로).
			// 프리셋의 뚜껑에는 `fill="lighten"`이 걸려 있는데 CSS에 대응이 없다 —
			// 웹의 뚜껑은 같은 채움에 선만 얹힌다.
			// Adjust: 웹이 h 기준으로 늘렸으므로 PPTX는 adj = 25000·h/ss (프리셋의
			// maxAdj = 50000·h/ss라 이 값은 언제나 그 절반이다 — 상한이 없다).
			Type: "cylinder", Preset: "can", Space: spaceNormalized,
			Paths: []shapePath{
				{D: "M0,12.5 A50,12.5 0 0 1 100,12.5 L100,87.5 A50,12.5 0 0 1 0,87.5 Z"},
				{D: "M100,12.5 A50,12.5 0 0 1 0,12.5 A50,12.5 0 0 1 100,12.5", Overlay: true},
			},
			// 텍스트 영역: 위는 뚜껑 타원 **전체**(2·y1 = 25%) 아래, 아래는 바닥 반타원(y1 = 12.5%)
			// 위다 — 좌우는 변이 곧 박스다 (실측 t 100px = 25.0% · b 50px = 12.5% · l/r 0). 세
			// 종횡비에서 같은 비율이다 (h=320에서 80/40px, h=560에서 140/70px).
			Adjust:   []ShapeAdjust{{Name: "adj", Base: 25000, Axis: AdjustAxisHeight}},
			TextRect: ShapeInsets{T: 25, B: 12.5},
		},
		{
			// chevron: `adj=50000` → `x1 = ss·adj/100000` = 50(노치 깊이), `x2 = r - x1`.
			// 경로는 (l,t) → (x2,t) → (r,vc) → (x2,b) → (l,b) → (x1,vc).
			// Adjust: adj = 50000·w/ss (프리셋의 maxAdj = 100000·w/ss라 언제나 그 절반이다).
			Type: "chevron", Preset: "chevron", Space: spaceNormalized,
			// 텍스트 영역: **박스 전체**다. 프리셋의 rect는 `il`~`ir`이고 `dx = x2 − x1`에서
			// `il = ?: dx x1 l` · `ir = ?: dx x2 r`이다 — 노치가 폭의 절반이면 dx = 0이라 명세로는
			// 박스 전체(l~r)지만, LibreOffice는 0을 양수로 읽어 폭 0 열(x1~x2)에 한 글자씩 쌓고,
			// 반올림이 adj를 조금 내리면 모든 엔진이 가는 열이 된다. 그래서 조정값을 한 단위
			// 넘긴다(Exceed): dx < 0이면 어느 엔진이든 박스 전체다.
			Paths:  []shapePath{{D: "M0,0 L50,0 L100,50 L50,100 L0,100 L50,50 Z"}},
			Adjust: []ShapeAdjust{{Name: "adj", Base: 50000, Axis: AdjustAxisWidth, Exceed: true}},
		},
		{
			// rightArrow: `adj1=50000` → `dy1 = h·adj1/200000` = 25(축 절반, h 기준),
			// `adj2=50000` → `dx1 = ss·adj2/100000` = 50(머리 길이), `x1 = r - dx1`.
			// 경로는 (l,y1) → (x1,y1) → (x1,t) → (r,vc) → (x1,b) → (x1,y2) → (l,y2).
			// Adjust: adj1은 h 기준이라 프리셋 기본값(50000) 그대로이고, adj2 = 50000·w/ss가
			// 보정이다. adj1을 고정값으로 함께 적는 이유는 AdjustAxisFixed에 있다 (adj2만 적힌
			// avLst는 PowerPoint가 복구하며 도형을 지운다).
			Type: "arrow-right", Preset: "rightArrow", Space: spaceNormalized,
			// 텍스트 영역: 축 띠(`y1`~`y2` = 위아래 25%)이고 오른쪽은 머리가 시작하는 `x1`(= 75%w)이다
			// — 왼쪽은 변이 곧 박스다 (실측 t/b 100·95px ≈ 25% · r 101px = 25.2%).
			Paths: []shapePath{{D: "M0,25 L50,25 L50,0 L100,50 L50,100 L50,75 L0,75 Z"}},
			Adjust: []ShapeAdjust{
				{Name: "adj1", Base: 50000, Axis: AdjustAxisFixed},
				{Name: "adj2", Base: 50000, Axis: AdjustAxisWidth},
			},
			TextRect: ShapeInsets{T: 25, R: 25, B: 25},
		},
		{
			// hexagon: `adj=25000` → `x1 = ss·adj/100000` = 25, `x2 = r - x1`.
			// 세로는 `vf=115470`이 `shd2 = hd2·vf/100000`을, `dy1 = shd2·sin(60°)`이 hd2와
			// 같아져 `y1 = t`·`y2 = b`가 된다 (기본 육각형은 박스 높이를 꽉 채운다).
			// Adjust: adj = 25000·w/ss (프리셋의 maxAdj = 50000·w/ss 안이다). vf는 기본값(115470)
			// 그대로 함께 적는다 — 이유는 AdjustAxisFixed에 있다.
			Type: "hexagon", Preset: "hexagon", Space: spaceNormalized,
			// 텍스트 영역: 왼위 사선을 비율 좌표로 쓰면 `u = ¼(1 − 2v)`이고, 네 변이 같은 비율 d로
			// 들어오는 점은 `d = ¼ − ½d` → `d = ⅙` = 16.6667%다 (비율 공간의 식이라 종횡비에
			// 무관하다 — 실측 66~67px = 16.5~16.8%).
			Paths: []shapePath{{D: "M0,50 L25,0 L75,0 L100,50 L75,100 L25,100 Z"}},
			Adjust: []ShapeAdjust{
				{Name: "adj", Base: 25000, Axis: AdjustAxisWidth},
				{Name: "vf", Base: 115470, Axis: AdjustAxisFixed},
			},
			TextRect: ShapeInsets{L: 16.6667, T: 16.6667, R: 16.6667, B: 16.6667},
		},
		{
			// parallelogram: `adj=25000` → `x2 = ss·adj/100000` = 25(기울기), `x5 = r - x2`.
			// 경로는 (l,b) → (x2,t) → (r,t) → (x5,b).
			// Adjust: adj = 25000·w/ss (프리셋의 maxAdj = 100000·w/ss 안이다).
			Type: "parallelogram", Preset: "parallelogram", Space: spaceNormalized,
			// 텍스트 영역: 기울기 `x2`(25%)의 ¾ = 18.75%가 네 변이다 (실측 74~76px = 18.5~19.0%).
			// 사선에 **닿지 않는다** — 왼위 사선은 `u = ¼(1 − v)`이라 v = 18.75%에서 u = 20.31%이고,
			// 그래서 이 rect의 두 꼭짓점은 실루엣을 1.6%p 넘어간다. 프리셋의 rect는 내접 사각형이
			// 아니라 PowerPoint가 정한 본문 영역이라는 증거다 (테스트가 그 성질을 단정한다).
			Paths:    []shapePath{{D: "M0,100 L25,0 L100,0 L75,100 Z"}},
			Adjust:   []ShapeAdjust{{Name: "adj", Base: 25000, Axis: AdjustAxisWidth}},
			TextRect: ShapeInsets{L: 18.75, T: 18.75, R: 18.75, B: 18.75},
		},
		{
			// wedgeRectCallout: `adj1=-20833`·`adj2=62500` → 꼬리 끝이
			// `(hc + w·adj1/100000, vc + h·adj2/100000)` = (29.167, 112.5)이고, 꼬리 밑변은
			// 아래 변의 `2w/12`~`5w/12`다 (`dxPos<0`·`dyPos>0`·`dz>0`에서 고른 xg/yg 조건).
			// 나머지 꼭짓점은 사각형 변 위에서 겹쳐 사라진다 (프리셋의 16점 중 7점만 남는다).
			// 꼬리 끝이 박스 아래로 12.5% 나가므로 svg가 `overflow: visible`이어야 한다.
			// 텍스트 영역: 본문은 사각형 전체다 — 꼬리는 박스 밖이므로 인셋이 없다 (실측이 rect 행과 같다).
			Type: "callout", Preset: "wedgeRectCallout", Space: spaceNormalized,
			Paths: []shapePath{{D: "M0,0 L100,0 L100,100 L41.6667,100 L29.167,112.5 L16.6667,100 L0,100 Z"}},
		},
		{
			// star5: `adj=19098`(내부 반지름 비), `hf=105146`·`vf=110557`(정규화 계수).
			// 꼭짓점은 `dx1 = swd2·cos18°`·`dy1 = shd2·sin18°`, 안쪽 점은
			// `iwd2 = swd2·adj/50000`에서 나온다 — 0.0001이 남는 것은 프리셋 계수가 정확한
			// 별 비율의 근사값이기 때문이고, 그 잔차가 이 수치의 출처를 증명한다.
			// 텍스트 영역: 안쪽 오각형의 bbox다 — 위 두 안쪽 점(38.1968), 아래 점(76.3927), 좌우 안쪽
			// 점(30.902·69.098)이 곧 네 변이라 이 행의 경로 좌표를 그대로 읽는다 (실측 l/r 123~124px =
			// 30.8~31.0% · t 153px = 38.25% · b 95px = 23.75%). bbox의 아래 두 꼭짓점은 오각형 밖이지만
			// **별 안쪽**이다 (아래 팔이 덮는다).
			Type: "star", Preset: "star5", Space: spaceNormalized,
			Paths: []shapePath{{D: "M0.0001,38.1965 L38.1968,38.1968 L50,0 L61.8032,38.1968 " +
				"L99.9999,38.1965 L69.098,61.8031 L80.9016,99.9997 L50,76.3927 L19.0984,99.9997 L30.902,61.8031 Z"}},
			TextRect: ShapeInsets{L: 30.902, T: 38.1968, R: 30.902, B: 23.6073},
		},
		{
			// cloud: 프리셋은 43200×43200 경로 공간의 원호 11개다 (조정값 없음) — 좌표와
			// 반지름을 432로 나눠 정규 공간으로 옮기고, `arcTo(wR,hR,stAng,swAng)`를
			// SVG `A`로 바꿨다 (중심 = 현재점 − (wR·cos stAng, hR·sin stAng), 끝점 = 중심 +
			// (wR·cos(stAng+swAng), hR·sin(stAng+swAng)), large-arc = |swAng|>180°,
			// sweep = swAng>0 — 두 규약의 회전 방향이 같다).
			// 마지막 원호가 시작점으로 정확히 돌아오지 않는 것은 프리셋 자체의 성질이고
			// (`close`가 짧은 직선을 그린다) PowerPoint도 같은 자리를 같게 그린다.
			// 둘째 경로는 프리셋의 `fill="none"` 경로 — 구름 안쪽 선이다.
			// 텍스트 영역: 프리셋의 rect가 경로와 같은 21600 공간의 유리수로 적혀 있다 (l 2977 ·
			// t 3262 · r 17087 · b 17337 → 아래 백분율). 구름은 원호 열하나라 기하로 유도할 형태가
			// 없고, 실측이 세 종횡비에서 이 값과 0.3% 안으로 맞는다 (400×400에서 l 55 · t 60 · r 83 ·
			// b 79px).
			Type: "cloud", Preset: "cloud", Space: spaceNormalized,
			Paths: []shapePath{
				{D: "M9.0278,33.2639 A15.6319,21.2731 0 0 1 30.58,9.8525 A12.3449,16.8218 0 0 1 47.8089,6.0796 " +
					"A10.1042,13.7616 0 0 1 63.1748,3.8997 A11.2431,15.2662 0 0 1 81.47,11.7966 " +
					"A12.3449,16.8356 0 0 1 89.7095,36.805 A15.6829,21.3426 0 0 1 80.5068,73.4524 " +
					"A13.3912,18.2106 0 0 1 61.5295,89.8038 A15.6296,21.331 0 0 1 35.5384,96.0343 " +
					"A17.8704,24.4051 0 0 1 14.2659,88.7381 A10.0926,13.6991 0 0 1 7.1542,64.3254 " +
					"A10.0579,13.7616 0 0 1 10.356,37.9513 Z"},
				{D: "M10.8634,60.5949 A10.0579,13.7616 0 0 1 6.3342,59.507 " +
					"M16.037,80.7847 A10.0926,13.6991 0 0 1 14.0587,81.2904 " +
					"M38.1435,90.4861 A15.6296,21.331 0 0 1 35.9275,86.2203 " +
					"M66.7292,80.4421 A15.6296,21.331 0 0 1 65.6435,86.1911 " +
					"M79.0023,53.1343 A13.3912,18.2106 0 0 1 87.8769,70.4769 " +
					"M96.7546,35.5417 A12.3449,16.8356 0 0 1 92.6271,41.3313 " +
					"M88.713,12.5602 A11.2431,15.2662 0 0 1 89.0328,16.4842 " +
					"M67.3102,9.1481 A11.2431,15.2662 0 0 1 69.5916,5.4454 " +
					"M51.2523,10.9259 A10.1042,13.7616 0 0 1 52.5824,7.1022 " +
					"M32.4074,12.0185 A15.6319,21.2731 0 0 1 35.3285,14.2633 " +
					"M9.5532,36.5486 A15.6319,21.2731 0 0 1 8.6405,32.3329", Overlay: true},
			},
			TextRect: ShapeInsets{L: 13.7824, T: 15.1019, R: 20.8935, B: 19.7361},
		},
	}
}

// shapeGeometryOf는 프리셋 이름으로 기하를 찾는다.
//
// 모르는 `type`을 에러로 두는 이유는 containerClass와 같다: 어휘에 프리셋을 더하고 기하를
// 잊으면 "파싱은 되는데 그려지지 않는" 도형이 되고, 그것이 닫힌 어휘로 막으려던 실패다.
func shapeGeometryOf(shapeType string) (shapeGeometry, error) {
	for _, geom := range shapeGeometries() {
		if geom.Type == shapeType {
			return geom, nil
		}
	}
	return shapeGeometry{}, fmt.Errorf("도형 type %q의 기하가 없다 (프리셋 %s)",
		shapeType, strings.Join(soml.ShapeTypes(), ", "))
}

// ShapePreset은 SOML `type` → OOXML `prstGeom@prst` 이름이다.
//
// 내보내는 이유는 ListTextIndent·ElementID와 같다: 목록이 두 벌이 되면 프리셋 하나를 더할 때
// 한쪽만 바뀌고, 그 어긋남은 웹과 PPTX가 **서로 다른 도형**을 그리는 것으로만 드러난다.
// 기하 유도와 프리셋 이름이 한 표에 있어야 그 짝이 유지된다.
func ShapePreset(shapeType string) (string, bool) {
	geom, err := shapeGeometryOf(shapeType)
	if err != nil {
		return "", false
	}
	return geom.Preset, true
}

// ShapeAdjusts는 이 프리셋이 요구하는 `prstGeom/avLst` 조정값 명세다 — 없으면 빈 목록이다.
//
// 내보내는 이유는 ShapePreset과 같다: 조정값 공식은 **웹이 정규 공간을 늘린 것의 보정**이라
// (파일 머리 주석) 늘린 쪽과 같은 표에 있어야 한다. 표가 두 벌이면 기하를 고칠 때 보정을
// 잊고, 그 어긋남은 같은 프리셋이 두 타깃에서 다른 비율로 그려지는 것으로만 드러난다.
//
// ok=false는 모르는 `type`이다 — 빈 목록(조정값 없는 프리셋)과 구별해야 PPTX 렌더러가
// "어휘에 있는데 표에 없는 프리셋"에서 멈출 수 있다.
func ShapeAdjusts(shapeType string) ([]ShapeAdjust, bool) {
	geom, err := shapeGeometryOf(shapeType)
	if err != nil {
		return nil, false
	}
	return geom.Adjust, true
}

// RoundRectAdjust는 `roundRect` 프리셋의 반경 조정값 명세다 — **도형과 그림이 같은 것을
// 쓴다**.
//
// `<shape type="rounded-rect">`(위 기하 표의 행)와 `radius`가 있는 `<image>`
// (render/pptx의 `pic/spPr/prstGeom`)가 같은 프리셋을 같은 공식으로 되돌려야 한다: 같은
// `radius="md"`가 두 요소에서 다른 모서리로 그려지면 저자는 그 차이를 도형과 그림을 나란히
// 놓고서야 본다. 값(`adj = 100000·radius/ss`, 상한 50000 = 짧은 변의 절반)의 근거는 표의
// 그 행 주석에 있다 — 여기서 다시 적지 않는다.
//
// 함수인 이유는 ShapeAdjusts와 같다: 값 타입의 복사본을 주어 호출자가 표를 바꿀 수 없게 한다.
func RoundRectAdjust() ShapeAdjust {
	return ShapeAdjust{Name: "adj", Base: 100000, Axis: AdjustAxisRadius, Max: 50000}
}

// ShapeAnchor는 도형이 쓸 세로 앵커다 — **첫 자식이 정한 값**이고, 정한 자식이 없으면 빈 값이다.
//
// 왜 하나를 고르는가: 확정 값(TextStyle.Anchor)은 자식마다 있지만 앵커는 **박스의 성질**이다
// (웹은 flex 컨테이너의 `justify-content`, PPTX는 `bodyPr@anchor` — 도형에 하나뿐이다).
// 첫 자식을 고른 이유는 규칙이 문서 순서로 읽히기 때문이고, 뒤 자식이 다른 앵커를 적으면
// 그 값은 **조용히 버려진다** (경고는 값 층의 몫이고 렌더러에는 진단 통로가 없다).
// 내보내는 이유는 ShapePreset과 같다: PPTX 렌더러가 같은 규칙으로 `bodyPr@anchor`를
// 골라야 두 타깃의 앵커가 갈라지지 않는다.
func ShapeAnchor(elem *soml.Element) soml.TextAnchor {
	if elem == nil {
		return ""
	}
	for _, child := range elem.Children {
		child, ok := child.(*soml.Element)
		if !ok {
			continue
		}
		if style := child.TextStyle(); style != nil && style.Anchor != "" {
			return style.Anchor
		}
	}
	return ""
}

// shapeHasContent는 도형이 그릴 자식을 갖는지다 — 공백 문자 데이터는 자식이 아니다
// (`<shape>\n  </shape>`는 빈 도형이다). ShapeAnchor와 같은 자리에 두는 이유는 같은 사실을
// 묻기 때문이다: 텍스트 영역·앵커는 담을 것이 있을 때만 뜻이 있는 박스의 성질이다.
func shapeHasContent(elem *soml.Element) bool {
	if elem == nil {
		return false
	}
	for _, child := range elem.Children {
		if _, ok := child.(*soml.Element); ok {
			return true
		}
	}
	return false
}

// writeShape는 `<shape>`를 박스 div + 그림 svg + 자식 텍스트로 쓴다.
//
// 식별자(data-balpyo-id)를 갖는다 — 도형은 **재는 것과 도형이 되는 것이 처음 일치하는
// 종류**다 (측정 대상 계약, 패키지 주석): 컨테이너는 재고 사라지고 `<li>`는 부모 도형에
// 담기는데, 도형의 잰 rect는 곧 `xfrm`이 된다.
func writeShape(b *strings.Builder, deck *soml.Deck, slide *soml.Slide, elem *soml.Element, path []int) error {
	style := elem.ShapeStyle()
	if style == nil {
		// 해석되지 않은 외관을 기본값으로 그리면 웹만 그럴듯하게 나오고 PPTX와 갈라진다
		// (writeBlock·writeContainer와 같은 관문).
		return fmt.Errorf("%s:%s <%s>의 도형 외관이 해석되지 않았다 — core.Resolve를 거치지 않은 덱이다",
			slide.Source, elem.Span, elem.Name)
	}
	geom, err := shapeGeometryOf(style.Type)
	if err != nil {
		return fmt.Errorf("%s:%s <%s>: %w", slide.Source, elem.Span, elem.Name, err)
	}
	// 텍스트 영역은 **담을 것이 있을 때만** 모양을 바꾼다 — 빈 도형은 놓을 글자가
	// 없으므로 인셋이 아무것도 하지 않고, 그래도 내면 골든 열두 장에 빈 래퍼와 쓰이지 않는
	// 패딩이 남아 정작 텍스트가 어디 놓였는지가 묻힌다 (`anchor`가 첫 자식에서 오는 것과 같은
	// 자리: 박스 CSS의 일부는 내용이 있을 때만 뜻이 있다).
	text := geom.TextRect
	if !shapeHasContent(elem) {
		text = ShapeInsets{}
	}
	decls, err := shapeBoxCSS(*style, elem, text)
	if err != nil {
		return fmt.Errorf("%s:%s <%s>: %w", slide.Source, elem.Span, elem.Name, err)
	}
	decls = appendSlot(decls, elem)
	decls, err = appendPlacement(decls, elem.Placement())
	if err != nil {
		return fmt.Errorf("%s:%s <%s>: %w", slide.Source, elem.Span, elem.Name, err)
	}
	class := shapeClass
	if style.Rotate != nil {
		class += " " + shapeRotateClass
	}
	fmt.Fprintf(b, "<div class=%q %s=%q %s=%q%s>\n",
		class, IDAttr, ElementID(slide.Index, path), ElAttr, elem.Name, styleAttr(decls))
	if err := writeShapeInk(b, geom, *style, deck.Theme, ElementID(slide.Index, path)); err != nil {
		return fmt.Errorf("%s:%s <%s>: %w", slide.Source, elem.Span, elem.Name, err)
	}
	// 백분율 인셋 프리셋은 텍스트를 그 사각형에 담는 래퍼를 하나 더 갖는다 —
	// 인셋이 px인 rounded-rect는 패딩으로 받았으므로 구조가 오늘과 같다.
	body := text.HasPercent()
	if body {
		bodyDecls, err := shapeBodyCSS(*style, elem, text)
		if err != nil {
			return fmt.Errorf("%s:%s <%s>: %w", slide.Source, elem.Span, elem.Name, err)
		}
		fmt.Fprintf(b, "<div class=%q%s>\n", shapeBodyClass, styleAttr(bodyDecls))
	}
	for i, child := range elem.Children {
		if err := writeNode(b, deck, slide, child, ChildPath(path, i)); err != nil {
			return err
		}
	}
	if body {
		b.WriteString("</div>\n")
	}
	b.WriteString("</div>\n")
	return nil
}

// writeShapeInk는 기하를 그리는 svg를 쓴다 — 채움·테두리·그림자·불투명도·대칭이 여기 걸린다.
//
// 기본이 `aria-hidden`인 이유: 그림은 장식이고 내용은 안의 텍스트다. 스크린 리더가 읽을 것이
// 없는 요소를 접근성 트리에 남기면 도형마다 빈 그래픽이 읽힌다. **저자의 `alt`이 그 판단의
// 예외를 선언한다** — 그러면 같은 자리가 `role="img" aria-label`이 된다 (inkAccessAttrs).
func writeShapeInk(b *strings.Builder, geom shapeGeometry, style soml.ShapeStyle, theme soml.Theme, id string) error {
	decls, defs, err := shapeInkCSS(style, theme, id)
	if err != nil {
		return err
	}
	if geom.Space == spacePixel {
		fmt.Fprintf(b, "<svg class=%q%s%s>", shapeInkClass, inkAccessAttrs(style.Alt), styleAttr(decls))
		b.WriteString(defs)
		attrs := geom.Attrs
		if geom.Rounded {
			// 반경은 이 프리셋의 기하다 — `rx`만 낸다 (SVG는 `ry`를 `rx`로 채우고, OOXML
			// roundRect의 모서리도 원호다). 브라우저가 반경을 박스 절반으로 자르는 것은
			// 프리셋이 `adj`를 50000으로 pin하는 것과 같은 규칙이다.
			radius := style.Radius
			if radius == nil {
				// CSS 기본 반경은 0이고 OOXML roundRect의 기본 조정값은 16.67%다 — 값이
				// 비어 있으면 두 타깃이 서로 다른 모서리를 그린다 (core가 컴포넌트 기본값으로
				// 채우는 자리다).
				return fmt.Errorf("type=%q인데 반경이 해석되지 않았다", geom.Type)
			}
			attrs += fmt.Sprintf(" rx=%q", formatNumber(radius.Px))
		} else if style.Radius != nil {
			// 반경을 실을 자리가 없는 프리셋에 반경이 남아 있으면 **웹만 둥글어지고 PPTX는
			// 각진 채** 남는 길이 열린다 (core가 그 조합을 막는다 — core/shape.go).
			return fmt.Errorf("type=%q에는 반경을 실을 자리가 없다", geom.Type)
		}
		fmt.Fprintf(b, "<%s %s/>", geom.Element, attrs)
		b.WriteString("</svg>\n")
		return nil
	}
	if style.Radius != nil {
		return fmt.Errorf("type=%q에는 반경을 실을 자리가 없다", geom.Type)
	}
	fmt.Fprintf(b, "<svg class=%q viewBox=\"0 0 100 100\" preserveAspectRatio=\"none\"%s%s>",
		shapeInkClass, inkAccessAttrs(style.Alt), styleAttr(decls))
	b.WriteString(defs)
	for _, p := range geom.Paths {
		b.WriteString("<path")
		if p.Overlay {
			b.WriteString(` fill="none"`)
		}
		// 늘린 공간에서 선 굵기를 균일하게 유지한다 (spaceNormalized의 주석). 상속되지 않는
		// 프로퍼티라 경로마다 붙는다.
		fmt.Fprintf(b, ` d=%q vector-effect="non-scaling-stroke"/>`, p.D)
	}
	b.WriteString("</svg>\n")
	return nil
}

// shapeBoxCSS는 도형 박스의 CSS 선언이다 — 안쪽 여백·크기·비율·회전 각도·세로 앵커.
//
// 순서는 "박스 → 변형 → 정렬"이다: 박스 속성은 레이아웃 컨테이너와 **같은 함수**에서 나오고
// (boxSizeCSS — soml.Box를 두 요소가 공유하므로 매핑도 하나여야 한다), 그 뒤에 도형만의
// 선언이 온다.
//
// text(프리셋의 텍스트 영역)에 백분율이 있으면 **여백과 앵커는 여기서 빠진다** — 그 둘은
// 텍스트가 사는 사각형의 성질이고, 텍스트가 래퍼로 옮겨 가면 래퍼가 받아야 한다.
func shapeBoxCSS(style soml.ShapeStyle, elem *soml.Element, text ShapeInsets) ([]string, error) {
	var decls []string
	if !text.HasPercent() {
		// `box-sizing: border-box`(baseCSS)와 함께 잰 박스를 키우지 않고 안쪽으로 들어간다 —
		// 그래야 잰 rect가 도형의 겉 박스이고 텍스트는 그 안에 놓인다.
		//
		// 반경 비례 인셋(rounded-rect)이 여백에 **더해지는** 이유: 그 값은 px이므로 패딩으로
		// 표현할 수 있고, 패딩은 내용이 높이를 정하는 것을 막지 않는다 — 카드가 그대로 산다.
		if pad := shapePadValue(style.Pad, text.Radius*radiusPx(style.Radius)); pad != "" {
			decls = append(decls, "padding: "+pad)
		}
	}
	sizes, err := boxSizeCSS(style.Box)
	if err != nil {
		return nil, err
	}
	decls = append(decls, sizes...)
	if style.Rotate != nil {
		// 각도만 싣는다 — `transform` 선언은 엔진 CSS가 갖는다 (측정 뷰가 그것을 무효화할 수
		// 있어야 한다, 파일 머리 주석의 측정과 회전 절).
		decls = append(decls, fmt.Sprintf("%s: %sdeg", shapeRotateVar, formatNumber(*style.Rotate)))
	}
	if anchor := ShapeAnchor(elem); anchor != "" && !text.HasPercent() {
		value, err := anchorValue(anchor)
		if err != nil {
			return nil, err
		}
		// 세로 앵커는 도형이 세로 flex 컨테이너라는 사실 위에서 주축 분배가 된다 (외관 매핑 표의
		// `anchor` 행 — 부모 도형의 `justify-content`). 앵커를 정하지 않은 도형은 선언이 없고, flex의 기본값이
		// OOXML `bodyPr` 기본 앵커(위)와 같다.
		decls = append(decls, "justify-content: "+value)
	}
	return decls, nil
}

// shapeBodyCSS는 프리셋의 텍스트 영역을 차지하는 래퍼의 CSS 선언이다.
//
// 네 변을 백분율 `left`/`top`/`right`/`bottom`으로 내는 이유: 인셋의 기준은 도형 박스이고
// 절대 배치의 백분율은 **각 축의** 포함 블록 크기를 쓴다 (padding/margin 백분율은 네 변 모두
// 부모 **폭** 기준이라 세로 인셋을 표현할 수 없다 — 마름모의 위 25%가 폭의 25%가 된다).
// 대가는 도형에 확정 높이가 필요한 것이고, 값 층이 그 조합을 먼저 막는다 (core/shape.go).
//
// 여백과 앵커가 여기 오는 이유는 두 타깃의 뜻이 같아야 하기 때문이다: PPTX에서 `pad`는
// `bodyPr` 인셋이고 그것은 프리셋 텍스트 영역 **안쪽에** 들어간다 (실측).
func shapeBodyCSS(style soml.ShapeStyle, elem *soml.Element, text ShapeInsets) ([]string, error) {
	decls := []string{
		"left: " + percentValue(text.L),
		"top: " + percentValue(text.T),
		"right: " + percentValue(text.R),
		"bottom: " + percentValue(text.B),
	}
	if pad := shapePadValue(style.Pad, 0); pad != "" {
		decls = append(decls, "padding: "+pad)
	}
	if anchor := ShapeAnchor(elem); anchor != "" {
		value, err := anchorValue(anchor)
		if err != nil {
			return nil, err
		}
		decls = append(decls, "justify-content: "+value)
	}
	return decls, nil
}

// percentValue는 백분율 인셋의 CSS 값이다 (0도 적는다 — 네 변이 함께 있어야 래퍼가
// 박스를 채우는 사각형으로 읽힌다).
func percentValue(v float64) string { return formatNumber(round(v)) + "%" }

// radiusPx는 반경의 px 값이다 (반경이 없으면 0 — 반경을 실을 자리가 없는 프리셋이거나
// writeShapeInk가 곧 멈출 덱이다).
func radiusPx(radius *soml.Radius) float64 {
	if radius == nil {
		return 0
	}
	return radius.Px
}

// shapePadValue는 도형 안쪽 여백의 CSS 값이다 — 저자의 `pad`에 프리셋의 px 인셋을 더한다.
//
// `calc()`로 더하는 이유는 공간 토큰이 변수 참조로 남아야 하기 때문이다 (lengthValue의 근거 —
// 여기서 px로 눕히면 테마의 리듬을 바꿨을 때 이 도형만 옛 여백으로 남는다).
func shapePadValue(pad *soml.Pad, edge float64) string {
	edge = round(edge)
	if edge == 0 {
		if pad == nil {
			return ""
		}
		return padValue(*pad)
	}
	px := formatNumber(edge) + "px"
	if pad == nil {
		return px
	}
	sides := [...]soml.Length{pad.Top, pad.Right, pad.Bottom, pad.Left}
	parts := make([]string, 0, len(sides))
	for _, side := range sides {
		parts = append(parts, fmt.Sprintf("calc(%s + %s)", lengthValue(side), px))
	}
	return strings.Join(parts, " ")
}

// anchorValue는 세로 앵커를 CSS 주축 분배 값으로 만든다.
//
// 물리 이름(top/bottom)을 CSS의 논리 이름(start/end)으로 옮기는 자리다 — 어휘가 물리 이름인
// 근거는 세로축에 방향 전환이 없다는 것이고(soml.TextAnchor), CSS 쪽 이름은 boxJustifyValue와
// 같은 어휘를 써서 justify-content 값의 표기가 한 가지로 남는다.
func anchorValue(anchor soml.TextAnchor) (string, error) {
	switch anchor {
	case soml.AnchorTop:
		return "start", nil
	case soml.AnchorCenter:
		return "center", nil
	case soml.AnchorBottom:
		return "end", nil
	}
	return "", fmt.Errorf("anchor %q는 세로 앵커 어휘 밖이다", string(anchor))
}

// shapeInkCSS는 확정된 도형 외관을 그림의 CSS 선언 목록으로 컴파일한다 (외관 매핑 표의 도형 외관).
//
// **순서는 매핑 표의 행 순서다** (fill → border → shadow → opacity → flip) — 표와 코드를 나란히
// 읽을 수 있어야 "표에 있고 코드에 없는 행"이 눈에 띈다 (그 대조를 테스트가 어휘 목록
// 순회로 한 번 더 한다). `radius`는 기하이므로 여기 없고(writeShapeInk), `rotate`는 박스
// 쪽이다 (shapeBoxCSS).
//
// 정하지 않은 속성은 선언을 **하나도** 내지 않는다 — `shadow`·`opacity`·`flip`의 nil은
// "선언 없음"이고 두 타깃의 암묵 기본값이 같다 (soml.ShapeStyle의 계약).
func shapeInkCSS(style soml.ShapeStyle, theme soml.Theme, id string) ([]string, string, error) {
	var decls []string
	var defs []string
	// fill: 토큰은 변수 참조, 리터럴은 #RRGGBB, `none`은 SVG의 `none`이다 (투명한 채움을
	// `transparent`로 내지 않는 이유: SVG에서 `fill: none`은 그리지 않는다는 뜻이라
	// OOXML `a:noFill`과 정확히 짝이 된다). 그라디언트는 svg `<defs>`의 `linearGradient`를
	// 만들고 `fill: url(#…)`로 가리킨다 (CSS `border-image`가 radius와 조합되지
	// 않아 엔진이 SVG def를 소유한다).
	if style.Fill == nil {
		return nil, "", fmt.Errorf("채움이 해석되지 않았다")
	}
	if style.Fill.None {
		decls = append(decls, "fill: none")
	} else if style.Fill.Color.IsGradient() {
		gradID := id + "-fill-grad"
		def, err := linearGradientDef(gradID, *style.Fill.Color.Gradient, theme)
		if err != nil {
			return nil, "", err
		}
		defs = append(defs, def)
		decls = append(decls, fmt.Sprintf("fill: url(#%s)", gradID))
	} else {
		fill, err := colorValue(style.Fill.Color, theme)
		if err != nil {
			return nil, "", err
		}
		decls = append(decls, "fill: "+fill)
	}
	// border: 굵기·스타일·색이 각자의 선언으로 갈라진다 (축약을 다시 만들지 않는 이유는
	// soml.Border의 주석 — 렌더러가 축약을 해석하면 규칙이 두 곳에 생긴다).
	if style.Border == nil {
		return nil, "", fmt.Errorf("테두리가 해석되지 않았다")
	}
	if style.Border.None {
		decls = append(decls, "stroke: none")
	} else {
		if style.Border.Width <= 0 {
			return nil, "", fmt.Errorf("테두리 굵기가 해석되지 않았다 (%s)", formatNumber(style.Border.Width))
		}
		dash, err := dashArray(style.Border.Style, style.Border.Width)
		if err != nil {
			return nil, "", err
		}
		// 그라디언트 테두리는 stroke가 gradient def를 가리킨다 (fill과 같은 수단 — SVG
		// stroke는 `url(#…)`를 받는다). 외관 매핑 표가 `stroke: url(#…)`로 적은
		// 자리이고, 채움 def가 이미 있으니 테두리는 def를 하나 더 만들 뿐이다.
		var stroke string
		if style.Border.Color.IsGradient() {
			gradID := id + "-stroke-grad"
			def, err := linearGradientDef(gradID, *style.Border.Color.Gradient, theme)
			if err != nil {
				return nil, "", err
			}
			defs = append(defs, def)
			stroke = fmt.Sprintf("url(#%s)", gradID)
		} else {
			s, err := colorValue(style.Border.Color, theme)
			if err != nil {
				return nil, "", err
			}
			stroke = s
		}
		decls = append(decls, "stroke: "+stroke,
			"stroke-width: "+formatNumber(style.Border.Width)+"px")
		if dash != "" {
			decls = append(decls, "stroke-dasharray: "+dash)
		}
	}
	if style.Shadow != nil {
		// 그림자는 **변수 참조**다 (선언은 themeCSS의 shadowVarsCSS — 참조와 선언이 한 표에서
		// 나온다는 계약). `box-shadow`가 아니라 `filter: drop-shadow`인 이유: box-shadow는
		// 박스의 사각형을 따르므로 열두 종 중 아홉이 엉뚱한 사각 그림자를 얻는다. drop-shadow는
		// 실루엣의 알파를 따라가 OOXML `outerShdw`와 같은 그림이 된다 (외관 매핑 표의
		// `shadow` 행이 `drop-shadow`인 이유다).
		if _, ok := theme.Shadow(style.Shadow.Name); !ok {
			// var()는 정의되지 않은 변수를 조용히 무시한다 — 검사하지 않으면 그림자가
			// 사라지고 아무 에러도 나지 않는다 (colorValue와 같은 근거).
			return nil, "", fmt.Errorf("그림자 토큰 %q가 테마에 없다", style.Shadow.Name)
		}
		decls = append(decls, fmt.Sprintf("filter: drop-shadow(var(%s%s))", shadowVarPrefix, style.Shadow.Name))
	}
	if style.Opacity != nil {
		// **채움·테두리에만** 걸린다 (그림 svg의 선언이므로 자식 텍스트는 흐려지지 않는다).
		// OOXML이 알파를 `solidFill`·`ln` 안에 싣는 것과 짝이다 — 도형 전체 투명도는
		// `a:alphaModFix` 효과이고 두 타깃이 그 지점에서 갈라지면 같은 덱이 다르게 보인다.
		if *style.Opacity < 0 || *style.Opacity > 1 {
			return nil, "", fmt.Errorf("불투명도 %s가 0~1 밖이다", formatNumber(*style.Opacity))
		}
		decls = append(decls, "opacity: "+formatNumber(*style.Opacity))
	}
	if style.Flip != "" {
		flip, err := flipValue(style.Flip)
		if err != nil {
			return nil, "", err
		}
		// 대칭은 **그림에만** 걸린다: PowerPoint의 flipH/flipV는 기하를 뒤집지만 글자는
		// 읽히는 방향으로 남는다. 박스에 걸면 CSS가 글자까지 거울로 만들어 두 타깃이
		// 갈라진다. 그림에만 걸리므로 잰 rect도 움직이지 않는다 (회전과 다른 점이다).
		decls = append(decls, "transform: "+flip)
	}
	return decls, defsBlock(defs), nil
}

// defsBlock은 그라디언트 정의들을 svg `<defs>` 블록 하나로 감싼다 (없으면 빈 문자열 —
// def가 없는 도형의 산출물에는 `<defs>`가 생기지 않는다).
func defsBlock(defs []string) string {
	if len(defs) == 0 {
		return ""
	}
	return "<defs>" + strings.Join(defs, "") + "</defs>"
}

// linearGradientDef는 그라디언트를 svg `<linearGradient>` 정의로 만든다 — 도형 채움·테두리가
// `url(#id)`로 가리킨다.
//
// 좌표는 `objectBoundingBox`(기본 gradientUnits)라 도형 박스에 상대적이다 — PPTX `a:lin`의
// `scaled="1"`과 같은 결정이다 (그라디언트가 도형 크기에 맞춰 늘어난다). 각도 벡터는
// CSS 시맨틱(0deg=아래→위, 시계방향)을 박스 대각선으로 옮긴 것이다 (gradientVector). 스톱
// 색은 `style="stop-color: …"`로 낸다: 프리젠테이션 속성 `stop-color`는 `var()`를 풀지
// 못하지만 CSS 프로퍼티는 풀어서 토큰 참조가 살아남는다.
func linearGradientDef(id string, grad soml.Gradient, theme soml.Theme) (string, error) {
	if len(grad.Stops) < 2 {
		return "", fmt.Errorf("그라디언트 스톱이 둘 미만이다 (%d)", len(grad.Stops))
	}
	x1, y1, x2, y2 := gradientVector(grad.Angle)
	var b strings.Builder
	fmt.Fprintf(&b, `<linearGradient id=%q x1=%q y1=%q x2=%q y2=%q>`,
		id, formatNumber(x1), formatNumber(y1), formatNumber(x2), formatNumber(y2))
	last := len(grad.Stops) - 1
	for i, stop := range grad.Stops {
		offset := formatNumber(round4(float64(i) / float64(last)))
		color, err := baseColorValue(stop, theme)
		if err != nil {
			return "", err
		}
		fmt.Fprintf(&b, `<stop offset=%q style="stop-color: %s"/>`, offset, color)
	}
	b.WriteString("</linearGradient>")
	return b.String(), nil
}

// gradientVector는 CSS 그라디언트 각도를 objectBoundingBox의 시작·끝 점으로 옮긴다.
//
// CSS는 0deg가 위(아래→위)이고 시계방향이다 (soml.Gradient). 방향 벡터는 (sinθ, -cosθ)이고
// (y축이 아래로 향하므로 위는 -cos), 박스 중심(0.5, 0.5)에서 ±0.5만큼 벌려 시작·끝을 잡는다.
// 180deg면 시작 (0.5,0)·끝 (0.5,1) = 위→아래로, CSS `linear-gradient(180deg)`와 같은
// 방향이다. 정사각형이 아닌 박스에서는 objectBoundingBox가 각을 기울여 CSS의 "magic corner"
// 보정과 미세하게 갈라지지만, 세로·가로(0/90/180/270deg)는 종횡비와 무관해 일치한다.
func gradientVector(cssDeg float64) (x1, y1, x2, y2 float64) {
	rad := cssDeg * math.Pi / 180
	sin, cos := math.Sin(rad), math.Cos(rad)
	x1 = round4(0.5 - 0.5*sin)
	y1 = round4(0.5 + 0.5*cos)
	x2 = round4(0.5 + 0.5*sin)
	y2 = round4(0.5 - 0.5*cos)
	return x1, y1, x2, y2
}

// round4는 소수 넷째 자리에서 반올림한다 — svg 좌표에 부동소수 잡음(0.49999999)이 실리지
// 않게 한다 (골든 안정성).
func round4(v float64) float64 {
	return math.Round(v*1e4) / 1e4
}

// dashArray는 테두리 스타일을 SVG `stroke-dasharray` 값으로 만든다 (실선은 빈 문자열).
//
// 수치는 OOXML `prstDash`의 비율이다 — ECMA-376의 프리셋 파선 표에서 `dash`가 4:3,
// `dot`이 1:3이고 단위는 **선 굵기**다 (`sys*` 계열은 3:1·1:1로 촘촘한 다른 값이다 —
// 계열 선택은 PPTX 매핑이 소유하고 render/pptx.prstDashValue가 이 수치와 짝을 맞춘다).
// CSS `border-style: dashed`의 패턴은 UA가 정하므로 그 이름을 그대로 옮길 수 없고, 여기서
// 수치를 정하는 편이 두 타깃이 같은 간격을 갖는 유일한 길이다.
func dashArray(style soml.BorderStyle, width float64) (string, error) {
	pattern := func(dash, gap float64) string {
		return formatNumber(dash*width) + "px " + formatNumber(gap*width) + "px"
	}
	switch style {
	case soml.BorderSolid:
		return "", nil
	case soml.BorderDashed:
		return pattern(4, 3), nil
	case soml.BorderDotted:
		return pattern(1, 3), nil
	}
	return "", fmt.Errorf("테두리 스타일 %q는 어휘 밖이다", string(style))
}

// flipValue는 대칭을 CSS transform 값으로 만든다.
func flipValue(flip soml.Flip) (string, error) {
	switch flip {
	case soml.FlipH:
		return "scaleX(-1)", nil
	case soml.FlipV:
		return "scaleY(-1)", nil
	case soml.FlipBoth:
		return "scale(-1, -1)", nil
	}
	return "", fmt.Errorf("flip %q는 대칭 어휘 밖이다", string(flip))
}

// shadowVarsCSS는 그림자 토큰의 CSS 변수 선언이다 — 참조(shapeInkCSS)와 선언이 **한 표**에서
// 나온다는 계약의 나머지 절반이다 (색·폰트·공간과 같은 구조 — themeCSS).
//
// 값의 모양은 `drop-shadow()`의 인자다: `<dx>px <dy>px <blur>px <색>`. 극좌표(dist·dir)를
// 직교로 푸는 자리가 여기다 — 토큰이 극좌표인 이유는 OOXML `outerShdw`가 극좌표이기
// 때문이고(soml.ShadowToken), 웹은 직교만 받는다. dir는 시계방향·0이 오른쪽이고 y축이 아래로
// 향하므로 두 타깃의 변환 공식이 같다.
//
// 색은 **토큰 참조를 유지한다**: 투명도를 `color-mix()`로 얹으면 변수 참조가 살아남아 테마
// 색을 바꿨을 때 그림자도 따라 움직인다 (외관 매핑 표의 색 변형 표가 `color-mix()`를 가리킨다).
// rgba로 눕히면 그 요소만 옛 색으로 남는다.
//
// 선언 순서는 표 순서다 (soml.Theme가 맵이 아닌 이유) — 골든이 실행마다 같아야 한다.
func shadowVarsCSS(theme soml.Theme) string {
	var b strings.Builder
	for _, token := range theme.Shadows {
		dx := token.Dist * cosDeg(token.Dir)
		dy := token.Dist * sinDeg(token.Dir)
		fmt.Fprintf(&b, "    %s%s: %spx %spx %spx %s;\n",
			shadowVarPrefix, token.Name,
			formatNumber(round(dx)), formatNumber(round(dy)), formatNumber(token.Blur),
			shadowColorValue(token))
	}
	return b.String()
}

// shadowColorValue는 그림자 색이다 — 불투명도가 1이면 색 그대로, 아니면 color-mix로 얹는다.
func shadowColorValue(token soml.ShadowToken) string {
	color := "#" + token.Color.RGB
	if token.Color.IsToken() {
		// 테마에 없는 토큰은 여기서 걸러지지 않는다 — 그림자 **참조**를 내기 전에
		// shapeInkCSS가 토큰 표를 확인하고, 표에 있는 토큰의 색은 core가 해석한 값이다.
		color = fmt.Sprintf("var(%s%s)", colorVarPrefix, token.Color.Token)
	}
	if token.Alpha >= 1 {
		return color
	}
	return fmt.Sprintf("color-mix(in srgb, %s %s%%, transparent)", color, formatNumber(round(token.Alpha*100)))
}

// shapeCSS는 도형의 엔진 CSS다 — **태그마다 고정인 선언만** 있다 (파일 머리 주석의 경계).
//
// 도형이 있는 덱에만 낸다 (listCSS·layoutCSS와 같은 근거): 도형이 없는 덱의 산출물이 도형
// 렌더가 붙었다는 이유로 달라지면 골든 diff가 무엇이 움직였는지 말해 주지 못한다.
func shapeCSS() string {
	return fmt.Sprintf(`
  /* <shape>는 **박스 하나와 그림 하나**다. 박스가 잰 rect(=PPTX xfrm)이고 그림은 절대
     배치로 흐름에서 빠지므로, 프리셋 기하가 배치에 영향을 주지 않는다. 세로 flex인 이유는
     도형 안 텍스트가 문서 순서대로 쌓이고 세로 앵커가 주축 분배가 되기 때문이다. */
  .balpyo-shape {
    position: relative;
    display: flex;
    flex-direction: column;
  }

  /* 그림은 박스를 덮는다. overflow: visible이어야 하는 이유가 둘이다 — 선이 경로 중앙에
     그려져 굵기의 절반이 박스 밖으로 번지고(OOXML a:ln과 같은 규약), callout의 꼬리가
     박스 아래로 나간다. UA는 바깥 <svg>에 overflow: hidden을 걸므로 명시한다. */
  .balpyo-shape-ink {
    position: absolute;
    left: 0;
    top: 0;
    width: 100%%;
    height: 100%%;
    overflow: visible;
  }

  /* 그림이 절대 배치라 흐름 콘텐츠보다 위에 칠해진다 — 자식을 position: relative로 올려
     글자가 그림 위에 오게 한다. z-index로 그림을 내리는 길은 스택 문맥에 따라 슬라이드
     바탕 아래로 사라질 수 있어 쓰지 않는다.

     목록이 다섯 줄인 이유는 **배경 모드**다: 도형이 컨테이너를 담으면 올릴 것이
     글자가 아니라 컨테이너다. 자식 전부를 한 줄로 집는 길(> :not(.balpyo-shape-ink))은 쓰지
     않는다 — 특정도 (0,2,0)이 아래 .balpyo-shape-body (0,1,0)를 이겨 프리셋 텍스트 영역의
     절대 배치가 풀린다 (텍스트 영역 래퍼가 흐름으로 돌아간다). */
  .balpyo-shape > .balpyo-text,
  .balpyo-shape > .balpyo-row,
  .balpyo-shape > .balpyo-col,
  .balpyo-shape > .balpyo-grid,
  .balpyo-shape > .balpyo-stack {
    position: relative;
  }

  /* 프리셋의 텍스트 영역. 네 변의 인셋은 인라인 백분율이고, 이 규칙은 "그
     사각형이 박스 안의 절대 배치 상자이며 텍스트가 세로로 쌓인다"는 정의만 갖는다.
     그림보다 문서 순서에서 뒤에 오므로 글자가 그림 위에 칠해진다 (위 규칙과 같은 이유로
     z-index를 쓰지 않는다). */
  .balpyo-shape-body {
    position: absolute;
    display: flex;
    flex-direction: column;
  }

  /* 회전 — 선언은 여기, 각도는 인라인 커스텀 프로퍼티다. 측정 뷰가 이 선언을 무효화해야
     잰 rect가 **회전 전** 박스가 된다 (getBoundingClientRect는 회전된 bbox를 준다: 45도에서
     1.41배). 인라인 style은 층 밖에서 이기므로 선언이 요소 쪽에 있으면 덮을 수 없다. */
  .balpyo-shape-rotate {
    transform: rotate(var(%s, 0deg));
  }
  html[%s=%q] .balpyo-shape-rotate {
    transform: none;
  }
`, shapeRotateVar, ViewAttr, MeasureView)
}

// cosDeg·sinDeg는 도 단위 삼각함수다 (그림자 극좌표 → 직교 변환).
func cosDeg(deg float64) float64 { return math.Cos(deg * math.Pi / 180) }
func sinDeg(deg float64) float64 { return math.Sin(deg * math.Pi / 180) }

// round는 삼각함수의 잔차를 산출물에서 지운다 — `cos(90°)`가 정확히 0이 아니어서
// `2.4492935982947064e-16px`가 골든에 실리는 것을 막는다. 소수 네 자리는 논리 px의 1/10000이라
// 그림자 위치에 뜻이 없는 자릿수이고, 반올림 자리를 여기 하나로 두면 산출이 결정적이다.
func round(v float64) float64 { return math.Round(v*1e4) / 1e4 }
