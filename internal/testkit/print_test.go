package testkit

import (
	"math"
	"path/filepath"
	"testing"

	"github.com/k2sebeom/balpyo/internal/pipeline"
	"github.com/k2sebeom/balpyo/render"
	"github.com/k2sebeom/balpyo/render/geom"
	"github.com/k2sebeom/balpyo/render/web"
)

// 인쇄의 브라우저 검증 — 산출물을 **실제로 인쇄해서** 한 장이 한 페이지가 되는지 본다.
//
// 이 게이트가 있는 이유는 실측이다 (ADR 0062): 인쇄 CSS가 없으면 발표자 뷰에서 Cmd+P를 누른
// 사람은 letter 한 페이지(활성 슬라이드 + 컨트롤 바)를 받고, 문서 뷰에서는 슬라이드가 페이지
// 경계에서 잘린 PDF를 받는다. 두 실패 모두 조용하다 — PDF는 정상적으로 나온다.
//
// 판정은 페이지 수·종이 크기·페이지마다의 잉크까지다. 픽셀 대조는 하지 않는다: 인쇄는 같은
// CSS의 다른 미디어이므로 게이트의 무게는 "덱이 온전히 나왔다"에 둔다.
func TestPrintIsOnePagePerSlide(t *testing.T) {
	fixture := textFixture(t)
	// HTML만 빌드한다 — 인쇄는 브라우저의 일이고 PPTX·지오메트리와 무관하다.
	result, err := pipeline.Build(pipeline.Options{
		ProjectDir: fixture.Dir,
		Targets:    []render.Target{render.TargetHTML},
		OutDir:     t.TempDir(),
	})
	if err != nil {
		t.Fatalf("%s 빌드 실패: %v", fixture.Name, err)
	}
	deck := result.Deck

	browser := NewBrowser(t)
	browser.Load(t, filepath.Join(result.OutDir, web.IndexFile))

	// 사람이 Cmd+P를 누르는 시점은 **발표자 뷰**다 (런타임이 로드 즉시 켠다). Load는 측정을
	// 위해 문서 뷰로 되돌리므로 여기서 다시 켠다.
	browser.Eval(t, web.PresenterViewExpr(), nil)

	pages := RenderPDFPoints(t, browser.PrintPDF(t))
	if len(pages) != len(deck.Slides) {
		t.Fatalf("발표자 뷰에서 인쇄한 페이지 = %d, want %d (슬라이드 수)", len(pages), len(deck.Slides))
	}

	// 종이는 논리 스테이지다 — PDF는 pt이고 CSS px은 1/96in이므로 0.75배다.
	wantW := int(math.Round(float64(deck.Stage.Width) * 0.75))
	wantH := int(math.Round(float64(deck.Stage.Height) * 0.75))
	for i, page := range pages {
		size := page.Bounds().Size()
		if abs(size.X-wantW) > 1 || abs(size.Y-wantH) > 1 {
			t.Errorf("페이지 %d 종이 크기 = %dx%dpt, want %dx%dpt", i+1, size.X, size.Y, wantW, wantH)
		}
		// 빈 페이지는 "슬라이드가 나왔다"와 구분되지 않는다 — 잉크가 있어야 한다.
		full := geom.Rect{W: float64(size.X), H: float64(size.Y)}
		if bands := InkBands(page, full); len(bands) == 0 {
			t.Errorf("페이지 %d가 비었다", i+1)
		}
	}

	// 문서 뷰에서 눌러도 같다 — 인쇄는 뷰가 아니라 미디어이므로 사용자가 무엇을 보고 있든
	// 같은 덱이 나온다.
	browser.Eval(t, web.DocumentViewExpr(), nil)
	if got := len(RenderPDFPoints(t, browser.PrintPDF(t))); got != len(deck.Slides) {
		t.Errorf("문서 뷰에서 인쇄한 페이지 = %d, want %d", got, len(deck.Slides))
	}
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}
