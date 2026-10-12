package pptx

import (
	"strings"
	"testing"

	"github.com/k2sebeom/balpyo/render/geom"
	"github.com/k2sebeom/balpyo/soml"
)

// 덱 언어는 OOXML 문화권 이름이 된다 — 지역 없는 코드는 기본 지역을 받고, 작성자가 고른 지역은
// 대소문자만 고쳐 그대로 둔다.
func TestCultureTag(t *testing.T) {
	for _, tc := range []struct{ in, want string }{
		{"", ""},
		{"ko", "ko-KR"},
		{"en", "en-US"},
		{"KO", "ko-KR"},
		{"ja-JP", "ja-JP"},
		{"en-gb", "en-GB"},
		{"pt_PT", "pt-PT"},
		{"zh-hant-tw", "zh-Hant-TW"},
		{"es-419", "es-419"},
		{"eo", "eo"},
	} {
		if got := cultureTag(tc.in); got != tc.want {
			t.Errorf("cultureTag(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

// 한국어 덱은 텍스트 상속의 뿌리(글상자 · 플레이스홀더 · 노트)와 문서 속성 모두에 ko-KR을 싣는다.
// 하나라도 빠지면 그 자리의 글자는 뷰어의 편집 언어로 맞춤법 검사를 받는다.
func TestRenderCarriesDeckLanguage(t *testing.T) {
	deck := layoutSlotDeck(t, soml.LayoutTitle, `<h1 slot="title">발표</h1>`)
	deck.Slides[0].Notes = "인사 30초."
	g := measuredGeometry(t, deck, geom.Rect{X: 64, Y: 64, W: 1152, H: 36})
	parts, _ := renderPackage(t, NewWithGeometry(g), deck)

	const lang = `<a:defPPr><a:defRPr lang="ko-KR"/></a:defPPr>`
	for _, tc := range []struct {
		part, scope string
	}{
		{presentationPart, "<p:defaultTextStyle>"},
		{masterPart, "<p:titleStyle>"},
		{masterPart, "<p:bodyStyle>"},
		{masterPart, "<p:otherStyle>"},
		{notesMasterPart, "<p:notesStyle>"},
	} {
		xml := parts[tc.part]
		i := strings.Index(xml, tc.scope)
		if i < 0 || !strings.HasPrefix(strings.TrimSpace(xml[i+len(tc.scope):]), lang) {
			t.Errorf("%s의 %s가 덱 언어로 시작하지 않는다\n%s", tc.part, tc.scope, xml)
		}
	}
	if !strings.Contains(parts[corePropsPart], "<dc:language>ko-KR</dc:language>") {
		t.Errorf("core.xml에 dc:language가 없다\n%s", parts[corePropsPart])
	}
}

// 언어가 비어 있는 덱은 어느 파트에도 `lang`을 싣지 않는다 — 빈 `lang=""`은 뷰어마다 다르게 읽힌다.
func TestRenderWithoutLanguageOmitsLang(t *testing.T) {
	deck := resolvedDeck(t, `<slide><text>본문</text><note>노트</note></slide>`)
	g := measuredGeometry(t, deck, geom.Rect{X: 64, Y: 64, W: 1152, H: 36})
	parts, _ := renderPackage(t, NewWithGeometry(g), deck)
	for name, xml := range parts {
		if strings.Contains(xml, `lang=`) || strings.Contains(xml, "dc:language") {
			t.Errorf("언어 없는 덱의 %s에 언어가 실렸다", name)
		}
	}
}
