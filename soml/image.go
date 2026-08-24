package soml

// 이 파일은 이미지(PRD §7.5·§7.6)의 **요소 종류·속성·값 어휘 카탈로그**다.
//
// 서브셋(subset.go)·레이아웃(layout.go)·도형(shape.go) 카탈로그와 같은 자리에 같은 태도로
// 둔다: 이 표를 파서(문법), 값 층(core의 `src` 해석·`fit` 검사·외관), 지오메트리 패스(잴
// 대상), 웹 렌더러(`<img>`), PPTX 렌더러(`p:pic` + 미디어 파트)가 전부 소비한다. 다섯 층이
// 각자 이름 목록을 들고 있으면 속성 하나를 추가할 때 한 곳을 빼먹는 것이 곧 "파싱은 되는데
// 그려지지 않는" 조용한 실패다.
//
// **다른 어휘와 갈라지는 하나: 값이 파일을 가리킨다.** 그래서 이 파일이 여는 것은 이름과
// 문법뿐이다 — `src`의 경로 규칙·존재 여부·포맷·고유 크기와 `fit`의 값 검사는 전부 값
// 층(core)의 몫이고, 파서는 `<shape type>`에서와 같이 문법만 본다 (Element.Attrs의 주석).
//
// `<arrow>`·`<line>`·`<group>`은 여기 없다 — Iteration 7·9다. 렌더 매핑 없는 태그를 어휘에
// 열면 저자가 쓴 것이 조용히 무시되고, 그것이 닫힌 어휘가 막으려던 실패다.

// ImageElement는 이미지 요소 이름이다. TextElement·ShapeElement와 같은 이유로 상수다 —
// 이름 문자열이 여러 층에 흩어지면 오타 하나가 조용한 실패가 된다.
const ImageElement = "image"

// ImageFit은 `<image fit>`의 값 어휘 2종이다 (PRD §7.5의 `fit(cover/contain)`).
//
// 타입을 두는 이유는 BorderStyle·Flip과 같다: 확정 값이 될 자리가 있는 어휘이고(값 층의
// `ImageStyle`), 문자열 리터럴로 두면 렌더러가 `"cover"`를 다시 적는다.
//
// 어휘를 둘로 닫은 기준도 같다 — **양 타깃에 대응이 있는 것만**이다. CSS `object-fit`의
// `fill`(비율 무시 늘리기)·`scale-down`·`none`은 PPTX `blipFill`에서 각각 다른 조합
// (`stretch` + `srcRect` 없음 / 고유 크기 비교 / 크롭 없이 넘침)이 되고, 그중 `none`은
// `pic`의 박스 밖으로 그림이 넘치는 상태라 PPTX에 대응이 아예 없다. 둘로 닫으면 저자가
// 고칠 방법이 늘 있다: 늘리고 싶으면 `ratio`로 박스를 그림 비율에 맞춘다.
type ImageFit string

const (
	// FitCover는 박스를 채우고 넘치는 쪽을 자른다 (CSS `object-fit: cover`,
	// PPTX `blipFill` + `srcRect` 크롭).
	FitCover ImageFit = "cover"
	// FitContain은 박스 안에 그림 전체가 들어가게 맞춘다 (CSS `object-fit: contain`).
	FitContain ImageFit = "contain"
)

// ImageFits는 맞춤 어휘 전체다 (순서 = PRD §7.5의 순서 = 진단·문서 순서). 목록은 soml에
// 있고 **검사는 core**다 — ShapeTypes·Theme.ColorNames와 같은 구조다.
func ImageFits() []ImageFit {
	return []ImageFit{FitCover, FitContain}
}

// 이미지 속성 (PRD §7.5·§7.6 표의 image 행). 넷으로 나눈 이유는 도형과 같다 — 소비자가
// 다르다: 고유 속성은 파서와 **에셋 해석**이, 외관은 캐스케이드가, 박스 속성은 도형·레이아웃
// 컨테이너와 **공유**한다 (BoxAttrs). 이어 붙인 순서가 곧 진단 메시지와 문서의 순서다.
var (
	// imageOwnAttrs는 이미지만의 속성이다.
	//
	// `alt`은 PRD의 표에 없지만 **연다**. 기준은 이 저장소가 속성을 여는 기준 그 자체다 —
	// 양 타깃에 실을 자리가 있다: 웹은 `<img alt>`, PPTX는 `p:cNvPr@descr`(PowerPoint의
	// "대체 텍스트" 칸). 자리가 있는데 열지 않으면 산출물이 접근성 없는 그림을 내고 저자가
	// 그것을 고칠 방법이 아예 없다. 생략했을 때 웹이 `alt=""`(장식 이미지)를 내는 것까지가
	// 이 결정의 일부다 — `<img>`에서 속성이 없는 것과 빈 문자열은 스크린 리더에 다른 뜻이다.
	//
	// PRD §7.5 표의 `frame`은 **열지 않는다**. 뜻이 PRD 어디에도 정의되어 있지 않고(§7.6의
	// 속성 표에 행이 없다), 액자가 뜻하는 것(테두리+반경+그림자의 묶음)은 이미 `variant`가
	// 소유하는 채널이다 — 같은 것을 두 이름으로 받으면 캐스케이드에 5단이 생기고 `variant`와
	// `frame`이 충돌할 때 누가 이기는지를 정해야 하는데, 그것은 저자가 외울 이유가 없는
	// 규칙이다. `<image variant="framed">`가 그 자리다 (테마가 정의하는 번들 — 항목 02).
	imageOwnAttrs = []string{"src", "fit", "alt"}
	// imageAppearanceAttrs는 외관 채널이다 (PRD §7.6 표에서 적용 대상에 image가 있는 행).
	//
	// **`fill`은 열지 않는다** — PRD §7.6 표의 `fill` 적용 대상에 image가 없고, 그림 뒤의
	// 칠이 보이는 자리는 `fit="contain"`의 여백뿐인데 그 여백은 PPTX에서 `pic`의 박스 밖이다.
	// 색을 받아 놓고 한 타깃에서만 그리는 속성이 된다. 그림 뒤에 칠이 필요하면 그것은
	// 도형의 일이다 (`<shape fill>` 위에 그림을 겹치는 것은 Iteration 6의 `<stack>`).
	// `stroke`/`cap`/`dash`도 line·arrow의 것이라 같은 이유로 닫혀 있다 (shape.go와 같다).
	imageAppearanceAttrs = []string{"variant", "border", "radius", "shadow", "opacity", "rotate", "flip"}
)

// ImageAttrs는 `<image>`의 고유·외관·박스 속성이다 (자식 배치 속성은 제외 — 그것은
// withChildPlacement가 잇는다).
//
// 내보내는 이유는 ShapeAttrs·ContainerAttrs와 같다: 값 층(core)이 이 목록을 소비해 "파서가
// 연 속성 = 값 층이 해석하는 속성"을 유지한다.
//
// **이미지에 크기가 필요한 이유**는 도형과 같고 더 강하다: `fit`은 **박스가 정해진 다음에**
// 뜻을 갖는다 (`cover`는 "박스를 채우고 넘치는 쪽을 자른다"이므로 박스가 없으면 자를 것이
// 없고, 크기 속성이 없는 그림의 박스는 고유 크기라 `fit`이 아무것도 하지 않는다).
// `ratio="16/9"`가 이미지에서 가장 흔한 표기가 되는 자리다.
//
// `pad`를 그대로 받는다 (BoxAttrs 한 벌) — 그림에서의 뜻은 **그림 영역이 여백만큼
// 줄어드는 것**이고 양 타깃에 자리가 있다: 웹은 `<img>`의 padding이 콘텐츠 박스를 줄이고
// (`object-fit`은 콘텐츠 박스에 적용된다), PPTX는 `pic`의 `xfrm`을 여백만큼 줄여서 낸다
// (`bodyPr`의 인셋은 텍스트의 것이므로 쓰지 않는다). 목록을 도형과 나누지 않은 이유는
// 목록이 두 벌이 되면 PRD 표에 크기 속성이 하나 늘 때 어느 태그가 빠졌는지 사람이 세어야
// 하기 때문이다. 항목 03·04가 두 타깃의 계산을 실제로 맞추지 못하면 그때 `<image>`용
// 목록을 따로 이어 `pad`만 뗀다 (그 편이 지금 닫아 두고 뒤에 여는 것보다 값싸다: 여백 없는
// 그림은 저자가 `<row gap>`으로 흉내 낼 수 있지만, 목록이 갈라지면 다시 합칠 근거를 잃는다).
func ImageAttrs() []string {
	box := BoxAttrs()
	attrs := make([]string, 0, len(imageOwnAttrs)+len(imageAppearanceAttrs)+len(box))
	attrs = append(attrs, imageOwnAttrs...)
	attrs = append(attrs, imageAppearanceAttrs...)
	return append(attrs, box...)
}

// ImageAppearanceAttrs는 이미지 외관 속성 목록이다 (순서 = 진단·문서 순서).
// ShapeAppearanceAttrs와 같은 이유로 내보낸다 — 캐스케이드가 이 목록의 속성만 해석한다.
func ImageAppearanceAttrs() []string {
	attrs := make([]string, len(imageAppearanceAttrs))
	copy(attrs, imageAppearanceAttrs)
	return attrs
}
