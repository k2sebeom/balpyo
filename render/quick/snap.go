package quick

import (
	"bytes"
	"fmt"
	"image"
	"image/png"
	pathpkg "path"

	"github.com/k2sebeom/balpyo/render"
	"github.com/k2sebeom/balpyo/render/geom"
	"github.com/k2sebeom/balpyo/render/web"
	"github.com/k2sebeom/balpyo/soml"
)

// Options는 그리기 선택이다.
type Options struct {
	// DPR은 논리 px 하나에 드는 장치 픽셀 수다 (0이면 1).
	DPR float64
	// Only가 비어 있지 않으면 그 슬라이드 인덱스만 그린다.
	Only map[int]bool
	// Steps면 reveal 스텝마다 장면을 따로 그린다 (Result.Steps).
	Steps bool
	// Outline이면 잰 박스와 요소 식별자를 덧그린다.
	Outline bool
	// Emit이 있으면 그림을 Result에 모으지 않고 그리는 대로 넘긴다 (Result.Slides · Steps는 빈다).
	// 덱 전체의 그림을 쥐고 있으면 메모리가 슬라이드 수와 배율의 제곱에 비례해 자라므로, 파일로
	// 쓰는 쪽은 받는 대로 쓰고 놓는다. 에러를 돌려주면 거기서 멈추고 그 에러를 낸다.
	Emit func(Frame) error
}

// Frame은 Emit이 받는 그림 하나다. 슬라이드마다 Steps면 스텝 0 … n-1이 먼저, 모든 스텝이
// 펼쳐진 장면(Step < 0)이 마지막에 온다 — GIF 프레임 순서 그대로다.
type Frame struct {
	// Slide는 덱 순서의 위치다 (0부터).
	Slide int
	// Step은 reveal 스텝이다 (-1이면 모든 스텝이 펼쳐진 장면).
	Step  int
	Image *image.RGBA
}

// Result는 덱 하나를 그린 결과다.
type Result struct {
	// Geometry는 내장 엔진이 잰 지오메트리다 (배치 진단 · PPTX가 그대로 소비한다).
	Geometry *geom.Geometry
	// Slides는 덱 순서의 그림이다 (모든 스텝이 펼쳐진 상태, 그리지 않은 슬라이드는 nil).
	Slides []*image.RGBA
	// Steps는 Options.Steps일 때 슬라이드마다 스텝 0 … n의 장면이다 (reveal이 없으면 Slides와 같은 한 장).
	Steps [][]*image.RGBA
}

// Snapshot은 덱을 슬라이드마다 그린다. 흐름은 빌드와 같다: 측정용 HTML을 배치해 지오메트리를
// 얻고, 그 박스로 연결선 · 잰 박스 도형을 구운 최종 HTML을 다시 배치해 그린다. only가 비어
// 있지 않으면 그 인덱스의 슬라이드만 그린다.
func Snapshot(deck *soml.Deck, opts Options) (*Result, error) {
	files, err := web.NewForMeasurement().Render(deck)
	if err != nil {
		return nil, fmt.Errorf("cannot prepare the layout input: %w", err)
	}
	doc, err := Layout(files[web.IndexFile], filesAssets(files))
	if err != nil {
		return nil, err
	}
	raw, err := doc.MeasurementJSON()
	if err != nil {
		return nil, err
	}
	g, err := geom.FromMeasurement(deck, raw)
	if err != nil {
		return nil, fmt.Errorf("builtin layout does not match the deck: %w", err)
	}
	final, err := web.NewWithBoxes(g.RouteBox).Render(deck)
	if err != nil {
		return nil, err
	}
	doc, err = Layout(final[web.IndexFile], filesAssets(final))
	if err != nil {
		return nil, err
	}
	out := &Result{Geometry: g}
	emit := opts.Emit
	if emit == nil {
		emit = func(f Frame) error {
			if f.Step < 0 {
				out.Slides[f.Slide] = f.Image
				if opts.Steps {
					out.Steps[f.Slide] = append(out.Steps[f.Slide], f.Image)
				}
				return nil
			}
			out.Steps[f.Slide] = append(out.Steps[f.Slide], f.Image)
			return nil
		}
		out.Slides = make([]*image.RGBA, len(doc.Slides))
		if opts.Steps {
			out.Steps = make([][]*image.RGBA, len(doc.Slides))
		}
	}
	for i, s := range doc.Slides {
		if len(opts.Only) > 0 && !opts.Only[s.Index] {
			continue
		}
		if opts.Steps {
			for step := 0; step < s.Steps(); step++ {
				if err := emit(Frame{Slide: i, Step: step, Image: doc.RenderStep(s, opts.DPR, step, opts.Outline)}); err != nil {
					return nil, err
				}
			}
		}
		if err := emit(Frame{Slide: i, Step: -1, Image: doc.RenderStep(s, opts.DPR, -1, opts.Outline)}); err != nil {
			return nil, err
		}
	}
	return out, nil
}

func filesAssets(files render.Files) func(string) ([]byte, bool) {
	return func(name string) ([]byte, bool) {
		data, ok := files[pathpkg.Clean(name)]
		return data, ok
	}
}

// EncodePNG는 그림을 PNG로 쓴다 (빠른 압축 — 미리보기는 저장 크기보다 시간이 중요하다).
func EncodePNG(img image.Image) ([]byte, error) {
	var buf bytes.Buffer
	enc := png.Encoder{CompressionLevel: png.BestSpeed}
	if err := enc.Encode(&buf, img); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
