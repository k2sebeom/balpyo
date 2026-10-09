package core

import "github.com/k2sebeom/balpyo/soml"

// Validate는 덱 모델 전체에 걸친 규칙을 검사한다.
//
// 파서가 파일 하나 안에서 볼 수 있는 것(닫힌 어휘, 문법)은 파서가 진단한다. 여기서는
// **파일을 넘어야 알 수 있는 것**만 본다 — 지금은 슬라이드 ID 충돌이 그것이다.
// 다른 요소의 id를 가리키는 참조처럼 파일을 넘는 검사가 생기면 이 층에 붙는다.
//
// check·build·lsp가 이 함수를 공유하므로 에디터에서 본 진단과 CI가 막는 진단이 같다.
func Validate(deck *soml.Deck) soml.Diagnostics {
	var diags soml.Diagnostics

	// 슬라이드 ID는 딥링크 앵커다. 충돌하면 링크가 어느 슬라이드를 가리키는지 알 수 없다.
	seen := make(map[string]*soml.Slide, len(deck.Slides))
	for _, slide := range deck.Slides {
		if slide.ID == "" {
			diags = append(diags, soml.Errorf(slide.Source, soml.Span{},
				"슬라이드 ID가 비어 있다"))
			continue
		}
		if first, dup := seen[slide.ID]; dup {
			diags = append(diags, soml.Errorf(slide.Source, soml.Span{},
				"슬라이드 ID %q가 %s와 충돌한다 (id 속성으로 구분한다)", slide.ID, first.Source))
			continue
		}
		seen[slide.ID] = slide
	}
	return diags
}
