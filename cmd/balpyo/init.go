package main

import (
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/k2sebeom/balpyo/core"
	"github.com/k2sebeom/balpyo/internal/scaffold"
	"github.com/k2sebeom/balpyo/soml"
)

// runInit은 `balpyo init`이다: 빌드되는 프로젝트 하나를 만든다.
//
// 디렉토리는 위치 인자로 받는다 (`balpyo init my-talk`) — 만드는 명령이므로 "어디에"가 첫
// 인자인 것이 자연스럽고, 같은 뜻의 `--project`도 받아 다른 서브커맨드와 표기를 맞춘다.
// 둘을 함께 주면 멈춘다: 어느 쪽이 이겼는지 사람이 추측하게 두지 않는다.
func runInit(args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("init", flag.ContinueOnError)
	fs.SetOutput(stderr)
	projectDir := fs.String("project", "", "root of the project to create (same as the positional argument)")
	title := fs.String("title", "", "deck title (default: directory name)")
	author := fs.String("author", "", "presenter")
	theme := fs.String("theme", "", "theme (default "+core.DefaultConfig().Theme+
		", available themes: "+strings.Join(core.BuiltinThemes(), ", ")+")")
	aspect := fs.String("aspect", "", "aspect ratio (default "+string(soml.DefaultAspect)+")")
	lang := fs.String("lang", "", "deck language (default "+scaffold.DefaultLang+")")
	// 위치 인자는 플래그 앞·사이·뒤 어디에 와도 같은 자리로 모은다. Go의 flag는 첫 비플래그 인자에서
	// 파싱을 멈추므로, 그대로 두면 사람이 가장 먼저 쳐 볼 표기(`balpyo init my-talk --author 나`)가
	// "--author를 인자로 받았다"는 엉뚱한 에러가 된다.
	positional, err := parseInterleaved(fs, args)
	if err != nil {
		return err
	}
	dir := *projectDir
	switch {
	case len(positional) > 1:
		return fmt.Errorf("init takes a single directory: %v", positional)
	case len(positional) == 1 && dir != "":
		return fmt.Errorf("directory given twice: %q and --project %q", positional[0], dir)
	case len(positional) == 1:
		dir = positional[0]
	case dir == "":
		dir = "."
	}

	result, err := scaffold.Write(scaffold.Options{
		Dir:    dir,
		Title:  *title,
		Author: *author,
		Theme:  *theme,
		Aspect: soml.Aspect(*aspect),
		Lang:   *lang,
	})
	if err != nil {
		return err
	}

	fmt.Fprintf(stdout, "created project: %s\n", result.Dir)
	for _, name := range result.Written {
		fmt.Fprintf(stdout, "  %s\n", name)
	}
	// 다음 명령을 **그대로 복사할 수 있게** 찍는다. 첫 덱까지의 거리가 채택을 정하므로, 여기서
	// 문서를 찾아가게 만들지 않는다.
	fmt.Fprintln(stdout, "\nNext:")
	if dir == "." {
		fmt.Fprintln(stdout, "  balpyo watch                     reloads the browser tab on every edit")
		fmt.Fprintln(stdout, "  balpyo build --target html,pptx  writes index.html and a .pptx")
		return nil
	}
	fmt.Fprintf(stdout, "  balpyo watch --project %s\n", dir)
	fmt.Fprintf(stdout, "  balpyo build --project %s --target html,pptx\n", dir)
	return nil
}
