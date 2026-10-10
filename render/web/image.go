package web

import (
	"fmt"
	"html"
	"strings"

	"github.com/k2sebeom/balpyo/render"
	"github.com/k2sebeom/balpyo/soml"
)

// 이 파일은 **`<image>` → 박스 + `<img>` + 에셋 파일**이다.
//
// # 그림 하나는 박스 하나와 `<img>` 하나다
//
//	<div class="balpyo-image" …>        ← 저자가 준 박스. pad·크기·비율·자식 배치가 여기 걸린다.
//	  <img class="balpyo-image-ink" …>  ← 잰 박스 = **그려진 그림**. fit·외관이 여기 걸린다.
//
// 도형과 같은 두 층이지만 **식별자가 붙는 층이 반대**다: 도형은 겉 박스를 재고(그림 svg는
// 박스를 덮으므로 둘이 같다), 그림은 `<img>`를 잰다. 이유는 아래 fit 결정이다 — 저자가 준
// 박스와 그려진 그림은 `contain`에서 갈라지고, `pic`의 `xfrm`이 되어야 하는 것은 **그림**이다.
//
// `<img>` 하나만 내고 박스 속성을 그 요소에 거는 길은 실측으로 버렸다: `contain`의 매핑
// (`max-width`/`max-height`)은 **부모가 준 공간**을 기준으로 하므로, 저자의 `w`/`h`/`ratio`가
// 같은 요소에 실리면 그 값이 곧 요소 크기가 되어 그림이 늘어난다 (`<img>`의 기본 object-fit은
// `fill`이다). 박스를 부모로 올리면 `100%`가 저자의 박스를 가리키고, 잰 rect는 언제나 그림이다.
//
// `width`/`height` **HTML 속성은 내지 않는다**: 박스는 크기 속성과 CSS가 정하고, 고유 크기를
// 속성으로 한 번 더 말하면 두 곳이 어긋난다 — `<img>`는 이미 자기 고유 크기를 기본 박스로 갖는다.
//
// # `fit`의 CSS — 잰 rect가 곧 그려진 그림이다
//
//	fit        CSS                                          잰 rect의 뜻
//	cover      width:100%; height:100%; object-fit:cover     박스 = 그림 (넘치는 쪽이 잘린다)
//	contain    max-width:100%; max-height:100%               박스 = 맞춰진 그림 (여백은 박스 밖)
//
// `contain`을 `object-fit: contain`으로 내지 **않는** 것이 이 파일의 중심 결정이다.
// `object-fit`은 요소 박스를 그대로 두고 그 안에 그림을 앉히므로 박스와 그림이 갈라지고,
// 그러면 셋이 따라온다: (a) 잰 rect가 그림이 아니라 **여백을 포함한 박스**여서 PPTX가
// letterbox를 다시 계산해야 한다 (좌표 계산은 브라우저가 한 번만 한다는 이 엔진의 전제와
// 어긋난다), (b) 웹의 `border`·`radius`·`shadow`는 그 박스에 붙고 PPTX `pic`의
// 선·그림자는 그림에 붙어 **두 타깃의 외관이 갈라진다**, (c) 갈라짐이 픽셀 비교에서만 보이고
// 원인은 보이지 않는다. `max-*`로 내면 요소 박스가 곧 맞춰진 그림이 되어 셋이 함께 사라지고,
// PPTX는 잰 rect를 `xfrm`으로 그대로 쓴다.
//
// **`max-*`는 위로 키우지 않는다** — 실측한 사실이고 결정의 일부다: 고유 크기 16×8 그림을
// 560×420 박스에 `contain`으로 넣으면 잰 rect가 16×8로 남는다 (`object-fit: contain`이라면
// 560×280으로 커진다). 그래서 이 엔진의 `contain`은 "박스를 넘지 않는다"이지 "박스를 채운다"가
// 아니다. 확대를 열려면 CSS에 그림의 고유 비율을 실어야 하는데(`aspect-ratio` + `width:100%`),
// 그것도 실측에서 **비율을 보존하지 못했다**: 폭이 확정된 요소에서 `max-height`의 자름은
// 폭으로 되돌지 않아 세로로 긴 그림이 늘어난다. 즉 "박스 = 맞춰진 그림"과 "확대"는 CSS만으로
// 함께 얻을 수 없고, 그중 좌표를 한 곳에서 오게 하는 쪽을 골랐다.
//
// # 크기 속성이 없는 그림
//
// `fit`은 **박스가 정해진 다음에** 뜻을 갖는다 (soml.ImageAttrs의 주석). 크기 속성이 없는
// 그림의 박스는 고유 크기이므로 자를 것도 맞출 것도 없다 — 그때는 `cover`도 `max-*`만 낸다.
// 그러지 않으면 `cover`의 `width:100%`가 그림을 부모 폭까지 늘려(실측 1152×576) "크기 속성이
// 없으면 고유 크기"라는 계약이 `fit` 값에 따라 깨진다.
//
// # 외관은 도형의 함수를 쓴다 (갈라지는 자리는 CSS 프로퍼티뿐)
//
// 값 문법은 core가 이미 확정했고(soml.ImageStyle), 색·그림자·불투명도·대칭·회전의 **값**을
// 만드는 함수는 도형과 같은 것이다 (colorValue·flipValue·shadowVarPrefix·formatNumber).
// 갈라지는 것은 값을 싣는 프로퍼티다: 도형은 SVG(`stroke`·`rx`)에 걸고 그림은 CSS
// (`border-*`·`border-radius`)에 건다 — `<img>`에 SVG 프로퍼티는 아무 일도 하지 않는다.
// 그 대조를 테스트가 값 단위로 한다 (image_test.go의 TestImageAndShapeShareAppearanceValues).
//
// 갈라짐을 하나 남긴다: 파선 패턴이다. 도형은 `stroke-dasharray`로 OOXML `prstDash`의 비율을
// 정확히 내지만(dashArray), CSS `border-style: dashed`의 간격은 UA가 정한다. `<img>`에
// 파선 테두리를 SVG로 내려면 그림을 svg 안으로 넣어야 하고, 그러면 `<img>`가 주는 것
// (고유 크기·`alt`·디코드 신호)을 전부 잃는다 — 그 값이 파선 간격보다 크다.
//
// # 에셋은 렌더러가 낸다
//
// `Renderer.Render`가 `index.html` 외에 에셋 바이트를 `render.Files`에 담는다 (assetFiles).
// 경로는 **저자가 쓴 정규화 경로 그대로**(`assets/photo.png` = soml.Asset.Key)이므로 `<img src>`가
// `.slide`에 적힌 것과 같고, 산출 디렉토리를 열어 본 사람이 무엇이 무엇인지 안다.
//
// 파이프라인이 복사하는 길을 버린 이유는 **측정**이다: 지오메트리 패스는 웹 산출물을 임시
// 디렉토리에 쓰고 그것을 잰다 (pipeline.measure). 에셋이 `Files`에 없으면 그 디렉토리에
// 그림이 없고, 브라우저는 **깨진 이미지의 박스**(고유 크기 0 → rect 0×0, 실측)를 재서 그
// 좌표를 PPTX `xfrm`으로 보낸다. 그림이 산출 파일이라는 사실이 렌더러의 반환값에 있어야
// 측정과 배포가 같은 바이트를 본다 — `render.Files`가 경로→바이트 맵인 이유의 첫 실사용이다.
//
// # 측정 대상
//
// `<image>`는 잰다. 인라인이 아니고, 박스를 가지며, **그 rect가 `pic`의 `xfrm`이 된다** —
// 재는 것과 도형이 되는 것이 일치하는 두 번째 종류다 (첫째는 `<shape>` — web 패키지 주석의
// 측정 대상 계약).

// 그림 요소의 엔진 CSS 클래스다.
const (
	// imageClass는 저자가 준 박스다 (재지 않는다 — 식별자가 없다).
	imageClass = "balpyo-image"
	// imageInkClass는 그림 자신이다 (`<img>`) — 식별자가 붙는 요소.
	imageInkClass = "balpyo-image-ink"
	// imageTransformClass는 회전·대칭 선언을 받는 표식이다. 둘 중 하나라도 있는 그림에만
	// 붙는다 — 모든 그림에 `transform`을 걸면 변형이 없는 그림까지 스택 문맥과 래스터화
	// 경로가 바뀌고, 그 변화는 픽셀 비교에서만 드러난다 (shapeRotateClass와 같은 근거).
	imageTransformClass = "balpyo-image-transform"
)

// 회전 각도와 대칭을 싣는 커스텀 프로퍼티다 (선언은 엔진 CSS, 값은 인라인 — imageCSS).
//
// **하나의 `transform`에 둘이 함께 사는** 것이 도형과 다른 점이다: 도형은 회전을 박스에,
// 대칭을 그림 svg에 걸어 선언이 둘로 나뉘지만 그림은 요소가 하나뿐이고 `transform`은 마지막
// 선언만 남는다. 합성 순서는 `rotate(…) scale(…)`이다 — 오른쪽이 먼저 적용되므로 "대칭 후
// 회전"이고, 그것이 OOXML `xfrm`(flipH/flipV로 뒤집힌 기하를 `rot`으로 돌린다)과 같은 순서다.
const (
	imageRotateVar = "--balpyo-image-rotate"
	imageFlipVar   = "--balpyo-image-flip"
)

// writeImage는 `<image>`를 박스 div + `<img>`로 쓴다.
//
// 식별자는 **`<img>`가** 갖는다 (박스가 아니다 — 파일 머리 주석의 첫 절). 덱 모델의 요소
// 하나에 식별자 하나이므로 geom.Refs와의 자리 맞대기는 그대로 성립한다.
func writeImage(b *strings.Builder, deck *soml.Deck, slide *soml.Slide, elem *soml.Element, path []int) error {
	style := elem.ImageStyle()
	if style == nil {
		// 해석되지 않은 외관을 기본값으로 그리면 웹만 그럴듯하게 나오고 PPTX와 갈라진다
		// (writeShape·writeBlock과 같은 관문).
		return fmt.Errorf("%s:%s <%s> image appearance is unresolved — the deck did not go through core.Resolve",
			slide.Source, elem.Span, elem.Name)
	}
	if style.Src == "" {
		// 값 층이 이미 진단했다 (없는 src·경로 문법·지원하지 않는 포맷 — core/image.go).
		// 그래도 여기서 멈추는 이유는 `<img src="">`가 **현재 문서를 다시 받는** 요청이 되어
		// 브라우저가 그림이 아닌 바이트를 디코드하려 하기 때문이다 (측정이 그 상태를 잰다).
		return fmt.Errorf("%s:%s <%s> src is unresolved — nothing to draw",
			slide.Source, elem.Span, elem.Name)
	}
	asset, ok := deck.Assets.Get(style.Src)
	if !ok {
		// 에셋 패스를 거치지 않은 덱이다 (core.ResolveAssets). 표에 없으면 산출물에 바이트가
		// 실리지 않으므로 그림이 깨진 채로 나가고, 그 상태를 재면 rect가 0×0이 된다.
		return fmt.Errorf("%s:%s <%s>: %q is not in the asset table — the deck did not go through core.ResolveAssets",
			slide.Source, elem.Span, elem.Name, style.Src)
	}
	boxDecls, err := imageBoxCSS(*style, elem)
	if err != nil {
		return fmt.Errorf("%s:%s <%s>: %w", slide.Source, elem.Span, elem.Name, err)
	}
	inkDecls, err := imageInkCSS(*style, deck.Theme)
	if err != nil {
		return fmt.Errorf("%s:%s <%s>: %w", slide.Source, elem.Span, elem.Name, err)
	}
	class := imageInkClass
	if style.Rotate != nil || style.Flip != "" {
		class += " " + imageTransformClass
	}
	fmt.Fprintf(b, "<div class=%q%s>\n", imageClass, styleAttr(boxDecls))
	// `alt`은 **언제나** 낸다: 없을 때 `alt=""`(장식 이미지)를 내는 것과 속성 자체를 빼는 것은
	// 스크린 리더에 다른 뜻이고, 후자는 리더가 파일 이름을 읽는 상태다 (soml.imageOwnAttrs).
	fmt.Fprintf(b, "<img class=%q %s=%q %s=%q src=%q alt=%q%s>\n",
		class, IDAttr, ElementID(slide.Index, path), ElAttr, elem.Name,
		html.EscapeString(asset.Key), html.EscapeString(style.Alt), styleAttr(inkDecls))
	b.WriteString("</div>\n")
	return nil
}

// imageBoxCSS는 저자가 준 박스의 CSS 선언이다 — 안쪽 여백·크기·비율·자식 배치.
//
// 도형과 **같은 함수**에서 나온다 (padValue·boxSizeCSS·appendPlacement): soml.Box를 세 요소가
// 임베드로 공유하므로 매핑도 하나여야 한다. `pad`가 박스에 걸리는 것이 `pad`를 이미지에 열어
// 둔 것의 실물이다 — `box-sizing: border-box`(baseCSS)와 함께 여백은 안쪽으로
// 들어가고, `100%`/`max-*`가 가리키는 공간이 그만큼 줄어들어 **그림 영역이 여백만큼 준다**.
// 잰 rect는 그림이므로 PPTX가 `xfrm`에서 여백을 다시 뺄 일이 없다.
func imageBoxCSS(style soml.ImageStyle, elem *soml.Element) ([]string, error) {
	var decls []string
	if style.Pad != nil {
		decls = append(decls, "padding: "+padValue(*style.Pad))
	}
	sizes, err := boxSizeCSS(style.Box)
	if err != nil {
		return nil, err
	}
	decls = append(decls, sizes...)
	// 자식 배치는 **박스**에 걸린다: 그림은 컨테이너의 자식이 될 수 있고(`<grid>`의 칸,
	// `<row basis>`), flex/grid의 항목은 이 박스다 — 배치 선언을 `<img>`에 걸면 부모가 아는
	// 항목과 선언이 붙은 요소가 어긋난다.
	return appendPlacement(appendSlot(decls, elem), elem.Placement())
}

// imageInkCSS는 `<img>`의 CSS 선언이다 — `fit` 다음에 외관이 온다.
//
// **순서는 외관 매핑 표의 행 순서다** (border → radius → shadow → opacity → rotate → flip) —
// 표와 코드를 나란히 읽을 수 있어야 "표에 있고 코드에 없는 행"이 눈에 띈다. 정하지 않은
// 속성은 선언을 **하나도** 내지 않는다 (soml.ImageStyle의 nil 계약): 그림은 자기 픽셀을
// 가지므로 두 타깃의 암묵 기본값이 갈라지지 않는다 — 그래서 테마의 이미지 컴포넌트 기본값도
// 비어 있다 (테마의 image.default).
func imageInkCSS(style soml.ImageStyle, theme soml.Theme) ([]string, error) {
	decls, err := imageFitCSS(style)
	if err != nil {
		return nil, err
	}
	if style.Border != nil && !style.Border.None {
		color, err := colorValue(style.Border.Color, theme)
		if err != nil {
			return nil, err
		}
		if style.Border.Width <= 0 {
			return nil, fmt.Errorf("border width is unresolved (%s)", formatNumber(style.Border.Width))
		}
		borderStyle, err := borderStyleValue(style.Border.Style)
		if err != nil {
			return nil, err
		}
		// 축약(`border`)으로 되돌리지 않는 이유는 도형과 같다 — core가 이미 세 조각으로
		// 풀었고(soml.Border), 렌더러가 다시 조립하면 축약 규칙이 두 곳에 생긴다.
		decls = append(decls,
			"border-width: "+formatNumber(style.Border.Width)+"px",
			"border-style: "+borderStyle,
			"border-color: "+color)
	}
	if style.Radius != nil {
		// 그림의 모양은 직사각형과 반경으로 끝난다 — 프리셋 기하가 없다는 것이 도형과 갈라지는
		// 유일한 기하 차이다. `border-radius`는 `<img>`의 내용까지 자르므로(브라우저 계약)
		// PPTX의 `pic/spPr/prstGeom prst="roundRect"`와 짝이 된다 (반경을 rounded-rect에만
		// 허용하는 제약은 도형의 것이다 — soml.ImageStyle.Radius의 주석).
		decls = append(decls, "border-radius: "+formatNumber(style.Radius.Px)+"px")
	}
	if style.Shadow != nil {
		if _, ok := theme.Shadow(style.Shadow.Name); !ok {
			// var()는 정의되지 않은 변수를 조용히 무시한다 — 검사하지 않으면 그림자가 사라지고
			// 아무 에러도 나지 않는다 (shapeInkCSS와 같은 근거).
			return nil, fmt.Errorf("shadow token %q is not in the theme", style.Shadow.Name)
		}
		// `filter: drop-shadow`는 그림의 **알파 실루엣**을 따라간다 — 투명 배경 PNG의
		// 그림자가 사각형이 되지 않고, `border-radius`로 자른 모서리도 따라온다.
		// `box-shadow`를 쓰면 잘린 모서리 밖에 그림자가 남는다 (도형이 같은 이유로
		// drop-shadow를 쓴다 — shapeInkCSS).
		decls = append(decls, fmt.Sprintf("filter: drop-shadow(var(%s%s))", shadowVarPrefix, style.Shadow.Name))
	}
	if style.Opacity != nil {
		if *style.Opacity < 0 || *style.Opacity > 1 {
			return nil, fmt.Errorf("opacity %s is outside 0–1", formatNumber(*style.Opacity))
		}
		decls = append(decls, "opacity: "+formatNumber(*style.Opacity))
	}
	if style.Rotate != nil {
		// 각도만 싣는다 — `transform` 선언은 엔진 CSS가 갖는다 (측정 뷰가 그것을 무효화할 수
		// 있어야 한다: 인라인 style은 층 밖에서 이기므로 스타일시트가 덮을 수 없다).
		decls = append(decls, fmt.Sprintf("%s: %sdeg", imageRotateVar, formatNumber(*style.Rotate)))
	}
	if style.Flip != "" {
		flip, err := flipValue(style.Flip)
		if err != nil {
			return nil, err
		}
		// 대칭도 커스텀 프로퍼티다 — 회전과 **한 선언**을 나눠 쓰기 때문이다 (imageFlipVar).
		decls = append(decls, imageFlipVar+": "+flip)
	}
	return decls, nil
}

// imageFitCSS는 `fit`의 CSS 선언이다 (파일 머리 주석의 표).
func imageFitCSS(style soml.ImageStyle) ([]string, error) {
	switch style.Fit {
	case soml.FitCover:
		if boxHasSize(style.Box) {
			return []string{"width: 100%", "height: 100%", "object-fit: cover"}, nil
		}
		// 박스가 정해지지 않은 그림은 자를 것이 없다 — 아래 contain과 같은 선언을 낸다.
		return containFitDecls(), nil
	case soml.FitContain:
		return containFitDecls(), nil
	}
	// 어휘 밖 값은 core가 이미 막았다 (닫힌 어휘). 여기서 멈추는 이유는 빈 문자열이다:
	// 해석되지 않은 `Fit`을 통과시키면 `fit` 선언이 하나도 없는 그림이 나가고, 그 그림은
	// 박스가 정해졌을 때 **늘어난다** (`<img>`의 기본 object-fit은 `fill`이다).
	return nil, fmt.Errorf("fit %q is not a known fit value", string(style.Fit))
}

// containFitDecls는 "박스를 넘지 않는다"의 선언이다.
//
// 위로 키우지 않는다는 성질이 결정의 일부다 (파일 머리 주석) — 그래서 크기 속성이 없는
// 그림에도 같은 선언이 나간다: 고유 크기로 남고 부모가 준 공간을 넘으면 줄어든다.
func containFitDecls() []string {
	return []string{"max-width: 100%", "max-height: 100%"}
}

// boxHasSize는 저자가 박스의 **크기**를 정했는지다 (`w`/`h`/`ratio`).
//
// min/max를 세지 않는 이유: 그 둘은 박스가 아니라 **한계**이고, 한계만 준 그림은 고유
// 크기로 남아야 한다 (`max-w="400"`이 작은 로고를 400px로 키우면 저자가 준 것보다 큰
// 그림이 나간다). `contain`의 매핑이 그 한계를 이미 지킨다.
func boxHasSize(box soml.Box) bool {
	return box.Width != nil || box.Height != nil || box.Ratio != nil
}

// borderStyleValue는 테두리 스타일을 CSS `border-style` 값으로 만든다.
//
// 이름이 그대로 통하는 것은 어휘를 CSS 이름으로 고른 결과다 (soml.BorderStyle). 검사하는
// 이유는 해석되지 않은 빈 값이 CSS에 흘러가면 브라우저가 조용히 무시하기 때문이다.
//
// **간격은 UA가 정한다** — 도형의 `stroke-dasharray`(OOXML `prstDash`의 비율을 정확히 낸다)와
// 갈라지는 유일한 자리이고, 그 대가를 받는 근거는 파일 머리 주석에 있다.
func borderStyleValue(style soml.BorderStyle) (string, error) {
	switch style {
	case soml.BorderSolid, soml.BorderDashed, soml.BorderDotted:
		return string(style), nil
	}
	return "", fmt.Errorf("unknown border style %q", string(style))
}

// imageCSS는 그림의 엔진 CSS다 — **태그마다 고정인 선언만** 있다 (shapeCSS와 같은 경계).
//
// 그림이 있는 덱에만 낸다 (listCSS·layoutCSS·shapeCSS와 같은 근거): 그림이 없는 덱의
// 산출물이 이미지 렌더가 붙었다는 이유로 달라지면 골든 diff가 무엇이 움직였는지 말해 주지
// 못한다.
func imageCSS() string {
	return fmt.Sprintf(`
  /* <image> is **one box and one picture**. The box has the size the author gave and the
     picture is the actual fitted rect inside it (=PPTX xfrm). It is centered so that the empty
     space of contain is not pushed to one side — the picture lands where the default object-fit
     position (50%% 50%%) puts it. flex: 0 0 auto keeps the box from being squeezed when the
     stack is taller than the stage (the fixed-box policy, same reasoning as .balpyo-text) — if
     a squeezed box were measured, PPTX would draw the picture at that size.

     min-*: 0 turns off the **automatic minimum size** of flex/grid items. With it on, the box
     cannot be smaller than its content, so a box with a ratio grows because of the picture —
     measured: a tall picture (8×16) in a fit="cover" w="fill" ratio="4:3" cell makes the box
     **558×1116** instead of 558×418.5 (the picture's height:100%% hits a cycle and resolves to
     auto, and that height becomes the box's minimum height again), and the measured rect
     extends past the author's box and off the slide. The box must be the size the author gave. */
  .balpyo-image {
    display: flex;
    align-items: center;
    justify-content: center;
    flex: 0 0 auto;
    min-width: 0;
    min-height: 0;
  }

  /* display: block removes the space below the baseline of an inline <img> — that space exists
     only in the box, not in the picture, so leaving it would put a few unexplained px between
     the author's box and the picture. min-*: 0 turns off the automatic minimum size of flex
     items (the automatic minimum size would stop max-* from shrinking a picture larger than the
     box — and then contain would overflow). */
  .balpyo-image-ink {
    display: block;
    min-width: 0;
    min-height: 0;
  }

  /* Rotation and flip — the declaration lives here, the values are inline custom properties
     (imageRotateVar, imageFlipVar). The measure view must cancel this declaration so the
     measured rect is the **pre-rotation** box: measured, rotate(15deg) on a 200×100 picture
     makes getBoundingClientRect return 219.07×148.36. Flipping does not change the bbox, so
     turning it off too leaves the measured rect the same (on shapes the flip sits on the
     picture svg and the measure view leaves it alone — a picture is a single element, so it
     cannot be split there, and need not be). */
  .balpyo-image-transform {
    transform: rotate(var(%s, 0deg)) var(%s, scale(1));
  }
  html[%s=%q] .balpyo-image-transform {
    transform: none;
  }
`, imageRotateVar, imageFlipVar, ViewAttr, MeasureView)
}

// assetFiles는 덱의 에셋을 산출 파일로 만든다 — 경로는 `soml.Asset.Key` 그대로다.
//
// 표의 순서를 그대로 도는 이유는 결정성이다 (soml.Assets가 맵이 아닌 근거): 같은 덱을 두 번
// 렌더하면 같은 경로에 같은 바이트가 나온다. 바이트를 **복사하지 않고 그대로 넘긴다** —
// 엔진은 그림을 고치지 않는다 (리사이즈·재압축·EXIF 회전 없음 — soml.Asset.Bytes의 계약).
func assetFiles(deck *soml.Deck) (render.Files, error) {
	if len(deck.Assets) == 0 {
		return nil, nil
	}
	files := make(render.Files, len(deck.Assets))
	for _, asset := range deck.Assets {
		if asset.Key == "" {
			return nil, fmt.Errorf("the asset table has an entry without a key")
		}
		if asset.Key == IndexFile {
			// 저자가 `src="index.html"`을 쓸 수는 없지만(확장자 어휘) 산출 파일 이름이 바뀌면
			// 여기가 조용히 문서를 덮어쓴다 — 덮어쓴 산출물은 열리지도 않는다.
			return nil, fmt.Errorf("asset path %q collides with the output document", asset.Key)
		}
		if len(asset.Bytes) == 0 {
			return nil, fmt.Errorf("asset %q has no bytes", asset.Key)
		}
		files[asset.Key] = asset.Bytes
	}
	return files, nil
}
