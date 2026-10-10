package main

import (
	"errors"
	"flag"
	"fmt"
	"io"

	"github.com/k2sebeom/balpyo/internal/pipeline"
	"github.com/k2sebeom/balpyo/soml"
)

// runCheck는 `balpyo check`다: 판정만 하고 아무것도 쓰지 않는다.
//
// 종료 코드 규약은 `build`와 **같다** — 0은 "에러가 없다"이고 경고는 0에 실려 나온다.
// `--strict`가 경고를 1로 바꾸는 유일한 채널이고, CI 레시피는
// `balpyo check --check-layout --strict`다. 규약을 하나로 두는 이유: 같은 프로젝트에서
// 두 명령의 코드가 갈리면 CI 로그를 읽는 사람이 어느 쪽이 진실인지 물어야 한다.
func runCheck(args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("check", flag.ContinueOnError)
	fs.SetOutput(stderr)
	projectDir := fs.String("project", ".", "project root")
	// build와 같은 플래그다: 테마가 배치를 바꾸므로(editorial의 표지는 아래에 앉는다)
	// 검사도 빌드가 볼 테마를 봐야 한다 — 두 명령이 다른 테마를 보면 CI의 판정이 거짓이 된다.
	theme := fs.String("theme", "", "theme (default: theme in balpyo.json — built-in, project themes/, or installed)")
	strict := fs.Bool("strict", false, "promote warnings to errors")
	checkLayout := fs.Bool("check-layout", false, "also check layout (overflow, off-stage, overlap — launches a browser)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() > 0 {
		return fmt.Errorf("check takes no arguments: %v", fs.Args())
	}
	// `--target`을 받지 않는다: 진단은 타깃에 의존하지 않는다 (배치 경고까지 잰 박스에서
	// 나온다). 렌더 단계에서만 나는 실패는 `build`의 판정이다.
	result, err := pipeline.Check(pipeline.Options{
		ProjectDir:  *projectDir,
		Theme:       *theme,
		ThemeHome:   themeHome(),
		Strict:      *strict,
		CheckLayout: *checkLayout,
	})
	printDiags(stderr, result.Diagnostics)
	if err != nil {
		var de *soml.DiagnosticsError
		if errors.As(err, &de) {
			return fmt.Errorf("check failed: %s", count(de.Count(), "error"))
		}
		return err
	}

	// 무엇을 봤는지까지 말한다. "깨끗하다"와 "배치는 보지 않았다"는 다른 문장이고, 그 차이를
	// 숨기면 사람은 재지 않은 판정을 다 본 판정으로 읽는다.
	scope := "layout not checked (--check-layout)"
	if result.Measured {
		scope = "layout checked"
	}
	if warnings := len(result.Diagnostics); warnings > 0 {
		fmt.Fprintf(stdout, "%s · %s · %s\n", count(len(result.Deck.Slides), "slide"), count(warnings, "warning"), scope)
		return nil
	}
	fmt.Fprintf(stdout, "%s · no diagnostics · %s\n", count(len(result.Deck.Slides), "slide"), scope)
	return nil
}

// count는 요약 줄의 수량을 영어 단수·복수로 쓴다 ("1 slide", "3 slides").
func count(n int, noun string) string {
	if n == 1 {
		return fmt.Sprintf("%d %s", n, noun)
	}
	return fmt.Sprintf("%d %ss", n, noun)
}
