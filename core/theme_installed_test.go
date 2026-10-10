package core

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/k2sebeom/balpyo/soml"
)

// 설치한 테마: 홈(`~/.balpyo`)의 `themes/<이름>/`이 내장·프로젝트 로컬과 같은 어휘에 든다. 겹침은
// 그 이름이 요청될 때만 에러이고, 설치한 테마의 배치 CSS 검사는 빌드가 아니라 설치가 본다.

// installedChild는 minimal을 상속하고 레이아웃 CSS에 외관 프로퍼티를 둔 테마다 — 검사(lint)가
// 경고를 내는 파일을 가진다.
func installedChild(name string) fstest.MapFS {
	return localTheme(name, `{"name":"`+name+`","extends":"minimal"}`, "", "",
		map[string]string{"title.css": "display: grid;\ncolor: red;"})
}

func TestResolveFindsInstalledTheme(t *testing.T) {
	home := standaloneLocal("acme")
	deck, diags := themeDeckIn(t, WithInstalledThemes(fstest.MapFS{}, home), "acme", themeProbe)
	if len(diags) != 0 {
		t.Fatalf("설치한 테마를 골랐는데 진단이 났다: %v", diags)
	}
	if deck.Theme.Name != "acme" {
		t.Errorf("theme = %q, want acme", deck.Theme.Name)
	}
}

func TestThemeNamesIncludeInstalled(t *testing.T) {
	fsys := WithInstalledThemes(standaloneLocal("corp"), mergeFS(standaloneLocal("acme"), standaloneLocal("corp")))
	want := []string{"acme", "corp", "editorial", "ink", "minimal"}
	if got := ThemeNames(fsys); !reflect.DeepEqual(got, want) {
		t.Errorf("ThemeNames = %v, want %v", got, want)
	}
}

// 홈 없이 넘기면 설치한 테마는 보이지 않는다 — 지금까지의 호출자(nil · os.DirFS)는 그대로다.
func TestInstalledThemesNeedAHome(t *testing.T) {
	if got := WithInstalledThemes(fstest.MapFS{}, nil); got == nil {
		t.Fatal("홈이 nil이면 프로젝트 FS를 그대로 돌려줘야 한다")
	}
	if names := InstalledThemeNames(standaloneLocal("acme")); len(names) != 0 {
		t.Errorf("홈 없는 FS에서 설치한 테마가 보인다: %v", names)
	}
}

// 로컬과 설치한 테마가 같은 이름이어도, 그 이름을 요청하지 않는 덱은 빌드된다.
func TestInstalledLocalClashOnlyWhenRequested(t *testing.T) {
	fsys := WithInstalledThemes(standaloneLocal("corp"), standaloneLocal("corp"))
	if _, diags := themeDeckIn(t, fsys, "ink", themeProbe); len(diags) != 0 {
		t.Fatalf("겹친 이름을 요청하지 않았는데 진단이 났다: %v", diags)
	}
	_, diags := themeDeckIn(t, fsys, "corp", themeProbe)
	if len(diags) != 1 || !strings.Contains(diags[0].Message, "and among installed themes") ||
		!strings.Contains(diags[0].Message, "balpyo theme remove corp") {
		t.Fatalf("겹친 이름을 요청하면 고치는 길을 적은 에러 하나여야 한다: %v", diags)
	}
	if err := CheckThemeName(fsys, "corp"); err == nil {
		t.Error("CheckThemeName이 겹친 이름을 받았다")
	}
}

// extends 체인에서 만나는 겹침도 요청이다.
func TestInstalledClashThroughExtends(t *testing.T) {
	project := mergeFS(standaloneLocal("corp"),
		localTheme("corp-dark", `{"name":"corp-dark","extends":"corp"}`, "", "", nil))
	_, diags := themeDeckIn(t, WithInstalledThemes(project, standaloneLocal("corp")), "corp-dark", themeProbe)
	if len(diags) != 1 || !strings.Contains(diags[0].Message, "and among installed themes") {
		t.Fatalf("extends가 겹친 이름에 닿으면 에러여야 한다: %v", diags)
	}
}

func TestInstalledBuiltinClashWhenRequested(t *testing.T) {
	fsys := WithInstalledThemes(fstest.MapFS{}, standaloneLocal("ink"))
	if _, diags := themeDeckIn(t, fsys, "minimal", themeProbe); len(diags) != 0 {
		t.Fatalf("겹치지 않은 이름을 요청했는데 진단이 났다: %v", diags)
	}
	_, diags := themeDeckIn(t, fsys, "ink", themeProbe)
	if len(diags) != 1 || !strings.Contains(diags[0].Message, "has the same name as a built-in theme") {
		t.Fatalf("내장 이름의 설치한 테마를 요청하면 에러여야 한다: %v", diags)
	}
}

// 로컬 테마가 설치한 테마를 상속할 수 있다 — 어휘가 하나다.
func TestLocalThemeExtendsInstalled(t *testing.T) {
	project := localTheme("talk", `{"name":"talk","extends":"acme"}`, "", "", nil)
	deck, diags := themeDeckIn(t, WithInstalledThemes(project, standaloneLocal("acme")), "talk", themeProbe)
	if len(diags) != 0 {
		t.Fatalf("로컬 → 설치한 테마 상속에 진단이 났다: %v", diags)
	}
	if c, _ := deck.Theme.Color("accent1"); c.RGB != "123456" {
		t.Errorf("accent1 = %q, want 설치한 부모의 123456", c.RGB)
	}
}

// 세 자리 어디에도 없는 이름은 그 사실을 말한다.
func TestUnknownThemeSaysNotInstalled(t *testing.T) {
	fsys := WithInstalledThemes(fstest.MapFS{}, standaloneLocal("acme"))
	err := CheckThemeName(fsys, "acmee")
	if err == nil {
		t.Fatal("없는 이름을 받았다")
	}
	for _, want := range []string{`unknown theme "acmee"`, "or an installed theme", "acme, editorial, ink, minimal"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("에러에 %q가 없다: %v", want, err)
		}
	}
}

// 설치한 테마의 배치 CSS 경고는 빌드에서 나지 않고(덱 저자가 고칠 수 없다), 설치 검사에서 난다.
func TestInstalledThemeLintIsAnInstallConcern(t *testing.T) {
	home := installedChild("acme")
	if _, diags := themeDeckIn(t, WithInstalledThemes(fstest.MapFS{}, home), "acme", themeProbe); len(diags) != 0 {
		t.Errorf("빌드가 설치한 테마의 CSS를 검사했다: %v", diags)
	}
	// 같은 파일이 로컬이면 빌드가 본다 — 차이는 자리뿐이다.
	if _, diags := themeDeckIn(t, home, "acme", themeProbe); len(diags) == 0 {
		t.Error("로컬 테마의 CSS 경고가 사라졌다")
	}
	diags, err := CheckInstalledTheme(home, "acme")
	if err != nil {
		t.Fatal(err)
	}
	if len(diags) != 1 || diags[0].Severity != soml.SeverityWarning ||
		diags[0].File != "themes/acme/layouts/title.css" {
		t.Errorf("설치 검사가 CSS 경고 하나를 내야 한다: %v", diags)
	}
}

func TestCheckInstalledThemeRejectsBrokenTheme(t *testing.T) {
	cases := map[string]fstest.MapFS{
		"없는 부모":  localTheme("acme", `{"name":"acme","extends":"nope"}`, "", "", nil),
		"불완전":    localTheme("acme", `{"name":"acme","layouts":["content"]}`, `{"color":[]}`, fullComponents, map[string]string{"content.css": ""}),
		"이름 불일치": localTheme("acme", `{"name":"other","extends":"minimal"}`, "", "", nil),
	}
	for name, home := range cases {
		if _, err := CheckInstalledTheme(home, "acme"); err == nil {
			t.Errorf("%s: 설치 검사가 깨진 테마를 받았다", name)
		}
	}
	if _, err := CheckInstalledTheme(fstest.MapFS{}, "acme"); err == nil {
		t.Error("없는 테마를 받았다")
	}
}

func TestReadThemeHeader(t *testing.T) {
	fsys := localTheme("x", `{"name":"acme-dark","extends":"acme"}`, "", "", nil)
	h, err := ReadThemeHeader(fsys, "themes/x")
	if err != nil {
		t.Fatal(err)
	}
	if h != (ThemeHeader{Name: "acme-dark", Extends: "acme"}) {
		t.Errorf("header = %+v", h)
	}
	if _, err := ReadThemeHeader(localTheme("x", `{"extends":"acme"}`, "", "", nil), "themes/x"); err == nil {
		t.Error("name 없는 매니페스트를 받았다")
	}
}

// 설정 로드가 설치한 테마의 이름을 받는다 (홈을 넘길 때만).
func TestLoadConfigWithInstalledTheme(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ConfigFile), []byte(`{"theme":"acme"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadConfig(dir); err == nil {
		t.Error("홈 없이 설치한 테마의 이름을 받았다")
	}
	cfg, err := LoadConfigWithThemes(dir, standaloneLocal("acme"))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Theme != "acme" {
		t.Errorf("theme = %q", cfg.Theme)
	}
}
