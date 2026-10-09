package pptx

import (
	"testing"

	"github.com/k2sebeom/balpyo/render/geom"
)

// PPTX는 `reveal`을 **모른다** — 순차 등장은 전개된 정적 상태로 평탄화된다 (docs/reference/soml/text-subset.md,
// ADR 0063 결정 3). 판정은 산출물 대조다: 같은 목록을 reveal 없이/있게 쓴 두 덱의 슬라이드
// 파트가 **글자 하나까지 같아야** 한다. 문단 수를 세는 것보다 강한 계약이다 — 스텝이
// 텍스트·좌표·불릿 중 어디로도 새지 않는다.
func TestRevealIsFlattenedInPPTX(t *testing.T) {
	const plainSrc = `<slide>
  <ul>
    <li>첫째</li>
    <li>둘째</li>
    <li>셋째</li>
  </ul>
</slide>`
	const revealSrc = `<slide>
  <ul>
    <li>첫째</li>
    <li reveal>둘째</li>
    <li reveal="3">셋째</li>
  </ul>
</slide>`
	rects := []geom.Rect{listRect, itemRect, itemRect, itemRect}

	plain := resolvedDeck(t, plainSrc)
	reveal := resolvedDeck(t, revealSrc)
	want := slideXML(t, plain, measuredGeometry(t, plain, rects...))
	got := slideXML(t, reveal, measuredGeometry(t, reveal, rects...))
	if got != want {
		t.Errorf("reveal이 PPTX 산출물을 바꿨다 — 평탄화가 깨졌다\n got: %s\nwant: %s", got, want)
	}
}
