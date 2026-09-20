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
	// 위치 인자를 **파싱 앞에서** 떼어낸다. Go의 flag는 첫 비플래그 인자에서 파싱을 멈추므로,
	// 이 한 줄이 없으면 사람이 가장 먼저 쳐 볼 표기(`balpyo init my-talk --author 나`)가
	// "--author를 인자로 받았다"는 엉뚱한 에러가 된다.
	var lead string
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		lead, args = args[0], args[1:]
	}

	fs := flag.NewFlagSet("init", flag.ContinueOnError)
	fs.SetOutput(stderr)
	projectDir := fs.String("project", "", "만들 프로젝트 루트 (위치 인자와 같은 뜻)")
	title := fs.String("title", "", "덱 제목 (기본: 디렉토리 이름)")
	author := fs.String("author", "", "발표자")
	theme := fs.String("theme", "", "테마 (기본 "+core.DefaultConfig().Theme+
		", 쓸 수 있는 테마: "+strings.Join(core.BuiltinThemes(), ", ")+")")
	aspect := fs.String("aspect", "", "종횡비 (기본 "+string(soml.DefaultAspect)+")")
	lang := fs.String("lang", "", "덱 언어 (기본 "+scaffold.DefaultLang+")")
	if err := fs.Parse(args); err != nil {
		return err
	}
	// 뒤에 붙은 위치 인자(`balpyo init --author 나 my-talk`)도 같은 자리로 모은다.
	positional := fs.Args()
	if lead != "" {
		positional = append([]string{lead}, positional...)
	}
	dir := *projectDir
	switch {
	case len(positional) > 1:
		return fmt.Errorf("init은 디렉토리 하나만 받는다: %v", positional)
	case len(positional) == 1 && dir != "":
		return fmt.Errorf("디렉토리를 두 번 줬다: %q와 --project %q", positional[0], dir)
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

	fmt.Fprintf(stdout, "프로젝트를 만들었다: %s\n", result.Dir)
	for _, name := range result.Written {
		fmt.Fprintf(stdout, "  %s\n", name)
	}
	// 다음 명령을 **그대로 복사할 수 있게** 찍는다. 첫 덱까지의 거리가 채택을 정하므로, 여기서
	// 문서를 찾아가게 만들지 않는다.
	fmt.Fprintln(stdout, "\n다음:")
	if dir == "." {
		fmt.Fprintln(stdout, "  balpyo watch                     고치면 브라우저 탭이 새로고침된다")
		fmt.Fprintln(stdout, "  balpyo build --target html,pptx  index.html과 .pptx를 낸다")
		return nil
	}
	fmt.Fprintf(stdout, "  balpyo watch --project %s\n", dir)
	fmt.Fprintf(stdout, "  balpyo build --project %s --target html,pptx\n", dir)
	return nil
}
