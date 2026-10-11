package core

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"math"
	"strconv"
	"strings"

	"github.com/k2sebeom/balpyo/soml"
)

// decodeSVGAsset은 SVG 파일 하나를 에셋으로 판정한다 — 실패하면 진단 문구를 돌려준다.
//
// 판정은 셋이다: XML이고 뿌리가 `svg`다, 고유 크기를 안다(`width` · `height` 또는 `viewBox`), **자기 안에서
// 끝난다**(스크립트 · 이벤트 속성 · `foreignObject` · 바깥 참조가 없다). 마지막 조건이 PPTX의 대체 그림을 웹과
// 같은 그림으로 만든다 — 바깥 파일 · 글꼴 · 스크립트는 대체 그림을 그린 때와 발표하는 때에 다를 수 있다.
func decodeSVGAsset(key string, data []byte) (soml.Asset, string) {
	dec := xml.NewDecoder(bytes.NewReader(data))
	dec.Strict = true
	var w, h float64
	root := true
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return soml.Asset{}, fmt.Sprintf("the content of %s is not SVG — it does not parse as XML (%v)", key, err)
		}
		switch t := tok.(type) {
		case xml.StartElement:
			if root {
				if t.Name.Local != "svg" {
					return soml.Asset{}, fmt.Sprintf("the content of %s is not SVG — the root element is <%s>, not <svg>", key, t.Name.Local)
				}
				var msg string
				if w, h, msg = svgSize(key, t); msg != "" {
					return soml.Asset{}, msg
				}
				root = false
			}
			if msg := svgClosed(key, t); msg != "" {
				return soml.Asset{}, msg
			}
		case xml.CharData:
			// `<style>` 안의 바깥 참조 — 글꼴 · 그림을 불러오는 CSS다.
			text := string(t)
			if strings.Contains(text, "@import") {
				return soml.Asset{}, fmt.Sprintf("%s imports an external stylesheet (@import) — an SVG image must be self-contained", key)
			}
			if ref := externalCSSURL(text); ref != "" {
				return soml.Asset{}, fmt.Sprintf("%s refers to %q outside the file — an SVG image must be self-contained (embed it or use an #id reference)", key, ref)
			}
		}
	}
	if root {
		return soml.Asset{}, fmt.Sprintf("the content of %s is not SVG — there is no <svg> element", key)
	}
	return soml.Asset{
		Key:    key,
		Format: soml.FormatSVG,
		Width:  int(math.Round(w)),
		Height: int(math.Round(h)),
		Bytes:  data,
	}, ""
}

// svgSize는 뿌리 `<svg>`의 고유 크기다 — `width` · `height`(px 또는 단위 없음)이고, 한쪽만 있으면 `viewBox`의
// 비율로 다른 쪽을 채운다.
//
// `viewBox`만 있는 SVG는 받지 않는다: 브라우저는 그런 그림에 고유 크기를 주지 않아(`<img>`가 부모 폭을 따르거나
// 0이 된다) 다른 그림처럼 "크기 속성이 없으면 고유 크기"가 성립하지 않는다. 고유 크기가 파일에 적혀 있어야 브라우저가
// 보는 크기와 이 판정이 같다.
func svgSize(key string, root xml.StartElement) (float64, float64, string) {
	var width, height, viewBox string
	for _, a := range root.Attr {
		switch a.Name.Local {
		case "width":
			width = a.Value
		case "height":
			height = a.Value
		case "viewBox":
			viewBox = a.Value
		}
	}
	var vw, vh float64
	if viewBox != "" {
		f := strings.FieldsFunc(viewBox, func(r rune) bool { return r == ',' || r == ' ' || r == '\t' || r == '\n' })
		if len(f) == 4 {
			vw, _ = strconv.ParseFloat(f[2], 64)
			vh, _ = strconv.ParseFloat(f[3], 64)
		}
	}
	px := func(name, v string) (float64, string) {
		if v == "" {
			return 0, ""
		}
		n, err := strconv.ParseFloat(strings.TrimSuffix(strings.TrimSpace(v), "px"), 64)
		if err != nil || n <= 0 {
			return 0, fmt.Sprintf("%s has %s=%q — an SVG image gives its size in px (or a unitless number)", key, name, v)
		}
		return n, ""
	}
	w, msg := px("width", width)
	if msg != "" {
		return 0, 0, msg
	}
	h, msg := px("height", height)
	if msg != "" {
		return 0, 0, msg
	}
	switch {
	case w > 0 && h > 0:
	case w > 0 && vw > 0 && vh > 0:
		h = w * vh / vw
	case h > 0 && vw > 0 && vh > 0:
		w = h * vw / vh
	case vw > 0 && vh > 0:
		return 0, 0, fmt.Sprintf("%s has only a viewBox — give the <svg> width and height in px (e.g. width=\"%s\" height=\"%s\"); browsers give a viewBox-only SVG no intrinsic size", key, formatNum(vw), formatNum(vh))
	default:
		return 0, 0, fmt.Sprintf("%s has no intrinsic size — give the <svg> width and height in px", key)
	}
	if math.Round(w) < 1 || math.Round(h) < 1 {
		return 0, 0, fmt.Sprintf("%s has an intrinsic size of %s×%s — nothing to draw", key, formatNum(w), formatNum(h))
	}
	return w, h, ""
}

// svgClosed는 요소 하나가 자기 안에서 끝나는지 본다 — 스크립트 · 이벤트 속성 · HTML 섬 · 바깥 참조가 없어야 한다.
func svgClosed(key string, el xml.StartElement) string {
	switch el.Name.Local {
	case "script":
		return fmt.Sprintf("%s contains <script> — an SVG image cannot run code", key)
	case "foreignObject":
		return fmt.Sprintf("%s contains <foreignObject> — HTML inside an SVG image has no PPTX equivalent", key)
	}
	for _, a := range el.Attr {
		name := strings.ToLower(a.Name.Local)
		if strings.HasPrefix(name, "on") {
			return fmt.Sprintf("%s has an event attribute %s on <%s> — an SVG image cannot run code", key, a.Name.Local, el.Name.Local)
		}
		if name == "href" && !selfContainedRef(a.Value) {
			return fmt.Sprintf("%s refers to %q outside the file — an SVG image must be self-contained (embed it or use an #id reference)", key, a.Value)
		}
		if name == "style" {
			if ref := externalCSSURL(a.Value); ref != "" {
				return fmt.Sprintf("%s refers to %q outside the file — an SVG image must be self-contained (embed it or use an #id reference)", key, ref)
			}
		}
	}
	return ""
}

// selfContainedRef는 참조가 파일 안에서 끝나는지다 — 문서 안의 `#id`, 또는 PNG · JPEG를 담은 data URI다.
func selfContainedRef(v string) bool {
	v = strings.TrimSpace(v)
	return strings.HasPrefix(v, "#") || strings.HasPrefix(v, "data:image/png") || strings.HasPrefix(v, "data:image/jpeg")
}

// externalCSSURL은 CSS 안의 첫 바깥 `url(…)`이다 (없으면 빈 문자열) — `url(#id)`와 data URI는 안이다.
func externalCSSURL(css string) string {
	rest := css
	for {
		i := strings.Index(rest, "url(")
		if i < 0 {
			return ""
		}
		rest = rest[i+4:]
		end := strings.IndexByte(rest, ')')
		if end < 0 {
			return ""
		}
		ref := strings.Trim(strings.TrimSpace(rest[:end]), `'"`)
		if !selfContainedRef(ref) {
			return ref
		}
		rest = rest[end:]
	}
}

// formatNum은 진단에 싣는 수치다 (불필요한 소수 없이).
func formatNum(v float64) string {
	return strconv.FormatFloat(v, 'f', -1, 64)
}
