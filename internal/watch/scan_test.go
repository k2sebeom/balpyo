package watch

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// 감시기의 계약은 두 문장이다 (ADR 0046).
//
//  1. 변경을 **보자마자 빌드하지 않는다** — 직전 스냅샷과 같아진 뒤에야 한 번 신호한다.
//     에디터는 저장 한 번에 여러 파일을 쓰므로(포맷터·임시 파일·`balpyo.json`), 보자마자
//     빌드하면 저장 한 번이 빌드 두세 번이 되고 앞의 것들은 **반쯤 쓰인 파일**을 파싱해
//     가짜 진단을 낸다.
//  2. 자기 산출물은 보지 않는다 — 빌드가 출력 디렉토리를 쓰면 그 쓰기가 다음 빌드를 부르고,
//     루프가 스스로를 먹는다.
//
// 이 파일은 그 두 문장만 본다. 빌드·서버는 각자의 테스트가 있다.

// 조용한 트리에서는 몇 번을 물어도 아무 일도 없다 — 폴링이 빌드를 만들어내지 않는다.
func TestWatcherIsSilentOnAStillTree(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "slides/001_a.slide", "<slide></slide>")
	w := newWatcher(t, root)
	for i := 0; i < 3; i++ {
		if fired := poll(t, w); fired {
			t.Fatalf("조용한 트리의 %d번째 Poll이 신호했다", i+1)
		}
	}
}

// 변경은 **멈춘 뒤에** 한 번 신호한다: 첫 Poll은 변경을 보고 기다리고, 다음 Poll이 신호하며,
// 그 다음은 다시 조용하다 (같은 변경으로 두 번 빌드하지 않는다).
func TestWatcherFiresOnceAfterChangesStop(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "slides/001_a.slide", "<slide></slide>")
	w := newWatcher(t, root)

	writeFile(t, root, "slides/001_a.slide", "<slide><h1>고쳤다</h1></slide>")
	if poll(t, w) {
		t.Fatal("변경을 본 첫 Poll이 바로 신호했다 — 조용해질 때까지 기다려야 한다")
	}
	if !poll(t, w) {
		t.Fatal("변경이 멈췄는데 신호하지 않았다")
	}
	if poll(t, w) {
		t.Fatal("같은 변경으로 두 번 신호했다")
	}
}

// 변경이 계속 들어오는 동안에는 신호하지 않는다 — 저장 폭풍 하나가 빌드 하나가 된다.
func TestWatcherWaitsWhileChangesKeepComing(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "slides/001_a.slide", "<slide></slide>")
	w := newWatcher(t, root)

	for i := 0; i < 3; i++ {
		writeFile(t, root, "slides/001_a.slide", repeat("x", i+1))
		if poll(t, w) {
			t.Fatalf("변경이 이어지는 중(%d번째)에 신호했다", i+1)
		}
	}
	if !poll(t, w) {
		t.Fatal("변경이 멈춘 뒤에 신호하지 않았다")
	}
}

// 잡아야 하는 변경의 종류: 새 파일 · 지운 파일 · **mtime이 같고 크기만 다른** 수정.
//
// 마지막 항목이 스냅샷 서명에 크기가 들어가는 이유다 — 슬라이드 한 줄을 지우는 편집은
// 파일 시스템의 mtime 해상도 안에서 일어날 수 있다.
func TestWatcherSeesCreateDeleteAndSizeOnlyEdits(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "slides/001_a.slide", "<slide></slide>")
	w := newWatcher(t, root)

	tests := []struct {
		name   string
		change func()
	}{
		{"새 슬라이드", func() { writeFile(t, root, "slides/002_b.slide", "<slide></slide>") }},
		{"새 에셋", func() { writeFile(t, root, "assets/dot.png", "not really a png") }},
		{"설정 변경", func() { writeFile(t, root, "balpyo.json", `{"title":"t"}`) }},
		{"크기만 다른 수정", func() {
			path := filepath.Join(root, "slides", "001_a.slide")
			stamp := modTime(t, path)
			writeFile(t, root, "slides/001_a.slide", "<slide><p>더 길다</p></slide>")
			// mtime을 되돌려 **크기만 다른** 상태를 만든다.
			if err := os.Chtimes(path, stamp, stamp); err != nil {
				t.Fatal(err)
			}
		}},
		{"지운 슬라이드", func() {
			if err := os.Remove(filepath.Join(root, "slides", "002_b.slide")); err != nil {
				t.Fatal(err)
			}
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.change()
			if poll(t, w) {
				t.Fatal("변경을 본 Poll이 바로 신호했다")
			}
			if !poll(t, w) {
				t.Fatal("변경이 감지되지 않았다")
			}
		})
	}
}

// 출력 디렉토리와 `.`으로 시작하는 디렉토리는 걷지 않는다 — 빌드가 쓴 산출물이 다음 빌드를
// 부르면 감시 루프가 멈추지 않는다.
func TestWatcherIgnoresOutputAndDotDirs(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "slides/001_a.slide", "<slide></slide>")
	writeFile(t, root, "dist/index.html", "<html></html>")
	w := newWatcher(t, root)

	writeFile(t, root, "dist/index.html", "<html>새 산출물</html>")
	writeFile(t, root, "dist/deck.pptx", "PK")
	writeFile(t, root, ".git/HEAD", "ref: refs/heads/main")
	for i := 0; i < 2; i++ {
		if poll(t, w) {
			t.Fatal("산출물·`.git`의 변경이 빌드를 불렀다 — 루프가 스스로를 먹는다")
		}
	}
	// 같은 감시기가 **소스** 변경은 여전히 본다 (무시 규칙이 트리 전체를 끄지 않는다).
	writeFile(t, root, "slides/001_a.slide", "<slide><h1>고쳤다</h1></slide>")
	poll(t, w)
	if !poll(t, w) {
		t.Fatal("무시 규칙이 소스 변경까지 가렸다")
	}
}

// 출력 디렉토리가 프로젝트 **밖**이면 무시 규칙은 점 디렉토리만 남는다 — 그 경우에도
// 감시기는 만들어져야 한다 (`--out /tmp/x`는 정상 사용법이다).
func TestProjectSkipAcceptsAnOutsideOutputDir(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "slides/001_a.slide", "<slide></slide>")
	skip, err := ProjectSkip(root, t.TempDir())
	if err != nil {
		t.Fatalf("프로젝트 밖 출력 디렉토리가 거부되었다: %v", err)
	}
	if skip("slides") || skip("slides/001_a.slide") {
		t.Error("소스가 무시 대상으로 잡혔다")
	}
	if !skip(".git") {
		t.Error("점 디렉토리가 무시되지 않았다")
	}
}

// 없는 루트는 에러다 — 감시기가 조용히 빈 트리를 보면 저자는 "고쳤는데 아무 일도 없다"를
// 겪는다 (`--project` 오타의 증상이 그것이다).
func TestScanRejectsAMissingRoot(t *testing.T) {
	if _, err := Scan(filepath.Join(t.TempDir(), "없다"), nil); err == nil {
		t.Fatal("없는 루트가 에러로 나오지 않았다")
	}
}

// newWatcher는 감시기를 만든다 — 출력 디렉토리는 프로젝트 안의 `dist`다 (기본 설정).
func newWatcher(t *testing.T, root string) *Watcher {
	t.Helper()
	skip, err := ProjectSkip(root, filepath.Join(root, "dist"))
	if err != nil {
		t.Fatal(err)
	}
	w, err := NewWatcher(root, skip)
	if err != nil {
		t.Fatal(err)
	}
	return w
}

// poll은 Poll 한 번이다 — 에러는 테스트를 멈춘다.
func poll(t *testing.T, w *Watcher) bool {
	t.Helper()
	changed, err := w.Poll()
	if err != nil {
		t.Fatalf("Poll 실패: %v", err)
	}
	return changed
}

// writeFile은 프로젝트 안에 파일 하나를 쓴다 (없는 디렉토리는 만든다).
func writeFile(t *testing.T, root, rel, body string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func modTime(t *testing.T, path string) time.Time {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	return info.ModTime()
}

func repeat(s string, n int) string {
	out := ""
	for i := 0; i < n; i++ {
		out += s
	}
	return out
}
