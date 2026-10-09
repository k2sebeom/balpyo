package pptx

import (
	"strings"
	"testing"

	"github.com/k2sebeom/balpyo/render/geom"
)

// 접근성 계약 중 패리티가 보지 않는 것: `descr` 이스케이프와, `order`가 읽기 순서를
// 바꾸지 않는다는 것. `descr`의 유무·값은 패리티(16_a11y)가 본다.

// XML 특수문자가 든 이름은 이스케이프된다 — 속성 안에서 인용부호가 깨지면 파일이 열리지 않는다
// (도형 이름·그림의 `descr`과 같은 escapeXML을 지난다).
func TestShapeAndTableDescrAreEscaped(t *testing.T) {
	deck := resolvedDeck(t, `<slide id="a11y">
  <shape type="rect" alt="a &amp; b &lt; c" w="240" h="120"/>
</slide>`)
	xml := slideXML(t, deck, geom.Synthetic(deck, geom.Rect{X: 64, Y: 64, W: 240, H: 120}))
	if !strings.Contains(xml, `descr="a &amp; b &lt; c"`) {
		t.Errorf("도형의 descr이 이스케이프되지 않았다:\n%s", xml)
	}

	table := tableDeck(t, `<table alt="a &amp; b &lt; c"><tr><td>x</td></tr></table>`)
	frame := renderTableFrame(t, table, geom.Synthetic(table, geom.Rect{X: 64, Y: 64, W: 600, H: 200}))
	if !strings.Contains(frame, `descr="a &amp; b &lt; c"`) {
		t.Errorf("표의 descr이 이스케이프되지 않았다:\n%s", frame)
	}
}

// **읽기 순서는 문서 순서다** — 두 타깃이 그 순서를 공유한다.
//
// `order`는 **시각** 순서를 바꾸는 채널이고(웹은 flex `order`, PPTX는 잰 좌표), 읽기 순서를
// 바꾸지 않는다: PPTX의 읽기 순서는 `spTree`의 도형 순서이고 그것은 조립기의 순회가 만든다.
// 언젠가 "도형을 z-순서로 정렬하자"는 최적화가 그 순회를 건드리면 읽기 순서가 **조용히**
// 뒤집힌다 — 그림은 같으므로 픽셀 비교도, 사람의 눈도 그것을 잡지 못한다.
func TestReadingOrderIsDocumentOrderInBothTargets(t *testing.T) {
	deck := resolvedDeck(t, `<slide id="order">
  <row gap="md">
    <p order="2">먼저 쓴 문단</p>
    <p order="-1">나중에 쓴 문단</p>
  </row>
</slide>`)
	g := geom.Synthetic(deck, geom.Rect{X: 64, Y: 64, W: 400, H: 60})
	xml := slideXML(t, deck, g)

	first := strings.Index(xml, "먼저 쓴 문단")
	last := strings.Index(xml, "나중에 쓴 문단")
	if first < 0 || last < 0 {
		t.Fatalf("두 문단이 산출물에 없다:\n%s", xml)
	}
	if first > last {
		t.Errorf("spTree의 도형 순서가 문서 순서와 다르다 — order가 읽기 순서를 뒤집었다:\n%s", xml)
	}
	// 같은 덱의 웹 DOM도 같은 순서다 (`order`는 CSS 선언으로만 나간다).
	html := renderWebString(t, deck)
	if strings.Index(html, "먼저 쓴 문단") > strings.Index(html, "나중에 쓴 문단") {
		t.Errorf("웹 DOM 순서가 문서 순서와 다르다:\n%s", html)
	}
	if !strings.Contains(html, "order: 2") {
		t.Errorf("order가 CSS 선언으로 나가지 않았다 — 시각 순서를 바꾸는 채널이 사라졌다:\n%s", html)
	}
}
