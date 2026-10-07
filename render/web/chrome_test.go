package web

import (
	"strings"
	"testing"

	"github.com/k2sebeom/balpyo/soml"
)

// captionStyle은 크롬이 받는 외관이다 — core가 캡션 역할에서 확정해 넘긴다 (ADR 0045).
func captionStyle() soml.TextStyle {
	return soml.TextStyle{Role: soml.RoleCaption, Font: soml.FontBody, Size: 16, Weight: 400,
		LineHeight: 1.4, Color: soml.TokenColor("muted"), Align: soml.AlignStart}
}

// chromeDeck은 확정된 크롬이 실린 덱이다 — 렌더러는 제외 규칙도 번호 계산도 하지 않는다.
func chromeDeck() *soml.Deck {
	stage, _ := soml.Aspect16x9.Stage()
	return &soml.Deck{
		Title:  "크롬",
		Lang:   "ko",
		Aspect: soml.Aspect16x9,
		Stage:  stage,
		Theme:  testTheme(),
		Slides: []*soml.Slide{
			{Index: 0, ID: "cover", Source: "slides/001_cover.slide"},
			{Index: 1, ID: "body", Source: "slides/002_body.slide", Chrome: []soml.ChromeItem{
				{Kind: soml.ChromeFooter, Text: "2026 Q3 · 내부용", Align: soml.AlignStart, Style: captionStyle()},
				{Kind: soml.ChromePageNumber, Text: "2", Align: soml.AlignEnd, Style: captionStyle()},
			}},
		},
	}
}

// 크롬은 **스테이지의 형제**다 (ADR 0045): 스테이지 안에 넣으면 흐름에 끼어 본문 높이를
// 먹고, 지오메트리 패스가 재는 Ref가 되어 저자가 고칠 수 없는 배치 경고가 나온다.
func TestChromeIsStageSibling(t *testing.T) {
	out := renderString(t, chromeDeck())
	slide := between(t, out, `id="body"`, "</section>")
	stageEnd := strings.Index(slide, "</div>")
	chrome := strings.Index(slide, ChromeClass)
	if chrome < 0 {
		t.Fatalf("크롬이 나오지 않았다:\n%s", slide)
	}
	if chrome < stageEnd {
		t.Errorf("크롬이 스테이지 안에 있다:\n%s", slide)
	}

	// 크롬이 없는 슬라이드에는 아무것도 붙지 않는다.
	cover := between(t, out, `id="cover"`, "</section>")
	if strings.Contains(cover, ChromeClass) {
		t.Errorf("크롬 없는 슬라이드에 크롬이 나왔다:\n%s", cover)
	}
}

// 마크업은 확정 모델을 그대로 옮긴다 — kind는 속성으로, 붙는 쪽은 클래스로, 외관은 인라인
// 선언으로. kind를 속성으로 내는 이유는 지오메트리 패스가 어느 rect가 무엇인지 알아야
// PPTX가 같은 자리에 같은 도형을 놓는다는 것이다.
func TestChromeMarkup(t *testing.T) {
	out := renderString(t, chromeDeck())
	slide := between(t, out, `id="body"`, "</section>")
	for _, want := range []string{
		ChromeKindAttr + `="footer"`,
		ChromeKindAttr + `="pageNumber"`,
		"balpyo-chrome-start",
		"balpyo-chrome-end",
		">2<",
		">2026 Q3 · 내부용<",
		"font-size: 16px",
	} {
		if !strings.Contains(slide, want) {
			t.Errorf("크롬 마크업에 %q가 없다:\n%s", want, slide)
		}
	}
	// 순서는 확정 모델의 순서다 (footer → pageNumber).
	if strings.Index(slide, `="footer"`) > strings.Index(slide, `="pageNumber"`) {
		t.Errorf("크롬 순서가 확정 모델과 다르다:\n%s", slide)
	}
}

// 크롬 글자는 설정에서 오므로 이스케이프해야 한다 — `footer`는 저자가 쓰는 값이다.
func TestChromeTextIsEscaped(t *testing.T) {
	deck := chromeDeck()
	deck.Slides[1].Chrome = []soml.ChromeItem{
		{Kind: soml.ChromeFooter, Text: `<b>A & B</b>`, Align: soml.AlignCenter, Style: captionStyle()},
	}
	out := renderString(t, deck)
	if strings.Contains(out, "<b>A & B</b>") {
		t.Error("크롬 글자가 이스케이프되지 않았다")
	}
	if !strings.Contains(out, "&lt;b&gt;A &amp; B&lt;/b&gt;") {
		t.Error("이스케이프된 크롬 글자가 없다")
	}
	if !strings.Contains(out, "balpyo-chrome-center") {
		t.Error("center 정렬 클래스가 없다")
	}
}
