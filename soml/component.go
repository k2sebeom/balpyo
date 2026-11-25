package soml

import (
	"regexp"
	"sort"
	"strings"
)

// DefineElement는 컴포넌트 파일의 루트 요소다 — `<define name="step" params="icon label">`.
// 파일의 역할은 확장자가 아니라 루트 요소가 정한다 (슬라이드도 컴포넌트도 `.slide`다).
const DefineElement = "define"

// SlotElement는 정의 안에서 **사용처의 자식이 들어갈 자리**다 (`<slot/>`). 정의 파일에서만 어휘다 —
// 슬라이드에 쓴 `<slot>`은 모르는 태그다.
const SlotElement = "slot"

// Component는 구조 컴포넌트 정의 하나다 — 이름, 인자, 그리고 펼쳐질 몸체.
//
// 정의는 **데이터**다: 반복 · 조건 · 식이 없고, 인자를 속성 값과 글자 안의 `{name}`에 치환하는 것과
// 사용처의 자식을 `<slot/>` 자리에 끼우는 것이 전부다. 반복은 사용처가 자식을 여러 개 적는 것으로
// 한다 (`<steps>` 안의 `<step>` 셋). 테마는 코드를 돌리지 않는다는 원칙이 컴포넌트에도 그대로 선다.
type Component struct {
	// Name은 사용처의 태그 이름이다.
	Name string
	// File은 정의 파일의 경로다 — 펼친 결과 안의 진단이 정의 위치를 함께 말하는 근거.
	File string
	// NameSpan은 `name` 값 구간이다 (이름이 겹치는 진단이 가리키는 곳).
	NameSpan Span
	// Params는 선언 순서의 인자다. 순서가 사용처 속성 진단에 실리는 순서다.
	Params []ComponentParam
	// Body는 정의의 유일한 자식 요소 — 펼침의 뿌리다. 그 안의 속성 값 · 글자는 `{name}`을 담은
	// 원문 그대로다 (치환은 펼칠 때 한다).
	Body *Element
	// slotParent는 `<slot/>`을 담은 요소의 이름이다 (`<slot/>`이 없으면 빈 값). 사용처 자식의 문법이
	// 이 요소의 것이다.
	slotParent string
	// use는 사용처 태그의 문법이다 (NewComponents가 채운다).
	use elementSpec
	// rejected는 선언했지만 문법에 걸린 인자 이름이다 — 그 이름의 `{name}`을 "모르는 인자"로 한 번 더
	// 말하지 않는다 (실수 하나에 진단 하나).
	rejected []string
	// unknownSeen은 몸체에서 모르는 `{name}`을 만났는지다. 만났으면 "쓰이지 않은 인자" 경고를 내지
	// 않는다 — `{lable}` 오타 하나가 label을 쓰이지 않은 인자로 만들어 진단이 둘이 된다.
	unknownSeen bool
}

// ComponentParam은 컴포넌트 인자 하나다 (`params="icon label note?"`의 한 낱말).
type ComponentParam struct {
	Name string
	// Optional은 `?`로 끝난 인자다 — 사용처가 생략할 수 있고, 생략하면 빈 문자열로 치환된다.
	Optional bool
	// Span은 `params` 값 안에서 이 낱말의 구간이다.
	Span Span
}

// HasSlot은 정의가 사용처의 자식을 받는지다. 받지 않는 컴포넌트는 사용처에서 void다 (`<marker/>`).
func (c *Component) HasSlot() bool { return c.slotParent != "" }

// Attrs는 사용처 태그가 받는 속성 이름이다 — 인자(선언 순서) 뒤에 뿌리로 넘기는 배치 속성이 온다.
func (c *Component) Attrs() []string {
	attrs := make([]string, len(c.use.attrs))
	copy(attrs, c.use.attrs)
	return attrs
}

// ForwardedAttrs는 사용처 속성 중 **인자가 아니라 정의의 뿌리로 넘어가는** 이름이다 (`grow`·`slot`·
// `enter` …) — `<row><step grow="1">`처럼 컴포넌트도 자기가 놓인 자리의 배치를 받는다. 뿌리가 받는
// 배치 속성만 넘긴다.
func (c *Component) ForwardedAttrs() []string {
	root, _ := elementSpecs[c.Body.Name]
	var forwarded []string
	for _, name := range withChildPlacement(nil) {
		if containsString(root.attrs, name) {
			forwarded = append(forwarded, name)
		}
	}
	return forwarded
}

// Components는 한 덱이 쓸 수 있는 컴포넌트 어휘다 — 테마(와 프로젝트)가 선언한 이름 전체.
// nil은 빈 어휘다.
type Components struct {
	byName map[string]*Component
	names  []string
}

// NewComponents는 정의들로 어휘를 만든다. 같은 이름의 정의가 둘이면 뒤의 것이 에러다 — 테마
// 상속에서 자식이 부모의 정의를 대체하는 규칙은 호출자가 이 목록을 만들기 전에 적용한다.
func NewComponents(defs ...*Component) (*Components, Diagnostics) {
	comps := &Components{byName: map[string]*Component{}}
	var diags Diagnostics
	for _, def := range defs {
		if def == nil {
			continue
		}
		if prior, dup := comps.byName[def.Name]; dup {
			diags = append(diags, Errorf(def.File, def.NameSpan, "component <%s> is already defined at %s:%s",
				def.Name, prior.File, prior.NameSpan.Start))
			continue
		}
		def.use = useSpec(def)
		comps.byName[def.Name] = def
		comps.names = append(comps.names, def.Name)
	}
	sort.Strings(comps.names)
	return comps, diags
}

// Lookup은 이름으로 정의를 찾는다.
func (c *Components) Lookup(name string) (*Component, bool) {
	if c == nil {
		return nil, false
	}
	def, ok := c.byName[name]
	return def, ok
}

// Names는 정의된 이름 전체다 (사전순).
func (c *Components) Names() []string {
	if c == nil {
		return nil
	}
	names := make([]string, len(c.names))
	copy(names, c.names)
	return names
}

// useSpec은 사용처 태그의 문법을 정의에서 파생한다. 속성은 인자 + 넘기는 배치 속성이고, 자식의
// 문법은 `<slot/>`을 담은 요소의 것이다 — 그래야 사용처의 자식이 펼친 자리에서도 유효하다.
func useSpec(def *Component) elementSpec {
	var spec elementSpec
	for _, param := range def.Params {
		spec.attrs = append(spec.attrs, param.Name)
	}
	spec.attrs = append(spec.attrs, def.ForwardedAttrs()...)
	if !def.HasSlot() {
		spec.model = modelVoid
		return spec
	}
	host := elementSpecs[def.slotParent]
	spec.model, spec.children = host.model, host.children
	return spec
}

// defineSpec은 `<define>`의 문법이다. 몸체는 슬라이드에 놓일 수 있는 콘텐츠 하나다 — 노트는
// 슬라이드의 메타데이터이고 연결선은 슬라이드 직속의 오버레이라 컴포넌트의 몸체가 될 수 없다.
var defineSpec = elementSpec{
	attrs:    []string{"name", "params"},
	model:    modelElements,
	children: defineChildren(),
}

// slotSpec은 `<slot/>`의 문법이다 — 자리 표시이므로 담는 것도 속성도 없다.
var slotSpec = elementSpec{model: modelVoid}

func defineChildren() []string {
	var children []string
	for _, name := range slideChildren() {
		if name != noteElement && name != ConnectorElement {
			children = append(children, name)
		}
	}
	return children
}

// componentName은 컴포넌트 이름과 인자 이름의 문법이다 — 소문자로 시작하는 케밥 낱말. 태그 이름과
// 속성 이름으로 그대로 쓰이므로 둘의 공통 부분집합이다.
var componentName = regexp.MustCompile(`^[a-z][a-z0-9]*(-[a-z0-9]+)*$`)

// placeholder는 정의 안의 인자 자리다 (`{label}`). 이름 문법에 맞지 않는 중괄호(`{ x }`, `{}`)는
// 글자 그대로 남는다.
var placeholder = regexp.MustCompile(`\{([a-z][a-z0-9]*(?:-[a-z0-9]+)*)\}`)

// ParseComponent는 컴포넌트 파일 하나를 파싱한다. 정의로 쓸 수 없는 파일(루트 · 이름 · 몸체가
// 없다)이면 정의는 nil이고 이유는 진단에 있다.
//
// 몸체는 슬라이드와 **같은 닫힌 어휘**로 읽는다 — 정의가 파스를 통과하면 펼친 결과도 문법상
// 유효하다. 다른 컴포넌트를 쓰는 정의는 아직 받지 않는다 (펼침이 한 겹이면 순환이 생길 수 없다).
func ParseComponent(file, src string) (*Component, Diagnostics) {
	p := &parser{file: file, toks: Tokenize(src), root: DefineElement}
	root := p.parseDocument()
	var def *Component
	if root == nil {
		p.errorf(SpanOf(StartPos(), StartPos()), "missing root element <%s>", DefineElement)
	} else if root.Name == DefineElement {
		def = p.fillDefine(root, src)
	}
	return def, p.sortedDiags()
}

// fillDefine은 루트 `<define>`에서 정의를 뽑는다.
func (p *parser) fillDefine(root *Element, src string) *Component {
	def := &Component{File: p.file}
	nameOK := false
	switch attr, ok := root.Attr("name"); {
	case !ok:
		p.errorf(root.NameSpan, "<%s> needs a name (name=\"step\")", DefineElement)
	case !attr.HasValue || attr.Value == "":
		p.errorf(attr.NameSpan, "the name attribute needs a value (name=\"step\")")
	case !componentName.MatchString(attr.Value):
		p.errorf(attr.ValueSpan, "component name %q must be lowercase letters, digits and single hyphens, starting with a letter (name=\"step\")", attr.Value)
	case attr.Value == DefineElement || attr.Value == SlotElement || isBuiltinTag(attr.Value):
		p.errorf(attr.ValueSpan, "component name %q is a built-in tag (choose another name)", attr.Value)
	default:
		def.Name, def.NameSpan, nameOK = attr.Value, attr.ValueSpan, true
	}
	if attr, ok := root.Attr("params"); ok {
		def.Params, def.rejected = p.parseParams(attr)
	}

	for _, child := range root.Children {
		elem, ok := child.(*Element)
		if !ok {
			continue
		}
		if _, known := p.spec(elem.Name); !known {
			continue // 이미 진단됐다.
		}
		if def.Body != nil {
			p.errorf(elem.NameSpan, "a component has exactly one root element (wrap <%s> and <%s> in <col> or <row>)",
				def.Body.Name, elem.Name)
			continue
		}
		def.Body = elem
	}
	if def.Body == nil {
		p.errorf(root.NameSpan, "<%s> needs one element as its body (<%s name=\"…\"><col>…</col></%s>)",
			DefineElement, DefineElement, DefineElement)
		return nil
	}
	p.scanBody(def, def.Body, src, map[string]bool{})
	if !nameOK {
		return nil
	}
	return def
}

// parseParams는 `params` 값을 낱말로 나눈다. `?`로 끝난 낱말은 생략할 수 있는 인자다.
func (p *parser) parseParams(attr Attr) (params []ComponentParam, rejected []string) {
	if !attr.HasValue {
		p.errorf(attr.NameSpan, "the params attribute needs a value (params=\"icon label\")")
		return nil, nil
	}
	forwarded := withChildPlacement(nil)
	seen := map[string]bool{}
	for _, word := range spaceWords(attr.Value) {
		span := attr.ValueSpan.Slice(attr.Value, word.start, word.end)
		param := ComponentParam{Name: word.text, Span: span}
		if name, optional := strings.CutSuffix(word.text, "?"); optional {
			param.Name, param.Optional = name, true
		}
		switch {
		case !componentName.MatchString(param.Name):
			p.errorf(span, "parameter %q must be lowercase letters, digits and single hyphens, starting with a letter (end it with ? to make it optional)", word.text)
			rejected = append(rejected, param.Name)
		case seen[param.Name]:
			p.errorf(span, "duplicate parameter %s", param.Name)
		case containsString(forwarded, param.Name):
			p.errorf(span, "parameter %s collides with the placement attribute of the same name (a component passes %s to its root element)", param.Name, param.Name)
			rejected = append(rejected, param.Name)
		default:
			seen[param.Name] = true
			params = append(params, param)
		}
	}
	if len(spaceWords(attr.Value)) == 0 {
		p.errorf(attr.ValueSpan, "the params attribute is empty (remove it for a component without parameters)")
	}
	return params, rejected
}

// scanBody는 몸체를 훑어 `<slot/>`의 자리와 `{name}` 인자 자리를 검사한다. used는 쓰인 인자다.
func (p *parser) scanBody(def *Component, elem *Element, src string, used map[string]bool) {
	for _, attr := range elem.Attrs {
		if attr.HasValue {
			p.checkPlaceholders(def, attr.ValueSpan, src, used)
		}
	}
	for _, child := range elem.Children {
		switch node := child.(type) {
		case *Text:
			p.checkPlaceholders(def, node.Span, src, used)
		case *Element:
			if node.Name == SlotElement {
				if def.HasSlot() {
					p.errorf(node.NameSpan, "a component has at most one <%s/> (the use site's children go to one place)", SlotElement)
					continue
				}
				def.slotParent = elem.Name
				continue
			}
			p.scanBody(def, node, src, used)
		}
	}
	if elem == def.Body && !def.unknownSeen {
		for _, param := range def.Params {
			if !used[param.Name] {
				p.diags = append(p.diags, Warningf(p.file, param.Span,
					"parameter %s is never used in the body of <%s> (write {%s} where it goes)", param.Name, def.Name, param.Name))
			}
		}
	}
}

// checkPlaceholders는 구간의 원문에서 `{name}`을 찾아 선언된 인자인지 본다. 원문을 읽는 이유는
// 위치다: 모델의 글자는 공백이 접히고 참조가 풀린 값이라 오프셋이 원문과 다르다.
func (p *parser) checkPlaceholders(def *Component, span Span, src string, used map[string]bool) {
	raw := span.Text(src)
	for _, m := range placeholder.FindAllStringSubmatchIndex(raw, -1) {
		name := raw[m[2]:m[3]]
		if def.param(name) {
			used[name] = true
			continue
		}
		if containsString(def.rejected, name) {
			continue
		}
		def.unknownSeen = true
		at := span.Slice(raw, m[0], m[1])
		if len(def.Params) == 0 {
			p.errorf(at, "unknown parameter {%s} (this component declares no params — add params=\"%s\" to <%s>)", name, name, DefineElement)
			continue
		}
		hint := ""
		if guess, ok := Suggest(name, def.paramNames()); ok {
			hint = " — did you mean {" + guess + "}?"
		}
		p.errorf(at, "unknown parameter {%s}%s (params: %s)", name, hint, strings.Join(def.paramNames(), ", "))
	}
}

func (c *Component) param(name string) bool {
	for _, param := range c.Params {
		if param.Name == name {
			return true
		}
	}
	return false
}

func (c *Component) paramNames() []string {
	names := make([]string, len(c.Params))
	for i, param := range c.Params {
		names[i] = param.Name
	}
	return names
}

// checkSlotPlacement는 정의 안 `<slot/>`의 자리를 본다 — 요소나 인라인을 담는 부모 안이면 어디든
// 된다. 사용처의 자식이 그 부모의 문법을 따르므로 자리별 자식 목록은 여기서 보지 않는다.
func (p *parser) checkSlotPlacement(elem, parent *Element, spec elementSpec) {
	switch {
	case parent.Name == DefineElement:
		p.errorf(elem.NameSpan, "<%s/> cannot be the body of a component (put it inside the body's root element)", SlotElement)
	case spec.model == modelText || spec.model == modelPre:
		p.errorf(elem.NameSpan, "<%s/> cannot be inside <%s> (it takes only character data)", SlotElement, parent.Name)
	}
}

// checkParams는 사용처의 인자를 본다 — 필수 인자가 빠졌거나 값이 없는 인자. 모르는 속성은
// checkAttrName이 이미 말한다.
func (p *parser) checkParams(elem *Element) {
	def, ok := p.comps.Lookup(elem.Name)
	if !ok {
		return
	}
	var missing []string
	for _, param := range def.Params {
		attr, given := elem.Attr(param.Name)
		switch {
		case given && !attr.HasValue:
			p.errorf(attr.NameSpan, "parameter %s on <%s> needs a value (%s=\"…\")", param.Name, elem.Name, param.Name)
		case !given && !param.Optional:
			missing = append(missing, param.Name)
		}
	}
	if len(missing) > 0 {
		write := make([]string, len(missing))
		for i, name := range missing {
			write[i] = name + `="…"`
		}
		p.errorf(elem.NameSpan, "<%s> is missing %s %s (write %s; defined at %s:%s)",
			elem.Name, plural(len(missing), "parameter", "parameters"), strings.Join(missing, ", "),
			strings.Join(write, " "), def.File, def.NameSpan.Start)
	}
}

// spec은 이 파일에서 태그 하나의 문법이다 — 내장 어휘, 정의 파일의 `<define>`·`<slot>`, 그리고 덱의
// 컴포넌트 어휘 순으로 찾는다.
func (p *parser) spec(name string) (elementSpec, bool) {
	if p.root == DefineElement {
		switch name {
		case DefineElement:
			return defineSpec, true
		case SlotElement:
			return slotSpec, true
		}
	}
	if spec, ok := elementSpecs[name]; ok {
		return spec, true
	}
	if def, ok := p.comps.Lookup(name); ok {
		return def.use, true
	}
	return elementSpec{}, false
}

// placedName은 이 태그가 **자식으로 놓일 때** 판정받는 이름이다 — 컴포넌트는 정의의 뿌리다.
func (p *parser) placedName(name string) string {
	if def, ok := p.comps.Lookup(name); ok {
		return def.Body.Name
	}
	return name
}

// containerName은 이 태그가 **부모일 때** 판정받는 이름이다 — 컴포넌트는 `<slot/>`을 담은 요소다.
func (p *parser) containerName(name string) string {
	if def, ok := p.comps.Lookup(name); ok && def.HasSlot() {
		return def.slotParent
	}
	return name
}

// childrenOf는 진단이 말할 자식 목록이다 — 내장 태그 뒤에, 펼친 뿌리가 그 자리에 놓일 수 있는
// 컴포넌트를 붙인다.
func (p *parser) childrenOf(spec elementSpec) []string {
	names := p.comps.Names()
	if len(names) == 0 {
		return spec.children
	}
	children := append([]string(nil), spec.children...)
	for _, name := range names {
		if containsString(spec.children, p.placedName(name)) {
			children = append(children, name)
		}
	}
	return children
}

// knownNames는 이 파일에서 쓸 수 있는 태그 이름 전체다 (사전순).
func (p *parser) knownNames() []string {
	names := append(knownElementNames(), p.comps.Names()...)
	if p.root == DefineElement {
		names = append(names, DefineElement, SlotElement)
	}
	sort.Strings(names)
	return names
}

// oneRootReason은 "루트가 둘" 진단의 이유 구절이다.
func (p *parser) oneRootReason() string {
	if p.root == DefineElement {
		return "one component per file"
	}
	return "one slide per file"
}

func isBuiltinTag(name string) bool {
	_, ok := elementSpecs[name]
	return ok || name == noteElement
}

func containsString(list []string, s string) bool {
	for _, item := range list {
		if item == s {
			return true
		}
	}
	return false
}

func plural(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return many
}

// spaceWord는 공백으로 나뉜 낱말과 그 바이트 구간이다.
type spaceWord struct {
	text       string
	start, end int
}

func spaceWords(s string) []spaceWord {
	var words []spaceWord
	start := -1
	for i := 0; i <= len(s); i++ {
		space := i == len(s) || strings.IndexByte(asciiSpace, s[i]) >= 0
		switch {
		case space && start >= 0:
			words = append(words, spaceWord{text: s[start:i], start: start, end: i})
			start = -1
		case !space && start < 0:
			start = i
		}
	}
	return words
}
