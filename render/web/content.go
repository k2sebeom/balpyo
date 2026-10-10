package web

import (
	"fmt"
	"html"
	"sort"
	"strconv"
	"strings"

	"github.com/k2sebeom/balpyo/soml"
)

// IDAttr은 렌더된 요소의 식별자 속성이다 — 지오메트리 패스가 브라우저 rect를 덱 요소에
// 되돌려 붙이는 열쇠다. 값 규칙은 패키지 주석에 있다.
const IDAttr = "data-balpyo-id"

// ElAttr은 그 요소를 만든 SOML 태그 이름이다. 식별자만으로 매핑은 되지만, 지오메트리 패스가
// 덱 트리를 훑으며 이 이름을 대조하면 렌더러와 지오메트리 패스가 어긋난 순간(한쪽이 요소를
// 더/덜 내는 경우) 좌표가 엉뚱한 요소에 붙는 대신 소리를 낸다.
const ElAttr = "data-balpyo-el"

// RevealAttr은 리스트 항목이 나타나는 스텝 번호다 (`<li reveal>`의 확정 값). 속성을 쓰지
// 않은 항목에는 **실리지 않는다**: 없음이 곧 "처음부터 보인다"라,
// reveal을 쓰지 않은 덱의 마크업이 이 기능 전과 같다.
//
// 스텝이 클래스가 아니라 속성인 이유는 판정의 주인이 둘이라는 것이다 — CSS는 "스텝을 가진
// 항목"까지만 겨눌 수 있고(수를 비교할 수 없다) 지금 스텝 이하인지는 런타임이 정해
// RevealedClass로 말한다.
const RevealAttr = "data-balpyo-reveal"

// 이 셋을 export하는 이유: 계약의 **양쪽**이 같은 문자열을 써야 한다. 지오메트리 패스는
// 이 선택자로 요소를 집고(render/geom), 테스트 하네스는 같은 선택자로 슬라이드를 찍는다
// (internal/testkit). 선택자 문자열이 세 곳에 흩어지면 클래스 이름을 바꾼 순간 한 곳만
// 남아 "요소를 못 찾았다"가 아니라 "좌표가 다르다"로 나타난다.
const (
	// SlideClass는 슬라이드 요소의 클래스다.
	SlideClass = "balpyo-slide"
	// SlideIndexAttr는 슬라이드의 0-based 렌더 순서다 (soml.Slide.Index).
	SlideIndexAttr = "data-slide-index"
	// NotesClass는 슬라이드 안의 숨은 발표자 노트 요소다 — 뷰어 런타임의 노트 패널과
	// 테스트가 이 클래스로 읽는다.
	NotesClass = "balpyo-notes"
	// StageClass는 슬라이드 안의 스테이지 요소다 — **콘텐츠 박스를 가진 박스**다.
	//
	// 지오메트리 패스가 이 요소의 패딩 안쪽을 재기 때문에 export한다 (geom의 콘텐츠 박스):
	// 여백은 StagePadding이 기본값이지만 테마의 레이아웃 CSS가 레이아웃마다 다시 정할 수
	// 있으므로(minimal의 `image-full.css`는 `padding: 0`이다) 판정 기준을 상수로 계산하면
	// 전면 배치 레이아웃에서 저자가 고칠 수 없는 경고가 나온다.
	StageClass = "balpyo-stage"
)

// SlideSelector는 문서 순서의 모든 슬라이드를 집는 선택자다.
func SlideSelector() string { return "." + SlideClass }

// SlideIndexSelector는 인덱스로 슬라이드 하나를 집는 선택자다.
func SlideIndexSelector(index int) string {
	return fmt.Sprintf("%s[%s=%q]", SlideSelector(), SlideIndexAttr, strconv.Itoa(index))
}

// ElementSelector는 식별자를 가진 모든 요소를 집는 선택자다 — 지오메트리 패스가 재는 대상.
func ElementSelector() string { return "[" + IDAttr + "]" }

// StageSelector는 슬라이드 안의 스테이지를 집는 선택자다 (StageClass의 주석).
func StageSelector() string { return "." + StageClass }

// 스테이지 여백과 스택 간격이다 (논리 px). 간격 24px은 본문 크기와 같아 한 글자 높이만큼
// 벌어진 리듬이 된다 — 역할 기본값의 수직 리듬(테마의 text.roles)과 같은 근거다.
//
// StagePadding을 export하는 이유는 ListTextIndent와 같다: 콘텐츠 박스(스테이지에서 이 여백을
// 뺀 사각형)를 배치 진단이 판정 기준으로 쓴다 (render/geom의 ContentBox). 값이 두
// 벌이 되면 "여백을 침범했다"는 진단이 실제 여백과 어긋난 채 나가고, 그 어긋남은 진단을
// 믿지 못하게 만든다.
const (
	// StagePadding은 스테이지 네 변의 여백이다 (논리 px).
	StagePadding = 64
	stackGap     = 24
)

// 리스트 배치 상수다 (논리 px). export하는 이유: PPTX 렌더러의 `marL`/`indent`가
// **같은 값**을 써야 두 타깃의 들여쓰기가 한 곳에서 나온다 — web.ElementID를 geom·pptx가
// 쓰는 것과 같은 단일 구현 원칙이다. 값이 두 벌이 되면 갈라진 것은 두 산출물을 나란히
// 볼 때까지 드러나지 않는다.
//
// 마커 배치는 **PPTX의 관례를 웹이 따른다**: OOXML은 마커를 `marL+indent` 위치에
// 왼쪽 정렬로 그리고 텍스트를 `marL`에서 시작한다 — 뷰어가 그리는 방식이라 우리가 바꿀 수
// 없다. 반면 CSS의 기본 마커(list-style-position: outside)는 텍스트 왼쪽에 **오른쪽 정렬**
// 이라 두 관례가 어긋난다. CSS는 ::before로 OOXML의 모델을 정확히 표현할 수 있지만 역은
// 불가능하므로 웹이 맞추는 쪽이다 (contentCSS의 리스트 절).
//
// 값 36은 PowerPoint 1수준 기본(marL=342900 EMU, indent=-342900)과 정확히 같다 —
// 342900 EMU ÷ 9525(EMU/px) = 36px. 익숙한 기본값을
// 벗어날 이유가 없고, PPTX 쪽 환산이 정수로 떨어진다.
const (
	// ListTextIndent는 li 텍스트가 리스트 박스 왼쪽에서 들어와 시작하는 거리다 —
	// PPTX `pPr@marL`이 이 값이 된다 (li의 padding-left).
	ListTextIndent = 36
	// ListMarkerHang은 마커 시작이 텍스트 시작보다 왼쪽으로 나가는 거리다 — PPTX
	// `pPr@indent`가 -ListMarkerHang이 된다. 마커와 텍스트 사이의 시각적 간격은
	// (이 값 − 마커 글리프 폭)으로 **파생**된다: OOXML도 간격을 직접 갖지 않고 마커
	// 시작 위치만 가지므로, 간격을 상수로 두면 PPTX가 표현할 수 없는 값이 된다.
	ListMarkerHang = 36
	// ListItemGap은 항목 사이 간격이다. 0 — 수직 리듬은 줄높이가 소유한다 (역할
	// 기본값의 lineHeight). 여기에 간격을 더하면 리듬의 출처가 둘이 된다.
	ListItemGap = 0
)

// ListBulletChar는 ul 마커 문자다. CSS `list-style-type: disc`의 원(브라우저가 그리는
// 도형)이 아니라 **폰트 글리프**를 쓰는 이유: PPTX의 `buChar`는 이 문자를 텍스트 폰트로
// 그리므로, 웹도 같은 글리프를 내야 두 타깃의 마커 모양이 같은 곳에서 나온다. ol은
// `counter() "."`로 내며 PPTX `buAutoNum@type="arabicPeriod"`("1.")와 짝이다.
const ListBulletChar = "\u2022" // •

// writeStage는 슬라이드의 콘텐츠 노드를 스테이지 div에 쓴다.
//
// 자식 배치는 **문서 순서의 세로 스택**이다 — 레이아웃 태그(`<row>`/`<col>`/`<grid>`)를
// 쓰지 않은 슬라이드의 기본 배치를 이 함수가 정의한다. 스테이지를 `<col>`로 재정의하면
// 기본 배치의 출처가 한 곳이 되지만, 그러면 CSS 문자열이 바뀌어 골든 세 개가 움직이고 그
// diff는 레이아웃 태그가 무엇을 하는지 말해 주지 못한다 — 레이아웃마다 다른 배치는 테마의
// 레이아웃 CSS가 스테이지에 얹는다 (layoutStagesCSS). 세로 스택을 고른 이유:
//
//   - 저자가 쓴 순서와 화면 순서가 같아야 SOML만 읽고 결과를 예측할 수 있다.
//   - 자식마다 **확정된 폭**(스테이지 콘텐츠 폭)을 주면 브라우저의 줄바꿈이 결정적이다.
//     지오메트리 패스가 잰 rect가 그대로 PPTX 텍스트 박스가 되고 autofit은 꺼지므로
//     폭이 내용에 따라 변하는 배치는 PowerPoint에서 줄바꿈이 달라질 여지를
//     만든다. 폭을 고정하면 두 타깃이 같은 폭 안에서 줄을 나눈다.
//   - 세로 정렬(가운데·아래)은 레이아웃의 선택이지 텍스트의 성질이 아니다 — 표지의 가운데
//     정렬은 `layout="title"`의 테마 배치가 소유한다. 기본은 위에서 아래로 쌓는다.
func writeStage(b *strings.Builder, deck *soml.Deck, slide *soml.Slide) error {
	// 캔버스 스테이지는 stack이다 — 스테이지가 단일 셀 grid가 되어 자식이
	// `x`/`y`·`at`으로 자기 자리를 정한다. 세로 스택(기본)과 갈라지는 자리는 이 클래스
	// 하나뿐이고, 겹침 CSS는 stack과 한 벌을 공유한다 (stackCSS). 스테이지를 `<col>`로
	// 재정의하지 않는 이유는 위의 골든 근거와 같다.
	stageClass := StageClass
	if slide.Layout == soml.LayoutCanvas {
		stageClass += " balpyo-stage-canvas"
	}
	// 배치 CSS를 가진 레이아웃(title·two-column 등)은 스테이지에 레이아웃 클래스를 얻어 테마가
	// 얹은 배치 규칙(themeCSS)의 표적이 된다. 배치 CSS가 없는 레이아웃(content 세로 스택·canvas
	// 자유 배치는 렌더러가 소유)은 맵에 없으므로 클래스가 붙지 않는다 — 그래야 그 레이아웃의
	// 산출물이 이 기능 때문에 움직이지 않는다 (soml.Theme.Layouts의 근거).
	// 슬롯 모드는 **슬라이드마다** 판정하고, 판정의 입력은 확정 모델 하나다: 이 슬라이드의
	// 직계 자식이 슬롯을 실었는가. 테마에 슬롯 영역이 있는지는
	// **여기서 보지 않는다** — core가 그 조건을 이미 보고, 테마가 못 받는 슬롯은 경고와 함께
	// 확정 모델에 싣지 않는다. 그 조건을 렌더러마다 적으면 웹과 PPTX가 갈라진다. 슬롯을
	// 쓰지 않은 슬라이드는 흐름 배치 그대로다 — 그래야 슬롯을 안 쓴 덱의
	// 산출물이 이 기능 때문에 움직이지 않는다 (골든 불변). 슬롯 모드에서도 레이아웃 클래스를
	// 붙여 슬롯 영역 선택자(`.balpyo-layout-<이름>.balpyo-slots`)가 맞도록 한다.
	_, hasFlow := deck.Theme.Layouts[string(slide.Layout)]
	slotMode := slideUsesSlots(slide)
	if hasFlow || slotMode {
		stageClass += " balpyo-layout-" + string(slide.Layout)
	}
	if slotMode {
		stageClass += " balpyo-slots"
	}
	// 슬라이드의 세로 분배는 스테이지의 인라인 선언이다 (`<slide justify>`).
	// 흐름 스테이지는 세로 flex이므로(base 층의 `.balpyo-stage`) `justify-content`가 그대로
	// 세로 분배다. 인라인인 이유는 요소마다 다른 값이라는 것이다 — 테마 층 CSS는 레이아웃마다
	// 고정인 배치를 담고(layoutStagesCSS), 슬라이드 하나의 선택은 그 요소에 실린다
	// (컨테이너의 gap·align이 인라인으로 나가는 것과 같은 경계). 슬롯 모드에서는 core가 이미
	// 에러로 막았으므로(세로 리듬은 테마의 몫) 이 값이 있으면 흐름 모드다.
	var stageStyle string
	if slide.Justify != "" {
		value, err := boxJustifyValue(slide.Justify)
		if err != nil {
			return fmt.Errorf("%s: %w", slide.Source, err)
		}
		stageStyle = styleAttr([]string{"justify-content: " + value})
	}
	if len(slide.Content) == 0 {
		// 빈 스테이지는 빈 요소 하나다 — 콘텐츠가 없는 슬라이드의 산출물이 콘텐츠 렌더가
		// 붙었다는 이유로 달라지면 골든 diff가 무엇이 움직였는지 말해 주지 못한다.
		fmt.Fprintf(b, "<div class=%q%s></div>\n", stageClass, stageStyle)
		return nil
	}
	fmt.Fprintf(b, "<div class=%q%s>\n", stageClass, stageStyle)
	for i, node := range slide.Content {
		if err := writeNode(b, deck, slide, node, []int{i}); err != nil {
			return err
		}
	}
	b.WriteString("</div>\n")
	return nil
}

// slideUsesSlots는 이 슬라이드의 **직계 자식** 중 슬롯 배정을 받은 요소가 있는지다.
//
// 직계 자식만 보는 이유는 core가 슬롯을 거기에만 싣기 때문이다 (resolveSlot — 중첩된 요소의
// slot은 에러). 그래서 이 판정과 `grid-area` 방출(appendSlot)이 같은 요소 집합을 겨눈다:
// 스테이지에 `.balpyo-slots`가 켜지면 그 자식들의 grid-area가 뜻을 얻는다. 슬롯을 안 쓴
// 슬라이드는 이 함수가 거짓을 주어 흐름 배치 그대로다 (골든 불변).
func slideUsesSlots(slide *soml.Slide) bool {
	for _, node := range slide.Content {
		if elem, ok := node.(*soml.Element); ok && elem.Appearance != nil && elem.Appearance.Slot != "" {
			return true
		}
	}
	return false
}

// nodeWriter는 요소 종류 하나를 웹으로 쓰는 함수다.
type nodeWriter func(b *strings.Builder, deck *soml.Deck, slide *soml.Slide, elem *soml.Element, path []int) error

// renderedKinds는 이 렌더러가 다루는 요소 종류의 **명시 목록**이다 (writerFor의 짝).
//
// switch 하나로 끝내지 않고 목록을 함께 두는 이유는 soml이 카탈로그를 목록으로 둔 이유와
// 같다: 어휘가 층마다 다른 속도로 자라므로 **대조할 수 있어야** 한다. `soml.KindRendered`가
// 참인 종류는 전부 여기 있어야 하고 거짓인 종류는 없어야 한다 — 그 대조는 render 패키지의
// 테스트가 하고, 목록과 writerFor가 어긋나지 않는지는 이 패키지의 테스트가 본다
// (파싱·검증을 통과한 종류가 렌더에서 빌드를 죽이지 않게 한다).
// `soml.KindNone`(`<slide>`·`<text>`·`<note>`)은 카탈로그 밖이라 여기 없다 — writeNode와
// writeStage가 이름으로 직접 다룬다.
var renderedKinds = []soml.ElementKind{
	soml.KindBlock,
	soml.KindListItem,
	soml.KindListContainer,
	soml.KindLayoutContainer,
	soml.KindShape,
	soml.KindImage,
	soml.KindSpacer,
	soml.KindInline,
	soml.KindTable,
	soml.KindTableRow,
	soml.KindTableCell,
	soml.KindConnector,
}

// RenderedKinds는 이 렌더러가 다루는 종류다 (선언 순서). soml 카탈로그와 대조하는 테스트가
// 쓴다 — 렌더러 바깥에서 "무엇을 그릴 수 있는가"를 물을 수 있는 유일한 창구다.
func RenderedKinds() []soml.ElementKind {
	return append([]soml.ElementKind(nil), renderedKinds...)
}

// writerFor는 종류 하나를 쓰는 함수다. 목록 밖의 종류는 nil이다.
func writerFor(kind soml.ElementKind) nodeWriter {
	switch kind {
	// 블록과 li는 저자가 쓴 이름 그대로 시맨틱 태그로 낸다 (근거는 writeBlock).
	case soml.KindBlock, soml.KindListItem:
		return writeNamedBlock
	case soml.KindListContainer:
		return writeList
	case soml.KindLayoutContainer:
		return writeContainer
	case soml.KindShape:
		return writeShape
	case soml.KindImage:
		return writeImage
	case soml.KindSpacer:
		return func(b *strings.Builder, _ *soml.Deck, slide *soml.Slide, elem *soml.Element, path []int) error {
			return writeSpacer(b, slide, elem, path)
		}
	case soml.KindInline:
		return func(b *strings.Builder, deck *soml.Deck, slide *soml.Slide, elem *soml.Element, _ []int) error {
			return writeInline(b, deck, slide, elem)
		}
	// 표 세 종류는 각자 자기 태그를 낸다 (`<table>`/`<tr>`/`<th>`·`<td>`) — 웹에서는 프레임만
	// 그리는 것이 아니라 격자 전체가 마크업이 되므로 셋 다 writer를 갖는다 (PPTX는 프레임
	// 하나가 도형이 되고 행·셀이 그 안에 접힌다 — render/pptx/shape.go의 비대칭).
	//
	// 실제 호출은 부모가 한다 (writeTable → writeTableRow → writeTableCell): 자식이 행인지
	// 셀인지를 부모가 확인해야 콘텐츠 모델이 깨진 덱에서 `<table>` 안에 `<p>`가 조용히 섞이지
	// 않는다. 그래도 종류마다 writer를 등록하는 이유는 이 함수가 "이 종류를 웹으로 쓸 수
	// 있는가"의 유일한 대답이라는 것이다 — 목록과 어긋나면 카탈로그 대조가 거짓말을 한다.
	case soml.KindTable:
		return writeTable
	case soml.KindTableRow:
		return writeTableRow
	case soml.KindTableCell:
		return writeTableCell
	case soml.KindConnector:
		// 흐름에서는 아무것도 내지 않는다 — 연결선은 측정이 끝난 뒤 슬라이드 오버레이에 그려진다
		// (writeConnectors). 흐름에 자리를 차지하지 않는 것이 측정한 배치와 구운 배치가 같은 근거다.
		return func(*strings.Builder, *soml.Deck, *soml.Slide, *soml.Element, []int) error { return nil }
	}
	return nil
}

// writeNamedBlock은 요소 이름을 그대로 출력 태그로 쓰는 블록이다 (블록 종류와 `<li>`).
func writeNamedBlock(b *strings.Builder, deck *soml.Deck, slide *soml.Slide, elem *soml.Element, path []int) error {
	return writeBlock(b, deck, slide, elem, path, elem.Name)
}

// writeNode는 콘텐츠 노드 하나를 쓴다. path는 슬라이드 콘텐츠부터 이 노드까지의 자식
// 인덱스 경로다 (ElementID의 입력).
func writeNode(b *strings.Builder, deck *soml.Deck, slide *soml.Slide, node soml.Node, path []int) error {
	switch node := node.(type) {
	case *soml.Text:
		// 문자 데이터는 그대로 흐른다 — 공백 접기는 파서가 이미 했다 (soml.Parse).
		// pre 안의 원문 공백도 여기서 그대로 나간다 (이스케이프만 한다).
		b.WriteString(html.EscapeString(node.Value))
		return nil
	case *soml.Element:
		if node.Name == soml.TextElement {
			return writeBlock(b, deck, slide, node, path, "div")
		}
		// 태그 종류의 판정은 soml의 카탈로그 한 곳에서 나온다 (soml.KindOf) — 렌더러가
		// 이름 목록을 따로 들면 태그 하나를 추가할 때 한 곳을 빼먹는 것이 곧 "좌표가
		// 엉뚱한 요소에 붙는" 실패다.
		if write := writerFor(soml.KindOf(node.Name)); write != nil {
			return write(b, deck, slide, node, path)
		}
		// 닫힌 어휘에 태그를 추가하고 렌더 매핑을 잊으면 "파싱은 되는데 그려지지 않는"
		// 상태가 된다 — 닫힌 어휘로 막으려던 바로 그 조용한 실패다. 여기서 멈춘다.
		// (파서가 이미 같은 사실을 진단하므로 여기까지 오는 것은 파서를 거치지 않은 덱뿐이다 —
		// soml.KindRendered와 renderedKinds를 대조하는 테스트가 그 계약을 지킨다.)
		return fmt.Errorf("%s:%s <%s> has no web mapping", slide.Source, node.Span, node.Name)
	default:
		return fmt.Errorf("%s: unknown node type %T", slide.Source, node)
	}
}

// writeBlock은 텍스트 역할을 가진 블록 하나(`<text>`, 서브셋 블록, `<li>`)를 쓴다.
//
// 서브셋 블록의 출력 태그는 **저자가 쓴 이름 그대로**다 (`<h2>`→`<h2>`, `<p>`→`<p>` …).
// 전부 `<div>`로 내는 길과 저울질했고, 시맨틱 태그를 골랐다: 스크린 리더가 제목 위계와
// 리스트를 읽을 수 있고, 저자 CSS(@layer balpyo.author)가 태그 선택자로 겨눌 표적이 생긴다.
// 조건이 있다 — **브라우저 기본 스타일에 기대는 값이 하나도 없어야 한다**: 텍스트 메트릭
// (font/size/weight/line-height)은 core가 확정한 스타일을 인라인 style로 전부 명시하고,
// margin·padding은 엔진 CSS(.balpyo-text, .balpyo-list)가 0으로 명시하므로, UA 시트의
// `h1 { font-size: 2em; margin: … }` 같은 규칙이 이길 자리가 없다. 그래서 메트릭은 어느
// 태그를 골라도 같고, 시맨틱 태그는 접근성과 저자 CSS의 표적으로만 쓰인다. `<text>`는
// 역할이 속성으로 오는 중립 요소라 대응 시맨틱 태그가 없다 — `<div>`로 남는다 (역할별
// 태그로 바꾸면 `<text role>` ≡ 블록 태그 등가가 산출물에서 두 갈래가 된다).
//
// 클래스는 여전히 `balpyo-text` 하나다 — 역할·태그별 클래스를 두지 않는다. 역할 기본값은
// core가 이미 캐스케이드에 녹여 놓았으므로, 역할을 CSS로 되살리려면 core의 기본값 표를
// 렌더러에 한 번 더 적어야 한다. 그 이중화가 두 타깃의 외관이 갈라지는 경로다.
func writeBlock(b *strings.Builder, deck *soml.Deck, slide *soml.Slide, elem *soml.Element, path []int, tag string) error {
	style := elem.TextStyle()
	if style == nil {
		// 해석되지 않은 외관을 기본값으로 그리면 웹만 그럴듯하게 나오고 PPTX와 갈라진다.
		return fmt.Errorf("%s:%s <%s> appearance is unresolved — the deck did not go through core.Resolve",
			slide.Source, elem.Span, elem.Name)
	}
	text, err := textStyleCSS(*style, deck.Theme)
	if err != nil {
		return fmt.Errorf("%s:%s <%s>: %w", slide.Source, elem.Span, elem.Name, err)
	}
	decls := []string{text}
	if elem.Name == "pre" {
		// 접히지 않은 원문(soml.Parse의 pre 보존)이 그대로 흐른다. wrap을 켜면 PPTX와
		// 줄 수가 갈라지므로 줄바꿈 없음 — 긴 줄은 오버플로 허용이다 (박스 고정 정책과
		// 같은 태도. PPTX 쪽 짝은 bodyPr@wrap이다).
		decls = append(decls, "white-space: pre")
	}
	// 가로 크기는 외관 뒤, 슬롯·배치 앞이다 (appendTextBox — 속성 표의 순서와 같다).
	decls, err = appendTextBox(decls, elem.Box())
	if err != nil {
		return fmt.Errorf("%s:%s <%s>: %w", slide.Source, elem.Span, elem.Name, err)
	}
	// 배치 선언은 **한 곳에서** 나온다 (appendPlacement) — 블록도 컨테이너의 자식이 될 수
	// 있으므로 컨테이너·spacer·리스트와 같은 코드를 쓴다. 요소 종류마다 따로 쓰면 자식
	// 배치 속성이 하나 늘 때 어느 종류가 빠졌는지 사람이 세어야 한다.
	decls = appendSlot(decls, elem)
	decls, err = appendPlacement(decls, elem.Placement())
	if err != nil {
		return fmt.Errorf("%s:%s <%s>: %w", slide.Source, elem.Span, elem.Name, err)
	}
	fmt.Fprintf(b, "<%s class=\"balpyo-text\" %s=%q %s=%q%s%s%s>",
		tag, IDAttr, ElementID(slide.Index, path), ElAttr, elem.Name, revealAttr(elem), enterAttr(elem), styleAttr(decls))
	if elem.Name == "pre" {
		if first, ok := firstText(elem); ok && strings.HasPrefix(first, "\n") {
			// 파서가 양 끝 개행을 하나씩 벗겼으므로 남은 선두 개행은 내용이다. 그런데 HTML
			// 파서도 <pre> 여는 태그 직후의 개행 하나를 벗기므로, 개행 하나를 덧대어
			// 브라우저가 그것을 벗기게 한다 — 덧대지 않으면 내용의 첫 줄이 사라진다.
			b.WriteString("\n")
		}
	}
	for i, child := range elem.Children {
		if err := writeNode(b, deck, slide, child, ChildPath(path, i)); err != nil {
			return err
		}
	}
	fmt.Fprintf(b, "</%s>\n", tag)
	return nil
}

// revealAttr는 순차 등장 스텝 속성이다 (스텝이 없으면 속성 자체가 없다 — styleAttr와 같은
// 태도). 웹이 이 값을 나르는 **유일한** 타깃이다: PPTX·인쇄는 Appearance.Reveal을 보지
// 않는 것으로 평탄화를 구현한다.
func revealAttr(elem *soml.Element) string {
	if elem.Appearance == nil || elem.Appearance.Reveal == 0 {
		return ""
	}
	return fmt.Sprintf(" %s=%q", RevealAttr, strconv.Itoa(elem.Appearance.Reveal))
}

// EnterAttr · EnterDelayAttr는 등장의 마크업이다 — 효과 이름과 지연(ms)을 나르고, 뷰어 런타임이 지연을
// 전환에 걸어 슬라이드에 들어갈 때 켠다 (발표자 뷰에서만 — viewer.css).
const (
	EnterAttr      = "data-balpyo-enter"
	EnterDelayAttr = "data-balpyo-enter-delay"
)

// enterAttr는 등장 속성이다 (등장이 없으면 속성 자체가 없다 — revealAttr와 같은 태도).
func enterAttr(elem *soml.Element) string {
	e := elem.EnterOf()
	if e == nil {
		return ""
	}
	return fmt.Sprintf(" %s=%q %s=\"%d\"", EnterAttr, string(e.Effect), EnterDelayAttr, e.Delay)
}

// firstText는 요소의 첫 문자 데이터 자식이다.
func firstText(elem *soml.Element) (string, bool) {
	if len(elem.Children) == 0 {
		return "", false
	}
	if text, ok := elem.Children[0].(*soml.Text); ok {
		return text.Value, true
	}
	return "", false
}

// writeList는 리스트 컨테이너(`<ul>`/`<ol>`)를 쓴다.
//
// 컨테이너는 **외관**을 갖지 않는다 (core가 TextStyle을 채우지 않는다 — 항목을 담을 뿐이다).
// 반면 **배치**는 갖는다: `<row><ul basis="360">`처럼 리스트가 컨테이너의 자식이 되는 것이
// 흔한 배치이므로 자식 배치 속성이 열려 있고(soml.ChildPlacementAttrs), 그때 Appearance는
// Placement만 채워진 상태로 온다.
//
// 식별자는 언제나 갖는다: 리스트는 PPTX에서 박스 하나가 되므로 이 rect가 곧
// PPTX 텍스트 박스다. 마커·들여쓰기는 엔진 CSS가 상수로 명시한다 (contentCSS의 리스트 절).
func writeList(b *strings.Builder, deck *soml.Deck, slide *soml.Slide, elem *soml.Element, path []int) error {
	// 리스트 컨테이너도 가로 크기를 받는다 (리스트 외관이 컨테이너에 붙는 것과 같은 자리). 순서는 블록과 같다: 크기 → 슬롯 → 배치.
	decls, err := appendTextBox(nil, elem.Box())
	if err != nil {
		return fmt.Errorf("%s:%s <%s>: %w", slide.Source, elem.Span, elem.Name, err)
	}
	decls, err = appendPlacement(appendSlot(decls, elem), elem.Placement())
	if err != nil {
		return fmt.Errorf("%s:%s <%s>: %w", slide.Source, elem.Span, elem.Name, err)
	}
	fmt.Fprintf(b, "<%s class=\"balpyo-list\" %s=%q %s=%q%s%s>\n",
		elem.Name, IDAttr, ElementID(slide.Index, path), ElAttr, elem.Name, enterAttr(elem), styleAttr(decls))
	for i, child := range elem.Children {
		if err := writeNode(b, deck, slide, child, ChildPath(path, i)); err != nil {
			return err
		}
	}
	fmt.Fprintf(b, "</%s>\n", elem.Name)
	return nil
}

// writeInline은 인라인 요소(strong/em/code/a/span/br)를 텍스트 런으로 쓴다.
//
// 식별자(data-balpyo-id)는 붙이지 않는다 — 측정 대상은 블록 종류뿐이다 (패키지 주석의
// 측정 대상 계약): 여러 줄에 걸친 인라인의 rect는 뜻이 없고, PPTX에서 인라인은 박스가
// 아니라 런이다. 태그 앞뒤에 개행·들여쓰기도 내지 않는다 — 블록과 달리 인라인 주변의
// 공백은 텍스트 흐름의 내용이 된다 (인라인 경계 공백 계약, soml.Parse의 trimEdgeSpace).
func writeInline(b *strings.Builder, deck *soml.Deck, slide *soml.Slide, elem *soml.Element) error {
	if elem.Name == soml.BreakElement {
		// <br>은 void다 — 서식도 식별자도 없는 흐름의 분절점 (core가 Appearance를 채우지
		// 않는 유일한 인라인). HTML의 void 표기 그대로 낸다.
		b.WriteString("<br>")
		return nil
	}
	style := elem.TextStyle()
	if style == nil {
		return fmt.Errorf("%s:%s <%s> appearance is unresolved — the deck did not go through core.Resolve",
			slide.Source, elem.Span, elem.Name)
	}
	decls, err := runStyleCSS(*style, deck.Theme)
	if err != nil {
		return fmt.Errorf("%s:%s <%s>: %w", slide.Source, elem.Span, elem.Name, err)
	}
	if elem.Name == soml.MarkElement {
		// 형광펜은 `<mark>` 자신의 배경이다 — 안의 런은 같은 Highlight를 물려받지만(PPTX는 런이
		// 평평해서 런마다 a:highlight가 필요하다) 웹에서 자식마다 다시 칠하면 알파 있는 색이
		// 겹쳐 진해진다. UA 시트의 `mark { background: mark }`도 이 선언이 덮는다.
		if style.Highlight == nil {
			return fmt.Errorf("%s:%s <mark> highlight is unresolved — the deck did not go through core.Resolve",
				slide.Source, elem.Span)
		}
		bg, err := colorValue(*style.Highlight, deck.Theme)
		if err != nil {
			return fmt.Errorf("%s:%s <mark>: %w", slide.Source, elem.Span, err)
		}
		decls += "; background-color: " + bg
	}
	if elem.Name == "a" {
		link := elem.Appearance.Link
		if link == "" {
			// core의 href 검증(없음/빈 값 에러)을 지나지 않은 값이 산출물에 실릴 길을 막는다.
			return fmt.Errorf("%s:%s <a> link is unresolved — the deck did not go through core.Resolve",
				slide.Source, elem.Span)
		}
		// href는 원문 그대로 싣되 HTML 속성으로 이스케이프한다 — &가 그대로 나가면
		// 엔티티로 읽혀 목적지가 조용히 바뀐다.
		fmt.Fprintf(b, "<a href=%q style=%q>", html.EscapeString(link), decls)
	} else {
		fmt.Fprintf(b, "<%s style=%q>", elem.Name, decls)
	}
	for _, child := range elem.Children {
		// 인라인의 자식은 문자 데이터와 인라인뿐이다 (콘텐츠 모델). 경로는 넘기지 않는다 —
		// 인라인 아래에는 식별자를 갖는 요소가 없다.
		if err := writeInlineChild(b, deck, slide, child); err != nil {
			return err
		}
	}
	fmt.Fprintf(b, "</%s>", elem.Name)
	return nil
}

// writeInlineChild는 인라인 요소의 자식 하나를 쓴다.
func writeInlineChild(b *strings.Builder, deck *soml.Deck, slide *soml.Slide, node soml.Node) error {
	switch node := node.(type) {
	case *soml.Text:
		b.WriteString(html.EscapeString(node.Value))
		return nil
	case *soml.Element:
		if soml.KindOf(node.Name) == soml.KindInline {
			return writeInline(b, deck, slide, node)
		}
		return fmt.Errorf("%s:%s <%s> inside inline content has no web mapping",
			slide.Source, node.Span, node.Name)
	default:
		return fmt.Errorf("%s: unknown node type %T", slide.Source, node)
	}
}

// ElementID는 요소 식별자를 만든다 — `s<슬라이드 인덱스>[.<자식 인덱스>]…`.
//
// 인덱스는 **덱 모델의 슬라이스 인덱스**다 (Slide.Content와 Element.Children). 그래서 이
// 값은 렌더 없이 덱 모델만 보고도 계산할 수 있고, 지오메트리 패스가 자기 쪽에서 같은 문자열을
// 만들어 rect를 찾을 수 있다. 저자가 준 `id`를 쓰지 않는 이유는 두 가지다: `<slide id>`는
// 슬라이드에만 있고(요소는 못 가진다), 딥링크용 이름이라 저자가 언제든 바꾼다.
//
// **구현이 하나여야 한다.** 지오메트리 패스가 같은 규칙을 따로 구현하면 규칙이 갈라지는 날
// rect가 엉뚱한 요소에 붙고, 그 실패는 산출물을 열어 보기 전까지 조용하다 — 그래서
// render/geom이 이 함수를 부른다.
func ElementID(slideIndex int, path []int) string {
	id := fmt.Sprintf("s%d", slideIndex)
	for _, i := range path {
		id = ChildID(id, i)
	}
	return id
}

// ChildID는 부모 식별자에 자식 인덱스를 더한 자식 식별자다 — `ElementID(slide, ChildPath(path, i))`와
// **같은 값**이다 (ElementID가 이 함수로 조립되므로 갈라질 수 없다).
//
// 경로 없이 식별자만 든 소비자를 위해 내보낸다: PPTX의 표 렌더러는 도형 writer 서명으로
// (요소, id)만 받는데(shapeWriter) 행·셀의 잰 박스를 찾으려면 자식 식별자가 필요하다. 그 자리에서
// 문자열을 손으로 이어 붙이면 ElementID와 별개 구현이 하나 더 생기고, 규칙이 갈라지는 날 좌표가
// 엉뚱한 셀에 붙는다 (ElementID 주석의 "구현이 하나여야 한다").
func ChildID(id string, i int) string {
	return fmt.Sprintf("%s.%d", id, i)
}

// ChildPath는 부모 경로에 자식 인덱스를 더한 **새** 경로다. append를 그대로 쓰면 형제끼리
// 배열을 공유해 한쪽의 인덱스가 다른 쪽에 실릴 수 있다 — 그런 식별자 충돌은 좌표가 엉뚱한
// 요소에 붙는 것으로만 드러난다.
//
// ElementID와 같은 이유로 내보낸다: 경로를 만드는 순회는 셋이다 (웹 렌더, 측정 대조
// geom.Refs, 도형 조립 render/pptx.shapesXML). 규칙이 층마다 복제되면 한 곳만 바뀐 순간
// 세 목록의 자리가 어긋나고, 그 실패는 좌표가 엉뚱한 요소에 붙은 산출물로만 드러난다.
func ChildPath(path []int, i int) []int {
	child := make([]int, len(path)+1)
	copy(child, path)
	child[len(path)] = i
	return child
}

// contentCSS는 콘텐츠 배치의 기본 CSS다 — writeStage가 정의한 스택 규칙의 CSS 절반이다.
// 덱이 실제로 쓴 어휘의 절(listCSS·layoutCSS)만 함께 낸다.
//
// `.balpyo-text`의 margin 0은 시맨틱 블록 태그(h1~h3, p, blockquote, pre)의 UA 기본
// margin도 지운다 — writeBlock의 "브라우저 기본 스타일에 기대는 값이 없어야 한다" 조건의
// 배치 절반이다 (메트릭 절반은 인라인 style이 명시한다).
//
// 여백(64px)과 간격(24px)은 엔진 상수다 (테마의 레이아웃 CSS가 레이아웃마다 다시 정할 수
// 있다). 스테이지 여백을 space 토큰(`slidePadding`/`gap`)의 변수 참조로 바꾸면 레이아웃 태그를
// 쓰지 않은 덱의 골든이 무관한 이유로 움직인다 (core/themes/minimal/tokens.json의 공간 토큰
// 주석에 같은 판단이 있다).
func contentCSS(feat features) string {
	css := fmt.Sprintf(`
  /* The stage stacks its children vertically in document order. The 64px margin is 5%% of the
     1280 width, so text is not cut off by projector overscan. */
  .balpyo-stage {
    position: relative;
    width: 100%%;
    height: 100%%;
    padding: %dpx;
    display: flex;
    flex-direction: column;
    gap: %dpx;
  }

  /* Korean breaks at word boundaries — the browser default of breaking between characters is
     not Korean typesetting practice. This is not about matching PPTX (line-break positions are
     outside the contract). overflow-wrap: anywhere breaks a word only when a single word is
     wider than a narrow cell — with keep-all alone that word overflows the cell. Chinese and
     Japanese have no spaces between words, so keep-all would block wrapping; hence this applies
     only when lang is ko. */
  .balpyo-stage:lang(ko) {
    word-break: keep-all;
    overflow-wrap: anywhere;
  }

  /* A text block takes the full content width — the wrap width must be fixed so the box the
     browser measured carries over to the PPTX text box as is. flex: 0 0 auto keeps the box from
     being squeezed when the stack is taller than the stage (the default policy is fixed boxes +
     allowed overflow). */
  .balpyo-text {
    margin: 0;
    width: 100%%;
    flex: 0 0 auto;
  }
`, StagePadding, stackGap)
	if feat.lists {
		css += listCSS()
	}
	if feat.layout {
		css += layoutCSS()
	}
	if feat.stack || feat.canvas {
		// stack 규칙은 layoutCSS와 별도 절이다 — stack이나 canvas를 실제로 쓴 덱에만 나가
		// 골든 불변을 지킨다 (stackCSS의 근거). 캔버스 스테이지가 stack과 같은 겹침 CSS를
		// 공유하므로 두 조건이 같은 절을 켠다.
		css += stackCSS()
	}
	if feat.shapes {
		css += shapeCSS()
	}
	if feat.images {
		css += imageCSS()
	}
	if feat.connectors {
		css += connectorCSS()
	}
	if feat.tables {
		css += tableCSS()
	}
	return css
}

// listCSS는 리스트 배치 CSS다 — 값은 전부 엔진 상수(ListTextIndent 등)에서 나온다.
//
// 리스트가 있는 덱에만 낸다: 리스트가 없는 덱의 산출물이 리스트 렌더가 붙었다는 이유로
// 달라지면 골든 diff가 무엇이 움직였는지 말해 주지 못한다 (빈 스테이지와 같은 근거).
func listCSS() string {
	return fmt.Sprintf(`
  /* Lists — the engine draws the markers (list-style: none + ::before), not the UA ::marker.
     The UA marker is right-aligned to the left of the text, while PPTX left-aligns it at
     marL+indent and that is how the viewers draw it, so it cannot change — the web follows the
     PPTX convention (see ListTextIndent). */
  .balpyo-list {
    margin: 0;
    padding: 0;
    list-style: none;
    width: 100%%;
    flex: 0 0 auto;
  }

  /* display: block removes the UA list-item marker box entirely — a second safeguard on top of
     list-style: none. The margin between items is ListItemGap (0) — line height owns the
     vertical rhythm. The hanging indent is padding-left + a negative text-indent (only the
     first line moves out into the marker slot) — the same model as PPTX marL + a negative
     indent. */
  .balpyo-list > li {
    display: block;
    margin: 0 0 %dpx;
    padding-left: %dpx;
    text-indent: -%dpx;
  }

  /* The marker is **the first box in the text flow** — an inline-block ListMarkerHang wide, so
     the text starts exactly at the padding edge. It is not taken out of the flow (absolutely
     positioned) because of alignment: OOXML aligns the marker with its paragraph, so with
     algn="ctr" the marker is centered along with the text, whereas an absolutely positioned
     marker would stay in the indent and the two targets would diverge. text-indent is reset
     because an inline-block is a block container that would inherit the li's negative value and
     apply it again to its own first line. The resolved style (font, color, line height) is
     still inherited, so the marker and the first line share a baseline — PPTX markers follow
     the run formatting too. */
  .balpyo-list > li::before {
    display: inline-block;
    width: %dpx;
    text-indent: 0;
  }

  ul.balpyo-list > li::before { content: %q; }

  /* ol uses an engine counter — the UA list-item counter only advances if display is restored,
     so it is not relied on. The "1." format pairs with PPTX buAutoNum@type="arabicPeriod". */
  ol.balpyo-list { counter-reset: balpyo-item; }
  ol.balpyo-list > li {
    counter-increment: balpyo-item;
  }
  ol.balpyo-list > li::before { content: counter(balpyo-item) "."; }
`, ListItemGap, ListTextIndent, ListMarkerHang, ListMarkerHang, ListBulletChar)
}

// features는 덱이 실제로 쓴 어휘다 — 어느 CSS 절을 낼지 정하는 근거다.
//
// 조건부로 두는 이유는 골든이다: 쓰지 않은 어휘의 CSS가 산출물에 들어가면, 그 어휘를
// 구현한 커밋에서 **모든** 골든이 움직이고 diff가 무엇이 변했는지 말해 주지 못한다
// (빈 스테이지를 빈 요소 하나로 낸 것과 같은 근거). 걸음을 한 번만 걷는 이유는 순회가
// 두 벌이 되면 한쪽만 새 태그를 세게 되기 때문이다.
type features struct {
	// lists는 리스트 컨테이너가 있는지다 (listCSS).
	lists bool
	// layout은 레이아웃 컨테이너나 `<spacer>`가 있는지다 (layoutCSS + 공간 토큰 변수).
	//
	// 공간 토큰 변수를 이 조건에 매단 이유: 변수를 **참조하는** 자리는 컨테이너의
	// `gap`/`pad`뿐이다 (soml.Length의 토큰 상태를 만드는 속성이 그 둘이다). 레이아웃
	// 태그가 없는 덱에 선언만 내면 참조 없는 변수가 골든에 남는다.
	layout bool
	// stack은 `<stack>`이 있는지다 (stackCSS). layout과 따로 두는 이유는 골든 불변이다:
	// stack CSS를 layoutCSS에 넣으면 stack을 안 쓴 레이아웃 덱(03_layout 등)의 골든이
	// 움직인다. `<stack>`은 레이아웃 컨테이너이기도 하므로 이 필드가 참이면 layout도 참이다
	// (그래서 stack 덱의 pad 토큰은 공간 변수를 얻는다).
	stack bool
	// canvas는 layout="canvas" 슬라이드가 있는지다 (캔버스 스테이지 CSS). 캔버스 스테이지는
	// stack과 같은 겹침 절(stackCSS)을 공유하므로 stack과 함께 그 절을 켠다.
	canvas bool
	// shapes는 `<shape>`가 있는지다 (shapeCSS + 그림자 토큰 변수).
	//
	// 그림자 변수를 이 조건에 매단 이유는 공간 토큰과 같다 — 참조하는 자리가 도형의
	// `shadow` 하나뿐이다. 도형이 있는 덱에서 그림자를 쓰지 않으면 선언만 남지만, 조건을
	// 토큰 참조 단위로 더 쪼개면 "무엇이 있으면 무엇이 나온다"의 규칙이 어휘 단위에서
	// 속성 단위로 내려가 골든을 읽는 사람이 규칙을 셋 이상 외워야 한다.
	shapes bool
	// images는 `<image>`가 있는지다 (imageCSS + 그림자 토큰 변수).
	//
	// 도형과 같은 조건에 그림자 변수를 매단 이유는 참조하는 자리가 그림의 `shadow`이기도
	// 하기 때문이다 — 도형 없이 그림만 있는 덱에서 선언을 내지 않으면
	// `filter: drop-shadow(var(--balpyo-shadow-soft))`가 정의되지 않은 변수를 가리켜
	// **그림자가 조용히 사라진다** (var()는 모르는 변수에 아무 소리도 내지 않는다).
	images bool
	// chrome은 확정된 덱 크롬이 실린 슬라이드가 있는지다 (chromeCSS). 콘텐츠 어휘가 아니라
	// 덱 수준의 사실이므로 순회가 아니라 슬라이드 루프에서 켠다 — canvas와 같은 자리다.
	chrome bool
	// tables는 `<table>`가 있는지다 (tableCSS). 표 CSS 절은 셀 여백·테두리 합침처럼 셀 하나로
	// 표현할 수 없는 값만 담는다 — 표를 안 쓴 덱의 골든이 이 절로 움직이지 않는다.
	tables bool
	// connectors는 `<connector>`가 있는지다 (connectorCSS) — 연결선을 안 쓴 덱의 산출물이 움직이지 않는다.
	connectors bool
	// entrances는 등장(`enter`)이 실린 요소가 있는지다 — 테마의 박자 변수(--balpyo-motion-duration)를 낸다.
	entrances bool
	// layouts는 이 덱이 실제로 쓴, **배치 CSS를 가진** 레이아웃 이름들이다 (정렬·중복 제거).
	// themeCSS가 이 목록만 규칙으로 낸다 — 다른 어휘 조건과 같은 근거다 (골든 불변): 쓰지 않은
	// 레이아웃의 배치 규칙이 산출물에 들어가면 diff가 무엇이 변했는지 말해 주지 못한다.
	layouts []string
	// slotLayouts는 이 덱이 **슬롯 모드로** 쓴, 슬롯 영역 CSS를 가진 레이아웃 이름들이다
	// (정렬·중복 제거). layouts와 따로 두는 이유는 골든 불변이자 두 모드의 분리다:
	// 슬롯 영역 규칙은 `.balpyo-slots` 모디파이어가 붙은 선택자로 나가고 흐름 규칙은
	// `:not(.balpyo-slots)`로 나가, 한 레이아웃을 두 슬라이드가 각각 흐름·슬롯으로 써도
	// 규칙이 서로 새지 않는다 (layoutStagesCSS).
	slotLayouts []string
}

// spaceTokens는 덱이 공간 토큰을 **참조하는지**다 (themeCSS의 선언 조건).
//
// 조건이 어휘 셋인 이유: 공간 토큰을 값으로 받는 속성은 컨테이너의 `gap`/`pad`와 도형의
// `pad`, 그리고 **그림의 `pad`**다 (soml.Box를 세 요소가 공유하므로 `pad`도 공유된다).
// 그림만 있는 덱에서 선언을 내지 않으면 `padding: var(--balpyo-space-md)`가 정의되지 않은
// 변수를 가리켜 여백이 조용히 0이 된다.
func (f features) spaceTokens() bool { return f.layout || f.shapes || f.images }

// shadowTokens는 덱이 그림자 토큰을 참조하는지다 — 도형과 그림이 그 자리다.
func (f features) shadowTokens() bool { return f.shapes || f.images }

// contentFeatures는 덱을 걸어 쓰인 어휘를 모은다. 태그 종류의 판정은 soml의 카탈로그
// 한 곳에서 나온다 (soml.KindOf) — 렌더러가 이름 목록을 따로 들면 태그를 추가할 때
// CSS 절이 빠진 채로 요소가 나가고, 그 실패는 산출물을 열어 봐야 드러난다.
func contentFeatures(deck *soml.Deck) features {
	var feat features
	var walk func(nodes []soml.Node)
	walk = func(nodes []soml.Node) {
		for _, node := range nodes {
			elem, ok := node.(*soml.Element)
			if !ok {
				continue
			}
			switch soml.KindOf(elem.Name) {
			case soml.KindListContainer:
				feat.lists = true
			case soml.KindLayoutContainer, soml.KindSpacer:
				feat.layout = true
				// `<stack>`은 레이아웃 컨테이너이지만 겹침 CSS는 별도 절이다 — 흐름으로
				// 가른다 (soml.FlowOf). 종류가 아니라 흐름으로 갈린다는 계약(soml.FlowStack)의
				// 렌더러 쪽 절반이 이 판정이다.
				if soml.FlowOf(elem.Name) == soml.FlowStack {
					feat.stack = true
				}
			case soml.KindShape:
				feat.shapes = true
			case soml.KindImage:
				feat.images = true
			case soml.KindTable:
				feat.tables = true
			case soml.KindConnector:
				feat.connectors = true
			}
			if elem.EnterOf() != nil {
				feat.entrances = true
			}
			walk(elem.Children)
		}
	}
	seenFlow := map[string]bool{}
	seenSlot := map[string]bool{}
	for _, slide := range deck.Slides {
		if slide.Layout == soml.LayoutCanvas {
			// 캔버스 스테이지는 stack이다 — 스테이지 CSS(stackCSS)가 이때만
			// 나간다. 슬라이드 수준 어휘라 콘텐츠 순회가 아니라 여기서 켠다.
			feat.canvas = true
		}
		if len(slide.Chrome) > 0 {
			// 크롬은 값 층이 이미 걸러 확정한 목록이다 — 설정이 켰는지도, 테마가
			// 선언했는지도 여기서 보지 않는다.
			feat.chrome = true
		}
		// 레이아웃 규칙은 이 슬라이드가 **슬롯 모드인가 흐름 모드인가**로 갈린다 (writeStage와
		// 같은 판정): 슬롯 모드면 슬롯 영역 CSS를, 아니면 흐름 배치 CSS를 모은다. 한 슬라이드는
		// 둘 중 하나뿐이므로 순수 흐름 덱은 슬롯 규칙을, 순수 슬롯 덱은 흐름 규칙을 내지 않아
		// 양쪽 골든이 모두 이 기능 때문에 움직이지 않는다. 판정은 writeStage와 **같은 한 줄**이다
		// (슬롯 탑재 여부만 본다 — 테마 조건은 core가 이미 걸렀다): 여기서
		// 조건이 갈리면 스테이지에 모디파이어는 붙었는데 그 규칙이 CSS에 없는(또는 반대의)
		// 산출물이 나온다.
		name := string(slide.Layout)
		if slideUsesSlots(slide) {
			if !seenSlot[name] {
				seenSlot[name] = true
				feat.slotLayouts = append(feat.slotLayouts, name)
			}
		} else if _, ok := deck.Theme.Layouts[name]; ok && !seenFlow[name] {
			// 배치 CSS를 가진 레이아웃만 모은다 — 없는 레이아웃(content·canvas)은 맵에 없다.
			seenFlow[name] = true
			feat.layouts = append(feat.layouts, name)
		}
		walk(slide.Content)
	}
	// 순서를 고정한다 — 규칙 출력 순서가 슬라이드 순서에 따라 흔들리면 골든이 무엇이 변했는지
	// 말해 주지 못한다 (테마 토큰 표를 순서 있게 둔 것과 같은 근거).
	sort.Strings(feat.layouts)
	sort.Strings(feat.slotLayouts)
	return feat
}
