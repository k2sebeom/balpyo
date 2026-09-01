package web

import (
	"fmt"
	"strings"
	"testing"

	"github.com/k2sebeom/balpyo/core"
	"github.com/k2sebeom/balpyo/soml"
)

// 이 파일은 **PRD §7.4 표의 회귀 장치**다.
//
// "레이아웃 태그는 CSS flex/grid 시맨틱의 이름 바꾸기 층이다"는 검증 가능한 주장이므로,
// 표를 그대로 옮긴 표 기반 테스트가 그 주장을 지킨다. 그래서 아래 두 표(containerCases·
// placementCases)는 속성 **이름**을 함께 들고 다닌다 — 어휘 목록(soml.ContainerAttrs·
// ChildPlacementAttrs)을 순회해 "표에 있고 코드에 없는 행"이 남지 않았는지 확인하는
// 커버리지 테스트가 그 이름으로 대조한다 (TestLayoutVocabularyIsFullyMapped).

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
// 있고 요소는 클래스만 들고 나온다 — 그 경계가 이 iteration의 설계 결정이다 (layout.go).
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

// **표에 있고 코드에 없는 행이 남지 않는다**: 어휘 목록(soml)을 순회해 위의 두 표가 속성을
// 전수로 덮는지 본다. 속성이 하나 늘면 이 테스트가 먼저 빨개진다 — 매핑을 잊은 속성은
// "파싱은 되는데 CSS로 나가지 않는" 조용한 실패가 된다.
func TestLayoutVocabularyIsFullyMapped(t *testing.T) {
	covered := func(cases []layoutCase) map[string]bool {
		seen := map[string]bool{}
		for _, tc := range cases {
			seen[tc.attr] = true
		}
		return seen
	}
	container := covered(containerCases())
	// `<grid>`의 목록이 컨테이너 속성 전체다 (flex 공통 + 트랙 + 크기).
	for _, attr := range soml.ContainerAttrs(soml.GridElement) {
		if !container[attr] {
			t.Errorf("컨테이너 속성 %q의 CSS 매핑 검증이 없다 (PRD §7.4 표의 행이 빠졌다)", attr)
		}
	}
	placement := covered(placementCases())
	for _, attr := range soml.ChildPlacementAttrs() {
		if !placement[attr] {
			t.Errorf("자식 배치 속성 %q의 CSS 매핑 검증이 없다 (PRD §7.4 자식 표의 행이 빠졌다)", attr)
		}
	}
	// 어휘 전체를 순회했는지도 본다 — 표가 어휘보다 넓어지면(없는 속성을 검증하면) 그것도
	// 카탈로그와 어긋난 상태다.
	for attr := range container {
		if !containsString(soml.ContainerAttrs(soml.GridElement), attr) {
			t.Errorf("컨테이너 속성 %q는 카탈로그에 없다", attr)
		}
	}
	for attr := range placement {
		if !containsString(soml.ChildPlacementAttrs(), attr) {
			t.Errorf("자식 배치 속성 %q는 카탈로그에 없다", attr)
		}
	}
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

// 레이아웃을 쓰지 않은 덱에는 레이아웃 CSS도, 공간 토큰 변수도 나가지 않는다 — 골든 불변의
// 근거다 (listCSS를 조건부로 둔 것과 같은 태도). 이 계약이 깨지면 Iteration 1·2 골든 세 개가
// 이 iteration과 무관한 이유로 움직인다.
func TestLayoutCSSOnlyWhenLayoutUsed(t *testing.T) {
	plain := renderString(t, subsetDeck(t, "<p>레이아웃 없음</p>"))
	for _, absent := range []string{".balpyo-row", ".balpyo-col", ".balpyo-grid", ".balpyo-spacer", spaceVarPrefix} {
		if strings.Contains(plain, absent) {
			t.Errorf("레이아웃이 없는 덱의 산출물에 %q가 있다", absent)
		}
	}
	// `<spacer>` 하나만 있어도 레이아웃 어휘를 쓴 덱이다 (컨테이너 없이 세로 스택에서 밀어낸다).
	spacer := renderString(t, subsetDeck(t, "<p>위</p>\n  <spacer/>\n  <p>아래</p>"))
	for _, want := range []string{".balpyo-spacer { flex: 1; }", spaceVarPrefix + "md"} {
		if !strings.Contains(spacer, want) {
			t.Errorf("레이아웃 어휘를 쓴 덱의 산출물에 %q가 없다", want)
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

// 배치 속성이 없는 요소는 선언을 **하나도** 더하지 않는다 — 골든 불변의 단위 테스트 절반이다
// (나머지 절반은 픽스처 골든이 바이트로 확인한다).
func TestElementsWithoutPlacementGainNoDeclarations(t *testing.T) {
	out := renderString(t, subsetDeck(t, `
  <row>
    <p>블록</p>
    <ul><li>항목</li></ul>
    <spacer/>
  </row>`))
	// 속성 없는 컨테이너·spacer·리스트에는 style 속성 자체가 없다.
	for _, want := range []string{
		fmt.Sprintf(`<div class=%q %s="s0.0" %s="row">`, rowClass, IDAttr, ElAttr),
		fmt.Sprintf(`<ul class="balpyo-list" %s="s0.0.1" %s="ul">`, IDAttr, ElAttr),
		fmt.Sprintf(`<div class=%q %s="s0.0.2" %s="spacer"></div>`, spacerClass, IDAttr, ElAttr),
	} {
		if !strings.Contains(out, want) {
			t.Errorf("배치 없는 요소의 여는 태그 %q가 없다 (선언이 붙었다):\n%s", want, stageOf(t, out))
		}
	}
	// 블록의 style은 확정 외관으로 끝난다 — 배치 선언이 뒤에 붙지 않았다.
	if block := openTagOf(t, out, "s0.0.0"); !strings.HasSuffix(block, `text-align: start">`) {
		t.Errorf("배치 없는 블록의 style에 선언이 더 붙었다: %s", block)
	}
}

// 중첩 레이아웃은 이 iteration의 핵심 장면이다 — 식별자는 덱 트리의 슬라이스 인덱스 경로이고
// (ElementID), 그 대칭이 지오메트리 패스의 자리 맞대기를 성립시킨다.
func TestNestedLayoutIdentity(t *testing.T) {
	out := renderString(t, subsetDeck(t, `
  <col gap="md">
    <row gap="sm">
      <p>중첩</p>
    </row>
  </col>`))
	for _, tc := range []struct{ id, el string }{
		{"s0.0", "col"}, {"s0.0.0", "row"}, {"s0.0.0.0", "p"},
	} {
		open := openTagOf(t, out, tc.id)
		if want := fmt.Sprintf("%s=%q", ElAttr, tc.el); !strings.Contains(open, want) {
			t.Errorf("%s의 여는 태그에 %s가 없다: %s", tc.id, want, open)
		}
	}
	// 컨테이너도 측정 대상이다 (측정 대상 계약이 넓어졌다 — 패키지 주석). 식별자 개수가
	// 요소 개수와 같아야 geom.Refs와 자리가 맞는다.
	if got, want := strings.Count(out, IDAttr+"="), 3; got != want {
		t.Errorf("식별자 개수 = %d, want %d (컨테이너·spacer도 잰다)", got, want)
	}
}

// 해석되지 않은 배치를 기본값으로 그리면 웹만 그럴듯하게 나오고, 그 좌표가 PPTX까지 간다.
func TestRenderRejectsUnresolvedContainer(t *testing.T) {
	deck := subsetDeck(t, `<row gap="md"><p>본문</p></row>`)
	row := deck.Slides[0].Content[0].(*soml.Element)
	row.Appearance.Container = nil
	if _, err := New().Render(deck); err == nil {
		t.Error("배치가 해석되지 않은 컨테이너는 에러여야 한다")
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
	} {
		t.Run(tc.name, func(t *testing.T) {
			deck := subsetDeck(t, `<row gap="md"><p>본문</p></row>`)
			tc.patch(deck.Slides[0].Content[0].(*soml.Element))
			if _, err := New().Render(deck); err == nil {
				t.Errorf("어휘 밖 %s는 에러여야 한다", tc.name)
			}
		})
	}
}

func TestLayoutRenderIsDeterministic(t *testing.T) {
	body := `<grid cols="3" gap="lg" pad="md sm"><p span="2" order="-1">격자</p><spacer grow="1"/></grid>`
	a, b := renderString(t, subsetDeck(t, body)), renderString(t, subsetDeck(t, body))
	if a != b {
		t.Error("레이아웃 렌더 결과가 실행마다 다르다")
	}
}

// `<stack>`은 단일 셀 grid다 — 모든 자식이 `grid-area: 1/1`에 겹치고 stack의 크기가 가장 큰
// 자식에 맞춰 자동 결정된다 (PRD §13-3). `grid-template-*`은 내지 않는다: 암시 트랙 하나가
// 정확히 필요한 것이고, 명시하면 `cols`를 닫아 둔 항목 01의 결정과 두 벌이 된다.
func TestStackCompilesToSingleCellGridOverlap(t *testing.T) {
	out := renderString(t, subsetDeck(t, `<stack><p>겹침</p></stack>`))
	// 컨테이너는 stack 흐름의 엔진 CSS 클래스를 든다 (containerClass의 FlowStack 분기).
	open := openTagOf(t, out, "s0.0")
	if want := fmt.Sprintf("class=%q", stackClass); !strings.Contains(open, want) {
		t.Errorf("<stack>의 여는 태그에 %s가 없다: %s", want, open)
	}
	css := between(t, out, "<style>", "</style>")
	for _, want := range []string{
		".balpyo-stack { display: grid;",
		"grid-area: 1 / 1;",
	} {
		if !strings.Contains(css, want) {
			t.Errorf("stack 엔진 CSS에 %q가 없다", want)
		}
	}
	// 겹침을 absolute 전면 방식으로 하지 않는다 (컨테이너 크기가 소실된다) — grid-area가
	// 그 근거다. grid-template-*을 내면 암시 트랙 하나라는 계약이 깨진다 (`cols`를 닫아 둔
	// 항목 01의 결정과 두 벌이 된다).
	stackRule := between(t, css, ".balpyo-stack {", "}")
	for _, absent := range []string{"grid-template-columns", "grid-template-rows"} {
		if strings.Contains(stackRule, absent) {
			t.Errorf("stack 규칙에 %q가 있다 — 암시 트랙 하나여야 한다: %s", absent, stackRule)
		}
	}
}

// stack의 모든 자식은 위치 지정 요소다 (`position: relative`) — 칠 순서를 문서 순서로
// 되돌리는 계약이다 (PRD §13-3, 항목 04의 4번 결정). CSS는 위치 지정된 형제를 문서 순서로
// 칠하므로, x/y 자식(absolute)과 정적 자식이 섞여도 겹침 순서가 PPTX(문서 순서)와 갈라지지
// 않는다. 이 한 줄이 없으면 절대 배치 자식이 정적 자식보다 뒤에 칠해져 순서가 뒤집힌다.
func TestStackChildrenArePositionedForPaintOrder(t *testing.T) {
	css := between(t, renderString(t, subsetDeck(t, `<stack><p>겹침</p></stack>`)), "<style>", "</style>")
	// 겹침·칠 순서 규칙이 stack 자식 전체(`> *`)에 걸린다.
	if !strings.Contains(css, "position: relative;") {
		t.Error("stack 자식에 position: relative(칠 순서 계약)가 없다")
	}
}

// stack 자식의 텍스트·리스트는 확정 폭(width: 100%)에서 벗어나 auto로 돌아간다 — 그래야
// `at`의 justify-self가 칸 안에서 자식을 움직인다 (폭이 칸을 다 먹으면 앵커가 아무것도 하지
// 않는다). **기존 컨테이너 자식 초기화 선택자에 stack을 끼워 넣지 않고 전용 규칙을 낸다** —
// 그 선택자는 feat.layout이 켜진 모든 덱에 나가므로(layoutCSS) 끼워 넣으면 골든이 움직인다.
func TestStackChildResetsWidthInSeparateRule(t *testing.T) {
	css := between(t, renderString(t, subsetDeck(t, `<stack><p>겹침</p></stack>`)), "<style>", "</style>")
	if !strings.Contains(css, ".balpyo-stack") || !strings.Contains(css, "width: auto;") {
		t.Errorf("stack 자식의 width: auto 초기화가 없다:\n%s", css)
	}
	// 기존 row/col/grid 초기화 선택자에 stack이 섞이지 않았다 — 섞이면 그 규칙이 나가는
	// 모든 골든이 움직인다 (03_layout·04_shape·05_image).
	base := ":is(.balpyo-row, .balpyo-col, .balpyo-grid) > :is(.balpyo-text, .balpyo-list) {"
	if idx := strings.Index(css, base); idx >= 0 {
		line := css[idx:]
		line = line[:strings.Index(line, "{")]
		if strings.Contains(line, "stack") {
			t.Errorf("기존 컨테이너 자식 초기화 선택자에 stack이 끼어들었다: %s", line)
		}
	}
}

// `x`/`y`는 그 자식만 absolute로 전환하고 좌표를 left/top으로 낸다 — **transform:translate을
// 쓰지 않는다.** 측정 뷰가 변형을 끄므로(ADR 0021) translate로 위치를 내면 잰 rect에 위치가
// 없고 PPTX가 겹치지 않은 좌표를 받는다. 다음 사람이 translate가 더 짧다고 바꾸는 것을
// 막는 회귀다.
func TestAbsolutePlacementUsesLeftTopNotTransform(t *testing.T) {
	open := openTagOf(t, renderString(t, subsetDeck(t, `<stack><p x="20" y="30">배지</p></stack>`)), "s0.0.0")
	for _, want := range []string{"position: absolute", "left: 20px", "top: 30px"} {
		if !strings.Contains(open, want) {
			t.Errorf("x/y 자식의 style에 %q가 없다: %s", want, open)
		}
	}
	for _, forbidden := range []string{"transform", "translate"} {
		if strings.Contains(open, forbidden) {
			t.Errorf("x/y 배치에 %q가 나갔다 — 측정 뷰가 변형을 끈다: %s", forbidden, open)
		}
	}
}

// 지정하지 않은 축은 선언을 내지 않는다 (`top: auto` = 정적 위치) — "정하지 않은 것에
// 기본값을 내지 않는다"는 containerCSS의 규칙 그대로다.
func TestAbsolutePlacementOmitsUnsetAxis(t *testing.T) {
	xOnly := openTagOf(t, renderString(t, subsetDeck(t, `<stack><p x="20">가로만</p></stack>`)), "s0.0.0")
	if !strings.Contains(xOnly, "left: 20px") {
		t.Errorf("x만 준 자식에 left가 없다: %s", xOnly)
	}
	if strings.Contains(xOnly, "top:") {
		t.Errorf("x만 준 자식에 top이 나갔다 (지정하지 않은 축): %s", xOnly)
	}
	yOnly := openTagOf(t, renderString(t, subsetDeck(t, `<stack><p y="30">세로만</p></stack>`)), "s0.0.0")
	if !strings.Contains(yOnly, "top: 30px") {
		t.Errorf("y만 준 자식에 top이 없다: %s", yOnly)
	}
	if strings.Contains(yOnly, "left:") {
		t.Errorf("y만 준 자식에 left가 나갔다 (지정하지 않은 축): %s", yOnly)
	}
	// 둘 다 있으면 position: absolute는 한 번만 나간다.
	both := openTagOf(t, renderString(t, subsetDeck(t, `<stack><p x="20" y="30">둘</p></stack>`)), "s0.0.0")
	if got := strings.Count(both, "position: absolute"); got != 1 {
		t.Errorf("position: absolute가 %d번 나온다, want 1: %s", got, both)
	}
}

// `<stack>` 안 stack, `<grid>` 안 stack — 중첩과 자식 배치가 함께 붙은 경우다. 식별자는
// 슬라이스 인덱스 경로이고 각 stack이 자기 CSS를 든다.
func TestNestedAndGridStack(t *testing.T) {
	out := renderString(t, subsetDeck(t, `
  <stack>
    <stack><p at="center">안쪽</p></stack>
  </stack>
  <grid cols="2">
    <stack><p at="bottom-right">셀 안 stack</p></stack>
  </grid>`))
	for _, tc := range []struct{ id, el string }{
		{"s0.0", "stack"}, {"s0.0.0", "stack"}, {"s0.0.0.0", "p"},
		{"s0.1", "grid"}, {"s0.1.0", "stack"}, {"s0.1.0.0", "p"},
	} {
		open := openTagOf(t, out, tc.id)
		if want := fmt.Sprintf("%s=%q", ElAttr, tc.el); !strings.Contains(open, want) {
			t.Errorf("%s의 여는 태그에 %s가 없다: %s", tc.id, want, open)
		}
	}
	// 셀 안 stack 자식의 앵커가 place-self로 나간다.
	if inner := openTagOf(t, out, "s0.1.0.0"); !strings.Contains(inner, "place-self: end end") {
		t.Errorf("grid>stack 자식의 앵커가 place-self로 나오지 않았다: %s", inner)
	}
}

// **stack·canvas를 쓰지 않은 덱에는 stack CSS가 나가지 않는다** — 골든 불변의 근거다
// (listCSS·layoutCSS를 조건부로 둔 것과 같은 태도). 이 계약이 깨지면 01_text~05_image
// 골든이 이 iteration과 무관한 이유로 움직인다.
func TestStackCSSOnlyWhenStackOrCanvasUsed(t *testing.T) {
	// 레이아웃을 쓰지만 stack은 아닌 덱 (row).
	row := renderString(t, subsetDeck(t, `<row gap="md"><p>자식</p></row>`))
	for _, absent := range []string{".balpyo-stack", "balpyo-stage-canvas", "grid-area: 1 / 1"} {
		if strings.Contains(row, absent) {
			t.Errorf("stack이 없는 덱의 산출물에 %q가 있다", absent)
		}
	}
	// stack을 쓴 덱.
	stack := renderString(t, subsetDeck(t, `<stack><p>겹침</p></stack>`))
	if !strings.Contains(stack, ".balpyo-stack { display: grid;") {
		t.Error("stack을 쓴 덱에 stack CSS가 없다")
	}
}

// **캔버스 스테이지는 stack이다** (항목 04의 5번 결정) — 스테이지가 단일 셀 grid가 되어
// 자식이 `x`/`y`·`at`으로 자기 자리를 정한다. 겹침 구현이 한 벌로 유지되도록 stack과 같은
// CSS 절을 공유한다.
func TestCanvasStageIsStack(t *testing.T) {
	out := renderString(t, canvasDeck(t, `<text x="0" y="0">모서리</text>`))
	// 스테이지가 캔버스 표기를 든다 (기존 balpyo-stage에 더해).
	if !strings.Contains(out, `<div class="balpyo-stage balpyo-stage-canvas">`) {
		t.Errorf("캔버스 스테이지에 balpyo-stage-canvas 클래스가 없다:\n%s", out)
	}
	css := between(t, out, "<style>", "</style>")
	for _, want := range []string{
		".balpyo-stage-canvas { display: grid;",
		"grid-area: 1 / 1;",
	} {
		if !strings.Contains(css, want) {
			t.Errorf("캔버스 스테이지 CSS에 %q가 없다", want)
		}
	}
	// 캔버스 직속 자식의 x/y가 absolute로 나간다.
	if open := openTagOf(t, out, "s0.0"); !strings.Contains(open, "position: absolute") || !strings.Contains(open, "left: 0px") {
		t.Errorf("캔버스 자식의 x/y가 절대 배치로 나오지 않았다: %s", open)
	}
}

// 기본(content) 슬라이드의 스테이지는 캔버스 표기를 갖지 않는다 — content와 canvas의
// 산출물이 갈라지는 유일한 자리가 이 클래스다.
func TestContentStageIsNotCanvas(t *testing.T) {
	out := renderString(t, subsetDeck(t, `<p>세로 스택</p>`))
	if strings.Contains(out, "balpyo-stage-canvas") {
		t.Errorf("content 슬라이드의 스테이지에 캔버스 표기가 나왔다:\n%s", stageOf(t, out))
	}
}

// canvasDeck은 layout="canvas" 슬라이드를 파이프라인처럼(layout 값·구간을 Slide로 옮기고
// 해석) 만든 덱이다 — subsetDeck은 `<slide id="subset">`만 내므로 layout을 실을 수 없다
// (core/slide_test.go resolveLayoutSlide와 같은 경로).
func canvasDeck(t *testing.T, body string) *soml.Deck {
	t.Helper()
	const source = "slides/001_canvas.slide"
	doc, diags := soml.Parse(source, `<slide id="canvas" layout="canvas">`+"\n"+body+"\n</slide>\n")
	if err := diags.Err(); err != nil {
		t.Fatalf("파싱 실패: %v", err)
	}
	stage, err := soml.Aspect16x9.Stage()
	if err != nil {
		t.Fatal(err)
	}
	deck := &soml.Deck{
		Title: "캔버스", Lang: "ko", Aspect: soml.Aspect16x9, Stage: stage,
		Slides: []*soml.Slide{{
			Index: 0, ID: "canvas", Source: source,
			Layout: soml.SlideLayout(doc.Layout), LayoutSpan: doc.LayoutSpan,
			Notes: doc.Notes, Content: doc.Content,
		}},
	}
	if err := core.Resolve(deck).Err(); err != nil {
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

// containsString은 목록 검사다 (커버리지 대조 — 순서 있는 어휘 목록에는 맵이 없다).
func containsString(names []string, name string) bool {
	for _, candidate := range names {
		if candidate == name {
			return true
		}
	}
	return false
}
