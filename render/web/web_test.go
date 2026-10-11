package web

import (
	"strings"
	"testing"

	"github.com/k2sebeom/balpyo/render"
	"github.com/k2sebeom/balpyo/soml"
)

func testDeck() *soml.Deck {
	stage, _ := soml.Aspect16x9.Stage()
	return &soml.Deck{
		Title:  "2026 아키텍처 리뷰",
		Lang:   "ko",
		Aspect: soml.Aspect16x9,
		Stage:  stage,
		Slides: []*soml.Slide{
			{Index: 0, ID: "title", Source: "slides/001_title.slide"},
			{Index: 1, ID: "agenda", Source: "slides/002_agenda.slide", Notes: "인사 30초"},
		},
	}
}

func TestRenderProducesSingleSelfContainedFile(t *testing.T) {
	files, err := New().Render(testDeck())
	if err != nil {
		t.Fatal(err)
	}
	if got := files.Paths(); len(got) != 1 || got[0] != IndexFile {
		t.Fatalf("산출 파일 = %v, want [%s]", got, IndexFile)
	}
	out := string(files[IndexFile])
	// 자체 완결형: 외부 CSS·JS 참조가 없어야 어디에 두어도 열린다. <link>는 data URI로
	// 인라인된 것(파비콘)만 허용한다.
	if strings.Contains(out, "<script src") {
		t.Error(`외부 참조 "<script src"가 들어갔다`)
	}
	for rest := out; ; {
		_, after, ok := strings.Cut(rest, "<link")
		if !ok {
			break
		}
		tag, _, _ := strings.Cut(after, ">")
		if !strings.Contains(tag, `href="data:`) {
			t.Errorf("외부 참조 <link%s>가 들어갔다", tag)
		}
		rest = after
	}
	if !strings.Contains(out, "<style>") {
		t.Error("CSS가 인라인되어야 한다")
	}
}

func TestRenderStageMatchesAspect(t *testing.T) {
	out := renderString(t, testDeck())
	if !strings.Contains(out, "--balpyo-stage-width: 1280px") ||
		!strings.Contains(out, "--balpyo-stage-height: 720px") {
		t.Error("논리 스테이지가 16:9(1280×720)로 나와야 한다")
	}

	deck := testDeck()
	deck.Aspect = soml.Aspect4x3
	deck.Stage, _ = soml.Aspect4x3.Stage()
	out = renderString(t, deck)
	if !strings.Contains(out, "--balpyo-stage-width: 960px") ||
		!strings.Contains(out, "--balpyo-stage-height: 720px") {
		t.Error("4:3은 960×720이어야 한다")
	}
}

func TestRenderDeclaresCascadeLayers(t *testing.T) {
	// 층 선언은 테마·저자 CSS가 실리기 전에 우선순위를 확정한다.
	out := renderString(t, testDeck())
	if !strings.Contains(out, "@layer balpyo.base, balpyo.theme, balpyo.author;") {
		t.Error("cascade layer 선언이 없다")
	}
}

func TestRenderSlideAnchorsAndOrder(t *testing.T) {
	out := renderString(t, testDeck())
	first := strings.Index(out, `id="title"`)
	second := strings.Index(out, `id="agenda"`)
	if first < 0 || second < 0 {
		t.Fatal("슬라이드 앵커가 없다")
	}
	if first > second {
		t.Error("슬라이드는 덱 순서대로 나와야 한다")
	}
	// 지오메트리 패스가 이 선택자로 요소를 집는다.
	if !strings.Contains(out, `data-slide-index="1"`) ||
		!strings.Contains(out, `data-slide-source="slides/002_agenda.slide"`) {
		t.Error("지오메트리 패스가 쓰는 data 속성이 없다")
	}
}

func TestRenderNotesAreHidden(t *testing.T) {
	out := renderString(t, testDeck())
	if !strings.Contains(out, `<div class="balpyo-notes" hidden>인사 30초</div>`) {
		t.Error("노트는 문서에 있지만 화면에는 나오지 않아야 한다")
	}
	// 노트가 없는 슬라이드에는 빈 노트 요소를 만들지 않는다. 클래스 이름만 세면 안 된다 —
	// 런타임 JS·CSS가 선택자로 같은 이름을 리터럴로 쓴다.
	if strings.Count(out, `<div class="balpyo-notes"`) != 1 {
		t.Error("노트 요소는 노트가 있는 슬라이드에만 있어야 한다")
	}
	// 노트는 스테이지에 한 번 더 그려지지 않는다 (파서가 Content에서 뺐다).
	if got := strings.Count(out, "인사 30초"); got != 1 {
		t.Errorf("노트 텍스트가 %d번 나온다, want 1 (숨은 노트 요소에만)", got)
	}
}

// 저자 텍스트는 어느 자리에서든 태그로 새지 않는다. `<script>` 자체는 뷰어 런타임으로
// 정당하게 존재하므로 저자 텍스트의 흔적만 본다.
func TestRenderEscapesAuthorText(t *testing.T) {
	li := func(text string) *soml.Deck {
		deck := contentDeck()
		style := bodyStyle()
		deck.Slides[0].Content = []soml.Node{&soml.Element{Name: "ul", Children: []soml.Node{&soml.Element{
			Name: "li", Appearance: &soml.Appearance{Text: &style}, Children: []soml.Node{&soml.Text{Value: text}},
		}}}}
		return deck
	}
	for _, tc := range []struct {
		name      string
		deck      func() *soml.Deck
		want, bad string
	}{
		{"덱 제목", func() *soml.Deck {
			deck := testDeck()
			deck.Title = `<script>alert("x")</script>`
			return deck
		}, "&lt;script&gt;", "<script>alert"},
		{"텍스트", func() *soml.Deck {
			deck := contentDeck()
			deck.Slides[0].Content = []soml.Node{textElem(`<script>alert("x")</script> & 5 > 3`, bodyStyle())}
			return deck
		}, "&amp; 5 &gt; 3", "<script>alert"},
		// 해석되지 않은 원문 문자가 그대로 온 트리 — 참조 경로는 TestRenderCharRefsEscapeOnce가 본다.
		{"리스트 항목", func() *soml.Deck { return li("a < b & c") }, "a &lt; b &amp; c", "a < b"},
		// &가 그대로 나가면 파싱에 따라 엔티티로 읽혀 목적지가 조용히 바뀐다.
		{"href", func() *soml.Deck {
			return subsetDeck(t, `<p><a href="https://example.com/?a=1&b=2">링크</a></p>`)
		}, `href="https://example.com/?a=1&amp;b=2"`, `href="https://example.com/?a=1&b=2"`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			out := renderString(t, tc.deck())
			if !strings.Contains(out, tc.want) {
				t.Errorf("이스케이프된 %q가 없다", tc.want)
			}
			if strings.Contains(out, tc.bad) {
				t.Errorf("원문 %q가 그대로 나갔다", tc.bad)
			}
		})
	}
}

func TestRenderRejectsInvalidDeck(t *testing.T) {
	if _, err := New().Render(nil); err == nil {
		t.Error("nil 덱은 에러여야 한다")
	}
	// 스테이지 크기가 없는 덱은 렌더할 수 없다 — 파이프라인이 채워야 한다.
	if _, err := New().Render(&soml.Deck{}); err == nil {
		t.Error("스테이지 없는 덱은 에러여야 한다")
	}
}

// 같은 입력은 바이트 단위로 같은 출력을 낸다 — 어휘가 많이 섞인 덱 하나로 본다.
func TestRenderIsDeterministic(t *testing.T) {
	body := `<grid cols="3" gap="lg" pad="md sm"><p span="2" order="-1">격자</p><spacer grow="1"/>
    <shape type="cloud" fill="accent1" shadow="soft" rotate="15" h="160"><p anchor="center">구름</p></shape>
    <shape type="cylinder" fill="none" border="2 dashed muted" flip="v"/></grid>`
	for name, deck := range map[string]func() *soml.Deck{
		"빈 덱":     testDeck,
		"콘텐츠":     contentDeck,
		"레이아웃·도형": func() *soml.Deck { return subsetDeck(t, body) },
		"픽스처":     func() *soml.Deck { return fixtureDeck(t, textFixture) },
	} {
		if renderString(t, deck()) != renderString(t, deck()) {
			t.Errorf("%s: 렌더 결과가 실행마다 다르다", name)
		}
	}
}

func TestRendererTarget(t *testing.T) {
	if got := New().Target(); got != render.TargetHTML {
		t.Errorf("Target() = %q", got)
	}
}

func renderString(t *testing.T, deck *soml.Deck) string {
	t.Helper()
	files, err := New().Render(deck)
	if err != nil {
		t.Fatal(err)
	}
	return string(files[IndexFile])
}
