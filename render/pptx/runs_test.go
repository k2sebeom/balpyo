package pptx

import (
	"strings"
	"testing"

	"github.com/k2sebeom/balpyo/render/geom"
	"github.com/k2sebeom/balpyo/soml"
)

// bodyRect는 이 파일의 테스트가 공유하는 rect다 — 관심은 txBody 안이라 좌표는 아무 값이어도 된다.
var bodyRect = geom.Rect{X: 64, Y: 64, W: 1152, H: 36}

// singleShapeXML은 요소 하나짜리 슬라이드의 slide1.xml이다.
func singleShapeXML(t *testing.T, src string) string {
	t.Helper()
	deck := resolvedDeck(t, src)
	g := measuredGeometry(t, deck, bodyRect)
	return slideXML(t, deck, g)
}

// <a>는 rPr 안의 hlinkClick이 되고, 목적지는 슬라이드 rels의 External 관계로 실린다.
// u="sng"·solidFill을 항상 낸다 — 뷰어의 자동 밑줄·자동 색에 기대면 뷰어가 바뀔 때
// 두 산출물이 갈라진다. href의 &는 원문 그대로다 (SOML은 엔티티를 해석하지 않는다) —
// rels에 실릴 때만 XML 이스케이프된다.
func TestRunHyperlink(t *testing.T) {
	deck := resolvedDeck(t, `<slide>
  <p>자료: <a href="https://example.com/deck?a=1&b=2">발표 자료</a></p>
</slide>`)
	g := measuredGeometry(t, deck, bodyRect)
	parts, _ := renderPackage(t, NewWithGeometry(g), deck)
	xml := parts[slidePart(1)]

	link := strings.Index(xml, "<a:t>발표 자료</a:t>")
	if link < 0 {
		t.Fatalf("링크 런이 없다\n%s", xml)
	}
	run := xml[strings.LastIndex(xml[:link], "<a:r>"):link]
	for _, want := range []string{
		`u="sng"`,
		// 링크 색은 accent1 토큰이다 (core의 링크 델타) — 테마 파트의 hlink 슬롯이 같은
		// 토큰에서 빌려 채워지므로 두 산출물의 링크 색이 한 표에서 나온다.
		`<a:schemeClr val="accent1"/>`,
		// hlinkClick은 latin/ea 뒤다 (CT_TextCharacterProperties의 자식 순서).
		`<a:hlinkClick r:id="rId3"/>`,
	} {
		if !strings.Contains(run, want) {
			t.Errorf("링크 런에 %q가 없다\n%s", want, run)
		}
	}
	if strings.Index(run, "<a:hlinkClick") < strings.Index(run, "<a:latin") {
		t.Error("hlinkClick이 latin보다 앞이다 — 스키마 순서 위반")
	}

	rels := parts[slideRelsPart(1)]
	want := `<Relationship Id="rId3" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/hyperlink" Target="https://example.com/deck?a=1&amp;b=2" TargetMode="External"/>`
	if !strings.Contains(rels, want) {
		t.Errorf("slide1.xml.rels에 %q가 없다\n%s", want, rels)
	}
}

// rel id는 결정적이다: rId1=레이아웃, rId2=노트(예약), rId3부터 외부 URL이 문서 순서로
// 붙는다. 같은 URL은 관계 하나를 재사용한다 — 관계의 정체는 (Type, Target, Mode)이고
// 중복은 rels 파트의 잡음일 뿐이다.
func TestRunHyperlinkRelIDsAreDeterministic(t *testing.T) {
	deck := resolvedDeck(t, `<slide>
  <p><a href="https://a.example">첫째</a> <a href="https://b.example">둘째</a> <a href="https://a.example">다시 첫째</a></p>
  <note>발표자 노트</note>
</slide>`)
	g := measuredGeometry(t, deck, bodyRect)
	parts, _ := renderPackage(t, NewWithGeometry(g), deck)
	xml := parts[slidePart(1)]
	rels := parts[slideRelsPart(1)]

	// 문서 순서: a.example → rId3, b.example → rId4. 세 번째 링크는 rId3 재사용.
	if got := strings.Count(xml, `<a:hlinkClick r:id="rId3"/>`); got != 2 {
		t.Errorf("rId3 hlinkClick %d개, want 2 (같은 URL은 관계를 재사용한다)\n%s", got, xml)
	}
	if !strings.Contains(xml, `<a:hlinkClick r:id="rId4"/>`) {
		t.Errorf("rId4 hlinkClick이 없다\n%s", xml)
	}
	for _, want := range []string{
		// 노트가 있어도 링크는 rId3부터다 — rId2는 노트 자리로 예약되어 노트 유무가
		// 링크 번호를 밀지 않는다.
		`<Relationship Id="rId2" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/notesSlide"`,
		`<Relationship Id="rId3" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/hyperlink" Target="https://a.example" TargetMode="External"/>`,
		`<Relationship Id="rId4" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/hyperlink" Target="https://b.example" TargetMode="External"/>`,
	} {
		if !strings.Contains(rels, want) {
			t.Errorf("slide1.xml.rels에 %q가 없다\n%s", want, rels)
		}
	}
	if got := strings.Count(rels, "https://a.example"); got != 1 {
		t.Errorf("같은 URL의 관계가 %d개다, want 1\n%s", got, rels)
	}
}

// pre의 보존된 개행은 a:br이 된다 — 웹의 white-space: pre와 같은 결정 (한쪽이 개행을
// 접으면 줄 수가 갈라진다). 빈 줄은 연속 a:br이고 빈 런을 만들지 않는다. bodyPr@wrap은
// none이다 — 웹이 줄바꿈하지 않기로 했으므로 PPTX도 접지 않는다.
func TestRunPreNewlines(t *testing.T) {
	xml := singleShapeXML(t, "<slide>\n  <pre>first line\n\nthird  line</pre>\n</slide>")

	if got := strings.Count(xml, "<a:br/>"); got != 2 {
		t.Fatalf("a:br %d개, want 2 (빈 줄 = 연속 a:br)\n%s", got, xml)
	}
	// 연속 공백도 내용이다 (파서가 pre 안을 접지 않는다).
	for _, want := range []string{"<a:t>first line</a:t>", "<a:t>third  line</a:t>"} {
		if !strings.Contains(xml, want) {
			t.Errorf("slide1.xml에 %q가 없다\n%s", want, xml)
		}
	}
	// 빈 잎·빈 줄이 빈 a:t를 만들지 않는다.
	if strings.Contains(xml, "<a:t></a:t>") {
		t.Errorf("빈 런이 있다\n%s", xml)
	}
	if !strings.Contains(xml, `<a:bodyPr wrap="none"`) {
		t.Errorf("pre의 bodyPr@wrap이 none이 아니다\n%s", xml)
	}
	if got := strings.Count(xml, "<a:p>"); got != 1 {
		t.Errorf("문단 %d개, want 1 — pre는 한 문단의 여러 줄이다\n%s", got, xml)
	}
}

// 자간은 em × 런 크기(pt) × 100 = `spc`(1/100pt)이고 대문자는 `cap="all"`이다. 원문은 저자가 쓴
// 대소문자 그대로 `a:t`에 남는다 — 대문자는 표시 속성이다.
func TestRunTrackingAndCase(t *testing.T) {
	deck := resolvedDeck(t, `<slide>
  <p>Kicker</p>
</slide>`)
	p := deck.Slides[0].Content[0].(*soml.Element)
	p.Appearance.Text.Tracking = 0.16
	p.Appearance.Text.Case = soml.CaseUpper
	xml := slideXML(t, deck, measuredGeometry(t, deck, bodyRect))
	// 본문 24px = 18pt → 0.16 × 18 × 100 = 288.
	for _, want := range []string{`spc="288"`, `cap="all"`, "<a:t>Kicker</a:t>"} {
		if !strings.Contains(xml, want) {
			t.Errorf("런에 %q가 없다\n%s", want, xml)
		}
	}
}

// <mark> 안의 런은 전부 a:highlight를 갖는다 (PPTX 런은 평평하다). 색은 schemeClr 참조 + 변형이고
// 자리는 채움 뒤 · latin 앞이다 (CT_TextCharacterProperties의 자식 순서).
func TestRunMarkHighlight(t *testing.T) {
	xml := singleShapeXML(t, `<slide>
  <p>EC2는 <mark>몇 분 안에 <strong>초 단위로</strong></mark> 낸다</p>
</slide>`)
	for _, text := range []string{"몇 분 안에 ", "초 단위로"} {
		at := strings.Index(xml, "<a:t>"+text+"</a:t>")
		if at < 0 {
			t.Fatalf("런 %q가 없다\n%s", text, xml)
		}
		run := xml[strings.LastIndex(xml[:at], "<a:r>"):at]
		hl := strings.Index(run, `<a:highlight>`)
		if hl < 0 || !strings.Contains(run[hl:], `<a:schemeClr val="accent4">`) {
			t.Errorf("런 %q에 토큰 형광펜이 없다\n%s", text, run)
		}
		if hl > strings.Index(run, "<a:latin") || hl < strings.Index(run, "</a:solidFill>") {
			t.Errorf("런 %q의 a:highlight 자리가 스키마 순서와 다르다\n%s", text, run)
		}
	}
	if at := strings.Index(xml, "<a:t>EC2는 </a:t>"); strings.Contains(xml[strings.LastIndex(xml[:at], "<a:r>"):at], "highlight") {
		t.Error("<mark> 밖의 런에 형광펜이 있다")
	}
}
