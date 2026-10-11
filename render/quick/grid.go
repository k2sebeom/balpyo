package quick

import (
	"math"
	"sort"
	"strconv"
	"strings"
)

// track은 트랙 정의 하나다: 최소 · 최대 크기 함수.
type track struct {
	minKind, maxKind string // "px" · "pct" · "auto" · "fr" · "min" · "max"
	minVal, maxVal   float64
}

func parseTrack(v string, fontSize float64) track {
	v = strings.TrimSpace(v)
	if strings.HasPrefix(v, "minmax(") {
		args := splitTop(v[len("minmax("):len(v)-1], ',')
		a := parseTrack(args[0], fontSize)
		b := a
		if len(args) > 1 {
			b = parseTrack(args[1], fontSize)
		}
		return track{minKind: a.minKind, minVal: a.minVal, maxKind: b.maxKind, maxVal: b.maxVal}
	}
	switch v {
	case "auto":
		return track{minKind: "auto", maxKind: "auto"}
	case "min-content":
		return track{minKind: "min", maxKind: "min"}
	case "max-content":
		return track{minKind: "max", maxKind: "max"}
	}
	if strings.HasSuffix(v, "fr") {
		f, _ := strconv.ParseFloat(strings.TrimSuffix(v, "fr"), 64)
		return track{minKind: "auto", maxKind: "fr", maxVal: f}
	}
	l := lengthOr(v, fontSize, autoLength)
	if l.auto {
		return track{minKind: "auto", maxKind: "auto"}
	}
	if l.pct != 0 {
		return track{minKind: "pct", minVal: l.pct, maxKind: "pct", maxVal: l.pct}
	}
	return track{minKind: "px", minVal: l.px, maxKind: "px", maxVal: l.px}
}

func parseTrackList(v string, fontSize float64) []track {
	v = strings.TrimSpace(v)
	if v == "" || v == "none" {
		return nil
	}
	var out []track
	for _, part := range splitValues(v) {
		if strings.HasPrefix(part, "repeat(") {
			args := splitTop(part[len("repeat("):len(part)-1], ',')
			n, _ := strconv.Atoi(strings.TrimSpace(args[0]))
			inner := parseTrackList(strings.Join(args[1:], ","), fontSize)
			for i := 0; i < n; i++ {
				out = append(out, inner...)
			}
			continue
		}
		if strings.HasPrefix(part, "[") {
			continue // 줄 이름
		}
		out = append(out, parseTrack(part, fontSize))
	}
	return out
}

type area struct{ r0, r1, c0, c1 int } // 0부터, 끝은 미포함

func parseAreas(v string) (map[string]area, int, int) {
	out := map[string]area{}
	var rows [][]string
	for _, part := range splitValues(v) {
		if strings.HasPrefix(part, `"`) || strings.HasPrefix(part, `'`) {
			rows = append(rows, strings.Fields(unquoteCSS(part)))
		}
	}
	cols := 0
	for r, cells := range rows {
		cols = max(cols, len(cells))
		for c, name := range cells {
			if strings.Trim(name, ".") == "" {
				continue
			}
			a, ok := out[name]
			if !ok {
				out[name] = area{r, r + 1, c, c + 1}
				continue
			}
			a.r0, a.r1 = min(a.r0, r), max(a.r1, r+1)
			a.c0, a.c1 = min(a.c0, c), max(a.c1, c+1)
			out[name] = a
		}
	}
	return out, len(rows), cols
}

type gridItem struct {
	b              *box
	r0, r1, c0, c1 int
	m              edges
}

// axisPlace는 한 축의 위치다: start가 -1이면 자동 배치다.
type axisPlace struct{ start, span int }

// resolveAxis는 grid-*-start/end 두 값을 읽는다: 줄 번호(1부터), span N, 영역 이름, auto.
func resolveAxis(startV, endV string, areas map[string]area, rows bool) axisPlace {
	startV, endV = strings.TrimSpace(startV), strings.TrimSpace(endV)
	spanOf := func(v string) int {
		if strings.HasPrefix(v, "span") {
			n, _ := strconv.Atoi(strings.TrimSpace(strings.TrimPrefix(v, "span")))
			return max(n, 1)
		}
		return 0
	}
	if a, ok := areas[startV]; ok {
		if rows {
			return axisPlace{a.r0, a.r1 - a.r0}
		}
		return axisPlace{a.c0, a.c1 - a.c0}
	}
	if n, err := strconv.Atoi(startV); err == nil && n > 0 {
		p := axisPlace{n - 1, 1}
		if m, err := strconv.Atoi(endV); err == nil && m > n {
			p.span = m - n
		} else if s := spanOf(endV); s > 0 {
			p.span = s
		}
		return p
	}
	if s := spanOf(startV); s > 0 {
		return axisPlace{-1, s}
	}
	if s := spanOf(endV); s > 0 {
		return axisPlace{-1, s}
	}
	return axisPlace{-1, 1}
}

// placeItems는 항목을 격자 칸에 놓는다 — CSS 격자 자동 배치 알고리즘이다. cols · rows는 명시 격자의
// 열 · 행 수, flow는 grid-auto-flow다 (row | column, dense).
//
// 알고리즘은 흐름 방향을 기준으로 적는다: 흐름이 row면 "줄"은 행이고 커서는 줄 안에서 열을 따라
// 걷는다. column이면 둘이 바뀐다. 순서는 1) 두 축이 다 정해진 항목 2) 줄만 정해진 항목 3) 줄 안의
// 트랙 수 확정 4) 나머지를 커서로 놓기다. sparse는 커서가 뒤로 가지 않고, dense는 항목마다 맨 앞에서
// 다시 찾아 앞에 남은 빈칸을 채운다.
func placeItems(kids []*box, areas map[string]area, cols, rows int, flow string) ([]*gridItem, int, int) {
	column, dense := false, false
	for _, f := range strings.Fields(flow) {
		switch f {
		case "column":
			column = true
		case "dense":
			dense = true
		}
	}
	// (줄, 칸) 좌표: 줄은 흐름이 넘어가는 축, 칸은 커서가 줄 안에서 걷는 축이다.
	type placed struct{ line, cell axisPlace }
	occupied := map[[2]int]bool{}
	fits := func(l0, c0, ls, cs int) bool {
		for l := l0; l < l0+ls; l++ {
			for c := c0; c < c0+cs; c++ {
				if occupied[[2]int{l, c}] {
					return false
				}
			}
		}
		return true
	}
	mark := func(p placed) {
		for l := p.line.start; l < p.line.start+p.line.span; l++ {
			for c := p.cell.start; c < p.cell.start+p.cell.span; c++ {
				occupied[[2]int{l, c}] = true
			}
		}
	}
	nCells := cols
	if column {
		nCells = rows
	}
	items := make([]*gridItem, len(kids))
	pos := make([]placed, len(kids))
	for i, k := range kids {
		items[i] = &gridItem{b: k}
		row := resolveAxis(k.st.get("grid-row-start"), k.st.get("grid-row-end"), areas, true)
		col := resolveAxis(k.st.get("grid-column-start"), k.st.get("grid-column-end"), areas, false)
		if column {
			pos[i] = placed{col, row}
		} else {
			pos[i] = placed{row, col}
		}
	}
	// 1. 두 축이 다 정해진 항목
	for i := range pos {
		if p := pos[i]; p.line.start >= 0 && p.cell.start >= 0 {
			mark(p)
		}
	}
	// 2. 줄만 정해진 항목: 그 줄에서 겹치지 않는 첫 칸. sparse면 이 단계가 그 줄에 앞서 놓은 항목 뒤에서
	// 찾는다.
	after := map[int]int{}
	for i := range pos {
		p := &pos[i]
		if p.line.start < 0 || p.cell.start >= 0 {
			continue
		}
		c := 0
		if !dense {
			c = after[p.line.start]
		}
		for !fits(p.line.start, c, p.line.span, p.cell.span) {
			c++
		}
		p.cell.start = c
		after[p.line.start] = c + p.cell.span
		mark(*p)
	}
	// 3. 줄 안의 트랙 수: 명시 격자, 칸이 정해진 항목의 끝, 자동 항목의 가장 긴 span 가운데 큰 것.
	for _, p := range pos {
		if p.cell.start >= 0 {
			nCells = max(nCells, p.cell.start+p.cell.span)
		} else {
			nCells = max(nCells, p.cell.span)
		}
	}
	nCells = max(nCells, 1)
	// 4. 나머지 — 줄이 auto인 항목. 커서는 마지막에 놓은 항목의 시작 자리다.
	curL, curC := 0, 0
	for i := range pos {
		p := &pos[i]
		if p.line.start >= 0 {
			continue
		}
		if dense {
			curL, curC = 0, 0
		}
		if p.cell.start >= 0 {
			// 칸이 정해졌다: 커서가 그 칸보다 뒤에 있으면 다음 줄로 넘어가 겹치지 않는 줄을 찾는다.
			if !dense && p.cell.start < curC {
				curL++
			}
			curC = p.cell.start
			for !fits(curL, curC, p.line.span, p.cell.span) {
				curL++
			}
		} else {
			for {
				if curC+p.cell.span > nCells {
					curL++
					curC = 0
					continue
				}
				if fits(curL, curC, p.line.span, p.cell.span) {
					break
				}
				curC++
			}
			p.cell.start = curC
		}
		p.line.start = curL
		mark(*p)
	}
	nRows, nCols := 0, 0
	for i, it := range items {
		row, col := pos[i].line, pos[i].cell
		if column {
			row, col = col, row
		}
		it.r0, it.r1, it.c0, it.c1 = row.start, row.start+row.span, col.start, col.start+col.span
		nRows, nCols = max(nRows, it.r1), max(nCols, it.c1)
	}
	if column {
		nRows = max(nRows, nCells)
	} else {
		nCols = max(nCols, nCells)
	}
	return items, nRows, nCols
}

func (e *engine) gridSetup(b *box) ([]*gridItem, []track, []track) {
	fs := b.fs()
	cols := parseTrackList(b.st.get("grid-template-columns"), fs)
	rows := parseTrackList(b.st.get("grid-template-rows"), fs)
	areas, areaRows, areaCols := parseAreas(b.st.get("grid-template-areas"))
	nCols := max(len(cols), areaCols)
	items, nr, nc := placeItems(ordered(b.kids), areas, nCols, max(len(rows), areaRows), b.st.get("grid-auto-flow"))
	// 항목이 모두 절대 배치여도 격자는 한 칸을 가진다 — 절대 배치 자식의 `grid-area: 1 / 1`이 그 칸을
	// 포함 블록으로 쓴다 (브라우저가 그렇게 잰다).
	nr = max(nr, max(len(rows), areaRows), 1)
	nc = max(nc, 1)
	autoRow := parseTrackList(b.st.get("grid-auto-rows"), fs)
	autoCol := parseTrackList(b.st.get("grid-auto-columns"), fs)
	for len(cols) < nc {
		if len(autoCol) > 0 {
			cols = append(cols, autoCol[0])
		} else {
			cols = append(cols, track{minKind: "auto", maxKind: "auto"})
		}
	}
	for len(rows) < nr {
		if len(autoRow) > 0 {
			rows = append(rows, autoRow[0])
		} else {
			rows = append(rows, track{minKind: "auto", maxKind: "auto"})
		}
	}
	return items, cols, rows
}

func (e *engine) gridIntrinsic(b *box, min bool) float64 {
	items, cols, _ := e.gridSetup(b)
	_, colGap := b.gaps(0, nil)
	sizes := e.sizeTracks(cols, items, true, 0, false, colGap, func(it *gridItem, minC bool) float64 {
		return e.gridContribution(it.b, minC || min)
	}, "start")
	total := 0.0
	for i, s := range sizes {
		if i > 0 {
			total += colGap
		}
		total += s
	}
	return total
}

// sizeTracks는 트랙 크기 알고리즘의 간이판이다: 고정 → 내용(auto · min/max-content) → fr → 남는
// 공간을 auto 트랙에 늘리기(stretch). contrib는 항목의 min/max 기여(바깥 크기)다.
func (e *engine) sizeTracks(tracks []track, items []*gridItem, columns bool, avail float64, availKnown bool, gap float64,
	contrib func(it *gridItem, min bool) float64, content string) []float64 {
	n := len(tracks)
	base := make([]float64, n)
	limit := make([]float64, n)
	for i, t := range tracks {
		switch t.minKind {
		case "px":
			base[i] = t.minVal
		case "pct":
			if availKnown {
				base[i] = t.minVal * avail / 100
			}
		}
		switch t.maxKind {
		case "px":
			limit[i] = t.maxVal
		case "pct":
			if availKnown {
				limit[i] = t.maxVal * avail / 100
			} else {
				limit[i] = math.Inf(1)
			}
		case "fr":
			limit[i] = math.Inf(1)
		default:
			// 내재 최대(auto · min/max-content)는 내용이 정한다. 항목이 하나도 없는 트랙은 여러 트랙에
			// 걸친 항목을 나누는 동안 한계가 없는 채로 남는다.
			limit[i] = math.Inf(1)
		}
	}
	span := func(it *gridItem) (int, int) {
		if columns {
			return it.c0, it.c1
		}
		return it.r0, it.r1
	}
	hasFr := func(a, b int) bool {
		for i := a; i < b; i++ {
			if tracks[i].maxKind == "fr" {
				return true
			}
		}
		return false
	}
	intrinsicMin := func(t track) bool { return t.minKind == "auto" || t.minKind == "min" || t.minKind == "max" }
	intrinsicMax := func(t track) bool { return t.maxKind == "auto" || t.maxKind == "min" || t.maxKind == "max" }
	// 1. 한 트랙 항목의 기여
	for _, it := range items {
		a, b := span(it)
		if b-a != 1 {
			continue
		}
		t := tracks[a]
		if t.maxKind == "fr" {
			if t.minKind == "auto" {
				base[a] = math.Max(base[a], contrib(it, true))
			}
			continue
		}
		if t.minKind == "auto" || t.minKind == "min" {
			base[a] = math.Max(base[a], contrib(it, true))
		} else if t.minKind == "max" {
			base[a] = math.Max(base[a], contrib(it, false))
		}
		if intrinsicMax(t) {
			c := contrib(it, t.maxKind == "min")
			if math.IsInf(limit[a], 1) {
				limit[a] = c
			} else {
				limit[a] = math.Max(limit[a], c)
			}
		}
	}
	for i := range limit {
		if limit[i] < base[i] {
			limit[i] = base[i]
		}
	}
	// 2. 여러 트랙에 걸친 항목 (fr 트랙을 지나지 않는 것) — 걸친 트랙 수가 작은 무리부터 차례로.
	// 모자라는 공간은 고르게 나누되 트랙마다 한계에서 멈춘다: base는 성장 한계까지, 성장 한계는
	// 제자리까지(무한히 자랄 수 있다고 표시된 트랙만 계속). 모두 멈추고도 남으면 그때 넘쳐서 나눈다.
	// 한계가 무한한 트랙(항목이 없는 auto 트랙)은 그래서 다른 트랙이 내용 크기에 머무는 동안 나머지를
	// 다 받는다. 한 무리 안에서는 항목마다 낸 증가분 가운데 큰 쪽을 트랙에 더한다.
	var spanning []*gridItem
	for _, it := range items {
		if a, b := span(it); b-a > 1 && !hasFr(a, b) {
			spanning = append(spanning, it)
		}
	}
	spanLen := func(it *gridItem) int { a, b := span(it); return b - a }
	sort.SliceStable(spanning, func(i, j int) bool { return spanLen(spanning[i]) < spanLen(spanning[j]) })
	growable := make([]bool, n)
	// fill은 space를 idx 트랙에 고르게 나눈다 — room(i)에 닿은 트랙은 멈춘다. 다 나누지 못한 몫을 낸다.
	fill := func(space float64, idx []int, inc []float64, room func(i int) float64) float64 {
		active := idx
		for space > 1e-9 && len(active) > 0 {
			share := space / float64(len(active))
			var next []int
			for _, i := range active {
				if r := math.Max(0, room(i)-inc[i]); r <= share {
					inc[i] += r
					space -= r
				} else {
					next = append(next, i)
				}
			}
			if len(next) == len(active) {
				for _, i := range next {
					inc[i] += share
				}
				return 0
			}
			active = next
		}
		return space
	}
	// grow는 한 무리의 항목을 받아 affected 트랙의 base(growth=false) 또는 성장 한계(growth=true)를
	// 키운다. minC는 최소 기여를 쓸지, beyond는 한계를 넘쳐 받을 트랙이다 (없으면 affected 전부).
	grow := func(group []*gridItem, affected func(t track) bool, minC, growth bool, beyond func(t track) bool, markGrowable bool) {
		planned := make([]float64, n)
		for _, it := range group {
			a, b := span(it)
			space := contrib(it, minC) - gap*float64(b-a-1)
			var idx []int
			for i := a; i < b; i++ {
				if growth && !math.IsInf(limit[i], 1) {
					space -= limit[i]
				} else {
					space -= base[i]
				}
				if affected(tracks[i]) {
					idx = append(idx, i)
				}
			}
			if space <= 1e-9 || len(idx) == 0 {
				continue
			}
			inc := make([]float64, n)
			space = fill(space, idx, inc, func(i int) float64 {
				if !growth {
					return limit[i] - base[i]
				}
				if growable[i] || math.IsInf(limit[i], 1) {
					return math.Inf(1)
				}
				return 0
			})
			if space > 1e-9 {
				var over []int
				for _, i := range idx {
					if beyond(tracks[i]) {
						over = append(over, i)
					}
				}
				if len(over) == 0 {
					over = idx
				}
				fill(space, over, inc, func(int) float64 { return math.Inf(1) })
			}
			for i := range planned {
				planned[i] = math.Max(planned[i], inc[i])
			}
		}
		for i, p := range planned {
			switch {
			case p <= 0:
			case !growth:
				base[i] += p
			case math.IsInf(limit[i], 1):
				limit[i] = base[i] + p
				growable[i] = markGrowable
			default:
				limit[i] += p
			}
		}
	}
	all := func(track) bool { return true }
	maxContentMax := func(t track) bool { return t.maxKind == "auto" || t.maxKind == "max" }
	for g := 0; g < len(spanning); {
		h := g
		for h < len(spanning) && spanLen(spanning[h]) == spanLen(spanning[g]) {
			h++
		}
		group := spanning[g:h]
		g = h
		grow(group, intrinsicMin, true, false, intrinsicMax, false)
		grow(group, func(t track) bool { return t.minKind == "min" || t.minKind == "max" }, true, false, intrinsicMax, false)
		grow(group, func(t track) bool { return t.minKind == "max" }, false, false, maxContentMax, false)
		for i := range limit {
			if limit[i] < base[i] {
				limit[i] = base[i]
			}
		}
		grow(group, intrinsicMax, true, true, all, true)
		grow(group, maxContentMax, false, true, all, false)
	}
	// 끝까지 한계가 무한한 내재 트랙(항목이 없는 트랙)은 base에서 멈춘다.
	for i, t := range tracks {
		if intrinsicMax(t) && math.IsInf(limit[i], 1) {
			limit[i] = base[i]
		}
	}
	gaps := gap * float64(max(n-1, 0))
	// 3. 최대화: 남는 공간을 limit까지 나눈다 (fr이 아닌 트랙).
	sizes := append([]float64(nil), base...)
	if availKnown {
		free := avail - gaps
		for i := range sizes {
			free -= sizes[i]
		}
		for iter := 0; free > 1e-9 && iter < 8; iter++ {
			var grow []int
			for i, t := range tracks {
				if t.maxKind != "fr" && sizes[i] < limit[i] && !math.IsInf(limit[i], 1) {
					grow = append(grow, i)
				}
			}
			if len(grow) == 0 {
				break
			}
			share := free / float64(len(grow))
			for _, i := range grow {
				add := math.Min(share, limit[i]-sizes[i])
				sizes[i] += add
				free -= add
			}
		}
	} else {
		for i, t := range tracks {
			if t.maxKind != "fr" && !math.IsInf(limit[i], 1) {
				sizes[i] = limit[i]
			}
		}
	}
	// 4. fr
	frTotal := 0.0
	for _, t := range tracks {
		if t.maxKind == "fr" {
			frTotal += t.maxVal
		}
	}
	if frTotal > 0 {
		if availKnown {
			// 기여가 fr 몫보다 큰 트랙은 굳히고 다시 나눈다.
			fixed := map[int]bool{}
			for iter := 0; iter < n+1; iter++ {
				left := avail - gaps
				frSum := 0.0
				for i, t := range tracks {
					if t.maxKind == "fr" && !fixed[i] {
						frSum += t.maxVal
					} else {
						left -= sizes[i]
					}
				}
				if frSum == 0 {
					break
				}
				unit := math.Max(0, left) / math.Max(frSum, 1)
				if frSum < 1 {
					unit = math.Max(0, left)
				}
				changed := false
				for i, t := range tracks {
					if t.maxKind != "fr" || fixed[i] {
						continue
					}
					v := unit * t.maxVal
					if v < base[i] {
						sizes[i] = base[i]
						fixed[i] = true
						changed = true
					} else {
						sizes[i] = v
					}
				}
				if !changed {
					break
				}
			}
		} else {
			// 크기가 미정이면 fr 하나의 크기는 가장 큰 (기여 / fr)이다.
			unit := 0.0
			for _, it := range items {
				a, b := span(it)
				fr := 0.0
				other := gap * float64(b-a-1)
				for i := a; i < b; i++ {
					if tracks[i].maxKind == "fr" {
						fr += tracks[i].maxVal
					} else {
						other += sizes[i]
					}
				}
				if fr > 0 {
					unit = math.Max(unit, (contrib(it, false)-other)/math.Max(fr, 1))
				}
			}
			for i, t := range tracks {
				if t.maxKind == "fr" {
					sizes[i] = math.Max(base[i], unit*t.maxVal)
				}
			}
		}
	}
	// 5. stretch: content 정렬이 normal/stretch면 남는 공간을 auto 트랙에 고르게 준다.
	if availKnown && (content == "" || content == "normal" || content == "stretch") {
		free := avail - gaps
		for _, s := range sizes {
			free -= s
		}
		if free > 0 {
			var autos []int
			for i, t := range tracks {
				if t.maxKind == "auto" {
					autos = append(autos, i)
				}
			}
			for _, i := range autos {
				sizes[i] += free / float64(len(autos))
			}
		}
	}
	return sizes
}

// layoutGrid는 grid 배치를 하고 내용 높이를 낸다.
func (e *engine) layoutGrid(b *box, innerW float64, innerH *float64, fr edges) float64 {
	items, cols, rows := e.gridSetup(b)
	rowGap, colGap := b.gaps(innerW, innerH)
	justifyItems := b.st.get("justify-items")
	alignItems := b.st.get("align-items")

	colSizes := e.sizeTracks(cols, items, true, innerW, true, colGap, func(it *gridItem, min bool) float64 {
		return e.gridContribution(it.b, min)
	}, b.st.get("justify-content"))
	colPos := trackPositions(colSizes, colGap, innerW, true, b.st.get("justify-content"))

	areaW := func(it *gridItem) float64 {
		return colPos[it.c1-1] + colSizes[it.c1-1] - colPos[it.c0]
	}
	// 항목 폭 (justify-self)
	widths := map[*gridItem]float64{}
	for _, it := range items {
		k := it.b
		aw := areaW(it)
		it.m = k.margin(aw)
		var w float64
		if sw, ok := k.specified("width", aw, true); ok {
			w = sw
		} else if justifySelf(k, justifyItems) == "stretch" && !(k.replaced != "" || k.ratio() > 0 && k.length("height").auto == false) {
			w = aw - it.m.h()
		} else {
			w = math.Min(e.maxContent(k), math.Max(e.minContent(k), aw-it.m.h()))
		}
		widths[it] = k.clampW(w, aw, true)
	}

	// 행 크기: 항목을 정한 폭으로 배치한 높이가 기여다.
	heightOf := map[*gridItem]float64{}
	measure := func(it *gridItem) float64 {
		if h, ok := heightOf[it]; ok {
			return h
		}
		k := it.b
		e.layout(k, widths[it], nil, areaW(it), 0, false)
		heightOf[it] = k.h + it.m.v()
		return heightOf[it]
	}
	availH, hKnown := 0.0, innerH != nil
	if hKnown {
		availH = *innerH
	}
	rowSizes := e.sizeTracks(rows, items, false, availH, hKnown, rowGap, func(it *gridItem, min bool) float64 {
		if min {
			if v, ok := rowMinContribution(it); ok {
				return v
			}
		}
		return measure(it)
	}, b.st.get("align-content"))
	total := 0.0
	for i, s := range rowSizes {
		if i > 0 {
			total += rowGap
		}
		total += s
	}
	contentH := total
	if hKnown {
		contentH = availH
	}
	rowPos := trackPositions(rowSizes, rowGap, contentH, hKnown, b.st.get("align-content"))

	areas, _, _ := parseAreas(b.st.get("grid-template-areas"))
	// 절대 배치 자식의 포함 블록: 줄을 정한 변은 그 줄, auto인 변은 padding box 끝이다.
	br := b.border()
	padW, padH := innerW+fr.h()-br.h(), contentH+fr.v()-br.v()
	if innerH == nil {
		padH = total + fr.v() - br.v()
	}
	edge := func(startV, endV string, rowsAxis bool) (float64, float64, bool) {
		pos, sizes, padStart, padEnd := colPos, colSizes, br.l, br.l+padW
		if rowsAxis {
			pos, sizes, padStart, padEnd = rowPos, rowSizes, br.t, br.t+padH
		}
		n := len(sizes)
		lineAt := func(i int) float64 {
			if i <= 0 {
				return pos[0]
			}
			if i >= n {
				return pos[n-1] + sizes[n-1]
			}
			return pos[i]
		}
		off := fr.l
		if rowsAxis {
			off = fr.t
		}
		startV, endV = strings.TrimSpace(startV), strings.TrimSpace(endV)
		lo, hi := padStart, padEnd
		explicit := false
		startLine := -1
		if a, ok := areas[startV]; ok {
			s, e := a.c0, a.c1
			if rowsAxis {
				s, e = a.r0, a.r1
			}
			lo, startLine, explicit = off+lineAt(s), s, true
			if endV == "" || endV == "auto" || endV == startV {
				hi = off + lineAt(e)
				return lo, hi, true
			}
		} else if v, err := strconv.Atoi(startV); err == nil && v > 0 {
			lo, startLine, explicit = off+lineAt(v-1), v-1, true
		}
		if v, err := strconv.Atoi(endV); err == nil && v > 0 {
			hi, explicit = off+lineAt(v-1), true
		} else if strings.HasPrefix(endV, "span") && startLine >= 0 {
			sp, _ := strconv.Atoi(strings.TrimSpace(strings.TrimPrefix(endV, "span")))
			hi = off + lineAt(startLine+max(sp, 1))
		}
		return lo, hi, explicit
	}
	b.gridArea = func(k *box) (rect, bool) {
		x0, x1, okX := edge(k.st.get("grid-column-start"), k.st.get("grid-column-end"), false)
		y0, y1, okY := edge(k.st.get("grid-row-start"), k.st.get("grid-row-end"), true)
		if !okX && !okY {
			return rect{}, false
		}
		return rect{x0, y0, x1 - x0, y1 - y0}, true
	}
	for _, it := range items {
		k := it.b
		ax := colPos[it.c0]
		aw := areaW(it)
		ay := rowPos[it.r0]
		ah := rowPos[it.r1-1] + rowSizes[it.r1-1] - ay
		w := widths[it]
		var hp *float64
		self := alignSelf(k, alignItems)
		if sh, ok := k.specified("height", ah, true); ok {
			v := k.clampH(sh, ah, true, aw)
			hp = &v
		} else if self == "stretch" && k.replaced == "" && k.ratio() == 0 {
			v := k.clampH(ah-it.m.v(), ah, true, aw)
			hp = &v
		}
		e.layout(k, w, hp, aw, ah, true)
		jx := 0.0
		switch justifySelf(k, justifyItems) {
		case "center":
			jx = (aw - it.m.h() - k.w) / 2
		case "end":
			jx = aw - it.m.h() - k.w
		}
		jy := 0.0
		switch self {
		case "center":
			jy = (ah - it.m.v() - k.h) / 2
		case "end":
			jy = ah - it.m.v() - k.h
		}
		k.x = fr.l + ax + it.m.l + jx
		k.y = fr.t + ay + it.m.t + jy
	}
	return total
}

// rowMinContribution은 행 트랙에 들어가는 항목의 최소 기여다 — 열의 gridContribution과 같은 규칙이다.
// 높이가 auto이거나 칸에 기대는 퍼센트이고 min-height가 정해져 있으면 최소 기여는 그 값이다
// (내용 높이가 아니다). 정해진 px 높이나 auto min-height면 ok=false — 내용 기반 기여를 쓴다.
//
// 이 규칙이 없으면 `1fr`(= minmax(auto, 1fr)) 행의 바닥이 내용 높이가 된다: image-full의 media
// 그림(min-height: 0, height: 100%)이 폭 1280에서 그림 비율대로 잰 높이로 행을 굳혀, 아래
// `auto` 행(caption)이 스테이지 밖으로 밀려난다. 브라우저는 min-height: 0을 바닥으로 삼아 fr
// 행이 남는 공간에 맞춰 줄어든다.
func rowMinContribution(it *gridItem) (float64, bool) {
	k := it.b
	if _, ok := k.specified("height", 0, false); ok {
		return 0, false
	}
	mh, ok := k.specified("min-height", 0, false)
	if !ok {
		return 0, false
	}
	return math.Max(mh, k.frame(0).v()) + it.m.v(), true
}

func justifySelf(k *box, justifyItems string) string {
	v := k.st.get("justify-self")
	if v == "" || v == "auto" {
		v = justifyItems
	}
	switch v {
	case "", "normal", "stretch", "legacy":
		return "stretch"
	case "start", "flex-start", "self-start", "left", "baseline":
		return "start"
	case "end", "flex-end", "self-end", "right":
		return "end"
	}
	return v
}

// trackPositions는 트랙 시작 위치다 (content 정렬로 남는 공간을 나눈다).
func trackPositions(sizes []float64, gap, avail float64, known bool, content string) []float64 {
	used := 0.0
	for i, s := range sizes {
		if i > 0 {
			used += gap
		}
		used += s
	}
	lead, between := 0.0, 0.0
	if known {
		lead, between = distribute(content, avail-used, len(sizes))
	}
	pos := make([]float64, len(sizes)+1)
	p := lead
	for i, s := range sizes {
		pos[i] = p
		p += s + gap + between
	}
	pos[len(sizes)] = p
	return pos
}

// gridContribution은 열 트랙에 들어가는 항목의 기여다. 최소 기여는 min-width가 정해져 있으면 그
// 값이다 (auto일 때만 내용 기반 최소 = min-content).
func (e *engine) gridContribution(k *box, min bool) float64 {
	if min {
		if mw := k.length("min-width"); !mw.auto {
			if w, ok := k.specified("width", 0, false); ok {
				return math.Max(mw.px, k.clampW(w, 0, false)) + k.margin(0).h()
			}
			return math.Max(mw.px, k.frame(0).h()) + k.margin(0).h()
		}
	}
	return e.contribution(k, min)
}
