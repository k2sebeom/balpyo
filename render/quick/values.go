package quick

import (
	"math"
	"strconv"
	"strings"
)

// length는 계산값 길이다 — px와 퍼센트의 일차 결합 (`calc(36px + 10%)`도 이 꼴이 된다).
// 퍼센트의 기준은 쓰는 자리가 정한다.
type length struct {
	px, pct float64
	auto    bool
}

func (l length) resolve(base float64) float64 { return l.px + l.pct*base/100 }

// definite는 기준 없이 값이 정해지는지다.
func (l length) definite() bool { return !l.auto && l.pct == 0 }

var autoLength = length{auto: true}

// parseLength는 길이 · 퍼센트 · calc()를 읽는다. auto · none · normal · fit-content 같은
// 키워드는 auto다.
func parseLength(v string, fontSize float64) (length, bool) {
	v = strings.TrimSpace(v)
	switch v {
	case "", "auto", "none", "normal", "fit-content", "max-content", "min-content", "initial":
		return autoLength, v != ""
	}
	e := calcParser{src: v, fontSize: fontSize}
	val, ok := e.parseSum()
	if !ok || !val.isLen {
		if ok && !val.isLen && val.scalar == 0 {
			return length{}, true // 단위 없는 0
		}
		return length{}, false
	}
	return length{px: val.px, pct: val.pct}, true
}

// lengthOr는 읽지 못하면 def다.
func lengthOr(v string, fontSize float64, def length) length {
	if l, ok := parseLength(v, fontSize); ok {
		return l
	}
	return def
}

// calcValue는 calc 식의 값이다: 길이(px + pct)이거나 무차원 수다.
type calcValue struct {
	px, pct float64
	scalar  float64
	isLen   bool
}

type calcParser struct {
	src      string
	pos      int
	fontSize float64
}

func (p *calcParser) skip() {
	for p.pos < len(p.src) && isSpace(p.src[p.pos]) {
		p.pos++
	}
}

func (p *calcParser) parseSum() (calcValue, bool) {
	left, ok := p.parseProduct()
	if !ok {
		return left, false
	}
	for {
		p.skip()
		if p.pos >= len(p.src) {
			return left, true
		}
		op := p.src[p.pos]
		if op != '+' && op != '-' {
			return left, true
		}
		p.pos++
		right, ok := p.parseProduct()
		if !ok {
			return left, false
		}
		sign := 1.0
		if op == '-' {
			sign = -1
		}
		if left.isLen || right.isLen {
			left = calcValue{px: left.px + sign*right.px, pct: left.pct + sign*right.pct, isLen: true}
		} else {
			left.scalar += sign * right.scalar
		}
	}
}

func (p *calcParser) parseProduct() (calcValue, bool) {
	left, ok := p.parseAtom()
	if !ok {
		return left, false
	}
	for {
		p.skip()
		if p.pos >= len(p.src) {
			return left, true
		}
		op := p.src[p.pos]
		if op != '*' && op != '/' {
			return left, true
		}
		p.pos++
		right, ok := p.parseAtom()
		if !ok {
			return left, false
		}
		switch {
		case op == '*' && left.isLen && !right.isLen:
			left.px *= right.scalar
			left.pct *= right.scalar
		case op == '*' && !left.isLen && right.isLen:
			right.px *= left.scalar
			right.pct *= left.scalar
			left = right
		case op == '*':
			left.scalar *= right.scalar
		case op == '/' && !right.isLen && right.scalar != 0:
			if left.isLen {
				left.px /= right.scalar
				left.pct /= right.scalar
			} else {
				left.scalar /= right.scalar
			}
		default:
			return left, false
		}
	}
}

func (p *calcParser) parseAtom() (calcValue, bool) {
	p.skip()
	if p.pos >= len(p.src) {
		return calcValue{}, false
	}
	rest := p.src[p.pos:]
	for _, fn := range []string{"calc(", "("} {
		if strings.HasPrefix(rest, fn) {
			p.pos += len(fn)
			v, ok := p.parseSum()
			p.skip()
			if !ok || p.pos >= len(p.src) || p.src[p.pos] != ')' {
				return v, false
			}
			p.pos++
			return v, true
		}
	}
	for _, fn := range []string{"min(", "max("} {
		if strings.HasPrefix(rest, fn) {
			// min/max는 기준을 모르면 고를 수 없다 — 첫 인자를 쓴다 (웹 렌더러 출력에는 없다).
			end := closingParen(p.src, p.pos+len(fn)-1)
			args := splitTop(p.src[p.pos+len(fn):end], ',')
			sub := calcParser{src: args[0], fontSize: p.fontSize}
			v, ok := sub.parseSum()
			p.pos = end + 1
			return v, ok
		}
	}
	// 수
	j := p.pos
	if j < len(p.src) && (p.src[j] == '-' || p.src[j] == '+') {
		j++
	}
	for j < len(p.src) && (p.src[j] >= '0' && p.src[j] <= '9' || p.src[j] == '.') {
		j++
	}
	if j < len(p.src) && (p.src[j] == 'e' || p.src[j] == 'E') && j+1 < len(p.src) && (p.src[j+1] >= '0' && p.src[j+1] <= '9' || p.src[j+1] == '-') {
		j += 2
		for j < len(p.src) && p.src[j] >= '0' && p.src[j] <= '9' {
			j++
		}
	}
	num, err := strconv.ParseFloat(p.src[p.pos:j], 64)
	if err != nil {
		return calcValue{}, false
	}
	k := j
	for k < len(p.src) && (p.src[k] >= 'a' && p.src[k] <= 'z' || p.src[k] == '%') {
		k++
	}
	unit := p.src[j:k]
	p.pos = k
	switch unit {
	case "":
		return calcValue{scalar: num}, true
	case "px":
		return calcValue{px: num, isLen: true}, true
	case "%":
		return calcValue{pct: num, isLen: true}, true
	case "em":
		return calcValue{px: num * p.fontSize, isLen: true}, true
	case "rem":
		return calcValue{px: num * 16, isLen: true}, true
	case "pt":
		return calcValue{px: num * 4 / 3, isLen: true}, true
	case "deg":
		return calcValue{scalar: num}, true
	case "rad":
		return calcValue{scalar: num * 180 / math.Pi}, true
	case "turn":
		return calcValue{scalar: num * 360}, true
	case "ms":
		return calcValue{scalar: num}, true
	case "s":
		return calcValue{scalar: num * 1000}, true
	}
	return calcValue{}, false
}

// parseNumber는 무차원 수를 읽는다 (calc() 포함).
func parseNumber(v string) (float64, bool) {
	v = strings.TrimSpace(v)
	if f, err := strconv.ParseFloat(v, 64); err == nil {
		return f, true
	}
	e := calcParser{src: v}
	val, ok := e.parseSum()
	if !ok || val.isLen {
		return 0, false
	}
	return val.scalar, true
}

func numberOr(v string, def float64) float64 {
	if f, ok := parseNumber(v); ok {
		return f
	}
	return def
}

// ── 색 ──

// rgba는 직선(비-premultiplied) 색이다 (0..1).
type rgba struct{ r, g, b, a float64 }

var transparent = rgba{}

var namedColors = map[string]rgba{
	"black": {0, 0, 0, 1}, "white": {1, 1, 1, 1}, "red": {1, 0, 0, 1}, "green": {0, 128.0 / 255, 0, 1},
	"blue": {0, 0, 1, 1}, "yellow": {1, 1, 0, 1}, "gray": {128.0 / 255, 128.0 / 255, 128.0 / 255, 1},
	"grey": {128.0 / 255, 128.0 / 255, 128.0 / 255, 1}, "orange": {1, 165.0 / 255, 0, 1},
	"transparent": transparent,
}

// parseColor는 색을 읽는다. current는 currentColor의 값이다.
func parseColor(v string, current rgba) (rgba, bool) {
	v = strings.TrimSpace(v)
	lower := strings.ToLower(v)
	if lower == "currentcolor" {
		return current, true
	}
	if c, ok := namedColors[lower]; ok {
		return c, true
	}
	if strings.HasPrefix(v, "#") {
		return parseHex(v[1:])
	}
	open := strings.IndexByte(v, '(')
	if open < 0 || !strings.HasSuffix(v, ")") {
		return rgba{}, false
	}
	fn := strings.ToLower(v[:open])
	inner := v[open+1 : len(v)-1]
	switch fn {
	case "rgb", "rgba":
		args, alpha := colorArgs(inner)
		if len(args) < 3 {
			return rgba{}, false
		}
		ch := func(s string) float64 {
			if strings.HasSuffix(s, "%") {
				f, _ := strconv.ParseFloat(strings.TrimSuffix(s, "%"), 64)
				return f / 100
			}
			f, _ := strconv.ParseFloat(s, 64)
			return f / 255
		}
		return rgba{clamp01(ch(args[0])), clamp01(ch(args[1])), clamp01(ch(args[2])), alpha}, true
	case "hsl", "hsla":
		if strings.HasPrefix(strings.TrimSpace(inner), "from ") {
			return relativeHSL(strings.TrimSpace(inner)[5:], current)
		}
		args, alpha := colorArgs(inner)
		if len(args) < 3 {
			return rgba{}, false
		}
		h, _ := strconv.ParseFloat(strings.TrimSuffix(args[0], "deg"), 64)
		s, _ := strconv.ParseFloat(strings.TrimSuffix(args[1], "%"), 64)
		l, _ := strconv.ParseFloat(strings.TrimSuffix(args[2], "%"), 64)
		c := hslToRGB(h, s, l)
		c.a = alpha
		return c, true
	case "color-mix":
		return colorMix(inner, current)
	}
	return rgba{}, false
}

func clamp01(f float64) float64 { return math.Max(0, math.Min(1, f)) }

// colorArgs는 `r g b / a` 와 `r, g, b, a` 두 문법을 같이 읽는다.
func colorArgs(inner string) ([]string, float64) {
	alpha := 1.0
	body := inner
	if main, a, ok := strings.Cut(inner, "/"); ok {
		body = main
		alpha = parseAlpha(a)
	}
	var args []string
	for _, part := range strings.FieldsFunc(body, func(r rune) bool { return r == ',' || r == ' ' }) {
		if part != "" {
			args = append(args, part)
		}
	}
	if len(args) == 4 {
		alpha = parseAlpha(args[3])
		args = args[:3]
	}
	return args, alpha
}

func parseAlpha(s string) float64 {
	s = strings.TrimSpace(s)
	if strings.HasSuffix(s, "%") {
		f, _ := strconv.ParseFloat(strings.TrimSuffix(s, "%"), 64)
		return clamp01(f / 100)
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 1
	}
	return clamp01(f)
}

func parseHex(h string) (rgba, bool) {
	expand := func(s string) string {
		var b strings.Builder
		for _, c := range s {
			b.WriteRune(c)
			b.WriteRune(c)
		}
		return b.String()
	}
	switch len(h) {
	case 3, 4:
		h = expand(h)
	}
	if len(h) != 6 && len(h) != 8 {
		return rgba{}, false
	}
	n, err := strconv.ParseUint(h, 16, 64)
	if err != nil {
		return rgba{}, false
	}
	if len(h) == 6 {
		n = n<<8 | 0xff
	}
	return rgba{
		float64(n>>24&0xff) / 255, float64(n>>16&0xff) / 255, float64(n>>8&0xff) / 255, float64(n&0xff) / 255,
	}, true
}

func hslToRGB(h, s, l float64) rgba {
	h = math.Mod(math.Mod(h, 360)+360, 360) / 360
	s, l = clamp01(s/100), clamp01(l/100)
	if s == 0 {
		return rgba{l, l, l, 1}
	}
	q := l * (1 + s)
	if l >= 0.5 {
		q = l + s - l*s
	}
	p := 2*l - q
	hue := func(t float64) float64 {
		if t < 0 {
			t++
		}
		if t > 1 {
			t--
		}
		switch {
		case t < 1.0/6:
			return p + (q-p)*6*t
		case t < 0.5:
			return q
		case t < 2.0/3:
			return p + (q-p)*(2.0/3-t)*6
		}
		return p
	}
	return rgba{hue(h + 1.0/3), hue(h), hue(h - 1.0/3), 1}
}

func rgbToHSL(c rgba) (h, s, l float64) {
	maxC := math.Max(c.r, math.Max(c.g, c.b))
	minC := math.Min(c.r, math.Min(c.g, c.b))
	l = (maxC + minC) / 2
	if maxC == minC {
		return 0, 0, l * 100
	}
	d := maxC - minC
	if l > 0.5 {
		s = d / (2 - maxC - minC)
	} else {
		s = d / (maxC + minC)
	}
	switch maxC {
	case c.r:
		h = (c.g - c.b) / d
		if c.g < c.b {
			h += 6
		}
	case c.g:
		h = (c.b-c.r)/d + 2
	default:
		h = (c.r-c.g)/d + 4
	}
	return h * 60, s * 100, l * 100
}

// relativeHSL은 `hsl(from <색> h s calc(l * a + b) [/ alpha])`를 읽는다 — 웹 렌더러가 테마 색의
// 명도 변형(shade · tint)에 쓰는 꼴이다.
func relativeHSL(rest string, current rgba) (rgba, bool) {
	parts := splitValues(rest)
	if len(parts) < 4 {
		return rgba{}, false
	}
	base, ok := parseColor(parts[0], current)
	if !ok {
		return rgba{}, false
	}
	h, s, l := rgbToHSL(base)
	vars := map[string]float64{"h": h, "s": s, "l": l, "alpha": base.a}
	eval := func(expr string) float64 {
		for name, val := range vars {
			if expr == name {
				return val
			}
		}
		// 식 안의 채널 이름을 수로 바꿔 calc로 계산한다.
		replaced := replaceChannels(expr, vars)
		f, _ := parseNumber(replaced)
		return f
	}
	nh, ns, nl := eval(parts[1]), eval(parts[2]), eval(parts[3])
	out := hslToRGB(nh, ns, nl)
	out.a = base.a
	if len(parts) >= 6 && parts[4] == "/" {
		out.a = clamp01(eval(parts[5]))
	}
	return out, true
}

func replaceChannels(expr string, vars map[string]float64) string {
	var b strings.Builder
	for i := 0; i < len(expr); {
		c := expr[i]
		if c >= 'a' && c <= 'z' {
			j := i
			for j < len(expr) && (expr[j] >= 'a' && expr[j] <= 'z') {
				j++
			}
			word := expr[i:j]
			if val, ok := vars[word]; ok {
				b.WriteString(strconv.FormatFloat(val, 'f', -1, 64))
			} else {
				b.WriteString(word)
			}
			i = j
			continue
		}
		b.WriteByte(c)
		i++
	}
	return b.String()
}

// colorMix는 `color-mix(in srgb, A p%, B)`를 읽는다.
func colorMix(inner string, current rgba) (rgba, bool) {
	args := splitTop(inner, ',')
	if len(args) != 3 {
		return rgba{}, false
	}
	readStop := func(s string) (rgba, float64, bool) {
		parts := splitValues(strings.TrimSpace(s))
		pct := -1.0
		colorText := parts[0]
		if len(parts) > 1 && strings.HasSuffix(parts[len(parts)-1], "%") {
			f, _ := strconv.ParseFloat(strings.TrimSuffix(parts[len(parts)-1], "%"), 64)
			pct = f
			colorText = strings.Join(parts[:len(parts)-1], " ")
		}
		c, ok := parseColor(colorText, current)
		return c, pct, ok
	}
	a, pa, ok1 := readStop(args[1])
	b, pb, ok2 := readStop(args[2])
	if !ok1 || !ok2 {
		return rgba{}, false
	}
	switch {
	case pa < 0 && pb < 0:
		pa, pb = 50, 50
	case pa < 0:
		pa = 100 - pb
	case pb < 0:
		pb = 100 - pa
	}
	wa, wb := pa/(pa+pb), pb/(pa+pb)
	// premultiplied 보간 (CSS Color 5)
	alpha := a.a*wa + b.a*wb
	if alpha == 0 {
		return rgba{}, true
	}
	mix := func(x, y float64) float64 { return (x*a.a*wa + y*b.a*wb) / alpha }
	out := rgba{mix(a.r, b.r), mix(a.g, b.g), mix(a.b, b.b), alpha}
	if sum := pa + pb; sum < 100 {
		out.a *= sum / 100
	}
	return out, true
}

// parseAngle은 deg 단위 각을 읽는다 (rotate() · 그라디언트 방향).
func parseAngle(v string) (float64, bool) {
	v = strings.TrimSpace(v)
	e := calcParser{src: v}
	val, ok := e.parseSum()
	if !ok || val.isLen {
		return 0, false
	}
	return val.scalar, true
}
