package themestore

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// copyDir는 로컬 출처 src를 dest 아래에 복사한다. 압축을 풀 때와 같은 규칙이다: 정규 파일과
// 디렉토리만 만들고, 링크는 따라가지 않고 links에 적어 둔다(테마 안에 있으면 설치가 거부한다).
// 홈은 복사하지 않는다 — 홈을 담은 디렉토리를 가리키면 복사가 자기 자신을 담게 된다.
//
// `.`으로 시작하는 디렉토리(`.git` 등)는 복사하지 않는다. 찾기(discover)가 어차피 보지 않는 자리이고,
// 로컬 작업 트리의 `.git`은 테마보다 훨씬 클 수 있다.
func copyDir(src Source, dest, home string) (snapshot, error) {
	snap := snapshot{dir: dest}
	homeAbs, _ := filepath.Abs(home)
	if real, err := filepath.EvalSymlinks(homeAbs); err == nil {
		homeAbs = real
	}
	var total int64
	files := 0
	err := filepath.WalkDir(src.Dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src.Dir, p)
		if err != nil {
			return err
		}
		slash := filepath.ToSlash(rel)
		if d.IsDir() {
			if rel != "." && (strings.HasPrefix(d.Name(), ".") || p == homeAbs) {
				return fs.SkipDir
			}
			return os.MkdirAll(filepath.Join(dest, rel), 0o755)
		}
		switch {
		case d.Type()&fs.ModeSymlink != 0:
			snap.links = append(snap.links, slash)
			return nil
		case !d.Type().IsRegular():
			return nil // 소켓 · 장치 · 파이프는 테마일 수 없다
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		files++
		total += info.Size()
		if files > maxArchiveFiles || total > maxArchiveBytes {
			return fmt.Errorf("디렉토리가 너무 크다 (파일 %d개 · %d바이트를 넘는다) — 테마 디렉토리를 직접 가리킨다",
				maxArchiveFiles, maxArchiveBytes)
		}
		f, err := os.Open(p)
		if err != nil {
			return err
		}
		defer f.Close()
		return writeFile(filepath.Join(dest, rel), f, info.Size())
	})
	if err != nil {
		return snapshot{}, fmt.Errorf("%s를 복사할 수 없다: %w", src.Dir, err)
	}
	return snap, nil
}
