package quick_test

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/k2sebeom/balpyo/internal/pipeline"
	"github.com/k2sebeom/balpyo/internal/testkit"
	"github.com/k2sebeom/balpyo/render/geom"
	"github.com/k2sebeom/balpyo/render/quick"
	"github.com/k2sebeom/balpyo/render/web"
)

// driftFloor는 브라우저와 1px 안에서 맞아야 하는 박스의 최소 비율이다. 내장 엔진은 미리보기라
// 줄바꿈이 갈리는 박스가 있을 수 있지만, 이 바닥 아래로 떨어지면 엔진의 배치가 회귀한 것이다.
const driftFloor = 0.99

// TestDrift는 내장 엔진의 측정과 브라우저 측정을 박스 단위로 견준다 — 예제 덱과 fixture의 같은
// HTML을 두 쪽이 잰다. BALPYO_DRIFT=1이면 덱마다 가장 많이 어긋난 박스를 보고한다.
func TestDrift(t *testing.T) {
	testkit.RequireChromium(t)
	verbose := os.Getenv("BALPYO_DRIFT") != ""
	session, err := geom.Launch("")
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()

	var projects []string
	for _, pattern := range []string{"../../examples/decks/*", "../../testdata/fixtures/*"} {
		matches, _ := filepath.Glob(pattern)
		for _, m := range matches {
			if _, err := os.Stat(filepath.Join(m, "balpyo.json")); err == nil {
				projects = append(projects, m)
			}
		}
	}
	filter := os.Getenv("BALPYO_DRIFT_ONLY")
	var all []float64
	for _, dir := range projects {
		if filter != "" && !strings.Contains(dir, filter) {
			continue
		}
		deck, diags, err := pipeline.LoadDeck(pipeline.Options{ProjectDir: dir})
		if err != nil || diags.Err() != nil {
			continue
		}
		files, err := web.NewForMeasurement().Render(deck)
		if err != nil {
			continue
		}
		tmp := t.TempDir()
		for _, name := range files.Paths() {
			p := filepath.Join(tmp, filepath.FromSlash(name))
			os.MkdirAll(filepath.Dir(p), 0o755)
			os.WriteFile(p, files[name], 0o644)
		}
		html := filepath.Join(tmp, web.IndexFile)
		want, err := session.Measure(deck, html)
		if err != nil {
			t.Logf("%s: browser: %v", dir, err)
			continue
		}
		got, err := quick.Measurer{}.Measure(deck, html)
		if err != nil {
			t.Errorf("%s: builtin: %v", filepath.Base(dir), err)
			continue
		}
		type miss struct {
			id   string
			el   string
			err  float64
			w, g geom.Rect
		}
		var misses []miss
		var errs []float64
		for _, wb := range want.Boxes {
			gb, ok := got.Box(wb.ID)
			if !ok {
				continue
			}
			e := math.Max(math.Max(math.Abs(wb.Rect.X-gb.Rect.X), math.Abs(wb.Rect.Y-gb.Rect.Y)),
				math.Max(math.Abs(wb.Rect.W-gb.Rect.W), math.Abs(wb.Rect.H-gb.Rect.H)))
			errs = append(errs, e)
			misses = append(misses, miss{wb.ID, wb.El, e, wb.Rect, gb.Rect})
		}
		all = append(all, errs...)
		sort.SliceStable(misses, func(i, j int) bool { return misses[i].err > misses[j].err })
		var worst []string
		for _, m := range misses {
			if m.err <= 1 || len(worst) >= 4 {
				break
			}
			worst = append(worst, fmt.Sprintf("%s<%s> %.0f (web %s, quick %s)", m.id, m.el, m.err, m.w, m.g))
		}
		if verbose {
			t.Logf("%-28s %s  %s", filepath.Base(dir), summary(errs), strings.Join(worst, "; "))
		}
	}
	t.Logf("%-28s %s", "TOTAL", summary(all))
	if share := within(all, 1); share < driftFloor {
		t.Errorf("only %.1f%% of boxes are within 1px of the browser (floor %.0f%%)", 100*share, 100*driftFloor)
	}
}

func within(errs []float64, d float64) float64 {
	if len(errs) == 0 {
		return 1
	}
	n := 0
	for _, e := range errs {
		if e <= d {
			n++
		}
	}
	return float64(n) / float64(len(errs))
}

func summary(errs []float64) string {
	if len(errs) == 0 {
		return "no boxes"
	}
	return fmt.Sprintf("%4d boxes  ≤1px %5.1f%%  ≤4px %5.1f%%  ≤16px %5.1f%%", len(errs),
		100*within(errs, 1), 100*within(errs, 4), 100*within(errs, 16))
}
