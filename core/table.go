package core

import "github.com/k2sebeom/balpyo/soml"

// 이 파일은 **표 외관의 값 해석**이다 (PRD §7 · ADR 0036) — 파서가 이름만 본 표 프레임·행·셀을
// 확정된 표 외관(soml.TableStyle·CellStyle)으로 눕힌다.
//
// 도형·이미지 해석(resolveShape·resolveImage)과 나란한 다섯 번째 갈래이되 **서브트리를 통째로**
// 지나는 점이 갈라진다: 프레임(`<table>`) 하나가 아니라 그 아래 행(`<tr>`)·셀(`<th>`/`<td>`)·
// 셀 안 인라인 런까지 한 번에 확정한다. 이유는 `<tr>`/`<th>`/`<td>`가 텍스트 역할을 갖지 않아
// (soml/table.go) node()의 텍스트 갈래가 셀 안 인라인 런을 데려가지 못하기 때문이다 —
// resolveInlines는 텍스트 확정을 끝낸 자리에서만 불린다.
//
// **셀 캐스케이드**는 텍스트·도형과 같은 기계를 쓴다 (mergeCell — cascade.go). 바닥은 body 역할
// 기본값(셀은 본문 텍스트다)이고 그 위에 테마 표 컴포넌트의 세 단이 얹힌다:
//
//	인라인 셀 속성 > (헤더면 header 델타 | 줄무늬면 stripe 채움) > 셀 기본값 > body 역할
//
// 프레임 변형(줄무늬)은 셀마다 다른 variant가 아니라 프레임 하나가 켜는 **행 채움**이다:
// `<table variant="striped">`가 홀수 본문 행의 비헤더 셀에 stripe 채움을 얹는다. 줄무늬를
// PPTX에서 명시 tcPr 채움으로 굽는 것이 ADR 0036의 "스타일 없음 + 명시 tcPr"이다 — 표 스타일
// 상속에 기대면 뷰어마다 다른 색이 나온다.

// tableStripedVariant는 표 프레임이 아는 유일한 변형 이름이다 (v1). 줄무늬는 셀 variant가
// 아니라 프레임이 켜는 행 채움이라(위 주석) 이름이 하나뿐이고, 그래서 상수로 둔다.
const tableStripedVariant = "striped"

// tableVariantNames는 표 프레임 변형 어휘다 — 모르는 variant 경고가 나열하는 목록이다
// (shapeVariantNames·textVariantNames와 같은 태도, 요소 종류에 맞는 어휘를 낸다).
func tableVariantNames() []string { return []string{tableStripedVariant} }

// resolveTable은 `<table>` 서브트리의 외관을 확정한다 — 프레임 변형, 행별 줄무늬, 셀 외관,
// 셀 안 인라인 런까지.
//
// 프레임 자체에 실을 외관은 변형 이름뿐이다 (좌표는 잰 박스가, 셀은 CellStyle이 소유한다 —
// soml.TableStyle). 모르는 variant는 에러가 아니라 경고 + 빈 값이다 (도형·텍스트와 같은 정책 —
// 콘텐츠의 테마 간 이식성, PRD §13-6).
func (r *resolver) resolveTable(elem *soml.Element) {
	style := soml.TableStyle{}
	striped := false
	if attr, ok := r.variantAttr(elem); ok {
		if attr.Value == tableStripedVariant {
			style.Variant = attr.Value
			striped = true
		} else {
			// 프레임 변형 어휘 밖 — 도형·텍스트와 같은 문구·정책으로 경고하고 줄무늬 없이 계속한다.
			r.warnUnknownVariant(attr, tableVariantNames())
		}
	}
	elemAppearance(elem).Table = &style

	// 본문 행 색인으로 얼룩(zebra)을 센다 — 헤더 행은 세지 않는다. 헤더 행은 **모든 셀이
	// `<th>`인 행**이다: 그래야 열 머리(첫 행 전체가 th)는 줄무늬에서 빠지고, 행 머리(각 행의
	// 첫 칸만 th)는 본문 행으로 세어 그 데이터 셀이 줄무늬를 받는다.
	bodyRow := 0
	rows := tableRows(elem)
	spansValid := true
	for _, row := range rows {
		header := headerRow(row)
		// 홀수 본문 행에 얼룩을 얹는다 (0-index의 1,3,5… — 첫 본문 행은 맨바탕, 그다음이 얼룩).
		stripe := striped && !header && bodyRow%2 == 1
		for _, cell := range tableCells(row) {
			if !r.resolveCell(cell, stripe) {
				spansValid = false
			}
		}
		if !header {
			bodyRow++
		}
	}

	// 격자 검증은 셀 span이 다 유효할 때만 한다 — `colspan="0"`이 이미 에러인 자리에 "행이
	// 좁다"를 얹으면 저자는 고칠 곳이 둘이라고 읽는다 (진단 하나에 원인 하나).
	if spansValid {
		r.checkTableGrid(rows)
	}
}

// tableRows는 표 프레임의 행 자식을, tableCells는 행의 셀 자식을 문서 순서로 낸다. 콘텐츠
// 모델은 파서가 이미 지켰으므로(soml/table.go) 여기서는 걸러내기만 한다 — 주석·문자 데이터가
// 사이에 있어도 격자 계산이 흔들리지 않게.
func tableRows(table *soml.Element) []*soml.Element {
	var rows []*soml.Element
	for _, node := range table.Children {
		if row, ok := node.(*soml.Element); ok && row.Name == soml.TableRowElement {
			rows = append(rows, row)
		}
	}
	return rows
}

func tableCells(row *soml.Element) []*soml.Element {
	var cells []*soml.Element
	for _, node := range row.Children {
		if cell, ok := node.(*soml.Element); ok && soml.IsTableCell(cell.Name) {
			cells = append(cells, cell)
		}
	}
	return cells
}

// checkTableGrid은 병합이 직사각 격자를 정확히 덮는지 검증한다 (ADR 0036이 값 층에 맡긴 일).
//
// **왜 값 층인가**: 두 타깃이 깨진 격자를 다르게 복구한다. 웹 `<table>`은 칸이 남으면 조용히
// 비우고 넘치면 열을 늘리지만, PPTX `a:tbl`은 `a:tblGrid`의 열 수와 행마다의 `a:tc` 수
// (`gridSpan`·`hMerge`·`vMerge` 포함)가 **정확히** 맞아야 열린다. 여기서 막지 않으면 웹만 보고
// 넘긴 표가 PowerPoint에서 복구 대화상자를 띄운다 — 두 렌더러가 하나의 확정 모델을 읽는다는
// 약속(PRD §7)은 그 모델이 격자로서 성립할 때만 지켜진다.
//
// 규칙은 HTML 표와 같다: **첫 행이 열 수를 정하고**, 이후 행은 위에서 내려온 rowspan이 덮은
// 칸을 건너뛰며 왼쪽부터 채운다. 열 수를 첫 행이 정하는 이유는 저자가 읽는 순서이기 때문이다
// (최대 행에서 역산하면 어느 행이 틀렸는지 말해 줄 수 없다).
func (r *resolver) checkTableGrid(rows []*soml.Element) {
	if len(rows) == 0 {
		return
	}
	cols := 0
	for _, cell := range tableCells(rows[0]) {
		cols += cellSpans(cell).col
	}
	if cols == 0 {
		r.errorf(rows[0].NameSpan, "표의 첫 행에 셀이 없다 — 첫 행이 표의 열 수를 정한다")
		return
	}

	// spill[c]는 c번째 열이 위 행들의 rowspan에 앞으로 몇 행 더 덮이는지다.
	spill := make([]int, cols)
	for i, row := range rows {
		covered := 0
		for _, n := range spill {
			if n > 0 {
				covered++
			}
		}
		cells := tableCells(row)
		filled := covered
		for _, cell := range cells {
			filled += cellSpans(cell).col
		}
		if filled != cols {
			// 행 하나에 진단 하나 — 어느 셀이 틀렸는지는 값 층이 알 수 없다 (칸 수만 안다).
			r.errorf(row.NameSpan, "이 행은 %d칸인데 표의 열은 %d칸이다 — 첫 행이 열 수를 정한다 (셀이나 colspan을 맞춘다)", filled, cols)
			decaySpill(spill)
			continue
		}

		// 칸 수가 맞으면 왼쪽부터 자리를 준다 — 덮인 칸을 건너뛰다가 colspan이 그 칸을 지나가면
		// 직사각형이 아니다 (칸 수만 세면 이 경우를 놓친다).
		col := 0
		overlap := false
		next := make([]int, cols)
		for _, cell := range cells {
			for col < cols && spill[col] > 0 {
				col++
			}
			cs := cellSpans(cell)
			for c := col; c < col+cs.col && c < cols; c++ {
				if spill[c] > 0 {
					r.errorf(cell.NameSpan, "병합이 겹친다 — %d번째 열은 위 행의 rowspan이 이미 덮었다 (colspan %d이 그 칸을 지난다)", c+1, cs.col)
					overlap = true
					break
				}
			}
			if overlap {
				break
			}
			if cs.row > 1 {
				if rest := len(rows) - 1 - i; cs.row-1 > rest {
					r.errorf(cell.NameSpan, "rowspan %d은 표의 마지막 행을 넘는다 — 이 셀 아래 남은 행은 %d개다", cs.row, rest)
				}
				for c := col; c < col+cs.col && c < cols; c++ {
					// 남은 행 수로 깎아 둔다 — 넘친 rowspan이 아래 행의 칸 수까지 어긋나게 해
					// 진단이 연쇄하지 않도록.
					next[c] = min(cs.row-1, len(rows)-1-i)
				}
			}
			col += cs.col
		}
		decaySpill(spill)
		if overlap {
			continue
		}
		for c, n := range next {
			if n > spill[c] {
				spill[c] = n
			}
		}
	}
}

// decaySpill은 rowspan이 덮는 남은 행 수를 한 행 줄인다 (행을 하나 지났다).
func decaySpill(spill []int) {
	for c, n := range spill {
		if n > 0 {
			spill[c] = n - 1
		}
	}
}

// gridSpan은 셀 하나가 격자에서 차지하는 칸 수다. cellSpans는 확정 CellStyle에서 읽는다 —
// 속성 해석(cellSpan)이 이미 지난 값이라 격자 검증이 문자열을 다시 읽지 않는다.
type gridSpan struct{ col, row int }

func cellSpans(cell *soml.Element) gridSpan {
	if cs := cell.CellStyle(); cs != nil {
		return gridSpan{col: cs.ColSpan, row: cs.RowSpan}
	}
	return gridSpan{col: 1, row: 1}
}

// headerRow는 행의 모든 셀이 `<th>`인지 본다 (빈 행은 헤더가 아니다). 얼룩 계산이 이 술어로
// 열 머리(줄무늬 제외)와 본문 행(줄무늬 대상)을 가른다 — resolveTable의 주석에 근거가 있다.
func headerRow(row *soml.Element) bool {
	cells := 0
	for _, node := range row.Children {
		cell, ok := node.(*soml.Element)
		if !ok || !soml.IsTableCell(cell.Name) {
			continue
		}
		cells++
		if !soml.IsHeaderCell(cell.Name) {
			return false
		}
	}
	return cells > 0
}

// resolveCell은 셀 하나(`<th>`/`<td>`)의 외관을 셀 캐스케이드로 확정한다.
//
//	인라인 셀 속성 > (헤더 델타 | 줄무늬 채움) > 셀 기본값 > body 역할 기본값
//
// 바닥이 body 역할 기본값인 이유는 셀이 본문 텍스트이기 때문이다 (텍스트 요소가 역할 기본값을
// 바닥에 까는 것과 같은 자리 — resolveText). 병합은 mergeCell 한 곳에서만 일어나고 우선순위는
// 아래 병합 순서가 전부다. 헤더와 줄무늬는 **배타**다: `<th>`는 헤더 델타를, 비헤더 줄무늬 셀은
// stripe 채움을 받는다 (헤더에 줄무늬를 겹치면 헤더 강조색이 행마다 흔들린다).
//
// 돌려주는 값은 **병합 칸 수가 유효했는지**다 — 격자 검증(checkTableGrid)이 이 신호를 보고,
// span 값 자체가 틀린 표에서는 격자 진단을 덧붙이지 않는다.
func (r *resolver) resolveCell(cell *soml.Element, stripe bool) (spansValid bool) {
	header := soml.IsHeaderCell(cell.Name)
	cs := &soml.CellStyle{Header: header, ColSpan: 1, RowSpan: 1}
	// 셀 텍스트의 바닥 = body 역할 기본값. mergeCell이 이 Text에 색·정렬·앵커·굵기를 얹는다.
	text := r.textRoleDefaults(soml.RoleBody)
	cs.Text = &text

	mergeCell(cs, r.comps.table.cell)
	if header {
		mergeCell(cs, r.comps.table.header)
	} else if stripe && r.comps.table.stripe != nil {
		mergeCell(cs, cellPatch{fill: r.comps.table.stripe})
	}
	patch, spansValid := r.cellAttrs(cell, cs)
	mergeCell(cs, patch)
	elemAppearance(cell).Cell = cs

	// 셀 안 인라인 런(strong/em/code/a/span)은 셀 텍스트를 상속한다 — 텍스트 요소의 인라인
	// 해석과 같은 기계다 (resolveInlines). 텍스트 전용 콘텐츠 모델이라(ADR 0036) 셀 아래에는
	// 문자 데이터와 인라인 태그만 있다.
	r.resolveInlines(cell, cs.Text)
	return spansValid
}

// cellAttrs는 셀 속성 한 벌을 훑어 **인라인 외관 패치**를 만든다 (shapeAttrs의 셀 판).
//
// 병합 칸 수(colspan/rowspan)는 패치가 아니라 cs에 직접 채운다 — 캐스케이드가 다루는 값이
// 아니라 격자 구조이기 때문이다 (shapeAttrs가 id·박스를 직접 채우는 것과 같은 근거,
// cellPatch의 주석). 값 어휘 검사·진단은 속성별 해석 함수(fill/border/color/align/anchor)가
// 그대로 소유한다 — 도형과 같은 함수를 지나 같은 값이 자리마다 다르게 읽히지 않는다.
//
// 두 번째 반환값은 병합 칸 수가 다 유효했는지다 (resolveCell이 격자 검증에 전한다).
func (r *resolver) cellAttrs(cell *soml.Element, cs *soml.CellStyle) (cellPatch, bool) {
	var patch cellPatch
	spansValid := true
	for _, attr := range cell.Attrs {
		switch attr.Name {
		case "colspan":
			if n, ok := r.cellSpan(attr); ok {
				cs.ColSpan = n
			} else {
				spansValid = false
			}
		case "rowspan":
			if n, ok := r.cellSpan(attr); ok {
				cs.RowSpan = n
			} else {
				spansValid = false
			}
		case "fill":
			if fill, ok := r.fill(attr); ok {
				patch.fill = &fill
			}
		case "border":
			if border, ok := r.border(attr); ok {
				patch.border = &border
			}
		case "color":
			if color, ok := r.color(attr); ok {
				patch.color = &color
			}
		case "align":
			if align, ok := r.align(attr); ok {
				patch.align = &align
			}
		case soml.AnchorAttr:
			if anchor, ok := r.anchor(attr); ok {
				patch.anchor = &anchor
			}
		default:
			// 셀 어휘 밖 속성이 값 층에 닿았다면 카탈로그(soml)와 값 층(core)이 어긋난 것이다 —
			// 조용히 지나가면 저자가 쓴 외관이 사라진다 (shapeAttrs의 catalogGap과 같은 태도).
			r.catalogGap(attr)
		}
	}
	return patch, spansValid
}

// cellSpan은 `colspan`/`rowspan`을 해석한다 — 1 이상 정수다 (span의 grid 검증과 같은 하한,
// 부모를 볼 필요가 없어 더 단순하다). 0·음수는 병합 칸이 아니라 격자를 망가뜨리는 값이므로
// 막는다 (span이 0을 막는 것과 같은 근거).
func (r *resolver) cellSpan(attr soml.Attr) (int, bool) {
	n, ok := r.integer(attr)
	if !ok {
		return 0, false
	}
	if n < 1 {
		r.errorf(attr.ValueSpan, "%s는 1 이상 정수다 (%q)", attr.Name, attr.Value)
		return 0, false
	}
	return n, true
}
