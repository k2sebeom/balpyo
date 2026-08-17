// Package web은 덱 모델을 자체 완결형 정적 HTML로 렌더한다.
//
// 산출물은 파일 하나다 — CSS와 (나중의) 런타임 JS를 인라인해 어디에 두어도 열린다.
// 같은 HTML/CSS가 PPTX 렌더러의 지오메트리 패스 입력이기도 하다: 웹과 PPTX의 배치가
// 구조적으로 일치하는 이유는 두 타깃이 같은 레이아웃 엔진의 결과를 쓰기 때문이다.
package web

import (
	"fmt"
	"html"
	"strings"

	"github.com/k2sebeom/balpyo/render"
	"github.com/k2sebeom/balpyo/soml"
)

// IndexFile은 웹 타깃의 산출 파일 이름이다.
const IndexFile = "index.html"

// Renderer는 웹 타깃 렌더러다.
type Renderer struct{}

// New는 웹 렌더러를 만든다.
func New() *Renderer { return &Renderer{} }

// Target은 render.TargetHTML이다.
func (*Renderer) Target() render.Target { return render.TargetHTML }

// Render는 덱을 index.html 하나로 만든다.
func (*Renderer) Render(deck *soml.Deck) (render.Files, error) {
	if deck == nil {
		return nil, fmt.Errorf("덱이 nil이다")
	}
	stage := deck.Stage
	if stage.Width <= 0 || stage.Height <= 0 {
		return nil, fmt.Errorf("논리 스테이지 크기가 유효하지 않다: %dx%d", stage.Width, stage.Height)
	}

	var b strings.Builder
	lang := deck.Lang
	if lang == "" {
		lang = "en"
	}

	b.WriteString("<!DOCTYPE html>\n")
	fmt.Fprintf(&b, "<html lang=%q>\n", lang)
	b.WriteString("<head>\n")
	b.WriteString(`<meta charset="utf-8">` + "\n")
	b.WriteString(`<meta name="viewport" content="width=device-width, initial-scale=1">` + "\n")
	fmt.Fprintf(&b, "<title>%s</title>\n", html.EscapeString(deck.Title))
	if deck.Author != "" {
		fmt.Fprintf(&b, "<meta name=\"author\" content=%q>\n", html.EscapeString(deck.Author))
	}
	b.WriteString("<style>\n")
	b.WriteString(baseCSS(stage))
	b.WriteString("</style>\n")
	b.WriteString("</head>\n")
	b.WriteString("<body>\n")

	fmt.Fprintf(&b, "<div class=\"balpyo-deck\" data-aspect=%q data-slides=\"%d\">\n",
		string(deck.Aspect), len(deck.Slides))
	for _, slide := range deck.Slides {
		writeSlide(&b, slide)
	}
	b.WriteString("</div>\n")
	b.WriteString("</body>\n")
	b.WriteString("</html>\n")

	return render.Files{IndexFile: []byte(b.String())}, nil
}

// writeSlide는 슬라이드 하나를 쓴다.
//
// `data-slide-*` 속성은 장식이 아니다 — 지오메트리 패스가 이 선택자로 요소를 집어
// 좌표를 읽고, 웹 런타임이 딥링크를 여기에 맞춘다.
func writeSlide(b *strings.Builder, slide *soml.Slide) {
	fmt.Fprintf(b, "<section class=\"balpyo-slide\" id=%q data-slide-index=\"%d\" data-slide-source=%q>\n",
		html.EscapeString(slide.ID), slide.Index, html.EscapeString(slide.Source))
	b.WriteString(`<div class="balpyo-stage">`)
	// 콘텐츠 노드 렌더는 Iteration 1부터다 (`<slide>` `<text>` `<note>`).
	b.WriteString("</div>\n")
	if slide.Notes != "" {
		// 노트는 발표자용이다 — 화면에 나오지 않지만 문서에는 있어야 한다
		// (PPTX 노트 슬라이드와 같은 내용).
		fmt.Fprintf(b, "<div class=\"balpyo-notes\" hidden>%s</div>\n", html.EscapeString(slide.Notes))
	}
	b.WriteString("</section>\n")
}

// baseCSS는 엔진이 소유하는 기본 CSS다.
//
// cascade layer로 기본 < 테마 < 저자 우선순위를 보장한다 (PRD §12.2). 층을 미리 선언해
// 두면 나중에 테마·저자 CSS가 어느 순서로 실려도 우선순위가 뒤집히지 않는다.
func baseCSS(stage soml.Stage) string {
	return fmt.Sprintf(`@layer balpyo.base, balpyo.theme, balpyo.author;

@layer balpyo.base {
  :root {
    --balpyo-stage-width: %dpx;
    --balpyo-stage-height: %dpx;
  }

  * { box-sizing: border-box; }

  html, body {
    margin: 0;
    padding: 0;
    background: #111;
  }

  .balpyo-deck {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 24px;
    padding: 24px;
  }

  /* 슬라이드는 논리 스테이지 크기를 그대로 갖는다. 저자·웹·지오메트리 패스가 같은
     좌표계를 쓰므로 지오메트리 패스가 읽은 rect가 곧 PPTX 좌표다 (환산 분기 없음). */
  .balpyo-slide {
    position: relative;
    width: var(--balpyo-stage-width);
    height: var(--balpyo-stage-height);
    overflow: hidden;
    background: #fff;
  }

  .balpyo-stage {
    position: relative;
    width: 100%%;
    height: 100%%;
  }
}
`, stage.Width, stage.Height)
}
