package web

import (
	"html"
	"regexp"
	"strings"
	"testing"
)

// 하이라이팅은 **렌더러를 손대지 않는다**: 값 층이 `<pre>`의 문자 데이터를 합성
// 인라인 `<span>`으로 컴파일하므로, 웹은 이미 있는 인라인 경로(writeInline)로 색 있는 런을
// 낸다. 이 파일은 그 계약이 **산출물에서** 지켜지는지 본다 — 색은 토큰 변수이고, 글자는
// 하나도 잃지 않으며, `lang`이 없는 `<pre>`는 예전과 같은 텍스트 하나다.

var htmlTagRE = regexp.MustCompile(`<[^>]*>`)

// preInner는 산출물의 첫 `<pre …>`와 `</pre>` 사이다.
func preInner(t *testing.T, out string) string {
	t.Helper()
	start := strings.Index(out, "<pre ")
	if start < 0 {
		t.Fatalf("산출물에 <pre>가 없다:\n%s", stageOf(t, out))
	}
	open := strings.Index(out[start:], ">")
	end := strings.Index(out[start:], "</pre>")
	if open < 0 || end < 0 {
		t.Fatalf("<pre>가 닫히지 않았다:\n%s", stageOf(t, out))
	}
	return out[start+open+1 : start+end]
}

// preText는 `<pre>` 안쪽에서 태그를 벗긴 글자다 — 하이라이팅이 글자를 잃거나 더하지 않았음을
// 보는 눈이다 (토큰 스트림의 왕복 계약).
func preText(t *testing.T, out string) string {
	t.Helper()
	return html.UnescapeString(htmlTagRE.ReplaceAllString(preInner(t, out), ""))
}

// 종류마다 `<span>`이 나오고 색은 **토큰 변수**다 — 리터럴 hex가 나오면 테마를 갈아도 코드
// 색만 남아 덱이 두 팔레트를 갖는다 (PPTX 짝은 schemeClr 참조 보존).
func TestRenderPreHighlightsWithColorVars(t *testing.T) {
	const code = "func main() {\n  n := 42 // 예열\n}"
	out := renderString(t, subsetDeck(t, `<pre lang="go">`+code+`</pre>`))
	inner := preInner(t, out)

	if !strings.Contains(inner, `<span style="`) {
		t.Fatalf("합성 런이 <span>으로 나오지 않았다:\n%s", inner)
	}
	for _, want := range []string{
		"var(--balpyo-color-accent6)", // keyword → accent6
		"var(--balpyo-color-accent4)", // number → accent4
		"var(--balpyo-color-muted)",   // comment → muted
	} {
		if !strings.Contains(inner, want) {
			t.Errorf("색 %q가 없다:\n%s", want, inner)
		}
	}
	if strings.Contains(inner, "color: #") {
		t.Errorf("코드 색이 리터럴 hex로 나왔다 — 토큰 참조가 끊겼다:\n%s", inner)
	}
	// 글자는 그대로다 — 태그를 벗기면 원문이 남는다.
	if got := preText(t, out); got != code {
		t.Errorf("하이라이팅이 글자를 바꿨다:\n%q\nwant %q", got, code)
	}
}

// `lang`이 없는 `<pre>`는 런으로 쪼개지지 않는다 — 하이라이팅은 저자가 켜는 것이다.
func TestRenderPreWithoutLangHasNoSpans(t *testing.T) {
	out := renderString(t, subsetDeck(t, "<pre>func main() {\n  println(1)\n}</pre>"))
	if inner := preInner(t, out); strings.Contains(inner, "<span") {
		t.Errorf("lang 없는 pre에 런이 생겼다:\n%s", inner)
	}
}
