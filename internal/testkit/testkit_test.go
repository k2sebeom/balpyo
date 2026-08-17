package testkit

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRepoRootHasGoMod(t *testing.T) {
	root := RepoRoot(t)
	if _, err := os.Stat(filepath.Join(root, "go.mod")); err != nil {
		t.Fatalf("저장소 루트가 아니다: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(FixturesDir))); err != nil {
		t.Errorf("픽스처 루트가 없다: %v", err)
	}
}

func TestLookToolPrefersEnv(t *testing.T) {
	dir := t.TempDir()
	fake := filepath.Join(dir, "fake-tool")
	if err := os.WriteFile(fake, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	t.Setenv("BALPYO_FAKE", fake)
	got, err := lookTool("BALPYO_FAKE", []string{"sh"})
	if err != nil {
		t.Fatal(err)
	}
	if got != fake {
		t.Errorf("환경 변수보다 PATH를 먼저 봤다: %q", got)
	}

	// 환경 변수가 가리키는 경로가 실행 파일이 아니면 조용히 PATH로 흘러가지 않는다 —
	// 사람이 지정한 경로가 틀렸다는 사실이 드러나야 한다.
	t.Setenv("BALPYO_FAKE", filepath.Join(dir, "없다"))
	if _, err := lookTool("BALPYO_FAKE", []string{"sh"}); err == nil {
		t.Error("잘못된 환경 변수 경로는 에러여야 한다")
	}

	// 실행 권한이 없는 파일도 도구가 아니다.
	plain := filepath.Join(dir, "plain")
	if err := os.WriteFile(plain, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := executable(plain); err == nil {
		t.Error("실행 권한이 없는 파일이 통과했다")
	}
}

func TestLookToolFallsBackToPath(t *testing.T) {
	t.Setenv("BALPYO_FAKE", "")
	got, err := lookTool("BALPYO_FAKE", []string{"이런-도구는-없다", "sh"})
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(got) != "sh" {
		t.Errorf("PATH에서 찾은 도구 = %q, want .../sh", got)
	}
	if _, err := lookTool("BALPYO_FAKE", []string{"이런-도구는-없다"}); err == nil {
		t.Error("없는 도구는 에러여야 한다")
	}
}

func TestSofficeNoiseKeepsOnlyRealWarnings(t *testing.T) {
	stderr := strings.Join([]string{
		"javaldx: Could not find a Java Runtime Environment!",
		"Warning: failed to launch javaldx - java may not function correctly",
		"",
		"Fontconfig warning: ignoring UTF-8: not a valid region tag",
	}, "\n")
	if noise := sofficeNoise(stderr); noise != "" {
		t.Errorf("무해한 잡음을 경고로 봤다: %q", noise)
	}

	broken := "Error: source file could not be loaded\nrepairing document"
	noise := sofficeNoise(broken)
	if !strings.Contains(noise, "Error: source file could not be loaded") ||
		!strings.Contains(noise, "repairing document") {
		t.Errorf("실제 경고를 놓쳤다: %q", noise)
	}
}

func TestReplaceExt(t *testing.T) {
	if got := replaceExt("my-talk.pptx", ".pdf"); got != "my-talk.pdf" {
		t.Errorf("got %q", got)
	}
	if got := replaceExt("2026.리뷰.pptx", ".pdf"); got != "2026.리뷰.pdf" {
		t.Errorf("점이 여러 개인 이름에서 마지막 확장자만 바뀌어야 한다: %q", got)
	}
}
