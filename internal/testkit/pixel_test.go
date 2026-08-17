package testkit

import (
	"image"
	"image/color"
	"testing"
)

// fill은 단색 이미지를 만든다.
func fill(w, h int, c color.RGBA) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.SetRGBA(x, y, c)
		}
	}
	return img
}

var (
	white = color.RGBA{R: 255, G: 255, B: 255, A: 255}
	black = color.RGBA{A: 255}
)

func TestDiffCountsOnlyPixelsOverTolerance(t *testing.T) {
	want := fill(10, 10, white)
	got := fill(10, 10, white)
	// 허용값 안에서 살짝 어긋난 픽셀 하나 — 안티앨리어싱이 내는 종류의 차이다.
	got.SetRGBA(0, 0, color.RGBA{R: 250, G: 250, B: 250, A: 255})
	// 허용값을 넘는 픽셀 하나.
	got.SetRGBA(1, 0, black)

	result, err := Diff(want, got, DiffOptions{Channel: 32, MaxRatio: 0.5})
	if err != nil {
		t.Fatal(err)
	}
	if result.Changed != 1 {
		t.Errorf("다른 픽셀 = %d, want 1", result.Changed)
	}
	if result.Total != 100 || result.Ratio != 0.01 {
		t.Errorf("Total = %d, Ratio = %v", result.Total, result.Ratio)
	}
	if result.MaxChannel != 255 {
		t.Errorf("MaxChannel = %d, want 255", result.MaxChannel)
	}
	// diff 이미지는 다른 픽셀만 마젠타로 칠한다.
	if got := result.Image.RGBAAt(1, 0); got != (color.RGBA{R: 255, B: 255, A: 255}) {
		t.Errorf("diff 이미지의 다른 픽셀 = %+v", got)
	}
	if result.Image.RGBAAt(5, 5) != (color.RGBA{R: 255, G: 255, B: 255, A: 255}) {
		t.Errorf("같은 흰 픽셀은 흰색으로 남아야 한다: %+v", result.Image.RGBAAt(5, 5))
	}
}

func TestDiffRejectsSizeMismatch(t *testing.T) {
	// 크기가 다르면 리샘플하지 않고 실패한다 — 크기를 맞추는 것은 렌더 단계의 책임이고,
	// 여기서 늘리면 "PPTX 페이지 크기가 틀렸다"가 오차 안에 숨는다.
	if _, err := Diff(fill(10, 10, white), fill(10, 9, white), DefaultDiffOptions()); err == nil {
		t.Fatal("크기 불일치는 에러여야 한다")
	}
}

func TestDiffCompositesTransparentOverWhite(t *testing.T) {
	// PPTX 렌더는 배경이 흰색이고 웹 스크린샷은 투명일 수 있다. 둘을 다르다고 세면
	// 모든 빈 슬라이드가 실패한다.
	transparent := image.NewRGBA(image.Rect(0, 0, 4, 4))
	result, err := Diff(fill(4, 4, white), transparent, DiffOptions{Channel: 0, MaxRatio: 0})
	if err != nil {
		t.Fatal(err)
	}
	if result.Changed != 0 {
		t.Errorf("투명 픽셀이 흰 배경과 다르게 세어졌다: %s", result)
	}
}

func TestPNGRoundTrip(t *testing.T) {
	img := fill(3, 2, color.RGBA{R: 10, G: 20, B: 30, A: 255})
	decoded, err := DecodePNG(EncodePNG(t, img))
	if err != nil {
		t.Fatal(err)
	}
	if decoded.Bounds() != img.Bounds() {
		t.Fatalf("Bounds = %v, want %v", decoded.Bounds(), img.Bounds())
	}
	result, err := Diff(img, decoded, DiffOptions{Channel: 0, MaxRatio: 0})
	if err != nil {
		t.Fatal(err)
	}
	if result.Changed != 0 {
		t.Errorf("PNG 왕복에서 픽셀이 변했다: %s", result)
	}
}
