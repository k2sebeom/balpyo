package soml

// TokenKind는 토큰의 종류다.
type TokenKind int

const (
	// TokenEOF는 소스의 끝이다. 항상 토큰 열의 마지막 원소이며 Text는 비어 있다.
	TokenEOF TokenKind = iota

	// TokenText는 요소 사이의 문자 데이터다. 공백도 여기 포함된다 —
	// 텍스트 콘텐츠의 공백은 트리비아가 아니라 내용이다.
	TokenText

	// TokenComment는 `<!-- ... -->` 주석 전체다 (구분자 포함).
	TokenComment
	// TokenCommentUnterminated는 `-->`로 닫히지 않고 소스가 끝난 주석이다.
	TokenCommentUnterminated

	// TokenCData는 `<![CDATA[ ... ]]>` 섹션 전체다 (구분자 포함). 트리비아가 **아니다** —
	// 내용은 문자 데이터이고, 그 안에서는 마크업도 문자 참조도 해석되지 않는다 (ADR 0037).
	TokenCData
	// TokenCDataUnterminated는 `]]>`로 닫히지 않고 소스가 끝난 CDATA다.
	TokenCDataUnterminated

	// TokenTagStart는 여는 태그의 `<`다.
	TokenTagStart
	// TokenTagStartClose는 닫는 태그의 `</`다.
	TokenTagStartClose
	// TokenTagEnd는 태그를 닫는 `>`다.
	TokenTagEnd
	// TokenTagEndSelfClose는 자기 닫는 태그의 `/>`다.
	TokenTagEndSelfClose

	// TokenName은 태그 이름 또는 속성 이름이다.
	TokenName
	// TokenEquals는 속성의 `=`다.
	TokenEquals
	// TokenString은 인용된 속성 값이다. **인용 부호를 포함**한다 — 어떤 부호를 썼는지가
	// format의 무손실 조건이다.
	TokenString
	// TokenStringUnterminated는 닫는 인용 부호 없이 줄 또는 소스가 끝난 값이다.
	TokenStringUnterminated

	// TokenWhitespace는 태그 내부의 공백이다 (`<slide  id="a">`의 두 칸).
	TokenWhitespace

	// TokenInvalid는 어떤 규칙에도 맞지 않는 룬 하나다. 렉서는 여기서 멈추지 않는다 —
	// 에러 복구는 파서의 몫이고, 렉서의 계약은 무손실이다.
	TokenInvalid
)

// tokenKindNames는 진단·테스트 출력용 이름표다.
var tokenKindNames = map[TokenKind]string{
	TokenEOF:                 "EOF",
	TokenText:                "Text",
	TokenComment:             "Comment",
	TokenCommentUnterminated: "CommentUnterminated",
	TokenCData:               "CData",
	TokenCDataUnterminated:   "CDataUnterminated",
	TokenTagStart:            "TagStart",
	TokenTagStartClose:       "TagStartClose",
	TokenTagEnd:              "TagEnd",
	TokenTagEndSelfClose:     "TagEndSelfClose",
	TokenName:                "Name",
	TokenEquals:              "Equals",
	TokenString:              "String",
	TokenStringUnterminated:  "StringUnterminated",
	TokenWhitespace:          "Whitespace",
	TokenInvalid:             "Invalid",
}

// String은 종류 이름을 돌려준다.
func (k TokenKind) String() string {
	if name, ok := tokenKindNames[k]; ok {
		return name
	}
	return "Unknown"
}

// IsTrivia는 이 종류가 트리비아인지 보고한다 — 문서의 의미에 기여하지 않지만
// 원문 복원에 필요한 토큰이다. 파서는 트리비아를 건너뛰며 읽고, format은 보존한다.
func (k TokenKind) IsTrivia() bool {
	return k == TokenWhitespace || k == TokenComment || k == TokenCommentUnterminated
}

// Token은 원문 조각 하나다.
//
// 계약: Tokenize가 낸 토큰의 Text를 순서대로 이어 붙이면 **원본 소스와 바이트 단위로
// 같다**. 트리비아를 버리지 않는 것이 이 계약의 전부이며, format과 lsp가 여기에 올라탄다
// (docs/decisions/0001-parser-trivia.md).
type Token struct {
	Kind TokenKind
	// Text는 이 토큰이 덮는 원문 그대로다 (인용 부호·주석 구분자 포함).
	Text string
	Span Span
}

// String은 테스트 실패 메시지용 표현이다.
func (t Token) String() string {
	return t.Kind.String() + "(" + t.Text + ")@" + t.Span.String()
}

// Tokens는 토큰 열이다.
type Tokens []Token

// Source는 토큰 열을 원문으로 되돌린다. 무손실 계약의 실행 가능한 형태다.
func (ts Tokens) Source() string {
	n := 0
	for _, t := range ts {
		n += len(t.Text)
	}
	b := make([]byte, 0, n)
	for _, t := range ts {
		b = append(b, t.Text...)
	}
	return string(b)
}

// WithoutTrivia는 트리비아와 EOF를 뺀 의미 토큰만 돌려준다. 테스트와 파서 진단용이다.
func (ts Tokens) WithoutTrivia() Tokens {
	out := make(Tokens, 0, len(ts))
	for _, t := range ts {
		if t.Kind.IsTrivia() || t.Kind == TokenEOF {
			continue
		}
		out = append(out, t)
	}
	return out
}
