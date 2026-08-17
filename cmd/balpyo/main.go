// Command balpyo는 SOML 슬라이드를 웹과 네이티브 PPTX로 렌더하는 CLI다.
//
// 바이너리 하나가 모든 서브커맨드를 담는다는 계약을 유지한다 (PRD §10). Iteration 0의
// 표면은 `build`와 `version` 둘뿐이며, init/new/server/check/format/lsp는 Phase 2다.
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime/debug"

	"github.com/k2sebeom/balpyo/internal/pipeline"
	"github.com/k2sebeom/balpyo/render"
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
		return fmt.Errorf("서브커맨드가 필요하다")
	}
	switch args[0] {
	case "build":
		return runBuild(args[1:], stdout, stderr)
	case "version":
		fmt.Fprintln(stdout, versionString())
		return nil
	case "help", "-h", "--help":
		usage(stdout)
		return nil
	default:
		usage(stderr)
		return fmt.Errorf("알 수 없는 서브커맨드 %q", args[0])
	}
}

func usage(w io.Writer) {
	fmt.Fprint(w, `balpyo — SOML 슬라이드를 웹과 편집 가능한 네이티브 PPTX로 렌더한다.

사용법:
  balpyo build [옵션]   슬라이드를 빌드한다
  balpyo version        버전을 출력한다

build 옵션:
  --target html,pptx    렌더 타깃 (기본 html)
  --out dist            출력 디렉토리 (기본 balpyo.json의 outDir)
  --project .           프로젝트 루트
  --strict              경고를 에러로 승격한다
`)
}

// runBuild는 `balpyo build`다. 플래그가 balpyo.json을 덮어쓴다.
func runBuild(args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("build", flag.ContinueOnError)
	fs.SetOutput(stderr)
	targetList := fs.String("target", "", "렌더 타깃 (html, pptx — 쉼표로 여러 개)")
	outDir := fs.String("out", "", "출력 디렉토리 (기본: balpyo.json의 outDir)")
	projectDir := fs.String("project", ".", "프로젝트 루트")
	strict := fs.Bool("strict", false, "경고를 에러로 승격한다")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() > 0 {
		return fmt.Errorf("build는 인자를 받지 않는다: %v", fs.Args())
	}

	opts := pipeline.Options{ProjectDir: *projectDir, OutDir: *outDir, Strict: *strict}
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
		return err
	}
	fmt.Fprintf(stdout, "슬라이드 %d장 → %s\n", len(result.Deck.Slides), result.OutDir)
	for _, name := range result.Written {
		fmt.Fprintf(stdout, "  %s\n", filepath.ToSlash(name))
	}
	return nil
}

// printDiagnostics는 진단을 사람이 읽을 순서로 찍는다.
func printDiagnostics(w io.Writer, result *pipeline.Result) {
	if result == nil {
		return
	}
	for _, d := range result.Diagnostics {
		fmt.Fprintln(w, d.String())
	}
}

// version은 릴리스 빌드에서 -ldflags로 주입한다. 개발 빌드는 모듈 정보에서 유도한다.
var version = ""

func versionString() string {
	if version != "" {
		return "balpyo " + version
	}
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" {
		return "balpyo " + info.Main.Version
	}
	return "balpyo (devel)"
}
