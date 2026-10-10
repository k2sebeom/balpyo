package testkit

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"unicode/utf8"
)

// AssertGolden은 산출 바이트를 골든 파일과 비교한다. 갱신 모드(`make golden`)에서는
// 비교 대신 파일을 쓴다.
func AssertGolden(t testing.TB, goldenPath string, got []byte) {
	t.Helper()
	if UpdateGolden() {
		writeGolden(t, goldenPath, got)
		return
	}
	want, err := os.ReadFile(goldenPath)
	if os.IsNotExist(err) {
		t.Fatalf("golden missing: %s — create it with `make golden`", goldenPath)
	}
	if err != nil {
		t.Fatalf("cannot read golden: %v", err)
	}
	if !bytes.Equal(want, got) {
		t.Errorf("differs from golden: %s\n%s\nto update, run `make golden`", goldenPath, diffReport(want, got))
	}
}

// AssertGoldenTree는 산출 파일 묶음 전체를 골든 디렉토리와 비교한다.
//
// 파일 하나씩 비교하는 것과 다른 점은 **남은 골든을 잡는다**는 것이다. 산출 파일 이름이
// 바뀌거나 사라지면 옛 골든이 그대로 남아 아무도 보지 않는 파일을 비교하며 통과한다.
// 갱신 모드에서는 그런 파일을 지운다.
func AssertGoldenTree(t testing.TB, goldenDir string, files map[string][]byte) {
	t.Helper()
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		AssertGolden(t, filepath.Join(goldenDir, filepath.FromSlash(name)), files[name])
	}

	for _, stale := range staleGoldens(t, goldenDir, files) {
		if UpdateGolden() {
			if err := os.Remove(filepath.Join(goldenDir, filepath.FromSlash(stale))); err != nil {
				t.Fatalf("cannot remove stale golden %s: %v", stale, err)
			}
			t.Logf("removed stale golden: %s", stale)
			continue
		}
		t.Errorf("golden %s has no matching output file — if it was renamed, run `make golden`", stale)
	}
}

// staleGoldens는 골든 디렉토리에 있지만 산출물에 없는 파일 목록이다 (골든 디렉토리 기준
// 슬래시 경로, 정렬순).
func staleGoldens(t testing.TB, goldenDir string, files map[string][]byte) []string {
	t.Helper()
	var stale []string
	err := filepath.Walk(goldenDir, func(path string, info os.FileInfo, err error) error {
		switch {
		case os.IsNotExist(err):
			return nil
		case err != nil:
			return err
		case info.IsDir():
			return nil
		}
		rel, err := filepath.Rel(goldenDir, path)
		if err != nil {
			return err
		}
		if name := filepath.ToSlash(rel); files[name] == nil {
			stale = append(stale, name)
		}
		return nil
	})
	if err != nil && !os.IsNotExist(err) {
		t.Fatalf("cannot walk golden directory: %v", err)
	}
	sort.Strings(stale)
	return stale
}

// writeGolden은 골든 파일을 쓴다. 내용이 같으면 파일을 건드리지 않는다 — 갱신 실행이
// 만드는 diff가 실제로 움직인 골든만 담게 한다.
func writeGolden(t testing.TB, goldenPath string, data []byte) {
	t.Helper()
	if old, err := os.ReadFile(goldenPath); err == nil && bytes.Equal(old, data) {
		return
	}
	if err := os.MkdirAll(filepath.Dir(goldenPath), 0o755); err != nil {
		t.Fatalf("cannot create golden directory: %v", err)
	}
	if err := os.WriteFile(goldenPath, data, 0o644); err != nil {
		t.Fatalf("cannot write golden: %v", err)
	}
	t.Logf("updated golden: %s", goldenPath)
}

// diffReport는 사람이 읽을 차이 요약이다. 첫 번째로 다른 줄과 그 앞뒤를 보여 준다 —
// 전체 diff는 골든 파일 자체의 git diff가 더 잘 보여 주므로 여기서는 "어디서부터
// 갈라졌나"만 짚는다.
func diffReport(want, got []byte) string {
	if !isText(want) || !isText(got) {
		return fmt.Sprintf("  binary: want %d bytes, got %d bytes (first difference at offset %d)",
			len(want), len(got), firstDiffOffset(want, got))
	}
	wantLines := strings.Split(string(want), "\n")
	gotLines := strings.Split(string(got), "\n")

	var b strings.Builder
	fmt.Fprintf(&b, "  want %d lines, got %d lines\n", len(wantLines), len(gotLines))
	for i := 0; i < len(wantLines) || i < len(gotLines); i++ {
		w, g := lineAt(wantLines, i), lineAt(gotLines, i)
		if w == g {
			continue
		}
		for j := max(0, i-2); j < i; j++ {
			fmt.Fprintf(&b, "   %4d | %s\n", j+1, lineAt(gotLines, j))
		}
		fmt.Fprintf(&b, "  -%4d | %s\n", i+1, w)
		fmt.Fprintf(&b, "  +%4d | %s\n", i+1, g)
		break
	}
	return strings.TrimRight(b.String(), "\n")
}

// lineAt은 범위를 벗어나면 없음 표시를 돌려준다.
func lineAt(lines []string, i int) string {
	if i < 0 || i >= len(lines) {
		return "<none>"
	}
	return lines[i]
}

// firstDiffOffset은 처음으로 다른 바이트 위치다.
func firstDiffOffset(a, b []byte) int {
	n := min(len(a), len(b))
	for i := 0; i < n; i++ {
		if a[i] != b[i] {
			return i
		}
	}
	return n
}

// isText는 줄 단위 diff를 보여도 되는 내용인지 본다.
func isText(data []byte) bool {
	return utf8.Valid(data) && !bytes.ContainsRune(data, 0)
}
