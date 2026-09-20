package geom

import (
	"strings"
	"testing"

	"github.com/k2sebeom/balpyo/core"
	"github.com/k2sebeom/balpyo/soml"
)

// measured는 브라우저 없이 만든 지오메트리다 — 식별자마다 rect(와 줄 박스)를 손으로 준다.
//
// Synthetic을 쓰지 않는 이유: 배치 판정의 입력은 요소마다 **다른** rect다 (모든 박스가 같은
// 자리에 있으면 판정할 것이 없다). 대신 Synthetic과 같은 대조 경로를 지나게 하려고 만든 뒤
// Verify를 돌린다 — 잰 지오메트리와 같은 계약을 갖지 않는 입력으로 판정을 시험하면, 판정이
// 통과시킨 것이 실제 빌드에서도 통과한다고 말할 수 없다.
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

// 제자리에 있는 덱은 조용하다. 이것이 판정의 가장 중요한 성질이다 — 정상 덱에서 경고가 나오면
// 저자는 경고를 읽지 않게 되고, 그 순간 나머지 판정 전부를 잃는다 (ADR 0039).
func TestDiagnoseIsSilentOnAFittingDeck(t *testing.T) {
	deck := textDeck(t, 2)
	g := measured(t, deck, map[string]Rect{
		"s0.0": {X: 64, Y: 64, W: 1152, H: 200},
		// 형제는 아래에 놓인다 — 변이 맞닿아도(264) 겹침이 아니다.
		"s0.1": {X: 64, Y: 264, W: 1152, H: 100},
	}, nil)
	if diags := Diagnose(deck, g); len(diags) > 0 {
		t.Errorf("경고 %d건, want 0건:\n%s", len(diags), diags)
	}
}

// 스테이지 밖은 어떤 저작 의도로도 설명되지 않는다 — 좌표를 쓴 요소도 예외가 없다.
func TestDiagnoseStageEscape(t *testing.T) {
	deck := textDeck(t, 1)
	g := measured(t, deck, map[string]Rect{"s0.0": {X: 64, Y: 600, W: 1152, H: 180}}, nil)
	wantDiags(t, Diagnose(deck, g), "스테이지를 아래로 60px 벗어난다")

	// canvas 슬라이드의 요소는 Free지만 스테이지 이탈은 그대로 말한다.
	canvas := textDeck(t, 1)
	canvas.Slides[0].Layout = soml.LayoutCanvas
	g = measured(t, canvas, map[string]Rect{"s0.0": {X: 1200, Y: 64, W: 200, H: 100}}, nil)
	wantDiags(t, Diagnose(canvas, g), "스테이지를 오른쪽으로 120px 벗어난다")
}

// 콘텐츠 박스 이탈은 스테이지 여백 침범이다 — 슬라이드 직계만 받는다.
func TestDiagnoseContentBoxEscape(t *testing.T) {
	deck := textDeck(t, 1)
	g := measured(t, deck, map[string]Rect{"s0.0": {X: 64, Y: 64, W: 1152, H: 626}}, nil)
	wantDiags(t, Diagnose(deck, g), "콘텐츠 박스를 아래로 34px 넘친다")

	// 좌표를 쓴 요소는 여백을 벗어나는 선택까지 저자가 한 것이다 (ADR 0030·0031).
	canvas := textDeck(t, 1)
	canvas.Slides[0].Layout = soml.LayoutCanvas
	g = measured(t, canvas, map[string]Rect{"s0.0": {X: 0, Y: 0, W: 400, H: 100}}, nil)
	if diags := Diagnose(canvas, g); len(diags) > 0 {
		t.Errorf("canvas 요소의 여백 침범 경고 %d건, want 0건:\n%s", len(diags), diags)
	}
}

// 부모 박스 이탈이 "고정 크기 박스가 내용을 담지 못했다"의 모양이다 — 저자가 `h`를 준 도형에
// 글이 넘치면 자식 박스가 부모 변을 넘어간다.
func TestDiagnoseParentEscape(t *testing.T) {
	// 도형을 `<stack>`에 담는 이유는 `x`를 쓸 수 있는 자리가 거기여서다 (ADR 0031) — 판정에는
	// 스택이 끼어들지 않는다 (같은 rect를 주면 이탈이 없다).
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
	// 도형은 140px인데 내용은 108px이고 여백이 36px이다 — 본문의 마지막 줄이 도형의 아래 변을
	// 가로지른다 (testdata/fixtures/04_shape가 실제로 이 모양이었다).
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

// 표의 셀은 부모 행을 넘겨도 사고가 아니다 — `rowspan`이 든 셀은 자기 행보다 높은 것이
// 정상이다 (격자를 짜는 것은 브라우저의 표 알고리즘이다). 이 예외가 없으면 병합이 든 표마다
// 저자가 고칠 수 없는 경고가 나온다 (testdata/fixtures/09_table/slides/002_merge.slide).
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

	// 잰 격자를 손으로 준다 — 병합 셀(120px)이 자기 행(60px)의 아래 변을 60px 넘는다. 식별자를
	// 적지 않고 Refs의 순서로 주는 이유: 표 안의 공백 문자 노드가 자식 번호를 밀므로(web.ChildPath)
	// 손으로 적은 식별자는 마크업을 한 줄 고칠 때마다 틀린다.
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

	// 예외는 셀에만 있다 — 행이 프레임을 넘치는 것은 격자가 설명하지 않는다.
	rects[refs[4].ID] = Rect{X: 364, Y: 124, W: 300, H: 100}
	rects[refs[5].ID] = Rect{X: 364, Y: 124, W: 300, H: 100}
	g = measured(t, deck, rects, nil)
	wantDiags(t, Diagnose(deck, g), "부모 <table>를 아래로 40px 넘친다")
}

// 글줄의 가로 이탈은 줄바꿈이 불가능한 글자다 (긴 URL·`<pre>`). 세로는 보지 않는다 — 글자
// 상자가 CSS 줄 박스보다 커서 정상 텍스트도 늘 몇 px 넘치기 때문이다 (ADR 0039).
func TestDiagnoseTextEscape(t *testing.T) {
	deck := textDeck(t, 1)
	box := Rect{X: 64, Y: 64, W: 400, H: 40}
	g := measured(t, deck, map[string]Rect{"s0.0": box},
		map[string][]Rect{"s0.0": {{X: 64, Y: 66, W: 520, H: 36}}})
	wantDiags(t, Diagnose(deck, g), "글줄이 박스를 오른쪽으로 120px 넘친다")

	// 세로로 넘친 줄 박스는 아무 말도 만들지 않는다 (박스 40px 안의 글자 상자 46px).
	g = measured(t, deck, map[string]Rect{"s0.0": box},
		map[string][]Rect{"s0.0": {{X: 64, Y: 64, W: 300, H: 46}}})
	if diags := Diagnose(deck, g); len(diags) > 0 {
		t.Errorf("세로 글줄 경고 %d건, want 0건:\n%s", len(diags), diags)
	}
}

// 형제 겹침은 같은 슬라이드·같은 부모 안에서만 본다. 슬라이드를 빼면 좌표가 슬라이드
// 상대이므로 모든 장의 첫 요소가 서로 겹친 것으로 보인다 (ADR 0039의 거짓 양성).
func TestDiagnoseSiblingOverlap(t *testing.T) {
	deck := textDeck(t, 2)
	g := measured(t, deck, map[string]Rect{
		"s0.0": {X: 64, Y: 64, W: 600, H: 200},
		"s0.1": {X: 500, Y: 200, W: 600, H: 200},
	}, nil)
	// 겹침은 나중에 온 형제에 붙는다 — 저자가 읽는 순서에서 "이것이 얹혔다"가 자연스럽다.
	wantDiags(t, Diagnose(deck, g), "형제 <text>와 164px×64px 겹친다")

	// 같은 좌표라도 다른 슬라이드면 겹침이 아니다.
	twoSlides := textDeck(t, 1, 1)
	g = measured(t, twoSlides, map[string]Rect{
		"s0.0": {X: 64, Y: 64, W: 600, H: 200},
		"s1.0": {X: 64, Y: 64, W: 600, H: 200},
	}, nil)
	if diags := Diagnose(twoSlides, g); len(diags) > 0 {
		t.Errorf("슬라이드가 다른 요소의 겹침 경고 %d건, want 0건:\n%s", len(diags), diags)
	}

	// `<stack>`의 자식은 **겹치도록** 쌓은 것이다 (ADR 0028·0029).
	stack := testDeck([]soml.Node{&soml.Element{Name: soml.StackElement, Children: []soml.Node{
		textElem(chars("바닥")), textElem(chars("위")),
	}}})
	if diags := core.Resolve(stack, nil); len(diags) > 0 {
		t.Fatalf("외관 해석 진단: %v", diags)
	}
	g = measured(t, stack, map[string]Rect{
		"s0.0":   {X: 64, Y: 64, W: 600, H: 200},
		"s0.0.0": {X: 64, Y: 64, W: 600, H: 200},
		"s0.0.1": {X: 64, Y: 64, W: 600, H: 200},
	}, nil)
	if diags := Diagnose(stack, g); len(diags) > 0 {
		t.Errorf("stack 자식의 겹침 경고 %d건, want 0건:\n%s", len(diags), diags)
	}
}

// 요소 하나가 받는 이탈 진단은 하나다 — 스테이지를 벗어난 박스는 여백도 침범하지만, 같은
// 사고를 두 이름으로 부르면 저자는 두 줄을 읽고 한 곳을 고친다.
func TestDiagnoseReportsOneEscapePerElement(t *testing.T) {
	deck := textDeck(t, 1)
	g := measured(t, deck, map[string]Rect{"s0.0": {X: 64, Y: 64, W: 1152, H: 800}}, nil)
	wantDiags(t, Diagnose(deck, g), "스테이지를 아래로")
}

// 1px 미만은 저자가 고칠 수 있는 사고가 아니다 — 반올림 경계마다 경고가 나면 경고를 읽지
// 않게 된다. 이 허용 오차 덕분에 메시지의 px는 늘 1 이상이다.
func TestDiagnoseToleratesSubPixelEscape(t *testing.T) {
	deck := textDeck(t, 2)
	g := measured(t, deck, map[string]Rect{
		// 콘텐츠 박스를 0.5px 넘긴 박스(바닥 656.5)와,
		"s0.0": {X: 64, Y: 64, W: 576.5, H: 592.5},
		// 그것과 0.5px 물린 형제다 (왼쪽 640 < 640.5).
		"s0.1": {X: 640, Y: 64, W: 576, H: 200},
	}, nil)
	if diags := Diagnose(deck, g); len(diags) > 0 {
		t.Errorf("경고 %d건, want 0건:\n%s", len(diags), diags)
	}
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

// 콘텐츠 박스는 **잰 값**이다 — 여백을 테마가 레이아웃마다 정하기 때문이다.
//
// 이 판정이 상수(web.StagePadding)로 돌던 동안 minimal의 전면 배치 레이아웃(`image-full`은
// `padding: 0`이다)에서 그림 한 장이 경고 두 줄을 받았고, 저자가 고칠 방법이 없었다 — 여백을
// 없앤 것이 테마의 결정이므로 소스에는 고칠 곳이 없다. 고칠 수 없는 경고는 나머지 판정 전부를
// 함께 잃게 만든다 (ADR 0039의 전제).
func TestDiagnoseUsesTheMeasuredContentBox(t *testing.T) {
	// 전면 배치: 스테이지를 꽉 채운 박스는 조용하다.
	full := textDeck(t, 1)
	g := measured(t, full, map[string]Rect{"s0.0": {X: 0, Y: 0, W: 1280, H: 720}}, nil)
	g.Content = []Rect{{X: 0, Y: 0, W: 1280, H: 720}}
	if diags := Diagnose(full, g); len(diags) > 0 {
		t.Errorf("여백 없는 레이아웃의 경고 %d건, want 0건:\n%s", len(diags), diags)
	}

	// 여백이 기본보다 넓은 레이아웃(96px)에서는 기본 여백에 맞춘 박스가 경고를 받는다 —
	// 판정이 상수가 아니라 잰 박스를 본다는 사실의 반대쪽 증거다.
	wide := textDeck(t, 1)
	g = measured(t, wide, map[string]Rect{"s0.0": {X: 64, Y: 96, W: 1152, H: 528}}, nil)
	g.Content = []Rect{{X: 96, Y: 96, W: 1088, H: 528}}
	wantDiags(t, Diagnose(wide, g), "콘텐츠 박스를 오른쪽으로 32px 넘친다")

	// 잰 값이 없는 지오메트리(Synthetic·JSON에서 되읽은 값)는 기본 여백으로 물러선다 —
	// 0×0 콘텐츠 박스로 모든 박스가 경고를 받으면 진단 전부를 잃는다.
	plain := textDeck(t, 1)
	g = measured(t, plain, map[string]Rect{"s0.0": {X: 64, Y: 64, W: 1152, H: 626}}, nil)
	if len(g.Content) != 0 {
		t.Fatalf("시험 입력에 잰 콘텐츠 박스가 있다: %v", g.Content)
	}
	wantDiags(t, Diagnose(plain, g), "콘텐츠 박스를 아래로 34px 넘친다")
}

// 슬라이드마다 다른 콘텐츠 박스를 쓴다 — 덱 하나에 여백이 한 값이 아니다 (전면 배치 한 장과
// 보통 레이아웃 한 장이 같은 덱에 있는 것이 정상이다).
func TestDiagnoseContentBoxIsPerSlide(t *testing.T) {
	deck := textDeck(t, 1, 1)
	g := measured(t, deck, map[string]Rect{
		// 첫 장은 전면 배치다 — 스테이지를 꽉 채워도 조용하다.
		"s0.0": {X: 0, Y: 0, W: 1280, H: 720},
		// 둘째 장은 기본 여백이다 — 같은 박스가 경고를 받는다.
		"s1.0": {X: 0, Y: 0, W: 1280, H: 720},
	}, nil)
	g.Content = []Rect{{X: 0, Y: 0, W: 1280, H: 720}, ContentBox(deck.Stage)}
	wantDiags(t, Diagnose(deck, g), "콘텐츠 박스를 아래로 64px 넘친다")
}
