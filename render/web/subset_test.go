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
	if err := core.Resolve(deck).Err(); err != nil {
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
		open := fmt.Sprintf("<%s class=\"balpyo-text\" data-balpyo-id=\"s0.%d\" data-balpyo-el=%q style=\"",
			tc.tag, i, tc.tag)
		if !strings.Contains(out, open) {
			t.Errorf("<%s>의 여는 태그 %q가 없다", tc.tag, open)
		}
		if !strings.Contains(out, "</"+tc.tag+">") {
			t.Errorf("<%s>가 닫히지 않았다", tc.tag)
		}
		// 역할·태그 기본값이 해석되어 인라인 style로 나오는지 — 태그별 크기가 그 증거다.
		start := strings.Index(out, open)
		if start < 0 {
			continue
		}
		line := out[start:]
		line = line[:strings.Index(line, ">")+1]
		if want := "font-size: " + tc.size + "px"; !strings.Contains(line, want) {
			t.Errorf("<%s>의 style에 %q가 없다: %s", tc.tag, want, line)
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

// 리스트 배치 CSS는 엔진 상수에서 나온다 — PPTX의 marL/indent(항목 07)가 같은 상수를
// import하므로, CSS가 상수와 어긋나면 두 타깃의 들여쓰기가 갈라진다.
func TestRenderListCSSComesFromConstants(t *testing.T) {
	out := renderString(t, subsetDeck(t, "<ul><li>항목</li></ul>"))
	css := between(t, out, "<style>", "</style>")
	for _, want := range []string{
		"list-style: none;",
		fmt.Sprintf("padding-left: %dpx;", ListTextIndent),
		fmt.Sprintf("left: %dpx;", ListTextIndent-ListMarkerHang),
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

// 측정 대상 계약: data-balpyo-id는 블록 종류(text·블록·리스트 컨테이너·li)에만 붙는다.
// 여러 줄에 걸친 인라인의 rect는 뜻이 없고, PPTX에서 인라인은 박스가 아니라 런이다.
func TestRenderInlineElementsCarryNoID(t *testing.T) {
	out := renderString(t, subsetDeck(t,
		`<p><strong>굵게 <em>기울여</em></strong> <a href="https://example.com">링크</a><br/>다음 줄</p>
  <ul><li><code>코드</code> 항목</li></ul>`))
	// 블록 종류만 센다: p, ul, li — 3개.
	if got, want := strings.Count(out, "data-balpyo-id="), 3; got != want {
		t.Errorf("식별자 개수 = %d, want %d (블록 종류에만 붙는다)", got, want)
	}
	for _, tag := range []string{"strong", "em", "code", "a"} {
		if strings.Contains(out, "<"+tag+" data-balpyo-id") || strings.Contains(out, "<"+tag+" "+IDAttr) {
			t.Errorf("인라인 <%s>에 식별자가 붙었다", tag)
		}
	}
}

// href는 원문 그대로 싣되 HTML 속성으로 이스케이프한다 — &가 그대로 나가면 파싱에 따라
// 엔티티로 읽혀 목적지가 조용히 바뀐다.
func TestRenderEscapesHref(t *testing.T) {
	out := renderString(t, subsetDeck(t, `<p><a href="https://example.com/?a=1&b=2">링크</a></p>`))
	if !strings.Contains(out, `href="https://example.com/?a=1&amp;b=2"`) {
		t.Error("href의 &가 이스케이프되지 않았다")
	}
	if strings.Contains(out, `href="https://example.com/?a=1&b=2"`) {
		t.Error("href가 원문 그대로 나갔다")
	}
}

// <br/>은 <br>로 — 서식도 식별자도 없는 흐름의 분절점이고, 앞뒤에 공백을 만들면 안 된다.
func TestRenderBreak(t *testing.T) {
	out := renderString(t, subsetDeck(t, "<p>첫 줄<br/>둘째 줄</p>"))
	if !strings.Contains(out, "첫 줄<br>둘째 줄") {
		t.Errorf("<br> 앞뒤로 공백이 생겼거나 태그가 다르다:\n%s", stageOf(t, out))
	}
}

// 인라인 경계 공백 계약(항목 01)이 렌더까지 통과한다: `파이프라인 <strong>자동화</strong>와`의
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
// 갈라진다 (PPTX 짝은 항목 06의 bodyPr@wrap).
func TestRenderPrePreservesWhitespace(t *testing.T) {
	out := renderString(t, subsetDeck(t, "<pre>\nfunc main() {\n\tgo build()  // 연속  공백\n}\n</pre>"))
	if !strings.Contains(out, "white-space: pre") {
		t.Error("pre의 style에 white-space: pre가 없다")
	}
	if !strings.Contains(out, "func main() {\n\tgo build()  // 연속  공백\n}") {
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

// 리스트·인라인이 섞인 텍스트도 이스케이프된다 — 저자 텍스트가 태그로 새면 안 된다.
// (SOML은 엔티티를 해석하지 않으므로 원문에 `<`를 실을 수 없다 — 트리를 손으로 만든다.)
func TestRenderEscapesSubsetText(t *testing.T) {
	deck := contentDeck()
	style := bodyStyle()
	li := &soml.Element{
		Name:       "li",
		Appearance: &soml.Appearance{Text: &style},
		Children:   []soml.Node{&soml.Text{Value: "a < b & c"}},
	}
	deck.Slides[0].Content = []soml.Node{&soml.Element{Name: "ul", Children: []soml.Node{li}}}
	out := renderString(t, deck)
	if !strings.Contains(out, "a &lt; b &amp; c") {
		t.Errorf("li 텍스트가 이스케이프되지 않았다:\n%s", stageOf(t, out))
	}
}

// 해석되지 않은 인라인 외관은 에러다 — 조용히 부모 스타일로 그리면 웹만 그럴듯하게 나오고
// PPTX 런 분할과 갈라진다.
func TestRenderRejectsUnresolvedInline(t *testing.T) {
	deck := contentDeck()
	style := bodyStyle()
	deck.Slides[0].Content = []soml.Node{&soml.Element{
		Name:       "p",
		Appearance: &soml.Appearance{Text: &style},
		Children: []soml.Node{
			&soml.Text{Value: "본문 "},
			&soml.Element{Name: "strong", Children: []soml.Node{&soml.Text{Value: "굵게"}}},
		},
	}}
	if _, err := New().Render(deck); err == nil {
		t.Error("외관이 해석되지 않은 인라인은 에러여야 한다")
	}
}

// <a>의 목적지가 확정되지 않은 덱도 에러다 — core의 href 검증을 지나지 않은 값이 산출물에
// 실릴 길을 막는다.
func TestRenderRejectsAnchorWithoutLink(t *testing.T) {
	deck := contentDeck()
	style := bodyStyle()
	deck.Slides[0].Content = []soml.Node{&soml.Element{
		Name:       "p",
		Appearance: &soml.Appearance{Text: &style},
		Children: []soml.Node{&soml.Element{
			Name:       "a",
			Appearance: &soml.Appearance{Text: &style},
			Children:   []soml.Node{&soml.Text{Value: "링크"}},
		}},
	}}
	if _, err := New().Render(deck); err == nil {
		t.Error("Link가 없는 <a>는 에러여야 한다")
	}
}
