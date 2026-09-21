// Command balpyo는 SOML 슬라이드를 웹과 네이티브 PPTX로 렌더하는 CLI다.
//
// 바이너리 하나가 모든 서브커맨드를 담는다는 계약을 유지한다 (PRD §10). 지금의 표면은
// `init`·`build`·`check`·`watch`·`preview`·`version`이며, new/server/format/lsp는 아직이다.
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
		return fmt.Errorf("서브커맨드가 필요하다")
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
  balpyo init [디렉토리] 빌드되는 프로젝트 하나를 만든다
  balpyo build [옵션]   슬라이드를 빌드한다
  balpyo check [옵션]   산출물 없이 진단만 한다
  balpyo watch [옵션]   고치면 다시 빌드하고 브라우저를 새로고침한다
  balpyo preview [옵션] 웹과 PPTX를 나란히 놓은 검수 갤러리를 만든다
  balpyo version        버전을 출력한다

init 옵션:
  --project .           만들 프로젝트 루트 (위치 인자와 같은 뜻)
  --title 제목          덱 제목 (기본: 디렉토리 이름)
  --author 이름         발표자
  --theme minimal       테마 (내장 테마만 받는다)
  --aspect 16:9         종횡비
  --lang ko             덱 언어

build 옵션:
  --target html,pptx    렌더 타깃 (기본 html)
  --out dist            출력 디렉토리 (기본 balpyo.json의 outDir)
  --project .           프로젝트 루트
  --theme ink           테마 (기본 balpyo.json의 theme — 내장 테마만 받는다)
  --strict              경고를 에러로 승격한다
  --check-layout        배치를 검사한다 (넘침·스테이지 이탈·겹침 — 브라우저를 띄운다)

check 옵션:
  --project .           프로젝트 루트
  --theme ink           테마 (build와 같다 — 테마가 배치를 바꾸므로 판정도 바뀐다)
  --strict              경고를 에러로 승격한다
  --check-layout        배치도 검사한다 (브라우저를 띄운다)

  판정은 build와 같고 산출물을 쓰지 않는다. CI: balpyo check --check-layout --strict

watch 옵션:
  build의 옵션 전부 (html 타깃은 필수) 그리고
  --addr 127.0.0.1:4600 감시 서버 주소

preview 옵션:
  --target html,pptx    검수할 렌더 (기본 html,pptx)
  --out dist            빌드 출력 디렉토리 (기본 balpyo.json의 outDir)
  --project .           프로젝트 루트
  --theme ink           테마 (build와 같다 — 한 덱을 두 테마로 검수하는 방법이다)
  --gallery preview     검수 갤러리 디렉토리
  --strict              경고를 에러로 승격한다

  pptx를 검수하려면 LibreOffice(soffice)와 pdftoppm(poppler-utils)이 필요하다.
`)
}

// runBuild는 `balpyo build`다. 플래그가 balpyo.json을 덮어쓴다.
func runBuild(args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("build", flag.ContinueOnError)
	fs.SetOutput(stderr)
	targetList := fs.String("target", "", "렌더 타깃 (html, pptx — 쉼표로 여러 개)")
	outDir := fs.String("out", "", "출력 디렉토리 (기본: balpyo.json의 outDir)")
	projectDir := fs.String("project", ".", "프로젝트 루트")
	// 테마 플래그가 설정을 덮는다 (ADR 0052) — 한 덱을 두 테마로 빌드해 견주는 것이 테마를
	// 고르는 방법이고, 그때 `balpyo.json`을 고쳐 되돌리는 절차는 되돌리기를 잊게 만든다.
	theme := fs.String("theme", "", "테마 (기본: balpyo.json의 theme — 내장 테마만 받는다)")
	strict := fs.Bool("strict", false, "경고를 에러로 승격한다")
	// pptx 타깃은 이미 재므로 배치 검사를 공짜로 받는다 — 이 플래그가 뜻을 갖는 자리는 html만
	// 쓰는 저작 루프다. 기본을 끈 이유는 브라우저 기동이 빌드의 가장 큰 고정 비용이라는 것이다
	// (ADR 0039).
	checkLayout := fs.Bool("check-layout", false, "배치를 검사한다 (넘침·스테이지 이탈·겹침)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() > 0 {
		return fmt.Errorf("build는 인자를 받지 않는다: %v", fs.Args())
	}

	opts := pipeline.Options{
		ProjectDir:  *projectDir,
		OutDir:      *outDir,
		Theme:       *theme,
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
			return fmt.Errorf("빌드 실패: 에러 %d건", de.Count())
		}
		return err
	}
	fmt.Fprintf(stdout, "슬라이드 %d장 → %s\n", len(result.Deck.Slides), result.OutDir)
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
