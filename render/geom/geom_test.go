package geom

import (
	"fmt"
	"math"
	"regexp"
	"strings"
	"testing"

	"github.com/k2sebeom/balpyo/core"
	"github.com/k2sebeom/balpyo/render/web"
	"github.com/k2sebeom/balpyo/soml"
)

// testDeck은 구조만 있는 덱이다. Refs는 외관을 보지 않으므로 해석하지 않는다.
func testDeck(slides ...[]soml.Node) *soml.Deck {
	stage, _ := soml.Aspect16x9.Stage()
	deck := &soml.Deck{Aspect: soml.Aspect16x9, Stage: stage}
	for i, content := range slides {
		deck.Slides = append(deck.Slides, &soml.Slide{
			Index:   i,
			ID:      fmt.Sprintf("slide%d", i),
			Source:  fmt.Sprintf("slides/%03d.slide", i+1),
			Content: content,
		})
	}
	return deck
}

// textElem은 `<text>` 요소다.
func textElem(children ...soml.Node) *soml.Element {
	return &soml.Element{Name: soml.TextElement, Children: children}
}

// chars는 문자 데이터 노드다.
func chars(s string) *soml.Text { return &soml.Text{Value: s} }

// shapeElem은 rect 도형이다.
func shapeElem(attrs []soml.Attr, children ...soml.Node) *soml.Element {
	return &soml.Element{
		Name:     soml.ShapeElement,
		Attrs:    append([]soml.Attr{{Name: "type", Value: "rect", HasValue: true}}, attrs...),
		Children: children,
	}
}

func refIDs(refs []Ref) string {
	var got []string
	for _, ref := range refs {
		got = append(got, ref.ID+"/"+ref.El)
	}
	return strings.Join(got, ",")
}

// 식별자는 덱 모델의 슬라이스 인덱스에서 나온다 (렌더 결과를 파싱하지 않는다). 측정 대상은
// 블록 종류·도형과 그 안 텍스트이고, 인라인·void는 건너뛴다 — PPTX에서 인라인은 박스가 아니라 런이다.
func TestRefs(t *testing.T) {
	tests := []struct {
		name string
		deck *soml.Deck
		want string
	}{
		{"콘텐츠가 없는 덱", testDeck(nil, nil), ""},
		{
			"슬라이드마다 자식 인덱스를 0부터 센다",
			testDeck([]soml.Node{textElem(chars("a")), textElem(chars("b"))}, []soml.Node{textElem()}),
			"s0.0/text,s0.1/text,s1.0/text",
		},
		{
			"문자 노드도 자식 인덱스를 차지한다",
			testDeck([]soml.Node{textElem(chars("앞"), textElem(chars("안")))}),
			"s0.0/text,s0.0.1/text",
		},
		{
			"인라인·void는 건너뛴다",
			testDeck([]soml.Node{
				&soml.Element{Name: "p", Children: []soml.Node{
					chars("배포 "),
					&soml.Element{Name: "strong", Children: []soml.Node{
						chars("자동"), &soml.Element{Name: "em", Children: []soml.Node{chars("화")}},
					}},
					&soml.Element{Name: "br"},
				}},
				&soml.Element{Name: "ul", Children: []soml.Node{
					&soml.Element{Name: "li", Children: []soml.Node{
						chars("항목 "), &soml.Element{Name: "code", Children: []soml.Node{chars("otel")}},
					}},
					&soml.Element{Name: "li", Children: []soml.Node{chars("둘째")}},
				}},
			}),
			"s0.0/p,s0.1/ul,s0.1.0/li,s0.1.1/li",
		},
		{
			"도형과 도형 안 텍스트는 둘 다 잰다",
			testDeck([]soml.Node{shapeElem(nil, textElem(chars("머리")), textElem(chars("본문"))), shapeElem(nil)}),
			"s0.0/shape,s0.0.0/text,s0.0.1/text,s0.1/shape",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := refIDs(Refs(tc.deck)); got != tc.want {
				t.Errorf("Refs = %s, want %s", got, tc.want)
			}
		})
	}
}

// 웹 렌더러가 쓴 식별자와 Refs가 갈라지면 좌표가 엉뚱한 요소에 붙는다 — 브라우저 없이 HTML에서
// 순서대로 뽑아 대조한다 (도형의 svg는 식별자를 갖지 않는다).
func TestRefsMatchRenderedHTML(t *testing.T) {
	deck := testDeck(
		[]soml.Node{textElem(chars("제목")), shapeElem(nil, textElem(chars("머리"))), shapeElem(nil)},
		[]soml.Node{textElem(chars("끝"))},
	)
	if diags := core.Resolve(deck, nil); len(diags) > 0 {
		t.Fatalf("외관 해석 진단: %v", diags)
	}
	files, err := web.New().Render(deck)
	if err != nil {
		t.Fatal(err)
	}
	pattern := regexp.MustCompile(regexp.QuoteMeta(web.IDAttr) + `="([^"]+)"`)
	var rendered, want []string
	for _, m := range pattern.FindAllStringSubmatch(string(files[web.IndexFile]), -1) {
		rendered = append(rendered, m[1])
	}
	for _, ref := range Refs(deck) {
		want = append(want, ref.ID)
	}
	if strings.Join(rendered, ",") != strings.Join(want, ",") {
		t.Errorf("HTML의 식별자 = %v, Refs = %v", rendered, want)
	}
}

// Ref는 배치 문맥(Parent·Free·Overlap)도 나른다 — 진단이 "이 이탈·겹침이 저자의 선택인가"를
// 묻는 사실이다. 진단이 덱을 따로 걸으면 두 순회가 갈라질 수 있다.
func TestRefsCarryPlacementFacts(t *testing.T) {
	shape := shapeElem
	// 첫 장: `<stack>`의 자식 둘 — 하나는 그냥 쌓이고 하나는 좌표로 비켜 앉는다.
	// 둘째 장: canvas 스테이지 — 직계 자식 전부가 좌표를 쓰는 자리다.
	deck := testDeck(
		[]soml.Node{&soml.Element{Name: soml.StackElement, Children: []soml.Node{
			shape(nil, textElem(chars("바닥"))),
			shape([]soml.Attr{{Name: "x", Value: "-16", HasValue: true}}, textElem(chars("배지"))),
		}}},
		[]soml.Node{shape(nil)},
	)
	deck.Slides[1].Layout = soml.LayoutCanvas
	if diags := core.Resolve(deck, nil); len(diags) > 0 {
		t.Fatalf("외관 해석 진단: %v", diags)
	}

	want := map[string]struct {
		parent        string
		free, overlap bool
	}{
		// 스택 자신은 흐름 안의 보통 박스다.
		"s0.0": {parent: "", free: false, overlap: false},
		// 스택의 직계는 한 칸에 쌓이므로 겹침을 묻지 않는다 — 좌표는 쓰지 않았다.
		"s0.0.0": {parent: "s0.0", free: false, overlap: true},
		// 도형 **안**은 다시 흐름이다: 겹침 문맥을 물려받지 않는다.
		"s0.0.0.0": {parent: "s0.0.0", free: false, overlap: false},
		// 좌표를 쓴 요소는 둘 다 참이다.
		"s0.0.1": {parent: "s0.0", free: true, overlap: true},
		// 그 **자손**은 부모 변을 넘어서는 선택을 물려받는다 (Free) — 겹침은 아니다.
		"s0.0.1.0": {parent: "s0.0.1", free: true, overlap: false},
		// canvas 슬라이드의 직계는 좌표를 쓰지 않아도 둘 다 참이다 (스테이지가 단일 셀 겹침이다).
		"s1.0": {parent: "", free: true, overlap: true},
	}
	refs := Refs(deck)
	if len(refs) != len(want) {
		t.Fatalf("Refs %d개, want %d개", len(refs), len(want))
	}
	for _, ref := range refs {
		fact, ok := want[ref.ID]
		if !ok {
			t.Errorf("뜻밖의 요소 %s <%s>", ref.ID, ref.El)
			continue
		}
		if ref.Parent != fact.parent || ref.Free != fact.free || ref.Overlap != fact.overlap {
			t.Errorf("%s <%s>: parent=%q free=%v overlap=%v, want parent=%q free=%v overlap=%v",
				ref.ID, ref.El, ref.Parent, ref.Free, ref.Overlap, fact.parent, fact.free, fact.overlap)
		}
	}
}

// 측정 스크립트는 측정 뷰를 켜고(회전 전 박스) 재고 finally에서 되돌린다 (web.MeasureView).
func TestMeasureExprTogglesMeasureView(t *testing.T) {
	expr := measureExpr()
	set, reset := strings.Index(expr, web.MeasureViewExpr()), strings.Index(expr, web.DocumentViewExpr())
	read := strings.Index(expr, "getBoundingClientRect")
	if set < 0 || reset < 0 || read < 0 {
		t.Fatalf("측정 스크립트에 뷰 전환이나 좌표 읽기가 없다:\n%s", expr)
	}
	if !(set < read && read < reset) {
		t.Errorf("순서가 어긋난다 — 켜기 %d, 읽기 %d, 되돌리기 %d", set, read, reset)
	}
	if at := strings.Index(expr, "} finally {"); at < 0 || at > reset {
		t.Errorf("되돌리기가 finally 안에 없다:\n%s", expr)
	}
}

// validMeasurement는 덱과 짝이 맞는 측정 결과다. 값은 검증 대상이 아니므로 단순하다.
func validMeasurement(deck *soml.Deck) measurement {
	m := measurement{DPR: 1}
	refs := Refs(deck)
	for i, slide := range deck.Slides {
		sm := slideMeasurement{
			Index: slide.Index,
			W:     float64(deck.Stage.Width),
			H:     float64(deck.Stage.Height),
		}
		for _, ref := range refs {
			if ref.Slide != i {
				continue
			}
			sm.Boxes = append(sm.Boxes, boxMeasurement{
				ID:    ref.ID,
				El:    ref.El,
				Rect:  Rect{X: 64, Y: 64, W: 1152, H: 36},
				Lines: []Rect{{X: 64, Y: 64, W: 900, H: 36}},
			})
		}
		m.Slides = append(m.Slides, sm)
	}
	return m
}

// 브라우저가 읽은 것과 덱 모델이 어긋나면 조용히 넘기지 않는다.
func TestNewGeometryRejectsMismatch(t *testing.T) {
	deck := testDeck(
		[]soml.Node{textElem(chars("제목")), textElem(chars("본문"))},
		[]soml.Node{textElem(chars("끝"))},
	)
	tests := []struct {
		name    string
		mutate  func(m *measurement)
		wantMsg string
	}{
		{"픽셀 비율이 1이 아니다", func(m *measurement) { m.DPR = 2 }, "devicePixelRatio"},
		{"슬라이드 수가 다르다", func(m *measurement) { m.Slides = m.Slides[:1] }, "슬라이드 수"},
		{"슬라이드 순서가 어긋났다", func(m *measurement) { m.Slides[1].Index = 7 }, "data-slide-index"},
		{"페이지가 스케일되어 있다", func(m *measurement) { m.Slides[0].W = 640; m.Slides[0].H = 360 }, "스케일"},
		{"요소가 덜 나왔다", func(m *measurement) { m.Slides[0].Boxes = m.Slides[0].Boxes[:1] }, "s0.1"},
		{
			name: "요소가 더 나왔다",
			mutate: func(m *measurement) {
				m.Slides[1].Boxes = append(m.Slides[1].Boxes, boxMeasurement{ID: "s1.1", El: "text"})
			},
			wantMsg: "s1.1",
		},
		{"식별자가 다르다", func(m *measurement) { m.Slides[0].Boxes[1].ID = "s0.9" }, "s0.9"},
		{"태그 이름이 다르다", func(m *measurement) { m.Slides[0].Boxes[0].El = "shape" }, "shape"},
		{"rect가 수치가 아니다", func(m *measurement) { m.Slides[0].Boxes[0].Rect.W = math.NaN() }, "유효하지 않다"},
		{"줄 박스가 무한대다", func(m *measurement) { m.Slides[0].Boxes[0].Lines[0].H = math.Inf(1) }, "유효하지 않다"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := validMeasurement(deck)
			tc.mutate(&m)
			g, err := newGeometry(deck, m)
			if err == nil {
				t.Fatalf("에러여야 한다 (지오메트리 = %+v)", g)
			}
			if !strings.Contains(err.Error(), tc.wantMsg) {
				t.Errorf("에러에 %q가 없다: %v", tc.wantMsg, err)
			}
		})
	}
}

func TestNewGeometryKeepsBoxesAndLines(t *testing.T) {
	deck := testDeck([]soml.Node{textElem(chars("제목"))}, []soml.Node{textElem(chars("끝"))})
	m := validMeasurement(deck)
	// 두 줄로 나뉜 요소 — 줄 박스가 그대로 남아야 배치 검사가 가로 넘침을 잰다.
	m.Slides[0].Boxes[0].Rect = Rect{X: 64, Y: 64, W: 1152, H: 72}
	m.Slides[0].Boxes[0].Lines = []Rect{
		{X: 64, Y: 64, W: 1100, H: 36},
		{X: 64, Y: 100, W: 300, H: 36},
	}

	g, err := newGeometry(deck, m)
	if err != nil {
		t.Fatal(err)
	}
	if g.Stage != deck.Stage {
		t.Errorf("Stage = %+v, want %+v", g.Stage, deck.Stage)
	}
	if len(g.Boxes) != 2 {
		t.Fatalf("박스 %d개, want 2개", len(g.Boxes))
	}
	box, ok := g.Box("s0.0")
	if !ok {
		t.Fatalf("s0.0을 찾을 수 없다 (%+v)", g.Boxes)
	}
	if box.Slide != 0 || box.El != soml.TextElement {
		t.Errorf("박스 = %+v", box)
	}
	if box.Rect != (Rect{X: 64, Y: 64, W: 1152, H: 72}) {
		t.Errorf("Rect = %s", box.Rect)
	}
	if len(box.Lines) != 2 {
		t.Errorf("Lines = %v, want 2줄", box.Lines)
	}
	if _, ok := g.Box("s9.9"); ok {
		t.Error("없는 식별자를 찾았다고 한다")
	}
}

// 소비 시점에도 대조한다 — 측정한 덱과 다른 덱으로 PPTX를 조립하면 좌표가 전부 어긋난다.
func TestVerifyRejectsForeignDeck(t *testing.T) {
	measured := testDeck([]soml.Node{textElem(chars("제목"))})
	g, err := newGeometry(measured, validMeasurement(measured))
	if err != nil {
		t.Fatal(err)
	}
	if err := g.Verify(measured); err != nil {
		t.Fatalf("측정한 덱은 통과해야 한다: %v", err)
	}

	other := testDeck([]soml.Node{textElem(chars("제목")), textElem(chars("본문"))})
	if err := g.Verify(other); err == nil {
		t.Error("요소가 늘어난 덱은 에러여야 한다")
	}

	stage43, _ := soml.Aspect4x3.Stage()
	scaled := testDeck([]soml.Node{textElem(chars("제목"))})
	scaled.Aspect, scaled.Stage = soml.Aspect4x3, stage43
	if err := g.Verify(scaled); err == nil {
		t.Error("스테이지가 다른 덱은 에러여야 한다")
	}
}
