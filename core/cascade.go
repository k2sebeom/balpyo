package core

import "github.com/k2sebeom/balpyo/soml"

// 이 파일은 외관 캐스케이드(PRD §8.3)의 병합 기계다.
//
//	인라인 속성 > variant > 컴포넌트 기본값 > 역할/토큰
//
// 역할 기본값(완전한 외관, theme.go)을 바닥에 깔고, 위 세 단은 **부분 패치**로 얹는다.
// variant와 컴포넌트 기본값은 일부 필드만 정하기 때문이다 (PRD §8.3의 components.json —
// `"card-header": { "font": "heading", "size": 24 }`는 color를 건드리지 않는다).
//
// 채널이 둘이다 (텍스트 외관·도형 외관). 두 채널은 **같은 기계를 복제해서** 쓴다
// (textPatch/mergeText ↔ shapePatch/mergeShape): 필드가 겹치지 않으므로 하나로 합칠 수는
// 없지만, 병합 규칙과 우선순위의 표현 방식은 하나여야 한다. 도형이 바닥을 고르는 방식만
// 갈라진다 — 텍스트에는 역할이 있고 도형에는 없어서 컴포넌트 기본값이 가장 아래 단이다
// (shapeComponentDefault).

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
	anchor     *soml.TextAnchor
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
	if patch.anchor != nil {
		style.Anchor = *patch.anchor
	}
}

// ptr는 패치 필드용 포인터 리터럴이다 — 하드코딩 테마 표(theme.go)가 값을 한 줄로 적기 위한
// 최소 도구이고, Iteration 10의 테마 패키지 로더가 이 표를 대체하면 함께 사라질 수 있다.
func ptr[T any](v T) *T { return &v }

// shapePatch는 도형 외관의 부분 지정이다 — textPatch의 도형 판이고 규약이 같다: 포인터로
// "정하지 않음"(nil)과 0값을 구별한다 (`opacity="0"`은 저자가 정한 0이고 "opacity를 말하지
// 않음"과 다르다).
//
// **Type·ID·Box는 없다.** Type은 외관 값이 아니라 어느 프리셋을 그리는가이고(variant가
// 종류를 바꿀 수 있으면 저자가 쓴 `type`이 테마에 따라 다른 그림이 된다 — textPatch에 Role이
// 없는 것과 같은 자리의 결정), ID는 참조 이름이라 테마가 붙일 수 없고, Box는 배치 채널이다
// (테마가 배치를 담지 않는다 — Iteration 3이 닫아 둔 질문, PRD §7.4).
type shapePatch struct {
	fill    *soml.Fill
	border  *soml.Border
	radius  *soml.Radius
	shadow  *soml.ShadowToken
	opacity *float64
	rotate  *float64
	flip    *soml.Flip
}

// mergeShape는 patch가 정한 필드만 style에 얹는다 — mergeText와 **같은 계약**이다.
//
// 도형 캐스케이드의 유일한 병합 지점이고, 우선순위는 이 함수를 부르는 순서(resolveShape)가
// 전부다. 두 번째 채널이 자기 병합 규칙을 새로 쓰면 "우선순위는 한 곳에만 있다"는 계약이 그
// 순간 사라진다 — 그래서 도형은 새 메커니즘을 만들지 않고 이 형태를 복제한다.
//
// 두 패치 타입의 공통을 제네릭으로 뽑는 길을 저울질했고 버렸다: 필드가 하나도 겹치지 않아
// (텍스트는 폰트·크기·색, 도형은 채움·테두리·그림자) 뽑아낼 것이 "포인터가 nil이 아니면
// 대입한다"는 한 줄뿐이고, 그걸 위해 필드마다 접근자나 리플렉션을 끼우면 얻는 것 없이 읽기가
// 어려워진다. 병합이 **한눈에 보이는 대입 목록**인 것이 이 파일의 값이다.
func mergeShape(style *soml.ShapeStyle, patch shapePatch) {
	// 포인터를 그대로 대입하지 않고 **값을 복사한다**: 지금은 테마 표가 호출마다 새로 만들어져
	// 차이가 없지만, 표가 파일에서 한 번 읽혀 캐시되는 날(Iteration 10) 여러 도형이 같은
	// Fill을 가리키게 되고 한 도형을 고치는 코드가 테마를 고친다.
	if patch.fill != nil {
		style.Fill = ptr(*patch.fill)
	}
	if patch.border != nil {
		style.Border = ptr(*patch.border)
	}
	if patch.radius != nil {
		style.Radius = ptr(*patch.radius)
	}
	if patch.shadow != nil {
		style.Shadow = ptr(*patch.shadow)
	}
	if patch.opacity != nil {
		style.Opacity = ptr(*patch.opacity)
	}
	if patch.rotate != nil {
		style.Rotate = ptr(*patch.rotate)
	}
	if patch.flip != nil {
		style.Flip = *patch.flip
	}
}

// cellPatch는 표 셀 외관의 부분 지정이다 — textPatch·shapePatch의 셀 판이고 규약이 같다:
// 포인터로 "정하지 않음"(nil)과 0값을 구별한다. 한 셀이 박스(fill·border)와 텍스트(color·
// align·anchor·weight)를 함께 지므로 두 채널의 필드를 한 패치에 담는다 — 셀은 `a:tc`가
// `tcPr`와 `txBody`를 함께 갖듯 두 채널이 한 요소에 산다 (soml.CellStyle의 근거).
//
// **ColSpan·RowSpan은 패치에 없다.** 병합 칸 수는 외관 값이 아니라 격자 구조이고 테마가
// 정할 수 없으므로(식별자·배치가 shapePatch에 없는 것과 같은 자리의 결정), 캐스케이드가
// 아니라 확정 CellStyle에 직접 채운다 (cellAttrs). "패치의 필드 = 캐스케이드가 다루는 필드"라는
// 읽는 근거를 지킨다.
type cellPatch struct {
	fill   *soml.Fill
	border *soml.Border
	color  *soml.Color
	align  *soml.TextAlign
	anchor *soml.TextAnchor
	weight *int
}

// mergeCell은 patch가 정한 필드만 style에 얹는다 — mergeText·mergeShape와 **같은 계약**이다.
//
// 네 번째 복제다 (텍스트·도형·이미지에 이어). 두 채널을 한 패치가 담으므로 대입도 두 갈래다:
// 박스 필드(fill·border)는 CellStyle에, 텍스트 필드(color·align·anchor·weight)는 CellStyle.Text에
// 얹는다. style.Text는 호출자가 역할 기본값으로 이미 채워 둔다(resolveCell) — 그래서 여기서
// nil 검사 없이 필드에 바로 대입한다 (셀은 텍스트 전용 콘텐츠 모델이라 언제나 Text를 갖는다).
//
// 박스 포인터는 값을 복사한다 (mergeShape와 같은 근거 — 테마 표가 캐시되는 날 한 셀을 고치는
// 코드가 테마를 고친다).
func mergeCell(style *soml.CellStyle, patch cellPatch) {
	if patch.fill != nil {
		style.Fill = ptr(*patch.fill)
	}
	if patch.border != nil {
		style.Border = ptr(*patch.border)
	}
	if patch.color != nil {
		style.Text.Color = *patch.color
	}
	if patch.align != nil {
		style.Text.Align = *patch.align
	}
	if patch.anchor != nil {
		style.Text.Anchor = *patch.anchor
	}
	if patch.weight != nil {
		style.Text.Weight = *patch.weight
	}
}

// imagePatch는 이미지 외관의 부분 지정이다 — shapePatch의 이미지 판이고 규약이 같다.
//
// **fill이 없다**: PRD §7.6 표의 `fill` 적용 대상에 image가 없고, 그림 뒤의 칠이 보이는
// 자리는 `fit="contain"`의 여백뿐인데 그 여백은 PPTX에서 `pic`의 박스 밖이다 (soml/image.go의
// 근거). shapePatch를 그대로 재사용하면 테마 표가 그림에 fill을 정할 수 있게 되고, 그 값은
// 웹에서만 그려지거나 조용히 버려진다 — 어느 쪽이든 표를 읽는 사람이 뜻을 알 수 없다.
//
// **Src·Fit·Alt·Box는 없다.** Src는 어느 파일인가이고(테마가 정할 수 있으면 저자가 쓴 그림이
// 테마에 따라 다른 파일이 된다 — textPatch에 Role이, shapePatch에 Type이 없는 것과 같은
// 자리의 결정), Fit은 그림의 **의미**이지 장식이 아니라 언어가 기본값을 갖고
// (soml.DefaultImageFit), Alt은 콘텐츠이고, Box는 배치 채널이다.
type imagePatch struct {
	border  *soml.Border
	radius  *soml.Radius
	shadow  *soml.ShadowToken
	opacity *float64
	rotate  *float64
	flip    *soml.Flip
}

// mergeImage는 patch가 정한 필드만 style에 얹는다 — mergeText·mergeShape와 **같은 계약**이다.
//
// 세 번째 복제다. 제네릭으로 합치는 길을 여기서 다시 저울질했고 또 버렸다: 이미지 패치가
// 도형 패치의 부분집합(fill 하나 차이)이라 "도형 패치를 재사용하고 fill만 버린다"가 가능해
// 보이지만, 그러면 **버리는 코드**가 생기고 그 코드가 곧 "테마가 정할 수 있는 것"의 두 번째
// 정의가 된다 (imagePatch의 필드 목록이 첫 번째다). 병합이 한눈에 보이는 대입 목록인 것이
// 이 파일의 값이고, 구조를 뽑는 것은 네 번째 채널(`<group>` — Iteration 9)에서 한다.
func mergeImage(style *soml.ImageStyle, patch imagePatch) {
	// 포인터를 그대로 대입하지 않고 값을 복사한다 (mergeShape와 같은 근거 — 테마 표가
	// 캐시되는 날 한 그림을 고치는 코드가 테마를 고친다).
	if patch.border != nil {
		style.Border = ptr(*patch.border)
	}
	if patch.radius != nil {
		style.Radius = ptr(*patch.radius)
	}
	if patch.shadow != nil {
		style.Shadow = ptr(*patch.shadow)
	}
	if patch.opacity != nil {
		style.Opacity = ptr(*patch.opacity)
	}
	if patch.rotate != nil {
		style.Rotate = ptr(*patch.rotate)
	}
	if patch.flip != nil {
		style.Flip = *patch.flip
	}
}
