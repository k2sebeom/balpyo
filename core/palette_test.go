package core

import (
	"math"
	"strconv"
	"testing"

	"github.com/k2sebeom/balpyo/soml"
)

// accent 팔레트가 지켜야 하는 성질을 **수치로** 못 박는다 (P4.1 — D12).
//
// 값 자체는 tokens.json이 소유하고 TestLoadBuiltinMinimal이 표로 지킨다. 이 파일이 더하는
// 것은 **왜 그 값인가**다: 강조색을 고르는 규칙이 코드에 없으면 다음 사람이 "파랑 옆에
// 어울리는 색"을 눈으로 골라 대비가 무너지고, 그 사실은 덱을 프로젝터에 걸어야 드러난다.
//
// 규칙 셋, 여섯 색 전수:
//
//  1. **bg 대비 4.5:1 이상** (WCAG AA 본문 기준). 대비율은 대칭이므로 이 한 수치가
//     "accent 글자를 bg 위에 쓴다"와 "bg 글자를 accent 면 위에 쓴다"를 함께 보장한다 —
//     둘 다 이 테마의 기본 사용법이다 (도형 채움 + `color="bg"` 라벨이 카드의 헤더 띠다).
//  2. **색상(hue)이 서로 25° 이상 떨어진다.** 위계를 만들려면 옆에 놓았을 때 다른 색으로
//     보여야 한다.
//  3. **밝기(상대 휘도)가 한 띠에 있다** (최대/최소 ≤ 1.25). 한 색만 밝거나 어두우면 강조의
//     위계가 아니라 소음이 된다 — accent4를 골랐을 때 accent1보다 더 크게 외치지 않아야 한다.
//
// surface(F3F4F6) 위에서는 같은 여섯 색이 4.33~4.89:1이다 (accent1이 4.49로 이미 4.5 아래다).
// 기준을 bg로 두는 이유: accent1의 값은 팔레트가 하나였을 때부터의 baseline이고, 새 색들을
// 그보다 엄격하게 잡으면 같은 팔레트 안에서 규칙이 둘이 된다. 회색 면 위의 작은 글자는
// 저자가 accent를 쓸 자리가 아니다.
func TestMinimalAccentPalette(t *testing.T) {
	theme := minimalTheme()
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
