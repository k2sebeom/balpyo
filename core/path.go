package core

import (
	"strconv"
	"strings"

	"github.com/k2sebeom/balpyo/soml"
)

// 이 파일은 경로 도형(`<shape type="path" d="…">`)의 값 층이다 — `d`를 두 타깃이 같은 원어로 옮길 수 있는
// 명령 목록으로 정규화한다.
//
// 문법은 SVG 경로의 닫힌 부분집합이다: 절대 좌표 명령 `M` · `L` · `H` · `V` · `C` · `Q` · `Z`. 상대 명령(소문자) ·
// 호(`A`) · 축약 곡선(`S` · `T`)은 진단이다 — OOXML `a:custGeom`에 1:1로 옮겨지는 명령만 연다 (호는 `arcTo`가
// 시작각 · 회전각으로 매개변수를 잡아 끝점 방식의 SVG `A`와 1:1이 아니다). `H` · `V`는 현재 점에서 `L`로 푼다.

// pathArgs는 명령마다 받는 수의 개수다.
var pathArgs = map[byte]int{'M': 2, 'L': 2, 'H': 1, 'V': 1, 'C': 6, 'Q': 4, 'Z': 0}

// pathData는 `d`를 해석한다. 진단은 attr의 값 구간을 가리킨다.
func (r *resolver) pathData(attr soml.Attr) ([]soml.PathCmd, bool) {
	if !r.hasValue(attr) {
		return nil, false
	}
	tokens := pathTokens(attr.Value)
	var cmds []soml.PathCmd
	var cx, cy float64
	started := false
	for i := 0; i < len(tokens); {
		op := tokens[i]
		if len(op) != 1 || !strings.ContainsAny(op, "MLHVCQZmlhvcqzASTast") {
			r.errorf(attr.ValueSpan, "path d: expected a command letter, got %q — commands are M L H V C Q Z with absolute coordinates", op)
			return nil, false
		}
		c := op[0]
		if strings.ContainsRune("mlhvcqzast", rune(c)) {
			r.errorf(attr.ValueSpan, "path d: relative command %q is not supported — write absolute coordinates with the uppercase command (%q)", op, strings.ToUpper(op))
			return nil, false
		}
		n, ok := pathArgs[c]
		if !ok {
			r.errorf(attr.ValueSpan, "path d: command %q is not supported (available commands: M, L, H, V, C, Q, Z) — arcs and shorthand curves have no one-to-one PPTX form", op)
			return nil, false
		}
		i++
		// 같은 명령의 수가 이어지면 그 명령을 되풀이한다 (SVG 규칙 — M 뒤의 수는 L이 된다).
		for first := true; first || (n > 0 && i < len(tokens) && isNumber(tokens[i])); first = false {
			args := make([]float64, n)
			for k := 0; k < n; k++ {
				if i >= len(tokens) || !isNumber(tokens[i]) {
					r.errorf(attr.ValueSpan, "path d: %c needs %d numbers", c, n)
					return nil, false
				}
				args[k], _ = strconv.ParseFloat(tokens[i], 64)
				i++
			}
			if !started && c != 'M' {
				r.errorf(attr.ValueSpan, "path d must start with M (got %c)", c)
				return nil, false
			}
			started = true
			switch c {
			case 'H':
				cmds = append(cmds, soml.PathCmd{Op: 'L', Pts: []float64{args[0], cy}})
				cx = args[0]
			case 'V':
				cmds = append(cmds, soml.PathCmd{Op: 'L', Pts: []float64{cx, args[0]}})
				cy = args[0]
			case 'Z':
				cmds = append(cmds, soml.PathCmd{Op: 'Z'})
			default:
				op := c
				if c == 'M' && !first {
					op = 'L'
				}
				cmds = append(cmds, soml.PathCmd{Op: op, Pts: args})
				cx, cy = args[n-2], args[n-1]
			}
		}
	}
	if len(cmds) == 0 {
		r.errorf(attr.ValueSpan, "path d is empty — e.g. d=%q", "M0,100 L50,0 L100,100 Z")
		return nil, false
	}
	return cmds, true
}

// pathTokens는 경로 원문을 명령 글자와 수로 나눈다 — 쉼표 · 공백이 구분자이고, 글자는 붙어 써도 된다 (`M0,0L10,10`).
func pathTokens(raw string) []string {
	var tokens []string
	var cur strings.Builder
	flush := func() {
		if cur.Len() > 0 {
			tokens = append(tokens, cur.String())
			cur.Reset()
		}
	}
	for _, ch := range raw {
		switch {
		case ch == ',' || ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r':
			flush()
		case (ch >= 'A' && ch <= 'Z') || (ch >= 'a' && ch <= 'z'):
			flush()
			tokens = append(tokens, string(ch))
		case ch == '-' && cur.Len() > 0 && !strings.HasSuffix(cur.String(), "e"):
			// 부호는 새 수를 연다 (`10-5` = 10, -5).
			flush()
			cur.WriteRune(ch)
		default:
			cur.WriteRune(ch)
		}
	}
	flush()
	return tokens
}

func isNumber(s string) bool {
	_, err := strconv.ParseFloat(s, 64)
	return err == nil
}
