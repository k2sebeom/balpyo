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

// 발표자 뷰의 브라우저 검증 — 산출물 index.html이 발표 가능한 웹앱인지 본다: 슬라이드가
// 한 장씩 화면에 맞춰 나오고, 하단 컨트롤 바·키보드·해시 딥링크로 넘기고, 노트 패널이
// 열린다.
//
// Session.Open은 측정을 위해 문서 뷰로 되돌리므로(render/geom의 계약), 여기서는 사용자가
// 파일을 열었을 때의 상태를 web.PresenterViewExpr로 다시 켠 뒤 조작한다. 런타임의 리스너와
// 컨트롤 바는 로드 때 이미 준비되어 있다 — 뷰 전환은 <html>의 속성 하나다.
func TestPresenterView(t *testing.T) {
	fixture := textFixture(t)
	// HTML만 빌드한다 — 이 테스트에 지오메트리·LibreOffice는 필요 없고, 브라우저가 없는
	// 환경의 정책(건너뛰기/실패)은 NewBrowser가 정한다.
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

	// Load(=Session.Open)가 문서 뷰로 되돌렸는지 — 지오메트리 패스와 스크린샷이 재는 배치다.
	var presenter bool
	browser.Eval(t, fmt.Sprintf("document.documentElement.hasAttribute(%q)", web.ViewAttr), &presenter)
	if presenter {
		t.Fatal("Open 뒤에도 발표자 뷰가 켜져 있다 — 측정이 스케일된 배치를 재게 된다")
	}
	if got := computedDisplay(t, browser, "."+web.ViewerBarClass); got != "none" {
		t.Fatalf("문서 뷰의 컨트롤 바 display = %q, want none", got)
	}

	// 사용자가 파일을 열었을 때의 상태로 되돌린다.
	browser.Eval(t, web.PresenterViewExpr(), nil)

	if got := visibleSlides(t, browser); len(got) != 1 || got[0] != 0 {
		t.Fatalf("발표자 뷰에서 보이는 슬라이드 = %v, want [0]", got)
	}
	if got := computedDisplay(t, browser, "."+web.ViewerBarClass); got == "none" {
		t.Fatal("발표자 뷰에서 컨트롤 바가 보이지 않는다")
	}
	waitCounter(t, browser, fmt.Sprintf("1 / %d", len(deck.Slides)))

	// 활성 슬라이드는 화면 안에 맞춰 스케일된다 — transform이 안 걸리면 1280px 슬라이드가
	// 뷰포트를 넘는다.
	var fits bool
	browser.Eval(t, fmt.Sprintf(`(() => {
  const r = document.querySelector(%q).getBoundingClientRect();
  return r.width > 0 && r.width <= window.innerWidth + 0.5 && r.height <= window.innerHeight + 0.5;
})()`, web.SlideIndexSelector(0)), &fits)
	if !fits {
		t.Error("활성 슬라이드가 화면에 맞춰 스케일되지 않았다")
	}

	// 다음 버튼 → 두 번째 슬라이드만 보이고, 해시가 그 슬라이드의 딥링크가 된다.
	click(t, browser, "."+web.ViewerNextClass)
	if got := visibleSlides(t, browser); len(got) != 1 || got[0] != 1 {
		t.Fatalf("다음 이동 뒤 보이는 슬라이드 = %v, want [1]", got)
	}
	waitCounter(t, browser, fmt.Sprintf("2 / %d", len(deck.Slides)))
	var hash string
	browser.Eval(t, "location.hash", &hash)
	if want := "#/" + deck.Slides[1].ID; hash != want {
		t.Errorf("이동 뒤 해시 = %q, want %q", hash, want)
	}

	// 노트 토글 → 패널이 열린다. 두 번째 슬라이드는 노트가 없으므로(픽스처 구조) 자리
	// 표시 문구가 나온다 — 빈 패널은 "안 열렸다"와 구분되지 않는다.
	if deck.Slides[1].Notes != "" || deck.Slides[2].Notes == "" {
		t.Fatal("픽스처 전제가 깨졌다: 01_text는 노트가 슬라이드 1·3에만 있다")
	}
	click(t, browser, "."+web.ViewerNotesToggleClass)
	var hidden bool
	browser.Eval(t, fmt.Sprintf("document.querySelector(%q).hidden", "."+web.ViewerNotesClass), &hidden)
	if hidden {
		t.Fatal("노트 토글 뒤에도 패널이 닫혀 있다")
	}
	if note := notesPanelText(t, browser); note == "" || note == deck.Slides[1].Notes {
		t.Errorf("노트 없는 슬라이드의 패널 = %q, want 자리 표시 문구", note)
	}

	// 키보드로 다음 슬라이드 → 노트가 있는 슬라이드에서는 패널이 그 노트를 보여준다.
	browser.Eval(t, `window.dispatchEvent(new KeyboardEvent("keydown", {key: "ArrowRight"}))`, nil)
	waitCounter(t, browser, fmt.Sprintf("3 / %d", len(deck.Slides)))
	if note := notesPanelText(t, browser); note != deck.Slides[2].Notes {
		t.Errorf("노트 패널 내용 = %q, want %q", note, deck.Slides[2].Notes)
	}

	// 해시 딥링크 → hashchange 구독이 살아 있다. 이벤트는 비동기라 카운터를 폴링한다.
	browser.Eval(t, fmt.Sprintf("location.hash = %q", "#/"+deck.Slides[0].ID), nil)
	waitCounter(t, browser, fmt.Sprintf("1 / %d", len(deck.Slides)))
	if got := visibleSlides(t, browser); len(got) != 1 || got[0] != 0 {
		t.Fatalf("딥링크 이동 뒤 보이는 슬라이드 = %v, want [0]", got)
	}
}

// computedDisplay는 선택자에 맞는 첫 요소의 계산된 display다.
func computedDisplay(t testing.TB, b *Browser, selector string) string {
	t.Helper()
	var out string
	b.Eval(t, fmt.Sprintf("getComputedStyle(document.querySelector(%q)).display", selector), &out)
	return out
}

// visibleSlides는 계산된 visibility가 visible인 슬라이드 인덱스 목록이다 — 발표자 뷰는
// display가 아니라 visibility로 숨기므로(레이아웃 유지) 이것이 "보인다"의 정의다.
func visibleSlides(t testing.TB, b *Browser) []int {
	t.Helper()
	var out []int
	b.Eval(t, fmt.Sprintf(`Array.from(document.querySelectorAll(%q))
  .map((s, i) => [i, getComputedStyle(s).visibility])
  .filter((p) => p[1] === "visible").map((p) => p[0])`, web.SlideSelector()), &out)
	return out
}

// click은 선택자에 맞는 첫 요소를 클릭한다.
func click(t testing.TB, b *Browser, selector string) {
	t.Helper()
	b.Eval(t, fmt.Sprintf("document.querySelector(%q).click()", selector), nil)
}

// notesPanelText는 노트 패널의 내용이다.
func notesPanelText(t testing.TB, b *Browser) string {
	t.Helper()
	var out string
	b.Eval(t, fmt.Sprintf("document.querySelector(%q).textContent", "."+web.ViewerNotesClass), &out)
	return out
}

// waitCounter는 컨트롤 바 카운터가 원하는 값이 될 때까지 폴링한다 — hashchange처럼
// 비동기로 처리되는 이동이 있어 한 번 읽기로는 경쟁이 된다.
func waitCounter(t testing.TB, b *Browser, want string) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for {
		var got string
		b.Eval(t, fmt.Sprintf("document.querySelector(%q).textContent", "."+web.ViewerCounterClass), &got)
		if got == want {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("카운터 = %q, want %q", got, want)
		}
		time.Sleep(50 * time.Millisecond)
	}
}
