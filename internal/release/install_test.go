package release

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
)

// 이 파일은 저장소 루트의 install.sh — `curl … | sh`로 사람이 받는 설치기 — 를 판정한다.
// 아카이브와 SHA256SUMS는 `make dist`와 **같은 코드**(Build)가 호스트 타깃으로 만들고,
// httptest가 GitHub Releases와 같은 경로(`download/<태그>/…`, `latest/download/…`)로 내준다.
// 그래서 설치기가 읽는 이름 규칙과 체크섬 형식이 릴리스 코드와 어긋나면 여기서 깨진다.

const installTestVersion = "v9.9.9-install"

var installScript = filepath.Join("..", "..", "install.sh")

var (
	installReleaseOnce sync.Once
	installRelease     Result
	installReleaseDir  string
	installReleaseErr  error
)

// hostRelease는 호스트 타깃 하나의 릴리스를 한 번만 만든다 — 크로스 컴파일은 비싸고, 세 판정이
// 같은 아카이브를 본다.
func hostRelease(t *testing.T) (Result, string) {
	t.Helper()
	if testing.Short() {
		t.Skip("실제 빌드는 -short에서 건너뛴다")
	}
	if runtime.GOOS != "linux" && runtime.GOOS != "darwin" {
		t.Skipf("설치기는 linux·darwin만 다룬다 (호스트 %s)", runtime.GOOS)
	}
	if _, err := exec.LookPath("sh"); err != nil {
		t.Skip("sh가 없다")
	}
	if _, errCurl := exec.LookPath("curl"); errCurl != nil {
		if _, errWget := exec.LookPath("wget"); errWget != nil {
			t.Skip("curl도 wget도 없다")
		}
	}
	installReleaseOnce.Do(func() {
		dir, err := os.MkdirTemp("", "balpyo-install-release-")
		if err != nil {
			installReleaseErr = err
			return
		}
		installReleaseDir = dir
		installRelease, installReleaseErr = Build(context.Background(), Options{
			ModuleDir: filepath.Join("..", ".."),
			OutDir:    dir,
			Version:   installTestVersion,
			Repo:      "k2sebeom/balpyo",
			Targets:   []Target{{runtime.GOOS, runtime.GOARCH}},
		})
	})
	if installReleaseErr != nil {
		t.Fatalf("Build: %v", installReleaseErr)
	}
	return installRelease, installReleaseDir
}

func TestMain(m *testing.M) {
	code := m.Run()
	if installReleaseDir != "" {
		os.RemoveAll(installReleaseDir)
	}
	os.Exit(code)
}

// releaseServer는 GitHub Releases의 두 경로를 흉내 낸다. checksums는 SHA256SUMS로 내줄 내용이다
// (변조 판정이 바꿔 끼운다).
func releaseServer(t *testing.T, result Result, dir string, checksums []byte) *httptest.Server {
	t.Helper()
	archive := result.Artifacts[0].Name
	archiveBytes, err := os.ReadFile(filepath.Join(dir, archive))
	if err != nil {
		t.Fatal(err)
	}
	files := map[string][]byte{
		"/releases/download/" + installTestVersion + "/" + ChecksumsName: checksums,
		"/releases/download/" + installTestVersion + "/" + archive:       archiveBytes,
		"/releases/latest/download/" + ChecksumsName:                     checksums,
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data, ok := files[r.URL.Path]
		if !ok {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write(data)
	}))
	t.Cleanup(server.Close)
	return server
}

// runInstaller는 `curl … | sh`처럼 스크립트를 표준 입력으로 sh에 흘려 넣는다.
func runInstaller(t *testing.T, serverURL, installDir string, extraEnv ...string) (string, error) {
	t.Helper()
	script, err := os.ReadFile(installScript)
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("sh")
	cmd.Stdin = strings.NewReader(string(script))
	cmd.Env = append(os.Environ(),
		"BALPYO_DOWNLOAD_URL="+serverURL+"/releases",
		"BALPYO_INSTALL_DIR="+installDir,
		"BALPYO_VERSION=",
	)
	cmd.Env = append(cmd.Env, extraEnv...)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func assertInstalled(t *testing.T, installDir, output string) {
	t.Helper()
	exe := filepath.Join(installDir, "balpyo")
	out, err := exec.Command(exe, "version").CombinedOutput()
	if err != nil {
		t.Fatalf("설치된 balpyo가 돌지 않는다: %v\n%s\n설치기 출력:\n%s", err, out, output)
	}
	want := "balpyo " + installTestVersion
	if first := strings.SplitN(string(out), "\n", 2)[0]; first != want {
		t.Errorf("설치된 balpyo version 첫 줄 = %q, want %q", first, want)
	}
	// 설치기는 끝에서 설치한 바이너리의 version을 찍는다 — 무엇이 깔렸는지 사람이 바로 본다.
	if !strings.Contains(output, want) {
		t.Errorf("설치기 출력에 %q가 없다:\n%s", want, output)
	}
	if !strings.Contains(output, "checksum verified") {
		t.Errorf("설치기 출력에 체크섬 확인이 없다:\n%s", output)
	}
}

// 태그를 고른 설치: 그 태그의 아카이브를 받아 맞추고 설치 디렉토리에 놓는다. 설치 디렉토리가
// PATH에 없으면 경고만 하고 성공한다.
func TestInstallScriptInstallsAPinnedVersion(t *testing.T) {
	result, dir := hostRelease(t)
	sums, err := os.ReadFile(result.ChecksumsPath)
	if err != nil {
		t.Fatal(err)
	}
	server := releaseServer(t, result, dir, sums)
	installDir := filepath.Join(t.TempDir(), "nested", "bin")

	output, err := runInstaller(t, server.URL, installDir, "BALPYO_VERSION="+installTestVersion)
	if err != nil {
		t.Fatalf("설치기가 실패했다: %v\n%s", err, output)
	}
	assertInstalled(t, installDir, output)
	if !strings.Contains(output, "is not on your PATH") {
		t.Errorf("PATH에 없는 설치 디렉토리를 경고하지 않았다:\n%s", output)
	}
}

// 태그를 고르지 않으면 최신 릴리스의 SHA256SUMS에 적힌 아카이브 이름에서 버전을 읽는다.
func TestInstallScriptResolvesTheLatestRelease(t *testing.T) {
	result, dir := hostRelease(t)
	sums, err := os.ReadFile(result.ChecksumsPath)
	if err != nil {
		t.Fatal(err)
	}
	server := releaseServer(t, result, dir, sums)
	installDir := t.TempDir()

	output, err := runInstaller(t, server.URL, installDir, "PATH="+installDir+":"+os.Getenv("PATH"))
	if err != nil {
		t.Fatalf("설치기가 실패했다: %v\n%s", err, output)
	}
	assertInstalled(t, installDir, output)
	if strings.Contains(output, "is not on your PATH") {
		t.Errorf("PATH에 있는 설치 디렉토리를 경고했다:\n%s", output)
	}
}

// 체크섬이 맞지 않으면 멈추고 **아무것도 놓지 않는다** — 받은 바이트를 믿을 근거가 체크섬뿐이다.
func TestInstallScriptRefusesATamperedChecksum(t *testing.T) {
	result, dir := hostRelease(t)
	tampered := strings.Repeat("0", 64) + "  " + result.Artifacts[0].Name + "\n"
	server := releaseServer(t, result, dir, []byte(tampered))
	installDir := filepath.Join(t.TempDir(), "bin")

	output, err := runInstaller(t, server.URL, installDir, "BALPYO_VERSION="+installTestVersion)
	if err == nil {
		t.Fatalf("변조된 체크섬으로 설치기가 성공했다:\n%s", output)
	}
	if !strings.Contains(output, "checksum mismatch") {
		t.Errorf("실패 사유가 체크섬 불일치가 아니다:\n%s", output)
	}
	if _, err := os.Stat(filepath.Join(installDir, "balpyo")); !os.IsNotExist(err) {
		t.Errorf("체크섬이 틀렸는데 바이너리가 놓였다 (err=%v)", err)
	}
}

// 없는 태그는 체크섬 파일에서 멈춘다 — 무엇이 없는지 말한다.
func TestInstallScriptReportsAMissingRelease(t *testing.T) {
	result, dir := hostRelease(t)
	sums, err := os.ReadFile(result.ChecksumsPath)
	if err != nil {
		t.Fatal(err)
	}
	server := releaseServer(t, result, dir, sums)
	installDir := filepath.Join(t.TempDir(), "bin")

	output, err := runInstaller(t, server.URL, installDir, "BALPYO_VERSION=v0.0.0-missing")
	if err == nil {
		t.Fatalf("없는 태그로 설치기가 성공했다:\n%s", output)
	}
	if !strings.Contains(output, "v0.0.0-missing") {
		t.Errorf("실패 메시지가 태그를 말하지 않는다:\n%s", output)
	}
	if _, err := os.Stat(filepath.Join(installDir, "balpyo")); !os.IsNotExist(err) {
		t.Errorf("없는 릴리스인데 바이너리가 놓였다 (err=%v)", err)
	}
}
