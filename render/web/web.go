// Package web은 덱 모델을 자체 완결형 정적 HTML로 렌더한다.
//
// 산출물은 파일 하나다 — CSS와 뷰어 런타임 JS를 인라인해 어디에 두어도 열린다. 사람이
// 열면 발표자 뷰다: 슬라이드 한 장이 화면에 맞춰 나오고, 하단 컨트롤 바로 넘기며, 노트를
// 패널로 연다 (runtime.go — 뷰 모드 계약도 거기에 있다). 같은 HTML/CSS가 PPTX 렌더러의
// 지오메트리 패스 입력이기도 하다: 웹과 PPTX의 배치가 구조적으로 일치하는 이유는 두
// 타깃이 같은 레이아웃 엔진의 결과를 쓰기 때문이다. 지오메트리 패스는 발표자 뷰가 아니라
// **문서 뷰**(스케일·겹침 없는 세로 스택)를 잰다 — 전환은 <html>의 ViewAttr 하나다.
//
// # 요소 식별자 — 지오메트리 패스의 계약
//
// 렌더된 요소는 모두 `data-balpyo-id` 속성을 갖는다. 지오메트리 패스는 이 속성으로 요소를
// 집어 `getBoundingClientRect`를 읽고, 그 rect를 덱 요소에 되돌려 붙인다. 값의 형태는
//
//	s<슬라이드 인덱스>[.<자식 인덱스>]…
//
// 이고, 인덱스는 **덱 모델의 슬라이스 인덱스**다 (Slide.Index, Slide.Content,
// Element.Children). 렌더 결과를 파싱하지 않고 덱 모델만 보고 같은 문자열을 만들 수 있어야
// 지오메트리 패스가 rect를 요소에 붙일 수 있다 — 그래서 저자가 준 이름(`<slide id>`)이 아니라
// 구조적 위치를 쓴다. 함께 나가는 `data-balpyo-el`은 그 요소를 만든 SOML 태그 이름이며,
// 지오메트리 패스가 자기가 걷는 덱 노드와 대조해 양쪽이 어긋난 순간을 잡는 용도다.
//
// 이 계약의 **구현은 하나다**: 속성 이름은 IDAttr·ElAttr, 값은 ElementID, 요소를 집는
// 선택자는 SlideSelector·SlideIndexSelector·ElementSelector다. 지오메트리 패스(render/geom)와
// 테스트 하네스(internal/testkit)가 이것들을 부른다 — 같은 규칙을 저쪽에서 다시 구현하면
// 규칙이 갈라지는 날 좌표가 엉뚱한 요소에 붙고, 그 실패는 산출물을 열어 보기 전까지 조용하다.
//
// # 외관
//
// 외관은 core가 해석을 끝낸 soml.Appearance에서만 나온다 — 이 패키지는 요소의 Attrs를 다시
// 읽지 않는다. 색·폰트는 덱에 실린 토큰 표(soml.Theme)에서 만든 CSS 변수를 참조하므로
// 리터럴 색·폰트 이름이 요소 쪽에 나오지 않는다 (style.go).
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
	b.WriteString(themeCSS(deck.Theme))
	b.WriteString("</style>\n")
	b.WriteString("</head>\n")
	b.WriteString("<body>\n")

	fmt.Fprintf(&b, "<div class=\"balpyo-deck\" data-aspect=%q data-slides=\"%d\">\n",
		string(deck.Aspect), len(deck.Slides))
	for _, slide := range deck.Slides {
		if err := writeSlide(&b, deck, slide); err != nil {
			return nil, err
		}
	}
	b.WriteString("</div>\n")
	// 뷰어 런타임은 마지막에 인라인한다 — DOM이 다 실린 뒤 실행되고, 외부 참조가 없어
	// 자체 완결형이 유지된다. JS가 꺼져 있으면 문서 뷰(세로 스택) 그대로 열린다.
	b.WriteString("<script>\n")
	b.WriteString(runtimeJS)
	b.WriteString("\n</script>\n")
	b.WriteString("</body>\n")
	b.WriteString("</html>\n")

	return render.Files{IndexFile: []byte(b.String())}, nil
}

// writeSlide는 슬라이드 하나를 쓴다.
//
// `data-slide-*` 속성은 장식이 아니다 — 지오메트리 패스가 이 선택자로 요소를 집어
// 좌표를 읽고, 웹 런타임이 딥링크를 여기에 맞춘다.
func writeSlide(b *strings.Builder, deck *soml.Deck, slide *soml.Slide) error {
	fmt.Fprintf(b, "<section class=%q id=%q %s=\"%d\" data-slide-source=%q>\n",
		SlideClass, html.EscapeString(slide.ID), SlideIndexAttr, slide.Index, html.EscapeString(slide.Source))
	if err := writeStage(b, deck, slide); err != nil {
		return err
	}
	if slide.Notes != "" {
		// 노트는 발표자용이다 — 슬라이드에는 나오지 않지만 문서에는 있어야 한다
		// (PPTX 노트 슬라이드와 같은 내용). 뷰어 런타임의 노트 패널이 여기서 읽는다.
		fmt.Fprintf(b, "<div class=%q hidden>%s</div>\n", NotesClass, html.EscapeString(slide.Notes))
	}
	b.WriteString("</section>\n")
	return nil
}

// baseCSS는 엔진이 소유하는 기본 CSS다 — 배치와 좌표계만 담고 색·폰트는 담지 않는다.
// 색·폰트는 전부 테마 층에서 나온다 (themeCSS): 기본 CSS에 색 리터럴을 하나라도 남기면
// 테마를 바꿨을 때 그 자리만 옛 색으로 남는다.
//
// 예외는 뷰어 크롬(바탕·컨트롤 바·노트 패널, viewerCSS)의 색이다. 슬라이드 **밖**의
// 색이므로 덱의 색이 아니고, 토큰으로 두지도 않는다: 색 토큰은 모두 clrScheme 슬롯을
// 가져야 하는데(soml.ColorToken) 이 색들에는 대응하는 PPTX 개념이 없다.
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

  /* 뷰어 크롬의 바탕 — 슬라이드 밖의 색이므로 덱의 색이 아니다 (테마 토큰이 아닌 유일한 색). */
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
  }
%s%s}
`, stage.Width, stage.Height, contentCSS(), viewerCSS())
}
