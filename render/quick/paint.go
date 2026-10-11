package quick

import (
	"image"
	"math"
	"sort"
	"strconv"
	"strings"

	"github.com/k2sebeom/balpyo/render/web"
)

// Render는 슬라이드 하나를 dpr배 해상도의 그림으로 그린다 (모든 스텝이 펼쳐진 상태).
func (d *Document) Render(s *Slide, dpr float64) *image.RGBA {
	return d.RenderStep(s, dpr, -1, false)
}

// Steps는 슬라이드의 단계적 등장(reveal) 스텝 수다 — 0이면 한 장면뿐이다.
func (s *Slide) Steps() int {
	most := 0
	for _, n := range s.node.find(func(n *node) bool { _, ok := n.attrs[web.RevealAttr]; return ok }) {
		if v, err := strconv.Atoi(n.attrs[web.RevealAttr]); err == nil && v > most {
			most = v
		}
	}
	return most
}

// RenderStep은 reveal 스텝 step까지 드러난 장면을 그린다 (-1이면 전부). 숨은 항목은 자리를
// 그대로 차지한다 — 웹 발표자 뷰처럼 visibility로 숨기므로 배치가 움직이지 않는다. outline이면
// 잰 박스의 외곽선과 요소 식별자를 덧그린다.
func (d *Document) RenderStep(s *Slide, dpr float64, step int, outline bool) *image.RGBA {
	if dpr <= 0 {
		dpr = 1
	}
	w := int(math.Round(s.box.w * dpr))
	h := int(math.Round(s.box.h * dpr))
	cv := newCanvas(w, h, dpr)
	d.e.fonts.mu.Lock()
	defer d.e.fonts.mu.Unlock()
	ids := map[string]*node{}
	collectIDs(d.root, ids)
	p := &painter{e: d.e, cv: cv, ids: ids, step: step}
	p.context(s.box)
	if outline {
		p.outline(s)
	}
	return cv.img
}

type painter struct {
	e    *engine
	cv   *canvas
	ids  map[string]*node
	step int
}

// hidden은 아직 드러나지 않은 reveal 항목인지 본다 (조상까지).
func (p *painter) hidden(b *box) bool {
	if p.step < 0 || b.n == nil {
		return false
	}
	for n := b.n; n != nil; n = n.parent {
		if v, ok := n.attrs[web.RevealAttr]; ok {
			if k, err := strconv.Atoi(v); err == nil && k > p.step {
				return true
			}
		}
	}
	return false
}

func (b *box) visible() bool {
	return b.st.get("visibility") != "hidden" && b.st.get("visibility") != "collapse"
}

// outline은 식별자를 가진 요소의 잰 박스를 깊이별 색으로 긋고 식별자를 붙인다.
func (p *painter) outline(s *Slide) {
	hues := []rgba{{0.86, 0.15, 0.15, 0.9}, {0.06, 0.4, 0.98, 0.9}, {0.05, 0.6, 0.3, 0.9}, {0.58, 0.2, 0.92, 0.9}, {0.85, 0.45, 0.02, 0.9}}
	label := fontStyle{families: `"Noto Sans Mono", monospace`, size: 10, weight: 400}
	for _, n := range s.node.find(func(n *node) bool { _, ok := n.attrs[web.IDAttr]; return ok }) {
		b := s.boxes[n]
		if b == nil {
			continue
		}
		id := n.attrs[web.IDAttr]
		c := hues[(strings.Count(id, ".")-1+len(hues))%len(hues)]
		p.cv.stroke(rectPath(b.ax+0.5, b.ay+0.5, math.Max(b.w-1, 0), math.Max(b.h-1, 0)), solidPaint(c), strokeStyle{width: 1})
		text := p.e.fonts.shape(label, id)
		fm := p.e.fonts.faceMetrics(p.e.fonts.primary(label))
		h := (fm.ascent + fm.descent) * label.size
		p.cv.fill(rectPath(b.ax, b.ay, text.width+4, h), solidPaint(c))
		p.cv.fill(p.glyphPath(text.glyphs, b.ax+2, b.ay+fm.ascent*label.size), solidPaint(rgba{1, 1, 1, 1}))
	}
}

// context는 CSS 2.1 부록 E의 간이판으로 box 하나와 그 자손을 그린다: 자기 배경 → 흐름 안
// block 자손의 배경 → 흐름 안 인라인 내용 → 배치된(positioned) 자손을 문서 순서로.
func (p *painter) context(b *box) {
	if p.hidden(b) {
		return
	}
	p.withEffects(b, func() {
		p.background(b)
		var positioned []*box
		p.blockPhase(b, &positioned, true)
		p.inlinePhase(b, true)
		// 배치된 자손은 문서 순서로 그린다 (z-index: auto).
		sort.SliceStable(positioned, func(i, j int) bool { return docOrder(positioned[i]) < docOrder(positioned[j]) })
		for _, k := range positioned {
			p.context(k)
		}
	})
}

// withEffects는 transform · opacity · filter를 box 하나에 씌운다.
func (p *painter) withEffects(b *box, draw func()) {
	saved := p.cv.ctm
	defer func() { p.cv.ctm = saved }()
	if m, ok := boxTransform(b); ok {
		p.cv.ctm = saved.mul(m)
	}
	opacity := numberOr(b.st.get("opacity"), 1)
	shadow, hasShadow := dropShadowOf(b)
	if opacity >= 1 && !hasShadow {
		draw()
		return
	}
	target := p.cv
	layer := target.layer()
	p.cv = layer
	draw()
	p.cv = target
	if hasShadow {
		dpr := target.ctm.scaleFactor()
		target.dropShadow(layer, shadow.dx*dpr, shadow.dy*dpr, shadow.blur*dpr, shadow.color)
	}
	target.composite(layer, opacity)
}

// boxTransform은 회전 · 뒤집기 · 비례를 box 가운데를 원점으로 적용하는 변환이다. 이동
// (translate)은 배치가 이미 좌표에 넣었다.
func boxTransform(b *box) (affine, bool) {
	v := b.st.get("transform")
	if v == "" || v == "none" {
		v = ""
	}
	m := identity
	any := false
	for _, fn := range splitValues(v) {
		name, args, ok := strings.Cut(fn, "(")
		if !ok {
			continue
		}
		args = strings.TrimSuffix(args, ")")
		parts := splitTop(args, ',')
		switch name {
		case "rotate":
			if deg, ok := parseAngle(parts[0]); ok && deg != 0 {
				m = m.mul(rotate(deg))
				any = true
			}
		case "scaleX":
			m = m.mul(scale(numberOr(parts[0], 1), 1))
			any = true
		case "scaleY":
			m = m.mul(scale(1, numberOr(parts[0], 1)))
			any = true
		case "scale":
			sx := numberOr(parts[0], 1)
			sy := sx
			if len(parts) > 1 {
				sy = numberOr(parts[1], sx)
			}
			if sx != 1 || sy != 1 {
				m = m.mul(scale(sx, sy))
				any = true
			}
		}
	}
	if r := b.st.get("rotate"); r != "" && r != "none" {
		if deg, ok := parseAngle(r); ok {
			m = rotate(deg).mul(m)
			any = true
		}
	}
	if sc := b.st.get("scale"); sc != "" && sc != "none" {
		parts := splitValues(sc)
		sx := numberOr(parts[0], 1)
		sy := sx
		if len(parts) > 1 {
			sy = numberOr(parts[1], sx)
		}
		m = scale(sx, sy).mul(m)
		any = true
	}
	if !any {
		return identity, false
	}
	cx, cy := b.ax+b.w/2, b.ay+b.h/2
	return translate(cx, cy).mul(m).mul(translate(-cx, -cy)), true
}

type shadowSpec struct {
	dx, dy, blur float64
	color        rgba
}

// dropShadowOf는 filter: drop-shadow(…)를 읽는다.
func dropShadowOf(b *box) (shadowSpec, bool) {
	v := b.st.get("filter")
	i := strings.Index(v, "drop-shadow(")
	if i < 0 {
		return shadowSpec{}, false
	}
	end := closingParen(v, i+len("drop-shadow"))
	parts := splitValues(v[i+len("drop-shadow(") : end])
	var lens []float64
	col := rgba{0, 0, 0, 1}
	for _, part := range parts {
		if l, ok := parseLength(part, b.fs()); ok && !l.auto {
			lens = append(lens, l.px)
			continue
		}
		if c, ok := parseColor(part, rgba{0, 0, 0, 1}); ok {
			col = c
		}
	}
	s := shadowSpec{color: col}
	if len(lens) > 0 {
		s.dx = lens[0]
	}
	if len(lens) > 1 {
		s.dy = lens[1]
	}
	if len(lens) > 2 {
		s.blur = lens[2]
	}
	return s, true
}

// blockPhase는 흐름 안 block 자손의 배경을 그리고, 배치된 자손을 모은다.
func (p *painter) blockPhase(b *box, positioned *[]*box, top bool) {
	for _, k := range b.kids {
		if k.positioned() || k.st.get("opacity") != "" || boxHasEffects(k) {
			*positioned = append(*positioned, k)
			continue
		}
		if p.hidden(k) {
			continue
		}
		p.background(k)
		p.blockPhase(k, positioned, false)
	}
	*positioned = append(*positioned, b.abs...)
}

func boxHasEffects(b *box) bool {
	if _, ok := boxTransform(b); ok {
		return true
	}
	_, ok := dropShadowOf(b)
	return ok
}

// inlinePhase는 흐름 안 인라인 내용(텍스트 · 대체 요소)을 그린다.
func (p *painter) inlinePhase(b *box, top bool) {
	if !top && (b.positioned() || b.st.get("opacity") != "" || boxHasEffects(b)) || p.hidden(b) {
		return
	}
	if b.replaced != "" && b.visible() {
		p.replaced(b)
	}
	if b.hasInline {
		p.text(b)
	}
	for _, k := range b.kids {
		p.inlinePhase(k, false)
	}
}

// ── 배경 · 테두리 ──

func (p *painter) radius(b *box) (float64, float64) {
	v := b.st.get("border-radius")
	if v == "" {
		v = b.st.get("border-top-left-radius")
	}
	if v == "" {
		return 0, 0
	}
	parts := splitValues(strings.Split(v, "/")[0])
	l := lengthOr(parts[0], b.fs(), length{})
	return l.resolve(b.w), l.resolve(b.h)
}

func (p *painter) background(b *box) {
	if !b.visible() || b.replaced != "" && b.replaced != "img" {
		if b.replaced == "svg" {
			return
		}
	}
	if !b.visible() {
		return
	}
	rx, ry := p.radius(b)
	shape := roundRectPath(b.ax, b.ay, b.w, b.h, rx, ry)
	color := p.currentColor(b)
	if bg, ok := parseColor(b.st.get("background-color"), color); ok && bg.a > 0 && b.st.get("background-clip") != "text" {
		p.cv.fill(shape, solidPaint(bg))
	}
	if img := b.st.get("background-image"); img != "" && img != "none" && b.st.get("background-clip") != "text" {
		layers := splitTop(img, ',')
		sizes := splitTop(b.st.get("background-size"), ',')
		// 맨 앞 층이 맨 위다 — 뒤에서부터 그린다.
		for i := len(layers) - 1; i >= 0; i-- {
			size := "auto"
			if i < len(sizes) {
				size = strings.TrimSpace(sizes[i])
			} else if len(sizes) > 0 {
				size = strings.TrimSpace(sizes[len(sizes)-1])
			}
			p.backgroundLayer(b, strings.TrimSpace(layers[i]), size, shape, color)
		}
	}
	p.borders(b, rx, ry, color)
}

// backgroundLayer는 배경 층 하나(그라디언트 · url 그림)를 그린다.
func (p *painter) backgroundLayer(b *box, layer, size string, shape path, current rgba) {
	if pt, ok := p.cssGradient(layer, rect{b.ax, b.ay, b.w, b.h}, current); ok {
		p.cv.fill(shape, pt)
		return
	}
	if strings.HasPrefix(layer, "url(") {
		src := strings.Trim(strings.TrimSpace(layer[4:len(layer)-1]), `"'`)
		img := p.e.image(src)
		if img == nil {
			return
		}
		fit := "none"
		switch size {
		case "cover":
			fit = "cover"
		case "contain":
			fit = "contain"
		case "100% 100%":
			fit = "fill"
		}
		saved := p.cv
		_ = saved
		p.e.drawImageClipped(p.cv, img, rect{b.ax, b.ay, b.w, b.h}, shape, fit, 1)
	}
}

// cssGradient는 linear-gradient() · radial-gradient()를 box 사각형 r 위의 칠로 바꾼다.
func (p *painter) cssGradient(v string, r rect, current rgba) (paint, bool) {
	radial := strings.HasPrefix(v, "radial-gradient(")
	if !strings.HasPrefix(v, "linear-gradient(") && !radial {
		return paint{}, false
	}
	open := strings.IndexByte(v, '(')
	args := splitTop(v[open+1:closingParen(v, open)], ',')
	angle := 180.0
	cx, cy := 0.5, 0.5
	circle, extent := false, "farthest-corner"
	first := strings.TrimSpace(args[0])
	if !radial {
		if a, ok := parseAngle(first); ok && strings.ContainsAny(first, "dgrt") {
			angle = a
			args = args[1:]
		} else if strings.HasPrefix(first, "to ") {
			switch strings.TrimSpace(first[3:]) {
			case "top":
				angle = 0
			case "right":
				angle = 90
			case "bottom":
				angle = 180
			case "left":
				angle = 270
			case "top right", "right top":
				angle = 45
			case "bottom right", "right bottom":
				angle = 135
			case "bottom left", "left bottom":
				angle = 225
			case "top left", "left top":
				angle = 315
			}
			args = args[1:]
		}
	} else if strings.Contains(first, "at ") || strings.Contains(first, "circle") || strings.Contains(first, "ellipse") ||
		strings.Contains(first, "side") || strings.Contains(first, "corner") {
		shapePart := first
		if before, at, ok := strings.Cut(first, "at "); ok {
			shapePart = before
			pos := splitValues(at)
			read := func(v string, horizontal bool) float64 {
				switch v {
				case "left", "top":
					return 0
				case "right", "bottom":
					return 1
				case "center":
					return 0.5
				}
				if l, ok := parseLength(v, 16); ok {
					base := r.h
					if horizontal {
						base = r.w
					}
					if base > 0 {
						return (l.px + l.pct*base/100) / base
					}
				}
				return 0.5
			}
			cx = read(pos[0], true)
			if len(pos) > 1 {
				cy = read(pos[1], false)
			}
		}
		circle = strings.Contains(shapePart, "circle")
		for _, k := range []string{"closest-side", "closest-corner", "farthest-side", "farthest-corner"} {
			if strings.Contains(shapePart, k) {
				extent = k
			}
		}
		args = args[1:]
	}
	var stops []stop
	for i, a := range args {
		parts := splitValues(strings.TrimSpace(a))
		off := -1.0
		colorText := strings.TrimSpace(a)
		if n := len(parts); n > 1 {
			if l, ok := parseLength(parts[n-1], 16); ok && !l.auto {
				off = l.pct / 100
				colorText = strings.Join(parts[:n-1], " ")
			}
		}
		c, ok := parseColor(colorText, current)
		if !ok {
			continue
		}
		if off < 0 {
			if len(args) > 1 {
				off = float64(i) / float64(len(args)-1)
			} else {
				off = 0
			}
		}
		stops = append(stops, stop{off: off, c: c})
	}
	if len(stops) == 0 {
		return paint{}, false
	}
	g := &gradientSrc{stops: stops, radial: radial}
	if radial {
		// 끝 모양(원 · 타원)과 크기 키워드로 반지름을 정하고, 단위 원 공간으로 바꿔 표본한다.
		px, py := cx*r.w, cy*r.h
		nearX, farX := math.Min(px, r.w-px), math.Max(px, r.w-px)
		nearY, farY := math.Min(py, r.h-py), math.Max(py, r.h-py)
		var rx, ry float64
		switch extent {
		case "closest-side":
			rx, ry = nearX, nearY
			if circle {
				rx = math.Min(nearX, nearY)
				ry = rx
			}
		case "farthest-side":
			rx, ry = farX, farY
			if circle {
				rx = math.Max(farX, farY)
				ry = rx
			}
		case "closest-corner":
			rx, ry = nearX*math.Sqrt2, nearY*math.Sqrt2
			if circle {
				rx = math.Hypot(nearX, nearY)
				ry = rx
			}
		default:
			rx, ry = farX*math.Sqrt2, farY*math.Sqrt2
			if circle {
				rx = math.Hypot(farX, farY)
				ry = rx
			}
		}
		rx, ry = math.Max(rx, 1e-6), math.Max(ry, 1e-6)
		g.r = 1
		g.inv = p.cv.ctm.mul(translate(r.x+px, r.y+py)).mul(scale(rx, ry)).invert()
		return paint{src: g, opacity: 1}, true
	} else {
		// CSS: 각도 방향으로 box 모서리가 0%와 100%에 닿는 선
		rad := angle * math.Pi / 180
		sx, sy := math.Sin(rad), -math.Cos(rad)
		half := (math.Abs(r.w*sx) + math.Abs(r.h*sy)) / 2
		mx, my := r.x+r.w/2, r.y+r.h/2
		g.x1, g.y1 = mx-sx*half, my-sy*half
		g.x2, g.y2 = mx+sx*half, my+sy*half
	}
	g.inv = p.cv.ctm.invert()
	return paint{src: g, opacity: 1}, true
}

func (p *painter) currentColor(b *box) rgba {
	c, ok := parseColor(b.st.get("color"), rgba{0, 0, 0, 1})
	if !ok {
		return rgba{0, 0, 0, 1}
	}
	return c
}

func (p *painter) borders(b *box, rx, ry float64, current rgba) {
	type side struct {
		w     float64
		style string
		color rgba
	}
	read := func(name string) side {
		st := b.st.get("border-" + name + "-style")
		if st == "" || st == "none" || st == "hidden" {
			return side{}
		}
		w := lengthOr(b.st.get("border-"+name+"-width"), b.fs(), length{px: 3}).px
		c, ok := parseColor(b.st.get("border-"+name+"-color"), current)
		if !ok {
			c = current
		}
		return side{w, st, c}
	}
	t, r, bt, l := read("top"), read("right"), read("bottom"), read("left")
	if t.w == 0 && r.w == 0 && bt.w == 0 && l.w == 0 {
		return
	}
	x, y, w, h := b.ax, b.ay, b.w, b.h
	if b.display == "table-cell" && b.parentTableCollapsed() {
		// 맞닿은 칸이 선 하나를 나눠 가지므로 선의 가운데가 box 경계에 온다.
		x, y, w, h = x-l.w/2, y-t.w/2, w+(l.w+r.w)/2, h+(t.w+bt.w)/2
	}
	if t == r && r == bt && bt == l {
		half := t.w / 2
		path := roundRectPath(x+half, y+half, w-t.w, h-t.w, math.Max(0, rx-half), math.Max(0, ry-half))
		ss := strokeStyle{width: t.w, join: "miter"}
		if t.style == "dashed" {
			ss.dash = []float64{t.w * 3, t.w * 3}
		} else if t.style == "dotted" {
			ss.dash = []float64{t.w, t.w}
		}
		p.cv.stroke(path, solidPaint(t.color), ss)
		return
	}
	if t.w > 0 {
		p.cv.fill(rectPath(x, y, w, t.w), solidPaint(t.color))
	}
	if bt.w > 0 {
		p.cv.fill(rectPath(x, y+h-bt.w, w, bt.w), solidPaint(bt.color))
	}
	if l.w > 0 {
		p.cv.fill(rectPath(x, y, l.w, h), solidPaint(l.color))
	}
	if r.w > 0 {
		p.cv.fill(rectPath(x+w-r.w, y, r.w, h), solidPaint(r.color))
	}
}

// ── 대체 요소 ──

func (p *painter) replaced(b *box) {
	switch b.replaced {
	case "svg":
		p.e.drawSVG(p.cv, b.n, b.ax, b.ay, b.w, b.h, p.ids)
	case "img":
		img := p.e.image(b.n.attrs["src"])
		f := b.frame(0)
		r := rect{b.ax + f.l, b.ay + f.t, b.w - f.h(), b.h - f.v()}
		if img == nil {
			// 읽지 못한 그림: 빗금 자리표시
			p.cv.fill(rectPath(r.x, r.y, r.w, r.h), solidPaint(rgba{0.85, 0.85, 0.85, 1}))
			return
		}
		fit := b.st.get("object-fit")
		if fit == "" {
			fit = "fill"
		}
		rx, ry := p.radius(b)
		clip := roundRectPath(r.x, r.y, r.w, r.h, rx, ry)
		p.e.drawImageClipped(p.cv, img, r, clip, fit, 1)
		p.borders(b, rx, ry, p.currentColor(b))
	}
}

// drawImageRect는 그림을 사각형에 object-fit으로 놓는다.
func (e *engine) drawImageRect(cv *canvas, img *decodedImage, x, y, w, h float64, fit string, opacity float64) {
	e.drawImageClipped(cv, img, rect{x, y, w, h}, rectPath(x, y, w, h), fit, opacity)
}

func (e *engine) drawImageClipped(cv *canvas, img *decodedImage, r rect, clip path, fit string, opacity float64) {
	if img.w <= 0 || img.h <= 0 || r.w <= 0 || r.h <= 0 {
		return
	}
	dw, dh := r.w, r.h
	switch fit {
	case "contain", "scale-down":
		s := math.Min(r.w/img.w, r.h/img.h)
		if fit == "scale-down" {
			s = math.Min(s, 1)
		}
		dw, dh = img.w*s, img.h*s
	case "cover":
		s := math.Max(r.w/img.w, r.h/img.h)
		dw, dh = img.w*s, img.h*s
	case "none":
		dw, dh = img.w, img.h
	}
	dx, dy := r.x+(r.w-dw)/2, r.y+(r.h-dh)/2
	if img.svg != nil {
		target := cv
		l := cv.layer()
		e.drawSVG(l, img.svg, dx, dy, dw, dh, nil)
		// 자르기: 레이어를 clip 경로 마스크로 덮는다.
		maskLayer := cv.layer()
		maskLayer.fill(clip, solidPaint(rgba{0, 0, 0, 1}))
		applyMask(l.img, maskLayer.img)
		l.dirty = l.dirty.Intersect(maskLayer.dirty)
		target.composite(l, opacity)
		return
	}
	m := cv.ctm.mul(translate(dx, dy)).mul(scale(dw/img.w, dh/img.h))
	b := img.img.Bounds()
	src := &imageSrc{img: img.img, inv: m.invert(), w: b.Dx(), h: b.Dy()}
	// 그림 사각형과 clip의 겹침을 칠한다.
	visible := intersectPath(clip, rect{dx, dy, dw, dh})
	cv.fill(visible, paint{src: src, opacity: opacity})
}

// intersectPath는 clip 경로를 사각형으로 자른다 (사각형이 clip을 덮으면 clip 그대로).
func intersectPath(clip path, r rect) path {
	bb := pathBounds(clip)
	if r.x <= bb.x+0.01 && r.y <= bb.y+0.01 && r.right() >= bb.right()-0.01 && r.bottom() >= bb.bottom()-0.01 {
		return clip
	}
	x0, y0 := math.Max(r.x, bb.x), math.Max(r.y, bb.y)
	x1, y1 := math.Min(r.right(), bb.right()), math.Min(r.bottom(), bb.bottom())
	if x1 <= x0 || y1 <= y0 {
		return nil
	}
	return rectPath(x0, y0, x1-x0, y1-y0)
}

func applyMask(dst, mask *image.RGBA) {
	for i := 3; i < len(dst.Pix); i += 4 {
		m := uint32(mask.Pix[i])
		if m == 255 {
			continue
		}
		for j := i - 3; j <= i; j++ {
			dst.Pix[j] = uint8(uint32(dst.Pix[j]) * m / 255)
		}
	}
}

// ── 텍스트 ──

func (p *painter) text(b *box) {
	if !b.visible() {
		return
	}
	// background-clip: text — 글자를 배경 그라디언트로 칠한다.
	var clipPaint *paint
	if clipsText(b.st) {
		if pt, ok := p.cssGradient(strings.TrimSpace(splitTop(b.st.get("background-image"), ',')[0]), rect{b.ax, b.ay, b.w, b.h}, p.currentColor(b)); ok {
			clipPaint = &pt
		}
	}
	// 인라인 요소(span)의 background-clip: text — 그라디언트 기준은 그 요소 조각들의 영역이다.
	spanRects := map[*style]rect{}
	for _, l := range b.lines {
		for _, f := range l.frags {
			if f.atomic == nil && f.st != b.st && clipsText(f.st) {
				r := rect{b.ax + f.x, b.ay + f.baseline - f.ascent, f.w, f.ascent + f.descent}
				prev, ok := spanRects[f.st]
				spanRects[f.st] = union(prev, r, ok)
			}
		}
	}
	for _, l := range b.lines {
		for _, f := range l.frags {
			if f.atomic != nil {
				p.context(f.atomic)
				continue
			}
			st := f.st
			if st.get("visibility") == "hidden" {
				continue
			}
			color := rgba{0, 0, 0, 1}
			if c, ok := parseColor(st.get("color"), color); ok {
				color = c
			}
			x0, base := b.ax+f.x, b.ay+f.baseline
			// 인라인 배경 (mark · span) — 계산값이 블록 자신의 것이 아닐 때만
			if st != b.st && !clipsText(st) {
				if bg, ok := parseColor(st.get("background-color"), color); ok && bg.a > 0 {
					p.cv.fill(rectPath(x0, base-f.ascent, f.w, f.ascent+f.descent), solidPaint(bg))
				}
			}
			if len(f.glyphs) > 0 && !f.space {
				pt := solidPaint(color)
				if clipPaint != nil {
					pt = *clipPaint
				}
				if r, ok := spanRects[st]; ok {
					layer := strings.TrimSpace(splitTop(st.get("background-image"), ',')[0])
					if gp, ok := p.cssGradient(layer, r, color); ok {
						pt = gp
					}
				}
				p.cv.fill(p.glyphPath(f.glyphs, x0, base), pt)
			}
			if strings.Contains(st.get("text-decoration"), "underline") || strings.Contains(st.get("text-decoration-line"), "underline") {
				fm := p.e.fonts.faceMetrics(p.e.fonts.primary(fontStyleOf(st)))
				thick := math.Max(1, fm.underlineThick*st.fontSize)
				y := base - fm.underlinePos*st.fontSize
				p.cv.fill(rectPath(x0, y, f.w, thick), solidPaint(color))
			}
		}
	}
}

func (p *painter) glyphPath(gs []glyph, x0, base float64) path {
	var out path
	for _, g := range gs {
		o, ok := p.e.fonts.glyphOutline(g.face, g.gid)
		if !ok || len(o.Segments) == 0 {
			continue
		}
		s := g.size / float64(g.face.Upem())
		ox, oy := x0+g.x, base+g.y
		shear := 0.0
		if g.slant {
			shear = 0.25 // Skia의 가짜 기울임과 같은 기울기
		}
		pt := func(x, y float32) (float64, float64) {
			return ox + (float64(x)+shear*float64(y))*s, oy - float64(y)*s
		}
		for _, seg := range o.Segments {
			a := seg.Args
			switch seg.Op {
			case 0: // MoveTo
				x, y := pt(a[0].X, a[0].Y)
				out.moveTo(x, y)
			case 1: // LineTo
				x, y := pt(a[0].X, a[0].Y)
				out.lineTo(x, y)
			case 2: // QuadTo
				x1, y1 := pt(a[0].X, a[0].Y)
				x, y := pt(a[1].X, a[1].Y)
				out.quadTo(x1, y1, x, y)
			case 3: // CubeTo
				x1, y1 := pt(a[0].X, a[0].Y)
				x2, y2 := pt(a[1].X, a[1].Y)
				x, y := pt(a[2].X, a[2].Y)
				out.cubeTo(x1, y1, x2, y2, x, y)
			}
		}
		out.close()
	}
	return out
}

func docOrder(b *box) int {
	if b.n == nil {
		return 0
	}
	return b.n.order
}

func clipsText(st *style) bool {
	return st.get("background-clip") == "text" || st.get("-webkit-background-clip") == "text"
}
