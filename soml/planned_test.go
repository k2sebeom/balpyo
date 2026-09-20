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
//
// **오늘 예정 목록은 비어 있다** (`lang`은 P2.4가, `<shape alt>`·`<table alt>`는 P5.3이
// 구현하며 지웠다) — 그래서 계획된 쪽은 목록을 잠시 세워 본다. 그 대비가 이 테스트의 값이다:
// 같은 파서가 한 이름에는 "아직 구현되지 않았다"를, 옆 이름에는 "알 수 없다"를 말해야 한다.
// 목록이 비어 있는 동안에도 **관문 자체가 사는지**를 함께 지킨다 — 다음 예정 속성이 적히는
// 날 문장이 돌아온다 (unrenderedKinds의 withUnrenderedKinds와 같은 태도).
func TestPlannedAttrDiagnostic(t *testing.T) {
	restore := withPlannedAttrs([]plannedAttr{{Tag: "shape", Name: "lang", Opens: "가상 항목"}})
	defer restore()

	tests := []struct {
		name string
		src  string
		want string
	}{
		{
			name: "예정된 속성은 구현을 말한다",
			src:  `<slide><shape type="rect" lang="go"/></slide>`,
			want: `1:27 <shape>의 lang는 ` + PlannedAttrMessage + ` — 가상 항목이 그 속성을 연다`,
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
