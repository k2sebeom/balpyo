package snapshot

import (
	"image"
	"image/color"
	"sort"
)

// 프레임 하나를 256색으로 줄인다. 팔레트는 프레임마다 따로 만든다 — GIF는 프레임마다 지역
// 색표를 둘 수 있고, 덱 전체에 한 팔레트를 쓰면 글자가 많은 장의 회색 계조가 자리를 다
// 차지해 사진 · 그러데이션이 있는 장이 엉뚱한 색 덩어리로 무너진다.
//
// 팔레트는 두 부분이다.
//   - 평면색: 화면의 일정 비율 이상을 덮는 정확한 색 (바탕 · 도형 면 · 글자색). 그대로 싣고,
//     그 색의 픽셀은 디더링하지 않는다 — 면이 점으로 흩어지면 슬라이드가 지저분해 보인다.
//   - 나머지: 남은 색을 채널당 5비트로 묶은 히스토그램을 중앙값 분할로 나눈 대표색.
//
// 서로 다른 색이 256개 이하면 모든 색을 그대로 싣는다 (근사 없음). 그보다 많으면 평면색이 아닌
// 픽셀을 Floyd–Steinberg로 디더링해 줄무늬 대신 고른 입자로 계조를 남긴다.

const (
	paletteSize = 256
	// exactLimit을 넘는 서로 다른 색은 더 세지 않는다 — 사진이 든 큰 프레임에서 맵이 수백만
	// 항목으로 자라지 않게 한다. 평면색은 넓은 면이라 그 전에 이미 세어진다.
	exactLimit = 1 << 16
	// flatShare는 평면색으로 칠 최소 비율의 역수다 (픽셀 1024개에 하나 이상).
	flatShare = 1024
	// flatMax는 평면색에 내줄 최대 자리다 — 나머지는 계조 몫으로 남긴다.
	flatMax = 128
)

type bucket struct {
	n       int
	r, g, b int
}

func bucketKey(r, g, b uint8) uint16 {
	return uint16(r>>3)<<10 | uint16(g>>3)<<5 | uint16(b>>3)
}

func packRGB(r, g, b uint8) uint32 { return uint32(r)<<16 | uint32(g)<<8 | uint32(b) }

// quantize는 프레임을 그 프레임만의 팔레트로 옮긴다.
func quantize(f *image.RGBA) *image.Paletted {
	w, h := f.Bounds().Dx(), f.Bounds().Dy()
	hist := make([]bucket, 1<<15)
	exact := map[uint32]int{}
	overflow := false
	for y := 0; y < h; y++ {
		row := f.Pix[y*f.Stride : y*f.Stride+w*4]
		last, run := uint32(1<<31), 0
		flush := func() {
			if run == 0 {
				return
			}
			if _, ok := exact[last]; ok || len(exact) < exactLimit {
				exact[last] += run
			} else {
				overflow = true
			}
		}
		for x := 0; x < w; x++ {
			r, g, b := row[x*4], row[x*4+1], row[x*4+2]
			bk := &hist[bucketKey(r, g, b)]
			bk.n++
			bk.r += int(r)
			bk.g += int(g)
			bk.b += int(b)
			if c := packRGB(r, g, b); c != last {
				flush()
				last, run = c, 0
			}
			run++
		}
		flush()
	}

	colors := make([]uint32, 0, len(exact))
	for c := range exact {
		colors = append(colors, c)
	}
	sort.Slice(colors, func(i, j int) bool {
		if exact[colors[i]] != exact[colors[j]] {
			return exact[colors[i]] > exact[colors[j]]
		}
		return colors[i] < colors[j]
	})
	if !overflow && len(colors) <= paletteSize {
		return mapExact(f, colors)
	}

	var flat []uint32
	for _, c := range colors {
		if len(flat) >= flatMax || exact[c]*flatShare < w*h {
			break
		}
		flat = append(flat, c)
		// 평면색 픽셀은 따로 실리므로 중앙값 분할의 몫에서 뺀다.
		r, g, b := uint8(c>>16), uint8(c>>8), uint8(c)
		n := exact[c]
		bk := &hist[bucketKey(r, g, b)]
		bk.n -= n
		bk.r -= n * int(r)
		bk.g -= n * int(g)
		bk.b -= n * int(b)
	}
	pal := make(color.Palette, 0, paletteSize)
	for _, c := range flat {
		pal = append(pal, color.RGBA{uint8(c >> 16), uint8(c >> 8), uint8(c), 255})
	}
	pal = append(pal, medianCut(hist, paletteSize-len(flat))...)
	return dither(f, pal, len(flat))
}

// mapExact는 서로 다른 색이 팔레트에 다 들어갈 때 색을 그대로 옮긴다.
func mapExact(f *image.RGBA, colors []uint32) *image.Paletted {
	pal := make(color.Palette, 0, len(colors))
	index := make(map[uint32]uint8, len(colors))
	for i, c := range colors {
		pal = append(pal, color.RGBA{uint8(c >> 16), uint8(c >> 8), uint8(c), 255})
		index[c] = uint8(i)
	}
	if len(pal) == 0 {
		pal = append(pal, color.RGBA{255, 255, 255, 255})
	}
	p := image.NewPaletted(image.Rect(0, 0, f.Bounds().Dx(), f.Bounds().Dy()), pal)
	w, h := p.Rect.Dx(), p.Rect.Dy()
	for y := 0; y < h; y++ {
		src := f.Pix[y*f.Stride:]
		dst := p.Pix[y*p.Stride:]
		for x := 0; x < w; x++ {
			dst[x] = index[packRGB(src[x*4], src[x*4+1], src[x*4+2])]
		}
	}
	return p
}

// cutBox는 중앙값 분할의 상자 하나다 — 히스토그램 칸 목록과 그 범위.
type cutBox struct {
	keys  []uint16
	n     int
	lo    [3]int
	hi    [3]int
	split bool // 칸이 둘 이상이라 더 나눌 수 있다
}

func (c *cutBox) longest() (axis, span int) {
	for a := 0; a < 3; a++ {
		if s := c.hi[a] - c.lo[a]; s > span {
			axis, span = a, s
		}
	}
	return axis, span
}

// channel은 칸의 평균색 중 한 채널이다.
func channel(bk *bucket, axis int) int {
	switch axis {
	case 0:
		return bk.r / bk.n
	case 1:
		return bk.g / bk.n
	}
	return bk.b / bk.n
}

func newBox(hist []bucket, keys []uint16) *cutBox {
	c := &cutBox{keys: keys, lo: [3]int{255, 255, 255}, split: len(keys) > 1}
	for _, k := range keys {
		bk := &hist[k]
		c.n += bk.n
		for a := 0; a < 3; a++ {
			v := channel(bk, a)
			c.lo[a] = min(c.lo[a], v)
			c.hi[a] = max(c.hi[a], v)
		}
	}
	return c
}

// medianCut은 히스토그램을 많아야 n개의 상자로 나눠 상자마다 평균색을 낸다. 나눌 상자는 픽셀
// 수 × 가장 긴 변이 가장 큰 것이다 — 넓게 퍼진 계조와 많이 쓰인 영역이 함께 자리를 얻는다.
func medianCut(hist []bucket, n int) color.Palette {
	var keys []uint16
	for k := range hist {
		if hist[k].n > 0 {
			keys = append(keys, uint16(k))
		}
	}
	if len(keys) == 0 || n <= 0 {
		return nil
	}
	boxes := []*cutBox{newBox(hist, keys)}
	for len(boxes) < n {
		best, score := -1, 0
		for i, c := range boxes {
			if !c.split {
				continue
			}
			_, span := c.longest()
			if s := c.n * span; s > score || best < 0 {
				best, score = i, s
			}
		}
		if best < 0 {
			break
		}
		c := boxes[best]
		axis, _ := c.longest()
		sort.Slice(c.keys, func(i, j int) bool {
			vi, vj := channel(&hist[c.keys[i]], axis), channel(&hist[c.keys[j]], axis)
			if vi != vj {
				return vi < vj
			}
			return c.keys[i] < c.keys[j]
		})
		// 누적 픽셀 수가 절반에 이르는 자리에서 자른다 (양쪽에 칸이 적어도 하나).
		cut, acc := 1, 0
		for i := 0; i < len(c.keys)-1; i++ {
			acc += hist[c.keys[i]].n
			cut = i + 1
			if acc*2 >= c.n {
				break
			}
		}
		left := append([]uint16(nil), c.keys[:cut]...)
		right := append([]uint16(nil), c.keys[cut:]...)
		boxes[best] = newBox(hist, left)
		boxes = append(boxes, newBox(hist, right))
	}
	pal := make(color.Palette, 0, len(boxes))
	for _, c := range boxes {
		var r, g, b int
		for _, k := range c.keys {
			r += hist[k].r
			g += hist[k].g
			b += hist[k].b
		}
		pal = append(pal, color.RGBA{uint8(r / c.n), uint8(g / c.n), uint8(b / c.n), 255})
	}
	return pal
}

// dither는 프레임을 팔레트로 옮기며 Floyd–Steinberg로 오차를 흩는다. 팔레트 앞의 flat개는
// 평면색이고, 원래 색이 그 색인 픽셀은 오차를 받지도 넘기지도 않고 그 색으로 남는다.
func dither(f *image.RGBA, pal color.Palette, flat int) *image.Paletted {
	type rgb struct{ r, g, b int32 }
	entries := make([]rgb, len(pal))
	for i, c := range pal {
		cc := c.(color.RGBA)
		entries[i] = rgb{int32(cc.R), int32(cc.G), int32(cc.B)}
	}
	flatIndex := make(map[uint32]uint8, flat)
	for i := 0; i < flat; i++ {
		e := entries[i]
		flatIndex[packRGB(uint8(e.r), uint8(e.g), uint8(e.b))] = uint8(i)
	}
	// 채널당 5비트 색마다 가장 가까운 팔레트 색을 한 번만 찾는다 — 픽셀마다 팔레트를 훑으면
	// 큰 프레임 하나에 수억 번의 비교가 든다. 칸 안의 어긋남은 디더링이 다음 픽셀로 넘긴다.
	lut := make([]uint8, 1<<15)
	for key := range lut {
		r, g, b := int32(key>>10<<3|4), int32(key>>5&31<<3|4), int32(key&31<<3|4)
		best, bestD := 0, int32(1<<30)
		for i, e := range entries {
			dr, dg, db := r-e.r, g-e.g, b-e.b
			if d := dr*dr + dg*dg + db*db; d < bestD {
				best, bestD = i, d
			}
		}
		lut[key] = uint8(best)
	}

	w, h := f.Bounds().Dx(), f.Bounds().Dy()
	p := image.NewPaletted(image.Rect(0, 0, w, h), pal)
	// 오차는 16배로 들고 다닌다 (Floyd–Steinberg 가중치의 분모).
	cur, next := make([]rgb, w+2), make([]rgb, w+2)
	clamp := func(v int32) int32 { return min(max(v, 0), 255) }
	for y := 0; y < h; y++ {
		src := f.Pix[y*f.Stride:]
		dst := p.Pix[y*p.Stride:]
		lastC, lastFlat, lastOK := uint32(1<<31), uint8(0), false
		for x := 0; x < w; x++ {
			sr, sg, sb := src[x*4], src[x*4+1], src[x*4+2]
			if c := packRGB(sr, sg, sb); c != lastC {
				lastC = c
				lastFlat, lastOK = flatIndex[c]
			}
			if lastOK {
				dst[x] = lastFlat
				continue
			}
			e := cur[x+1]
			r := clamp(int32(sr) + e.r/16)
			g := clamp(int32(sg) + e.g/16)
			b := clamp(int32(sb) + e.b/16)
			idx := lut[bucketKey(uint8(r), uint8(g), uint8(b))]
			dst[x] = idx
			q := entries[idx]
			er, eg, eb := r-q.r, g-q.g, b-q.b
			cur[x+2].r += er * 7
			cur[x+2].g += eg * 7
			cur[x+2].b += eb * 7
			next[x].r += er * 3
			next[x].g += eg * 3
			next[x].b += eb * 3
			next[x+1].r += er * 5
			next[x+1].g += eg * 5
			next[x+1].b += eb * 5
			next[x+2].r += er
			next[x+2].g += eg
			next[x+2].b += eb
		}
		cur, next = next, cur
		clear(next)
	}
	return p
}
