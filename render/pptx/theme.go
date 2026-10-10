package pptx

import (
	"fmt"
	"strings"

	"github.com/k2sebeom/balpyo/soml"
)

// clrSchemeFallbacks는 테마 표에 그 슬롯의 토큰이 **없을 때** 값을 빌려 오는 슬롯이다.
// 자리와 순서 자체는 soml.ColorSlots()가 소유한다 (열두 자리 — 값 층의 검사와 같은 목록을
// 읽어야 "로더는 받는데 렌더러는 모르는 슬롯"이 생기지 않는다). 여기 없는 슬롯은 **필수**다:
// 테마 표가 반드시 채워야 한다.
//
// 자리를 비워 둘 수 없는 이유: clrScheme은 12자리 고정이고, PowerPoint의 테마 색 편집 UI에
// 전부 보인다. Office 기본값 같은 팔레트 밖 색으로 채우면 사용자가 그 자리를 골랐을 때 테마와
// 무관한 색이 나온다 — 이미 있는 가장 가까운 토큰에서 빌린다 (두 번째 어두움/밝음은 첫
// 번째에서, 나머지 강조와 링크는 accent1에서). 표의 모든 토큰이 자기 슬롯을 가지므로
// (soml.ColorToken), 빌린 자리를 **참조하는** 도형은 생기지 않는다 — 이 채움은 편집 UI를
// 위한 것이다.
//
// 내장 minimal 테마는 강조색 여섯을 모두 갖는다 — 그래서 기본 테마에서
// 빌리는 자리는 hlink/folHlink 둘뿐이다 (어휘에 링크 색이 없다). 폴백을 남겨 두는 이유는
// 다른 테마가 강조색을 하나만 정해도 열리는 파일을 내야 하기 때문이다.
var clrSchemeFallbacks = map[string]string{
	"dk2":      "dk1",
	"lt2":      "lt1",
	"accent2":  "accent1",
	"accent3":  "accent1",
	"accent4":  "accent1",
	"accent5":  "accent1",
	"accent6":  "accent1",
	"hlink":    "accent1",
	"folHlink": "accent1",
}

// clrSchemeXML은 덱에 실린 토큰 표를 테마 파트의 clrScheme으로 컴파일한다.
//
// 웹의 CSS 변수(render/web.themeCSS)와 이 함수가 **같은 한 표**에서 나오는 것이 토큰 보존
// 계약의 뿌리다: `color="accent1"`인 텍스트가 웹에서는 `var(--balpyo-color-accent1)`,
// PPTX에서는 `schemeClr val="accent1"`이 되고, 둘 다 이 표의 같은 RGB로 해석된다.
// 표가 렌더러마다 따로 있으면 같은 토큰이 두 색이 되는 것을 막을 방법이 없다.
func clrSchemeXML(theme soml.Theme) (string, error) {
	values := make(map[string]string, len(theme.Colors))
	for _, token := range theme.Colors {
		if token.Slot == "" {
			// 슬롯 없는 색 토큰은 테마 색 편집을 따라가지 못한다 (soml.ColorToken의 주석) —
			// 여기서 막아야 그 상태가 파일을 열어 보기 전에 드러난다.
			return "", fmt.Errorf("color token %q has no clrScheme slot", token.Name)
		}
		if first, dup := values[token.Slot]; dup {
			return "", fmt.Errorf("clrScheme slot %q is assigned twice (%s and %s) — cannot decide which color wins",
				token.Slot, first, token.RGB)
		}
		values[token.Slot] = token.RGB
	}

	var b strings.Builder
	fmt.Fprintf(&b, `    <a:clrScheme name="%s">`+"\n", escapeXML(theme.Name))
	for _, slot := range soml.ColorSlots() {
		rgb, ok := values[slot]
		if !ok {
			if fallback, has := clrSchemeFallbacks[slot]; has {
				rgb, ok = values[fallback]
			}
		}
		if !ok {
			return "", fmt.Errorf("theme %q has no color token for clrScheme slot %s", theme.Name, slot)
		}
		fmt.Fprintf(&b, `      <a:%s><a:srgbClr val="%s"/></a:%s>`+"\n", slot, rgb, slot)
	}
	b.WriteString("    </a:clrScheme>")
	return b.String(), nil
}

// fontSchemeXML은 폰트 토큰 표를 테마 파트의 fontScheme으로 컴파일한다.
//
// major/minor 두 자리는 rPr의 `+mj-lt`/`+mn-lt` 참조(shape.go의 typefaces)가 해석되는
// 곳이다 — 슬롯이 빈 토큰(mono)은 rPr에 리터럴로 실리므로 여기 나오지 않는다.
// cs(complex script) 자리는 빈 문자열로 둔다: 토큰 표에 대응 개념이 없고, 빈 typeface는
// 뷰어의 폰트 대체에 맡긴다는 뜻이다.
func fontSchemeXML(theme soml.Theme) (string, error) {
	var major, minor *soml.FontToken
	for i := range theme.Fonts {
		token := &theme.Fonts[i]
		switch token.Scheme {
		case "major":
			major = token
		case "minor":
			minor = token
		}
	}
	if major == nil || minor == nil {
		return "", fmt.Errorf("theme %q font table has empty fontScheme major/minor slots — rPr +mj-lt/+mn-lt references have nothing to resolve to",
			theme.Name)
	}

	var b strings.Builder
	fmt.Fprintf(&b, `    <a:fontScheme name="%s">`+"\n", escapeXML(theme.Name))
	for _, f := range []struct {
		tag   string
		token *soml.FontToken
	}{{"majorFont", major}, {"minorFont", minor}} {
		fmt.Fprintf(&b, "      <a:%s>\n", f.tag)
		fmt.Fprintf(&b, `        <a:latin typeface="%s"/>`+"\n", escapeXML(f.token.Latin))
		fmt.Fprintf(&b, `        <a:ea typeface="%s"/>`+"\n", escapeXML(f.token.EastAsian))
		b.WriteString("        <a:cs typeface=\"\"/>\n")
		fmt.Fprintf(&b, "      </a:%s>\n", f.tag)
	}
	b.WriteString("    </a:fontScheme>")
	return b.String(), nil
}
