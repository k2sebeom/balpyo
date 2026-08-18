package soml

// 이 파일은 텍스트 서브셋(PRD §7.7)의 **요소 종류 카탈로그**다.
//
// 표를 한 곳에만 두는 이유는 TextElement 상수와 같다: 이 구분은 파서(배치 규칙)만이 아니라
// 지오메트리 패스(잴 대상), 웹 렌더러(블록 vs 런), PPTX 렌더러(텍스트 박스 vs 런)가 전부
// 소비한다. 네 층이 각자 이름 목록을 들고 있으면 태그 하나를 추가할 때 한 곳을 빼먹는 것이
// 곧 "좌표가 엉뚱한 요소에 붙는" 조용한 실패다.

// ElementKind는 텍스트 서브셋 요소의 종류다.
type ElementKind int

const (
	// KindNone은 서브셋 밖의 요소다 (<slide>, <text>, <note> 등 — 각자의 규칙을 따른다).
	KindNone ElementKind = iota
	// KindBlock은 블록 요소다. 슬라이드 직속에 놓이고, 문자 데이터와 인라인 태그를 담는다.
	// 블록끼리는 겹치지 않는다 — v1의 블록은 문단 하나이지 문서 섹션이 아니다.
	KindBlock
	// KindListContainer는 리스트 컨테이너다 (<ul>/<ol>). 자식은 <li>뿐이다.
	KindListContainer
	// KindListItem은 리스트 항목이다 (<li>). 리스트 컨테이너의 자식으로만 온다.
	KindListItem
	// KindInline은 인라인 요소다. 블록·<text>·<li> 안에서 텍스트 런 서식이 되고,
	// 인라인끼리 중첩할 수 있다 (<a> 안의 <a>만 예외).
	KindInline
)

// 종류별 이름 목록. **순서가 곧 진단 메시지와 문서의 순서다** — 맵이 아니라 슬라이스인
// 이유이고, elementKinds 맵은 여기서 파생된다.
var (
	blockElements         = []string{"h1", "h2", "h3", "p", "blockquote", "pre"}
	listContainerElements = []string{"ul", "ol"}
	inlineElements        = []string{"strong", "em", "code", "a", "br", "span"}
)

// ListItemElement는 리스트 항목 태그 이름이다.
const ListItemElement = "li"

// BreakElement는 줄바꿈 태그 이름이다. 서브셋의 유일한 void 요소다 — 내용을 갖지 않고
// `<br/>`(자기 닫힘)로만 쓴다. 렌더러가 런 분할에서 이 이름을 특별 취급한다.
const BreakElement = "br"

// elementKinds는 이름 → 종류 색인이다. 위의 순서 있는 목록에서 파생된다.
var elementKinds = buildElementKinds()

func buildElementKinds() map[string]ElementKind {
	kinds := make(map[string]ElementKind)
	for _, name := range blockElements {
		kinds[name] = KindBlock
	}
	for _, name := range listContainerElements {
		kinds[name] = KindListContainer
	}
	kinds[ListItemElement] = KindListItem
	for _, name := range inlineElements {
		kinds[name] = KindInline
	}
	return kinds
}

// KindOf는 태그 이름의 종류다. 서브셋 밖의 이름은 KindNone이다.
func KindOf(name string) ElementKind {
	return elementKinds[name]
}

// IsVoidElement는 이 이름이 void 요소(자식·문자 데이터를 갖지 않는 요소)인지 보고한다.
func IsVoidElement(name string) bool {
	return name == BreakElement
}
