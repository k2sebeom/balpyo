package quick

import (
	"sort"
	"strconv"
	"strings"
)

// style은 요소 하나의 계산된 선언이다 — cascade를 마치고 var()를 풀어 낸 값의 표.
// 값은 문자열로 두고 쓰는 쪽이 타입별 파서로 읽는다 (퍼센트는 배치 때에야 기준을 안다).
type style struct {
	props    map[string]string
	parent   *style
	fontSize float64
}

// inherited는 부모에게서 물려받는 속성이다. text-decoration은 CSS에서 상속이 아니라
// 인라인 자손에 "전파"되지만, 그리는 결과가 같으므로 상속으로 다룬다.
var inherited = map[string]bool{
	"color": true, "font-family": true, "font-size": true, "font-weight": true, "font-style": true,
	"line-height": true, "text-align": true, "letter-spacing": true, "text-transform": true,
	"word-break": true, "overflow-wrap": true, "white-space": true, "visibility": true,
	"text-indent": true, "fill": true, "stroke": true, "stroke-width": true, "stroke-dasharray": true,
	"fill-opacity": true, "fill-rule": true, "stroke-opacity": true, "text-decoration": true, "list-style": true,
	"stroke-linecap": true, "stroke-linejoin": true, "stroke-miterlimit": true,
	"marker-start": true, "marker-end": true, "marker-mid": true,
}

func (s *style) get(name string) string {
	if s == nil {
		return ""
	}
	return s.props[name]
}

func (s *style) has(name string) bool {
	if s == nil {
		return false
	}
	_, ok := s.props[name]
	return ok
}

// uaCSS는 웹 렌더러 출력이 기대는 브라우저 기본값의 부분집합이다.
const uaCSS = `
html, body, div, section, p, h1, h2, h3, ul, ol, li, blockquote, pre, table, figure { display: block; }
head, script, style, title, meta, link, defs, marker, pattern, linearGradient, radialGradient, [hidden] { display: none; }
span, strong, em, code, a, mark, br, b, i, u, s, sub, sup, small { display: inline; }
img, svg { display: inline; }
table { display: table; }
tr { display: table-row; }
td, th { display: table-cell; vertical-align: middle; }
th { font-weight: 700; text-align: center; }
strong, b, th { font-weight: 700; }
em, i { font-style: italic; }
pre, code { font-family: monospace; }
pre { white-space: pre; }
p { margin: 1em 0; }
h1 { font-size: 2em; margin: 0.67em 0; font-weight: 700; }
h2 { font-size: 1.5em; margin: 0.83em 0; font-weight: 700; }
h3 { font-size: 1.17em; margin: 1em 0; font-weight: 700; }
ul, ol { margin: 1em 0; padding-left: 40px; }
blockquote { margin: 1em 40px; }
a { text-decoration: underline; }
mark { background-color: yellow; }
html { color: black; font-size: 16px; line-height: normal; font-family: sans-serif; }
`

// cascader는 규칙 전체를 들고 요소마다 계산값을 만든다.
type cascader struct {
	sheet *sheet
}

func newCascader(sheets []string) *cascader {
	s := newSheet()
	s.parse(uaCSS, rankUA)
	for _, src := range sheets {
		s.parse(src, rankUnlayered)
	}
	return &cascader{sheet: s}
}

type matched struct {
	rank  int
	spec  specificity
	order int
	decls []decl
}

// apply는 트리 전체에 계산값을 붙인다.
func (c *cascader) apply(root *node) {
	var walk func(n *node, parent *style)
	walk = func(n *node, parent *style) {
		if n.isText() {
			n.style = parent
			return
		}
		n.style = c.compute(n, parent, "")
		if before := c.compute(n, n.style, "before"); before != nil {
			if content := before.get("content"); content != "" && content != "none" && content != "normal" {
				n.before = before
			}
		}
		for _, child := range n.children {
			walk(child, n.style)
		}
	}
	walk(root, nil)
}

func (c *cascader) compute(n *node, parent *style, pseudo string) *style {
	var hits []matched
	for _, r := range c.sheet.rules {
		var best *complexSel
		for i := range r.selectors {
			sel := &r.selectors[i]
			if sel.pseudo != pseudo {
				continue
			}
			if sel.matches(n) && (best == nil || best.spec.less(sel.spec)) {
				best = sel
			}
		}
		if best != nil {
			hits = append(hits, matched{rank: r.rank, spec: best.spec, order: r.order, decls: r.decls})
		}
	}
	if pseudo != "" && len(hits) == 0 {
		return nil
	}
	if pseudo == "" {
		if inline, ok := n.attrs["style"]; ok {
			hits = append(hits, matched{rank: rankInline, order: 1 << 30, decls: parseDecls(inline)})
		}
		// SVG 표현 속성(fill="none" 등)은 작성자 스타일보다 아래, UA보다 위다.
		if pres := presentationDecls(n); len(pres) > 0 {
			hits = append(hits, matched{rank: rankUA + 1, order: -1, decls: pres})
		}
	}
	sort.SliceStable(hits, func(i, j int) bool {
		a, b := hits[i], hits[j]
		if a.rank != b.rank {
			return a.rank < b.rank
		}
		if a.spec != b.spec {
			return a.spec.less(b.spec)
		}
		return a.order < b.order
	})

	own := map[string]string{}
	for _, h := range hits {
		for _, d := range h.decls {
			expandDecl(own, d.name, d.value)
		}
	}

	st := &style{props: map[string]string{}, parent: parent}
	// 상속값과 사용자 지정 속성(항상 상속)을 먼저 깐다.
	if parent != nil {
		for k, v := range parent.props {
			if inherited[k] || strings.HasPrefix(k, "--") {
				st.props[k] = v
			}
		}
	}
	// 사용자 지정 속성: 자기 선언끼리의 참조를 풀어야 하므로 두 단계다.
	for k, v := range own {
		if strings.HasPrefix(k, "--") {
			st.props[k] = v
		}
	}
	for k := range own {
		if strings.HasPrefix(k, "--") {
			st.props[k] = substituteVars(st.props[k], st.props, 0)
		}
	}
	for k, v := range own {
		if strings.HasPrefix(k, "--") {
			continue
		}
		switch v {
		case "inherit":
			if parent != nil {
				if pv, ok := parent.props[k]; ok {
					st.props[k] = pv
					continue
				}
			}
			delete(st.props, k)
			continue
		case "initial", "unset", "revert", "revert-layer":
			delete(st.props, k)
			continue
		}
		st.props[k] = substituteVars(v, st.props, 0)
	}

	// font-size는 em의 기준이므로 여기서 px로 확정한다.
	parentSize := 16.0
	if parent != nil {
		parentSize = parent.fontSize
	}
	st.fontSize = parentSize
	if v, ok := own["font-size"]; ok && v != "inherit" {
		if l, ok := parseLength(st.props["font-size"], parentSize); ok && !l.auto {
			st.fontSize = l.px + l.pct*parentSize/100
		}
	}
	st.props["font-size"] = strconv.FormatFloat(st.fontSize, 'f', -1, 64) + "px"
	return st
}

// presentationDecls는 SVG 요소의 표현 속성을 선언으로 바꾼다.
func presentationDecls(n *node) []decl {
	var out []decl
	for _, name := range []string{"fill", "stroke", "stroke-width", "stroke-dasharray", "opacity", "fill-opacity", "fill-rule", "stroke-opacity", "stop-color", "stop-opacity"} {
		if v, ok := n.attrs[name]; ok {
			out = append(out, decl{name: name, value: v})
		}
	}
	return out
}

// substituteVars는 var(--x, fallback)을 표의 값으로 바꾼다.
func substituteVars(v string, props map[string]string, depth int) string {
	if depth > 8 || !strings.Contains(v, "var(") {
		return v
	}
	var b strings.Builder
	for {
		i := strings.Index(v, "var(")
		if i < 0 {
			b.WriteString(v)
			break
		}
		b.WriteString(v[:i])
		end := closingParen(v, i+3)
		inner := v[i+4 : end]
		name, fallback, hasFallback := strings.Cut(inner, ",")
		name = strings.TrimSpace(name)
		if val, ok := props[name]; ok {
			b.WriteString(substituteVars(val, props, depth+1))
		} else if hasFallback {
			b.WriteString(substituteVars(strings.TrimSpace(fallback), props, depth+1))
		}
		v = v[end+1:]
	}
	return b.String()
}

// closingParen은 open 위치의 '('와 짝인 ')'의 위치다.
func closingParen(v string, open int) int {
	depth := 0
	for j := open; j < len(v); j++ {
		switch v[j] {
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				return j
			}
		}
	}
	return len(v) - 1
}

// expandDecl은 단축 속성을 개별 속성으로 펼쳐 표에 쓴다.
func expandDecl(props map[string]string, name, value string) {
	set := func(k, v string) { props[k] = v }
	box := func(prefix, suffix string) {
		parts := splitValues(value)
		var t, r, b, l string
		switch len(parts) {
		case 1:
			t, r, b, l = parts[0], parts[0], parts[0], parts[0]
		case 2:
			t, r, b, l = parts[0], parts[1], parts[0], parts[1]
		case 3:
			t, r, b, l = parts[0], parts[1], parts[2], parts[1]
		default:
			t, r, b, l = parts[0], parts[1], parts[2], parts[3]
		}
		set(prefix+"top"+suffix, t)
		set(prefix+"right"+suffix, r)
		set(prefix+"bottom"+suffix, b)
		set(prefix+"left"+suffix, l)
	}
	switch name {
	case "padding":
		box("padding-", "")
	case "margin":
		box("margin-", "")
	case "inset":
		box("", "")
	case "border-width":
		box("border-", "-width")
	case "border-style":
		box("border-", "-style")
	case "border-color":
		box("border-", "-color")
	case "padding-inline":
		parts := splitValues(value)
		set("padding-left", parts[0])
		set("padding-right", parts[len(parts)-1])
	case "padding-block":
		parts := splitValues(value)
		set("padding-top", parts[0])
		set("padding-bottom", parts[len(parts)-1])
	case "margin-inline":
		parts := splitValues(value)
		set("margin-left", parts[0])
		set("margin-right", parts[len(parts)-1])
	case "margin-block":
		parts := splitValues(value)
		set("margin-top", parts[0])
		set("margin-bottom", parts[len(parts)-1])
	case "padding-inline-start":
		set("padding-left", value)
	case "padding-inline-end":
		set("padding-right", value)
	case "margin-inline-start":
		set("margin-left", value)
	case "margin-inline-end":
		set("margin-right", value)
	case "padding-block-start":
		set("padding-top", value)
	case "padding-block-end":
		set("padding-bottom", value)
	case "margin-block-start":
		set("margin-top", value)
	case "margin-block-end":
		set("margin-bottom", value)
	case "inline-size":
		set("width", value)
	case "block-size":
		set("height", value)
	case "min-inline-size":
		set("min-width", value)
	case "min-block-size":
		set("min-height", value)
	case "max-inline-size":
		set("max-width", value)
	case "max-block-size":
		set("max-height", value)
	case "border", "border-top", "border-right", "border-bottom", "border-left":
		sides := []string{"top", "right", "bottom", "left"}
		if name != "border" {
			sides = []string{strings.TrimPrefix(name, "border-")}
		}
		width, styleV, color := "medium", "none", "currentcolor"
		for _, p := range splitValues(value) {
			switch {
			case isBorderStyle(p):
				styleV = p
			case isLengthish(p):
				width = p
			default:
				color = p
			}
		}
		for _, s := range sides {
			set("border-"+s+"-width", width)
			set("border-"+s+"-style", styleV)
			set("border-"+s+"-color", color)
		}
	case "flex":
		parts := splitValues(value)
		grow, shrink, basis := "0", "1", "auto"
		switch {
		case len(parts) == 1 && parts[0] == "none":
			grow, shrink, basis = "0", "0", "auto"
		case len(parts) == 1 && parts[0] == "auto":
			grow, shrink, basis = "1", "1", "auto"
		case len(parts) == 1 && isNumber(parts[0]):
			grow, shrink, basis = parts[0], "1", "0px"
		case len(parts) == 1:
			grow, shrink, basis = "1", "1", parts[0]
		case len(parts) == 2 && isNumber(parts[1]):
			grow, shrink, basis = parts[0], parts[1], "0px"
		case len(parts) == 2:
			grow, shrink, basis = parts[0], "1", parts[1]
		default:
			grow, shrink, basis = parts[0], parts[1], parts[2]
		}
		set("flex-grow", grow)
		set("flex-shrink", shrink)
		set("flex-basis", basis)
	case "gap", "grid-gap":
		parts := splitValues(value)
		set("row-gap", parts[0])
		set("column-gap", parts[len(parts)-1])
	case "grid-area":
		parts := splitTop(value, '/')
		for i := range parts {
			parts[i] = strings.TrimSpace(parts[i])
		}
		if len(parts) == 1 && isIdent(parts[0]) {
			set("grid-row-start", parts[0])
			set("grid-column-start", parts[0])
			set("grid-row-end", parts[0])
			set("grid-column-end", parts[0])
			return
		}
		get := func(i int, def string) string {
			if i < len(parts) {
				return parts[i]
			}
			return def
		}
		set("grid-row-start", get(0, "auto"))
		set("grid-column-start", get(1, "auto"))
		set("grid-row-end", get(2, "auto"))
		set("grid-column-end", get(3, "auto"))
	case "grid-column", "grid-row":
		parts := splitTop(value, '/')
		set(name+"-start", strings.TrimSpace(parts[0]))
		if len(parts) > 1 {
			set(name+"-end", strings.TrimSpace(parts[1]))
		} else {
			set(name+"-end", "auto")
		}
	case "place-self":
		parts := splitValues(value)
		set("align-self", parts[0])
		set("justify-self", parts[len(parts)-1])
	case "place-items":
		parts := splitValues(value)
		set("align-items", parts[0])
		set("justify-items", parts[len(parts)-1])
	case "place-content":
		parts := splitValues(value)
		set("align-content", parts[0])
		set("justify-content", parts[len(parts)-1])
	case "background":
		// 미리보기가 쓰는 형태는 색 하나 또는 그라디언트 하나다.
		if strings.Contains(value, "gradient(") || strings.Contains(value, "url(") {
			set("background-image", value)
		} else {
			set("background-color", value)
		}
	case "flex-flow":
		for _, p := range splitValues(value) {
			if strings.Contains(p, "wrap") {
				set("flex-wrap", p)
			} else {
				set("flex-direction", p)
			}
		}
	case "list-style":
		set("list-style", value)
	case "font":
		// 뷰어 크롬에서만 쓴다 — 미리보기는 그리지 않는다.
	default:
		set(name, value)
	}
}

func isIdent(v string) bool {
	if v == "" || (v[0] >= '0' && v[0] <= '9') {
		return false
	}
	for i := 0; i < len(v); i++ {
		if !isIdentByte(v[i]) {
			return false
		}
	}
	return v != "auto"
}

func isBorderStyle(v string) bool {
	switch v {
	case "none", "hidden", "solid", "dashed", "dotted", "double", "groove", "ridge", "inset", "outset":
		return true
	}
	return false
}

func isNumber(v string) bool {
	_, err := strconv.ParseFloat(v, 64)
	return err == nil
}

func isLengthish(v string) bool {
	if v == "thin" || v == "medium" || v == "thick" || strings.HasPrefix(v, "calc(") {
		return true
	}
	if len(v) > 0 && (v[0] >= '0' && v[0] <= '9' || v[0] == '.' || v[0] == '-') {
		return true
	}
	return false
}

// splitValues는 공백으로 값을 나눈다 (괄호 · 문자열 안의 공백은 나누지 않는다).
func splitValues(v string) []string {
	var out []string
	depth := 0
	inStr := byte(0)
	start := -1
	for j := 0; j < len(v); j++ {
		c := v[j]
		if inStr != 0 {
			if c == inStr {
				inStr = 0
			}
			continue
		}
		switch {
		case c == '"' || c == '\'':
			inStr = c
			if start < 0 {
				start = j
			}
		case c == '(':
			depth++
			if start < 0 {
				start = j
			}
		case c == ')':
			depth--
		case isSpace(c) && depth == 0:
			if start >= 0 {
				out = append(out, v[start:j])
				start = -1
			}
		default:
			if start < 0 {
				start = j
			}
		}
	}
	if start >= 0 {
		out = append(out, v[start:])
	}
	if len(out) == 0 {
		out = []string{""}
	}
	return out
}
