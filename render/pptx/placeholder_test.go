package pptx

import (
	"strings"
	"testing"

	"github.com/k2sebeom/balpyo/render/geom"
)

// 슬롯을 채운 슬라이드 직계 자식은 `<p:nvPr>` 안에 `<p:ph>`를 얻는다 (ADR 0035 선택지 3).
// content 레이아웃(resolvedDeck의 기본)은 title(type=title, idx 생략)·body(type=body, idx=1)
// 슬롯을 가지므로 phType/idx 매핑이 soml 카탈로그(LayoutSlot)에서 그대로 나온다.
//
// **명시 xfrm/rPr은 그대로 둔다**: 도형은 플레이스홀더 표식을 얻되 좌표는 잰 rect가 소유한다
// (off/ext가 여전히 나온다) — 웹·PPTX 픽셀 일치 계약. 그래서 `<p:ph>`는 표식일 뿐이다.
func TestRenderSlotEmitsPlaceholder(t *testing.T) {
	deck := resolvedDeck(t, `<slide>
  <h1 slot="title">제목</h1>
  <p slot="body">본문</p>
</slide>`)
	g := measuredGeometry(t, deck,
		geom.Rect{X: 64, Y: 64, W: 1152, H: 67.2},
		geom.Rect{X: 64, Y: 155.2, W: 1152, H: 36},
	)
	xml := slideXML(t, deck, g)

	// title 슬롯 → type만, idx 생략 (제목류의 관례 — PhIndex 0).
	if !strings.Contains(xml, `<p:ph type="title"/>`) {
		t.Errorf("title 슬롯의 <p:ph type=\"title\"/>가 없다\n%s", xml)
	}
	// body 슬롯 → idx=1.
	if !strings.Contains(xml, `<p:ph type="body" idx="1"/>`) {
		t.Errorf("body 슬롯의 <p:ph type=\"body\" idx=\"1\"/>가 없다\n%s", xml)
	}
	// 플레이스홀더는 표식일 뿐 — 좌표는 여전히 잰 rect에서 나온다 (self-closing nvPr이 아님을
	// 확인했으니 off/ext가 사라지지 않았는지도 본다).
	if !strings.Contains(xml, `<a:off x="609600" y="609600"/>`) {
		t.Error("플레이스홀더 도형이 잰 좌표를 잃었다 — ph는 표식일 뿐 xfrm을 대체하지 않는다")
	}
	// nvPr이 self-closing이 아니라 ph를 감싼다 (골든 잡음 방지 — 정확한 여닫음).
	if strings.Count(xml, "<p:nvPr>") != 2 || strings.Count(xml, "</p:nvPr>") != 2 {
		t.Errorf("슬롯 도형 2개의 <p:nvPr>…</p:nvPr>가 정확히 나와야 한다\n%s", xml)
	}
}

// 슬롯을 쓰지 않은 덱은 예전처럼 self-closing `<p:nvPr/>`를 유지한다 (골든 불변) — 빈
// Appearance.Slot이 곧 "슬롯 아님"이고, 이 필드 때문에 기존 산출물이 달라지지 않는다.
func TestRenderNoSlotKeepsEmptyNvPr(t *testing.T) {
	deck := resolvedDeck(t, `<slide>
  <text role="title">제목</text>
  <text>본문</text>
</slide>`)
	g := measuredGeometry(t, deck,
		geom.Rect{X: 64, Y: 64, W: 1152, H: 67.2},
		geom.Rect{X: 64, Y: 155.2, W: 1152, H: 36},
	)
	xml := slideXML(t, deck, g)

	if strings.Contains(xml, "<p:ph") {
		t.Errorf("슬롯 없는 덱에 <p:ph>가 새어 나왔다\n%s", xml)
	}
	if !strings.Contains(xml, "<p:nvPr/>") {
		t.Errorf("슬롯 없는 도형은 self-closing <p:nvPr/>여야 한다\n%s", xml)
	}
}

// 슬롯을 쓴 덱은 그 레이아웃의 플레이스홀더 파트(slideLayout2.xml)를 얻고, 마스터·
// Content_Types·슬라이드 관계가 그것을 가리키도록 재배선된다 (ADR 0035 Part B) — 그래야
// 슬라이드 도형의 `<p:ph>`가 떠 있는 표식이 아니라 진짜 플레이스홀더가 된다.
func TestRenderSlotEmitsLayoutPart(t *testing.T) {
	deck := resolvedDeck(t, `<slide>
  <h1 slot="title">제목</h1>
  <p slot="body">본문</p>
</slide>`)
	g := measuredGeometry(t, deck,
		geom.Rect{X: 64, Y: 64, W: 1152, H: 67.2},
		geom.Rect{X: 64, Y: 155.2, W: 1152, H: 36},
	)
	parts, _ := renderPackage(t, NewWithGeometry(g), deck)

	// content 레이아웃 파트가 존재하고 title·body 플레이스홀더를 선언한다.
	layout, ok := parts["ppt/slideLayouts/slideLayout2.xml"]
	if !ok {
		t.Fatalf("슬롯 레이아웃 파트 slideLayout2.xml가 없다")
	}
	for _, want := range []string{
		`<p:cSld name="content">`,
		`<p:ph type="title"/>`,
		`<p:ph type="body" idx="1"/>`,
	} {
		if !strings.Contains(layout, want) {
			t.Errorf("slideLayout2.xml에 %q가 없다\n%s", want, layout)
		}
	}
	// 레이아웃 파트의 관계 파일은 마스터를 가리킨다.
	if lr, ok := parts["ppt/slideLayouts/_rels/slideLayout2.xml.rels"]; !ok || !strings.Contains(lr, "slideMaster1.xml") {
		t.Errorf("slideLayout2.xml.rels가 마스터를 가리키지 않는다: %q", lr)
	}
	// Content_Types가 새 레이아웃 파트를 선언한다.
	if !strings.Contains(parts[contentTypesPart], `/ppt/slideLayouts/slideLayout2.xml`) {
		t.Error("Content_Types에 slideLayout2.xml Override가 없다")
	}
	// 마스터가 title/body 프로토타입 + sldLayoutId + txStyles를 얻는다.
	master := parts[masterPart]
	for _, want := range []string{
		`<p:ph type="title"/>`,
		`<p:ph type="body" idx="1"/>`,
		`<p:sldLayoutId id="2147483650" r:id="rId3"/>`,
		"<p:txStyles>",
	} {
		if !strings.Contains(master, want) {
			t.Errorf("마스터에 %q가 없다", want)
		}
	}
	// 마스터 관계가 슬롯 레이아웃 파트를 rId3로 가리킨다 (rId1=layout1·rId2=theme 뒤).
	if !strings.Contains(parts[masterRelsPart], `Id="rId3"`) || !strings.Contains(parts[masterRelsPart], `slideLayout2.xml`) {
		t.Errorf("마스터 관계가 slideLayout2를 rId3로 가리키지 않는다: %q", parts[masterRelsPart])
	}
	// 슬롯 슬라이드의 관계가 blank가 아니라 자기 레이아웃 파트를 가리킨다.
	if !strings.Contains(parts[slideRelsPart(1)], `Target="../slideLayouts/slideLayout2.xml"`) {
		t.Errorf("슬롯 슬라이드가 slideLayout2를 가리키지 않는다: %q", parts[slideRelsPart(1)])
	}
}

// 슬롯을 쓰지 않은 덱은 슬롯 레이아웃 파트를 내지 않고, 마스터·Content_Types·슬라이드 관계가
// 예전 그대로다 (골든 불변) — SlotLayouts가 nil이라 range·조건이 전부 꺼진다.
func TestRenderNoSlotKeepsSingleBlankLayout(t *testing.T) {
	deck := resolvedDeck(t, `<slide>
  <text role="title">제목</text>
  <text>본문</text>
</slide>`)
	g := measuredGeometry(t, deck,
		geom.Rect{X: 64, Y: 64, W: 1152, H: 67.2},
		geom.Rect{X: 64, Y: 155.2, W: 1152, H: 36},
	)
	parts, _ := renderPackage(t, NewWithGeometry(g), deck)

	if _, ok := parts["ppt/slideLayouts/slideLayout2.xml"]; ok {
		t.Error("슬롯 없는 덱에 slideLayout2.xml가 새어 나왔다")
	}
	master := parts[masterPart]
	// 마스터 spTree에 플레이스홀더·txStyles가 없어야 한다 (blank 마스터 유지).
	if strings.Contains(master, "<p:ph") || strings.Contains(master, "<p:txStyles>") {
		t.Error("슬롯 없는 덱의 마스터에 플레이스홀더/txStyles가 새어 나왔다")
	}
	if strings.Count(master, "<p:sldLayoutId ") != 1 {
		t.Error("슬롯 없는 덱의 마스터는 sldLayoutId가 하나(blank)여야 한다")
	}
	// 슬라이드 관계는 예전처럼 blank를 가리킨다.
	if !strings.Contains(parts[slideRelsPart(1)], `Target="../slideLayouts/slideLayout1.xml"`) {
		t.Errorf("슬롯 없는 슬라이드가 blank 레이아웃을 가리키지 않는다: %q", parts[slideRelsPart(1)])
	}
}
