package themestore

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/k2sebeom/balpyo/core"
	"github.com/k2sebeom/balpyo/soml"
)

// HomeEnv는 홈 디렉토리를 바꾸는 환경 변수다 (CI · 테스트 · 여러 홈을 나눠 쓰는 사람).
const HomeEnv = "BALPYO_HOME"

// Home은 설치한 테마의 홈이다: HomeEnv가 있으면 그것, 없으면 `~/.balpyo`.
//
// 캐시 디렉토리(`~/.cache`)가 아닌 이유는 그 자리가 "지워도 되는 것"을 뜻한다는 것이다 — 설치한
// 테마를 지우면 그것을 부르는 덱이 빌드되지 않는다.
func Home() (string, error) {
	if dir := os.Getenv(HomeEnv); dir != "" {
		return filepath.Abs(dir)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("홈 디렉토리를 알 수 없다 (%s로 정할 수 있다): %w", HomeEnv, err)
	}
	return filepath.Join(home, ".balpyo"), nil
}

const (
	themesDir = "themes"
	indexFile = "themes.json"
)

// Record는 설치한 테마 하나의 출처다.
type Record struct {
	Name string `json:"name"`
	// Repo는 `github.com/<owner>/<repo>`다 (Repo도 Local도 비면 출처를 모르는 테마 — 손으로 넣은 디렉토리).
	Repo string `json:"repo"`
	// Local은 로컬 디렉토리에서 복사했을 때 그 디렉토리(절대 경로)다. update가 거기서 다시 복사한다.
	Local string `json:"local,omitempty"`
	// Path는 저장소(또는 Local) 안의 테마 디렉토리다 (루트면 ".").
	Path string `json:"path"`
	// Ref는 설치할 때 준 ref다 (비면 기본 브랜치). update가 같은 ref를 다시 받는다.
	Ref string `json:"ref,omitempty"`
	// Commit은 받은 압축이 알려 준 커밋이다.
	Commit string `json:"commit,omitempty"`
}

// HasOrigin은 출처를 아는 기록인지다 — update가 다시 받을 수 있는 테마다.
func (r Record) HasOrigin() bool { return r.Repo != "" || r.Local != "" }

// source는 기록한 출처다 (update가 다시 받는 자리).
func (r Record) source() (Source, error) {
	if r.Local != "" {
		return Source{Dir: r.Local}, nil
	}
	src, err := ParseSource(r.Repo)
	if err != nil {
		return Source{}, err
	}
	src.Ref = r.Ref
	return src, nil
}

// Origin은 사람이 읽는 출처다 (`github.com/o/r/path@ref`, 로컬이면 테마 디렉토리의 경로).
func (r Record) Origin() string {
	if r.Local != "" {
		return filepath.Join(r.Local, filepath.FromSlash(r.Path))
	}
	if r.Repo == "" {
		return ""
	}
	out := r.Repo
	if r.Path != "." && r.Path != "" {
		out += "/" + r.Path
	}
	if r.Ref != "" {
		out += "@" + r.Ref
	}
	return out
}

type index struct {
	Themes []Record `json:"themes"`
}

// Store는 홈 하나다.
type Store struct {
	// Dir은 홈 디렉토리다 (`themes/`와 `themes.json`을 담는다).
	Dir    string
	Client Client
}

// ThemesFS는 core.WithInstalledThemes에 넘길 홈이다.
func (s *Store) ThemesFS() fs.FS { return os.DirFS(s.Dir) }

func (s *Store) themePath(name string) string { return filepath.Join(s.Dir, themesDir, name) }

func (s *Store) readIndex() (map[string]Record, error) {
	data, err := os.ReadFile(filepath.Join(s.Dir, indexFile))
	if errors.Is(err, fs.ErrNotExist) {
		return map[string]Record{}, nil
	}
	if err != nil {
		return nil, err
	}
	var idx index
	if err := json.Unmarshal(data, &idx); err != nil {
		return nil, fmt.Errorf("%s 파싱 실패: %w", filepath.Join(s.Dir, indexFile), err)
	}
	out := map[string]Record{}
	for _, r := range idx.Themes {
		out[r.Name] = r
	}
	return out, nil
}

// writeIndex는 기록을 이름 순으로 쓴다 (파일이 실행마다 같은 바이트여야 diff로 읽힌다). 지금 홈에
// 없는 테마의 기록은 남기지 않는다.
func (s *Store) writeIndex(records map[string]Record) error {
	var idx index
	for name, r := range records {
		if _, err := os.Stat(s.themePath(name)); err == nil {
			idx.Themes = append(idx.Themes, r)
		}
	}
	sort.Slice(idx.Themes, func(i, j int) bool { return idx.Themes[i].Name < idx.Themes[j].Name })
	if idx.Themes == nil {
		idx.Themes = []Record{}
	}
	data, err := json.MarshalIndent(idx, "", "  ")
	if err != nil {
		return err
	}
	tmp := filepath.Join(s.Dir, indexFile+".tmp")
	if err := os.WriteFile(tmp, append(data, '\n'), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, filepath.Join(s.Dir, indexFile))
}

// List는 홈에 있는 테마다 (이름 순). 출처 기록이 없는 디렉토리도 Repo가 빈 기록으로 나온다.
func (s *Store) List() ([]Record, error) {
	records, err := s.readIndex()
	if err != nil {
		return nil, err
	}
	var out []Record
	for _, name := range core.InstalledThemeNames(core.WithInstalledThemes(nil, s.ThemesFS())) {
		r, ok := records[name]
		if !ok {
			r = Record{Name: name}
		}
		out = append(out, r)
	}
	return out, nil
}

// InstallOptions는 설치의 선택이다.
type InstallOptions struct {
	// Names가 비면 찾은 테마를 전부 설치한다. 주면 그 테마와, 같은 저장소 안의 extends 부모를 설치한다.
	Names []string
	// Force는 다른 출처에서 설치한 같은 이름의 테마를 갈아 끼운다.
	Force bool
}

// InstallResult는 설치 한 번의 결과다.
type InstallResult struct {
	Source Source
	Commit string
	// Themes는 설치한 테마의 기록이다 (이름 순).
	Themes []Record
	// Lint는 설치한 테마의 배치 CSS 경고다 — 빌드는 이 검사를 돌리지 않는다.
	Lint soml.Diagnostics
}

// candidate는 받은 저장소에서 찾은 테마 디렉토리 하나다.
type candidate struct {
	dir    string // 저장소 상대 경로 ("."는 루트)
	header core.ThemeHeader
	err    error
}

var themeName = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.-]*$`)

// Install은 src의 저장소를 받아(로컬 디렉토리면 복사해) 테마를 설치한다. 전부 되거나 전부 안 된다: 옮긴 뒤 빌드와 같은
// 로더로 로드해 보고, 하나라도 실패하면 홈을 원래 상태로 되돌린다.
func (s *Store) Install(ctx context.Context, src Source, opts InstallOptions) (InstallResult, error) {
	return s.install(ctx, src, opts.Force, func(cands []candidate) ([]candidate, error) {
		return pickByName(src, cands, opts.Names)
	})
}

// Update는 설치한 테마를 기록한 출처 · ref에서 다시 받는다. names가 비면 출처를 아는 테마 전부다.
// 같은 저장소 · ref의 테마는 한 번에 받는다.
func (s *Store) Update(ctx context.Context, names []string) ([]InstallResult, error) {
	records, err := s.readIndex()
	if err != nil {
		return nil, err
	}
	if len(names) == 0 {
		installed, err := s.List()
		if err != nil {
			return nil, err
		}
		for _, r := range installed {
			if r.HasOrigin() {
				names = append(names, r.Name)
			}
		}
	}
	type group struct {
		src     Source
		records []Record
	}
	groups := map[string]*group{}
	var order []string
	for _, name := range names {
		r, ok := records[name]
		if _, err := os.Stat(s.themePath(name)); err != nil {
			return nil, fmt.Errorf("설치한 테마 %q가 없다", name)
		}
		if !ok || !r.HasOrigin() {
			return nil, fmt.Errorf("테마 %q는 출처 기록이 없어 다시 받을 수 없다 — balpyo theme install로 설치한다", name)
		}
		src, err := r.source()
		if err != nil {
			return nil, err
		}
		key := src.String()
		if groups[key] == nil {
			groups[key] = &group{src: src}
			order = append(order, key)
		}
		groups[key].records = append(groups[key].records, r)
	}
	var results []InstallResult
	for _, key := range order {
		g := groups[key]
		res, err := s.install(ctx, g.src, true, func(cands []candidate) ([]candidate, error) {
			return pickByRecord(g.src, cands, g.records)
		})
		if err != nil {
			return results, err
		}
		results = append(results, res)
	}
	return results, nil
}

// Remove는 설치한 테마를 지운다. 남는 테마가 상속하는 테마는 지우지 않는다 — 지우면 그 테마가
// 로드되지 않는다.
func (s *Store) Remove(names []string) error {
	removing := map[string]bool{}
	for _, name := range names {
		if _, err := os.Stat(s.themePath(name)); err != nil || !themeName.MatchString(name) {
			return fmt.Errorf("설치한 테마 %q가 없다", name)
		}
		removing[name] = true
	}
	home := s.ThemesFS()
	for _, other := range core.InstalledThemeNames(core.WithInstalledThemes(nil, home)) {
		if removing[other] {
			continue
		}
		h, err := core.ReadThemeHeader(home, path.Join(themesDir, other))
		if err == nil && removing[h.Extends] {
			return fmt.Errorf("설치한 테마 %q가 %q를 상속한다 — 함께 지우거나 %q를 먼저 지운다", other, h.Extends, other)
		}
	}
	records, err := s.readIndex()
	if err != nil {
		return err
	}
	for _, name := range names {
		if err := os.RemoveAll(s.themePath(name)); err != nil {
			return err
		}
		delete(records, name)
	}
	return s.writeIndex(records)
}

func (s *Store) install(ctx context.Context, src Source, force bool,
	pick func([]candidate) ([]candidate, error)) (InstallResult, error) {
	res := InstallResult{Source: src}
	if err := os.MkdirAll(filepath.Join(s.Dir, themesDir), 0o755); err != nil {
		return res, err
	}
	// 임시 디렉토리를 홈 안(themes/ 밖)에 두는 이유: 옮기기가 같은 파일 시스템의 rename이어야 한다.
	tmp, err := os.MkdirTemp(s.Dir, ".install-")
	if err != nil {
		return res, err
	}
	defer os.RemoveAll(tmp)

	repoDir := filepath.Join(tmp, "repo")
	var snap snapshot
	if src.IsLocal() {
		snap, err = copyDir(src, repoDir, s.Dir)
	} else {
		snap, err = s.Client.fetch(ctx, src, repoDir)
	}
	if err != nil {
		return res, err
	}
	res.Commit = snap.commit
	cands, err := discover(os.DirFS(repoDir), src)
	if err != nil {
		return res, err
	}
	picked, err := pick(cands)
	if err != nil {
		return res, err
	}
	records, err := s.readIndex()
	if err != nil {
		return res, err
	}
	for _, c := range picked {
		if err := s.checkInstallable(src, c, snap.links, records, force); err != nil {
			return res, err
		}
	}

	// 옮기기: 있던 테마는 임시 디렉토리로 비켜 두고(되돌리기용), 받은 테마를 제자리에 놓는다.
	backup := filepath.Join(tmp, "old")
	if err := os.MkdirAll(backup, 0o755); err != nil {
		return res, err
	}
	var moved, backedUp []string
	rollback := func() {
		for _, name := range moved {
			os.RemoveAll(s.themePath(name))
		}
		for _, name := range backedUp {
			os.Rename(filepath.Join(backup, name), s.themePath(name))
		}
	}
	for _, c := range picked {
		name := c.header.Name
		if _, err := os.Stat(s.themePath(name)); err == nil {
			if err := os.Rename(s.themePath(name), filepath.Join(backup, name)); err != nil {
				rollback()
				return res, err
			}
			backedUp = append(backedUp, name)
		}
		if err := os.Rename(filepath.Join(repoDir, filepath.FromSlash(c.dir)), s.themePath(name)); err != nil {
			rollback()
			return res, err
		}
		moved = append(moved, name)
	}
	for _, c := range picked {
		lint, err := core.CheckInstalledTheme(s.ThemesFS(), c.header.Name)
		if err != nil {
			rollback()
			return res, fmt.Errorf("%s의 테마 %q를 로드할 수 없어 설치하지 않았다: %w",
				src.at(c.dir), c.header.Name, err)
		}
		res.Lint = append(res.Lint, lint...)
	}
	for _, c := range picked {
		r := Record{Name: c.header.Name, Path: c.dir, Ref: src.Ref, Commit: snap.commit}
		if src.IsLocal() {
			r.Local = src.Dir
		} else {
			r.Repo = src.RepoID()
		}
		records[r.Name] = r
		res.Themes = append(res.Themes, r)
	}
	if err := s.writeIndex(records); err != nil {
		rollback()
		return res, err
	}
	sort.Slice(res.Themes, func(i, j int) bool { return res.Themes[i].Name < res.Themes[j].Name })
	return res, nil
}

// checkInstallable은 받은 테마 하나를 홈에 둘 수 있는지 본다.
func (s *Store) checkInstallable(src Source, c candidate, links []string, records map[string]Record, force bool) error {
	name := c.header.Name
	where := src.at(c.dir)
	if !themeName.MatchString(name) {
		return fmt.Errorf("%s의 테마 이름 %q는 디렉토리 이름으로 쓸 수 없다", where, name)
	}
	if core.IsBuiltinTheme(name) {
		return fmt.Errorf("%s의 테마 %q는 내장 테마와 이름이 같아 설치할 수 없다 — 설치할 테마의 이름을 골라 다시 친다", where, name)
	}
	for _, l := range links {
		if insideTheme(c.dir, l) {
			return fmt.Errorf("%s의 테마 %q에 링크 %s가 있다 — 테마는 정규 파일만 담는다", where, name, l)
		}
	}
	if _, err := os.Stat(s.themePath(name)); err != nil || force {
		return nil
	}
	r, ok := records[name]
	switch {
	case !ok || !r.HasOrigin():
		return fmt.Errorf("테마 %q가 이미 홈에 있다 (출처 기록 없음) — 갈아 끼우려면 --force", name)
	case !sameOrigin(r, src, c.dir):
		return fmt.Errorf("테마 %q는 이미 %s에서 설치했다 — 갈아 끼우려면 --force", name, r.Origin())
	}
	return nil
}

// sameOrigin은 기록 r이 src의 테마 디렉토리 dir에서 온 것인지다 — 같은 자리에서 다시 설치하는 것은
// 갈아 끼우기가 아니다.
func sameOrigin(r Record, src Source, dir string) bool {
	if r.Path != dir {
		return false
	}
	if src.IsLocal() {
		return r.Local == src.Dir
	}
	return r.Local == "" && r.Repo == src.RepoID()
}

// insideTheme은 저장소 상대 경로 p가 테마 디렉토리 dir 안인지다. 루트가 테마면 `.`으로 시작하는
// 디렉토리(`.github` 등)는 테마의 일부로 보지 않는다 — 찾기(discover)가 그 디렉토리를 보지 않는 것과 같다.
func insideTheme(dir, p string) bool {
	if dir == "." {
		return !strings.HasPrefix(p, ".")
	}
	return p == dir || strings.HasPrefix(p, dir+"/")
}

// discover는 src.Path 아래에서 theme.json이 있는 디렉토리를 찾는다. `.`으로 시작하는 디렉토리는 보지
// 않고(`.github` 등), 테마 디렉토리 안으로는 내려가지 않는다 — 테마 안의 `assets/`가 다른 테마일 수 없다.
func discover(repo fs.FS, src Source) ([]candidate, error) {
	root := src.Path
	if root == "" {
		root = "."
	}
	if info, err := fs.Stat(repo, root); err != nil || !info.IsDir() {
		return nil, fmt.Errorf("%s에 디렉토리 %s가 없다", src.RepoID(), src.Path)
	}
	var cands []candidate
	err := fs.WalkDir(repo, root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			return nil
		}
		if p != root && strings.HasPrefix(d.Name(), ".") {
			return fs.SkipDir
		}
		if _, err := fs.Stat(repo, path.Join(p, "theme.json")); err != nil {
			return nil
		}
		h, herr := core.ReadThemeHeader(repo, p)
		cands = append(cands, candidate{dir: p, header: h, err: herr})
		return fs.SkipDir
	})
	if err != nil {
		return nil, err
	}
	if len(cands) == 0 {
		return nil, fmt.Errorf("%s에서 테마를 찾지 못했다 — theme.json이 있는 디렉토리가 없다", src)
	}
	return cands, nil
}

// pickByName은 설치할 테마를 고른다. names가 비면 전부, 주면 그것과 같은 저장소의 extends 부모다.
func pickByName(src Source, cands []candidate, names []string) ([]candidate, error) {
	byName := map[string][]candidate{}
	var available []string
	for _, c := range cands {
		if c.err != nil {
			continue
		}
		if byName[c.header.Name] == nil {
			available = append(available, c.header.Name)
		}
		byName[c.header.Name] = append(byName[c.header.Name], c)
	}
	sort.Strings(available)
	one := func(name string) (candidate, error) {
		list := byName[name]
		switch len(list) {
		case 0:
			return candidate{}, fmt.Errorf("%s에 테마 %q가 없다 (있는 테마: %s)", src, name, strings.Join(available, ", "))
		case 1:
			return list[0], nil
		}
		var dirs []string
		for _, c := range list {
			dirs = append(dirs, c.dir)
		}
		return candidate{}, fmt.Errorf("%s에 테마 %q가 여럿이다: %s — 하위 경로를 주소에 붙여 하나를 가리킨다",
			src, name, strings.Join(dirs, ", "))
	}

	var picked []candidate
	if len(names) == 0 {
		for _, c := range cands {
			if c.err != nil {
				return nil, fmt.Errorf("%s: %w", src.at(c.dir), c.err)
			}
		}
		for _, name := range available {
			c, err := one(name)
			if err != nil {
				return nil, err
			}
			picked = append(picked, c)
		}
		return picked, nil
	}
	seen := map[string]bool{}
	queue := append([]string(nil), names...)
	for len(queue) > 0 {
		name := queue[0]
		queue = queue[1:]
		if seen[name] {
			continue
		}
		seen[name] = true
		c, err := one(name)
		if err != nil {
			return nil, err
		}
		picked = append(picked, c)
		// 부모가 같은 저장소에 있으면 함께 온다. 내장 이름의 부모는 내장 테마다 (저장소에 같은 이름의
		// 사본이 있어도 설치할 수 없으므로 따라가지 않는다). 저장소에 없는 부모는 이미 설치되어 있어야
		// 하고, 그 판정은 로드 검사가 한다.
		if p := c.header.Extends; p != "" && !core.IsBuiltinTheme(p) && len(byName[p]) > 0 {
			queue = append(queue, p)
		}
	}
	return picked, nil
}

// pickByRecord는 update가 기록한 디렉토리의 테마를 다시 고른다.
func pickByRecord(src Source, cands []candidate, records []Record) ([]candidate, error) {
	var picked []candidate
	for _, r := range records {
		found := false
		for _, c := range cands {
			if c.dir != r.Path {
				continue
			}
			if c.err != nil {
				return nil, fmt.Errorf("%s: %w", src.at(c.dir), c.err)
			}
			if c.header.Name != r.Name {
				return nil, fmt.Errorf("%s의 테마 이름이 %q에서 %q로 바뀌었다 — 새 이름으로 다시 설치한다",
					src.at(c.dir), r.Name, c.header.Name)
			}
			picked = append(picked, c)
			found = true
		}
		if !found {
			return nil, fmt.Errorf("%s에 테마 %q의 디렉토리 %s가 더 이상 없다", src, r.Name, r.Path)
		}
	}
	return picked, nil
}
