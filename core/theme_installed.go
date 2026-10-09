package core

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"path"

	"github.com/k2sebeom/balpyo/soml"
)

// 이 파일은 **설치한 테마**의 자리다 — `balpyo theme install`이 홈(`~/.balpyo`)의 `themes/<이름>/`에
// 놓은 테마를 내장·프로젝트 로컬과 같은 어휘로 보이게 한다.
//
// 홈의 구조는 내장 테마의 임베드 FS · 프로젝트 루트와 같다 (`themes/<이름>/`). 그래서 로더는 세 번째
// 자리를 새 코드 없이 읽는다 — 다른 것은 "어느 FS인가"뿐이다.

// projectFS는 프로젝트 루트에 설치한 테마의 홈을 붙인 파일 시스템이다.
//
// 파일 열기는 전부 프로젝트 루트로 간다: 슬라이드 수집·에셋 해석·로컬 테마는 지금처럼 프로젝트 상대
// 경로를 읽는다. 홈은 테마 이름을 찾는 자리(locateTheme)만 본다. 인자 하나에 싣는 이유는
// Resolve·ResolveAssets의 시그니처가 이미 "프로젝트 루트 하나"이고, 설치한 테마를 몰라도 되는 호출자
// (테스트·프리뷰)가 nil이나 os.DirFS를 그대로 넘길 수 있어야 한다는 것이다.
type projectFS struct {
	root      fs.FS
	installed fs.FS
}

// WithInstalledThemes는 프로젝트 루트(project)에 설치한 테마의 홈(home)을 붙인다. home이 nil이면
// project를 그대로 돌려준다. project가 nil이면 프로젝트 파일은 없고 내장·설치한 테마만 보인다.
func WithInstalledThemes(project, home fs.FS) fs.FS {
	if home == nil {
		return project
	}
	return &projectFS{root: project, installed: home}
}

func (p *projectFS) Open(name string) (fs.File, error) {
	if p.root == nil {
		return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrNotExist}
	}
	return p.root.Open(name)
}

// installedFS는 fsys에 붙은 설치한 테마의 홈이다 (없으면 nil).
func installedFS(fsys fs.FS) fs.FS {
	if p, ok := fsys.(*projectFS); ok {
		return p.installed
	}
	return nil
}

// InstalledThemeNames는 fsys에 붙은 홈에 설치한 테마 이름이다 (정렬). 홈이 없거나 비었으면 빈 목록이다.
func InstalledThemeNames(fsys fs.FS) []string {
	return themeDirNames(installedFS(fsys))
}

// ThemeHeader는 테마 매니페스트에서 설치가 알아야 하는 두 필드다 — 부를 이름과 상속하는 부모.
type ThemeHeader struct {
	Name    string
	Extends string
}

// ReadThemeHeader는 fsys의 dir/theme.json에서 이름과 부모를 읽는다. 받은 저장소에서 테마를 찾는
// 쪽(설치)이 쓴다 — 매니페스트의 모양을 아는 자리가 이 패키지 하나여야 한다.
func ReadThemeHeader(fsys fs.FS, dir string) (ThemeHeader, error) {
	data, err := fs.ReadFile(fsys, path.Join(dir, "theme.json"))
	if err != nil {
		return ThemeHeader{}, fmt.Errorf("theme.json을 읽을 수 없다: %w", err)
	}
	var m themeManifest
	if err := json.Unmarshal(data, &m); err != nil {
		return ThemeHeader{}, fmt.Errorf("%s/theme.json 파싱 실패: %w", dir, err)
	}
	if m.Name == "" {
		return ThemeHeader{}, errors.New(dir + "/theme.json에 name이 없다")
	}
	return ThemeHeader{Name: m.Name, Extends: m.Extends}, nil
}

// CheckInstalledTheme은 홈에 설치한 테마 하나를 **빌드와 같은 로더로** 로드해 본다 — 완전성·`extends`
// 체인·이름이 맞지 않으면 에러다. 돌려주는 진단은 설치한 테마의 배치 CSS 검사(lint)다: 빌드는 그
// 검사를 돌리지 않으므로(덱 저자가 고칠 수 없는 파일이다) 설치하는 사람이 보는 자리가 여기다.
func CheckInstalledTheme(home fs.FS, name string) (soml.Diagnostics, error) {
	fsys := WithInstalledThemes(nil, home)
	loc, ok, err := locateTheme(fsys, name)
	if err != nil {
		return nil, err
	}
	if !ok || loc.origin != originInstalled {
		return nil, fmt.Errorf("설치한 테마 %q가 없다", name)
	}
	chain, err := readThemeChain(fsys, name)
	if err != nil {
		return nil, err
	}
	merged := chain[0]
	for _, child := range chain[1:] {
		merged = mergeRawTheme(merged, child)
	}
	if _, err := buildThemePackage(merged); err != nil {
		return nil, fmt.Errorf("테마 %q: %w", name, err)
	}
	return lintThemeChain(chain, originInstalled), nil
}
