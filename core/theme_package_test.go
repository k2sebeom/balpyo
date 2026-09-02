package core

import (
	"reflect"
	"testing"

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
