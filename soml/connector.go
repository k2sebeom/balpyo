package soml

// 이 파일은 연결선(`<connector>`)의 **요소 종류·속성 어휘 카탈로그**다.
//
// 연결선은 다른 카탈로그와 갈라지는 자리가 둘 있다. (1) **재지 않는다** — 흐름에 자리를 차지하지
// 않고, 끝점은 측정이 끝난 뒤 두 상자의 잰 rect에서 유도된다. (2) **슬라이드 직속뿐이다** —
// 렌더러가 슬라이드마다 오버레이 하나(웹) · 콘텐츠 도형 뒤(PPTX)에 문서 순서로 그리므로, 흐름 안에
// 놓을 자리가 없다.

// ConnectorElement는 연결선 태그 이름이다 (KindConnector). void다 — 선은 담을 것이 없다.
const ConnectorElement = "connector"

// 연결선 값 어휘. 문법은 파서가, 값은 값 층이 본다 — 목록은 언어가 갖는다 (shapeTypes와 같은 분담).
var (
	// connectorRoutes는 경로 종류다 (순서 = 진단·문서 순서). 첫 이름이 기본값이다.
	connectorRoutes = []string{"straight", "elbow", "curved"}
	// connectorSides는 붙는 면이다. `auto`는 상대 상자를 향하는 면이다 (기본값).
	connectorSides = []string{"auto", "top", "right", "bottom", "left"}
	// connectorEnds는 양 끝 표식이다.
	connectorEnds = []string{"none", "arrow", "dot"}
)

// connectorAttrs는 `<connector>`의 속성이다 (순서 = 진단·문서 순서).
//
//   - `from`·`to`: 잇는 두 상자의 `id` (같은 슬라이드의 `<shape>`·`<image>`). **필수**다.
//   - `route`: 경로 종류. `from-side`·`to-side`: 붙는 면.
//   - `line`: 선 — 도형 `border`와 같은 문법(`<굵기> <스타일> <색>`). `start`·`end`: 양 끝 표식.
//   - `variant`: 테마가 이름 붙인 선 번들.
//   - `label`: 선 옆의 짧은 글 (`reboot`). 한 줄이고, 경로 길이의 가운데에 앉는다.
//
// 박스 · 배치 속성이 없다: 연결선의 자리는 두 상자가 정한다.
var connectorAttrs = []string{"from", "to", "route", "from-side", "to-side", "line", "start", "end", "variant", "label"}

// ConnectorAttrs는 `<connector>`의 속성이다. 값 층이 이 목록을 소비한다 (ShapeAttrs와 같은 이유).
func ConnectorAttrs() []string {
	attrs := make([]string, len(connectorAttrs))
	copy(attrs, connectorAttrs)
	return attrs
}

// ConnectorRoutes · ConnectorSides · ConnectorEnds는 값 어휘다 (진단과 문서 게이트가 같은 목록을 읽는다).
func ConnectorRoutes() []string { return append([]string(nil), connectorRoutes...) }
func ConnectorSides() []string  { return append([]string(nil), connectorSides...) }
func ConnectorEnds() []string   { return append([]string(nil), connectorEnds...) }

// ConnectorEnd는 연결선 끝의 표식이다.
type ConnectorEnd string

const (
	EndNone  ConnectorEnd = "none"
	EndArrow ConnectorEnd = "arrow"
	EndDot   ConnectorEnd = "dot"
)

// ConnectorStyle은 core가 확정한 연결선이다 — 렌더러는 이것만 읽는다 (Attrs를 다시 해석하지 않는다).
type ConnectorStyle struct {
	// From · To는 잇는 두 상자다 (같은 슬라이드의 `<shape>`·`<image>` 요소). 렌더러는 이 요소의 잰
	// rect로 끝점을 계산한다.
	From, To *Element
	// Route는 경로 종류(straight · elbow · curved), FromSide · ToSide는 면이다 (빈 값은 자동).
	Route            string
	FromSide, ToSide string
	// Line은 선이다 (굵기 · 스타일 · 색). None은 확정 값에 오지 않는다 — 선 없는 연결선은 값 층의 에러다.
	Line Border
	// Start · End는 양 끝 표식이다.
	Start, End ConnectorEnd
	// Label은 선 라벨의 글이다 (빈 값은 라벨 없음). LabelStyle은 그 글의 확정 외관이다 — caption 역할
	// 위에 테마의 `connector` 태그 기본값이 얹힌 것. 라벨은 한 줄이고, 지오메트리 패스가 그 크기를
	// 재고(연결선의 요소 식별자로) 두 렌더러가 경로 옆 같은 자리에 놓는다 (route.Path.LabelRect).
	Label      string
	LabelStyle *TextStyle
}

// HasConnectorLabel은 요소가 라벨 있는 확정 연결선인지다 — 지오메트리 패스가 그 라벨을 잰다.
func HasConnectorLabel(elem *Element) bool {
	return elem.Name == ConnectorElement && elem.Appearance != nil && elem.Appearance.Connector != nil &&
		elem.Appearance.Connector.Label != ""
}
