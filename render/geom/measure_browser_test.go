// 이 파일만 브라우저를 띄운다 — 외부 도구 정책(없으면 로컬은 건너뛰고 CI(Strict)에서는
// 실패)은 internal/testkit이 소유하므로, 패키지 밖 테스트로 두어 testkit을 임포트한다
// (testkit → pipeline → render/pptx → render/geom 이므로 같은 패키지 안에서는 순환이 된다).
package geom_test

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/k2sebeom/balpyo/core"
	"github.com/k2sebeom/balpyo/internal/testkit"
	"github.com/k2sebeom/balpyo/render/geom"
	"github.com/k2sebeom/balpyo/render/web"
	"github.com/k2sebeom/balpyo/soml"
)

// renderHTML은 덱을 임시 디렉토리에 쓴다. 에셋까지 전부 쓴다 — 빼고 재면 깨진 이미지(0×0)를 잰다.
func renderHTML(t *testing.T, deck *soml.Deck, transform func(string) string) string {
	t.Helper()
	files, err := web.New().Render(deck)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	for _, name := range files.Paths() {
		data := files[name]
		if name == web.IndexFile && transform != nil {
			data = []byte(transform(string(data)))
		}
		dest := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(dest, data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return filepath.Join(dir, web.IndexFile)
}

// parseDeck은 원문을 파싱·해석한 덱이다 — 손으로 만든 트리는 값 층을 건너뛴다. slideAttrs는
// `<slide>`에 그대로 붙는다 (layout 등).
func parseDeck(t *testing.T, slideAttrs string, bodies ...string) *soml.Deck {
	t.Helper()
	stage, err := soml.Aspect16x9.Stage()
	if err != nil {
		t.Fatal(err)
	}
	deck := &soml.Deck{Title: "layout", Lang: "ko", Aspect: soml.Aspect16x9, Stage: stage}
	for i, body := range bodies {
		source := fmt.Sprintf("slides/%03d_layout%s", i+1, soml.Ext)
		id := fmt.Sprintf("layout%d", i)
		doc, diags := soml.Parse(source, fmt.Sprintf("<slide id=%q%s>\n%s\n</slide>\n", id, slideAttrs, body))
		if err := diags.Err(); err != nil {
			t.Fatalf("파싱 실패: %v", err)
		}
		deck.Slides = append(deck.Slides, &soml.Slide{
			Index: i, ID: id, Source: source,
			Layout: soml.SlideLayout(doc.Layout), LayoutSpan: doc.LayoutSpan,
			Notes: doc.Notes, Content: doc.Content,
		})
	}
	if err := core.Resolve(deck, nil).Err(); err != nil {
		t.Fatalf("해석 실패: %v", err)
	}
	return deck
}

func layoutDeck(t *testing.T, bodies ...string) *soml.Deck {
	t.Helper()
	return parseDeck(t, "", bodies...)
}

// imageDeck은 layoutDeck에 에셋 패스를 더한 것이다 (파일 시스템은 그림 픽스처).
func imageDeck(t *testing.T, bodies ...string) *soml.Deck {
	t.Helper()
	deck := layoutDeck(t, bodies...)
	dir := filepath.Join("..", "..", "testdata", "fixtures", "05_image")
	if err := core.ResolveAssets(deck, os.DirFS(dir)).Err(); err != nil {
		t.Fatalf("에셋 해석 실패: %v", err)
	}
	return deck
}

// measureLayout은 덱을 잰다. 성공 자체가 Refs ↔ querySelectorAll 자리 맞대기 통과다.
func measureLayout(t *testing.T, deck *soml.Deck) *geom.Geometry {
	t.Helper()
	testkit.RequireChromium(t)
	g, err := testkit.SharedSession(t).Measure(deck, renderHTML(t, deck, nil))
	if err != nil {
		t.Fatalf("지오메트리 패스 실패: %v", err)
	}
	return g
}

func spaceToken(t *testing.T, deck *soml.Deck, name string) float64 {
	t.Helper()
	token, ok := deck.Theme.Space(name)
	if !ok {
		t.Fatalf("공간 토큰 %q가 테마에 없다", name)
	}
	return token.Px
}

func boxOf(t *testing.T, g *geom.Geometry, id string) geom.Box {
	t.Helper()
	box, ok := g.Box(id)
	if !ok {
		t.Fatalf("%s가 측정 목록에 없다: %+v", id, g.Boxes)
	}
	return box
}

func near(a, b float64) bool { return math.Abs(a-b) <= 0.5 }

// 좌표는 논리 px이고 **슬라이드 상대**다 — 중첩 깊이와 무관하다. 같은 내용의 두 장이 같은
// rect를 가져야 한다 (문서 상대라면 둘째 장의 y가 768px 아래다).
func TestMeasureReadsSlideRelativeBoxes(t *testing.T) {
	body := `<text role="title">2026 아키텍처 리뷰</text>
  <col pad="lg"><row><text>중첩된 텍스트</text></row></col>`
	deck := layoutDeck(t, body, body)
	g := measureLayout(t, deck)

	// 스테이지 여백 64px, 콘텐츠 폭 1152px — 여기가 어긋나면 PPTX의 모든 좌표가 틀린다.
	if title := boxOf(t, g, "s0.0"); title.Rect.X != 64 || title.Rect.Y != 64 || title.Rect.W != 1152 {
		t.Errorf("제목 박스 = %s, want (64,64 1152×…)", title.Rect)
	}
	for _, id := range []string{"0", "1", "1.0", "1.0.0"} {
		first, second := boxOf(t, g, "s0."+id), boxOf(t, g, "s1."+id)
		if first.Rect != second.Rect {
			t.Errorf("%s: 슬라이드 0 = %s, 슬라이드 1 = %s — 좌표가 슬라이드 상대가 아니다", id, first.Rect, second.Rect)
		}
		if second.Rect.Y >= float64(deck.Stage.Height) {
			t.Errorf("s1.%s의 y = %v — 문서 좌표가 섞였다", id, second.Rect.Y)
		}
	}
}

// 인라인 런의 글자도 그 블록의 줄이다. 줄 전체가 런이면 줄이 0개가 되고(가로 이탈 검사가
// 글자를 못 본다), 런이 섞이면 조각마다 줄을 센다.
func TestMeasureLinesIncludeInlineRuns(t *testing.T) {
	deck := layoutDeck(t,
		`<p><strong>줄 전체가 런이다</strong></p>`,
		`<p>배포 <strong>자동화</strong>와 <code>관측성</code> 스택</p>`,
		`<ul><li>모놀리스 분리 완료</li></ul>`)
	g := measureLayout(t, deck)

	for _, id := range []string{"s0.0", "s1.0"} {
		box := boxOf(t, g, id)
		if len(box.Lines) != 1 {
			t.Fatalf("%s의 줄 수 = %d, want 1 (%+v)", id, len(box.Lines), box.Lines)
		}
		line := box.Lines[0]
		if !near(line.X, box.Rect.X) || line.W <= 0 || line.Right() > box.Rect.Right()+0.5 {
			t.Errorf("%s의 줄 = %s, 박스 = %s", id, line, box.Rect)
		}
		// 줄 높이는 글자 상자다 — 박스 높이를 베끼면 줄 슬롯을 잰 것이다.
		if line.H <= 0 || line.H > box.Rect.H {
			t.Errorf("%s의 줄 높이 = %v, 박스 높이 = %v", id, line.H, box.Rect.H)
		}
	}
	// li 글자는 리스트 왼쪽에서 ListTextIndent만큼 들어온다 — PPTX `marL`이 같은 상수를 쓴다.
	ul, li := boxOf(t, g, "s2.0"), boxOf(t, g, "s2.0.0")
	if len(li.Lines) == 0 || !near(li.Lines[0].X, ul.Rect.X+web.ListTextIndent) {
		t.Errorf("li 줄 = %v, want x = %v (ul.x + %d)", li.Lines, ul.Rect.X+web.ListTextIndent, web.ListTextIndent)
	}
}

// 페이지가 조용히 스케일되면 모든 좌표가 틀린 채로 산출물까지 간다.
func TestMeasureRejectsScaledPage(t *testing.T) {
	testkit.RequireChromium(t)
	deck := layoutDeck(t, `<p>본문</p>`)
	path := renderHTML(t, deck, func(html string) string {
		return strings.Replace(html, "</head>", "<style>.balpyo-slide { zoom: 0.5; }</style>\n</head>", 1)
	})
	_, err := testkit.SharedSession(t).Measure(deck, path)
	if err == nil || !strings.Contains(err.Error(), "scaled") {
		t.Errorf("스케일된 페이지의 에러 = %v, want 스케일 진단", err)
	}
}

func TestMeasureRejectsDeckMismatch(t *testing.T) {
	testkit.RequireChromium(t)
	deck := layoutDeck(t, `<p>본문</p>`)
	path := renderHTML(t, deck, nil)

	extra := *deck
	extra.Slides = append(append([]*soml.Slide{}, deck.Slides...), &soml.Slide{
		Index: 1, ID: "extra", Source: "slides/002_extra.slide",
		Content: []soml.Node{&soml.Element{Name: soml.TextElement}},
	})
	if _, err := testkit.SharedSession(t).Measure(&extra, path); err == nil {
		t.Error("HTML에 없는 슬라이드가 있는 덱은 에러여야 한다")
	}
}

// 회전·대칭한 도형·그림의 잰 rect는 **회전 전** 박스다 — OOXML `xfrm@rot`은 회전 전 `ext`를
// 돌린 것이다. 측정 뷰를 켜서 재고 되돌리므로, 되돌린 뒤의 rect는 회전된 bbox여야 한다.
func TestMeasureRotatedIsUnrotatedBox(t *testing.T) {
	deck := imageDeck(t, `<row gap="md">
    <shape type="rect" fill="accent1" w="120" h="120" rotate="45"/>
    <shape type="rect" fill="accent1" w="120" h="120"/>
    <image src="assets/wide.png" fit="cover" w="200" h="100" rotate="45" alt="회전"/>
    <image src="assets/wide.png" fit="cover" w="200" h="100" flip="h" alt="대칭"/>
  </row>`)
	g := measureLayout(t, deck)

	for _, tc := range []struct {
		id   string
		w, h float64
	}{{"s0.0.0", 120, 120}, {"s0.0.1", 120, 120}, {"s0.0.2", 200, 100}, {"s0.0.3", 200, 100}} {
		if box := boxOf(t, g, tc.id); !near(box.Rect.W, tc.w) || !near(box.Rect.H, tc.h) {
			t.Errorf("%s의 잰 박스 = %s, want %v×%v — 변형이 섞였다", tc.id, box.Rect, tc.w, tc.h)
		}
	}

	// 되돌리지 않으면 그 뒤의 스크린샷이 측정 배치를 찍는다.
	session := testkit.SharedSession(t)
	var view string
	if err := session.Eval(fmt.Sprintf("document.documentElement.getAttribute(%q) || \"\"", web.ViewAttr), &view); err != nil {
		t.Fatal(err)
	}
	if view != "" {
		t.Errorf("측정 뒤 뷰 속성이 %q로 남았다", view)
	}
	// 문서 뷰에서는 회전이 살아 있다 — 회전을 아예 렌더하지 않는 구현을 거른다.
	rects, err := session.Rects(web.ElementSelector())
	if err != nil {
		t.Fatal(err)
	}
	if len(rects) != 5 {
		t.Fatalf("측정 대상 %d개, want 5 (row + 도형 둘 + 그림 둘)", len(rects))
	}
	if want := 120 * math.Sqrt2; math.Abs(rects[1].W-want) > 1 {
		t.Errorf("문서 뷰에서 회전 도형의 bbox 폭 = %v, want %v", rects[1].W, want)
	}
}

// `pad`는 도형 박스를 키우지 않고(border-box) 안쪽으로 들어가며, `anchor`가 세로 위치를 바꾼다.
// 앵커가 보이는 유일한 자리가 도형 안이다.
func TestMeasureShapePadAndAnchor(t *testing.T) {
	deck := layoutDeck(t, `<row gap="md" align="stretch">
    <shape type="rect" fill="surface" w="240" h="220" pad="md"><p anchor="top">위</p></shape>
    <shape type="rect" fill="surface" w="240" h="220" pad="md"><p anchor="center">가운데</p></shape>
    <shape type="rect" fill="surface" w="240" h="220" pad="md"><p anchor="bottom">아래</p></shape>
  </row>`)
	g := measureLayout(t, deck)

	pad := spaceToken(t, deck, "md")
	var shapes, texts [3]geom.Box
	for i := range shapes {
		shapes[i], texts[i] = boxOf(t, g, fmt.Sprintf("s0.0.%d", i)), boxOf(t, g, fmt.Sprintf("s0.0.%d.0", i))
		if !near(shapes[i].Rect.W, 240) || !near(shapes[i].Rect.H, 220) {
			t.Errorf("도형 %d의 박스 = %s, want 240×220 (pad가 박스를 키웠다)", i, shapes[i].Rect)
		}
	}
	if !near(texts[0].Rect.X, shapes[0].Rect.X+pad) || !near(texts[0].Rect.Y, shapes[0].Rect.Y+pad) {
		t.Errorf("anchor=top 텍스트 = %s, want 도형 좌상단 + pad %v", texts[0].Rect, pad)
	}
	above := texts[1].Rect.Y - (shapes[1].Rect.Y + pad)
	below := (shapes[1].Rect.Bottom() - pad) - texts[1].Rect.Bottom()
	if !near(above, below) || above <= 0 {
		t.Errorf("anchor=center 텍스트의 위 여백 %v, 아래 여백 %v", above, below)
	}
	if want := shapes[2].Rect.Bottom() - pad; !near(texts[2].Rect.Bottom(), want) {
		t.Errorf("anchor=bottom 텍스트의 바닥 = %v, want %v", texts[2].Rect.Bottom(), want)
	}
}

// 그림의 외관 선언이 브라우저에서 실제 값이 되는지 — `var()`·무효한 `filter`는 조용히 사라진다.
func TestMeasureShapeInkStylesResolve(t *testing.T) {
	testkit.RequireChromium(t)
	deck := layoutDeck(t, `<row gap="md">
    <shape type="rect" fill="accent1" border="3 solid muted" shadow="soft" opacity="0.4" w="180" h="120"/>
  </row>`)
	browser := testkit.NewBrowser(t)
	browser.Load(t, renderHTML(t, deck, nil))

	var got struct {
		Fill    string `json:"fill"`
		Stroke  string `json:"stroke"`
		Width   string `json:"width"`
		Filter  string `json:"filter"`
		Opacity string `json:"opacity"`
	}
	browser.Eval(t, `(() => {
  const ink = document.querySelector('[data-balpyo-el="shape"] > svg');
  const s = getComputedStyle(ink);
  return {fill: s.fill, stroke: s.stroke, width: s.strokeWidth, filter: s.filter, opacity: s.opacity};
})()`, &got)

	// accent1 #0F65F9, muted #6B7280. 그림자 soft: dist 4 · dir 90 → 아래로 4px.
	if got.Fill != "rgb(15, 101, 249)" || got.Stroke != "rgb(107, 114, 128)" || got.Width != "3px" || got.Opacity != "0.4" {
		t.Errorf("계산된 외관 = %+v", got)
	}
	if !strings.Contains(got.Filter, "drop-shadow") || !strings.Contains(got.Filter, "0px 4px 16px") {
		t.Errorf("그림자 filter = %q — 토큰 변수나 color-mix()가 풀리지 않았다", got.Filter)
	}
}

// 색 변형(shade/tint)은 웹에서 PPTX와 **같은 색**이 된다. PPTX는 `lumMod`/`lumOff`로 내고 그 둘은
// HSL의 L에 곱하고 더하는 연산이다 — 기대값을 그 식으로 Go에서 따로 계산해 브라우저가 계산한 색과
// 맞댄다 (한 채널 ±1은 반올림 몫). 채도 높은 어두운 색(accent3 #15803D)이 증인이다: sRGB에서
// 흰색과 섞으면 채도가 빠져 PPTX의 밝은 초록과 다른 회녹색이 된다.
func TestMeasureColorVariantsMatchLumModLumOff(t *testing.T) {
	testkit.RequireChromium(t)
	deck := layoutDeck(t, `<row gap="md">
    <shape type="rect" fill="tint(accent3, 40%)" border="none" w="80" h="80"/>
    <shape type="rect" fill="shade(accent3, 40%)" border="none" w="80" h="80"/>
    <shape type="rect" fill="tint(accent1, 25%)" border="none" w="80" h="80"/>
    <shape type="rect" fill="shade(#F59E0B, 70%)" border="none" w="80" h="80"/>
  </row>`)
	browser := testkit.NewBrowser(t)
	browser.Load(t, renderHTML(t, deck, nil))
	var fills []string
	browser.Eval(t, `Array.from(document.querySelectorAll('[data-balpyo-el="shape"] > svg')).map((svg) => getComputedStyle(svg).fill)`, &fills)

	accent1, _ := deck.Theme.Color("accent1")
	accent3, _ := deck.Theme.Color("accent3")
	want := []struct {
		name     string
		base     string
		mod, off float64
	}{
		{"tint(accent3, 40%)", accent3.RGB, 0.6, 0.4},
		{"shade(accent3, 40%)", accent3.RGB, 0.6, 0},
		{"tint(accent1, 25%)", accent1.RGB, 0.75, 0.25},
		{"shade(#F59E0B, 70%)", "F59E0B", 0.3, 0},
	}
	if len(fills) != len(want) {
		t.Fatalf("도형 %d개, want %d", len(fills), len(want))
	}
	for i, w := range want {
		r, g, b := lumModOff(t, w.base, w.mod, w.off)
		gr, gg, gb := computedRGB(t, fills[i])
		if absInt(gr-r) > 1 || absInt(gg-g) > 1 || absInt(gb-b) > 1 {
			t.Errorf("%s: 웹 %s, PPTX(lumMod %v + lumOff %v) rgb(%d, %d, %d)", w.name, fills[i], w.mod, w.off, r, g, b)
		}
	}
}

// lumModOff는 OOXML `lumMod`/`lumOff`를 #RRGGBB에 적용한다 — RGB → HSL, L' = L·mod + off, → RGB.
func lumModOff(t *testing.T, hex string, mod, off float64) (r, g, b int) {
	t.Helper()
	hex = strings.TrimPrefix(hex, "#")
	var ri, gi, bi int
	if _, err := fmt.Sscanf(hex, "%02x%02x%02x", &ri, &gi, &bi); err != nil {
		t.Fatalf("색 %q: %v", hex, err)
	}
	rf, gf, bf := float64(ri)/255, float64(gi)/255, float64(bi)/255
	hi, lo := math.Max(rf, math.Max(gf, bf)), math.Min(rf, math.Min(gf, bf))
	l := (hi + lo) / 2
	var h, s float64
	if d := hi - lo; d > 0 {
		if l > 0.5 {
			s = d / (2 - hi - lo)
		} else {
			s = d / (hi + lo)
		}
		switch hi {
		case rf:
			h = math.Mod((gf-bf)/d+6, 6)
		case gf:
			h = (bf-rf)/d + 2
		default:
			h = (rf-gf)/d + 4
		}
		h /= 6
	}
	l = math.Min(1, math.Max(0, l*mod+off))
	channel := func(p, q, t float64) float64 {
		t = math.Mod(t+1, 1)
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
	q := l * (1 + s)
	if l >= 0.5 {
		q = l + s - l*s
	}
	p := 2*l - q
	to := func(v float64) int { return int(math.Round(v * 255)) }
	return to(channel(p, q, h+1.0/3)), to(channel(p, q, h)), to(channel(p, q, h-1.0/3))
}

// computedRGB는 계산된 색을 8비트 RGB로 읽는다. 상대 색 문법의 결과는 `rgb()`가 아니라
// `color(srgb r g b)`(0–1 실수)로 직렬화된다.
func computedRGB(t *testing.T, value string) (r, g, b int) {
	t.Helper()
	if _, err := fmt.Sscanf(value, "rgb(%d, %d, %d)", &r, &g, &b); err == nil {
		return r, g, b
	}
	var rf, gf, bf float64
	if _, err := fmt.Sscanf(value, "color(srgb %g %g %g)", &rf, &gf, &bf); err != nil {
		t.Fatalf("계산된 색 %q를 읽을 수 없다", value)
	}
	to := func(v float64) int { return int(math.Round(v * 255)) }
	return to(rf), to(gf), to(bf)
}

func absInt(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

// 잰 rect는 저자가 준 박스가 아니라 **그려진 그림**이다 — PPTX가 letterbox를 다시 계산하지
// 않는다. wide.png 1200×600, tall.png 세로 1:2, dot.png 16×8.
func TestMeasureImageFitRects(t *testing.T) {
	deck := imageDeck(t, `<row gap="md">
    <image src="assets/wide.png" fit="cover" w="480" h="240" alt="cover"/>
    <image src="assets/dot.png" fit="contain" w="480" h="240" alt="contain 확대 없음"/>
    <image src="assets/wide.png" fit="contain" w="8" h="8" alt="contain 축소"/>
    <image src="assets/tall.png" fit="contain" w="4" h="8" alt="contain 축소 세로"/>
  </row>`, `<row gap="md">
    <image src="assets/wide.png" fit="cover" w="240" h="120" pad="20" alt="여백 20"/>
    <image src="assets/dot.png" fit="contain" alt="고유 크기"/>
  </row>`)
	g := measureLayout(t, deck)

	for _, tc := range []struct {
		id   string
		name string
		w, h float64
	}{
		{"s0.0.0", "cover: 박스 = 그림", 480, 240},
		// `max-*`는 키우지 않는다 — 박스보다 작은 그림은 고유 크기로 남는다.
		{"s0.0.1", "contain, 박스가 그림보다 크다", 16, 8},
		{"s0.0.2", "contain, 폭이 걸린다", 8, 4},
		{"s0.0.3", "contain, 높이가 걸린다", 4, 8},
		// 4변 20px을 뺀 그림이다 — PPTX가 `xfrm`에서 여백을 다시 빼지 않는다.
		{"s1.0.0", "pad가 그림을 줄인다", 200, 80},
		// 디코드 전에 재면 고유 크기를 몰라 0×0이다.
		{"s1.0.1", "디코드를 기다린다", 16, 8},
	} {
		box := boxOf(t, g, tc.id)
		if box.El != soml.ImageElement || box.Rect.W != tc.w || box.Rect.H != tc.h {
			t.Errorf("%s(%s) = <%s> %s, want %v×%v", tc.id, tc.name, box.El, box.Rect, tc.w, tc.h)
		}
	}
	// 박스 div는 재지 않는다 — 그림마다 `pic` 하나다.
	if len(g.Boxes) != len(geom.Refs(deck)) {
		t.Errorf("박스 %d개, 덱 요소 %d개 — 박스 div가 측정에 섞였다", len(g.Boxes), len(geom.Refs(deck)))
	}
}

// `ratio` 박스는 그림 때문에 커지지 않는다 — flex/grid의 자동 최소 크기가 켜져 있으면 세로로
// 긴 그림의 칸이 558×1116이 된다 (imageCSS의 min-*: 0).
func TestMeasureImageRatioBoxIsNotGrownByPicture(t *testing.T) {
	deck := imageDeck(t, `<grid cols="2" gap="lg">
    <col gap="xs">
      <image src="assets/wide.png" fit="cover" w="fill" ratio="4:3" alt="넓은 그림"/>
      <text role="caption">cover · 넓은 그림</text>
    </col>
    <col gap="xs">
      <image src="assets/tall.png" fit="cover" w="fill" ratio="4:3" alt="좁은 그림"/>
      <text role="caption">cover · 좁은 그림</text>
    </col>
  </grid>`)
	g := measureLayout(t, deck)
	// (1152 − gap lg 36) / 2 = 558, 4:3이면 418.5.
	for _, id := range []string{"s0.0.0.0", "s0.0.1.0"} {
		if box := boxOf(t, g, id); box.Rect.W != 558 || box.Rect.H != 418.5 {
			t.Errorf("%s의 rect = %s, want 558×418.5", id, box.Rect)
		}
	}
}

// 로드 실패는 에러다 — 통과시키면 그림 없는 PPTX가 정상으로 열린다.
func TestOpenFailsOnBrokenImage(t *testing.T) {
	testkit.RequireChromium(t)
	deck := imageDeck(t, `<image src="assets/wide.png" fit="cover" w="200" h="100" alt="깨진 그림"/>`)
	path := renderHTML(t, deck, func(html string) string {
		return strings.Replace(html, `src="assets/wide.png"`, `src="assets/missing.png"`, 1)
	})
	err := testkit.SharedSession(t).Open(path)
	if err == nil || !strings.Contains(err.Error(), "assets/missing.png") {
		t.Errorf("깨진 그림의 에러 = %v, want 어느 src인지 말하는 에러", err)
	}
}

// `<stack>`은 가장 큰 자식 크기이고(단일 셀 grid — absolute였다면 0으로 붕괴한다) 자식이 같은
// 칸에 겹친다. `x`/`y` 자식은 stack 상대다 (stack이 포함 블록). `self="start"`는 stretch를 끈다.
func TestMeasureStackSizeAndOffset(t *testing.T) {
	deck := layoutDeck(t, `<stack self="start">
    <shape type="rect" fill="surface" w="300" h="200"/>
    <shape type="rect" fill="accent1" w="120" h="80"/>
    <text x="20" y="30">badge</text>
  </stack>`)
	g := measureLayout(t, deck)

	stack, big, small, badge := boxOf(t, g, "s0.0"), boxOf(t, g, "s0.0.0"), boxOf(t, g, "s0.0.1"), boxOf(t, g, "s0.0.2")
	if !near(stack.Rect.W, 300) || !near(stack.Rect.H, 200) {
		t.Errorf("stack 크기 = %s, want 300×200 (가장 큰 자식)", stack.Rect)
	}
	if !near(big.Rect.X, small.Rect.X) || !near(big.Rect.Y, small.Rect.Y) {
		t.Errorf("자식이 같은 칸에 겹치지 않는다: %s, %s", big.Rect, small.Rect)
	}
	if !near(badge.Rect.X, stack.Rect.X+20) || !near(badge.Rect.Y, stack.Rect.Y+30) {
		t.Errorf("x=20 y=30 자식 = %s, want stack 좌상단 + (20,30) (%s)", badge.Rect, stack.Rect)
	}
}

// `at` 9지점 — 축 순서(place-self: <세로> <가로>)가 뒤집히면 대각이 어긋나므로 전수로 잰다.
func TestMeasureStackNineAnchors(t *testing.T) {
	anchors := []string{
		"top-left", "top-center", "top-right",
		"center-left", "center", "center-right",
		"bottom-left", "bottom-center", "bottom-right",
	}
	var body strings.Builder
	body.WriteString(`<stack w="400" h="300">`)
	for _, at := range anchors {
		fmt.Fprintf(&body, "\n    <text at=%q>%s</text>", at, at)
	}
	body.WriteString("\n  </stack>")
	g := measureLayout(t, layoutDeck(t, body.String()))

	stack := boxOf(t, g, "s0.0")
	if !near(stack.Rect.W, 400) || !near(stack.Rect.H, 300) {
		t.Fatalf("stack 크기 = %s, want 400×300", stack.Rect)
	}
	// 축 위치 0·1·2 = 시작 변·중심·끝 변.
	pos := func(start, size float64, i int) float64 { return start + size*float64(i)/2 }
	for i, at := range anchors {
		child := boxOf(t, g, fmt.Sprintf("s0.0.%d", i))
		gotX := pos(child.Rect.X, child.Rect.W, i%3)
		gotY := pos(child.Rect.Y, child.Rect.H, i/3)
		wantX, wantY := pos(stack.Rect.X, stack.Rect.W, i%3), pos(stack.Rect.Y, stack.Rect.H, i/3)
		if math.Abs(gotX-wantX) > 1 || math.Abs(gotY-wantY) > 1 {
			t.Errorf("%s: (%v,%v), want (%v,%v) (%s)", at, gotX, gotY, wantX, wantY, child.Rect)
		}
	}
}

// 캔버스 직속 자식의 `x="0"`은 슬라이드 좌표 64다 — 포함 블록이 스테이지의 콘텐츠 상자(여백
// 안쪽)이므로 가장자리에 닿으려면 음수 좌표를 쓴다.
func TestMeasureCanvasChildCountsFromPaddingBox(t *testing.T) {
	deck := parseDeck(t, ` layout="canvas"`, `<text x="0" y="0">corner</text>
  <text x="100" y="50">inner</text>
  <text x="-64" y="-64">edge</text>`)
	g := measureLayout(t, deck)
	for _, tc := range []struct {
		id   string
		x, y float64
	}{{"s0.0", 64, 64}, {"s0.1", 164, 114}, {"s0.2", 0, 0}} {
		if box := boxOf(t, g, tc.id); !near(box.Rect.X, tc.x) || !near(box.Rect.Y, tc.y) {
			t.Errorf("%s = %s, want (%v,%v)", tc.id, box.Rect, tc.x, tc.y)
		}
	}
}

// 칠 순서 = 문서 순서 (뒤가 위, PPTX와 같다). 웹 기본은 위치 지정된 x/y 자식을 나중에 칠하므로
// stack 자식의 `position:relative`가 계약을 지킨다 — 그 한 줄을 빼면 색이 뒤집혀야 한다.
// 자식은 평범한 블록이다: 도형은 그 자체로 position:relative라 뒤집힘이 재현되지 않는다.
func TestStackPaintOrderIsDocumentOrder(t *testing.T) {
	testkit.RequireChromium(t)
	// 앞: x/y absolute 파랑(#0F65F9), 뒤: 정적 검정(#1A1A1A). 완전히 겹친다.
	deck := layoutDeck(t, `<stack w="200" h="200">
    <p x="0" y="0" style="width: 200px; height: 200px; background: #0F65F9"></p>
    <p style="width: 200px; height: 200px; background: #1A1A1A"></p>
  </stack>`)

	const (
		blue = "파랑(문서상 앞)"
		dark = "검정(문서상 뒤)"
	)
	classify := func(t *testing.T, transform func(string) string) string {
		t.Helper()
		browser := testkit.NewBrowser(t)
		browser.Load(t, renderHTML(t, deck, transform))
		img := browser.ScreenshotElement(t, web.ElementSelector())
		b := img.Bounds()
		r16, g16, bl16, _ := img.At(b.Min.X+b.Dx()/2, b.Min.Y+b.Dy()/2).RGBA()
		r, gg, bb := int(r16>>8), int(g16>>8), int(bl16>>8)
		if bb > 150 && r < 120 {
			return blue
		}
		if r < 80 && gg < 80 && bb < 80 {
			return dark
		}
		t.Fatalf("겹친 픽셀 색을 분류할 수 없다: rgb(%d,%d,%d)", r, gg, bb)
		return ""
	}

	if got := classify(t, nil); got != dark {
		t.Errorf("겹친 픽셀 = %s, want %s", got, dark)
	}
	stripped := classify(t, func(html string) string {
		return strings.Replace(html, "    grid-area: 1 / 1;\n    position: relative;\n", "    grid-area: 1 / 1;\n", 1)
	})
	if stripped != blue {
		t.Errorf("position:relative를 뺐는데 겹친 픽셀 = %s, want %s — 테스트가 계약을 지키지 않는다", stripped, blue)
	}
}

// 콘텐츠 박스는 슬라이드마다 잰다 — 여백은 테마가 레이아웃마다 정한다 (image-full은 padding 0).
func TestMeasureReadsPerSlideContentBox(t *testing.T) {
	stage, err := soml.Aspect16x9.Stage()
	if err != nil {
		t.Fatal(err)
	}
	deck := &soml.Deck{Title: "content box", Lang: "ko", Aspect: soml.Aspect16x9, Stage: stage}
	for i, layout := range []string{"content", "image-full"} {
		source := fmt.Sprintf("slides/%03d_%s%s", i+1, layout, soml.Ext)
		doc, diags := soml.Parse(source, fmt.Sprintf("<slide id=%q layout=%q>\n  <p>본문</p>\n</slide>\n", layout, layout))
		if err := diags.Err(); err != nil {
			t.Fatalf("파싱 실패: %v", err)
		}
		deck.Slides = append(deck.Slides, &soml.Slide{
			Index: i, ID: layout, Source: source,
			Layout: soml.SlideLayout(doc.Layout), Content: doc.Content,
		})
	}
	if err := core.Resolve(deck, nil).Err(); err != nil {
		t.Fatalf("해석 실패: %v", err)
	}

	g := measureLayout(t, deck)
	if got, want := g.ContentBox(0), (geom.Rect{X: 64, Y: 64, W: 1152, H: 592}); got != want {
		t.Errorf("content의 콘텐츠 박스 = %s, want %s", got, want)
	}
	if got, want := g.ContentBox(1), (geom.Rect{W: 1280, H: 720}); got != want {
		t.Errorf("image-full의 콘텐츠 박스 = %s, want %s", got, want)
	}
	for _, diag := range geom.Diagnose(deck, g) {
		if strings.Contains(diag.Message, "콘텐츠 박스") {
			t.Errorf("전면 배치에서 여백 침범 경고가 났다: %s", diag.String())
		}
	}
}

// 기동이 한 번 실패해도 세션이 뜬다 — 첫 실행에서만 죽는 가짜 브라우저가 진짜 앞에 선다.
func TestLaunchRetriesAFailedStart(t *testing.T) {
	real := testkit.RequireChromium(t)
	dir := t.TempDir()
	mark := filepath.Join(dir, "started-once")
	fake := filepath.Join(dir, "flaky-chrome")
	script := fmt.Sprintf("#!/bin/sh\nif [ ! -e %q ]; then : > %q; exit 1; fi\nexec %q \"$@\"\n", mark, mark, real)
	// 윈도우에는 sh가 없을 수 있고 실행 파일은 확장자가 정한다 — 같은 가짜를 배치 파일로 쓴다.
	if runtime.GOOS == "windows" {
		fake += ".cmd"
		script = fmt.Sprintf("@echo off\r\nif not exist \"%s\" (\r\n  type nul > \"%s\"\r\n  exit /b 1\r\n)\r\n\"%s\" %%*\r\n", mark, mark, real)
	}
	if err := os.WriteFile(fake, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	s, err := geom.Launch(fake)
	if err != nil {
		t.Fatalf("한 번 실패한 기동을 다시 시도하지 않았다: %v", err)
	}
	s.Close()
	if _, err := os.Stat(mark); err != nil {
		t.Fatalf("가짜 브라우저가 실행되지 않았다: %v", err)
	}
}

// 빈 셀도 한 줄을 차지한다. PPTX 셀은 비어 있어도 셀 글자 크기의 빈 문단 하나를 갖고
// PowerPoint는 행을 그 글자보다 낮게 줄이지 않는다 — 웹이 빈 행을 여백 + 테두리만큼으로 재면
// `a:tr@h`가 그 값을 싣고, PPTX 표만 행마다 한 줄씩 자라 뒤의 행들이 잰 rect 아래로 밀린다.
// 그래서 빈 행(태그만 · 공백만)은 글자 한 줄짜리 행과 같은 높이로 재여야 한다.
func TestMeasureEmptyTableRowHoldsOneLine(t *testing.T) {
	deck := layoutDeck(t, `<table>
    <tr><th>Topic</th><th>Description</th></tr>
    <tr><td></td><td></td></tr>
    <tr><td> </td><td></td></tr>
    <tr><td>x</td><td>y</td></tr>
  </table>`)
	g := measureLayout(t, deck)
	filled := boxOf(t, g, "s0.0.3")
	for _, id := range []string{"s0.0.1", "s0.0.2"} {
		if row := boxOf(t, g, id); !near(row.Rect.H, filled.Rect.H) {
			t.Errorf("빈 행 %s의 높이 = %v, 한 줄 행 = %v — 빈 셀이 줄 하나를 차지하지 않는다", id, row.Rect.H, filled.Rect.H)
		}
	}
}
