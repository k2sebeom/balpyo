package pptx

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/k2sebeom/balpyo/render/geom"
	"github.com/k2sebeom/balpyo/soml"
)

// bodyRect는 이 파일의 테스트가 공유하는 rect다 — 런 분할 테스트의 관심은 txBody 안이라
// 좌표는 아무 값이어도 된다 (좌표 단정은 shape_test.go의 몫).
var bodyRect = geom.Rect{X: 64, Y: 64, W: 1152, H: 36}

// singleShapeXML은 요소 하나짜리 슬라이드의 slide1.xml이다.
func singleShapeXML(t *testing.T, src string) string {
	t.Helper()
	deck := resolvedDeck(t, src)
	g := measuredGeometry(t, deck, bodyRect)
	return slideXML(t, deck, g)
}

// 인라인 서식이 든 블록은 서식 경계마다 런이 끊긴다 (PRD §9.4 — 인라인 서식 → 텍스트 런
// rPr). 경계 공백은 앞 런의 끝에 남는다 — 파서의 인라인 경계 공백 계약(trimEdgeSpace가
// 양 끝만 벗긴다)이 그대로 흐른다.
func TestRunSplitting(t *testing.T) {
	xml := singleShapeXML(t, `<slide>
  <p>배포 파이프라인 <strong>자동화</strong></p>
</slide>`)

	if got := strings.Count(xml, "<a:r>"); got != 2 {
		t.Fatalf("런 %d개, want 2\n%s", got, xml)
	}
	// 경계 공백이 앞 런의 끝에 남는다.
	if !strings.Contains(xml, "<a:t>배포 파이프라인 </a:t>") {
		t.Errorf("첫 런이 경계 공백을 잃었다\n%s", xml)
	}
	if !strings.Contains(xml, "<a:t>자동화</a:t>") {
		t.Errorf("strong 런이 없다\n%s", xml)
	}
	// 블록 직속 문자 데이터의 런은 기존 단일 런과 같은 rPr다 (i/u 없음 — 골든 불변의 근거,
	// 웹의 textStyleCSS/runStyleCSS 분리와 같은 결정).
	if !strings.Contains(xml, `<a:rPr sz="1800" b="0">`) {
		t.Errorf("블록 직속 런의 rPr가 기존 모양이 아니다\n%s", xml)
	}
	// 인라인 런은 i/u까지 전부 명시한다 — 뷰어 기본값이 런의 모양을 정할 자리를 없앤다
	// (웹 runStyleCSS의 font-style/text-decoration 항상 명시와 같은 근거).
	if !strings.Contains(xml, `<a:rPr sz="1800" b="1" i="0" u="none">`) {
		t.Errorf("strong 런의 rPr가 완전 명시가 아니다\n%s", xml)
	}
	// 런 순서 = 글자 순서.
	if strings.Index(xml, "배포 파이프라인 ") > strings.Index(xml, "<a:t>자동화</a:t>") {
		t.Error("런 순서가 문서 순서가 아니다")
	}
}

// 중첩 인라인은 확정 스타일이 누적된 런 하나다 — 잎을 감싼 가장 안쪽 요소의 TextStyle이
// 곧 런 서식이다 (core.resolveInlines의 상속 = 확정 스타일 복사).
func TestRunNestedInlineAccumulates(t *testing.T) {
	xml := singleShapeXML(t, `<slide>
  <p><strong><em>강조</em></strong></p>
</slide>`)

	if got := strings.Count(xml, "<a:r>"); got != 1 {
		t.Fatalf("런 %d개, want 1\n%s", got, xml)
	}
	if !strings.Contains(xml, `<a:rPr sz="1800" b="1" i="1" u="none">`) {
		t.Errorf("중첩 런의 rPr에 b+i가 없다\n%s", xml)
	}
}

// span의 토큰 색은 런에서도 schemeClr 참조로 남는다 — 토큰 보존 계약이 런 단위까지 간다.
func TestRunSpanColorKeepsToken(t *testing.T) {
	xml := singleShapeXML(t, `<slide>
  <p>지연 <span color="accent1">-38%</span></p>
</slide>`)

	strong := strings.Index(xml, `<a:rPr sz="1800" b="0" i="0" u="none">`)
	if strong < 0 {
		t.Fatalf("span 런의 rPr가 없다\n%s", xml)
	}
	if !strings.Contains(xml[strong:], `<a:schemeClr val="accent1"/>`) {
		t.Errorf("span 런의 색이 schemeClr 참조가 아니다\n%s", xml)
	}
}

// code 런은 부모 크기를 유지한 채 mono 리터럴 typeface다 — fontScheme 슬롯이 없는 유일한
// 예외 (typefaces의 기존 경로가 런에서도 그대로 쓰인다).
func TestRunCodeUsesLiteralMonoTypeface(t *testing.T) {
	xml := singleShapeXML(t, `<slide>
  <p>실행: <code>balpyo build</code></p>
</slide>`)

	code := strings.Index(xml, "<a:t>balpyo build</a:t>")
	if code < 0 {
		t.Fatalf("code 런이 없다\n%s", xml)
	}
	run := xml[:code]
	run = run[strings.LastIndex(run, "<a:r>"):]
	for _, want := range []string{
		`sz="1800"`, // 부모(body 24px) 크기 유지 — code 역할(20px)을 끌어오지 않는다.
		`<a:latin typeface="Noto Sans Mono"/>`,
	} {
		if !strings.Contains(run, want) {
			t.Errorf("code 런에 %q가 없다\n%s", want, run)
		}
	}
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

// 링크가 든 덱도 같은 입력이 같은 바이트를 내야 한다 — rel id 배정에 맵 순회가 새면
// 여기서 잡힌다.
func TestRunHyperlinkDeckIsByteIdentical(t *testing.T) {
	render := func() []byte {
		deck := resolvedDeck(t, `<slide>
  <p><a href="https://a.example">가</a> <a href="https://b.example">나</a></p>
</slide>`)
		g := measuredGeometry(t, deck, bodyRect)
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

// 웹과 PPTX의 링크 색이 같은 표의 같은 값에서 나온다 — core의 accent1 델타와 테마 파트의
// hlink fallback(accent1)이 짝이라는 관례를 산출물로 고정한다 (NOTES 02/04의 열린 질문).
func TestRunHyperlinkColorMatchesThemeHlinkSlot(t *testing.T) {
	deck := resolvedDeck(t, `<slide>
  <p><a href="https://example.com">링크</a></p>
</slide>`)
	g := measuredGeometry(t, deck, bodyRect)
	parts, _ := renderPackage(t, NewWithGeometry(g), deck)

	accent1, ok := deck.Theme.Color("accent1")
	if !ok {
		t.Fatal("테마에 accent1 토큰이 없다")
	}
	// 런은 accent1 참조, 테마의 hlink 슬롯은 accent1의 RGB — 뷰어가 hlink 자동 색을
	// 입히더라도 같은 값이다.
	if !strings.Contains(parts[slidePart(1)], `<a:schemeClr val="accent1"/>`) {
		t.Error("링크 런이 accent1 참조가 아니다")
	}
	if want := `<a:hlink><a:srgbClr val="` + accent1.RGB + `"/></a:hlink>`; !strings.Contains(parts[themePart], want) {
		t.Errorf("테마 hlink 슬롯이 accent1 값이 아니다 (want %q)", want)
	}
}

// <br/>은 a:br이다 — 문단을 쪼개지 않는다 (문단 사이 간격이 줄간격과 갈라질 수 있다).
func TestRunBreakElement(t *testing.T) {
	xml := singleShapeXML(t, `<slide>
  <p>첫 줄<br/>둘째 줄</p>
</slide>`)

	if got := strings.Count(xml, "<a:br/>"); got != 1 {
		t.Fatalf("a:br %d개, want 1\n%s", got, xml)
	}
	if got := strings.Count(xml, "<a:p>"); got != 1 {
		t.Errorf("문단 %d개, want 1 — br이 문단을 쪼갰다\n%s", got, xml)
	}
	// br이 두 런 사이에 있다.
	if !(strings.Index(xml, "<a:t>첫 줄</a:t>") < strings.Index(xml, "<a:br/>") &&
		strings.Index(xml, "<a:br/>") < strings.Index(xml, "<a:t>둘째 줄</a:t>")) {
		t.Errorf("a:br의 위치가 런 사이가 아니다\n%s", xml)
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

// 블록 태그 하나 = p:sp 하나 — writeShapeNode가 <text>만 보던 분기를 블록 종류 카탈로그로
// 넓힌다. 좌표·외관은 <text>와 같은 경로다.
func TestRenderBlockTagShapes(t *testing.T) {
	deck := resolvedDeck(t, `<slide>
  <h2>회고</h2>
  <blockquote>측정하지 않으면 개선할 수 없다</blockquote>
</slide>`)
	g := measuredGeometry(t, deck,
		geom.Rect{X: 64, Y: 64, W: 1152, H: 46.8},
		geom.Rect{X: 64, Y: 134.8, W: 1152, H: 36},
	)
	xml := slideXML(t, deck, g)

	if got := strings.Count(xml, "<p:sp>"); got != 2 {
		t.Fatalf("p:sp %d개, want 2\n%s", got, xml)
	}
	for _, want := range []string{
		`<p:cNvPr id="2" name="h2 s0.0"/>`,
		`<p:cNvPr id="3" name="blockquote s0.1"/>`,
		// h2 = heading 역할: 36px → sz 2700, 굵게, major 폰트.
		`<a:rPr sz="2700" b="1">`,
		// blockquote = body 역할 + muted 색 (테마의 태그 기본값).
		`<a:schemeClr val="tx2"/>`,
	} {
		if !strings.Contains(xml, want) {
			t.Errorf("slide1.xml에 %q가 없다", want)
		}
	}
}

// 외관이 해석되지 않은 인라인은 기본값으로 그리지 않는다 (웹 writeInline과 같은 계약).
func TestRunUnresolvedInlineFails(t *testing.T) {
	deck := resolvedDeck(t, `<slide>
  <p><strong>강조</strong></p>
</slide>`)
	block := deck.Slides[0].Content[0].(*soml.Element)
	block.Children[0].(*soml.Element).Appearance = nil
	g := measuredGeometry(t, deck, bodyRect)

	_, err := NewWithGeometry(g).Render(deck)
	if err == nil {
		t.Fatal("해석되지 않은 인라인이 렌더되었다")
	}
	if !strings.Contains(err.Error(), "core.Resolve") {
		t.Errorf("에러가 해석 누락을 가리키지 않는다: %v", err)
	}
}

// Link 없는 <a>는 렌더 에러다 — core의 href 검증을 지나지 않은 값이 산출물에 실릴 길을
// 막는다 (웹과 같은 계약).
func TestRunLinklessAnchorFails(t *testing.T) {
	deck := resolvedDeck(t, `<slide>
  <p><a href="https://example.com">링크</a></p>
</slide>`)
	block := deck.Slides[0].Content[0].(*soml.Element)
	block.Children[0].(*soml.Element).Appearance.Link = ""
	g := measuredGeometry(t, deck, bodyRect)

	_, err := NewWithGeometry(g).Render(deck)
	if err == nil {
		t.Fatal("링크 없는 <a>가 렌더되었다")
	}
	if !strings.Contains(err.Error(), "core.Resolve") {
		t.Errorf("에러가 해석 누락을 가리키지 않는다: %v", err)
	}
}

// 서식 없는 텍스트의 XML은 런 분할 전과 같아야 한다 — 01_text 골든 불변의 단위 테스트 짝.
// (골든 자체는 internal/testkit의 픽스처 테스트가 지킨다.)
func TestRunPlainTextUnchanged(t *testing.T) {
	xml := singleShapeXML(t, `<slide>
  <text>본문</text>
</slide>`)
	want := strings.Join([]string{
		"            <a:r>",
		`              <a:rPr sz="1800" b="0">`,
		"                <a:solidFill>",
		`                  <a:schemeClr val="tx1"/>`,
		"                </a:solidFill>",
		`                <a:latin typeface="+mn-lt"/>`,
		`                <a:ea typeface="+mn-ea"/>`,
		"              </a:rPr>",
		"              <a:t>본문</a:t>",
		"            </a:r>",
	}, "\n")
	if !strings.Contains(xml, want) {
		t.Errorf("서식 없는 런의 XML이 기존 모양이 아니다\nwant:\n%s\ngot:\n%s", want, xml)
	}
}

// 지오메트리 rect가 요소 순서와 맞는지 이 파일의 헬퍼가 의존하는 전제를 확인한다 —
// 인라인은 측정 대상이 아니므로 rect는 블록 하나면 된다.
func TestRunFixtureGeometryAssumption(t *testing.T) {
	deck := resolvedDeck(t, `<slide>
  <p>본문 <strong>강조</strong></p>
</slide>`)
	refs := geom.Refs(deck)
	if len(refs) != 1 {
		refIDs := make([]string, len(refs))
		for i, r := range refs {
			refIDs[i] = fmt.Sprintf("%s(%s)", r.ID, r.El)
		}
		t.Fatalf("측정 대상 %d개, want 1 (블록만): %s", len(refs), strings.Join(refIDs, ", "))
	}
}
