package core

import (
	"strings"
	"testing"
	"testing/fstest"

	"github.com/k2sebeom/balpyo/soml"
)

// SVG 그림의 판정 — 고유 크기는 width · height(px), 없으면 viewBox에서 오고, 자기 안에서 끝나지 않는 SVG는 진단이다.
func TestDecodeSVGAsset(t *testing.T) {
	for _, tc := range []struct {
		name, svg string
		w, h      int
	}{
		{"width · height", `<svg xmlns="http://www.w3.org/2000/svg" width="48" height="32"/>`, 48, 32},
		{"px 접미", `<svg xmlns="http://www.w3.org/2000/svg" width="48px" height="32px"/>`, 48, 32},
		{"width와 viewBox 비율", `<svg xmlns="http://www.w3.org/2000/svg" width="32" viewBox="0 0 64 16"/>`, 32, 8},
		{"안의 참조와 data URI", `<svg xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" width="8" height="8"><defs><linearGradient id="g"/></defs><rect fill="url(#g)" style="fill: url(#g)"/><use xlink:href="#g"/><image href="data:image/png;base64,AAAA"/></svg>`, 8, 8},
	} {
		t.Run(tc.name, func(t *testing.T) {
			asset, msg := decodeImageAsset("assets/a.svg", []byte(tc.svg))
			if msg != "" {
				t.Fatalf("진단: %s", msg)
			}
			if asset.Format != soml.FormatSVG || asset.Width != tc.w || asset.Height != tc.h {
				t.Errorf("에셋 = %s %d×%d, want svg %d×%d", asset.Format, asset.Width, asset.Height, tc.w, tc.h)
			}
		})
	}
	for _, tc := range []struct{ name, svg, want string }{
		{"XML이 아니다", `<svg width="4"`, "does not parse as XML"},
		{"뿌리가 svg가 아니다", `<html/>`, "the root element is <html>, not <svg>"},
		{"크기가 없다", `<svg xmlns="http://www.w3.org/2000/svg"/>`, "has no intrinsic size"},
		{"viewBox만", `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 64 16"/>`, `has only a viewBox — give the <svg> width and height in px (e.g. width="64" height="16")`},
		{"em 단위", `<svg xmlns="http://www.w3.org/2000/svg" width="2em" height="2em"/>`, `width="2em" — an SVG image gives its size in px`},
		{"스크립트", `<svg xmlns="http://www.w3.org/2000/svg" width="4" height="4"><script>x()</script></svg>`, "contains <script>"},
		{"이벤트 속성", `<svg xmlns="http://www.w3.org/2000/svg" width="4" height="4" onload="x()"/>`, "event attribute onload"},
		{"foreignObject", `<svg xmlns="http://www.w3.org/2000/svg" width="4" height="4"><foreignObject/></svg>`, "contains <foreignObject>"},
		{"바깥 그림", `<svg xmlns="http://www.w3.org/2000/svg" width="4" height="4"><image href="logo.png"/></svg>`, `refers to "logo.png" outside the file`},
		{"바깥 CSS url", `<svg xmlns="http://www.w3.org/2000/svg" width="4" height="4"><style>rect { fill: url(other.svg#g) }</style></svg>`, `refers to "other.svg#g" outside the file`},
		{"@import", `<svg xmlns="http://www.w3.org/2000/svg" width="4" height="4"><style>@import url(x.css);</style></svg>`, "imports an external stylesheet"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, msg := decodeImageAsset("assets/a.svg", []byte(tc.svg))
			if !strings.Contains(msg, tc.want) {
				t.Errorf("진단 = %q, want %q", msg, tc.want)
			}
		})
	}
}

// 테마의 배경 · 로고는 SVG를 받지 않는다 — 슬라이드의 `<image>`만 SVG를 연다.
func TestThemeDeclarationsRejectSVG(t *testing.T) {
	svg := `<svg xmlns="http://www.w3.org/2000/svg" width="8" height="8"/>`
	a := &assetResolver{files: map[string]rawAsset{
		"assets/logo.svg": {owner: "talk", fsys: fstest.MapFS{"assets/logo.svg": {Data: []byte(svg)}}, path: "assets/logo.svg"},
	}}
	_, err := a.resolve("logo", "assets/logo.svg")
	if err == nil || !strings.Contains(err.Error(), `logo "assets/logo.svg" is an SVG — theme backgrounds and logos are png or jpeg`) {
		t.Errorf("에러 = %v", err)
	}
	// 같은 파일을 슬라이드의 `theme:` 그림으로 쓰는 길은 열려 있다.
	b := &assetResolver{files: a.files}
	if _, found, msg := b.use("assets/logo.svg"); !found || msg != "" {
		t.Errorf("theme: 그림 = found %v, msg %q", found, msg)
	}
}
