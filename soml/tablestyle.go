package soml

// 이 파일은 **확정된 표 외관**이다 — 카탈로그(table.go)가 이름·속성 어휘를 소유하고 여기가
// 값의 모양을 소유한다 (shape.go ↔ shapestyle.go와 같은 나눔).
//
// Appearance·ShapeStyle과 같은 계약을 진다: **렌더러는 이 구조체만 읽고 Attrs를 다시
// 해석하지 않는다.** 병합(테마 header/본문/줄무늬 기본값 위에 인라인 셀 속성) · 색 · 정렬 ·
// 병합 span은 core가 한 번만 확정한다. 표 외관이 값 층을 두 번 지나면 웹의 `<table>`과 PPTX의
// `a:tbl`이 갈라지고, 그 종류의 버그는 두 산출물을 나란히 볼 때까지 드러나지 않는다.

// TableStyle은 확정된 표 **프레임** 외관이다 (ADR 0036).
//
// 프레임 자체에 실을 외관은 거의 없다 — 좌표는 잰 프레임 박스가, 셀 채움·테두리·텍스트는
// CellStyle이 소유한다 (줄무늬조차 셀 채움으로 명시로 굽는다 — "스타일 없음 + 명시 tcPr").
// 그래도 표시자로 두는 이유는 Shape·Image와 같다: 렌더러가 "해석을 끝낸 표"임을 KindOf 없이
// Appearance로 확인하고, 웹이 변형 훅(`balpyo-table-<변형>`)을 낼 근거를 여기서 읽는다.
type TableStyle struct {
	// Variant는 확정된 프레임 변형 이름이다 (`<table variant="striped">`). 어휘 검증·폴백은
	// core가 했고(모르는 변형은 경고 + 빈 값), 여기 남는 것은 테마에 있는 이름뿐이다. 웹은
	// 이 이름으로 CSS 훅을 낼 수 있고, PPTX는 읽지 않는다 — 명시 tcPr이 외관을 소유하므로
	// 셀 채움이 이미 줄무늬를 담고 있다.
	Variant string
}

// CellStyle은 확정된 셀 외관이다 (`<th>`/`<td>`). 박스(채움·테두리)와 텍스트를 함께 담는다 —
// OOXML `a:tc`가 `tcPr`(채움·테두리)와 `txBody`(텍스트)를 함께 갖는 것과 같은 모양이고, 웹의
// `<th>`/`<td>`도 배경·테두리와 글자 스타일을 한 셀에 싣는다.
//
// "정하지 않음"은 포인터의 nil로 나타낸다 (ShapeStyle의 Fill·Border와 같은 근거): 인라인 셀
// 속성이 없으면 테마 header/본문 기본값이 채운 값이 그대로 남고, 아무 기본값도 없는 채널은
// nil로 남아 렌더러가 선언을 내지 않는다.
type CellStyle struct {
	// Header는 이 셀이 `<th>`인지다. 웹은 `<th>`/`<td>` 태그를, PPTX는 헤더 힌트를 이 값으로
	// 가른다 — 외관 자체는 이미 Fill·Border·Text에 확정되어 있으므로 이 불리언은 태그 선택용이다.
	Header bool
	// Fill은 셀 배경이다 — 색(토큰·리터럴) 또는 `none`, 정하지 않으면 nil. 헤더 채움·줄무늬
	// 채움·인라인 `fill`이 모두 이 자리에 확정된다 (줄무늬를 명시로 굽는 자리다 — ADR 0036).
	Fill *Fill
	// Border는 셀 테두리다 — 축약 3조각 또는 `none`, 정하지 않으면 nil. 도형 테두리와 같은
	// 구조를 공유한다 (같은 값 문법이 자리마다 다르게 읽히지 않게).
	Border *Border
	// Text는 셀 텍스트 런의 확정 외관이다 — body 역할 기본값 위에 헤더 굵기·색·인라인
	// `color`/`align`/`anchor`가 얹힌 것이다. 셀 안 인라인 태그(strong/em/…)의 런은 이 스타일을
	// 상속한다 (resolveInlines). 텍스트 전용 콘텐츠 모델이라(ADR 0036) 셀은 언제나 이 하나를 갖는다.
	Text *TextStyle
	// ColSpan·RowSpan은 병합 칸 수다 (`colspan`/`rowspan`, 없으면 1). 구조 정보지만 여기 담는
	// 이유는 "렌더러가 Attrs를 다시 읽지 않는다"는 계약이다: 웹은 `<td colspan>` 속성으로,
	// PPTX는 `a:tc@gridSpan`/`@rowSpan`과 병합에 덮이는 칸의 `hMerge`/`vMerge` 빈 `a:tc`로 낸다.
	ColSpan int
	RowSpan int
}

// TableStyle은 이 요소의 확정 표 프레임 외관이다 — 표가 아니거나 해석 전이면 nil이다
// (ShapeStyle()·TextStyle()과 같은 이유로 요소에 직접 둔다).
func (e *Element) TableStyle() *TableStyle {
	if e.Appearance == nil {
		return nil
	}
	return e.Appearance.Table
}

// CellStyle은 이 요소의 확정 셀 외관이다 — 셀이 아니거나 해석 전이면 nil이다.
func (e *Element) CellStyle() *CellStyle {
	if e.Appearance == nil {
		return nil
	}
	return e.Appearance.Cell
}
