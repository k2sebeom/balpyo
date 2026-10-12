package snapshot

import (
	"bytes"
	"image"
	"image/color"
	"image/draw"
	"image/gif"
	"testing"
	"time"
)

// 판정은 픽셀 하나하나가 아니라 블록 평균의 오차다 — 디더링은 픽셀마다 오차를 흩지만 눈이 보는
// 것은 주변 평균이고, 줄무늬 · 엉뚱한 색 덩어리는 블록 평균에서 크게 벌어진다.
const block = 8

// textFrame은 흰 바탕에 안티에일리어스 회색 계조가 화면 대부분을 덮은 장면이다 — 글자가 많은
// 슬라이드가 팔레트를 회색으로 채우는 상황을 과장한 것이다.
func textFrame(w, h int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.Draw(img, img.Bounds(), image.NewUniform(color.White), image.Point{}, draw.Src)
	for y := 0; y < h*3/4; y++ {
		for x := 0; x < w; x++ {
			v := uint8((x + y*7) % 256)
			img.SetRGBA(x, y, color.RGBA{v, v, v, 255})
		}
	}
	return img
}

// gradientFrame은 보라에서 초록으로 가는 2차원 그러데이션이다 — 사진처럼 색이 많은 장면.
func gradientFrame(w, h int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			fx, fy := float64(x)/float64(w-1), float64(y)/float64(h-1)
			r := uint8(128*(1-fx) + 30*fx)
			g := uint8(40*(1-fx) + 200*fx)
			b := uint8(200*(1-fy) + 60*fy)
			img.SetRGBA(x, y, color.RGBA{r, g, b, 255})
		}
	}
	return img
}

// flatFrame은 평면색 몇 개만 쓴 장면이다 (도형 · 바탕).
func flatFrame(w, h int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	fills := []color.RGBA{{250, 248, 245, 255}, {37, 99, 235, 255}, {17, 24, 39, 255}, {220, 38, 38, 255}}
	for i, c := range fills {
		draw.Draw(img, image.Rect(0, i*h/len(fills), w, (i+1)*h/len(fills)), image.NewUniform(c), image.Point{}, draw.Src)
	}
	return img
}

func decodeFrames(t *testing.T, frames []*image.RGBA) []*image.Paletted {
	t.Helper()
	data, err := encodeGIF(frames, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	anim, err := gif.DecodeAll(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	if len(anim.Image) != len(frames) {
		t.Fatalf("%d GIF frames, want %d", len(anim.Image), len(frames))
	}
	return anim.Image
}

// blockError는 블록 평균끼리의 채널 차이 중 가장 큰 것이다.
func blockError(want *image.RGBA, got image.Image) int {
	worst := 0
	b := want.Bounds()
	for by := b.Min.Y; by < b.Max.Y; by += block {
		for bx := b.Min.X; bx < b.Max.X; bx += block {
			var sw, sg [3]int
			n := 0
			for y := by; y < by+block && y < b.Max.Y; y++ {
				for x := bx; x < bx+block && x < b.Max.X; x++ {
					c := want.RGBAAt(x, y)
					r, g, bl, _ := got.At(x, y).RGBA()
					sw[0], sw[1], sw[2] = sw[0]+int(c.R), sw[1]+int(c.G), sw[2]+int(c.B)
					sg[0], sg[1], sg[2] = sg[0]+int(r>>8), sg[1]+int(g>>8), sg[2]+int(bl>>8)
					n++
				}
			}
			for ch := 0; ch < 3; ch++ {
				d := (sw[ch] - sg[ch]) / n
				if d < 0 {
					d = -d
				}
				if d > worst {
					worst = d
				}
			}
		}
	}
	return worst
}

func TestGIFGradientSurvivesTextHeavyFrames(t *testing.T) {
	const w, h = 320, 180
	text, grad := textFrame(w, h), gradientFrame(w, h)
	got := decodeFrames(t, []*image.RGBA{text, text, text, grad})
	if e := blockError(grad, got[3]); e > 4 {
		t.Errorf("gradient frame: block average off by %d levels, want at most 4", e)
	}
	if e := blockError(text, got[0]); e > 4 {
		t.Errorf("text frame: block average off by %d levels, want at most 4", e)
	}
}

func TestGIFKeepsFlatColorsExact(t *testing.T) {
	const w, h = 320, 180
	flat := flatFrame(w, h)
	// 같은 GIF의 다른 프레임이 색을 많이 써도 평면색 프레임의 바탕은 흩어지지 않아야 한다.
	got := decodeFrames(t, []*image.RGBA{gradientFrame(w, h), flat})
	if e := blockError(flat, got[1]); e != 0 {
		t.Errorf("flat frame: block average off by %d levels, want exact", e)
	}
	// 평면색과 그러데이션이 한 프레임에 있어도 평면색 면은 정확히 남는다.
	mixed := gradientFrame(w, h)
	fill := color.RGBA{37, 99, 235, 255}
	draw.Draw(mixed, image.Rect(0, 0, w/2, h/2), image.NewUniform(fill), image.Point{}, draw.Src)
	got = decodeFrames(t, []*image.RGBA{mixed})
	for y := 0; y < h/2; y++ {
		for x := 0; x < w/2; x++ {
			if r, g, b, _ := got[0].At(x, y).RGBA(); uint8(r>>8) != fill.R || uint8(g>>8) != fill.G || uint8(b>>8) != fill.B {
				t.Fatalf("flat fill next to a gradient is dithered at (%d,%d)", x, y)
			}
		}
	}
	if e := blockError(mixed, got[0]); e > 4 {
		t.Errorf("mixed frame: block average off by %d levels, want at most 4", e)
	}
}

func TestGIFIsDeterministic(t *testing.T) {
	frames := []*image.RGBA{textFrame(160, 90), gradientFrame(160, 90)}
	a, err := encodeGIF(frames, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	b, err := encodeGIF(frames, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(a, b) {
		t.Error("two encodings of the same frames differ")
	}
}
