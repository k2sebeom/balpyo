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
	for _, node := range elem.Children {
		row, ok := node.(*soml.Element)
		if !ok || row.Name != soml.TableRowElement {
			continue
		}
		header := headerRow(row)
		// 홀수 본문 행에 얼룩을 얹는다 (0-index의 1,3,5… — 첫 본문 행은 맨바탕, 그다음이 얼룩).
		stripe := striped && !header && bodyRow%2 == 1
		for _, cellNode := range row.Children {
			cell, ok := cellNode.(*soml.Element)
			if !ok || !soml.IsTableCell(cell.Name) {
				continue
			}
			r.resolveCell(cell, stripe)
		}
		if !header {
			bodyRow++
		}
	}
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
func (r *resolver) resolveCell(cell *soml.Element, stripe bool) {
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
	mergeCell(cs, r.cellAttrs(cell, cs))
	elemAppearance(cell).Cell = cs

	// 셀 안 인라인 런(strong/em/code/a/span)은 셀 텍스트를 상속한다 — 텍스트 요소의 인라인
	// 해석과 같은 기계다 (resolveInlines). 텍스트 전용 콘텐츠 모델이라(ADR 0036) 셀 아래에는
	// 문자 데이터와 인라인 태그만 있다.
	r.resolveInlines(cell, cs.Text)
}

// cellAttrs는 셀 속성 한 벌을 훑어 **인라인 외관 패치**를 만든다 (shapeAttrs의 셀 판).
//
// 병합 칸 수(colspan/rowspan)는 패치가 아니라 cs에 직접 채운다 — 캐스케이드가 다루는 값이
// 아니라 격자 구조이기 때문이다 (shapeAttrs가 id·박스를 직접 채우는 것과 같은 근거,
// cellPatch의 주석). 값 어휘 검사·진단은 속성별 해석 함수(fill/border/color/align/anchor)가
// 그대로 소유한다 — 도형과 같은 함수를 지나 같은 값이 자리마다 다르게 읽히지 않는다.
func (r *resolver) cellAttrs(cell *soml.Element, cs *soml.CellStyle) cellPatch {
	var patch cellPatch
	for _, attr := range cell.Attrs {
		switch attr.Name {
		case "colspan":
			if n, ok := r.cellSpan(attr); ok {
				cs.ColSpan = n
			}
		case "rowspan":
			if n, ok := r.cellSpan(attr); ok {
				cs.RowSpan = n
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
	return patch
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
