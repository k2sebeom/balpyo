package pptx

import (
	"fmt"
	"math"

	"github.com/k2sebeom/balpyo/render/geom"
	"github.com/k2sebeom/balpyo/render/web"
	"github.com/k2sebeom/balpyo/soml"
)

// 단위 변환 상수.
//
// 저자와 지오메트리 패스는 논리 스테이지 px 하나만 쓴다. OOXML로 넘어가는 경계에서만
// 아래 계수로 환산하므로 좌표계 분기가 없다.
const (
	// EMUPerPixel은 논리 px 하나가 갖는 EMU 수다. 96 DPI 기준 (914400 EMU/inch ÷ 96).
	EMUPerPixel = 9525
	// EMUPerInch는 OOXML의 인치 환산 계수다.
	EMUPerInch = 914400
	// PointsPerPixel은 폰트 크기 환산 계수다 (16px = 12pt).
	PointsPerPixel = 0.75
	// UnitsPerDegree는 OOXML 각도 단위다 (ST_Angle — 60000분의 1도). `xfrm@rot`과
	// `outerShdw@dir`가 같은 단위를 쓴다.
	UnitsPerDegree = 60000
	// FullRotation은 한 바퀴다 (360 × UnitsPerDegree) — 각도 정규화의 법(modulus).
	FullRotation = 360 * UnitsPerDegree
	// PercentScale은 OOXML의 100000분율이다 (ST_Percentage의 정수 표기). 알파
	// (`a:alpha@val`)와 프리셋 조정값(`a:gd fmla="val N"`)이 같은 눈금을 쓴다 — 둘 다
	// "100000 = 100%"이고, 조정값에서 그 100%가 무엇의 100%인지는 프리셋이 정한다
	// (web.ShapeAdjust).
	PercentScale = 100000
)

// EMU는 논리 px를 EMU로 바꾼다. OOXML의 xfrm 좌표 단위다.
func EMU(px float64) int64 {
	return int64(math.Round(px * EMUPerPixel))
}

// FontSize는 논리 px 폰트 크기를 OOXML `sz` 속성 값(1/100 pt)으로 바꾼다.
func FontSize(px float64) int64 {
	return int64(math.Round(px * PointsPerPixel * 100))
}

// LineSpacing은 줄높이 배수와 폰트 크기(논리 px)를 OOXML `a:spcPts@val`(1/100 pt)로 바꾼다.
//
// 배수를 그대로 `spcPct`로 내지 않는 이유: 두 표기는
// 기준이 다르다. CSS line-height 배수는 **폰트 크기**의 배수이지만, LibreOffice(그리고
// PowerPoint 계열의 구현들)는 spcPct를 **폰트 고유 줄 높이**(ascent+descent, Noto Sans는
// ≈1.36em)의 배수로 해석한다 — 같은 150%가 웹에서는 1.5em, LibreOffice에서는 ≈2.04em이
// 되어 모든 글줄이 8~13px 아래로 밀린다. 절대값(pt)으로 눕히면 두 엔진이 같은 줄 전진
// 폭을 쓴다.
func LineSpacing(multiplier, sizePx float64) int64 {
	return int64(math.Round(multiplier * sizePx * PointsPerPixel * 100))
}

// Offset은 지오메트리 패스가 잰 박스의 좌상단을 OOXML `a:off`(EMU)로 바꾼다.
//
// **단위 경계는 이 함수와 Extent·FontSize뿐이다**. 지오메트리 패스는 논리 px만
// 알고 EMU를 모른다 (units_test.go가 그것을 못 박는다). 경계를 한 곳에 두는 이유: 환산이
// 여러 곳에 흩어지면 어떤 값이 px이고 어떤 값이 EMU인지 타입으로 드러나지 않아, 두 번 환산된
// 좌표(도형이 화면 밖으로 나간다)나 환산되지 않은 좌표(도형이 왼쪽 위 점에 뭉친다)가 생긴다.
//
// 좌표는 슬라이드 상대이므로 그대로 `a:off`가 된다 — 슬라이드 원점이 두 좌표계에서 같다.
// 음수도 그대로 넘긴다: 스테이지를 넘어간 박스는 잘리지 않고 넘친다.
func Offset(r geom.Rect) (x, y int64) {
	return EMU(r.X), EMU(r.Y)
}

// Extent는 박스 크기를 OOXML `a:ext`(EMU)로 바꾼다.
//
// 크기 0을 여기서 막지 않는다 — 크기 0인 도형이 유효한지는 조립하는 쪽의 판단이고(빈 요소를
// 낼지 건너뛸지), 여기서 에러를 내면 환산 함수가 조립 정책을 갖게 된다.
func Extent(r geom.Rect) (cx, cy int64) {
	return EMU(r.W), EMU(r.H)
}

// SlideSize는 논리 스테이지를 PPTX 페이지 크기(EMU)로 바꾼다.
//
// 16:9(1280×720)는 12192000×6858000 = 13.333×7.5in, 4:3(960×720)은
// 9144000×6858000 = 10×7.5in — 둘 다 PowerPoint 표준 페이지 크기와 정확히 일치한다.
func SlideSize(stage soml.Stage) (cx, cy int64) {
	return EMU(float64(stage.Width)), EMU(float64(stage.Height))
}

// Rotation은 시계방향 각도(도)를 OOXML `xfrm@rot`으로 바꾼다.
//
// 두 타깃의 회전 방향 규약이 같아 부호를 뒤집지 않는다 (CSS `rotate()`도 시계방향이고 y축이
// 아래로 향한다 — soml.ShapeStyle.Rotate). 그래서 이 함수가 하는 일은 단위와 **정규화**뿐이다:
// 음수와 360 이상을 한 바퀴 안으로 접는다. 접는 이유는 뷰어의 관용에 기대지 않는 것이다 —
// `rot="-2700000"`을 PowerPoint는 받지만 재저장하면 정규화해서 쓰므로, 우리가 먼저 정규화해
// 두면 저자가 파일을 열어 저장했을 때의 diff가 이 속성에서 나지 않는다 (zipEpoch·rel id를
// 결정적으로 두는 것과 같은 태도).
//
// 회전 **중심**은 환산에 없다: OOXML은 `ext`의 중심을 돌리고 웹의 `transform-origin` 기본값이
// 같은 중심이라 보정할 값이 없다 (render/web/shape.go의 회전 절). 잰 rect도 회전 **전** 박스다
// (측정 뷰가 변형을 끈다 — web.MeasureView), 그래서 좌표에 손댈 것이 없다.
func Rotation(deg float64) int64 {
	rot := int64(math.Round(deg*UnitsPerDegree)) % FullRotation
	if rot < 0 {
		rot += FullRotation
	}
	return rot
}

// Alpha는 0~1 불투명도를 OOXML `a:alpha@val`(100000분율)로 바꾼다.
//
// 범위를 여기서 막지 않는다 — 0~1 검사는 값 층(core)의 몫이고, 환산 함수가 검증을 가지면
// 같은 규칙이 두 곳에 생긴다 (Extent가 크기 0을 막지 않는 것과 같은 근거).
func Alpha(v float64) int64 {
	return int64(math.Round(v * PercentScale))
}

// LumPct는 백분율(0~100)을 OOXML 광도 변조 값(100000분율)으로 바꾼다 — `a:lumMod`/`a:lumOff`의
// `val`이다.
//
// **색 변형(shade/tint)이 여기로 온다** (외관 매핑 표의 색 변형 표 — PPTX는 `lumMod`/`lumOff`):
// `shade(색, p%)`는 어둡게이므로 광도를 (100-p)%로 조절한다(`lumMod val`), `tint`은 밝게이므로
// 같은 lumMod에 `lumOff val=p%`를 더한다. 이 표기가 PowerPoint의 테마 색 변형 UI가 쓰는
// 것과 같아, 변형된 토큰 색도 `schemeClr` 참조를 유지한 채 테마 색 편집을 따라간다
// (soml.ColorMod). 웹은 같은 공간의 같은 식을 CSS 상대 색(`hsl(from … h s calc(l …))`)으로 낸다
// (render/web) — lumMod/lumOff는 HSL의 L에 곱하고 더하는 연산이다.
func LumPct(pct float64) int64 {
	return int64(math.Round(pct / 100 * PercentScale))
}

// GradientAngle은 CSS 그라디언트 각도를 OOXML `gradFill`의 `a:lin@ang`으로 바꾼다.
//
// CSS `linear-gradient`는 0deg = 아래→위, 시계방향이고(soml.Gradient), OOXML `a:lin@ang`은
// 0 = 왼→오, 시계방향(60000분의 1도)이다 — 기준축이 90도 어긋나 있다. 그래서 변환은
// `ooxml = (css - 90) mod 360`이다: CSS 180deg(위→아래)는 OOXML 90도(위→아래)가 되고,
// CSS 90deg(왼→오)는 OOXML 0도가 된다. y축이 두 좌표계에서 모두 아래로 향하므로 부호를
// 뒤집지 않는다 (회전·그림자 방향과 같은 규약, Rotation의 주석).
func GradientAngle(cssDeg float64) int64 {
	d := math.Mod(cssDeg-90, 360)
	if d < 0 {
		d += 360
	}
	return int64(math.Round(d * UnitsPerDegree))
}

// AdjustValue는 프리셋 조정값 명세를 잰 박스로 평가한다 — `prstGeom/avLst`의 `val N`이다.
//
// 공식은 `val = Base × 값 ÷ ss`이고 `ss = min(w, h)`다 (web.ShapeAdjust). 이 계산이 PPTX
// 쪽에 있는 이유는 **웹이 도형의 픽셀 크기를 모르기 때문**이다: 웹은 프리셋을 정사각형에서
// 평가한 정규 공간에 두고 박스에 늘리고(render/web/shape.go), 잰 박스는 지오메트리 패스를
// 지나 이 렌더러에만 온다. 그래서 명세의 소유자(웹의 기하 표)와 평가의 소유자(여기)가
// 갈라진다 — ListTextIndent를 웹이 소유하고 EMU 환산을 여기서 하는 것과 같은 구조다.
//
// 상한에서 자르는 이유: 프리셋은 조정값을 `pin`으로 자르므로 넘긴 값도 파일이 **열리고**
// 뷰어가 조용히 잘라 그린다 — 넘겼다는 사실이 산출물의 모양으로도 드러나지 않는다.
// 여기서 자르면 골든 XML이 실제로 그려질 값을 담는다.
//
// 크기 0 도형(`ss = 0`)은 조정값 0이다. 나눗셈을 피하려는 것이 아니라 그릴 것이 없기
// 때문이고, 0으로 나눈 Inf가 `val`에 실리면 파일이 열리지 않는다.
func AdjustValue(spec web.ShapeAdjust, rect geom.Rect, radius float64) (int64, error) {
	var value float64
	switch spec.Axis {
	case web.AdjustAxisWidth:
		value = rect.W
	case web.AdjustAxisHeight:
		value = rect.H
	case web.AdjustAxisRadius:
		value = radius
	default:
		// 기하 표에 축을 추가하고 평가를 잊으면 조용히 0(각진 모서리·기본 비율)이 되는
		// 대신 멈춘다 — 렌더 매핑 부재를 에러로 두는 이 패키지의 태도 그대로다.
		return 0, fmt.Errorf("조정값 %s의 축 %q를 모른다", spec.Name, string(spec.Axis))
	}
	ss := math.Min(rect.W, rect.H)
	if ss <= 0 {
		return 0, nil
	}
	val := math.Round(spec.Base * value / ss)
	if spec.Max > 0 && val > spec.Max {
		val = spec.Max
	}
	if val < 0 {
		// 음수 반경·크기는 값 층이 막지만(core), 여기서도 0으로 눕힌다 — 음수 조정값은
		// 프리셋마다 뜻이 다르고(callout의 꼬리는 음수가 정상이다) 반경에는 뜻이 없다.
		val = 0
	}
	return int64(val), nil
}
