package soml

import (
	"strconv"
	"strings"
	"testing"
)

// dumpNodes는 콘텐츠 트리를 한 줄로 편다 — 표 기반 테스트에서 트리 전체(이름·속성·자식·
// 텍스트)를 문자열 하나로 단정하기 위한 표현이다.
//
//	<p>배포 <strong>자동화</strong></p>  →  p("배포 " strong("자동화"))
func dumpNodes(nodes []Node) string {
	parts := make([]string, len(nodes))
	for i, node := range nodes {
		parts[i] = dumpNode(node)
	}
	return strings.Join(parts, " ")
}

func dumpNode(node Node) string {
	switch n := node.(type) {
	case *Text:
		return strconv.Quote(n.Value)
	case *Element:
		var b strings.Builder
		b.WriteString(n.Name)
		if len(n.Attrs) > 0 {
			b.WriteString("[")
			for i, attr := range n.Attrs {
				if i > 0 {
					b.WriteString(" ")
				}
				b.WriteString(attr.Name + "=" + attr.Value)
			}
			b.WriteString("]")
		}
		if len(n.Children) > 0 {
			b.WriteString("(" + dumpNodes(n.Children) + ")")
		}
		return b.String()
	}
	return "?"
}

// 텍스트 서브셋의 정상 트리 — 태그별 이름·자식·속성이 파싱을 그대로 통과한다.
func TestParseTextSubset(t *testing.T) {
	runTreeCases(t, []treeCase{
		{
			name: "제목 위계",
			src:  `<slide><h1>제목</h1><h2>부제</h2><h3>소제</h3></slide>`,
			want: `h1("제목") h2("부제") h3("소제")`,
		},
		{
			name: "문단과 인용과 pre",
			src:  `<slide><p>본문</p><blockquote>인용</blockquote><pre>x := 1</pre></slide>`,
			want: `p("본문") blockquote("인용") pre("x := 1")`,
		},
		{
			name: "리스트 둘과 인라인 섞인 항목",
			src:  `<slide><ul><li>배포 <strong>자동화</strong></li></ul><ol><li>첫째</li><li>둘째</li></ol></slide>`,
			want: `ul(li("배포 " strong("자동화"))) ol(li("첫째") li("둘째"))`,
		},
		{
			name: "인라인 중첩·코드·링크·줄바꿈·span",
			src: `<slide><p><strong><em>강조</em></strong> <code>go build</code> <a href="https://example.com">문서</a>` +
				`첫 줄<br/>둘째 줄 <span color="accent1" weight="700">강조</span></p></slide>`,
			want: `p(strong(em("강조")) " " code("go build") " " a[href=https://example.com]("문서") "첫 줄" br "둘째 줄 " span[color=accent1 weight=700]("강조"))`,
		},
		{
			name: "블록 ≡ text 역할 등가 — 나란히 파싱된다",
			src:  `<slide><text role="body">본문 <em>강조</em></text><p>본문</p></slide>`,
			want: `text[role=body]("본문 " em("강조")) p("본문")`,
		},
		{
			name: "variant — text와 블록에 열려 있다",
			src:  `<slide><text variant="kicker">개요</text><p variant="lead">본문</p></slide>`,
			want: `text[variant=kicker]("개요") p[variant=lead]("본문")`,
		},
		{
			// anchor는 도형 밖에서도 문법은 유효하다 — 자리에 따라 속성 어휘가 갈라지면 저자가
			// 태그를 옮길 때마다 문법이 바뀐다.
			name: "블록의 외관 속성과 가로 크기",
			src:  `<slide><h3 size="40" color="accent1" align="center" anchor="center" w="fill" max-w="720">소제</h3></slide>`,
			want: `h3[size=40 color=accent1 align=center anchor=center w=fill max-w=720]("소제")`,
		},
		{
			name: "text의 외관·크기·배치가 함께",
			src:  `<slide><text size="16" w="140" min-w="80" max-w="50%" align="center" grow="1">x</text></slide>`,
			want: `text[size=16 w=140 min-w=80 max-w=50% align=center grow=1]("x")`,
		},
		{
			// 리스트 컨테이너는 블록과 같은 외관 4종과 가로 크기를 받는다 (variant는 닫혀 있다).
			name: "리스트 컨테이너의 외관·크기·배치",
			src:  `<slide><row><ul size="20" color="accent1" align="center" anchor="center" w="480" basis="360"><li>항목</li></ul></row></slide>`,
			want: `row(ul[size=20 color=accent1 align=center anchor=center w=480 basis=360](li("항목")))`,
		},
		{
			// reveal은 이름만 파서가 본다 — 값 어휘(정수 ≥ 1)는 값 층의 것이다.
			name: "li의 reveal — 값 없음과 번호",
			src:  `<slide><ul><li>하나</li><li reveal>둘</li><li reveal="2">셋</li></ul></slide>`,
			want: `ul(li("하나") li[reveal=]("둘") li[reveal=2]("셋"))`,
		},
		{
			// 슬롯이 그 레이아웃에 있는가·직계 자식인가는 core가 검증한다.
			name: "slot은 슬라이드 직계 자식에 열린다",
			src: `<slide layout="two-column"><text slot="title">제목</text><col slot="left"><text>왼쪽</text></col>` +
				`<shape slot="right" type="rect"><text>오른쪽</text></shape><text>슬롯 없음</text></slide>`,
			want: `text[slot=title]("제목") col[slot=left](text("왼쪽")) shape[slot=right type=rect](text("오른쪽")) text("슬롯 없음")`,
		},
	})
}

// 텍스트 서브셋의 배치 위반·void 위반·모르는 속성 — 진단의 line:col과 문구까지 단정한다.
func TestParseTextSubsetErrors(t *testing.T) {
	runDiagCases(t, []diagCase{
		{
			name: "li가 슬라이드 직속",
			src:  `<slide><li>항목</li></slide>`,
			want: []string{"1:9 <li> can only be a child of <ul>/<ol>"},
		},
		{
			name: "li가 블록 안",
			src:  `<slide><p><li>항목</li></p></slide>`,
			want: []string{"1:12 <li> can only be a child of <ul>/<ol>"},
		},
		{
			name: "중첩 리스트는 아직이다",
			src:  `<slide><ul><li><ul><li>깊다</li></ul></li></ul></slide>`,
			want: []string{"1:17 <ul> cannot be nested inside <li>"},
		},
		{
			name: "블록 안의 블록",
			src:  `<slide><ul><li><h2>제목</h2></li></ul></slide>`,
			want: []string{"1:17 block tag <h2> is not allowed inside <li> (blocks do not nest)"},
		},
		{
			name: "a 안의 a — 사이에 인라인이 있어도",
			src:  `<slide><p><a href="x"><strong><a href="y">z</a></strong></a></p></slide>`,
			want: []string{"1:32 <a> cannot be nested inside <a>"},
		},
		{
			name: "void 위반 — br에 내용",
			src:  `<slide><p><br>텍스트</br></p></slide>`,
			want: []string{"1:12 void tag <br> cannot have content (write <br/>)"},
		},
		{
			name: "span의 속성 목록",
			src:  `<slide><p><span align="center">x</span></p></slide>`,
			want: []string{"1:17 unknown attribute align on <span> (available attributes: color, weight, size)"},
		},
		{
			// 블록은 역할을 표에서만 얻는다 — role 속성을 열면 <p role="title">이 두 표기의
			// 등가를 깬다.
			name: "블록에는 role이 없다",
			src:  `<slide><h1 role="body">제목</h1></slide>`,
			want: []string{"1:12 unknown attribute role on <h1> (available attributes: variant, size, color, align, anchor, w, min-w, max-w, grow, shrink, basis, span, self, order, at, x, y, style, slot, enter)"},
		},
		{
			name: "인라인에는 속성이 없다",
			src:  `<slide><p><strong variant="kicker">x</strong></p></slide>`,
			want: []string{"1:19 <strong> takes no attributes (variant)"},
		},
		{
			// 항목은 외관을 상속만 받는다 — 열린 것은 순서(reveal)뿐이다.
			name: "li에는 외관 속성이 없다",
			src:  `<slide><ul><li size="20">x</li></ul></slide>`,
			want: []string{"1:16 unknown attribute size on <li> (available attributes: reveal)"},
		},
		{
			// 세로 축·pad·ratio는 텍스트에 닫혀 있다.
			name: "text에는 h가 없다",
			src:  `<slide><text h="56">x</text></slide>`,
			want: []string{`1:14 unknown attribute h on <text> (available attributes: role, variant, size, color, align, anchor, w, min-w, max-w, grow, shrink, basis, span, self, order, at, x, y, style, slot, enter)`},
		},
		{
			name: "pre 안의 인라인은 아직이다",
			src:  `<slide><pre><code>x</code></pre></slide>`,
			want: []string{"1:14 only character data is allowed inside <pre> (<code> is not allowed)"},
		},
		{
			name: "블록 안의 text",
			src:  `<slide><p><text>x</text></p></slide>`,
			want: []string{"1:12 only character data and inline tags are allowed inside <p> (<text> is not allowed)"},
		},
		{
			name: "인라인이 슬라이드 직속 — 슬라이드의 자식 어휘",
			src:  `<slide><strong>x</strong></slide>`,
			want: []string{"1:9 <slide> cannot contain <strong> (available tags: <text>, <shape>, <image>, <table>, <chart>, <connector>, <note>, <h1>, <h2>, <h3>, <p>, <blockquote>, <pre>, <ul>, <ol>, <row>, <col>, <grid>, <stack>, <spacer>)"},
		},
		{
			name: "ul 안의 문자 데이터",
			src:  `<slide><ul>글자</ul></slide>`,
			want: []string{"1:12 character data cannot appear directly inside <ul> (wrap it in <li>)"},
		},
		{
			name: "ul의 자식은 li뿐",
			src:  `<slide><ul><p>x</p></ul></slide>`,
			want: []string{"1:13 <ul> cannot contain <p> (available tags: <li>)"},
		},
	})
}

// 인라인 경계 공백 계약 — 여기서 접히는 방식이 곧 런 분할의 입력이다.
//
// 계약: 요소(블록이든 인라인이든) **양 끝**의 공백은 들여쓰기이므로 내용이 아니고,
// 형제 **사이**의 공백은 내용이다. 태그 안쪽 끝에 붙인 공백은 잘린다.
func TestParseInlineEdgeSpace(t *testing.T) {
	runTreeCases(t, []treeCase{
		{
			name: "인라인 앞의 한 칸은 내용",
			src:  `<slide><p>배포 파이프라인 <strong>자동화</strong></p></slide>`,
			want: `p("배포 파이프라인 " strong("자동화"))`,
		},
		{
			name: "인라인 뒤의 한 칸도 내용",
			src:  `<slide><p><strong>배포</strong> 자동화</p></slide>`,
			want: `p(strong("배포") " 자동화")`,
		},
		{
			name: "블록 양 끝의 들여쓰기는 내용이 아니다",
			src:  "<slide><p>\n  배포 <strong>자동화</strong>\n</p></slide>",
			want: `p("배포 " strong("자동화"))`,
		},
		{
			name: "인라인 양 끝도 트리밍된다",
			src:  `<slide><p>배포 <strong> 자동화 </strong></p></slide>`,
			want: `p("배포 " strong("자동화"))`,
		},
		{
			name: "인라인 사이의 공백은 내용",
			src:  `<slide><p><strong>a</strong> <em>b</em></p></slide>`,
			want: `p(strong("a") " " em("b"))`,
		},
		{
			name: "여러 줄 인라인",
			src:  "<slide><p>배포\n  <strong>\n    자동화\n  </strong>\n</p></slide>",
			want: `p("배포 " strong("자동화"))`,
		},
	})
}

// pre는 공백을 보존한다 — 여는 태그 직후·닫는 태그 직전의 개행 하나만 벗긴다 (HTML 관례).
func TestParsePrePreservesWhitespace(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want string
	}{
		{
			name: "개행과 탭이 그대로 남는다",
			src:  "<slide><pre>\nfunc main() {\n\tfmt.Println()\n}\n</pre></slide>",
			want: "func main() {\n\tfmt.Println()\n}",
		},
		{name: "연속 공백은 접히지 않는다", src: `<slide><pre>a  b</pre></slide>`, want: "a  b"},
		{name: "양 끝 개행은 하나씩만 벗긴다", src: "<slide><pre>\n\ncode\n\n</pre></slide>", want: "\ncode\n"},
		{name: "CRLF도 개행 하나다", src: "<slide><pre>\r\ncode\r\n</pre></slide>", want: "code"},
		{name: "안쪽 들여쓰기는 내용이다", src: "<slide><pre>\n  indented\n</pre></slide>", want: "  indented"},
		{name: "닫는 태그 앞의 들여쓰기는 남는다", src: "<slide><pre>\ncode\n  </pre></slide>", want: "code\n  "},
		// 벗기고 남은 빈 텍스트 노드를 트리에 남기지 않는다.
		{name: "개행뿐인 pre는 빈 요소다", src: "<slide><pre>\n</pre></slide>", want: ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			pre := elementAt(t, parseOK(t, tc.src).Content, 0)
			if got := pre.TextContent(); got != tc.want {
				t.Errorf("pre 내용 = %q, want %q", got, tc.want)
			}
			if tc.want == "" && len(pre.Children) != 0 {
				t.Errorf("자식 %d개 (%s), want 0개", len(pre.Children), dumpNodes(pre.Children))
			}
		})
	}
}

// 태그 → 역할 표는 테마가 아니라 **언어**의 계약이다 — 표가 흔들리면
// `<p>` ≡ `<text role="body">` 등가가 문서마다 달라진다. 표 전체를 여기서 못 박는다.
func TestTagTextRole(t *testing.T) {
	cases := []struct {
		tag  string
		role TextRole
	}{
		{"h1", RoleTitle},
		{"h2", RoleHeading},
		{"h3", RoleHeading},
		{"p", RoleBody},
		{"blockquote", RoleBody},
		{"pre", RoleCode},
		{ListItemElement, RoleBody},
	}
	for _, tc := range cases {
		role, ok := TagTextRole(tc.tag)
		if !ok || role != tc.role {
			t.Errorf("TagTextRole(%q) = %q, %v — want %q", tc.tag, role, ok, tc.role)
		}
	}
	// 표 밖의 이름 전부는 역할이 없다 — <text>는 role 속성이 정하고, 나머지는 텍스트가 아니다.
	inTable := map[string]bool{}
	for _, tc := range cases {
		inTable[tc.tag] = true
	}
	for _, name := range append(allElementNames(), TextElement, RootElement, "h6") {
		if inTable[name] {
			continue
		}
		if role, ok := TagTextRole(name); ok {
			t.Errorf("TagTextRole(%q) = %q — 표에 없어야 한다", name, role)
		}
	}
}
