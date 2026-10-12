package testkit

import (
	"bytes"
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

// 좌표로 놓은 요소(`x`/`y` — canvas 스테이지와 `<stack>`의 자식)도 종이에 찍힌다. 화면에서는 보이는데
// 인쇄에서만 빠지던 회귀다: Chromium의 페이지 나눔이 grid 영역 안의 절대 위치 박스를 잃어버려 canvas
// 장이 꼬리말만 남은 빈 장으로 인쇄됐다.
//
// 판정은 링크 주석이다 — Chromium은 종이에 그린 링크마다 `/URI` 주석을 남기므로, 좌표로 놓은 글이
// 품은 링크가 PDF에 있으면 그 글이 그려진 것이다 (래스터화하지 않는다). 첫 장은 흐름 배치라 둘째 장이
// 페이지 나눔을 지난다.
func TestPrintKeepsCoordinatePlacedChildren(t *testing.T) {
	outDir := buildHTML(t, map[string]string{
		"balpyo.json":           `{"title": "print"}`,
		"slides/001_flow.slide": `<slide id="flow"><h1>Flow</h1></slide>`,
		"slides/002_canvas.slide": `<slide id="canvas" layout="canvas">
  <text x="100" y="200"><a href="https://example.com/canvas-xy">canvas</a></text>
  <stack w="400" h="200" x="500" y="300"><shape type="rect" w="400" h="200"/><text x="20" y="20"><a href="https://example.com/stack-xy">stack</a></text></stack>
</slide>`,
	})
	browser := NewBrowser(t)
	browser.Load(t, filepath.Join(outDir, web.IndexFile))
	browser.Eval(t, web.PresenterViewExpr(), nil)
	pdf, err := os.ReadFile(browser.PrintPDF(t))
	if err != nil {
		t.Fatal(err)
	}
	for _, uri := range []string{"https://example.com/canvas-xy", "https://example.com/stack-xy"} {
		if !bytes.Contains(pdf, []byte(uri)) {
			t.Errorf("인쇄한 PDF에 %s가 없다 — 좌표로 놓은 글이 종이에서 빠졌다", uri)
		}
	}
}

// 등장 효과는 발표자 뷰의 것이다 — 문서 뷰로 연 순간(스크린숏 · 측정 · 인쇄가 보는 상태) 모든 요소가 이미
// 끝 상태다. 지연이 인라인 `transition-delay`로 걸려 있으면 길이 0인 전환이 그 지연만큼 요소를 시작
// 상태(opacity 0)에 붙잡아, preview 스크린숏이 연결선 몇 개가 빠진 장을 찍었다 (실행마다 달랐다).
func TestEntrancesAreAtTheirEndStateOutsideThePresenterView(t *testing.T) {
	outDir := buildHTML(t, map[string]string{
		"balpyo.json":            `{"title": "enter"}`,
		"slides/001_first.slide": `<slide id="first"><h1>First</h1></slide>`,
		"slides/002_staged.slide": `<slide id="staged" enter="fade">
  <h1>One</h1><p>Two</p><p>Three</p><p>Four</p><p>Five</p><p>Six</p>
</slide>`,
	})
	browser := NewBrowser(t)
	browser.Load(t, filepath.Join(outDir, web.IndexFile))
	var hidden []string
	browser.Eval(t, `[...document.querySelectorAll("[`+web.EnterAttr+`]")]
  .filter((el) => getComputedStyle(el).opacity !== "1")
  .map((el) => el.getAttribute("data-balpyo-id"))`, &hidden)
	if len(hidden) > 0 {
		t.Errorf("문서 뷰로 연 직후에 아직 보이지 않는 등장 요소: %v", hidden)
	}
}

// 전환 도중에 인쇄해도 모든 장이 종이에 찍힌다. 발표자가 다음 장으로 넘긴 직후(전환 길이 안)에
// Cmd+P를 누르면, 이동 표식이 남긴 translate·opacity가 인쇄까지 따라와 나가는 장이 종이 밖으로
// 밀리거나 투명해진 빈 페이지로 나왔다.
//
// 판정은 링크 주석이다 (좌표로 놓은 글의 인쇄와 같은 방법): 장마다 링크를 하나씩 두고, 인쇄한 PDF에
// 둘 다 있으면 두 장 모두 그려진 것이다. push와 fade 둘 다 본다 — 둘은 다른 속성으로 숨긴다.
func TestPrintDuringASlideTransitionKeepsEverySlide(t *testing.T) {
	for _, kind := range []string{"push", "fade"} {
		t.Run(kind, func(t *testing.T) {
			outDir := buildHTML(t, map[string]string{
				"balpyo.json":        `{"title": "print"}`,
				"slides/001_a.slide": `<slide id="a"><h1><a href="https://example.com/leaving">leaving</a></h1></slide>`,
				"slides/002_b.slide": `<slide id="b" transition="` + kind + `"><h1><a href="https://example.com/arriving">arriving</a></h1></slide>`,
			})
			browser := NewBrowser(t)
			browser.Load(t, filepath.Join(outDir, web.IndexFile))
			browser.Eval(t, web.PresenterViewExpr(), nil)
			// 다음 장으로 넘기고 전환이 끝나기 전에 인쇄한다. 표식이 아직 남아 있음을 먼저 확인한다 —
			// 그렇지 않으면 이 테스트는 전환과 무관하게 통과한다. 측정 브라우저는 동작 줄이기를 켜고
			// 뜨므로(런타임은 그때 전환을 건너뛴다) 이 페이지에서만 그 질의를 거짓으로 답한다.
			var moving bool
			browser.Eval(t, `(() => {
  const media = window.matchMedia.bind(window);
  window.matchMedia = (q) => q.includes("prefers-reduced-motion") ? {matches: false, media: q} : media(q);
  window.dispatchEvent(new KeyboardEvent("keydown", {key: "ArrowRight"}));
  return document.querySelectorAll(".balpyo-viewer-moving").length > 0;
})()`, &moving)
			if !moving {
				t.Fatal("다음 장으로 넘긴 직후에 전환 표식이 없다 — 전환 도중의 인쇄를 재현하지 못한다")
			}
			pdf, err := os.ReadFile(browser.PrintPDF(t))
			if err != nil {
				t.Fatal(err)
			}
			for _, uri := range []string{"https://example.com/leaving", "https://example.com/arriving"} {
				if !bytes.Contains(pdf, []byte(uri)) {
					t.Errorf("전환 도중에 인쇄한 PDF에 %s가 없다 — 그 장이 종이에서 빠졌다", uri)
				}
			}
		})
	}
}

// buildHTML은 files(프로젝트 상대 경로 → 내용)로 프로젝트를 만들어 HTML만 빌드하고 출력 디렉토리를 돌려준다.
func buildHTML(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for name, body := range files {
		path := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	result, err := pipeline.Build(pipeline.Options{
		ProjectDir: dir,
		Targets:    []render.Target{render.TargetHTML},
		OutDir:     t.TempDir(),
	})
	if err != nil {
		t.Fatal(err)
	}
	return result.OutDir
}
