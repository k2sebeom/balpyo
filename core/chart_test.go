package core

import (
	"math"
	"sort"
	"strings"
	"testing"

	"github.com/k2sebeom/balpyo/soml"
)

// `<chart>`의 펼침: 값 층이 차트를 저자가 쓸 수 있는 SOML(도형·텍스트)로 바꾸고,
// 그 뒤의 파이프라인은 차트를 모른다.

// chartSlide는 차트 하나를 담은 슬라이드다 — `<chart>`는 2행 3열이고 점은 3행부터다.
func chartSlide(attrs, points string) string {
	open := "<chart"
	if attrs != "" {
		open += " " + attrs
	}
	return "<slide id=\"c\">\n  " + open + ">\n" + points + "  </chart>\n</slide>\n"
}

// chartPoint는 점 한 줄이다 (4열 들여쓰기).
func chartPoint(attrs string) string {
	return "    <point " + attrs + "/>\n"
}

// resolvedChart는 진단 없이 해석된 차트 슬라이드다.
func resolvedChart(t *testing.T, attrs, points string) *soml.Deck {
	t.Helper()
	deck, diags := resolveSlide(t, chartSlide(attrs, points))
	if len(diags) != 0 {
		t.Fatalf("진단이 없어야 한다: %v", diags)
	}
	return deck
}

// chartElements는 덱의 모든 요소를 문서 순서로 모은다 (펼친 트리를 훑는 도구).
func chartElements(deck *soml.Deck) []*soml.Element {
	var elems []*soml.Element
	var walk func(node soml.Node)
	walk = func(node soml.Node) {
		elem, ok := node.(*soml.Element)
		if !ok {
			return
		}
		elems = append(elems, elem)
		for _, child := range elem.Children {
			walk(child)
		}
	}
	for _, slide := range deck.Slides {
		for _, node := range slide.Content {
			walk(node)
		}
	}
	return elems
}

// chartBars는 펼침이 낸 막대들이다 — 높이가 백분율인 도형이 막대다 (기준선은 1px, 뿌리는
// 저자가 준 높이다).
func chartBars(deck *soml.Deck) []*soml.Element {
	var bars []*soml.Element
	for _, elem := range chartElements(deck) {
		style := elem.ShapeStyle()
		if style == nil || style.Height == nil || !style.Height.Percent {
			continue
		}
		bars = append(bars, elem)
	}
	return bars
}

// chartTexts는 펼침이 낸 라벨들이다 (문서 순서 = 읽기 순서).
func chartTexts(deck *soml.Deck) []*soml.Element {
	var texts []*soml.Element
	for _, elem := range chartElements(deck) {
		if elem.Name == soml.TextElement {
			texts = append(texts, elem)
		}
	}
	return texts
}

// dumpTree는 해석된 트리를 이름·속성(이름순 — 주장은 "같은 선언"이지 "같은 표기 순서"가
// 아니다)·문자 데이터로 적는다.
func dumpTree(nodes []soml.Node) string {
	parts := make([]string, 0, len(nodes))
	for _, node := range nodes {
		switch n := node.(type) {
		case *soml.Text:
			parts = append(parts, "\""+n.Value+"\"")
		case *soml.Element:
			var b strings.Builder
			b.WriteString(n.Name)
			if len(n.Attrs) > 0 {
				pairs := make([]string, 0, len(n.Attrs))
				for _, attr := range n.Attrs {
					pairs = append(pairs, attr.Name+"="+attr.Value)
				}
				sort.Strings(pairs)
				b.WriteString("[" + strings.Join(pairs, " ") + "]")
			}
			if len(n.Children) > 0 {
				b.WriteString("(" + dumpTree(n.Children) + ")")
			}
			parts = append(parts, b.String())
		}
	}
	return strings.Join(parts, " ")
}

// 펼침 등가 — 차트와 손으로 쓴 같은 뜻의 덱이 해석 후 같은 트리다. 이 한
// 장면이 기본 축(데이터 최대 ÷ 0.9 — 값 라벨의 헤드룸), 값+unit 라벨, 점 fill이 막대와 라벨을
// 함께 칠하는 것, alt이 배경 뿌리 도형에 실리는 것을 모두 못 박는다.
func TestChartExpandsToHandWrittenDeck(t *testing.T) {
	chart, diags := resolveSlide(t, chartSlide(`type="bar" unit="분" alt="월별 평균 탐지 시간" w="fill" h="420"`,
		chartPoint(`label="5월" value="22"`)+
			chartPoint(`label="6월" value="19"`)+
			chartPoint(`label="9월" value="12" fill="accent4"`)))
	if len(diags) != 0 {
		t.Fatalf("차트 덱의 진단이 없어야 한다: %v", diags)
	}

	hand, handDiags := resolveSlide(t, `<slide id="c">
  <shape type="rect" fill="none" border="none" alt="월별 평균 탐지 시간" w="fill" h="420">
    <row gap="0" align="stretch" h="fill">
      <col grow="1" basis="0" gap="0" align="center" h="fill">
        <col grow="1" basis="0" w="fill" justify="end" align="center" gap="xs">
          <text role="caption" align="center" w="fill">22분</text>
          <shape type="rect" fill="accent1" border="none" w="70%" h="90%" shrink="0"/>
        </col>
        <shape type="rect" fill="muted" border="none" w="fill" h="1"/>
        <text role="caption" align="center" w="fill">5월</text>
      </col>
      <col grow="1" basis="0" gap="0" align="center" h="fill">
        <col grow="1" basis="0" w="fill" justify="end" align="center" gap="xs">
          <text role="caption" align="center" w="fill">19분</text>
          <shape type="rect" fill="accent1" border="none" w="70%" h="77.73%" shrink="0"/>
        </col>
        <shape type="rect" fill="muted" border="none" w="fill" h="1"/>
        <text role="caption" align="center" w="fill">6월</text>
      </col>
      <col grow="1" basis="0" gap="0" align="center" h="fill">
        <col grow="1" basis="0" w="fill" justify="end" align="center" gap="xs">
          <text role="caption" align="center" w="fill" color="accent4">12분</text>
          <shape type="rect" fill="accent4" border="none" w="70%" h="49.09%" shrink="0"/>
        </col>
        <shape type="rect" fill="muted" border="none" w="fill" h="1"/>
        <text role="caption" align="center" w="fill" color="accent4">9월</text>
      </col>
    </row>
  </shape>
</slide>
`)
	if len(handDiags) != 0 {
		t.Fatalf("손으로 쓴 덱의 진단이 없어야 한다: %v", handDiags)
	}
	got, want := dumpTree(chart.Slides[0].Content), dumpTree(hand.Slides[0].Content)
	if got != want {
		t.Errorf("펼친 트리가 손으로 쓴 덱과 다르다\n펼침: %s\n손:   %s", got, want)
	}
}

// 저자가 정한 max가 축의 기준이다 (같은 값의 막대는 박스 끝까지 찬다). 값 라벨은 저자가 적은
// 표기 그대로 + unit이고(재포맷하지 않는다), 값 0은 막대 없이 라벨만 남는다 ("0"도 데이터다).
func TestChartAxisLabelsAndZero(t *testing.T) {
	deck := resolvedChart(t, `type="bar" max="44" unit="분" h="420"`,
		chartPoint(`label="5월" value="22"`)+chartPoint(`label="6월" value="19.5"`)+chartPoint(`label="7월" value="0"`))
	bars := chartBars(deck)
	want := []float64{50, 44.32}
	if len(bars) != len(want) {
		t.Fatalf("막대 %d개, want %d개 (0인 점은 막대가 없다)", len(bars), len(want))
	}
	for i, w := range want {
		if h := bars[i].ShapeStyle().Height; math.Abs(h.Value-w) > 0.005 {
			t.Errorf("막대[%d] 높이 = %g%%, want %g%%", i, h.Value, w)
		}
	}
	var labels []string
	for _, text := range chartTexts(deck) {
		labels = append(labels, text.TextContent())
	}
	if got, want := strings.Join(labels, "|"), "22분|5월|19.5분|6월|0분|7월"; got != want {
		t.Errorf("라벨 = %s, want %s", got, want)
	}
	if first := chartTexts(resolvedChart(t, `type="bar" h="420"`, chartPoint(`label="5월" value="22"`)))[0].TextContent(); first != "22" {
		t.Errorf("단위 없는 값 라벨 = %q, want 22", first)
	}
}

// 계열색(series)은 모든 막대를 칠하되 라벨은 칠하지 않는다 — 면과 글은 대비 요구가
// 달라(surface 글자는 흰 배경에서 사라진다) 같은 채널이 될 수 없다. 점의 fill은 계열을 이긴다.
func TestChartSeriesColorsEveryBarButNotTheLabels(t *testing.T) {
	deck := resolvedChart(t, `type="bar" series="surface" h="420"`,
		chartPoint(`label="5월" value="22"`)+chartPoint(`label="9월" value="12" fill="accent4"`))
	bars := chartBars(deck)
	texts := chartTexts(deck)
	if len(bars) != 2 || len(texts) != 4 {
		t.Fatalf("막대 %d개 · 라벨 %d개, want 2 · 4", len(bars), len(texts))
	}
	caption := textRoleDefaults(soml.RoleCaption).Color
	for i, want := range []string{"surface", "accent4"} {
		if fill := bars[i].ShapeStyle().Fill; fill == nil || fill.Color.Token != want {
			t.Errorf("막대[%d] 채움 = %v, want %s", i, fill, want)
		}
	}
	for i, text := range texts {
		want := caption
		if i >= 2 {
			want = soml.TokenColor("accent4")
		}
		if got := text.TextStyle().Color; got != want {
			t.Errorf("라벨[%d] 색 = %v, want %v", i, got, want)
		}
	}
}

// 펼침 뒤의 트리에는 펼쳐지는 종류가 하나도 없다 — 렌더러가 `<chart>`를 보지 않는다는 사실의
// 기계 판정이다 (렌더러 쪽 짝은 render/vocabulary_test.go).
func TestResolvedTreeHasNoDesugaredKinds(t *testing.T) {
	deck := resolvedChart(t, `type="bar" unit="분" h="420"`,
		chartPoint(`label="5월" value="22"`)+chartPoint(`label="6월" value="19"`))
	for _, elem := range chartElements(deck) {
		if soml.KindDesugared(soml.KindOf(elem.Name)) {
			t.Errorf("확정 트리에 <%s>가 남아 있다 (%s) — 값 층이 펼쳐야 한다", elem.Name, elem.Span)
		}
	}
}

// 진단 — 값 어휘와 데이터의 요구는 값 층이 소유하고, 자리는 합성 노드가 아니라 저자가 쓴
// `<chart>`/`<point>`의 속성이다.
func TestChartDiagnostics(t *testing.T) {
	p := chartPoint(`label="5월" value="22"`)
	tests := []struct {
		name, attrs, points string
		contains            []string
	}{
		{"type이 없다", `h="420"`, p, []string{"<chart> needs a type", "available types: bar", "no way to guess"}},
		{"어휘 밖의 type", `type="pie" h="420"`, p, []string{`unknown chart type "pie"`, "available types: bar, line, area, step"}},
		{"선 차트의 점 강조", `type="line" h="420"`, chartPoint(`label="May" value="22" fill="accent4"`), []string{"<point fill> highlights one bar — a line chart"}},
		// 막대 높이가 축에 대한 백분율이라 차트 높이를 내용이 정하면 순환한다.
		{"h도 ratio도 없다", `type="bar"`, p, []string{"<chart> needs h or ratio", "would be circular"}},
		{"점이 없다", `type="bar" h="420"`, "", []string{"<chart> has no <point>", "no data to draw"}},
		{"label이 없다", `type="bar" h="420"`, chartPoint(`value="22"`), []string{"<point> needs a label"}},
		{"value가 없다", `type="bar" h="420"`, chartPoint(`label="5월"`), []string{"<point> needs a value"}},
		{"value가 수치가 아니다", `type="bar" unit="분" h="420"`, chartPoint(`label="5월" value="22분"`),
			[]string{`value is not a number ("22분")`, "put the unit in <chart unit>"}},
		{"value가 음수다", `type="bar" h="420"`, chartPoint(`label="5월" value="-3"`), []string{"value cannot be negative", "starts at 0"}},
		{"값이 전부 0이다", `type="bar" h="420"`, chartPoint(`label="5월" value="0"`) + chartPoint(`label="6월" value="0"`),
			[]string{"are 0", "axis cannot be determined"}},
		{"value가 max를 넘는다", `type="bar" max="20" h="420"`, p, []string{"value 22 is greater than max 20", "the bar exceeds the axis"}},
		{"max가 수치가 아니다", `type="bar" max="스물" h="420"`, p, []string{`max is not a number ("스물")`}},
		{"max가 0 이하다", `type="bar" max="0" h="420"`, chartPoint(`label="5월" value="1"`), []string{"max must be greater than 0"}},
		{"unit에 값이 없다", `type="bar" unit h="420"`, p, []string{"the unit attribute needs a value"}},
		{"alt이 비었다", `type="bar" alt="" h="420"`, p, []string{"alt cannot be empty"}},
		// 점의 fill·series는 도형 fill과 같은 어휘·문구이고, 한 오타에 진단은 하나다 (막대와 라벨에
		// 두 번 말하지 않는다). 진단은 저자가 쓴 이름으로 말한다.
		{"점의 fill이 테마에 없다", `type="bar" h="420"`, chartPoint(`label="5월" value="22" fill="brand"`),
			[]string{`fill: color token "brand" is not in the theme`}},
		{"series가 테마에 없다", `type="bar" series="brand" h="420"`, p, []string{`series: color token "brand" is not in the theme`}},
		// 그리지 않는 막대는 데이터가 아니라 none만 빠진다.
		{"series가 none이다", `type="bar" series="none" h="420"`, p, []string{`series cannot be none`, `value="0"`}},
		{"점의 fill이 none이다", `type="bar" h="420"`, chartPoint(`label="5월" value="22" fill="none"`), []string{`fill cannot be none`}},
		{"series에 값이 없다", `type="bar" series h="420"`, p, []string{"the series attribute needs a value"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, diag := oneDiag(t, chartSlide(tc.attrs, tc.points))
			assertDiag(t, diag, 0, 0, tc.contains...)
		})
	}

	// 점의 값 진단은 그 점(둘째 점 = 4행)의 값에 밑줄을 긋는다. col은 룬으로 센다 (soml.Pos).
	attrs := `label="6월" value="-3"`
	_, diag := oneDiag(t, chartSlide(`type="bar" h="420"`, p+chartPoint(attrs)))
	assertDiag(t, diag, 4, len([]rune(`    <point label="6월" value="`))+1)
}

// 선 차트는 경로 도형 하나로 펼쳐진다 — 점은 범주 칸의 가운데(i + ½)/n, 세로는 값 ÷ 축을 아래에서 잰 자리다.
// 계단은 칸마다 수평인 단이고, 영역은 같은 선 아래를 기준선까지 닫은 면을 먼저 깐다.
func TestChartLineExpandsToAPath(t *testing.T) {
	for _, tc := range []struct{ kind, want string }{
		{"line", "M25,75 L75,25"},
		{"step", "M0,75 H50 V25 H100"},
		{"area", "M25,75 L75,25 L75,100 L25,100 Z"},
	} {
		t.Run(tc.kind, func(t *testing.T) {
			deck, diags := resolveSlide(t, "<slide>\n  <chart type=\""+tc.kind+"\" max=\"100\" h=\"300\">"+
				chartPoint(`label="a" value="25"`)+chartPoint(`label="b" value="75"`)+"</chart>\n</slide>\n")
			if len(diags) != 0 {
				t.Fatalf("진단: %v", diags)
			}
			found := false
			var walk func(n soml.Node)
			walk = func(n soml.Node) {
				e, ok := n.(*soml.Element)
				if !ok {
					return
				}
				if d, ok := e.Attr("d"); ok && d.Value == tc.want {
					found = true
				}
				for _, c := range e.Children {
					walk(c)
				}
			}
			for _, n := range deck.Slides[0].Content {
				walk(n)
			}
			if !found {
				t.Errorf("펼친 트리에 d=%q인 경로 도형이 없다", tc.want)
			}
		})
	}
}

// chartSeriesXML은 계열 하나다 — 점들은 속성 문자열의 목록이다.
func chartSeriesXML(attrs string, points ...string) string {
	b := "    <series " + attrs + ">\n"
	for _, p := range points {
		b += "  " + chartPoint(p)
	}
	return b + "    </series>\n"
}

// 계열 형식: 축은 모든 계열의 최대에서 정하고, 막대는 범주마다 계열 수만큼 서며, 범례가 계열마다 이름 하나를 단다.
func TestChartSeriesShareTheAxisAndGetALegend(t *testing.T) {
	deck := resolvedChart(t, `type="bar" h="420"`,
		chartSeriesXML(`name="A"`, `label="Q1" value="9"`, `label="Q2" value="18"`)+
			chartSeriesXML(`name="B" fill="accent4"`, `label="Q1" value="4.5"`, `label="Q2" value="9"`))
	bars := chartBars(deck)
	if len(bars) != 4 {
		t.Fatalf("막대 %d개, want 4 (범주 2 × 계열 2)", len(bars))
	}
	// 축 = 18 ÷ 0.9 = 20 — 둘째 계열의 9는 45%다.
	want := []float64{45, 90, 22.5, 45}
	order := []int{0, 2, 1, 3} // 문서 순서는 범주마다 계열 순서다 (Q1-A, Q1-B, Q2-A, Q2-B).
	for i, bar := range bars {
		got := bar.ShapeStyle().Height.Value
		if math.Abs(got-want[order[i]]) > 0.01 {
			t.Errorf("막대 %d 높이 = %v%%, want %v%%", i, got, want[order[i]])
		}
	}
	if fill := bars[1].ShapeStyle().Fill.Color.Token; fill != "accent4" {
		t.Errorf("둘째 계열 막대 색 = %s, want accent4", fill)
	}
	if fill := bars[0].ShapeStyle().Fill.Color.Token; fill != "accent1" {
		t.Errorf("첫 계열 기본색 = %s, want accent1", fill)
	}
	var names []string
	for _, text := range chartTexts(deck) {
		if v := dumpTree(text.Children); v == `"A"` || v == `"B"` {
			names = append(names, v)
		}
	}
	if strings.Join(names, " ") != `"A" "B"` {
		t.Errorf("범례 이름 = %v", names)
	}
}

func TestChartSeriesDiagnostics(t *testing.T) {
	a := chartSeriesXML(`name="A"`, `label="Q1" value="1"`, `label="Q2" value="2"`)
	for _, tc := range []struct {
		name, attrs, body, want string
	}{
		{"점과 계열을 섞었다", `type="bar" h="420"`, a + chartPoint(`label="Q1" value="1"`), "<chart> holds either <point>s or <series>, not both"},
		{"프레임 series", `type="bar" series="accent2" h="420"`, a, "<chart series> colors the only series"},
		{"이름이 없다", `type="bar" h="420"`, a + chartSeriesXML(`fill="accent2"`, `label="Q1" value="1"`, `label="Q2" value="2"`), "<series> needs a name"},
		{"범주 수", `type="bar" h="420"`, a + chartSeriesXML(`name="B"`, `label="Q1" value="1"`), `<series name="B"> has 1 points but the first series has 2`},
		{"범주 라벨", `type="bar" h="420"`, a + chartSeriesXML(`name="B"`, `label="Q1" value="1"`, `label="Q3" value="2"`), `<point label="Q3"> is category 2 — the first series has "Q2" there`},
		{"막대 차트의 계열 type", `type="bar" h="420"`, a + chartSeriesXML(`name="B" type="line"`, `label="Q1" value="1"`, `label="Q2" value="2"`), "<series type> is for line, area and step charts"},
		{"모르는 계열 type", `type="line" h="420"`, a + chartSeriesXML(`name="B" type="bar"`, `label="Q1" value="1"`, `label="Q2" value="2"`), `unknown series type "bar" (available series types: line, area, step)`},
		{"빈 계열", `type="bar" h="420"`, a + "    <series name=\"B\"></series>\n", "<series> has no <point>"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, diags := resolveSlide(t, chartSlide(tc.attrs, tc.body))
			found := false
			for _, d := range diags {
				found = found || strings.Contains(d.Message, tc.want)
			}
			if !found {
				t.Errorf("진단 = %v, want %q", diags, tc.want)
			}
		})
	}
}
