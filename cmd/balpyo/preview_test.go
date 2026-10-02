package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/k2sebeom/balpyo/internal/preview"
	"github.com/k2sebeom/balpyo/internal/testkit"
)

// `balpyo preview`의 CLI 계약이다 (ADR 0047).
//
//  1. 갤러리 경로를 **찍는다** — 사람이 그 줄을 열어 눈으로 본다.
//  2. 허용 오차를 넘어도 **종료 코드는 0이다**: 두 렌더가 달라야 정답인 슬라이드가 있다.
//  3. 도구가 없으면 빌드 전에 멈춘다 — 조용히 웹 열만 있는 갤러리를 내지 않는다.

func TestPreviewCommandWritesGallery(t *testing.T) {
	testkit.RequireChromium(t)
	testkit.RequireSoffice(t)
	testkit.RequirePdftoppm(t)

	dir := newProject(t)
	var stdout, stderr bytes.Buffer
	if err := run([]string{"preview", "--project", dir}, &stdout, &stderr); err != nil {
		t.Fatalf("%v\nstderr: %s", err, stderr.String())
	}

	gallery := filepath.Join(dir, preview.DefaultDirName, preview.IndexFile)
	if _, err := os.Stat(gallery); err != nil {
		t.Fatalf("갤러리가 만들어지지 않았다: %v", err)
	}
	out := stdout.String()
	for _, want := range []string{"슬라이드 1장", gallery, "PowerPoint"} {
		if !strings.Contains(out, want) {
			t.Errorf("출력에 %q가 없다:\n%s", want, out)
		}
	}
	// 검수는 빌드 산출물도 남긴다 — 갤러리로 판정할 수 없는 확인(PPTX를 직접 열어 보기)이 있다.
	if _, err := os.Stat(filepath.Join(dir, "dist", "my-talk.pptx")); err != nil {
		t.Errorf("PPTX 산출물이 없다: %v", err)
	}
}

// 웹만 보는 검수는 LibreOffice 없이 돈다 — 저작 루프가 래스터 도구에 묶이지 않아야 한다.
func TestPreviewCommandWebOnlyNeedsNoLibreOffice(t *testing.T) {
	testkit.RequireChromium(t)
	t.Setenv(preview.EnvSoffice, filepath.Join(t.TempDir(), "없는-soffice"))

	dir := newProject(t)
	var stdout, stderr bytes.Buffer
	if err := run([]string{"preview", "--project", dir, "--target", "html",
		"--gallery", filepath.Join(dir, "검수")}, &stdout, &stderr); err != nil {
		t.Fatalf("%v\nstderr: %s", err, stderr.String())
	}
	if _, err := os.Stat(filepath.Join(dir, "검수", preview.IndexFile)); err != nil {
		t.Fatalf("--gallery가 무시됐다: %v", err)
	}
	// 볼 PPTX가 없으면 PPTX를 열어 보라고 하지 않는다.
	if strings.Contains(stdout.String(), "PowerPoint") {
		t.Errorf("pptx를 만들지 않았는데 직접 열어 보라고 한다:\n%s", stdout.String())
	}
}

// 도구가 없으면 **빌드 전에** 멈추고, 다음에 할 일을 문장으로 말한다.
func TestPreviewCommandStopsWhenToolsAreMissing(t *testing.T) {
	dir := newProject(t)
	t.Setenv(preview.EnvSoffice, filepath.Join(t.TempDir(), "없는-soffice"))

	var stdout, stderr bytes.Buffer
	err := run([]string{"preview", "--project", dir, "--target", "pptx"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("도구가 없으면 에러여야 한다")
	}
	for _, want := range []string{preview.EnvSoffice, "--target html"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("에러에 %q가 없다: %v", want, err)
		}
	}
	if _, statErr := os.Stat(filepath.Join(dir, preview.DefaultDirName)); !os.IsNotExist(statErr) {
		t.Errorf("도구가 없는데 갤러리 디렉토리를 만들었다: %v", statErr)
	}
}

func TestPreviewCommandRejectsBadInput(t *testing.T) {
	dir := newProject(t)
	tests := []struct {
		name string
		args []string
	}{
		{"알 수 없는 타깃", []string{"preview", "--project", dir, "--target", "pdf"}},
		{"남는 인자", []string{"preview", "--project", dir, "slides"}},
		{"빈 타깃", []string{"preview", "--project", dir, "--target", ""}},
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

// 사용법에 preview가 있어야 한다 — 서브커맨드를 늘리고 안내를 잊으면 아무도 찾지 못한다.
func TestUsageMentionsPreview(t *testing.T) {
	var out bytes.Buffer
	usage(&out)
	for _, want := range []string{"balpyo preview", "--gallery", "LibreOffice"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("사용법에 %q가 없다:\n%s", want, out.String())
		}
	}
}
