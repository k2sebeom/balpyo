package soml

// 이 파일은 텍스트 서브셋(PRD §7.7)의 **요소 종류 카탈로그**다.
//
// 표를 한 곳에만 두는 이유는 TextElement 상수와 같다: 이 구분은 파서(배치 규칙)만이 아니라
// 지오메트리 패스(잴 대상), 웹 렌더러(블록 vs 런), PPTX 렌더러(텍스트 박스 vs 런)가 전부
// 소비한다. 네 층이 각자 이름 목록을 들고 있으면 태그 하나를 추가할 때 한 곳을 빼먹는 것이
// 곧 "좌표가 엉뚱한 요소에 붙는" 조용한 실패다.

// ElementKind는 요소의 종류다 — 텍스트 서브셋과 레이아웃(layout.go)을 함께 담는다.
//
// 열거형을 파일 하나에 모으는 이유: 카탈로그 파일은 어휘별로 나뉘어도 **종류 값은 한
// 곳에서 봐야** 두 어휘가 같은 값을 쓰거나 빈 값을 남기는 사고가 없다. 이름 목록과
// 색인 파생은 각 카탈로그 파일이 갖는다.
type ElementKind int

const (
	// KindNone은 카탈로그 밖의 요소다 (<slide>, <text>, <note> 등 — 각자의 규칙을 따른다).
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
	// KindLayoutContainer는 레이아웃 컨테이너다 (<row>/<col>/<grid> — layout.go). 배치
	// 채널만 갖고 자기 외관은 없으며, PPTX에서는 도형이 되지 않고 자식 좌표만 남기고
	// 사라진다 (TASKS Iteration 3).
	KindLayoutContainer
	// KindSpacer는 <spacer>다. 컨테이너와 **다른 종류**로 두는 이유는 세 층에서 산출이
	// 다르기 때문이다: 콘텐츠 모델이 void이고(자식 없음), 웹은 빈 div를 내고, PPTX에는
	// 대응 도형이 아예 없다. 컨테이너와 같은 종류로 두면 세 층이 각자 `name == "spacer"`
	// 비교를 들게 되고, 그것이 카탈로그를 한 곳에 둔 이유를 무너뜨린다.
	KindSpacer
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

// elementKinds는 이름 → 종류 색인이다. 위의 순서 있는 목록과 레이아웃 카탈로그
// (layout.go)에서 파생된다 — 종류를 갖는 이름은 전부 여기를 지난다.
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
	for _, name := range layoutContainerElements {
		kinds[name] = KindLayoutContainer
	}
	kinds[SpacerElement] = KindSpacer
	return kinds
}

// KindOf는 태그 이름의 종류다. 카탈로그 밖의 이름은 KindNone이다.
func KindOf(name string) ElementKind {
	return elementKinds[name]
}

// IsVoidElement는 이 이름이 void 요소(자식·문자 데이터를 갖지 않는 요소)인지 보고한다.
// `<br>`(런 분할)과 `<spacer>`(밀어내기)가 그것이다 — 둘 다 자기 닫힘으로만 쓴다.
func IsVoidElement(name string) bool {
	return name == BreakElement || name == SpacerElement
}

// TagTextRole은 블록·리스트 항목 태그의 텍스트 역할이다 — PRD §7.7의 등가 표
// (`<p>` ≡ `<text role="body">`, `<h2>` ≡ `<text role="heading">`).
//
// 이 표가 테마(core/theme.go)가 아니라 **언어**(soml)에 있는 이유: 등가는 PRD가 명세한
// 계약이지 테마의 선택이 아니다. 테마가 `<p>`를 caption으로 바꿀 수 있으면 같은 문서가
// 테마마다 다른 역할로 해석되어 "같은 계약의 두 표기"가 성립하지 않는다. 태그별
// **기본값**(h3의 크기, blockquote의 색)은 외관이므로 테마가 소유한다 — 역할과 기본값의
// 소유가 갈라지는 경계가 여기다 (Iteration 10의 components.json이 기본값만 파일로 옮긴다).
//
// h1 → title인 이유: 슬라이드는 문서가 아니라 화면 하나다. v1의 블록은 문단이지 문서
// 섹션이 아니므로(KindBlock 주석) h1이 이끄는 "장"이라는 위계가 없고, 슬라이드에서 가장
// 큰 표제는 곧 그 슬라이드의 제목이다. h1을 heading으로 두면 title 역할에 대응하는 블록
// 표기가 아예 없어져 서브셋만 쓰는 저자가 표지를 만들 수 없다. caption은 반대로 대응
// 태그가 없다 — figcaption은 어휘 밖이고 `<text role="caption">`이 그 자리를 맡는다.
//
// 표에 없는 이름은 ok=false다: `<text>`는 role 속성이 역할을 정하고, 리스트 컨테이너와
// 인라인 태그는 역할을 갖지 않는다 (컨테이너는 항목을 담을 뿐이고, 인라인은 런 서식이다).
func TagTextRole(name string) (TextRole, bool) {
	switch name {
	case "h1":
		return RoleTitle, true
	case "h2", "h3":
		return RoleHeading, true
	case "p", "blockquote", ListItemElement:
		return RoleBody, true
	case "pre":
		return RoleCode, true
	}
	return "", false
}
