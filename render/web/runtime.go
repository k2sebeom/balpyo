package web

import (
	_ "embed"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/k2sebeom/balpyo/soml"
)

// 이 파일은 뷰어 런타임이다 — 산출물 index.html을 발표 가능한 웹앱으로 만드는 인라인
// JS/CSS. 슬라이드를 한 장씩 화면에 맞춰 보여주고, 하단 가운데 컨트롤 바로 넘기며,
// 발표자 노트를 패널로 열어 본다.
//
// # 문서는 두 가지 뷰를 갖는다
//
// **문서 뷰**(기본, JS 없음)는 모든 슬라이드를 논리 스테이지 크기 그대로 세로로 쌓는다.
// 지오메트리 패스와 스크린샷 하네스가 재는 배치가 이것이다 — 슬라이드 rect가 논리
// 스테이지와 1:1이어야 좌표가 그대로 PPTX로 간다 (geom.newGeometry가 확인한다).
//
// **발표자 뷰**는 런타임 JS가 로드 즉시 <html>에 ViewAttr(=PresenterView)를 켜서 들어간다.
// 발표자 뷰의 CSS는 전부 이 속성에 걸려 있으므로(viewerCSS), 속성 하나가 뷰 전환의
// 전부다: 지우면 문서 뷰로 돌아온다. 지오메트리 패스가 페이지를 열 때 정확히 그렇게 한다
// (geom.Session.Open이 DocumentViewExpr를 평가한다) — 측정과 발표가 **같은 파일**을 쓰면서도
// 측정은 스케일·겹침이 없는 배치를 본다. `balpyo server`의 발표자 뷰도 같은 속성 계약을 쓴다.
//
// 발표자 뷰가 숨기는 슬라이드는 display:none이 아니라 visibility:hidden이다 — 레이아웃이
// 남아 있어야 발표자 뷰가 켜진 채로도 rect가 0으로 무너지지 않고, 뷰 전환이 줄바꿈을
// 다시 계산하게 만들지 않는다. 화면 맞춤은 transform: scale이므로 레이아웃 크기
// (offsetWidth)는 논리 스테이지 그대로 남고, 런타임이 그 값으로 배율을 계산한다.
//
// # 런타임 자산은 파일이고, 전달은 그대로 인라인이다
//
// JS와 CSS는 assets/ 아래의 파일이고 go:embed로 실린다. **전달 방식은 바뀌지
// 않는다**: 둘 다 index.html 안에 그대로 들어가고 외부 참조는 없다 (파일로 열어도 도는
// 계약). 파일로 뺀 근거는 크기가 아니라 도구다 — Go의 raw string 안에서는 백틱을
// 한 글자도 쓸 수 없어 JS 템플릿 리터럴이 막히고, 에디터·문법 검사는 그 문자열 안을
// 들여다보지 않는다. 번들러는 들이지 않는다 (환경은 하나다): 자산은 손으로 쓴 vanilla
// JS/CSS 그대로다. 인라인을 끝내는 문자열(`</script>`·`</style>`)이 자산에 들어가면 문서가
// 그 자리에서 깨지므로, 그 조건은 runtime_test가 게이트로 판정한다.

// ViewAttr는 <html>에 실리는 뷰 모드 속성이다. 없으면 문서 뷰다.
const ViewAttr = "data-balpyo-view"

// PresenterView는 발표자 뷰의 ViewAttr 값이다.
const PresenterView = "presenter"

// MeasureView는 **측정 뷰**의 ViewAttr 값이다 — 지오메트리 패스가 rect를 읽는 동안에만 켠다
// (geom.measureScript가 켜고 끈다).
//
// 문서 뷰(속성 없음)와 다른 뷰를 하나 더 둔 이유는 **회전**이다: `transform: rotate()`가
// 걸린 요소의 `getBoundingClientRect`는 회전된 외곽 bbox를 돌려주는데(45도에서 1.41배),
// OOXML `xfrm@rot`은 회전 **전** `ext`를 중심 회전한 것이다. 그대로 재면 좌표 대조는
// 통과하면서(측정값과 산출물이 같으므로) 산출물만 틀린다. 그래서 측정 뷰에서 도형의 회전
// 선언을 무효화하고(render/web.shapeCSS), 회전은 사람이 보는 뷰에서만 적용된다.
//
// 발표자 뷰와 같은 수단(속성 하나)을 쓰는 이유: 뷰 전환의 구현이 하나면
// 측정과 발표가 같은 파일을 보면서도 각자에게 맞는 배치를 본다. `offsetWidth`/`offsetParent`
// 체인으로 회전 전 박스를 재는 길은 **두 번째 측정 경로**를 만들므로 쓰지 않는다 (측정이 두
// 벌이 되면 어느 쪽이 진짜인지 판정할 방법이 없다).
const MeasureView = "measure"

// OverviewView는 **오버뷰**의 ViewAttr 값이다 — 슬라이드 전부를 격자로 줄여 놓고 하나를 골라
// 들어가는 화면이다 (런타임의 `o`·Esc·컨트롤 바 버튼이 켜고 끈다).
//
// 발표자 뷰 안의 클래스가 아니라 **네 번째 뷰 값**인 이유:
// 뷰 전환의 구현이 하나면 측정·발표·오버뷰가 같은 파일을 보면서도 각자에게 맞는 배치를 본다.
// 줄이는 수단은 `zoom`이고 감싸는 요소를 만들지 않는다 — `transform`으로 격자를 만들려면
// 타일마다 래퍼가 필요하고, 그 요소는 지오메트리 패스가 읽는 부모 사슬을 바꾼다.
const OverviewView = "overview"

// 뷰어 크롬의 클래스다. 런타임 JS가 만드는 요소이므로 문서 골든에는 나오지 않지만,
// 발표자 뷰를 조작하는 테스트(internal/testkit)가 같은 문자열로 요소를 집는다 — 식별자
// 계약(IDAttr 등)과 같은 이유로 여기서 export한다.
const (
	// ViewerBarClass는 하단 가운데 컨트롤 바다.
	ViewerBarClass = "balpyo-viewer-bar"
	// ViewerPrevClass·ViewerNextClass는 이전/다음 버튼이다.
	ViewerPrevClass = "balpyo-viewer-prev"
	ViewerNextClass = "balpyo-viewer-next"
	// ViewerCounterClass는 "3 / 12" 카운터다.
	ViewerCounterClass = "balpyo-viewer-counter"
	// ViewerNotesToggleClass는 노트 패널 토글 버튼이다.
	ViewerNotesToggleClass = "balpyo-viewer-notes-toggle"
	// ViewerOverviewClass는 오버뷰 토글 버튼이다 (OverviewView).
	ViewerOverviewClass = "balpyo-viewer-overview"
	// ViewerNotesClass는 노트 패널이다 (숨은 슬라이드 노트의 내용을 보여준다).
	ViewerNotesClass = "balpyo-viewer-notes"
	// viewerActiveClass는 지금 보이는 슬라이드의 표식이다 (런타임이 토글한다).
	viewerActiveClass = "balpyo-viewer-active"
	// viewerEmptyClass는 노트가 없는 슬라이드에서 패널이 갖는 표식이다.
	viewerEmptyClass = "balpyo-viewer-empty"
	// viewerRevealedClass는 **이미 나타난** 순차 등장 항목의 표식이다 (런타임이 지금 스텝에
	// 맞춰 토글한다). Go 코드가 쓰지 않는 유일한 이 목록의 이름이지만
	// 여기 적는다: 계약 문자열의 목록이 한 곳에 있어야 자산(runtime.js·viewer.css) 두 벌이
	// 어긋난 것을 사람이 셀 필요가 없다.
	viewerRevealedClass = "balpyo-viewer-revealed"
)

// PresenterViewExpr는 발표자 뷰를 켜는 JS 표현식이다 — 런타임이 로드 시 하는 일과 같다.
// 발표자 뷰의 동작을 검증하는 테스트가 Session.Open이 되돌린 문서 뷰에서 다시 켤 때 쓴다.
func PresenterViewExpr() string {
	return fmt.Sprintf("document.documentElement.setAttribute(%q, %q)", ViewAttr, PresenterView)
}

// DocumentViewExpr는 문서 뷰로 되돌리는 JS 표현식이다 — 지오메트리 패스가 재기 전에
// 평가한다 (geom.Session.Open). 발표자 뷰 CSS는 전부 ViewAttr에 걸려 있으므로 속성을
// 지우는 것으로 충분하고, balpyo 산출물이 아닌 페이지에서도 아무 일도 하지 않아 안전하다.
func DocumentViewExpr() string {
	return fmt.Sprintf("document.documentElement.removeAttribute(%q)", ViewAttr)
}

// MeasureViewExpr는 측정 뷰를 켜는 JS 표현식이다 (MeasureView의 주석). 측정 스크립트가
// 재기 직전에 켜고 **끝나면 되돌린다** — 켠 채로 두면 그 뒤에 찍는 스크린샷에서 회전이
// 사라져 픽셀 비교가 산출물이 아닌 측정 상태를 본다.
func MeasureViewExpr() string {
	return fmt.Sprintf("document.documentElement.setAttribute(%q, %q)", ViewAttr, MeasureView)
}

//go:embed assets/viewer.css
var viewerCSSAsset string

// viewerCSS는 발표자 뷰와 뷰어 크롬의 CSS다 — baseCSS가 balpyo.base 층 안에 싣는다.
//
// 발표자 뷰 규칙은 전부 `html[data-balpyo-view="presenter"]` 아래에 있다. 게이트 밖에
// 남는 것은 (a) 문서 뷰에서 크롬을 숨기는 규칙과 (b) 게이트된 규칙이 이미 노출을 제어하는
// 크롬 내부 장식뿐이다 — 게이트 없는 규칙이 슬라이드 배치를 건드리면 측정이 그만큼 틀어진다.
//
// 크롬의 색·폰트는 테마 토큰이 아니다 (baseCSS의 뷰어 바탕색과 같은 판단): 슬라이드 밖의
// 색이라 덱의 색이 아니고, 대응하는 PPTX 개념도 없다. 폰트도 system-ui로 두어 테마 폰트
// 이름이 크롬에 새지 않게 한다.
//
// 자산은 층 안에 들어가는 규칙 묶음이므로 2칸 들여쓰기로 시작한다. 앞의 빈 줄은 앞 절과
// 띄우는 것이다 (contentCSS·chromeCSS도 같은 모양이다).
func viewerCSS() string { return "\n" + viewerCSSAsset }

// printPageCSS는 인쇄의 **종이 크기**다 — 논리 스테이지 그대로에 여백 0이므로 한 장이 한
// 페이지를 꽉 채운다. 되돌리기 규칙은 자산(viewer.css)의 @media print 절에 있고, 여기서
// 만드는 것은 스테이지에 따라 달라지는 이 한 규칙뿐이다 (`--balpyo-stage-width`와 같은 근거).
//
// 리터럴을 주는 이유: Chromium은 @page 안의 var()도 풀어 준다 (실측 — 같은 PDF가 나온다).
// 그러나 `size`를 이해하면서 커스텀 속성은 풀지 못하는 엔진에서는 선언이 통째로 무효가 되어
// 종이가 조용히 letter로 돌아간다. 값이 하나뿐인 자리에서 그 위험을 지지 않는다.
func printPageCSS(stage soml.Stage) string {
	return fmt.Sprintf(`
  @media print {
    @page { size: %dpx %dpx; margin: 0; }
  }
`, stage.Width, stage.Height)
}

//go:embed assets/favicon.png
var faviconPNG []byte

// faviconLink는 브라우저 탭에 balpyo 로고를 띄우는 <link>다.
//
// data URI로 인라인한다 — 산출물은 파일 하나로 열려야 하므로 아이콘도 외부 파일을 가리킬 수
// 없다. 자산은 원본 로고(1254px, 300KB)가 아니라 마크만 잘라 64px로 줄인 PNG다: 아이콘은
// 덱마다 실리므로 탭이 쓰는 해상도 이상은 산출물 크기만 늘린다. 바탕은 투명이라 어두운 탭
// 바에서도 흰 사각형이 뜨지 않는다.
//
// 덱 데이터와 무관한 고정 문자열이다 — 같은 입력에 같은 바이트가 나온다.
var faviconLink = `<link rel="icon" type="image/png" href="data:image/png;base64,` +
	base64.StdEncoding.EncodeToString(faviconPNG) + `">`

//go:embed assets/runtime.js
var runtimeJSAsset string

// runtimeJS는 뷰어 런타임이다 — Render가 <script>로 인라인한다 (외부 참조 없음).
//
// 덱 데이터를 보간하지 않는 **고정 문자열**이다: 슬라이드·노트는 전부 DOM에서 읽으므로
// 저자 텍스트가 스크립트로 새는 경로가 없다. 클래스·속성 리터럴은 이 패키지의 계약 상수와
// 같아야 한다 — 어긋나면 runtime_test가 잡는다 (baseCSS가 .balpyo-slide를 리터럴로 쓰는
// 것과 같은 관계다).
//
// 조작: 컨트롤 바(이전/다음/카운터/노트 토글), 키보드(화살표·Space·PageUp/Down·Home/End,
// n=노트), URL 해시 딥링크(`#/<슬라이드 id>` 또는 `#/<1-based 번호>`).
//
// 파일 끝의 줄바꿈은 뗀다 — Render가 </script> 앞에 하나 넣는다.
var runtimeJS = strings.TrimRight(runtimeJSAsset, "\n")
