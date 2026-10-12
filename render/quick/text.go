package quick

import (
	"io"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"unicode"

	"github.com/go-text/typesetting/di"
	"github.com/go-text/typesetting/font"
	"github.com/go-text/typesetting/fontscan"
	"github.com/go-text/typesetting/language"
	"github.com/go-text/typesetting/shaping"
	"golang.org/x/image/math/fixed"
)

// fontStyle은 글꼴 선택과 셰이핑에 드는 계산값이다.
type fontStyle struct {
	families string // 쉼표로 이은 원문 (캐시 키)
	size     float64
	weight   int
	italic   bool
	spacing  float64 // letter-spacing (px)
}

func fontStyleOf(st *style) fontStyle {
	fs := fontStyle{
		families: st.get("font-family"),
		size:     st.fontSize,
		weight:   parseWeight(st.get("font-weight")),
		italic:   st.get("font-style") == "italic" || st.get("font-style") == "oblique",
	}
	if ls := st.get("letter-spacing"); ls != "" && ls != "normal" {
		if l, ok := parseLength(ls, st.fontSize); ok && !l.auto {
			fs.spacing = l.px
		}
	}
	return fs
}

func parseWeight(v string) int {
	switch v {
	case "", "normal":
		return 400
	case "bold", "bolder":
		return 700
	case "lighter":
		return 300
	}
	if n, err := strconv.Atoi(v); err == nil {
		return n
	}
	return 400
}

func familyList(v string) []string {
	var out []string
	for _, part := range splitTop(v, ',') {
		name := strings.Trim(strings.TrimSpace(part), `"'`)
		if name != "" {
			out = append(out, name)
		}
	}
	if len(out) == 0 {
		out = []string{"sans-serif"}
	}
	return out
}

// fontKit은 시스템 글꼴 목록과 셰이퍼, 그리고 셰이핑 · 윤곽 캐시다. 슬라이드 사이에 공유한다.
type fontKit struct {
	mu      sync.Mutex
	fm      *fontscan.FontMap
	shaper  shaping.HarfbuzzShaper
	faces   map[faceKey]*font.Face
	shaped  map[shapeKey]shapedText
	metrics map[metricKey]faceMetrics
	outline map[glyphKey]font.GlyphOutline
	query   string
}

type faceKey struct {
	families string
	weight   int
	italic   bool
	r        rune
}

type shapeKey struct {
	fs   fontStyle
	text string
}

type metricKey struct {
	face *font.Face
}

type glyphKey struct {
	face *font.Face
	gid  font.GID
}

// faceMetrics는 글꼴 단위의 메트릭이다 (em 비율).
type faceMetrics struct {
	ascent, descent float64 // 둘 다 양수, em 비율
	underlinePos    float64
	underlineThick  float64
}

var (
	kitOnce sync.Once
	kitErr  error
	kit     *fontKit
)

// sharedFonts는 프로세스 하나에서 한 번 시스템 글꼴을 읽는다 (인덱스는 사용자 캐시에 남는다).
func sharedFonts() (*fontKit, error) {
	kitOnce.Do(func() {
		fm := fontscan.NewFontMap(log.New(io.Discard, "", 0))
		dir, err := os.UserCacheDir()
		if err != nil {
			dir = os.TempDir()
		}
		if err := fm.UseSystemFonts(filepath.Join(dir, "balpyo", "fonts")); err != nil {
			kitErr = err
			return
		}
		kit = &fontKit{
			fm:      fm,
			faces:   map[faceKey]*font.Face{},
			shaped:  map[shapeKey]shapedText{},
			metrics: map[metricKey]faceMetrics{},
			outline: map[glyphKey]font.GlyphOutline{},
		}
	})
	return kit, kitErr
}

func (k *fontKit) setQuery(fs fontStyle) {
	key := fs.families + "|" + strconv.Itoa(fs.weight) + "|" + strconv.FormatBool(fs.italic)
	if k.query == key {
		return
	}
	k.query = key
	aspect := font.Aspect{Weight: font.Weight(fs.weight)}
	if fs.italic {
		aspect.Style = font.StyleItalic
	}
	k.fm.SetQuery(fontscan.Query{Families: familyList(fs.families), Aspect: aspect})
}

// face는 문자 하나를 그릴 글꼴이다 (가족 목록 → 시스템 폴백).
func (k *fontKit) face(fs fontStyle, r rune) *font.Face {
	key := faceKey{fs.families, fs.weight, fs.italic, r}
	if f, ok := k.faces[key]; ok {
		return f
	}
	k.setQuery(fs)
	// 문자 체계를 알려야 가족 목록에 없는 문자(기호 · 이모지)를 그 체계의 글꼴로 폴백한다.
	k.fm.SetScript(language.LookupScript(r))
	f := k.fm.ResolveFace(r)
	k.faces[key] = f
	return f
}

// primary는 줄 높이의 기준이 되는 첫 글꼴이다 (CSS의 first available font).
func (k *fontKit) primary(fs fontStyle) *font.Face { return k.face(fs, ' ') }

func (k *fontKit) faceMetrics(f *font.Face) faceMetrics {
	if m, ok := k.metrics[metricKey{f}]; ok {
		return m
	}
	m := faceMetrics{ascent: 0.9, descent: 0.25, underlinePos: -0.1, underlineThick: 0.05}
	if f != nil {
		upem := float64(f.Upem())
		if ext, ok := f.FontHExtents(); ok && upem > 0 {
			m.ascent = float64(ext.Ascender) / upem
			m.descent = -float64(ext.Descender) / upem
		}
		if upem > 0 {
			if pos := f.LineMetric(font.UnderlinePosition); pos != 0 {
				m.underlinePos = float64(pos) / upem
			}
			if th := f.LineMetric(font.UnderlineThickness); th != 0 {
				m.underlineThick = float64(th) / upem
			}
		}
	}
	k.metrics[metricKey{f}] = m
	return m
}

// glyph는 글리프 하나의 위치다 (조각 원점 기준 px, y는 기준선).
type glyph struct {
	face *font.Face
	gid  font.GID
	x, y float64
	size float64
	// slant는 합성 기울임이다 — italic을 청했는데 글꼴에 italic 면이 없으면 브라우저처럼 기울여 그린다.
	slant bool
	// pen은 글리프의 펜 위치(오프셋을 뺀 x), adv는 진행 폭(자간 포함)이다. cluster는 글리프가 나온
	// 첫 문자의 셰이핑한 문자열 안 위치다 — 긴 문자열을 한 번 셰이핑하고 문자 구간으로 자를 때 쓴다.
	pen, adv float64
	cluster  int
}

// shapedText는 셰이핑한 조각이다.
type shapedText struct {
	width  float64
	glyphs []glyph
}

// shapeScale은 셰이핑할 때 글꼴 크기에 곱하는 배율이다. 셰이퍼는 크기와 위치를 1/64px 고정
// 소수로 다뤄 글리프마다 진행 폭이 1/64px 단위로 반올림되는데, 브라우저는 1/65536px 단위로 재고
// 나서 더한다 — 글자가 쌓일수록 내장 엔진의 줄이 짧아져 브라우저가 넘기는 줄을 한 줄에 담는다.
// 1024배로 셰이핑하고 결과를 나누면 같은 정밀도가 된다.
const shapeScale = 1024

// shape는 줄바꿈 기회 사이의 조각 하나를 셰이핑한다. 글꼴 폴백은 문자 단위로 나눈 run이다.
func (k *fontKit) shape(fs fontStyle, text string) shapedText {
	key := shapeKey{fs, text}
	if s, ok := k.shaped[key]; ok {
		return s
	}
	runes := []rune(text)
	var out shapedText
	x := 0.0
	for start := 0; start < len(runes); {
		f := k.face(fs, runes[start])
		end := start + 1
		for end < len(runes) {
			r := runes[end]
			// 결합 문자 · 공백은 앞 run의 글꼴을 따른다.
			if unicode.Is(unicode.Mn, r) || unicode.Is(unicode.Variation_Selector, r) {
				end++
				continue
			}
			if k.face(fs, r) != f {
				break
			}
			end++
		}
		if f == nil {
			start = end
			continue
		}
		in := shaping.Input{
			Text:      runes,
			RunStart:  start,
			RunEnd:    end,
			Direction: di.DirectionLTR,
			Face:      f,
			Size:      fixed.Int26_6(fs.size * 64 * shapeScale),
			Script:    language.LookupScript(runes[start]),
			Language:  language.DefaultLanguage(),
		}
		res := k.shaper.Shape(in)
		slant := false
		if fs.italic {
			_, aspect := k.fm.FontMetadata(f.Font)
			slant = aspect.Style != font.StyleItalic
		}
		for _, g := range res.Glyphs {
			adv := fixedToF(g.XAdvance) / shapeScale
			if fs.spacing != 0 {
				adv += fs.spacing * float64(max(g.RuneCount, 1))
			}
			out.glyphs = append(out.glyphs, glyph{
				face: f, gid: g.GlyphID, size: fs.size, slant: slant,
				x: x + fixedToF(g.XOffset)/shapeScale, y: -fixedToF(g.YOffset) / shapeScale,
				pen: x, adv: adv, cluster: g.ClusterIndex,
			})
			x += adv
		}
		start = end
	}
	out.width = x
	k.shaped[key] = out
	return out
}

// slice는 셰이핑 결과에서 문자 구간 [from, to)에서 나온 글리프만 떼어 그 구간 원점 기준으로 옮긴다.
// 구간 경계 너머의 이웃과 맺은 커닝 · 문맥은 그대로 남는다 — 브라우저가 텍스트를 줄바꿈 기회마다
// 따로 셰이핑하지 않고 한 덩어리로 셰이핑한 뒤 자르기 때문이다.
func (s shapedText) slice(from, to int) shapedText {
	var out shapedText
	origin := 0.0
	for _, g := range s.glyphs {
		if g.cluster < from || g.cluster >= to {
			continue
		}
		if len(out.glyphs) == 0 {
			origin = g.pen
		}
		g.x -= origin
		g.pen -= origin
		out.glyphs = append(out.glyphs, g)
		out.width += g.adv
	}
	return out
}

func (k *fontKit) glyphOutline(f *font.Face, gid font.GID) (font.GlyphOutline, bool) {
	key := glyphKey{f, gid}
	if o, ok := k.outline[key]; ok {
		return o, true
	}
	data := f.GlyphData(gid)
	o, ok := data.(font.GlyphOutline)
	if !ok {
		return font.GlyphOutline{}, false
	}
	k.outline[key] = o
	return o, true
}

func fixedToF(v fixed.Int26_6) float64 { return float64(v) / 64 }

// applyTransform은 text-transform을 적용한다.
func applyTransform(text, transform string) string {
	switch transform {
	case "uppercase":
		return strings.ToUpper(text)
	case "lowercase":
		return strings.ToLower(text)
	}
	return text
}
