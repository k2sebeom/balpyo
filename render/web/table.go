package web

import (
	"fmt"
	"strings"

	"github.com/k2sebeom/balpyo/soml"
)

// 이 파일은 **표의 웹 렌더**다 — 프레임은 진짜 `<table>`, 행은 `<tr>`, 셀은
// `<th>`/`<td>`다.
//
// CSS 그리드로 흉내지 않은 이유는 두 가지다: (1) 스크린 리더가 행·열 관계를 읽는다 (시맨틱
// 블록 태그를 고른 것과 같은 근거 — writeBlock), (2) 브라우저의 표 알고리즘이 정한 **열 너비와
// 행 높이를 그대로 잰다**. 그 잰 값이 PPTX `a:tblGrid`의 열 너비와 `a:tr@h`가 되므로(지오메트리
// 계약), 격자 기하를 우리가 계산하지 않는다 — 텍스트 줄바꿈을 브라우저에 맡긴 것과 같은 선택이다.
//
// **셀 외관은 전부 인라인 style이다.** 브라우저 기본값(`th { font-weight: bold; text-align:
// center }`, `td { vertical-align: middle }`)이나 표 스타일 상속이 셀 모양을 정할 자리를 남기지
// 않는다 — PPTX 쪽에서 표 스타일을 가리키지 않고 `tcPr`을 명시로 굽는 것의 웹 절반이고,
// 두 타깃이 같은 CellStyle 하나를 읽는다는 계약(soml.CellStyle)의 실물이다.

// 셀 여백이다 (논리 px). export하는 이유는 ListTextIndent와 같다: PPTX 렌더러가 `a:tcPr`의
// `marL`/`marR`/`marT`/`marB`에 **같은 값**을 실어야 두 타깃의 글자가 같은 자리에서 시작한다.
// 값이 두 벌이 되면 갈라짐이 두 산출물을 나란히 볼 때까지 드러나지 않는다.
//
// 테마가 아니라 엔진 상수인 이유는 테마 `table` 컴포넌트가 여백을 갖지 않기 때문이다
// (core/themes/minimal/components.json — 채움·테두리·글자만 테마의 몫이다). 값은 px이므로
// EMU 환산(×9525)이 정수로 떨어진다 (12px = 114300, 8px = 76200).
const (
	// TableCellPadX는 셀 좌우 여백이다 — 스택 간격(24px)의 절반으로, 글자와 셀 선 사이가
	// 한 글자 폭의 절반만큼 벌어진다.
	TableCellPadX = 12
	// TableCellPadY는 셀 상하 여백이다. 좌우보다 좁다 — 행 높이는 줄높이가 이미 소유하므로
	// (역할 기본값의 lineHeight) 여기서 같은 리듬을 두 번 더하지 않는다.
	TableCellPadY = 8
)

// writeTable은 표 프레임 하나를 쓴다 (`<table>` → `<table>`).
//
// 프레임에 실리는 것은 변형 훅과 **배치**뿐이다: 표는 크기 속성을 갖지 않고(soml.tableOwnAttrs —
// 잰 내용이 곧 크기다) 외관은 셀이 소유한다. 반면 배치는 갖는다 — 표는 레이아웃 컨테이너의
// 자식이자 슬라이드 직계 자식이 될 수 있으므로 리스트와 같은 코드(appendSlot·appendPlacement)를
// 지난다.
func writeTable(b *strings.Builder, deck *soml.Deck, slide *soml.Slide, elem *soml.Element, path []int) error {
	style := elem.TableStyle()
	if style == nil {
		// 해석되지 않은 외관을 기본값으로 그리면 웹만 그럴듯하게 나오고 PPTX와 갈라진다
		// (writeBlock과 같은 계약).
		return fmt.Errorf("%s:%s <%s> appearance is unresolved — the deck did not go through core.Resolve",
			slide.Source, elem.Span, elem.Name)
	}
	class := "balpyo-table"
	if style.Variant != "" {
		// 변형은 CSS 훅으로만 나간다 — 줄무늬 **색**은 core가 셀 채움에 이미 구워 놓았다
		// 클래스는 저자 CSS(@layer balpyo.author)의 표적이다.
		class += " balpyo-table-" + style.Variant
	}
	decls, err := appendPlacement(appendSlot(nil, elem), elem.Placement())
	if err != nil {
		return fmt.Errorf("%s:%s <%s>: %w", slide.Source, elem.Span, elem.Name, err)
	}
	// 저자의 `alt`은 **표의 이름**이다 — 셀은 그대로 항해 가능하다 (labelAttr).
	fmt.Fprintf(b, "<table class=%q %s=%q %s=%q%s%s%s>\n",
		class, IDAttr, ElementID(slide.Index, path), ElAttr, elem.Name,
		labelAttr(style.Alt), enterAttr(elem), styleAttr(decls))
	for i, child := range elem.Children {
		row, ok := child.(*soml.Element)
		if !ok || row.Name != soml.TableRowElement {
			// 콘텐츠 모델은 파서가 지켰다 (tableChildren) — 여기까지 온 다른 노드는 파서를
			// 거치지 않은 덱이므로 조용히 버리지 않는다.
			return fmt.Errorf("%s:%s <table> child %T is not a row — the deck did not go through the parser",
				slide.Source, elem.Span, child)
		}
		if err := writeTableRow(b, deck, slide, row, ChildPath(path, i)); err != nil {
			return err
		}
	}
	b.WriteString("</table>\n")
	return nil
}

// writeTableRow는 행 하나를 쓴다. 행은 외관을 갖지 않는다 (줄무늬조차 셀 채움이다) —
// 그래도 식별자를 갖는 이유는 **행 높이**다: PPTX `a:tr@h`가 잰 이 박스의 높이에서 온다.
func writeTableRow(b *strings.Builder, deck *soml.Deck, slide *soml.Slide, row *soml.Element, path []int) error {
	fmt.Fprintf(b, "<tr %s=%q %s=%q>\n", IDAttr, ElementID(slide.Index, path), ElAttr, row.Name)
	for i, child := range row.Children {
		cell, ok := child.(*soml.Element)
		if !ok || !soml.IsTableCell(cell.Name) {
			return fmt.Errorf("%s:%s <tr> child %T is not a cell — the deck did not go through the parser",
				slide.Source, row.Span, child)
		}
		if err := writeTableCell(b, deck, slide, cell, ChildPath(path, i)); err != nil {
			return err
		}
	}
	b.WriteString("</tr>\n")
	return nil
}

// writeTableCell은 셀 하나를 쓴다 — 저자가 쓴 태그 그대로(`<th>`/`<td>`) 낸다.
//
// 태그를 가르는 이유는 접근성이다 (`<th>`가 행·열 머리를 읽게 한다). 외관은 태그가 아니라
// 확정 CellStyle이 정하므로 UA의 `th` 기본값에 기대는 값이 하나도 없다 — 폰트·굵기·색·정렬은
// 인라인 style이 전부 명시한다 (textStyleCSS).
func writeTableCell(b *strings.Builder, deck *soml.Deck, slide *soml.Slide, cell *soml.Element, path []int) error {
	cs := cell.CellStyle()
	if cs == nil {
		return fmt.Errorf("%s:%s <%s> appearance is unresolved — the deck did not go through core.Resolve",
			slide.Source, cell.Span, cell.Name)
	}
	decls, err := cellStyleCSS(*cs, deck.Theme)
	if err != nil {
		return fmt.Errorf("%s:%s <%s>: %w", slide.Source, cell.Span, cell.Name, err)
	}
	// 병합은 HTML 속성이다 — 브라우저의 표 알고리즘이 격자를 그것으로 짜고, 그렇게 나온 열
	// 너비·행 높이를 우리가 잰다. 1은 적지 않는다 (기본값을 적으면 뜻 없는 속성이 골든을
	// 붐비게 한다 — 배치 선언의 nil 계약과 같은 태도).
	var spans string
	if cs.ColSpan > 1 {
		spans += fmt.Sprintf(" colspan=\"%d\"", cs.ColSpan)
	}
	if cs.RowSpan > 1 {
		spans += fmt.Sprintf(" rowspan=\"%d\"", cs.RowSpan)
	}
	fmt.Fprintf(b, "<%s %s=%q %s=%q%s%s>",
		cell.Name, IDAttr, ElementID(slide.Index, path), ElAttr, cell.Name, spans, styleAttr(decls))
	for i, child := range cell.Children {
		// 셀의 콘텐츠 모델은 문자 데이터와 인라인뿐이다 — writeNode가 그 둘을 이미
		// 다루므로 같은 코드를 지난다 (인라인 서식이 셀에서만 다르게 나갈 길을 막는다).
		if err := writeNode(b, deck, slide, child, ChildPath(path, i)); err != nil {
			return err
		}
	}
	fmt.Fprintf(b, "</%s>\n", cell.Name)
	return nil
}

// cellStyleCSS는 확정 셀 외관을 CSS 선언 목록으로 컴파일한다 — 글자(textStyleCSS) + 채움 +
// 테두리 + 세로 앵커다.
//
// 순서는 소비자가 읽는 순서다: 글자 → 박스(채움·테두리) → 세로 정렬. `none`은 **선언 없음이
// 아니라 명시**로 낸다 (`background-color: transparent`·`border-style: none`) — 테마가 얹은
// 헤더 채움·셀 테두리를 지우는 유일한 방법이므로, 선언을 내지 않으면 저자가 지운 것이 조용히
// 되살아난다.
func cellStyleCSS(cs soml.CellStyle, theme soml.Theme) ([]string, error) {
	if cs.Text == nil {
		return nil, fmt.Errorf("cell text appearance is unresolved")
	}
	text, err := textStyleCSS(*cs.Text, theme)
	if err != nil {
		return nil, err
	}
	decls := []string{text}
	if cs.Fill != nil {
		if cs.Fill.None {
			decls = append(decls, "background-color: transparent")
		} else {
			color, err := colorValue(cs.Fill.Color, theme)
			if err != nil {
				return nil, err
			}
			decls = append(decls, "background-color: "+color)
		}
	}
	if cs.Border != nil {
		if cs.Border.None {
			decls = append(decls, "border-style: none")
		} else {
			// 축약(`border`)으로 되돌리지 않는 이유는 도형·그림과 같다 — core가 이미 세 조각으로
			// 풀었고, 다시 합치면 조각 하나가 빠졌을 때 CSS가 조용히 선언 전체를 버린다.
			borderStyle, err := borderStyleValue(cs.Border.Style)
			if err != nil {
				return nil, err
			}
			color, err := colorValue(cs.Border.Color, theme)
			if err != nil {
				return nil, err
			}
			decls = append(decls,
				"border-width: "+formatNumber(cs.Border.Width)+"px",
				"border-style: "+borderStyle,
				"border-color: "+color)
		}
	}
	// 세로 앵커는 셀의 세로 정렬이다 (도형에서는 flex 주축 분배였다 — anchorValue). 언제나
	// 명시한다: UA의 `td { vertical-align: middle }`이 이기면 두 타깃의 글자가 세로로 갈라진다.
	anchor, err := cellAnchorValue(cs.Text.Anchor)
	if err != nil {
		return nil, err
	}
	return append(decls, "vertical-align: "+anchor), nil
}

// cellAnchorValue는 세로 앵커를 CSS `vertical-align` 값으로 만든다.
//
// 도형의 anchorValue(flex 분배 start/center/end)와 갈라지는 이유는 CSS 쪽 어휘다 — 표 셀의
// 세로 정렬 프로퍼티는 `top`/`middle`/`bottom`만 받는다. 어휘 이름은 같은 곳에서 온다
// (soml.TextAnchor).
func cellAnchorValue(anchor soml.TextAnchor) (string, error) {
	switch anchor {
	case soml.AnchorTop:
		return "top", nil
	case soml.AnchorCenter:
		return "middle", nil
	case soml.AnchorBottom:
		return "bottom", nil
	}
	return "", fmt.Errorf("anchor %q is not a known vertical anchor", string(anchor))
}

// tableCSS는 표 배치 CSS다 — 표를 쓴 덱에만 나간다 (listCSS와 같은 골든 불변 규칙).
//
// 여기 있는 것은 **셀 하나로 표현할 수 없는 값**뿐이다: 테두리 합침과 프레임 폭, 그리고 엔진
// 상수인 셀 여백. 나머지(채움·테두리·글자·세로 정렬)는 셀의 인라인 style이 명시한다.
func tableCSS() string {
	return fmt.Sprintf(`
  /* Tables — the frame takes the full content width and the browser's table algorithm decides
     the column widths. Those measured values become the PPTX a:tblGrid column widths, so the
     engine does not compute grid geometry.

     border-collapse: collapse merges the borders of neighboring cells into one line — PPTX a:tc
     borders share a single grid line too, so without it the lines would overlap at 2px on the
     web only. */
  .balpyo-table {
    width: 100%%;
    flex: 0 0 auto;
    border-collapse: collapse;
    table-layout: auto;
  }

  /* Cell padding is an engine constant (TableCellPadX, TableCellPadY) — PPTX a:tcPr mar* uses
     the same values. Inline style states the rest of the cell appearance in full
     (cellStyleCSS). */
  .balpyo-table th,
  .balpyo-table td {
    padding: %dpx %dpx;
  }
`, TableCellPadY, TableCellPadX)
}
