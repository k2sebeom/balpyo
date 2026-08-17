// Package pptx는 덱 모델을 편집 가능한 네이티브 PPTX로 조립한다.
//
// 전략은 두 갈래다 (PRD §9.3):
//
//	① 지오메트리 패스 — 웹 렌더러와 같은 HTML/CSS를 헤드리스 브라우저에 올려 요소별
//	   rect만 읽고 EMU로 바꾼다. 읽는 것은 좌표뿐이다.
//	② 외관 직행 — 해석된 외관 스키마를 OOXML 속성으로 순방향 컴파일한다. 토큰 색은
//	   schemeClr로 보존되어 산출물이 PowerPoint 안의 테마 색 편집을 따라간다.
//
// computed-style 역추론은 없다. Iteration 0은 ②의 뼈대(테마·마스터·레이아웃 파트가 있는
// 열리는 패키지)만 세우고, 도형은 Iteration 1부터 spTree에 들어간다.
package pptx

import (
	"archive/zip"
	"bytes"
	"fmt"
	"time"

	"github.com/k2sebeom/balpyo/render"
	"github.com/k2sebeom/balpyo/soml"
)

// zipEpoch는 모든 zip 엔트리에 쓰는 고정 수정 시각이다.
//
// 빌드 시각을 넣으면 같은 입력이 매번 다른 바이트를 내고 골든 비교가 불가능해진다.
// 1980-01-01은 zip 포맷이 표현할 수 있는 가장 이른 시각이다.
var zipEpoch = time.Date(1980, 1, 1, 0, 0, 0, 0, time.UTC)

// Renderer는 PPTX 타깃 렌더러다.
type Renderer struct{}

// New는 PPTX 렌더러를 만든다.
func New() *Renderer { return &Renderer{} }

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
func (*Renderer) Render(deck *soml.Deck) (render.Files, error) {
	if deck == nil {
		return nil, fmt.Errorf("덱이 nil이다")
	}
	if len(deck.Slides) == 0 {
		// 슬라이드가 없는 프레젠테이션은 PowerPoint가 열지 못한다.
		return nil, fmt.Errorf("슬라이드가 없다")
	}
	parts, err := buildParts(deck)
	if err != nil {
		return nil, err
	}
	data, err := zipParts(parts)
	if err != nil {
		return nil, err
	}
	return render.Files{FileName(deck): data}, nil
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
