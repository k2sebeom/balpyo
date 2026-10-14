// Package preview는 산출물을 사람이 볼 수 있는 그림으로 되돌린다 — PPTX를 슬라이드별 PNG로
// 래스터화하고(OnlyOffice `x2t`, 없으면 `soffice` → PDF → `pdftoppm`), 웹 스크린샷과 나란히 놓은
// 갤러리 하나를 쓴다.
//
// 왜 `internal/testkit`이 아닌가: 같은 코드가 이미 그 안에 있었지만 진입점마다
// `testing.TB`를 받아 **테스트에 묶여** 있었다. 저자가 PPTX 한 장을 눈으로 보려면 임시
// 테스트 파일을 쓰는 우회를 해야 했다. CLI가 하네스를
// 임포트하는 쪽은 답이 아니다 — 배포 바이너리에 `testing`이 들어오고, "도구가 없으면
// 건너뛴다"는 테스트 정책이 사용자 명령의 동작이 된다.
//
// 그래서 방향을 뒤집었다: 검수 엔진은 이 패키지가 소유하고, `internal/testkit`이 그것을
// `testing.TB`에 맞춰 감싼다. 구현은 한 벌이므로 테스트와
// `balpyo preview`는 같은 코드를 지난다.
package preview

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/k2sebeom/balpyo/render/geom"
)

// 도구 경로를 지정하는 환경 변수. PATH에 없는 설치를 가리킬 때 쓴다.
const (
	// EnvSoffice는 LibreOffice 실행 파일 경로다.
	EnvSoffice = "BALPYO_SOFFICE"
	// EnvPdftoppm은 poppler의 pdftoppm 실행 파일 경로다.
	EnvPdftoppm = "BALPYO_PDFTOPPM"
)

// 탐색 순서: 환경 변수 → PATH → 알려진 설치 경로. macOS 경로를 함께 두는 이유는 앱 번들로
// 설치한 LibreOffice가 PATH에 없기 때문이다.
var (
	// SofficeCandidates는 LibreOffice 실행 파일 후보다.
	SofficeCandidates = append([]string{
		"soffice",
		"libreoffice",
		"/Applications/LibreOffice.app/Contents/MacOS/soffice",
	}, sofficeWindowsCandidates(runtime.GOOS, os.Getenv)...)
	// PdftoppmCandidates는 pdftoppm 실행 파일 후보다.
	PdftoppmCandidates = []string{
		"pdftoppm",
		"/opt/homebrew/bin/pdftoppm",
		"/usr/local/bin/pdftoppm",
	}
)

// sofficeWindowsCandidates는 윈도우 설치 프로그램이 LibreOffice를 두는 자리다 — PATH에 넣지
// 않으므로 PATH만 보면 못 찾는다.
func sofficeWindowsCandidates(goos string, getenv func(string) string) []string {
	if goos != "windows" {
		return nil
	}
	return windowsInstalls(getenv, "ProgramFiles", `LibreOffice\program\soffice.exe`)
}

// windowsInstalls는 설치 폴더 환경 변수(ProgramFiles 등) 아래의 경로들이다. 값이 빈 변수는
// 건너뛴다 — 빈 값으로 만든 `\…`는 현재 드라이브의 엉뚱한 경로가 된다.
func windowsInstalls(getenv func(string) string, env, rel string) []string {
	base := getenv(env)
	if base == "" {
		return nil
	}
	return []string{strings.TrimRight(base, `\`) + `\` + rel}
}

// exeName은 도구 이름을 이 운영체제의 실행 파일 이름으로 만든다 — 윈도우는 `.exe`를 붙인다.
// 설치 디렉토리 안의 파일을 경로로 부를 때는 PATH 탐색처럼 확장자를 대신 붙여 주는 쪽이 없다.
func exeName(name string) string {
	if runtime.GOOS == "windows" {
		return name + ".exe"
	}
	return name
}

// LookSoffice는 LibreOffice 실행 파일을 찾는다.
func LookSoffice() (string, error) { return LookTool(EnvSoffice, SofficeCandidates) }

// LookPdftoppm은 pdftoppm 실행 파일을 찾는다.
func LookPdftoppm() (string, error) { return LookTool(EnvPdftoppm, PdftoppmCandidates) }

// LookTool은 환경 변수 → PATH → 후보 경로 순으로 실행 파일을 찾는다.
//
// 브라우저 탐색(render/geom)도 같은 순서를 쓰지만 목록은 그쪽이 소유한다 — 후보 목록이 두
// 벌이면 이미지에 있는 브라우저를 한쪽만 찾는 상태가 생긴다.
func LookTool(env string, candidates []string) (string, error) {
	if custom := os.Getenv(env); custom != "" {
		if err := executable(custom); err != nil {
			return "", fmt.Errorf("%s=%s: %w", env, custom, err)
		}
		return custom, nil
	}
	for _, candidate := range candidates {
		if strings.ContainsRune(candidate, filepath.Separator) {
			if err := executable(candidate); err == nil {
				return candidate, nil
			}
			continue
		}
		if path, err := exec.LookPath(candidate); err == nil {
			return path, nil
		}
	}
	return "", fmt.Errorf("cannot find executable (candidates: %s)", strings.Join(candidates, ", "))
}

// executable은 경로가 실행 가능한 파일인지 본다. 판정은 브라우저 탐색(render/geom)과 같은
// 하나다 — 윈도우에는 실행 권한 비트가 없어서 운영체제마다 다르게 봐야 한다.
func executable(path string) error { return geom.Executable(path) }
