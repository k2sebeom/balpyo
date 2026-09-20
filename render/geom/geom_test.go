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

// 측정 대상은 블록 종류(soml.KindOf의 카탈로그: text·블록·리스트 컨테이너·li)뿐이다.
// 인라인·void는 건너뛴다 — 여러 줄에 걸친 인라인의 rect는 뜻이 없고, PPTX에서 인라인은
// 박스가 아니라 런이다. 웹 렌더러가 같은 카탈로그로 식별자를 붙이므로(render/web) 이
// 대칭이 깨지면 Refs ↔ querySelectorAll 자리 맞대기가 런타임에 실패한다.
func TestRefsSkipInlineElements(t *testing.T) {
	p := &soml.Element{Name: "p", Children: []soml.Node{
		chars("배포 "),
		&soml.Element{Name: "strong", Children: []soml.Node{
			chars("자동"),
			&soml.Element{Name: "em", Children: []soml.Node{chars("화")}},
		}},
		&soml.Element{Name: "br"},
		&soml.Element{Name: "a", Children: []soml.Node{chars("링크")}},
	}}
	ul := &soml.Element{Name: "ul", Children: []soml.Node{
		&soml.Element{Name: "li", Children: []soml.Node{
			chars("항목 "),
			&soml.Element{Name: "code", Children: []soml.Node{chars("otel")}},
		}},
		&soml.Element{Name: "li", Children: []soml.Node{chars("둘째")}},
	}}
	refs := Refs(testDeck([]soml.Node{p, ul}))

	var ids, els []string
	for _, ref := range refs {
		ids = append(ids, ref.ID)
		els = append(els, ref.El)
	}
	if want := "s0.0,s0.1,s0.1.0,s0.1.1"; strings.Join(ids, ",") != want {
		t.Errorf("Refs = %v, want %s", ids, want)
	}
	if want := "p,ul,li,li"; strings.Join(els, ",") != want {
		t.Errorf("El = %v, want %s", els, want)
	}
}

// 식별자는 덱 모델의 슬라이스 인덱스에서 나온다 — 렌더 결과를 파싱하지 않고 만들 수 있어야
// 지오메트리 패스가 rect를 요소에 붙일 수 있다 (render/web 패키지 주석의 계약).
func TestRefsFollowDeckStructure(t *testing.T) {
	tests := []struct {
		name string
		deck *soml.Deck
		want []string
	}{
		{
			name: "콘텐츠가 없는 덱은 측정할 것이 없다",
			deck: testDeck(nil, nil),
			want: nil,
		},
		{
			name: "슬라이드마다 자식 인덱스를 0부터 센다",
			deck: testDeck(
				[]soml.Node{textElem(chars("a")), textElem(chars("b"))},
				[]soml.Node{textElem()},
			),
			want: []string{"s0.0", "s0.1", "s1.0"},
		},
		{
			name: "문자 노드도 자식 인덱스를 차지한다",
			deck: testDeck([]soml.Node{textElem(chars("앞"), textElem(chars("안")))}),
			want: []string{"s0.0", "s0.0.1"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			refs := Refs(tc.deck)
			var got []string
			for _, ref := range refs {
				got = append(got, ref.ID)
			}
			if strings.Join(got, ",") != strings.Join(tc.want, ",") {
				t.Fatalf("Refs = %v, want %v", got, tc.want)
			}
			for _, ref := range refs {
				if ref.El != soml.TextElement {
					t.Errorf("%s의 El = %q, want %q", ref.ID, ref.El, soml.TextElement)
				}
			}
		})
	}
}

// 지오메트리 패스가 만든 식별자와 웹 렌더러가 쓴 식별자가 갈라지면 좌표가 엉뚱한 요소에
// 붙는다. 브라우저 없이도 잡힌다 — HTML에서 순서대로 뽑아 Refs와 대조한다.
func TestRefsMatchRenderedHTML(t *testing.T) {
	deck := testDeck(
		[]soml.Node{textElem(chars("제목")), textElem(chars("본문"))},
		[]soml.Node{textElem(chars("끝"))},
	)
	if diags := core.Resolve(deck, nil); len(diags) > 0 {
		t.Fatalf("외관 해석 진단: %v", diags)
	}
	files, err := web.New().Render(deck)
	if err != nil {
		t.Fatal(err)
	}
	html := string(files[web.IndexFile])

	pattern := regexp.MustCompile(regexp.QuoteMeta(web.IDAttr) + `="([^"]+)"`)
	var rendered []string
	for _, m := range pattern.FindAllStringSubmatch(html, -1) {
		rendered = append(rendered, m[1])
	}
	var want []string
	for _, ref := range Refs(deck) {
		want = append(want, ref.ID)
	}
	if strings.Join(rendered, ",") != strings.Join(want, ",") {
		t.Errorf("HTML의 식별자 = %v, Refs = %v", rendered, want)
	}
}

// **도형과 도형 안 텍스트는 둘 다 잰다** — `<shape>`는 재는 것과 도형이 되는 것이 처음
// 일치하는 종류이고(잰 rect가 곧 `xfrm`), 도형 안 텍스트는 재지만 PPTX가 소비하지 않는다
// (`<li>`와 같은 처지 — 같은 도형의 `txBody` 문단이 된다). 웹 렌더러가 같은 카탈로그로
// 식별자를 붙이므로 두 목록의 자리가 맞는다 (아래 HTML 대조가 그 절반이다).
func TestRefsCollectShapesAndTheirText(t *testing.T) {
	shape := func(children ...soml.Node) *soml.Element {
		return &soml.Element{
			Name:     soml.ShapeElement,
			Attrs:    []soml.Attr{{Name: "type", Value: "rect", HasValue: true}},
			Children: children,
		}
	}
	deck := testDeck([]soml.Node{
		shape(textElem(chars("머리")), textElem(chars("본문"))),
		shape(),
	})
	if diags := core.Resolve(deck, nil); len(diags) > 0 {
		t.Fatalf("외관 해석 진단: %v", diags)
	}
	var got []string
	for _, ref := range Refs(deck) {
		got = append(got, ref.ID+"/"+ref.El)
	}
	want := []string{"s0.0/shape", "s0.0.0/text", "s0.0.1/text", "s0.1/shape"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("Refs = %v, want %v", got, want)
	}

	// 웹 렌더러가 낸 식별자와 순서가 같아야 자리 맞대기가 성립한다 — 그림 svg는 식별자를
	// 갖지 않으므로 목록에 끼어들지 않는다.
	files, err := web.New().Render(deck)
	if err != nil {
		t.Fatal(err)
	}
	pattern := regexp.MustCompile(regexp.QuoteMeta(web.IDAttr) + `="([^"]+)"`)
	var rendered []string
	for _, m := range pattern.FindAllStringSubmatch(string(files[web.IndexFile]), -1) {
		rendered = append(rendered, m[1])
	}
	if want := []string{"s0.0", "s0.0.0", "s0.0.1", "s0.1"}; strings.Join(rendered, ",") != strings.Join(want, ",") {
		t.Errorf("HTML의 식별자 = %v, want %v", rendered, want)
	}
}

// Ref는 좌표 말고 **배치 문맥**도 나른다 (Parent·Free·Overlap) — 배치 진단이 "이 이탈·겹침이
// 저자의 선택인가"를 물을 때 쓰는 사실이다 (layout.go, ADR 0039). 순회가 이미 덱을 걷고 있으므로
// 그 문맥은 여기서 나오는 것이 맞다: 진단이 덱을 한 번 더 걸으면 두 순회가 갈라질 수 있다.
func TestRefsCarryPlacementFacts(t *testing.T) {
	shape := func(attrs []soml.Attr, children ...soml.Node) *soml.Element {
		return &soml.Element{
			Name:     soml.ShapeElement,
			Attrs:    append([]soml.Attr{{Name: "type", Value: "rect", HasValue: true}}, attrs...),
			Children: children,
		}
	}
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
		// canvas 슬라이드의 직계는 좌표를 쓰지 않아도 둘 다 참이다 (PRD §13-3).
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

// 측정 스크립트는 **측정 뷰를 켜고 재고 되돌린다** — 회전 도형의 rect가 회전 전 박스여야
// 하기 때문이다 (web.MeasureView). 표현식을 리터럴로 쓰지 않는 계약(양쪽이 같은 문자열)과
// 순서(켜기 → 읽기 → finally 되돌리기)를 브라우저 없이 확인한다.
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
	// 되돌리기는 예외가 나도 실행되어야 한다 — 페이지가 측정 상태로 남으면 그 뒤의
	// 스크린샷이 회전 없는 그림을 찍는다.
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

// 브라우저가 읽은 것과 덱 모델이 어긋나면 조용히 넘기지 않는다 — 좌표가 엉뚱한 요소에 붙은
// PPTX는 열어 보기 전까지 정상으로 보인다.
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
		{
			name:    "픽셀 비율이 1이 아니다",
			mutate:  func(m *measurement) { m.DPR = 2 },
			wantMsg: "devicePixelRatio",
		},
		{
			name:    "슬라이드 수가 다르다",
			mutate:  func(m *measurement) { m.Slides = m.Slides[:1] },
			wantMsg: "슬라이드 수",
		},
		{
			name:    "슬라이드 순서가 어긋났다",
			mutate:  func(m *measurement) { m.Slides[1].Index = 7 },
			wantMsg: "data-slide-index",
		},
		{
			name:    "페이지가 스케일되어 있다",
			mutate:  func(m *measurement) { m.Slides[0].W = 640; m.Slides[0].H = 360 },
			wantMsg: "스케일",
		},
		{
			name:    "요소가 덜 나왔다",
			mutate:  func(m *measurement) { m.Slides[0].Boxes = m.Slides[0].Boxes[:1] },
			wantMsg: "s0.1",
		},
		{
			name: "요소가 더 나왔다",
			mutate: func(m *measurement) {
				m.Slides[1].Boxes = append(m.Slides[1].Boxes, boxMeasurement{ID: "s1.1", El: "text"})
			},
			wantMsg: "s1.1",
		},
		{
			name:    "식별자가 다르다",
			mutate:  func(m *measurement) { m.Slides[0].Boxes[1].ID = "s0.9" },
			wantMsg: "s0.9",
		},
		{
			name:    "태그 이름이 다르다",
			mutate:  func(m *measurement) { m.Slides[0].Boxes[0].El = "shape" },
			wantMsg: "shape",
		},
		{
			name:    "rect가 수치가 아니다",
			mutate:  func(m *measurement) { m.Slides[0].Boxes[0].Rect.W = math.NaN() },
			wantMsg: "유효하지 않다",
		},
		{
			name:    "줄 박스가 무한대다",
			mutate:  func(m *measurement) { m.Slides[0].Boxes[0].Lines[0].H = math.Inf(1) },
			wantMsg: "유효하지 않다",
		},
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
	// 두 줄로 나뉜 요소를 만들어 텍스트 메트릭 파생값을 검증한다.
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
	if box.LineCount() != 2 {
		t.Errorf("LineCount = %d, want 2", box.LineCount())
	}
	if got := box.MaxLineWidth(); got != 1100 {
		t.Errorf("MaxLineWidth = %v, want 1100", got)
	}
	// 여유 폭이 줄바꿈 임계값의 재료다: 다른 엔진의 글자 폭이 이 값을 넘어서면 줄이 하나
	// 늘어난다 (Iteration 1 리뷰 포인트, PRD §13-1).
	if got := box.Slack(); got != 52 {
		t.Errorf("Slack = %v, want 52", got)
	}
	if got := box.LineHeight(); got != 36 {
		t.Errorf("LineHeight = %v, want 36", got)
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
