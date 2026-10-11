package pipeline_test

import (
	"math"
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
