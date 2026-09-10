package core

import (
	"reflect"
	"testing"
	"testing/fstest"

	"github.com/k2sebeom/balpyo/soml"
)

// TestLoadBuiltinMinimal은 내장 minimal 패키지가 로드되고, 그 토큰 표가 옛 하드코딩
// minimalTheme()과 **값·순서까지** 같음을 못 박는다 — byte 동일성 계약(theme_package.go)의
// 앵커다. 값이 여기 그대로 적혀 있는 이유: tokens.json이 실수로 바뀌면(순서·수치) 이 테스트가
// 잡고, 기존 골든이 조용히 움직이는 것을 막는다.
func TestLoadBuiltinMinimal(t *testing.T) {
	pkg := loadBuiltinTheme("minimal")
	if pkg.manifest.Name != "minimal" {
		t.Fatalf("manifest.Name = %q, want minimal", pkg.manifest.Name)
	}
	// 매니페스트가 표준 레이아웃 11종을 전부 구현한다고 선언한다 (폴백 기준).
	wantLayouts := []string{"title", "section-divider", "content", "two-column",
		"image-full", "image-text", "quote", "code", "stat", "canvas", "end"}
	if !reflect.DeepEqual(pkg.manifest.Layouts, wantLayouts) {
		t.Errorf("manifest.Layouts = %v\nwant %v", pkg.manifest.Layouts, wantLayouts)
	}

	got := pkg.tokens
	want := soml.Theme{
		Name: "minimal",
		Colors: []soml.ColorToken{
			{Name: "fg", RGB: "1A1A1A", Slot: "dk1", Ref: "tx1"},
			{Name: "bg", RGB: "FFFFFF", Slot: "lt1", Ref: "bg1"},
			{Name: "muted", RGB: "6B7280", Slot: "dk2", Ref: "tx2"},
			{Name: "surface", RGB: "F3F4F6", Slot: "lt2", Ref: "bg2"},
			{Name: "accent1", RGB: "0F65F9", Slot: "accent1", Ref: "accent1"},
		},
		Fonts: []soml.FontToken{
			{Name: soml.FontHeading, Latin: "Noto Sans", EastAsian: "Noto Sans CJK KR", Generic: "sans-serif", Scheme: "major"},
			{Name: soml.FontBody, Latin: "Noto Sans", EastAsian: "Noto Sans CJK KR", Generic: "sans-serif", Scheme: "minor"},
			{Name: soml.FontMono, Latin: "Noto Sans Mono", EastAsian: "Noto Sans CJK KR", Generic: "monospace"},
		},
		Spaces: []soml.SpaceToken{
			{Name: "xs", Px: 6}, {Name: "sm", Px: 12}, {Name: "md", Px: 24},
			{Name: "lg", Px: 36}, {Name: "xl", Px: 48}, {Name: "2xl", Px: 72},
		},
		Radii: []soml.RadiusToken{
			{Name: "sm", Px: 6}, {Name: "md", Px: 12}, {Name: "lg", Px: 24},
		},
		Shadows: []soml.ShadowToken{
			{Name: "soft", Dist: 4, Dir: 90, Blur: 16, Color: soml.TokenColor("fg"), Alpha: 0.15},
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("로드된 minimal 토큰이 기대와 다르다:\ngot  %+v\nwant %+v", got, want)
	}
	// minimalTheme()도 같은 표를 준다 (resolve.go가 쓰는 경로).
	if !reflect.DeepEqual(minimalTheme(), want) {
		t.Errorf("minimalTheme()가 로드 결과와 다르다")
	}
}

// minimalFonts·minimalShadows는 옛 하드코딩 표를 대체한 **테스트 헬퍼**다 — 로드된 테마에서
// 뽑아 기대값을 만든다. 프로덕션 코드는 loadBuiltinTheme를 지나므로(theme.go의 minimalTheme)
// 이 헬퍼가 그 표를 다시 적지 않고 파생해, 헬퍼와 파일이 어긋날 수 없다.
func minimalFonts() []soml.FontToken     { return minimalTheme().Fonts }
func minimalShadows() []soml.ShadowToken { return minimalTheme().Shadows }

// TestLoadBuiltinComponents는 내장 minimal의 components.json이 로드되고, 그 표가 옛 하드코딩
// (theme.go)과 값·순서까지 같음을 못 박는다 — byte 동일성 계약의 컴포넌트 앵커다.
func TestLoadBuiltinComponents(t *testing.T) {
	pkg := loadBuiltinTheme("minimal")
	theme := pkg.tokens
	comps := pkg.components

	// 텍스트 역할 5종 — 각 역할이 옛 textRoleDefaults와 같다.
	for _, role := range soml.TextRoles() {
		got, ok := comps.textRoles[role]
		if !ok {
			t.Fatalf("역할 %q가 로드된 표에 없다", role)
		}
		if !reflect.DeepEqual(got, textRoleDefaults(role)) {
			t.Errorf("역할 %q: got %+v\nwant %+v", role, got, textRoleDefaults(role))
		}
	}
	if len(comps.textRoles) != 5 {
		t.Errorf("텍스트 역할 수 = %d, want 5", len(comps.textRoles))
	}

	// 컴포넌트 기본값 — h3·blockquote만 정하고, 나머지 태그는 표에 없어 빈 패치로 폴백한다.
	if !reflect.DeepEqual(comps.textComponentDefaults["h3"], textComponentDefault("h3")) {
		t.Errorf("h3 기본값이 다르다")
	}
	if !reflect.DeepEqual(comps.textComponentDefaults["blockquote"], textComponentDefault("blockquote")) {
		t.Errorf("blockquote 기본값이 다르다")
	}
	if len(comps.textComponentDefaults) != 2 {
		t.Errorf("컴포넌트 기본값 수 = %d, want 2", len(comps.textComponentDefaults))
	}

	// 텍스트 variant — 값과 **순서**가 옛 표와 같다.
	if !reflect.DeepEqual(comps.textVariants, textVariants()) {
		t.Errorf("텍스트 variant 표가 다르다:\ngot  %+v\nwant %+v", comps.textVariants, textVariants())
	}

	// 도형 기본값·variant — 값과 순서가 옛 표와 같다 (radius·shadow 토큰 해석 포함).
	if !reflect.DeepEqual(comps.shapeDefault, shapeComponentDefault(theme)) {
		t.Errorf("도형 기본값이 다르다:\ngot  %+v\nwant %+v", comps.shapeDefault, shapeComponentDefault(theme))
	}
	if !reflect.DeepEqual(comps.shapeVariants, shapeVariants(theme)) {
		t.Errorf("도형 variant 표가 다르다:\ngot  %+v\nwant %+v", comps.shapeVariants, shapeVariants(theme))
	}

	// 이미지 기본값(비어 있음)·variant.
	if !reflect.DeepEqual(comps.imageDefault, imageComponentDefault()) {
		t.Errorf("이미지 기본값이 다르다:\ngot  %+v\nwant %+v", comps.imageDefault, imageComponentDefault())
	}
	if !reflect.DeepEqual(comps.imageVariants, imageVariants(theme)) {
		t.Errorf("이미지 variant 표가 다르다:\ngot  %+v\nwant %+v", comps.imageVariants, imageVariants(theme))
	}
}

// TestLoadBuiltinComponentsPreservesVariantOrder는 variant 배열이 파일 순서 그대로임을 못 박는다
// — 모르는 variant 경고가 나열하는 목록이 결정적이어야 하므로 (JSON 객체 키 순서에 기대지 않는다).
func TestLoadBuiltinComponentsPreservesVariantOrder(t *testing.T) {
	comps := loadBuiltinTheme("minimal").components
	wantText := []string{"kicker", "lead"}
	for i, v := range comps.textVariants {
		if v.name != wantText[i] {
			t.Errorf("텍스트 variant[%d] = %q, want %q", i, v.name, wantText[i])
		}
	}
	wantShape := []string{"card", "card-header", "card-body", "card-backdrop", "stat-card", "badge"}
	if len(comps.shapeVariants) != len(wantShape) {
		t.Fatalf("도형 variant 수 = %d, want %d", len(comps.shapeVariants), len(wantShape))
	}
	for i, v := range comps.shapeVariants {
		if v.name != wantShape[i] {
			t.Errorf("도형 variant[%d] = %q, want %q", i, v.name, wantShape[i])
		}
	}
}

// TestLoadComponentsRejectsUnknownField는 미지원/오타 키를 조용히 무시하지 않고 진단함을 못 박는다
// (닫힌 어휘). extends는 아직 구현하지 않은 상속 키의 예다.
func TestLoadComponentsRejectsUnknownField(t *testing.T) {
	fsys := fstest.MapFS{
		"t/components.json": {Data: []byte(`{"extends":"base","text":{"roles":[]}}`)},
	}
	if _, err := loadComponents(fsys, "t", soml.Theme{}); err == nil {
		t.Fatal("미지원 필드 extends가 진단되지 않았다 (조용히 무시됨)")
	}
}
