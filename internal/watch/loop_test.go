package watch

import (
	"context"
	"io"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/k2sebeom/balpyo/internal/pipeline"
	"github.com/k2sebeom/balpyo/internal/testkit"
	"github.com/k2sebeom/balpyo/render"
	"github.com/k2sebeom/balpyo/render/geom"
	"github.com/k2sebeom/balpyo/render/web"
)

// 감시 루프의 계약 셋이다.
//
//  1. **산출물은 build와 같다** — 루프는 같은 파이프라인을 돌려 같은 디렉토리에 쓴다. 저자가
//     보며 쓴 것이 배포하는 것이어야 한다.
//  2. **실패한 빌드가 산출물을 지우지 않는다** — 마지막 성공본이 디스크에 남고, 실패는 상태로만
//     말한다 (그 상태를 오버레이가 마지막 성공본 위에 얹는다).
//  3. **고치면 다시 빌드한다** — 변경이 멈춘 뒤 한 번, 세대 번호가 하나 오른다.
//
// 이 테스트는 html 타깃만 쓴다 — 브라우저 없이 루프의 배선을 보는 것이 목적이고, 지오메트리
// 패스의 배선은 pipeline 쪽 테스트(Options.Measurer)가 본다.

const watchSlide = `<slide id="hero">
  <text role="title">감시 중</text>
</slide>`

// 첫 빌드는 산출물을 쓰고 성공 상태를 띄운다 — 서버가 그 파일을 바로 서브한다.
func TestRunnerBuildsAndPublishesStatus(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "slides/001_a.slide", watchSlide)
	runner := newRunner(t, root)

	status := runner.Build()
	if !status.OK {
		t.Fatalf("첫 빌드가 실패했다: %+v", status)
	}
	if status.Generation != 1 {
		t.Errorf("세대 = %d, want 1", status.Generation)
	}
	if status.Slides != 1 {
		t.Errorf("슬라이드 %d개, want 1개", status.Slides)
	}
	if status.BuiltAt == "" {
		t.Error("빌드 시각이 비었다 — 화면이 언제 것인지 알 수 없다")
	}
	if got := readFile(t, filepath.Join(root, "dist"), web.IndexFile); !strings.Contains(got, "감시 중") {
		t.Errorf("산출물이 이 덱의 것이 아니다:\n%s", got)
	}
	// 상태는 서버에도 실린다 — 브라우저가 붙는 즉시 같은 값을 본다.
	if got := runner.Server().Status(); got.Generation != 1 || !got.OK {
		t.Errorf("서버의 상태 = %+v, want generation 1 · 성공", got)
	}
}

// 실패한 빌드는 **디스크를 건드리지 않는다**: 마지막 성공본이 남고 진단만 상태로 나간다.
//
// 산출물을 지우거나 반쯤 갱신하면 저자는 방금 고친 자리를 비교할 대상을 잃는다 — 그 비교가
// 저작 루프의 전부다.
func TestRunnerKeepsTheLastGoodOutputWhenABuildFails(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "slides/001_a.slide", watchSlide)
	runner := newRunner(t, root)
	if status := runner.Build(); !status.OK {
		t.Fatalf("첫 빌드가 실패했다: %+v", status)
	}
	good := readFile(t, filepath.Join(root, "dist"), web.IndexFile)

	writeFile(t, root, "slides/001_a.slide", "<slide>\n  <txt>제목</txt>\n</slide>")
	status := runner.Build()
	if status.OK {
		t.Fatal("닫힌 어휘 위반이 성공으로 나왔다")
	}
	if status.Generation != 2 {
		t.Errorf("세대 = %d, want 2", status.Generation)
	}
	if status.Error == "" {
		t.Error("실패에 이유가 없다")
	}
	// 이유는 **요약 한 줄**이다 — 상세는 진단 목록이 낸다. 전문이 여기 들어가면 오버레이 제목이
	// 에러 9건의 전문이 되고(그 줄에는 파일 이름도 안 보인다) 터미널 한 줄도 같은 것이 된다.
	if strings.ContainsAny(status.Error, "\n") || strings.Contains(status.Error, "<txt>") {
		t.Errorf("실패 이유가 요약이 아니다: %q", status.Error)
	}
	if len(status.Diagnostics) != 1 {
		t.Fatalf("진단 %d건, want 1건: %+v", len(status.Diagnostics), status.Diagnostics)
	}
	diag := status.Diagnostics[0]
	if diag.File != "slides/001_a.slide" || diag.Line != 2 || diag.Column != 4 {
		t.Errorf("진단의 자리 = %s:%d:%d, want slides/001_a.slide:2:4", diag.File, diag.Line, diag.Column)
	}
	if !strings.Contains(diag.Message, "<txt>") {
		t.Errorf("진단 메시지 = %q", diag.Message)
	}
	if got := readFile(t, filepath.Join(root, "dist"), web.IndexFile); got != good {
		t.Error("실패한 빌드가 마지막 성공본을 지웠다")
	}
}

// 고치면 다시 빌드한다 — 감시 루프가 스캐너와 빌드를 잇는다.
func TestRunnerRebuildsWhenTheProjectChanges(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "slides/001_a.slide", watchSlide)
	runner := newRunner(t, root)
	if status := runner.Build(); !status.OK {
		t.Fatalf("첫 빌드가 실패했다: %+v", status)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- runner.Watch(ctx) }()

	writeFile(t, root, "slides/001_a.slide", `<slide id="hero">
  <text role="title">고쳤다</text>
</slide>`)
	status := waitForGeneration(t, runner, 2)
	if !status.OK {
		t.Fatalf("다시 빌드한 결과가 실패다: %+v", status)
	}
	if got := readFile(t, filepath.Join(root, "dist"), web.IndexFile); !strings.Contains(got, "고쳤다") {
		t.Error("산출물이 갱신되지 않았다")
	}

	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Errorf("감시 루프가 에러로 끝났다: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Error("ctx가 끝났는데 감시 루프가 돌아오지 않았다")
	}
}

// 감시 루프는 **자기 산출물**로는 다시 빌드하지 않는다 — 빌드가 쓴 파일이 다음 빌드를 부르면
// 루프가 멈추지 않고, 그 상태는 CPU 하나를 계속 먹는다.
func TestRunnerDoesNotRebuildOnItsOwnOutput(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "slides/001_a.slide", watchSlide)
	runner := newRunner(t, root)
	if status := runner.Build(); !status.OK {
		t.Fatalf("첫 빌드가 실패했다: %+v", status)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go runner.Watch(ctx)

	time.Sleep(20 * DefaultTestInterval)
	if got := runner.Server().Status().Generation; got != 1 {
		t.Errorf("가만히 둔 프로젝트가 %d세대까지 빌드했다 — 루프가 스스로를 먹는다", got)
	}
}

// 재는 루프는 브라우저 **하나**를 계속 쓴다.
//
// 기동이 지오메트리 패스 비용의 대부분이므로(~0.6초) 빌드마다 다시 띄우면 `--check-layout`을 켠
// 감시는 저장마다 그 값을 내고, 그러면 저자는 검사를 끈다 — 이 테스트가 그 회귀를 잡는다.
func TestRunnerReusesOneWarmBrowser(t *testing.T) {
	testkit.RequireChromium(t)
	root := t.TempDir()
	writeFile(t, root, "slides/001_a.slide", watchSlide)
	runner, err := New(Options{
		Build: pipeline.Options{
			ProjectDir:  root,
			Targets:     []render.Target{render.TargetHTML},
			CheckLayout: true,
		},
		Interval: DefaultTestInterval,
		Log:      io.Discard,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(runner.Close)

	if status := runner.Build(); !status.OK {
		t.Fatalf("첫 빌드가 실패했다: %+v", status)
	}
	live := runner.session.live
	if live == nil {
		t.Fatal("--check-layout 빌드가 브라우저를 띄우지 않았다")
	}
	if status := runner.Build(); !status.OK {
		t.Fatalf("두 번째 빌드가 실패했다: %+v", status)
	}
	if runner.session.live != live {
		t.Error("두 번째 빌드가 브라우저를 다시 띄웠다 — 웜 세션이 아니다")
	}
}

// 없는 프로젝트는 감시를 시작하기 **전에** 멈춘다.
func TestNewRejectsAMissingProject(t *testing.T) {
	_, err := New(Options{Build: pipeline.Options{ProjectDir: filepath.Join(t.TempDir(), "없다")}})
	if err == nil {
		t.Fatal("없는 프로젝트가 감시를 시작했다")
	}
}

// html이 아닌 타깃만 주면 감시할 것이 없다 — 볼 페이지가 없으면 이 명령은 뜻이 없다.
func TestNewRequiresTheHTMLTarget(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "slides/001_a.slide", watchSlide)
	_, err := New(Options{Build: pipeline.Options{
		ProjectDir: root,
		Targets:    []render.Target{render.TargetPPTX},
	}})
	if err == nil {
		t.Fatal("html 없이 감시가 시작되었다")
	}
	if !strings.Contains(err.Error(), "html") {
		t.Errorf("에러가 이유를 말하지 않는다: %v", err)
	}
}

// DefaultTestInterval은 테스트의 폴링 간격이다 — 실제 간격(250ms)으로 돌면 테스트가 초 단위로
// 길어진다. 디바운스의 규칙은 스캐너 테스트가 보므로 여기서는 짧게 돌린다.
const DefaultTestInterval = 10 * time.Millisecond

// newRunner는 브라우저를 쓰지 않는 감시기다 (html 타깃만, 출력은 프로젝트 안의 dist).
func newRunner(t *testing.T, root string) *Runner {
	t.Helper()
	// 브라우저를 띄우려 하면 실패하게 둔다 — html 전용 루프가 기동 비용을 내지 않는다는 사실을
	// 이 한 줄이 지킨다 (pipeline의 noBrowser와 같은 장치다).
	t.Setenv(geom.EnvChromium, filepath.Join(t.TempDir(), "없는-크로미움"))
	runner, err := New(Options{
		Build:    pipeline.Options{ProjectDir: root, Targets: []render.Target{render.TargetHTML}},
		Interval: DefaultTestInterval,
		Log:      io.Discard,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(runner.Close)
	return runner
}

// waitForGeneration은 서버의 상태가 그 세대에 이를 때까지 기다린다.
func waitForGeneration(t *testing.T, runner *Runner, generation int) Status {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		if status := runner.Server().Status(); status.Generation >= generation {
			return status
		}
		time.Sleep(DefaultTestInterval)
	}
	t.Fatalf("%d세대 빌드가 오지 않았다 (지금 %+v)", generation, runner.Server().Status())
	return Status{}
}
