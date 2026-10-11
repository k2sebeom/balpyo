package main

import (
	"context"
	"flag"
	"fmt"
	"io"

	"github.com/k2sebeom/balpyo/internal/pipeline"
	"github.com/k2sebeom/balpyo/internal/watch"
	"github.com/k2sebeom/balpyo/render"
)

// defaultWatchAddr는 감시 서버의 기본 주소다.
//
// 루프백에 고정한 이유: 이 서버는 저자 한 사람의 도구이고, 산출물은 아직 공개되지
// 않은 발표 자료다. 0.0.0.0에 붙으면 같은 카페 와이파이의 누구나 볼 수 있다 — 공유는 빌드한
// 산출물을 올리는 일이고, 그 길은 이미 있다.
const defaultWatchAddr = "127.0.0.1:4600"

// runWatch는 `balpyo watch`다: 빌드하고 서브하고, 고치면 다시 빌드한다.
//
// ctx를 받는 이유: 이 명령은 Ctrl+C까지 돌아오지 않으므로 종료 신호가 인자여야 테스트가 같은
// 흐름을 돌릴 수 있다 (main이 signal.NotifyContext를 준다).
func runWatch(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("watch", flag.ContinueOnError)
	fs.SetOutput(stderr)
	// 플래그는 build와 같은 이름·같은 뜻이다 — 저작 루프에서 쓰는 명령이 빌드와 다른 옵션을
	// 요구하면 저자는 두 벌을 외워야 한다.
	targetList := fs.String("target", "", "render targets (html, pptx — comma-separated; html is required)")
	outDir := fs.String("out", "", "output directory (default: outDir in balpyo.json)")
	projectDir := fs.String("project", ".", "project root")
	theme := fs.String("theme", "", "theme (default: theme in balpyo.json — built-in, project themes/, or installed)")
	strict := fs.Bool("strict", false, "promote warnings to errors")
	checkLayout := fs.Bool("check-layout", false, "check layout (overflow, off-stage, overlap)")
	addr := fs.String("addr", defaultWatchAddr, "watch server address")
	if err := parseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() > 0 {
		return fmt.Errorf("watch takes no arguments: %v", fs.Args())
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

	runner, err := watch.New(watch.Options{Build: opts, Log: stdout})
	if err != nil {
		return err
	}
	defer runner.Close()

	// 주소를 **먼저** 잡는다: 포트가 이미 쓰이고 있으면(watch를 두 번 띄웠다) 첫 빌드를 기다린
	// 뒤에 그 사실을 아는 것은 낭비다.
	ln, err := runner.Server().Listen(*addr)
	if err != nil {
		return err
	}
	fmt.Fprintf(stdout, "watching %s → http://%s  (Ctrl+C to stop)\n", *projectDir, ln.Addr())

	served := make(chan error, 1)
	go func() { served <- runner.Server().Serve(ctx, ln) }()
	// 첫 빌드는 서버가 뜬 뒤에 한다 — 첫 빌드가 실패해도(고치는 중이었다) 브라우저가 붙을
	// 자리가 있고, 그 자리에서 이유를 읽는다.
	runner.Build()

	watchErr := runner.Watch(ctx)
	// 서버가 먼저 죽었으면 그 이유가 더 많은 것을 말한다 (감시 루프는 그때 ctx로 조용히 끝난다).
	select {
	case err := <-served:
		if err != nil {
			return err
		}
	default:
	}
	return watchErr
}
