package web

import (
	"strings"
	"testing"

	"github.com/k2sebeom/balpyo/core"
	"github.com/k2sebeom/balpyo/soml"
)

// tableDeck는 표 하나를 담은 슬라이드를 파이프라인과 같은 순서(파싱 → 해석)로 덱으로 만든다
// (subsetDeck의 표 판 — 손으로 만든 트리 대신 원문을 쓰는 이유도 같다).
//
// 파서 진단에서 "아직 그리지 못하는 어휘"만 걸러내는 이유는 순서다: 두 렌더러가 **함께**
// 붙는 순간에만 `soml.unrenderedKinds`에서 표를 지울 수 있으므로(render/vocabulary_test.go의
// 양방향 대조), 웹 쪽 절반을 짜는 동안은 파서가 아직 표를 막는다. 이 필터는 그 플립에서
// 사라진다 — 그때 이 테스트들은 renderString을 그대로 쓴다.
func tableDeck(t *testing.T, body string) *soml.Deck {
	t.Helper()
	const source = "slides/001_table.slide"
	doc, diags := soml.Parse(source, "<slide id=\"table\">\n"+body+"\n</slide>\n")
	for _, d := range diags {
		if strings.Contains(d.Message, soml.UnrenderedVocabMessage) {
			continue
		}
		t.Fatalf("파싱 실패: %v", diags)
	}
	stage, err := soml.Aspect16x9.Stage()
	if err != nil {
		t.Fatal(err)
	}
	deck := &soml.Deck{
		Title: "표", Lang: "ko", Aspect: soml.Aspect16x9, Stage: stage,
		Slides: []*soml.Slide{{
			Index: 0, ID: "table", Source: source, Content: doc.Content,
		}},
	}
	if err := core.Resolve(deck).Err(); err != nil {
		t.Fatalf("외관 해석 실패: %v", err)
	}
	return deck
}

// renderTable는 덱의 첫 콘텐츠 노드(표 프레임)를 HTML로 쓴다 — 디스패치가 아직 표를 받지
// 않으므로 writer를 직접 부른다 (tableDeck의 주석과 같은 순서 근거).
func renderTable(t *testing.T, deck *soml.Deck) string {
	t.Helper()
	slide := deck.Slides[0]
	elem, ok := slide.Content[0].(*soml.Element)
	if !ok || elem.Name != soml.TableElement {
		t.Fatalf("첫 콘텐츠 노드가 표가 아니다: %T", slide.Content[0])
	}
	var b strings.Builder
	if err := writeTable(&b, deck, slide, elem, []int{0}); err != nil {
		t.Fatalf("표 렌더 실패: %v", err)
	}
	return b.String()
}

// TestRenderTableIsARealTable은 표가 진짜 `<table>`/`<tr>`/`<th>`/`<td>`로 나오고 프레임·행·
// 셀이 **모두** 식별자를 갖는지 못 박는다 (ADR 0036: 열 너비·행 높이는 잰 셀 박스에서 온다 —
// 셀에 식별자가 없으면 PPTX 격자를 잴 수 없다).
func TestRenderTableIsARealTable(t *testing.T) {
	out := renderTable(t, tableDeck(t, `  <table>
    <tr><th>지역</th><th>Q1</th></tr>
    <tr><td>서울</td><td>12.4</td></tr>
  </table>`))
	for _, want := range []string{
		`<table class="balpyo-table" data-balpyo-id="s0.0" data-balpyo-el="table"`,
		`<tr data-balpyo-id="s0.0.0" data-balpyo-el="tr"`,
		`<th data-balpyo-id="s0.0.0.0" data-balpyo-el="th"`,
		`<td data-balpyo-id="s0.0.1.1" data-balpyo-el="td"`,
		"지역", "12.4",
		"</table>",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("표 HTML에 %q가 없다:\n%s", want, out)
		}
	}
	// 식별자는 문서 순서다 — 지오메트리 패스가 같은 순서로 덱을 걸으며 대조한다 (geom.Refs).
	order := []string{"s0.0", "s0.0.0", "s0.0.0.0", "s0.0.0.1", "s0.0.1", "s0.0.1.0", "s0.0.1.1"}
	at := -1
	for _, id := range order {
		i := strings.Index(out, `data-balpyo-id="`+id+`"`)
		if i < 0 {
			t.Fatalf("식별자 %q가 없다:\n%s", id, out)
		}
		if i < at {
			t.Errorf("식별자 %q가 문서 순서를 벗어났다:\n%s", id, out)
		}
		at = i
	}
}

// TestRenderTableCellAppearance는 셀의 확정 외관이 **인라인 style로 전부 명시**되는지 못
// 박는다 — 브라우저 기본 스타일(`th { font-weight: bold; text-align: center }`)이나 테마 표
// 스타일 상속이 셀 모양을 정할 자리를 남기지 않는다 (ADR 0036의 "명시 tcPr"의 웹 절반).
func TestRenderTableCellAppearance(t *testing.T) {
	out := renderTable(t, tableDeck(t, `  <table>
    <tr><th>머리</th></tr>
    <tr><td anchor="bottom">본문</td></tr>
  </table>`))
	for _, want := range []string{
		// 헤더 셀: 테마 header 델타(채움 accent1 · 색 bg · 굵기 700) + 셀 기본 테두리.
		"background-color: var(--balpyo-color-accent1)",
		"color: var(--balpyo-color-bg)",
		"font-weight: 700",
		"border-width: 1px", "border-style: solid", "border-color: var(--balpyo-color-muted)",
		// 데이터 셀: body 역할 텍스트 + 앵커는 셀의 세로 정렬이다.
		"font-weight: 400",
		"vertical-align: bottom",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("셀 style에 %q가 없다:\n%s", want, out)
		}
	}
	// 앵커는 언제나 명시한다 — UA의 `td { vertical-align: middle }`가 이길 자리를 남기지 않는다.
	if n := strings.Count(out, "vertical-align:"); n != 2 {
		t.Errorf("vertical-align 선언 %d개, want 셀마다 하나(2개):\n%s", n, out)
	}
}

// TestRenderTableCellFillAndBorderNone은 `none` 두 가지가 **선언 없음이 아니라 명시**로
// 나가는지 못 박는다 — 테마가 얹은 헤더 채움·셀 테두리를 지우는 유일한 방법이므로, 선언을
// 내지 않으면 저자가 지운 것이 조용히 되살아난다.
func TestRenderTableCellFillAndBorderNone(t *testing.T) {
	out := renderTable(t, tableDeck(t, `  <table>
    <tr><th fill="none" border="none">머리</th></tr>
  </table>`))
	for _, want := range []string{"background-color: transparent", "border-style: none"} {
		if !strings.Contains(out, want) {
			t.Errorf("셀 style에 %q가 없다:\n%s", want, out)
		}
	}
	if strings.Contains(out, "border-width") {
		t.Errorf("border=none에 굵기가 나갔다:\n%s", out)
	}
}

// TestRenderTableSpans는 병합 칸 수가 HTML 속성으로 나가고 **1일 때는 나가지 않는지** 못
// 박는다 (기본값을 적으면 골든이 뜻 없는 속성으로 붐빈다 — 배치 선언의 nil 계약과 같은 태도).
func TestRenderTableSpans(t *testing.T) {
	out := renderTable(t, tableDeck(t, `  <table>
    <tr><td colspan="2">합친 칸</td><td rowspan="2">오른쪽</td></tr>
    <tr><td>a</td><td>b</td></tr>
  </table>`))
	for _, want := range []string{`colspan="2"`, `rowspan="2"`} {
		if !strings.Contains(out, want) {
			t.Errorf("표 HTML에 %q가 없다:\n%s", want, out)
		}
	}
	if strings.Contains(out, `colspan="1"`) || strings.Contains(out, `rowspan="1"`) {
		t.Errorf("병합하지 않은 셀에 span 속성이 나갔다:\n%s", out)
	}
}

// TestRenderTableFrameVariant는 프레임 변형이 CSS 훅(클래스)으로 나가는지 못 박는다 —
// 줄무늬 **색**은 셀 채움으로 이미 구워져 있고(core), 클래스는 저자 CSS의 표적이다.
func TestRenderTableFrameVariant(t *testing.T) {
	out := renderTable(t, tableDeck(t, `  <table variant="striped">
    <tr><td>a</td></tr>
    <tr><td>b</td></tr>
  </table>`))
	if !strings.Contains(out, `class="balpyo-table balpyo-table-striped"`) {
		t.Errorf("줄무늬 변형 클래스가 없다:\n%s", out)
	}
	// 줄무늬 채움은 셀에 명시로 실린다 (홀수 본문 행 = 둘째 행).
	if !strings.Contains(out, "background-color: var(--balpyo-color-surface)") {
		t.Errorf("줄무늬 채움이 셀에 실리지 않았다:\n%s", out)
	}
}

// TestTableCSSOnlyWhenUsed는 표 CSS 절이 표를 쓴 덱에만 나가는지 못 박는다 (골든 불변 —
// 리스트·도형 절과 같은 규칙: 쓰지 않은 어휘의 CSS가 들어가면 diff가 무엇이 변했는지
// 말해 주지 못한다).
func TestTableCSSOnlyWhenUsed(t *testing.T) {
	deck := tableDeck(t, "  <table>\n    <tr><td>a</td></tr>\n  </table>")
	feat := contentFeatures(deck)
	if !feat.tables {
		t.Fatal("표를 쓴 덱의 tables 특성이 거짓이다")
	}
	css := contentCSS(feat)
	for _, want := range []string{".balpyo-table", "border-collapse: collapse"} {
		if !strings.Contains(css, want) {
			t.Errorf("표 CSS에 %q가 없다:\n%s", want, css)
		}
	}
	// 표가 없는 덱에는 한 줄도 나가지 않는다.
	if plain := contentCSS(contentFeatures(contentDeck())); strings.Contains(plain, "balpyo-table") {
		t.Errorf("표 없는 덱에 표 CSS가 나갔다:\n%s", plain)
	}
}
