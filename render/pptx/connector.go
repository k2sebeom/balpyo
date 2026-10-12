package pptx

import (
	"fmt"
	"math"
	"strings"

	"github.com/k2sebeom/balpyo/render/geom"
	"github.com/k2sebeom/balpyo/render/route"
	"github.com/k2sebeom/balpyo/render/web"
	"github.com/k2sebeom/balpyo/soml"
)

// 이 파일은 연결선(`<connector>`)의 PPTX 렌더다 — `p:cxnSp` 하나가 연결선 하나다.
//
// 자리는 콘텐츠 도형들 **뒤**, 크롬 **앞**이다 (웹의 오버레이가 스테이지 위 · 크롬 아래인 것과 같다).
// 경로는 render/route가 두 상자의 잰 rect에서 계산한 프리셋 · xfrm이고, 양 끝은 연결점 순서를 아는
// 프리셋에만 **붙는다**(`a:stCxn`/`a:endCxn`) — 붙은 연결선은 PowerPoint에서 상자를 끌면 따라온다.

// connectionSites는 붙일 수 있는 프리셋의 면 → 연결점 번호다 (ECMA-376 presetShapeDefinitions의
// cxnLst 순서). 목록 밖 프리셋을 잇는 끝은 같은 자리에 그려지되 붙지 않는다 — 번호를 확인하지 않은
// 채 붙이면 PowerPoint에서 상자를 끌 때 선이 다른 연결점으로 튄다.
var connectionSites = map[string]map[route.Side]int{
	"rect":         {route.Top: 0, route.Left: 1, route.Bottom: 2, route.Right: 3},
	"rounded-rect": {route.Top: 0, route.Left: 1, route.Bottom: 2, route.Right: 3},
	"diamond":      {route.Top: 0, route.Left: 1, route.Bottom: 2, route.Right: 3},
	"ellipse":      {route.Top: 0, route.Left: 2, route.Bottom: 4, route.Right: 6},
	// 그림(`p:pic`)은 사각형의 연결점 넷을 갖는다.
	soml.ImageElement: {route.Top: 0, route.Left: 1, route.Bottom: 2, route.Right: 3},
}

// ConnectionSite는 요소의 면이 붙는 연결점 번호다 — 붙일 수 없는 요소면 false다 (테스트 하네스가 같은
// 표를 읽는다).
func ConnectionSite(elem *soml.Element, side route.Side) (int, bool) {
	key := elem.Name
	if s := elem.ShapeStyle(); s != nil {
		key = s.Type
	}
	sites, ok := connectionSites[key]
	if !ok {
		return 0, false
	}
	idx, ok := sites[side]
	return idx, ok
}

// writeConnectorShapes는 슬라이드 직속 연결선을 문서 순서로 쓴다.
func writeConnectorShapes(b *strings.Builder, deck *soml.Deck, slide *soml.Slide, shapeID *int, g *geom.Geometry, rels *slideRels) error {
	var ids map[*soml.Element]string
	for _, node := range slide.Content {
		elem, ok := node.(*soml.Element)
		if !ok || elem.Name != soml.ConnectorElement {
			continue
		}
		if elem.Appearance == nil || elem.Appearance.Connector == nil {
			return fmt.Errorf("%s:%s <connector> is unresolved — the deck did not go through core.Resolve", slide.Source, elem.Span)
		}
		if ids == nil {
			ids = web.ElementIDs(slide)
		}
		conn := elem.Appearance.Connector
		p, err := web.RouteConnector(slide, ids, conn, g.RouteBox)
		if err != nil {
			return err
		}
		if err := writeCxnSp(b, deck, ids[elem], *shapeID, conn, p, rels); err != nil {
			return fmt.Errorf("%s:%s <connector>: %w", slide.Source, elem.Span, err)
		}
		if rels.shapeIDs == nil {
			rels.shapeIDs = map[*soml.Element]int{}
		}
		if rels.drawFrom == nil {
			rels.drawFrom = map[*soml.Element]route.Side{}
		}
		rels.shapeIDs[elem] = *shapeID
		rels.drawFrom[elem] = p.DrawFrom()
		*shapeID++
	}
	// 라벨은 모든 선 **뒤**다 — 웹이 라벨을 오버레이 뒤에 두는 것과 같은 칠 순서다 (글자가 선 위에 온다).
	for _, node := range slide.Content {
		elem, ok := node.(*soml.Element)
		if !ok || !soml.HasConnectorLabel(elem) {
			continue
		}
		rect, err := web.ConnectorLabelRect(slide, ids, elem, g.RouteBox)
		if err != nil {
			return err
		}
		conn := elem.Appearance.Connector
		// 몸체는 크롬 푸터와 같은 한 줄 글상자다 (wrap none · 여백 0 · 가운데 앵커): 잰 rect가 곧 글자 크기다.
		body, err := chromeBodyXML(soml.ChromeItem{Kind: soml.ChromeFooter, Text: conn.Label, Style: *conn.LabelStyle,
			Align: soml.AlignCenter}, deck.Theme)
		if err != nil {
			return fmt.Errorf("%s:%s <connector> label: %w", slide.Source, elem.Span, err)
		}
		spec := textBoxSp(*shapeID, "connector-label "+ids[elem], geom.Rect{X: rect.X, Y: rect.Y, W: rect.W, H: rect.H}, body)
		if err := writeSp(b, spec, deck.Theme); err != nil {
			return err
		}
		if rels.labelIDs == nil {
			rels.labelIDs = map[*soml.Element]int{}
		}
		rels.labelIDs[elem] = *shapeID
		*shapeID++
	}
	return nil
}

func writeCxnSp(b *strings.Builder, deck *soml.Deck, id string, shapeID int, conn *soml.ConnectorStyle, p route.Path, rels *slideRels) error {
	w := func(depth int, line string) {
		b.WriteString("\n" + shapeIndent + strings.Repeat("  ", depth) + line)
	}
	w(0, "<p:cxnSp>")
	w(1, "<p:nvCxnSpPr>")
	w(2, fmt.Sprintf(`<p:cNvPr id="%d" name="%s"/>`, shapeID, escapeXML("connector "+id)))
	glue := func(elem *soml.Element, side route.Side, tag string) string {
		sid, ok := rels.shapeIDs[elem]
		if !ok {
			return ""
		}
		idx, ok := ConnectionSite(elem, side)
		if !ok {
			return ""
		}
		return fmt.Sprintf(`<a:%s id="%d" idx="%d"/>`, tag, sid, idx)
	}
	st, end := glue(conn.From, p.StartSide, "stCxn"), glue(conn.To, p.EndSide, "endCxn")
	if st == "" && end == "" {
		w(2, "<p:cNvCxnSpPr/>")
	} else {
		w(2, "<p:cNvCxnSpPr>")
		if st != "" {
			w(3, st)
		}
		if end != "" {
			w(3, end)
		}
		w(2, "</p:cNvCxnSpPr>")
	}
	w(2, "<p:nvPr/>")
	w(1, "</p:nvCxnSpPr>")
	w(1, "<p:spPr>")
	var xfrm []string
	if p.Rot != 0 {
		xfrm = append(xfrm, fmt.Sprintf(`rot="%d"`, Rotation(float64(p.Rot))))
	}
	if p.FlipH {
		xfrm = append(xfrm, `flipH="1"`)
	}
	if p.FlipV {
		xfrm = append(xfrm, `flipV="1"`)
	}
	open := "<a:xfrm>"
	if len(xfrm) > 0 {
		open = "<a:xfrm " + strings.Join(xfrm, " ") + ">"
	}
	w(2, open)
	w(3, fmt.Sprintf(`<a:off x="%d" y="%d"/>`, EMU(p.Off.X), EMU(p.Off.Y)))
	w(3, fmt.Sprintf(`<a:ext cx="%d" cy="%d"/>`, EMU(p.W), EMU(p.H)))
	w(2, "</a:xfrm>")
	w(2, fmt.Sprintf(`<a:prstGeom prst="%s">`, p.Preset))
	if adj := connectorAdjusts(p.Adj); adj != nil {
		w(3, "<a:avLst>")
		for i, v := range adj {
			w(4, fmt.Sprintf(`<a:gd name="adj%d" fmla="val %d"/>`, i+1, v))
		}
		w(3, "</a:avLst>")
	} else {
		w(3, "<a:avLst/>")
	}
	w(2, "</a:prstGeom>")
	if err := writeConnectorLine(w, 2, conn, deck.Theme); err != nil {
		return err
	}
	w(1, "</p:spPr>")
	w(0, "</p:cxnSp>")
	return nil
}

// writeConnectorLine은 연결선의 `a:ln`이다 — 도형 테두리와 같은 굵기 · 색 · 대시에 양 끝 표식이 붙는다
// (CT_LineProperties 순서: 채움 → prstDash → headEnd → tailEnd). 표식은 굵기의 5배(`lg`)다 — 웹의
// `<marker>` 크기와 같은 규칙이다.
func writeConnectorLine(w func(depth int, line string), depth int, conn *soml.ConnectorStyle, theme soml.Theme) error {
	line := conn.Line
	if line.Width <= 0 {
		return fmt.Errorf("connector line width is unresolved (%v)", line.Width)
	}
	dash, err := prstDashValue(line.Style)
	if err != nil {
		return err
	}
	w(depth, fmt.Sprintf(`<a:ln w="%d">`, EMU(line.Width)))
	w(depth+1, "<a:solidFill>")
	if err := writeColor(w, depth+2, line.Color, nil, theme); err != nil {
		return err
	}
	w(depth+1, "</a:solidFill>")
	w(depth+1, fmt.Sprintf(`<a:prstDash val="%s"/>`, dash))
	w(depth+1, lineEndXML("headEnd", conn.Start))
	w(depth+1, lineEndXML("tailEnd", conn.End))
	w(depth, "</a:ln>")
	return nil
}

// lineEndXML은 선 끝 표식이다 — 화살은 `triangle`, 점은 `oval`, 없음도 명시한다 (테마의 선 기본값에
// 기대지 않는다).
func lineEndXML(tag string, end soml.ConnectorEnd) string {
	switch end {
	case soml.EndArrow:
		return fmt.Sprintf(`<a:%s type="triangle" w="lg" len="lg"/>`, tag)
	case soml.EndDot:
		return fmt.Sprintf(`<a:%s type="oval" w="lg" len="lg"/>`, tag)
	}
	return fmt.Sprintf(`<a:%s type="none"/>`, tag)
}

// connectorAdjusts는 연결선 프리셋의 `avLst` 값(비율 × 100000)이다. 모든 값이 프리셋 기본(가운데,
// 50000)이면 nil이다 — 기본값을 적으면 바이트만 늘어난다.
func connectorAdjusts(adj []float64) []int64 {
	var out []int64
	custom := false
	for _, a := range adj {
		v := int64(math.Round(a * 100000))
		custom = custom || v != 50000
		out = append(out, v)
	}
	if !custom {
		return nil
	}
	return out
}
