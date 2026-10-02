package core

import (
	"math"
	"strconv"
	"testing"

	"github.com/k2sebeom/balpyo/soml"
)

// 강조색을 고르는 규칙을 수치로 못 박는다 — 값은 tokens.json이 소유하고, 여기는 "왜 그 값인가"다.
// 내장 테마 전수, 여섯 색 전수:
//
//  1. bg 대비 4.5:1 이상 (WCAG AA). 대비율은 대칭이라 "accent 글자 on bg"와 "bg 글자 on accent
//     면"을 함께 보장하고, 밝은 테마와 어두운 테마에서 같은 문장으로 성립한다.
//  2. 색상(hue)이 서로 25° 이상 떨어진다 — 옆에 놓았을 때 다른 색으로 보여야 위계가 된다.
//  3. 상대 휘도가 한 띠에 있다 (최대/최소 ≤ 1.25) — 한 색만 튀면 위계가 아니라 소음이다.
//
// 기준이 surface가 아니라 bg인 이유: 회색 면 위의 작은 글자는 저자가 accent를 쓸 자리가 아니다.
func TestBuiltinAccentPalettes(t *testing.T) {
	for _, name := range BuiltinThemes() {
		t.Run(name, func(t *testing.T) {
			checkAccentPalette(t, loadBuiltinTheme(name).tokens)
		})
	}
}

// checkAccentPalette는 테마 하나의 강조색 여섯에 세 규칙을 전수로 적용한다.
func checkAccentPalette(t *testing.T, theme soml.Theme) {
	t.Helper()
	accents := make([]soml.ColorToken, 0, 6)
	for _, token := range theme.Colors {
		if len(token.Name) == 7 && token.Name[:6] == "accent" {
			accents = append(accents, token)
		}
	}
	if len(accents) != 6 {
		t.Fatalf("accent 토큰 %d개, want 6 (%v)", len(accents), accents)
	}

	bg := colorTokenRGB(t, theme, "bg")
	for i, token := range accents {
		want := "accent" + strconv.Itoa(i+1)
		if token.Name != want {
			t.Errorf("accent %d번째 토큰 이름 = %q, want %q (표의 순서가 곧 슬롯 순서다)", i+1, token.Name, want)
		}
		// 강조색은 전부 자기 accent 슬롯을 갖는다 — 슬롯을 빌리면 PowerPoint의 테마 색
		// 편집이 그 색을 따라가지 못한다 (clrSchemeSlots의 폴백은 편집 UI를 위한 것이다).
		if token.Slot != want || token.Ref != want {
			t.Errorf("%s: slot=%q ref=%q, want 둘 다 %q", token.Name, token.Slot, token.Ref, want)
		}
		if got := contrastRatio(t, token.RGB, bg); got < 4.5 {
			t.Errorf("%s(%s)의 bg 대비 = %.2f:1, want ≥ 4.5:1", token.Name, token.RGB, got)
		}
	}

	// 색상 간격 — 모든 쌍을 본다 (원 위의 거리이므로 정렬만으로는 부족하다).
	for i := range accents {
		for j := i + 1; j < len(accents); j++ {
			a, b := accents[i], accents[j]
			if gap := hueGap(t, a.RGB, b.RGB); gap < 25 {
				t.Errorf("%s(%s)와 %s(%s)의 색상 간격 = %.0f°, want ≥ 25° — 옆에 놓으면 같은 색으로 보인다",
					a.Name, a.RGB, b.Name, b.RGB, gap)
			}
		}
	}

	// 밝기 띠.
	min, max := math.Inf(1), 0.0
	for _, token := range accents {
		lum := relativeLuminance(t, token.RGB)
		min, max = math.Min(min, lum), math.Max(max, lum)
	}
	if ratio := max / min; ratio > 1.25 {
		t.Errorf("accent 휘도 띠 최대/최소 = %.2f, want ≤ 1.25 — 한 색만 튀면 위계가 아니라 소음이다", ratio)
	}
}

// TestBuiltinBaseColorContrast는 바탕·글자·보조 글자의 대비를 전 테마에서 못 박는다 — 어두운
// 테마를 더하는 순간 필요해진 판정이다 (밝은 테마 하나였을 때는 fg가 검정이라 물어볼 것이
// 없었다). muted는 본문보다 작은 글자에 쓰이지만 기준을 4.5:1로 같이 두는 이유는 그 색이 캡션
// **역할의 색**이고, 캡션은 읽히기 위해 있는 글자라는 것이다.
func TestBuiltinBaseColorContrast(t *testing.T) {
	for _, name := range BuiltinThemes() {
		theme := loadBuiltinTheme(name).tokens
		bg := colorTokenRGB(t, theme, "bg")
		for _, token := range []string{"fg", "muted"} {
			if got := contrastRatio(t, colorTokenRGB(t, theme, token), bg); got < 4.5 {
				t.Errorf("%s의 %s 대비 = %.2f:1, want ≥ 4.5:1", name, token, got)
			}
		}
		// surface는 글자색이 아니라 면이다 — bg와 **구별되기만** 하면 된다 (카드가 스테이지
		// 위에서 보이는 최소 조건). 같은 값이면 카드가 사라진다.
		if colorTokenRGB(t, theme, "surface") == bg {
			t.Errorf("%s의 surface가 bg와 같다 — 카드가 스테이지에서 보이지 않는다", name)
		}
	}
}

// colorTokenRGB는 이름으로 색 토큰의 RGB를 집는다.
func colorTokenRGB(t *testing.T, theme soml.Theme, name string) string {
	t.Helper()
	for _, token := range theme.Colors {
		if token.Name == name {
			return token.RGB
		}
	}
	t.Fatalf("색 토큰 %q가 테마에 없다", name)
	return ""
}

// relativeLuminance는 WCAG 2.1의 상대 휘도다.
func relativeLuminance(t *testing.T, rgb string) float64 {
	t.Helper()
	r, g, b := parseHex(t, rgb)
	channel := func(u float64) float64 {
		if u <= 0.03928 {
			return u / 12.92
		}
		return math.Pow((u+0.055)/1.055, 2.4)
	}
	return 0.2126*channel(r) + 0.7152*channel(g) + 0.0722*channel(b)
}

// contrastRatio는 WCAG 2.1의 대비율이다 — 두 색의 순서에 대해 대칭이다.
func contrastRatio(t *testing.T, a, b string) float64 {
	t.Helper()
	la, lb := relativeLuminance(t, a), relativeLuminance(t, b)
	return (math.Max(la, lb) + 0.05) / (math.Min(la, lb) + 0.05)
}

// hueGap은 HSV 색상환에서 두 색이 떨어진 각도다 (0~180°).
func hueGap(t *testing.T, a, b string) float64 {
	t.Helper()
	gap := math.Abs(hueDegrees(t, a) - hueDegrees(t, b))
	if gap > 180 {
		gap = 360 - gap
	}
	return gap
}

// hueDegrees는 HSV 색상이다 (무채색은 0°).
func hueDegrees(t *testing.T, rgb string) float64 {
	t.Helper()
	r, g, b := parseHex(t, rgb)
	max := math.Max(r, math.Max(g, b))
	min := math.Min(r, math.Min(g, b))
	span := max - min
	if span == 0 {
		return 0
	}
	var hue float64
	switch max {
	case r:
		hue = math.Mod((g-b)/span, 6)
	case g:
		hue = (b-r)/span + 2
	default:
		hue = (r-g)/span + 4
	}
	hue *= 60
	if hue < 0 {
		hue += 360
	}
	return hue
}

// parseHex는 `RRGGBB`를 0~1 채널 셋으로 푼다.
func parseHex(t *testing.T, rgb string) (float64, float64, float64) {
	t.Helper()
	if len(rgb) != 6 {
		t.Fatalf("색 %q가 RRGGBB 6자리가 아니다", rgb)
	}
	channel := func(s string) float64 {
		v, err := strconv.ParseUint(s, 16, 8)
		if err != nil {
			t.Fatalf("색 %q를 읽을 수 없다: %v", rgb, err)
		}
		return float64(v) / 255
	}
	return channel(rgb[0:2]), channel(rgb[2:4]), channel(rgb[4:6])
}
