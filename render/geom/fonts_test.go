package geom

import (
	"testing"

	"github.com/chromedp/cdproto/css"
	"github.com/k2sebeom/balpyo/soml"
)

// 브라우저가 그린 면이 요청한 가족인지: 가족 이름이 그대로 오거나, 지역화된 가족 이름 대신
// ASCII PostScript 이름으로만 알아볼 수 있는 경우가 있다. 글리프를 하나도 그리지 않은 면은
// 쓰인 것이 아니다.
func TestUsesFamily(t *testing.T) {
	use := func(family, ps string, glyphs float64) []*css.PlatformFontUsage {
		return []*css.PlatformFontUsage{{FamilyName: family, PostScriptName: ps, GlyphCount: glyphs}}
	}
	for _, tc := range []struct {
		name   string
		fonts  []*css.PlatformFontUsage
		family string
		want   bool
	}{
		{"가족 이름", use("Noto Sans CJK KR", "NotoSansCJKkr-Regular", 6), "Noto Sans CJK KR", true},
		{"대소문자", use("noto sans", "", 6), "Noto Sans", true},
		{"지역화된 가족 · PostScript로 맞다", use("맑은 고딕", "MalgunGothic", 6), "Malgun Gothic", true},
		{"굵기 접미", use("맑은 고딕", "MalgunGothic-Bold", 6), "Malgun Gothic", true},
		{"대체 글꼴", use("DejaVu Sans", "DejaVuSans", 6), "Malgun Gothic", false},
		{"그리지 않은 면", use("Malgun Gothic", "MalgunGothic", 0), "Malgun Gothic", false},
		{"빈 목록", nil, "Noto Sans", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := usesFamily(tc.fonts, tc.family); got != tc.want {
				t.Errorf("usesFamily = %v, want %v", got, tc.want)
			}
		})
	}
}

// 이름은 CSS 문자열로 감싼다 — 감싸지 않으면 `serif`가 제네릭 키워드가 된다.
func TestQuoteFamily(t *testing.T) {
	for in, want := range map[string]string{
		"Noto Sans": `"Noto Sans"`,
		"serif":     `"serif"`,
		`a"b\c`:     `"a\"b\\c"`,
	} {
		if got := quoteFamily(in); got != want {
			t.Errorf("quoteFamily(%q) = %s, want %s", in, got, want)
		}
	}
}

// 테마의 글꼴 토큰은 latin과 ea를 각자 표본 글자로 묻는다 (토큰 순서).
func TestThemeFontProbes(t *testing.T) {
	theme := soml.Theme{Fonts: []soml.FontToken{
		{Name: soml.FontHeading, Latin: "Arial", EastAsian: "Malgun Gothic"},
		{Name: soml.FontMono, Latin: "Consolas"},
	}}
	got := ThemeFontProbes(theme)
	want := []FontProbe{
		{Family: "Arial", Sample: LatinSample},
		{Family: "Malgun Gothic", Sample: EastAsianSample},
		{Family: "Consolas", Sample: LatinSample},
	}
	if len(got) != len(want) {
		t.Fatalf("probes = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("probes[%d] = %v, want %v", i, got[i], want[i])
		}
	}
}

// 없는 글꼴 하나가 경고 하나다. 그 이름을 적은 토큰 · 슬롯을 모두 말한다 — 테마를 고칠 사람이
// 찾을 자리다. 위치는 없다: 원인은 슬라이드가 아니라 테마와 이 기계의 조합이다.
func TestFontDiagnostics(t *testing.T) {
	theme := soml.Theme{Name: "aws", Fonts: []soml.FontToken{
		{Name: soml.FontHeading, Latin: "Arial", EastAsian: "Malgun Gothic"},
		{Name: soml.FontBody, Latin: "Arial", EastAsian: "Malgun Gothic"},
	}}
	diags := FontDiagnostics(theme, []string{"Malgun Gothic"})
	if len(diags) != 1 {
		t.Fatalf("경고 한 건이어야 한다: %v", diags)
	}
	want := `warning: theme "aws" font "Malgun Gothic" (heading.ea, body.ea) is missing from the measuring browser — ` +
		`the layout was measured with a fallback font, so text boxes will be off when PowerPoint draws with this font. ` +
		`Install the font on the build machine, or change the theme's font token to an installed font`
	if got := diags[0].String(); got != want {
		t.Errorf("진단\n got: %s\nwant: %s", got, want)
	}
	if FontDiagnostics(theme, nil) != nil {
		t.Error("없는 글꼴이 없으면 경고도 없다")
	}
}
