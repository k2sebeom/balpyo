package core

import "github.com/k2sebeom/balpyo/soml"

// minimalTheme는 하드코딩된 최소 테마다.
//
// Iteration 10의 테마 패키지 포맷(theme.json + tokens.json + components.json)이 이 함수를
// 해체한다. 순서를 이렇게 잡은 이유: 포맷을 먼저 정하면 무엇이 실제로 필요한지 모르는 상태로
// 스키마를 고정하게 되고, 스키마는 사용자 파일이 되는 순간 바꾸기 어려워진다. 하드코딩된
// 표는 두 렌더러가 실제로 무엇을 요구하는지 드러낸 다음에 파일로 나간다.
//
// 담는 것은 이번 iteration이 쓰는 만큼이다 — 색 5종, 폰트 3종, 공간 6종, 반경 3종,
// 그림자 1종, 텍스트 역할 5종. 어휘를 늘리는 근거는 하나다: **픽스처가 실제로 쓴다.**
// 그래서 PRD §8.4의 표준 어휘 중 `accent2`는 아직 없다 (도형 픽스처가 쓰지 않는다 — 쓰지
// 않는 토큰을 넣으면 어느 값이 실제로 두 타깃에서 검증되었는지 표만 보고는 알 수 없다).
func minimalTheme() soml.Theme {
	return soml.Theme{
		Name:    "minimal",
		Colors:  minimalColors(),
		Fonts:   minimalFonts(),
		Spaces:  minimalSpaces(),
		Radii:   minimalRadii(),
		Shadows: minimalShadows(),
	}
}

// minimalColors는 색 토큰 표다.
//
// 이름은 PRD §8.4의 표준 어휘를 따른다. 강조색만 `accent`가 아니라 `accent1`인데, OOXML
// clrScheme에 accent1~6 여섯 자리가 있어 번호를 1부터 쓰면 토큰과 슬롯이 1:1로 붙는다
// (PRD §7.6의 `gradient(accent1 accent4 accent5)` 표기도 이 번호를 쓴다).
//
// 값은 흰 배경 위의 최소 테마다: 순수 검정 대신 살짝 든 검정(#1A1A1A)이 화면에서 덜 딱딱하고,
// muted와 accent1은 흰 배경 대비 4.5:1을 넘겨(각각 약 4.8:1, 4.9:1) 캡션과 소제목이
// 프로젝터에서도 읽힌다.
//
// `surface`는 도형이 요구하는 토큰이다 (PRD §8.3의 컴포넌트 기본 채움 `"fill": "surface"`) —
// 카드 배경이 흰 스테이지에서 보이려면 bg와 다른 값이어야 한다. #F3F4F6은 muted(#6B7280)와
// 같은 회색 계열의 밝은 끝이고, 그 위의 fg는 대비 15:1을 넘어 본문이 카드 안에서도 읽힌다.
// 슬롯을 lt2(참조 bg2)에 두는 이유: surface는 **배경**이므로 clrScheme의 두 번째 밝은 자리가
// 뜻이 맞고, 남은 자리 중 dk2는 이미 muted가 쓴다. 슬롯 없는 색을 하나라도 만들면 그
// 도형만 PowerPoint의 테마 색 편집을 따라가지 못한다 (soml.ColorToken의 계약).
func minimalColors() []soml.ColorToken {
	return []soml.ColorToken{
		{Name: "fg", RGB: "1A1A1A", Slot: "dk1", Ref: "tx1"},
		{Name: "bg", RGB: "FFFFFF", Slot: "lt1", Ref: "bg1"},
		{Name: "muted", RGB: "6B7280", Slot: "dk2", Ref: "tx2"},
		{Name: "surface", RGB: "F3F4F6", Slot: "lt2", Ref: "bg2"},
		{Name: "accent1", RGB: "0F65F9", Slot: "accent1", Ref: "accent1"},
	}
}

// minimalFonts는 폰트 토큰 표다.
//
// 테스트 이미지에 설치·핀된 Noto로 맞춘다 (docker/test.Dockerfile) — 텍스트 메트릭 비교는
// 폰트가 결정적일 때만 뜻이 있다. v1은 폰트를 임베드하지 않으므로(PRD §13-10) 스택은 이름과
// 폴백으로만 이루어지고, **웹과 PPTX가 같은 스택을 쓴다**: 웹은 font-family 목록으로,
// PPTX는 latin/ea 두 자리로 낸다.
//
// heading과 body가 같은 패밀리인 것은 의도다 — 최소 테마의 위계는 크기와 굵기로 만든다.
func minimalFonts() []soml.FontToken {
	return []soml.FontToken{
		{Name: soml.FontHeading, Latin: "Noto Sans", EastAsian: "Noto Sans CJK KR", Generic: "sans-serif", Scheme: "major"},
		{Name: soml.FontBody, Latin: "Noto Sans", EastAsian: "Noto Sans CJK KR", Generic: "sans-serif", Scheme: "minor"},
		{Name: soml.FontMono, Latin: "Noto Sans Mono", EastAsian: "Noto Sans CJK KR", Generic: "monospace"},
	}
}

// minimalSpaces는 공간 토큰 표다 (PRD §8.4의 `space` 스케일, 어휘 6종).
//
// 값의 근거는 색 토큰의 대비율과 같은 자리에 있다 — **수직 리듬**이다 (textRoleDefaults):
// 본문 24px, 줄박스 24×1.5 = 36px, 스테이지 720px = 본문 20줄. 사다리는 md를 본문 크기
// 24px에 못 박고 그 위아래를 6px(= 24의 ¼, 리듬의 최소 눈금) 배수로 벌린다:
//
//	xs 6(¼ md) · sm 12(½ md) · md 24(본문 한 글자) · lg 36(줄박스 한 줄) · xl 48(2 md) · 2xl 72(줄박스 두 줄)
//
// md = 24를 고른 이유는 웹 렌더러의 세로 스택 간격(render/web.stackGap)이 이미 24이기
// 때문이다 — 저자가 `gap="md"`로 적은 간격과 엔진의 기본 스택 간격이 같은 값이어야
// 레이아웃 태그를 쓰기 시작한 슬라이드의 리듬이 옆 슬라이드와 어긋나지 않는다. lg·2xl이
// 줄박스(36)의 1·2배인 것도 같은 이유다: 컨테이너 사이 여백이 글줄 격자에 얹힌다.
//
// 4·8·16·32 같은 2의 거듭제곱 사다리와 저울질했고 버렸다 — 익숙하지만 36px 줄박스의
// 약수·배수가 아니어서 여백이 글줄 격자에서 매번 미끄러진다. 리듬이 이 제품의 근거이므로
// 사다리도 리듬에서 나오는 편이 일관된다.
//
// **`ColorToken`의 clrScheme 슬롯 계약이 여기에는 없다** (soml.SpaceToken의 주석) —
// OOXML 테마 파트에 공간 개념이 없고, 배치는 좌표로 구워져 산출물에 남는다.
//
// 스테이지 여백(render/web.stagePadding, 64)은 이 표에 넣지 않았다: 산출 CSS가 변수
// 참조로 바뀌면 기존 골든 3개가 이유 없이 움직인다. 값의 출처가 두 곳이 된 사실은
// PRD §8.4의 `slidePadding` 토큰이 들어오는 Iteration 10이 합친다.
func minimalSpaces() []soml.SpaceToken {
	return []soml.SpaceToken{
		{Name: "xs", Px: 6},
		{Name: "sm", Px: 12},
		{Name: "md", Px: 24},
		{Name: "lg", Px: 36},
		{Name: "xl", Px: 48},
		{Name: "2xl", Px: 72},
	}
}

// minimalRadii는 모서리 반경 토큰 표다 (PRD §8.4의 `shape.radius-*`, 어휘 3종).
//
// 값은 공간 사다리와 **같은 눈금**에서 나온다 (minimalSpaces의 수직 리듬): 최소 눈금 6px에
// 얹혀 6 · 12 · 24이고, 각각 공간 토큰 xs·sm·md와 같은 값이다. 반경이 여백과 같은 눈금을
// 쓰는 이유는 카드가 그 둘을 나란히 보이기 때문이다 — 안쪽 여백 24px에 모서리 7px 같은 값이
// 섞이면 카드가 손으로 그린 것처럼 보인다.
//
// 사다리가 공간(6종)보다 짧은 이유는 반경이 **포화**하기 때문이다: 짧은 변의 절반을 넘으면
// 모양이 더 바뀌지 않는다 (알약이 된다). 그래서 PRD §8.4도 반경 어휘를 sm/md/lg 셋으로
// 적는다.
//
// 공간 토큰을 그대로 쓰게 하는 길(`radius="md"`가 space md를 가리키는 것)과 저울질했고
// 버렸다: 두 사다리의 뜻이 다르므로(여백은 리듬, 반경은 형태) 한쪽을 조율하면 다른 쪽이
// 끌려가고, 테마 제작자가 반경만 바꿀 방법이 사라진다.
func minimalRadii() []soml.RadiusToken {
	return []soml.RadiusToken{
		{Name: "sm", Px: 6},
		{Name: "md", Px: 12},
		{Name: "lg", Px: 24},
	}
}

// minimalShadows는 그림자 토큰 표다 (PRD §8.4의 `shape.shadow-soft`, 어휘 1종).
//
// 어휘가 하나인 근거는 색 토큰과 같다 — 픽스처가 쓰는 만큼이다 (`shadow="soft"`).
//
// 값은 "떠 있지만 떠 보이지 않는" 카드다: 아래로 4px(Dir 90도) 밀고 16px 번지며, 색은
// **fg 토큰**이고 알파 0.15다. 검정 리터럴 대신 fg를 쓰는 이유는 clrScheme 참조가 그림자에도
// 유지되어야 하기 때문이다 (soml.ShadowToken) — 어두운 테마로 바꾸면 그림자도 그 테마의
// 어두운 색을 따라간다. 거리 4는 리듬의 최소 눈금 6보다 작다: 그림자는 배치가 아니라
// 빛의 방향이고, 눈금에 맞추면 카드가 종이에서 들린 것처럼 보인다. 번짐 16 ≈ 거리의 4배가
// "soft"의 근거이고(그림자 경계가 눈에 보이지 않는 비율), 알파 0.15는 흰 배경에서 회색
// #E6E6E6 정도로 읽혀 surface(#F3F4F6)와 구별된다.
func minimalShadows() []soml.ShadowToken {
	return []soml.ShadowToken{
		{Name: "soft", Dist: 4, Dir: 90, Blur: 16, Color: soml.TokenColor("fg"), Alpha: 0.15},
	}
}

// textComponentDefault는 텍스트 컴포넌트의 태그별 기본 외관 패치다 — 캐스케이드의 셋째 단
// (PRD §8.3 components.json에서 컴포넌트의 `default`가 놓이는 자리). 태그 → 역할 대응은
// 언어의 계약(soml.TagTextRole)이지만 태그별 **기본값**은 외관이므로 테마가 소유한다 —
// Iteration 10의 components.json이 이 표만 파일로 옮기고 등가 표는 soml에 남는다.
//
// 역할 5종에 태그가 7개라 역할만으로 부족한 자리 두 곳만 정하고, 나머지 태그(그리고
// `<text>` 자신)는 **빈 패치**다 — 그래서 `<p>` ≡ `<text role="body">`가 깊은 비교로
// 정확히 성립한다. 빈 패치인 자리에 필드를 정하면 역할과 다른 값이면 등가가 깨지고, 같은
// 값이면 아무것도 증명하지 못하는 중복이다 (단의 존재는 resolveText의 병합 구조와
// cascade_test가 증명한다).
func textComponentDefault(tag string) textPatch {
	switch tag {
	case "h3":
		// 역할은 heading이되 크기 한 단 아래 — heading(36)과 body(24) 사이의 반 단계다.
		// 28을 고른 근거는 수직 리듬(textRoleDefaults): heading의 lineHeight 1.3을 그대로
		// 받으므로 줄박스가 28×1.3 ≈ 36.4px로 본문 한 줄(24×1.5 = 36px)에 얹힌다 —
		// 36/1.3 ≈ 27.7의 가장 가까운 정수다. 기하 평균 √(24×36) ≈ 29.4는 줄박스가
		// 38.2px로 리듬에서 더 멀어 버렸다.
		return textPatch{size: ptr(28.0)}
	case "blockquote":
		// 역할은 body, 색만 muted — 인용을 본문과 구별하는 최소 수단이다. 왼쪽 괘선 같은
		// 시각 장식은 도형 외관이 없는 이번 iteration에서 열지 않는다 (Iteration 4).
		return textPatch{color: ptr(soml.TokenColor("muted"))}
	}
	return textPatch{}
}

// textVariant는 테마가 이름 붙인 외관 번들 하나다.
type textVariant struct {
	name  string
	patch textPatch
}

// textVariants는 minimal 테마의 텍스트 variant 표다 — 캐스케이드의 둘째 단(PRD §8.3
// components.json의 `variants` 자리, 파일 포맷은 Iteration 10).
//
// variant 어휘는 표준화하지 않는다는 결정(PRD §13-6)에 따라 이 이름들은 minimal 테마의
// 사유물이다. 값은 캐스케이드의 각 단이 실제로 구별되도록 골랐다 — 역할 기본값과 같은
// 값을 정하는 variant는 우선순위 테스트가 아무것도 증명하지 못한다.
//
// 맵이 아니라 순서 있는 슬라이스인 이유는 색 토큰과 같다: 모르는 variant 경고가 나열하는
// 목록이 결정적이어야 한다.
func textVariants() []textVariant {
	return []textVariant{
		// kicker: 제목 위에 얹는 짧은 안내 줄. 캡션 크기(18)에 강조색과 굵기를 더한다 —
		// body 기본값(24/400/fg)과 세 필드 모두 구별된다.
		{name: "kicker", patch: textPatch{size: ptr(18.0), weight: ptr(700), color: ptr(soml.TokenColor("accent1"))}},
		// lead: 본문보다 큰 도입 문단. size만 정한다(부분 패치) — 색·정렬·굵기는 아래 단이
		// 소유하고, 32는 어느 역할 기본값(18/20/24/36/56)과도 겹치지 않는다.
		{name: "lead", patch: textPatch{size: ptr(32.0)}},
	}
}

// textVariantPatch는 이름으로 variant를 찾는다.
func textVariantPatch(name string) (textPatch, bool) {
	for _, v := range textVariants() {
		if v.name == name {
			return v.patch, true
		}
	}
	return textPatch{}, false
}

// textVariantNames는 variant 이름 목록이다 — 경고 메시지가 허용 어휘를 나열하는 근거다
// (Theme.ColorNames와 같은 태도). 도형 어휘와 **다른 목록**이다: 경고는 요소 종류에 맞는
// 목록을 나열해야 한다 (resolver.warnUnknownVariant).
func textVariantNames() []string {
	variants := textVariants()
	names := make([]string, len(variants))
	for i, v := range variants {
		names[i] = v.name
	}
	return names
}

// textRoleDefaults는 텍스트 역할 5종의 기본 외관이다 (PRD §8.3 components.json의 `text`에
// 대응). 캐스케이드의 가장 아래 단이며, 인라인 속성이 이것을 덮어쓴다.
//
// 수치의 근거는 수직 리듬이다: 본문 24px × 줄높이 1.5 = 36px이고 스테이지 높이 720px은
// 정확히 본문 20줄이다 (두 종횡비가 높이 720을 공유하므로 4:3에서도 같다 — ADR 0002).
// 나머지 크기는 본문에서 약 1.5배씩 벌린 단계(18 / 24 / 36 / 56)이고, 큰 글자는 줄 사이가
// 상대적으로 넓어 보이므로 줄높이를 낮춘다. code는 같은 px에서 mono가 더 크게 보여 본문보다
// 한 단 낮춘 20px이다.
//
// 정렬 기본값은 전부 start다 — 중앙 정렬은 표지 같은 특정 레이아웃의 선택이지 텍스트의
// 성질이 아니고, 본문을 중앙 정렬로 두면 여러 줄이 될 때 읽는 눈이 줄 시작을 매번 찾는다.
func textRoleDefaults(role soml.TextRole) soml.TextStyle {
	switch role {
	case soml.RoleTitle:
		return soml.TextStyle{Role: role, Font: soml.FontHeading, Size: 56, Weight: 700, LineHeight: 1.2,
			Color: soml.TokenColor("fg"), Align: soml.AlignStart}
	case soml.RoleHeading:
		return soml.TextStyle{Role: role, Font: soml.FontHeading, Size: 36, Weight: 700, LineHeight: 1.3,
			Color: soml.TokenColor("fg"), Align: soml.AlignStart}
	case soml.RoleCaption:
		return soml.TextStyle{Role: role, Font: soml.FontBody, Size: 18, Weight: 400, LineHeight: 1.4,
			Color: soml.TokenColor("muted"), Align: soml.AlignStart}
	case soml.RoleCode:
		return soml.TextStyle{Role: role, Font: soml.FontMono, Size: 20, Weight: 400, LineHeight: 1.5,
			Color: soml.TokenColor("fg"), Align: soml.AlignStart}
	default: // soml.RoleBody
		return soml.TextStyle{Role: soml.RoleBody, Font: soml.FontBody, Size: 24, Weight: 400, LineHeight: 1.5,
			Color: soml.TokenColor("fg"), Align: soml.AlignStart}
	}
}

// ── 도형 컴포넌트 (PRD §8.3의 `shape`) ──────────────────────────────────────
//
// 아래 두 표가 도형 캐스케이드의 **아래 두 단**이다. 텍스트 쪽 표(textRoleDefaults ·
// textComponentDefault · textVariants)와 나란한 자리이고, 값을 고르는 기준도 같다:
// 단마다 값이 실제로 구별되어야 하고, 순서가 결정적이어야 한다.
//
// 두 표가 **테마를 인자로 받는** 이유는 텍스트 패치와 갈라지는 자리다: 도형의 확정 값은
// 토큰 이름과 **해석된 수치를 함께** 담는다 (soml.Radius·ShadowToken — PPTX가 반경을
// `prstGeom` 조정값으로, 그림자를 `outerShdw` 파라미터로 굽기 때문이다). 텍스트의 색·폰트는
// 이름만으로 끝나 표가 테마를 볼 필요가 없었다.

// shapeComponentDefault는 도형 컴포넌트의 기본 외관 패치다 (PRD §8.3의
// `"shape": { "default": … }`) — 도형 캐스케이드의 **가장 아래 단**이다.
//
// 텍스트와 갈라지는 지점이 여기다: 텍스트는 역할 기본값이 완전한 외관을 깔고 컴포넌트
// 기본값이 부분 패치였지만(textRoleDefaults ↔ textComponentDefault), **도형에는 역할이
// 없다**. 그래서 이 표가 textRoleDefaults의 자리를 대신하고 `<shape type="rect"/>` 하나만
// 써도 그려지는 외관이 나온다 — 값이 비어 남으면 그것을 채우는 판단이 렌더러로 내려가고,
// 두 타깃이 서로 다른 값을 채우는 순간(아래 radius) 웹과 PPTX가 갈라진다.
func shapeComponentDefault(theme soml.Theme) shapePatch {
	patch := shapePatch{
		// 카드 배경이 흰 스테이지에서 보이는 최소 조합이다 (minimalColors의 surface 근거).
		fill:   colorFill("surface"),
		border: solidBorder(1, "muted"),
	}
	// radius가 기본값에 있는 이유는 **두 타깃의 암묵 기본값이 다르다**는 것이다: CSS는
	// `border-radius` 없으면 0(각진 모서리)이고 OOXML `prstGeom/roundRect`는 조정값 기본이
	// 짧은 변의 16.67%다. 저자가 `type="rounded-rect"`만 썼을 때 웹은 각지고 PPTX는 둥근
	// 도형이 나오는 것이 그 갈라짐이며, 확정 값이 그 자리를 메운다.
	//
	// 이 기본값은 `rect`에서 항목 02의 규칙("radius는 rounded-rect 전용")과 부딪힌다. 푼
	// 방법은 **검사를 인라인 단에만** 걸고 테마 단의 radius는 다른 프리셋에서 조용히 버리는
	// 것이다 (themeShapePatch). 근거는 **저자가 고칠 수 있는가**다: 인라인 radius는 저자가
	// 지울 수 있으니 진단이 고칠 방법을 말하지만, 테마 기본값은 저자가 손댈 수 없어 진단을
	// 내면 `<shape type="rect"/>`가 무조건 에러가 된다. 기본값에서 radius를 빼는 길도
	// 있었지만 위의 갈라짐이 되살아난다 — 저자가 지울 수 없는 값과 지울 수 있는 값에 같은
	// 규칙을 걸 수 없다는 것이 이 자리의 결론이다.
	//
	// 이것이 "테마 기본값과 저자 값에 같은 규칙을 적용할 수 있는가"의 첫 사례다.
	// Iteration 10의 테마 포맷이 같은 질문을 다시 만난다: 테마 **파일**의 값은 저자가 아니라
	// 테마 제작자의 것이므로 진단의 수신자가 또 달라지고, 그때는 `theme lint`가 받는 편이
	// 자연스럽다 (PRD §8.5).
	if radius := radiusToken(theme, "md"); radius != nil {
		patch.radius = radius
	}
	return patch
}

// shapeVariant는 테마가 이름 붙인 도형 외관 번들 하나다.
type shapeVariant struct {
	name  string
	patch shapePatch
}

// shapeVariants는 minimal 테마의 도형 variant 표다 — 캐스케이드의 둘째 단.
//
// 이름은 PRD §13-6의 **권장 어휘**에서 골랐다 (`card`, `card-header`, `card-body`,
// `card-backdrop`, `stat-card`, `badge`). variant 어휘를 표준화하지 않는다는 결정은 그대로고
// (계약은 캐스케이드 메커니즘까지다), 공식 테마들이 같은 권장 어휘를 공유해 관례를 만드는
// 것이 PRD의 방식이다.
//
// **토큰 표(minimalColors 등)와 어휘를 늘리는 기준이 다르다**: 토큰은 "픽스처가 실제로
// 쓴다"가 유일한 근거였다 — 값이 두 타깃에서 검증되어야 하기 때문이다. variant는 **이미
// 검증된 토큰의 조합**일 뿐 새 값을 만들지 않으므로, 픽스처가 쓰지 않는 variant가 있어도
// 산출물 위험이 없다. 대신 권장 어휘를 한 벌로 갖춰 두면 다른 테마로 만든 덱이 이 테마에서
// 폴백 경고 없이 빌드된다 (PRD §13-6이 지키려는 것이 그 이식성이다).
//
// **도형 variant는 자식 텍스트의 외관을 담지 않는다.** PRD §8.3의 `card-header`는
// `font`/`size`/`weight`를 함께 담지만 여기서는 담지 않는다: 담으면 캐스케이드가 트리를 타고
// 내려가는 **상속**이 되고, 상속은 한 요소 안에서 끝나는 지금의 병합(resolveText ·
// resolveShape)과 다른 두 번째 메커니즘이다. 두 메커니즘이 생기는 순간 "우선순위는 병합
// 함수 한 곳에 있다"는 계약에 첫 예외가 난다. 도형 안 텍스트의 외관은 그 `<text>`의
// 역할·variant가 소유하고, 저자는 `<shape variant="card-header"><text variant="kicker">`로
// 두 채널을 각각 쓴다. **이 결정은 PRD §8.3과 어긋나므로** 항목 07이 PRD를 함께 고친다.
//
// 값은 컴포넌트 기본값(fill surface · border 1 solid muted · radius md)과 **구별되도록**
// 골랐다 — 같은 값을 정하는 variant는 우선순위 테스트가 아무것도 증명하지 못한다
// (textVariants와 같은 기준). 맵이 아니라 순서 있는 슬라이스인 이유도 같다: 모르는 variant
// 경고가 나열하는 목록이 결정적이어야 한다.
func shapeVariants(theme soml.Theme) []shapeVariant {
	return []shapeVariant{
		// card: 카드 한 장. 바닥을 흰색(bg)으로 바꿔 surface 스테이지·컨테이너 위에서 떠
		// 보이게 하고, 모서리를 한 단 크게 잡는다. 테두리는 정하지 않는다 — 기본값
		// `1 solid muted`가 카드 경계로 그대로 쓸 만하고, 같은 값을 다시 적으면 이 단이
		// 무엇을 정하는지가 흐려진다 (부분 패치).
		{name: "card", patch: shapePatch{fill: colorFill("bg"), radius: radiusToken(theme, "lg")}},
		// card-header: card-body 위에 붙는 띠. fill을 정하지 않는 부분 패치다 — 기본값
		// surface가 본문(bg)과 갈라져 띠로 읽힌다. 테두리를 지우는 이유는 본문 테두리와
		// 맞닿아 2px로 겹치는 선을 만들지 않는 것이다.
		//
		// PRD §8.3의 `card-header`가 담던 타이포그래피(font/size/weight)는 여기 없다 —
		// 위의 상속 결정이다. 헤더 글자를 키우려면 저자가 `<text variant="…">`를 쓴다.
		{name: "card-header", patch: shapePatch{border: ptr(soml.NoBorder()), radius: radiusToken(theme, "sm")}},
		// card-body: 카드의 내용 칸 — 바닥만 흰색으로 바꾼다 (테두리·모서리는 기본값).
		{name: "card-body", patch: shapePatch{fill: colorFill("bg")}},
		// card-backdrop: 카드 뒤에 깔리는 테두리 면. PRD §8.3의 값은
		// `border: 3 gradient(accent1 accent4 accent5)`인데 **단색으로 대체**했다 — 색 함수는
		// Iteration 8이고(colorValue가 그 진단을 소유한다), 그 항목이 이 자리를 그라디언트로
		// 되돌린다. 채움이 `none`인 것은 원래 값 그대로다: 뒤에 깔리는 면이므로 위 카드를
		// 가리면 안 된다.
		{name: "card-backdrop", patch: shapePatch{fill: ptr(soml.NoFill()),
			border: solidBorder(3, "accent1"), radius: radiusToken(theme, "lg")}},
		// stat-card: 수치를 얹는 카드 — 테두리 대신 **그림자**로 띄운다. 선과 그림자를 같이
		// 쓰면 경계가 두 번 그려져 카드가 무거워진다 (PRD §8.3의 stat-card도 shadow만 담는다).
		{name: "stat-card", patch: shapePatch{border: ptr(soml.NoBorder()),
			radius: radiusToken(theme, "lg"), shadow: shadowToken(theme, "soft")}},
		// badge: 짧은 라벨을 감싸는 알약. 채움은 기본값(surface)을 그대로 두고 강조색 선만
		// 얹는다 — 채움을 accent1로 하면 그 위의 글자색이 bg여야 하고, 그 요구는 도형
		// variant가 자식 타이포그래피를 담지 않기로 한 결정과 정면으로 부딪힌다 (저자가 색을
		// 손으로 맞춰야 하는 variant는 번들이라고 할 수 없다).
		{name: "badge", patch: shapePatch{border: solidBorder(1, "accent1"), radius: radiusToken(theme, "lg")}},
	}
}

// shapeVariantPatch는 이름으로 도형 variant를 찾는다 (textVariantPatch의 도형 판).
//
// 텍스트 표와 **다른 표**를 보는 것이 계약이다: 두 어휘가 같은 이름을 가질 수 있고, 요소
// 종류가 어느 표를 볼지 정한다. 표를 하나로 합치면 `<shape variant="kicker">`가 텍스트
// 외관 번들을 받아 아무 일도 일어나지 않거나, 도형 variant가 텍스트 필드를 정하는 상속이
// 몰래 생긴다.
func shapeVariantPatch(theme soml.Theme, name string) (shapePatch, bool) {
	for _, v := range shapeVariants(theme) {
		if v.name == name {
			return v.patch, true
		}
	}
	return shapePatch{}, false
}

// shapeVariantNames는 도형 variant 이름 목록이다 — 경고가 나열하는 어휘다.
func shapeVariantNames(theme soml.Theme) []string {
	variants := shapeVariants(theme)
	names := make([]string, len(variants))
	for i, v := range variants {
		names[i] = v.name
	}
	return names
}

// imageComponentDefault는 이미지 컴포넌트의 기본 외관 패치다 — 캐스케이드의 셋째 단이고
// **비어 있다.**
//
// 도형과 정면으로 갈라지는 자리다 (shapeComponentDefault는 채움·테두리·반경을 정한다).
// 근거는 "값이 비면 렌더러가 채우는 판단을 하게 된다"는 그 규칙이 그림에는 걸리지 않는다는
// 것이다: 채움 없는 도형은 **보이지 않으므로** 두 타깃이 각자 무언가를 채우게 되지만, 그림은
// 자기 픽셀을 갖고 있어 장식이 하나도 없어도 완전한 그림이다. 두 타깃의 암묵 기본값도 여기서는
// 갈라지지 않는다 — 웹 `<img>`와 PPTX `pic`은 둘 다 테두리·그림자 없이 그린다.
//
// 그래서 **액자는 저자가 말한다**: 기본으로 테두리를 두르면 스크린샷·도표처럼 자기 배경이 있는
// 그림에 이중 경계가 생기고, 저자가 그것을 지우는 표기(`border="none"`)를 모든 그림에 적어야
// 한다. 액자가 필요한 자리에는 variant가 있다 (imageVariants의 `framed`) — 그 번들이 `frame`
// 속성을 열지 않은 결정(항목 01)의 실물이다.
//
// 인자를 받지 않는 것도 그 결과다 (shapeComponentDefault는 반경 토큰을 찾느라 테마를 받는다).
// Iteration 10의 components.json이 이 자리에 값을 실을 수 있고, 그때 인자가 생긴다.
func imageComponentDefault() imagePatch {
	return imagePatch{}
}

// imageVariant는 테마가 이름 붙인 이미지 외관 번들 하나다.
type imageVariant struct {
	name  string
	patch imagePatch
}

// imageVariants는 minimal 테마의 이미지 variant 표다 — 캐스케이드의 둘째 단.
//
// 하나뿐인 이유는 도형 표와 어휘를 늘리는 기준이 같기 때문이다: variant는 이미 검증된 토큰의
// **조합**이라 산출물 위험이 없지만, 이름이 관례가 되므로 뜻이 분명한 것만 넣는다. 액자
// (`framed`)는 PRD가 `frame` 속성으로 적었던 그 뜻이고, 그것을 속성이 아니라 번들로 옮긴 것이
// Iteration 5의 결정이다 (항목 01) — 표에 이름이 하나라도 있어야 그 결정에 실물이 있다.
//
// 값은 **컴포넌트 기본값과 구별된다**: 기본값이 비어 있으므로 세 필드 전부가 델타다. 세 조각을
// 함께 정하는 것이 번들의 뜻이다 — 저자가 `border`만 쓰면 액자가 아니라 선 하나이고, 그 차이를
// 테마가 정하는 것이 `frame` 속성 대신 variant를 고른 이유다.
func imageVariants(theme soml.Theme) []imageVariant {
	patch := imagePatch{border: solidBorder(1, "muted")}
	if radius := radiusToken(theme, "md"); radius != nil {
		patch.radius = radius
	}
	if shadow := shadowToken(theme, "soft"); shadow != nil {
		patch.shadow = shadow
	}
	return []imageVariant{{name: "framed", patch: patch}}
}

// imageVariantPatch는 이름으로 이미지 variant를 찾는다 (shapeVariantPatch의 이미지 판).
//
// 표가 **셋**이 된 것이 계약이다 (텍스트·도형·이미지): 세 어휘가 같은 이름을 가질 수 있고,
// 요소 종류가 어느 표를 볼지 정한다. 표를 합치면 `<image variant="card">`가 도형 외관 번들을
// 받아 fill을 정하려 들고(이미지에 없는 채널이다) 그 값이 조용히 버려진다.
func imageVariantPatch(theme soml.Theme, name string) (imagePatch, bool) {
	for _, v := range imageVariants(theme) {
		if v.name == name {
			return v.patch, true
		}
	}
	return imagePatch{}, false
}

// imageVariantNames는 이미지 variant 이름 목록이다 — 경고가 나열하는 어휘다.
func imageVariantNames(theme soml.Theme) []string {
	variants := imageVariants(theme)
	names := make([]string, len(variants))
	for i, v := range variants {
		names[i] = v.name
	}
	return names
}

// 아래 넷은 테마 표가 값을 **한 줄로** 적기 위한 도구다 (ptr와 같은 자리). 토큰 이름을 받는
// 이유는 표의 색·반경·그림자가 전부 토큰 참조여야 하기 때문이다 — 테마가 리터럴을 쓰면
// PPTX에서 `schemeClr` 참조가 끊겨 그 도형만 테마 색 편집을 따라가지 못한다 (제품 핵심 계약).
func colorFill(token string) *soml.Fill { return ptr(soml.ColorFill(soml.TokenColor(token))) }

func solidBorder(width float64, token string) *soml.Border {
	return &soml.Border{Width: width, Style: soml.BorderSolid, Color: soml.TokenColor(token)}
}

// radiusToken·shadowToken은 표에 없는 이름이면 nil을 준다 — 그 단이 그 필드를 정하지 않은
// 것이 된다. 지금 테마는 두 표를 모두 갖고 있어 nil이 나올 수 없지만, Iteration 10의 사용자
// 테마에는 없을 수 있고 그때 빌드가 깨지는 대신 그 필드가 비는 편이 맞다 (모르는 variant를
// 경고로 두는 것과 같은 태도 — 테마 교체가 덱을 멈추지 않는다).
func radiusToken(theme soml.Theme, name string) *soml.Radius {
	token, ok := theme.Radius(name)
	if !ok {
		return nil
	}
	return ptr(soml.TokenRadius(name, token.Px))
}

func shadowToken(theme soml.Theme, name string) *soml.ShadowToken {
	token, ok := theme.Shadow(name)
	if !ok {
		return nil
	}
	return &token
}
