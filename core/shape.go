package core

import (
	"strconv"
	"strings"

	"github.com/k2sebeom/balpyo/soml"
)

// 이 파일은 **도형 외관 속성의 값 해석**이다 — 파서가 이름만 본 외관 속성을 확정된 도형 외관(soml.ShapeStyle)으로 눕힌다.
//
// 텍스트 외관 해석(resolveText)·배치 해석(resolveLayout)과 나란한 세 번째 갈래다. 한 요소가
// 여러 채널을 함께 쓰는 구조는 그대로다 (elemAppearance — 대입이 아니라 필드 채우기):
// `<shape grow="1" fill="surface">`는 배치와 외관을 동시에 쓴 것이고, 도형 안 `<text>`는
// 기존 텍스트 경로를 그대로 지난다 (부모가 도형이라는 사실이 텍스트 해석을 바꾸지 않는다 —
// 바꾸는 것은 상속이고 상속은 이 언어에 없다).
//
// 속성 8종 + 박스 8종처럼 보이지만 실제로는 **값 문법 몇 개의 표**다: 색(토큰·리터럴·
// `none`), 축약(border), 토큰-또는-수치(radius), 토큰만(shadow), 0~1 실수(opacity),
// 각도(rotate), 닫힌 어휘(type·flip). 박스(`pad`·크기·`ratio`)는 **레이아웃 컨테이너와 같은
// 헬퍼**를 지난다 (boxAttr) — 배치 값 문법이 두 벌이 되면 같은 표기가 두 곳에서 해석되고,
// 그 둘이 갈라지는 것은 배치 하나가 웹에서만 먹는 것으로만 드러난다.
//
// 값 문법 위에 **캐스케이드**가 얹힌다 (resolveShape): 인라인 패치를 만들고 테마의 두 단
// (컴포넌트 기본값·variant) 위에 병합한다. 그래서 속성이 없는 도형도 완전한 외관을 갖는다 —
// 병합 기계는 텍스트와 공유하는 cascade.go에 있다.

// resolveShape는 `<shape>`의 외관을 4단 캐스케이드로 확정한다:
//
//	인라인 속성 > variant > 컴포넌트 기본값 > (역할 없음)
//
// 텍스트와 갈라지는 자리는 **바닥**이다: 도형에는 역할이 없으므로 컴포넌트 기본값이 가장
// 아래 단이다 (shapeComponentDefault). 병합은 mergeShape 한 곳에서만 일어나고, 우선순위는
// 아래 세 줄의 순서가 전부다 — resolveText와 같은 형태이며 다른 어디에도 순서가 없어야 한다.
//
// 인라인 단을 **먼저 훑는** 이유는 순서가 아니라 입력이다: `radius` 검증이 형제 속성 `type`을
// 보고(radius의 주석), `id`와 박스는 캐스케이드 밖이라 style에 직접 채워진다. 우선순위는
// 훑는 순서가 아니라 병합 순서가 정한다.
func (r *resolver) resolveShape(elem *soml.Element) {
	style := soml.ShapeStyle{Type: r.shapeType(elem)}
	inline := r.shapeAttrs(elem, &style)
	backdrop := soml.ShapeHoldsLayout(elem)
	mergeShape(&style, themeShapePatch(r.comps.shapeDefault, style.Type, backdrop))
	mergeShape(&style, themeShapePatch(r.shapeVariant(elem), style.Type, backdrop))
	mergeShape(&style, inline)
	if style.Type == soml.PathShapeType && style.Path == nil {
		if _, ok := elem.Attr("d"); !ok {
			r.errorf(elem.NameSpan, "a type=%q shape needs d — its outline in a 0–100 box (e.g. d=%q)", soml.PathShapeType, "M0,100 L50,0 L100,100 Z")
		}
	}
	elemAppearance(elem).Shape = &style
	r.shapeInsetTextHeight(elem, &style)
}

// shapeInsetTextHeight는 **텍스트 영역이 박스보다 좁은 프리셋**(soml.ShapeInsetText)이 내용을
// 담으면 확정 높이를 요구한다.
//
// 근거는 기준의 순환이다: 그 여덟 프리셋의 텍스트 영역은 자기 박스에 대한 **백분율**이므로
// (render/web의 TextRect) 영역을 잡으려면 높이가 먼저 정해져 있어야 하는데, 내용이 높이를
// 정하는 도형에서는 내용을 놓을 자리가 내용의 크기에 달리게 된다. 웹은 그 영역을 절대 배치
// 래퍼로 잡으므로 높이가 없으면 도형이 납작해지고 — 진단하지 않으면 그 납작함이 저자에게는
// "글자가 사라졌다"로 보인다.
//
// 진단하는 층이 값 층인 이유는 **고칠 사람이 저자**라는 것이다: 렌더러가 멈추면 어느
// 타깃에서 무엇이 잘못됐는지가 산출물 쪽 말로 나오고, 웹·PPTX가 각자 같은 규칙을 들게 된다.
// rounded-rect가 여기 걸리지 않는 것이 이 규칙의 좁음을 말해 준다 — 인셋이 px이라 패딩으로
// 받으므로 내용이 높이를 정하는 카드가 그대로 산다.
func (r *resolver) shapeInsetTextHeight(elem *soml.Element, style *soml.ShapeStyle) {
	if !soml.ShapeInsetText(style.Type) || style.Box.Height != nil || style.Box.Ratio != nil {
		return
	}
	for _, child := range elem.Children {
		if _, ok := child.(*soml.Element); !ok {
			// 문자 데이터는 파서가 이미 막았다 (도형의 내용 모델은 블록이다).
			continue
		}
		r.errorf(elem.NameSpan, "a type=%q shape with content needs h or ratio (e.g. h=\"120\") — this preset's text area is a percentage of the box height, so letting the content set the height would be circular",
			style.Type)
		return
	}
}

// themeShapePatch는 테마 단(컴포넌트 기본값·variant)의 패치를 이 도형에 맞게 다듬는다 —
// `rounded-rect`가 아닌 프리셋에서는 `radius`를, 배경 모드에서는 `rotate`를 **조용히 버린다**.
//
// 같은 값이 인라인 단에서는 에러인데(radius·rotate의 주석) 여기서는 침묵인 근거는 **저자가
// 고칠 수 있는가**다: 테마 기본값은 저자가 손댈 수 없으므로 진단이 고칠 방법을 말할 수 없고,
// 진단을 내면 `<shape type="rect"/>`나 배치를 담은 도형이 무조건 에러가 된다. 버리는 것이
// 안전한 이유는 두 경우 모두 값을 남겨 두면 **웹만** 달라지기 때문이다 — 다른 프리셋에는
// 반경 조정값을 실을 곳이 아예 없고(`prstGeom/avLst`), 배경 도형의 회전은 형제가 된 자식이
// 물려받지 않는다. 버리는 쪽이 두 타깃을 같게 만든다. 자세한 근거는 shapeComponentDefault에
// 있다.
func themeShapePatch(patch shapePatch, shapeType string, backdrop bool) shapePatch {
	if shapeType != roundedRectType {
		patch.radius = nil
	}
	if backdrop {
		patch.rotate = nil
	}
	return patch
}

// shapeVariant는 `variant`를 테마의 **도형** variant 표에서 찾는다 — 모르는 이름은 에러가
// 아니라 경고 + 빈 패치(컴포넌트 기본 외관 폴백)다 (텍스트와 같은 정책이고
// 문구도 같은 곳에서 나온다 — warnUnknownVariant).
func (r *resolver) shapeVariant(elem *soml.Element) shapePatch {
	attr, ok := r.variantAttr(elem)
	if !ok {
		return shapePatch{}
	}
	patch, known := r.shapeVariantPatch(attr.Value)
	if !known {
		r.warnUnknownVariant(attr, r.shapeVariantNames())
		return shapePatch{}
	}
	return patch
}

// shapeVariantPatch·shapeVariantNames는 도형 variant 단을 로드된 표(r.comps)에서 읽는다 —
// 텍스트 판(resolve.go)과 나란한 자리이고, 요소 종류가 어느 표를 볼지 정하는 계약은 그대로다.
func (r *resolver) shapeVariantPatch(name string) (shapePatch, bool) {
	for _, v := range r.comps.shapeVariants {
		if v.name == name {
			return v.patch, true
		}
	}
	return shapePatch{}, false
}

func (r *resolver) shapeVariantNames() []string {
	names := make([]string, len(r.comps.shapeVariants))
	for i, v := range r.comps.shapeVariants {
		names[i] = v.name
	}
	return names
}

// shapeAttrs는 속성 한 벌을 훑어 **인라인 외관 패치**를 만든다 (inlinePatch의 도형 판).
//
// 캐스케이드 밖 필드(`id`와 박스)는 패치가 아니라 style에 직접 채운다 — 테마가 정할 수 없는
// 값이므로(식별자·배치) 패치에 담으면 병합이 다룰 필요 없는 필드를 지게 되고, "패치의 필드
// 목록 = 캐스케이드가 다루는 필드"라는 읽는 근거가 흐려진다.
//
// 값 어휘 검사와 진단은 속성별 해석 함수가 그대로 소유한다 — 틀린 값은 패치에 실리지 않으므로
// 아래 단(테마 기본값)이 남는다 (틀린 role이 기본 역할로 계속하는 것과 같은 정책).
func (r *resolver) shapeAttrs(elem *soml.Element, style *soml.ShapeStyle) shapePatch {
	var patch shapePatch
	for _, attr := range elem.Attrs {
		switch attr.Name {
		case "type":
			// 위에서 해석했다 — 여기서 다시 보면 진단이 두 번 난다.
		case "id":
			if id, ok := r.shapeID(attr, elem); ok {
				style.ID = id
			}
		case "alt":
			// 캐스케이드 밖이다 — 저자의 내용이므로 테마도 variant도 얹지 않는다 (alt.go).
			if alt, ok := r.altName(attr); ok {
				style.Alt = alt
			}
		case "d":
			// 경로는 `type="path"`의 기하다 — 다른 종류에는 그릴 자리가 없다 (프리셋의 기하는 프리셋이 정한다).
			if style.Type != soml.PathShapeType {
				r.errorf(attr.NameSpan, "d is the outline of type=%q shapes — this shape is type=%q (use type=%q to draw a custom outline)",
					soml.PathShapeType, style.Type, soml.PathShapeType)
				continue
			}
			if cmds, ok := r.pathData(attr); ok {
				style.Path = cmds
			}
		case "variant":
			// 캐스케이드 둘째 단이라 shapeVariant가 소유한다 — 여기서 또 보면 모르는 variant
			// 경고가 두 번 난다.
		case "fill":
			if fill, ok := r.shapeFill(attr); ok {
				patch.fill = &fill
			}
		case "border":
			if border, ok := r.border(attr); ok {
				patch.border = &border
			}
		case "radius":
			if radius, ok := r.radius(attr, style.Type); ok {
				patch.radius = &radius
			}
		case "shadow":
			if shadow, ok := r.shadow(attr); ok {
				patch.shadow = &shadow
			}
		case "opacity":
			if opacity, ok := r.opacity(attr); ok {
				patch.opacity = &opacity
			}
		case "rotate":
			// 배경 모드에서는 값 문법을 보기 전에 막는다 — 값이 아니라
			// 속성과 내용의 **조합**이 문제이므로 진단은 속성 이름을 가리키고, 실린 각도는
			// 남기지 않는다 (아래 themeShapePatch가 테마 단의 각도도 버린다).
			if soml.ShapeHoldsLayout(elem) {
				r.errorf(attr.NameSpan, "a shape that holds layout cannot use rotate — in PPTX the children become sibling shapes and do not inherit the rotation, while the web rotates the box with its children, so the same SOML would look different in the two targets (remove the rotation, or drop the container and hold only text)")
				continue
			}
			if rotate, ok := r.rotate(attr); ok {
				patch.rotate = &rotate
			}
		case "flip":
			if flip, ok := r.flip(attr); ok {
				patch.flip = &flip
			}
		default:
			// 박스 속성(`pad`·크기·`ratio`)은 컨테이너와 공유하는 헬퍼가 해석하고, 자식 배치
			// 속성과 슬롯 배정은 resolveLayout·resolveSlot이 이미 지났다 (요소 종류와 무관한
			// 한 벌). 그 셋도 아니면 카탈로그와 값 층이 어긋난 것이다 — 조용히 지나가면
			// 저자가 쓴 외관이 사라진다.
			//
			// `slot`을 따로 적는 이유: 파서는 그것을 childPlacementAttrs **밖에서** 잇는다
			// (withChildPlacement의 마지막 한 칸 — Placement 해석과 섞이지 않게 한 의도다).
			// 그래서 배치 목록만 면제하면 슬롯을 쓴 도형이 엔진 버그 진단을 받는다.
			if r.boxAttr(&style.Box, attr) ||
				containsName(soml.ChildPlacementAttrs(), attr.Name) ||
				attr.Name == soml.SlotAttr || attr.Name == soml.EnterAttr {
				continue
			}
			r.catalogGap(elem, attr)
		}
	}
	return patch
}

// shapeType은 `type`을 해석한다 — 12종 닫힌 어휘다 (soml.ShapeTypes).
//
// **속성이 없으면 에러다.** 기본값 `rect`로 계속하는 길과 저울질했고 버렸다: 도형의 종류는
// 추측할 수 없고, `<shape/>`가 조용히 사각형이 되면 저자의 오타(`type="elipse"`)와 구별되지
// 않는다. 이 정책이 `role`의 결(진단하고 기본값으로 계속)과 갈리는 근거는 **기본값의
// 존재**다: `<p>` ≡ `<text role="body">`는 언어가 정한 등가라 생략에 뜻이 있지만,
// 도형에는 그런 등가가 없다.
//
// 어휘 밖의 값도 에러다 (기본값으로 계속하지 않는다) — 잘못된 종류로 그린 도형은 저자가
// 의도한 그림이 아니고, 그 자리에 사각형을 놓는 것은 오타를 산출물에 통과시키는 일이다.
func (r *resolver) shapeType(elem *soml.Element) string {
	attr, ok := elem.Attr("type")
	if !ok {
		r.errorf(elem.NameSpan, "<%s> needs a type (available types: %s) — the shape type cannot be guessed",
			soml.ShapeElement, shapeTypeList())
		return ""
	}
	if !r.hasValue(attr) {
		return ""
	}
	for _, name := range soml.ShapeTypes() {
		if name == attr.Value {
			return name
		}
	}
	r.errorf(attr.ValueSpan, "unknown shape type %q%s (available types: %s)", attr.Value, didYouMean(attr.Value, shapeTypeList()), shapeTypeList())
	return ""
}

// shapeID는 `id`를 해석한다 — 값 층이 지금 할 수 있는 검사는 **중복**이다.
//
// 범위는 **슬라이드**다 (resolver가 슬라이드 하나의 상태이므로 자리가 곧 범위다). 덱 전역
// 유일을 요구하는 길을 버린 근거: 참조(`<arrow from="db">`)는 슬라이드를 넘지 않으므로
// 전역 유일은 저자가 슬라이드마다 같은 이름(`db`)을 쓰지 못하게 막을 뿐이다.
//
// 참조가 붙기 전에 검사하는 이유: 그때 처음 막으면 중복 id로 이미 쓴 덱이 깨진다. 지금은
// id가 산출물에 아무것도 그리지 않아 고치는 비용이 이름 하나다.
func (r *resolver) shapeID(attr soml.Attr, elem *soml.Element) (string, bool) {
	if !r.hasValue(attr) {
		return "", false
	}
	if first, dup := r.ids[attr.Value]; dup {
		r.errorf(attr.ValueSpan, "id %q already exists on this slide (%s) — a reference could not tell which shape it means",
			attr.Value, first.Start)
		return "", false
	}
	if r.ids == nil {
		r.ids = map[string]soml.Span{}
	}
	r.ids[attr.Value] = attr.ValueSpan
	if r.idElems == nil {
		r.idElems = map[string]*soml.Element{}
	}
	r.idElems[attr.Value] = elem
	return attr.Value, true
}

// fill은 채움을 해석한다 — **토큰명 → 리터럴 → `none`** 순으로 본다.
//
// 색 문법은 텍스트 `color`와 **같은 코드**를 지난다 (colorValue): 토큰은 이름으로 남고
// (soml.Color — RGB로 눕히면 `schemeClr`를 만들 수 없다), 리터럴은 대문자 6자리로
// 정규화되고, 색 함수(`gradient(…)` 등)도 같은 자리에서 갈린다. 색 문법이
// 속성마다 따로 있으면 같은 값이 자리에 따라 다르게 읽힌다.
func (r *resolver) fill(attr soml.Attr) (soml.Fill, bool) {
	if !r.hasValue(attr) {
		return soml.Fill{}, false
	}
	if attr.Value == noneValue {
		return soml.NoFill(), true
	}
	color, ok := r.colorValue(attr.Name, attr.Value, attr.ValueSpan)
	if !ok {
		return soml.Fill{}, false
	}
	return soml.ColorFill(color), true
}

// patternFunc는 무늬 채움의 함수 이름이다.
const patternFunc = "pattern"

// shapeFill은 도형의 채움이다 — fill의 모든 값에 더해 무늬(`pattern(<무늬> <앞색> <뒷색>)`)를 받는다.
// 무늬를 도형에만 여는 이유: 글자 · 선 · 표 칸 · 배경에는 두 타깃에 같은 원어가 없거나 길이 하나 더 생긴다.
func (r *resolver) shapeFill(attr soml.Attr) (soml.Fill, bool) {
	if r.hasValue(attr) && strings.HasPrefix(attr.Value, patternFunc+"(") {
		return r.patternFill(attr.Value, attr.ValueSpan)
	}
	return r.fill(attr)
}

// patternFill은 `pattern(<무늬> <앞색> <뒷색>)`을 해석한다. 두 색은 그라디언트 스톱과 같은 문법이다 (`bg/0`).
func (r *resolver) patternFill(raw string, span soml.Span) (soml.Fill, bool) {
	if !strings.HasSuffix(raw, ")") {
		r.errorf(span, "color function %s(…) is missing the closing %q (%q)", patternFunc, ")", raw)
		return soml.Fill{}, false
	}
	inner := raw[len(patternFunc)+1 : len(raw)-1]
	fields := strings.Fields(inner)
	if len(fields) != 3 || strings.ContainsAny(inner, "(,") {
		r.errorf(span, "pattern takes a pattern name and two colors, `pattern(<pattern> <foreground> <background>)` (%q) — e.g. pattern(diagonal-up accent5 bg/0)", raw)
		return soml.Fill{}, false
	}
	kinds := soml.PatternKinds()
	names := make([]string, len(kinds))
	known := false
	for i, k := range kinds {
		names[i] = string(k)
		known = known || string(k) == fields[0]
	}
	ok := true
	if !known {
		list := strings.Join(names, ", ")
		r.errorf(span, "unknown pattern %q%s (available values: %s)", fields[0], didYouMean(fields[0], list), list)
		ok = false
	}
	colors, cok := r.gradientStops("fill", fields[1:], span)
	if !ok || !cok {
		return soml.Fill{}, false
	}
	return soml.Fill{Pattern: &soml.Pattern{Kind: soml.PatternKind(fields[0]), Fg: colors[0], Bg: colors[1]}}, true
}

// noneValue는 "그리지 않음"이다 — `fill`·`border`가 공유한다. 상수로 두는 이유는 두 곳의
// 표기가 갈라지지 않게 하는 것이다.
const noneValue = "none"

// borderParts는 `border` 축약의 조각 수다 — `<굵기> <스타일> <색>`.
const borderParts = 3

// border는 테두리를 해석한다 — 조각 수와 **순서를 고정한다** (굵기·스타일·색).
//
// 순서를 자유롭게 두는 길을 버린 근거는 진단이다: `border="solid 3"`에서 무엇이 빠졌는지
// 말하려면 조각의 자리가 정해져 있어야 한다. 자리가 없으면 "스타일 같아 보이는 조각"을
// 찾아 헤매는 추측 논리가 값 층에 들어오고, 그 추측은 저자마다 다른 표기를 만든다.
// 순서는 CSS `border` 축약과 같다 — 저자가 이미 아는 순서다.
func (r *resolver) border(attr soml.Attr) (soml.Border, bool) {
	if !r.hasValue(attr) {
		return soml.Border{}, false
	}
	if attr.Value == noneValue {
		return soml.NoBorder(), true
	}
	parts := splitValues(attr.Value)
	// `gradient(…)`가 낀 축약은 **스타일과 색을 함께 대신한다** — 조각이 둘이다
	// (`<굵기> gradient(…)`). 그래서
	// 그라디언트가 있으면 3조각 규칙 대신 이 갈래로 간다. `shade`/`tint`/`alpha`는 색
	// 함수지만 **색 자리 하나**를 채우므로 여기 걸리지 않는다 (3조각 그대로, 색 조각이
	// colorValue를 지난다) — 그라디언트만 축약의 모양을 바꾼다.
	for _, part := range parts {
		if strings.HasPrefix(part.text, "gradient(") {
			return r.gradientBorder(attr, parts)
		}
	}
	if len(parts) != borderParts {
		r.errorf(attr.ValueSpan, "border has 3 parts, `<width> <style> <color>` (%q has %d) — e.g. %q, or %q",
			attr.Value, len(parts), "1 solid muted", noneValue)
		return soml.Border{}, false
	}
	border := soml.Border{}
	width, okW := r.borderWidth(parts[0], attr)
	style, okS := r.borderStyle(parts[1], attr)
	color, okC := r.colorValue(attr.Name, parts[2].text, parts[2].span(attr))
	if !okW || !okS || !okC {
		// 조각 셋을 모두 진단한 뒤에 버린다 — 진단 하나가 다른 조각의 진단을 삼키면 저자가
		// 빌드를 조각 수만큼 돌게 된다 (값 층 전체의 정책).
		return soml.Border{}, false
	}
	border.Width, border.Style, border.Color = width, style, color
	return border, true
}

// gradientBorder는 그라디언트 테두리를 해석한다 — `<굵기> gradient(…)` 2조각이다.
//
// 그라디언트는 스타일을 함께 대신하므로 스타일 조각이 없다: 스타일은 **실선으로 고정**한다
// (파선 그라디언트 테두리는 v1에 없다 — OOXML `ln`의 `prstDash`는 색과 독립이지만 웹의 SVG
// 오버레이 구현이 복잡해진다). 저자가 `border="1 dashed gradient(…)"`처럼 스타일을
// 끼우면 3조각이 되어 gradient가 색 자리가 아닌 곳에 오고, colorValue가 그 조각의 자리를
// 짚는다.
func (r *resolver) gradientBorder(attr soml.Attr, parts []valuePart) (soml.Border, bool) {
	if len(parts) != gradientBorderParts {
		r.errorf(attr.ValueSpan, "a gradient border has 2 parts, `<width> gradient(…)` (%q has %d) — the gradient stands in for both style and color",
			attr.Value, len(parts))
		return soml.Border{}, false
	}
	width, okW := r.borderWidth(parts[0], attr)
	color, okC := r.colorValue(attr.Name, parts[1].text, parts[1].span(attr))
	if !okW || !okC {
		return soml.Border{}, false
	}
	return soml.Border{Width: width, Style: soml.BorderSolid, Color: color}, true
}

// gradientBorderParts는 그라디언트 테두리 축약의 조각 수다 — `<굵기> gradient(…)`.
const gradientBorderParts = 2

// borderWidth는 축약의 첫 조각이다 — 논리 px 수치이고 0·음수는 에러다.
//
// 0을 막는 이유: `border="0 solid muted"`는 CSS에서 선이 사라지는 선언이고, 그 표기의 뜻은
// "테두리 없음"이다. 같은 뜻의 표기가 둘이면(`none`) 저자가 어느 쪽을 썼는지에 따라 산출물
// 바이트가 갈라진다 — 웹은 `border: 0`을, PPTX는 `a:ln w="0"`을 낼지 `a:noFill`을 낼지
// 정해야 하고 그 판단이 렌더러마다 생긴다.
func (r *resolver) borderWidth(part valuePart, attr soml.Attr) (float64, bool) {
	span := part.span(attr)
	if unit := numberSuffix(part.text); unit != "" {
		r.errorf(span, "the border width is a unitless number in logical px (drop the %[2]q from %[1]q) — px, pt and em suffixes are not allowed",
			part.text, unit)
		return 0, false
	}
	width, err := strconv.ParseFloat(part.text, 64)
	if err != nil {
		r.errorf(span, "the border width is not a number (%q) — the part order is `<width> <style> <color>`", part.text)
		return 0, false
	}
	if width <= 0 {
		r.errorf(span, "the border width must be greater than 0 (%q) — use border=%q to remove the border",
			part.text, noneValue)
		return 0, false
	}
	return width, true
}

// borderStyle은 축약의 둘째 조각이다 — 어휘 3종이다 (soml.BorderStyles).
func (r *resolver) borderStyle(part valuePart, attr soml.Attr) (soml.BorderStyle, bool) {
	for _, style := range soml.BorderStyles() {
		if string(style) == part.text {
			return style, true
		}
	}
	r.errorf(part.span(attr), "unknown border style %q%s (available values: %s) — the part order is `<width> <style> <color>`",
		part.text, didYouMean(part.text, borderStyleList()), borderStyleList())
	return "", false
}

// radius는 모서리 반경을 해석한다 — 토큰(sm/md/lg) 또는 수치다.
//
// **`type="rounded-rect"`에서만 뜻이 있다.** 다른 프리셋에는 조정값을 실을 곳이 없어
// (`prstGeom/avLst`에 반경 조정값이 없다) 웹만 둥글어지고 PPTX는 각진 채 남는다 — 그
// 조용한 갈라짐을 막는 것이 이 검증이다. 선례가 둘 있다: 파서가 `<row cols>`를 막은 것
// (태그를 바꾸는 것이 고칠 방법이라 이름 층이 막았다)과 `span`이 부모를 보는 것(부모를 봐야
// 하는 검증은 값 층이 한다). 여기는 **형제 속성**을 보는 자리이므로 값 층이다.
//
// **이 제약은 도형의 것이다**: `<image radius>`는 프리셋이 아니라 자기
// 클립을 갖기 때문에 언제나 유효하고, 그래서 값 문법만 떼어 공유한다 (radiusValue).
func (r *resolver) radius(attr soml.Attr, shapeType string) (soml.Radius, bool) {
	if !r.hasValue(attr) {
		return soml.Radius{}, false
	}
	if shapeType != roundedRectType {
		// type이 아예 해석되지 않았으면(없음·오타) 그 진단이 이미 났다 — 같은 도형에
		// 두 번째 진단을 얹지 않는다 (고칠 곳이 하나다).
		if shapeType == "" {
			return soml.Radius{}, false
		}
		r.errorf(attr.NameSpan, "radius is only for type=%q (this is %q) — other presets have no radius adjustment, so only the web would be rounded",
			roundedRectType, shapeType)
		return soml.Radius{}, false
	}
	return r.radiusValue(attr)
}

// radiusValue는 반경의 **값 문법**이다 — 토큰 또는 무단위 px 수치. 값이 있다는 것은 호출자가
// 이미 확인했다 (hasValue).
//
// 도형과 이미지가 이 함수를 공유한다: 프리셋 제약은 도형의 것이지만 값 문법은 하나다 — 두 벌이
// 되면 같은 표기가 자리에 따라 다르게 읽히고, 그 갈라짐은 두 산출물을 나란히 볼 때 드러난다.
//
// 토큰 표를 수치 판정보다 먼저 보는 이유는 length와 같다 (lengthValue의 주석) — 표에 있는
// 이름은 토큰이고, 없으면 수치 문법이 판정한다.
func (r *resolver) radiusValue(attr soml.Attr) (soml.Radius, bool) {
	if token, ok := r.theme.Radius(attr.Value); ok {
		// 토큰 **이름**이 확정 값에 남는다 — 웹이 변수 참조로 내야 테마를 바꿨을 때 모서리가
		// 따라 움직인다 (soml.Radius.Token, 색·공간 토큰과 같은 근거).
		return soml.TokenRadius(attr.Value, token.Px), true
	}
	if !startsNumeric(attr.Value) {
		names := strings.Join(r.theme.RadiusNames(), ", ")
		r.errorf(attr.ValueSpan, "unknown radius token %q%s (available tokens: %s, or a number)",
			attr.Value, didYouMean(attr.Value, names), names)
		return soml.Radius{}, false
	}
	if unit := numberSuffix(attr.Value); unit != "" {
		r.errorf(attr.ValueSpan, "radius is a unitless number in logical px (drop the %[2]q from %[1]q) — px, pt and em suffixes are not allowed",
			attr.Value, unit)
		return soml.Radius{}, false
	}
	px, err := strconv.ParseFloat(attr.Value, 64)
	if err != nil {
		r.errorf(attr.ValueSpan, "radius value is not a number (%q) — use a token (%s) or a number",
			attr.Value, strings.Join(r.theme.RadiusNames(), ", "))
		return soml.Radius{}, false
	}
	if px < 0 {
		r.errorf(attr.ValueSpan, "radius cannot be negative (%q)", attr.Value)
		return soml.Radius{}, false
	}
	return soml.PxRadius(px), true
}

// roundedRectType은 반경 조정값을 갖는 유일한 프리셋이다 (radius의 검증 대상).
const roundedRectType = "rounded-rect"

// shadow는 그림자를 해석한다 — **토큰만** 받는다.
//
// CSS `box-shadow`의 4수치 축약을 여는 길을 버린 근거는 soml.ShadowToken의 주석에 있다
// (OOXML `outerShdw`와 1:1이 아니다 — 번짐의 정의가 다르고 확산에 대응 항목이 없다).
// 확정 값은 토큰 항목을 그대로 담는다: 이름은 웹의 변수 참조가 되고 파라미터는 PPTX가
// 굽는다.
func (r *resolver) shadow(attr soml.Attr) (soml.ShadowToken, bool) {
	if !r.hasValue(attr) {
		return soml.ShadowToken{}, false
	}
	token, ok := r.theme.Shadow(attr.Value)
	if !ok {
		names := strings.Join(r.theme.ShadowNames(), ", ")
		r.errorf(attr.ValueSpan, "unknown shadow token %q%s (available tokens: %s) — shadows are written as tokens, not numbers (the two targets define blur differently, so the same numbers would draw differently)",
			attr.Value, didYouMean(attr.Value, names), names)
		return soml.ShadowToken{}, false
	}
	return token, true
}

// opacity는 불투명도를 해석한다 — 0~1 실수다 (`0.8`).
//
// `%` 표기를 막는 이유: 색 변형 함수 `alpha(색, 0~1)`이 같은 범위를 쓰므로,
// `50%`를 받으면 투명도의 표기가 둘이 된다. 범위 밖도 에러다 — CSS는 조용히 자르고
// (clamp) OOXML은 알파 값 범위를 벗어난 파일을 뷰어마다 다르게 읽는다.
func (r *resolver) opacity(attr soml.Attr) (float64, bool) {
	if !r.hasValue(attr) {
		return 0, false
	}
	raw := attr.Value
	if strings.HasSuffix(raw, "%") {
		r.errorf(attr.ValueSpan, "opacity cannot use %% (%q) — write a real number from 0 to 1 (keeping one notation with the alpha() color modifier)", raw)
		return 0, false
	}
	if unit := numberSuffix(raw); unit != "" {
		r.errorf(attr.ValueSpan, "opacity is a unitless real number from 0 to 1 (drop the %[2]q from %[1]q)", raw, unit)
		return 0, false
	}
	value, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		r.errorf(attr.ValueSpan, "opacity value is not a number (%q) — write a real number from 0 to 1", raw)
		return 0, false
	}
	if value < 0 || value > 1 {
		r.errorf(attr.ValueSpan, "opacity is between 0 and 1 (%q)", raw)
		return 0, false
	}
	return value, true
}

// rotate는 회전 각도를 해석한다 — 도, 시계방향, 음수 허용이다.
//
// 접미사 `deg`를 받지 않는 이유: 단위 계약이 허용하는 접미사가 `%` 하나뿐이고 레퍼런스의
// 예시도 `rotate="45"`다 (그라디언트의 `45deg`는 함수 인자 안의 문법이라 다른 자리다).
// 각도 범위를 제한하지 않는 이유: 400도는 40도와 같은 결과이고 두 타깃이 모두 그렇게 읽는다 —
// 저자가 애니메이션 없이 400도를 적을 일은 없지만, 막으면 생성기가 만든 SOML이 이유 없이
// 실패한다.
func (r *resolver) rotate(attr soml.Attr) (float64, bool) {
	if !r.hasValue(attr) {
		return 0, false
	}
	raw := attr.Value
	if unit := numberSuffix(raw); unit != "" {
		r.errorf(attr.ValueSpan, "rotate is a unitless angle (drop the %[2]q from %[1]q) — the only allowed suffix is %%",
			raw, unit)
		return 0, false
	}
	deg, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		r.errorf(attr.ValueSpan, "rotate value is not a number (%q) — write the clockwise angle as a number (negative allowed)", raw)
		return 0, false
	}
	return deg, true
}

// flip은 대칭을 해석한다 — 어휘 3종이다 (soml.Flips).
func (r *resolver) flip(attr soml.Attr) (soml.Flip, bool) {
	if !r.hasValue(attr) {
		return "", false
	}
	for _, flip := range soml.Flips() {
		if string(flip) == attr.Value {
			return flip, true
		}
	}
	r.errorf(attr.ValueSpan, "unknown flip %q%s (available values: %s — %q is both axes)",
		attr.Value, didYouMean(attr.Value, flipList()), flipList(), string(soml.FlipBoth))
	return "", false
}

// anchor는 텍스트의 세로 앵커를 해석한다 — 어휘 3종이다 (soml.TextAnchors).
//
// 자리가 텍스트인 이유: 앵커는 텍스트 외관이고(`bodyPr@anchor`는 텍스트 본문의 속성이다),
// 도형은 자식의 앵커를 읽어 쓴다.
func (r *resolver) anchor(attr soml.Attr) (soml.TextAnchor, bool) {
	if !r.hasValue(attr) {
		return "", false
	}
	for _, anchor := range soml.TextAnchors() {
		if string(anchor) == attr.Value {
			return anchor, true
		}
	}
	// 저자는 가로 정렬(align)의 논리 이름을 세로에도 쓴다 — 대응하는 물리 이름을 짚어 준다
	// (align의 left/right 진단과 같은 태도). 세로축은 문단 방향으로 뒤집히지 않으므로
	// 논리 이름을 쓸 근거가 없다 (soml.TextAnchor의 주석).
	if physical, ok := map[string]soml.TextAnchor{
		string(soml.AlignStart): soml.AnchorTop,
		string(soml.AlignEnd):   soml.AnchorBottom,
	}[attr.Value]; ok {
		r.errorf(attr.ValueSpan, "anchor takes vertical names — instead of %q, use %q (the vertical axis does not flip with paragraph direction)",
			attr.Value, string(physical))
		return "", false
	}
	r.errorf(attr.ValueSpan, "unknown anchor %q%s (available values: %s)", attr.Value, didYouMean(attr.Value, anchorList()), anchorList())
	return "", false
}

// 진단에 실을 어휘 목록이다 — 어휘 정의(soml)와 메시지가 갈라지지 않도록 손으로 쓰지 않는다
// (roleList·boxAlignList와 같은 태도).
func shapeTypeList() string { return strings.Join(soml.ShapeTypes(), ", ") }

func borderStyleList() string {
	names := make([]string, 0, len(soml.BorderStyles()))
	for _, style := range soml.BorderStyles() {
		names = append(names, string(style))
	}
	return strings.Join(names, ", ")
}

func flipList() string {
	names := make([]string, 0, len(soml.Flips()))
	for _, flip := range soml.Flips() {
		names = append(names, string(flip))
	}
	return strings.Join(names, ", ")
}

func anchorList() string {
	names := make([]string, 0, len(soml.TextAnchors()))
	for _, anchor := range soml.TextAnchors() {
		names = append(names, string(anchor))
	}
	return strings.Join(names, ", ")
}
