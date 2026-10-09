package core

import (
	"strings"
	"testing"

	"github.com/k2sebeom/balpyo/soml"
)

// styleOf는 `style` 하나만 붙인 자식의 확정 선언 문자열이다 (진단 0건을 전제한다).
func styleOf(t *testing.T, decl string) string {
	t.Helper()
	placement := resolvedElement(t, childSrc("row", `style="`+decl+`"`), "p").Placement()
	if placement == nil {
		t.Fatalf("style이 해석되지 않았다 (%q)", decl)
	}
	return placement.Style
}

// 배치 선언은 경고 없이 통과한다 — 탈출구의 정상 경로다.
func TestResolveStyleAllowsPlacementDeclarations(t *testing.T) {
	cases := []struct{ name, decl, want string }{
		{"단일 선언", "align-content: space-evenly", "align-content: space-evenly"},
		{"선언 여러 개", "order: 2; margin-inline-start: 40px", "order: 2; margin-inline-start: 40px"},
		// 정규화: 공백과 마지막 `;`의 유무가 산출물을 흔들지 않아야 골든이 무엇이 변했는지
		// 말해 준다.
		{"공백 정규화", "  gap:8px ;   place-self:end  ", "gap: 8px; place-self: end"},
		{"마지막 세미콜론", "display: grid;", "display: grid"},
		{"빈 선언 무시", "width: 50%;; height: 100%", "width: 50%; height: 100%"},
		// 프로퍼티 이름은 CSS와 같이 대소문자를 구별하지 않는다 — 값은 그대로 둔다
		// (색·문자열의 대소문자는 내용이다).
		{"이름 소문자화", "GRID-TEMPLATE-COLUMNS: repeat(2, 1FR)", "grid-template-columns: repeat(2, 1FR)"},
		// 값은 검증하지 않는다 — 엔진이 CSS 값 문법을 재구현하지 않는다는 것이 계약이다.
		{"검증하지 않는 값", "width: clamp(10px, 5vw, 100px)", "width: clamp(10px, 5vw, 100px)"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := styleOf(t, tc.decl); got != tc.want {
				t.Errorf("style = %q, want %q", got, tc.want)
			}
		})
	}
}

// allowlist는 컨테이너에도 열려 있다 (`style`은 자식 배치 속성이므로 컨테이너도 자식이다).
func TestResolveStyleOnContainer(t *testing.T) {
	elem := resolvedElement(t, containerSrc("row", `gap="md" style="align-content: space-evenly"`), "row")
	if elem.Container() == nil || elem.Container().Gap == nil {
		t.Fatal("컨테이너 배치가 해석되지 않았다")
	}
	placement := elem.Placement()
	if placement == nil || placement.Style != "align-content: space-evenly" {
		t.Fatalf("컨테이너의 style = %+v", placement)
	}
}

// allowlist 밖의 선언은 **경고**다 (에러가 아니다 — 에러로 두면 엔진이 CSS 프로퍼티 전체를
// 알아야 하고, 그것은 탈출구의 존재 이유와 모순된다).
func TestResolveStyleWarnsOnAppearanceDeclarations(t *testing.T) {
	cases := []struct {
		name string
		decl string
		col  int
		prop string
	}{
		{"배경색", "background: #0F65F9", 18, "background"},
		{"테두리", "border: 2px solid red", 18, "border"},
		// 경고는 값 전체가 아니라 문제가 된 **선언**을 가리킨다.
		{"둘째 선언", "gap: 8px; color: red", 28, "color"},
		{"엔진이 모르는 프로퍼티", "contain-intrinsic-size: 10px", 18, "contain-intrinsic-size"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			deck, diags := resolveSlide(t, layoutSlide(`<row><p style="`+tc.decl+`">내용</p></row>`))
			if len(diags) != 1 {
				t.Fatalf("진단 %d개, want 1개: %v", len(diags), diags)
			}
			diag := diags[0]
			if diag.Severity != soml.SeverityWarning {
				t.Errorf("심각도 = %s, want warning — 외관 선언은 빌드를 멈추지 않는다", diag.Severity)
			}
			if !strings.Contains(diag.Message, tc.prop) ||
				!strings.Contains(diag.Message, "웹에만 적용되고 PPTX에는 이식되지 않는다") {
				t.Errorf("진단 = %q", diag.String())
			}
			if diag.Span.Start.Line != 2 || diag.Span.Start.Col != tc.col {
				t.Errorf("위치 = %s, want 2:%d", diag.Span.Start, tc.col)
			}
			// 경고는 선언을 버리지 않는다 — 저자가 통과시킨 CSS는 웹에 그대로 나간다.
			placement := findElement(t, deck, "p").Placement()
			if placement == nil || !strings.Contains(placement.Style, tc.prop) {
				t.Errorf("경고 난 선언이 확정 배치에서 빠졌다: %+v", placement)
			}
			// --strict가 에러로 승격한다 (모르는 variant와 같은 경로 — soml.Diagnostics.Strict).
			if !diags.Strict().HasErrors() {
				t.Error("--strict가 경고를 에러로 승격하지 않았다")
			}
		})
	}
}

// 문법 위반은 에러다 — 값은 보지 않지만 `속성: 값` 형태는 본다.
func TestResolveStyleSyntaxErrors(t *testing.T) {
	cases := []struct {
		name     string
		decl     string
		col      int
		contains string
	}{
		{"콜론 없음", "display flex", 18, "콜론이 없다"},
		{"속성 이름 없음", ": 40px", 18, "속성 이름이 없다"},
		{"값 없음", "width:", 18, "값이 없다"},
		{"둘째 선언의 위반", "gap: 8px; width", 28, "콜론이 없다"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			deck, diags := resolveSlide(t, layoutSlide(`<row><p style="`+tc.decl+`">내용</p></row>`))
			if len(diags) != 1 {
				t.Fatalf("진단 %d개, want 1개: %v", len(diags), diags)
			}
			diag := diags[0]
			if diag.Severity != soml.SeverityError || !strings.Contains(diag.Message, tc.contains) {
				t.Errorf("진단 = %q, want 에러 %q 포함", diag.String(), tc.contains)
			}
			if diag.Span.Start.Line != 2 || diag.Span.Start.Col != tc.col {
				t.Errorf("위치 = %s, want 2:%d", diag.Span.Start, tc.col)
			}
			// 문법 위반이 있으면 속성 전체를 싣지 않는다 — 반쯤 통과한 선언 목록은 저자가
			// 고친 뒤의 결과를 예측할 수 없게 만든다.
			if placement := findElement(t, deck, "p").Placement(); placement != nil && placement.Style != "" {
				t.Errorf("문법 위반이 있는 style이 실렸다: %q", placement.Style)
			}
		})
	}
}

// 빈 값과 값 없는 속성은 외관 속성과 같은 진단 경로다 (hasValue).
func TestResolveStyleEmpty(t *testing.T) {
	cases := []struct {
		name     string
		content  string
		contains string
	}{
		{"빈 값", `<row><p style="">내용</p></row>`, "비어 있다"},
		{"값 없음", `<row><p style>내용</p></row>`, "값이 필요하다"},
		{"공백뿐", `<row><p style="  ">내용</p></row>`, "선언이 없다"},
		{"세미콜론뿐", `<row><p style=";">내용</p></row>`, "선언이 없다"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, diags := resolveSlide(t, layoutSlide(tc.content))
			if len(diags) != 1 {
				t.Fatalf("진단 %d개, want 1개: %v", len(diags), diags)
			}
			if diags[0].Severity != soml.SeverityError || !strings.Contains(diags[0].Message, tc.contains) {
				t.Errorf("진단 = %q, want 에러 %q 포함", diags[0].String(), tc.contains)
			}
		})
	}
}

// allowlist는 **순서 있는 한 곳**이고 문서가 같은 목록을 인용한다. 여기서 못 박는
// 것은 목록의 모양(중복·빈 항목 없음)과 가족 접두의 판정이다.
func TestLayoutStylePropertyAllowlist(t *testing.T) {
	seen := map[string]bool{}
	for _, pattern := range layoutStyleProperties() {
		if pattern == "" || pattern == "*" {
			t.Errorf("빈 패턴이 있다: %q", pattern)
		}
		if seen[pattern] {
			t.Errorf("패턴이 중복됐다: %q", pattern)
		}
		seen[pattern] = true
	}
	// 배치 속성이 CSS로 컴파일되는 이름은 전부 목록에 있어야 한다 — 저자가
	// 레이아웃 속성으로 적을 수 있는 것을 style로 적었다고 경고하면 목록이 거짓말을 한다.
	for _, property := range []string{
		"display", "flex-direction", "flex-grow", "flex-basis", "flex-wrap",
		"grid-template-columns", "grid-column", "gap", "row-gap",
		"padding", "padding-inline", "margin-inline-start",
		"width", "min-width", "max-height", "aspect-ratio",
		"align-items", "align-self", "justify-content", "place-content",
		"order", "position", "inset", "left", "translate",
	} {
		if !isLayoutStyleProperty(property) {
			t.Errorf("%q가 배치 allowlist에 없다", property)
		}
	}
	// 반대쪽: 외관은 경고 대상이다. rotate/scale이 여기 있는 이유는 style.go의 주석에 있다 —
	// 회전·확대는 잰 박스가 외접 사각형으로 남아 PPTX에서 다른 도형이 된다.
	for _, property := range []string{
		"background", "background-color", "border", "border-radius", "box-shadow",
		"color", "font-size", "opacity", "rotate", "scale", "transform", "--custom",
	} {
		if isLayoutStyleProperty(property) {
			t.Errorf("%q가 배치 allowlist에 있다 — 외관은 경고 대상이다", property)
		}
	}
}
