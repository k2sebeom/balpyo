package core

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"path"
	"strconv"
	"strings"

	"github.com/k2sebeom/balpyo/soml"
)

// 이 파일은 **테마 패키지 포맷**(PRD §8.2)의 로더다 — Iteration 10이 core/theme.go의 하드코딩
// 표를 데이터로 해체한 자리다. 순서를 이렇게 잡은 이유는 theme.go의 패키지 주석에 있다:
// 하드코딩 표가 두 렌더러의 실제 요구를 드러낸 다음에야 스키마를 파일로 고정한다.
//
// 포맷은 **데이터 전용**이다 (JSON + CSS + 에셋, JS 없음 — PRD §8.2·§13-11): `theme get`으로
// 받은 원격 테마가 임의 코드 실행 경로가 되어서는 안 된다. 로더는 fs.FS 위에서 도므로 내장
// 테마(go:embed)와 프로젝트 로컬 테마(os.DirFS)가 같은 코드를 지난다.
//
// **byte 동일성 계약**: 내장 minimal을 이 로더로 읽은 soml.Theme는 옛 하드코딩
// minimalTheme()과 값·순서가 정확히 같아야 한다. 그래야 기존 골든(themeCSS·clrScheme·픽스처)이
// 움직이지 않고, 해체가 동작을 바꾸지 않았음이 증명된다. 순서를 지키려고 tokens.json은 객체가
// 아니라 **순서 있는 배열**을 쓴다 (JSON 객체 키 순서에 기대지 않는다 — PRD §8.4의 객체
// 스케치와 갈라지는 자리이고 ADR이 근거를 남긴다).

// builtinThemes는 바이너리에 내장된 공식 테마다 (PRD §8.6). v1은 minimal 하나이며 기본값이자
// 폴백 기준이다 — `balpyo init` 직후 네트워크·설치 없이 예쁜 결과가 나오는 것이 첫인상을
// 지배하므로 기본 경로에서 네트워크를 제거한다.
//
//go:embed themes
var builtinThemes embed.FS

// themeManifest는 theme.json의 파싱 결과다 (PRD §8.2).
type themeManifest struct {
	Name string `json:"name"`
	// Extends는 상속할 테마 이름이다 (PRD §8.5) — v1 minimal에서는 비어 있다. 상속 병합은
	// 아직 구현하지 않으므로 비지 않은 값은 로더가 진단한다 (미구현을 조용히 무시하지 않는다).
	Extends string `json:"extends"`
	// Layouts는 이 테마가 구현하는 표준 레이아웃 이름 목록이다. 표준 카탈로그 중 여기 없는
	// 이름은 minimal 구현으로 폴백하고 경고한다 (core.slideLayout, 10/03). 테마 자체
	// 레이아웃(`corp-hero`)도 여기 실려 어휘가 된다.
	Layouts []string `json:"layouts"`
	// Chrome은 전 슬라이드 장식(로고·푸터) 선언이다 (PRD §8.1·§8.4) — v1 minimal은 비어 있다.
	Chrome []chromeDecl `json:"chrome"`
}

// chromeDecl은 크롬 요소 하나의 선언이다 (10/04·10/05가 소비한다). v1 minimal은 크롬이 없어
// 필드가 최소이며, 크롬 어휘가 실제로 필요한 iteration이 채운다.
type chromeDecl struct {
	Kind string `json:"kind"`
}

// themePackage는 로드된 테마 하나다 — 토큰 표(soml.Theme)와 매니페스트를 함께 담는다.
// 컴포넌트 외관(components.json)과 레이아웃 CSS(layouts/*.css)는 뒤 하위 단계가 이 구조체에
// 필드를 더한다.
type themePackage struct {
	manifest themeManifest
	tokens   soml.Theme
	// components는 로드·변환된 컴포넌트 외관 표다 (components.json). tokens와 달리 값 문법을
	// 캐스케이드 내부 타입으로 이미 눕혀 담는다 — resolveText/Shape/Image가 재변환 없이 읽는다.
	components components
}

// ── tokens.json 스키마 ──────────────────────────────────────────────────────
// 필드는 소문자 JSON 키에 태그로 붙는다. 배열이라 순서가 파일 그대로 유지된다.

type tokensFile struct {
	Color  []colorTok  `json:"color"`
	Font   []fontTok   `json:"font"`
	Space  []spaceTok  `json:"space"`
	Radius []radiusTok `json:"radius"`
	Shadow []shadowTok `json:"shadow"`
}

type colorTok struct {
	Name string `json:"name"`
	RGB  string `json:"rgb"`
	Slot string `json:"slot"`
	Ref  string `json:"ref"`
}

type fontTok struct {
	Name    string `json:"name"`
	Latin   string `json:"latin"`
	EA      string `json:"ea"`
	Generic string `json:"generic"`
	Scheme  string `json:"scheme"`
}

type spaceTok struct {
	Name string  `json:"name"`
	Px   float64 `json:"px"`
}

type radiusTok struct {
	Name string  `json:"name"`
	Px   float64 `json:"px"`
}

type shadowTok struct {
	Name  string  `json:"name"`
	Dist  float64 `json:"dist"`
	Dir   float64 `json:"dir"`
	Blur  float64 `json:"blur"`
	Color string  `json:"color"`
	Alpha float64 `json:"alpha"`
}

// ── components.json 스키마 (원본 JSON) ──────────────────────────────────────
// 배열이라 순서가 파일 그대로 유지된다 (tokens.json과 같은 이유). optional 필드는 포인터라
// "정하지 않음"(nil)과 0값을 구별한다 — 부분 패치가 아래 단을 지우지 않게 한다.

type componentsFile struct {
	// Comment는 문서용 $comment 키를 허용하기 위한 자리다 — DisallowUnknownFields가 그 외
	// 모든 미지원 키(예: extends·오타)를 진단하되 문서 주석은 통과시킨다 (닫힌 어휘).
	Comment string              `json:"$comment"`
	Text    textComponentsFile  `json:"text"`
	Shape   shapeComponentsFile `json:"shape"`
	Image   imageComponentsFile `json:"image"`
}

type textComponentsFile struct {
	Roles             []textRoleFile             `json:"roles"`
	ComponentDefaults []textComponentDefaultFile `json:"componentDefaults"`
	Variants          []textVariantFile          `json:"variants"`
}

type textRoleFile struct {
	Name       string  `json:"name"`
	Font       string  `json:"font"`
	Size       float64 `json:"size"`
	Weight     int     `json:"weight"`
	LineHeight float64 `json:"lineHeight"`
	Color      string  `json:"color"`
	Align      string  `json:"align"`
}

type textComponentDefaultFile struct {
	Tag   string   `json:"tag"`
	Size  *float64 `json:"size"`
	Color *string  `json:"color"`
}

type textVariantFile struct {
	Name   string   `json:"name"`
	Size   *float64 `json:"size"`
	Weight *int     `json:"weight"`
	Color  *string  `json:"color"`
}

type shapeComponentsFile struct {
	Default  shapeStyleFile     `json:"default"`
	Variants []shapeVariantFile `json:"variants"`
}

type shapeStyleFile struct {
	Fill   string `json:"fill"`
	Border string `json:"border"`
	Radius string `json:"radius"`
}

type shapeVariantFile struct {
	Name   string  `json:"name"`
	Fill   *string `json:"fill"`
	Border *string `json:"border"`
	Radius *string `json:"radius"`
	Shadow *string `json:"shadow"`
}

type imageComponentsFile struct {
	Default  imageStyleFile     `json:"default"`
	Variants []imageVariantFile `json:"variants"`
}

type imageStyleFile struct{}

type imageVariantFile struct {
	Name   string  `json:"name"`
	Border *string `json:"border"`
	Radius *string `json:"radius"`
	Shadow *string `json:"shadow"`
}

// ── 변환된 컴포넌트 표 (캐스케이드 내부 타입) ───────────────────────────────
// 캐스케이드가 이미 쓰는 타입 그대로 담는다 (theme.go의 옛 함수 반환형과 같다) — 그래야
// resolveText/Shape/Image가 재변환 없이 읽고, byte 동일성 테스트가 옛 함수와 직접 비교한다.

type components struct {
	textRoles             map[soml.TextRole]soml.TextStyle
	textComponentDefaults map[string]textPatch
	textVariants          []textVariant
	shapeDefault          shapePatch
	shapeVariants         []shapeVariant
	imageDefault          imagePatch
	imageVariants         []imageVariant
}

// loadTheme는 fsys의 dir/ 아래 테마 패키지를 읽는다. dir은 fs 경로다 (내장은
// "themes/minimal", 로컬은 프로젝트 상대 경로). 이름은 매니페스트가 소유한다.
func loadTheme(fsys fs.FS, dir string) (themePackage, error) {
	var pkg themePackage

	manifestBytes, err := fs.ReadFile(fsys, path.Join(dir, "theme.json"))
	if err != nil {
		return pkg, fmt.Errorf("theme.json을 읽을 수 없다: %w", err)
	}
	if err := json.Unmarshal(manifestBytes, &pkg.manifest); err != nil {
		return pkg, fmt.Errorf("theme.json 파싱 실패: %w", err)
	}
	if pkg.manifest.Name == "" {
		return pkg, fmt.Errorf("theme.json에 name이 없다")
	}
	if pkg.manifest.Extends != "" {
		// 상속 병합은 아직 구현하지 않았다 — 조용히 무시하면 저자가 오버라이드가 먹었다고
		// 착각한다 (닫힌 어휘가 막으려던 조용한 실패의 변종).
		return pkg, fmt.Errorf("theme.json의 extends는 아직 지원하지 않는다 (%q)", pkg.manifest.Extends)
	}

	tokens, err := loadTokens(fsys, dir, pkg.manifest.Name)
	if err != nil {
		return pkg, err
	}
	pkg.tokens = tokens

	comps, err := loadComponents(fsys, dir, tokens)
	if err != nil {
		return pkg, err
	}
	pkg.components = comps
	return pkg, nil
}

// loadTokens는 tokens.json을 soml.Theme로 읽는다. 배열 순서를 그대로 옮겨 byte 동일성을
// 지킨다.
func loadTokens(fsys fs.FS, dir, name string) (soml.Theme, error) {
	var theme soml.Theme
	raw, err := fs.ReadFile(fsys, path.Join(dir, "tokens.json"))
	if err != nil {
		return theme, fmt.Errorf("tokens.json을 읽을 수 없다: %w", err)
	}
	var file tokensFile
	if err := json.Unmarshal(raw, &file); err != nil {
		return theme, fmt.Errorf("tokens.json 파싱 실패: %w", err)
	}

	theme.Name = name
	for _, c := range file.Color {
		theme.Colors = append(theme.Colors, soml.ColorToken{Name: c.Name, RGB: c.RGB, Slot: c.Slot, Ref: c.Ref})
	}
	for _, f := range file.Font {
		theme.Fonts = append(theme.Fonts, soml.FontToken{
			Name: soml.FontRole(f.Name), Latin: f.Latin, EastAsian: f.EA, Generic: f.Generic, Scheme: f.Scheme,
		})
	}
	for _, s := range file.Space {
		theme.Spaces = append(theme.Spaces, soml.SpaceToken{Name: s.Name, Px: s.Px})
	}
	for _, r := range file.Radius {
		theme.Radii = append(theme.Radii, soml.RadiusToken{Name: r.Name, Px: r.Px})
	}
	for _, s := range file.Shadow {
		// color는 토큰 참조다 (표가 리터럴을 쓰면 PPTX에서 schemeClr 참조가 끊긴다 —
		// theme.go의 옛 minimalShadows 근거). soml.TokenColor로 감싼다.
		theme.Shadows = append(theme.Shadows, soml.ShadowToken{
			Name: s.Name, Dist: s.Dist, Dir: s.Dir, Blur: s.Blur,
			Color: soml.TokenColor(s.Color), Alpha: s.Alpha,
		})
	}
	return theme, nil
}

// loadComponents는 components.json을 캐스케이드 내부 표로 읽는다. theme는 radius·shadow 토큰을
// 해석하려고 받는다 (loadTokens가 먼저 채운 표). 미지원 키는 DisallowUnknownFields가 진단한다
// — 조용히 무시하면 저자가 오버라이드가 먹었다고 착각한다 (닫힌 어휘, manifest.Extends와 같은 태도).
func loadComponents(fsys fs.FS, dir string, theme soml.Theme) (components, error) {
	var comps components
	raw, err := fs.ReadFile(fsys, path.Join(dir, "components.json"))
	if err != nil {
		return comps, fmt.Errorf("components.json을 읽을 수 없다: %w", err)
	}
	var file componentsFile
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&file); err != nil {
		return comps, fmt.Errorf("components.json 파싱 실패: %w", err)
	}

	comps.textRoles = map[soml.TextRole]soml.TextStyle{}
	for _, r := range file.Text.Roles {
		comps.textRoles[soml.TextRole(r.Name)] = soml.TextStyle{
			Role: soml.TextRole(r.Name), Font: soml.FontRole(r.Font), Size: r.Size,
			Weight: r.Weight, LineHeight: r.LineHeight,
			Color: soml.TokenColor(r.Color), Align: soml.TextAlign(r.Align),
		}
	}

	comps.textComponentDefaults = map[string]textPatch{}
	for _, d := range file.Text.ComponentDefaults {
		var patch textPatch
		if d.Size != nil {
			patch.size = d.Size
		}
		if d.Color != nil {
			c := soml.TokenColor(*d.Color)
			patch.color = &c
		}
		comps.textComponentDefaults[d.Tag] = patch
	}

	for _, v := range file.Text.Variants {
		var patch textPatch
		if v.Size != nil {
			patch.size = v.Size
		}
		if v.Weight != nil {
			patch.weight = v.Weight
		}
		if v.Color != nil {
			c := soml.TokenColor(*v.Color)
			patch.color = &c
		}
		comps.textVariants = append(comps.textVariants, textVariant{name: v.Name, patch: patch})
	}

	shapeDefault, err := shapePatchFromFile(theme, file.Shape.Default.Fill,
		file.Shape.Default.Border, file.Shape.Default.Radius, "")
	if err != nil {
		return comps, fmt.Errorf("shape.default: %w", err)
	}
	comps.shapeDefault = shapeDefault
	for _, v := range file.Shape.Variants {
		patch, err := shapeVariantPatchFromFile(theme, v)
		if err != nil {
			return comps, fmt.Errorf("shape variant %q: %w", v.Name, err)
		}
		comps.shapeVariants = append(comps.shapeVariants, shapeVariant{name: v.Name, patch: patch})
	}

	for _, v := range file.Image.Variants {
		patch, err := imageVariantPatchFromFile(theme, v)
		if err != nil {
			return comps, fmt.Errorf("image variant %q: %w", v.Name, err)
		}
		comps.imageVariants = append(comps.imageVariants, imageVariant{name: v.Name, patch: patch})
	}

	return comps, nil
}

// shapePatchFromFile은 shape default의 문자열 필드를 shapePatch로 눕힌다. 빈 문자열은 "정하지
// 않음"이다. fill/border 색·radius/shadow 토큰 해석은 theme.go의 헬퍼(colorFill·solidBorder·
// radiusToken·shadowToken)를 그대로 지나 byte 동일성을 지킨다.
func shapePatchFromFile(theme soml.Theme, fill, border, radius, shadow string) (shapePatch, error) {
	var patch shapePatch
	if fill != "" {
		f, err := fillFromString(fill)
		if err != nil {
			return patch, err
		}
		patch.fill = f
	}
	if border != "" {
		b, err := borderFromString(border)
		if err != nil {
			return patch, err
		}
		patch.border = b
	}
	if radius != "" {
		patch.radius = radiusToken(theme, radius)
	}
	if shadow != "" {
		patch.shadow = shadowToken(theme, shadow)
	}
	return patch, nil
}

func shapeVariantPatchFromFile(theme soml.Theme, v shapeVariantFile) (shapePatch, error) {
	fill, border, radius, shadow := "", "", "", ""
	if v.Fill != nil {
		fill = *v.Fill
	}
	if v.Border != nil {
		border = *v.Border
	}
	if v.Radius != nil {
		radius = *v.Radius
	}
	if v.Shadow != nil {
		shadow = *v.Shadow
	}
	return shapePatchFromFile(theme, fill, border, radius, shadow)
}

func imageVariantPatchFromFile(theme soml.Theme, v imageVariantFile) (imagePatch, error) {
	var patch imagePatch
	if v.Border != nil {
		b, err := borderFromString(*v.Border)
		if err != nil {
			return patch, err
		}
		patch.border = b
	}
	if v.Radius != nil {
		patch.radius = radiusToken(theme, *v.Radius)
	}
	if v.Shadow != nil {
		patch.shadow = shadowToken(theme, *v.Shadow)
	}
	return patch, nil
}

// fillFromString은 `none` 또는 색 토큰명을 *soml.Fill로 눕힌다 (theme.go의 colorFill/NoFill과
// 같은 결과). 테마 데이터는 토큰만 쓴다 (리터럴 금지 — theme.go의 근거).
func fillFromString(s string) (*soml.Fill, error) {
	if s == noneValue {
		return ptr(soml.NoFill()), nil
	}
	return colorFill(s), nil
}

// borderFromString은 `none` 또는 `<굵기> <스타일> <색>` 축약을 *soml.Border로 눕힌다 —
// resolver.border의 값 문법과 같고, `1 solid muted`는 theme.go의 solidBorder(1,"muted")와 같다.
func borderFromString(s string) (*soml.Border, error) {
	if s == noneValue {
		return ptr(soml.NoBorder()), nil
	}
	fields := strings.Fields(s)
	if len(fields) != borderParts {
		return nil, fmt.Errorf("border는 %q 또는 `<굵기> <스타일> <색>` 3조각이다 (%q)", noneValue, s)
	}
	width, err := strconv.ParseFloat(fields[0], 64)
	if err != nil {
		return nil, fmt.Errorf("border 굵기가 수치가 아니다 (%q)", fields[0])
	}
	style := soml.BorderStyle(fields[1])
	ok := false
	for _, s := range soml.BorderStyles() {
		if s == style {
			ok = true
		}
	}
	if !ok {
		return nil, fmt.Errorf("알 수 없는 border 스타일 %q", fields[1])
	}
	return &soml.Border{Width: width, Style: style, Color: soml.TokenColor(fields[2])}, nil
}

// loadBuiltinTheme는 내장 테마를 이름으로 읽는다. 내장 파일은 빌드 시점에 고정되므로 로드
// 실패는 프로그래머 오류다 (regexp.MustCompile과 같은 자리) — 호출자(minimalTheme)가 에러를
// 다룰 자리가 없고, TestLoadBuiltinMinimal이 이 불변을 못 박는다.
func loadBuiltinTheme(name string) themePackage {
	pkg, err := loadTheme(builtinThemes, path.Join("themes", name))
	if err != nil {
		panic(fmt.Sprintf("내장 테마 %q 로드 실패: %v", name, err))
	}
	return pkg
}
