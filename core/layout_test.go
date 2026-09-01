package core

import (
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/k2sebeom/balpyo/soml"
)

// layoutSlide는 콘텐츠 **한 줄**을 담은 슬라이드 원문이다. 들여쓰기를 2칸으로 고정하는
// 이유는 진단 위치 단정이 표에서 읽히게 하기 위함이다 — 콘텐츠는 항상 2행 3열에서 시작한다.
func layoutSlide(content string) string {
	return "<slide id=\"layout\">\n  " + content + "\n</slide>\n"
}

// containerSrc는 컨테이너 하나와 자식 하나로 이루어진 슬라이드다.
func containerSrc(tag, attrs string) string {
	if attrs != "" {
		attrs = " " + attrs
	}
	return layoutSlide("<" + tag + attrs + "><p>내용</p></" + tag + ">")
}

// childSrc는 컨테이너 안의 `<p>`에 자식 배치 속성을 붙인 슬라이드다 — 자식 속성은 부모가
// 있어야 뜻이 있다 (span은 부모가 <grid>여야 한다).
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

// PRD §7.4 컨테이너 표의 값 문법을 전수로 못 박는다 — 속성 → **확정 배치 필드**.
// 렌더러는 Attrs를 다시 읽지 않으므로(soml.Appearance의 계약) 이 표가 곧 CSS의 입력이다.
func TestResolveContainerLayout(t *testing.T) {
	cases := []struct {
		name string
		attr string
		tag  string
		decl string
		want soml.ContainerLayout
	}{
		// 길이 — 무단위 수치는 논리 스테이지 px, `%`는 백분율, 공간 토큰은 이름과 테마 표의
		// 값을 **함께** 담는다 (웹이 var() 참조로 내야 하므로 — soml.Length.Token).
		{"gap 수치", "gap", "row", `gap="16"`, soml.ContainerLayout{Gap: ptr(soml.Px(16))}},
		{"gap 공간 토큰", "gap", "row", `gap="xl"`, soml.ContainerLayout{Gap: ptr(soml.SpaceLength("xl", 48))}},
		{"gap 백분율", "gap", "row", `gap="10%"`, soml.ContainerLayout{Gap: ptr(soml.Percent(10))}},
		// 0은 저자가 정한 값이다 — "정하지 않음"(nil)과 구별된다.
		{"gap 0", "gap", "row", `gap="0"`, soml.ContainerLayout{Gap: ptr(soml.Px(0))}},

		{"align", "align", "col", `align="stretch"`,
			soml.ContainerLayout{Flow: soml.FlowColumn, Align: soml.BoxAlignStretch}},
		{"justify", "justify", "row", `justify="between"`,
			soml.ContainerLayout{Justify: soml.BoxJustifyBetween}},

		// wrap은 값 없는 속성과 `="true"` 두 표기를 받는다 (resolver.wrap의 근거).
		{"wrap 값 없음", "wrap", "row", `wrap`, soml.ContainerLayout{Wrap: true}},
		{"wrap true", "wrap", "row", `wrap="true"`, soml.ContainerLayout{Wrap: true}},

		// pad 1~4값 — CSS padding의 상·우·하·좌 축약과 같은 규칙으로 4변까지 풀린다.
		// 박스 속성(pad·크기·ratio)은 `<shape>`와 **같은 구조체**에 담긴다 (soml.Box) —
		// 도형 해석 테스트가 같은 문자열에서 같은 값을 얻는지 이 표와 대조한다.
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

		// 크기 — `fill`은 해당 축의 100%로 눕는다 (표기가 둘이 되지 않게).
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
				t.Errorf("컨테이너 배치 = %s, want %s", showContainer(*got), showContainer(tc.want))
			}
			// 컨테이너 속성은 자식 배치가 아니다 — 둘이 섞이면 렌더러가 부모의 배치를
			// 자기 자리로 낸다.
			if elem.Placement() != nil {
				t.Errorf("자식 배치가 붙었다: %+v", *elem.Placement())
			}
		})
	}

	// 표가 PRD 표 전체를 덮는지 — 카탈로그가 연 속성을 값 층이 해석하지 않으면 저자가 쓴
	// 배치가 조용히 사라진다. 커버리지 판정을 테스트가 들고 있어야 항목 01의 목록이 넓어질 때
	// 이 표가 빨갛게 된다.
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

// PRD §7.4 자식 표 — 요소 종류와 무관하게 같은 표가 채운다 (아래 대상 전수 테스트가 짝이다).
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
		// span은 <grid>의 자식에서만 뜻이 있다 (부모가 row면 에러 — 아래 별도 테스트).
		{"span", "span", "grid", `span="2"`, soml.Placement{Span: ptr(2)}},
		{"self", "self", "row", `self="center"`, soml.Placement{Self: soml.BoxAlignCenter}},
		// order만 음수를 받는다 — "먼저 보이게"의 CSS 관용 표기다.
		{"order 음수", "order", "row", `order="-1"`, soml.Placement{Order: ptr(-1)}},
		// at·x·y는 겹침 배치 채널이다 — `at`은 stack 셀의 9지점 앵커, `x`/`y`는 절대 오프셋.
		// stack의 자식에서만(또는 x/y는 canvas 직속에서도) 뜻이 있다 (부모 한정은 아래 별도 테스트).
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
				t.Errorf("자식 배치 = %+v, want %+v", *got, tc.want)
			}
			// 자식은 컨테이너가 아니다 — Container가 non-nil이면 렌더러가 <p>에
			// display:flex를 낸다.
			if elem.Container() != nil {
				t.Errorf("컨테이너 배치가 붙었다: %+v", *elem.Container())
			}
		})
	}

	covered := map[string]bool{}
	for _, tc := range cases {
		covered[tc.attr] = true
	}
	// 자식 배치 속성은 전부 값 해석이 있어야 한다 — 카탈로그가 연 속성을 값 층이 모르면
	// 저자가 쓴 배치가 산출물에서 조용히 사라진다. at·x·y의 값 어휘(9지점 앵커·좌표 문법)와
	// 부모 한정 검증은 항목 03이 더했다 (아래 전수·부모 인식 테스트가 짝이다).
	for _, attr := range soml.ChildPlacementAttrs() {
		if !covered[attr] {
			t.Errorf("자식 배치 속성 %q의 해석 테스트가 없다", attr)
		}
	}
}

// 자식 배치 속성은 **요소 종류와 무관하게 같은 코드**가 해석한다 — 종류마다 표가 따로 있으면
// 속성 하나를 추가할 때 어느 종류가 빠졌는지 사람이 세어야 한다 (항목 01의 카탈로그와 짝).
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
		t.Run(tc.name, func(t *testing.T) {
			elem := resolvedElement(t, layoutSlide(tc.content), tc.tag)
			placement := elem.Placement()
			if placement == nil || placement.Grow == nil || *placement.Grow != 1 {
				t.Fatalf("<%s>의 grow가 해석되지 않았다: %+v", tc.tag, placement)
			}
		})
	}
}

// 배치와 텍스트 외관이 한 요소에 함께 얹힌다 — 두 채널은 서로를 모른다 (PRD §5.3).
// 한쪽이 다른 쪽의 Appearance를 지우면 산출물에서 배치나 외관 하나가 조용히 사라진다.
func TestResolvePlacementAndTextTogether(t *testing.T) {
	elem := resolvedElement(t, layoutSlide(`<row><p grow="1" size="30" color="muted">내용</p></row>`), "p")
	placement := elem.Placement()
	if placement == nil || placement.Grow == nil || *placement.Grow != 1 {
		t.Fatalf("배치가 해석되지 않았다: %+v", placement)
	}
	style := elem.TextStyle()
	if style == nil || style.Size != 30 || style.Color != soml.TokenColor("muted") {
		t.Fatalf("외관이 해석되지 않았다: %+v", style)
	}
	if style.Role != soml.RoleBody {
		t.Errorf("역할 = %q, want body — 배치 속성이 캐스케이드를 건드리면 안 된다", style.Role)
	}
}

// 배치 속성이 **없는** 요소는 배치 필드가 nil로 남는다. 렌더러의 골든 불변이 이 nil에
// 달려 있다 (항목 03) — 빈 구조체를 채우면 "정하지 않음"과 0을 구별할 수 없다.
func TestResolveLayoutNilWithoutAttrs(t *testing.T) {
	deck, diags := resolveSlide(t, layoutSlide(`<p>내용</p><ul><li>항목</li></ul><text>내용</text>`))
	if len(diags) != 0 {
		t.Fatalf("진단이 없어야 한다: %v", diags)
	}
	for _, tag := range []string{"p", "ul", "li", soml.TextElement} {
		elem := findElement(t, deck, tag)
		if elem.Container() != nil || elem.Placement() != nil {
			t.Errorf("<%s>에 배치가 붙었다: %+v", tag, *elem.Appearance)
		}
	}
	// 리스트 컨테이너는 외관도 배치도 없으므로 Appearance 자체가 nil이다 — Iteration 2의
	// 계약 그대로다 (컨테이너는 항목을 담을 뿐이다).
	if list := findElement(t, deck, "ul"); list.Appearance != nil {
		t.Errorf("<ul>에 Appearance가 생겼다: %+v", *list.Appearance)
	}
}

// 반대쪽: 레이아웃 컨테이너는 **속성이 하나도 없어도** Container를 갖는다. 배치 흐름은
// 속성이 아니라 태그가 정하는 값이고, 렌더러가 "이 요소가 컨테이너인가"를 다시 판단하지
// 않아야 컨테이너 CSS 전부가 이 구조체에서 나온다.
func TestResolveBareContainerCarriesFlow(t *testing.T) {
	cases := []struct {
		tag  string
		want soml.ContainerFlow
	}{
		{soml.RowElement, soml.FlowRow},
		{soml.ColElement, soml.FlowColumn},
		{soml.GridElement, soml.FlowGrid},
	}
	for _, tc := range cases {
		t.Run(tc.tag, func(t *testing.T) {
			elem := resolvedElement(t, containerSrc(tc.tag, ""), tc.tag)
			container := elem.Container()
			if container == nil {
				t.Fatalf("<%s>에 확정 컨테이너 배치가 없다", tc.tag)
			}
			if container.Flow != tc.want {
				t.Errorf("흐름 = %d, want %d", container.Flow, tc.want)
			}
			if !reflect.DeepEqual(*container, soml.ContainerLayout{Flow: tc.want}) {
				t.Errorf("속성 없는 컨테이너에 값이 실렸다: %s", showContainer(*container))
			}
		})
	}
	// <spacer>는 컨테이너가 아니다 — 배치 속성이 없으면 아무 배치도 붙지 않는다.
	if elem := resolvedElement(t, layoutSlide("<spacer/>"), soml.SpacerElement); elem.Appearance != nil {
		t.Errorf("<spacer>에 배치가 붙었다: %+v", *elem.Appearance)
	}
}

// 어휘는 순서 있는 슬라이스가 소유한다 (진단이 결정적이어야 한다) — 어휘 전체가 실제로
// 해석되는지 여기서 못 박는다. 목록에 값을 더하고 해석을 잊으면 빨갛게 된다.
func TestResolveBoxAlignVocabulary(t *testing.T) {
	for _, align := range soml.BoxAligns() {
		t.Run(string(align), func(t *testing.T) {
			container := resolvedElement(t, containerSrc("row", `align="`+string(align)+`"`), "row").Container()
			if container == nil || container.Align != align {
				t.Fatalf("align = %+v, want %q", container, align)
			}
			// self는 align과 같은 어휘를 쓴다 — 어휘가 갈라지면 저자가 두 표를 외운다.
			placement := resolvedElement(t, childSrc("row", `self="`+string(align)+`"`), "p").Placement()
			if placement == nil || placement.Self != align {
				t.Fatalf("self = %+v, want %q", placement, align)
			}
		})
	}
}

func TestResolveBoxJustifyVocabulary(t *testing.T) {
	for _, justify := range soml.BoxJustifies() {
		t.Run(string(justify), func(t *testing.T) {
			container := resolvedElement(t, containerSrc("row", `justify="`+string(justify)+`"`), "row").Container()
			if container == nil || container.Justify != justify {
				t.Fatalf("justify = %+v, want %q", container, justify)
			}
		})
	}
}

// canvasSlide는 layout="canvas" 슬라이드다 — layoutSlide와 같은 2칸 들여쓰기라 진단 위치
// 단정이 같은 열에서 읽힌다. `x`/`y`가 canvas 슬라이드의 직속 자식에서 유효한지 보려면
// 슬라이드의 layout이 canvas여야 한다 (coordParentOK).
func canvasSlide(content string) string {
	return "<slide id=\"canvas\" layout=\"canvas\">\n  " + content + "\n</slide>\n"
}

// at의 9지점 앵커 어휘 전수 — 순서 있는 슬라이스가 소유하고(진단이 결정적) 어휘 전체가 실제로
// 해석되는지 못 박는다. stack의 자식에서만 뜻이 있으므로 부모는 stack이다.
func TestResolveStackAnchorVocabulary(t *testing.T) {
	for _, anchor := range soml.StackAnchors() {
		t.Run(string(anchor), func(t *testing.T) {
			elem := resolvedElement(t, childSrc("stack", `at="`+string(anchor)+`"`), "p")
			placement := elem.Placement()
			if placement == nil || placement.At != anchor {
				t.Fatalf("at = %+v, want %q", placement, anchor)
			}
			// at은 자식 배치다 — 컨테이너가 아니다.
			if elem.Container() != nil {
				t.Errorf("컨테이너 배치가 붙었다: %+v", *elem.Container())
			}
		})
	}
}

// x/y의 값 문법 — 수치·%·음수·소수. 지정하지 않은 축은 nil로 남는다 ("정하지 않음"과 0을
// 구별한다 — 골든 불변의 근거와 같은 종류).
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
			if !reflect.DeepEqual(placement.X, tc.wantX) {
				t.Errorf("x = %v, want %v", showCoord(placement.X), showCoord(tc.wantX))
			}
			if !reflect.DeepEqual(placement.Y, tc.wantY) {
				t.Errorf("y = %v, want %v", showCoord(placement.Y), showCoord(tc.wantY))
			}
		})
	}
}

// 부모 인식 — 같은 속성이 <stack> 안에서는 통과하고 <row> 안·슬라이드 직속에서는 실패한다
// (resolver.span의 테스트가 하는 방식으로 쌍을 나란히 둔다). at은 stack만, x/y는 stack이나
// canvas 직속만이다 — 이 경계가 틀리면 캔버스 전체가 절대 배치 지대가 된다.
func TestResolveAtNeedsStackParent(t *testing.T) {
	if p := resolvedElement(t, childSrc("stack", `at="center"`), "p").Placement(); p == nil ||
		p.At != soml.StackCenter {
		t.Fatalf("<stack>의 자식 at이 해석되지 않았다: %+v", p)
	}
	cases := []struct {
		name    string
		content string
		col     int
	}{
		{"flex 컨테이너의 자식", `<row><p at="center">내용</p></row>`, 11},
		{"슬라이드 직속", `<p at="center">내용</p>`, 6},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			deck, diags := resolveSlide(t, layoutSlide(tc.content))
			if len(diags) != 1 {
				t.Fatalf("진단 %d개, want 1개: %v", len(diags), diags)
			}
			diag := diags[0]
			if diag.Severity != soml.SeverityError || !strings.Contains(diag.Message, "at은 <stack>") {
				t.Errorf("진단 = %q, want at 부모 진단", diag.String())
			}
			// 진단은 속성 **이름**을 가리킨다 — 고칠 것이 값이 아니라 속성의 자리다 (span과 같다).
			if diag.Span.Start.Line != 2 || diag.Span.Start.Col != tc.col {
				t.Errorf("위치 = %s, want 2:%d", diag.Span.Start, tc.col)
			}
			if p := findElement(t, deck, "p").Placement(); p != nil && p.At != "" {
				t.Errorf("무효한 at이 실렸다: %+v", *p)
			}
		})
	}
}

// x/y의 부모 인식 — <stack>의 자식과 canvas 슬라이드의 직속 자식은 통과하고, content 슬라이드의
// 직속·어느 슬라이드든 <row>의 자식은 실패한다. canvas 슬라이드의 <row> 자식이 실패하는 것이
// 경계의 요점이다 (부모가 stack도 canvas 직속도 아니다 — 캔버스가 통째로 절대 배치 지대가 되면
// 안 된다).
func TestResolveCoordNeedsStackOrCanvas(t *testing.T) {
	// 통과: <stack>의 자식.
	if p := resolvedElement(t, childSrc("stack", `x="40"`), "p").Placement(); p == nil ||
		p.X == nil || *p.X != soml.Px(40) {
		t.Fatalf("<stack>의 자식 x가 해석되지 않았다: %+v", p)
	}
	// 통과: canvas 슬라이드의 직속 자식 (부모가 요소가 아니다 — parent == nil이고 layout canvas).
	if p := resolvedElement(t, canvasSlide(`<p x="40">내용</p>`), "p").Placement(); p == nil ||
		p.X == nil || *p.X != soml.Px(40) {
		t.Fatalf("canvas 직속 x가 해석되지 않았다: %+v", p)
	}
	cases := []struct {
		name string
		src  string
		col  int
	}{
		{"content 슬라이드 직속", layoutSlide(`<p x="40">내용</p>`), 6},
		{"content의 <row> 자식", layoutSlide(`<row><p x="40">내용</p></row>`), 11},
		{"canvas의 <row> 자식", canvasSlide(`<row><p x="40">내용</p></row>`), 11},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			deck, diags := resolveSlide(t, tc.src)
			if len(diags) != 1 {
				t.Fatalf("진단 %d개, want 1개: %v", len(diags), diags)
			}
			diag := diags[0]
			// 진단이 고칠 방법(<stack>·layout="canvas")을 담는다.
			if diag.Severity != soml.SeverityError || !strings.Contains(diag.Message, "canvas") ||
				!strings.Contains(diag.Message, soml.StackElement) {
				t.Errorf("진단 = %q, want x 부모 진단", diag.String())
			}
			if diag.Span.Start.Line != 2 || diag.Span.Start.Col != tc.col {
				t.Errorf("위치 = %s, want 2:%d", diag.Span.Start, tc.col)
			}
			if p := findElement(t, deck, "p").Placement(); p != nil && p.X != nil {
				t.Errorf("무효한 x가 실렸다: %+v", *p)
			}
		})
	}
}

// at과 x/y를 같은 자식에 함께 쓰면 에러다 — 둘 다 자리를 정하는 채널이라 하나만 쓴다
// (ADR 0027의 variant/frame 기준). 진단은 하나이고 at을 가리키며 x의 자리를 문구에 담는다.
// 충돌한 채널은 확정 배치에 실리지 않는다 (저자가 고친 뒤의 결과를 예측할 수 있어야 한다).
func TestResolveAnchorCoordConflict(t *testing.T) {
	deck, diags := resolveSlide(t, layoutSlide(`<stack><p at="center" x="10" y="20">내용</p></stack>`))
	if len(diags) != 1 {
		t.Fatalf("진단 %d개, want 1개: %v", len(diags), diags)
	}
	diag := diags[0]
	if diag.Severity != soml.SeverityError || !strings.Contains(diag.Message, "함께 쓸 수 없다") {
		t.Errorf("진단 = %q, want at/x 충돌", diag.String())
	}
	// at의 이름 자리를 가리킨다: <stack>(3-9) 뒤 <p (10-12), a=13.
	if diag.Span.Start.Line != 2 || diag.Span.Start.Col != 13 {
		t.Errorf("위치 = %s, want 2:13", diag.Span.Start)
	}
	// 충돌한 at/x/y는 실리지 않는다 — 다른 배치 속성이 없으므로 Placement 자체가 nil이다.
	if p := findElement(t, deck, "p").Placement(); p != nil {
		t.Errorf("충돌한 배치가 실렸다: %+v", *p)
	}
}

// at의 어휘 진단 — 짧은 이름·오타·축 뒤바뀜은 올바른 9지점을 짚어 주고(고치는 방법이 진단
// 안에 있다), 값 없는 at·x/y의 값 문법 위반은 문제가 된 조각을 line:col로 가리킨다.
func TestResolveAnchorCoordDiagnostics(t *testing.T) {
	cases := []struct {
		name     string
		content  string
		line     int
		col      int
		contains string
	}{
		// at — 짧은 이름은 나머지 축을 center로 채운 자리를 권한다.
		{"at 짧은 이름", `<stack><p at="top">내용</p></stack>`, 2, 17, "top-center"},
		{"at 축 뒤바뀜", `<stack><p at="right-top">내용</p></stack>`, 2, 17, "top-right"},
		{"at 오타", `<stack><p at="centre">내용</p></stack>`, 2, 17, "알 수 없는 at"},
		{"at 어휘 밖", `<stack><p at="stretch">내용</p></stack>`, 2, 17, "알 수 없는 at"},
		{"center-center 거부", `<stack><p at="center-center">내용</p></stack>`, 2, 17, "center"},
		{"값 없는 at", `<stack><p at>내용</p></stack>`, 2, 13, "값이 필요하다"},
		// x/y — 좌표는 토큰·fill을 받지 않고, 단위 접미사는 기존 numberSuffix 진단을 재사용한다.
		{"x 공간 토큰", `<stack><p x="md">내용</p></stack>`, 2, 16, "gap·pad에만"},
		{"x fill", `<stack><p x="fill">내용</p></stack>`, 2, 16, "수치가 아니다"},
		{"x 단위 접미사", `<stack><p x="10px">내용</p></stack>`, 2, 16, "단위 없는 논리 px"},
		{"값 없는 y", `<stack><p y>내용</p></stack>`, 2, 13, "값이 필요하다"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, diags := resolveSlide(t, layoutSlide(tc.content))
			if len(diags) != 1 {
				t.Fatalf("진단 %d개, want 1개: %v", len(diags), diags)
			}
			diag := diags[0]
			if !diag.Span.Start.IsValid() || diag.Span.Start.Line != tc.line || diag.Span.Start.Col != tc.col {
				t.Errorf("위치 = %s, want %d:%d (%s)", diag.Span.Start, tc.line, tc.col, diag.Message)
			}
			if diag.Severity != soml.SeverityError || !strings.Contains(diag.Message, tc.contains) {
				t.Errorf("진단 = %q, want %q 포함", diag.String(), tc.contains)
			}
		})
	}
}

// signed 축의 회귀 (항목 03이 lengthValue에 signed를 더했다): 크기·간격·basis의 음수는
// **여전히** 에러이고 좌표 x/y의 음수만 통과한다. 한 함수가 두 규칙을 갖게 되므로 회귀가
// 여기서 난다 — 두 규칙을 한 테스트에 나란히 둔다.
func TestResolveSignedAxisRegression(t *testing.T) {
	for _, tc := range []struct {
		name    string
		content string
		col     int
	}{
		{"음수 폭", `<row w="-10"><p>내용</p></row>`, 11},
		{"음수 gap", `<row gap="-4"><p>내용</p></row>`, 13},
		{"음수 basis", `<row><p basis="-1">내용</p></row>`, 18},
		{"음수 min-h", `<row min-h="-5"><p>내용</p></row>`, 15},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, diags := resolveSlide(t, layoutSlide(tc.content))
			if len(diags) != 1 {
				t.Fatalf("진단 %d개, want 1개: %v", len(diags), diags)
			}
			diag := diags[0]
			if diag.Severity != soml.SeverityError || !strings.Contains(diag.Message, "0보다 작을 수 없다") {
				t.Errorf("진단 = %q, want 음수 금지", diag.String())
			}
			if diag.Span.Start.Line != 2 || diag.Span.Start.Col != tc.col {
				t.Errorf("위치 = %s, want 2:%d", diag.Span.Start, tc.col)
			}
		})
	}
	// 좌표는 음수를 받는다 — 박스 밖으로 걸친 배지.
	p := resolvedElement(t, childSrc("stack", `x="-16" y="-8"`), "p").Placement()
	if p == nil || p.X == nil || *p.X != soml.Px(-16) || p.Y == nil || *p.Y != soml.Px(-8) {
		t.Fatalf("음수 좌표가 해석되지 않았다: %+v", p)
	}
}

// _06_stack 픽스처는 이 iteration 전체의 시험대다 (항목 06이 활성화하고 골든을 붙인다). 이
// 항목의 몫은 **배치**이므로, 값 층이 픽스처의 at/x/y/stack을 진단 없이 해석하는지 못 박는다.
//
// 남은 진단은 전부 색 토큰(accent2~5·on-accent)이 하드코딩 테마(minimalTheme)에 아직 없어서
// 나는 외관 진단이다 — 그 팔레트를 채우는 것은 픽스처 활성화(항목 06)의 몫이고 배치와 무관하다.
// 그래서 "진단 0건"이 아니라 "**배치 진단 0건**"을 단정한다: 배치 진단이 하나라도 있으면 색
// 토큰 진단이 아니므로 여기서 빨갛게 된다.
func TestResolveStackFixture(t *testing.T) {
	deck := readFixtureDeck(t, "_06_stack")
	diags := Resolve(deck)
	for _, diag := range diags {
		// 모든 진단은 색 토큰 외관 진단이어야 한다 — 배치(at/x/y/stack/canvas) 진단이 새면
		// 이 문자열을 담지 않으므로 드러난다.
		if !strings.Contains(diag.Message, "색 토큰") {
			t.Errorf("배치 관련 진단이 픽스처에서 났다: %s", diag.String())
		}
	}
	// 9지점 앵커가 전부 확정 배치에 실렸는지 (001_anchors가 9개를 전수로 쓴다 — accent1·surface만
	// 써서 색 진단 없이 해석된다).
	seen := map[soml.StackAnchor]bool{}
	var walk func(node soml.Node)
	walk = func(node soml.Node) {
		elem, ok := node.(*soml.Element)
		if !ok {
			return
		}
		if p := elem.Placement(); p != nil && p.At != "" {
			seen[p.At] = true
		}
		for _, child := range elem.Children {
			walk(child)
		}
	}
	for _, slide := range deck.Slides {
		for _, node := range slide.Content {
			walk(node)
		}
	}
	for _, anchor := range soml.StackAnchors() {
		if !seen[anchor] {
			t.Errorf("9지점 앵커 %q가 픽스처에서 확정되지 않았다", anchor)
		}
	}
}

// showCoord는 실패 메시지용 좌표 문자열이다 — 포인터라 %v로는 주소만 보인다.
func showCoord(l *soml.Length) string {
	if l == nil {
		return "nil"
	}
	return showLength(*l)
}

// 공간 토큰과 수치가 같은 자리에서 갈라지는 경계 — 진단의 정확도가 이 판정에 달려 있다
// (`gap="huge"`는 "모르는 토큰", `gap="16px"`은 "단위를 떼라").
func TestResolveSpaceTokenBoundary(t *testing.T) {
	tokenGap := resolvedElement(t, containerSrc("row", `gap="md"`), "row").Container().Gap
	numberGap := resolvedElement(t, containerSrc("row", `gap="16"`), "row").Container().Gap
	if tokenGap == nil || numberGap == nil {
		t.Fatal("gap이 해석되지 않았다")
	}
	if *tokenGap != soml.SpaceLength("md", 24) {
		t.Errorf("gap=md → %+v, want md(24px) 토큰 참조", *tokenGap)
	}
	if *numberGap != soml.Px(16) {
		t.Errorf("gap=16 → %+v, want 16px", *numberGap)
	}
	// 둘은 같은 타입에 도달하되 **구별된 상태**로 도달한다: 토큰은 이름을 남기고 저자가 적은
	// 수치는 남기지 않는다. 웹이 토큰만 var() 참조로 내려면(soml.Length.Token) 이 구별이
	// 값 층에서 살아 있어야 하고, px로만 눕히면 `gap="24"`와 `gap="md"`가 산출물에서 같은
	// 것이 된다 — 테마의 리듬을 바꿨을 때 후자만 따라 움직여야 한다.
	if numberGap.IsToken() {
		t.Errorf("저자가 적은 수치에 토큰 이름이 붙었다 (%+v)", *numberGap)
	}
	if tokenGap.Percent || numberGap.Percent {
		t.Error("논리 px 길이가 백분율로 해석됐다")
	}
	// 토큰 이름이 숫자로 시작해도 토큰이다 — `2xl`은 어휘(PRD §8.4)의 유일한 그런 이름이고,
	// 수치 판정을 먼저 하면 이 토큰만 영원히 "단위를 떼라"는 진단을 받는다 (항목 03에서
	// `pad="2xl"`이 CSS 변수 참조로 나가지 않는 것으로 드러났다).
	twoXL := resolvedElement(t, containerSrc("row", `pad="2xl"`), "row").Container().Pad
	if twoXL == nil || *twoXL != (soml.Pad{
		Top: soml.SpaceLength("2xl", 72), Right: soml.SpaceLength("2xl", 72),
		Bottom: soml.SpaceLength("2xl", 72), Left: soml.SpaceLength("2xl", 72)}) {
		t.Errorf("pad=2xl → %+v, want 2xl(72px) 토큰 참조 4변", twoXL)
	}
	// 그래도 단위 접미사·모르는 토큰 진단은 그대로다 — 표에 없는 이름은 수치 문법이
	// 판정한다 (그 경로는 TestResolveLayoutDiagnostics의 `gap="16px"`·`gap="huge"` 행이 본다).
}

// 테마의 공간 토큰 표 — 어휘는 PRD §8.4의 6종이고 값은 수직 리듬에서 나온다.
func TestThemeSpaceTokens(t *testing.T) {
	theme := minimalTheme()
	want := []string{"xs", "sm", "md", "lg", "xl", "2xl"}
	if got := theme.SpaceNames(); !reflect.DeepEqual(got, want) {
		t.Errorf("공간 토큰 = %v, want %v (PRD §8.4의 어휘, 작은 것부터)", got, want)
	}
	prev := 0.0
	for _, token := range theme.Spaces {
		if token.Px <= prev {
			t.Errorf("%s(%gpx)가 앞 토큰(%gpx)보다 크지 않다 — 사다리는 단조 증가해야 한다",
				token.Name, token.Px, prev)
		}
		prev = token.Px
	}
	// 값이 리듬에 얹혀 있는지: 본문 줄박스 36px과 본문 크기 24px의 약수·배수여야 여백이
	// 글줄 격자에서 미끄러지지 않는다 (core/theme.go의 근거).
	for _, token := range theme.Spaces {
		if token.Px != float64(int(token.Px)) || int(token.Px)%6 != 0 {
			t.Errorf("%s = %gpx — 사다리는 6px(리듬의 최소 눈금) 배수여야 한다", token.Name, token.Px)
		}
	}
	// clrScheme 슬롯 계약이 공간 토큰에는 없다 (soml.SpaceToken의 주석) — 이 비대칭이
	// 의도임을 색 토큰 테스트와 나란히 남긴다.
	if len(theme.Colors) == 0 || len(theme.Spaces) == 0 {
		t.Fatal("테마 표가 비어 있다")
	}
}

// 값 층의 진단은 파서와 같은 정밀도를 갖는다 — 문제가 된 **값(조각)**을 line:col로 가리킨다.
func TestResolveLayoutDiagnostics(t *testing.T) {
	cases := []struct {
		name     string
		content  string
		line     int
		col      int
		contains string
	}{
		// 길이·공간 토큰
		{"모르는 공간 토큰", `<row gap="huge"><p>내용</p></row>`, 2, 13, "알 수 없는 공간 토큰"},
		{"단위 접미사", `<row gap="16px"><p>내용</p></row>`, 2, 13, "단위 없는 논리 px"},
		{"음수 길이", `<row gap="-4"><p>내용</p></row>`, 2, 13, "0보다 작을 수 없다"},
		{"크기에 쓴 공간 토큰", `<row w="md"><p>내용</p></row>`, 2, 11, "gap·pad에만"},
		{"크기에 쓴 이름", `<row w="가득"><p>내용</p></row>`, 2, 11, "수치가 아니다"},
		{"gap에 쓴 fill", `<row gap="fill"><p>내용</p></row>`, 2, 13, "알 수 없는 공간 토큰"},
		// 닫힌 어휘
		{"오타 align", `<row align="centre"><p>내용</p></row>`, 2, 15, "알 수 없는 align"},
		{"CSS 접두 align", `<row align="flex-start"><p>내용</p></row>`, 2, 15, `"start"`},
		{"오타 justify", `<row justify="middle"><p>내용</p></row>`, 2, 17, "알 수 없는 justify"},
		{"CSS 접두 justify", `<row justify="space-between"><p>내용</p></row>`, 2, 17, `"between"`},
		{"오타 self", `<row><p self="middle">내용</p></row>`, 2, 17, "알 수 없는 self"},
		// 불리언
		{"wrap false", `<row wrap="false"><p>내용</p></row>`, 2, 14, "끄려면 속성을 지운다"},
		{"빈 wrap", `<row wrap=""><p>내용</p></row>`, 2, 14, "비어 있다"},
		// 목록 — 진단은 값 전체가 아니라 문제가 된 조각을 가리킨다.
		{"pad 5값", `<row pad="1 2 3 4 5"><p>내용</p></row>`, 2, 13, "1~4개 값"},
		{"pad 둘째 값", `<row pad="sm huge"><p>내용</p></row>`, 2, 16, "알 수 없는 공간 토큰"},
		{"트랙 개수 0", `<grid cols="0"><p>내용</p></grid>`, 2, 15, "1 이상 정수"},
		{"모르는 트랙", `<grid cols="넷"><p>내용</p></grid>`, 2, 15, "트랙 값"},
		{"음수 트랙", `<grid cols="2fr -1fr"><p>내용</p></grid>`, 2, 19, "0보다 작을 수 없다"},
		// 비율
		{"콜론 없는 ratio", `<row ratio="16"><p>내용</p></row>`, 2, 15, "W:H 정수 쌍"},
		{"세 값 ratio", `<row ratio="16:9:2"><p>내용</p></row>`, 2, 15, "W:H 정수 쌍"},
		{"0 ratio", `<row ratio="0:9"><p>내용</p></row>`, 2, 15, "0보다 커야"},
		// 정수
		{"grow 소수", `<row><p grow="1.5">내용</p></row>`, 2, 17, "정수다"},
		{"grow 음수", `<row><p grow="-1">내용</p></row>`, 2, 17, "0보다 작을 수 없다"},
		{"order 비수치", `<row><p order="처음">내용</p></row>`, 2, 18, "정수다"},
		{"span 0", `<grid><p span="0">내용</p></grid>`, 2, 18, "1 이상 정수"},
		// 값 없음·빈 값 — 외관 속성과 같은 진단 경로다 (hasValue).
		{"값 없는 크기", `<row w><p>내용</p></row>`, 2, 8, "값이 필요하다"},
		{"빈 gap", `<row gap=""><p>내용</p></row>`, 2, 13, "비어 있다"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, diags := resolveSlide(t, layoutSlide(tc.content))
			if len(diags) != 1 {
				t.Fatalf("진단 %d개, want 1개: %v", len(diags), diags)
			}
			diag := diags[0]
			if !diag.Span.Start.IsValid() || diag.Span.Start.Line != tc.line || diag.Span.Start.Col != tc.col {
				t.Errorf("위치 = %s, want %d:%d (%s)", diag.Span.Start, tc.line, tc.col, diag.Message)
			}
			if diag.Severity != soml.SeverityError || !strings.Contains(diag.Message, tc.contains) {
				t.Errorf("진단 = %q, want %q 포함", diag.String(), tc.contains)
			}
		})
	}
}

// span은 <grid>의 자식에서만 뜻이 있다 — flex 컨테이너와 슬라이드 직속에서 grid-column은
// CSS가 조용히 버리는 선언이다. 파서는 부모를 모르는 자리에서 속성 표를 만들므로 이 검증의
// 자리는 순회하는 값 층뿐이다.
func TestResolveSpanNeedsGridParent(t *testing.T) {
	if placement := resolvedElement(t, childSrc("grid", `span="2"`), "p").Placement(); placement == nil ||
		placement.Span == nil || *placement.Span != 2 {
		t.Fatalf("<grid>의 자식 span이 해석되지 않았다: %+v", placement)
	}
	cases := []struct {
		name     string
		content  string
		col      int
		contains string
	}{
		{"flex 컨테이너의 자식", `<row><p span="2">내용</p></row>`, 11, "부모는 <row>다"},
		{"슬라이드 직속", `<p span="2">내용</p>`, 6, "슬라이드 직속"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			deck, diags := resolveSlide(t, layoutSlide(tc.content))
			if len(diags) != 1 {
				t.Fatalf("진단 %d개, want 1개: %v", len(diags), diags)
			}
			diag := diags[0]
			if diag.Severity != soml.SeverityError || !strings.Contains(diag.Message, tc.contains) ||
				!strings.Contains(diag.Message, "grid") {
				t.Errorf("진단 = %q, want %q 포함", diag.String(), tc.contains)
			}
			// 진단은 속성 **이름**을 가리킨다 — 고칠 것이 값이 아니라 속성의 자리이기 때문이다.
			if diag.Span.Start.Line != 2 || diag.Span.Start.Col != tc.col {
				t.Errorf("위치 = %s, want 2:%d", diag.Span.Start, tc.col)
			}
			// 틀린 값은 확정 배치에 실리지 않는다.
			if placement := findElement(t, deck, "p").Placement(); placement != nil && placement.Span != nil {
				t.Errorf("무효한 span이 실렸다: %+v", *placement)
			}
		})
	}
}

// 진단 하나가 다른 속성의 진단을 삼키지 않는다 — 저자가 빌드를 오타 수만큼 돌지 않게
// (파서·외관 해석과 같은 정책).
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

// 픽스처는 이 iteration 전체의 시험대다 (항목 05가 골든·픽셀 비교를 붙였다) — 값 층이
// 픽스처를 해석하는지, 경고가 나는 슬라이드가 **하나뿐**인지 여기서 못 박는다. 픽스처 러너의
// 골든 `diagnostics.txt`가 같은 사실을 산출물 쪽에서 지킨다 (경고는 있을 때만 골든이 된다).
func TestResolveLayoutFixture(t *testing.T) {
	deck := readFixtureDeck(t, "03_layout")
	diags := Resolve(deck)
	for _, diag := range diags {
		if diag.Severity != soml.SeverityWarning {
			t.Errorf("픽스처에 에러가 있다: %s", diag.String())
		}
		if !strings.HasSuffix(diag.File, "007_style_appearance"+soml.Ext) {
			t.Errorf("경고가 007 밖에서 났다: %s", diag.String())
		}
	}
	// 007의 외관 선언 2개(background, border)가 경고의 전부다.
	if len(diags) != 2 {
		t.Fatalf("경고 %d개, want 2개: %v", len(diags), diags)
	}

	// 002_grid_cards — TASKS의 픽셀 비교 대상. `<grid cols="4" gap="lg">` 한 줄이 카드
	// 좌표의 근거이므로 값이 여기서 확정되어야 한다.
	grid := findElement(t, deck, soml.GridElement)
	container := grid.Container()
	if container == nil || container.Flow != soml.FlowGrid {
		t.Fatalf("<grid>의 배치가 해석되지 않았다: %+v", container)
	}
	if container.Cols == nil || container.Cols.Count != 4 || len(container.Cols.List) != 0 {
		t.Errorf("cols = %+v, want 4트랙 균등 분할", container.Cols)
	}
	if container.Gap == nil || *container.Gap != soml.SpaceLength("lg", 36) {
		t.Errorf("gap = %+v, want lg(36px) 토큰 참조", container.Gap)
	}
}

// readFixtureDeck는 픽스처의 슬라이드 파일을 읽어 해석 전 덱을 만든다.
//
// 픽스처는 완전한 프로젝트다 (testdata/fixtures/README.md). 여기서는 파이프라인을 거치지
// 않고 슬라이드 파일만 읽는다 — core는 internal/pipeline을 임포트할 수 없다 (임포트 순환).
func readFixtureDeck(t *testing.T, fixture string) *soml.Deck {
	t.Helper()
	dir := filepath.Join("..", "testdata", "fixtures", fixture, "slides")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), soml.Ext) {
			names = append(names, entry.Name())
		}
	}
	sort.Strings(names)
	if len(names) == 0 {
		t.Fatalf("%s에 슬라이드가 없다", dir)
	}
	deck := &soml.Deck{}
	for _, name := range names {
		src, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatal(err)
		}
		doc, parseDiags := soml.Parse("slides/"+name, string(src))
		if len(parseDiags) != 0 {
			t.Fatalf("%s 파서 진단: %v", name, parseDiags)
		}
		deck.Slides = append(deck.Slides, &soml.Slide{
			Index: len(deck.Slides), ID: doc.ID, Source: doc.File,
			Layout: soml.SlideLayout(doc.Layout), LayoutSpan: doc.LayoutSpan,
			Content: doc.Content, Tokens: doc.Tokens,
		})
	}
	return deck
}

// showContainer는 실패 메시지용 문자열이다 — 포인터 필드가 많아 %+v로는 주소만 보인다.
func showContainer(c soml.ContainerLayout) string {
	var b strings.Builder
	fields := []struct {
		name   string
		length *soml.Length
	}{
		{"gap", c.Gap}, {"w", c.Width}, {"h", c.Height},
		{"min-w", c.MinWidth}, {"min-h", c.MinHeight}, {"max-w", c.MaxWidth}, {"max-h", c.MaxHeight},
	}
	b.WriteString("{flow:")
	b.WriteString([]string{"row", "col", "grid"}[c.Flow])
	for _, field := range fields {
		if field.length != nil {
			b.WriteString(" " + field.name + ":" + showLength(*field.length))
		}
	}
	if c.Align != "" {
		b.WriteString(" align:" + string(c.Align))
	}
	if c.Justify != "" {
		b.WriteString(" justify:" + string(c.Justify))
	}
	if c.Wrap {
		b.WriteString(" wrap")
	}
	if c.Pad != nil {
		b.WriteString(" pad:" + showLength(c.Pad.Top) + "/" + showLength(c.Pad.Right) +
			"/" + showLength(c.Pad.Bottom) + "/" + showLength(c.Pad.Left))
	}
	for name, tracks := range map[string]*soml.Tracks{"cols": c.Cols, "rows": c.Rows} {
		if tracks != nil {
			b.WriteString(" " + name + ":" + showTracks(*tracks))
		}
	}
	if c.Ratio != nil {
		b.WriteString(" ratio:" + strconv.Itoa(c.Ratio.W) + ":" + strconv.Itoa(c.Ratio.H))
	}
	b.WriteString("}")
	return b.String()
}

func showLength(l soml.Length) string {
	unit := "px"
	if l.Percent {
		unit = "%"
	}
	return strconv.FormatFloat(l.Value, 'g', -1, 64) + unit
}

func showTracks(t soml.Tracks) string {
	if t.Count > 0 {
		return "repeat(" + strconv.Itoa(t.Count) + ")"
	}
	parts := make([]string, 0, len(t.List))
	for _, track := range t.List {
		value := strconv.FormatFloat(track.Value, 'g', -1, 64)
		switch track.Kind {
		case soml.TrackAuto:
			parts = append(parts, "auto")
		case soml.TrackFr:
			parts = append(parts, value+"fr")
		case soml.TrackPercent:
			parts = append(parts, value+"%")
		default:
			parts = append(parts, value+"px")
		}
	}
	return "[" + strings.Join(parts, " ") + "]"
}
