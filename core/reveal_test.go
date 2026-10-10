package core

import (
	"strings"
	"testing"
)

// `reveal`의 값 어휘는 값 층의 것이다: 값이 없으면 **자기 차례**이고,
// 값이 있으면 1 이상의 정수다. 파서는 이름만 보므로(soml) 0·음수·소수·낱말은 여기서 잡는다.
func TestRevealValueIsOptionalOrPositiveInteger(t *testing.T) {
	for _, src := range []string{
		`<slide><ul><li reveal>하나</li><li reveal>둘</li></ul></slide>`,
		`<slide><ul><li reveal="1">하나</li><li reveal="1">함께</li></ul></slide>`,
		`<slide><ul><li reveal="3">셋째로</li><li>먼저</li></ul></slide>`,
		`<slide><ul><li>순서 없는 항목만</li></ul></slide>`,
	} {
		if _, diags := resolveSlide(t, src); len(diags) != 0 {
			t.Errorf("%s\n진단이 나왔다:\n%s", src, diags.Err())
		}
	}
}

// 틀린 값은 진단이고, **그 항목만** 순서를 잃는다 — 틀린 값 하나가 나머지 항목의 진단을
// 삼키지 않도록 값 해석이 항목마다 끝난다.
func TestRevealValueErrors(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want []string
	}{
		{
			name: "빈 값",
			src:  `<slide><ul><li reveal="">항목</li></ul></slide>`,
			want: []string{`1:24 the reveal attribute is empty`},
		},
		{
			name: "0은 스텝이 아니다",
			src:  `<slide><ul><li reveal="0">항목</li></ul></slide>`,
			want: []string{`1:24 reveal is an integer of at least 1 ("0")`},
		},
		{
			name: "음수",
			src:  `<slide><ul><li reveal="-1">항목</li></ul></slide>`,
			want: []string{`1:24 reveal is an integer of at least 1 ("-1")`},
		},
		{
			name: "소수",
			src:  `<slide><ul><li reveal="1.5">항목</li></ul></slide>`,
			want: []string{`1:24 reveal is an integer of at least 1 ("1.5")`},
		},
		{
			name: "낱말",
			src:  `<slide><ul><li reveal="두번째">항목</li></ul></slide>`,
			want: []string{`1:24 reveal is an integer of at least 1 ("두번째")`},
		},
		{
			name: "틀린 값 둘이면 진단도 둘이다",
			src:  `<slide><ul><li reveal="0">하나</li><li reveal="x">둘</li></ul></slide>`,
			want: []string{
				`1:24 reveal is an integer of at least 1 ("0")`,
				`1:46 reveal is an integer of at least 1 ("x")`,
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, diags := resolveSlide(t, tc.src)
			if len(diags) != len(tc.want) {
				t.Fatalf("진단 %d개, want %d개:\n%s", len(diags), len(tc.want), diags.Err())
			}
			for i, want := range tc.want {
				pos, message, _ := strings.Cut(want, " ")
				if got := diags[i].Span.Start.String(); got != pos {
					t.Errorf("진단[%d] 위치 = %s, want %s (%s)", i, got, pos, diags[i].Message)
				}
				if !strings.Contains(diags[i].Message, message) {
					t.Errorf("진단[%d] 메시지 = %q, want %q 포함", i, diags[i].Message, message)
				}
			}
		})
	}
}
