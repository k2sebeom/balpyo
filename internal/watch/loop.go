package watch

import (
	"context"
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"github.com/k2sebeom/balpyo/internal/pipeline"
	"github.com/k2sebeom/balpyo/render"
	"github.com/k2sebeom/balpyo/render/geom"
	"github.com/k2sebeom/balpyo/soml"
)

// Options는 감시 루프의 입력이다.
type Options struct {
	// Build는 빌드 옵션이다 — `balpyo build`가 받는 것과 같은 값이며, Measurer는 루프가 채운다
	// (웜 브라우저의 수명은 루프의 것이다).
	Build pipeline.Options
	// Interval이 0이면 DefaultInterval이다.
	Interval time.Duration
	// Log가 nil이면 로그를 버린다. 터미널에 찍히는 줄과 오버레이에 뜨는 값은 같은 Status에서
	// 나온다 — 두 곳이 다른 것을 말하면 저자는 어느 쪽을 믿을지 정해야 한다.
	Log io.Writer
}

// Runner는 감시 루프다: 빌드 → 서브 → 변경을 기다림 → 다시 빌드.
type Runner struct {
	opts    Options
	server  *Server
	watcher *Watcher
	session *warmSession
	outDir  string

	// mu는 빌드 하나를 직렬화한다. 빌드가 겹치면 두 빌드가 같은 출력 디렉토리에 쓰고 세대
	// 번호도 뒤섞인다 — 루프는 한 번에 하나씩만 빌드한다.
	mu         sync.Mutex
	generation int
}

// New는 감시기를 만든다. 산출물 디렉토리를 정해 서버를 세우고, 지금 트리를 기준선으로 잡는다.
//
// 첫 빌드는 하지 않는다 — 호출자가 서버를 띄운 **뒤에** 빌드해야 첫 빌드가 오래 걸리거나
// 실패해도 브라우저가 붙을 자리가 이미 있다.
func New(opts Options) (*Runner, error) {
	if opts.Interval <= 0 {
		opts.Interval = DefaultInterval
	}
	if opts.Log == nil {
		opts.Log = io.Discard
	}
	targets := opts.Build.Targets
	if len(targets) == 0 {
		targets = render.DefaultTargets()
	}
	if !hasHTML(targets) {
		// html이 없으면 볼 페이지가 없다. pptx만 감시하는 것은 "저장하면 파일이 갱신된다"일 뿐
		// 이고, 그것은 `balpyo build`를 다시 돌리는 일이다 (ADR 0046에서 열지 않은 항목이다).
		return nil, fmt.Errorf("watch는 html 타깃이 필요하다: --target에 html을 넣어라")
	}
	opts.Build.Targets = targets

	root := opts.Build.ProjectDir
	if root == "" {
		root = "."
	}
	// 프로젝트 루트를 **먼저** 본다: 출력 디렉토리를 만드는 일이 앞에 오면 `--project` 오타가
	// 디렉토리를 새로 만들고 "슬라이드가 없다"로 나타난다 (오타를 그 메시지에서 알아볼 수 없다).
	info, err := os.Stat(root)
	if err != nil {
		return nil, fmt.Errorf("감시할 프로젝트를 열 수 없다: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("감시할 프로젝트가 디렉토리가 아니다: %s", root)
	}
	outDir, err := pipeline.OutDir(opts.Build)
	if err != nil {
		return nil, err
	}
	skip, err := ProjectSkip(root, outDir)
	if err != nil {
		return nil, err
	}
	watcher, err := NewWatcher(root, skip)
	if err != nil {
		return nil, err
	}
	session := &warmSession{}
	opts.Build.Measurer = session
	return &Runner{
		opts:    opts,
		server:  NewServer(outDir),
		watcher: watcher,
		session: session,
		outDir:  outDir,
	}, nil
}

// Server는 산출물을 서브하는 서버다 — 주소를 잡고 서브하는 일은 호출자가 한다.
func (r *Runner) Server() *Server { return r.server }

// OutDir은 산출물 디렉토리다 (절대 경로).
func (r *Runner) OutDir() string { return r.outDir }

// Close는 웜 브라우저를 닫는다.
func (r *Runner) Close() { r.session.Close() }

// Build는 한 번 빌드하고 그 상태를 띄운다. 실패도 상태다 — 에러를 돌려주지 않는 이유는 감시
// 루프가 실패로 멈추지 않는다는 것이다 (저자는 고치는 중이고, 고치는 중의 파일은 자주 깨진다).
//
// 실패한 빌드는 산출물을 건드리지 않는다 (pipeline.Build가 쓰기 전에 멈춘다) — 마지막 성공본이
// 화면에 남고 그 위에 진단이 얹힌다.
func (r *Runner) Build() Status {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.generation++
	started := time.Now()
	result, err := pipeline.Build(r.opts.Build)
	status := Status{Generation: r.generation, BuiltAt: time.Now().Format("15:04:05")}
	if result != nil {
		status.Diagnostics = DiagnosticsOf(result.Diagnostics)
	}
	if err != nil {
		status.Error = err.Error()
	} else {
		status.OK = true
		status.Slides = len(result.Deck.Slides)
	}
	r.log(status, result, time.Since(started))
	r.server.Publish(status)
	return status
}

// Watch는 ctx가 끝날 때까지 감시하고 변경이 멈출 때마다 다시 빌드한다.
//
// 첫 빌드는 하지 않는다 — Build를 먼저 부르는 쪽이 호출자다.
func (r *Runner) Watch(ctx context.Context) error {
	ticker := time.NewTicker(r.opts.Interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			changed, err := r.watcher.Poll()
			if err != nil {
				// 스캔 실패는 프로젝트가 사라졌다는 뜻이다(이동·삭제). 조용히 계속 돌면 저자는
				// 고쳐도 아무 일이 없는 창을 본다 — 멈추고 이유를 말한다.
				return err
			}
			if changed {
				r.Build()
			}
		}
	}
}

// log는 빌드 하나를 터미널에 한 줄로 찍고, 진단은 그 아래에 붙인다.
//
// 진단을 **한 번 더** 터미널에 내는 이유: 저자가 브라우저를 아직 열지 않았거나 다른 화면을 보고
// 있을 수 있고, 그때 유일한 단서가 이 로그다.
func (r *Runner) log(status Status, result *pipeline.Result, elapsed time.Duration) {
	mark := "✓"
	summary := fmt.Sprintf("슬라이드 %d개", status.Slides)
	if result != nil && len(result.Written) > 0 {
		summary += fmt.Sprintf(" · 파일 %d개", len(result.Written))
	}
	if !status.OK {
		mark = "✗"
		summary = status.Error
	}
	fmt.Fprintf(r.opts.Log, "%s %s  %s (%s)\n", mark, status.BuiltAt, summary, elapsed.Round(time.Millisecond))
	if result != nil {
		for _, diag := range result.Diagnostics {
			fmt.Fprintf(r.opts.Log, "  %s\n", diag.String())
		}
	}
}

// hasHTML은 타깃 목록에 html이 있는지다.
func hasHTML(targets []render.Target) bool {
	for _, target := range targets {
		if target == render.TargetHTML {
			return true
		}
	}
	return false
}

// warmSession은 루프가 재사용하는 브라우저다 (ADR 0046).
//
// 지오메트리 패스 하나의 비용은 대부분 **기동**이다(~0.6초). 저작 루프가 루프마다 그 값을 내면
// `--check-layout`이나 pptx를 켠 감시는 쓸 수 없이 느려진다. 그래서 세션 하나를 들고 있는다.
//
// 처음 재는 순간에 띄운다 (New에서가 아니라): `watch --target html`은 재지 않으므로 그 루프는
// 브라우저를 아예 띄우지 않아야 한다 — 크로미움이 없는 환경에서도 웹 감시는 돌아간다.
type warmSession struct {
	mu   sync.Mutex
	live *geom.Session
}

// Measure는 웜 세션으로 잰다. 실패하면 **한 번** 다시 띄우고 재시도한다.
//
// 다시 띄우는 이유: 세션은 오래 산다(geom.sessionLifetime 10분이 CI의 안전망이다). 그 수명이
// 끝났거나 브라우저가 죽은 뒤의 첫 실패는 덱의 문제가 아니라 세션의 문제이고, 그때 저자에게
// "지오메트리 패스 실패"를 보여 주면 방금 고친 슬라이드를 의심하게 된다. 두 번째 실패는 덱의
// 문제로 보고 그대로 알린다 — 무한히 다시 띄우면 깨진 덱 하나가 브라우저를 계속 낳는다.
func (w *warmSession) Measure(deck *soml.Deck, htmlPath string) (*geom.Geometry, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.live == nil {
		live, err := geom.Launch("")
		if err != nil {
			return nil, err
		}
		w.live = live
	}
	g, err := w.live.Measure(deck, htmlPath)
	if err == nil {
		return g, nil
	}
	w.closeLocked()
	live, launchErr := geom.Launch("")
	if launchErr != nil {
		// 다시 띄우지도 못하면 처음 실패가 더 많은 것을 말한다.
		return nil, err
	}
	w.live = live
	return w.live.Measure(deck, htmlPath)
}

// Close는 웜 세션을 닫는다. 여러 번 불러도 괜찮다.
func (w *warmSession) Close() {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.closeLocked()
}

func (w *warmSession) closeLocked() {
	if w.live != nil {
		w.live.Close()
		w.live = nil
	}
}
