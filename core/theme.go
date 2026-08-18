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
