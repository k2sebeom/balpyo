package core

import (
	"errors"
	"fmt"
	"io/fs"
	"path"
)

// 이 파일은 **테마 README**의 자리다 — 테마를 만든 사람이 덱 저자에게 쓰는 글(어떤 덱에 맞는지,
// variant와 그림을 무슨 뜻으로 넣었는지)을 테마 디렉토리의 README.md에서 읽는다.
//
// README는 로더도 렌더러도 읽지 않는다: 테마 파일이 담는 것은 값이고 README가 담는 것은 의도다. 그래서
// 없어도 테마는 올바르고, 빌드는 그 사실을 경고하지 않는다 (설치한 테마의 README는 덱 저자가 고칠 수
// 없다). 파일 이름은 정확히 README.md 하나다 — 대소문자 변형까지 찾으면 어느 파일을 읽었는지가 파일
// 시스템마다 갈린다.

// themeReadme는 테마 디렉토리 안 README의 파일 이름이다.
const themeReadme = "README.md"

// ThemeSource는 이름이 풀린 자리다 — 내장, 프로젝트의 themes/, 설치한 테마의 홈.
type ThemeSource string

const (
	ThemeBuiltin     ThemeSource = "built-in"
	ThemeFromProject ThemeSource = "project"
	ThemeInstalled   ThemeSource = "installed"
)

// ThemeInfo는 테마 하나를 덱 저자에게 소개하는 데 필요한 사실이다.
type ThemeInfo struct {
	Name    string
	Extends string
	Source  ThemeSource
	// Dir는 그 자리의 루트에서 본 테마 디렉토리다 (`themes/<이름>`) — 내장이면 바이너리 안의 경로다.
	Dir string
	// Readme는 README.md의 바이트 그대로다 (HasReadme가 false면 nil).
	Readme    []byte
	HasReadme bool
}

// ReadThemeInfo는 이름을 **빌드와 같은 해석**으로 찾아 매니페스트의 이름·부모와 README를 읽는다.
// fsys는 Resolve가 받는 것과 같은 프로젝트 루트다 (nil이면 내장 테마만, WithInstalledThemes로 감쌌으면
// 설치한 테마도). 테마 전체를 로드하지는 않는다 — 깨진 테마의 README도 읽을 수 있어야 고칠 길을 찾는다.
func ReadThemeInfo(fsys fs.FS, name string) (ThemeInfo, error) {
	if err := checkThemeShadowing(fsys); err != nil {
		return ThemeInfo{}, err
	}
	loc, ok, err := locateTheme(fsys, name)
	if err != nil {
		return ThemeInfo{}, err
	}
	if !ok {
		return ThemeInfo{}, unknownThemeError(fsys, name)
	}
	header, err := ReadThemeHeader(loc.fsys, loc.dir)
	if err != nil {
		return ThemeInfo{}, fmt.Errorf("theme %q: %w", name, err)
	}
	if header.Name != name {
		return ThemeInfo{}, fmt.Errorf("%s/theme.json has name %q — it must match the directory name %q",
			loc.dir, header.Name, name)
	}
	info := ThemeInfo{Name: name, Extends: header.Extends, Source: loc.origin.source(), Dir: loc.dir}
	data, err := readExactFile(loc.fsys, loc.dir, themeReadme)
	switch {
	case err == nil:
		info.Readme, info.HasReadme = data, true
	case !errors.Is(err, fs.ErrNotExist):
		return ThemeInfo{}, fmt.Errorf("theme %q: cannot read %s: %w", name, themeReadme, err)
	}
	return info, nil
}

// readExactFile은 dir/name을 읽되 디렉토리 목록에서 **바이트가 같은 이름**을 확인한 뒤에만 연다 —
// 대소문자를 가리지 않는 파일 시스템(macOS·Windows)에서 readme.md가 README.md로 열리지 않게 한다.
func readExactFile(fsys fs.FS, dir, name string) ([]byte, error) {
	entries, err := fs.ReadDir(fsys, dir)
	if err != nil {
		return nil, err
	}
	for _, e := range entries {
		if e.Name() == name && e.Type().IsRegular() {
			return fs.ReadFile(fsys, path.Join(dir, name))
		}
	}
	return nil, fs.ErrNotExist
}

func (o themeOrigin) source() ThemeSource {
	switch o {
	case originLocal:
		return ThemeFromProject
	case originInstalled:
		return ThemeInstalled
	}
	return ThemeBuiltin
}
