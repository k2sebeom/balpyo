// Package route는 연결선의 경로 계산이다 — 잰 두 박스와 양 끝의 면, 경로 종류에서 **웹의 SVG
// 경로와 PPTX의 연결선 프레임을 함께** 낸다.
//
// 이 계산이 한 곳에 있어야 하는 이유: 웹과 PPTX가 같은 선을 그린다는 계약은 같은 끝점 · 같은
// 꺾는 자리에서 나온다. 두 렌더러가 따로 계산하면 반올림 하나로 선이 갈라지고, 그 차이는 두
// 산출물을 나란히 볼 때까지 조용하다. 그래서 웹의 SVG 경로도 **PPTX 프리셋 정의를 같은 박스에서
// 평가한 것**이다 — 프리셋(`bentConnector3` 등)의 국소 경로를 뒤집기 · 회전으로 옮긴 점들이다.
//
// 이 패키지는 다른 balpyo 패키지를 임포트하지 않는다: render/web은 render/geom을 임포트할 수 없고
// (geom이 web을 부른다), 두 렌더러가 함께 부를 자리는 둘 다 모르는 순수 함수뿐이다.
package route

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// Rect는 논리 스테이지 px의 박스다 — 지오메트리 패스가 잰 rect의 복사다.
type Rect struct{ X, Y, W, H float64 }

// Point는 논리 스테이지 px의 점이다.
type Point struct{ X, Y float64 }

// Side는 연결선이 붙는 면이다. 빈 값은 자동이다.
type Side string

const (
	Auto   Side = ""
	Top    Side = "top"
	Right  Side = "right"
	Bottom Side = "bottom"
	Left   Side = "left"
)

// horizontal은 이 면에서 나가는 선이 가로로 나가는지다 (왼쪽 · 오른쪽 면).
func (s Side) horizontal() bool { return s == Left || s == Right }

// Kind는 경로 종류다.
type Kind string

const (
	Straight Kind = "straight"
	Elbow    Kind = "elbow"
	Curved   Kind = "curved"
)

// Path는 계산된 연결선 하나다.
type Path struct {
	// Start · End는 끝점이고 StartSide · EndSide는 확정된 면이다 (자동이 풀린 값).
	Start, End         Point
	StartSide, EndSide Side
	// Preset은 PPTX `prstGeom@prst`다.
	Preset string
	// Off · W · H는 PPTX `xfrm`의 **회전 전** 박스다 (Rot이 90이면 W · H가 화면의 세로 · 가로다).
	Off  Point
	W, H float64
	// Rot은 시계 방향 회전(도)이다 — 0 또는 90.
	Rot          int
	FlipH, FlipV bool
	// Points는 경로를 그리는 점들이다 — 첫 점은 MoveTo, 그다음은 Segments가 소비한다.
	Points []Point
	// Segments는 경로 조각이다: 'L'은 점 하나, 'C'는 점 셋(제어점 둘 + 끝점)을 먹는다.
	Segments []byte
}

// SVG는 경로의 SVG `d` 값이다 (절대 좌표, 논리 스테이지 px — 소수 둘째 자리에서 자른다).
func (p Path) SVG() string {
	var b strings.Builder
	b.WriteString("M" + pt(p.Points[0]))
	i := 1
	for _, seg := range p.Segments {
		switch seg {
		case 'L':
			b.WriteString(" L" + pt(p.Points[i]))
			i++
		case 'C':
			b.WriteString(" C" + pt(p.Points[i]) + " " + pt(p.Points[i+1]) + " " + pt(p.Points[i+2]))
			i += 3
		}
	}
	return b.String()
}

// Midpoint는 두 끝점의 가운데다 — 직선과 꺾임 둘 경로(가운데에서 꺾는다)에서는 경로 위의 점이고,
// 꺾임 하나 경로에서는 모서리 안쪽이다 (선 라벨이 앉는 자리).
func (p Path) Midpoint() Point {
	return Point{(p.Start.X + p.End.X) / 2, (p.Start.Y + p.End.Y) / 2}
}

func pt(p Point) string { return num(p.X) + "," + num(p.Y) }

func num(v float64) string {
	v = math.Round(v*100) / 100
	if v == 0 {
		v = 0 // -0을 0으로 — 결정성.
	}
	return strconv.FormatFloat(v, 'f', -1, 64)
}

// Anchor는 박스의 면 가운데 점이다 — 연결점이 그 자리다.
func Anchor(r Rect, s Side) Point {
	switch s {
	case Top:
		return Point{r.X + r.W/2, r.Y}
	case Bottom:
		return Point{r.X + r.W/2, r.Y + r.H}
	case Left:
		return Point{r.X, r.Y + r.H/2}
	default:
		return Point{r.X + r.W, r.Y + r.H/2}
	}
}

// facing은 from 박스에서 to 박스를 향하는 면이다: 두 중심의 가로 차가 세로 차 이상이면 가로 면,
// 아니면 세로 면이다.
func facing(from, to Rect) Side {
	dx := (to.X + to.W/2) - (from.X + from.W/2)
	dy := (to.Y + to.H/2) - (from.Y + from.H/2)
	if math.Abs(dx) >= math.Abs(dy) {
		if dx >= 0 {
			return Right
		}
		return Left
	}
	if dy >= 0 {
		return Bottom
	}
	return Top
}

// Route는 from 박스에서 to 박스로 가는 연결선을 계산한다. 면이 Auto이면 상대 박스를 향하는 면이다.
func Route(from, to Rect, fromSide, toSide Side, kind Kind) (Path, error) {
	for _, s := range []Side{fromSide, toSide} {
		switch s {
		case Auto, Top, Right, Bottom, Left:
		default:
			return Path{}, fmt.Errorf("unknown connector side %q", string(s))
		}
	}
	if fromSide == Auto {
		fromSide = facing(from, to)
	}
	if toSide == Auto {
		toSide = facing(to, from)
	}
	p := Path{StartSide: fromSide, EndSide: toSide}
	p.Start, p.End = Anchor(from, fromSide), Anchor(to, toSide)
	dx, dy := p.End.X-p.Start.X, p.End.Y-p.Start.Y

	// 세로로 시작하는 꺾인 경로는 같은 프리셋을 90° 돌린 것이다. 직선은 돌리지 않는다.
	rotated := kind != Straight && !fromSide.horizontal()
	sameAxis := fromSide.horizontal() == toSide.horizontal()
	switch kind {
	case Straight:
		p.Preset = "straightConnector1"
	case Elbow:
		p.Preset = "bentConnector2"
		if sameAxis {
			p.Preset = "bentConnector3"
		}
	case Curved:
		p.Preset = "curvedConnector2"
		if sameAxis {
			p.Preset = "curvedConnector3"
		}
	default:
		return Path{}, fmt.Errorf("unknown connector route %q", string(kind))
	}

	if rotated {
		// 시계 방향 90° 회전은 국소 (u, v)를 화면 (−v, u)로 보낸다. 그래서 국소 가로(W)가 화면 세로
		// 거리이고, 국소 끝점 (±W, ±H)이 화면 (∓H, ±W) = (dx, dy)가 되도록 뒤집는다.
		p.Rot = 90
		p.W, p.H = math.Abs(dy), math.Abs(dx)
		p.FlipH, p.FlipV = dy < 0, dx > 0
		c := Point{(p.Start.X + p.End.X) / 2, (p.Start.Y + p.End.Y) / 2}
		p.Off = Point{c.X - p.W/2, c.Y - p.H/2}
	} else {
		p.W, p.H = math.Abs(dx), math.Abs(dy)
		p.FlipH, p.FlipV = dx < 0, dy < 0
		p.Off = Point{math.Min(p.Start.X, p.End.X), math.Min(p.Start.Y, p.End.Y)}
	}

	local, segs := presetPath(p.Preset, p.W, p.H)
	p.Segments = segs
	p.Points = make([]Point, len(local))
	for i, l := range local {
		p.Points[i] = p.toWorld(l)
	}
	return p, nil
}

// toWorld는 프리셋 국소 좌표(박스 안 (u, v))를 화면 좌표로 옮긴다 — 뒤집기 다음 회전이다
// (DrawingML의 xfrm이 그 순서다: 뒤집기는 도형 좌표계 안에서, 회전은 박스 중심을 축으로).
func (p Path) toWorld(l Point) Point {
	u, v := l.X, l.Y
	if p.FlipH {
		u = p.W - u
	}
	if p.FlipV {
		v = p.H - v
	}
	x, y := p.Off.X+u, p.Off.Y+v
	if p.Rot == 90 {
		cx, cy := p.Off.X+p.W/2, p.Off.Y+p.H/2
		rx, ry := x-cx, y-cy
		x, y = cx-ry, cy+rx
	}
	return Point{x, y}
}

// presetPath는 연결선 프리셋의 국소 경로다 (ECMA-376 presetShapeDefinitions — 조정값은 기본 50%).
func presetPath(preset string, w, h float64) ([]Point, []byte) {
	switch preset {
	case "bentConnector2":
		return []Point{{0, 0}, {w, 0}, {w, h}}, []byte("LL")
	case "bentConnector3":
		x1 := w / 2
		return []Point{{0, 0}, {x1, 0}, {x1, h}, {w, h}}, []byte("LLL")
	case "curvedConnector2":
		return []Point{{0, 0}, {w / 2, 0}, {w, h / 2}, {w, h}}, []byte("C")
	case "curvedConnector3":
		x1 := w / 2
		x2, x3 := x1/2, (w+x1)/2
		return []Point{{0, 0}, {x2, 0}, {x1, h / 4}, {x1, h / 2}, {x1, h * 3 / 4}, {x3, h}, {w, h}}, []byte("CC")
	default: // straightConnector1
		return []Point{{0, 0}, {w, h}}, []byte("L")
	}
}
