package web

import (
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

func TestRenderTextContent(t *testing.T) {
	out := renderString(t, contentDeck())
	for _, want := range []string{
		`<div class="balpyo-text"`,
		"2026 아키텍처 리뷰",
		"상반기 회고와 하반기 계획",
		"font-size: 56px",
		"color: var(--balpyo-color-accent1)",
		"text-align: center",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("산출 HTML에 %q가 없다", want)
		}
	}
	// 텍스트는 스테이지 안에 들어간다 — 지오메트리 패스가 스테이지 좌표계로 rect를 읽는다.
	stage := stageOf(t, out)
	if !strings.Contains(stage, "2026 아키텍처 리뷰") {
		t.Errorf("텍스트가 스테이지 밖에 있다:\n%s", stage)
	}
}

// 문서 순서가 곧 스택 순서다 — 저자가 쓴 순서와 화면 순서가 다르면 SOML을 읽어 결과를
// 예측할 수 없다.
func TestRenderKeepsContentOrder(t *testing.T) {
	out := renderString(t, contentDeck())
	first := strings.Index(out, "2026 아키텍처 리뷰")
	second := strings.Index(out, "상반기 회고와 하반기 계획")
	if first < 0 || second < 0 {
		t.Fatal("텍스트가 없다")
	}
	if first > second {
		t.Error("콘텐츠는 문서 순서대로 나와야 한다")
	}
}

// 요소 식별자는 지오메트리 패스(Iteration 1의 다음 조각)가 브라우저 rect를 덱 요소에
// 되돌려 붙이는 유일한 열쇠다. 규칙: `s<슬라이드 인덱스>[.<자식 슬라이스 인덱스>]…`.
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
	if n := strings.Count(out, `data-balpyo-id="s0.0"`); n != 1 {
		t.Errorf(`data-balpyo-id="s0.0"가 %d번 나온다`, n)
	}
	if got, want := strings.Count(out, "data-balpyo-id="), 3; got != want {
		t.Errorf("식별자 개수 = %d, want %d (렌더된 요소마다 하나)", got, want)
	}
}

// 노트는 발표자용 메타데이터다 (파서가 Content에서 빼 Slide.Notes로 보낸다). 스테이지에
// 한 번 더 그리면 화면에 노트가 새어 나온다.
func TestRenderNoteStaysOutOfStage(t *testing.T) {
	out := renderString(t, contentDeck())
	if stage := stageOf(t, out); strings.Contains(stage, "인사 30초") {
		t.Errorf("노트가 스테이지 안에 들어갔다:\n%s", stage)
	}
	if got := strings.Count(out, "인사 30초"); got != 1 {
		t.Errorf("노트 텍스트가 %d번 나온다, want 1 (숨은 노트 요소에만)", got)
	}
}

func TestRenderEscapesTextContent(t *testing.T) {
	deck := contentDeck()
	deck.Slides[0].Content = []soml.Node{textElem(`<script>alert("x")</script> & 5 > 3`, bodyStyle())}
	out := renderString(t, deck)
	// `<script>` 자체는 뷰어 런타임으로 정당하게 존재한다 — 저자 텍스트가 태그로 새는
	// 것만 잡는다.
	if strings.Contains(out, "<script>alert") {
		t.Error("텍스트 콘텐츠가 이스케이프되지 않았다")
	}
	if !strings.Contains(out, "&amp; 5 &gt; 3") {
		t.Error("`&`와 `>`가 이스케이프되어야 한다")
	}
}

// 콘텐츠가 없는 슬라이드의 스테이지는 빈 요소 하나다 — 빈 슬라이드의 산출물이 콘텐츠
// 렌더가 붙었다고 달라지면 00_empty 골든이 이유 없이 움직인다.
func TestRenderEmptyStageStaysCompact(t *testing.T) {
	deck := contentDeck()
	deck.Slides[0].Content = nil
	out := renderString(t, deck)
	if !strings.Contains(out, `<div class="balpyo-stage"></div>`) {
		t.Error("빈 스테이지는 빈 요소 하나여야 한다")
	}
}

// 해석되지 않은 외관을 조용히 기본값으로 그리면 웹과 PPTX가 서로 다른 것을 그리기 시작한다.
func TestRenderRejectsUnresolvedAppearance(t *testing.T) {
	deck := contentDeck()
	unresolved := &soml.Element{Name: soml.TextElement, Children: []soml.Node{&soml.Text{Value: "제목"}}}
	deck.Slides[0].Content = []soml.Node{unresolved}
	if _, err := New().Render(deck); err == nil {
		t.Error("외관이 해석되지 않은 요소는 에러여야 한다")
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

func TestRenderContentIsDeterministic(t *testing.T) {
	a, b := renderString(t, contentDeck()), renderString(t, contentDeck())
	if a != b {
		t.Error("콘텐츠 렌더 결과가 실행마다 다르다")
	}
}

// 스테이지는 자식을 위에서 아래로 쌓고, 텍스트 블록은 콘텐츠 폭 전체를 갖는다.
// 폭이 확정되어야 브라우저의 줄바꿈이 결정적이고, 그 rect가 그대로 PPTX 텍스트 박스가 된다
// (PRD §9.3: 박스 명시 고정 + autofit off).
func TestRenderStageStacksChildren(t *testing.T) {
	out := renderString(t, contentDeck())
	css := between(t, out, "<style>", "</style>")
	for _, want := range []string{
		"flex-direction: column;",
		"gap: 24px;",
		"padding: 64px;",
		"width: 100%;",
		"flex: 0 0 auto;",
	} {
		if !strings.Contains(css, want) {
			t.Errorf("스택 CSS에 %q가 없다", want)
		}
	}
}

// 잰 폭은 브라우저의 기본값에 기대지 않는다 — 스테이지가 스크립트 간격을 **끈 상태로**
// 못 박는다. 이 한 선언이 없으면 브라우저가 한글↔라틴 경계에 1/4 em을 넣기 시작하는 날
// (CSS Text 4의 `text-autospace`) 모든 줄의 잰 폭이 움직이고, 그 폭이 곧 PPTX 좌표라
// 산출물이 조용히 다시 접힌다 (D11 — docs/guides/known-differences.md).
func TestRenderStagePinsScriptSpacing(t *testing.T) {
	css := between(t, renderString(t, contentDeck()), "<style>", "</style>")
	if !strings.Contains(css, "text-autospace: no-autospace;") {
		t.Errorf("스테이지 CSS에 스크립트 간격 고정이 없다:\n%s", css)
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
