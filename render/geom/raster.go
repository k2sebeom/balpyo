package geom

import (
	"context"
	"encoding/base64"
	"fmt"
	"math"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/emulation"
	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"

	"github.com/k2sebeom/balpyo/soml"
)

// fallbackMaxSide는 대체 그림의 긴 변 상한(px)이다 — 큰 SVG(포스터 크기의 viewBox)가 수십 MB의 PNG가 되지 않게 한다.
// 배율은 이 상한 안에서 FallbackScale까지다.
const fallbackMaxSide = 4096

// rasterScript는 빈 문서에 SVG 그림 하나를 고유 크기로 놓고 디코드를 기다린다. 그림은 `<img>`다 — 웹 산출물이 SVG를
// 그리는 것과 같은 요소라 같은 그림이 나온다 (스크립트 · 바깥 참조가 막힌 그림 문맥).
const rasterScript = `(async () => {
  document.documentElement.style.background = "transparent";
  document.body.style.margin = "0";
  document.body.style.background = "transparent";
  document.body.replaceChildren();
  const img = new Image();
  img.style.cssText = "display:block;width:%[2]dpx;height:%[3]dpx";
  img.src = %[1]q;
  document.body.appendChild(img);
  await img.decode();
  return true;
})()`

// RasterizeSVG는 SVG 바이트를 w×h 논리 px로 그린 투명 바탕 PNG다 (해상도 배율 FallbackScale, 긴 변 상한
// fallbackMaxSide). 열려 있던 문서를 빈 문서로 바꾼다.
func (s *Session) RasterizeSVG(svg []byte, w, h int) ([]byte, error) {
	if w <= 0 || h <= 0 {
		return nil, fmt.Errorf("cannot rasterize a %d×%d SVG", w, h)
	}
	scale := math.Min(FallbackScale, fallbackMaxSide/float64(max(w, h)))
	src := "data:image/svg+xml;base64," + base64.StdEncoding.EncodeToString(svg)
	var buf []byte
	err := s.run(
		chromedp.Navigate("about:blank"),
		chromedp.Evaluate(fmt.Sprintf(rasterScript, src, w, h), nil, withAwaitPromise),
		chromedp.ActionFunc(func(ctx context.Context) error {
			// 바탕을 투명하게 둔다 — 대체 그림은 슬라이드 위에 얹히는 그림이고, 흰 바탕이면 어두운 장에서 상자가 보인다.
			if err := emulation.SetDefaultBackgroundColorOverride().WithColor(&cdp.RGBA{R: 0, G: 0, B: 0, A: 0}).Do(ctx); err != nil {
				return err
			}
			var err error
			buf, err = page.CaptureScreenshot().
				WithFormat(page.CaptureScreenshotFormatPng).
				WithCaptureBeyondViewport(true).
				WithClip(&page.Viewport{X: 0, Y: 0, Width: float64(w), Height: float64(h), Scale: scale}).Do(ctx)
			if err != nil {
				return err
			}
			return emulation.SetDefaultBackgroundColorOverride().Do(ctx)
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("cannot rasterize the SVG: %w", err)
	}
	return buf, nil
}

// svgFallbacks는 덱의 SVG 에셋마다 대체 그림을 그린다 (에셋 표 순서).
func (s *Session) svgFallbacks(deck *soml.Deck) ([]Fallback, error) {
	var out []Fallback
	for _, asset := range deck.Assets {
		if asset.Format != soml.FormatSVG {
			continue
		}
		png, err := s.RasterizeSVG(asset.Bytes, asset.Width, asset.Height)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", asset.Key, err)
		}
		out = append(out, Fallback{Key: asset.Key, PNG: png})
	}
	return out, nil
}
