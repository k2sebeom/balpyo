package core

import "github.com/k2sebeom/balpyo/soml"

// minimalTheme는 하드코딩된 최소 테마다.
//
// Iteration 10의 테마 패키지 포맷(theme.json + tokens.json + components.json)이 이 함수를
// 해체한다. 순서를 이렇게 잡은 이유: 포맷을 먼저 정하면 무엇이 실제로 필요한지 모르는 상태로
// 스키마를 고정하게 되고, 스키마는 사용자 파일이 되는 순간 바꾸기 어려워진다. 하드코딩된
// 표는 두 렌더러가 실제로 무엇을 요구하는지 드러낸 다음에 파일로 나간다.
//
// 담는 것은 이번 iteration이 쓰는 만큼이다 — 색 4종, 폰트 3종, 텍스트 역할 5종. 도형이
// 필요한 토큰(surface, accent2~6, radius/shadow)은 Iteration 4에서 함께 들어온다.
func minimalTheme() soml.Theme {
	return soml.Theme{
		Name:   "minimal",
		Colors: minimalColors(),
		Fonts:  minimalFonts(),
		Spaces: minimalSpaces(),
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
func minimalColors() []soml.ColorToken {
	return []soml.ColorToken{
		{Name: "fg", RGB: "1A1A1A", Slot: "dk1", Ref: "tx1"},
		{Name: "bg", RGB: "FFFFFF", Slot: "lt1", Ref: "bg1"},
		{Name: "muted", RGB: "6B7280", Slot: "dk2", Ref: "tx2"},
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
// (Theme.ColorNames와 같은 태도).
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
