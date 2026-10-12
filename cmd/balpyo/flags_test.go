package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// 위치 인자는 플래그 **사이**에도 올 수 있다 — 앞이나 뒤만 받으면 가운데에 둔 디렉토리 뒤의
// 플래그가 전부 위치 인자로 삼켜진다.
func TestInitAcceptsDirectoryBetweenFlags(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "가운데")
	var stdout, stderr bytes.Buffer
	if err := run([]string{"init", "--author", "A", dir, "--theme", "ink"}, &stdout, &stderr); err != nil {
		t.Fatalf("%v\nstderr: %s", err, stderr.String())
	}
	cfg, err := os.ReadFile(filepath.Join(dir, "balpyo.json"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`"author": "A"`, `"theme": "ink"`} {
		if !strings.Contains(string(cfg), want) {
			t.Errorf("balpyo.json에 %s가 없다:\n%s", want, cfg)
		}
	}
}

// 모르는 플래그는 한 번만 말한다: FlagSet이 사용법과 함께 이미 찍었으므로 main이 되풀이하지 않는다.
func TestUnknownFlagReportedOnce(t *testing.T) {
	for _, args := range [][]string{
		{"build", "--bogus"},
		{"check", "--bogus"},
		{"init", "--bogus"},
		{"theme", "install", "--bogus"},
	} {
		var stdout, stderr bytes.Buffer
		if code := cli(args, &stdout, &stderr); code != 1 {
			t.Errorf("%v: 종료 코드 = %d, want 1", args, code)
		}
		if n := strings.Count(stderr.String(), "flag provided but not defined: -bogus"); n != 1 {
			t.Errorf("%v: 에러가 %d번 찍혔다 (1번이어야 한다):\n%s", args, n, stderr.String())
		}
	}
}

// `-h`는 사용법을 청한 것이지 실패가 아니다 — 0으로 끝나고 `flag: help requested`를 덧붙이지 않는다.
func TestSubcommandHelpExitsZero(t *testing.T) {
	for _, args := range [][]string{
		{"build", "-h"},
		{"check", "--help"},
		{"init", "-h"},
		{"snap", "-h"},
		{"theme", "install", "-h"},
	} {
		var stdout, stderr bytes.Buffer
		if code := cli(args, &stdout, &stderr); code != 0 {
			t.Errorf("%v: 종료 코드 = %d, want 0", args, code)
		}
		out := stdout.String() + stderr.String()
		if !strings.Contains(out, "Usage of") {
			t.Errorf("%v: 사용법이 없다:\n%s", args, out)
		}
		if strings.Contains(out, "help requested") {
			t.Errorf("%v: 내부 에러 문구가 새어 나왔다:\n%s", args, out)
		}
	}
}

// 플래그가 아닌 에러는 main이 찍는다 — 한 번, 종료 코드 1로.
func TestOtherErrorsPrintedByMain(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := cli([]string{"build", "--project", t.TempDir(), "extra"}, &stdout, &stderr); code != 1 {
		t.Errorf("종료 코드 = %d, want 1", code)
	}
	if n := strings.Count(stderr.String(), "build takes no arguments"); n != 1 {
		t.Errorf("에러가 %d번 찍혔다 (1번이어야 한다):\n%s", n, stderr.String())
	}
}
