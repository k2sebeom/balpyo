package pptx

import (
	"math"

	"github.com/k2sebeom/balpyo/render/geom"
	"github.com/k2sebeom/balpyo/soml"
)

// 단위 변환 상수 (PRD §13-5).
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
// 배수를 그대로 `spcPct`로 내지 않는 이유(Iteration 1의 측정, NOTES.md 01/07): 두 표기는
// 기준이 다르다. CSS line-height 배수는 **폰트 크기**의 배수이지만, LibreOffice(그리고
// PowerPoint 계열의 구현들)는 spcPct를 **폰트 고유 줄 높이**(ascent+descent, Noto Sans는
// ≈1.36em)의 배수로 해석한다 — 같은 150%가 웹에서는 1.5em, LibreOffice에서는 ≈2.04em이
// 되어 모든 글줄이 8~13px 아래로 밀렸다. 절대값(pt)으로 눕히면 두 엔진이 같은 줄 전진
// 폭을 쓴다.
func LineSpacing(multiplier, sizePx float64) int64 {
	return int64(math.Round(multiplier * sizePx * PointsPerPixel * 100))
}

// Offset은 지오메트리 패스가 잰 박스의 좌상단을 OOXML `a:off`(EMU)로 바꾼다.
//
// **단위 경계는 이 함수와 Extent·FontSize뿐이다** (PRD §13-5). 지오메트리 패스는 논리 px만
// 알고 EMU를 모른다 (units_test.go가 그것을 못 박는다). 경계를 한 곳에 두는 이유: 환산이
// 여러 곳에 흩어지면 어떤 값이 px이고 어떤 값이 EMU인지 타입으로 드러나지 않아, 두 번 환산된
// 좌표(도형이 화면 밖으로 나간다)나 환산되지 않은 좌표(도형이 왼쪽 위 점에 뭉친다)가 생긴다.
//
// 좌표는 슬라이드 상대이므로 그대로 `a:off`가 된다 — 슬라이드 원점이 두 좌표계에서 같다.
// 음수도 그대로 넘긴다: 스테이지를 넘어간 박스는 잘리지 않고 넘친다 (PRD §13-1).
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
