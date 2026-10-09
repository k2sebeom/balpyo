package web

import (
	"fmt"
	"strings"
	"testing"

	"github.com/k2sebeom/balpyo/core"
	"github.com/k2sebeom/balpyo/soml"
)

// subsetDeck는 서브셋 SOML 원문을 파이프라인과 같은 순서(파싱 → 해석)로 덱으로 만든다.
// 손으로 만든 트리 대신 원문을 쓰는 이유는 fixture_test와 같다 — 파서·해석기가 실제로
// 채우는 모양과 렌더러가 기대하는 모양이 갈라지는 순간을 여기서 잡는다.
func subsetDeck(t *testing.T, body string) *soml.Deck {
	t.Helper()
	const source = "slides/001_subset.slide"
	doc, diags := soml.Parse(source, "<slide id=\"subset\">\n"+body+"\n</slide>\n")
	if err := diags.Err(); err != nil {
		t.Fatalf("파싱 실패: %v", err)
	}
	stage, err := soml.Aspect16x9.Stage()
	if err != nil {
		t.Fatal(err)
	}
	deck := &soml.Deck{
		Title: "서브셋", Lang: "ko", Aspect: soml.Aspect16x9, Stage: stage,
		Slides: []*soml.Slide{{
			Index: 0, ID: "subset", Source: source, Notes: doc.Notes, Content: doc.Content,
		}},
	}
	if err := core.Resolve(deck, nil).Err(); err != nil {
		t.Fatalf("외관 해석 실패: %v", err)
	}
	return deck
}

// 블록 태그는 저자가 쓴 이름 그대로 시맨틱 태그로 나온다 — 단, 외관은 전부 인라인 style과
// 엔진 CSS가 명시하므로 브라우저 기본 스타일이 메트릭에 개입할 자리가 없다.
func TestRenderBlockTagMapping(t *testing.T) {
	out := renderString(t, subsetDeck(t, `
  <h1>제목</h1>
  <h2>소제목</h2>
  <h3>부제목</h3>
  <p>본문</p>
  <blockquote>인용</blockquote>`))
	for i, tc := range []struct {
		tag  string
		size string
	}{
		{"h1", "56"}, {"h2", "36"}, {"h3", "28"}, {"p", "24"}, {"blockquote", "24"},
	} {
		open := openTagOf(t, out, fmt.Sprintf("s0.%d", i))
		// 역할·태그 기본값이 해석되어 인라인 style로 나온다 — 태그별 크기가 그 증거다.
		prefix := fmt.Sprintf("<%s class=\"balpyo-text\" ", tc.tag)
		if !strings.HasPrefix(open, prefix) || !strings.Contains(open, "font-size: "+tc.size+"px") {
			t.Errorf("<%s>의 여는 태그 = %s, want %q로 시작 + font-size %spx", tc.tag, open, prefix, tc.size)
		}
	}
}

// 리스트: 컨테이너는 식별자만 갖고(외관 없음 — Appearance nil), 항목은 블록처럼 확정
// 스타일을 갖는다.
func TestRenderListMarkup(t *testing.T) {
	out := renderString(t, subsetDeck(t, `
  <ul>
    <li>첫째</li>
    <li>둘째</li>
  </ul>
  <ol>
    <li>먼저</li>
  </ol>`))
	for _, want := range []string{
		`<ul class="balpyo-list" data-balpyo-id="s0.0" data-balpyo-el="ul">`,
		`<ol class="balpyo-list" data-balpyo-id="s0.1" data-balpyo-el="ol">`,
		`<li class="balpyo-text" data-balpyo-id="s0.0.0" data-balpyo-el="li" style="`,
		`<li class="balpyo-text" data-balpyo-id="s0.0.1" data-balpyo-el="li" style="`,
		`<li class="balpyo-text" data-balpyo-id="s0.1.0" data-balpyo-el="li" style="`,
		"</ul>", "</ol>", "</li>",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("리스트 마크업 %q가 없다", want)
		}
	}
	// li는 body 역할 등가다 (soml.TagTextRole) — 확정 스타일이 인라인 style로 나온다.
	if !strings.Contains(out, `data-balpyo-el="li" style="font-family: var(--balpyo-font-body); font-size: 24px;`) {
		t.Error("li의 확정 스타일(body 등가)이 인라인 style로 나오지 않았다")
	}
}

// 리스트 배치 CSS는 엔진 상수에서 나온다 — PPTX의 marL/indent가 같은 상수를
// import하므로, CSS가 상수와 어긋나면 두 타깃의 들여쓰기가 갈라진다.
func TestRenderListCSSComesFromConstants(t *testing.T) {
	out := renderString(t, subsetDeck(t, "<ul><li>항목</li></ul>"))
	css := between(t, out, "<style>", "</style>")
	for _, want := range []string{
		"list-style: none;",
		fmt.Sprintf("padding-left: %dpx;", ListTextIndent),
		// 행잉 인덴트는 음수 text-indent이고 마커는 흐름 안의 inline-block이다 — 마커가
		// 문단과 **함께** 정렬되어야 `<ul align="center">`가 PPTX와 갈라지지 않는다.
		fmt.Sprintf("text-indent: -%dpx;", ListMarkerHang),
		fmt.Sprintf("width: %dpx;", ListMarkerHang),
		"display: inline-block;",
		fmt.Sprintf("margin: 0 0 %dpx;", ListItemGap),
		fmt.Sprintf("content: %q;", ListBulletChar),
		`content: counter(balpyo-item) ".";`,
		"counter-reset: balpyo-item;",
		"counter-increment: balpyo-item;",
	} {
		if !strings.Contains(css, want) {
			t.Errorf("리스트 CSS에 %q가 없다", want)
		}
	}
}

// 인라인 태그는 대응 HTML 태그로 나오되, 확정 TextStyle을 **전부 명시**한 인라인 style을
// 갖는다 — strong{font-weight:bolder} 같은 UA 규칙이 개입할 자리를 남기지 않는다.
func TestRenderInlineRunStyles(t *testing.T) {
	out := renderString(t, subsetDeck(t,
		`<p>배포 <strong>자동화</strong>와 <em>관측성</em>, <code>otel</code>, `+
			`<a href="https://example.com/docs">레퍼런스</a>, <span color="accent1">강조</span></p>`))
	for _, tc := range []struct {
		tag   string
		wants []string
	}{
		{"strong", []string{"font-weight: 700", "font-style: normal", "text-decoration: none"}},
		{"em", []string{"font-weight: 400", "font-style: italic", "text-decoration: none"}},
		{"code", []string{"font-family: var(--balpyo-font-mono)", "font-size: 24px"}},
		{"a", []string{"text-decoration: underline", "color: var(--balpyo-color-accent1)"}},
		{"span", []string{"color: var(--balpyo-color-accent1)", "font-style: normal"}},
	} {
		start := strings.Index(out, "<"+tc.tag+" ")
		if start < 0 {
			t.Errorf("<%s>가 산출물에 없다", tc.tag)
			continue
		}
		open := out[start:]
		open = open[:strings.Index(open, ">")+1]
		for _, want := range tc.wants {
			if !strings.Contains(open, want) {
				t.Errorf("<%s>의 style에 %q가 없다: %s", tc.tag, want, open)
			}
		}
		// 전부 명시의 최소 계약: 메트릭을 정하는 선언이 태그마다 다 있어야 한다.
		for _, decl := range []string{"font-family:", "font-size:", "font-weight:", "line-height:",
			"font-style:", "text-decoration:", "color:"} {
			if !strings.Contains(open, decl) {
				t.Errorf("<%s>의 style에 %s 선언이 없다: %s", tc.tag, decl, open)
			}
		}
	}
}

// <br/>은 <br>로 — 서식도 식별자도 없는 흐름의 분절점이고, 앞뒤에 공백을 만들면 안 된다.
func TestRenderBreak(t *testing.T) {
	out := renderString(t, subsetDeck(t, "<p>첫 줄<br/>둘째 줄</p>"))
	if !strings.Contains(out, "첫 줄<br>둘째 줄") {
		t.Errorf("<br> 앞뒤로 공백이 생겼거나 태그가 다르다:\n%s", stageOf(t, out))
	}
}

// 인라인 경계 공백 계약이 렌더까지 통과한다: `파이프라인 <strong>자동화</strong>와`의
// 한 칸은 내용이다 — 렌더러가 태그 앞뒤에 개행·들여쓰기를 더하면 그 공백이 글자 사이에 낀다.
func TestRenderPreservesInlineBoundarySpace(t *testing.T) {
	out := renderString(t, subsetDeck(t, "<p>배포 파이프라인 <strong>자동화</strong>와 관측성</p>"))
	start := strings.Index(out, "배포 파이프라인 <strong")
	if start < 0 {
		t.Fatalf("경계 공백이 보존되지 않았다:\n%s", stageOf(t, out))
	}
	if !strings.Contains(out, "</strong>와 관측성") {
		t.Errorf("닫는 태그 뒤에 공백이 생겼다:\n%s", stageOf(t, out))
	}
}

// pre는 원문이 그대로 흐른다: white-space: pre + 줄바꿈 없음. wrap을 켜면 PPTX와 줄 수가
// 갈라진다 (PPTX 짝은 bodyPr@wrap).
func TestRenderPrePreservesWhitespace(t *testing.T) {
	out := renderString(t, subsetDeck(t, "<pre>\nfunc main() {\n\tgo build()  // 연속  공백\n}\n</pre>"))
	if !strings.Contains(out, "white-space: pre") {
		t.Error("pre의 style에 white-space: pre가 없다")
	}
	// 탭은 값 층이 테마 폭(minimal은 2)으로 **펴서** 보낸다 — CSS tab-size와
	// OOXML a:defTabSz는 같은 규칙이 아니어서, 탭을 그대로 흘리면 두 산출물의 줄이 갈라진다.
	// 연속 공백은 그대로다 — 접히지 않는 것이 pre의 계약이다.
	if !strings.Contains(out, "func main() {\n  go build()  // 연속  공백\n}") {
		t.Errorf("pre 안의 개행·탭·연속 공백이 보존되지 않았다:\n%s", stageOf(t, out))
	}
	// 파서가 양 끝 개행을 벗겼으므로 여는 태그 직후에 바로 내용이 온다.
	if !strings.Contains(out, ">func main() {") {
		t.Error("여는 태그와 내용 사이에 공백이 끼었다")
	}
}

// 파서가 벗기고 남은 선두 개행은 내용이다 — HTML 파서도 <pre> 직후 개행 하나를 벗기므로,
// 지키려면 렌더러가 개행 하나를 덧대야 한다.
func TestRenderPreProtectsLeadingNewline(t *testing.T) {
	style := soml.TextStyle{Role: soml.RoleCode, Font: soml.FontMono, Size: 20, Weight: 400,
		LineHeight: 1.5, Color: soml.TokenColor("fg"), Align: soml.AlignStart}
	deck := contentDeck()
	deck.Slides[0].Content = []soml.Node{&soml.Element{
		Name:       "pre",
		Children:   []soml.Node{&soml.Text{Value: "\n둘째 줄부터"}},
		Appearance: &soml.Appearance{Text: &style},
	}}
	out := renderString(t, deck)
	if !strings.Contains(out, ">\n\n둘째 줄부터</pre>") {
		t.Errorf("선두 개행이 보호되지 않았다:\n%s", stageOf(t, out))
	}
}

// 원문의 문자 참조는 **한 번만** 이스케이프되어 나간다: `&lt;`(원문) →
// `<`(모델) → `&lt;`(HTML). 두 번 이스케이프되면 화면에 `&lt;`가 그대로 보인다 — SOML을
// 소개하는 덱이 자기 문법을 못 띄운다.
func TestRenderCharRefsEscapeOnce(t *testing.T) {
	out := renderString(t, subsetDeck(t, `
  <p>&lt;slide id=&quot;x&quot;&gt; &amp; a &lt; b</p>
  <pre><![CDATA[<text role="title">제목</text>]]></pre>
  <p>비교: a &lt; b, 리터럴: R&D</p>`))
	for _, want := range []string{
		`&lt;slide id=&#34;x&#34;&gt; &amp; a &lt; b`,
		`&lt;text role=&#34;title&#34;&gt;제목&lt;/text&gt;`,
		`비교: a &lt; b, 리터럴: R&amp;D`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("%q가 없다:\n%s", want, stageOf(t, out))
		}
	}
	if strings.Contains(out, "&amp;lt;") {
		t.Errorf("문자 참조가 두 번 이스케이프됐다:\n%s", stageOf(t, out))
	}
}
