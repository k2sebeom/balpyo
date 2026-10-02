package testkit

import (
	"strings"
	"testing"
)

// TestFixtures는 픽스처 스위트다 — 모든 반복(iteration)이 여기에 픽스처를 더한다.
//
// 픽스처마다: 빌드 → 골든 비교(layout.txt · diagnostics.txt) → 동일성 판정 → 패키지 불변식 →
// 렌더 비교. `make review`는 같은 테스트를 리뷰 모드로 돌려 갤러리를 남긴다.
func TestFixtures(t *testing.T) {
	NewSuite(t).Run(t)
}

func TestFixturesAreDiscovered(t *testing.T) {
	fixtures := Fixtures(t)
	for _, f := range fixtures {
		if strings.HasPrefix(f.Name, "_") {
			t.Errorf("작업 중 픽스처가 수집되었다: %s", f.Name)
		}
		if !strings.HasPrefix(f.Dir, RepoRoot(t)) {
			t.Errorf("픽스처 경로가 저장소 밖이다: %s", f.Dir)
		}
	}
	// 접두 숫자가 실행 순서다 — 정렬이 깨지면 리뷰 갤러리 순서가 iteration 순서와 어긋난다.
	for i := 1; i < len(fixtures); i++ {
		if fixtures[i-1].Name > fixtures[i].Name {
			t.Errorf("픽스처 순서가 정렬되지 않았다: %s > %s", fixtures[i-1].Name, fixtures[i].Name)
		}
	}
}

// 픽스처 빌드는 픽스처 디렉토리를 건드리지 않는다 — 산출물이 입력 옆에 쌓이면 다음 실행의
// 입력이 되고, 골든만 커밋한다는 규칙이 무너진다.
func TestFixtureBuildLeavesFixtureDirClean(t *testing.T) {
	fixture := Fixtures(t)[0]
	result := fixture.Build(t)
	if strings.HasPrefix(result.OutDir, fixture.Dir) {
		t.Errorf("산출물이 픽스처 디렉토리 안에 쓰였다: %s", result.OutDir)
	}
	if len(result.Written) == 0 {
		t.Error("산출물이 없다")
	}
}

// 골든은 두 종뿐이다 — 구조 스냅샷과 (경고가 있을 때) 진단. 산출물 스냅샷이 다시 끼어들면
// 환경 고정이 다시 필요해진다 (ADR 0064).
func TestFixtureGoldenTreeShape(t *testing.T) {
	fixture := Fixtures(t)[0]
	files := fixture.goldenFiles(t, fixture.Build(t))
	if len(files[layoutGolden]) == 0 {
		t.Error("구조 스냅샷이 비었다")
	}
	for name := range files {
		if name != layoutGolden && name != diagnosticsGolden {
			t.Errorf("골든에 산출물 스냅샷이 있다: %s", name)
		}
	}
}
