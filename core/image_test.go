package core

import (
	"reflect"
	"strings"
	"testing"

	"github.com/k2sebeom/balpyo/soml"
)

// imageSlide는 그림 하나(void)를 담은 슬라이드다 — 그림은 항상 2행 3열에서 시작한다.
func imageSlide(attrs string) string {
	return "<slide id=\"image\">\n  <image " + attrs + "/>\n</slide>\n"
}

// imageCol은 속성 문자열 안 needle이 놓이는 열이다 (shapeCol의 이미지 판).
func imageCol(attrs, needle string) int {
	index := strings.Index(attrs, needle)
	if index < 0 {
		panic("needle이 속성에 없다: " + needle)
	}
	return 3 + len("<image ") + index
}

// resolvedImage는 진단 없이 해석된 그림의 확정 외관이다.
func resolvedImage(t *testing.T, attrs string) *soml.ImageStyle {
	t.Helper()
	style := resolvedElement(t, imageSlide(attrs), soml.ImageElement).ImageStyle()
	if style == nil {
		t.Fatalf("<image %s>에 확정 외관이 없다", attrs)
	}
	return style
}

// wantImageStyle는 표의 항목(컴포넌트 기본값 위의 델타)을 기대값으로 바꾼다 (wantShapeStyle의 이미지 판).
func wantImageStyle(delta soml.ImageStyle) soml.ImageStyle {
	style := componentImageStyle()
	var flip *soml.Flip
	if delta.Flip != "" {
		flip = &delta.Flip
	}
	mergeImage(&style, imagePatch{border: delta.Border, radius: delta.Radius, shadow: delta.Shadow,
		opacity: delta.Opacity, rotate: delta.Rotate, flip: flip})
	style.Src, style.ID, style.Alt, style.Box = delta.Src, delta.ID, delta.Alt, delta.Box
	if delta.Fit != "" {
		style.Fit = delta.Fit
	}
	return style
}

// image 속성 표 전수 — 속성 → 확정 ImageStyle 필드.
func TestResolveImageStyle(t *testing.T) {
	softShadow := minimalShadows()[0]
	const src = `src="assets/wide.png"`
	cases := []struct {
		name  string
		attr  string
		attrs string
		want  soml.ImageStyle
	}{
		{"src", "src", src, soml.ImageStyle{Src: "assets/wide.png"}},
		{"id", "id", src + ` id="icon"`, soml.ImageStyle{Src: "assets/wide.png", ID: "icon"}},
		{"fit cover", "fit", src + ` fit="cover"`,
			soml.ImageStyle{Src: "assets/wide.png", Fit: soml.FitCover}},
		{"alt", "alt", src + ` alt="a chart"`,
			soml.ImageStyle{Src: "assets/wide.png", Alt: "a chart"}},

		{"variant", "variant", src + ` variant="framed"`,
			soml.ImageStyle{Src: "assets/wide.png", Border: solidBorder(1, "muted"),
				Radius: ptr(soml.TokenRadius("md", 12)), Shadow: &softShadow}},

		{"border 토큰 색", "border", src + ` border="1 solid muted"`,
			soml.ImageStyle{Src: "assets/wide.png", Border: &soml.Border{
				Width: 1, Style: soml.BorderSolid, Color: soml.TokenColor("muted")}}},
		{"border none", "border", src + ` border="none"`,
			soml.ImageStyle{Src: "assets/wide.png", Border: ptr(soml.NoBorder())}},
		// radius는 그림에서 프리셋 제약 없이 유효하다 — 그림은 자기 클립을 갖는다 (프리셋 제약은 도형의 것).
		{"radius 토큰", "radius", src + ` radius="lg"`,
			soml.ImageStyle{Src: "assets/wide.png", Radius: ptr(soml.TokenRadius("lg", 24))}},
		{"radius 수치", "radius", src + ` radius="12"`,
			soml.ImageStyle{Src: "assets/wide.png", Radius: ptr(soml.PxRadius(12))}},
		{"shadow 토큰", "shadow", src + ` shadow="soft"`,
			soml.ImageStyle{Src: "assets/wide.png", Shadow: &softShadow}},
		{"opacity", "opacity", src + ` opacity="0.4"`,
			soml.ImageStyle{Src: "assets/wide.png", Opacity: ptr(0.4)}},
		{"rotate", "rotate", src + ` rotate="-15"`,
			soml.ImageStyle{Src: "assets/wide.png", Rotate: ptr(-15.0)}},
		{"flip", "flip", src + ` flip="h"`,
			soml.ImageStyle{Src: "assets/wide.png", Flip: soml.FlipH}},

		{"pad", "pad", src + ` pad="md"`,
			soml.ImageStyle{Src: "assets/wide.png", Box: soml.Box{Pad: &soml.Pad{
				Top: soml.SpaceLength("md", 24), Right: soml.SpaceLength("md", 24),
				Bottom: soml.SpaceLength("md", 24), Left: soml.SpaceLength("md", 24)}}}},
		{"w fill", "w", src + ` w="fill"`,
			soml.ImageStyle{Src: "assets/wide.png", Box: soml.Box{Width: ptr(soml.Percent(100))}}},
		{"h 수치", "h", src + ` h="360"`,
			soml.ImageStyle{Src: "assets/wide.png", Box: soml.Box{Height: ptr(soml.Px(360))}}},
		{"min-w", "min-w", src + ` min-w="120"`,
			soml.ImageStyle{Src: "assets/wide.png", Box: soml.Box{MinWidth: ptr(soml.Px(120))}}},
		{"min-h", "min-h", src + ` min-h="80"`,
			soml.ImageStyle{Src: "assets/wide.png", Box: soml.Box{MinHeight: ptr(soml.Px(80))}}},
		{"max-w", "max-w", src + ` max-w="360"`,
			soml.ImageStyle{Src: "assets/wide.png", Box: soml.Box{MaxWidth: ptr(soml.Px(360))}}},
		{"max-h", "max-h", src + ` max-h="240"`,
			soml.ImageStyle{Src: "assets/wide.png", Box: soml.Box{MaxHeight: ptr(soml.Px(240))}}},
		{"ratio", "ratio", src + ` ratio="16:9"`,
			soml.ImageStyle{Src: "assets/wide.png", Box: soml.Box{Ratio: &soml.Ratio{W: 16, H: 9}}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := resolvedImage(t, tc.attrs)
			if want := wantImageStyle(tc.want); !reflect.DeepEqual(*got, want) {
				t.Errorf("이미지 외관 = %s, want %s", show(*got), show(want))
			}
			elem := resolvedElement(t, imageSlide(tc.attrs), soml.ImageElement)
			if elem.Placement() != nil || elem.ShapeStyle() != nil {
				t.Errorf("배치나 도형 외관이 실렸다: %s", show(elem.Appearance))
			}
		})
	}

	covered := map[string]bool{}
	for _, tc := range cases {
		covered[tc.attr] = true
	}
	for _, attr := range soml.ImageAttrs() {
		if !covered[attr] {
			t.Errorf("이미지 속성 %q의 해석 테스트가 없다", attr)
		}
	}
}

// src의 기준은 프로젝트 루트이고 키는 path.Clean 정규화다 — 두 표기가 두 미디어 파트가 되면 안 된다.
func TestResolveImageSrcNormalization(t *testing.T) {
	cases := []struct{ raw, want string }{
		{"assets/wide.png", "assets/wide.png"},
		{"./assets/wide.png", "assets/wide.png"},
		{"assets//wide.png", "assets/wide.png"},
		{"assets/deep/../wide.png", "assets/wide.png"},
		{"wide.png", "wide.png"},
		// assets/ 아래일 것을 강제하지 않는다 (프로젝트 구조는 관례다).
		{"img/ch1/Photo.JPG", "img/ch1/Photo.JPG"},
	}
	for _, tc := range cases {
		t.Run(tc.raw, func(t *testing.T) {
			if got := resolvedImage(t, `src="`+tc.raw+`"`).Src; got != tc.want {
				t.Errorf("src = %q, want %q", got, tc.want)
			}
		})
	}
}

// 그림 값 층의 진단 — 값 스팬(needle, 없으면 태그 이름 2:4)을 가리키고 고칠 방법을 말한다.
// 틀린 값은 확정 외관에 실리지 않는다 (src는 빈 값, fit은 언어 기본값).
func TestResolveImageDiagnostics(t *testing.T) {
	const src = `src="assets/wide.png" `
	cases := []struct {
		name, attrs, needle string
		col                 int // needle로 셀 수 없는 위치
		contains            []string
	}{
		{name: "src 없음", attrs: `fit="cover"`, contains: []string{"needs a src", "cannot be guessed"}},
		{name: "src 값 없음", attrs: `src`, needle: "src", contains: []string{"the src attribute needs a value"}},
		{name: "src 빈 값", attrs: `src=""`, col: imageCol(`src=""`, `""`) + 1, contains: []string{"src is empty"}},
		{name: "절대 경로", attrs: `src="/Users/me/assets/a.png"`, needle: "/Users", contains: []string{"is absolute", "relative to the project root"}},
		{name: "프로젝트 밖", attrs: `src="../shared/a.png"`, needle: "..", contains: []string{"points outside the project"}},
		{name: "http", attrs: `src="http://example.com/a.png"`, needle: "http", contains: []string{"cannot be a URL", "offline"}},
		{name: "https", attrs: `src="https://example.com/a.png"`, needle: "https", contains: []string{"cannot be a URL"}},
		{name: "data URL", attrs: `src="data:image/png;base64,iVBORw0KGgo="`, needle: "data:",
			contains: []string{"cannot be a data: URL", "commit the image as a file"}},
		{name: "확장자 없음", attrs: `src="assets/photo"`, needle: "assets/photo", contains: []string{"has no extension", ".png"}},
		{name: "svg", attrs: `src="assets/logo.svg"`, needle: "assets/logo.svg", contains: []string{"v1 cannot render", "png", "jpeg"}},
		{name: "gif", attrs: `src="assets/anim.gif"`, needle: "assets/anim.gif", contains: []string{"v1 cannot render"}},
		{name: "webp", attrs: `src="assets/photo.webp"`, needle: "assets/photo.webp", contains: []string{"v1 cannot render"}},
		// CSS object-fit의 나머지 값은 PPTX에 대응이 없다는 사실까지 진단이 말한다.
		{name: "fit fill", attrs: src + `fit="fill"`, needle: "fill", contains: []string{"object-fit", "cover, contain", "ratio"}},
		{name: "fit none", attrs: src + `fit="none"`, needle: "none", contains: []string{"object-fit", "cover, contain"}},
		{name: "fit scale-down", attrs: src + `fit="scale-down"`, needle: "scale-down", contains: []string{"object-fit", "cover, contain"}},
		{name: "fit 어휘 밖", attrs: src + `fit="crop"`, needle: "crop", contains: []string{"unknown fit", "cover, contain"}},
		// 빈 alt는 장식 이미지의 표기라 받지만, 값 없는 alt는 뜻을 정할 수 없다.
		{name: "값 없는 alt", attrs: src + `alt`, needle: "alt", contains: []string{"the alt attribute needs a value", `alt=""`}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			deck, diag := oneDiag(t, imageSlide(tc.attrs))
			col := tc.col
			switch {
			case col != 0:
			case tc.needle == "":
				col = 4
			default:
				col = imageCol(tc.attrs, tc.needle)
			}
			assertDiag(t, diag, 2, col, tc.contains...)
			style := findElement(t, deck, soml.ImageElement).ImageStyle()
			if style == nil || (strings.HasPrefix(tc.attrs, src) && style.Fit != soml.DefaultImageFit) ||
				(!strings.HasPrefix(tc.attrs, src) && style.Src != "") {
				t.Errorf("무효한 값이 실렸다: %s", show(style))
			}
		})
	}
}

// fit 어휘 2종과 언어의 기본값(contain) — 저자가 자르라고 말하지 않았으면 자르지 않는다.
// alt의 빈 값은 생략과 같다.
func TestResolveImageFitAndAlt(t *testing.T) {
	for _, fit := range soml.ImageFits() {
		if got := resolvedImage(t, `src="assets/wide.png" fit="`+string(fit)+`"`).Fit; got != fit {
			t.Errorf("fit = %q, want %q", got, fit)
		}
	}
	for _, attrs := range []string{`src="assets/wide.png"`, `src="assets/wide.png" variant="framed"`} {
		if got := resolvedImage(t, attrs).Fit; got != soml.FitContain {
			t.Errorf("%s: fit = %q, want %q", attrs, got, soml.FitContain)
		}
	}
	for _, attrs := range []string{`src="assets/wide.png" alt=""`, `src="assets/wide.png"`} {
		if got := resolvedImage(t, attrs).Alt; got != "" {
			t.Errorf("%s: alt = %q, want 빈 문자열", attrs, got)
		}
	}
}

// 외관 값 문법과 진단 문구는 도형과 같다 — 해석 함수가 두 벌이 되면 여기서 죽는다. 속성마다
// 그 필드만 대조한다 (컴포넌트 기본값은 두 요소가 다르다).
func TestResolveImageAppearanceSharesShapeSyntax(t *testing.T) {
	decls := []string{`border="2.5 dashed #6B7280"`, `border="none"`, `radius="lg"`, `radius="12"`,
		`shadow="soft"`, `opacity="0"`, `opacity="1"`, `rotate="45"`, `rotate="-7.5"`, `flip="hv"`,
		`pad="1 2 3 4%"`, `w="fill"`, `h="240"`, `min-w="120"`, `min-h="80"`, `max-w="360"`,
		`max-h="240"`, `ratio="4:3"`}
	for _, decl := range decls {
		t.Run(decl, func(t *testing.T) {
			name := strings.SplitN(decl, "=", 2)[0]
			shapeType := "rect"
			if name == "radius" {
				shapeType = "rounded-rect"
			}
			image := resolvedImage(t, `src="assets/wide.png" `+decl)
			shape := resolvedShape(t, `type="`+shapeType+`" `+decl)
			var same bool
			switch name {
			case "border":
				same = reflect.DeepEqual(image.Border, shape.Border)
			case "radius":
				same = reflect.DeepEqual(image.Radius, shape.Radius)
			case "shadow":
				same = reflect.DeepEqual(image.Shadow, shape.Shadow)
			case "opacity":
				same = reflect.DeepEqual(image.Opacity, shape.Opacity)
			case "rotate":
				same = reflect.DeepEqual(image.Rotate, shape.Rotate)
			case "flip":
				same = image.Flip == shape.Flip
			default:
				same = reflect.DeepEqual(image.Box, shape.Box)
			}
			if !same {
				t.Errorf("같은 표기가 다르게 해석됐다: 이미지 %s / 도형 %s",
					show(*image), show(*shape))
			}
		})
	}

	// variant는 테마 표의 어휘가 달라 대조하지 않는다.
	covered := map[string]bool{"variant": true}
	for _, decl := range decls {
		covered[strings.SplitN(decl, "=", 2)[0]] = true
	}
	for _, attr := range soml.ImageAppearanceAttrs() {
		if !covered[attr] {
			t.Errorf("외관 속성 %q의 공유 테스트가 없다", attr)
		}
	}
	for _, attr := range soml.BoxAttrs() {
		if !covered[attr] {
			t.Errorf("박스 속성 %q의 공유 테스트가 없다", attr)
		}
	}

	errCases := []struct {
		decl     string
		needle   string
		contains []string
	}{
		{`border="solid 3"`, "solid 3", []string{"has 3 parts", "1 solid muted"}},
		{`border="1 dashdot muted"`, "dashdot", []string{"unknown border style"}},
		{`border="1 solid brand"`, "brand", []string{"border: color token", "is not in the theme"}},
		{`shadow="hard"`, "hard", []string{"unknown shadow token", "soft"}},
		{`opacity="50%"`, "50%", []string{"opacity cannot use %"}},
		{`rotate="45deg"`, "45deg", []string{"unitless angle"}},
		{`flip="both"`, "both", []string{"unknown flip", "h, v, hv"}},
		{`radius="huge"`, "huge", []string{"unknown radius token", "sm, md, lg"}},
		{`radius="-4"`, "-4", []string{"cannot be negative"}},
	}
	for _, tc := range errCases {
		t.Run(tc.decl, func(t *testing.T) {
			attrs := `src="assets/wide.png" ` + tc.decl
			_, imageDiag := oneDiag(t, imageSlide(attrs))
			assertDiag(t, imageDiag, 2, imageCol(attrs, tc.needle), tc.contains...)
			_, shapeDiag := oneDiag(t, shapeSlide(`type="rounded-rect" `+tc.decl))
			if imageDiag.Message != shapeDiag.Message {
				t.Errorf("진단 문구가 갈라졌다:\n이미지 %q\n도형   %q",
					imageDiag.Message, shapeDiag.Message)
			}
		})
	}
}

// 모르는 variant는 경고 + 빈 패치다 (variant 이름은 테마의 것이다). 나열하는 어휘는 그림의 것이어야 한다 — 도형
// 목록을 나열하면 저자를 그 테마에 없는 방향으로 보낸다.
func TestResolveImageUnknownVariantWarns(t *testing.T) {
	attrs := `src="assets/wide.png" variant="framd"`
	deck, diags := resolveSlide(t, imageSlide(attrs))
	if len(diags) != 1 {
		t.Fatalf("진단 %d개, want 1개: %v", len(diags), diags)
	}
	if diags[0].Severity != soml.SeverityWarning {
		t.Errorf("심각도 = %v, want warning (%s)", diags[0].Severity, diags[0].String())
	}
	if diags[0].Span.Start.Col != imageCol(attrs, "framd") {
		t.Errorf("위치 = %s, want 2:%d", diags[0].Span.Start, imageCol(attrs, "framd"))
	}
	for _, want := range []string{"is not in the theme", "continuing with the default appearance", "framed"} {
		if !strings.Contains(diags[0].Message, want) {
			t.Errorf("진단 = %q, want %q 포함", diags[0].String(), want)
		}
	}
	for _, name := range shapeVariantNames(minimalTheme()) {
		if strings.Contains(diags[0].Message, name) {
			t.Errorf("진단이 도형 variant %q를 나열했다: %s", name, diags[0].String())
		}
	}
	style := findElement(t, deck, soml.ImageElement).ImageStyle()
	if want := componentImageStyle(); style == nil ||
		!reflect.DeepEqual(soml.ImageStyle{Src: style.Src, Fit: style.Fit,
			Border: style.Border, Radius: style.Radius, Shadow: style.Shadow},
			soml.ImageStyle{Src: "assets/wide.png", Fit: want.Fit,
				Border: want.Border, Radius: want.Radius, Shadow: want.Shadow}) {
		t.Errorf("모르는 variant가 외관을 바꿨다: %s", show(*style))
	}
}

// 한 요소가 그림 외관 + 자기 박스 + 자식 배치를 함께 쓴다 (elemAppearance는 필드 채우기다).
func TestResolveImageKeepsPlacement(t *testing.T) {
	src := "<slide id=\"s\">\n  <grid cols=\"3\"><image src=\"assets/wide.png\" radius=\"md\" pad=\"md\" span=\"2\"/></grid>\n</slide>\n"
	elem := resolvedElement(t, src, soml.ImageElement)
	style, placement := elem.ImageStyle(), elem.Placement()
	if style == nil || style.Radius == nil || *style.Radius != soml.TokenRadius("md", 12) {
		t.Errorf("외관이 없다: %+v", style)
	}
	if style.Box.Pad == nil || style.Box.Pad.Top != soml.SpaceLength("md", 24) {
		t.Errorf("박스가 없다: %+v", style.Box)
	}
	if placement == nil || placement.Span == nil || *placement.Span != 2 {
		t.Errorf("배치가 없다: %+v", placement)
	}
}

// 그림 캐스케이드의 단 순서는 도형과 같다 — 인라인 > variant > 컴포넌트 기본값(비어 있음).
// 정하지 않은 필드와 틀린 인라인 값은 아래 단(variant)을 남긴다.
func TestResolveImageVariantStage(t *testing.T) {
	base := componentImageStyle()
	framed := resolvedImage(t, `src="assets/wide.png" variant="framed"`)
	if framed.Border == nil || framed.Radius == nil || framed.Shadow == nil {
		t.Fatalf("variant가 액자를 정하지 않았다: %s", show(*framed))
	}
	if base.Border != nil || base.Radius != nil || base.Shadow != nil {
		t.Errorf("컴포넌트 기본값이 장식을 가졌다: %s", show(base))
	}

	inline := resolvedImage(t, `src="assets/wide.png" variant="framed" border="none" radius="sm"`)
	if inline.Border == nil || !inline.Border.None {
		t.Errorf("인라인 border = %v, want none", inline.Border)
	}
	if inline.Radius == nil || *inline.Radius != soml.TokenRadius("sm", 6) {
		t.Errorf("인라인 radius = %v, want 토큰 sm(6)", inline.Radius)
	}
	if !reflect.DeepEqual(inline.Shadow, framed.Shadow) {
		t.Errorf("인라인이 말하지 않은 shadow가 variant 단을 잃었다: %v", inline.Shadow)
	}
	deck, _ := oneDiag(t, imageSlide(`src="assets/wide.png" variant="framed" border="0 solid muted"`))
	style := findElement(t, deck, soml.ImageElement).ImageStyle()
	if style == nil || !reflect.DeepEqual(style.Border, framed.Border) {
		t.Errorf("border = %+v, want variant 단의 값", style)
	}
}
