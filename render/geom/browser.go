package geom

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/chromedp"

	"github.com/k2sebeom/balpyo/render/web"
)

// 브라우저를 찾는 환경 변수. CI가 값을 넣고(.github/workflows/ci.yml), 사용자도
// 같은 변수로 브라우저를 지정한다 — 빌드와 테스트가 다른 변수를 보면 "테스트는 되는데 빌드는
// 브라우저를 못 찾는다"가 된다.
const (
	// EnvChromium은 Chromium/Chrome 실행 파일 경로다.
	EnvChromium = "BALPYO_CHROMIUM"
	// EnvNoSandbox가 1이면 --no-sandbox를 붙인다 (root · 샌드박스를 막는 러너에서 필요).
	EnvNoSandbox = "BALPYO_CHROMIUM_NO_SANDBOX"
)

const (
	// actionTimeout은 브라우저 동작 하나에 주는 시간이다. 헤드리스 브라우저는 실패할 때
	// 응답하지 않고 멈추는 쪽이라 상한이 없으면 빌드와 CI가 매달린다.
	actionTimeout = 60 * time.Second
	// sessionLifetime은 세션 하나의 상한이다.
	sessionLifetime = 10 * time.Minute
	// launchAttempts는 기동을 시도하는 횟수다. 브라우저 여럿이 한꺼번에 뜨면(`go test ./...`가
	// 패키지를 나란히 돌릴 때, watch와 build가 겹칠 때) 기동이 가끔 `context canceled` ·
	// `dial … EOF`로 실패하고, 같은 명령을 다시 하면 뜬다 — 그 다시 하기를 사람 대신 한다.
	launchAttempts = 3
	// launchBackoff는 시도 사이의 기다림이다 (n번째 실패 뒤 n배).
	launchBackoff = 250 * time.Millisecond
)

// Candidates는 브라우저 실행 파일 후보다 (탐색 순서).
//
// 브라우저를 내려받지 않고 시스템에 있는 것을 쓴다 — macOS 경로를 함께 두는 이유는
// 로컬에서도 컨테이너 없이 빌드가 돌아야 하기 때문이다.
func Candidates() []string {
	return []string{
		"chromium",
		"chromium-browser",
		"google-chrome",
		"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
		"/Applications/Chromium.app/Contents/MacOS/Chromium",
	}
}

// LookChromium은 브라우저 실행 파일을 찾는다: 환경 변수 → PATH → 알려진 설치 경로.
func LookChromium() (string, error) {
	if custom := os.Getenv(EnvChromium); custom != "" {
		if err := executable(custom); err != nil {
			return "", fmt.Errorf("%s=%s: %w", EnvChromium, custom, err)
		}
		return custom, nil
	}
	candidates := Candidates()
	for _, candidate := range candidates {
		if strings.ContainsRune(candidate, filepath.Separator) {
			if err := executable(candidate); err == nil {
				return candidate, nil
			}
			continue
		}
		if path, err := exec.LookPath(candidate); err == nil {
			return path, nil
		}
	}
	return "", fmt.Errorf("no browser found (tried: %s) — set its path with %s",
		strings.Join(candidates, ", "), EnvChromium)
}

// NoSandbox는 --no-sandbox가 필요한 환경인지 알려준다.
func NoSandbox() bool { return os.Getenv(EnvNoSandbox) == "1" }

// executable은 경로가 실행 가능한 파일인지 본다.
func executable(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return fmt.Errorf("is a directory")
	}
	if info.Mode()&0o111 == 0 {
		return fmt.Errorf("not executable")
	}
	return nil
}

// Session은 재사용하는 헤드리스 브라우저 세션이다.
//
// 페이지마다 브라우저를 새로 띄우면 빌드 시간이 기동 시간의 합이 된다. 세션 하나를 공유하고
// 페이지만 바꿔 여는 이유이며, `balpyo server`의 웜 브라우저 세션과 테스트 하네스
// (internal/testkit)가 같은 세션 타입을 쓴다 — CDP 클라이언트가 둘이면 빌드가 재는 좌표와
// 테스트가 재는 좌표가 다른 플래그 아래에서 나온다.
type Session struct {
	ctx     context.Context
	cancels []context.CancelFunc
}

// Launch는 브라우저를 띄운다. bin이 비면 LookChromium으로 찾는다. 정리는 호출자가 Close로 한다.
//
// 플래그는 전부 렌더 결정성을 위한 것이다: 스케일 1, sRGB 고정, 힌팅·LCD 서브픽셀 끔 —
// 같은 호스트에서 같은 입력이 같은 rect를 내야 산출 바이트가 같다 (불변식 6).
func Launch(bin string) (*Session, error) {
	if bin == "" {
		found, err := LookChromium()
		if err != nil {
			return nil, err
		}
		bin = found
	}
	var err error
	for attempt := 1; attempt <= launchAttempts; attempt++ {
		var s *Session
		if s, err = launchOnce(bin); err == nil {
			return s, nil
		}
		if attempt < launchAttempts {
			time.Sleep(time.Duration(attempt) * launchBackoff)
		}
	}
	return nil, fmt.Errorf("cannot launch the browser (%s, %d attempts): %w", bin, launchAttempts, err)
}

// launchOnce는 브라우저를 한 번 띄운다.
func launchOnce(bin string) (*Session, error) {
	opts := append([]chromedp.ExecAllocatorOption{}, chromedp.DefaultExecAllocatorOptions[:]...)
	opts = append(opts,
		chromedp.ExecPath(bin),
		// 뷰포트를 못 박는다. 지오메트리 패스는 슬라이드 상대 좌표를 쓰므로 스크롤·뷰포트가
		// 결과를 바꾸지 않지만, 레이아웃(뷰포트 단위·중앙 정렬 오프셋)이 브라우저 기본값에
		// 따라 달라지면 좌표가 환경마다 흔들린다. 16:9 스테이지가 여백까지 들어가는 크기로 고정한다.
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
		// 애니메이션·전환이 측정 시점에 따라 다른 좌표를 내지 않게 한다.
		chromedp.Flag("force-prefers-reduced-motion", true),
	)
	if NoSandbox() {
		opts = append(opts, chromedp.NoSandbox)
	}

	lifeCtx, cancelLife := context.WithTimeout(context.Background(), sessionLifetime)
	allocCtx, cancelAlloc := chromedp.NewExecAllocator(lifeCtx, opts...)
	ctx, cancelCtx := chromedp.NewContext(allocCtx)
	s := &Session{ctx: ctx, cancels: []context.CancelFunc{cancelCtx, cancelAlloc, cancelLife}}

	// 기동은 **파생 컨텍스트가 아니라 s.ctx로** 돌려야 한다.
	//
	// chromedp는 브라우저 수명을 첫 Run에 넘긴 컨텍스트에 묶는다. 여기에 타임아웃 파생
	// 컨텍스트를 쓰면 그 컨텍스트가 취소되는 순간(=이 함수가 돌아오는 순간) 브라우저가 죽고,
	// 다음 동작이 "context canceled"로 실패한다. 기동한 뒤의 동작은 파생 컨텍스트로 돌려도
	// 안전하다 (Session.run).
	if err := chromedp.Run(s.ctx, chromedp.Navigate("about:blank")); err != nil {
		s.Close()
		return nil, err
	}
	return s, nil
}

// Close는 브라우저 컨텍스트를 역순으로 닫는다.
func (s *Session) Close() {
	for i := len(s.cancels) - 1; i >= 0; i-- {
		s.cancels[i]()
	}
}

// run은 액션 묶음을 시간 제한 안에서 돌린다.
func (s *Session) run(actions ...chromedp.Action) error {
	ctx, cancel := context.WithTimeout(s.ctx, actionTimeout)
	defer cancel()
	return chromedp.Run(ctx, actions...)
}

// Open은 로컬 HTML 파일을 연다. **폰트와 이미지가 다 실린 뒤에** 돌아온다 — 로드 전에 잰
// 좌표는 대체 폰트의 글자 폭으로 계산된 것이어서 줄바꿈부터 다르고, 이미지는 고유 크기를
// 모르는 상태의 박스다 (실측: 디코드 전 `naturalWidth`가 0이고 `<img>`의 rect가 0×0이다 —
// 그 좌표가 그대로 PPTX `pic`의 `xfrm`이 된다).
//
// 열자마자 **문서 뷰로 되돌린다** (web.DocumentViewExpr). 웹 산출물의 런타임은 로드 즉시
// 발표자 뷰를 켜는데, 발표자 뷰는 슬라이드를 겹치고 화면에 맞춰 스케일하므로 그 배치에서
// 잰 rect는 논리 스테이지와 어긋난다 (newGeometry의 크기 검증이 잡는 바로 그 상태).
// 발표자 뷰의 CSS는 전부 <html>의 속성 하나에 걸려 있어 지우면 측정 배치로 돌아오고,
// balpyo 산출물이 아닌 페이지에서는 이 평가가 아무 일도 하지 않는다. 여기(모든 열기의
// 관문)에서 되돌려야 지오메트리 패스와 스크린샷 하네스가 같은 배치를 본다 — 이미지 대기를
// 같은 자리에 둔 근거도 같다: 스크린샷 하네스(internal/testkit)가 같은 대기를 공짜로 받는다.
func (s *Session) Open(htmlPath string) error {
	abs, err := filepath.Abs(htmlPath)
	if err != nil {
		return fmt.Errorf("cannot make the path absolute: %w", err)
	}
	fileURL := (&url.URL{Scheme: "file", Path: filepath.ToSlash(abs)}).String()
	if err := s.run(
		chromedp.Navigate(fileURL),
		chromedp.WaitReady("body", chromedp.ByQuery),
		chromedp.Evaluate(web.DocumentViewExpr(), nil),
		chromedp.Evaluate(`document.fonts.ready.then(() => true)`, nil, withAwaitPromise),
		chromedp.Evaluate(imagesReadyScript, nil, withAwaitPromise),
	); err != nil {
		return fmt.Errorf("cannot open %s: %w", htmlPath, err)
	}
	return nil
}

// imagesReadyScript는 문서의 모든 이미지가 디코드될 때까지 기다리고, **읽히지 않은 이미지가
// 하나라도 있으면 던진다**.
//
// 실패를 통과시키면 그림 없는 PPTX가 **정상으로 열린다**: 에셋 패스의 존재·포맷 검사를 지나온
// 파일이 브라우저에서 읽히지 않는 경우(산출 파일이 빠졌다, 경로가 어긋났다, 브라우저가 그
// 변형을 모른다)가 여기서 잡히는 마지막 자리다. 판정을 `naturalWidth === 0`으로 두는 이유는
// `decode()`의 거부가 유일한 실패 신호가 아니기 때문이다 — 404는 `complete`가 true인 채
// 고유 크기가 0으로 남는다 (실측).
//
// `decode()`를 쓰고 `complete`를 함께 보는 이유: `complete`는 로드가 끝났다는 뜻이고 디코드가
// 끝났다는 뜻이 아니다. 거부를 삼키는 것(`catch`)은 실패를 무시하는 것이 아니라 **판정을 한
// 곳으로 모으는 것**이다 — 거부한 이미지는 아래 고유 크기 검사에서 이름과 함께 걸린다.
const imagesReadyScript = `(async () => {
  const images = Array.from(document.images);
  await Promise.all(images.map((img) => img.decode().catch(() => null)));
  const broken = images.filter((img) => img.naturalWidth === 0).map((img) => img.getAttribute("src"));
  if (broken.length > 0) {
    throw new Error("cannot load images: " + broken.join(", "));
  }
  return true;
})()`

// withAwaitPromise는 프로미스가 풀릴 때까지 기다리는 평가 옵션이다.
func withAwaitPromise(p *runtime.EvaluateParams) *runtime.EvaluateParams {
	return p.WithAwaitPromise(true)
}

// Eval은 표현식을 평가해 결과를 out에 담는다 (out이 nil이면 값을 버린다).
func (s *Session) Eval(expr string, out any) error {
	if err := s.run(chromedp.Evaluate(expr, out)); err != nil {
		return fmt.Errorf("script evaluation failed: %w", err)
	}
	return nil
}

// Print는 지금 열려 있는 문서를 PDF로 인쇄한다 (사람이 Cmd+P를 누르는 것과 같은 경로).
//
// 종이 크기는 **문서가 정한다**: PreferCSSPageSize가 CSS의 `@page { size }`를 따르게 하므로
// 한 장이 한 페이지를 꽉 채운다 (web.printPageCSS). Go에서 크기를 넘기면 인쇄 CSS가
// 틀렸을 때도 종이는 맞게 나와 게이트가 아무것도 판정하지 못한다.
//
// 배경은 함께 찍는다 — 슬라이드의 바탕색·배경 이미지는 장식이 아니라 내용이고, 브라우저
// 기본값(배경 없음)으로 인쇄하면 덱이 흰 종이에 검은 글자로만 나온다.
func (s *Session) Print() ([]byte, error) {
	var buf []byte
	if err := s.run(chromedp.ActionFunc(func(ctx context.Context) error {
		var err error
		buf, _, err = page.PrintToPDF().
			WithPreferCSSPageSize(true).
			WithPrintBackground(true).Do(ctx)
		return err
	})); err != nil {
		return nil, fmt.Errorf("print failed: %w", err)
	}
	return buf, nil
}

// rectScript는 요소 사각형을 **문서 좌표**로 읽는 스니펫이다.
//
// **문자열 하나로 두는 이유**: 요소 스크린샷의 잘라낼 영역(Session.Clip)과 화살표 앵커
// 계산이 같은 코드를 평가해야 한다. 좌표 로직이 Go와 JS에 이중 구현되는 순간 웹과 PPTX의
// 배치가 갈린다.
const rectScript = `(() => {
  const els = Array.from(document.querySelectorAll(%q));
  return els.map(el => {
    const r = el.getBoundingClientRect();
    return {x: r.x + window.scrollX, y: r.y + window.scrollY, w: r.width, h: r.height};
  });
})()`

// Rects는 선택자에 맞는 모든 요소의 **문서 좌표** 사각형을 문서 순서로 읽는다.
//
// 지오메트리 패스가 쓰는 좌표는 이것이 아니라 슬라이드 상대다 (Measure) — 문서 좌표는
// 스크린샷을 잘라낼 때만 필요하다.
func (s *Session) Rects(selector string) ([]Rect, error) {
	var rects []Rect
	if err := s.Eval(fmt.Sprintf(rectScript, selector), &rects); err != nil {
		return nil, fmt.Errorf("cannot read the coordinates of %s: %w", selector, err)
	}
	return rects, nil
}

// Clip은 **문서 좌표** 사각형을 PNG로 찍는다.
//
// 뷰포트로 스크롤한 뒤 잘라내는 방식(chromedp의 요소 스크린샷 헬퍼)을 쓰지 않는다: 슬라이드는
// 뷰포트보다 크므로 잘라낸 영역이 요소 박스와 스크롤량만큼 어긋나고 뷰포트 밖은 페이지 배경으로
// 채워진다. 그러면 픽셀 비교가 렌더 차이가 아니라 스크린샷 버그를 보고한다 (빈
// 슬라이드조차 22% 다르다고 나온다).
func (s *Session) Clip(r Rect) ([]byte, error) {
	if r.W <= 0 || r.H <= 0 {
		return nil, fmt.Errorf("cannot capture a zero-size area: %s", r)
	}
	var buf []byte
	if err := s.run(chromedp.ActionFunc(func(ctx context.Context) error {
		var err error
		buf, err = page.CaptureScreenshot().
			WithFormat(page.CaptureScreenshotFormatPng).
			WithCaptureBeyondViewport(true).
			WithClip(&page.Viewport{X: r.X, Y: r.Y, Width: r.W, Height: r.H, Scale: 1}).Do(ctx)
		return err
	})); err != nil {
		return nil, fmt.Errorf("screenshot of %s failed: %w", r, err)
	}
	return buf, nil
}
