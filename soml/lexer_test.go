package soml

import (
	"slices"
	"strings"
	"testing"
)

// 무손실 계약: 어떤 입력이든 토큰을 이어 붙이면 원문이 된다. 렉서는 깨진 입력(lsp가 매 키
// 입력마다 만나는 상태)에서도 멈추지 않고 진행하며, 토큰 열은 원문 끝의 EOF 하나로 끝난다.
func TestTokenizeIsLossless(t *testing.T) {
	inputs := []string{
		"",
		"<slide></slide>",
		"<!-- slides/001_title.slide -->\n<slide layout=\"title\" skip>\n  <text>제목</text>\n</slide>\n",
		"<slide id='a'  layout = \"title\" ><br/></slide>",
		"텍스트만 있는 파일",
		"<slide>\n\n  여러 줄\t탭\r\n</slide>",
		"<shape type=\"round",
		"<slide",
		"</",
		"<!-- 닫히지 않은 주석",
		"<slide @#$>",
		"<p>a < b && c &lt; d</p>",
		"<pre><![CDATA[<slide id=\"x\">]]></pre>",
		"<pre><![CDATA[닫히지 않은",
		"\uFEFF<slide></slide>",
		"\uFEFF",
		"<slide>\uFEFF</slide>",
		strings.Repeat("<@ '\" ->/", 200),
	}
	for _, src := range inputs {
		toks := Tokenize(src)
		if got := toks.Source(); got != src {
			t.Errorf("round-trip 실패\n입력: %q\n복원: %q", src, got)
		}
		for i, tok := range toks[:len(toks)-1] {
			if tok.Kind == TokenEOF || tok.Span.Len() == 0 {
				t.Fatalf("%q: 토큰 %d(%v)가 0바이트를 소비했다", src[:min(len(src), 20)], i, tok.Kind)
			}
		}
		if last := toks[len(toks)-1]; last.Kind != TokenEOF || last.Span.Start.Offset != len(src) {
			t.Errorf("%q: 마지막 토큰 %v @%d, want EOF @%d", src[:min(len(src), 20)], last.Kind, last.Span.Start.Offset, len(src))
		}
	}
}

func TestTokenizeKinds(t *testing.T) {
	tests := []struct {
		name  string
		src   string
		want  []TokenKind
		texts []string
	}{
		{
			name: "여는 태그와 속성",
			src:  `<slide layout="title">`,
			want: []TokenKind{TokenTagStart, TokenName, TokenWhitespace, TokenName, TokenEquals, TokenString, TokenTagEnd, TokenEOF},
			// 인용 부호는 토큰 안에 남는다 — format의 무손실 조건.
			texts: []string{"<", "slide", " ", "layout", "=", `"title"`, ">", ""},
		},
		{
			name:  "자기 닫는 태그",
			src:   `<br/>`,
			want:  []TokenKind{TokenTagStart, TokenName, TokenTagEndSelfClose, TokenEOF},
			texts: []string{"<", "br", "/>"},
		},
		{
			name:  "닫는 태그",
			src:   `</slide>`,
			want:  []TokenKind{TokenTagStartClose, TokenName, TokenTagEnd, TokenEOF},
			texts: []string{"</", "slide", ">"},
		},
		{
			name:  "주석은 트리비아 토큰",
			src:   "<!-- 메모 --><slide>",
			want:  []TokenKind{TokenComment, TokenTagStart, TokenName, TokenTagEnd, TokenEOF},
			texts: []string{"<!-- 메모 -->", "<", "slide", ">"},
		},
		{
			name:  "요소 사이 공백은 텍스트다",
			src:   "<slide>\n  글\n</slide>",
			want:  []TokenKind{TokenTagStart, TokenName, TokenTagEnd, TokenText, TokenTagStartClose, TokenName, TokenTagEnd, TokenEOF},
			texts: []string{"<", "slide", ">", "\n  글\n", "</", "slide", ">"},
		},
		{
			name: "닫히지 않은 값은 줄 끝에서 멈춘다",
			src:  "<shape type=\"round\n<slide>",
			want: []TokenKind{
				TokenTagStart, TokenName, TokenWhitespace, TokenName, TokenEquals, TokenStringUnterminated,
				TokenWhitespace, TokenTagStart, TokenName, TokenTagEnd, TokenEOF,
			},
			texts: []string{"<", "shape", " ", "type", "=", `"round`, "\n", "<", "slide", ">"},
		},
		{
			name:  "닫히지 않은 주석은 끝까지 삼킨다",
			src:   "<!-- 메모\n계속",
			want:  []TokenKind{TokenCommentUnterminated, TokenEOF},
			texts: []string{"<!-- 메모\n계속"},
		},
		{
			name:  "알 수 없는 문자는 Invalid 하나",
			src:   `<slide @>`,
			want:  []TokenKind{TokenTagStart, TokenName, TokenWhitespace, TokenInvalid, TokenTagEnd, TokenEOF},
			texts: []string{"<", "slide", " ", "@", ">"},
		},
		{
			name:  "Invalid는 UTF-8을 쪼개지 않는다",
			src:   `<slide 한>`,
			want:  []TokenKind{TokenTagStart, TokenName, TokenWhitespace, TokenInvalid, TokenTagEnd, TokenEOF},
			texts: []string{"<", "slide", " ", "한", ">"},
		},
		{
			// 마크업이 아닌 `<`는 텍스트에 남고 텍스트 토큰을 끊지 않는다.
			name:  "마크업이 아닌 <는 텍스트다",
			src:   `a < b <p>`,
			want:  []TokenKind{TokenText, TokenTagStart, TokenName, TokenTagEnd, TokenEOF},
			texts: []string{"a < b ", "<", "p", ">"},
		},
		{
			name:  "태그 안의 마크업 아닌 <는 Invalid 하나다",
			src:   `<slide <>`,
			want:  []TokenKind{TokenTagStart, TokenName, TokenWhitespace, TokenInvalid, TokenTagEnd, TokenEOF},
			texts: []string{"<", "slide", " ", "<", ">"},
		},
		{
			name: "CDATA는 토큰 하나다",
			src:  `<pre><![CDATA[<slide>]]></pre>`,
			want: []TokenKind{
				TokenTagStart, TokenName, TokenTagEnd, TokenCData,
				TokenTagStartClose, TokenName, TokenTagEnd, TokenEOF,
			},
			texts: []string{"<", "pre", ">", `<![CDATA[<slide>]]>`, "</", "pre", ">"},
		},
		{
			name:  "닫히지 않은 CDATA는 끝까지 삼킨다",
			src:   "<![CDATA[열린 채\n계속",
			want:  []TokenKind{TokenCDataUnterminated, TokenEOF},
			texts: []string{"<![CDATA[열린 채\n계속"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			toks := Tokenize(tc.src)
			var got []TokenKind
			for _, tok := range toks {
				got = append(got, tok.Kind)
			}
			if !slices.Equal(got, tc.want) {
				t.Fatalf("종류 열 불일치\ngot:  %v\nwant: %v", got, tc.want)
			}
			for i, want := range tc.texts {
				if toks[i].Text != want {
					t.Errorf("토큰 %d Text = %q, want %q", i, toks[i].Text, want)
				}
			}
			// 모든 토큰의 Span은 원문을 그대로 가리켜야 한다.
			for _, tok := range toks {
				if got := tok.Span.Text(tc.src); got != tok.Text {
					t.Errorf("%v: Span이 가리키는 원문 %q != Text %q", tok.Kind, got, tok.Text)
				}
			}
		})
	}
}

func TestWithoutTrivia(t *testing.T) {
	src := "<!-- 메모 -->\n<slide  skip>"
	got := Tokenize(src).WithoutTrivia()
	want := []TokenKind{TokenText, TokenTagStart, TokenName, TokenName, TokenTagEnd}
	var kinds []TokenKind
	for _, tok := range got {
		kinds = append(kinds, tok.Kind)
	}
	if !slices.Equal(kinds, want) {
		t.Errorf("got %v, want %v", kinds, want)
	}
	// 주석 바로 뒤 줄바꿈은 텍스트이므로 남는다 — 트리비아 판정은 종류로만 한다.
	if got[0].Text != "\n" {
		t.Errorf("첫 토큰 Text = %q, want %q", got[0].Text, "\n")
	}
}

func TestTokenKindString(t *testing.T) {
	if got := TokenName.String(); got != "Name" {
		t.Errorf("TokenName.String() = %q", got)
	}
	if got := TokenKind(999).String(); got != "Unknown" {
		t.Errorf("알 수 없는 종류 = %q", got)
	}
}
