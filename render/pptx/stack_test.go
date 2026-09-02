package pptx

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/k2sebeom/balpyo/render/geom"
)

// 이 파일은 Iteration 6의 겹침 계약을 산출물에 못 박는다: **문서 순서가 곧 z-순서다**
// (PRD §13-3). `<stack>`은 KindLayoutContainer이므로 조립기는 이미 자동으로 맞지만
// (writeShapeNode의 레이아웃 컨테이너 분기 — 도형 0개 + 자식 재귀), 테스트가 없으면 다음
// 사람이 stack에 `p:grpSp`를 붙이는 것(Iteration 9의 `<group>`과 혼동)을 막을 것이 없다.
//
// **stack과 group은 이 iteration에서 처음 실물로 갈라진다**: stack은 배치 후 소멸하고
// (자식이 절대 좌표 도형이 된다), group은 편집 단위로 남는다(grpSp + 그룹 상대 좌표,
// Iteration 9). group이 열리는 날 "stack은 여전히 소멸한다"를 지키는 것이 아래 테스트들이다.

// `<stack>`은 레이아웃 컨테이너다 — `<row>`/`<col>`과 똑같이 도형이 되지 않고 자식만
// 도형이 된다. 표로 나란히 세워 stack이 낸 도형 수(0)와 자식이 낸 도형 수(2)를 함께 못
// 박는다: 겹침 슬라이드에서도 컨테이너 항은 사라지고 자식만 남는다.
func TestRenderStackEmitsNoShapesLikeLayoutContainers(t *testing.T) {
	// `<row>`·`<col>`과 나란히 둔다 — stack이 셋과 다른 산출을 내면(예: grpSp) 여기가
	// 빨개진다. 셋 다 자식 둘이 같은 좌표로 겹쳐도 도형 둘이 문서 순서로 나온다.
	for _, container := range []string{"row", "col", "stack"} {
		t.Run(container, func(t *testing.T) {
			deck := resolvedDeck(t, fmt.Sprintf(`<slide>
  <%s>
    <text>아래</text>
    <text>위</text>
  </%s>
</slide>`, container, container))
			g := measuredGeometry(t, deck,
				geom.Rect{X: 7, Y: 7, W: 7, H: 7},      // 컨테이너 — 도형이 아니다
				geom.Rect{X: 64, Y: 64, W: 400, H: 36}, // text "아래"
				geom.Rect{X: 64, Y: 64, W: 400, H: 36}, // text "위" (같은 좌표 = 겹침)
			)
			xml := slideXML(t, deck, g)

			ids, names := shapeNames(xml)
			// 컨테이너 0개 + 자식 2개 — 이 두 수가 나란히 계약이다.
			want := []string{"text s0.0.0", "text s0.0.1"}
			if strings.Join(names, "|") != strings.Join(want, "|") {
				t.Errorf("<%s>의 도형이 %v, want %v — 컨테이너가 도형이 되었거나 자식이 빠졌다\n%s",
					container, names, want, xml)
			}
			// 컨테이너 이름이 도형으로 나오면 안 된다.
			if strings.Contains(xml, fmt.Sprintf(`name="%s s0.0"`, container)) {
				t.Errorf("<%s>가 도형이 되었다", container)
			}
			// 컨테이너는 도형 id를 소비하지 않는다 — 자식이 2·3이다 (id=1은 spTree 그룹).
			if strings.Join(ids, ",") != "2,3" {
				t.Errorf("<%s>의 도형 id가 %v, want [2 3] — 컨테이너가 번호를 소비했다", container, ids)
			}
			// 컨테이너의 잰 rect(7px = 66675 EMU)는 아무 도형도 쓰지 않는다.
			if strings.Contains(xml, containerRectEMU) {
				t.Errorf("<%s>의 rect가 도형 좌표로 쓰였다\n%s", container, xml)
			}
		})
	}
}

// zorderCNvPr는 spTree의 도형(p:sp)과 그림(p:pic)이 공유하는 이름표다.
var zorderCNvPr = regexp.MustCompile(`<p:cNvPr id="(\d+)" name="([^"]*)"`)

// shapesInDocumentOrder는 spTree에 **등장 순서**로 나온 도형·그림의 (id, 이름)이다.
//
// shapeNames(layout_test.go)와 갈라 둔 이유: 그쪽은 `<p:sp>`만 본다. 겹침 z-순서는 도형과
// 그림을 한 순회에서 보아야 하므로(그림도 문서 순서 자리에 낀다) cNvPr를 종류 무관하게
// 훑고, spTree 그룹(id=1)만 뺀다.
func shapesInDocumentOrder(xml string) (ids, names []string) {
	for _, m := range zorderCNvPr.FindAllStringSubmatch(xml, -1) {
		if m[1] == "1" {
			continue // spTree 그룹(nvGrpSpPr) — 도형이 아니다
		}
		ids = append(ids, m[1])
		names = append(names, m[2])
	}
	return ids, names
}

// 겹침 슬라이드에서 spTree의 도형 등장 순서 = 문서 순서 = z-순서다 — OOXML은 나중 도형이
// 위에 그려지므로 저자가 쓴 순서가 겹침 순서가 된다 (웹의 문서 순서 스택과 같은 규칙).
// 특히 항목 04가 손댄 배합(`x`/`y` 좌표 자식과 `at`·정적 자식을 한 stack에 섞은 슬라이드)에서
// 순서가 유지되는지, 그리고 p:sp(도형·텍스트)와 p:pic(그림)이 같은 순회에서 문서 순서로
// 나오는지 본다. 웹의 칠 순서는 픽셀 비교(항목 06)가 보고, 여기서는 XML 순서를 본다.
func TestRenderStackZOrderIsDocumentOrder(t *testing.T) {
	deck := imageDeck(t, fmt.Sprintf(`<slide>
  <stack>
    <image %s w="400" h="240"/>
    <shape type="rect" fill="surface" w="200" h="120" at="center"/>
    <text at="center">가운데 배지</text>
    <text x="-24" y="-16">모서리 밖 배지</text>
  </stack>
</slide>`, wideSrc))
	g := measuredGeometry(t, deck,
		geom.Rect{X: 400, Y: 200, W: 400, H: 240}, // stack — 컨테이너(도형 아님)
		geom.Rect{X: 400, Y: 200, W: 400, H: 240}, // image (맨 아래)
		geom.Rect{X: 500, Y: 260, W: 200, H: 120}, // shape (가운데 카드)
		geom.Rect{X: 520, Y: 300, W: 160, H: 36},  // text  (가운데 배지)
		geom.Rect{X: 376, Y: 184, W: 200, H: 36},  // text  (모서리 밖 배지 — 맨 위)
	)
	xml := slideXML(t, deck, g)

	ids, names := shapesInDocumentOrder(xml)
	wantNames := []string{"image s0.0.0", "shape s0.0.1", "text s0.0.2", "text s0.0.3"}
	if strings.Join(names, "|") != strings.Join(wantNames, "|") {
		t.Fatalf("spTree 등장 순서가 %v, want %v — 문서 순서 ≠ z-순서\n%s", names, wantNames, xml)
	}
	// 정규식 순서가 아니라 **실제 문자열 위치**로도 확인한다 — 문서 순서대로 등장 위치가
	// 단조 증가해야 한다 (겹침 계약의 전부가 이 순서다).
	prev := -1
	for _, name := range wantNames {
		at := strings.Index(xml, fmt.Sprintf(`name="%s"`, name))
		if at < 0 {
			t.Fatalf("%s가 spTree에 없다\n%s", name, xml)
		}
		if at <= prev {
			t.Errorf("%s가 앞 도형보다 먼저 나온다 — z-순서가 문서 순서가 아니다", name)
		}
		prev = at
	}
	// 도형 id는 도형을 실제로 낸 자리에서만 2부터 오른다 — stack(컨테이너)이 끼어도
	// 건너뛰지 않고(shape.go firstShapeID·writeShapeNode), p:pic도 같은 id 공간을 쓴다.
	if strings.Join(ids, ",") != "2,3,4,5" {
		t.Errorf("도형 id가 %v, want [2 3 4 5] — 컨테이너가 번호를 소비했거나 p:pic이 id 공간을 벗어났다", ids)
	}
}

// 음수·슬라이드 밖 좌표가 산출물까지 그대로 흐르는가 (항목 03이 `x`/`y`에 음수를 열었다,
// PRD §13-1의 오버플로 허용). geom은 음수 rect를 통과시키고(valid는 W·H만 음수를 막는다),
// Offset은 음수 EMU를 0쪽으로 자르지 않고 그대로 찍는다 — 박스 밖으로 반쯤 걸친 배지가
// 오버레이의 가장 흔한 장면이므로 이 경로가 살아 있어야 한다.
func TestRenderStackChildNegativeCoordinateEmitsSignedOffset(t *testing.T) {
	deck := resolvedDeck(t, `<slide>
  <stack>
    <text x="-24" y="-16">모서리 밖 배지</text>
  </stack>
</slide>`)
	g := measuredGeometry(t, deck,
		geom.Rect{X: 400, Y: 200, W: 400, H: 240}, // stack
		geom.Rect{X: -24, Y: -16, W: 200, H: 36},  // text — 슬라이드 밖으로 걸침
	)
	xml := slideXML(t, deck, g)

	// 기본 정렬(start)이라 안전 여유는 오른쪽으로만 붙어 x는 잰 값 그대로다: -24px = -228600,
	// -16px = -152400 EMU. 부호 있는 좌표는 OOXML `a:off`에 정당하다.
	if want := `<a:off x="-228600" y="-152400"/>`; !strings.Contains(xml, want) {
		t.Errorf("음수 좌표가 부호 있는 a:off로 나오지 않았다 (want %q)\n%s", want, xml)
	}
}

// 겹침은 순서에 의존하므로 결정성을 한 번 더 본다 (도형 조립에 맵 순회가 새면 겹친 도형의
// 순서·관계 id가 흔들린다). 같은 겹침 덱을 두 번 빌드하면 바이트가 같아야 골든 비교가
// 성립한다 (기존 계약 — TestRenderIsByteIdentical의 겹침판).
func TestRenderStackOverlapDeckIsByteIdentical(t *testing.T) {
	build := func() []byte {
		deck := imageDeck(t, fmt.Sprintf(`<slide>
  <stack>
    <image %s w="400" h="240"/>
    <shape type="rect" fill="surface" w="200" h="120" at="center"/>
    <text at="center">배지</text>
  </stack>
</slide>`, wideSrc))
		g := measuredGeometry(t, deck,
			geom.Rect{X: 400, Y: 200, W: 400, H: 240},
			geom.Rect{X: 400, Y: 200, W: 400, H: 240},
			geom.Rect{X: 500, Y: 260, W: 200, H: 120},
			geom.Rect{X: 520, Y: 300, W: 160, H: 36},
		)
		files, err := NewWithGeometry(g).Render(deck)
		if err != nil {
			t.Fatal(err)
		}
		return files[FileName(deck)]
	}
	if !bytes.Equal(build(), build()) {
		t.Error("같은 겹침 덱이 다른 바이트를 냈다 — 겹침 순서가 결정적이지 않다")
	}
}
