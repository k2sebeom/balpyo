// Package themestore는 설치한 테마의 홈(`~/.balpyo`)이다 — GitHub 저장소에서 테마를 받아
// `themes/<이름>/`에 두고, 어디서 왔는지 `themes.json`에 적는다.
//
// 빌드는 이 패키지를 쓰지 않는다: 설치한 테마는 홈의 디렉토리일 뿐이고, 그것을 이름의 어휘에 넣는
// 것은 core(WithInstalledThemes)다. 네트워크를 쓰는 자리가 사람이 친 명령(`balpyo theme install`)
// 하나여야 빌드가 오프라인에서도 돌고 같은 입력에 같은 바이트를 낸다.
package themestore

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Source는 테마를 받을 자리다 — GitHub 저장소의 한 자리이거나, 이 기계의 디렉토리(Dir)다.
type Source struct {
	Owner string
	Repo  string
	// Path는 저장소 안의 하위 디렉토리다 (슬래시 구분, 비면 저장소 루트). 테마는 이 아래에서 찾는다.
	Path string
	// Ref는 브랜치 · 태그 · 커밋이다 (비면 저장소의 기본 브랜치).
	Ref string
	// Dir이 있으면 GitHub이 아니라 이 기계의 디렉토리(절대 경로)에서 복사한다. 그때 나머지 필드는 비어 있다.
	Dir string
}

// IsLocal은 이 기계의 디렉토리에서 복사하는 출처인지다.
func (s Source) IsLocal() bool { return s.Dir != "" }

// RepoID는 출처 기록에 적는 저장소 이름이다 (`github.com/<owner>/<repo>`, 로컬이면 디렉토리).
func (s Source) RepoID() string {
	if s.IsLocal() {
		return s.Dir
	}
	return "github.com/" + s.Owner + "/" + s.Repo
}

// at은 출처 안의 테마 디렉토리 dir(슬래시 구분, 루트는 ".")을 사람이 읽는 모양으로 적는다.
func (s Source) at(dir string) string {
	if s.IsLocal() {
		return filepath.Join(s.Dir, filepath.FromSlash(dir))
	}
	if dir == "." || dir == "" {
		return s.RepoID()
	}
	return s.RepoID() + "/" + dir
}

// String은 사람이 다시 칠 수 있는 모양이다 (`github.com/o/r/path@ref`, 로컬이면 디렉토리).
func (s Source) String() string {
	if s.IsLocal() {
		return s.Dir
	}
	out := s.RepoID()
	if s.Path != "" {
		out += "/" + s.Path
	}
	if s.Ref != "" {
		out += "@" + s.Ref
	}
	return out
}

var (
	namePart = regexp.MustCompile(`^[A-Za-z0-9_.-]+$`)
	refChars = regexp.MustCompile(`^[A-Za-z0-9_./-]+$`)
)

// ParseSource는 사람이 붙여 넣는 GitHub 주소를 읽는다. 받는 모양:
//
//	github.com/acme/themes                        저장소 루트, 기본 브랜치
//	https://github.com/acme/themes.git            스킴 · .git은 떼어 낸다
//	github.com/acme/themes/brand/dark@v1.2.0      하위 경로와 ref
//	https://github.com/acme/themes/tree/v1/brand  브라우저 주소창의 모양
//	git@github.com:acme/themes.git                SSH 모양
//
// ref를 `/tree/<ref>/…`로 줄 때 ref에 `/`가 있으면 경로와 갈라 읽을 수 없다 — 그때는 `@ref`를 쓴다.
func ParseSource(raw string) (Source, error) {
	s := strings.TrimSpace(raw)
	if rest, ok := strings.CutPrefix(s, "git@github.com:"); ok {
		s = "github.com/" + rest
	}
	for _, scheme := range []string{"https://", "http://"} {
		s = strings.TrimPrefix(s, scheme)
	}
	s = strings.TrimPrefix(s, "www.")
	rest, ok := strings.CutPrefix(s, "github.com/")
	if !ok {
		return Source{}, fmt.Errorf("not a GitHub repository address: %q (e.g. github.com/<owner>/<repo>)", raw)
	}

	var src Source
	if i := strings.LastIndex(rest, "@"); i >= 0 {
		src.Ref, rest = rest[i+1:], rest[:i]
		if src.Ref == "" {
			return Source{}, fmt.Errorf("%q: no ref after @", raw)
		}
	}
	parts := strings.Split(strings.Trim(rest, "/"), "/")
	if len(parts) < 2 || parts[0] == "" || parts[1] == "" {
		return Source{}, fmt.Errorf("%q: the repository address must have both owner and repo", raw)
	}
	src.Owner, src.Repo = parts[0], strings.TrimSuffix(parts[1], ".git")
	if !namePart.MatchString(src.Owner) || !namePart.MatchString(src.Repo) {
		return Source{}, fmt.Errorf("%q: owner or repo name contains invalid characters", raw)
	}

	tail := parts[2:]
	if len(tail) > 0 && tail[0] == "blob" {
		return Source{}, fmt.Errorf("%q points to a file — it must point to a theme directory or a repository", raw)
	}
	if len(tail) > 0 && tail[0] == "tree" {
		if len(tail) < 2 {
			return Source{}, fmt.Errorf("%q: no ref after /tree/", raw)
		}
		if src.Ref != "" {
			return Source{}, fmt.Errorf("%q: ref given twice, with /tree/ and @", raw)
		}
		src.Ref, tail = tail[1], tail[2:]
	}
	for _, seg := range tail {
		if seg == "" || seg == "." || seg == ".." {
			return Source{}, fmt.Errorf("%q: subpath contains an empty name or . or ..", raw)
		}
	}
	src.Path = strings.Join(tail, "/")
	if src.Ref != "" && (!refChars.MatchString(src.Ref) || strings.Contains(src.Ref, "..")) {
		return Source{}, fmt.Errorf("%q: ref %q contains invalid characters", raw, src.Ref)
	}
	return src, nil
}

// ResolveSource는 `balpyo theme install`이 받는 인자를 읽는다 — GitHub 주소이거나 이 기계의 디렉토리다.
//
//   - `/`·`./`·`../`·`~/`로 시작하거나 `.`·`..`·`~`이면 언제나 경로다.
//   - 그 밖에는 GitHub 주소로 읽고, 주소가 아니면 있는 디렉토리일 때 경로다 (`my-themes`).
//
// 경로가 먼저인 모양을 두는 이유: `github.com/acme/x`라는 디렉토리가 우연히 있어도 주소의 뜻이 바뀌지
// 않아야 하고, 경로를 뜻하는 사람은 `./`를 붙여 그 뜻을 못박을 수 있어야 한다.
func ResolveSource(raw string) (Source, error) {
	s := strings.TrimSpace(raw)
	if looksLikePath(s) {
		return LocalSource(s)
	}
	src, err := ParseSource(s)
	if err == nil {
		return src, nil
	}
	if info, serr := os.Stat(s); serr == nil && info.IsDir() {
		return LocalSource(s)
	}
	return Source{}, fmt.Errorf("%w — for a local directory, write it as a path like ./%s", err, s)
}

func looksLikePath(s string) bool {
	switch s {
	case ".", "..", "~":
		return true
	}
	for _, p := range []string{"./", "../", "~/", "/"} {
		if strings.HasPrefix(s, p) {
			return true
		}
	}
	if filepath.Separator != '/' {
		for _, p := range []string{`.\`, `..\`, `~\`} {
			if strings.HasPrefix(s, p) {
				return true
			}
		}
	}
	return filepath.IsAbs(s)
}

// LocalSource는 이 기계의 디렉토리 dir를 출처로 삼는다. `~`를 펼치고 절대 경로로 바꾸고, 심볼릭 링크를
// 풀어 둔다 — 기록한 경로로 update가 다시 복사하므로 실행한 자리에 매이지 않아야 한다.
func LocalSource(dir string) (Source, error) {
	p := dir
	if p == "~" || strings.HasPrefix(p, "~/") || strings.HasPrefix(p, `~\`) {
		home, err := os.UserHomeDir()
		if err != nil {
			return Source{}, fmt.Errorf("%q: cannot determine the home directory: %w", dir, err)
		}
		p = filepath.Join(home, p[1:])
	}
	abs, err := filepath.Abs(p)
	if err != nil {
		return Source{}, err
	}
	info, err := os.Stat(abs)
	switch {
	case errors.Is(err, os.ErrNotExist):
		return Source{}, fmt.Errorf("directory %s does not exist", abs)
	case err != nil:
		return Source{}, err
	case !info.IsDir():
		return Source{}, fmt.Errorf("%s is not a directory — point at a theme directory (where theme.json is) or a directory containing one", abs)
	}
	if real, err := filepath.EvalSymlinks(abs); err == nil {
		abs = real
	}
	return Source{Dir: abs}, nil
}
