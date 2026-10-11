package quick

import (
	"strings"
)

// 레이어 순위다 — 값이 클수록 이긴다 (보통 선언 기준). UA 기본값이 맨 아래, 이름 없는 레이어
// 밖의 규칙이 레이어보다 위, inline style이 맨 위다.
const (
	rankUA        = 0
	rankUnlayered = 1000
	rankInline    = 2000
)

// decl은 선언 하나다 (이름은 소문자, 값은 앞뒤 공백을 뗀 원문).
type decl struct {
	name  string
	value string
}

// rule은 선택자 목록 하나와 그 선언들이다.
type rule struct {
	selectors []complexSel
	decls     []decl
	rank      int
	order     int
}

// sheet는 cascade에 들어가는 규칙 전체다 (문서 순서).
type sheet struct {
	rules  []rule
	layers map[string]int // 레이어 이름 → 순위 (선언된 순서대로 1부터)
}

func newSheet() *sheet { return &sheet{layers: map[string]int{}} }

func (s *sheet) layerRank(name string) int {
	if r, ok := s.layers[name]; ok {
		return r
	}
	r := len(s.layers) + 1
	s.layers[name] = r
	return r
}

// parse는 스타일시트 하나를 읽어 규칙을 더한다. @media · @keyframes · @page · @font-face는
// 건너뛴다 — 미리보기는 화면 매체이고 애니메이션 전/후 상태가 아니라 최종 상태를 그린다.
func (s *sheet) parse(src string, rank int) {
	src = stripComments(src)
	s.parseBlock(src, rank, "")
}

func (s *sheet) parseBlock(src string, rank int, layerPrefix string) {
	i := 0
	for i < len(src) {
		// 공백 건너뛰기
		for i < len(src) && isSpace(src[i]) {
			i++
		}
		if i >= len(src) {
			return
		}
		if src[i] == '@' {
			end := i
			for end < len(src) && src[end] != '{' && src[end] != ';' {
				end++
			}
			prelude := strings.TrimSpace(src[i:end])
			word, rest, _ := strings.Cut(prelude, " ")
			if end < len(src) && src[end] == ';' {
				if word == "@layer" {
					for _, name := range strings.Split(rest, ",") {
						s.layerRank(joinLayer(layerPrefix, strings.TrimSpace(name)))
					}
				}
				i = end + 1
				continue
			}
			if end >= len(src) {
				return
			}
			close := matchBrace(src, end)
			body := src[end+1 : close]
			if word == "@layer" {
				name := joinLayer(layerPrefix, strings.TrimSpace(rest))
				s.parseBlock(body, s.layerRank(name), name)
			}
			i = close + 1
			continue
		}
		open := strings.IndexByte(src[i:], '{')
		if open < 0 {
			return
		}
		open += i
		close := matchBrace(src, open)
		selText := strings.TrimSpace(src[i:open])
		decls := parseDecls(src[open+1 : close])
		if sels := parseSelectorList(selText); len(sels) > 0 {
			s.rules = append(s.rules, rule{selectors: sels, decls: decls, rank: rank, order: len(s.rules)})
		}
		i = close + 1
	}
}

func joinLayer(prefix, name string) string {
	if prefix == "" {
		return name
	}
	return prefix + "." + name
}

func matchBrace(src string, open int) int {
	depth := 0
	inStr := byte(0)
	for j := open; j < len(src); j++ {
		c := src[j]
		if inStr != 0 {
			if c == '\\' {
				j++
			} else if c == inStr {
				inStr = 0
			}
			continue
		}
		switch c {
		case '"', '\'':
			inStr = c
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return j
			}
		}
	}
	return len(src) - 1
}

func stripComments(src string) string {
	var b strings.Builder
	for {
		i := strings.Index(src, "/*")
		if i < 0 {
			b.WriteString(src)
			return b.String()
		}
		b.WriteString(src[:i])
		j := strings.Index(src[i+2:], "*/")
		if j < 0 {
			return b.String()
		}
		src = src[i+2+j+2:]
	}
}

func isSpace(c byte) bool { return c == ' ' || c == '\t' || c == '\n' || c == '\r' || c == '\f' }

// parseDecls는 `a: b; c: d` 목록을 읽는다 (괄호 · 문자열 안의 `;`는 구분자가 아니다).
func parseDecls(src string) []decl {
	var out []decl
	for _, part := range splitTop(src, ';') {
		name, value, ok := strings.Cut(part, ":")
		if !ok {
			continue
		}
		name = strings.ToLower(strings.TrimSpace(name))
		value = strings.TrimSpace(value)
		value = strings.TrimSpace(strings.TrimSuffix(value, "!important"))
		if name == "" || value == "" {
			continue
		}
		out = append(out, decl{name: name, value: value})
	}
	return out
}

// splitTop은 괄호와 문자열 밖의 sep에서 자른다.
func splitTop(src string, sep byte) []string {
	var out []string
	depth := 0
	inStr := byte(0)
	start := 0
	for j := 0; j < len(src); j++ {
		c := src[j]
		if inStr != 0 {
			if c == '\\' {
				j++
			} else if c == inStr {
				inStr = 0
			}
			continue
		}
		switch c {
		case '"', '\'':
			inStr = c
		case '(', '[':
			depth++
		case ')', ']':
			depth--
		case sep:
			if depth == 0 {
				out = append(out, src[start:j])
				start = j + 1
			}
		}
	}
	out = append(out, src[start:])
	return out
}

// ── 선택자 ──

type specificity [3]int

func (a specificity) less(b specificity) bool {
	for i := range a {
		if a[i] != b[i] {
			return a[i] < b[i]
		}
	}
	return false
}

func (a specificity) add(b specificity) specificity {
	return specificity{a[0] + b[0], a[1] + b[1], a[2] + b[2]}
}

// complexSel은 결합자로 이은 compound의 사슬이다 (오른쪽이 주어 요소).
type complexSel struct {
	parts  []compoundSel
	combs  []byte // parts[i]와 parts[i+1] 사이의 결합자 (' ' 또는 '>')
	pseudo string // 끝에 붙은 가상 요소 ("before" · "after" · "marker" …)
	spec   specificity
	bad    bool // 이 엔진이 모르는 것이 있어 절대 맞지 않는다
}

type attrSel struct {
	name, op, value string
}

type compoundSel struct {
	tag     string
	ids     []string
	classes []string
	attrs   []attrSel
	pseudos []pseudoSel
}

type pseudoSel struct {
	name string
	args []complexSel
	arg  string
}

func parseSelectorList(src string) []complexSel {
	var out []complexSel
	for _, part := range splitTop(src, ',') {
		sel := parseComplex(strings.TrimSpace(part))
		if !sel.bad {
			out = append(out, sel)
		}
	}
	return out
}

func parseComplex(src string) complexSel {
	var sel complexSel
	i := 0
	pendingComb := byte(0)
	for i < len(src) {
		c := src[i]
		if isSpace(c) {
			if len(sel.parts) > 0 && pendingComb == 0 {
				pendingComb = ' '
			}
			i++
			continue
		}
		if c == '>' || c == '+' || c == '~' {
			pendingComb = c
			i++
			continue
		}
		comp, pseudo, n, ok := parseCompound(src[i:])
		if !ok {
			sel.bad = true
			return sel
		}
		if len(sel.parts) > 0 {
			if pendingComb == 0 {
				pendingComb = ' '
			}
			if pendingComb != ' ' && pendingComb != '>' {
				sel.bad = true
				return sel
			}
			sel.combs = append(sel.combs, pendingComb)
		}
		pendingComb = 0
		sel.parts = append(sel.parts, comp)
		if pseudo != "" {
			sel.pseudo = pseudo
		}
		i += n
	}
	if len(sel.parts) == 0 {
		sel.bad = true
		return sel
	}
	for _, p := range sel.parts {
		sel.spec = sel.spec.add(p.spec())
	}
	if sel.pseudo != "" {
		sel.spec[2]++
	}
	return sel
}

func (c compoundSel) spec() specificity {
	s := specificity{len(c.ids), len(c.classes) + len(c.attrs), 0}
	if c.tag != "" && c.tag != "*" {
		s[2]++
	}
	for _, p := range c.pseudos {
		switch p.name {
		case "is", "not":
			var best specificity
			for _, a := range p.args {
				if best.less(a.spec) {
					best = a.spec
				}
			}
			s = s.add(best)
		default:
			s[1]++
		}
	}
	return s
}

func isIdentByte(c byte) bool {
	return c == '-' || c == '_' || c >= 0x80 || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9')
}

func readIdent(src string) string {
	j := 0
	for j < len(src) && (isIdentByte(src[j]) || src[j] == '\\') {
		if src[j] == '\\' {
			j++
		}
		j++
	}
	if j > len(src) {
		j = len(src)
	}
	return strings.ReplaceAll(src[:j], "\\", "")
}

// parseCompound는 compound 하나를 읽고 소비한 바이트 수를 낸다.
func parseCompound(src string) (comp compoundSel, pseudo string, n int, ok bool) {
	i := 0
	for i < len(src) {
		c := src[i]
		switch {
		case c == '*':
			comp.tag = "*"
			i++
		case c == '.':
			name := readIdent(src[i+1:])
			comp.classes = append(comp.classes, name)
			i += 1 + len(name)
		case c == '#':
			name := readIdent(src[i+1:])
			comp.ids = append(comp.ids, name)
			i += 1 + len(name)
		case c == '[':
			end := strings.IndexByte(src[i:], ']')
			if end < 0 {
				return comp, "", 0, false
			}
			comp.attrs = append(comp.attrs, parseAttrSel(src[i+1:i+end]))
			i += end + 1
		case c == ':':
			if strings.HasPrefix(src[i:], "::") {
				name := readIdent(src[i+2:])
				pseudo = strings.ToLower(name)
				i += 2 + len(name)
				continue
			}
			name := strings.ToLower(readIdent(src[i+1:]))
			i += 1 + len(name)
			p := pseudoSel{name: name}
			if i < len(src) && src[i] == '(' {
				depth := 0
				end := i
				for ; end < len(src); end++ {
					if src[end] == '(' {
						depth++
					} else if src[end] == ')' {
						depth--
						if depth == 0 {
							break
						}
					}
				}
				p.arg = strings.TrimSpace(src[i+1 : end])
				if name == "is" || name == "not" || name == "where" {
					p.args = parseSelectorList(p.arg)
				}
				i = end + 1
			}
			switch p.name {
			case "is", "not", "where", "lang", "root", "first-child", "last-child":
			case "before", "after", "marker":
				// 단일 콜론 가상 요소 표기
				pseudo = p.name
				continue
			default:
				return comp, "", 0, false
			}
			comp.pseudos = append(comp.pseudos, p)
		case isIdentByte(c):
			name := readIdent(src[i:])
			comp.tag = strings.ToLower(name)
			i += len(name)
		default:
			return comp, pseudo, i, i > 0
		}
	}
	return comp, pseudo, i, i > 0
}

func parseAttrSel(src string) attrSel {
	for _, op := range []string{"^=", "$=", "*=", "~=", "|=", "="} {
		if name, value, ok := strings.Cut(src, op); ok {
			value = strings.TrimSpace(value)
			value = strings.Trim(value, `"'`)
			return attrSel{name: strings.TrimSpace(name), op: op, value: value}
		}
	}
	return attrSel{name: strings.TrimSpace(src)}
}

// matches는 요소가 선택자의 주어인지 본다 (가상 요소는 호출 쪽이 따로 거른다).
func (sel complexSel) matches(n *node) bool {
	last := len(sel.parts) - 1
	if !sel.parts[last].matches(n) {
		return false
	}
	return sel.matchFrom(last-1, n)
}

func (sel complexSel) matchFrom(i int, subject *node) bool {
	if i < 0 {
		return true
	}
	comb := sel.combs[i]
	for anc := subject.parent; anc != nil; anc = anc.parent {
		if sel.parts[i].matches(anc) && sel.matchFrom(i-1, anc) {
			return true
		}
		if comb == '>' {
			return false
		}
	}
	return false
}

func (c compoundSel) matches(n *node) bool {
	if n.isText() {
		return false
	}
	if c.tag != "" && c.tag != "*" && c.tag != n.tag {
		return false
	}
	for _, id := range c.ids {
		if n.attrs["id"] != id {
			return false
		}
	}
	for _, class := range c.classes {
		if !n.hasClass(class) {
			return false
		}
	}
	for _, a := range c.attrs {
		v, ok := n.attrs[a.name]
		if !ok {
			return false
		}
		switch a.op {
		case "=":
			ok = v == a.value
		case "^=":
			ok = strings.HasPrefix(v, a.value)
		case "$=":
			ok = strings.HasSuffix(v, a.value)
		case "*=":
			ok = strings.Contains(v, a.value)
		case "~=":
			ok = false
			for _, f := range strings.Fields(v) {
				if f == a.value {
					ok = true
				}
			}
		case "|=":
			ok = v == a.value || strings.HasPrefix(v, a.value+"-")
		}
		if !ok {
			return false
		}
	}
	for _, p := range c.pseudos {
		switch p.name {
		case "is", "where":
			hit := false
			for _, a := range p.args {
				if a.pseudo == "" && a.matches(n) {
					hit = true
					break
				}
			}
			if !hit {
				return false
			}
		case "not":
			for _, a := range p.args {
				if a.pseudo == "" && a.matches(n) {
					return false
				}
			}
		case "root":
			if n.parent != nil {
				return false
			}
		case "lang":
			lang := ""
			for cur := n; cur != nil; cur = cur.parent {
				if v, ok := cur.attrs["lang"]; ok {
					lang = v
					break
				}
			}
			want := strings.ToLower(p.arg)
			lang = strings.ToLower(lang)
			if lang != want && !strings.HasPrefix(lang, want+"-") {
				return false
			}
		case "first-child":
			if i, _ := n.elementIndex(); i != 0 {
				return false
			}
		case "last-child":
			if i, count := n.elementIndex(); i != count-1 {
				return false
			}
		}
	}
	return true
}
