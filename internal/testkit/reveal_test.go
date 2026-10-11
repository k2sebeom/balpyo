package testkit

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/k2sebeom/balpyo/internal/pipeline"
	"github.com/k2sebeom/balpyo/render"
	"github.com/k2sebeom/balpyo/render/web"
)

// `reveal`의 브라우저 검증 — 발표자 뷰에서 항목이 한 스텝씩 나타난다.
//
// 여기서 보는 것 다섯: (1) 슬라이드에 들어가면 스텝 0이다 — reveal을 쓴 항목은 숨어 있고
// 나머지는 보인다, (2) 다음 키가 스텝을 먼저 먹고 슬라이드는 그대로다, (3) 항목이 나타날 때
// **나머지 항목이 움직이지 않는다**(visibility로 숨기는 근거), (4) 스텝이 끝나면
// 다음 키가 다음 슬라이드로 가고, 뒤로 들어온 슬라이드는 **전부 전개된** 상태다,
// (5) 문서 뷰와 오버뷰는 언제나 전부 보인다.
func TestRevealSteps(t *testing.T) {
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

	// 문서 뷰(Load가 되돌린 상태)에서는 전부 보인다 — 지오메트리 패스와 스크린샷이 재는
	// 배치이므로 숨은 항목이 하나라도 있으면 그 좌표가 PPTX 텍스트 박스가 된다.
	slide := revealSlideIndex(t, browser)
	if slide < 0 {
		t.Fatal("픽스처에 reveal 항목이 없다 — 02_text_subset의 리스트 슬라이드가 순차 등장을 담아야 한다")
	}
	steps := revealSteps(t, browser, slide)
	if len(steps) < 2 {
		t.Fatalf("reveal 항목이 %d개다 — 스텝 판정에는 둘 이상이 필요하다", len(steps))
	}
	if got := hiddenReveals(t, browser, slide); len(got) != 0 {
		t.Fatalf("문서 뷰에서 숨은 reveal 항목 = %v, want 없음", got)
	}

	// 사용자가 파일을 열었을 때의 상태로 되돌리고 그 슬라이드로 곧장 들어간다 — 딥링크로
	// 들어간 장은 아직 말하지 않은 장이므로 스텝 0이다.
	browser.Eval(t, web.PresenterViewExpr(), nil)
	browser.Eval(t, fmt.Sprintf("location.hash = %q", "#/"+deck.Slides[slide].ID), nil)
	waitCounter(t, browser, fmt.Sprintf("%d / %d", slide+1, n))
	if got, want := hiddenReveals(t, browser, slide), steps; !equalInts(got, want) {
		t.Fatalf("들어간 직후 숨은 항목 = %v, want %v (스텝 0)", got, want)
	}
	// 자리는 스텝과 무관하다 — 숨김이 visibility이므로 레이아웃이 남는다.
	tops := revealTops(t, browser, slide)

	// 다음 키는 스텝을 먼저 먹는다: 첫 스텝의 항목만 나타나고 슬라이드는 그대로다.
	for step := 1; step <= steps[len(steps)-1]; step++ {
		pressKey(t, browser, "ArrowRight")
		waitCounter(t, browser, fmt.Sprintf("%d / %d", slide+1, n))
		var want []int
		for _, s := range steps {
			if s > step {
				want = append(want, s)
			}
		}
		if got := hiddenReveals(t, browser, slide); !equalInts(got, want) {
			t.Fatalf("스텝 %d에서 숨은 항목 = %v, want %v", step, got, want)
		}
		if got := revealTops(t, browser, slide); !equalFloats(got, tops) {
			t.Fatalf("스텝 %d에서 항목 자리가 움직였다: %v → %v", step, tops, got)
		}
	}

	// 스텝이 끝나면 다음 키가 다음 슬라이드로 간다.
	pressKey(t, browser, "ArrowRight")
	waitCounter(t, browser, fmt.Sprintf("%d / %d", slide+2, n))

	// 뒤로 들어온 슬라이드는 전부 전개된 상태다 — 이미 지나온 장을 다시 한 단계씩 누르게
	// 만들지 않는다.
	pressKey(t, browser, "ArrowLeft")
	waitCounter(t, browser, fmt.Sprintf("%d / %d", slide+1, n))
	if got := hiddenReveals(t, browser, slide); len(got) != 0 {
		t.Fatalf("뒤로 들어온 슬라이드의 숨은 항목 = %v, want 없음", got)
	}

	// 이전 키는 스텝을 하나 되감는다 — 발표 중 한 번 더 누른 실수를 되돌릴 길이다.
	last := steps[len(steps)-1]
	pressKey(t, browser, "ArrowLeft")
	waitCounter(t, browser, fmt.Sprintf("%d / %d", slide+1, n))
	if got := hiddenReveals(t, browser, slide); !equalInts(got, []int{last}) {
		t.Fatalf("한 번 되감은 뒤 숨은 항목 = %v, want %v", got, []int{last})
	}

	// 오버뷰는 덱을 **읽는** 화면이므로 전부 전개된 상태다.
	pressKey(t, browser, "o")
	waitView(t, browser, web.OverviewView)
	if got := hiddenReveals(t, browser, slide); len(got) != 0 {
		t.Fatalf("오버뷰에서 숨은 항목 = %v, want 없음", got)
	}
	pressKey(t, browser, "Escape")
	waitView(t, browser, web.PresenterView)
}

// revealSlideIndex는 reveal 항목이 있는 첫 슬라이드의 인덱스다 (없으면 -1).
func revealSlideIndex(t testing.TB, b *Browser) int {
	t.Helper()
	var out int
	b.Eval(t, fmt.Sprintf(`Array.from(document.querySelectorAll(%q))
  .findIndex((s) => s.querySelector("[data-balpyo-reveal]") !== null)`, web.SlideSelector()), &out)
	return out
}

// revealSteps는 그 슬라이드의 reveal 항목이 가진 스텝 번호다 (문서 순서).
func revealSteps(t testing.TB, b *Browser, slide int) []int {
	t.Helper()
	var out []int
	b.Eval(t, fmt.Sprintf(`Array.from(document.querySelectorAll(%q)[%d]
  .querySelectorAll("[data-balpyo-reveal]"))
  .map((e) => Number(e.getAttribute("data-balpyo-reveal")))`, web.SlideSelector(), slide), &out)
	return out
}

// hiddenReveals는 지금 숨어 있는 reveal 항목의 스텝 번호다 — "보인다"의 정의는 발표자
// 뷰와 같다 (계산된 visibility).
func hiddenReveals(t testing.TB, b *Browser, slide int) []int {
	t.Helper()
	var out []int
	b.Eval(t, fmt.Sprintf(`Array.from(document.querySelectorAll(%q)[%d]
  .querySelectorAll("[data-balpyo-reveal]"))
  .filter((e) => getComputedStyle(e).visibility !== "visible")
  .map((e) => Number(e.getAttribute("data-balpyo-reveal")))`, web.SlideSelector(), slide), &out)
	return out
}

// revealTops는 그 슬라이드 항목들의 레이아웃 자리다 (offsetTop — transform·스케일과 무관한
// 값이므로 스텝이 바뀌어도 같아야 한다).
func revealTops(t testing.TB, b *Browser, slide int) []float64 {
	t.Helper()
	var out []float64
	b.Eval(t, fmt.Sprintf(`Array.from(document.querySelectorAll(%q)[%d].querySelectorAll("li"))
  .map((e) => e.offsetTop)`, web.SlideSelector(), slide), &out)
	return out
}

func equalInts(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// Home은 언제나 "첫 장, 스텝 0"이고 End는 "마지막 장, 들어간 상태(스텝 0)"다 — 결과가
// 지금 어느 장에 있는지에 따라 달라지면 안 된다. 이미 첫 장(마지막 장)에서 스텝을 전개한
// 뒤에 누르는 경우가 그 갈림길이다: 장이 그대로라도 스텝은 되감겨야 한다.
func TestHomeAndEndRestartTheSlideTheyLandOn(t *testing.T) {
	slide := func(id string) string {
		return `<slide id="` + id + `"><ul><li reveal>One</li><li reveal>Two</li></ul></slide>`
	}
	outDir := buildHTML(t, map[string]string{
		"balpyo.json":            `{"title": "reveal"}`,
		"slides/001_first.slide": slide("first"),
		"slides/002_last.slide":  slide("last"),
	})
	browser := NewBrowser(t)
	browser.Load(t, filepath.Join(outDir, web.IndexFile))
	browser.Eval(t, web.PresenterViewExpr(), nil)
	waitCounter(t, browser, "1 / 2")

	for _, c := range []struct {
		key     string
		slide   int
		counter string
	}{
		{"Home", 0, "1 / 2"},
		{"End", 1, "2 / 2"},
	} {
		pressKey(t, browser, c.key)
		waitCounter(t, browser, c.counter)
		pressKey(t, browser, "ArrowRight")
		pressKey(t, browser, "ArrowRight")
		waitCounter(t, browser, c.counter)
		if got := hiddenReveals(t, browser, c.slide); len(got) != 0 {
			t.Fatalf("%s 장을 전개한 뒤 숨은 항목 = %v, want 없음", c.key, got)
		}
		pressKey(t, browser, c.key)
		waitCounter(t, browser, c.counter)
		if got := hiddenReveals(t, browser, c.slide); !equalInts(got, []int{1, 2}) {
			t.Errorf("같은 장에서 %s 뒤 숨은 항목 = %v, want [1 2] (스텝 0)", c.key, got)
		}
		var prevDisabled bool
		browser.Eval(t, fmt.Sprintf("document.querySelector(%q).disabled", "."+web.ViewerPrevClass), &prevDisabled)
		if want := c.slide == 0; prevDisabled != want {
			t.Errorf("같은 장에서 %s 뒤 이전 버튼 disabled = %v, want %v", c.key, prevDisabled, want)
		}
	}
}

func equalFloats(a, b []float64) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
