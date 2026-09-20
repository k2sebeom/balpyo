package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/k2sebeom/balpyo/internal/testkit"
)

// `balpyo init`의 계약은 파일 목록이 아니라 **그 다음 명령이 성공하는 것**이다 (ADR 0048).
// 그래서 이 테스트는 init → build를 이어서 돌린다.

func TestInitScaffoldBuildsCleanly(t *testing.T) {
	testkit.RequireChromium(t)
	dir := filepath.Join(t.TempDir(), "my-talk")

	var stdout, stderr bytes.Buffer
	if err := run([]string{"init", dir, "--author", "이세범"}, &stdout, &stderr); err != nil {
		t.Fatalf("%v\nstderr: %s", err, stderr.String())
	}
	out := stdout.String()
	for _, want := range []string{"balpyo.json", "slides/001_title.slide", ".gitignore", "balpyo watch"} {
		if !strings.Contains(out, want) {
			t.Errorf("출력에 %q가 없다:\n%s", want, out)
		}
	}

	// 스캐폴딩 덱은 두 타깃 + 배치 검사 + --strict를 **경고 0으로** 통과한다. 첫 명령이 경고를
	// 내면 저자는 경고를 읽지 않는 습관부터 배운다.
	stdout.Reset()
	stderr.Reset()
	if err := run([]string{"build", "--project", dir, "--target", "html,pptx",
		"--check-layout", "--strict"}, &stdout, &stderr); err != nil {
		t.Fatalf("스캐폴딩 덱이 --strict 빌드를 통과하지 못했다: %v\nstderr: %s", err, stderr.String())
	}
	if stderr.Len() != 0 {
		t.Errorf("진단이 나왔다:\n%s", stderr.String())
	}
	if !strings.Contains(stdout.String(), "슬라이드 3장") {
		t.Errorf("슬라이드 3장이 아니다:\n%s", stdout.String())
	}
	// 제목이 없으면 디렉토리 이름이 제목이고, PPTX 파일 이름이 그 제목에서 나온다.
	if _, err := os.Stat(filepath.Join(dir, "dist", "my-talk.pptx")); err != nil {
		t.Errorf("PPTX 산출물이 없다: %v", err)
	}
}

// 브라우저 없이 판정할 수 있는 것: 파일이 생기고, 설정에 값이 실리고, 덮어쓰지 않는다.
func TestInitWritesProjectAndRefusesTwice(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "덱")
	var stdout, stderr bytes.Buffer
	if err := run([]string{"init", "--project", dir, "--title", "첫 덱", "--lang", "en"},
		&stdout, &stderr); err != nil {
		t.Fatalf("%v\nstderr: %s", err, stderr.String())
	}
	cfg, err := os.ReadFile(filepath.Join(dir, "balpyo.json"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`"title": "첫 덱"`, `"lang": "en"`, `"theme": "minimal"`} {
		if !strings.Contains(string(cfg), want) {
			t.Errorf("설정에 %s가 없다:\n%s", want, cfg)
		}
	}

	stdout.Reset()
	stderr.Reset()
	if err := run([]string{"init", dir}, &stdout, &stderr); err == nil {
		t.Error("이미 프로젝트인 자리에 두 번째 init이 성공했다")
	}
}

// 디렉토리는 플래그 앞에도 뒤에도 올 수 있다 — 사람이 가장 먼저 쳐 보는 표기가
// `balpyo init my-talk --author 나`이고, Go의 flag는 그 표기에서 파싱을 멈춘다.
func TestInitAcceptsDirectoryBeforeOrAfterFlags(t *testing.T) {
	root := t.TempDir()
	for i, args := range [][]string{
		{"init", filepath.Join(root, "앞"), "--title", "앞"},
		{"init", "--title", "뒤", filepath.Join(root, "뒤")},
	} {
		var stdout, stderr bytes.Buffer
		if err := run(args, &stdout, &stderr); err != nil {
			t.Fatalf("%d번째 표기가 실패했다: %v\nstderr: %s", i, err, stderr.String())
		}
	}
	for _, name := range []string{"앞", "뒤"} {
		cfg, err := os.ReadFile(filepath.Join(root, name, "balpyo.json"))
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if !strings.Contains(string(cfg), `"title": "`+name+`"`) {
			t.Errorf("%s의 제목이 실리지 않았다:\n%s", name, cfg)
		}
	}
}

func TestInitRejectsBadInput(t *testing.T) {
	dir := t.TempDir()
	tests := []struct {
		name string
		args []string
	}{
		{"내장 테마가 아니다", []string{"init", filepath.Join(dir, "a"), "--theme", "corp"}},
		{"없는 종횡비", []string{"init", filepath.Join(dir, "b"), "--aspect", "21:9"}},
		{"디렉토리 둘", []string{"init", filepath.Join(dir, "c"), filepath.Join(dir, "d")}},
		{"위치 인자와 --project를 함께", []string{"init", filepath.Join(dir, "e"), "--project", filepath.Join(dir, "f")}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			if err := run(tc.args, &stdout, &stderr); err == nil {
				t.Error("에러여야 한다")
			}
		})
	}
	// 거부한 입력은 디렉토리를 만들지 않는다.
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Errorf("거부했는데 %d개를 만들었다", len(entries))
	}
}

func TestUsageMentionsInitAndCheck(t *testing.T) {
	var out bytes.Buffer
	usage(&out)
	for _, want := range []string{"balpyo init", "balpyo check", "--check-layout --strict"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("사용법에 %q가 없다:\n%s", want, out.String())
		}
	}
}
