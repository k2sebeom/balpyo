package pptx

import (
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/k2sebeom/balpyo/render/geom"
	"github.com/k2sebeom/balpyo/soml"
)

// 이 파일은 이 iteration의 핵심 계약을 산출물에 못 박는다: **레이아웃 태그는 PPTX 도형이
// 되지 않는다** (TASKS Iteration 3). 지오메트리 패스가 컨테이너도 재지만(render/geom.collect —
// 픽셀 비교의 진단용), 조립기는 그 rect를 소비하지 않고 자식 좌표만 도형으로 낸다.
//
// 좌표 정확성의 판정은 픽셀 비교(항목 05)가 하고, 여기서는 구조를 본다: 도형 수·이름·순서·
// id와, 자식의 xfrm이 **컨테이너 상대가 아니라 측정 rect 그대로**인지.

// layoutSlide는 이 파일의 기준 슬라이드다 — 중첩(col > row) + spacer + grid 안의 리스트 +
// 빈 컨테이너를 한 장에 담는다. 요소 순서(= geom.Refs 순서, 식별자):
//
//	0 col      s0.0        레이아웃 컨테이너 → 도형 0개
//	1 row      s0.0.0      레이아웃 컨테이너 → 도형 0개
//	2 text     s0.0.0.0    → 도형
//	3 spacer   s0.0.0.1    → 도형 0개
//	4 p        s0.0.0.2    → 도형
//	5 grid     s0.0.1      레이아웃 컨테이너 → 도형 0개
//	6 ul       s0.0.1.0    → 도형 **하나** (리스트 컨테이너의 대비)
//	7 li       s0.0.1.0.0  → 도형 0개 (컨테이너의 도형이 좌표를 소유한다)
//	8 text     s0.0.1.1    → 도형
//	9 row      s0.0.2      빈 컨테이너 → 도형 0개
const layoutSlide = `<slide>
  <col gap="lg">
    <row gap="md">
      <text>왼쪽</text>
      <spacer/>
      <p>오른쪽</p>
    </row>
    <grid cols="2">
      <ul>
        <li>항목</li>
      </ul>
      <text>카드</text>
    </grid>
    <row></row>
  </col>
</slide>`

// layoutRects는 layoutSlide의 요소 순서에 붙일 rect다 — 요소마다 다른 값이어야 좌표가 어느
// 도형으로 갔는지 구별할 수 있다 (measuredGeometry의 근거).
//
// 컨테이너·spacer·li의 rect는 **아무 도형도 쓰지 않아야 하는 값**이다: 7px 계열로 몰아 두어
// 조립기가 그것을 소비하면 EMU 66675가 산출 XML에 나타난다 (containerRectEMU).
func layoutRects() []geom.Rect {
	return []geom.Rect{
		{X: 7, Y: 7, W: 7, H: 7},        // col
		{X: 7, Y: 7, W: 7, H: 7},        // row
		{X: 64, Y: 64, W: 300, H: 36},   // text
		{X: 7, Y: 7, W: 7, H: 7},        // spacer
		{X: 916, Y: 64, W: 300, H: 36},  // p
		{X: 7, Y: 7, W: 7, H: 7},        // grid
		{X: 64, Y: 136, W: 564, H: 80},  // ul
		{X: 7, Y: 7, W: 7, H: 7},        // li
		{X: 652, Y: 136, W: 564, H: 36}, // text
		{X: 7, Y: 7, W: 7, H: 7},        // 빈 row
	}
}

// containerRectEMU는 layoutRects의 "쓰이면 안 되는" 좌표의 EMU다 (7px × 9525).
const containerRectEMU = "66675"

// layoutShapeNames는 layoutSlide가 내야 하는 도형 이름을 spTree 순서로 적은 것이다 —
// 이름은 태그 + 지오메트리 식별자다 (writeSp).
var layoutShapeNames = []string{
	"text s0.0.0.0",
	"p s0.0.0.2",
	"ul s0.0.1.0",
	"text s0.0.1.1",
}

// shapeNameRE는 도형 조각에서 id와 이름을 뽑는다.
var shapeNameRE = regexp.MustCompile(`<p:cNvPr id="(\d+)" name="([^"]*)"/>`)

// shapeNames는 슬라이드 XML의 도형 (id, 이름)을 spTree 순서로 돌려준다.
//
// `<p:sp>` 조각 안에서만 찾는다 — spTree의 그룹(nvGrpSpPr)도 같은 모양의 cNvPr를 갖고
// id=1을 쓰므로, XML 전체에서 찾으면 도형이 아닌 그 자리가 목록에 섞인다.
func shapeNames(xml string) (ids []string, names []string) {
	for _, sp := range strings.Split(xml, "<p:sp>")[1:] {
		m := shapeNameRE.FindStringSubmatch(sp)
		if m == nil {
			continue
		}
		ids = append(ids, m[1])
		names = append(names, m[2])
	}
	return ids, names
}

// 레이아웃 컨테이너와 `<spacer>`는 도형을 내지 않고, 자식만 도형이 된다 — 이 iteration의
// 핵심 계약이다. 깊이 우선 순회이므로 문서 순서가 그대로 spTree 순서(= z-순서)이고, 도형
// id는 도형을 낸 자리에서만 오른다 (배치 구조가 번호를 밀지 않는다 — writeShapeNode).
func TestRenderLayoutContainersEmitNoShapes(t *testing.T) {
	deck := resolvedDeck(t, layoutSlide)
	g := measuredGeometry(t, deck, layoutRects()...)
	xml := slideXML(t, deck, g)

	ids, names := shapeNames(xml)
	if got := strings.Count(xml, "<p:sp>"); got != len(layoutShapeNames) {
		t.Fatalf("p:sp %d개, want %d (도형 %v)\n%s", got, len(layoutShapeNames), names, xml)
	}
	// 이름과 순서 전수 — 문서 순서 = spTree 순서가 중첩에서도 유지된다.
	if strings.Join(names, "|") != strings.Join(layoutShapeNames, "|") {
		t.Errorf("도형 이름·순서가 %v, want %v", names, layoutShapeNames)
	}
	// 도형 id는 2부터 빈틈 없이 이어진다 (id=1은 spTree 그룹). 컨테이너·spacer·li가 번호를
	// 소비하면 여기가 2,3,4,5에서 벌어진다.
	if want := []string{"2", "3", "4", "5"}; strings.Join(ids, ",") != strings.Join(want, ",") {
		t.Errorf("도형 id가 %v, want %v — 도형이 아닌 요소가 번호를 소비했다", ids, want)
	}
	// 컨테이너·spacer·빈 컨테이너의 이름이 도형으로 나오면 안 된다.
	for _, name := range []string{"col s0.0", "row s0.0.0", "spacer s0.0.0.1", "grid s0.0.1", "row s0.0.2", "li s0.0.1.0.0"} {
		if strings.Contains(xml, fmt.Sprintf(`name="%s"`, name)) {
			t.Errorf("%s가 도형이 되었다", name)
		}
	}
	// 컨테이너의 잰 rect는 아무 도형도 쓰지 않는다 — 소비했다면 이 EMU가 xfrm에 나타난다.
	if strings.Contains(xml, containerRectEMU) {
		t.Errorf("컨테이너·spacer의 rect(EMU %s)가 도형 좌표로 쓰였다\n%s", containerRectEMU, xml)
	}
}

// 중첩된 자식의 xfrm은 **잰 rect 그대로**(+ 정렬 방향의 안전 여유)다 — 컨테이너 안이라는
// 사실이 좌표에 영향을 주지 않는다 (rect는 이미 슬라이드 상대다). 컨테이너 상대로 잘못
// 계산하면(예: 부모 x를 더하면) 이 테스트가 잡는다.
func TestRenderNestedShapeCoordinatesComeFromMeasuredRects(t *testing.T) {
	deck := resolvedDeck(t, layoutSlide)
	rects := layoutRects()
	g := measuredGeometry(t, deck, rects...)
	xml := slideXML(t, deck, g)

	// 도형이 되는 요소와 그 rect·정렬 (리스트는 언제나 start — writeListShape).
	for _, tc := range []struct {
		name  string
		rect  geom.Rect
		align soml.TextAlign
	}{
		{"text s0.0.0.0", rects[2], soml.AlignStart},
		{"p s0.0.0.2", rects[4], soml.AlignStart},
		{"ul s0.0.1.0", rects[6], soml.AlignStart},
		{"text s0.0.1.1", rects[8], soml.AlignStart},
	} {
		box := TextBoxRect(tc.rect, tc.align)
		x, y := Offset(box)
		cx, cy := Extent(box)
		want := fmt.Sprintf("<a:off x=\"%d\" y=\"%d\"/>\n            <a:ext cx=\"%d\" cy=\"%d\"/>", x, y, cx, cy)
		if !strings.Contains(xml, want) {
			t.Errorf("%s의 xfrm이 측정값 %s(+여유)에서 오지 않았다 (want %q)\n%s", tc.name, tc.rect, want, xml)
		}
	}
	// 환산이 실제로 EMU인지 한 값은 손으로 못 박는다: 중첩된 첫 `<text>`는 (64,64) 300×36에
	// start 여유 4%가 붙어 312px 폭이다 — 64px = 609600, 312px = 2971800, 36px = 342900.
	if want := `<a:off x="609600" y="609600"/>`; !strings.Contains(xml, want) {
		t.Errorf("중첩 자식의 a:off가 %q가 아니다", want)
	}
	if want := `<a:ext cx="2971800" cy="342900"/>`; !strings.Contains(xml, want) {
		t.Errorf("중첩 자식의 a:ext가 %q가 아니다", want)
	}
}

// 같은 단어를 쓰는 두 "컨테이너"가 정반대로 매핑된다: 리스트 컨테이너는 도형 **하나**,
// 레이아웃 컨테이너는 도형 **0개**다 (writeShapeNode의 대비). 태그를 늘리는 사람이 가장
// 헷갈릴 자리이므로 한 슬라이드에서 나란히 단정한다.
func TestRenderListContainerVersusLayoutContainer(t *testing.T) {
	deck := resolvedDeck(t, `<slide>
  <ul>
    <li>항목</li>
  </ul>
  <col>
    <text>칼럼</text>
  </col>
</slide>`)
	g := measuredGeometry(t, deck,
		geom.Rect{X: 64, Y: 64, W: 1152, H: 40},  // ul   s0.0   → 도형 하나
		geom.Rect{X: 64, Y: 64, W: 1152, H: 40},  // li   s0.0.0 → 도형 없음
		geom.Rect{X: 64, Y: 120, W: 1152, H: 36}, // col  s0.1   → 도형 없음
		geom.Rect{X: 64, Y: 120, W: 1152, H: 36}, // text s0.1.0 → 도형
	)
	xml := slideXML(t, deck, g)

	_, names := shapeNames(xml)
	want := []string{"ul s0.0", "text s0.1.0"}
	if strings.Join(names, "|") != strings.Join(want, "|") {
		t.Errorf("도형이 %v, want %v — 리스트 1개 + 레이아웃 0개가 아니다\n%s", names, want, xml)
	}
	// 리스트는 항목이 여럿이어도 도형 하나다(문단이 늘어난다), 레이아웃은 자식이 여럿이어도
	// 도형 0개다(자식이 각자 도형이다) — 그 비대칭이 이 두 수에 있다.
	if got := strings.Count(xml, "<a:buChar"); got != 1 {
		t.Errorf("불릿 문단 %d개, want 1", got)
	}

	// 웹에서는 반대로 컨테이너가 실제 요소로 남는다 — 두 타깃의 결정이 갈라지는 것이 아니라,
	// 컨테이너가 웹에서는 배치를 하고 PPTX에서는 좌표로 구워져 사라지는 것이다.
	html := renderWebString(t, deck)
	if !strings.Contains(html, `data-balpyo-el="col"`) {
		t.Error("웹 산출물에 <col> 요소가 없다 — 대비의 절반이 성립하지 않는다")
	}
}

// 빈 컨테이너(`<row></row>`)와 `<spacer/>`는 도형을 내지 않는다. 대비되는 선례는 빈
// `<text>`가 도형을 내는 것이다 (writeTextShape) — 차이는 `<text>`가 저자의 콘텐츠이고
// 컨테이너·spacer는 배치 수단이라는 것이다.
func TestRenderEmptyContainerAndSpacerEmitNoShapesButEmptyTextDoes(t *testing.T) {
	deck := resolvedDeck(t, `<slide>
  <row>
    <spacer/>
  </row>
  <text></text>
</slide>`)
	g := measuredGeometry(t, deck,
		geom.Rect{X: 64, Y: 64, W: 1152, H: 0}, // row    s0.0
		geom.Rect{X: 64, Y: 64, W: 1152, H: 0}, // spacer s0.0.0
		geom.Rect{X: 64, Y: 64, W: 0, H: 0},    // text   s0.1
	)
	xml := slideXML(t, deck, g)

	_, names := shapeNames(xml)
	if len(names) != 1 || names[0] != "text s0.1" {
		t.Errorf("도형이 %v, want [text s0.1]\n%s", names, xml)
	}
}

// 레이아웃 컨테이너 안의 문자 데이터는 파서가 막는다 (modelElements) — 파서를 거치지 않은
// 덱이 조용히 통과하면 그 자리의 글자가 산출물에서 사라진다.
func TestRenderLayoutContainerRejectsNonElementChild(t *testing.T) {
	deck := resolvedDeck(t, `<slide>
  <row>
    <text>본문</text>
  </row>
</slide>`)
	row := deck.Slides[0].Content[0].(*soml.Element)
	row.Children = append(row.Children, &soml.Text{Value: "떠 있는 글자"})
	// 문자 데이터는 측정 대상이 아니므로(geom.collect) 지오메트리는 그대로 유효하다 —
	// 순회가 조용히 건너뛰면 실패가 드러나지 않는 것이 정확히 이 상황이다.
	g := measuredGeometry(t, deck,
		geom.Rect{X: 64, Y: 64, W: 1152, H: 36},
		geom.Rect{X: 64, Y: 64, W: 1152, H: 36},
	)

	_, err := NewWithGeometry(g).Render(deck)
	if err == nil {
		t.Fatal("컨테이너 안의 문자 데이터가 조용히 통과했다")
	}
	if !strings.Contains(err.Error(), "요소가 아닌 노드") {
		t.Errorf("에러가 노드 종류를 가리키지 않는다: %v", err)
	}
	// 진단은 어느 자리인지 말해야 한다 — 컨테이너의 식별자가 아니라 그 자식 자리다.
	if !strings.Contains(err.Error(), "s0.0.1") {
		t.Errorf("에러가 자리(식별자)를 가리키지 않는다: %v", err)
	}
}

// 레이아웃 컨테이너가 든 덱도 소비 시점의 대조(geom.Verify)를 지난다 — 컨테이너는 도형이
// 되지 않지만 **측정 대상이므로** Refs에는 있다 (render/geom.collect). 조립 순회가 컨테이너를
// 건너뛰는 것과 측정 목록이 컨테이너를 세는 것은 서로 다른 계약이고, 둘이 어긋나면 좌표가
// 엉뚱한 요소로 간다.
func TestRenderLayoutDeckPassesGeometryVerify(t *testing.T) {
	deck := resolvedDeck(t, layoutSlide)
	refs := geom.Refs(deck)
	if len(refs) != len(layoutRects()) {
		t.Fatalf("Refs %d개, 기대 %d개 — 측정 대상 목록이 바뀌었다면 layoutRects도 따라야 한다",
			len(refs), len(layoutRects()))
	}
	// 컨테이너·spacer가 Refs에 있다는 사실 자체를 못 박는다 (도형이 되지 않는 것과 별개다).
	var kinds []soml.ElementKind
	for _, ref := range refs {
		kinds = append(kinds, soml.KindOf(ref.El))
	}
	var containers, spacers int
	for _, kind := range kinds {
		switch kind {
		case soml.KindLayoutContainer:
			containers++
		case soml.KindSpacer:
			spacers++
		}
	}
	if containers != 4 || spacers != 1 {
		t.Errorf("측정 대상에 컨테이너 %d개·spacer %d개, want 4·1", containers, spacers)
	}
	// 도형 수 = 측정 대상 수 − 컨테이너 − spacer − li (픽스처 수준에서 같은 관계를
	// internal/testkit이 실제 산출물로 한 번 더 단정한다).
	if want := len(refs) - containers - spacers - 1; want != len(layoutShapeNames) {
		t.Errorf("도형 수 기대가 %d, 이 파일의 목록은 %d개다", want, len(layoutShapeNames))
	}

	g := measuredGeometry(t, deck, layoutRects()...)
	if err := g.Verify(deck); err != nil {
		t.Fatalf("레이아웃 덱의 지오메트리 대조가 실패했다: %v", err)
	}
	if _, err := NewWithGeometry(g).Render(deck); err != nil {
		t.Fatalf("레이아웃 덱이 렌더되지 않았다: %v", err)
	}
}
