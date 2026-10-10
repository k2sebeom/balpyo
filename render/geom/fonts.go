package geom

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/css"
	"github.com/chromedp/cdproto/dom"
	"github.com/chromedp/chromedp"

	"github.com/k2sebeom/balpyo/soml"
)

// FontProbe는 측정 브라우저에 물어볼 글꼴 하나다.
type FontProbe struct {
	// Family는 테마가 적은 글꼴 이름이다 (`latin`·`ea` 슬롯의 값).
	Family string
	// Sample은 그 글꼴로 찍어 볼 글자다. 슬롯마다 다르다: `ea` 글꼴을 라틴 글자로 물으면
	// 그 글꼴이 있어도 라틴 글리프가 없는 면은 다른 글꼴로 떨어질 수 있다.
	Sample string
}

// 슬롯별 표본 글자. 라틴은 대소문자·숫자를, 동아시아는 한글·한자·가나를 고루 담는다 —
// 한 문자권만 담은 CJK 면(한글 전용 글꼴)도 자기 글자는 그리므로 "있다"로 읽혀야 한다.
const (
	LatinSample     = "Hamburgefonstiv 0123"
	EastAsianSample = "한글 漢字 かな"
)

// fontProbeClass는 탐침 요소의 클래스다. 산출물 문서에 잠깐 붙였다가 떼므로 산출물의
// 어휘(render/web)와 겹치지 않는 이름이어야 한다.
const fontProbeClass = "balpyo-font-probe"

// MissingFonts는 지금 열린 문서에서 probes 가운데 **브라우저가 실제로 쓰지 못한** 글꼴 이름을
// 돌려준다 (probes 순서, 중복 없음).
//
// 판정은 브라우저가 글리프를 그릴 때 쓴 면을 묻는 것이다 (CDP `CSS.getPlatformFontsForNode`).
// 폭 비교로는 알 수 없다: CJK 글리프는 거의 모든 글꼴에서 전각 1em이라 대체 글꼴로 떨어져도
// 폭이 같다. 그런데 이 어긋남이 문제인 이유는 폭이 아니라 **누가 어떤 면을 쓰는가**다 —
// 지오메트리 패스는 대체 글꼴로 재고, PowerPoint는 테마에 적힌 글꼴로 그린다. 그 글꼴로 그린
// 줄 폭·줄 높이가 잰 박스와 맞지 않으면 PPTX의 텍스트 박스가 넘치거나 남는다.
//
// 탐침은 글꼴 하나에 `<span>` 하나다. 스택에 제네릭을 두지 않는다 — 없을 때 무엇으로
// 떨어지든 그 면의 이름은 요청한 이름과 다르다.
func (s *Session) MissingFonts(probes []FontProbe) ([]string, error) {
	if len(probes) == 0 {
		return nil, nil
	}
	specs := make([]fontSpec, len(probes))
	for i, p := range probes {
		specs[i] = fontSpec{Stack: quoteFamily(p.Family), Sample: p.Sample}
	}
	usages, err := s.platformFonts(specs)
	if err != nil {
		return nil, err
	}
	var missing []string
	seen := map[string]bool{}
	for i, p := range probes {
		if seen[p.Family] {
			continue
		}
		if !usesFamily(usages[i], p.Family) {
			seen[p.Family] = true
			missing = append(missing, p.Family)
		}
	}
	return missing, nil
}

// fontSpec은 탐침 하나다 — Stack은 CSS `font-family` 값 그대로다.
type fontSpec struct {
	Stack  string `json:"stack"`
	Sample string `json:"sample"`
}

// quoteFamily는 글꼴 이름을 CSS 문자열로 감싼다. 감싸지 않으면 `serif` 같은 이름이 제네릭
// 키워드로 읽히고, 숫자로 시작하는 이름은 선언 자체가 무효가 된다.
func quoteFamily(name string) string {
	return `"` + strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(name) + `"`
}

// platformFonts는 탐침마다 브라우저가 글리프를 그린 면 목록을 돌려준다 (specs 순서).
func (s *Session) platformFonts(specs []fontSpec) ([][]*css.PlatformFontUsage, error) {
	payload, err := json.Marshal(specs)
	if err != nil {
		return nil, err
	}
	// 화면 밖에 붙인다 — 측정은 이미 끝났지만, 같은 세션이 뒤이어 스크린샷을 찍을 수 있다
	// (`balpyo preview`). 그리고 끝나면 반드시 뗀다.
	inject := fmt.Sprintf(`(() => {
  for (const p of %s) {
    const span = document.createElement("span");
    span.className = %q;
    span.style.cssText = "position:fixed;left:-10000px;top:0;white-space:nowrap;font-size:32px";
    span.style.fontFamily = p.stack;
    span.textContent = p.sample;
    document.body.appendChild(span);
  }
  // 붙인 탐침을 배치시킨 뒤 묻는다 — 배치 전의 노드는 그린 면이 없어 빈 목록이 온다. 새 이름이
  // 글꼴 적재를 시작했으면 그것도 끝나야 대체 면이 아니라 그 면으로 그려진다.
  void document.body.offsetWidth;
  return document.fonts.ready.then(() => true);
})()`, payload, fontProbeClass)
	remove := fmt.Sprintf(`document.querySelectorAll(%q).forEach((el) => el.remove())`, "."+fontProbeClass)

	if err := s.run(chromedp.Evaluate(inject, nil, withAwaitPromise)); err != nil {
		return nil, fmt.Errorf("cannot attach the font probes: %w", err)
	}
	defer s.Eval(remove, nil)

	var usages [][]*css.PlatformFontUsage
	err = s.run(chromedp.ActionFunc(func(ctx context.Context) error {
		if err := dom.Enable().Do(ctx); err != nil {
			return err
		}
		if err := css.Enable().Do(ctx); err != nil {
			return err
		}
		// 깊이 전체(-1)로 받는다: 글꼴 사용은 글자를 그린 **텍스트 노드**에 붙어 있고(요소
		// 노드에 물으면 빈 목록이 온다), 텍스트 노드의 id는 트리를 받아야 생긴다.
		root, err := dom.GetDocument().WithDepth(-1).Do(ctx)
		if err != nil {
			return err
		}
		texts := probeTextNodes(root)
		if len(texts) != len(specs) {
			return fmt.Errorf("attached %d font probes but found %d", len(specs), len(texts))
		}
		for _, id := range texts {
			fonts, err := css.GetPlatformFontsForNode(id).Do(ctx)
			if err != nil {
				return err
			}
			usages = append(usages, fonts)
		}
		return nil
	}))
	if err != nil {
		return nil, fmt.Errorf("cannot query fonts in the measuring browser: %w", err)
	}
	return usages, nil
}

// probeTextNodes는 탐침 `<span>`의 텍스트 노드 id를 문서 순서로 모은다 (붙인 순서 = probes 순서).
func probeTextNodes(n *cdp.Node) []cdp.NodeID {
	var out []cdp.NodeID
	var walk func(*cdp.Node)
	walk = func(n *cdp.Node) {
		if n.NodeType == cdp.NodeTypeElement && n.AttributeValue("class") == fontProbeClass {
			for _, c := range n.Children {
				if c.NodeType == cdp.NodeTypeText {
					out = append(out, c.NodeID)
					break
				}
			}
			return
		}
		for _, c := range n.Children {
			walk(c)
		}
	}
	walk(n)
	return out
}

// usesFamily는 브라우저가 그린 면 가운데 family가 있는지 본다.
//
// 가족 이름 비교만으로는 모자라다: 운영체제가 가족 이름을 지역화해 돌려줄 수 있다 (한국어
// Windows는 "Malgun Gothic"을 "맑은 고딕"으로 부를 수 있다). PostScript 이름은 ASCII로
// 고정이라 그 쪽도 본다 — 공백을 뺀 이름이 접두로 맞으면 같은 가족이다
// ("MalgunGothic-Bold", "NotoSansCJKkr-Regular"는 대소문자를 무시하고 비교한다).
func usesFamily(fonts []*css.PlatformFontUsage, family string) bool {
	want := strings.ToLower(family)
	squashed := strings.ReplaceAll(want, " ", "")
	for _, f := range fonts {
		if f == nil || f.GlyphCount == 0 {
			continue
		}
		if strings.ToLower(f.FamilyName) == want {
			return true
		}
		ps := strings.ToLower(f.PostScriptName)
		if ps != "" && strings.HasPrefix(ps, squashed) {
			return true
		}
	}
	return false
}

// FontDiagnostics는 측정 브라우저에 없던 테마 글꼴마다 경고 하나를 낸다 (missing 순서).
//
// 위치가 없는 경고다 — 원인은 슬라이드가 아니라 테마와 이 기계의 조합이고, 같은 테마를 쓴 덱은
// 설치된 기계에서 경고 없이 빌드된다. 그래서 메시지가 고칠 길 둘(설치하거나 바꾸거나)을 함께
// 말한다. 어느 토큰 · 슬롯이 그 이름을 적었는지도 말한다: 테마를 고칠 사람이 찾을 자리다.
func FontDiagnostics(theme soml.Theme, missing []string) soml.Diagnostics {
	var diags soml.Diagnostics
	for _, family := range missing {
		var slots []string
		for _, token := range theme.Fonts {
			if token.Latin == family {
				slots = append(slots, string(token.Name)+".latin")
			}
			if token.EastAsian == family {
				slots = append(slots, string(token.Name)+".ea")
			}
		}
		diags = append(diags, soml.Warningf("", soml.Span{},
			"theme %q font %q (%s) is missing from the measuring browser — the layout was measured with a fallback font, so "+
				"text boxes will be off when PowerPoint draws with this font. Install the font on the build machine, or change the theme's font token to an installed font",
			theme.Name, family, strings.Join(slots, ", ")))
	}
	return diags
}
