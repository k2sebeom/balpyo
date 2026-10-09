// Package watch는 `balpyo watch`의 본체다 — 프로젝트를 감시해 다시 빌드하고, 그 산출물을
// 리로드 클라이언트와 함께 서브한다.
//
// 왜 internal 패키지인가: `cmd/balpyo`는 main이라 테스트가 임포트할 수 없고(pipeline이 같은
// 이유로 internal에 있다), 감시 루프에는 테스트로 못 박을 계약이 셋 있다 — 디바운스,
// 응답 주입, 진단 오버레이. CLI는 플래그만 옮긴다.
package watch

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// DefaultInterval은 폴링 간격이다. 디바운스가 이 간격 하나이므로(스냅샷이 안정될 때까지
// 기다린다) 저장에서 새 화면까지의 지연에 이 값이 그대로 들어간다.
const DefaultInterval = 250 * time.Millisecond

// SkipFunc은 걷지 않을 경로를 고른다. rel은 루트 기준 **슬래시** 경로다 (OS 구분자가 아니라
// 슬래시인 이유: 무시 규칙을 쓰는 쪽이 플랫폼마다 다른 문자열을 적지 않게 한다).
//
// 디렉토리에 true를 돌려주면 그 하위 트리 전체를 걷지 않는다.
type SkipFunc func(rel string) bool

// stamp는 파일 하나의 서명이다. 내용을 해시하지 않는 이유는 감시가 **빌드보다 자주** 도는
// 일이라는 것이다 — 크기와 mtime이면 편집을 놓치지 않으면서 비용이 Stat 하나다.
//
// 크기를 함께 보는 이유: mtime 해상도 안에서 일어난 편집(한 줄 지우기)은 mtime만으로는
// 같아 보인다. 둘 다 같고 내용만 다른 편집은 이 감시기가 놓치는 유일한 경우이고, 그때는
// 저장을 한 번 더 하거나 감시를 다시 시작하면 된다 — 해시로 올리는 비용을 그 희귀한 경우에
// 내지 않는다.
type stamp struct {
	size    int64
	modTime time.Time
}

// Snapshot은 감시 대상 트리 한 순간의 상태다 (rel → 서명).
type Snapshot map[string]stamp

// Equal은 두 스냅샷이 같은 트리인지 본다 (파일이 생겼다·없어졌다·달라졌다).
func (s Snapshot) Equal(other Snapshot) bool {
	if len(s) != len(other) {
		return false
	}
	for path, got := range s {
		want, ok := other[path]
		if !ok || got.size != want.size || !got.modTime.Equal(want.modTime) {
			return false
		}
	}
	return true
}

// Scan은 루트 아래 파일들의 스냅샷을 만든다. skip이 nil이면 트리 전체를 걷는다.
//
// 루트가 없으면 **에러**다: 조용히 빈 스냅샷을 돌려주면 `--project` 오타가 "고쳤는데 아무
// 일도 없다"로 나타난다 (그때 저자는 자기 슬라이드를 의심한다).
func Scan(root string, skip SkipFunc) (Snapshot, error) {
	info, err := os.Stat(root)
	if err != nil {
		return nil, fmt.Errorf("감시할 디렉토리를 열 수 없다: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("감시할 디렉토리가 아니다: %s", root)
	}
	snap := Snapshot{}
	err = filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			// 걷는 중에 사라진 파일·디렉토리는 실패가 아니다 — 에디터의 임시 파일은
			// 스캔 도중에 지워진다. 다음 스캔이 그 자리의 진실을 본다.
			if os.IsNotExist(err) {
				return nil
			}
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		rel = filepath.ToSlash(rel)
		if skip != nil && skip(rel) {
			if entry.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		if entry.IsDir() {
			return nil
		}
		stat, err := entry.Info()
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return err
		}
		snap[rel] = stamp{size: stat.Size(), modTime: stat.ModTime()}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("%s를 걸을 수 없다: %w", root, err)
	}
	return snap, nil
}

// ProjectSkip은 프로젝트 감시의 무시 규칙이다: 출력 디렉토리와 `.`으로 시작하는 경로.
//
// 출력 디렉토리를 빼는 이유는 루프가 스스로를 먹기 때문이다 — 빌드가 산출물을 쓰면 그 쓰기가
// 다음 빌드를 부르고, 그 빌드가 또 쓴다. 점 디렉토리를 빼는 이유는 `.git`이 커밋 하나에
// 수백 개 파일을 움직인다는 것이다 (그 변경은 슬라이드와 무관하다).
//
// 출력 디렉토리가 프로젝트 **밖**이면(`--out /tmp/x`) 점 규칙만 남는다.
func ProjectSkip(root, outDir string) (SkipFunc, error) {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("프로젝트 경로를 정할 수 없다: %w", err)
	}
	var outRel string
	if outDir != "" {
		absOut, err := filepath.Abs(outDir)
		if err != nil {
			return nil, fmt.Errorf("출력 디렉토리 경로를 정할 수 없다: %w", err)
		}
		rel, err := filepath.Rel(absRoot, absOut)
		if err == nil && rel != "." && !strings.HasPrefix(rel, "..") {
			outRel = filepath.ToSlash(rel)
		}
	}
	return func(rel string) bool {
		if outRel != "" && (rel == outRel || strings.HasPrefix(rel, outRel+"/")) {
			return true
		}
		for _, segment := range strings.Split(rel, "/") {
			if strings.HasPrefix(segment, ".") {
				return true
			}
		}
		return false
	}, nil
}

// Watcher는 폴링 감시기다. Poll이 true를 돌려주는 순간은 **변경이 멈춘 직후 한 번**이다.
//
// 타이머 대신 스냅샷 둘을 견주는 이유: 에디터는 저장 한 번에 여러 파일을 쓴다.
// 변경을 보자마자 빌드하면 저장 한 번이 빌드 두세 번이 되고, 앞의 것들은 반쯤 쓰인 파일을
// 파싱해 가짜 진단을 낸다. "직전 스캔과 같아졌다"는 조건이 그 둘을 타이머 없이 없앤다.
type Watcher struct {
	root string
	skip SkipFunc
	// seen은 **마지막으로 본** 트리다 (아직 빌드하지 않았을 수 있다).
	seen Snapshot
	// built는 마지막으로 신호한 트리다 — 같은 변경으로 두 번 빌드하지 않기 위한 기준이다.
	built Snapshot
}

// NewWatcher는 지금 트리를 기준으로 감시기를 만든다. 첫 빌드는 호출자가 한다 — 감시기는
// "무엇이 달라졌는가"만 알고, "처음에 한 번 빌드한다"는 루프의 규칙이다.
func NewWatcher(root string, skip SkipFunc) (*Watcher, error) {
	snap, err := Scan(root, skip)
	if err != nil {
		return nil, err
	}
	return &Watcher{root: root, skip: skip, seen: snap, built: snap}, nil
}

// Poll은 한 번 스캔한다. 트리가 직전 스캔과 같고 마지막 신호 이후 달라졌으면 true다.
func (w *Watcher) Poll() (bool, error) {
	snap, err := Scan(w.root, w.skip)
	if err != nil {
		return false, err
	}
	stable := snap.Equal(w.seen)
	w.seen = snap
	if !stable {
		return false, nil
	}
	if snap.Equal(w.built) {
		return false, nil
	}
	w.built = snap
	return true, nil
}
