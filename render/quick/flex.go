package quick

import (
	"math"
	"sort"
	"strconv"
)

func flexRow(b *box) bool {
	d := b.st.get("flex-direction")
	return d == "" || d == "row" || d == "row-reverse"
}

func flexWraps(b *box) bool {
	w := b.st.get("flex-wrap")
	return w == "wrap" || w == "wrap-reverse"
}

func (b *box) gaps(innerW float64, innerH *float64) (row, col float64) {
	rg := lengthOr(b.st.get("row-gap"), b.fs(), length{})
	cg := lengthOr(b.st.get("column-gap"), b.fs(), length{})
	if !rg.auto {
		base := 0.0
		if innerH != nil {
			base = *innerH
		}
		row = rg.resolve(base)
	}
	if !cg.auto {
		col = cg.resolve(innerW)
	}
	return row, col
}

func orderOf(b *box) int {
	n, err := strconv.Atoi(b.st.get("order"))
	if err != nil {
		return 0
	}
	return n
}

// ordered는 order 속성으로 정렬한 항목이다 (같은 값은 문서 순서).
func ordered(kids []*box) []*box {
	out := append([]*box(nil), kids...)
	sort.SliceStable(out, func(i, j int) bool { return orderOf(out[i]) < orderOf(out[j]) })
	return out
}

func (e *engine) flexIntrinsic(b *box, min bool) float64 {
	_, colGap := b.gaps(0, nil)
	if !flexRow(b) {
		best := 0.0
		for _, k := range b.kids {
			best = math.Max(best, e.contribution(k, min))
		}
		return best
	}
	total, best := 0.0, 0.0
	for i, k := range b.kids {
		c := e.contribution(k, min)
		// flex-basis가 정해진 px면 max-content 기여는 그것이다.
		if l := k.length("flex-basis"); !l.auto && l.definite() && !min {
			c = math.Max(l.px, 0) + k.margin(0).h()
		}
		best = math.Max(best, c)
		total += c
		if i > 0 {
			total += colGap
		}
	}
	if min && flexWraps(b) {
		return best
	}
	return total
}

type flexItem struct {
	b                *box
	basis, hypo      float64 // 바깥 크기가 아닌 border-box 주축 크기
	inner            float64 // inner flex base size (shrink 가중치 — padding · border 제외)
	minMain, maxMain float64
	grow, shrink     float64
	main             float64
	frozen           bool
	marginMain       float64 // 주축 margin 합
	cross            float64 // border-box 교차축 크기
	m                edges
	crossSet         bool
	baseline         float64 // 바깥 교차 시작(margin 끝)에서 기준선까지 — baseline 정렬 항목만
}

func styleNumber(b *box, name string, def float64) float64 {
	v := b.st.get(name)
	if v == "" {
		return def
	}
	return numberOr(v, def)
}

// layoutFlex는 flex 배치를 하고 내용 높이를 낸다.
func (e *engine) layoutFlex(b *box, innerW float64, innerH *float64, fr edges) float64 {
	row := flexRow(b)
	rowGap, colGap := b.gaps(innerW, innerH)
	mainGap, crossGap := colGap, rowGap
	if !row {
		mainGap, crossGap = rowGap, colGap
	}
	mainSize, mainKnown := innerW, true
	if !row {
		mainKnown = innerH != nil
		if mainKnown {
			mainSize = *innerH
		}
	}
	cbh, cbhKnown := 0.0, innerH != nil
	if cbhKnown {
		cbh = *innerH
	}
	alignItems := b.st.get("align-items")

	kids := ordered(b.kids)
	items := make([]*flexItem, 0, len(kids))
	for _, k := range kids {
		it := &flexItem{b: k, m: k.margin(innerW)}
		it.grow = styleNumber(k, "flex-grow", 0)
		it.shrink = styleNumber(k, "flex-shrink", 1)
		if row {
			it.marginMain = it.m.h()
		} else {
			it.marginMain = it.m.v()
		}
		// 교차축 크기를 먼저 안다 (열 방향은 폭이 정해져야 높이를 잴 수 있다).
		if !row {
			it.cross = e.flexCrossWidth(b, k, innerW, alignItems, it.m, flexWraps(b))
		}
		it.basis = e.flexBasis(k, row, mainSize, mainKnown, innerW, cbh, cbhKnown, it.cross)
		it.minMain, it.maxMain = e.flexMinMax(k, row, mainSize, mainKnown, innerW, cbh, cbhKnown, it.cross)
		it.hypo = math.Max(it.minMain, math.Min(it.maxMain, it.basis))
		it.inner = math.Max(0, it.basis-k.frameMain(row, innerW))
		items = append(items, it)
	}

	// 줄 나누기
	var lines [][]*flexItem
	if flexWraps(b) && mainKnown {
		var cur []*flexItem
		used := 0.0
		for _, it := range items {
			outer := it.hypo + it.marginMain
			if len(cur) > 0 && used+mainGap+outer > mainSize+0.01 {
				lines = append(lines, cur)
				cur, used = nil, 0
			}
			if len(cur) > 0 {
				used += mainGap
			}
			used += outer
			cur = append(cur, it)
		}
		if len(cur) > 0 {
			lines = append(lines, cur)
		}
	} else if len(items) > 0 {
		lines = [][]*flexItem{items}
	}

	b.firstLine = 0
	if len(lines) > 0 {
		b.firstLine = len(lines[0])
	}

	// 주축 크기 확정 (줄마다)
	contentMain := 0.0
	for _, line := range lines {
		resolveFlexible(line, mainSize, mainKnown, mainGap)
		used := 0.0
		for i, it := range line {
			if i > 0 {
				used += mainGap
			}
			used += it.main + it.marginMain
		}
		contentMain = math.Max(contentMain, used)
	}
	if !mainKnown {
		mainSize = contentMain
	}

	// 교차축: 항목을 확정한 주축 크기로 배치해 교차 크기를 잰다. 행 방향의 baseline 항목은
	// 기준선을 맞춘 묶음으로 줄 교차 크기에 든다 (기준선 위 최대 + 기준선 아래 최대).
	lineCross := make([]float64, len(lines))
	lineBaseline := make([]float64, len(lines))
	for li, line := range lines {
		above, below := 0.0, 0.0
		for _, it := range line {
			k := it.b
			if row {
				hp := (*float64)(nil)
				if sh, ok := k.specified("height", cbh, cbhKnown); ok {
					v := k.clampH(sh, cbh, cbhKnown, innerW)
					hp = &v
				}
				e.layout(k, it.main, hp, innerW, cbh, cbhKnown)
				it.cross = k.h
				if alignSelf(k, alignItems) == "baseline" {
					it.baseline = it.m.t + itemBaseline(k)
					above = math.Max(above, it.baseline)
					below = math.Max(below, k.h+it.m.v()-it.baseline)
					continue
				}
				lineCross[li] = math.Max(lineCross[li], k.h+it.m.v())
			} else {
				h := it.main
				e.layout(k, it.cross, &h, innerW, cbh, cbhKnown)
				lineCross[li] = math.Max(lineCross[li], it.cross+it.m.h())
			}
		}
		lineCross[li] = math.Max(lineCross[li], above+below)
		lineBaseline[li] = above
	}
	crossSize := 0.0
	crossKnown := false
	if row && innerH != nil {
		crossSize, crossKnown = *innerH, true
	} else if !row {
		crossSize, crossKnown = innerW, true
	}
	if len(lines) == 1 && crossKnown && !flexWraps(b) {
		lineCross[0] = crossSize
	}
	// 여러 줄 컨테이너의 align-content: normal(= stretch)은 남는 교차 공간을 줄마다 똑같이 더한다
	// (줄이 하나여도 여러 줄 컨테이너면 같다).
	if crossKnown && flexWraps(b) && len(lines) > 0 {
		switch b.st.get("align-content") {
		case "", "normal", "stretch":
			used := crossGap * float64(len(lines)-1)
			for _, c := range lineCross {
				used += c
			}
			if free := crossSize - used; free > 0 {
				for li := range lineCross {
					lineCross[li] += free / float64(len(lines))
				}
			}
		}
	}

	// stretch: 교차 크기가 auto인 항목을 줄 교차 크기로 늘리고 다시 배치한다.
	for li, line := range lines {
		for _, it := range line {
			k := it.b
			self := alignSelf(k, alignItems)
			if self != "stretch" {
				continue
			}
			if row {
				if _, ok := k.specified("height", cbh, cbhKnown); ok {
					continue
				}
				h := k.clampH(lineCross[li]-it.m.v(), cbh, cbhKnown, innerW)
				if math.Abs(h-k.h) > 0.001 || true {
					e.layout(k, it.main, &h, innerW, cbh, cbhKnown)
				}
				it.cross = h
			} else {
				if _, ok := k.specified("width", innerW, true); ok {
					continue
				}
				w := k.clampW(lineCross[li]-it.m.h(), innerW, true)
				if math.Abs(w-it.cross) > 0.001 {
					h := it.main
					e.layout(k, w, &h, innerW, cbh, cbhKnown)
					it.cross = w
				}
			}
		}
	}

	// 줄을 교차축에 놓는다 (align-content는 여러 줄 컨테이너에서만 뜻이 있다).
	crossUsed := 0.0
	for i, c := range lineCross {
		if i > 0 {
			crossUsed += crossGap
		}
		crossUsed += c
	}
	lineStart := make([]float64, len(lines))
	{
		free := 0.0
		if crossKnown && flexWraps(b) {
			free = crossSize - crossUsed
		}
		lead, between := distribute(b.st.get("align-content"), free, len(lines))
		pos := lead
		for i := range lines {
			lineStart[i] = pos
			pos += lineCross[i] + crossGap + between
		}
	}

	justify := b.st.get("justify-content")
	reverse := b.st.get("flex-direction") == "row-reverse" || b.st.get("flex-direction") == "column-reverse"
	for li, line := range lines {
		used := 0.0
		for i, it := range line {
			if i > 0 {
				used += mainGap
			}
			used += it.main + it.marginMain
		}
		lead, between := distribute(justify, mainSize-used, len(line))
		pos := lead
		order := line
		if reverse {
			order = make([]*flexItem, len(line))
			for i := range line {
				order[i] = line[len(line)-1-i]
			}
			pos = mainSize - used - lead
			if pos < 0 {
				pos = 0
			}
		}
		for _, it := range order {
			k := it.b
			self := alignSelf(k, alignItems)
			var crossOuter float64
			if row {
				crossOuter = k.h + it.m.v()
			} else {
				crossOuter = k.w + it.m.h()
			}
			off := 0.0
			switch self {
			case "center":
				off = (lineCross[li] - crossOuter) / 2
			case "end", "flex-end", "self-end":
				off = lineCross[li] - crossOuter
			case "baseline":
				// 열 방향에서는 교차축이 글줄 방향이라 맞출 기준선이 없다 — flex-start로 물러선다.
				if row {
					off = lineBaseline[li] - it.baseline
				}
			}
			if row {
				k.x = fr.l + pos + it.m.l
				k.y = fr.t + lineStart[li] + off + it.m.t
				pos += k.w + it.marginMain + mainGap + between
			} else {
				k.y = fr.t + pos + it.m.t
				k.x = fr.l + lineStart[li] + off + it.m.l
				pos += k.h + it.marginMain + mainGap + between
			}
		}
	}

	if row {
		return crossUsed
	}
	return contentMain
}

// alignSelf는 항목의 교차축 정렬이다 (normal은 stretch, first baseline은 baseline).
func alignSelf(k *box, alignItems string) string {
	v := k.st.get("align-self")
	if v == "" || v == "auto" {
		v = alignItems
	}
	switch v {
	case "", "normal", "stretch":
		return "stretch"
	case "baseline", "first baseline":
		return "baseline"
	case "flex-start", "start", "self-start", "first", "last":
		return "start"
	case "flex-end", "end", "self-end":
		return "end"
	}
	return v
}

// itemBaseline은 flex 항목의 첫 기준선이다 (border box 위에서부터). 기준선이 없으면 border box의
// 아래 변으로 합성한다.
func itemBaseline(k *box) float64 {
	if v, ok := firstBaseline(k); ok {
		return v
	}
	return k.h
}

// firstBaseline은 box의 첫 기준선(border box 위에서부터)이다. 인라인 서식 맥락이면 첫 줄의
// 기준선, block이면 기준선을 가진 첫 자식의 것이다. flex · grid 컨테이너는 첫 항목의 기준선이고,
// 그 항목에 기준선이 없으면 border box로 합성한다. 대체 요소는 기준선이 없다.
func firstBaseline(b *box) (float64, bool) {
	if b.replaced != "" {
		return 0, false
	}
	if b.hasInline {
		if len(b.lines) == 0 {
			return 0, false
		}
		return b.lines[0].baseline, true
	}
	switch b.display {
	case "flex", "grid":
		kids := ordered(b.kids)
		if len(kids) == 0 {
			return 0, false
		}
		k := kids[0]
		if b.display == "flex" && flexRow(b) {
			// 첫 줄의 baseline 정렬 항목이 있으면 그 묶음의 기준선이 컨테이너의 것이다.
			for _, c := range kids[:min(b.firstLine, len(kids))] {
				if alignSelf(c, b.st.get("align-items")) == "baseline" {
					k = c
					break
				}
			}
		}
		return k.y + itemBaseline(k), true
	}
	for _, k := range b.kids {
		if v, ok := firstBaseline(k); ok {
			return k.y + v, true
		}
	}
	return 0, false
}

// distribute는 남는 공간을 justify/align-content 값으로 나눠 앞 여백과 항목 사이 추가 간격을 낸다.
func distribute(v string, free float64, n int) (lead, between float64) {
	if n == 0 {
		return 0, 0
	}
	switch v {
	case "center":
		return free / 2, 0
	case "end", "flex-end":
		return free, 0
	case "space-between":
		if free < 0 || n == 1 {
			return 0, 0
		}
		return 0, free / float64(n-1)
	case "space-around":
		if free < 0 {
			return free / 2, 0
		}
		return free / float64(n) / 2, free / float64(n)
	case "space-evenly":
		if free < 0 {
			return free / 2, 0
		}
		return free / float64(n+1), free / float64(n+1)
	}
	return 0, 0
}

// flexCrossWidth는 열 방향 항목의 폭이다: 지정 폭, stretch면 줄 폭, 아니면 fit-content.
// 여러 줄 컨테이너에서는 줄 폭이 아직 없으므로 stretch 항목도 fit-content로 재고(가상 교차 크기),
// 줄 폭이 정해진 뒤 늘린다.
func (e *engine) flexCrossWidth(b, k *box, innerW float64, alignItems string, m edges, multiLine bool) float64 {
	if w, ok := k.specified("width", innerW, true); ok {
		return k.clampW(w, innerW, true)
	}
	if k.replaced != "" && (alignSelf(k, alignItems) != "stretch" || multiLine) {
		w, _ := e.replacedSize(k, innerW, 0, false)
		return w
	}
	avail := innerW - m.h()
	if alignSelf(k, alignItems) == "stretch" && !multiLine {
		return k.clampW(avail, innerW, true)
	}
	w := math.Min(e.maxContent(k), math.Max(e.minContent(k), avail))
	return k.clampW(w, innerW, true)
}

// flexBasis는 항목의 flex base size다 (border-box).
func (e *engine) flexBasis(k *box, row bool, mainSize float64, mainKnown bool, innerW, cbh float64, cbhKnown bool, crossW float64) float64 {
	basis := k.length("flex-basis")
	if !basis.auto && (basis.pct == 0 || mainKnown) {
		return math.Max(basis.resolve(mainSize), k.frameMain(row, innerW))
	}
	if k.replaced != "" {
		w, h := e.replacedSize(k, innerW, cbh, cbhKnown)
		if row {
			return w
		}
		return h
	}
	if row {
		if w, ok := k.specified("width", innerW, true); ok {
			return w
		}
		if r := k.ratio(); r > 0 {
			if h, ok := k.specified("height", cbh, cbhKnown); ok {
				return h * r
			}
		}
		return e.maxContent(k)
	}
	if h, ok := k.specified("height", cbh, cbhKnown); ok {
		return h
	}
	if r := k.ratio(); r > 0 {
		return crossW / r
	}
	return e.contentHeight(k, crossW, innerW, cbh, cbhKnown)
}

func (b *box) frameMain(row bool, cbw float64) float64 {
	f := b.frame(cbw)
	if row {
		return f.h()
	}
	return f.v()
}

// contentHeight는 폭 w로 배치했을 때의 높이다 (높이 미정).
func (e *engine) contentHeight(k *box, w, cbw, cbh float64, cbhKnown bool) float64 {
	saveH := k.st.get("height")
	_ = saveH
	e.layout(k, w, nil, cbw, cbh, cbhKnown)
	return k.h
}

// sizelessHeight는 지정 높이를 auto로 보고 폭 w로 배치했을 때의 내용 높이다. 잰 뒤에는 배치
// 캐시를 비워 다음 배치가 지정 높이로 다시 하게 한다.
func (e *engine) sizelessHeight(k *box, w, cbw, cbh float64, cbhKnown bool) float64 {
	if _, ok := k.specified("height", cbh, cbhKnown); !ok {
		return e.contentHeight(k, w, cbw, cbh, cbhKnown)
	}
	k.sizeless = true
	e.layout(k, w, nil, cbw, cbh, cbhKnown)
	k.sizeless = false
	k.last = nil
	return k.h
}

// flexMinMax는 주축 min/max다. min이 auto면 내용 기반 최소(min-content)다.
func (e *engine) flexMinMax(k *box, row bool, mainSize float64, mainKnown bool, innerW, cbh float64, cbhKnown bool, crossW float64) (float64, float64) {
	minName, maxName := "min-width", "max-width"
	base, baseKnown := innerW, true
	if !row {
		minName, maxName = "min-height", "max-height"
		base, baseKnown = cbh, cbhKnown
	}
	maxV := math.Inf(1)
	if v, ok := k.specified(maxName, base, baseKnown); ok {
		maxV = v
	}
	minL := k.length(minName)
	minV := 0.0
	if !minL.auto {
		if minL.pct == 0 || baseKnown {
			minV = minL.resolve(base)
		}
	} else if ov := k.st.get("overflow"); ov == "" || ov == "visible" {
		// 자동 최소 크기: 내용 기반 최소(min-content)이고, 지정 크기가 있으면 둘 중 작은 쪽이다.
		// 지정 크기가 내용보다 크면 항목은 내용 크기까지 줄어든다.
		var content float64
		if row {
			if k.replaced != "" {
				content = e.minContent(k)
			} else {
				content = e.contentMinContent(k)
			}
			if w, ok := k.specified("width", innerW, true); ok {
				content = math.Min(content, w)
			}
			if k.replaced != "" && k.length("max-width").pct > 0 {
				content = 0
			}
		} else {
			h, hok := k.specified("height", cbh, cbhKnown)
			if r := k.ratio(); r > 0 && k.replaced == "" {
				content = crossW / r
			} else {
				content = e.sizelessHeight(k, crossW, innerW, cbh, cbhKnown)
			}
			if hok {
				content = math.Min(content, h)
			}
			if k.replaced != "" {
				content = 0
			}
		}
		minV = math.Min(content, maxV)
	}
	minV = math.Max(minV, k.frameMain(row, innerW))
	return minV, math.Max(maxV, minV)
}

// resolveFlexible은 CSS Flexbox §9.7의 유연한 길이 해석이다.
func resolveFlexible(line []*flexItem, mainSize float64, mainKnown bool, gap float64) {
	for _, it := range line {
		it.main = it.hypo
		it.frozen = false
	}
	if !mainKnown {
		return
	}
	sumHypo := gap * float64(max(len(line)-1, 0))
	for _, it := range line {
		sumHypo += it.hypo + it.marginMain
	}
	growing := sumHypo < mainSize
	for _, it := range line {
		if (growing && it.grow == 0) || (!growing && it.shrink == 0) ||
			(growing && it.basis > it.hypo) || (!growing && it.basis < it.hypo) {
			it.frozen = true
			it.main = it.hypo
		}
	}
	for iter := 0; iter < 20; iter++ {
		free := mainSize - gap*float64(max(len(line)-1, 0))
		active := 0
		sumFactor := 0.0
		for _, it := range line {
			if it.frozen {
				free -= it.main + it.marginMain
			} else {
				free -= it.basis + it.marginMain
				active++
				if growing {
					sumFactor += it.grow
				} else {
					sumFactor += it.shrink * it.inner
				}
			}
		}
		if active == 0 {
			break
		}
		if growing && sumFactor < 1 {
			// 합이 1보다 작으면 남는 공간의 그 비율만 나눈다.
			initial := mainSize - sumHypo
			if s := initial * sumFactor; math.Abs(s) < math.Abs(free) {
				free = s
			}
		}
		totalViolation := 0.0
		for _, it := range line {
			if it.frozen {
				continue
			}
			target := it.basis
			if sumFactor > 0 {
				if growing {
					target = it.basis + free*it.grow/sumFactor
				} else {
					target = it.basis + free*(it.shrink*it.inner)/sumFactor
				}
			}
			clamped := math.Max(it.minMain, math.Min(it.maxMain, target))
			totalViolation += clamped - target
			it.main = clamped
		}
		switch {
		case math.Abs(totalViolation) < 1e-9:
			for _, it := range line {
				it.frozen = true
			}
		case totalViolation > 0:
			for _, it := range line {
				if !it.frozen && it.main == it.minMain {
					it.frozen = true
				}
			}
		default:
			for _, it := range line {
				if !it.frozen && it.main == it.maxMain {
					it.frozen = true
				}
			}
		}
		done := true
		for _, it := range line {
			if !it.frozen {
				done = false
			}
		}
		if done {
			break
		}
	}
}
