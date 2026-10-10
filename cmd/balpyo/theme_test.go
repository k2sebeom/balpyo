package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// `--theme`의 CLI 계약: 플래그가 `balpyo.json`의 theme를 덮는다.
//
// 이 플래그가 있는 이유는 **한 덱을 두 테마로 보는 일**이 테마를 고르는 방법이라는 것이다 —
// 설정을 고쳐 빌드하고 되돌리는 것은 그 비교를 손으로 하는 일이고, 되돌리기를 잊으면 잘못된
// 테마가 커밋된다. `--out`이 이미 같은 태도를 갖는다 (플래그가 설정을 덮는다).

func TestBuildThemeFlagOverridesConfig(t *testing.T) {
	dir := newProject(t)
	var stdout, stderr bytes.Buffer
	if err := run([]string{"build", "--project", dir, "--theme", "ink"}, &stdout, &stderr); err != nil {
		t.Fatalf("%v\nstderr: %s", err, stderr.String())
	}
	html, err := os.ReadFile(filepath.Join(dir, "dist", "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	// 산출물이 ink의 토큰을 들고 있어야 한다 — 설정에는 theme가 없으므로(기본 minimal) 이
	// 값이 곧 플래그가 값 층에 닿았다는 증거다. 바탕색을 보는 이유: 테마의 첫 결정이고,
	// 밝은 테마와 어두운 테마를 한 문자열로 구별한다.
	if !strings.Contains(string(html), "#15171B") {
		t.Errorf("ink의 bg 토큰이 산출물에 없다 — --theme가 설정을 덮지 않았다")
	}
	if strings.Contains(string(html), "#FFFFFF") {
		t.Errorf("minimal의 bg 토큰이 남아 있다 — 두 테마가 섞였다")
	}
}

// 설정에 theme가 있어도 플래그가 이긴다 — 덮는다는 말의 뜻이다.
func TestBuildThemeFlagBeatsConfigValue(t *testing.T) {
	dir := newProject(t)
	cfg := `{"title": "my talk", "theme": "ink"}`
	if err := os.WriteFile(filepath.Join(dir, "balpyo.json"), []byte(cfg), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	if err := run([]string{"build", "--project", dir, "--theme", "editorial"}, &stdout, &stderr); err != nil {
		t.Fatalf("%v\nstderr: %s", err, stderr.String())
	}
	html, err := os.ReadFile(filepath.Join(dir, "dist", "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(html), "#FAF8F4") {
		t.Errorf("editorial의 bg 토큰이 산출물에 없다 — 설정의 ink가 이겼다")
	}
}

// 어휘 밖 이름은 **플래그를 가리키며** 멈춘다: 고칠 자리가 `balpyo.json`이 아니라 방금 친
// 명령이므로, 진단이 설정 파일을 가리키면 사람이 없는 오타를 찾는다.
func TestBuildThemeFlagRejectsUnknownName(t *testing.T) {
	dir := newProject(t)
	var stdout, stderr bytes.Buffer
	err := run([]string{"build", "--project", dir, "--theme", "corp"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("없는 테마를 통과시켰다")
	}
	for _, want := range []string{"--theme", "corp", "minimal"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("에러 메시지에 %q가 없다: %v", want, err)
		}
	}
}

// check도 같은 플래그를 받는다 — 판정은 빌드와 같은 입력을 봐야 한다: 테마를
// 바꾸면 배치가 바뀌므로(editorial의 표지는 아래에 앉는다) `--check-layout`의 판정도 달라진다.
func TestCheckAcceptsThemeFlag(t *testing.T) {
	dir := newProject(t)
	var stdout, stderr bytes.Buffer
	if err := run([]string{"check", "--project", dir, "--theme", "editorial"}, &stdout, &stderr); err != nil {
		t.Fatalf("%v\nstderr: %s", err, stderr.String())
	}
	if !strings.Contains(stdout.String(), "1 slide") {
		t.Errorf("요약이 없다:\n%s", stdout.String())
	}
}
