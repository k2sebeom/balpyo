package core

import "github.com/k2sebeom/balpyo/soml"

// Resolve는 덱 모델의 외관을 해석한다 — 렌더러가 보기 전에 모든 외관 값이 확정되어야
// 한다는 계약을 이 함수가 진다.
//
// 해석 순서는 4단 캐스케이드다 (PRD §8.3):
//
//	인라인 속성 > variant > 컴포넌트 기본값 > 역할/토큰
//
// 이 순서를 렌더러가 아니라 여기서 한 번만 적용하기 때문에 웹과 PPTX가 같은 외관을 본다.
// computed-style 역추론은 없다 — 외관은 스키마에서 양 타깃으로 순방향 컴파일된다.
//
// Iteration 0에서는 통과만 한다. 해석할 외관 속성이 아직 없다 (첫 속성은 Iteration 1의
// role/size/color/align, 캐스케이드 완성은 Iteration 2).
func Resolve(deck *soml.Deck) soml.Diagnostics {
	_ = deck
	return nil
}
