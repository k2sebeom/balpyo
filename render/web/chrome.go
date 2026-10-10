package web

import (
	"fmt"
	"html"
	"strings"

	"github.com/k2sebeom/balpyo/soml"
)

// 이 파일은 덱 크롬(페이지 번호·푸터)의 웹 렌더다.
//
// 렌더러가 하는 일은 확정 모델을 옮기는 것뿐이다 — 제외 규칙도, 역할 → 외관 캐스케이드도,
// 번호 문자열도 값 층이 이미 적용했다 (core/chrome.go). 여기서 조건을 다시 보면 웹과 PPTX가
// 갈라진다.

// 크롬 선택자다. export하는 이유는 StageClass와 같다: 지오메트리 패스가 이 선택자로 크롬을
// 집어 rect를 읽고(render/geom), PPTX가 그 rect에 도형을 놓는다. 문자열이 두 벌이 되면
// 크롬이 "못 찾았다"가 아니라 "PPTX에만 없다"로 조용히 사라진다.
const (
	// ChromeClass는 크롬 요소의 클래스다 — 스테이지의 형제다.
	ChromeClass = "balpyo-chrome"
	// ChromeKindAttr는 크롬 종류다 (soml.ChromeKind). 지오메트리 패스가 어느 rect가 번호이고
	// 어느 rect가 푸터인지 알아야 PPTX가 같은 자리에 같은 도형을 놓는다 — 문서 순서만으로는
	// 제외 규칙이 한쪽을 뺀 슬라이드에서 짝이 어긋난다.
	ChromeKindAttr = "data-chrome-kind"
)

// ChromeSelector는 슬라이드 안의 크롬을 문서 순서로 집는 선택자다.
func ChromeSelector() string { return "." + ChromeClass }

// writeChrome은 슬라이드의 확정된 크롬을 쓴다 — 스테이지 **다음**, 슬라이드 안이다.
//
// 항목이 없으면 아무것도 쓰지 않는다 (빈 컨테이너도 없다): 크롬을 켜지 않은 덱의 산출물이
// 이 기능 때문에 움직이지 않아야 한다 (골든 불변, 빈 스테이지를 빈 요소 하나로 낸 것과 같은
// 근거). 항목마다 요소 하나인 이유는 붙는 쪽이 항목마다 다르다는 것이다 — 한 컨테이너에
// 넣고 `justify-content: space-between`으로 흩으면 푸터만 켠 덱에서 푸터가 왼쪽이 아니라
// 띠 전체를 차지해 PPTX의 도형 폭과 갈라진다.
func writeChrome(b *strings.Builder, deck *soml.Deck, slide *soml.Slide) error {
	for _, item := range slide.Chrome {
		align, err := chromeAlignClass(item.Align)
		if err != nil {
			return fmt.Errorf("%s: %w", slide.Source, err)
		}
		if item.Kind == soml.ChromeLogo {
			if err := writeChromeLogo(b, deck, item, align); err != nil {
				return fmt.Errorf("%s: chrome (%s): %w", slide.Source, item.Kind, err)
			}
			continue
		}
		decls, err := textStyleCSS(item.Style, deck.Theme)
		if err != nil {
			return fmt.Errorf("%s: chrome (%s): %w", slide.Source, item.Kind, err)
		}
		fmt.Fprintf(b, "<div class=%q %s=%q%s>%s</div>\n",
			ChromeClass+" "+align, ChromeKindAttr, string(item.Kind),
			styleAttr([]string{decls}), html.EscapeString(item.Prefix+item.Text+item.Suffix))
	}
	return nil
}

// chromeAlignClass는 붙는 쪽의 클래스다. 어휘 밖 값은 에러다 — core가 이미 걸렀으므로
// 여기 닿는 것은 렌더러 버그이고, 조용히 한쪽으로 붙이면 웹과 PPTX가 갈라진다.
func chromeAlignClass(align soml.TextAlign) (string, error) {
	switch align {
	case soml.AlignStart:
		return ChromeClass + "-start", nil
	case soml.AlignCenter:
		return ChromeClass + "-center", nil
	case soml.AlignEnd:
		return ChromeClass + "-end", nil
	}
	return "", fmt.Errorf("unknown chrome alignment %q (start, center, end)", string(align))
}

// chromeCSS는 바닥 띠의 배치다 — 크롬을 쓴 덱에만 나간다 (features.chrome).
//
// 띠는 스테이지 여백(StagePadding)과 **같은 높이**다: 크롬이 콘텐츠 박스를 침범하지 않는다는
// 사실이 상수 하나로 보여야 한다 (여백 안에 있으므로 본문과 겹칠 수 없다). 좌우 들여쓰기도
// 같은 상수라 크롬의 왼쪽 끝이 본문의 왼쪽 끝과 맞는다.
//
// 세로 가운데 정렬(flex)은 PPTX의 `anchor="ctr"`과 짝이다 — 지오메트리 패스가 읽는 rect는
// 띠 전체 높이이고, 그 안에서 글줄이 어디 앉는지는 두 타깃이 같은 규칙으로 정해야 한다.
// `white-space: nowrap`은 크롬이 한 줄이라는 계약이다 (두 줄이 되면 띠를 넘어 본문을 먹는다).
func chromeCSS(feat features) string {
	if !feat.chrome {
		return ""
	}
	return fmt.Sprintf(`
  /* Deck chrome is a sibling of the stage — it sits outside the flow and does not take up body
     height. */
  .%[1]s {
    position: absolute;
    bottom: 0;
    height: %[2]dpx;
    display: flex;
    align-items: center;
    white-space: nowrap;
  }

  .%[1]s-start { left: %[2]dpx; }
  .%[1]s-center { left: 50%%; transform: translateX(-50%%); }
  .%[1]s-end { right: %[2]dpx; }
`, ChromeClass, StagePadding)
}

// writeChromeLogo는 그림 크롬 하나를 `<img>`로 쓴다 — 크롬 클래스를 **그림 자신**이 가진다.
//
// 글자 크롬은 띠 높이의 상자이고 글자가 그 가운데에 놓이지만, 그림은 상자가 곧 그림이어야 한다: 지오메트리
// 패스가 크롬 요소의 rect를 재고 PPTX가 그 rect에 `p:pic`을 놓으므로, 띠 상자를 재면 그림이 띠 높이로
// 늘어난다. 그래서 높이는 선언의 값이고, 띠 가운데에 오도록 바닥에서 (띠 − 높이)/2만큼 띄운다.
func writeChromeLogo(b *strings.Builder, deck *soml.Deck, item soml.ChromeItem, align string) error {
	if item.Image == "" || item.Height <= 0 {
		return fmt.Errorf("logo has no picture or height — the deck did not go through core.Resolve")
	}
	if _, ok := deck.Assets.Get(item.Image); !ok {
		return fmt.Errorf("%q is not in the asset table — the deck did not go through core.ResolveAssets", item.Image)
	}
	decls := []string{
		fmt.Sprintf("height: %spx", formatNumber(item.Height)),
		"width: auto",
		fmt.Sprintf("bottom: %spx", formatNumber((float64(StagePadding)-item.Height)/2)),
	}
	fmt.Fprintf(b, "<img class=%q %s=%q src=%q alt=%q%s>\n",
		ChromeClass+" "+align, ChromeKindAttr, string(item.Kind), item.Image, html.EscapeString(item.Alt), styleAttr(decls))
	return nil
}
