package testkit

import (
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/k2sebeom/balpyo/internal/pipeline"
	"github.com/k2sebeom/balpyo/render"
	"github.com/k2sebeom/balpyo/render/web"
)

// 오버뷰의 브라우저 검증 — 슬라이드 전부를 격자로 줄여 놓고 하나를 골라 들어가는 화면이다.
//
// 오버뷰는 네 번째 뷰 값이고 줄이는 수단은 zoom이다. 여기서 보는 것 넷: 들어가면
// 전부 보인다, 타일이 **화면 안에** 들어온다, 나오면 보고 있던 슬라이드로 돌아온다(오버뷰는
// 이동이 아니라 보기다), 타일을 클릭하면 그 슬라이드로 들어간다.
func TestOverviewView(t *testing.T) {
	fixture := textFixture(t)
	result, err := pipeline.Build(pipeline.Options{
		ProjectDir: fixture.Dir,
		Targets:    []render.Target{render.TargetHTML},
		OutDir:     t.TempDir(),
	})
	if err != nil {
		t.Fatalf("%s 빌드 실패: %v", fixture.Name, err)
	}
	deck := result.Deck
	if len(deck.Slides) < 3 {
		t.Fatalf("픽스처 전제가 깨졌다: 오버뷰 판정에는 슬라이드 3장이 필요하다 (%d장)", len(deck.Slides))
	}
	n := len(deck.Slides)

	browser := NewBrowser(t)
	browser.Load(t, filepath.Join(result.OutDir, web.IndexFile))
	browser.Eval(t, web.PresenterViewExpr(), nil)
	waitCounter(t, browser, fmt.Sprintf("1 / %d", n))

	// 첫 슬라이드가 아닌 자리에서 들어간다 — 그래야 "나오면 자리를 지킨다"가 판정된다.
	click(t, browser, "."+web.ViewerNextClass)
	waitCounter(t, browser, fmt.Sprintf("2 / %d", n))

	pressKey(t, browser, "o")
	waitView(t, browser, web.OverviewView)

	if got := visibleSlides(t, browser); len(got) != n {
		t.Fatalf("오버뷰에서 보이는 슬라이드 = %v, want %d장 전부", got, n)
	}
	// 오버뷰는 이동이 아니다 — 카운터가 그대로여야 나올 자리를 안다.
	waitCounter(t, browser, fmt.Sprintf("2 / %d", n))

	// 타일은 줄어들고 전부 화면 안에 들어온다 (열 수·배율을 런타임이 창에 맞춰 계산한다).
	var fits bool
	browser.Eval(t, fmt.Sprintf(`(() => {
  const slides = Array.from(document.querySelectorAll(%q));
  return slides.every((s) => {
    const r = s.getBoundingClientRect();
    return r.width > 0 && r.width < s.offsetWidth &&
      r.left >= -0.5 && r.top >= -0.5 &&
      r.right <= window.innerWidth + 0.5 && r.bottom <= window.innerHeight + 0.5;
  });
})()`, web.SlideSelector()), &fits)
	if !fits {
		t.Error("오버뷰 타일이 줄어들지 않았거나 화면을 넘는다")
	}
	// 컨트롤 바는 오버뷰에서도 보인다 — 키보드를 모르는 사람이 나올 길이 그것이다.
	if got := computedDisplay(t, browser, "."+web.ViewerBarClass); got == "none" {
		t.Error("오버뷰에서 컨트롤 바가 사라졌다")
	}

	// Esc → 발표자 뷰, 보고 있던 슬라이드.
	pressKey(t, browser, "Escape")
	waitView(t, browser, web.PresenterView)
	if got := visibleSlides(t, browser); len(got) != 1 || got[0] != 1 {
		t.Fatalf("오버뷰에서 나온 뒤 보이는 슬라이드 = %v, want [1]", got)
	}

	// 바 버튼으로 들어가고, 타일을 클릭하면 그 슬라이드로 들어간다.
	click(t, browser, "."+web.ViewerOverviewClass)
	waitView(t, browser, web.OverviewView)
	click(t, browser, web.SlideIndexSelector(2))
	waitView(t, browser, web.PresenterView)
	waitCounter(t, browser, fmt.Sprintf("3 / %d", n))
	if got := visibleSlides(t, browser); len(got) != 1 || got[0] != 2 {
		t.Fatalf("타일 클릭 뒤 보이는 슬라이드 = %v, want [2]", got)
	}
}

// 오버뷰의 타일 클릭은 링크 위에서도 그 슬라이드로 들어가는 것이지 링크를 따라가는 것이
// 아니다 — 타일이 줄어 있어 링크 제목이 타일의 큰 몫을 덮고, 따라가면 덱을 떠난다. 발표자
// 뷰에서는 링크가 그대로 동작해야 한다.
//
// 판정은 기본 동작이 막혔는지(defaultPrevented)로 한다: 실제로 이동시키면 페이지가 바뀌어
// 다음 판정을 할 문서가 사라진다. 문서에 단 리스너는 슬라이드의 리스너 뒤에 불리므로 그
// 시점의 defaultPrevented가 런타임의 결정이고, 리스너가 스스로 막아 이동은 일어나지 않는다.
func TestOverviewTileClickOnLinkStaysInDeck(t *testing.T) {
	fixture := fixtureByName(t, "02_text_subset")
	result, err := pipeline.Build(pipeline.Options{
		ProjectDir: fixture.Dir,
		Targets:    []render.Target{render.TargetHTML},
		OutDir:     t.TempDir(),
	})
	if err != nil {
		t.Fatalf("%s 빌드 실패: %v", fixture.Name, err)
	}
	deck := result.Deck
	n := len(deck.Slides)
	linked := -1
	for i, s := range deck.Slides {
		if s.ID == "inline" {
			linked = i
		}
	}
	if linked < 1 {
		t.Fatalf("픽스처 전제가 깨졌다: 링크가 있는 슬라이드 inline이 첫 슬라이드 뒤에 있어야 한다 (%d)", linked)
	}

	browser := NewBrowser(t)
	browser.Load(t, filepath.Join(result.OutDir, web.IndexFile))
	browser.Eval(t, web.PresenterViewExpr(), nil)
	waitCounter(t, browser, fmt.Sprintf("1 / %d", n))

	clickLink := func() bool {
		t.Helper()
		var prevented bool
		browser.Eval(t, fmt.Sprintf(`(() => {
  const link = document.querySelectorAll(%q)[%d].querySelector("a[href]");
  if (!link) throw new Error("no link in slide");
  let prevented = null;
  const probe = (e) => { prevented = e.defaultPrevented; e.preventDefault(); };
  document.addEventListener("click", probe);
  link.dispatchEvent(new MouseEvent("click", {bubbles: true, cancelable: true}));
  document.removeEventListener("click", probe);
  return prevented;
})()`, web.SlideSelector(), linked), &prevented)
		return prevented
	}

	pressKey(t, browser, "o")
	waitView(t, browser, web.OverviewView)
	if !clickLink() {
		t.Error("오버뷰에서 링크 위 타일 클릭이 링크를 따라간다 — 덱을 떠난다")
	}
	waitView(t, browser, web.PresenterView)
	waitCounter(t, browser, fmt.Sprintf("%d / %d", linked+1, n))

	// 발표자 뷰에서는 링크가 링크다.
	if clickLink() {
		t.Error("발표자 뷰에서 링크 클릭의 기본 동작이 막혔다")
	}
}

// pressKey는 문서에 키를 누른 것과 같은 이벤트를 보낸다 (런타임의 리스너는 window에 있다).
func pressKey(t testing.TB, b *Browser, key string) {
	t.Helper()
	b.Eval(t, fmt.Sprintf(
		`window.dispatchEvent(new KeyboardEvent("keydown", {key: %q, bubbles: true, cancelable: true}))`, key), nil)
}

// waitView는 <html>의 뷰 속성이 원하는 값이 될 때까지 폴링한다.
func waitView(t testing.TB, b *Browser, want string) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for {
		var got string
		b.Eval(t, fmt.Sprintf("document.documentElement.getAttribute(%q) || %q", web.ViewAttr, ""), &got)
		if got == want {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("뷰 = %q, want %q", got, want)
		}
		time.Sleep(50 * time.Millisecond)
	}
}

// 오버뷰의 화살표는 타일 사이를 옮긴다 — 오버뷰에서는 reveal 항목이 늘 보이므로 스텝을
// 먹으면 강조가 제자리에 머문 채 키만 사라진다. 옮겨 들어간 장은 타일 클릭처럼 아직 말하지
// 않은 장이어서 나오면 스텝 0이다 (보지도 않은 스텝이 전개된 것으로 남지 않는다).
func TestOverviewArrowsMoveBetweenTiles(t *testing.T) {
	fixture := fixtureByName(t, "02_text_subset")
	result, err := pipeline.Build(pipeline.Options{
		ProjectDir: fixture.Dir,
		Targets:    []render.Target{render.TargetHTML},
		OutDir:     t.TempDir(),
	})
	if err != nil {
		t.Fatalf("%s 빌드 실패: %v", fixture.Name, err)
	}
	deck := result.Deck
	n := len(deck.Slides)

	browser := NewBrowser(t)
	browser.Load(t, filepath.Join(result.OutDir, web.IndexFile))
	slide := revealSlideIndex(t, browser)
	if slide < 0 || slide+1 >= n {
		t.Fatalf("픽스처 전제가 깨졌다: reveal 슬라이드(%d) 뒤에 슬라이드가 하나 더 있어야 한다 (%d장)", slide, n)
	}
	steps := revealSteps(t, browser, slide)
	if len(steps) < 2 {
		t.Fatalf("reveal 항목이 %d개다 — 판정에는 둘 이상이 필요하다", len(steps))
	}

	browser.Eval(t, web.PresenterViewExpr(), nil)
	browser.Eval(t, fmt.Sprintf("location.hash = %q", "#/"+deck.Slides[slide].ID), nil)
	waitCounter(t, browser, fmt.Sprintf("%d / %d", slide+1, n))

	pressKey(t, browser, "o")
	waitView(t, browser, web.OverviewView)

	// 한 번 누르면 다음 타일이다 — 스텝이 남아 있어도.
	pressKey(t, browser, "ArrowRight")
	waitCounter(t, browser, fmt.Sprintf("%d / %d", slide+2, n))
	pressKey(t, browser, "ArrowLeft")
	waitCounter(t, browser, fmt.Sprintf("%d / %d", slide+1, n))

	pressKey(t, browser, "Escape")
	waitView(t, browser, web.PresenterView)
	if got := hiddenReveals(t, browser, slide); !equalInts(got, steps) {
		t.Fatalf("오버뷰에서 옮겨 들어간 장의 숨은 항목 = %v, want %v (스텝 0)", got, steps)
	}
}
