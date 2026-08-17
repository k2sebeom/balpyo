package soml

import (
	"strings"
	"unicode/utf8"
)

// Tokenize는 SOML 소스를 토큰 열로 자른다.
//
// 렉서는 **무손실**이다: 결과 토큰의 Text를 이어 붙이면 src와 같다 (Tokens.Source).
// 실패하지 않는다 — 문법에 맞지 않는 입력도 TokenInvalid나 TokenUnterminated*로 덮어
// 통과시키고, 진단은 파서 층에서 낸다. 입력 중인 문서가 항상 깨져 있는 lsp를 전제한
// 설계다.
func Tokenize(src string) Tokens {
	lx := &lexer{src: src, pos: StartPos()}
	// 토큰 수는 대략 소스 길이의 1/4 수준이다. 재할당을 줄이는 초기 용량.
	toks := make(Tokens, 0, len(src)/4+2)

	inTag := false
	for !lx.atEnd() {
		var tok Token
		if inTag {
			tok = lx.lexInTag()
			if tok.Kind == TokenTagEnd || tok.Kind == TokenTagEndSelfClose {
				inTag = false
			}
		} else {
			tok = lx.lexContent()
			if tok.Kind == TokenTagStart || tok.Kind == TokenTagStartClose {
				inTag = true
			}
		}
		toks = append(toks, tok)
	}
	return append(toks, Token{Kind: TokenEOF, Span: SpanOf(lx.pos, lx.pos)})
}

// lexer는 소스를 훑는 커서다.
type lexer struct {
	src string
	pos Pos
}

func (lx *lexer) atEnd() bool { return lx.pos.Offset >= len(lx.src) }

func (lx *lexer) rest() string { return lx.src[lx.pos.Offset:] }

// take는 n바이트를 소비하고 그 구간의 토큰을 만든다.
func (lx *lexer) take(kind TokenKind, n int) Token {
	start := lx.pos
	text := lx.src[start.Offset : start.Offset+n]
	for _, r := range text {
		if r == '\n' {
			lx.pos.Line++
			lx.pos.Col = 1
		} else {
			lx.pos.Col++
		}
	}
	lx.pos.Offset += n
	return Token{Kind: kind, Text: text, Span: SpanOf(start, lx.pos)}
}

// takeRune은 룬 하나를 소비한다. Invalid 토큰이 UTF-8을 쪼개지 않게 하려고 쓴다.
func (lx *lexer) takeRune(kind TokenKind) Token {
	_, size := utf8.DecodeRuneInString(lx.rest())
	if size == 0 {
		size = 1
	}
	return lx.take(kind, size)
}

// lexContent는 태그 밖(문자 데이터) 위치에서 토큰 하나를 읽는다.
func (lx *lexer) lexContent() Token {
	rest := lx.rest()
	if rest[0] != '<' {
		// 다음 `<`까지가 텍스트다. 없으면 소스 끝까지.
		if i := strings.IndexByte(rest, '<'); i > 0 {
			return lx.take(TokenText, i)
		}
		return lx.take(TokenText, len(rest))
	}
	switch {
	case strings.HasPrefix(rest, "<!--"):
		if i := strings.Index(rest[len("<!--"):], "-->"); i >= 0 {
			return lx.take(TokenComment, len("<!--")+i+len("-->"))
		}
		return lx.take(TokenCommentUnterminated, len(rest))
	case strings.HasPrefix(rest, "</"):
		return lx.take(TokenTagStartClose, 2)
	default:
		return lx.take(TokenTagStart, 1)
	}
}

// lexInTag는 태그 안(`<`와 `>` 사이) 위치에서 토큰 하나를 읽는다.
func (lx *lexer) lexInTag() Token {
	rest := lx.rest()
	switch c := rest[0]; {
	case isTagSpace(c):
		return lx.take(TokenWhitespace, countLeading(rest, isTagSpace))
	case c == '>':
		return lx.take(TokenTagEnd, 1)
	case c == '/':
		if strings.HasPrefix(rest, "/>") {
			return lx.take(TokenTagEndSelfClose, 2)
		}
		return lx.takeRune(TokenInvalid)
	case c == '=':
		return lx.take(TokenEquals, 1)
	case c == '"' || c == '\'':
		return lx.lexString(c)
	case c == '<':
		// 태그 안의 `<`는 앞 태그가 닫히지 않았다는 뜻이다. `<`를 새 태그의 시작으로
		// 넘겨 파서가 "닫히지 않은 태그" 하나만 진단하고 계속 읽게 한다 — 이 문자를
		// Invalid로 삼키면 뒤따르는 문서 전체가 속성 자리로 오해된다.
		return lx.lexContent()
	case isNameStart(c):
		return lx.take(TokenName, countLeading(rest, isNameByte))
	default:
		return lx.takeRune(TokenInvalid)
	}
}

// lexString은 인용된 속성 값을 읽는다. 인용 부호는 토큰 Text에 남는다.
//
// 닫는 부호를 찾지 못하면 **줄 끝에서 멈춘다** — 인용 부호 하나를 빠뜨린 문서가 파일
// 나머지 전체를 문자열로 삼켜 진단이 엉뚱한 곳을 가리키는 일을 막는다.
func (lx *lexer) lexString(quote byte) Token {
	rest := lx.rest()
	for i := 1; i < len(rest); i++ {
		switch rest[i] {
		case quote:
			return lx.take(TokenString, i+1)
		case '\n':
			return lx.take(TokenStringUnterminated, i)
		}
	}
	return lx.take(TokenStringUnterminated, len(rest))
}

// isTagSpace는 태그 내부에서 구분자로 쓰이는 공백이다.
func isTagSpace(c byte) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == '\r'
}

// isNameStart는 이름의 첫 바이트로 허용되는 문자다 — 영문자만.
func isNameStart(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

// isNameByte는 이름의 두 번째 이후 바이트로 허용되는 문자다.
// 하이픈은 합성 컴포넌트 이름(`dim-card`)에, 숫자는 `h1`~`h3`에 쓰인다.
func isNameByte(c byte) bool {
	return isNameStart(c) || (c >= '0' && c <= '9') || c == '-' || c == '_'
}

// countLeading은 pred를 만족하는 선행 바이트 수를 센다 (최소 1).
func countLeading(s string, pred func(byte) bool) int {
	n := 0
	for n < len(s) && pred(s[n]) {
		n++
	}
	return n
}
