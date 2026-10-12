package soml

import "testing"

// 덱 안 링크의 목적지는 웹 뷰어의 해시 딥링크 규칙 그대로다 — id가 번호보다 먼저이고, `#/`와
// `#`는 같으며, 퍼센트 인코딩을 푼다. `#`로 시작하지 않으면 덱 밖이다.
func TestLinkedSlide(t *testing.T) {
	deck := &Deck{Slides: []*Slide{{ID: "cover"}, {ID: "3"}, {ID: "소개"}}}
	for _, tc := range []struct {
		href   string
		want   string // 빈 값이면 목적지 없음
		inDeck bool
	}{
		{"#cover", "cover", true},
		{"#/cover", "cover", true},
		{"#/2", "3", true},
		// 번호처럼 생긴 id가 번호보다 먼저다 — `#3`은 셋째 슬라이드가 아니라 id "3"이다.
		{"#3", "3", true},
		{"#%EC%86%8C%EA%B0%9C", "소개", true},
		{"#", "cover", true},
		{"#/", "cover", true},
		{"#/4", "", true},
		{"#/0", "", true},
		{"#nowhere", "", true},
		{"https://example.com/#cover", "", false},
		{"slides.html", "", false},
	} {
		slide, inDeck := deck.LinkedSlide(tc.href)
		got := ""
		if slide != nil {
			got = slide.ID
		}
		if got != tc.want || inDeck != tc.inDeck {
			t.Errorf("LinkedSlide(%q) = (%q, %v), want (%q, %v)", tc.href, got, inDeck, tc.want, tc.inDeck)
		}
	}
}
