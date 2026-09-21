package core

import (
	"math"
	"sort"
	"strings"
	"testing"

	"github.com/k2sebeom/balpyo/soml"
)

// 이 파일은 `<chart>`의 **펼침**을 판정한다 (ADR 0059): 값 층이 차트 하나를 도형·텍스트의
// 하위 트리로 바꾸고, 그 뒤의 파이프라인은 차트라는 것을 모른다.
//
// 첫 테스트가 이 항목의 계약 전부다 — **같은 뜻을 손으로 쓴 덱과 펼친 트리가 같다.** 그
// 판정이 성립하는 이유는 펼침이 저자가 쓸 수 있는 SOML만 쓰기 때문이고, 그것이 곧 "차트가
// 비공개 채널을 몰래 열지 못한다"는 유일한 관문이다.

// chartSlide는 차트 하나를 담은 슬라이드 원문이다. `<chart>`는 2행 3열이고 점은 3행부터다 —
// 진단의 line:col을 단정하는 케이스가 그 자리를 센다.
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

// chartTexts는 펼침이 낸 라벨들이다 (문서 순서 = 읽기 순서 — ADR 0058).
func chartTexts(deck *soml.Deck) []*soml.Element {
	var texts []*soml.Element
	for _, elem := range chartElements(deck) {
		if elem.Name == soml.TextElement {
			texts = append(texts, elem)
		}
	}
	return texts
}

// dumpTree는 해석된 트리를 이름·속성(이름순)·문자 데이터로 적는다 — 두 덱을 **깊은 비교**로
// 맞대는 도구다. 속성을 정렬하는 이유: 이 비교의 주장은 "같은 선언"이지 "같은 표기 순서"가
// 아니다 (속성 순서는 산출물을 바꾸지 않는다).
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

// **펼침 등가** — 차트 하나와 손으로 쓴 같은 뜻의 덱이 해석 후 같은 트리다 (ADR 0059의 결정 2).
//
// 손으로 쓴 쪽이 이 문법의 문서이기도 하다: 막대는 백분율 높이를 가진 도형이고, 값 라벨과
// 범주 라벨은 캡션 역할의 텍스트이며, 기준선은 1px 도형이다. 좌표 산수가 어디에도 없다 —
// 높이는 `값 ÷ 축 최대`의 백분율이고 픽셀은 브라우저가 잰다.
func TestChartExpandsToHandWrittenDeck(t *testing.T) {
	chart, diags := resolveSlide(t, chartSlide(`type="bar" unit="분" alt="월별 평균 탐지 시간" w="fill" h="420"`,
		chartPoint(`label="5월" value="22"`)+
			chartPoint(`label="6월" value="19"`)+
			chartPoint(`label="9월" value="12" fill="accent4"`)))
	if len(diags) != 0 {
		t.Fatalf("차트 덱의 진단이 없어야 한다: %v", diags)
	}

	// 축 최대는 데이터 최대 ÷ 0.9다 — 가장 높은 막대가 플롯 밴드의 90%가 되어 그 위에 올라탄
	// 값 라벨이 들어갈 자리가 남는다 (헤드룸이 없으면 라벨이 박스를 넘는다).
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

// 막대 높이는 `값 ÷ 축 최대`의 백분율이다 — 코어가 픽셀을 세지 않는다 (ADR 0059의 결정 3).
func TestChartBarHeightsAreValueOverAxisMax(t *testing.T) {
	tests := []struct {
		name  string
		attrs string
		want  []float64
	}{
		{
			// 축 최대를 생략하면 데이터 최대 ÷ 0.9다 (헤드룸 10%).
			name:  "기본 축 — 데이터 최대가 90%다",
			attrs: `type="bar" h="420"`,
			want:  []float64{90, 77.73, 49.09},
		},
		{
			// 저자가 축을 정하면 그 축이 기준이다 — `max`와 같은 값의 막대는 박스 끝까지 찬다
			// (값 라벨이 위로 삐져나오는 것은 저자의 선택이다).
			name:  "명시 축 — max가 기준이다",
			attrs: `type="bar" max="44" h="420"`,
			want:  []float64{50, 43.18, 27.27},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			deck := resolvedChart(t, tc.attrs,
				chartPoint(`label="5월" value="22"`)+
					chartPoint(`label="6월" value="19"`)+
					chartPoint(`label="9월" value="12"`))
			bars := chartBars(deck)
			if len(bars) != len(tc.want) {
				t.Fatalf("막대 %d개, want %d개", len(bars), len(tc.want))
			}
			for i, want := range tc.want {
				height := bars[i].ShapeStyle().Height
				if math.Abs(height.Value-want) > 0.005 {
					t.Errorf("막대[%d] 높이 = %g%%, want %g%%", i, height.Value, want)
				}
			}
		})
	}
}

// 값 라벨은 **저자가 적은 값 그대로** + `unit`이다 — 수치를 다시 포맷하지 않는다 (`22.5`가
// `22.50`이 되면 저자가 적은 것과 슬라이드에 보이는 것이 갈린다). 범주 라벨이 그 뒤에 온다
// (읽기 순서 = 문서 순서 — ADR 0058).
func TestChartLabelsCarryValueAndUnit(t *testing.T) {
	deck := resolvedChart(t, `type="bar" unit="분" h="420"`,
		chartPoint(`label="5월" value="22"`)+chartPoint(`label="6월" value="19.5"`))
	var got []string
	for _, text := range chartTexts(deck) {
		got = append(got, text.TextContent())
	}
	want := []string{"22분", "5월", "19.5분", "6월"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Errorf("라벨 = %v, want %v", got, want)
	}

	// 단위가 없으면 값만이다.
	deck = resolvedChart(t, `type="bar" h="420"`, chartPoint(`label="5월" value="22"`))
	if first := chartTexts(deck)[0].TextContent(); first != "22" {
		t.Errorf("단위 없는 값 라벨 = %q, want %q", first, "22")
	}
}

// 점의 `fill`은 그 점의 **막대와 라벨 둘**을 칠한다 — 008의 마지막 열이 막대·값·범주까지
// accent4인 것이 그 실물 요구다. 나머지 점은 기본 계열색이고 라벨은 캡션 색 그대로다.
func TestChartPointFillColorsBarAndLabels(t *testing.T) {
	deck := resolvedChart(t, `type="bar" h="420"`,
		chartPoint(`label="5월" value="22"`)+chartPoint(`label="9월" value="12" fill="accent4"`))
	bars := chartBars(deck)
	if len(bars) != 2 {
		t.Fatalf("막대 %d개, want 2개", len(bars))
	}
	if fill := bars[0].ShapeStyle().Fill; fill == nil || fill.Color.Token != "accent1" {
		t.Errorf("기본 막대의 채움 = %v, want accent1 (계열색)", fill)
	}
	if fill := bars[1].ShapeStyle().Fill; fill == nil || fill.Color.Token != "accent4" {
		t.Errorf("강조 막대의 채움 = %v, want accent4", fill)
	}
	texts := chartTexts(deck)
	if len(texts) != 4 {
		t.Fatalf("라벨 %d개, want 4개", len(texts))
	}
	caption := texts[0].TextStyle().Color
	for i, text := range texts[2:] {
		if got := text.TextStyle().Color; got.Token != "accent4" {
			t.Errorf("강조 점의 라벨[%d] 색 = %v, want accent4", i, got)
		}
	}
	for i, text := range texts[:2] {
		if got := text.TextStyle().Color; got != caption {
			t.Errorf("기본 점의 라벨[%d] 색 = %v, want 캡션 색 %v", i, got, caption)
		}
	}
}

// 값 0은 막대를 그리지 않는다 — 높이 0의 도형은 산출물에서 아무것도 아니므로 선언을 내지
// 않는다 (라벨은 남는다: "0"이 데이터다).
func TestChartZeroValueDrawsNoBar(t *testing.T) {
	deck := resolvedChart(t, `type="bar" h="420"`,
		chartPoint(`label="5월" value="22"`)+chartPoint(`label="6월" value="0"`))
	if bars := chartBars(deck); len(bars) != 1 {
		t.Errorf("막대 %d개, want 1개 (0인 점은 막대가 없다)", len(bars))
	}
	var labels []string
	for _, text := range chartTexts(deck) {
		labels = append(labels, text.TextContent())
	}
	if want := "22|5월|0|6월"; strings.Join(labels, "|") != want {
		t.Errorf("라벨 = %v, want %s", labels, want)
	}
}

// `alt`은 펼침의 **뿌리 도형**에 실린다 (ADR 0058) — 이름이 붙을 자리가 이미 있고, 막대와
// 라벨은 두 타깃 모두 읽히는 글로 남는다.
func TestChartAltNamesTheRootShape(t *testing.T) {
	deck := resolvedChart(t, `type="bar" alt="월별 평균 탐지 시간" h="420"`, chartPoint(`label="5월" value="22"`))
	root, ok := deck.Slides[0].Content[0].(*soml.Element)
	if !ok || root.Name != soml.ShapeElement {
		t.Fatalf("뿌리 = %v, want <shape>", deck.Slides[0].Content[0])
	}
	style := root.ShapeStyle()
	if style == nil || style.Alt != "월별 평균 탐지 시간" {
		t.Fatalf("뿌리 도형의 Alt = %v, want 저자가 준 이름", style)
	}
	// 뿌리는 **배경**이다 (ADR 0049) — 채움도 테두리도 없어서 차트가 상자에 담기지 않는다.
	if style.Fill == nil || !style.Fill.None {
		t.Errorf("뿌리 도형의 채움 = %v, want none", style.Fill)
	}
	if style.Border == nil || !style.Border.None {
		t.Errorf("뿌리 도형의 테두리 = %v, want none", style.Border)
	}
	// 그리고 안쪽 라벨은 이름에 흡수되지 않는다 — 글은 글로 남는다.
	if len(chartTexts(deck)) != 2 {
		t.Errorf("라벨 %d개, want 2개 (값·범주)", len(chartTexts(deck)))
	}
}

// 펼침 뒤의 트리에는 **펼쳐지는 종류가 하나도 없다** (ADR 0059) — 렌더러가 `<chart>`를 보지
// 않는다는 것이 산문이 아니라 기계가 지키는 사실이 되는 자리다. 렌더러 쪽의 짝은
// render/vocabulary_test.go가 본다 (두 렌더러의 능력 표에 펼쳐지는 종류가 없어야 한다).
func TestResolvedTreeHasNoDesugaredKinds(t *testing.T) {
	deck := resolvedChart(t, `type="bar" unit="분" h="420"`,
		chartPoint(`label="5월" value="22"`)+chartPoint(`label="6월" value="19"`))
	for _, elem := range chartElements(deck) {
		if soml.KindDesugared(soml.KindOf(elem.Name)) {
			t.Errorf("확정 트리에 <%s>가 남아 있다 (%s) — 값 층이 펼쳐야 한다", elem.Name, elem.Span)
		}
	}
}

// 진단 — 값 어휘와 데이터의 요구는 값 층이 소유한다. 자리는 **저자의 것**이다: 합성 노드가
// 아니라 `<chart>`/`<point>`의 속성을 가리킨다 (ADR 0059의 결정 11).
func TestChartDiagnostics(t *testing.T) {
	tests := []struct {
		name     string
		attrs    string
		points   string
		contains []string
	}{
		{
			name:     "type이 없다",
			attrs:    `h="420"`,
			points:   chartPoint(`label="5월" value="22"`),
			contains: []string{"<chart>에 type이 필요하다", "쓸 수 있는 종류: bar", "추측할 수 없다"},
		},
		{
			name:     "어휘 밖의 type",
			attrs:    `type="line" h="420"`,
			points:   chartPoint(`label="5월" value="22"`),
			contains: []string{`알 수 없는 차트 type "line"`, "쓸 수 있는 종류: bar"},
		},
		{
			// 막대 높이가 축에 대한 백분율이므로 차트 자신의 높이는 내용이 정할 수 없다 —
			// 백분율 인셋 프리셋과 같은 순환이다 (shapeInsetTextHeight의 선례).
			name:     "h도 ratio도 없다",
			attrs:    `type="bar"`,
			points:   chartPoint(`label="5월" value="22"`),
			contains: []string{"<chart>에 h나 ratio가 필요하다", "기준이 순환한다"},
		},
		{
			name:     "점이 없다",
			attrs:    `type="bar" h="420"`,
			points:   "",
			contains: []string{"<chart>에 <point>가 하나도 없다", "그릴 데이터가 없다"},
		},
		{
			name:     "label이 없다",
			attrs:    `type="bar" h="420"`,
			points:   chartPoint(`value="22"`),
			contains: []string{"<point>에 label이 필요하다"},
		},
		{
			name:     "value가 없다",
			attrs:    `type="bar" h="420"`,
			points:   chartPoint(`label="5월"`),
			contains: []string{"<point>에 value가 필요하다"},
		},
		{
			name:     "value가 수치가 아니다",
			attrs:    `type="bar" unit="분" h="420"`,
			points:   chartPoint(`label="5월" value="22분"`),
			contains: []string{`value가 수치가 아니다 ("22분")`, "단위는 <chart unit>에 쓴다"},
		},
		{
			name:     "value가 음수다",
			attrs:    `type="bar" h="420"`,
			points:   chartPoint(`label="5월" value="-3"`),
			contains: []string{"value는 음수일 수 없다", "0에서 시작한다"},
		},
		{
			name:     "값이 전부 0이다",
			attrs:    `type="bar" h="420"`,
			points:   chartPoint(`label="5월" value="0"`) + chartPoint(`label="6월" value="0"`),
			contains: []string{"값이 모두 0이다", "축을 정할 수 없다"},
		},
		{
			name:     "value가 max를 넘는다",
			attrs:    `type="bar" max="20" h="420"`,
			points:   chartPoint(`label="5월" value="22"`),
			contains: []string{"value 22가 max 20보다 크다", "막대가 축을 넘는다"},
		},
		{
			name:     "max가 수치가 아니다",
			attrs:    `type="bar" max="스물" h="420"`,
			points:   chartPoint(`label="5월" value="22"`),
			contains: []string{`max가 수치가 아니다 ("스물")`},
		},
		{
			name:     "max가 0 이하다",
			attrs:    `type="bar" max="0" h="420"`,
			points:   chartPoint(`label="5월" value="1"`),
			contains: []string{"max는 0보다 커야 한다"},
		},
		{
			name:     "unit에 값이 없다",
			attrs:    `type="bar" unit h="420"`,
			points:   chartPoint(`label="5월" value="22"`),
			contains: []string{"unit 속성에는 값이 필요하다"},
		},
		{
			name:     "alt이 비었다",
			attrs:    `type="bar" alt="" h="420"`,
			points:   chartPoint(`label="5월" value="22"`),
			contains: []string{"alt은 비어 있을 수 없다"},
		},
		{
			// 점의 fill은 도형의 fill과 **같은 어휘**다 — 진단도 같은 문구이고, 한 오타에
			// 진단은 하나다 (막대와 라벨 둘에 같은 값을 실어 두 번 말하지 않는다).
			name:     "점의 fill이 테마에 없는 토큰이다",
			attrs:    `type="bar" h="420"`,
			points:   chartPoint(`label="5월" value="22" fill="brand"`),
			contains: []string{`fill의 색 토큰 "brand"가 테마에 없다`},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, diags := resolveSlide(t, chartSlide(tc.attrs, tc.points))
			if len(diags) != 1 {
				t.Fatalf("진단 %d개, want 1개: %v", len(diags), diags)
			}
			if diags[0].Severity != soml.SeverityError {
				t.Errorf("심각도 = %v, want error (%s)", diags[0].Severity, diags[0].String())
			}
			for _, want := range tc.contains {
				if !strings.Contains(diags[0].Message, want) {
					t.Errorf("진단 = %q, want %q 포함", diags[0].String(), want)
				}
			}
		})
	}
}

// 진단의 **자리**가 저자의 것인지 — 점의 값을 가리키는 진단은 그 점의 그 속성에 밑줄을
// 긋는다 (합성 노드는 자기 Span을 갖지 않는다). 첫 점이 3행, 둘째 점이 4행이다.
func TestChartDiagnosticsPointAtTheAuthorsSpan(t *testing.T) {
	attrs := `label="6월" value="-3"`
	_, diags := resolveSlide(t, chartSlide(`type="bar" h="420"`,
		chartPoint(`label="5월" value="22"`)+chartPoint(attrs)))
	if len(diags) != 1 {
		t.Fatalf("진단 %d개, want 1개: %v", len(diags), diags)
	}
	// 값 구간은 둘째 점(4행)의 `-3`이다 — 인용 부호를 뺀 값에 밑줄이 그어진다.
	//
	// 칸을 **룬으로** 센다 (soml.Pos): 진단의 col은 사람이 보는 좌표계이므로 라벨의 한글
	// 한 글자가 세 칸으로 세어지지 않는다 — 바이트로 세면 이 단정이 라벨 글자 수에 따라
	// 흔들린다.
	prefix := "    <point " + attrs[:strings.Index(attrs, `value="`)] + `value="`
	wantCol := len([]rune(prefix)) + 1
	if diags[0].Span.Start.Line != 4 || diags[0].Span.Start.Col != wantCol {
		t.Errorf("위치 = %s, want 4:%d (%s)", diags[0].Span.Start, wantCol, diags[0].String())
	}
}
