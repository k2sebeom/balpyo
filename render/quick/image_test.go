package quick

import (
	"image"
	"image/color"
	"testing"

	"github.com/k2sebeom/balpyo/soml"
)

// EXIF 방향은 픽셀을 세운다 — 3×1 그림(빨강·초록·파랑)이 방향마다 놓이는 자리.
func TestOrientImage(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 3, 1))
	r, g, b := color.RGBA{R: 255, A: 255}, color.RGBA{G: 255, A: 255}, color.RGBA{B: 255, A: 255}
	src.Set(0, 0, r)
	src.Set(1, 0, g)
	src.Set(2, 0, b)
	tests := []struct {
		o    soml.Orientation
		w, h int
		want []color.RGBA // 행 우선
	}{
		{1, 3, 1, []color.RGBA{r, g, b}},
		{2, 3, 1, []color.RGBA{b, g, r}},
		{3, 3, 1, []color.RGBA{b, g, r}},
		{6, 1, 3, []color.RGBA{r, g, b}},
		{8, 1, 3, []color.RGBA{b, g, r}},
		{5, 1, 3, []color.RGBA{r, g, b}},
		{7, 1, 3, []color.RGBA{b, g, r}},
	}
	for _, tc := range tests {
		got := orientImage(src, tc.o)
		if size := got.Bounds().Size(); size.X != tc.w || size.Y != tc.h {
			t.Errorf("방향 %d: 크기 %v, want %d×%d", tc.o, size, tc.w, tc.h)
			continue
		}
		for i, want := range tc.want {
			x, y := i%tc.w, i/tc.w
			if c := color.RGBAModel.Convert(got.At(x, y)).(color.RGBA); c != want {
				t.Errorf("방향 %d: (%d,%d) = %v, want %v", tc.o, x, y, c, want)
			}
		}
	}
}
