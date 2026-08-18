package pptx

import (
	"fmt"
	"strings"

	"github.com/k2sebeom/balpyo/render/geom"
	"github.com/k2sebeom/balpyo/render/web"
	"github.com/k2sebeom/balpyo/soml"
)

// boldWeight는 rPr@b를 켜는 최소 font-weight다.
//
// OOXML 런 속성에는 굵기 단계가 없어 수치를 켬/끔으로 눕혀야 한다 (soml.TextStyle.Weight의
// 주석이 이 판단을 렌더러에 남겼다). 600을 경계로 두는 이유: regular(400)·bold(700) 두 벌만
// 있는 폰트에서 CSS 폰트 매칭이 600부터 bold 면을 고른다 — 웹이 굵게 그리는 지점과 PPTX가
// 굵게 내는 지점이 같아야 두 산출물이 갈라지지 않는다.
const boldWeight = 600

// shapeIndent는 spTree 자식의 들여쓰기다 (slideTemplate의 nvGrpSpPr와 같은 깊이).
const shapeIndent = "      "

// firstShapeID는 슬라이드 안 도형 id의 시작값이다. id=1은 spTree의 그룹(nvGrpSpPr)이
// 이미 쓰고 있다.
const firstShapeID = 2

// textBoxSlackRatio는 텍스트 박스 폭에 더하는 안전 여유 비율이다 (PRD §9.3 — "측정된
// 크기에 약간의 안전 여유를 더해 명시 고정").
//
// 값의 근거 (Iteration 1의 측정, NOTES.md 01/07): LibreOffice는 한글↔라틴 스크립트 경계마다
// 간격을 넣어(스크립트 간격 — OOXML로 끌 수 없다) 혼합 문자 줄을 브라우저보다 최대 +2.7%
// 넓게 그린다. 박스를 측정 폭 그대로 고정하면 여유 폭(Slack)이 그보다 작은 줄은 LibreOffice
// 에서 한 줄 늘어난다 — 01_text 픽스처의 본문(여유 2.4%)이 정확히 그렇게 갈라졌다. 4%는
// 측정 최대치에 여유를 더한 값이고, 반대 방향의 위험(브라우저가 넘긴 단어가 넓힌 박스에
// 들어가 줄이 줄어드는 것)은 단어 하나 폭보다 훨씬 작아야 하므로 이 이상 키우지 않는다.
const textBoxSlackRatio = 0.04

// TextBoxRect는 잰 박스에 안전 여유를 더한 PPTX 텍스트 박스 사각형이다.
//
// 넓히는 방향은 정렬이 정한다 — 글자가 닻을 내린 변은 움직이지 않아야 웹과 같은 자리에
// 그려진다: start는 오른쪽으로만, end는 왼쪽으로만, center는 양쪽으로 반씩 넓힌다.
// justify는 넓히지 않는다: 양쪽 정렬은 글줄을 박스 폭까지 늘이므로, 박스를 넓히면 웹보다
// 긴 글줄이 된다 (스크립트 간격 위험은 남지만 자리 어긋남보다 낫다).
func TextBoxRect(r geom.Rect, align soml.TextAlign) geom.Rect {
	if align == soml.AlignJustify {
		return r
	}
	slack := r.W * textBoxSlackRatio
	switch align {
	case soml.AlignCenter:
		r.X -= slack / 2
	case soml.AlignEnd:
		r.X -= slack
	}
	r.W += slack
	return r
}

// shapesXML은 슬라이드 콘텐츠를 p:sp 목록으로 컴파일한다 — 이 함수가 지오메트리(①)와
// 해석된 외관(②)이 OOXML에서 만나는 자리다 (PRD §9.3).
//
// 문서 순서가 곧 spTree 순서다: OOXML은 나중 도형이 위에 그려지므로, 저자가 쓴 순서가
// 겹침 z-순서가 된다 (웹의 문서 순서 스택과 같은 규칙 — Iteration 6의 겹침이 이 위에 선다).
//
// 결과는 빈 문자열(요소 없음)이거나, 각 도형이 "\n"으로 시작해 개행 없이 끝나는 조각이다 —
// slideTemplate의 `{{.Shapes}}` 자리에 그대로 들어간다. 함께 돌려주는 links는 순회가
// 배정한 외부 하이퍼링크 관계 목록이다 — 슬라이드 rels 파트가 같은 배정을 실어야
// hlinkClick의 r:id가 살아난다 (buildParts).
func shapesXML(deck *soml.Deck, slide *soml.Slide, g *geom.Geometry) (string, *slideLinks, error) {
	var b strings.Builder
	links := &slideLinks{}
	id := firstShapeID
	for i, node := range slide.Content {
		if err := writeShapeNode(&b, deck, slide, node, []int{i}, &id, g, links); err != nil {
			return "", nil, err
		}
	}
	return b.String(), links, nil
}

// slideLinks는 슬라이드 하나가 배정한 외부 하이퍼링크 관계 목록이다.
//
// rel id는 결정적이어야 한다 — 같은 덱을 두 번 조립하면 같은 바이트가 나와야 골든 비교가
// 성립한다. 배정은 순회의 문서 순서를 따르고, **같은 URL은 관계 하나를 재사용한다**:
// OPC 관계의 정체는 (Type, Target, TargetMode)이라 같은 목적지의 관계 둘은 정보가 없고,
// PowerPoint도 저장하며 하나로 합치므로 매번 내면 재저장 diff에 잡음만 남는다.
type slideLinks struct {
	urls []string
}

// hyperlinkRelBase는 첫 하이퍼링크 관계의 번호다. rId1은 레이아웃, rId2는 노트 슬라이드
// 자리로 **노트가 없어도 예약**한다 — 노트 유무가 링크 번호를 밀면 노트 한 줄을 더한
// 커밋의 diff에 링크 관계 전부가 잡음으로 실린다. OPC 관계 id는 임의 문자열이라 빈
// 번호는 유효하다 (notesSlidePart의 번호 규칙과 같은 태도).
const hyperlinkRelBase = 3

// relID는 URL의 관계 id다 — 처음 보는 URL이면 다음 번호를 배정한다.
func (l *slideLinks) relID(url string) string {
	for i, seen := range l.urls {
		if seen == url {
			return fmt.Sprintf("rId%d", hyperlinkRelBase+i)
		}
	}
	l.urls = append(l.urls, url)
	return fmt.Sprintf("rId%d", hyperlinkRelBase+len(l.urls)-1)
}

// writeShapeNode는 콘텐츠 노드 하나를 도형으로 쓴다. path 규칙은 웹 렌더러와 같다
// (web.ElementID의 입력 — 지오메트리 조회의 열쇠).
func writeShapeNode(b *strings.Builder, deck *soml.Deck, slide *soml.Slide, node soml.Node, path []int, shapeID *int, g *geom.Geometry, links *slideLinks) error {
	elem, ok := node.(*soml.Element)
	if !ok {
		// 슬라이드 직속 문자 데이터는 파서가 이미 막는다 (modelElements) — 여기 도달했다면
		// 파서를 거치지 않은 덱이므로 조용히 건너뛰지 않고 멈춘다.
		return fmt.Errorf("%s: 슬라이드 콘텐츠에 요소가 아닌 노드 %T가 있다", slide.Source, node)
	}
	// 텍스트 박스가 되는 것은 <text>와 블록 종류 카탈로그다 (soml.KindOf — 렌더러가 이름
	// 목록을 따로 들지 않는 이유는 웹과 같다). 리스트 컨테이너(<ul>/<ol>)는 항목 07의
	// buChar/들여쓰기까지 아래 가드 에러로 남는다 — 조용히 빠지는 대신 멈춘다.
	if elem.Name != soml.TextElement && soml.KindOf(elem.Name) != soml.KindBlock {
		// 웹 렌더러와 같은 이유로 멈춘다 (render/web.writeNode): 어휘에 태그를 추가하고 렌더
		// 매핑을 잊으면 "파싱은 되는데 그려지지 않는" 조용한 실패가 된다.
		return fmt.Errorf("%s:%s <%s>를 PPTX로 렌더하는 매핑이 없다", slide.Source, elem.Span, elem.Name)
	}
	if err := writeTextShape(b, deck, slide, elem, web.ElementID(slide.Index, path), *shapeID, g, links); err != nil {
		return err
	}
	*shapeID++
	return nil
}

// writeTextShape는 텍스트 역할 블록 하나(`<text>`, 서브셋 블록)를 p:sp(텍스트 박스)로 쓴다.
//
// 빈 텍스트·크기 0 박스도 도형을 낸다. 건너뛰면 spTree 순서(=z-순서)와 도형 id가 내용에
// 따라 흔들리고, 저자는 빈 요소가 산출물 어디에 있는지 PowerPoint에서 확인할 수 없다 —
// 크기 0 도형은 OOXML에서 유효하므로 (Extent가 이 판단을 호출자에게 남긴 이유) 그대로 낸다.
func writeTextShape(b *strings.Builder, deck *soml.Deck, slide *soml.Slide, elem *soml.Element, id string, shapeID int, g *geom.Geometry, links *slideLinks) error {
	box, ok := g.Box(id)
	if !ok {
		// checkGeometry의 Verify가 먼저 잡지만, 이 함수 단독 호출에서도 좌표 없는 도형이
		// 원점에 조용히 놓이는 일은 없어야 한다.
		return fmt.Errorf("%s: %s <%s>의 측정된 좌표가 없다", slide.Source, id, elem.Name)
	}
	style := elem.TextStyle()
	if style == nil {
		// 해석되지 않은 외관을 기본값으로 그리면 PPTX만 그럴듯하게 나오고 웹과 갈라진다
		// (render/web.writeText와 같은 계약).
		return fmt.Errorf("%s:%s <%s>의 외관이 해석되지 않았다 — core.Resolve를 거치지 않은 덱이다",
			slide.Source, elem.Span, elem.Name)
	}
	body, err := textBodyXML(elem, *style, deck.Theme, links)
	if err != nil {
		return fmt.Errorf("%s:%s <%s>: %w", slide.Source, elem.Span, elem.Name, err)
	}

	rect := TextBoxRect(box.Rect, style.Align)
	x, y := Offset(rect)
	cx, cy := Extent(rect)

	w := func(depth int, line string) {
		b.WriteString("\n" + shapeIndent + strings.Repeat("  ", depth) + line)
	}
	w(0, "<p:sp>")
	w(1, "<p:nvSpPr>")
	// 이름은 지오메트리 식별자다 — PowerPoint 선택 창과 진단 메시지가 같은 이름으로
	// 같은 요소를 가리킨다.
	w(2, fmt.Sprintf(`<p:cNvPr id="%d" name="%s"/>`, shapeID, escapeXML(elem.Name+" "+id)))
	w(2, `<p:cNvSpPr txBox="1"/>`)
	w(2, "<p:nvPr/>")
	w(1, "</p:nvSpPr>")
	w(1, "<p:spPr>")
	w(2, "<a:xfrm>")
	w(3, fmt.Sprintf(`<a:off x="%d" y="%d"/>`, x, y))
	w(3, fmt.Sprintf(`<a:ext cx="%d" cy="%d"/>`, cx, cy))
	w(2, "</a:xfrm>")
	w(2, `<a:prstGeom prst="rect">`)
	w(3, "<a:avLst/>")
	w(2, "</a:prstGeom>")
	// 텍스트 박스는 배경도 테두리도 갖지 않는다 — 명시하지 않으면 테마 fmtScheme의
	// 기본 채움/선이 상속되어 웹에 없는 사각형이 그려진다.
	w(2, "<a:noFill/>")
	w(2, "<a:ln>")
	w(3, "<a:noFill/>")
	w(2, "</a:ln>")
	w(1, "</p:spPr>")
	b.WriteString(body)
	w(0, "</p:sp>")
	return nil
}

// textBodyXML은 해석된 텍스트 외관과 블록의 자식들을 p:txBody로 컴파일한다 (PRD §9.4의
// `font`/`size`/`weight`/`color`/`align` 행과 "인라인 서식 → 텍스트 런 rPr/hlinkClick" 행).
// 문단 서식은 블록의 확정 스타일에서, 런 서식은 잎을 감싼 가장 안쪽 요소의 확정 스타일에서
// 나온다 — 요소의 Attrs를 다시 읽지 않는다 (render/web.textStyleCSS와 같은 계약).
func textBodyXML(elem *soml.Element, style soml.TextStyle, theme soml.Theme, links *slideLinks) (string, error) {
	algn, err := algnValue(style.Align)
	if err != nil {
		return "", err
	}
	// 블록 자체의 색·폰트도 여기서 먼저 검증한다 — 런이 없는 빈 요소에서도 해석되지 않은
	// 외관은 소리를 내야 한다 (writeTextShape의 빈 도형 정책과 짝).
	if _, err := colorXML(style.Color, theme); err != nil {
		return "", err
	}
	if _, _, err := typefaces(style.Font, theme); err != nil {
		return "", err
	}
	// 웹 렌더러와 같은 검사다 (textStyleCSS) — 해석되지 않은 0 값이 sz="0"으로 흘러가면
	// 파일은 열리지만 글자가 보이지 않는다.
	if style.Size <= 0 {
		return "", fmt.Errorf("폰트 크기가 해석되지 않았다 (%v)", style.Size)
	}
	if style.Weight <= 0 {
		return "", fmt.Errorf("폰트 굵기가 해석되지 않았다 (%d)", style.Weight)
	}
	if style.LineHeight <= 0 {
		return "", fmt.Errorf("줄높이가 해석되지 않았다 (%v)", style.LineHeight)
	}

	var b strings.Builder
	w := func(depth int, line string) {
		b.WriteString("\n" + shapeIndent + strings.Repeat("  ", depth) + line)
	}
	// 인셋 0: 웹이 잰 rect는 border box이고 `.balpyo-text`는 padding·border가 0이므로
	// 곧 콘텐츠 박스다 (render/web.contentCSS). OOXML 기본 인셋(좌우 91440, 상하 45720)을
	// 그대로 두면 좌표는 맞는데 글자만 몇 px 밀리고, 그 오차는 픽셀 비교에서 원인 불명으로
	// 나타난다. wrap="square"는 박스 폭 안에서 줄을 바꾸는 웹과 같은 결정이고, noAutofit은
	// 박스 고정 + 오버플로 허용이다 (PRD §13-1) — autofit 위임은 폰트 크기를 바꿔 웹과의
	// 시각 일치를 깬다. pre만 wrap="none"이다: 웹이 줄바꿈하지 않기로 했으므로
	// (render/web.writeBlock의 white-space: pre) PPTX가 접으면 줄 수가 갈라진다.
	wrap := "square"
	if elem.Name == "pre" {
		wrap = "none"
	}
	w(1, `<p:txBody>`)
	w(2, fmt.Sprintf(`<a:bodyPr wrap="%s" lIns="0" tIns="0" rIns="0" bIns="0" anchor="t">`, wrap))
	w(3, "<a:noAutofit/>")
	w(2, "</a:bodyPr>")
	w(2, "<a:lstStyle/>")
	w(2, "<a:p>")
	w(3, fmt.Sprintf(`<a:pPr algn="%s">`, algn))
	w(4, "<a:lnSpc>")
	// 절대 줄간격(spcPts)이다 — spcPct는 엔진마다 기준이 달라 글줄이 밀린다 (LineSpacing).
	w(5, fmt.Sprintf(`<a:spcPts val="%d"/>`, LineSpacing(style.LineHeight, style.Size)))
	w(4, "</a:lnSpc>")
	w(3, "</a:pPr>")
	// 빈 텍스트는 런 없는 문단으로 남는다 — 도형 자체는 낸다 (writeTextShape의 정책).
	if err := writeRuns(w, elem, style, theme, links, runContext{}); err != nil {
		return "", err
	}
	w(2, "</a:p>")
	w(1, "</p:txBody>")
	return b.String(), nil
}

// runContext는 순회가 잎까지 끌고 내려가는 런 상태다.
type runContext struct {
	// inline은 이 잎이 인라인 요소 안에 있는지다. 인라인 런만 i/u를 항상 명시한다 —
	// 웹의 textStyleCSS(블록 6선언)/runStyleCSS(런 8선언) 분리와 같은 결정이다: 지금
	// 어휘에서 블록 스타일은 Italic/Underline을 가질 수 없으므로, 블록 직속 런에 i="0"
	// u="none"을 더하면 Iteration 1 골든만 이유 없이 움직인다. 블록이 Italic을 가질 수
	// 있게 되는 날 이 구분이 함께 사라져야 한다 (runStyleCSS 주석과 같은 이월).
	inline bool
	// link는 가장 가까운 <a> 조상의 목적지다 (없으면 빈 문자열). Appearance.Link는 <a>
	// 자신에만 실리므로 (core.resolveInlines) 중첩 인라인의 잎까지는 순회가 나른다.
	link string
}

// writeRuns는 요소의 자식들을 텍스트 런 목록으로 쓴다 — 문자 데이터 잎마다 a:r 하나,
// `<br/>`마다 a:br 하나다. 순회는 웹 렌더러(writeBlock → writeInline)와 같은 문서 순서다 —
// 런 순서가 곧 글자 순서다.
//
// 인접한 같은 스타일 잎은 합치지 않는다: 파서가 연속 문자 데이터를 토큰 하나로 담으므로
// (soml lexConten의 "다음 `<`까지") 잎의 경계는 대개 요소이고, 주석이 문자 데이터를 쪼갠
// 경우에만 같은 스타일의 이웃 런이 생긴다 — rPr가 같은 이웃 런은 뷰어에서 구별되지 않는
// 유효한 XML이라 합치는 코드는 값어치가 없다.
func writeRuns(w func(depth int, line string), parent *soml.Element, style soml.TextStyle, theme soml.Theme, links *slideLinks, ctx runContext) error {
	for _, child := range parent.Children {
		switch node := child.(type) {
		case *soml.Text:
			if err := writeTextRuns(w, node.Value, style, theme, links, ctx); err != nil {
				return err
			}
		case *soml.Element:
			if node.Name == soml.BreakElement {
				// <br>은 void다 — 서식 없는 흐름의 분절점 (core가 Appearance를 채우지 않는
				// 유일한 인라인). 문단을 쪼개지 않는다: 문단 사이 간격은 줄간격과 다른
				// 축이라 웹의 <br>(같은 블록 안 줄바꿈)과 갈라질 수 있다.
				w(3, "<a:br/>")
				continue
			}
			if soml.KindOf(node.Name) != soml.KindInline {
				// 콘텐츠 모델상 블록 안에는 인라인뿐이다 — 여기 도달했다면 파서를 거치지
				// 않은 덱이므로 조용히 건너뛰지 않고 멈춘다 (웹 writeInlineChild와 같은 계약).
				return fmt.Errorf("%s 인라인 자리의 <%s>를 PPTX로 렌더하는 매핑이 없다", node.Span, node.Name)
			}
			childStyle := node.TextStyle()
			if childStyle == nil {
				return fmt.Errorf("%s <%s>의 외관이 해석되지 않았다 — core.Resolve를 거치지 않은 덱이다",
					node.Span, node.Name)
			}
			childCtx := runContext{inline: true, link: ctx.link}
			if node.Name == "a" {
				childCtx.link = node.Appearance.Link
				if childCtx.link == "" {
					// core의 href 검증(없음/빈 값 에러)을 지나지 않은 값이 산출물에 실릴
					// 길을 막는다 (render/web.writeInline과 같은 계약).
					return fmt.Errorf("%s <a>의 링크가 해석되지 않았다 — core.Resolve를 거치지 않은 덱이다",
						node.Span)
				}
			}
			if err := writeRuns(w, node, *childStyle, theme, links, childCtx); err != nil {
				return err
			}
		default:
			return fmt.Errorf("알 수 없는 노드 종류 %T", node)
		}
	}
	return nil
}

// writeTextRuns는 문자 데이터 잎 하나를 a:r로 쓴다.
//
// 보존된 원문 개행(<pre> — 나머지 모델은 파서가 접는다)은 a:br이 된다: 웹의 white-space:
// pre와 같은 결정이다 — 한쪽이 개행을 접으면 줄 수가 갈라진다. 문단을 줄마다 쪼개는 대안은
// 버렸다: 문단 사이 간격이 줄간격과 달라질 수 있고, pre는 한 문단의 여러 줄이다. 빈 줄은
// a:br만 남는다 — 빈 a:t는 유효하지만 정보가 없다.
func writeTextRuns(w func(depth int, line string), text string, style soml.TextStyle, theme soml.Theme, links *slideLinks, ctx runContext) error {
	fill, err := colorXML(style.Color, theme)
	if err != nil {
		return err
	}
	latin, ea, err := typefaces(style.Font, theme)
	if err != nil {
		return err
	}
	for i, line := range strings.Split(text, "\n") {
		if i > 0 {
			w(3, "<a:br/>")
		}
		// CRLF 원문의 \r도 개행의 일부다 — a:t에 남기면 뷰어마다 다르게 그린다.
		line = strings.TrimSuffix(line, "\r")
		if line == "" {
			continue
		}
		attrs := fmt.Sprintf(`sz="%d" b="%s"`, FontSize(style.Size), boldValue(style.Weight))
		if ctx.inline {
			// 인라인 런은 i/u까지 전부 명시한다 — hlinkClick의 자동 밑줄·자동 색 같은 뷰어
			// 기본값에 기대는 값은 뷰어가 바뀌면 움직이고, 그 움직임은 두 산출물을 나란히
			// 볼 때만 드러난다 (웹 runStyleCSS의 완전 명시와 같은 근거).
			attrs += fmt.Sprintf(` i="%s" u="%s"`, italicValue(style.Italic), underlineValue(style.Underline))
		}
		w(3, "<a:r>")
		w(4, fmt.Sprintf(`<a:rPr %s>`, attrs))
		w(5, "<a:solidFill>")
		w(6, fill)
		w(5, "</a:solidFill>")
		w(5, latin)
		w(5, ea)
		if ctx.link != "" {
			// hlinkClick은 latin/ea 뒤다 — CT_TextCharacterProperties의 자식 순서가 폰트
			// 요소들 뒤에 hlinkClick을 둔다 (어기면 LibreOffice가 복구를 묻는다).
			w(5, fmt.Sprintf(`<a:hlinkClick r:id="%s"/>`, links.relID(ctx.link)))
		}
		w(4, "</a:rPr>")
		w(4, fmt.Sprintf("<a:t>%s</a:t>", escapeXML(line)))
		w(3, "</a:r>")
	}
	return nil
}

// algnValue는 논리 정렬 이름을 OOXML `a:pPr@algn`으로 만든다.
//
// `l`/`r`은 물리 이름처럼 보이지만 OOXML에서는 문단 방향(rtl)에 따라 뜻이 뒤집히는
// **논리 값**이다 — start/end를 고른 이유와 정확히 짝이 맞는다 (soml.TextAlign의 주석).
func algnValue(align soml.TextAlign) (string, error) {
	switch align {
	case soml.AlignStart:
		return "l", nil
	case soml.AlignCenter:
		return "ctr", nil
	case soml.AlignEnd:
		return "r", nil
	case soml.AlignJustify:
		return "just", nil
	}
	return "", fmt.Errorf("정렬 %q는 논리 이름 어휘 밖이다", string(align))
}

// boldValue는 font-weight 수치를 rPr@b로 눕힌다.
func boldValue(weight int) string {
	if weight >= boldWeight {
		return "1"
	}
	return "0"
}

// italicValue는 확정 Italic을 rPr@i로 만든다 — 웹 font-style: italic/normal의 짝.
func italicValue(italic bool) string {
	if italic {
		return "1"
	}
	return "0"
}

// underlineValue는 확정 Underline을 rPr@u로 만든다 — 웹 text-decoration:
// underline/none의 짝. sng는 한 줄 밑줄이다 (OOXML u 어휘의 기본형).
func underlineValue(underline bool) string {
	if underline {
		return "sng"
	}
	return "none"
}

// colorXML은 해석된 색을 채움 자식 요소로 만든다.
//
// 토큰은 **schemeClr 참조로 남는다** — 여기서 RGB로 눕히면 내보낸 파일이 PowerPoint의
// 테마 색 편집을 따라가지 못한다 (soml.Color의 주석, 제품 핵심 계약). 참조 이름은 토큰
// 표의 Ref다 (clrMap을 거친 이름 — 파생 규칙을 여기 두지 않는다).
func colorXML(color soml.Color, theme soml.Theme) (string, error) {
	switch {
	case color.IsZero():
		return "", fmt.Errorf("색이 해석되지 않았다")
	case color.IsToken():
		token, ok := theme.Color(color.Token)
		if !ok {
			return "", fmt.Errorf("색 토큰 %q가 테마에 없다", color.Token)
		}
		if token.Ref == "" {
			// Ref 없는 토큰은 schemeClr를 만들 수 없다 — RGB로 눕히는 폴백은 토큰 보존
			// 계약을 조용히 깨므로 두지 않는다.
			return "", fmt.Errorf("색 토큰 %q에 schemeClr 참조(Ref)가 없다", color.Token)
		}
		return fmt.Sprintf(`<a:schemeClr val="%s"/>`, token.Ref), nil
	default:
		return fmt.Sprintf(`<a:srgbClr val="%s"/>`, color.RGB), nil
	}
}

// typefaces는 폰트 토큰을 rPr의 latin/ea 요소로 만든다.
//
// fontScheme 슬롯이 있는 토큰은 `+mj-lt` 같은 **테마 참조**로 남는다 — 색과 같은 이유로,
// 참조여야 내보낸 파일이 PowerPoint의 테마 폰트 변경을 따라간다. 슬롯 없는 토큰(mono)은
// 리터럴 typeface다: fontScheme에는 major/minor 두 자리뿐이라 세 번째 폰트를 실을 곳이
// 없다 (soml.FontToken.Scheme의 주석 — 테마 폰트 편집을 따라가지 못하는 유일한 예외).
func typefaces(role soml.FontRole, theme soml.Theme) (latin, ea string, err error) {
	if role == "" {
		return "", "", fmt.Errorf("폰트 토큰이 해석되지 않았다")
	}
	token, ok := theme.Font(role)
	if !ok {
		return "", "", fmt.Errorf("폰트 토큰 %q가 테마에 없다", string(role))
	}
	switch token.Scheme {
	case "major":
		return `<a:latin typeface="+mj-lt"/>`, `<a:ea typeface="+mj-ea"/>`, nil
	case "minor":
		return `<a:latin typeface="+mn-lt"/>`, `<a:ea typeface="+mn-ea"/>`, nil
	case "":
		return fmt.Sprintf(`<a:latin typeface="%s"/>`, escapeXML(token.Latin)),
			fmt.Sprintf(`<a:ea typeface="%s"/>`, escapeXML(token.EastAsian)), nil
	}
	return "", "", fmt.Errorf("폰트 토큰 %q의 fontScheme 슬롯 %q를 모른다 (major/minor/빈 문자열)",
		string(role), token.Scheme)
}
