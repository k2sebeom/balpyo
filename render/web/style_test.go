package web

import (
	"strings"
	"testing"

	"github.com/k2sebeom/balpyo/soml"
)

// testTheme는 단위 테스트용 토큰 표다. core의 최소 테마와 값이 같을 필요는 없다 — 렌더러는
// 표를 해석하지 않고 CSS로 옮기기만 하므로, 값이 아니라 **표에서 나왔다는 사실**이 계약이다.
func testTheme() soml.Theme {
	return soml.Theme{
		Name: "test",
		Colors: []soml.ColorToken{
			{Name: "fg", RGB: "1A1A1A", Slot: "dk1", Ref: "tx1"},
			{Name: "bg", RGB: "FFFFFF", Slot: "lt1", Ref: "bg1"},
			{Name: "muted", RGB: "6B7280", Slot: "dk2", Ref: "tx2"},
			{Name: "accent1", RGB: "0F65F9", Slot: "accent1", Ref: "accent1"},
		},
		Fonts: []soml.FontToken{
			{Name: soml.FontHeading, Latin: "Noto Sans", EastAsian: "Noto Sans CJK KR", Generic: "sans-serif", Scheme: "major"},
			{Name: soml.FontBody, Latin: "Noto Sans", EastAsian: "Noto Sans CJK KR", Generic: "sans-serif", Scheme: "minor"},
			{Name: soml.FontMono, Latin: "Noto Sans Mono", Generic: "monospace"},
		},
		// 공간 토큰은 최소 테마의 사다리와 같다 (core.minimalSpaces) — 레이아웃 태그의
		// gap/pad가 이 표를 var() 참조로 가리킨다.
		Spaces: []soml.SpaceToken{
			{Name: "xs", Px: 6}, {Name: "sm", Px: 12}, {Name: "md", Px: 24},
			{Name: "lg", Px: 36}, {Name: "xl", Px: 48}, {Name: "2xl", Px: 72},
		},
	}
}

// 토큰은 **표 순서대로**(맵 순회가 아니다 — 결정성) 커스텀 프로퍼티로 선언되고, 슬라이드
// 배경도 bg 토큰을 참조한다. 테마가 없으면 테마 층도 없다.
func TestThemeCSSDeclaresTokensAsCustomProperties(t *testing.T) {
	css := themeCSS(testTheme(), features{})
	at := 0
	for _, want := range []string{
		"@layer balpyo.theme {",
		"--balpyo-color-fg: #1A1A1A;",
		"--balpyo-color-bg: #FFFFFF;",
		"--balpyo-color-muted: #6B7280;",
		"--balpyo-color-accent1: #0F65F9;",
		`--balpyo-font-heading: "Noto Sans", "Noto Sans CJK KR", sans-serif;`,
		`--balpyo-font-body: "Noto Sans", "Noto Sans CJK KR", sans-serif;`,
		`--balpyo-font-mono: "Noto Sans Mono", monospace;`,
		"background: var(--balpyo-color-bg);",
	} {
		found := strings.Index(css[at:], want)
		if found < 0 {
			t.Fatalf("테마 CSS에 %q가 없거나 순서가 다르다:\n%s", want, css)
		}
		at += found + len(want)
	}
	if css := themeCSS(soml.Theme{}, features{layout: true}); css != "" {
		t.Errorf("빈 테마의 CSS = %q, want 빈 문자열", css)
	}
}

// 토큰 색·폰트는 var() 참조로만 나간다 — 인라인 16진수는 테마 색 편집을 따라가지 못한다.
func TestTextStyleCSS(t *testing.T) {
	tests := []struct {
		name  string
		style soml.TextStyle
		want  string
	}{
		{
			name: "역할 기본값 (title)",
			style: soml.TextStyle{Role: soml.RoleTitle, Font: soml.FontHeading, Size: 56, Weight: 700,
				LineHeight: 1.2, Color: soml.TokenColor("fg"), Align: soml.AlignStart},
			want: "font-family: var(--balpyo-font-heading); font-size: 56px; font-weight: 700; " +
				"line-height: 1.2; color: var(--balpyo-color-fg); text-align: start",
		},
		{
			name: "리터럴 색은 16진수로 나간다",
			style: soml.TextStyle{Role: soml.RoleBody, Font: soml.FontBody, Size: 24, Weight: 400,
				LineHeight: 1.5, Color: soml.LiteralColor("FF00AA"), Align: soml.AlignJustify},
			want: "font-family: var(--balpyo-font-body); font-size: 24px; font-weight: 400; " +
				"line-height: 1.5; color: #FF00AA; text-align: justify",
		},
		{
			name: "소수 크기는 자릿수를 늘리지 않는다",
			style: soml.TextStyle{Role: soml.RoleBody, Font: soml.FontBody, Size: 20.5, Weight: 400,
				LineHeight: 1.45, Color: soml.TokenColor("fg"), Align: soml.AlignStart},
			want: "font-family: var(--balpyo-font-body); font-size: 20.5px; font-weight: 400; " +
				"line-height: 1.45; color: var(--balpyo-color-fg); text-align: start",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := textStyleCSS(tt.style, testTheme())
			if err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Errorf("textStyleCSS()\n got: %s\nwant: %s", got, tt.want)
			}
		})
	}
}

func TestTextStyleCSSRejectsBrokenStyle(t *testing.T) {
	base := soml.TextStyle{Role: soml.RoleBody, Font: soml.FontBody, Size: 24, Weight: 400,
		LineHeight: 1.5, Color: soml.TokenColor("fg"), Align: soml.AlignStart}
	mutate := func(f func(*soml.TextStyle)) soml.TextStyle {
		style := base
		f(&style)
		return style
	}
	tests := []struct {
		name  string
		style soml.TextStyle
	}{
		{"테마에 없는 색 토큰", mutate(func(s *soml.TextStyle) { s.Color = soml.TokenColor("surface") })},
		{"테마에 없는 폰트 토큰", mutate(func(s *soml.TextStyle) { s.Font = "display" })},
		{"해석되지 않은 색", mutate(func(s *soml.TextStyle) { s.Color = soml.Color{} })},
		{"어휘 밖 정렬", mutate(func(s *soml.TextStyle) { s.Align = "left" })},
		{"0 크기", mutate(func(s *soml.TextStyle) { s.Size = 0 })},
		{"0 굵기", mutate(func(s *soml.TextStyle) { s.Weight = 0 })},
		{"0 줄높이", mutate(func(s *soml.TextStyle) { s.LineHeight = 0 })},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got, err := textStyleCSS(tt.style, testTheme()); err == nil {
				t.Errorf("에러여야 한다 (got %q)", got)
			}
		})
	}
}

// 공백이 든 패밀리 이름은 인용하고 CSS 제네릭 키워드는 맨몸으로 둔다 — 제네릭을 인용하면
// 패밀리 이름으로 해석되어 마지막 폴백이 사라진다.
func TestFontStackQuoting(t *testing.T) {
	tests := []struct {
		token soml.FontToken
		want  string
	}{
		{soml.FontToken{Latin: "Noto Sans", EastAsian: "Noto Sans CJK KR", Generic: "sans-serif"},
			`"Noto Sans", "Noto Sans CJK KR", sans-serif`},
		{soml.FontToken{Latin: "Arial", Generic: "sans-serif"}, `Arial, sans-serif`},
		// 같은 이름이 두 자리에 오면 Family가 접는다 — 스택에도 한 번만 나온다.
		{soml.FontToken{Latin: "Noto Sans", EastAsian: "Noto Sans", Generic: "sans-serif"},
			`"Noto Sans", sans-serif`},
	}
	for _, tt := range tests {
		if got := fontStack(tt.token); got != tt.want {
			t.Errorf("fontStack(%+v) = %s, want %s", tt.token, got, tt.want)
		}
	}
}
