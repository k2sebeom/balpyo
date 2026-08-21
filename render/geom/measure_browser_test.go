// 이 파일만 브라우저를 띄운다 — 외부 도구 정책(없으면 로컬은 건너뛰고 테스트 이미지에서는
// 실패)은 internal/testkit이 소유하므로, 패키지 밖 테스트로 두어 testkit을 임포트한다
// (testkit → pipeline → render/pptx → render/geom 이므로 같은 패키지 안에서는 순환이 된다).
package geom_test

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/k2sebeom/balpyo/core"
	"github.com/k2sebeom/balpyo/internal/testkit"
	"github.com/k2sebeom/balpyo/render/geom"
	"github.com/k2sebeom/balpyo/render/web"
	"github.com/k2sebeom/balpyo/soml"
)

// longBody는 어떤 폰트에서도 1152px 안에서 여러 줄이 되는 길이다 — 줄 박스 수집을 검증하려면
// 실제로 줄이 나뉘어야 한다.
const longBody = "모놀리스 분리를 마쳤고, 배포 파이프라인을 자동화했으며, 관측성 스택(OpenTelemetry + Grafana)을 " +
	"도입했다. 하반기에는 멀티 리전 배포와 비용 최적화를 함께 진행하며, 팀별 서비스 경계를 다시 정리한다."

// browserDeck은 같은 내용의 슬라이드 두 장을 가진 덱이다. 두 장이 같아야 좌표가 슬라이드
// 상대인지 문서 상대인지가 드러난다 — 문서 상대라면 두 번째 슬라이드의 y가 크게 다르다.
func browserDeck(t *testing.T) *soml.Deck {
	t.Helper()
	stage, err := soml.Aspect16x9.Stage()
	if err != nil {
		t.Fatal(err)
	}
	content := func() []soml.Node {
		return []soml.Node{
			&soml.Element{
				Name:     soml.TextElement,
				Attrs:    []soml.Attr{{Name: "role", Value: "title", HasValue: true}},
				Children: []soml.Node{&soml.Text{Value: "2026 아키텍처 리뷰"}},
			},
			&soml.Element{
				Name:     soml.TextElement,
				Children: []soml.Node{&soml.Text{Value: longBody}},
			},
		}
	}
	deck := &soml.Deck{
		Title: "geometry", Lang: "ko", Aspect: soml.Aspect16x9, Stage: stage,
		Slides: []*soml.Slide{
			{Index: 0, ID: "first", Source: "slides/001_first.slide", Content: content()},
			{Index: 1, ID: "second", Source: "slides/002_second.slide", Content: content()},
		},
	}
	if diags := core.Resolve(deck); len(diags) > 0 {
		t.Fatalf("외관 해석 진단: %v", diags)
	}
	return deck
}

// renderHTML은 덱을 임시 디렉토리의 index.html로 쓴다 — 측정 대상은 사용자가 받는 것과 같은
// 바이트여야 한다.
func renderHTML(t *testing.T, deck *soml.Deck, transform func(string) string) string {
	t.Helper()
	files, err := web.New().Render(deck)
	if err != nil {
		t.Fatal(err)
	}
	html := string(files[web.IndexFile])
	if transform != nil {
		html = transform(html)
	}
	path := filepath.Join(t.TempDir(), web.IndexFile)
	if err := os.WriteFile(path, []byte(html), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestMeasureReadsSlideRelativeBoxes(t *testing.T) {
	testkit.RequireChromium(t)
	deck := browserDeck(t)
	path := renderHTML(t, deck, nil)

	g, err := geom.Measure(deck, path)
	if err != nil {
		t.Fatalf("지오메트리 패스 실패: %v", err)
	}
	if len(g.Boxes) != len(geom.Refs(deck)) {
		t.Fatalf("박스 %d개, 덱 요소 %d개", len(g.Boxes), len(geom.Refs(deck)))
	}

	title, ok := g.Box("s0.0")
	if !ok {
		t.Fatalf("s0.0이 없다: %+v", g.Boxes)
	}
	// 스테이지 여백 64px, 콘텐츠 폭 1280-128 = 1152px (render/web contentCSS). 논리 px가
	// 그대로 읽혀야 한다 — 여기가 어긋나면 PPTX의 모든 좌표가 틀린다.
	if title.Rect.X != 64 || title.Rect.Y != 64 || title.Rect.W != 1152 {
		t.Errorf("제목 박스 = %s, want (64,64 1152×…)", title.Rect)
	}

	// 두 번째 슬라이드는 문서에서 768px 아래에 있다. 좌표가 슬라이드 상대라면 두 슬라이드의
	// 박스가 완전히 같다 (같은 내용이므로).
	for _, id := range []string{"0", "1"} {
		first, ok1 := g.Box("s0." + id)
		second, ok2 := g.Box("s1." + id)
		if !ok1 || !ok2 {
			t.Fatalf("s0.%s/s1.%s를 찾을 수 없다", id, id)
		}
		if first.Rect != second.Rect {
			t.Errorf("자식 %s: 슬라이드 0 = %s, 슬라이드 1 = %s — 좌표가 슬라이드 상대가 아니다",
				id, first.Rect, second.Rect)
		}
		if second.Rect.Y >= float64(deck.Stage.Height) {
			t.Errorf("s1.%s의 y = %v — 스테이지 높이를 넘었다 (문서 좌표가 섞였다)", id, second.Rect.Y)
		}
	}

	// 세로 스택: 본문은 제목 아래 간격(24px)만큼 떨어진다.
	body, ok := g.Box("s0.1")
	if !ok {
		t.Fatal("s0.1이 없다")
	}
	if want := title.Rect.Y + title.Rect.H + 24; math.Abs(body.Rect.Y-want) > 0.01 {
		t.Errorf("본문 y = %v, want %v (제목 아래 24px)", body.Rect.Y, want)
	}
}

// 텍스트 메트릭: 줄 박스는 Iteration 1 리뷰 포인트(브라우저가 잰 박스가 다른 엔진에서 같은
// 줄바꿈을 유지하는가, PRD §13-1)의 수치 재료다.
func TestMeasureCollectsLineBoxes(t *testing.T) {
	testkit.RequireChromium(t)
	deck := browserDeck(t)
	g, err := geom.Measure(deck, renderHTML(t, deck, nil))
	if err != nil {
		t.Fatal(err)
	}

	body, ok := g.Box("s0.1")
	if !ok {
		t.Fatal("s0.1이 없다")
	}
	if body.LineCount() < 2 {
		t.Fatalf("줄 %d개 — 긴 본문은 여러 줄이어야 한다 (%s)", body.LineCount(), body.Rect)
	}
	// 줄 박스는 위에서 아래로, 박스 안에 들어 있어야 한다. 벗어나면 좌표계가 섞인 것이다.
	prev := -math.MaxFloat64
	for i, line := range body.Lines {
		if line.Y < prev {
			t.Errorf("줄 %d의 y = %v가 앞 줄(%v)보다 위다", i, line.Y, prev)
		}
		prev = line.Y
		if line.X < body.Rect.X-0.5 || line.X+line.W > body.Rect.X+body.Rect.W+0.5 {
			t.Errorf("줄 %d(%s)이 박스(%s) 밖이다", i, line, body.Rect)
		}
	}
	// 박스 높이 ≈ 줄 수 × 줄 간격. 이 관계가 성립해야 "박스 고정 + autofit off"로 내보낸
	// 박스가 텍스트를 담고, 줄이 하나 늘었을 때 얼마나 넘치는지가 이 간격으로 계산된다.
	advance := body.LineAdvance()
	if want := float64(body.LineCount()) * advance; math.Abs(body.Rect.H-want) > 1 {
		t.Errorf("박스 높이 = %v, 줄 %d개 × 간격 %v = %v", body.Rect.H, body.LineCount(), advance, want)
	}
	// 줄 박스는 글자 상자이므로 줄 간격보다 낮다 — 이 둘이 같아지면 잰 것이 글자 상자가
	// 아니라 줄 슬롯이라는 뜻이고, 그러면 겹치는 줄을 구분할 수 없다.
	if body.LineHeight() <= 0 || body.LineHeight() > advance {
		t.Errorf("줄 박스 높이 = %v, 줄 간격 = %v", body.LineHeight(), advance)
	}
	if slack := body.Slack(); slack < 0 || slack > body.Rect.W {
		t.Errorf("Slack = %v, 박스 폭 %v 안이어야 한다", slack, body.Rect.W)
	}
}

// 스테이지가 1:1이라는 전제는 확인해야 하는 사실이다 — 페이지가 조용히 스케일되면 모든
// 좌표가 틀린 채로 산출물까지 간다.
func TestMeasureRejectsScaledPage(t *testing.T) {
	testkit.RequireChromium(t)
	deck := browserDeck(t)
	path := renderHTML(t, deck, func(html string) string {
		return strings.Replace(html, "</head>", "<style>.balpyo-slide { zoom: 0.5; }</style>\n</head>", 1)
	})
	_, err := geom.Measure(deck, path)
	if err == nil {
		t.Fatal("스케일된 페이지는 에러여야 한다")
	}
	if !strings.Contains(err.Error(), "스케일") {
		t.Errorf("에러에 스케일 진단이 없다: %v", err)
	}
}

// 렌더된 HTML과 덱이 어긋나면 rect를 붙일 곳이 없다. 조용히 넘기지 않는다.
func TestMeasureRejectsDeckMismatch(t *testing.T) {
	testkit.RequireChromium(t)
	deck := browserDeck(t)
	path := renderHTML(t, deck, nil)

	extra := *deck
	extra.Slides = append(append([]*soml.Slide{}, deck.Slides...), &soml.Slide{
		Index: 2, ID: "third", Source: "slides/003_third.slide",
		Content: []soml.Node{&soml.Element{Name: soml.TextElement}},
	})
	if _, err := geom.Measure(&extra, path); err == nil {
		t.Error("HTML에 없는 슬라이드가 있는 덱은 에러여야 한다")
	}
}

// 서브셋 슬라이드의 측정 대상 계약: 식별자는 블록 종류(text·블록·리스트 컨테이너·li)에만
// 붙는다. Measure가 성공한다는 것 자체가 Refs ↔ querySelectorAll 자리 맞대기가 통과했다는
// 뜻이다 — 웹 렌더러가 인라인에 식별자를 붙이거나 Refs가 인라인을 모으면 여기서 어긋난다.
func TestMeasureSubsetSlide(t *testing.T) {
	testkit.RequireChromium(t)
	const source = "slides/001_subset.slide"
	doc, diags := soml.Parse(source, `<slide id="subset">
  <h2>올해의 세 가지 변화</h2>
  <p>배포 파이프라인 <strong>자동화</strong>와 <em>관측성</em>이 핵심이다.
  자세한 내용은 <a href="https://example.com/docs">레퍼런스</a>를 본다.</p>
  <ul>
    <li>모놀리스 분리 완료</li>
    <li>배포 파이프라인 <strong>자동화</strong></li>
    <li>관측성 스택 도입 (<code>otel</code> + Grafana)</li>
  </ul>
  <ol>
    <li>먼저 잰다</li>
    <li>그 다음 고친다<br/>고친 것을 다시 잰다</li>
  </ol>
  <pre>
func main() {
	fmt.Println("balpyo")
}
</pre>
</slide>`)
	if err := diags.Err(); err != nil {
		t.Fatalf("파싱 실패: %v", err)
	}
	stage, err := soml.Aspect16x9.Stage()
	if err != nil {
		t.Fatal(err)
	}
	deck := &soml.Deck{
		Title: "subset", Lang: "ko", Aspect: soml.Aspect16x9, Stage: stage,
		Slides: []*soml.Slide{{Index: 0, ID: "subset", Source: source, Content: doc.Content}},
	}
	if err := core.Resolve(deck).Err(); err != nil {
		t.Fatalf("외관 해석 실패: %v", err)
	}

	g, err := geom.Measure(deck, renderHTML(t, deck, nil))
	if err != nil {
		t.Fatalf("지오메트리 패스 실패: %v", err)
	}

	// 인라인·void는 측정 목록에 없어야 한다 — rect가 뜻을 갖지 않는 요소다.
	seen := map[string]bool{}
	for _, box := range g.Boxes {
		seen[box.El] = true
		if kind := soml.KindOf(box.El); kind == soml.KindInline {
			t.Errorf("인라인 <%s>(%s)가 측정 목록에 있다", box.El, box.ID)
		}
	}
	for _, el := range []string{"h2", "p", "ul", "ol", "li", "pre"} {
		if !seen[el] {
			t.Errorf("<%s>가 측정 목록에 없다", el)
		}
	}

	// li rect는 문서 순서대로 아래로 내려가고 겹치지 않는다 — 겹치면 리스트 CSS가 틀린
	// 것이고, 항목 08의 픽셀 비교가 갈라질 때 이 rect가 어느 항목부터 밀렸는지 말해 준다.
	var lis []geom.Box
	for _, box := range g.Boxes {
		if box.El == "li" {
			lis = append(lis, box)
		}
	}
	if len(lis) != 5 {
		t.Fatalf("li %d개, want 5", len(lis))
	}
	for i := 1; i < len(lis); i++ {
		prev, cur := lis[i-1], lis[i]
		if cur.Rect.Y < prev.Rect.Bottom()-0.5 {
			t.Errorf("li %s(%s)가 앞 항목 %s(%s)와 겹친다", cur.ID, cur.Rect, prev.ID, prev.Rect)
		}
	}
	// 항목 사이 간격은 0이다 (수직 리듬은 줄높이가 소유한다) — 같은 리스트 안에서 다음
	// 항목이 정확히 앞 항목의 바닥에서 시작한다.
	first, second := lis[0], lis[1]
	if math.Abs(second.Rect.Y-first.Rect.Bottom()) > 0.5 {
		t.Errorf("li 간격 = %v, want 0 (%s→%s)", second.Rect.Y-first.Rect.Bottom(), first.ID, second.ID)
	}
	// 마커 자리: li 텍스트는 리스트 왼쪽에서 ListTextIndent만큼 들어와 시작한다.
	ul, ok := g.Box("s0.2")
	if !ok {
		t.Fatal("s0.2(ul)가 없다")
	}
	if len(first.Lines) == 0 {
		t.Fatalf("첫 li에 줄 박스가 없다: %+v", first)
	}
	if want := ul.Rect.X + web.ListTextIndent; math.Abs(first.Lines[0].X-want) > 0.5 {
		t.Errorf("첫 li의 텍스트 시작 x = %v, want %v (ul.x + %d)", first.Lines[0].X, want, web.ListTextIndent)
	}
}

// layoutDeck는 레이아웃 슬라이드로 만든 해석된 덱이다 — 슬라이드마다 본문 하나.
//
// 원문을 파싱해 쓰는 이유는 픽스처와 같다: 파서·값 층이 실제로 채우는 모양을 재야 "웹이
// 낸 CSS가 브라우저에서 그 배치가 되는가"를 확인한 것이 된다 (손으로 만든 트리는 값 층을
// 건너뛴다).
func layoutDeck(t *testing.T, bodies ...string) *soml.Deck {
	t.Helper()
	stage, err := soml.Aspect16x9.Stage()
	if err != nil {
		t.Fatal(err)
	}
	deck := &soml.Deck{Title: "layout", Lang: "ko", Aspect: soml.Aspect16x9, Stage: stage}
	for i, body := range bodies {
		source := fmt.Sprintf("slides/%03d_layout%s", i+1, soml.Ext)
		id := fmt.Sprintf("layout%d", i)
		doc, diags := soml.Parse(source, fmt.Sprintf("<slide id=%q>\n%s\n</slide>\n", id, body))
		if err := diags.Err(); err != nil {
			t.Fatalf("파싱 실패: %v", err)
		}
		deck.Slides = append(deck.Slides, &soml.Slide{
			Index: i, ID: id, Source: source, Notes: doc.Notes, Content: doc.Content,
		})
	}
	if err := core.Resolve(deck).Err(); err != nil {
		t.Fatalf("해석 실패: %v", err)
	}
	return deck
}

// measureLayout은 레이아웃 덱을 재서 지오메트리를 준다.
//
// Measure가 성공한다는 것 자체가 **Refs ↔ querySelectorAll 자리 맞대기 통과**다 — 측정
// 대상 계약이 레이아웃 컨테이너·`<spacer>`까지 넓어졌으므로(ADR 0009의 후속), 웹 렌더러와
// geom.collect 중 한쪽만 컨테이너를 세면 여기서 개수·순서가 어긋나 실패한다.
func measureLayout(t *testing.T, deck *soml.Deck) *geom.Geometry {
	t.Helper()
	g, err := geom.Measure(deck, renderHTML(t, deck, nil))
	if err != nil {
		t.Fatalf("지오메트리 패스 실패: %v", err)
	}
	return g
}

// spaceToken은 테마 표의 공간 토큰 값이다 — 실측을 리터럴 수치가 아니라 **표**와 맞댄다
// (요소는 var() 참조로 그 표를 가리킨다).
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

// `<grid cols="4" gap="lg">` 한 줄이 카드 그리드의 좌표 계산 전부를 대체한다 (PRD §7.4) —
// TASKS Iteration 3이 명시한 장면이다. 카드 4개가 같은 y에, 등간격 x에, 토큰 값만큼
// 벌어져야 그 주장이 성립한다.
func TestMeasureGridCards(t *testing.T) {
	testkit.RequireChromium(t)
	deck := layoutDeck(t, `<grid cols="4" gap="lg">
    <p>8분</p>
    <p>99.95%</p>
    <p>12회</p>
    <p>3분</p>
  </grid>`)
	g := measureLayout(t, deck)

	grid := boxOf(t, g, "s0.0")
	cards := make([]geom.Box, 0, 4)
	for i := 0; i < 4; i++ {
		cards = append(cards, boxOf(t, g, fmt.Sprintf("s0.0.%d", i)))
	}
	gap := spaceToken(t, deck, "lg")
	// 같은 행이므로 y가 같다 — 하나라도 다르면 트랙이 아니라 줄이 나뉜 것이다.
	for i, card := range cards {
		if math.Abs(card.Rect.Y-cards[0].Rect.Y) > 0.5 {
			t.Errorf("카드 %d의 y = %v, 첫 카드 = %v (같은 행이어야 한다)", i, card.Rect.Y, cards[0].Rect.Y)
		}
	}
	// 등간격 + 간격이 토큰 값: 트랙 폭은 (컨테이너 폭 − 간격 3개) / 4다.
	wantWidth := (grid.Rect.W - 3*gap) / 4
	for i, card := range cards {
		if math.Abs(card.Rect.W-wantWidth) > 0.5 {
			t.Errorf("카드 %d의 폭 = %v, want %v (균등 트랙)", i, card.Rect.W, wantWidth)
		}
		if want := grid.Rect.X + float64(i)*(wantWidth+gap); math.Abs(card.Rect.X-want) > 0.5 {
			t.Errorf("카드 %d의 x = %v, want %v (gap %v = 토큰 lg)", i, card.Rect.X, want, gap)
		}
	}
	// 첫·둘째 카드 사이의 실제 여백이 토큰 값이다 — var(--balpyo-space-lg)가 그 표를
	// 가리키고 있다는 실측 확인이다 (참조와 선언이 한 표에서 나온다는 계약의 런타임 절반).
	if got := cards[1].Rect.X - cards[0].Rect.Right(); math.Abs(got-gap) > 0.5 {
		t.Errorf("카드 사이 여백 = %v, want %v (공간 토큰 lg)", got, gap)
	}
}

// `<row>`의 gap과 `justify="between"` — 주축 분배가 CSS의 space-between으로 번역되는지의
// 실측이다 (어휘 이름을 렌더러가 되돌려 붙인다).
func TestMeasureRowGapAndJustify(t *testing.T) {
	testkit.RequireChromium(t)
	deck := layoutDeck(t,
		`<row gap="md" w="fill"><p basis="200">왼쪽</p><p basis="200">오른쪽</p></row>`,
		`<row justify="between" w="fill"><p basis="200">왼쪽</p><p basis="200">오른쪽</p></row>`)
	g := measureLayout(t, deck)

	gapRow, left, right := boxOf(t, g, "s0.0"), boxOf(t, g, "s0.0.0"), boxOf(t, g, "s0.0.1")
	gap := spaceToken(t, deck, "md")
	if got := right.Rect.X - left.Rect.Right(); math.Abs(got-gap) > 0.5 {
		t.Errorf("gap=md의 실제 간격 = %v, want %v", got, gap)
	}
	if math.Abs(left.Rect.X-gapRow.Rect.X) > 0.5 {
		t.Errorf("첫 자식의 x = %v, want %v (기본 분배는 start)", left.Rect.X, gapRow.Rect.X)
	}

	// justify=between: 첫 자식은 왼쪽 끝, 마지막 자식은 오른쪽 끝에 붙는다.
	betweenRow := boxOf(t, g, "s1.0")
	first, last := boxOf(t, g, "s1.0.0"), boxOf(t, g, "s1.0.1")
	if math.Abs(first.Rect.X-betweenRow.Rect.X) > 0.5 {
		t.Errorf("between의 첫 자식 x = %v, want %v", first.Rect.X, betweenRow.Rect.X)
	}
	if math.Abs(last.Rect.Right()-betweenRow.Rect.Right()) > 0.5 {
		t.Errorf("between의 마지막 자식 오른쪽 = %v, want %v", last.Rect.Right(), betweenRow.Rect.Right())
	}
	// 자식 폭 합이 컨테이너보다 작으므로 사이가 벌어져 있어야 한다 — 붙어 있으면 분배가
	// 적용되지 않은 것이다.
	if last.Rect.X <= first.Rect.Right() {
		t.Errorf("between인데 자식이 붙어 있다 (%s, %s)", first.Rect, last.Rect)
	}
}

// `<spacer/>`는 빈 요소로 뒤 자식을 컨테이너 끝으로 밀어낸다 — PPTX에 대응 도형이 없는
// 요소가 좌표로는 살아 있다는 뜻이고, 그것이 "배치는 좌표로 구워진다"의 실측이다.
func TestMeasureSpacerPushesSiblings(t *testing.T) {
	testkit.RequireChromium(t)
	deck := layoutDeck(t,
		`<row gap="sm" w="fill"><p>왼쪽 꼬리글</p><spacer/><p>오른쪽 페이지 번호</p></row>`,
		"<p>머리글</p>\n  <spacer/>\n  <p>바닥에 붙는 꼬리글</p>")
	g := measureLayout(t, deck)

	row, tail := boxOf(t, g, "s0.0"), boxOf(t, g, "s0.0.2")
	if math.Abs(tail.Rect.Right()-row.Rect.Right()) > 0.5 {
		t.Errorf("spacer 뒤 자식의 오른쪽 = %v, want %v (컨테이너 끝)", tail.Rect.Right(), row.Rect.Right())
	}
	spacer := boxOf(t, g, "s0.0.1")
	if spacer.Rect.W <= 0 {
		t.Errorf("spacer의 폭 = %v — 밀어내지 못했다", spacer.Rect.W)
	}
	if spacer.El != soml.SpacerElement {
		t.Errorf("s0.0.1의 태그 = %q, want %q", spacer.El, soml.SpacerElement)
	}

	// 슬라이드 직속 spacer는 세로 스택에서 밀어낸다 — 꼬리글이 스테이지 아래쪽에 붙는다.
	footer := boxOf(t, g, "s1.2")
	stageBottom := float64(deck.Stage.Height) - 64 // stagePadding (render/web contentCSS)
	if math.Abs(footer.Rect.Bottom()-stageBottom) > 0.5 {
		t.Errorf("꼬리글의 바닥 = %v, want %v (스테이지 콘텐츠 바닥)", footer.Rect.Bottom(), stageBottom)
	}
}

// 중첩이 처음 생기는 iteration이므로 못 박는다: 컨테이너 안 자식의 rect도 **슬라이드
// 상대**다 (getBoundingClientRect에서 슬라이드 rect를 빼므로 중첩 깊이와 무관하다 —
// measure.go). 깨져 있으면 PPTX 좌표가 컨테이너 안에서만 어긋나고, 그 실패는 산출물을
// 열어 보기 전까지 조용하다.
func TestMeasureNestedLayoutIsSlideRelative(t *testing.T) {
	testkit.RequireChromium(t)
	body := `<col gap="md" pad="lg" w="fill">
    <row gap="sm">
      <text>중첩된 텍스트</text>
    </row>
  </col>`
	// 같은 내용의 슬라이드 두 장 — 문서 상대 좌표라면 둘째 슬라이드의 y가 크게 다르다.
	deck := layoutDeck(t, body, body)
	g := measureLayout(t, deck)

	for _, id := range []string{"0", "0.0", "0.0.0"} {
		first, second := boxOf(t, g, "s0."+id), boxOf(t, g, "s1."+id)
		if first.Rect != second.Rect {
			t.Errorf("%s: 슬라이드 0 = %s, 슬라이드 1 = %s — 중첩 좌표가 슬라이드 상대가 아니다",
				id, first.Rect, second.Rect)
		}
		if second.Rect.Y >= float64(deck.Stage.Height) {
			t.Errorf("s1.%s의 y = %v — 스테이지 높이를 넘었다 (문서 좌표가 섞였다)", id, second.Rect.Y)
		}
	}

	// 중첩은 안쪽으로 들어간다: col의 pad(lg)만큼 row가 안으로, 그 안에 텍스트가 있다.
	col, row, text := boxOf(t, g, "s0.0"), boxOf(t, g, "s0.0.0"), boxOf(t, g, "s0.0.0.0")
	pad := spaceToken(t, deck, "lg")
	if want := col.Rect.X + pad; math.Abs(row.Rect.X-want) > 0.5 {
		t.Errorf("row의 x = %v, want %v (col.x + pad lg)", row.Rect.X, want)
	}
	if text.Rect.X < row.Rect.X-0.5 || text.Rect.Right() > row.Rect.Right()+0.5 {
		t.Errorf("텍스트(%s)가 부모 row(%s) 밖이다", text.Rect, row.Rect)
	}
	if text.Rect.Y < row.Rect.Y-0.5 || text.Rect.Bottom() > row.Rect.Bottom()+0.5 {
		t.Errorf("텍스트(%s)가 부모 row(%s)의 세로 범위 밖이다", text.Rect, row.Rect)
	}
	// 컨테이너도 측정 목록에 있다 (측정 대상 계약이 넓어졌다) — 픽셀 비교가 갈라질 때
	// "어느 칼럼부터 밀렸는가"를 이 rect가 말해 준다.
	seen := map[string]bool{}
	for _, box := range g.Boxes {
		seen[box.El] = true
	}
	for _, el := range []string{soml.ColElement, soml.RowElement, soml.TextElement} {
		if !seen[el] {
			t.Errorf("<%s>가 측정 목록에 없다", el)
		}
	}
}

// **회전 도형의 잰 rect는 회전 전 박스다** — 이 iteration에서 가장 조용한 함정의 실측이다.
//
// `getBoundingClientRect`는 회전된 외곽 bbox를 주므로(45도에서 1.41배) 그대로 재면 PPTX의
// 도형만 커진다: OOXML `xfrm@rot`은 회전 **전** `ext`를 중심 회전한 것이다. 측정 패스는
// 측정 뷰를 켜서 회전 선언을 무효화하고(web.MeasureView), 측정이 끝나면 되돌린다 — 두 절반을
// 한 테스트에서 본다: 잰 값은 회전 전, 되돌린 뒤 읽은 값은 회전된 bbox여야 한다.
func TestMeasureRotatedShapeIsUnrotatedBox(t *testing.T) {
	bin := testkit.RequireChromium(t)
	deck := layoutDeck(t, `<row gap="md">
    <shape type="rect" fill="accent1" w="120" h="120" rotate="45"/>
    <shape type="rect" fill="accent1" w="120" h="120"/>
  </row>`)
	path := renderHTML(t, deck, nil)

	session, err := geom.Launch(bin)
	if err != nil {
		t.Fatalf("브라우저를 띄울 수 없다: %v", err)
	}
	defer session.Close()
	g, err := session.Measure(deck, path)
	if err != nil {
		t.Fatalf("지오메트리 패스 실패: %v", err)
	}

	rotated, plain := boxOf(t, g, "s0.0.0"), boxOf(t, g, "s0.0.1")
	if math.Abs(rotated.Rect.W-120) > 0.5 || math.Abs(rotated.Rect.H-120) > 0.5 {
		t.Errorf("회전 도형의 잰 박스 = %s, want 120×120 (회전된 bbox는 169.7이다)", rotated.Rect)
	}
	if rotated.Rect.W != plain.Rect.W || rotated.Rect.H != plain.Rect.H {
		t.Errorf("회전 도형 %s와 회전 없는 도형 %s의 크기가 다르다", rotated.Rect, plain.Rect)
	}
	// 같은 행이므로 y가 같고, 회전 도형이 먼저이므로 x가 앞이다 — 변형은 배치를 바꾸지 않는다.
	if math.Abs(rotated.Rect.Y-plain.Rect.Y) > 0.5 {
		t.Errorf("회전 도형의 y = %v, 회전 없는 도형 = %v (변형이 배치를 바꿨다)", rotated.Rect.Y, plain.Rect.Y)
	}

	// 측정이 끝나면 문서 뷰로 돌아와 있어야 한다 — 남아 있으면 그 뒤에 찍는 스크린샷이
	// 산출물이 아니라 측정 배치를 찍는다.
	var view string
	if err := session.Eval(fmt.Sprintf("document.documentElement.getAttribute(%q) || \"\"", web.ViewAttr), &view); err != nil {
		t.Fatal(err)
	}
	if view != "" {
		t.Errorf("측정 뒤 뷰 속성이 %q로 남았다 — 되돌리기가 되지 않았다", view)
	}
	// 되돌린 뷰에서는 회전이 살아 있다: 같은 요소의 rect가 회전된 bbox(1.41배)다. 이 절반이
	// 없으면 "회전을 아예 렌더하지 않는" 구현도 위 단정을 통과한다.
	rects, err := session.Rects(web.ElementSelector())
	if err != nil {
		t.Fatal(err)
	}
	// 문서 순서: row, 회전 도형, 회전 없는 도형.
	if len(rects) != 3 {
		t.Fatalf("측정 대상 %d개, want 3 (row + 도형 둘)", len(rects))
	}
	if want := 120 * math.Sqrt2; math.Abs(rects[1].W-want) > 1 {
		t.Errorf("발표 뷰에서 회전 도형의 bbox 폭 = %v, want %v (회전이 적용되지 않았다)", rects[1].W, want)
	}
	if math.Abs(rects[2].W-120) > 0.5 {
		t.Errorf("회전 없는 도형의 bbox 폭 = %v, want 120", rects[2].W)
	}
}

// 도형 안 텍스트: `pad`만큼 안쪽이고, `anchor` 3종이 세로 위치를 실제로 바꾼다.
//
// 앵커가 **보이는 유일한 자리가 도형 안**이므로(soml.AnchorAttr) 여기가 그 계약의 유일한
// 실측이다. `pad`가 잰 박스를 키우지 않는 것도 함께 본다 — `box-sizing: border-box`가 아니면
// 도형 rect가 여백만큼 커져 PPTX 좌표가 밀린다.
func TestMeasureShapePadAndAnchor(t *testing.T) {
	testkit.RequireChromium(t)
	deck := layoutDeck(t, `<row gap="md" align="stretch">
    <shape type="rect" fill="surface" w="240" h="220" pad="md"><p anchor="top">위</p></shape>
    <shape type="rect" fill="surface" w="240" h="220" pad="md"><p anchor="center">가운데</p></shape>
    <shape type="rect" fill="surface" w="240" h="220" pad="md"><p anchor="bottom">아래</p></shape>
    <shape type="rect" fill="surface" w="240" h="220"><p anchor="top">여백 없음</p></shape>
  </row>`)
	g := measureLayout(t, deck)

	pad := spaceToken(t, deck, "md")
	top, center, bottom := boxOf(t, g, "s0.0.0"), boxOf(t, g, "s0.0.1"), boxOf(t, g, "s0.0.2")
	for i, shape := range []geom.Box{top, center, bottom} {
		// 잰 박스는 저자가 준 크기 그대로다 — 여백은 안쪽으로 들어간다.
		if math.Abs(shape.Rect.W-240) > 0.5 || math.Abs(shape.Rect.H-220) > 0.5 {
			t.Errorf("도형 %d의 박스 = %s, want 240×220 (pad가 박스를 키웠다)", i, shape.Rect)
		}
	}
	topText := boxOf(t, g, "s0.0.0.0")
	if want := top.Rect.X + pad; math.Abs(topText.Rect.X-want) > 0.5 {
		t.Errorf("도형 안 텍스트의 x = %v, want %v (도형 x + pad md)", topText.Rect.X, want)
	}
	if want := top.Rect.Y + pad; math.Abs(topText.Rect.Y-want) > 0.5 {
		t.Errorf("anchor=top 텍스트의 y = %v, want %v (도형 y + pad md)", topText.Rect.Y, want)
	}
	centerText, bottomText := boxOf(t, g, "s0.0.1.0"), boxOf(t, g, "s0.0.2.0")
	// 가운데: 위아래 여백이 같다 (도형 안쪽 높이의 절반에 텍스트 절반).
	above := centerText.Rect.Y - (center.Rect.Y + pad)
	below := (center.Rect.Bottom() - pad) - centerText.Rect.Bottom()
	if math.Abs(above-below) > 0.5 {
		t.Errorf("anchor=center 텍스트의 위 여백 %v ≠ 아래 여백 %v", above, below)
	}
	if above <= 0 {
		t.Errorf("anchor=center 텍스트가 위로 붙었다 (위 여백 %v)", above)
	}
	// 아래: 텍스트 바닥이 도형 안쪽 바닥이다.
	if want := bottom.Rect.Bottom() - pad; math.Abs(bottomText.Rect.Bottom()-want) > 0.5 {
		t.Errorf("anchor=bottom 텍스트의 바닥 = %v, want %v", bottomText.Rect.Bottom(), want)
	}
	// 세 앵커가 서로 다른 y를 만든다 — 선언이 나갔지만 먹지 않는 상태를 잡는다.
	if !(topText.Rect.Y < centerText.Rect.Y && centerText.Rect.Y < bottomText.Rect.Y) {
		t.Errorf("앵커가 세로 위치를 바꾸지 않았다: top %v, center %v, bottom %v",
			topText.Rect.Y, centerText.Rect.Y, bottomText.Rect.Y)
	}
	// 여백 없는 도형은 글자가 변에 붙는다 (pad의 유무가 그림에서 보이는 차이다).
	none, noneText := boxOf(t, g, "s0.0.3"), boxOf(t, g, "s0.0.3.0")
	if math.Abs(noneText.Rect.X-none.Rect.X) > 0.5 {
		t.Errorf("pad 없는 도형의 텍스트 x = %v, want %v (변에 붙는다)", noneText.Rect.X, none.Rect.X)
	}
}

// `<grid cols="4">` 안 도형 카드가 같은 y·등간격 x다 — 03_layout이 텍스트로 세운 계약
// (TestMeasureGridCards)이 도형에서도 성립하는지 본다. 도형은 그 자체가 그림이므로 카드
// 그리드가 이 iteration의 실제 용도다 (`_04_shape/006_cards`).
//
// Measure가 성공한다는 것 자체가 **Refs ↔ querySelectorAll 자리 맞대기 통과**다 — 측정 대상
// 계약이 `<shape>`와 도형 안 텍스트까지 넓어졌으므로, 웹 렌더러와 geom.collect 중 한쪽만
// 도형을 세면 여기서 개수·순서가 어긋나 실패한다.
func TestMeasureShapeGridCards(t *testing.T) {
	testkit.RequireChromium(t)
	deck := layoutDeck(t, `<grid cols="4" gap="lg">
    <shape type="rounded-rect" fill="surface" w="fill" h="160" pad="md"><p>공정성</p></shape>
    <shape type="rounded-rect" fill="surface" w="fill" h="160" pad="md"><p>신뢰성</p></shape>
    <shape type="rounded-rect" fill="surface" w="fill" h="160" pad="md"><p>투명성</p></shape>
    <shape type="rounded-rect" fill="surface" w="fill" h="160" pad="md"><p>책임성</p></shape>
  </grid>`)
	g := measureLayout(t, deck)

	grid := boxOf(t, g, "s0.0")
	cards := make([]geom.Box, 0, 4)
	for i := 0; i < 4; i++ {
		cards = append(cards, boxOf(t, g, fmt.Sprintf("s0.0.%d", i)))
	}
	gap := spaceToken(t, deck, "lg")
	wantWidth := (grid.Rect.W - 3*gap) / 4
	for i, card := range cards {
		if card.El != soml.ShapeElement {
			t.Errorf("카드 %d의 태그 = %q, want %q", i, card.El, soml.ShapeElement)
		}
		if math.Abs(card.Rect.Y-cards[0].Rect.Y) > 0.5 {
			t.Errorf("카드 %d의 y = %v, 첫 카드 = %v (같은 행이어야 한다)", i, card.Rect.Y, cards[0].Rect.Y)
		}
		if math.Abs(card.Rect.H-160) > 0.5 {
			t.Errorf("카드 %d의 높이 = %v, want 160", i, card.Rect.H)
		}
		if math.Abs(card.Rect.W-wantWidth) > 0.5 {
			t.Errorf("카드 %d의 폭 = %v, want %v (w=fill이 트랙 폭이다)", i, card.Rect.W, wantWidth)
		}
		if want := grid.Rect.X + float64(i)*(wantWidth+gap); math.Abs(card.Rect.X-want) > 0.5 {
			t.Errorf("카드 %d의 x = %v, want %v (gap %v = 토큰 lg)", i, card.Rect.X, want, gap)
		}
	}
	// 도형 안 텍스트도 잰다 (PPTX가 소비하지 않는 rect — `<li>`와 같은 처지다). 자기 도형
	// 안에 들어 있어야 앵커·여백이 제대로 걸린 것이다.
	for i := range cards {
		text := boxOf(t, g, fmt.Sprintf("s0.0.%d.0", i))
		if text.Rect.X < cards[i].Rect.X-0.5 || text.Rect.Right() > cards[i].Rect.Right()+0.5 {
			t.Errorf("카드 %d의 텍스트(%s)가 도형(%s) 밖이다", i, text.Rect, cards[i].Rect)
		}
	}
}

// 그림의 외관 선언이 브라우저에서 **실제로 값이 되는지** 본다.
//
// 이 확인이 필요한 이유는 침묵이다: `var()`는 정의되지 않은 변수를 조용히 무시하고, 무효한
// `filter` 값은 `none`이 된다 — 토큰 표와 참조가 어긋나거나 `color-mix()`를 못 쓰는 엔진에서
// 그림자·색이 사라지고 아무 에러도 나지 않는다. 렌더 단위 테스트는 문자열까지만 보므로
// 여기가 그 계약의 런타임 절반이다.
func TestMeasureShapeInkStylesResolve(t *testing.T) {
	testkit.RequireChromium(t)
	deck := layoutDeck(t, `<row gap="md">
    <shape type="rect" fill="accent1" border="3 solid muted" shadow="soft" opacity="0.4" w="180" h="120"/>
  </row>`)
	path := renderHTML(t, deck, nil)
	browser := testkit.NewBrowser(t)
	browser.Load(t, path)

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

	// 토큰 참조가 테마 표의 색으로 풀린다 (accent1 #0F65F9, muted #6B7280).
	if got.Fill != "rgb(15, 101, 249)" {
		t.Errorf("채움 = %q, want rgb(15, 101, 249) — 색 변수가 풀리지 않았다", got.Fill)
	}
	if got.Stroke != "rgb(107, 114, 128)" {
		t.Errorf("테두리 색 = %q, want rgb(107, 114, 128)", got.Stroke)
	}
	if got.Width != "3px" {
		t.Errorf("테두리 굵기 = %q, want 3px", got.Width)
	}
	// 그림자 토큰: 무효한 값이면 계산값이 none이다. dist 4 · dir 90 → 아래로 4px.
	if got.Filter == "none" || !strings.Contains(got.Filter, "drop-shadow") {
		t.Errorf("그림자 filter = %q — 토큰 변수나 color-mix()가 풀리지 않았다", got.Filter)
	}
	if !strings.Contains(got.Filter, "0px 4px 16px") {
		t.Errorf("그림자 값 = %q, want 오프셋 0px 4px · 번짐 16px (극좌표 dist 4 · dir 90)", got.Filter)
	}
	if got.Opacity != "0.4" {
		t.Errorf("불투명도 = %q, want 0.4", got.Opacity)
	}
}
