package testkit

import (
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"testing"

	"github.com/k2sebeom/balpyo/internal/pipeline"
	"github.com/k2sebeom/balpyo/render"
	"github.com/k2sebeom/balpyo/render/web"
)

// 인쇄의 브라우저 검증 — 산출물을 **실제로 인쇄해서** 한 장이 한 페이지가 되는지 본다.
//
// 이 게이트가 있는 이유는 실측이다: 인쇄 CSS가 없으면 발표자 뷰에서 Cmd+P를 누른
// 사람은 letter 한 페이지(활성 슬라이드 + 컨트롤 바)를 받고, 문서 뷰에서는 슬라이드가 페이지
// 경계에서 잘린 PDF를 받는다. 두 실패 모두 조용하다 — PDF는 정상적으로 나온다.
//
// 판정은 페이지 수와 종이 크기다 — PDF의 페이지 사전을 직접 읽는다 (래스터화하지 않는다).
// 인쇄는 같은 CSS의 다른 미디어이므로 게이트의 무게는 "한 장이 한 페이지로 나왔다"에 둔다.
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

	pages := pdfPages(t, browser.PrintPDF(t))
	if len(pages) != len(deck.Slides) {
		t.Fatalf("발표자 뷰에서 인쇄한 페이지 = %d, want %d (슬라이드 수)", len(pages), len(deck.Slides))
	}

	// 종이는 논리 스테이지다 — PDF는 pt이고 CSS px은 1/96in이므로 0.75배다.
	wantW := math.Round(float64(deck.Stage.Width) * 0.75)
	wantH := math.Round(float64(deck.Stage.Height) * 0.75)
	for i, page := range pages {
		if math.Abs(page[0]-wantW) > 1 || math.Abs(page[1]-wantH) > 1 {
			t.Errorf("페이지 %d 종이 크기 = %vx%vpt, want %vx%vpt", i+1, page[0], page[1], wantW, wantH)
		}
	}

	// 문서 뷰에서 눌러도 같다 — 인쇄는 뷰가 아니라 미디어이므로 사용자가 무엇을 보고 있든
	// 같은 덱이 나온다.
	browser.Eval(t, web.DocumentViewExpr(), nil)
	if got := len(pdfPages(t, browser.PrintPDF(t))); got != len(deck.Slides) {
		t.Errorf("문서 뷰에서 인쇄한 페이지 = %d, want %d", got, len(deck.Slides))
	}
}

// pdfMediaBox는 페이지 사전의 종이 크기다 (`/MediaBox [0 0 w h]`). Chromium은 페이지 사전을
// 객체 스트림에 압축하지 않으므로 바이트에서 바로 읽힌다.
var pdfMediaBox = regexp.MustCompile(`/MediaBox\s*\[\s*0\s+0\s+([\d.]+)\s+([\d.]+)\s*\]`)

// pdfPages는 PDF의 페이지마다 종이 크기(pt)를 문서 순서로 돌려준다.
func pdfPages(t testing.TB, pdfPath string) [][2]float64 {
	t.Helper()
	data, err := os.ReadFile(pdfPath)
	if err != nil {
		t.Fatal(err)
	}
	var pages [][2]float64
	for _, m := range pdfMediaBox.FindAllSubmatch(data, -1) {
		w, _ := strconv.ParseFloat(string(m[1]), 64)
		h, _ := strconv.ParseFloat(string(m[2]), 64)
		pages = append(pages, [2]float64{w, h})
	}
	return pages
}
