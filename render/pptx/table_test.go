package pptx

import (
	"encoding/xml"
	"io"
	"strconv"
	"strings"
	"testing"

	"github.com/k2sebeom/balpyo/core"
	"github.com/k2sebeom/balpyo/render/geom"
	"github.com/k2sebeom/balpyo/soml"
)

// tableDeck는 표 하나를 담은 슬라이드를 파이프라인과 같은 순서(파싱 → 해석)로 덱으로 만든다
// (resolvedDeck의 표 판 — render/web/table_test.go의 같은 이름 헬퍼와 짝이다).
func tableDeck(t *testing.T, body string) *soml.Deck {
	t.Helper()
	const source = "slides/001_table.slide"
	doc, diags := soml.Parse(source, "<slide id=\"table\">\n"+body+"\n</slide>\n")
	if len(diags) != 0 {
		t.Fatalf("파싱 실패: %v", diags)
	}
	stage, err := soml.Aspect16x9.Stage()
	if err != nil {
		t.Fatal(err)
	}
	deck := &soml.Deck{
		Title: "표", Lang: "ko", Aspect: soml.Aspect16x9, Stage: stage,
		Slides: []*soml.Slide{{
			Index: 0, ID: "table", Source: source, Content: doc.Content, Tokens: doc.Tokens,
		}},
	}
	if err := core.Resolve(deck, nil).Err(); err != nil {
		t.Fatalf("외관 해석 실패: %v", err)
	}
	return deck
}

// renderTableFrame는 덱의 첫 콘텐츠 노드(표 프레임)를 writeShapeNode로 쓴다 — 표가 도형 번호를
// 하나만 소비하는지(행·셀은 프레임 안에 접힌다)와 조각이 well-formed인지 함께 본다.
func renderTableFrame(t *testing.T, deck *soml.Deck, g *geom.Geometry) string {
	t.Helper()
	slide := deck.Slides[0]
	elem, ok := slide.Content[0].(*soml.Element)
	if !ok || elem.Name != soml.TableElement {
		t.Fatalf("첫 콘텐츠 노드가 표가 아니다: %T", slide.Content[0])
	}
	var b strings.Builder
	shapeID := firstShapeID
	if err := writeShapeNode(&b, deck, slide, elem, []int{0}, &shapeID, g, &slideRels{}); err != nil {
		t.Fatalf("표 렌더 실패: %v", err)
	}
	if shapeID != firstShapeID+1 {
		t.Errorf("표가 도형 번호 %d개를 썼다, want 1개 (행·셀은 프레임 안에 접힌다)", shapeID-firstShapeID)
	}
	out := b.String()
	dec := xml.NewDecoder(strings.NewReader(out))
	for {
		if _, err := dec.Token(); err == io.EOF {
			break
		} else if err != nil {
			t.Fatalf("표 XML이 well-formed가 아니다: %v\n%s", err, out)
		}
	}
	return out
}

// sections는 여는 태그 prefix부터 닫는 태그까지의 조각들이다 — 같은 요소가 여럿인 자리를 순서로 자른다.
func sections(t *testing.T, xml, open, close string) []string {
	t.Helper()
	var out []string
	rest := xml
	for {
		i := strings.Index(rest, open)
		if i < 0 {
			return out
		}
		rest = rest[i:]
		j := strings.Index(rest, close)
		if j < 0 {
			t.Fatalf("%s가 %s로 닫히지 않았다:\n%s", open, close, rest)
		}
		out = append(out, rest[:j+len(close)])
		rest = rest[j+len(close):]
	}
}

// gridColWidths는 열 너비를 산출물에서 되읽는다.
func gridColWidths(t *testing.T, xml string) []int64 {
	t.Helper()
	return attrInts(t, xml, `<a:gridCol w="`)
}

func attrInts(t *testing.T, xml, prefix string) []int64 {
	t.Helper()
	var out []int64
	for _, part := range strings.Split(xml, prefix)[1:] {
		value := part[:strings.Index(part, `"`)]
		n, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			t.Fatalf("%s%s는 정수가 아니다", prefix, value)
		}
		out = append(out, n)
	}
	return out
}

func sum(ns []int64) int64 {
	var total int64
	for _, n := range ns {
		total += n
	}
	return total
}

// 아무 셀도 말해 주지 않는 내부 경계는 균등 분할로 채우는지 못 박는다 (fillEdges) — 모든 행이
// 두 열을 colspan으로 덮으면 가운데 경계를 잰 값이 없다. 웹에서도 그 자리에 선을 그리는 셀이
// 없으므로 어디로 정해도 두 타깃이 갈라지지 않지만, **열 수는 맞아야** 파일이 열린다.
func TestRenderTableHiddenColumnEdge(t *testing.T) {
	deck := tableDeck(t, `  <table>
    <tr><td colspan="2">위</td></tr>
    <tr><td colspan="2">아래</td></tr>
  </table>`)
	g := measuredGeometry(t, deck,
		geom.Rect{X: 64, Y: 64, W: 1152, H: 80},
		geom.Rect{X: 64, Y: 64, W: 1152, H: 40},
		geom.Rect{X: 64, Y: 64, W: 1152, H: 40},
		geom.Rect{X: 64, Y: 104, W: 1152, H: 40},
		geom.Rect{X: 64, Y: 104, W: 1152, H: 40},
	)
	out := renderTableFrame(t, deck, g)

	cols := gridColWidths(t, out)
	if len(cols) != 2 || cols[0] != EMU(576) || cols[1] != EMU(576) {
		t.Errorf("열 너비 = %v, want 균등 분할 2열 (%d씩):\n%s", cols, EMU(576), out)
	}
	if sum(cols) != EMU(1152) {
		t.Errorf("열 너비 합 = %d, want cx %d", sum(cols), EMU(1152))
	}
}

// 머리행 힌트는 **첫 행이 전부 `<th>`일 때**만 켠다 — core의 얼룩 계산이 머리행을 가르는 규칙과
// 같다 (headerRow). 행 머리(각 행의 첫 칸만 th)인 표에서 첫 행을 머리행이라 말하면 PowerPoint의
// "머리글 행" 체크가 본문 행 하나를 가리킨다.
func TestRenderTableFirstRowHint(t *testing.T) {
	deck := tableDeck(t, `  <table>
    <tr><th>r0</th><td>0</td></tr>
    <tr><th>r1</th><td>1</td></tr>
  </table>`)
	g := measuredGeometry(t, deck,
		geom.Rect{X: 64, Y: 64, W: 1152, H: 80},
		geom.Rect{X: 64, Y: 64, W: 1152, H: 40},
		geom.Rect{X: 64, Y: 64, W: 400, H: 40},
		geom.Rect{X: 464, Y: 64, W: 752, H: 40},
		geom.Rect{X: 64, Y: 104, W: 1152, H: 40},
		geom.Rect{X: 64, Y: 104, W: 400, H: 40},
		geom.Rect{X: 464, Y: 104, W: 752, H: 40},
	)
	out := renderTableFrame(t, deck, g)
	if strings.Contains(out, "firstRow") {
		t.Errorf("행 머리 표에 머리행 힌트가 나갔다:\n%s", out)
	}
	if !strings.Contains(out, "<a:tblPr>") {
		t.Errorf("힌트 없는 tblPr이 빈 속성으로 나오지 않았다:\n%s", out)
	}
	// 줄무늬 힌트는 어느 표에도 켜지 않는다 — 줄무늬 색은 셀 채움에 이미 구워져 있다.
	if strings.Contains(out, "bandRow") {
		t.Errorf("줄무늬 힌트가 나갔다:\n%s", out)
	}
}

// 줄무늬 채움이 **셀 채움으로 구워져** 나오는지 못 박는다 ("스타일 없음 + 명시
// tcPr") — 표 스타일 상속에 기대면 뷰어마다 다른 색이 나온다. 홀수 본문 행(둘째 행)만 받는다.
func TestRenderTableStripedFillsAreBaked(t *testing.T) {
	deck := tableDeck(t, `  <table variant="striped">
    <tr><td>a</td></tr>
    <tr><td>b</td></tr>
  </table>`)
	g := measuredGeometry(t, deck,
		geom.Rect{X: 64, Y: 64, W: 1152, H: 80},
		geom.Rect{X: 64, Y: 64, W: 1152, H: 40},
		geom.Rect{X: 64, Y: 64, W: 1152, H: 40},
		geom.Rect{X: 64, Y: 104, W: 1152, H: 40},
		geom.Rect{X: 64, Y: 104, W: 1152, H: 40},
	)
	out := renderTableFrame(t, deck, g)
	cells := sections(t, out, "<a:tcPr", "</a:tcPr>")
	if len(cells) != 2 {
		t.Fatalf("tcPr %d개, want 2개:\n%s", len(cells), out)
	}
	// 첫 본문 행은 맨바탕(채움 없음), 둘째 행에 줄무늬 채움이 실린다. 테두리는 테마 기본값이
	// 살아 있으므로 noFill은 채움 하나뿐이다.
	if n := strings.Count(cells[0], "<a:noFill/>"); n != 1 {
		t.Errorf("짝수 본문 행에 채움이 실렸다:\n%s", cells[0])
	}
	if !strings.Contains(cells[1], "<a:solidFill>") {
		t.Errorf("홀수 본문 행에 줄무늬 채움이 구워지지 않았다:\n%s", cells[1])
	}
}

// 잰 좌표가 없으면 멈춘다 — 좌표 없는 표가 원점에 조용히 놓이거나, 격자가 0으로 접히는 일은
// 없어야 한다 (writeTextShape·writePicShape와 같은 관문).
func TestRenderTableWithoutGeometry(t *testing.T) {
	deck := tableDeck(t, "  <table>\n    <tr><td>a</td></tr>\n  </table>")
	slide := deck.Slides[0]
	elem := slide.Content[0].(*soml.Element)
	var b strings.Builder
	err := writeGraphicFrameTbl(&b, deck, slide, elem, "s0.0",
		firstShapeID, &geom.Geometry{Stage: deck.Stage}, &slideRels{})
	if err == nil || !strings.Contains(err.Error(), "has no measured coordinates") {
		t.Errorf("err = %v, want 측정된 좌표가 없다", err)
	}
}
