package soml

// 이 파일은 차트의 **요소 종류·속성 어휘 카탈로그**다.
//
// 다른 카탈로그(subset.go·layout.go·shape.go·image.go·table.go)와 갈라지는 자리가 하나
// 있다: **소비자가 렌더러까지 가지 않는다.** `<chart>`는 값 층(core)이 도형·텍스트의 하위
// 트리로 **펼치고**, 확정 트리에는 남지 않는다 — 그래서 이 표를 읽는 층은 파서(문법·배치
// 규칙)와 값 층(펼침·데이터 검증) 둘이다. 웹·PPTX 렌더러에는 차트 코드가 없다.
//
// 그 사실을 침묵에 맡기지 않으려고 카탈로그에 **셋째 범주**를 둔다 (subset.go의
// desugaredKinds): 종류를 표에 넣되 "펼쳐지는 종류"로 표시하고, 렌더러 능력 표와의 대조가
// 그 종류를 **다루면 실패**로 판정한다 (render/vocabulary_test.go). 종류를 표에 아예 넣지
// 않으면 파서가 아는 태그가 카탈로그 밖으로 빠져 어느 대조도 보지 못하고, 그냥 넣으면 대조가
// 두 렌더러에 차트 코드를 요구한다 — 둘 다 차트를 값 층에서 펼친다는 설계와 어긋난다.

// 차트 요소 이름. 문자열 리터럴이 층에 흩어지지 않도록 상수로 둔다 (TableElement와 같은 근거).
const (
	// ChartElement는 차트 프레임이다 (KindChart). 슬라이드 직계 자식이자 레이아웃 컨테이너의
	// 자식이 될 수 있고, 값 층이 배경 도형 + 막대·라벨의 하위 트리로 펼친다.
	ChartElement = "chart"
	// ChartPointElement는 데이터 한 점이다 (KindChartPoint) — void다. `<chart>`의 직계
	// 자식으로만 오고, 라벨·값·강조색이 전부 속성이다.
	ChartPointElement = "point"
	// ChartSeriesElement는 계열 하나다 (KindChartSeries) — `<chart>`의 직계 자식이고 점들을 담는다. 이름 · 색 ·
	// (선 계열 차트에서) 종류를 갖는다. 계열을 쓰면 차트의 점은 전부 계열 안에 있다.
	ChartSeriesElement = "series"
)

// 펼친 차트의 틀 노드가 속한 부분 (Element.Part). 배치 진단이 넘친 부분을 이 이름으로 말하므로
// 값은 진단 문장에 그대로 들어가는 영어 구절이다.
const (
	// ChartPartPlot은 막대·선이 서는 밴드와 그 칸·기준선이다.
	ChartPartPlot = "plot"
	// ChartPartCategoryLabels는 플롯 아래의 범주 라벨 줄이다.
	ChartPartCategoryLabels = "category labels"
	// ChartPartLegend는 계열 형식의 범례 줄이다.
	ChartPartLegend = "legend"
)

// 차트 종류 — 모두 단일 계열이다. bar는 세로 막대, line은 점을 잇는 꺾은선, area는 꺾은선 아래를 칠한 영역,
// step은 범주 칸마다 수평인 계단이다. 선 · 영역 · 계단은 경로 도형(`type="path"`) 하나로 펼쳐진다.
const (
	ChartTypeBar  = "bar"
	ChartTypeLine = "line"
	ChartTypeArea = "area"
	ChartTypeStep = "step"
)

// chartTypes는 차트 종류의 **닫힌 목록**이다 (순서 = 진단·문서 순서). 값 어휘이므로 검사는
// 값 층이 하고 목록은 언어가 갖는다 — shapeTypes와 같은 분담이다: 진단 문구와 문서가 같은
// 목록을 읽어야 "쓸 수 있는 종류"가 두 곳에서 갈라지지 않는다.
//
// 하나로 시작하는 근거는 실측된 마찰이다: `line`·`pie`는 문법이 아니라 **값의
// 추가**이므로 근거가 생기면 이 줄에 이름을 더한다.
var chartTypes = []string{ChartTypeBar, ChartTypeLine, ChartTypeArea, ChartTypeStep}

// 차트 속성. 둘로 나눈 이유는 소비자가 아니라 **소유자**가 다르기 때문이다: 프레임은 종류·축
// (데이터를 그림으로 바꾸는 규칙)과 **계열색**을 갖고, 한 점의 강조는 점이 갖는다
// (`<point fill>`).
//
// 프레임에 `fill`을 열지 않는 것은 그대로다: 펼침의 뿌리는 계약상 배경이므로 같은
// 이름이 "이 박스의 면"과 "막대의 면"을 동시에 뜻하면 저자가 `fill="surface"`를 쓰고 회색 상자를
// 받는다 — 그래서 계열색의 이름이 `series`다.
var (
	// chartOwnAttrs는 차트 프레임만의 속성이다 (순서 = 진단·문서 순서).
	//
	//   - `type`: 종류. **필수**다 — `<chart/>`가 조용히 막대가 되면 저자의 오타와 구별되지
	//     않는다 (`<shape type>`과 같은 태도).
	//   - `unit`: 값 라벨의 꼬리 ("22" + "분"). 값이 아니라 **표기**이므로 축 계산에 들어가지
	//     않는다.
	//   - `max`: 값축의 최대. 생략하면 데이터 최대에서 유도한다 (값 층이 헤드룸을 준다).
	//   - `series`: **모든 막대**의 색. 생략하면 값 층의 기본 계열색이다. 라벨은
	//     칠하지 않는다 — 면의 색과 글의 색은 대비 요구가 다르다 (`surface` 막대와 `surface`
	//     글자는 같은 값이 아니다).
	//   - `alt`: 그림의 접근성 이름 — 펼침의 뿌리 도형에 실린다.
	//
	// 크기(`w`·`h`·`ratio` …)를 BoxAttrs로 받는 이유는 도형과 같다: 차트는 그 자체가 그림이라
	// 내용이 크기를 정할 수 없다. 오히려 **높이는 필수에 가깝다** — 막대 높이가 축에 대한
	// 백분율이라 내용이 높이를 정하면 기준이 순환한다 (값 층이 `h`나 `ratio`를 요구한다).
	chartOwnAttrs = []string{"type", "unit", "max", "series", "alt"}
	// chartPointAttrs는 점의 속성이다 (순서 = 진단·문서 순서). `label`·`value`는 필수이고
	// `fill`은 **계열에서 벗어난 한 점**의 강조다 — 막대와 그 점의 두 라벨을 함께 칠한다
	// (008의 마지막 열이 막대·값·범주까지 accent4인 것이 그 실물 요구다).
	//
	// 배치·크기·슬롯은 붙지 않는다 — 점은 flex 항목이 아니라 데이터 한 칸이고, 막대의 폭과
	// 자리는 차트가 정한다 (`<li>`·셀이 childPlacement를 받지 않는 것과 같은 근거).
	chartPointAttrs = []string{"label", "value", "fill"}
	// chartSeriesAttrs는 계열의 속성이다 (순서 = 진단 · 문서 순서). `name`은 필수다 (범례), `fill`은 계열색,
	// `type`은 선 계열 차트에서 이 계열만의 종류다 (line · area · step).
	chartSeriesAttrs = []string{"name", "fill", "type"}
)

// ChartAttrs는 `<chart>`의 고유 속성 + 자기 박스다 (자식 배치·슬롯은 withChildPlacement가
// 잇는다). 내보내는 이유는 ShapeAttrs·TableAttrs와 같다: 값 층이 이 목록을 소비해 "파서가 연
// 속성 = 값 층이 해석하는 속성"을 유지한다.
func ChartAttrs() []string {
	box := BoxAttrs()
	attrs := make([]string, 0, len(chartOwnAttrs)+len(box))
	attrs = append(attrs, chartOwnAttrs...)
	return append(attrs, box...)
}

// ChartPointAttrs는 `<point>`의 속성이다 (순서 = 진단·문서 순서).
func ChartPointAttrs() []string {
	attrs := make([]string, len(chartPointAttrs))
	copy(attrs, chartPointAttrs)
	return attrs
}

// ChartSeriesAttrs는 `<series>`의 속성이다 (순서 = 진단·문서 순서).
func ChartSeriesAttrs() []string {
	attrs := make([]string, len(chartSeriesAttrs))
	copy(attrs, chartSeriesAttrs)
	return attrs
}

// ChartTypes는 차트 종류 어휘다 (순서 = 진단·문서 순서). ShapeTypes와 같은 이유로
// 내보낸다 — 값 층의 진단과 문서 게이트가 같은 목록을 읽는다.
func ChartTypes() []string {
	types := make([]string, len(chartTypes))
	copy(types, chartTypes)
	return types
}

// chartChildren은 `<chart>`의 자식 어휘다 — 점뿐이다 (`<ul>`의 자식이 `<li>`뿐인 것과 같다).
// 데이터가 자식 요소인 근거: 속성 CSV에는 값마다 색을 줄 자리가 없고
// 진단이 "다섯 값 중 셋째"를 가리킬 방법이 없다.
func chartChildren() []string {
	return []string{ChartPointElement, ChartSeriesElement}
}

// chartSeriesChildren은 `<series>`의 자식 어휘다 — 점뿐이다.
func chartSeriesChildren() []string {
	return []string{ChartPointElement}
}
