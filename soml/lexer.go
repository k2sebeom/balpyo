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
	lx.pos = advance(start, text)
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

// cdataOpen·cdataClose는 CDATA 섹션의 구분자다. 섹션 안의 내용은 마크업이
// 아니라 문자 그대로이고, 그것이 XML 예제를 원문대로 슬라이드에 띄우는 유일한 길이다.
const (
	cdataOpen  = "<![CDATA["
	cdataClose = "]]>"
)

// startsMarkup은 `<`로 시작하는 이 자리가 마크업인지 보고한다.
//
// 마크업은 네 가지뿐이다 — 여는 태그(`<` + 글자), 닫는 태그(`</`), 주석(`<!--`),
// CDATA(`<![CDATA[`). **나머지 `<`는 내용이다**: `a < b`·`x <= y`·`1<2`가 그냥 쓰인다.
// 이 규칙이 오탈자를 잃지 않는 근거는 태그 이름이 항상 글자로 시작한다는 것이다
// (isNameStart) — 리터럴로 읽히는 `<`는 태그일 수 없는 `<`뿐이므로 `<txt>`의 오타는
// 여전히 "알 수 없는 태그"로 잡힌다.
func startsMarkup(rest string) bool {
	if len(rest) < 2 || rest[0] != '<' {
		return false
	}
	switch {
	case isNameStart(rest[1]), rest[1] == '/':
		return true
	case rest[1] == '!':
		return strings.HasPrefix(rest, "<!--") || strings.HasPrefix(rest, cdataOpen)
	}
	return false
}

// lexContent는 태그 밖(문자 데이터) 위치에서 토큰 하나를 읽는다.
func (lx *lexer) lexContent() Token {
	rest := lx.rest()
	if !startsMarkup(rest) {
		// 다음 **마크업** `<`까지가 텍스트다. 마크업이 아닌 `<`는 내용이므로 텍스트 토큰을
		// 끊지 않는다 — 끊으면 같은 문장이 토큰 여러 개로 갈라져 접기·참조 해석이 조각마다
		// 따로 일어난다.
		return lx.take(TokenText, textLen(rest))
	}
	switch {
	case strings.HasPrefix(rest, "<!--"):
		if i := strings.Index(rest[len("<!--"):], "-->"); i >= 0 {
			return lx.take(TokenComment, len("<!--")+i+len("-->"))
		}
		return lx.take(TokenCommentUnterminated, len(rest))
	case strings.HasPrefix(rest, cdataOpen):
		if i := strings.Index(rest[len(cdataOpen):], cdataClose); i >= 0 {
			return lx.take(TokenCData, len(cdataOpen)+i+len(cdataClose))
		}
		return lx.take(TokenCDataUnterminated, len(rest))
	case strings.HasPrefix(rest, "</"):
		return lx.take(TokenTagStartClose, 2)
	default:
		return lx.take(TokenTagStart, 1)
	}
}

// textLen은 rest의 앞에서 문자 데이터가 차지하는 바이트 수다 — 다음 마크업 `<`까지다.
// 0바이트를 돌려주지 않는다 (렉서는 항상 전진한다): 첫 바이트가 마크업 아닌 `<`면 그것을
// 지나서 센다.
func textLen(rest string) int {
	for i := 1; i < len(rest); i++ {
		if rest[i] == '<' && startsMarkup(rest[i:]) {
			return i
		}
	}
	return len(rest)
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
		if !startsMarkup(rest) {
			// 태그 **안**에서 마크업이 아닌 `<`는 쓸 수 없는 문자다 (속성 이름도 값도
			// 아니다). lexContent로 넘기면 텍스트 토큰이 뒤따르는 태그까지 삼켜 진단이
			// 엉뚱한 곳을 가리킨다 — 태그 밖의 리터럴 `<` 규칙(startsMarkup)과 갈라지는
			// 유일한 자리다.
			return lx.takeRune(TokenInvalid)
		}
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
