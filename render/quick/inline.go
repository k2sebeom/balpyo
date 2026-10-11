package quick

import (
	"math"
	"strings"
	"unicode"

	"github.com/go-text/typesetting/segmenter"
)

// inlineItem은 인라인 내용 한 조각이다: 텍스트 · 강제 줄바꿈 · 원자 인라인(inline-block, 대체 요소).
type inlineItem struct {
	text   string
	st     *style
	br     bool
	atomic *box
}

// lineBox는 줄 하나다. 좌표는 소유 box의 border box 기준이다.
type lineBox struct {
	x, y, w, h float64
	baseline   float64
	frags      []frag
	// textRect는 텍스트 노드의 rect를 합친 것이다 (지오메트리 패스의 lines와 같은 정의).
	textRect rect
	hasText  bool
}

// frag는 줄 안의 셰이핑 조각이나 원자 인라인이다.
type frag struct {
	x, w     float64
	baseline float64 // 소유 box 기준 y
	st       *style
	glyphs   []glyph
	atomic   *box
	ascent   float64 // 글꼴 내용 영역 (px)
	descent  float64
	space    bool // 공백만인 조각
}

type rect struct{ x, y, w, h float64 }

func (r rect) right() float64  { return r.x + r.w }
func (r rect) bottom() float64 { return r.y + r.h }

func union(a, b rect, aOK bool) rect {
	if !aOK {
		return b
	}
	x0, y0 := math.Min(a.x, b.x), math.Min(a.y, b.y)
	x1, y1 := math.Max(a.right(), b.right()), math.Max(a.bottom(), b.bottom())
	return rect{x0, y0, x1 - x0, y1 - y0}
}

// inlineItems는 노드 하나의 인라인 내용을 편다. 절대 배치 자식은 owner의 abs로 간다.
func (e *engine) inlineItems(n *node, owner *box) []inlineItem {
	if n.isText() {
		return []inlineItem{{text: n.text, st: n.style}}
	}
	d := displayOf(n)
	if d == "none" {
		return nil
	}
	pos := n.style.get("position")
	if pos == "absolute" || pos == "fixed" {
		if b := e.build(n, owner.display); b != nil {
			owner.abs = append(owner.abs, b)
		}
		return nil
	}
	if n.tag == "br" {
		return []inlineItem{{br: true, st: n.style}}
	}
	if d == "inline-block" || d == "inline-flex" || d == "inline-grid" || n.tag == "img" || n.tag == "svg" {
		b := e.build(n, "")
		if b == nil {
			return nil
		}
		return []inlineItem{{atomic: b, st: n.style}}
	}
	var out []inlineItem
	pb := e.beforeBox(&box{n: n, st: n.style})
	if pb != nil {
		out = append(out, inlineItem{atomic: pb, st: pb.st})
	}
	for _, c := range n.children {
		out = append(out, e.inlineItems(c, owner)...)
	}
	return out
}

// para는 줄바꿈 전의 단락이다: 공백을 처리한 문자열과 문자별 출처.
type para struct {
	runes   []rune
	styles  []*style // 문자별 계산값
	atomics map[int]*box
	forced  map[int]bool // 이 문자 앞에서 반드시 끊는다
}

const objectChar = '￼'

func whiteSpace(st *style) string {
	v := st.get("white-space")
	if v == "" {
		return "normal"
	}
	return v
}

func preserves(ws string) bool { return ws == "pre" || ws == "pre-wrap" || ws == "break-spaces" }
func wraps(ws string) bool     { return ws != "nowrap" && ws != "pre" }

func buildPara(items []inlineItem) *para {
	p := &para{atomics: map[int]*box{}, forced: map[int]bool{}}
	lastSpace := true // 줄 시작의 공백은 버린다
	for _, it := range items {
		switch {
		case it.br:
			p.forced[len(p.runes)] = true
			lastSpace = true
		case it.atomic != nil:
			p.atomics[len(p.runes)] = it.atomic
			p.runes = append(p.runes, objectChar)
			p.styles = append(p.styles, it.st)
			lastSpace = false
		default:
			ws := whiteSpace(it.st)
			for _, r := range it.text {
				if preserves(ws) {
					if r == '\n' {
						p.forced[len(p.runes)] = true
						lastSpace = true
						continue
					}
					if r == '\t' {
						for i := 0; i < 8; i++ {
							p.runes = append(p.runes, ' ')
							p.styles = append(p.styles, it.st)
						}
						continue
					}
					p.runes = append(p.runes, r)
					p.styles = append(p.styles, it.st)
					lastSpace = false
					continue
				}
				if r == '\n' || r == '\t' || r == '\r' || r == '\f' {
					r = ' '
				}
				if r == ' ' {
					if lastSpace {
						continue
					}
					lastSpace = true
				} else {
					lastSpace = false
				}
				p.runes = append(p.runes, r)
				p.styles = append(p.styles, it.st)
			}
		}
	}
	return p
}

// word는 줄바꿈 기회 사이의 구간이다 [start, end). spaceStart부터는 줄 끝에서 걸리는 공백이다.
type word struct {
	start, end, spaceStart int
	mandatory              bool // 이 단어 앞에서 반드시 끊는다
}

func isCJKLetter(r rune) bool {
	return unicode.Is(unicode.Hangul, r) || unicode.Is(unicode.Han, r) ||
		unicode.Is(unicode.Hiragana, r) || unicode.Is(unicode.Katakana, r)
}

func isWordChar(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsNumber(r) || unicode.Is(unicode.Mn, r)
}

// isTableASCII는 브라우저의 ASCII 줄바꿈 표가 다루는 범위('!'부터 DEL까지)다.
func isTableASCII(r rune) bool { return r >= '!' && r <= 0x7F }

// words는 단락을 줄바꿈 기회로 자른다.
func (p *para) words() []word {
	if len(p.runes) == 0 {
		return nil
	}
	var seg segmenter.Segmenter
	seg.Init(p.runes)
	it := seg.LineIterator()
	breaks := map[int]bool{}
	for it.Next() {
		l := it.Line()
		end := l.Offset + len(l.Text)
		if end < len(p.runes) {
			breaks[end] = true
		}
	}
	for pos := range breaks {
		prev, next := p.runes[pos-1], p.runes[pos]
		st := p.styles[pos]
		if !wraps(whiteSpace(st)) || !wraps(whiteSpace(p.styles[pos-1])) {
			delete(breaks, pos)
			continue
		}
		// 앞뒤가 모두 출력 가능한 ASCII인 쌍은 브라우저가 UAX #14 대신 자기 표로 판정하고, 그 표는
		// '/' 뒤를 끊지 않는다 (URL · 경로가 한 단어로 남는다). ASCII가 아닌 글자 앞은 UAX #14대로다.
		if prev == '/' && isTableASCII(next) {
			delete(breaks, pos)
			continue
		}
		// keep-all: 낱자 사이의 기회를 지운다 (공백 · 구두점 자리만 남는다).
		if st.get("word-break") == "keep-all" && isWordChar(prev) && isWordChar(next) &&
			(isCJKLetter(prev) || isCJKLetter(next)) {
			delete(breaks, pos)
		}
	}
	for pos := range p.forced {
		if pos > 0 && pos < len(p.runes) {
			breaks[pos] = true
		}
	}
	var out []word
	start := 0
	cut := func(end int) {
		w := word{start: start, end: end, mandatory: p.forced[start] && start > 0}
		w.spaceStart = end
		for w.spaceStart > start && p.runes[w.spaceStart-1] == ' ' && !preserves(whiteSpace(p.styles[w.spaceStart-1])) {
			w.spaceStart--
		}
		out = append(out, w)
		start = end
	}
	for i := 1; i < len(p.runes); i++ {
		if breaks[i] {
			cut(i)
		}
	}
	cut(len(p.runes))
	return out
}

// piece는 단어 안에서 계산값이 같은 구간의 셰이핑 결과다.
type piece struct {
	start, end int
	st         *style
	shaped     shapedText
	atomic     *box
}

func (e *engine) shapeRange(p *para, start, end int) []piece {
	var out []piece
	for i := start; i < end; {
		if a, ok := p.atomics[i]; ok {
			out = append(out, piece{start: i, end: i + 1, st: p.styles[i], atomic: a})
			i++
			continue
		}
		j := i + 1
		for j < end && p.styles[j] == p.styles[i] {
			if _, ok := p.atomics[j]; ok {
				break
			}
			j++
		}
		st := p.styles[i]
		text := applyTransform(string(p.runes[i:j]), st.get("text-transform"))
		out = append(out, piece{start: i, end: j, st: st, shaped: e.fonts.shape(fontStyleOf(st), text)})
		i = j
	}
	return out
}

func (e *engine) atomicWidth(a *box, cbw float64) float64 {
	if w, ok := a.specified("width", cbw, true); ok {
		return a.clampW(w, cbw, true)
	}
	if a.replaced != "" {
		return e.maxContent(a)
	}
	return math.Min(e.maxContent(a), math.Max(e.minContent(a), cbw))
}

func (e *engine) piecesWidth(ps []piece, cbw float64) float64 {
	w := 0.0
	for i := range ps {
		if ps[i].atomic != nil {
			a := ps[i].atomic
			w += e.atomicWidth(a, cbw) + a.margin(cbw).h()
			continue
		}
		w += ps[i].shaped.width
	}
	return w
}

func textIndent(st *style, cbw float64) float64 {
	v := st.get("text-indent")
	if v == "" {
		return 0
	}
	return lengthOr(v, st.fontSize, length{}).resolve(cbw)
}

func anywhere(st *style) bool {
	v := st.get("overflow-wrap")
	if v == "" {
		v = st.get("word-wrap")
	}
	return v == "anywhere"
}

func breakWord(st *style) bool {
	v := st.get("overflow-wrap")
	if v == "" {
		v = st.get("word-wrap")
	}
	return v == "anywhere" || v == "break-word" || st.get("word-break") == "break-all"
}

// inlineIntrinsic은 인라인 내용의 내재 폭이다 (content box).
func (e *engine) inlineIntrinsic(b *box, min bool) float64 {
	p := buildPara(b.inline)
	ws := p.words()
	indent := textIndent(b.st, 0)
	best := 0.0
	line := indent
	for i, w := range ws {
		if w.mandatory && i > 0 {
			best = math.Max(best, line)
			line = 0
		}
		core := e.piecesWidth(e.shapeRange(p, w.start, w.spaceStart), 0)
		if min {
			wordW := core
			if anywhere(p.styles[w.start]) {
				wordW = e.widestCluster(p, w.start, w.spaceStart)
			}
			if i == 0 {
				wordW += indent
			}
			best = math.Max(best, wordW)
			continue
		}
		space := e.piecesWidth(e.shapeRange(p, w.spaceStart, w.end), 0)
		best = math.Max(best, line+core)
		line += core + space
	}
	if !min {
		best = math.Max(best, line-0)
	}
	return math.Max(0, best)
}

func (e *engine) widestCluster(p *para, start, end int) float64 {
	best := 0.0
	for i := start; i < end; i++ {
		best = math.Max(best, e.piecesWidth(e.shapeRange(p, i, i+1), 0))
	}
	return best
}

// placed는 줄에 놓인 조각이다 (줄 시작 기준 x).
type placed struct {
	piece
	x, w  float64
	trail bool // 줄 끝 공백
}

// layoutInline은 줄을 채우고 내용 높이를 낸다.
func (e *engine) layoutInline(b *box, innerW float64, fr edges) float64 {
	b.lines = nil
	p := buildPara(b.inline)
	ws := p.words()
	if len(p.runes) == 0 {
		return 0
	}
	indent := textIndent(b.st, innerW)

	var lines [][]placed
	var cur []placed
	x := indent
	first := true
	flush := func() {
		lines = append(lines, cur)
		cur = nil
		x = 0
		first = false
	}
	_ = first
	place := func(ps []piece, trail bool) {
		for _, pc := range ps {
			w := 0.0
			if pc.atomic != nil {
				a := pc.atomic
				aw := e.atomicWidth(a, innerW)
				e.layout(a, aw, nil, innerW, 0, false)
				w = aw + a.margin(innerW).h()
			} else {
				w = pc.shaped.width
			}
			cur = append(cur, placed{piece: pc, x: x, w: w, trail: trail})
			x += w
		}
	}
	const eps = 0.01
	for i, w := range ws {
		if w.mandatory && i > 0 {
			flush()
		}
		corePieces := e.shapeRange(p, w.start, w.spaceStart)
		core := e.piecesWidth(corePieces, innerW)
		if len(cur) > 0 && x+core > innerW+eps && wraps(whiteSpace(p.styles[w.start])) {
			flush()
		}
		if x+core > innerW+eps && breakWord(p.styles[w.start]) && wraps(whiteSpace(p.styles[w.start])) {
			// 한 단어가 줄보다 넓다: 글자 단위로 끊는다.
			for j := w.start; j < w.spaceStart; j++ {
				ps := e.shapeRange(p, j, j+1)
				cw := e.piecesWidth(ps, innerW)
				if len(cur) > 0 && x+cw > innerW+eps {
					flush()
				}
				place(ps, false)
			}
		} else {
			place(corePieces, false)
		}
		if w.spaceStart < w.end {
			place(e.shapeRange(p, w.spaceStart, w.end), true)
		}
	}
	flush()

	// 줄 높이와 기준선
	y := fr.t
	align := b.st.get("text-align")
	for li, items := range lines {
		strut := e.inlineMetrics(b.st)
		above, below := strut.above, strut.below
		for _, it := range items {
			if it.atomic != nil {
				a := it.atomic
				base := atomicBaseline(a)
				m := a.margin(innerW)
				above = math.Max(above, base+m.t)
				below = math.Max(below, a.h-base+m.b)
				continue
			}
			if it.trail && len(items) > 0 {
				// 걸린 공백도 줄 높이에 든다 (같은 계산값이면 영향이 없다).
			}
			m := e.inlineMetrics(it.st)
			above = math.Max(above, m.above)
			below = math.Max(below, m.below)
		}
		lh := above + below
		// 줄 끝 공백을 뺀 폭으로 정렬한다.
		used := 0.0
		for _, it := range items {
			if !it.trail || hasContentAfter(items, it) {
				used = math.Max(used, it.x+it.w)
			}
		}
		start := 0.0
		if li == 0 {
			start = 0 // indent는 x에 이미 들어 있다
		}
		offset := start
		switch align {
		case "center":
			offset += (innerW - used) / 2
		case "end", "right":
			offset += innerW - used
		}
		if wraps(whiteSpace(b.st)) || align == "start" || align == "" || align == "left" {
			// 넘친 줄은 시작 쪽에 붙는다 (CSS: 넘침은 끝 방향으로).
			if used > innerW && (align == "center" || align == "end" || align == "right") {
				offset = 0
			}
		}
		lb := lineBox{x: fr.l, y: y, w: innerW, h: lh, baseline: y + above}
		for _, it := range items {
			f := frag{x: fr.l + offset + it.x, w: it.w, baseline: lb.baseline, st: it.st, atomic: it.atomic}
			if it.atomic != nil {
				a := it.atomic
				m := a.margin(innerW)
				a.x = f.x + m.l
				a.y = lb.baseline - atomicBaseline(a)
			} else {
				f.glyphs = it.shaped.glyphs
				fm := e.fonts.faceMetrics(e.fonts.primary(fontStyleOf(it.st)))
				f.ascent = fm.ascent * it.st.fontSize
				f.descent = fm.descent * it.st.fontSize
				f.space = isBlank(p.runes[it.start:it.end])
				if !(it.trail && !hasContentAfter(items, it)) {
					r := rect{f.x, lb.baseline - f.ascent, f.w, f.ascent + f.descent}
					lb.textRect = union(lb.textRect, r, lb.hasText)
					lb.hasText = true
				}
			}
			lb.frags = append(lb.frags, f)
		}
		b.lines = append(b.lines, lb)
		y += lh
	}
	return y - fr.t
}

func isBlank(rs []rune) bool {
	for _, r := range rs {
		if r != ' ' {
			return false
		}
	}
	return true
}

func hasContentAfter(items []placed, it placed) bool {
	seen := false
	for _, o := range items {
		if o.start == it.start && o.end == it.end {
			seen = true
			continue
		}
		if seen && !o.trail {
			return true
		}
	}
	return false
}

// inlineMetrics는 계산값 하나의 인라인 box가 기준선 위아래로 차지하는 높이다 (half-leading 포함).
type lineMetrics struct{ above, below float64 }

func (e *engine) inlineMetrics(st *style) lineMetrics {
	fs := fontStyleOf(st)
	m := e.fonts.faceMetrics(e.fonts.primary(fs))
	a, d := m.ascent*fs.size, m.descent*fs.size
	lh := lineHeight(st, a+d)
	half := (lh - (a + d)) / 2
	return lineMetrics{above: a + half, below: d + half}
}

// lineHeight는 줄 높이(px)다. normal은 글꼴 내용 영역이다.
func lineHeight(st *style, content float64) float64 {
	v := st.get("line-height")
	if v == "" || v == "normal" {
		return content
	}
	if f, ok := parseNumber(v); ok && !strings.ContainsAny(v, "px%") {
		return f * st.fontSize
	}
	if l, ok := parseLength(v, st.fontSize); ok && !l.auto {
		return l.resolve(st.fontSize)
	}
	return content
}

// atomicBaseline은 원자 인라인의 기준선(위에서부터)이다: 마지막 줄의 기준선, 없으면 아래 margin 끝.
func atomicBaseline(a *box) float64 {
	if a.replaced == "" {
		if n := len(a.lines); n > 0 {
			return a.lines[n-1].baseline
		}
	}
	return a.h
}
