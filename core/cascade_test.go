package core

import (
	"reflect"
	"strings"
	"testing"

	"github.com/k2sebeom/balpyo/soml"
)

// 우선순위 표: 같은 필드를 여러 단이 정할 때 인라인 > variant > 컴포넌트 기본값 >
// 역할 기본값 순으로 이긴다. 단마다 하나씩 빼 가며 단정한다 — 병합이 mergeText 한 곳에서만
// 일어나므로 여기서 순서를 못 박으면 캐스케이드 전체의 우선순위가 못 박힌다.
func TestMergeTextCascadeOrder(t *testing.T) {
	component := textPatch{size: ptr(30.0)}
	variant := textPatch{size: ptr(40.0)}
	inline := textPatch{size: ptr(50.0)}
	empty := textPatch{}

	cases := []struct {
		name                       string
		component, variant, inline textPatch
		want                       float64
	}{
		{"인라인이 전부를 이긴다", component, variant, inline, 50},
		{"인라인이 빠지면 variant", component, variant, empty, 40},
		{"variant까지 빠지면 컴포넌트 기본값", component, empty, empty, 30},
		{"전부 빠지면 역할 기본값", empty, empty, empty, 24},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			style := textRoleDefaults(soml.RoleBody) // size 24
			mergeText(&style, tc.component)
			mergeText(&style, tc.variant)
			mergeText(&style, tc.inline)
			if style.Size != tc.want {
				t.Errorf("Size = %v, want %v", style.Size, tc.want)
			}
		})
	}
}

// 부분성: 패치는 정한 필드만 얹고, "정하지 않음"(nil)은 아래 단의 값을 지우지 않는다.
func TestMergeTextAppliesOnlySetFields(t *testing.T) {
	base := textRoleDefaults(soml.RoleBody)

	// 빈 패치는 아무것도 바꾸지 않는다.
	got := base
	mergeText(&got, textPatch{})
	if got != base {
		t.Errorf("빈 패치가 외관을 바꿨다: %+v → %+v", base, got)
	}

	// 필드 전부를 정한 패치는 전부를 바꾼다 — 병합이 다루는 필드 목록의 회귀 감시다.
	// TextStyle에 필드가 늘었는데 textPatch가 따라가지 않으면 이 단정이 잡는다.
	full := textPatch{
		font:       ptr(soml.FontMono),
		size:       ptr(99.0),
		weight:     ptr(700),
		lineHeight: ptr(2.0),
		color:      ptr(soml.LiteralColor("ABCDEF")),
		align:      ptr(soml.AlignJustify),
	}
	got = base
	mergeText(&got, full)
	want := soml.TextStyle{Role: soml.RoleBody, Font: soml.FontMono, Size: 99, Weight: 700,
		LineHeight: 2.0, Color: soml.LiteralColor("ABCDEF"), Align: soml.AlignJustify}
	if got != want {
		t.Errorf("전체 패치 = %+v, want %+v", got, want)
	}

	// 일부만 정한 패치는 나머지를 아래 단에 남긴다.
	got = base
	mergeText(&got, textPatch{size: ptr(32.0)})
	if got.Size != 32 {
		t.Errorf("Size = %v, want 32", got.Size)
	}
	if got.Color != base.Color || got.Align != base.Align || got.Font != base.Font ||
		got.Weight != base.Weight || got.LineHeight != base.LineHeight {
		t.Errorf("정하지 않은 필드가 아래 단을 잃었다: %+v", got)
	}
}

// "정하지 않음"과 0값의 구별: 0값을 **명시적으로** 정한 패치는 0값을 얹는다.
// 이 구별이 없으면 (완성 스타일끼리 덮어쓰면) "weight를 0으로" 같은 지정과 "weight를
// 말하지 않음"이 같은 코드가 되어 어느 단이 이겼는지 알 수 없다.
func TestMergeTextDistinguishesUnsetFromZero(t *testing.T) {
	style := textRoleDefaults(soml.RoleBody)
	mergeText(&style, textPatch{weight: ptr(0), size: ptr(0.0)})
	if style.Weight != 0 || style.Size != 0 {
		t.Errorf("명시된 0값이 적용되지 않았다: %+v", style)
	}
}

// minimal 테마의 variant 표: 이름은 결정적 순서로 나열되고 (경고 메시지의 근거),
// 각 variant는 역할 기본값과 **구별되는** 값만 정한다 — 같은 값을 정하는 variant는
// 우선순위 테스트가 아무것도 증명하지 못한다.
func TestThemeTextVariantsAreDistinctFromRoleDefaults(t *testing.T) {
	variants := textVariants()
	if len(variants) < 2 {
		t.Fatalf("variant %d개 — 캐스케이드를 증명하려면 2개 이상이 필요하다", len(variants))
	}
	seen := map[string]bool{}
	for _, v := range variants {
		if seen[v.name] {
			t.Errorf("variant 이름이 중복됐다: %q", v.name)
		}
		seen[v.name] = true
		if v.patch == (textPatch{}) {
			t.Errorf("variant %q가 아무 필드도 정하지 않는다", v.name)
		}
		// 어느 역할 기본값과도 겹치지 않아야 하는 것은 아니다 — 기본 역할(body) 기준으로
		// variant가 정한 필드가 실제로 값을 바꾸는지만 본다.
		base := textRoleDefaults(soml.DefaultTextRole)
		merged := base
		mergeText(&merged, v.patch)
		if merged == base {
			t.Errorf("variant %q가 기본 역할의 외관을 하나도 바꾸지 않는다", v.name)
		}
	}
}

func TestTextVariantLookup(t *testing.T) {
	for _, v := range textVariants() {
		patch, ok := textVariantPatch(v.name)
		if !ok {
			t.Errorf("textVariantPatch(%q)를 찾지 못했다", v.name)
			continue
		}
		// 패치는 포인터 필드라 값 비교가 안 된다 — 같은 바닥에 얹은 결과로 비교한다.
		wantStyle, gotStyle := textRoleDefaults(soml.DefaultTextRole), textRoleDefaults(soml.DefaultTextRole)
		mergeText(&wantStyle, v.patch)
		mergeText(&gotStyle, patch)
		if gotStyle != wantStyle {
			t.Errorf("textVariantPatch(%q) = %+v, want %+v", v.name, gotStyle, wantStyle)
		}
	}
	if _, ok := textVariantPatch("없는-이름"); ok {
		t.Error("없는 variant가 찾아졌다")
	}
}

// ── 도형 캐스케이드 ─────────────────────────────────────────────────────────
//
// 4단 캐스케이드가 텍스트 밖으로 처음 나가는 자리다. 아래 테스트가 못 박는 것은 값이
// 아니라 **구조**다: 도형이 텍스트와 같은 병합 기계(mergeShape ↔ mergeText)를 쓰고,
// 우선순위가 resolveShape의 병합 세 줄에만 있다는 것.

// componentShapeStyle는 컴포넌트 기본값만 얹은 확정 외관이다 — 캐스케이드의 **바닥**이고,
// 모르는 variant의 폴백 기대값이자 표 기반 테스트(shape_test.go)의 기대값 바닥이다.
func componentShapeStyle(shapeType string) soml.ShapeStyle {
	style := soml.ShapeStyle{Type: shapeType}
	mergeShape(&style, themeShapePatch(shapeComponentDefault(minimalTheme()), shapeType))
	return style
}

// 우선순위 표: 인라인 > variant > 컴포넌트 기본값. 텍스트와 갈라지는 자리는 **바닥**이다 —
// 도형에는 역할이 없으므로 컴포넌트 기본값이 가장 아래 단이고, 어느 단도 정하지 않은 필드는
// 비어 남는다 (nil = 선언 없음).
func TestMergeShapeCascadeOrder(t *testing.T) {
	component := shapePatch{fill: colorFill("surface")}
	variant := shapePatch{fill: colorFill("bg")}
	inline := shapePatch{fill: colorFill("accent1")}
	empty := shapePatch{}

	cases := []struct {
		name                       string
		component, variant, inline shapePatch
		want                       *soml.Fill
	}{
		{"인라인이 전부를 이긴다", component, variant, inline, colorFill("accent1")},
		{"인라인이 빠지면 variant", component, variant, empty, colorFill("bg")},
		{"variant까지 빠지면 컴포넌트 기본값", component, empty, empty, colorFill("surface")},
		{"전부 빠지면 비어 남는다", empty, empty, empty, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			style := soml.ShapeStyle{Type: "rect"}
			mergeShape(&style, tc.component)
			mergeShape(&style, tc.variant)
			mergeShape(&style, tc.inline)
			if !reflect.DeepEqual(style.Fill, tc.want) {
				t.Errorf("Fill = %v, want %v", style.Fill, tc.want)
			}
		})
	}

	// 필드마다 **어느 단에서 왔는지**를 구별되게 짠 표 — 한 필드로는 병합이 "덮어쓰기"라는
	// 것만 보이고, 세 단이 서로 다른 필드를 정할 때 각자의 값이 남는지는 보이지 않는다.
	style := soml.ShapeStyle{Type: "rounded-rect"}
	mergeShape(&style, shapePatch{fill: colorFill("surface"), border: solidBorder(1, "muted"),
		radius: ptr(soml.TokenRadius("md", 12)), opacity: ptr(1.0)})
	mergeShape(&style, shapePatch{fill: colorFill("bg"), radius: ptr(soml.TokenRadius("lg", 24)),
		shadow: ptr(minimalShadows()[0])})
	mergeShape(&style, shapePatch{fill: colorFill("accent1"), rotate: ptr(45.0)})
	want := soml.ShapeStyle{
		Type:    "rounded-rect",
		Fill:    colorFill("accent1"),            // 인라인
		Border:  solidBorder(1, "muted"),         // 컴포넌트 기본값 (위 두 단이 말하지 않았다)
		Radius:  ptr(soml.TokenRadius("lg", 24)), // variant (기본값 md를 이긴다)
		Shadow:  ptr(minimalShadows()[0]),        // variant
		Opacity: ptr(1.0),                        // 컴포넌트 기본값
		Rotate:  ptr(45.0),                       // 인라인
	}
	if !reflect.DeepEqual(style, want) {
		t.Errorf("단별 병합 = %s, want %s", showShape(style), showShape(want))
	}
}

// 부분성: 패치는 정한 필드만 얹고, "정하지 않음"(nil)은 아래 단의 값을 지우지 않는다.
func TestMergeShapeAppliesOnlySetFields(t *testing.T) {
	base := componentShapeStyle("rounded-rect")

	// 빈 패치는 아무것도 바꾸지 않는다.
	got := base
	mergeShape(&got, shapePatch{})
	if !reflect.DeepEqual(got, base) {
		t.Errorf("빈 패치가 외관을 바꿨다: %s → %s", showShape(base), showShape(got))
	}

	// 필드 전부를 정한 패치는 전부를 바꾼다.
	full := shapePatch{
		fill:    ptr(soml.NoFill()),
		border:  ptr(soml.NoBorder()),
		radius:  ptr(soml.PxRadius(9)),
		shadow:  ptr(minimalShadows()[0]),
		opacity: ptr(0.5),
		rotate:  ptr(45.0),
		flip:    ptr(soml.FlipBoth),
	}
	got = base
	mergeShape(&got, full)
	want := soml.ShapeStyle{Type: "rounded-rect", Fill: full.fill, Border: full.border,
		Radius: full.radius, Shadow: full.shadow, Opacity: full.opacity, Rotate: full.rotate,
		Flip: soml.FlipBoth}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("전체 패치 = %s, want %s", showShape(got), showShape(want))
	}

	// ShapeStyle에 외관 필드가 늘었는데 shapePatch가 따라가지 않으면 이 단정이 잡는다
	// (textPatch의 전체 패치 단정과 같은 장치 — 필드가 많아 반사로 훑는다).
	value := reflect.ValueOf(got)
	for i := 0; i < value.NumField(); i++ {
		switch name := value.Type().Field(i).Name; name {
		case "Type", "ID", "Box":
			// 캐스케이드 밖이다 — 테마가 정할 수 없는 값이라 패치에 필드가 없다.
		default:
			if value.Field(i).IsZero() {
				t.Errorf("전체 패치가 %s를 정하지 않았다 — shapePatch에 필드가 빠졌다", name)
			}
		}
	}

	// 일부만 정한 패치는 나머지를 아래 단에 남긴다.
	got = base
	mergeShape(&got, shapePatch{opacity: ptr(0.4)})
	if got.Opacity == nil || *got.Opacity != 0.4 {
		t.Errorf("Opacity = %v, want 0.4", got.Opacity)
	}
	if !reflect.DeepEqual(got.Fill, base.Fill) || !reflect.DeepEqual(got.Border, base.Border) ||
		!reflect.DeepEqual(got.Radius, base.Radius) {
		t.Errorf("정하지 않은 필드가 아래 단을 잃었다: %s", showShape(got))
	}

	// "정하지 않음"과 0값의 구별 — 0을 **명시한** 패치는 0을 얹는다.
	got = base
	mergeShape(&got, shapePatch{opacity: ptr(0.0), rotate: ptr(0.0)})
	if got.Opacity == nil || *got.Opacity != 0 || got.Rotate == nil || *got.Rotate != 0 {
		t.Errorf("명시된 0값이 적용되지 않았다: %s", showShape(got))
	}

	// 식별자와 박스는 병합이 건드리지 않는다 (패치에 그 필드가 없다는 사실의 회귀 감시).
	got = base
	got.ID, got.Box = "db", soml.Box{Width: ptr(soml.Px(220))}
	mergeShape(&got, full)
	if got.ID != "db" || got.Box.Width == nil || *got.Box.Width != soml.Px(220) {
		t.Errorf("병합이 캐스케이드 밖 필드를 건드렸다: %s", showShape(got))
	}
}

// `<shape type="rect"/>` 하나가 **완전한 외관**을 갖는다 — 컴포넌트 기본값이 가장 아래
// 단이라는 계약이다 (PRD §8.3의 `shape.default`). 값이 비어 남으면 렌더러가 채우게 되고,
// 두 타깃의 암묵 기본값이 다른 순간 웹과 PPTX가 갈라진다.
func TestResolveShapeComponentDefault(t *testing.T) {
	style := resolvedShape(t, `type="rect"`)
	if style.Fill == nil || style.Fill.None || style.Fill.Color != soml.TokenColor("surface") {
		t.Errorf("Fill = %v, want 토큰 surface", style.Fill)
	}
	wantBorder := soml.Border{Width: 1, Style: soml.BorderSolid, Color: soml.TokenColor("muted")}
	if style.Border == nil || *style.Border != wantBorder {
		t.Errorf("Border = %v, want %+v", style.Border, wantBorder)
	}
	// 나머지 세 필드는 비어 남는 것이 정상이다: 선언 없음의 뜻(불투명·회전 없음·대칭 없음)이
	// 두 타깃에서 같으므로 렌더러가 "채우는" 판단을 할 자리가 없다.
	if style.Shadow != nil || style.Opacity != nil || style.Rotate != nil || style.Flip != "" {
		t.Errorf("기본값이 선언을 더 냈다: %s", showShape(*style))
	}
}

// 컴포넌트 기본 `radius`는 **`rounded-rect`에서만** 확정 값이 된다. 검사를 인라인 단에만
// 거는 결정(shapeComponentDefault의 근거)의 실행 가능한 형태다: 저자가 고칠 수 없는 값에는
// 진단을 내지 않고 조용히 버린다.
func TestResolveShapeThemeRadiusFollowsType(t *testing.T) {
	rounded := resolvedShape(t, `type="rounded-rect"`)
	if rounded.Radius == nil || *rounded.Radius != soml.TokenRadius("md", 12) {
		t.Errorf("rounded-rect Radius = %v, want 토큰 md(12) — 두 타깃의 암묵 기본값이 달라 기본값이 필요하다", rounded.Radius)
	}
	// `type="rect"`는 진단 없이(저자가 고칠 것이 없다) radius 없이 확정된다.
	if rect := resolvedShape(t, `type="rect"`); rect.Radius != nil {
		t.Errorf("rect Radius = %v, want nil — 프리셋에 반경 조정값이 없다", rect.Radius)
	}
	// variant가 실은 radius도 같은 규칙을 지난다 (card는 radius lg를 정한다).
	if rect := resolvedShape(t, `type="rect" variant="card"`); rect.Radius != nil {
		t.Errorf("variant radius가 rect에 실렸다: %v", rect.Radius)
	}
	// **인라인 radius는 여전히 에러다** — 저자가 지울 수 있는 값이므로 진단이 고칠 방법을
	// 말한다 (TestResolveShapeRadiusNeedsRoundedRect가 문구·위치를 못 박는다).
	if _, diags := resolveSlide(t, shapeSlide(`type="rect" radius="md"`)); len(diags) != 1 {
		t.Errorf("진단 %d개, want 1개 (인라인 radius): %v", len(diags), diags)
	}
}

// minimal 테마의 도형 variant 표: 이름은 결정적 순서로 나열되고(경고 메시지의 근거),
// 각 variant는 **컴포넌트 기본값과 구별되는** 값을 정한다 — 같은 값을 정하는 variant는
// 우선순위 테스트가 아무것도 증명하지 못한다 (textVariants와 같은 기준).
func TestThemeShapeVariantsAreDistinctFromComponentDefault(t *testing.T) {
	theme := minimalTheme()
	variants := shapeVariants(theme)
	if len(variants) < 2 {
		t.Fatalf("variant %d개 — 캐스케이드를 증명하려면 2개 이상이 필요하다", len(variants))
	}
	seen := map[string]bool{}
	for _, v := range variants {
		if seen[v.name] {
			t.Errorf("variant 이름이 중복됐다: %q", v.name)
		}
		seen[v.name] = true
		if v.patch == (shapePatch{}) {
			t.Errorf("variant %q가 아무 필드도 정하지 않는다", v.name)
		}
		// radius를 정하는 variant도 구별되게 하려면 rounded-rect를 바닥으로 쓴다 (다른
		// 프리셋에서는 테마 단의 radius가 버려진다 — themeShapePatch).
		base := componentShapeStyle("rounded-rect")
		merged := base
		mergeShape(&merged, v.patch)
		if reflect.DeepEqual(merged, base) {
			t.Errorf("variant %q가 컴포넌트 기본 외관을 하나도 바꾸지 않는다", v.name)
		}
	}
	// 이름 목록은 표 순서를 그대로 준다 — 경고가 나열하는 목록이 결정적이어야 한다.
	names := shapeVariantNames(theme)
	if len(names) != len(variants) {
		t.Fatalf("이름 %d개, variant %d개", len(names), len(variants))
	}
	for i, v := range variants {
		if names[i] != v.name {
			t.Errorf("이름 목록 %d번 = %q, want %q", i, names[i], v.name)
		}
	}
	// 권장 어휘(PRD §13-6)의 카드 세 조각은 픽스처가 가정하고 있다 (_04_shape/007).
	for _, want := range []string{"card", "card-header", "card-body"} {
		if !seen[want] {
			t.Errorf("권장 어휘 %q가 표에 없다 (PRD §13-6)", want)
		}
	}
}

// 단마다 값이 어디서 왔는지 구별되는 표 — 해석을 통과한 뒤(확정 값)에도 순서가 유지되는지.
func TestResolveShapeVariantStage(t *testing.T) {
	// card: fill을 bg로, radius를 lg로 정한다. border는 정하지 않으므로 컴포넌트 기본값이
	// 남는다 (부분성).
	card := resolvedShape(t, `type="rounded-rect" variant="card"`)
	base := componentShapeStyle("rounded-rect")
	if card.Fill == nil || card.Fill.Color != soml.TokenColor("bg") {
		t.Errorf("variant fill = %v, want 토큰 bg", card.Fill)
	}
	if card.Radius == nil || *card.Radius != soml.TokenRadius("lg", 24) {
		t.Errorf("variant radius = %v, want 토큰 lg(24)", card.Radius)
	}
	if !reflect.DeepEqual(card.Border, base.Border) {
		t.Errorf("variant가 정하지 않은 border가 컴포넌트 기본값을 잃었다: %v", card.Border)
	}

	// card-header는 fill을 정하지 않는다 — 기본값 surface가 그대로 남고 테두리만 지워진다.
	header := resolvedShape(t, `type="rounded-rect" variant="card-header"`)
	if !reflect.DeepEqual(header.Fill, base.Fill) {
		t.Errorf("card-header fill = %v, want 컴포넌트 기본값 %v", header.Fill, base.Fill)
	}
	if header.Border == nil || !header.Border.None {
		t.Errorf("card-header border = %v, want none", header.Border)
	}

	// 인라인은 variant를 이긴다 (_04_shape/007의 오른쪽 열이 그림으로 보이는 장면).
	inline := resolvedShape(t, `type="rounded-rect" variant="card" fill="muted" radius="sm"`)
	if inline.Fill == nil || inline.Fill.Color != soml.TokenColor("muted") {
		t.Errorf("인라인 fill = %v, want 토큰 muted", inline.Fill)
	}
	if inline.Radius == nil || *inline.Radius != soml.TokenRadius("sm", 6) {
		t.Errorf("인라인 radius = %v, want 토큰 sm(6)", inline.Radius)
	}
}

// 모르는 variant는 에러가 아니라 **경고 + 컴포넌트 기본 외관 폴백**이다 (PRD §13-6) —
// 다른 테마로 만든 콘텐츠가 이 테마에서도 빌드되어야 한다.
func TestResolveShapeUnknownVariantWarnsAndFallsBack(t *testing.T) {
	attrs := `type="rect" variant="card-heder"`
	deck, diags := resolveSlide(t, shapeSlide(attrs))
	if len(diags) != 1 {
		t.Fatalf("진단 %d개, want 1개: %v", len(diags), diags)
	}
	diag := diags[0]
	if diag.Severity != soml.SeverityWarning {
		t.Errorf("severity = %s, want warning — 에러면 테마 간 이식성이 깨진다", diag.Severity)
	}
	if diag.Span.Start.Line != 2 || diag.Span.Start.Col != shapeCol(attrs, "card-heder") {
		t.Errorf("위치 = %s, want 2:%d (값을 가리켜야 한다)", diag.Span.Start, shapeCol(attrs, "card-heder"))
	}
	if !strings.Contains(diag.Message, `variant "card-heder"`) {
		t.Errorf("경고 문구 = %q, want variant 이름 포함", diag.Message)
	}
	if diags.HasErrors() {
		t.Error("경고만 있는 빌드는 실패하면 안 된다")
	}
	// --strict는 기존 경로를 그대로 탄다 (PRD §13-6).
	if !diags.Strict().HasErrors() {
		t.Error("--strict에서 경고가 에러로 승격되지 않았다")
	}
	// 폴백은 컴포넌트 기본 외관과 **깊은 비교로** 같다 — 렌더러가 빈 자리를 만나지 않는다.
	got := *findElement(t, deck, soml.ShapeElement).ShapeStyle()
	if want := componentShapeStyle("rect"); !reflect.DeepEqual(got, want) {
		t.Errorf("폴백 외관 = %s, want 컴포넌트 기본값 %s", showShape(got), showShape(want))
	}

	// variant가 틀려도 인라인 속성은 계속 해석한다 (역할이 틀렸을 때와 같은 정책).
	deck, diags = resolveSlide(t, shapeSlide(`type="rect" variant="card-heder" fill="accent1"`))
	if len(diags) != 1 || diags[0].Severity != soml.SeverityWarning {
		t.Fatalf("경고 1개여야 한다: %v", diags)
	}
	if style := findElement(t, deck, soml.ShapeElement).ShapeStyle(); style.Fill == nil ||
		style.Fill.Color != soml.TokenColor("accent1") {
		t.Errorf("인라인 fill이 사라졌다: %s", showShape(*style))
	}
}

// 경고가 나열하는 어휘는 **요소 종류에 맞는 목록**이다. variant 어휘가 텍스트용·도형용 두
// 벌이 된 순간 생긴 요구다: `<shape variant="kicker">`에 텍스트 목록을 나열하면 저자가 없는
// 방향으로 간다. 두 표가 같은 이름을 가져도 무관하다 — 요소 종류가 어느 표를 볼지 정한다.
func TestResolveVariantVocabulariesAreSeparate(t *testing.T) {
	theme := minimalTheme()
	shapeNames := strings.Join(shapeVariantNames(theme), ", ")
	textNames := strings.Join(textVariantNames(), ", ")

	// 도형에 텍스트 variant 이름을 쓰면 경고이고, 목록은 도형 어휘다.
	_, diags := resolveSlide(t, shapeSlide(`type="rect" variant="kicker"`))
	if len(diags) != 1 {
		t.Fatalf("진단 %d개, want 1개: %v", len(diags), diags)
	}
	if !strings.Contains(diags[0].Message, shapeNames) || strings.Contains(diags[0].Message, textNames) {
		t.Errorf("도형 경고 = %q, want 도형 목록(%s)", diags[0].Message, shapeNames)
	}

	// 텍스트에 도형 variant 이름을 써도 같다 (목록은 텍스트 어휘).
	_, diags = resolveSlide(t, "<slide id=\"t\">\n  <text variant=\"card\">내용</text>\n</slide>\n")
	if len(diags) != 1 {
		t.Fatalf("진단 %d개, want 1개: %v", len(diags), diags)
	}
	if !strings.Contains(diags[0].Message, textNames) || strings.Contains(diags[0].Message, shapeNames) {
		t.Errorf("텍스트 경고 = %q, want 텍스트 목록(%s)", diags[0].Message, textNames)
	}

	// 같은 이름이 두 표에 있어도 서로를 오염시키지 않는다 — 표를 하나로 합치면 이 단정이
	// 죽는다 (도형 variant가 텍스트 필드를 정할 수 없다는 것이 곧 두 표의 이유다).
	if _, ok := textVariantPatch("card"); ok {
		t.Error("텍스트 표에 도형 variant 이름이 들어왔다")
	}
	if _, ok := shapeVariantPatch(theme, "kicker"); ok {
		t.Error("도형 표에 텍스트 variant 이름이 들어왔다")
	}
}

// 도형 variant는 **자식 텍스트의 외관을 담지 않는다** (PRD §8.3과 어긋나는 결정 —
// shapeVariants의 근거). 담으면 캐스케이드가 트리를 타고 내려가는 상속이 되고, 상속은 한
// 요소 안에서 끝나는 지금의 병합과 다른 두 번째 메커니즘이다. 이 테스트가 그 상속이 몰래
// 들어오는 것을 잡는다.
func TestResolveShapeVariantDoesNotTouchChildText(t *testing.T) {
	for _, variant := range shapeVariantNames(minimalTheme()) {
		t.Run(variant, func(t *testing.T) {
			src := "<slide id=\"s\">\n  <shape type=\"rounded-rect\" variant=\"" + variant +
				"\"><p>내용</p></shape>\n</slide>\n"
			inShape := resolvedElement(t, src, "p").TextStyle()
			direct := tagStyle(t, "<slide id=\"s\">\n  <p>내용</p>\n</slide>\n", "p")
			if inShape == nil {
				t.Fatal("도형 안 <p>에 확정 외관이 없다")
			}
			if *inShape != *direct {
				t.Errorf("도형 안 <p> 외관 = %+v, 슬라이드 직속 = %+v", *inShape, *direct)
			}
		})
	}
}
