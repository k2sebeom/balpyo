package quick

import (
	"math"
	"strconv"
	"strings"
)

// box는 배치 단위 하나다. 좌표(x, y)는 부모 box의 border box 원점 기준이고 w · h는
// border box다 (balpyo CSS는 `* { box-sizing: border-box }`다).
type box struct {
	n       *node
	st      *style
	pseudo  bool
	display string

	kids      []*box       // 흐름 안의 자식 (block · flex/grid 항목 · 표 행/칸)
	abs       []*box       // 절대 배치 자식 (포함 블록 = 이 box의 padding box)
	inline    []inlineItem // 인라인 서식 맥락을 가지면 그 내용
	hasInline bool
	replaced  string // "img" · "svg"

	x, y, w, h float64
	lines      []lineBox

	// 절대 좌표 (slide 기준, transform 이전) — finalize가 채운다.
	ax, ay float64

	minCache, maxCache *float64
	contentMinCache    *float64
	// sizeless면 layout이 지정 높이를 auto로 본다 (내용 높이를 잴 때만 잠깐 켠다).
	sizeless bool
	last     *layoutKey // 직전 layout 호출의 입력 — 같으면 결과가 이미 box에 있다

	// gridArea는 grid 컨테이너가 절대 배치 자식에게 주는 포함 블록이다 (grid 배치 속성이 줄을
	// 정하면 그 영역, 아니면 false — padding box로 물러선다).
	gridArea func(k *box) (rect, bool)
}

// engine은 문서 하나의 배치 상태다.
type engine struct {
	fonts  *fontKit
	assets assetMap
	images map[string]*decodedImage
	lang   string
	stageW float64
}

func (b *box) positioned() bool {
	p := b.st.get("position")
	return p == "relative" || p == "absolute" || p == "fixed" || p == "sticky"
}

func (b *box) absolute() bool {
	p := b.st.get("position")
	return p == "absolute" || p == "fixed"
}

func (b *box) fs() float64 { return b.st.fontSize }

func (b *box) length(name string) length {
	return lengthOr(b.st.get(name), b.fs(), autoLength)
}

type edges struct{ t, r, b, l float64 }

func (e edges) h() float64 { return e.l + e.r }
func (e edges) v() float64 { return e.t + e.b }

// padding은 퍼센트를 포함 블록 폭으로 푼다 (CSS: 네 변 모두 폭 기준).
func (b *box) padding(cbw float64) edges {
	get := func(side string) float64 {
		l := lengthOr(b.st.get("padding-"+side), b.fs(), length{})
		if l.auto {
			return 0
		}
		return math.Max(0, l.resolve(cbw))
	}
	return edges{get("top"), get("right"), get("bottom"), get("left")}
}

func (b *box) margin(cbw float64) edges {
	get := func(side string) float64 {
		l := lengthOr(b.st.get("margin-"+side), b.fs(), length{})
		if l.auto {
			return 0
		}
		return l.resolve(cbw)
	}
	return edges{get("top"), get("right"), get("bottom"), get("left")}
}

func (b *box) border() edges {
	get := func(side string) float64 {
		s := b.st.get("border-" + side + "-style")
		if s == "" || s == "none" || s == "hidden" {
			return 0
		}
		w := b.st.get("border-" + side + "-width")
		switch w {
		case "thin":
			return 1
		case "", "medium":
			return 3
		case "thick":
			return 5
		}
		l := lengthOr(w, b.fs(), length{})
		return math.Max(0, l.px)
	}
	if b.display == "table-cell" && b.parentTableCollapsed() {
		// border-collapse: 칸 사이 선 하나를 두 칸이 나눠 가진다 — 배치에서는 반씩.
		e := edges{get("top"), get("right"), get("bottom"), get("left")}
		return edges{e.t / 2, e.r / 2, e.b / 2, e.l / 2}
	}
	return edges{get("top"), get("right"), get("bottom"), get("left")}
}

func (b *box) parentTableCollapsed() bool {
	for cur := b.n; cur != nil; cur = cur.parent {
		if cur.tag == "table" {
			return cur.style.get("border-collapse") == "collapse"
		}
	}
	return false
}

// frame은 padding + border다 (border box와 content box의 차이).
func (b *box) frame(cbw float64) edges {
	p, br := b.padding(cbw), b.border()
	return edges{p.t + br.t, p.r + br.r, p.b + br.b, p.l + br.l}
}

// ratio는 aspect-ratio (w / h)다. 없으면 0.
func (b *box) ratio() float64 {
	v := b.st.get("aspect-ratio")
	if v == "" || v == "auto" {
		return 0
	}
	w, h, ok := strings.Cut(v, "/")
	if !ok {
		return numberOr(v, 0)
	}
	hw, hh := numberOr(strings.TrimSpace(w), 0), numberOr(strings.TrimSpace(h), 0)
	if hw <= 0 || hh <= 0 {
		return 0
	}
	return hw / hh
}

// ── 박스 트리 ──

func displayOf(n *node) string {
	d := n.style.get("display")
	if d == "" {
		d = "inline"
	}
	if n.attrs != nil {
		if _, hidden := n.attrs["hidden"]; hidden {
			return "none"
		}
	}
	return d
}

func blockify(d string) string {
	switch d {
	case "inline", "inline-block", "list-item":
		return "block"
	case "inline-flex":
		return "flex"
	case "inline-grid":
		return "grid"
	}
	return d
}

func isBlockLevel(d string) bool {
	switch d {
	case "block", "flex", "grid", "table", "list-item", "table-row", "table-cell", "table-row-group":
		return true
	}
	return false
}

func (e *engine) newBox(n *node, st *style, display string) *box {
	b := &box{n: n, st: st, display: display}
	switch n.tag {
	case "img", "svg":
		b.replaced = n.tag
	}
	return b
}

// build는 요소 하나의 box를 만든다 (display: none이면 nil).
func (e *engine) build(n *node, parentDisplay string) *box {
	d := displayOf(n)
	if d == "none" {
		return nil
	}
	pos := n.style.get("position")
	if pos == "absolute" || pos == "fixed" || parentDisplay == "flex" || parentDisplay == "grid" {
		d = blockify(d)
	}
	if d == "list-item" {
		d = "block"
	}
	b := e.newBox(n, n.style, d)
	if b.replaced != "" {
		return b
	}
	switch d {
	case "flex", "grid":
		e.buildItems(b)
	case "table":
		e.buildTable(b)
	case "table-row":
		e.buildRow(b)
	default:
		e.buildFlow(b)
	}
	return b
}

func isCollapsible(s string) bool { return strings.TrimSpace(s) == "" }

// buildItems는 flex · grid 항목을 만든다. 텍스트는 익명 항목이 된다 (공백뿐이면 버린다).
func (e *engine) buildItems(b *box) {
	if pb := e.beforeBox(b); pb != nil {
		pb.display = "block"
		b.kids = append(b.kids, pb)
	}
	var pending []*node
	flush := func() {
		if len(pending) == 0 {
			return
		}
		anon := &box{n: b.n, st: anonStyle(b.st), display: "block"}
		anon.hasInline = true
		for _, t := range pending {
			anon.inline = append(anon.inline, e.inlineItems(t, anon)...)
		}
		b.kids = append(b.kids, anon)
		pending = nil
	}
	for _, c := range b.n.children {
		if c.isText() {
			if !isCollapsible(c.text) {
				pending = append(pending, c)
			}
			continue
		}
		flush()
		kid := e.build(c, b.display)
		if kid == nil {
			continue
		}
		if kid.absolute() {
			b.abs = append(b.abs, kid)
		} else {
			b.kids = append(b.kids, kid)
		}
	}
	flush()
}

// anonStyle은 익명 box의 계산값이다 — 상속 속성만 물려받는다.
func anonStyle(parent *style) *style {
	st := &style{props: map[string]string{}, parent: parent, fontSize: parent.fontSize}
	for k, v := range parent.props {
		if inherited[k] || strings.HasPrefix(k, "--") {
			st.props[k] = v
		}
	}
	return st
}

// buildFlow는 block 컨테이너의 내용을 만든다: block 자식이 하나라도 있으면 block 배치(연속한
// 인라인 내용은 익명 block으로 감싼다), 아니면 인라인 서식 맥락이다.
func (e *engine) buildFlow(b *box) {
	hasBlock := false
	for _, c := range b.n.children {
		if c.isText() {
			continue
		}
		d := displayOf(c)
		if d == "none" {
			continue
		}
		pos := c.style.get("position")
		if pos == "absolute" || pos == "fixed" {
			continue
		}
		if isBlockLevel(d) {
			hasBlock = true
			break
		}
	}
	if !hasBlock {
		b.hasInline = true
		if pb := e.beforeBox(b); pb != nil {
			b.inline = append(b.inline, inlineItem{atomic: pb, st: pb.st})
		}
		for _, c := range b.n.children {
			b.inline = append(b.inline, e.inlineItems(c, b)...)
		}
		return
	}
	var pending []*node
	flush := func() {
		if len(pending) == 0 {
			return
		}
		blank := true
		for _, p := range pending {
			if !p.isText() || !isCollapsible(p.text) {
				blank = false
			}
		}
		if !blank {
			anon := &box{n: b.n, st: anonStyle(b.st), display: "block", hasInline: true}
			for _, p := range pending {
				anon.inline = append(anon.inline, e.inlineItems(p, b)...)
			}
			b.kids = append(b.kids, anon)
		}
		pending = nil
	}
	for _, c := range b.n.children {
		if c.isText() {
			pending = append(pending, c)
			continue
		}
		d := displayOf(c)
		if d == "none" {
			continue
		}
		pos := c.style.get("position")
		if pos == "absolute" || pos == "fixed" {
			if kid := e.build(c, b.display); kid != nil {
				b.abs = append(b.abs, kid)
			}
			continue
		}
		if !isBlockLevel(d) {
			pending = append(pending, c)
			continue
		}
		flush()
		if kid := e.build(c, b.display); kid != nil {
			b.kids = append(b.kids, kid)
		}
	}
	flush()
}

// beforeBox는 ::before 가상 요소의 box다 (content의 문자열 · 카운터를 편다).
func (e *engine) beforeBox(b *box) *box {
	if b.n.before == nil || b.pseudo {
		return nil
	}
	st := b.n.before
	text := e.contentText(b.n, st.get("content"))
	pseudoNode := &node{tag: "::before", attrs: map[string]string{}, parent: b.n, style: st}
	pseudoNode.children = []*node{{text: text, parent: pseudoNode, style: st}}
	d := st.get("display")
	if d == "" {
		d = "inline"
	}
	pb := &box{n: pseudoNode, st: st, display: d, pseudo: true}
	pb.hasInline = true
	pb.inline = []inlineItem{{text: text, st: st}}
	return pb
}

// contentText는 content 값을 편다 — 문자열과 counter()만 쓴다.
func (e *engine) contentText(n *node, v string) string {
	var b strings.Builder
	for _, part := range splitValues(v) {
		switch {
		case strings.HasPrefix(part, `"`) || strings.HasPrefix(part, `'`):
			b.WriteString(unquoteCSS(part))
		case strings.HasPrefix(part, "counter("):
			b.WriteString(strconv.Itoa(counterValue(n)))
		}
	}
	return b.String()
}

func unquoteCSS(s string) string {
	if len(s) >= 2 {
		s = s[1 : len(s)-1]
	}
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == '\\' && i+1 < len(s) {
			// \2022 같은 16진 이스케이프
			j := i + 1
			for j < len(s) && j < i+7 && isHex(s[j]) {
				j++
			}
			if j > i+1 {
				if r, err := strconv.ParseUint(s[i+1:j], 16, 32); err == nil {
					b.WriteRune(rune(r))
				}
				if j < len(s) && s[j] == ' ' {
					j++
				}
				i = j - 1
				continue
			}
			i++
			b.WriteByte(s[i])
			continue
		}
		b.WriteByte(s[i])
	}
	return b.String()
}

func isHex(c byte) bool {
	return c >= '0' && c <= '9' || c >= 'a' && c <= 'f' || c >= 'A' && c <= 'F'
}

// counterValue는 li의 순번이다 (ol의 counter-reset · li의 counter-increment 꼴만 쓴다).
func counterValue(n *node) int {
	if n.parent == nil {
		return 1
	}
	count := 0
	for _, sib := range n.parent.children {
		if sib.isText() {
			continue
		}
		if sib.tag == n.tag {
			count++
		}
		if sib == n {
			break
		}
	}
	return count
}

// ── 크기 ──

// specified는 폭 · 높이 속성을 기준 길이로 푼 border-box 값이다 (auto · 기준 없는 퍼센트면 false).
func (b *box) specified(name string, base float64, baseKnown bool) (float64, bool) {
	l := b.length(name)
	if l.auto {
		return 0, false
	}
	if l.pct != 0 && !baseKnown {
		return 0, false
	}
	return l.resolve(base), true
}

// clampW는 min/max-width를 적용한다.
func (b *box) clampW(w, cbw float64, cbKnown bool) float64 {
	if mx, ok := b.specified("max-width", cbw, cbKnown); ok && w > mx {
		w = mx
	}
	if mn, ok := b.specified("min-width", cbw, cbKnown); ok && w < mn {
		w = mn
	}
	return math.Max(w, b.frame(cbw).h())
}

func (b *box) clampH(h, cbh float64, cbKnown bool, cbw float64) float64 {
	if mx, ok := b.specified("max-height", cbh, cbKnown); ok && h > mx {
		h = mx
	}
	if mn, ok := b.specified("min-height", cbh, cbKnown); ok && h < mn {
		h = mn
	}
	return math.Max(h, b.frame(cbw).v())
}

// minContent · maxContent는 border-box 폭의 내재 크기다.
func (e *engine) minContent(b *box) float64 {
	if b.minCache != nil {
		return *b.minCache
	}
	v := e.intrinsic(b, true)
	b.minCache = &v
	return v
}

func (e *engine) maxContent(b *box) float64 {
	if b.maxCache != nil {
		return *b.maxCache
	}
	v := e.intrinsic(b, false)
	b.maxCache = &v
	return v
}

// contribution은 부모 내재 크기에 들어가는 바깥 폭이다 (지정 폭이 있으면 그것).
func (e *engine) contribution(b *box, min bool) float64 {
	m := b.margin(0)
	if w, ok := b.specified("width", 0, false); ok {
		return b.clampW(w, 0, false) + m.h()
	}
	var w float64
	if min {
		w = e.minContent(b)
	} else {
		w = e.maxContent(b)
	}
	return b.clampW(w, 0, false) + m.h()
}

func (e *engine) intrinsic(b *box, min bool) float64 {
	if w, ok := b.specified("width", 0, false); ok {
		return b.clampW(w, 0, false)
	}
	return e.contentIntrinsic(b, min)
}

// contentMinContent는 지정 폭을 보지 않은 내용의 min-content 폭(border-box)이다 — flex 항목의
// 자동 최소 크기가 지정 크기와 견주는 내용 기반 값이다.
func (e *engine) contentMinContent(b *box) float64 {
	if _, ok := b.specified("width", 0, false); !ok {
		return e.minContent(b)
	}
	if b.contentMinCache != nil {
		return *b.contentMinCache
	}
	v := e.contentIntrinsic(b, true)
	b.contentMinCache = &v
	return v
}

// contentIntrinsic은 지정 폭 없이 내용으로 정한 내재 폭이다.
func (e *engine) contentIntrinsic(b *box, min bool) float64 {
	fr := b.frame(0).h()
	if b.replaced != "" {
		w, h := e.naturalSize(b)
		if r := b.ratio(); r > 0 {
			if hh, ok := b.specified("height", 0, false); ok {
				return hh * r
			}
		} else if hh, ok := b.specified("height", 0, false); ok && h > 0 {
			return hh * w / h
		}
		if min && (b.length("max-width").pct > 0 || b.length("width").pct > 0) {
			// 퍼센트 폭의 대체 요소는 줄어들 수 있다 — min-content 기여는 0이다 (CSS Sizing §5.2.2).
			return fr
		}
		return w
	}
	var content float64
	switch {
	case b.hasInline:
		content = e.inlineIntrinsic(b, min)
	case b.display == "flex":
		content = e.flexIntrinsic(b, min)
	case b.display == "grid":
		content = e.gridIntrinsic(b, min)
	case b.display == "table":
		mins, maxs := e.tableColumns(b)
		content = e.collapsedFrame(b).h()
		for i := range mins {
			if min {
				content += mins[i]
			} else {
				content += maxs[i]
			}
		}
	default:
		for _, k := range b.kids {
			content = math.Max(content, e.contribution(k, min))
		}
	}
	return content + fr
}

// ── 배치 ──

// layout은 border-box 폭 w로 b를 배치하고 높이를 정한다. h가 nil이 아니면 높이가 정해져 있다
// (stretch · flex로 정해진 크기, 정해진 퍼센트 높이).
type layoutKey struct {
	w, h, cbw, cbh  float64
	hKnown, cbKnown bool
	sizeless        bool
}

func (e *engine) layout(b *box, w float64, h *float64, cbw, cbh float64, cbhKnown bool) {
	key := layoutKey{w: w, cbw: cbw, cbh: cbh, cbKnown: cbhKnown, sizeless: b.sizeless}
	if h != nil {
		key.h, key.hKnown = *h, true
	}
	if b.last != nil && *b.last == key {
		return
	}
	b.last = &key
	b.w = w
	fr := b.frame(cbw)
	innerW := math.Max(0, w-fr.h())

	// 높이가 정해졌는가: 호출자가 주었거나, 지정 높이를 기준으로 풀 수 있거나.
	var defH *float64
	if h != nil {
		v := *h
		defH = &v
	} else if sh, ok := b.specified("height", cbh, cbhKnown); ok && !b.sizeless {
		v := b.clampH(sh, cbh, cbhKnown, cbw)
		defH = &v
	} else if r := b.ratio(); r > 0 && !b.sizeless {
		v := b.clampH(w/r, cbh, cbhKnown, cbw)
		defH = &v
	}
	var innerH *float64
	if defH != nil {
		v := math.Max(0, *defH-fr.v())
		innerH = &v
	}

	var contentH float64
	switch {
	case b.replaced != "":
		if defH == nil {
			rw, rh := e.replacedSize(b, cbw, cbh, cbhKnown)
			contentH = rh - fr.v()
			if math.Abs(rw-w) > 0.01 && rw > 0 {
				// 폭이 바깥에서 정해졌다 (flex · stretch): 비율로 높이를 낸다.
				contentH = w*rh/rw - fr.v()
			}
		}
	case b.hasInline:
		contentH = e.layoutInline(b, innerW, fr)
	case b.display == "flex":
		contentH = e.layoutFlex(b, innerW, innerH, fr)
	case b.display == "grid":
		contentH = e.layoutGrid(b, innerW, innerH, fr)
	case b.display == "table":
		contentH = e.layoutTable(b, innerW, innerH, fr)
	default:
		contentH = e.layoutBlock(b, innerW, innerH, fr)
	}

	if defH != nil {
		b.h = *defH
	} else {
		b.h = b.clampH(contentH+fr.v(), cbh, cbhKnown, cbw)
	}
	e.layoutAbsolute(b)
}

// layoutBlock은 block 자식을 위에서 아래로 쌓는다 (margin 상쇄는 없다 — balpyo의 블록은
// margin을 0으로 둔다).
func (e *engine) layoutBlock(b *box, innerW float64, innerH *float64, fr edges) float64 {
	y := 0.0
	cbh, cbhKnown := 0.0, false
	if innerH != nil {
		cbh, cbhKnown = *innerH, true
	}
	for _, k := range b.kids {
		m := k.margin(innerW)
		w, ok := k.specified("width", innerW, true)
		if !ok {
			w = innerW - m.h()
		}
		w = k.clampW(w, innerW, true)
		e.layout(k, w, nil, innerW, cbh, cbhKnown)
		k.x = fr.l + m.l
		k.y = fr.t + y + m.t
		if align := b.st.get("text-align"); w < innerW-m.h() && k.st.get("margin-left") == "auto" {
			_ = align
			k.x = fr.l + (innerW-w)/2
		}
		y += m.t + k.h + m.b
	}
	return y
}

// layoutAbsolute는 절대 배치 자식을 이 box의 padding box에 놓는다.
func (e *engine) layoutAbsolute(b *box) {
	if len(b.abs) == 0 {
		return
	}
	br := b.border()
	for _, k := range b.abs {
		cb := rect{br.l, br.t, math.Max(0, b.w-br.h()), math.Max(0, b.h-br.v())}
		if b.gridArea != nil {
			if area, ok := b.gridArea(k); ok {
				cb = area
			}
		}
		pw, ph := cb.w, cb.h
		left, lok := k.specified("left", pw, true)
		right, rok := k.specified("right", pw, true)
		top, tok := k.specified("top", ph, true)
		bottom, bok := k.specified("bottom", ph, true)
		m := k.margin(pw)
		w, wok := k.specified("width", pw, true)
		if !wok {
			if lok && rok {
				w = pw - left - right - m.h()
			} else {
				// shrink-to-fit
				avail := pw - m.h()
				if lok {
					avail -= left
				}
				w = math.Min(math.Max(e.minContent(k), avail), e.maxContent(k))
			}
		}
		w = k.clampW(w, pw, true)
		var hp *float64
		if hh, ok := k.specified("height", ph, true); ok {
			v := k.clampH(hh, ph, true, pw)
			hp = &v
		} else if tok && bok {
			v := k.clampH(ph-top-bottom-m.v(), ph, true, pw)
			hp = &v
		}
		e.layout(k, w, hp, pw, ph, true)
		switch {
		case lok:
			k.x = cb.x + left + m.l
		case rok:
			k.x = cb.x + pw - right - m.r - k.w
		default:
			k.x = cb.x + m.l
		}
		switch {
		case tok:
			k.y = cb.y + top + m.t
		case bok:
			k.y = cb.y + ph - bottom - m.b - k.h
		default:
			k.y = cb.y + m.t
		}
	}
}

// ── 좌표 확정 ──

// finalize는 상대 좌표를 slide 기준 절대 좌표로 바꾼다. translate는 getBoundingClientRect가
// 보는 값이라 측정 좌표에 넣는다 (회전은 측정 뷰가 끄므로 넣지 않는다).
func finalize(b *box, ox, oy float64) {
	tx, ty := translateOf(b)
	b.ax, b.ay = ox+b.x+tx, oy+b.y+ty
	for _, k := range b.kids {
		finalize(k, b.ax, b.ay)
	}
	for _, k := range b.abs {
		finalize(k, b.ax, b.ay)
	}
	for _, it := range b.inline {
		if it.atomic != nil {
			finalize(it.atomic, b.ax, b.ay)
		}
	}
}

// translateOf는 translate 속성과 transform의 이동 성분이다.
func translateOf(b *box) (float64, float64) {
	var tx, ty float64
	if v := b.st.get("translate"); v != "" && v != "none" {
		parts := splitValues(v)
		tx = lengthOr(parts[0], b.fs(), length{}).resolve(b.w)
		if len(parts) > 1 {
			ty = lengthOr(parts[1], b.fs(), length{}).resolve(b.h)
		}
	}
	if v := b.st.get("transform"); v != "" && v != "none" {
		for _, fn := range splitValues(v) {
			name, args, ok := strings.Cut(fn, "(")
			if !ok {
				continue
			}
			args = strings.TrimSuffix(args, ")")
			parts := splitTop(args, ',')
			switch name {
			case "translateX":
				tx += lengthOr(parts[0], b.fs(), length{}).resolve(b.w)
			case "translateY":
				ty += lengthOr(parts[0], b.fs(), length{}).resolve(b.h)
			case "translate":
				tx += lengthOr(parts[0], b.fs(), length{}).resolve(b.w)
				if len(parts) > 1 {
					ty += lengthOr(parts[1], b.fs(), length{}).resolve(b.h)
				}
			}
		}
	}
	return tx, ty
}
