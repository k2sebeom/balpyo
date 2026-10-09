package testkit

import (
	"os"
	"path/filepath"
	"testing"
)

// 도구 탐색·soffice 잡음 필터·확장자 치환의 테스트는 그 코드와 함께 internal/preview에 있다.
// 여기 남는 것은 하네스가 소유하는 정책뿐이다.

func TestRepoRootHasGoMod(t *testing.T) {
	root := RepoRoot(t)
	if _, err := os.Stat(filepath.Join(root, "go.mod")); err != nil {
		t.Fatalf("저장소 루트가 아니다: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(FixturesDir))); err != nil {
		t.Errorf("픽스처 루트가 없다: %v", err)
	}
}
