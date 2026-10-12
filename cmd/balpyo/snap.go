package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/k2sebeom/balpyo/internal/pipeline"
	"github.com/k2sebeom/balpyo/internal/snapshot"
	"github.com/k2sebeom/balpyo/soml"
)

// runSnap은 `balpyo snap`이다: 브라우저 없이 내장 렌더러로 슬라이드를 그림으로 그린다.
func runSnap(args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("snap", flag.ContinueOnError)
	fs.SetOutput(stderr)
	projectDir := fs.String("project", ".", "project root")
	theme := fs.String("theme", "", "theme (default: theme in balpyo.json — built-in, project themes/, or installed)")
	out := fs.String("out", "", "image directory (default: <outDir>/snap)")
	slides := fs.String("slides", "", "slides to draw, 1-based (e.g. 1,3-5; default all)")
	scale := fs.Float64("scale", 1, "device pixels per stage px (2 for sharper images)")
	makeGIF := fs.Bool("gif", false, "also write deck.gif with one frame per reveal step")
	outline := fs.Bool("outline", false, "draw the measured box and id of every element")
	strict := fs.Bool("strict", false, "promote warnings to errors")
	if err := parseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() > 0 {
		return fmt.Errorf("snap takes no arguments: %v", fs.Args())
	}
	list, err := parseSlideList(*slides)
	if err != nil {
		return err
	}
	if *scale <= 0 || *scale > 8 {
		return fmt.Errorf("--scale %v is out of range (0 < scale ≤ 8)", *scale)
	}
	res, err := snapshot.Run(snapshot.Options{
		Build: pipeline.Options{
			ProjectDir: *projectDir,
			Theme:      *theme,
			ThemeHome:  themeHome(),
			Strict:     *strict,
		},
		Dir:     *out,
		Slides:  list,
		Scale:   *scale,
		GIF:     *makeGIF,
		Outline: *outline,
	})
	if res != nil {
		printDiags(stderr, res.Diagnostics)
	}
	if err != nil {
		var de *soml.DiagnosticsError
		if errors.As(err, &de) {
			return fmt.Errorf("snap failed: %s", count(de.Count(), "error"))
		}
		return err
	}
	fmt.Fprintf(stdout, "%s → %s (%s, builtin renderer)\n",
		count(len(res.Written), "image"), res.Dir, res.Elapsed.Round(1e6))
	for _, name := range res.Written {
		fmt.Fprintf(stdout, "  %s\n", filepath.ToSlash(name))
	}
	return nil
}

// parseSlideList는 `1,3-5` 꼴의 슬라이드 번호 목록을 읽는다.
func parseSlideList(v string) ([]int, error) {
	var out []int
	for _, part := range strings.Split(v, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		lo, hi, isRange := strings.Cut(part, "-")
		a, err := strconv.Atoi(strings.TrimSpace(lo))
		if err != nil {
			return nil, fmt.Errorf("--slides: %q is not a slide number", part)
		}
		b := a
		if isRange {
			if b, err = strconv.Atoi(strings.TrimSpace(hi)); err != nil || b < a {
				return nil, fmt.Errorf("--slides: %q is not a slide range", part)
			}
		}
		for n := a; n <= b; n++ {
			out = append(out, n)
		}
	}
	return out, nil
}
