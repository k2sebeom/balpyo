package soml

// 이 파일은 텍스트 서브셋의 **요소 종류 카탈로그**다.
//
// 표를 한 곳에만 두는 이유는 TextElement 상수와 같다: 이 구분은 파서(배치 규칙)만이 아니라
// 지오메트리 패스(잴 대상), 웹 렌더러(블록 vs 런), PPTX 렌더러(텍스트 박스 vs 런)가 전부
// 소비한다. 네 층이 각자 이름 목록을 들고 있으면 태그 하나를 추가할 때 한 곳을 빼먹는 것이
// 곧 "좌표가 엉뚱한 요소에 붙는" 조용한 실패다.

// ElementKind는 요소의 종류다 — 텍스트 서브셋과 레이아웃(layout.go)·도형(shape.go)·
// 이미지(image.go)를 함께 담는다.
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
	// 사라진다.
	KindLayoutContainer
	// KindSpacer는 <spacer>다. 컨테이너와 **다른 종류**로 두는 이유는 세 층에서 산출이
	// 다르기 때문이다: 콘텐츠 모델이 void이고(자식 없음), 웹은 빈 div를 내고, PPTX에는
	// 대응 도형이 아예 없다. 컨테이너와 같은 종류로 두면 세 층이 각자 `name == "spacer"`
	// 비교를 들게 되고, 그것이 카탈로그를 한 곳에 둔 이유를 무너뜨린다.
	KindSpacer
	// KindShape는 <shape>다 (shape.go). 별도 종류인 근거는 네 층의 산출이 전부 다르다는
	// 것이다: 콘텐츠 모델이 다르고(텍스트를 담는다), 잰 rect가 **도형 좌표로 쓰이며**
	// (컨테이너·<li>와 다르다), 웹은 모양을 그려야 하고, PPTX는 `prstGeom`을 낸다.
	// **재는 것과 도형이 되는 것이 처음 일치하는 종류**이므로 종류를 나누지 않으면 세 층이
	// 각자 `name == "shape"` 비교를 들게 되고, 그것이 카탈로그를 한 곳에 둔 이유를
	// 무너뜨린다 (KindSpacer의 근거와 같은 종류다).
	KindShape
	// KindImage는 <image>다 (image.go). 별도 종류인 근거는 KindShape와 같다 — 네 층의
	// 산출이 전부 다르다: 콘텐츠 모델이 다르고(void다 — 담을 것이 없다), 잰 rect가
	// **그림 좌표로 쓰이고**(`p:pic`의 `xfrm`), 웹은 `<img>`를 내고, PPTX는 `blipFill` +
	// 미디어 파트를 낸다. 종류를 나누지 않으면 네 층이 각자 `name == "image"` 비교를 들게
	// 된다.
	//
	// **재는 것과 도형이 되는 것이 일치하는 두 번째 종류다** (KindShape가 첫 번째).
	// KindShape와 합치지 않는 이유가 그 하나로 갈라진다: 도형은 자기 안에 `txBody`를
	// 담고(자식 텍스트를 흡수한다) 그림은 담을 것이 없으며, PPTX에서 `sp`와 `pic`은 다른
	// 요소다 — 합쳐 두면 두 렌더러가 다시 이름으로 갈라야 한다.
	KindImage
	// KindTable은 `<table>`이다 (table.go). 별도 종류인 근거는 네 층의 산출이 전부 다르다는
	// 것이다: PPTX에서 처음으로 `p:sp`도 `p:pic`도 아닌 `p:graphicFrame`+`a:tbl`이 되고,
	// 잰 프레임 rect가 그래픽 프레임 `xfrm`이 되며(도형·그림처럼 재는 것과 도형이 일치),
	// 웹은 진짜 `<table>`을 그린다. 리스트·레이아웃 컨테이너의 교차다 — 프레임은 도형이되
	// 내부 격자는 잰 셀 박스에서 유도한다.
	KindTable
	// KindTableRow는 `<tr>`이다 (table.go). 도형이 되지 않는다 — `a:tr`로 프레임 안에
	// 접힌다. 별도 종류인 근거는 KindListItem과 같다: 콘텐츠 모델(셀만 담는다)과 배치
	// 규칙(<table>의 자식으로만)이 다르고, 리스트 컨테이너·레이아웃 컨테이너 어느 것과도
	// 산출이 다르다 (자식 좌표도 남기지 않는다 — 행 높이만 프레임에 실린다).
	KindTableRow
	// KindTableCell은 `<th>`/`<td>`다 (table.go). 도형이 되지 않는다 — `a:tc`로 프레임 안에
	// 접힌다. 텍스트를 담는 단일 프레임이라(`a:txBody` 하나) 콘텐츠 모델은 `<li>`와 같은
	// modelInline이지만, 잰 셀 박스가 **좌표가 아니라 열 너비·행 높이로** 소비되는 것이
	// `<li>`(측정만 되고 버려진다)와도 다르다 — 그래서 별도 종류다. 헤더·데이터 셀은 종류가
	// 같고 외관 기본값만 갈라진다 (IsHeaderCell).
	KindTableCell
	// KindChart는 `<chart>`다 (chart.go). **렌더러에 닿지 않는 첫 종류다** — 값 층이 도형·
	// 텍스트의 하위 트리로 펼치므로 확정 트리에 남지 않는다 (desugaredKinds).
	//
	// 그래도 별도 종류인 근거는 KindShape·KindTable과 같다: 파서의 콘텐츠 모델이 다르고
	// (점만 담는다), 값 층에서 **다른 기계**가 소비한다(펼침). 종류를 나누지 않으면 파서와
	// 값 층이 각자 `name == "chart"` 비교를 들게 되고, 그것이 카탈로그를 한 곳에 둔 이유를
	// 무너뜨린다.
	KindChart
	// KindChartPoint는 `<point>`다 (chart.go) — void이고, KindChart와 함께 펼쳐진다.
	// `<tr>`처럼 도형이 되지 않지만 갈라지는 자리가 있다: 행은 프레임 안에 `a:tr`로 **접히고**
	// 점은 아예 **사라진다** (점 하나가 도형 1 + 텍스트 2가 된다).
	KindChartPoint
)

// ElementKinds는 카탈로그의 모든 종류다 (KindNone은 카탈로그 밖이므로 빠진다).
// **순서는 상수 선언 순서**이고 그것이 문서·진단·테스트의 순서다.
//
// 목록을 노출하는 이유는 카탈로그를 한 곳에 둔 이유의 연장이다: 렌더러가 다루는 종류
// 집합과 이 카탈로그를 **대조할 수 있어야** "파싱은 되는데 그려지지 않는" 종류가 생기지
// 않는다 (`<table>`이 정확히 그 상태였다). 대조는 render 패키지의 테스트가
// 한다: 여기 있는 종류는 전부 두 렌더러가 다루거나 KindRendered가 거짓이어야 한다.
func ElementKinds() []ElementKind {
	return []ElementKind{
		KindBlock, KindListContainer, KindListItem, KindInline,
		KindLayoutContainer, KindSpacer, KindShape, KindImage,
		KindTable, KindTableRow, KindTableCell,
		KindChart, KindChartPoint,
	}
}

// elementKindNames는 진단·테스트 출력용 이름표다.
var elementKindNames = map[ElementKind]string{
	KindNone:            "None",
	KindBlock:           "Block",
	KindListContainer:   "ListContainer",
	KindListItem:        "ListItem",
	KindInline:          "Inline",
	KindLayoutContainer: "LayoutContainer",
	KindSpacer:          "Spacer",
	KindShape:           "Shape",
	KindImage:           "Image",
	KindTable:           "Table",
	KindTableRow:        "TableRow",
	KindTableCell:       "TableCell",
	KindChart:           "Chart",
	KindChartPoint:      "ChartPoint",
}

// String은 종류 이름을 돌려준다.
func (k ElementKind) String() string {
	if name, ok := elementKindNames[k]; ok {
		return name
	}
	return "Unknown"
}

// ElementNamesOf는 이 종류를 갖는 태그 이름 전부다 (카탈로그 순서). 진단과 테스트가
// "이 종류가 무슨 태그인가"를 이름 목록을 따로 들지 않고 물을 수 있게 한다.
func ElementNamesOf(kind ElementKind) []string {
	var names []string
	for _, name := range allElementNames() {
		if elementKinds[name] == kind {
			names = append(names, name)
		}
	}
	return names
}

// allElementNames는 종류를 갖는 이름 전부다 — 카탈로그 순서(종류 선언 순서 안에서 각
// 목록의 순서)를 유지하려고 맵을 훑지 않고 목록을 이어 붙인다.
func allElementNames() []string {
	names := make([]string, 0, len(elementKinds))
	names = append(names, blockElements...)
	names = append(names, listContainerElements...)
	names = append(names, ListItemElement)
	names = append(names, inlineElements...)
	names = append(names, layoutContainerElements...)
	names = append(names, SpacerElement, ShapeElement, ImageElement, TableElement, TableRowElement)
	names = append(names, tableCellElements...)
	return append(names, ChartElement, ChartPointElement)
}

// unrenderedKinds는 **파서와 값 층은 아는데 어느 렌더러도 그리지 못하는** 종류다.
//
// **지금은 비어 있다** — 카탈로그의 종류 전부가 두 렌더러에 닿는다 (표가 마지막이었다:
// render/web/table.go · render/pptx/table.go). 목록을 남겨 두는 이유는 어휘가 층마다 다른
// 속도로 자란다는 것이다: 새 태그는 카탈로그와 값 해석부터 열리고 렌더러가 뒤따르는 순서로
// 들어오므로 그 사이의 상태를 말할 자리가 필요하다. 목록을
// **언어 층에 명시**해 두면 세 가지가 동시에 성립한다 — (1) 파서가 저자에게 즉시 말할 수 있고
// (checkRendered), (2) 렌더러 디스패치와 대조하는 테스트가 "빠뜨린 것"과 "적어 두고 구현한
// 것"을 둘 다 잡고(render/vocabulary_test.go), (3) 렌더러를 구현하는 커밋이 여기서 종류를
// 지우지 않으면 그 대조가 실패한다.
//
// 목록이 빈 동안 관문 자체는 목록을 잠시 세워 보는 테스트가 지킨다 (withUnrenderedKinds).
var unrenderedKinds []ElementKind

// PlannedAttrMessage는 "이름은 맞는데 아직 구현되지 않은 속성" 진단의 핵 문구다.
// UnrenderedVocabMessage와 같은 이유로 상수다 — 진단을 내는 곳과 걸러 보는 곳이 같은
// 문자열을 써야 한다.
const PlannedAttrMessage = "is an attribute that is not implemented yet"

// plannedAttr는 어휘 표에 **아직 없지만 이름이 이미 정해진** 속성 하나다. `Opens`는 어느
// 항목이 그 속성을 여는지다 — 저자가 기다릴 곳을 알아야 진단이 끝난 문장이 된다.
type plannedAttr struct {
	Tag   string
	Name  string
	Opens string
}

// plannedAttrs는 계획된 속성의 **닫힌** 목록이다.
//
// 이 목록이 있는 이유: 이름은 맞는데 구현이 아직인 속성(`<pre lang="go">`)이 "알 수 없는
// 속성"으로 죽으면 저자가 **이름을 의심하며** 여러 번 바꿔 본다 — 없는 것은 이름이 아니라
// 구현이다. 목록 밖 이름은 지금처럼 "알 수 없는 속성"으로 말한다: 오타와 예정을 한 문장으로
// 뭉치면 둘 다 흐려진다.
//
// 목록은 **썩을 수 있다** — 속성을 구현하는 커밋이 여기서 줄을 지우지 않으면 저자는 이미
// 되는 것을 "아직 안 된다"고 듣는다. 그래서 어휘 표와 양방향으로 대조한다 (planned_test.go —
// unrenderedKinds ↔ 렌더러 디스패치와 같은 장치).
//
// 목록이 비는 날에도 관문은 목록을 잠시 세워 보는 테스트가 지킨다 (withPlannedAttrs).
// 지금 이 목록은 **비어 있다** — 그것이 이 장치의 정상 상태다: 목록은 약속이고 약속은
// 갚으면 사라진다.
var plannedAttrs = []plannedAttr{}

// desugaredKinds는 **값 층이 도형·텍스트로 펼치는** 종류다.
//
// unrenderedKinds의 거울이지만 뜻이 반대다: 저것은 **빚**이고(렌더러가 뒤따르면 줄이 사라진다)
// 이것은 **계약**이다 — `<chart>`는 렌더러를 영원히 갖지 않는다. 차트를 도형으로 펼친 결정의
// 값은 "산출물에 차트라는 물건이 없다"는 것이므로, 어느 렌더러가 이 종류를 다루게 되는 날은
// 그 결정이 깨진 날이다. 그래서 이 목록의 종류는 렌더러 능력 표에 **있으면 실패**다
// (render/vocabulary_test.go의 셋째 갈래).
//
// 목록이 언어 층에 있는 이유는 unrenderedKinds와 같다 — 파서가 아는 종류의 운명을 파서 옆에
// 적어야 세 층의 대조가 한 곳을 읽는다. 값 층도 이 목록을 읽는다: 확정 트리에 펼쳐지는 종류가
// 하나도 없다는 것을 core의 테스트가 이 목록으로 확인한다 (TestResolvedTreeHasNoDesugaredKinds).
var desugaredKinds = []ElementKind{KindChart, KindChartPoint}

// DesugaredKinds는 펼쳐지는 종류 전부다 (카탈로그 순서). ElementKinds의 부분집합이다.
func DesugaredKinds() []ElementKind {
	kinds := make([]ElementKind, len(desugaredKinds))
	copy(kinds, desugaredKinds)
	return kinds
}

// KindDesugared는 이 종류가 값 층에서 펼쳐지는지 보고한다 — 참이면 확정 트리에 남지 않고
// 렌더러에 닿지 않는다.
func KindDesugared(kind ElementKind) bool {
	for _, k := range desugaredKinds {
		if k == kind {
			return true
		}
	}
	return false
}

// KindRendered는 두 렌더러가 이 종류를 그릴 수 있는지 보고한다. KindNone(카탈로그 밖 —
// `<slide>`·`<text>`·`<note>`)은 각자의 규칙을 따르므로 참이다.
//
// 펼쳐지는 종류(KindDesugared)도 **참이다** — 그 종류는 그려진다, 다만 자기 이름으로 그려지지
// 않는다. 두 물음을 한 함수에 뭉치면 "렌더러가 다뤄야 하는가"와 "저자에게 아직 안 된다고
// 말해야 하는가"가 섞인다: 차트는 저자가 지금 쓸 수 있는 어휘이므로 checkRendered가 막아서는
// 안 된다. 렌더러가 다뤄야 하는 종류는 두 물음의 곱이다 (KindRendered && !KindDesugared).
func KindRendered(kind ElementKind) bool {
	for _, k := range unrenderedKinds {
		if k == kind {
			return false
		}
	}
	return true
}

// 종류별 이름 목록. **순서가 곧 진단 메시지와 문서의 순서다** — 맵이 아니라 슬라이스인
// 이유이고, elementKinds 맵은 여기서 파생된다.
var (
	blockElements         = []string{"h1", "h2", "h3", "p", "blockquote", "pre"}
	listContainerElements = []string{"ul", "ol"}
	inlineElements        = []string{"strong", "em", "code", "a", "br", "span", MarkElement}
)

// ListItemElement는 리스트 항목 태그 이름이다.
const ListItemElement = "li"

// RevealAttr는 리스트 항목의 **순차 등장 스텝**이다 — `<li>`가 여는
// 유일한 속성이고, 값을 생략하면 그 슬라이드에서 자기 차례를 받는다 (`reveal="2"`는 그
// 자리를 덮어쓴다). 값 어휘(1 이상 정수)는 값 층의 것이고 파서는 이름만 본다.
//
// 이 속성이 외관 어휘가 아니라 **발표 순서**라는 점이 중요하다: 숨김은 발표자 뷰 하나의
// 성질이고, 문서 뷰·측정·인쇄·PPTX는 전부 전개된 상태를 본다.
const RevealAttr = "reveal"

// MarkElement는 형광펜 태그 이름이다 — 색은 테마의 것이라 속성이 없고, 값 층이 런의
// Highlight를 채운다. 웹 렌더러가 이 이름에만 배경을 칠한다 (안의 런은 같은 색을 물려받는다).
const MarkElement = "mark"

// BreakElement는 줄바꿈 태그 이름이다. 서브셋의 유일한 void 요소다 — 내용을 갖지 않고
// `<br/>`(자기 닫힘)로만 쓴다. 렌더러가 런 분할에서 이 이름을 특별 취급한다.
const BreakElement = "br"

// PreElement는 코드 블록 태그 이름이다. 서브셋에서 **문자 데이터만** 담는 유일한 블록이고
// (modelPre), 공백·개행이 내용이다 — 값 층과 두 렌더러가 이 이름을 특별 취급한다.
const PreElement = "pre"

// LangAttr는 `<pre>`의 코드 언어다 — 하이라이팅을 켜는 **유일한** 어휘이고,
// 값은 엔진이 닫은 목록(go·xml·soml·json·bash)이다. 색·줄 번호·테마 이름 같은 손잡이는
// 열지 않는다: 종류 → 색은 테마의 것이고, 저자가 코드마다 색을 고르면 덱이 팔레트를 잃는다.
const LangAttr = "lang"

// elementKinds는 이름 → 종류 색인이다. 위의 순서 있는 목록과 레이아웃·도형·이미지 카탈로그
// (layout.go·shape.go·image.go)에서 파생된다 — 종류를 갖는 이름은 전부 여기를 지난다.
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
	kinds[ShapeElement] = KindShape
	kinds[ImageElement] = KindImage
	kinds[TableElement] = KindTable
	kinds[TableRowElement] = KindTableRow
	for _, name := range tableCellElements {
		kinds[name] = KindTableCell
	}
	kinds[ChartElement] = KindChart
	kinds[ChartPointElement] = KindChartPoint
	return kinds
}

// KindOf는 태그 이름의 종류다. 카탈로그 밖의 이름은 KindNone이다.
func KindOf(name string) ElementKind {
	return elementKinds[name]
}

// IsVoidElement는 이 이름이 void 요소(자식·문자 데이터를 갖지 않는 요소)인지 보고한다.
// `<br>`(런 분할)·`<spacer>`(밀어내기)·`<image>`(그림)가 그것이다 — 전부 자기 닫힘으로만
// 쓴다. 그림이 void인 근거는 `<image src="…" />` 하나가 전부이고 담을 것이 없다는 것이다:
// 캡션은 `<image>`의 자식이 아니라 **형제 `<text>`**이고, 그림 안 글자는 도형의 일이며
// (`<shape>` 안 `<text>`) 그림 **위** 글자는 겹침(`<stack>`)의 일이다.
func IsVoidElement(name string) bool {
	return name == BreakElement || name == SpacerElement || name == ImageElement ||
		name == ChartPointElement
}

// TagTextRole은 블록·리스트 항목 태그의 텍스트 역할이다 — 역할 등가 표
// (`<p>` ≡ `<text role="body">`, `<h2>` ≡ `<text role="heading">`).
//
// 이 표가 테마(core/themes/*/components.json)가 아니라 **언어**(soml)에 있는 이유: 등가는 언어가 정한
// 계약이지 테마의 선택이 아니다. 테마가 `<p>`를 caption으로 바꿀 수 있으면 같은 문서가
// 테마마다 다른 역할로 해석되어 "같은 계약의 두 표기"가 성립하지 않는다. 태그별
// **기본값**(h3의 크기, blockquote의 색)은 외관이므로 테마가 소유한다 — 역할과 기본값의
// 소유가 갈라지는 경계가 여기다 (테마의 components.json은 기본값만 담는다).
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
