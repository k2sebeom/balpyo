// Command balpyo는 SOML 슬라이드를 웹과 네이티브 PPTX로 렌더하는 CLI다.
//
// 바이너리 하나가 모든 서브커맨드를 담는다는 계약을 유지한다. 지금의 표면은
// `init`·`build`·`check`·`watch`·`preview`·`snap`·`theme`·`version`이며, new/server/format/lsp는 아직이다.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/k2sebeom/balpyo/internal/pipeline"
	"github.com/k2sebeom/balpyo/render"
	"github.com/k2sebeom/balpyo/soml"
)

func main() {
	if err := run(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// run은 CLI 본체다. 종료 코드 대신 error를 돌려 테스트가 그대로 호출할 수 있게 한다.
func run(args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		usage(stderr)
		return fmt.Errorf("a subcommand is required")
	}
	switch args[0] {
	case "init":
		return runInit(args[1:], stdout, stderr)
	case "build":
		return runBuild(args[1:], stdout, stderr)
	case "check":
		return runCheck(args[1:], stdout, stderr)
	case "watch":
		// 종료 신호를 ctx로 받는다 — Ctrl+C가 웜 브라우저를 닫고(runner.Close) 서버를 내린 뒤
		// 돌아와야 다음 watch가 같은 포트를 잡고 프로세스가 남지 않는다.
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()
		return runWatch(ctx, args[1:], stdout, stderr)
	case "preview":
		return runPreview(args[1:], stdout, stderr)
	case "snap":
		return runSnap(args[1:], stdout, stderr)
	case "theme":
		return runTheme(args[1:], stdout, stderr)
	case "version":
		fmt.Fprintln(stdout, versionString())
		return nil
	case "help", "-h", "--help":
		usage(stdout)
		return nil
	default:
		usage(stderr)
		return fmt.Errorf("unknown subcommand %q", args[0])
	}
}

func usage(w io.Writer) {
	fmt.Fprint(w, `balpyo — renders SOML slides to the web and to native, editable PPTX.

Usage:
  balpyo init [dir]      create a project that builds out of the box
  balpyo build [options] build the slides
  balpyo check [options] run diagnostics without writing output
  balpyo watch [options] rebuild on change and reload the browser
  balpyo preview [opts]  build a review gallery with web and PPTX side by side
  balpyo snap [options]  draw the slides to PNG/GIF with the builtin renderer (no browser)
  balpyo theme <command> install themes from GitHub or a local directory and manage installed themes
  balpyo version         print the version

init options:
  --project .           root of the project to create (same as the positional argument)
  --title TITLE         deck title (default: directory name)
  --author NAME         presenter
  --theme minimal       theme (built-in themes only)
  --aspect 16:9         aspect ratio
  --lang en             deck language

build options:
  --target html,pptx    render targets (default html)
  --out dist            output directory (default: outDir in balpyo.json)
  --project .           project root
  --theme ink           theme (default: theme in balpyo.json — built-in, project, or installed)
  --strict              promote warnings to errors
  --check-layout        check layout (overflow, off-stage, overlap — launches a browser)

check options:
  --project .           project root
  --theme ink           theme (as in build — the theme changes the layout, so it changes the verdict)
  --strict              promote warnings to errors
  --check-layout        also check layout (launches a browser)

  Same verdict as build, without writing output. CI: balpyo check --check-layout --strict

watch options:
  all build options (the html target is required), plus
  --addr 127.0.0.1:4600 watch server address

preview options:
  --target html,pptx    renders to review (default html,pptx)
  --out dist            build output directory (default: outDir in balpyo.json)
  --project .           project root
  --theme ink           theme (as in build — this is how to review one deck in two themes)
  --gallery preview     review gallery directory
  --strict              promote warnings to errors

  Reviewing pptx needs OnlyOffice Document Builder, or else LibreOffice (soffice) and
  pdftoppm (poppler-utils).

snap options:
  --project .           project root
  --theme ink           theme (as in build)
  --out DIR             image directory (default: <outDir>/snap)
  --slides 1,3-5        slides to draw (default all)
  --scale 1             device pixels per stage px (2 for sharper images)
  --gif                 also write deck.gif, one frame per reveal step
  --outline             draw the measured box and id of every element
  --strict              promote warnings to errors

  A fast preview for writing: no browser, no OnlyOffice. Layout matches the browser box by
  box, but line breaks and glyphs may differ — build output stays the reference.

theme commands:
  balpyo theme install <GitHub URL | path> [name...] [--force]
                        install the themes in a repository or directory (all of them if no name is given)
  balpyo theme list [--project .]
                        list the available themes with their sources
  balpyo theme info <name> [--project .]
                        show where a theme comes from and print its README.md
  balpyo theme remove <name...>
                        remove installed themes
  balpyo theme update [name...]
                        re-fetch installed themes from their recorded sources

  Installed themes live in ~/.balpyo/themes/<name>/ (override with BALPYO_HOME).
  Use that name as theme in balpyo.json.
`)
}

// runBuild는 `balpyo build`다. 플래그가 balpyo.json을 덮어쓴다.
func runBuild(args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("build", flag.ContinueOnError)
	fs.SetOutput(stderr)
	targetList := fs.String("target", "", "render targets (html, pptx — comma-separated)")
	outDir := fs.String("out", "", "output directory (default: outDir in balpyo.json)")
	projectDir := fs.String("project", ".", "project root")
	// 테마 플래그가 설정을 덮는다 — 한 덱을 두 테마로 빌드해 견주는 것이 테마를
	// 고르는 방법이고, 그때 `balpyo.json`을 고쳐 되돌리는 절차는 되돌리기를 잊게 만든다.
	theme := fs.String("theme", "", "theme (default: theme in balpyo.json — built-in, project themes/, or installed)")
	strict := fs.Bool("strict", false, "promote warnings to errors")
	// pptx 타깃은 이미 재므로 배치 검사를 공짜로 받는다 — 이 플래그가 뜻을 갖는 자리는 html만
	// 쓰는 저작 루프다. 기본을 끈 이유는 브라우저 기동이 빌드의 가장 큰 고정 비용이라는 것이다.
	checkLayout := fs.Bool("check-layout", false, "check layout (overflow, off-stage, overlap)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() > 0 {
		return fmt.Errorf("build takes no arguments: %v", fs.Args())
	}

	opts := pipeline.Options{
		ProjectDir:  *projectDir,
		OutDir:      *outDir,
		Theme:       *theme,
		ThemeHome:   themeHome(),
		Strict:      *strict,
		CheckLayout: *checkLayout,
	}
	if *targetList != "" {
		targets, err := render.ParseTargets(*targetList)
		if err != nil {
			return err
		}
		opts.Targets = targets
	}

	result, err := pipeline.Build(opts)
	// 경고는 빌드가 실패해도 사람에게 보여야 한다.
	printDiagnostics(stderr, result)
	if err != nil {
		// 진단 에러의 상세는 printDiagnostics가 이미 냈다 — 다시 통째로 찍지 않고
		// 개수만 요약한다. 진단이 아닌 에러(설정·파일 등)는 그대로 올린다.
		var de *soml.DiagnosticsError
		if errors.As(err, &de) {
			return fmt.Errorf("build failed: %s", count(de.Count(), "error"))
		}
		return err
	}
	fmt.Fprintf(stdout, "%s → %s\n", count(len(result.Deck.Slides), "slide"), result.OutDir)
	for _, name := range result.Written {
		fmt.Fprintf(stdout, "  %s\n", filepath.ToSlash(name))
	}
	return nil
}

// printDiagnostics는 빌드 결과의 진단을 찍는다.
func printDiagnostics(w io.Writer, result *pipeline.Result) {
	if result == nil {
		return
	}
	printDiags(w, result.Diagnostics)
}

// printDiags는 진단을 사람이 읽을 순서로 찍는다. 빌드 결과를 들고 있지 않은 호출자
// (`balpyo preview`)도 같은 모양으로 찍어야 하므로 갈라 둔다.
func printDiags(w io.Writer, diags soml.Diagnostics) {
	for _, d := range diags {
		fmt.Fprintln(w, d.String())
	}
}
