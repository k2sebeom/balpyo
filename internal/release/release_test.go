package release

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// 이 파일은 **릴리스 절차 자체**를 판정한다: 플랫폼 표·아카이브 이름·
// 체크섬 파일·Homebrew 포뮬러·주입 플래그가 쉘 스크립트가 아니라 이 패키지에 있으므로,
// 타깃 하나를 지우거나 이름 규칙을 바꾸는 커밋을 테스트가 본다.

// 플랫폼 표는 **재본 넷**이다. 윈도우가 없는 것은 결정이므로 판정 대상이다 — 누군가 표에
// 한 줄 더하려면 이 테스트를 먼저 고쳐야 하고, 그 자리에서 "재봤는가"를 묻게 된다.
func TestTargetsAreTheFourMeasuredPlatforms(t *testing.T) {
	var got []string
	for _, target := range Targets() {
		got = append(got, target.String())
	}
	want := []string{"darwin/arm64", "darwin/amd64", "linux/amd64", "linux/arm64"}
	if strings.Join(got, " ") != strings.Join(want, " ") {
		t.Errorf("Targets() = %v, want %v", got, want)
	}
	for _, target := range Targets() {
		if target.OS == "windows" {
			t.Errorf("표에 windows가 있다 — 측정하지 않은 플랫폼의 아카이브는 놓지 않는다")
		}
	}
}

// 아카이브 이름은 사람이 URL에서 읽는 줄이다 — 버전과 플랫폼이 이름 안에 있다.
func TestArchiveNameCarriesVersionAndPlatform(t *testing.T) {
	cases := []struct {
		version string
		target  Target
		want    string
	}{
		{"v0.3.0", Target{"darwin", "arm64"}, "balpyo_v0.3.0_darwin_arm64.tar.gz"},
		{"v0.3.0", Target{"linux", "amd64"}, "balpyo_v0.3.0_linux_amd64.tar.gz"},
		{"v1.0.0-rc.1", Target{"linux", "arm64"}, "balpyo_v1.0.0-rc.1_linux_arm64.tar.gz"},
	}
	for _, tc := range cases {
		if got := ArchiveName(tc.version, tc.target); got != tc.want {
			t.Errorf("ArchiveName(%q, %s) = %q, want %q", tc.version, tc.target, got, tc.want)
		}
	}
}

// 주입은 **태그 하나**다. 커밋·시각을 여기에 넣으면 같은 사실의 두 번째
// 출처가 생기므로, `-X`가 하나인 것 자체를 판정한다.
func TestLDFlagsInjectOnlyTheVersion(t *testing.T) {
	flags := LDFlags("v0.3.0")
	if want := "-X main.version=v0.3.0"; !strings.Contains(flags, want) {
		t.Errorf("LDFlags = %q, want %q 포함", flags, want)
	}
	if n := strings.Count(flags, "-X "); n != 1 {
		t.Errorf("LDFlags = %q — -X가 %d개다 (태그 하나만 주입한다)", flags, n)
	}
	for _, forbidden := range []string{"commit", "date", "time", "builtBy"} {
		if strings.Contains(flags, forbidden) {
			t.Errorf("LDFlags = %q — %q는 툴체인의 vcs.*가 소유한다", flags, forbidden)
		}
	}
}

// 체크섬 파일은 `sha256sum -c`가 읽는 형식이다 (해시, 공백 둘, 이름).
func TestChecksumsAreShaSumFormat(t *testing.T) {
	got := Checksums([]Artifact{
		{Name: "balpyo_v0.3.0_linux_amd64.tar.gz", SHA256: "aa", Target: Target{"linux", "amd64"}},
		{Name: "balpyo_v0.3.0_darwin_arm64.tar.gz", SHA256: "bb", Target: Target{"darwin", "arm64"}},
	})
	want := "aa  balpyo_v0.3.0_linux_amd64.tar.gz\nbb  balpyo_v0.3.0_darwin_arm64.tar.gz\n"
	if got != want {
		t.Errorf("Checksums =\n%q\nwant\n%q", got, want)
	}
}

// 포뮬러는 **체크섬에서 나온다** — 네 타깃이 OS × 아키텍처 네 칸에 각자의 해시와 함께 들어간다.
func TestFormulaCoversEveryTargetWithItsChecksum(t *testing.T) {
	var artifacts []Artifact
	hashes := map[string]string{
		"darwin/arm64": "1111111111111111111111111111111111111111111111111111111111111111",
		"darwin/amd64": "2222222222222222222222222222222222222222222222222222222222222222",
		"linux/amd64":  "3333333333333333333333333333333333333333333333333333333333333333",
		"linux/arm64":  "4444444444444444444444444444444444444444444444444444444444444444",
	}
	for _, target := range Targets() {
		artifacts = append(artifacts, Artifact{
			Name:   ArchiveName("v0.3.0", target),
			SHA256: hashes[target.String()],
			Target: target,
		})
	}

	formula, err := Formula(FormulaSpec{Version: "v0.3.0", Repo: "k2sebeom/balpyo", Artifacts: artifacts})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"class Balpyo < Formula",
		`url "https://github.com/k2sebeom/balpyo/releases/download/v0.3.0/balpyo_v0.3.0_darwin_arm64.tar.gz"`,
		"on_macos", "on_linux", "on_arm", "on_intel",
		`bin.install "balpyo"`,
		`balpyo v#{version}`, // test 스탠자가 설치된 바이너리를 실제로 돌려 버전을 맞춘다
	} {
		if !strings.Contains(formula, want) {
			t.Errorf("포뮬러에 %q가 없다:\n%s", want, formula)
		}
	}
	for platform, hash := range hashes {
		if !strings.Contains(formula, hash) {
			t.Errorf("포뮬러에 %s의 체크섬이 없다:\n%s", platform, formula)
		}
	}
	// 버전은 `v` 없이 Homebrew의 것이다 (`version "0.3.0"`) — tap의 관례다.
	if !strings.Contains(formula, `version "0.3.0"`) {
		t.Errorf("포뮬러의 version 스탠자가 없다(또는 v가 붙었다):\n%s", formula)
	}
}

// 타깃 하나의 아티팩트가 없으면 포뮬러를 내지 않는다 — 반쪽 포뮬러는 받는 사람의 기계에서만
// 실패한다 (그리고 그 실패는 우리 로그에 없다).
func TestFormulaRefusesAMissingTarget(t *testing.T) {
	_, err := Formula(FormulaSpec{
		Version: "v0.3.0", Repo: "k2sebeom/balpyo",
		Artifacts: []Artifact{{Name: "balpyo_v0.3.0_linux_amd64.tar.gz", SHA256: "aa", Target: Target{"linux", "amd64"}}},
	})
	if err == nil {
		t.Fatal("타깃 셋이 빠진 포뮬러가 통과했다")
	}
	if !strings.Contains(err.Error(), "darwin/arm64") {
		t.Errorf("에러 = %v — 어느 타깃이 없는지 말해야 한다", err)
	}
}

// 그리고 실제 빌드 — 호스트 타깃 하나를 끝까지 만든다: 아카이브 · 체크섬 · 포뮬러가 디스크에
// 놓이고, 아카이브에서 꺼낸 바이너리가 **주입한 태그를 말한다**. (커밋 줄을 보는 판정은 여기에
// 있다 — `go test -c` 바이너리에는 vcs.*가 없으므로 cmd/balpyo의 테스트는 볼 수 없다.)
func TestBuildProducesARunnableArchive(t *testing.T) {
	if testing.Short() {
		t.Skip("실제 크로스 컴파일은 -short에서 건너뛴다")
	}
	host := Target{runtime.GOOS, runtime.GOARCH}
	dir := t.TempDir()

	result, err := Build(context.Background(), Options{
		ModuleDir: filepath.Join("..", ".."),
		OutDir:    dir,
		Version:   "v9.9.9-test",
		Repo:      "k2sebeom/balpyo",
		Targets:   []Target{host},
	})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if len(result.Artifacts) != 1 {
		t.Fatalf("아티팩트 %d개, want 1개", len(result.Artifacts))
	}
	artifact := result.Artifacts[0]
	if want := ArchiveName("v9.9.9-test", host); artifact.Name != want {
		t.Errorf("아티팩트 이름 = %q, want %q", artifact.Name, want)
	}

	// 체크섬 파일과 포뮬러는 파일로 있고, 체크섬은 아카이브의 실제 해시다.
	sums, err := os.ReadFile(result.ChecksumsPath)
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(result.ChecksumsPath) != ChecksumsName {
		t.Errorf("체크섬 파일 = %q, want %q", result.ChecksumsPath, ChecksumsName)
	}
	if got, want := string(sums), artifact.SHA256+"  "+artifact.Name+"\n"; got != want {
		t.Errorf("SHA256SUMS = %q, want %q", got, want)
	}
	if got := fileSHA256(t, filepath.Join(dir, artifact.Name)); got != artifact.SHA256 {
		t.Errorf("아카이브의 실제 해시 %s ≠ 보고한 해시 %s", got, artifact.SHA256)
	}
	// 타깃 하나만 만든 빌드는 **포뮬러를 내지 않는다**. 처음에는 여기서 포뮬러 파일을 요구했는데,
	// 실물을 보고 그 요구를 버렸다: 한 타깃으로 만든 .rb는 나머지 세 칸이 빈 url·빈 sha256이었다
	// (`make dist TARGETS=linux/amd64`). 그런 파일은 tap에 올라간 뒤 받는 사람의 기계에서만
	// 실패하므로, 없는 것이 맞다.
	if result.FormulaPath != "" {
		t.Errorf("타깃 하나로 포뮬러를 냈다: %s", result.FormulaPath)
	}
	if _, err := os.Stat(filepath.Join(dir, FormulaName)); !os.IsNotExist(err) {
		t.Errorf("%s가 디스크에 있다 (반쪽 포뮬러) — err=%v", FormulaName, err)
	}

	// 아카이브에는 바이너리 **하나**만 있다.
	names, binary := extractTarGz(t, filepath.Join(dir, artifact.Name))
	if len(names) != 1 || names[0] != "balpyo" {
		t.Fatalf("아카이브 내용 = %v, want [balpyo]", names)
	}

	// 그리고 그 바이너리가 주입한 태그를 말한다.
	exe := filepath.Join(t.TempDir(), "balpyo")
	if err := os.WriteFile(exe, binary, 0o755); err != nil {
		t.Fatal(err)
	}
	out, err := exec.Command(exe, "version").CombinedOutput()
	if err != nil {
		t.Fatalf("balpyo version: %v\n%s", err, out)
	}
	lines := strings.Split(strings.TrimRight(string(out), "\n"), "\n")
	if want := "balpyo v9.9.9-test"; lines[0] != want {
		t.Errorf("첫 줄 = %q, want %q", lines[0], want)
	}
	// 릴리스 바이너리는 커밋을 안다 — 툴체인이 박고 -trimpath가 지우지 않는다.
	if len(lines) != 3 || !strings.HasPrefix(lines[1], "커밋 ") {
		t.Errorf("릴리스 바이너리의 버전 출력이 세 줄(커밋 포함)이 아니다:\n%s", out)
	}
	if want := host.String() + " · " + runtime.Version(); lines[len(lines)-1] != want {
		t.Errorf("마지막 줄 = %q, want %q", lines[len(lines)-1], want)
	}
}

// 재현성은 계약이다: 같은 버전으로 두 번 만든 아카이브는 **바이트까지 같다**.
// tar 헤더의 시각·소유자까지 고정해야 통과한다.
func TestBuildIsReproducible(t *testing.T) {
	if testing.Short() {
		t.Skip("실제 크로스 컴파일은 -short에서 건너뛴다")
	}
	host := Target{runtime.GOOS, runtime.GOARCH}
	var hashes []string
	for range 2 {
		dir := t.TempDir()
		result, err := Build(context.Background(), Options{
			ModuleDir: filepath.Join("..", ".."),
			OutDir:    dir,
			Version:   "v9.9.9-test",
			Repo:      "k2sebeom/balpyo",
			Targets:   []Target{host},
		})
		if err != nil {
			t.Fatalf("Build: %v", err)
		}
		hashes = append(hashes, fileSHA256(t, filepath.Join(dir, result.Artifacts[0].Name)))
	}
	if hashes[0] != hashes[1] {
		t.Errorf("두 번 빌드한 아카이브가 다르다:\n%s\n%s", hashes[0], hashes[1])
	}
}

func fileSHA256(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// extractTarGz는 이름 목록과 첫 파일의 내용을 돌려준다 — tar 헤더가 결정적인지(시각·소유자)도
// 여기서 함께 본다.
func extractTarGz(t *testing.T, path string) ([]string, []byte) {
	t.Helper()
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	gz, err := gzip.NewReader(file)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	var first []byte
	reader := tar.NewReader(gz)
	for {
		header, err := reader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		names = append(names, header.Name)
		// tar는 시각을 epoch 오프셋으로 적으므로 "없음"을 적을 자리가 없다 — 0(1970-01-01)이
		// 우리가 쓸 수 있는 "빌드한 시각을 담지 않았다"다 (읽어 오면 IsZero가 아니라 Unix()==0이다).
		if header.ModTime.Unix() != 0 {
			t.Errorf("tar 헤더의 시각이 %v다 — 재현 가능한 아카이브는 epoch 0을 쓴다", header.ModTime)
		}
		if header.Uid != 0 || header.Gid != 0 || header.Uname != "" || header.Gname != "" {
			t.Errorf("tar 헤더에 빌드한 사람이 남아 있다: uid=%d gid=%d uname=%q gname=%q",
				header.Uid, header.Gid, header.Uname, header.Gname)
		}
		if header.Mode != 0o755 {
			t.Errorf("tar 헤더의 모드 = %o, want 755 (받은 사람이 바로 실행한다)", header.Mode)
		}
		if first == nil {
			data, err := io.ReadAll(reader)
			if err != nil {
				t.Fatal(err)
			}
			first = data
		}
	}
	return names, first
}
