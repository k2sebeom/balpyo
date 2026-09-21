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
		for _, sel := range selectorsOf(t, out, decl) {
			if !strings.Contains(sel, gate) {
				t.Errorf("%q가 게이트 없는 선택자 %q에 있다", decl, sel)
			}
		}
	}
}

// selectorsOf는 선언이 나오는 모든 규칙의 선택자다 — 직전 `{`까지 거슬러 올라간다.
// 선언이 하나도 없으면 실패한다 (없는 규칙을 "게이트되어 있다"로 통과시키지 않는다).
func selectorsOf(t testing.TB, css, decl string) []string {
	t.Helper()
	var sels []string
	for rest, offset := css, 0; ; {
		i := strings.Index(rest, decl)
		if i < 0 {
			break
		}
		block := css[:offset+i]
		open := strings.LastIndex(block, "{")
		selStart := strings.LastIndex(block[:open], "}") + 1
		if at := strings.LastIndex(block[:open], "{"); at > selStart {
			// 중첩 블록(@media 안의 규칙)이면 여는 괄호 뒤부터가 선택자다.
			selStart = at + 1
		}
		sels = append(sels, strings.TrimSpace(block[selStart:open]))
		offset += i + len(decl)
		rest = css[offset:]
	}
	if len(sels) == 0 {
		t.Fatalf("선언 %q가 CSS에 없다", decl)
	}
	return sels
}

// 뷰 전환 표현식은 같은 속성을 다룬다 — 지오메트리 패스(DocumentViewExpr·MeasureViewExpr)와
// 발표자 뷰 테스트(PresenterViewExpr)가 계약의 양쪽이다.
func TestViewExprsShareAttr(t *testing.T) {
	doc, pres, measure := DocumentViewExpr(), PresenterViewExpr(), MeasureViewExpr()
	for name, expr := range map[string]string{
		"DocumentViewExpr": doc, "PresenterViewExpr": pres, "MeasureViewExpr": measure,
	} {
		if !strings.Contains(expr, fmt.Sprintf("%q", ViewAttr)) {
			t.Errorf("%s가 ViewAttr를 쓰지 않는다: %s", name, expr)
		}
	}
	if !strings.Contains(pres, fmt.Sprintf("%q", PresenterView)) {
		t.Errorf("PresenterViewExpr가 PresenterView 값을 쓰지 않는다: %s", pres)
	}
	if !strings.Contains(measure, fmt.Sprintf("%q", MeasureView)) {
		t.Errorf("MeasureViewExpr가 MeasureView 값을 쓰지 않는다: %s", measure)
	}
	// 뷰 값은 서로 달라야 한다 — 측정 뷰가 발표자 뷰와 같은 값이면 측정이 겹쳐진 슬라이드를
	// 재고, 문서 뷰(값 없음)와 같으면 회전이 꺼지지 않는다. 오버뷰도 같은 속성에 살므로
	// (ADR 0062) 같은 자리에서 판정한다.
	seen := map[string]string{}
	for name, value := range map[string]string{
		"PresenterView": PresenterView, "MeasureView": MeasureView, "OverviewView": OverviewView,
	} {
		if value == "" {
			t.Errorf("%s가 비었다 — 값 없음은 문서 뷰다", name)
		}
		if other, dup := seen[value]; dup {
			t.Errorf("%s와 %s가 같은 값 %q를 쓴다", name, other, value)
		}
		seen[value] = name
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

// 뷰어 자산은 파일(go:embed)이 되었지만 전달은 그대로 인라인이다 — 문서를 그 자리에서
// 끝내는 문자열이 자산에 들어가면 산출물이 깨진다. 자산이 Go 문자열에서 파일로 나가면서
// 사람의 눈 밖으로 나가는 유일한 위험이므로 게이트에 둔다 (ADR 0062 결정 2).
func TestViewerAssetsAreInlineSafe(t *testing.T) {
	assets := map[string]string{"runtime.js": runtimeJS, "viewer.css": viewerCSS()}
	for name, asset := range assets {
		if strings.TrimSpace(asset) == "" {
			t.Errorf("%s 자산이 비었다 — embed 경로가 어긋났다", name)
		}
		for _, bad := range []string{"</script", "</style", "<!--", "]]>"} {
			if strings.Contains(strings.ToLower(asset), bad) {
				t.Errorf("%s가 인라인을 끝내는 문자열 %q를 담았다", name, bad)
			}
		}
	}
}

// 오버뷰는 네 번째 뷰 값이다 (ADR 0062 결정 3) — 새 페이지도 아니고 발표자 뷰 안의
// 클래스도 아니다. 뷰 전환의 구현이 하나면 측정·발표·오버뷰가 같은 파일을 본다.
func TestOverviewIsAViewValue(t *testing.T) {
	gate := fmt.Sprintf("html[%s=%q]", ViewAttr, OverviewView)
	if !strings.Contains(viewerCSS(), gate) {
		t.Errorf("오버뷰 CSS가 게이트 %s를 쓰지 않는다", gate)
	}
	for _, want := range []string{
		fmt.Sprintf("%q", OverviewView),
		fmt.Sprintf("%q", ViewerOverviewClass),
	} {
		if !strings.Contains(runtimeJS, want) {
			t.Errorf("런타임 JS에 계약 문자열 %s가 없다", want)
		}
	}
}

// 오버뷰는 zoom으로 줄이고 감싸는 요소를 만들지 않는다 (ADR 0062 결정 4) — 격자를
// transform으로 만들려면 타일마다 래퍼가 필요하고, 그 요소는 지오메트리 패스가 읽는 부모
// 사슬을 바꾼다. zoom·격자 선언이 게이트 밖으로 새면 문서 뷰의 배치가 줄어든다.
func TestOverviewCSSIsGated(t *testing.T) {
	out := renderString(t, testDeck())
	attrGate := fmt.Sprintf("html[%s", ViewAttr)
	for _, decl := range []string{"zoom:", "display: grid"} {
		for _, sel := range selectorsOf(t, out, decl) {
			if !strings.Contains(sel, attrGate) {
				t.Errorf("%q가 게이트 없는 선택자 %q에 있다", decl, sel)
			}
		}
	}
	// 격자는 런타임이 계산한 열 수와 배율을 받는다 — 슬라이드 수와 창 크기에 따라 전부
	// 화면에 들어와야 하고, 그 계산은 CSS가 할 수 없다.
	for _, v := range []string{"--balpyo-overview-columns", "--balpyo-overview-zoom"} {
		if !strings.Contains(out, v) {
			t.Errorf("오버뷰 CSS에 %s가 없다", v)
		}
		if !strings.Contains(runtimeJS, v) {
			t.Errorf("런타임이 %s를 설정하지 않는다", v)
		}
	}
}

// 인쇄는 문서 뷰의 그림이고, 어느 뷰에서 눌러도 그렇다 (ADR 0062 결정 6).
//
// 사람이 Cmd+P를 누르는 시점은 발표자 뷰다. 그래서 인쇄 규칙은 base 층의 **마지막**에
// 오고, 뷰 게이트와 **같은 명시도**의 선택자로 되돌린다 — `!important`는 그것으로 부족한
// 자리에만 쓴다.
func TestPrintCSSRestoresTheDocumentView(t *testing.T) {
	deck := testDeck()
	stage := deck.Stage
	base := baseCSS(stage, contentFeatures(deck))

	for _, want := range []string{
		"@media print",
		fmt.Sprintf("@page { size: %dpx %dpx; margin: 0; }", stage.Width, stage.Height),
		"break-after: page",
	} {
		if !strings.Contains(base, want) {
			t.Errorf("인쇄 CSS에 %q가 없다", want)
		}
	}
	if strings.Contains(printCSS(stage), "!important") {
		t.Error("인쇄 CSS가 !important를 쓴다 — 같은 명시도로 뒤에 두면 충분하다")
	}
	// 뷰가 켜진 상태를 되돌리는 선택자는 값을 묻지 않는다 (발표자 뷰든 오버뷰든 같이 잡힌다).
	if !strings.Contains(printCSS(stage), fmt.Sprintf("html[%s] ", ViewAttr)) {
		t.Errorf("인쇄 CSS가 뷰 게이트를 같은 명시도로 되돌리지 않는다:\n%s", printCSS(stage))
	}
	// base 층 안, 그리고 뷰 규칙 **뒤**에 온다.
	printAt := strings.Index(base, "@media print")
	presenterAt := strings.LastIndex(base, fmt.Sprintf("html[%s=%q]", ViewAttr, PresenterView))
	if printAt < presenterAt {
		t.Error("인쇄 CSS가 발표자 뷰 규칙보다 앞에 있다 — 되돌리기가 이기지 못한다")
	}
	// 층 밖으로 새면 테마·저자 CSS가 인쇄 규칙을 덮는다.
	layerAt := strings.Index(base, "@layer balpyo.base {")
	if layerAt < 0 {
		t.Fatal("base 층 선언이 없다")
	}
	if layerAt > printAt {
		t.Error("인쇄 CSS가 base 층 밖에 있다")
	}
}
