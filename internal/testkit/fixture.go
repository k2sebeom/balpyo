package testkit

import (
	"archive/zip"
	"bytes"
	"io"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/k2sebeom/balpyo/internal/pipeline"
	"github.com/k2sebeom/balpyo/render"
)

// 픽스처 레이아웃. 픽스처 하나가 곧 완전한 balpyo 프로젝트이므로 사람이 그 디렉토리를
// `balpyo build`로 직접 빌드해 볼 수 있다.
const (
	// FixturesDir는 픽스처 루트다 (저장소 기준).
	FixturesDir = "testdata/fixtures"
	// goldenDirName은 픽스처 안의 골든 디렉토리 이름이다.
	goldenDirName = "golden"
	// pptxGoldenPrefix는 PPTX 파트 골든이 들어가는 하위 경로다.
	pptxGoldenPrefix = "pptx"
	// diagnosticsGolden은 경고가 있을 때만 생기는 골든이다.
	diagnosticsGolden = "diagnostics.txt"
)

// Fixture는 픽스처 하나다 — `testdata/fixtures/<NN>_<이름>/`.
type Fixture struct {
	// Name은 픽스처 디렉토리 이름이다 (`00_empty`).
	Name string
	// Dir는 픽스처 프로젝트 루트다 (절대 경로).
	Dir string
}

// GoldenDir는 이 픽스처의 골든 디렉토리다.
func (f Fixture) GoldenDir() string { return filepath.Join(f.Dir, goldenDirName) }

// Fixtures는 픽스처 목록을 이름 순으로 돌려준다. 이름의 숫자 접두사가 iteration 순서이자
// 실행 순서다.
func Fixtures(t testing.TB) []Fixture {
	t.Helper()
	root := filepath.Join(RepoRoot(t), filepath.FromSlash(FixturesDir))
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatalf("픽스처 디렉토리를 읽을 수 없다: %v", err)
	}
	var fixtures []Fixture
	for _, entry := range entries {
		// `_`로 시작하는 디렉토리는 작업 중인 픽스처다 — 슬라이드 수집 규칙과 같은 관례를 쓴다.
		if !entry.IsDir() || strings.HasPrefix(entry.Name(), "_") || strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		fixtures = append(fixtures, Fixture{Name: entry.Name(), Dir: filepath.Join(root, entry.Name())})
	}
	if len(fixtures) == 0 {
		t.Fatalf("픽스처가 없다: %s", root)
	}
	sort.Slice(fixtures, func(i, j int) bool { return fixtures[i].Name < fixtures[j].Name })
	return fixtures
}

// Build는 픽스처를 두 타깃 모두로 빌드한다. 산출물은 임시 디렉토리에 쓰므로 픽스처
// 디렉토리는 입력 전용으로 남는다 (골든만 사람이 커밋한다).
func (f Fixture) Build(t testing.TB) *pipeline.Result {
	t.Helper()
	result, err := pipeline.Build(pipeline.Options{
		ProjectDir: f.Dir,
		Targets:    []render.Target{render.TargetHTML, render.TargetPPTX},
		OutDir:     t.TempDir(),
	})
	if err != nil {
		t.Fatalf("%s 빌드 실패: %v", f.Name, err)
	}
	return result
}

// AssertGolden은 빌드 산출물을 골든 트리와 비교한다.
func (f Fixture) AssertGolden(t testing.TB, result *pipeline.Result) {
	t.Helper()
	AssertGoldenTree(t, f.GoldenDir(), f.goldenFiles(t, result))
}

// goldenFiles는 골든으로 남길 파일 묶음을 만든다.
//
// PPTX는 zip 바이트 대신 **압축을 푼 파트별 XML**을 남긴다. 이유가 두 개다: zip 바이트는
// 압축기 구현이 바뀌면 내용이 같은데도 달라지고, 무엇보다 XML 골든은 git diff에서 읽힌다 —
// "도형이 하나 늘었다"가 diff로 보이는 것이 이 골든의 목적이다.
func (f Fixture) goldenFiles(t testing.TB, result *pipeline.Result) map[string][]byte {
	t.Helper()
	files := map[string][]byte{}
	for _, name := range result.Written {
		data, err := os.ReadFile(filepath.Join(result.OutDir, filepath.FromSlash(name)))
		if err != nil {
			t.Fatalf("산출물을 읽을 수 없다: %v", err)
		}
		if strings.EqualFold(path.Ext(name), ".pptx") {
			for part, partData := range PPTXParts(t, data) {
				files[path.Join(pptxGoldenPrefix, part)] = partData
			}
			continue
		}
		files[name] = data
	}
	// 경고는 산출물의 일부다 — 골든에 남겨야 "언제부터 경고가 생겼나"가 diff에 보인다.
	if diags := result.Diagnostics; len(diags) > 0 {
		var b strings.Builder
		for _, d := range diags {
			b.WriteString(d.String())
			b.WriteString("\n")
		}
		files[diagnosticsGolden] = []byte(b.String())
	}
	return files
}

// PPTXParts는 pptx 바이트에서 파트별 내용을 뽑는다. PPTX 구조 테스트가 XML을 어서션할 때도
// 쓰는 진입점이다.
func PPTXParts(t testing.TB, data []byte) map[string][]byte {
	t.Helper()
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatalf("pptx를 열 수 없다: %v", err)
	}
	parts := make(map[string][]byte, len(zr.File))
	for _, entry := range zr.File {
		rc, err := entry.Open()
		if err != nil {
			t.Fatalf("파트 %s를 열 수 없다: %v", entry.Name, err)
		}
		content, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			t.Fatalf("파트 %s를 읽을 수 없다: %v", entry.Name, err)
		}
		parts[entry.Name] = content
	}
	return parts
}

// Output은 빌드 산출물 하나의 절대 경로다. 렌더 헬퍼에 넘길 파일을 집을 때 쓴다.
func (f Fixture) Output(t testing.TB, result *pipeline.Result, ext string) string {
	t.Helper()
	for _, name := range result.Written {
		if strings.EqualFold(path.Ext(name), ext) {
			return filepath.Join(result.OutDir, filepath.FromSlash(name))
		}
	}
	t.Fatalf("%s: 확장자가 %s인 산출물이 없다 (%v)", f.Name, ext, result.Written)
	return ""
}
