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
	projectDir := fs.String("project", ".", "프로젝트 루트")
	// build와 같은 플래그다: 테마가 배치를 바꾸므로(editorial의 표지는 아래에 앉는다)
	// 검사도 빌드가 볼 테마를 봐야 한다 — 두 명령이 다른 테마를 보면 CI의 판정이 거짓이 된다.
	theme := fs.String("theme", "", "테마 (기본: balpyo.json의 theme — 내장·프로젝트 themes/·설치한 테마)")
	strict := fs.Bool("strict", false, "경고를 에러로 승격한다")
	checkLayout := fs.Bool("check-layout", false, "배치도 검사한다 (넘침·스테이지 이탈·겹침 — 브라우저를 띄운다)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() > 0 {
		return fmt.Errorf("check는 인자를 받지 않는다: %v", fs.Args())
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
			return fmt.Errorf("검사 실패: 에러 %d건", de.Count())
		}
		return err
	}

	// 무엇을 봤는지까지 말한다. "깨끗하다"와 "배치는 보지 않았다"는 다른 문장이고, 그 차이를
	// 숨기면 사람은 재지 않은 판정을 다 본 판정으로 읽는다.
	scope := "배치는 보지 않았다 (--check-layout)"
	if result.Measured {
		scope = "배치까지 봤다"
	}
	if warnings := len(result.Diagnostics); warnings > 0 {
		fmt.Fprintf(stdout, "슬라이드 %d장 · 경고 %d건 · %s\n", len(result.Deck.Slides), warnings, scope)
		return nil
	}
	fmt.Fprintf(stdout, "슬라이드 %d장 · 진단 없다 · %s\n", len(result.Deck.Slides), scope)
	return nil
}
