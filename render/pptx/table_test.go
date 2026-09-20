package pptx

import (
	"encoding/xml"
	"io"
	"strconv"
	"strings"
	"testing"

	"github.com/k2sebeom/balpyo/core"
	"github.com/k2sebeom/balpyo/render/geom"
	"github.com/k2sebeom/balpyo/render/web"
	"github.com/k2sebeom/balpyo/soml"
)

// tableDeck는 표 하나를 담은 슬라이드를 파이프라인과 같은 순서(파싱 → 해석)로 덱으로 만든다
// (resolvedDeck의 표 판 — render/web/table_test.go의 같은 이름 헬퍼와 짝이다).
//
// 파서 진단에서 "아직 그리지 못하는 어휘"만 걸러내는 이유는 순서다: 두 렌더러가 **함께** 붙는
// 순간에만 `soml.unrenderedKinds`에서 표를 지울 수 있으므로(render/vocabulary_test.go의 양방향
// 대조), PPTX 쪽 절반을 짜는 동안은 파서가 아직 표를 막는다. 이 필터는 그 플립에서 사라진다.
func tableDeck(t *testing.T, body string) *soml.Deck {
	t.Helper()
	const source = "slides/001_table.slide"
	doc, diags := soml.Parse(source, "<slide id=\"table\">\n"+body+"\n</slide>\n")
	for _, d := range diags {
		if strings.Contains(d.Message, soml.UnrenderedVocabMessage) {
			continue
		}
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
	if err := core.Resolve(deck).Err(); err != nil {
		t.Fatalf("외관 해석 실패: %v", err)
	}
	return deck
}

// renderTableFrame는 덱의 첫 콘텐츠 노드(표 프레임)를 도형 조각으로 쓴다 — 디스패치가 아직
// 표를 받지 않으므로 writer를 직접 부른다 (tableDeck의 주석과 같은 순서 근거).
//
// 조각이 XML로서 well-formed인지도 함께 본다: 표는 우리가 내는 가장 깊은 중첩이라
// (graphicFrame > graphic > graphicData > tbl > tr > tc > txBody > p > r) 닫는 태그 하나만
// 어긋나도 PowerPoint가 복구를 묻는데, 그 실패는 골든 문자열 비교로는 잡히지 않는다.
func renderTableFrame(t *testing.T, deck *soml.Deck, g *geom.Geometry) string {
	t.Helper()
	slide := deck.Slides[0]
	elem, ok := slide.Content[0].(*soml.Element)
	if !ok || elem.Name != soml.TableElement {
		t.Fatalf("첫 콘텐츠 노드가 표가 아니다: %T", slide.Content[0])
	}
	var b strings.Builder
	id := web.ElementID(slide.Index, []int{0})
	if err := writeGraphicFrameTbl(&b, deck, slide, elem, id, firstShapeID, g, &slideRels{}); err != nil {
		t.Fatalf("표 렌더 실패: %v", err)
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

// section은 여는 태그 prefix부터 닫는 태그까지의 조각들이다 — `a:tr`·`a:tcPr`처럼 같은 요소가
// 여럿 나오는 자리에서 "이 셀 안에서의 순서"를 단정하려면 조각을 갈라야 한다 (spSection의
// 표 판이되, 이름으로 찾을 수 없어 순서로 자른다).
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

// gridColWidths·rowHeights는 격자 기하를 산출물에서 되읽는다 — 합이 프레임 크기와 같은지
// (타일링 불변식) 손으로 더하지 않기 위해서다.
func gridColWidths(t *testing.T, xml string) []int64 {
	t.Helper()
	return attrInts(t, xml, `<a:gridCol w="`)
}

func rowHeights(t *testing.T, xml string) []int64 {
	t.Helper()
	return attrInts(t, xml, `<a:tr h="`)
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

// 표 프레임 하나가 `p:graphicFrame` + `a:tbl`이 되고, 격자 기하가 **잰 박스**에서 오는지 못
// 박는다 (ADR 0036): 열 너비는 잰 셀 박스의 좌우 경계, 행 높이는 잰 `<tr>` 박스의 위아래
// 경계다. 표가 도형 하나를 소비하는 것(도형 id 하나)도 여기서 단정한다.
func TestRenderTableGraphicFrame(t *testing.T) {
	deck := tableDeck(t, `  <table>
    <tr><th>지역</th><th>Q1</th></tr>
    <tr><td>서울</td><td>12.4</td></tr>
  </table>`)
	// 프레임 1152×100, 열 경계 600px, 행 경계 40px (문서 순서: 표 → 행 → 셀들 → 행 → 셀들).
	g := measuredGeometry(t, deck,
		geom.Rect{X: 64, Y: 64, W: 1152, H: 100},
		geom.Rect{X: 64, Y: 64, W: 1152, H: 40},
		geom.Rect{X: 64, Y: 64, W: 600, H: 40},
		geom.Rect{X: 664, Y: 64, W: 552, H: 40},
		geom.Rect{X: 64, Y: 104, W: 1152, H: 60},
		geom.Rect{X: 64, Y: 104, W: 600, H: 60},
		geom.Rect{X: 664, Y: 104, W: 552, H: 60},
	)
	out := renderTableFrame(t, deck, g)

	for _, want := range []string{
		// 도형 하나 — p:sp도 p:pic도 아닌 세 번째 골격이다.
		"<p:graphicFrame>",
		`<p:cNvPr id="2" name="table s0.0"/>`,
		"<p:cNvGraphicFramePr/>",
		"<p:nvPr/>",
		// 좌표는 p:xfrm이다 (a:xfrm이 아니다) — 64px = 609600, 1152px = 10972800, 100px = 952500.
		"<p:xfrm>",
		`<a:off x="609600" y="609600"/>`,
		`<a:ext cx="10972800" cy="952500"/>`,
		// 표라는 선언 + 스타일 없음(ADR 0036) + 첫 행이 머리행이라는 힌트.
		`<a:graphicData uri="http://schemas.openxmlformats.org/drawingml/2006/table">`,
		`<a:tblPr firstRow="1">`,
		"<a:tableStyleId>{5940675A-B579-460E-94D1-54222C63F5DA}</a:tableStyleId>",
		// 열 너비 = 잰 셀 경계 (600px = 5715000, 552px = 5257800).
		`<a:gridCol w="5715000"/>`,
		`<a:gridCol w="5257800"/>`,
		// 행 높이 = 잰 <tr> 경계 (40px = 381000, 60px = 571500).
		`<a:tr h="381000">`,
		`<a:tr h="571500">`,
		"<a:t>지역</a:t>", "<a:t>12.4</a:t>",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("표 XML에 %q가 없다:\n%s", want, out)
		}
	}
	if n := strings.Count(out, "</a:tc>"); n != 4 {
		t.Errorf("a:tc %d개, want 4개 (2×2 격자):\n%s", n, out)
	}
	// 행·셀은 도형이 되지 않는다 — 프레임 안에 접힌다 (soml.unrenderedKinds의 계약이 아니라
	// 도형 수의 계약이다: 표 하나가 도형 하나를 소비한다).
	if n := strings.Count(out, "<p:sp>") + strings.Count(out, "<p:pic>"); n != 0 {
		t.Errorf("표가 p:sp/p:pic을 냈다 (%d개):\n%s", n, out)
	}
	if n := strings.Count(out, "<p:cNvPr "); n != 1 {
		t.Errorf("cNvPr %d개, want 1개 — 행·셀이 도형 번호를 쓰면 안 된다:\n%s", n, out)
	}
}

// 격자가 프레임을 **정확히** 타일링하는지 못 박는다 (edgeLengths): 열 너비의 합은 `cx`와,
// 행 높이의 합은 `cy`와 한 EMU도 다르지 않아야 한다. 경계가 정수 px가 아닌 값으로 잰
// 표에서만 드러나는 불변식이라(칸마다 반올림하면 합이 어긋난다) 소수 좌표로 잰다.
func TestRenderTableGridTilesTheFrame(t *testing.T) {
	deck := tableDeck(t, `  <table>
    <tr><td>a</td><td>b</td><td>c</td></tr>
  </table>`)
	g := measuredGeometry(t, deck,
		geom.Rect{X: 64, Y: 64, W: 1152, H: 40},
		geom.Rect{X: 64, Y: 64, W: 1152, H: 40},
		geom.Rect{X: 64, Y: 64, W: 383.5, H: 40},
		geom.Rect{X: 447.5, Y: 64, W: 383.5, H: 40},
		geom.Rect{X: 831, Y: 64, W: 385, H: 40},
	)
	out := renderTableFrame(t, deck, g)

	cols := gridColWidths(t, out)
	if len(cols) != 3 {
		t.Fatalf("gridCol %d개, want 3개:\n%s", len(cols), out)
	}
	if got, want := sum(cols), EMU(1152); got != want {
		t.Errorf("열 너비 합 = %d, want cx %d — 마지막 열이 프레임을 삐져나온다 (%v)", got, want, cols)
	}
	if got, want := sum(rowHeights(t, out)), EMU(40); got != want {
		t.Errorf("행 높이 합 = %d, want cy %d", got, want)
	}
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

// 셀의 확정 외관이 `a:tcPr`에 **전부 명시**되는지, 스키마 순서(선 → 채움)가 도형과 반대인
// 것을 지키는지 못 박는다 (ADR 0036 · CT_TableCellProperties).
//
// 정하지 않은 채널까지 명시하는 것이 요점이다: 채움 없음은 `a:noFill`, 테두리 없음은 네 변의
// `noFill`이다 — 비워 두면 뷰어의 표 스타일이 그 자리를 채우고 그 색은 웹 산출물에 없다.
func TestRenderTableCellPropertiesAreExplicit(t *testing.T) {
	deck := tableDeck(t, `  <table>
    <tr><th>머리</th></tr>
    <tr><td fill="none" border="none" anchor="bottom">본문</td></tr>
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
	header, body := cells[0], cells[1]

	// 여백은 웹의 셀 padding과 같은 상수다 (12px = 114300, 8px = 76200) — 두 벌이 되면
	// 글자 시작점이 갈라진다. 세로 앵커는 tcPr의 속성이다 (bodyPr이 아니다).
	if want := `<a:tcPr marL="114300" marR="114300" marT="76200" marB="76200" anchor="t">`; !strings.Contains(header, want) {
		t.Errorf("머리 셀 tcPr = %q, want %q로 시작", header, want)
	}
	if !strings.Contains(body, `anchor="b"`) {
		t.Errorf("본문 셀의 앵커가 tcPr에 없다:\n%s", body)
	}
	// 테두리는 네 변에 같은 값을 싣는다 (어휘에 변별 테두리가 없다). 머리 셀은 테마 기본
	// 테두리(1px = 9525), 본문 셀은 `none`이라 네 변이 noFill이다.
	for _, side := range []string{"a:lnL", "a:lnR", "a:lnT", "a:lnB"} {
		if !strings.Contains(header, "<"+side+` w="9525">`) {
			t.Errorf("머리 셀에 %s 굵기가 없다:\n%s", side, header)
		}
		if !strings.Contains(body, "<"+side+">") {
			t.Errorf("본문 셀에 %s가 없다 (선 없음도 명시한다):\n%s", side, body)
		}
	}
	if n := strings.Count(body, "<a:noFill/>"); n != 5 {
		t.Errorf("본문 셀의 noFill %d개, want 5개 (테두리 넷 + 채움):\n%s", n, body)
	}
	// 머리 셀 채움은 테마 header 델타(accent1)가 명시로 구워진다 — 표 스타일 상속이 아니다.
	if !strings.Contains(header, "<a:solidFill>") {
		t.Errorf("머리 셀에 채움이 없다:\n%s", header)
	}
	// **스키마 순서**: 네 변의 선이 먼저고 채움이 나중이다 (spPr과 반대 — 어기면 복구를 묻는다).
	if at, fill := strings.Index(header, "</a:lnB>"), strings.LastIndex(header, "<a:solidFill>"); at > fill {
		t.Errorf("머리 셀의 채움이 선보다 먼저 나왔다 (CT_TableCellProperties 위반):\n%s", header)
	}
	if at, fill := strings.Index(body, "</a:lnB>"), strings.LastIndex(body, "<a:noFill/>"); at > fill {
		t.Errorf("본문 셀의 채움이 선보다 먼저 나왔다:\n%s", body)
	}
}

// 셀 안 텍스트가 블록 텍스트와 **같은 문단 조립기**를 지나는지, 셀의 `a:bodyPr`이 비어 있는지
// 못 박는다 — 여백·앵커가 tcPr에 살므로(위 테스트) bodyPr에 같은 값을 또 실으면 뷰어가 둘을
// 더하는 순간 글자가 여백 두 벌만큼 밀린다.
func TestRenderTableCellText(t *testing.T) {
	deck := tableDeck(t, `  <table>
    <tr><td align="center">가운데 <strong>굵게</strong></td></tr>
  </table>`)
	g := measuredGeometry(t, deck,
		geom.Rect{X: 64, Y: 64, W: 1152, H: 40},
		geom.Rect{X: 64, Y: 64, W: 1152, H: 40},
		geom.Rect{X: 64, Y: 64, W: 1152, H: 40},
	)
	out := renderTableFrame(t, deck, g)

	for _, want := range []string{
		// 셀 본문의 bodyPr은 비어 있다 — 텍스트 박스의 인셋·앵커 표기가 여기 오면 안 된다.
		"<a:bodyPr/>",
		"<a:lstStyle/>",
		// 문단은 도형·블록과 같은 조립기다 (writeTextParagraph): 정렬과 절대 줄간격.
		`<a:pPr algn="ctr">`,
		"<a:spcPts val=",
		// 인라인 런도 같은 코드를 지난다 (writeRuns) — 굵게는 rPr@b다.
		"<a:t>가운데 </a:t>", `b="1"`, "<a:t>굵게</a:t>",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("셀 텍스트에 %q가 없다:\n%s", want, out)
		}
	}
	if strings.Contains(out, "<a:bodyPr ") {
		t.Errorf("셀 bodyPr에 속성이 실렸다 (여백·앵커는 tcPr의 몫이다):\n%s", out)
	}
}

// 병합이 `gridSpan`/`rowSpan`과 **덮인 칸**으로 나오는지 못 박는다 (ADR 0036): `a:tbl`은 행마다
// 열 수만큼의 `a:tc`를 요구하므로, 덮인 자리에 `hMerge`/`vMerge` 빈 칸이 서야 파일이 열린다.
func TestRenderTableMergedCells(t *testing.T) {
	deck := tableDeck(t, `  <table>
    <tr><td colspan="2" rowspan="2">큰 칸</td><td>c</td></tr>
    <tr><td>f</td></tr>
    <tr><td>g</td><td>h</td><td>i</td></tr>
  </table>`)
	g := measuredGeometry(t, deck,
		geom.Rect{X: 64, Y: 64, W: 1152, H: 120}, // table
		geom.Rect{X: 64, Y: 64, W: 1152, H: 40},  // tr 0
		geom.Rect{X: 64, Y: 64, W: 800, H: 80},   // 큰 칸 (2열 × 2행)
		geom.Rect{X: 864, Y: 64, W: 352, H: 40},  // c
		geom.Rect{X: 64, Y: 104, W: 1152, H: 40}, // tr 1
		geom.Rect{X: 864, Y: 104, W: 352, H: 40}, // f
		geom.Rect{X: 64, Y: 144, W: 1152, H: 40}, // tr 2
		geom.Rect{X: 64, Y: 144, W: 400, H: 40},  // g
		geom.Rect{X: 464, Y: 144, W: 400, H: 40}, // h
		geom.Rect{X: 864, Y: 144, W: 352, H: 40}, // i
	)
	out := renderTableFrame(t, deck, g)

	for _, want := range []string{
		// 병합 셀 자신 — 속성 순서는 웹의 colspan/rowspan 순서를 따른다.
		`<a:tc gridSpan="2" rowSpan="2">`,
		// 오른쪽으로 덮인 칸, 아래로 덮인 칸, 그리고 둘 다인 대각 칸.
		`<a:tc hMerge="1">`,
		`<a:tc vMerge="1">`,
		`<a:tc hMerge="1" vMerge="1">`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("표 XML에 %q가 없다:\n%s", want, out)
		}
	}
	// 행마다 정확히 열 수(3)만큼의 칸이 있어야 한다 — 이 수가 어긋나면 PowerPoint가 복구를 묻는다.
	rows := sections(t, out, "<a:tr ", "</a:tr>")
	if len(rows) != 3 {
		t.Fatalf("a:tr %d개, want 3개:\n%s", len(rows), out)
	}
	for i, row := range rows {
		if n := strings.Count(row, "</a:tc>"); n != 3 {
			t.Errorf("%d번째 행의 a:tc %d개, want 3개:\n%s", i, n, row)
		}
	}
	// 병합하지 않은 셀에 기본값을 적지 않는다 (웹이 colspan="1"을 내지 않는 것과 같은 태도).
	if strings.Contains(out, `gridSpan="1"`) || strings.Contains(out, `rowSpan="1"`) {
		t.Errorf("병합하지 않은 셀에 span 속성이 나갔다:\n%s", out)
	}
	// 덮인 칸은 외관을 갖지 않는다 — 빈 본문과 빈 tcPr이다 (병합을 무시하는 소비자에서 유령
	// 셀이 보이지 않게).
	if n := strings.Count(out, "<a:tcPr/>"); n != 3 {
		t.Errorf("빈 tcPr %d개, want 3개 (덮인 칸 셋):\n%s", n, out)
	}
	// 열 너비는 나중 행이 드러낸 경계까지 모아야 맞는다 (첫 행은 400px 경계를 가린다).
	cols := gridColWidths(t, out)
	if len(cols) != 3 || cols[0] != EMU(400) || cols[1] != EMU(400) || cols[2] != EMU(352) {
		t.Errorf("열 너비 = %v, want [%d %d %d]:\n%s", cols, EMU(400), EMU(400), EMU(352), out)
	}
	if got := rowHeights(t, out); len(got) != 3 || sum(got) != EMU(120) {
		t.Errorf("행 높이 = %v, want 합 %d", got, EMU(120))
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
	// 줄무늬 힌트는 어느 표에도 켜지 않는다 — 줄무늬 색은 셀 채움에 이미 구워져 있다 (ADR 0036).
	if strings.Contains(out, "bandRow") {
		t.Errorf("줄무늬 힌트가 나갔다:\n%s", out)
	}
}

// 줄무늬 채움이 **셀 채움으로 구워져** 나오는지 못 박는다 (ADR 0036의 "스타일 없음 + 명시
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
	if err == nil || !strings.Contains(err.Error(), "측정된 좌표가 없다") {
		t.Errorf("err = %v, want 측정된 좌표가 없다", err)
	}
}
