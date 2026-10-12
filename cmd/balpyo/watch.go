package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"net"
	"strings"

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

// checkLoopbackAddr는 감시 서버 주소가 루프백인지 본다.
//
// 기본값만 루프백이면 고정이라고 할 수 없다: `--addr 0.0.0.0:…`이나 `:…`(호스트 생략은 모든
// 인터페이스다)가 그대로 통하면 같은 네트워크의 누구나 덱을 연다. 그래서 호스트는 루프백 IP
// 리터럴이나 localhost만 받는다 — 다른 이름은 무엇으로 풀릴지 여기서 알 수 없으므로 받지 않는다.
func checkLoopbackAddr(addr string) error {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return fmt.Errorf("invalid --addr %q: %w", addr, err)
	}
	if strings.EqualFold(host, "localhost") {
		return nil
	}
	if ip := net.ParseIP(host); ip != nil && ip.IsLoopback() {
		return nil
	}
	return fmt.Errorf("--addr %q is not a loopback address: the watch server only listens on 127.0.0.1, ::1 or localhost (share a deck by publishing the built output)", addr)
}

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
	addr := fs.String("addr", defaultWatchAddr, "watch server address (loopback only)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() > 0 {
		return fmt.Errorf("watch takes no arguments: %v", fs.Args())
	}
	// 주소는 첫 빌드 준비보다 먼저 본다 — 바깥으로 열리는 주소는 한 번이라도 잡으면 이미 열린
	// 것이므로, 잡기 전에 멈춰야 한다.
	if err := checkLoopbackAddr(*addr); err != nil {
		return err
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
