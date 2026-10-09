// `balpyo check`의 판정 계약이다: Build와 **같은 판정**, 디스크에는 **아무것도**.
package pipeline_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/k2sebeom/balpyo/internal/pipeline"
	"github.com/k2sebeom/balpyo/internal/testkit"
	"github.com/k2sebeom/balpyo/soml"
)

// 깨끗한 덱: 진단이 없고, 출력 디렉토리가 **생기지 않는다**. 판정만 필요한 곳(CI·에디터)이
// 워크스페이스를 더럽히지 않는 것이 이 명령의 절반이다.
func TestCheckWritesNothing(t *testing.T) {
	noBrowser(t)
	dir := project(t, map[string]string{
		"balpyo.json":            `{"title": "검사"}`,
		"slides/001_title.slide": textSlide,
	})

	result, err := pipeline.Check(pipeline.Options{ProjectDir: dir})
	if err != nil {
		t.Fatalf("깨끗한 덱이 실패했다: %v", err)
	}
	if len(result.Diagnostics) != 0 {
		t.Errorf("진단이 있다:\n%s", result.Diagnostics)
	}
	if len(result.Deck.Slides) != 1 {
		t.Errorf("슬라이드 %d장, want 1", len(result.Deck.Slides))
	}
	if result.Measured {
		t.Error("배치 검사를 켜지 않았는데 쟀다")
	}
	// 디렉토리 목록이 입력 그대로여야 한다 — dist/가 생기면 여기서 걸린다.
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, e := range entries {
		names = append(names, e.Name())
	}
	if strings.Join(names, ",") != "balpyo.json,slides" {
		t.Errorf("check가 무언가를 남겼다: %v", names)
	}
}

// 경고는 남고 판정은 통과다 (Build와 같은 규칙). `--strict`가 그 경고를 에러로 만든다.
func TestCheckKeepsWarningsAndStrictPromotes(t *testing.T) {
	noBrowser(t)
	files := map[string]string{
		"slides/001_title.slide": "<slide>\n  <text variant=\"hero\">제목</text>\n</slide>\n",
	}

	result, err := pipeline.Check(pipeline.Options{ProjectDir: project(t, files)})
	if err != nil {
		t.Fatalf("경고가 검사를 실패시켰다: %v", err)
	}
	if len(result.Diagnostics) != 1 || result.Diagnostics[0].Severity != soml.SeverityWarning {
		t.Fatalf("경고 1건이 남아야 한다: %s", result.Diagnostics)
	}

	result, err = pipeline.Check(pipeline.Options{ProjectDir: project(t, files), Strict: true})
	if err == nil {
		t.Fatal("--strict가 경고를 통과시켰다")
	}
	if !result.Diagnostics.HasErrors() {
		t.Errorf("승격되지 않은 진단: %s", result.Diagnostics)
	}
}

// 진단 에러는 에러다 — 같은 덱에서 Build가 실패하는 것과 같은 판정이어야 "check는 통과했는데
// build가 실패한다"가 불가능해진다.
func TestCheckFailsOnDiagnosticErrors(t *testing.T) {
	noBrowser(t)
	dir := project(t, map[string]string{
		"slides/001_title.slide": `<slide><text role="heding">제목</text></slide>`,
	})
	result, err := pipeline.Check(pipeline.Options{ProjectDir: dir})
	if err == nil {
		t.Fatal("알 수 없는 role을 통과시켰다")
	}
	var de *soml.DiagnosticsError
	if !errors.As(err, &de) {
		t.Fatalf("진단 에러가 아니다: %v", err)
	}
	if !result.Diagnostics.HasErrors() {
		t.Errorf("결과에 에러 진단이 없다: %s", result.Diagnostics)
	}
	// 실패한 검사도 디스크를 건드리지 않는다.
	if _, err := os.Stat(filepath.Join(dir, "dist")); !os.IsNotExist(err) {
		t.Errorf("출력 디렉토리를 만들었다: %v", err)
	}
}

// 배치 경고는 플래그가 켠다 (Build와 같은 규칙 — 브라우저 기동을 기본으로 내지 않는다).
func TestCheckLayoutMeasuresAndWarns(t *testing.T) {
	testkit.RequireChromium(t)
	dir := overflowProject(t)

	// 켜지 않으면 재지 않는다: 같은 덱에서 배치 경고가 나오지 않는다.
	quiet, err := pipeline.Check(pipeline.Options{ProjectDir: dir})
	if err != nil {
		t.Fatalf("배치 경고 없이 실패했다: %v", err)
	}
	if quiet.Measured || len(quiet.Diagnostics) != 0 {
		t.Errorf("플래그 없이 배치를 봤다: measured=%v\n%s", quiet.Measured, quiet.Diagnostics)
	}

	result, err := pipeline.Check(pipeline.Options{ProjectDir: dir, CheckLayout: true})
	if err != nil {
		t.Fatalf("경고가 검사를 실패시켰다: %v", err)
	}
	if !result.Measured {
		t.Error("배치 검사를 켰는데 재지 않았다")
	}
	if len(result.Diagnostics) != 3 {
		t.Fatalf("배치 진단 %d건, want 3건:\n%s", len(result.Diagnostics), result.Diagnostics)
	}
	// 승격되면 실패한다 — CI 레시피(`check --check-layout --strict`)가 이 줄이다.
	if _, err := pipeline.Check(pipeline.Options{
		ProjectDir: dir, CheckLayout: true, Strict: true,
	}); err == nil {
		t.Error("--strict가 배치 경고를 통과시켰다")
	}
	// 재는 것은 임시 디렉토리에서 끝난다.
	if _, err := os.Stat(filepath.Join(dir, "dist")); !os.IsNotExist(err) {
		t.Errorf("측정이 출력 디렉토리를 만들었다: %v", err)
	}
}
