package pptx

import (
	"strings"
	"testing"

	"github.com/k2sebeom/balpyo/render/geom"
)

// notesSources는 픽스처 01_text와 같은 배치다 — 슬라이드 1·3에만 노트가 있고 2에는 없다.
// 노트만 있는 슬라이드는 좌표가 필요한 요소가 없으므로 지오메트리 없이 렌더된다
// (TestRenderNotesNeedNoGeometry가 못 박는다).
func notesSources() []string {
	return []string{
		`<slide><note>인사 30초. 청중에 인프라팀 포함.</note></slide>`,
		`<slide></slide>`,
		`<slide><note>질문이 없으면 로드맵으로 되돌아간다.</note></slide>`,
	}
}

// 노트 파트·관계·Content_Types가 전부 있고, 노트 없는 슬라이드(2번)는 아무것도 갖지 않는다.
func TestRenderNotesParts(t *testing.T) {
	deck := resolvedDeck(t, notesSources()...)
	parts, _ := renderPackage(t, New(), deck)

	for _, want := range []string{
		notesMasterPart, notesMasterRelsPart,
		notesSlidePart(1), notesSlideRelsPart(1),
		notesSlidePart(3), notesSlideRelsPart(3),
	} {
		if _, ok := parts[want]; !ok {
			t.Errorf("파트 %q가 없다", want)
		}
	}
	// 노트 없는 슬라이드는 노트 파트도 관계도 갖지 않는다.
	if _, ok := parts[notesSlidePart(2)]; ok {
		t.Error("노트 없는 슬라이드 2에 노트 파트가 있으면 안 된다")
	}
	if strings.Contains(parts[slideRelsPart(2)], "notesSlide") {
		t.Error("노트 없는 슬라이드 2의 rels에 notesSlide 관계가 있으면 안 된다")
	}

	// Content_Types Override — 빠지면 PowerPoint가 복구를 건다.
	ct := parts[contentTypesPart]
	for _, want := range []string{
		`<Override PartName="/ppt/notesMasters/notesMaster1.xml" ContentType="application/vnd.openxmlformats-officedocument.presentationml.notesMaster+xml"/>`,
		`<Override PartName="/ppt/notesSlides/notesSlide1.xml" ContentType="application/vnd.openxmlformats-officedocument.presentationml.notesSlide+xml"/>`,
		`<Override PartName="/ppt/notesSlides/notesSlide3.xml" ContentType="application/vnd.openxmlformats-officedocument.presentationml.notesSlide+xml"/>`,
	} {
		if !strings.Contains(ct, want) {
			t.Errorf("[Content_Types].xml에 %q가 없다", want)
		}
	}
	if strings.Contains(ct, "notesSlide2.xml") {
		t.Error("[Content_Types].xml에 없는 파트 notesSlide2의 Override가 있다")
	}
}

// 노트 마스터의 관계 ID는 기존 규칙(rId1=마스터, rId2..=슬라이드, 그다음=테마)의 어느
// 번호도 밀지 않고 맨 뒤에 붙는다 — 밀리면 슬라이드 관계가 엉뚱한 파트를 가리킨다.
func TestRenderNotesMasterRelAppendsAfterTheme(t *testing.T) {
	deck := resolvedDeck(t, notesSources()...)
	parts, _ := renderPackage(t, New(), deck)

	if !strings.Contains(parts[presentationPart], `<p:notesMasterId r:id="rId6"/>`) {
		t.Errorf("presentation.xml에 notesMasterIdLst(rId6)가 없다\n%s", parts[presentationPart])
	}
	presRels := parts[presRelsPart]
	// 슬라이드 3개: rId2..rId4 = 슬라이드, rId5 = 테마 (기존 규칙 그대로), rId6 = 노트 마스터.
	for _, want := range []string{
		`Id="rId5" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/theme"`,
		`Id="rId6" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/notesMaster" Target="notesMasters/notesMaster1.xml"`,
	} {
		if !strings.Contains(presRels, want) {
			t.Errorf("presentation.xml.rels에 %q가 없다\n%s", want, presRels)
		}
	}
	// 노트 마스터 → 테마 관계 (슬라이드 마스터와 같은 theme1.xml).
	if !strings.Contains(parts[notesMasterRelsPart], `Target="../theme/theme1.xml"`) {
		t.Error("노트 마스터가 테마를 가리키지 않는다")
	}
}

// 노트 파트 ↔ 슬라이드 관계가 정확한 슬라이드를 가리킨다 — 이 항목에서 가장 조용히 틀릴 수
// 있는 부분이다 (틀려도 파일은 열리고, 노트만 다른 슬라이드에 붙는다).
func TestRenderNotesSlideRelsPointAtOwnSlide(t *testing.T) {
	deck := resolvedDeck(t, notesSources()...)
	parts, _ := renderPackage(t, New(), deck)

	for _, tc := range []struct {
		rels  string
		slide string
	}{
		{notesSlideRelsPart(1), `Target="../slides/slide1.xml"`},
		{notesSlideRelsPart(3), `Target="../slides/slide3.xml"`},
	} {
		if !strings.Contains(parts[tc.rels], tc.slide) {
			t.Errorf("%s가 %q를 가리키지 않는다\n%s", tc.rels, tc.slide, parts[tc.rels])
		}
		if !strings.Contains(parts[tc.rels], `Target="../notesMasters/notesMaster1.xml"`) {
			t.Errorf("%s에 노트 마스터 관계가 없다", tc.rels)
		}
	}
	// 반대 방향: 슬라이드 rels → 자기 번호의 노트 슬라이드.
	if !strings.Contains(parts[slideRelsPart(1)], `Target="../notesSlides/notesSlide1.xml"`) {
		t.Error("슬라이드 1이 notesSlide1을 가리키지 않는다")
	}
	if !strings.Contains(parts[slideRelsPart(3)], `Target="../notesSlides/notesSlide3.xml"`) {
		t.Error("슬라이드 3이 notesSlide3을 가리키지 않는다")
	}
}

// 노트 슬라이드 본문: sldImg·body 플레이스홀더가 있고, 좌표는 마스터에서 상속받는다 —
// 플레이스홀더에 xfrm이 없어야 한다 (spTree 그룹의 xfrm 하나만 남는다).
func TestRenderNotesSlideBodyInheritsFromMaster(t *testing.T) {
	deck := resolvedDeck(t, notesSources()...)
	parts, _ := renderPackage(t, New(), deck)

	notes := parts[notesSlidePart(1)]
	for _, want := range []string{
		`<p:ph type="sldImg"/>`,
		`<p:ph type="body" idx="1"/>`,
		`<a:t>인사 30초. 청중에 인프라팀 포함.</a:t>`,
	} {
		if !strings.Contains(notes, want) {
			t.Errorf("notesSlide1.xml에 %q가 없다\n%s", want, notes)
		}
	}
	if got := strings.Count(notes, "<a:xfrm>"); got != 1 {
		t.Errorf("notesSlide1.xml의 a:xfrm이 %d개다 (spTree 그룹 1개여야 한다 — 플레이스홀더 좌표는 마스터 상속)", got)
	}
	// 마스터에는 두 플레이스홀더의 좌표가 있다 (노트 페이지 6858000×9144000, 16:9 이미지).
	master := parts[notesMasterPart]
	for _, want := range []string{
		`<a:off x="685800" y="1143000"/>`,
		`<a:ext cx="5486400" cy="3086100"/>`,
	} {
		if !strings.Contains(master, want) {
			t.Errorf("notesMaster1.xml에 %q가 없다", want)
		}
	}
}

// 노트가 하나도 없는 덱은 노트 마스터·notesMasterIdLst·Override를 전혀 내지 않는다 —
// 00_empty 골든이 이 항목 전후로 같아야 한다.
func TestRenderDeckWithoutNotesHasNoNotesParts(t *testing.T) {
	parts, _ := renderPackage(t, New(), testDeck())
	if _, ok := parts[notesMasterPart]; ok {
		t.Error("노트 없는 덱에 노트 마스터가 있다")
	}
	for name, want := range map[string]string{
		presentationPart: "notesMasterIdLst",
		presRelsPart:     "notesMaster",
		contentTypesPart: "notesSlide",
	} {
		if strings.Contains(parts[name], want) {
			t.Errorf("%s에 %q가 있으면 안 된다", name, want)
		}
	}
}

// 노트는 XML 특수문자를 담을 수 있다.
func TestRenderNotesEscapesXML(t *testing.T) {
	deck := resolvedDeck(t, `<slide><note>R&D 예산 > 작년 & "인용"</note></slide>`)
	parts, _ := renderPackage(t, New(), deck)
	want := `<a:t>R&amp;D 예산 &gt; 작년 &amp; &quot;인용&quot;</a:t>`
	if !strings.Contains(parts[notesSlidePart(1)], want) {
		t.Errorf("notesSlide1.xml에 %q가 없다\n%s", want, parts[notesSlidePart(1)])
	}
}

// 노트 원문의 개행은 파서가 공백으로 접는다 — 웹의 balpyo-notes와 PPTX 노트 슬라이드가
// **같은 문자열**을 받는다 (결정은 soml.Parse 한 곳이다). 한쪽이 개행을 되살리면 저자가
// 발표자 뷰에서 읽는 글의 줄 구조가 타깃마다 달라진다.
func TestRenderNotesNewlinesAgreeWithWebRenderer(t *testing.T) {
	deck := resolvedDeck(t, "<slide>\n  <note>인사 30초.\n  데모는 생략 가능.</note>\n</slide>")

	const collapsed = "인사 30초. 데모는 생략 가능."
	parts, _ := renderPackage(t, New(), deck)
	notes := parts[notesSlidePart(1)]
	if !strings.Contains(notes, "<a:t>"+collapsed+"</a:t>") {
		t.Errorf("PPTX 노트가 접힌 문자열을 내지 않았다\n%s", notes)
	}
	if strings.Contains(notes, "<a:br") {
		t.Error("PPTX 노트가 a:br을 냈다 — 파서가 접은 개행이 되살아났다")
	}

	html := renderWebString(t, deck)
	if !strings.Contains(html, `<div class="balpyo-notes" hidden>`+collapsed+"</div>") {
		t.Errorf("웹 노트가 접힌 문자열을 내지 않았다\n%s", html)
	}
}

// 노트는 지오메트리 패스의 좌표 대상이 아니다 — 노트만 있는 덱은 브라우저 없이 렌더된다.
// Refs가 노트를 세기 시작하면 checkGeometry가 지오메트리를 요구하고, 노트만 있는 덱이
// 브라우저를 띄우게 된다.
func TestRenderNotesNeedNoGeometry(t *testing.T) {
	deck := resolvedDeck(t, notesSources()...)
	if refs := geom.Refs(deck); len(refs) != 0 {
		t.Fatalf("노트만 있는 덱의 좌표 대상이 %d개다 (0이어야 한다): %v", len(refs), refs)
	}
	// New() = 지오메트리 없는 렌더러 — 성공해야 한다 (renderPackage 경유 위 테스트들이
	// 이미 쓰지만, 여기서 명시적으로 못 박는다).
	if _, err := New().Render(deck); err != nil {
		t.Fatalf("노트만 있는 덱이 지오메트리 없이 렌더되지 않는다: %v", err)
	}
}
