package soml

// 이 파일은 표의 **요소 종류·속성 어휘 카탈로그**다.
//
// 서브셋(subset.go)·레이아웃(layout.go)·도형(shape.go)·이미지(image.go) 카탈로그와 같은
// 자리에 같은 태도로 둔다: 이 표를 파서(문법·배치 규칙), 값 층(core의 병합·색·정렬 검사),
// 지오메트리 패스(잴 대상 — 프레임 박스와 셀 박스), 웹 렌더러(진짜 `<table>`), PPTX
// 렌더러(`p:graphicFrame`+`a:tbl`)가 전부 소비한다. 다섯 층이 각자 이름 목록을 들고 있으면
// 태그 하나·속성 하나를 추가할 때 한 곳을 빼먹는 것이 곧 "파싱은 되는데 그려지지 않는"
// 조용한 실패다.
//
// **표는 리스트와 레이아웃 컨테이너의 교차다**: 프레임(`<table>`)은 도형 하나가
// 되되(리스트가 도형 하나이듯), 내부 격자의 기하(열 너비·행 높이)는 잰 셀 박스에서 유도한다
// (레이아웃 컨테이너가 자식 좌표만 남기고 사라지는 것의 변주). `<tr>`/`<th>`/`<td>`는 도형이
// 되지 않는다 — `a:tr`/`a:tc`로 프레임 안에 접힌다.

// 표 요소 이름. 문자열 리터럴이 여러 층에 흩어지지 않도록 상수로 둔다 (ShapeElement·
// ListItemElement와 같은 근거).
const (
	// TableElement는 표 프레임이다 (KindTable → p:graphicFrame + a:tbl). 슬라이드 직계
	// 자식이자 레이아웃 컨테이너 자식이 될 수 있는 도형-경계 요소다.
	TableElement = "table"
	// TableRowElement는 표 행이다 (KindTableRow → a:tr). <table>의 직계 자식으로만 온다.
	TableRowElement = "tr"
	// TableHeaderCellElement는 헤더 셀이다 (KindTableCell → a:tc, 헤더 외관). <tr>의 자식으로만.
	TableHeaderCellElement = "th"
	// TableDataCellElement는 데이터 셀이다 (KindTableCell → a:tc, 본문 외관). <tr>의 자식으로만.
	TableDataCellElement = "td"
)

// tableCellElements는 셀 태그 목록이다 (순서 = 진단·문서 순서 — 헤더가 먼저). <tr>의 자식
// 어휘이자 KindTableCell 색인의 파생원이다.
var tableCellElements = []string{TableHeaderCellElement, TableDataCellElement}

// 표 속성. 셋으로 나눈 이유는 소비자가 다르기 때문이다: 프레임의 `variant`는 테마 표
// 스타일(줄무늬 계열)을 고르고, 셀의 병합 속성은 파서·PPTX 격자 조립(gridSpan/rowSpan)이,
// 셀의 외관 속성은 캐스케이드가 소비한다. 이어 붙인 순서가 곧 진단 메시지·문서의 순서다.
var (
	// tableOwnAttrs는 표 프레임만의 속성이다. `variant`는 테마 `table` 컴포넌트의 변형
	// (예: `striped`)을 고른다 — 값 어휘·폴백은 값 층(core)이 본다 (도형 `variant`의 선례).
	//
	// 크기 속성(w/h)을 열지 않는 이유는 리스트(`<ul>`)와 같다: 표는 잰 내용이 곧 크기다.
	// 프레임 `xfrm`은 지오메트리 패스가 잰 `<table>` 박스에서, 열 너비·행 높이는 잰 셀
	// 박스에서 온다 — 저자가 EMU를 손으로 적을 자리가 아니다.
	//
	// `alt`은 **표의 이름**이다 — 웹은 `<table aria-label>`, PPTX는
	// `p:graphicFrame`의 `p:cNvPr@descr`이다. 셀은 두 타깃 모두 항해 가능한 채로 남으므로
	// 이 이름은 내용을 대체하지 않고 더한다.
	tableOwnAttrs = []string{"alt", "variant"}
	// tableCellMergeAttrs는 셀 병합 속성이다 (HTML식). `colspan`→OOXML `gridSpan`,
	// `rowspan`→`rowSpan`. 병합에 덮이는 격자 셀은 렌더러가 `hMerge`/`vMerge` 빈 `a:tc`로
	// 낸다 (OOXML 요구사항) — 저자가 쓰는 것은 span 두 개뿐이다. 값(정수·격자 초과)은 값
	// 층이 본다.
	tableCellMergeAttrs = []string{"colspan", "rowspan"}
	// tableCellAppearanceAttrs는 셀 외관 채널이다. `fill`은 셀 배경(줄무늬를 우리가 명시로
	// 칠한다 — 표 스타일 없이 `tcPr`에 명시한다), `border`는 셀 테두리, `color`/`align`/
	// `anchor`는 셀 텍스트의 색·가로 정렬·세로 앵커다. 값 어휘·캐스케이드는 값 층이 소유한다
	// (도형 외관 속성의 선례와 같다).
	//
	// **`variant`는 셀에 열지 않는다** — 표의 테마 변형(줄무늬 계열)은 프레임 `<table variant>`
	// 하나가 고르고, 헤더/본문/줄무늬 기본값은 테마 `table` 컴포넌트가 셀에 자동으로 얹는다.
	// 셀마다 변형을 열면 받아 놓고 쓸 곳 없는 선언이 되고(v1 테마에 셀 변형이 없다), 그 침묵이
	// 닫힌 어휘가 막으려던 실패다. 셀 단위 조정은 fill/border/color/align/anchor 오버라이드다.
	tableCellAppearanceAttrs = []string{"fill", "border", "color", "align", AnchorAttr}
)

// TableAttrs는 `<table>`의 고유 속성이다 (자식 배치·슬롯은 withChildPlacement가 잇는다).
//
// 내보내는 이유는 ShapeAttrs·ContainerAttrs와 같다: 값 층(core)이 이 목록을 소비해 "파서가
// 연 속성 = 값 층이 해석하는 속성"을 유지한다.
func TableAttrs() []string {
	attrs := make([]string, len(tableOwnAttrs))
	copy(attrs, tableOwnAttrs)
	return attrs
}

// TableCellAttrs는 `<th>`/`<td>`의 병합·외관 속성이다 (순서 = 진단·문서 순서). 자식 배치·슬롯
// 속성은 붙지 않는다 — 셀은 레이아웃 컨테이너의 자식(flex 항목)도 슬라이드 직계 자식도
// 아니기 때문이다 (`<li>`가 childPlacement를 받지 않는 것과 같은 근거).
func TableCellAttrs() []string {
	attrs := make([]string, 0, len(tableCellMergeAttrs)+len(tableCellAppearanceAttrs))
	attrs = append(attrs, tableCellMergeAttrs...)
	return append(attrs, tableCellAppearanceAttrs...)
}

// TableCellAppearanceAttrs는 셀 외관 속성 목록이다 (순서 = 진단·문서 순서). 캐스케이드가
// 이 목록의 속성만 해석한다 (ShapeAppearanceAttrs의 선례와 같은 이유로 내보낸다).
func TableCellAppearanceAttrs() []string {
	attrs := make([]string, len(tableCellAppearanceAttrs))
	copy(attrs, tableCellAppearanceAttrs)
	return attrs
}

// tableChildren은 `<table>`의 자식 어휘다 — 행뿐이다 (`<ul>`의 자식이 `<li>`뿐인 것과 같다).
func tableChildren() []string {
	return []string{TableRowElement}
}

// tableRowChildren은 `<tr>`의 자식 어휘다 — 헤더·데이터 셀이다 (순서 = 진단 순서).
func tableRowChildren() []string {
	children := make([]string, len(tableCellElements))
	copy(children, tableCellElements)
	return children
}

// IsTableCell은 이름이 셀 태그(`<th>`/`<td>`)인지 보고한다. 헤더 여부(`<th>`)는 IsHeaderCell이
// 가른다 — 두 셀은 종류가 같고(KindTableCell) 외관 기본값만 다르기 때문이다.
func IsTableCell(name string) bool {
	return name == TableHeaderCellElement || name == TableDataCellElement
}

// IsHeaderCell은 이름이 헤더 셀(`<th>`)인지 보고한다. core가 셀 텍스트 외관을 테마 `table`
// 컴포넌트의 header/body 중 어느 기본값으로 해석할지, 렌더러가 웹의 `<th>`/`<td>`와 PPTX의
// 헤더 힌트를 가를지 이 술어로 정한다.
func IsHeaderCell(name string) bool {
	return name == TableHeaderCellElement
}
