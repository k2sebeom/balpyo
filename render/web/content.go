package web

import (
	"fmt"
	"html"
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
)

// SlideSelector는 문서 순서의 모든 슬라이드를 집는 선택자다.
func SlideSelector() string { return "." + SlideClass }

// SlideIndexSelector는 인덱스로 슬라이드 하나를 집는 선택자다.
func SlideIndexSelector(index int) string {
	return fmt.Sprintf("%s[%s=%q]", SlideSelector(), SlideIndexAttr, strconv.Itoa(index))
}

// ElementSelector는 식별자를 가진 모든 요소를 집는 선택자다 — 지오메트리 패스가 재는 대상.
func ElementSelector() string { return "[" + IDAttr + "]" }

// 스테이지 여백과 스택 간격이다 (논리 px). 간격 24px은 본문 크기와 같아 한 글자 높이만큼
// 벌어진 리듬이 된다 — 역할 기본값의 수직 리듬(core/theme.go)과 같은 근거다.
const (
	stagePadding = 64
	stackGap     = 24
)

// 리스트 배치 상수다 (논리 px). export하는 이유: PPTX 렌더러의 `marL`/`indent`(항목 07)가
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
// 342900 EMU ÷ 9525(EMU/px) = 36px (docs/decisions/0002의 환산 상수). 익숙한 기본값을
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
// `counter() "."`로 내며 PPTX `buAutoNum@type="arabicPeriod"`("1.")와 짝이다 (항목 07).
const ListBulletChar = "\u2022" // •

// writeStage는 슬라이드의 콘텐츠 노드를 스테이지 div에 쓴다.
//
// 자식 배치는 **문서 순서의 세로 스택**이다. 레이아웃 태그(`<row>`/`<col>`/`<grid>`)는
// Iteration 3이므로 이번 iteration의 기본 배치는 이 함수가 정의한다. 세로 스택을 고른 이유:
//
//   - 저자가 쓴 순서와 화면 순서가 같아야 SOML만 읽고 결과를 예측할 수 있다.
//   - 자식마다 **확정된 폭**(스테이지 콘텐츠 폭)을 주면 브라우저의 줄바꿈이 결정적이다.
//     지오메트리 패스가 잰 rect가 그대로 PPTX 텍스트 박스가 되고 autofit은 꺼지므로
//     (PRD §9.3), 폭이 내용에 따라 변하는 배치는 PowerPoint에서 줄바꿈이 달라질 여지를
//     만든다. 폭을 고정하면 두 타깃이 같은 폭 안에서 줄을 나눈다.
//   - 세로 정렬(가운데·아래)은 레이아웃의 선택이지 텍스트의 성질이 아니다 — 표지의 가운데
//     정렬은 Iteration 10의 `layout="title"`이 소유한다. 그때까지 위에서 아래로 쌓는다.
func writeStage(b *strings.Builder, deck *soml.Deck, slide *soml.Slide) error {
	if len(slide.Content) == 0 {
		// 빈 스테이지는 빈 요소 하나다 — 콘텐츠가 없는 슬라이드의 산출물이 콘텐츠 렌더가
		// 붙었다는 이유로 달라지면 골든 diff가 무엇이 움직였는지 말해 주지 못한다.
		b.WriteString(`<div class="balpyo-stage"></div>` + "\n")
		return nil
	}
	b.WriteString(`<div class="balpyo-stage">` + "\n")
	for i, node := range slide.Content {
		if err := writeNode(b, deck, slide, node, []int{i}); err != nil {
			return err
		}
	}
	b.WriteString("</div>\n")
	return nil
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
		switch soml.KindOf(node.Name) {
		case soml.KindBlock, soml.KindListItem:
			// 블록과 li는 저자가 쓴 이름 그대로 시맨틱 태그로 낸다 (근거는 writeBlock).
			return writeBlock(b, deck, slide, node, path, node.Name)
		case soml.KindListContainer:
			return writeList(b, deck, slide, node, path)
		case soml.KindInline:
			return writeInline(b, deck, slide, node)
		}
		// 닫힌 어휘에 태그를 추가하고 렌더 매핑을 잊으면 "파싱은 되는데 그려지지 않는"
		// 상태가 된다 — 닫힌 어휘로 막으려던 바로 그 조용한 실패다. 여기서 멈춘다.
		return fmt.Errorf("%s:%s <%s>를 웹으로 렌더하는 매핑이 없다", slide.Source, node.Span, node.Name)
	default:
		return fmt.Errorf("%s: 알 수 없는 노드 종류 %T", slide.Source, node)
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
		return fmt.Errorf("%s:%s <%s>의 외관이 해석되지 않았다 — core.Resolve를 거치지 않은 덱이다",
			slide.Source, elem.Span, elem.Name)
	}
	decls, err := textStyleCSS(*style, deck.Theme)
	if err != nil {
		return fmt.Errorf("%s:%s <%s>: %w", slide.Source, elem.Span, elem.Name, err)
	}
	if elem.Name == "pre" {
		// 접히지 않은 원문(soml.Parse의 pre 보존)이 그대로 흐른다. wrap을 켜면 PPTX와
		// 줄 수가 갈라지므로 줄바꿈 없음 — 긴 줄은 오버플로 허용이다 (박스 고정 정책,
		// PRD §13-1과 같은 태도. PPTX 쪽 짝 bodyPr@wrap은 항목 06이 맞춘다).
		decls += "; white-space: pre"
	}
	fmt.Fprintf(b, "<%s class=\"balpyo-text\" %s=%q %s=%q style=%q>",
		tag, IDAttr, ElementID(slide.Index, path), ElAttr, elem.Name, decls)
	if elem.Name == "pre" {
		if first, ok := firstText(elem); ok && strings.HasPrefix(first, "\n") {
			// 파서가 양 끝 개행을 하나씩 벗겼으므로 남은 선두 개행은 내용이다. 그런데 HTML
			// 파서도 <pre> 여는 태그 직후의 개행 하나를 벗기므로, 개행 하나를 덧대어
			// 브라우저가 그것을 벗기게 한다 — 덧대지 않으면 내용의 첫 줄이 사라진다.
			b.WriteString("\n")
		}
	}
	for i, child := range elem.Children {
		if err := writeNode(b, deck, slide, child, childPath(path, i)); err != nil {
			return err
		}
	}
	fmt.Fprintf(b, "</%s>\n", tag)
	return nil
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
// 컨테이너는 외관을 갖지 않는다 (core가 Appearance를 채우지 않는다 — 항목을 담을 뿐이다).
// 그래도 식별자는 갖는다: 리스트는 PPTX에서 박스 하나가 되므로(항목 07) 이 rect가 곧
// PPTX 텍스트 박스다. 마커·들여쓰기는 엔진 CSS가 상수로 명시한다 (contentCSS의 리스트 절).
func writeList(b *strings.Builder, deck *soml.Deck, slide *soml.Slide, elem *soml.Element, path []int) error {
	fmt.Fprintf(b, "<%s class=\"balpyo-list\" %s=%q %s=%q>\n",
		elem.Name, IDAttr, ElementID(slide.Index, path), ElAttr, elem.Name)
	for i, child := range elem.Children {
		if err := writeNode(b, deck, slide, child, childPath(path, i)); err != nil {
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
		return fmt.Errorf("%s:%s <%s>의 외관이 해석되지 않았다 — core.Resolve를 거치지 않은 덱이다",
			slide.Source, elem.Span, elem.Name)
	}
	decls, err := runStyleCSS(*style, deck.Theme)
	if err != nil {
		return fmt.Errorf("%s:%s <%s>: %w", slide.Source, elem.Span, elem.Name, err)
	}
	if elem.Name == "a" {
		link := elem.Appearance.Link
		if link == "" {
			// core의 href 검증(없음/빈 값 에러)을 지나지 않은 값이 산출물에 실릴 길을 막는다.
			return fmt.Errorf("%s:%s <a>의 링크가 해석되지 않았다 — core.Resolve를 거치지 않은 덱이다",
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
		return fmt.Errorf("%s:%s 인라인 안의 <%s>를 웹으로 렌더하는 매핑이 없다",
			slide.Source, node.Span, node.Name)
	default:
		return fmt.Errorf("%s: 알 수 없는 노드 종류 %T", slide.Source, node)
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
	var b strings.Builder
	fmt.Fprintf(&b, "s%d", slideIndex)
	for _, i := range path {
		fmt.Fprintf(&b, ".%d", i)
	}
	return b.String()
}

// childPath는 부모 경로에 자식 인덱스를 더한 **새** 경로다. append를 그대로 쓰면 형제끼리
// 배열을 공유해 한쪽의 인덱스가 다른 쪽에 실릴 수 있다 — 그런 식별자 충돌은 좌표가 엉뚱한
// 요소에 붙는 것으로만 드러난다.
func childPath(path []int, i int) []int {
	child := make([]int, len(path)+1)
	copy(child, path)
	child[len(path)] = i
	return child
}

// contentCSS는 콘텐츠 배치의 기본 CSS다 — writeStage가 정의한 스택 규칙의 CSS 절반이다.
// lists가 참이면 리스트 절(listCSS)을 함께 낸다.
//
// `.balpyo-text`의 margin 0은 시맨틱 블록 태그(h1~h3, p, blockquote, pre)의 UA 기본
// margin도 지운다 — writeBlock의 "브라우저 기본 스타일에 기대는 값이 없어야 한다" 조건의
// 배치 절반이다 (메트릭 절반은 인라인 style이 명시한다).
//
// 여백(64px)과 간격(24px)은 지금 엔진 상수다. 테마의 space 토큰(`slidePadding`/`gap`,
// PRD §8.4)이 Iteration 10에서 이 값을 대체한다 — 지금 core의 토큰 표에 space를 넣으면
// 두 렌더러가 실제로 무엇을 요구하는지 모르는 상태로 어휘를 고정하게 된다 (core/theme.go의
// 같은 판단).
func contentCSS(lists bool) string {
	css := fmt.Sprintf(`
  /* 스테이지는 자식을 문서 순서대로 세로로 쌓는다. 여백 64px은 1280폭의 5%%로,
     프로젝터의 오버스캔에서도 글자가 잘리지 않는다. */
  .balpyo-stage {
    position: relative;
    width: 100%%;
    height: 100%%;
    padding: %dpx;
    display: flex;
    flex-direction: column;
    gap: %dpx;
  }

  /* 텍스트 블록은 콘텐츠 폭 전체를 갖는다 — 줄바꿈 폭이 확정되어야 브라우저가 잰 박스가
     PPTX 텍스트 박스로 그대로 간다. flex: 0 0 auto는 스택이 스테이지보다 길 때 박스가
     눌리는 것을 막는다 (기본 정책은 박스 고정 + 오버플로 허용 — PRD §13-1). */
  .balpyo-text {
    margin: 0;
    width: 100%%;
    flex: 0 0 auto;
  }
`, stagePadding, stackGap)
	if lists {
		css += listCSS()
	}
	return css
}

// listCSS는 리스트 배치 CSS다 — 값은 전부 엔진 상수(ListTextIndent 등)에서 나온다.
//
// 리스트가 있는 덱에만 낸다: 리스트가 없는 덱의 산출물이 리스트 렌더가 붙었다는 이유로
// 달라지면 골든 diff가 무엇이 움직였는지 말해 주지 못한다 (빈 스테이지와 같은 근거).
func listCSS() string {
	return fmt.Sprintf(`
  /* 리스트 — 마커는 UA의 ::marker가 아니라 엔진이 그린다 (list-style: none + ::before).
     UA 마커는 텍스트 왼쪽에 오른쪽 정렬이지만 PPTX는 marL+indent 위치에 왼쪽 정렬이고
     뷰어가 그리는 방식이라 바꿀 수 없다 — 웹이 PPTX의 관례를 따른다 (ListTextIndent 주석). */
  .balpyo-list {
    margin: 0;
    padding: 0;
    list-style: none;
    width: 100%%;
    flex: 0 0 auto;
  }

  /* display: block은 UA의 list-item 마커 박스를 아예 없앤다 — list-style: none 위에
     이중 안전판이다. 항목 간 margin은 ListItemGap(0) — 수직 리듬은 줄높이가 소유한다. */
  .balpyo-list > li {
    display: block;
    position: relative;
    margin: 0 0 %dpx;
    padding-left: %dpx;
  }

  /* 마커는 첫 줄의 정적 위치에 절대 배치된다 — li의 확정 스타일(폰트·색·줄높이)을
     상속하므로 첫 줄과 기준선이 맞는다. PPTX의 마커도 런 서식을 따른다 (항목 07). */
  .balpyo-list > li::before {
    position: absolute;
    left: %dpx;
  }

  ul.balpyo-list > li::before { content: %q; }

  /* ol은 엔진 카운터를 쓴다 — UA의 list-item 카운터는 display를 되돌려야 움직이므로
     의존하지 않는다. "1." 형식은 PPTX buAutoNum@type="arabicPeriod"와 짝이다. */
  ol.balpyo-list { counter-reset: balpyo-item; }
  ol.balpyo-list > li {
    counter-increment: balpyo-item;
  }
  ol.balpyo-list > li::before { content: counter(balpyo-item) "."; }
`, ListItemGap, ListTextIndent, ListTextIndent-ListMarkerHang, ListBulletChar)
}

// hasLists는 덱에 리스트 컨테이너가 있는지 본다 (listCSS를 낼지 결정하는 근거).
// 지금의 콘텐츠 모델에서 리스트는 슬라이드 직속이지만, 레이아웃 태그(Iteration 3)가
// 컨테이너를 들이면 깊어지므로 처음부터 재귀로 본다.
func hasLists(deck *soml.Deck) bool {
	var walk func(nodes []soml.Node) bool
	walk = func(nodes []soml.Node) bool {
		for _, node := range nodes {
			elem, ok := node.(*soml.Element)
			if !ok {
				continue
			}
			if soml.KindOf(elem.Name) == soml.KindListContainer {
				return true
			}
			if walk(elem.Children) {
				return true
			}
		}
		return false
	}
	for _, slide := range deck.Slides {
		if walk(slide.Content) {
			return true
		}
	}
	return false
}
