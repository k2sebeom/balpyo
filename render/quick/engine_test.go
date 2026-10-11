package quick

import (
	"math"
	"strings"
	"testing"
)

func near(a, b float64) bool { return math.Abs(a-b) < 0.01 }

func TestParseLengthCalc(t *testing.T) {
	for _, tc := range []struct {
		in      string
		px, pct float64
	}{
		{"24px", 24, 0},
		{"50%", 0, 50},
		{"calc(36px + 7.0294px)", 43.0294, 0},
		{"calc(10% + 4px)", 4, 10},
		{"calc(-1 * 14px)", -14, 0},
		{"1.5em", 36, 0},
		{"0", 0, 0},
	} {
		l, ok := parseLength(tc.in, 24)
		if !ok || !near(l.px, tc.px) || !near(l.pct, tc.pct) {
			t.Errorf("parseLength(%q) = %+v, %v; want px %v pct %v", tc.in, l, ok, tc.px, tc.pct)
		}
	}
}

func TestParseColor(t *testing.T) {
	for _, tc := range []struct {
		in   string
		want rgba
	}{
		{"#0F65F9", rgba{15.0 / 255, 101.0 / 255, 249.0 / 255, 1}},
		{"rgb(0 0 0 / 0.2)", rgba{0, 0, 0, 0.2}},
		{"rgba(255, 255, 255, 0.5)", rgba{1, 1, 1, 0.5}},
		// 웹 렌더러의 tint: 명도를 0.25배 + 75로 올린다
		{"hsl(from #000000 h s calc(l * 0.25 + 75))", rgba{0.75, 0.75, 0.75, 1}},
		{"color-mix(in srgb, #000000 15%, transparent)", rgba{0, 0, 0, 0.15}},
	} {
		got, ok := parseColor(tc.in, rgba{})
		if !ok || !near(got.r, tc.want.r) || !near(got.g, tc.want.g) || !near(got.b, tc.want.b) || !near(got.a, tc.want.a) {
			t.Errorf("parseColor(%q) = %+v, %v; want %+v", tc.in, got, ok, tc.want)
		}
	}
}

func TestSelectorMatching(t *testing.T) {
	root, _, err := parseHTML([]byte(`<html lang="ko"><body><div class="balpyo-row"><p class="balpyo-text" id="a">x</p></div></body></html>`))
	if err != nil {
		t.Fatal(err)
	}
	p := root.find(func(n *node) bool { return n.attrs["id"] == "a" })[0]
	for sel, want := range map[string]bool{
		":is(.balpyo-row, .balpyo-col) > :is(.balpyo-text, .balpyo-list)": true,
		".balpyo-row p":                      true,
		"body > p":                           false,
		"p:lang(ko)":                         true,
		"p:not(.balpyo-text)":                false,
		`html[data-balpyo-view="measure"] p`: false,
	} {
		sels := parseSelectorList(sel)
		if len(sels) != 1 {
			t.Fatalf("%q did not parse", sel)
		}
		if got := sels[0].matches(p); got != want {
			t.Errorf("%q matches = %v, want %v", sel, got, want)
		}
	}
	if a, b := parseComplex(":is(.a, #b) p").spec, parseComplex(".a .b p").spec; !b.less(a) {
		t.Errorf(":is() takes the specificity of its most specific argument: %v vs %v", a, b)
	}
}

// layoutBody는 슬라이드 하나짜리 문서를 배치한다 (스테이지 CSS 없이 주어진 마크업 그대로).
func layoutBody(t *testing.T, css, body string) (*Document, map[string]*box) {
	t.Helper()
	src := `<html lang="ko"><head><style>` + css + `</style></head><body>` +
		`<section class="balpyo-slide" data-slide-index="0" style="position: relative; width: 1280px; height: 720px">` +
		body + `</section></body></html>`
	doc, err := Layout([]byte(src), func(string) ([]byte, bool) { return nil, false })
	if err != nil {
		t.Fatal(err)
	}
	byID := map[string]*box{}
	for n, b := range doc.Slides[0].boxes {
		if id, ok := n.attrs["id"]; ok {
			byID[id] = b
		}
	}
	return doc, byID
}

const baseCSS = `* { box-sizing: border-box; } body { margin: 0; }`

// 줄어드는 몫은 inner flex base size(padding 제외)에 비례한다 — border-box에 비례하면 padding을
// 가진 항목이 더 많이 줄어든다.
func TestFlexShrinkUsesInnerBase(t *testing.T) {
	_, b := layoutBody(t, baseCSS, `<div id="row" style="display: flex; gap: 36px; width: 1152px">`+
		`<div id="a" style="width: 100%; padding: 24px"></div><div id="b" style="width: 100%"></div></div>`)
	if !near(b["a"].w, 570.6383) || !near(b["b"].w, 545.3617) {
		t.Errorf("widths = %.4f, %.4f; want 570.6383, 545.3617", b["a"].w, b["b"].w)
	}
}

func TestGridTemplateAreas(t *testing.T) {
	_, b := layoutBody(t, baseCSS, `<div id="g" style="display: grid; width: 1000px; height: 600px; gap: 20px; `+
		`grid-template-columns: 1fr 1fr; grid-template-rows: auto 1fr; grid-template-areas: 'title title' 'left right'">`+
		`<div id="t" style="grid-area: title; height: 80px"></div>`+
		`<div id="l" style="grid-area: left"></div><div id="r" style="grid-area: right"></div></div>`)
	if b["t"].w != 1000 || b["l"].y != 100 || !near(b["l"].h, 500) || !near(b["r"].x, 510) || !near(b["r"].w, 490) {
		t.Errorf("title %+v left %+v right %+v", *b["t"], *b["l"], *b["r"])
	}
}

// 행의 최소 기여는 min-height다 — 높이가 퍼센트(칸에 기댄다)이고 min-height: 0인 항목은 내용이
// 더 커도 `1fr` 행을 굳히지 않는다. 내용 높이를 바닥으로 삼으면 fr 행이 컨테이너를 다 먹고
// 아래 auto 행이 컨테이너 밖으로 밀려난다 (image-full의 media 그림 + caption).
func TestGridRowMinContributionUsesMinHeight(t *testing.T) {
	_, b := layoutBody(t, baseCSS, `<div id="g" style="display: grid; width: 1280px; height: 720px; row-gap: 24px; `+
		`grid-template-rows: 1fr auto; grid-template-areas: 'media' 'caption'">`+
		`<div id="m" style="grid-area: media; height: 100%; min-height: 0"><div style="height: 900px"></div></div>`+
		`<div id="c" style="grid-area: caption; height: 25px"></div></div>`)
	if !near(b["m"].y, 0) || !near(b["m"].h, 671) || !near(b["c"].y, 695) || !near(b["c"].h, 25) {
		t.Errorf("media y=%.1f h=%.1f, caption y=%.1f h=%.1f; want media 0+671, caption 695+25",
			b["m"].y, b["m"].h, b["c"].y, b["c"].h)
	}
}

// 여러 auto 행에 걸친 항목이 모자란 높이를 채울 때 행은 성장 한계(내용 높이)까지만 먼저 자라고,
// 항목이 없는 행은 한계가 없어 나머지를 다 받는다. 모든 행이 한계에 닿았을 때만 고르게 넘친다.
// 기대값은 Chromium이 잰 값이다.
func TestGridSpanningItemGrowsRowsUpToLimits(t *testing.T) {
	_, b := layoutBody(t, baseCSS,
		// 둘째 행이 비었다: 첫 행은 내용 높이 40에 머물고 빈 행이 150을 받는다.
		`<div id="g1" style="display: grid; width: 400px; gap: 10px; grid-template-columns: 1fr 1fr">`+
			`<div style="grid-row: span 2; height: 200px"></div><div id="a1"><div style="height: 40px"></div></div></div>`+
			// 세 행 중 가운데만 비었다: 40 · 240 · 20.
			`<div id="g2" style="display: grid; width: 400px; grid-template-columns: 1fr 1fr 1fr; grid-template-rows: auto auto auto">`+
			`<div style="grid-row: span 3; height: 300px"></div><div id="a2" style="height: 40px"></div>`+
			`<div id="c2" style="grid-column: 3; grid-row: 3; height: 20px"></div></div>`+
			// 두 행 모두 한계(40 · 0)에 닿으면 남는 160을 고르게 나눈다: 120 · 80.
			`<div id="g3" style="display: grid; width: 400px; grid-template-columns: 1fr 1fr">`+
			`<div style="grid-row: span 2; height: 200px"></div><div style="height: 40px"></div><div id="b3"></div></div>`)
	if !near(b["a1"].h, 40) {
		t.Errorf("empty second row: first row item h=%.1f, want 40", b["a1"].h)
	}
	if !near(b["a2"].h, 40) || !near(b["c2"].y, 280) {
		t.Errorf("empty middle row: a h=%.1f, c y=%.1f; want 40, 280", b["a2"].h, b["c2"].y)
	}
	if !near(b["b3"].y, 120) || !near(b["b3"].h, 80) {
		t.Errorf("both rows at limit: second row y=%.1f h=%.1f; want 120, 80", b["b3"].y, b["b3"].h)
	}
}

// grid-auto-flow는 자동 배치의 방향과 채우기를 정한다: column은 열 우선으로 채우고, dense는 앞에 남은
// 빈칸으로 되돌아가 채운다. 한 축이 정해진 항목은 그 축이 흐름 방향의 줄이면 자동 항목보다 먼저 놓인다.
// 기대값은 Chromium이 잰 값이다 (x/y는 컨테이너 기준).
func TestGridAutoFlow(t *testing.T) {
	_, b := layoutBody(t, baseCSS,
		`<div style="display: grid; width: 600px; grid-template-columns: repeat(3, 1fr); grid-auto-flow: column; grid-template-rows: repeat(2, 60px)">`+
			`<div id="i1"></div><div id="i2"></div><div id="i3"></div><div id="i4"></div><div id="i5"></div></div>`+
			`<div style="display: grid; width: 600px; grid-auto-flow: column; grid-template-rows: repeat(2, 50px)">`+
			`<div id="j1"></div><div id="j2" style="grid-row: span 2"></div><div id="j3"></div><div id="j4" style="grid-row: 2"></div><div id="j5"></div></div>`+
			`<div style="display: grid; width: 600px; grid-template-columns: repeat(3, 1fr); grid-auto-flow: row dense">`+
			`<div id="d1" style="height: 24px"></div><div id="d2" style="grid-column: span 3; height: 24px"></div><div id="d3" style="height: 24px"></div>`+
			`<div id="d4" style="grid-column: span 2; height: 24px"></div><div id="d5" style="height: 24px"></div></div>`+
			`<div style="display: grid; width: 600px; grid-template-rows: repeat(3, 30px); grid-auto-flow: column dense">`+
			`<div id="e1"></div><div id="e2" style="grid-row: span 3"></div><div id="e3"></div><div id="e4" style="grid-column: 1"></div><div id="e5" style="grid-row: 3"></div></div>`+
			// 기본 흐름(row)에서도 행이 정해진 항목(f2 · f3)이 앞선 자동 항목보다 먼저 첫 행을 차지한다.
			`<div style="display: grid; width: 400px; grid-template-columns: 1fr 1fr; grid-auto-rows: 20px">`+
			`<div id="f1"></div><div id="f2" style="grid-row: 1"></div><div id="f3" style="grid-row: 1"></div><div id="f4" style="grid-column: 1"></div><div id="f5"></div></div>`)
	want := map[string][4]float64{
		"i1": {0, 0, 200, 60}, "i2": {0, 60, 200, 60}, "i3": {200, 0, 200, 60}, "i4": {200, 60, 200, 60}, "i5": {400, 0, 200, 60},
		"j1": {0, 0, 150, 50}, "j2": {150, 0, 150, 100}, "j3": {300, 0, 150, 50}, "j4": {300, 50, 150, 50}, "j5": {450, 0, 150, 50},
		"d1": {0, 0, 200, 24}, "d2": {0, 24, 600, 24}, "d3": {200, 0, 200, 24}, "d4": {0, 48, 400, 24}, "d5": {400, 0, 200, 24},
		"e1": {0, 30, 200, 30}, "e2": {200, 0, 200, 90}, "e3": {0, 60, 200, 30}, "e4": {0, 0, 200, 30}, "e5": {400, 60, 200, 30},
		"f1": {0, 20, 200, 20}, "f2": {0, 0, 200, 20}, "f3": {200, 0, 200, 20}, "f4": {0, 40, 200, 20}, "f5": {200, 40, 200, 20},
	}
	for id, r := range want {
		k := b[id]
		if !near(k.x, r[0]) || !near(k.y, r[1]) || !near(k.w, r[2]) || !near(k.h, r[3]) {
			t.Errorf("%s = (%.1f,%.1f %.1fx%.1f), want (%.0f,%.0f %.0fx%.0f)", id, k.x, k.y, k.w, k.h, r[0], r[1], r[2], r[3])
		}
	}
}

// grid 컨테이너의 절대 배치 자식은 grid-area가 정한 영역을 포함 블록으로 쓴다 — 끝 줄이 auto인
// 변은 padding box 끝까지 간다.
func TestAbsoluteInGridArea(t *testing.T) {
	_, b := layoutBody(t, baseCSS, `<div id="g" style="display: grid; position: relative; width: 1280px; height: 720px; padding: 64px">`+
		`<div id="k" style="grid-area: 1 / 1; position: absolute; left: 50%; top: 10px; width: 10px; height: 10px"></div></div>`)
	if !near(b["k"].x, 64+0.5*1216) || !near(b["k"].y, 74) {
		t.Errorf("abs child at (%.1f, %.1f), want (672, 74)", b["k"].x, b["k"].y)
	}
}

// 한국어 keep-all: 어절 사이(공백)에서만 접는다.
func TestKeepAllBreaksAtSpaces(t *testing.T) {
	_, b := layoutBody(t, baseCSS, `<div id="p" style="width: 130px; font-size: 24px; line-height: 1.5; `+
		`font-family: 'Noto Sans', 'Noto Sans CJK KR', sans-serif; word-break: keep-all">슬라이드를 한 번 쓰면</div>`)
	p := b["p"]
	if len(p.lines) < 2 {
		t.Fatalf("want the paragraph to wrap, got %d line(s)", len(p.lines))
	}
	para := buildPara(p.inline)
	for _, w := range para.words() {
		word := strings.TrimSpace(string(para.runes[w.start:w.end]))
		if strings.Contains(word, " ") || (word != "슬라이드를" && word != "한" && word != "번" && word != "쓰면") {
			t.Errorf("break opportunity inside a word: %q", word)
		}
	}
}
