package testkit

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAssertGoldenWritesThenCompares(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "out.html")

	// 갱신 모드는 상위 디렉토리까지 만들고 쓴다.
	t.Setenv(EnvUpdateGolden, "1")
	AssertGolden(t, path, []byte("<p>안녕</p>\n"))
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("골든이 쓰이지 않았다: %v", err)
	}
	if string(data) != "<p>안녕</p>\n" {
		t.Errorf("골든 내용 = %q", data)
	}

	// 비교 모드에서 같은 내용은 통과한다.
	t.Setenv(EnvUpdateGolden, "")
	AssertGolden(t, path, []byte("<p>안녕</p>\n"))
}

func TestStaleGoldensAreDetected(t *testing.T) {
	dir := t.TempDir()
	write := func(name, body string) {
		t.Helper()
		dest := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(dest, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("index.html", "지금도 나온다")
	write("pptx/ppt/slides/slide1.xml", "지금도 나온다")
	// 산출물 이름이 바뀌어 남은 골든. 이걸 잡지 못하면 아무도 보지 않는 파일을 비교하며
	// 테스트가 통과한다.
	write("pptx/ppt/slides/slide2.xml", "옛 산출물")

	files := map[string][]byte{
		"index.html":                 []byte("지금도 나온다"),
		"pptx/ppt/slides/slide1.xml": []byte("지금도 나온다"),
	}
	stale := staleGoldens(t, dir, files)
	if len(stale) != 1 || stale[0] != "pptx/ppt/slides/slide2.xml" {
		t.Fatalf("남은 골든 = %v, want [pptx/ppt/slides/slide2.xml]", stale)
	}

	// 갱신 모드는 남은 골든을 지운다.
	t.Setenv(EnvUpdateGolden, "1")
	AssertGoldenTree(t, dir, files)
	if _, err := os.Stat(filepath.Join(dir, "pptx/ppt/slides/slide2.xml")); !os.IsNotExist(err) {
		t.Errorf("남은 골든이 지워지지 않았다: %v", err)
	}
}

func TestStaleGoldensOnMissingDirectory(t *testing.T) {
	// 골든이 아직 없는 새 픽스처에서 순회가 실패하면 `make golden`이 첫 실행에서 죽는다.
	if stale := staleGoldens(t, filepath.Join(t.TempDir(), "없다"), nil); len(stale) != 0 {
		t.Errorf("남은 골든 = %v, want 없음", stale)
	}
}

func TestDiffReport(t *testing.T) {
	report := diffReport([]byte("a\nb\nc\n"), []byte("a\nB\nc\n"))
	for _, want := range []string{"want 4줄", "-   2 | b", "+   2 | B"} {
		if !strings.Contains(report, want) {
			t.Errorf("diff 요약에 %q가 없다:\n%s", want, report)
		}
	}
	// 바이너리는 줄 단위 diff가 의미 없다 — 크기와 첫 차이 위치만 알려준다.
	binary := diffReport([]byte{0, 1, 2}, []byte{0, 9, 2})
	if !strings.Contains(binary, "바이너리") || !strings.Contains(binary, "오프셋 1") {
		t.Errorf("바이너리 요약 = %q", binary)
	}
}
