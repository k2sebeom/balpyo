package web

import (
	"fmt"
	"html"
	"math"
	"strings"

	"github.com/k2sebeom/balpyo/render/route"
	"github.com/k2sebeom/balpyo/soml"
)

// 이 파일은 연결선(`<connector>`)의 웹 렌더다.
//
// 연결선은 **측정이 끝난 뒤** 그려진다: 끝점이 두 상자의 잰 rect에서 나오는데, 이 렌더러의 HTML이 곧
// 측정 대상이기 때문이다. 그래서 렌더러는 세 상태다 — 측정용(연결선을 내지 않는다), 잰 박스를 받은
// 렌더(연결선을 굽는다), 둘 다 아닌 렌더(연결선이 있으면 에러 — 조용히 선을 빼지 않는다).
//
// 연결선은 흐름에 끼지 않으므로 측정한 HTML과 연결선을 구운 HTML의 배치가 같다. 슬라이드마다 SVG
// 오버레이 하나가 스테이지 위 · 크롬 아래에 놓이고(PPTX가 콘텐츠 도형 뒤 · 크롬 앞에 `p:cxnSp`를 두는
// 것과 같은 자리), 경로는 render/route가 PPTX 프리셋에서 계산한 것을 그대로 쓴다.

// BoxLookup은 요소 식별자(ElementID)의 잰 rect다 — 연결선이 잇는 두 상자의 끝점이 여기서 나온다.
// render/geom을 임포트하지 않으려고 함수로 받는다 (geom이 이 패키지를 부른다).
type BoxLookup func(id string) (route.Rect, bool)

// ConnectorClass는 연결선 오버레이의 클래스다.
const ConnectorClass = "balpyo-connectors"

// ConnectorAttr는 연결선 경로의 요소 식별자 속성이다 (테스트 하네스가 읽는다).
const ConnectorAttr = "data-balpyo-connector"

// connectorMode는 렌더러가 연결선을 어떻게 다루는지다.
type connectorMode int

const (
	// connectorsRequireBoxes는 기본값이다 — 잰 박스가 없으므로 연결선이 있는 덱은 에러다.
	connectorsRequireBoxes connectorMode = iota
	// connectorsOmitted는 측정용 렌더다 — 연결선을 내지 않는다 (흐름에 끼지 않으므로 배치가 같다).
	connectorsOmitted
	// connectorsDrawn은 잰 박스를 받은 렌더다.
	connectorsDrawn
)

// elementIDs는 슬라이드의 요소 → 식별자 표다 (ElementID와 같은 경로 규칙 — 같은 함수로 조립한다).
func elementIDs(slide *soml.Slide) map[*soml.Element]string {
	ids := map[*soml.Element]string{}
	var walk func(node soml.Node, path []int)
	walk = func(node soml.Node, path []int) {
		elem, ok := node.(*soml.Element)
		if !ok {
			return
		}
		ids[elem] = ElementID(slide.Index, path)
		for i, child := range elem.Children {
			walk(child, ChildPath(path, i))
		}
	}
	for i, node := range slide.Content {
		walk(node, []int{i})
	}
	return ids
}

// slideConnectors는 슬라이드 직속의 연결선이다 (문서 순서).
func slideConnectors(slide *soml.Slide) []*soml.Element {
	var out []*soml.Element
	for _, node := range slide.Content {
		if elem, ok := node.(*soml.Element); ok && elem.Name == soml.ConnectorElement {
			out = append(out, elem)
		}
	}
	return out
}

// RouteConnector는 확정된 연결선의 경로다 — 두 렌더러와 테스트 하네스가 같은 계산을 부른다.
func RouteConnector(slide *soml.Slide, ids map[*soml.Element]string, conn *soml.ConnectorStyle, boxes BoxLookup) (route.Path, error) {
	from, ok := boxes(ids[conn.From])
	if !ok {
		return route.Path{}, fmt.Errorf("%s: connector start %s has no measured box", slide.Source, ids[conn.From])
	}
	to, ok := boxes(ids[conn.To])
	if !ok {
		return route.Path{}, fmt.Errorf("%s: connector end %s has no measured box", slide.Source, ids[conn.To])
	}
	return route.Route(from, to, route.Side(conn.FromSide), route.Side(conn.ToSide), route.Kind(conn.Route))
}

// ElementIDs는 elementIDs의 공개판이다 (PPTX 렌더러와 테스트 하네스가 연결선의 끝을 식별자로 찾는다).
func ElementIDs(slide *soml.Slide) map[*soml.Element]string { return elementIDs(slide) }

// writeConnectors는 슬라이드의 연결선 오버레이를 쓴다 (연결선이 없으면 아무것도 쓰지 않는다).
func writeConnectors(b *strings.Builder, deck *soml.Deck, slide *soml.Slide, mode connectorMode, boxes BoxLookup) error {
	conns := slideConnectors(slide)
	if len(conns) == 0 || mode == connectorsOmitted {
		return nil
	}
	if mode != connectorsDrawn {
		return fmt.Errorf("%s: the slide has connectors but the web renderer has no measured boxes — connectors are drawn after the geometry pass", slide.Source)
	}
	ids := elementIDs(slide)
	fmt.Fprintf(b, "<svg class=%q viewBox=\"0 0 %d %d\" aria-hidden=\"true\">\n", ConnectorClass, deck.Stage.Width, deck.Stage.Height)
	for _, elem := range conns {
		if elem.Appearance == nil || elem.Appearance.Connector == nil {
			return fmt.Errorf("%s:%s <connector> is unresolved — the deck did not go through core.Resolve", slide.Source, elem.Span)
		}
		conn := elem.Appearance.Connector
		p, err := RouteConnector(slide, ids, conn, boxes)
		if err != nil {
			return err
		}
		id := ids[elem]
		if err := writeConnectorPath(b, deck, id, conn, p); err != nil {
			return fmt.Errorf("%s:%s <connector>: %w", slide.Source, elem.Span, err)
		}
	}
	b.WriteString("</svg>\n")
	return nil
}

// 끝 표식의 크기는 선 굵기의 배수다 — PPTX `headEnd`/`tailEnd`의 `w="lg" len="lg"`(굵기의 5배)와
// 같은 크기다.
const markerScale = 5

// writeConnectorPath는 연결선 하나를 쓴다 — 표식이 있으면 그 연결선만의 `<marker>`가 앞에 온다
// (색이 선마다 다르다).
func writeConnectorPath(b *strings.Builder, deck *soml.Deck, id string, conn *soml.ConnectorStyle, p route.Path) error {
	stroke, err := colorValue(conn.Line.Color, deck.Theme)
	if err != nil {
		return err
	}
	dash, err := dashArray(conn.Line.Style, conn.Line.Width)
	if err != nil {
		return err
	}
	// 화살촉 아래로 선의 끝이 삐져나오지 않게 화살 쪽 끝을 표식 길이의 절반만큼 줄인다 — PPTX도 화살촉
	// 아래의 선을 줄여 그린다. 표식의 끝은 줄인 만큼 앞으로 나가 원래 끝점에 닿는다 (refX).
	shorten := markerScale * conn.Line.Width / 2
	points := append([]route.Point(nil), p.Points...)
	if conn.End == soml.EndArrow {
		points[len(points)-1] = pullBack(points[len(points)-1], points[len(points)-2], shorten)
	}
	if conn.Start == soml.EndArrow {
		points[0] = pullBack(points[0], points[1], shorten)
	}
	p.Points = points

	attrs := []string{fmt.Sprintf("%s=%q", ConnectorAttr, id), fmt.Sprintf("d=%q", p.SVG())}
	for _, end := range []struct {
		which  string
		marker soml.ConnectorEnd
	}{{"start", conn.Start}, {"end", conn.End}} {
		if end.marker == soml.EndNone || end.marker == "" {
			continue
		}
		markerID := "balpyo-" + strings.ReplaceAll(id, ".", "-") + "-" + end.which
		writeMarker(b, markerID, end.marker, end.which, stroke)
		attrs = append(attrs, fmt.Sprintf("marker-%s=\"url(#%s)\"", end.which, markerID))
	}
	decls := []string{"fill: none", "stroke: " + stroke, "stroke-width: " + formatNumber(conn.Line.Width) + "px"}
	if dash != "" {
		decls = append(decls, "stroke-dasharray: "+dash)
	}
	fmt.Fprintf(b, "<path %s%s/>\n", strings.Join(attrs, " "), styleAttr(decls))
	return nil
}

// writeMarker는 끝 표식 하나를 쓴다 — 화살은 끝점을 향한 삼각형, 점은 끝점에 중심을 둔 원이다.
// markerUnits가 선 굵기라 굵은 선의 표식이 함께 커진다 (PPTX의 크기 규칙과 같다).
func writeMarker(b *strings.Builder, id string, end soml.ConnectorEnd, which, color string) {
	orient := "auto"
	if which == "start" {
		orient = "auto-start-reverse"
	}
	fill := html.EscapeString("fill: " + color)
	switch end {
	case soml.EndArrow:
		fmt.Fprintf(b, "<marker id=%q viewBox=\"0 0 10 10\" refX=\"5\" refY=\"5\" markerUnits=\"strokeWidth\" markerWidth=\"%d\" markerHeight=\"%d\" orient=%q><path d=\"M0,0 L10,5 L0,10 Z\" style=\"%s\"/></marker>\n",
			id, markerScale, markerScale, orient, fill)
	case soml.EndDot:
		fmt.Fprintf(b, "<marker id=%q viewBox=\"0 0 10 10\" refX=\"5\" refY=\"5\" markerUnits=\"strokeWidth\" markerWidth=\"%d\" markerHeight=\"%d\" orient=%q><circle cx=\"5\" cy=\"5\" r=\"5\" style=\"%s\"/></marker>\n",
			id, markerScale, markerScale, orient, fill)
	}
}

// pullBack은 끝점 end를 이웃 점 toward 쪽으로 d만큼 당긴다. 조각이 d보다 짧으면 그대로 둔다.
func pullBack(end, toward route.Point, d float64) route.Point {
	dx, dy := toward.X-end.X, toward.Y-end.Y
	length := math.Hypot(dx, dy)
	if length <= d || length == 0 {
		return end
	}
	return route.Point{X: end.X + dx/length*d, Y: end.Y + dy/length*d}
}

// connectorCSS는 오버레이의 기본 CSS다 — 슬라이드 전체를 덮고, 클릭을 가로채지 않으며, 스테이지보다
// 위에 칠한다 (위치 지정 요소는 문서 순서로 칠하므로 스테이지 뒤에 둔 오버레이가 위에 온다).
func connectorCSS() string {
	return `
  .balpyo-connectors {
    position: absolute;
    left: 0;
    top: 0;
    width: 100%;
    height: 100%;
    overflow: visible;
    pointer-events: none;
  }
`
}
