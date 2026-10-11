package route

import (
	"fmt"
	"math"
	"testing"
)

func near(a, b Point) bool { return math.Abs(a.X-b.X) < 1e-9 && math.Abs(a.Y-b.Y) < 1e-9 }

// 모든 면 · 경로 조합에서 프리셋 국소 경로를 xfrm(뒤집기 → 회전)으로 옮긴 첫 점과 끝 점이 두 끝점과
// 같다 — PPTX가 그릴 선과 웹이 그릴 선이 같은 출처라는 계약의 판정이다.
func TestRouteEndpointsMatchTheFrame(t *testing.T) {
	a := Rect{X: 100, Y: 100, W: 200, H: 80}
	others := []Rect{
		{X: 600, Y: 400, W: 160, H: 80}, // 오른쪽 아래
		{X: 600, Y: 20, W: 160, H: 60},  // 오른쪽 위
		{X: 0, Y: 500, W: 90, H: 90},    // 왼쪽 아래
		{X: 120, Y: 400, W: 200, H: 80}, // 바로 아래
	}
	sides := []Side{Auto, Top, Right, Bottom, Left}
	for _, b := range others {
		for _, kind := range []Kind{Straight, Elbow, Curved} {
			for _, fs := range sides {
				for _, ts := range sides {
					if fs == ts {
						continue
					}
					p, err := Route(a, b, fs, ts, kind)
					if err != nil {
						t.Fatal(err)
					}
					if !near(p.Points[0], p.Start) || !near(p.Points[len(p.Points)-1], p.End) {
						t.Errorf("%v %s %q→%q: 경로 %v…%v, 끝점 %v→%v", b, kind, fs, ts,
							p.Points[0], p.Points[len(p.Points)-1], p.Start, p.End)
					}
				}
			}
		}
	}
}

// 자동 면은 상대 박스를 향한다: 가로 차가 크면 마주 보는 가로 면, 아니면 세로 면.
func TestRouteAutoSidesFaceEachOther(t *testing.T) {
	a := Rect{X: 0, Y: 0, W: 100, H: 100}
	for _, tc := range []struct {
		b          Rect
		from, to   Side
		startPoint Point
	}{
		{Rect{X: 400, Y: 50, W: 100, H: 100}, Right, Left, Point{100, 50}},
		{Rect{X: 20, Y: 400, W: 100, H: 100}, Bottom, Top, Point{50, 100}},
		{Rect{X: -400, Y: 0, W: 100, H: 100}, Left, Right, Point{0, 50}},
		{Rect{X: 0, Y: -300, W: 100, H: 100}, Top, Bottom, Point{50, 0}},
	} {
		p, err := Route(a, tc.b, Auto, Auto, Straight)
		if err != nil {
			t.Fatal(err)
		}
		if p.StartSide != tc.from || p.EndSide != tc.to || !near(p.Start, tc.startPoint) {
			t.Errorf("%v: %s→%s @%v, want %s→%s @%v", tc.b, p.StartSide, p.EndSide, p.Start, tc.from, tc.to, tc.startPoint)
		}
	}
}

// 꺾임 경로는 축이 같은 두 면이면 꺾임 둘(가운데에서 꺾는다), 다르면 꺾임 하나다. 세로로 시작하면
// 같은 프리셋을 90° 돌린다 — 그때 xfrm의 W · H는 화면의 세로 · 가로 거리다.
func TestRouteElbowPresets(t *testing.T) {
	a := Rect{X: 0, Y: 0, W: 100, H: 100}
	b := Rect{X: 300, Y: 300, W: 100, H: 100}
	for _, tc := range []struct {
		fs, ts Side
		preset string
		rot    int
		svg    string
	}{
		{Right, Left, "bentConnector3", 0, "M100,50 L200,50 L200,350 L300,350"},
		{Bottom, Top, "bentConnector3", 90, "M50,100 L50,200 L350,200 L350,300"},
		{Right, Top, "bentConnector2", 0, "M100,50 L350,50 L350,300"},
		{Bottom, Left, "bentConnector2", 90, "M50,100 L50,350 L300,350"},
	} {
		p, err := Route(a, b, tc.fs, tc.ts, Elbow)
		if err != nil {
			t.Fatal(err)
		}
		if p.Preset != tc.preset || p.Rot != tc.rot || p.SVG() != tc.svg {
			t.Errorf("%s→%s: %s rot %d %q, want %s rot %d %q", tc.fs, tc.ts, p.Preset, p.Rot, p.SVG(), tc.preset, tc.rot, tc.svg)
		}
	}
	p, _ := Route(a, b, Bottom, Top, Elbow)
	if p.W != 200 || p.H != 300 || p.Off != (Point{100, 50}) || p.FlipH || !p.FlipV {
		t.Errorf("회전한 프레임 = off %v ext %gx%g flipH %v flipV %v", p.Off, p.W, p.H, p.FlipH, p.FlipV)
	}
}

// 직선은 돌리지 않고 뒤집기만 한다 — 왼쪽 위로 가는 선은 두 축을 다 뒤집는다.
func TestRouteStraightFlips(t *testing.T) {
	p, err := Route(Rect{X: 300, Y: 300, W: 100, H: 100}, Rect{X: 0, Y: 0, W: 100, H: 100}, Left, Bottom, Straight)
	if err != nil {
		t.Fatal(err)
	}
	if p.Rot != 0 || !p.FlipH || !p.FlipV || p.Off != (Point{50, 100}) || p.W != 250 || p.H != 250 {
		t.Errorf("프레임 = %+v", p)
	}
	if got := p.SVG(); got != "M300,350 L50,100" {
		t.Errorf("SVG = %q", got)
	}
}

func TestRouteRejectsUnknownValues(t *testing.T) {
	if _, err := Route(Rect{}, Rect{}, "north", Auto, Straight); err == nil {
		t.Error("모르는 면이 통과했다")
	}
	if _, err := Route(Rect{}, Rect{}, Auto, Auto, "zigzag"); err == nil {
		t.Error("모르는 경로가 통과했다")
	}
}

// 같은 면끼리 잇는 U자 경로는 두 상자 바깥에서 꺾는다 — 위 → 위는 더 높은 윗변보다 위로 나간다.
// 같은 높이(국소 가로 길이 0)에서도 1px 깊이의 프레임으로 비율을 적을 수 있다.
func TestRouteUTurn(t *testing.T) {
	a := Rect{X: 0, Y: 100, W: 100, H: 60}
	b := Rect{X: 400, Y: 100, W: 100, H: 60}
	for _, kind := range []Kind{Elbow, Curved} {
		p, err := Route(a, b, Top, Top, kind)
		if err != nil {
			t.Fatal(err)
		}
		if p.Adj[0] >= 0 && p.Adj[0] <= 1 {
			t.Errorf("%s: adj %g는 프레임 안이다 — U자는 바깥에서 꺾어야 한다", kind, p.Adj[0])
		}
		first, last := p.Points[0], p.Points[len(p.Points)-1]
		if math.Hypot(first.X-p.Start.X, first.Y-p.Start.Y) > 1.01 || math.Hypot(last.X-p.End.X, last.Y-p.End.Y) > 1.01 {
			t.Errorf("%s: 경로 끝 %v…%v가 끝점 %v→%v에서 1px보다 멀다", kind, first, last, p.Start, p.End)
		}
		top := math.Inf(1)
		for _, pt := range p.Points {
			top = math.Min(top, pt.Y)
		}
		if want := 100 - uTurnGap(kind, 400); math.Abs(top-want) > 1e-6 {
			t.Errorf("%s: 가장 높은 점 y = %g, want %g", kind, top, want)
		}
	}
	p, _ := Route(a, Rect{X: 400, Y: 300, W: 100, H: 60}, Right, Right, Elbow)
	if got, want := p.Points[1].X, 500+uTurnGap(Elbow, 0); math.Abs(got-want) > 1e-6 {
		t.Errorf("오른쪽 → 오른쪽: 꺾는 x = %g, want %g", got, want)
	}
}

// 라벨은 경로 길이의 가운데에 앉는다 — 가로 선이면 위, 세로 선이면 오른쪽이다.
func TestRouteLabelRect(t *testing.T) {
	p, _ := Route(Rect{X: 0, Y: 0, W: 100, H: 100}, Rect{X: 400, Y: 0, W: 100, H: 100}, Auto, Auto, Straight)
	if got := p.LabelRect(80, 20); got != (Rect{X: 210, Y: 50 - LabelGap - 20, W: 80, H: 20}) {
		t.Errorf("가로 선의 라벨 = %+v", got)
	}
	p, _ = Route(Rect{X: 0, Y: 0, W: 100, H: 100}, Rect{X: 0, Y: 400, W: 100, H: 100}, Auto, Auto, Straight)
	if got := p.LabelRect(80, 20); got != (Rect{X: 50 + LabelGap, Y: 240, W: 80, H: 20}) {
		t.Errorf("세로 선의 라벨 = %+v", got)
	}
}

// sample은 경로를 촘촘한 점으로 편다 — 곡선은 flatten, 직선 조각은 1px 간격으로 자른다.
func sample(p Path) []Point {
	pts := p.flatten()
	out := []Point{pts[0]}
	for k := 1; k < len(pts); k++ {
		a, b := pts[k-1], pts[k]
		n := int(math.Ceil(math.Hypot(b.X-a.X, b.Y-a.Y)))
		for i := 1; i <= n; i++ {
			t := float64(i) / float64(n)
			out = append(out, Point{a.X + (b.X-a.X)*t, a.Y + (b.Y-a.Y)*t})
		}
	}
	return out
}

// inside는 점이 박스 안쪽(변에서 1px 넘게 들어간 자리)인지다 — 변 위를 지나는 선은 가로지른 것이
// 아니고, 길이 0인 프레임 변에 준 1px 깊이만큼 끝점이 비키는 것도 가로지른 것이 아니다.
func inside(pt Point, r Rect) bool {
	const m = 1.01
	return pt.X > r.X+m && pt.X < r.X+r.W-m && pt.Y > r.Y+m && pt.Y < r.Y+r.H-m
}

// outward는 면의 바깥 방향 단위 벡터다.
func outward(s Side) Point {
	switch s {
	case Top:
		return Point{0, -1}
	case Bottom:
		return Point{0, 1}
	case Left:
		return Point{-1, 0}
	default:
		return Point{1, 0}
	}
}

// 면이 상대를 등져도 선은 요청한 면에서 바깥으로 나가고, 끝 면에는 바깥에서 들어가며, 두 상자를
// 가로지르지 않는다 — 오른쪽 → 왼쪽인데 상대가 왼쪽에 있거나, 위 → 아래인데 상대가 아래에 있는 경우다.
func TestRouteSidesFacingAway(t *testing.T) {
	a := Rect{X: 700, Y: 100, W: 160, H: 80}
	b := Rect{X: 200, Y: 400, W: 160, H: 80}
	c := Rect{X: 700, Y: 400, W: 160, H: 80}
	cases := []struct {
		name     string
		from, to Rect
		fs, ts   Side
		preset   string
	}{
		{"right→left, target left", a, b, Right, Left, "Connector5"},
		{"top→bottom, target below", a, c, Top, Bottom, "Connector5"},
		{"right→top, target below-left", a, b, Right, Top, "Connector4"},
		{"bottom→left, target below-left", a, b, Bottom, Left, "Connector4"},
		{"left→bottom, target below-left", a, b, Left, Bottom, "Connector4"},
		{"left→right, target right", b, a, Left, Right, "Connector5"},
	}
	for _, tc := range cases {
		for _, kind := range []Kind{Elbow, Curved} {
			p, err := Route(tc.from, tc.to, tc.fs, tc.ts, kind)
			if err != nil {
				t.Fatal(err)
			}
			if want := map[Kind]string{Elbow: "bent", Curved: "curved"}[kind] + tc.preset; p.Preset != want {
				t.Errorf("%s %s: preset %s, want %s", tc.name, kind, p.Preset, want)
			}
			checkDetour(t, tc.name+" "+string(kind), p, tc.from, tc.to)
		}
	}
}

// 상자 둘을 여러 자리에 놓고 서로 다른 두 면의 모든 조합에서, 꺾인 경로는 요청한 면 바깥으로 나가
// 바깥에서 들어오고 어느 상자도 가로지르지 않는다. 같은 면끼리의 U자는 여기서 보지 않는다 — 한 상자가
// 다른 상자를 그 면 쪽에서 가리면 U자도 그 상자를 지난다.
func TestRouteNeverCrossesTheBoxes(t *testing.T) {
	a := Rect{X: 500, Y: 300, W: 160, H: 80}
	sides := []Side{Top, Right, Bottom, Left}
	for _, d := range []Point{{-400, -200}, {0, -200}, {400, -200}, {-400, 0}, {400, 0}, {-400, 200}, {0, 200}, {400, 200}, {-100, 200}, {300, 40}} {
		b := Rect{X: a.X + d.X, Y: a.Y + d.Y, W: 160, H: 80}
		for _, kind := range []Kind{Elbow, Curved} {
			for _, fs := range sides {
				for _, ts := range sides {
					if fs == ts {
						continue
					}
					p, err := Route(a, b, fs, ts, kind)
					if err != nil {
						t.Fatal(err)
					}
					checkDetour(t, fmt.Sprintf("%v %s %s→%s %s", d, kind, fs, ts, p.Preset), p, a, b)
				}
			}
		}
	}
}

func checkDetour(t *testing.T, name string, p Path, from, to Rect) {
	t.Helper()
	n := len(p.Points)
	first := Point{p.Points[1].X - p.Points[0].X, p.Points[1].Y - p.Points[0].Y}
	last := Point{p.Points[n-1].X - p.Points[n-2].X, p.Points[n-1].Y - p.Points[n-2].Y}
	if o := outward(p.StartSide); first.X*o.X+first.Y*o.Y <= 0 {
		t.Errorf("%s: 첫 토막 %v가 %s 면 바깥으로 나가지 않는다 (%s)", name, first, p.StartSide, p.SVG())
	}
	if o := outward(p.EndSide); last.X*o.X+last.Y*o.Y >= 0 {
		t.Errorf("%s: 끝 토막 %v가 %s 면에 바깥에서 들어오지 않는다 (%s)", name, last, p.EndSide, p.SVG())
	}
	for _, pt := range sample(p) {
		if inside(pt, from) || inside(pt, to) {
			t.Errorf("%s: 경로가 %v에서 상자를 가로지른다 (%s)", name, pt, p.SVG())
			break
		}
	}
}
