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

// 로컬 디렉토리에서 설치 → 이름으로 빌드 → 목록이 경로를 출처로 보인다 → 원본을 고치고 update.
// 설치한 것은 사본이라 원본 디렉토리를 지워도 빌드된다.
func TestThemeInstallFromLocalPath(t *testing.T) {
	isolatedHome(t)
	src := filepath.Join(t.TempDir(), "brand")
	for name, data := range map[string]string{
		"theme.json":        `{"name":"acme","extends":"ink"}`,
		"layouts/title.css": "/* v1 */",
	} {
		p := filepath.Join(src, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(data), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	var stdout, stderr bytes.Buffer
	if err := run([]string{"theme", "install", src}, &stdout, &stderr); err != nil {
		t.Fatalf("%v\nstderr: %s", err, stderr.String())
	}
	for _, want := range []string{"테마 1개를 복사해 설치했다", "acme", `"theme": "acme"`} {
		if !strings.Contains(stdout.String(), want) {
			t.Errorf("설치 출력에 %q가 없다:\n%s", want, stdout.String())
		}
	}

	dir := newProject(t)
	if err := os.WriteFile(filepath.Join(dir, "balpyo.json"), []byte(`{"title":"t","theme":"acme"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	if err := run([]string{"theme", "list", "--project", dir}, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), "acme") || !strings.Contains(stdout.String(), "brand") {
		t.Errorf("목록이 로컬 출처를 보이지 않았다:\n%s", stdout.String())
	}

	if err := os.WriteFile(filepath.Join(src, "layouts", "title.css"), []byte("/* v2 */"), 0o644); err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	if err := run([]string{"theme", "update", "acme"}, &stdout, &stderr); err != nil {
		t.Fatalf("%v\nstderr: %s", err, stderr.String())
	}
	home := os.Getenv(themestore.HomeEnv)
	if css, _ := os.ReadFile(filepath.Join(home, "themes", "acme", "layouts", "title.css")); string(css) != "/* v2 */" {
		t.Errorf("update가 원본에서 다시 복사하지 않았다: %q", css)
	}

	if err := os.RemoveAll(src); err != nil {
		t.Fatal(err)
	}
	if err := run([]string{"build", "--project", dir, "--strict"}, &stdout, &stderr); err != nil {
		t.Fatalf("원본을 지운 뒤 설치한 사본으로 빌드되지 않았다: %v\nstderr: %s", err, stderr.String())
	}
	html, _ := os.ReadFile(filepath.Join(dir, "dist", "index.html"))
	if !strings.Contains(string(html), "#15171B") {
		t.Error("산출물이 상속한 ink의 bg 토큰을 들고 있지 않다")
	}
}

// 설치한 테마의 그림을 `theme:`으로 가리킨다 — 프로젝트에는 themes/가 없다. 부모에만 있는 그림은 부모의
// 자리, 자식이 덮은 그림은 자식의 자리로 산출물에 놓이고, 쓰지 않은 그림은 나가지 않는다.
func TestInstalledThemeImagesBuildWithoutProjectThemes(t *testing.T) {
	isolatedHome(t)
	var stdout, stderr bytes.Buffer
	if err := run([]string{"theme", "install", filepath.Join("..", "..", "testdata", "fixtures", "19_theme_image", "themes")}, &stdout, &stderr); err != nil {
		t.Fatalf("%v\nstderr: %s", err, stderr.String())
	}
	dir := newProject(t)
	if err := os.WriteFile(filepath.Join(dir, "balpyo.json"), []byte(`{"title":"t","theme":"corp-dark"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	slide := `<slide><image src="theme:icons/check.png" w="64"/><image src="theme:icons/arrow.png" w="64"/></slide>`
	if err := os.WriteFile(filepath.Join(dir, "slides", "001_title.slide"), []byte(slide), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := run([]string{"build", "--project", dir, "--target", "html", "--strict"}, &stdout, &stderr); err != nil {
		t.Fatalf("설치한 테마의 그림으로 빌드되지 않았다: %v\nstderr: %s", err, stderr.String())
	}
	for path, want := range map[string]bool{
		"themes/corp/assets/icons/check.png":      true,
		"themes/corp-dark/assets/icons/arrow.png": true,
		"themes/corp/assets/icons/arrow.png":      false,
		"themes/corp/assets/cover.png":            false,
	} {
		_, err := os.Stat(filepath.Join(dir, "dist", filepath.FromSlash(path)))
		if (err == nil) != want {
			t.Errorf("dist/%s 있음 = %v, want %v", path, err == nil, want)
		}
	}
}
