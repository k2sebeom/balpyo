package pptx

import (
	"strings"
	"testing"

	"github.com/k2sebeom/balpyo/render/geom"
)

// 이 파일은 접근성 계약 둘을 못 박는다 (ADR 0058): 저자의 `alt`이 `p:cNvPr@descr`이 되는 것과,
// **읽기 순서가 문서 순서**라는 것. 둘 다 그림을 바꾸지 않으므로 픽셀 비교로는 잡히지 않고,
// 뒤집혀도 파일은 열린다 — 어서션이 유일한 감시자다.

// cNvPrLines는 슬라이드의 모든 `p:cNvPr` 줄이다 (도형·그림·표 프레임이 같은 요소를 쓴다).
// `spTree` 자신의 줄(`name=""`)은 요소가 아니므로 센 것에서 뺀다.
func cNvPrLines(t *testing.T, xml string) []string {
	t.Helper()
	var out []string
	for _, line := range strings.Split(xml, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "<p:cNvPr ") && !strings.Contains(line, `name=""`) {
			out = append(out, line)
		}
	}
	if len(out) == 0 {
		t.Fatalf("p:cNvPr가 없다:\n%s", xml)
	}
	return out
}

// 도형의 `alt`은 `descr`이 되고 **본문은 그대로 남는다** — PowerPoint에서 도형에 대체 텍스트를
// 넣어도 그 안의 글은 글이다. 웹이 잉크 svg에만 이름을 붙이는 것이 이 사실과 짝이다.
func TestShapeDescrIsTheAuthorsAlt(t *testing.T) {
	deck := resolvedDeck(t, `<slide id="a11y">
  <shape type="arrow-right" alt="배포 흐름의 마지막 단계" w="240" h="120"><p>배포</p></shape>
</slide>`)
	xml := slideXML(t, deck, geom.Synthetic(deck, geom.Rect{X: 64, Y: 64, W: 240, H: 120}))

	lines := cNvPrLines(t, xml)
	if len(lines) != 1 {
		t.Fatalf("p:cNvPr %d개, want 1개 (흡수 모드 도형은 도형 하나다):\n%s", len(lines), xml)
	}
	if !strings.Contains(lines[0], ` descr="배포 흐름의 마지막 단계"`) {
		t.Errorf("도형에 descr이 없다: %s", lines[0])
	}
	// `descr`은 이름을 **더한다** — 흡수한 본문을 지우지 않는다.
	if !strings.Contains(xml, "<a:t>배포</a:t>") {
		t.Errorf("도형의 본문이 사라졌다:\n%s", xml)
	}
}

// `alt`이 없으면 `descr`을 내지 않는다 — 그림의 규칙과 같다 (`descr=""`는 "장식"이 아니라 "빈
// 대체 텍스트"이고, 빈 속성은 재저장 diff의 잡음이 된다).
func TestShapeWithoutAltHasNoDescr(t *testing.T) {
	deck := resolvedDeck(t, `<slide id="a11y">
  <shape type="rect" w="240" h="120"/>
</slide>`)
	xml := slideXML(t, deck, geom.Synthetic(deck, geom.Rect{X: 64, Y: 64, W: 240, H: 120}))
	if strings.Contains(xml, "descr=") {
		t.Errorf("이름 없는 도형에 descr이 붙었다:\n%s", xml)
	}
}

// 표의 `alt`은 **프레임**의 `descr`이다 (`p:graphicFrame`의 `p:cNvPr`) — 셀은 `a:tbl` 안에
// 그대로 남아 항해 가능하다. 웹의 `<table aria-label>`과 같은 자리다.
func TestTableFrameDescrIsTheAuthorsAlt(t *testing.T) {
	deck := tableDeck(t, `<table alt="분기별 매출 표">
  <tr><th>분기</th><th>매출</th></tr>
  <tr><td>Q3</td><td>12억</td></tr>
</table>`)
	frame := renderTableFrame(t, deck, geom.Synthetic(deck, geom.Rect{X: 64, Y: 64, W: 600, H: 200}))

	lines := cNvPrLines(t, frame)
	if len(lines) != 1 {
		t.Fatalf("p:cNvPr %d개, want 1개 (행·셀은 프레임 안에 접힌다):\n%s", len(lines), frame)
	}
	if !strings.Contains(lines[0], ` descr="분기별 매출 표"`) {
		t.Errorf("표 프레임에 descr이 없다: %s", lines[0])
	}
	// 셀은 그대로다 — 이름은 격자 위에 얹히고 격자를 지우지 않는다.
	if got := strings.Count(frame, "<a:tc>") + strings.Count(frame, "<a:tc "); got < 4 {
		t.Errorf("a:tc %d개 — 표의 격자가 달라졌다:\n%s", got, frame)
	}
}

func TestTableWithoutAltHasNoDescr(t *testing.T) {
	deck := tableDeck(t, "<table>\n  <tr><td>Q3</td></tr>\n</table>")
	frame := renderTableFrame(t, deck, geom.Synthetic(deck, geom.Rect{X: 64, Y: 64, W: 600, H: 200}))
	if strings.Contains(frame, "descr=") {
		t.Errorf("이름 없는 표에 descr이 붙었다:\n%s", frame)
	}
}

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

// **읽기 순서는 문서 순서다** (ADR 0058) — 두 타깃이 그 순서를 공유한다.
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
