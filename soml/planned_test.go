package soml

import (
	"strings"
	"testing"
)

// withPlannedAttrs는 예정 목록을 잠시 세운다 — 목록이 비는 날에도 관문이 사는지 보려면
// 목록 자체를 갈아 봐야 한다 (unrendered_test.go의 withUnrenderedKinds와 같은 장치).
func withPlannedAttrs(list []plannedAttr) func() {
	saved := plannedAttrs
	plannedAttrs = list
	return func() { plannedAttrs = saved }
}

// 계획된 속성은 오타와 **다른 문장으로** 말한다 (D16 · ADR 0051). 2회차 피드백의 마찰점이
// 그것이었다: `<pre lang="go">`가 "알 수 없는 속성"으로 죽어서 저자가 이름을 의심하며
// 여러 번 바꿔 봤다 — 없는 것은 이름이 아니라 구현이었다.
func TestPlannedAttrDiagnostic(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want string
	}{
		{
			name: "도형의 alt는 예정된 속성이다",
			src:  `<slide><shape type="rect" alt="아키텍처 그림"/></slide>`,
			want: `1:27 <shape>의 alt는 ` + PlannedAttrMessage + ` — P5.3 접근성이 그 속성을 연다`,
		},
		{
			name: "표 프레임의 alt도 같은 자리다",
			src:  `<slide><table alt="분기 매출"><tr><td>x</td></tr></table></slide>`,
			want: `1:15 <table>의 alt는 ` + PlannedAttrMessage,
		},
		{
			// 오타는 지금처럼 말한다 — 예정 목록은 **닫혀** 있고, 목록 밖은 어휘의 문제다.
			name: "목록 밖 이름은 여전히 오타다",
			src:  `<slide><shape type="rect" atl="x"/></slide>`,
			want: `1:27 <shape>에 알 수 없는 속성 atl`,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, diags := Parse("slides/001_title.slide", tc.src)
			if len(diags) != 1 {
				t.Fatalf("진단 %d개, want 1개:\n%s", len(diags), diags.Err())
			}
			got := diags[0].Span.Start.String() + " " + diags[0].Message
			if !strings.Contains(got, tc.want) {
				t.Errorf("진단 = %q, want %q 포함", got, tc.want)
			}
		})
	}
}

// 예정 목록과 실제 어휘의 **양방향 대조**다 (unrenderedKinds ↔ 렌더러 디스패치와 같은 장치).
// 속성을 구현하는 커밋이 목록에서 그 줄을 지우지 않으면 여기서 깨진다 — 목록이 썩는 것을
// 사람의 기억이 아니라 테스트가 막는다. `lang`이 이 규칙의 첫 증인이다: ADR 0051이 그 줄을
// 적고 같은 항목이 지웠다.
func TestPlannedAttrsStayOutsideTheVocabulary(t *testing.T) {
	if len(plannedAttrs) == 0 {
		t.Skip("예정된 속성이 없다 — 관문은 withPlannedAttrs가 지킨다")
	}
	for _, planned := range plannedAttrs {
		spec, known := elementSpecs[planned.Tag]
		if !known {
			t.Errorf("예정 목록의 <%s>는 어휘에 없는 태그다", planned.Tag)
			continue
		}
		for _, allowed := range spec.attrs {
			if allowed == planned.Name {
				t.Errorf("<%s>의 %s는 이미 구현된 속성이다 — 예정 목록에서 지운다 (%s)",
					planned.Tag, planned.Name, planned.Opens)
			}
		}
		if planned.Opens == "" {
			t.Errorf("<%s>의 %s에 그 속성을 여는 항목이 적혀 있지 않다", planned.Tag, planned.Name)
		}
	}
}

// 목록이 비는 날에도 관문이 사는지는 목록을 **잠시 세워 보고** 확인한다
// (unrenderedKinds의 withUnrenderedKinds와 같은 태도).
func TestPlannedAttrGateSurvivesAnEmptyList(t *testing.T) {
	restore := withPlannedAttrs([]plannedAttr{{Tag: "h2", Name: "lang", Opens: "가상 항목"}})
	defer restore()

	_, diags := Parse("slides/001_title.slide", `<slide><h2 lang="go">제목</h2></slide>`)
	if len(diags) != 1 {
		t.Fatalf("진단 %d개, want 1개:\n%s", len(diags), diags.Err())
	}
	if !strings.Contains(diags[0].Message, PlannedAttrMessage) {
		t.Errorf("진단 = %q, want %q 포함", diags[0].Message, PlannedAttrMessage)
	}
}
