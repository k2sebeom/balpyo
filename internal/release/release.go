// Package release는 **릴리스 절차**다 (ADR 0061): 플랫폼 표, 아카이브 이름, 체크섬 파일,
// Homebrew 포뮬러, 그리고 빌드에 주입하는 플래그를 한 자리에서 소유한다.
//
// 왜 쉘 스크립트가 아닌가: 쉘에 든 절차는 테스트가 읽지 않는다. 플랫폼 표가 쉘 루프에 있으면
// 타깃 하나를 지우는 커밋을 아무 게이트도 보지 못하고, 아카이브 이름을 바꾸는 커밋은 사람이
// 받는 URL을 조용히 바꾼다. `make dist`는 이 패키지를 부르는 한 줄이고(internal/release/cmd),
// 테스트는 사람이 받는 것과 **같은 코드**를 판정한다 (ADR 0047의 internal/preview와 같은 자리).
package release

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// binaryName은 아카이브 안에 들어가는 이름이자 설치되는 이름이다 — 바이너리 하나가 모든
// 서브커맨드를 담는다.
const binaryName = "balpyo"

// ChecksumsName은 체크섬 파일 하나의 이름이다. 형식은 `sha256sum -c`가 읽는 그것이다.
const ChecksumsName = "SHA256SUMS"

// FormulaName은 생성되는 Homebrew 포뮬러다. 이것을 tap 저장소에 올리는 일은 사람의 것이다
// (우리 저장소가 아닌 곳에 쓰는 토큰을 CI에 두지 않는다 — ADR 0061 결정 7).
const FormulaName = "balpyo.rb"

// Target은 아카이브 하나가 향하는 플랫폼이다.
type Target struct {
	OS   string
	Arch string
}

func (t Target) String() string { return t.OS + "/" + t.Arch }

// Targets는 아카이브를 놓는 플랫폼 표다 — **재본 넷**이다 (ADR 0061 결정 3).
//
// 윈도우가 없는 근거는 없음이 아니라 **재보지 않았음**이다: 경로 구분자·CRLF·폰트 대체·
// `soffice` 위치를 한 번도 측정하지 않았고, 측정하지 않은 플랫폼의 아카이브를 놓는 것은
// "빌드된다"를 "된다"로 발표하는 일이다. 재고 나서 이 표에 한 줄 더한다.
func Targets() []Target {
	return []Target{
		{"darwin", "arm64"},
		{"darwin", "amd64"},
		{"linux", "amd64"},
		{"linux", "arm64"},
	}
}

// ArchiveName은 사람이 URL에서 읽는 이름이다 — 버전과 플랫폼이 이름 안에 있다.
func ArchiveName(version string, t Target) string {
	return fmt.Sprintf("%s_%s_%s_%s.tar.gz", binaryName, version, t.OS, t.Arch)
}

// LDFlags는 릴리스 빌드에 주입하는 **하나**다: 태그. 커밋·시각·더러움은 툴체인이 `vcs.*`에
// 박으므로 여기서 다시 넣으면 같은 사실의 두 번째 출처가 생긴다 (ADR 0061 결정 4).
func LDFlags(version string) string {
	return "-X main.version=" + version
}

// Artifact는 놓인 아카이브 하나다.
type Artifact struct {
	Name   string
	SHA256 string
	Target Target
}

// Checksums는 `sha256sum -c SHA256SUMS`가 읽는 형식이다 (해시, 공백 둘, 이름).
func Checksums(artifacts []Artifact) string {
	var b strings.Builder
	for _, a := range artifacts {
		fmt.Fprintf(&b, "%s  %s\n", a.SHA256, a.Name)
	}
	return b.String()
}

// FormulaSpec은 포뮬러 하나를 만드는 데 필요한 전부다.
type FormulaSpec struct {
	// Version은 태그다 (`v0.3.0`). 포뮬러의 version 스탠자는 `v`를 뗀다 — tap의 관례다.
	Version string
	// Repo는 `소유자/이름`이다 — 다운로드 URL이 여기서 나온다.
	Repo string
	// Artifacts는 놓인 아카이브들이다 — 플랫폼 표(Targets)를 전부 덮어야 한다.
	Artifacts []Artifact
}

// Formula는 Homebrew tap 포뮬러를 만든다. 덮어야 하는 타깃의 아티팩트가 없으면 **거부한다** —
// 반쪽 포뮬러는 받는 사람의 기계에서만 실패하고, 그 실패는 우리 로그에 없다.
func Formula(spec FormulaSpec) (string, error) {
	byTarget := make(map[Target]Artifact, len(spec.Artifacts))
	for _, a := range spec.Artifacts {
		byTarget[a.Target] = a
	}
	var missing []string
	for _, t := range Targets() {
		if _, ok := byTarget[t]; !ok {
			missing = append(missing, t.String())
		}
	}
	if len(missing) > 0 {
		return "", fmt.Errorf("포뮬러를 만들 수 없다: 아티팩트가 없는 타깃 %s", strings.Join(missing, ", "))
	}

	var b strings.Builder
	// 이 파일은 생성물이다 — 손으로 고치면 다음 릴리스가 덮는다.
	fmt.Fprintf(&b, "# `make dist`가 만든 파일이다 — 손으로 고치지 않는다.\n")
	fmt.Fprintf(&b, "class Balpyo < Formula\n")
	fmt.Fprintf(&b, "  desc \"Render SOML slides to web presentations and editable native PPTX\"\n")
	fmt.Fprintf(&b, "  homepage \"https://github.com/%s\"\n", spec.Repo)
	fmt.Fprintf(&b, "  version \"%s\"\n", strings.TrimPrefix(spec.Version, "v"))
	for _, platform := range []struct{ goos, stanza string }{{"darwin", "on_macos"}, {"linux", "on_linux"}} {
		fmt.Fprintf(&b, "\n  %s do\n", platform.stanza)
		for _, arch := range []struct{ goarch, stanza string }{{"arm64", "on_arm"}, {"amd64", "on_intel"}} {
			artifact := byTarget[Target{platform.goos, arch.goarch}]
			fmt.Fprintf(&b, "    %s do\n", arch.stanza)
			fmt.Fprintf(&b, "      url \"https://github.com/%s/releases/download/%s/%s\"\n",
				spec.Repo, spec.Version, artifact.Name)
			fmt.Fprintf(&b, "      sha256 \"%s\"\n", artifact.SHA256)
			fmt.Fprintf(&b, "    end\n")
		}
		fmt.Fprintf(&b, "  end\n")
	}
	// 설치는 바이너리 하나를 옮기는 일이고, test 스탠자는 그 바이너리를 **실제로 돌려** 버전을
	// 맞춘다 (받은 아카이브가 그 태그의 것인지 tap이 스스로 판정한다).
	fmt.Fprintf(&b, "\n  def install\n    bin.install \"%s\"\n  end\n", binaryName)
	fmt.Fprintf(&b, "\n  test do\n    assert_match \"%s v#{version}\", shell_output(\"#{bin}/%s version\")\n  end\nend\n",
		binaryName, binaryName)
	return b.String(), nil
}

// Options는 `make dist` 한 번이다.
type Options struct {
	// ModuleDir은 모듈 루트다 (`go build`가 여기서 돈다 — vcs 스탬핑이 이 저장소를 본다).
	ModuleDir string
	// OutDir은 아카이브·체크섬·포뮬러가 놓이는 곳이다 (`dist`).
	OutDir string
	// Version은 태그다. 주입되는 유일한 값이다.
	Version string
	// Repo는 `소유자/이름`이다 (포뮬러의 URL).
	Repo string
	// Targets는 만들 플랫폼이다. 비어 있으면 Targets()다.
	Targets []Target
}

// Result는 놓인 것들이다.
type Result struct {
	Artifacts     []Artifact
	ChecksumsPath string
	FormulaPath   string
}

// Build는 타깃마다 크로스 컴파일해 아카이브를 놓고, 체크섬 한 파일과 포뮬러 하나를 낸다.
//
// 재현성은 계약이다 (ADR 0061 결정 9): `CGO_ENABLED=0` · `-trimpath` · 주입은 태그 하나 ·
// tar 헤더에 시각과 소유자를 남기지 않는다. 같은 태그에서 두 번 빌드하면 같은 바이트가 나온다.
func Build(ctx context.Context, opts Options) (Result, error) {
	targets := opts.Targets
	if len(targets) == 0 {
		targets = Targets()
	}
	if opts.Version == "" {
		return Result{}, fmt.Errorf("버전(태그)이 필요하다")
	}
	if err := os.MkdirAll(opts.OutDir, 0o755); err != nil {
		return Result{}, err
	}

	// 바이너리는 임시 디렉토리에 나오고 아카이브만 OutDir에 남는다 — 저자가 `dist/`에서 찾을
	// 것은 받을 수 있는 파일뿐이다.
	work, err := os.MkdirTemp("", "balpyo-release-")
	if err != nil {
		return Result{}, err
	}
	defer os.RemoveAll(work)

	var result Result
	for _, target := range targets {
		binary := filepath.Join(work, target.OS+"_"+target.Arch, binaryName)
		if err := os.MkdirAll(filepath.Dir(binary), 0o755); err != nil {
			return Result{}, err
		}
		if err := buildBinary(ctx, opts, target, binary); err != nil {
			return Result{}, fmt.Errorf("%s 빌드 실패: %w", target, err)
		}
		name := ArchiveName(opts.Version, target)
		path := filepath.Join(opts.OutDir, name)
		if err := writeArchive(path, binary); err != nil {
			return Result{}, fmt.Errorf("%s 아카이브 실패: %w", target, err)
		}
		sum, err := sha256File(path)
		if err != nil {
			return Result{}, err
		}
		result.Artifacts = append(result.Artifacts, Artifact{Name: name, SHA256: sum, Target: target})
	}

	result.ChecksumsPath = filepath.Join(opts.OutDir, ChecksumsName)
	if err := os.WriteFile(result.ChecksumsPath, []byte(Checksums(result.Artifacts)), 0o644); err != nil {
		return Result{}, err
	}
	// 포뮬러는 **플랫폼 표를 전부 만들었을 때만** 나온다. 타깃 몇 개만 만든 빌드(`TARGETS=`)에서
	// 포뮬러를 내면 빈 url·빈 sha256을 담은 .rb가 놓이고, 그것은 받는 사람의 기계에서만 실패한다.
	if len(result.Artifacts) == len(Targets()) {
		formula, err := Formula(FormulaSpec{Version: opts.Version, Repo: opts.Repo, Artifacts: result.Artifacts})
		if err != nil {
			return Result{}, err
		}
		result.FormulaPath = filepath.Join(opts.OutDir, FormulaName)
		if err := os.WriteFile(result.FormulaPath, []byte(formula), 0o644); err != nil {
			return Result{}, err
		}
	}
	return result, nil
}

// buildBinary는 타깃 하나를 빌드한다. `-buildvcs`를 끄지 않는 것이 요점이다 — 커밋·시각·더러움은
// 여기서 툴체인이 박는다.
func buildBinary(ctx context.Context, opts Options, target Target, out string) error {
	absOut, err := filepath.Abs(out)
	if err != nil {
		return err
	}
	cmd := exec.CommandContext(ctx, "go", "build",
		"-trimpath", "-ldflags", LDFlags(opts.Version), "-o", absOut, "./cmd/"+binaryName)
	cmd.Dir = opts.ModuleDir
	cmd.Env = append(os.Environ(),
		"GOOS="+target.OS,
		"GOARCH="+target.Arch,
		"CGO_ENABLED=0",
	)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("%w: %s", err, strings.TrimSpace(string(output)))
	}
	return nil
}

// writeArchive는 바이너리 하나를 담은 tar.gz를 쓴다. 헤더는 **결정적**이다: 시각은 epoch 0,
// 소유자는 0, 이름은 없다 (빌드한 사람과 시각이 아카이브에 남으면 같은 태그의 두 빌드가 다른
// 바이트가 된다).
func writeArchive(path, binary string) error {
	data, err := os.ReadFile(binary)
	if err != nil {
		return err
	}
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	// gzip 헤더도 시각을 담는다 — 기본(zero ModTime)이 0을 쓴다.
	gz := gzip.NewWriter(file)
	tw := tar.NewWriter(gz)
	header := &tar.Header{
		Typeflag: tar.TypeReg,
		Name:     binaryName,
		Mode:     0o755,
		Size:     int64(len(data)),
		ModTime:  time.Unix(0, 0).UTC(),
		Format:   tar.FormatUSTAR,
	}
	if err := tw.WriteHeader(header); err != nil {
		return err
	}
	if _, err := tw.Write(data); err != nil {
		return err
	}
	if err := tw.Close(); err != nil {
		return err
	}
	if err := gz.Close(); err != nil {
		return err
	}
	return file.Close()
}

func sha256File(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}
