package main

import (
	"errors"
	"flag"
	"fmt"
	"io"

	"github.com/k2sebeom/balpyo/internal/pipeline"
	"github.com/k2sebeom/balpyo/internal/preview"
	"github.com/k2sebeom/balpyo/render"
	"github.com/k2sebeom/balpyo/soml"
)

// runPreview는 `balpyo preview`다: 빌드하고, PPTX를 그림으로 되돌려 웹과 나란히 놓는다.
//
// **판정하지 않는다**. 두 타깃의 계약은 컴포넌트·레이아웃 동일성이고 그것은 `go test`가
// 본다 — 이 명령은 그림을 나란히 놓을 뿐이다. `--target`을 주지 않으면 html과, 래스터 도구가 있을
// 때 pptx를 그린다. 1로 끝나는 경우는 빌드가 실패했거나 **명시한** pptx의 도구가 없는 경우뿐이다 —
// 둘 다 "검수할 것이 만들어지지 않았다"는 뜻이다.
func runPreview(args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("preview", flag.ContinueOnError)
	fs.SetOutput(stderr)
	// 플래그 이름과 뜻은 build와 같다 — `--out`은 여기서도 **빌드 산출물**이 가는 곳이다.
	// 갤러리에 같은 이름을 쓰면 `--out dist`가 어느 디렉토리를 뜻하는지 사람이 매번 확인해야 한다.
	targetList := fs.String("target", "", "renders to review (html, pptx — comma-separated; default: html, plus pptx when the tools are available)")
	outDir := fs.String("out", "", "build output directory (default: outDir in balpyo.json)")
	projectDir := fs.String("project", ".", "project root")
	theme := fs.String("theme", "", "theme (default: theme in balpyo.json — built-in, project themes/, or installed)")
	strict := fs.Bool("strict", false, "promote warnings to errors")
	galleryDir := fs.String("gallery", "", "review gallery directory (default: <project>/"+preview.DefaultDirName+")")
	if err := parseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() > 0 {
		return fmt.Errorf("preview takes no arguments: %v", fs.Args())
	}
	// 주지 않으면 비워 둔다 — 기본 렌더에서만 도구가 없을 때 PPTX 열을 뺀다 (preview.Options.Renders).
	// 빈 값을 **준** 것은 주지 않은 것과 다르다 — 그것은 잘못된 입력이다.
	var targets []render.Target
	if flagSet(fs, "target") {
		parsed, err := render.ParseTargets(*targetList)
		if err != nil {
			return err
		}
		targets = parsed
	}

	result, err := preview.Run(preview.Options{
		Build: pipeline.Options{
			ProjectDir: *projectDir,
			OutDir:     *outDir,
			Theme:      *theme,
			ThemeHome:  themeHome(),
			Strict:     *strict,
			// 검수는 배치를 **항상** 본다: 눈으로 볼 자리를 찾으러 온 사람에게 이미 숫자로 아는
			// 넘침·이탈을 말하지 않을 이유가 없다 (브라우저는 어차피 떠 있다).
			CheckLayout: true,
		},
		Renders: targets,
		Dir:     *galleryDir,
		Log:     stdout,
	})
	// 경고는 검수가 실패해도 사람에게 보여야 한다 — 승격된 경고가 곧 그 실패의 이유다.
	// (빌드를 시작하기 전에 멈춘 경우 result는 nil이다 — 도구가 없다, 타깃 이름이 틀렸다.)
	if result != nil {
		printDiags(stderr, result.Diagnostics)
	}
	if err != nil {
		var de *soml.DiagnosticsError
		if errors.As(err, &de) {
			return fmt.Errorf("could not build the output to review: %s", count(de.Count(), "error"))
		}
		return err
	}

	fmt.Fprintf(stdout, "%s → %s\n", count(len(result.Slides), "slide"), result.Index)
	// 갤러리로 판정할 수 없는 확인이 남는다 — PPTX를 PowerPoint/Keynote로 열어 텍스트 박스가
	// 진짜 텍스트 박스인지, 테마 색 편집이 먹는지 보는 일. 어느 엔진의 렌더도 그 확인을 대신하지
	// 못한다 (그라디언트 선 · 다중 스톱 같은 자리에서 뷰어끼리 갈라진다).
	if hasTarget(result.Renders, render.TargetPPTX) {
		fmt.Fprintf(stdout, "open the PPTX in PowerPoint or Keynote to review it: %s\n", result.OutDir)
	}
	return nil
}

// hasTarget은 목록에 타깃이 있는지다.
func hasTarget(targets []render.Target, want render.Target) bool {
	for _, target := range targets {
		if target == want {
			return true
		}
	}
	return false
}

// flagSet은 명령줄에서 그 플래그를 줬는지다 (기본값과 같은 값을 준 경우도 준 것이다).
func flagSet(fs *flag.FlagSet, name string) bool {
	set := false
	fs.Visit(func(f *flag.Flag) {
		if f.Name == name {
			set = true
		}
	})
	return set
}
