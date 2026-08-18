package testkit

import (
	"image"
	"image/color"
	"testing"

	"github.com/k2sebeom/balpyo/render/geom"
)

// inkProbe는 흰 배경에 검은 "줄" 두 개를 그린 합성 이미지다. 글자 사이의 세로 틈(같은 행 안의
// 배경 픽셀)이 띠를 쪼개지 않는지, 옅은 안티앨리어싱 행이 잉크로 세어지지 않는지 본다.
func inkProbe() *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, 200, 100))
	white := color.RGBA{R: 255, G: 255, B: 255, A: 255}
	black := color.RGBA{A: 255}
	for y := 0; y < 100; y++ {
		for x := 0; x < 200; x++ {
			img.SetRGBA(x, y, white)
		}
	}
	// 첫 줄: y 10..19, 글자 두 덩어리 (x 20..49, 70..119) — 사이 틈은 같은 줄이다.
	for y := 10; y < 20; y++ {
		for x := 20; x < 50; x++ {
			img.SetRGBA(x, y, black)
		}
		for x := 70; x < 120; x++ {
			img.SetRGBA(x, y, black)
		}
	}
	// 첫 줄 밑의 옅은 가장자리 행 (y=20): 배경과의 채널 차가 inkChannel 이하라 잉크가 아니다.
	for x := 20; x < 120; x++ {
		img.SetRGBA(x, 20, color.RGBA{R: 210, G: 210, B: 210, A: 255})
	}
	// 둘째 줄: y 40..54, x 20..179.
	for y := 40; y < 55; y++ {
		for x := 20; x < 180; x++ {
			img.SetRGBA(x, y, black)
		}
	}
	return img
}

func TestInkBands(t *testing.T) {
	bands := InkBands(inkProbe(), geom.Rect{X: 0, Y: 0, W: 200, H: 100})
	want := []geom.Rect{
		{X: 20, Y: 10, W: 100, H: 10},
		{X: 20, Y: 40, W: 160, H: 15},
	}
	if len(bands) != len(want) {
		t.Fatalf("띠 %d개, want %d개: %v", len(bands), len(want), bands)
	}
	for i, band := range bands {
		if band != want[i] {
			t.Errorf("띠 %d = %s, want %s", i, band, want[i])
		}
	}
}

// 박스를 좁혀 재면 박스 밖 잉크는 세지 않아야 한다 — 요소 박스 단위로 줄 수를 재는 계약.
func TestInkBandsClipsToRect(t *testing.T) {
	bands := InkBands(inkProbe(), geom.Rect{X: 0, Y: 0, W: 200, H: 30})
	if len(bands) != 1 {
		t.Fatalf("띠 %d개, want 1개: %v", len(bands), bands)
	}
	if want := (geom.Rect{X: 20, Y: 10, W: 100, H: 10}); bands[0] != want {
		t.Errorf("띠 = %s, want %s", bands[0], want)
	}
}

// 잉크가 없는 영역은 빈 결과다 (0개 줄 = 빈 텍스트).
func TestInkBandsEmptyRegion(t *testing.T) {
	if bands := InkBands(inkProbe(), geom.Rect{X: 130, Y: 0, W: 60, H: 30}); len(bands) != 0 {
		t.Errorf("빈 영역에서 띠 %d개: %v", len(bands), bands)
	}
}

// 이미지 밖으로 나간 박스는 겹치는 부분만 잰다 — 오버플로가 허용되는 정책(PRD §13-1)에서
// 박스가 스테이지를 넘어도 측정이 죽지 않아야 한다.
func TestInkBandsOutOfBounds(t *testing.T) {
	if bands := InkBands(inkProbe(), geom.Rect{X: 190, Y: 90, W: 50, H: 50}); len(bands) != 0 {
		t.Errorf("경계 밖 영역에서 띠 %d개: %v", len(bands), bands)
	}
	if bands := InkBands(inkProbe(), geom.Rect{X: 300, Y: 0, W: 50, H: 50}); bands != nil {
		t.Errorf("이미지 밖 영역은 nil이어야 한다: %v", bands)
	}
}
