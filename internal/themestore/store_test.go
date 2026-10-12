package themestore

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"sync"
	"testing"

	"github.com/k2sebeom/balpyo/core"
)

// 설치의 계약: 이름은 테마의 theme.json이 정하고, 한 저장소의 테마 여럿은 이름을 주지 않으면 전부,
// 주면 그것과 같은 저장소의 부모가 온다. 설치는 전부 되거나 전부 안 된다.

// repo는 가짜 GitHub의 저장소 하나다 (ref → 내용).
type repo struct {
	files  map[string]string // 저장소 상대 경로 → 내용
	links  map[string]string // 저장소 상대 경로 → 링크 대상
	raw    []string          // 저장소 밖을 가리키는 원문 경로 (압축에 그대로 싣는다)
	commit string
}

// fakeGitHub는 codeload 모양(/owner/repo/tar.gz/ref)으로 tarball을 내는 서버다.
type fakeGitHub struct {
	mu    sync.Mutex
	repos map[string]repo // "owner/repo@ref"
	hits  []string
}

func (g *fakeGitHub) set(key string, r repo) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.repos[key] = r
}

func (g *fakeGitHub) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	parts := strings.SplitN(strings.TrimPrefix(req.URL.Path, "/"), "/", 4)
	if len(parts) != 4 || parts[2] != "tar.gz" {
		http.NotFound(w, req)
		return
	}
	g.mu.Lock()
	key := parts[0] + "/" + parts[1] + "@" + parts[3]
	g.hits = append(g.hits, key)
	r, ok := g.repos[key]
	g.mu.Unlock()
	if !ok {
		http.NotFound(w, req)
		return
	}
	w.Write(tarball(parts[1]+"-"+parts[3], r))
}

func tarball(top string, r repo) []byte {
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	if r.commit != "" {
		tw.WriteHeader(&tar.Header{Typeflag: tar.TypeXGlobalHeader, Name: "pax_global_header",
			PAXRecords: map[string]string{"comment": r.commit}, Format: tar.FormatPAX})
	}
	tw.WriteHeader(&tar.Header{Typeflag: tar.TypeDir, Name: top + "/", Mode: 0o755})
	names := make([]string, 0, len(r.files))
	for name := range r.files {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		data := r.files[name]
		tw.WriteHeader(&tar.Header{Typeflag: tar.TypeReg, Name: top + "/" + name, Mode: 0o644, Size: int64(len(data))})
		tw.Write([]byte(data))
	}
	for name, target := range r.links {
		tw.WriteHeader(&tar.Header{Typeflag: tar.TypeSymlink, Name: top + "/" + name, Linkname: target})
	}
	for _, name := range r.raw {
		tw.WriteHeader(&tar.Header{Typeflag: tar.TypeReg, Name: name, Mode: 0o644, Size: 1})
		tw.Write([]byte("x"))
	}
	tw.Close()
	gz.Close()
	return buf.Bytes()
}

// child는 dir에 minimal(또는 parent)을 상속한 테마를 둔다 — 부분 파일이어도 되므로 매니페스트 하나다.
func child(files map[string]string, dir, name, parent string) {
	if parent == "" {
		parent = "minimal"
	}
	files[dir+"theme.json"] = `{"name":"` + name + `","extends":"` + parent + `"}`
}

func newStore(t *testing.T) (*Store, *fakeGitHub) {
	t.Helper()
	gh := &fakeGitHub{repos: map[string]repo{}}
	srv := httptest.NewServer(gh)
	t.Cleanup(srv.Close)
	return &Store{Dir: filepath.Join(t.TempDir(), ".balpyo"), Client: Client{BaseURL: srv.URL}}, gh
}

func mustSource(t *testing.T, raw string) Source {
	t.Helper()
	src, err := ParseSource(raw)
	if err != nil {
		t.Fatal(err)
	}
	return src
}

func installedNames(t *testing.T, s *Store) []string {
	t.Helper()
	list, err := s.List()
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, r := range list {
		names = append(names, r.Name)
	}
	return names
}

func TestParseSource(t *testing.T) {
	cases := map[string]Source{
		"github.com/acme/themes":                         {Owner: "acme", Repo: "themes"},
		"https://github.com/acme/themes.git":             {Owner: "acme", Repo: "themes"},
		"https://www.github.com/acme/themes/":            {Owner: "acme", Repo: "themes"},
		"github.com/acme/themes/brand/dark@v1.2.0":       {Owner: "acme", Repo: "themes", Path: "brand/dark", Ref: "v1.2.0"},
		"https://github.com/acme/themes/tree/main/brand": {Owner: "acme", Repo: "themes", Path: "brand", Ref: "main"},
		"git@github.com:acme/themes.git":                 {Owner: "acme", Repo: "themes"},
		"github.com/acme/themes@feature/x":               {Owner: "acme", Repo: "themes", Ref: "feature/x"},
	}
	for raw, want := range cases {
		got, err := ParseSource(raw)
		if err != nil {
			t.Errorf("%s: %v", raw, err)
			continue
		}
		if got != want {
			t.Errorf("%s: got %+v, want %+v", raw, got, want)
		}
	}
	for _, raw := range []string{
		"acme/themes", "https://gitlab.com/acme/themes", "github.com/acme", "github.com/acme/themes@",
		"github.com/acme/themes/blob/main/theme.json", "github.com/acme/themes/tree/main/x@v1",
		"github.com/acme/themes/../x", "github.com/acme/themes@a..b",
	} {
		if _, err := ParseSource(raw); err == nil {
			t.Errorf("%s: 받으면 안 되는 주소를 받았다", raw)
		}
	}
	if s := mustSource(t, "github.com/acme/themes/brand@v1").String(); s != "github.com/acme/themes/brand@v1" {
		t.Errorf("String = %q", s)
	}
}

// 저장소 루트의 테마 하나: 이름은 저장소가 아니라 theme.json이 정하고, 출처가 기록된다.
func TestInstallRootTheme(t *testing.T) {
	s, gh := newStore(t)
	files := map[string]string{"README.md": "# acme"}
	child(files, "", "acme", "")
	gh.set("acme/slide-theme@HEAD", repo{files: files, commit: "abc123"})

	res, err := s.Install(context.Background(), mustSource(t, "github.com/acme/slide-theme"), InstallOptions{})
	if err != nil {
		t.Fatal(err)
	}
	want := []Record{{Name: "acme", Repo: "github.com/acme/slide-theme", Path: ".", Commit: "abc123"}}
	if !reflect.DeepEqual(res.Themes, want) {
		t.Errorf("themes = %+v, want %+v", res.Themes, want)
	}
	list, _ := s.List()
	if !reflect.DeepEqual(list, want) {
		t.Errorf("list = %+v, want %+v", list, want)
	}
	if err := core.CheckThemeName(core.WithInstalledThemes(nil, s.ThemesFS()), "acme"); err != nil {
		t.Errorf("설치한 테마가 어휘에 없다: %v", err)
	}
}

// 한 저장소의 테마 여럿: 이름을 주지 않으면 전부다. `.` 디렉토리는 보지 않고 테마 안으로 내려가지 않는다.
func TestInstallAllThemesOfARepo(t *testing.T) {
	s, gh := newStore(t)
	files := map[string]string{}
	child(files, "themes/light/", "acme", "")
	child(files, "themes/dark/", "acme-dark", "acme")
	child(files, "brand/", "acme-print", "")
	child(files, ".github/fixture/", "ignored", "")
	child(files, "brand/assets/nested/", "nested", "")
	gh.set("acme/themes@v1", repo{files: files, commit: "c1"})

	res, err := s.Install(context.Background(), mustSource(t, "github.com/acme/themes@v1"), InstallOptions{})
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, r := range res.Themes {
		got = append(got, r.Name+"="+r.Path)
	}
	want := []string{"acme=themes/light", "acme-dark=themes/dark", "acme-print=brand"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("installed = %v, want %v", got, want)
	}
}

// 이름을 주면 그 테마와 같은 저장소의 부모가 온다 (내장 부모는 따라가지 않는다).
func TestInstallByNamePullsParents(t *testing.T) {
	s, gh := newStore(t)
	files := map[string]string{}
	child(files, "light/", "acme", "")
	child(files, "dark/", "acme-dark", "acme")
	child(files, "other/", "other", "")
	gh.set("acme/themes@HEAD", repo{files: files})

	if _, err := s.Install(context.Background(), mustSource(t, "github.com/acme/themes"),
		InstallOptions{Names: []string{"acme-dark"}}); err != nil {
		t.Fatal(err)
	}
	if got := installedNames(t, s); !reflect.DeepEqual(got, []string{"acme", "acme-dark"}) {
		t.Errorf("installed = %v", got)
	}
	_, err := s.Install(context.Background(), mustSource(t, "github.com/acme/themes"),
		InstallOptions{Names: []string{"nope"}})
	if err == nil || !strings.Contains(err.Error(), "themes found: acme, acme-dark, other") {
		t.Errorf("없는 이름은 있는 테마를 나열해야 한다: %v", err)
	}
}

// 하위 경로는 찾는 범위를 좁힌다.
func TestInstallSubpath(t *testing.T) {
	s, gh := newStore(t)
	files := map[string]string{}
	child(files, "a/", "alpha", "")
	child(files, "b/", "beta", "")
	gh.set("acme/themes@main", repo{files: files})
	if _, err := s.Install(context.Background(), mustSource(t, "https://github.com/acme/themes/tree/main/b"),
		InstallOptions{}); err != nil {
		t.Fatal(err)
	}
	if got := installedNames(t, s); !reflect.DeepEqual(got, []string{"beta"}) {
		t.Errorf("installed = %v", got)
	}
	if _, err := s.Install(context.Background(), mustSource(t, "github.com/acme/themes/zzz@main"),
		InstallOptions{}); err == nil {
		t.Error("없는 하위 경로를 받았다")
	}
}

// 저장소 안에서 같은 이름이 두 번 나오면 두 경로를 적는다. 그 이름을 고르지 않으면 상관없다.
func TestInstallDuplicateNames(t *testing.T) {
	s, gh := newStore(t)
	files := map[string]string{}
	child(files, "v1/", "acme", "")
	child(files, "v2/", "acme", "")
	child(files, "x/", "other", "")
	gh.set("acme/themes@HEAD", repo{files: files})
	src := mustSource(t, "github.com/acme/themes")
	_, err := s.Install(context.Background(), src, InstallOptions{})
	if err == nil || !strings.Contains(err.Error(), "v1, v2") {
		t.Fatalf("중복 이름은 두 경로를 적은 에러여야 한다: %v", err)
	}
	if _, err := s.Install(context.Background(), src, InstallOptions{Names: []string{"other"}}); err != nil {
		t.Errorf("중복이 아닌 이름을 고르면 설치되어야 한다: %v", err)
	}
}

// 내장 이름을 가진 테마를 담은 저장소(이 저장소처럼): 전부는 에러, 이름을 고르면 된다.
func TestInstallRepoWithBuiltinNames(t *testing.T) {
	s, gh := newStore(t)
	files := map[string]string{}
	child(files, "core/themes/minimal/", "minimal", "ink")
	child(files, "examples/themes/harbor/", "harbor", "")
	gh.set("k2sebeom/balpyo@HEAD", repo{files: files})
	src := mustSource(t, "github.com/k2sebeom/balpyo")
	_, err := s.Install(context.Background(), src, InstallOptions{})
	if err == nil || !strings.Contains(err.Error(), "same name as a built-in theme") {
		t.Fatalf("내장 이름은 설치 에러여야 한다: %v", err)
	}
	if names := installedNames(t, s); len(names) != 0 {
		t.Errorf("실패한 설치가 테마를 남겼다: %v", names)
	}
	if _, err := s.Install(context.Background(), src, InstallOptions{Names: []string{"harbor"}}); err != nil {
		t.Errorf("이름을 고르면 설치되어야 한다: %v", err)
	}
}

// 로드되지 않는 테마는 설치되지 않고, 있던 테마는 그대로 남는다.
func TestInstallRollsBackOnBrokenTheme(t *testing.T) {
	s, gh := newStore(t)
	good := map[string]string{"layouts/title.css": "/* v1 */"}
	child(good, "", "acme", "")
	gh.set("acme/theme@v1", repo{files: good})
	src := mustSource(t, "github.com/acme/theme@v1")
	if _, err := s.Install(context.Background(), src, InstallOptions{}); err != nil {
		t.Fatal(err)
	}
	broken := map[string]string{}
	child(broken, "", "acme", "nope")
	gh.set("acme/theme@v1", repo{files: broken})
	_, err := s.Install(context.Background(), src, InstallOptions{})
	if err == nil || !strings.Contains(err.Error(), "nothing was installed") {
		t.Fatalf("깨진 테마는 설치 에러여야 한다: %v", err)
	}
	css, err := os.ReadFile(filepath.Join(s.Dir, "themes", "acme", "layouts", "title.css"))
	if err != nil || string(css) != "/* v1 */" {
		t.Errorf("되돌리기가 있던 테마를 지켰어야 한다: %q %v", css, err)
	}
	entries, _ := os.ReadDir(s.Dir)
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".install-") {
			t.Errorf("임시 디렉토리가 남았다: %s", e.Name())
		}
	}
}

// 다른 출처의 같은 이름은 --force 없이 갈아 끼우지 않는다.
func TestInstallConflictNeedsForce(t *testing.T) {
	s, gh := newStore(t)
	files := map[string]string{}
	child(files, "", "acme", "")
	gh.set("acme/theme@HEAD", repo{files: files})
	gh.set("fork/theme@HEAD", repo{files: files})
	if _, err := s.Install(context.Background(), mustSource(t, "github.com/acme/theme"), InstallOptions{}); err != nil {
		t.Fatal(err)
	}
	fork := mustSource(t, "github.com/fork/theme")
	_, err := s.Install(context.Background(), fork, InstallOptions{})
	if err == nil || !strings.Contains(err.Error(), "installed from github.com/acme/theme") {
		t.Fatalf("다른 출처의 같은 이름은 에러여야 한다: %v", err)
	}
	if _, err := s.Install(context.Background(), fork, InstallOptions{Force: true}); err != nil {
		t.Fatal(err)
	}
	list, _ := s.List()
	if list[0].Repo != "github.com/fork/theme" {
		t.Errorf("--force가 출처를 갈아 끼우지 않았다: %+v", list)
	}
}

func TestInstallRejectsLinksAndEscapes(t *testing.T) {
	s, gh := newStore(t)
	files := map[string]string{}
	child(files, "theme/", "acme", "")
	gh.set("acme/links@HEAD", repo{files: files, links: map[string]string{"theme/assets/bg.png": "/etc/passwd"}})
	if _, err := s.Install(context.Background(), mustSource(t, "github.com/acme/links"), InstallOptions{}); err == nil ||
		!strings.Contains(err.Error(), "link") {
		t.Errorf("테마 안의 링크는 에러여야 한다: %v", err)
	}
	// 테마 밖의 링크는 상관없다.
	gh.set("acme/outside@HEAD", repo{files: files, links: map[string]string{"docs/x": "../y"}})
	if _, err := s.Install(context.Background(), mustSource(t, "github.com/acme/outside"), InstallOptions{}); err != nil {
		t.Errorf("테마 밖의 링크는 설치를 막지 않는다: %v", err)
	}
	gh.set("acme/escape@HEAD", repo{files: files, raw: []string{"top/../../evil"}})
	if _, err := s.Install(context.Background(), mustSource(t, "github.com/acme/escape"), InstallOptions{}); err == nil {
		t.Error("저장소 밖을 가리키는 경로를 받았다")
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(s.Dir), "evil")); err == nil {
		t.Error("저장소 밖에 파일이 생겼다")
	}
}

func TestInstallReportsLint(t *testing.T) {
	s, gh := newStore(t)
	files := map[string]string{"layouts/title.css": "color: red;"}
	child(files, "", "acme", "")
	gh.set("acme/theme@HEAD", repo{files: files})
	res, err := s.Install(context.Background(), mustSource(t, "github.com/acme/theme"), InstallOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Lint) != 1 {
		t.Fatalf("설치가 CSS 경고 하나를 내야 한다: %v", res.Lint)
	}
	// 경고는 고칠 수 있는 원본 파일(저장소 안의 자리)을 가리킨다 — 설치 홈의 이름 기준 경로가 아니다.
	if got, want := res.Lint[0].File, "github.com/acme/theme/layouts/title.css"; got != want {
		t.Errorf("경고 파일 = %q, want %q", got, want)
	}
}

func TestInstallMissingRepo(t *testing.T) {
	s, _ := newStore(t)
	_, err := s.Install(context.Background(), mustSource(t, "github.com/acme/nothing"), InstallOptions{})
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Errorf("없는 저장소는 찾을 수 없다는 에러여야 한다: %v", err)
	}
}

func TestRemove(t *testing.T) {
	s, gh := newStore(t)
	files := map[string]string{}
	child(files, "light/", "acme", "")
	child(files, "dark/", "acme-dark", "acme")
	gh.set("acme/themes@HEAD", repo{files: files})
	if _, err := s.Install(context.Background(), mustSource(t, "github.com/acme/themes"), InstallOptions{}); err != nil {
		t.Fatal(err)
	}
	if err := s.Remove([]string{"acme"}); err == nil || !strings.Contains(err.Error(), `"acme-dark" extends "acme"`) {
		t.Fatalf("남는 테마가 상속하는 테마는 지우지 않아야 한다: %v", err)
	}
	if err := s.Remove([]string{"acme", "acme-dark"}); err != nil {
		t.Fatal(err)
	}
	if names := installedNames(t, s); len(names) != 0 {
		t.Errorf("지운 테마가 남았다: %v", names)
	}
	if err := s.Remove([]string{"acme"}); err == nil {
		t.Error("없는 테마를 지웠다")
	}
	data, _ := os.ReadFile(filepath.Join(s.Dir, "themes.json"))
	if strings.Contains(string(data), "acme") {
		t.Errorf("기록이 남았다: %s", data)
	}
}

// update는 기록한 출처 · ref를 다시 받는다 — 같은 저장소의 테마는 한 번에 받는다.
func TestUpdate(t *testing.T) {
	s, gh := newStore(t)
	files := map[string]string{"light/layouts/title.css": "/* v1 */"}
	child(files, "light/", "acme", "")
	child(files, "dark/", "acme-dark", "acme")
	gh.set("acme/themes@main", repo{files: files, commit: "c1"})
	if _, err := s.Install(context.Background(), mustSource(t, "github.com/acme/themes@main"), InstallOptions{}); err != nil {
		t.Fatal(err)
	}
	files["light/layouts/title.css"] = "/* v2 */"
	gh.set("acme/themes@main", repo{files: files, commit: "c2"})
	gh.hits = nil
	results, err := s.Update(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || len(gh.hits) != 1 {
		t.Errorf("한 저장소는 한 번 받아야 한다: results=%d hits=%v", len(results), gh.hits)
	}
	css, _ := os.ReadFile(filepath.Join(s.Dir, "themes", "acme", "layouts", "title.css"))
	if string(css) != "/* v2 */" {
		t.Errorf("update가 새 내용을 받지 않았다: %q", css)
	}
	list, _ := s.List()
	for _, r := range list {
		if r.Commit != "c2" || r.Ref != "main" {
			t.Errorf("기록이 갱신되지 않았다: %+v", r)
		}
	}
	if _, err := s.Update(context.Background(), []string{"nope"}); err == nil {
		t.Error("없는 테마를 갱신했다")
	}
}
