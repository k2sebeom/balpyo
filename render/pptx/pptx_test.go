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

// testTheme는 값이 core의 최소 테마와 **다른** 토큰 표다 — 산출 XML의 색·폰트가
// 하드코딩이 아니라 이 표에서 나왔다는 것을 값으로 증명한다.
func testTheme() soml.Theme {
	return soml.Theme{
		Name: "test",
		Colors: []soml.ColorToken{
			{Name: "fg", RGB: "111111", Slot: "dk1", Ref: "tx1"},
			{Name: "bg", RGB: "FEFEFE", Slot: "lt1", Ref: "bg1"},
			{Name: "muted", RGB: "667788", Slot: "dk2", Ref: "tx2"},
			{Name: "accent1", RGB: "3355FF", Slot: "accent1", Ref: "accent1"},
		},
		Fonts: []soml.FontToken{
			{Name: soml.FontHeading, Latin: "Test Sans", EastAsian: "Test CJK", Scheme: "major"},
			{Name: soml.FontBody, Latin: "Test Sans", EastAsian: "Test CJK", Scheme: "minor"},
			{Name: soml.FontMono, Latin: "Test Mono", EastAsian: "Test CJK"},
		},
	}
}

func testDeck() *soml.Deck {
	stage, _ := soml.Aspect16x9.Stage()
	return &soml.Deck{
		Title:  "2026 아키텍처 리뷰",
		Author: "이세범",
		Aspect: soml.Aspect16x9,
		Stage:  stage,
		Theme:  testTheme(),
		Slides: []*soml.Slide{
			{Index: 0, ID: "title", Source: "slides/001_title.slide"},
			{Index: 1, ID: "agenda", Source: "slides/002_agenda.slide"},
		},
	}
}

// renderPackage는 덱을 렌더하고 zip 안의 파트를 이름 → 내용으로 돌려준다.
func renderPackage(t *testing.T, r *Renderer, deck *soml.Deck) (map[string]string, []string) {
	t.Helper()
	files, err := r.Render(deck)
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

		parts, _ := renderPackage(t, New(), deck)
		if !strings.Contains(parts[presentationPart], tc.want) {
			t.Errorf("%s: %q가 없다", tc.aspect, tc.want)
		}
	}
}

// 테마 파트의 clrScheme과 마스터의 clrMap이 있어야 schemeClr 참조가 살아난다 —
// "PPTX 산출물이 테마 편집을 따라간다"는 제품 계약의 뿌리다. 값은 덱의 토큰 표에서
// 나와야 한다: 하드코딩이 남아 있으면 `color="accent1"` 텍스트가 웹과 PPTX에서 다른
// 색으로 나온다.
func TestRenderThemeAndColorMap(t *testing.T) {
	parts, _ := renderPackage(t, New(), testDeck())
	theme := parts[themePart]
	for _, want := range []string{
		"<a:clrScheme", "<a:fontScheme", "<a:fmtScheme",
		// 표의 슬롯 그대로.
		`<a:dk1><a:srgbClr val="111111"/></a:dk1>`,
		`<a:lt1><a:srgbClr val="FEFEFE"/></a:lt1>`,
		`<a:dk2><a:srgbClr val="667788"/></a:dk2>`,
		`<a:accent1><a:srgbClr val="3355FF"/></a:accent1>`,
		// 표에 없는 자리는 가장 가까운 토큰에서 빌린다 (theme.go의 clrSchemeSlots).
		`<a:lt2><a:srgbClr val="FEFEFE"/></a:lt2>`,
		`<a:accent4><a:srgbClr val="3355FF"/></a:accent4>`,
		`<a:hlink><a:srgbClr val="3355FF"/></a:hlink>`,
		// 폰트도 같은 표에서.
		`<a:latin typeface="Test Sans"/>`,
		`<a:ea typeface="Test CJK"/>`,
	} {
		if !strings.Contains(theme, want) {
			t.Errorf("theme1.xml에 %q가 없다", want)
		}
	}
	// 슬라이드 배경은 마스터의 bgRef(bg1) → clrMap → lt1 = `bg` 토큰이다 (웹 배경과 같은 토큰).
	for _, want := range []string{`<p:clrMap bg1="lt1" tx1="dk1"`, `<p:bgRef idx="1001">`, `<a:schemeClr val="bg1"/>`} {
		if !strings.Contains(parts[masterPart], want) {
			t.Errorf("slideMaster1.xml에 %q가 없다", want)
		}
	}
}

// 조립할 수 없는 덱은 멈춘다 — 테마 표가 없을 때 하드코딩 기본값으로 채우면 웹과 다른 색의
// 파일이 조용히 만들어지고, 슬라이드 0개·스테이지 없는 파일은 PowerPoint가 열지 못한다.
func TestRenderRejectsIncompleteDeck(t *testing.T) {
	noTheme, noStage := testDeck(), testDeck()
	noTheme.Theme = soml.Theme{}
	noStage.Stage = soml.Stage{}
	for _, tc := range []struct {
		name string
		deck *soml.Deck
	}{
		{"nil", nil},
		{"테마 없음", noTheme},
		{"슬라이드 0개", &soml.Deck{Stage: soml.Stage{Width: 1280, Height: 720}, Theme: testTheme()}},
		{"스테이지 없음", noStage},
	} {
		if _, err := New().Render(tc.deck); err == nil {
			t.Errorf("%s: 에러여야 한다", tc.name)
		}
	}
}

func TestRenderEscapesXML(t *testing.T) {
	deck := testDeck()
	deck.Title = `a & b <c>`
	deck.Slides[0].ID = `x"y`
	parts, _ := renderPackage(t, New(), deck)
	if !strings.Contains(parts[corePropsPart], "a &amp; b &lt;c&gt;") {
		t.Errorf("제목이 이스케이프되지 않았다: %s", parts[corePropsPart])
	}
	if !strings.Contains(parts[slidePart(1)], `name="x&quot;y"`) {
		t.Error("슬라이드 이름이 이스케이프되지 않았다")
	}
}

// 같은 덱이 같은 바이트를 낸다 (zip 타임스탬프 1980-01-01 고정 포함).
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
		t.Error("같은 덱이 다른 바이트를 냈다")
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
