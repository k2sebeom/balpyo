package quick_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/k2sebeom/balpyo/internal/pipeline"
	"github.com/k2sebeom/balpyo/render/quick"
)

func projects(t *testing.T) []string {
	t.Helper()
	var out []string
	for _, pattern := range []string{"../../examples/decks/*", "../../testdata/fixtures/*"} {
		matches, _ := filepath.Glob(pattern)
		for _, m := range matches {
			if _, err := os.Stat(filepath.Join(m, "balpyo.json")); err == nil {
				out = append(out, m)
			}
		}
	}
	return out
}

// 모든 예제 덱과 fixture가 브라우저 없이 그려진다: 내장 측정이 덱과 대조를 통과하고(지오메트리
// 패스와 같은 검증), 슬라이드마다 스테이지 크기의 그림이 나오며, 빈 그림이 아니다.
func TestSnapshotProjects(t *testing.T) {
	for _, dir := range projects(t) {
		deck, diags, err := pipeline.LoadDeck(pipeline.Options{ProjectDir: dir})
		if err != nil || diags.Err() != nil {
			continue // 에러 진단을 시험하는 fixture
		}
		res, err := quick.Snapshot(deck, quick.Options{DPR: 1, Steps: true})
		if err != nil {
			t.Errorf("%s: %v", filepath.Base(dir), err)
			continue
		}
		if len(res.Slides) != len(deck.Slides) || len(res.Steps) != len(deck.Slides) {
			t.Errorf("%s: %d images, %d step lists for %d slides", filepath.Base(dir), len(res.Slides), len(res.Steps), len(deck.Slides))
			continue
		}
		for i, img := range res.Slides {
			b := img.Bounds()
			if b.Dx() != deck.Stage.Width || b.Dy() != deck.Stage.Height {
				t.Errorf("%s slide %d: %dx%d, want the %dx%d stage", filepath.Base(dir), i+1, b.Dx(), b.Dy(), deck.Stage.Width, deck.Stage.Height)
			}
			if len(deck.Slides[i].Content) > 0 && uniform(img.Pix) {
				t.Errorf("%s slide %d: the image is a single color", filepath.Base(dir), i+1)
			}
		}
	}
}

func uniform(pix []byte) bool {
	for i := 4; i < len(pix); i += 4 {
		if !bytes.Equal(pix[i:i+4], pix[:4]) {
			return false
		}
	}
	return true
}

// 같은 덱은 같은 그림이 된다 (배율 2배 포함).
func TestSnapshotDeterministic(t *testing.T) {
	deck, _, err := pipeline.LoadDeck(pipeline.Options{ProjectDir: "../../examples/decks/ec2-intro"})
	if err != nil {
		t.Fatal(err)
	}
	only := map[int]bool{deck.Slides[5].Index: true}
	a, err := quick.Snapshot(deck, quick.Options{DPR: 2, Only: only})
	if err != nil {
		t.Fatal(err)
	}
	b, err := quick.Snapshot(deck, quick.Options{DPR: 2, Only: only})
	if err != nil {
		t.Fatal(err)
	}
	if a.Slides[5] == nil || !bytes.Equal(a.Slides[5].Pix, b.Slides[5].Pix) {
		t.Error("two snapshots of the same slide differ")
	}
	if a.Slides[0] != nil {
		t.Error("a slide outside Only was drawn")
	}
}
