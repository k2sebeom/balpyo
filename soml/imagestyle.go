package soml

// 이 파일은 **확정된 이미지 외관**이다 — 카탈로그(image.go)가 이름과 값 어휘를 소유하고
// 여기가 값의 모양을 소유한다 (shape.go ↔ shapestyle.go와 같은 나눔).
//
// Appearance·ShapeStyle과 같은 계약을 진다: **렌더러는 이 구조체만 읽고 Attrs를 다시
// 해석하지 않는다.** 값 문법(색 토큰·리터럴, border 축약, radius·shadow 토큰, 0~1 실수,
// 각도, 닫힌 어휘)은 core가 한 번만 적용하고, 그 해석 함수는 도형과 **같은 것**이다 —
// 같은 일을 하는 코드가 두 벌이 되면 웹의 CSS와 PPTX의 OOXML이 갈라지고, 그 종류의 버그는
// 두 산출물을 나란히 볼 때까지 드러나지 않는다.

// ImageStyle은 확정된 이미지 외관이다.
//
// **ShapeStyle과 공용 구조체를 뽑지 않는다**: 소비자가 둘일 때는 함수를 공유하고, 구조를
// 뽑는 것은 세 번째 소비자(`<group>`)가 생길 때 한다. 지금 뽑으면 도형에만 있는 것
// (`Type`·`Fill`·`ID`)과 이미지에만 있는 것(`Src`·`Fit`·`Alt`)의 경계를 두 번 정하게 된다.
//
// "정하지 않음"의 표현은 ShapeStyle과 같다 — 포인터의 nil과 닫힌 어휘의 빈 문자열
// (`opacity="1"`은 저자가 정한 1이고 "opacity를 말하지 않음"과 다르다).
type ImageStyle struct {
	// Src는 **프로젝트 루트 기준 정규화 경로**다 — Deck.Assets의 키다 (Asset.Key).
	//
	// 바이트와 고유 크기를 여기 담지 않는 이유: 같은 그림을 두 번 쓴 슬라이드에서 두 확정
	// 값이 자기 크기를 다르게 말할 수 있게 된다. 표가 그 사실의 유일한 출처이고
	// (`deck.Assets.Get(style.Src)`), 확정 값은 **어느 에셋인가**만 담는다.
	//
	// 해석에 실패한 `src`(없음·빈 값·프로젝트 밖·지원하지 않는 포맷)는 빈 문자열로 남는다 —
	// 틀린 값이 확정 외관에 실리지 않는다는 값 층 전체의 정책이고, 그 빈 값이 렌더러의
	// "그릴 것이 없다" 가드를 마지막 그물로 남긴다.
	Src string
	// Fit은 박스 안에서 그림을 어떻게 맞추는가다 (ImageFits 2종). 생략하면 `contain`이다 —
	// **언어의 기본값**이고 테마의 컴포넌트 기본값이 아니다 (DefaultImageFit의 주석).
	Fit ImageFit
	// Alt은 대체 텍스트다 (웹 `<img alt>`, PPTX `p:cNvPr@descr`). 생략과 `alt=""`는 같은
	// 값이다 — 둘 다 "장식 이미지"이고, `<img>`에서 그 뜻을 내는 표기가 `alt=""`다.
	Alt string
	// 아래 외관 여섯은 도형과 **같은 값 문법**을 지난다 (core의 같은 함수).
	Border *Border
	// Radius는 모서리 반경이다 — **이미지에서는 항상 유효하다.** 도형에서 `radius`가
	// `rounded-rect` 전용인 근거는 프리셋 기하에 반경 조정값 자리가 없다는 것이었고,
	// 그림은 프리셋이 아니라 자기 클립을 갖는다 (웹 `border-radius`, PPTX는
	// `pic/spPr/prstGeom prst="roundRect"`) — 그 제한의 적용 범위가 도형에 한정된다.
	Radius  *Radius
	Shadow  *ShadowToken
	Opacity *float64
	Rotate  *float64
	Flip    Flip
	// Box는 그림 자신의 박스다 — 안쪽 여백·크기·비율. 도형·레이아웃 컨테이너와 **같은
	// 구조체**를 공유한다 (Box의 주석). `fit`이 뜻을 갖는 것은 이 박스가 정해졌을 때뿐이다.
	Box
}

// ImageStyle은 이 요소의 확정 이미지 외관이다. 이미지가 아니거나 해석 전이면 nil이다 —
// ShapeStyle()·Container()와 같은 이유로 요소에 직접 둔다 (렌더러가 두 단계 nil 검사를
// 쓰지 않게).
func (e *Element) ImageStyle() *ImageStyle {
	if e.Appearance == nil {
		return nil
	}
	return e.Appearance.Image
}
