package web

import (
	"strings"
	"testing"

	"github.com/k2sebeom/balpyo/soml"
)

// renderTable는 덱의 첫 콘텐츠 노드(표 프레임)를 HTML로 쓴다.
//
// writeTable을 직접 부르지 않고 **writeNode를 지난다** — 슬라이드 껍데기 없이 조각만 보면서도
// 디스패치(writerFor)가 표를 받는다는 사실까지 함께 단정하는 자리다.
func renderTable(t *testing.T, deck *soml.Deck) string {
	t.Helper()
	slide := deck.Slides[0]
	elem, ok := slide.Content[0].(*soml.Element)
	if !ok || elem.Name != soml.TableElement {
		t.Fatalf("첫 콘텐츠 노드가 표가 아니다: %T", slide.Content[0])
	}
	var b strings.Builder
	if err := writeNode(&b, deck, slide, elem, []int{0}); err != nil {
		t.Fatalf("표 렌더 실패: %v", err)
	}
	return b.String()
}

// TestRenderTableIsARealTable은 표가 진짜 `<table>`/`<tr>`/`<th>`/`<td>`로 나오고 프레임·행·
// 셀이 **모두** 식별자를 갖는지 못 박는다 (열 너비·행 높이는 잰 셀 박스에서 온다 —
// 셀에 식별자가 없으면 PPTX 격자를 잴 수 없다).
func TestRenderTableIsARealTable(t *testing.T) {
	out := renderTable(t, subsetDeck(t, `  <table>
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
	if got := strings.Count(out, IDAttr+"="); got != 7 {
		t.Errorf("식별자 %d개, want 7 (프레임 · 행 2 · 셀 4)", got)
	}
}

// TestRenderTableCellAppearance는 셀의 확정 외관이 **인라인 style로 전부 명시**되는지 못
// 박는다 — 브라우저 기본 스타일(`th { font-weight: bold; text-align: center }`)이나 테마 표
// 스타일 상속이 셀 모양을 정할 자리를 남기지 않는다 (PPTX "명시 tcPr"의 웹 절반).
func TestRenderTableCellAppearance(t *testing.T) {
	out := renderTable(t, subsetDeck(t, `  <table>
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
	out := renderTable(t, subsetDeck(t, `  <table>
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
// 박는다 (배치 선언의 nil 계약과 같은 태도).
func TestRenderTableSpans(t *testing.T) {
	out := renderTable(t, subsetDeck(t, `  <table>
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
	out := renderTable(t, subsetDeck(t, `  <table variant="striped">
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
