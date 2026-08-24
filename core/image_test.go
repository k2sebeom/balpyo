package core

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/k2sebeom/balpyo/soml"
)

// imageSlide는 그림 하나(void)를 담은 슬라이드 원문이다. 들여쓰기를 2칸으로 고정하는 이유는
// shapeSlide와 같다 — 진단 위치 단정이 표에서 읽히게 하기 위함이다 (그림은 항상 2행 3열에서
// 시작한다).
func imageSlide(attrs string) string {
	return "<slide id=\"image\">\n  <image " + attrs + "/>\n</slide>\n"
}

// imageCol은 속성 문자열 안 needle이 놓이는 열이다 (shapeCol의 이미지 판) — 속성은 ASCII여야
// 바이트 오프셋이 곧 룬 열이다 (soml.Pos.Col).
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

// wantImageStyle는 표의 항목을 기대값으로 바꾼다 — 항목은 **컴포넌트 기본값 위의 델타**다
// (wantShapeStyle과 같은 장치). 얹는 데 운영 코드의 mergeImage를 쓰는 이유는 병합 규칙을
// 테스트가 다시 쓰지 않기 위함이다.
func wantImageStyle(delta soml.ImageStyle) soml.ImageStyle {
	style := componentImageStyle()
	var flip *soml.Flip
	if delta.Flip != "" {
		flip = &delta.Flip
	}
	mergeImage(&style, imagePatch{border: delta.Border, radius: delta.Radius, shadow: delta.Shadow,
		opacity: delta.Opacity, rotate: delta.Rotate, flip: flip})
	// src·fit·alt와 박스는 캐스케이드 밖이다 (imagePatch의 근거 — 테마가 정할 수 없는 값).
	style.Src, style.Alt, style.Box = delta.Src, delta.Alt, delta.Box
	if delta.Fit != "" {
		style.Fit = delta.Fit
	}
	return style
}

// PRD §7.5·§7.6 표의 image 행을 전수로 못 박는다 — 속성 → **확정 ImageStyle 필드**.
// 렌더러는 Attrs를 다시 읽지 않으므로(soml.Appearance의 계약) 이 표가 곧 CSS·OOXML의 입력이다.
func TestResolveImageStyle(t *testing.T) {
	softShadow := minimalShadows()[0]
	const src = `src="assets/wide.png"`
	cases := []struct {
		name  string
		attr  string
		attrs string
		want  soml.ImageStyle
	}{
		// 고유 — src는 정규화 키로 남고(에셋 표의 키다), fit은 어휘 2종, alt은 원문이다.
		{"src", "src", src, soml.ImageStyle{Src: "assets/wide.png"}},
		{"fit cover", "fit", src + ` fit="cover"`,
			soml.ImageStyle{Src: "assets/wide.png", Fit: soml.FitCover}},
		{"alt", "alt", src + ` alt="a chart"`,
			soml.ImageStyle{Src: "assets/wide.png", Alt: "a chart"}},

		// variant는 테마의 외관 번들이다 — `frame` 속성을 열지 않은 결정의 실물이다
		// (액자는 속성이 아니라 테마가 정의한 번들이다).
		{"variant", "variant", src + ` variant="framed"`,
			soml.ImageStyle{Src: "assets/wide.png", Border: solidBorder(1, "muted"),
				Radius: ptr(soml.TokenRadius("md", 12)), Shadow: &softShadow}},

		// 외관 — 값 문법은 도형과 **같은 함수**를 지난다 (아래 대조 테스트가 짝이다).
		{"border 토큰 색", "border", src + ` border="1 solid muted"`,
			soml.ImageStyle{Src: "assets/wide.png", Border: &soml.Border{
				Width: 1, Style: soml.BorderSolid, Color: soml.TokenColor("muted")}}},
		{"border none", "border", src + ` border="none"`,
			soml.ImageStyle{Src: "assets/wide.png", Border: ptr(soml.NoBorder())}},
		// radius는 이미지에서 **프리셋 제약 없이** 유효하다 (ADR 0019의 적용 범위는 도형이다).
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

		// 박스 — 레이아웃 컨테이너·도형과 **같은 헬퍼**를 지난다.
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
				t.Errorf("이미지 외관 = %s, want %s", showImage(*got), showImage(want))
			}
			// 외관은 자식 배치가 아니다 — 둘이 섞이면 렌더러가 외관을 자기 자리로 낸다.
			if placement := resolvedElement(t, imageSlide(tc.attrs), soml.ImageElement).Placement(); placement != nil {
				t.Errorf("자식 배치가 붙었다: %+v", *placement)
			}
			// 도형 자리는 비어 있다 — 두 확정 값이 형제라는 계약.
			if shape := resolvedElement(t, imageSlide(tc.attrs), soml.ImageElement).ShapeStyle(); shape != nil {
				t.Errorf("도형 외관이 실렸다: %+v", *shape)
			}
		})
	}

	// 표가 카탈로그 전체를 덮는지 — 파서가 연 속성을 값 층이 해석하지 않으면 저자가 쓴 것이
	// 조용히 사라진다 (도형·컨테이너 커버리지 테스트와 같은 장치).
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

// `src` 해석 규칙: 기준은 **프로젝트 루트**이고 키는 path.Clean 정규화다 (두 표기가 두 파트가
// 되면 산출물에 같은 바이트가 두 번 들어간다).
func TestResolveImageSrcNormalization(t *testing.T) {
	cases := []struct{ raw, want string }{
		{"assets/wide.png", "assets/wide.png"},
		{"./assets/wide.png", "assets/wide.png"},
		{"assets//wide.png", "assets/wide.png"},
		{"assets/deep/../wide.png", "assets/wide.png"},
		{"wide.png", "wide.png"},
		// 하위 디렉토리는 프로젝트 안이면 어디든 좋다 — `assets/` 아래일 것을 강제하지
		// 않는다 (PRD §6.1의 디렉토리 구조는 관례다).
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

// 막는 것과 그 이유 — 진단은 전부 **값 스팬**을 가리키고 고칠 방법을 말한다.
func TestResolveImageSrcErrors(t *testing.T) {
	cases := []struct {
		name     string
		attrs    string
		needle   string
		contains []string
	}{
		{"src 없음", `fit="cover"`, "", []string{"src가 필요하다", "추측할 수 없다"}},
		{"값 없음", `src`, "src", []string{"src 속성에는 값이 필요하다"}},
		{"빈 값", `src=""`, `""`, []string{"src가 비어 있다"}},
		{"절대 경로", `src="/Users/me/assets/a.png"`, "/Users", []string{"절대 경로", "프로젝트 루트 기준"}},
		{"프로젝트 밖", `src="../shared/a.png"`, "..", []string{"프로젝트 밖을 가리킨다"}},
		{"http", `src="http://example.com/a.png"`, "http", []string{"URL은 쓸 수 없다", "오프라인"}},
		{"https", `src="https://example.com/a.png"`, "https", []string{"URL은 쓸 수 없다"}},
		{"data URL", `src="data:image/png;base64,iVBORw0KGgo="`, "data:",
			[]string{"data: URL은 쓸 수 없다", "파일로 커밋한다"}},
		{"확장자 없음", `src="assets/photo"`, "assets/photo", []string{"확장자가 없다", ".png"}},
		{"svg", `src="assets/logo.svg"`, "assets/logo.svg",
			[]string{"v1이 그리지 못한다", "png", "jpeg"}},
		{"gif", `src="assets/anim.gif"`, "assets/anim.gif", []string{"v1이 그리지 못한다"}},
		{"webp", `src="assets/photo.webp"`, "assets/photo.webp", []string{"v1이 그리지 못한다"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			deck, diags := resolveSlide(t, imageSlide(tc.attrs))
			if len(diags) != 1 {
				t.Fatalf("진단 %d개, want 1개: %v", len(diags), diags)
			}
			col := 4 // src가 아예 없으면 진단은 **태그 이름**을 가리킨다.
			if tc.needle != "" {
				col = imageCol(tc.attrs, tc.needle)
				if strings.HasPrefix(tc.needle, `"`) {
					col++ // 값 전체를 가리키는 진단은 인용 부호 안쪽에서 시작한다.
				}
			}
			assertImageDiag(t, diags[0], col, tc.contains...)
			// 틀린 src는 확정 외관에 실리지 않는다 — 그래야 렌더러의 "그릴 것이 없다" 가드가
			// 마지막 그물로 남는다.
			if style := findElement(t, deck, soml.ImageElement).ImageStyle(); style == nil || style.Src != "" {
				t.Errorf("무효한 src가 실렸다: %+v", style)
			}
		})
	}
}

// `fit` 어휘 2종과 **언어의 기본값**(contain). 어휘 밖은 에러이고, CSS object-fit의 나머지
// 값은 PPTX에 대응이 없다는 사실까지 진단이 말한다.
func TestResolveImageFit(t *testing.T) {
	for _, fit := range soml.ImageFits() {
		t.Run(string(fit), func(t *testing.T) {
			attrs := `src="assets/wide.png" fit="` + string(fit) + `"`
			if got := resolvedImage(t, attrs).Fit; got != fit {
				t.Errorf("fit = %q, want %q", got, fit)
			}
		})
	}

	// 인라인·variant·컴포넌트 기본값이 전부 없으면 contain이다 — 저자가 자르라고 말하지
	// 않았으면 자르지 않는다.
	if got := resolvedImage(t, `src="assets/wide.png"`).Fit; got != soml.FitContain {
		t.Errorf("기본 fit = %q, want %q", got, soml.FitContain)
	}
	if got := resolvedImage(t, `src="assets/wide.png" variant="framed"`).Fit; got != soml.FitContain {
		t.Errorf("variant가 fit을 바꿨다: %q", got)
	}

	cases := []struct {
		name     string
		value    string
		contains []string
	}{
		{"CSS 어휘 fill", "fill", []string{"object-fit", "cover, contain", "ratio"}},
		{"CSS 어휘 none", "none", []string{"object-fit", "cover, contain"}},
		{"CSS 어휘 scale-down", "scale-down", []string{"object-fit", "cover, contain"}},
		{"어휘 밖", "crop", []string{"알 수 없는 fit", "cover, contain"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			attrs := `src="assets/wide.png" fit="` + tc.value + `"`
			deck, diags := resolveSlide(t, imageSlide(attrs))
			if len(diags) != 1 {
				t.Fatalf("진단 %d개, want 1개: %v", len(diags), diags)
			}
			assertImageDiag(t, diags[0], imageCol(attrs, tc.value), tc.contains...)
			// 틀린 값은 실리지 않는다 — 언어의 기본값이 남는다.
			if style := findElement(t, deck, soml.ImageElement).ImageStyle(); style == nil ||
				style.Fit != soml.DefaultImageFit {
				t.Errorf("무효한 fit이 실렸다: %+v", style)
			}
		})
	}
}

// `alt`의 **빈 값은 뜻이 있다** — `<img alt="">`가 장식 이미지의 표기다. 그래서 다른 외관
// 속성처럼 "비어 있다"로 막지 않는다 (생략과 같은 값이 된다).
func TestResolveImageAlt(t *testing.T) {
	if got := resolvedImage(t, `src="assets/wide.png" alt=""`).Alt; got != "" {
		t.Errorf("alt = %q, want 빈 문자열", got)
	}
	if got := resolvedImage(t, `src="assets/wide.png"`).Alt; got != "" {
		t.Errorf("생략한 alt = %q, want 빈 문자열 (생략 ≡ alt=\"\")", got)
	}
	// 값 **없는** alt는 에러다 — 뜻을 정할 수 없는 형태다 (`alt` vs `alt=""`).
	attrs := `src="assets/wide.png" alt`
	_, diags := resolveSlide(t, imageSlide(attrs))
	if len(diags) != 1 {
		t.Fatalf("진단 %d개, want 1개: %v", len(diags), diags)
	}
	assertImageDiag(t, diags[0], imageCol(attrs, "alt"), "alt 속성에는 값이 필요하다", `alt=""`)
}

// 외관 값 문법이 도형과 **같은 확정 값**에 도달하는지 — 해석 함수가 두 벌이 되면 이 테스트가
// 먼저 죽는다 (같은 일을 하는 코드가 두 벌이 되는 것이 이 저장소에서 가장 비싼 실패다).
//
// 속성마다 그 필드만 대조한다: 나머지 필드는 컴포넌트 기본값이 갈라져 있고(도형은 채움·테두리·
// 반경을 갖고 그림은 아무 장식도 갖지 않는다) 그 차이는 이 테스트가 볼 것이 아니다.
func TestResolveImageAppearanceSharesShapeSyntax(t *testing.T) {
	decls := []string{`border="2.5 dashed #6B7280"`, `border="none"`, `radius="lg"`, `radius="12"`,
		`shadow="soft"`, `opacity="0"`, `opacity="1"`, `rotate="45"`, `rotate="-7.5"`, `flip="hv"`,
		`pad="1 2 3 4%"`, `w="fill"`, `h="240"`, `min-w="120"`, `min-h="80"`, `max-w="360"`,
		`max-h="240"`, `ratio="4:3"`}
	for _, decl := range decls {
		t.Run(decl, func(t *testing.T) {
			name := strings.SplitN(decl, "=", 2)[0]
			// radius는 도형에서 프리셋을 요구한다 (제약은 도형의 것이고 값 문법은 하나다).
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
				// 박스 속성은 레이아웃 컨테이너까지 셋이 같은 헬퍼를 지난다 (boxAttr).
				same = reflect.DeepEqual(image.Box, shape.Box)
			}
			if !same {
				t.Errorf("같은 표기가 다르게 해석됐다: 이미지 %s / 도형 %s",
					showImage(*image), showShape(*shape))
			}
		})
	}

	// 이미지 외관 카탈로그를 전부 덮었는지 — `variant`는 테마 표가 갈라져 있어(어휘가 다르다)
	// 여기서 대조할 것이 아니다.
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

	// 값 문법 위반의 진단도 같은 곳에서 나온다 — 문구가 갈라지면 저자가 같은 실수를 두 규칙으로
	// 읽는다.
	errCases := []struct {
		decl     string
		needle   string
		contains []string
	}{
		{`border="solid 3"`, `"solid 3"`, []string{"3조각이다", "1 solid muted"}},
		{`border="1 dashdot muted"`, "dashdot", []string{"알 수 없는 border 스타일"}},
		{`border="1 solid brand"`, "brand", []string{"border의 색 토큰", "테마에 없다"}},
		{`shadow="hard"`, "hard", []string{"알 수 없는 shadow 토큰", "soft"}},
		{`opacity="50%"`, "50%", []string{"opacity에 %는 쓸 수 없다"}},
		{`rotate="45deg"`, "45deg", []string{"단위 없는 각도 수치"}},
		{`flip="both"`, "both", []string{"알 수 없는 flip", "h, v, hv"}},
		{`radius="huge"`, "huge", []string{"알 수 없는 radius 토큰", "sm, md, lg"}},
		{`radius="-4"`, "-4", []string{"0보다 작을 수 없다"}},
	}
	for _, tc := range errCases {
		t.Run(tc.decl, func(t *testing.T) {
			attrs := `src="assets/wide.png" ` + tc.decl
			_, imageDiags := resolveSlide(t, imageSlide(attrs))
			if len(imageDiags) != 1 {
				t.Fatalf("진단 %d개, want 1개: %v", len(imageDiags), imageDiags)
			}
			col := imageCol(attrs, tc.needle)
			if strings.HasPrefix(tc.needle, `"`) {
				col++
			}
			assertImageDiag(t, imageDiags[0], col, tc.contains...)
			// 도형에서 같은 표기가 낸 진단과 **문구가 같다** (radius는 프리셋 제약이 있어
			// rounded-rect로 비교한다).
			shapeAttrs := `type="rounded-rect" ` + tc.decl
			_, shapeDiags := resolveSlide(t, shapeSlide(shapeAttrs))
			if len(shapeDiags) != 1 {
				t.Fatalf("도형 진단 %d개, want 1개: %v", len(shapeDiags), shapeDiags)
			}
			if imageDiags[0].Message != shapeDiags[0].Message {
				t.Errorf("진단 문구가 갈라졌다:\n이미지 %q\n도형   %q",
					imageDiags[0].Message, shapeDiags[0].Message)
			}
		})
	}
}

// `radius`는 `<image>`에서 **항상 유효하다** — ADR 0019(rounded-rect 전용)의 근거는 프리셋
// 기하에 조정값 자리가 없다는 것이었고, 그림은 프리셋이 아니라 자기 클립을 갖는다.
func TestResolveImageRadiusHasNoPresetConstraint(t *testing.T) {
	style := resolvedImage(t, `src="assets/wide.png" radius="lg"`)
	if style.Radius == nil || *style.Radius != soml.TokenRadius("lg", 24) {
		t.Fatalf("radius = %+v, want 토큰 lg", style.Radius)
	}
	// 같은 표기가 도형에서는 프리셋을 요구한다 — 두 규칙이 다른 이유가 여기 있다.
	if _, diags := resolveSlide(t, shapeSlide(`type="rect" radius="lg"`)); len(diags) != 1 {
		t.Errorf("도형의 radius 제약이 사라졌다: %v", diags)
	}
}

// 모르는 variant는 **경고 + 빈 패치**다 (PRD §13-6, 텍스트·도형과 같은 정책이고 문구도 같은
// 곳에서 나온다) — 그리고 나열하는 어휘가 **이미지의 것**이어야 한다: 도형 목록을 나열하면
// 저자를 그 테마에 없는 방향으로 보낸다.
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
	for _, want := range []string{"테마에 없다", "기본 외관으로 계속한다", "framed"} {
		if !strings.Contains(diags[0].Message, want) {
			t.Errorf("진단 = %q, want %q 포함", diags[0].String(), want)
		}
	}
	// 도형 어휘가 섞이지 않는다 (어휘 표가 둘이라는 계약).
	for _, name := range shapeVariantNames(minimalTheme()) {
		if strings.Contains(diags[0].Message, name) {
			t.Errorf("진단이 도형 variant %q를 나열했다: %s", name, diags[0].String())
		}
	}
	// 경고여도 해석은 계속된다 — 빈 패치이므로 컴포넌트 기본값(장식 없음)이 남는다.
	style := findElement(t, deck, soml.ImageElement).ImageStyle()
	if want := componentImageStyle(); style == nil ||
		!reflect.DeepEqual(soml.ImageStyle{Src: style.Src, Fit: style.Fit,
			Border: style.Border, Radius: style.Radius, Shadow: style.Shadow},
			soml.ImageStyle{Src: "assets/wide.png", Fit: want.Fit,
				Border: want.Border, Radius: want.Radius, Shadow: want.Shadow}) {
		t.Errorf("모르는 variant가 외관을 바꿨다: %s", showImage(*style))
	}
}

// 외관 속성이 **없는** 그림은 아무 장식도 갖지 않는다 — 컴포넌트 기본값을 비운 결정이다
// (도형과 갈라지는 자리: 채움 없는 도형은 보이지 않지만 그림은 자기 픽셀을 갖는다).
func TestResolveImageComponentDefaultIsEmpty(t *testing.T) {
	style := resolvedImage(t, `src="assets/wide.png"`)
	want := soml.ImageStyle{Src: "assets/wide.png", Fit: soml.DefaultImageFit}
	if !reflect.DeepEqual(*style, want) {
		t.Errorf("이미지 외관 = %s, want %s (장식 없음)", showImage(*style), showImage(want))
	}
	// 테마 단이 실제로 빈 패치인지 — 값이 붙으면 위의 단정과 이 단정이 함께 빨갛게 된다.
	if patch := imageComponentDefault(); patch != (imagePatch{}) {
		t.Errorf("컴포넌트 기본값 = %+v, want 빈 패치", patch)
	}
}

// 한 요소가 이미지 외관 + 자기 박스 + 자식 배치를 함께 쓴다 — 대입이 아니라 필드 채우기라는
// 구조(elemAppearance)가 이미지에서도 유지되는지.
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

// 단마다 값이 어디서 왔는지 구별되는 표 — 해석을 통과한 뒤(확정 값)에도 4단의 순서가
// **도형과 같은지**. 캐스케이드가 채널마다 다른 순서를 갖기 시작하면 저자가 외울 규칙이 둘이 된다.
func TestResolveImageVariantStage(t *testing.T) {
	base := componentImageStyle()
	framed := resolvedImage(t, `src="assets/wide.png" variant="framed"`)
	if framed.Border == nil || framed.Radius == nil || framed.Shadow == nil {
		t.Fatalf("variant가 액자를 정하지 않았다: %s", showImage(*framed))
	}
	// 컴포넌트 기본값이 비어 있으므로 variant의 세 필드 전부가 델타다.
	if base.Border != nil || base.Radius != nil || base.Shadow != nil {
		t.Errorf("컴포넌트 기본값이 장식을 가졌다: %s", showImage(base))
	}

	// 인라인이 variant를 이기고, **정하지 않은 필드는 variant가 남긴다** (부분성).
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
	// 틀린 인라인 값은 실리지 않고 아래 단(variant)이 남는다 — 도형과 같은 정책이다.
	deck, diags := resolveSlide(t, imageSlide(`src="assets/wide.png" variant="framed" border="0 solid muted"`))
	if len(diags) != 1 {
		t.Fatalf("진단 %d개, want 1개: %v", len(diags), diags)
	}
	style := findElement(t, deck, soml.ImageElement).ImageStyle()
	if style == nil || !reflect.DeepEqual(style.Border, framed.Border) {
		t.Errorf("border = %+v, want variant 단의 값", style)
	}
}

// assertImageDiag는 진단의 심각도·위치·문구를 단정한다 (assertShapeDiag의 이미지 판).
func assertImageDiag(t *testing.T, diag soml.Diagnostic, col int, contains ...string) {
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

// showImage는 실패 메시지용 문자열이다 — 포인터 필드가 많아 %+v로는 주소만 보인다
// (showShape와 같은 이유).
func showImage(s soml.ImageStyle) string {
	var b strings.Builder
	fmt.Fprintf(&b, "{src:%s fit:%s", s.Src, s.Fit)
	if s.Alt != "" {
		b.WriteString(" alt:" + s.Alt)
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
		fmt.Fprintf(&b, " shadow:%s", s.Shadow.Name)
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
	fmt.Fprintf(&b, " box:%s}", showBox(s.Box))
	return b.String()
}
