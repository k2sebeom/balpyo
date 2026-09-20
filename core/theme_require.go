package core

import (
	"fmt"
	"strings"

	"github.com/k2sebeom/balpyo/soml"
)

// 이 파일은 **테마 패키지의 완전성 계약**이다 (ADR 0052의 결정 3).
//
// 폴백을 지운 대가로 로더가 완전성을 요구한다. 엔진이 하드코딩 표를 갖고 있던 동안에는 테마의
// 누락이 조용히 그 표로 메워졌고, 그것을 지운 지금은 누락이 **영(0) 값**으로 렌더된다 —
// `size: 0`의 제목, 색이 없는 캡션, 해석되지 않는 `color="accent6"`. 그 실패가 나타나는 자리
// (슬라이드)는 원인(테마 파일)에서 멀다. 그래서 로드 시점에 멈춘다.
//
// 검사가 요구하는 것은 **어휘**이지 값이 아니다: 이름이 있어야 저자가 속성에 쓸 수 있고
// (`color="accent6"`·`gap="md"`), 그 이름이 무엇으로 보이는지는 테마의 자유다. 문서의
// "채워야 하는 이름" 표(docs/reference/theme.md)가 이 함수다 — 표와 코드가 갈라지지 않도록
// 필수 이름을 여기 한 곳에 적는다.

// 필수 토큰 이름. 색 팔레트 10종의 근거는 ADR 0050(accent 6종은 위계가 아니라 **구별**이다),
// 공간 6종·반경 3종은 저자가 쓰는 축약 어휘(`gap="md"`)이고, 그림자 `soft` 하나는 v1이 그림자를
// 한 종만 여는 결정이다 (도형 variant의 `shadow`가 이 이름을 가리킨다).
var (
	requiredColors = []string{"fg", "bg", "muted", "surface",
		"accent1", "accent2", "accent3", "accent4", "accent5", "accent6"}
	requiredFonts   = []soml.FontRole{soml.FontHeading, soml.FontBody, soml.FontMono}
	requiredSpaces  = []string{"xs", "sm", "md", "lg", "xl", "2xl"}
	requiredRadii   = []string{"sm", "md", "lg"}
	requiredShadows = []string{"soft"}
)

// requireTokens는 토큰 표가 필수 어휘를 전부 갖췄는지 본다. 이름 하나를 메시지에 담고 멈추는
// 이유는 테마 제작자가 파일 하나를 고치는 흐름이라는 것이다 — 전부 모으면 첫 오류를 읽기 전에
// 목록을 읽어야 한다.
func requireTokens(theme soml.Theme) error {
	for _, name := range requiredColors {
		if _, ok := theme.Color(name); !ok {
			return fmt.Errorf("tokens.json에 색 토큰 %q가 없다 (필수: %s)",
				name, strings.Join(requiredColors, ", "))
		}
	}
	for _, name := range requiredFonts {
		if _, ok := theme.Font(name); !ok {
			return fmt.Errorf("tokens.json에 폰트 토큰 %q가 없다 (필수: %s)",
				name, fontRoleList(requiredFonts))
		}
	}
	for _, name := range requiredSpaces {
		if _, ok := theme.Space(name); !ok {
			return fmt.Errorf("tokens.json에 공간 토큰 %q가 없다 (필수: %s)",
				name, strings.Join(requiredSpaces, ", "))
		}
	}
	for _, name := range requiredRadii {
		if _, ok := theme.Radius(name); !ok {
			return fmt.Errorf("tokens.json에 반경 토큰 %q가 없다 (필수: %s)",
				name, strings.Join(requiredRadii, ", "))
		}
	}
	for _, name := range requiredShadows {
		if _, ok := theme.Shadow(name); !ok {
			return fmt.Errorf("tokens.json에 그림자 토큰 %q가 없다 (필수: %s)",
				name, strings.Join(requiredShadows, ", "))
		}
	}
	return nil
}

// requireComponents는 컴포넌트 표가 **역할 5종을 전부** 정하고, 각 역할이 이 테마의 토큰을
// 가리키는지 본다.
//
// 역할은 캐스케이드의 가장 아래 단이라서(PRD §8.3) 완전한 외관을 깔 책임을 진다 — 위 세 단은
// 부분 패치다. 그래서 크기·줄높이가 0이면 그 역할을 쓴 모든 블록이 보이지 않고, 폰트·색이
// 이 테마에 없는 이름이면 확정 외관이 해석할 수 없는 참조를 담은 채 렌더러에 도달한다
// (웹은 CSS 변수가 비고, PPTX는 schemeClr 참조가 끊긴다).
func requireComponents(theme soml.Theme, comps components) error {
	for _, role := range soml.TextRoles() {
		style, ok := comps.textRoles[role]
		if !ok {
			return fmt.Errorf("components.json의 text.roles에 역할 %q가 없다 (필수: %s)",
				role, textRoleList(soml.TextRoles()))
		}
		if style.Size <= 0 {
			return fmt.Errorf("역할 %q의 size가 %g다 — 역할은 캐스케이드의 바닥이라 완전한 값이어야 한다",
				role, style.Size)
		}
		if style.LineHeight <= 0 {
			return fmt.Errorf("역할 %q의 lineHeight가 %g다 — 역할은 캐스케이드의 바닥이라 완전한 값이어야 한다",
				role, style.LineHeight)
		}
		if _, ok := theme.Font(style.Font); !ok {
			return fmt.Errorf("역할 %q의 font %q가 이 테마의 폰트 토큰이 아니다 (이 테마: %s)",
				role, style.Font, fontRoleList(themeFontNames(theme)))
		}
		// 역할의 색은 **토큰 참조**만 뜻이 있다 (리터럴은 PPTX의 테마 색 편집을 끊는다 —
		// tokens.json의 shadow color와 같은 계약).
		if !style.Color.IsToken() {
			return fmt.Errorf("역할 %q의 color가 색 토큰 이름이 아니다 (이 테마: %s)",
				role, strings.Join(theme.ColorNames(), ", "))
		}
		if _, ok := theme.Color(style.Color.Token); !ok {
			return fmt.Errorf("역할 %q의 color %q가 이 테마의 색 토큰이 아니다 (이 테마: %s)",
				role, style.Color.Token, strings.Join(theme.ColorNames(), ", "))
		}
		if style.Align == "" {
			return fmt.Errorf("역할 %q의 align이 없다 (쓸 수 있는 align: %s)",
				role, textAlignList(soml.TextAligns()))
		}
	}
	return nil
}

// themeFontNames는 이 테마가 정한 폰트 이름들이다 (진단 문구용, 파일 순서).
func themeFontNames(theme soml.Theme) []soml.FontRole {
	names := make([]soml.FontRole, 0, len(theme.Fonts))
	for _, f := range theme.Fonts {
		names = append(names, f.Name)
	}
	return names
}

func fontRoleList(roles []soml.FontRole) string {
	names := make([]string, 0, len(roles))
	for _, r := range roles {
		names = append(names, string(r))
	}
	return strings.Join(names, ", ")
}

func textRoleList(roles []soml.TextRole) string {
	names := make([]string, 0, len(roles))
	for _, r := range roles {
		names = append(names, string(r))
	}
	return strings.Join(names, ", ")
}

func textAlignList(aligns []soml.TextAlign) string {
	names := make([]string, 0, len(aligns))
	for _, a := range aligns {
		names = append(names, string(a))
	}
	return strings.Join(names, ", ")
}
