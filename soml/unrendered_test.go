package soml

import (
	"strings"
	"testing"
)

// withUnrenderedKinds는 종류 몇 개를 "아직 그리지 못하는 어휘"로 잠시 세운다 — 목록이 빈
// 동안 관문을 시험할 방법은 이것뿐이다 (checkRendered는 KindRendered만 묻으므로 대역이면 된다).
func withUnrenderedKinds(t *testing.T, kinds ...ElementKind) {
	t.Helper()
	prev := unrenderedKinds
	unrenderedKinds = kinds
	t.Cleanup(func() { unrenderedKinds = prev })
}

// 그리지 못하는 어휘는 **파스 시점에** 막힌다 — 다 쓴 뒤 렌더에서 죽는 것은 진단의 실패다.
func TestParseRejectsUnrenderedVocabulary(t *testing.T) {
	withUnrenderedKinds(t, KindTable, KindTableRow, KindTableCell)
	src := `<slide><table><tr><td>남미</td><td>18%</td></tr></table></slide>`
	_, diags := Parse("slides/001_table.slide", src)
	if len(diags) != 1 {
		t.Fatalf("진단 %d건, want 1건 (표 하나에 진단 하나):\n%s", len(diags), diags.Err())
	}
	if got := diags[0].Span.Start.String(); got != "1:9" {
		t.Errorf("진단 위치 = %s, want 1:9 (<table> 태그 이름)", got)
	}
	for _, want := range []string{"<table>", "아직 구현되지 않은 어휘"} {
		if !strings.Contains(diags[0].Message, want) {
			t.Errorf("메시지 = %q, want %q 포함", diags[0].Message, want)
		}
	}
	if diags[0].Severity != SeverityError {
		t.Errorf("진단은 에러여야 한다: %v", diags[0])
	}
}

// 그리지 못하는 어휘라도 **배치 규칙은 그대로** 본다 — 두 진단은 서로 다른 사실이고
// (여기 놓을 수 없다 / 아직 그릴 수 없다), 저자는 둘 다 알아야 한다.
func TestParseUnrenderedVocabularyKeepsPlacementRules(t *testing.T) {
	withUnrenderedKinds(t, KindTableRow, KindTableCell)
	_, diags := Parse("slides/001.slide", `<slide><tr><td>x</td></tr></slide>`)
	if len(diags) != 2 {
		t.Fatalf("진단 %d건, want 2건:\n%s", len(diags), diags.Err())
	}
	if !strings.Contains(diags[0].Message, "<tr>는 <table>의 자식으로만") {
		t.Errorf("진단[0] = %q, want 배치 진단", diags[0].Message)
	}
	if !strings.Contains(diags[1].Message, UnrenderedVocabMessage) {
		t.Errorf("진단[1] = %q, want 미구현 어휘 진단", diags[1].Message)
	}
	// 안쪽 <td>는 말하지 않는다 — 같은 사실의 반복이다 (<td>도 세워 두었지만 진단은 하나다).
	unrendered := 0
	for _, d := range diags {
		if strings.Contains(d.Message, UnrenderedVocabMessage) {
			unrendered++
		}
	}
	if unrendered != 1 {
		t.Errorf("미구현 어휘 진단 %d건, want 1건 (가장 바깥에서만 말한다):\n%s", unrendered, diags.Err())
	}
}
