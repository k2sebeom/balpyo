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
	r.chartType(elem)
	r.chartFrameHeight(elem)
	spec := chartSpec{unit: r.chartUnit(elem), series: r.chartSeries(elem)}
	data, hasPoints := r.chartData(elem)
	axis, axisOK := r.chartAxis(elem, data)
	spec.axis = axis

	elem.Name = soml.ShapeElement
	elem.Attrs = chartRootAttrs(elem)
	elem.Children = nil
	if hasPoints && axisOK && len(data) > 0 {
		elem.Children = []soml.Node{chartPlot(data, spec)}
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
func (r *resolver) chartType(elem *soml.Element) bool {
	attr, ok := elem.Attr("type")
	if !ok || !attr.HasValue || attr.Value == "" {
		span := elem.NameSpan
		if ok {
			span = attr.NameSpan
		}
		r.errorf(span, "<%s> needs a type (available types: %s) — without a type there is no way to guess what to draw",
			soml.ChartElement, chartTypeList())
		return false
	}
	for _, name := range soml.ChartTypes() {
		if name == attr.Value {
			return true
		}
	}
	r.errorf(attr.ValueSpan, "unknown chart type %q%s (available types: %s)", attr.Value, didYouMean(attr.Value, chartTypeList()), chartTypeList())
	return false
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
			soml.ChartElement, soml.ChartPointElement, soml.ChartPointElement)
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
	value, err := strconv.ParseFloat(attr.Value, 64)
	if err != nil {
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
	axis, err := strconv.ParseFloat(attr.Value, 64)
	if err != nil {
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

// chartPlot은 막대 밴드다 — 범주 칸이 나란히 선 `<row>` 하나.
//
// 칸이 `grow="1" basis="0"`인 이유는 균등 분배다 (막대 폭·간격을 픽셀로 세지 않는다).
// `align="stretch" h="fill"`이 칸을 프레임 높이까지 늘리고, 그 안에서 플롯 밴드가 `grow="1"`로
// 남은 높이를 전부 먹는다 — 그래서 막대의 백분율 높이가 **밴드**에 대한 비율이 되고 범주
// 라벨·기준선 높이는 거기서 빠진다.
func chartPlot(data []chartDatum, spec chartSpec) *soml.Element {
	row := chartNode(soml.RowElement, data[0].span,
		"gap", "0",
		"align", string(soml.BoxAlignStretch),
		"h", chartFillAxis)
	for _, datum := range data {
		row.Children = append(row.Children, chartColumn(datum, spec))
	}
	return row
}

// chartColumn은 범주 한 칸이다 — 위에서 아래로 플롯 밴드·기준선·범주 라벨이다.
//
// 기준선이 칸마다 있는 이유는 chartBaselineFill의 주석에 있다 (`gap="0"`으로 이어 붙는다).
// 읽기 순서가 값 → 범주인 것은 문서 순서가 곧 읽기 순서라는 계약을 따른다:
// 스크린 리더는 "22분, 5월"로 읽는다.
func chartColumn(datum chartDatum, spec chartSpec) *soml.Element {
	col := chartNode(soml.ColElement, datum.span,
		"grow", "1",
		"basis", "0",
		"gap", "0",
		"align", string(soml.BoxAlignCenter),
		"h", chartFillAxis)
	col.Children = []soml.Node{
		chartBand(datum, spec),
		chartNode(soml.ShapeElement, datum.span,
			"type", chartFrameType,
			"fill", chartBaselineFill,
			"border", noneValue,
			"w", chartFillAxis,
			"h", chartBaselineHeight),
		chartTextNode(datum.label, datum.fill, datum.span),
	}
	return col
}

// chartBand는 한 칸의 플롯 밴드다 — 값 라벨과 막대가 **바닥에 정렬되어** 쌓인다
// (`justify="end"`). 막대의 `shrink="0"`이 없으면 브라우저의 flex 축소가 100%에 가까운 막대를
// 줄여 버린다 (뷰어에서 실측했다).
//
// 값 0은 막대를 내지 않는다: 높이 0의 도형은 산출물에서 아무것도 아니므로 선언을 낼 이유가
// 없다. 라벨은 남는다 — "0"이 데이터다.
func chartBand(datum chartDatum, spec chartSpec) *soml.Element {
	band := chartNode(soml.ColElement, datum.span,
		"grow", "1",
		"basis", "0",
		"w", chartFillAxis,
		"justify", string(soml.BoxJustifyEnd),
		"align", string(soml.BoxAlignCenter),
		"gap", chartLabelGap)
	band.Children = []soml.Node{chartTextNode(datum.raw+spec.unit, datum.fill, datum.span)}
	if datum.value == 0 {
		return band
	}
	// 계열색이 바탕이고 점의 강조색이 그 점만 이긴다.
	fill := spec.series
	if datum.fill != "" {
		fill = datum.fill
	}
	band.Children = append(band.Children, chartNode(soml.ShapeElement, datum.span,
		"type", chartFrameType,
		"fill", fill,
		"border", noneValue,
		"w", chartBarWidth,
		"h", chartPercent(datum.value/spec.axis*100),
		"shrink", "0"))
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

// chartTypeList는 진단에 실을 종류 목록이다 (어휘 순서 — soml.ChartTypes).
func chartTypeList() string {
	return strings.Join(soml.ChartTypes(), ", ")
}
