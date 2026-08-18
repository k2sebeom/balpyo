package pptx

import (
	"strings"
	"testing"

	"github.com/k2sebeom/balpyo/render/geom"
	"github.com/k2sebeom/balpyo/soml"
)

// contentDeck은 좌표가 필요한 요소가 있는 덱이다.
func contentDeck() *soml.Deck {
	deck := testDeck()
	deck.Slides[0].Content = []soml.Node{
		&soml.Element{Name: soml.TextElement, Children: []soml.Node{&soml.Text{Value: "제목"}}},
	}
	return deck
}

// 좌표를 요구하는 요소가 있는데 지오메트리가 없으면 멈춘다. 그냥 조립하면 도형이 빠졌거나
// 전부 원점에 겹친 파일이 나오고, 그 상태는 PowerPoint에서 열어 보기 전까지 보이지 않는다.
func TestRenderRequiresGeometryForContent(t *testing.T) {
	_, err := New().Render(contentDeck())
	if err == nil {
		t.Fatal("지오메트리 없이 렌더되었다")
	}
	if !strings.Contains(err.Error(), "지오메트리") {
		t.Errorf("에러가 지오메트리를 가리키지 않는다: %v", err)
	}
}

// 콘텐츠가 없는 덱은 잰 좌표가 필요하지 않다 — 빈 슬라이드 픽스처가 브라우저 없이 돈다.
func TestRenderWithoutGeometryForEmptyDeck(t *testing.T) {
	if _, err := New().Render(testDeck()); err != nil {
		t.Fatalf("빈 덱은 지오메트리 없이 렌더되어야 한다: %v", err)
	}
}

// 다른 덱에서 잰 지오메트리를 받으면 좌표가 전부 엉뚱한 요소로 간다 — 소비 시점에도 대조한다.
func TestRenderRejectsForeignGeometry(t *testing.T) {
	other := contentDeck()
	other.Slides[0].Content = append(other.Slides[0].Content,
		&soml.Element{Name: soml.TextElement, Children: []soml.Node{&soml.Text{Value: "본문"}}})

	g := geom.Synthetic(other, geom.Rect{X: 64, Y: 64, W: 1152, H: 36})
	if _, err := NewWithGeometry(g).Render(contentDeck()); err == nil {
		t.Error("덱과 짝이 맞지 않는 지오메트리는 에러여야 한다")
	}
}
