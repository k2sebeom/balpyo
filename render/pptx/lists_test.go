package pptx

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/k2sebeom/balpyo/render/geom"
	"github.com/k2sebeom/balpyo/render/web"
)

// listRect는 이 파일의 테스트가 쓰는 리스트 컨테이너 rect다. li rect는 지오메트리에
// 존재하지만(측정 대상 — geom.Refs) PPTX 리스트 도형은 컨테이너 rect 하나만 쓴다.
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

// <ul> 하나 = p:sp 하나, <li> 하나 = a:p 하나다. marL/indent는 웹 리스트 CSS 상수의
// EMU 환산이고(들여쓰기 값이 한 곳에서 나온다 — 항목 05의 export), indent는 음수
// 행잉 인덴트로 마커를 내민다. lnSpc는 블록과 같은 spcPts 규칙, buFont는 li의 확정
// 폰트(웹 li::before의 폰트 상속과 짝)다.
func TestRenderUnorderedList(t *testing.T) {
	deck := resolvedDeck(t, `<slide>
  <text>앞 블록</text>
  <ul>
    <li>첫째</li>
    <li>둘째</li>
    <li>셋째</li>
  </ul>
  <p>뒤 블록</p>
</slide>`)
	g := measuredGeometry(t, deck,
		bodyRect, listRect, itemRect, itemRect, itemRect,
		geom.Rect{X: 64, Y: 320, W: 1152, H: 36},
	)
	xml := slideXML(t, deck, g)

	// 리스트 전체가 텍스트 박스 하나다 — 항목마다 도형을 내면 PowerPoint에서 리스트가
	// 낱개 텍스트 박스로 조각난다.
	if got := strings.Count(xml, "<p:sp>"); got != 3 {
		t.Fatalf("p:sp %d개, want 3 (text/ul/p)\n%s", got, xml)
	}
	sp := spSection(t, xml, "ul s0.1")
	if got := strings.Count(sp, "<a:p>"); got != 3 {
		t.Fatalf("리스트 도형 안 a:p %d개, want 3 (li 하나 = 문단 하나)\n%s", got, sp)
	}

	// marL/indent는 웹 상수 × 9525다 — 값이 두 렌더러에 따로 적히면 들여쓰기가 갈라진다.
	wantPPr := fmt.Sprintf(`<a:pPr marL="%d" indent="-%d" algn="l">`,
		web.ListTextIndent*EMUPerPixel, web.ListMarkerHang*EMUPerPixel)
	if got := strings.Count(sp, wantPPr); got != 3 {
		t.Errorf("pPr %q %d개, want 3\n%s", wantPPr, got, sp)
	}
	// 행잉 인덴트 부호: marL 양수(텍스트 시작), indent 음수(마커 내밈).
	if !strings.Contains(sp, `marL="342900" indent="-342900"`) {
		t.Errorf("행잉 인덴트 값이 PowerPoint 1수준 기본(342900/-342900)이 아니다\n%s", sp)
	}
	for _, want := range []string{
		// li = body 역할: 24px × 1.5 × 0.75 × 100 = 2700 (블록과 같은 spcPts 규칙).
		`<a:spcPts val="2700"/>`,
		// 마커 폰트는 li의 확정 폰트다 — 명시하지 않으면 뷰어 기본 폰트가 마커를 그린다.
		`<a:buFont typeface="+mn-lt"/>`,
		// 마커 문자는 웹과 같은 상수에서 나온다.
		fmt.Sprintf(`<a:buChar char="%s"/>`, web.ListBulletChar),
	} {
		if got := strings.Count(sp, want); got != 3 {
			t.Errorf("%q %d개, want 3\n%s", want, got, sp)
		}
	}
	// 항목 사이 추가 간격은 없다 — 수직 리듬은 줄높이 소유다 (웹 ListItemGap=0의 짝).
	for _, banned := range []string{"<a:spcBef", "<a:spcAft"} {
		if strings.Contains(sp, banned) {
			t.Errorf("리스트 문단에 %s가 있다 — 항목 간격은 웹이 0으로 정했다\n%s", banned, sp)
		}
	}
	if web.ListItemGap != 0 {
		t.Error("web.ListItemGap이 0이 아니다 — PPTX 쪽 spcAft 대응을 함께 구현해야 한다")
	}

	// 좌표는 컨테이너의 잰 rect + 안전 여유(start: 오른쪽만)다.
	for _, want := range []string{
		`<a:off x="609600" y="1905000"/>`,           // 64px, 200px
		`<a:ext cx="11411712" cy="1028700"/>`,       // 1152px×1.04, 108px
		`<p:cNvSpPr txBox="1"/>`,                    // 텍스트 박스 표시
		`<a:bodyPr wrap="square" lIns="0" tIns="0"`, // 블록과 같은 인셋 0 + 줄바꿈
	} {
		if !strings.Contains(sp, want) {
			t.Errorf("리스트 도형에 %q가 없다\n%s", want, sp)
		}
	}

	// 문서 순서 = spTree 순서, 도형 id 연속성 (리스트가 id를 건너뛰지 않는다).
	for _, want := range []string{
		`<p:cNvPr id="2" name="text s0.0"/>`,
		`<p:cNvPr id="3" name="ul s0.1"/>`,
		`<p:cNvPr id="4" name="p s0.2"/>`,
	} {
		if !strings.Contains(xml, want) {
			t.Errorf("slide1.xml에 %q가 없다\n%s", want, xml)
		}
	}
	if !(strings.Index(xml, `name="text s0.0"`) < strings.Index(xml, `name="ul s0.1"`) &&
		strings.Index(xml, `name="ul s0.1"`) < strings.Index(xml, `name="p s0.2"`)) {
		t.Error("spTree 순서가 문서 순서가 아니다")
	}
	// 문단 순서 = li 순서.
	if !(strings.Index(sp, "<a:t>첫째</a:t>") < strings.Index(sp, "<a:t>둘째</a:t>") &&
		strings.Index(sp, "<a:t>둘째</a:t>") < strings.Index(sp, "<a:t>셋째</a:t>")) {
		t.Error("문단 순서가 li 순서가 아니다")
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

// li 안의 인라인은 항목 06의 런 분할기를 그대로 지난다 — strong은 rPr 완전 명시 런,
// <a>는 hlinkClick + 슬라이드 rels의 External 관계다. rel id는 텍스트 블록과 리스트가
// 섞여도 문서 순서로 결정적이고, 같은 URL은 관계를 재사용한다.
func TestRenderListItemInlineRuns(t *testing.T) {
	deck := resolvedDeck(t, `<slide>
  <p>자료: <a href="https://a.example">첫째</a></p>
  <ul>
    <li>참고 <a href="https://b.example">문서</a></li>
    <li><strong>강조</strong> 항목, 다시 <a href="https://a.example">첫째</a></li>
  </ul>
</slide>`)
	g := measuredGeometry(t, deck, bodyRect, listRect, itemRect, itemRect)
	parts, _ := renderPackage(t, NewWithGeometry(g), deck)
	xml := parts[slidePart(1)]

	sp := spSection(t, xml, "ul s0.1")
	// strong 런은 인라인 완전 명시다 (i/u 포함 — 항목 06의 runContext 규칙).
	if !strings.Contains(sp, `<a:rPr sz="1800" b="1" i="0" u="none">`) {
		t.Errorf("li 안 strong 런의 rPr가 완전 명시가 아니다\n%s", sp)
	}
	// 경계 공백은 앞 런 끝에 남는다 (파서의 인라인 경계 공백 계약).
	if !strings.Contains(sp, "<a:t>참고 </a:t>") {
		t.Errorf("li의 첫 런이 경계 공백을 잃었다\n%s", sp)
	}
	// rel id: 문서 순서로 a.example→rId3(p에서), b.example→rId4(리스트 경유).
	// 리스트 안의 세 번째 링크는 rId3을 재사용한다.
	if !strings.Contains(sp, `<a:hlinkClick r:id="rId4"/>`) {
		t.Errorf("li 링크 런에 rId4 hlinkClick이 없다\n%s", sp)
	}
	if !strings.Contains(sp, `<a:hlinkClick r:id="rId3"/>`) {
		t.Errorf("li의 같은 URL 링크가 rId3을 재사용하지 않았다\n%s", sp)
	}
	rels := parts[slideRelsPart(1)]
	for _, want := range []string{
		`<Relationship Id="rId3" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/hyperlink" Target="https://a.example" TargetMode="External"/>`,
		`<Relationship Id="rId4" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/hyperlink" Target="https://b.example" TargetMode="External"/>`,
	} {
		if !strings.Contains(rels, want) {
			t.Errorf("slide1.xml.rels에 %q가 없다\n%s", want, rels)
		}
	}
}

// <li> 안의 <br/>은 같은 문단 안의 a:br이다 — 마커 없는 이어짐 줄 (웹의 <br>과 짝:
// li 블록 안 줄바꿈이지 새 항목이 아니다).
func TestRenderListItemBreak(t *testing.T) {
	deck := resolvedDeck(t, `<slide>
  <ul>
    <li>첫 줄<br/>이어짐 줄</li>
  </ul>
</slide>`)
	g := measuredGeometry(t, deck, listRect, itemRect)
	xml := slideXML(t, deck, g)

	if got := strings.Count(xml, "<a:p>"); got != 1 {
		t.Fatalf("문단 %d개, want 1 — br이 문단(=항목)을 쪼갰다\n%s", got, xml)
	}
	if got := strings.Count(xml, "<a:br/>"); got != 1 {
		t.Fatalf("a:br %d개, want 1\n%s", got, xml)
	}
	if !(strings.Index(xml, "<a:t>첫 줄</a:t>") < strings.Index(xml, "<a:br/>") &&
		strings.Index(xml, "<a:br/>") < strings.Index(xml, "<a:t>이어짐 줄</a:t>")) {
		t.Errorf("a:br의 위치가 런 사이가 아니다\n%s", xml)
	}
}

// 빈 <li>는 런 없는 문단으로 남는다 — 건너뛰면 항목 순서(문단 = 항목)와 자리가 내용에
// 따라 흔들린다 (writeTextShape의 빈 텍스트 정책과 같은 태도).
func TestRenderEmptyListItemKeepsParagraph(t *testing.T) {
	deck := resolvedDeck(t, `<slide>
  <ul>
    <li>첫째</li>
    <li></li>
    <li>셋째</li>
  </ul>
</slide>`)
	g := measuredGeometry(t, deck, listRect, itemRect, itemRect, itemRect)
	xml := slideXML(t, deck, g)

	if got := strings.Count(xml, "<a:p>"); got != 3 {
		t.Fatalf("문단 %d개, want 3 (빈 li도 문단이다)\n%s", got, xml)
	}
	if got := strings.Count(xml, "<a:r>"); got != 2 {
		t.Errorf("런 %d개, want 2 (빈 li는 런이 없다)\n%s", got, xml)
	}
}

// <li> 없는 <ul>은 파서가 막지 않는다 — 도형은 낸다 (spTree 순서·도형 id가 내용에 따라
// 흔들리지 않는다). txBody는 문단 하나 이상을 요구하므로(CT_TextBody의 p+) 서식 없는
// 빈 문단 하나가 스키마를 채운다.
func TestRenderEmptyListStillEmitsShape(t *testing.T) {
	deck := resolvedDeck(t, `<slide>
  <ul></ul>
  <p>뒤 블록</p>
</slide>`)
	g := measuredGeometry(t, deck, geom.Rect{X: 64, Y: 64, W: 1152, H: 0}, bodyRect)
	xml := slideXML(t, deck, g)

	if got := strings.Count(xml, "<p:sp>"); got != 2 {
		t.Fatalf("p:sp %d개, want 2\n%s", got, xml)
	}
	sp := spSection(t, xml, "ul s0.0")
	if !strings.Contains(sp, "<a:p/>") {
		t.Errorf("빈 리스트의 txBody에 빈 문단이 없다 (p+ 스키마)\n%s", sp)
	}
	if strings.Contains(sp, "<a:r>") || strings.Contains(sp, "<a:buChar") {
		t.Errorf("빈 리스트에 런·마커가 있다\n%s", sp)
	}
	// 뒤 블록의 도형 id가 이어진다.
	if !strings.Contains(xml, `<p:cNvPr id="3" name="p s0.1"/>`) {
		t.Errorf("빈 리스트 뒤 도형의 id가 이어지지 않았다\n%s", xml)
	}
}

// 리스트가 든 덱도 같은 입력이 같은 바이트를 내야 한다 — 문단 조립·rel id 배정에 맵
// 순회가 새면 여기서 잡힌다.
func TestRenderListDeckIsByteIdentical(t *testing.T) {
	render := func() []byte {
		deck := resolvedDeck(t, `<slide>
  <ol>
    <li>첫째 <a href="https://a.example">링크</a></li>
    <li><em>둘째</em></li>
  </ol>
</slide>`)
		g := measuredGeometry(t, deck, listRect, itemRect, itemRect)
		files, err := NewWithGeometry(g).Render(deck)
		if err != nil {
			t.Fatal(err)
		}
		return files[FileName(deck)]
	}
	if !bytes.Equal(render(), render()) {
		t.Error("같은 덱이 다른 바이트를 냈다")
	}
}
