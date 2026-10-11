package quick

import (
	"math"
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

// placeItems는 항목을 격자 칸에 놓는다 (명시 위치 → 행 우선 sparse 자동 배치).
func placeItems(kids []*box, areas map[string]area, cols int) ([]*gridItem, int, int) {
	occupied := map[[2]int]bool{}
	mark := func(it *gridItem) {
		for r := it.r0; r < it.r1; r++ {
			for c := it.c0; c < it.c1; c++ {
				occupied[[2]int{r, c}] = true
			}
		}
	}
	fits := func(r0, c0, rs, cs int) bool {
		for r := r0; r < r0+rs; r++ {
			for c := c0; c < c0+cs; c++ {
				if occupied[[2]int{r, c}] {
					return false
				}
			}
		}
		return true
	}
	type pending struct {
		it       *gridItem
		row, col axisPlace
	}
	var items []*gridItem
	var autos []pending
	rows := 0
	for _, k := range kids {
		it := &gridItem{b: k}
		row := resolveAxis(k.st.get("grid-row-start"), k.st.get("grid-row-end"), areas, true)
		col := resolveAxis(k.st.get("grid-column-start"), k.st.get("grid-column-end"), areas, false)
		items = append(items, it)
		if row.start >= 0 && col.start >= 0 {
			it.r0, it.r1, it.c0, it.c1 = row.start, row.start+row.span, col.start, col.start+col.span
			mark(it)
			rows, cols = max(rows, it.r1), max(cols, it.c1)
			continue
		}
		cols = max(cols, col.span)
		if col.start >= 0 {
			cols = max(cols, col.start+col.span)
		}
		autos = append(autos, pending{it, row, col})
	}
	cols = max(cols, 1)
	curR, curC := 0, 0
	for _, p := range autos {
		it := p.it
		switch {
		case p.row.start >= 0:
			// 행이 정해졌다: 그 행에서 들어갈 첫 열
			c := 0
			for !fits(p.row.start, c, p.row.span, p.col.span) {
				c++
			}
			it.r0, it.c0 = p.row.start, c
		case p.col.start >= 0:
			if p.col.start < curC {
				curR++
			}
			for !fits(curR, p.col.start, p.row.span, p.col.span) {
				curR++
			}
			it.r0, it.c0 = curR, p.col.start
			curC = p.col.start + p.col.span
		default:
			for {
				if curC+p.col.span > cols {
					curR++
					curC = 0
					continue
				}
				if fits(curR, curC, p.row.span, p.col.span) {
					break
				}
				curC++
			}
			it.r0, it.c0 = curR, curC
			curC += p.col.span
		}
		it.r1, it.c1 = it.r0+p.row.span, it.c0+p.col.span
		mark(it)
		rows, cols = max(rows, it.r1), max(cols, it.c1)
	}
	return items, rows, cols
}

func (e *engine) gridSetup(b *box) ([]*gridItem, []track, []track) {
	fs := b.fs()
	cols := parseTrackList(b.st.get("grid-template-columns"), fs)
	rows := parseTrackList(b.st.get("grid-template-rows"), fs)
	areas, areaRows, areaCols := parseAreas(b.st.get("grid-template-areas"))
	nCols := max(len(cols), areaCols)
	items, nr, nc := placeItems(ordered(b.kids), areas, nCols)
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
			limit[i] = -1 // 내용에서 정한다
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
		if limit[a] < 0 || (t.maxKind == "auto" || t.maxKind == "max" || t.maxKind == "min") {
			c := contrib(it, t.maxKind == "min")
			if limit[a] < 0 {
				limit[a] = c
			} else {
				limit[a] = math.Max(limit[a], c)
			}
		}
	}
	for i := range limit {
		if limit[i] < 0 {
			limit[i] = base[i]
		}
		if limit[i] < base[i] {
			limit[i] = base[i]
		}
	}
	// 2. 여러 트랙에 걸친 항목 — fr이 없는 구간에서 모자라는 만큼 auto 트랙에 고르게 더한다.
	for _, it := range items {
		a, b := span(it)
		if b-a <= 1 || hasFr(a, b) {
			continue
		}
		need := contrib(it, true) - gap*float64(b-a-1)
		have := 0.0
		var autos []int
		for i := a; i < b; i++ {
			have += base[i]
			if tracks[i].minKind == "auto" {
				autos = append(autos, i)
			}
		}
		if need > have && len(autos) > 0 {
			add := (need - have) / float64(len(autos))
			for _, i := range autos {
				base[i] += add
				limit[i] = math.Max(limit[i], base[i])
			}
		}
		needMax := contrib(it, false) - gap*float64(b-a-1)
		haveMax := 0.0
		for i := a; i < b; i++ {
			haveMax += limit[i]
		}
		if needMax > haveMax && len(autos) > 0 {
			add := (needMax - haveMax) / float64(len(autos))
			for _, i := range autos {
				limit[i] += add
			}
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
	rowSizes := e.sizeTracks(rows, items, false, availH, hKnown, rowGap, func(it *gridItem, _ bool) float64 {
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
