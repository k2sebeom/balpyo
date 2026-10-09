package pptx

import (
	"strings"
	"testing"

	"github.com/k2sebeom/balpyo/core"
	"github.com/k2sebeom/balpyo/render/geom"
	"github.com/k2sebeom/balpyo/soml"
)

// 슬롯을 쓴 덱은 그 레이아웃의 플레이스홀더 파트를 얻고 마스터가 프로토타입·txStyles를 얻는다
// — 그래야 슬라이드의 `<p:ph>`가 진짜 플레이스홀더가 된다. 슬롯을 쓰지 않은
// 덱은 blank 레이아웃 하나만 갖는다. 슬라이드의 ph 매핑과 파트 간 연결은 패리티·교차 파트 검증이 본다.
func TestRenderSlotEmitsLayoutPart(t *testing.T) {
	rects := []geom.Rect{{X: 64, Y: 64, W: 1152, H: 67.2}, {X: 64, Y: 155.2, W: 1152, H: 36}}
	deck := resolvedDeck(t, `<slide>
  <h1 slot="title">제목</h1>
  <p slot="body">본문</p>
</slide>`)
	parts, _ := renderPackage(t, NewWithGeometry(measuredGeometry(t, deck, rects...)), deck)

	for part, wants := range map[string][]string{
		"ppt/slideLayouts/slideLayout2.xml": {`<p:cSld name="content">`, `<p:ph type="title"/>`, `<p:ph type="body" idx="1"/>`},
		masterPart:                          {`<p:ph type="title"/>`, `<p:ph type="body" idx="1"/>`, `<p:sldLayoutId id="2147483650" r:id="rId3"/>`, "<p:txStyles>"},
	} {
		for _, want := range wants {
			if !strings.Contains(parts[part], want) {
				t.Errorf("%s에 %q가 없다", part, want)
			}
		}
	}

	plain := resolvedDeck(t, `<slide>
  <text role="title">제목</text>
  <text>본문</text>
</slide>`)
	parts, _ = renderPackage(t, NewWithGeometry(measuredGeometry(t, plain, rects...)), plain)
	if _, ok := parts["ppt/slideLayouts/slideLayout2.xml"]; ok {
		t.Error("슬롯 없는 덱에 slideLayout2.xml가 새어 나왔다")
	}
	if master := parts[masterPart]; strings.Contains(master, "<p:ph") || strings.Contains(master, "<p:txStyles>") ||
		strings.Count(master, "<p:sldLayoutId ") != 1 {
		t.Errorf("슬롯 없는 덱의 마스터가 blank 하나가 아니다:\n%s", master)
	}
}

// layoutSlotDeck은 지정한 레이아웃을 실은 슬라이드 하나의 덱이다 — resolvedDeck은 `layout`을
// 옮기지 않으므로(언제나 기본 content) 레이아웃별 슬롯을 보는 테스트가 쓸 경로가 따로 필요하다
// (render/web의 layoutDeck과 같은 조립).
func layoutSlotDeck(t *testing.T, layout soml.SlideLayout, body string) *soml.Deck {
	t.Helper()
	const source = "slides/001_slots.slide"
	doc, diags := soml.Parse(source, `<slide id="s" layout="`+string(layout)+`">`+"\n"+body+"\n</slide>\n")
	if err := diags.Err(); err != nil {
		t.Fatalf("파싱 실패: %v", err)
	}
	stage, err := soml.Aspect16x9.Stage()
	if err != nil {
		t.Fatal(err)
	}
	deck := &soml.Deck{
		Title: "슬롯", Lang: "ko", Aspect: soml.Aspect16x9, Stage: stage,
		Slides: []*soml.Slide{{
			Index: 0, ID: "s", Source: source,
			Layout: soml.SlideLayout(doc.Layout), LayoutSpan: doc.LayoutSpan,
			Notes: doc.Notes, Content: doc.Content,
		}},
	}
	if err := core.Resolve(deck, nil).Err(); err != nil {
		t.Fatalf("외관 해석 실패: %v", err)
	}
	return deck
}

// **슬롯 모드 판정이 두 타깃에서 같은 결론에 이른다**.
//
// 한쪽만 슬롯 모드를 켜면 한 소스가 웹에서는 조용히 무효, PPTX에서는 플레이스홀더가 된다.
// 대조는 두 렌더러의 결론으로 한다 (스테이지의 `balpyo-slots` 모디파이어 ↔ `<p:ph>`).
func TestSlotModeAgreesAcrossTargets(t *testing.T) {
	for _, layout := range soml.SlideLayouts() {
		slots := soml.LayoutSlots(layout)
		if len(slots) == 0 {
			continue // canvas — 슬롯 어휘가 비어 있다 (slot=은 빌드 에러다)
		}
		t.Run(string(layout), func(t *testing.T) {
			deck := layoutSlotDeck(t, layout, `<p slot="`+slots[0].Name+`">본문</p>`)
			g := measuredGeometry(t, deck, geom.Rect{X: 64, Y: 64, W: 1152, H: 36})

			webSlotMode := strings.Contains(stageClassOf(t, renderWebString(t, deck)), "balpyo-slots")
			pptxPlaceholder := strings.Contains(slideXML(t, deck, g), "<p:ph")
			if webSlotMode != pptxPlaceholder {
				t.Errorf("슬롯 %q: 웹 슬롯 모드=%v, PPTX 플레이스홀더=%v — 같은 소스가 두 타깃에서 갈렸다",
					slots[0].Name, webSlotMode, pptxPlaceholder)
			}
		})
	}
}

// stageClassOf는 첫 스테이지의 class 속성 값이다.
//
// 문서 전체를 substring으로 보면 안 되는 이유가 있다: 흐름 배치 규칙의 선택자가
// `:not(.balpyo-slots)`라서(layoutStagesCSS) 슬롯 모드가 **꺼진** 덱의 CSS에도
// `balpyo-slots`라는 글자가 들어 있다. 판정은 스테이지 요소의 클래스 목록에서만 읽는다.
func stageClassOf(t *testing.T, out string) string {
	t.Helper()
	const open = `<div class="balpyo-stage`
	i := strings.Index(out, open)
	if i < 0 {
		t.Fatalf("스테이지 여는 태그를 찾지 못했다\n%s", out)
	}
	rest := out[i+len(`<div class="`):]
	end := strings.IndexByte(rest, '"')
	if end < 0 {
		t.Fatalf("스테이지 class 속성이 닫히지 않았다: %q", rest)
	}
	return rest[:end]
}
