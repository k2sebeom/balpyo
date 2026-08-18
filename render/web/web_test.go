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
	// 자체 완결형: 외부 CSS·JS 참조가 없어야 어디에 두어도 열린다.
	for _, forbidden := range []string{"<link", "<script src"} {
		if strings.Contains(out, forbidden) {
			t.Errorf("외부 참조 %q가 들어갔다", forbidden)
		}
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
}

func TestRenderEscapesText(t *testing.T) {
	deck := testDeck()
	deck.Title = `<script>alert("x")</script>`
	out := renderString(t, deck)
	// `<script>` 자체는 뷰어 런타임으로 정당하게 존재한다 — 저자 텍스트가 태그로 새는
	// 것만 잡는다.
	if strings.Contains(out, "<script>alert") {
		t.Error("제목이 이스케이프되지 않았다")
	}
	if !strings.Contains(out, "&lt;script&gt;") {
		t.Error("이스케이프된 제목이 문서에 없다")
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

func TestRenderIsDeterministic(t *testing.T) {
	// 골든 테스트가 성립하려면 같은 입력이 바이트 단위로 같은 출력을 내야 한다.
	a, b := renderString(t, testDeck()), renderString(t, testDeck())
	if a != b {
		t.Error("렌더 결과가 실행마다 다르다")
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
