package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/k2sebeom/balpyo/internal/themestore"
)

// TestMain은 이 패키지의 모든 테스트를 빈 홈에 묶는다 — 개발자가 ~/.balpyo에 설치한 테마가 CLI
// 테스트의 어휘에 새지 않는다.
func TestMain(m *testing.M) {
	home, err := os.MkdirTemp("", "balpyo-home-")
	if err != nil {
		panic(err)
	}
	os.Setenv(themestore.HomeEnv, home)
	code := m.Run()
	os.RemoveAll(home)
	os.Exit(code)
}

// isolatedHome은 테스트 하나에 새 홈을 준다.
func isolatedHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv(themestore.HomeEnv, home)
	return home
}

// serveRepo는 codeload 모양으로 저장소 하나를 내는 서버를 themeClient에 건다.
func serveRepo(t *testing.T, files map[string]string) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/acme/themes/tar.gz/HEAD" {
			http.NotFound(w, r)
			return
		}
		var buf bytes.Buffer
		gz := gzip.NewWriter(&buf)
		tw := tar.NewWriter(gz)
		for name, data := range files {
			tw.WriteHeader(&tar.Header{Typeflag: tar.TypeReg, Name: "themes-main/" + name, Mode: 0o644, Size: int64(len(data))})
			tw.Write([]byte(data))
		}
		tw.Close()
		gz.Close()
		w.Write(buf.Bytes())
	}))
	t.Cleanup(srv.Close)
	prev := themeClient
	themeClient = themestore.Client{BaseURL: srv.URL}
	t.Cleanup(func() { themeClient = prev })
}

// 설치 → 이름으로 빌드 → 목록 → 지우기 → 지운 이름은 알 수 없는 테마. 한 저장소의 테마 둘(상속 관계)이다.
func TestThemeInstallBuildListRemove(t *testing.T) {
	isolatedHome(t)
	serveRepo(t, map[string]string{
		"light/theme.json":       `{"name":"acme","extends":"ink"}`,
		"dark/theme.json":        `{"name":"acme-dark","extends":"acme"}`,
		"dark/layouts/title.css": "display: grid;\ncolor: red;",
	})
	var stdout, stderr bytes.Buffer
	if err := run([]string{"theme", "install", "github.com/acme/themes"}, &stdout, &stderr); err != nil {
		t.Fatalf("%v\nstderr: %s", err, stderr.String())
	}
	for _, want := range []string{"테마 2개를 설치했다", "acme", "acme-dark", `"theme": "acme"`} {
		if !strings.Contains(stdout.String(), want) {
			t.Errorf("설치 출력에 %q가 없다:\n%s", want, stdout.String())
		}
	}
	if !strings.Contains(stderr.String(), "themes/acme-dark/layouts/title.css") {
		t.Errorf("설치가 CSS 경고를 내지 않았다:\n%s", stderr.String())
	}

	dir := newProject(t)
	if err := os.WriteFile(filepath.Join(dir, "balpyo.json"), []byte(`{"title":"t","theme":"acme-dark"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	stderr.Reset()
	// --strict: 설치한 테마의 CSS 경고는 빌드에서 나지 않는다 (덱 저자가 고칠 수 없다).
	if err := run([]string{"build", "--project", dir, "--strict"}, &stdout, &stderr); err != nil {
		t.Fatalf("설치한 테마로 빌드되지 않았다: %v\nstderr: %s", err, stderr.String())
	}
	html, _ := os.ReadFile(filepath.Join(dir, "dist", "index.html"))
	if !strings.Contains(string(html), "#15171B") {
		t.Error("산출물이 상속한 ink의 bg 토큰을 들고 있지 않다")
	}

	stdout.Reset()
	if err := run([]string{"theme", "list", "--project", dir}, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"내장", "minimal", "설치한 테마", "acme-dark", "github.com/acme/themes/dark"} {
		if !strings.Contains(stdout.String(), want) {
			t.Errorf("목록에 %q가 없다:\n%s", want, stdout.String())
		}
	}

	stdout.Reset()
	if err := run([]string{"theme", "remove", "acme", "acme-dark"}, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	err := run([]string{"build", "--project", dir}, &stdout, &stderr)
	if err == nil || !strings.Contains(err.Error(), `알 수 없는 테마 "acme-dark"`) ||
		!strings.Contains(err.Error(), "설치한 테마도 아니다") {
		t.Errorf("지운 테마의 이름은 알 수 없는 테마여야 한다: %v", err)
	}
}

// 프로젝트 themes/와 설치한 테마가 겹쳐도 그 이름을 요청하지 않으면 빌드되고, 목록이 겹침을 표시한다.
func TestThemeListMarksClash(t *testing.T) {
	home := isolatedHome(t)
	dir := newProject(t)
	for _, root := range []string{home, dir} {
		theme := filepath.Join(root, "themes", "corp")
		if err := os.MkdirAll(theme, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(theme, "theme.json"), []byte(`{"name":"corp","extends":"minimal"}`), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	var stdout, stderr bytes.Buffer
	if err := run([]string{"build", "--project", dir}, &stdout, &stderr); err != nil {
		t.Fatalf("겹친 이름을 요청하지 않았는데 빌드가 멈췄다: %v", err)
	}
	if err := run([]string{"build", "--project", dir, "--theme", "corp"}, &stdout, &stderr); err == nil {
		t.Error("겹친 이름을 요청했는데 빌드됐다")
	}
	stdout.Reset()
	if err := run([]string{"theme", "list", "--project", dir}, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), "출처 기록 없음") || !strings.Contains(stdout.String(), "프로젝트 themes/와 겹친다") {
		t.Errorf("목록이 출처 없음 · 겹침을 표시하지 않았다:\n%s", stdout.String())
	}
}

func TestThemeCommandErrors(t *testing.T) {
	isolatedHome(t)
	var stdout, stderr bytes.Buffer
	cases := map[string][]string{
		"명령 없음":    {"theme"},
		"모르는 명령":   {"theme", "get"},
		"주소 없음":    {"theme", "install"},
		"GitHub 밖": {"theme", "install", "gitlab.com/acme/themes"},
		"이름 없음":    {"theme", "remove"},
		"없는 테마":    {"theme", "remove", "nope"},
	}
	for name, args := range cases {
		if err := run(args, &stdout, &stderr); err == nil {
			t.Errorf("%s: 에러여야 한다", name)
		}
	}
}
