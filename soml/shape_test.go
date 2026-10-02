package soml

import "testing"

// 도형의 정상 트리 — 속성 넷(고유·외관·박스·자식 배치)과 자식 어휘가 파싱을 그대로 통과한다.
// 파서는 값을 보지 않는다 — 12종 밖의 type이나 type 없는 도형도 진단은 값 층의 몫이다.
func TestParseShape(t *testing.T) {
	runTreeCases(t, []treeCase{
		{
			name: "고유·외관·박스 속성",
			src: `<slide><shape type="rounded-rect" id="db" variant="card-header" fill="surface" border="1 solid muted" ` +
				`radius="lg" shadow="soft" opacity="0.8" rotate="45" flip="h" pad="md" w="fill" h="220" ratio="1:1"/></slide>`,
			want: `shape[type=rounded-rect id=db variant=card-header fill=surface border=1 solid muted ` +
				`radius=lg shadow=soft opacity=0.8 rotate=45 flip=h pad=md w=fill h=220 ratio=1:1]`,
		},
		{
			name: "자식 배치 속성과 style — 컨테이너의 자식",
			src:  `<slide><grid cols="4"><shape type="rect" span="2" grow="1" self="center" style="order: 2"/></grid></slide>`,
			want: `grid[cols=4](shape[type=rect span=2 grow=1 self=center style=order: 2])`,
		},
		{
			// 개수를 문법으로 세지 않는다 — 자식 여럿은 PPTX에서 txBody의 문단 여럿이다.
			name: "도형 안 텍스트·블록·인라인",
			src:  `<slide><shape type="rect"><text anchor="end">배지</text><h3 anchor="center">제목</h3><p>배포 <strong>자동화</strong></p></shape></slide>`,
			want: `shape[type=rect](text[anchor=end]("배지") h3[anchor=center]("제목") p("배포 " strong("자동화")))`,
		},
		{
			// 컨테이너가 있으면 도형은 배경이 된다 (ADR 0049) — 컨테이너 넷이 열리고, 그림·리스트·
			// 중첩 도형은 컨테이너를 거쳐 들어온다.
			name: "도형 안 컨테이너 넷과 그 안의 리스트·도형",
			src: `<slide><shape type="rect"><text>헤더</text><row><text>a</text></row><col><ul><li>항목</li></ul><shape type="ellipse"/></col>` +
				`<grid cols="2"><text>c</text></grid><stack><text>d</text></stack></shape></slide>`,
			want: `shape[type=rect](text("헤더") row(text("a")) col(ul(li("항목")) shape[type=ellipse]) grid[cols=2](text("c")) stack(text("d")))`,
		},
		{
			name: "빈 도형 — 자기 닫힘, 짝 태그, type 없음, 12종 밖",
			src:  `<slide><shape type="rect"/><shape type="ellipse"></shape><shape/><shape type="dodecagon"/></slide>`,
			want: `shape[type=rect] shape[type=ellipse] shape shape[type=dodecagon]`,
		},
	})
}

// 도형의 배치 위반과 닫힌 속성 — 직속 자식은 텍스트와 컨테이너뿐이다 (ADR 0049 결정 4).
func TestParseShapeErrors(t *testing.T) {
	runDiagCases(t, []diagCase{
		{
			name: "도형 안의 문자 데이터",
			src:  `<slide><shape type="rect">배지</shape></slide>`,
			want: []string{"1:27 <shape> 안에 문자 데이터를 직접 쓸 수 없다 (<text>로 감싼다)"},
		},
		{
			// 진단의 태그 목록에 컨테이너 넷이 실려 고칠 방법이 보인다.
			name: "도형 안의 리스트",
			src:  `<slide><shape type="rect"><ul><li>항목</li></ul></shape></slide>`,
			want: []string{"1:28 <shape>의 자식으로 <ul>는 올 수 없다 (쓸 수 있는 태그: <text>, <h1>, <h2>, <h3>, <p>, <blockquote>, <pre>, <row>, <col>, <grid>, <stack>)"},
		},
		{name: "도형 안의 spacer", src: `<slide><shape type="rect"><spacer/></shape></slide>`, want: []string{"1:28 <shape>의 자식으로 <spacer>는 올 수 없다"}},
		{name: "도형 안의 도형", src: `<slide><shape type="rect"><shape type="ellipse"/></shape></slide>`, want: []string{"1:28 <shape>의 자식으로 <shape>는 올 수 없다"}},
		{name: "도형 안의 note", src: `<slide><shape type="rect"><note>메모</note></shape></slide>`, want: []string{"1:28 <shape>의 자식으로 <note>는 올 수 없다"}},
		{name: "도형 안의 li", src: `<slide><shape type="rect"><li>항목</li></shape></slide>`, want: []string{"1:28 <li>는 <ul>/<ol>의 자식으로만 쓸 수 있다"}},
		{name: "도형 안의 인라인", src: `<slide><shape type="rect"><strong>굵게</strong></shape></slide>`, want: []string{"1:28 <shape>의 자식으로 <strong>는 올 수 없다"}},
		{
			// stroke/cap/dash는 선의 것이다 — 도형에 받아 놓고 버리지 않는다.
			name: "도형에 stroke는 없다",
			src:  `<slide><shape type="rect" stroke="2 accent1"/></slide>`,
			want: []string{`1:27 <shape>에 알 수 없는 속성 stroke (쓸 수 있는 속성: type, id, alt, variant, fill, border, radius, shadow, opacity, rotate, flip, pad, w, h, min-w, min-h, max-w, max-h, ratio, grow, shrink, basis, span, self, order, at, x, y, style, slot)`},
		},
	})
}

// 모드 판정은 **어휘 층의 술어 하나**다 (ADR 0049 결정 3) — 값 층·웹·PPTX가 같은 함수를 읽어야
// 세 층이 갈라질 방법이 없다. 모드는 **직계 자식**이 정한다.
func TestShapeHoldsLayout(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want bool
	}{
		{name: "빈 도형", src: `<slide><shape type="rect"/></slide>`, want: false},
		{name: "텍스트만 — 흡수 모드", src: `<slide><shape type="rect"><text>배지</text><p>본문</p></shape></slide>`, want: false},
		{name: "컨테이너 하나 — 배경 모드", src: `<slide><shape type="rect"><col><p>본문</p></col></shape></slide>`, want: true},
		{name: "섞이면 배경 모드", src: `<slide><shape type="rect"><text>헤더</text><row><p>왼</p></row></shape></slide>`, want: true},
		{name: "손자 컨테이너는 모드를 바꾸지 않는다", src: `<slide><row><shape type="rect"><p>본문</p></shape></row></slide>`, want: false},
		{name: "빈 컨테이너도 배경 모드", src: `<slide><shape type="rect"><col></col></shape></slide>`, want: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			shape := findShape(parseOK(t, tc.src).Content)
			if shape == nil {
				t.Fatal("트리에 <shape>가 없다")
			}
			if got := ShapeHoldsLayout(shape); got != tc.want {
				t.Errorf("ShapeHoldsLayout() = %v, want %v", got, tc.want)
			}
		})
	}
	// 호출자가 문자 데이터 노드를 걸러 내지 않고 물어볼 수 있어야 한다.
	if ShapeHoldsLayout(nil) {
		t.Error("ShapeHoldsLayout(nil) = true")
	}
}

// findShape는 트리에서 첫 `<shape>`를 찾는다 (깊이 우선).
func findShape(nodes []Node) *Element {
	for _, node := range nodes {
		if elem, ok := node.(*Element); ok {
			if elem.Name == ShapeElement {
				return elem
			}
			if found := findShape(elem.Children); found != nil {
				return found
			}
		}
	}
	return nil
}
