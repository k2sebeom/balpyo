package web

import (
	"fmt"
	"strings"
	"testing"
)

// 이 파일은 **PRD §9.4 표의 그라디언트·색 변형 행 회귀 장치**다. 도형 채움·
// 테두리 그라디언트는 svg `<defs>`의 `linearGradient`를 만들고 `url(#…)`로 가리키며, 텍스트
// 색 그라디언트는 `background-clip: text`, 색 변형은 `color-mix()`다.

// 도형 채움 그라디언트 → svg linearGradient def + fill: url(#…).
func TestShapeFillGradientEmitsSVGDef(t *testing.T) {
	out := renderString(t, subsetDeck(t, `<shape type="rect" fill="gradient(accent1 surface muted)"/>`))
	ink := inkOf(t, out, "s0.0")
	for _, want := range []string{
		"fill: url(#s0.0-fill-grad)",
		`<defs>`,
		`<linearGradient id="s0.0-fill-grad"`,
		"stop-color: var(--balpyo-color-accent1)",
		"stop-color: var(--balpyo-color-surface)",
		"stop-color: var(--balpyo-color-muted)",
	} {
		if !strings.Contains(ink, want) {
			t.Errorf("채움 그라디언트에 %q가 없다:\n%s", want, ink)
		}
	}
	// 균등 스톱 3개: offset 0 · 0.5 · 1.
	for _, off := range []string{`offset="0"`, `offset="0.5"`, `offset="1"`} {
		if !strings.Contains(ink, off) {
			t.Errorf("스톱 %q가 없다:\n%s", off, ink)
		}
	}
}

// 각도 → linearGradient 벡터 (기본 180deg = 위→아래).
func TestShapeFillGradientAngleVector(t *testing.T) {
	for _, tc := range []struct{ fill, want string }{
		{"gradient(accent1 bg)", `x1="0.5" y1="0" x2="0.5" y2="1"`},
		{"gradient(accent1 bg, 90deg)", `x1="0" y1="0.5" x2="1" y2="0.5"`},
	} {
		out := renderString(t, subsetDeck(t, fmt.Sprintf(`<shape type="rect" fill="%s"/>`, tc.fill)))
		if ink := inkOf(t, out, "s0.0"); !strings.Contains(ink, tc.want) {
			t.Errorf("fill=%q의 벡터에 %q가 없다:\n%s", tc.fill, tc.want, ink)
		}
	}
}

// 테두리 그라디언트 → stroke: url(#…) + stroke def.
func TestShapeBorderGradientEmitsStrokeURL(t *testing.T) {
	out := renderString(t, subsetDeck(t, `<shape type="rect" border="3 gradient(accent1 muted)"/>`))
	ink := inkOf(t, out, "s0.0")
	for _, want := range []string{
		"stroke: url(#s0.0-stroke-grad)",
		"stroke-width: 3px",
		`<linearGradient id="s0.0-stroke-grad"`,
	} {
		if !strings.Contains(ink, want) {
			t.Errorf("테두리 그라디언트에 %q가 없다:\n%s", want, ink)
		}
	}
}

// 색 변형 → color-mix() (토큰 변수 참조가 살아남는다).
func TestColorVariantsCompileToColorMix(t *testing.T) {
	cases := []struct {
		name string
		fill string
		want string
	}{
		{"shade", `shade(accent1, 40%)`, "fill: color-mix(in srgb, var(--balpyo-color-accent1), black 40%)"},
		{"tint", `tint(muted, 25%)`, "fill: color-mix(in srgb, var(--balpyo-color-muted), white 25%)"},
		{"alpha", `alpha(fg, 0.6)`, "fill: color-mix(in srgb, transparent, var(--balpyo-color-fg) 60%)"},
		{"shade 리터럴", `shade(#0F65F9, 30%)`, "fill: color-mix(in srgb, #0F65F9, black 30%)"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out := renderString(t, subsetDeck(t, fmt.Sprintf(`<shape type="rect" fill="%s"/>`, tc.fill)))
			ink := inkOf(t, out, "s0.0")
			if !strings.Contains(ink, tc.want) {
				t.Errorf("%s: %q가 없다:\n%s", tc.name, tc.want, ink)
			}
		})
	}
}

// 텍스트 색 그라디언트 → background-clip: text (색은 투명).
func TestTextColorGradientUsesBackgroundClip(t *testing.T) {
	out := renderString(t, subsetDeck(t, `<text color="gradient(accent1 muted, 45deg)">제목</text>`))
	for _, want := range []string{
		"background-image: linear-gradient(45deg, var(--balpyo-color-accent1), var(--balpyo-color-muted))",
		"background-clip: text",
		"color: transparent",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("텍스트 그라디언트에 %q가 없다", want)
		}
	}
}
