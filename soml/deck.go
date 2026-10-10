// Package soml은 SOML 파서와 덱 모델을 담는다.
//
// 덱 모델(Deck)이 이 제품의 중심이다 — SOML은 저작 문법이고, 각 타깃 렌더러는 덱 모델을
// 소비한다. 그래서 이 패키지는 외부에서 임포트 가능한 공개 API로 유지한다 (생성기·에이전트
// 같은 제3자 도구가 재사용한다).
package soml

import "fmt"

// Aspect는 슬라이드 종횡비다. v1은 2종만 받는다 (16:9·4:3).
type Aspect string

const (
	// Aspect16x9는 논리 스테이지 1280×720이다.
	Aspect16x9 Aspect = "16:9"
	// Aspect4x3은 논리 스테이지 960×720이다.
	Aspect4x3 Aspect = "4:3"
)

// DefaultAspect는 aspect를 지정하지 않은 덱의 비율이다.
const DefaultAspect = Aspect16x9

// Stage는 논리 스테이지 크기다. 단위는 px 하나뿐이며 저자·웹 렌더러·지오메트리
// 패스가 모두 이 좌표계를 쓴다. 타깃 단위(EMU, pt) 변환은 렌더러가 소유한다.
type Stage struct {
	Width  int
	Height int
}

// Stage는 종횡비에 대응하는 논리 스테이지를 돌려준다. 알 수 없는 비율은 에러다.
//
// 두 비율은 높이 720을 공유한다 — 폰트 크기와 수직 리듬이 비율을 넘어 그대로 통하고,
// px에 상수 9525를 곱하면 PowerPoint 표준 페이지 크기(13.333×7.5in / 10×7.5in)가
// 정확히 나온다.
func (a Aspect) Stage() (Stage, error) {
	switch a {
	case Aspect16x9:
		return Stage{Width: 1280, Height: 720}, nil
	case Aspect4x3:
		return Stage{Width: 960, Height: 720}, nil
	default:
		return Stage{}, fmt.Errorf("unknown aspect ratio %q (%q or %q)", string(a), Aspect16x9, Aspect4x3)
	}
}

// SlideLayout은 슬라이드의 배치 레이아웃이다 (`<slide layout>`). 어휘는 표준 카탈로그
// 11종이다.
//
// Aspect와 나란히 여기 있는 이유: 둘 다 슬라이드의 **구조**를 정하는 닫힌 어휘이고 외관이
// 아니다. 외관 어휘(role·align)가 테마와 함께 자라 appearance.go에 있는 것과 갈라진다 —
// 표준 레이아웃 카탈로그의 **이름**은 엔진이 소유하고(이 목록이 닫힌 어휘다), 테마가 더하는
// 것은 그 위에 얹는 자체 레이아웃(`layout="corp-hero"`)이다. 표준 이름의 **배치 구현**은
// 테마가 소유하며(layouts/*.css), 테마는 11종을 **전부** 구현해야 한다 — 빠뜨린 레이아웃을
// 다른 테마로 폴백하지 않는 이유: 한 덱이 두 디자인 시스템으로 나온다.
// 어휘 밖 이름(오타·테마에도 없는 이름)은 빌드 에러다.
type SlideLayout string

const (
	// LayoutTitle은 표지다 — 슬롯 kicker·title·subtitle·author·date.
	LayoutTitle SlideLayout = "title"
	// LayoutSectionDivider는 챕터 구분이다 — 슬롯 title·subtitle.
	LayoutSectionDivider SlideLayout = "section-divider"
	// LayoutContent는 제목 + 본문 흐름이다 (기본값) — 슬롯 title·body. 세로 스택 스테이지의
	// 이름이기도 하다 (render/web.writeStage): `layout`을 생략한 슬라이드의 기본값이자,
	// 저자가 명시할 수 있는 정당한 표기다.
	LayoutContent SlideLayout = "content"
	// LayoutTwoColumn은 2단 비교·병렬이다 — 슬롯 title·left·right.
	LayoutTwoColumn SlideLayout = "two-column"
	// LayoutImageFull은 전면 이미지다 — 슬롯 media·caption.
	LayoutImageFull SlideLayout = "image-full"
	// LayoutImageText는 이미지+텍스트 조합이다 — 슬롯 title·media·body.
	LayoutImageText SlideLayout = "image-text"
	// LayoutQuote는 인용이다 — 슬롯 quote·attribution.
	LayoutQuote SlideLayout = "quote"
	// LayoutCode는 코드 중심이다 — 슬롯 title·code·caption.
	LayoutCode SlideLayout = "code"
	// LayoutStat은 수치 강조 콜아웃이다 — 슬롯 title·value·label·body.
	LayoutStat SlideLayout = "stat"
	// LayoutCanvas는 자유 배치다 — 슬롯이 없고, 스테이지가 자식을 세로로 쌓지 않고 자식이
	// `x`/`y`·`at`으로 자기 자리를 정한다 (테마는 배경·여백·외관 기본값만 소유한다).
	LayoutCanvas SlideLayout = "canvas"
	// LayoutEnd는 마무리다 — 슬롯 title·contact.
	LayoutEnd SlideLayout = "end"
)

// DefaultSlideLayout은 `layout`을 생략한 슬라이드의 레이아웃이다 — 속성이 없을 때의 기본값과
// 저자가 명시한 `layout="content"`가 같은 이름을 갖는다 (그래서 둘의 산출물이 바이트가 같다).
// 어휘 밖 layout 값의 **폴백 기준**이기도 하다 (core.slideLayout의 진단).
const DefaultSlideLayout = LayoutContent

// SlideLayouts는 표준 레이아웃 어휘 전체다 (카탈로그 순서). 순서가 진단 메시지에
// 실리는 순서이자 문서 순서다 — 표지(title)에서 마무리(end)로 발표의 흐름을 따른다.
func SlideLayouts() []SlideLayout {
	return []SlideLayout{
		LayoutTitle, LayoutSectionDivider, LayoutContent, LayoutTwoColumn,
		LayoutImageFull, LayoutImageText, LayoutQuote, LayoutCode, LayoutStat,
		LayoutCanvas, LayoutEnd,
	}
}

// SlotAttr는 슬롯 배정 속성이다 (`slot="title"`) — 레이아웃 슬라이드의 직계 자식이 어느
// 슬롯에 들어갈지 정한다. 슬롯 없는 직계 자식은 기본 슬롯(DefaultSlot)에 들어간다.
//
// 채널을 넓게 열고 좁게 검증하는 것이 계약이다 (`x`/`y`와 같은 태도): 속성 allowlist는
// 태그별이라 위치를 모르므로(같은 `<text>` spec이 슬라이드 직속에도 컨테이너 안에도 쓰인다)
// 슬라이드 직계 자식이 될 수 있는 요소 전부에 `slot`을 열고, "슬롯은 슬라이드 직계 자식에만",
// "그 레이아웃에 그 슬롯이 있는가"는 부모와 레이아웃을 아는 값 층(core)이 검증한다.
const SlotAttr = "slot"

// DefaultSlot은 `slot`을 지정하지 않은 직계 자식이 들어가는 슬롯 이름이다.
const DefaultSlot = "body"

// Deck은 정규화된 덱 모델이다. 렌더러가 보는 유일한 입력이며, 여기 도달한 시점에는
// 합성 컴포넌트가 전개되고 외관이 해석되어 있다.
type Deck struct {
	Title  string
	Author string
	Lang   string
	Aspect Aspect
	Stage  Stage
	// ThemeName은 **테마 요청**이다 (`balpyo.json`의 theme 또는 CLI의 -theme).
	// 파이프라인이 설정에서 옮기고 core.Resolve가 이 이름으로 테마 패키지를 로드한다.
	//
	// 아래 Theme(결과)의 형제이고, 둘이 나란히 있는 모양은 크롬과 같다 (Chrome 요청 →
	// Slide.Chrome 결과): 읽는 사람이 "무엇이 입력이고 무엇이 파생인가"를 구조체 하나에서 본다.
	// 빈 문자열은 기본 테마다 — 설정 없이 Deck을 만드는 길(테스트·프리뷰)이 있으므로 빈 값이
	// 에러가 아니고, 어휘 밖 **이름**은 에러다 (폴백 테마는 없다).
	ThemeName string
	// Theme는 해석된 외관의 색·폰트 참조가 가리키는 토큰 표다 (appearance.go). core.Resolve가
	// ThemeName이 가리킨 패키지에서 채우고, 두 렌더러가 이 한 표에서 CSS 변수와 PPTX 테마
	// 파트를 만든다.
	Theme Theme
	// Assets는 `<image src>`가 가리킨 파일의 바이트·포맷·고유 크기다 (asset.go).
	// core.ResolveAssets가 채우고 두 렌더러가 읽는다 — **Theme의 형제**다: 파생 데이터가
	// 아니라 입력이고, 렌더 전에 결정적으로 한 번 채워지므로 타깃 순서와 무관하다
	// (곁 표로 두지 않은 근거는 asset.go의 패키지 주석에 있다).
	Assets Assets
	// Chrome은 덱 크롬 요청이다 (`balpyo.json`의 footer·pageNumber). 파이프라인이
	// 설정에서 옮기고 core.Resolve가 테마 선언과 합쳐 슬라이드마다 Slide.Chrome을 확정한다.
	Chrome DeckChrome
	Slides []*Slide
}

// Slide는 슬라이드 하나다 — `.slide` 파일 하나에 대응한다.
type Slide struct {
	// Index는 0-based 렌더 순서다 (Skip된 슬라이드는 애초에 Deck에 없다).
	Index int
	// ID는 딥링크 앵커다. 생략 시 파일명에서 유도한다 (SlideFile.DerivedID).
	ID string
	// IDSpan은 ID가 정해진 자리다 (SlideDoc.IDSpan) — ID 충돌 진단의 위치.
	IDSpan Span
	// Source는 프로젝트 루트 기준 상대 경로다. 진단이 가리키는 파일.
	Source string
	// Chapter는 슬라이드가 속한 하위 디렉토리 경로다. 루트 직속이면 빈 문자열이다.
	Chapter string
	// Layout은 슬라이드 배치 레이아웃이다 (`<slide layout>`). 파이프라인이
	// SlideDoc.Layout(원문)을 옮기고 core.Resolve가 2종 어휘를 확정한다 — 어휘 밖 값은
	// 진단이 되고, 생략된 슬라이드는 DefaultSlideLayout(content)으로 채워진다. 렌더러는
	// 문자열이 아니라 이 타입을 소비한다 — `"canvas"`가 세 렌더러에 흩어지지
	// 않게 한다.
	Layout SlideLayout
	// LayoutSpan은 layout 값 구간이다 — core의 어휘 진단이 부호가 아니라 값을 가리키게
	// 한다. Slide에 루트 요소 참조가 없으므로(덱 모델은 해석이 끝난 값이다) 값과 함께
	// 구간을 나른다.
	LayoutSpan Span
	// Justify는 **흐름 모드** 스테이지의 세로 분배다 (`<slide justify>`).
	// 파이프라인이 원문을 옮기고 core.Resolve가 6종 어휘를 확정한다 (컨테이너의 justify와
	// 같은 어휘 — BoxJustifies). 빈 값은 "선언 없음"이고 그때 스테이지는 위에서부터 쌓는다.
	// 슬롯 모드 슬라이드에서는 에러이므로(세로 리듬은 테마의 `*.slots.css`가 소유한다)
	// 렌더러는 이 값이 있으면 흐름 모드라고 믿을 수 있다.
	Justify BoxJustify
	// JustifySpan은 justify 값 구간이다 (LayoutSpan과 같은 근거).
	JustifySpan Span
	// Enter는 `<slide enter>`의 원문이다 — 직속 자식마다 같은 등장을 준다. 어휘는 core.Resolve가 본다.
	Enter string
	// EnterSpan은 enter 값 구간이다.
	EnterSpan Span
	// Invert는 이 장의 색 매핑을 뒤집는지다 (`<slide invert>`) — 배경 · 글자 네 자리(bg1↔tx1, bg2↔tx2)가 서로
	// 바뀐다. 렌더러는 이 값 하나로 PPTX `overrideClrMapping`과 웹 토큰 변수의 재선언을 낸다.
	Invert bool
	// Transition은 이 장에 들어올 때의 전환이다. 파이프라인이 원문을 옮기고 core.Resolve가 테마의 기본값과
	// 합쳐 **확정 값**(fade · push · 빈 값)으로 바꾼다 — 렌더러는 확정 값만 읽는다.
	Transition SlideTransition
	// TransitionSpan은 transition 값 구간이다.
	TransitionSpan Span
	// TransitionDuration은 확정된 전환의 길이(ms)다 (전환이 없으면 0).
	TransitionDuration int
	// Notes는 발표자 노트다 (`<note>`). PPTX에서는 노트 슬라이드가 된다.
	Notes string
	// Chrome은 이 슬라이드에 실릴 확정된 덱 크롬이다 — 테마 선언 순서를 지킨다.
	//
	// Content의 형제이고 그 안이 아닌 이유: 크롬은 슬라이드의 내용이 아니라 덱의 사실이고,
	// 콘텐츠 트리에 넣으면 스테이지 흐름에 끼어 본문 높이를 먹는다 (그리고 Refs가 재는
	// 대상이 되어 배치 판정이 저자가 고칠 수 없는 경고를 낸다). 제외 규칙이 이미 적용된
	// 목록이므로 표지·섹션 구분에서는 비어 있다 — 렌더러는 이 목록을 그대로 그린다.
	Chrome []ChromeItem
	// Background는 테마가 이 슬라이드의 레이아웃에 정한 배경이다 (nil이면 테마의 `bg` 색 그대로 —
	// 배경을 선언하지 않은 테마의 산출물은 예전과 같다). Chrome처럼 덱의 사실을 슬라이드마다 확정한
	// 값이고, 렌더러는 규칙을 모르고 이 값만 그린다.
	Background *Background
	// Content는 슬라이드 콘텐츠 트리다 (`<note>`는 여기 없다 — Notes로 간다).
	Content []Node
	// Tokens는 이 슬라이드를 만든 원문 토큰 열이다. format이 원문을 되돌려 쓰고,
	// 진단이 위치를 찾는 근거다.
	Tokens Tokens
}

// Node는 슬라이드 콘텐츠 트리의 노드다. 구현체는 Element와 Text다 (node.go).
type Node interface {
	// NodeSpan은 이 노드를 만든 원문 구간이다 — 렌더 단계의 진단도 위치를 갖는다.
	NodeSpan() Span
}

// Background는 슬라이드 배경 하나다 — 채움이거나 그림이다 (정확히 하나).
//
// 배경은 슬라이드의 **내용이 아니다**: 콘텐츠 트리에 넣으면 도형이 되어 저자가 PowerPoint에서 그것을
// 끌어 옮기고, 배치 판정이 그것을 잰다. 웹은 슬라이드 요소의 배경, PPTX는 슬라이드의 `p:bg`다 —
// 둘 다 "슬라이드 바탕"이라는 같은 자리다.
type Background struct {
	// Fill은 바탕 채움이다 (단색 · 그라디언트 — 토큰 참조). 그림 배경이면 nil이다.
	Fill *Fill
	// Image는 바탕 그림의 에셋 키다 (비면 그림 없음). 그림은 슬라이드를 **덮는다**(cover):
	// 비율을 지키고 넘치는 쪽을 가운데 기준으로 잘라 낸다.
	Image string
}
