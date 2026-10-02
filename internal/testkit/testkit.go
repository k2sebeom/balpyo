// Package testkit은 픽스처 기반 테스트 하네스다 — 골든 비교, 동일성 판정(웹 트리 == PPTX 트리),
// 패키지 불변식 · 스키마 검증, 외부 도구 헬퍼.
//
// **PPTX의 유효성은 ECMA-376 스키마가 판정한다** (ADR 0065) — LibreOffice는 게이트가 아니다.
// LibreOffice · pdftoppm은 `balpyo preview`의 래스터 도구라 그 테스트만 부르고, 없으면 언제나
// 건너뛴다 (사람이 보는 도구다 — ADR 0047).
//
// **도구가 없을 때의 정책**: BALPYO_TEST_STRICT=1(테스트 이미지가 켠다)이면 건너뛰지
// 않고 실패한다. 그래서 로컬 macOS에서 `go test ./...`가 초록이어도 — 거기서는 LibreOffice·
// Chromium 테스트가 조용히 건너뛰어진다 — 이미지 안에서는 반드시 돌고, "건너뛴 줄 몰랐다"가
// 생기지 않는다.
package testkit

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/k2sebeom/balpyo/internal/preview"
	"github.com/k2sebeom/balpyo/render/geom"
)

// 하네스가 읽는 환경 변수. 이미지가 켜는 값은 docker/test.Dockerfile에 있다.
const (
	// EnvStrict가 1이면 외부 도구가 없을 때 건너뛰지 않고 실패한다.
	EnvStrict = "BALPYO_TEST_STRICT"
	// EnvUpdateGolden이 1이면 골든을 비교하는 대신 새로 쓴다 (`make golden`).
	EnvUpdateGolden = "BALPYO_UPDATE_GOLDEN"

	// EnvChromium은 Chromium/Chrome 실행 파일 경로다. 빌드(지오메트리 패스)가 보는 변수와
	// 같아야 한다 — 다르면 "테스트는 되는데 빌드는 브라우저를 못 찾는다"가 된다.
	EnvChromium = geom.EnvChromium
	// EnvChromiumNoSandbox가 1이면 --no-sandbox를 붙인다 (컨테이너 안에서는 root라 필수).
	EnvChromiumNoSandbox = geom.EnvNoSandbox
	// EnvSoffice는 LibreOffice 실행 파일 경로다.
	EnvSoffice = preview.EnvSoffice
	// EnvPdftoppm은 poppler의 pdftoppm 실행 파일 경로다.
	EnvPdftoppm = preview.EnvPdftoppm
)

// Strict는 외부 도구가 없을 때 실패해야 하는지 알려준다.
func Strict() bool { return os.Getenv(EnvStrict) == "1" }

// UpdateGolden은 골든을 갱신하는 실행인지 알려준다.
func UpdateGolden() bool { return os.Getenv(EnvUpdateGolden) == "1" }

// RepoRoot는 저장소 루트다. 픽스처와 골든은 패키지 상대가 아니라 저장소 상대 경로에
// 있으므로(여러 패키지의 테스트가 같은 픽스처를 본다) 루트를 찾아야 한다.
func RepoRoot(t testing.TB) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("작업 디렉토리를 알 수 없다: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("go.mod가 있는 저장소 루트를 찾을 수 없다")
		}
		dir = parent
	}
}

// 도구 탐색은 internal/preview가 소유한다 (환경 변수 → PATH → 알려진 설치 경로). 목록과
// 순서가 두 벌이면 `balpyo preview`가 찾는 도구와 테스트가 찾는 도구가 갈린다 — 브라우저
// 후보를 render/geom과 공유하는 것과 같은 이유다.
//
// 브라우저 후보는 빌드가 소유한다 (render/geom).
var chromiumCandidates = geom.Candidates()

// LookChromium은 Chromium 실행 파일을 찾는다.
func LookChromium() (string, error) { return preview.LookTool(EnvChromium, chromiumCandidates) }

// LookSoffice는 LibreOffice 실행 파일을 찾는다.
func LookSoffice() (string, error) { return preview.LookSoffice() }

// LookPdftoppm은 pdftoppm 실행 파일을 찾는다.
func LookPdftoppm() (string, error) { return preview.LookPdftoppm() }

// RequireChromium은 Chromium이 필요한 테스트의 진입점이다. 없으면 Strict에서 실패,
// 아니면 건너뛴다.
func RequireChromium(t testing.TB) string {
	t.Helper()
	return requireTool(t, "Chromium", EnvChromium, LookChromium)
}

// RequireSoffice는 LibreOffice가 필요한 테스트(preview의 PPTX 래스터)의 진입점이다. 없으면
// Strict에서도 건너뛴다 — 게이트가 아닌 도구다.
func RequireSoffice(t testing.TB) string {
	t.Helper()
	return optionalTool(t, "LibreOffice(soffice)", EnvSoffice, LookSoffice)
}

// RequirePdftoppm은 pdftoppm이 필요한 테스트의 진입점이다 (RequireSoffice와 같은 정책).
func RequirePdftoppm(t testing.TB) string {
	t.Helper()
	return optionalTool(t, "pdftoppm(poppler-utils)", EnvPdftoppm, LookPdftoppm)
}

// ChromiumNoSandbox는 --no-sandbox가 필요한 환경인지 알려준다.
func ChromiumNoSandbox() bool { return geom.NoSandbox() }

// requireTool은 탐색 실패를 Strict 정책에 따라 실패 또는 건너뛰기로 바꾼다.
func requireTool(t testing.TB, name, env string, look func() (string, error)) string {
	t.Helper()
	path, err := look()
	if err == nil {
		return path
	}
	msg := fmt.Sprintf("%s가 필요하다: %v — %s로 경로를 지정하거나 `make test`로 테스트 이미지 안에서 돌린다",
		name, err, env)
	if Strict() {
		t.Fatal(msg)
	}
	t.Skip(msg)
	return ""
}

// optionalTool은 탐색 실패를 언제나 건너뛰기로 바꾼다.
func optionalTool(t testing.TB, name, env string, look func() (string, error)) string {
	t.Helper()
	path, err := look()
	if err != nil {
		t.Skipf("%s가 없어 건너뛴다: %v — %s로 경로를 지정한다", name, err, env)
	}
	return path
}
