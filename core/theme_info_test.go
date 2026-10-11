package core

import (
	"strings"
	"testing"
	"testing/fstest"
)

// 테마 README: 테마 디렉토리의 README.md를 빌드와 같은 이름 해석으로 찾아 바이트 그대로 돌려준다.

func withReadme(fsys fstest.MapFS, name, readme string) fstest.MapFS {
	fsys["themes/"+name+"/README.md"] = &fstest.MapFile{Data: []byte(readme)}
	return fsys
}

func TestThemeInfoReadsTheReadmeFromEachPlace(t *testing.T) {
	project := withReadme(installedChild("corp"), "corp", "# corp\nproject theme\n")
	home := withReadme(standaloneLocal("acme"), "acme", "# acme\n")
	fsys := WithInstalledThemes(project, home)

	cases := []struct {
		name, extends string
		source        ThemeSource
		readme        string
	}{
		{"corp", "minimal", ThemeFromProject, "# corp\nproject theme\n"},
		{"acme", "", ThemeInstalled, "# acme\n"},
	}
	for _, c := range cases {
		info, err := ReadThemeInfo(fsys, c.name)
		if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		if info.Name != c.name || info.Extends != c.extends || info.Source != c.source ||
			info.Dir != "themes/"+c.name || string(info.Readme) != c.readme || !info.HasReadme {
			t.Errorf("%s: got %+v", c.name, info)
		}
	}
}

func TestThemeInfoWithoutReadme(t *testing.T) {
	info, err := ReadThemeInfo(standaloneLocal("acme"), "acme")
	if err != nil {
		t.Fatal(err)
	}
	if info.HasReadme || info.Readme != nil {
		t.Errorf("README가 없는데 있다고 했다: %+v", info)
	}
}

// README는 정확히 README.md다 — 대소문자 변형은 찾지 않는다.
func TestThemeInfoReadsOnlyReadmeMd(t *testing.T) {
	fsys := standaloneLocal("acme")
	fsys["themes/acme/readme.md"] = &fstest.MapFile{Data: []byte("lowercase")}
	info, err := ReadThemeInfo(fsys, "acme")
	if err != nil {
		t.Fatal(err)
	}
	if info.HasReadme {
		t.Errorf("readme.md를 README로 읽었다: %q", info.Readme)
	}
}

func TestThemeInfoBuiltinThemesHaveReadmes(t *testing.T) {
	for _, name := range BuiltinThemes() {
		info, err := ReadThemeInfo(nil, name)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if info.Source != ThemeBuiltin || !info.HasReadme || !strings.HasPrefix(string(info.Readme), "# ") {
			t.Errorf("%s: 내장 테마에 README.md(# 제목으로 시작)가 없다: %+v", name, info.Source)
		}
	}
}

func TestThemeInfoErrors(t *testing.T) {
	broken := localTheme("corp", `{"name":"other"}`, "", "", nil)
	cases := []struct {
		fsys fstest.MapFS
		name string
		want string
	}{
		{fstest.MapFS{}, "nope", `unknown theme "nope"`},
		{broken, "corp", `themes/corp/theme.json has name "other" — it must match the directory name "corp"`},
		{localTheme("minimal", `{"name":"minimal"}`, "", "", nil), "minimal", `same name as the built-in theme "minimal"`},
	}
	for _, c := range cases {
		_, err := ReadThemeInfo(c.fsys, c.name)
		if err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: err = %v, want %q", c.name, err, c.want)
		}
	}
}
