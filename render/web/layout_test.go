package web

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/k2sebeom/balpyo/core"
	"github.com/k2sebeom/balpyo/soml"
)

// 이 파일은 **PRD §7.4 표의 회귀 장치**다 — 레이아웃 태그는 CSS flex/grid의 이름 바꾸기 층이다.
// 두 표(containerCases·placementCases)는 속성 이름을 들고 다녀 어휘 목록과 대조된다.

// layoutCase는 속성 하나가 CSS 선언 하나로 컴파일되는 것을 확인하는 행이다.
type layoutCase struct {
	// attr은 어휘 목록의 속성 이름이다 (커버리지 대조의 열쇠).
	attr string
	name string
	// tag는 이 속성을 받는 컨테이너 태그다 (트랙 속성은 `<grid>` 전용이다).
	tag string
	// decl은 슬라이드 원문에 쓸 속성 표기다.
	decl string
	// want는 인라인 style에 나와야 하는 CSS 선언이다.
	want string
}

// containerCases는 PRD §7.4 **컨테이너 표**를 그대로 옮긴 것이다 (표 순서).
func containerCases() []layoutCase {
	return []layoutCase{
		// gap — 공간 토큰은 변수 참조로, 수치는 논리 px로, `%`는 백분율로 나간다.
		{"gap", "공간 토큰", "row", `gap="lg"`, "gap: var(--balpyo-space-lg)"},
		{"gap", "수치", "row", `gap="16"`, "gap: 16px"},
		{"gap", "백분율", "row", `gap="10%"`, "gap: 10%"},

		{"align", "start", "row", `align="start"`, "align-items: start"},
		{"align", "center", "row", `align="center"`, "align-items: center"},
		{"align", "end", "row", `align="end"`, "align-items: end"},
		{"align", "stretch", "row", `align="stretch"`, "align-items: stretch"},
		{"align", "baseline", "row", `align="baseline"`, "align-items: baseline"},

		// justify — 접두 없는 이름을 CSS의 `space-*`로 되돌리는 것이 렌더러의 몫이다.
		{"justify", "start", "row", `justify="start"`, "justify-content: start"},
		{"justify", "center", "row", `justify="center"`, "justify-content: center"},
		{"justify", "end", "row", `justify="end"`, "justify-content: end"},
		{"justify", "between", "row", `justify="between"`, "justify-content: space-between"},
		{"justify", "around", "row", `justify="around"`, "justify-content: space-around"},
		{"justify", "evenly", "row", `justify="evenly"`, "justify-content: space-evenly"},

		{"wrap", "값 없는 표기", "row", `wrap`, "flex-wrap: wrap"},
		{"wrap", "true", "row", `wrap="true"`, "flex-wrap: wrap"},

		// pad — core가 4변으로 푼 값을 렌더러가 다시 축약하지 않는다 (padValue의 근거).
		{"pad", "1값 토큰", "row", `pad="md"`,
			"padding: var(--balpyo-space-md) var(--balpyo-space-md) var(--balpyo-space-md) var(--balpyo-space-md)"},
		{"pad", "2값 토큰·수치", "row", `pad="sm 24"`,
			"padding: var(--balpyo-space-sm) 24px var(--balpyo-space-sm) 24px"},
		{"pad", "4값", "row", `pad="1 2 3 4%"`, "padding: 1px 2px 3px 4%"},

		// 트랙 — 수치 하나는 개수(repeat), 그 밖은 목록이다.
		{"cols", "개수", "grid", `cols="4"`, "grid-template-columns: repeat(4, 1fr)"},
		{"cols", "트랙 목록", "grid", `cols="2fr 1fr auto 25% 120"`,
			"grid-template-columns: 2fr 1fr auto 25% 120px"},
		{"rows", "트랙 목록", "grid", `rows="auto 1fr"`, "grid-template-rows: auto 1fr"},

		// 크기 6종 — `fill`은 값 층에서 해당 축의 100%로 눕는다 (soml.Length).
		{"w", "수치", "row", `w="320"`, "width: 320px"},
		{"w", "fill", "row", `w="fill"`, "width: 100%"},
		{"h", "백분율", "row", `h="50%"`, "height: 50%"},
		{"min-w", "수치", "row", `min-w="200"`, "min-width: 200px"},
		{"min-h", "수치", "row", `min-h="120"`, "min-height: 120px"},
		{"max-w", "수치", "row", `max-w="360"`, "max-width: 360px"},
		{"max-h", "수치", "row", `max-h="220"`, "max-height: 220px"},

		// ratio — 정수 쌍이 그대로 나간다 (하나의 부동소수로 눕히지 않는 이유는 soml.Ratio).
		{"ratio", "16:9", "row", `ratio="16:9"`, "aspect-ratio: 16 / 9"},
		{"ratio", "1:1", "row", `ratio="1:1"`, "aspect-ratio: 1 / 1"},
	}
}

// placementCases는 PRD §7.4 **자식 표**를 그대로 옮긴 것이다 (표 순서).
//
// tag는 자식을 담는 부모 컨테이너다 — `span`은 `<grid>`의 자식에서만 뜻이 있다 (core가
// 부모를 보고 검증한다).
func placementCases() []layoutCase {
	return []layoutCase{
		{"grow", "수치", "row", `grow="2"`, "flex-grow: 2"},
		{"grow", "0", "row", `grow="0"`, "flex-grow: 0"},
		{"shrink", "0", "row", `shrink="0"`, "flex-shrink: 0"},
		{"basis", "수치", "row", `basis="240"`, "flex-basis: 240px"},
		{"basis", "백분율", "row", `basis="50%"`, "flex-basis: 50%"},
		{"span", "grid 자식", "grid", `span="2"`, "grid-column: span 2"},
		{"self", "center", "row", `self="center"`, "align-self: center"},
		{"self", "baseline", "row", `self="baseline"`, "align-self: baseline"},
		// order는 음수를 받는다 — "먼저 보이게"의 CSS 관용 표기다.
		{"order", "음수", "row", `order="-1"`, "order: -1"},
		// at — 9지점 앵커가 place-self 조합으로 나간다. 부모는 stack이다 (core가 부모를 보고
		// at을 stack 자식으로 한정한다). 축 순서(place-self: <세로> <가로>)가 뒤집히면 9지점
		// 중 4개가 대각으로 어긋나므로 9종을 전수로 둔다 (표 순서 = StackAnchors 순서).
		{"at", "top-left", "stack", `at="top-left"`, "place-self: start start"},
		{"at", "top-center", "stack", `at="top-center"`, "place-self: start center"},
		{"at", "top-right", "stack", `at="top-right"`, "place-self: start end"},
		{"at", "center-left", "stack", `at="center-left"`, "place-self: center start"},
		{"at", "center", "stack", `at="center"`, "place-self: center center"},
		{"at", "center-right", "stack", `at="center-right"`, "place-self: center end"},
		{"at", "bottom-left", "stack", `at="bottom-left"`, "place-self: end start"},
		{"at", "bottom-center", "stack", `at="bottom-center"`, "place-self: end center"},
		{"at", "bottom-right", "stack", `at="bottom-right"`, "place-self: end end"},
		// x/y — 그 자식만 absolute로 전환하고 부모 상대 좌표를 left/top으로 낸다. 좌표는
		// 크기가 아니므로 음수를 받는다 (박스 밖으로 걸친 배지). transform:translate은 쓰지
		// 않는다 (측정 뷰가 변형을 끄기 때문 — appendPlacement의 근거).
		{"x", "수치", "stack", `x="20"`, "left: 20px"},
		{"y", "백분율", "stack", `y="10%"`, "top: 10%"},
		{"style", "배치 탈출구", "row", `style="place-self: end"`, "place-self: end"},
	}
}

// 태그 → display/flex-direction (PRD §7.4의 컴파일 표). 고정 선언은 엔진 CSS 클래스에
// 있고 요소는 클래스만 들고 나온다 (layout.go).
func TestLayoutTagsCompileToFlexGridCSS(t *testing.T) {
	out := renderString(t, subsetDeck(t, `
  <row><p>가로</p></row>
  <col><p>세로</p></col>
  <grid cols="2"><p>격자</p></grid>
  <spacer/>`))
	css := between(t, out, "<style>", "</style>")
	for _, tc := range []struct {
		el    string
		id    string
		class string
		rule  string
	}{
		{"row", "s0.0", rowClass, ".balpyo-row { display: flex; flex-direction: row; }"},
		{"col", "s0.1", colClass, ".balpyo-col { display: flex; flex-direction: column; }"},
		{"grid", "s0.2", gridClass, ".balpyo-grid { display: grid; }"},
		{"spacer", "s0.3", spacerClass, ".balpyo-spacer { flex: 1; }"},
	} {
		open := openTagOf(t, out, tc.id)
		if want := fmt.Sprintf("class=%q", tc.class); !strings.Contains(open, want) {
			t.Errorf("<%s>의 여는 태그에 %s가 없다: %s", tc.el, want, open)
		}
		if want := fmt.Sprintf("%s=%q", ElAttr, tc.el); !strings.Contains(open, want) {
			t.Errorf("<%s>의 여는 태그에 %s가 없다: %s", tc.el, want, open)
		}
		if !strings.Contains(css, tc.rule) {
			t.Errorf("엔진 CSS에 %q가 없다", tc.rule)
		}
		// 고정 선언이 요소 쪽에 새면 저자 CSS가 그것을 덮을 수 없다 (인라인 style이 층 밖에서
		// 이긴다) — 경계가 무너진 것을 여기서 잡는다.
		for _, leaked := range []string{"display:", "flex-direction:", "flex: 1"} {
			if strings.Contains(open, leaked) {
				t.Errorf("<%s>의 인라인 style에 태그 고정 선언 %q가 새어 나왔다: %s", tc.el, leaked, open)
			}
		}
	}
}

// 컨테이너 자식은 스테이지 스택의 확정 폭 규칙에서 벗어나 **브라우저 기본값**으로 돌아간다.
// 그 규칙이 컨테이너 안까지 따라오면 자식마다 부모 폭 전체를 요구해 `<spacer>`가 받을 남은
// 공간이 사라진다 (render/geom의 spacer 실측이 그것을 잡았다).
func TestLayoutResetsStageWidthRuleInsideContainers(t *testing.T) {
	css := between(t, renderString(t, subsetDeck(t, `<row gap="md"><p>자식</p></row>`)), "<style>", "</style>")
	for _, want := range []string{
		":is(.balpyo-row, .balpyo-col, .balpyo-grid) > :is(.balpyo-text, .balpyo-list) {",
		"width: auto;",
		"flex: 0 1 auto;",
	} {
		if !strings.Contains(css, want) {
			t.Errorf("컨테이너 자식 초기화 CSS에 %q가 없다", want)
		}
	}
	// 스테이지 스택의 규칙은 그대로다 — 두 규칙이 각자의 자리에서 살아 있어야 한다.
	for _, want := range []string{"width: 100%;", "flex: 0 0 auto;"} {
		if !strings.Contains(css, want) {
			t.Errorf("스테이지 스택의 확정 폭 규칙 %q가 사라졌다", want)
		}
	}
}

// 컨테이너 속성 → CSS 선언 (PRD §7.4 컨테이너 표 그대로).
func TestContainerAttributesCompileToCSS(t *testing.T) {
	for _, tc := range containerCases() {
		t.Run(tc.attr+"/"+tc.name, func(t *testing.T) {
			out := renderString(t, subsetDeck(t, containerBody(tc)))
			open := openTagOf(t, out, "s0.0")
			if !strings.Contains(open, tc.want) {
				t.Errorf("<%s %s>의 인라인 style에 %q가 없다: %s", tc.tag, tc.decl, tc.want, open)
			}
		})
	}
}

// 자식 배치 속성 → CSS 선언 (PRD §7.4 자식 표 그대로). 컨테이너의 자식에 붙는다.
func TestChildPlacementAttributesCompileToCSS(t *testing.T) {
	for _, tc := range placementCases() {
		t.Run(tc.attr+"/"+tc.name, func(t *testing.T) {
			out := renderString(t, subsetDeck(t, placementBody(tc)))
			open := openTagOf(t, out, "s0.0.0")
			if !strings.Contains(open, tc.want) {
				t.Errorf("<p %s>의 인라인 style에 %q가 없다: %s", tc.decl, tc.want, open)
			}
		})
	}
}

// **표에 있고 코드에 없는 행이 남지 않는다**: 속성이 하나 늘면 이 테스트가 먼저 빨개진다 —
// 매핑을 잊은 속성은 "파싱은 되는데 CSS로 나가지 않는" 조용한 실패가 된다.
func TestLayoutVocabularyIsFullyMapped(t *testing.T) {
	attrs := func(cases []layoutCase) []string {
		var names []string
		for _, tc := range cases {
			names = append(names, tc.attr)
		}
		return names
	}
	// `<grid>`의 목록이 컨테이너 속성 전체다 (flex 공통 + 트랙 + 크기).
	assertCoversCatalog(t, "컨테이너 속성", attrs(containerCases()), soml.ContainerAttrs(soml.GridElement))
	assertCoversCatalog(t, "자식 배치 속성", attrs(placementCases()), soml.ChildPlacementAttrs())
}

// 배치 선언을 만드는 코드는 하나다 (appendPlacement) — 자식이 될 수 있는 요소 종류 전부가
// 같은 선언을 낸다. 종류마다 따로 쓰면 태그를 늘릴 때 한 곳을 빼먹고, 그 결과는 "웹에서만
// 배치가 사라지는 요소"다.
func TestPlacementAppliesToEveryChildKind(t *testing.T) {
	out := renderString(t, subsetDeck(t, `
  <row gap="md">
    <text grow="1">text</text>
    <p grow="1">블록</p>
    <ul grow="1"><li>리스트</li></ul>
    <col grow="1"><p>중첩 컨테이너</p></col>
    <spacer grow="1"/>
  </row>`))
	for _, tc := range []struct {
		el string
		id string
	}{
		{"text", "s0.0.0"},
		{"p", "s0.0.1"},
		{"ul", "s0.0.2"},
		{"col", "s0.0.3"},
		{"spacer", "s0.0.4"},
	} {
		open := openTagOf(t, out, tc.id)
		if !strings.Contains(open, "flex-grow: 1") {
			t.Errorf("<%s>(%s)의 인라인 style에 flex-grow: 1이 없다: %s", tc.el, tc.id, open)
		}
	}
}

// 공간 토큰은 **참조와 선언이 한 표에서 나온다**: 요소는 var()로 가리키고 테마 층이 같은
// 이름을 선언한다. 리터럴 px가 요소 쪽에 남으면 테마의 리듬을 바꿨을 때 그 요소만 옛
// 여백으로 남는다 (색·폰트와 같은 계약).
func TestSpaceTokenReferenceAndDeclarationComeFromOneTable(t *testing.T) {
	deck := subsetDeck(t, `<row gap="xl" pad="2xl"><p>토큰</p></row>`)
	out := renderString(t, deck)
	open := openTagOf(t, out, "s0.0")
	for _, want := range []string{"gap: var(--balpyo-space-xl)", "var(--balpyo-space-2xl)"} {
		if !strings.Contains(open, want) {
			t.Errorf("컨테이너의 인라인 style에 %q가 없다: %s", want, open)
		}
	}
	// 해석된 px(48·72)가 요소 쪽에 새면 참조가 아니라 값이 나간 것이다.
	for _, leaked := range []string{"gap: 48px", "72px"} {
		if strings.Contains(open, leaked) {
			t.Errorf("공간 토큰이 리터럴 px로 나갔다 (%q): %s", leaked, open)
		}
	}
	css := between(t, out, "<style>", "</style>")
	if len(deck.Theme.Spaces) == 0 {
		t.Fatal("테마에 공간 토큰 표가 없다 — 참조와 선언이 같은 표에서 나오는지 볼 수 없다")
	}
	for _, token := range deck.Theme.Spaces {
		want := fmt.Sprintf("%s%s: %spx;", spaceVarPrefix, token.Name, formatNumber(token.Px))
		if !strings.Contains(css, want) {
			t.Errorf("테마 층에 공간 토큰 선언 %q가 없다", want)
		}
	}
}

// `style` 탈출구는 인라인 style **맨 끝**에 붙는다 — 같은 프로퍼티면 뒤가 이기므로 저자의
// 탈출구가 엔진 선언을 덮는다. 그것이 탈출구의 뜻이다 (PRD §7.9).
func TestStyleEscapeHatchComesLast(t *testing.T) {
	out := renderString(t, subsetDeck(t, `
  <row gap="md" style="gap: 40px">
    <p grow="1" style="flex-grow: 3">덮어쓴다</p>
  </row>`))
	container := openTagOf(t, out, "s0.0")
	engine, hatch := strings.Index(container, "gap: var(--balpyo-space-md)"), strings.Index(container, "gap: 40px")
	if engine < 0 || hatch < 0 {
		t.Fatalf("엔진 선언과 탈출구가 함께 나오지 않았다: %s", container)
	}
	if engine > hatch {
		t.Errorf("탈출구가 엔진 선언보다 앞에 있다 — 저자가 덮을 수 없다: %s", container)
	}
	block := openTagOf(t, out, "s0.0.0")
	if e, h := strings.Index(block, "flex-grow: 1"), strings.Index(block, "flex-grow: 3"); e < 0 || h < 0 || e > h {
		t.Errorf("블록의 탈출구가 맨 끝이 아니다: %s", block)
	}
	// 텍스트 메트릭(확정 외관)보다도 뒤다 — 배치 탈출구가 외관을 덮을 일은 없지만, 순서가
	// 뒤집히면 저자가 배치를 덮을 수 없는 자리가 생긴다.
	if m, h := strings.Index(block, "text-align:"), strings.Index(block, "flex-grow: 3"); m > h {
		t.Errorf("탈출구가 확정 외관보다 앞에 있다: %s", block)
	}
}

// 탈출구 값은 HTML 이스케이프된다 — 인용 부호가 그대로 나가면 속성이 거기서 끊겨 마크업이
// 깨진다 (core는 값 문법을 보지 않으므로 이 자리가 마지막이다).
func TestStyleEscapeHatchIsEscaped(t *testing.T) {
	out := renderString(t, subsetDeck(t, `<row gap="md" style='place-content: end; content: "x"'><p>따옴표</p></row>`))
	open := openTagOf(t, out, "s0.0")
	if !strings.Contains(open, "&#34;x&#34;") {
		t.Errorf("탈출구의 인용 부호가 이스케이프되지 않았다: %s", open)
	}
	// 속성이 끊기지 않았다는 것은 여는 태그 안에 맨몸 `"`가 style 값으로 남지 않았다는 뜻이다.
	if strings.Contains(open, `content: "x"`) {
		t.Errorf("인용 부호가 원문 그대로 나갔다: %s", open)
	}
}

// 렌더러는 core가 확정한 값만 읽는다 — 원문 Attrs를 다시 읽는 길이 없어야 한다. 확정 배치를
// 지운 덱은 배치 선언을 하나도 내지 않는다 (Attrs에 그 값이 그대로 남아 있어도).
func TestRendererIgnoresRawAttrs(t *testing.T) {
	deck := subsetDeck(t, `<row gap="md"><p grow="2" style="order: 3">본문</p></row>`)
	row := deck.Slides[0].Content[0].(*soml.Element)
	block := row.Children[0].(*soml.Element)
	row.Appearance.Container = &soml.ContainerLayout{Flow: soml.FlowRow}
	block.Appearance.Placement = nil
	out := renderString(t, deck)
	if got := openTagOf(t, out, "s0.0"); strings.Contains(got, "gap") {
		t.Errorf("확정 배치에 없는 gap이 Attrs에서 나왔다: %s", got)
	}
	if got := openTagOf(t, out, "s0.0.0"); strings.Contains(got, "flex-grow") || strings.Contains(got, "order") {
		t.Errorf("확정 배치가 없는데 자식 배치 선언이 나왔다: %s", got)
	}
}

// 어휘 밖의 값이 CSS로 흘러가면 브라우저가 조용히 무시한다 — 확정 값이 어휘 안임을 렌더러가
// 마지막으로 확인한다 (alignValue와 같은 태도).
func TestRenderRejectsUnknownBoxVocabulary(t *testing.T) {
	for _, tc := range []struct {
		name  string
		patch func(*soml.Element)
	}{
		{"align", func(row *soml.Element) { row.Appearance.Container.Align = soml.BoxAlign("flex-start") }},
		{"justify", func(row *soml.Element) { row.Appearance.Container.Justify = soml.BoxJustify("space-between") }},
		{"흐름", func(row *soml.Element) { row.Appearance.Container.Flow = soml.ContainerFlow(9) }},
		{"ratio", func(row *soml.Element) { row.Appearance.Container.Ratio = &soml.Ratio{} }},
		{"self", func(row *soml.Element) {
			child := row.Children[0].(*soml.Element)
			child.Appearance.Placement = &soml.Placement{Self: soml.BoxAlign("left")}
		}},
		// 해석되지 않은 배치를 기본값으로 그리면 웹만 그럴듯하게 나오고, 그 좌표가 PPTX까지 간다.
		{"해석되지 않은 컨테이너", func(row *soml.Element) { row.Appearance.Container = nil }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			deck := subsetDeck(t, `<row gap="md"><p>본문</p></row>`)
			tc.patch(deck.Slides[0].Content[0].(*soml.Element))
			if _, err := New().Render(deck); err == nil {
				t.Errorf("%s는 에러여야 한다", tc.name)
			}
		})
	}
}

// `<stack>`과 캔버스 스테이지는 같은 단일 셀 grid다 — 모든 자식이 `grid-area: 1/1`에 겹치고
// 크기는 가장 큰 자식이 정한다 (PRD §13-3). absolute 전면 방식은 컨테이너 크기를 잃고,
// `grid-template-*`을 내면 암시 트랙 하나라는 계약이 깨진다. 자식의 `position: relative`는
// 칠 순서를 문서 순서(= PPTX 순서)로 되돌린다.
func TestStackCompilesToSingleCellGridOverlap(t *testing.T) {
	out := renderString(t, subsetDeck(t, `<stack><p>겹침</p></stack>`))
	if open := openTagOf(t, out, "s0.0"); !strings.Contains(open, fmt.Sprintf("class=%q", stackClass)) {
		t.Errorf("<stack>의 여는 태그에 %s 클래스가 없다: %s", stackClass, open)
	}
	css := between(t, out, "<style>", "</style>")
	for _, want := range []string{".balpyo-stack { display: grid;", "grid-area: 1 / 1;", "position: relative;"} {
		if !strings.Contains(css, want) {
			t.Errorf("stack 엔진 CSS에 %q가 없다", want)
		}
	}
	stackRule := between(t, css, ".balpyo-stack {", "}")
	if strings.Contains(stackRule, "grid-template-") {
		t.Errorf("stack 규칙에 grid-template-*이 있다 — 암시 트랙 하나여야 한다: %s", stackRule)
	}
	canvas := renderString(t, layoutDeck(t, "canvas", `<text x="0" y="0">모서리</text>`))
	if !strings.Contains(canvas, ".balpyo-stage-canvas { display: grid;") {
		t.Error("캔버스 스테이지 CSS가 stack과 같은 단일 셀 grid가 아니다")
	}
}

// `x`/`y`는 그 자식만 absolute로 전환하고 좌표를 left/top으로 낸다. transform:translate은 쓰지
// 않는다 — 측정 뷰가 변형을 끄므로(ADR 0021) 잰 rect에서 위치가 사라진다. 지정하지 않은 축은
// 선언을 내지 않는다 (정적 위치).
func TestAbsolutePlacementUsesLeftTop(t *testing.T) {
	for _, tc := range []struct {
		name       string
		deck       func() *soml.Deck
		want, gone []string
	}{
		{"x·y", func() *soml.Deck { return subsetDeck(t, `<stack><p x="20" y="30">배지</p></stack>`) },
			[]string{"position: absolute", "left: 20px", "top: 30px"}, []string{"transform", "translate"}},
		{"x만", func() *soml.Deck { return subsetDeck(t, `<stack><p x="20">가로만</p></stack>`) },
			[]string{"left: 20px"}, []string{"top:"}},
		{"y만", func() *soml.Deck { return subsetDeck(t, `<stack><p y="30">세로만</p></stack>`) },
			[]string{"top: 30px"}, []string{"left:"}},
		{"캔버스 직속 자식", func() *soml.Deck { return layoutDeck(t, "canvas", `<text x="0" y="0">모서리</text>`) },
			[]string{"position: absolute", "left: 0px"}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			deck := tc.deck()
			id := "s0.0.0"
			if deck.Slides[0].Layout == "canvas" {
				id = "s0.0"
			}
			open := openTagOf(t, renderString(t, deck), id)
			for _, want := range tc.want {
				if !strings.Contains(open, want) {
					t.Errorf("style에 %q가 없다: %s", want, open)
				}
			}
			for _, gone := range tc.gone {
				if strings.Contains(open, gone) {
					t.Errorf("style에 %q가 나갔다: %s", gone, open)
				}
			}
			if got := strings.Count(open, "position: absolute"); got != 1 {
				t.Errorf("position: absolute가 %d번 나온다, want 1: %s", got, open)
			}
		})
	}
}

// 스테이지의 여는 태그가 슬라이드 수준 결정을 싣는다: 배치 CSS가 있는 레이아웃은 클래스를,
// 슬롯을 쓴 슬라이드는 `balpyo-slots`(ADR 0035)를, `justify`는 인라인 분배를(ADR 0044) 얻는다.
// 흐름 규칙은 `:not(.balpyo-slots)`로 슬롯 규칙과 배타적으로 겨눈다.
func TestStageCarriesSlideLayout(t *testing.T) {
	for _, tc := range []struct {
		name, attrs, body string
		stage, css, gone  string
	}{
		{"content 흐름", `layout="content"`, `<h1>제목</h1><p>본문</p>`,
			`<div class="balpyo-stage">`, "", "grid-area:"},
		{"justify", `justify="between"`, `<p>위</p><p>아래</p>`,
			`<div class="balpyo-stage" style="justify-content: space-between">`, "", ""},
		{"title", `layout="title"`, `<h1>제목</h1>`, `<div class="balpyo-stage balpyo-layout-title">`,
			".balpyo-stage.balpyo-layout-title:not(.balpyo-slots) { justify-content: center; }", ""},
		{"two-column", `layout="two-column"`, `<p>왼쪽</p><p>오른쪽</p>`,
			`<div class="balpyo-stage balpyo-layout-two-column">`, "grid-template-columns: 1fr 1fr", ""},
		{"canvas", `layout="canvas"`, `<text x="0" y="0">모서리</text>`,
			`<div class="balpyo-stage balpyo-stage-canvas">`, "", ""},
		{"슬롯", `layout="content"`, `<h1 slot="title">제목</h1><p slot="body">본문</p>`,
			`<div class="balpyo-stage balpyo-layout-content balpyo-slots">`, `grid-template-areas: "title" "body"`, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			out := renderString(t, slideAttrDeck(t, tc.attrs, tc.body))
			if !strings.Contains(out, tc.stage) {
				t.Errorf("스테이지 여는 태그 %q가 없다", tc.stage)
			}
			if tc.css != "" && !strings.Contains(out, tc.css) {
				t.Errorf("배치 CSS %q가 없다", tc.css)
			}
			if tc.gone != "" && strings.Contains(out, tc.gone) {
				t.Errorf("%q가 나왔다", tc.gone)
			}
		})
	}
	// 슬롯 자식마다 `grid-area: <슬롯>`이다.
	out := renderString(t, layoutDeck(t, "content", `<h1 slot="title">제목</h1><p slot="body">본문</p>`))
	for id, want := range map[string]string{"s0.0": "grid-area: title", "s0.1": "grid-area: body"} {
		if open := openTagOf(t, out, id); !strings.Contains(open, want) {
			t.Errorf("%s에 %q가 없다: %s", id, want, open)
		}
	}
}

// layoutDeck은 지정한 layout을 실은 슬라이드 하나의 덱이다 (subsetDeck은 layout을 실을 수 없다).
func layoutDeck(t *testing.T, layout, body string) *soml.Deck {
	t.Helper()
	return slideAttrDeck(t, `layout="`+layout+`"`, body)
}

// slideAttrDeck은 슬라이드 수준 속성을 원문으로 받아 파이프라인처럼 덱을 만든다.
func slideAttrDeck(t *testing.T, attrs, body string) *soml.Deck {
	t.Helper()
	const source = "slides/001_layout.slide"
	doc, diags := soml.Parse(source, `<slide id="l" `+attrs+`>`+"\n"+body+"\n</slide>\n")
	if err := diags.Err(); err != nil {
		t.Fatalf("파싱 실패: %v", err)
	}
	stage, err := soml.Aspect16x9.Stage()
	if err != nil {
		t.Fatal(err)
	}
	deck := &soml.Deck{
		Title: "레이아웃", Lang: "ko", Aspect: soml.Aspect16x9, Stage: stage,
		Slides: []*soml.Slide{{
			Index: 0, ID: "l", Source: source,
			Layout: soml.SlideLayout(doc.Layout), LayoutSpan: doc.LayoutSpan,
			Justify: soml.BoxJustify(doc.Justify), JustifySpan: doc.JustifySpan,
			Notes: doc.Notes, Content: doc.Content,
		}},
	}
	if err := core.Resolve(deck, nil).Err(); err != nil {
		t.Fatalf("외관 해석 실패: %v", err)
	}
	return deck
}

// containerBody는 컨테이너 속성 한 행의 슬라이드 본문이다 — 컨테이너는 슬라이드 직속(s0.0)이다.
func containerBody(tc layoutCase) string {
	if tc.tag == soml.GridElement && tc.attr != "cols" {
		// `<grid>`는 트랙 없이도 유효하지만, 트랙 속성 행이 아닌데 grid를 쓸 일은 없다.
		return fmt.Sprintf("<grid %s><p>격자</p></grid>", tc.decl)
	}
	return fmt.Sprintf("<%s %s><p>자식</p></%s>", tc.tag, tc.decl, tc.tag)
}

// placementBody는 자식 배치 한 행의 슬라이드 본문이다 — 자식은 컨테이너의 첫 자식(s0.0.0)이다.
func placementBody(tc layoutCase) string {
	if tc.tag == soml.GridElement {
		return fmt.Sprintf(`<grid cols="3"><p %s>자식</p></grid>`, tc.decl)
	}
	return fmt.Sprintf("<%s><p %s>자식</p></%s>", tc.tag, tc.decl, tc.tag)
}

// openTagOf는 식별자로 요소의 여는 태그를 잘라낸다 — 인라인 style을 한 문자열로 보고
// 선언의 **순서**까지 단정할 수 있다 (탈출구가 맨 끝인지가 순서의 문제다).
func openTagOf(t *testing.T, out, id string) string {
	t.Helper()
	marker := fmt.Sprintf("%s=%q", IDAttr, id)
	at := strings.Index(out, marker)
	if at < 0 {
		t.Fatalf("식별자 %s를 가진 요소가 없다:\n%s", id, out)
	}
	start := strings.LastIndex(out[:at], "<")
	end := strings.Index(out[at:], ">")
	if start < 0 || end < 0 {
		t.Fatalf("식별자 %s의 여는 태그를 자를 수 없다", id)
	}
	return out[start : at+end+1]
}

// assertCoversCatalog는 표 행의 속성 이름이 어휘 목록과 정확히 같은 집합인지 본다 —
// 빠진 행도, 카탈로그에 없는 행도 어긋난 상태다.
func assertCoversCatalog(t *testing.T, what string, covered, catalog []string) {
	t.Helper()
	for _, attr := range catalog {
		if !slices.Contains(covered, attr) {
			t.Errorf("%s %q의 CSS 매핑 검증이 없다 (PRD 표의 행이 빠졌다)", what, attr)
		}
	}
	for _, attr := range covered {
		if !slices.Contains(catalog, attr) {
			t.Errorf("%s %q는 카탈로그에 없다", what, attr)
		}
	}
}
