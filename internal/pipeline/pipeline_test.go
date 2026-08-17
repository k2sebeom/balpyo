package pipeline

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/k2sebeom/balpyo/render"
	"github.com/k2sebeom/balpyo/render/pptx"
	"github.com/k2sebeom/balpyo/render/web"
	"github.com/k2sebeom/balpyo/soml"
)

// newProject는 파일 맵으로 프로젝트 디렉토리를 만든다.
func newProject(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for name, body := range files {
		dest := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(dest, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func TestLoadDeckCollectsAndOrdersSlides(t *testing.T) {
	dir := newProject(t, map[string]string{
		"balpyo.json":                   `{"title": "리뷰", "author": "이세범", "lang": "ko"}`,
		"slides/010_architecture.slide": "<slide></slide>\n",
		"slides/001_title.slide":        "<slide></slide>\n",
		"slides/_draft.slide":           "<slide></slide>\n",
	})
	deck, diags, err := LoadDeck(Options{ProjectDir: dir})
	if err != nil {
		t.Fatal(err)
	}
	if len(diags) != 0 {
		t.Errorf("진단이 없어야 한다: %v", diags)
	}
	if len(deck.Slides) != 2 {
		t.Fatalf("슬라이드 %d개, want 2개 (_draft는 무시)", len(deck.Slides))
	}
	if deck.Slides[0].ID != "title" || deck.Slides[1].ID != "architecture" {
		t.Errorf("ID·순서 불일치: %q, %q", deck.Slides[0].ID, deck.Slides[1].ID)
	}
	if deck.Slides[1].Index != 1 || deck.Slides[1].Source != "slides/010_architecture.slide" {
		t.Errorf("Index·Source 불일치: %+v", deck.Slides[1])
	}
	if deck.Title != "리뷰" || deck.Author != "이세범" || deck.Lang != "ko" {
		t.Errorf("설정이 덱에 반영되지 않았다: %+v", deck)
	}
	// 논리 스테이지는 종횡비에서 유도된다 — 렌더러가 비어 있는 Stage를 받으면 안 된다.
	if deck.Stage.Width != 1280 || deck.Stage.Height != 720 {
		t.Errorf("Stage = %+v", deck.Stage)
	}
}

// 트리비아 보존 토큰은 덱 모델에 남는다 — format이 원문을 되돌려 쓸 근거다.
func TestLoadDeckKeepsSourceTokens(t *testing.T) {
	src := "<!-- 메모 -->\n<slide>\n</slide>\n"
	dir := newProject(t, map[string]string{"slides/001_title.slide": src})
	deck, _, err := LoadDeck(Options{ProjectDir: dir})
	if err != nil {
		t.Fatal(err)
	}
	if got := deck.Slides[0].Tokens.Source(); got != src {
		t.Errorf("토큰에서 원문이 복원되지 않았다:\ngot  %q\nwant %q", got, src)
	}
}

// 파서가 읽은 슬라이드 속성이 덱 모델까지 온다.
func TestLoadDeckAppliesSlideAttributes(t *testing.T) {
	dir := newProject(t, map[string]string{
		"slides/001_title.slide": `<slide id="hero">
  <text role="title">2026 아키텍처 리뷰</text>
  <note>인사 30초</note>
</slide>`,
	})
	deck, _, err := LoadDeck(Options{ProjectDir: dir})
	if err != nil {
		t.Fatal(err)
	}
	slide := deck.Slides[0]
	// id 속성이 있으면 파일명 유도(`title`)를 이긴다.
	if slide.ID != "hero" {
		t.Errorf("ID = %q, want %q", slide.ID, "hero")
	}
	if slide.Notes != "인사 30초" {
		t.Errorf("Notes = %q", slide.Notes)
	}
	if len(slide.Content) != 1 {
		t.Fatalf("콘텐츠 노드 %d개, want 1개 (<note>는 Notes로 간다)", len(slide.Content))
	}
	elem, ok := slide.Content[0].(*soml.Element)
	if !ok || elem.Name != "text" || elem.TextContent() != "2026 아키텍처 리뷰" {
		t.Errorf("콘텐츠 = %+v", slide.Content[0])
	}
}

// skip된 슬라이드는 덱에 없고, Index는 남은 슬라이드로 0부터 다시 센다.
func TestLoadDeckSkipsSlides(t *testing.T) {
	dir := newProject(t, map[string]string{
		"slides/001_title.slide":   "<slide></slide>",
		"slides/002_draft.slide":   `<slide skip="true"></slide>`,
		"slides/003_wip.slide":     "<slide skip></slide>",
		"slides/004_closing.slide": "<slide></slide>",
	})
	deck, _, err := LoadDeck(Options{ProjectDir: dir})
	if err != nil {
		t.Fatal(err)
	}
	if len(deck.Slides) != 2 {
		t.Fatalf("슬라이드 %d개, want 2개", len(deck.Slides))
	}
	for i, slide := range deck.Slides {
		if slide.Index != i {
			t.Errorf("슬라이드 %s의 Index = %d, want %d", slide.ID, slide.Index, i)
		}
	}
	if deck.Slides[1].ID != "closing" {
		t.Errorf("두 번째 슬라이드 = %q, want %q", deck.Slides[1].ID, "closing")
	}
}

func TestLoadDeckAllSlidesSkipped(t *testing.T) {
	dir := newProject(t, map[string]string{"slides/001_title.slide": "<slide skip></slide>"})
	_, _, err := LoadDeck(Options{ProjectDir: dir})
	if err == nil || !strings.Contains(err.Error(), "모두 skip") {
		t.Errorf("에러 = %v, want \"모두 skip\"", err)
	}
}

// 파서 진단은 파일 경로와 line:col을 갖고 빌드를 멈춘다.
func TestLoadDeckReportsParseErrors(t *testing.T) {
	dir := newProject(t, map[string]string{
		"slides/001_title.slide": "<slide>\n  <txt>제목</txt>\n</slide>",
	})
	_, diags, err := LoadDeck(Options{ProjectDir: dir})
	if err == nil {
		t.Fatal("닫힌 어휘 위반은 빌드를 멈춘다")
	}
	if len(diags) != 1 {
		t.Fatalf("진단 %d개, want 1개: %v", len(diags), diags)
	}
	want := "slides/001_title.slide:2:4: error: 알 수 없는 태그 <txt>"
	if !strings.HasPrefix(diags[0].String(), want) {
		t.Errorf("진단 = %q, want %q로 시작", diags[0].String(), want)
	}
}

func TestLoadDeckWithoutConfig(t *testing.T) {
	// balpyo.json 없이도 빌드가 되어야 첫 결과까지의 거리가 짧다.
	dir := newProject(t, map[string]string{"slides/001_title.slide": "<slide></slide>"})
	deck, _, err := LoadDeck(Options{ProjectDir: dir})
	if err != nil {
		t.Fatal(err)
	}
	if deck.Aspect != soml.DefaultAspect || len(deck.Slides) != 1 {
		t.Errorf("기본값으로 빌드되지 않았다: %+v", deck)
	}
}

func TestLoadDeckErrors(t *testing.T) {
	tests := []struct {
		name    string
		files   map[string]string
		wantMsg string
	}{
		{
			name:    "슬라이드 디렉토리가 없다",
			files:   map[string]string{"balpyo.json": `{}`},
			wantMsg: "slides",
		},
		{
			name:    "슬라이드 파일이 없다",
			files:   map[string]string{"slides/README.md": "메모"},
			wantMsg: "슬라이드가 없다",
		},
		{
			name: "ID가 충돌한다",
			files: map[string]string{
				"slides/001_intro.slide":    "<slide></slide>",
				"slides/ch/002_intro.slide": "<slide></slide>",
			},
			wantMsg: "충돌",
		},
		{
			name:    "설정이 깨졌다",
			files:   map[string]string{"balpyo.json": `{"aspect": "21:9"}`, "slides/a.slide": ""},
			wantMsg: "aspect",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, _, err := LoadDeck(Options{ProjectDir: newProject(t, tc.files)})
			if err == nil {
				t.Fatal("에러여야 한다")
			}
			if !strings.Contains(err.Error(), tc.wantMsg) {
				t.Errorf("에러 메시지에 %q가 없다: %v", tc.wantMsg, err)
			}
		})
	}
}

func TestBuildWritesBothTargets(t *testing.T) {
	dir := newProject(t, map[string]string{
		"balpyo.json":            `{"title": "my talk"}`,
		"slides/001_title.slide": "<slide></slide>",
	})
	result, err := Build(Options{
		ProjectDir: dir,
		Targets:    []render.Target{render.TargetHTML, render.TargetPPTX},
	})
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(dir, "dist"); result.OutDir != want {
		// t.TempDir는 심볼릭 링크일 수 있으므로 접미로 비교한다.
		if !strings.HasSuffix(result.OutDir, filepath.Join(filepath.Base(dir), "dist")) {
			t.Errorf("OutDir = %q, want %q", result.OutDir, want)
		}
	}
	wantFiles := []string{web.IndexFile, "my-talk.pptx"}
	if len(result.Written) != len(wantFiles) {
		t.Fatalf("쓴 파일 = %v, want %v", result.Written, wantFiles)
	}
	for i, want := range wantFiles {
		if result.Written[i] != want {
			t.Errorf("쓴 파일[%d] = %q, want %q", i, result.Written[i], want)
		}
		info, err := os.Stat(filepath.Join(result.OutDir, want))
		if err != nil {
			t.Fatalf("%s가 디스크에 없다: %v", want, err)
		}
		if info.Size() == 0 {
			t.Errorf("%s가 비어 있다", want)
		}
	}
}

func TestBuildDefaultsToHTMLOnly(t *testing.T) {
	dir := newProject(t, map[string]string{"slides/001_title.slide": "<slide></slide>"})
	result, err := Build(Options{ProjectDir: dir})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Written) != 1 || result.Written[0] != web.IndexFile {
		t.Errorf("기본 타깃 산출물 = %v, want [%s]", result.Written, web.IndexFile)
	}
}

func TestBuildHonorsOutDir(t *testing.T) {
	dir := newProject(t, map[string]string{"slides/001_title.slide": "<slide></slide>"})
	out := filepath.Join(t.TempDir(), "elsewhere")
	result, err := Build(Options{ProjectDir: dir, OutDir: out})
	if err != nil {
		t.Fatal(err)
	}
	if result.OutDir != out {
		t.Errorf("OutDir = %q, want %q", result.OutDir, out)
	}
	if _, err := os.Stat(filepath.Join(out, web.IndexFile)); err != nil {
		t.Error(err)
	}
}

func TestRenderRejectsUnknownTarget(t *testing.T) {
	stage, _ := soml.Aspect16x9.Stage()
	deck := &soml.Deck{Stage: stage, Slides: []*soml.Slide{{ID: "a"}}}
	if _, err := Render(deck, []render.Target{"pdf"}); err == nil {
		t.Error("PDF는 아직 렌더러가 없다 — 에러여야 한다")
	}
}

func TestRenderersCoverAllParsableTargets(t *testing.T) {
	// --target이 받는 이름과 렌더러 목록이 어긋나면 CLI가 통과시킨 타깃이 런타임에 죽는다.
	targets, err := render.ParseTargets("html,pptx")
	if err != nil {
		t.Fatal(err)
	}
	available := Renderers()
	for _, target := range targets {
		if _, ok := available[target]; !ok {
			t.Errorf("타깃 %q의 렌더러가 없다", target)
		}
	}
	if len(available) != len(targets) {
		t.Errorf("렌더러 %d개, --target이 받는 이름 %d개", len(available), len(targets))
	}
}

func TestPPTXFileNameFollowsTitle(t *testing.T) {
	dir := newProject(t, map[string]string{
		"balpyo.json":            `{"title": "2026 아키텍처 리뷰"}`,
		"slides/001_title.slide": "<slide></slide>",
	})
	deck, _, err := LoadDeck(Options{ProjectDir: dir})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := pptx.FileName(deck), "2026-아키텍처-리뷰.pptx"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}
