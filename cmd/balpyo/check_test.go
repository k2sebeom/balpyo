package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/k2sebeom/balpyo/internal/testkit"
)

// `balpyo check`의 CLI 계약 (ADR 0048): 진단은 stderr에 build와 **같은 모양**으로, 요약은
// stdout에 한 줄로, 그리고 디스크에는 아무것도.

func TestCheckCommandPassesAndWritesNothing(t *testing.T) {
	dir := newProject(t)
	var stdout, stderr bytes.Buffer
	if err := run([]string{"check", "--project", dir}, &stdout, &stderr); err != nil {
		t.Fatalf("%v\nstderr: %s", err, stderr.String())
	}
	out := stdout.String()
	for _, want := range []string{"슬라이드 1장", "진단 없다", "--check-layout"} {
		if !strings.Contains(out, want) {
			t.Errorf("출력에 %q가 없다:\n%s", want, out)
		}
	}
	if stderr.Len() != 0 {
		t.Errorf("깨끗한 덱에 stderr 출력이 있다:\n%s", stderr.String())
	}
	if _, err := os.Stat(filepath.Join(dir, "dist")); !os.IsNotExist(err) {
		t.Errorf("check가 출력 디렉토리를 만들었다: %v", err)
	}
}

// 경고는 stderr에 나오고 종료 코드는 0이다 (build와 같은 규약). `--strict`가 그것을 1로 바꾼다.
func TestCheckCommandWarningsExitZeroAndStrictFails(t *testing.T) {
	dir := newProject(t)
	slide := `<slide id="v"><text variant="hero">제목</text></slide>`
	if err := os.WriteFile(filepath.Join(dir, "slides", "001_title.slide"), []byte(slide), 0o644); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	if err := run([]string{"check", "--project", dir}, &stdout, &stderr); err != nil {
		t.Fatalf("경고가 검사를 실패시켰다: %v", err)
	}
	if !strings.Contains(stderr.String(), "warning: variant") {
		t.Errorf("경고가 stderr에 없다:\n%s", stderr.String())
	}
	if !strings.Contains(stdout.String(), "경고 1건") {
		t.Errorf("요약에 경고 수가 없다:\n%s", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	err := run([]string{"check", "--project", dir, "--strict"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("--strict가 경고를 통과시켰다")
	}
	if !strings.Contains(err.Error(), "검사 실패: 에러 1건") {
		t.Errorf("에러 요약 = %v", err)
	}
	// 진단은 stderr에 **한 번** 나오고 요약이 그것을 다시 찍지 않는다 (build와 같은 규칙).
	if strings.Count(stderr.String(), "variant \"hero\"") != 1 {
		t.Errorf("진단이 한 번 찍히지 않았다:\n%s", stderr.String())
	}
}

func TestCheckCommandReportsDiagnosticErrors(t *testing.T) {
	dir := newProject(t)
	slide := `<slide id="e"><text role="heding">제목</text></slide>`
	if err := os.WriteFile(filepath.Join(dir, "slides", "001_title.slide"), []byte(slide), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	err := run([]string{"check", "--project", dir}, &stdout, &stderr)
	if err == nil {
		t.Fatal("알 수 없는 role을 통과시켰다")
	}
	if !strings.Contains(err.Error(), "검사 실패: 에러 1건") {
		t.Errorf("에러 요약 = %v", err)
	}
	if !strings.Contains(stderr.String(), "error: 알 수 없는 role") {
		t.Errorf("진단이 stderr에 없다:\n%s", stderr.String())
	}
	if stdout.Len() != 0 {
		t.Errorf("실패한 검사가 요약을 찍었다:\n%s", stdout.String())
	}
}

// `--check-layout`이 배치 경고를 켠다 — CI 레시피의 나머지 절반이다.
func TestCheckCommandLayoutWarnings(t *testing.T) {
	testkit.RequireChromium(t)
	dir := newProject(t)
	// 1200px 도형은 1152px 콘텐츠 폭을 48px 넘친다.
	slide := `<slide id="wide"><shape type="rect" fill="surface" w="1200" h="100"></shape></slide>`
	if err := os.WriteFile(filepath.Join(dir, "slides", "001_title.slide"), []byte(slide), 0o644); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	if err := run([]string{"check", "--project", dir, "--check-layout"}, &stdout, &stderr); err != nil {
		t.Fatalf("배치 경고가 검사를 실패시켰다: %v", err)
	}
	if !strings.Contains(stderr.String(), "48px 넘친다") {
		t.Errorf("배치 경고가 없다:\n%s", stderr.String())
	}
	if !strings.Contains(stdout.String(), "배치까지 봤다") {
		t.Errorf("무엇을 봤는지 말하지 않는다:\n%s", stdout.String())
	}
	if _, err := os.Stat(filepath.Join(dir, "dist")); !os.IsNotExist(err) {
		t.Errorf("측정이 출력 디렉토리를 만들었다: %v", err)
	}
}

func TestCheckCommandRejectsArgs(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if err := run([]string{"check", newProject(t)}, &stdout, &stderr); err == nil {
		t.Error("위치 인자를 받아들였다 (--project가 있다)")
	}
}
