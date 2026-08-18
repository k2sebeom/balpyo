package web

import (
	"fmt"
	"strings"
	"testing"
)

// 런타임은 문서에 정확히 한 번 인라인된다 — 외부 참조가 생기면 자체 완결형이 깨지고,
// 두 번 실리면 컨트롤 바가 두 개 생긴다.
func TestRenderInlinesRuntimeOnce(t *testing.T) {
	out := renderString(t, testDeck())
	if got := strings.Count(out, "<script>"); got != 1 {
		t.Errorf("<script>가 %d번 나온다, want 1", got)
	}
	if strings.Contains(out, "<script src") {
		t.Error("런타임이 외부 참조로 실렸다")
	}
	if !strings.Contains(out, runtimeJS) {
		t.Error("런타임 JS가 문서에 없다")
	}
}

// 런타임 JS·CSS는 계약 문자열(뷰 속성·클래스)을 리터럴로 쓴다 (baseCSS가 .balpyo-slide를
// 리터럴로 쓰는 것과 같은 관계). 상수를 바꾸고 리터럴을 잊으면 지오메트리 패스가 지우는
// 속성과 런타임이 켜는 속성이 갈라져 측정이 발표자 뷰를 재게 된다 — 여기서 잡는다.
func TestRuntimeLiteralsMatchContract(t *testing.T) {
	if !strings.Contains(runtimeJS, fmt.Sprintf("root.setAttribute(%q, %q)", ViewAttr, PresenterView)) {
		t.Errorf("런타임이 켜는 뷰 속성이 계약(%s=%s)과 다르다", ViewAttr, PresenterView)
	}
	for _, want := range []string{
		fmt.Sprintf("querySelectorAll(%q)", SlideSelector()),
		fmt.Sprintf("querySelector(%q)", "."+NotesClass),
		fmt.Sprintf("%q", ViewerBarClass),
		fmt.Sprintf("%q", ViewerPrevClass),
		fmt.Sprintf("%q", ViewerNextClass),
		fmt.Sprintf("%q", ViewerCounterClass),
		fmt.Sprintf("%q", ViewerNotesToggleClass),
		fmt.Sprintf("%q", ViewerNotesClass),
	} {
		if !strings.Contains(runtimeJS, want) {
			t.Errorf("런타임 JS에 계약 문자열 %s가 없다", want)
		}
	}
	gate := fmt.Sprintf("html[%s=%q]", ViewAttr, PresenterView)
	if !strings.Contains(viewerCSS(), gate) {
		t.Errorf("발표자 뷰 CSS가 게이트 %s를 쓰지 않는다", gate)
	}
}

// 발표자 뷰 CSS는 전부 뷰 속성 게이트 아래에 있고, 문서 뷰(속성 없음)에서 뷰어 크롬은
// 그려지지 않는다 — 게이트 밖 규칙이 배치를 건드리면 지오메트리 패스가 그만큼 틀린 값을 잰다.
func TestPresenterCSSIsGated(t *testing.T) {
	out := renderString(t, testDeck())
	if !strings.Contains(out, "."+ViewerBarClass+", ."+ViewerNotesClass+" { display: none; }") {
		t.Error("문서 뷰에서 뷰어 크롬을 숨기는 규칙이 없다")
	}
	// 슬라이드 배치를 바꾸는 규칙(position/transform/visibility)은 게이트 안에서만 나온다.
	gate := fmt.Sprintf("html[%s=%q]", ViewAttr, PresenterView)
	for _, decl := range []string{"translate(-50%, -50%)", "visibility: hidden;"} {
		i := strings.Index(out, decl)
		if i < 0 {
			t.Fatalf("발표자 뷰 규칙 %q가 없다", decl)
		}
		// 선언이 속한 규칙의 선택자를 찾는다 — 직전 `{`까지 거슬러 올라간 선택자에 게이트가
		// 있어야 한다.
		block := out[:i]
		open := strings.LastIndex(block, "{")
		selStart := strings.LastIndex(block[:open], "}") + 1
		if sel := block[selStart:open]; !strings.Contains(sel, gate) {
			t.Errorf("%q가 게이트 없는 선택자 %q에 있다", decl, strings.TrimSpace(sel))
		}
	}
}

// 뷰 전환 표현식은 같은 속성을 다룬다 — 지오메트리 패스(DocumentViewExpr)와 발표자 뷰
// 테스트(PresenterViewExpr)가 계약의 양쪽이다.
func TestViewExprsShareAttr(t *testing.T) {
	doc, pres := DocumentViewExpr(), PresenterViewExpr()
	for name, expr := range map[string]string{"DocumentViewExpr": doc, "PresenterViewExpr": pres} {
		if !strings.Contains(expr, fmt.Sprintf("%q", ViewAttr)) {
			t.Errorf("%s가 ViewAttr를 쓰지 않는다: %s", name, expr)
		}
	}
	if !strings.Contains(pres, fmt.Sprintf("%q", PresenterView)) {
		t.Errorf("PresenterViewExpr가 PresenterView 값을 쓰지 않는다: %s", pres)
	}
}

// 런타임은 덱 데이터를 보간하지 않는 고정 문자열이다 — 저자 텍스트가 스크립트로 새는
// 경로가 없어야 하고, 골든이 덱마다 다른 JS를 담지 않아야 한다.
func TestRuntimeIsStatic(t *testing.T) {
	a := renderString(t, testDeck())
	deck := testDeck()
	deck.Title = "다른 덱"
	deck.Slides = deck.Slides[:1]
	b := renderString(t, deck)
	scriptOf := func(out string) string {
		start := strings.Index(out, "<script>")
		end := strings.Index(out, "</script>")
		if start < 0 || end < 0 {
			t.Fatal("런타임 스크립트가 없다")
		}
		return out[start:end]
	}
	if scriptOf(a) != scriptOf(b) {
		t.Error("런타임 JS가 덱에 따라 달라진다")
	}
}
