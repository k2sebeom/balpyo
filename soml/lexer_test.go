package soml

import (
	"strings"
	"testing"
)

// 무손실 계약: 어떤 입력이든 토큰을 이어 붙이면 원문이 된다.
func TestTokenizeIsLossless(t *testing.T) {
	inputs := []string{
		"",
		"<slide></slide>",
		"<!-- slides/001_title.slide -->\n<slide layout=\"title\" skip>\n  <text>제목</text>\n</slide>\n",
		"<slide id='a'  layout = \"title\" ><br/></slide>",
		"텍스트만 있는 파일",
		"<slide>\n\n  여러 줄\t탭\r\n</slide>",
		// 깨진 입력 — lsp가 매 키 입력마다 만나는 상태.
		"<shape type=\"round",
		"<slide",
		"<slide>",
		"</",
		"<!-- 닫히지 않은 주석",
		"<slide @#$>",
		"a < b",
	}
	for _, src := range inputs {
		if got := Tokenize(src).Source(); got != src {
			t.Errorf("round-trip 실패\n입력: %q\n복원: %q", src, got)
		}
	}
}

// 토큰 열의 마지막은 항상 EOF 하나다.
func TestTokenizeEndsWithEOF(t *testing.T) {
	for _, src := range []string{"", "<slide>", "텍스트"} {
		toks := Tokenize(src)
		if len(toks) == 0 {
			t.Fatalf("%q: 토큰이 없다", src)
		}
		last := toks[len(toks)-1]
		if last.Kind != TokenEOF {
			t.Errorf("%q: 마지막 토큰이 %v", src, last.Kind)
		}
		if last.Span.Start.Offset != len(src) {
			t.Errorf("%q: EOF 오프셋 %d, want %d", src, last.Span.Start.Offset, len(src))
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
			name:  "값 없는 불리언 속성",
			src:   `<slide skip>`,
			want:  []TokenKind{TokenTagStart, TokenName, TokenWhitespace, TokenName, TokenTagEnd, TokenEOF},
			texts: []string{"<", "slide", " ", "skip", ">"},
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
			name:  "하이픈·숫자 이름",
			src:   `<dim-card><h2>`,
			want:  []TokenKind{TokenTagStart, TokenName, TokenTagEnd, TokenTagStart, TokenName, TokenTagEnd, TokenEOF},
			texts: []string{"<", "dim-card", ">", "<", "h2", ">"},
		},
		{
			name:  "홑따옴표 값",
			src:   `<slide id='a'>`,
			want:  []TokenKind{TokenTagStart, TokenName, TokenWhitespace, TokenName, TokenEquals, TokenString, TokenTagEnd, TokenEOF},
			texts: []string{"<", "slide", " ", "id", "=", "'a'", ">"},
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
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			toks := Tokenize(tc.src)
			var got []TokenKind
			for _, tok := range toks {
				got = append(got, tok.Kind)
			}
			if !equalKinds(got, tc.want) {
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

// 위치는 진단의 값이다 — 줄바꿈과 한글을 지나서도 정확해야 한다.
func TestTokenizePositions(t *testing.T) {
	src := "<slide>\n  <text>한글 텍스트</text>\n</slide>"
	toks := Tokenize(src)

	// 두 번째 줄의 `<text>` 여는 `<`: 2줄 3칸.
	var textTag *Token
	for i, tok := range toks {
		if tok.Kind == TokenName && tok.Text == "text" {
			textTag = &toks[i-1]
			break
		}
	}
	if textTag == nil {
		t.Fatal("`text` 이름 토큰을 찾지 못했다")
	}
	if textTag.Span.Start.Line != 2 || textTag.Span.Start.Col != 3 {
		t.Errorf("`<` 위치 = %v, want 2:3", textTag.Span.Start)
	}

	// 한글 4글자를 지난 `</text>`는 룬 기준 칸수여야 한다.
	// `  <text>한글 텍스트` → `<`가 3칸이므로 `</`는 3 + len("<text>") + 6글자 = 15칸.
	var closeTag *Token
	for i := range toks {
		if toks[i].Kind == TokenTagStartClose {
			closeTag = &toks[i]
			break
		}
	}
	if closeTag == nil {
		t.Fatal("닫는 태그 토큰을 찾지 못했다")
	}
	if closeTag.Span.Start.Line != 2 || closeTag.Span.Start.Col != 15 {
		t.Errorf("`</` 위치 = %v, want 2:15 (Col은 룬 기준)", closeTag.Span.Start)
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
	if !equalKinds(kinds, want) {
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

// 렉서는 어떤 입력에서도 멈추지 않고 진행한다 — 무한 루프가 없음을 확인한다.
func TestTokenizeAlwaysProgresses(t *testing.T) {
	src := strings.Repeat("<@ '\" ->/", 200)
	toks := Tokenize(src)
	if toks.Source() != src {
		t.Error("round-trip 실패")
	}
	for i, tok := range toks {
		if tok.Kind != TokenEOF && tok.Span.Len() == 0 {
			t.Fatalf("토큰 %d(%v)가 0바이트를 소비했다", i, tok.Kind)
		}
	}
}

func equalKinds(a, b []TokenKind) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
