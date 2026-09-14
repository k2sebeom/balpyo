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
