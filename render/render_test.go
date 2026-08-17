package render

import "testing"

func TestParseTargets(t *testing.T) {
	tests := []struct {
		in   string
		want []Target
	}{
		{"html", []Target{TargetHTML}},
		{"html,pptx", []Target{TargetHTML, TargetPPTX}},
		{"pptx, html", []Target{TargetPPTX, TargetHTML}},
		{"html,html", []Target{TargetHTML}}, // 중복은 접는다
	}
	for _, tc := range tests {
		got, err := ParseTargets(tc.in)
		if err != nil {
			t.Fatalf("%q: %v", tc.in, err)
		}
		if len(got) != len(tc.want) {
			t.Fatalf("%q → %v, want %v", tc.in, got, tc.want)
		}
		for i := range got {
			if got[i] != tc.want[i] {
				t.Errorf("%q → %v, want %v", tc.in, got, tc.want)
			}
		}
	}
}

func TestParseTargetsRejectsUnknown(t *testing.T) {
	// 오타가 "아무것도 안 만들어졌다"로 나타나면 안 된다.
	for _, in := range []string{"pdf", "htlm", "html,pdf", "", ","} {
		if _, err := ParseTargets(in); err == nil {
			t.Errorf("%q는 에러여야 한다", in)
		}
	}
}

func TestFilesPathsAreSorted(t *testing.T) {
	files := Files{"b.txt": nil, "a.txt": nil, "ppt/slide1.xml": nil}
	got := files.Paths()
	want := []string{"a.txt", "b.txt", "ppt/slide1.xml"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
}
