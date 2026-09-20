package pptx

import (
	"fmt"
	"math"
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
// 간격을 넣어(스크립트 간격 — OOXML로 끌 수 없다) 혼합 문자 줄을 브라우저보다 넓게 그린다.
// 박스를 측정 폭 그대로 고정하면 여유 폭(Slack)이 그보다 작은 줄은 LibreOffice에서 한 줄
// 늘어난다 — 01_text 픽스처의 본문(여유 2.4%, 오차 +3.3%)이 정확히 그렇게 갈라졌다.
//
// **4%는 그 오차의 상한이 아니다.** 스크립트 간격은 경계 수에 비례하므로(경계당 +0.245 em)
// 짧은 줄에 문장부호가 몰리면 +14.5%까지 나온다 (P1.5의 측정 —
// docs/guides/known-differences.md). 여유로 덮는 길은 막혀 있다: 반대 방향의 위험(브라우저가
// 넘긴 단어가 넓힌 박스에 들어가 줄이 **줄어드는** 것) 때문에 여유는 단어 하나 폭보다 작아야
// 한다. 4%는 "단어 하나보다 작은 최대치"이고, 오차가 큰 줄이 접히지 않는 근거는 여유가 아니라
// 그 줄에 남은 여유 공간(좁은 트랙의 짧은 줄 — 40~46%)이다. 줄 수 갈라짐은 허용 오차 0의
// 줄 수 검사(TestFixtureLineBreaksMatchLibreOffice)가 따로 감시한다.
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

// shapesXML은 슬라이드 콘텐츠를 도형 목록으로 컴파일한다 — 이 함수가 지오메트리(①)와
// 해석된 외관(②)이 OOXML에서 만나는 자리다 (PRD §9.3).
//
// 문서 순서가 곧 spTree 순서다: OOXML은 나중 도형이 위에 그려지므로, 저자가 쓴 순서가
// 겹침 z-순서가 된다 (웹의 문서 순서 스택과 같은 규칙 — Iteration 6의 겹침이 이 위에 선다).
//
// 순회는 **재귀**다: 레이아웃 컨테이너는 도형이 되지 않고 자식으로 이어지므로 (writeShapeNode)
// 도형 수는 측정 박스 수와 같지 않다 — 깊이 우선 순회가 중첩에서도 문서 순서를 유지한다.
//
// 결과는 빈 문자열(요소 없음)이거나, 각 도형이 "\n"으로 시작해 개행 없이 끝나는 조각이다 —
// slideTemplate의 `{{.Shapes}}` 자리에 그대로 들어간다. 함께 돌려주는 rels는 순회가 배정한
// 관계 목록이다 — 슬라이드 rels 파트가 같은 배정을 실어야 `hlinkClick@r:id`와
// `blip@r:embed`가 살아난다 (buildParts).
func shapesXML(deck *soml.Deck, slide *soml.Slide, g *geom.Geometry) (string, *slideRels, error) {
	var b strings.Builder
	rels := &slideRels{}
	id := firstShapeID
	for i, node := range slide.Content {
		if err := writeShapeNode(&b, deck, slide, node, []int{i}, &id, g, rels); err != nil {
			return "", nil, err
		}
	}
	// 덱 크롬은 콘텐츠 뒤다 (ADR 0045) — 확정 모델이 비어 있으면 아무것도 나가지 않으므로
	// 크롬을 켜지 않은 덱의 바이트가 움직이지 않는다 (chrome.go).
	if err := writeChromeShapes(&b, deck, slide, &id, g); err != nil {
		return "", nil, err
	}
	return b.String(), rels, nil
}

// slideRels는 슬라이드 하나가 배정한 관계 목록이다 — 외부 하이퍼링크(`hlinkClick`)와
// 미디어(`blip@r:embed`)가 **한 카운터**를 나눠 쓴다.
//
// rel id는 결정적이어야 한다 — 같은 덱을 두 번 조립하면 같은 바이트가 나와야 골든 비교가
// 성립한다. 배정은 순회의 문서 순서를 따르고, **같은 목적지는 관계 하나를 재사용한다**:
// OPC 관계의 정체는 (Type, Target, TargetMode)이라 같은 목적지의 관계 둘은 정보가 없고,
// PowerPoint도 저장하며 하나로 합치므로 매번 내면 재저장 diff에 잡음만 남는다 — 같은 그림을
// 두 번 쓴 슬라이드의 두 `p:pic`이 같은 `r:embed`를 가리키는 것이 그 결과다.
//
// **번호를 한 카운터로 둔 것이 이 타입의 내용 전부다.** 종류마다 번호 구간을 주려면
// (예: 그림은 `하이퍼링크 수 + 3`부터) 첫 그림을 쓰는 순간 그 슬라이드의 **총 하이퍼링크
// 수**를 이미 알아야 하고, 그러려면 슬라이드를 두 번 걸어야 한다 — 두 순회는 어긋날 수 있고
// 어긋난 순간이 파일을 열 때까지 조용하다 (그림이 사라지거나 링크가 그림을 가리킨다). 한
// 카운터로 문서 순서대로 배정하면 그 어긋남을 만들 방법 자체가 없다.
type slideRels struct {
	items []slideRel
}

// slideRel은 슬라이드 rels 파트에 실리는 관계 하나다 — 템플릿이 이 값을 그대로 쓴다
// (Type이 데이터인 이유: 한 목록에 두 종류가 섞이므로 템플릿이 종류를 알면 안 된다).
type slideRel struct {
	ID     string
	Type   string
	Target string
	// TargetMode는 패키지 **밖** 대상에만 있다 (하이퍼링크). 미디어는 패키지 안의 파트라
	// 빈 값이고, 템플릿이 그때 속성을 아예 내지 않는다 — 내부 파트에
	// `TargetMode="External"`이 붙으면 소비자가 파트 경로를 URL로 읽어 그림이 조용히 사라진다.
	TargetMode string
}

// slideRelBase는 첫 배정 관계의 번호다. rId1은 레이아웃, rId2는 노트 슬라이드 자리로
// **노트가 없어도 예약**한다 — 노트 유무가 뒤 번호를 밀면 노트 한 줄을 더한 커밋의 diff에
// 관계 전부가 잡음으로 실린다. OPC 관계 id는 임의 문자열이라 빈 번호는 유효하다
// (notesSlidePart의 번호 규칙과 같은 태도).
const slideRelBase = 3

// relID는 목적지의 관계 id다 — 처음 보는 목적지면 다음 번호를 배정한다.
func (r *slideRels) relID(relType, target, targetMode string) string {
	for _, item := range r.items {
		if item.Type == relType && item.Target == target {
			return item.ID
		}
	}
	id := fmt.Sprintf("rId%d", slideRelBase+len(r.items))
	r.items = append(r.items, slideRel{ID: id, Type: relType, Target: target, TargetMode: targetMode})
	return id
}

// linkID는 외부 하이퍼링크의 관계 id다.
func (r *slideRels) linkID(url string) string {
	return r.relID(hyperlinkRelType, url, externalTargetMode)
}

// mediaID는 미디어 파트의 관계 id다 (target은 슬라이드 파트 기준 상대 경로 — mediaTarget).
func (r *slideRels) mediaID(target string) string {
	return r.relID(imageRelType, target, "")
}

// shapeWriter는 **도형 하나가 되는** 종류를 쓰는 함수다. 네 함수가 같은 서명을 갖는 것이
// 아래 표가 성립하는 근거다 — 도형이 되는 종류는 전부 (요소, id, 도형 번호, 지오메트리,
// 관계)만으로 자기 몫을 쓴다.
type shapeWriter func(b *strings.Builder, deck *soml.Deck, slide *soml.Slide, elem *soml.Element, id string, shapeID int, g *geom.Geometry, rels *slideRels) error

// renderedKinds는 이 렌더러가 다루는 요소 종류의 **명시 목록**이다 (soml 카탈로그와 대조하는
// 짝은 render 패키지의 테스트다 — TASKS P0.2: `<table>`이 파싱·검증을 통과한 뒤 렌더에서
// 빌드를 죽인 사고).
//
// 목록에 **어디서 다루는지**를 함께 적는 이유: 이 렌더러의 디스패치는 세 자리로 갈라져 있다.
// 도형이 되는 종류는 shapeWriterFor가, 부모 도형에 흡수되는 종류는 그 부모가 쓴다 — "다룬다"의
// 뜻이 종류마다 다르므로 목록만 보고 도형 수를 세면 틀린다 (writeShapeNode 주석의 대비).
var renderedKinds = []soml.ElementKind{
	soml.KindBlock,           // 도형 하나       (writeTextShape — `<text>`도 같은 자리다)
	soml.KindListContainer,   // 도형 하나       (writeListShape: 불릿 문단이 한 박스에 산다)
	soml.KindShape,           // 도형 하나+흡수  (writeShapeSp)
	soml.KindImage,           // 도형 하나       (writePicShape — p:sp가 아니라 p:pic)
	soml.KindTable,           // 도형 하나       (writeGraphicFrameTbl — p:sp도 p:pic도 아닌 p:graphicFrame)
	soml.KindTableRow,        // 도형 0개        (a:tr: 표 프레임 안에 접힌다 — 잰 높이만 쓰인다)
	soml.KindTableCell,       // 도형 0개        (a:tc: 같다 — 잰 좌우 경계가 열 너비가 된다)
	soml.KindListItem,        // 도형 0개        (writeListItem: 리스트 도형의 문단이 된다)
	soml.KindInline,          // 도형 0개        (writeRuns: 문단의 런이 된다)
	soml.KindLayoutContainer, // 도형 0개        (자식으로 내려간다 — 좌표만 남는다)
	soml.KindSpacer,          // 도형 0개        (공간은 이미 이웃 도형의 좌표에 들어갔다)
}

// RenderedKinds는 이 렌더러가 다루는 종류다 (선언 순서). soml 카탈로그와 대조하는 테스트가
// 쓴다 — 렌더러 바깥에서 "무엇을 그릴 수 있는가"를 물을 수 있는 유일한 창구다
// (render/web.RenderedKinds와 같은 계약).
func RenderedKinds() []soml.ElementKind {
	return append([]soml.ElementKind(nil), renderedKinds...)
}

// shapeWriterFor는 도형 하나가 되는 종류의 writer다. 흡수되거나 사라지는 종류는 nil이다 —
// nil이 곧 "여기서 도형 번호를 쓰지 않는다"는 뜻이고, 그 판정을 한 곳에 모은 것이
// writeShapeNode에서 번호가 뛰지 않는 근거다.
func shapeWriterFor(kind soml.ElementKind) shapeWriter {
	switch kind {
	case soml.KindBlock:
		return writeTextShape
	case soml.KindListContainer:
		return writeListShape
	case soml.KindShape:
		return writeShapeSp
	case soml.KindImage:
		return writePicShape
	case soml.KindTable:
		// 표는 도형 하나를 소비한다 — 행·셀은 이 프레임 안에 접히므로 자기 번호를 갖지 않는다
		// (`<shape>`가 자식 텍스트를 흡수하는 것과 같은 모양 — render/pptx/table.go).
		return writeGraphicFrameTbl
	}
	return nil
}

// writeShapeNode는 콘텐츠 노드 하나를 도형으로 쓴다 — 자기 도형을 갖지 않는 노드는 자식으로
// 내려가거나(레이아웃 컨테이너) 아무것도 내지 않는다(`<spacer/>`).
//
// path 규칙은 웹 렌더러와 같다 (web.ChildPath → web.ElementID — 지오메트리 조회의 열쇠).
// 자식의 좌표는 지오메트리 곁 표에서 **그대로** 온다: 컨테이너 안이라는 사실이 좌표에 영향을
// 주지 않는다 (잰 rect는 이미 슬라이드 상대다 — geom 패키지 주석). 컨테이너 상대 좌표를
// 다시 더하는 코드가 여기 없는 것이 "좌표 계산 0회"의 렌더러 쪽 절반이다.
//
// # 분기는 여섯 가지다 (같은 단어를 쓰는 개념들이 정반대로 매핑되는 자리)
//
//	리스트 컨테이너 (<ul>/<ol>)             → 도형 **하나**  (writeListShape: 불릿 문단이 한 텍스트 박스에 산다)
//	<shape>                                 → 도형 **하나** + 자식을 **흡수** (writeShapeSp: 자식 텍스트가 이 도형의 txBody 문단이 된다)
//	<image/>                                → 도형 **하나**  (writePicShape: p:sp가 아니라 **p:pic**이다 — 자식은 없다)
//	<table>                                 → 도형 **하나** + 자식을 **흡수** (writeGraphicFrameTbl: 행·셀이 a:tr/a:tc로 접힌다 — p:graphicFrame이다)
//	레이아웃 컨테이너 (<row>/<col>/<grid>)  → 도형 **0개** + 자식으로 **내려가기** (좌표만 남고 컨테이너는 사라진다)
//	<spacer/>                               → 도형 **0개**  (자식도 없다 — void)
//
// 리스트는 하나의 텍스트 흐름이라 도형 하나가 편집 단위이고(항목마다 도형을 내면 편집 단위가
// 조각난다), 레이아웃은 배치 **수단**이라 좌표로 구워진 뒤 산출물에 남을 것이 없다 —
// 저자가 좌표를 한 번도 계산하지 않고 만든 그리드가 PowerPoint에서 절대 좌표 도형들로
// 나오는 것이 제품의 약속이다 (PRD §9.3, TASKS Iteration 3). 같은 단어를 쓰는 두 개념이
// 정반대로 매핑되므로 테스트가 같은 슬라이드에서 이 대비를 단정한다 — 태그를 늘리는 사람이
// 가장 헷갈릴 자리다.
//
// **`<shape>`는 컨테이너와 정반대로 처리된다**: 컨테이너는 사라지고 자식이 도형이 되는데,
// 도형은 남고 자식이 사라진다 (자식 텍스트가 부모의 `txBody`가 되므로 여기서 **자식으로
// 내려가지 않는다**). 그래서 도형 안 텍스트는 도형 id를 소비하지 않고, 잰 rect도 소비하지
// 않는다 — `<li>`와 같은 처지다 (재지만 도형 좌표가 아니다 — render/web 패키지 주석의
// 측정 대상 계약).
//
// `<group>` → `grpSp`(Iteration 9)와도 갈라진다: 그룹은 편집 단위로 묶이는 도형이고 레이아웃
// 컨테이너는 사라지는 배치 수단이다 (PRD §7.5). 배치 태그가 `grpSp`가 되면 `<group>`과
// `<row>`의 구분이 산출물에서 사라진다. **도형이 자식을 흡수하는 것도 `grpSp`가 아니다** —
// 그룹은 여러 도형을 묶는 것이고, 여기서는 텍스트가 도형 하나의 본문이 된다.
func writeShapeNode(b *strings.Builder, deck *soml.Deck, slide *soml.Slide, node soml.Node, path []int, shapeID *int, g *geom.Geometry, rels *slideRels) error {
	elem, ok := node.(*soml.Element)
	if !ok {
		// 슬라이드 콘텐츠와 레이아웃 컨테이너는 요소만 담는다 (파서의 modelElements) —
		// 여기 도달했다면 파서를 거치지 않은 덱이므로 조용히 건너뛰지 않고 멈춘다.
		return fmt.Errorf("%s: %s 자리에 요소가 아닌 노드 %T가 있다",
			slide.Source, web.ElementID(slide.Index, path), node)
	}
	// 텍스트 박스가 되는 것은 <text>·블록 종류 카탈로그·리스트 컨테이너다 (soml.KindOf —
	// 렌더러가 이름 목록을 따로 들지 않는 이유는 웹과 같다).
	id := web.ElementID(slide.Index, path)
	kind := soml.KindOf(elem.Name)
	if elem.Name == soml.TextElement {
		// `<text>`는 카탈로그 밖(KindNone)이지만 블록과 **같은** 텍스트 박스가 된다 —
		// PRD §7.7의 등가 표(`<p>` ≡ `<text role="body">`)가 산출물에서도 등가라는 뜻이다.
		kind = soml.KindBlock
	}
	if write := shapeWriterFor(kind); write != nil {
		// 도형 하나가 되는 종류는 전부 이 한 자리를 지난다 (`<shape>`는 자식을 흡수하므로
		// 자식으로 내려가지 않고, 그림은 p:sp가 아니라 p:pic이지만 — 도형 번호를 하나
		// 쓴다는 점은 같다: `p:sp`와 `p:pic`의 id가 겹치면 PowerPoint가 파일을 복구한다).
		if err := write(b, deck, slide, elem, id, *shapeID, g, rels); err != nil {
			return err
		}
		// 도형 id는 **도형을 실제로 낸 자리에서만** 오른다 (컨테이너는 id를 소비하지
		// 않는다 — 도형 id는 도형에만 붙는다): 번호가 배치 구조에 따라 뛰면 슬라이드에
		// `<row>` 하나를 더한 커밋의 diff에 도형 번호 전부가 잡음으로 실린다 — 노트가
		// 없어도 rId2를 예약한 것과 같은 태도다 (slideRelBase).
		*shapeID++
		if soml.ShapeHoldsLayout(elem) {
			// 배경 모드: 도형을 낸 **뒤에** 자식으로 내려간다 (ADR 0049). 위에서 낸 `p:sp`가
			// 배경이고 — shapeBodyXML이 빈 문단만 냈다 — 자식은 자기 잰 좌표를 가진 **형제**
			// 도형이 된다. `p:grpSp`가 아닌 근거는 좌표 공간이다: 그룹은 `chOff`/`chExt`로
			// 둘째 좌표계를 만들어 "잰 rect가 곧 xfrm"을 깬다 (PRD §9.3).
			//
			// 순서가 이 방향인 것이 z-순서다 — 배경이 먼저 나가야 글자 아래에 칠해진다
			// (웹의 문서 순서 규칙과 같은 결과: 그림이 흐름 뒤에 있다).
			return writeChildShapes(b, deck, slide, elem, path, shapeID, g, rels)
		}
		return nil
	}
	switch kind {
	case soml.KindLayoutContainer:
		// 컨테이너 자신은 도형이 되지 않고 자식으로 내려간다. 빈 컨테이너(`<row></row>`)는
		// 자식이 없으니 도형 0개다 — 대비되는 선례는 빈 `<text>`가 도형을 내는 것이고
		// (writeTextShape), 차이는 `<text>`가 저자의 콘텐츠라는 것이다.
		//
		if err := writeChildShapes(b, deck, slide, elem, path, shapeID, g, rels); err != nil {
			return err
		}
	case soml.KindSpacer:
		// `<spacer/>`는 도형을 내지 않는다. 빈 박스이고 뜻은 "여기에 공간이 있다"이며, 그
		// 공간은 이미 이웃 도형의 좌표에 들어가 있다 — 밀어내기의 결과가 형제의 rect다.
		// 도형을 내면 PowerPoint의 선택 창에 뜻 없는 빈 텍스트 박스가 남는다 (빈 `<text>`와
		// 갈라지는 이유: `<text>`는 저자의 콘텐츠이고 spacer는 배치 수단이다).
	default:
		// 웹 렌더러와 같은 이유로 멈춘다 (render/web.writeNode): 어휘에 태그를 추가하고 렌더
		// 매핑을 잊으면 "파싱은 되는데 그려지지 않는" 조용한 실패가 된다.
		return fmt.Errorf("%s:%s <%s>를 PPTX로 렌더하는 매핑이 없다", slide.Source, elem.Span, elem.Name)
	}
	return nil
}

// writeChildShapes는 자식들을 차례로 도형으로 쓴다 — **자식이 형제 도형이 되는 두 자리**가
// 공유한다: 사라지는 레이아웃 컨테이너와, 남아서 배경이 되는 도형(ADR 0049).
//
// 깊이 우선 순회이므로 **문서 순서 = spTree 순서 = z-순서**가 중첩에서도 이어진다
// (shapesXML의 계약, 웹의 문서 순서 스택과 같은 규칙 — Iteration 6의 겹침이 이 순서 위에
// 선다). 좌표를 더하는 코드가 여기 없는 것이 "좌표 계산 0회"의 절반이다: 부모가 컨테이너든
// 배경 도형이든 자식의 잰 rect는 이미 슬라이드 상대다.
//
// 빈 부모는 도형 0개다 (`<row></row>`·`<shape><col/></shape>`) — 배경 도형은 자기 `p:sp`를
// 이미 냈으므로 그 경우에도 슬라이드에서 사라지지 않는다.
func writeChildShapes(b *strings.Builder, deck *soml.Deck, slide *soml.Slide, elem *soml.Element, path []int, shapeID *int, g *geom.Geometry, rels *slideRels) error {
	for i, child := range elem.Children {
		if err := writeShapeNode(b, deck, slide, child, web.ChildPath(path, i), shapeID, g, rels); err != nil {
			return err
		}
	}
	return nil
}

// writeTextShape는 텍스트 역할 블록 하나(`<text>`, 서브셋 블록)를 p:sp(텍스트 박스)로 쓴다.
//
// 빈 텍스트·크기 0 박스도 도형을 낸다. 건너뛰면 spTree 순서(=z-순서)와 도형 id가 내용에
// 따라 흔들리고, 저자는 빈 요소가 산출물 어디에 있는지 PowerPoint에서 확인할 수 없다 —
// 크기 0 도형은 OOXML에서 유효하므로 (Extent가 이 판단을 호출자에게 남긴 이유) 그대로 낸다.
func writeTextShape(b *strings.Builder, deck *soml.Deck, slide *soml.Slide, elem *soml.Element, id string, shapeID int, g *geom.Geometry, rels *slideRels) error {
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
	body, err := textBodyXML(elem, *style, deck.Theme, rels)
	if err != nil {
		return fmt.Errorf("%s:%s <%s>: %w", slide.Source, elem.Span, elem.Name, err)
	}
	spec := textBoxSp(shapeID, elem.Name+" "+id, TextBoxRect(box.Rect, style.Align), body)
	spec.ph = slotPlaceholder(slide, elem)
	return writeSp(b, spec, deck.Theme)
}

// writeListShape는 리스트 컨테이너 하나(`<ul>`/`<ol>`)를 p:sp(텍스트 박스) **하나**로 쓴다 —
// `<li>` 하나가 문단(a:p) 하나다. 항목마다 도형을 내면 PowerPoint에서 리스트가 낱개 텍스트
// 박스로 조각나 편집 단위가 깨진다 — 리스트는 하나의 텍스트 흐름이다.
//
// 좌표는 컨테이너의 잰 rect다 (항목 05가 ul/ol에 식별자를 붙였다 — li rect는 측정되지만
// 도형 좌표로 쓰지 않는다). 안전 여유는 블록과 같은 TextBoxRect를 **start 정렬로** 적용한다:
// li는 align 속성을 받지 않아 (파서 allowlist) 확정 정렬이 언제나 역할 기본값 start이므로,
// 글자가 닻을 내린 왼쪽 변(=마커 위치)이 움직이지 않는 방향은 오른쪽 확장뿐이다. 여유 비율의
// 근거(스크립트 간격)는 텍스트 폭 기준인데 리스트 폭에는 마커 자리(ListTextIndent)가
// 포함되어 절대 여유가 텍스트만일 때보다 약간 크다 — 줄 늘어남(LibreOffice에서 한 줄
// 증가)을 막는 안전한 방향의 오차이므로 그대로 두고, 실제 판정은 항목 08의 픽셀 비교가 한다.
func writeListShape(b *strings.Builder, deck *soml.Deck, slide *soml.Slide, elem *soml.Element, id string, shapeID int, g *geom.Geometry, rels *slideRels) error {
	box, ok := g.Box(id)
	if !ok {
		return fmt.Errorf("%s: %s <%s>의 측정된 좌표가 없다", slide.Source, id, elem.Name)
	}
	body, err := listBodyXML(elem, deck.Theme, rels)
	if err != nil {
		return fmt.Errorf("%s:%s <%s>: %w", slide.Source, elem.Span, elem.Name, err)
	}
	spec := textBoxSp(shapeID, elem.Name+" "+id, TextBoxRect(box.Rect, soml.AlignStart), body)
	spec.ph = slotPlaceholder(slide, elem)
	return writeSp(b, spec, deck.Theme)
}

// writeShapeSp는 `<shape>` 하나를 p:sp(프리셋 도형)로 쓴다 — **자식 텍스트를 자기 txBody로
// 흡수한다** (writeShapeNode의 네 분기).
//
// 좌표는 도형의 잰 rect **그대로**다. 안전 여유(TextBoxRect)를 적용하지 않는 것이 텍스트
// 박스와 갈라지는 자리이고, 근거는 여유의 뜻이다: 그것은 "브라우저가 잰 글자 폭보다
// LibreOffice가 넓게 잴 수 있다"를 흡수하는 장치인데(textBoxSlackRatio), 도형의 박스는
// 글자를 잰 결과가 아니라 **저자가 정한 크기**다 (`w`/`h`/`ratio` — soml.ShapeAttrs의 근거).
// 넓히면 그림이 커져 배치가 어긋나고 글자는 도형 밖으로 나간다. 도형 안 글자의 줄바꿈 여유는
// 대신 `pad`를 뺀 안쪽과 글자 폭의 차이가 소유한다 — 구조적으로 더 크지만 보장은 아니다
// (NOTES의 측정).
func writeShapeSp(b *strings.Builder, deck *soml.Deck, slide *soml.Slide, elem *soml.Element, id string, shapeID int, g *geom.Geometry, rels *slideRels) error {
	box, ok := g.Box(id)
	if !ok {
		return fmt.Errorf("%s: %s <%s>의 측정된 좌표가 없다", slide.Source, id, elem.Name)
	}
	style := elem.ShapeStyle()
	if style == nil {
		// 해석되지 않은 외관을 기본값으로 그리면 PPTX만 그럴듯하게 나오고 웹과 갈라진다
		// (render/web.writeShape와 같은 관문).
		return fmt.Errorf("%s:%s <%s>의 도형 외관이 해석되지 않았다 — core.Resolve를 거치지 않은 덱이다",
			slide.Source, elem.Span, elem.Name)
	}
	spec, err := shapeSp(shapeID, elem.Name+" "+id, box.Rect, *style)
	if err != nil {
		return fmt.Errorf("%s:%s <%s>: %w", slide.Source, elem.Span, elem.Name, err)
	}
	body, err := shapeBodyXML(elem, *style, box.Rect, deck.Theme, rels)
	if err != nil {
		return fmt.Errorf("%s:%s <%s>: %w", slide.Source, elem.Span, elem.Name, err)
	}
	spec.body = body
	spec.ph = slotPlaceholder(slide, elem)
	if err := writeSp(b, spec, deck.Theme); err != nil {
		return fmt.Errorf("%s:%s <%s>: %w", slide.Source, elem.Span, elem.Name, err)
	}
	return nil
}

// spSpec은 p:sp 하나의 조립 명세다 — **텍스트 박스와 프리셋 도형이 같은 함수를 지난다**
// (writeSp).
//
// 조립된 XML 조각이 아니라 값으로 받는 이유: 골격을 함수 하나에 두어야 스키마 순서
// (CT_ShapeProperties의 xfrm → 지오메트리 → 채움 → ln → effectLst)가 한 곳에서만 지켜진다.
// 조각을 받으면 호출자마다 순서를 다시 알아야 하고, 어긴 순서는 LibreOffice의 복구 경고로만
// 드러난다.
type spSpec struct {
	id   int
	name string
	// rect는 도형 좌표다 — 텍스트 박스는 안전 여유를 더한 값, 도형은 잰 값 그대로다
	// (writeShapeSp의 근거).
	rect geom.Rect
	// textBox는 `p:cNvSpPr@txBox`를 켠다. **도형에는 켜지 않는다**: PowerPoint가 텍스트
	// 박스와 도형에 다른 서식 창을 주고, 도형이 텍스트 박스로 표시되면 "모양 변경"이
	// 사라진다 — "내보낸 도형이 진짜 프리셋 도형인가"라는 리뷰 항목의 절반이 이 한 속성이다.
	textBox bool
	// xfrm는 회전·대칭이다 — **그림과 같은 코드**를 쓴다 (transformOf).
	xfrm
	// preset은 `prstGeom@prst`이고 adjust는 `avLst`의 조정값이다 (빈 목록이면 `<a:avLst/>`).
	preset string
	adjust []spAdjust
	// fill·border는 확정 외관이다. 텍스트 박스는 둘 다 `none`을 준다 — 명시하지 않으면
	// 테마 fmtScheme의 기본 채움/선이 상속되어 웹에 없는 사각형이 그려진다.
	fill   soml.Fill
	border soml.Border
	// shadow는 그림자 토큰이다 (nil이면 effectLst 없음).
	shadow *soml.ShadowToken
	// alpha는 채움·선·그림자 색에 실을 불투명도다 (nil이면 알파 없음).
	//
	// **OOXML에는 도형 단위 불투명도가 없다** — 알파는 색 안에 산다. 웹도 그림(svg)에만
	// `opacity`를 걸어 자식 텍스트를 건드리지 않으므로(render/web.shapeInkCSS), 이쪽도
	// 채움·선·그림자 색에만 싣는다. **텍스트 런 색에 같은 알파를 싣는 길은 버렸다**: 웹이
	// 글자를 흐리게 하지 않으므로 그렇게 하면 두 타깃이 갈라진다 (도형 전체 투명도인
	// `a:alphaModFix`를 쓰지 않는 이유도 같다 — 그쪽은 글자까지 흐려진다).
	alpha *float64
	// body는 조립된 p:txBody다 (빈 문자열이면 본문 없음).
	body string
	// ph는 이 도형이 채우는 슬롯 플레이스홀더다 (nil이면 `<p:nvPr/>` — 슬롯 아님).
	ph *placeholder
}

// spAdjust는 `avLst`의 조정값 하나다 — 잰 박스로 평가가 끝난 값이다 (AdjustValue).
type spAdjust struct {
	name string
	val  int64
}

// placeholder는 이 도형이 채우는 슬롯 플레이스홀더다 (`<p:nvPr>`의 `<p:ph>`, ADR 0035).
//
// 슬롯을 쓴 슬라이드 직계 자식만 갖는다 (soml.Appearance.Slot이 거기에만 실린다 — core.resolveSlot).
// **명시 xfrm/rPr은 그대로 둔다** (ADR 0035 선택지 3): `<p:ph>`는 PowerPoint가 이 도형을 진짜
// 플레이스홀더로 잡아 개요 보기·서식 초기화·"클릭해 입력"에서 특별 취급하게 하는 표식일 뿐이고,
// 좌표·서식은 잰 박스와 확정 rPr이 소유한다 (웹·PPTX 픽셀 일치 계약). 그래서 슬라이드 도형은
// 플레이스홀더이면서도 자기 xfrm을 갖는다 — 레이아웃 플레이스홀더의 위치를 상속만 받으면 잰
// 박스와 갈라진다.
type placeholder struct {
	// typ는 `p:ph@type`이다 (soml.Slot.PhType — ctrTitle·title·subTitle·body·pic).
	typ string
	// idx는 `p:ph@idx`다 — **0이면 속성을 생략한다** (제목류의 관례, soml.Slot.PhIndex).
	idx int
}

// xml은 `<p:ph>` 요소다. idx 0은 속성을 생략한다 (제목 플레이스홀더의 관례 — soml.Slot.PhIndex).
func (p placeholder) xml() string {
	if p.idx == 0 {
		return fmt.Sprintf(`<p:ph type="%s"/>`, p.typ)
	}
	return fmt.Sprintf(`<p:ph type="%s" idx="%d"/>`, p.typ, p.idx)
}

// slotPlaceholder는 요소가 채우는 슬롯의 플레이스홀더 매핑이다 — 슬롯이 없으면 nil.
//
// 매핑(슬롯 이름 → phType/idx)은 soml이 소유한다 (LayoutSlot — 닫힌 카탈로그, 렌더러가
// 재구현하지 않는다). 어휘 밖 슬롯은 core가 이미 빌드 에러로 막았으므로(resolveSlot) 여기서
// 조회가 실패하면 조용히 nil을 준다 — 이 자리는 해석된 덱만 지난다.
func slotPlaceholder(slide *soml.Slide, elem *soml.Element) *placeholder {
	if elem.Appearance == nil || elem.Appearance.Slot == "" {
		return nil
	}
	slot, ok := soml.LayoutSlot(slide.Layout, elem.Appearance.Slot)
	if !ok {
		return nil
	}
	return &placeholder{typ: slot.PhType, idx: slot.PhIndex}
}

// writeNvPr는 `<p:nvPr>`를 쓴다 — 슬롯 플레이스홀더가 있으면 `<p:ph>`를 자식으로 담고, 없으면
// self-closing이다 (`<p:nvPr/>`). `p:sp`(도형)와 `p:pic`(그림)이 같은 함수를 지나 두 곳의
// 플레이스홀더 방출이 갈라지지 않는다.
func writeNvPr(w func(depth int, line string), depth int, ph *placeholder) {
	if ph == nil {
		w(depth, "<p:nvPr/>")
		return
	}
	w(depth, "<p:nvPr>")
	w(depth+1, ph.xml())
	w(depth, "</p:nvPr>")
}

// rectPreset은 텍스트 박스의 프리셋 이름이다 — 도형 어휘의 `rect`와 같은 값이지만
// (web.ShapePreset("rect")) 텍스트 박스는 도형 어휘를 거치지 않는다.
const rectPreset = "rect"

// textBoxSp는 텍스트 박스 도형의 명세다 — 도형과 갈라지는 값이 여기 한곳에 모인다
// (`txBox="1"`, `rect` 프리셋, 채움·선 없음, 변형·효과 없음).
func textBoxSp(id int, name string, rect geom.Rect, body string) spSpec {
	return spSpec{
		id: id, name: name, rect: rect, textBox: true,
		preset: rectPreset,
		fill:   soml.NoFill(), border: soml.NoBorder(),
		body: body,
	}
}

// shapeSp는 확정 도형 외관을 p:sp 명세로 컴파일한다 (PRD §9.4 표의 오른쪽 열).
//
// **순서는 PRD 표 순서다** (fill → border → radius → shadow → opacity → rotate → flip) —
// 웹의 shapeInkCSS와 나란히 읽을 수 있어야 "표에 있고 한쪽에만 있는 행"이 눈에 띈다.
// `radius`는 여기서 **기하**가 된다 (`avLst` 조정값): 그것이 OOXML에서 반경이 사는 유일한
// 자리이고, 웹에서 그것이 SVG `rx`인 것과 짝이다.
func shapeSp(id int, name string, rect geom.Rect, style soml.ShapeStyle) (spSpec, error) {
	preset, ok := web.ShapePreset(style.Type)
	if !ok {
		// 어휘에 프리셋을 더하고 매핑을 잊으면 조용히 사각형이 되는 대신 멈춘다 — `prst`
		// 이름이 틀려도 파일은 열리므로(뷰어가 모르는 프리셋을 무시한다) 이 관문과 골든
		// 어서션만이 그것을 잡는다.
		return spSpec{}, fmt.Errorf("도형 type %q의 prstGeom 매핑이 없다", style.Type)
	}
	spec := spSpec{id: id, name: name, rect: rect, preset: preset}
	if style.Fill == nil {
		return spSpec{}, fmt.Errorf("채움이 해석되지 않았다")
	}
	spec.fill = *style.Fill
	if style.Border == nil {
		return spSpec{}, fmt.Errorf("테두리가 해석되지 않았다")
	}
	spec.border = *style.Border
	adjust, err := shapeAdjusts(style, rect)
	if err != nil {
		return spSpec{}, err
	}
	spec.adjust = adjust
	spec.shadow = style.Shadow
	if style.Opacity != nil {
		if *style.Opacity < 0 || *style.Opacity > 1 {
			// 웹과 같은 관문이다 (shapeInkCSS) — 범위 밖 알파는 뷰어가 조용히 자른다.
			return spSpec{}, fmt.Errorf("불투명도 %v가 0~1 밖이다", *style.Opacity)
		}
		spec.alpha = style.Opacity
	}
	transform, err := transformOf(style.Rotate, style.Flip)
	if err != nil {
		return spSpec{}, err
	}
	spec.xfrm = transform
	return spec, nil
}

// shapeAdjusts는 이 도형의 `avLst` 조정값이다 — 명세는 웹의 기하 표가 소유하고(web.ShapeAdjusts)
// 평가는 잰 박스로 여기서 한다 (AdjustValue).
//
// `radius`가 프리셋과 어긋나면 에러다 (웹의 writeShapeInk와 같은 관문): rounded-rect에 반경이
// 없으면 두 타깃의 암묵 기본값이 다르고(CSS 0 ↔ roundRect 16.67%), 다른 프리셋에 반경이
// 남아 있으면 실을 자리가 없어 **웹만 둥글어진다**. core가 이미 막지만 렌더러가 마지막
// 관문이다. 어느 프리셋이 반경을 받는지는 표가 말한다 — 이름 비교를 두 번째로 적지 않는다.
func shapeAdjusts(style soml.ShapeStyle, rect geom.Rect) ([]spAdjust, error) {
	specs, ok := web.ShapeAdjusts(style.Type)
	if !ok {
		return nil, fmt.Errorf("도형 type %q의 조정값 명세가 없다", style.Type)
	}
	needsRadius := false
	for _, spec := range specs {
		if spec.Axis == web.AdjustAxisRadius {
			needsRadius = true
		}
	}
	var radius float64
	switch {
	case needsRadius && style.Radius == nil:
		return nil, fmt.Errorf("type=%q인데 반경이 해석되지 않았다", style.Type)
	case !needsRadius && style.Radius != nil:
		return nil, fmt.Errorf("type=%q에는 반경을 실을 자리가 없다", style.Type)
	case needsRadius:
		radius = style.Radius.Px
	}
	adjusts := make([]spAdjust, 0, len(specs))
	for _, spec := range specs {
		val, err := AdjustValue(spec, rect, radius)
		if err != nil {
			return nil, err
		}
		adjusts = append(adjusts, spAdjust{name: spec.Name, val: val})
	}
	return adjusts, nil
}

// writeSp는 도형 하나의 골격을 쓴다 — nvSpPr(id·이름·txBox 표시)와 spPr(좌표·변형·프리셋
// 지오메트리·채움·선·효과) 사이에 조립된 txBody를 끼운다.
//
// 자식 순서는 CT_ShapeProperties가 정한 그대로다: xfrm → 지오메트리 → 채움 → ln →
// effectLst. 어기면 LibreOffice가 복구를 묻는다 (rPr의 hlinkClick 자리와 같은 종류의 제약).
func writeSp(b *strings.Builder, spec spSpec, theme soml.Theme) error {
	x, y := Offset(spec.rect)
	cx, cy := Extent(spec.rect)

	w := func(depth int, line string) {
		b.WriteString("\n" + shapeIndent + strings.Repeat("  ", depth) + line)
	}
	w(0, "<p:sp>")
	w(1, "<p:nvSpPr>")
	// 이름은 지오메트리 식별자다 — PowerPoint 선택 창과 진단 메시지가 같은 이름으로
	// 같은 요소를 가리킨다. 저자의 `id`(soml.ShapeStyle.ID)는 아직 산출물에 나가지 않는다:
	// 그것은 `<arrow from="db">`의 참조 대상 이름이고(Iteration 7), 웹도 쓰지 않는다.
	w(2, fmt.Sprintf(`<p:cNvPr id="%d" name="%s"/>`, spec.id, escapeXML(spec.name)))
	if spec.textBox {
		w(2, `<p:cNvSpPr txBox="1"/>`)
	} else {
		w(2, "<p:cNvSpPr/>")
	}
	writeNvPr(w, 2, spec.ph)
	w(1, "</p:nvSpPr>")
	w(1, "<p:spPr>")
	// 회전·대칭은 xfrm의 **속성**이고 좌표를 건드리지 않는다: 회전 중심이 `ext`의 중심이고
	// 잰 rect는 회전 **전** 박스이므로 보정할 값이 없다 (Rotation의 주석). flipH/flipV는
	// 기하만 뒤집고 글자는 읽히는 방향으로 남는다 — 웹이 대칭을 그림에만 걸어 글자를 거울로
	// 만들지 않는 것과 같은 그림이다 (render/web.shapeInkCSS).
	w(2, "<a:xfrm"+spec.attrs()+">")
	w(3, fmt.Sprintf(`<a:off x="%d" y="%d"/>`, x, y))
	w(3, fmt.Sprintf(`<a:ext cx="%d" cy="%d"/>`, cx, cy))
	w(2, "</a:xfrm>")
	writeGeom(w, 2, spec.preset, spec.adjust)
	if err := writeFill(w, 2, spec.fill, spec.alpha, theme); err != nil {
		return err
	}
	if err := writeLine(w, 2, spec.border, spec.alpha, theme); err != nil {
		return err
	}
	if err := writeEffect(w, 2, spec.shadow, spec.alpha, theme); err != nil {
		return err
	}
	w(1, "</p:spPr>")
	b.WriteString(spec.body)
	w(0, "</p:sp>")
	return nil
}

// writeGeom은 프리셋 지오메트리와 조정값을 쓴다 — **도형과 그림이 같은 함수를 지난다**
// (그림의 프리셋은 `rect` 또는 `roundRect` 둘뿐이지만 조립은 같은 모양이다 — image.go).
func writeGeom(w func(depth int, line string), depth int, preset string, adjust []spAdjust) {
	w(depth, fmt.Sprintf(`<a:prstGeom prst="%s">`, preset))
	if len(adjust) == 0 {
		w(depth+1, "<a:avLst/>")
	} else {
		w(depth+1, "<a:avLst>")
		for _, item := range adjust {
			// 명시하지 않은 조정값은 프리셋 기본값이 산다 — avLst는 전체 목록이 아니라
			// 덮어쓰기다 (rightArrow가 adj2만 싣는 근거 — web.ShapeAdjusts).
			w(depth+2, fmt.Sprintf(`<a:gd name="%s" fmla="val %d"/>`, item.name, item.val))
		}
		w(depth+1, "</a:avLst>")
	}
	w(depth, "</a:prstGeom>")
}

// xfrm는 `a:xfrm`의 변형(회전·대칭)이다 — 도형(p:sp)과 그림(p:pic)이 **같은 값**을 쓴다.
//
// 두 요소가 같은 CT_Transform2D를 갖고 두 타깃의 회전 규약도 하나이므로(Rotation), 조립을
// 나누면 같은 저자 표기가 두 요소에서 다른 속성이 될 길이 생긴다 — `rotate="15"`가 도형에는
// 실리고 그림에는 빠지는 종류의 실패는 두 산출물을 나란히 볼 때까지 보이지 않는다.
type xfrm struct {
	// rot은 `xfrm@rot`이다 (nil이면 속성 없음 — 회전을 정하지 않은 도형·그림과 텍스트 박스).
	rot          *int64
	flipH, flipV bool
}

// transformOf는 확정 외관의 회전·대칭을 xfrm으로 컴파일한다 (도형·그림 공용).
func transformOf(rotate *float64, flip soml.Flip) (xfrm, error) {
	var t xfrm
	if rotate != nil {
		rot := Rotation(*rotate)
		t.rot = &rot
	}
	switch flip {
	case "":
	case soml.FlipH:
		t.flipH = true
	case soml.FlipV:
		t.flipV = true
	case soml.FlipBoth:
		t.flipH, t.flipV = true, true
	default:
		return xfrm{}, fmt.Errorf("flip %q는 대칭 어휘 밖이다", string(flip))
	}
	return t, nil
}

// attrs는 `a:xfrm`의 변형 속성이다 — 없으면 빈 문자열이다 (텍스트 박스의 xfrm에는 속성이
// 없고, 그래서 텍스트 박스 산출은 이 함수가 생기기 전과 바이트가 같다).
func (t xfrm) attrs() string {
	var attrs string
	if t.rot != nil {
		attrs += fmt.Sprintf(` rot="%d"`, *t.rot)
	}
	// flipH/flipV는 rot 뒤다 (CT_Transform2D의 속성 순서 — 속성 순서는 스키마 유효성과
	// 무관하지만 골든이 한 가지 표기로 남는다).
	if t.flipH {
		attrs += ` flipH="1"`
	}
	if t.flipV {
		attrs += ` flipV="1"`
	}
	return attrs
}

// writeFill은 확정 채움을 spPr의 채움 요소로 쓴다 — `none`은 `a:noFill`, 색은 `a:solidFill`이다.
func writeFill(w func(depth int, line string), depth int, fill soml.Fill, alpha *float64, theme soml.Theme) error {
	if fill.None {
		// 웹의 `fill: none`(그리지 않는다)과 정확히 짝이다 — 투명한 채움이 아니라 채움 없음이다.
		w(depth, "<a:noFill/>")
		return nil
	}
	if fill.Color.IsGradient() {
		// 그라디언트 채움은 solidFill이 아니라 gradFill이다 (PRD §9.4). 도형 불투명도는
		// 스톱 색마다 알파로 실린다 (writeGradFill).
		return writeGradFill(w, depth, *fill.Color.Gradient, alpha, theme)
	}
	w(depth, "<a:solidFill>")
	if err := writeColor(w, depth+1, fill.Color, alpha, theme); err != nil {
		return err
	}
	w(depth, "</a:solidFill>")
	return nil
}

// writeLine은 확정 테두리를 `a:ln`으로 쓴다 — 굵기는 속성(EMU), 색은 solidFill 자식, 스타일은
// prstDash 자식이다 (조각이 서로 다른 자리에 실리는 것이 soml.Border를 3조각으로 푼 이유다).
//
// **`a:ln`은 도형 외곽선 중앙에 그려진다** — 웹도 SVG `stroke`로 경로 중앙에 그리므로 굵기의
// 절반이 잰 박스 밖으로 **두 타깃 모두** 번진다 (render/web/shape.go의 테두리 절). 그래서
// 도형 rect를 굵기 절반만큼 줄이는 보정을 하지 않는다: 보정하면 PPTX만 작아진다.
func writeLine(w func(depth int, line string), depth int, border soml.Border, alpha *float64, theme soml.Theme) error {
	return writeLineElem(w, depth, "a:ln", border, alpha, theme)
}

// writeLineElem은 writeLine의 요소 이름을 고른 판이다 — 표 셀의 테두리는 같은 선 어휘를
// 변마다 다른 이름으로 싣는다 (`a:lnL`/`a:lnR`/`a:lnT`/`a:lnB` — render/pptx/table.go).
//
// 이름만 다르고 내용이 같아야 하는 것이 요점이다: 셀 테두리를 따로 조립하면 굵기 환산·그라디언트·
// 대시 어휘가 두 벌이 되고, 갈라진 순간 도형 테두리와 셀 테두리가 같은 `border` 값에서 다르게
// 나간다.
func writeLineElem(w func(depth int, line string), depth int, name string, border soml.Border, alpha *float64, theme soml.Theme) error {
	if border.None {
		// 선 없음도 **명시한다** — 비워 두면 테마 fmtScheme의 기본 선이 상속되어 웹에 없는
		// 외곽선이 그려진다 (textBoxSp의 근거와 같다).
		w(depth, "<"+name+">")
		w(depth+1, "<a:noFill/>")
		w(depth, "</"+name+">")
		return nil
	}
	if border.Width <= 0 {
		return fmt.Errorf("테두리 굵기가 해석되지 않았다 (%v)", border.Width)
	}
	dash, err := prstDashValue(border.Style)
	if err != nil {
		return err
	}
	w(depth, fmt.Sprintf(`<%s w="%d">`, name, EMU(border.Width)))
	if border.Color.IsGradient() {
		// 그라디언트 테두리는 `ln` 안의 `gradFill`이다 (PRD §9.4의 `border=… gradient(…)` 행 —
		// PPTX는 `ln` + `gradFill`). 웹은 SVG stroke가 gradient def를 참조한다 (render/web).
		if err := writeGradFill(w, depth+1, *border.Color.Gradient, alpha, theme); err != nil {
			return err
		}
	} else {
		w(depth+1, "<a:solidFill>")
		if err := writeColor(w, depth+2, border.Color, alpha, theme); err != nil {
			return err
		}
		w(depth+1, "</a:solidFill>")
	}
	// 스키마 순서(CT_LineProperties): 채움 → prstDash → 조인·화살촉.
	w(depth+1, fmt.Sprintf(`<a:prstDash val="%s"/>`, dash))
	w(depth, "</"+name+">")
	return nil
}

// writeEffect는 그림자 토큰을 `effectLst > outerShdw`로 쓴다 (그림자가 없으면 아무것도 쓰지
// 않는다 — soml.ShapeStyle의 "정하지 않음" 계약).
//
// **`fmtScheme/effectStyleLst` 참조로 내는 길을 버렸다.** 그 목록은 스타일이 세 개뿐이고
// 도형이 `style/effectRef`로 하나를 고르는 구조라, 도형마다 다른 그림자를 실을 수 없다 —
// 그림자 토큰이 셋을 넘는 순간 매핑이 무너지고, 무너지는 방식은 "어떤 도형의 그림자가 다른
// 토큰의 값으로 그려진다"다 (soml.ShadowToken이 남긴 결정을 여기서 닫는다). 도형마다 값을
// 굽는 대가는 테마에서 그림자를 편집할 수 없다는 것이고, 그것은 반경·공간과 같은 종류의
// 손실이다 (OOXML에 그 테마 개념이 없다).
//
// 극좌표를 그대로 낸다 — `outerShdw@dist/dir`가 극좌표이므로 변환이 없다. 직교로 푸는 쪽은
// 웹이다 (render/web.shadowVarsCSS): 손실 있는 방향(atan2 → 60000분의 1도 반올림)을 산출
// 경계에 두지 않는다는 결정의 결과가 이 비대칭이다.
//
// `rotWithShape="1"`인 이유: 웹에서 회전은 **박스**에 걸리고 그림자는 그 안 svg의
// `filter`이므로 그림자가 도형과 함께 돈다 (render/web/shape.go의 박스/그림 분담). 0으로 두면
// 45도 회전 도형의 그림자만 두 타깃에서 다른 방향으로 진다.
func writeEffect(w func(depth int, line string), depth int, shadow *soml.ShadowToken, alpha *float64, theme soml.Theme) error {
	if shadow == nil {
		return nil
	}
	w(depth, "<a:effectLst>")
	w(depth+1, fmt.Sprintf(`<a:outerShdw blurRad="%d" dist="%d" dir="%d" rotWithShape="1">`,
		EMU(shadow.Blur), EMU(shadow.Dist), Rotation(shadow.Dir)))
	if err := writeColor(w, depth+2, shadow.Color, shadowAlpha(*shadow, alpha), theme); err != nil {
		return err
	}
	w(depth+1, "</a:outerShdw>")
	w(depth, "</a:effectLst>")
	return nil
}

// shadowAlpha는 그림자 색의 실효 불투명도다 — 토큰의 알파에 도형 불투명도를 곱한다
// (1 이상이면 nil: 알파 요소를 내지 않는다).
//
// 곱하는 근거는 웹의 합성 순서다: `opacity`가 svg에 걸리고 그림자는 그 svg의 `filter`
// 결과이므로 그림자도 함께 흐려진다 (render/web.shapeInkCSS). 곱하지 않으면 반투명 도형의
// 그림자만 PPTX에서 진하게 남는다.
func shadowAlpha(shadow soml.ShadowToken, opacity *float64) *float64 {
	alpha := shadow.Alpha
	if opacity != nil {
		alpha *= *opacity
	}
	if alpha >= 1 {
		return nil
	}
	return &alpha
}

// prstDashValue는 테두리 스타일을 OOXML `a:prstDash@val`로 만든다.
//
// **계열 선택이 이 함수의 내용 전부다** (soml.BorderStyle이 렌더 매핑에 넘긴 결정): `sys*`
// 계열이 아니라 `dash`(4:3)·`dot`(1:3)을 고른다. 근거는 웹이 이미 그 비율을 냈다는 것이다 —
// `stroke-dasharray`의 수치가 ECMA-376의 프리셋 파선 표에서 왔고(render/web.dashArray),
// `sysDash`(3:1)·`sysDot`(1:1)을 고르면 같은 저자 값이 두 타깃에서 다른 간격으로 그려진다.
// 이름이 아니라 **수치**가 두 타깃을 잇는 것이 파선의 계약이다.
func prstDashValue(style soml.BorderStyle) (string, error) {
	switch style {
	case soml.BorderSolid:
		// 실선도 명시한다 — 비워 두면 뷰어·테마의 기본값에 기대게 된다 (선 없음을 명시하는
		// 것과 같은 근거).
		return "solid", nil
	case soml.BorderDashed:
		return "dash", nil
	case soml.BorderDotted:
		return "dot", nil
	}
	return "", fmt.Errorf("테두리 스타일 %q는 어휘 밖이다", string(style))
}

// textBodyXML은 해석된 텍스트 외관과 블록의 자식들을 p:txBody로 컴파일한다 (PRD §9.4의
// `font`/`size`/`weight`/`color`/`align` 행과 "인라인 서식 → 텍스트 런 rPr/hlinkClick" 행).
// 문단 서식은 블록의 확정 스타일에서, 런 서식은 잎을 감싼 가장 안쪽 요소의 확정 스타일에서
// 나온다 — 요소의 Attrs를 다시 읽지 않는다 (render/web.textStyleCSS와 같은 계약).
func textBodyXML(elem *soml.Element, style soml.TextStyle, theme soml.Theme, rels *slideRels) (string, error) {
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
	//
	// anchor="t"는 슬라이드 직속 텍스트 박스의 고정값이다 — 잰 rect가 곧 내용 크기라 세로
	// 앵커가 움직일 여백이 없다 (soml.TextStyle.Anchor). 앵커가 뜻을 갖는 자리는 도형
	// 안이고, 그쪽 bodyPr은 shapeBodyXML이 낸다.
	wrap := "square"
	if elem.Name == "pre" {
		wrap = "none"
	}
	w(1, `<p:txBody>`)
	w(2, fmt.Sprintf(`<a:bodyPr wrap="%s" lIns="0" tIns="0" rIns="0" bIns="0" anchor="t">`, wrap))
	w(3, "<a:noAutofit/>")
	w(2, "</a:bodyPr>")
	w(2, "<a:lstStyle/>")
	// 빈 텍스트는 런 없는 문단으로 남는다 — 도형 자체는 낸다 (writeTextShape의 정책).
	if err := writeTextParagraph(w, elem, style, theme, rels); err != nil {
		return "", err
	}
	w(1, "</p:txBody>")
	return b.String(), nil
}

// writeTextParagraph는 텍스트 요소 하나를 마커 없는 문단(a:p)으로 쓴다 — 블록 하나가 문단
// 하나이고, **도형 안 텍스트도 같은 함수를 지난다** (shapeBodyXML).
//
// 한 함수인 이유: 도형 안 텍스트가 슬라이드 직속 텍스트와 다른 문단 서식을 받으면, 같은
// `<p>`가 도형 안에서만 다르게 그려진다 — 부모가 도형이라는 사실은 텍스트 해석을 바꾸지
// 않는다는 값 층의 결정(core/shape.go)의 렌더러 쪽 절반이다.
func writeTextParagraph(w func(depth int, line string), elem *soml.Element, style soml.TextStyle, theme soml.Theme, rels *slideRels) error {
	algn, err := algnValue(style.Align)
	if err != nil {
		return err
	}
	// 블록 자체의 색·폰트·메트릭도 여기서 먼저 검증한다 — 런이 없는 빈 요소에서도 해석되지
	// 않은 외관은 소리를 내야 한다 (writeTextShape의 빈 도형 정책과 짝).
	if err := checkTextStyle(style, theme); err != nil {
		return err
	}
	w(2, "<a:p>")
	w(3, fmt.Sprintf(`<a:pPr algn="%s">`, algn))
	w(4, "<a:lnSpc>")
	// 절대 줄간격(spcPts)이다 — spcPct는 엔진마다 기준이 달라 글줄이 밀린다 (LineSpacing).
	w(5, fmt.Sprintf(`<a:spcPts val="%d"/>`, LineSpacing(style.LineHeight, style.Size)))
	w(4, "</a:lnSpc>")
	w(3, "</a:pPr>")
	if err := writeRuns(w, elem, style, theme, rels, runContext{}); err != nil {
		return err
	}
	w(2, "</a:p>")
	return nil
}

// shapeBodyXML은 도형의 자식 텍스트를 그 도형의 p:txBody로 컴파일한다 — **자식 하나가 문단
// (a:p) 하나**다 (리스트가 `<li>` 하나를 문단 하나로 내는 것과 같은 구조 — listBodyXML).
//
// 도형 `sp` 하나와 텍스트 `sp` 하나로 나누는 대안을 버린 근거는 **편집 단위**다: PowerPoint
// 에서 도형을 끌면 글자가 따라와야 하고, 두 도형으로 내면 저자가 카드를 옮길 때마다 글자를
// 따로 옮긴다 (리스트를 도형 하나로 낸 것과 같은 근거). 그래서 도형 안 텍스트는 자기 도형도
// 도형 id도 갖지 않고, 잰 rect도 소비되지 않는다 (writeShapeNode).
//
// bodyPr이 텍스트 박스와 갈라지는 자리는 셋이다:
//   - **인셋은 저자의 `pad`**다 (텍스트 박스는 0). 웹에서 `pad`가 도형의 padding이고
//     `box-sizing: border-box`라 잰 박스를 키우지 않으므로(render/web.shapeBoxCSS), 같은 값이
//     같은 자리에 들어가는 것이 두 타깃의 글자 위치를 맞춘다.
//   - **앵커는 첫 자식이 정한다** (web.ShapeAnchor — 규칙을 웹과 공유해야 두 타깃의 앵커가
//     갈라지지 않는다). 정한 자식이 없으면 `t`이고, 그것이 flex의 기본 분배와 같은 뜻이다.
//   - wrap은 **언제나 square**다. `<pre>` 자식이 있어도 바꾸지 않는다: bodyPr은 본문 하나에
//     하나뿐이라 wrap="none"은 형제 문단까지 줄바꿈하지 않게 만들고, 그러면 그려진 도형 밖으로
//     글자가 나가는 것이 **pre가 아닌 자식에서** 일어난다. 대가는 도형 안 `<pre>`의 긴 줄이
//     웹에서는 넘치고 PPTX에서는 접히는 것이고, 그 갈라짐은 그 자식 하나에 갇힌다.
func shapeBodyXML(elem *soml.Element, style soml.ShapeStyle, rect geom.Rect, theme soml.Theme, rels *slideRels) (string, error) {
	anchor, err := bodyAnchorValue(web.ShapeAnchor(elem))
	if err != nil {
		return "", err
	}
	insets, err := bodyInsets(style.Pad, style.Type, rect)
	if err != nil {
		return "", err
	}

	var b strings.Builder
	w := func(depth int, line string) {
		b.WriteString("\n" + shapeIndent + strings.Repeat("  ", depth) + line)
	}
	w(1, `<p:txBody>`)
	w(2, fmt.Sprintf(`<a:bodyPr wrap="square" %s anchor="%s">`, insets, anchor))
	w(3, "<a:noAutofit/>")
	w(2, "</a:bodyPr>")
	w(2, "<a:lstStyle/>")
	if len(elem.Children) == 0 || soml.ShapeHoldsLayout(elem) {
		// 자식 없는 도형(`<shape/>`)도 txBody를 갖는다 — CT_TextBody가 문단 하나 이상을
		// 요구하므로(p+) 서식 없는 빈 문단이 스키마를 채운다 (li 없는 리스트와 같은 처리).
		// 본문을 아예 내지 않는 길도 있었지만(txBody는 sp의 선택 자식이다) 그러면 저자가
		// PowerPoint에서 그 도형에 글자를 넣을 때 서식이 뷰어 기본값에서 온다.
		//
		// **배경 모드가 같은 길을 지난다** (ADR 0049 결정 5): 자식이 형제 도형을 가졌으므로
		// 이 본문에 실을 글자가 없고, 남은 것은 자식 없는 도형과 정확히 같은 상황이다 —
		// 빈 문단 + `pad` 인셋 + 앵커. 인셋을 함께 남기는 것이 뜻이 있다: 저자가 PowerPoint에서
		// 배경 도형에 글자를 더하면 카드의 패딩 안쪽에서 시작한다.
		w(2, "<a:p/>")
	} else {
		for _, child := range elem.Children {
			text, ok := child.(*soml.Element)
			if !ok {
				// 도형 직속 문자 데이터는 파서가 막는다 (modelElements) — 여기 도달했다면
				// 파서를 거치지 않은 덱이므로 조용히 건너뛰지 않고 멈춘다.
				return "", fmt.Errorf("<%s> 안에 요소가 아닌 노드 %T가 있다", elem.Name, child)
			}
			if text.Name != soml.TextElement && soml.KindOf(text.Name) != soml.KindBlock {
				// 흡수 모드에 남은 자식은 텍스트 요소뿐이다 (파서의 children allowlist —
				// soml.shapeChildren이 여는 나머지는 컨테이너 넷이고, 그것이 하나라도 있으면
				// 위 분기가 이 도형을 배경으로 돌린다). 여기 도달한 태그는 어휘와 렌더 매핑이
				// 어긋난 것이다.
				return "", fmt.Errorf("%s <%s> 안의 <%s>를 PPTX로 렌더하는 매핑이 없다",
					text.Span, elem.Name, text.Name)
			}
			textStyle := text.TextStyle()
			if textStyle == nil {
				return "", fmt.Errorf("%s <%s>의 외관이 해석되지 않았다 — core.Resolve를 거치지 않은 덱이다",
					text.Span, text.Name)
			}
			if err := writeTextParagraph(w, text, *textStyle, theme, rels); err != nil {
				return "", err
			}
		}
	}
	w(1, "</p:txBody>")
	return b.String(), nil
}

// bodyAnchorValue는 세로 앵커를 `bodyPr@anchor`로 만든다 (빈 앵커는 위 = OOXML 기본값).
//
// 물리 이름(top/bottom)이 OOXML에서도 물리 이름인 것은 세로축에 방향 전환이 없기 때문이다 —
// `algn`이 논리 값(l/r)인 것과 갈라지는 지점이고, soml.TextAnchor가 물리 이름을 고른 근거와
// 같다. 웹은 같은 값을 flex 주축 분배로 낸다 (render/web.anchorValue).
func bodyAnchorValue(anchor soml.TextAnchor) (string, error) {
	switch anchor {
	case "", soml.AnchorTop:
		return "t", nil
	case soml.AnchorCenter:
		return "ctr", nil
	case soml.AnchorBottom:
		return "b", nil
	}
	return "", fmt.Errorf("anchor %q는 세로 앵커 어휘 밖이다", string(anchor))
}

// bodyInsets는 도형의 `pad`를 `bodyPr`의 인셋 속성 넷으로 만든다 (`pad`가 없으면 전부 0).
//
// 백분율 `pad`는 에러다. CSS 백분율 padding은 **부모의 폭**에 대한 비율인데(자기 폭이 아니다)
// OOXML 인셋은 절대 EMU이므로, 잰 도형 rect만으로는 웹이 실제로 쓴 값을 복원할 수 없다.
// 근사해서 내면 도형 안 글자 위치가 조용히 어긋나므로 — 두 타깃이 다른 값을 쓰는 것을
// 렌더러가 아는 순간 멈추는 편이 낫다 (해석되지 않은 값을 에러로 두는 것과 같은 태도).
//
// 프리셋이 들고 있는 rect가 우리가 쓰는 텍스트 영역과 다르면 그 몫을 **뺀다** (ADR 0041,
// web.ShapeTextCancel — chevron 하나다): 인셋은 뷰어가 프리셋의 rect **안쪽으로** 더 들어가는
// 값이므로, 음수 인셋이 그 rect를 우리 영역까지 되돌린다. 값이 잰 박스에 달렸기 때문에
// (백분율) 이 함수가 rect를 받는다 — 조정값 보정(AdjustValue)과 같은 구조다.
func bodyInsets(pad *soml.Pad, shapeType string, rect geom.Rect) (string, error) {
	insets := [4]int64{}
	if pad != nil {
		for i, side := range [4]soml.Length{pad.Left, pad.Top, pad.Right, pad.Bottom} {
			if side.Percent {
				return "", fmt.Errorf("도형의 pad는 백분율을 쓸 수 없다 (%v%%) — OOXML bodyPr 인셋은 절대 EMU이고 CSS 백분율 padding은 부모 폭 기준이다",
					side.Value)
			}
			insets[i] = EMU(side.Value)
		}
	}
	cancel, ok := web.ShapeTextCancel(shapeType)
	if !ok {
		// 어휘에 있는데 표에 없는 프리셋이다 — 조용히 0으로 두면 그 도형의 글자만 뷰어의
		// rect에 갇힌다 (shapeGeometryOf를 에러로 둔 것과 같은 근거).
		return "", fmt.Errorf("도형 type %q의 텍스트 영역 보정이 없다", shapeType)
	}
	if cancel.HasPercent() {
		left, top, right, bottom := cancel.Px(rect.W, rect.H, 0)
		for i, px := range [4]float64{left, top, right, bottom} {
			insets[i] -= EMU(px)
		}
	}
	return fmt.Sprintf(`lIns="%d" tIns="%d" rIns="%d" bIns="%d"`,
		insets[0], insets[1], insets[2], insets[3]), nil
}

// checkTextStyle은 확정 텍스트 스타일이 OOXML로 컴파일될 수 있는지 검증한다 — 웹 렌더러의
// textStyleCSS와 같은 검사다. 해석되지 않은 0 값이 sz="0"으로 흘러가면 파일은 열리지만
// 글자가 보이지 않는다.
func checkTextStyle(style soml.TextStyle, theme soml.Theme) error {
	if err := validateColor(style.Color, theme); err != nil {
		return err
	}
	if _, _, err := typefaces(style.Font, theme); err != nil {
		return err
	}
	if style.Size <= 0 {
		return fmt.Errorf("폰트 크기가 해석되지 않았다 (%v)", style.Size)
	}
	if style.Weight <= 0 {
		return fmt.Errorf("폰트 굵기가 해석되지 않았다 (%d)", style.Weight)
	}
	if style.LineHeight <= 0 {
		return fmt.Errorf("줄높이가 해석되지 않았다 (%v)", style.LineHeight)
	}
	return nil
}

// listBodyXML은 리스트 컨테이너의 자식들을 p:txBody로 컴파일한다 — `<li>` 하나가 문단
// (a:p) 하나다 (PRD §9.4의 "리스트 → buChar/들여쓰기" 행). 문단 서식은 li의 확정
// 스타일에서, 들여쓰기·마커는 웹 리스트 CSS 상수(render/web — 항목 05)에서 나온다.
func listBodyXML(list *soml.Element, theme soml.Theme, rels *slideRels) (string, error) {
	var b strings.Builder
	w := func(depth int, line string) {
		b.WriteString("\n" + shapeIndent + strings.Repeat("  ", depth) + line)
	}
	// 인셋 0·wrap="square"·noAutofit의 근거는 블록과 같다 (textBodyXML): `.balpyo-list`도
	// padding·border 0의 border box라 잰 rect가 곧 콘텐츠 박스이고, li의 들여쓰기는 인셋이
	// 아니라 문단의 marL이 소유한다 (웹의 li padding-left와 짝).
	w(1, `<p:txBody>`)
	w(2, `<a:bodyPr wrap="square" lIns="0" tIns="0" rIns="0" bIns="0" anchor="t">`)
	w(3, "<a:noAutofit/>")
	w(2, "</a:bodyPr>")
	w(2, "<a:lstStyle/>")
	if len(list.Children) == 0 {
		// <li> 없는 리스트는 파서가 막지 않는다 — 도형은 낸다 (빈 텍스트와 같은 정책:
		// spTree 순서와 도형 id가 내용에 따라 흔들리지 않는다). txBody는 문단 하나
		// 이상을 요구하므로(CT_TextBody의 p+) 서식 없는 빈 문단이 스키마를 채운다 —
		// li가 없으니 실을 확정 스타일도 마커도 없다.
		w(2, "<a:p/>")
	}
	for _, child := range list.Children {
		li, ok := child.(*soml.Element)
		if !ok {
			// 리스트 컨테이너 직속 문자 데이터는 파서가 막는다 (modelElements) — 여기
			// 도달했다면 파서를 거치지 않은 덱이므로 조용히 건너뛰지 않고 멈춘다.
			return "", fmt.Errorf("<%s> 안에 요소가 아닌 노드 %T가 있다", list.Name, child)
		}
		if soml.KindOf(li.Name) != soml.KindListItem {
			// 콘텐츠 모델상 리스트 컨테이너의 자식은 <li>뿐이다 (파서의 children allowlist).
			return "", fmt.Errorf("%s <%s> 안의 <%s>를 PPTX로 렌더하는 매핑이 없다",
				li.Span, list.Name, li.Name)
		}
		if err := writeListItem(w, list, li, theme, rels); err != nil {
			return "", err
		}
	}
	w(1, "</p:txBody>")
	return b.String(), nil
}

// writeListItem은 `<li>` 하나를 마커 있는 문단(a:p)으로 쓴다.
//
// pPr의 들여쓰기는 행잉 인덴트다: marL은 텍스트 시작(웹 li의 padding-left =
// web.ListTextIndent), indent는 음수로 마커를 내민다(-web.ListMarkerHang). 값은 웹
// 리스트 CSS 상수의 EMU 환산이다 — 두 렌더러에 따로 적히면 들여쓰기가 갈라진다 (환산은
// units.go의 경계 함수 EMU만 쓴다). 항목 사이 추가 간격(spcBef/spcAft)은 내지 않는다 —
// 웹이 ListItemGap=0으로 정했다 (수직 리듬은 줄높이 소유). 상수가 0이 아니게 되는 날
// spcAft 대응을 함께 구현해야 한다 (lists_test의 상수 단정이 그 날을 잡는다).
//
// 마커 폰트(buFont)는 li의 확정 폰트다 — 명시하지 않으면 뷰어 기본 폰트가 마커를 그려
// 크기·모양이 흔들리고, 웹의 마커(li::before)는 li의 폰트를 상속하므로 짝이 맞으려면
// 같은 폰트여야 한다. 색·크기는 명시하지 않는다: OOXML 기본(buClrTx/buSzTx)이 이미
// "텍스트를 따른다"이고, 그것이 정확히 ::before의 상속과 같은 뜻이다.
func writeListItem(w func(depth int, line string), list, li *soml.Element, theme soml.Theme, rels *slideRels) error {
	style := li.TextStyle()
	if style == nil {
		return fmt.Errorf("%s <%s>의 외관이 해석되지 않았다 — core.Resolve를 거치지 않은 덱이다",
			li.Span, li.Name)
	}
	algn, err := algnValue(style.Align)
	if err != nil {
		return err
	}
	if err := checkTextStyle(*style, theme); err != nil {
		return err
	}
	buFont, err := bulletFontXML(style.Font, theme)
	if err != nil {
		return err
	}
	bullet, err := bulletXML(list.Name)
	if err != nil {
		return err
	}
	w(2, "<a:p>")
	w(3, fmt.Sprintf(`<a:pPr marL="%d" indent="%d" algn="%s">`,
		EMU(web.ListTextIndent), -EMU(web.ListMarkerHang), algn))
	w(4, "<a:lnSpc>")
	// 절대 줄간격(spcPts)이다 — 블록과 같은 규칙 (textBodyXML의 근거).
	w(5, fmt.Sprintf(`<a:spcPts val="%d"/>`, LineSpacing(style.LineHeight, style.Size)))
	w(4, "</a:lnSpc>")
	// 스키마 순서(CT_TextParagraphProperties): lnSpc → buFont → buChar/buAutoNum.
	w(4, buFont)
	w(4, bullet)
	w(3, "</a:pPr>")
	// li 안의 인라인은 블록과 같은 런 분할기를 지난다 (항목 06) — <br/>은 같은 문단 안의
	// a:br(마커 없는 이어짐 줄)이고, 빈 li는 런 없는 문단으로 남는다 (자리와 순서 보존).
	if err := writeRuns(w, li, *style, theme, rels, runContext{}); err != nil {
		return err
	}
	w(2, "</a:p>")
	return nil
}

// bulletXML은 리스트 컨테이너 이름의 마커 요소다 — 웹 마커와 같은 곳에서 나온다:
// ul은 li::before의 글리프(web.ListBulletChar), ol은 엔진 카운터의 "1." 형식과 짝인
// arabicPeriod다 (render/web.listCSS).
func bulletXML(container string) (string, error) {
	switch container {
	case "ul":
		return fmt.Sprintf(`<a:buChar char="%s"/>`, escapeXML(web.ListBulletChar)), nil
	case "ol":
		return `<a:buAutoNum type="arabicPeriod"/>`, nil
	}
	// 컨테이너 어휘(soml.KindListContainer)에 이름을 추가하고 마커 매핑을 잊으면
	// 조용히 마커 없는 리스트가 되는 대신 멈춘다.
	return "", fmt.Errorf("<%s>의 마커 매핑이 없다", container)
}

// bulletFontXML은 마커 폰트 요소다 — li의 확정 폰트 토큰에서 나온다 (근거는 writeListItem).
// buFont는 typeface 하나뿐이라 latin 슬롯을 쓴다: 마커 글리프(•·숫자)는 라틴 폰트가 그린다.
func bulletFontXML(role soml.FontRole, theme soml.Theme) (string, error) {
	latin, _, err := typefaceValues(role, theme)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf(`<a:buFont typeface="%s"/>`, escapeXML(latin)), nil
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
func writeRuns(w func(depth int, line string), parent *soml.Element, style soml.TextStyle, theme soml.Theme, rels *slideRels, ctx runContext) error {
	for _, child := range parent.Children {
		switch node := child.(type) {
		case *soml.Text:
			if err := writeTextRuns(w, node.Value, style, theme, rels, ctx); err != nil {
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
			if err := writeRuns(w, node, *childStyle, theme, rels, childCtx); err != nil {
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
func writeTextRuns(w func(depth int, line string), text string, style soml.TextStyle, theme soml.Theme, rels *slideRels, ctx runContext) error {
	if err := validateColor(style.Color, theme); err != nil {
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
		// 텍스트 색은 rPr의 채움이다 — 그라디언트는 `gradFill`, 단색·변형은 `solidFill`이다
		// (PRD §9.4의 텍스트 `color=gradient` 행 — PPTX는 rPr + gradFill). 웹은
		// background-clip:text로 같은 결과를 낸다 (render/web).
		if err := writeTextFill(w, 5, style.Color, theme); err != nil {
			return err
		}
		w(5, latin)
		w(5, ea)
		if ctx.link != "" {
			// hlinkClick은 latin/ea 뒤다 — CT_TextCharacterProperties의 자식 순서가 폰트
			// 요소들 뒤에 hlinkClick을 둔다 (어기면 LibreOffice가 복구를 묻는다).
			w(5, fmt.Sprintf(`<a:hlinkClick r:id="%s"/>`, rels.linkID(ctx.link)))
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

// writeTextFill은 텍스트 색을 rPr의 채움 요소로 쓴다 — 그라디언트는 `gradFill`, 단색·변형은
// `solidFill`이다 (PRD §9.4의 텍스트 `color=gradient` 행).
//
// 토큰은 **schemeClr 참조로 남는다** — 여기서 RGB로 눕히면 내보낸 파일이 PowerPoint의
// 테마 색 편집을 따라가지 못한다 (soml.Color의 주석, 제품 핵심 계약). 텍스트에는 도형
// 불투명도가 없어 외부 알파는 nil이다 (변형 `alpha()`는 색 자신이 싣는다).
func writeTextFill(w func(depth int, line string), depth int, color soml.Color, theme soml.Theme) error {
	if color.IsGradient() {
		return writeGradFill(w, depth, *color.Gradient, nil, theme)
	}
	w(depth, "<a:solidFill>")
	if err := writeColor(w, depth+1, color, nil, theme); err != nil {
		return err
	}
	w(depth, "</a:solidFill>")
	return nil
}

// colorElem은 색 요소의 태그 이름과 속성이다 — 알파를 자식으로 갖는 표기(writeColor)와 한 줄
// 표기(colorXML)가 같은 판정을 쓰도록 나눠 둔 자리다. **단색만** 받는다: 그라디언트는 색
// 요소가 아니라 `gradFill`이므로 여기서 걸러 낸다 (writeGradFill이 소유한다).
func colorElem(color soml.Color, theme soml.Theme) (name, attrs string, err error) {
	switch {
	case color.IsGradient():
		return "", "", fmt.Errorf("그라디언트는 단일 색 요소가 아니다 — gradFill로 낸다")
	case color.IsZero():
		return "", "", fmt.Errorf("색이 해석되지 않았다")
	case color.IsToken():
		token, ok := theme.Color(color.Token)
		if !ok {
			return "", "", fmt.Errorf("색 토큰 %q가 테마에 없다", color.Token)
		}
		if token.Ref == "" {
			// Ref 없는 토큰은 schemeClr를 만들 수 없다 — RGB로 눕히는 폴백은 토큰 보존
			// 계약을 조용히 깨므로 두지 않는다.
			return "", "", fmt.Errorf("색 토큰 %q에 schemeClr 참조(Ref)가 없다", color.Token)
		}
		return "a:schemeClr", fmt.Sprintf(`val="%s"`, token.Ref), nil
	default:
		return "a:srgbClr", fmt.Sprintf(`val="%s"`, color.RGB), nil
	}
}

// colorTransforms는 색 요소의 변환 자식(lumMod/lumOff/alpha)을 만든다 — **색 변형 함수와
// 외부 불투명도(opacity)를 한 자리에서 합친다** (PRD §9.4의 색 변형 행).
//
// 순서는 lumMod → lumOff → alpha로 고정한다 (스키마는 순서를 강제하지 않지만 골든이 한 가지
// 표기로 남는다). 알파는 하나만 낸다: 색 자신의 `alpha()`(mod)와 도형 불투명도(extraAlpha)가
// 함께 있으면 곱한다 — 웹의 합성 순서(색 알파 × svg opacity)와 같다. **schemeClr 참조는
// 눕지 않는다**: 변환은 색 요소의 자식이므로 토큰 색이 여전히 테마 편집을 따라간다.
func colorTransforms(mod *soml.ColorMod, extraAlpha *float64) []string {
	var lines []string
	effAlpha := 1.0
	alphaSet := false
	if extraAlpha != nil {
		effAlpha *= *extraAlpha
		alphaSet = true
	}
	if mod != nil {
		switch mod.Kind {
		case soml.ModShade:
			lines = append(lines, fmt.Sprintf(`<a:lumMod val="%d"/>`, LumPct(100-mod.Amount)))
		case soml.ModTint:
			lines = append(lines,
				fmt.Sprintf(`<a:lumMod val="%d"/>`, LumPct(100-mod.Amount)),
				fmt.Sprintf(`<a:lumOff val="%d"/>`, LumPct(mod.Amount)))
		case soml.ModAlpha:
			effAlpha *= mod.Amount
			alphaSet = true
		}
	}
	if alphaSet {
		lines = append(lines, fmt.Sprintf(`<a:alpha val="%d"/>`, Alpha(effAlpha)))
	}
	return lines
}

// writeColor는 색을 채움 요소의 자식으로 쓴다 — 변형(lumMod/lumOff/alpha)과 외부 불투명도가
// 있으면 색 요소가 그 자식을 갖는다.
//
// **알파가 색 안에 사는 것이 OOXML의 구조다** (도형 단위 불투명도가 없다 — spSpec.alpha).
// 그래서 채움·선·그림자가 각자 자기 색에 같은 알파를 싣고, 웹은 그림 svg 하나에 `opacity`를
// 걸어 같은 결과를 낸다 (render/web.shapeInkCSS). 변형이 없고 알파도 없으면 자식 없는 한 줄
// 표기라 기존 산출물과 바이트가 같다.
func writeColor(w func(depth int, line string), depth int, color soml.Color, alpha *float64, theme soml.Theme) error {
	name, attrs, err := colorElem(color, theme)
	if err != nil {
		return err
	}
	children := colorTransforms(color.Mod, alpha)
	if len(children) == 0 {
		w(depth, fmt.Sprintf("<%s %s/>", name, attrs))
		return nil
	}
	w(depth, fmt.Sprintf("<%s %s>", name, attrs))
	for _, child := range children {
		w(depth+1, child)
	}
	w(depth, fmt.Sprintf("</%s>", name))
	return nil
}

// writeGradFill은 그라디언트 색을 `a:gradFill`로 쓴다 (PRD §9.4의 `fill=gradient` 행 —
// PPTX는 `gradFill`).
//
// 스톱은 균등 배치다 (soml.Gradient): n개 스톱이 0, 1/(n-1), …, 1(100000분율)에 놓인다.
// 각 스톱 색은 `schemeClr`/`srgbClr`를 유지하므로(colorElem) 토큰 스톱이 테마 편집을
// 따라간다. 각도는 GradientAngle이 OOXML `a:lin@ang`으로 바꾸고, `scaled="1"`은 그라디언트가
// 도형 크기에 맞춰 늘어난다는 뜻이다 (웹의 objectBoundingBox와 같은 결정).
//
// 외부 불투명도(도형 opacity)는 스톱마다 `a:alpha`로 실린다 — 도형 전체 알파를 gradFill이
// 통째로 받는 표기가 없어(alpha는 색의 성질이다) 스톱 색에 나눠 싣는다.
func writeGradFill(w func(depth int, line string), depth int, grad soml.Gradient, alpha *float64, theme soml.Theme) error {
	if len(grad.Stops) < 2 {
		return fmt.Errorf("그라디언트 스톱이 둘 미만이다 (%d)", len(grad.Stops))
	}
	w(depth, "<a:gradFill>")
	w(depth+1, "<a:gsLst>")
	last := len(grad.Stops) - 1
	for i, stop := range grad.Stops {
		pos := int64(math.Round(float64(i) / float64(last) * PercentScale))
		name, attrs, err := colorElem(stop, theme)
		if err != nil {
			return err
		}
		w(depth+2, fmt.Sprintf(`<a:gs pos="%d">`, pos))
		children := colorTransforms(stop.Mod, alpha)
		if len(children) == 0 {
			w(depth+3, fmt.Sprintf("<%s %s/>", name, attrs))
		} else {
			w(depth+3, fmt.Sprintf("<%s %s>", name, attrs))
			for _, child := range children {
				w(depth+4, child)
			}
			w(depth+3, fmt.Sprintf("</%s>", name))
		}
		w(depth+2, "</a:gs>")
	}
	w(depth+1, "</a:gsLst>")
	w(depth+1, fmt.Sprintf(`<a:lin ang="%d" scaled="1"/>`, GradientAngle(grad.Angle)))
	w(depth, "</a:gradFill>")
	return nil
}

// validateColor는 색(단색·변형·그라디언트)이 OOXML로 컴파일될 수 있는지 검증한다 —
// checkTextStyle이 텍스트 색을 미리 검사하는 데 쓴다 (해석되지 않은 색이 산출물로 새는 것을
// 막는다).
func validateColor(color soml.Color, theme soml.Theme) error {
	if color.IsGradient() {
		if len(color.Gradient.Stops) < 2 {
			return fmt.Errorf("그라디언트 스톱이 둘 미만이다 (%d)", len(color.Gradient.Stops))
		}
		for _, stop := range color.Gradient.Stops {
			if _, _, err := colorElem(stop, theme); err != nil {
				return err
			}
		}
		return nil
	}
	_, _, err := colorElem(color, theme)
	return err
}

// typefaces는 폰트 토큰을 rPr의 latin/ea 요소로 만든다 (값은 typefaceValues).
func typefaces(role soml.FontRole, theme soml.Theme) (latin, ea string, err error) {
	lt, et, err := typefaceValues(role, theme)
	if err != nil {
		return "", "", err
	}
	return fmt.Sprintf(`<a:latin typeface="%s"/>`, escapeXML(lt)),
		fmt.Sprintf(`<a:ea typeface="%s"/>`, escapeXML(et)), nil
}

// typefaceValues는 폰트 토큰의 latin/ea typeface 값이다 — rPr의 latin/ea(typefaces)와
// 리스트 마커의 buFont(bulletFontXML)가 같은 값을 쓴다.
//
// fontScheme 슬롯이 있는 토큰은 `+mj-lt` 같은 **테마 참조**로 남는다 — 색과 같은 이유로,
// 참조여야 내보낸 파일이 PowerPoint의 테마 폰트 변경을 따라간다. 슬롯 없는 토큰(mono)은
// 리터럴 typeface다: fontScheme에는 major/minor 두 자리뿐이라 세 번째 폰트를 실을 곳이
// 없다 (soml.FontToken.Scheme의 주석 — 테마 폰트 편집을 따라가지 못하는 유일한 예외).
func typefaceValues(role soml.FontRole, theme soml.Theme) (latin, ea string, err error) {
	if role == "" {
		return "", "", fmt.Errorf("폰트 토큰이 해석되지 않았다")
	}
	token, ok := theme.Font(role)
	if !ok {
		return "", "", fmt.Errorf("폰트 토큰 %q가 테마에 없다", string(role))
	}
	switch token.Scheme {
	case "major":
		return "+mj-lt", "+mj-ea", nil
	case "minor":
		return "+mn-lt", "+mn-ea", nil
	case "":
		return token.Latin, token.EastAsian, nil
	}
	return "", "", fmt.Errorf("폰트 토큰 %q의 fontScheme 슬롯 %q를 모른다 (major/minor/빈 문자열)",
		string(role), token.Scheme)
}
