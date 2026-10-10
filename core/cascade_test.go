package core

import (
	"reflect"
	"strings"
	"testing"

	"github.com/k2sebeom/balpyo/soml"
)

// 우선순위: 인라인 > variant > 컴포넌트 기본값 > 역할 기본값. 병합은 mergeText 한 곳에서만
// 일어나므로 단을 하나씩 빼 가며 여기서 순서를 못 박는다.
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

// 부분성: 패치는 정한 필드만 얹고, "정하지 않음"(nil)은 아래 단을 지우지 않으며, 명시한 0은
// 0을 얹는다 — 이 구별이 없으면 어느 단이 이겼는지 알 수 없다.
func TestMergeTextAppliesOnlySetFields(t *testing.T) {
	base := textRoleDefaults(soml.RoleBody)

	got := base
	mergeText(&got, textPatch{})
	if got != base {
		t.Errorf("빈 패치가 외관을 바꿨다: %+v → %+v", base, got)
	}

	// 전체 패치는 전부를 바꾼다 — TextStyle에 필드가 늘었는데 textPatch가 따라가지 않으면 잡힌다.
	got = base
	mergeText(&got, textPatch{
		font:       ptr(soml.FontMono),
		size:       ptr(99.0),
		weight:     ptr(700),
		lineHeight: ptr(2.0),
		color:      ptr(soml.LiteralColor("ABCDEF")),
		align:      ptr(soml.AlignJustify),
	})
	want := soml.TextStyle{Role: soml.RoleBody, Font: soml.FontMono, Size: 99, Weight: 700,
		LineHeight: 2.0, Color: soml.LiteralColor("ABCDEF"), Align: soml.AlignJustify}
	if got != want {
		t.Errorf("전체 패치 = %+v, want %+v", got, want)
	}

	got = base
	mergeText(&got, textPatch{size: ptr(32.0)})
	want = base
	want.Size = 32
	if got != want {
		t.Errorf("부분 패치 = %+v, want %+v", got, want)
	}

	got = base
	mergeText(&got, textPatch{weight: ptr(0), size: ptr(0.0)})
	if got.Weight != 0 || got.Size != 0 {
		t.Errorf("명시된 0값이 적용되지 않았다: %+v", got)
	}
}

// componentShapeStyle는 컴포넌트 기본값만 얹은 확정 외관이다 — 도형 캐스케이드의 바닥이자
// 모르는 variant의 폴백 기대값이다 (흡수 모드: 바닥을 쓰는 표가 전부 텍스트 자식이나 빈 도형이다).
func componentShapeStyle(shapeType string) soml.ShapeStyle {
	style := soml.ShapeStyle{Type: shapeType}
	mergeShape(&style, themeShapePatch(shapeComponentDefault(minimalTheme()), shapeType, false))
	return style
}

// componentImageStyle는 그림 캐스케이드의 바닥이다. fit은 테마 단이 아니라 언어의 기본값이라
// 초기값으로 들어간다.
func componentImageStyle() soml.ImageStyle {
	style := soml.ImageStyle{Fit: soml.DefaultImageFit}
	mergeImage(&style, imageComponentDefault())
	return style
}

// 도형·그림의 우선순위는 텍스트와 같고 바닥만 다르다 — 역할이 없으므로 어느 단도 정하지 않은
// 필드는 비어 남는다. 세 단이 서로 다른 필드를 정해 필드마다 어느 단에서 왔는지 구별한다.
func TestMergeShapeAndImageCascadeOrder(t *testing.T) {
	soft := ptr(minimalShadows()[0])
	style := soml.ShapeStyle{Type: "rounded-rect"}
	mergeShape(&style, shapePatch{fill: colorFill("surface"), border: solidBorder(1, "muted"),
		radius: ptr(soml.TokenRadius("md", 12)), opacity: ptr(1.0)})
	mergeShape(&style, shapePatch{fill: colorFill("bg"), radius: ptr(soml.TokenRadius("lg", 24)), shadow: soft})
	mergeShape(&style, shapePatch{fill: colorFill("accent1"), rotate: ptr(45.0)})
	want := soml.ShapeStyle{
		Type:    "rounded-rect",
		Fill:    colorFill("accent1"),            // 인라인 > variant > 기본값
		Border:  solidBorder(1, "muted"),         // 기본값 (위 두 단이 말하지 않았다)
		Radius:  ptr(soml.TokenRadius("lg", 24)), // variant > 기본값
		Shadow:  soft,                            // variant
		Opacity: ptr(1.0),                        // 기본값
		Rotate:  ptr(45.0),                       // 인라인
	}
	if !reflect.DeepEqual(style, want) {
		t.Errorf("도형 단별 병합 = %s, want %s", show(style), show(want))
	}

	image := soml.ImageStyle{Src: "assets/wide.png", Fit: soml.FitCover}
	mergeImage(&image, imagePatch{border: solidBorder(1, "muted"), radius: ptr(soml.TokenRadius("md", 12)),
		opacity: ptr(1.0)})
	mergeImage(&image, imagePatch{border: solidBorder(2, "accent1"), radius: ptr(soml.TokenRadius("lg", 24)), shadow: soft})
	mergeImage(&image, imagePatch{border: ptr(soml.NoBorder()), rotate: ptr(45.0)})
	wantImage := soml.ImageStyle{
		Src:     "assets/wide.png",
		Fit:     soml.FitCover,
		Border:  ptr(soml.NoBorder()),            // 인라인 > variant > 기본값
		Radius:  ptr(soml.TokenRadius("lg", 24)), // variant > 기본값
		Shadow:  soft,                            // variant
		Opacity: ptr(1.0),                        // 기본값
		Rotate:  ptr(45.0),                       // 인라인
	}
	if !reflect.DeepEqual(image, wantImage) {
		t.Errorf("그림 단별 병합 = %s, want %s", show(image), show(wantImage))
	}
}

// 부분성(도형·그림): 빈 패치는 아무것도 바꾸지 않고, 전체 패치는 캐스케이드 밖이 아닌 필드를
// 전부 정하며(필드가 늘면 반사로 잡는다), 캐스케이드 밖 필드는 건드리지 않고, 명시한 0은 0이다.
func TestMergeShapeAndImageApplyOnlySetFields(t *testing.T) {
	assertFull := func(t *testing.T, v any, outside ...string) {
		t.Helper()
		value := reflect.ValueOf(v)
		for i := 0; i < value.NumField(); i++ {
			name := value.Type().Field(i).Name
			skip := false
			for _, o := range outside {
				skip = skip || o == name
			}
			if !skip && value.Field(i).IsZero() {
				t.Errorf("전체 패치가 %s를 정하지 않았다 — 패치에 필드가 빠졌다", name)
			}
		}
	}
	soft := ptr(minimalShadows()[0])

	shapeBase := componentShapeStyle("rounded-rect")
	shapeBase.ID, shapeBase.Box = "db", soml.Box{Width: ptr(soml.Px(220))}
	got := shapeBase
	mergeShape(&got, shapePatch{})
	if !reflect.DeepEqual(got, shapeBase) {
		t.Errorf("빈 패치가 도형 외관을 바꿨다: %s → %s", show(shapeBase), show(got))
	}
	mergeShape(&got, shapePatch{fill: ptr(soml.NoFill()), border: ptr(soml.NoBorder()),
		radius: ptr(soml.PxRadius(9)), shadow: soft, opacity: ptr(0.5), rotate: ptr(45.0), flip: ptr(soml.FlipBoth)})
	assertFull(t, got, "Type", "ID", "Alt", "Box")
	if got.ID != "db" || !reflect.DeepEqual(got.Box, shapeBase.Box) {
		t.Errorf("병합이 캐스케이드 밖 필드를 건드렸다: %s", show(got))
	}
	got = shapeBase
	mergeShape(&got, shapePatch{opacity: ptr(0.0), rotate: ptr(0.0)})
	if got.Opacity == nil || *got.Opacity != 0 || got.Rotate == nil || *got.Rotate != 0 ||
		!reflect.DeepEqual(got.Fill, shapeBase.Fill) || !reflect.DeepEqual(got.Radius, shapeBase.Radius) {
		t.Errorf("부분 패치 = %s", show(got))
	}

	imageBase := componentImageStyle()
	imageBase.Src, imageBase.Alt, imageBase.Box = "assets/wide.png", "설명", soml.Box{Width: ptr(soml.Px(220))}
	image := imageBase
	mergeImage(&image, imagePatch{})
	if !reflect.DeepEqual(image, imageBase) {
		t.Errorf("빈 패치가 그림 외관을 바꿨다: %s → %s", show(imageBase), show(image))
	}
	mergeImage(&image, imagePatch{border: ptr(soml.NoBorder()), radius: ptr(soml.PxRadius(9)),
		shadow: soft, opacity: ptr(0.5), rotate: ptr(45.0), flip: ptr(soml.FlipBoth)})
	assertFull(t, image, "Src", "ID", "Fit", "Alt", "Box")
	if image.Src != imageBase.Src || image.Fit != imageBase.Fit || image.Alt != imageBase.Alt ||
		!reflect.DeepEqual(image.Box, imageBase.Box) {
		t.Errorf("병합이 캐스케이드 밖 필드를 건드렸다: %s", show(image))
	}
	image = imageBase
	mergeImage(&image, imagePatch{opacity: ptr(0.0), rotate: ptr(0.0)})
	if image.Opacity == nil || *image.Opacity != 0 || image.Rotate == nil || *image.Rotate != 0 {
		t.Errorf("명시된 0값이 적용되지 않았다: %s", show(image))
	}
}

// `<shape type="rect"/>` 하나가 완전한 외관을 갖는다 — 값이 비어 남으면 렌더러가 채우게 되고,
// 두 타깃의 암묵 기본값이 다른 순간 갈라진다. 기본 radius는 rounded-rect에서만 확정되고 다른
// 프리셋에서는 진단 없이 버린다 (저자가 고칠 수 없는 값이다). 인라인 radius는 에러로 남는다.
func TestResolveShapeComponentDefault(t *testing.T) {
	style := resolvedShape(t, `type="rect"`)
	wantBorder := soml.Border{Width: 1, Style: soml.BorderSolid, Color: soml.TokenColor("muted")}
	if style.Fill == nil || style.Fill.None || style.Fill.Color != soml.TokenColor("surface") ||
		style.Border == nil || *style.Border != wantBorder {
		t.Errorf("기본 외관 = %s, want surface 채움 + 1 solid muted", show(*style))
	}
	if style.Radius != nil || style.Shadow != nil || style.Opacity != nil || style.Rotate != nil || style.Flip != "" {
		t.Errorf("기본값이 선언을 더 냈다: %s", show(*style))
	}
	if rounded := resolvedShape(t, `type="rounded-rect"`); rounded.Radius == nil || *rounded.Radius != soml.TokenRadius("md", 12) {
		t.Errorf("rounded-rect Radius = %v, want 토큰 md(12)", rounded.Radius)
	}
	if rect := resolvedShape(t, `type="rect" variant="card"`); rect.Radius != nil {
		t.Errorf("variant radius가 rect에 실렸다: %v", rect.Radius)
	}
	if _, diags := resolveSlide(t, shapeSlide(`type="rect" radius="md"`)); len(diags) != 1 {
		t.Errorf("진단 %d개, want 1개 (인라인 radius): %v", len(diags), diags)
	}
}

// 해석을 통과한 확정 값에서도 단 순서가 유지된다. card는 fill·radius를, card-header는 border만
// 정한다 — 정하지 않은 필드는 컴포넌트 기본값이 남는다.
func TestResolveShapeVariantStage(t *testing.T) {
	base := componentShapeStyle("rounded-rect")
	card := resolvedShape(t, `type="rounded-rect" variant="card"`)
	if card.Fill == nil || card.Fill.Color != soml.TokenColor("bg") ||
		card.Radius == nil || *card.Radius != soml.TokenRadius("lg", 24) || !reflect.DeepEqual(card.Border, base.Border) {
		t.Errorf("card = %s, want bg 채움 + lg 반경 + 기본 테두리", show(*card))
	}
	header := resolvedShape(t, `type="rounded-rect" variant="card-header"`)
	if !reflect.DeepEqual(header.Fill, base.Fill) || header.Border == nil || !header.Border.None {
		t.Errorf("card-header = %s, want 기본 채움 + 테두리 none", show(*header))
	}
	inline := resolvedShape(t, `type="rounded-rect" variant="card" fill="muted" radius="sm"`)
	if inline.Fill == nil || inline.Fill.Color != soml.TokenColor("muted") ||
		inline.Radius == nil || *inline.Radius != soml.TokenRadius("sm", 6) {
		t.Errorf("인라인 = %s, want muted 채움 + sm 반경", show(*inline))
	}
}

// 모르는 variant는 경고 + 컴포넌트 기본 외관 폴백이다 (다른 테마로 만든 콘텐츠도
// 빌드되어야 한다). --strict는 에러로 승격하고, 인라인 속성은 계속 해석한다.
func TestResolveShapeUnknownVariantWarnsAndFallsBack(t *testing.T) {
	attrs := `type="rect" variant="card-heder"`
	deck, diags := resolveSlide(t, shapeSlide(attrs))
	if len(diags) != 1 {
		t.Fatalf("진단 %d개, want 1개: %v", len(diags), diags)
	}
	diag := diags[0]
	if diag.Severity != soml.SeverityWarning || diags.HasErrors() || !diags.Strict().HasErrors() {
		t.Errorf("severity = %s, want warning (--strict에서만 에러)", diag.Severity)
	}
	if diag.Span.Start.Line != 2 || diag.Span.Start.Col != shapeCol(attrs, "card-heder") {
		t.Errorf("위치 = %s, want 2:%d", diag.Span.Start, shapeCol(attrs, "card-heder"))
	}
	if !strings.Contains(diag.Message, `variant "card-heder"`) {
		t.Errorf("경고 문구 = %q, want variant 이름 포함", diag.Message)
	}
	if got, want := *findElement(t, deck, soml.ShapeElement).ShapeStyle(), componentShapeStyle("rect"); !reflect.DeepEqual(got, want) {
		t.Errorf("폴백 외관 = %s, want 컴포넌트 기본값 %s", show(got), show(want))
	}

	deck, diags = resolveSlide(t, shapeSlide(`type="rect" variant="card-heder" fill="accent1"`))
	if len(diags) != 1 || diags[0].Severity != soml.SeverityWarning {
		t.Fatalf("경고 1개여야 한다: %v", diags)
	}
	if style := findElement(t, deck, soml.ShapeElement).ShapeStyle(); style.Fill == nil ||
		style.Fill.Color != soml.TokenColor("accent1") {
		t.Errorf("인라인 fill이 사라졌다: %s", show(*style))
	}
}

// 경고가 나열하는 어휘는 요소 종류에 맞는 목록이다 — `<shape variant="kicker">`에 텍스트 목록을
// 나열하면 저자가 없는 방향으로 간다.
func TestResolveVariantVocabulariesAreSeparate(t *testing.T) {
	shapeNames := strings.Join(shapeVariantNames(minimalTheme()), ", ")
	textNames := strings.Join(textVariantNames(), ", ")
	for _, tc := range []struct{ src, want, not string }{
		{shapeSlide(`type="rect" variant="kicker"`), shapeNames, textNames},
		{"<slide id=\"t\">\n  <text variant=\"card\">내용</text>\n</slide>\n", textNames, shapeNames},
	} {
		_, diags := resolveSlide(t, tc.src)
		if len(diags) != 1 || !strings.Contains(diags[0].Message, tc.want) || strings.Contains(diags[0].Message, tc.not) {
			t.Errorf("경고 = %v, want 목록 %q만", diags, tc.want)
		}
	}
}

// 도형 variant는 자식 텍스트의 외관을 담지 않는다 — 담으면 캐스케이드가 트리를 타고 내려가는
// 두 번째 메커니즘(상속)이 된다.
func TestResolveShapeVariantDoesNotTouchChildText(t *testing.T) {
	direct := tagStyle(t, "<slide id=\"s\">\n  <p>내용</p>\n</slide>\n", "p")
	for _, variant := range shapeVariantNames(minimalTheme()) {
		src := "<slide id=\"s\">\n  <shape type=\"rounded-rect\" variant=\"" + variant +
			"\"><p>내용</p></shape>\n</slide>\n"
		if inShape := resolvedElement(t, src, "p").TextStyle(); inShape == nil || *inShape != *direct {
			t.Errorf("%s: 도형 안 <p> 외관 = %+v, 슬라이드 직속 = %+v", variant, inShape, *direct)
		}
	}
}
