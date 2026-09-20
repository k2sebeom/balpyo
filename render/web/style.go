package web

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/k2sebeom/balpyo/soml"
)

// 토큰 → CSS 커스텀 프로퍼티 이름 규칙이다 (PRD §8.4의 `--balpyo-*`).
//
// 접두를 붙이는 이유는 이름 충돌이 아니라 **소유권 표시**다: 저자 CSS가 `--balpyo-`로
// 시작하는 변수를 보면 그것이 엔진이 만든 값이고 테마를 바꾸면 따라 움직인다는 것을 알 수
// 있다. 이름의 두 번째 칸(`color`/`font`)은 토큰 종류이므로, 서로 다른 종류에 같은 이름의
// 토큰이 있어도(색 `mono`와 폰트 `mono`) 변수가 겹치지 않는다.
const (
	colorVarPrefix = "--balpyo-color-"
	fontVarPrefix  = "--balpyo-font-"
)

// themeCSS는 덱에 실린 토큰 표를 CSS로 옮긴다 — 색은 커스텀 프로퍼티, 폰트는 패밀리 스택,
// 공간은 커스텀 프로퍼티다.
//
// 이 함수가 **표를 해석하지 않는다**는 점이 계약이다: 값의 의미(대비, 폴백 순서, 리듬)는
// 테마가 정하고 여기서는 옮기기만 한다. PPTX 렌더러의 테마 파트도 같은 표에서 나오므로 두
// 타깃의 색·폰트가 구조적으로 같아진다 (soml.Theme의 주석). 공간만 예외다 — OOXML 테마
// 파트에 공간 개념이 없어 **이 변수는 웹에만 산다** (soml.SpaceToken의 비대칭).
//
// spaces는 덱이 공간 토큰을 참조하는지다 (레이아웃 태그가 있는 덱 — web.features). 참조가
// 없을 때 선언을 내지 않는 이유는 골든이다: 레이아웃을 쓰지 않은 덱의 산출물이 이
// iteration 때문에 달라지면 diff가 무엇이 변했는지 말해 주지 못한다. 그림자 토큰도 같은
// 이유로 **참조하는 어휘**(도형·그림)가 있는 덱에만 나간다 (feat.shadowTokens).
//
// 테마 층(`@layer balpyo.theme`)에 넣는 이유: 저자 CSS가 같은 변수를 덮어쓸 수 있어야 하고
// (author 층이 뒤), 엔진 기본 CSS는 이 변수를 참조하되 이기지 못해야 한다 (base 층이 앞).
func themeCSS(theme soml.Theme, feat features) string {
	if len(theme.Colors) == 0 && len(theme.Fonts) == 0 {
		// 해석을 거치지 않은 덱(테마 없음)에는 층을 만들지 않는다. 빈 :root 블록은 골든에
		// 잡음만 남기고, 변수가 없다는 사실은 참조하는 쪽에서 어차피 드러난다.
		return ""
	}

	var b strings.Builder
	b.WriteString("\n@layer balpyo.theme {\n")
	fmt.Fprintf(&b, "  /* 테마 %q의 토큰 표 — 웹 변수와 PPTX 테마 파트가 이 한 표에서 나온다. */\n", theme.Name)
	b.WriteString("  :root {\n")
	// 선언 순서는 표의 순서다 (soml.Theme가 맵이 아닌 이유) — 골든이 실행마다 같아야 한다.
	for _, token := range theme.Colors {
		fmt.Fprintf(&b, "    %s%s: #%s;\n", colorVarPrefix, token.Name, token.RGB)
	}
	for _, token := range theme.Fonts {
		fmt.Fprintf(&b, "    %s%s: %s;\n", fontVarPrefix, token.Name, fontStack(token))
	}
	if feat.spaceTokens() {
		// 참조(gap/pad의 var())와 선언이 **한 표**에서 나온다 — 배치조차 테마의 리듬을
		// 따라간다는 계약(PRD §8.4)의 웹 쪽 절반이다.
		b.WriteString(spaceVarsCSS(theme))
	}
	if feat.shadowTokens() {
		// 그림자도 같은 계약이다 (shadowVarsCSS) — 요소는 var()로 가리키고 표가 선언한다.
		b.WriteString(shadowVarsCSS(theme))
	}
	b.WriteString("  }\n")
	if _, ok := theme.Color(bgToken); ok {
		// 슬라이드 바탕도 토큰에서 나온다. 렌더러가 배경색을 따로 갖고 있으면 테마를 바꿨을 때
		// 콘텐츠만 새 색이 되고 바탕은 옛 색으로 남는다 — 그 어긋남은 렌더를 보기 전까지
		// 드러나지 않는다.
		b.WriteString("\n")
		fmt.Fprintf(&b, "  .balpyo-slide { background: var(%s%s); }\n", colorVarPrefix, bgToken)
	}
	b.WriteString(layoutStagesCSS(theme, feat))
	b.WriteString("}\n")
	return b.String()
}

// layoutStagesCSS는 덱이 쓴 레이아웃의 배치 규칙을 테마 층에 낸다 (M1.1). 스테이지에 얹으므로
// 기본 스택 규칙(`.balpyo-stage`, base 층)을 이긴다 — 테마 층이 base 층 뒤라 특정도와 무관하게
// 이긴다 (PRD §12.2의 cascade layer). 배치 원문은 core가 이미 정규화·검증했다 (soml.Theme.Layouts).
//
// feat.layouts/slotLayouts만 도는 이유는 골든 불변이다 — 쓰지 않은 레이아웃의 규칙이 산출물에
// 들어가면 diff가 무엇이 변했는지 말해 주지 못한다 (themeCSS의 다른 조건과 같은 근거).
//
// # 흐름 규칙과 슬롯 규칙은 서로 배타적이다 (ADR 0035)
//
// 한 레이아웃은 슬라이드마다 흐름 모드이거나 슬롯 모드다 (writeStage). 그래서 흐름 규칙은
// `:not(.balpyo-slots)`로, 슬롯 영역 규칙은 `.balpyo-slots`로 겨눠 **한 레이아웃을 두 슬라이드가
// 각각 흐름·슬롯으로 써도** 규칙이 서로 새지 않는다. `:not`이 없으면 슬롯 슬라이드에도 흐름
// 규칙이 얹혀(두 클래스가 다 있으므로) 흐름의 정렬·트랙이 슬롯 영역과 뒤섞인다 — 그 갈라짐은
// 픽셀 비교에서만 드러난다. 그래서 슬롯 영역 CSS는 `display:grid`부터 스스로 갖춘다 (자족적).
func layoutStagesCSS(theme soml.Theme, feat features) string {
	if len(feat.layouts) == 0 && len(feat.slotLayouts) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("\n")
	if len(feat.slotLayouts) > 0 {
		// 슬롯 영역 자식의 텍스트·리스트는 확정 폭(width:100%, contentCSS)에서 벗어나 auto로
		// 돌아간다 — **세 번째 리셋 문맥이다** (ADR 0043). 앞의 둘은 레이아웃 컨테이너의
		// 자식(layoutCSS)과 stack·캔버스의 자식(stackCSS)이고, 근거 문장이 셋 다 같다: 폭이
		// 칸을 다 먹으면 테마가 쓴 정렬(`justify-items`·`place-self`)이 자식을 움직일 여지가
		// 없다. ADR 0035가 "배치의 실체는 테마가 소유한다"고 정했는데, 이 리셋이 없으면
		// 테마가 쓴 선언이 엔진 base 층에 조용히 지워진다 (`title.slots.css`의
		// `justify-items: center`가 그랬다).
		//
		// 슬롯 영역 CSS가 있는 덱에만 나간다 — 흐름 모드 덱과 슬롯을 쓰지 않은 덱의 산출물은
		// 바이트가 같다 (이 함수의 다른 조건과 같은 근거). `justify-items`의 기본값이
		// stretch이므로 정렬을 쓰지 않은 슬롯 영역에서는 자식이 그대로 영역 폭까지 늘어난다.
		b.WriteString("  .balpyo-stage.balpyo-slots > :is(.balpyo-text, .balpyo-list) { width: auto; }\n")
	}
	for _, name := range feat.layouts {
		// feat.layouts는 배치 CSS가 있는 레이아웃만 담으므로 조회는 언제나 성공한다.
		fmt.Fprintf(&b, "  .balpyo-stage.balpyo-layout-%s:not(.balpyo-slots) { %s; }\n", name, theme.Layouts[name])
	}
	for _, name := range feat.slotLayouts {
		// feat.slotLayouts는 슬롯 영역 CSS가 있는 레이아웃만 담으므로 조회는 언제나 성공한다.
		fmt.Fprintf(&b, "  .balpyo-stage.balpyo-layout-%s.balpyo-slots { %s; }\n", name, theme.SlotLayouts[name])
	}
	return b.String()
}

// bgToken은 슬라이드 바탕 색 토큰의 이름이다 (PRD §8.4의 표준 어휘).
const bgToken = "bg"

// fontStack은 폰트 토큰의 CSS font-family 값이다.
//
// 인용 규칙: 식별자가 아닌 이름(공백이 든 `Noto Sans` 등)만 인용한다. CSS 제네릭 패밀리
// (`sans-serif`)는 인용하면 그 이름의 폰트를 찾는 뜻이 되어 **마지막 폴백이 사라진다** —
// 폰트가 없는 환경에서만 드러나는 종류의 실패다.
func fontStack(token soml.FontToken) string {
	names := token.Family()
	quoted := make([]string, len(names))
	for i, name := range names {
		quoted[i] = quoteFamily(name)
	}
	return strings.Join(quoted, ", ")
}

// quoteFamily는 필요할 때만 패밀리 이름을 인용한다. CSS 식별자 문법(`[A-Za-z_-]`로 시작,
// 이어서 영숫자·`_`·`-`)에 맞지 않으면 인용한다.
func quoteFamily(name string) string {
	ident := name != ""
	for i := 0; i < len(name); i++ {
		c := name[i]
		switch {
		case c >= 'A' && c <= 'Z', c >= 'a' && c <= 'z', c == '_', c == '-':
		case c >= '0' && c <= '9' && i > 0:
		default:
			ident = false
		}
		if !ident {
			break
		}
	}
	if ident {
		return name
	}
	return `"` + name + `"`
}

// textStyleCSS는 **해석이 끝난** 텍스트 외관을 CSS 선언 목록으로 컴파일한다 (PRD §9.4의
// `font`/`size`/`weight`/`color`/`align` 행).
//
// 입력은 soml.TextStyle 하나뿐이다 — 요소의 Attrs를 다시 읽지 않는다. 캐스케이드는 core가
// 이미 한 번 적용했고, 렌더러가 다시 해석하면 웹과 PPTX가 서로 다른 규칙으로 같은 속성을
// 읽게 된다 (soml.Appearance의 주석).
//
// 값은 전부 명시한다 — 상속으로 채워지는 값이 하나라도 있으면 기본 CSS를 고칠 때 텍스트
// 메트릭이 조용히 움직이고, 지오메트리 패스가 잰 박스가 PPTX 박스이므로 그 움직임이
// 산출물까지 간다.
func textStyleCSS(style soml.TextStyle, theme soml.Theme) (string, error) {
	font, err := fontVar(style.Font, theme)
	if err != nil {
		return "", err
	}
	align, err := alignValue(style.Align)
	if err != nil {
		return "", err
	}
	// 텍스트 색은 단색이면 `color`, 그라디언트면 `background-clip: text`다 (PRD §9.4 —
	// 텍스트 `color=gradient`는 웹에서 background-clip). 그라디언트는 단일 색 값이 아니라
	// colorValue를 지나지 못하므로 여기서 갈린다.
	var colorDecl string
	if style.Color.IsGradient() {
		grad, err := gradientCSS(*style.Color.Gradient, theme)
		if err != nil {
			return "", err
		}
		// 글자 실루엣으로 그라디언트를 자른다: 배경 그라디언트 → 글자 모양 클립 → 채움 투명.
		// -webkit- 접두를 함께 내는 이유는 background-clip:text가 표준화 전부터 접두로만
		// 동작하는 엔진이 있기 때문이다 (측정 브라우저·뷰어 모두 커버).
		colorDecl = fmt.Sprintf("background-image: %s; -webkit-background-clip: text; background-clip: text; color: transparent", grad)
	} else {
		color, err := colorValue(style.Color, theme)
		if err != nil {
			return "", err
		}
		colorDecl = "color: " + color
	}
	if style.Size <= 0 {
		return "", fmt.Errorf("폰트 크기가 해석되지 않았다 (%s)", formatNumber(style.Size))
	}
	if style.Weight <= 0 {
		return "", fmt.Errorf("폰트 굵기가 해석되지 않았다 (%d)", style.Weight)
	}
	if style.LineHeight <= 0 {
		return "", fmt.Errorf("줄높이가 해석되지 않았다 (%s)", formatNumber(style.LineHeight))
	}

	var b strings.Builder
	fmt.Fprintf(&b, "font-family: %s; ", font)
	// 크기는 논리 스테이지 px 그대로다 (PRD §13-5). 저자·웹·지오메트리 패스가 한 좌표계를
	// 쓰므로 환산이 없고, pt 환산은 PPTX 렌더러의 경계에서만 일어난다.
	fmt.Fprintf(&b, "font-size: %spx; ", formatNumber(style.Size))
	fmt.Fprintf(&b, "font-weight: %d; ", style.Weight)
	// 줄높이는 무단위 배수다 — CSS line-height는 배수를 그대로 받고, PPTX는 절대 줄간격
	// (lnSpc@spcPts)으로 환산해 낸다 (render/pptx.LineSpacing). spcPct(배수)는 엔진마다
	// 기준이 달라 같은 150%가 글줄을 밀었다 — docs/decisions/0005-line-spacing-and-slack.md.
	fmt.Fprintf(&b, "line-height: %s; ", formatNumber(style.LineHeight))
	fmt.Fprintf(&b, "%s; ", colorDecl)
	fmt.Fprintf(&b, "text-align: %s", align)
	return b.String(), nil
}

// runStyleCSS는 인라인 런(strong/em/code/a/span)의 확정 스타일을 CSS 선언 목록으로
// 컴파일한다 — textStyleCSS에 `font-style`(Italic)과 `text-decoration`(Underline)을 더한
// **완전 명시**다.
//
// 항상 낸다 (italic이 아니어도 `normal`, 밑줄이 없어도 `none`): UA 시트의
// `em { font-style: italic }`·`a { text-decoration: underline }` 같은 규칙이 개입할 자리를
// 남기면, 확정 스타일이 아니라 브라우저 기본값이 런의 모양을 정하게 된다 — textStyleCSS의
// "상속으로 채워지는 값이 하나라도 있으면" 계약과 같은 근거다.
//
// 블록(textStyleCSS)이 이 두 선언을 내지 않는 이유: 서브셋 블록 태그(div/h1~h3/p/
// blockquote/pre/li)에는 이탤릭·밑줄 UA 규칙이 없어 메트릭이 움직일 경로가 없고, 지금의
// 어휘에서 블록 스타일은 Italic/Underline을 가질 수 없다 — 선언을 더하면 Iteration 1
// 골든만 이유 없이 움직인다. 블록이 Italic을 가질 수 있게 되는 날(variant가 필드를 얻는
// 날) textStyleCSS가 이 두 선언을 함께 얻어야 한다.
func runStyleCSS(style soml.TextStyle, theme soml.Theme) (string, error) {
	decls, err := textStyleCSS(style, theme)
	if err != nil {
		return "", err
	}
	var b strings.Builder
	b.WriteString(decls)
	fontStyle := "normal"
	if style.Italic {
		fontStyle = "italic"
	}
	decoration := "none"
	if style.Underline {
		decoration = "underline"
	}
	fmt.Fprintf(&b, "; font-style: %s; text-decoration: %s", fontStyle, decoration)
	return b.String(), nil
}

// fontVar는 폰트 토큰을 CSS 변수 참조로 만든다. 패밀리 스택을 여기서 펼치지 않는 이유는
// 색과 같다 — 테마를 바꿨을 때 요소가 따라가야 한다.
func fontVar(role soml.FontRole, theme soml.Theme) (string, error) {
	if role == "" {
		return "", fmt.Errorf("폰트 토큰이 해석되지 않았다")
	}
	if _, ok := theme.Font(role); !ok {
		return "", fmt.Errorf("폰트 토큰 %q가 테마에 없다", string(role))
	}
	return fmt.Sprintf("var(%s%s)", fontVarPrefix, string(role)), nil
}

// colorValue는 색을 CSS 값으로 만든다. 토큰은 **변수 참조로 남는다** — 여기서 RGB로 눕히면
// 테마 색을 바꿨을 때 이 요소만 옛 색으로 남고, 그것이 PPTX의 schemeClr 보존과 짝을 이루는
// 웹 쪽 절반이다 (soml.Color의 주석). 변형(shade/tint/alpha)은 `color-mix()`로 감싸 변수
// 참조를 살린다 (PRD §9.4의 색 변형 행 — 웹은 `color-mix()`). 그라디언트는 단일 색 값이
// 아니므로(background-image·gradient def) 여기서 받지 않는다 — 호출자가 갈린다.
func colorValue(color soml.Color, theme soml.Theme) (string, error) {
	if color.IsGradient() {
		return "", fmt.Errorf("그라디언트는 단일 색 값이 아니다 — linear-gradient나 gradient def로 낸다")
	}
	base, err := baseColorValue(color, theme)
	if err != nil {
		return "", err
	}
	if color.Mod == nil {
		return base, nil
	}
	// 변형은 base(var()·#hex 무엇이든)를 감싸므로 변수 참조가 살아남는다 — 테마 색을 바꾸면
	// 변형된 색도 따라 움직인다. `color-mix(in srgb, …)`는 sRGB 공간에서 섞는다 (PPTX
	// lumMod는 광도 공간이라 미세한 색 차가 남지만, 참조 보존이 매핑의 목적이다).
	switch color.Mod.Kind {
	case soml.ModShade:
		return fmt.Sprintf("color-mix(in srgb, %s, black %s%%)", base, formatNumber(color.Mod.Amount)), nil
	case soml.ModTint:
		return fmt.Sprintf("color-mix(in srgb, %s, white %s%%)", base, formatNumber(color.Mod.Amount)), nil
	case soml.ModAlpha:
		// transparent와 base를 섞으면 base가 (a×100)%만큼 남아 그 알파가 된다.
		return fmt.Sprintf("color-mix(in srgb, transparent, %s %s%%)", base, formatNumber(color.Mod.Amount*100)), nil
	}
	return "", fmt.Errorf("색 변형 %q는 어휘 밖이다", string(color.Mod.Kind))
}

// baseColorValue는 변형·그라디언트를 벗긴 단색의 CSS 값이다 (var() 참조 또는 #리터럴) —
// colorValue와 그라디언트 스톱(gradientCSS)이 공유한다.
func baseColorValue(color soml.Color, theme soml.Theme) (string, error) {
	switch {
	case color.IsZero():
		return "", fmt.Errorf("색이 해석되지 않았다")
	case color.IsToken():
		if _, ok := theme.Color(color.Token); !ok {
			// var()는 정의되지 않은 변수를 조용히 무시한다 — 검사하지 않으면 글자가 상속색으로
			// 그려지고 아무 에러도 나지 않는다.
			return "", fmt.Errorf("색 토큰 %q가 테마에 없다", color.Token)
		}
		return fmt.Sprintf("var(%s%s)", colorVarPrefix, color.Token), nil
	default:
		return "#" + color.RGB, nil
	}
}

// gradientCSS는 그라디언트를 CSS `linear-gradient()` 값으로 만든다 (PRD §9.4의
// `fill=gradient`·텍스트 `color=gradient` 행 — 웹은 `linear-gradient`/`background-clip`).
//
// 각도는 CSS 시맨틱 그대로다 (soml.Gradient가 CSS 각도를 담는다 — 변환은 PPTX 몫). 스톱은
// 균등 배치라 위치를 적지 않는다 (CSS가 균등 분배한다). 스톱 색은 baseColorValue를 지나
// 토큰이 var() 참조로 남는다.
func gradientCSS(grad soml.Gradient, theme soml.Theme) (string, error) {
	if len(grad.Stops) < 2 {
		return "", fmt.Errorf("그라디언트 스톱이 둘 미만이다 (%d)", len(grad.Stops))
	}
	stops := make([]string, len(grad.Stops))
	for i, stop := range grad.Stops {
		v, err := baseColorValue(stop, theme)
		if err != nil {
			return "", err
		}
		stops[i] = v
	}
	return fmt.Sprintf("linear-gradient(%sdeg, %s)", formatNumber(grad.Angle), strings.Join(stops, ", ")), nil
}

// alignValue는 논리 정렬 이름을 CSS text-align 값으로 만든다.
//
// 이름이 그대로 통하는 것은 우연이 아니다 — 논리 이름을 고른 이유가 CSS `start`/`end`와
// OOXML `algn`이 둘 다 문단 방향을 따르기 때문이다 (soml.TextAlign의 주석). 그래도 어휘를
// 검사하는 이유는 해석되지 않은 빈 값이나 물리 이름(`left`)이 CSS에 그대로 흘러가면
// 브라우저가 조용히 무시하기 때문이다.
func alignValue(align soml.TextAlign) (string, error) {
	for _, known := range soml.TextAligns() {
		if align == known {
			return string(align), nil
		}
	}
	return "", fmt.Errorf("정렬 %q는 논리 이름 어휘 밖이다", string(align))
}

// formatNumber는 수치를 자릿수를 늘리지 않고 쓴다 (24 → "24", 20.5 → "20.5").
// %g가 큰 수에서 지수 표기로 바뀌는 것을 피한다 — 골든에 `1e+03px`가 들어가면 안 된다.
func formatNumber(v float64) string {
	return strconv.FormatFloat(v, 'f', -1, 64)
}
