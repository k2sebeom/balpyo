package core

import "github.com/k2sebeom/balpyo/soml"

// 이 파일은 외관 캐스케이드(PRD §8.3)의 병합 기계다.
//
//	인라인 속성 > variant > 컴포넌트 기본값 > 역할/토큰
//
// 역할 기본값(완전한 외관, theme.go)을 바닥에 깔고, 위 세 단은 **부분 패치**로 얹는다.
// variant와 컴포넌트 기본값은 일부 필드만 정하기 때문이다 (PRD §8.3의 components.json —
// `"card-header": { "font": "heading", "size": 24 }`는 color를 건드리지 않는다).
// Iteration 4의 도형 외관(fill/border …)이 같은 기계를 재사용한다.

// textPatch는 텍스트 외관의 부분 지정이다. 포인터로 "정하지 않음"(nil)과 0값을 구별한다 —
// 완성된 TextStyle끼리 덮어쓰면 "weight를 말하지 않은 variant"가 weight 0을 정한 것이 되어
// 아래 단의 값을 지운다.
//
// Role은 패치에 없다: 역할은 외관 값이 아니라 캐스케이드의 바닥을 **고르는** 축이다
// (textRoleDefaults). variant가 역할을 바꿀 수 있으면 "무엇의 기본값 위에 얹는가"가
// 병합 도중에 바뀌어 순서가 정의되지 않는다.
type textPatch struct {
	font       *soml.FontRole
	size       *float64
	weight     *int
	lineHeight *float64
	color      *soml.Color
	align      *soml.TextAlign
}

// mergeText는 patch가 정한 필드만 style에 얹는다.
//
// 캐스케이드의 **유일한** 병합 지점이다 — 단마다 병합 코드가 따로 있으면 단 사이의
// 우선순위가 코드 곳곳에 흩어져, 어긋난 순간을 찾을 수 없다. 우선순위는 이 함수를
// 부르는 순서(resolveText)가 전부다: 낮은 단부터 높은 단으로, 한 방향으로만.
func mergeText(style *soml.TextStyle, patch textPatch) {
	if patch.font != nil {
		style.Font = *patch.font
	}
	if patch.size != nil {
		style.Size = *patch.size
	}
	if patch.weight != nil {
		style.Weight = *patch.weight
	}
	if patch.lineHeight != nil {
		style.LineHeight = *patch.lineHeight
	}
	if patch.color != nil {
		style.Color = *patch.color
	}
	if patch.align != nil {
		style.Align = *patch.align
	}
}

// ptr는 패치 필드용 포인터 리터럴이다 — 하드코딩 테마 표(theme.go)가 값을 한 줄로 적기 위한
// 최소 도구이고, Iteration 10의 테마 패키지 로더가 이 표를 대체하면 함께 사라질 수 있다.
func ptr[T any](v T) *T { return &v }
