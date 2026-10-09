package core

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/k2sebeom/balpyo/soml"
)

// shapeSlide는 도형 하나(자기 닫힘)를 담은 슬라이드다 — 도형은 항상 2행 3열에서 시작한다.
func shapeSlide(attrs string) string {
	return "<slide id=\"shape\">\n  <shape " + attrs + "/>\n</slide>\n"
}

// shapeCol은 속성 문자열 안 needle이 놓이는 열이다 (속성은 ASCII라 바이트 오프셋이 곧 열이다).
func shapeCol(attrs, needle string) int {
	index := strings.Index(attrs, needle)
	if index < 0 {
		panic("needle이 속성에 없다: " + needle)
	}
	return 3 + len("<shape ") + index
}

// resolvedShape는 진단 없이 해석된 도형의 확정 외관이다.
func resolvedShape(t *testing.T, attrs string) *soml.ShapeStyle {
	t.Helper()
	style := resolvedElement(t, shapeSlide(attrs), soml.ShapeElement).ShapeStyle()
	if style == nil {
		t.Fatalf("<shape %s>에 확정 외관이 없다", attrs)
	}
	return style
}

// wantShapeStyle는 표의 항목(컴포넌트 기본값 위의 델타)을 기대값으로 바꾼다. 병합 규칙을
// 다시 쓰지 않으려고 운영 코드의 mergeShape를 쓴다.
func wantShapeStyle(delta soml.ShapeStyle) soml.ShapeStyle {
	style := componentShapeStyle(delta.Type)
	var flip *soml.Flip
	if delta.Flip != "" {
		flip = &delta.Flip
	}
	mergeShape(&style, shapePatch{fill: delta.Fill, border: delta.Border, radius: delta.Radius,
		shadow: delta.Shadow, opacity: delta.Opacity, rotate: delta.Rotate, flip: flip})
	style.ID, style.Alt, style.Box = delta.ID, delta.Alt, delta.Box
	return style
}

// 도형 외관 속성 표(docs/reference/soml/shape.md) 전수 — 속성 → 확정 ShapeStyle 필드.
// want는 컴포넌트 기본값 위의 델타다.
func TestResolveShapeStyle(t *testing.T) {
	softShadow := minimalShadows()[0]
	cases := []struct {
		name  string
		attr  string
		attrs string
		want  soml.ShapeStyle
	}{
		{"type", "type", `type="ellipse"`, soml.ShapeStyle{Type: "ellipse"}},
		{"id", "id", `type="rect" id="db"`, soml.ShapeStyle{Type: "rect", ID: "db"}},
		{"alt", "alt", `type="arrow-right" alt="배포 흐름의 마지막 단계"`,
			soml.ShapeStyle{Type: "arrow-right", Alt: "배포 흐름의 마지막 단계"}},

		{"variant", "variant", `type="rounded-rect" variant="card"`,
			soml.ShapeStyle{Type: "rounded-rect", Fill: ptr(soml.ColorFill(soml.TokenColor("bg"))),
				Radius: ptr(soml.TokenRadius("lg", 24))}},

		// 토큰은 이름으로 남고(schemeClr) 리터럴은 대문자 6자리로 정규화된다.
		{"fill 토큰", "fill", `type="rect" fill="surface"`,
			soml.ShapeStyle{Type: "rect", Fill: ptr(soml.ColorFill(soml.TokenColor("surface")))}},
		{"fill 리터럴 소문자", "fill", `type="rect" fill="#0f65f9"`,
			soml.ShapeStyle{Type: "rect", Fill: ptr(soml.ColorFill(soml.LiteralColor("0F65F9")))}},
		{"fill 리터럴 3자리", "fill", `type="rect" fill="#fff"`,
			soml.ShapeStyle{Type: "rect", Fill: ptr(soml.ColorFill(soml.LiteralColor("FFFFFF")))}},
		{"fill none", "fill", `type="rect" fill="none"`,
			soml.ShapeStyle{Type: "rect", Fill: ptr(soml.NoFill())}},

		{"border 토큰 색", "border", `type="rect" border="1 solid muted"`,
			soml.ShapeStyle{Type: "rect", Border: &soml.Border{
				Width: 1, Style: soml.BorderSolid, Color: soml.TokenColor("muted")}}},
		{"border 리터럴 색·소수 굵기", "border", `type="rect" border="2.5 dashed #6B7280"`,
			soml.ShapeStyle{Type: "rect", Border: &soml.Border{
				Width: 2.5, Style: soml.BorderDashed, Color: soml.LiteralColor("6B7280")}}},
		{"border dotted", "border", `type="rect" border="3 dotted accent1"`,
			soml.ShapeStyle{Type: "rect", Border: &soml.Border{
				Width: 3, Style: soml.BorderDotted, Color: soml.TokenColor("accent1")}}},
		{"border none", "border", `type="rect" border="none"`,
			soml.ShapeStyle{Type: "rect", Border: ptr(soml.NoBorder())}},

		{"radius 토큰", "radius", `type="rounded-rect" radius="lg"`,
			soml.ShapeStyle{Type: "rounded-rect", Radius: ptr(soml.TokenRadius("lg", 24))}},
		{"radius 수치", "radius", `type="rounded-rect" radius="32"`,
			soml.ShapeStyle{Type: "rounded-rect", Radius: ptr(soml.PxRadius(32))}},
		{"radius 0", "radius", `type="rounded-rect" radius="0"`,
			soml.ShapeStyle{Type: "rounded-rect", Radius: ptr(soml.PxRadius(0))}},

		{"shadow 토큰", "shadow", `type="rect" shadow="soft"`,
			soml.ShapeStyle{Type: "rect", Shadow: &softShadow}},

		{"opacity", "opacity", `type="rect" opacity="0.4"`,
			soml.ShapeStyle{Type: "rect", Opacity: ptr(0.4)}},
		// 0은 저자가 정한 값이다 — "정하지 않음"(nil)과 구별된다.
		{"opacity 0", "opacity", `type="rect" opacity="0"`,
			soml.ShapeStyle{Type: "rect", Opacity: ptr(0.0)}},
		{"opacity 1", "opacity", `type="rect" opacity="1"`,
			soml.ShapeStyle{Type: "rect", Opacity: ptr(1.0)}},

		{"rotate", "rotate", `type="rect" rotate="45"`,
			soml.ShapeStyle{Type: "rect", Rotate: ptr(45.0)}},
		{"rotate 음수", "rotate", `type="rect" rotate="-90"`,
			soml.ShapeStyle{Type: "rect", Rotate: ptr(-90.0)}},
		{"rotate 소수", "rotate", `type="rect" rotate="7.5"`,
			soml.ShapeStyle{Type: "rect", Rotate: ptr(7.5)}},

		{"flip h", "flip", `type="chevron" flip="h"`,
			soml.ShapeStyle{Type: "chevron", Flip: soml.FlipH}},
		{"flip v", "flip", `type="chevron" flip="v"`,
			soml.ShapeStyle{Type: "chevron", Flip: soml.FlipV}},
		{"flip 두 축", "flip", `type="chevron" flip="hv"`,
			soml.ShapeStyle{Type: "chevron", Flip: soml.FlipBoth}},

		{"pad 축약", "pad", `type="rect" pad="sm md"`,
			soml.ShapeStyle{Type: "rect", Box: soml.Box{Pad: &soml.Pad{
				Top: soml.SpaceLength("sm", 12), Right: soml.SpaceLength("md", 24),
				Bottom: soml.SpaceLength("sm", 12), Left: soml.SpaceLength("md", 24)}}}},
		{"w 수치", "w", `type="rect" w="220"`,
			soml.ShapeStyle{Type: "rect", Box: soml.Box{Width: ptr(soml.Px(220))}}},
		{"h fill", "h", `type="rect" h="fill"`,
			soml.ShapeStyle{Type: "rect", Box: soml.Box{Height: ptr(soml.Percent(100))}}},
		{"min-w", "min-w", `type="rect" min-w="120"`,
			soml.ShapeStyle{Type: "rect", Box: soml.Box{MinWidth: ptr(soml.Px(120))}}},
		{"min-h", "min-h", `type="rect" min-h="80"`,
			soml.ShapeStyle{Type: "rect", Box: soml.Box{MinHeight: ptr(soml.Px(80))}}},
		{"max-w", "max-w", `type="rect" max-w="360"`,
			soml.ShapeStyle{Type: "rect", Box: soml.Box{MaxWidth: ptr(soml.Px(360))}}},
		{"max-h", "max-h", `type="rect" max-h="240"`,
			soml.ShapeStyle{Type: "rect", Box: soml.Box{MaxHeight: ptr(soml.Px(240))}}},
		{"ratio", "ratio", `type="rect" ratio="16:9"`,
			soml.ShapeStyle{Type: "rect", Box: soml.Box{Ratio: &soml.Ratio{W: 16, H: 9}}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := resolvedShape(t, tc.attrs)
			if want := wantShapeStyle(tc.want); !reflect.DeepEqual(*got, want) {
				t.Errorf("도형 외관 = %s, want %s", show(*got), show(want))
			}
			if placement := resolvedElement(t, shapeSlide(tc.attrs), soml.ShapeElement).Placement(); placement != nil {
				t.Errorf("자식 배치가 붙었다: %+v", *placement)
			}
		})
	}

	covered := map[string]bool{}
	for _, tc := range cases {
		covered[tc.attr] = true
	}
	for _, attr := range soml.ShapeAttrs() {
		if !covered[attr] {
			t.Errorf("도형 속성 %q의 해석 테스트가 없다", attr)
		}
	}
}

// `type` 12종 전수.
func TestResolveShapeTypeVocabulary(t *testing.T) {
	for _, name := range soml.ShapeTypes() {
		t.Run(name, func(t *testing.T) {
			if got := resolvedShape(t, `type="`+name+`"`).Type; got != name {
				t.Errorf("type = %q, want %q", got, name)
			}
		})
	}
}

// shapeWith는 내용을 담은 도형 하나의 슬라이드다.
func shapeWith(attrs, children string) string {
	return "<slide id=\"shape\">\n  <shape " + attrs + ">" + children + "</shape>\n</slide>\n"
}

// 도형 값 층의 진단 전수 — 각 진단은 문제가 된 값(needle)을 2행에서 가리키고 고치는 방법을
// 담는다. needle이 없으면 태그 이름(2:4)을 가리킨다 (없는 속성·내용과의 조합). dropped가 있으면
// 틀린 값이 확정 외관에 실리지 않았는지도 본다.
func TestResolveShapeDiagnostics(t *testing.T) {
	baseBorder := componentShapeStyle("rect").Border
	keepsBorder := func(s *soml.ShapeStyle) bool { return reflect.DeepEqual(s.Border, baseBorder) }
	noType := func(s *soml.ShapeStyle) bool { return s.Type == "" }
	cases := []struct {
		name, attrs, children, needle string
		col                           int // needle로 셀 수 없는 위치
		contains                      []string
		dropped                       func(*soml.ShapeStyle) bool
	}{
		// type은 생략할 수 없다 — 조용히 사각형이 되면 저자의 오타와 구별되지 않는다.
		{name: "type 없음", attrs: `w="120"`, contains: []string{"type이 필요하다", "rect", "cloud"}, dropped: noType},
		{name: "type 어휘 밖", attrs: `type="elipse"`, needle: "elipse",
			contains: []string{"알 수 없는 도형 type", "ellipse"}, dropped: noType},
		{name: "type 값 없음", attrs: `type`, needle: "type", contains: []string{"type 속성에는 값이 필요하다"}, dropped: noType},
		{name: "type 빈 값", attrs: `type=""`, col: shapeCol(`type=""`, `""`) + 1,
			contains: []string{"type 속성이 비어 있다"}, dropped: noType},

		// 색 문법은 텍스트 color와 같은 코드(colorValue)를 지난다.
		{name: "fill 모르는 토큰", attrs: `type="rect" fill="brand"`, needle: "brand",
			contains: []string{"fill의 색 토큰", "테마에 없다", "surface", "#RRGGBB"}},
		{name: "fill 그라디언트 색 하나", attrs: `type="rect" fill="gradient(accent1)"`, needle: "gradient",
			contains: []string{"gradient는 색이 둘 이상"}},
		{name: "fill 잘못된 리터럴", attrs: `type="rect" fill="#12345"`, needle: "#12345",
			contains: []string{"#RRGGBB 또는 #RGB"}},
		{name: "border 모르는 토큰", attrs: `type="rect" border="1 solid brand"`, needle: "brand",
			contains: []string{"border의 색 토큰", "테마에 없다"}},
		{name: "border shade 백분율 없음", attrs: `type="rect" border="3 solid shade(accent1, 20)"`, needle: "shade",
			contains: []string{"shade의 두 번째 인자는 백분율"}},
		// 그라디언트는 스타일과 색을 함께 대신하므로 축약이 2조각이다.
		{name: "border 그라디언트 스타일 끼움", attrs: `type="rect" border="3 dashed gradient(accent1 muted)"`, needle: "3 dashed",
			contains: []string{"2조각", "스타일과 색을 함께 대신"}},

		// border 축약 — 조각 수 진단은 값 전체를, 나머지는 조각을 가리킨다. 틀린 인라인 값 대신
		// 컴포넌트 기본 테두리가 남는다.
		{name: "border 조각 부족", attrs: `type="rect" border="solid 3"`, needle: "solid 3",
			contains: []string{"3조각이다", "2조각", "1 solid muted"}, dropped: keepsBorder},
		{name: "border 조각 초과", attrs: `type="rect" border="1 solid muted inset"`, needle: "1 solid",
			contains: []string{"3조각이다", "4조각"}, dropped: keepsBorder},
		{name: "border 굵기 0", attrs: `type="rect" border="0 solid muted"`, needle: "0 solid",
			contains: []string{"0보다 커야 한다", `border="none"`}, dropped: keepsBorder},
		{name: "border 굵기 음수", attrs: `type="rect" border="-1 solid muted"`, needle: "-1",
			contains: []string{"0보다 커야 한다"}, dropped: keepsBorder},
		{name: "border 굵기 단위", attrs: `type="rect" border="1px solid muted"`, needle: "1px",
			contains: []string{"단위 없는 논리 px 수치", `"px"`}, dropped: keepsBorder},
		{name: "border 모르는 스타일", attrs: `type="rect" border="1 dashdot muted"`, needle: "dashdot",
			contains: []string{"알 수 없는 border 스타일", "solid, dashed, dotted"}, dropped: keepsBorder},

		// radius는 형제 type을 본다 — 다른 프리셋엔 반경 조정값이 없어 웹만 둥글어진다. 진단은
		// 고칠 것이 형제 속성이라 속성 이름을 가리킨다.
		{name: "radius는 rounded-rect 전용", attrs: `type="rect" radius="lg"`, needle: "radius",
			contains: []string{"radius는 type=\"rounded-rect\"에서만 쓴다", "지금은 \"rect\""},
			dropped:  func(s *soml.ShapeStyle) bool { return s.Radius == nil }},
		{name: "radius 모르는 토큰", attrs: `type="rounded-rect" radius="huge"`, needle: "huge",
			contains: []string{"알 수 없는 radius 토큰", "sm, md, lg", "수치"}},
		{name: "radius 단위", attrs: `type="rounded-rect" radius="8px"`, needle: "8px", contains: []string{"단위 없는 논리 px 수치"}},
		{name: "radius 음수", attrs: `type="rounded-rect" radius="-4"`, needle: "-4", contains: []string{"0보다 작을 수 없다"}},
		{name: "shadow 모르는 토큰", attrs: `type="rect" shadow="hard"`, needle: "hard",
			contains: []string{"알 수 없는 shadow 토큰", "soft", "수치가 아니라 토큰"}},
		{name: "shadow 수치 축약", attrs: `type="rect" shadow="0 2 8 #000000"`, needle: "0 2 8",
			contains: []string{"알 수 없는 shadow 토큰"}},
		{name: "opacity 백분율", attrs: `type="rect" opacity="50%"`, needle: "50%",
			contains: []string{"opacity에 %는 쓸 수 없다", "0~1"}},
		{name: "opacity 범위 초과", attrs: `type="rect" opacity="1.5"`, needle: "1.5", contains: []string{"0~1 사이다"}},
		{name: "opacity 음수", attrs: `type="rect" opacity="-0.2"`, needle: "-0.2", contains: []string{"0~1 사이다"}},
		{name: "opacity 수치 아님", attrs: `type="rect" opacity="반쯤"`, needle: "반쯤", contains: []string{"수치가 아니다"}},
		{name: "rotate 단위", attrs: `type="rect" rotate="45deg"`, needle: "45deg",
			contains: []string{"단위 없는 각도 수치", `"deg"`}},
		{name: "rotate 수치 아님", attrs: `type="rect" rotate="비스듬히"`, needle: "비스듬히",
			contains: []string{"수치가 아니다", "음수 가능"}},
		{name: "flip 어휘 밖", attrs: `type="chevron" flip="both"`, needle: "both", contains: []string{"알 수 없는 flip", "h, v, hv"}},
		{name: "flip 공백 표기", attrs: `type="chevron" flip="h v"`, needle: "h v", contains: []string{"알 수 없는 flip"}},

		// 배경 모드(배치 자식) 도형의 rotate는 에러다 (ADR 0049 결정 6) — PPTX의 형제 도형은 남의
		// 회전을 물려받지 않아 두 타깃이 다른 그림이 된다.
		{name: "배치를 담은 도형의 rotate", attrs: `type="rect" rotate="12"`, children: `<col gap="sm"><p>본문</p></col>`,
			needle: "rotate", contains: []string{"배치를 담은 도형은 rotate를 쓸 수 없다", "자식은 형제 도형이 되어"},
			dropped: func(s *soml.ShapeStyle) bool { return s.Rotate == nil }},
		// 텍스트 영역이 박스보다 좁은 프리셋은 내용이 컨테이너여도 확정 높이를 요구한다 (ADR 0040·0049).
		{name: "inset 프리셋의 컨테이너 내용", attrs: `type="ellipse"`, children: `<col gap="sm"><p>라벨</p></col>`,
			contains: []string{"h나 ratio가 필요하다", "ellipse"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			src := shapeSlide(tc.attrs)
			if tc.children != "" {
				src = shapeWith(tc.attrs, tc.children)
			}
			deck, diag := oneDiag(t, src)
			col := tc.col
			switch {
			case col != 0:
			case tc.needle == "":
				col = 4
			default:
				col = shapeCol(tc.attrs, tc.needle)
			}
			assertDiag(t, diag, 2, col, tc.contains...)
			if tc.dropped != nil {
				if style := findElement(t, deck, soml.ShapeElement).ShapeStyle(); style == nil || !tc.dropped(style) {
					t.Errorf("무효한 값이 실렸다: %s", show(style))
				}
			}
		})
	}
}

// 조각 셋을 모두 진단한다 — 순서를 어기면 조각마다, 셋 다 틀리면 셋 모두 소스 순서로 난다.
// 같은 도형에서 type이 이미 틀렸으면 radius 진단은 얹지 않는다 (고칠 곳이 하나다).
func TestResolveShapeReportsEveryBadPiece(t *testing.T) {
	_, diags := resolveSlide(t, shapeSlide(`type="rect" border="solid 3 muted"`))
	if len(diags) != 2 {
		t.Fatalf("진단 %d개, want 2개: %v", len(diags), diags)
	}
	assertDiag(t, diags[0], 2, shapeCol(`type="rect" border="solid 3 muted"`, "solid"), "굵기가 수치가 아니다", "조각 순서")

	_, diags = resolveSlide(t, shapeSlide(`type="rect" border="0 dashdot brand"`))
	if len(diags) != 3 {
		t.Fatalf("진단 %d개, want 3개: %v", len(diags), diags)
	}
	for i := 1; i < len(diags); i++ {
		if diags[i-1].Span.Start.Offset > diags[i].Span.Start.Offset {
			t.Errorf("진단이 소스 순서가 아니다: %v", diags)
		}
	}

	if _, diags := resolveSlide(t, shapeSlide(`type="elipse" radius="lg"`)); len(diags) != 1 {
		t.Errorf("진단 %d개, want 1개 (type 하나): %v", len(diags), diags)
	}
}

// 그라디언트·색 변형 문법이 확정 색 모델로 해석된다 (ADR 0067 결정 5). 토큰은 이름으로 남는다.
// 그라디언트 테두리(`<굵기> gradient(…)`)는 스타일을 실선으로 함께 정한다.
func TestResolveColorFunctions(t *testing.T) {
	cases := []struct{ name, fill, want string }{
		{"그라디언트 토큰 3색 기본각", `gradient(accent1 surface muted)`,
			"gradient(token(accent1) token(surface) token(muted) @180)"},
		{"그라디언트 토큰+리터럴 각도", `gradient(accent1 #FF0000, 45deg)`,
			"gradient(token(accent1) #FF0000 @45)"},
		{"그라디언트 두 색 세로", `gradient(accent1 bg, 0deg)`,
			"gradient(token(accent1) token(bg) @0)"},
		{"shade 토큰", `shade(accent1, 40%)`, "shade(token(accent1),40)"},
		{"tint 토큰", `tint(muted, 25%)`, "tint(token(muted),25)"},
		{"alpha 토큰", `alpha(fg, 0.6)`, "alpha(token(fg),0.6)"},
		{"shade 리터럴", `shade(#0F65F9, 30%)`, "shade(#0F65F9,30)"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			style := resolvedShape(t, `type="rect" fill="`+tc.fill+`"`)
			if style.Fill == nil {
				t.Fatal("채움이 해석되지 않았다")
			}
			if got := showFill(*style.Fill); got != tc.want {
				t.Errorf("fill = %q, want %q", got, tc.want)
			}
		})
	}

	border := resolvedShape(t, `type="rect" border="3 gradient(accent1 muted)"`).Border
	if border == nil || border.Width != 3 || border.Style != soml.BorderSolid ||
		showColor(border.Color) != "gradient(token(accent1) token(muted) @180)" {
		t.Errorf("그라디언트 테두리 = %s, want 3/solid/2색 그라디언트", show(border))
	}
}

// 텍스트 영역이 박스보다 좁은 프리셋(soml.ShapeInsetText)은 내용을 담을 때 확정 높이를 요구한다
// (ADR 0040) — 12종 전수로 요구하는 쪽과 아닌 쪽을 함께 못 박는다. 규칙이 넓어지면 내용이 높이를
// 정하는 카드가 죽는다.
func TestResolveShapeInsetTextNeedsDefiniteHeight(t *testing.T) {
	for _, shapeType := range soml.ShapeTypes() {
		t.Run(shapeType, func(t *testing.T) {
			attrs := `type="` + shapeType + `"`
			_, diags := resolveSlide(t, shapeWith(attrs, "<p>라벨</p>"))
			if !soml.ShapeInsetText(shapeType) {
				if len(diags) != 0 {
					t.Fatalf("진단 %d개, want 0개 (이 프리셋의 텍스트 영역은 박스다): %v", len(diags), diags)
				}
				return
			}
			if len(diags) != 1 {
				t.Fatalf("진단 %d개, want 1개: %v", len(diags), diags)
			}
			assertDiag(t, diags[0], 2, 4, "h나 ratio가 필요하다", shapeType, "기준이 순환한다")
			// 높이가 확정이거나 내용이 없으면 조용하다.
			for _, quiet := range []string{
				shapeWith(attrs+` h="120"`, "<p>라벨</p>"),
				shapeWith(attrs+` h="fill"`, "<p>라벨</p>"),
				shapeWith(attrs+` ratio="1:1"`, "<p>라벨</p>"),
				shapeWith(attrs+` h="200"`, `<col gap="sm"><p>라벨</p></col>`),
				shapeWith(attrs, "\n  "),
			} {
				if _, diags := resolveSlide(t, quiet); len(diags) != 0 {
					t.Errorf("진단 %d개, want 0개 (%s): %v", len(diags), quiet, diags)
				}
			}
		})
	}
	// 흡수 모드(텍스트 자식)와 빈 도형의 회전은 그대로다 — 문단은 도형과 함께 돌아간다.
	for _, quiet := range []string{shapeWith(`type="rect" rotate="12"`, "<p>본문</p>"), shapeSlide(`type="rect" rotate="12"`)} {
		if _, diags := resolveSlide(t, quiet); len(diags) != 0 {
			t.Errorf("진단 %d개, want 0개 (%s): %v", len(diags), quiet, diags)
		}
	}
}

// 도형의 박스 값 문법은 컨테이너와 같은 확정 값에 도달한다 (soml.Box를 공유한 이유).
func TestResolveShapeBoxSharesContainerSyntax(t *testing.T) {
	decls := []string{`pad="lg"`, `pad="1 2 3 4%"`, `w="fill"`, `h="240"`, `min-w="120"`,
		`min-h="80"`, `max-w="360"`, `max-h="240"`, `ratio="4:3"`}
	for _, decl := range decls {
		t.Run(decl, func(t *testing.T) {
			shape := resolvedShape(t, `type="rect" `+decl).Box
			container := resolvedElement(t, containerSrc("row", decl), "row").Container()
			if container == nil {
				t.Fatal("<row>의 배치가 해석되지 않았다")
			}
			if !reflect.DeepEqual(shape, container.Box) {
				t.Errorf("도형 박스 = %+v, 컨테이너 박스 = %+v — 같은 표기가 다르게 해석됐다",
					shape, container.Box)
			}
		})
	}

	covered := map[string]bool{}
	for _, decl := range decls {
		covered[strings.SplitN(decl, "=", 2)[0]] = true
	}
	for _, attr := range soml.BoxAttrs() {
		if !covered[attr] {
			t.Errorf("박스 속성 %q의 공유 테스트가 없다", attr)
		}
	}
}

// 도형 id 중복 검사의 범위는 슬라이드다 (참조가 슬라이드를 넘지 않는다). 진단은 두 번째 자리에서
// 나고 처음 자리를 문구에 실으며, 중복은 확정 값에 실리지 않는다.
func TestResolveShapeDuplicateID(t *testing.T) {
	src := "<slide id=\"shape\">\n  <shape type=\"rect\" id=\"db\"/>\n  <shape type=\"ellipse\" id=\"db\"/>\n</slide>\n"
	deck, diag := oneDiag(t, src)
	assertDiag(t, diag, 3, 29, `id "db"가 이 슬라이드에 이미 있다`, "2:", "참조가 어느 도형인지")
	shapes := shapeStyles(t, deck)
	if len(shapes) != 2 || shapes[0].ID != "db" || shapes[1].ID != "" {
		t.Errorf("id = %q, %q, want \"db\", \"\"", shapes[0].ID, shapes[1].ID)
	}
}

// anchor는 텍스트 외관이고 어휘는 물리 이름이다 (세로축은 문단 방향으로 뒤집히지 않는다). 속성이
// 없으면 빈 값으로 남는다 — 역할 기본값이 채우지 않는 유일한 필드다.
func TestResolveTextAnchor(t *testing.T) {
	for _, anchor := range soml.TextAnchors() {
		src := "<slide id=\"shape\">\n  <shape type=\"rect\"><text anchor=\"" + string(anchor) + "\">내용</text></shape>\n</slide>\n"
		if style := resolvedElement(t, src, soml.TextElement).TextStyle(); style == nil || style.Anchor != anchor {
			t.Errorf("anchor = %+v, want %q", style, anchor)
		}
	}
	blockSrc := "<slide id=\"shape\">\n  <shape type=\"rect\"><p anchor=\"bottom\">내용</p></shape>\n</slide>\n"
	if style := resolvedElement(t, blockSrc, "p").TextStyle(); style == nil || style.Anchor != soml.AnchorBottom {
		t.Errorf("<p>의 anchor가 해석되지 않았다: %+v", style)
	}
	if style := firstStyle(t, "<slide id=\"t\">\n  <text>내용</text>\n</slide>\n"); style.Anchor != "" {
		t.Errorf("anchor = %q, want 빈 문자열", style.Anchor)
	}

	for _, tc := range []struct {
		value    string
		contains []string
	}{
		{"start", []string{"anchor는 세로 이름을 쓴다", `"top"`}},
		{"end", []string{"anchor는 세로 이름을 쓴다", `"bottom"`}},
		{"middle", []string{"알 수 없는 anchor", "top, center, bottom"}},
	} {
		t.Run(tc.value, func(t *testing.T) {
			deck, diag := oneDiag(t, "<slide id=\"t\">\n  <text anchor=\""+tc.value+"\">내용</text>\n</slide>\n")
			assertDiag(t, diag, 0, 0, tc.contains...)
			if style := findElement(t, deck, soml.TextElement).TextStyle(); style == nil || style.Anchor != "" {
				t.Errorf("무효한 anchor가 실렸다: %+v", style)
			}
		})
	}
}

// 도형 안 <p>는 슬라이드 직속과 같은 외관이다 — 상속은 이 언어에 없다.
func TestResolveShapeChildTextUnchanged(t *testing.T) {
	inShape := resolvedElement(t,
		"<slide id=\"s\">\n  <shape type=\"rect\"><p>내용</p></shape>\n</slide>\n", "p").TextStyle()
	direct := tagStyle(t, "<slide id=\"s\">\n  <p>내용</p>\n</slide>\n", "p")
	if inShape == nil || !reflect.DeepEqual(*inShape, *direct) {
		t.Errorf("도형 안 <p> 외관 = %+v, 슬라이드 직속 = %+v", inShape, *direct)
	}
}

// 외관(캐스케이드)·박스(도형 자신)·배치(부모 안의 자리)는 소유자가 다른 세 채널이다 — 하나가
// 다른 하나를 지우면 산출물에서 그 채널만 조용히 사라진다.
func TestResolveShapeKeepsPlacement(t *testing.T) {
	src := "<slide id=\"s\">\n  <row><shape type=\"rect\" fill=\"surface\" pad=\"md\" grow=\"1\"/></row>\n</slide>\n"
	elem := resolvedElement(t, src, soml.ShapeElement)
	style, placement := elem.ShapeStyle(), elem.Placement()
	want := wantShapeStyle(soml.ShapeStyle{Type: "rect", Fill: ptr(soml.ColorFill(soml.TokenColor("surface"))),
		Box: soml.Box{Pad: &soml.Pad{Top: soml.SpaceLength("md", 24), Right: soml.SpaceLength("md", 24),
			Bottom: soml.SpaceLength("md", 24), Left: soml.SpaceLength("md", 24)}}})
	if style == nil || !reflect.DeepEqual(*style, want) {
		t.Errorf("도형 외관 = %s, want %s", show(style), show(want))
	}
	if placement == nil || placement.Grow == nil || *placement.Grow != 1 {
		t.Errorf("배치가 없다: %+v", placement)
	}
}

// shapeStyles는 덱의 모든 도형 외관을 문서 순서로 모은다.
func shapeStyles(t *testing.T, deck *soml.Deck) []*soml.ShapeStyle {
	t.Helper()
	var styles []*soml.ShapeStyle
	for _, slide := range deck.Slides {
		for _, node := range slide.Content {
			walkShapes(node, func(style *soml.ShapeStyle) { styles = append(styles, style) })
		}
	}
	if len(styles) == 0 {
		t.Fatal("도형이 없다")
	}
	return styles
}

// walkShapes는 트리의 도형 외관을 문서 순서로 훑는다.
func walkShapes(node soml.Node, visit func(*soml.ShapeStyle)) {
	elem, ok := node.(*soml.Element)
	if !ok {
		return
	}
	if style := elem.ShapeStyle(); style != nil {
		visit(style)
	}
	for _, child := range elem.Children {
		walkShapes(child, visit)
	}
}

// showFill·showColor는 색 모델의 표기다 — 토큰·리터럴·변형·그라디언트를 구별해 읽힌다.
func showFill(fill soml.Fill) string {
	if fill.None {
		return "none"
	}
	return showColor(fill.Color)
}

func showColor(color soml.Color) string {
	if color.IsGradient() {
		stops := make([]string, len(color.Gradient.Stops))
		for i, s := range color.Gradient.Stops {
			stops[i] = showColor(s)
		}
		return fmt.Sprintf("gradient(%s @%g)", strings.Join(stops, " "), color.Gradient.Angle)
	}
	base := "#" + color.RGB
	if color.IsToken() {
		base = "token(" + color.Token + ")"
	}
	if color.Mod != nil {
		return fmt.Sprintf("%s(%s,%g)", color.Mod.Kind, base, color.Mod.Amount)
	}
	return base
}
