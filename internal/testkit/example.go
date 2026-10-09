package testkit

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/k2sebeom/balpyo/internal/pipeline"
	"github.com/k2sebeom/balpyo/render"
)

// ExamplesDir는 예제 덱 루트다 (저장소 기준).
const ExamplesDir = "examples/decks"

// ExampleThemesDir는 예제 테마 루트다 (저장소 기준). 테마 하나가 디렉토리 하나이고, 사용자는
// 그 디렉토리를 자기 덱의 `themes/<이름>/`으로 복사해 쓴다.
const ExampleThemesDir = "examples/themes"

// Example은 예제 덱 하나다 — `examples/decks/<이름>/`.
//
// 픽스처와 무엇이 다른가: 픽스처는 어휘 하나를 겨눈 2~4장이고 골든이 그 산출물을 바이트로
// 못 박는다. 예제는 **사람이 실제로 쓴 덱**이라 어휘 수십 개가 한 장에 겹쳐 있고, 그래서
// 골든을 걸지 않는다 — 어휘를 하나 열 때마다 28장의 XML diff를 사람이 넘기게 되면 그 diff는
// 읽히지 않는 채 승인되고, 승인되지 않는 골든은 게이트가 아니라 게이트처럼 보이는 것이다.
// 예제가 지키는 것은 하나다: **빌드되고 조용하다.**
type Example struct {
	// Name은 예제 디렉토리 이름이다 (`balpyo-intro`).
	Name string
	// Dir는 예제 프로젝트 루트다 (절대 경로).
	Dir string
}

// Examples는 예제 목록을 이름 순으로 돌려준다 — 목록은 디렉토리를 걸어 얻으므로 새 예제가
// 자동으로 게이트에 들어온다 (수집 규칙은 픽스처와 같다: `_`·`.` 접두사는 작업 중이다).
func Examples(t testing.TB) []Example {
	t.Helper()
	root := filepath.Join(RepoRoot(t), filepath.FromSlash(ExamplesDir))
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatalf("예제 디렉토리를 읽을 수 없다: %v", err)
	}
	var examples []Example
	for _, entry := range entries {
		if !entry.IsDir() || strings.HasPrefix(entry.Name(), "_") || strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		examples = append(examples, Example{Name: entry.Name(), Dir: filepath.Join(root, entry.Name())})
	}
	if len(examples) == 0 {
		t.Fatalf("예제가 없다: %s", root)
	}
	sort.Slice(examples, func(i, j int) bool { return examples[i].Name < examples[j].Name })
	return examples
}

// Build는 예제를 두 타깃 모두로, **경고를 에러로 승격해서** 빌드한다.
//
// `Strict`가 게이트의 전부다: 넘침·이탈·겹침(pptx 타깃이 지오메트리 패스를 돌리므로 공짜로
// 들어온다)과 테마에 없는 variant가 모두 실패가 된다. 경고가 통과인 것은 사용자
// 덱의 규칙이고, 예제는 제품이 쓴 문서라 다른 기준을 받는다 — 경고를 내는 예제는 읽는 사람에게
// "경고는 무시해도 된다"를 가르친다.
//
// 산출물은 임시 디렉토리에 쓴다: 예제 디렉토리는 입력 전용이고 `examples/decks/*/dist`는 커밋하지
// 않는다. 도구 정책은 픽스처와 같은 자리에서 적용한다 (CI에서는 실패, 브라우저 없는
// 호스트에서는 건너뛴다).
func (e Example) Build(t testing.TB) *pipeline.Result {
	t.Helper()
	session := SharedSession(t)
	result, err := pipeline.Build(pipeline.Options{
		ProjectDir: e.Dir,
		Targets:    []render.Target{render.TargetHTML, render.TargetPPTX},
		OutDir:     t.TempDir(),
		Measurer:   session,
		Strict:     true,
	})
	if err != nil {
		// 승격된 경고는 진단으로 이미 찍혔으므로 여기서 다시 나열하지 않는다 — 호출자가
		// 진단을 보고한다.
		t.Fatalf("%s 빌드 실패: %v", e.Name, err)
	}
	return result
}

// ExampleTheme은 예제 테마 하나다 — `examples/themes/<이름>/`.
type ExampleTheme struct {
	// Name은 테마 디렉토리 이름이자 `balpyo.json`의 `theme`에 적는 이름이다.
	Name string
	// Dir는 테마 패키지 루트다 (절대 경로).
	Dir string
}

// ExampleThemes는 예제 테마 목록을 이름 순으로 돌려준다 (수집 규칙은 예제 덱과 같다).
func ExampleThemes(t testing.TB) []ExampleTheme {
	t.Helper()
	root := filepath.Join(RepoRoot(t), filepath.FromSlash(ExampleThemesDir))
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatalf("예제 테마 디렉토리를 읽을 수 없다: %v", err)
	}
	var themes []ExampleTheme
	for _, entry := range entries {
		if !entry.IsDir() || strings.HasPrefix(entry.Name(), "_") || strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		themes = append(themes, ExampleTheme{Name: entry.Name(), Dir: filepath.Join(root, entry.Name())})
	}
	if len(themes) == 0 {
		t.Fatalf("예제 테마가 없다: %s", root)
	}
	sort.Slice(themes, func(i, j int) bool { return themes[i].Name < themes[j].Name })
	return themes
}

// BuildWithTheme은 예제 덱을 예제 테마로 Build와 같은 기준(두 타깃 · 경고는 에러)으로 빌드한다.
//
// 사용자가 하는 그대로 한다: 덱을 임시 디렉토리로 복사하고, 테마를 그 안의 `themes/<이름>/`에
// 넣고, `--theme`으로 고른다. 예제 디렉토리는 건드리지 않는다.
func (e Example) BuildWithTheme(t testing.TB, theme ExampleTheme) *pipeline.Result {
	t.Helper()
	project := t.TempDir()
	if err := os.CopyFS(project, os.DirFS(e.Dir)); err != nil {
		t.Fatalf("예제 %s를 복사할 수 없다: %v", e.Name, err)
	}
	if err := os.CopyFS(filepath.Join(project, "themes", theme.Name), os.DirFS(theme.Dir)); err != nil {
		t.Fatalf("예제 테마 %s를 복사할 수 없다: %v", theme.Name, err)
	}
	session := SharedSession(t)
	result, err := pipeline.Build(pipeline.Options{
		ProjectDir: project,
		Targets:    []render.Target{render.TargetHTML, render.TargetPPTX},
		OutDir:     t.TempDir(),
		Measurer:   session,
		Strict:     true,
		Theme:      theme.Name,
	})
	if err != nil {
		t.Fatalf("%s를 테마 %s로 빌드하지 못했다: %v", e.Name, theme.Name, err)
	}
	return result
}
