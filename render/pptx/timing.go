package pptx

import (
	"fmt"
	"strings"

	"github.com/k2sebeom/balpyo/render/route"
	"github.com/k2sebeom/balpyo/soml"
)

// 이 파일은 등장(`enter`)의 PPTX 렌더다 — 슬라이드의 `p:timing`.
//
// 등장 하나는 그 요소 아래의 **객체마다** 효과 하나가 된다 (PowerPoint의 효과는 도형마다 걸린다).
// 객체는 가장 가까운 `enter` 조상의 등장에 속한다 — 컨테이너는 도형이 되지 않으므로 컨테이너의 등장은
// 그 자손 객체들이 함께 나타나는 것이다. 모든 효과는 슬라이드가 시작될 때 자동으로 시작하는 한 묶음
// 안에서 "이전 효과와 함께" + 지연으로 엇갈린다 (첫 효과만 "이전 효과 다음") — PowerPoint의 애니메이션
// 창에 그대로 나열되고 편집된다.

// enterPreset은 효과 이름 → PowerPoint 기본 등장 효과(presetID · presetSubtype)다.
var enterPreset = map[soml.EnterEffect][2]int{
	soml.EnterFade: {10, 0},  // 페이드
	soml.EnterRise: {42, 0},  // 떠오르기 (Float In)
	soml.EnterZoom: {53, 16}, // 확대 (Zoom — 개체 가운데)
	soml.EnterWipe: {22, 8},  // 닦아내기 (Wipe — 왼쪽에서)
}

// wipeFilter · wipeSubtype은 닦아내기가 시작하는 변 → PowerPoint의 필터 · 하위 종류다 (위 1 · 오른쪽 2 ·
// 아래 4 · 왼쪽 8).
var (
	wipeFilter  = map[route.Side]string{route.Left: "wipe(left)", route.Right: "wipe(right)", route.Top: "wipe(up)", route.Bottom: "wipe(down)"}
	wipeSubtype = map[route.Side]int{route.Top: 1, route.Right: 2, route.Bottom: 4, route.Left: 8}
)

// riseOffset은 떠오르기가 출발하는 거리다 — 슬라이드 높이의 비율(`#ppt_y+…`). 웹의 24px(720px 스테이지)과
// 같은 길이다.
const riseOffset = "0.0333"

// zoomFrom은 확대가 출발하는 크기 비율이다 — 웹의 `scale: 0.9`와 같다.
const zoomFrom = "0.9"

// entranceObject는 효과 하나의 대상이다.
type entranceObject struct {
	entrance *soml.Entrance
	shapeID  int
	kind     soml.ElementKind
	// from은 그려지는 연결선의 닦아내기가 시작하는 쪽이다 (draw만).
	from route.Side
}

// entranceObjects는 슬라이드의 객체를 등장 순서 · 문서 순서로 모은다 — 객체(도형 번호를 받은 요소)마다
// 가장 가까운 `enter` 조상의 등장을 붙인다.
func entranceObjects(slide *soml.Slide, rels *slideRels) []entranceObject {
	shapeIDs := rels.shapeIDs
	var out []entranceObject
	var walk func(node soml.Node, current *soml.Entrance)
	walk = func(node soml.Node, current *soml.Entrance) {
		elem, ok := node.(*soml.Element)
		if !ok {
			return
		}
		if e := elem.EnterOf(); e != nil {
			current = e
		}
		if id, ok := shapeIDs[elem]; ok && current != nil {
			kind := soml.KindOf(elem.Name)
			out = append(out, entranceObject{entrance: current, shapeID: id, kind: kind, from: rels.drawFrom[elem]})
			if id, ok := rels.labelIDs[elem]; ok {
				// 연결선의 라벨은 그 선과 같은 시각에 페이드로 나타난다 (웹의 라벨과 같다).
				fade := *current
				fade.Effect = soml.EnterFade
				out = append(out, entranceObject{entrance: &fade, shapeID: id, kind: soml.KindBlock})
			}
		}
		for _, child := range elem.Children {
			walk(child, current)
		}
	}
	for _, node := range slide.Content {
		walk(node, nil)
	}
	// 등장 순서로 고르게 정렬한다 (같은 등장 안은 문서 순서 그대로 — 안정 정렬).
	sorted := make([]entranceObject, 0, len(out))
	maxOrder := -1
	for _, o := range out {
		if o.entrance.Order > maxOrder {
			maxOrder = o.entrance.Order
		}
	}
	for order := 0; order <= maxOrder; order++ {
		for _, o := range out {
			if o.entrance.Order == order {
				sorted = append(sorted, o)
			}
		}
	}
	return sorted
}

// timingXML은 슬라이드의 `p:timing`이다 — 등장이 없으면 빈 문자열이다 (슬라이드 바이트가 그대로다).
func timingXML(slide *soml.Slide, rels *slideRels) (string, error) {
	objects := entranceObjects(slide, rels)
	if len(objects) == 0 {
		return "", nil
	}
	var b strings.Builder
	w := func(depth int, line string) { b.WriteString("\n" + strings.Repeat("  ", depth+1) + line) }
	id := 0
	next := func() int { id++; return id }

	w(0, "<p:timing>")
	w(1, "<p:tnLst>")
	w(2, "<p:par>")
	w(3, fmt.Sprintf(`<p:cTn id="%d" dur="indefinite" restart="never" nodeType="tmRoot">`, next()))
	w(4, "<p:childTnLst>")
	w(5, `<p:seq concurrent="1" nextAc="seek">`)
	mainSeq := next()
	w(6, fmt.Sprintf(`<p:cTn id="%d" dur="indefinite" nodeType="mainSeq">`, mainSeq))
	w(7, "<p:childTnLst>")
	w(8, "<p:par>")
	// 이 묶음은 클릭을 기다리지 않는다 — 주 시퀀스가 시작될 때(슬라이드 진입) 함께 시작한다.
	w(9, fmt.Sprintf(`<p:cTn id="%d" fill="hold">`, next()))
	w(10, "<p:stCondLst>")
	w(11, `<p:cond delay="indefinite"/>`)
	w(11, fmt.Sprintf(`<p:cond evt="onBegin" delay="0"><p:tn val="%d"/></p:cond>`, mainSeq))
	w(10, "</p:stCondLst>")
	w(10, "<p:childTnLst>")
	w(11, "<p:par>")
	w(12, fmt.Sprintf(`<p:cTn id="%d" fill="hold">`, next()))
	w(13, `<p:stCondLst><p:cond delay="0"/></p:stCondLst>`)
	w(13, "<p:childTnLst>")
	for i, o := range objects {
		node := "withEffect"
		if i == 0 {
			node = "afterEffect"
		}
		if err := writeEntranceEffect(w, 14, o, node, next); err != nil {
			return "", err
		}
	}
	w(13, "</p:childTnLst>")
	w(12, "</p:cTn>")
	w(11, "</p:par>")
	w(10, "</p:childTnLst>")
	w(9, "</p:cTn>")
	w(8, "</p:par>")
	w(7, "</p:childTnLst>")
	w(6, "</p:cTn>")
	w(6, `<p:prevCondLst><p:cond evt="onPrev" delay="0"><p:tgtEl><p:sldTgt/></p:tgtEl></p:cond></p:prevCondLst>`)
	w(6, `<p:nextCondLst><p:cond evt="onNext" delay="0"><p:tgtEl><p:sldTgt/></p:tgtEl></p:cond></p:nextCondLst>`)
	w(5, "</p:seq>")
	w(4, "</p:childTnLst>")
	w(3, "</p:cTn>")
	w(2, "</p:par>")
	w(1, "</p:tnLst>")
	// 짓기 목록 — 글을 담는 도형은 "개체 하나로" 짓고(문단 단위가 아니다), 표는 통째로 짓는다. 그림은
	// 목록에 오지 않는다 (PowerPoint도 쓰지 않는다).
	var builds []string
	for _, o := range objects {
		switch o.kind {
		case soml.KindImage, soml.KindConnector:
		case soml.KindTable:
			builds = append(builds, fmt.Sprintf(`<p:bldGraphic spid="%d" grpId="0"><p:bldAsOne/></p:bldGraphic>`, o.shapeID))
		default:
			builds = append(builds, fmt.Sprintf(`<p:bldP spid="%d" grpId="0" animBg="1"/>`, o.shapeID))
		}
	}
	if len(builds) > 0 {
		w(1, "<p:bldLst>")
		for _, line := range builds {
			w(2, line)
		}
		w(1, "</p:bldLst>")
	}
	w(0, "</p:timing>")
	return b.String(), nil
}

// writeEntranceEffect는 객체 하나의 등장 효과다 — 보이게 하기(`set style.visibility`) + 효과의 동작들.
func writeEntranceEffect(w func(int, string), depth int, o entranceObject, node string, next func() int) error {
	effect := o.entrance.Effect
	wipe := "wipe(left)"
	if effect == soml.EnterDraw {
		// 선 그리기는 선이 시작하는 쪽에서 닦아내는 것이다 (PowerPoint에 경로를 따라 그리는 등장이 없다 —
		// 가장 가까운 원어다).
		effect = soml.EnterWipe
		wipe = wipeFilter[o.from]
	}
	preset, ok := enterPreset[effect]
	if !ok {
		return fmt.Errorf("enter %q has no PowerPoint effect", string(o.entrance.Effect))
	}
	dur := o.entrance.Duration
	tgt := fmt.Sprintf(`<p:tgtEl><p:spTgt spid="%d"/></p:tgtEl>`, o.shapeID)
	w(depth, "<p:par>")
	subtype := preset[1]
	if o.entrance.Effect == soml.EnterDraw {
		subtype = wipeSubtype[o.from]
	}
	w(depth+1, fmt.Sprintf(`<p:cTn id="%d" presetID="%d" presetClass="entr" presetSubtype="%d" fill="hold" grpId="0" nodeType="%s">`,
		next(), preset[0], subtype, node))
	w(depth+2, fmt.Sprintf(`<p:stCondLst><p:cond delay="%d"/></p:stCondLst>`, o.entrance.Delay))
	w(depth+2, "<p:childTnLst>")
	w(depth+3, "<p:set>")
	w(depth+4, fmt.Sprintf(`<p:cBhvr><p:cTn id="%d" dur="1" fill="hold"><p:stCondLst><p:cond delay="0"/></p:stCondLst></p:cTn>%s<p:attrNameLst><p:attrName>style.visibility</p:attrName></p:attrNameLst></p:cBhvr>`, next(), tgt))
	w(depth+4, `<p:to><p:strVal val="visible"/></p:to>`)
	w(depth+3, "</p:set>")
	fade := func() {
		w(depth+3, fmt.Sprintf(`<p:animEffect transition="in" filter="fade"><p:cBhvr><p:cTn id="%d" dur="%d"/>%s</p:cBhvr></p:animEffect>`, next(), dur, tgt))
	}
	anim := func(attr, from, to string) {
		w(depth+3, `<p:anim calcmode="lin" valueType="num">`)
		w(depth+4, fmt.Sprintf(`<p:cBhvr><p:cTn id="%d" dur="%d" decel="100000" fill="hold"/>%s<p:attrNameLst><p:attrName>%s</p:attrName></p:attrNameLst></p:cBhvr>`, next(), dur, tgt, attr))
		w(depth+4, fmt.Sprintf(`<p:tavLst><p:tav tm="0"><p:val><p:strVal val="%s"/></p:val></p:tav><p:tav tm="100000"><p:val><p:strVal val="%s"/></p:val></p:tav></p:tavLst>`, from, to))
		w(depth+3, "</p:anim>")
	}
	switch effect {
	case soml.EnterFade:
		fade()
	case soml.EnterRise:
		fade()
		anim("ppt_x", "#ppt_x", "#ppt_x")
		anim("ppt_y", "#ppt_y+"+riseOffset, "#ppt_y")
	case soml.EnterZoom:
		fade()
		anim("ppt_w", "#ppt_w*"+zoomFrom, "#ppt_w")
		anim("ppt_h", "#ppt_h*"+zoomFrom, "#ppt_h")
	case soml.EnterWipe:
		w(depth+3, fmt.Sprintf(`<p:animEffect transition="in" filter="%s"><p:cBhvr><p:cTn id="%d" dur="%d"/>%s</p:cBhvr></p:animEffect>`, wipe, next(), dur, tgt))
	}
	w(depth+2, "</p:childTnLst>")
	w(depth+1, "</p:cTn>")
	w(depth, "</p:par>")
	return nil
}

// transitionXML은 슬라이드의 `p:transition`이다 — 전환이 없으면 빈 문자열이다. 길이는 가장 가까운 속도
// 단계로 접힌다 (fast 0.5초 · med 0.75초 · slow 1초 — 정밀한 길이는 확장 네임스페이스의 것이다).
func transitionXML(slide *soml.Slide) (string, error) {
	var effect string
	switch slide.Transition {
	case "":
		return "", nil
	case soml.TransitionFade:
		effect = "<p:fade/>"
	case soml.TransitionPush:
		// 들어오는 장이 오른쪽에서 밀고 들어온다 — 웹의 다음 넘김과 같은 방향이다.
		effect = `<p:push dir="l"/>`
	default:
		return "", fmt.Errorf("transition %q has no PowerPoint mapping", string(slide.Transition))
	}
	return fmt.Sprintf("\n  <p:transition spd=%q>%s</p:transition>", TransitionSpeed(slide.TransitionDuration), effect), nil
}

// TransitionSpeed는 전환 길이(ms)를 PowerPoint의 속도 단계(500 · 750 · 1000ms)로 접는다 — 600ms까지 fast, 875ms까지
// med, 그보다 길면 slow다.
func TransitionSpeed(ms int) string {
	switch {
	case ms <= 600:
		return "fast"
	case ms <= 875:
		return "med"
	}
	return "slow"
}
