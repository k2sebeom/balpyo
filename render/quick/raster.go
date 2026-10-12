package quick

import (
	"image"
	"image/color"
	"image/draw"
	"math"

	"golang.org/x/image/vector"
)

// affine은 2D 아핀 변환이다: (x, y) → (a·x + c·y + e, b·x + d·y + f).
type affine struct{ a, b, c, d, e, f float64 }

var identity = affine{1, 0, 0, 1, 0, 0}

func (m affine) apply(x, y float64) (float64, float64) {
	return m.a*x + m.c*y + m.e, m.b*x + m.d*y + m.f
}

// mul은 m 다음에 n을 적용하는 것이 아니라, n을 먼저 적용하고 m을 적용하는 변환이다 (m·n).
func (m affine) mul(n affine) affine {
	return affine{
		a: m.a*n.a + m.c*n.b, b: m.b*n.a + m.d*n.b,
		c: m.a*n.c + m.c*n.d, d: m.b*n.c + m.d*n.d,
		e: m.a*n.e + m.c*n.f + m.e, f: m.b*n.e + m.d*n.f + m.f,
	}
}

func translate(x, y float64) affine { return affine{1, 0, 0, 1, x, y} }
func scale(x, y float64) affine     { return affine{x, 0, 0, y, 0, 0} }
func rotate(deg float64) affine {
	r := deg * math.Pi / 180
	c, s := math.Cos(r), math.Sin(r)
	return affine{c, s, -s, c, 0, 0}
}

func (m affine) invert() affine {
	det := m.a*m.d - m.b*m.c
	if det == 0 {
		return identity
	}
	return affine{
		a: m.d / det, b: -m.b / det, c: -m.c / det, d: m.a / det,
		e: (m.c*m.f - m.d*m.e) / det, f: (m.b*m.e - m.a*m.f) / det,
	}
}

// scaleFactor는 변환이 길이를 늘리는 평균 비율이다 (선 굵기 환산).
func (m affine) scaleFactor() float64 { return math.Sqrt(math.Abs(m.a*m.d - m.b*m.c)) }

// ── 경로 ──

type pathOp struct {
	op  byte // 'M' 'L' 'Q' 'C' 'Z'
	pts [6]float64
}

type path []pathOp

func (p *path) moveTo(x, y float64) { *p = append(*p, pathOp{op: 'M', pts: [6]float64{x, y}}) }
func (p *path) lineTo(x, y float64) { *p = append(*p, pathOp{op: 'L', pts: [6]float64{x, y}}) }
func (p *path) quadTo(x1, y1, x, y float64) {
	*p = append(*p, pathOp{op: 'Q', pts: [6]float64{x1, y1, x, y}})
}
func (p *path) cubeTo(x1, y1, x2, y2, x, y float64) {
	*p = append(*p, pathOp{op: 'C', pts: [6]float64{x1, y1, x2, y2, x, y}})
}
func (p *path) close() { *p = append(*p, pathOp{op: 'Z'}) }

func rectPath(x, y, w, h float64) path {
	var p path
	p.moveTo(x, y)
	p.lineTo(x+w, y)
	p.lineTo(x+w, y+h)
	p.lineTo(x, y+h)
	p.close()
	return p
}

// kappa는 원호 사분면을 3차 베지어로 근사하는 상수다.
const kappa = 0.5522847498

// roundRectPath는 네 모서리 반지름이 같은 둥근 사각형이다 (반지름은 짧은 변의 반으로 줄인다).
func roundRectPath(x, y, w, h, rx, ry float64) path {
	rx = math.Min(rx, w/2)
	ry = math.Min(ry, h/2)
	if rx <= 0 || ry <= 0 {
		return rectPath(x, y, w, h)
	}
	kx, ky := rx*kappa, ry*kappa
	var p path
	p.moveTo(x+rx, y)
	p.lineTo(x+w-rx, y)
	p.cubeTo(x+w-rx+kx, y, x+w, y+ry-ky, x+w, y+ry)
	p.lineTo(x+w, y+h-ry)
	p.cubeTo(x+w, y+h-ry+ky, x+w-rx+kx, y+h, x+w-rx, y+h)
	p.lineTo(x+rx, y+h)
	p.cubeTo(x+rx-kx, y+h, x, y+h-ry+ky, x, y+h-ry)
	p.lineTo(x, y+ry)
	p.cubeTo(x, y+ry-ky, x+rx-kx, y, x+rx, y)
	p.close()
	return p
}

func ellipsePath(cx, cy, rx, ry float64) path {
	kx, ky := rx*kappa, ry*kappa
	var p path
	p.moveTo(cx+rx, cy)
	p.cubeTo(cx+rx, cy+ky, cx+kx, cy+ry, cx, cy+ry)
	p.cubeTo(cx-kx, cy+ry, cx-rx, cy+ky, cx-rx, cy)
	p.cubeTo(cx-rx, cy-ky, cx-kx, cy-ry, cx, cy-ry)
	p.cubeTo(cx+kx, cy-ry, cx+rx, cy-ky, cx+rx, cy)
	p.close()
	return p
}

func (p path) transform(m affine) path {
	out := make(path, len(p))
	for i, op := range p {
		out[i] = op
		n := 0
		switch op.op {
		case 'M', 'L':
			n = 1
		case 'Q':
			n = 2
		case 'C':
			n = 3
		}
		for j := 0; j < n; j++ {
			out[i].pts[2*j], out[i].pts[2*j+1] = m.apply(op.pts[2*j], op.pts[2*j+1])
		}
	}
	return out
}

// flatten은 경로를 하위 경로별 꺾은선으로 편다 (장치 좌표에서 쓴다).
func (p path) flatten() (polys [][][2]float64, closed []bool) {
	var cur [][2]float64
	var cx, cy, sx, sy float64
	finish := func(isClosed bool) {
		if len(cur) > 1 {
			polys = append(polys, cur)
			closed = append(closed, isClosed)
		}
		cur = nil
	}
	steps := func(dist float64) int { return max(2, min(64, int(dist/2)+2)) }
	for _, op := range p {
		switch op.op {
		case 'M':
			finish(false)
			cx, cy = op.pts[0], op.pts[1]
			sx, sy = cx, cy
			cur = append(cur, [2]float64{cx, cy})
		case 'L':
			if len(cur) == 0 {
				cur = append(cur, [2]float64{cx, cy})
			}
			cx, cy = op.pts[0], op.pts[1]
			cur = append(cur, [2]float64{cx, cy})
		case 'Q':
			if len(cur) == 0 {
				cur = append(cur, [2]float64{cx, cy})
			}
			x1, y1, x, y := op.pts[0], op.pts[1], op.pts[2], op.pts[3]
			n := steps(math.Hypot(x1-cx, y1-cy) + math.Hypot(x-x1, y-y1))
			for i := 1; i <= n; i++ {
				t := float64(i) / float64(n)
				u := 1 - t
				cur = append(cur, [2]float64{u*u*cx + 2*u*t*x1 + t*t*x, u*u*cy + 2*u*t*y1 + t*t*y})
			}
			cx, cy = x, y
		case 'C':
			if len(cur) == 0 {
				cur = append(cur, [2]float64{cx, cy})
			}
			x1, y1, x2, y2, x, y := op.pts[0], op.pts[1], op.pts[2], op.pts[3], op.pts[4], op.pts[5]
			n := steps(math.Hypot(x1-cx, y1-cy) + math.Hypot(x2-x1, y2-y1) + math.Hypot(x-x2, y-y2))
			for i := 1; i <= n; i++ {
				t := float64(i) / float64(n)
				u := 1 - t
				cur = append(cur, [2]float64{
					u*u*u*cx + 3*u*u*t*x1 + 3*u*t*t*x2 + t*t*t*x,
					u*u*u*cy + 3*u*u*t*y1 + 3*u*t*t*y2 + t*t*t*y,
				})
			}
			cx, cy = x, y
		case 'Z':
			if len(cur) > 0 {
				cur = append(cur, [2]float64{sx, sy})
			}
			finish(true)
			cx, cy = sx, sy
		}
	}
	finish(false)
	return polys, closed
}

// ── 캔버스 ──

// canvas는 슬라이드 한 장의 그림판이다. 좌표는 논리 px이고 dpr배로 그린다.
type canvas struct {
	img *image.RGBA
	ctm affine
	z   vector.Rasterizer
	// dirty는 지금까지 칠한 장치 영역이다 — 레이어 합성 · 흐림을 그 영역에서만 한다.
	dirty image.Rectangle
}

func newCanvas(w, h int, dpr float64) *canvas {
	return &canvas{img: image.NewRGBA(image.Rect(0, 0, w, h)), ctm: scale(dpr, dpr)}
}

// paint는 칠할 거리다: 단색이거나 장치 좌표에서 색을 내는 이미지(그라디언트 · 무늬 · 그림).
type paint struct {
	solid   rgba
	src     image.Image
	opacity float64
}

func solidPaint(c rgba) paint { return paint{solid: c, opacity: 1} }

func (p paint) visible() bool {
	if p.src != nil {
		return p.opacity > 0
	}
	return p.solid.a*p.opacity > 0
}

func premul(c rgba, alpha float64) color.RGBA {
	a := clamp01(c.a * alpha)
	return color.RGBA{
		R: uint8(math.Round(clamp01(c.r) * a * 255)), G: uint8(math.Round(clamp01(c.g) * a * 255)),
		B: uint8(math.Round(clamp01(c.b) * a * 255)), A: uint8(math.Round(a * 255)),
	}
}

// polyBounds는 꺾은선 묶음을 덮는 장치 픽셀 사각형이다 (캔버스 밖은 잘라 낸다).
func (cv *canvas) polyBounds(polys [][][2]float64) (image.Rectangle, bool) {
	minX, minY := math.Inf(1), math.Inf(1)
	maxX, maxY := math.Inf(-1), math.Inf(-1)
	for _, poly := range polys {
		for _, pt := range poly {
			minX, minY = math.Min(minX, pt[0]), math.Min(minY, pt[1])
			maxX, maxY = math.Max(maxX, pt[0]), math.Max(maxY, pt[1])
		}
	}
	bounds := cv.img.Bounds()
	x0 := max(int(math.Floor(minX)), bounds.Min.X)
	y0 := max(int(math.Floor(minY)), bounds.Min.Y)
	x1 := min(int(math.Ceil(maxX))+1, bounds.Max.X)
	y1 := min(int(math.Ceil(maxY))+1, bounds.Max.Y)
	if x1 <= x0 || y1 <= y0 {
		return image.Rectangle{}, false
	}
	return image.Rect(x0, y0, x1, y1), true
}

// fillDevice는 장치 좌표의 꺾은선 묶음을 채운다 (nonzero에 가까운 누적 규칙).
func (cv *canvas) fillPolys(polys [][][2]float64, p paint) {
	if !p.visible() || len(polys) == 0 {
		return
	}
	r, ok := cv.polyBounds(polys)
	if !ok {
		return
	}
	x0, y0 := r.Min.X, r.Min.Y
	cv.z.Reset(r.Dx(), r.Dy())
	cv.z.DrawOp = draw.Over
	ox, oy := float32(x0), float32(y0)
	for _, poly := range polys {
		cv.z.MoveTo(float32(poly[0][0])-ox, float32(poly[0][1])-oy)
		for _, pt := range poly[1:] {
			cv.z.LineTo(float32(pt[0])-ox, float32(pt[1])-oy)
		}
		cv.z.ClosePath()
	}
	cv.dirty = cv.dirty.Union(r)
	if p.src != nil {
		src := p.src
		if p.opacity < 1 {
			src = &fadeSrc{src: src, a: clamp01(p.opacity)}
		}
		cv.z.Draw(cv.img, r, src, r.Min)
		return
	}
	cv.z.Draw(cv.img, r, image.NewUniform(premul(p.solid, p.opacity)), image.Point{})
}

// fill은 사용자 좌표 경로를 현재 변환으로 채운다.
func (cv *canvas) fill(p path, pt paint) {
	polys, _ := p.transform(cv.ctm).flatten()
	cv.fillPolys(polys, pt)
}

// fillEvenOdd는 fill과 같되 짝홀 규칙(SVG `fill-rule: evenodd`)으로 채운다 — 하위 경로가 짝수 번
// 겹친 곳은 감은 방향과 상관없이 구멍이 된다.
func (cv *canvas) fillEvenOdd(p path, pt paint) {
	polys, _ := p.transform(cv.ctm).flatten()
	if len(polys) < 2 {
		cv.fillPolys(polys, pt)
		return
	}
	cv.fillPolysEvenOdd(polys, pt)
}

// fillPolysEvenOdd는 짝홀 규칙으로 채운다. 래스터라이저는 감긴 수를 누적해 1에서 자르므로
// (nonzero에 가깝다) 하위 경로마다 따로 덮임을 구하고, 덮임을 배타적 논리합(a + b − 2ab)으로
// 합친 마스크로 칠한다 — 완전히 덮인 픽셀은 덮인 횟수의 홀짝과 같고, 가장자리만 근사다.
func (cv *canvas) fillPolysEvenOdd(polys [][][2]float64, p paint) {
	if !p.visible() {
		return
	}
	r, ok := cv.polyBounds(polys)
	if !ok {
		return
	}
	ox, oy := float32(r.Min.X), float32(r.Min.Y)
	mask := image.NewAlpha(image.Rect(0, 0, r.Dx(), r.Dy()))
	one := image.NewAlpha(mask.Rect)
	for _, poly := range polys {
		clear(one.Pix)
		cv.z.Reset(r.Dx(), r.Dy())
		cv.z.DrawOp = draw.Src
		cv.z.MoveTo(float32(poly[0][0])-ox, float32(poly[0][1])-oy)
		for _, pt := range poly[1:] {
			cv.z.LineTo(float32(pt[0])-ox, float32(pt[1])-oy)
		}
		cv.z.ClosePath()
		cv.z.Draw(one, one.Rect, image.Opaque, image.Point{})
		for i, b := range one.Pix {
			a := int(mask.Pix[i])
			mask.Pix[i] = uint8(a + int(b) - (2*a*int(b)+127)/255)
		}
	}
	cv.dirty = cv.dirty.Union(r)
	src := image.Image(image.NewUniform(premul(p.solid, p.opacity)))
	if p.src != nil {
		src = p.src
		if p.opacity < 1 {
			src = &fadeSrc{src: src, a: clamp01(p.opacity)}
		}
	}
	draw.DrawMask(cv.img, r, src, r.Min, mask, image.Point{}, draw.Over)
}

// strokeStyle은 선 모양이다. 굵기는 사용자 단위이고 nonScaling이면 논리 px다.
type strokeStyle struct {
	width      float64
	dash       []float64
	cap, join  string
	nonScaling bool
}

// stroke는 경로를 선으로 그린다: 장치 좌표에서 펼친 꺾은선의 각 마디를 사각형으로, 꺾이는 점을
// 이음 모양으로 만들어 한 번에 채운다 (같은 방향으로 감아 겹침이 지워지지 않게 한다).
func (cv *canvas) stroke(p path, pt paint, st strokeStyle) {
	if st.width <= 0 || !pt.visible() {
		return
	}
	dev := p.transform(cv.ctm)
	polys, closed := dev.flatten()
	w := st.width * cv.ctm.scaleFactor()
	if st.nonScaling {
		w = st.width * cv.ctm.scaleFactor()
	}
	half := w / 2
	dash := make([]float64, len(st.dash))
	for i, d := range st.dash {
		dash[i] = d * cv.ctm.scaleFactor()
	}
	var shapes [][][2]float64
	for i, poly := range polys {
		segs := [][][2]float64{poly}
		if len(dash) > 0 {
			segs = applyDash(poly, dash)
		}
		for _, s := range segs {
			shapes = append(shapes, strokePolyline(s, half, closed[i] && len(dash) == 0, st.cap, st.join)...)
		}
	}
	cv.fillPolys(shapes, pt)
}

func orient(poly [][2]float64) [][2]float64 {
	area := 0.0
	for i := range poly {
		j := (i + 1) % len(poly)
		area += poly[i][0]*poly[j][1] - poly[j][0]*poly[i][1]
	}
	if area < 0 {
		out := make([][2]float64, len(poly))
		for i := range poly {
			out[i] = poly[len(poly)-1-i]
		}
		return out
	}
	return poly
}

func circlePoly(x, y, r float64) [][2]float64 {
	n := max(8, min(48, int(r*2)))
	out := make([][2]float64, n)
	for i := 0; i < n; i++ {
		a := 2 * math.Pi * float64(i) / float64(n)
		out[i] = [2]float64{x + r*math.Cos(a), y + r*math.Sin(a)}
	}
	return out
}

func strokePolyline(pts [][2]float64, half float64, closed bool, capStyle, join string) [][][2]float64 {
	// 겹치는 점을 뺀다.
	clean := pts[:0:0]
	for _, p := range pts {
		if n := len(clean); n > 0 && math.Hypot(p[0]-clean[n-1][0], p[1]-clean[n-1][1]) < 1e-6 {
			continue
		}
		clean = append(clean, p)
	}
	var out [][][2]float64
	if len(clean) < 2 {
		if len(clean) == 1 && capStyle == "round" {
			out = append(out, orient(circlePoly(clean[0][0], clean[0][1], half)))
		}
		return out
	}
	for i := 0; i+1 < len(clean); i++ {
		a, b := clean[i], clean[i+1]
		dx, dy := b[0]-a[0], b[1]-a[1]
		l := math.Hypot(dx, dy)
		nx, ny := -dy/l*half, dx/l*half
		ax, ay, bx, by := a[0], a[1], b[0], b[1]
		if !closed && capStyle == "square" {
			ex, ey := dx/l*half, dy/l*half
			if i == 0 {
				ax, ay = ax-ex, ay-ey
			}
			if i+2 == len(clean) {
				bx, by = bx+ex, by+ey
			}
		}
		quad := [][2]float64{{ax + nx, ay + ny}, {bx + nx, by + ny}, {bx - nx, by - ny}, {ax - nx, ay - ny}}
		out = append(out, orient(quad))
	}
	// 이음: 안쪽은 사각형끼리 겹치고 바깥쪽 틈을 채운다.
	joinAt := func(prev, p, next [2]float64) {
		if join == "round" {
			out = append(out, orient(circlePoly(p[0], p[1], half)))
			return
		}
		d1x, d1y := p[0]-prev[0], p[1]-prev[1]
		d2x, d2y := next[0]-p[0], next[1]-p[1]
		l1, l2 := math.Hypot(d1x, d1y), math.Hypot(d2x, d2y)
		n1x, n1y := -d1y/l1*half, d1x/l1*half
		n2x, n2y := -d2y/l2*half, d2x/l2*half
		cross := d1x*d2y - d1y*d2x
		if cross == 0 {
			return
		}
		sign := 1.0
		if cross > 0 {
			sign = -1
		}
		o1 := [2]float64{p[0] + sign*n1x, p[1] + sign*n1y}
		o2 := [2]float64{p[0] + sign*n2x, p[1] + sign*n2y}
		bevel := [][2]float64{p, o1, o2}
		if join != "bevel" {
			// miter: 두 바깥 변의 교점 (miterlimit 4)
			cos := (d1x*d2x + d1y*d2y) / (l1 * l2)
			theta := math.Acos(math.Max(-1, math.Min(1, cos)))
			if miter := 1 / math.Sin((math.Pi-theta)/2); miter <= 4 {
				bx, by := (o1[0]+o2[0])/2-p[0], (o1[1]+o2[1])/2-p[1]
				bl := math.Hypot(bx, by)
				if bl > 0 {
					tip := [2]float64{p[0] + bx/bl*half*miter, p[1] + by/bl*half*miter}
					bevel = [][2]float64{p, o1, tip, o2}
				}
			}
		}
		out = append(out, orient(bevel))
	}
	for i := 1; i+1 < len(clean); i++ {
		joinAt(clean[i-1], clean[i], clean[i+1])
	}
	if closed && len(clean) > 2 {
		n := len(clean)
		first, last := clean[0], clean[n-1]
		if math.Hypot(first[0]-last[0], first[1]-last[1]) < 1e-6 {
			joinAt(clean[n-2], clean[0], clean[1])
		}
	} else if capStyle == "round" {
		out = append(out, orient(circlePoly(clean[0][0], clean[0][1], half)))
		out = append(out, orient(circlePoly(clean[len(clean)-1][0], clean[len(clean)-1][1], half)))
	}
	return out
}

// applyDash는 꺾은선을 대시 조각으로 자른다.
func applyDash(poly [][2]float64, dash []float64) [][][2]float64 {
	total := 0.0
	for _, d := range dash {
		total += d
	}
	if total <= 0 {
		return [][][2]float64{poly}
	}
	if len(dash)%2 == 1 {
		dash = append(dash, dash...)
	}
	var out [][][2]float64
	var cur [][2]float64
	idx, left := 0, dash[0]
	on := true
	for i := 0; i+1 < len(poly); i++ {
		a, b := poly[i], poly[i+1]
		segLen := math.Hypot(b[0]-a[0], b[1]-a[1])
		pos := 0.0
		for segLen-pos > 1e-9 {
			step := math.Min(left, segLen-pos)
			t0, t1 := pos/segLen, (pos+step)/segLen
			p0 := [2]float64{a[0] + (b[0]-a[0])*t0, a[1] + (b[1]-a[1])*t0}
			p1 := [2]float64{a[0] + (b[0]-a[0])*t1, a[1] + (b[1]-a[1])*t1}
			if on {
				if len(cur) == 0 {
					cur = append(cur, p0)
				}
				cur = append(cur, p1)
			}
			pos += step
			left -= step
			if left <= 1e-9 {
				if on && len(cur) > 1 {
					out = append(out, cur)
				}
				cur = nil
				on = !on
				idx = (idx + 1) % len(dash)
				left = dash[idx]
			}
		}
	}
	if on && len(cur) > 1 {
		out = append(out, cur)
	}
	return out
}

// ── 레이어 ──

// layer는 그룹 합성(opacity · drop-shadow)을 위한 별도 그림판이다.
func (cv *canvas) layer() *canvas {
	return &canvas{img: image.NewRGBA(cv.img.Bounds()), ctm: cv.ctm}
}

// composite는 레이어를 불투명도 alpha로 덮어 그린다.
func (cv *canvas) composite(l *canvas, alpha float64) {
	r := l.dirty.Intersect(cv.img.Bounds())
	if r.Empty() {
		return
	}
	cv.dirty = cv.dirty.Union(r)
	if alpha >= 1 {
		draw.Draw(cv.img, r, l.img, r.Min, draw.Over)
		return
	}
	mask := image.NewUniform(color.Alpha{A: uint8(math.Round(clamp01(alpha) * 255))})
	draw.DrawMask(cv.img, r, l.img, r.Min, mask, image.Point{}, draw.Over)
}

// dropShadow는 레이어의 알파를 흐려 색을 입힌 그림자를 dx, dy만큼 옮겨 그린다 (장치 px).
func (cv *canvas) dropShadow(l *canvas, dx, dy, blur float64, c rgba) {
	// 흐림이 퍼지는 만큼 넓힌 영역에서만 계산한다.
	pad := int(math.Ceil(blur*1.5)) + 2
	ox, oy := int(math.Round(dx)), int(math.Round(dy))
	src := l.dirty.Inset(-pad).Intersect(l.img.Bounds())
	if src.Empty() {
		return
	}
	w, h := src.Dx(), src.Dy()
	alpha := make([]float32, w*h)
	for y := 0; y < h; y++ {
		row := l.img.Pix[(src.Min.Y+y)*l.img.Stride+src.Min.X*4:]
		for x := 0; x < w; x++ {
			alpha[y*w+x] = float32(row[x*4+3]) / 255
		}
	}
	// 가우시안 근사: 상자 흐림 세 번 (sigma = blur / 2)
	sigma := blur / 2
	if sigma > 0.5 {
		r := int(math.Round(math.Sqrt(12*sigma*sigma/3+1)-1) / 2)
		r = max(r, 1)
		for i := 0; i < 3; i++ {
			alpha = boxBlur(alpha, w, h, r)
		}
	}
	dst := src.Add(image.Pt(ox, oy)).Intersect(cv.img.Bounds())
	if dst.Empty() {
		return
	}
	shadow := image.NewRGBA(dst)
	for y := dst.Min.Y; y < dst.Max.Y; y++ {
		sy := y - oy - src.Min.Y
		for x := dst.Min.X; x < dst.Max.X; x++ {
			sx := x - ox - src.Min.X
			a := float64(alpha[sy*w+sx])
			if a <= 0 {
				continue
			}
			px := premul(c, a)
			i := shadow.PixOffset(x, y)
			shadow.Pix[i], shadow.Pix[i+1], shadow.Pix[i+2], shadow.Pix[i+3] = px.R, px.G, px.B, px.A
		}
	}
	cv.dirty = cv.dirty.Union(dst)
	draw.Draw(cv.img, dst, shadow, dst.Min, draw.Over)
}

func boxBlur(src []float32, w, h, r int) []float32 {
	tmp := make([]float32, len(src))
	out := make([]float32, len(src))
	norm := float32(1) / float32(2*r+1)
	for y := 0; y < h; y++ {
		row := src[y*w : (y+1)*w]
		var acc float32
		for x := -r; x <= r; x++ {
			acc += row[clampInt(x, 0, w-1)]
		}
		for x := 0; x < w; x++ {
			tmp[y*w+x] = acc * norm
			acc += row[clampInt(x+r+1, 0, w-1)] - row[clampInt(x-r, 0, w-1)]
		}
	}
	for x := 0; x < w; x++ {
		var acc float32
		for y := -r; y <= r; y++ {
			acc += tmp[clampInt(y, 0, h-1)*w+x]
		}
		for y := 0; y < h; y++ {
			out[y*w+x] = acc * norm
			acc += tmp[clampInt(y+r+1, 0, h-1)*w+x] - tmp[clampInt(y-r, 0, h-1)*w+x]
		}
	}
	return out
}

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// ── 그라디언트 · 무늬 · 그림 원천 ──

type stop struct {
	off float64
	c   rgba
}

// gradientSrc는 장치 좌표에서 색을 내는 그라디언트다. inv는 장치 → 그라디언트 공간 변환이다.
type gradientSrc struct {
	inv    affine
	radial bool
	// 선형: (x1,y1) → (x2,y2) 위 사영. 원형: 중심 (x1,y1), 반지름 r.
	x1, y1, x2, y2, r float64
	stops             []stop
}

func (g *gradientSrc) ColorModel() color.Model { return color.RGBAModel }
func (g *gradientSrc) Bounds() image.Rectangle {
	return image.Rect(-1<<20, -1<<20, 1<<20, 1<<20)
}

func (g *gradientSrc) At(x, y int) color.Color {
	ux, uy := g.inv.apply(float64(x)+0.5, float64(y)+0.5)
	var t float64
	if g.radial {
		if g.r > 0 {
			t = math.Hypot(ux-g.x1, uy-g.y1) / g.r
		}
	} else {
		dx, dy := g.x2-g.x1, g.y2-g.y1
		if l2 := dx*dx + dy*dy; l2 > 0 {
			t = ((ux-g.x1)*dx + (uy-g.y1)*dy) / l2
		}
	}
	return premul(sampleStops(g.stops, t), 1)
}

func sampleStops(stops []stop, t float64) rgba {
	if len(stops) == 0 {
		return transparent
	}
	if t <= stops[0].off {
		return stops[0].c
	}
	for i := 1; i < len(stops); i++ {
		if t <= stops[i].off {
			a, b := stops[i-1], stops[i]
			span := b.off - a.off
			f := 0.0
			if span > 0 {
				f = (t - a.off) / span
			}
			// premultiplied 보간
			aa := a.c.a*(1-f) + b.c.a*f
			if aa == 0 {
				return transparent
			}
			mix := func(x, y float64) float64 { return (x*a.c.a*(1-f) + y*b.c.a*f) / aa }
			return rgba{mix(a.c.r, b.c.r), mix(a.c.g, b.c.g), mix(a.c.b, b.c.b), aa}
		}
	}
	return stops[len(stops)-1].c
}

// tileSrc는 무늬 타일을 장치 좌표에서 반복한다.
type tileSrc struct {
	tile *image.RGBA
	inv  affine // 장치 → 타일 사용자 좌표
	tw   float64
	th   float64
	tdpr float64
}

func (t *tileSrc) ColorModel() color.Model { return color.RGBAModel }
func (t *tileSrc) Bounds() image.Rectangle {
	return image.Rect(-1<<20, -1<<20, 1<<20, 1<<20)
}
func (t *tileSrc) At(x, y int) color.Color {
	ux, uy := t.inv.apply(float64(x)+0.5, float64(y)+0.5)
	ux = math.Mod(math.Mod(ux, t.tw)+t.tw, t.tw)
	uy = math.Mod(math.Mod(uy, t.th)+t.th, t.th)
	px, py := int(ux*t.tdpr), int(uy*t.tdpr)
	b := t.tile.Bounds()
	px, py = clampInt(px, 0, b.Dx()-1), clampInt(py, 0, b.Dy()-1)
	return t.tile.RGBAAt(px, py)
}

// imageSrc는 그림을 사용자 사각형에 늘려 놓는다 (쌍선형 표본).
type imageSrc struct {
	img  image.Image
	inv  affine // 장치 → 그림 픽셀 좌표
	w, h int
}

func (s *imageSrc) ColorModel() color.Model { return color.RGBAModel }
func (s *imageSrc) Bounds() image.Rectangle {
	return image.Rect(-1<<20, -1<<20, 1<<20, 1<<20)
}
func (s *imageSrc) At(x, y int) color.Color {
	u, v := s.inv.apply(float64(x)+0.5, float64(y)+0.5)
	u -= 0.5
	v -= 0.5
	x0, y0 := int(math.Floor(u)), int(math.Floor(v))
	fx, fy := u-float64(x0), v-float64(y0)
	b := s.img.Bounds()
	get := func(px, py int) [4]float64 {
		px = clampInt(px, 0, s.w-1) + b.Min.X
		py = clampInt(py, 0, s.h-1) + b.Min.Y
		r, g, bb, a := s.img.At(px, py).RGBA()
		return [4]float64{float64(r), float64(g), float64(bb), float64(a)}
	}
	c00, c10, c01, c11 := get(x0, y0), get(x0+1, y0), get(x0, y0+1), get(x0+1, y0+1)
	var out [4]float64
	for i := 0; i < 4; i++ {
		out[i] = (c00[i]*(1-fx)+c10[i]*fx)*(1-fy) + (c01[i]*(1-fx)+c11[i]*fx)*fy
	}
	return color.RGBA64{uint16(out[0]), uint16(out[1]), uint16(out[2]), uint16(out[3])}
}

// fadeSrc는 원천에 불투명도를 곱한다.
type fadeSrc struct {
	src image.Image
	a   float64
}

func (f *fadeSrc) ColorModel() color.Model { return color.RGBA64Model }
func (f *fadeSrc) Bounds() image.Rectangle { return f.src.Bounds() }
func (f *fadeSrc) At(x, y int) color.Color {
	r, g, b, a := f.src.At(x, y).RGBA()
	return color.RGBA64{uint16(float64(r) * f.a), uint16(float64(g) * f.a), uint16(float64(b) * f.a), uint16(float64(a) * f.a)}
}
