package testkit

import (
	"bytes"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/k2sebeom/balpyo/render/geom"
)

// Rect는 요소의 문서 좌표 사각형이다 (논리 스테이지 px).
//
// 별칭인 이유: 좌표 타입이 둘이면 하네스가 본 값과 빌드가 낸 값을 비교할 때 변환이 끼어들고,
// 그 변환이 곧 두 좌표계가 갈라지는 자리가 된다.
type Rect = geom.Rect

// Browser는 재사용하는 헤드리스 브라우저 세션이다 — 지오메트리 패스의 세션(geom.Session)을
// testing.TB에 맞춰 감싼 것이다.
//
// **CDP 클라이언트를 여기에 다시 만들지 않는다.** 기동 플래그와 좌표 스니펫이 두 벌이 되면
// 테스트가 재는 좌표와 빌드가 재는 좌표가 서로 다른 조건에서 나오고, 그 차이는 골든이
// "픽셀이 다르다"로만 알려 준다 — 원인이 렌더인지 하네스인지 모르는 상태다. 이 파일이
// 소유하는 것은 **테스트 정책**뿐이다: 도구가 없을 때의 건너뛰기/실패(RequireChromium)와
// t.Cleanup 정리, 그리고 실패를 t.Fatal로 바꾸는 얇은 층.
//
// 픽스처마다 브라우저를 새로 띄우지 않는 이유는 스위트 시간이다 — 세션 하나를 공유하고
// 페이지만 바꿔 연다 (SharedSession).
type Browser struct {
	session *geom.Session
}

// NewBrowser는 공유 세션(SharedSession) 위의 브라우저다. 페이지는 Load가 연다.
func NewBrowser(t testing.TB) *Browser {
	t.Helper()
	return &Browser{session: SharedSession(t)}
}

var shared struct {
	once    sync.Once
	session *geom.Session
	err     error
}

// SharedSession은 테스트 바이너리 하나가 공유하는 브라우저 세션이다 — 픽스처 빌드 · 예제 빌드 ·
// 측정 테스트가 모두 이 세션을 쓴다.
//
// 공유하는 이유는 스위트 시간이다: 기동이 빌드 한 번보다 비싸서, 빌드마다 브라우저를 띄우면
// `go test ./...`의 대부분이 기동 시간이었다. 세션은 페이지를 열 때마다 새 문서를 그리므로
// (`balpyo watch`의 웜 세션과 같은 근거 — ADR 0046) 앞 테스트의 상태가 다음 측정에 남지 않는다.
// 테스트를 병렬로 돌리지 않는다는 전제다 — 세션 하나는 한 번에 한 페이지를 연다.
//
// 정리는 그 패키지의 TestMain이 CloseSharedSession으로 한다.
func SharedSession(t testing.TB) *geom.Session {
	t.Helper()
	bin := RequireChromium(t)
	shared.once.Do(func() { shared.session, shared.err = geom.Launch(bin) })
	if shared.err != nil {
		t.Fatalf("브라우저를 띄울 수 없다: %v", shared.err)
	}
	return shared.session
}

// CloseSharedSession은 공유 세션을 닫는다 (띄운 적이 없으면 아무것도 하지 않는다).
func CloseSharedSession() {
	if shared.session != nil {
		shared.session.Close()
	}
}

// Load는 로컬 HTML 파일을 연다. 폰트가 다 실린 뒤에 돌아온다 — 폰트 로드 전에 찍은
// 스크린샷은 대체 폰트로 그려져 픽셀 비교를 통째로 무너뜨린다.
func (b *Browser) Load(t testing.TB, htmlPath string) {
	t.Helper()
	if err := b.session.Open(htmlPath); err != nil {
		t.Fatal(err)
	}
}

// ScreenshotElement는 선택자에 맞는 첫 요소를 찍는다. 요소 크기 그대로 나오므로
// 논리 스테이지 1280×720 슬라이드는 1280×720 PNG가 된다.
//
// 잘라내기는 geom.Session.Clip이 한다 — 문서 좌표 rect를 읽어 CaptureBeyondViewport로 그
// 사각형만 잘라낸다. 요소를 뷰포트로 스크롤한 뒤 잘라내는 방식(chromedp의 요소 스크린샷
// 헬퍼)을 쓰지 않는 이유는 그 함수의 주석에 있다 (Iteration 0에서 덱 여백이 그대로 diff에
// 잡혔다). 좌표를 지오메트리 패스와 **같은 스니펫**에서 얻으므로 스크린샷과 PPTX 좌표가
// 갈라지지 않는다.
func (b *Browser) ScreenshotElement(t testing.TB, selector string) image.Image {
	t.Helper()
	rects := b.Rects(t, selector)
	if len(rects) == 0 {
		t.Fatalf("%s에 맞는 요소가 없다", selector)
	}
	buf, err := b.session.Clip(rects[0])
	if err != nil {
		t.Fatalf("%s: %v", selector, err)
	}
	img, err := png.Decode(bytes.NewReader(buf))
	if err != nil {
		t.Fatalf("%s: %v", selector, err)
	}
	return img
}

// PrintPDF는 지금 열려 있는 문서를 인쇄해 PDF 파일 경로를 돌려준다 (사람이 Cmd+P를 누르는
// 것과 같은 경로, geom.Session.Print).
//
// 바이트가 아니라 경로인 이유: 판정은 렌더한 페이지로 하고(RenderPDFPoints), 렌더 도구는
// 파일을 받는다. 호출마다 새 임시 디렉토리에 쓰므로 한 테스트가 두 번 인쇄해도 앞의 결과를
// 다시 읽지 않는다.
func (b *Browser) PrintPDF(t testing.TB) string {
	t.Helper()
	buf, err := b.session.Print()
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "print.pdf")
	if err := os.WriteFile(path, buf, 0o644); err != nil {
		t.Fatalf("인쇄 결과를 쓸 수 없다: %v", err)
	}
	return path
}

// Rects는 선택자에 맞는 모든 요소의 문서 좌표 사각형을 문서 순서로 읽는다.
func (b *Browser) Rects(t testing.TB, selector string) []Rect {
	t.Helper()
	rects, err := b.session.Rects(selector)
	if err != nil {
		t.Fatal(err)
	}
	return rects
}

// Eval은 임의 표현식을 평가해 결과를 out에 담는다 (out이 nil이면 값을 버린다).
func (b *Browser) Eval(t testing.TB, expr string, out any) {
	t.Helper()
	if err := b.session.Eval(expr, out); err != nil {
		t.Fatalf("%v\n%s", err, expr)
	}
}
