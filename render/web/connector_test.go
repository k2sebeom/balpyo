package web

import (
	"strings"
	"testing"

	"github.com/k2sebeom/balpyo/core"
	"github.com/k2sebeom/balpyo/render/route"
	"github.com/k2sebeom/balpyo/soml"
)

func connectorTestDeck(t *testing.T) *soml.Deck {
	t.Helper()
	const source = "slides/001_flow.slide"
	doc, diags := soml.Parse(source, `<slide id="flow" layout="canvas">
  <shape type="rect" id="a" w="100" h="100" x="0" y="0"/>
  <shape type="rect" id="b" w="100" h="100" x="300" y="0"/>
  <connector from="a" to="b" variant="dashed"/>
</slide>`)
	if err := diags.Err(); err != nil {
		t.Fatal(err)
	}
	stage, _ := soml.Aspect16x9.Stage()
	deck := &soml.Deck{Title: "flow", Aspect: soml.Aspect16x9, Stage: stage,
		Slides: []*soml.Slide{{ID: "flow", Source: source, Content: doc.Content, Layout: soml.LayoutCanvas}}}
	if err := core.Resolve(deck, nil).Err(); err != nil {
		t.Fatal(err)
	}
	return deck
}

// 렌더러는 세 상태다: 측정용은 연결선을 내지 않고, 잰 박스를 받으면 굽고, 둘 다 아니면 에러다 —
// 조용히 선을 빼지 않는다.
func TestConnectorRenderModes(t *testing.T) {
	deck := connectorTestDeck(t)
	measured, err := NewForMeasurement().Render(deck)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(measured[IndexFile]), ConnectorClass+`"`) {
		t.Error("측정용 HTML에 연결선 오버레이가 있다")
	}
	if _, err := New().Render(deck); err == nil || !strings.Contains(err.Error(), "no measured boxes") {
		t.Errorf("잰 박스 없는 렌더 = %v, want 에러", err)
	}
	boxes := map[string]route.Rect{"s0.0": {X: 64, Y: 64, W: 100, H: 100}, "s0.1": {X: 364, Y: 64, W: 100, H: 100}}
	files, err := NewWithBoxes(func(id string) (route.Rect, bool) { r, ok := boxes[id]; return r, ok }).Render(deck)
	if err != nil {
		t.Fatal(err)
	}
	out := string(files[IndexFile])
	// 화살 끝은 표식 길이의 절반(5 × 2px / 2 = 5px)만큼 당겨지고, 표식이 그만큼 앞으로 나가 끝점에 닿는다.
	for _, want := range []string{
		`<svg class="balpyo-connectors" viewBox="0 0 1280 720" aria-hidden="true">`,
		`data-balpyo-connector="s0.2" d="M164,114 L359,114"`,
		`marker-end="url(#balpyo-s0-2-end)"`,
		`stroke: var(--balpyo-color-muted); stroke-width: 2px; stroke-dasharray: 8px 6px`,
		`<marker id="balpyo-s0-2-end" viewBox="0 0 10 10" refX="5" refY="5" markerUnits="strokeWidth" markerWidth="5" markerHeight="5" orient="auto">`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("산출물에 %q가 없다", want)
		}
	}
	// 오버레이는 스테이지 뒤 · 크롬 앞이다 (위치 지정 요소는 문서 순서로 칠한다).
	body := out[strings.Index(out, "<section"):]
	if strings.Index(body, `class="balpyo-stage`) > strings.Index(body, `<svg class="balpyo-connectors"`) {
		t.Error("오버레이가 스테이지 앞에 있다 — 콘텐츠 아래에 칠해진다")
	}
}
