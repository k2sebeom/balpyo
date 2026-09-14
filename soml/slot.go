package soml

// 슬롯 카탈로그 — 레이아웃별 슬롯 어휘와 각 슬롯의 PPTX 플레이스홀더 매핑 (PRD §7.3, ADR 0035).
//
// SlideLayout이 슬라이드의 **구조 어휘**이듯(deck.go) 슬롯도 닫힌 구조 어휘다: 레이아웃마다
// 어떤 슬롯 이름이 유효한지는 PRD §7.3의 고정 카탈로그이고, 어휘 밖 이름(`slot="titel"`)은
// core가 잡는 **빌드 에러**다 (ADR 0032의 layout과 같은 태도 — 오타가 조용히 기본 슬롯으로
// 떨어지면 콘텐츠가 엉뚱한 자리에 놓이고 그 실패는 렌더를 볼 때까지 안 보인다). 슬롯→
// 플레이스홀더 타입/idx 매핑도 표준이라(테마가 아니라 언어가 정한다) 여기 있다 — 슬롯의
// **시각 위치**(grid-area)만 테마가 소유한다 (layouts/*.css).

// PhType은 OOXML 플레이스홀더 타입이다 (`p:ph@type`, ST_PlaceholderType의 부분집합).
//
// 어휘를 넷으로 좁힌 이유: v1 슬롯은 전부 텍스트(제목류·본문류)이거나 그림이다. 제목류는
// PowerPoint가 개요 보기·"제목을 클릭해 입력"에서 특별 취급하므로 본문과 구분해야 하고
// (title/ctrTitle/subTitle), 나머지 텍스트 슬롯은 전부 body다 — 이름(left/value/caption…)이
// 달라도 PPTX 플레이스홀더 종류는 같고 idx로만 갈린다. 그림 슬롯(media)은 pic이다.
const (
	// PhCtrTitle은 표지의 가운데 제목이다 (title 레이아웃의 title 슬롯). PowerPoint가 표지
	// 슬라이드에서 쓰는 특수 제목이라 일반 title과 구분한다.
	PhCtrTitle = "ctrTitle"
	// PhTitle은 일반 제목이다 (표지가 아닌 레이아웃의 title 슬롯).
	PhTitle = "title"
	// PhSubTitle은 부제다 (subtitle 슬롯).
	PhSubTitle = "subTitle"
	// PhBody는 본문류 텍스트 슬롯 전부다 (body·left·right·value·label·caption·code·quote·
	// attribution·contact·kicker·author·date). idx로 서로 갈린다.
	PhBody = "body"
	// PhPic은 그림 슬롯이다 (media). `<image slot="media">`가 채우는 그림 플레이스홀더.
	PhPic = "pic"
)

// Slot은 레이아웃 슬롯 하나다 — 이름(저자가 쓰는 것)과 PPTX 플레이스홀더 매핑(phType·idx).
type Slot struct {
	// Name은 PRD §7.3의 슬롯 이름이다 (`slot="title"`의 값).
	Name string
	// PhType은 이 슬롯이 되는 OOXML 플레이스홀더 타입이다 (`p:ph@type`).
	PhType string
	// PhIndex는 `p:ph@idx`다 — **0이면 idx 속성을 생략한다** (title/ctrTitle의 관례: 제목
	// 플레이스홀더는 idx를 갖지 않는다). 나머지는 레이아웃 안에서 유일한 양수여야 한다 —
	// 한 레이아웃에 idx가 겹치는 플레이스홀더 둘이면 PowerPoint가 하나를 지운다.
	PhIndex int
}

// layoutSlots는 레이아웃별 슬롯 카탈로그다 (PRD §7.3 표). 목록 순서 = 진단이 허용 슬롯을
// 나열하는 순서 = PRD 표 순서. canvas는 슬롯이 없다 (자유 배치 — 저자가 배치를 소유한다).
//
// idx 배정: 제목류(ctrTitle/title)는 0(속성 생략), subTitle은 1, 본문·그림류는 2부터
// 목록 순서로 유일하게 준다. 관례를 따르되 유일성만 지키면 되므로 값 자체는 결정적이면 된다.
var layoutSlots = map[SlideLayout][]Slot{
	LayoutTitle: {
		{Name: "kicker", PhType: PhBody, PhIndex: 2},
		{Name: "title", PhType: PhCtrTitle, PhIndex: 0},
		{Name: "subtitle", PhType: PhSubTitle, PhIndex: 1},
		{Name: "author", PhType: PhBody, PhIndex: 3},
		{Name: "date", PhType: PhBody, PhIndex: 4},
	},
	LayoutSectionDivider: {
		{Name: "title", PhType: PhTitle, PhIndex: 0},
		{Name: "subtitle", PhType: PhSubTitle, PhIndex: 1},
	},
	LayoutContent: {
		{Name: "title", PhType: PhTitle, PhIndex: 0},
		{Name: "body", PhType: PhBody, PhIndex: 1},
	},
	LayoutTwoColumn: {
		{Name: "title", PhType: PhTitle, PhIndex: 0},
		{Name: "left", PhType: PhBody, PhIndex: 1},
		{Name: "right", PhType: PhBody, PhIndex: 2},
	},
	LayoutImageFull: {
		{Name: "media", PhType: PhPic, PhIndex: 1},
		{Name: "caption", PhType: PhBody, PhIndex: 2},
	},
	LayoutImageText: {
		{Name: "title", PhType: PhTitle, PhIndex: 0},
		{Name: "media", PhType: PhPic, PhIndex: 1},
		{Name: "body", PhType: PhBody, PhIndex: 2},
	},
	LayoutQuote: {
		{Name: "quote", PhType: PhBody, PhIndex: 1},
		{Name: "attribution", PhType: PhBody, PhIndex: 2},
	},
	LayoutCode: {
		{Name: "title", PhType: PhTitle, PhIndex: 0},
		{Name: "code", PhType: PhBody, PhIndex: 1},
		{Name: "caption", PhType: PhBody, PhIndex: 2},
	},
	LayoutStat: {
		{Name: "title", PhType: PhTitle, PhIndex: 0},
		{Name: "value", PhType: PhBody, PhIndex: 1},
		{Name: "label", PhType: PhBody, PhIndex: 2},
		{Name: "body", PhType: PhBody, PhIndex: 3},
	},
	LayoutCanvas: nil,
	LayoutEnd: {
		{Name: "title", PhType: PhTitle, PhIndex: 0},
		{Name: "contact", PhType: PhBody, PhIndex: 1},
	},
}

// LayoutSlots는 레이아웃이 가진 슬롯 목록이다 (PRD §7.3 순서). 슬롯이 없는 레이아웃(canvas)은
// 빈 슬라이스다.
func LayoutSlots(layout SlideLayout) []Slot {
	return layoutSlots[layout]
}

// LayoutSlot은 레이아웃의 이름 붙은 슬롯을 찾는다 — 없으면 (Slot{}, false).
func LayoutSlot(layout SlideLayout, name string) (Slot, bool) {
	for _, slot := range layoutSlots[layout] {
		if slot.Name == name {
			return slot, true
		}
	}
	return Slot{}, false
}

// SlotNames는 레이아웃 슬롯 이름 목록이다 — 없는 슬롯 진단이 허용 어휘를 나열하는 근거다
// (Theme.ColorNames와 같은 태도).
func SlotNames(layout SlideLayout) []string {
	slots := layoutSlots[layout]
	names := make([]string, len(slots))
	for i, slot := range slots {
		names[i] = slot.Name
	}
	return names
}
