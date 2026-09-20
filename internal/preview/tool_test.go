package preview

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLookToolPrefersEnv(t *testing.T) {
	dir := t.TempDir()
	fake := filepath.Join(dir, "fake-tool")
	if err := os.WriteFile(fake, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	t.Setenv("BALPYO_FAKE", fake)
	got, err := LookTool("BALPYO_FAKE", []string{"sh"})
	if err != nil {
		t.Fatal(err)
	}
	if got != fake {
		t.Errorf("환경 변수보다 PATH를 먼저 봤다: %q", got)
	}

	// 환경 변수가 가리키는 경로가 실행 파일이 아니면 조용히 PATH로 흘러가지 않는다 —
	// 사람이 지정한 경로가 틀렸다는 사실이 드러나야 한다.
	t.Setenv("BALPYO_FAKE", filepath.Join(dir, "없다"))
	if _, err := LookTool("BALPYO_FAKE", []string{"sh"}); err == nil {
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
	got, err := LookTool("BALPYO_FAKE", []string{"이런-도구는-없다", "sh"})
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(got) != "sh" {
		t.Errorf("PATH에서 찾은 도구 = %q, want .../sh", got)
	}
	if _, err := LookTool("BALPYO_FAKE", []string{"이런-도구는-없다"}); err == nil {
		t.Error("없는 도구는 에러여야 한다")
	}
}
