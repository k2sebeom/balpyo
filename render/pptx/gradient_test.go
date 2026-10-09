package pptx

import (
	"strings"
	"testing"

	"github.com/k2sebeom/balpyo/render/geom"
)

// 그라디언트·색 변형 → OOXML. 채움은 `a:gradFill`, 테두리는
// `a:ln` > `gradFill`, 텍스트 색은 rPr > `gradFill`, 변형은 `lumMod`/`lumOff`/`alpha`다. 토큰
// 스톱은 schemeClr로 남는다.
func TestRenderGradientsAndColorVariants(t *testing.T) {
	tests := []struct {
		name, body string
		from       string // 이 표기부터의 조각에서 찾는다
		want       []string
	}{
		{"채움", `<shape type="rect" fill="gradient(accent1 surface muted)" w="200" h="100"/>`, "<a:gradFill>", []string{
			`<a:gs pos="0">`, `<a:gs pos="50000">`, `<a:gs pos="100000">`,
			`<a:schemeClr val="accent1"/>`, `<a:schemeClr val="bg2"/>`, `<a:schemeClr val="tx2"/>`,
			// 기본 180deg(위→아래) → OOXML 90도.
			`<a:lin ang="5400000" scaled="1"/>`,
		}},
		// 45deg → (45-90) mod 360 = 315도.
		{"채움 각도", `<shape type="rect" fill="gradient(accent1 bg, 45deg)" w="200" h="100"/>`, "<a:gradFill>", []string{
			`<a:lin ang="18900000" scaled="1"/>`,
		}},
		{"테두리", `<shape type="rect" border="3 gradient(accent1 muted)" fill="none" w="200" h="100"/>`, "<a:ln ", []string{
			`<a:ln w="28575">`, "<a:gradFill>", `<a:prstDash val="solid"/>`,
		}},
		{"텍스트 색", `<text color="gradient(accent1 muted)">제목</text>`, "<a:rPr ", []string{
			"<a:gradFill>", `<a:schemeClr val="accent1"/>`, `<a:schemeClr val="tx2"/>`,
		}},
		// shade 40% → lumMod 60%, tint 25% → lumMod 75% + lumOff 25%, alpha 0.6 → 60000.
		{"shade", `<shape type="rect" fill="shade(accent1, 40%)" w="200" h="100"/>`, "<a:solidFill>", []string{
			`<a:schemeClr val="accent1">`, `<a:lumMod val="60000"/>`,
		}},
		{"tint", `<shape type="rect" fill="tint(muted, 25%)" w="200" h="100"/>`, "<a:solidFill>", []string{
			`<a:schemeClr val="tx2">`, `<a:lumMod val="75000"/>`, `<a:lumOff val="25000"/>`,
		}},
		{"alpha", `<shape type="rect" fill="alpha(fg, 0.6)" w="200" h="100"/>`, "<a:solidFill>", []string{
			`<a:schemeClr val="tx1">`, `<a:alpha val="60000"/>`,
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			deck := resolvedDeck(t, "<slide>\n  "+tc.body+"\n</slide>")
			xml := slideXML(t, deck, measuredGeometry(t, deck, geom.Rect{X: 100, Y: 100, W: 200, H: 100}))
			at := strings.Index(xml, tc.from)
			if at < 0 {
				t.Fatalf("%q가 없다:\n%s", tc.from, xml)
			}
			for _, want := range tc.want {
				if !strings.Contains(xml[at:], want) {
					t.Errorf("%q 뒤에 %q가 없다:\n%s", tc.from, want, xml[at:])
				}
			}
		})
	}
}
