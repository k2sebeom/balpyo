package pptx

import (
	"strings"
	"testing"

	"github.com/k2sebeom/balpyo/soml"
)

// clrScheme은 12자리를 스키마 순서대로 전부 채워야 한다 — 자리가 빠지면 LibreOffice가
// 복구를 묻고, 순서가 어긋나면 스키마 검증에서 벗어난다.
func TestClrSchemeFillsAllTwelveSlots(t *testing.T) {
	xml, err := clrSchemeXML(testTheme())
	if err != nil {
		t.Fatal(err)
	}
	slots := []string{"dk1", "lt1", "dk2", "lt2", "accent1", "accent2", "accent3",
		"accent4", "accent5", "accent6", "hlink", "folHlink"}
	last := -1
	for _, slot := range slots {
		i := strings.Index(xml, "<a:"+slot+">")
		if i < 0 {
			t.Errorf("clrScheme에 %s 자리가 없다", slot)
			continue
		}
		if i < last {
			t.Errorf("%s 자리가 스키마 순서를 벗어났다", slot)
		}
		last = i
	}
}

// 표에 없는 자리는 가장 가까운 토큰에서 빌린다 — Office 기본값 같은 팔레트 밖 색을 남기면
// 사용자가 테마 색 편집 UI에서 그 자리를 골랐을 때 테마와 무관한 색이 나온다.
func TestClrSchemeFallbacks(t *testing.T) {
	xml, err := clrSchemeXML(testTheme())
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct{ slot, rgb string }{
		{"lt2", "FEFEFE"},      // ← lt1 (bg)
		{"accent2", "3355FF"},  // ← accent1
		{"accent6", "3355FF"},  // ← accent1
		{"hlink", "3355FF"},    // ← accent1
		{"folHlink", "3355FF"}, // ← accent1
	}
	for _, tc := range tests {
		want := "<a:" + tc.slot + `><a:srgbClr val="` + tc.rgb + `"/></a:` + tc.slot + ">"
		if !strings.Contains(xml, want) {
			t.Errorf("clrScheme에 %q가 없다\n%s", want, xml)
		}
	}
}

// 필수 슬롯(dk1/lt1/accent1)이 표에 없으면 멈춘다 — 임의 값으로 채우면 웹과 다른 색을
// 내는 파일이 조용히 만들어진다.
func TestClrSchemeRequiresCoreSlots(t *testing.T) {
	for _, missing := range []string{"dk1", "lt1", "accent1"} {
		theme := testTheme()
		kept := theme.Colors[:0]
		for _, token := range theme.Colors {
			if token.Slot != missing {
				kept = append(kept, token)
			}
		}
		theme.Colors = kept
		if _, err := clrSchemeXML(theme); err == nil {
			t.Errorf("%s 슬롯 없는 테마가 통과했다", missing)
		}
	}
}

// 슬롯 없는 토큰과 겹치는 슬롯은 에러다 — 슬롯 없는 색은 테마 편집을 따라가지 못하고
// (soml.ColorToken의 계약), 겹치면 어느 색이 이기는지 정할 수 없다.
func TestClrSchemeRejectsBadTokens(t *testing.T) {
	noSlot := testTheme()
	noSlot.Colors = append(noSlot.Colors, soml.ColorToken{Name: "extra", RGB: "000000"})
	if _, err := clrSchemeXML(noSlot); err == nil {
		t.Error("슬롯 없는 토큰이 통과했다")
	}

	dup := testTheme()
	dup.Colors = append(dup.Colors, soml.ColorToken{Name: "fg2", RGB: "222222", Slot: "dk1", Ref: "tx1"})
	if _, err := clrSchemeXML(dup); err == nil {
		t.Error("겹치는 슬롯이 통과했다")
	}
}

// fontScheme의 major/minor는 rPr의 +mj-lt/+mn-lt 참조가 해석되는 곳이다 — 슬롯이 비면
// 참조가 허공을 가리킨다.
func TestFontSchemeFromTokenTable(t *testing.T) {
	xml, err := fontSchemeXML(testTheme())
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"<a:majorFont>", "<a:minorFont>",
		`<a:latin typeface="Test Sans"/>`,
		`<a:ea typeface="Test CJK"/>`,
	} {
		if !strings.Contains(xml, want) {
			t.Errorf("fontScheme에 %q가 없다", want)
		}
	}
	// mono는 fontScheme에 자리가 없다 — rPr에 리터럴로 실린다 (shape.go의 typefaces).
	if strings.Contains(xml, "Test Mono") {
		t.Error("mono 토큰이 fontScheme에 실렸다")
	}

	missing := testTheme()
	missing.Fonts = missing.Fonts[:1] // minor가 없다
	if _, err := fontSchemeXML(missing); err == nil {
		t.Error("minor 슬롯 없는 폰트 표가 통과했다")
	}
}
