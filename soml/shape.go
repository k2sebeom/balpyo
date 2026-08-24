package soml

// 이 파일은 도형(PRD §7.5·§7.6)의 **요소 종류·속성·값 어휘 카탈로그**다.
//
// 서브셋(subset.go)·레이아웃(layout.go) 카탈로그와 같은 자리에 같은 태도로 둔다: 이 표를
// 파서(문법), 값 층(core의 `type`·색·길이 검사), 지오메트리 패스(잴 대상), 웹 렌더러(모양을
// 그린다), PPTX 렌더러(`prstGeom`)가 전부 소비한다. 다섯 층이 각자 이름 목록을 들고 있으면
// 태그 하나·프리셋 하나를 추가할 때 한 곳을 빼먹는 것이 곧 "파싱은 되는데 그려지지 않는"
// 조용한 실패다.
//
// `<arrow>`·`<line>`·`<group>`은 여기 없다 — Iteration 7·9다. `<image>`는 Iteration 5에
// 자기 카탈로그(image.go)로 열렸다. 렌더 매핑 없는 태그를 어휘에 열면 저자가 쓴 것이 조용히
// 무시되고, 그것이 닫힌 어휘가 막으려던 실패다.

// ShapeElement는 도형 요소 이름이다. TextElement와 같은 이유로 상수다 — 이름 문자열이
// 여러 층에 흩어지면 오타 하나가 조용한 실패가 된다.
const ShapeElement = "shape"

// shapeTypes는 `<shape type>`의 v1 값 어휘 12종이다 (PRD §7.5). **순서가 PRD 표 순서이자
// 진단·문서의 순서다** — 맵이 아니라 슬라이스인 이유이고, 모르는 `type` 진단이 나열하는
// 목록이 결정적이어야 한다 (Theme.ColorNames·textVariantNames와 같은 태도).
//
// 목록은 soml에 있고 **검사는 core**다: 파서는 문법과 이름만 보고 값의 의미는 값 층이
// 본다 (Element.Attrs 주석). 12종은 전부 OOXML 프리셋 지오메트리의 부분집합이며, 확장할
// 때도 OOXML에 존재하는 것만 더해 번역 가능성을 지킨다 (PRD §7.5).
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

// BorderStyle은 `border` 축약의 스타일 조각이다 (PRD §8.3의 `"1 solid muted"`).
//
// **이름은 CSS 쪽이다** (`dashed`/`dotted`, OOXML의 `dash`/`dot`이 아니다). 근거는 축약
// 자체가 CSS `border`의 표기이기 때문이다 — 저자가 아는 표기를 그대로 쓰는 것이 이
// 저장소의 결이고(표기를 새로 발명하지 않는다), CSS 모양의 축약 안에 OOXML 어휘를 섞으면
// 둘 다 아닌 제3의 표기가 된다. PRD §7.6 표의 `dash` 행 예시도 `dashed`다.
//
// 어휘를 셋으로 닫은 기준은 **양 타깃에 대응이 있는 것만**이다: 셋은 OOXML `prstDash`의
// `solid`/`sysDash`/`sysDot`(계열 선택은 렌더 매핑의 몫 — 항목 05)에 대응한다. CSS의
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

// Flip은 도형 대칭이다 (PRD §7.6의 `flip="h"`).
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

// 도형 속성 (PRD §7.6 표의 shape 행). 넷으로 나눈 이유는 소비자가 다르기 때문이다:
// 고유 속성은 파서·참조 해석(Iteration 7)이, 외관은 캐스케이드가, 박스 속성은 레이아웃
// 컨테이너와 **공유**한다 (layout.go의 boxPadAttrs·boxSizeAttrs). 이어 붙인 순서가 곧
// 진단 메시지와 문서의 순서다.
var (
	// shapeOwnAttrs는 도형만의 속성이다.
	//
	// `id`는 산출물에 아무것도 그리지 않는다 — 그것이 정상이고 `<slide id>`의 선례가 이미
	// 그렇다 (파일명 정렬·진단에만 쓰인다). 도형 `id`는 `<arrow from="db">`의 **참조 대상
	// 이름**이며 참조 해석은 Iteration 7이다.
	shapeOwnAttrs = []string{"type", "id"}
	// shapeAppearanceAttrs는 외관 채널이다 (PRD §7.6). `stroke`/`cap`/`dash`는 line·arrow의
	// 것이라 열지 않는다 — 도형에 받아 놓고 버리는 선언이 되고, 그 침묵이 닫힌 어휘가
	// 막으려던 실패다.
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

// AnchorAttr는 텍스트의 **세로 앵커**다 (PRD §7.6의 text 행). `<text>`와 블록 태그에만
// 열린다.
//
// 앵커가 **보이는 유일한 자리는 도형 안**이다: 슬라이드 직속 텍스트 박스는 잰 rect가 곧
// 내용 크기라 세로 앵커가 움직일 여백이 없다 (`bodyPr@anchor`는 유효하지만 무해하다).
// 그래서 도형 어휘와 같은 항목에서 열린다.
const AnchorAttr = "anchor"

// shapeChildren은 `<shape>`의 자식 어휘다 (순서 = 진단 순서) — **텍스트 요소**다.
//
// 개수는 제한하지 않는다: 문법으로 개수를 세는 새 규칙을 만들지 않고, 자식 여럿은 PPTX에서
// `txBody`의 **문단 여럿**으로 뜻을 갖는다 (render/pptx.writeListShape가 이미 여러 문단을
// 한 도형에 담는 선례다).
//
// 넣지 않은 것과 근거:
//   - 리스트 컨테이너 — 도형 안 리스트는 불릿 문단 조립(listBodyXML)과 도형 `txBody` 조립을
//     합치는 일이고, 그 합침은 어휘를 여는 것과 다른 크기의 작업이다.
//   - 레이아웃 컨테이너·`<shape>` 중첩 — 도형 안에 배치를 넣는 것은 겹침(Iteration 6의
//     `<stack>`)과 편집 단위(Iteration 9의 `<group>`)의 자리이고, "도형은 `txBody` 하나를
//     갖는다"는 이번 매핑과 정면으로 부딪힌다.
//   - `<image>` — 같은 근거다 (Iteration 5). 도형 안 그림은 PPTX에서 `sp` 안의 `pic`이
//     아니라 도형의 `blipFill`이거나 그룹이고, 둘 다 이 매핑과 다른 것이다 — 그림 위에
//     도형을 얹는 것은 `<stack>`의 자리다.
//   - `<note>` — 콘텐츠가 아니라 슬라이드 메타데이터다 (layoutChildren의 근거와 같다).
//
// `<shape/>` 자기 닫힘은 그대로 유효하다 — void가 아니어도 자식 없는 태그로 닫힌다 (파서의
// 기존 동작). PRD §7.6의 예제가 그 표기다.
func shapeChildren() []string {
	children := make([]string, 0, 1+len(blockElements))
	children = append(children, TextElement)
	return append(children, blockElements...)
}
