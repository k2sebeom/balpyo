package geom

import (
	"strings"
	"testing"

	"github.com/k2sebeom/balpyo/core"
	"github.com/k2sebeom/balpyo/soml"
)

// measured는 식별자마다 rect(와 줄 박스)를 손으로 준 지오메트리다. 잰 지오메트리와 같은
// 계약을 갖도록 Verify를 지나게 한다.
func measured(t *testing.T, deck *soml.Deck, rects map[string]Rect, lines map[string][]Rect) *Geometry {
	t.Helper()
	refs := Refs(deck)
	boxes := make([]Box, 0, len(refs))
	for _, ref := range refs {
		rect, ok := rects[ref.ID]
		if !ok {
			t.Fatalf("%s <%s>의 rect가 없다 — 판정 입력은 요소 전부를 덮어야 한다", ref.ID, ref.El)
		}
		boxes = append(boxes, Box{ID: ref.ID, El: ref.El, Slide: ref.Slide, Rect: rect, Lines: lines[ref.ID]})
	}
	g := &Geometry{Stage: deck.Stage, Boxes: boxes, index: newIndex(boxes)}
	if err := g.Verify(deck); err != nil {
		t.Fatalf("시험 입력이 덱과 어긋난다: %v", err)
	}
	return g
}

// textDeck은 슬라이드 직속 텍스트 n개인 덱이다 (해석까지 끝난 상태).
func textDeck(t *testing.T, counts ...int) *soml.Deck {
	t.Helper()
	slides := make([][]soml.Node, 0, len(counts))
	for _, n := range counts {
		content := make([]soml.Node, 0, n)
		for i := 0; i < n; i++ {
			content = append(content, textElem(chars("내용")))
		}
		slides = append(slides, content)
	}
	deck := testDeck(slides...)
	if diags := core.Resolve(deck, nil); len(diags) > 0 {
		t.Fatalf("외관 해석 진단: %v", diags)
	}
	return deck
}

// wantDiags는 진단 목록이 기대한 조각들을 순서대로 담는지 본다 (개수까지).
func wantDiags(t *testing.T, diags soml.Diagnostics, want ...string) {
	t.Helper()
	if len(diags) != len(want) {
		t.Fatalf("진단 %d건, want %d건:\n%s", len(diags), len(want), diags)
	}
	for i, fragment := range want {
		if !strings.Contains(diags[i].Message, fragment) {
			t.Errorf("%d번째 진단 = %q, want %q 포함", i, diags[i].Message, fragment)
		}
		if diags[i].Severity != soml.SeverityWarning {
			t.Errorf("%d번째 진단의 심각도 = %v, want warning — 승격은 --strict의 몫이다",
				i, diags[i].Severity)
		}
	}
}

// canvasDeck은 슬라이드 직속 텍스트 하나인 canvas 덱이다 — 직계가 전부 Free다.
func canvasDeck(t *testing.T) *soml.Deck {
	deck := textDeck(t, 1)
	deck.Slides[0].Layout = soml.LayoutCanvas
	return deck
}

// stackDeck은 `<stack>`에 텍스트 둘을 쌓은 덱이다.
func stackDeck(t *testing.T) *soml.Deck {
	deck := testDeck([]soml.Node{&soml.Element{Name: soml.StackElement, Children: []soml.Node{
		textElem(chars("바닥")), textElem(chars("위")),
	}}})
	if diags := core.Resolve(deck, nil); len(diags) > 0 {
		t.Fatalf("외관 해석 진단: %v", diags)
	}
	return deck
}

// 판정 하나하나의 경계다. 정상 덱이 조용한 것이 가장 중요하다 — 고칠 수 없는 경고가 나면
// 저자는 경고를 읽지 않게 되고 나머지 판정 전부를 잃는다.
func TestDiagnose(t *testing.T) {
	one := func(t *testing.T) *soml.Deck { return textDeck(t, 1) }
	two := func(t *testing.T) *soml.Deck { return textDeck(t, 2) }
	tests := []struct {
		name    string
		deck    func(t *testing.T) *soml.Deck
		rects   map[string]Rect
		lines   map[string][]Rect
		content []Rect // 잰 콘텐츠 박스 (nil이면 기본 여백)
		want    []string
	}{
		{
			name: "제자리에 있는 덱은 조용하다 (변이 맞닿는 것은 겹침이 아니다)",
			deck: two,
			rects: map[string]Rect{
				"s0.0": {X: 64, Y: 64, W: 1152, H: 200},
				"s0.1": {X: 64, Y: 264, W: 1152, H: 100},
			},
		},
		{
			name:  "스테이지 이탈",
			deck:  one,
			rects: map[string]Rect{"s0.0": {X: 64, Y: 600, W: 1152, H: 180}},
			want:  []string{"스테이지를 아래로 60px 벗어난다"},
		},
		{
			name:  "canvas의 Free 요소도 스테이지 이탈은 말한다",
			deck:  canvasDeck,
			rects: map[string]Rect{"s0.0": {X: 1200, Y: 64, W: 200, H: 100}},
			want:  []string{"스테이지를 오른쪽으로 120px 벗어난다"},
		},
		{
			name:  "콘텐츠 박스 이탈 (잰 값이 없으면 기본 여백)",
			deck:  one,
			rects: map[string]Rect{"s0.0": {X: 64, Y: 64, W: 1152, H: 626}},
			want:  []string{"콘텐츠 박스를 아래로 34px 넘친다"},
		},
		{
			name:  "좌표를 쓴 요소의 여백 침범은 저자의 선택이다",
			deck:  canvasDeck,
			rects: map[string]Rect{"s0.0": {X: 0, Y: 0, W: 400, H: 100}},
		},
		{
			name:  "요소 하나에 이탈 진단 하나",
			deck:  one,
			rects: map[string]Rect{"s0.0": {X: 64, Y: 64, W: 1152, H: 800}},
			want:  []string{"스테이지를 아래로"},
		},
		{
			name: "1px 미만의 이탈·겹침은 허용 오차다",
			deck: two,
			rects: map[string]Rect{
				"s0.0": {X: 64, Y: 64, W: 576.5, H: 592.5},
				"s0.1": {X: 640, Y: 64, W: 576, H: 200},
			},
		},
		{
			name:  "글줄의 가로 이탈",
			deck:  one,
			rects: map[string]Rect{"s0.0": {X: 64, Y: 64, W: 400, H: 40}},
			lines: map[string][]Rect{"s0.0": {{X: 64, Y: 66, W: 520, H: 36}}},
			want:  []string{"글줄이 박스를 오른쪽으로 120px 넘친다"},
		},
		{
			// 글자 상자가 CSS 줄 박스보다 커서 정상 텍스트도 세로로 몇 px 넘친다.
			name:  "글줄의 세로 이탈은 보지 않는다",
			deck:  one,
			rects: map[string]Rect{"s0.0": {X: 64, Y: 64, W: 400, H: 40}},
			lines: map[string][]Rect{"s0.0": {{X: 64, Y: 64, W: 300, H: 46}}},
		},
		{
			name: "형제 겹침은 나중 형제에 붙는다",
			deck: two,
			rects: map[string]Rect{
				"s0.0": {X: 64, Y: 64, W: 600, H: 200},
				"s0.1": {X: 500, Y: 200, W: 600, H: 200},
			},
			want: []string{"형제 <text>와 164px×64px 겹친다"},
		},
		{
			name: "좌표가 슬라이드 상대이므로 다른 슬라이드는 겹침이 아니다",
			deck: func(t *testing.T) *soml.Deck { return textDeck(t, 1, 1) },
			rects: map[string]Rect{
				"s0.0": {X: 64, Y: 64, W: 600, H: 200},
				"s1.0": {X: 64, Y: 64, W: 600, H: 200},
			},
		},
		{
			name: "stack의 자식은 겹치도록 쌓은 것이다",
			deck: stackDeck,
			rects: map[string]Rect{
				"s0.0":   {X: 64, Y: 64, W: 600, H: 200},
				"s0.0.0": {X: 64, Y: 64, W: 600, H: 200},
				"s0.0.1": {X: 64, Y: 64, W: 600, H: 200},
			},
		},
		{
			name:    "잰 콘텐츠 박스가 스테이지 전체면 꽉 찬 박스도 조용하다",
			deck:    one,
			rects:   map[string]Rect{"s0.0": {X: 0, Y: 0, W: 1280, H: 720}},
			content: []Rect{{X: 0, Y: 0, W: 1280, H: 720}},
		},
		{
			name:    "기본보다 넓은 잰 여백은 기본 여백에 맞춘 박스를 경고한다",
			deck:    one,
			rects:   map[string]Rect{"s0.0": {X: 64, Y: 96, W: 1152, H: 528}},
			content: []Rect{{X: 96, Y: 96, W: 1088, H: 528}},
			want:    []string{"콘텐츠 박스를 오른쪽으로 32px 넘친다"},
		},
		{
			name: "콘텐츠 박스는 슬라이드마다 다르다",
			deck: func(t *testing.T) *soml.Deck { return textDeck(t, 1, 1) },
			rects: map[string]Rect{
				"s0.0": {X: 0, Y: 0, W: 1280, H: 720},
				"s1.0": {X: 0, Y: 0, W: 1280, H: 720},
			},
			content: []Rect{{X: 0, Y: 0, W: 1280, H: 720}, ContentBox(soml.Stage{Width: 1280, Height: 720})},
			want:    []string{"콘텐츠 박스를 아래로 64px 넘친다"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			deck := tc.deck(t)
			g := measured(t, deck, tc.rects, tc.lines)
			g.Content = tc.content
			wantDiags(t, Diagnose(deck, g), tc.want...)
		})
	}
}

// 부모 박스 이탈이 "고정 크기 박스가 내용을 담지 못했다"의 모양이다 — 저자가 `h`를 준 도형에
// 글이 넘치면 자식 박스가 부모 변을 넘어간다.
func TestDiagnoseParentEscape(t *testing.T) {
	// `<stack>`은 `x`를 쓸 수 있는 자리여서다.
	shapeDeck := func(t *testing.T, attrs ...soml.Attr) *soml.Deck {
		t.Helper()
		shape := &soml.Element{
			Name:     soml.ShapeElement,
			Attrs:    append([]soml.Attr{{Name: "type", Value: "rect", HasValue: true}}, attrs...),
			Children: []soml.Node{textElem(chars("담기지 않는 긴 본문"))},
		}
		deck := testDeck([]soml.Node{&soml.Element{Name: soml.StackElement, Children: []soml.Node{shape}}})
		if diags := core.Resolve(deck, nil); len(diags) > 0 {
			t.Fatalf("외관 해석 진단: %v", diags)
		}
		return deck
	}
	// 도형 140px 안에 여백 36px + 내용 108px — 아래 변을 4px 넘는다.
	rects := map[string]Rect{
		"s0.0":     {X: 64, Y: 64, W: 400, H: 140},
		"s0.0.0":   {X: 64, Y: 64, W: 400, H: 140},
		"s0.0.0.0": {X: 100, Y: 100, W: 328, H: 108},
	}

	deck := shapeDeck(t)
	g := measured(t, deck, rects, nil)
	wantDiags(t, Diagnose(deck, g), "부모 <shape>를 아래로 4px 넘친다")

	// 좌표로 비켜 앉은 도형의 **자손**은 부모 변을 넘어서는 것이 저작 수단이다 (배지·꼬리).
	free := shapeDeck(t, soml.Attr{Name: "x", Value: "-16", HasValue: true})
	g = measured(t, free, rects, nil)
	if diags := Diagnose(free, g); len(diags) > 0 {
		t.Errorf("좌표를 쓴 요소의 경고 %d건, want 0건:\n%s", len(diags), diags)
	}
}

// `rowspan` 셀은 자기 행보다 높은 것이 정상이다 — 예외가 없으면 병합이 든 표마다 고칠 수
// 없는 경고가 나온다. 예외는 셀에만 있다.
func TestDiagnoseTableCellMayOutgrowItsRow(t *testing.T) {
	source := "slides/001_table" + soml.Ext
	doc, diags := soml.Parse(source, `<slide id="t">
  <table>
    <tr><th rowspan="2">구분</th><th>Q1</th></tr>
    <tr><th>Q2</th></tr>
  </table>
</slide>
`)
	if len(diags) != 0 {
		t.Fatalf("파서 진단이 없어야 한다: %v", diags)
	}
	stage, err := soml.Aspect16x9.Stage()
	if err != nil {
		t.Fatal(err)
	}
	deck := &soml.Deck{Aspect: soml.Aspect16x9, Stage: stage, Slides: []*soml.Slide{
		{Index: 0, ID: "t", Source: source, Content: doc.Content},
	}}
	if diags := core.Resolve(deck, nil); len(diags) > 0 {
		t.Fatalf("외관 해석 진단: %v", diags)
	}

	// 병합 셀(120px)이 자기 행(60px)을 넘는다. 공백 문자 노드가 자식 번호를 미므로 식별자 대신
	// Refs 순서로 준다.
	grid := []struct {
		el   string
		rect Rect
	}{
		{soml.TableElement, Rect{X: 64, Y: 64, W: 600, H: 120}},
		{soml.TableRowElement, Rect{X: 64, Y: 64, W: 600, H: 60}},
		{soml.TableHeaderCellElement, Rect{X: 64, Y: 64, W: 300, H: 120}}, // rowspan="2"
		{soml.TableHeaderCellElement, Rect{X: 364, Y: 64, W: 300, H: 60}},
		{soml.TableRowElement, Rect{X: 364, Y: 124, W: 300, H: 60}},
		{soml.TableHeaderCellElement, Rect{X: 364, Y: 124, W: 300, H: 60}},
	}
	refs := Refs(deck)
	if len(refs) != len(grid) {
		t.Fatalf("표의 요소 %d개, want %d개", len(refs), len(grid))
	}
	rects := make(map[string]Rect, len(refs))
	for i, ref := range refs {
		if ref.El != grid[i].el {
			t.Fatalf("%d번째 요소가 <%s>다, want <%s>", i, ref.El, grid[i].el)
		}
		rects[ref.ID] = grid[i].rect
	}
	g := measured(t, deck, rects, nil)
	if diags := Diagnose(deck, g); len(diags) > 0 {
		t.Errorf("병합 셀의 경고 %d건, want 0건:\n%s", len(diags), diags)
	}

	rects[refs[4].ID] = Rect{X: 364, Y: 124, W: 300, H: 100}
	rects[refs[5].ID] = Rect{X: 364, Y: 124, W: 300, H: 100}
	g = measured(t, deck, rects, nil)
	wantDiags(t, Diagnose(deck, g), "부모 <table>를 아래로 40px 넘친다")
}

// 잰 덱과 다른 덱이면 아무 말도 하지 않는다 — 그 어긋남은 이미 빌드를 멈추는 에러이고
// (newGeometry·Verify), 엉뚱한 요소를 가리키는 경고를 더하면 진짜 원인이 묻힌다.
func TestDiagnoseIsSilentOnForeignGeometry(t *testing.T) {
	deck := textDeck(t, 1)
	g := measured(t, deck, map[string]Rect{"s0.0": {X: 64, Y: 64, W: 1152, H: 800}}, nil)
	if diags := Diagnose(textDeck(t, 2), g); diags != nil {
		t.Errorf("다른 덱에 대한 진단 %d건, want 0건:\n%s", len(diags), diags)
	}
	if diags := Diagnose(nil, g); diags != nil {
		t.Errorf("덱 없는 진단 %d건, want 0건", len(diags))
	}
	if diags := Diagnose(deck, nil); diags != nil {
		t.Errorf("지오메트리 없는 진단 %d건, want 0건", len(diags))
	}
}

// 콘텐츠 박스는 스테이지에서 웹 렌더러의 여백을 뺀 사각형이다 — 16:9에서 1152×592.
func TestContentBox(t *testing.T) {
	stage, err := soml.Aspect16x9.Stage()
	if err != nil {
		t.Fatal(err)
	}
	if got, want := ContentBox(stage), (Rect{X: 64, Y: 64, W: 1152, H: 592}); got != want {
		t.Errorf("ContentBox = %s, want %s", got, want)
	}
}

// 사고 하나는 경고 하나다: 스테이지를 넘친 목록은 `<ul>` 한 줄로 말하고, 함께 벗어난 `<li>`들은 그
// 줄에 세어 붙인다. 항목마다 경고를 내면 한 곳을 고칠 일이 일곱 줄이 되고, 읽히지 않는 경고는 없는
// 것과 같다. 벗어나지 않은 항목은 세지 않는다.
func TestDiagnoseFoldsDescendantsIntoTheirAncestorsEscape(t *testing.T) {
	items := make([]soml.Node, 0, 4)
	for _, text := range []string{"하나", "둘", "셋", "넷"} {
		items = append(items, &soml.Element{Name: "li", Children: []soml.Node{chars(text)}})
	}
	deck := testDeck([]soml.Node{&soml.Element{Name: "ul", Children: items}})
	if diags := core.Resolve(deck, nil); len(diags) > 0 {
		t.Fatalf("외관 해석 진단: %v", diags)
	}
	// 목록이 스테이지 바닥(720)을 120px 넘는다. 첫 항목은 안에 있고 셋은 바닥 아래로 나갔다.
	rects := map[string]Rect{
		"s0.0":   {X: 64, Y: 440, W: 600, H: 400},
		"s0.0.0": {X: 64, Y: 440, W: 600, H: 100},
		"s0.0.1": {X: 64, Y: 540, W: 600, H: 100},
		"s0.0.2": {X: 64, Y: 640, W: 600, H: 100},
		"s0.0.3": {X: 64, Y: 740, W: 600, H: 100},
	}
	g := measured(t, deck, rects, nil)
	wantDiags(t, Diagnose(deck, g), "<ul>가 스테이지를 아래로 120px 벗어난다")
	if msg := Diagnose(deck, g)[0].Message; !strings.Contains(msg, "안의 <li> 2개도 함께 벗어난다") {
		t.Errorf("함께 벗어난 항목을 세지 않는다: %q", msg)
	}
}
