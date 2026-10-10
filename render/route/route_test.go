package route

import (
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
		if p.Adj >= 0 && p.Adj <= 1 {
			t.Errorf("%s: adj %g는 프레임 안이다 — U자는 바깥에서 꺾어야 한다", kind, p.Adj)
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
