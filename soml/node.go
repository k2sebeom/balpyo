package soml

import "strings"

// Element는 요소 노드다 — `<text role="title">제목</text>` 하나.
//
// 속성 값은 **해석하지 않고 원문 문자열로** 담는다. 파서는 문법과 태그·속성 **이름**만 보고,
// 값의 의미(role 어휘, size 수치, color 토큰)는 core가 본다. 값 스팬을 함께 남기므로 해석
// 층의 진단도 파서와 같은 정밀도로 line:col을 가리킨다 — 진단의 위치 정밀도가 층을 내려갈수록
// 나빠지면 "에디터에서 본 진단 = CI가 막는 진단"이 성립하지 않는다.
type Element struct {
	Name string
	// NameSpan은 태그 이름 구간이다 (`<`는 제외) — 태그 어휘 진단이 가리키는 곳.
	NameSpan Span
	Attrs    []Attr
	Children []Node
	// Span은 여는 `<`부터 닫는 태그의 `>`까지다. 닫히지 않은 요소는 여는 태그에서 멈춘다.
	Span Span
	// Appearance는 core.Resolve가 채운 확정 외관이다 (appearance.go). 파서는 건드리지 않고,
	// **렌더러는 여기만 읽는다** — Attrs를 다시 해석하지 않는다.
	Appearance *Appearance
	// Authored는 값 층이 펼친 **합성 노드**에서 저자가 쓴 태그 이름이다 (`<chart>`의 막대와 값
	// 라벨은 "point"). 저자가 쓴 노드는 빈 값이다. 배치 진단이 저자가 쓰지 않은 태그(`<text>`)를
	// 말하지 않게 하는 자리이고, 렌더러는 읽지 않는다 — 그려지는 것은 Name의 종류다.
	Authored string
}

// AuthoredName은 진단이 이 요소를 부를 태그 이름이다 — 합성 노드면 저자가 쓴 태그, 아니면 Name.
func (e *Element) AuthoredName() string {
	if e.Authored != "" {
		return e.Authored
	}
	return e.Name
}

// NodeSpan은 Node 계약이다.
func (e *Element) NodeSpan() Span { return e.Span }

// Attr는 이름으로 속성을 찾는다. 중복 속성은 파싱 시 진단되고 첫 번째만 남으므로
// 이 조회는 모호하지 않다.
func (e *Element) Attr(name string) (Attr, bool) {
	for _, attr := range e.Attrs {
		if attr.Name == name {
			return attr, true
		}
	}
	return Attr{}, false
}

// TextContent는 이 요소 아래의 문자 데이터를 이어 붙인다. 공백은 이미 접힌 상태다 (Parse).
func (e *Element) TextContent() string {
	var b strings.Builder
	for _, child := range e.Children {
		switch node := child.(type) {
		case *Text:
			b.WriteString(node.Value)
		case *Element:
			b.WriteString(node.TextContent())
		}
	}
	return b.String()
}

// Attr는 속성 하나다.
type Attr struct {
	Name     string
	NameSpan Span
	// Value는 인용 부호를 벗긴 원문이다. 엔티티·이스케이프 해석은 없다.
	Value string
	// ValueSpan은 인용 부호를 **제외한** 값 구간이다 — `role="titel"`의 진단이 부호가 아니라
	// `titel`을 가리켜야 한다.
	ValueSpan Span
	// HasValue는 `=`와 값이 있었는지다. `<slide skip>`처럼 값 없는 속성은 false다.
	HasValue bool
}

// Text는 문자 데이터 노드다.
type Text struct {
	// Value는 공백이 접힌 텍스트다 (접기 규칙은 Parse의 주석 참조).
	Value string
	Span  Span
}

// NodeSpan은 Node 계약이다.
func (t *Text) NodeSpan() Span { return t.Span }
