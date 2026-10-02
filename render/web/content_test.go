package web

import (
	"fmt"
	"strings"
	"testing"

	"github.com/k2sebeom/balpyo/soml"
)

// textElem은 해석이 끝난 `<text>` 하나다 (파서 + core.Resolve를 거친 상태).
func textElem(content string, style soml.TextStyle) *soml.Element {
	return &soml.Element{
		Name:       soml.TextElement,
		Children:   []soml.Node{&soml.Text{Value: content}},
		Appearance: &soml.Appearance{Text: &style},
	}
}

func bodyStyle() soml.TextStyle {
	return soml.TextStyle{Role: soml.RoleBody, Font: soml.FontBody, Size: 24, Weight: 400,
		LineHeight: 1.5, Color: soml.TokenColor("fg"), Align: soml.AlignStart}
}

func titleStyle() soml.TextStyle {
	return soml.TextStyle{Role: soml.RoleTitle, Font: soml.FontHeading, Size: 56, Weight: 700,
		LineHeight: 1.2, Color: soml.TokenColor("accent1"), Align: soml.AlignCenter}
}

// contentDeck은 콘텐츠가 있는 덱이다 — 슬라이드 하나에 텍스트 둘, 노트 하나.
func contentDeck() *soml.Deck {
	stage, _ := soml.Aspect16x9.Stage()
	return &soml.Deck{
		Title:  "콘텐츠 덱",
		Lang:   "ko",
		Aspect: soml.Aspect16x9,
		Stage:  stage,
		Theme:  testTheme(),
		Slides: []*soml.Slide{{
			Index:  0,
			ID:     "title",
			Source: "slides/001_title.slide",
			Notes:  "인사 30초",
			Content: []soml.Node{
				textElem("2026 아키텍처 리뷰", titleStyle()),
				textElem("상반기 회고와 하반기 계획", bodyStyle()),
			},
		}},
	}
}

// 요소 식별자는 지오메트리 패스가 브라우저 rect를 덱 요소에 되돌려 붙이는 유일한 열쇠다. 규칙: `s<슬라이드 인덱스>[.<자식 슬라이스 인덱스>]…`.
func TestRenderElementIdentity(t *testing.T) {
	deck := contentDeck()
	deck.Slides = append(deck.Slides, &soml.Slide{
		Index:   1,
		ID:      "second",
		Source:  "slides/002_second.slide",
		Content: []soml.Node{textElem("두 번째", bodyStyle())},
	})
	out := renderString(t, deck)
	for _, want := range []string{
		`data-balpyo-id="s0.0"`,
		`data-balpyo-id="s0.1"`,
		`data-balpyo-id="s1.0"`,
		`data-balpyo-el="text"`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("요소 식별자 %q가 없다", want)
		}
	}
	// 식별자는 문서 전체에서 유일해야 한다 — 겹치면 rect가 다른 요소에 붙는다.
	if got, want := strings.Count(out, "data-balpyo-id="), 3; got != want {
		t.Errorf("식별자 개수 = %d, want %d (렌더된 요소마다 하나)", got, want)
	}
}

// 측정 대상 계약: 식별자는 덱 트리의 슬라이스 인덱스 경로이고, 잴 요소(블록·리스트·컨테이너·
// spacer·도형과 그 안의 텍스트)에만 붙는다. 인라인은 PPTX에서 박스가 아니라 런이고, 도형의
// 그림 svg는 박스가 좌표를 소유한다. 개수가 geom.Refs와 맞아야 자리 맞대기가 성립한다.
func TestElementIDContract(t *testing.T) {
	type ref struct{ id, el string }
	for _, tc := range []struct {
		name string
		body string
		want []ref
	}{
		{"인라인에는 없다", `<p><strong>굵게 <em>기울여</em></strong> <a href="https://example.com">링크</a><br/>다음 줄</p>
  <ul><li><code>코드</code> 항목</li></ul>`,
			[]ref{{"s0.0", "p"}, {"s0.1", "ul"}, {"s0.1.0", "li"}}},
		{"중첩 레이아웃", `<col gap="md"><row gap="sm"><p>중첩</p></row></col>`,
			[]ref{{"s0.0", "col"}, {"s0.0.0", "row"}, {"s0.0.0.0", "p"}}},
		{"stack 중첩과 grid 안 stack", `<stack><stack><p at="center">안쪽</p></stack></stack>
  <grid cols="2"><stack><p at="bottom-right">셀 안 stack</p></stack></grid>`,
			[]ref{{"s0.0", "stack"}, {"s0.0.0", "stack"}, {"s0.0.0.0", "p"},
				{"s0.1", "grid"}, {"s0.1.0", "stack"}, {"s0.1.0.0", "p"}}},
		{"도형과 그 안의 텍스트", `<grid cols="2"><shape type="rect"><h3>머리</h3><p>본문</p></shape><shape type="ellipse"/></grid>`,
			[]ref{{"s0.0", "grid"}, {"s0.0.0", "shape"}, {"s0.0.0.0", "h3"}, {"s0.0.0.1", "p"}, {"s0.0.1", "shape"}}},
		{"배치를 담은 도형", `<shape type="rounded-rect" pad="lg" w="320" h="240"><col gap="sm">
      <text role="heading">카드 제목</text><p>본문 한 줄.</p><text role="caption">푸터</text></col></shape>`,
			[]ref{{"s0.0", "shape"}, {"s0.0.0", "col"},
				{"s0.0.0.0", "text"}, {"s0.0.0.1", "p"}, {"s0.0.0.2", "text"}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			out := renderString(t, subsetDeck(t, tc.body))
			for _, r := range tc.want {
				if open := openTagOf(t, out, r.id); !strings.Contains(open, fmt.Sprintf("%s=%q", ElAttr, r.el)) {
					t.Errorf("%s의 여는 태그에 %s=%q가 없다: %s", r.id, ElAttr, r.el, open)
				}
			}
			if got := strings.Count(out, IDAttr+"="); got != len(tc.want) {
				t.Errorf("식별자 개수 = %d, want %d", got, len(tc.want))
			}
			if strings.Contains(out, fmt.Sprintf("<svg class=%q %s", shapeInkClass, IDAttr)) {
				t.Error("그림 svg에 식별자가 붙었다 — 측정 대상이 아니다")
			}
		})
	}
}

// 해석되지 않은 외관을 조용히 기본값으로 그리면 웹만 그럴듯하게 나오고 PPTX와 갈라진다.
func TestRenderRejectsUnresolvedText(t *testing.T) {
	style := bodyStyle()
	para := func(children ...soml.Node) soml.Node {
		return &soml.Element{Name: "p", Appearance: &soml.Appearance{Text: &style}, Children: children}
	}
	for name, node := range map[string]soml.Node{
		"외관 없는 텍스트": &soml.Element{Name: soml.TextElement, Children: []soml.Node{&soml.Text{Value: "제목"}}},
		"외관 없는 인라인": para(&soml.Element{Name: "strong", Children: []soml.Node{&soml.Text{Value: "굵게"}}}),
		// core의 href 검증을 지나지 않은 값이 산출물에 실릴 길을 막는다.
		"Link 없는 <a>": para(&soml.Element{Name: "a", Appearance: &soml.Appearance{Text: &style},
			Children: []soml.Node{&soml.Text{Value: "링크"}}}),
	} {
		deck := contentDeck()
		deck.Slides[0].Content = []soml.Node{node}
		if _, err := New().Render(deck); err == nil {
			t.Errorf("%s는 에러여야 한다", name)
		}
	}
}

// 닫힌 어휘에 태그를 추가하고 렌더 매핑을 잊으면 "파싱은 되는데 그려지지 않는" 상태가 된다.
// 렌더러가 모르는 요소는 조용히 사라지는 대신 빌드를 멈춘다.
func TestRenderRejectsUnknownElement(t *testing.T) {
	deck := contentDeck()
	deck.Slides[0].Content = []soml.Node{&soml.Element{Name: "shape"}}
	_, err := New().Render(deck)
	if err == nil {
		t.Fatal("모르는 요소는 에러여야 한다")
	}
	// 진단은 파일과 위치를 갖는다 — 렌더 단계의 에러도 저자가 고칠 자리를 가리켜야 한다.
	if !strings.Contains(err.Error(), "slides/001_title.slide") {
		t.Errorf("에러가 파일을 가리키지 않는다: %v", err)
	}
}

// 한국어 덱은 어절에서 접는다 — 규칙은 lang이 ko일 때만 걸린다 (중국어·일본어는 keep-all이
// 줄바꿈을 막는다).
func TestRenderStageBreaksKoreanAtWords(t *testing.T) {
	css := between(t, renderString(t, contentDeck()), "<style>", "</style>")
	want := ".balpyo-stage:lang(ko) {\n    word-break: keep-all;\n    overflow-wrap: anywhere;\n  }"
	if !strings.Contains(css, want) {
		t.Errorf("스테이지 CSS에 한국어 줄바꿈 규칙이 없다:\n%s", css)
	}
}

// stageOf는 첫 슬라이드의 스테이지 안쪽 HTML이다. 스테이지 닫는 태그는 그 자체로 한 줄이고
// 텍스트 블록은 내용까지 한 줄에 담기므로 줄 단위로 구간을 잘라낼 수 있다.
func stageOf(t *testing.T, out string) string {
	t.Helper()
	lines := strings.Split(out, "\n")
	start := -1
	for i, line := range lines {
		if strings.Contains(line, `class="balpyo-stage"`) {
			start = i + 1
			break
		}
	}
	if start < 0 {
		t.Fatal("스테이지가 없다")
	}
	for i := start; i < len(lines); i++ {
		if lines[i] == "</div>" {
			return strings.Join(lines[start:i], "\n")
		}
	}
	t.Fatal("스테이지가 닫히지 않았다")
	return ""
}

func between(t *testing.T, s, open, shut string) string {
	t.Helper()
	start := strings.Index(s, open)
	if start < 0 {
		t.Fatalf("%q를 찾을 수 없다", open)
	}
	rest := s[start+len(open):]
	end := strings.Index(rest, shut)
	if end < 0 {
		t.Fatalf("%q를 찾을 수 없다", shut)
	}
	return rest[:end]
}

// renderedKinds는 writerFor의 짝이다 — 목록이 거짓말을 하면 카탈로그와의 대조
// (render 패키지의 TestRenderersCoverCatalog)도 같이 거짓말을 한다. 목록은 바깥에 내보이는
// 얼굴이고 writerFor가 실제 동작이므로 둘을 여기서 맞춘다 (TASKS P0.2).
func TestRenderedKindsMatchDispatch(t *testing.T) {
	listed := make(map[soml.ElementKind]bool, len(renderedKinds))
	for _, kind := range RenderedKinds() {
		listed[kind] = true
	}
	for _, kind := range soml.ElementKinds() {
		if got, want := writerFor(kind) != nil, listed[kind]; got != want {
			t.Errorf("%v: writerFor 있음=%v, 목록에 있음=%v — 목록과 디스패치가 어긋났다", kind, got, want)
		}
	}
	// 카탈로그 밖(`<slide>`·`<text>`·`<note>`)은 writeNode·writeStage가 이름으로 다룬다.
	if writerFor(soml.KindNone) != nil {
		t.Error("writerFor(KindNone)이 함수를 준다 — 카탈로그 밖의 요소는 이름으로 다룬다")
	}
}
