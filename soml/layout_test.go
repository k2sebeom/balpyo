package soml

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// 레이아웃 태그의 정상 트리 — 태그·컨테이너 속성·자식 배치 속성·중첩이 파싱을 그대로
// 통과한다. dumpNodes의 표현은 subset_test.go와 공유한다.
func TestParseLayout(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want string
	}{
		{
			name: "row와 col 중첩",
			src:  `<slide><row gap="lg"><col><p>왼쪽</p></col><col><p>오른쪽</p></col></row></slide>`,
			want: `row[gap=lg](col(p("왼쪽")) col(p("오른쪽")))`,
		},
		{
			name: "컨테이너 배치 속성",
			src:  `<slide><row gap="xl" align="center" justify="between" wrap="true" pad="md"><text>a</text></row></slide>`,
			want: `row[gap=xl align=center justify=between wrap=true pad=md](text("a"))`,
		},
		{
			name: "grid의 트랙 목록",
			src:  `<slide><grid cols="4" rows="2fr 1fr" gap="lg"><text>카드</text></grid></slide>`,
			want: `grid[cols=4 rows=2fr 1fr gap=lg](text("카드"))`,
		},
		{
			name: "크기와 비율",
			src:  `<slide><col w="fill" h="320" min-w="200" min-h="120" max-w="900" max-h="480" ratio="16:9"><p>x</p></col></slide>`,
			want: `col[w=fill h=320 min-w=200 min-h=120 max-w=900 max-h=480 ratio=16:9](p("x"))`,
		},
		{
			name: "spacer는 void다",
			src:  `<slide><row><text>a</text><spacer/><text>b</text></row></slide>`,
			want: `row(text("a") spacer text("b"))`,
		},
		{
			name: "슬라이드 직속 spacer — 세로 스택의 밀어내기",
			src:  `<slide><h2>제목</h2><spacer/><p>바닥</p></slide>`,
			want: `h2("제목") spacer p("바닥")`,
		},
		{
			name: "spacer도 자식 배치 속성을 받는다",
			src:  `<slide><row><spacer grow="2" shrink="0"/></row></slide>`,
			want: `row(spacer[grow=2 shrink=0])`,
		},
		{
			name: "자식 배치 속성 — text",
			src:  `<slide><row><text grow="1" self="center" order="2">a</text></row></slide>`,
			want: `row(text[grow=1 self=center order=2]("a"))`,
		},
		{
			name: "자식 배치 속성 — 블록",
			src:  `<slide><row><p shrink="0" basis="240">a</p></row></slide>`,
			want: `row(p[shrink=0 basis=240]("a"))`,
		},
		{
			name: "자식 배치 속성 — 리스트 컨테이너",
			src:  `<slide><row><ul basis="360"><li>항목</li></ul></row></slide>`,
			want: `row(ul[basis=360](li("항목")))`,
		},
		{
			name: "자식 배치 속성 — 중첩 컨테이너",
			src:  `<slide><row><col grow="2" span="2"><p>a</p></col></row></slide>`,
			want: `row(col[grow=2 span=2](p("a")))`,
		},
		{
			// 블록의 배치 규칙이 넓어졌다: "슬라이드 직속"이 "슬라이드 또는 레이아웃
			// 컨테이너 직속"이 된다. 리스트도 함께 컨테이너의 자식이 될 수 있다.
			name: "컨테이너의 자식 어휘 — text·블록·리스트·중첩·spacer",
			src:  `<slide><col><text>a</text><h1>b</h1><pre>c</pre><ol><li>d</li></ol><row><p>e</p></row><spacer/></col></slide>`,
			want: `col(text("a") h1("b") pre("c") ol(li("d")) row(p("e")) spacer)`,
		},
		{
			name: "빈 컨테이너는 유효하다",
			src:  `<slide><row></row><col/></slide>`,
			want: `row col`,
		},
		{
			name: "style 탈출구",
			src:  `<slide><row style="align-content: space-evenly"><p style="margin-inline-start: 40px">a</p></row></slide>`,
			want: `row[style=align-content: space-evenly](p[style=margin-inline-start: 40px]("a"))`,
		},
		{
			// stack은 단일 셀 겹침 컨테이너다 — align/justify는 셀 안 모든 자식의 기본 정렬.
			name: "stack의 정렬 속성",
			src:  `<slide><stack align="center" justify="center"><text>배지</text></stack></slide>`,
			want: `stack[align=center justify=center](text("배지"))`,
		},
		{
			// at은 자식 하나가 셀 안에서 자기 앵커를 정한다 (place-self) — 이름만 파서가 열고
			// 9지점 값 어휘는 값 층이 본다.
			name: "stack 자식의 at 앵커",
			src:  `<slide><stack><shape type="rect" at="top-right"/><text at="center">글자</text></stack></slide>`,
			want: `stack(shape[type=rect at=top-right] text[at=center]("글자"))`,
		},
		{
			// x/y는 절대 오프셋이다 — 부호·단위·백분율의 값 문법은 값 층이 본다.
			name: "stack 자식의 x·y 절대 배치",
			src:  `<slide><stack><image src="assets/dot.png" x="20" y="-10"/><shape type="star" x="50%" y="0"/></stack></slide>`,
			want: `stack(image[src=assets/dot.png x=20 y=-10] shape[type=star x=50% y=0])`,
		},
		{
			// 겹침이 흐름 배치와 만나는 장면 — grid 셀 안의 stack, row 안의 stack.
			name: "stack이 컨테이너의 자식",
			src:  `<slide><grid cols="2"><stack><text>a</text></stack></grid><row><stack><text>b</text></stack></row></slide>`,
			want: `grid[cols=2](stack(text("a"))) row(stack(text("b")))`,
		},
		{
			name: "stack 안의 stack — 중첩 겹침",
			src:  `<slide><stack><stack><text>안쪽</text></stack></stack></slide>`,
			want: `stack(stack(text("안쪽")))`,
		},
		{
			name: "깊은 중첩 — grid 안의 row 안의 col",
			src:  `<slide><grid cols="2"><row><col><p>a</p></col></row></grid></slide>`,
			want: `grid[cols=2](row(col(p("a"))))`,
		},
		{
			name: "컨테이너 안의 인라인 서식은 블록을 지난다",
			src:  `<slide><row><p>배포 <strong>자동화</strong></p></row></slide>`,
			want: `row(p("배포 " strong("자동화")))`,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			doc := parseOK(t, tc.src)
			if got := dumpNodes(doc.Content); got != tc.want {
				t.Errorf("트리 = %s\nwant %s", got, tc.want)
			}
		})
	}
}

// 요소의 Span은 여는 `<`부터 닫는 태그의 `>`까지다 — 컨테이너와 void spacer도 같은 계약을
// 지켜야 진단·지오메트리가 저자의 태그를 가리킨다.
func TestParseLayoutSpans(t *testing.T) {
	doc := parseOK(t, "<slide>\n  <row gap=\"md\">\n    <spacer/>\n  </row>\n</slide>")
	row := elementAt(t, doc.Content, 0)
	if got, want := row.Span.Start.String()+"-"+row.Span.End.String(), "2:3-4:9"; got != want {
		t.Errorf("<row> Span = %s, want %s", got, want)
	}
	spacer := elementAt(t, row.Children, 0)
	if got, want := spacer.Span.Start.String()+"-"+spacer.Span.End.String(), "3:5-3:14"; got != want {
		t.Errorf("<spacer> Span = %s, want %s", got, want)
	}
}

// 에러 케이스 전수 — 진단의 line:col과 문구까지 단정한다.
func TestParseLayoutErrors(t *testing.T) {
	tests := []struct {
		name string
		src  string
		// want의 각 원소는 "line:col 메시지조각"이다.
		want []string
	}{
		{
			name: "spacer는 자식을 가질 수 없다",
			src:  `<slide><spacer><text>a</text></spacer></slide>`,
			want: []string{"1:9 void 태그 <spacer>은 내용을 가질 수 없다 (<spacer/>로 쓴다)"},
		},
		{
			name: "spacer는 문자 데이터를 가질 수 없다",
			src:  `<slide><spacer>여백</spacer></slide>`,
			want: []string{"1:9 void 태그 <spacer>은 내용을 가질 수 없다"},
		},
		{
			name: "컨테이너 안의 문자 데이터",
			src:  `<slide><row>글자</row></slide>`,
			want: []string{"1:13 <row> 안에 문자 데이터를 직접 쓸 수 없다 (<text>로 감싼다)"},
		},
		{
			// 노트는 콘텐츠가 아니라 슬라이드 메타데이터다 (SlideDoc.Notes). 목록에 <shape>가
			// 들어온 것은 Iteration 4, <image>는 Iteration 5다 — 컨테이너 안 도형이 카드
			// 그리드의 핵심 장면이고, 컨테이너 안 그림이 그림+텍스트 2단의 핵심 장면이다.
			name: "컨테이너 안의 note",
			src:  `<slide><row><note>메모</note></row></slide>`,
			want: []string{"1:14 <row>의 자식으로 <note>는 올 수 없다 (쓸 수 있는 태그: <text>, <shape>, <image>, <h1>, <h2>, <h3>, <p>, <blockquote>, <pre>, <ul>, <ol>, <row>, <col>, <grid>, <stack>, <spacer>)"},
		},
		{
			name: "컨테이너 안의 li",
			src:  `<slide><col><li>항목</li></col></slide>`,
			want: []string{"1:14 <li>는 <ul>/<ol>의 자식으로만 쓸 수 있다"},
		},
		{
			// <li>는 리스트 컨테이너의 자식이지 flex 항목이 아니다 (childPlacementAttrs).
			name: "li에는 자식 배치 속성이 없다",
			src:  `<slide><ul><li grow="1">x</li></ul></slide>`,
			want: []string{"1:16 <li>에는 속성을 쓸 수 없다 (grow)"},
		},
		{
			// 인라인은 박스가 아니라 런이다.
			name: "인라인에는 자식 배치 속성이 없다",
			src:  `<slide><p><strong order="1">x</strong></p></slide>`,
			want: []string{"1:19 <strong>에는 속성을 쓸 수 없다 (order)"},
		},
		{
			name: "인라인에는 style도 없다",
			src:  `<slide><p><em style="order: 1">x</em></p></slide>`,
			want: []string{"1:15 <em>에는 속성을 쓸 수 없다 (style)"},
		},
		{
			// grid-template-*는 flex 컨테이너의 CSS에서 무효다 — 받아 놓고 버리지 않는다.
			name: "row에는 cols가 없다",
			src:  `<slide><row cols="3"></row></slide>`,
			want: []string{"1:13 <row>에 알 수 없는 속성 cols (쓸 수 있는 속성: gap, align, justify, wrap, pad, w, h, min-w, min-h, max-w, max-h, ratio, grow, shrink, basis, span, self, order, at, x, y, style)"},
		},
		{
			name: "col에는 rows가 없다",
			src:  `<slide><col rows="2fr 1fr"></col></slide>`,
			want: []string{"1:13 <col>에 알 수 없는 속성 rows"},
		},
		{
			name: "spacer는 컨테이너가 아니다",
			src:  `<slide><spacer gap="lg"/></slide>`,
			want: []string{"1:16 <spacer>에 알 수 없는 속성 gap (쓸 수 있는 속성: grow, shrink, basis, span, self, order, at, x, y, style)"},
		},
		{
			// <stack>은 Iteration 6에 열렸다 — 이제 단일 셀 겹침 컨테이너다. cols/rows는
			// 절대 열지 않는다: 단일 셀이라는 것이 stack의 정의다 (stackFlowAttrs·PRD §13-3).
			name: "stack에는 cols가 없다 (단일 셀)",
			src:  `<slide><stack cols="2"><text>a</text></stack></slide>`,
			want: []string{"1:15 <stack>에 알 수 없는 속성 cols (쓸 수 있는 속성: align, justify, pad, w, h, min-w, min-h, max-w, max-h, ratio, grow, shrink, basis, span, self, order, at, x, y, style)"},
		},
		{
			// gap은 셀 사이의 간격인데 자식이 전부 같은 칸에 있어 뜻이 없다 — 받아 놓고
			// 아무것도 하지 않는 속성은 열지 않는다 (stackFlowAttrs).
			name: "stack에는 gap이 없다",
			src:  `<slide><stack gap="md"><text>a</text></stack></slide>`,
			want: []string{"1:15 <stack>에 알 수 없는 속성 gap"},
		},
		{
			// wrap은 flex 전용(`flex-wrap`)이라 grid 흐름인 stack에서 무효다.
			name: "stack에는 wrap이 없다",
			src:  `<slide><stack wrap="true"><text>a</text></stack></slide>`,
			want: []string{"1:15 <stack>에 알 수 없는 속성 wrap"},
		},
		{
			// 노트는 콘텐츠가 아니라 슬라이드 메타데이터다 (layoutChildren의 근거와 같다) —
			// stack도 레이아웃 컨테이너이므로 자식 어휘가 같다.
			name: "stack 안의 note",
			src:  `<slide><stack><note>메모</note></stack></slide>`,
			want: []string{"1:16 <stack>의 자식으로 <note>는 올 수 없다"},
		},
		{
			name: "닫히지 않은 stack",
			src:  "<slide>\n  <stack>\n    <text>배지</text>\n</slide>",
			want: []string{"2:4 <stack> 태그가 </slide>보다 먼저 닫혀야 한다"},
		},
		{
			// <li>는 리스트 컨테이너의 자식이지 flex 항목이 아니다 (childPlacementAttrs) —
			// at도 웹에서만 뜻을 갖는 선언이 되므로 열리지 않는다.
			name: "li에는 자식 배치 속성 at이 없다",
			src:  `<slide><ul><li at="center">항목</li></ul></slide>`,
			want: []string{"1:16 <li>에는 속성을 쓸 수 없다 (at)"},
		},
		{
			name: "컨테이너 자신에게 role은 없다",
			src:  `<slide><row role="body"></row></slide>`,
			want: []string{"1:13 <row>에 알 수 없는 속성 role"},
		},
		{
			name: "컨테이너는 외관 속성을 갖지 않는다",
			src:  `<slide><col color="accent1"></col></slide>`,
			want: []string{"1:13 <col>에 알 수 없는 속성 color"},
		},
		{
			name: "모르는 컨테이너 속성",
			src:  `<slide><row gpa="md"></row></slide>`,
			want: []string{"1:13 <row>에 알 수 없는 속성 gpa"},
		},
		{
			// 넓어진 것은 컨테이너의 자식 어휘뿐이다 — 블록 안의 블록은 여전히 에러다.
			name: "컨테이너 안에서도 블록 안의 블록은 에러다",
			src:  `<slide><row><p><p>본문</p></p></row></slide>`,
			want: []string{"1:17 <p> 안에 블록 태그 <p>는 올 수 없다 (블록은 겹치지 않는다)"},
		},
		{
			name: "블록 안의 컨테이너",
			src:  `<slide><p><row><text>a</text></row></p></slide>`,
			want: []string{"1:12 <p> 안에는 문자 데이터와 인라인 태그만 올 수 있다 (<row>는 쓸 수 없다)"},
		},
		{
			name: "text 안의 spacer",
			src:  `<slide><text><spacer/></text></slide>`,
			want: []string{"1:15 <text> 안에는 문자 데이터와 인라인 태그만 올 수 있다 (<spacer>는 쓸 수 없다)"},
		},
		{
			name: "li 안의 컨테이너",
			src:  `<slide><ul><li><col><p>a</p></col></li></ul></slide>`,
			want: []string{"1:17 <li> 안에는 문자 데이터와 인라인 태그만 올 수 있다 (<col>는 쓸 수 없다)"},
		},
		{
			name: "리스트 컨테이너의 자식은 여전히 li뿐이다",
			src:  `<slide><ul><row><p>x</p></row></ul></slide>`,
			want: []string{"1:13 <ul>의 자식으로 <row>는 올 수 없다 (쓸 수 있는 태그: <li>)"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, diags := Parse("slides/001_title.slide", tc.src)
			for i, want := range tc.want {
				pos, message, _ := strings.Cut(want, " ")
				if i >= len(diags) {
					t.Errorf("진단 %d개뿐이다, want %q", len(diags), want)
					break
				}
				if got := diags[i].Span.Start.String(); got != pos {
					t.Errorf("진단[%d] 위치 = %s, want %s (%s)", i, got, pos, diags[i].Message)
				}
				if !strings.Contains(diags[i].Message, message) {
					t.Errorf("진단[%d] 메시지 = %q, want %q 포함", i, diags[i].Message, message)
				}
			}
			if len(diags) != len(tc.want) {
				t.Errorf("진단 %d개, want %d개:\n%s", len(diags), len(tc.want), diags.Err())
			}
		})
	}
}

// 종류 카탈로그는 네 층(파서·geom·web·pptx)이 공유하는 한 표다 — 표가 흔들리면 좌표가
// 엉뚱한 요소에 붙는다. 레이아웃 컨테이너와 <spacer>가 **다른 종류**인 것이 계약이다.
func TestLayoutKinds(t *testing.T) {
	for _, name := range []string{RowElement, ColElement, GridElement, StackElement} {
		if got := KindOf(name); got != KindLayoutContainer {
			t.Errorf("KindOf(%q) = %v, want KindLayoutContainer", name, got)
		}
	}
	if got := KindOf(SpacerElement); got != KindSpacer {
		t.Errorf("KindOf(%q) = %v, want KindSpacer", SpacerElement, got)
	}
	// 카탈로그의 모든 레이아웃 이름이 문법 표에도 있어야 한다 — 종류만 있고 문법이 없는
	// 이름은 "알 수 없는 태그"로 진단되면서 렌더러는 종류로 분기하는 어긋난 상태가 된다.
	for _, name := range append(append([]string{}, layoutContainerElements...), SpacerElement) {
		if _, known := elementSpecs[name]; !known {
			t.Errorf("레이아웃 태그 %q가 문법 표(elementSpecs)에 없다", name)
		}
	}
	// <spacer>는 <br>과 함께 void다. 컨테이너는 void가 아니다.
	if !IsVoidElement(SpacerElement) || !IsVoidElement(BreakElement) {
		t.Error("IsVoidElement가 <spacer>·<br>을 void로 보지 않는다")
	}
	for _, name := range layoutContainerElements {
		if IsVoidElement(name) {
			t.Errorf("IsVoidElement(%q) = true — 컨테이너는 자식을 갖는다", name)
		}
	}
	// 레이아웃 태그는 텍스트 역할을 갖지 않는다 — 배치 채널만 갖고 자기 텍스트가 없다.
	for _, name := range append(append([]string{}, layoutContainerElements...), SpacerElement) {
		if role, ok := TagTextRole(name); ok {
			t.Errorf("TagTextRole(%q) = %q — 레이아웃 태그는 역할 표에 없어야 한다", name, role)
		}
	}
	// <stack>은 Iteration 6에 열렸다 — 단일 셀 겹침 컨테이너이자 레이아웃 컨테이너 종류다
	// (새 종류 KindStack을 만들지 않는다). 문법 표에도 있고 흐름은 FlowStack이다.
	if got := KindOf(StackElement); got != KindLayoutContainer {
		t.Errorf("KindOf(%q) = %v, want KindLayoutContainer (새 종류가 아니다)", StackElement, got)
	}
	if _, known := elementSpecs[StackElement]; !known {
		t.Errorf("<%s>가 문법 표(elementSpecs)에 없다", StackElement)
	}
	if got := FlowOf(StackElement); got != FlowStack {
		t.Errorf("FlowOf(%q) = %d, want FlowStack", StackElement, got)
	}
	// stack의 컨테이너 속성은 gap·wrap·cols·rows가 **없고** align·justify·pad·크기가 **있다**.
	stackAttrs := strings.Join(ContainerAttrs(StackElement), " ")
	for _, absent := range []string{"gap", "wrap", "cols", "rows"} {
		if strings.Contains(" "+stackAttrs+" ", " "+absent+" ") {
			t.Errorf("ContainerAttrs(stack)에 %q가 있다 — stack에서 뜻이 없다: %s", absent, stackAttrs)
		}
	}
	for _, present := range []string{"align", "justify", "pad", "w", "h", "ratio"} {
		if !strings.Contains(" "+stackAttrs+" ", " "+present+" ") {
			t.Errorf("ContainerAttrs(stack)에 %q가 없다: %s", present, stackAttrs)
		}
	}
	// row·grid의 컨테이너 속성은 한 항목도 움직이지 않았다.
	if got, want := strings.Join(ContainerAttrs(RowElement), " "),
		"gap align justify wrap pad w h min-w min-h max-w max-h ratio"; got != want {
		t.Errorf("ContainerAttrs(row) = %q, want %q", got, want)
	}
	if got, want := strings.Join(ContainerAttrs(GridElement), " "),
		"gap align justify wrap pad cols rows w h min-w min-h max-w max-h ratio"; got != want {
		t.Errorf("ContainerAttrs(grid) = %q, want %q", got, want)
	}
	// 자식 배치 속성의 순서 — at·x·y가 order 뒤, style(탈출구) 앞이다.
	if got, want := strings.Join(ChildPlacementAttrs(), " "),
		"grow shrink basis span self order at x y style"; got != want {
		t.Errorf("ChildPlacementAttrs() = %q, want %q", got, want)
	}
	// `<arrow>`·`<line>`·`<group>`은 여전히 닫혀 있다 (Iteration 7·9).
	for _, name := range []string{"arrow", "line", "group"} {
		if KindOf(name) != KindNone {
			t.Errorf("<%s>가 카탈로그에 있다 — 뒤 iteration의 어휘다", name)
		}
		if _, known := elementSpecs[name]; known {
			t.Errorf("<%s>가 문법 표에 있다 — 뒤 iteration의 어휘다", name)
		}
	}
}

// 자식 배치 속성은 한 벌이고, 배치의 대상이 되는 태그 전부가 그 한 벌을 그대로 받는다 —
// 태그마다 표를 복제하면 PRD 표에 속성이 하나 늘 때 어느 태그가 빠졌는지 사람이 세어야 한다.
func TestChildPlacementAttrsCoverage(t *testing.T) {
	targets := []string{TextElement}
	targets = append(targets, blockElements...)
	targets = append(targets, listContainerElements...)
	targets = append(targets, layoutContainerElements...)
	targets = append(targets, SpacerElement)
	for _, name := range targets {
		spec := elementSpecs[name]
		for _, attr := range childPlacementAttrs {
			found := false
			for _, allowed := range spec.attrs {
				if allowed == attr {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("<%s>에 자식 배치 속성 %s가 없다", name, attr)
			}
		}
	}
	// 배치의 대상이 아닌 태그에는 하나도 없어야 한다.
	for _, name := range append(append([]string{}, inlineElements...), ListItemElement, RootElement, noteElement) {
		for _, attr := range elementSpecs[name].attrs {
			for _, placement := range childPlacementAttrs {
				if attr == placement {
					t.Errorf("<%s>에 자식 배치 속성 %s가 열려 있다", name, attr)
				}
			}
		}
	}
}

// 픽스처 `03_layout`의 슬라이드가 파서 어휘와 어긋나지 않는지 파싱만 검증한다 — 골든·렌더
// 검증은 픽스처 러너(internal/testkit)의 몫이고, 여기는 파서 계약(진단 0건)만 본다.
// style에 외관을 쓴 슬라이드도 파서에는 진단이 없다: 값 allowlist와 경고는 값 층(항목 02)이
// 소유한다.
func TestParseLayoutFixture(t *testing.T) {
	dir := filepath.Join("..", "testdata", "fixtures", "03_layout", "slides")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("픽스처 디렉토리를 읽을 수 없다: %v", err)
	}
	parsed := 0
	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), Ext) {
			continue
		}
		src, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			t.Fatalf("%s: %v", entry.Name(), err)
		}
		if _, diags := Parse(entry.Name(), string(src)); len(diags) != 0 {
			t.Errorf("%s: 진단이 없어야 한다:\n%s", entry.Name(), diags.Err())
		}
		parsed++
	}
	if parsed == 0 {
		t.Fatal("파싱한 슬라이드가 없다 — 픽스처가 비었다")
	}
}

// 픽스처 `06_stack`은 이 iteration의 시험대다 (항목 06이 활성화했다). 여기서 검증하는 계약은
// 하나다: 열린 어휘(`<stack>`·자식 `at`·`x`·`y`)를 쓴 모든 슬라이드가 진단 0건으로 파싱된다.
// 값 해석·골든은 항목 03~06의 몫이고, 이 검사는 브라우저 없이 파싱만으로 그 절반을 지킨다.
func TestParseStackFixture(t *testing.T) {
	dir := filepath.Join("..", "testdata", "fixtures", "06_stack", "slides")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("픽스처 디렉토리를 읽을 수 없다: %v", err)
	}
	parsed := 0
	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), Ext) {
			continue
		}
		src, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			t.Fatalf("%s: %v", entry.Name(), err)
		}
		if _, diags := Parse(entry.Name(), string(src)); len(diags) != 0 {
			t.Errorf("%s: 진단이 없어야 한다:\n%s", entry.Name(), diags.Err())
		}
		parsed++
	}
	if parsed == 0 {
		t.Fatal("파싱한 슬라이드가 없다 — 06_stack 픽스처가 비었다")
	}
}
