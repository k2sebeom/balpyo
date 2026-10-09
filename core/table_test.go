package core

import (
	"reflect"
	"strings"
	"testing"

	"github.com/k2sebeom/balpyo/soml"
)

// tableSlide는 표 하나를 담은 슬라이드 원문이다. inner는 `<table>` 안의 행 마크업이다.
func tableSlide(attrs, inner string) string {
	open := "<table"
	if attrs != "" {
		open += " " + attrs
	}
	return "<slide id=\"t\">\n  " + open + ">" + inner + "</table>\n</slide>\n"
}

// cellStyles는 덱의 모든 셀 외관을 문서 순서로 모은다.
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

// minimal 테마 `table` 컴포넌트의 셀 기본 테두리.
func tableCellBorder() *soml.Border {
	return &soml.Border{Width: 1, Style: soml.BorderSolid, Color: soml.TokenColor("muted")}
}

// 있는 변형은 이름이 남고(웹 CSS 훅), 없으면 빈 값, 모르는 변형은 경고 + 빈 값이다 (ADR 0067 결정 6).
func TestResolveTableFrameVariant(t *testing.T) {
	for _, tc := range []struct {
		attrs, want string
		warn        bool
	}{
		{`variant="striped"`, "striped", false},
		{"", "", false},
		{`variant="zebra"`, "", true},
	} {
		deck, diags := resolveSlide(t, tableSlide(tc.attrs, "<tr><td>a</td></tr>"))
		if !tc.warn && len(diags) != 0 {
			t.Fatalf("%q: 진단이 없어야 한다: %v", tc.attrs, diags)
		}
		if tc.warn {
			if len(diags) != 1 || diags[0].Severity != soml.SeverityWarning {
				t.Fatalf("%q: 진단 = %v, want 경고 하나", tc.attrs, diags)
			}
			if !strings.Contains(diags[0].Message, `variant "zebra"`) || !strings.Contains(diags[0].Message, "striped") {
				t.Errorf("경고 문구 = %q, want 변형 이름과 테마 변형 목록 포함", diags[0].Message)
			}
		}
		frame := findElement(t, deck, soml.TableElement).TableStyle()
		if frame == nil || frame.Variant != tc.want {
			t.Errorf("%q: TableStyle = %+v, want Variant %q", tc.attrs, frame, tc.want)
		}
	}
}

// 속성 없는 데이터 셀은 테마 셀 기본값 + body 역할 + 앵커 top이다 (셀 캐스케이드의 바닥).
func TestResolveCellDefaults(t *testing.T) {
	cell := resolvedCells(t, "", "<tr><td>내용</td></tr>")[0]
	if cell.Header || cell.ColSpan != 1 || cell.RowSpan != 1 || cell.Fill != nil {
		t.Errorf("데이터 셀 = %+v, want 비헤더 · 1×1 · 채움 없음", cell)
	}
	if !reflect.DeepEqual(cell.Border, tableCellBorder()) {
		t.Errorf("Border = %+v, want 테마 cell 기본값 %+v", cell.Border, tableCellBorder())
	}
	if cell.Text == nil {
		t.Fatal("셀 텍스트 외관이 없다")
	}
	want := textRoleDefaults(soml.RoleBody)
	want.Anchor = soml.AnchorTop
	if *cell.Text != want {
		t.Errorf("셀 텍스트 = %+v, want %+v", *cell.Text, want)
	}
}

// 헤더 셀은 테마 header 델타(채움·색·굵기)를 셀 기본값 위에 부분 패치로 얹는다.
func TestResolveHeaderCell(t *testing.T) {
	cell := resolvedCells(t, "", "<tr><th>제목</th></tr>")[0]
	if !cell.Header {
		t.Error("<th>가 헤더로 표시되지 않았다")
	}
	if want := soml.ColorFill(soml.TokenColor("accent1")); cell.Fill == nil || !reflect.DeepEqual(*cell.Fill, want) {
		t.Errorf("Fill = %+v, want 헤더 채움 %+v", cell.Fill, want)
	}
	if !reflect.DeepEqual(cell.Border, tableCellBorder()) {
		t.Errorf("Border = %+v, want 테마 cell 기본값이 남아야 한다", cell.Border)
	}
	body := textRoleDefaults(soml.RoleBody)
	if cell.Text.Color != soml.TokenColor("bg") || cell.Text.Weight != 700 {
		t.Errorf("헤더 텍스트 = %+v, want 색 bg + 굵기 700", *cell.Text)
	}
	if cell.Text.Size != body.Size || cell.Text.Font != body.Font || cell.Text.Align != body.Align {
		t.Errorf("헤더가 정하지 않은 텍스트 필드가 body 기본값을 잃었다: %+v", *cell.Text)
	}
}

// 인라인 셀 속성은 테마 기본값을 이기고 도형과 같은 해석 함수를 지난다. fill="none"은 헤더
// 채움을 지우는 유일한 길이다.
func TestResolveCellOverrides(t *testing.T) {
	inner := `<tr><td fill="surface" border="2 dashed fg" color="accent1" align="center" anchor="bottom">x</td></tr>`
	cell := resolvedCells(t, "", inner)[0]
	if want := soml.ColorFill(soml.TokenColor("surface")); cell.Fill == nil || !reflect.DeepEqual(*cell.Fill, want) {
		t.Errorf("Fill = %+v, want 인라인 채움 %+v", cell.Fill, want)
	}
	wantBorder := &soml.Border{Width: 2, Style: soml.BorderDashed, Color: soml.TokenColor("fg")}
	if !reflect.DeepEqual(cell.Border, wantBorder) {
		t.Errorf("Border = %+v, want %+v", cell.Border, wantBorder)
	}
	if cell.Text.Color != soml.TokenColor("accent1") || cell.Text.Align != soml.AlignCenter || cell.Text.Anchor != soml.AnchorBottom {
		t.Errorf("셀 텍스트 = %+v, want accent1 · center · bottom", *cell.Text)
	}

	none := resolvedCells(t, "", `<tr><th fill="none">x</th></tr>`)[0]
	if none.Fill == nil || !none.Fill.None {
		t.Errorf("Fill = %+v, want none (헤더 채움을 인라인이 지운다)", none.Fill)
	}
}

// striped는 홀수 본문 행의 비헤더 셀에만 줄무늬를 얹는다. 헤더 행은 세지 않고, 헤더 셀은
// 본문 행에 있어도 헤더 채움을 받는다 (헤더와 줄무늬는 배타).
func TestResolveTableStriping(t *testing.T) {
	header := soml.ColorFill(soml.TokenColor("accent1"))
	stripe := soml.ColorFill(soml.TokenColor("surface"))
	for _, tc := range []struct {
		name, attrs, inner string
		want               []*soml.Fill
	}{
		{"헤더 행 + 본문 셋", `variant="striped"`,
			"<tr><th>H</th></tr><tr><td>0</td></tr><tr><td>1</td></tr><tr><td>2</td></tr>",
			[]*soml.Fill{&header, nil, &stripe, nil}},
		{"변형 없음", "",
			"<tr><th>H</th></tr><tr><td>0</td></tr><tr><td>1</td></tr><tr><td>2</td></tr>",
			[]*soml.Fill{&header, nil, nil, nil}},
		{"행 머리", `variant="striped"`,
			"<tr><th>r0</th><td>0</td></tr><tr><th>r1</th><td>1</td></tr>",
			[]*soml.Fill{&header, nil, &header, &stripe}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cells := resolvedCells(t, tc.attrs, tc.inner)
			if len(cells) != len(tc.want) {
				t.Fatalf("셀 %d개, want %d개", len(cells), len(tc.want))
			}
			for i, want := range tc.want {
				if !reflect.DeepEqual(cells[i].Fill, want) {
					t.Errorf("셀[%d] Fill = %+v, want %+v", i, cells[i].Fill, want)
				}
			}
		})
	}
}

// 병합 칸 수는 외관이 아니라 격자 구조라 CellStyle에 직접 실린다.
func TestResolveCellSpan(t *testing.T) {
	inner := "<tr><td colspan=\"2\" rowspan=\"3\">x</td><td>a</td></tr>" +
		"<tr><td>b</td></tr>" +
		"<tr><td>c</td></tr>"
	cell := resolvedCells(t, "", inner)[0]
	if cell.ColSpan != 2 || cell.RowSpan != 3 {
		t.Errorf("병합 칸 = %d×%d, want 2×3", cell.ColSpan, cell.RowSpan)
	}
}

// 셀이 서브트리를 통째로 확정하므로 셀 안 인라인 런은 셀 텍스트(헤더 굵기·색)를 상속한다.
func TestResolveCellInlineRunsInherit(t *testing.T) {
	deck, diags := resolveSlide(t, tableSlide("", "<tr><th>제목 <em>기울임</em></th></tr>"))
	if len(diags) != 0 {
		t.Fatalf("진단이 없어야 한다: %v", diags)
	}
	em := findElement(t, deck, "em").TextStyle()
	if em == nil || !em.Italic || em.Weight != 700 || em.Color != soml.TokenColor("bg") {
		t.Errorf("em 외관 = %+v, want 이탤릭 + 헤더 굵기·색 상속", em)
	}
}

// 격자 검증 (ADR 0036): 웹은 어긋난 표를 조용히 복구하지만 PPTX a:tbl은 열 수가 맞아야 열린다.
// 진단은 원인 하나에 하나 — span 값이 틀린 표에는 격자 진단을 덧붙이지 않는다.
func TestResolveTableGridIsRectangular(t *testing.T) {
	valid := "<tr><th>h0</th><th>h1</th><th>h2</th></tr>" +
		`<tr><td rowspan="2">a</td><td colspan="2">b</td></tr>` +
		"<tr><td>c</td><td>d</td></tr>"
	if _, diags := resolveSlide(t, tableSlide("", valid)); len(diags) != 0 {
		t.Fatalf("격자가 맞는 표에 진단이 났다: %v", diags)
	}

	for _, tc := range []struct {
		name, inner string
		want        []string
	}{
		{"colspan 0", `<tr><td colspan="0">x</td></tr>`, []string{"1 이상 정수"}},
		{"rowspan 음수", `<tr><td rowspan="-1">x</td></tr>`, []string{"1 이상 정수"}},
		{"span 에러면 격자 진단 없음", `<tr><td colspan="0">a</td></tr><tr><td>b</td><td>c</td></tr>`, []string{"1 이상 정수"}},
		{"행이 좁다", "<tr><td>a</td><td>b</td></tr><tr><td>c</td></tr>", []string{"1칸", "열은 2칸"}},
		{"행이 넓다", "<tr><td>a</td></tr><tr><td>b</td><td>c</td></tr>", []string{"2칸", "열은 1칸"}},
		{"rowspan이 마지막 행을 넘는다", `<tr><td rowspan="2">a</td></tr>`, []string{"rowspan 2", "마지막 행"}},
		// 칸 수(1+2+1=4)는 맞고 자리만 겹친다.
		{"병합이 겹친다", "<tr><td>a</td><td>b</td><td>c</td><td>d</td></tr>" +
			`<tr><td>e</td><td rowspan="2">f</td><td>g</td><td>h</td></tr>` +
			`<tr><td colspan="2">i</td><td>j</td></tr>`, []string{"겹친다", "2번째 열"}},
		{"첫 행이 비었다", "<tr></tr><tr><td>a</td></tr>", []string{"첫 행", "열 수"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, diags := resolveSlide(t, tableSlide("", tc.inner))
			if len(diags) != 1 {
				t.Fatalf("진단 %d개, want 1개: %v", len(diags), diags)
			}
			if diags[0].Severity != soml.SeverityError {
				t.Errorf("severity = %s, want error", diags[0].Severity)
			}
			for _, want := range tc.want {
				if !strings.Contains(diags[0].Message, want) {
					t.Errorf("진단 = %q, want %q 포함", diags[0].Message, want)
				}
			}
		})
	}
}

// 값 층이 셀 속성 카탈로그 전체를 해석해야 저자가 쓴 외관이 조용히 사라지지 않는다.
func TestResolveTableCellAttrCoverage(t *testing.T) {
	values := map[string]string{
		"fill":   "surface",
		"border": "2 solid fg",
		"color":  "accent1",
		"align":  "center",
		"anchor": "bottom",
	}
	// 병합 속성은 격자가 맞아야 하므로 표 전체를 준다.
	grids := map[string]string{
		"colspan": `<tr><td colspan="2">x</td></tr><tr><td>a</td><td>b</td></tr>`,
		"rowspan": `<tr><td rowspan="2">x</td><td>a</td></tr><tr><td>b</td></tr>`,
	}
	for _, attr := range soml.TableCellAttrs() {
		inner, ok := grids[attr]
		if !ok {
			value, found := values[attr]
			if !found {
				t.Errorf("셀 속성 %q의 해석 테스트가 없다", attr)
				continue
			}
			inner = "<tr><td " + attr + `="` + value + `">x</td></tr>`
		}
		if _, diags := resolveSlide(t, tableSlide("", inner)); len(diags) != 0 {
			t.Errorf("%q 해석에 진단이 났다: %v", attr, diags)
		}
	}
}
