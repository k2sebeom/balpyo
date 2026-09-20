package soml

// 이 파일은 레이아웃 태그(PRD §7.4)의 **요소 종류·속성 카탈로그**다.
//
// 서브셋 카탈로그(subset.go)와 같은 자리에 같은 태도로 둔다: 이 표를 파서(문법·배치 규칙),
// 지오메트리 패스(잴 대상), 웹 렌더러(flex/grid CSS), PPTX 렌더러(컨테이너는 도형이 되지
// 않고 자식 좌표만 남긴다)가 전부 소비한다. 네 층이 각자 이름 목록을 들고 있으면 태그
// 하나를 추가할 때 한 곳을 빼먹는 것이 곧 "파싱은 되는데 그려지지 않는" 조용한 실패다.
//
// 레이아웃 태그는 **CSS flex/grid 시맨틱의 이름 바꾸기 층**이다 (PRD §7.4) — 새 배치
// 모델이 아니다. 그래서 속성 이름과 **순서**를 PRD 표 그대로 둔다: 순서가 곧 진단
// 메시지의 순서이자 문서의 순서다.
//
// `<stack>`은 Iteration 6에 열렸다 — 겹침 시맨틱(PRD §13-3)의 겹침 렌더가 웹(FlowStack의
// CSS)·PPTX(문서 순서 z-축)에 함께 오는 iteration이다. 자식 배치 속성 `at`·`x`·`y`도 같은
// iteration에 열렸다 (childPlacementAttrs). 값 어휘·부모 한정 검증은 값 층(core)의 몫이다.

// 레이아웃 태그 이름. 렌더러가 grid와 flex를 가르는 판단(`display:grid` vs
// `display:flex`)을 문자열 리터럴이 아니라 이 상수로 하도록 내보낸다 — TextElement·
// BreakElement를 상수로 둔 것과 같은 근거다.
const (
	RowElement  = "row"
	ColElement  = "col"
	GridElement = "grid"
	// StackElement는 **겹침 컨테이너**다 (PRD §7.4·§13-3) — 단일 셀 grid로, 모든 자식이
	// 같은 칸에 쌓인다. 배치 흐름은 KindStack이 아니라 FlowStack이다 (placement.go의
	// 근거): 파서·core·PPTX·지오메트리에서 `<row>`와 산출이 전부 같고 다른 것은 웹의
	// CSS 한 줄뿐이며, 그 한 줄은 이미 종류가 아니라 흐름으로 갈린다.
	StackElement = "stack"
	// SpacerElement는 밀어내기용 빈 요소다 (`flex:1`). void 요소이므로 `<spacer/>`로만
	// 쓴다 — 자식도 문자 데이터도 갖지 않는다 (IsVoidElement).
	SpacerElement = "spacer"
)

// StyleAttr는 **배치 전용 CSS 탈출구**다 (PRD §7.9). 배치는 어차피 좌표로 구워지므로
// PPTX 충실도를 해치지 않지만, 외관 선언(색·테두리)을 쓰면 웹 전용이 되어 경고가 붙는다 —
// allowlist 검사와 경고는 값 층(core)의 몫이고 파서는 이름만 본다.
const StyleAttr = "style"

// layoutContainerElements는 레이아웃 컨테이너다. **순서가 곧 진단·문서 순서다** — 맵이
// 아니라 슬라이스인 이유이고, elementKinds 맵은 여기서 파생된다. `<stack>`은 `<grid>`
// 뒤에 온다 (진단의 자식 태그 목록이 `…, <grid>, <stack>, <spacer>`가 된다).
var layoutContainerElements = []string{RowElement, ColElement, GridElement, StackElement}

// 컨테이너 속성 (PRD §7.4 컨테이너 표)과 **박스 속성**. 넷으로 나눈 이유는 소비자가 다르기
// 때문이다 — 트랙 목록은 `<grid>`에만 열리고, 박스 속성은 `<shape>`와 공유한다. 나눠 두어도
// 이어 붙인 순서(ContainerAttrs)는 PRD 표 순서와 같다.
var (
	// containerFlowAttrs는 flex·grid가 함께 받는 배치 속성이다.
	containerFlowAttrs = []string{"gap", "align", "justify", "wrap"}
	// stackFlowAttrs는 `<stack>`이 받는 흐름 속성이다 — flex 공통 넷에서 `gap`·`wrap`을 뺀
	// 것이다. 근거는 단일 셀이라는 stack의 정의다 (PRD §13-3): `gap`은 셀 사이의 간격인데
	// 자식이 전부 같은 칸에 있어 산출 CSS에 나가도 아무것도 하지 않고, `wrap`은 flex 전용
	// (`flex-wrap`)이라 grid 흐름인 stack에서 무효다. 받아 놓고 아무것도 하지 않는 속성은
	// 저자가 "왜 안 되지"를 산출물에서 발견하게 만들고, 그것이 닫힌 어휘로 막으려던
	// 실패의 변종이다. 남는 `align`/`justify`는 stack에서 뜻이 있다 — 단일 셀 안 모든
	// 자식의 기본 정렬이고, `at`이 그것을 자식 하나에서 덮는다.
	stackFlowAttrs = []string{"align", "justify"}
	// containerTrackAttrs는 `grid-template-columns/rows`다 — `<grid>` 전용이다.
	// flex 컨테이너에 여는 길과 저울질했고 닫는 쪽을 골랐다: `grid-template-*`는 flex
	// 컨테이너의 CSS에서 무효이므로 `<row cols="3">`은 받아 놓고 조용히 버려지는 선언이
	// 되고, 그 침묵이 닫힌 어휘가 막으려던 실패다. 저자가 트랙을 원하면 태그를
	// `<grid>`로 바꾸는 것이 고칠 방법이고, 진단이 그것을 말한다.
	containerTrackAttrs = []string{"cols", "rows"}
	// boxPadAttrs는 박스의 **안쪽 여백**이다. containerFlowAttrs에서 떼어 냈다 — 여백은
	// flex/grid의 성질이 아니라 박스의 성질이고, `<shape>`가 같은 속성을 같은 뜻으로
	// 받는다 (도형에서는 PPTX `bodyPr`의 인셋이 된다).
	boxPadAttrs = []string{"pad"}
	// boxSizeAttrs는 박스 자신의 크기·비율이다 (옛 containerBoxAttrs).
	//
	// 이름을 다시 붙인 이유: 크기는 컨테이너의 성질이 아니라 박스의 성질이므로 컨테이너와
	// 도형이 **같은 목록**을 소비해야 한다. 목록이 두 벌이 되면 PRD 표에 크기 속성이 하나
	// 늘 때 어느 태그가 빠졌는지 사람이 세어야 한다 — withChildPlacement를 만든 것과 같은
	// 근거다.
	boxSizeAttrs = []string{"w", "h", "min-w", "min-h", "max-w", "max-h", "ratio"}
	// textSizeAttrs는 텍스트 박스가 받는 크기 속성이다 — **가로 축만** (ADR 0043).
	//
	// boxSizeAttrs의 **부분집합이고 순서도 같다** (TestTextSizeAttrsAreBoxSubset이 지킨다):
	// 값 문법을 컨테이너·도형과 공유하므로(`core.boxAttr`) `fill`·`%`·공간 토큰이 같은 뜻이고,
	// 목록이 두 벌이면 같은 표기가 두 뜻을 갖게 된다.
	//
	// 세로 축(`h`·`min-h`·`max-h`)과 `pad`·`ratio`를 닫아 두는 이유 (ADR 0043):
	//   - 박스 안 세로 정렬 채널은 `anchor`이고 그 채널은 **도형 안에서만** 뜻을 가지므로
	//     (웹은 도형 본문의 flex, PPTX는 `bodyPr@anchor`), 높이만 열면 저자가 만든 여백 안에서
	//     글을 움직일 방법이 없다 — 닿을 수 없는 공간이 생긴다.
	//   - `pad`는 PPTX 텍스트 박스의 인셋 0 계약(`.balpyo-text`의 padding·border가 0이므로 잰
	//     rect가 곧 글이 흐르는 사각형)을 두 타깃에서 동시에 다시 정하게 만든다.
	//   - `ratio`는 텍스트 박스의 높이를 글줄 수가 정하므로 대응이 없다.
	textSizeAttrs = []string{"w", "min-w", "max-w"}
)

// BoxAttrs는 박스의 안쪽 여백과 크기다 (`pad` + 크기·비율 7종). `<shape>`가 그대로 받고
// (ShapeAttrs), 레이아웃 컨테이너는 사이에 트랙 목록이 끼므로 두 목록을 따로 잇는다
// (ContainerAttrs) — 순서가 진단 메시지에 실리기 때문이다.
//
// 내보내는 이유는 ContainerAttrs와 같다: 값 층(core)이 이 목록을 소비해 "파서가 연 속성 =
// 값 층이 해석하는 속성"을 유지한다.
func BoxAttrs() []string {
	attrs := make([]string, 0, len(boxPadAttrs)+len(boxSizeAttrs))
	attrs = append(attrs, boxPadAttrs...)
	return append(attrs, boxSizeAttrs...)
}

// TextSizeAttrs는 텍스트 박스의 가로 크기 속성이다 (`w`·`min-w`·`max-w` — ADR 0043).
// `<text>`·블록·리스트 컨테이너가 받고, 값 층(core)이 BoxAttrs와 **같은 헬퍼**로 해석한다.
//
// BoxAttrs와 같은 이유로 내보낸다: 파서가 연 속성과 값 층이 해석하는 속성이 갈라지면
// 갈라진 쪽은 조용히 무시된다.
func TextSizeAttrs() []string {
	attrs := make([]string, len(textSizeAttrs))
	copy(attrs, textSizeAttrs)
	return attrs
}

// withTextSize는 텍스트 요소의 외관 속성 뒤에 가로 크기 속성을 잇는다 — 속성 표가
// **외관 → 박스(가로) → 자식 배치 → 슬롯** 순이 된다 (`<shape>`의 ShapeAttrs와 같은 순서:
// 고유·외관이 먼저, 박스가 다음, 배치가 끝이다). withChildPlacement와 같은 이유로 함수다:
// 목록을 태그마다 손으로 적으면 크기 속성이 하나 늘 때 어느 태그가 빠졌는지 사람이 센다.
func withTextSize(attrs []string) []string {
	merged := make([]string, 0, len(attrs)+len(textSizeAttrs))
	merged = append(merged, attrs...)
	return append(merged, textSizeAttrs...)
}

// childPlacementAttrs는 **자식 배치 속성**이다 (PRD §7.4 자식 표) — 부모 컨테이너 안에서
// 자기 자리를 정하는 채널이므로 컨테이너가 아니라 **자식**에 붙는다.
//
// PRD는 "모든 컴포넌트/레이아웃 태그에 적용 가능"이라 하지만, 여는 곳은 **레이아웃
// 컨테이너의 자식이 될 수 있는 요소**로 한정한다 (withChildPlacement의 호출자):
// `<text>`·블록·리스트 컨테이너·레이아웃 컨테이너·`<spacer>`. `<li>`와 인라인 태그는
// 빠진다 — `<li>`는 리스트 컨테이너의 자식이지 flex 항목이 아니고 (PPTX에서 리스트는
// 도형 하나다 — render/pptx.writeListShape), 인라인은 박스가 아니라 런이다. 열어 두면
// `<li grow="1">`이 웹에서만 뜻을 갖고 PPTX에서는 사라지는 선언이 된다.
//
// `at`·`x`·`y`는 **겹침 배치 채널**이다 (PRD §7.4 자식 표) — Iteration 6에 열렸다. `at`은
// stack 셀 안의 9지점 앵커(`place-self`), `x`/`y`는 절대 오프셋이다. `span`이 `<grid>`의
// 자식에서만 뜻이 있듯 이들도 부모가 한정한다(`at`은 stack만, `x`/`y`는 stack·canvas만)
// — 목록은 한 벌이므로 여기서 이름만 열고, 값 어휘와 부모 한정 검증은 값 층(core)이
// 한다 (span의 선례와 같다). 순서는 PRD 자식 표를 따라 `order` 뒤, `style`(탈출구이므로
// 항상 끝) 앞이다.
var childPlacementAttrs = []string{"grow", "shrink", "basis", "span", "self", "order", "at", "x", "y", StyleAttr}

// ContainerAttrs는 레이아웃 컨테이너 name의 컨테이너 속성이다 (자식 배치 속성은 제외 —
// 그것은 withChildPlacement가 잇는다).
//
// 내보내는 이유: 값 층(core)이 이 목록을 소비해 "파서가 연 속성 = 값 층이 해석하는 속성"을
// 유지한다. 목록이 두 벌이 되면 속성 하나를 추가할 때 값 층에서 빠진 것이 조용한 무시로
// 나타난다 — 카탈로그를 한 곳에 둔 이유가 그것이다.
func ContainerAttrs(name string) []string {
	flow := containerFlowAttrs
	if name == StackElement {
		// stack은 단일 셀이라 `gap`·`wrap`이 뜻이 없다 (stackFlowAttrs의 근거). grid 분기가
		// 트랙을 더하듯, stack 분기는 흐름 속성을 좁힌다.
		flow = stackFlowAttrs
	}
	attrs := make([]string, 0, len(flow)+len(boxPadAttrs)+len(containerTrackAttrs)+len(boxSizeAttrs))
	attrs = append(attrs, flow...)
	attrs = append(attrs, boxPadAttrs...)
	if name == GridElement {
		attrs = append(attrs, containerTrackAttrs...)
	}
	return append(attrs, boxSizeAttrs...)
}

// ChildPlacementAttrs는 자식 배치 속성 목록이다 (순서 = 진단·문서 순서). ContainerAttrs와
// 같은 이유로 내보낸다.
func ChildPlacementAttrs() []string {
	attrs := make([]string, len(childPlacementAttrs))
	copy(attrs, childPlacementAttrs)
	return attrs
}

// withChildPlacement는 요소 고유 속성 뒤에 자식 배치 속성을 잇고, 마지막으로 슬롯 배정
// 속성(`slot`)을 붙인다.
//
// 태그마다 속성 표를 손으로 적지 않는 이유: 자식 배치 속성은 한 벌이고, 표를 복제하면
// PRD 표에 속성이 하나 늘 때 어느 태그가 빠졌는지 사람이 세어야 한다. 고유 속성이
// 앞에 오는 순서는 진단 메시지가 "이 태그다운 속성"부터 나열하게 한다.
//
// `slot`이 여기 붙는 이유(SlotAttr): 이 함수의 수신자 집합(`<text>`·`<shape>`·`<image>`·
// 블록·리스트 컨테이너·레이아웃 컨테이너·`<spacer>`)이 곧 **슬라이드 직계 자식이 될 수 있는
// 요소 집합**과 일치한다 (`<note>`만 빠지는데 그것은 콘텐츠가 아니라 메타데이터다). 슬롯은
// 배치 채널이 아니라 슬라이드 수준 배정이므로 childPlacementAttrs에는 넣지 않고
// (ChildPlacementAttrs를 소비하는 core의 Placement 해석과 섞이지 않게) 탈출구(style)보다도
// 뒤, 목록의 맨 끝에 둔다 — 진단이 "이 태그다운 속성 → 배치 → 탈출구 → 슬롯" 순으로 읽힌다.
func withChildPlacement(attrs []string) []string {
	merged := make([]string, 0, len(attrs)+len(childPlacementAttrs)+1)
	merged = append(merged, attrs...)
	merged = append(merged, childPlacementAttrs...)
	return append(merged, SlotAttr)
}

// layoutChildren은 레이아웃 컨테이너의 자식 어휘다 (순서 = 진단 순서).
//
// 컴포넌트(`<text>`·`<shape>`·`<image>`)가 앞에 오고 텍스트 서브셋·레이아웃이 뒤따른다.
// `<shape>`가 여기 있는 것이 Iteration 4의 핵심 장면이었다 — `<grid cols="4">` 안의 카드
// 도형. `<image>`가 더해진 것이 Iteration 5의 핵심 장면이다 — 그림+텍스트 2단(`<row>`)과
// 그림 그리드(`<grid cols="3">`).
//
// 레이아웃 컨테이너가 자신도 목록에 있는 것은 의도다 — 중첩 레이아웃(`<row>` 안의
// `<col>`)이 이 어휘의 핵심 장면이다. `<note>`는 없다: 노트는 콘텐츠가 아니라 슬라이드
// 메타데이터라 SlideDoc.Notes로 빠지므로(noteElement), 컨테이너 안의 노트는 "어느
// 슬라이드의 노트인가"가 같은 답이면서 트리 위치만 다른 중복이 된다. `<li>`도 없다 —
// 리스트 컨테이너의 자식으로만 온다.
func layoutChildren() []string {
	children := make([]string, 0, 4+len(blockElements)+len(listContainerElements)+len(layoutContainerElements)+1)
	children = append(children, TextElement, ShapeElement, ImageElement, TableElement)
	children = append(children, blockElements...)
	children = append(children, listContainerElements...)
	children = append(children, layoutContainerElements...)
	return append(children, SpacerElement)
}
