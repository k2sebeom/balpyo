package geom

import "github.com/k2sebeom/balpyo/soml"

// Synthetic은 브라우저 없이 만든 지오메트리다 — 덱의 모든 요소에 같은 rect를 붙인다.
//
// **빌드 경로에서 쓰면 안 된다.** 값은 잰 것이 아니므로 여기서 나온 좌표로 만든 산출물은
// 배치가 뜻을 갖지 않는다.
//
// 그런데도 export하는 이유: 좌표를 **소비**하는 쪽(render/pptx의 도형 조립, Iteration 1의
// item 5)은 테스트에서 브라우저가 필요하지 않다. 그 테스트가 브라우저를 띄우면 두 가지를
// 잃는다 — 로컬에서 브라우저 없이 돌아가는 테스트가 줄고, 조립 로직의 실패와 측정의 실패가
// 한 테스트에서 섞인다. 소비 측은 "이 좌표가 이 XML이 된다"만 보면 된다.
//
// 나가는 값은 실제 측정과 같은 검사를 통과한 것이다 (같은 대조 경로를 쓴다) — 그래야 소비
// 측이 테스트에서 본 지오메트리와 빌드에서 받는 지오메트리가 같은 계약을 갖는다.
func Synthetic(deck *soml.Deck, rect Rect) *Geometry {
	if deck == nil {
		return nil
	}
	refs := Refs(deck)
	boxes := make([]Box, 0, len(refs))
	for _, ref := range refs {
		boxes = append(boxes, Box{ID: ref.ID, El: ref.El, Slide: ref.Slide, Rect: rect})
	}
	// 콘텐츠 박스는 채우지 않는다 — 잰 값이 아니므로 여기서 만들면 "테마가 정한 여백"이라는
	// 뜻이 없는 숫자가 된다. 소비 측은 기본 여백으로 물러선다 (Geometry.ContentBox).
	return &Geometry{Stage: deck.Stage, Boxes: boxes, index: newIndex(boxes)}
}
