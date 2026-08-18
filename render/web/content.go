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
		b.WriteString(html.EscapeString(node.Value))
		return nil
	case *soml.Element:
		if node.Name != soml.TextElement {
			// 닫힌 어휘에 태그를 추가하고 렌더 매핑을 잊으면 "파싱은 되는데 그려지지 않는"
			// 상태가 된다 — 닫힌 어휘로 막으려던 바로 그 조용한 실패다. 여기서 멈춘다.
			return fmt.Errorf("%s:%s <%s>를 웹으로 렌더하는 매핑이 없다", slide.Source, node.Span, node.Name)
		}
		return writeText(b, deck, slide, node, path)
	default:
		return fmt.Errorf("%s: 알 수 없는 노드 종류 %T", slide.Source, node)
	}
}

// writeText는 `<text>` 하나를 블록 요소로 쓴다.
//
// 태그를 `<div>`로 고른 이유: 역할별 시맨틱 태그(`<h1>`, `<p>`)는 브라우저 기본 스타일
// (폰트 크기·굵기·margin)을 끌고 들어오고, 그것을 전부 되돌리는 CSS를 쓰지 않으면 텍스트
// 메트릭이 브라우저 기본값에 의존하게 된다. HTML 서브셋 태그(Iteration 2)가 들어올 때
// 시맨틱 매핑을 함께 정한다.
//
// 클래스도 `balpyo-text` 하나다 — 역할별 클래스를 두지 않는다. 역할 기본값은 core가 이미
// 캐스케이드에 녹여 놓았으므로(soml.Appearance), 역할을 CSS로 되살리려면 core의 기본값 표를
// 렌더러에 한 번 더 적어야 한다. 그 이중화가 두 타깃의 외관이 갈라지는 경로다.
func writeText(b *strings.Builder, deck *soml.Deck, slide *soml.Slide, elem *soml.Element, path []int) error {
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
	fmt.Fprintf(b, "<div class=\"balpyo-text\" %s=%q %s=%q style=%q>",
		IDAttr, ElementID(slide.Index, path), ElAttr, elem.Name, decls)
	for i, child := range elem.Children {
		if err := writeNode(b, deck, slide, child, childPath(path, i)); err != nil {
			return err
		}
	}
	b.WriteString("</div>\n")
	return nil
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
//
// 여백(64px)과 간격(24px)은 지금 엔진 상수다. 테마의 space 토큰(`slidePadding`/`gap`,
// PRD §8.4)이 Iteration 10에서 이 값을 대체한다 — 지금 core의 토큰 표에 space를 넣으면
// 두 렌더러가 실제로 무엇을 요구하는지 모르는 상태로 어휘를 고정하게 된다 (core/theme.go의
// 같은 판단).
func contentCSS() string {
	return fmt.Sprintf(`
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
}
