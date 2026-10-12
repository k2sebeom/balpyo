package pptx

import (
	"fmt"
	"strings"
	"testing"

	"github.com/k2sebeom/balpyo/render/geom"
	"github.com/k2sebeom/balpyo/soml"
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

// 선형 그라디언트의 각도는 웹이 그것을 그리는 방식과 같은 뜻이어야 한다. 배경·덮개·텍스트 색은 웹에서
// CSS `linear-gradient`라 각도가 박스 위의 참 각도다 → `scaled="0"`. 도형 채움·테두리는 웹에서 SVG
// objectBoundingBox라 각도가 박스 비율로 늘어난다 → `scaled="1"`. 둘이 엇갈리면 16:9 슬라이드의 210deg
// 띠가 웹에서는 약 30도, PPTX에서는 약 18도 기운다.
func TestGradientAngleScalingFollowsWebPath(t *testing.T) {
	grad := func() *soml.Fill {
		return &soml.Fill{Color: soml.Color{Gradient: &soml.Gradient{
			Stops: []soml.Color{{Token: "accent1"}, {Token: "bg"}}, Angle: 210,
		}}}
	}
	deck := resolvedDeck(t, "<slide>\n"+
		`  <text color="gradient(accent1 muted, 210deg)">제목</text>`+"\n"+
		`  <shape type="rect" fill="gradient(accent1 bg, 210deg)" border="3 gradient(accent1 muted, 210deg)" w="200" h="100"/>`+"\n"+
		"</slide>")
	deck.Slides[0].Background = &soml.Background{Fill: grad(), Overlay: grad()}
	html := renderWebString(t, deck)
	xml := slideXML(t, deck, measuredGeometry(t, deck,
		geom.Rect{X: 100, Y: 100, W: 400, H: 60}, geom.Rect{X: 100, Y: 300, W: 200, H: 100}))

	// 웹이 어느 길로 그리는지 먼저 확인한다 — 이 테스트의 전제다.
	if n := strings.Count(html, "linear-gradient(210deg"); n < 3 {
		t.Fatalf("웹의 CSS 그라디언트가 배경·덮개·텍스트 셋이어야 한다 (%d개):\n%s", n, html)
	}
	if !strings.Contains(html, "<linearGradient ") {
		t.Fatalf("웹 도형 그라디언트가 SVG linearGradient가 아니다")
	}

	ang := GradientAngle(210)
	trueAngle := fmt.Sprintf(`<a:lin ang="%d" scaled="0"/>`, ang)
	boxAngle := fmt.Sprintf(`<a:lin ang="%d" scaled="1"/>`, ang)
	section := func(from, to string) string {
		at := strings.Index(xml, from)
		if at < 0 {
			t.Fatalf("%q가 없다:\n%s", from, xml)
		}
		rest := xml[at:]
		end := strings.Index(rest, to)
		if end < 0 {
			t.Fatalf("%q 뒤에 %q가 없다:\n%s", from, to, rest)
		}
		return rest[:end]
	}
	cases := []struct{ name, part, want string }{
		{"배경", section("<p:bg>", "</p:bg>"), trueAngle},
		{"덮개", section(`name="background overlay"`, "</p:sp>"), trueAngle},
		{"텍스트 색", section("<a:rPr ", "</a:rPr>"), trueAngle},
		// 도형은 잰 rect(100, 300)의 xfrm으로 찾는다 — 덮개도 rect 프리셋이다.
		{"도형 채움", section(fmt.Sprintf(`<a:off x="%d" y="%d"/>`, EMU(100), EMU(300)), "<a:ln "), boxAngle},
		{"도형 테두리", section(`<a:ln w="28575">`, "</a:ln>"), boxAngle},
	}
	for _, tc := range cases {
		if strings.Count(tc.part, "<a:lin ") != 1 || !strings.Contains(tc.part, tc.want) {
			t.Errorf("%s: %s가 아니다:\n%s", tc.name, tc.want, tc.part)
		}
	}
}
