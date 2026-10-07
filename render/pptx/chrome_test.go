package pptx

import (
	"strings"
	"testing"

	"github.com/k2sebeom/balpyo/core"
	"github.com/k2sebeom/balpyo/render/geom"
	"github.com/k2sebeom/balpyo/soml"
)

// chromeDeck은 확정된 크롬이 실린 덱이다 — 렌더러는 제외 규칙도 번호 계산도 하지 않는다
// (ADR 0045). 콘텐츠도 하나 둔다: 크롬 도형이 콘텐츠 **뒤에** 오는지 보려면 둘이 있어야 한다.
func chromeDeck(t *testing.T) *soml.Deck {
	t.Helper()
	deck := resolvedContentDeck(t)
	style := soml.TextStyle{Role: soml.RoleCaption, Font: soml.FontBody, Size: 16, Weight: 400,
		LineHeight: 1.4, Color: soml.TokenColor("muted"), Align: soml.AlignStart}
	deck.Slides[0].Chrome = []soml.ChromeItem{
		{Kind: soml.ChromeFooter, Text: "2026 Q3 · 내부용", Align: soml.AlignStart, Style: style},
		{Kind: soml.ChromePageNumber, Text: "1", Align: soml.AlignEnd, Style: style},
	}
	return deck
}

// resolvedContentDeck은 외관이 해석된 콘텐츠 덱이다. 크롬은 **해석 뒤에** 싣는다 —
// core.Resolve는 설정이 켠 크롬만 확정하므로(여기서는 꺼져 있다) 먼저 실으면 지워진다.
func resolvedContentDeck(t *testing.T) *soml.Deck {
	t.Helper()
	deck := contentDeck()
	if err := core.Resolve(deck, nil).Err(); err != nil {
		t.Fatalf("외관 해석 실패: %v", err)
	}
	return deck
}

// 번호는 **필드**다 (ADR 0045): 정적 텍스트로 내면 뷰어에서 장을 더하거나 순서를 바꾼 순간
// 번호가 거짓말을 한다. `a:t`에 잰 숫자를 캐시로 싣는 이유는 필드를 계산하지 않는 뷰어(썸네일·
// 변환기)도 맞는 숫자를 보여야 한다는 것이다.
func TestChromePageNumberIsAField(t *testing.T) {
	deck := chromeDeck(t)
	xml := slideXML(t, deck, geom.Synthetic(deck, geom.Rect{X: 64, Y: 592, W: 120, H: 64}))
	if !strings.Contains(xml, `type="slidenum"`) {
		t.Fatalf("번호가 필드가 아니다:\n%s", xml)
	}
	field := sliceBetween(t, xml, "<a:fld", "</a:fld>")
	if !strings.Contains(field, "<a:t>1</a:t>") {
		t.Errorf("필드에 캐시된 숫자가 없다:\n%s", field)
	}
	if !strings.Contains(field, chromeFieldID) {
		t.Errorf("필드 id가 고정 GUID가 아니다:\n%s", field)
	}
	// 푸터는 필드가 아니라 보통 런이다 — 설정의 문자열이 그대로 나간다.
	footer := sliceBetween(t, xml, `name="chrome footer"`, "</p:sp>")
	if strings.Contains(footer, "<a:fld") {
		t.Errorf("푸터가 필드로 나갔다:\n%s", footer)
	}
	if !strings.Contains(footer, "<a:t>2026 Q3 · 내부용</a:t>") {
		t.Errorf("푸터 글자가 없다:\n%s", footer)
	}
}

// 크롬 도형은 **플레이스홀더가 아니다** (ADR 0045): 흐름 모드 슬라이드는 레이아웃 파트 하나를
// 공유하므로(slideLayoutFile) `p:ph`로는 "표지에는 번호가 없다"를 표현할 수 없다.
func TestChromeShapeIsNotAPlaceholder(t *testing.T) {
	deck := chromeDeck(t)
	xml := slideXML(t, deck, geom.Synthetic(deck, geom.Rect{X: 64, Y: 592, W: 120, H: 64}))
	number := sliceBetween(t, xml, `name="chrome pageNumber"`, "</p:sp>")
	if strings.Contains(number, "<p:ph") {
		t.Errorf("크롬이 플레이스홀더로 나갔다:\n%s", number)
	}
	// 세로 가운데 정렬은 웹의 `align-items: center`와 짝이다 (띠 전체가 rect이므로).
	if !strings.Contains(number, `anchor="ctr"`) {
		t.Errorf("크롬 bodyPr의 세로 앵커가 ctr이 아니다:\n%s", number)
	}
	// 웹이 한 줄로 두기로 한 띠를 PPTX가 접으면 띠를 넘쳐 본문 위로 올라간다.
	if !strings.Contains(number, `wrap="none"`) {
		t.Errorf("크롬 bodyPr가 wrap=none이 아니다:\n%s", number)
	}
}

// 잰 좌표가 없으면 **멈춘다** — 조용히 건너뛰면 웹에는 있고 PPTX에만 없는 크롬이 된다.
func TestChromeRequiresMeasuredRect(t *testing.T) {
	deck := chromeDeck(t)
	// 크롬 없는 덱에서 잰 지오메트리다 (요소 수는 같다).
	g := geom.Synthetic(resolvedContentDeck(t), geom.Rect{X: 64, Y: 64, W: 1152, H: 36})
	if _, err := NewWithGeometry(g).Render(deck); err == nil {
		t.Fatal("크롬 좌표가 없는데 렌더되었다")
	} else if !strings.Contains(err.Error(), "크롬") {
		t.Errorf("에러가 크롬을 가리키지 않는다: %v", err)
	}
}

// sliceBetween은 산출물에서 한 구간을 떼어낸다.
func sliceBetween(t *testing.T, xml, from, to string) string {
	t.Helper()
	start := strings.Index(xml, from)
	if start < 0 {
		t.Fatalf("%q를 찾을 수 없다:\n%s", from, xml)
	}
	rest := xml[start:]
	end := strings.Index(rest, to)
	if end < 0 {
		t.Fatalf("%q 뒤에서 %q를 찾을 수 없다:\n%s", from, to, rest)
	}
	return rest[:end]
}
