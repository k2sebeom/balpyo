package pptx

import (
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/k2sebeom/balpyo/render/geom"
	"github.com/k2sebeom/balpyo/render/web"
)

func TestEMU(t *testing.T) {
	tests := []struct {
		px   float64
		want int64
	}{
		{0, 0},
		{1, 9525},
		{96, EMUPerInch}, // 96px = 1인치
		{1280, 12192000},
		{0.5, 4763}, // 반올림
		{-2, -19050},
	}
	for _, tc := range tests {
		if got := EMU(tc.px); got != tc.want {
			t.Errorf("EMU(%v) = %d, want %d", tc.px, got, tc.want)
		}
	}
}

func TestLineSpacing(t *testing.T) {
	// OOXML spcPts@val은 1/100 pt 단위다 — 배수 × 폰트 크기(px) × 0.75(pt/px) × 100.
	// 절대값으로 내는 이유는 units.go LineSpacing의 주석에 있다 (spcPct는 엔진마다 기준이 다르다).
	tests := []struct {
		multiplier float64
		sizePx     float64
		want       int64
	}{
		{1.5, 24, 2700}, // 본문: 36px 줄 = 27pt
		{1.2, 56, 5040}, // 제목: 67.2px 줄 = 50.4pt
		{1.4, 18, 1890}, // 캡션
		{1.3, 36, 3510}, // 헤딩
		{1, 16, 1200},
		{1.05, 21, 1654}, // 반올림 (22.05px = 16.5375pt)
	}
	for _, tc := range tests {
		if got := LineSpacing(tc.multiplier, tc.sizePx); got != tc.want {
			t.Errorf("LineSpacing(%v, %v) = %d, want %d", tc.multiplier, tc.sizePx, got, tc.want)
		}
	}
}

func TestFontSize(t *testing.T) {
	// OOXML sz는 1/100 pt 단위이고 pt = px × 0.75다.
	tests := []struct {
		px   float64
		want int64
	}{
		{16, 1200}, // 16px = 12pt
		{64, 4800},
		{14, 1050},
	}
	for _, tc := range tests {
		if got := FontSize(tc.px); got != tc.want {
			t.Errorf("FontSize(%v) = %d, want %d", tc.px, got, tc.want)
		}
	}
}

// 지오메트리 패스가 잰 박스는 정수 EMU로 정확히 떨어져야 한다 — 좌표에 부동소수 잔차가
// 남으면 같은 입력이 실행마다 다른 xfrm을 내고 골든이 흔들린다.
func TestOffsetAndExtent(t *testing.T) {
	tests := []struct {
		name           string
		rect           geom.Rect
		wantX, wantY   int64
		wantCX, wantCY int64
	}{
		{
			name:  "스테이지 전체",
			rect:  geom.Rect{X: 0, Y: 0, W: 1280, H: 720},
			wantX: 0, wantY: 0, wantCX: 12192000, wantCY: 6858000,
		},
		{
			name:  "스테이지 여백 안의 텍스트 박스",
			rect:  geom.Rect{X: 64, Y: 64, W: 1152, H: 67.2},
			wantX: 609600, wantY: 609600, wantCX: 10972800, wantCY: 640080,
		},
		{
			name:  "서브픽셀은 반올림한다",
			rect:  geom.Rect{X: 0.5, Y: 1.25, W: 0.5, H: 100.125},
			wantX: 4763, wantY: 11906, wantCX: 4763, wantCY: 953691,
		},
		{
			name:  "스테이지를 넘어간 박스도 그대로 간다 (오버플로 허용 — ADR 0067 결정 1)",
			rect:  geom.Rect{X: -32, Y: 700, W: 1344, H: 100},
			wantX: -304800, wantY: 6667500, wantCX: 12801600, wantCY: 952500,
		},
		{
			// 반올림은 0에서 먼 쪽이다: -0.5px = -4762.5 EMU → -4763 (절단이면 -4762가 되어
			// 음수 좌표만 반 px씩 어긋난다).
			name:  "음수 좌표는 0쪽으로 치우치지 않는다 (0.5 EMU 경계)",
			rect:  geom.Rect{X: -0.5, Y: -1.5, W: 0.5, H: 1.5},
			wantX: -4763, wantY: -14288, wantCX: 4763, wantCY: 14288,
		},
		{
			name:  "스테이지 밖 음수 소수 좌표",
			rect:  geom.Rect{X: -100.25, Y: -2.5, W: 10, H: 10},
			wantX: -954881, wantY: -23813, wantCX: 95250, wantCY: 95250,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			x, y := Offset(tc.rect)
			cx, cy := Extent(tc.rect)
			if x != tc.wantX || y != tc.wantY {
				t.Errorf("Offset(%s) = (%d,%d), want (%d,%d)", tc.rect, x, y, tc.wantX, tc.wantY)
			}
			if cx != tc.wantCX || cy != tc.wantCY {
				t.Errorf("Extent(%s) = (%d,%d), want (%d,%d)", tc.rect, cx, cy, tc.wantCX, tc.wantCY)
			}
		})
	}
}

// EMU·pt 환산은 이 패키지 경계에서만 일어난다 (ADR 0067 결정 5). 지오메트리 패스가 EMU를 알게
// 되면 좌표계가 둘로 갈라지고, 어느 쪽이 논리 px인지 코드를 읽어 봐야 알게 된다.
//
// 주석은 검사에서 뺀다 — 경계를 **설명하는** 문장(render/geom의 패키지 주석)은 있어야 하고,
// 없어야 하는 것은 환산하는 코드다.
func TestGeometryPassKnowsNoEMU(t *testing.T) {
	dir := filepath.Join("..", "geom")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	files := 0
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		files++
		code := codeWithoutComments(t, filepath.Join(dir, name))
		for _, banned := range []string{"EMU", "9525", "914400", "PointsPerPixel"} {
			if strings.Contains(code, banned) {
				t.Errorf("render/geom/%s의 코드에 %q가 있다 — 단위 변환은 render/pptx가 소유한다", name, banned)
			}
		}
	}
	if files == 0 {
		t.Fatalf("%s에서 검사할 파일을 찾지 못했다", dir)
	}
}

// codeWithoutComments는 주석을 뺀 소스다.
func codeWithoutComments(t *testing.T, path string) string {
	t.Helper()
	fset := token.NewFileSet()
	// ParseComments를 주지 않으면 주석이 AST에 붙지 않으므로 출력에서도 사라진다.
	file, err := parser.ParseFile(fset, path, nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	var b strings.Builder
	if err := format.Node(&b, fset, file); err != nil {
		t.Fatal(err)
	}
	return b.String()
}

// 회전은 단위 환산 + 정규화다 — 방향 규약이 두 타깃에서 같으므로 부호를 뒤집지 않는다.
func TestRotation(t *testing.T) {
	tests := []struct {
		deg  float64
		want int64
	}{
		{0, 0},
		{45, 2700000},
		{90, 5400000},
		{359.5, 21570000},
		// 음수·한 바퀴 이상은 접는다 — 뷰어의 관용에 기대지 않고, 재저장 diff가 이 속성에서
		// 나지 않게 한다 (Rotation의 근거).
		{-90, 16200000},
		{-360, 0},
		{360, 0},
		{405, 2700000},
		{-450, 16200000},
		{720.5, 30000},
	}
	for _, tc := range tests {
		got := Rotation(tc.deg)
		if got != tc.want {
			t.Errorf("Rotation(%v) = %d, want %d", tc.deg, got, tc.want)
		}
		if got < 0 || got >= FullRotation {
			t.Errorf("Rotation(%v) = %d가 한 바퀴 밖이다", tc.deg, got)
		}
	}
}

// 알파는 100000분율이다 (0~1 검사는 값 층의 몫 — 환산 함수는 검증을 갖지 않는다).
func TestAlpha(t *testing.T) {
	tests := []struct {
		v    float64
		want int64
	}{
		{0, 0},
		{0.15, 15000},
		{0.4, 40000},
		{0.06, 6000},
		{1, 100000},
		{0.123456, 12346}, // 반올림
	}
	for _, tc := range tests {
		if got := Alpha(tc.v); got != tc.want {
			t.Errorf("Alpha(%v) = %d, want %d", tc.v, got, tc.want)
		}
	}
}

// 조정값 평가 — `val = Base × 값 ÷ ss`의 경계들. 명세는 웹의 기하 표가 소유하고
// (web.ShapeAdjusts) 여기서 잰 박스로 평가된다.
func TestAdjustValue(t *testing.T) {
	radiusSpec := web.ShapeAdjust{Name: "adj", Base: 100000, Axis: web.AdjustAxisRadius, Max: 50000}
	widthSpec := web.ShapeAdjust{Name: "adj", Base: 50000, Axis: web.AdjustAxisWidth}
	heightSpec := web.ShapeAdjust{Name: "adj", Base: 25000, Axis: web.AdjustAxisHeight}

	tests := []struct {
		name   string
		spec   web.ShapeAdjust
		rect   geom.Rect
		radius float64
		want   int64
	}{
		// 정사각형: ss가 두 축과 같으므로 w/h 기준 공식이 프리셋 기본값 그대로 남는다.
		{"정사각형/폭", widthSpec, geom.Rect{W: 200, H: 200}, 0, 50000},
		{"정사각형/높이", heightSpec, geom.Rect{W: 200, H: 200}, 0, 25000},
		// 아주 납작한 도형: ss = h이므로 w 기준 조정값이 비율만큼 커진다 (늘린 정규 공간을
		// 되돌리는 것이 이 값의 뜻이다).
		{"납작/폭", widthSpec, geom.Rect{W: 1000, H: 50}, 0, 1000000},
		{"납작/높이", heightSpec, geom.Rect{W: 1000, H: 50}, 0, 25000},
		// 아주 좁은 도형: ss = w.
		{"좁음/높이", heightSpec, geom.Rect{W: 50, H: 1000}, 0, 500000},
		// 반경: ss의 절반이 상한이다 (adj 50000 = 완전한 반원).
		{"반경/일반", radiusSpec, geom.Rect{W: 300, H: 160}, 12, 7500},
		{"반경/상한 정확히", radiusSpec, geom.Rect{W: 300, H: 160}, 80, 50000},
		{"반경/상한 넘김", radiusSpec, geom.Rect{W: 300, H: 160}, 400, 50000},
		{"반경/0", radiusSpec, geom.Rect{W: 300, H: 160}, 0, 0},
		{"반경/반올림", radiusSpec, geom.Rect{W: 220, H: 110}, 12, 10909},
		// 크기 0 도형은 그릴 것이 없다 — 0으로 나눈 Inf가 val에 실리면 파일이 열리지 않는다.
		{"크기 0", radiusSpec, geom.Rect{}, 12, 0},
		{"높이 0", widthSpec, geom.Rect{W: 200}, 0, 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := AdjustValue(tc.spec, tc.rect, tc.radius)
			if err != nil {
				t.Fatalf("AdjustValue: %v", err)
			}
			if got != tc.want {
				t.Errorf("AdjustValue(%s, %s, r=%v) = %d, want %d",
					tc.spec.Name, tc.rect, tc.radius, got, tc.want)
			}
		})
	}

	// 기하 표에 축을 추가하고 평가를 잊으면 조용히 0이 되는 대신 멈춘다.
	if _, err := AdjustValue(web.ShapeAdjust{Name: "adj", Base: 1, Axis: "diagonal"},
		geom.Rect{W: 100, H: 100}, 0); err == nil {
		t.Error("모르는 축은 에러여야 한다")
	}
}
