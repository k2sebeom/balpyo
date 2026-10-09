package soml

import (
	"strings"
	"testing"
)

// parseOK는 진단이 없어야 하는 소스를 파싱한다.
func parseOK(t *testing.T, src string) *SlideDoc {
	t.Helper()
	doc, diags := Parse("slides/001_title.slide", src)
	if len(diags) != 0 {
		t.Fatalf("진단이 없어야 한다:\n%s", diags.Err())
	}
	return doc
}

// elementAt은 콘텐츠 노드를 요소로 꺼낸다.
func elementAt(t *testing.T, nodes []Node, i int) *Element {
	t.Helper()
	if i >= len(nodes) {
		t.Fatalf("콘텐츠 노드가 %d개뿐이다 (want > %d)", len(nodes), i)
	}
	elem, ok := nodes[i].(*Element)
	if !ok {
		t.Fatalf("노드 %d가 요소가 아니다: %T", i, nodes[i])
	}
	return elem
}

// treeCase는 정상 트리 케이스다 — want는 dumpNodes 표현이다.
type treeCase struct {
	name string
	src  string
	want string
}

func runTreeCases(t *testing.T, tests []treeCase) {
	t.Helper()
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			doc := parseOK(t, tc.src)
			if got := dumpNodes(doc.Content); got != tc.want {
				t.Errorf("트리 = %s\nwant %s", got, tc.want)
			}
		})
	}
}

// diagCase는 진단 케이스다 — want의 각 원소는 "line:col 메시지조각"이고 순서대로 대조한다.
// want가 비면 진단 0건을 단정한다.
type diagCase struct {
	name string
	src  string
	want []string
}

func runDiagCases(t *testing.T, tests []diagCase) {
	t.Helper()
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
				if diags[i].Severity != SeverityError {
					t.Errorf("진단[%d]은 에러여야 한다: %v", i, diags[i])
				}
			}
			if len(diags) != len(tc.want) {
				t.Errorf("진단 %d개, want %d개:\n%s", len(diags), len(tc.want), diags.Err())
			}
		})
	}
}

func TestParseSlide(t *testing.T) {
	src := `<slide id="architecture">
  <text role="title" size="48" color="fg" align="center">2026 아키텍처 리뷰</text>
  <text role="body">
    Platform Team의 상반기 회고
  </text>
  <note>인사 30초. 청중에 인프라팀 포함.</note>
</slide>
`
	doc := parseOK(t, src)

	if doc.ID != "architecture" {
		t.Errorf("ID = %q, want %q", doc.ID, "architecture")
	}
	if doc.Root == nil || doc.Root.Name != "slide" {
		t.Fatalf("루트 = %+v", doc.Root)
	}
	// `<note>`는 콘텐츠가 아니라 슬라이드 메타데이터다.
	if len(doc.Content) != 2 {
		t.Fatalf("콘텐츠 노드 %d개, want 2개 (<note> 제외)", len(doc.Content))
	}
	if doc.Notes != "인사 30초. 청중에 인프라팀 포함." {
		t.Errorf("Notes = %q", doc.Notes)
	}

	title := elementAt(t, doc.Content, 0)
	if title.Name != "text" || title.TextContent() != "2026 아키텍처 리뷰" {
		t.Errorf("첫 노드 = <%s>%q", title.Name, title.TextContent())
	}
	wantAttrs := map[string]string{"role": "title", "size": "48", "color": "fg", "align": "center"}
	if len(title.Attrs) != len(wantAttrs) {
		t.Fatalf("속성 %d개, want %d개", len(title.Attrs), len(wantAttrs))
	}
	for name, want := range wantAttrs {
		attr, ok := title.Attr(name)
		if !ok {
			t.Fatalf("속성 %s가 없다", name)
		}
		if attr.Value != want {
			t.Errorf("속성 %s = %q, want %q", name, attr.Value, want)
		}
		// 값 스팬은 인용 부호를 제외한다 — core가 값 어휘를 진단할 때 밑줄이 값에 그어진다.
		if got := attr.ValueSpan.Text(src); got != want {
			t.Errorf("속성 %s의 값 스팬 = %q, want %q", name, got, want)
		}
	}

	if got := title.Span.Text(src); !strings.HasPrefix(got, "<text role") || !strings.HasSuffix(got, "</text>") {
		t.Errorf("요소 스팬 = %q", got)
	}
	body := elementAt(t, doc.Content, 1)
	if got := body.TextContent(); got != "Platform Team의 상반기 회고" {
		t.Errorf("본문 = %q", got)
	}
}

// 무손실 계약은 파서를 지나서도 유지된다 — format이 되돌려 쓸 근거가 덱 모델에 남는다.
// 깨진 입력에서도 성립해야 한다.
func TestParseKeepsSourceTokens(t *testing.T) {
	for _, src := range []string{
		"",
		"<slide></slide>",
		"<!-- 메모 -->\n<slide>\n  <text>제목</text>\n</slide>\n",
		"<slide id='a'  skip >\n  <note>노트</note>\n</slide>",
		"<slide><txt>x</txt>",
		"<slide id=\"a>\n</slide>",
		"</slide>",
		"<slide @#$>",
	} {
		doc, _ := Parse("a.slide", src)
		if got := doc.Tokens.Source(); got != src {
			t.Errorf("round-trip 실패\n입력: %q\n복원: %q", src, got)
		}
	}
}

// 슬라이드 속성 — 파서는 layout·justify의 원문 값과 **값 구간**만 싣는다 (어휘 검사는 core다).
// 구간은 인용 부호를 제외한다 — core의 진단이 부호가 아니라 값에 밑줄을 긋는다.
func TestParseSlideAttributes(t *testing.T) {
	tests := []struct {
		name        string
		src         string
		wantID      string
		wantSkip    bool
		wantLayout  string
		wantJustify string
	}{
		{name: "속성 없음", src: `<slide></slide>`},
		{name: "홑따옴표 id", src: `<slide id='intro'></slide>`, wantID: "intro"},
		// 값 없는 불리언 속성은 참이다 (HTML 관례).
		{name: "값 없는 skip", src: `<slide id="a" skip></slide>`, wantID: "a", wantSkip: true},
		{name: "skip=true", src: `<slide skip="true"></slide>`, wantSkip: true},
		{name: "skip=false", src: `<slide skip="false"></slide>`},
		{name: "layout", src: `<slide layout="two-column" id="hero"></slide>`, wantID: "hero", wantLayout: "two-column"},
		{name: "어휘 밖 layout도 원문 그대로", src: `<slide layout="titel"></slide>`, wantLayout: "titel"},
		{name: "justify", src: `<slide justify="between"></slide>`, wantJustify: "between"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			doc := parseOK(t, tc.src)
			if doc.ID != tc.wantID || doc.Skip != tc.wantSkip || doc.Layout != tc.wantLayout || doc.Justify != tc.wantJustify {
				t.Errorf("ID/Skip/Layout/Justify = %q/%v/%q/%q, want %q/%v/%q/%q",
					doc.ID, doc.Skip, doc.Layout, doc.Justify, tc.wantID, tc.wantSkip, tc.wantLayout, tc.wantJustify)
			}
			if tc.wantLayout != "" {
				if got := doc.LayoutSpan.Text(tc.src); got != tc.wantLayout {
					t.Errorf("LayoutSpan = %q, want %q", got, tc.wantLayout)
				}
			}
			if tc.wantJustify != "" {
				if got := doc.JustifySpan.Text(tc.src); got != tc.wantJustify {
					t.Errorf("JustifySpan = %q, want %q", got, tc.wantJustify)
				}
			}
		})
	}
}

// 표준 레이아웃 카탈로그는 11종이고 순서가 문서·진단 순서다.
func TestSlideLayoutsCatalog(t *testing.T) {
	got := make([]string, len(SlideLayouts()))
	for i, l := range SlideLayouts() {
		got[i] = string(l)
	}
	want := "title section-divider content two-column image-full image-text quote code stat canvas end"
	if strings.Join(got, " ") != want {
		t.Errorf("SlideLayouts() = %q\nwant %q", strings.Join(got, " "), want)
	}
	if DefaultSlideLayout != LayoutContent || DefaultSlot != "body" {
		t.Errorf("DefaultSlideLayout/DefaultSlot = %q/%q, want content/body", DefaultSlideLayout, DefaultSlot)
	}
}

// 공백 접기는 결정적이어야 한다 — 웹은 브라우저가 접고 PPTX는 접지 않으므로 파서가 접는다.
func TestParseCollapsesWhitespace(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want string
	}{
		{name: "그대로", src: "<slide><text>제목</text></slide>", want: "제목"},
		{name: "양 끝을 자른다", src: "<slide><text>\n  제목\n  </text></slide>", want: "제목"},
		{name: "런은 한 칸", src: "<slide><text>두   낱말</text></slide>", want: "두 낱말"},
		{name: "탭과 줄바꿈도 공백", src: "<slide><text>a\tb\nc</text></slide>", want: "a b c"},
		{name: "공백만이면 비운다", src: "<slide><text>\n   \n</text></slide>", want: ""},
		// 비ASCII 공백은 저자가 의도해 넣은 문자다.
		{name: "U+00A0은 남는다", src: "<slide><text>한 글</text></slide>", want: "한 글"},
		{name: "노트도 같은 규칙", src: "<slide><note>\n  줄1\n  줄2\n</note></slide>", want: "줄1 줄2"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			doc := parseOK(t, tc.src)
			elem := doc.Root.Children[0].(*Element)
			if got := elem.TextContent(); got != tc.want {
				t.Errorf("텍스트 = %q, want %q", got, tc.want)
			}
			if tc.want == "" && len(elem.Children) != 0 {
				t.Errorf("빈 텍스트는 노드를 남기지 않는다: %d개", len(elem.Children))
			}
		})
	}
}

// 요소만 담는 모델에서 요소 사이 공백은 내용이 아니다 — 콘텐츠 트리에 텍스트 노드가 없다.
func TestParseDropsLayoutWhitespace(t *testing.T) {
	doc := parseOK(t, "<slide>\n  <text>a</text>\n  <text>b</text>\n</slide>\n")
	if got := dumpNodes(doc.Root.Children); got != `text("a") text("b")` {
		t.Errorf("자식 = %s", got)
	}
	doc = parseOK(t, `<slide><text/><note/></slide>`)
	if len(doc.Content) != 1 || doc.Notes != "" {
		t.Errorf("콘텐츠 %d개, Notes %q", len(doc.Content), doc.Notes)
	}
}

// 요소의 Span은 여는 `<`부터 닫는 태그의 `>`(자기 닫힘은 `/>`)까지다 — 진단·지오메트리가
// 저자의 태그를 정확히 가리켜야 한다.
func TestParseElementSpans(t *testing.T) {
	tests := []struct {
		src  string
		path []int // 콘텐츠에서 자식 색인을 따라 내려간다
		want string
	}{
		{"<slide>\n  <shape type=\"rect\">\n    <text>배지</text>\n  </shape>\n</slide>", []int{0}, "2:3-4:11"},
		{"<slide>\n  <shape type=\"rect\">\n    <text>배지</text>\n  </shape>\n</slide>", []int{0, 0}, "3:5-3:20"},
		{"<slide>\n  <shape type=\"rect\" id=\"db\"/>\n</slide>", []int{0}, "2:3-2:31"},
		{"<slide>\n  <image src=\"assets/wide.png\"/>\n</slide>", []int{0}, "2:3-2:33"},
		{"<slide>\n  <row gap=\"md\">\n    <spacer/>\n  </row>\n</slide>", []int{0}, "2:3-4:9"},
		{"<slide>\n  <row gap=\"md\">\n    <spacer/>\n  </row>\n</slide>", []int{0, 0}, "3:5-3:14"},
	}
	for _, tc := range tests {
		nodes := parseOK(t, tc.src).Content
		var elem *Element
		for _, i := range tc.path {
			elem = elementAt(t, nodes, i)
			nodes = elem.Children
		}
		if got := elem.Span.Start.String() + "-" + elem.Span.End.String(); got != tc.want {
			t.Errorf("%q %v의 <%s> Span = %s, want %s", tc.src, tc.path, elem.Name, got, tc.want)
		}
	}
}

// 문서 층의 진단 — line:col과 문구까지 단정한다. 진단은 소스 순서로 정렬되어 저자가 위에서
// 아래로 고칠 수 있다.
func TestParseErrors(t *testing.T) {
	runDiagCases(t, []diagCase{
		{
			name: "오타 태그",
			src:  "<slide>\n  <txt>제목</txt>\n</slide>",
			want: []string{"2:4 알 수 없는 태그 <txt>"},
		},
		{
			// 닫힌 어휘 위반이면 허용 어휘 전부를 함께 낸다 — 카탈로그가 닫혀 있다는 증거다.
			name: "어휘 밖 태그는 허용 어휘를 싣는다",
			src:  "<slide><h6>제목</h6></slide>",
			want: []string{"1:9 알 수 없는 태그 <h6> (쓸 수 있는 태그: <a>, <blockquote>, <br>, <chart>, <code>, <col>, <em>, <grid>, <h1>, <h2>, <h3>, <image>, <li>, <note>, <ol>, <p>, <point>, <pre>, <row>, <shape>, <slide>, <spacer>, <span>, <stack>, <strong>, <table>, <td>, <text>, <th>, <tr>, <ul>)"},
		},
		{
			name: "알 수 없는 속성",
			src:  `<slide reveal="fade"></slide>`,
			want: []string{"1:8 알 수 없는 속성 reveal (쓸 수 있는 속성: layout, id, skip, justify)"},
		},
		{
			name: "<note>에는 속성이 없다",
			src:  `<slide><note role="body">a</note></slide>`,
			want: []string{"1:14 <note>에는 속성을 쓸 수 없다"},
		},
		{
			name: "루트가 <slide>가 아니다",
			src:  "<text>제목</text>",
			want: []string{"1:2 루트 요소는 <slide>여야 한다"},
		},
		{
			name: "루트 요소가 없다",
			src:  "<!-- 빈 파일 -->\n",
			want: []string{"1:1 루트 요소 <slide>가 없다"},
		},
		{
			name: "루트 요소가 둘",
			src:  "<slide></slide>\n<slide></slide>",
			want: []string{"2:2 루트 요소 <slide>는 파일에 하나여야 한다"},
		},
		{
			name: "닫히지 않은 태그",
			src:  "<slide>\n  <text>제목\n</slide>",
			want: []string{"2:4 <text> 태그가 </slide>보다 먼저 닫혀야 한다"},
		},
		{
			name: "파일 끝까지 닫히지 않았다",
			src:  "<slide>\n  <text>제목</text>\n",
			want: []string{"1:2 <slide> 태그가 닫히지 않았다"},
		},
		{
			name: "짝이 맞지 않는 닫는 태그",
			src:  "<slide></text></slide>",
			want: []string{"1:8 </text>와 짝이 맞는 여는 태그가 없다"},
		},
		{
			name: "중복 <note>",
			src:  "<slide>\n  <note>첫</note>\n  <note>둘</note>\n</slide>",
			want: []string{"3:4 <note>는 슬라이드당 하나만"},
		},
		{
			name: "루트 레벨 문자 데이터",
			src:  "안녕\n<slide></slide>",
			want: []string{"1:1 요소 밖에 문자 데이터가 있다"},
		},
		{
			name: "<slide> 안의 문자 데이터",
			src:  "<slide>\n  제목\n</slide>",
			want: []string{"2:3 <slide> 안에 문자 데이터를 직접 쓸 수 없다"},
		},
		{
			name: "인용 부호 미닫힘",
			src:  "<slide id=\"a>\n</slide>",
			want: []string{
				"1:2 <slide> 태그가 `>`로 닫히지 않았다",
				"1:11 속성 id의 값에 닫는 인용 부호가 없다",
			},
		},
		{
			name: "인용 부호 없는 값",
			src:  `<slide id=a></slide>`,
			want: []string{`1:11 속성 값은 인용 부호로 감싼다`},
		},
		{
			name: "값이 없다",
			src:  `<slide id=></slide>`,
			want: []string{"1:8 속성 id에 `=`만 있고 값이 없다"},
		},
		{name: "값 없는 id", src: `<slide id></slide>`, want: []string{"1:8 id 속성에는 값이 필요하다"}},
		{name: "id가 비었다", src: `<slide id=""></slide>`, want: []string{"1:12 id 속성이 비어 있다"}},
		{name: "skip 값이 어휘 밖", src: `<slide skip="maybe"></slide>`, want: []string{"1:14 skip 값은 true 또는 false다"}},
		{name: "값 없는 layout", src: `<slide layout></slide>`, want: []string{"1:8 layout 속성에는 값이 필요하다"}},
		{name: "layout이 비었다", src: `<slide layout=""></slide>`, want: []string{"1:16 layout 속성이 비어 있다"}},
		{name: "값 없는 justify", src: `<slide justify></slide>`, want: []string{"1:8 justify 속성에는 값이 필요하다"}},
		{name: "justify가 비었다", src: `<slide justify=""></slide>`, want: []string{"1:17 justify 속성이 비어 있다"}},
		{
			name: "속성 중복",
			src:  `<slide id="a" id="b"></slide>`,
			want: []string{"1:15 속성 id가 중복됐다 (앞의 것만 쓴다)"},
		},
		{
			name: "쓸 수 없는 문자",
			src:  `<slide @></slide>`,
			want: []string{`1:8 태그 안에 쓸 수 없는 문자 "@"`},
		},
		{
			name: "잘못된 중첩 — 텍스트 안의 텍스트",
			src:  `<slide><text><text>a</text></text></slide>`,
			want: []string{"1:15 <text> 안에는 문자 데이터와 인라인 태그만 올 수 있다"},
		},
		{
			name: "잘못된 중첩 — 슬라이드 안의 슬라이드",
			src:  `<slide><slide></slide></slide>`,
			want: []string{"1:9 <slide>의 자식으로 <slide>는 올 수 없다"},
		},
		{
			// `<` + 공백은 마크업이 아니므로 문자 데이터다.
			name: "마크업이 아닌 <는 문자 데이터다",
			src:  `<slide>< ></slide>`,
			want: []string{"1:8 <slide> 안에 문자 데이터를 직접 쓸 수 없다"},
		},
		{
			name: "닫는 태그에 이름이 없다",
			src:  `<slide></ ></slide>`,
			want: []string{"1:8 `</` 뒤에 태그 이름이 없다"},
		},
		{
			name: "닫히지 않은 주석",
			src:  "<slide>\n<!-- 메모\n</slide>",
			want: []string{
				"1:2 <slide> 태그가 닫히지 않았다",
				"2:1 주석이 `-->`로 닫히지 않았다",
			},
		},
		{
			name: "한 파일의 진단을 모아 낸다",
			src:  "<slide reveal=\"a\">\n  <txt>제목</txt>\n  <text badattr=\"1\">본문</text>\n</slide>",
			want: []string{
				"1:8 알 수 없는 속성 reveal",
				"2:4 알 수 없는 태그 <txt>",
				"3:9 알 수 없는 속성 badattr",
			},
		},
	})
}

// 파서는 어떤 입력에서도 멈춘다 — 에러 복구가 있는 손으로 쓴 루프이므로 진행 보장을 검사한다.
func TestParseAlwaysTerminates(t *testing.T) {
	for _, src := range []string{
		strings.Repeat("<@ '\" ->/", 200),
		strings.Repeat("<slide>", 200),
		strings.Repeat("</text>", 200),
		strings.Repeat("<text a=", 200),
		"<slide" + strings.Repeat(" id=\"a\"", 200),
	} {
		doc, diags := Parse("a.slide", src)
		if !diags.HasErrors() {
			t.Errorf("%q: 진단이 있어야 한다", src[:min(len(src), 20)])
		}
		if doc.Tokens.Source() != src {
			t.Error("round-trip 실패")
		}
	}
}

// Col은 룬 기준이다 — 한글이 섞인 줄에서도 진단이 글자를 가리켜야 한다 (Pos 참조).
func TestParseColumnIsRuneBased(t *testing.T) {
	src := "<slide>\n  <text>한글 섞인 줄</text><txt>x</txt>\n</slide>"
	_, diags := Parse("a.slide", src)
	if len(diags) != 1 {
		t.Fatalf("진단 %d개, want 1개:\n%s", len(diags), diags.Err())
	}
	// `  <text>한글 섞인 줄</text><` → `txt`는 2줄 24칸이다 (한글 6글자 = 6칸).
	if got := diags[0].Span.Start.String(); got != "2:24" {
		t.Errorf("위치 = %s, want 2:24 (%s)", got, diags[0].Message)
	}
	if got := diags[0].Span.Text(src); got != "txt" {
		t.Errorf("스팬이 가리키는 원문 = %q, want %q", got, "txt")
	}
}

// 진단은 파일 경로를 갖는다 — 에디터가 클릭할 수 있는 `file:line:col` 형태가 된다.
func TestParseDiagnosticCarriesFile(t *testing.T) {
	_, diags := Parse("slides/ch/002_intro.slide", "<slide><txt/></slide>")
	if len(diags) != 1 {
		t.Fatalf("진단 %d개, want 1개", len(diags))
	}
	if got, want := diags[0].String(), "slides/ch/002_intro.slide:1:9: error: "; !strings.HasPrefix(got, want) {
		t.Errorf("진단 = %q, want %q로 시작", got, want)
	}
}
