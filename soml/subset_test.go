package soml

import (
	"os"
	"path/filepath"
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
	tests := []struct {
		name string
		src  string
		want string
	}{
		{
			name: "제목 위계",
			src:  `<slide><h1>제목</h1><h2>부제</h2><h3>소제</h3></slide>`,
			want: `h1("제목") h2("부제") h3("소제")`,
		},
		{
			name: "문단과 인용",
			src:  `<slide><p>본문</p><blockquote>인용</blockquote></slide>`,
			want: `p("본문") blockquote("인용")`,
		},
		{
			name: "순서 없는 리스트",
			src:  `<slide><ul><li>하나</li><li>둘</li></ul></slide>`,
			want: `ul(li("하나") li("둘"))`,
		},
		{
			name: "순서 있는 리스트",
			src:  `<slide><ol><li>첫째</li><li>둘째</li></ol></slide>`,
			want: `ol(li("첫째") li("둘째"))`,
		},
		{
			name: "인라인 섞인 리스트 항목",
			src:  `<slide><ul><li>배포 <strong>자동화</strong></li></ul></slide>`,
			want: `ul(li("배포 " strong("자동화")))`,
		},
		{
			name: "인라인 중첩",
			src:  `<slide><p><strong><em>강조</em></strong></p></slide>`,
			want: `p(strong(em("강조")))`,
		},
		{
			name: "인라인 코드",
			src:  `<slide><p>명령은 <code>go build</code>다</p></slide>`,
			want: `p("명령은 " code("go build") "다")`,
		},
		{
			name: "링크",
			src:  `<slide><p><a href="https://example.com">문서</a></p></slide>`,
			want: `p(a[href=https://example.com]("문서"))`,
		},
		{
			name: "줄바꿈",
			src:  `<slide><p>첫 줄<br/>둘째 줄</p></slide>`,
			want: `p("첫 줄" br "둘째 줄")`,
		},
		{
			name: "span 색·굵기",
			src:  `<slide><p><span color="accent1" weight="700">강조</span></p></slide>`,
			want: `p(span[color=accent1 weight=700]("강조"))`,
		},
		{
			name: "text 안의 인라인",
			src:  `<slide><text>본문 <em>강조</em></text></slide>`,
			want: `text("본문 " em("강조"))`,
		},
		{
			name: "블록 ≡ text 역할 등가 — 나란히 파싱된다",
			src:  `<slide><text role="body">본문</text><p>본문</p></slide>`,
			want: `text[role=body]("본문") p("본문")`,
		},
		{
			name: "variant — text와 블록에 열려 있다",
			src:  `<slide><text variant="kicker">개요</text><p variant="lead">본문</p></slide>`,
			want: `text[variant=kicker]("개요") p[variant=lead]("본문")`,
		},
		{
			name: "블록의 인라인 외관 속성 — text와 같은 어휘 (role 제외)",
			src:  `<slide><h3 size="40" color="accent1" align="center">소제</h3></slide>`,
			want: `h3[size=40 color=accent1 align=center]("소제")`,
		},
		{
			name: "pre 코드 블록",
			src:  `<slide><pre>x := 1</pre></slide>`,
			want: `pre("x := 1")`,
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

// 배치 위반·void 위반·모르는 속성 — 진단의 line:col과 문구까지 단정한다.
func TestParseTextSubsetErrors(t *testing.T) {
	tests := []struct {
		name string
		src  string
		// want의 각 원소는 "line:col 메시지조각"이다.
		want []string
	}{
		{
			name: "li가 슬라이드 직속",
			src:  `<slide><li>항목</li></slide>`,
			want: []string{"1:9 <li>는 <ul>/<ol>의 자식으로만 쓸 수 있다"},
		},
		{
			name: "li가 블록 안",
			src:  `<slide><p><li>항목</li></p></slide>`,
			want: []string{"1:12 <li>는 <ul>/<ol>의 자식으로만 쓸 수 있다"},
		},
		{
			name: "중첩 리스트는 아직이다",
			src:  `<slide><ul><li><ul><li>깊다</li></ul></li></ul></slide>`,
			want: []string{"1:17 중첩 리스트는 아직 지원하지 않는다"},
		},
		{
			name: "블록 안의 블록 — p 안의 p",
			src:  `<slide><p><p>본문</p></p></slide>`,
			want: []string{"1:12 <p> 안에 블록 태그 <p>는 올 수 없다"},
		},
		{
			name: "블록 안의 블록 — li 안의 h2",
			src:  `<slide><ul><li><h2>제목</h2></li></ul></slide>`,
			want: []string{"1:17 <li> 안에 블록 태그 <h2>는 올 수 없다"},
		},
		{
			name: "a 안의 a",
			src:  `<slide><p><a href="x"><a href="y">z</a></a></p></slide>`,
			want: []string{"1:24 <a> 안에 <a>를 겹칠 수 없다"},
		},
		{
			name: "a 안의 a — 사이에 인라인이 있어도",
			src:  `<slide><p><a href="x"><strong><a href="y">z</a></strong></a></p></slide>`,
			want: []string{"1:32 <a> 안에 <a>를 겹칠 수 없다"},
		},
		{
			name: "void 위반 — br에 내용",
			src:  `<slide><p><br>텍스트</br></p></slide>`,
			want: []string{"1:12 void 태그 <br>은 내용을 가질 수 없다 (<br/>로 쓴다)"},
		},
		{
			name: "void 위반 — 자기 닫힘이 아닌 br",
			src:  `<slide><p><br></br></p></slide>`,
			want: []string{"1:12 void 태그 <br>은 내용을 가질 수 없다"},
		},
		{
			name: "span의 size는 막혀 있다",
			src:  `<slide><p><span size="20">x</span></p></slide>`,
			want: []string{"1:17 <span>에 알 수 없는 속성 size (쓸 수 있는 속성: color, weight)"},
		},
		{
			name: "a의 속성은 href뿐",
			src:  `<slide><p><a target="_blank">x</a></p></slide>`,
			want: []string{"1:14 <a>에 알 수 없는 속성 target (쓸 수 있는 속성: href)"},
		},
		{
			// 블록은 역할을 표에서만 얻는다 — role 속성을 열면 <p role="title">이
			// 두 표기의 등가(PRD §7.7)를 깬다. 어휘 뒤쪽은 Iteration 3이 더한 자식 배치
			// 속성이다 (블록이 레이아웃 컨테이너의 자식이 될 수 있다 — soml/layout.go).
			name: "블록에는 role이 없다",
			src:  `<slide><h1 role="body">제목</h1></slide>`,
			want: []string{"1:12 <h1>에 알 수 없는 속성 role (쓸 수 있는 속성: variant, size, color, align, anchor, grow, shrink, basis, span, self, order, at, x, y, style)"},
		},
		{
			name: "인라인에는 variant가 없다",
			src:  `<slide><p><strong variant="kicker">x</strong></p></slide>`,
			want: []string{"1:19 <strong>에는 속성을 쓸 수 없다 (variant)"},
		},
		{
			// 리스트 컨테이너의 외관은 여전히 닫혀 있다. 열린 것은 배치뿐이다 —
			// `<row><ul basis="360">`이 유효해야 하므로 (Iteration 3, soml/layout.go).
			name: "리스트 컨테이너에는 variant가 없다",
			src:  `<slide><ul variant="lead"><li>x</li></ul></slide>`,
			want: []string{"1:12 <ul>에 알 수 없는 속성 variant (쓸 수 있는 속성: grow, shrink, basis, span, self, order, at, x, y, style)"},
		},
		{
			name: "li에는 variant가 없다",
			src:  `<slide><ul><li variant="lead">x</li></ul></slide>`,
			want: []string{"1:16 <li>에는 속성을 쓸 수 없다 (variant)"},
		},
		{
			name: "pre 안의 인라인은 아직이다",
			src:  `<slide><pre><code>x</code></pre></slide>`,
			want: []string{"1:14 <pre> 안에는 문자 데이터만 올 수 있다 (<code>는 쓸 수 없다)"},
		},
		{
			name: "블록 안의 text",
			src:  `<slide><p><text>x</text></p></slide>`,
			want: []string{"1:12 <p> 안에는 문자 데이터와 인라인 태그만 올 수 있다 (<text>는 쓸 수 없다)"},
		},
		{
			name: "인라인이 슬라이드 직속",
			src:  `<slide><strong>x</strong></slide>`,
			want: []string{"1:9 <slide>의 자식으로 <strong>는 올 수 없다"},
		},
		{
			name: "ul 안의 문자 데이터",
			src:  `<slide><ul>글자</ul></slide>`,
			want: []string{"1:12 <ul> 안에 문자 데이터를 직접 쓸 수 없다 (<li>로 감싼다)"},
		},
		{
			name: "ul의 자식은 li뿐",
			src:  `<slide><ul><p>x</p></ul></slide>`,
			want: []string{"1:13 <ul>의 자식으로 <p>는 올 수 없다 (쓸 수 있는 태그: <li>)"},
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

// 인라인 경계 공백 계약 — 여기서 접히는 방식이 곧 런 분할(항목 06)의 입력이다.
//
// 계약: 요소(블록이든 인라인이든) **양 끝**의 공백은 들여쓰기이므로 내용이 아니고,
// 형제 **사이**의 공백은 내용이다. `<strong>배포 </strong>자동화`처럼 태그 안쪽 끝에
// 붙인 공백은 잘린다 — 공백은 태그 밖에 쓴다.
func TestParseInlineEdgeSpace(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want string
	}{
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
		{
			name: "연속 공백은 접히지 않는다",
			src:  `<slide><pre>a  b</pre></slide>`,
			want: "a  b",
		},
		{
			name: "양 끝 개행은 하나씩만 벗긴다",
			src:  "<slide><pre>\n\ncode\n\n</pre></slide>",
			want: "\ncode\n",
		},
		{
			name: "CRLF도 개행 하나다",
			src:  "<slide><pre>\r\ncode\r\n</pre></slide>",
			want: "code",
		},
		{
			name: "개행이 없으면 그대로",
			src:  `<slide><pre>code</pre></slide>`,
			want: "code",
		},
		{
			name: "안쪽 들여쓰기는 내용이다",
			src:  "<slide><pre>\n  indented\n</pre></slide>",
			want: "  indented",
		},
		{
			name: "닫는 태그 앞의 들여쓰기는 개행이 아니므로 남는다",
			src:  "<slide><pre>\ncode\n  </pre></slide>",
			want: "code\n  ",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			doc := parseOK(t, tc.src)
			pre := elementAt(t, doc.Content, 0)
			if got := pre.TextContent(); got != tc.want {
				t.Errorf("pre 내용 = %q, want %q", got, tc.want)
			}
		})
	}
}

// 개행뿐인 pre는 빈 요소가 된다 — 벗기고 남은 빈 텍스트 노드를 트리에 남기지 않는다.
func TestParsePreOnlyNewline(t *testing.T) {
	doc := parseOK(t, "<slide><pre>\n</pre></slide>")
	pre := elementAt(t, doc.Content, 0)
	if len(pre.Children) != 0 {
		t.Errorf("자식 %d개 (%s), want 0개", len(pre.Children), dumpNodes(pre.Children))
	}
}

// 태그 → 역할 표는 테마가 아니라 **언어**의 계약이다 (PRD §7.7) — 표가 흔들리면
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
	// 표 밖의 이름: <text>는 role 속성이 역할을 정하고, 컨테이너·인라인은 역할이 없다.
	for _, name := range []string{TextElement, "ul", "ol", "strong", "br", RootElement, "h6"} {
		if role, ok := TagTextRole(name); ok {
			t.Errorf("TagTextRole(%q) = %q — 표에 없어야 한다", name, role)
		}
	}
	// 블록·li 전부가 표에 있어야 한다 — 표에 없는 블록은 해석되지 않은 채 렌더러에 닿는다.
	for _, name := range append(append([]string{}, blockElements...), ListItemElement) {
		if _, ok := TagTextRole(name); !ok {
			t.Errorf("서브셋 태그 %q가 역할 표에 없다", name)
		}
	}
}

// 활성 픽스처 02_text_subset(항목 08이 `_`를 뗐다)의 슬라이드가 파서 어휘와 어긋나지
// 않는지 파싱만 검증한다 — 골든·렌더 검증은 픽스처 러너(internal/testkit)의 몫이고,
// 여기는 파서 계약(진단 0건)만 본다. 이 파일들이 서브셋 항목 전부의 시험대다.
func TestParseTextSubsetFixture(t *testing.T) {
	dir := filepath.Join("..", "testdata", "fixtures", "02_text_subset", "slides")
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
