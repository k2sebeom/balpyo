package testkit

import (
	"encoding/xml"
	"fmt"
	"math"
	"os"
	"path"
	"strconv"
	"strings"
	"testing"

	"github.com/k2sebeom/balpyo/internal/pipeline"
	"github.com/k2sebeom/balpyo/render/geom"
	"github.com/k2sebeom/balpyo/render/route"
	"github.com/k2sebeom/balpyo/render/web"
	"github.com/k2sebeom/balpyo/soml"
)

// 동일성 판정: 두 타깃에서 각각 뽑은 **컴포넌트 트리**가 같은가.
//
// 웹 쪽 트리는 덱 모델 + 지오메트리 side table에서, PPTX 쪽 트리는 산출된 슬라이드 XML의
// `spTree`를 다시 읽어서 만든다. 두 트리는 같은 줄 형식으로 직렬화되고 줄 단위로 비교된다 —
// 같은 실행의 같은 측정값을 두 번 읽는 것이라 설치된 폰트와 무관하다.
//
// 노드 한 줄: 종류 · 요소 ID · rect · 플레이스홀더 · 대체 텍스트 · 채움/선. 그 아래 문단마다
// 한 줄: 정렬 · 마커 · 런(글자 + 글꼴 참조 · 크기 · 굵기 · 기울임 · 밑줄 · 색 토큰 · 링크).
// 줄이 어디서 접히는지는 계약 밖이라 트리에 없다.

// LayoutSlide는 슬라이드 하나의 컴포넌트 트리다.
type LayoutSlide struct {
	ID string
	// Background는 테마가 정한 슬라이드 배경의 요약이다 — 채움(`fill=accent1` · `fill=gradient(…)`)이거나
	// `image`다. 비면 배경을 정하지 않은 슬라이드다 (마스터의 bg1). 그림 바이트가 입력과 같은지는 패키지
	// 판정(미디어 파트 = 에셋)이 본다.
	Background string
	// Transition은 이 장에 들어올 때의 전환이다 (`fade/fast` — 효과/속도 단계). 비면 전환이 없다.
	Transition string
	// Inverse는 이 장의 색 매핑이 뒤집혔는지다 (`colors inverse`).
	Inverse bool
	Nodes   []LayoutNode
}

// LayoutNode는 PPTX 객체 하나(= 웹 요소 하나)다. 문서 순서 = 칠 순서다.
type LayoutNode struct {
	// Kind는 객체 종류다: textbox · shape:<prst> · pic · table.
	Kind string
	// Name은 `cNvPr@name`이다 — `<요소 이름> <요소 ID>` (크롬은 `chrome <종류>`).
	Name string
	// X·Y·W·H는 EMU다 — 비교는 정수가 같은지다 (계약의 "EMU 반올림 안에서").
	X, Y, W, H int64
	// Ph는 플레이스홀더다 (`type/idx`) — 슬롯이 없으면 빈 문자열.
	Ph string
	// Alt는 대체 텍스트다 (`cNvPr@descr`).
	Alt string
	// Fill·Line은 도형 채움·선의 요약이다 (텍스트 박스·그림·표는 빈 문자열).
	Fill, Line string
	// Extra는 종류 고유의 꼬리다 — 연결선의 프레임(회전 · 뒤집기 · 꺾는 자리), 붙은 양 끝, 대시,
	// 끝 표식 (다른 종류는 빈 문자열).
	Extra string
	Paras []LayoutPara
}

// LayoutPara는 문단 하나다 — 런이 없는 문단은 트리에 넣지 않는다 (빈 도형의 `<a:p/>`).
type LayoutPara struct {
	// Prefix는 문단의 자리다 — 표 셀은 `r<행>`, 그 밖은 빈 문자열.
	Prefix string
	Align  string
	// Bullet은 리스트 마커 종류다 (ul · ol) — 리스트 밖은 빈 문자열.
	Bullet string
	Runs   []string
}

// FormatLayout은 트리를 줄 형식으로 직렬화한다. withRect가 false면 rect를 뺀다 —
// `golden/layout.txt`의 형식이다 (rect는 폰트에 따라 움직이므로 골든에 넣지 않는다).
func FormatLayout(slides []LayoutSlide, withRect bool) string {
	var b strings.Builder
	for i, s := range slides {
		fmt.Fprintf(&b, "# slide %d %s\n", i+1, s.ID)
		if s.Background != "" {
			b.WriteString("bg " + s.Background + "\n")
		}
		if s.Transition != "" {
			b.WriteString("transition " + s.Transition + "\n")
		}
		if s.Inverse {
			b.WriteString("colors inverse\n")
		}
		for _, n := range s.Nodes {
			b.WriteString(n.Kind + " " + strconv.Quote(n.Name))
			if withRect {
				// EMU 정수 그대로다 — px로 되돌려 소수점에서 자르면 1 EMU 어긋남이 같은 줄이 된다.
				fmt.Fprintf(&b, " @%d,%d %d×%d", n.X, n.Y, n.W, n.H)
			}
			if n.Ph != "" {
				b.WriteString(" ph=" + n.Ph)
			}
			if n.Alt != "" {
				b.WriteString(" alt=" + strconv.Quote(n.Alt))
			}
			if n.Fill != "" {
				b.WriteString(" fill=" + n.Fill)
			}
			if n.Line != "" {
				b.WriteString(" line=" + n.Line)
			}
			if n.Extra != "" {
				b.WriteString(" " + n.Extra)
			}
			b.WriteString("\n")
			for _, p := range n.Paras {
				b.WriteString("  ¶")
				if p.Prefix != "" {
					b.WriteString(" " + p.Prefix)
				}
				b.WriteString(" " + p.Align)
				if p.Bullet != "" {
					b.WriteString(" " + p.Bullet)
				}
				b.WriteString(" | " + strings.Join(p.Runs, " · ") + "\n")
			}
		}
	}
	return b.String()
}

// parityBoldWeight는 렌더러가 rPr@b를 켜는 최소 font-weight다 (render/pptx의 같은 이름 상수).
// 웹 쪽 요약이 수치를 굵기 여부로 접을 때 쓴다.
const parityBoldWeight = 600

// layoutGolden은 픽스처 골든의 구조 스냅샷이다 (rect 없는 트리).
const layoutGolden = "layout.txt"

// Layouts는 빌드 결과에서 두 트리를 뽑는다 — 웹 쪽(덱 + 지오메트리)과 PPTX 쪽(spTree).
func (f Fixture) Layouts(t testing.TB, result *pipeline.Result) (webTree, pptxTree []LayoutSlide) {
	t.Helper()
	webTree, err := WebLayout(result.Deck, result.Geometry)
	if err != nil {
		t.Fatalf("%s: web tree: %v", f.Name, err)
	}
	data, err := os.ReadFile(f.Output(t, result, ".pptx"))
	if err != nil {
		t.Fatalf("%s: %v", f.Name, err)
	}
	pptxTree, err = PPTXLayout(PPTXParts(t, data), len(result.Deck.Slides), result.Deck.Theme)
	if err != nil {
		t.Fatalf("%s: PPTX tree: %v", f.Name, err)
	}
	return webTree, pptxTree
}

// AssertParity는 두 트리가 같은지 단정한다 — rect까지 (EMU 정수로) 비교한다.
func (f Fixture) AssertParity(t testing.TB, result *pipeline.Result) {
	t.Helper()
	webTree, pptxTree := f.Layouts(t, result)
	want, got := FormatLayout(webTree, true), FormatLayout(pptxTree, true)
	if want != got {
		t.Errorf("%s: web tree and PPTX tree differ (want = web, got = PPTX)\n%s",
			f.Name, diffReport([]byte(want), []byte(got)))
	}
}

// ── 웹 쪽: 덱 모델 + 지오메트리 ─────────────────────────────────────────────

// WebLayout은 덱 모델과 지오메트리에서 컴포넌트 트리를 만든다.
//
// 요소가 무엇이 되는지는 렌더 매핑의 표 그대로다 (pptx.RenderedKinds의 주석): 블록 · 리스트
// 컨테이너 · 도형 · 그림 · 표는 객체 하나, 레이아웃 컨테이너 · spacer · li · 인라인 · 행/셀은
// 객체가 아니다. 흡수 모드 도형의 자식은 그 도형의 문단이 된다.
func WebLayout(deck *soml.Deck, g *geom.Geometry) ([]LayoutSlide, error) {
	var slides []LayoutSlide
	for _, slide := range deck.Slides {
		s := LayoutSlide{ID: slide.ID, Inverse: slide.Invert}
		if bg := slide.Background; bg != nil && bg.Overlay != nil {
			// 배경 덮개는 맨 앞 객체다 — PPTX는 잠긴 전면 사각형이다 (웹은 배경의 윗겹).
			n := LayoutNode{Kind: "shape:rect", Name: "background overlay", Fill: webFill(bg.Overlay, deck.Theme), Line: "none"}
			n.X, n.Y, n.W, n.H = rectEMU(geom.Rect{W: float64(deck.Stage.Width), H: float64(deck.Stage.Height)})
			s.Nodes = append(s.Nodes, n)
		}
		if slide.Transition != "" {
			// 속도 단계는 렌더러와 따로 접는다 — 600ms까지 fast, 875ms까지 med, 그보다 길면 slow.
			spd := "slow"
			switch {
			case slide.TransitionDuration <= 600:
				spd = "fast"
			case slide.TransitionDuration <= 875:
				spd = "med"
			}
			s.Transition = string(slide.Transition) + "/" + spd
		}
		if bg := slide.Background; bg != nil {
			if bg.Image != "" {
				s.Background = "image"
			} else {
				s.Background = "fill=" + webFill(bg.Fill, deck.Theme)
			}
		}
		// enter는 가장 가까운 등장 조상이다 — 객체는 그 등장과 함께 나타난다 (PPTX는 객체마다 효과 하나).
		var walk func(node soml.Node, p []int, enter *soml.Entrance) error
		walk = func(node soml.Node, p []int, enter *soml.Entrance) error {
			elem, ok := node.(*soml.Element)
			if !ok {
				return nil
			}
			if e := elem.EnterOf(); e != nil {
				enter = e
			}
			id := web.ElementID(slide.Index, p)
			kind := soml.KindOf(elem.Name)
			if elem.Name == soml.TextElement {
				kind = soml.KindBlock
			}
			switch kind {
			case soml.KindLayoutContainer:
				for i, child := range elem.Children {
					if err := walk(child, web.ChildPath(p, i), enter); err != nil {
						return err
					}
				}
				return nil
			case soml.KindSpacer, soml.KindConnector:
				// 연결선은 콘텐츠 도형 뒤에 한꺼번에 온다 (webConnectors).
				return nil
			}
			box, ok := g.Box(id)
			if !ok {
				return fmt.Errorf("%s: %s <%s> has no measured box", slide.Source, id, elem.Name)
			}
			n := LayoutNode{Name: elem.Name + " " + id, Ph: webPlaceholder(slide, elem)}
			rect := box.Rect
			switch kind {
			case soml.KindBlock:
				style := elem.TextStyle()
				if style == nil {
					return fmt.Errorf("%s: %s <%s> appearance is not resolved", slide.Source, id, elem.Name)
				}
				n.Kind = "textbox"
				n.Paras = webParas(n.Paras, elem, *style, deck.Theme, "", "")
			case soml.KindListContainer:
				n.Kind = "textbox"
				n.Paras = webListParas(n.Paras, elem, deck.Theme)
			case soml.KindShape:
				style := elem.ShapeStyle()
				if style == nil {
					return fmt.Errorf("%s: %s <%s> shape appearance is not resolved", slide.Source, id, elem.Name)
				}
				preset, _ := web.ShapePreset(style.Type)
				n.Kind = "shape:" + preset
				n.Alt = style.Alt
				n.Fill, n.Line = webFill(style.Fill, deck.Theme), webLine(style.Border, deck.Theme)
				if !soml.ShapeHoldsLayout(elem) {
					for _, child := range elem.Children {
						text, ok := child.(*soml.Element)
						if !ok {
							continue
						}
						if soml.KindOf(text.Name) == soml.KindListContainer {
							n.Paras = webListParas(n.Paras, text, deck.Theme)
						} else if ts := text.TextStyle(); ts != nil {
							n.Paras = webParas(n.Paras, text, *ts, deck.Theme, "", "")
						}
					}
				}
			case soml.KindImage:
				n.Kind = "pic"
				if style := elem.ImageStyle(); style != nil {
					n.Alt = style.Alt
				}
			case soml.KindTable:
				n.Kind = "table"
				if style := elem.TableStyle(); style != nil {
					n.Alt = style.Alt
				}
				n.Paras = webTableParas(elem, deck.Theme)
			default:
				return fmt.Errorf("%s: %s <%s> has no mapping to an object", slide.Source, id, elem.Name)
			}
			n.X, n.Y, n.W, n.H = rectEMU(rect)
			if enter != nil {
				n.Extra = fmt.Sprintf("enter=%s@%d+%d", enter.Effect, enter.Delay, enter.Duration)
			}
			s.Nodes = append(s.Nodes, n)
			// 배경 모드 도형의 자식은 형제 객체다.
			if kind == soml.KindShape && soml.ShapeHoldsLayout(elem) {
				for i, child := range elem.Children {
					if err := walk(child, web.ChildPath(p, i), enter); err != nil {
						return err
					}
				}
			}
			return nil
		}
		for i, node := range slide.Content {
			if err := walk(node, []int{i}, nil); err != nil {
				return nil, err
			}
		}
		conns, err := webConnectors(slide, g, deck.Theme)
		if err != nil {
			return nil, err
		}
		s.Nodes = append(s.Nodes, conns...)
		// 크롬은 콘텐츠 다음이다 (spTree 순서 = z-순서 — 웹에서도 스테이지 뒤 형제다).
		for _, item := range slide.Chrome {
			rect, ok := g.ChromeRect(slide.Index, item.Kind)
			if !ok {
				return nil, fmt.Errorf("%s: chrome %s has no measured box", slide.Source, item.Kind)
			}
			n := LayoutNode{Kind: "textbox", Name: "chrome " + string(item.Kind)}
			if item.Kind == soml.ChromeLogo {
				// 그림 크롬은 그림이다 — 대체 텍스트가 이름표다.
				n = LayoutNode{Kind: "pic", Name: "chrome " + string(item.Kind), Alt: item.Alt}
			}
			n.X, n.Y, n.W, n.H = rectEMU(rect)
			if item.Text != "" {
				style := webRunStyle(item.Style, deck.Theme, "")
				var runs []string
				if item.Prefix != "" {
					runs = append(runs, formatRun(item.Prefix, style))
				}
				field := style
				field.field = item.Kind == soml.ChromePageNumber
				runs = append(runs, formatRun(item.Text, field))
				if item.Suffix != "" {
					runs = append(runs, formatRun(item.Suffix, style))
				}
				n.Paras = []LayoutPara{{Align: string(item.Align), Runs: runs}}
			}
			s.Nodes = append(s.Nodes, n)
		}
		slides = append(slides, s)
	}
	return slides, nil
}

// parityConnectionSites는 붙일 수 있는 상자의 면 → 연결점 번호다 — 렌더러의 표를 부르지 않고 따로
// 적는다 (ECMA-376 presetShapeDefinitions의 cxnLst 순서). 표 밖의 상자는 붙지 않는다.
var parityConnectionSites = map[string]map[route.Side]int{
	"rect":            {route.Top: 0, route.Left: 1, route.Bottom: 2, route.Right: 3},
	"rounded-rect":    {route.Top: 0, route.Left: 1, route.Bottom: 2, route.Right: 3},
	"diamond":         {route.Top: 0, route.Left: 1, route.Bottom: 2, route.Right: 3},
	"ellipse":         {route.Top: 0, route.Left: 2, route.Bottom: 4, route.Right: 6},
	soml.ImageElement: {route.Top: 0, route.Left: 1, route.Bottom: 2, route.Right: 3},
}

// webConnectors는 슬라이드 직속 연결선의 기대 노드다 — 덱 + 지오메트리에서 경로 계산(render/route)을
// 거쳐 프레임을 만든다. 렌더러의 조립(xfrm · 붙은 끝 · 선)을 판정하는 쪽이다: 경로 계산 자체는 두
// 렌더러가 공유하는 계약이고, route의 단위 테스트가 프리셋 평가와 끝점의 일치를 본다.
func webConnectors(slide *soml.Slide, g *geom.Geometry, theme soml.Theme) ([]LayoutNode, error) {
	var nodes, labels []LayoutNode
	ids := web.ElementIDs(slide)
	for _, node := range slide.Content {
		elem, ok := node.(*soml.Element)
		if !ok || elem.Name != soml.ConnectorElement {
			continue
		}
		if elem.Appearance == nil || elem.Appearance.Connector == nil {
			return nil, fmt.Errorf("%s: %s <connector> is not resolved", slide.Source, ids[elem])
		}
		conn := elem.Appearance.Connector
		box := func(e *soml.Element) (route.Rect, error) {
			b, ok := g.Box(ids[e])
			if !ok {
				return route.Rect{}, fmt.Errorf("%s: connector end %s has no measured box", slide.Source, ids[e])
			}
			return route.Rect{X: b.Rect.X, Y: b.Rect.Y, W: b.Rect.W, H: b.Rect.H}, nil
		}
		from, err := box(conn.From)
		if err != nil {
			return nil, err
		}
		to, err := box(conn.To)
		if err != nil {
			return nil, err
		}
		p, err := route.Route(from, to, route.Side(conn.FromSide), route.Side(conn.ToSide), route.Kind(conn.Route))
		if err != nil {
			return nil, err
		}
		n := LayoutNode{Kind: "connector:" + p.Preset, Name: "connector " + ids[elem], Line: webColor(conn.Line.Color)}
		n.X, n.Y = int64(math.Round(p.Off.X*9525)), int64(math.Round(p.Off.Y*9525))
		n.W, n.H = int64(math.Round(p.W*9525)), int64(math.Round(p.H*9525))
		extra := []string{fmt.Sprintf("rot=%d", p.Rot)}
		if p.FlipH {
			extra = append(extra, "flipH")
		}
		if p.FlipV {
			extra = append(extra, "flipV")
		}
		extra = append(extra, fmt.Sprintf("adj=%d", int64(math.Round(p.Adj*100000))))
		glue := func(which string, e *soml.Element, side route.Side) string {
			key := e.Name
			if s := e.ShapeStyle(); s != nil {
				key = s.Type
			}
			idx, ok := parityConnectionSites[key][side]
			if !ok {
				return which + "=-"
			}
			return fmt.Sprintf("%s=%s:%d", which, strconv.Quote(e.Name+" "+ids[e]), idx)
		}
		dash := map[soml.BorderStyle]string{soml.BorderSolid: "solid", soml.BorderDashed: "dash", soml.BorderDotted: "dot"}[conn.Line.Style]
		end := map[soml.ConnectorEnd]string{soml.EndNone: "none", soml.EndArrow: "triangle", soml.EndDot: "oval"}
		extra = append(extra, glue("from", conn.From, p.StartSide), glue("to", conn.To, p.EndSide),
			"dash="+dash, "ends="+end[conn.Start]+"/"+end[conn.End])
		enter := elem.EnterOf()
		if enter != nil {
			// 선 그리기는 PPTX에서 닦아내기다 — 트리에는 PPTX가 되읽는 이름으로 적는다.
			effect := string(enter.Effect)
			if enter.Effect == soml.EnterDraw {
				effect = "wipe"
			}
			extra = append(extra, fmt.Sprintf("enter=%s@%d+%d", effect, enter.Delay, enter.Duration))
		}
		n.Extra = strings.Join(extra, " ")
		nodes = append(nodes, n)
		if conn.Label != "" {
			// 라벨은 모든 선 뒤의 한 줄 글상자다 — 잰 크기(연결선의 식별자)를 경로 옆에 놓는다.
			size, ok := g.Box(ids[elem])
			if !ok {
				return nil, fmt.Errorf("%s: connector %s label has no measured box", slide.Source, ids[elem])
			}
			rect := p.LabelRect(size.Rect.W, size.Rect.H)
			label := LayoutNode{Kind: "textbox", Name: "connector-label " + ids[elem]}
			label.X, label.Y, label.W, label.H = rectEMU(geom.Rect{X: rect.X, Y: rect.Y, W: rect.W, H: rect.H})
			label.Paras = []LayoutPara{{Align: string(soml.AlignCenter),
				Runs: []string{formatRun(conn.Label, webRunStyle(*conn.LabelStyle, theme, ""))}}}
			if enter != nil {
				label.Extra = fmt.Sprintf("enter=fade@%d+%d", enter.Delay, enter.Duration)
			}
			labels = append(labels, label)
		}
	}
	return append(nodes, labels...), nil
}

// rectEMU는 잰 rect의 EMU다 — 렌더러의 pptx.Offset/Extent를 부르지 않고 **따로** 계산한다
// (1px = 9525 EMU, 반올림). 같은 함수를 두 쪽이 부르면 그 함수의 결함은 두 트리에 똑같이
// 실려 판정을 지나간다.
func rectEMU(r geom.Rect) (x, y, w, h int64) {
	emu := func(px float64) int64 { return int64(math.Round(px * 9525)) }
	return emu(r.X), emu(r.Y), emu(r.W), emu(r.H)
}

func webPlaceholder(slide *soml.Slide, elem *soml.Element) string {
	if elem.Appearance == nil || elem.Appearance.Slot == "" {
		return ""
	}
	slot, ok := soml.LayoutSlot(slide.Layout, elem.Appearance.Slot)
	if !ok {
		return ""
	}
	return fmt.Sprintf("%s/%d", slot.PhType, slot.PhIndex)
}

// webParas는 블록 하나의 문단을 더한다 — 블록은 문단 하나다 (`<br>`은 문단을 쪼개지 않는다).
func webParas(paras []LayoutPara, elem *soml.Element, style soml.TextStyle, theme soml.Theme, prefix, bullet string) []LayoutPara {
	runs := webRuns(nil, elem, style, theme, "")
	if len(runs) == 0 {
		return paras
	}
	return append(paras, LayoutPara{Prefix: prefix, Align: string(style.Align), Bullet: bullet, Runs: runs})
}

func webListParas(paras []LayoutPara, list *soml.Element, theme soml.Theme) []LayoutPara {
	for _, child := range list.Children {
		li, ok := child.(*soml.Element)
		if !ok || li.TextStyle() == nil {
			continue
		}
		paras = webParas(paras, li, *li.TextStyle(), theme, "", list.Name)
	}
	return paras
}

func webTableParas(table *soml.Element, theme soml.Theme) []LayoutPara {
	var paras []LayoutPara
	row := 0
	for _, child := range table.Children {
		tr, ok := child.(*soml.Element)
		if !ok {
			continue
		}
		for _, c := range tr.Children {
			cell, ok := c.(*soml.Element)
			if !ok || cell.CellStyle() == nil || cell.CellStyle().Text == nil {
				continue
			}
			paras = webParas(paras, cell, *cell.CellStyle().Text, theme, fmt.Sprintf("r%d", row), "")
		}
		row++
	}
	return paras
}

// webRuns는 블록 안 문자 데이터 잎마다 런을 낸다 — 인라인은 자기 확정 외관으로 내려간다.
// 원문 개행(`<pre>`)과 `<br>`은 ⏎다. 빈 줄은 런을 남기지 않는다 (렌더러와 같은 규칙).
func webRuns(runs []string, parent *soml.Element, style soml.TextStyle, theme soml.Theme, link string) []string {
	for _, child := range parent.Children {
		switch node := child.(type) {
		case *soml.Text:
			for i, line := range strings.Split(node.Value, "\n") {
				if i > 0 {
					runs = append(runs, "⏎")
				}
				line = strings.TrimSuffix(line, "\r")
				if line != "" {
					runs = append(runs, formatRun(line, webRunStyle(style, theme, link)))
				}
			}
		case *soml.Element:
			if node.Name == soml.BreakElement {
				runs = append(runs, "⏎")
				continue
			}
			childStyle := node.TextStyle()
			if childStyle == nil {
				continue
			}
			childLink := link
			if node.Appearance.Link != "" {
				childLink = node.Appearance.Link
			}
			runs = webRuns(runs, node, *childStyle, theme, childLink)
		}
	}
	return runs
}

func webRunStyle(style soml.TextStyle, theme soml.Theme, link string) runStyle {
	return runStyle{
		font:      string(style.Font),
		size:      int64(math.Round(style.Size * 0.75 * 100)),
		bold:      style.Weight >= parityBoldWeight,
		italic:    style.Italic,
		underline: style.Underline,
		color:     webColor(style.Color),
		link:      link,
		// 자간은 렌더러와 따로 환산한다 — em × 크기(px) × 0.75 × 100 = 1/100pt.
		tracking: int64(math.Round(style.Tracking * style.Size * 0.75 * 100)),
		upper:    style.Case == soml.CaseUpper,
		mark:     webHighlight(style.Highlight),
	}
}

// webHighlight는 런의 형광펜 색이다 (없으면 빈 문자열 — 줄에 나오지 않는다).
func webHighlight(c *soml.Color) string {
	if c == nil {
		return ""
	}
	return webColor(*c)
}

func webColor(c soml.Color) string {
	switch {
	case c.Gradient != nil:
		stops := make([]string, len(c.Gradient.Stops))
		for i, stop := range c.Gradient.Stops {
			stops[i] = webColor(stop)
		}
		if c.Gradient.Radial {
			return "radial(" + strings.Join(stops, ",") + ")"
		}
		return "gradient(" + strings.Join(stops, ",") + ")"
	case c.Token != "":
		return c.Token
	case c.RGB != "":
		return "#" + strings.ToUpper(c.RGB)
	}
	return "?"
}

func webFill(f *soml.Fill, theme soml.Theme) string {
	if f == nil {
		return "?"
	}
	if f.None {
		return "none"
	}
	return webColor(f.Color)
}

func webLine(b *soml.Border, theme soml.Theme) string {
	if b == nil {
		return "?"
	}
	if b.None {
		return "none"
	}
	return webColor(b.Color)
}

// ── 런 요약: 두 쪽이 같은 형식으로 쓴다 ─────────────────────────────────────

type runStyle struct {
	// field는 계산되는 필드다 (`a:fld` — 크롬의 쪽 번호).
	field                   bool
	font                    string
	size                    int64
	bold, italic, underline bool
	color                   string
	link                    string
	// tracking은 자간(1/100pt)이고 upper는 대문자 표시다 — 정했을 때만 줄에 나온다.
	tracking int64
	upper    bool
	// mark는 형광펜 색이다 (토큰 이름 · 리터럴 — 없으면 빈 문자열).
	mark string
}

func formatRun(text string, s runStyle) string {
	var attrs []string
	if s.field {
		attrs = append(attrs, "field")
	}
	attrs = append(attrs, s.font, strconv.FormatFloat(float64(s.size)/100, 'f', -1, 64)+"pt")
	if s.bold {
		attrs = append(attrs, "b")
	}
	if s.italic {
		attrs = append(attrs, "i")
	}
	if s.underline {
		attrs = append(attrs, "u")
	}
	if s.tracking != 0 {
		attrs = append(attrs, "tracking="+strconv.FormatFloat(float64(s.tracking)/100, 'f', -1, 64)+"pt")
	}
	if s.upper {
		attrs = append(attrs, "upper")
	}
	attrs = append(attrs, s.color)
	if s.mark != "" {
		attrs = append(attrs, "mark="+s.mark)
	}
	if s.link != "" {
		attrs = append(attrs, "→"+s.link)
	}
	return strconv.Quote(text) + "{" + strings.Join(attrs, " ") + "}"
}

// ── PPTX 쪽: 산출된 spTree를 다시 읽는다 ───────────────────────────────────

type xSlide struct {
	CSld struct {
		Name string `xml:"name,attr"`
		Bg   *struct {
			BgPr struct {
				xFills
				BlipFill *struct{} `xml:"blipFill"`
			} `xml:"bgPr"`
		} `xml:"bg"`
		SpTree struct {
			Items []xObject `xml:",any"`
		} `xml:"spTree"`
	} `xml:"cSld"`
	// ClrMapOvr는 색 매핑 덮기다 — 반전된 장은 bg1이 dk1을 가리킨다.
	ClrMapOvr struct {
		Override *struct {
			Bg1 string `xml:"bg1,attr"`
			Tx1 string `xml:"tx1,attr"`
			Bg2 string `xml:"bg2,attr"`
			Tx2 string `xml:"tx2,attr"`
		} `xml:"overrideClrMapping"`
	} `xml:"clrMapOvr"`
	// Transition은 슬라이드 전환이다 — 효과 요소 이름과 속도를 되읽는다.
	Transition *struct {
		Spd  string    `xml:"spd,attr"`
		Fade *struct{} `xml:"fade"`
		Push *struct {
			Dir string `xml:"dir,attr"`
		} `xml:"push"`
	} `xml:"transition"`
	// Timing은 등장 효과다 — 효과마다 대상 도형 번호 · 효과 번호 · 지연 · 길이를 되읽는다.
	Timing *struct {
		Effects []xEffect `xml:"tnLst>par>cTn>childTnLst>seq>cTn>childTnLst>par>cTn>childTnLst>par>cTn>childTnLst>par"`
	} `xml:"timing"`
}

// xEffect는 등장 효과 하나다 (`p:par > p:cTn[presetClass=entr]`).
type xEffect struct {
	CTn struct {
		PresetID  int    `xml:"presetID,attr"`
		PresetCls string `xml:"presetClass,attr"`
		Cond      struct {
			Delay int `xml:"delay,attr"`
		} `xml:"stCondLst>cond"`
		Set struct {
			SpTgt struct {
				Spid string `xml:"spid,attr"`
			} `xml:"cBhvr>tgtEl>spTgt"`
		} `xml:"childTnLst>set"`
		Effect []struct {
			CTn struct {
				Dur int `xml:"dur,attr"`
			} `xml:"cBhvr>cTn"`
		} `xml:"childTnLst>animEffect"`
	} `xml:"cTn"`
}

// pptxEnterNames는 PowerPoint 등장 효과 번호 → SOML 효과 이름이다 (렌더러의 표를 부르지 않고 따로 적는다).
var pptxEnterNames = map[int]string{10: "fade", 42: "rise", 53: "zoom", 22: "wipe"}

type xNv struct {
	CNvPr struct {
		ID    string `xml:"id,attr"`
		Name  string `xml:"name,attr"`
		Descr string `xml:"descr,attr"`
	} `xml:"cNvPr"`
	CNvCxnSpPr struct {
		StCxn  *xCxn `xml:"stCxn"`
		EndCxn *xCxn `xml:"endCxn"`
	} `xml:"cNvCxnSpPr"`
	CNvSpPr struct {
		TxBox string `xml:"txBox,attr"`
	} `xml:"cNvSpPr"`
	NvPr struct {
		Ph *struct {
			Type string `xml:"type,attr"`
			Idx  string `xml:"idx,attr"`
		} `xml:"ph"`
	} `xml:"nvPr"`
}

// xCxn은 연결선 끝이 붙은 도형(`cNvPr@id`)과 연결점 번호다.
type xCxn struct {
	ID  string `xml:"id,attr"`
	Idx string `xml:"idx,attr"`
}

type xXfrm struct {
	Rot   int64  `xml:"rot,attr"`
	FlipH string `xml:"flipH,attr"`
	FlipV string `xml:"flipV,attr"`
	Off   struct {
		X int64 `xml:"x,attr"`
		Y int64 `xml:"y,attr"`
	} `xml:"off"`
	Ext struct {
		Cx int64 `xml:"cx,attr"`
		Cy int64 `xml:"cy,attr"`
	} `xml:"ext"`
}

type xColorVal struct {
	Val string `xml:"val,attr"`
}

type xColor struct {
	SchemeClr *xColorVal `xml:"schemeClr"`
	SrgbClr   *xColorVal `xml:"srgbClr"`
}

type xFills struct {
	NoFill   *struct{} `xml:"noFill"`
	GradFill *struct {
		Gs   []xColor  `xml:"gsLst>gs"`
		Path *struct{} `xml:"path"`
	} `xml:"gradFill"`
	SolidFill *xColor `xml:"solidFill"`
}

type xTxBody struct {
	P []struct {
		PPr struct {
			Algn      string    `xml:"algn,attr"`
			BuChar    *struct{} `xml:"buChar"`
			BuAutoNum *struct{} `xml:"buAutoNum"`
		} `xml:"pPr"`
		Items []xRunItem `xml:",any"`
	} `xml:"p"`
}

type xRunItem struct {
	XMLName xml.Name
	RPr     struct {
		Sz int64  `xml:"sz,attr"`
		B  string `xml:"b,attr"`
		I  string `xml:"i,attr"`
		U  string `xml:"u,attr"`
		// Spc는 자간(1/100pt), Cap은 대문자 표시다.
		Spc int64  `xml:"spc,attr"`
		Cap string `xml:"cap,attr"`
		xFills
		Highlight *xColor `xml:"highlight"`
		Latin     struct {
			Typeface string `xml:"typeface,attr"`
		} `xml:"latin"`
		HlinkClick *struct {
			ID string `xml:"http://schemas.openxmlformats.org/officeDocument/2006/relationships id,attr"`
		} `xml:"hlinkClick"`
	} `xml:"rPr"`
	T string `xml:"t"`
}

type xObject struct {
	XMLName          xml.Name
	NvSpPr           *xNv `xml:"nvSpPr"`
	NvPicPr          *xNv `xml:"nvPicPr"`
	NvGraphicFramePr *xNv `xml:"nvGraphicFramePr"`
	NvCxnSpPr        *xNv `xml:"nvCxnSpPr"`
	SpPr             struct {
		Xfrm     xXfrm `xml:"xfrm"`
		PrstGeom struct {
			Prst string `xml:"prst,attr"`
			Gd   []struct {
				Name string `xml:"name,attr"`
				Fmla string `xml:"fmla,attr"`
			} `xml:"avLst>gd"`
		} `xml:"prstGeom"`
		// CustGeom은 경로 도형의 사용자 지오메트리다 (`shape:custGeom`).
		CustGeom *struct{} `xml:"custGeom"`
		xFills
		Ln *struct {
			xFills
			PrstDash struct {
				Val string `xml:"val,attr"`
			} `xml:"prstDash"`
			HeadEnd struct {
				Type string `xml:"type,attr"`
			} `xml:"headEnd"`
			TailEnd struct {
				Type string `xml:"type,attr"`
			} `xml:"tailEnd"`
		} `xml:"ln"`
	} `xml:"spPr"`
	Xfrm    xXfrm    `xml:"xfrm"`
	TxBody  *xTxBody `xml:"txBody"`
	Graphic struct {
		GraphicData struct {
			Tbl *struct {
				Tr []struct {
					Tc []struct {
						HMerge string  `xml:"hMerge,attr"`
						VMerge string  `xml:"vMerge,attr"`
						TxBody xTxBody `xml:"txBody"`
					} `xml:"tc"`
				} `xml:"tr"`
			} `xml:"tbl"`
		} `xml:"graphicData"`
	} `xml:"graphic"`
}

type xRels struct {
	Rel []struct {
		ID     string `xml:"Id,attr"`
		Target string `xml:"Target,attr"`
	} `xml:"Relationship"`
}

// PPTXLayout은 PPTX 패키지의 파트(PPTXParts)에서 슬라이드마다 컴포넌트 트리를 다시 읽는다.
// 색·글꼴 참조는 테마 표로 토큰 이름에 되돌린다 — 웹 쪽 트리가 토큰 이름을 들고 있다.
func PPTXLayout(parts map[string][]byte, slideCount int, theme soml.Theme) ([]LayoutSlide, error) {
	var slides []LayoutSlide
	for i := 1; i <= slideCount; i++ {
		name := fmt.Sprintf("ppt/slides/slide%d.xml", i)
		data, ok := parts[name]
		if !ok {
			return nil, fmt.Errorf("part %s is missing", name)
		}
		var x xSlide
		if err := xml.Unmarshal(data, &x); err != nil {
			return nil, fmt.Errorf("%s: %w", name, err)
		}
		links := map[string]string{}
		if relData, ok := parts[path.Join("ppt/slides/_rels", path.Base(name)+".rels")]; ok {
			var rels xRels
			if err := xml.Unmarshal(relData, &rels); err != nil {
				return nil, fmt.Errorf("%s rels: %w", name, err)
			}
			for _, r := range rels.Rel {
				links[r.ID] = r.Target
			}
		}
		s := LayoutSlide{ID: x.CSld.Name}
		if o := x.ClrMapOvr.Override; o != nil {
			s.Inverse = o.Bg1 == "dk1" && o.Tx1 == "lt1" && o.Bg2 == "dk2" && o.Tx2 == "lt2"
		}
		if t := x.Transition; t != nil {
			kind := "?"
			switch {
			case t.Fade != nil:
				kind = "fade"
			case t.Push != nil && t.Push.Dir == "l":
				kind = "push"
			}
			s.Transition = kind + "/" + t.Spd
		}
		if bg := x.CSld.Bg; bg != nil {
			if bg.BgPr.BlipFill != nil {
				s.Background = "image"
			} else {
				s.Background = "fill=" + pptxFill(bg.BgPr.xFills, theme)
			}
		}
		// 연결선의 붙은 끝은 도형 번호(`cNvPr@id`)로 적혀 있다 — 트리에는 이름으로 되돌려 싣는다.
		names := map[string]string{}
		for _, obj := range x.CSld.SpTree.Items {
			for _, nv := range []*xNv{obj.NvSpPr, obj.NvPicPr, obj.NvGraphicFramePr, obj.NvCxnSpPr} {
				if nv != nil {
					names[nv.CNvPr.ID] = nv.CNvPr.Name
				}
			}
		}
		// 등장은 대상 도형 번호로 객체에 붙는다.
		enters := map[string]string{}
		if x.Timing != nil {
			for _, e := range x.Timing.Effects {
				name, ok := pptxEnterNames[e.CTn.PresetID]
				if !ok || e.CTn.PresetCls != "entr" {
					name = fmt.Sprintf("preset%d", e.CTn.PresetID)
				}
				dur := 0
				if len(e.CTn.Effect) > 0 {
					dur = e.CTn.Effect[0].CTn.Dur
				}
				enters[e.CTn.Set.SpTgt.Spid] = fmt.Sprintf("enter=%s@%d+%d", name, e.CTn.Cond.Delay, dur)
			}
		}
		for _, obj := range x.CSld.SpTree.Items {
			n, ok := pptxNode(obj, theme, links, names)
			if ok {
				for _, nv := range []*xNv{obj.NvSpPr, obj.NvPicPr, obj.NvGraphicFramePr, obj.NvCxnSpPr} {
					if nv == nil {
						continue
					}
					if e, ok := enters[nv.CNvPr.ID]; ok {
						if n.Extra != "" {
							n.Extra += " "
						}
						n.Extra += e
					}
				}
				s.Nodes = append(s.Nodes, n)
			}
		}
		slides = append(slides, s)
	}
	return slides, nil
}

func pptxNode(obj xObject, theme soml.Theme, links map[string]string, names map[string]string) (LayoutNode, bool) {
	var n LayoutNode
	var nv *xNv
	xfrm := obj.SpPr.Xfrm
	switch obj.XMLName.Local {
	case "sp":
		nv = obj.NvSpPr
		if nv != nil && nv.CNvSpPr.TxBox == "1" {
			n.Kind = "textbox"
		} else {
			n.Kind = "shape:" + obj.SpPr.PrstGeom.Prst
			if obj.SpPr.CustGeom != nil {
				n.Kind = "shape:custGeom"
			}
			n.Fill = pptxFill(obj.SpPr.xFills, theme)
			n.Line = "?"
			if obj.SpPr.Ln != nil {
				n.Line = pptxFill(obj.SpPr.Ln.xFills, theme)
			}
		}
		if obj.TxBody != nil {
			n.Paras = pptxParas(nil, *obj.TxBody, "", theme, links)
		}
	case "pic":
		nv = obj.NvPicPr
		n.Kind = "pic"
	case "graphicFrame":
		nv = obj.NvGraphicFramePr
		xfrm = obj.Xfrm
		n.Kind = "graphicFrame"
		if tbl := obj.Graphic.GraphicData.Tbl; tbl != nil {
			n.Kind = "table"
			for r, tr := range tbl.Tr {
				for _, tc := range tr.Tc {
					if tc.HMerge == "1" || tc.VMerge == "1" {
						continue
					}
					n.Paras = pptxParas(n.Paras, tc.TxBody, fmt.Sprintf("r%d", r), theme, links)
				}
			}
		}
	case "cxnSp":
		nv = obj.NvCxnSpPr
		n.Kind = "connector:" + obj.SpPr.PrstGeom.Prst
		n.Line = "?"
		var extra []string
		adj := "50000"
		for _, gd := range obj.SpPr.PrstGeom.Gd {
			if gd.Name == "adj1" {
				adj = strings.TrimPrefix(gd.Fmla, "val ")
			}
		}
		x := obj.SpPr.Xfrm
		extra = append(extra, fmt.Sprintf("rot=%d", x.Rot/60000))
		if x.FlipH == "1" {
			extra = append(extra, "flipH")
		}
		if x.FlipV == "1" {
			extra = append(extra, "flipV")
		}
		extra = append(extra, "adj="+adj)
		glue := func(which string, c *xCxn) string {
			if c == nil {
				return which + "=-"
			}
			return which + "=" + strconv.Quote(names[c.ID]) + ":" + c.Idx
		}
		if nv != nil {
			extra = append(extra, glue("from", nv.CNvCxnSpPr.StCxn), glue("to", nv.CNvCxnSpPr.EndCxn))
		}
		if ln := obj.SpPr.Ln; ln != nil {
			n.Line = pptxFill(ln.xFills, theme)
			extra = append(extra, "dash="+ln.PrstDash.Val, "ends="+ln.HeadEnd.Type+"/"+ln.TailEnd.Type)
		}
		n.Extra = strings.Join(extra, " ")
	default:
		// nvGrpSpPr · grpSpPr — 트리의 뿌리 속성이지 객체가 아니다.
		return n, false
	}
	if nv != nil {
		n.Name = nv.CNvPr.Name
		n.Alt = nv.CNvPr.Descr
		if ph := nv.NvPr.Ph; ph != nil {
			idx := ph.Idx
			if idx == "" {
				idx = "0"
			}
			n.Ph = ph.Type + "/" + idx
		}
	}
	n.X, n.Y = xfrm.Off.X, xfrm.Off.Y
	n.W, n.H = xfrm.Ext.Cx, xfrm.Ext.Cy
	return n, true
}

var pptxAligns = map[string]string{
	"l":    string(soml.AlignStart),
	"ctr":  string(soml.AlignCenter),
	"r":    string(soml.AlignEnd),
	"just": string(soml.AlignJustify),
	"":     string(soml.AlignStart),
}

func pptxParas(paras []LayoutPara, body xTxBody, prefix string, theme soml.Theme, links map[string]string) []LayoutPara {
	for _, p := range body.P {
		var runs []string
		for _, item := range p.Items {
			switch item.XMLName.Local {
			case "br":
				runs = append(runs, "⏎")
			case "r", "fld":
				link := ""
				if h := item.RPr.HlinkClick; h != nil {
					link = links[h.ID]
				}
				runs = append(runs, formatRun(item.T, runStyle{
					field:     item.XMLName.Local == "fld",
					font:      pptxFont(item.RPr.Latin.Typeface, theme),
					size:      item.RPr.Sz,
					bold:      item.RPr.B == "1",
					italic:    item.RPr.I == "1",
					underline: item.RPr.U != "" && item.RPr.U != "none",
					color:     pptxColor(item.RPr.xFills, theme),
					link:      link,
					tracking:  item.RPr.Spc,
					upper:     item.RPr.Cap == "all",
					mark:      pptxHighlight(item.RPr.Highlight, theme),
				}))
			}
		}
		if len(runs) == 0 {
			continue
		}
		bullet := ""
		switch {
		case p.PPr.BuChar != nil:
			bullet = "ul"
		case p.PPr.BuAutoNum != nil:
			bullet = "ol"
		}
		align, ok := pptxAligns[p.PPr.Algn]
		if !ok {
			align = "algn=" + p.PPr.Algn
		}
		paras = append(paras, LayoutPara{Prefix: prefix, Align: align, Bullet: bullet, Runs: runs})
	}
	return paras
}

// pptxFont는 typeface를 폰트 토큰 이름에 되돌린다 — 테마 참조(`+mj-lt`/`+mn-lt`)는
// fontScheme 슬롯으로, 리터럴(mono)은 latin 이름으로 찾는다.
func pptxFont(typeface string, theme soml.Theme) string {
	for _, f := range theme.Fonts {
		switch {
		case f.Scheme == "major" && typeface == "+mj-lt",
			f.Scheme == "minor" && typeface == "+mn-lt",
			f.Scheme == "" && typeface == f.Latin:
			return string(f.Name)
		}
	}
	return "typeface=" + typeface
}

func pptxColor(f xFills, theme soml.Theme) string {
	switch {
	case f.GradFill != nil:
		stops := make([]string, len(f.GradFill.Gs))
		for i, gs := range f.GradFill.Gs {
			stops[i] = pptxSolid(&gs, theme)
		}
		if f.GradFill.Path != nil {
			return "radial(" + strings.Join(stops, ",") + ")"
		}
		return "gradient(" + strings.Join(stops, ",") + ")"
	case f.SolidFill != nil:
		return pptxSolid(f.SolidFill, theme)
	}
	return "?"
}

// pptxHighlight는 런의 `a:highlight` 색이다 (없으면 빈 문자열).
func pptxHighlight(c *xColor, theme soml.Theme) string {
	if c == nil {
		return ""
	}
	return pptxSolid(c, theme)
}

// pptxSolid는 색 요소 하나를 토큰 이름(`schemeClr` → 테마 표의 Ref)이나 `#RRGGBB`로 되돌린다.
func pptxSolid(c *xColor, theme soml.Theme) string {
	switch {
	case c.SchemeClr != nil:
		for _, token := range theme.Colors {
			if token.Ref == c.SchemeClr.Val {
				return token.Name
			}
		}
		return "schemeClr=" + c.SchemeClr.Val
	case c.SrgbClr != nil:
		return "#" + strings.ToUpper(c.SrgbClr.Val)
	}
	return "?"
}

func pptxFill(f xFills, theme soml.Theme) string {
	if f.NoFill != nil {
		return "none"
	}
	return pptxColor(f, theme)
}
