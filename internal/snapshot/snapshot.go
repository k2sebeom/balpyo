// Package snapshot은 `balpyo snap`이다 — 내장 렌더러(render/quick)로 슬라이드를 PNG · GIF로 그린다.
//
// 브라우저도 OnlyOffice도 띄우지 않는다. 쓰는 동안 결과를 빨리 보는 것이 목적이라 기준은
// 여전히 브라우저 측정이다: 배치는 브라우저와 박스 단위로 견준 엔진이지만 줄바꿈 · 글리프까지
// 같다고 약속하지 않는다.
package snapshot

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/gif"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/k2sebeom/balpyo/internal/pipeline"
	"github.com/k2sebeom/balpyo/render/geom"
	"github.com/k2sebeom/balpyo/render/quick"
	"github.com/k2sebeom/balpyo/soml"
)

// DirName은 출력 디렉토리 아래 그림이 떨어지는 디렉토리 이름이다.
const DirName = "snap"

// GIFName은 덱 전체를 담은 GIF 파일 이름이다.
const GIFName = "deck.gif"

// Options는 실행의 입력이다.
type Options struct {
	Build pipeline.Options
	// Dir은 그림을 쓸 디렉토리다 (비면 <출력 디렉토리>/snap).
	Dir string
	// Slides는 그릴 슬라이드 번호다 (1부터, 비면 전부).
	Slides []int
	// Scale은 장치 픽셀 배율이다 (0이면 1).
	Scale float64
	// GIF면 덱 전체를 GIF 하나로도 쓴다 (reveal 스텝마다 한 프레임).
	GIF bool
	// Outline이면 잰 박스와 요소 식별자를 덧그린다.
	Outline bool
	// FrameDelay는 GIF 프레임 하나의 길이다 (0이면 1.5초).
	FrameDelay time.Duration
}

// Result는 실행의 산출이다.
type Result struct {
	Deck        *soml.Deck
	Diagnostics soml.Diagnostics
	Dir         string
	Written     []string
	Elapsed     time.Duration
}

// Run은 덱을 읽어 그리고 파일을 쓴다.
func Run(opts Options) (*Result, error) {
	start := time.Now()
	deck, diags, err := pipeline.LoadDeck(opts.Build)
	res := &Result{Deck: deck, Diagnostics: diags}
	if err != nil {
		return res, err
	}
	if err := diags.Err(); err != nil {
		return res, err
	}
	only := map[int]bool{}
	for _, n := range opts.Slides {
		if n < 1 || n > len(deck.Slides) {
			return res, fmt.Errorf("slide %d does not exist (the deck has %d)", n, len(deck.Slides))
		}
		only[deck.Slides[n-1].Index] = true
	}
	drawn, err := quick.Snapshot(deck, quick.Options{DPR: opts.Scale, Only: only, Steps: opts.GIF, Outline: opts.Outline})
	if err != nil {
		return res, err
	}
	// 내장 엔진이 잰 박스로도 배치 진단을 낸다 — 빌드의 --check-layout과 같은 판정이다.
	layout := geom.Diagnose(deck, drawn.Geometry)
	if opts.Build.Strict {
		layout = layout.Strict()
	}
	res.Diagnostics = append(res.Diagnostics, layout...)

	dir := opts.Dir
	if dir == "" {
		out, err := pipeline.OutDirPath(opts.Build)
		if err != nil {
			return res, err
		}
		dir = filepath.Join(out, DirName)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return res, err
	}
	res.Dir = dir
	for i, img := range drawn.Slides {
		if img == nil {
			continue
		}
		data, err := quick.EncodePNG(img)
		if err != nil {
			return res, err
		}
		name := fmt.Sprintf("%03d.png", i+1)
		if err := os.WriteFile(filepath.Join(dir, name), data, 0o644); err != nil {
			return res, err
		}
		res.Written = append(res.Written, name)
	}
	if opts.GIF {
		delay := opts.FrameDelay
		if delay <= 0 {
			delay = 1500 * time.Millisecond
		}
		var frames []*image.RGBA
		for _, steps := range drawn.Steps {
			frames = append(frames, steps...)
		}
		data, err := encodeGIF(frames, delay)
		if err != nil {
			return res, err
		}
		if err := os.WriteFile(filepath.Join(dir, GIFName), data, 0o644); err != nil {
			return res, err
		}
		res.Written = append(res.Written, GIFName)
	}
	res.Elapsed = time.Since(start)
	return res, layout.Err()
}

// encodeGIF는 프레임을 한 팔레트의 GIF로 쓴다.
func encodeGIF(frames []*image.RGBA, delay time.Duration) ([]byte, error) {
	var nonNil []*image.RGBA
	for _, f := range frames {
		if f != nil {
			nonNil = append(nonNil, f)
		}
	}
	if len(nonNil) == 0 {
		return nil, fmt.Errorf("no slide to put in the GIF")
	}
	pal := buildPalette(nonNil)
	// 채널당 5비트 색마다 가장 가까운 팔레트 색을 한 번만 찾는다 — 픽셀마다 팔레트를 훑으면
	// 1280×720 프레임 하나에 수억 번의 비교가 든다.
	lut := make([]uint8, 1<<15)
	for key := range lut {
		r, g, b := uint8(key>>10<<3|4), uint8(key>>5&31<<3|4), uint8(key&31<<3|4)
		lut[key] = uint8(pal.Index(color.RGBA{r, g, b, 255}))
	}
	anim := &gif.GIF{LoopCount: 0}
	for _, f := range nonNil {
		p := image.NewPaletted(f.Bounds(), pal)
		w, h := f.Bounds().Dx(), f.Bounds().Dy()
		for y := 0; y < h; y++ {
			src := f.Pix[y*f.Stride:]
			dst := p.Pix[y*p.Stride:]
			for x := 0; x < w; x++ {
				r, g, b := src[x*4], src[x*4+1], src[x*4+2]
				dst[x] = lut[uint16(r>>3)<<10|uint16(g>>3)<<5|uint16(b>>3)]
			}
		}
		anim.Image = append(anim.Image, p)
		anim.Delay = append(anim.Delay, int(delay/(10*time.Millisecond)))
	}
	var buf bytes.Buffer
	if err := gif.EncodeAll(&buf, anim); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// buildPalette는 프레임에서 가장 자주 쓰인 색 256개를 고른다 (채널당 5비트로 묶어 센다).
// 슬라이드는 대부분 평면색이라 빈도 순이 중앙값 분할보다 글자 · 면을 더 정확히 남긴다.
func buildPalette(frames []*image.RGBA) color.Palette {
	type bucket struct {
		n       int
		r, g, b int
	}
	counts := map[uint16]*bucket{}
	for _, f := range frames {
		for i := 0; i+3 < len(f.Pix); i += 4 * 3 { // 세 픽셀에 하나씩 본다
			r, g, b := int(f.Pix[i]), int(f.Pix[i+1]), int(f.Pix[i+2])
			key := uint16(r>>3)<<10 | uint16(g>>3)<<5 | uint16(b>>3)
			bk := counts[key]
			if bk == nil {
				bk = &bucket{}
				counts[key] = bk
			}
			bk.n++
			bk.r += r
			bk.g += g
			bk.b += b
		}
	}
	keys := make([]uint16, 0, len(counts))
	for k := range counts {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		if counts[keys[i]].n != counts[keys[j]].n {
			return counts[keys[i]].n > counts[keys[j]].n
		}
		return keys[i] < keys[j]
	})
	pal := color.Palette{}
	for _, k := range keys {
		if len(pal) >= 256 {
			break
		}
		bk := counts[k]
		pal = append(pal, color.RGBA{uint8(bk.r / bk.n), uint8(bk.g / bk.n), uint8(bk.b / bk.n), 255})
	}
	if len(pal) == 0 {
		pal = append(pal, color.RGBA{255, 255, 255, 255})
	}
	return pal
}
