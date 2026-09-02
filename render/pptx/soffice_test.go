// LibreOffice 열림 테스트. 도형이 든 슬라이드의 XML이 스키마에서 벗어나면 골든·단위
// 테스트는 초록인 채 사용자만 복구 경고를 본다 — 그래서 조립 결과를 실제 뷰어로 연다.
//
// 외부 테스트 패키지(pptx_test)인 이유: internal/testkit이 internal/pipeline을 거쳐 이
// 패키지를 임포트하므로, 패키지 안 테스트에서 testkit을 부르면 순환이 된다.
package pptx_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/k2sebeom/balpyo/core"
	"github.com/k2sebeom/balpyo/internal/testkit"
	"github.com/k2sebeom/balpyo/render/geom"
	"github.com/k2sebeom/balpyo/render/pptx"
	"github.com/k2sebeom/balpyo/soml"
)

// LibreOffice가 텍스트 박스가 든 덱을 경고·복구 없이 여는지 본다. 원문은 실제 픽스처가
// 요구하는 것들을 담는다 — 한글, &, 인라인 외관, 빈 요소, 리스트, 레이아웃, 프리셋 도형,
// 그리고 노트. 노트는 **2번 슬라이드에만** 있다: notesSlide2가 notesSlide1 없이 존재하는,
// 번호가 비어 뛰는 경우다 (notesSlidePart의 번호 규칙) — 노트 파트는 관계·Content_Types가
// 조금만 틀려도 복구 경고로 드러나므로 이 게이트가 여기서 특히 값을 한다.
func TestLibreOfficeOpensTextShapes(t *testing.T) {
	// 도구가 없으면 렌더를 시작하기 전에 건너뛴다(로컬) 또는 실패한다(테스트 이미지).
	testkit.RequireSoffice(t)

	stage, err := soml.Aspect16x9.Stage()
	if err != nil {
		t.Fatal(err)
	}
	deck := &soml.Deck{Title: "open test", Aspect: soml.Aspect16x9, Stage: stage}
	sources := []string{
		`<slide>
  <text role="title" align="center">2026 아키텍처 리뷰</text>
  <text color="accent1">R&D 로드맵</text>
  <text role="caption" size="18" color="muted" align="end">2026 H1</text>
</slide>`,
		`<slide>
  <text></text>
  <text role="code" color="#abcdef">balpyo build --target pptx</text>
  <note>데모 실패 시 스크린샷으로 대체. R&D 예산 언급은 생략.</note>
</slide>`,
		// 런 분할·hlinkClick·a:br·pre — 슬라이드 rels의 External 관계와 rPr의 i/u·
		// hlinkClick 순서가 스키마를 벗어나면 여기서 복구 경고로 드러난다 (항목 06).
		// 같은 URL이 두 번 나와 관계 재사용 경로도 지난다.
		"<slide>\n" +
			"  <h2>배포 <strong>자동화</strong> 회고</h2>\n" +
			"  <p>지연 <span color=\"accent1\">-38%</span>, 상세는 <a href=\"https://example.com/deck?a=1&b=2\">발표 자료</a>와\n" +
			"  <a href=\"https://example.com/deck?a=1&b=2\">부록</a> 참고<br/>문의: <em>플랫폼팀</em> (<code>#platform</code>)</p>\n" +
			"  <pre>$ balpyo build\n\ndist/deck.pptx</pre>\n" +
			"</slide>",
		// 리스트 — pPr의 marL/indent(행잉 인덴트)·buChar/buAutoNum·buFont(테마 참조
		// "+mn-lt")·li 안의 인라인 런·<br/>·빈 li가 스키마를 벗어나면 여기서 복구
		// 경고로 드러난다 (항목 07).
		`<slide>
  <h2>다음 분기 계획</h2>
  <ul>
    <li>측정 파이프라인 <strong>안정화</strong></li>
    <li>상세는 <a href="https://example.com/plan">계획 문서</a> 참고<br/>이어짐 줄</li>
    <li></li>
  </ul>
  <ol>
    <li>첫째 단계</li>
    <li>둘째 단계</li>
  </ol>
</slide>`,
		// 레이아웃 — 중첩 컨테이너 + <grid cols="4"> 카드 + <spacer/>. 컨테이너는 도형이
		// 되지 않으므로 구조적으로는 텍스트 박스 여러 개일 뿐이지만(항목 04), 좌표가
		// 많아지면 겹침·음수 좌표 같은 실수가 여기서 처음 드러난다. 빈 컨테이너와
		// 컨테이너 안의 리스트도 함께 지난다 — 도형 0개와 도형 1개가 같은 순회에서 나온다.
		`<slide>
  <col gap="lg" pad="lg">
    <row justify="between" align="center">
      <h2>분기 지표</h2>
      <spacer/>
      <text role="caption" color="muted">2026 H1</text>
    </row>
    <grid cols="4" gap="md">
      <col><text role="heading">배포</text><text>주 12회</text></col>
      <col><text role="heading">지연</text><text>-38%</text></col>
      <col><text role="heading">가동률</text><text>99.95%</text></col>
      <col><text role="heading">비용</text><text>-12%</text></col>
    </grid>
    <row gap="md">
      <ul basis="480">
        <li>측정 파이프라인 <strong>안정화</strong></li>
        <li>리뷰 갤러리 자동화</li>
      </ul>
      <col grow="1"></col>
    </row>
  </col>
</slide>`,
		// 프리셋 도형 12종 — `prstGeom` 이름이 틀리거나 `avLst` 문법이 어긋나면 여기서 처음
		// 소리가 난다 (골든·단위 테스트는 초록인 채 사용자만 복구 경고를 본다). 조정값이
		// 실리는 프리셋 여섯이 이 슬라이드에 다 있다 — 합성 지오메트리가 모든 도형에 같은
		// rect를 주므로(1152×67.2) `ss = h`이고, 그래서 w 기준 조정값이 프리셋 기본값보다
		// 훨씬 큰 값으로 계산된다: 상한을 넘는 조정값이 스키마 유효성을 깨지 않는지도 함께
		// 지난다.
		`<slide>
  <row gap="md">
    <shape type="rect" w="120" h="80"/>
    <shape type="rounded-rect" w="120" h="80"/>
    <shape type="ellipse" w="120" h="80"/>
    <shape type="diamond" w="120" h="80"/>
    <shape type="cylinder" w="120" h="80"/>
    <shape type="chevron" w="120" h="80"/>
  </row>
  <row gap="md">
    <shape type="arrow-right" w="120" h="80"/>
    <shape type="hexagon" w="120" h="80"/>
    <shape type="parallelogram" w="120" h="80"/>
    <shape type="callout" w="120" h="80"/>
    <shape type="star" w="120" h="80"/>
    <shape type="cloud" w="120" h="80"/>
  </row>
</slide>`,
		// 도형 외관 + 도형 안 텍스트 — `xfrm@rot`/`flipH`/`flipV`, `effectLst > outerShdw`,
		// 색 안의 `a:alpha`, `bodyPr`의 인셋·앵커, 그리고 문단 여럿이 든 도형 하나가
		// 복구 경고 없이 열리는지. `<a>`가 든 도형은 도형 sp의 txBody에서도 슬라이드
		// rels의 External 관계가 살아나는지를 함께 지난다 (도형 본문의 첫 하이퍼링크다).
		`<slide>
  <row gap="lg" align="stretch">
    <shape type="rounded-rect" fill="accent1" border="2 dashed muted" radius="lg" shadow="soft" opacity="0.4" rotate="-45" flip="hv" w="300" h="180" pad="md">
      <h3 color="bg">회전·대칭·그림자</h3>
      <p color="bg" anchor="center">알파는 채움·선·그림자 색 안에 산다.</p>
    </shape>
    <shape type="rect" fill="none" border="3 dotted accent1" w="300" h="180" pad="lg">
      <p anchor="bottom">상세는 <a href="https://example.com/shape">도형 문서</a> 참고<br/>이어짐 줄</p>
    </shape>
    <shape type="chevron" fill="muted" w="180" h="180"/>
  </row>
</slide>`,
	}
	for i, src := range sources {
		file := fmt.Sprintf("slides/%03d.slide", i+1)
		doc, diags := soml.Parse(file, src)
		if err := diags.Err(); err != nil {
			t.Fatalf("%s 파싱 실패: %v", file, err)
		}
		deck.Slides = append(deck.Slides, &soml.Slide{
			Index: i, ID: fmt.Sprintf("s%d", i+1), Source: file,
			Notes: doc.Notes, Content: doc.Content, Tokens: doc.Tokens,
		})
	}
	if err := core.Resolve(deck).Err(); err != nil {
		t.Fatalf("외관 해석 실패: %v", err)
	}

	// 합성 지오메트리로 충분하다 — 이 테스트의 관심은 좌표 값이 아니라 XML의 스키마
	// 유효성이다 (좌표 정확성은 픽셀 비교가 본다, 항목 07).
	g := geom.Synthetic(deck, geom.Rect{X: 64, Y: 64, W: 1152, H: 67.2})
	files, err := pptx.NewWithGeometry(g).Render(deck)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), pptx.FileName(deck))
	if err := os.WriteFile(path, files[pptx.FileName(deck)], 0o644); err != nil {
		t.Fatal(err)
	}
	testkit.OpenPPTX(t, path)
}

// LibreOffice가 겹침(stack)과 음수·슬라이드 밖 좌표가 든 덱을 경고·복구 없이 여는지 —
// 하네스가 stderr의 복구·경고를 실패로 보므로 음수 `a:off`나 스테이지 밖 도형이 뷰어를
// 거스르면 여기서 드러난다 (골든·단위 테스트는 초록인 채 사용자만 경고를 본다).
//
// 항목 05는 이 경로에 새 코드를 더하지 않는다 — 문서 순서 = z-순서는 레이아웃 컨테이너
// 분기(도형 0개 + 자식 재귀)로 자동으로 맞고, 음수 좌표는 Offset이 그대로 흘려보낸다. 그래서
// 이 테스트의 일은 "자동으로 맞을 것이 실제 뷰어에서도 맞는지" 확인하는 것이다: 배경 카드
// 위에 배지가 겹치고(같은 좌표), 한 배지는 슬라이드 왼쪽 위 밖으로 반쯤 걸친다(음수 x/y).
func TestLibreOfficeOpensStackOverlapAndNegativeCoords(t *testing.T) {
	testkit.RequireSoffice(t)

	stage, err := soml.Aspect16x9.Stage()
	if err != nil {
		t.Fatal(err)
	}
	src := `<slide>
  <stack align="center" justify="center">
    <shape type="rect" fill="accent1" w="600" h="360"/>
    <text color="bg" at="center">가운데 배지</text>
    <text role="caption" color="bg" x="-40" y="-24">모서리 밖으로 걸친 배지</text>
  </stack>
</slide>`
	doc, diags := soml.Parse("slides/001.slide", src)
	if err := diags.Err(); err != nil {
		t.Fatalf("파싱 실패: %v", err)
	}
	deck := &soml.Deck{Title: "stack open test", Aspect: soml.Aspect16x9, Stage: stage}
	deck.Slides = append(deck.Slides, &soml.Slide{
		Index: 0, ID: "s1", Source: "slides/001.slide",
		Notes: doc.Notes, Content: doc.Content, Tokens: doc.Tokens,
	})
	if err := core.Resolve(deck).Err(); err != nil {
		t.Fatalf("외관 해석 실패: %v", err)
	}

	// 겹침·음수 좌표를 실제로 담은 지오메트리 — 합성(모든 요소 같은 rect)은 음수·스테이지
	// 밖을 만들 수 없다. refs 순서: stack, shape, text, text.
	refs := geom.Refs(deck)
	rects := []geom.Rect{
		{X: 340, Y: 180, W: 600, H: 360}, // stack (컨테이너)
		{X: 340, Y: 180, W: 600, H: 360}, // shape (배경 카드)
		{X: 520, Y: 340, W: 240, H: 40},  // text  (가운데 — 카드 위에 겹침)
		{X: -40, Y: -24, W: 320, H: 40},  // text  (슬라이드 왼쪽 위 밖으로 걸침)
	}
	if len(refs) != len(rects) {
		t.Fatalf("refs %d개, rects %d개 — 측정 대상 목록이 바뀌었다", len(refs), len(rects))
	}
	boxes := make([]geom.Box, len(refs))
	for i, ref := range refs {
		boxes[i] = geom.Box{ID: ref.ID, El: ref.El, Slide: ref.Slide, Rect: rects[i]}
	}
	g := &geom.Geometry{Stage: deck.Stage, Boxes: boxes}

	files, err := pptx.NewWithGeometry(g).Render(deck)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), pptx.FileName(deck))
	if err := os.WriteFile(path, files[pptx.FileName(deck)], 0o644); err != nil {
		t.Fatal(err)
	}
	testkit.OpenPPTX(t, path)
}
