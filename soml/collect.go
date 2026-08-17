package soml

import (
	"fmt"
	"io/fs"
	"path"
	"sort"
	"strings"
	"unicode"
)

// Ext는 SOML 파일의 유일한 확장자다 (PRD §13-2). 슬라이드와 합성 컴포넌트가 같은 확장자를
// 쓰고, 파일의 역할은 루트 요소와 디렉토리가 결정한다.
const Ext = ".slide"

// SlideFile은 수집된 슬라이드 파일 하나다. 아직 읽지 않은 상태 — 경로와 순서만 안다.
type SlideFile struct {
	// Path는 슬라이드 디렉토리 기준 상대 경로다 (항상 슬래시 구분).
	Path string
	// Chapter는 상위 디렉토리 경로다. 루트 직속이면 빈 문자열이다.
	Chapter string
	// Name은 확장자를 포함한 파일명이다.
	Name string
}

// DerivedID는 `id` 속성이 없는 슬라이드의 딥링크 앵커를 파일명에서 유도한다.
//
//	010_architecture.slide → architecture
//	001_title.slide        → title
//	deep dive.slide        → deep-dive
//
// 순서용 숫자 접두는 앵커의 일부가 아니다 — 슬라이드를 재배치해도 링크가 살아남아야 한다.
func (f SlideFile) DerivedID() string {
	name := strings.TrimSuffix(f.Name, Ext)
	if stripped := stripOrderPrefix(name); stripped != "" {
		name = stripped
	}
	return slugify(name)
}

// Collect는 슬라이드 디렉토리를 순회해 렌더 순서대로 정렬된 슬라이드 파일 목록을 낸다.
//
// 규칙 (PRD §6.1):
//   - `.slide` 파일만 수집한다.
//   - 하위 디렉토리는 챕터다.
//   - `_`로 시작하는 파일과 디렉토리는 무시한다 (초안, 보류 슬라이드).
//   - 순서는 경로 세그먼트별 사전순이다 — 숫자 접두(`001_`, `010_`, `900_`) 관례가
//     자연스러운 순서를 만든다.
func Collect(fsys fs.FS, dir string) ([]SlideFile, error) {
	if dir == "" {
		dir = "."
	}
	info, err := fs.Stat(fsys, dir)
	if err != nil {
		return nil, fmt.Errorf("슬라이드 디렉토리 %q를 열 수 없다: %w", dir, err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("슬라이드 디렉토리 %q가 디렉토리가 아니다", dir)
	}

	var files []SlideFile
	err = fs.WalkDir(fsys, dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, relErr := relativeTo(dir, p)
		if relErr != nil {
			return relErr
		}
		if rel == "." {
			return nil
		}
		if strings.HasPrefix(d.Name(), "_") {
			if d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		if d.IsDir() || path.Ext(d.Name()) != Ext {
			return nil
		}
		files = append(files, SlideFile{
			Path:    rel,
			Chapter: chapterOf(rel),
			Name:    d.Name(),
		})
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("슬라이드 디렉토리 %q 순회 실패: %w", dir, err)
	}

	sort.Slice(files, func(i, j int) bool {
		return lessPath(files[i].Path, files[j].Path)
	})
	return files, nil
}

// relativeTo는 base 아래의 경로를 base 기준 상대 경로로 바꾼다.
func relativeTo(base, p string) (string, error) {
	if base == "." {
		return p, nil
	}
	if p == base {
		return ".", nil
	}
	prefix := base + "/"
	if !strings.HasPrefix(p, prefix) {
		return "", fmt.Errorf("경로 %q가 %q 아래가 아니다", p, base)
	}
	return strings.TrimPrefix(p, prefix), nil
}

// chapterOf는 상대 경로에서 챕터(상위 디렉토리) 부분을 떼어낸다.
func chapterOf(rel string) string {
	dir := path.Dir(rel)
	if dir == "." {
		return ""
	}
	return dir
}

// lessPath는 경로를 세그먼트별로 비교한다.
//
// 문자열 전체를 비교하면 `/`(0x2F)가 `.`(0x2E)보다 커서 디렉토리 경계가 정렬에 끼어든다.
// 세그먼트별 비교는 "각 디렉토리 안에서 파일명 정렬"이라는 저자의 멘탈 모델과 일치한다.
func lessPath(a, b string) bool {
	as, bs := strings.Split(a, "/"), strings.Split(b, "/")
	for i := 0; i < len(as) && i < len(bs); i++ {
		if as[i] != bs[i] {
			return as[i] < bs[i]
		}
	}
	return len(as) < len(bs)
}

// stripOrderPrefix는 순서용 숫자 접두(`001_`, `010-`, `900`)를 떼어낸다.
// 남는 부분이 없으면 빈 문자열을 돌려준다 (호출자가 원본을 유지한다).
func stripOrderPrefix(name string) string {
	i := 0
	for i < len(name) && name[i] >= '0' && name[i] <= '9' {
		i++
	}
	if i == 0 {
		return name
	}
	for i < len(name) && (name[i] == '_' || name[i] == '-' || name[i] == ' ') {
		i++
	}
	return name[i:]
}

// slugify는 앵커로 쓸 수 있는 형태로 정규화한다. 한글 같은 비 ASCII 문자는 유지한다 —
// 앵커는 URL 프래그먼트이고 퍼센트 인코딩으로 실릴 수 있다.
func slugify(s string) string {
	var b strings.Builder
	lastDash := false
	for _, r := range strings.ToLower(s) {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			b.WriteRune(r)
			lastDash = false
		default:
			if !lastDash && b.Len() > 0 {
				b.WriteByte('-')
				lastDash = true
			}
		}
	}
	return strings.TrimRight(b.String(), "-")
}
