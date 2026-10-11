package quick

import (
	"bytes"
	"encoding/base64"
	"image"
	_ "image/jpeg" // 에셋 디코더
	_ "image/png"
	"math"
	"net/url"
	pathpkg "path"
	"strings"
)

// decodedImage는 에셋 하나다. SVG는 래스터가 아니라 문서 트리로 들고 그릴 때 래스터화한다.
type decodedImage struct {
	img  image.Image
	svg  *node
	w, h float64
}

// asset은 src(상대 경로 · data: URI)의 바이트다.
func (e *engine) asset(src string) ([]byte, bool) {
	if strings.HasPrefix(src, "data:") {
		meta, data, ok := strings.Cut(src[5:], ",")
		if !ok {
			return nil, false
		}
		if strings.HasSuffix(meta, ";base64") {
			raw, err := base64.StdEncoding.DecodeString(data)
			return raw, err == nil
		}
		text, err := url.PathUnescape(data)
		return []byte(text), err == nil
	}
	name := pathpkg.Clean(strings.TrimPrefix(src, "./"))
	if decoded, err := url.PathUnescape(name); err == nil {
		name = decoded
	}
	data, ok := e.assets.get(name)
	return data, ok
}

func (e *engine) image(src string) *decodedImage {
	if img, ok := e.images[src]; ok {
		return img
	}
	var out *decodedImage
	if data, ok := e.asset(src); ok {
		if svg := parseSVGDoc(data); svg != nil {
			w, h := svgNaturalSize(svg)
			out = &decodedImage{svg: svg, w: w, h: h}
		} else if img, _, err := image.Decode(bytes.NewReader(data)); err == nil {
			b := img.Bounds()
			out = &decodedImage{img: img, w: float64(b.Dx()), h: float64(b.Dy())}
		}
	}
	e.images[src] = out
	return out
}

// naturalSize는 대체 요소의 고유 크기다 (img: 에셋, svg: width/height 속성 · viewBox, 기본 300×150).
func (e *engine) naturalSize(b *box) (float64, float64) {
	switch b.replaced {
	case "img":
		if img := e.image(b.n.attrs["src"]); img != nil && img.w > 0 && img.h > 0 {
			return img.w, img.h
		}
		return 0, 0
	case "svg":
		return svgNaturalSize(b.n)
	}
	return 0, 0
}

func svgNaturalSize(n *node) (float64, float64) {
	w := svgLength(n.attrs["width"])
	h := svgLength(n.attrs["height"])
	vb := parseViewBox(n.attrs["viewBox"])
	switch {
	case w.definite() && h.definite():
		return w.px, h.px
	case w.definite() && vb != nil && vb[2] > 0:
		return w.px, w.px * vb[3] / vb[2]
	case h.definite() && vb != nil && vb[3] > 0:
		return h.px * vb[2] / vb[3], h.px
	case vb != nil:
		return vb[2], vb[3]
	}
	return 300, 150
}

func parseViewBox(v string) []float64 {
	fields := strings.FieldsFunc(v, func(r rune) bool { return r == ' ' || r == ',' })
	if len(fields) != 4 {
		return nil
	}
	out := make([]float64, 4)
	for i, f := range fields {
		n, ok := parseNumber(f)
		if !ok {
			return nil
		}
		out[i] = n
	}
	return out
}

// replacedSize는 대체 요소의 사용 크기다 — CSS 2.1 §10.4의 제약 표(비율을 지키며 min/max를
// 맞춘다). 퍼센트는 포함 블록이 정해졌을 때만 푼다.
func (e *engine) replacedSize(b *box, cbw, cbh float64, cbhKnown bool) (float64, float64) {
	nw, nh := e.naturalSize(b)
	ratio := b.ratio()
	if ratio == 0 && nw > 0 && nh > 0 {
		ratio = nw / nh
	}
	w, wok := b.specified("width", cbw, true)
	h, hok := b.specified("height", cbh, cbhKnown)
	switch {
	case wok && hok:
	case wok:
		if ratio > 0 {
			h = w / ratio
		} else {
			h = nh
		}
	case hok:
		if ratio > 0 {
			w = h * ratio
		} else {
			w = nw
		}
	default:
		w, h = nw, nh
		if w == 0 && h == 0 {
			w, h = 300, 150
		}
	}
	maxW, okMW := b.specified("max-width", cbw, true)
	maxH, okMH := b.specified("max-height", cbh, cbhKnown)
	if !okMW {
		maxW = math.Inf(1)
	}
	if !okMH {
		maxH = math.Inf(1)
	}
	minW, _ := b.specified("min-width", cbw, true)
	minH, _ := b.specified("min-height", cbh, cbhKnown)
	if wok || hok || ratio == 0 {
		return math.Max(minW, math.Min(maxW, w)), math.Max(minH, math.Min(maxH, h))
	}
	// 제약 표: 비율을 지키며 넘친 쪽에 맞춘다.
	switch {
	case w > maxW && h > maxH:
		if maxW/w <= maxH/h {
			w, h = maxW, math.Max(minH, maxW/ratio)
		} else {
			w, h = math.Max(minW, maxH*ratio), maxH
		}
	case w > maxW:
		w, h = maxW, math.Max(minH, maxW/ratio)
	case h > maxH:
		w, h = math.Max(minW, maxH*ratio), maxH
	case w < minW:
		w, h = minW, math.Min(maxH, minW/ratio)
	case h < minH:
		w, h = math.Min(maxW, minH*ratio), minH
	}
	return w, h
}

// svgLength는 SVG 속성 길이다 — 단위 없는 수는 사용자 단위(px)다.
func svgLength(v string) length {
	if f, ok := parseNumber(v); ok && !strings.ContainsAny(v, "%") {
		return length{px: f}
	}
	return lengthOr(v, 16, autoLength)
}
