package core

import (
	"fmt"
	"sort"
	"strings"

	"github.com/k2sebeom/balpyo/soml"
)

// 이 파일은 **테마 패키지의 완전성 계약**이다.
//
// 폴백을 지운 대가로 로더가 완전성을 요구한다. 엔진이 하드코딩 표를 갖고 있던 동안에는 테마의
// 누락이 조용히 그 표로 메워졌고, 그것을 지운 지금은 누락이 **영(0) 값**으로 렌더된다 —
// `size: 0`의 제목, 색이 없는 캡션, 해석되지 않는 `color="accent6"`. 그 실패가 나타나는 자리
// (슬라이드)는 원인(테마 파일)에서 멀다. 그래서 로드 시점에 멈춘다.
//
// 검사가 요구하는 것은 **어휘**이지 값이 아니다: 이름이 있어야 저자가 속성에 쓸 수 있고
// (`color="accent6"`·`gap="md"`), 그 이름이 무엇으로 보이는지는 테마의 자유다. 문서의
// "채워야 하는 이름" 표가 이 함수다 — 표와 코드가 갈라지지 않도록
// 필수 이름을 여기 한 곳에 적는다.

// 필수 토큰 이름. 색 팔레트 10종의 근거는 구별 가능성이다(accent 6종은 위계가 아니라 **구별**이다),
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
			return fmt.Errorf("tokens.json has no color token %q (required: %s)",
				name, strings.Join(requiredColors, ", "))
		}
	}
	for _, name := range requiredFonts {
		if _, ok := theme.Font(name); !ok {
			return fmt.Errorf("tokens.json has no font token %q (required: %s)",
				name, fontRoleList(requiredFonts))
		}
	}
	for _, name := range requiredSpaces {
		if _, ok := theme.Space(name); !ok {
			return fmt.Errorf("tokens.json has no space token %q (required: %s)",
				name, strings.Join(requiredSpaces, ", "))
		}
	}
	for _, name := range requiredRadii {
		if _, ok := theme.Radius(name); !ok {
			return fmt.Errorf("tokens.json has no radius token %q (required: %s)",
				name, strings.Join(requiredRadii, ", "))
		}
	}
	for _, name := range requiredShadows {
		if _, ok := theme.Shadow(name); !ok {
			return fmt.Errorf("tokens.json has no shadow token %q (required: %s)",
				name, strings.Join(requiredShadows, ", "))
		}
	}
	return nil
}

// requireColorSlots는 색 토큰 표가 clrScheme에 **실릴 수 있는 모양**인지 본다.
//
// 완전성 검사 옆에 있는 이유: 이것은 PPTX 조립에서도 드러나는 계약이고
// (render/pptx.clrSchemeXML이 같은 판정을 한다 — 렌더러는 자기 계약을 스스로 지킨다), 그
// 자리에서 멈추면 진단이 "pptx 렌더 실패"를 가리켜 고칠 파일(tokens.json)에서 멀어진다.
// 테마의 문제는 슬라이드를 렌더하기 전에 멈춘다.
//
// 그래서 색 토큰 표에는 **상한**이 있다: 슬롯은 열둘이고 겹칠 수 없으므로 새 색 이름을 더하려면
// 남은 자리가 있어야 한다 — 부모를 물려받은 테마가 색을 더하는 대신 갈아 끼우는 이유다.
// 반경·공간·그림자·폰트에는 이 상한이 없다.
func requireColorSlots(theme soml.Theme) error {
	// 자리 임자를 **먼저** 다 모은다: 빈 슬롯 진단은 "이 토큰 앞까지"가 아니라 이 테마의 남은
	// 자리를 나열해야 고칠 수 있다.
	owner := make(map[string]string, len(theme.Colors))
	for _, token := range theme.Colors {
		if token.Slot == "" || !knownColorSlot(token.Slot) {
			continue
		}
		if first, dup := owner[token.Slot]; dup {
			return fmt.Errorf("tokens.json clrScheme slot %q is shared by color tokens %q and %q — cannot tell which color owns the slot",
				token.Slot, first, token.Name)
		}
		owner[token.Slot] = token.Name
	}
	for _, token := range theme.Colors {
		if token.Slot == "" {
			return fmt.Errorf("tokens.json color token %q has no clrScheme slot — every color token must follow PPTX theme color edits (%s)",
				token.Name, freeColorSlots(owner))
		}
		if !knownColorSlot(token.Slot) {
			return fmt.Errorf("tokens.json color token %q has an unknown clrScheme slot %q (available slots: %s)",
				token.Name, token.Slot, strings.Join(soml.ColorSlots(), ", "))
		}
	}
	return nil
}

// knownColorSlot은 이름이 clrScheme의 열두 자리 중 하나인지 본다.
func knownColorSlot(slot string) bool {
	for _, known := range soml.ColorSlots() {
		if slot == known {
			return true
		}
	}
	return false
}

// freeColorSlots는 빈 슬롯 진단의 뒷말이다 — 남은 자리를 나열하고, 없으면 고치는 방법을 적는다
// (내장 테마 셋은 열을 쓰므로 남는 자리가 링크 둘뿐이다).
func freeColorSlots(owner map[string]string) string {
	slots := soml.ColorSlots()
	free := make([]string, 0, len(slots))
	for _, slot := range slots {
		if _, taken := owner[slot]; !taken {
			free = append(free, slot)
		}
	}
	if len(free) == 0 {
		return "no free slots — replace the value of an existing name instead of adding a new color name"
	}
	return "free slots: " + strings.Join(free, ", ")
}

// requireComponents는 컴포넌트 표가 **역할 5종을 전부** 정하고, 각 역할이 이 테마의 토큰을
// 가리키는지 본다.
//
// 역할은 캐스케이드의 가장 아래 단이라서 완전한 외관을 깔 책임을 진다 — 위 세 단은
// 부분 패치다. 그래서 크기·줄높이가 0이면 그 역할을 쓴 모든 블록이 보이지 않고, 폰트·색이
// 이 테마에 없는 이름이면 확정 외관이 해석할 수 없는 참조를 담은 채 렌더러에 도달한다
// (웹은 CSS 변수가 비고, PPTX는 schemeClr 참조가 끊긴다).
func requireComponents(theme soml.Theme, comps components) error {
	for _, role := range soml.TextRoles() {
		style, ok := comps.textRoles[role]
		if !ok {
			return fmt.Errorf("components.json text.roles has no role %q (required: %s)",
				role, textRoleList(soml.TextRoles()))
		}
		if style.Size <= 0 {
			return fmt.Errorf("role %q has size %g — roles are the bottom of the cascade and must be complete",
				role, style.Size)
		}
		if style.LineHeight <= 0 {
			return fmt.Errorf("role %q has lineHeight %g — roles are the bottom of the cascade and must be complete",
				role, style.LineHeight)
		}
		if _, ok := theme.Font(style.Font); !ok {
			return fmt.Errorf("role %q font %q is not a font token of this theme (this theme: %s)",
				role, style.Font, fontRoleList(themeFontNames(theme)))
		}
		// 역할의 색은 **토큰 참조**만 뜻이 있다 (리터럴은 PPTX의 테마 색 편집을 끊는다 —
		// tokens.json의 shadow color와 같은 계약).
		if !style.Color.IsToken() {
			return fmt.Errorf("role %q color is not a color token name (this theme: %s)",
				role, strings.Join(theme.ColorNames(), ", "))
		}
		if _, ok := theme.Color(style.Color.Token); !ok {
			return fmt.Errorf("role %q color %q is not a color token of this theme (this theme: %s)",
				role, style.Color.Token, strings.Join(theme.ColorNames(), ", "))
		}
		if style.Align == "" {
			return fmt.Errorf("role %q has no align (available aligns: %s)",
				role, textAlignList(soml.TextAligns()))
		}
	}
	return nil
}

// requireComponentColors는 컴포넌트 표의 **모든 색 참조**(역할 밖 — 컴포넌트 기본값·variant의
// color·fill·border, 이미지 테두리, 표 셀)가 이 테마의 색 토큰 이름인지 본다.
//
// 테마 데이터는 색을 토큰 이름으로만 적는다 (리터럴은 PPTX의 테마 색 편집을 끊는다 — colorFill의
// 근거). 그 이름을 여기서 보지 않으면 `gradient(…)`·`#FF9900`·오타가 토큰 이름인 채로 확정
// 외관에 실려, 그 variant를 인라인 덮기 없이 쓴 슬라이드가 나올 때 **렌더러**에서야 멈춘다 —
// `check`는 렌더하지 않으므로 통과하고 `build`만 실패한다. 역할의 색(requireComponents)과 같은
// 자리에서 같은 태도로 멈춰, 진단이 고칠 파일(components.json)과 자리를 가리킨다.
func requireComponentColors(theme soml.Theme, comps components) error {
	// 토큰 참조는 셋이다: 토큰 이름, 변형된 토큰(`shade(accent1, 20%)`), 스톱이 전부 토큰인 그라디언트.
	// 셋 다 PPTX에서 `schemeClr` 참조로 남아 테마 색 편집을 따라간다. 리터럴(`#FF9900`, 리터럴 스톱)은 아니다.
	check := func(where string, color *soml.Color) error {
		if color == nil {
			return nil
		}
		if bad, ok := notTokenRef(theme, *color); ok {
			return fmt.Errorf("components.json %s %q is not a color token reference of this theme — themes use only color token names (including modifiers and token gradients) (this theme: %s)",
				where, bad, strings.Join(theme.ColorNames(), ", "))
		}
		return nil
	}
	fill := func(where string, f *soml.Fill) error {
		if f == nil || f.None {
			return nil
		}
		return check(where+" fill", &f.Color)
	}
	border := func(where string, b *soml.Border) error {
		if b == nil || b.None {
			return nil
		}
		return check(where+" border", &b.Color)
	}

	tags := make([]string, 0, len(comps.textComponentDefaults))
	for tag := range comps.textComponentDefaults {
		tags = append(tags, tag)
	}
	sort.Strings(tags)
	var errs []error
	for _, tag := range tags {
		errs = append(errs, check(fmt.Sprintf("text.componentDefaults %q color", tag), comps.textComponentDefaults[tag].color))
	}
	slots := make([]string, 0, len(comps.textSlotDefaults))
	for slot := range comps.textSlotDefaults {
		slots = append(slots, slot)
	}
	sort.Strings(slots)
	for _, slot := range slots {
		errs = append(errs, check(fmt.Sprintf("text.componentDefaults slot %q color", slot), comps.textSlotDefaults[slot].color))
	}
	for _, v := range comps.textVariants {
		errs = append(errs, check(fmt.Sprintf("text variant %q color", v.name), v.patch.color))
	}
	errs = append(errs, check("text.mark highlight", comps.markHighlight))
	if l := comps.connectorDefault.line; l != nil {
		errs = append(errs, check("connector.default line", &l.Color))
	}
	for _, v := range comps.connectorVariants {
		if l := v.patch.line; l != nil {
			errs = append(errs, check(fmt.Sprintf("connector variant %q line", v.name), &l.Color))
		}
	}
	errs = append(errs, fill("shape.default", comps.shapeDefault.fill),
		border("shape.default", comps.shapeDefault.border))
	for _, v := range comps.shapeVariants {
		where := fmt.Sprintf("shape variant %q", v.name)
		errs = append(errs, fill(where, v.patch.fill), border(where, v.patch.border))
	}
	errs = append(errs, border("image.default", comps.imageDefault.border))
	for _, v := range comps.imageVariants {
		errs = append(errs, border(fmt.Sprintf("image variant %q", v.name), v.patch.border))
	}
	for _, c := range []struct {
		where string
		patch cellPatch
	}{{"table.cell", comps.table.cell}, {"table.header", comps.table.header}} {
		errs = append(errs, fill(c.where, c.patch.fill), border(c.where, c.patch.border),
			check(c.where+" color", c.patch.color))
	}
	errs = append(errs, fill("table.stripe", comps.table.stripe))
	// 첫 에러 하나로 멈춘다 — requireTokens와 같은 이유 (파일 하나를 고치는 흐름).
	for _, err := range errs {
		if err != nil {
			return err
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
