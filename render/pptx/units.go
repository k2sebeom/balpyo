package pptx

import (
	"math"

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

// SlideSize는 논리 스테이지를 PPTX 페이지 크기(EMU)로 바꾼다.
//
// 16:9(1280×720)는 12192000×6858000 = 13.333×7.5in, 4:3(960×720)은
// 9144000×6858000 = 10×7.5in — 둘 다 PowerPoint 표준 페이지 크기와 정확히 일치한다.
func SlideSize(stage soml.Stage) (cx, cy int64) {
	return EMU(float64(stage.Width)), EMU(float64(stage.Height))
}
