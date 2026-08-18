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
	"github.com/k2sebeom/balpyo/soml"
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
	// OOXML spcPct@val은 1/1000 % 단위다 — 배수 1.5 = 150%.
	tests := []struct {
		multiplier float64
		want       int64
	}{
		{1.5, 150000},
		{1.2, 120000},
		{1.3, 130000},
		{1, 100000},
	}
	for _, tc := range tests {
		if got := LineSpacing(tc.multiplier); got != tc.want {
			t.Errorf("LineSpacing(%v) = %d, want %d", tc.multiplier, got, tc.want)
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
			name:  "스테이지를 넘어간 박스도 그대로 간다 (오버플로 허용 — PRD §13-1)",
			rect:  geom.Rect{X: -32, Y: 700, W: 1344, H: 100},
			wantX: -304800, wantY: 6667500, wantCX: 12801600, wantCY: 952500,
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

// EMU·pt 환산은 이 패키지 경계에서만 일어난다 (PRD §13-5). 지오메트리 패스가 EMU를 알게
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

// 논리 스테이지 → PPTX 페이지 크기가 PowerPoint 표준과 정확히 일치해야 한다.
// 어긋나면 산출물을 기존 덱에 붙일 때 PowerPoint가 슬라이드 크기 변환을 묻는다.
func TestSlideSizeMatchesPowerPointStandards(t *testing.T) {
	tests := []struct {
		aspect     soml.Aspect
		wantCX     int64
		wantCY     int64
		wantInches string
	}{
		{soml.Aspect16x9, 12192000, 6858000, "13.333×7.5in"},
		{soml.Aspect4x3, 9144000, 6858000, "10×7.5in"},
	}
	for _, tc := range tests {
		stage, err := tc.aspect.Stage()
		if err != nil {
			t.Fatal(err)
		}
		cx, cy := SlideSize(stage)
		if cx != tc.wantCX || cy != tc.wantCY {
			t.Errorf("%s(%dx%dpx) = %dx%d EMU, want %dx%d (%s)",
				tc.aspect, stage.Width, stage.Height, cx, cy, tc.wantCX, tc.wantCY, tc.wantInches)
		}
	}
}
