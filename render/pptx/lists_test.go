package pptx

import (
	"fmt"
	"strings"
	"testing"

	"github.com/k2sebeom/balpyo/render/geom"
	"github.com/k2sebeom/balpyo/render/web"
)

// listRect·itemRect는 리스트 컨테이너와 li의 rect다 — PPTX 리스트 도형은 컨테이너 rect만 쓴다.
var (
	listRect = geom.Rect{X: 64, Y: 200, W: 1152, H: 108}
	itemRect = geom.Rect{X: 64, Y: 200, W: 1152, H: 36}
)

// spSection은 slide XML에서 이름이 name인 p:sp 조각 하나를 돌려준다.
func spSection(t *testing.T, xml, name string) string {
	t.Helper()
	at := strings.Index(xml, fmt.Sprintf("name=%q", name))
	if at < 0 {
		t.Fatalf("이름 %q의 도형이 없다\n%s", name, xml)
	}
	end := strings.Index(xml[at:], "</p:sp>")
	if end < 0 {
		t.Fatalf("%q 도형이 닫히지 않았다\n%s", name, xml)
	}
	return xml[at : at+end]
}

// li 하나 = a:p 하나이고, 마커는 웹 리스트 CSS와 같은 상수에서 나온다: marL/indent는 웹 상수의
// EMU 환산(음수 indent가 마커를 내민다), buFont는 li의 확정 폰트, 항목 사이 간격은 0이다
// (수직 리듬은 줄높이 소유 — 웹 ListItemGap=0의 짝).
func TestRenderUnorderedList(t *testing.T) {
	deck := resolvedDeck(t, `<slide>
  <ul>
    <li>첫째</li>
    <li>둘째</li>
    <li>셋째</li>
  </ul>
</slide>`)
	sp := spSection(t, slideXML(t, deck, measuredGeometry(t, deck, listRect, itemRect, itemRect, itemRect)), "ul s0.0")

	for _, want := range []string{
		fmt.Sprintf(`<a:pPr marL="%d" indent="-%d" algn="l">`,
			web.ListTextIndent*EMUPerPixel, web.ListMarkerHang*EMUPerPixel),
		`<a:spcPts val="2700"/>`, // body 24px × 1.5 × 0.75
		`<a:buFont typeface="+mn-lt"/>`,
		fmt.Sprintf(`<a:buChar char="%s"/>`, web.ListBulletChar),
	} {
		if got := strings.Count(sp, want); got != 3 {
			t.Errorf("%q %d개, want 3\n%s", want, got, sp)
		}
	}
	for _, banned := range []string{"<a:spcBef", "<a:spcAft"} {
		if strings.Contains(sp, banned) {
			t.Errorf("리스트 문단에 %s가 있다\n%s", banned, sp)
		}
	}
	if web.ListItemGap != 0 {
		t.Error("web.ListItemGap이 0이 아니다 — PPTX 쪽 spcAft 대응을 함께 구현해야 한다")
	}
}

// ul과 ol의 차이는 마커 종류(buChar/buAutoNum)뿐이어야 한다 — 들여쓰기·간격·줄간격이
// 갈라지면 리스트 종류를 바꿀 때 배치가 움직인다.
func TestRenderOrderedListDiffersOnlyInBullet(t *testing.T) {
	render := func(container string) string {
		deck := resolvedDeck(t, fmt.Sprintf(`<slide>
  <%s>
    <li>첫째</li>
    <li>둘째</li>
  </%s>
</slide>`, container, container))
		g := measuredGeometry(t, deck, listRect, itemRect, itemRect)
		return slideXML(t, deck, g)
	}
	ul, ol := render("ul"), render("ol")

	if got := strings.Count(ol, `<a:buAutoNum type="arabicPeriod"/>`); got != 2 {
		t.Fatalf("buAutoNum %d개, want 2 (웹의 \"1.\" 형식과 짝)\n%s", got, ol)
	}
	if strings.Contains(ol, "<a:buChar") {
		t.Errorf("ol에 buChar가 있다\n%s", ol)
	}
	// 마커 요소와 도형 이름을 치환하면 두 XML이 같아야 한다 — 그 밖의 차이는 전부 버그다.
	transformed := strings.ReplaceAll(ul,
		fmt.Sprintf(`<a:buChar char="%s"/>`, web.ListBulletChar),
		`<a:buAutoNum type="arabicPeriod"/>`)
	transformed = strings.ReplaceAll(transformed, `name="ul s0.0"`, `name="ol s0.0"`)
	if transformed != ol {
		t.Errorf("ul과 ol의 차이가 마커뿐이 아니다\nul(치환):\n%s\nol:\n%s", transformed, ol)
	}
}
