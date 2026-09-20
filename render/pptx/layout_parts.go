package pptx

import (
	"fmt"

	"github.com/k2sebeom/balpyo/soml"
)

// slotLayoutPart는 슬롯 모드 슬라이드가 참조하는 슬라이드 레이아웃 파트 하나의 명세다 (ADR 0035).
//
// blank 레이아웃(slideLayout1)은 흐름 모드 슬라이드가 그대로 쓰고, 슬롯을 실제로 쓴 슬라이드는
// 자기 레이아웃의 플레이스홀더 집합을 선언한 이 파트를 가리킨다 — 그래야 슬라이드 도형의
// `<p:ph>`가 **떠 있는 표식이 아니라** 레이아웃 플레이스홀더에 연결된 진짜 플레이스홀더가 된다
// (PowerPoint의 개요·초기화·선택 창이 그렇게 인식한다).
//
// 이 목록이 비면(슬롯을 쓴 슬라이드가 없으면) 마스터·Content_Types·슬라이드 관계가 예전과
// 바이트 동일하게 나온다 — 기존 골든 불변의 근거다 (buildParts의 조건들과 같은 태도).
type slotLayoutPart struct {
	// N은 파트 번호다 (2부터 — slideLayout1은 blank가 차지한다).
	N int
	// File은 파트 파일 이름이다 (`slideLayout2.xml`) — 슬라이드 관계와 마스터 관계가 이 이름을 가리킨다.
	File string
	// Layout은 이 파트가 대응하는 슬라이드 레이아웃이다 (`p:cSld@name`으로도 나간다).
	Layout soml.SlideLayout
	// MasterRelID는 마스터 관계 파일에서 이 파트를 가리키는 rId다 (rId1=layout1·rId2=theme 뒤).
	MasterRelID string
	// SldLayoutID는 마스터 sldLayoutIdLst의 유일 id다 (2147483650부터 — layout1이 …649를 쓴다).
	SldLayoutID int
	// Placeholders는 이 레이아웃이 선언하는 플레이스홀더 집합이다 (soml.LayoutSlots 순서 그대로).
	Placeholders []layoutPlaceholder
}

// layoutPlaceholder는 레이아웃 파트의 `<p:sp>` 플레이스홀더 하나다.
//
// **xfrm을 내지 않는다** — 좌표는 마스터의 title/body 프로토타입에서 상속받고, 슬라이드는
// 잰 rect로 덮어쓴다 (ADR 0035 선택지 3: 레이아웃 플레이스홀더의 위치는 슬라이드에 영향을
// 주지 않는다, 잰 박스가 좌표를 소유한다). 그래서 여기서 필요한 것은 **연결을 위한 type/idx뿐**이다.
type layoutPlaceholder struct {
	// ID는 이 레이아웃 spTree 안의 `cNvPr@id`다 (2부터 — id 1은 spTree 그룹).
	ID int
	// Name은 `cNvPr@name` = 선택 창 이름이다 (슬롯 이름 — 레이아웃 안에서 유일하다).
	Name string
	// Type은 `p:ph@type`이고, HasIdx가 참일 때만 Idx를 `@idx`로 낸다 (0 = 속성 생략).
	Type   string
	HasIdx bool
	Idx    int
}

// deckSlotLayouts는 슬롯 모드 슬라이드가 쓰는 레이아웃을 어휘 순서로 distinct 수집해
// 레이아웃 파트 명세로 만든다. 슬롯을 쓴 슬라이드가 없으면 nil이다 (골든 불변).
//
// rId·sldLayoutId 배정은 blank 레이아웃(rId1·id 2147483649)과 theme(rId2)를 밀지 않는
// 자리에서 시작한다 — 마스터 관계 rId3부터, sldLayoutId는 2147483650부터. 어휘 순서
// (soml.SlideLayouts)로 도므로 파트 번호가 실행마다 같다 (골든 결정성).
func deckSlotLayouts(deck *soml.Deck) []slotLayoutPart {
	used := map[soml.SlideLayout]bool{}
	for _, slide := range deck.Slides {
		if slideUsesSlots(slide) {
			used[slide.Layout] = true
		}
	}
	if len(used) == 0 {
		return nil
	}
	var parts []slotLayoutPart
	n := 2                    // slideLayout1 = blank
	masterRel := 3            // rId1 = layout1, rId2 = theme
	sldLayoutID := 2147483650 // layout1 = 2147483649
	for _, layout := range soml.SlideLayouts() {
		if !used[layout] {
			continue
		}
		slots := soml.LayoutSlots(layout)
		phs := make([]layoutPlaceholder, len(slots))
		for i, slot := range slots {
			phs[i] = layoutPlaceholder{
				ID:     i + 2, // id 1 = spTree 그룹
				Name:   slot.Name,
				Type:   slot.PhType,
				HasIdx: slot.PhIndex != 0,
				Idx:    slot.PhIndex,
			}
		}
		parts = append(parts, slotLayoutPart{
			N:            n,
			File:         fmt.Sprintf("slideLayout%d.xml", n),
			Layout:       layout,
			MasterRelID:  fmt.Sprintf("rId%d", masterRel),
			SldLayoutID:  sldLayoutID,
			Placeholders: phs,
		})
		n++
		masterRel++
		sldLayoutID++
	}
	return parts
}

// slideUsesSlots는 슬라이드가 슬롯 모드인지다 — **직계 자식**이 하나라도 슬롯을 채웠는지 본다
// (render/web.slideUsesSlots와 같은 판정 — 슬롯은 직계 자식에만 실린다, core.resolveSlot).
func slideUsesSlots(slide *soml.Slide) bool {
	for _, node := range slide.Content {
		if elem, ok := node.(*soml.Element); ok && elem.Appearance != nil && elem.Appearance.Slot != "" {
			return true
		}
	}
	return false
}

// slideLayoutFile은 슬라이드가 관계에서 가리킬 레이아웃 파트 파일 이름이다.
//
// 슬롯 모드 슬라이드는 자기 레이아웃의 플레이스홀더 파트를, 흐름 모드 슬라이드는 blank
// (slideLayout1)를 가리킨다 — 한 레이아웃을 두 슬라이드가 흐름·슬롯으로 나눠 써도 각자
// 맞는 파트에 연결된다 (웹의 `:not(.balpyo-slots)`/`.balpyo-slots` 배타성과 같은 그림).
func slideLayoutFile(slide *soml.Slide, slotLayouts []slotLayoutPart) string {
	if slideUsesSlots(slide) {
		for _, p := range slotLayouts {
			if p.Layout == slide.Layout {
				return p.File
			}
		}
	}
	return "slideLayout1.xml"
}

// masterPlaceholders는 슬롯 레이아웃이 상속받는 마스터 title/body 프로토타입의 좌표(EMU)다.
//
// 표준 PowerPoint 마스터의 관례를 따른다 (좌우 0.5in 여백, 제목 위·본문 아래). 슬라이드가
// 잰 rect로 덮어쓰므로 이 좌표는 **레이아웃 썸네일에만** 보이고 산출물 픽셀에는 닿지 않는다
// (ADR 0035 선택지 3) — 노트 마스터 좌표(notesMasterLayout)와 같은 처지다.
type masterPlaceholders struct {
	TitleX, TitleY, TitleCX, TitleCY int64
	BodyX, BodyY, BodyCX, BodyCY     int64
}

// masterPlaceholderLayout은 슬라이드 크기 안에 title/body 프로토타입 자리를 잡는다.
func masterPlaceholderLayout(slideCX, slideCY int64) masterPlaceholders {
	const margin int64 = EMUPerInch / 2    // 좌우 0.5in
	var titleY int64 = EMUPerInch * 3 / 10 // 제목 상단 0.3in
	var titleCY int64 = EMUPerInch * 5 / 4 // 제목 높이 1.25in
	bodyY := titleY + titleCY + EMUPerInch/5
	width := slideCX - 2*margin
	return masterPlaceholders{
		TitleX: margin, TitleY: titleY, TitleCX: width, TitleCY: titleCY,
		BodyX: margin, BodyY: bodyY, BodyCX: width, BodyCY: slideCY - bodyY - margin,
	}
}
