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
	"os"
	"path/filepath"
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

	// 그림은 그리는 대로 파일로 쓰고 놓는다 — 덱 전체를 쥐면 메모리가 슬라이드 수와 배율의
	// 제곱에 비례해 자란다. GIF는 임시 파일에 프레임을 이어 쓰고 끝까지 성공했을 때만 이름을 바꾼다.
	var anim *gifWriter
	var animFile *os.File
	gifPath := filepath.Join(dir, GIFName)
	if opts.GIF {
		f, err := os.CreateTemp(dir, GIFName+".*.tmp")
		if err != nil {
			return res, err
		}
		animFile = f
		defer func() {
			if animFile != nil {
				animFile.Close()
				os.Remove(animFile.Name())
			}
		}()
		anim = newGIFWriter(f)
	}
	delay := opts.FrameDelay
	if delay <= 0 {
		delay = 1500 * time.Millisecond
	}
	emit := func(f quick.Frame) error {
		if anim != nil {
			if err := anim.Add(quantize(f.Image), delay); err != nil {
				return err
			}
		}
		if f.Step >= 0 {
			return nil
		}
		data, err := quick.EncodePNG(f.Image)
		if err != nil {
			return err
		}
		name := fmt.Sprintf("%03d.png", f.Slide+1)
		if err := os.WriteFile(filepath.Join(dir, name), data, 0o644); err != nil {
			return err
		}
		res.Written = append(res.Written, name)
		return nil
	}
	drawn, err := quick.Snapshot(deck, quick.Options{DPR: opts.Scale, Only: only, Steps: opts.GIF, Outline: opts.Outline, Emit: emit})
	if err != nil {
		return res, err
	}
	if anim != nil {
		if err := anim.Close(); err != nil {
			return res, err
		}
		if err := animFile.Close(); err != nil {
			return res, err
		}
		if err := os.Chmod(animFile.Name(), 0o644); err != nil {
			return res, err
		}
		if err := os.Rename(animFile.Name(), gifPath); err != nil {
			return res, err
		}
		animFile = nil
		res.Written = append(res.Written, GIFName)
	}
	// 내장 엔진이 잰 박스로도 배치 진단을 낸다 — 빌드의 --check-layout과 같은 판정이다.
	layout := geom.Diagnose(deck, drawn.Geometry)
	if opts.Build.Strict {
		layout = layout.Strict()
	}
	res.Diagnostics = append(res.Diagnostics, layout...)
	res.Elapsed = time.Since(start)
	return res, layout.Err()
}

// encodeGIF는 프레임마다 그 프레임의 팔레트로 GIF를 메모리에 쓴다.
func encodeGIF(frames []*image.RGBA, delay time.Duration) ([]byte, error) {
	var buf bytes.Buffer
	w := newGIFWriter(&buf)
	for _, f := range frames {
		if f == nil {
			continue
		}
		if err := w.Add(quantize(f), delay); err != nil {
			return nil, err
		}
	}
	if err := w.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
