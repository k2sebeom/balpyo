package soml

import "testing"

// withPlannedAttrs는 예정 목록을 잠시 세운다 — 목록이 빈 동안에도 관문이 사는지 보는 장치다.
func withPlannedAttrs(list []plannedAttr) func() {
	saved := plannedAttrs
	plannedAttrs = list
	return func() { plannedAttrs = saved }
}

// 계획된 속성은 오타와 **다른 문장으로** 말한다 — 없는 것이 이름이 아니라 구현일
// 때 저자가 이름을 의심하지 않게. 오늘 예정 목록은 비어 있어 목록을 잠시 세워 관문이 사는지 본다.
func TestPlannedAttrDiagnostic(t *testing.T) {
	restore := withPlannedAttrs([]plannedAttr{{Tag: "shape", Name: "lang", Opens: "a hypothetical item"}})
	defer restore()
	runDiagCases(t, []diagCase{
		{
			name: "예정된 속성은 구현을 말한다",
			src:  `<slide><shape type="rect" lang="go"/></slide>`,
			want: []string{`1:27 lang on <shape> ` + PlannedAttrMessage + ` — a hypothetical item opens that attribute`},
		},
		{
			// 예정 목록은 **닫혀** 있고, 목록 밖은 어휘의 문제다.
			name: "목록 밖 이름은 여전히 오타다",
			src:  `<slide><shape type="rect" atl="x"/></slide>`,
			want: []string{`1:27 unknown attribute atl on <shape>`},
		},
	})
}

// 예정 목록과 실제 어휘의 대조 — 속성을 구현하는 커밋이 목록에서 그 줄을 지우지 않으면
// 여기서 깨진다.
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
