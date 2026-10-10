package core

import (
	"strings"
	"testing"

	"github.com/k2sebeom/balpyo/soml"
)

// 값 층의 "engine bug" 가드는 **카탈로그가 연 속성**에만 소리를 내야 한다 (catalogGap).
//
// 파서가 이미 "알 수 없는 속성"으로 거절한 이름까지 그 가드에 닿으면 저자는 오타 하나에
// 진단 둘을 받고, 둘째가 "engine bug"라고 말한다 — 자기 오타를 제품의 결함으로 읽게 만드는
// 진단이다. 컨테이너는 카탈로그 목록에 없는 이름을
// 조용히 지나가므로, 이 테스트는 도형·그림이 그 결을 따르는지 본다.
func TestValueLayerStaysQuietOnAttrsTheParserRejected(t *testing.T) {
	cases := []struct {
		name string
		body string
	}{
		{"도형의 오타", `<shape type="rect" atl="x"/>`},
		{"도형에 컨테이너 속성", `<shape type="rect" justify="center"><text>x</text></shape>`},
		{"그림의 오타", `<image src="a.png" atl="x"/>`},
		{"그림에 도형 속성", `<image src="a.png" fill="accent1"/>`},
		// 대조군: 이미 조용한 자리들이 조용한 채로 남는지 함께 못 박는다.
		{"컨테이너의 오타", `<row atl="x"><text>x</text></row>`},
		{"컨테이너에 앵커", `<col anchor="center"><text>x</text></col>`},
		{"표 셀의 오타", `<table><tr><td atl="x">y</td></tr></table>`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			src := "<slide id=\"s\">\n  " + tc.body + "\n</slide>\n"
			doc, parseDiags := soml.Parse("slides/001_gap.slide", src)
			if len(parseDiags) == 0 {
				t.Fatalf("파서가 이 원문을 거절해야 한다: %s", tc.body)
			}
			deck := &soml.Deck{Slides: []*soml.Slide{{
				ID: "s", Source: doc.File, Content: doc.Content, Tokens: doc.Tokens,
			}}}
			for _, diag := range Resolve(deck, nil) {
				if strings.Contains(diag.Message, "engine bug") {
					t.Errorf("파서가 거절한 이름에 값 층이 엔진 버그를 말한다: %s", diag.Message)
				}
			}
		})
	}
}

// 그리고 **카탈로그가 연 이름에는 여전히 소리를 낸다.** 가드를 조용하게 만든 변경이 가드를
// 지운 것이기도 하면, 값 층이 속성 하나를 빠뜨려도 아무도 모른다 — 그 실패의 가장 조용한 길이
// soml.ElementAttrs가 빈 목록을 돌려주기 시작하는 것이다 (그러면 위 테스트는 초록인데 아무것도
// 지키지 않는다). 위 테스트와 이 테스트는 같은 한 줄의 두 방향이다.
func TestCatalogGapStillSpeaksForAttrsTheCatalogOpened(t *testing.T) {
	r := &resolver{file: "slides/001_gap.slide"}
	r.catalogGap(&soml.Element{Name: soml.ShapeElement}, soml.Attr{Name: "w"})
	if len(r.diags) != 1 {
		t.Fatalf("진단 %d건, want 1 — 카탈로그가 <shape w>를 열었으므로 값 층의 공백은 엔진 버그다", len(r.diags))
	}
	if !strings.Contains(r.diags[0].Message, "engine bug") {
		t.Errorf("진단 = %q, want 엔진 버그", r.diags[0].String())
	}
}
