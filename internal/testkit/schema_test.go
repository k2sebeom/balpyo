package testkit

import (
	"regexp"
	"strings"
	"testing"

	"github.com/k2sebeom/balpyo/core"
	"github.com/k2sebeom/balpyo/render/geom"
	"github.com/k2sebeom/balpyo/render/pptx"
	"github.com/k2sebeom/balpyo/soml"
)

// schemaDeckParts는 브라우저 없이 렌더한 작은 덱의 파트 표다 — 텍스트 · 리스트 · 프리셋 도형 · 표 · 노트.
// 변이 테스트의 관심은 좌표가 아니라 XML의 모양이므로 합성 지오메트리로 충분하다.
func schemaDeckParts(t *testing.T) map[string][]byte {
	t.Helper()
	stage, err := soml.Aspect16x9.Stage()
	if err != nil {
		t.Fatal(err)
	}
	doc, diags := soml.Parse("slides/001.slide", `<slide>
  <h1>스키마 검증</h1>
  <ul><li>첫째 <strong>항목</strong></li></ul>
  <shape type="rect" fill="accent1" w="120" h="80"/>
  <table><tr><th>열</th></tr><tr><td>값</td></tr></table>
  <note>노트</note>
</slide>`)
	if err := diags.Err(); err != nil {
		t.Fatal(err)
	}
	deck := &soml.Deck{Title: "schema", Aspect: soml.Aspect16x9, Stage: stage}
	deck.Slides = []*soml.Slide{{ID: "s1", Source: "slides/001.slide",
		Notes: doc.Notes, Content: doc.Content, Tokens: doc.Tokens}}
	if err := core.Resolve(deck, nil).Err(); err != nil {
		t.Fatal(err)
	}
	files, err := pptx.NewWithGeometry(geom.Synthetic(deck, geom.Rect{X: 64, Y: 64, W: 1152, H: 64})).Render(deck)
	if err != nil {
		t.Fatal(err)
	}
	return AssertValidPPTX(t, files[pptx.FileName(deck)])
}

// 검증기가 실제로 거르는지 못 박는다 (ADR 0065) — 판정이 조용히 느슨해지면(스키마 적재가 lax로
// 떨어지거나 의존성이 바뀌면) 렌더러의 결함보다 이 테스트가 먼저 빨개진다. 변이 하나가 PowerPoint가
// 복구를 묻는 흔한 원인 하나다.
func TestSchemaRejectsMutatedParts(t *testing.T) {
	parts := schemaDeckParts(t)
	const slide = "ppt/slides/slide1.xml"
	for _, m := range []struct {
		name, part, pattern, replace, want string
	}{
		{"자식 순서", slide, `(<a:off [^>]*/>)(\s*)(<a:ext [^>]*/>)`, "$3$2$1", "is not permitted here"},
		{"필수 속성", slide, `(<p:cNvPr id="2") name="[^"]*"`, "$1", "required attribute name is missing"},
		{"열거값", slide, `algn="l"`, `algn="middle"`, "ST_TextAlignType"},
		{"프리셋 이름", slide, `prst="rect"`, `prst="rectt"`, "ST_ShapeType"},
		{"모르는 요소", slide, `<a:lstStyle/>`, `<a:lstStyle/><a:foo/>`, "foo is not permitted here"},
		{"모르는 속성", slide, `<a:bodyPr `, `<a:bodyPr foo="1" `, "attribute foo is not permitted here"},
		{"범위", slide, `<a:spcPts val="\d+"`, `<a:spcPts val="-1"`, "minInclusive"},
		{"타입", slide, `<a:off x="\d+"`, `<a:off x="abc"`, "abc"},
		{"관계 대상", "_rels/.rels", ` Target="ppt/presentation.xml"`, "", "required attribute Target is missing"},
		{"content type 속성", contentTypesPart, `<Default `, `<Default Bogus="x" `, "attribute Bogus is not permitted here"},
		{"모르는 이름공간", slide, `xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main"`,
			`xmlns:p="urn:balpyo:not-ooxml"`, "no element declaration"},
	} {
		t.Run(m.name, func(t *testing.T) {
			re := regexp.MustCompile(m.pattern)
			src := string(parts[m.part])
			if !re.MatchString(src) {
				t.Fatalf("%s에 변이할 자리 %q가 없다 — 렌더러 출력이 바뀌었으면 패턴을 고친다", m.part, m.pattern)
			}
			loc := re.FindStringIndex(src)
			mutated := src[:loc[0]] + re.ReplaceAllString(src[loc[0]:loc[1]], m.replace) + src[loc[1]:]
			err := ValidatePart(t, []byte(mutated))
			if err == nil {
				t.Fatal("변이한 파트가 스키마를 통과했다 — 검증기가 거르지 않는다")
			}
			if !strings.Contains(err.Error(), m.want) {
				t.Errorf("에러가 변이를 가리키지 않는다 (%q 없음):\n%v", m.want, err)
			}
		})
	}
}
