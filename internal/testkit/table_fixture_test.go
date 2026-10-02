package testkit

import (
	"encoding/xml"
	"fmt"
	"math"
	"os"
	"path"
	"testing"

	"github.com/k2sebeom/balpyo/render/geom"
	"github.com/k2sebeom/balpyo/render/pptx"
	"github.com/k2sebeom/balpyo/soml"
)

// 표 픽스처(09_table) 수준의 검증이다. 종류 · 자리 · 셀 글자는 동일성 판정(AssertParity)이
// 이미 본다 — 여기서 보는 것은 컴포넌트 트리에 없는 **표 내부의 OOXML 계약**이다 (ADR 0036):
//
//  1. 격자가 기대한 크기이고 표 스타일을 끈다 (No Style, No Grid · 줄무늬 힌트 없음).
//  2. `a:tbl`의 모든 행은 **열 수만큼의 `a:tc`**를 갖는다 (병합에 덮인 자리는 `hMerge`/`vMerge`
//     빈 셀이다). 이것이 깨진 파일은 PowerPoint가 열 때 복구를 묻는다.
//  3. 셀 외관은 **명시**다 (네 변 + 앵커) — 뷰어가 얹을 값이 없어야 웹과 같은 그림이 된다.
//  4. 격자 수치(`a:gridCol@w`·`a:tr@h`)는 **잰 값**이고 그 합이 프레임 크기와 정확히 같다.
const tableFixtureName = "09_table"

// noStyleNoGridID는 PowerPoint 내장 표 스타일 "No Style, No Grid"의 식별자다.
//
// 이 값을 쓰는 이유는 **스타일을 고르는 것이 아니라 끄는 것**이다: `a:tblPr`에 스타일 참조가
// 아예 없으면 뷰어가 기본 스타일(줄무늬·머리행 색)을 얹고, 그 색은 우리가 셀마다 명시한 값 위에
// 덮인다. 다른 값이 들어오면 그림이 웹과 갈라지므로 전수로 못 박는다.
const noStyleNoGridID = "{5940675A-B579-460E-94D1-54222C63F5DA}"

// tableFixtureTables는 09_table의 슬라이드별 표 기대값이다 (슬라이드 id → 문서 순서의 표들).
//
// 열·행 수를 적는 이유: 병합이 든 표에서 **격자의 크기**는 저자가 쓴 셀 수와 다르다
// (002_merge는 셀 26개가 5열 6행이 된다). 그 수가 골든 XML의 모양으로만 남으면 격자 계산이
// 틀어진 변경이 "diff가 크다"로만 보인다.
//
// firstRow는 힌트다 — 첫 행이 **전부** 헤더 셀일 때만 켠다. 픽스처의 표는 모두 그렇다: 섞인
// 첫 행에서 꺼지는 쪽은 단위 테스트가 본다 (render/pptx.TestRenderTableFirstRowHint) — 픽스처
// 슬라이드를 그 경우만을 위해 늘리지 않는 이유는 힌트가 **그림을 바꾸지 않아서**다 (픽셀 비교가
// 볼 것이 없다).
var tableFixtureTables = map[string][]tableFixtureGrid{
	"table-basic":        {{cols: 4, rows: 4, firstRow: true}},
	"table-merge":        {{cols: 5, rows: 6, firstRow: true}},
	"table-striped":      {{cols: 3, rows: 4, firstRow: true}},
	"table-slots":        {{cols: 3, rows: 3, firstRow: true, placeholder: "body"}},
	"table-in-container": {{cols: 2, rows: 3, firstRow: true}, {cols: 3, rows: 3, firstRow: true}},
}

// tableFixtureGrid는 표 하나의 기대 격자다.
type tableFixtureGrid struct {
	cols int
	rows int
	// firstRow는 `a:tblPr@firstRow` 힌트가 켜져야 하는지다.
	firstRow bool
	// placeholder는 이 프레임이 채우는 슬롯의 종류다 (슬롯을 쓰지 않으면 "").
	placeholder string
}

// 표는 `p:graphicFrame` + `a:tbl`로 나오고, 그 격자가 기대한 크기이며, 표 스타일을 가리키지
// 않는다 (줄무늬 힌트도 켜지 않는다 — 줄무늬는 값 층이 셀 채움으로 구워 놓는다).
func TestTableFixtureTablesAreGraphicFrames(t *testing.T) {
	seen := map[string]bool{}
	for _, slide := range tableFixtureSlides(t) {
		want, ok := tableFixtureTables[slide.id]
		if !ok {
			t.Errorf("%s(%s)가 표에 없다 — 슬라이드가 늘면 기대 격자를 함께 적는다", slide.part, slide.id)
			continue
		}
		seen[slide.id] = true
		if len(slide.frames) != len(want) {
			t.Errorf("%s(%s): 표 프레임 %d개, want %d개", slide.part, slide.id, len(slide.frames), len(want))
			continue
		}
		for i, frame := range slide.frames {
			tbl, spec := frame.Tbl, want[i]
			if got := len(tbl.Cols); got != spec.cols {
				t.Errorf("%s(%s) %d번째 표: 열 %d개, want %d개", slide.part, slide.id, i, got, spec.cols)
			}
			if got := len(tbl.Rows); got != spec.rows {
				t.Errorf("%s(%s) %d번째 표: 행 %d개, want %d개", slide.part, slide.id, i, got, spec.rows)
			}
			if tbl.Pr.StyleID != noStyleNoGridID {
				t.Errorf("%s(%s) %d번째 표: tableStyleId = %q, want %q — 스타일을 끄지 않으면 뷰어 기본값이 셀 외관 위에 얹힌다",
					slide.part, slide.id, i, tbl.Pr.StyleID, noStyleNoGridID)
			}
			// 줄무늬·머리열 힌트는 **뷰어가 색을 얹는** 값이다. 우리가 이미 셀에 구워 놓은
			// 색과 겹치면 표 스타일을 바꾼 순간 줄무늬가 두 벌이 된다.
			for _, hint := range []struct{ name, value string }{
				{"bandRow", tbl.Pr.BandRow}, {"bandCol", tbl.Pr.BandCol}, {"firstCol", tbl.Pr.FirstCol},
			} {
				if hint.value != "" {
					t.Errorf("%s(%s) %d번째 표: %s=%q가 켜져 있다 — 뷰어가 셀 외관 위에 색을 얹는다",
						slide.part, slide.id, i, hint.name, hint.value)
				}
			}
			wantFirstRow := ""
			if spec.firstRow {
				wantFirstRow = "1"
			}
			if tbl.Pr.FirstRow != wantFirstRow {
				t.Errorf("%s(%s) %d번째 표: firstRow=%q, want %q", slide.part, slide.id, i, tbl.Pr.FirstRow, wantFirstRow)
			}
			// 슬롯을 채운 표는 플레이스홀더를 싣는다 — 도형·그림과 **같은 자리**의 코드다
			// (pptx.slotPlaceholder). 좌표는 여전히 잰 박스가 소유한다 (p:xfrm이 그대로 실린다).
			ph := frame.NV.NvPr.Ph
			switch {
			case spec.placeholder == "" && ph != nil:
				t.Errorf("%s(%s) %d번째 표에 p:ph(%s)가 있다 — 슬롯을 쓰지 않은 표다",
					slide.part, slide.id, i, ph.Type)
			case spec.placeholder != "" && ph == nil:
				t.Errorf("%s(%s) %d번째 표에 p:ph가 없다, want type=%q", slide.part, slide.id, i, spec.placeholder)
			case spec.placeholder != "" && ph.Type != spec.placeholder:
				t.Errorf("%s(%s) %d번째 표의 p:ph@type = %q, want %q", slide.part, slide.id, i, ph.Type, spec.placeholder)
			}
			if frame.Xfrm.Ext.Cx <= 0 || frame.Xfrm.Ext.Cy <= 0 {
				t.Errorf("%s(%s) %d번째 표의 ext가 %d×%d다 — 프레임은 잰 크기를 갖는다",
					slide.part, slide.id, i, frame.Xfrm.Ext.Cx, frame.Xfrm.Ext.Cy)
			}
		}
	}
	for id := range tableFixtureTables {
		if !seen[id] {
			t.Errorf("표의 슬라이드 %s가 산출물에 없다", id)
		}
	}
}

// `a:tbl`의 모든 행은 열 수만큼의 `a:tc`를 갖는가 — 병합에 덮인 자리마다 `hMerge`/`vMerge` 빈
// 셀이 서 있는가.
//
// 이 계약이 깨진 파일은 **열리지만 복구를 묻는다**: PowerPoint는 행마다 격자 폭을 채우기를
// 요구하고, 모자란 파일을 스스로 고치며 "이 프레젠테이션의 내용에 문제가 있습니다"를 띄운다.
// 골든 비교로는 그것을 알 수 없고(XML은 그럴듯하다) 픽셀 비교로도 알 수 없다 (LibreOffice는
// 조용히 그린다). 그래서 덮인 칸을 **세는** 자리가 필요하다.
func TestTableFixtureEveryRowFillsTheGrid(t *testing.T) {
	for _, slide := range tableFixtureSlides(t) {
		for i, frame := range slide.frames {
			tbl := frame.Tbl
			cols := len(tbl.Cols)
			// pending[c]는 c열에서 아래로 몇 행이 더 덮여 있는지다 (rowSpan이 채운다).
			pending := make([]int, cols)
			for r, row := range tbl.Rows {
				// **`a:tc` 하나가 격자 칸 하나다** — `gridSpan`이 든 셀도 자리를 하나만 차지하고,
				// 그 셀이 덮는 칸에는 `hMerge` 빈 셀이 따로 선다 (병합은 "칸을 없애는" 것이
				// 아니라 "칸을 표시하는" 것이다). 그래서 행의 셀 수가 곧 열 수여야 한다.
				if len(row.Cells) != cols {
					t.Errorf("%s(%s) %d번째 표 %d행: a:tc %d개, want %d개 (열 수) — 모자란 파일은 PowerPoint가 복구를 묻는다",
						slide.part, slide.id, i, r+1, len(row.Cells), cols)
					continue
				}
				// hMerge가 몇 칸 더 서야 하는지다 (gridSpan이 채운다).
				across := 0
				for c, cell := range row.Cells {
					where := fmt.Sprintf("%s(%s) %d번째 표 %d행 %d열", slide.part, slide.id, i, r+1, c+1)
					// 가로 병합: gridSpan=n 뒤로 정확히 n−1개의 hMerge가 따라온다.
					switch {
					case across > 0 && cell.HMerge != "1":
						t.Errorf("%s: gridSpan에 덮인 자리인데 hMerge가 없다", where)
					case across == 0 && cell.HMerge == "1":
						t.Errorf("%s: hMerge인데 왼쪽에서 뻗어 온 gridSpan이 없다", where)
					}
					if across > 0 {
						across--
					} else if cell.GridSpan > 1 {
						across = cell.GridSpan - 1
					}

					// 세로 병합: rowSpan=n이면 아래 n−1행의 같은 열에 vMerge가 선다.
					switch {
					case pending[c] > 0 && cell.VMerge != "1":
						t.Errorf("%s: rowSpan에 덮인 자리인데 vMerge가 없다", where)
					case pending[c] == 0 && cell.VMerge == "1":
						t.Errorf("%s: vMerge인데 위에서 내려온 rowSpan이 없다", where)
					}
					if pending[c] > 0 {
						pending[c]--
					} else if cell.RowSpan > 1 {
						pending[c] = cell.RowSpan - 1
					}

					// 덮인 셀은 내용을 갖지 않는다 — 병합의 글자는 주인 셀에 한 번만 있어야
					// 한다 (두 곳에 있으면 뷰어가 둘 다 그린다).
					if (cell.HMerge == "1" || cell.VMerge == "1") && len(cell.Runs) != 0 {
						t.Errorf("%s: 덮인 셀에 글자 %v가 있다", where, cell.Runs)
					}
				}
				if across != 0 {
					t.Errorf("%s(%s) %d번째 표 %d행: gridSpan이 행 끝을 %d칸 넘긴다",
						slide.part, slide.id, i, r+1, across)
				}
			}
			for c, left := range pending {
				if left != 0 {
					t.Errorf("%s(%s) %d번째 표 %d열: rowSpan이 표 끝을 %d행 넘긴다", slide.part, slide.id, i, c+1, left)
				}
			}
		}
	}
}

// 셀 외관은 **명시**인가 — 네 변(`lnL`·`lnR`·`lnT`·`lnB`)과 세로 앵커가 모든 셀에 있는가.
//
// 표 스타일을 끈 대가가 여기다: 뷰어가 얹을 값이 없으므로 **우리가 내지 않은 것은 없는 것**이다.
// 테두리를 지운 셀(`border="none"`)도 선언을 빼지 않고 네 변을 `noFill`로 낸다 — 선언이 없는
// 변과 없다고 선언한 변은 뷰어가 표 스타일을 바꿀 때 갈라진다.
//
// 덮인 셀(`hMerge`/`vMerge`)은 예외다: 그 자리의 외관은 주인 셀이 소유한다 (빈 `a:tcPr`).
func TestTableFixtureCellsCarryExplicitAppearance(t *testing.T) {
	cells, noFills := 0, 0
	for _, slide := range tableFixtureSlides(t) {
		for i, frame := range slide.frames {
			for r, row := range frame.Tbl.Rows {
				for c, cell := range row.Cells {
					where := fmt.Sprintf("%s(%s) %d번째 표 %d행 %d열", slide.part, slide.id, i, r+1, c+1)
					if cell.HMerge == "1" || cell.VMerge == "1" {
						if cell.Pr.Anchor != "" || cell.Pr.LnL != nil {
							t.Errorf("%s: 덮인 셀에 외관이 실렸다 — 그 자리의 외관은 주인 셀이 소유한다", where)
						}
						continue
					}
					cells++
					if cell.Pr.Anchor == "" {
						t.Errorf("%s: anchor가 없다 — 세로 앵커는 뷰어 기본값에 맡기지 않는다", where)
					}
					for _, side := range []struct {
						name string
						ln   *tblLn
					}{{"lnL", cell.Pr.LnL}, {"lnR", cell.Pr.LnR}, {"lnT", cell.Pr.LnT}, {"lnB", cell.Pr.LnB}} {
						if side.ln == nil {
							t.Errorf("%s: %s가 없다 — 네 변을 모두 명시해야 표 스타일이 끼어들지 않는다", where, side.name)
							continue
						}
						if side.ln.NoFill == nil && side.ln.SolidFill == nil {
							t.Errorf("%s: %s에 채움도 noFill도 없다", where, side.name)
						}
						if side.ln.NoFill != nil {
							noFills++
						}
					}
				}
			}
		}
	}
	if cells == 0 {
		t.Fatal("셀을 하나도 보지 않았다 — 산출물에 표가 없다")
	}
	// `border="none"` 셀이 실제로 `noFill` 네 변을 냈는가 (003_striped의 보류 셀 하나) —
	// 없으면 위 순회가 "테두리가 있는 셀만" 보고 통과한 것이다.
	if noFills != 4 {
		t.Errorf("noFill 변 %d개, want 4개 — 003_striped의 border=\"none\" 셀 하나가 네 변을 지운다", noFills)
	}
}

// 격자 수치는 **잰 값**인가 — 프레임 좌표가 측정 rect의 EMU이고, 열 너비 합·행 높이 합이
// 프레임 크기와 **정확히** 같고, 각 행 높이가 잰 `<tr>` 높이와 같은가.
//
// 합이 정확히 같아야 하는 이유는 타일링이다 (render/pptx.edgeLengths): 칸마다 반올림하면 합이
// 프레임을 삐져나오고, 그 모양은 마지막 열이 잘린 표로 나타난다. 개별 행·열은 1px 허용 오차를
// 둔다 — 경계를 환산해 차분하므로 칸 하나는 반올림 반 EMU를 떠안고, 첫 행은 테두리 합침의
// 반 픽셀까지 흡수한다 (tableRowHeights).
//
// 브라우저를 띄우는 검사다 (지오메트리 패스).
func TestTableFixtureGridComesFromMeasuredBoxes(t *testing.T) {
	RequireChromium(t)
	fixture := fixtureByName(t, tableFixtureName)
	result := fixture.Build(t)
	if result.Geometry == nil {
		t.Fatal("빌드에 지오메트리가 없다")
	}
	data, err := os.ReadFile(fixture.Output(t, result, ".pptx"))
	if err != nil {
		t.Fatal(err)
	}
	parts := PPTXParts(t, data)
	frames := make([][]tblFrame, len(result.Deck.Slides))
	for i := range result.Deck.Slides {
		part := fmt.Sprintf("ppt/slides/slide%d.xml", i+1)
		frames[i] = decodeFrames(t, part, parts[part])
	}

	refs := geom.Refs(result.Deck)
	if len(refs) != len(result.Geometry.Boxes) {
		t.Fatalf("요소 %d개, 측정 박스 %d개", len(refs), len(result.Geometry.Boxes))
	}
	// 박스를 식별자로 찾는 색인이다 — 행·셀은 부모 식별자로 묶는다 (문서 순서의 자식).
	tables := 0
	at := make([]int, len(result.Deck.Slides)) // 슬라이드별로 본 표 수
	for i, ref := range refs {
		if ref.El != soml.TableElement {
			continue
		}
		box := result.Geometry.Boxes[i]
		list := frames[box.Slide]
		n := at[box.Slide]
		at[box.Slide]++
		if n >= len(list) {
			t.Fatalf("슬라이드 %d의 표 %s에 대응하는 프레임이 없다 (프레임 %d개)", box.Slide+1, box.ID, len(list))
		}
		frame := list[n]
		tables++

		x, y := pptx.Offset(box.Rect)
		cx, cy := pptx.Extent(box.Rect)
		if frame.Xfrm.Off.X != x || frame.Xfrm.Off.Y != y || frame.Xfrm.Ext.Cx != cx || frame.Xfrm.Ext.Cy != cy {
			t.Errorf("%s의 p:xfrm = (%d,%d %d×%d), want (%d,%d %d×%d) — 잰 rect %s의 EMU다",
				box.ID, frame.Xfrm.Off.X, frame.Xfrm.Off.Y, frame.Xfrm.Ext.Cx, frame.Xfrm.Ext.Cy, x, y, cx, cy, box.Rect)
		}

		var colSum int64
		for _, col := range frame.Tbl.Cols {
			colSum += col.W
		}
		if colSum != frame.Xfrm.Ext.Cx {
			t.Errorf("%s의 열 너비 합 %d EMU, want %d EMU (프레임 폭) — 합이 다르면 마지막 열이 프레임을 삐져나온다",
				box.ID, colSum, frame.Xfrm.Ext.Cx)
		}
		var rowSum int64
		for _, row := range frame.Tbl.Rows {
			rowSum += row.H
		}
		if rowSum != frame.Xfrm.Ext.Cy {
			t.Errorf("%s의 행 높이 합 %d EMU, want %d EMU (프레임 높이)", box.ID, rowSum, frame.Xfrm.Ext.Cy)
		}

		// 행 높이는 잰 `<tr>` 박스에서 온다 — 순서까지 같아야 한다 (행이 뒤바뀌면 글자는
		// 제자리인데 높이만 서로 옮겨 앉는다).
		rows := childBoxes(refs, result.Geometry.Boxes, box.ID, soml.TableRowElement)
		if len(rows) != len(frame.Tbl.Rows) {
			t.Errorf("%s: 잰 행 %d개, a:tr %d개", box.ID, len(rows), len(frame.Tbl.Rows))
			continue
		}
		for r, rowBox := range rows {
			want := pptx.EMU(rowBox.Rect.H)
			if got := frame.Tbl.Rows[r].H; !withinEMU(got, want, tableGridToleranceEMU) {
				t.Errorf("%s %d행: a:tr@h = %d EMU, want %d EMU (잰 높이 %.1fpx)",
					box.ID, r+1, got, want, rowBox.Rect.H)
			}
		}
	}
	if tables != 6 {
		t.Errorf("표 %d개를 보았다, want 6개 (001~005의 표 합)", tables)
	}
}

// tableGridToleranceEMU는 행·열 하나의 허용 오차다 (1px).
//
// 1px인 이유: 경계를 EMU로 환산한 뒤 차분하므로 칸 하나는 반올림 반 EMU를 떠안고, 첫 행은
// 테두리 합침(border-collapse)의 반 픽셀까지 흡수한다 (render/pptx.tableRowHeights). 그보다 큰
// 차이는 반올림이 아니라 **다른 박스를 읽은 것**이다.
const tableGridToleranceEMU = int64(9525)

// withinEMU는 두 EMU 값이 허용 오차 안인지다.
func withinEMU(got, want, tol int64) bool { return math.Abs(float64(got-want)) <= float64(tol) }

// childBoxes는 부모 식별자 아래의 특정 종류 자식 박스를 문서 순서로 모은다.
func childBoxes(refs []geom.Ref, boxes []geom.Box, parent, el string) []geom.Box {
	var found []geom.Box
	for i, ref := range refs {
		if ref.Parent == parent && ref.El == el {
			found = append(found, boxes[i])
		}
	}
	return found
}

// tableFixtureSlide는 슬라이드 파트 하나에서 읽은 표들이다.
type tableFixtureSlide struct {
	part   string
	id     string
	frames []tblFrame
}

// tableFixtureSlides는 09_table을 한 번 빌드해 슬라이드를 순서대로 읽는다 — 세 검사가 같은
// 산출물을 보므로 빌드를 나눠 쓴다 (파트는 메모리에 남는다).
//
// 슬라이드를 **id로** 기억하는 이유는 다른 픽스처 검사와 같다: 파트 번호로 표를 짜면 슬라이드를
// 하나 끼워 넣는 순간 표가 엉뚱한 슬라이드를 가리킨다.
func tableFixtureSlides(t *testing.T) []tableFixtureSlide {
	t.Helper()
	if tableSlidesCache == nil {
		fixture := fixtureByName(t, tableFixtureName)
		result := fixture.Build(t)
		data, err := os.ReadFile(fixture.Output(t, result, ".pptx"))
		if err != nil {
			t.Fatal(err)
		}
		parts := PPTXParts(t, data)
		for i, slide := range result.Deck.Slides {
			part := fmt.Sprintf("ppt/slides/slide%d.xml", i+1)
			tableSlidesCache = append(tableSlidesCache, tableFixtureSlide{
				part: path.Base(part), id: slide.ID, frames: decodeFrames(t, part, parts[part]),
			})
		}
	}
	if len(tableSlidesCache) != len(tableFixtureTables) {
		t.Fatalf("슬라이드 %d장, want %d장", len(tableSlidesCache), len(tableFixtureTables))
	}
	return tableSlidesCache
}

// tableSlidesCache는 tableFixtureSlides가 한 번 읽은 결과다.
var tableSlidesCache []tableFixtureSlide

// decodeFrames는 슬라이드 파트에서 `p:graphicFrame`을 문서 순서로 읽는다.
//
// 정규식이 아니라 XML 디코더를 쓰는 이유: 이 검사들이 보는 것은 **중첩된 구조**다 (행 안의
// 셀, 셀 안의 네 변). 정규식으로는 "이 행의 셀"과 "다음 행의 셀"을 가를 수 없고, 가르지 못하면
// 병합 격자를 셀 수 없다.
func decodeFrames(t *testing.T, part string, data []byte) []tblFrame {
	t.Helper()
	if len(data) == 0 {
		t.Fatalf("파트 %s가 없다", part)
	}
	var tree struct {
		Frames []tblFrame `xml:"cSld>spTree>graphicFrame"`
	}
	if err := xml.Unmarshal(data, &tree); err != nil {
		t.Fatalf("%s를 읽을 수 없다: %v", part, err)
	}
	return tree.Frames
}

// tblFrame은 `p:graphicFrame` 하나다 (표만 담는다 — 이번 어휘에서 프레임을 내는 종류는
// `<table>`뿐이다).
type tblFrame struct {
	NV struct {
		CNvPr struct {
			ID   int    `xml:"id,attr"`
			Name string `xml:"name,attr"`
		} `xml:"cNvPr"`
		NvPr struct {
			Ph *struct {
				Type string `xml:"type,attr"`
				Idx  string `xml:"idx,attr"`
			} `xml:"ph"`
		} `xml:"nvPr"`
	} `xml:"nvGraphicFramePr"`
	Xfrm struct {
		Off struct {
			X int64 `xml:"x,attr"`
			Y int64 `xml:"y,attr"`
		} `xml:"off"`
		Ext struct {
			Cx int64 `xml:"cx,attr"`
			Cy int64 `xml:"cy,attr"`
		} `xml:"ext"`
	} `xml:"xfrm"`
	Tbl tblTable `xml:"graphic>graphicData>tbl"`
}

// tblTable은 `a:tbl` 하나다.
type tblTable struct {
	Pr struct {
		FirstRow string `xml:"firstRow,attr"`
		FirstCol string `xml:"firstCol,attr"`
		BandRow  string `xml:"bandRow,attr"`
		BandCol  string `xml:"bandCol,attr"`
		StyleID  string `xml:"tableStyleId"`
	} `xml:"tblPr"`
	Cols []struct {
		W int64 `xml:"w,attr"`
	} `xml:"tblGrid>gridCol"`
	Rows []tblRow `xml:"tr"`
}

// tblRow은 `a:tr` 하나다.
type tblRow struct {
	H     int64     `xml:"h,attr"`
	Cells []tblCell `xml:"tc"`
}

// tblCell는 `a:tc` 하나다.
type tblCell struct {
	GridSpan int    `xml:"gridSpan,attr"`
	RowSpan  int    `xml:"rowSpan,attr"`
	HMerge   string `xml:"hMerge,attr"`
	VMerge   string `xml:"vMerge,attr"`
	Pr       struct {
		Anchor string `xml:"anchor,attr"`
		LnL    *tblLn `xml:"lnL"`
		LnR    *tblLn `xml:"lnR"`
		LnT    *tblLn `xml:"lnT"`
		LnB    *tblLn `xml:"lnB"`
	} `xml:"tcPr"`
	// Runs는 셀 글자다 — 덮인 셀이 비어 있는지 보는 데 쓴다.
	Runs []string `xml:"txBody>p>r>t"`
}

// tblLn은 셀 테두리 한 변이다 (`a:lnL` 등).
type tblLn struct {
	NoFill    *struct{} `xml:"noFill"`
	SolidFill *struct{} `xml:"solidFill"`
}
