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
// 요구하는 것들을 담는다 — 한글, &, 인라인 외관, 빈 요소, 그리고 노트. 노트는 **2번
// 슬라이드에만** 있다: notesSlide2가 notesSlide1 없이 존재하는, 번호가 비어 뛰는 경우다
// (notesSlidePart의 번호 규칙) — 노트 파트는 관계·Content_Types가 조금만 틀려도 복구
// 경고로 드러나므로 이 게이트가 여기서 특히 값을 한다.
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
