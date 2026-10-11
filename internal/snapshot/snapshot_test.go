package snapshot

import (
	"image/gif"
	"os"
	"path/filepath"
	"testing"

	"github.com/k2sebeom/balpyo/internal/pipeline"
)

func TestRunWritesImagesAndGIF(t *testing.T) {
	dir := t.TempDir()
	res, err := Run(Options{
		Build:  pipeline.Options{ProjectDir: "../../testdata/fixtures/02_text_subset"},
		Dir:    dir,
		Slides: []int{1, 2},
		GIF:    true,
	})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"001.png", "002.png", GIFName}
	if len(res.Written) != len(want) {
		t.Fatalf("written %v, want %v", res.Written, want)
	}
	for i, name := range want {
		if res.Written[i] != name {
			t.Errorf("written[%d] = %s, want %s", i, res.Written[i], name)
		}
		if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
			t.Error(err)
		}
	}
	f, err := os.Open(filepath.Join(dir, GIFName))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	anim, err := gif.DecodeAll(f)
	if err != nil {
		t.Fatal(err)
	}
	// 둘째 슬라이드의 목록은 reveal이 없으므로 슬라이드마다 한 프레임이다.
	if len(anim.Image) < 2 {
		t.Errorf("%d GIF frames, want at least one per slide", len(anim.Image))
	}
}

func TestRunRejectsMissingSlide(t *testing.T) {
	_, err := Run(Options{Build: pipeline.Options{ProjectDir: "../../testdata/fixtures/01_text"}, Dir: t.TempDir(), Slides: []int{99}})
	if err == nil {
		t.Fatal("want an error for a slide number past the deck")
	}
}
