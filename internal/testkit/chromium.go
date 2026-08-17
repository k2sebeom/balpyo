package testkit

import (
	"context"
	"fmt"
	"image"
	"net/url"
	"path/filepath"
	"testing"
	"time"

	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/chromedp"
)

const (
	// browserTimeout은 브라우저 동작 하나에 주는 시간이다. 헤드리스 브라우저는 실패할 때
	// 응답하지 않고 멈추는 쪽이라 상한이 없으면 CI가 매달린다.
	browserTimeout = 60 * time.Second
	// browserLifetime은 세션 하나의 상한이다. 스위트 전체가 이 안에 끝난다 — 브라우저가
	// 기동에서 멈춰도 테스트 바이너리 타임아웃(원인을 알려주지 않는다)까지 가지 않는다.
	browserLifetime = 10 * time.Minute
)

// Browser는 재사용하는 헤드리스 브라우저 세션이다.
//
// 픽스처마다 브라우저를 새로 띄우면 스위트 시간이 기동 시간의 합이 된다. 세션 하나를
// 공유하고 페이지만 바꿔 여는 이유이며, `balpyo server`의 "웜 브라우저 세션"과 같은
// 전략이다 (PRD §11).
type Browser struct {
	ctx     context.Context
	cancels []context.CancelFunc
}

// NewBrowser는 결정적인 플래그로 브라우저를 띄운다. 정리는 t.Cleanup이 한다.
func NewBrowser(t testing.TB) *Browser {
	t.Helper()
	bin := RequireChromium(t)
	browser, err := newBrowser(bin)
	if err != nil {
		t.Fatalf("브라우저를 띄울 수 없다 (%s): %v", bin, err)
	}
	t.Cleanup(browser.close)
	return browser
}

// newBrowser는 브라우저를 띄운다. 정리 책임을 호출자에게 남기므로 스위트가 세션을
// 픽스처보다 오래 살릴 수 있다 (Suite.Browser).
//
// 플래그는 전부 렌더 결정성을 위한 것이다: 스케일 1, sRGB 고정, 힌팅·LCD 서브픽셀 끔.
// 이미지 안의 fontconfig 설정(docker/fonts.conf)과 같은 방향이다 — 브라우저와
// LibreOffice가 같은 글자 폭을 내야 픽셀 비교가 성립한다.
func newBrowser(bin string) (*Browser, error) {
	opts := append([]chromedp.ExecAllocatorOption{}, chromedp.DefaultExecAllocatorOptions[:]...)
	opts = append(opts,
		chromedp.ExecPath(bin),
		// 뷰포트를 못 박는다. 스크린샷은 요소 박스로 잘라내므로 뷰포트가 결과를 좌우하지
		// 않지만, 레이아웃(뷰포트 단위·중앙 정렬 오프셋)이 브라우저 기본값에 따라 달라지면
		// 지오메트리 패스가 읽는 좌표가 환경마다 흔들린다. 16:9 스테이지가 여백까지 들어가는
		// 크기로 고정한다.
		chromedp.WindowSize(1440, 900),
		chromedp.Flag("hide-scrollbars", true),
		chromedp.Flag("force-device-scale-factor", "1"),
		chromedp.Flag("force-color-profile", "srgb"),
		chromedp.Flag("font-render-hinting", "none"),
		chromedp.Flag("disable-lcd-text", true),
		chromedp.Flag("disable-font-subpixel-positioning", true),
		chromedp.Flag("disable-dev-shm-usage", true),
		chromedp.Flag("disable-extensions", true),
		chromedp.Flag("disable-background-timer-throttling", true),
		// 애니메이션·전환이 스크린샷 시점에 따라 다른 픽셀을 내지 않게 한다.
		chromedp.Flag("force-prefers-reduced-motion", true),
	)
	if ChromiumNoSandbox() {
		opts = append(opts, chromedp.NoSandbox)
	}

	lifeCtx, cancelLife := context.WithTimeout(context.Background(), browserLifetime)
	allocCtx, cancelAlloc := chromedp.NewExecAllocator(lifeCtx, opts...)
	ctx, cancelCtx := chromedp.NewContext(allocCtx)
	b := &Browser{ctx: ctx, cancels: []context.CancelFunc{cancelCtx, cancelAlloc, cancelLife}}

	// 기동은 **파생 컨텍스트가 아니라 b.ctx로** 돌려야 한다.
	//
	// chromedp는 브라우저 수명을 첫 Run에 넘긴 컨텍스트에 묶는다. 여기에 타임아웃 파생
	// 컨텍스트를 쓰면 그 컨텍스트가 취소되는 순간(=이 함수가 돌아오는 순간) 브라우저가
	// 죽고, 다음 동작이 "context canceled"로 실패한다. 기동한 뒤의 동작은 파생 컨텍스트로
	// 돌려도 안전하다 (b.run).
	if err := chromedp.Run(b.ctx, chromedp.Navigate("about:blank")); err != nil {
		b.close()
		return nil, err
	}
	return b, nil
}

// close는 브라우저 컨텍스트를 역순으로 닫는다.
func (b *Browser) close() {
	for i := len(b.cancels) - 1; i >= 0; i-- {
		b.cancels[i]()
	}
}

// run은 액션 묶음을 시간 제한 안에서 돌린다.
func (b *Browser) run(actions ...chromedp.Action) error {
	ctx, cancel := context.WithTimeout(b.ctx, browserTimeout)
	defer cancel()
	return chromedp.Run(ctx, actions...)
}

// Load는 로컬 HTML 파일을 연다. 폰트가 다 실린 뒤에 돌아온다 — 폰트 로드 전에 찍은
// 스크린샷은 대체 폰트로 그려져 픽셀 비교를 통째로 무너뜨린다.
func (b *Browser) Load(t testing.TB, htmlPath string) {
	t.Helper()
	abs, err := filepath.Abs(htmlPath)
	if err != nil {
		t.Fatalf("경로를 절대 경로로 바꿀 수 없다: %v", err)
	}
	fileURL := (&url.URL{Scheme: "file", Path: filepath.ToSlash(abs)}).String()
	if err := b.run(
		chromedp.Navigate(fileURL),
		chromedp.WaitReady("body", chromedp.ByQuery),
		chromedp.Evaluate(`document.fonts.ready.then(() => true)`, nil, withAwaitPromise),
	); err != nil {
		t.Fatalf("%s를 열 수 없다: %v", htmlPath, err)
	}
}

// withAwaitPromise는 프로미스가 풀릴 때까지 기다리는 평가 옵션이다.
func withAwaitPromise(p *runtime.EvaluateParams) *runtime.EvaluateParams {
	return p.WithAwaitPromise(true)
}

// ScreenshotElement는 선택자에 맞는 첫 요소를 찍는다. 요소 크기 그대로 나오므로
// 논리 스테이지 1280×720 슬라이드는 1280×720 PNG가 된다.
//
// **chromedp.Screenshot(요소 스크린샷 헬퍼)을 쓰지 않는다.** 그 헬퍼는 요소를 뷰포트로
// 스크롤한 뒤 잘라내는데, 슬라이드는 뷰포트보다 크므로 잘라낸 영역이 요소 박스와
// 스크롤량만큼 어긋나고 뷰포트 밖은 페이지 배경으로 채워진다. 그러면 픽셀 비교가
// 렌더 차이가 아니라 스크린샷 버그를 보고한다 (덱 여백이 그대로 diff에 잡혔다).
//
// 대신 문서 좌표 rect를 읽어 CaptureBeyondViewport로 그 사각형만 잘라낸다 — 스크롤
// 위치와 뷰포트 크기에 무관하게 요소 박스가 나온다. 지오메트리 패스가 쓰는 rect와
// **같은 스니펫**(rectScript)에서 좌표를 얻으므로 스크린샷과 PPTX 좌표가 갈라지지 않는다.
func (b *Browser) ScreenshotElement(t testing.TB, selector string) image.Image {
	t.Helper()
	rects := b.Rects(t, selector)
	if len(rects) == 0 {
		t.Fatalf("%s에 맞는 요소가 없다", selector)
	}
	rect := rects[0]
	if rect.W <= 0 || rect.H <= 0 {
		t.Fatalf("%s의 크기가 0이다: %s", selector, rect)
	}

	var buf []byte
	if err := b.run(chromedp.ActionFunc(func(ctx context.Context) error {
		var err error
		buf, err = page.CaptureScreenshot().
			WithFormat(page.CaptureScreenshotFormatPng).
			WithCaptureBeyondViewport(true).
			WithClip(&page.Viewport{
				X: rect.X, Y: rect.Y, Width: rect.W, Height: rect.H, Scale: 1,
			}).Do(ctx)
		return err
	})); err != nil {
		t.Fatalf("%s 스크린샷 실패: %v", selector, err)
	}
	img, err := DecodePNG(buf)
	if err != nil {
		t.Fatalf("%s: %v", selector, err)
	}
	return img
}

// Rect는 요소의 문서 좌표 사각형이다 (논리 스테이지 px).
type Rect struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	W float64 `json:"w"`
	H float64 `json:"h"`
}

// String은 진단·스냅샷에 쓰는 표기다.
func (r Rect) String() string {
	return fmt.Sprintf("(%.1f,%.1f %.1f×%.1f)", r.X, r.Y, r.W, r.H)
}

// rectScript는 요소 사각형을 읽는 스니펫이다.
//
// **문자열 하나로 두는 이유**: Iteration 1의 지오메트리 패스와 Iteration 7의 앵커 계산이
// 같은 코드를 평가해야 한다. 좌표 로직이 Go와 JS에 이중 구현되는 순간 웹과 PPTX의 배치가
// 갈린다 (PRD §9.3).
const rectScript = `(() => {
  const els = Array.from(document.querySelectorAll(%q));
  return els.map(el => {
    const r = el.getBoundingClientRect();
    return {x: r.x + window.scrollX, y: r.y + window.scrollY, w: r.width, h: r.height};
  });
})()`

// Rects는 선택자에 맞는 모든 요소의 사각형을 문서 순서로 읽는다.
func (b *Browser) Rects(t testing.TB, selector string) []Rect {
	t.Helper()
	var rects []Rect
	expr := fmt.Sprintf(rectScript, selector)
	if err := b.run(chromedp.Evaluate(expr, &rects)); err != nil {
		t.Fatalf("%s 좌표를 읽을 수 없다: %v", selector, err)
	}
	return rects
}

// Eval은 임의 표현식을 평가해 결과를 out에 담는다 (out이 nil이면 값을 버린다).
func (b *Browser) Eval(t testing.TB, expr string, out any) {
	t.Helper()
	if err := b.run(chromedp.Evaluate(expr, out)); err != nil {
		t.Fatalf("스크립트 평가 실패: %v\n%s", err, expr)
	}
}
