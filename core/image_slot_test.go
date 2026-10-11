package core

import (
	"testing"

	"github.com/k2sebeom/balpyo/soml"
)

// 그림 슬롯(`media`)에 크기 없이 놓인 그림은 슬롯 영역을 채운다 — 그렇지 않으면 박스가 그림의 원래
// 픽셀 크기라 `fit="cover"`가 덮을 영역이 없고, 전면 그림이 가운데에 작게 남는다. 저자가 크기를 하나라도
// 적었거나 그림 슬롯이 아니면 손대지 않는다.
func TestImageInPictureSlotFillsTheSlot(t *testing.T) {
	cases := []struct {
		name, src  string
		wantFilled bool
	}{
		{"image-full media", `<slide layout="image-full"><image slot="media" src="assets/p.png" fit="cover" alt="p"/></slide>`, true},
		{"image-text media", `<slide layout="image-text"><image slot="media" src="assets/p.png" alt="p"/></slide>`, true},
		{"저자의 h", `<slide layout="image-full"><image slot="media" src="assets/p.png" h="300" alt="p"/></slide>`, false},
		{"저자의 ratio", `<slide layout="image-text"><image slot="media" src="assets/p.png" w="400" ratio="4:3" alt="p"/></slide>`, false},
		{"그림 슬롯이 아니다", `<slide layout="title"><image slot="subtitle" src="assets/p.png" alt="p"/></slide>`, false},
		{"슬롯 밖", `<slide layout="image-full"><image src="assets/p.png" alt="p"/></slide>`, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			deck, diags := themeDeck(t, "", tc.src)
			if len(diags) != 0 {
				t.Fatalf("진단이 없어야 한다: %v", diags)
			}
			style := deck.Slides[0].Content[0].(*soml.Element).Appearance.Image
			full := soml.Percent(100)
			filled := style.Box.Width != nil && *style.Box.Width == full &&
				style.Box.Height != nil && *style.Box.Height == full
			if filled != tc.wantFilled {
				t.Errorf("슬롯을 채운다 = %v, want %v (w=%v h=%v)", filled, tc.wantFilled, style.Box.Width, style.Box.Height)
			}
		})
	}
}
