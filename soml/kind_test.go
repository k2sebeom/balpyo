package soml

import (
	"strings"
	"testing"
)

// 카탈로그는 스스로 닫혀 있어야 한다 — 이름 색인(elementKinds)에 있는 종류가 ElementKinds()에
// 빠지면 렌더러 대조 테스트(render 패키지)가 그 종류를 아예 보지 못한다.
func TestElementKindsCoverCatalog(t *testing.T) {
	listed := make(map[ElementKind]bool, len(ElementKinds()))
	for _, kind := range ElementKinds() {
		if kind == KindNone {
			t.Error("ElementKinds()에 KindNone이 있다 — 카탈로그 밖의 값이다")
		}
		if listed[kind] {
			t.Errorf("ElementKinds()에 %v가 두 번 있다", kind)
		}
		listed[kind] = true
	}
	for name, kind := range elementKinds {
		if !listed[kind] {
			t.Errorf("<%s>의 종류 %v가 ElementKinds()에 없다", name, kind)
		}
		// 종류만 있고 문법이 없는 이름은 "알 수 없는 태그"로 진단되면서 렌더러는 종류로
		// 분기하는 어긋난 상태가 된다.
		if _, known := elementSpecs[name]; !known {
			t.Errorf("<%s>가 문법 표(elementSpecs)에 없다", name)
		}
	}
	for _, kind := range ElementKinds() {
		if len(ElementNamesOf(kind)) == 0 {
			t.Errorf("%v에 해당하는 태그 이름이 없다", kind)
		}
		if kind.String() == "Unknown" {
			t.Errorf("%d에 이름표가 없다 (elementKindNames)", int(kind))
		}
	}
	if got := ElementNamesOf(KindNone); got != nil {
		t.Errorf("ElementNamesOf(KindNone) = %v, want nil (카탈로그 밖이다)", got)
	}
}

// 종류 카탈로그는 파서·값 층·geom·web·pptx가 공유하는 한 표다 — 표가 흔들리면 좌표가 엉뚱한
// 요소에 붙는다. 종류별 이름(카탈로그 순서 — 진단이 이 순서를 싣는다)과 void 여부를 못 박는다.
func TestElementKindCatalog(t *testing.T) {
	tests := []struct {
		kind  ElementKind
		names string
		void  bool
	}{
		{KindBlock, "h1 h2 h3 p blockquote pre", false},
		{KindListContainer, "ul ol", false},
		{KindListItem, "li", false},
		{KindInline, "strong em code a br span mark", false},
		{KindLayoutContainer, "row col grid stack", false},
		{KindSpacer, "spacer", true},
		{KindShape, "shape", false},
		{KindImage, "image", true},
		{KindTable, "table", false},
		{KindTableRow, "tr", false},
		{KindTableCell, "th td", false},
		{KindChart, "chart", false},
		{KindChartPoint, "point", true},
		{KindConnector, "connector", true},
	}
	if len(tests) != len(ElementKinds()) {
		t.Errorf("표의 종류 %d개, ElementKinds() %d개 — 새 종류를 여기에 적는다", len(tests), len(ElementKinds()))
	}
	for _, tc := range tests {
		names := ElementNamesOf(tc.kind)
		if got := strings.Join(names, " "); got != tc.names {
			t.Errorf("ElementNamesOf(%v) = %q, want %q", tc.kind, got, tc.names)
		}
		if tc.kind == KindInline {
			continue // <br>만 void다
		}
		for _, name := range names {
			if IsVoidElement(name) != tc.void {
				t.Errorf("IsVoidElement(%q) = %v, want %v", name, !tc.void, tc.void)
			}
		}
	}
	if !IsVoidElement(BreakElement) {
		t.Error("<br>은 void다")
	}
	if !IsHeaderCell(TableHeaderCellElement) || IsHeaderCell(TableDataCellElement) {
		t.Error("IsHeaderCell은 <th>에만 참이어야 한다")
	}
	if !IsTableCell(TableHeaderCellElement) || !IsTableCell(TableDataCellElement) || IsTableCell(TableRowElement) {
		t.Error("IsTableCell은 <th>/<td>에만 참이어야 한다")
	}
}

// 태그마다 열린 속성 목록과 순서 — 진단 메시지("쓸 수 있는 속성: …")가 이 순서를 그대로 싣고,
// 목록에 없는 속성은 진단이다. 고유 → 외관 → 박스 → 자식 배치 → 슬롯 순으로 이어진다.
func TestElementAttrs(t *testing.T) {
	const (
		place = "grow shrink basis span self order at x y style slot enter"
		box   = "pad w h min-w min-h max-w max-h ratio"
		block = "variant size color align anchor w min-w max-w " + place
	)
	want := map[string]string{
		RootElement:  "layout id skip justify enter transition invert",
		noteElement:  "",
		TextElement:  "role " + block,
		"h1":         block,
		"h2":         block,
		"h3":         block,
		"p":          block,
		"blockquote": block,
		"pre":        "lang " + block,
		"ul":         "size color align anchor w min-w max-w " + place,
		"ol":         "size color align anchor w min-w max-w " + place,
		"li":         "reveal",
		"strong":     "",
		"em":         "",
		"code":       "",
		"a":          "href",
		"br":         "",
		"span":       "color weight size",
		"mark":       "",
		"row":        "gap align justify wrap " + box + " " + place,
		"col":        "gap align justify wrap " + box + " " + place,
		"grid":       "gap align justify wrap pad cols rows w h min-w min-h max-w max-h ratio " + place,
		"stack":      "align justify " + box + " " + place,
		"spacer":     place,
		"shape":      "type id alt d variant fill border radius shadow opacity rotate flip " + box + " " + place,
		"image":      "src id fit alt variant border radius shadow opacity rotate flip " + box + " " + place,
		"table":      "alt variant " + place,
		"tr":         "",
		"th":         "colspan rowspan fill border color align anchor",
		"td":         "colspan rowspan fill border color align anchor",
		"chart":      "type unit max series alt " + box + " " + place,
		"point":      "label value fill",
		"connector":  "from to route from-side to-side line start end variant label enter",
	}
	if len(want) != len(elementSpecs) {
		t.Errorf("표의 태그 %d개, elementSpecs %d개", len(want), len(elementSpecs))
	}
	for name, spec := range elementSpecs {
		if got := strings.Join(spec.attrs, " "); got != want[name] {
			t.Errorf("<%s> 속성 = %q\nwant %q", name, got, want[name])
		}
	}
	// 값 층이 소비하는 술어가 속성 표와 갈라지면 저자가 쓴 `w`가 파싱은 통과하고 조용히 사라진다.
	for _, name := range allElementNames() {
		open := name == TextElement || KindOf(name) == KindBlock || KindOf(name) == KindListContainer
		if got := TakesTextSize(name); got != open {
			t.Errorf("TakesTextSize(%q) = %v, want %v", name, got, open)
		}
	}
}

// 공개 카탈로그 접근자 — 순서가 진단·문서의 순서이고, 복사본이어야 호출자가 표를 흔들 수 없다.
func TestCatalogAccessors(t *testing.T) {
	tests := []struct {
		name string
		get  func() []string
		want string
	}{
		{"BoxAttrs", BoxAttrs, "pad w h min-w min-h max-w max-h ratio"},
		{"TextSizeAttrs", TextSizeAttrs, "w min-w max-w"},
		{"ChildPlacementAttrs", ChildPlacementAttrs, "grow shrink basis span self order at x y style"},
		{"ContainerAttrs(row)", func() []string { return ContainerAttrs(RowElement) }, "gap align justify wrap pad w h min-w min-h max-w max-h ratio"},
		{"ContainerAttrs(stack)", func() []string { return ContainerAttrs(StackElement) }, "align justify pad w h min-w min-h max-w max-h ratio"},
		{"ShapeTypes", ShapeTypes, "rect rounded-rect ellipse diamond cylinder chevron arrow-right hexagon parallelogram callout star cloud triangle cube left-brace right-brace path"},
		{"ShapeAttrs", ShapeAttrs, "type id alt d variant fill border radius shadow opacity rotate flip pad w h min-w min-h max-w max-h ratio"},
		{"ShapeAppearanceAttrs", ShapeAppearanceAttrs, "variant fill border radius shadow opacity rotate flip"},
		{"ImageAttrs", ImageAttrs, "src id fit alt variant border radius shadow opacity rotate flip pad w h min-w min-h max-w max-h ratio"},
		{"ImageAppearanceAttrs", ImageAppearanceAttrs, "variant border radius shadow opacity rotate flip"},
		{"TableCellAttrs", TableCellAttrs, "colspan rowspan fill border color align anchor"},
		{"ChartAttrs", ChartAttrs, "type unit max series alt pad w h min-w min-h max-w max-h ratio"},
		{"ChartPointAttrs", ChartPointAttrs, "label value fill"},
		{"ChartTypes", ChartTypes, "bar line area step"},
	}
	for _, tc := range tests {
		got := tc.get()
		if strings.Join(got, " ") != tc.want {
			t.Errorf("%s() = %q\nwant %q", tc.name, strings.Join(got, " "), tc.want)
			continue
		}
		got[0] = "짓밟기"
		if tc.get()[0] == "짓밟기" {
			t.Errorf("%s()가 내부 슬라이스를 그대로 준다", tc.name)
		}
	}
}

// 지금은 **카탈로그의 종류 전부**를 그릴 수 있다 (unrenderedKinds가 비었다). 두 렌더러가
// 정말로 받는지는 render/vocabulary_test.go가 보고, 여기서는 언어 층의 대답만 고정한다.
func TestKindRendered(t *testing.T) {
	for _, kind := range append(ElementKinds(), KindNone) {
		if !KindRendered(kind) {
			t.Errorf("KindRendered(%v) = false, want true", kind)
		}
	}
}

// **셋째 범주**다: 카탈로그에 있고 파서가 통과시키지만 값 층이 도형·텍스트로
// 펼치므로 렌더러가 **영원히** 보지 않는다. unrenderedKinds(갚을 빚)와 한 목록으로 묶지 않는다.
func TestChartKindsAreDesugared(t *testing.T) {
	desugared := map[ElementKind]bool{KindChart: true, KindChartPoint: true}
	for _, kind := range append(ElementKinds(), KindNone) {
		if got := KindDesugared(kind); got != desugared[kind] {
			t.Errorf("KindDesugared(%v) = %v, want %v", kind, got, desugared[kind])
		}
	}
	if got := DesugaredKinds(); len(got) != len(desugared) || !desugared[got[0]] || !desugared[got[1]] {
		t.Errorf("DesugaredKinds() = %v, want 차트 두 종류", got)
	}
}
