package web

import (
	"fmt"
	"html"
	"strings"

	"github.com/k2sebeom/balpyo/soml"
)

// 이 파일은 **레이아웃 태그 → flex/grid CSS**다 (PRD §7.4).
//
// 계약이 하나다: 레이아웃 태그는 **CSS flex/grid 시맨틱의 이름 바꾸기 층**이지 새 배치
// 모델이 아니다. 그래서 이 파일에 배치 계산이 없다 — 태그와 속성이 PRD 표의 CSS로 1:1
// 컴파일되고, 실제 배치는 브라우저가 한다 (그 결과를 지오메트리 패스가 읽어 PPTX 좌표로
// 쓴다 — 좌표 계산 0회의 근거).
//
// # 엔진 CSS와 인라인 style의 경계
//
// 두 자리로 나눈 기준은 **태그의 성질인가, 요소마다 다른 값인가**다 (ADR 0003의 연장 —
// 배치는 엔진 CSS, 요소별 확정 값은 인라인 style):
//
//	엔진 CSS (layoutCSS, 태그마다 고정):
//	  display, flex-direction            — <row>/<col>/<grid>의 정의 그 자체
//	  flex: 1                            — <spacer>의 정의 (밀어내기)
//
//	인라인 style (요소마다 다른 확정 값):
//	  gap, align-items, justify-content, flex-wrap, padding,
//	  grid-template-columns/rows, width/height/min-*/max-*, aspect-ratio   — 컨테이너 속성
//	  flex-grow, flex-shrink, flex-basis, grid-column, align-self, order   — 자식 속성
//	  저자의 `style` 탈출구 (맨 끝)
//
// 고정 선언을 클래스로 두는 이유: 같은 바이트가 요소마다 반복되면 골든이 커지고, 무엇보다
// `display:flex`가 요소 쪽에 있으면 저자 CSS(@layer balpyo.author)가 그것을 덮을 수 없다
// (인라인 style은 층 밖에서 이긴다). 반대로 확정 **값**은 요소마다 달라 클래스로 둘 수 없다.
//
// # 값은 core가 확정한 것만 읽는다
//
// 입력은 soml.ContainerLayout·soml.Placement뿐이다 — 요소의 Attrs를 다시 읽지 않는다
// (soml.Appearance·soml.Placement의 계약). 값 어휘(공간 토큰, 트랙 목록, `fill`, 축약 pad)는
// core가 한 번만 적용했고, 렌더러가 다시 해석하면 CSS와 지오메트리 패스가 서로 다른 값을
// 볼 자리가 생긴다.

// 레이아웃 요소의 엔진 CSS 클래스다. 배치 흐름에서 파생되므로 태그 이름으로 고르지
// 않는다 — 컨테이너 CSS 전부가 Container 구조체 하나에서 나온다는 계약(soml.ContainerFlow)의
// 렌더러 쪽 절반이다.
const (
	rowClass    = "balpyo-row"
	colClass    = "balpyo-col"
	gridClass   = "balpyo-grid"
	spacerClass = "balpyo-spacer"
)

// spaceVarPrefix는 공간 토큰의 CSS 커스텀 프로퍼티 이름 규칙이다 (색·폰트와 같은 규칙 —
// style.go의 colorVarPrefix 주석).
//
// **이 참조는 웹에만 산다**: PPTX는 배치를 좌표로 구우므로 gap이 토큰이었다는 사실이
// 산출물에 남지 않는다 (soml.SpaceToken의 clrScheme 비대칭). 그래도 웹에서 참조로 내는
// 이유는 색과 같다 — 테마를 바꿨을 때 여백이 따라 움직여야 하고, 리터럴 px가 요소 쪽에
// 남으면 그 요소만 옛 리듬으로 남는다.
const spaceVarPrefix = "--balpyo-space-"

// containerClass는 배치 흐름의 엔진 CSS 클래스다.
//
// 모르는 흐름을 에러로 두는 이유: 흐름 값이 늘어나는 것은 태그가 늘어나는 것이므로
// (`<stack>` — Iteration 6), 매핑을 잊으면 "파싱은 되는데 배치가 없는" 컨테이너가 된다.
func containerClass(flow soml.ContainerFlow) (string, error) {
	switch flow {
	case soml.FlowRow:
		return rowClass, nil
	case soml.FlowColumn:
		return colClass, nil
	case soml.FlowGrid:
		return gridClass, nil
	}
	return "", fmt.Errorf("배치 흐름 %d를 CSS로 낼 매핑이 없다", int(flow))
}

// writeContainer는 레이아웃 컨테이너(`<row>`/`<col>`/`<grid>`)를 div로 쓴다.
//
// 식별자(data-balpyo-id)를 갖는다 — 컨테이너는 진짜 박스이므로 rect에 뜻이 있다 (측정 대상
// 계약, 패키지 주석). PPTX가 이 rect를 소비하지 않아도(컨테이너는 도형이 아니다) 픽셀
// 비교가 갈라질 때 "어느 칼럼부터 밀렸는가"를 이 좌표가 말해 준다.
func writeContainer(b *strings.Builder, deck *soml.Deck, slide *soml.Slide, elem *soml.Element, path []int) error {
	container := elem.Container()
	if container == nil {
		// 해석되지 않은 배치를 기본값으로 그리면 웹만 그럴듯하게 나오고 좌표가 PPTX까지 간다.
		return fmt.Errorf("%s:%s <%s>의 배치가 해석되지 않았다 — core.Resolve를 거치지 않은 덱이다",
			slide.Source, elem.Span, elem.Name)
	}
	class, err := containerClass(container.Flow)
	if err != nil {
		return fmt.Errorf("%s:%s <%s>: %w", slide.Source, elem.Span, elem.Name, err)
	}
	decls, err := containerCSS(*container)
	if err != nil {
		return fmt.Errorf("%s:%s <%s>: %w", slide.Source, elem.Span, elem.Name, err)
	}
	decls, err = appendPlacement(decls, elem.Placement())
	if err != nil {
		return fmt.Errorf("%s:%s <%s>: %w", slide.Source, elem.Span, elem.Name, err)
	}
	fmt.Fprintf(b, "<div class=%q %s=%q %s=%q%s>\n",
		class, IDAttr, ElementID(slide.Index, path), ElAttr, elem.Name, styleAttr(decls))
	for i, child := range elem.Children {
		if err := writeNode(b, deck, slide, child, ChildPath(path, i)); err != nil {
			return err
		}
	}
	b.WriteString("</div>\n")
	return nil
}

// writeSpacer는 `<spacer/>`를 빈 div로 쓴다.
//
// void 요소이지만 HTML의 void 표기(`<spacer>`처럼 닫지 않는 태그)로 낼 수는 없다 — div는
// 닫아야 하고, 빈 div가 flex 항목이 되어야 밀어낸다. `flex: 1`은 엔진 CSS에 있고
// (태그의 정의), 저자가 적은 배치 속성(`<spacer grow="2">`)은 인라인 style로 그것을
// 덮는다 — 인라인 style이 층 밖에서 이기므로 순서를 따질 필요가 없다.
func writeSpacer(b *strings.Builder, slide *soml.Slide, elem *soml.Element, path []int) error {
	decls, err := appendPlacement(nil, elem.Placement())
	if err != nil {
		return fmt.Errorf("%s:%s <%s>: %w", slide.Source, elem.Span, elem.Name, err)
	}
	fmt.Fprintf(b, "<div class=%q %s=%q %s=%q%s></div>\n",
		spacerClass, IDAttr, ElementID(slide.Index, path), ElAttr, elem.Name, styleAttr(decls))
	return nil
}

// containerCSS는 확정된 컨테이너 배치를 CSS 선언 목록으로 컴파일한다 (PRD §7.4 컨테이너 표).
//
// **순서는 PRD 표 순서다** — 표와 코드를 나란히 읽을 수 있어야 "표에 있고 코드에 없는 행"이
// 눈에 띈다 (그 대조를 테스트가 어휘 목록 순회로 한 번 더 한다).
//
// 정하지 않은 속성은 선언을 **하나도** 내지 않는다: 기본값을 명시하면 저자 CSS와 브라우저
// 기본값이 이길 자리가 사라지고("브라우저 기본값 그대로"라는 PRD의 계약이 깨진다), 배치
// 속성이 없는 요소의 산출물이 이 iteration 때문에 달라진다.
func containerCSS(container soml.ContainerLayout) ([]string, error) {
	var decls []string
	if container.Gap != nil {
		decls = append(decls, "gap: "+lengthValue(*container.Gap))
	}
	if container.Align != "" {
		align, err := boxAlignValue(container.Align, "align")
		if err != nil {
			return nil, err
		}
		decls = append(decls, "align-items: "+align)
	}
	if container.Justify != "" {
		justify, err := boxJustifyValue(container.Justify)
		if err != nil {
			return nil, err
		}
		decls = append(decls, "justify-content: "+justify)
	}
	if container.Wrap {
		// grid 컨테이너에서 `flex-wrap`은 CSS가 무시하는 선언이다. 그래도 흐름별로 빼지 않는
		// 이유: 속성이 어느 태그에 붙을 수 있는가는 **이름 층**(soml의 카탈로그)의 결정이고,
		// 렌더러가 그 판단을 한 번 더 하면 적용 가능성이 두 곳에서 정해진다 — 파서가
		// `<row cols>`를 막은 것과 같은 자리에서 `<grid wrap>`도 막아야 한다 (NOTES의 열린 질문).
		decls = append(decls, "flex-wrap: wrap")
	}
	if container.Pad != nil {
		decls = append(decls, "padding: "+padValue(*container.Pad))
	}
	if container.Cols != nil {
		decls = append(decls, "grid-template-columns: "+tracksValue(*container.Cols))
	}
	if container.Rows != nil {
		decls = append(decls, "grid-template-rows: "+tracksValue(*container.Rows))
	}
	// 크기 6종은 이름만 다른 같은 매핑이다 — 표로 돌리면 속성이 하나 늘 때 빠뜨릴 자리가 없다.
	for _, size := range []struct {
		property string
		length   *soml.Length
	}{
		{"width", container.Width},
		{"height", container.Height},
		{"min-width", container.MinWidth},
		{"min-height", container.MinHeight},
		{"max-width", container.MaxWidth},
		{"max-height", container.MaxHeight},
	} {
		if size.length != nil {
			decls = append(decls, size.property+": "+lengthValue(*size.length))
		}
	}
	if container.Ratio != nil {
		ratio := *container.Ratio
		if ratio.W <= 0 || ratio.H <= 0 {
			// core가 이미 막았지만(0·음수는 에러) 확정 값이 CSS로 나가기 전 마지막 자리다 —
			// 무효 aspect-ratio는 브라우저가 조용히 버린다.
			return nil, fmt.Errorf("비율 %d:%d이 해석되지 않았다", ratio.W, ratio.H)
		}
		decls = append(decls, fmt.Sprintf("aspect-ratio: %d / %d", ratio.W, ratio.H))
	}
	return decls, nil
}

// appendPlacement는 자식 배치 선언을 이어 붙인다 (PRD §7.4 자식 표).
//
// **배치 선언을 만드는 코드는 이 함수 하나다.** `<text>`·블록·리스트·컨테이너·`<spacer>`가
// 모두 컨테이너의 자식이 될 수 있으므로(soml.Placement의 계약) 요소 종류마다 따로 쓰면
// 태그를 늘릴 때 한 곳을 빼먹고, 그 결과는 "웹에서만 배치가 사라지는 요소"다.
//
// 저자의 `style` 탈출구는 **맨 끝**이다 — 같은 프로퍼티면 뒤가 이기므로 탈출구가 엔진
// 선언을 덮는다. 그것이 탈출구의 뜻이다 (PRD §7.9).
func appendPlacement(decls []string, placement *soml.Placement) ([]string, error) {
	if placement == nil {
		// 배치 속성이 없는 요소는 선언을 하나도 더하지 않는다 — Iteration 1·2 골든이 이
		// nil에 달려 있다 (core가 제로값을 nil로 남기는 계약의 렌더러 쪽 절반).
		return decls, nil
	}
	if placement.Grow != nil {
		decls = append(decls, fmt.Sprintf("flex-grow: %d", *placement.Grow))
	}
	if placement.Shrink != nil {
		decls = append(decls, fmt.Sprintf("flex-shrink: %d", *placement.Shrink))
	}
	if placement.Basis != nil {
		decls = append(decls, "flex-basis: "+lengthValue(*placement.Basis))
	}
	if placement.Span != nil {
		decls = append(decls, fmt.Sprintf("grid-column: span %d", *placement.Span))
	}
	if placement.Self != "" {
		self, err := boxAlignValue(placement.Self, "self")
		if err != nil {
			return nil, err
		}
		decls = append(decls, "align-self: "+self)
	}
	if placement.Order != nil {
		decls = append(decls, fmt.Sprintf("order: %d", *placement.Order))
	}
	if placement.Style != "" {
		// core가 정규화한 선언 목록을 그대로 잇는다 (`프로퍼티: 값`을 `; `로 이은 문자열).
		// 원문 Attrs를 읽지 않는다 — 값 검증·경고를 지나지 않은 문자열이 산출물에 실릴
		// 길을 막는 것이 렌더러의 계약이다.
		decls = append(decls, placement.Style)
	}
	return decls, nil
}

// styleAttr는 선언 목록을 인라인 style 속성으로 만든다 (선언이 없으면 속성 자체가 없다).
//
// 값을 HTML 이스케이프하는 이유는 `style` 탈출구다: 엔진이 만드는 선언은 ASCII 안전하지만
// 저자가 통과시킨 값에는 인용 부호가 들어올 수 있고(`style='content: "x"'`), 그것이
// 그대로 나가면 속성이 여기서 끊겨 마크업이 깨진다.
func styleAttr(decls []string) string {
	if len(decls) == 0 {
		return ""
	}
	return " style=\"" + html.EscapeString(strings.Join(decls, "; ")) + "\""
}

// lengthValue는 확정 길이를 CSS 값으로 만든다.
//
// 공간 토큰은 **변수 참조로 남는다** — 여기서 px로 눕히면 테마의 리듬을 바꿨을 때 이
// 요소만 옛 여백으로 남는다 (colorValue가 토큰을 var()로 남기는 것과 같은 근거).
func lengthValue(length soml.Length) string {
	switch {
	case length.IsToken():
		return fmt.Sprintf("var(%s%s)", spaceVarPrefix, length.Token)
	case length.Percent:
		return formatNumber(length.Value) + "%"
	default:
		// 무단위 수치는 논리 스테이지 px다 (PRD §13-5). 스테이지가 CSS px와 1:1이므로
		// 환산이 없다 — 지오메트리 패스가 읽은 값이 그대로 PPTX 좌표가 되는 이유다.
		return formatNumber(length.Value) + "px"
	}
}

// padValue는 4변 패딩을 CSS `padding` 값으로 만든다 — 항상 4값이다.
//
// 축약으로 되돌리지 않는 이유: core가 CSS 규칙으로 이미 풀었고(soml.Pad), 렌더러가 다시
// 축약하면 축약 규칙이 두 곳에 생긴다. 산출물이 조금 길어지는 대신 `padding`의 네 변이
// 골든에서 눈에 보인다.
func padValue(pad soml.Pad) string {
	return strings.Join([]string{
		lengthValue(pad.Top), lengthValue(pad.Right), lengthValue(pad.Bottom), lengthValue(pad.Left),
	}, " ")
}

// tracksValue는 확정 트랙을 `grid-template-columns/rows` 값으로 만든다.
//
// 개수는 `repeat(N, 1fr)`로 낸다 — `1fr`을 N번 적은 것과 같은 뜻이지만 골든이 저자가 쓴
// `cols="4"`를 그대로 읽히게 한다 (soml.Tracks가 두 상태를 유지하는 이유).
func tracksValue(tracks soml.Tracks) string {
	if tracks.Count > 0 {
		return fmt.Sprintf("repeat(%d, 1fr)", tracks.Count)
	}
	parts := make([]string, 0, len(tracks.List))
	for _, track := range tracks.List {
		parts = append(parts, trackValue(track))
	}
	return strings.Join(parts, " ")
}

// trackValue는 트랙 하나의 CSS 값이다.
func trackValue(track soml.Track) string {
	switch track.Kind {
	case soml.TrackAuto:
		return "auto"
	case soml.TrackFr:
		return formatNumber(track.Value) + "fr"
	case soml.TrackPercent:
		return formatNumber(track.Value) + "%"
	default: // soml.TrackPx
		return formatNumber(track.Value) + "px"
	}
}

// boxAlignValue는 교차축 정렬을 CSS 값으로 만든다 — 논리 이름이 그대로 통한다.
//
// 이름이 통하는 것은 우연이 아니다: CSS Box Alignment의 `start`/`end`가 flex와 grid에서
// 같은 뜻이어서 soml이 접두 없는 이름을 골랐다 (soml.BoxAlign의 주석). 그래도 어휘를
// 검사하는 이유는 alignValue와 같다 — 해석되지 않은 빈 값이나 물리 이름(`left`)이 CSS로
// 흘러가면 브라우저가 조용히 무시한다.
func boxAlignValue(align soml.BoxAlign, attr string) (string, error) {
	for _, known := range soml.BoxAligns() {
		if align == known {
			return string(align), nil
		}
	}
	return "", fmt.Errorf("%s %q는 교차축 정렬 어휘 밖이다", attr, string(align))
}

// boxJustifyValue는 주축 분배를 CSS 값으로 만든다 — `space-*` 접두를 다시 붙이는 자리다.
//
// 접두를 뗀 이름을 어휘로 고른 것은 PRD §7.4이고(저자가 `between`으로 적는다), 번역은
// 렌더러의 몫이라고 정한 것도 거기다 (soml.BoxJustify의 주석).
func boxJustifyValue(justify soml.BoxJustify) (string, error) {
	switch justify {
	case soml.BoxJustifyStart, soml.BoxJustifyCenter, soml.BoxJustifyEnd:
		return string(justify), nil
	case soml.BoxJustifyBetween, soml.BoxJustifyAround, soml.BoxJustifyEvenly:
		return "space-" + string(justify), nil
	}
	return "", fmt.Errorf("justify %q는 주축 분배 어휘 밖이다", string(justify))
}

// layoutCSS는 레이아웃 태그의 엔진 CSS다 — **태그마다 고정인 선언만** 있다 (파일 머리
// 주석의 경계). PRD §7.4의 컴파일 표를 그대로 옮긴 것이고, 그 밖의 배치 규칙(폭·정렬
// 기본값)을 여기 넣지 않는 이유는 "브라우저 기본값 그대로"가 계약이기 때문이다.
//
// 레이아웃 태그가 있는 덱에만 낸다 (listCSS와 같은 근거): 레이아웃이 없는 덱의 산출물이
// 레이아웃 렌더가 붙었다는 이유로 달라지면 골든 diff가 무엇이 움직였는지 말해 주지 못한다.
func layoutCSS() string {
	return `
  /* 레이아웃 태그 = CSS flex/grid의 이름 바꾸기 층 (PRD §7.4). 여기 있는 것은 태그의
     정의 그 자체이고, 요소마다 다른 값(gap·align·pad·cols…)은 인라인 style로 나간다. */
  .balpyo-row { display: flex; flex-direction: row; }
  .balpyo-col { display: flex; flex-direction: column; }
  .balpyo-grid { display: grid; }

  /* <spacer>는 밀어내기가 내용인 빈 박스다. flex: 1은 남은 공간을 전부 받는다는 뜻이고,
     저자가 적은 배치 속성(<spacer grow="2">)은 인라인 style로 이 값을 덮는다. */
  .balpyo-spacer { flex: 1; }

  /* 레이아웃 컨테이너의 자식은 **브라우저 기본값**으로 돌아간다.
     스테이지의 세로 스택은 텍스트·리스트에 확정 폭(width: 100%)과 눌리지 않는
     flex(0 0 auto)를 준다 (contentCSS) — 스택 안에서 줄바꿈을 결정적으로 만들기 위한
     규칙이다. 그 규칙이 컨테이너 안까지 따라오면 자식마다 부모 폭 전체를 요구하는 셈이 되어
     가로 컨테이너를 넘치고, <spacer>가 받을 "남은 공간"이 사라진다 (실측으로 드러났다 —
     render/geom의 spacer 테스트). 컨테이너 안에서 폭을 정하는 것은 컨테이너다: 트랙,
     교차축 정렬(stretch), 그리고 자식이 적은 basis/grow가 그 값을 만든다.
     폭이 확정된다는 사실은 그대로다 — 브라우저가 정하고 지오메트리 패스가 그 값을 잰다. */
  :is(.balpyo-row, .balpyo-col, .balpyo-grid) > :is(.balpyo-text, .balpyo-list) {
    width: auto;
    flex: 0 1 auto;
  }
`
}

// spaceVarsCSS는 공간 토큰의 CSS 변수 선언이다 — 참조(lengthValue)와 선언이 **한 표**에서
// 나온다는 계약의 나머지 절반이다 (색·폰트와 같은 구조 — themeCSS).
//
// 선언 순서는 표 순서다 (soml.Theme가 맵이 아닌 이유) — 골든이 실행마다 같아야 한다.
func spaceVarsCSS(theme soml.Theme) string {
	var b strings.Builder
	for _, token := range theme.Spaces {
		fmt.Fprintf(&b, "    %s%s: %spx;\n", spaceVarPrefix, token.Name, formatNumber(token.Px))
	}
	return b.String()
}
