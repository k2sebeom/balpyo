package soml

import "testing"

// 레이아웃 태그의 정상 트리 — 컨테이너 속성·자식 배치 속성·중첩·겹침이 파싱을 그대로
// 통과한다. at·x·y의 값 어휘는 값 층이 본다.
func TestParseLayout(t *testing.T) {
	runTreeCases(t, []treeCase{
		{
			name: "컨테이너 속성과 크기",
			src:  `<slide><row gap="xl" align="center" justify="between" wrap="true" pad="md" w="fill" h="320" min-w="200" max-h="480" ratio="16:9"><text>a</text></row></slide>`,
			want: `row[gap=xl align=center justify=between wrap=true pad=md w=fill h=320 min-w=200 max-h=480 ratio=16:9](text("a"))`,
		},
		{
			name: "grid의 트랙 목록과 깊은 중첩",
			src:  `<slide><grid cols="4" rows="2fr 1fr"><row><col grow="2" span="2"><p>a</p></col></row></grid></slide>`,
			want: `grid[cols=4 rows=2fr 1fr](row(col[grow=2 span=2](p("a"))))`,
		},
		{
			name: "컨테이너의 자식 어휘 — text·블록·리스트·중첩·spacer와 배치 속성",
			src:  `<slide><col><text grow="1" self="center" order="2">a</text><p shrink="0" basis="240" style="margin: 0">b</p><ol basis="360"><li>c</li></ol><row><p>e</p></row><spacer grow="2"/></col></slide>`,
			want: `col(text[grow=1 self=center order=2]("a") p[shrink=0 basis=240 style=margin: 0]("b") ol[basis=360](li("c")) row(p("e")) spacer[grow=2])`,
		},
		{
			name: "슬라이드 직속 spacer와 빈 컨테이너",
			src:  `<slide><h2>제목</h2><spacer/><row></row><col/></slide>`,
			want: `h2("제목") spacer row col`,
		},
		{
			// stack은 단일 셀 겹침 컨테이너다 — at은 셀 안 앵커, x/y는 절대 오프셋이다.
			name: "stack의 정렬과 자식의 at·x·y",
			src:  `<slide><stack align="center" justify="center"><shape type="rect" at="top-right"/><image src="assets/dot.png" x="20" y="-10"/><stack><text at="center">안쪽</text></stack></stack></slide>`,
			want: `stack[align=center justify=center](shape[type=rect at=top-right] image[src=assets/dot.png x=20 y=-10] stack(text[at=center]("안쪽")))`,
		},
	})
}

// 레이아웃 태그의 배치 위반과 닫힌 속성 — 진단의 line:col과 문구까지 단정한다.
func TestParseLayoutErrors(t *testing.T) {
	runDiagCases(t, []diagCase{
		{
			name: "spacer는 void다",
			src:  `<slide><spacer><text>a</text></spacer></slide>`,
			want: []string{"1:9 void 태그 <spacer>은 내용을 가질 수 없다 (<spacer/>로 쓴다)"},
		},
		{
			name: "컨테이너 안의 문자 데이터",
			src:  `<slide><row>글자</row></slide>`,
			want: []string{"1:13 <row> 안에 문자 데이터를 직접 쓸 수 없다 (<text>로 감싼다)"},
		},
		{
			// 노트는 콘텐츠가 아니라 슬라이드 메타데이터다 (SlideDoc.Notes).
			name: "컨테이너 안의 note",
			src:  `<slide><row><note>메모</note></row></slide>`,
			want: []string{"1:14 <row>의 자식으로 <note>는 올 수 없다 (쓸 수 있는 태그: <text>, <shape>, <image>, <table>, <chart>, <h1>, <h2>, <h3>, <p>, <blockquote>, <pre>, <ul>, <ol>, <row>, <col>, <grid>, <stack>, <spacer>)"},
		},
		{
			// <li>는 리스트 컨테이너의 자식이지 flex 항목이 아니다.
			name: "li에는 자식 배치 속성이 없다",
			src:  `<slide><ul><li grow="1">x</li></ul></slide>`,
			want: []string{"1:16 <li>에 알 수 없는 속성 grow (쓸 수 있는 속성: reveal)"},
		},
		{
			// 인라인은 박스가 아니라 런이다.
			name: "인라인에는 style도 없다",
			src:  `<slide><p><em style="order: 1">x</em></p></slide>`,
			want: []string{"1:15 <em>에는 속성을 쓸 수 없다 (style)"},
		},
		{
			// grid-template-*는 flex 컨테이너에서 무효다 — 받아 놓고 버리지 않는다.
			name: "row에는 cols가 없다",
			src:  `<slide><row cols="3"></row></slide>`,
			want: []string{"1:13 <row>에 알 수 없는 속성 cols (쓸 수 있는 속성: gap, align, justify, wrap, pad, w, h, min-w, min-h, max-w, max-h, ratio, grow, shrink, basis, span, self, order, at, x, y, style, slot)"},
		},
		{
			// 단일 셀이라는 것이 stack의 정의다 — gap·wrap·cols·rows는 뜻이 없다.
			name: "stack에는 cols가 없다",
			src:  `<slide><stack cols="2"><text>a</text></stack></slide>`,
			want: []string{"1:15 <stack>에 알 수 없는 속성 cols (쓸 수 있는 속성: align, justify, pad, w, h, min-w, min-h, max-w, max-h, ratio, grow, shrink, basis, span, self, order, at, x, y, style, slot)"},
		},
		{
			name: "spacer는 컨테이너가 아니다",
			src:  `<slide><spacer gap="lg"/></slide>`,
			want: []string{"1:16 <spacer>에 알 수 없는 속성 gap (쓸 수 있는 속성: grow, shrink, basis, span, self, order, at, x, y, style, slot)"},
		},
		{
			name: "블록 안의 컨테이너",
			src:  `<slide><p><row><text>a</text></row></p></slide>`,
			want: []string{"1:12 <p> 안에는 문자 데이터와 인라인 태그만 올 수 있다 (<row>는 쓸 수 없다)"},
		},
	})
}
