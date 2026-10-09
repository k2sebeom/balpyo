package soml

// 이 파일은 도형의 **요소 종류·속성·값 어휘 카탈로그**다.
//
// 서브셋(subset.go)·레이아웃(layout.go) 카탈로그와 같은 자리에 같은 태도로 둔다: 이 표를
// 파서(문법), 값 층(core의 `type`·색·길이 검사), 지오메트리 패스(잴 대상), 웹 렌더러(모양을
// 그린다), PPTX 렌더러(`prstGeom`)가 전부 소비한다. 다섯 층이 각자 이름 목록을 들고 있으면
// 태그 하나·프리셋 하나를 추가할 때 한 곳을 빼먹는 것이 곧 "파싱은 되는데 그려지지 않는"
// 조용한 실패다.
//
// `<arrow>`·`<line>`·`<group>`은 여기 없다 — 아직 렌더 매핑이 없다. `<image>`는
// 자기 카탈로그(image.go)에 있다. 렌더 매핑 없는 태그를 어휘에 열면 저자가 쓴 것이 조용히
// 무시되고, 그것이 닫힌 어휘가 막으려던 실패다.

// ShapeElement는 도형 요소 이름이다. TextElement와 같은 이유로 상수다 — 이름 문자열이
// 여러 층에 흩어지면 오타 하나가 조용한 실패가 된다.
const ShapeElement = "shape"

// shapeTypes는 `<shape type>`의 v1 값 어휘 12종이다. **순서가 진단·문서의
// 순서다** — 맵이 아니라 슬라이스인 이유이고, 모르는 `type` 진단이 나열하는
// 목록이 결정적이어야 한다 (Theme.ColorNames·textVariantNames와 같은 태도).
//
// 목록은 soml에 있고 **검사는 core**다: 파서는 문법과 이름만 보고 값의 의미는 값 층이
// 본다 (Element.Attrs 주석). 12종은 전부 OOXML 프리셋 지오메트리의 부분집합이며, 확장할
// 때도 OOXML에 존재하는 것만 더해 번역 가능성을 지킨다.
var shapeTypes = []string{
	"rect", "rounded-rect", "ellipse", "diamond", "cylinder", "chevron",
	"arrow-right", "hexagon", "parallelogram", "callout", "star", "cloud",
}

// ShapeTypes는 도형 프리셋 이름 목록이다 (순서 = 진단·문서 순서). 값 층과 렌더러가
// 소비하도록 복사본을 준다 — 카탈로그가 호출자에 의해 흔들리면 표를 한 곳에 둔 뜻이 없다.
func ShapeTypes() []string {
	types := make([]string, len(shapeTypes))
	copy(types, shapeTypes)
	return types
}

// shapeInsetTextTypes는 **텍스트 영역이 박스보다 좁은** 프리셋 8종이다 — 뷰어가
// `prstGeom`마다 들고 있는 본문 사각형이 실루엣 안쪽으로 백분율만큼 들어와 있는 종류다.
// 나머지 넷은 영역이 곧 박스이거나(rect·chevron·callout) 인셋이 px이다(rounded-rect).
//
// 수치가 아니라 **이름만** 여기 있는 이유는 층의 소유다: 백분율 수치는 기하 유도표의 한
// 열이고(render/web의 shapeGeometries — 우리가 그린 실루엣과 짝이어야 한다), 값 층이 알아야
// 하는 것은 수치가 아니라 "이 프리셋은 자기 박스를 기준으로 삼는다"는 사실 하나다. 값 층이
// 렌더러를 import하면 층이 뒤집히므로(core는 soml만 본다) 사실은 어휘 카탈로그에 둔다.
//
// 두 곳이 갈라지는 위험은 테스트가 막는다 — render/web이 이 목록과 자기 표의
// `TextRect.HasPercent()`가 같은지 단정한다 (프리셋을 더할 때 한쪽만 고치면 멈춘다).
var shapeInsetTextTypes = []string{
	"ellipse", "diamond", "cylinder", "arrow-right", "hexagon", "parallelogram", "star", "cloud",
}

// ShapeInsetText는 이 프리셋의 텍스트 영역이 박스보다 좁은지다 (shapeInsetTextTypes).
//
// 값 층은 이 사실로 **확정 높이를 요구한다**: 백분율 인셋은 자기 박스 높이를
// 기준으로 하므로 내용이 높이를 정하는 도형에서는 기준이 순환한다.
func ShapeInsetText(shapeType string) bool {
	for _, name := range shapeInsetTextTypes {
		if name == shapeType {
			return true
		}
	}
	return false
}

// BorderStyle은 `border` 축약의 스타일 조각이다 (`"1 solid muted"`).
//
// **이름은 CSS 쪽이다** (`dashed`/`dotted`, OOXML의 `dash`/`dot`이 아니다). 근거는 축약
// 자체가 CSS `border`의 표기이기 때문이다 — 저자가 아는 표기를 그대로 쓰는 것이 이
// 저장소의 결이고(표기를 새로 발명하지 않는다), CSS 모양의 축약 안에 OOXML 어휘를 섞으면
// 둘 다 아닌 제3의 표기가 된다.
//
// 어휘를 셋으로 닫은 기준은 **양 타깃에 대응이 있는 것만**이다: 셋은 OOXML `prstDash`의
// `solid`/`sysDash`/`sysDot`(계열 선택은 렌더 매핑의 몫)에 대응한다. CSS의
// `double`/`groove`/`ridge`는 `prstDash`에 없어 PPTX에서 조용히 실선이 되므로 열지 않는다.
type BorderStyle string

const (
	BorderSolid  BorderStyle = "solid"
	BorderDashed BorderStyle = "dashed"
	BorderDotted BorderStyle = "dotted"
)

// BorderStyles는 테두리 스타일 어휘 전체다 (순서 = 진단 메시지 순서 — 진단이 결정적이어야
// 하는 이유는 TextAligns와 같다).
func BorderStyles() []BorderStyle {
	return []BorderStyle{BorderSolid, BorderDashed, BorderDotted}
}

// Flip은 도형 대칭이다 (`flip="h"`).
//
// 값 셋을 닫은 근거는 OOXML `xfrm@flipH`/`flipV` 두 불리언이다 — 축이 둘이므로 뜻 있는
// 조합은 h·v·둘 다 셋뿐이고, "둘 다"의 표기를 `hv`로 정한 이유는 축약이 두 축을 쓴 순서를
// 그대로 읽히게 하는 것이다 (`h v` 공백 표기는 조각을 세는 새 문법을 만들고, `both`는
// 축이 셋 이상으로 늘 수 없는데도 축 이름을 지운다).
type Flip string

const (
	FlipH    Flip = "h"
	FlipV    Flip = "v"
	FlipBoth Flip = "hv"
)

// Flips는 대칭 어휘 전체다 (순서 = 진단 메시지 순서).
func Flips() []Flip {
	return []Flip{FlipH, FlipV, FlipBoth}
}

// 도형 속성. 넷으로 나눈 이유는 소비자가 다르기 때문이다:
// 고유 속성은 파서·참조 해석이, 외관은 캐스케이드가, 박스 속성은 레이아웃
// 컨테이너와 **공유**한다 (layout.go의 boxPadAttrs·boxSizeAttrs). 이어 붙인 순서가 곧
// 진단 메시지와 문서의 순서다.
var (
	// shapeOwnAttrs는 도형만의 속성이다.
	//
	// `id`는 산출물에 아무것도 그리지 않는다 — 그것이 정상이고 `<slide id>`의 선례가 이미
	// 그렇다 (파일명 정렬·진단에만 쓰인다). 도형 `id`는 `<arrow from="db">`의 **참조 대상
	// 이름**이며 참조 해석은 아직 없다.
	//
	// `alt`은 **그림의 이름**이다 — 도형 안 글을 대체하지 않는다. 그림 쪽 선례와
	// 같은 기준으로 열었다: 양 타깃에 실을 자리가 있다 (웹은 잉크 `<svg role="img">`,
	// PPTX는 `p:cNvPr@descr`). 외관이 아니므로 외관 목록이 아니라 여기 있다 — 테마도
	// `variant`도 얹지 않는다.
	shapeOwnAttrs = []string{"type", "id", "alt"}
	// shapeAppearanceAttrs는 외관 채널이다. `stroke`/`cap`/
	// `dash`는 line·arrow의 것이라 열지 않는다 — 도형에 받아 놓고 버리는 선언이 되고, 그
	// 침묵이 닫힌 어휘가 막으려던 실패다.
	shapeAppearanceAttrs = []string{"variant", "fill", "border", "radius", "shadow", "opacity", "rotate", "flip"}
)

// ShapeAttrs는 `<shape>`의 고유·외관·박스 속성이다 (자식 배치 속성은 제외 — 그것은
// withChildPlacement가 잇는다).
//
// 내보내는 이유는 ContainerAttrs와 같다: 값 층(core)이 이 목록을 소비해 "파서가 연 속성 =
// 값 층이 해석하는 속성"을 유지한다. 목록이 두 벌이 되면 속성 하나를 추가할 때 값 층에서
// 빠진 것이 조용한 무시로 나타난다.
//
// **도형에 크기가 필요한 이유**: 도형은 그 자체가 그림이다. 내용이 없으면 잰 크기가 0이고
// 0×0 도형은 산출물에서 아무것도 아니다 (빈 `<text>`가 텍스트 박스를 내는 정책과 갈라지는
// 자리다 — 그쪽은 저자가 자리를 확인해야 하는 **콘텐츠**다). `w`/`h`/`ratio` 없이는 도형이
// "텍스트를 감싼 상자"로만 존재해 배지·구분면·카드 배경 같은 실제 용도가 전부 닫힌다.
// `pad`는 도형에서 **안쪽 여백**이고 PPTX에서는 `bodyPr`의 인셋이 된다 — 글자가 도형 변에
// 붙는 카드는 저자가 만들 수 없는 것이 정상이다.
func ShapeAttrs() []string {
	box := BoxAttrs()
	attrs := make([]string, 0, len(shapeOwnAttrs)+len(shapeAppearanceAttrs)+len(box))
	attrs = append(attrs, shapeOwnAttrs...)
	attrs = append(attrs, shapeAppearanceAttrs...)
	return append(attrs, box...)
}

// ShapeAppearanceAttrs는 도형 외관 속성 목록이다 (순서 = 진단·문서 순서). ShapeAttrs와
// 같은 이유로 내보낸다 — 캐스케이드가 이 목록의 속성만 해석한다.
func ShapeAppearanceAttrs() []string {
	attrs := make([]string, len(shapeAppearanceAttrs))
	copy(attrs, shapeAppearanceAttrs)
	return attrs
}

// AnchorAttr는 텍스트의 **세로 앵커**다. `<text>`와 블록 태그에만
// 열린다.
//
// 앵커가 **보이는 유일한 자리는 도형 안**이다: 슬라이드 직속 텍스트 박스는 잰 rect가 곧
// 내용 크기라 세로 앵커가 움직일 여백이 없다 (`bodyPr@anchor`는 유효하지만 무해하다).
// 그래서 도형 어휘와 같은 항목에서 열린다.
const AnchorAttr = "anchor"

// shapeChildren은 `<shape>`의 자식 어휘다 (순서 = 진단 순서) — **텍스트 요소와 레이아웃
// 컨테이너 넷**이다. 둘이 섞이는 것도 유효하고, 어느 쪽이 왔는지가 도형의 **모드**를
// 정한다 (ShapeHoldsLayout): 텍스트뿐이면 글자가 도형의 `txBody`로 흡수되고,
// 컨테이너가 하나라도 있으면 도형은 **배경**이 되고 자식은 잰 좌표의 형제 도형이 된다.
//
// 개수는 제한하지 않는다: 문법으로 개수를 세는 새 규칙을 만들지 않고, 자식 여럿은 흡수
// 모드에서 `txBody`의 **문단 여럿**으로 뜻을 갖는다 (render/pptx.writeListShape가 이미 여러
// 문단을 한 도형에 담는 선례다).
//
// 컨테이너만 열고 나머지 어휘는 **컨테이너를 거쳐** 오게 한다. 리스트·
// `<image>`·`<table>`·중첩 `<shape>`는 layoutChildren에 이미 있으므로 `<shape><col><ul>…`이
// 새 매핑 없이 성립하고, 도형의 직속 자식 목록은 "흡수냐 배경이냐"만 가르는 좁은 판정으로
// 남는다. 직속으로도 열면 같은 뜻을 두 갈래로 쓸 수 있게 되는데, 그 둘은 배치 규칙이 다르다
// (컨테이너는 `gap`·`align`을 갖고 직속 자식은 갖지 않는다).
//
// 넣지 않은 것과 근거:
//   - `<spacer/>` — 밀어내기는 flex 흐름 안에서만 뜻이 있다. 도형의 직속 자식에는 흐름이
//     없다 (흡수 모드는 문단이고, 배경 모드에서 흐름을 만드는 것은 컨테이너다).
//   - `<li>` — 리스트 컨테이너의 자식으로만 온다 (layoutChildren의 근거와 같다).
//   - `<note>` — 콘텐츠가 아니라 슬라이드 메타데이터다 (layoutChildren의 근거와 같다).
//
// `<shape/>` 자기 닫힘은 그대로 유효하다 — void가 아니어도 자식 없는 태그로 닫힌다 (파서의
// 기존 동작).
func shapeChildren() []string {
	children := make([]string, 0, 1+len(blockElements)+len(layoutContainerElements))
	children = append(children, TextElement)
	children = append(children, blockElements...)
	return append(children, layoutContainerElements...)
}

// ShapeHoldsLayout은 도형이 **배경 모드**인지 말한다 — 직속 자식에 레이아웃 컨테이너가
// 하나라도 있으면 참이다.
//
// 모드 판정의 **단일 소유자**다: core의 측정 소비, render/web의 층 규칙, render/pptx의
// 형제 순회가 모두 이 함수를 읽는다. 세 층이 각자 자식을 훑어 판정하면 "도형 하나가 한
// 모드"라는 계약이 층마다 조금씩 갈라질 수 있고, 그 갈라짐은 두 타깃이 달라 보이는 것으로만
// 드러난다 — 가장 늦게 보이는 실패다.
//
// **직속 자식만** 본다. 손자 컨테이너(`<shape><p>…</p></shape>` 안의 무엇)는 모드를 바꾸지
// 않는다: 문단 안에 컨테이너가 올 수 없으므로 손자 컨테이너는 이미 다른 도형·컨테이너의
// 자식이고, 그 조상의 모드가 그것을 소유한다. 빈 컨테이너(`<shape><col/></shape>`)도 참이다 —
// 자식이 없는 것은 배치가 없는 것이 아니라 빈 배치이고, 도형은 빈 문단을 갖는 배경이 된다.
//
// nil은 거짓이다 (도형이 아닌 것을 묻는 호출자에게 분기를 강요하지 않는다).
func ShapeHoldsLayout(elem *Element) bool {
	if elem == nil || elem.Name != ShapeElement {
		return false
	}
	for _, child := range elem.Children {
		childElem, ok := child.(*Element)
		if !ok {
			continue
		}
		if KindOf(childElem.Name) == KindLayoutContainer {
			return true
		}
	}
	return false
}
