package core

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/k2sebeom/balpyo/soml"
)

// layoutSlide는 콘텐츠 한 줄을 담은 슬라이드다 — 콘텐츠는 항상 2행 3열에서 시작한다.
func layoutSlide(content string) string {
	return "<slide id=\"layout\">\n  " + content + "\n</slide>\n"
}

// canvasSlide는 layoutSlide와 같은 들여쓰기의 layout="canvas" 슬라이드다.
func canvasSlide(content string) string {
	return "<slide id=\"canvas\" layout=\"canvas\">\n  " + content + "\n</slide>\n"
}

// containerSrc는 컨테이너 하나와 자식 하나로 이루어진 슬라이드다.
func containerSrc(tag, attrs string) string {
	if attrs != "" {
		attrs = " " + attrs
	}
	return layoutSlide("<" + tag + attrs + "><p>내용</p></" + tag + ">")
}

// childSrc는 컨테이너 안의 `<p>`에 자식 배치 속성을 붙인 슬라이드다.
func childSrc(parent, attrs string) string {
	return layoutSlide("<" + parent + "><p " + attrs + ">내용</p></" + parent + ">")
}

// resolvedElement는 진단 없이 해석된 덱에서 tag인 첫 요소다.
func resolvedElement(t *testing.T, src, tag string) *soml.Element {
	t.Helper()
	deck, diags := resolveSlide(t, src)
	if len(diags) != 0 {
		t.Fatalf("진단이 없어야 한다: %v", diags)
	}
	return findElement(t, deck, tag)
}

// show는 실패 메시지용 문자열이다 — 포인터 필드가 많아 %+v로는 주소만 보인다.
func show(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return err.Error()
	}
	return string(b)
}

// 컨테이너 속성 표(docs/reference/soml/layout.md)의 값 문법 전수 — 속성 → 확정 배치 필드.
// 렌더러는 Attrs를 다시 읽지 않으므로 이 표가 곧 CSS의 입력이다.
func TestResolveContainerLayout(t *testing.T) {
	cases := []struct {
		name string
		attr string
		tag  string
		decl string
		want soml.ContainerLayout
	}{
		// 공간 토큰은 이름과 값을 함께 담는다 (웹이 var() 참조로 낸다).
		{"gap 수치", "gap", "row", `gap="16"`, soml.ContainerLayout{Gap: ptr(soml.Px(16))}},
		{"gap 공간 토큰", "gap", "row", `gap="xl"`, soml.ContainerLayout{Gap: ptr(soml.SpaceLength("xl", 48))}},
		{"gap 백분율", "gap", "row", `gap="10%"`, soml.ContainerLayout{Gap: ptr(soml.Percent(10))}},
		// 0은 저자가 정한 값이다 — "정하지 않음"(nil)과 구별된다.
		{"gap 0", "gap", "row", `gap="0"`, soml.ContainerLayout{Gap: ptr(soml.Px(0))}},

		{"align", "align", "col", `align="stretch"`,
			soml.ContainerLayout{Flow: soml.FlowColumn, Align: soml.BoxAlignStretch}},
		{"justify", "justify", "row", `justify="between"`,
			soml.ContainerLayout{Justify: soml.BoxJustifyBetween}},

		{"wrap 값 없음", "wrap", "row", `wrap`, soml.ContainerLayout{Wrap: true}},
		{"wrap true", "wrap", "row", `wrap="true"`, soml.ContainerLayout{Wrap: true}},

		// pad 1~4값은 CSS padding 축약 규칙으로 4변까지 풀린다.
		{"pad 1값", "pad", "row", `pad="lg"`, soml.ContainerLayout{Box: soml.Box{Pad: &soml.Pad{
			Top: soml.SpaceLength("lg", 36), Right: soml.SpaceLength("lg", 36),
			Bottom: soml.SpaceLength("lg", 36), Left: soml.SpaceLength("lg", 36)}}}},
		{"pad 2값", "pad", "row", `pad="sm md"`, soml.ContainerLayout{Box: soml.Box{Pad: &soml.Pad{
			Top: soml.SpaceLength("sm", 12), Right: soml.SpaceLength("md", 24),
			Bottom: soml.SpaceLength("sm", 12), Left: soml.SpaceLength("md", 24)}}}},
		{"pad 3값", "pad", "row", `pad="4 8 12"`, soml.ContainerLayout{Box: soml.Box{Pad: &soml.Pad{
			Top: soml.Px(4), Right: soml.Px(8), Bottom: soml.Px(12), Left: soml.Px(8)}}}},
		{"pad 4값", "pad", "row", `pad="1 2 3 4%"`, soml.ContainerLayout{Box: soml.Box{Pad: &soml.Pad{
			Top: soml.Px(1), Right: soml.Px(2), Bottom: soml.Px(3), Left: soml.Percent(4)}}}},
		// 숫자로 시작하는 토큰(2xl)도 토큰이다 — 수치 판정을 먼저 하면 "단위를 떼라"가 된다.
		{"pad 2xl", "pad", "row", `pad="2xl"`, soml.ContainerLayout{Box: soml.Box{Pad: &soml.Pad{
			Top: soml.SpaceLength("2xl", 72), Right: soml.SpaceLength("2xl", 72),
			Bottom: soml.SpaceLength("2xl", 72), Left: soml.SpaceLength("2xl", 72)}}}},

		// 트랙 — 수치 하나는 개수(repeat), 그 밖은 목록이다.
		{"cols 개수", "cols", "grid", `cols="4"`,
			soml.ContainerLayout{Flow: soml.FlowGrid, Cols: &soml.Tracks{Count: 4}}},
		{"cols 트랙 목록", "cols", "grid", `cols="2fr 1fr"`,
			soml.ContainerLayout{Flow: soml.FlowGrid, Cols: &soml.Tracks{List: []soml.Track{
				{Kind: soml.TrackFr, Value: 2}, {Kind: soml.TrackFr, Value: 1}}}}},
		{"cols 혼합 트랙", "cols", "grid", `cols="200 auto 25% 1fr"`,
			soml.ContainerLayout{Flow: soml.FlowGrid, Cols: &soml.Tracks{List: []soml.Track{
				{Kind: soml.TrackPx, Value: 200}, {Kind: soml.TrackAuto},
				{Kind: soml.TrackPercent, Value: 25}, {Kind: soml.TrackFr, Value: 1}}}}},
		{"rows 트랙 목록", "rows", "grid", `rows="auto 1fr"`,
			soml.ContainerLayout{Flow: soml.FlowGrid, Rows: &soml.Tracks{List: []soml.Track{
				{Kind: soml.TrackAuto}, {Kind: soml.TrackFr, Value: 1}}}}},

		// 크기 — `fill`은 해당 축의 100%로 눕는다.
		{"w 수치", "w", "row", `w="320"`, soml.ContainerLayout{Box: soml.Box{Width: ptr(soml.Px(320))}}},
		{"w 백분율", "w", "row", `w="40%"`, soml.ContainerLayout{Box: soml.Box{Width: ptr(soml.Percent(40))}}},
		{"w fill", "w", "row", `w="fill"`, soml.ContainerLayout{Box: soml.Box{Width: ptr(soml.Percent(100))}}},
		{"h fill", "h", "row", `h="fill"`, soml.ContainerLayout{Box: soml.Box{Height: ptr(soml.Percent(100))}}},
		{"min-w", "min-w", "row", `min-w="200"`, soml.ContainerLayout{Box: soml.Box{MinWidth: ptr(soml.Px(200))}}},
		{"min-h", "min-h", "row", `min-h="120"`, soml.ContainerLayout{Box: soml.Box{MinHeight: ptr(soml.Px(120))}}},
		{"max-w", "max-w", "row", `max-w="360"`, soml.ContainerLayout{Box: soml.Box{MaxWidth: ptr(soml.Px(360))}}},
		{"max-h", "max-h", "row", `max-h="220"`, soml.ContainerLayout{Box: soml.Box{MaxHeight: ptr(soml.Px(220))}}},
		{"ratio", "ratio", "row", `ratio="16:9"`, soml.ContainerLayout{Box: soml.Box{Ratio: &soml.Ratio{W: 16, H: 9}}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			elem := resolvedElement(t, containerSrc(tc.tag, tc.decl), tc.tag)
			got := elem.Container()
			if got == nil {
				t.Fatalf("<%s>에 확정 컨테이너 배치가 없다", tc.tag)
			}
			if !reflect.DeepEqual(*got, tc.want) {
				t.Errorf("컨테이너 배치 = %s, want %s", show(*got), show(tc.want))
			}
			if elem.Placement() != nil {
				t.Errorf("자식 배치가 붙었다: %+v", *elem.Placement())
			}
		})
	}

	// 카탈로그가 연 속성을 값 층이 모르면 저자가 쓴 배치가 조용히 사라진다.
	covered := map[string]bool{}
	for _, tc := range cases {
		covered[tc.attr] = true
	}
	for _, attr := range soml.ContainerAttrs(soml.GridElement) {
		if !covered[attr] {
			t.Errorf("컨테이너 속성 %q의 해석 테스트가 없다", attr)
		}
	}
}

// 자식 속성 표(docs/reference/soml/layout.md). span은 grid, at은 stack, x/y는 stack 또는
// canvas 직속에서만 뜻이 있다.
func TestResolvePlacement(t *testing.T) {
	cases := []struct {
		name   string
		attr   string
		parent string
		decl   string
		want   soml.Placement
	}{
		{"grow", "grow", "row", `grow="2"`, soml.Placement{Grow: ptr(2)}},
		{"shrink 0", "shrink", "row", `shrink="0"`, soml.Placement{Shrink: ptr(0)}},
		{"basis 수치", "basis", "row", `basis="240"`, soml.Placement{Basis: ptr(soml.Px(240))}},
		{"basis 백분율", "basis", "row", `basis="50%"`, soml.Placement{Basis: ptr(soml.Percent(50))}},
		{"span", "span", "grid", `span="2"`, soml.Placement{Span: ptr(2)}},
		{"self", "self", "row", `self="center"`, soml.Placement{Self: soml.BoxAlignCenter}},
		// order만 음수를 받는다 — "먼저 보이게"의 CSS 관용 표기다.
		{"order 음수", "order", "row", `order="-1"`, soml.Placement{Order: ptr(-1)}},
		{"at 앵커", "at", "stack", `at="center"`, soml.Placement{At: soml.StackCenter}},
		{"x 수치", "x", "stack", `x="40"`, soml.Placement{X: ptr(soml.Px(40))}},
		{"y 백분율", "y", "stack", `y="50%"`, soml.Placement{Y: ptr(soml.Percent(50))}},
		{"style", soml.StyleAttr, "row", `style="place-self: end"`,
			soml.Placement{Style: "place-self: end"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			elem := resolvedElement(t, childSrc(tc.parent, tc.decl), "p")
			got := elem.Placement()
			if got == nil {
				t.Fatal("<p>에 확정 자식 배치가 없다")
			}
			if !reflect.DeepEqual(*got, tc.want) {
				t.Errorf("자식 배치 = %s, want %s", show(*got), show(tc.want))
			}
			if elem.Container() != nil {
				t.Errorf("컨테이너 배치가 붙었다: %+v", *elem.Container())
			}
		})
	}

	covered := map[string]bool{}
	for _, tc := range cases {
		covered[tc.attr] = true
	}
	for _, attr := range soml.ChildPlacementAttrs() {
		if !covered[attr] {
			t.Errorf("자식 배치 속성 %q의 해석 테스트가 없다", attr)
		}
	}

	// canvas 슬라이드의 직속 자식은 부모 요소 없이도 x/y를 받는다.
	if p := resolvedElement(t, canvasSlide(`<p x="40">내용</p>`), "p").Placement(); p == nil ||
		p.X == nil || *p.X != soml.Px(40) {
		t.Fatalf("canvas 직속 x가 해석되지 않았다: %+v", p)
	}
}

// 자식 배치 속성은 요소 종류와 무관하게 같은 코드가 해석한다.
func TestResolvePlacementOnEveryKind(t *testing.T) {
	cases := []struct{ name, content, tag string }{
		{"text", `<row><text grow="1">내용</text></row>`, soml.TextElement},
		{"블록", `<row><p grow="1">내용</p></row>`, "p"},
		{"리스트 컨테이너", `<row><ul grow="1"><li>항목</li></ul></row>`, "ul"},
		{"중첩 컨테이너", `<row><col grow="1"><p>내용</p></col></row>`, soml.ColElement},
		{"spacer", `<row><spacer grow="1"/></row>`, soml.SpacerElement},
		{"슬라이드 직속", `<p grow="1">내용</p>`, "p"},
	}
	for _, tc := range cases {
		placement := resolvedElement(t, layoutSlide(tc.content), tc.tag).Placement()
		if placement == nil || placement.Grow == nil || *placement.Grow != 1 {
			t.Errorf("%s: <%s>의 grow가 해석되지 않았다: %+v", tc.name, tc.tag, placement)
		}
	}
}

// 배치와 텍스트 외관은 서로를 모르는 두 채널이다 — 한쪽이 다른 쪽을 지우면 안 된다.
func TestResolvePlacementAndTextTogether(t *testing.T) {
	elem := resolvedElement(t, layoutSlide(`<row><p grow="1" size="30" color="muted">내용</p></row>`), "p")
	placement := elem.Placement()
	if placement == nil || placement.Grow == nil || *placement.Grow != 1 {
		t.Fatalf("배치가 해석되지 않았다: %+v", placement)
	}
	style := elem.TextStyle()
	if style == nil || style.Size != 30 || style.Color != soml.TokenColor("muted") || style.Role != soml.RoleBody {
		t.Fatalf("외관 = %+v, want body + size 30 + muted", style)
	}
}

// 배치 속성이 없는 요소는 배치 필드가 nil이다 ("정하지 않음"과 0의 구별). 레이아웃 컨테이너는
// 속성이 없어도 태그가 정한 흐름을 Container로 갖고, <spacer>는 아무 배치도 갖지 않는다.
func TestResolveLayoutDefaults(t *testing.T) {
	deck, diags := resolveSlide(t, layoutSlide(`<p>내용</p><ul><li>항목</li></ul><text>내용</text>`))
	if len(diags) != 0 {
		t.Fatalf("진단이 없어야 한다: %v", diags)
	}
	for _, tag := range []string{"p", "li", soml.TextElement} {
		elem := findElement(t, deck, tag)
		if elem.Container() != nil || elem.Placement() != nil {
			t.Errorf("<%s>에 배치가 붙었다: %+v", tag, *elem.Appearance)
		}
	}
	if list := findElement(t, deck, "ul"); list.Appearance != nil {
		t.Errorf("<ul>에 Appearance가 생겼다: %+v", *list.Appearance)
	}
	for tag, flow := range map[string]soml.ContainerFlow{
		soml.RowElement: soml.FlowRow, soml.ColElement: soml.FlowColumn, soml.GridElement: soml.FlowGrid,
	} {
		container := resolvedElement(t, containerSrc(tag, ""), tag).Container()
		if container == nil || !reflect.DeepEqual(*container, soml.ContainerLayout{Flow: flow}) {
			t.Errorf("속성 없는 <%s> = %s, want 흐름만", tag, show(container))
		}
	}
	if elem := resolvedElement(t, layoutSlide("<spacer/>"), soml.SpacerElement); elem.Appearance != nil {
		t.Errorf("<spacer>에 배치가 붙었다: %+v", *elem.Appearance)
	}
}

// 닫힌 어휘 전체가 실제로 해석된다 — 목록에 값을 더하고 해석을 잊으면 빨갛게 된다. self는
// align과 같은 어휘를 쓴다.
func TestResolveLayoutVocabularies(t *testing.T) {
	for _, align := range soml.BoxAligns() {
		container := resolvedElement(t, containerSrc("row", `align="`+string(align)+`"`), "row").Container()
		if container == nil || container.Align != align {
			t.Errorf("align = %+v, want %q", container, align)
		}
		placement := resolvedElement(t, childSrc("row", `self="`+string(align)+`"`), "p").Placement()
		if placement == nil || placement.Self != align {
			t.Errorf("self = %+v, want %q", placement, align)
		}
	}
	for _, justify := range soml.BoxJustifies() {
		container := resolvedElement(t, containerSrc("row", `justify="`+string(justify)+`"`), "row").Container()
		if container == nil || container.Justify != justify {
			t.Errorf("justify = %+v, want %q", container, justify)
		}
	}
	for _, anchor := range soml.StackAnchors() {
		elem := resolvedElement(t, childSrc("stack", `at="`+string(anchor)+`"`), "p")
		if p := elem.Placement(); p == nil || p.At != anchor || elem.Container() != nil {
			t.Errorf("at = %+v, want %q", p, anchor)
		}
	}
}

// x/y의 값 문법 — 수치·%·음수·소수. 정하지 않은 축은 nil이다.
func TestResolveCoordSyntax(t *testing.T) {
	cases := []struct {
		name  string
		decl  string
		wantX *soml.Length
		wantY *soml.Length
	}{
		{"x 수치", `x="120"`, ptr(soml.Px(120)), nil},
		{"y 수치", `y="80"`, nil, ptr(soml.Px(80))},
		{"x 백분율", `x="50%"`, ptr(soml.Percent(50)), nil},
		{"x 음수", `x="-16"`, ptr(soml.Px(-16)), nil},
		{"y 음수 백분율", `y="-12.5%"`, nil, ptr(soml.Percent(-12.5))},
		{"x 소수", `x="12.5"`, ptr(soml.Px(12.5)), nil},
		{"x y 함께", `x="10" y="-20"`, ptr(soml.Px(10)), ptr(soml.Px(-20))},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			placement := resolvedElement(t, childSrc("stack", tc.decl), "p").Placement()
			if placement == nil {
				t.Fatal("<p>에 확정 배치가 없다")
			}
			if !reflect.DeepEqual(placement.X, tc.wantX) || !reflect.DeepEqual(placement.Y, tc.wantY) {
				t.Errorf("x, y = %s, %s, want %s, %s", show(placement.X), show(placement.Y), show(tc.wantX), show(tc.wantY))
			}
		})
	}
}

// 공간 토큰과 수치는 같은 타입에 구별된 상태로 도달한다 — px로만 눕히면 테마 리듬을 바꿨을 때
// `gap="md"`만 따라 움직여야 하는 구별이 사라진다.
func TestResolveSpaceTokenBoundary(t *testing.T) {
	tokenGap := resolvedElement(t, containerSrc("row", `gap="md"`), "row").Container().Gap
	numberGap := resolvedElement(t, containerSrc("row", `gap="16"`), "row").Container().Gap
	if tokenGap == nil || *tokenGap != soml.SpaceLength("md", 24) {
		t.Errorf("gap=md → %s, want md(24px) 토큰 참조", show(tokenGap))
	}
	if numberGap == nil || *numberGap != soml.Px(16) || numberGap.IsToken() {
		t.Errorf("gap=16 → %s, want 토큰 아닌 16px", show(numberGap))
	}
}

// 공간 토큰 어휘는 6종이고(docs/reference/theme.md), 사다리는 단조 증가하는 6px(리듬의 최소 눈금) 배수다.
func TestThemeSpaceTokens(t *testing.T) {
	theme := minimalTheme()
	want := []string{"xs", "sm", "md", "lg", "xl", "2xl"}
	if got := theme.SpaceNames(); !reflect.DeepEqual(got, want) {
		t.Errorf("공간 토큰 = %v, want %v", got, want)
	}
	prev := 0.0
	for _, token := range theme.Spaces {
		if token.Px <= prev || token.Px != float64(int(token.Px)) || int(token.Px)%6 != 0 {
			t.Errorf("%s = %gpx (앞 토큰 %gpx) — 단조 증가하는 6px 배수여야 한다", token.Name, token.Px, prev)
		}
		prev = token.Px
	}
}

// 배치 진단은 문제가 된 값(목록이면 그 조각)을, 부모 한정 위반은 속성 이름을 line:col로
// 가리킨다. 무효한 at/x/y/span은 확정 배치에 실리지 않는다.
func TestResolveLayoutDiagnostics(t *testing.T) {
	l, c := layoutSlide, canvasSlide
	cases := []struct {
		name      string
		src       string
		line, col int
		contains  []string
	}{
		// 길이·공간 토큰
		{"모르는 공간 토큰", l(`<row gap="huge"><p>내용</p></row>`), 2, 13, []string{"알 수 없는 공간 토큰"}},
		{"단위 접미사", l(`<row gap="16px"><p>내용</p></row>`), 2, 13, []string{"단위 없는 논리 px"}},
		{"크기에 쓴 공간 토큰", l(`<row w="md"><p>내용</p></row>`), 2, 11, []string{"gap·pad에만"}},
		{"크기에 쓴 이름", l(`<row w="가득"><p>내용</p></row>`), 2, 11, []string{"수치가 아니다"}},
		{"gap에 쓴 fill", l(`<row gap="fill"><p>내용</p></row>`), 2, 13, []string{"알 수 없는 공간 토큰"}},
		// 음수는 크기·간격·basis에서 에러, 좌표에서만 통과한다 (lengthValue의 signed 축).
		{"음수 gap", l(`<row gap="-4"><p>내용</p></row>`), 2, 13, []string{"0보다 작을 수 없다"}},
		{"음수 폭", l(`<row w="-10"><p>내용</p></row>`), 2, 11, []string{"0보다 작을 수 없다"}},
		{"음수 basis", l(`<row><p basis="-1">내용</p></row>`), 2, 18, []string{"0보다 작을 수 없다"}},
		{"음수 min-h", l(`<row min-h="-5"><p>내용</p></row>`), 2, 15, []string{"0보다 작을 수 없다"}},
		// 닫힌 어휘
		{"오타 align", l(`<row align="centre"><p>내용</p></row>`), 2, 15, []string{"알 수 없는 align"}},
		{"CSS 접두 align", l(`<row align="flex-start"><p>내용</p></row>`), 2, 15, []string{`"start"`}},
		{"오타 justify", l(`<row justify="middle"><p>내용</p></row>`), 2, 17, []string{"알 수 없는 justify"}},
		{"CSS 접두 justify", l(`<row justify="space-between"><p>내용</p></row>`), 2, 17, []string{`"between"`}},
		{"오타 self", l(`<row><p self="middle">내용</p></row>`), 2, 17, []string{"알 수 없는 self"}},
		// 불리언
		{"wrap false", l(`<row wrap="false"><p>내용</p></row>`), 2, 14, []string{"끄려면 속성을 지운다"}},
		{"빈 wrap", l(`<row wrap=""><p>내용</p></row>`), 2, 14, []string{"비어 있다"}},
		// 목록
		{"pad 5값", l(`<row pad="1 2 3 4 5"><p>내용</p></row>`), 2, 13, []string{"1~4개 값"}},
		{"pad 둘째 값", l(`<row pad="sm huge"><p>내용</p></row>`), 2, 16, []string{"알 수 없는 공간 토큰"}},
		{"트랙 개수 0", l(`<grid cols="0"><p>내용</p></grid>`), 2, 15, []string{"1 이상 정수"}},
		{"모르는 트랙", l(`<grid cols="넷"><p>내용</p></grid>`), 2, 15, []string{"트랙 값"}},
		{"음수 트랙", l(`<grid cols="2fr -1fr"><p>내용</p></grid>`), 2, 19, []string{"0보다 작을 수 없다"}},
		// 비율
		{"콜론 없는 ratio", l(`<row ratio="16"><p>내용</p></row>`), 2, 15, []string{"W:H 정수 쌍"}},
		{"세 값 ratio", l(`<row ratio="16:9:2"><p>내용</p></row>`), 2, 15, []string{"W:H 정수 쌍"}},
		{"0 ratio", l(`<row ratio="0:9"><p>내용</p></row>`), 2, 15, []string{"0보다 커야"}},
		// 정수
		{"grow 소수", l(`<row><p grow="1.5">내용</p></row>`), 2, 17, []string{"정수다"}},
		{"grow 음수", l(`<row><p grow="-1">내용</p></row>`), 2, 17, []string{"0보다 작을 수 없다"}},
		{"order 비수치", l(`<row><p order="처음">내용</p></row>`), 2, 18, []string{"정수다"}},
		{"span 0", l(`<grid><p span="0">내용</p></grid>`), 2, 18, []string{"1 이상 정수"}},
		// 값 없음·빈 값
		{"값 없는 크기", l(`<row w><p>내용</p></row>`), 2, 8, []string{"값이 필요하다"}},
		{"빈 gap", l(`<row gap=""><p>내용</p></row>`), 2, 13, []string{"비어 있다"}},
		// at 어휘 — 고치는 방법(올바른 9지점)이 진단 안에 있다.
		{"at 짧은 이름", l(`<stack><p at="top">내용</p></stack>`), 2, 17, []string{"top-center"}},
		{"at 축 뒤바뀜", l(`<stack><p at="right-top">내용</p></stack>`), 2, 17, []string{"top-right"}},
		{"at 오타", l(`<stack><p at="centre">내용</p></stack>`), 2, 17, []string{"알 수 없는 at"}},
		{"at 어휘 밖", l(`<stack><p at="stretch">내용</p></stack>`), 2, 17, []string{"알 수 없는 at"}},
		{"center-center 거부", l(`<stack><p at="center-center">내용</p></stack>`), 2, 17, []string{"center"}},
		{"값 없는 at", l(`<stack><p at>내용</p></stack>`), 2, 13, []string{"값이 필요하다"}},
		// 좌표는 토큰·fill을 받지 않는다.
		{"x 공간 토큰", l(`<stack><p x="md">내용</p></stack>`), 2, 16, []string{"gap·pad에만"}},
		{"x fill", l(`<stack><p x="fill">내용</p></stack>`), 2, 16, []string{"수치가 아니다"}},
		{"x 단위 접미사", l(`<stack><p x="10px">내용</p></stack>`), 2, 16, []string{"단위 없는 논리 px"}},
		{"값 없는 y", l(`<stack><p y>내용</p></stack>`), 2, 13, []string{"값이 필요하다"}},
		// 부모 한정 — 진단은 속성 이름을 가리킨다. canvas의 <row> 자식이 경계의 요점이다
		// (캔버스 전체가 절대 배치 지대가 되면 안 된다).
		{"at: flex 자식", l(`<row><p at="center">내용</p></row>`), 2, 11, []string{"at은 <stack>"}},
		{"at: 슬라이드 직속", l(`<p at="center">내용</p>`), 2, 6, []string{"at은 <stack>"}},
		{"x: content 직속", l(`<p x="40">내용</p>`), 2, 6, []string{"canvas", soml.StackElement}},
		{"x: content의 row 자식", l(`<row><p x="40">내용</p></row>`), 2, 11, []string{"canvas", soml.StackElement}},
		{"x: canvas의 row 자식", c(`<row><p x="40">내용</p></row>`), 2, 11, []string{"canvas", soml.StackElement}},
		{"span: flex 자식", l(`<row><p span="2">내용</p></row>`), 2, 11, []string{"부모는 <row>다", "grid"}},
		{"span: 슬라이드 직속", l(`<p span="2">내용</p>`), 2, 6, []string{"슬라이드 직속", "grid"}},
		// at과 x/y는 둘 다 자리를 정하는 채널이라 함께 쓸 수 없다 — 진단은 at을 가리킨다.
		{"at/x 충돌", l(`<stack><p at="center" x="10" y="20">내용</p></stack>`), 2, 13, []string{"함께 쓸 수 없다"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			deck, diag := oneDiag(t, tc.src)
			assertDiag(t, diag, tc.line, tc.col, tc.contains...)
			if !strings.Contains(tc.src, "<p") {
				return
			}
			if p := findElement(t, deck, "p").Placement(); p != nil && (p.At != "" || p.X != nil || p.Y != nil || p.Span != nil) {
				t.Errorf("무효한 배치가 실렸다: %s", show(*p))
			}
		})
	}
}

// 진단 하나가 다른 속성의 진단을 삼키지 않는다 (파서·외관 해석과 같은 정책).
func TestResolveLayoutReportsEveryBadValue(t *testing.T) {
	_, diags := resolveSlide(t, layoutSlide(`<row gap="huge" align="centre" pad="1 2 3 4 5"><p>내용</p></row>`))
	if len(diags) != 3 {
		t.Fatalf("진단 %d개, want 3개: %v", len(diags), diags)
	}
	for i := 1; i < len(diags); i++ {
		if diags[i-1].Span.Start.Offset > diags[i].Span.Start.Offset {
			t.Errorf("진단이 소스 순서가 아니다: %v", diags)
		}
	}
}
