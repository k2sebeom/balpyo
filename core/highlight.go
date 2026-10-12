package core

// 코드 하이라이팅의 값 층이다.
//
// 이 파일이 하는 일은 둘이다. **탭을 편다**(칼럼을 세어 스페이스로 — 폭은 테마의
// `code.tabSize`), 그리고 `lang`이 붙은 `<pre>`의 문자 데이터를 **합성 인라인 런**으로
// 컴파일한다. 두 렌더러는 손대지 않는다: 웹은 인라인을 `<span style>`로, PPTX는 `a:r` +
// `rPr`로 내는 경로를 이미 갖고 있고(확정 스타일의 상속-복사), 하이라이팅은 그
// 경로에 **색이 확정된 조각**을 넣는 일이다. 색은 테마가 정하므로(components.json의
// `code.tokens`) 합성 런은 색 토큰 이름을 가진 `color` 속성을 달고 나가며, 그 이름의 해석은
// 저자가 쓴 `<span color>`와 **같은 함수**를 지난다 (resolveInlines) — 두 벌이 되면 진단과
// 토큰 참조 보존이 갈라진다.
//
// 토크나이저는 Chroma다. 우리 것은 `chroma.TokenType` → **우리가 닫은 종류 7종**으로의
// 축약뿐이고, Chroma의 스타일·포매터·HTML 출력은 쓰지 않는다 — 색의 주인이 둘이 되면
// 테마를 갈아도 코드 색만 남는다.

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/k2sebeom/balpyo/soml"
)

// codeClass는 코드 토큰 종류의 **닫힌** 어휘다. Chroma의 수백 종류를
// 여기로 축약하는 것이 우리 쪽 경계다: 종류가 늘면 테마마다 적을 표가 늘고, 적지 않은 테마의
// 코드가 종류마다 다르게 죽는다.
type codeClass string

const (
	classKeyword codeClass = "keyword"
	className    codeClass = "name"
	classAttr    codeClass = "attr"
	classString  codeClass = "string"
	classNumber  codeClass = "number"
	classComment codeClass = "comment"
	classPunct   codeClass = "punct"
	// classPlain은 **표에 적을 수 없는** 이름이다 — 색이 없는 자리이므로 테마가 정할 것이
	// 없고, `<pre>`의 확정 색을 그대로 물려받는다 (공백·연산자·평범한 식별자·Error).
	classPlain codeClass = "plain"
)

// codeClasses는 테마가 색을 적을 수 있는 종류 전부다 (순서 = 진단과 문서의 순서).
var codeClasses = []codeClass{
	classKeyword, className, classAttr, classString, classNumber, classComment, classPunct,
}

// codeLangs는 `lang` 값 → Chroma 렉서의 **닫힌** 표다 (순서 = 진단의 순서).
//
// 목록 밖은 에러다 (variant처럼 폴백하지 않는다): variant 어휘는 테마의 사유물이라
// 다른 테마로 만든 덱도 열려야 하지만, `lang`은 엔진 어휘이므로 목록 밖 이름은 오타이거나
// 아직 없는 언어다. 조용히 하이라이팅을 끄면 저자는 자기 색이 왜 안 나오는지 알 수 없다.
//
// `soml`이 `xml`의 별칭인 이유: 이 저장소의 덱이 자기 문법을 띄우는 것이 가장 흔한 코드
// 블록이고, 저자가 먼저 쓸 이름이 `soml`이다. 별칭을 언어로 세지 않는 것이 값이다 — SOML의
// 토크나이저를 따로 들이면 어휘가 자랄 때마다 렉서를 고쳐야 한다.
var codeLangs = []struct {
	name  string
	lexer string
}{
	{name: "go", lexer: "go"},
	{name: "xml", lexer: "xml"},
	{name: "soml", lexer: "xml"},
	{name: "json", lexer: "json"},
	{name: "bash", lexer: "bash"},
}

// codeLangNames는 진단에 싣는 쓸 수 있는 값 목록이다.
func codeLangNames() []string {
	names := make([]string, 0, len(codeLangs))
	for _, lang := range codeLangs {
		names = append(names, lang.name)
	}
	return names
}

// codeRun은 색이 하나인 글자 조각이다 — 합성 런 하나가 된다. color가 비어 있으면 plain이다.
type codeRun struct {
	color string
	text  string
}

// resolveCode는 `<pre>`의 내용을 코드로 해석한다 (resolve.go의 텍스트 역할 갈래에서 불린다 —
// 블록의 확정 스타일이 채워진 **뒤**, 인라인 런이 해석되기 **전**이다: 합성 런이 그 확정
// 스타일을 부모로 상속해야 색만 다른 조각이 된다).
//
// 탭은 `lang`과 무관하게 편다 — 탭은 하이라이팅의 문제가 아니라 글자의 문제다. CSS
// `tab-size`와 OOXML의 절대 탭 정지점(`a:defTabSz`)은 같은 규칙이 아니어서, 탭을 그대로
// 흘리면 두 산출물의 줄이 갈라진다.
func (r *resolver) resolveCode(elem *soml.Element) {
	r.expandCodeTabs(elem)
	attr, ok := elem.Attr(soml.LangAttr)
	if !ok {
		return
	}
	if !r.hasValue(attr) {
		return
	}
	lexerName, ok := r.codeLexer(attr)
	if !ok {
		return
	}
	r.compileCodeRuns(elem, attr, lexerName)
}

// codeLexer는 `lang` 값을 렉서 이름으로 바꾼다. 목록 밖은 에러다.
func (r *resolver) codeLexer(attr soml.Attr) (string, bool) {
	for _, lang := range codeLangs {
		if lang.name == attr.Value {
			return lang.lexer, true
		}
	}
	langs := strings.Join(codeLangNames(), ", ")
	r.errorf(attr.ValueSpan, "unknown %s %q%s (available values: %s)",
		soml.LangAttr, attr.Value, didYouMean(attr.Value, langs), langs)
	return "", false
}

// expandCodeTabs는 `<pre>`의 탭을 칼럼을 세어 스페이스로 편다.
//
// 정지점은 "다음 `tabSize` 배수까지"지 "N칸"이 아니다 — gofmt 출력을 그대로 붙여 넣었을 때
// 들여쓰기 계단이 고르게 보이는 규칙이 그것이다 (글자 하나 뒤의 탭은 한 칸만 먹는다).
// 칼럼 상태는 텍스트 노드를 **가로질러** 이어진다: `<pre>`의 자식이 여럿일 수 있고(문자
// 참조·CDATA가 섞인 원문), 노드마다 0에서 다시 세면 같은 줄의 탭이 다르게 펴진다.
//
// 탭이 없는 노드는 문자열을 새로 만들지 않는다 — 탭을 쓰지 않은 덱의 산출물은 이 패스 때문에
// 한 바이트도 움직이지 않아야 한다 (골든이 그 증인이다).
func (r *resolver) expandCodeTabs(elem *soml.Element) {
	size := r.comps.code.tabSize
	if size < 1 {
		// 테마 로더가 이미 막는다 (codeComponentsFromFile) — 손으로 만든 components로
		// 해석하는 테스트에서도 0으로 나누지 않게 바닥을 둔다.
		size = defaultCodeTabSize
	}
	col := 0
	for _, child := range elem.Children {
		text, ok := child.(*soml.Text)
		if !ok {
			// `<pre>`의 콘텐츠 모델은 문자 데이터뿐이다 (modelPre) — 요소가 있다면 이 패스가
			// 두 번 돈 것이므로 칼럼만 흘려보낸다.
			continue
		}
		text.Value, col = expandTabs(text.Value, size, col)
	}
}

// expandTabs는 문자열 하나의 탭을 펴고 **끝 칼럼**을 돌려준다.
//
// 칼럼은 글자 수로 센다 — 동아시아 문자의 폭 2를 반영하지 않는다. 폭을 재는 규칙을 여기
// 두면 같은 계단이 폰트마다 달라지고(값 층은 폰트를 모른다), 두 렌더러가 받는 글자는 어차피
// 여기서 만든 **한 문자열**이라 두 산출물이 갈라질 자리는 없다.
func expandTabs(s string, size, col int) (string, int) {
	if !strings.ContainsRune(s, '\t') {
		return s, columnAfter(s, col)
	}
	var b strings.Builder
	b.Grow(len(s))
	for _, ch := range s {
		switch ch {
		case '\t':
			n := size - col%size
			b.WriteString(strings.Repeat(" ", n))
			col += n
		case '\n':
			b.WriteRune(ch)
			col = 0
		case '\r':
			// CR은 개행의 일부다 (CRLF 원문) — 칼럼을 늘리지 않는다.
			b.WriteRune(ch)
		default:
			b.WriteRune(ch)
			col++
		}
	}
	return b.String(), col
}

// columnAfter는 문자열을 지난 뒤의 칼럼이다.
func columnAfter(s string, col int) int {
	if i := strings.LastIndexByte(s, '\n'); i >= 0 {
		return utf8.RuneCountInString(s[i+1:])
	}
	return col + utf8.RuneCountInString(s)
}

// compileCodeRuns는 `<pre>`의 문자 데이터를 토큰화해 합성 런으로 바꾼다.
func (r *resolver) compileCodeRuns(elem *soml.Element, attr soml.Attr, lexerName string) {
	source := elem.TextContent()
	if source == "" {
		return
	}
	lexer := lexers.Get(lexerName)
	if lexer == nil {
		// 내장 렉서가 사라졌다 — 어휘 표와 Chroma 버전이 갈라진 것이므로 조용히 넘기지 않는다.
		r.errorf(attr.ValueSpan, "no tokenizer found for %s=%q", soml.LangAttr, attr.Value)
		return
	}
	// EnsureLF를 끈다: 켜면 CRLF가 LF로 바뀌어 **토큰 값을 이어 붙인 것이 원문과 달라진다**.
	// 개행 정규화는 파서와 렌더러가 이미 각자의 규칙으로 하고 있고(soml의 pre 트리밍,
	// PPTX의 a:br), 토크나이저가 글자를 바꾸면 그 둘이 서로를 못 믿게 된다.
	iter, err := lexer.Tokenise(&chroma.TokeniseOptions{State: "root"}, source)
	if err != nil {
		r.errorf(attr.ValueSpan, "tokenizing %s=%q failed: %v", soml.LangAttr, attr.Value, err)
		return
	}
	runs := coalesceRuns(iter.Tokens(), r.comps.code.colors)

	// 왕복 검사다 — 토큰 값을 이어 붙인 것이 원문과 같아야 한다. 하이라이팅이 글자를 **한 개**
	// 잃는 실패는 산출물을 눈으로 볼 때까지 안 보이고, 그때는 어느 층이 먹었는지 알 수 없다.
	var joined strings.Builder
	for _, run := range runs {
		joined.WriteString(run.text)
	}
	if joined.String() != source {
		r.errorf(attr.ValueSpan, "tokenizing %s=%q did not preserve the source text — rendering without highlighting",
			soml.LangAttr, attr.Value)
		return
	}
	elem.Children = codeNodes(runs, elem)
}

// coalesceRuns는 토큰 스트림을 **색이 바뀌는 자리마다** 끊은 런 목록으로 만든다.
//
// 합치지 않으면 런이 글자 수만큼 는다 — Chroma는 공백마다 토큰을 끊으므로 Go 다섯 줄이
// 서른 토큰을 넘고, 그대로 내면 `a:r`가 서른 개가 된다 (편집 가능한 PPTX의 값은 사람이
// 고칠 수 있는 모양에 있다). 경계를 **종류**가 아니라 **색**으로 잡는 이유는 표가 부분
// 패치라는 것이다: 테마가 적지 않은 종류(minimal의 `punct`)는 plain과 같은 색이므로 끊을
// 이유가 없다.
func coalesceRuns(tokens []chroma.Token, colors map[codeClass]soml.ColorToken) []codeRun {
	var runs []codeRun
	for _, token := range tokens {
		if token.Value == "" {
			continue
		}
		color := ""
		if tok, ok := colors[classOf(token.Type)]; ok {
			color = tok.Name
		}
		if n := len(runs); n > 0 && runs[n-1].color == color {
			runs[n-1].text += token.Value
			continue
		}
		runs = append(runs, codeRun{color: color, text: token.Value})
	}
	return runs
}

// classOf는 Chroma 종류를 우리 7종으로 축약한다.
//
// 카테고리로 받는 것이 이 표의 요령이다: Chroma의 종류는 1000 단위 카테고리·100 단위
// 하위 카테고리로 번호가 매겨져 있어(`KeywordDeclaration`은 `Keyword`의 하위) 언어가 늘어도
// 축약이 언어를 모른 채 성립한다. 예외 세 줄이 있다 — `NameAttribute`는 XML·JSON에서
// 이름과 다른 자리이므로 따로 칠하고, `Name`·`NameOther`는 **평범한 식별자**라 색이 없다
// (전부 칠하면 코드가 색종이가 된다).
func classOf(t chroma.TokenType) codeClass {
	switch {
	case t.InCategory(chroma.Comment):
		return classComment
	case t.InCategory(chroma.Keyword):
		return classKeyword
	case t == chroma.NameAttribute:
		return classAttr
	case t == chroma.Name || t == chroma.NameOther:
		return classPlain
	case t.InCategory(chroma.Name):
		return className
	case t.InSubCategory(chroma.LiteralString):
		return classString
	case t.InSubCategory(chroma.LiteralNumber):
		return classNumber
	case t == chroma.Punctuation:
		return classPunct
	default:
		// 공백·연산자·Generic·Error·그 밖의 리터럴. 색이 없다 — `<pre>`의 확정 색을 물려받는다.
		return classPlain
	}
}

// codeNodes는 런 목록을 `<pre>`의 새 자식으로 만든다.
//
// **plain은 요소가 되지 않는다** — 색이 없으므로 감쌀 이유가 없고, 첫 자식이 문자 데이터로
// 남아야 웹의 선두 개행 보호(render/web.firstText)가 눈이 멀지 않는다. 색 있는 조각은
// `color` 속성을 가진 합성 `<span>`이고, 그 속성은 저자가 쓴 `<span color>`와 같은 해석
// 함수를 지난다 (resolveInlines).
//
// Span은 전부 `<pre>`의 것이다 — 합성 노드의 진단은 저자가 고칠 수 있는 자리, 즉 `<pre>`를
// 가리켜야 한다 (토큰의 위치는 원문 줄·칼럼이 아니라 코드 안의 오프셋이다).
func codeNodes(runs []codeRun, pre *soml.Element) []soml.Node {
	nodes := make([]soml.Node, 0, len(runs))
	for _, run := range runs {
		if run.color == "" {
			nodes = append(nodes, &soml.Text{Value: run.text, Span: pre.Span})
			continue
		}
		nodes = append(nodes, &soml.Element{
			Name:     "span",
			Span:     pre.Span,
			NameSpan: pre.NameSpan,
			Attrs: []soml.Attr{{
				Name:      "color",
				NameSpan:  pre.NameSpan,
				Value:     run.color,
				ValueSpan: pre.NameSpan,
				HasValue:  true,
			}},
			Children: []soml.Node{&soml.Text{Value: run.text, Span: pre.Span}},
		})
	}
	return nodes
}

// codeComponents는 테마의 `code` 절이다 (변환된 내부 표).
type codeComponents struct {
	// tabSize는 탭 정지점의 간격이다 (칼럼 수).
	tabSize int
	// colors는 종류 → 색 토큰이다. **부분 패치**이므로 적지 않은 종류는 plain이 된다.
	colors map[codeClass]soml.ColorToken
}

// defaultCodeTabSize는 테마가 없는 경로(손으로 만든 components)의 바닥값이다. 테마는 반드시
// 자기 값을 적어야 한다 — 로더가 1 미만을 막는다.
const defaultCodeTabSize = 2

// codeComponentsFromFile은 `code` 절을 검사해 내부 표로 바꾼다.
//
// 테마는 **저자가 고칠 수 없는 자리**이므로 덱을 렌더하기 전에 멈춘다 (크롬 선언·슬롯 대조와
// 같은 태도): 모르는 종류 이름은 오타이고, 리터럴 색은 토큰 참조를 끊어 PowerPoint의 테마
// 색 편집이 코드 블록만 두고 가게 만든다.
func codeComponentsFromFile(file codeComponentsFile, theme soml.Theme) (codeComponents, error) {
	code := codeComponents{tabSize: file.TabSize, colors: map[codeClass]soml.ColorToken{}}
	if file.TabSize < 1 {
		return code, declAt(fmt.Errorf("tabSize must be at least 1 (the value layer expands tabs by counting columns)"), atKey("tabSize"))
	}
	for _, token := range file.Tokens {
		entry := []declStep{atKey("tokens"), atEntry("class", token.Class)}
		class := codeClass(token.Class)
		if !knownCodeClass(class) {
			return code, declAt(fmt.Errorf("unknown code token class %q (available types: %s)",
				token.Class, strings.Join(codeClassNames(), ", ")), entry...)
		}
		if _, dup := code.colors[class]; dup {
			return code, declAt(fmt.Errorf("code token class %q is listed twice", token.Class), entry...)
		}
		if !isColorTokenName(token.Color) {
			return code, declAt(fmt.Errorf("the color of code token %q must be a color token name (got %q) — a literal color breaks the reference",
				token.Class, token.Color), append(entry, atKey("color"))...)
		}
		tok, known := theme.Color(token.Color)
		if !known {
			return code, declAt(fmt.Errorf("code token %q: color token %q is not in the theme (available names: %s)",
				token.Class, token.Color, strings.Join(theme.ColorNames(), ", ")), append(entry, atKey("color"))...)
		}
		code.colors[class] = tok
	}
	return code, nil
}

func knownCodeClass(class codeClass) bool {
	for _, known := range codeClasses {
		if known == class {
			return true
		}
	}
	return false
}

func codeClassNames() []string {
	names := make([]string, 0, len(codeClasses))
	for _, class := range codeClasses {
		names = append(names, string(class))
	}
	return names
}

// isColorTokenName은 값이 색 **토큰 이름**인지 본다 — 리터럴 hex(`#DC2626`)와 색 함수
// (`shade(accent1, 40%)`)를 여기서 거른다. 토큰 이름 어휘는 소문자·숫자뿐이다 (tokens.json).
func isColorTokenName(value string) bool {
	if value == "" {
		return false
	}
	for _, ch := range value {
		switch {
		case ch >= 'a' && ch <= 'z', ch >= '0' && ch <= '9', ch == '-':
		default:
			return false
		}
	}
	return true
}
