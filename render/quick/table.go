package quick

import (
	"math"
	"strconv"
	"strings"
)

// 표는 웹 렌더러가 내는 꼴만 다룬다: table > (tbody) > tr > th/td, colspan 없음,
// border-collapse, table-layout: auto.

func (e *engine) buildTable(b *box) {
	var rows func(n *node)
	rows = func(n *node) {
		for _, c := range n.children {
			if c.isText() {
				continue
			}
			switch displayOf(c) {
			case "table-row":
				if r := e.build(c, "table"); r != nil {
					b.kids = append(b.kids, r)
				}
			case "none":
			default:
				if c.tag == "tbody" || c.tag == "thead" || c.tag == "tfoot" {
					rows(c)
				}
			}
		}
	}
	rows(b.n)
}

func (e *engine) buildRow(b *box) {
	for _, c := range b.n.children {
		if c.isText() || displayOf(c) == "none" {
			continue
		}
		cell := e.newBox(c, c.style, "table-cell")
		e.buildFlow(cell)
		b.kids = append(b.kids, cell)
	}
}

// tableCell은 격자에 놓인 칸 하나다.
type tableCell struct {
	b              *box
	row            *box
	r0, r1, c0, c1 int
}

func spanAttr(n *node, name string) int {
	v, err := strconv.Atoi(strings.TrimSpace(n.attrs[name]))
	if err != nil || v < 1 {
		return 1
	}
	return v
}

// tableGrid는 rowspan · colspan을 펼쳐 칸을 격자에 놓는다 (HTML 표 모델).
func tableGrid(b *box) ([]tableCell, int) {
	occupied := map[[2]int]bool{}
	var cells []tableCell
	cols := 0
	for r, row := range b.kids {
		c := 0
		for _, cell := range row.kids {
			for occupied[[2]int{r, c}] {
				c++
			}
			rs, cs := spanAttr(cell.n, "rowspan"), spanAttr(cell.n, "colspan")
			rs = min(rs, len(b.kids)-r)
			for i := r; i < r+rs; i++ {
				for j := c; j < c+cs; j++ {
					occupied[[2]int{i, j}] = true
				}
			}
			cells = append(cells, tableCell{b: cell, row: row, r0: r, r1: r + rs, c0: c, c1: c + cs})
			c += cs
			cols = max(cols, c)
		}
	}
	return cells, cols
}

// tableColumns는 열마다 min/max-content 폭이다 (칸의 padding · border 포함). 여러 열에 걸친 칸은
// 모자라는 만큼을 걸친 열의 max 비율로 나눠 더한다.
func (e *engine) tableColumns(b *box) ([]float64, []float64) {
	cells, cols := tableGrid(b)
	mins, maxs := make([]float64, cols), make([]float64, cols)
	for _, c := range cells {
		if c.c1-c.c0 != 1 {
			continue
		}
		mins[c.c0] = math.Max(mins[c.c0], e.contribution(c.b, true))
		maxs[c.c0] = math.Max(maxs[c.c0], e.contribution(c.b, false))
	}
	for _, c := range cells {
		if c.c1-c.c0 == 1 {
			continue
		}
		spread := func(vals []float64, need float64) {
			have, weight := 0.0, 0.0
			for i := c.c0; i < c.c1; i++ {
				have += vals[i]
				weight += maxs[i]
			}
			if need <= have {
				return
			}
			for i := c.c0; i < c.c1; i++ {
				share := 1 / float64(c.c1-c.c0)
				if weight > 0 {
					share = maxs[i] / weight
				}
				vals[i] += (need - have) * share
			}
		}
		spread(mins, e.contribution(c.b, true))
		spread(maxs, e.contribution(c.b, false))
	}
	for i := range maxs {
		maxs[i] = math.Max(maxs[i], mins[i])
	}
	return mins, maxs
}

// collapsedFrame은 border-collapse 표의 바깥 테두리다: 바깥 칸 테두리의 반이 표 쪽에 놓인다.
func (e *engine) collapsedFrame(b *box) edges {
	if b.st.get("border-collapse") != "collapse" {
		return edges{}
	}
	cells, cols := tableGrid(b)
	var out edges
	full := func(c *box, side string) float64 {
		st := c.st.get("border-" + side + "-style")
		if st == "" || st == "none" || st == "hidden" {
			return 0
		}
		return lengthOr(c.st.get("border-"+side+"-width"), c.fs(), length{}).px
	}
	for _, c := range cells {
		if c.r0 == 0 {
			out.t = math.Max(out.t, full(c.b, "top")/2)
		}
		if c.r1 == len(b.kids) {
			out.b = math.Max(out.b, full(c.b, "bottom")/2)
		}
		if c.c0 == 0 {
			out.l = math.Max(out.l, full(c.b, "left")/2)
		}
		if c.c1 == cols {
			out.r = math.Max(out.r, full(c.b, "right")/2)
		}
	}
	return out
}

func (e *engine) layoutTable(b *box, innerW float64, innerH *float64, fr edges) float64 {
	outer := e.collapsedFrame(b)
	innerW -= outer.h()
	cells, cols := tableGrid(b)
	mins, maxs := e.tableColumns(b)
	widths := make([]float64, cols)
	sumMin, sumMax := 0.0, 0.0
	for i := range mins {
		sumMin += mins[i]
		sumMax += maxs[i]
	}
	switch {
	case sumMax <= innerW && sumMax > 0:
		for i := range widths {
			widths[i] = maxs[i] + (innerW-sumMax)*maxs[i]/sumMax
		}
	case sumMin <= innerW && sumMax > sumMin:
		t := (innerW - sumMin) / (sumMax - sumMin)
		for i := range widths {
			widths[i] = mins[i] + (maxs[i]-mins[i])*t
		}
	default:
		copy(widths, mins)
	}
	colX := make([]float64, cols+1)
	for i, w := range widths {
		colX[i+1] = colX[i] + w
	}
	// 행 높이: 한 행 칸이 먼저, 여러 행에 걸친 칸은 모자라는 만큼 마지막 행에 더한다.
	rows := len(b.kids)
	rowH := make([]float64, rows)
	contentH := map[*box]float64{}
	for _, c := range cells {
		e.layout(c.b, colX[c.c1]-colX[c.c0], nil, innerW, 0, false)
		contentH[c.b] = c.b.h
		if c.r1-c.r0 == 1 {
			rowH[c.r0] = math.Max(rowH[c.r0], c.b.h)
		}
	}
	for _, c := range cells {
		if c.r1-c.r0 == 1 {
			continue
		}
		have := 0.0
		for i := c.r0; i < c.r1; i++ {
			have += rowH[i]
		}
		if need := contentH[c.b]; need > have {
			rowH[c.r1-1] += need - have
		}
	}
	// 표 높이가 정해졌고(stretch · 지정 높이) 행 합보다 크면 남는 높이를 행에 고르게 나눈다.
	if innerH != nil && rows > 0 {
		sum := outer.v()
		for _, h := range rowH {
			sum += h
		}
		if extra := *innerH - sum; extra > 0 {
			for i := range rowH {
				rowH[i] += extra / float64(rows)
			}
		}
	}
	rowY := make([]float64, rows+1)
	for i, h := range rowH {
		rowY[i+1] = rowY[i] + h
	}
	for i, row := range b.kids {
		row.x, row.y, row.w, row.h = fr.l+outer.l, fr.t+outer.t+rowY[i], innerW, rowH[i]
	}
	for _, c := range cells {
		h := rowY[c.r1] - rowY[c.r0]
		w := colX[c.c1] - colX[c.c0]
		e.layout(c.b, w, &h, innerW, 0, false)
		dy := 0.0
		switch c.b.st.get("vertical-align") {
		case "middle", "":
			dy = (h - contentH[c.b]) / 2
		case "bottom":
			dy = h - contentH[c.b]
		}
		shiftContent(c.b, dy)
		c.b.x, c.b.y = colX[c.c0], rowY[c.r0]-rowY[c.r0]
	}
	return rowY[rows] + outer.v()
}

// shiftContent는 box의 흐름 내용(줄 · 자식)을 아래로 dy만큼 옮긴다.
func shiftContent(b *box, dy float64) {
	if dy == 0 {
		return
	}
	for i := range b.lines {
		l := &b.lines[i]
		l.y += dy
		l.baseline += dy
		l.textRect.y += dy
		for j := range l.frags {
			l.frags[j].baseline += dy
			if a := l.frags[j].atomic; a != nil {
				a.y += dy
			}
		}
	}
	for _, k := range b.kids {
		k.y += dy
	}
}
