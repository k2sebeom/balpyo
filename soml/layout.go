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
// `<stack>`은 여기 없다 — Iteration 6이다. 겹침 시맨틱(PRD §13-3)의 렌더 없이 어휘에
// 열면 저자가 쓴 겹침이 조용히 무시되고, 그것이 닫힌 어휘로 막으려던 실패다. 자식 속성
// `at`·`x`·`y`도 같은 이유로 닫혀 있다.

// 레이아웃 태그 이름. 렌더러가 grid와 flex를 가르는 판단(`display:grid` vs
// `display:flex`)을 문자열 리터럴이 아니라 이 상수로 하도록 내보낸다 — TextElement·
// BreakElement를 상수로 둔 것과 같은 근거다.
const (
	RowElement  = "row"
	ColElement  = "col"
	GridElement = "grid"
	// SpacerElement는 밀어내기용 빈 요소다 (`flex:1`). void 요소이므로 `<spacer/>`로만
	// 쓴다 — 자식도 문자 데이터도 갖지 않는다 (IsVoidElement).
	SpacerElement = "spacer"
)

// StyleAttr는 **배치 전용 CSS 탈출구**다 (PRD §7.9). 배치는 어차피 좌표로 구워지므로
// PPTX 충실도를 해치지 않지만, 외관 선언(색·테두리)을 쓰면 웹 전용이 되어 경고가 붙는다 —
// allowlist 검사와 경고는 값 층(core)의 몫이고 파서는 이름만 본다.
const StyleAttr = "style"

// layoutContainerElements는 레이아웃 컨테이너다. **순서가 곧 진단·문서 순서다** — 맵이
// 아니라 슬라이스인 이유이고, elementKinds 맵은 여기서 파생된다.
var layoutContainerElements = []string{RowElement, ColElement, GridElement}

// 컨테이너 속성 (PRD §7.4 컨테이너 표). 셋으로 나눈 이유는 트랙 목록이 `<grid>`에만
// 열리기 때문이고, 나눠 두어도 이어 붙인 순서가 PRD 표 순서와 같다.
var (
	// containerFlowAttrs는 flex·grid가 함께 받는 배치 속성이다.
	containerFlowAttrs = []string{"gap", "align", "justify", "wrap", "pad"}
	// containerTrackAttrs는 `grid-template-columns/rows`다 — `<grid>` 전용이다.
	// flex 컨테이너에 여는 길과 저울질했고 닫는 쪽을 골랐다: `grid-template-*`는 flex
	// 컨테이너의 CSS에서 무효이므로 `<row cols="3">`은 받아 놓고 조용히 버려지는 선언이
	// 되고, 그 침묵이 닫힌 어휘가 막으려던 실패다. 저자가 트랙을 원하면 태그를
	// `<grid>`로 바꾸는 것이 고칠 방법이고, 진단이 그것을 말한다.
	containerTrackAttrs = []string{"cols", "rows"}
	// containerBoxAttrs는 컨테이너 자신의 크기·비율이다.
	containerBoxAttrs = []string{"w", "h", "min-w", "min-h", "max-w", "max-h", "ratio"}
)

// childPlacementAttrs는 **자식 배치 속성**이다 (PRD §7.4 자식 표) — 부모 컨테이너 안에서
// 자기 자리를 정하는 채널이므로 컨테이너가 아니라 **자식**에 붙는다.
//
// PRD는 "모든 컴포넌트/레이아웃 태그에 적용 가능"이라 하지만, 여는 곳은 **레이아웃
// 컨테이너의 자식이 될 수 있는 요소**로 한정한다 (withChildPlacement의 호출자):
// `<text>`·블록·리스트 컨테이너·레이아웃 컨테이너·`<spacer>`. `<li>`와 인라인 태그는
// 빠진다 — `<li>`는 리스트 컨테이너의 자식이지 flex 항목이 아니고 (PPTX에서 리스트는
// 도형 하나다 — render/pptx.writeListShape), 인라인은 박스가 아니라 런이다. 열어 두면
// `<li grow="1">`이 웹에서만 뜻을 갖고 PPTX에서는 사라지는 선언이 된다.
var childPlacementAttrs = []string{"grow", "shrink", "basis", "span", "self", "order", StyleAttr}

// ContainerAttrs는 레이아웃 컨테이너 name의 컨테이너 속성이다 (자식 배치 속성은 제외 —
// 그것은 withChildPlacement가 잇는다).
//
// 내보내는 이유: 값 층(core)이 이 목록을 소비해 "파서가 연 속성 = 값 층이 해석하는 속성"을
// 유지한다. 목록이 두 벌이 되면 속성 하나를 추가할 때 값 층에서 빠진 것이 조용한 무시로
// 나타난다 — 카탈로그를 한 곳에 둔 이유가 그것이다.
func ContainerAttrs(name string) []string {
	attrs := make([]string, 0, len(containerFlowAttrs)+len(containerTrackAttrs)+len(containerBoxAttrs))
	attrs = append(attrs, containerFlowAttrs...)
	if name == GridElement {
		attrs = append(attrs, containerTrackAttrs...)
	}
	return append(attrs, containerBoxAttrs...)
}

// ChildPlacementAttrs는 자식 배치 속성 목록이다 (순서 = 진단·문서 순서). ContainerAttrs와
// 같은 이유로 내보낸다.
func ChildPlacementAttrs() []string {
	attrs := make([]string, len(childPlacementAttrs))
	copy(attrs, childPlacementAttrs)
	return attrs
}

// withChildPlacement는 요소 고유 속성 뒤에 자식 배치 속성을 잇는다.
//
// 태그마다 속성 표를 손으로 적지 않는 이유: 자식 배치 속성은 한 벌이고, 표를 복제하면
// PRD 표에 속성이 하나 늘 때 어느 태그가 빠졌는지 사람이 세어야 한다. 고유 속성이
// 앞에 오는 순서는 진단 메시지가 "이 태그다운 속성"부터 나열하게 한다.
func withChildPlacement(attrs []string) []string {
	merged := make([]string, 0, len(attrs)+len(childPlacementAttrs))
	merged = append(merged, attrs...)
	return append(merged, childPlacementAttrs...)
}

// layoutChildren은 레이아웃 컨테이너의 자식 어휘다 (순서 = 진단 순서).
//
// 레이아웃 컨테이너가 자신도 목록에 있는 것은 의도다 — 중첩 레이아웃(`<row>` 안의
// `<col>`)이 이 어휘의 핵심 장면이다. `<note>`는 없다: 노트는 콘텐츠가 아니라 슬라이드
// 메타데이터라 SlideDoc.Notes로 빠지므로(noteElement), 컨테이너 안의 노트는 "어느
// 슬라이드의 노트인가"가 같은 답이면서 트리 위치만 다른 중복이 된다. `<li>`도 없다 —
// 리스트 컨테이너의 자식으로만 온다.
func layoutChildren() []string {
	children := make([]string, 0, 1+len(blockElements)+len(listContainerElements)+len(layoutContainerElements)+1)
	children = append(children, TextElement)
	children = append(children, blockElements...)
	children = append(children, listContainerElements...)
	children = append(children, layoutContainerElements...)
	return append(children, SpacerElement)
}
