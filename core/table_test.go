package core

import (
	"reflect"
	"strings"
	"testing"

	"github.com/k2sebeom/balpyo/soml"
)

// tableSlide는 표 하나를 담은 슬라이드 원문이다. inner는 `<table>`과 `</table>` 사이의 행
// 마크업이다 — 들여쓰기를 고정하지 않는 이유는 표 테스트가 진단 위치가 아니라 확정 외관을
// 보기 때문이다 (위치를 보는 테스트는 needle로 열을 따로 잰다).
func tableSlide(attrs, inner string) string {
	open := "<table"
	if attrs != "" {
		open += " " + attrs
	}
	return "<slide id=\"t\">\n  " + open + ">" + inner + "</table>\n</slide>\n"
}

// cellStyles는 덱의 모든 셀 외관을 문서 순서로 모은다 — 셀은 여럿이라(도형의 shapeStyles와
// 같은 이유) 트리를 훑어 CellStyle을 순서대로 담는다.
func cellStyles(t *testing.T, deck *soml.Deck) []*soml.CellStyle {
	t.Helper()
	var styles []*soml.CellStyle
	var walk func(node soml.Node)
	walk = func(node soml.Node) {
		elem, ok := node.(*soml.Element)
		if !ok {
			return
		}
		if style := elem.CellStyle(); style != nil {
			styles = append(styles, style)
		}
		for _, child := range elem.Children {
			walk(child)
		}
	}
	for _, slide := range deck.Slides {
		for _, node := range slide.Content {
			walk(node)
		}
	}
	return styles
}

// resolvedCells는 진단 없이 해석된 표의 셀 외관을 문서 순서로 낸다.
func resolvedCells(t *testing.T, attrs, inner string) []*soml.CellStyle {
	t.Helper()
	deck, diags := resolveSlide(t, tableSlide(attrs, inner))
	if len(diags) != 0 {
		t.Fatalf("진단이 없어야 한다: %v", diags)
	}
	cells := cellStyles(t, deck)
	if len(cells) == 0 {
		t.Fatal("셀이 없다")
	}
	return cells
}

// 테마 `table` 컴포넌트의 값 — components.json의 minimal 테마와 짝이다 (값이 어긋나면 이
// 상수가 먼저 빨갛게 된다). 셀 기본값·헤더 델타·줄무늬 채움을 한자리에 모아, 기대값을 손으로
// 펼치지 않는다 (wantShapeStyle이 컴포넌트 기본값을 재사용하는 것과 같은 태도).
func tableCellBorder() *soml.Border {
	return &soml.Border{Width: 1, Style: soml.BorderSolid, Color: soml.TokenColor("muted")}
}

// TestResolveTableFrameVariant는 프레임 변형이 확정 TableStyle에 도달하는지, 모르는 변형이
// 에러가 아니라 경고 + 빈 값으로 폴백하는지 못 박는다 (도형·텍스트 variant와 같은 정책 —
// 콘텐츠의 테마 간 이식성, PRD §13-6).
func TestResolveTableFrameVariant(t *testing.T) {
	// 있는 변형은 이름이 남는다 — 웹이 CSS 훅을 낼 근거다 (soml.TableStyle.Variant).
	deck, diags := resolveSlide(t, tableSlide(`variant="striped"`, "<tr><td>a</td></tr>"))
	if len(diags) != 0 {
		t.Fatalf("진단이 없어야 한다: %v", diags)
	}
	frame := findElement(t, deck, soml.TableElement).TableStyle()
	if frame == nil {
		t.Fatal("<table>에 확정 외관이 없다")
	}
	if frame.Variant != "striped" {
		t.Errorf("Variant = %q, want \"striped\"", frame.Variant)
	}

	// 변형 없는 표는 빈 문자열이다 (뜻 없는 자리에 선언을 내지 않는다).
	deck, _ = resolveSlide(t, tableSlide("", "<tr><td>a</td></tr>"))
	if frame := findElement(t, deck, soml.TableElement).TableStyle(); frame == nil || frame.Variant != "" {
		t.Errorf("변형 없는 표 = %+v, want 빈 Variant", frame)
	}

	// 모르는 변형은 경고 하나 + 빈 값이다. 경고 문구는 테마가 가진 변형 목록을 나열한다
	// (도형·텍스트 variant 경고와 같은 태도).
	deck, diags = resolveSlide(t, tableSlide(`variant="zebra"`, "<tr><td>a</td></tr>"))
	if len(diags) != 1 {
		t.Fatalf("진단 %d개, want 1개: %v", len(diags), diags)
	}
	if diags[0].Severity != soml.SeverityWarning {
		t.Errorf("severity = %s, want warning — 에러면 테마 간 이식성이 깨진다", diags[0].Severity)
	}
	if !strings.Contains(diags[0].Message, `variant "zebra"`) || !strings.Contains(diags[0].Message, "striped") {
		t.Errorf("경고 문구 = %q, want 변형 이름과 테마 변형 목록 포함", diags[0].Message)
	}
	if frame := findElement(t, deck, soml.TableElement).TableStyle(); frame == nil || frame.Variant != "" {
		t.Errorf("모르는 변형이 실렸다: %+v", frame)
	}
}

// TestResolveCellDefaults는 속성 없는 데이터 셀이 테마 셀 기본값 + body 역할 텍스트로
// 확정되는지 못 박는다 — 셀 캐스케이드의 바닥이다 (도형의 컴포넌트 기본값과 같은 자리).
func TestResolveCellDefaults(t *testing.T) {
	cell := resolvedCells(t, "", "<tr><td>내용</td></tr>")[0]
	if cell.Header {
		t.Error("데이터 셀이 헤더로 표시됐다")
	}
	if cell.ColSpan != 1 || cell.RowSpan != 1 {
		t.Errorf("병합 칸 = %d×%d, want 1×1", cell.ColSpan, cell.RowSpan)
	}
	if cell.Fill != nil {
		t.Errorf("데이터 셀에 채움이 붙었다: %+v (테마 cell 기본값에 fill이 없다)", cell.Fill)
	}
	if !reflect.DeepEqual(cell.Border, tableCellBorder()) {
		t.Errorf("Border = %+v, want 테마 cell 기본값 %+v", cell.Border, tableCellBorder())
	}
	// 텍스트는 body 역할 기본값 그대로다 (헤더·인라인이 아무것도 얹지 않았다).
	if cell.Text == nil {
		t.Fatal("셀 텍스트 외관이 없다")
	}
	if want := textRoleDefaults(soml.RoleBody); *cell.Text != want {
		t.Errorf("셀 텍스트 = %+v, want body 역할 기본값 %+v", *cell.Text, want)
	}
}

// TestResolveHeaderCell은 헤더 셀이 테마 header 델타(채움·색·굵기)를 셀 기본값 위에 얹는지
// 못 박는다. 헤더는 태그(`<th>`)로 정해지고 외관은 캐스케이드가 확정한다 (셀 variant가 아니라
// 테마 header 단이다 — soml/table.go의 근거).
func TestResolveHeaderCell(t *testing.T) {
	cell := resolvedCells(t, "", "<tr><th>제목</th></tr>")[0]
	if !cell.Header {
		t.Error("<th>가 헤더로 표시되지 않았다")
	}
	// header가 채움을 정하고(accent1) 셀 기본 테두리는 남는다 (부분 패치).
	if want := soml.ColorFill(soml.TokenColor("accent1")); cell.Fill == nil || !reflect.DeepEqual(*cell.Fill, want) {
		t.Errorf("Fill = %+v, want 헤더 채움 %+v", cell.Fill, want)
	}
	if !reflect.DeepEqual(cell.Border, tableCellBorder()) {
		t.Errorf("Border = %+v, want 테마 cell 기본값이 남아야 한다", cell.Border)
	}
	// header가 텍스트 색·굵기를 얹고, body 역할의 나머지 필드는 남는다.
	body := textRoleDefaults(soml.RoleBody)
	if cell.Text.Color != soml.TokenColor("bg") || cell.Text.Weight != 700 {
		t.Errorf("헤더 텍스트 = %+v, want 색 bg + 굵기 700", *cell.Text)
	}
	if cell.Text.Size != body.Size || cell.Text.Font != body.Font || cell.Text.Align != body.Align {
		t.Errorf("헤더가 정하지 않은 텍스트 필드가 body 기본값을 잃었다: %+v", *cell.Text)
	}
}

// TestResolveCellOverrides는 인라인 셀 속성이 캐스케이드 최상단으로 테마 기본값을 이기는지,
// 값 문법이 도형과 **같은 해석 함수**를 지나 같은 필드에 도달하는지 못 박는다.
func TestResolveCellOverrides(t *testing.T) {
	inner := `<tr><td fill="surface" border="2 dashed fg" color="accent1" align="center" anchor="bottom">x</td></tr>`
	cell := resolvedCells(t, "", inner)[0]
	if want := soml.ColorFill(soml.TokenColor("surface")); cell.Fill == nil || !reflect.DeepEqual(*cell.Fill, want) {
		t.Errorf("Fill = %+v, want 인라인 채움 %+v", cell.Fill, want)
	}
	wantBorder := &soml.Border{Width: 2, Style: soml.BorderDashed, Color: soml.TokenColor("fg")}
	if !reflect.DeepEqual(cell.Border, wantBorder) {
		t.Errorf("Border = %+v, want 인라인 테두리 %+v (테마 기본값을 이긴다)", cell.Border, wantBorder)
	}
	if cell.Text.Color != soml.TokenColor("accent1") {
		t.Errorf("색 = %+v, want 인라인 accent1", cell.Text.Color)
	}
	if cell.Text.Align != soml.AlignCenter {
		t.Errorf("정렬 = %q, want center", cell.Text.Align)
	}
	if cell.Text.Anchor != soml.AnchorBottom {
		t.Errorf("앵커 = %q, want bottom", cell.Text.Anchor)
	}
}

// TestResolveCellFillNone은 셀 채움의 `none`이 색이 아닌 세 번째 상태로 확정되는지 못 박는다
// (도형 fill과 같은 값 문법 — 테마 헤더 채움을 셀에서 지우는 유일한 방법이다).
func TestResolveCellFillNone(t *testing.T) {
	cell := resolvedCells(t, "", `<tr><th fill="none">x</th></tr>`)[0]
	if cell.Fill == nil || !cell.Fill.None {
		t.Errorf("Fill = %+v, want none (헤더 채움을 인라인이 지운다)", cell.Fill)
	}
}

// TestResolveTableStriping은 프레임 변형 `striped`가 **홀수 본문 행의 비헤더 셀**에만 줄무늬
// 채움을 얹는지 못 박는다 (resolveTable의 얼룩 규칙). 헤더 행은 세지 않고, 헤더 셀은 줄무늬
// 대신 헤더 채움을 받는다.
func TestResolveTableStriping(t *testing.T) {
	// 첫 행은 헤더(모든 셀이 th), 이어 본문 행 셋. 본문 색인 0·1·2 중 홀수(1)만 줄무늬.
	inner := "<tr><th>H</th></tr>" +
		"<tr><td>0</td></tr>" +
		"<tr><td>1</td></tr>" +
		"<tr><td>2</td></tr>"
	cells := resolvedCells(t, `variant="striped"`, inner)
	if len(cells) != 4 {
		t.Fatalf("셀 %d개, want 4개", len(cells))
	}
	stripe := soml.ColorFill(soml.TokenColor("surface"))
	// 헤더 셀 — 줄무늬가 아니라 헤더 채움이다.
	if want := soml.ColorFill(soml.TokenColor("accent1")); cells[0].Fill == nil || !reflect.DeepEqual(*cells[0].Fill, want) {
		t.Errorf("헤더 셀 Fill = %+v, want 헤더 채움 (줄무늬 아님)", cells[0].Fill)
	}
	// 본문 0 — 맨바탕 (채움 없음).
	if cells[1].Fill != nil {
		t.Errorf("본문 첫 행 Fill = %+v, want 없음 (짝수 본문 행)", cells[1].Fill)
	}
	// 본문 1 — 줄무늬.
	if cells[2].Fill == nil || !reflect.DeepEqual(*cells[2].Fill, stripe) {
		t.Errorf("본문 둘째 행 Fill = %+v, want 줄무늬 %+v", cells[2].Fill, stripe)
	}
	// 본문 2 — 맨바탕.
	if cells[3].Fill != nil {
		t.Errorf("본문 셋째 행 Fill = %+v, want 없음 (짝수 본문 행)", cells[3].Fill)
	}

	// 변형이 없으면 줄무늬가 없다 — 같은 마크업이 프레임 변형 하나로만 갈린다.
	plain := resolvedCells(t, "", inner)
	for i, cell := range plain {
		if i == 0 {
			continue // 헤더는 늘 헤더 채움
		}
		if cell.Fill != nil {
			t.Errorf("변형 없는 표의 본문 셀[%d]에 채움이 붙었다: %+v", i, cell.Fill)
		}
	}
}

// TestResolveStripedHeaderCellUnaffected는 줄무늬 표에서도 헤더 셀이 줄무늬가 아닌 헤더
// 채움을 받는지 못 박는다 (헤더와 줄무늬는 배타 — resolveCell의 근거). 행 머리(각 행 첫 칸만
// th)인 표로, 본문 행에 든 `<th>`도 헤더로 남는지 본다.
func TestResolveStripedHeaderCellUnaffected(t *testing.T) {
	inner := "<tr><th>r0</th><td>0</td></tr>" +
		"<tr><th>r1</th><td>1</td></tr>"
	cells := resolvedCells(t, `variant="striped"`, inner)
	if len(cells) != 4 {
		t.Fatalf("셀 %d개, want 4개", len(cells))
	}
	header := soml.ColorFill(soml.TokenColor("accent1"))
	stripe := soml.ColorFill(soml.TokenColor("surface"))
	// 행 0(본문 색인 0, 짝수): 헤더 th는 헤더 채움, td는 맨바탕.
	if cells[0].Fill == nil || !reflect.DeepEqual(*cells[0].Fill, header) {
		t.Errorf("행0 th Fill = %+v, want 헤더 채움", cells[0].Fill)
	}
	if cells[1].Fill != nil {
		t.Errorf("행0 td Fill = %+v, want 없음", cells[1].Fill)
	}
	// 행 1(본문 색인 1, 홀수): 헤더 th는 여전히 헤더 채움(줄무늬 아님), td만 줄무늬.
	if cells[2].Fill == nil || !reflect.DeepEqual(*cells[2].Fill, header) {
		t.Errorf("행1 th Fill = %+v, want 헤더 채움 (줄무늬 아님)", cells[2].Fill)
	}
	if cells[3].Fill == nil || !reflect.DeepEqual(*cells[3].Fill, stripe) {
		t.Errorf("행1 td Fill = %+v, want 줄무늬", cells[3].Fill)
	}
}

// TestResolveCellSpan은 병합 칸 수가 확정 CellStyle에 도달하고, 1 미만이 에러인지 못 박는다.
// span은 외관 값이 아니라 격자 구조라 CellStyle에 직접 실린다 (cellPatch에 없는 근거).
func TestResolveCellSpan(t *testing.T) {
	cell := resolvedCells(t, "", `<tr><td colspan="2" rowspan="3">x</td></tr>`)[0]
	if cell.ColSpan != 2 || cell.RowSpan != 3 {
		t.Errorf("병합 칸 = %d×%d, want 2×3", cell.ColSpan, cell.RowSpan)
	}

	for _, tc := range []struct{ name, attrs string }{
		{"colspan 0", `colspan="0"`},
		{"rowspan 음수", `rowspan="-1"`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, diags := resolveSlide(t, tableSlide("", "<tr><td "+tc.attrs+">x</td></tr>"))
			if len(diags) != 1 {
				t.Fatalf("진단 %d개, want 1개: %v", len(diags), diags)
			}
			if diags[0].Severity != soml.SeverityError || !strings.Contains(diags[0].Message, "1 이상 정수") {
				t.Errorf("진단 = %q, want 1 이상 정수 에러", diags[0].String())
			}
		})
	}
}

// TestResolveCellInlineRunsInherit는 셀 안 인라인 런(strong/em/…)이 셀 텍스트를 상속하는지
// 못 박는다 — 셀이 서브트리를 통째로 확정하므로(resolveTable의 근거) 셀 안 런도 여기서
// 확정된다. 헤더 셀의 굵기·색을 인라인 런이 물려받는 것이 그 증거다.
func TestResolveCellInlineRunsInherit(t *testing.T) {
	deck, diags := resolveSlide(t, tableSlide("", "<tr><th>제목 <em>기울임</em></th></tr>"))
	if len(diags) != 0 {
		t.Fatalf("진단이 없어야 한다: %v", diags)
	}
	em := findElement(t, deck, "em").TextStyle()
	if em == nil {
		t.Fatal("<em>에 확정 외관이 없다")
	}
	// em은 셀 텍스트(헤더 굵기 700·색 bg)를 상속하고 이탤릭만 얹는다.
	if !em.Italic {
		t.Error("em이 이탤릭이 아니다")
	}
	if em.Weight != 700 || em.Color != soml.TokenColor("bg") {
		t.Errorf("em 상속 외관 = %+v, want 헤더 굵기·색 상속", *em)
	}
}

// TestResolveTableCellAttrCoverage는 값 층이 셀 속성 카탈로그(soml.TableCellAttrs) 전체를
// 덮는지 못 박는다 — 파서가 연 속성을 값 층이 해석하지 않으면 저자가 쓴 외관이 조용히
// 사라진다 (도형 속성 커버리지 테스트와 같은 장치).
func TestResolveTableCellAttrCoverage(t *testing.T) {
	// 각 속성에 유효한 값을 하나씩 실어, 진단 없이 확정 외관에 흔적을 남기는지 본다.
	values := map[string]string{
		"colspan": "2",
		"rowspan": "2",
		"fill":    "surface",
		"border":  "2 solid fg",
		"color":   "accent1",
		"align":   "center",
		"anchor":  "bottom",
	}
	for _, attr := range soml.TableCellAttrs() {
		value, ok := values[attr]
		if !ok {
			t.Errorf("셀 속성 %q의 해석 테스트가 없다", attr)
			continue
		}
		t.Run(attr, func(t *testing.T) {
			inner := "<tr><td " + attr + `="` + value + `">x</td></tr>`
			if _, diags := resolveSlide(t, tableSlide("", inner)); len(diags) != 0 {
				t.Errorf("%q 해석에 진단이 났다: %v", attr, diags)
			}
		})
	}
}
