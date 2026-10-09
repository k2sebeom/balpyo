package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/k2sebeom/balpyo/internal/testkit"
)

// newProject는 슬라이드 하나가 있는 프로젝트를 만든다.
func newProject(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "slides"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "balpyo.json"), []byte(`{"title": "my talk"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "slides", "001_title.slide"), []byte("<slide></slide>\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestBuildCommand(t *testing.T) {
	dir := newProject(t)
	var stdout, stderr bytes.Buffer
	err := run([]string{"build", "--project", dir, "--target", "html,pptx"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("%v\nstderr: %s", err, stderr.String())
	}
	out := stdout.String()
	for _, want := range []string{"슬라이드 1장", "index.html", "my-talk.pptx"} {
		if !strings.Contains(out, want) {
			t.Errorf("출력에 %q가 없다:\n%s", want, out)
		}
	}
	for _, name := range []string{"index.html", "my-talk.pptx"} {
		if _, err := os.Stat(filepath.Join(dir, "dist", name)); err != nil {
			t.Errorf("%s가 만들어지지 않았다: %v", name, err)
		}
	}
}

func TestBuildCommandOutFlagOverridesConfig(t *testing.T) {
	dir := newProject(t)
	out := filepath.Join(t.TempDir(), "public")
	var stdout, stderr bytes.Buffer
	if err := run([]string{"build", "--project", dir, "--out", out}, &stdout, &stderr); err != nil {
		t.Fatalf("%v\nstderr: %s", err, stderr.String())
	}
	if _, err := os.Stat(filepath.Join(out, "index.html")); err != nil {
		t.Errorf("--out이 balpyo.json을 덮어쓰지 않았다: %v", err)
	}
}

// `--check-layout`은 좌표를 쓰는 타깃이 없어도 배치를 검사한다 — 저작 루프가
// 넘침을 빌드 로그에서 보는 길이다. 경고이므로 종료 코드는 0이고, `--strict`가 그것을 막는다.
func TestBuildCheckLayoutPrintsWarnings(t *testing.T) {
	testkit.RequireChromium(t)
	dir := newProject(t)
	// 1200px 도형은 1152px 콘텐츠 폭을 48px 넘친다 — 넘친 양이 선언한 px에서 나온다.
	slide := `<slide id="wide"><shape type="rect" fill="surface" w="1200" h="100"></shape></slide>`
	if err := os.WriteFile(filepath.Join(dir, "slides", "001_title.slide"), []byte(slide), 0o644); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	if err := run([]string{"build", "--project", dir, "--check-layout"}, &stdout, &stderr); err != nil {
		t.Fatalf("경고가 빌드를 멈췄다: %v\nstderr: %s", err, stderr.String())
	}
	if want := "콘텐츠 박스를 오른쪽으로 48px 넘친다"; !strings.Contains(stderr.String(), want) {
		t.Errorf("stderr에 %q가 없다:\n%s", want, stderr.String())
	}

	// 플래그가 없으면 브라우저를 띄우지 않으므로 경고도 없다 (기본은 꺼짐).
	stdout.Reset()
	stderr.Reset()
	if err := run([]string{"build", "--project", dir}, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	if stderr.Len() > 0 {
		t.Errorf("--check-layout 없는 빌드의 stderr:\n%s", stderr.String())
	}

	stderr.Reset()
	if err := run([]string{"build", "--project", dir, "--check-layout", "--strict"}, &stdout, &stderr); err == nil {
		t.Error("--strict가 배치 경고를 통과시켰다")
	}
}

func TestBuildCommandRejectsBadInput(t *testing.T) {
	dir := newProject(t)
	tests := []struct {
		name string
		args []string
	}{
		{"알 수 없는 타깃", []string{"build", "--project", dir, "--target", "pdf"}},
		{"남는 인자", []string{"build", "--project", dir, "slides"}},
		{"슬라이드 없는 디렉토리", []string{"build", "--project", t.TempDir()}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			if err := run(tc.args, &stdout, &stderr); err == nil {
				t.Error("에러여야 한다")
			}
		})
	}
}

// TestBuildErrorDiagnosticsPrintedOnce는 빌드가 진단 에러로 실패할 때 각 진단이 stderr에
// 정확히 한 번만 나오는지 본다 — 예전에는 printDiagnostics와 err 출력이 겹쳐 두 번 찍혔다.
func TestBuildErrorDiagnosticsPrintedOnce(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "slides"), 0o755); err != nil {
		t.Fatal(err)
	}
	// 알 수 없는 태그 하나 — 정확히 진단 하나를 낸다. 이름은 어휘에 **없는** 것이어야 한다:
	// 예전에는 <table>이었는데 그 태그가 구현되면서 이 테스트가 다른 실패 경로를 봤다.
	if err := os.WriteFile(filepath.Join(dir, "slides", "001.slide"),
		[]byte("<slide><marquee></marquee></slide>\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	err := run([]string{"build", "--project", dir, "--target", "html"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("에러여야 한다")
	}
	// 상세 진단은 stderr에 한 번만.
	if n := strings.Count(stderr.String(), "알 수 없는 태그 <marquee>"); n != 1 {
		t.Errorf("진단이 stderr에 %d번 찍혔다 (1번이어야 한다):\n%s", n, stderr.String())
	}
	// 돌려준 에러는 요약이어야 한다 — main이 이걸 다시 출력해도 진단이 겹치지 않게.
	if strings.Contains(err.Error(), "알 수 없는 태그 <marquee>") {
		t.Errorf("반환 에러가 상세 진단을 되풀이한다 (요약이어야 한다): %v", err)
	}
}

func TestVersionAndHelp(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if err := run([]string{"version"}, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(stdout.String(), "balpyo ") {
		t.Errorf("version 출력 = %q", stdout.String())
	}

	stdout.Reset()
	if err := run([]string{"help"}, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), "balpyo build") {
		t.Error("help가 build를 설명해야 한다")
	}
}

func TestUnknownSubcommand(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if err := run([]string{"serve"}, &stdout, &stderr); err == nil {
		t.Error("알 수 없는 서브커맨드는 에러여야 한다")
	}
	// 사용법은 stderr로 나가야 파이프를 오염시키지 않는다.
	if !strings.Contains(stderr.String(), "사용법") {
		t.Error("에러 시 사용법을 보여줘야 한다")
	}

	stdout.Reset()
	stderr.Reset()
	if err := run(nil, &stdout, &stderr); err == nil {
		t.Error("서브커맨드 없이 호출하면 에러여야 한다")
	}
}
