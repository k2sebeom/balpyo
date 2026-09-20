package soml

import (
	"strings"
	"testing"
)

// 그리지 못하는 어휘는 **파스 시점에** 막힌다 — 다 쓴 뒤 렌더에서 죽는 것은 진단의 실패다.
func TestParseRejectsUnrenderedVocabulary(t *testing.T) {
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
	// 안쪽 <td>는 말하지 않는다 — 같은 사실의 반복이다.
	if got := withoutUnrenderedVocab(diags); len(got) != 1 {
		t.Errorf("미구현 진단을 뺀 나머지 %d건, want 1건:\n%s", len(got), got.Err())
	}
}

// withoutUnrenderedVocab는 "아직 구현되지 않은 어휘" 진단을 걸러낸다.
//
// 표 어휘는 파서와 값 층까지 열려 있고 렌더러만 없으므로(KindRendered), 트리·배치·값
// 테스트는 그 진단을 빼고 나머지를 단정한다 — 어휘가 층마다 다른 속도로 자라는 동안의
// 임시 장치다. 렌더러가 붙으면(TASKS P2.1) 이 헬퍼와 호출부가 함께 사라진다.
func withoutUnrenderedVocab(diags Diagnostics) Diagnostics {
	out := make(Diagnostics, 0, len(diags))
	for _, d := range diags {
		if strings.Contains(d.Message, UnrenderedVocabMessage) {
			continue
		}
		out = append(out, d)
	}
	return out
}
