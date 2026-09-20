package core

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/k2sebeom/balpyo/soml"
)

// shapeSlide는 도형 하나(자기 닫힘)를 담은 슬라이드 원문이다. 들여쓰기를 2칸으로 고정하는
// 이유는 layoutSlide와 같다 — 진단 위치 단정이 표에서 읽히게 하기 위함이다 (도형은 항상
// 2행 3열에서 시작한다).
func shapeSlide(attrs string) string {
	return "<slide id=\"shape\">\n  <shape " + attrs + "/>\n</slide>\n"
}

// shapeCol은 속성 문자열 안 needle이 놓이는 열이다 — 열을 손으로 세면 표가 늘 때 위치
// 단정이 조용히 어긋난다. 속성은 ASCII이므로 바이트 오프셋이 곧 룬 열이다 (soml.Pos.Col).
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

// wantShapeStyle는 표의 항목을 기대값으로 바꾼다 — 항목은 **컴포넌트 기본값 위의 델타**다.
//
// 캐스케이드가 붙은 뒤(항목 03) 속성 없는 도형도 fill·border를 갖기 때문이다. 델타를
// 손으로 펼치면 표의 모든 줄에 같은 세 필드가 반복되어 정작 그 줄이 무엇을 시험하는지가
// 묻히고, 기본값을 고칠 때 표 전체를 고쳐야 한다. 얹는 데 **운영 코드의 mergeShape**를
// 쓰는 이유는 병합 규칙을 테스트가 다시 쓰지 않기 위함이다.
func wantShapeStyle(delta soml.ShapeStyle) soml.ShapeStyle {
	style := componentShapeStyle(delta.Type)
	var flip *soml.Flip
	if delta.Flip != "" {
		flip = &delta.Flip
	}
	mergeShape(&style, shapePatch{fill: delta.Fill, border: delta.Border, radius: delta.Radius,
		shadow: delta.Shadow, opacity: delta.Opacity, rotate: delta.Rotate, flip: flip})
	// 식별자·대체 텍스트·박스는 캐스케이드 밖이라 병합이 다루지 않는다 (shapePatch의 근거).
	style.ID, style.Alt, style.Box = delta.ID, delta.Alt, delta.Box
	return style
}

// PRD §7.6 표의 shape 행을 전수로 못 박는다 — 속성 → **확정 ShapeStyle 필드**. 렌더러는
// Attrs를 다시 읽지 않으므로(soml.Appearance의 계약) 이 표가 곧 CSS·OOXML의 입력이다.
//
// `want`는 컴포넌트 기본값 위의 **델타**다 (wantShapeStyle) — 인라인 단이 아래 단을 이기는
// 것은 캐스케이드 테스트(cascade_test.go)가 따로 못 박고, 여기서는 값 문법이 도달하는
// 필드를 본다.
func TestResolveShapeStyle(t *testing.T) {
	softShadow := minimalShadows()[0]
	cases := []struct {
		name  string
		attr  string
		attrs string
		want  soml.ShapeStyle
	}{
		// 고유 — type은 생략할 수 없고(아래 별도 테스트), id는 산출물에 아무것도 그리지 않지만
		// 참조(Iteration 7)의 이름이라 확정 값에 남는다.
		{"type", "type", `type="ellipse"`, soml.ShapeStyle{Type: "ellipse"}},
		{"id", "id", `type="rect" id="db"`, soml.ShapeStyle{Type: "rect", ID: "db"}},
		// alt은 **그림의 이름**이고 안의 글을 대체하지 않는다 (ADR 0058) — 빈 값의 뜻은
		// alt_test.go가 따로 못 박는다.
		{"alt", "alt", `type="arrow-right" alt="배포 흐름의 마지막 단계"`,
			soml.ShapeStyle{Type: "arrow-right", Alt: "배포 흐름의 마지막 단계"}},

		// variant는 테마의 외관 번들이다 — 캐스케이드 둘째 단이므로 확정 값이 표의 값과
		// **다른 자리**에서 온다 (여기서는 card가 정하는 두 필드만 델타로 적는다).
		{"variant", "variant", `type="rounded-rect" variant="card"`,
			soml.ShapeStyle{Type: "rounded-rect", Fill: ptr(soml.ColorFill(soml.TokenColor("bg"))),
				Radius: ptr(soml.TokenRadius("lg", 24))}},

		// 색 — 토큰은 **이름으로** 남고(schemeClr 참조가 되어야 한다) 리터럴은 대문자 6자리로
		// 정규화된다. `none`은 색이 아니라 세 번째 상태다.
		{"fill 토큰", "fill", `type="rect" fill="surface"`,
			soml.ShapeStyle{Type: "rect", Fill: ptr(soml.ColorFill(soml.TokenColor("surface")))}},
		{"fill 리터럴 소문자", "fill", `type="rect" fill="#0f65f9"`,
			soml.ShapeStyle{Type: "rect", Fill: ptr(soml.ColorFill(soml.LiteralColor("0F65F9")))}},
		{"fill 리터럴 3자리", "fill", `type="rect" fill="#fff"`,
			soml.ShapeStyle{Type: "rect", Fill: ptr(soml.ColorFill(soml.LiteralColor("FFFFFF")))}},
		{"fill none", "fill", `type="rect" fill="none"`,
			soml.ShapeStyle{Type: "rect", Fill: ptr(soml.NoFill())}},

		// border 축약 — `<굵기> <스타일> <색>`이 세 필드로 풀린다 (렌더러가 축약을 다시
		// 해석하지 않는다).
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

		// radius — 토큰이면 이름이 남고(웹이 변수 참조로 낸다) 수치면 px다. rounded-rect에서만
		// 뜻이 있다 (아래 별도 테스트).
		{"radius 토큰", "radius", `type="rounded-rect" radius="lg"`,
			soml.ShapeStyle{Type: "rounded-rect", Radius: ptr(soml.TokenRadius("lg", 24))}},
		{"radius 수치", "radius", `type="rounded-rect" radius="32"`,
			soml.ShapeStyle{Type: "rounded-rect", Radius: ptr(soml.PxRadius(32))}},
		{"radius 0", "radius", `type="rounded-rect" radius="0"`,
			soml.ShapeStyle{Type: "rounded-rect", Radius: ptr(soml.PxRadius(0))}},

		// shadow는 토큰만 받으므로 확정 값이 곧 테마 표의 항목이다 (soml.ShadowToken).
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

		// 박스 — 레이아웃 컨테이너와 **같은 헬퍼**를 지난다 (아래 대조 테스트가 짝이다).
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
				t.Errorf("도형 외관 = %s, want %s", showShape(*got), showShape(want))
			}
			// 외관은 자식 배치가 아니다 — 둘이 섞이면 렌더러가 외관을 자기 자리로 낸다.
			if placement := resolvedElement(t, shapeSlide(tc.attrs), soml.ShapeElement).Placement(); placement != nil {
				t.Errorf("자식 배치가 붙었다: %+v", *placement)
			}
		})
	}

	// 표가 카탈로그 전체를 덮는지 — 파서가 연 속성을 값 층이 해석하지 않으면 저자가 쓴 외관이
	// 조용히 사라진다 (컨테이너 속성 커버리지 테스트와 같은 장치).
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

// `type` 12종 전수 — 목록이 soml에 있고 검사가 core라는 구조(항목 01)의 실행 가능한 형태다.
func TestResolveShapeTypeVocabulary(t *testing.T) {
	for _, name := range soml.ShapeTypes() {
		t.Run(name, func(t *testing.T) {
			if got := resolvedShape(t, `type="`+name+`"`).Type; got != name {
				t.Errorf("type = %q, want %q", got, name)
			}
		})
	}
}

// `type`은 생략할 수 없고 어휘 밖 값도 에러다 — 도형의 종류는 추측할 수 없고, 조용히
// 사각형이 되면 저자의 오타와 구별되지 않는다 (resolver.shapeType의 근거).
func TestResolveShapeTypeIsRequired(t *testing.T) {
	cases := []struct {
		name     string
		attrs    string
		col      int
		contains []string
	}{
		// 속성이 없으면 가리킬 값이 없다 — 진단은 **태그 이름**을 가리킨다 (파서의 어휘
		// 진단과 같은 자리).
		{"type 없음", `w="120"`, 4, []string{"type이 필요하다", "rect", "cloud"}},
		{"어휘 밖", `type="elipse"`, shapeCol(`type="elipse"`, "elipse"),
			[]string{"알 수 없는 도형 type", "ellipse"}},
		{"값 없음", `type`, shapeCol(`type`, "type"), []string{"type 속성에는 값이 필요하다"}},
		{"빈 값", `type=""`, shapeCol(`type=""`, `""`) + 1, []string{"type 속성이 비어 있다"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			deck, diags := resolveSlide(t, shapeSlide(tc.attrs))
			if len(diags) != 1 {
				t.Fatalf("진단 %d개, want 1개: %v", len(diags), diags)
			}
			assertShapeDiag(t, diags[0], tc.col, tc.contains...)
			// 틀린 값은 확정 외관에 실리지 않는다 — 그래야 렌더러의 "매핑이 없다" 가드가
			// 마지막 그물로 남는다.
			if style := findElement(t, deck, soml.ShapeElement).ShapeStyle(); style == nil || style.Type != "" {
				t.Errorf("무효한 type이 실렸다: %+v", style)
			}
		})
	}
}

// 색 문법은 텍스트 `color`와 **같은 코드**를 지난다 (colorValue) — 같은 값이 자리에 따라
// 다르게 읽히면 두 산출물이 갈라진다.
func TestResolveShapeColorErrors(t *testing.T) {
	cases := []struct {
		name     string
		attrs    string
		needle   string
		contains []string
	}{
		{"fill 모르는 토큰", `type="rect" fill="brand"`, "brand",
			[]string{"fill의 색 토큰", "테마에 없다", "surface", "#RRGGBB"}},
		{"fill 그라디언트 색 하나", `type="rect" fill="gradient(accent1)"`, "gradient",
			[]string{"gradient는 색이 둘 이상"}},
		{"fill 잘못된 리터럴", `type="rect" fill="#12345"`, "#12345",
			[]string{"#RRGGBB 또는 #RGB"}},
		{"border 모르는 토큰", `type="rect" border="1 solid brand"`, "brand",
			[]string{"border의 색 토큰", "테마에 없다"}},
		{"border shade 백분율 없음", `type="rect" border="3 solid shade(accent1, 20)"`, "shade",
			[]string{"shade의 두 번째 인자는 백분율"}},
		// 그라디언트가 스타일을 함께 대신하므로 축약은 2조각이다 (`<굵기> gradient(…)`). 스타일
		// 조각을 끼우면 3조각이 되어 그 갈래의 조각 수 에러가 난다 (gradientBorder).
		{"border 그라디언트 스타일 끼움", `type="rect" border="3 dashed gradient(accent1 muted)"`, "3 dashed",
			[]string{"2조각", "스타일과 색을 함께 대신"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, diags := resolveSlide(t, shapeSlide(tc.attrs))
			if len(diags) != 1 {
				t.Fatalf("진단 %d개, want 1개: %v", len(diags), diags)
			}
			assertShapeDiag(t, diags[0], shapeCol(tc.attrs, tc.needle), tc.contains...)
		})
	}
}

// TestResolveColorFunctions는 그라디언트·색 변형 문법이 확정 색 모델로 해석되는지 못 박는다
// (Iteration 8, PRD §7.6·§13-5). 토큰은 이름으로 남는다 (schemeClr 보존) — showColor가
// 토큰/리터럴/변형/그라디언트를 구별해 표기한다.
func TestResolveColorFunctions(t *testing.T) {
	cases := []struct {
		name string
		fill string
		want string
	}{
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
			deck, diags := resolveSlide(t, shapeSlide(`type="rect" fill="`+tc.fill+`"`))
			if len(diags) != 0 {
				t.Fatalf("진단이 났다: %v", diags)
			}
			style := findElement(t, deck, soml.ShapeElement).ShapeStyle()
			if style == nil || style.Fill == nil {
				t.Fatal("채움이 해석되지 않았다")
			}
			if got := showFill(*style.Fill); got != tc.want {
				t.Errorf("fill = %q, want %q", got, tc.want)
			}
		})
	}
}

// TestResolveGradientBorder는 그라디언트 테두리(`<굵기> gradient(…)`)가 굵기·실선·색으로
// 확정되는지 못 박는다 — 그라디언트가 스타일을 실선으로 함께 정한다.
func TestResolveGradientBorder(t *testing.T) {
	deck, diags := resolveSlide(t, shapeSlide(`type="rect" border="3 gradient(accent1 muted)"`))
	if len(diags) != 0 {
		t.Fatalf("진단이 났다: %v", diags)
	}
	style := findElement(t, deck, soml.ShapeElement).ShapeStyle()
	if style == nil || style.Border == nil {
		t.Fatal("테두리가 해석되지 않았다")
	}
	if style.Border.Width != 3 {
		t.Errorf("굵기 = %g, want 3", style.Border.Width)
	}
	if style.Border.Style != soml.BorderSolid {
		t.Errorf("스타일 = %q, want solid (그라디언트는 실선 고정)", style.Border.Style)
	}
	if got := showColor(style.Border.Color); got != "gradient(token(accent1) token(muted) @180)" {
		t.Errorf("색 = %q, want 그라디언트 2색", got)
	}
}

// border 축약의 조각 수·순서·굵기를 못 박는다. 순서를 고정한 근거는 진단이다 (resolver.border) —
// 자리가 없으면 "무엇이 빠졌는가"를 말할 수 없다.
func TestResolveShapeBorderShorthand(t *testing.T) {
	cases := []struct {
		name     string
		attrs    string
		needle   string
		diags    int
		contains []string
	}{
		{"조각 부족", `type="rect" border="solid 3"`, `"solid 3"`, 1,
			[]string{"3조각이다", "2조각", "1 solid muted"}},
		{"조각 초과", `type="rect" border="1 solid muted inset"`, `"1 solid`, 1,
			[]string{"3조각이다", "4조각"}},
		// 순서를 어기면 조각마다 진단이 난다 — 자리가 정해져 있어야 "무엇이 빠졌는가"를
		// 말할 수 있고, 조각 셋을 모두 진단하는 것이 값 층의 정책이다.
		{"순서 위반", `type="rect" border="solid 3 muted"`, "solid", 2,
			[]string{"굵기가 수치가 아니다", "조각 순서"}},
		{"굵기 0", `type="rect" border="0 solid muted"`, "0 solid", 1,
			[]string{"0보다 커야 한다", `border="none"`}},
		{"굵기 음수", `type="rect" border="-1 solid muted"`, "-1", 1,
			[]string{"0보다 커야 한다"}},
		{"굵기 단위", `type="rect" border="1px solid muted"`, "1px", 1,
			[]string{"단위 없는 논리 px 수치", `"px"`}},
		{"모르는 스타일", `type="rect" border="1 dashdot muted"`, "dashdot", 1,
			[]string{"알 수 없는 border 스타일", "solid, dashed, dotted"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			deck, diags := resolveSlide(t, shapeSlide(tc.attrs))
			if len(diags) != tc.diags {
				t.Fatalf("진단 %d개, want %d개: %v", len(diags), tc.diags, diags)
			}
			col := shapeCol(tc.attrs, tc.needle)
			if strings.HasPrefix(tc.needle, `"`) {
				// 조각 수 진단은 값 전체를 가리킨다 — 인용 부호 안쪽이 값의 시작이다.
				col++
			}
			assertShapeDiag(t, diags[0], col, tc.contains...)
			// 틀린 인라인 값은 실리지 않는다 — 아래 단(컴포넌트 기본 테두리)이 남는다.
			style := findElement(t, deck, soml.ShapeElement).ShapeStyle()
			if want := componentShapeStyle("rect").Border; style == nil ||
				!reflect.DeepEqual(style.Border, want) {
				t.Errorf("border = %+v, want 컴포넌트 기본값 %+v", style, want)
			}
		})
	}

	// 조각 셋을 모두 진단한다 — 진단 하나가 다른 조각의 진단을 삼키면 저자가 빌드를 조각
	// 수만큼 돌게 된다 (값 층 전체의 정책).
	_, diags := resolveSlide(t, shapeSlide(`type="rect" border="0 dashdot brand"`))
	if len(diags) != 3 {
		t.Fatalf("진단 %d개, want 3개: %v", len(diags), diags)
	}
	for i := 1; i < len(diags); i++ {
		if diags[i-1].Span.Start.Offset > diags[i].Span.Start.Offset {
			t.Errorf("진단이 소스 순서가 아니다: %v", diags)
		}
	}
}

// radius는 **형제 속성 `type`을 보는 검증**이다 — 다른 프리셋에는 반경 조정값이 없어 웹만
// 둥글어지고 PPTX는 각진 채 남는다. 이 테스트가 그 조용한 갈라짐의 회귀 장치다.
func TestResolveShapeRadiusNeedsRoundedRect(t *testing.T) {
	attrs := `type="rect" radius="lg"`
	deck, diags := resolveSlide(t, shapeSlide(attrs))
	if len(diags) != 1 {
		t.Fatalf("진단 %d개, want 1개: %v", len(diags), diags)
	}
	// 진단은 속성 **이름**을 가리킨다 — 고칠 것이 값이 아니라 형제 속성이기 때문이다
	// (span이 부모를 볼 때와 같은 태도).
	assertShapeDiag(t, diags[0], shapeCol(attrs, "radius"),
		"radius는 type=\"rounded-rect\"에서만 쓴다", "지금은 \"rect\"")
	if style := findElement(t, deck, soml.ShapeElement).ShapeStyle(); style == nil || style.Radius != nil {
		t.Errorf("무효한 radius가 실렸다: %+v", style)
	}

	// type이 아예 해석되지 않았으면 진단은 하나다 — 고칠 곳이 하나이므로 같은 도형에 두
	// 번째 진단을 얹지 않는다.
	if _, diags := resolveSlide(t, shapeSlide(`type="elipse" radius="lg"`)); len(diags) != 1 {
		t.Errorf("진단 %d개, want 1개 (type 하나): %v", len(diags), diags)
	}
}

// 텍스트 영역이 박스보다 좁은 프리셋(soml.ShapeInsetText)은 **내용을 담을 때** 확정 높이를
// 요구한다 (ADR 0040) — 프리셋 12종을 전수로 지나며 요구하는 쪽과 요구하지 않는 쪽을 함께
// 못 박는다. 규칙이 넓어지면(예: rounded-rect까지) 내용이 높이를 정하는 카드가 죽는다.
func TestResolveShapeInsetTextNeedsDefiniteHeight(t *testing.T) {
	shapeWith := func(attrs, children string) string {
		return "<slide id=\"shape\">\n  <shape " + attrs + ">" + children + "</shape>\n</slide>\n"
	}
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
			// 진단은 **요소 이름**을 가리킨다 — 고칠 것이 어느 속성의 값이 아니라 없는 속성이다
			// (type 누락 진단과 같은 자리).
			assertShapeDiag(t, diags[0], 4, "h나 ratio가 필요하다", shapeType, "기준이 순환한다")

			// 높이가 확정이면(h·ratio) 조용하고, 내용이 없으면 애초에 규칙이 걸리지 않는다 —
			// 텍스트 없는 도형은 영역을 쓰지 않으므로 높이를 요구할 근거가 없다.
			for _, quiet := range []string{
				shapeWith(attrs+` h="120"`, "<p>라벨</p>"),
				shapeWith(attrs+` h="fill"`, "<p>라벨</p>"),
				shapeWith(attrs+` ratio="1:1"`, "<p>라벨</p>"),
				shapeWith(attrs, "\n  "),
			} {
				if _, diags := resolveSlide(t, quiet); len(diags) != 0 {
					t.Errorf("진단 %d개, want 0개 (%s): %v", len(diags), quiet, diags)
				}
			}
		})
	}
}

// ADR 0040의 확정 높이 요구는 **내용이 컨테이너일 때도** 걸린다 (ADR 0049): 배경 모드의
// 자식은 백분율 텍스트 영역 안에서 흐르므로 기준의 순환이 그대로다.
func TestResolveShapeInsetTextHeightCountsContainers(t *testing.T) {
	shapeWith := func(attrs, children string) string {
		return "<slide id=\"shape\">\n  <shape " + attrs + ">" + children + "</shape>\n</slide>\n"
	}
	attrs := `type="ellipse"`
	_, diags := resolveSlide(t, shapeWith(attrs, `<col gap="sm"><p>라벨</p></col>`))
	if len(diags) != 1 {
		t.Fatalf("진단 %d개, want 1개: %v", len(diags), diags)
	}
	assertShapeDiag(t, diags[0], 4, "h나 ratio가 필요하다", "ellipse")
	if _, diags := resolveSlide(t, shapeWith(attrs+` h="200"`, `<col gap="sm"><p>라벨</p></col>`)); len(diags) != 0 {
		t.Errorf("진단 %d개, want 0개: %v", len(diags), diags)
	}
}

// 배경 모드 도형의 `rotate`는 에러다 (ADR 0049 결정 6). 웹은 도형 박스를 돌려 **자식까지**
// 돌리는데 PPTX의 형제 도형은 남의 회전을 물려받지 않으므로, 받아 두면 같은 SOML이 두
// 타깃에서 다른 그림이 된다 — 조용한 갈라짐을 막는 유일한 자리가 값 층이다.
func TestResolveShapeRotateWithLayoutIsError(t *testing.T) {
	shapeWith := func(attrs, children string) string {
		return "<slide id=\"shape\">\n  <shape " + attrs + ">" + children + "</shape>\n</slide>\n"
	}
	attrs := `type="rect" rotate="12"`
	deck, diags := resolveSlide(t, shapeWith(attrs, `<col gap="sm"><p>본문</p></col>`))
	if len(diags) != 1 {
		t.Fatalf("진단 %d개, want 1개: %v", len(diags), diags)
	}
	// 진단은 `rotate` **속성 이름**을 가리킨다 — 고칠 것이 값이 아니라 속성과 내용의
	// 조합이고, 저자가 고르는 쪽(회전을 지우거나 배치를 빼는 것)을 메시지가 적는다.
	assertShapeDiag(t, diags[0], shapeCol(attrs, "rotate"),
		"배치를 담은 도형은 rotate를 쓸 수 없다", "자식은 형제 도형이 되어")
	if style := findElement(t, deck, soml.ShapeElement).ShapeStyle(); style == nil || style.Rotate != nil {
		t.Errorf("무효한 rotate가 실렸다: %+v", style)
	}

	// 흡수 모드(텍스트 자식)와 빈 도형의 회전은 그대로다 — 문단은 도형과 함께 돌아간다.
	for _, quiet := range []string{
		shapeWith(attrs, "<p>본문</p>"),
		shapeSlide(attrs),
	} {
		if _, diags := resolveSlide(t, quiet); len(diags) != 0 {
			t.Errorf("진단 %d개, want 0개 (%s): %v", len(diags), quiet, diags)
		}
	}
}

// 나머지 값 문법의 위반 전수 — 각 진단이 **고치는 방법**을 담고 있어야 한다.
func TestResolveShapeValueErrors(t *testing.T) {
	cases := []struct {
		name     string
		attrs    string
		needle   string
		contains []string
	}{
		{"radius 모르는 토큰", `type="rounded-rect" radius="huge"`, "huge",
			[]string{"알 수 없는 radius 토큰", "sm, md, lg", "수치"}},
		{"radius 단위", `type="rounded-rect" radius="8px"`, "8px",
			[]string{"단위 없는 논리 px 수치"}},
		{"radius 음수", `type="rounded-rect" radius="-4"`, "-4",
			[]string{"0보다 작을 수 없다"}},
		{"shadow 모르는 토큰", `type="rect" shadow="hard"`, "hard",
			[]string{"알 수 없는 shadow 토큰", "soft", "수치가 아니라 토큰"}},
		{"shadow 수치 축약", `type="rect" shadow="0 2 8 #000000"`, "0 2 8",
			[]string{"알 수 없는 shadow 토큰"}},
		{"opacity 백분율", `type="rect" opacity="50%"`, "50%",
			[]string{"opacity에 %는 쓸 수 없다", "0~1"}},
		{"opacity 범위 초과", `type="rect" opacity="1.5"`, "1.5",
			[]string{"0~1 사이다"}},
		{"opacity 음수", `type="rect" opacity="-0.2"`, "-0.2",
			[]string{"0~1 사이다"}},
		{"opacity 수치 아님", `type="rect" opacity="반쯤"`, "반쯤",
			[]string{"수치가 아니다"}},
		{"rotate 단위", `type="rect" rotate="45deg"`, "45deg",
			[]string{"단위 없는 각도 수치", `"deg"`}},
		{"rotate 수치 아님", `type="rect" rotate="비스듬히"`, "비스듬히",
			[]string{"수치가 아니다", "음수 가능"}},
		{"flip 어휘 밖", `type="chevron" flip="both"`, "both",
			[]string{"알 수 없는 flip", "h, v, hv"}},
		{"flip 공백 표기", `type="chevron" flip="h v"`, "h v",
			[]string{"알 수 없는 flip"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, diags := resolveSlide(t, shapeSlide(tc.attrs))
			if len(diags) != 1 {
				t.Fatalf("진단 %d개, want 1개: %v", len(diags), diags)
			}
			assertShapeDiag(t, diags[0], shapeCol(tc.attrs, tc.needle), tc.contains...)
		})
	}
}

// 도형의 박스 값 문법이 컨테이너와 **같은 확정 값**에 도달하는지 — 값 문법이 두 벌이 되면
// 이 테스트가 먼저 죽는다 (soml.Box를 공유한 이유의 실행 가능한 형태다).
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

	// 박스 속성 카탈로그(soml.BoxAttrs)를 전부 덮었는지 — 목록이 넓어지면 이 표가 빨갛게 된다.
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

// 외관 속성이 **없는** 도형도 완전한 외관을 갖는다 — 컴포넌트 기본값이 캐스케이드의 가장
// 아래 단이기 때문이다 (값과 근거는 TestResolveShapeComponentDefault). 여기서는 인라인 단이
// 아무것도 더하지 않는다는 것만 본다.
func TestResolveShapeWithoutAppearanceUsesComponentDefault(t *testing.T) {
	style := resolvedShape(t, `type="rect"`)
	want := componentShapeStyle("rect")
	if !reflect.DeepEqual(*style, want) {
		t.Errorf("도형 외관 = %s, want %s", showShape(*style), showShape(want))
	}
}

// 도형 `id`의 중복 검사 범위는 **슬라이드**다 — 참조(`<arrow from>`)가 슬라이드를 넘지
// 않으므로 덱 전역 유일을 요구하면 저자가 슬라이드마다 같은 이름을 쓸 수 없다.
func TestResolveShapeDuplicateID(t *testing.T) {
	src := "<slide id=\"shape\">\n  <shape type=\"rect\" id=\"db\"/>\n  <shape type=\"ellipse\" id=\"db\"/>\n</slide>\n"
	deck, diags := resolveSlide(t, src)
	if len(diags) != 1 {
		t.Fatalf("진단 %d개, want 1개: %v", len(diags), diags)
	}
	// 진단은 두 번째 자리에서 나고 **처음 자리**를 문구에 싣는다 — 저자가 어느 둘이
	// 겹쳤는지 알아야 한다.
	if diags[0].Span.Start.Line != 3 {
		t.Errorf("위치 = %s, want 3행 (두 번째 도형)", diags[0].Span.Start)
	}
	for _, want := range []string{`id "db"가 이 슬라이드에 이미 있다`, "2:", "참조가 어느 도형인지"} {
		if !strings.Contains(diags[0].Message, want) {
			t.Errorf("진단 = %q, want %q 포함", diags[0].String(), want)
		}
	}
	// 중복은 확정 값에 실리지 않는다 (첫 도형의 id는 남는다).
	shapes := shapeStyles(t, deck)
	if len(shapes) != 2 || shapes[0].ID != "db" || shapes[1].ID != "" {
		t.Errorf("id = %q, %q, want \"db\", \"\"", shapes[0].ID, shapes[1].ID)
	}

	// 슬라이드가 다르면 같은 이름을 쓸 수 있다 — 범위가 슬라이드라는 뜻이다.
	deck, diags = resolveSlide(t, shapeSlide(`type="rect" id="db"`))
	if len(diags) != 0 {
		t.Fatalf("진단이 없어야 한다: %v", diags)
	}
	second, secondDiags := resolveSlide(t, shapeSlide(`type="rect" id="db"`))
	if len(secondDiags) != 0 {
		t.Fatalf("다른 슬라이드의 같은 id는 진단이 없어야 한다: %v", secondDiags)
	}
	if shapeStyles(t, deck)[0].ID != shapeStyles(t, second)[0].ID {
		t.Error("두 슬라이드의 같은 id가 다르게 해석됐다")
	}
}

// `anchor`는 텍스트 외관이다 (도형이 자식의 앵커를 읽는다 — 항목 04·05). 어휘는 **물리
// 이름**이다: 세로축은 문단 방향으로 뒤집히지 않으므로 논리 이름을 쓸 근거가 없다.
func TestResolveTextAnchor(t *testing.T) {
	for _, anchor := range soml.TextAnchors() {
		t.Run(string(anchor), func(t *testing.T) {
			src := "<slide id=\"shape\">\n  <shape type=\"rect\"><text anchor=\"" +
				string(anchor) + "\">내용</text></shape>\n</slide>\n"
			style := resolvedElement(t, src, soml.TextElement).TextStyle()
			if style == nil {
				t.Fatal("<text>에 확정 외관이 없다")
			}
			if style.Anchor != anchor {
				t.Errorf("anchor = %q, want %q", style.Anchor, anchor)
			}
		})
	}

	// 블록 태그도 받는다 (파서가 두 자리에 열었다 — 항목 01).
	blockSrc := "<slide id=\"shape\">\n  <shape type=\"rect\"><p anchor=\"bottom\">내용</p></shape>\n</slide>\n"
	if style := resolvedElement(t, blockSrc, "p").TextStyle(); style == nil || style.Anchor != soml.AnchorBottom {
		t.Errorf("<p>의 anchor가 해석되지 않았다: %+v", style)
	}

	// 속성이 없으면 **빈 문자열로 남는다** — 역할 기본값이 채우지 않는 유일한 필드이고,
	// 그 빈 값이 기존 골든을 지킨다 (뜻 없는 자리에 앵커 선언을 내지 않는다).
	if style := firstStyle(t, "<slide id=\"t\">\n  <text>내용</text>\n</slide>\n"); style.Anchor != "" {
		t.Errorf("anchor = %q, want 빈 문자열", style.Anchor)
	}

	cases := []struct {
		name     string
		value    string
		contains []string
	}{
		{"논리 이름 start", "start", []string{"anchor는 세로 이름을 쓴다", `"top"`}},
		{"논리 이름 end", "end", []string{"anchor는 세로 이름을 쓴다", `"bottom"`}},
		{"어휘 밖", "middle", []string{"알 수 없는 anchor", "top, center, bottom"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			src := "<slide id=\"t\">\n  <text anchor=\"" + tc.value + "\">내용</text>\n</slide>\n"
			deck, diags := resolveSlide(t, src)
			if len(diags) != 1 {
				t.Fatalf("진단 %d개, want 1개: %v", len(diags), diags)
			}
			for _, want := range tc.contains {
				if !strings.Contains(diags[0].Message, want) {
					t.Errorf("진단 = %q, want %q 포함", diags[0].String(), want)
				}
			}
			// 틀린 값은 실리지 않는다 — 아래 단(빈 값)이 남는다.
			if style := findElement(t, deck, soml.TextElement).TextStyle(); style == nil || style.Anchor != "" {
				t.Errorf("무효한 anchor가 실렸다: %+v", style)
			}
		})
	}
}

// 도형 안 `<text>`는 **기존 텍스트 경로를 그대로** 지난다 — 부모가 도형이라는 사실이 텍스트
// 해석을 바꾸지 않는다 (바꾸는 것은 상속이고, 상속은 이 언어에 없다).
func TestResolveShapeChildTextUnchanged(t *testing.T) {
	inShape := resolvedElement(t,
		"<slide id=\"s\">\n  <shape type=\"rect\"><p>내용</p></shape>\n</slide>\n", "p").TextStyle()
	direct := tagStyle(t, "<slide id=\"s\">\n  <p>내용</p>\n</slide>\n", "p")
	if inShape == nil {
		t.Fatal("도형 안 <p>에 확정 외관이 없다")
	}
	if !reflect.DeepEqual(*inShape, *direct) {
		t.Errorf("도형 안 <p> 외관 = %+v, 슬라이드 직속 = %+v", *inShape, *direct)
	}
}

// 한 요소가 도형 외관 + 자기 박스 + 자식 배치를 함께 쓴다 — 대입이 아니라 필드 채우기라는
// 구조(elemAppearance)가 도형에서도 유지되는지. 세 채널의 소유자가 다르므로(외관은
// 캐스케이드, 박스는 도형 자신, 배치는 부모 안의 자리) 하나가 다른 하나를 지우면 산출물에서
// 그 채널만 조용히 사라진다.
func TestResolveShapeKeepsPlacement(t *testing.T) {
	src := "<slide id=\"s\">\n  <row><shape type=\"rect\" fill=\"surface\" pad=\"md\" grow=\"1\"/></row>\n</slide>\n"
	elem := resolvedElement(t, src, soml.ShapeElement)
	style, placement := elem.ShapeStyle(), elem.Placement()
	if style == nil || style.Fill == nil || style.Fill.Color != soml.TokenColor("surface") {
		t.Errorf("외관이 없다: %+v", style)
	}
	// 캐스케이드의 아래 단도 함께 확정된다 (인라인이 fill만 말했다).
	if want := componentShapeStyle("rect").Border; !reflect.DeepEqual(style.Border, want) {
		t.Errorf("border = %+v, want 컴포넌트 기본값 %+v", style.Border, want)
	}
	if style.Box.Pad == nil || style.Box.Pad.Top != soml.SpaceLength("md", 24) {
		t.Errorf("박스가 없다: %+v", style.Box)
	}
	if placement == nil || placement.Grow == nil || *placement.Grow != 1 {
		t.Errorf("배치가 없다: %+v", placement)
	}
}

// 픽스처는 이번 iteration 전체의 시험대다 (`04_shape` — 항목 06이 수집 대상으로 켰다).
// 값 층이 픽스처를 해석하는지, 경고가 나는 자리가 예상한 곳뿐인지 여기서 못 박는다.
func TestResolveShapeFixture(t *testing.T) {
	deck := readFixtureDeck(t, "04_shape")
	diags := Resolve(deck, nil)
	for _, diag := range diags {
		if diag.Severity != soml.SeverityWarning {
			t.Errorf("픽스처에 에러가 있다: %s", diag.String())
		}
	}
	// 경고는 둘이다 — 008의 `<text variant="leed">`와 같은 슬라이드의
	// `<shape variant="card-heder">`. 캐스케이드가 붙으면서 도형 쪽 경고가 생겼고, 두
	// 경고가 **각자의 어휘 목록**을 나열한다 (TestResolveVariantVocabulariesAreSeparate).
	if len(diags) != 2 {
		t.Fatalf("경고 %d개, want 2개: %v", len(diags), diags)
	}
	for _, diag := range diags {
		if !strings.HasSuffix(diag.File, "008_variant_unknown"+soml.Ext) {
			t.Errorf("경고가 008 밖에서 났다: %s", diag.String())
		}
	}
	if !strings.Contains(diags[0].Message, "card-heder") {
		t.Errorf("첫 경고 = %s, want 008의 도형 variant", diags[0].String())
	}
	if !strings.Contains(diags[1].Message, "leed") {
		t.Errorf("둘째 경고 = %s, want 008의 텍스트 variant", diags[1].String())
	}

	// 007은 조용하다 — 픽스처가 쓴 도형 variant 이름이 테마 표에 있어야 한다 (이름이 어긋나면
	// 그림이 기본 외관으로 폴백해 이 iteration의 시험대가 무의미해진다).
	for _, slide := range deck.Slides {
		if !strings.HasSuffix(slide.Source, "007_variant"+soml.Ext) {
			continue
		}
		for _, diag := range diags {
			if diag.File == slide.Source {
				t.Errorf("007에 경고가 있다: %s", diag.String())
			}
		}
	}

	// 001_shape_types — 프리셋 12종 전수가 확정 값에 도달했는지 (항목 04·05의 시험대).
	types := map[string]bool{}
	for _, style := range shapeStyles(t, deck) {
		if style.Type == "" {
			t.Error("type이 해석되지 않은 도형이 있다")
		}
		types[style.Type] = true
	}
	for _, name := range soml.ShapeTypes() {
		if !types[name] {
			t.Errorf("픽스처에 type=%q 도형이 없다", name)
		}
	}

	// 002_token_colors — 제품 핵심 계약: 토큰 색은 **이름으로** 남는다 (schemeClr 참조가
	// 되어야 한다). RGB로 눕는 회귀를 여기서 잡는다.
	shape := findElement(t, deck, soml.ShapeElement)
	for _, slide := range deck.Slides {
		if !strings.HasSuffix(slide.Source, "002_token_colors"+soml.Ext) {
			continue
		}
		found := false
		for _, node := range slide.Content {
			walkShapes(node, func(style *soml.ShapeStyle) {
				if found || style.Fill == nil || style.Fill.None {
					return
				}
				if !style.Fill.Color.IsToken() {
					t.Errorf("토큰 색이 리터럴로 눕었다: %+v", style.Fill.Color)
				}
				found = true
			})
		}
		if !found {
			t.Error("002에 토큰 채움 도형이 없다")
		}
	}
	if shape.ShapeStyle() == nil {
		t.Fatal("첫 도형의 외관이 없다")
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

// assertShapeDiag는 진단이 에러이고, 2행 col을 가리키고, 문구를 담는지 본다.
func assertShapeDiag(t *testing.T, diag soml.Diagnostic, col int, contains ...string) {
	t.Helper()
	if diag.Severity != soml.SeverityError {
		t.Errorf("심각도 = %v, want error (%s)", diag.Severity, diag.String())
	}
	if diag.Span.Start.Line != 2 || diag.Span.Start.Col != col {
		t.Errorf("위치 = %s, want 2:%d (%s)", diag.Span.Start, col, diag.String())
	}
	for _, want := range contains {
		if !strings.Contains(diag.Message, want) {
			t.Errorf("진단 = %q, want %q 포함", diag.String(), want)
		}
	}
}

// showShape는 실패 메시지용 문자열이다 — 포인터 필드가 많아 %+v로는 주소만 보인다
// (showContainer와 같은 이유).
func showShape(s soml.ShapeStyle) string {
	var b strings.Builder
	fmt.Fprintf(&b, "{type:%s", s.Type)
	if s.ID != "" {
		b.WriteString(" id:" + s.ID)
	}
	if s.Fill != nil {
		b.WriteString(" fill:" + showFill(*s.Fill))
	}
	if s.Border != nil {
		border := *s.Border
		if border.None {
			b.WriteString(" border:none")
		} else {
			fmt.Fprintf(&b, " border:%g/%s/%s", border.Width, border.Style, showColor(border.Color))
		}
	}
	if s.Radius != nil {
		fmt.Fprintf(&b, " radius:%s", showRadius(*s.Radius))
	}
	if s.Shadow != nil {
		fmt.Fprintf(&b, " shadow:%s(%g/%g/%g/%s/%g)", s.Shadow.Name, s.Shadow.Dist, s.Shadow.Dir,
			s.Shadow.Blur, showColor(s.Shadow.Color), s.Shadow.Alpha)
	}
	if s.Opacity != nil {
		fmt.Fprintf(&b, " opacity:%g", *s.Opacity)
	}
	if s.Rotate != nil {
		fmt.Fprintf(&b, " rotate:%g", *s.Rotate)
	}
	if s.Flip != "" {
		b.WriteString(" flip:" + string(s.Flip))
	}
	b.WriteString(showBox(s.Box))
	b.WriteString("}")
	return b.String()
}

// showBox는 박스 필드의 문자열이다 (showContainer와 같은 형태 — 두 소비자가 같은 표기를
// 보이게).
func showBox(box soml.Box) string {
	var b strings.Builder
	for _, field := range []struct {
		name   string
		length *soml.Length
	}{
		{"w", box.Width}, {"h", box.Height}, {"min-w", box.MinWidth}, {"min-h", box.MinHeight},
		{"max-w", box.MaxWidth}, {"max-h", box.MaxHeight},
	} {
		if field.length != nil {
			b.WriteString(" " + field.name + ":" + showLength(*field.length))
		}
	}
	if box.Pad != nil {
		b.WriteString(" pad:" + showLength(box.Pad.Top) + "/" + showLength(box.Pad.Right) +
			"/" + showLength(box.Pad.Bottom) + "/" + showLength(box.Pad.Left))
	}
	if box.Ratio != nil {
		fmt.Fprintf(&b, " ratio:%d:%d", box.Ratio.W, box.Ratio.H)
	}
	return b.String()
}

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

func showRadius(radius soml.Radius) string {
	if radius.IsToken() {
		return fmt.Sprintf("token(%s=%g)", radius.Token, radius.Px)
	}
	return fmt.Sprintf("%g", radius.Px)
}
