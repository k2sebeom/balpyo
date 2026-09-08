package pptx

import (
	"strings"
	"testing"

	"github.com/k2sebeom/balpyo/render/geom"
)

// 이 파일은 **PRD §9.4 표의 그라디언트·색 변형 행 회귀 장치**다 (Iteration 8) — PPTX 쪽.
// 채움 그라디언트는 `a:gradFill`, 테두리 그라디언트는 `a:ln`+`gradFill`, 텍스트 색
// 그라디언트는 rPr+`gradFill`, 색 변형은 `lumMod`/`lumOff`/`alpha`다. 토큰 스톱은
// `schemeClr` 참조를 유지한다 (제품 핵심 계약).

func gradientShapeXML(t *testing.T, body string) string {
	t.Helper()
	deck := resolvedDeck(t, "<slide>\n  "+body+"\n</slide>")
	g := measuredGeometry(t, deck, geom.Rect{X: 100, Y: 100, W: 200, H: 100})
	return slideXML(t, deck, g)
}

// 채움 그라디언트 → gradFill > gsLst > gs(pos) + lin(ang, scaled). 스톱은 schemeClr 유지.
func TestRenderShapeFillGradient(t *testing.T) {
	xml := gradientShapeXML(t, `<shape type="rect" fill="gradient(accent1 surface muted)" w="200" h="100"/>`)
	for _, want := range []string{
		"<a:gradFill>",
		"<a:gsLst>",
		`<a:gs pos="0">`,
		`<a:gs pos="50000">`,
		`<a:gs pos="100000">`,
		// 토큰 스톱은 schemeClr 참조로 남는다 (accent1→accent1, surface→bg2, muted→tx2).
		`<a:schemeClr val="accent1"/>`,
		`<a:schemeClr val="bg2"/>`,
		`<a:schemeClr val="tx2"/>`,
		// 기본 180deg(위→아래) → OOXML 90도 = 5400000.
		`<a:lin ang="5400000" scaled="1"/>`,
	} {
		if !strings.Contains(xml, want) {
			t.Errorf("채움 그라디언트에 %q가 없다:\n%s", want, xml)
		}
	}
}

// 각도 인자 → OOXML ang 변환 (45deg → (45-90) mod 360 = 315도 = 18900000).
func TestRenderShapeFillGradientAngle(t *testing.T) {
	xml := gradientShapeXML(t, `<shape type="rect" fill="gradient(accent1 bg, 45deg)" w="200" h="100"/>`)
	if !strings.Contains(xml, `<a:lin ang="18900000" scaled="1"/>`) {
		t.Errorf("45deg → ang 18900000이 아니다:\n%s", xml)
	}
}

// 테두리 그라디언트 → a:ln > gradFill (+ prstDash).
func TestRenderShapeBorderGradient(t *testing.T) {
	xml := gradientShapeXML(t, `<shape type="rect" border="3 gradient(accent1 muted)" fill="none" w="200" h="100"/>`)
	ln := xml[strings.Index(xml, "<a:ln "):]
	for _, want := range []string{
		`<a:ln w="28575">`,
		"<a:gradFill>",
		`<a:prstDash val="solid"/>`,
	} {
		if !strings.Contains(ln, want) {
			t.Errorf("테두리 그라디언트에 %q가 없다:\n%s", want, ln)
		}
	}
}

// 색 변형 → lumMod/lumOff/alpha, schemeClr 유지.
func TestRenderColorVariants(t *testing.T) {
	cases := []struct {
		name string
		fill string
		want []string
	}{
		// shade 40% → 어둡게 → lumMod 60%.
		{"shade", `shade(accent1, 40%)`, []string{`<a:schemeClr val="accent1">`, `<a:lumMod val="60000"/>`}},
		// tint 25% → 밝게 → lumMod 75% + lumOff 25%.
		{"tint", `tint(muted, 25%)`, []string{`<a:schemeClr val="tx2">`, `<a:lumMod val="75000"/>`, `<a:lumOff val="25000"/>`}},
		// alpha 0.6 → a:alpha 60000.
		{"alpha", `alpha(fg, 0.6)`, []string{`<a:schemeClr val="tx1">`, `<a:alpha val="60000"/>`}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			xml := gradientShapeXML(t, `<shape type="rect" fill="`+tc.fill+`" w="200" h="100"/>`)
			for _, want := range tc.want {
				if !strings.Contains(xml, want) {
					t.Errorf("%s: %q가 없다:\n%s", tc.name, want, xml)
				}
			}
		})
	}
}

// 텍스트 색 그라디언트 → rPr 안의 gradFill (solidFill이 아니다).
func TestRenderTextColorGradient(t *testing.T) {
	deck := resolvedDeck(t, `<slide>
  <text color="gradient(accent1 muted)">제목</text>
</slide>`)
	g := measuredGeometry(t, deck, geom.Rect{X: 64, Y: 64, W: 1152, H: 36})
	xml := slideXML(t, deck, g)
	rpr := xml[strings.Index(xml, "<a:rPr "):]
	for _, want := range []string{
		"<a:gradFill>",
		`<a:schemeClr val="accent1"/>`,
		`<a:schemeClr val="tx2"/>`,
	} {
		if !strings.Contains(rpr, want) {
			t.Errorf("텍스트 그라디언트 rPr에 %q가 없다:\n%s", want, rpr)
		}
	}
}
