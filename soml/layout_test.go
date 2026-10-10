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
			want: []string{"1:9 void tag <spacer> cannot have content (write <spacer/>)"},
		},
		{
			name: "컨테이너 안의 문자 데이터",
			src:  `<slide><row>글자</row></slide>`,
			want: []string{"1:13 character data cannot appear directly inside <row> (wrap it in <text>)"},
		},
		{
			// 노트는 콘텐츠가 아니라 슬라이드 메타데이터다 (SlideDoc.Notes).
			name: "컨테이너 안의 note",
			src:  `<slide><row><note>메모</note></row></slide>`,
			want: []string{"1:14 <row> cannot contain <note> (available tags: <text>, <shape>, <image>, <table>, <chart>, <h1>, <h2>, <h3>, <p>, <blockquote>, <pre>, <ul>, <ol>, <row>, <col>, <grid>, <stack>, <spacer>)"},
		},
		{
			// <li>는 리스트 컨테이너의 자식이지 flex 항목이 아니다.
			name: "li에는 자식 배치 속성이 없다",
			src:  `<slide><ul><li grow="1">x</li></ul></slide>`,
			want: []string{"1:16 unknown attribute grow on <li> (available attributes: reveal)"},
		},
		{
			// 인라인은 박스가 아니라 런이다.
			name: "인라인에는 style도 없다",
			src:  `<slide><p><em style="order: 1">x</em></p></slide>`,
			want: []string{"1:15 <em> takes no attributes (style)"},
		},
		{
			// grid-template-*는 flex 컨테이너에서 무효다 — 받아 놓고 버리지 않는다.
			name: "row에는 cols가 없다",
			src:  `<slide><row cols="3"></row></slide>`,
			want: []string{"1:13 unknown attribute cols on <row> (available attributes: gap, align, justify, wrap, pad, w, h, min-w, min-h, max-w, max-h, ratio, grow, shrink, basis, span, self, order, at, x, y, style, slot, enter)"},
		},
		{
			// 단일 셀이라는 것이 stack의 정의다 — gap·wrap·cols·rows는 뜻이 없다.
			name: "stack에는 cols가 없다",
			src:  `<slide><stack cols="2"><text>a</text></stack></slide>`,
			want: []string{"1:15 unknown attribute cols on <stack> (available attributes: align, justify, pad, w, h, min-w, min-h, max-w, max-h, ratio, grow, shrink, basis, span, self, order, at, x, y, style, slot, enter)"},
		},
		{
			name: "spacer는 컨테이너가 아니다",
			src:  `<slide><spacer gap="lg"/></slide>`,
			want: []string{"1:16 unknown attribute gap on <spacer> (available attributes: grow, shrink, basis, span, self, order, at, x, y, style, slot, enter)"},
		},
		{
			name: "블록 안의 컨테이너",
			src:  `<slide><p><row><text>a</text></row></p></slide>`,
			want: []string{"1:12 only character data and inline tags are allowed inside <p> (<row> is not allowed)"},
		},
	})
}
