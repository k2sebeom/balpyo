package soml

import "testing"

// 도그푸딩에서 실제로 난 오타 넷이 후보를 얻는다 — 이 넷이 이 기능의 존재 이유다. 넷째인
// `img` → `image`는 짧은 낱말의 거리 2라 편집 거리가 아니라 HTML 별칭이 답한다 (TestSuggestTag).
func TestSuggestFindsRealTypos(t *testing.T) {
	tests := []struct {
		word       string
		candidates []string
		want       string
	}{
		{"two-colum", []string{"content", "title", "two-column", "image-text"}, "two-column"},
		{"heding", []string{"title", "heading", "body", "caption", "code"}, "heading"},
		{"elipse", []string{"rect", "rounded-rect", "ellipse", "diamond"}, "ellipse"},
		// 이웃 자리 바꿈은 한 번의 실수다.
		{"haeding", []string{"title", "heading", "body"}, "heading"},
		// 대소문자만 다른 이름.
		{"Text", []string{"text", "shape"}, "text"},
	}
	for _, tc := range tests {
		got, ok := Suggest(tc.word, tc.candidates)
		if !ok || got != tc.want {
			t.Errorf("Suggest(%q) = %q, %v; want %q", tc.word, got, ok, tc.want)
		}
	}
}

// 거리가 멀거나 낱말이 너무 짧으면 후보를 만들지 않는다 — 엉뚱한 후보는 후보가 없는 것보다 나쁘다:
// 저자가 그 이름을 쓰고 다른 진단을 받는다.
func TestSuggestRejectsFarWords(t *testing.T) {
	tests := []struct {
		word       string
		candidates []string
	}{
		{"arrow", []string{"rect", "ellipse", "diamond"}}, // 다른 낱말
		{"fill", []string{"src", "fit", "alt"}},           // 짧은 낱말의 거리 2는 다른 개념이다
		{"gap", []string{"grow", "at", "x"}},
		{"p", []string{"a", "br"}},                      // 한 글자는 무엇이든 거리 1이다
		{"gradient-text", []string{"title", "heading"}}, // 멀다
		{"", []string{"text"}},                          // 빈 이름
		{"text", []string{"text"}},                      // 같은 이름은 후보가 아니다
	}
	for _, tc := range tests {
		if got, ok := Suggest(tc.word, tc.candidates); ok {
			t.Errorf("Suggest(%q) = %q — 후보가 없어야 한다", tc.word, got)
		}
	}
}

// 거리가 같으면 목록 앞쪽이 이긴다 — 결과가 실행마다 같아야 진단 골든이 흔들리지 않는다.
func TestSuggestTieGoesToListOrder(t *testing.T) {
	got, _ := Suggest("ba", []string{"bb", "aa", "ca"})
	if got != "bb" {
		t.Errorf("Suggest(ba) = %q, want bb (목록의 첫 거리-1 후보)", got)
	}
}

// 태그 후보: HTML 습관이 먼저, 자리 목록(거리 2), 어휘 전체(거리 1) 순서다.
func TestSuggestTag(t *testing.T) {
	here := []string{"li"}
	all := []string{"image", "li", "strong", "text"}
	tests := []struct{ name, want string }{
		{"b", "strong"},     // 별칭
		{"IMG", "image"},    // 별칭은 대소문자를 가리지 않는다
		{"il", "li"},        // 자리 목록
		{"texts", "text"},   // 어휘 전체, 거리 1
		{"imagez", "image"}, // 어휘 전체, 거리 1
	}
	for _, tc := range tests {
		if got, ok := SuggestTag(tc.name, here, all); !ok || got != tc.want {
			t.Errorf("SuggestTag(%q) = %q, %v; want %q", tc.name, got, ok, tc.want)
		}
	}
	if got, ok := SuggestTag("imgx", here, all); ok {
		t.Errorf("SuggestTag(imgx) = %q — 자리 밖 거리 2는 후보가 아니다", got)
	}
}
