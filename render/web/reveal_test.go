package web

import (
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/k2sebeom/balpyo/core"
	"github.com/k2sebeom/balpyo/soml"
)

// revealDeck는 슬라이드 여럿을 원문에서 만든다 — 스텝 번호가 **슬라이드마다** 1부터
// 시작하는지 보려면 슬라이드가 둘 이상이어야 한다 (subsetDeck은 한 장이다).
func revealDeck(t *testing.T, bodies ...string) *soml.Deck {
	t.Helper()
	stage, err := soml.Aspect16x9.Stage()
	if err != nil {
		t.Fatal(err)
	}
	deck := &soml.Deck{Title: "reveal", Lang: "ko", Aspect: soml.Aspect16x9, Stage: stage}
	for i, body := range bodies {
		source := fmt.Sprintf("slides/%03d_reveal.slide", i+1)
		id := fmt.Sprintf("reveal-%d", i+1)
		doc, diags := soml.Parse(source, fmt.Sprintf("<slide id=%q>\n%s\n</slide>\n", id, body))
		if err := diags.Err(); err != nil {
			t.Fatalf("파싱 실패: %v", err)
		}
		deck.Slides = append(deck.Slides, &soml.Slide{
			Index: i, ID: id, Source: source, Notes: doc.Notes, Content: doc.Content,
		})
	}
	if err := core.Resolve(deck, nil).Err(); err != nil {
		t.Fatalf("외관 해석 실패: %v", err)
	}
	return deck
}

// liSteps는 산출물의 `<li>`마다 실린 스텝 번호다 (문서 순서). 속성이 없으면 "" —
// reveal을 쓰지 않은 항목의 마크업은 이 기능 전과 **같아야** 한다.
func liSteps(t *testing.T, out string) []string {
	t.Helper()
	tags := regexp.MustCompile(`<li [^>]*>`).FindAllString(out, -1)
	if len(tags) == 0 {
		t.Fatal("산출물에 <li>가 없다")
	}
	steps := make([]string, len(tags))
	for i, tag := range tags {
		if m := regexp.MustCompile(`data-balpyo-reveal="([^"]*)"`).FindStringSubmatch(tag); m != nil {
			steps[i] = m[1]
		}
	}
	return steps
}

// 스텝 번호는 마크업이 나른다 (ADR 0063 결정 5) — 값 없는 reveal은 그 슬라이드에서 자기
// 차례를 받고, 번호는 슬라이드마다 1부터 다시 센다.
func TestRevealStepsAreNumberedPerSlide(t *testing.T) {
	out := renderString(t, revealDeck(t,
		`  <ul>
    <li>먼저 보인다</li>
    <li reveal>둘째</li>
    <li reveal>셋째</li>
  </ul>
  <ol>
    <li reveal>넷째 — 스텝은 목록을 넘어 이어진다</li>
  </ol>`,
		`  <ul>
    <li reveal>다음 장의 첫 스텝이다</li>
  </ul>`))
	want := []string{"", "1", "2", "3", "1"}
	if got := liSteps(t, out); !equalStrings(got, want) {
		t.Errorf("스텝 = %v, want %v", got, want)
	}
}

// 번호를 준 reveal은 자기 차례를 덮어쓰고, 같은 번호를 받은 항목들은 함께 나타난다
// (ADR 0063 결정 2).
func TestRevealExplicitStepOverridesItsTurn(t *testing.T) {
	out := renderString(t, revealDeck(t,
		`  <ul>
    <li reveal="2">나중에</li>
    <li reveal="1">먼저</li>
    <li reveal="1">먼저와 함께</li>
    <li reveal>자기 차례는 넷이다</li>
  </ul>`))
	want := []string{"2", "1", "1", "4"}
	if got := liSteps(t, out); !equalStrings(got, want) {
		t.Errorf("스텝 = %v, want %v", got, want)
	}
}

// reveal을 쓰지 않은 덱의 마크업은 이 기능 전과 같다 — 속성이 없으면 렌더러가 아무것도
// 싣지 않는다 (ADR 0063 결과).
func TestRevealAbsentLeavesMarkupAlone(t *testing.T) {
	out := renderString(t, revealDeck(t, `  <ul>
    <li>하나</li>
    <li>둘</li>
  </ul>`))
	if strings.Contains(out, "data-balpyo-reveal") {
		t.Error("reveal 없는 덱의 산출물에 스텝 속성이 실렸다")
	}
	if got := liSteps(t, out); !equalStrings(got, []string{"", ""}) {
		t.Errorf("스텝 = %v, want 둘 다 없음", got)
	}
}

// 숨기는 곳은 **발표자 뷰 하나**다 (ADR 0063 결정 3): 문서 뷰·측정 뷰·오버뷰·인쇄는 전부
// 전개된 상태를 본다. 게이트 밖으로 새면 지오메트리 패스가 숨은 항목을 재고 그 좌표가
// PPTX 텍스트 박스가 된다 — 조용한 실패의 전형이다.
func TestRevealHidingIsPresenterOnly(t *testing.T) {
	css := viewerCSS()
	printAt := strings.Index(css, "@media print")
	if printAt < 0 {
		t.Fatal("뷰어 CSS에 인쇄 절이 없다")
	}
	gate := fmt.Sprintf("html[%s=%q]", ViewAttr, PresenterView)
	rules := revealRules(css)
	if len(rules) == 0 {
		t.Fatal("스텝을 숨기는 규칙이 없다")
	}
	hides := 0
	for _, rule := range rules {
		if rule.at > printAt {
			continue // 인쇄 절의 되돌리기는 다른 테스트가 본다.
		}
		if !strings.Contains(rule.selector, gate) {
			t.Errorf("스텝 선택자 %q가 발표자 뷰 게이트 밖에 있다", rule.selector)
		}
		if !strings.Contains(rule.body, "visibility") {
			continue
		}
		hides++
		// 숨기는 수단은 visibility다 (결정 4) — display: none은 등장마다 목록을 밀어 올린다.
		if strings.Contains(rule.body, "display: none") {
			t.Errorf("스텝을 display: none으로 숨긴다 (%q) — 등장마다 나머지 줄이 움직인다", rule.selector)
		}
	}
	if hides == 0 {
		t.Error("발표자 뷰에서 스텝을 숨기는 규칙이 없다")
	}
}

// revealRules는 CSS에서 스텝 속성을 겨누는 규칙 전부다 (선택자·선언·자리).
type revealRule struct {
	selector string
	body     string
	at       int
}

func revealRules(css string) []revealRule {
	var rules []revealRule
	for at := 0; ; {
		i := strings.Index(css[at:], "data-balpyo-reveal")
		if i < 0 {
			return rules
		}
		i += at
		// 선택자는 직전의 `{`·`}`·`;` 다음부터다 (중첩 @media 안에서도 같다).
		start := 0
		for _, sep := range []string{"{", "}", ";"} {
			if s := strings.LastIndex(css[:i], sep); s+1 > start {
				start = s + 1
			}
		}
		open := strings.Index(css[i:], "{")
		end := strings.Index(css[i:], "}")
		if open < 0 || end < 0 {
			return rules
		}
		rules = append(rules, revealRule{
			selector: strings.TrimSpace(css[start : i+open]),
			body:     css[i+open : i+end],
			at:       i,
		})
		at = i + len("data-balpyo-reveal")
	}
}

// 인쇄는 전개된 상태다 (ADR 0063 결정 3) — 발표 중에 Cmd+P를 누른 사람의 종이에 아직
// 나타나지 않은 항목이 빠지면 안 된다.
func TestPrintShowsEveryRevealStep(t *testing.T) {
	deck := revealDeck(t, `  <ul>
    <li reveal>하나</li>
  </ul>`)
	base := baseCSS(deck.Stage, contentFeatures(deck))
	printAt := strings.Index(base, "@media print")
	if printAt < 0 {
		t.Fatal("인쇄 절이 없다")
	}
	print := base[printAt:]
	if !strings.Contains(print, "data-balpyo-reveal") {
		t.Errorf("인쇄 CSS가 스텝 숨김을 되돌리지 않는다:\n%s", print)
	}
}

// 지금 스텝의 판정은 런타임의 것이다 (ADR 0063 결정 5) — CSS는 수를 비교할 수 없다.
func TestRuntimeDrivesRevealSteps(t *testing.T) {
	for _, want := range []string{"data-balpyo-reveal", "balpyo-viewer-revealed"} {
		if !strings.Contains(runtimeJS, want) {
			t.Errorf("런타임 JS에 계약 문자열 %q가 없다", want)
		}
		if !strings.Contains(viewerCSS(), want) {
			t.Errorf("뷰어 CSS에 계약 문자열 %q가 없다", want)
		}
	}
}

func equalStrings(a, b []string) bool {
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
