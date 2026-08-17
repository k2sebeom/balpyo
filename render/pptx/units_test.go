package pptx

import (
	"testing"

	"github.com/k2sebeom/balpyo/soml"
)

func TestEMU(t *testing.T) {
	tests := []struct {
		px   float64
		want int64
	}{
		{0, 0},
		{1, 9525},
		{96, EMUPerInch}, // 96px = 1인치
		{1280, 12192000},
		{0.5, 4763}, // 반올림
		{-2, -19050},
	}
	for _, tc := range tests {
		if got := EMU(tc.px); got != tc.want {
			t.Errorf("EMU(%v) = %d, want %d", tc.px, got, tc.want)
		}
	}
}

func TestFontSize(t *testing.T) {
	// OOXML sz는 1/100 pt 단위이고 pt = px × 0.75다.
	tests := []struct {
		px   float64
		want int64
	}{
		{16, 1200}, // 16px = 12pt
		{64, 4800},
		{14, 1050},
	}
	for _, tc := range tests {
		if got := FontSize(tc.px); got != tc.want {
			t.Errorf("FontSize(%v) = %d, want %d", tc.px, got, tc.want)
		}
	}
}

// 논리 스테이지 → PPTX 페이지 크기가 PowerPoint 표준과 정확히 일치해야 한다.
// 어긋나면 산출물을 기존 덱에 붙일 때 PowerPoint가 슬라이드 크기 변환을 묻는다.
func TestSlideSizeMatchesPowerPointStandards(t *testing.T) {
	tests := []struct {
		aspect     soml.Aspect
		wantCX     int64
		wantCY     int64
		wantInches string
	}{
		{soml.Aspect16x9, 12192000, 6858000, "13.333×7.5in"},
		{soml.Aspect4x3, 9144000, 6858000, "10×7.5in"},
	}
	for _, tc := range tests {
		stage, err := tc.aspect.Stage()
		if err != nil {
			t.Fatal(err)
		}
		cx, cy := SlideSize(stage)
		if cx != tc.wantCX || cy != tc.wantCY {
			t.Errorf("%s(%dx%dpx) = %dx%d EMU, want %dx%d (%s)",
				tc.aspect, stage.Width, stage.Height, cx, cy, tc.wantCX, tc.wantCY, tc.wantInches)
		}
	}
}
