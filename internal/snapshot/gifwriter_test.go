package snapshot

import (
	"bytes"
	"image"
	"image/color"
	"image/gif"
	"testing"
	"time"
)

func palettedFrame(w, h, colors, seed int) *image.Paletted {
	pal := make(color.Palette, colors)
	for i := range pal {
		pal[i] = color.RGBA{uint8(i * 7), uint8(255 - i), uint8(i*13 + seed), 255}
	}
	p := image.NewPaletted(image.Rect(0, 0, w, h), pal)
	for i := range p.Pix {
		p.Pix[i] = uint8((i*31 + seed) % colors)
	}
	return p
}

// 흘려 쓴 GIF를 표준 디코더가 그대로 읽는다: 프레임 수 · 지연 · 무한 반복 · 프레임마다의
// 팔레트와 색 번호. 색 수가 2의 거듭제곱이 아니거나 하나뿐인 팔레트, 255바이트 블록 경계를
// 넘는 프레임을 섞는다.
func TestGIFWriterRoundTrips(t *testing.T) {
	frames := []*image.Paletted{
		palettedFrame(97, 61, 256, 0),
		palettedFrame(97, 61, 3, 1),
		palettedFrame(97, 61, 1, 2),
		palettedFrame(97, 61, 130, 3),
	}
	var buf bytes.Buffer
	w := newGIFWriter(&buf)
	for _, f := range frames {
		if err := w.Add(f, 1500*time.Millisecond); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	anim, err := gif.DecodeAll(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	if anim.LoopCount != 0 {
		t.Errorf("loop count %d, want 0 (forever)", anim.LoopCount)
	}
	if anim.Config.Width != 97 || anim.Config.Height != 61 {
		t.Errorf("screen %dx%d, want 97x61", anim.Config.Width, anim.Config.Height)
	}
	if len(anim.Image) != len(frames) {
		t.Fatalf("%d frames, want %d", len(anim.Image), len(frames))
	}
	for i, want := range frames {
		got := anim.Image[i]
		if anim.Delay[i] != 150 {
			t.Errorf("frame %d: delay %d, want 150", i, anim.Delay[i])
		}
		if !bytes.Equal(got.Pix, want.Pix) {
			t.Errorf("frame %d: color indices differ", i)
		}
		for j, c := range want.Palette {
			if got.Palette[j] != c {
				t.Errorf("frame %d: palette[%d] = %v, want %v", i, j, got.Palette[j], c)
				break
			}
		}
	}
}

func TestGIFWriterRejectsBadFrames(t *testing.T) {
	var buf bytes.Buffer
	w := newGIFWriter(&buf)
	if err := w.Close(); err == nil {
		t.Error("want an error for a GIF without frames")
	}
	w = newGIFWriter(&buf)
	if err := w.Add(palettedFrame(10, 10, 4, 0), time.Second); err != nil {
		t.Fatal(err)
	}
	if err := w.Add(palettedFrame(11, 10, 4, 0), time.Second); err == nil {
		t.Error("want an error for a frame of another size")
	}
	w = newGIFWriter(&buf)
	bad := palettedFrame(10, 10, 4, 0)
	bad.Pix[5] = 9
	if err := w.Add(bad, time.Second); err == nil {
		t.Error("want an error for a color index past the palette")
	}
}
