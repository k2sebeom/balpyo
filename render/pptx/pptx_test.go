package pptx

import (
	"archive/zip"
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/k2sebeom/balpyo/render"
	"github.com/k2sebeom/balpyo/soml"
)

func testDeck() *soml.Deck {
	stage, _ := soml.Aspect16x9.Stage()
	return &soml.Deck{
		Title:  "2026 아키텍처 리뷰",
		Author: "이세범",
		Aspect: soml.Aspect16x9,
		Stage:  stage,
		Slides: []*soml.Slide{
			{Index: 0, ID: "title", Source: "slides/001_title.slide"},
			{Index: 1, ID: "agenda", Source: "slides/002_agenda.slide"},
		},
	}
}

// renderPackage는 덱을 렌더하고 zip 안의 파트를 이름 → 내용으로 돌려준다.
func renderPackage(t *testing.T, deck *soml.Deck) (map[string]string, []string) {
	t.Helper()
	files, err := New().Render(deck)
	if err != nil {
		t.Fatal(err)
	}
	name := FileName(deck)
	data, ok := files[name]
	if !ok {
		t.Fatalf("%s가 산출물에 없다: %v", name, files.Paths())
	}
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatalf("zip으로 읽을 수 없다: %v", err)
	}
	parts := make(map[string]string, len(zr.File))
	order := make([]string, 0, len(zr.File))
	for _, f := range zr.File {
		rc, err := f.Open()
		if err != nil {
			t.Fatal(err)
		}
		body, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			t.Fatal(err)
		}
		parts[f.Name] = string(body)
		order = append(order, f.Name)
	}
	return parts, order
}

func TestRenderHasRequiredParts(t *testing.T) {
	parts, order := renderPackage(t, testDeck())

	// [Content_Types].xml은 첫 엔트리여야 한다.
	if order[0] != contentTypesPart {
		t.Errorf("첫 엔트리 = %q, want %q", order[0], contentTypesPart)
	}
	for _, want := range []string{
		contentTypesPart, rootRelsPart, presentationPart, presRelsPart,
		masterPart, masterRelsPart, layoutPart, layoutRelsPart, themePart,
		corePropsPart, appPropsPart,
		slidePart(1), slideRelsPart(1), slidePart(2), slideRelsPart(2),
	} {
		if _, ok := parts[want]; !ok {
			t.Errorf("파트 %q가 없다", want)
		}
	}
	// 슬라이드는 덱의 슬라이드 수만큼만 있어야 한다.
	if _, ok := parts[slidePart(3)]; ok {
		t.Error("슬라이드 3이 있으면 안 된다")
	}
}

// 슬라이드 크기는 PowerPoint 표준 페이지 크기와 정확히 일치해야 한다.
func TestRenderSlideSizeMatchesAspect(t *testing.T) {
	tests := []struct {
		aspect soml.Aspect
		want   string
	}{
		{soml.Aspect16x9, `<p:sldSz cx="12192000" cy="6858000"/>`}, // 13.333×7.5in
		{soml.Aspect4x3, `<p:sldSz cx="9144000" cy="6858000"/>`},   // 10×7.5in
	}
	for _, tc := range tests {
		deck := testDeck()
		deck.Aspect = tc.aspect
		stage, err := tc.aspect.Stage()
		if err != nil {
			t.Fatal(err)
		}
		deck.Stage = stage

		parts, _ := renderPackage(t, deck)
		if !strings.Contains(parts[presentationPart], tc.want) {
			t.Errorf("%s: %q가 없다", tc.aspect, tc.want)
		}
	}
}

// 관계 파일이 실제 존재하는 파트를 가리키는지 — 여기가 깨지면 PowerPoint가 복구를 묻는다.
func TestRenderRelationshipsResolve(t *testing.T) {
	parts, _ := renderPackage(t, testDeck())

	presRels := parts[presRelsPart]
	for _, want := range []string{
		`Target="slideMasters/slideMaster1.xml"`,
		`Target="slides/slide1.xml"`,
		`Target="slides/slide2.xml"`,
		`Target="theme/theme1.xml"`,
	} {
		if !strings.Contains(presRels, want) {
			t.Errorf("presentation.xml.rels에 %q가 없다", want)
		}
	}
	// sldIdLst의 r:id가 rels의 Id와 짝이 맞아야 한다.
	for _, want := range []string{`r:id="rId2"`, `r:id="rId3"`} {
		if !strings.Contains(parts[presentationPart], want) {
			t.Errorf("presentation.xml에 %q가 없다", want)
		}
	}
	if !strings.Contains(presRels, `Id="rId4"`) {
		t.Error("테마 관계 ID가 슬라이드 뒤 번호를 이어받아야 한다")
	}
	if !strings.Contains(parts[slideRelsPart(1)], `Target="../slideLayouts/slideLayout1.xml"`) {
		t.Error("슬라이드가 레이아웃을 가리켜야 한다")
	}
	if !strings.Contains(parts[masterRelsPart], `Target="../slideLayouts/slideLayout1.xml"`) {
		t.Error("마스터가 레이아웃을 가리켜야 한다")
	}
}

// 슬라이드 ID는 256부터 시작하고 겹치지 않아야 한다.
func TestRenderSlideIDsStartAt256(t *testing.T) {
	parts, _ := renderPackage(t, testDeck())
	for _, want := range []string{`id="256"`, `id="257"`} {
		if !strings.Contains(parts[presentationPart], want) {
			t.Errorf("presentation.xml에 %q가 없다", want)
		}
	}
}

// 테마 파트의 clrScheme과 마스터의 clrMap이 있어야 schemeClr 참조가 살아난다 —
// "PPTX 산출물이 테마 편집을 따라간다"는 제품 계약의 뿌리다.
func TestRenderThemeAndColorMap(t *testing.T) {
	parts, _ := renderPackage(t, testDeck())
	theme := parts[themePart]
	for _, want := range []string{
		"<a:clrScheme", "<a:fontScheme", "<a:fmtScheme",
		`<a:accent1><a:srgbClr val="4472C4"/></a:accent1>`,
		`<a:latin typeface="Noto Sans"/>`,
	} {
		if !strings.Contains(theme, want) {
			t.Errorf("theme1.xml에 %q가 없다", want)
		}
	}
	if !strings.Contains(parts[masterPart], `<p:clrMap bg1="lt1" tx1="dk1"`) {
		t.Error("마스터에 clrMap이 없다")
	}
}

func TestRenderSlideNamesUseSlideID(t *testing.T) {
	parts, _ := renderPackage(t, testDeck())
	if !strings.Contains(parts[slidePart(1)], `<p:cSld name="title">`) {
		t.Error("슬라이드 이름이 슬라이드 ID여야 한다")
	}
	// Iteration 0의 spTree는 비어 있다 — 도형은 Iteration 1부터.
	if strings.Contains(parts[slidePart(1)], "<p:sp>") {
		t.Error("Iteration 0에는 도형이 없어야 한다")
	}
}

func TestRenderEscapesXML(t *testing.T) {
	deck := testDeck()
	deck.Title = `a & b <c>`
	deck.Slides[0].ID = `x"y`
	parts, _ := renderPackage(t, deck)
	if !strings.Contains(parts[corePropsPart], "a &amp; b &lt;c&gt;") {
		t.Errorf("제목이 이스케이프되지 않았다: %s", parts[corePropsPart])
	}
	if !strings.Contains(parts[slidePart(1)], `name="x&quot;y"`) {
		t.Error("슬라이드 이름이 이스케이프되지 않았다")
	}
}

// 골든 비교가 성립하려면 같은 덱이 같은 바이트를 내야 한다 (zip 타임스탬프 포함).
func TestRenderIsByteIdentical(t *testing.T) {
	first, err := New().Render(testDeck())
	if err != nil {
		t.Fatal(err)
	}
	second, err := New().Render(testDeck())
	if err != nil {
		t.Fatal(err)
	}
	name := FileName(testDeck())
	if !bytes.Equal(first[name], second[name]) {
		t.Error("같은 덱이 다른 바이트를 냈다 — 골든 비교가 불가능해진다")
	}
}

func TestRenderRejectsEmptyDeck(t *testing.T) {
	if _, err := New().Render(nil); err == nil {
		t.Error("nil 덱은 에러여야 한다")
	}
	// 슬라이드가 없는 pptx는 PowerPoint가 열지 못한다.
	if _, err := New().Render(&soml.Deck{Stage: soml.Stage{Width: 1280, Height: 720}}); err == nil {
		t.Error("슬라이드 0개는 에러여야 한다")
	}
	// 스테이지가 없으면 페이지 크기를 정할 수 없다.
	deck := testDeck()
	deck.Stage = soml.Stage{}
	if _, err := New().Render(deck); err == nil {
		t.Error("스테이지 없는 덱은 에러여야 한다")
	}
}

func TestFileName(t *testing.T) {
	tests := []struct{ title, want string }{
		{"my talk", "my-talk.pptx"},
		{"2026 아키텍처 리뷰", "2026-아키텍처-리뷰.pptx"},
		{"", "deck.pptx"},
		{"///", "deck.pptx"},
	}
	for _, tc := range tests {
		if got := FileName(&soml.Deck{Title: tc.title}); got != tc.want {
			t.Errorf("%q → %q, want %q", tc.title, got, tc.want)
		}
	}
}

func TestRendererTarget(t *testing.T) {
	if got := New().Target(); got != render.TargetPPTX {
		t.Errorf("Target() = %q", got)
	}
}
