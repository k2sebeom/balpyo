package soml

import (
	"strings"
	"testing"
)

// 차트의 정상 트리 — 태그(chart/point)·고유 속성·박스·자식 배치·슬롯이 파싱을 그대로
// 통과한다. dumpNodes의 표현은 subset_test.go와 공유한다.
//
// 이 파일이 카탈로그의 다른 시험들과 갈라지는 자리는 **종류의 셋째 범주**다 (ADR 0059):
// `<chart>`와 `<point>`는 카탈로그에 있고 파서가 통과시키지만 렌더러에는 **영원히** 닿지
// 않는다 — 값 층이 도형·텍스트로 펼치기 때문이다. 그 사실을 아래 TestChartKindsAreDesugared가
// 못 박고, 렌더러 쪽의 대조는 render/vocabulary_test.go가 본다.
func TestParseChart(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want string
	}{
		{
			name: "막대 차트 — 점 셋",
			src:  `<slide><chart type="bar" h="420"><point label="5월" value="22"/><point label="6월" value="19"/><point label="7월" value="15"/></chart></slide>`,
			want: `chart[type=bar h=420](point[label=5월 value=22] point[label=6월 value=19] point[label=7월 value=15])`,
		},
		{
			name: "고유 속성 넷 — type·unit·max·alt",
			src:  `<slide><chart type="bar" unit="분" max="25" alt="월별 탐지 시간" h="420"><point label="5월" value="22"/></chart></slide>`,
			want: `chart[type=bar unit=분 max=25 alt=월별 탐지 시간 h=420](point[label=5월 value=22])`,
		},
		{
			name: "점의 fill — 한 점만 강조한다",
			src:  `<slide><chart type="bar" h="420"><point label="9월" value="12" fill="accent4"/></chart></slide>`,
			want: `chart[type=bar h=420](point[label=9월 value=12 fill=accent4])`,
		},
		{
			name: "차트는 레이아웃 컨테이너의 자식이다",
			src:  `<slide><row><chart type="bar" grow="1" h="300"><point label="a" value="1"/></chart></row></slide>`,
			want: `row(chart[type=bar grow=1 h=300](point[label=a value=1]))`,
		},
		{
			name: "차트는 슬롯을 채울 수 있다",
			src:  `<slide layout="content"><chart type="bar" slot="body" h="300"><point label="a" value="1"/></chart></slide>`,
			want: `chart[type=bar slot=body h=300](point[label=a value=1])`,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			doc, diags := Parse("slides/001_chart.slide", tc.src)
			if len(diags) != 0 {
				t.Fatalf("진단이 없어야 한다:\n%s", diags.Err())
			}
			if got := dumpNodes(doc.Content); got != tc.want {
				t.Errorf("트리 = %q\nwant %q", got, tc.want)
			}
		})
	}
}

// 차트 요소의 종류 — 카탈로그(subset.go)에 두 종류가 더해진다.
func TestChartKinds(t *testing.T) {
	cases := map[string]ElementKind{
		ChartElement:      KindChart,
		ChartPointElement: KindChartPoint,
	}
	for name, want := range cases {
		if got := KindOf(name); got != want {
			t.Errorf("KindOf(%q) = %v, want %v", name, got, want)
		}
	}
	// `<point>`는 void다 — 값과 라벨이 속성이고 담을 것이 없다 (`<image>`와 같은 근거).
	if !IsVoidElement(ChartPointElement) {
		t.Errorf("IsVoidElement(%q) = false, want true", ChartPointElement)
	}
	if IsVoidElement(ChartElement) {
		t.Errorf("IsVoidElement(%q) = true, want false — 차트는 점을 담는다", ChartElement)
	}
	if got := strings.Join(ElementNamesOf(KindChart), " "); got != ChartElement {
		t.Errorf("ElementNamesOf(KindChart) = %q, want %q", got, ChartElement)
	}
	if got := strings.Join(ElementNamesOf(KindChartPoint), " "); got != ChartPointElement {
		t.Errorf("ElementNamesOf(KindChartPoint) = %q, want %q", got, ChartPointElement)
	}
}

// **셋째 범주**다 (ADR 0059): 카탈로그에 있고 파서가 통과시키지만 렌더러에 닿지 않는다.
//
// `unrenderedKinds`("아직 못 그린다")와 장치는 같고 뜻은 반대다 — 이쪽은 값 층이 펼치므로
// 렌더러가 **영원히** 보지 않는다. 둘을 한 목록으로 묶지 않는 이유가 그것이다: 하나는 갚을
// 빚이고 하나는 계약이다.
func TestChartKindsAreDesugared(t *testing.T) {
	for _, kind := range []ElementKind{KindChart, KindChartPoint} {
		if !KindDesugared(kind) {
			t.Errorf("KindDesugared(%v) = false, want true — 값 층이 도형·텍스트로 펼친다 (ADR 0059)", kind)
		}
		// 펼쳐지는 종류도 **어휘로서는 정상**이다 — 파서가 막지 않으므로 KindRendered는 참이다
		// (거짓이면 저자가 `<chart>`를 "아직 못 그린다"는 말로 듣는다).
		if !KindRendered(kind) {
			t.Errorf("KindRendered(%v) = false, want true — 펼쳐지는 종류는 미구현이 아니다", kind)
		}
	}
	for _, kind := range []ElementKind{KindShape, KindTable, KindBlock, KindNone} {
		if KindDesugared(kind) {
			t.Errorf("KindDesugared(%v) = true, want false", kind)
		}
	}
	// 목록은 카탈로그 안에 있어야 한다 — 카탈로그 밖 종류를 적으면 렌더러 대조가 그 줄을
	// 보지 못한다 (ElementKinds가 순회의 출처다).
	catalog := make(map[ElementKind]bool, len(ElementKinds()))
	for _, kind := range ElementKinds() {
		catalog[kind] = true
	}
	for _, kind := range DesugaredKinds() {
		if !catalog[kind] {
			t.Errorf("DesugaredKinds()의 %v가 ElementKinds()에 없다", kind)
		}
	}
	if len(DesugaredKinds()) == 0 {
		t.Error("DesugaredKinds()가 비었다 — 차트 두 종류가 들어 있어야 한다")
	}
}

// 속성 목록과 순서 — 진단 메시지가 이 순서를 그대로 싣는다 (ShapeAttrs·TableCellAttrs의
// 회귀 시험과 같은 태도). 고유 넷 → 박스 → 자식 배치 → 슬롯 순이다.
func TestChartAttrsOrder(t *testing.T) {
	want := "type, unit, max, alt, pad, w, h, min-w, min-h, max-w, max-h, ratio"
	if got := strings.Join(ChartAttrs(), ", "); got != want {
		t.Errorf("ChartAttrs() = %q\nwant %q", got, want)
	}
	attrs := strings.Join(ElementAttrs(ChartElement), ", ")
	for _, need := range []string{"type, unit, max, alt", "grow", "shrink", SlotAttr} {
		if !strings.Contains(attrs, need) {
			t.Errorf("<chart>의 속성 목록에 %q가 없다: %s", need, attrs)
		}
	}
	// 외관은 **점이 소유한다** — 프레임에 fill을 주면 막대 색과 겹쳐 어느 쪽이 보이는지가
	// 갈린다 (ADR 0059의 결정 8, `<table>`이 외관을 셀에 둔 것과 같은 판단).
	for _, banned := range []string{"fill", "border", "variant", "radius", "shadow", "rotate"} {
		for _, a := range ElementAttrs(ChartElement) {
			if a == banned {
				t.Errorf("<chart>에 %q가 열려선 안 된다 (외관은 <point>가 소유한다)", banned)
			}
		}
	}
}

// 점의 속성 — 라벨·값·강조색 셋이다. 배치·슬롯·크기는 붙지 않는다 (점은 flex 항목이 아니라
// 데이터 한 칸이다 — `<li>`·셀이 childPlacement를 받지 않는 것과 같은 근거).
func TestChartPointAttrsOrder(t *testing.T) {
	want := "label, value, fill"
	if got := strings.Join(ChartPointAttrs(), ", "); got != want {
		t.Errorf("ChartPointAttrs() = %q\nwant %q", got, want)
	}
	for _, banned := range []string{SlotAttr, "grow", "span", StyleAttr, "w", "h", "variant"} {
		for _, a := range ElementAttrs(ChartPointElement) {
			if a == banned {
				t.Errorf("<point>에 %q가 열려선 안 된다", banned)
			}
		}
	}
}

// 차트의 v1 종류 어휘는 `bar` 하나다 — 값 어휘이므로 검사는 값 층이 하고, 목록은 언어가
// 갖는다 (ShapeTypes의 선례: 진단 문구와 문서가 같은 목록을 읽는다).
func TestChartTypes(t *testing.T) {
	if got := strings.Join(ChartTypes(), ", "); got != "bar" {
		t.Errorf("ChartTypes() = %q, want %q", got, "bar")
	}
}

// 배치 위반 — 진단의 line:col과 문구까지 단정한다 (표·리스트 규칙의 선례와 같은 태도).
func TestParseChartPlacementErrors(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want []string
	}{
		{
			name: "point는 chart 밖에 올 수 없다",
			src:  `<slide><point label="a" value="1"/></slide>`,
			want: []string{"1:9 <point>는 <chart>의 자식으로만 쓸 수 있다"},
		},
		{
			name: "chart의 자식은 point뿐이다",
			src:  `<slide><chart type="bar" h="300"><p>본문</p></chart></slide>`,
			want: []string{"1:35 <chart>의 자식으로 <p>는 올 수 없다 (쓸 수 있는 태그: <point>)"},
		},
		{
			name: "chart 안의 문자 데이터는 점으로 감싸라고 말한다",
			src:  `<slide><chart type="bar" h="300">22</chart></slide>`,
			want: []string{"1:34 <chart> 안에 문자 데이터를 직접 쓸 수 없다 (<point>로 감싼다)"},
		},
		{
			name: "point는 void다 — 내용을 담을 수 없다",
			src:  `<slide><chart type="bar" h="300"><point label="a" value="1">22</point></chart></slide>`,
			want: []string{"1:35 void 태그 <point>은 내용을 가질 수 없다 (<point/>로 쓴다)"},
		},
		{
			name: "차트는 도형의 직속 자식이 아니다 (컨테이너로 감싼다 — 표와 같은 규칙)",
			src:  `<slide><shape type="rect" w="400" h="300"><chart type="bar" h="200"><point label="a" value="1"/></chart></shape></slide>`,
			want: []string{"1:46 <shape>의 자식으로 <chart>는 올 수 없다"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, diags := Parse("slides/001_chart.slide", tc.src)
			if len(diags) != len(tc.want) {
				t.Fatalf("진단 %d개, want %d개:\n%s", len(diags), len(tc.want), diags.Err())
			}
			for i, want := range tc.want {
				pos, _, _ := strings.Cut(want, " ")
				message := strings.TrimPrefix(want, pos+" ")
				if got := diags[i].Span.Start.String(); got != pos {
					t.Errorf("진단[%d] 위치 = %s, want %s (%s)", i, got, pos, diags[i].Message)
				}
				if !strings.Contains(diags[i].Message, message) {
					t.Errorf("진단[%d] 메시지 = %q, want %q 포함", i, diags[i].Message, message)
				}
			}
		})
	}
}
