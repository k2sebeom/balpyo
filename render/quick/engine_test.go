package quick

import (
	"fmt"
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

// spanXs는 블록의 줄에서 인라인 요소(span)가 시작하는 x를 문서 순서대로 모은다 — 한 span이
// 여러 조각(CJK는 글자마다 끊을 수 있다)이면 첫 조각의 x다.
func spanXs(b *box) []float64 {
	var xs []float64
	prev := b.st
	for _, l := range b.lines {
		for _, f := range l.frags {
			if f.atomic == nil && f.st != b.st && f.st != prev && !f.space {
				xs = append(xs, f.x)
			}
			prev = f.st
		}
	}
	return xs
}

// text-align: justify는 마지막 줄과 강제 줄바꿈 앞 줄을 빼고 남는 폭을 확장 기회(공백, 한자 ·
// 가나 사이)에 고루 나눈다. 기댓값은 Chromium이 잰 span의 x다.
func TestJustifyDistributesSpace(t *testing.T) {
	const font = `width: 300px; font-size: 20px; line-height: 30px; text-align: justify; font-family: 'Noto Sans'`
	_, b := layoutBody(t, baseCSS,
		`<div id="j" style="`+font+`">one <span>two</span> three four <span>five</span> six seven <span>eight</span> nine</div>`+
			`<div id="k" style="`+font+`">one <span>two</span> three<br>four <span>five</span> six seven eight nine ten <span>eleven</span></div>`+
			`<div id="m" style="`+font+`, 'Noto Sans CJK KR'">슬라이드를 한 번 쓰면 <span>웹과</span> 편집 가능한 파워포인트가 <span>나온다</span></div>`+
			`<div id="n" style="`+font+`, 'Noto Sans CJK JP'; width: 290px">日本語の文章を<span>両端</span>揃えにすると文字の間が広がります</div>`+
			`<div id="r" style="`+font+`">one&nbsp;<span>two</span> three four five six seven eight nine</div>`+
			`<div style="`+font+`">one two<span id="ib" style="display: inline-block; width: 20px; height: 10px"></span>three four five six seven eight</div>`)
	for _, tc := range []struct {
		id   string
		want []float64
	}{
		// 첫 줄은 양 끝에 붙고, 마지막 줄은 시작 쪽에 붙는다.
		{"j", []float64{51.9, 225.0, 59.5}},
		// <br> 앞 줄과 마지막 줄은 늘리지 않는다.
		{"k", []float64{41.0, 51.2, 36.1}},
		// 한글은 어절 사이 공백만 늘어난다.
		{"m", []float64{193.9, 157.6}},
		// 한자 · 가나는 글자 사이가 늘어난다 (290px에 14자, 남는 10px을 13곳에).
		{"n", []float64{145.3}},
		// 줄바꿈 없는 공백도 늘어나는 자리다.
		{"r", []float64{51.9}},
	} {
		got := spanXs(b[tc.id])
		if len(got) != len(tc.want) {
			t.Errorf("%s: %d span fragments, want %d", tc.id, len(got), len(tc.want))
			continue
		}
		for i := range got {
			if math.Abs(got[i]-tc.want[i]) > 0.5 {
				t.Errorf("%s: span %d at x=%.1f, want %.1f", tc.id, i, got[i], tc.want[i])
			}
		}
	}
	// 인라인 블록은 늘어나는 자리가 아니고, 앞의 공백이 늘어난 만큼 밀린다.
	if math.Abs(b["ib"].x-86.0) > 0.5 {
		t.Errorf("inline-block at x=%.1f, want 86.0", b["ib"].x)
	}
	if r := b["j"].lines[0].textRect; math.Abs(r.right()-300) > 0.5 {
		t.Errorf("justified line ends at %.1f, want 300", r.right())
	}
}

// 두 문자가 모두 ASCII일 때 Chromium은 '/' 뒤에서 줄을 바꾸지 않는다 — 200px 칸의 긴 URL은
// 넘쳐도 한 줄이다. ASCII가 아닌 글자 앞에서는 UAX #14대로 끊는다.
func TestNoBreakAfterSlashBeforeASCII(t *testing.T) {
	const font = `font-size: 20px; line-height: 30px; font-family: 'Noto Sans'`
	_, b := layoutBody(t, baseCSS,
		`<div id="u" style="width: 200px; `+font+`">aaaaaaaaaa/bbbbbbbbbb</div>`+
			`<div id="v" style="width: 200px; `+font+`">https://example.com/path/to/some/long/resource</div>`+
			`<div id="w" style="width: 30px; `+font+`">1/a</div>`+
			`<div id="x" style="width: 40px; `+font+`">aaaaa/éaaaaa</div>`)
	for id, want := range map[string]float64{"u": 30, "v": 30, "w": 30, "x": 60} {
		if !near(b[id].h, want) {
			t.Errorf("%s: height %.1f, want %.1f", id, b[id].h, want)
		}
	}
}

// 줄에 꼭 맞는 내용. Chromium은 글리프 진행 폭을 1/64px보다 곱게 더한 내용 폭을 1/64px로 올리고,
// 줄 폭에는 1/64px을 더 허용해 견준다. 줄바꿈 기회 너머의 커닝(に와 す 사이 +0.2px)도 폭에 들고,
// 줄이 거기서 끝나면 빠진다. 기댓값은 Chromium이 잰 줄 수와 첫 줄의 텍스트 폭이다.
func TestExactFitLine(t *testing.T) {
	const jp = `日本語の文章を両端揃えにすると`
	const plain = `日本語日本語日本語日本語日本語`
	cases := []struct {
		w, family, text string
		lines           int
		first           float64 // 0이면 보지 않는다
	}{
		// 커닝 때문에 15자가 300px을 넘친다 — 정렬과 상관없다.
		{"300px", "'Noto Sans CJK JP'", jp, 2, 280.2},
		{"300px", "'Noto Sans CJK JP'; text-align: center", jp, 2, 280.2},
		// に가 들어가는지 볼 때도 뒤 글자와의 커닝이 든다.
		{"240px", "'Noto Sans CJK JP'", jp, 2, 220},
		// 줄이 に에서 끝나면 다음 줄 글자와의 커닝은 빠진다.
		{"250px", "'Noto Sans CJK JP'", jp, 2, 240},
		// 꼭 맞으면 한 줄이고, 1/64px 넘침까지는 봐준다.
		{"300px", "'Noto Sans CJK JP'", plain, 1, 300},
		{"299.984375px", "'Noto Sans CJK JP'", plain, 1, 300},
		{"299.96875px", "'Noto Sans CJK JP'", plain, 2, 0},
		// 1/64px보다 작은 진행 폭의 끝자리가 쌓인다.
		{"143.5px", "'Noto Sans'", "Hello world foo", 2, 0},
		{"143.515625px", "'Noto Sans'", "Hello world foo", 2, 0},
		{"143.53125px", "'Noto Sans'", "Hello world foo", 1, 0},
		{"99.25px", "'Noto Sans'", "aaaa bbbb", 2, 0},
		{"99.265625px", "'Noto Sans'", "aaaa bbbb", 1, 0},
	}
	var body string
	for i, c := range cases {
		body += fmt.Sprintf(`<div id="f%d" style="width: %s; font-size: 20px; line-height: 30px; font-family: %s">%s</div>`,
			i, c.w, c.family, c.text)
	}
	_, b := layoutBody(t, baseCSS, body)
	for i, c := range cases {
		box := b[fmt.Sprintf("f%d", i)]
		if len(box.lines) != c.lines {
			t.Errorf("%q in %s: %d lines, want %d", c.text, c.w, len(box.lines), c.lines)
			continue
		}
		if c.first != 0 && math.Abs(box.lines[0].textRect.w-c.first) > 0.05 {
			t.Errorf("%q in %s: first line %.3fpx wide, want %.1f", c.text, c.w, box.lines[0].textRect.w, c.first)
		}
	}
}
