package quick

import (
	"image"
	"math"
	"strings"
)

// parsePathData는 SVG 경로 데이터(d)를 읽는다. 호(A)는 3차 베지어로 바꾼다.
func parsePathData(d string) path {
	var p path
	toks := pathTokens(d)
	i := 0
	var cmd byte
	var cx, cy, sx, sy float64
	var lastCtrlX, lastCtrlY float64
	var lastCmd byte
	num := func() float64 {
		if i >= len(toks) {
			return 0
		}
		f, _ := parseNumber(toks[i])
		i++
		return f
	}
	isNum := func() bool {
		if i >= len(toks) {
			return false
		}
		c := toks[i][0]
		return c == '-' || c == '+' || c == '.' || (c >= '0' && c <= '9')
	}
	for i < len(toks) {
		if !isNum() {
			cmd = toks[i][0]
			i++
			if cmd == 'Z' || cmd == 'z' {
				p.close()
				cx, cy = sx, sy
				lastCmd = 'Z'
				continue
			}
		}
		if !isNum() {
			continue
		}
		rel := cmd >= 'a' && cmd <= 'z'
		ox, oy := 0.0, 0.0
		if rel {
			ox, oy = cx, cy
		}
		switch cmd {
		case 'M', 'm':
			x, y := num()+ox, num()+oy
			p.moveTo(x, y)
			cx, cy, sx, sy = x, y, x, y
			if cmd == 'M' {
				cmd = 'L'
			} else {
				cmd = 'l'
			}
		case 'L', 'l':
			x, y := num()+ox, num()+oy
			p.lineTo(x, y)
			cx, cy = x, y
		case 'H', 'h':
			x := num() + ox
			p.lineTo(x, cy)
			cx = x
		case 'V', 'v':
			y := num() + oy
			p.lineTo(cx, y)
			cy = y
		case 'C', 'c':
			x1, y1 := num()+ox, num()+oy
			x2, y2 := num()+ox, num()+oy
			x, y := num()+ox, num()+oy
			p.cubeTo(x1, y1, x2, y2, x, y)
			lastCtrlX, lastCtrlY = x2, y2
			cx, cy = x, y
		case 'S', 's':
			x1, y1 := cx, cy
			if lastCmd == 'C' || lastCmd == 'S' {
				x1, y1 = 2*cx-lastCtrlX, 2*cy-lastCtrlY
			}
			x2, y2 := num()+ox, num()+oy
			x, y := num()+ox, num()+oy
			p.cubeTo(x1, y1, x2, y2, x, y)
			lastCtrlX, lastCtrlY = x2, y2
			cx, cy = x, y
		case 'Q', 'q':
			x1, y1 := num()+ox, num()+oy
			x, y := num()+ox, num()+oy
			p.quadTo(x1, y1, x, y)
			lastCtrlX, lastCtrlY = x1, y1
			cx, cy = x, y
		case 'T', 't':
			x1, y1 := cx, cy
			if lastCmd == 'Q' || lastCmd == 'T' {
				x1, y1 = 2*cx-lastCtrlX, 2*cy-lastCtrlY
			}
			x, y := num()+ox, num()+oy
			p.quadTo(x1, y1, x, y)
			lastCtrlX, lastCtrlY = x1, y1
			cx, cy = x, y
		case 'A', 'a':
			rx, ry, rot := num(), num(), num()
			large, sweep := num() != 0, num() != 0
			x, y := num()+ox, num()+oy
			arcTo(&p, cx, cy, rx, ry, rot, large, sweep, x, y)
			cx, cy = x, y
		default:
			i++
		}
		lastCmd = cmd &^ 0x20
	}
	return p
}

func pathTokens(d string) []string {
	var out []string
	i := 0
	for i < len(d) {
		c := d[i]
		switch {
		case c == ' ' || c == ',' || c == '\n' || c == '\t' || c == '\r':
			i++
		case (c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z') && c != 'e' && c != 'E':
			out = append(out, string(c))
			i++
		default:
			j := i
			if d[j] == '-' || d[j] == '+' {
				j++
			}
			dot := false
			for j < len(d) {
				ch := d[j]
				if ch >= '0' && ch <= '9' {
					j++
				} else if ch == '.' && !dot {
					dot = true
					j++
				} else if (ch == 'e' || ch == 'E') && j+1 < len(d) {
					j++
					if d[j] == '-' || d[j] == '+' {
						j++
					}
				} else {
					break
				}
			}
			if j == i {
				j++
			}
			out = append(out, d[i:j])
			i = j
		}
	}
	return out
}

// arcTo는 SVG 호를 3차 베지어 조각으로 근사한다 (SVG 1.1 부록 F.6).
func arcTo(p *path, x1, y1, rx, ry, phiDeg float64, large, sweep bool, x2, y2 float64) {
	if rx == 0 || ry == 0 {
		p.lineTo(x2, y2)
		return
	}
	rx, ry = math.Abs(rx), math.Abs(ry)
	phi := phiDeg * math.Pi / 180
	cosP, sinP := math.Cos(phi), math.Sin(phi)
	dx, dy := (x1-x2)/2, (y1-y2)/2
	x1p := cosP*dx + sinP*dy
	y1p := -sinP*dx + cosP*dy
	lambda := x1p*x1p/(rx*rx) + y1p*y1p/(ry*ry)
	if lambda > 1 {
		s := math.Sqrt(lambda)
		rx, ry = rx*s, ry*s
	}
	num := rx*rx*ry*ry - rx*rx*y1p*y1p - ry*ry*x1p*x1p
	den := rx*rx*y1p*y1p + ry*ry*x1p*x1p
	coef := 0.0
	if den != 0 && num > 0 {
		coef = math.Sqrt(num / den)
	}
	if large == sweep {
		coef = -coef
	}
	cxp, cyp := coef*rx*y1p/ry, -coef*ry*x1p/rx
	cx := cosP*cxp - sinP*cyp + (x1+x2)/2
	cy := sinP*cxp + cosP*cyp + (y1+y2)/2
	angle := func(ux, uy, vx, vy float64) float64 {
		a := math.Atan2(ux*vy-uy*vx, ux*vx+uy*vy)
		return a
	}
	th1 := angle(1, 0, (x1p-cxp)/rx, (y1p-cyp)/ry)
	dth := angle((x1p-cxp)/rx, (y1p-cyp)/ry, (-x1p-cxp)/rx, (-y1p-cyp)/ry)
	if !sweep && dth > 0 {
		dth -= 2 * math.Pi
	} else if sweep && dth < 0 {
		dth += 2 * math.Pi
	}
	n := int(math.Ceil(math.Abs(dth) / (math.Pi / 2)))
	delta := dth / float64(n)
	t := 4.0 / 3 * math.Tan(delta/4)
	for i := 0; i < n; i++ {
		a1 := th1 + float64(i)*delta
		a2 := a1 + delta
		c1, s1 := math.Cos(a1), math.Sin(a1)
		c2, s2 := math.Cos(a2), math.Sin(a2)
		pt := func(x, y float64) (float64, float64) {
			return cosP*rx*x - sinP*ry*y + cx, sinP*rx*x + cosP*ry*y + cy
		}
		ax, ay := pt(c1-t*s1, s1+t*c1)
		bx, by := pt(c2+t*s2, s2-t*c2)
		ex, ey := pt(c2, s2)
		p.cubeTo(ax, ay, bx, by, ex, ey)
	}
}

// parseTransform은 SVG transform 속성을 읽는다.
func parseTransform(v string) affine {
	m := identity
	for v = strings.TrimSpace(v); v != ""; v = strings.TrimSpace(v) {
		open := strings.IndexByte(v, '(')
		close := strings.IndexByte(v, ')')
		if open < 0 || close < open {
			break
		}
		name := strings.TrimSpace(strings.TrimLeft(v[:open], ", "))
		var args []float64
		for _, a := range strings.FieldsFunc(v[open+1:close], func(r rune) bool { return r == ',' || r == ' ' }) {
			f, _ := parseNumber(a)
			args = append(args, f)
		}
		arg := func(i int, def float64) float64 {
			if i < len(args) {
				return args[i]
			}
			return def
		}
		var t affine
		switch name {
		case "matrix":
			t = affine{arg(0, 1), arg(1, 0), arg(2, 0), arg(3, 1), arg(4, 0), arg(5, 0)}
		case "translate":
			t = translate(arg(0, 0), arg(1, 0))
		case "scale":
			t = scale(arg(0, 1), arg(1, arg(0, 1)))
		case "rotate":
			t = rotate(arg(0, 0))
			if len(args) == 3 {
				t = translate(args[1], args[2]).mul(t).mul(translate(-args[1], -args[2]))
			}
		case "skewX":
			t = affine{1, 0, math.Tan(arg(0, 0) * math.Pi / 180), 1, 0, 0}
		case "skewY":
			t = affine{1, math.Tan(arg(0, 0) * math.Pi / 180), 0, 1, 0, 0}
		default:
			t = identity
		}
		m = m.mul(t)
		v = v[close+1:]
	}
	return m
}

// ── SVG 그리기 ──

// svgContext는 SVG 문서 하나를 그리는 동안의 상태다.
type svgContext struct {
	e      *engine
	cv     *canvas
	ids    map[string]*node
	vw, vh float64 // 퍼센트 기준 (viewBox 또는 viewport)
	color  rgba
}

func collectIDs(n *node, out map[string]*node) {
	if n.isText() {
		return
	}
	if id, ok := n.attrs["id"]; ok {
		if _, seen := out[id]; !seen {
			out[id] = n
		}
	}
	for _, c := range n.children {
		collectIDs(c, out)
	}
}

// drawSVG는 svg 요소를 사용자 좌표의 viewport (x, y, w, h)에 그린다.
func (e *engine) drawSVG(cv *canvas, svg *node, x, y, w, h float64, docIDs map[string]*node) {
	if w <= 0 || h <= 0 {
		return
	}
	ids := map[string]*node{}
	collectIDs(svg, ids)
	for k, v := range docIDs {
		if _, ok := ids[k]; !ok {
			ids[k] = v
		}
	}
	saved := cv.ctm
	defer func() { cv.ctm = saved }()
	vw, vh := w, h
	m := translate(x, y)
	if vb := parseViewBox(svg.attrs["viewBox"]); vb != nil && vb[2] > 0 && vb[3] > 0 {
		vw, vh = vb[2], vb[3]
		sx, sy := w/vb[2], h/vb[3]
		par := strings.Fields(svg.attrs["preserveAspectRatio"])
		align, slice := "xMidYMid", false
		if len(par) > 0 {
			align = par[0]
		}
		if len(par) > 1 && par[1] == "slice" {
			slice = true
		}
		tx, ty := 0.0, 0.0
		if align != "none" {
			s := math.Min(sx, sy)
			if slice {
				s = math.Max(sx, sy)
			}
			sx, sy = s, s
			if strings.Contains(align, "xMid") {
				tx = (w - vb[2]*s) / 2
			} else if strings.Contains(align, "xMax") {
				tx = w - vb[2]*s
			}
			if strings.Contains(align, "YMid") {
				ty = (h - vb[3]*s) / 2
			} else if strings.Contains(align, "YMax") {
				ty = h - vb[3]*s
			}
		}
		m = m.mul(translate(tx, ty)).mul(scale(sx, sy)).mul(translate(-vb[0], -vb[1]))
	}
	cv.ctm = saved.mul(m)
	cur, _ := parseColor(svg.style.get("color"), rgba{0, 0, 0, 1})
	ctx := &svgContext{e: e, cv: cv, ids: ids, vw: vw, vh: vh, color: cur}
	for _, c := range svg.children {
		ctx.draw(c)
	}
}

func (s *svgContext) num(v string, base float64) float64 {
	v = strings.TrimSpace(v)
	if strings.HasSuffix(v, "%") {
		f, _ := parseNumber(strings.TrimSuffix(v, "%"))
		return f * base / 100
	}
	if f, ok := parseNumber(v); ok {
		return f
	}
	return lengthOr(v, 16, length{}).px
}

func (s *svgContext) diag() float64 { return math.Sqrt(s.vw*s.vw+s.vh*s.vh) / math.Sqrt2 }

// shapePath는 기본 도형 요소의 경로다.
func (s *svgContext) shapePath(n *node) (path, bool) {
	a := n.attrs
	switch n.tag {
	case "rect":
		x, y := s.num(a["x"], s.vw), s.num(a["y"], s.vh)
		w, h := s.num(a["width"], s.vw), s.num(a["height"], s.vh)
		if w <= 0 || h <= 0 {
			return nil, false
		}
		_, hasRX := a["rx"]
		_, hasRY := a["ry"]
		rx, ry := s.num(a["rx"], s.vw), s.num(a["ry"], s.vh)
		if hasRX && !hasRY {
			ry = rx
		} else if hasRY && !hasRX {
			rx = ry
		}
		return roundRectPath(x, y, w, h, rx, ry), true
	case "circle":
		r := s.num(a["r"], s.diag())
		return ellipsePath(s.num(a["cx"], s.vw), s.num(a["cy"], s.vh), r, r), r > 0
	case "ellipse":
		rx, ry := s.num(a["rx"], s.vw), s.num(a["ry"], s.vh)
		return ellipsePath(s.num(a["cx"], s.vw), s.num(a["cy"], s.vh), rx, ry), rx > 0 && ry > 0
	case "line":
		var p path
		p.moveTo(s.num(a["x1"], s.vw), s.num(a["y1"], s.vh))
		p.lineTo(s.num(a["x2"], s.vw), s.num(a["y2"], s.vh))
		return p, true
	case "polyline", "polygon":
		var p path
		pts := strings.FieldsFunc(a["points"], func(r rune) bool { return r == ',' || r == ' ' || r == '\n' || r == '\t' })
		for i := 0; i+1 < len(pts); i += 2 {
			x, _ := parseNumber(pts[i])
			y, _ := parseNumber(pts[i+1])
			if i == 0 {
				p.moveTo(x, y)
			} else {
				p.lineTo(x, y)
			}
		}
		if n.tag == "polygon" {
			p.close()
		}
		return p, len(p) > 1
	case "path":
		p := parsePathData(a["d"])
		return p, len(p) > 0
	}
	return nil, false
}

func pathBounds(p path) rect {
	minX, minY := math.Inf(1), math.Inf(1)
	maxX, maxY := math.Inf(-1), math.Inf(-1)
	polys, _ := p.flatten()
	for _, poly := range polys {
		for _, pt := range poly {
			minX, minY = math.Min(minX, pt[0]), math.Min(minY, pt[1])
			maxX, maxY = math.Max(maxX, pt[0]), math.Max(maxY, pt[1])
		}
	}
	if math.IsInf(minX, 1) {
		return rect{}
	}
	return rect{minX, minY, maxX - minX, maxY - minY}
}

func (s *svgContext) draw(n *node) {
	if n.isText() || n.style == nil {
		return
	}
	if n.style.get("display") == "none" || n.style.get("visibility") == "hidden" {
		switch n.tag {
		case "g", "svg", "use":
		default:
			if n.style.get("display") == "none" {
				return
			}
		}
	}
	switch n.tag {
	case "defs", "linearGradient", "radialGradient", "pattern", "marker", "clipPath", "mask", "style", "title", "desc", "metadata", "symbol":
		return
	}
	saved := s.cv.ctm
	defer func() { s.cv.ctm = saved }()
	if t, ok := n.attrs["transform"]; ok {
		s.cv.ctm = saved.mul(parseTransform(t))
	}
	opacity := numberOr(n.style.get("opacity"), 1)
	target := s.cv
	if opacity < 1 {
		s.cv = target.layer()
		defer func() {
			l := s.cv
			s.cv = target
			target.composite(l, opacity)
		}()
	}
	switch n.tag {
	case "g", "a":
		for _, c := range n.children {
			s.draw(c)
		}
		return
	case "svg":
		w, h := s.num(n.attrs["width"], s.vw), s.num(n.attrs["height"], s.vh)
		if _, ok := n.attrs["width"]; !ok {
			w = s.vw
		}
		if _, ok := n.attrs["height"]; !ok {
			h = s.vh
		}
		s.e.drawSVG(s.cv, n, s.num(n.attrs["x"], s.vw), s.num(n.attrs["y"], s.vh), w, h, s.ids)
		return
	case "use":
		href := strings.TrimPrefix(n.attrs["href"], "#")
		if ref, ok := s.ids[href]; ok && ref != n {
			s.cv.ctm = s.cv.ctm.mul(translate(s.num(n.attrs["x"], s.vw), s.num(n.attrs["y"], s.vh)))
			if ref.tag == "symbol" {
				for _, c := range ref.children {
					s.draw(c)
				}
			} else {
				s.draw(ref)
			}
		}
		return
	case "image":
		s.drawImage(n)
		return
	}
	p, ok := s.shapePath(n)
	if !ok {
		return
	}
	st := n.style
	bbox := pathBounds(p)
	if fill := st.get("fill"); fill != "none" {
		if fill == "" {
			fill = "black"
		}
		if pt, ok := s.paintOf(fill, bbox, numberOr(st.get("fill-opacity"), 1)); ok {
			if st.get("fill-rule") == "evenodd" {
				s.cv.fillEvenOdd(p, pt)
			} else {
				s.cv.fill(p, pt)
			}
		}
	}
	if stroke := st.get("stroke"); stroke != "" && stroke != "none" {
		if pt, ok := s.paintOf(stroke, bbox, numberOr(st.get("stroke-opacity"), 1)); ok {
			sw := s.num(st.get("stroke-width"), s.diag())
			if st.get("stroke-width") == "" {
				sw = 1
			}
			ss := strokeStyle{
				width: sw, cap: st.get("stroke-linecap"), join: st.get("stroke-linejoin"),
				nonScaling: n.attrs["vector-effect"] == "non-scaling-stroke" || st.get("vector-effect") == "non-scaling-stroke",
			}
			if ss.nonScaling {
				// 화면 기준 굵기: 변환의 배율을 걷어낸다 (장치 배율 dpr만 남긴다).
				ss.width = sw * s.devicePerPx() / math.Max(s.cv.ctm.scaleFactor(), 1e-9)
			}
			if d := st.get("stroke-dasharray"); d != "" && d != "none" {
				for _, part := range strings.FieldsFunc(d, func(r rune) bool { return r == ',' || r == ' ' }) {
					ss.dash = append(ss.dash, s.num(part, s.diag()))
				}
				if ss.nonScaling {
					for i := range ss.dash {
						ss.dash[i] *= s.devicePerPx() / math.Max(s.cv.ctm.scaleFactor(), 1e-9)
					}
				}
			}
			s.cv.stroke(p, pt, ss)
		}
	}
	s.drawMarkers(n, p)
}

// devicePerPx는 논리 px 하나가 장치 픽셀 몇 개인지다 (캔버스의 기본 배율).
func (s *svgContext) devicePerPx() float64 {
	return float64(s.cv.img.Bounds().Dx()) / math.Max(s.e.stageW, 1)
}

// paintOf는 fill · stroke 값을 칠할 거리로 바꾼다.
func (s *svgContext) paintOf(v string, bbox rect, opacity float64) (paint, bool) {
	v = strings.TrimSpace(v)
	if strings.HasPrefix(v, "url(") {
		end := strings.IndexByte(v, ')')
		id := strings.Trim(strings.TrimSpace(v[4:end]), `"'#`)
		ref, ok := s.ids[id]
		if !ok {
			// 대체 색 (url(#x) red)
			if rest := strings.TrimSpace(v[end+1:]); rest != "" {
				return s.paintOf(rest, bbox, opacity)
			}
			return paint{}, false
		}
		switch ref.tag {
		case "linearGradient", "radialGradient":
			g := s.gradient(ref, bbox)
			return paint{src: g, opacity: opacity}, g != nil
		case "pattern":
			t := s.pattern(ref)
			return paint{src: t, opacity: opacity}, t != nil
		}
		return paint{}, false
	}
	c, ok := parseColor(v, s.color)
	if !ok {
		return paint{}, false
	}
	return paint{solid: c, opacity: opacity}, true
}

// gradientChain은 href로 이어진 그라디언트에서 속성과 stop을 찾는다.
func (s *svgContext) gradientAttr(n *node, name string) (string, bool) {
	for i := 0; n != nil && i < 8; i++ {
		if v, ok := n.attrs[name]; ok {
			return v, true
		}
		href := strings.TrimPrefix(n.attrs["href"], "#")
		n = s.ids[href]
	}
	return "", false
}

func (s *svgContext) gradientStops(n *node) []stop {
	for i := 0; n != nil && i < 8; i++ {
		var stops []stop
		for _, c := range n.children {
			if c.tag != "stop" {
				continue
			}
			off := s.num(c.attrs["offset"], 1)
			if strings.HasSuffix(strings.TrimSpace(c.attrs["offset"]), "%") {
				off = s.num(c.attrs["offset"], 1)
			}
			col, ok := parseColor(c.style.get("stop-color"), s.color)
			if !ok {
				col = rgba{0, 0, 0, 1}
			}
			col.a *= numberOr(c.style.get("stop-opacity"), 1)
			if len(stops) > 0 && off < stops[len(stops)-1].off {
				off = stops[len(stops)-1].off
			}
			stops = append(stops, stop{off: clamp01(off), c: col})
		}
		if len(stops) > 0 {
			return stops
		}
		n = s.ids[strings.TrimPrefix(n.attrs["href"], "#")]
	}
	return nil
}

func (s *svgContext) gradient(n *node, bbox rect) *gradientSrc {
	stops := s.gradientStops(n)
	if len(stops) == 0 {
		return nil
	}
	units, _ := s.gradientAttr(n, "gradientUnits")
	user := units == "userSpaceOnUse"
	get := func(name string, def float64, base float64) float64 {
		v, ok := s.gradientAttr(n, name)
		if !ok {
			return def
		}
		if user {
			return s.num(v, base)
		}
		return s.num(v, 1)
	}
	g := &gradientSrc{stops: stops, radial: n.tag == "radialGradient"}
	space := identity
	if !user {
		space = translate(bbox.x, bbox.y).mul(scale(bbox.w, bbox.h))
	}
	if t, ok := s.gradientAttr(n, "gradientTransform"); ok {
		space = space.mul(parseTransform(t))
	}
	if g.radial {
		defC := 0.5
		if user {
			defC = 0
		}
		g.x1, g.y1 = get("cx", defC, s.vw), get("cy", defC, s.vh)
		g.r = get("r", 0.5, s.diag())
	} else {
		g.x1, g.y1 = get("x1", 0, s.vw), get("y1", 0, s.vh)
		g.x2, g.y2 = get("x2", 1, s.vw), get("y2", 0, s.vh)
		if user {
			if _, ok := s.gradientAttr(n, "x2"); !ok {
				g.x2 = s.vw
			}
		}
	}
	g.inv = s.cv.ctm.mul(space).invert()
	return g
}

// pattern은 무늬 타일을 장치 배율로 그려 반복 원천으로 만든다 (userSpaceOnUse만 쓴다).
func (s *svgContext) pattern(n *node) *tileSrc {
	tw, th := s.num(n.attrs["width"], s.vw), s.num(n.attrs["height"], s.vh)
	if tw <= 0 || th <= 0 {
		return nil
	}
	space := s.cv.ctm
	if t, ok := n.attrs["patternTransform"]; ok {
		space = space.mul(parseTransform(t))
	}
	dpr := math.Max(space.scaleFactor(), 1)
	w, h := max(1, int(math.Ceil(tw*dpr))), max(1, int(math.Ceil(th*dpr)))
	tile := &canvas{img: image.NewRGBA(image.Rect(0, 0, w, h)), ctm: scale(dpr, dpr)}
	sub := &svgContext{e: s.e, cv: tile, ids: s.ids, vw: tw, vh: th, color: s.color}
	for _, c := range n.children {
		sub.draw(c)
	}
	return &tileSrc{tile: tile.img, inv: space.invert(), tw: tw, th: th, tdpr: dpr}
}

func (s *svgContext) drawImage(n *node) {
	href := n.attrs["href"]
	img := s.e.image(href)
	if img == nil {
		return
	}
	x, y := s.num(n.attrs["x"], s.vw), s.num(n.attrs["y"], s.vh)
	w, h := s.num(n.attrs["width"], s.vw), s.num(n.attrs["height"], s.vh)
	s.e.drawImageRect(s.cv, img, x, y, w, h, "contain", 1)
}

// drawMarkers는 marker-start · marker-end를 경로 끝점에 그린다 (orient auto).
func (s *svgContext) drawMarkers(n *node, p path) {
	sw := s.num(n.style.get("stroke-width"), s.diag())
	if n.style.get("stroke-width") == "" {
		sw = 1
	}
	for _, which := range []string{"marker-start", "marker-end"} {
		v := n.attrs[which]
		if v == "" {
			v = n.style.get(which)
		}
		if !strings.HasPrefix(v, "url(") {
			continue
		}
		id := strings.Trim(strings.TrimSpace(v[4:strings.IndexByte(v, ')')]), `"'#`)
		m, ok := s.ids[id]
		if !ok {
			continue
		}
		polys, _ := p.flatten()
		if len(polys) == 0 {
			continue
		}
		var px, py, angle float64
		if which == "marker-start" {
			pts := polys[0]
			px, py = pts[0][0], pts[0][1]
			angle = math.Atan2(pts[1][1]-pts[0][1], pts[1][0]-pts[0][0]) * 180 / math.Pi
		} else {
			pts := polys[len(polys)-1]
			k := len(pts) - 1
			px, py = pts[k][0], pts[k][1]
			angle = math.Atan2(pts[k][1]-pts[k-1][1], pts[k][0]-pts[k-1][0]) * 180 / math.Pi
		}
		orient := m.attrs["orient"]
		switch {
		case orient == "auto":
		case orient == "auto-start-reverse":
			if which == "marker-start" {
				angle += 180
			}
		default:
			angle = numberOr(orient, 0)
		}
		mw, mh := s.num(m.attrs["markerWidth"], s.vw), s.num(m.attrs["markerHeight"], s.vh)
		if _, ok := m.attrs["markerWidth"]; !ok {
			mw = 3
		}
		if _, ok := m.attrs["markerHeight"]; !ok {
			mh = 3
		}
		unit := 1.0
		if m.attrs["markerUnits"] != "userSpaceOnUse" {
			unit = sw
		}
		saved := s.cv.ctm
		mt := translate(px, py).mul(rotate(angle)).mul(scale(unit, unit))
		refX, refY := s.num(m.attrs["refX"], mw), s.num(m.attrs["refY"], mh)
		if vb := parseViewBox(m.attrs["viewBox"]); vb != nil && vb[2] > 0 && vb[3] > 0 {
			sx, sy := mw/vb[2], mh/vb[3]
			sc := math.Min(sx, sy)
			mt = mt.mul(scale(sc, sc)).mul(translate(-refX, -refY))
		} else {
			mt = mt.mul(translate(-refX, -refY))
		}
		s.cv.ctm = saved.mul(mt)
		for _, c := range m.children {
			s.draw(c)
		}
		s.cv.ctm = saved
	}
}
