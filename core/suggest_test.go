package core

import (
	"strings"
	"testing"
)

// 도그푸딩의 실제 오타 셋(값 층)이 후보를 얻는다: 레이아웃 · 역할 · 도형 종류. 후보는 진단이 이미 싣는
// 목록에서만 나온다 — 목록에 없는 이름을 권하지 않는다.
func TestValueTyposSuggestACandidate(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want string
	}{
		{"layout", `<slide layout="two-colum"></slide>`,
			`unknown layout "two-colum" — did you mean "two-column"? (available layouts: `},
		{"role", `<slide><text role="heding">제목</text></slide>`,
			`unknown role "heding" — did you mean "heading"? (available roles: `},
		{"도형 type", `<slide><shape type="elipse" w="100" h="100"/></slide>`,
			`unknown shape type "elipse" — did you mean "ellipse"? (available types: `},
		{"색 토큰", `<slide><text color="acent1">a</text></slide>`,
			`: color token "acent1" is not in the theme — did you mean "accent1"? (available tokens: `},
		{"align", `<slide><text align="centre">a</text></slide>`,
			`unknown align "centre" — did you mean "center"? (available values: `},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, diags := resolveSlide(t, tc.src)
			for _, diag := range diags {
				if strings.Contains(diag.Message, tc.want) {
					return
				}
			}
			t.Errorf("진단에 %q가 없다: %v", tc.want, diags)
		})
	}
}

// 먼 이름에는 후보가 없다 — 엉뚱한 후보는 저자를 다른 진단으로 보낸다.
func TestValueTyposWithoutCandidate(t *testing.T) {
	_, diags := resolveSlide(t, `<slide><shape type="arrow" w="100" h="100"/></slide>`)
	for _, diag := range diags {
		if strings.Contains(diag.Message, "did you mean") {
			t.Errorf("먼 이름에 후보가 붙었다: %s", diag.Message)
		}
	}
	if len(diags) == 0 {
		t.Fatal("어휘 밖 type이 진단되지 않았다")
	}
}
