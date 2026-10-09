package core

import (
	"strings"
	"testing"

	"github.com/k2sebeom/balpyo/soml"
)

// slotOf는 덱에서 이름이 tag인 첫 요소의 확정 슬롯을 돌려준다 (Appearance 없으면 빈 문자열).
func slotOf(t *testing.T, deck *soml.Deck, tag string) string {
	t.Helper()
	elem := findElement(t, deck, tag)
	if elem.Appearance == nil {
		return ""
	}
	return elem.Appearance.Slot
}

// 슬라이드 직계 자식의 유효한 slot은 진단 없이 요소에 실린다 (Appearance.Slot). 레이아웃마다
// 어휘가 다르므로(soml.LayoutSlots) 대표 레이아웃 몇 개를 확인한다.
func TestResolveSlotConfirmed(t *testing.T) {
	tests := []struct {
		name string
		src  string
		tag  string
		want string
	}{
		{"content body", `<slide layout="content"><p slot="body">x</p></slide>`, "p", "body"},
		{"content title", `<slide layout="content"><h1 slot="title">T</h1></slide>`, "h1", "title"},
		{"title 표지 제목", `<slide layout="title"><h1 slot="title">T</h1></slide>`, "h1", "title"},
		{"two-column left", `<slide layout="two-column"><col slot="left"><p>x</p></col></slide>`, "col", "left"},
		{"stat value", `<slide layout="stat"><p slot="value">42</p></slide>`, "p", "value"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			deck, diags := resolveSlide(t, tc.src)
			if len(diags) != 0 {
				t.Fatalf("진단이 없어야 한다: %v", diags)
			}
			if got := slotOf(t, deck, tc.tag); got != tc.want {
				t.Errorf("<%s> 슬롯 = %q, want %q", tc.tag, got, tc.want)
			}
		})
	}
}

// slot을 쓰지 않은 직계 자식은 빈 슬롯이다 — 이 값이 곧 "슬롯 배치 없음"이라, 슬롯을 쓰지
// 않은 덱의 산출물이 이 필드 때문에 달라지지 않는다 (골든 불변).
func TestResolveSlotOmitted(t *testing.T) {
	deck, diags := resolveSlide(t, `<slide layout="content"><p>x</p></slide>`)
	if len(diags) != 0 {
		t.Fatalf("진단이 없어야 한다: %v", diags)
	}
	if got := slotOf(t, deck, "p"); got != "" {
		t.Errorf("slot 없는 직계 자식의 슬롯 = %q, want \"\"", got)
	}
}

// 레이아웃 어휘 밖 슬롯 이름은 **line:col을 가진** 진단이다 — 구간이 값을 가리키고 허용
// 목록이 어휘 순서(soml.SlotNames)로 실린다. 폴백하지 않는 것이 계약이다 (layout과 같은
// 태도, ADR 0032): `slot="titel"` 오타가 조용히 기본 슬롯으로 떨어지면 콘텐츠가 엉뚱한
// 자리에 놓이고 그 실패는 렌더를 볼 때까지 안 보인다.
func TestResolveSlotUnknown(t *testing.T) {
	// 값 `bogus`는 `<slide layout="content"><p slot="`(33칸) 다음 34칸에서 시작한다.
	deck, diags := resolveSlide(t, `<slide layout="content"><p slot="bogus">x</p></slide>`)
	if len(diags) != 1 {
		t.Fatalf("진단 %d개, want 1개: %v", len(diags), diags)
	}
	diag := diags[0]
	if !diag.Span.Start.IsValid() || diag.Span.Start.Line != 1 || diag.Span.Start.Col != 34 {
		t.Errorf("위치 = %s, want 1:34 (%s)", diag.Span.Start, diag.Message)
	}
	if diag.Severity != soml.SeverityError || !strings.Contains(diag.Message, "없는 슬롯") {
		t.Errorf("진단 = %q, want '없는 슬롯' 에러", diag.String())
	}
	// 허용 목록이 어휘 순서(content: title, body)로 실린다.
	if !strings.Contains(diag.Message, "title") || !strings.Contains(diag.Message, "body") {
		t.Errorf("메시지에 허용 슬롯 목록이 없다: %q", diag.Message)
	}
	// 슬롯이 확정되지 않았다 (회복: 슬롯 없음).
	if got := slotOf(t, deck, "p"); got != "" {
		t.Errorf("어휘 밖 슬롯 뒤 슬롯 = %q, want \"\"", got)
	}
}

// 슬롯이 없는 레이아웃(canvas)의 slot도 같은 경로로 잡힌다 — LayoutSlot이 false를 준다.
func TestResolveSlotOnSlotlessLayout(t *testing.T) {
	_, diags := resolveSlide(t, `<slide layout="canvas"><p slot="body">x</p></slide>`)
	if len(diags) != 1 {
		t.Fatalf("진단 %d개, want 1개: %v", len(diags), diags)
	}
	if diags[0].Severity != soml.SeverityError || !strings.Contains(diags[0].Message, "없는 슬롯") {
		t.Errorf("진단 = %q, want '없는 슬롯' 에러", diags[0].String())
	}
}

// 슬롯은 슬라이드 **직계 자식에만** 유효하다 — 중첩 요소의 slot은 뜻이 없으므로 진단이다.
// 조용히 무시하면 저자는 콘텐츠가 왜 그 영역에 안 가는지 알 수 없다.
func TestResolveSlotNested(t *testing.T) {
	// 중첩 <p slot="body">의 값 `body`는 34칸에서 시작한다:
	// `<slide layout="content"><row><p slot="`의 마지막 " 다음.
	deck, diags := resolveSlide(t, `<slide layout="content"><row><p slot="body">x</p></row></slide>`)
	if len(diags) != 1 {
		t.Fatalf("진단 %d개, want 1개: %v", len(diags), diags)
	}
	diag := diags[0]
	if diag.Severity != soml.SeverityError || !strings.Contains(diag.Message, "직계 자식") {
		t.Errorf("진단 = %q, want '직계 자식' 에러", diag.String())
	}
	// 중첩 요소에는 슬롯이 실리지 않는다.
	if got := slotOf(t, deck, "p"); got != "" {
		t.Errorf("중첩 요소의 슬롯 = %q, want \"\"", got)
	}
}

// 테마가 그 레이아웃의 슬롯 영역을 갖지 않으면 슬롯은 **경고 + 미탑재**다 (ADR 0044 D3).
//
// 이 판정이 core에 있는 이유가 결정의 전부다: 렌더러 두 곳이 각자 조건을 적었을 때 그 둘이
// 갈라졌다 (웹은 `Theme.SlotLayouts`를, PPTX는 슬롯 탑재 여부만 봤다). 확정 모델에 싣지
// **않는** 쪽을 고르면 판정이 한 번만 일어나고 두 렌더러는 이미 그 하나의 입력을 읽는다.
//
// 에러가 아니라 경고인 이유: 슬롯 이름은 어휘에 있고 오타도 아니다 — 부족한 것은 테마이며
// 테마를 바꾸면 같은 소스가 그대로 뜻을 얻는다 (모르는 variant의 선례, ADR 0067 결정 6).
// `--strict`가 승격한다.
func TestResolveSlotWarnsWhenThemeHasNoArea(t *testing.T) {
	doc, diags := soml.Parse("slides/001_slots.slide", `<slide layout="content"><p slot="body">x</p></slide>`)
	if err := diags.Err(); err != nil {
		t.Fatalf("파싱 실패: %v", err)
	}
	elem := doc.Content[0].(*soml.Element)
	// 슬롯 영역을 하나도 갖지 않은 테마다 — minimal이 11종을 다 채운 뒤에도 외부 테마·
	// 부분 구현 테마가 이 자리에 온다.
	r := resolver{
		theme:  soml.Theme{SlotLayouts: map[string]string{}},
		file:   "slides/001_slots.slide",
		layout: soml.LayoutContent,
	}
	r.resolveSlot(elem, nil)

	if len(r.diags) != 1 {
		t.Fatalf("진단 %d개, want 1개: %v", len(r.diags), r.diags)
	}
	diag := r.diags[0]
	if diag.Severity != soml.SeverityWarning {
		t.Errorf("심각도 = %v, want 경고 (테마 교체로 풀리는 이식성 문제다)", diag.Severity)
	}
	if !strings.Contains(diag.Message, "테마") {
		t.Errorf("진단이 테마를 짚지 않는다: %q", diag.Message)
	}
	// 값 구간을 짚는다 — 진단이 부호가 아니라 슬롯 이름에 밑줄을 긋는다.
	if !diag.Span.Start.IsValid() {
		t.Error("진단에 위치가 없다")
	}
	// **슬롯이 실리지 않았다**: 이 한 줄이 두 렌더러의 합의 지점이다.
	if elem.Appearance != nil && elem.Appearance.Slot != "" {
		t.Errorf("슬롯 = %q, want \"\" — 테마가 못 받는 슬롯은 확정 모델에 실리지 않는다", elem.Appearance.Slot)
	}
}

// 테마가 영역을 가진 레이아웃은 경고 없이 그대로 실린다 (위 테스트의 짝 — 조건이 뒤집히지
// 않았는지 본다).
func TestResolveSlotQuietWhenThemeHasArea(t *testing.T) {
	deck, diags := resolveSlide(t, `<slide layout="content"><p slot="body">x</p></slide>`)
	if len(diags) != 0 {
		t.Fatalf("minimal은 content 슬롯 영역을 갖는다 — 진단이 없어야 한다: %v", diags)
	}
	if got := slotOf(t, deck, "p"); got != "body" {
		t.Errorf("슬롯 = %q, want \"body\"", got)
	}
}

// 슬롯은 **도형·그림에도** 실린다 — 두 요소의 속성 순회가 slot을 모르면 저자는 "엔진 버그"
// 진단을 받는다 (catalogGap).
//
// 이 자리가 조용히 깨져 있던 이유: 파서는 slot을 childPlacementAttrs **밖에서** 잇는데
// (withChildPlacement의 마지막 한 칸 — core의 Placement 해석과 섞이지 않게 한 의도다)
// shapeAttrs·imageAttrs의 exhaustive 검사는 배치 목록만 면제해 줬다. 그래서 `slot`을 쓴
// 도형이 어휘·테마를 다 만족해도 빌드가 멈췄고, 그 결과 media 슬롯을 가진 두 레이아웃
// (image-full·image-text)은 **테마에 영역이 있어도 저작할 수 없었다**.
//
// 텍스트 요소가 멀쩡했던 것이 이 버그를 오래 숨겼다: 텍스트 경로에는 그 exhaustive 검사가
// 없어서 슬롯 테스트 전부가 통과했다.
func TestResolveSlotOnShapeAndImage(t *testing.T) {
	tests := []struct {
		name string
		src  string
		tag  string
		want string
	}{
		{"도형", `<slide layout="image-text"><shape type="rect" h="240" slot="media"/></slide>`, "shape", "media"},
		{"그림", `<slide layout="image-full"><image src="a.png" slot="media"/></slide>`, "image", "media"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			deck, diags := resolveSlide(t, tc.src)
			if len(diags) != 0 {
				t.Fatalf("진단이 없어야 한다: %v", diags)
			}
			if got := slotOf(t, deck, tc.tag); got != tc.want {
				t.Errorf("<%s> 슬롯 = %q, want %q", tc.tag, got, tc.want)
			}
		})
	}
}
