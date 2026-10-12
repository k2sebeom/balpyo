package soml

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
	"image/jpeg"
	"testing"
)

// withOrientation은 JPEG 바이트의 SOI 바로 뒤에 방향 태그 하나짜리 EXIF APP1을 끼운다.
func withOrientation(t *testing.T, order binary.ByteOrder, o uint16) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, image.NewGray(image.Rect(0, 0, 4, 2)), nil); err != nil {
		t.Fatal(err)
	}
	tiff := make([]byte, 26)
	if order == binary.LittleEndian {
		copy(tiff, "II")
	} else {
		copy(tiff, "MM")
	}
	order.PutUint16(tiff[2:], 42)
	order.PutUint32(tiff[4:], 8)
	order.PutUint16(tiff[8:], 1)
	order.PutUint16(tiff[10:], exifOrientationTag)
	order.PutUint16(tiff[12:], 3)
	order.PutUint32(tiff[14:], 1)
	order.PutUint16(tiff[18:], o)
	payload := append([]byte("Exif\x00\x00"), tiff...)
	segment := []byte{0xFF, 0xE1, 0, 0}
	binary.BigEndian.PutUint16(segment[2:], uint16(len(payload)+2))
	raw := buf.Bytes()
	out := append([]byte{}, raw[:2]...)
	out = append(out, segment...)
	out = append(out, payload...)
	return append(out, raw[2:]...)
}

// 방향 태그는 두 바이트 순서 모두에서 읽힌다. 태그가 없거나 값이 어휘 밖이면 0이다 — 브라우저도
// 그런 태그는 무시하고 저장된 그대로 그린다.
func TestJPEGOrientation(t *testing.T) {
	for _, order := range []binary.ByteOrder{binary.BigEndian, binary.LittleEndian} {
		for o := uint16(1); o <= 8; o++ {
			if got := JPEGOrientation(withOrientation(t, order, o)); got != Orientation(o) {
				t.Errorf("%v 방향 %d → %d", order, o, got)
			}
		}
		if got := JPEGOrientation(withOrientation(t, order, 9)); got != 0 {
			t.Errorf("%v 어휘 밖 방향 9 → %d, want 0", order, got)
		}
	}
	var plain bytes.Buffer
	if err := jpeg.Encode(&plain, image.NewGray(image.Rect(0, 0, 4, 2)), nil); err != nil {
		t.Fatal(err)
	}
	for name, data := range map[string][]byte{
		"EXIF 없음":  plain.Bytes(),
		"JPEG가 아님": []byte("\x89PNG\r\n\x1a\n"),
		"잘린 세그먼트":  {0xFF, 0xD8, 0xFF, 0xE1, 0x10, 0x00, 'E'},
		"빈 바이트":    nil,
	} {
		if got := JPEGOrientation(data); got != 0 {
			t.Errorf("%s → %d, want 0", name, got)
		}
	}
}

// 변환 표 — 대칭 다음 회전으로 적은 여덟 방향과, 가로·세로가 바뀌는 네 방향.
func TestOrientationTransform(t *testing.T) {
	tests := []struct {
		o      Orientation
		rotate int
		flipH  bool
		swaps  bool
	}{
		{0, 0, false, false},
		{1, 0, false, false},
		{2, 0, true, false},
		{3, 180, false, false},
		{4, 180, true, false},
		{5, 270, true, true},
		{6, 90, false, true},
		{7, 90, true, true},
		{8, 270, false, true},
		{42, 0, false, false},
	}
	for _, tc := range tests {
		if tc.o.Rotation() != tc.rotate || tc.o.FlipH() != tc.flipH || tc.o.SwapsAxes() != tc.swaps {
			t.Errorf("방향 %d = (%d°, flipH %v, swaps %v), want (%d°, %v, %v)",
				tc.o, tc.o.Rotation(), tc.o.FlipH(), tc.o.SwapsAxes(), tc.rotate, tc.flipH, tc.swaps)
		}
	}
}

// EXIF 방향은 픽셀을 세운다 — 3×1 그림(빨강·초록·파랑)이 방향마다 놓이는 자리.
func TestOrientationUpright(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 3, 1))
	r, g, b := color.RGBA{R: 255, A: 255}, color.RGBA{G: 255, A: 255}, color.RGBA{B: 255, A: 255}
	src.Set(0, 0, r)
	src.Set(1, 0, g)
	src.Set(2, 0, b)
	tests := []struct {
		o    Orientation
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
		got := tc.o.Upright(src)
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
