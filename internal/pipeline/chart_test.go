package pipeline_test

import (
	"math"
	"strconv"
	"strings"
	"testing"

	"github.com/k2sebeom/balpyo/internal/pipeline"
	"github.com/k2sebeom/balpyo/internal/testkit"
	"github.com/k2sebeom/balpyo/render"
	"github.com/k2sebeom/balpyo/render/geom"
)

// 막대 차트는 낮게 그려도 제자리에 있고, 막대 높이는 잰 뒤에도 데이터의 비율이다.
//
// 두 회귀를 함께 본다: (1) 값 라벨은 가장 높은 막대 위에서도 밴드를 넘지 않는다 — 헤드룸
// 비율만으로는 낮은 차트에서 캡션 한 줄이 들어가지 않았다. (2) 두 줄로 접히는 범주 라벨이
// 그 칸의 막대만 줄이지 않는다 — 모든 막대가 같은 기준선에 서고 높이가 값에 비례한다.
func TestShortBarChartFitsAndKeepsProportions(t *testing.T) {
	testkit.RequireChromium(t)
	dir := project(t, map[string]string{
		"balpyo.json": `{"title": "chart"}`,
		"slides/001_chart.slide": `<slide id="c">
  <chart type="bar" w="480" h="160">
    <point label="A" value="80"/>
    <point label="A category label long enough to wrap onto two lines" value="40"/>
    <point label="C" value="20"/>
  </chart>
</slide>`,
	})
	result, err := pipeline.Build(pipeline.Options{
		ProjectDir:  dir,
		Targets:     []render.Target{render.TargetHTML, render.TargetPPTX},
		CheckLayout: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Diagnostics) > 0 {
		t.Errorf("낮은 차트의 진단 %d건, want 0건:\n%s", len(result.Diagnostics), result.Diagnostics)
	}
	// 막대는 차트 뿌리(가장 큰 도형)와 1px 기준선을 뺀 도형이다.
	var bars []geom.Rect
	for _, box := range result.Geometry.Boxes {
		if box.El == "shape" && box.Rect.H > 1 && box.Rect.W < 480 {
			bars = append(bars, box.Rect)
		}
	}
	if len(bars) != 3 {
		t.Fatalf("막대 %d개, want 3: %v", len(bars), bars)
	}
	for i, want := range []float64{1, 0.5, 0.25} {
		if got := bars[i].H / bars[0].H; math.Abs(got-want) > 0.02 {
			t.Errorf("막대[%d] 높이 비 = %.3f, want %.2f (높이 %v)", i, got, want, bars)
		}
		if math.Abs(bars[i].Bottom()-bars[0].Bottom()) > 0.5 {
			t.Errorf("막대[%d]의 바닥 %.1f, 첫 막대 %.1f — 기준선이 어긋났다", i, bars[i].Bottom(), bars[0].Bottom())
		}
	}
}

// 펼친 차트의 노드가 넘치면 진단은 저자가 쓴 태그로 말한다 — 값 라벨과 막대는 `<point>`이고
// 자리도 `<point>`다. 저자는 `<text>`를 쓰지 않았으므로 `<text>`라고 들으면 고칠 자리를 찾지 못한다.
func TestChartLayoutDiagnosticsNameTheAuthoredTag(t *testing.T) {
	testkit.RequireChromium(t)
	dir := project(t, map[string]string{
		"balpyo.json": `{"title": "chart"}`,
		"slides/001_chart.slide": `<slide id="c">
  <chart type="bar" w="240" h="200">
    <point label="A" value="80"/>
    <point label="Pneumonoultramicroscopicsilicovolcanoconiosis" value="40"/>
    <point label="C" value="20"/>
  </chart>
</slide>`,
	})
	result, err := pipeline.Build(pipeline.Options{
		ProjectDir:  dir,
		Targets:     []render.Target{render.TargetHTML},
		CheckLayout: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Diagnostics) == 0 {
		t.Fatal("넘치는 범주 라벨이 진단을 내지 않았다")
	}
	for _, d := range result.Diagnostics {
		if strings.Contains(d.Message, "<text>") || strings.Contains(d.Message, "<shape>") {
			t.Errorf("진단이 저자가 쓰지 않은 태그를 말한다: %s", d)
		}
		// 긴 라벨이 칸을 밀어 넘친 것은 이웃 칸일 수 있다 — 자리는 어느 `<point>` 줄(3~5)이든 된다.
		if line := d.Span.Start.Line; !strings.HasPrefix(d.Message, "<point>") || line < 3 || line > 5 {
			t.Errorf("진단 = %s, want <point> 줄의 <point>", d)
		}
	}
}

// chartBoxes는 한 차트 슬라이드에서 잰 막대와 범주 라벨이다 — 막대는 차트 뿌리(가장 큰 도형)와 1px
// 기준선을 뺀 도형이고, 범주 라벨은 막대 밴드 바닥보다 아래에 선 글자 상자다.
func chartBoxes(g *geom.Geometry, chartW float64) (bars, labels []geom.Rect) {
	bottom := 0.0
	for _, box := range g.Boxes {
		if box.El == "shape" && box.Rect.H > 1 && box.Rect.W < chartW {
			bars = append(bars, box.Rect)
			bottom = math.Max(bottom, box.Rect.Bottom())
		}
	}
	for _, box := range g.Boxes {
		if box.El == "text" && box.Rect.Y >= bottom {
			labels = append(labels, box.Rect)
		}
	}
	return bars, labels
}

// 범주 라벨은 자기 막대와 같은 열에 선다 — 칸보다 넓은 라벨 하나가 자기 칸을 넓혀 이웃 라벨을
// 밀어내지 않는다. 넓은 라벨은 자기 칸 안에서 넘치고, 그 넘침이 그 `<point>`의 경고가 된다.
func TestChartCategoryLabelsStayInTheirColumns(t *testing.T) {
	testkit.RequireChromium(t)
	dir := project(t, map[string]string{
		"balpyo.json": `{"title": "chart"}`,
		"slides/001_chart.slide": `<slide id="c">
  <chart type="bar" alt="x" w="600" h="300">
    <point label="Internationalization" value="10"/>
    <point label="b" value="20"/>
    <point label="c" value="30"/>
    <point label="d" value="40"/>
    <point label="e" value="50"/>
  </chart>
</slide>`,
	})
	result, err := pipeline.Build(pipeline.Options{
		ProjectDir:  dir,
		Targets:     []render.Target{render.TargetHTML, render.TargetPPTX},
		CheckLayout: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	bars, labels := chartBoxes(result.Geometry, 600)
	if len(bars) != 5 || len(labels) != 5 {
		t.Fatalf("막대 %d개 · 라벨 %d개, want 5 · 5: %v %v", len(bars), len(labels), bars, labels)
	}
	for i := range bars {
		barMid, labelMid := bars[i].X+bars[i].W/2, labels[i].X+labels[i].W/2
		if math.Abs(barMid-labelMid) > 0.5 {
			t.Errorf("라벨[%d] 가운데 %.1f, 막대 가운데 %.1f — 라벨이 자기 열을 벗어났다", i, labelMid, barMid)
		}
		if math.Abs(labels[i].W-labels[0].W) > 0.5 {
			t.Errorf("라벨[%d] 칸 폭 %.1f, 첫 칸 %.1f — 칸이 균등하지 않다", i, labels[i].W, labels[0].W)
		}
	}
	diags := result.Diagnostics
	if len(diags) != 1 || !strings.HasPrefix(diags[0].Message, "<point> text overflows its box") || diags[0].Span.Start.Line != 3 {
		t.Errorf("진단 = %v, want 3행 <point>의 글자 넘침 하나", diags)
	}
}

// 점이 많아 라벨이 칸보다 넓어도 라벨 줄은 막대 밴드보다 넓어지지 않는다 — 라벨 i의 가운데는
// 차트 폭을 n칸으로 나눈 i번째 칸의 가운데다. 선 차트의 점도 같은 칸 나눔에 선다.
func TestManyChartLabelsKeepTheirColumns(t *testing.T) {
	testkit.RequireChromium(t)
	const n = 40
	for _, kind := range []string{"bar", "line"} {
		t.Run(kind, func(t *testing.T) {
			var points strings.Builder
			for i := 0; i < n; i++ {
				points.WriteString(`    <point label="May" value="` + strconv.Itoa(i+1) + `"/>` + "\n")
			}
			dir := project(t, map[string]string{
				"balpyo.json":            `{"title": "chart"}`,
				"slides/001_chart.slide": "<slide id=\"c\">\n  <chart type=\"" + kind + "\" alt=\"x\" w=\"600\" h=\"300\">\n" + points.String() + "  </chart>\n</slide>\n",
			})
			result, err := pipeline.Build(pipeline.Options{
				ProjectDir:  dir,
				Targets:     []render.Target{render.TargetHTML},
				CheckLayout: true,
			})
			if err != nil {
				t.Fatal(err)
			}
			chart := result.Geometry.Boxes[0].Rect
			if chart.W != 600 {
				t.Fatalf("첫 박스 %v, want 폭 600의 차트 뿌리", chart)
			}
			_, labels := chartBoxes(result.Geometry, 600)
			if len(labels) != n {
				t.Fatalf("라벨 %d개, want %d", len(labels), n)
			}
			for i, label := range labels {
				want := chart.X + (float64(i)+0.5)*chart.W/n
				if mid := label.X + label.W/2; math.Abs(mid-want) > 0.5 {
					t.Errorf("라벨[%d] 가운데 %.1f, want %.1f (차트 %v)", i, mid, want, chart)
				}
			}
		})
	}
}
