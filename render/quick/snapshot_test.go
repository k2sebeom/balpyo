package quick_test

import (
	"bytes"
	"errors"
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

// Emit이 있으면 그림은 그리는 대로 넘어가고 Result에 남지 않는다 — 덱 전체의 그림을 메모리에
// 쥐지 않는다. 순서는 슬라이드마다 스텝 0 … n-1, 그다음 모든 스텝이 펼쳐진 장면이다.
func TestSnapshotEmitStreamsFrames(t *testing.T) {
	deck, _, err := pipeline.LoadDeck(pipeline.Options{ProjectDir: "../../testdata/fixtures/02_text_subset"})
	if err != nil {
		t.Fatal(err)
	}
	collected, err := quick.Snapshot(deck, quick.Options{DPR: 1, Steps: true})
	if err != nil {
		t.Fatal(err)
	}
	var got []quick.Frame
	res, err := quick.Snapshot(deck, quick.Options{DPR: 1, Steps: true, Emit: func(f quick.Frame) error {
		got = append(got, f)
		return nil
	}})
	if err != nil {
		t.Fatal(err)
	}
	if res.Slides != nil || res.Steps != nil {
		t.Errorf("Result holds %d images and %d step lists, want none when Emit is set", len(res.Slides), len(res.Steps))
	}
	if res.Geometry == nil {
		t.Error("Result lost the geometry")
	}
	i := 0
	for slide, steps := range collected.Steps {
		for step, want := range steps {
			if i >= len(got) {
				t.Fatalf("only %d frames emitted", len(got))
			}
			f := got[i]
			wantStep := step
			if step == len(steps)-1 {
				wantStep = -1
			}
			if f.Slide != slide || f.Step != wantStep {
				t.Errorf("frame %d is slide %d step %d, want slide %d step %d", i, f.Slide, f.Step, slide, wantStep)
			}
			if !bytes.Equal(f.Image.Pix, want.Pix) {
				t.Errorf("frame %d (slide %d step %d) differs from the collected one", i, slide, step)
			}
			i++
		}
	}
	if i != len(got) {
		t.Errorf("%d frames emitted, want %d", len(got), i)
	}
	if i <= len(deck.Slides) {
		t.Errorf("%d frames for %d slides — the fixture has reveal steps, so want more", i, len(deck.Slides))
	}

	// Emit의 에러는 그리기를 멈추고 그대로 나온다.
	stop := errors.New("stop")
	calls := 0
	_, err = quick.Snapshot(deck, quick.Options{DPR: 1, Emit: func(quick.Frame) error {
		calls++
		return stop
	}})
	if !errors.Is(err, stop) || calls != 1 {
		t.Errorf("err = %v after %d calls, want the Emit error after one call", err, calls)
	}
}
