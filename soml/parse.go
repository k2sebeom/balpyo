package soml

import (
	"sort"
	"strings"
)

// RootElement는 `slides/` 아래 파일의 루트 요소 이름이다. 파일의 역할은 확장자가 아니라
// 루트 요소가 정한다 (PRD §13-2) — `components/`의 `<define>`은 Iteration 11에서 열린다.
const RootElement = "slide"

// noteElement는 발표자 노트 태그다. 콘텐츠가 아니라 슬라이드 메타데이터이므로 콘텐츠
// 트리에서 빠지고 SlideDoc.Notes로 간다.
const noteElement = "note"

// elementSpecs는 **닫힌 어휘**다 (PRD §7.2). 여기 없는 태그 이름은 무시되지 않고 에러가 된다.
//
// 어휘는 iteration을 따라 열린다 — 텍스트 서브셋(subset.go의 카탈로그)은 Iteration 2,
// 레이아웃 태그는 Iteration 3. 어휘를 미리 열어 두면 "파싱은 되는데 렌더는 안 되는" 상태가
// 생기고, 그것이 닫힌 어휘로 막으려던 바로 그 조용한 실패다. 렌더 매핑이 아직 없는 태그는
// 렌더러가 명시적으로 멈춘다 (render/web.writeNode, render/pptx.writeShapeNode).
var elementSpecs = buildElementSpecs()

// buildElementSpecs는 문법 표를 만든다. 서브셋 태그의 규칙은 카탈로그(subset.go)의 종류에서
// 파생된다 — 태그를 카탈로그에 추가하면 문법이 따라오고, 두 표가 어긋날 수 없다.
func buildElementSpecs() map[string]elementSpec {
	specs := map[string]elementSpec{
		RootElement: {
			attrs: []string{"id", "skip"},
			model: modelElements,
			// PRD §7.7의 `reveal` 속성은 열지 않는다 — 웹 런타임의 reveal은 Phase 2이고,
			// 렌더가 없는 속성을 어휘에 열면 닫힌 어휘가 막으려던 조용한 무시가 된다.
			children: slideChildren(),
		},
		TextElement: {
			attrs: []string{"role", "variant", "size", "color", "align"},
			model: modelInline,
		},
		noteElement: {model: modelText},
	}
	for _, name := range blockElements {
		model := modelInline
		if name == "pre" {
			// v1의 pre는 문자 데이터만 담는다 — <pre><code> 관용은 코드 하이라이팅(P1)이
			// 인라인 서식과 어떻게 만나는지 정해질 때 연다. 지금 열면 그 결정 전에
			// "code가 pre 안에서 무엇을 뜻하는가"가 렌더러마다 임의로 정해진다.
			model = modelPre
		}
		// 블록의 속성은 variant 하나다. 블록은 역할의 축약(`<p>` ≡ `<text role="body">`)이라
		// 문단 단위 외관 선택인 variant가 자연스럽게 올 자리다. variant를 열지 **않는** 곳의
		// 근거: 인라인 태그의 외관은 런 규칙(Iteration 2 항목 04)이 소유하고, 리스트
		// 컨테이너와 <li>의 위계는 역할이 소유한다 — 항목마다 다른 외관 번들이 붙으면
		// 리스트의 타이포그래피가 리스트이기를 멈춘다.
		specs[name] = elementSpec{attrs: []string{"variant"}, model: model}
	}
	for _, name := range listContainerElements {
		specs[name] = elementSpec{model: modelElements, children: []string{ListItemElement}}
	}
	specs[ListItemElement] = elementSpec{model: modelInline}
	for _, name := range inlineElements {
		spec := elementSpec{model: modelInline}
		switch name {
		case "a":
			// 속성은 href 하나다. href 없는 <a>의 진단은 core(값 층)의 몫이다 — 파서는
			// 이름 allowlist만 보고, "속성이 있어야 한다"는 값의 요구이기 때문이다.
			spec.attrs = []string{"href"}
		case "span":
			// color·weight만 연다. size를 막는 이유: 런 하나가 커지면 웹의 줄 박스 높이가
			// 변하는데 PPTX 줄간격은 문단 절대값(spcPts)이라 두 타깃의 글줄이 갈라진다 —
			// core가 size="%"를 막은 것과 같은 종류의 근거다 (core/resolve.go).
			spec.attrs = []string{"color", "weight"}
		case BreakElement:
			spec.model = modelVoid
		}
		specs[name] = spec
	}
	return specs
}

// slideChildren은 <slide>의 자식 어휘다. 순서가 진단 메시지의 순서다.
func slideChildren() []string {
	children := []string{TextElement, noteElement}
	children = append(children, blockElements...)
	return append(children, listContainerElements...)
}

// elementSpec은 태그 하나의 문법 규칙이다. 값 어휘는 여기 없다 — 값은 core가 본다.
type elementSpec struct {
	// attrs는 허용 속성 이름이다. 진단 메시지에 그대로 실리므로 순서가 곧 문서 순서다.
	attrs []string
	model contentModel
	// children은 modelElements일 때 허용되는 자식 태그다.
	children []string
}

// contentModel은 요소가 담을 수 있는 것이다.
type contentModel int

const (
	// modelElements는 요소만 담는다. 공백 아닌 문자 데이터는 에러다 — 슬라이드에 그냥 쓴
	// 글자가 어떤 역할의 텍스트인지는 정할 방법이 없고, 닫힌 어휘는 그런 추측을 하지 않는다.
	modelElements contentModel = iota
	// modelText는 문자 데이터만 담는다 (<note>).
	modelText
	// modelInline은 문자 데이터와 인라인 태그를 담는다 (<text>·블록·<li>).
	modelInline
	// modelPre는 문자 데이터만 담되 **공백을 원문 그대로 보존**한다 (<pre>).
	// 여는 태그 직후·닫는 태그 직전의 개행 하나만 벗긴다 — HTML의 관례로, 저자가 <pre>
	// 다음 줄부터 쓰게 하기 위한 것이지 내용이 아니다 (trimPreEdges).
	modelPre
	// modelVoid는 아무것도 담지 않는다 (<br>). 자기 닫힘(<br/>)만 허용된다.
	modelVoid
)

// SlideDoc은 `.slide` 파일 하나의 파싱 결과다.
//
// Slide(덱 모델)와 나누는 이유: Index·Chapter·파일명 유도 ID는 파일 하나를 봐서는 알 수 없고,
// Skip은 덱에 들어가기 전에 소비되어 사라진다. 파일 안에서 알 수 있는 것만 여기 담고,
// 덱 조립은 파이프라인이 한다.
type SlideDoc struct {
	// File은 진단에 찍히는 경로다 (프로젝트 루트 기준 상대 경로).
	File string
	// Root는 루트 `<slide>` 요소다. 루트를 찾지 못하면 nil이다 (진단이 남는다).
	Root *Element
	// ID는 `<slide id>`의 값이다. 속성이 없으면 빈 문자열 — 파일명 유도는 호출자 몫이다
	// (SlideFile.DerivedID).
	ID string
	// Skip은 `<slide skip>`이다. true면 렌더에서 제외된다 (PRD §7.1). 제외 판정은 파서가
	// 노출하고 덱에서 빼는 것은 파이프라인이 한다 — Index가 남은 슬라이드 기준이어야 하므로.
	Skip bool
	// Notes는 `<note>`의 텍스트다.
	Notes string
	// Content는 `<slide>`의 콘텐츠 자식이다 (`<note>` 제외).
	Content []Node
	// Tokens는 이 파일의 원문 토큰 열이다 — 무손실 계약(Tokens.Source)이 여기까지 유지된다.
	Tokens Tokens
}

// Parse는 `.slide` 파일 하나를 파싱한다. file은 진단에 찍히는 경로다.
//
// 엄격 파서다: 닫힌 어휘 밖의 태그·속성 이름, 짝이 맞지 않는 태그, 인용 부호 미닫힘은 모두
// **line:col을 가진 진단**이 된다. 대신 첫 에러에서 멈추지 않고 가능한 만큼 계속 읽어 한
// 파일의 진단을 한 번에 모아 낸다 — 빌드를 돌릴 때마다 오타 하나씩 알려주는 파서는 저자의
// 시간을 오타 수만큼 쓴다. 편집 중이라 늘 깨져 있는 문서를 다루는 lsp의 요구와도 같다.
//
// 공백 접기: 문자 데이터의 ASCII 공백(스페이스·탭·CR·LF) 런은 한 칸으로 접고, 요소 양 끝의
// 공백은 잘라낸다 (HTML `white-space: normal` 관례). 이 접기를 **파서가** 하는 이유는 타깃
// 때문이다 — 웹은 브라우저가 접고 PPTX는 접는 개념이 아예 없어서, 렌더러에 맡기면 같은 원문이
// 타깃마다 다른 문자열이 된다. U+00A0 같은 비ASCII 공백은 저자가 의도해 넣은 문자이므로
// 건드리지 않는다.
func Parse(file, src string) (*SlideDoc, Diagnostics) {
	p := &parser{file: file, toks: Tokenize(src)}
	doc := &SlideDoc{File: file, Tokens: p.toks}
	doc.Root = p.parseDocument()
	if doc.Root == nil {
		p.errorf(SpanOf(StartPos(), StartPos()), "루트 요소 <%s>가 없다", RootElement)
	} else {
		p.fillSlide(doc)
	}
	// 진단은 소스 순서로 낸다 — 저자가 위에서 아래로 고칠 수 있게. 닫히지 않은 태그처럼
	// 나중에 발견되는 진단이 앞을 가리키기 때문에 정렬이 필요하다.
	sort.SliceStable(p.diags, func(i, j int) bool {
		return p.diags[i].Span.Start.Offset < p.diags[j].Span.Start.Offset
	})
	return doc, p.diags
}

// parser는 토큰 열을 훑는 커서다. 트리비아는 건너뛰며 읽는다 (docs/decisions/0001).
type parser struct {
	file  string
	toks  Tokens
	i     int
	diags Diagnostics
}

func (p *parser) errorf(span Span, format string, args ...any) {
	p.diags = append(p.diags, Errorf(p.file, span, format, args...))
}

// peek은 트리비아를 건너뛴 다음 토큰이다. 닫히지 않은 주석은 지나면서 한 번 진단한다 —
// 렉서는 실패하지 않으므로 그 진단은 이 층의 몫이다.
func (p *parser) peek() Token {
	for p.i < len(p.toks) {
		tok := p.toks[p.i]
		if !tok.Kind.IsTrivia() {
			return tok
		}
		if tok.Kind == TokenCommentUnterminated {
			p.errorf(tok.Span, "주석이 `-->`로 닫히지 않았다")
		}
		p.i++
	}
	return Token{Kind: TokenEOF}
}

// take는 다음 의미 토큰을 소비한다.
func (p *parser) take() Token {
	tok := p.peek()
	if tok.Kind != TokenEOF {
		p.i++
	}
	return tok
}

// parseDocument는 파일 전체를 읽고 루트 요소를 돌려준다.
func (p *parser) parseDocument() *Element {
	var root *Element
	var stack []*Element
	for {
		switch tok := p.peek(); tok.Kind {
		case TokenEOF:
			// 남은 여는 태그는 전부 닫히지 않았다. 문서 순서(바깥부터)로 보고한다.
			for _, open := range stack {
				p.errorf(open.NameSpan, "<%s> 태그가 닫히지 않았다", open.Name)
			}
			return root

		case TokenText:
			p.i++
			p.charData(tok, parentOf(stack))

		case TokenTagStart:
			elem, stillOpen := p.parseStartTag()
			if elem == nil {
				continue
			}
			parent := parentOf(stack)
			p.checkPlacement(elem, stack, root)
			switch {
			case parent != nil:
				parent.Children = append(parent.Children, elem)
			case root == nil:
				root = elem
			}
			if stillOpen {
				if _, known := elementSpecs[elem.Name]; known && specOf(elem).model == modelVoid {
					// void 요소는 자기 닫힘이 표준이다. 진단하되 스택에는 올린다 — 짝이 되는
					// </br>가 "여는 태그가 없다"는 두 번째 진단을 만들지 않게.
					p.errorf(elem.NameSpan, "void 태그 <%s>은 내용을 가질 수 없다 (<%s/>로 쓴다)", elem.Name, elem.Name)
				}
				stack = append(stack, elem)
			} else {
				p.closeElement(elem)
			}

		case TokenTagStartClose:
			name, tagSpan := p.parseEndTag()
			if name == "" {
				continue
			}
			depth := -1
			for i := len(stack) - 1; i >= 0; i-- {
				if stack[i].Name == name {
					depth = i
					break
				}
			}
			if depth < 0 {
				p.errorf(tagSpan, "</%s>와 짝이 맞는 여는 태그가 없다", name)
				continue
			}
			// 안쪽에 열린 채 남은 요소는 닫히지 않았다. 각각 진단하고 함께 닫아 뒤따르는
			// 문서를 계속 읽는다.
			for i := len(stack) - 1; i > depth; i-- {
				p.errorf(stack[i].NameSpan, "<%s> 태그가 </%s>보다 먼저 닫혀야 한다", stack[i].Name, name)
				p.closeElement(stack[i])
			}
			stack[depth].Span.End = tagSpan.End
			p.closeElement(stack[depth])
			stack = stack[:depth]

		default:
			// 렉서는 문자 데이터 위치에서 다른 종류를 내지 않는다. 방어적으로 건너뛴다.
			p.i++
		}
	}
}

// charData는 문자 데이터 토큰을 부모의 콘텐츠 모델에 맞게 처리한다.
func (p *parser) charData(tok Token, parent *Element) {
	if parent != nil {
		spec, known := elementSpecs[parent.Name]
		if !known {
			// 미지의 태그는 이미 진단됐다. 그 안의 내용까지 진단해 하나의 오타를 여러 줄로
			// 부풀리지 않는다.
			return
		}
		switch spec.model {
		case modelText, modelInline:
			parent.Children = append(parent.Children, &Text{Value: collapseSpace(tok.Text), Span: tok.Span})
			return
		case modelPre:
			// pre 안에서는 접지 않는다 — 개행·연속 공백이 곧 내용이다. 양 끝 개행 규칙은
			// 닫힐 때 적용된다 (trimPreEdges).
			parent.Children = append(parent.Children, &Text{Value: tok.Text, Span: tok.Span})
			return
		case modelVoid:
			// void 위반은 여는 태그에서 이미 진단됐다 (parseDocument). 내용까지 진단해
			// 하나의 오용을 여러 줄로 부풀리지 않는다.
			return
		}
	}
	if strings.Trim(tok.Text, asciiSpace) == "" {
		// 요소만 담는 모델에서 들여쓰기·줄바꿈은 내용이 아니다.
		return
	}
	if parent == nil {
		p.errorf(dataSpan(tok), "요소 밖에 문자 데이터가 있다 (내용은 <%s> 안에 쓴다)", RootElement)
		return
	}
	if KindOf(parent.Name) == KindListContainer {
		p.errorf(dataSpan(tok), "<%s> 안에 문자 데이터를 직접 쓸 수 없다 (<%s>로 감싼다)", parent.Name, ListItemElement)
		return
	}
	p.errorf(dataSpan(tok), "<%s> 안에 문자 데이터를 직접 쓸 수 없다 (<text>로 감싼다)", parent.Name)
}

// parseStartTag는 여는 태그를 읽는다. stillOpen은 요소가 열린 채로 남는지다 —
// 자기 닫는 태그와 이미 진단된 깨진 태그는 스택에 올리지 않는다.
func (p *parser) parseStartTag() (elem *Element, stillOpen bool) {
	open := p.take() // `<`
	nameTok := p.peek()
	if nameTok.Kind != TokenName {
		p.errorf(open.Span, "`<` 뒤에 태그 이름이 없다")
		p.skipToTagEnd()
		return nil, false
	}
	p.i++
	elem = &Element{
		Name:     nameTok.Text,
		NameSpan: nameTok.Span,
		Span:     SpanOf(open.Span.Start, nameTok.Span.End),
	}
	if _, known := elementSpecs[elem.Name]; !known {
		p.errorf(elem.NameSpan, "알 수 없는 태그 <%s> (쓸 수 있는 태그: %s)", elem.Name, knownElements())
	}

	for {
		switch tok := p.peek(); tok.Kind {
		case TokenTagEnd:
			p.i++
			elem.Span.End = tok.Span.End
			return elem, true
		case TokenTagEndSelfClose:
			p.i++
			elem.Span.End = tok.Span.End
			return elem, false
		case TokenName:
			p.parseAttr(elem)
		case TokenEquals:
			p.errorf(tok.Span, "`=` 앞에 속성 이름이 없다")
			p.i++
		case TokenString, TokenStringUnterminated:
			p.errorf(tok.Span, "속성 이름 없이 값만 있다")
			p.i++
		case TokenInvalid:
			p.errorf(tok.Span, "태그 안에 쓸 수 없는 문자 %q가 있다", tok.Text)
			p.i++
		case TokenEOF:
			p.errorf(elem.NameSpan, "<%s> 태그가 `>` 없이 파일이 끝났다", elem.Name)
			return elem, false
		default:
			// `<`나 `</`가 태그 안에 나왔다 — 이 태그의 `>`가 없다는 뜻이다. 여기서 태그를
			// 끝내고 그 토큰은 문서 흐름으로 넘긴다 (렉서가 `<`를 넘겨 준 이유와 같다).
			p.errorf(elem.NameSpan, "<%s> 태그가 `>`로 닫히지 않았다", elem.Name)
			return elem, true
		}
	}
}

// parseAttr는 속성 하나를 읽어 elem에 붙인다.
func (p *parser) parseAttr(elem *Element) {
	nameTok := p.take()
	attr := Attr{Name: nameTok.Text, NameSpan: nameTok.Span}
	if p.peek().Kind == TokenEquals {
		p.i++
		attr.HasValue = true
		switch value := p.peek(); value.Kind {
		case TokenString:
			p.i++
			attr.Value, attr.ValueSpan = unquote(value)
		case TokenStringUnterminated:
			p.i++
			p.errorf(value.Span, "속성 %s의 값에 닫는 인용 부호가 없다", attr.Name)
			attr.Value, attr.ValueSpan = unquote(value)
		case TokenName:
			p.i++
			p.errorf(value.Span, "속성 값은 인용 부호로 감싼다 (%s=%q)", attr.Name, value.Text)
			attr.Value, attr.ValueSpan = value.Text, value.Span
		default:
			// 값이 아예 없는 속성은 모델에 넣지 않는다 — 진단은 이미 나갔고, 뒤 층이
			// `<slide id>`(값 없는 불리언)와 구별할 수 없는 반쪽 속성을 보게 하지 않는다.
			p.errorf(nameTok.Span, "속성 %s에 `=`만 있고 값이 없다", attr.Name)
			return
		}
	}

	if _, dup := elem.Attr(attr.Name); dup {
		p.errorf(attr.NameSpan, "속성 %s가 중복됐다 (앞의 것만 쓴다)", attr.Name)
		return
	}
	p.checkAttrName(elem, attr)
	elem.Attrs = append(elem.Attrs, attr)
}

// parseEndTag는 닫는 태그를 읽는다. 이름을 찾지 못하면 빈 이름을 돌려준다.
func (p *parser) parseEndTag() (name string, tagSpan Span) {
	open := p.take() // `</`
	nameTok := p.peek()
	if nameTok.Kind != TokenName {
		p.errorf(open.Span, "`</` 뒤에 태그 이름이 없다")
		p.skipToTagEnd()
		return "", open.Span
	}
	p.i++
	name = nameTok.Text
	extra := false
	for {
		switch tok := p.peek(); tok.Kind {
		case TokenTagEnd, TokenTagEndSelfClose:
			p.i++
			if extra {
				p.errorf(nameTok.Span, "닫는 태그 </%s>에는 속성을 쓸 수 없다", name)
			}
			return name, SpanOf(open.Span.Start, tok.Span.End)
		case TokenEOF, TokenTagStart, TokenTagStartClose:
			p.errorf(nameTok.Span, "닫는 태그 </%s>가 `>`로 닫히지 않았다", name)
			return name, SpanOf(open.Span.Start, nameTok.Span.End)
		default:
			extra = true
			p.i++
		}
	}
}

// skipToTagEnd는 회복용으로 태그의 나머지를 버린다.
func (p *parser) skipToTagEnd() {
	for {
		switch p.peek().Kind {
		case TokenTagEnd, TokenTagEndSelfClose:
			p.i++
			return
		case TokenEOF, TokenTagStart, TokenTagStartClose:
			return
		default:
			p.i++
		}
	}
}

// checkPlacement는 요소가 놓인 자리를 검증한다. stack은 아직 열려 있는 조상들이다 —
// <a> 안의 <a>처럼 부모만 봐서는 잡을 수 없는 규칙이 있어 부모가 아니라 스택을 받는다.
func (p *parser) checkPlacement(elem *Element, stack []*Element, root *Element) {
	if _, known := elementSpecs[elem.Name]; !known {
		return // 미지의 태그는 이미 진단됐다 — 같은 오타로 진단을 두 번 내지 않는다.
	}
	parent := parentOf(stack)
	if parent == nil {
		switch {
		case elem.Name != RootElement:
			p.errorf(elem.NameSpan, "루트 요소는 <%s>여야 한다 (<%s>는 그 안에 쓴다)", RootElement, elem.Name)
		case root != nil:
			p.errorf(elem.NameSpan, "루트 요소 <%s>는 파일에 하나여야 한다 (슬라이드 하나가 파일 하나다)", RootElement)
		}
		return
	}
	spec, known := elementSpecs[parent.Name]
	if !known {
		return
	}
	switch spec.model {
	case modelText, modelPre:
		p.errorf(elem.NameSpan, "<%s> 안에는 문자 데이터만 올 수 있다 (<%s>는 쓸 수 없다)", parent.Name, elem.Name)
		return
	case modelVoid:
		return // void 위반은 여는 태그에서 이미 진단됐다 (parseDocument).
	}
	// <li>는 어디에 있든 리스트 컨테이너의 자식이어야 한다 — 자리별 일반 메시지보다
	// "리스트 안에 쓴다"가 고칠 방법을 곧장 말한다.
	if KindOf(elem.Name) == KindListItem && KindOf(parent.Name) != KindListContainer {
		p.errorf(elem.NameSpan, "<%s>는 <%s>/<%s>의 자식으로만 쓸 수 있다",
			ListItemElement, listContainerElements[0], listContainerElements[1])
		return
	}
	if spec.model == modelInline {
		switch KindOf(elem.Name) {
		case KindInline:
			if elem.Name != "a" {
				return
			}
			// <a> 안의 <a>는 에러다 — 겹친 링크는 "무엇을 클릭했는가"가 정의되지 않고,
			// PPTX hlinkClick은 런 하나에 링크 하나다. 사이에 다른 인라인이 끼어도 잡히게
			// 부모가 아니라 열린 조상 전체를 본다.
			for _, open := range stack {
				if open.Name == "a" {
					p.errorf(elem.NameSpan, "<a> 안에 <a>를 겹칠 수 없다")
					return
				}
			}
			return
		case KindListContainer:
			if KindOf(parent.Name) == KindListItem {
				// PPTX의 들여쓰기 단계(lvl) 매핑 없이 열면 웹에서만 동작한다 — 한 타깃만
				// 그리는 어휘는 열지 않는다.
				p.errorf(elem.NameSpan, "<%s> 안에 <%s>을 겹칠 수 없다 — 중첩 리스트는 아직 지원하지 않는다",
					parent.Name, elem.Name)
				return
			}
			p.errorf(elem.NameSpan, "<%s> 안에 블록 태그 <%s>는 올 수 없다 (블록은 겹치지 않는다)", parent.Name, elem.Name)
			return
		case KindBlock:
			// 블록은 슬라이드 직속이다 — 블록 안의 블록은 "문단 안의 문단"이라 두 타깃
			// 어디에도 대응하는 구조가 없다.
			p.errorf(elem.NameSpan, "<%s> 안에 블록 태그 <%s>는 올 수 없다 (블록은 겹치지 않는다)", parent.Name, elem.Name)
			return
		default:
			p.errorf(elem.NameSpan, "<%s> 안에는 문자 데이터와 인라인 태그만 올 수 있다 (<%s>는 쓸 수 없다)",
				parent.Name, elem.Name)
			return
		}
	}
	for _, allowed := range spec.children {
		if allowed == elem.Name {
			return
		}
	}
	p.errorf(elem.NameSpan, "<%s>의 자식으로 <%s>는 올 수 없다 (쓸 수 있는 태그: %s)",
		parent.Name, elem.Name, tagList(spec.children))
}

// checkAttrName은 속성 이름이 태그의 닫힌 어휘 안에 있는지 본다. 값은 보지 않는다.
func (p *parser) checkAttrName(elem *Element, attr Attr) {
	spec, known := elementSpecs[elem.Name]
	if !known {
		return
	}
	for _, allowed := range spec.attrs {
		if allowed == attr.Name {
			return
		}
	}
	if len(spec.attrs) == 0 {
		p.errorf(attr.NameSpan, "<%s>에는 속성을 쓸 수 없다 (%s)", elem.Name, attr.Name)
		return
	}
	p.errorf(attr.NameSpan, "<%s>에 알 수 없는 속성 %s (쓸 수 있는 속성: %s)",
		elem.Name, attr.Name, strings.Join(spec.attrs, ", "))
}

// closeElement는 요소를 닫으며 콘텐츠 모델에 따른 정규화를 마친다.
func (p *parser) closeElement(elem *Element) {
	switch specOf(elem).model {
	case modelText, modelInline:
		elem.Children = trimEdgeSpace(elem.Children)
	case modelPre:
		elem.Children = trimPreEdges(elem.Children)
	}
}

// fillSlide는 루트 `<slide>`에서 슬라이드 메타데이터를 뽑는다.
//
// `id`와 `skip`은 외관이 아니라 **구조** 속성이다 — 어떤 슬라이드가 덱에 들어가고 어떤 앵커를
// 갖는지를 정하므로 값 해석까지 파서가 한다. `<text>`의 role/size/color/align은 원문 그대로
// 남겨 core가 해석한다: 외관 값의 어휘는 테마(Iteration 10)와 함께 자라기 때문에 파서가 알 수
// 있는 것이 아니다.
func (p *parser) fillSlide(doc *SlideDoc) {
	root := doc.Root
	if root.Name != RootElement {
		return
	}
	if attr, ok := root.Attr("id"); ok {
		switch {
		case !attr.HasValue:
			p.errorf(attr.NameSpan, "id 속성에는 값이 필요하다 (id=\"architecture\")")
		case attr.Value == "":
			p.errorf(attr.ValueSpan, "id 속성이 비어 있다 (지우면 파일명에서 유도한다)")
		default:
			doc.ID = attr.Value
		}
	}
	if attr, ok := root.Attr("skip"); ok {
		switch {
		case !attr.HasValue, attr.Value == "true":
			// 값 없는 `<slide skip>`은 참이다 — HTML 불리언 속성 관례.
			doc.Skip = true
		case attr.Value == "false":
		default:
			p.errorf(attr.ValueSpan, "skip 값은 true 또는 false다 (%q)", attr.Value)
		}
	}

	var note *Element
	for _, child := range root.Children {
		elem, ok := child.(*Element)
		if !ok {
			continue
		}
		if _, known := elementSpecs[elem.Name]; !known {
			continue // 이미 진단됐다. 진단이 있는 문서는 렌더에 쓰이지 않는다.
		}
		if elem.Name != noteElement {
			doc.Content = append(doc.Content, elem)
			continue
		}
		if note != nil {
			p.errorf(elem.NameSpan, "<%s>는 슬라이드당 하나만 쓸 수 있다 (%s의 것을 쓴다)",
				noteElement, note.NameSpan.Start)
			continue
		}
		note = elem
		doc.Notes = elem.TextContent()
	}
}

// specOf는 요소의 문법 규칙이다. 미지의 태그는 요소만 담는 모델로 취급한다 — 그 태그의
// 진단은 이미 나갔고, 안쪽 내용으로 진단을 부풀리지 않는 쪽을 고른다.
func specOf(elem *Element) elementSpec {
	return elementSpecs[elem.Name]
}

func parentOf(stack []*Element) *Element {
	if len(stack) == 0 {
		return nil
	}
	return stack[len(stack)-1]
}

// knownElements는 진단에 실을 태그 어휘 목록이다. 정렬해서 메시지가 결정적이게 한다.
func knownElements() string {
	names := make([]string, 0, len(elementSpecs))
	for name := range elementSpecs {
		names = append(names, name)
	}
	sort.Strings(names)
	return tagList(names)
}

func tagList(names []string) string {
	quoted := make([]string, len(names))
	for i, name := range names {
		quoted[i] = "<" + name + ">"
	}
	return strings.Join(quoted, ", ")
}

// asciiSpace는 접기 대상 공백이다 — 렉서가 태그 안에서 구분자로 보는 집합과 같다.
const asciiSpace = " \t\n\r"

// collapseSpace는 공백 런을 한 칸으로 접는다. 바이트 단위로 훑어도 안전하다: UTF-8
// 연속 바이트는 ASCII 범위에 들어오지 않는다.
func collapseSpace(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	space := false
	for i := 0; i < len(s); i++ {
		if strings.IndexByte(asciiSpace, s[i]) >= 0 {
			space = true
			continue
		}
		if space {
			b.WriteByte(' ')
			space = false
		}
		b.WriteByte(s[i])
	}
	if space {
		b.WriteByte(' ')
	}
	return b.String()
}

// trimEdgeSpace는 자식 목록 **양 끝**의 공백만 잘라낸다.
//
// 요소 경계의 공백은 저자가 들여쓰기로 넣은 것이지 내용이 아니다 — `<text>\n  제목\n</text>`는
// "제목"이다. 반면 자식 사이의 공백은 남긴다: Iteration 2의 인라인 태그에서
// `<strong>배포</strong> 자동화`의 한 칸은 내용이다.
func trimEdgeSpace(children []Node) []Node {
	if len(children) == 0 {
		return children
	}
	if first, ok := children[0].(*Text); ok {
		first.Value = strings.TrimLeft(first.Value, " ")
	}
	if last, ok := children[len(children)-1].(*Text); ok {
		last.Value = strings.TrimRight(last.Value, " ")
	}
	out := children[:0]
	for _, child := range children {
		if text, ok := child.(*Text); ok && text.Value == "" {
			continue
		}
		out = append(out, child)
	}
	return out
}

// trimPreEdges는 <pre> 양 끝의 개행을 **하나씩만** 벗긴다 (HTML 관례).
//
// `<pre>` 직후의 개행은 저자가 다음 줄부터 쓰기 위한 것이지 내용이 아니고, 닫는 태그
// 직전의 개행도 같다. 딱 하나만 벗기므로 빈 줄로 시작하는 코드(`\n\ncode`)는 표현할 수
// 있다. 개행 뒤에 들여쓰기가 붙은 `code\n  </pre>`의 개행은 "직전"이 아니므로 벗기지
// 않는다 — pre 안의 공백은 전부 내용이라는 원칙이 우선한다.
func trimPreEdges(children []Node) []Node {
	if len(children) == 0 {
		return children
	}
	if first, ok := children[0].(*Text); ok {
		first.Value = cutEdgeNewline(first.Value, strings.CutPrefix)
	}
	if last, ok := children[len(children)-1].(*Text); ok {
		last.Value = cutEdgeNewline(last.Value, strings.CutSuffix)
	}
	out := children[:0]
	for _, child := range children {
		if text, ok := child.(*Text); ok && text.Value == "" {
			continue
		}
		out = append(out, child)
	}
	return out
}

// cutEdgeNewline은 한쪽 끝의 개행 하나를 벗긴다. CRLF도 개행 하나다.
func cutEdgeNewline(s string, cut func(string, string) (string, bool)) string {
	for _, newline := range []string{"\r\n", "\n"} {
		if rest, ok := cut(s, newline); ok {
			return rest
		}
	}
	return s
}

// unquote는 값 토큰에서 값과 인용 부호를 제외한 구간을 뽑는다. 값 안에 줄바꿈은 있을 수
// 없으므로(렉서가 줄 끝에서 멈춘다) 끝 지점은 같은 줄에서 한 칸 물러난 곳이다.
func unquote(tok Token) (string, Span) {
	start, end := tok.Span.Start, tok.Span.End
	start.Offset++
	start.Col++
	if tok.Kind == TokenString {
		end.Offset--
		end.Col--
		return tok.Text[1 : len(tok.Text)-1], SpanOf(start, end)
	}
	// 닫히지 않은 값은 여는 부호만 벗긴다.
	return tok.Text[1:], SpanOf(start, end)
}

// dataSpan은 문자 데이터에서 공백을 뺀 구간이다 — 진단이 들여쓰기가 아니라 글자를 가리키게 한다.
func dataSpan(tok Token) Span {
	body := strings.TrimLeft(tok.Text, asciiSpace)
	start := advance(tok.Span.Start, tok.Text[:len(tok.Text)-len(body)])
	body = strings.TrimRight(body, asciiSpace)
	return SpanOf(start, advance(start, body))
}
