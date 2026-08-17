package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
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
