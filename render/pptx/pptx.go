// Package pptx는 덱 모델을 편집 가능한 네이티브 PPTX로 조립한다.
//
// 전략은 두 갈래다 (PRD §9.3):
//
//	① 지오메트리 패스 — 웹 렌더러와 같은 HTML/CSS를 헤드리스 브라우저에 올려 요소별
//	   rect만 읽고 EMU로 바꾼다. 읽는 것은 좌표뿐이다.
//	② 외관 직행 — 해석된 외관 스키마를 OOXML 속성으로 순방향 컴파일한다. 토큰 색은
//	   schemeClr로 보존되어 산출물이 PowerPoint 안의 테마 색 편집을 따라간다.
//
// computed-style 역추론은 없다. ①과 ②가 만나는 자리는 shape.go의 shapesXML이다 —
// `<text>` 하나가 p:sp(텍스트 박스) 하나가 되고, 좌표는 xfrm, 외관은 rPr/pPr가 된다.
// `<image>`는 p:sp가 아니라 **p:pic**이 되고, 그 그림의 바이트가 패키지의 미디어 파트로
// 나간다 — 파트·관계·콘텐츠 타입 세 자리가 함께 맞아야 하는 유일한 요소다 (image.go).
// 레이아웃 태그(`<row>`/`<col>`/`<grid>`/`<spacer>`)는 **도형이 되지 않는다**: 배치는 ①에서
// 좌표로 구워지므로 자식의 절대 좌표만 남고 컨테이너 자체는 사라진다 (writeShapeNode).
// 테마 파트(clrScheme/fontScheme)는 덱의 토큰 표에서 컴파일된다 (theme.go).
package pptx

import (
	"archive/zip"
	"bytes"
	"fmt"
	"time"

	"github.com/k2sebeom/balpyo/render"
	"github.com/k2sebeom/balpyo/render/geom"
	"github.com/k2sebeom/balpyo/soml"
)

// zipEpoch는 모든 zip 엔트리에 쓰는 고정 수정 시각이다.
//
// 빌드 시각을 넣으면 같은 입력이 매번 다른 바이트를 내고 골든 비교가 불가능해진다.
// 1980-01-01은 zip 포맷이 표현할 수 있는 가장 이른 시각이다.
var zipEpoch = time.Date(1980, 1, 1, 0, 0, 0, 0, time.UTC)

// Renderer는 PPTX 타깃 렌더러다.
//
// 지오메트리는 생성자로 받는다 — 렌더 중에 브라우저를 띄우지 않는다. 그래야 이 패키지가
// 브라우저를 모르는 상태로 남고(임포트 방향이 한쪽이다), 좌표가 언제 어떤 HTML에서 나왔는지가
// 호출자 쪽 코드에 드러난다. 좌표 의존이 타입에 없으면 "지오메트리 없이 조립된 PPTX"가
// 조용히 만들어진다.
type Renderer struct {
	geometry *geom.Geometry
}

// New는 지오메트리 없는 PPTX 렌더러를 만든다 — 좌표가 필요한 요소가 없는 덱만 렌더할 수 있다.
func New() *Renderer { return &Renderer{} }

// NewWithGeometry는 잰 좌표를 가진 PPTX 렌더러를 만든다.
func NewWithGeometry(g *geom.Geometry) *Renderer { return &Renderer{geometry: g} }

// Target은 render.TargetPPTX다.
func (*Renderer) Target() render.Target { return render.TargetPPTX }

// FileName은 덱 제목에서 산출 파일 이름을 만든다. 제목이 파일명으로 쓸 수 없으면
// `deck.pptx`로 떨어진다.
func FileName(deck *soml.Deck) string {
	if slug := soml.Slug(deck.Title); slug != "" {
		return slug + ".pptx"
	}
	return "deck.pptx"
}

// Render는 덱을 pptx 파일 하나로 조립한다.
func (r *Renderer) Render(deck *soml.Deck) (render.Files, error) {
	if deck == nil {
		return nil, fmt.Errorf("덱이 nil이다")
	}
	if len(deck.Slides) == 0 {
		// 슬라이드가 없는 프레젠테이션은 PowerPoint가 열지 못한다.
		return nil, fmt.Errorf("슬라이드가 없다")
	}
	if err := r.checkGeometry(deck); err != nil {
		return nil, err
	}
	parts, err := buildParts(deck, r.geometry)
	if err != nil {
		return nil, err
	}
	data, err := zipParts(parts)
	if err != nil {
		return nil, err
	}
	return render.Files{FileName(deck): data}, nil
}

// checkGeometry는 이 덱을 조립하는 데 필요한 좌표가 있는지 본다.
//
// 좌표가 필요한 요소가 하나라도 있으면 지오메트리를 요구한다. 없이 조립하면 도형이 빠지거나
// 전부 원점에 겹친 파일이 나오는데, 그 파일은 정상으로 열리므로 실패가 PowerPoint에서 눈으로
// 볼 때까지 드러나지 않는다. 빈 슬라이드만 있는 덱은 잰 좌표가 필요하지 않다 — 그래서
// 빈 슬라이드 픽스처는 브라우저 없이 돈다 (internal/pipeline이 같은 판정을 쓴다).
func (r *Renderer) checkGeometry(deck *soml.Deck) error {
	refs := geom.Refs(deck)
	if len(refs) == 0 {
		return nil
	}
	if r.geometry == nil {
		return fmt.Errorf("좌표가 필요한 요소가 %d개인데 지오메트리가 없다 (%s) — 지오메트리 패스를 거치지 않았다",
			len(refs), refs[0].Source)
	}
	// 잰 덱과 조립하는 덱이 다르면 좌표가 전부 엉뚱한 요소로 간다.
	return r.geometry.Verify(deck)
}

// zipParts는 파트 목록을 OPC 패키지(zip)로 묶는다.
//
// `[Content_Types].xml`이 첫 엔트리여야 한다 — 일부 소비자가 스트리밍으로 읽으며 이
// 파트를 먼저 기대한다. 나머지는 buildParts가 준 순서를 유지한다.
func zipParts(parts []part) ([]byte, error) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for _, p := range parts {
		w, err := zw.CreateHeader(&zip.FileHeader{
			Name:     p.name,
			Method:   zip.Deflate,
			Modified: zipEpoch,
		})
		if err != nil {
			return nil, fmt.Errorf("zip 엔트리 %s 생성 실패: %w", p.name, err)
		}
		if _, err := w.Write(p.data); err != nil {
			return nil, fmt.Errorf("zip 엔트리 %s 쓰기 실패: %w", p.name, err)
		}
	}
	if err := zw.Close(); err != nil {
		return nil, fmt.Errorf("zip 마무리 실패: %w", err)
	}
	return buf.Bytes(), nil
}
