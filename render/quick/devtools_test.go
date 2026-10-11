package quick_test

// 개발 도구다 — 환경 변수로 켤 때만 돈다. 내장 엔진을 고칠 때 브라우저와 어디서 갈리는지 본다.
//
//	DUMP_PROJECT=<dir> DUMP_PREFIX=s3.1 [DUMP_LINES=1]  박스 · 줄 rect를 브라우저와 나란히 찍는다
//	COMPARE_PROJECTS=<glob> COMPARE_OUT=<dir>           슬라이드마다 web | quick 나란히 그림을 쓴다

import (
	"bytes"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/k2sebeom/balpyo/internal/pipeline"
	"github.com/k2sebeom/balpyo/render/geom"
	"github.com/k2sebeom/balpyo/render/quick"
	"github.com/k2sebeom/balpyo/render/web"
)

// TestDump는 한 프로젝트의 박스 rect를 브라우저와 내장 엔진 양쪽으로 찍는다.
func TestDump(t *testing.T) {
	dir := os.Getenv("DUMP_PROJECT")
	prefix := os.Getenv("DUMP_PREFIX")
	if dir == "" {
		t.Skip()
	}
	deck, _, err := pipeline.LoadDeck(pipeline.Options{ProjectDir: dir})
	if err != nil {
		t.Fatal(err)
	}
	files, _ := web.NewForMeasurement().Render(deck)
	tmp := t.TempDir()
	for _, name := range files.Paths() {
		p := filepath.Join(tmp, filepath.FromSlash(name))
		os.MkdirAll(filepath.Dir(p), 0o755)
		os.WriteFile(p, files[name], 0o644)
	}
	html := filepath.Join(tmp, web.IndexFile)
	want, err := geom.Measure(deck, html)
	if err != nil {
		t.Fatal(err)
	}
	got, err := quick.Measurer{}.Measure(deck, html)
	if err != nil {
		t.Fatal(err)
	}
	for _, wb := range want.Boxes {
		if !strings.HasPrefix(wb.ID, prefix) {
			continue
		}
		gb, _ := got.Box(wb.ID)
		t.Logf("%-14s %-6s web %-28s quick %-28s lines web %d quick %d", wb.ID, wb.El, wb.Rect, gb.Rect, len(wb.Lines), len(gb.Lines))
		if os.Getenv("DUMP_LINES") != "" {
			for i := range wb.Lines {
				q := geom.Rect{}
				if i < len(gb.Lines) {
					q = gb.Lines[i]
				}
				t.Logf("      line web %s quick %s", wb.Lines[i], q)
			}
		}
	}
}

// TestCompareDump는 web | quick 나란히 그림을 COMPARE_OUT에 쓴다 (COMPARE_PROJECTS: 쉼표 없는 glob).
func TestCompareDump(t *testing.T) {
	pattern, out := os.Getenv("COMPARE_PROJECTS"), os.Getenv("COMPARE_OUT")
	if pattern == "" {
		t.Skip()
	}
	session, err := geom.Launch("")
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	dirs, _ := filepath.Glob(pattern)
	for _, dir := range dirs {
		if _, err := os.Stat(filepath.Join(dir, "balpyo.json")); err != nil {
			continue
		}
		deck, diags, err := pipeline.LoadDeck(pipeline.Options{ProjectDir: dir})
		if err != nil || diags.Err() != nil {
			continue
		}
		start := time.Now()
		res, err := quick.Snapshot(deck, quick.Options{DPR: 1})
		if err != nil {
			t.Errorf("%s: %v", dir, err)
			continue
		}
		took := time.Since(start)
		files, err := web.NewWithBoxes(res.Geometry.RouteBox).Render(deck)
		if err != nil {
			t.Errorf("%s: %v", dir, err)
			continue
		}
		tmp := t.TempDir()
		for _, name := range files.Paths() {
			p := filepath.Join(tmp, filepath.FromSlash(name))
			os.MkdirAll(filepath.Dir(p), 0o755)
			os.WriteFile(p, files[name], 0o644)
		}
		if err := session.Open(filepath.Join(tmp, web.IndexFile)); err != nil {
			t.Fatal(err)
		}
		name := filepath.Base(dir)
		os.MkdirAll(filepath.Join(out, name), 0o755)
		for i, slide := range deck.Slides {
			rects, err := session.Rects(web.SlideIndexSelector(slide.Index))
			if err != nil || len(rects) == 0 {
				continue
			}
			buf, err := session.Clip(rects[0])
			if err != nil {
				continue
			}
			ref, err := png.Decode(bytes.NewReader(buf))
			if err != nil {
				continue
			}
			mine := res.Slides[i]
			w, h := ref.Bounds().Dx(), ref.Bounds().Dy()
			canvas := image.NewRGBA(image.Rect(0, 0, w*2+8, h))
			draw.Draw(canvas, canvas.Bounds(), image.Black, image.Point{}, draw.Src)
			draw.Draw(canvas, image.Rect(0, 0, w, h), ref, ref.Bounds().Min, draw.Src)
			draw.Draw(canvas, image.Rect(w+8, 0, w*2+8, h), mine, image.Point{}, draw.Src)
			var b bytes.Buffer
			png.Encode(&b, canvas)
			os.WriteFile(filepath.Join(out, name, fmt.Sprintf("%03d.png", i+1)), b.Bytes(), 0o644)
		}
		t.Logf("%-24s %2d slides, builtin %s", name, len(deck.Slides), took)
	}
}
