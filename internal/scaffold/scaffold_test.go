package scaffold

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/k2sebeom/balpyo/core"
	"github.com/k2sebeom/balpyo/internal/pipeline"
	"github.com/k2sebeom/balpyo/render"
)

// 스캐폴딩의 계약은 "파일이 생긴다"가 아니다: **그 덱이 경고 0으로 빌드된다**.
// 여기서는 브라우저 없이 판정할 수 있는 것까지 보고(수집·파싱·값 층·에셋·검증), 배치까지 잰
// 판정은 cmd/balpyo/init_test.go가 컨테이너에서 한다.
func TestScaffoldDeckLoadsWithoutDiagnostics(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "my-talk")
	result, err := Write(Options{Dir: dir, Title: "첫 덱", Author: "나"})
	if err != nil {
		t.Fatal(err)
	}

	want := []string{
		core.ConfigFile,
		"slides/001_title.slide",
		"slides/002_content.slide",
		"slides/003_end.slide",
		GitignoreFile,
	}
	if strings.Join(result.Written, ",") != strings.Join(want, ",") {
		t.Errorf("Written = %v\nwant %v", result.Written, want)
	}
	for _, name := range result.Written {
		if _, err := os.Stat(filepath.Join(dir, filepath.FromSlash(name))); err != nil {
			t.Errorf("%s가 없다: %v", name, err)
		}
	}

	deck, diags, err := pipeline.LoadDeck(pipeline.Options{ProjectDir: dir, Strict: true})
	if err != nil {
		t.Fatalf("스캐폴딩 덱이 빌드되지 않는다: %v", err)
	}
	// --strict로 읽었으므로 경고 하나도 에러가 된다 — 위 err가 nil이면 진단이 비어야 한다.
	if len(diags) != 0 {
		t.Errorf("진단이 있다:\n%s", diags)
	}
	if len(deck.Slides) != 3 {
		t.Errorf("슬라이드 %d장, want 3", len(deck.Slides))
	}
	// 제목은 표지에도 실린다 — 치환이 빠지면 저자는 자기 제목을 두 번 쓴다.
	if deck.Title != "첫 덱" {
		t.Errorf("deck.Title = %q", deck.Title)
	}
	cover, err := os.ReadFile(filepath.Join(dir, "slides", "001_title.slide"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"첫 덱", "나"} {
		if !strings.Contains(string(cover), want) {
			t.Errorf("표지에 %q가 치환되지 않았다:\n%s", want, cover)
		}
	}
	if strings.Contains(string(cover), "{{") {
		t.Errorf("표지에 치환되지 않은 템플릿이 남았다:\n%s", cover)
	}
}

// 스캐폴딩이 쓴 설정은 로더가 **그대로** 읽는다. 키 이름이 틀리면 DisallowUnknownFields가
// 멈추므로, 이 왕복이 템플릿과 스키마가 갈라지는 것을 잡는다.
func TestScaffoldConfigRoundTripsThroughLoader(t *testing.T) {
	dir := t.TempDir()
	if _, err := Write(Options{Dir: dir, Title: "왕복", Author: "저자", Aspect: "4:3"}); err != nil {
		t.Fatal(err)
	}
	cfg, err := core.LoadConfig(dir)
	if err != nil {
		t.Fatalf("스캐폴딩이 쓴 %s를 로더가 읽지 못한다: %v", core.ConfigFile, err)
	}
	if cfg.Title != "왕복" || cfg.Author != "저자" {
		t.Errorf("제목·저자가 설정에 없다: %+v", cfg)
	}
	if cfg.Aspect != "4:3" {
		t.Errorf("aspect = %q, want 4:3", cfg.Aspect)
	}
	if cfg.Theme != "minimal" || cfg.Lang != DefaultLang || cfg.OutDir != "dist" {
		t.Errorf("기본값이 설정에 없다: %+v", cfg)
	}
}

// 제목을 주지 않으면 디렉토리 이름이 제목이다 (core.LoadConfig의 규칙과 같다).
func TestScaffoldTitleDefaultsToDirName(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "분기-리뷰")
	if _, err := Write(Options{Dir: dir}); err != nil {
		t.Fatal(err)
	}
	cfg, err := core.LoadConfig(dir)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Title != "분기-리뷰" {
		t.Errorf("title = %q, want 분기-리뷰", cfg.Title)
	}
	// 발표자를 주지 않으면 **자리표시자**가 남는다 — 빈 텍스트는 고칠 자리를 숨긴다.
	cover, err := os.ReadFile(filepath.Join(dir, "slides", "001_title.slide"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(cover), "Presenter") {
		t.Errorf("표지에 발표자 자리표시자가 없다:\n%s", cover)
	}
}

// 덮어쓰지 않는다: 같은 자리에 두 번 돌리면 멈추고, 있던 파일은 그대로다.
func TestScaffoldRefusesToOverwrite(t *testing.T) {
	t.Run("설정이 있다", func(t *testing.T) {
		dir := t.TempDir()
		if _, err := Write(Options{Dir: dir}); err != nil {
			t.Fatal(err)
		}
		mine := filepath.Join(dir, "slides", "001_title.slide")
		if err := os.WriteFile(mine, []byte("<slide id=\"mine\"></slide>\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := Write(Options{Dir: dir}); err == nil {
			t.Fatal("이미 프로젝트인 자리에 다시 썼다")
		}
		data, err := os.ReadFile(mine)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(data), "mine") {
			t.Errorf("내가 쓴 슬라이드가 덮어써졌다:\n%s", data)
		}
	})

	// 설정은 없지만 슬라이드가 있는 자리도 프로젝트로 본다 — balpyo.json은 선택이다.
	t.Run("슬라이드만 있다", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.MkdirAll(filepath.Join(dir, "slides"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "slides", "001_a.slide"), []byte("<slide></slide>"), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := Write(Options{Dir: dir}); err == nil {
			t.Fatal("슬라이드가 있는 자리에 썼다")
		}
		if _, err := os.Stat(filepath.Join(dir, core.ConfigFile)); !os.IsNotExist(err) {
			t.Errorf("멈추기로 했는데 설정을 썼다: %v", err)
		}
	})

	// 저자가 쓴 .gitignore는 손대지 않는다 (있으면 건너뛴다).
	t.Run("gitignore가 있다", func(t *testing.T) {
		dir := t.TempDir()
		ignore := filepath.Join(dir, GitignoreFile)
		if err := os.WriteFile(ignore, []byte("내-것/\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		result, err := Write(Options{Dir: dir})
		if err != nil {
			t.Fatal(err)
		}
		data, err := os.ReadFile(ignore)
		if err != nil {
			t.Fatal(err)
		}
		if string(data) != "내-것/\n" {
			t.Errorf(".gitignore를 덮어썼다:\n%s", data)
		}
		for _, name := range result.Written {
			if name == GitignoreFile {
				t.Error("쓰지 않은 파일을 썼다고 보고한다")
			}
		}
	})
}

// 값 층이 읽지 않는 테마 이름을 설정에 써 주지 않는다.
func TestScaffoldRejectsUnknownTheme(t *testing.T) {
	dir := t.TempDir()
	_, err := Write(Options{Dir: dir, Theme: "corp"})
	if err == nil {
		t.Fatal("없는 테마를 받아들였다")
	}
	if !strings.Contains(err.Error(), "minimal") {
		t.Errorf("에러가 쓸 수 있는 테마를 말하지 않는다: %v", err)
	}
	// 판정은 디스크를 건드리기 전이다 — 반쯤 만들어진 프로젝트가 남지 않는다.
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Errorf("거부했는데 %d개를 썼다", len(entries))
	}
}

func TestScaffoldRejectsBadAspect(t *testing.T) {
	if _, err := Write(Options{Dir: t.TempDir(), Aspect: "21:9"}); err == nil {
		t.Fatal("없는 종횡비를 받아들였다")
	}
}

// 언어 태그가 아닌 값은 init이 쓰기 전에 멈춘다 — 빌드가 거부할 설정을 만들지 않는다.
func TestScaffoldRejectsBadLang(t *testing.T) {
	dir := t.TempDir()
	if _, err := Write(Options{Dir: dir, Lang: "korean"}); err == nil {
		t.Fatal(`"korean"을 언어 태그로 받아들였다`)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Errorf("거부했는데 %d개를 썼다", len(entries))
	}
}

// 스캐폴딩 덱은 두 타깃의 어휘 안에 있다 — 웹은 브라우저 없이 렌더되므로 여기서 판정한다.
// (PPTX는 좌표가 필요해 측정을 요구하므로 cmd/balpyo/init_test.go가 컨테이너에서 본다.)
func TestScaffoldDeckRendersWeb(t *testing.T) {
	dir := t.TempDir()
	if _, err := Write(Options{Dir: dir}); err != nil {
		t.Fatal(err)
	}
	deck, _, err := pipeline.LoadDeck(pipeline.Options{ProjectDir: dir})
	if err != nil {
		t.Fatal(err)
	}
	rendered, _, diags, err := pipeline.Render(deck, pipeline.Options{
		Targets: []render.Target{render.TargetHTML},
	})
	if err != nil {
		t.Fatalf("웹 렌더 실패: %v", err)
	}
	if len(diags) != 0 {
		t.Errorf("렌더 진단이 있다:\n%s", diags)
	}
	if len(rendered[render.TargetHTML]) == 0 {
		t.Error("웹 산출물이 비었다")
	}
}
