package pptx

import (
	"fmt"
	"strings"

	"github.com/k2sebeom/balpyo/render/geom"
	"github.com/k2sebeom/balpyo/render/web"
	"github.com/k2sebeom/balpyo/soml"
)

// 이 파일은 **표의 PPTX 렌더**다 — 프레임 하나가 `p:graphicFrame` + `a:tbl`이 되고,
// 행·셀은 그 안의 `a:tr`/`a:tc`로 접힌다 (도형이 되지 않는다 — soml/table.go의 종류 계약).
//
// # `p:sp`도 `p:pic`도 아닌 세 번째 요소다
//
// 도형과 그림은 좌표·기하·채움·선을 같은 함수로 쓰지만(image.go의 "도형의 함수를 그대로 쓴다"),
// 표는 그 골격 자체가 다르다: `xfrm`이 `p:xfrm`(a:가 아니다)이고, 기하가 없고(격자가 기하다),
// 채움·선이 프레임에 없다(셀이 소유한다). 그래서 writeSp·writePic과 나란한 세 번째 조립기를
// 둔다 — 같은 함수에 분기를 넣으면 스키마 순서 셋이 한 함수에 얽힌다.
//
// 재사용하는 것은 **셀 안쪽**이다: 채움(writeFill) · 선(writeLineElem) · 세로 앵커
// (bodyAnchorValue) · 문단과 런(writeTextParagraph → writeRuns). 셀 텍스트가 블록 텍스트와 같은
// 함수를 지나는 것이 "표 안에서만 글자가 다르게 나가는 일"을 막는다 (shapeBodyXML이 도형 안
// 텍스트에 같은 문단 조립기를 쓰는 것과 같은 근거).
//
// # 격자 기하는 **잰 값**이다 — 계산하지 않는다
//
//	프레임 xfrm  ← 잰 `<table>` 박스
//	a:gridCol@w  ← 잰 셀 박스의 좌우 경계 (브라우저의 표 알고리즘이 정한 열 너비)
//	a:tr@h       ← 잰 `<tr>` 박스의 위아래 경계
//
// 열 너비를 셀 **폭**이 아니라 **경계**에서 뽑는 이유는 정확한 타일링이다 (edgeLengths): 칸마다
// 반올림해 더하면 합이 프레임 폭과 어긋나 마지막 열이 삐져나오거나 표가 좁아진다. 경계를 먼저
// EMU로 환산하고 차분하면 합이 프레임 크기와 **정확히** 같다.
//
// # 명시 tcPr + 스타일 없음
//
// 셀 채움·테두리·여백·세로 앵커를 셀마다 명시한다. 정하지 않은 채널도 명시한다 — 채움 없음은
// `a:noFill`, 테두리 없음은 네 변의 `a:lnX > a:noFill`이다: 비워 두면 뷰어의 표 스타일이 그
// 자리를 채우고, 그 색은 웹 산출물에 없다 (writeLine·writeFill의 "명시한다"와 같은 근거이고,
// 웹 쪽 절반은 cellStyleCSS가 `transparent`·`border-style: none`을 내는 것이다).

const (
	// tableGraphicDataURI는 `a:graphicData@uri`다 — 이 프레임에 든 그래픽이 **표**라는 선언이다.
	// 값은 DrawingML 표의 고정 URI이고, 틀리면 뷰어가 프레임을 빈 자리로 그린다 (차트·다이어그램이
	// 같은 자리에 다른 URI로 온다).
	tableGraphicDataURI = "http://schemas.openxmlformats.org/drawingml/2006/table"
	// tableNoStyleID는 내장 표 스타일 "No Style, No Grid"의 GUID다.
	//
	// **가리키는 것 자체가 "스타일 없음"의 선언이다.** 생략해도 스키마상 유효하지만,
	// 그러면 뷰어가 `tableStyles.xml`의 기본 스타일(`@def`)을 얹을 여지가 남고 그 순간 우리가
	// 명시하지 않은 채널이 뷰어의 값으로 채워진다. 이 GUID를 가리키면 얹을 것이 없고, 내장
	// 스타일이라 `tableStyles.xml` 파트를 만들 필요도 없다.
	tableNoStyleID = "{5940675A-B579-460E-94D1-54222C63F5DA}"
)

// writeGraphicFrameTbl는 표 프레임 하나를 `p:graphicFrame` + `a:tbl`로 쓴다.
//
// 서명은 shapeWriter다 — 표는 **도형 하나**를 소비하므로(도형 id 하나, spTree 자리 하나) 도형이
// 되는 다른 종류와 같은 자리를 지난다 (shapeWriterFor). 자식으로 내려가지 않는 점은 `<shape>`와
// 같고, 자식이 자기 도형을 갖지 않는 이유도 같다 — 행·셀은 프레임 안에 접힌다.
func writeGraphicFrameTbl(b *strings.Builder, deck *soml.Deck, slide *soml.Slide, elem *soml.Element, id string, shapeID int, g *geom.Geometry, rels *slideRels) error {
	box, ok := g.Box(id)
	if !ok {
		return fmt.Errorf("%s: %s <%s>의 측정된 좌표가 없다", slide.Source, id, elem.Name)
	}
	style := elem.TableStyle()
	if style == nil {
		// 해석되지 않은 외관을 기본값으로 그리면 PPTX만 그럴듯하게 나오고 웹과 갈라진다
		// (render/web.writeTable과 같은 관문).
		return fmt.Errorf("%s:%s <%s>의 외관이 해석되지 않았다 — core.Resolve를 거치지 않은 덱이다",
			slide.Source, elem.Span, elem.Name)
	}
	spec, err := tableSpec(slide, elem, id, shapeID, box.Rect, g)
	if err != nil {
		return fmt.Errorf("%s:%s <%s>: %w", slide.Source, elem.Span, elem.Name, err)
	}
	// 저자의 `alt`은 프레임의 이름이다 — `ph`와 같이 명세 바깥에서 얹는다: 격자를
	// 만드는 계산(tableSpec)에 관여하지 않는 값이다.
	spec.descr = style.Alt
	spec.ph = slotPlaceholder(slide, elem)
	if err := writeGraphicFrame(b, spec, deck.Theme, rels); err != nil {
		return fmt.Errorf("%s:%s <%s>: %w", slide.Source, elem.Span, elem.Name, err)
	}
	return nil
}

// tableInfo는 표 하나의 조립 명세다 (spSpec·picInfo의 형제 — 골격을 함수 하나에 두어야 스키마
// 순서가 한 곳에서만 지켜진다는 근거도 같다).
type tableInfo struct {
	id   int
	name string
	// descr은 `cNvPr@descr` = 저자의 `alt`이고 **표 전체의 이름**이다 — 셀은
	// `a:tbl` 안에 그대로 남아 항해 가능하다 (웹의 `<table aria-label>`과 같은 자리).
	descr string
	// rect는 잰 프레임 박스 그대로다 (열 너비의 합이 프레임 폭이다).
	rect geom.Rect
	ph   *placeholder
	// firstRow는 `a:tblPr@firstRow` 힌트다 — 첫 행이 머리행일 때만 켠다.
	firstRow bool
	// cols는 열 너비다 (EMU). 합은 프레임 폭과 정확히 같다 (edgeLengths).
	cols []int64
	rows []tableRowInfo
}

// tableRowInfo는 행 하나다 — 높이(잰 `<tr>` 박스)와 격자 칸이다. cells의 길이는 **언제나**
// 열 수와 같다: 병합에 덮인 칸까지 `a:tc`로 내야 `a:tbl`이 열린다.
type tableRowInfo struct {
	h     int64
	cells []tableCellInfo
}

// tableCellInfo는 격자 칸 하나다 — 저자가 쓴 셀이거나 병합에 덮인 빈 칸이다.
type tableCellInfo struct {
	// elem·style은 저자가 쓴 셀이다. 덮인 칸은 둘 다 nil이고 hMerge/vMerge만 참이다.
	elem  *soml.Element
	style *soml.CellStyle
	// colSpan·rowSpan은 `gridSpan`/`rowSpan`이다 (1이면 속성을 내지 않는다).
	colSpan, rowSpan int
	// hMerge·vMerge는 이 칸이 왼쪽·위쪽 셀의 병합에 덮였다는 표시다. 2×2 병합의 대각 칸처럼
	// 둘 다 참일 수 있다 (OOXML이 둘을 함께 받는다).
	hMerge, vMerge bool
}

// covered는 이 칸이 병합에 덮인 빈 칸인지다.
func (c tableCellInfo) covered() bool { return c.elem == nil }

// tableSpec은 해석된 표와 잰 박스를 표 명세로 컴파일한다 — 격자 펼치기(병합 → 덮인 칸)와
// 기하 뽑기(경계 → 열 너비·행 높이)가 전부다.
func tableSpec(slide *soml.Slide, elem *soml.Element, id string, shapeID int, rect geom.Rect, g *geom.Geometry) (tableInfo, error) {
	rows, err := tableRowNodes(elem, id)
	if err != nil {
		return tableInfo{}, err
	}
	if len(rows) == 0 {
		// 행 없는 표는 격자가 없다 — `a:tblGrid`는 열 하나 이상을 요구하므로 파일이 열리지
		// 않는다 (빈 `<ul>`이 빈 문단으로 도형을 내는 것과 갈라지는 자리: 리스트는 도형
		// 하나이고 표는 격자다).
		return tableInfo{}, fmt.Errorf("행이 없다 — 표는 행 하나 이상을 담아야 한다")
	}
	grid, err := tableGridCells(slide, rows)
	if err != nil {
		return tableInfo{}, err
	}
	cols, err := tableColWidths(g, rect, grid, rows)
	if err != nil {
		return tableInfo{}, err
	}
	heights, err := tableRowHeights(g, rect, rows)
	if err != nil {
		return tableInfo{}, err
	}

	spec := tableInfo{
		id: shapeID, name: elem.Name + " " + id,
		rect: rect, cols: cols,
		// 머리행 힌트는 **첫 행이 전부 `<th>`일 때**만 켠다 — core의 얼룩 계산이 머리행을
		// 가르는 규칙과 같다 (headerRow): 행 머리(각 행의 첫 칸만 th)인 표에서 첫 행을
		// 머리행이라 말하면 PowerPoint의 "머리글 행" 체크가 본문 행 하나를 가리킨다.
		firstRow: headerGridRow(grid[0]),
	}
	for i, cells := range grid {
		spec.rows = append(spec.rows, tableRowInfo{h: heights[i], cells: cells})
	}
	return spec, nil
}

// tableNode는 표 서브트리의 요소 하나와 그 지오메트리 식별자다 — 행·셀의 잰 박스를 찾는 열쇠라
// 요소와 함께 나른다 (web.ChildID).
type tableNode struct {
	elem *soml.Element
	id   string
}

// tableRowNodes는 프레임의 행을, tableCellNodes는 행의 셀을 문서 순서로 낸다.
//
// 자식 색인을 **그대로** 식별자로 쓴다 (걸러내지 않는다): 잰 식별자는 웹 렌더러가 낸 순회에서
// 왔으므로(writeTable → ChildPath) 색인이 하나라도 밀리면 모든 셀 박스가 다른 셀의 것이 된다.
// 콘텐츠 모델은 파서가 지켰으니(soml.tableChildren) 다른 노드는 파서를 거치지 않은 덱이다.
func tableRowNodes(table *soml.Element, id string) ([]tableNode, error) {
	var rows []tableNode
	for i, child := range table.Children {
		row, ok := child.(*soml.Element)
		if !ok || row.Name != soml.TableRowElement {
			return nil, fmt.Errorf("<table>의 자식 %T는 행이 아니다 — 파서를 거치지 않은 덱이다", child)
		}
		rows = append(rows, tableNode{elem: row, id: web.ChildID(id, i)})
	}
	return rows, nil
}

func tableCellNodes(row tableNode) ([]tableNode, error) {
	var cells []tableNode
	for i, child := range row.elem.Children {
		cell, ok := child.(*soml.Element)
		if !ok || !soml.IsTableCell(cell.Name) {
			return nil, fmt.Errorf("<tr>의 자식 %T는 셀이 아니다 — 파서를 거치지 않은 덱이다", child)
		}
		cells = append(cells, tableNode{elem: cell, id: web.ChildID(row.id, i)})
	}
	return cells, nil
}

// tableGridCells는 저자가 쓴 셀을 격자 칸으로 펼친다 — 행마다 정확히 열 수만큼의 칸을 내고,
// 병합에 덮인 자리는 `hMerge`/`vMerge` 빈 칸이 된다.
//
// 배치 규칙은 값 층의 격자 검증과 **같다** (core.checkTableGrid): 첫 행이 열 수를 정하고, 이후
// 행은 위에서 내려온 rowspan이 덮은 칸을 건너뛰며 왼쪽부터 채운다. 같은 규칙을 두 층이 쓰는
// 것이 "웹에서 본 격자가 PPTX의 격자다"의 근거이고, 여기서 어긋남을 만나면 **에러**다 —
// 값 층이 이미 진단으로 막은 모양이므로(격자가 깨진 표는 빌드가 멈춘다) 여기 도달한 어긋남은
// 두 층의 규칙이 갈라졌다는 뜻이다 (조용히 칸을 채우면 PowerPoint가 복구 대화상자를 띄운다).
func tableGridCells(slide *soml.Slide, rows []tableNode) ([][]tableCellInfo, error) {
	authored := make([][]tableCellInfo, len(rows))
	for i, row := range rows {
		nodes, err := tableCellNodes(row)
		if err != nil {
			return nil, err
		}
		for _, node := range nodes {
			style := node.elem.CellStyle()
			if style == nil {
				return nil, fmt.Errorf("%s <%s>의 외관이 해석되지 않았다 — core.Resolve를 거치지 않은 덱이다",
					node.elem.Span, node.elem.Name)
			}
			authored[i] = append(authored[i], tableCellInfo{
				elem: node.elem, style: style,
				colSpan: cellSpanOf(style.ColSpan), rowSpan: cellSpanOf(style.RowSpan),
			})
		}
	}

	cols := 0
	for _, cell := range authored[0] {
		cols += cell.colSpan
	}
	if cols == 0 {
		// 값 층이 같은 문구로 이미 막았다 (checkTableGrid) — 빈 `a:tblGrid`는 파일이 열리지
		// 않으므로 여기서도 멈춘다.
		return nil, fmt.Errorf("첫 행에 셀이 없다 — 첫 행이 표의 열 수를 정한다")
	}

	// owner[i][c]는 (i,c) 칸을 차지한 셀의 **좌상단 자리**다 — 그 자리와 다른 칸이 덮인 칸이고,
	// 어느 방향으로 덮였는지가 hMerge/vMerge를 정한다.
	type origin struct{ row, col int }
	owner := make([][]*origin, len(rows))
	grid := make([][]tableCellInfo, len(rows))
	for i := range rows {
		owner[i] = make([]*origin, cols)
		grid[i] = make([]tableCellInfo, cols)
	}
	for i, cells := range authored {
		col := 0
		for _, cell := range cells {
			for col < cols && owner[i][col] != nil {
				col++
			}
			if col+cell.colSpan > cols || i+cell.rowSpan > len(rows) {
				return nil, fmt.Errorf("%s <%s>의 병합이 격자를 넘는다 (%d×%d 표의 %d번째 열) — 값 층의 격자 검증과 갈라졌다",
					cell.elem.Span, cell.elem.Name, cols, len(rows), col+1)
			}
			here := &origin{row: i, col: col}
			for dr := 0; dr < cell.rowSpan; dr++ {
				for dc := 0; dc < cell.colSpan; dc++ {
					if owner[i+dr][col+dc] != nil {
						return nil, fmt.Errorf("%s <%s>의 병합이 겹친다 (%d번째 열) — 값 층의 격자 검증과 갈라졌다",
							cell.elem.Span, cell.elem.Name, col+dc+1)
					}
					owner[i+dr][col+dc] = here
				}
			}
			grid[i][col] = cell
			col += cell.colSpan
		}
	}
	for i := range grid {
		for c := range grid[i] {
			own := owner[i][c]
			if own == nil {
				return nil, fmt.Errorf("%s: 격자의 %d행 %d열이 비었다 — 값 층의 격자 검증과 갈라졌다",
					slide.Source, i+1, c+1)
			}
			if own.row == i && own.col == c {
				continue
			}
			grid[i][c] = tableCellInfo{hMerge: c > own.col, vMerge: i > own.row}
		}
	}
	return grid, nil
}

// cellSpanOf는 확정 병합 칸 수를 1 이상으로 눕힌다. 값 층이 1 미만을 이미 에러로 막았지만
// (core.cellSpan) 진단이 난 덱도 이 렌더러에 닿을 수 있고(경고만 보고 계속하는 호출자), 0은
// 격자 배치를 무한 루프로 만든다.
func cellSpanOf(n int) int {
	if n < 1 {
		return 1
	}
	return n
}

// headerGridRow는 격자 행의 칸이 **모두** 머리 셀인지 본다 (덮인 칸은 세지 않는다) —
// core.headerRow의 격자 판이다.
func headerGridRow(cells []tableCellInfo) bool {
	found := false
	for _, cell := range cells {
		if cell.covered() {
			continue
		}
		if !cell.style.Header {
			return false
		}
		found = true
	}
	return found
}

// tableColWidths는 잰 셀 박스에서 열 너비(EMU)를 뽑는다.
//
// 셀의 **폭**이 아니라 **좌우 경계**를 모으는 이유는 정확한 타일링이다 (파일 머리 주석): 경계를
// 프레임 기준 상대 좌표로 모아 EMU로 환산한 뒤 차분하면 합이 프레임 폭과 정확히 같다.
//
// 경계는 병합 셀이 가릴 수 있다 (모든 행이 두 열을 colspan="2"로 덮으면 가운데 경계를 아무도
// 말해 주지 않는다). 그때는 남은 구간을 균등 분할한다 (fillEdges) — 웹에서도 그 경계는 보이지
// 않으므로(그 자리에 선을 그리는 셀이 없다) 어디로 정해도 두 타깃이 갈라지지 않는다.
func tableColWidths(g *geom.Geometry, frame geom.Rect, grid [][]tableCellInfo, rows []tableNode) ([]int64, error) {
	cols := len(grid[0])
	edges := make([]float64, cols+1)
	known := make([]bool, cols+1)
	edges[0], known[0] = 0, true
	edges[cols], known[cols] = frame.W, true

	for i, cells := range grid {
		nodes, err := tableCellNodes(rows[i])
		if err != nil {
			return nil, err
		}
		at := 0 // 저자가 쓴 셀의 순서 — 덮인 칸은 요소가 없다.
		for c, cell := range cells {
			if cell.covered() {
				continue
			}
			if at >= len(nodes) {
				return nil, fmt.Errorf("격자의 %d행 셀 수가 잰 셀 수와 어긋난다", i+1)
			}
			node := nodes[at]
			at++
			box, ok := g.Box(node.id)
			if !ok {
				return nil, fmt.Errorf("%s <%s>의 측정된 좌표가 없다 (%s)", node.id, node.elem.Name, node.elem.Span)
			}
			// 첫 행의 경계가 이긴다 (문서 순서) — 같은 경계를 여러 행이 말할 때 값이 미세하게
			// 다를 수 있고, 한 행의 값으로 통일해야 열 너비가 흔들리지 않는다.
			if !known[c] {
				edges[c], known[c] = box.Rect.X-frame.X, true
			}
			if right := c + cell.colSpan; !known[right] {
				edges[right], known[right] = box.Rect.Right()-frame.X, true
			}
		}
	}
	fillEdges(edges, known)
	return edgeLengths(edges), nil
}

// tableRowHeights는 잰 `<tr>` 박스에서 행 높이(EMU)를 뽑는다 — 열 너비와 같은 경계 방식이다
// (행의 **높이**를 하나씩 환산하면 합이 프레임 높이와 어긋난다).
//
// 행이 자기 식별자를 갖는 이유가 여기다: `<tr>`은 외관을 갖지 않지만(줄무늬조차 셀 채움이다)
// `a:tr@h`가 그 박스에서 온다 (render/web.writeTableRow의 주석과 짝).
func tableRowHeights(g *geom.Geometry, frame geom.Rect, rows []tableNode) ([]int64, error) {
	edges := make([]float64, len(rows)+1)
	edges[len(rows)] = frame.H
	for i, row := range rows {
		box, ok := g.Box(row.id)
		if !ok {
			return nil, fmt.Errorf("%s <%s>의 측정된 좌표가 없다 (%s)", row.id, row.elem.Name, row.elem.Span)
		}
		if i == 0 {
			// 첫 행의 위 경계는 프레임의 위 변이다 — 잰 값을 쓰면 테두리 합침(border-collapse)의
			// 반 픽셀이 첫 행에만 실린다.
			continue
		}
		edges[i] = box.Rect.Y - frame.Y
	}
	return edgeLengths(edges), nil
}

// fillEdges는 아무도 말해 주지 않은 경계를 이웃 경계 사이에 균등 분할로 채운다 (양 끝은 늘
// 알려져 있으므로 — 프레임의 두 변 — 채울 구간은 언제나 유한하다).
func fillEdges(edges []float64, known []bool) {
	for i := range edges {
		if known[i] {
			continue
		}
		prev := i - 1
		for prev > 0 && !known[prev] {
			prev--
		}
		next := i + 1
		for next < len(edges)-1 && !known[next] {
			next++
		}
		step := (edges[next] - edges[prev]) / float64(next-prev)
		edges[i], known[i] = edges[prev]+step*float64(i-prev), true
	}
}

// edgeLengths는 경계 위치 목록(프레임 상대 px)을 칸 길이(EMU)로 바꾼다.
//
// **환산이 차분보다 먼저다**: `EMU(경계[i+1]) - EMU(경계[i])`의 합은 정의상 `EMU(프레임 크기)`와
// 정확히 같다. 반대 순서(`EMU(경계[i+1] - 경계[i])`)는 칸마다 반올림 오차가 남아 합이 최대 칸
// 수의 절반 EMU만큼 어긋나고, 그 어긋남은 마지막 열이 프레임을 삐져나오는 모양으로 나타난다.
//
// 뒤로 가는 경계는 0 길이로 눕힌다 — 측정 잡음이 음수 `w`를 만들면 파일이 열리지 않는다.
func edgeLengths(edges []float64) []int64 {
	lengths := make([]int64, 0, len(edges)-1)
	at := EMU(edges[0])
	for _, edge := range edges[1:] {
		next := EMU(edge)
		if next < at {
			next = at
		}
		lengths = append(lengths, next-at)
		at = next
	}
	return lengths
}

// writeGraphicFrame는 표 하나의 골격을 쓴다 — nvGraphicFramePr(id·이름·플레이스홀더) →
// p:xfrm(좌표) → a:graphic > a:graphicData > a:tbl.
//
// 자식 순서는 CT_GraphicalObjectFrame이 정한 그대로다 (writeSp·writePic과 같은 종류의 제약).
// **`p:xfrm`은 `a:xfrm`이 아니다** — 같은 좌표를 담지만 이름공간이 다르고, 회전·대칭 속성도
// 여기서는 쓰지 않는다 (표에 `rotate`/`flip` 어휘가 없다 — soml.tableOwnAttrs).
func writeGraphicFrame(b *strings.Builder, spec tableInfo, theme soml.Theme, rels *slideRels) error {
	x, y := Offset(spec.rect)
	cx, cy := Extent(spec.rect)

	w := func(depth int, line string) {
		b.WriteString("\n" + shapeIndent + strings.Repeat("  ", depth) + line)
	}
	w(0, "<p:graphicFrame>")
	w(1, "<p:nvGraphicFramePr>")
	// 이름·대체 텍스트는 도형·그림과 같은 규칙이다 (지오메트리 식별자 + `descr` — cNvPrXML).
	w(2, cNvPrXML(spec.id, spec.name, spec.descr))
	// `a:graphicFrameLocks noGrp="1"`을 넣지 않는다 — PowerPoint가 표에 그 잠금을 쓰지만,
	// 저자가 아니라 엔진이 잠근 제약은 파일을 열어 본 사람이 이유를 알 수 없다 (그림에
	// `picLocks noChangeAspect`를 넣지 않은 것과 같은 판단 — writePic).
	w(2, "<p:cNvGraphicFramePr/>")
	writeNvPr(w, 2, spec.ph)
	w(1, "</p:nvGraphicFramePr>")
	w(1, "<p:xfrm>")
	w(2, fmt.Sprintf(`<a:off x="%d" y="%d"/>`, x, y))
	w(2, fmt.Sprintf(`<a:ext cx="%d" cy="%d"/>`, cx, cy))
	w(1, "</p:xfrm>")
	w(1, "<a:graphic>")
	w(2, fmt.Sprintf(`<a:graphicData uri="%s">`, tableGraphicDataURI))
	if err := writeTbl(w, 3, spec, theme, rels); err != nil {
		return err
	}
	w(2, "</a:graphicData>")
	w(1, "</a:graphic>")
	w(0, "</p:graphicFrame>")
	return nil
}

// writeTbl는 `a:tbl`을 쓴다 — 표 속성(스타일 없음 + 머리행 힌트) → 열 격자 → 행들.
func writeTbl(w func(depth int, line string), depth int, spec tableInfo, theme soml.Theme, rels *slideRels) error {
	if len(spec.cols) == 0 {
		return fmt.Errorf("열이 없다 — a:tblGrid는 열 하나 이상을 요구한다")
	}
	w(depth, "<a:tbl>")
	// firstRow는 **힌트**다: 스타일이 없으므로 그림을 바꾸지 않고, PowerPoint의 "머리글 행"
	// 체크와 스크린 리더의 머리 셀 인식이 이 값을 읽는다 (웹에서 `<th>`가 하는 일의 절반).
	// 줄무늬 힌트(bandRow)는 켜지 않는다 — 줄무늬 색은 셀 채움에 이미 구워져 있고,
	// 힌트까지 켜면 저자가 뷰어에서 표 스타일을 바꾼 순간 줄무늬가 두 벌로 겹친다.
	attrs := ""
	if spec.firstRow {
		attrs = ` firstRow="1"`
	}
	w(depth+1, "<a:tblPr"+attrs+">")
	// 스키마 순서(CT_TableProperties): 채움 → 효과 → tableStyle|tableStyleId.
	w(depth+2, fmt.Sprintf("<a:tableStyleId>%s</a:tableStyleId>", tableNoStyleID))
	w(depth+1, "</a:tblPr>")
	w(depth+1, "<a:tblGrid>")
	for _, width := range spec.cols {
		w(depth+2, fmt.Sprintf(`<a:gridCol w="%d"/>`, width))
	}
	w(depth+1, "</a:tblGrid>")
	for _, row := range spec.rows {
		w(depth+1, fmt.Sprintf(`<a:tr h="%d">`, row.h))
		for _, cell := range row.cells {
			if err := writeTc(w, depth+2, cell, theme, rels); err != nil {
				return err
			}
		}
		w(depth+1, "</a:tr>")
	}
	w(depth, "</a:tbl>")
	return nil
}

// writeTc는 격자 칸 하나를 `a:tc`로 쓴다 — 저자가 쓴 셀이면 텍스트와 명시 `tcPr`를, 병합에
// 덮인 칸이면 `hMerge`/`vMerge` 표시와 빈 본문을 낸다.
//
// 덮인 칸을 비워 두는 이유: 그 칸은 그려지지 않으므로 외관을 실을 자리가 아니고(병합 영역의
// 테두리·채움은 좌상단 셀이 소유한다), 값을 실으면 병합을 무시하는 소비자에서 **유령 셀**이
// 보인다. 본문을 아예 내지 않는 길도 있었지만(txBody는 선택 자식이다) PowerPoint가 내는 표기를
// 따른다 — 빈 문단 하나가 있어야 저자가 병합을 풀었을 때 그 칸의 서식이 뷰어 기본값에서 오지
// 않는다.
func writeTc(w func(depth int, line string), depth int, cell tableCellInfo, theme soml.Theme, rels *slideRels) error {
	// 속성 순서는 웹의 `colspan`/`rowspan` 순서를 따른다 (스키마는 순서를 묻지 않는다) —
	// 두 산출물을 나란히 읽을 때 같은 값이 같은 순서로 보여야 한다. 1은 적지 않는다
	// (기본값이 1이다 — 웹이 `colspan="1"`을 내지 않는 것과 같은 태도).
	attrs := ""
	if cell.colSpan > 1 {
		attrs += fmt.Sprintf(` gridSpan="%d"`, cell.colSpan)
	}
	if cell.rowSpan > 1 {
		attrs += fmt.Sprintf(` rowSpan="%d"`, cell.rowSpan)
	}
	if cell.hMerge {
		attrs += ` hMerge="1"`
	}
	if cell.vMerge {
		attrs += ` vMerge="1"`
	}
	w(depth, "<a:tc"+attrs+">")
	if cell.covered() {
		w(depth+1, "<a:txBody>")
		w(depth+2, "<a:bodyPr/>")
		w(depth+2, "<a:lstStyle/>")
		w(depth+2, "<a:p/>")
		w(depth+1, "</a:txBody>")
		w(depth+1, "<a:tcPr/>")
		w(depth, "</a:tc>")
		return nil
	}
	if cell.style.Text == nil {
		return fmt.Errorf("%s <%s>의 텍스트 외관이 해석되지 않았다 — core.Resolve를 거치지 않은 덱이다",
			cell.elem.Span, cell.elem.Name)
	}
	w(depth+1, "<a:txBody>")
	// **셀의 bodyPr은 비어 있다.** 여백과 세로 앵커가 `a:tcPr`에 사는 것이 표와 텍스트 박스가
	// 갈라지는 자리다 (텍스트 박스는 bodyPr에 인셋 0과 anchor를 싣는다 — textBodyXML): 같은
	// 값을 두 곳에 적으면 뷰어가 둘을 더하는 순간 글자가 여백 두 벌만큼 밀린다. 비워 두는 것이
	// PowerPoint가 내는 표기이기도 하다.
	w(depth+2, "<a:bodyPr/>")
	w(depth+2, "<a:lstStyle/>")
	// 셀 안 문단은 블록 텍스트와 **같은 조립기**를 지난다 — 깊이만 옮긴다: writeTextParagraph는
	// 텍스트 박스 기준으로 `<a:p>`를 깊이 2에 쓰므로, 여기서 원하는 깊이(depth+2)만큼 밀어 준다.
	pw := func(d int, line string) { w(d+depth, line) }
	if err := writeTextParagraph(pw, cell.elem, *cell.style.Text, theme, rels); err != nil {
		return err
	}
	w(depth+1, "</a:txBody>")
	if err := writeTcPr(w, depth+1, *cell.style, theme); err != nil {
		return err
	}
	w(depth, "</a:tc>")
	return nil
}

// writeTcPr는 셀의 확정 외관을 `a:tcPr`로 쓴다 — 여백·세로 앵커는 속성, 테두리 네 변과 채움은
// 자식이다.
//
// **스키마 순서가 도형과 반대다** (CT_TableCellProperties): 선(lnL·lnR·lnT·lnB)이 먼저고 채움이
// 나중이다 — `spPr`은 채움이 먼저고 선이 나중이다 (writeSp). 어기면 LibreOffice가 복구를 묻는다.
//
// 여백은 웹의 셀 padding과 **같은 상수**다 (web.TableCellPadX/Y): 두 벌이 되면 글자 시작점이
// 갈라지고 그 차이는 두 산출물을 나란히 볼 때까지 드러나지 않는다.
func writeTcPr(w func(depth int, line string), depth int, style soml.CellStyle, theme soml.Theme) error {
	anchor, err := bodyAnchorValue(style.Text.Anchor)
	if err != nil {
		return err
	}
	w(depth, fmt.Sprintf(`<a:tcPr marL="%d" marR="%d" marT="%d" marB="%d" anchor="%s">`,
		EMU(web.TableCellPadX), EMU(web.TableCellPadX),
		EMU(web.TableCellPadY), EMU(web.TableCellPadY), anchor))
	// 테두리는 네 변에 같은 값을 싣는다 — 어휘가 변별 테두리를 갖지 않기 때문이다
	// (`border` 하나가 셀 전체다 — soml.CellStyle.Border). 이웃 셀이 같은 격자선을 두 번
	// 그리지만 겹치는 선은 같은 색·굵기라 한 줄로 보인다 (웹의 `border-collapse: collapse`와
	// 같은 그림이다).
	border := soml.NoBorder()
	if style.Border != nil {
		border = *style.Border
	}
	for _, side := range []string{"a:lnL", "a:lnR", "a:lnT", "a:lnB"} {
		if err := writeLineElem(w, depth+1, side, border, nil, theme); err != nil {
			return err
		}
	}
	// 채움을 정하지 않은 셀은 **채움 없음**이다 (웹에서 `background-color` 선언이 없는 것과
	// 같은 그림): 비워 두면 표 스타일이 그 자리를 채울 여지가 남는다.
	fill := soml.NoFill()
	if style.Fill != nil {
		fill = *style.Fill
	}
	if err := writeFill(w, depth+1, fill, nil, theme); err != nil {
		return err
	}
	w(depth, "</a:tcPr>")
	return nil
}
