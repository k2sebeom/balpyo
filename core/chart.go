package core

import (
	"math"
	"strconv"
	"strings"

	"github.com/k2sebeom/balpyo/soml"
)

// 이 파일은 `<chart>`의 **펼침**이다 — 값 층의 다른 갈래와 갈라지는 자리가 하나
// 있다: 나머지 갈래는 속성을 확정 값으로 눕히지만(resolveShape·resolveTable) 이 갈래는
// **트리를 바꾼다**. 차트 하나가 도형·텍스트의 하위 트리가 되고, 그 뒤의 파이프라인
// (지오메트리·웹·PPTX)은 차트라는 것을 모른다.
//
// 그렇게 하는 근거는 둘이다:
//
//   - **산출물에 차트라는 물건이 없다.** 네이티브 차트를 내면 값축·눈금·범례·라벨
//     타이포그래피를 뷰어가 고르므로 같은 소스가 웹과 PPTX에서 다른 그림이 되고, 픽셀 비교로
//     회귀를 잡을 수 없다. 도형·텍스트는 두 타깃이 이미 같게 그린다.
//   - **펼침은 저자가 쓸 수 있는 SOML만 쓴다.** 그래서 손으로 쓴 같은 뜻의 덱과 펼친 트리가
//     같은지 단정할 수 있고(TestChartExpandsToHandWrittenDeck), 그 판정이 "차트가 비공개
//     채널을 몰래 열지 못한다"는 유일한 관문이다.
//
// 좌표 산수는 어디에도 없다: 막대 높이는 `값 ÷ 축 최대`의 **백분율**이고 픽셀은 브라우저가
// 잰다. 그것이 이 파일이 200줄로 끝나는 이유다.

// 펼침의 상수. 값을 이름으로 두는 이유는 문서·픽스처와 같은 숫자를 봐야 하기 때문이다 —
// 헤드룸이나 막대 폭을 바꾸는 커밋은 여기 한 줄만 고친다.
const (
	// chartFrameType은 뿌리 도형과 막대·기준선의 프리셋이다. 차트에 다른 프리셋을 쓸 자리는
	// 없다 — 막대는 사각형이고, 모서리를 둥글리는 손잡이는 열지 않았다.
	chartFrameType = "rect"
	// chartHeadroom은 축을 생략했을 때 **가장 높은 막대가 차지하는 비율**이다. 1.0이 아닌
	// 근거는 값 라벨이 막대 위에 올라탄다는 것이다: 헤드룸이 없으면 가장 높은 막대의 라벨이
	// 플롯 밴드를 넘어 잘린다 (뷰어에서 실측했다).
	chartHeadroom = 0.9
	// chartBarWidth는 한 범주 칸에서 막대가 차지하는 폭이다. 칸 폭은 `grow="1" basis="0"`이
	// 균등 분배하므로 막대 사이 간격은 이 백분율의 나머지다 — 간격을 따로 세지 않는다.
	// 70%는 008이 손으로 쓰던 140/200과 같은 리듬이다.
	chartBarWidth = "70%"
	// chartSeriesFill은 **기본** 계열색이다 — 저자가 `series`를 쓰면 그 색이 이긴다.
	// 단일 계열이므로 색 하나면 충분하고, 계열에서 벗어난 한 점의 강조는 점의 `fill`이 갖는다.
	chartSeriesFill = "accent1"
	// chartBaselineFill·chartBaselineHeight는 범주 칸의 바닥선이다. 칸마다 1px 도형을 두고
	// `gap="0"`으로 붙이면 선이 이어져 축이 된다 — 축 도형 하나를 따로 놓으면 막대의 기준선과
	// 축선이 다른 상자에 살게 되어 두 타깃에서 어긋난다.
	chartBaselineFill   = "muted"
	chartBaselineHeight = "1"
	// chartLabelGap은 값 라벨과 막대 사이다 (공간 토큰 — 테마가 값을 갖는다).
	chartLabelGap = "xs"
	// chartLegendGap은 플롯과 범례 사이, chartLegendItemGap은 범례 항목 사이다 (공간 토큰).
	chartLegendGap     = "sm"
	chartLegendItemGap = "md"
	// chartFillAxis는 "축을 채운다"는 길이 표기다 (`w="fill"`) — 값 층의 lengthValue가 같은
	// 문자열을 판정한다 (core/layout.go). 합성 노드가 저자의 표기를 쓴다는 것이 펼침의 계약이다.
	chartFillAxis = "fill"
)

// resolveChart는 `<chart>`를 도형·텍스트의 하위 트리로 펼친다.
//
// 이 함수가 끝나면 elem은 **`<shape>`다** — 이름·속성·자식이 전부 바뀐다. 제자리에서 바꾸는
// 이유는 두 가지다: 저자의 Span이 그대로 남아야 뒤따르는 진단이 저자의 자리를 가리키고
// (뿌리 도형의 `alt` 진단이 그 자리다), 부모 슬라이스에 손을 뻗지 않아야 순회가 단순하다.
//
// 호출 자리는 r.node의 도형 갈래 **바로 앞**이다 (core/resolve.go): 배치·슬롯은 `<chart>`의
// 이름으로 이미 지났고(진단이 `<shape>`를 말하면 저자는 자기가 쓰지 않은 태그를 듣는다),
// 이름을 바꾼 뒤에는 같은 순회가 뿌리를 평범한 배경 도형으로 해석한다. 자식은 트리 순회가
// 데려간다 — 합성 노드도 저자가 쓴 것과 **같은 경로로** 해석된다는 것이 펼침의 계약이다.
//
// 검증이 실패해도 노드는 도형이 된다. 근거는 불변식이다: 확정 트리에 펼쳐지는 종류가 하나도
// 없어야 하고(TestResolvedTreeHasNoDesugaredKinds) 그 사실이 진단 유무에 달려 있으면 실패한
// 빌드의 트리를 보는 코드가 `<chart>`를 만난다. 에러가 있으면 빌드는 산출물을 내지 않으므로
// 껍데기 도형의 내용은 아무도 보지 않는다.
func (r *resolver) resolveChart(elem *soml.Element) {
	kind := r.chartType(elem)
	r.chartFrameHeight(elem)
	spec := chartSpec{unit: r.chartUnit(elem)}
	series, grouped, hasPoints := r.chartAllSeries(elem, kind)
	if kind != "" && kind != soml.ChartTypeBar {
		r.chartLineEmphasis(elem, kind)
	}
	var all []chartDatum
	for _, s := range series {
		all = append(all, s.data...)
	}
	axis, axisOK := r.chartAxis(elem, all)
	spec.axis = axis

	elem.Name = soml.ShapeElement
	elem.Authored = soml.ChartElement
	elem.Attrs = chartRootAttrs(elem)
	elem.Children = nil
	if !hasPoints || !axisOK || len(series) == 0 || len(series[0].data) == 0 {
		return
	}
	for _, s := range series {
		if len(s.data) != len(series[0].data) {
			return // 범주가 어긋났다 — 진단은 이미 냈다.
		}
	}
	var plot *soml.Element
	switch {
	case !grouped && (kind == soml.ChartTypeBar || kind == ""):
		spec.series = series[0].fill
		plot = chartPlot(series[0].data, spec)
	case kind == soml.ChartTypeBar || kind == "":
		plot = chartGroupedPlot(series, spec)
	default:
		plot = chartLinePlot(series, spec)
	}
	if !grouped {
		elem.Children = []soml.Node{plot}
		markChartAuthored(elem)
		return
	}
	// 계열 형식은 범례를 아래에 단다 — 플롯이 남은 높이를 먹고(grow) 범례는 자기 높이다.
	setChartAttr(plot, "h", "")
	plot.Attrs = append(plot.Attrs, chartAttr("grow", "1", plot.Span), chartAttr("basis", "0", plot.Span))
	body := chartNode(soml.ColElement, elem.NameSpan, "gap", chartLegendGap, "w", chartFillAxis, "h", chartFillAxis)
	body.Children = []soml.Node{plot, chartLegend(series)}
	elem.Children = []soml.Node{body}
	markChartAuthored(elem)
}

// markChartAuthored는 펼친 하위 트리에서 아직 출처가 없는 노드를 `<chart>`로 적는다
// (soml.Element.Authored). 한 점·한 계열에서 나온 노드 — 막대, 값·범주 라벨, 범례 항목 — 는 만드는
// 자리에서 `<point>`·`<series>`를 이미 받았다. 나머지(플롯·칸·기준선·라벨 줄)는 어느 한 점의 것이
// 아니라 차트의 틀이다.
func markChartAuthored(root *soml.Element) {
	for _, node := range root.Children {
		elem, ok := node.(*soml.Element)
		if !ok {
			continue
		}
		if elem.Authored == "" {
			elem.Authored = soml.ChartElement
		}
		markChartAuthored(elem)
	}
}

// chartSeries는 검증을 통과한 계열 하나다 — 단일 계열 차트(직계 점)도 이름 없는 계열 하나로 읽는다.
type chartSeries struct {
	name string
	// fill은 계열색 표기다 (저자의 것 그대로 또는 기본 accentN).
	fill string
	// kind는 이 계열의 종류다 — 선 계열 차트에서 계열이 `type`으로 바꿀 수 있다.
	kind string
	data []chartDatum
	span soml.Span
}

// chartSeriesDefaults는 계열색을 생략한 계열의 색이다 — 계열 순서대로 강조색 여섯을 돈다.
var chartSeriesDefaults = []string{"accent1", "accent2", "accent3", "accent4", "accent5", "accent6"}

// chartAllSeries는 차트의 계열들이다. grouped는 `<series>` 형식인지(범례 · 묶음 막대), hasPoints는 점을 하나라도
// 적었는지다 (chartData와 같은 뜻).
//
// 한 차트는 점 아니면 계열이다 — 섞으면 직계 점이 어느 계열인지 정할 수 없다. 범주는 첫 계열의 라벨이고, 다른 계열은
// 같은 순서로 같은 라벨을 적어야 한다 (어긋난 점을 진단이 가리킨다).
func (r *resolver) chartAllSeries(elem *soml.Element, kind string) ([]chartSeries, bool, bool) {
	var seriesElems []*soml.Element
	directPoints := 0
	for _, child := range elem.Children {
		e, ok := child.(*soml.Element)
		if !ok {
			continue
		}
		switch e.Name {
		case soml.ChartSeriesElement:
			seriesElems = append(seriesElems, e)
		case soml.ChartPointElement:
			directPoints++
		}
	}
	if len(seriesElems) == 0 {
		data, hasPoints := r.chartData(elem)
		return []chartSeries{{fill: r.chartSeries(elem), kind: kind, data: data, span: elem.NameSpan}}, false, hasPoints
	}
	if directPoints > 0 {
		r.errorf(elem.NameSpan, "<%s> holds either <%s>s or <%s>, not both — put every point inside a <%s>",
			soml.ChartElement, soml.ChartPointElement, soml.ChartSeriesElement, soml.ChartSeriesElement)
		return nil, true, false
	}
	if attr, ok := elem.Attr("series"); ok {
		r.errorf(attr.NameSpan, "<%s series> colors the only series — with <%s> each series takes its own fill",
			soml.ChartElement, soml.ChartSeriesElement)
	}
	var out []chartSeries
	hasPoints := true
	for i, se := range seriesElems {
		s := chartSeries{kind: kind, span: se.NameSpan, fill: chartSeriesDefaults[i%len(chartSeriesDefaults)]}
		if attr, ok := se.Attr("name"); ok && attr.HasValue && attr.Value != "" {
			s.name = attr.Value
		} else {
			r.errorf(se.NameSpan, "<%s> needs a name — it is the legend entry", soml.ChartSeriesElement)
		}
		if attr, ok := se.Attr("fill"); ok {
			if color, ok := r.chartColor(attr); ok {
				s.fill = color
			}
		}
		if attr, ok := se.Attr("type"); ok {
			s.kind = r.chartSeriesType(attr, kind)
		}
		data, ok := r.chartData(se)
		hasPoints = hasPoints && ok
		s.data = data
		out = append(out, s)
	}
	r.chartCategories(out, seriesElems)
	return out, true, hasPoints
}

// chartSeriesType은 계열의 종류다 — 선 계열 차트(line · area · step) 안에서만 셋 중 하나를 고른다. 막대는 칸을 나눠
// 서는 다른 배치라 선과 한 밴드에 섞지 않는다.
func (r *resolver) chartSeriesType(attr soml.Attr, kind string) string {
	if kind == soml.ChartTypeBar {
		r.errorf(attr.NameSpan, "<%s type> is for line, area and step charts — every series of a bar chart is bars", soml.ChartSeriesElement)
		return kind
	}
	switch attr.Value {
	case soml.ChartTypeLine, soml.ChartTypeArea, soml.ChartTypeStep:
		return attr.Value
	}
	r.errorf(attr.ValueSpan, "unknown series type %q (available series types: line, area, step)", attr.Value)
	return kind
}

// chartCategories는 모든 계열이 첫 계열과 같은 범주를 같은 순서로 적었는지 본다.
func (r *resolver) chartCategories(series []chartSeries, elems []*soml.Element) {
	if len(series) == 0 {
		return
	}
	first := series[0]
	for i, s := range series[1:] {
		if len(s.data) != len(first.data) {
			r.errorf(elems[i+1].NameSpan, "<%s name=%q> has %d points but the first series has %d — every series lists the same categories",
				soml.ChartSeriesElement, s.name, len(s.data), len(first.data))
			continue
		}
		for j, d := range s.data {
			if d.label != first.data[j].label {
				r.errorf(d.span, "<%s label=%q> is category %d — the first series has %q there (every series lists the same categories in the same order)",
					soml.ChartPointElement, d.label, j+1, first.data[j].label)
			}
		}
	}
}

// chartLineEmphasis는 선 · 영역 · 계단 차트의 점 강조를 막는다 — 강조색은 **막대 하나**를 칠하는 채널이고, 선은
// 계열 하나가 한 경로라 점 하나만 다른 색으로 칠할 자리가 없다.
func (r *resolver) chartLineEmphasis(elem *soml.Element, kind string) {
	var points []*soml.Element
	for _, child := range elem.Children {
		e, ok := child.(*soml.Element)
		if !ok {
			continue
		}
		switch e.Name {
		case soml.ChartPointElement:
			points = append(points, e)
		case soml.ChartSeriesElement:
			for _, c := range e.Children {
				if p, ok := c.(*soml.Element); ok && p.Name == soml.ChartPointElement {
					points = append(points, p)
				}
			}
		}
	}
	for _, point := range points {
		if attr, ok := point.Attr("fill"); ok {
			r.errorf(attr.NameSpan, "<%s fill> highlights one bar — a %s chart draws its series as one line, so a point cannot take its own color",
				soml.ChartPointElement, kind)
		}
	}
}

// chartSpec은 펼침이 **데이터 밖에서** 쓰는 값 셋이다 — 축·단위·계열색. 점마다 같은 값이므로
// 트리 빌더에 인자 셋으로 흘리지 않고 하나로 묶는다 (계열이 여럿이 되는 날 이 구조체가 자란다).
type chartSpec struct {
	// axis는 값축의 최대다 — 막대 높이의 분모.
	axis float64
	// unit은 값 라벨의 꼬리다 ("22" + "분").
	unit string
	// series는 모든 막대의 색이다 — 점의 강조색이 있으면 그 점만 이 색에서 벗어난다.
	series string
}

// chartRootAttrs는 뿌리 도형의 속성이다 — 저자의 박스·배치·슬롯·`alt`을 그대로 물려받고
// 차트 고유 속성(`type`·`unit`·`max`·`series`)은 소비되어 사라진다.
//
// 뿌리가 **배경**인 것이 이 표의 결정이다 (`fill="none" border="none"`): 차트가
// 상자에 담기면 저자가 지울 수 없는 테두리가 생긴다. 배경을 원하면 저자가 `<shape>`로 감싼다.
//
// `alt`을 물려받는 자리가 여기인 근거: 이름이 붙을 자리가 뿌리 도형에 이미
// 있고(`p:sp`의 `descr`·`<svg aria-label>`), 값 검사도 도형의 것을 그대로 쓴다 (altName).
// 값이 비었을 때의 진단이 저자의 `alt=""`을 가리키는 것은 Attr을 복사하지 않고 **옮기기**
// 때문이다.
func chartRootAttrs(elem *soml.Element) []soml.Attr {
	span := elem.NameSpan
	attrs := []soml.Attr{
		chartAttr("type", chartFrameType, span),
		chartAttr("fill", noneValue, span),
		chartAttr("border", noneValue, span),
	}
	for _, attr := range elem.Attrs {
		switch attr.Name {
		case "type", "unit", "max", "series":
			continue // 차트가 소비한다 — 도형의 어휘에는 없는 이름이다.
		}
		attrs = append(attrs, attr)
	}
	return attrs
}

// chartType은 차트 종류를 검증한다. 값을 돌려주지 않는 이유는 v1의 종류가 하나라는 것이다 —
// 종류가 둘이 되는 커밋이 펼침을 분기하며 반환값을 만든다 (지금 반환하면 쓰지 않는 값이 된다).
//
// **필수**인 근거: `<chart/>`가 조용히 막대가 되면 저자의 오타(`tpye="bar"`)와 구별되지 않는다.
// 값 없는 `type`도 같은 문구로 말한다 — hasValue의 일반 문구("값이 필요하다")는 어느 종류를
// 쓸 수 있는지 말하지 않으므로 저자가 문서를 찾아야 한다.
func (r *resolver) chartType(elem *soml.Element) string {
	attr, ok := elem.Attr("type")
	if !ok || !attr.HasValue || attr.Value == "" {
		span := elem.NameSpan
		if ok {
			span = attr.NameSpan
		}
		r.errorf(span, "<%s> needs a type (available types: %s) — without a type there is no way to guess what to draw",
			soml.ChartElement, chartTypeList())
		return ""
	}
	for _, name := range soml.ChartTypes() {
		if name == attr.Value {
			return name
		}
	}
	r.errorf(attr.ValueSpan, "unknown chart type %q%s (available types: %s)", attr.Value, didYouMean(attr.Value, chartTypeList()), chartTypeList())
	return ""
}

// chartFrameHeight는 차트가 **자기 높이를 갖는지** 본다 — `h`나 `ratio` 중 하나가 있어야 한다.
//
// 근거는 기준의 순환이다: 막대 높이는 플롯 밴드에 대한 백분율이므로 밴드가 먼저 정해져야
// 하는데, 내용이 높이를 정하는 차트에서는 밴드가 막대에 달리게 된다. 백분율 인셋 프리셋이
// 확정 높이를 요구하는 것과 같은 종류의 순환이다 (shapeInsetTextHeight).
//
// `h="fill"`도 높이다 (부모가 준다) — 값의 형태를 보지 않고 **있음**만 보는 이유가 그것이다.
func (r *resolver) chartFrameHeight(elem *soml.Element) bool {
	for _, name := range []string{"h", "ratio"} {
		if attr, ok := elem.Attr(name); ok && attr.HasValue && attr.Value != "" {
			return true
		}
	}
	r.errorf(elem.NameSpan,
		"<%s> needs h or ratio — bar heights are percentages of the axis, so letting the content set the height would be circular",
		soml.ChartElement)
	return false
}

// chartUnit은 값 라벨의 꼬리다 (`"22"` + `"분"`). 값이 아니라 **표기**이므로 축 계산에 들어가지
// 않는다 — `value="22분"`을 막는 진단이 저자를 이 속성으로 보낸다 (chartValue).
func (r *resolver) chartUnit(elem *soml.Element) string {
	attr, ok := elem.Attr("unit")
	if !ok {
		return ""
	}
	if !r.hasValue(attr) {
		return ""
	}
	return attr.Value
}

// chartSeries는 계열색이다 — **모든 막대**의 색이고, 생략하면 기본 계열색이다.
//
// 이 속성이 생긴 자리는 리뷰다: 008은 `surface` 막대와 `muted` 라벨을 함께 요구하고, 점의
// `fill` 하나로는 그 그림을 쓸 수 없었다 (그 채널은 막대와 라벨을 함께 칠한다). 그래서 계열색은
// 프레임이 갖고 **라벨에 오지 않는다** — 면의 색과 글의 색은 대비 요구가 다르다 (`surface`
// 글자는 흰 배경에서 1.06:1이다).
//
// 이름이 `fill`이 아닌 근거도 있다: 뿌리 도형은 계약상 배경이므로(`fill="none"`)
// 같은 이름을 쓰면 저자가 막대를 칠하려다 회색 상자를 받는다.
func (r *resolver) chartSeries(elem *soml.Element) string {
	attr, ok := elem.Attr("series")
	if !ok {
		return chartSeriesFill
	}
	if color, ok := r.chartColor(attr); ok {
		return color
	}
	return chartSeriesFill // 진단은 이미 냈다 — 껍데기 트리의 색은 아무도 보지 않는다.
}

// chartColor는 차트의 채움 값 하나를 검증하고 **저자의 표기 그대로** 돌려준다 (`series`와
// `<point fill>`이 공유한다). 문자열로 도는 이유는 펼침의 계약이다: 합성 노드가 저자가 쓸 수
// 있는 표기를 그대로 실어야 손으로 쓴 덱과 같은 트리가 된다.
//
// 도형의 `fill`과 갈라지는 자리가 하나다 — **`none`을 받지 않는다**.
// 근거는 둘이다: 그리지 않는 막대는 데이터가 아니라 빠뜨린 점이고(높이 0을 원하면 `value="0"`이
// 그 표기다), 이 값은 라벨의 `color`로도 흐르므로 `none`을 통과시키면 저자가 쓰지 않은 이름으로
// 진단이 난다 — 리뷰에서 실측한 모양이 그것이었다 ("color의 색 토큰 none이 테마에 없다" 둘).
func (r *resolver) chartColor(attr soml.Attr) (string, bool) {
	if !r.hasValue(attr) {
		return "", false
	}
	if attr.Value == noneValue {
		r.errorf(attr.ValueSpan, "%s cannot be %s — a bar that is not drawn is not data (an empty bar is value=\"0\")",
			attr.Name, noneValue)
		return "", false
	}
	if _, ok := r.colorValue(attr.Name, attr.Value, attr.ValueSpan); !ok {
		return "", false
	}
	return attr.Value, true
}

// chartDatum은 검증을 통과한 점 하나다. `raw`를 함께 드는 이유는 라벨이 **저자가 적은 표기
// 그대로**여야 한다는 것이다 — `19.5`를 다시 포맷하면 소스와 슬라이드가 갈라진다.
type chartDatum struct {
	label string
	value float64
	raw   string
	// fill은 이 점의 강조색 토큰이다 (없으면 빈 문자열 — 계열색을 쓴다).
	fill string
	// span은 저자의 `<point>` 구간이다. 이 점에서 나온 합성 노드가 물려받는다.
	span soml.Span
}

// chartData는 점들을 읽는다. 두 번째 반환값은 **점을 하나라도 적었는지**다 — 검증에 실패한
// 점이 있는 것과 아예 없는 것을 갈라야 "점이 없다"를 두 번 말하지 않는다 (값이 하나뿐이고
// 그것이 음수인 차트는 진단이 하나여야 한다).
func (r *resolver) chartData(elem *soml.Element) ([]chartDatum, bool) {
	owner := soml.ChartElement
	if elem.Name == soml.ChartSeriesElement {
		owner = soml.ChartSeriesElement
	}
	var data []chartDatum
	written := 0
	for _, child := range elem.Children {
		point, ok := child.(*soml.Element)
		if !ok || point.Name != soml.ChartPointElement {
			continue
		}
		written++
		if datum, ok := r.chartPoint(point); ok {
			data = append(data, datum)
		}
	}
	if written == 0 {
		r.errorf(elem.NameSpan, "<%s> has no <%s> — there is no data to draw (<%s label=\"May\" value=\"22\"/>)",
			owner, soml.ChartPointElement, soml.ChartPointElement)
		return nil, false
	}
	return data, true
}

// chartPoint는 점 하나를 검증한다 — 라벨·값은 필수이고 강조색은 계열색과 같은 어휘다
// (chartColor).
//
// 색을 여기서 **한 번** 검증하는 이유는 진단의 수다: 같은 값이 막대와 두 라벨에 실리므로
// 합성 노드에 맡기면 오타 하나가 진단 셋이 된다. 실패하면 색을 버린다 (계열색으로 그려진다 —
// 어차피 빌드는 멈춘다).
func (r *resolver) chartPoint(point *soml.Element) (chartDatum, bool) {
	datum := chartDatum{span: point.Span}
	label, labelOK := r.chartLabel(point)
	datum.label = label
	value, raw, valueOK := r.chartValue(point)
	datum.value, datum.raw = value, raw
	if attr, ok := point.Attr("fill"); ok {
		if color, fillOK := r.chartColor(attr); fillOK {
			datum.fill = color
		}
	}
	return datum, labelOK && valueOK
}

// chartLabel은 점의 범주 라벨이다 — 막대가 **무엇의** 값인지다.
func (r *resolver) chartLabel(point *soml.Element) (string, bool) {
	attr, ok := point.Attr("label")
	if !ok || !attr.HasValue || attr.Value == "" {
		span := point.NameSpan
		if ok {
			span = attr.NameSpan
		}
		r.errorf(span, "<%s> needs a label — without saying what a bar measures, the chart cannot be read",
			soml.ChartPointElement)
		return "", false
	}
	return attr.Value, true
}

// chartValue는 점의 값이다 — 막대 높이가 이 수치다.
//
// 단위 접미사를 **받지 않는다** (`value="22분"`은 에러다): 표기를 값에서 떼어 `unit`에 둔 것이
// 펼침의 결정이고, 값마다 단위를 적으면 축 계산이 저자가 적은 문자열에 달린다. 진단이
// 고칠 자리를 말한다.
//
// 음수도 받지 않는다 — 막대 차트의 축은 0에서 시작하고, 음수 막대는 축을 0 아래로 내리는
// 다른 그림이다 (v1에 없다).
func (r *resolver) chartValue(point *soml.Element) (float64, string, bool) {
	attr, ok := point.Attr("value")
	if !ok || !attr.HasValue {
		span := point.NameSpan
		if ok {
			span = attr.NameSpan
		}
		r.errorf(span, "<%s> needs a value — it is the bar height", soml.ChartPointElement)
		return 0, "", false
	}
	value, ok := parseNumber(attr.Value)
	if !ok {
		r.errorf(attr.ValueSpan, "value is not a number (%q) — put the unit in <%s unit> (value=\"22\" unit=\"min\")",
			attr.Value, soml.ChartElement)
		return 0, "", false
	}
	if value < 0 {
		r.errorf(attr.ValueSpan, "value cannot be negative (%q) — a bar chart axis starts at 0", attr.Value)
		return 0, "", false
	}
	return value, attr.Value, true
}

// chartAxis는 값축의 최대다 — 막대 높이의 분모이고, 이 함수가 차트의 **유일한 산수**다.
//
// 저자가 `max`를 적으면 그 축이고, 적지 않으면 데이터 최대 ÷ 헤드룸이다. 자동 축이 데이터
// 최대와 같지 않은 이유는 값 라벨이 막대 위에 올라탄다는 것이다 (chartHeadroom).
//
// `max`를 넘는 값은 에러다: 막대를 잘라 그리면 그림이 데이터를 속이고, 100%를 넘겨 그리면
// 박스를 넘는다 — 어느 쪽도 저자가 의도한 그림이 아니므로 고르지 않고 묻는다.
func (r *resolver) chartAxis(elem *soml.Element, data []chartDatum) (float64, bool) {
	dataMax := 0.0
	for _, datum := range data {
		dataMax = math.Max(dataMax, datum.value)
	}
	attr, ok := elem.Attr("max")
	if !ok {
		if dataMax == 0 {
			if len(data) == 0 {
				return 0, false // 점이 없다 — 진단은 chartData가 이미 냈다.
			}
			r.errorf(elem.NameSpan, "all values of <%s> are 0 — the axis cannot be determined (set max to draw an empty chart)",
				soml.ChartElement)
			return 0, false
		}
		return dataMax / chartHeadroom, true
	}
	if !r.hasValue(attr) {
		return 0, false
	}
	axis, ok := parseNumber(attr.Value)
	if !ok {
		r.errorf(attr.ValueSpan, "max is not a number (%q) — write the value axis maximum as a number", attr.Value)
		return 0, false
	}
	if axis <= 0 {
		r.errorf(attr.ValueSpan, "max must be greater than 0 (%q) — a zero axis leaves bar heights without a denominator", attr.Value)
		return 0, false
	}
	for _, datum := range data {
		if datum.value > axis {
			r.errorf(attr.ValueSpan, "value %s is greater than max %s — the bar exceeds the axis (raise or remove max)",
				datum.raw, attr.Value)
			return 0, false
		}
	}
	return axis, true
}

// chartPlot은 막대 차트의 본체다 — 위에서 아래로 막대 밴드(범주 칸이 나란히 선 `<row>`)와
// 범주 라벨의 줄이다.
//
// 칸이 `grow="1" basis="0"`인 이유는 균등 분배다 (막대 폭·간격을 픽셀로 세지 않는다).
// 범주 라벨을 칸 안이 아니라 **따로 한 줄로** 두는 이유: 칸 안에 두면 두 줄로 접힌 라벨 하나가
// 그 칸의 밴드만 줄여서 그 막대가 이웃보다 짧게 그려진다 — 그림이 데이터를 속인다. 한 줄로
// 모으면 줄의 높이는 가장 높은 라벨이 정하고 모든 칸의 밴드가 같은 높이를 갖는다. 라벨 줄의
// 칸도 `grow="1" basis="0"`에 최소 폭 0이라 막대 칸과 같은 열에 선다 (chartCellMinWidth).
func chartPlot(data []chartDatum, spec chartSpec) *soml.Element {
	row := chartBandRow(data[0].span)
	for _, datum := range data {
		row.Children = append(row.Children, chartColumn(datum.span, chartBand(datum, spec, chartBarWidth)))
	}
	return chartBody(row, data)
}

// chartBandRow는 범주 칸이 나란히 서는 줄이다 — 라벨 줄을 뺀 높이를 전부 먹는다.
func chartBandRow(span soml.Span) *soml.Element {
	return chartNode(soml.RowElement, span,
		"gap", "0",
		"align", string(soml.BoxAlignStretch),
		"grow", "1",
		"basis", "0",
		"w", chartFillAxis)
}

// chartColumn은 범주 한 칸이다 — 위에서 아래로 플롯 밴드와 기준선이다.
//
// 기준선이 칸마다 있는 이유는 chartBaselineFill의 주석에 있다 (`gap="0"`으로 이어 붙는다).
func chartColumn(span soml.Span, band *soml.Element) *soml.Element {
	col := chartNode(soml.ColElement, span,
		"grow", "1",
		"basis", "0",
		"min-w", chartCellMinWidth,
		"gap", "0",
		"align", string(soml.BoxAlignCenter))
	col.Children = []soml.Node{
		band,
		chartNode(soml.ShapeElement, span,
			"type", chartFrameType,
			"fill", chartBaselineFill,
			"border", noneValue,
			"w", chartFillAxis,
			"h", chartBaselineHeight),
	}
	return col
}

// chartBody는 막대 밴드 줄 아래에 범주 라벨 줄을 단다. 라벨은 점의 강조색을 따른다.
func chartBody(bands *soml.Element, categories []chartDatum) *soml.Element {
	labels := chartNode(soml.RowElement, categories[0].span, "gap", "0", "w", chartFillAxis)
	for _, datum := range categories {
		labels.Children = append(labels.Children, chartCategoryLabel(datum, datum.fill))
	}
	body := chartNode(soml.ColElement, categories[0].span, "gap", "0", "w", chartFillAxis, "h", chartFillAxis)
	body.Children = []soml.Node{bands, labels}
	return body
}

// chartCellMinWidth는 범주 칸(막대 칸 · 라벨 칸 · 묶음 막대의 계열 밴드)의 최소 폭이다.
//
// 칸은 `grow="1" basis="0"`인 flex 항목이라 균등 분배가 되어야 하는데, flex 항목의 최소 폭은 기본이
// `auto` — 내용의 min-content 폭이다. 그래서 칸보다 넓은 낱말 하나(`Internationalization`)가 자기 칸을
// 넓히고 이웃 칸들을 밀어, 라벨 줄과 막대 밴드가 서로 다른 열로 나뉜다. 최소 폭 0이면 칸은 언제나
// 줄을 n등분한 폭이고, 넓은 라벨은 자기 칸 안에서 넘친다 — 그 넘침은 그 `<point>`의 글자 넘침
// 경고가 된다.
const chartCellMinWidth = "0"

// chartCategoryLabel은 범주 라벨 줄의 칸 하나다 — 막대 칸과 같은 `grow="1" basis="0"`이고 최소 폭이
// 0이라 같은 열에 선다 (chartCellMinWidth).
func chartCategoryLabel(datum chartDatum, color string) *soml.Element {
	label := chartTextNode(datum.label, color, datum.span)
	label.Attrs = append(label.Attrs,
		chartAttr("grow", "1", datum.span),
		chartAttr("basis", "0", datum.span),
		chartAttr("min-w", chartCellMinWidth, datum.span))
	return label
}

// chartBand는 한 칸의 플롯 밴드다 — 위에서 아래로 여백·값 라벨·막대가 쌓인다.
//
// 막대 높이를 `h="p%"`로 적지 않는 이유: 백분율은 **밴드 전체**에 대한 비율이라 라벨 한 줄이
// 들어갈 자리가 남지 않는다. 낮은 차트에서는 가장 높은 막대의 라벨이 밴드 위로 넘친다 (헤드룸
// 10%가 캡션 한 줄보다 작다). 대신 여백과 막대가 `basis`로 밴드의 (100−p)%와 p%를 잡고, 넘치는
// 만큼(라벨 + 간격)을 flex 축소가 **basis에 비례해** 둘에게서 거둔다. 그래서 막대는 언제나
// "밴드 − 라벨 − 간격"의 p%가 되고 — 칸마다 라벨 높이가 같으므로 막대끼리의 비율은 데이터
// 그대로다 — 라벨은 밴드 안에 남는다. 라벨은 축소되지 않는다 (`shrink="0"`).
//
// 값 0은 막대를 내지 않는다: 높이 0의 도형은 산출물에서 아무것도 아니므로 선언을 낼 이유가
// 없다. 라벨은 남는다 — "0"이 데이터다. 여백이 100%를 잡아 라벨을 기준선에 내려놓는다.
func chartBand(datum chartDatum, spec chartSpec, barWidth string) *soml.Element {
	band := chartNode(soml.ColElement, datum.span,
		"grow", "1",
		"basis", "0",
		"min-w", chartCellMinWidth,
		"w", chartFillAxis,
		"align", string(soml.BoxAlignCenter),
		"gap", chartLabelGap)
	percent := 0.0
	if datum.value != 0 {
		percent = math.Round(datum.value/spec.axis*10000) / 100
	}
	label := chartTextNode(datum.raw+spec.unit, datum.fill, datum.span)
	label.Attrs = append(label.Attrs, chartAttr("shrink", "0", datum.span))
	band.Children = []soml.Node{
		chartNode(soml.SpacerElement, datum.span, "grow", "0", "basis", chartPercent(100-percent)),
		label,
	}
	if datum.value == 0 {
		return band
	}
	// 계열색이 바탕이고 점의 강조색이 그 점만 이긴다.
	fill := spec.series
	if datum.fill != "" {
		fill = datum.fill
	}
	bar := chartNode(soml.ShapeElement, datum.span,
		"type", chartFrameType,
		"fill", fill,
		"border", noneValue,
		"w", barWidth,
		"basis", chartPercent(percent))
	bar.Authored = soml.ChartPointElement
	band.Children = append(band.Children, bar)
	return band
}

// chartTextNode는 라벨 하나다 — 캡션 역할이고, 점의 강조색이 있으면 그 색을 쓴다.
//
// 역할이 캡션인 이유: 라벨은 그림을 읽는 보조 문구이고 테마가 그 크기·색을 이미 갖는다
// (차트 전용 타이포그래피를 만들면 테마가 두 벌을 들게 된다). `w="fill"`은 칸 폭을 채워
// `align="center"`가 칸 가운데를 뜻하게 한다 — 텍스트 자신의 폭 가운데가 아니다.
func chartTextNode(text, color string, span soml.Span) *soml.Element {
	pairs := []string{
		"role", string(soml.RoleCaption),
		"align", string(soml.AlignCenter),
		"w", chartFillAxis,
	}
	if color != "" {
		pairs = append(pairs, "color", color)
	}
	label := chartNode(soml.TextElement, span, pairs...)
	label.Authored = soml.ChartPointElement
	label.Children = []soml.Node{&soml.Text{Value: text, Span: span}}
	return label
}

// chartNode는 합성 요소 하나다 — 속성은 이름·값 쌍으로 받는다.
//
// 구간을 저자의 것으로 채우는 이유: 합성 노드가 진단을 낼 일은 없어야 하지만(값은 우리가
// 만들었다) 만약 낸다면 저자가 고칠 수 있는 자리를 가리켜야 한다. 0 구간은 `0:0`으로 찍혀
// 아무 곳도 아니다.
func chartNode(name string, span soml.Span, pairs ...string) *soml.Element {
	elem := &soml.Element{Name: name, NameSpan: span, Span: span}
	for i := 0; i+1 < len(pairs); i += 2 {
		elem.Attrs = append(elem.Attrs, chartAttr(pairs[i], pairs[i+1], span))
	}
	return elem
}

// chartAttr는 합성 속성 하나다.
func chartAttr(name, value string, span soml.Span) soml.Attr {
	return soml.Attr{Name: name, NameSpan: span, Value: value, ValueSpan: span, HasValue: true}
}

// chartPercent는 백분율 표기다 — **소수 둘째 자리까지**다.
//
// 자리를 자르는 이유는 두 타깃의 반올림이 아니라 읽을 수 있는 소스다: 펼친 트리는 저자가
// 손으로 쓸 수 있는 SOML이어야 하고(등가 시험이 그 표기를 그대로 적는다) `77.72727272727273%`는
// 사람이 쓰지 않는 표기다. 0.01pp의 차이는 1000px 높이에서 0.1px이다.
func chartPercent(percent float64) string {
	return strconv.FormatFloat(math.Round(percent*100)/100, 'f', -1, 64) + "%"
}

// 선 · 영역 · 계단 차트의 펼침 상수다.
const (
	// chartLineWidth는 계열 선의 굵기(px)다 — 막대 사이의 기준선(1px)보다 굵어야 데이터가 축으로 읽히지 않는다.
	chartLineWidth = "3"
	// chartAreaAlpha는 영역 차트의 칠 불투명도다 — 선 아래의 면은 선보다 물러나야 선이 읽힌다.
	chartAreaAlpha = "0.2"
)

// chartLinePlot은 선 · 영역 · 계단 차트의 밴드다 — 위에서 아래로 플롯 밴드(경로 도형 하나) · 기준선 · 범주 라벨의 줄.
//
// 점의 가로 자리는 범주 칸의 **가운데**다 (i + ½) / n — 막대 차트와 같은 칸 나눔이라 범주 라벨이 같은 열에 선다.
// 세로 자리는 막대 높이와 같은 비율(값 ÷ 축)을 밴드의 아래에서 위로 잰 것이다. 그래서 산수는 백분율뿐이고, 경로
// 도형이 그 0~100 상자를 밴드 크기로 늘린다 — 픽셀은 여전히 브라우저가 잰다. 계단은 칸마다 수평인 한 단이다.
// 영역은 같은 선 아래를 기준선까지 닫아 계열색 20%로 칠하고, 그 위에 선을 다시 긋는다 (`<stack>`으로 겹친다).
func chartLinePlot(series []chartSeries, spec chartSpec) *soml.Element {
	data := series[0].data
	span := data[0].span
	n := float64(len(data))
	x := func(i int) float64 { return (float64(i) + 0.5) / n * 100 }
	y := func(v float64) float64 { return 100 - v/spec.axis*100 }
	num := func(v float64) string { return strconv.FormatFloat(math.Round(v*100)/100, 'f', -1, 64) }
	path := func(kind string, data []chartDatum) string {
		var line strings.Builder
		for i, datum := range data {
			switch {
			case kind == soml.ChartTypeStep:
				left, right := float64(i)/n*100, float64(i+1)/n*100
				if i == 0 {
					line.WriteString("M" + num(left) + "," + num(y(datum.value)))
				} else {
					line.WriteString(" V" + num(y(datum.value)))
				}
				line.WriteString(" H" + num(right))
			case i == 0:
				line.WriteString("M" + num(x(i)) + "," + num(y(datum.value)))
			default:
				line.WriteString(" L" + num(x(i)) + "," + num(y(datum.value)))
			}
		}
		return line.String()
	}
	band := chartNode(soml.StackElement, span, "grow", "1", "basis", "0", "w", chartFillAxis)
	// 영역의 면을 먼저 다 칠하고 선을 그 위에 긋는다 — 뒤 계열의 면이 앞 계열의 선을 덮지 않는다.
	for _, s := range series {
		if s.kind != soml.ChartTypeArea {
			continue
		}
		area := path(s.kind, s.data) + " L" + num(x(len(s.data)-1)) + ",100 L" + num(x(0)) + ",100 Z"
		band.Children = append(band.Children, chartNode(soml.ShapeElement, s.data[0].span,
			"type", soml.PathShapeType, "d", area,
			"fill", "alpha("+s.fill+", "+chartAreaAlpha+")", "border", noneValue,
			"w", chartFillAxis, "h", chartFillAxis))
	}
	for _, s := range series {
		band.Children = append(band.Children, chartNode(soml.ShapeElement, s.data[0].span,
			"type", soml.PathShapeType, "d", path(s.kind, s.data),
			"fill", noneValue, "border", chartLineWidth+" solid "+s.fill,
			"w", chartFillAxis, "h", chartFillAxis))
	}

	labels := chartNode(soml.RowElement, span, "gap", "0", "w", chartFillAxis)
	for _, datum := range data {
		labels.Children = append(labels.Children, chartCategoryLabel(datum, ""))
	}
	col := chartNode(soml.ColElement, span, "gap", "0", "w", chartFillAxis, "h", chartFillAxis)
	col.Children = []soml.Node{
		band,
		chartNode(soml.ShapeElement, span,
			"type", chartFrameType, "fill", chartBaselineFill, "border", noneValue,
			"w", chartFillAxis, "h", chartBaselineHeight),
		labels,
	}
	return col
}

// chartGroupedPlot은 묶음 막대 밴드다 — 범주 칸마다 계열 수만큼의 막대가 칸의 70% 폭을 나눠 붙어 선다. 칸 · 기준선 ·
// 범주 라벨은 단일 계열의 칸과 같고, 플롯 밴드만 계열마다의 작은 밴드로 나뉜다 (막대 높이는 그 작은 밴드에 대한 비율 —
// 작은 밴드가 칸의 밴드 높이를 그대로 갖는다).
func chartGroupedPlot(series []chartSeries, spec chartSpec) *soml.Element {
	first := series[0].data
	row := chartBandRow(first[0].span)
	for i, category := range first {
		group := chartNode(soml.RowElement, category.span,
			"grow", "1", "basis", "0", "min-w", chartCellMinWidth, "w", chartBarWidth, "gap", "0",
			"align", string(soml.BoxAlignStretch))
		for _, s := range series {
			datum := s.data[i]
			group.Children = append(group.Children, chartBand(datum, chartSpec{axis: spec.axis, unit: spec.unit, series: s.fill}, chartFillAxis))
		}
		row.Children = append(row.Children, chartColumn(category.span, group))
	}
	// 범주 라벨은 계열의 색을 따르지 않는다 — 한 범주가 여러 계열에 걸친다.
	categories := make([]chartDatum, len(first))
	for i, category := range first {
		categories[i] = chartDatum{label: category.label, span: category.span}
	}
	return chartBody(row, categories)
}

// chartLegend는 계열 형식의 범례다 — 계열마다 색 견본과 이름이 가운데 정렬된 한 줄이다. 견본은 막대 · 영역이면 사각형,
// 선 · 계단이면 굵은 선이다 (그림에서 그 계열이 보이는 모양).
func chartLegend(series []chartSeries) *soml.Element {
	legend := chartNode(soml.RowElement, series[0].span,
		"gap", chartLegendItemGap,
		"justify", string(soml.BoxJustifyCenter),
		"align", string(soml.BoxAlignCenter),
		"w", chartFillAxis)
	for _, s := range series {
		w, h := "12", "12"
		if s.kind == soml.ChartTypeLine || s.kind == soml.ChartTypeStep {
			w, h = "18", chartLineWidth
		}
		item := chartNode(soml.RowElement, s.span, "gap", chartLabelGap, "align", string(soml.BoxAlignCenter))
		name := chartNode(soml.TextElement, s.span, "role", string(soml.RoleCaption))
		name.Children = []soml.Node{&soml.Text{Value: s.name, Span: s.span}}
		swatch := chartNode(soml.ShapeElement, s.span, "type", chartFrameType, "fill", s.fill, "border", noneValue, "w", w, "h", h)
		item.Children = []soml.Node{swatch, name}
		for _, e := range item.Children {
			e.(*soml.Element).Authored = soml.ChartSeriesElement
		}
		item.Authored = soml.ChartSeriesElement
		legend.Children = append(legend.Children, item)
	}
	return legend
}

// setChartAttr는 합성 노드의 속성 하나를 바꾼다 — 빈 값이면 지운다.
func setChartAttr(elem *soml.Element, name, value string) {
	attrs := elem.Attrs[:0]
	for _, a := range elem.Attrs {
		if a.Name != name {
			attrs = append(attrs, a)
		}
	}
	elem.Attrs = attrs
	if value != "" {
		elem.Attrs = append(elem.Attrs, chartAttr(name, value, elem.Span))
	}
}

// chartTypeList는 진단에 실을 종류 목록이다 (어휘 순서 — soml.ChartTypes).
func chartTypeList() string {
	return strings.Join(soml.ChartTypes(), ", ")
}
