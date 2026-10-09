package pipeline

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
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

// TestLoadDeckCarriesTheThemeRequest는 `balpyo.json`의 theme가 **덱을 통해 값 층에 닿는다**는
// 배선을 잇는다 (P4.2 · ADR 0052). 그전에는 이 필드가 설정 검사만 받고 버려졌다 — 파이프라인이
// 옮기지 않으면 core는 언제나 기본 테마를 로드한다.
func TestLoadDeckCarriesTheThemeRequest(t *testing.T) {
	dir := newProject(t, map[string]string{
		"balpyo.json":            `{"theme": "ink"}`,
		"slides/001_title.slide": "<slide><h1>제목</h1></slide>",
	})
	deck, diags, err := LoadDeck(Options{ProjectDir: dir})
	if err != nil {
		t.Fatal(err)
	}
	if len(diags) != 0 {
		t.Fatalf("진단이 없어야 한다: %v", diags)
	}
	if deck.ThemeName != "ink" {
		t.Errorf("deck.ThemeName = %q, want ink (설정의 요청)", deck.ThemeName)
	}
	if deck.Theme.Name != "ink" {
		t.Errorf("deck.Theme.Name = %q, want ink (요청의 결과)", deck.Theme.Name)
	}
}

// 없는 테마 이름은 설정 로드에서 멈춘다 — 값 층도 같은 판정을 하지만(core) 설정 검사가 먼저
// 서서 `balpyo.json`을 가리킨다.
func TestLoadDeckRejectsUnknownTheme(t *testing.T) {
	dir := newProject(t, map[string]string{
		"balpyo.json":            `{"theme": "corp"}`,
		"slides/001_title.slide": "<slide><h1>제목</h1></slide>",
	})
	_, _, err := LoadDeck(Options{ProjectDir: dir})
	if err == nil {
		t.Fatal("없는 테마인데 덱이 로드됐다")
	}
	for _, want := range []string{"balpyo.json", "corp", "minimal"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("에러가 %q를 말하지 않는다: %v", want, err)
		}
	}
}

// TestLoadDeckThemeOptionOverridesConfig는 Options.Theme가 설정을 덮음을 못 박는다 (ADR 0052의
// 결정 9 — `balpyo build --theme`의 값 경로). OutDir과 같은 태도다: 플래그는 설정의 한 값을
// 이 실행에 한해 갈아 끼우고, 파일은 그대로 남는다.
func TestLoadDeckThemeOptionOverridesConfig(t *testing.T) {
	dir := newProject(t, map[string]string{
		"balpyo.json":            `{"theme": "ink"}`,
		"slides/001_title.slide": "<slide><h1>제목</h1></slide>",
	})
	deck, _, err := LoadDeck(Options{ProjectDir: dir, Theme: "editorial"})
	if err != nil {
		t.Fatal(err)
	}
	if deck.Theme.Name != "editorial" {
		t.Errorf("deck.Theme.Name = %q, want editorial (옵션이 설정을 덮는다)", deck.Theme.Name)
	}
}

// 옵션의 어휘 밖 이름은 **옵션을 가리키며** 멈춘다: 고칠 자리가 `balpyo.json`이 아니라 방금
// 친 명령이므로, 진단이 설정 파일을 가리키면 사람이 없는 오타를 찾는다.
func TestLoadDeckRejectsUnknownThemeOption(t *testing.T) {
	dir := newProject(t, map[string]string{
		"balpyo.json":            `{"theme": "ink"}`,
		"slides/001_title.slide": "<slide><h1>제목</h1></slide>",
	})
	_, _, err := LoadDeck(Options{ProjectDir: dir, Theme: "corp"})
	if err == nil {
		t.Fatal("없는 테마인데 덱이 로드됐다")
	}
	if strings.Contains(err.Error(), "balpyo.json") {
		t.Errorf("옵션의 오타를 설정 파일 탓으로 돌린다: %v", err)
	}
	for _, want := range []string{"theme", "corp", "minimal"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("에러가 %q를 말하지 않는다: %v", want, err)
		}
	}
}

// TestLoadDeckUsesAProjectLocalTheme는 프로젝트의 `themes/<이름>/`이 설정과 플래그 **양쪽**에서
// 이름으로 불림을 못 박는다 (P4.3 · ADR 0054). 파이프라인이 프로젝트 파일시스템을 값 층에
// 넘기지 않으면 로컬 테마는 core에서 보이지 않는다.
func TestLoadDeckUsesAProjectLocalTheme(t *testing.T) {
	files := map[string]string{
		"balpyo.json":                   `{"theme": "corp"}`,
		"slides/001_title.slide":        "<slide><h1>제목</h1></slide>",
		"themes/corp/theme.json":        `{"name":"corp","extends":"minimal"}`,
		"themes/corp/tokens.json":       `{"color":[{"name":"accent1","rgb":"112233","slot":"accent1","ref":"accent1"}]}`,
		"themes/corp/layouts/title.css": "/* corp의 표지 */\ndisplay: grid;",
	}
	dir := newProject(t, files)
	deck, diags, err := LoadDeck(Options{ProjectDir: dir})
	if err != nil {
		t.Fatalf("로컬 테마를 요청한 덱이 로드되지 않았다: %v", err)
	}
	if len(diags) != 0 {
		t.Fatalf("진단이 없어야 한다: %v", diags)
	}
	if deck.Theme.Name != "corp" {
		t.Errorf("deck.Theme.Name = %q, want corp", deck.Theme.Name)
	}
	accent1, ok := deck.Theme.Color("accent1")
	if !ok || accent1.RGB != "112233" {
		t.Errorf("accent1 = %+v, want 112233 (로컬 테마의 값)", accent1)
	}
	if !strings.Contains(deck.Theme.Layouts["title"], "display: grid") {
		t.Errorf("표지 배치가 로컬 테마의 파일이 아니다: %q", deck.Theme.Layouts["title"])
	}
	// 나머지 배치는 부모 것을 그대로 물려받는다. 같은 프로젝트를 minimal로 한 번 더 읽어
	// 견주는 이유는 이 맵이 **선언이 있는 레이아웃만** 싣는다는 것이다 (layoutStyles — 주석만
	// 있는 스텁은 여기 없다). 그래서 "11개"가 아니라 "부모와 같은 집합"이 상속의 증인이다.
	base, _, err := LoadDeck(Options{ProjectDir: dir, Theme: "minimal"})
	if err != nil {
		t.Fatalf("부모 테마로 견줄 덱이 로드되지 않았다: %v", err)
	}
	if len(deck.Theme.Layouts) != len(base.Theme.Layouts) {
		t.Errorf("배치 CSS %d개, want %d — 부모의 파일을 물려받지 않았다",
			len(deck.Theme.Layouts), len(base.Theme.Layouts))
	}
	for name, css := range base.Theme.Layouts {
		if name == "title" {
			continue // 자식이 자기 파일로 갈아 끼운 하나.
		}
		if deck.Theme.Layouts[name] != css {
			t.Errorf("레이아웃 %q가 부모 것과 다르다: %q", name, deck.Theme.Layouts[name])
		}
	}

	// 플래그도 같은 어휘를 받는다 — 설정이 내장 테마를 적어도 `--theme corp`가 된다.
	files["balpyo.json"] = `{"theme": "ink"}`
	flagged, _, err := LoadDeck(Options{ProjectDir: newProject(t, files), Theme: "corp"})
	if err != nil {
		t.Fatalf("--theme이 로컬 이름을 받지 않았다: %v", err)
	}
	if flagged.Theme.Name != "corp" {
		t.Errorf("deck.Theme.Name = %q, want corp (플래그가 설정을 덮는다)", flagged.Theme.Name)
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

// 모르는 variant 경고는 기본 빌드를 통과시키고 --strict가 에러로 승격한다 (ADR 0067 결정 6).
// core의 경고가 pipeline의 승격 경로를 실제로 타는지 여기서 잇는다.
func TestLoadDeckStrictPromotesUnknownVariantWarning(t *testing.T) {
	files := map[string]string{
		"slides/001_title.slide": "<slide>\n  <text variant=\"hero\">제목</text>\n</slide>\n",
	}

	deck, diags, err := LoadDeck(Options{ProjectDir: newProject(t, files)})
	if err != nil {
		t.Fatalf("경고만으로 빌드가 실패했다: %v", err)
	}
	if deck == nil || len(diags) != 1 || diags[0].Severity != soml.SeverityWarning {
		t.Fatalf("경고 1개가 남아야 한다: %v", diags)
	}
	if !strings.Contains(diags[0].String(), "slides/001_title.slide:2:18: warning:") {
		t.Errorf("진단 = %q, want 파일:행:열과 warning 표시", diags[0].String())
	}

	_, diags, err = LoadDeck(Options{ProjectDir: newProject(t, files), Strict: true})
	if err == nil {
		t.Fatal("--strict는 경고를 에러로 승격해 빌드를 멈춰야 한다")
	}
	if len(diags) != 1 || diags[0].Severity != soml.SeverityError {
		t.Errorf("승격된 진단 = %v, want error 1개", diags)
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
	if _, _, _, err := Render(deck, Options{Targets: []render.Target{"pdf"}}); err == nil {
		t.Error("PDF는 아직 렌더러가 없다 — 에러여야 한다")
	}
}

func TestRenderersCoverAllParsableTargets(t *testing.T) {
	// --target이 받는 이름과 렌더러 목록이 어긋나면 CLI가 통과시킨 타깃이 런타임에 죽는다.
	targets, err := render.ParseTargets("html,pptx")
	if err != nil {
		t.Fatal(err)
	}
	available := Renderers(nil)
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

// 에셋 패스가 조립에 들어와 있는지 — `<image src>`의 기준이 **프로젝트 루트**이고, 표는 슬라이드
// 수집과 **같은 fs.FS**에서 채워진다 (LoadDeck의 fsys 하나).
func TestLoadDeckResolvesAssets(t *testing.T) {
	wide := testPNG(t, 16, 8)
	dir := newProject(t, map[string]string{
		"balpyo.json": `{"title": "그림"}`,
		// 챕터 안 슬라이드가 같은 그림을 가리킨다 — 기준이 슬라이드 파일이면 이 경로가 달라야
		// 하고, 프로젝트 루트 기준이므로 루트 직속 슬라이드와 **같은 표기**다.
		"slides/001_title.slide":      "<slide id=\"a\">\n  <image src=\"assets/wide.png\"/>\n</slide>\n",
		"slides/02_deep/001_in.slide": "<slide id=\"b\">\n  <image src=\"./assets/wide.png\"/>\n</slide>\n",
		"assets/wide.png":             string(wide),
	})
	deck, diags, err := LoadDeck(Options{ProjectDir: dir})
	if err != nil {
		t.Fatal(err)
	}
	if len(diags) != 0 {
		t.Errorf("진단이 없어야 한다: %v", diags)
	}
	// 같은 파일을 두 슬라이드가 가리켜도 표 항목은 하나다 (정규화 키 → 미디어 파트 하나).
	if len(deck.Assets) != 1 {
		t.Fatalf("에셋 %d개, want 1개: %v", len(deck.Assets), deck.Assets.Keys())
	}
	asset := deck.Assets[0]
	if asset.Key != "assets/wide.png" || asset.Format != soml.FormatPNG ||
		asset.Width != 16 || asset.Height != 8 {
		t.Errorf("에셋 = %+v, want assets/wide.png png 16×8", soml.Asset{Key: asset.Key,
			Format: asset.Format, Width: asset.Width, Height: asset.Height})
	}
	if string(asset.Bytes) != string(wide) {
		t.Error("에셋 바이트가 커밋된 파일과 다르다 — 엔진은 바이트를 고치지 않는다")
	}
}

// 없는 `src`는 **빌드 에러**다 (TASKS Iteration 5) — 산출물이 빈 그림 자리를 갖고 나가면
// 저자는 발표 직전에 그것을 발견한다.
func TestLoadDeckRejectsMissingAsset(t *testing.T) {
	dir := newProject(t, map[string]string{
		"slides/001_title.slide": "<slide id=\"a\">\n  <image src=\"assets/none.png\"/>\n</slide>\n",
	})
	_, diags, err := LoadDeck(Options{ProjectDir: dir})
	if err == nil {
		t.Fatal("없는 src는 빌드 에러여야 한다")
	}
	if !diags.HasErrors() || !strings.Contains(diags[0].Message, "파일이 없다") {
		t.Errorf("진단 = %v, want 파일이 없다", diags)
	}
	// 진단은 슬라이드 파일의 값 위치를 가리킨다 — 저자가 에디터에서 바로 고친다.
	if diags[0].File != "slides/001_title.slide" || diags[0].Span.Start.Line != 2 {
		t.Errorf("위치 = %s:%s", diags[0].File, diags[0].Span.Start)
	}
}

// testPNG는 w×h PNG 바이트다 (테스트가 그때 만든다 — 커밋된 바이너리를 픽스처 밖에 두지 않는다).
func testPNG(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	img.Set(0, 0, color.RGBA{R: 255, A: 255})
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// layout 값이 파서 → 파이프라인 → core를 지나 Slide.Layout에 **확정 값**으로 도착한다.
// 생략한 슬라이드는 core가 기본값(content)으로 채운다 — 렌더러가 빈 값을 보지 않는다.
func TestLoadDeckAppliesSlideLayout(t *testing.T) {
	dir := newProject(t, map[string]string{
		"slides/001_free.slide":    `<slide id="free" layout="canvas"></slide>`,
		"slides/002_default.slide": `<slide id="default"></slide>`,
	})
	deck, diags, err := LoadDeck(Options{ProjectDir: dir})
	if err != nil {
		t.Fatal(err)
	}
	if len(diags) != 0 {
		t.Errorf("진단이 없어야 한다: %v", diags)
	}
	if deck.Slides[0].Layout != soml.LayoutCanvas {
		t.Errorf("Slides[0].Layout = %q, want canvas", deck.Slides[0].Layout)
	}
	if deck.Slides[1].Layout != soml.LayoutContent {
		t.Errorf("Slides[1].Layout = %q, want content(기본값)", deck.Slides[1].Layout)
	}
}

// 어휘 밖 layout은 파일:행:열을 가진 빌드 에러다 — core가 값 층에서 잡는다. 표준 카탈로그
// 11종에도 (활성 테마에도) 없는 이름만 여기 해당한다.
func TestLoadDeckRejectsUnknownLayout(t *testing.T) {
	dir := newProject(t, map[string]string{
		"slides/001_title.slide": "<slide id=\"a\" layout=\"corp-hero\"></slide>\n",
	})
	_, diags, err := LoadDeck(Options{ProjectDir: dir})
	if err == nil {
		t.Fatal("어휘 밖 layout은 빌드 에러여야 한다")
	}
	if len(diags) != 1 || !strings.Contains(diags[0].Message, "알 수 없는 layout") {
		t.Fatalf("진단 = %v, want '알 수 없는 layout'", diags)
	}
	// 진단은 슬라이드 파일의 값 위치를 가리킨다 — 저자가 에디터에서 바로 고친다.
	if diags[0].File != "slides/001_title.slide" || diags[0].Span.Start.Line != 1 || diags[0].Span.Start.Col != 23 {
		t.Errorf("위치 = %s:%s, want slides/001_title.slide:1:23", diags[0].File, diags[0].Span.Start)
	}
}

// filesEqual은 두 산출 파일 집합이 경로·바이트까지 같은지 본다.
func filesEqual(t *testing.T, a, b render.Files) {
	t.Helper()
	ap, bp := a.Paths(), b.Paths()
	if len(ap) != len(bp) {
		t.Fatalf("파일 수가 다르다: %v vs %v", ap, bp)
	}
	for i, name := range ap {
		if bp[i] != name {
			t.Fatalf("파일 경로가 다르다: %v vs %v", ap, bp)
		}
		if !bytes.Equal(a[name], b[name]) {
			t.Errorf("%s의 바이트가 다르다", name)
		}
	}
}

// layout="content"는 기본 스테이지(세로 스택)의 이름이므로 그 **웹 산출물**은 layout 없는
// 덱과 바이트가 같아야 한다 (DefaultSlideLayout == content — 이름을 명시해도 아무것도 바뀌지
// 않는다). **canvas는 다르다**: 항목 04가 캔버스를 실제 스테이지(balpyo-stage-canvas = 단일 셀
// grid 겹침)로 구현했으므로 웹 산출물이 base와 갈라지는 것이 정답이다 — 그 갈라짐의 내용
// (스테이지 클래스 + 조건부 stack CSS)은 render/web의 TestCanvasStageIsStack·
// TestContentStageIsNotCanvas·TestStackCSSOnlyWhenStackOrCanvasUsed가 소유한다. 여기서는
// 파이프라인 수준에서 content가 여전히 무변경이고 canvas만 갈라지는지를 결정적으로 못 박는다
// (웹 렌더는 브라우저가 필요 없다).
//
// (항목 06의 발견: 이 검사는 항목 02가 canvas를 무변경으로 단정한 채 남았고, 항목 04가 캔버스
// 스테이지를 구현하면서 갱신하지 않아 커밋 시점에 빨간 채였다 — NOTES 06/06 "앞 항목의 버그".)
func TestBuildLayoutWebOutputInvariant(t *testing.T) {
	webFiles := func(attr string) render.Files {
		dir := newProject(t, map[string]string{
			"balpyo.json":            `{"title": "t"}`,
			"slides/001_title.slide": "<slide id=\"a\"" + attr + ">\n  <h1>제목</h1>\n  <p>본문 한 줄</p>\n</slide>\n",
		})
		deck, _, err := LoadDeck(Options{ProjectDir: dir})
		if err != nil {
			t.Fatal(err)
		}
		files, _, _, err := Render(deck, Options{Targets: []render.Target{render.TargetHTML}})
		if err != nil {
			t.Fatal(err)
		}
		return files[render.TargetHTML]
	}
	base := webFiles("")
	// content는 기본 스테이지의 이름 — 웹 산출물이 무변경이다.
	filesEqual(t, base, webFiles(` layout="content"`))
	// canvas는 실제 스테이지 — base와 달라야 한다 (갈라짐의 내용 판정은 render/web의 몫).
	if canvas := webFiles(` layout="canvas"`); bytes.Equal(canvas["index.html"], base["index.html"]) {
		t.Error(`layout="canvas"의 웹 산출물이 base와 같다 — 항목 04의 캔버스 스테이지가 웹에 닿지 않았다`)
	}
}

// canvas는 PPTX 파트를 새로 요구하지 않는다 (모든 슬라이드가 slideLayout1.xml 하나를
// 가리키고 도형은 절대 좌표다). 빈 슬라이드는 잴 요소가 없어 브라우저 없이 렌더되므로,
// 이 검사가 layout 이름이 PPTX 바이트에 새지 않는지를 결정적으로 못 박는다.
func TestBuildLayoutPPTXOutputInvariant(t *testing.T) {
	pptxFiles := func(attr string) render.Files {
		dir := newProject(t, map[string]string{
			"balpyo.json":            `{"title": "t"}`,
			"slides/001_title.slide": "<slide id=\"a\"" + attr + "></slide>\n",
		})
		deck, _, err := LoadDeck(Options{ProjectDir: dir})
		if err != nil {
			t.Fatal(err)
		}
		files, _, _, err := Render(deck, Options{Targets: []render.Target{render.TargetPPTX}})
		if err != nil {
			t.Fatal(err)
		}
		return files[render.TargetPPTX]
	}
	base := pptxFiles("")
	filesEqual(t, base, pptxFiles(` layout="content"`))
	filesEqual(t, base, pptxFiles(` layout="canvas"`))
}
