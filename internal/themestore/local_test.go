package themestore

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/k2sebeom/balpyo/core"
)

// 로컬 설치의 계약: 디렉토리를 GitHub 저장소처럼 훑고 같은 규칙으로 고르되, 받는 대신 복사한다.
// 설치한 테마는 원본과 떨어진 사본이고, 출처는 그 디렉토리로 기록되어 update가 다시 복사한다.

// writeTree는 root 아래에 파일을 쓴다 (슬래시 구분 경로 → 내용).
func writeTree(t *testing.T, root string, files map[string]string) {
	t.Helper()
	for name, data := range files {
		p := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(data), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func mustLocal(t *testing.T, dir string) Source {
	t.Helper()
	src, err := LocalSource(dir)
	if err != nil {
		t.Fatal(err)
	}
	return src
}

func TestResolveSource(t *testing.T) {
	work := t.TempDir()
	writeTree(t, work, map[string]string{"my-themes/theme.json": "{}", "file.txt": "x"})
	t.Chdir(work)
	real, _ := filepath.EvalSymlinks(work)

	locals := map[string]string{
		"./my-themes":                    filepath.Join(real, "my-themes"),
		"my-themes":                      filepath.Join(real, "my-themes"), // 주소가 아닌 있는 디렉토리
		".":                              real,
		filepath.Join(work, "my-themes"): filepath.Join(real, "my-themes"),
		"./my-themes/../my-themes/":      filepath.Join(real, "my-themes"),
	}
	for raw, want := range locals {
		src, err := ResolveSource(raw)
		if err != nil {
			t.Errorf("%s: %v", raw, err)
			continue
		}
		if !src.IsLocal() || src.Dir != want {
			t.Errorf("%s: got %+v, want Dir %s", raw, src, want)
		}
	}
	// GitHub 주소의 뜻은 같은 이름의 디렉토리가 있어도 바뀌지 않는다.
	writeTree(t, work, map[string]string{"github.com/acme/themes/theme.json": "{}"})
	if src, err := ResolveSource("github.com/acme/themes"); err != nil || src.IsLocal() || src.Owner != "acme" {
		t.Errorf("GitHub 주소가 경로로 읽혔다: %+v %v", src, err)
	}
	if src, err := ResolveSource("./github.com/acme/themes"); err != nil || !src.IsLocal() {
		t.Errorf("./로 시작하면 경로여야 한다: %+v %v", src, err)
	}
	for raw, want := range map[string]string{
		"./nope":                 "does not exist",
		"file.txt":               "not a GitHub repository address", // 디렉토리가 아니면 경로로 보지 않는다
		"./file.txt":             "is not a directory",
		"gitlab.com/acme/themes": "path like ./gitlab.com/acme/themes",
	} {
		if _, err := ResolveSource(raw); err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("%s: %q를 담은 에러여야 한다: %v", raw, want, err)
		}
	}
}

// 로컬 디렉토리의 테마 여럿: GitHub과 같이 찾고, 복사한 사본을 설치하고, 디렉토리를 출처로 적는다.
func TestInstallFromLocalDir(t *testing.T) {
	s, _ := newStore(t)
	dir := t.TempDir()
	files := map[string]string{"themes/light/layouts/title.css": "/* v1 */", ".git/config": "x"}
	child(files, "themes/light/", "acme", "")
	child(files, "themes/dark/", "acme-dark", "acme")
	child(files, ".git/themes/hidden/", "hidden", "")
	writeTree(t, dir, files)
	src := mustLocal(t, dir)

	res, err := s.Install(context.Background(), src, InstallOptions{})
	if err != nil {
		t.Fatal(err)
	}
	want := []Record{
		{Name: "acme", Local: src.Dir, Path: "themes/light"},
		{Name: "acme-dark", Local: src.Dir, Path: "themes/dark"},
	}
	if !reflect.DeepEqual(res.Themes, want) {
		t.Errorf("themes = %+v, want %+v", res.Themes, want)
	}
	if list, _ := s.List(); !reflect.DeepEqual(list, want) {
		t.Errorf("list = %+v, want %+v", list, want)
	}
	if got := want[0].Origin(); got != filepath.Join(src.Dir, "themes", "light") {
		t.Errorf("Origin = %q", got)
	}
	if err := core.CheckThemeName(core.WithInstalledThemes(nil, s.ThemesFS()), "acme-dark"); err != nil {
		t.Errorf("설치한 테마가 어휘에 없다: %v", err)
	}

	// 설치한 것은 사본이다 — 원본을 고쳐도 update 전까지는 그대로다.
	writeTree(t, dir, map[string]string{"themes/light/layouts/title.css": "/* v2 */"})
	installed := filepath.Join(s.Dir, "themes", "acme", "layouts", "title.css")
	if css, _ := os.ReadFile(installed); string(css) != "/* v1 */" {
		t.Errorf("설치한 테마가 원본을 따라 바뀌었다: %q", css)
	}
	if info, err := os.Lstat(filepath.Join(s.Dir, "themes", "acme")); err != nil || !info.IsDir() {
		t.Errorf("설치한 테마는 링크가 아니라 디렉토리여야 한다: %v", err)
	}

	// 같은 자리에서 다시 설치하는 것은 갈아 끼우기가 아니다 (--force 없이 된다).
	if _, err := s.Install(context.Background(), src, InstallOptions{Names: []string{"acme"}}); err != nil {
		t.Errorf("같은 디렉토리에서 다시 설치하지 못했다: %v", err)
	}
	if css, _ := os.ReadFile(installed); string(css) != "/* v2 */" {
		t.Errorf("다시 설치가 새 내용을 복사하지 않았다: %q", css)
	}

	// update는 기록한 디렉토리에서 다시 복사한다.
	writeTree(t, dir, map[string]string{"themes/light/layouts/title.css": "/* v3 */"})
	results, err := s.Update(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Errorf("한 디렉토리는 한 번 복사해야 한다: %d", len(results))
	}
	if css, _ := os.ReadFile(installed); string(css) != "/* v3 */" {
		t.Errorf("update가 새 내용을 복사하지 않았다: %q", css)
	}
}

// 테마 디렉토리 하나를 바로 가리켜도 된다 (루트가 테마).
func TestInstallFromLocalThemeDir(t *testing.T) {
	s, _ := newStore(t)
	dir := filepath.Join(t.TempDir(), "whatever-name")
	files := map[string]string{"assets/logo.svg": "<svg/>"}
	child(files, "", "acme", "")
	writeTree(t, dir, files)
	res, err := s.Install(context.Background(), mustLocal(t, dir), InstallOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Themes) != 1 || res.Themes[0].Name != "acme" || res.Themes[0].Path != "." {
		t.Errorf("themes = %+v", res.Themes)
	}
	if _, err := os.Stat(filepath.Join(s.Dir, "themes", "acme", "assets", "logo.svg")); err != nil {
		t.Errorf("테마의 파일이 복사되지 않았다: %v", err)
	}
}

// 로컬과 GitHub은 다른 출처다 — 같은 이름은 --force 없이 갈아 끼우지 않는다.
func TestInstallLocalConflictsWithGitHub(t *testing.T) {
	s, gh := newStore(t)
	files := map[string]string{}
	child(files, "", "acme", "")
	gh.set("acme/theme@HEAD", repo{files: files})
	if _, err := s.Install(context.Background(), mustSource(t, "github.com/acme/theme"), InstallOptions{}); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	writeTree(t, dir, files)
	local := mustLocal(t, dir)
	_, err := s.Install(context.Background(), local, InstallOptions{})
	if err == nil || !strings.Contains(err.Error(), "installed from github.com/acme/theme") {
		t.Fatalf("다른 출처의 같은 이름은 에러여야 한다: %v", err)
	}
	if _, err := s.Install(context.Background(), local, InstallOptions{Force: true}); err != nil {
		t.Fatal(err)
	}
	list, _ := s.List()
	if list[0].Local != local.Dir || list[0].Repo != "" {
		t.Errorf("--force가 출처를 갈아 끼우지 않았다: %+v", list)
	}
	// 거꾸로도 같다.
	if _, err := s.Install(context.Background(), mustSource(t, "github.com/acme/theme"), InstallOptions{}); err == nil ||
		!strings.Contains(err.Error(), "installed from "+local.Dir) {
		t.Errorf("로컬에서 설치한 이름을 GitHub이 --force 없이 갈아 끼웠다: %v", err)
	}
}

// 링크는 따라가지 않는다: 테마 안의 링크는 설치 에러, 테마 밖의 링크는 상관없다.
func TestInstallLocalRejectsLinks(t *testing.T) {
	s, _ := newStore(t)
	dir := t.TempDir()
	files := map[string]string{"secret.txt": "x"}
	child(files, "theme/", "acme", "")
	writeTree(t, dir, files)
	if err := os.Symlink(filepath.Join(dir, "secret.txt"), filepath.Join(dir, "theme", "bg.png")); err != nil {
		t.Skipf("심볼릭 링크를 만들 수 없다: %v", err)
	}
	src := mustLocal(t, dir)
	if _, err := s.Install(context.Background(), src, InstallOptions{}); err == nil || !strings.Contains(err.Error(), "link") {
		t.Errorf("테마 안의 링크는 에러여야 한다: %v", err)
	}
	if names := installedNames(t, s); len(names) != 0 {
		t.Errorf("실패한 설치가 테마를 남겼다: %v", names)
	}
	os.Remove(filepath.Join(dir, "theme", "bg.png"))
	if err := os.Symlink(filepath.Join(dir, "secret.txt"), filepath.Join(dir, "outside")); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Install(context.Background(), src, InstallOptions{}); err != nil {
		t.Errorf("테마 밖의 링크는 설치를 막지 않는다: %v", err)
	}
}

// 홈을 담은 디렉토리를 가리켜도 홈은 복사하지 않는다 (복사가 자기 자신을 담지 않는다).
func TestInstallLocalSkipsHome(t *testing.T) {
	dir := t.TempDir()
	files := map[string]string{}
	child(files, "brand/", "acme", "")
	writeTree(t, dir, files)
	s := &Store{Dir: filepath.Join(dir, "balpyo-home")}
	res, err := s.Install(context.Background(), mustLocal(t, dir), InstallOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Themes) != 1 || res.Themes[0].Name != "acme" {
		t.Errorf("themes = %+v", res.Themes)
	}
	// 홈 안의 설치한 테마가 다음 설치의 후보로 잡히면 같은 이름이 둘이 된다.
	if _, err := s.Install(context.Background(), mustLocal(t, dir), InstallOptions{}); err != nil {
		t.Errorf("홈을 담은 디렉토리에서 다시 설치하지 못했다: %v", err)
	}
}

// 로컬 설치의 CSS 경고는 저자가 고칠 원본 파일을 가리킨다 — 테마 이름과 디렉토리 이름이 달라도
// 홈의 사본이나 `themes/<이름>/`이 아니라 원본 디렉토리 안의 파일이다. 같이 설치한 부모 테마의
// 경고도 그 부모의 원본 자리를 가리킨다.
func TestInstallLocalLintNamesSourceFile(t *testing.T) {
	s, _ := newStore(t)
	dir := t.TempDir()
	files := map[string]string{
		"w/layouts/content.css": "color: red;",
		"p/layouts/title.css":   "color: blue;",
	}
	child(files, "p/", "parent", "")
	child(files, "w/", "warny", "parent")
	writeTree(t, dir, files)
	src := mustLocal(t, dir)

	res, err := s.Install(context.Background(), src, InstallOptions{})
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]bool{
		filepath.Join(src.Dir, "w", "layouts", "content.css"): true,
		filepath.Join(src.Dir, "p", "layouts", "title.css"):   true,
	}
	if len(res.Lint) == 0 {
		t.Fatal("설치가 CSS 경고를 내지 않았다")
	}
	for _, d := range res.Lint {
		if !want[d.File] {
			t.Errorf("경고가 원본에 없는 파일을 가리킨다: %s", d)
		}
	}

	// 부모가 이미 홈에 있고 자식만 다른 디렉토리에서 설치하면 부모는 이번 설치의 것이 아니다 —
	// 부모의 경고는 홈의 사본을 가리킨다.
	other := t.TempDir()
	files = map[string]string{"x/layouts/content.css": "color: red;"}
	child(files, "x/", "warny2", "parent")
	writeTree(t, other, files)
	src2 := mustLocal(t, other)
	res, err = s.Install(context.Background(), src2, InstallOptions{})
	if err != nil {
		t.Fatal(err)
	}
	want = map[string]bool{
		filepath.Join(src2.Dir, "x", "layouts", "content.css"):           true,
		filepath.Join(s.Dir, "themes", "parent", "layouts", "title.css"): true,
	}
	for _, d := range res.Lint {
		if !want[d.File] {
			t.Errorf("경고가 없는 파일을 가리킨다: %s", d)
		}
	}
	if len(res.Lint) != 2 {
		t.Errorf("자식과 부모의 경고가 하나씩이어야 한다: %v", res.Lint)
	}
}
