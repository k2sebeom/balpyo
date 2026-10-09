package soml

// 덱 크롬은 슬라이드마다 반복되는 덱 전체의 한 줄이다 — 페이지 번호와 푸터.
//
// **태그 어휘가 아니다.** 파서는 이 파일의 타입을 모른다: 내용과 켜짐은 `balpyo.json`이,
// 자리·정렬·역할·제외 규칙은 테마의 `chrome` 선언이 소유하고, 값 층이 둘을 합쳐 슬라이드마다
// ChromeItem을 확정한다. 크롬을 `<footer>` 태그로 열지 않은 이유: 같은
// 문자열을 슬라이드 수만큼 쓰게 되고, 스테이지 흐름에 들어가 본문 높이를 먹으며,
// 페이지 번호는 애초에 저자가 쓸 수 없는 값이다.

// ChromeKind는 크롬 종류다. 어휘는 2종으로 닫혀 있다 — 러닝 헤더·날짜·로고는 덱이 실제로
// 막힐 때 연다.
type ChromeKind string

const (
	// ChromeFooter는 덱 전체에 반복되는 한 줄이다 (`balpyo.json`의 footer 문자열).
	ChromeFooter ChromeKind = "footer"
	// ChromePageNumber는 슬라이드 번호다. PPTX에서 `a:fld type="slidenum"`이 되므로 뷰어에서
	// 장을 더하면 번호가 따라온다 — 정적 텍스트로 내면 그 약속이 깨진다.
	ChromePageNumber ChromeKind = "pageNumber"
	// ChromeLogo는 테마의 로고 그림이다. 다른 둘과 달리 **내용과 켜짐이 테마의 것**이다 — 그림이
	// 테마 패키지의 자산이고, 선언하면 켜진다 (`balpyo.json`이 정할 것이 없다).
	ChromeLogo ChromeKind = "logo"
)

// ChromeKinds는 크롬 종류 어휘다 (진단 문구의 순서이자 선언 검사 순서).
func ChromeKinds() []ChromeKind { return []ChromeKind{ChromeFooter, ChromePageNumber, ChromeLogo} }

// ChromeAligns는 크롬이 띠 안에서 붙을 쪽이다 — TextAligns에서 `justify`를 뺀 3종이다.
// 크롬 박스는 내용 폭이라(띠를 가로지르지 않는다) 양쪽 정렬이 가리킬 여백이 없다.
func ChromeAligns() []TextAlign { return []TextAlign{AlignStart, AlignCenter, AlignEnd} }

// DeckChrome은 덱 크롬 **요청**이다 (`balpyo.json`의 두 필드). 파이프라인이 설정에서 옮기고
// 값 층이 테마 선언과 합친다 — Title·Author·Lang과 같은 자리의 덱 사실이다.
//
// 요청과 확정(Slide.Chrome)을 따로 두는 이유: 요청은 덱에 하나이고 확정은 슬라이드마다다
// (제외 규칙이 레이아웃으로 걸리고, 번호가 슬라이드마다 다르다).
type DeckChrome struct {
	// Footer는 푸터 한 줄이다. 빈 문자열이면 푸터가 꺼져 있다.
	Footer string
	// PageNumber는 페이지 번호를 켤지다.
	PageNumber bool
}

// Empty는 요청이 아무것도 켜지 않았는지다 — 그때 확정 모델은 예전과 같다 (골든 불변).
func (c DeckChrome) Empty() bool { return c.Footer == "" && !c.PageNumber }

// ChromeItem은 슬라이드 하나에 실릴 확정된 크롬 하나다.
//
// 두 렌더러가 읽는 입력은 이 구조체뿐이다: 제외 규칙도, 역할 → 외관 캐스케이드도, 번호
// 문자열도 값 층이 이미 적용했다. 그 조건을 렌더러마다 적으면 웹과 PPTX가 갈라진다.
type ChromeItem struct {
	Kind ChromeKind
	// Text는 화면에 나갈 글자다. 번호는 **덱 안의 위치**(1-based)이고 제외된 장도 센다 —
	// `a:fld type="slidenum"`이 주는 값이 그것이라 웹이 다른 수를 찍으면 두 타깃이 갈린다.
	// PPTX는 이 값을 필드의 캐시 텍스트로 싣는다 (필드를 계산하지 않는 뷰어도 맞는 숫자를 본다).
	Text string
	// Align은 바닥 띠에서 붙을 쪽이다 (ChromeAligns).
	Align TextAlign
	// Style은 테마 역할에서 확정된 텍스트 외관이다 — 크롬은 자기 크기·색을 갖지 않는다.
	// 그림 크롬(로고)에는 없다 (영값).
	Style TextStyle
	// Image는 그림 크롬의 에셋 키다 (Deck.Assets — 테마 자산은 `themes/<테마>/assets/…`). 글자 크롬에서는
	// 비어 있다.
	Image string
	// Height는 그림 크롬의 높이다 (논리 px) — 폭은 그림의 비율이 정한다.
	Height float64
	// Alt는 그림 크롬의 대체 텍스트다 (웹 `alt` · PPTX `cNvPr@descr`).
	Alt string
}
