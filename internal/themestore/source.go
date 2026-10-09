// Package themestore는 설치한 테마의 홈(`~/.balpyo`)이다 — GitHub 저장소에서 테마를 받아
// `themes/<이름>/`에 두고, 어디서 왔는지 `themes.json`에 적는다.
//
// 빌드는 이 패키지를 쓰지 않는다: 설치한 테마는 홈의 디렉토리일 뿐이고, 그것을 이름의 어휘에 넣는
// 것은 core(WithInstalledThemes)다. 네트워크를 쓰는 자리가 사람이 친 명령(`balpyo theme install`)
// 하나여야 빌드가 오프라인에서도 돌고 같은 입력에 같은 바이트를 낸다.
package themestore

import (
	"fmt"
	"regexp"
	"strings"
)

// Source는 테마를 받을 GitHub 저장소의 한 자리다.
type Source struct {
	Owner string
	Repo  string
	// Path는 저장소 안의 하위 디렉토리다 (슬래시 구분, 비면 저장소 루트). 테마는 이 아래에서 찾는다.
	Path string
	// Ref는 브랜치 · 태그 · 커밋이다 (비면 저장소의 기본 브랜치).
	Ref string
}

// RepoID는 출처 기록에 적는 저장소 이름이다 (`github.com/<owner>/<repo>`).
func (s Source) RepoID() string { return "github.com/" + s.Owner + "/" + s.Repo }

// String은 사람이 다시 칠 수 있는 모양이다 (`github.com/o/r/path@ref`).
func (s Source) String() string {
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
		return Source{}, fmt.Errorf("GitHub 저장소 주소가 아니다: %q (예: github.com/<owner>/<repo>)", raw)
	}

	var src Source
	if i := strings.LastIndex(rest, "@"); i >= 0 {
		src.Ref, rest = rest[i+1:], rest[:i]
		if src.Ref == "" {
			return Source{}, fmt.Errorf("%q: @ 뒤에 ref가 없다", raw)
		}
	}
	parts := strings.Split(strings.Trim(rest, "/"), "/")
	if len(parts) < 2 || parts[0] == "" || parts[1] == "" {
		return Source{}, fmt.Errorf("%q: 저장소 주소에 owner와 repo가 둘 다 있어야 한다", raw)
	}
	src.Owner, src.Repo = parts[0], strings.TrimSuffix(parts[1], ".git")
	if !namePart.MatchString(src.Owner) || !namePart.MatchString(src.Repo) {
		return Source{}, fmt.Errorf("%q: owner·repo 이름에 쓸 수 없는 글자가 있다", raw)
	}

	tail := parts[2:]
	if len(tail) > 0 && tail[0] == "blob" {
		return Source{}, fmt.Errorf("%q는 파일을 가리킨다 — 테마 디렉토리나 저장소를 가리켜야 한다", raw)
	}
	if len(tail) > 0 && tail[0] == "tree" {
		if len(tail) < 2 {
			return Source{}, fmt.Errorf("%q: /tree/ 뒤에 ref가 없다", raw)
		}
		if src.Ref != "" {
			return Source{}, fmt.Errorf("%q: ref를 /tree/와 @로 두 번 줬다", raw)
		}
		src.Ref, tail = tail[1], tail[2:]
	}
	for _, seg := range tail {
		if seg == "" || seg == "." || seg == ".." {
			return Source{}, fmt.Errorf("%q: 하위 경로에 빈 이름이나 .·..이 있다", raw)
		}
	}
	src.Path = strings.Join(tail, "/")
	if src.Ref != "" && (!refChars.MatchString(src.Ref) || strings.Contains(src.Ref, "..")) {
		return Source{}, fmt.Errorf("%q: ref %q에 쓸 수 없는 글자가 있다", raw, src.Ref)
	}
	return src, nil
}
