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

// flex 항목의 자동 최소 크기는 min(지정 크기, 내용 기반 최소)다 — 지정 크기가 있어도 내용이 비면
// 컨테이너에 맞게 줄어든다. 지정 크기를 바닥으로 삼으면 항목이 컨테이너 밖으로 넘친다.
// 기댓값은 Chromium이 같은 마크업에 낸 값이다.
func TestFlexAutoMinimumIsSmallerOfSpecifiedAndContent(t *testing.T) {
	_, b := layoutBody(t, baseCSS, `<div style="display: flex; flex-direction: column; width: 500px; height: 400px">`+
		`<div id="a" style="height: 390px"></div><div id="b" style="height: 36px"></div></div>`+
		`<div style="display: flex; width: 400px; height: 100px">`+
		`<div id="c" style="width: 400px"></div><div id="d" style="width: 36px"></div></div>`)
	if !near(b["a"].h, 366.1972) || !near(b["b"].y, 366.1972) || !near(b["b"].h, 33.8028) {
		t.Errorf("column: a h=%.4f, b y=%.4f h=%.4f; want 366.1972, 366.1972 + 33.8028", b["a"].h, b["b"].y, b["b"].h)
	}
	if !near(b["c"].w, 366.9725) || !near(b["d"].x, 366.9725) || !near(b["d"].w, 33.0275) {
		t.Errorf("row: c w=%.4f, d x=%.4f w=%.4f; want 366.9725, 366.9725 + 33.0275", b["c"].w, b["d"].x, b["d"].w)
	}
}

// 내용의 min-content가 지정 크기보다 작으면 항목은 거기까지 줄고, 줄일 수 없는 낱말은 지켜진다
// (Chromium: 98.6 + 10.6 — 컨테이너 100px을 넘친다).
func TestFlexAutoMinimumKeepsContentMinimum(t *testing.T) {
	_, b := layoutBody(t, baseCSS, `<div style="display: flex; width: 100px; font-size: 20px; font-family: 'Noto Sans'; line-height: 30px">`+
		`<div id="a" style="width: 300px">abcdefghij</div><div id="b" style="width: 300px">x</div></div>`)
	if math.Abs(b["a"].w-98.6) > 0.5 || math.Abs(b["b"].w-10.6) > 0.5 {
		t.Errorf("a w=%.2f, b w=%.2f; want about 98.6 and 10.6", b["a"].w, b["b"].w)
	}
}
