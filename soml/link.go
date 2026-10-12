package soml

import (
	"net/url"
	"strconv"
	"strings"
)

// LinkedSlide는 href가 가리키는 덱 안의 슬라이드다.
//
// `#`로 시작하는 href는 웹 뷰어의 해시 딥링크와 같은 규칙으로 읽는다: 앞의 `#`(와 `/` 하나)를
// 떼고 퍼센트 인코딩을 풀어, 슬라이드 id가 먼저 · 1부터 세는 번호가 다음이다. 빈 조각은
// 첫 슬라이드다. 검증과 PPTX 렌더러가 이 한 함수를 나눠 써야 웹 뷰어에서 가는 링크와
// PPTX에서 가는 링크가 같은 슬라이드를 가리킨다.
//
// inDeck은 href가 덱 안을 가리키는 모양(`#…`)인지다. inDeck인데 slide가 nil이면 그런
// 슬라이드가 없다 (skip된 슬라이드도 덱에 없으므로 여기에 든다). `#`로 시작하지 않는 href는
// 덱 밖의 목적지라 inDeck이 거짓이다.
func (d *Deck) LinkedSlide(href string) (slide *Slide, inDeck bool) {
	if !strings.HasPrefix(href, "#") {
		return nil, false
	}
	raw := strings.TrimPrefix(strings.TrimPrefix(href, "#"), "/")
	if decoded, err := url.PathUnescape(raw); err == nil {
		raw = decoded
	}
	if len(d.Slides) == 0 {
		return nil, true
	}
	if raw == "" {
		return d.Slides[0], true
	}
	for _, s := range d.Slides {
		if s.ID == raw {
			return s, true
		}
	}
	if n, err := strconv.Atoi(raw); err == nil && n >= 1 && n <= len(d.Slides) {
		return d.Slides[n-1], true
	}
	return nil, true
}
