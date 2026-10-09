package testkit

import (
	"maps"
	"strings"
	"testing"
)

// 파트 사이 검사가 실제로 거르는지 못 박는다 — 변이마다 스키마는 통과하지만 PowerPoint가
// 복구를 묻거나 자리를 잃는 패키지다.
func TestCrossPartsRejectsMutations(t *testing.T) {
	parts := schemaDeckParts(t)
	if problems := CheckCrossParts(parts); len(problems) > 0 {
		t.Fatalf("변이 전 덱이 이미 위반이다:\n%s", strings.Join(problems, "\n"))
	}
	const (
		slide      = "ppt/slides/slide1.xml"
		slideRels  = "ppt/slides/_rels/slide1.xml.rels"
		master     = "ppt/slideMasters/slideMaster1.xml"
		notesRels  = "ppt/notesSlides/_rels/notesSlide1.xml.rels"
		relPattern = `<Relationship [^>]*/relationships/%s"[^>]*/>`
	)
	for _, m := range []struct {
		name, part, pattern, replace, want string
	}{
		{"도형 id 중복", slide, `<p:cNvPr id="3"`, `<p:cNvPr id="2"`, "cNvPr@id=2가 두 번"},
		{"sldId 범위", presPart, `<p:sldId id="256"`, `<p:sldId id="255"`, "sldId@id=255"},
		{"관계 종류", presPart, `(<p:sldId [^>]*r:id=")rId2"`, `${1}rId3"`, "theme 관계 rId3"},
		{"없는 관계", presPart, `(<p:sldId [^>]*r:id=")rId2"`, `${1}rId9"`, "없는 관계 rId9"},
		{"모르는 참조", slide, `<p:cNvPr id="3"`, `<p:cNvPr r:id="rId1" id="3"`, "relRefTypes"},
		{"레이아웃 없음", slideRels, strings.Replace(relPattern, "%s", "slideLayout", 1), "", "slideLayout 관계가 0개"},
		{"플레이스홀더", slide, `<p:ph type="title"/>`, `<p:ph type="body" idx="9"/>`, "idx=9(body)"},
		{"마스터의 레이아웃 목록", master, `<p:sldLayoutId [^>]*/>`, "", "sldLayoutIdLst"},
		{"마스터 id 공간", master, `(<p:sldLayoutId id=")\d+`, "${1}2147483648", "한 id 공간"},
		{"노트의 슬라이드", notesRels, strings.Replace(relPattern, "%s", "slide", 1), "", "slide 관계가 0개"},
	} {
		t.Run(m.name, func(t *testing.T) {
			mutated := maps.Clone(parts)
			mutated[m.part] = mutatePart(t, parts, m.part, m.pattern, m.replace)
			problems := CheckCrossParts(mutated)
			if len(problems) == 0 {
				t.Fatal("변이한 패키지가 파트 사이 검사를 통과했다")
			}
			if joined := strings.Join(problems, "\n"); !strings.Contains(joined, m.want) {
				t.Errorf("위반이 변이를 가리키지 않는다 (%q 없음):\n%s", m.want, joined)
			}
		})
	}
}
