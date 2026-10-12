package quick_test

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/png"
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

// image-full의 media 그림이 슬롯을 암묵으로 채우면(w·h·ratio 없음) 내장 엔진도 브라우저처럼 그림을
// `1fr` 행에 맞추고 caption을 아래 `auto` 행에 놓는다. 16:9 그림은 스테이지 폭에서 스테이지 높이와
// 같아지므로, 행의 최소 기여를 내용 높이로 잡으면 그림이 스테이지를 다 먹고 caption이 밖으로 밀린다.
func TestDriftImageFullCaption(t *testing.T) {
	testkit.RequireChromium(t)
	img := image.NewRGBA(image.Rect(0, 0, 640, 360))
	for y := 0; y < 360; y++ {
		for x := 0; x < 640; x++ {
			img.Set(x, y, color.RGBA{0x30, 0x60, 0x90, 0xff})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	want, got := measureBoth(t, map[string][]byte{
		"balpyo.json": []byte(`{"title": "image-full", "theme": "minimal"}`),
		"slides/01.slide": []byte(`<slide id="full" layout="image-full">
  <image slot="media" src="assets/photo.png" fit="cover" alt="photo"/>
  <text slot="caption" role="caption">CAPTION TEXT</text>
</slide>`),
		"assets/photo.png": buf.Bytes(),
	})
	for _, id := range []string{"s0.0", "s0.1"} {
		wb, ok := want.Box(id)
		if !ok {
			t.Fatalf("browser did not measure %s", id)
		}
		gb, ok := got.Box(id)
		if !ok {
			t.Errorf("%s: builtin engine has no box (web %s)", id, wb.Rect)
			continue
		}
		e := math.Max(math.Max(math.Abs(wb.Rect.X-gb.Rect.X), math.Abs(wb.Rect.Y-gb.Rect.Y)),
			math.Max(math.Abs(wb.Rect.W-gb.Rect.W), math.Abs(wb.Rect.H-gb.Rect.H)))
		if e > 1 {
			t.Errorf("%s: web %s, quick %s", id, wb.Rect, gb.Rect)
		}
	}
}

// 한국어 덱의 어절은 칸에 자리가 있으면 통째로 남는다 — 표의 짧은 열과 row의 짧은 항목은 긴
// 이웃 때문에 어절 폭 아래로 줄지 않는다. 어절을 글자 사이에서 자르는 것은 어절 하나가 박스보다
// 넓을 때뿐이라, 최소 내용 폭은 음절이 아니라 어절이다. 비교 기준은 같은 어절이 홀로 놓인 row
// 항목의 폭(어절 폭)이고, 두 엔진이 다 지켜야 한다.
func TestKoreanWordKeepsItsWidthBesideLongNeighbor(t *testing.T) {
	testkit.RequireChromium(t)
	const long = "파드 요구사항에 맞춰 노드를 즉시 프로비저닝하고 가장 저렴한 인스턴스 타입을 자동으로 고르며 빈 노드를 정리합니다"
	want, got := measureBoth(t, map[string][]byte{
		"balpyo.json": []byte(`{"title": "ko", "lang": "ko", "aspect": "4:3", "theme": "minimal"}`),
		"slides/01.slide": []byte(`<slide id="squeezed">
  <table><tr><th>구분</th><th>설명</th></tr><tr><td>오토스케일러</td><td>` + long + `</td></tr></table>
  <row gap="md"><text>컨테이너오케스트레이션</text><text>` + long + `</text></row>
</slide>`),
		"slides/02.slide": []byte(`<slide id="alone">
  <row><text>오토스케일러</text></row>
  <row><text>컨테이너오케스트레이션</text></row>
</slide>`),
	})
	for _, c := range []struct{ name, squeezed, alone string }{
		{"table cell", "s0.0.1.0", "s1.0.0"},
		{"row item", "s0.1.0", "s1.1.0"},
	} {
		for engine, g := range map[string]*geom.Geometry{"web": want, "quick": got} {
			sq, ok1 := g.Box(c.squeezed)
			al, ok2 := g.Box(c.alone)
			if !ok1 || !ok2 {
				t.Fatalf("%s: %s did not measure %s / %s", c.name, engine, c.squeezed, c.alone)
			}
			if sq.Rect.W < al.Rect.W-0.5 {
				t.Errorf("%s (%s): squeezed to %.1fpx below its word width %.1fpx", c.name, engine, sq.Rect.W, al.Rect.W)
			}
		}
	}
}

// measureBoth는 프로젝트 파일로 덱을 빌드해 같은 측정용 HTML을 브라우저와 내장 엔진이 각각 잰
// 결과를 돌려준다.
func measureBoth(t *testing.T, project map[string][]byte) (want, got *geom.Geometry) {
	t.Helper()
	dir := t.TempDir()
	for name, body := range project {
		p := filepath.Join(dir, filepath.FromSlash(name))
		os.MkdirAll(filepath.Dir(p), 0o755)
		if err := os.WriteFile(p, body, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	deck, diags, err := pipeline.LoadDeck(pipeline.Options{ProjectDir: dir})
	if err != nil || diags.Err() != nil {
		t.Fatalf("load: %v %v", err, diags)
	}
	files, err := web.NewForMeasurement().Render(deck)
	if err != nil {
		t.Fatal(err)
	}
	out := t.TempDir()
	for _, name := range files.Paths() {
		p := filepath.Join(out, filepath.FromSlash(name))
		os.MkdirAll(filepath.Dir(p), 0o755)
		os.WriteFile(p, files[name], 0o644)
	}
	html := filepath.Join(out, web.IndexFile)
	session, err := geom.Launch("")
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	if want, err = session.Measure(deck, html); err != nil {
		t.Fatal(err)
	}
	if got, err = (quick.Measurer{}).Measure(deck, html); err != nil {
		t.Fatal(err)
	}
	return want, got
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
