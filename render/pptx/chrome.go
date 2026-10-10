package pptx

import (
	"fmt"
	"strings"

	"github.com/k2sebeom/balpyo/render/geom"
	"github.com/k2sebeom/balpyo/soml"
)

// 이 파일은 덱 크롬(페이지 번호·푸터)의 PPTX 조립이다.
//
// 두 결정이 이 파일의 내용 전부다.
//
// **플레이스홀더가 아니다.** 크롬은 `p:ph` 없는 보통 `p:sp`다. 흐름 모드 슬라이드는 레이아웃
// 파트 하나를 공유하므로(parts.go의 slideLayoutFile) `p:ph type="sldNum"`으로는 "표지에는
// 번호가 없다"를 표현할 수 없다 — 제외 규칙은 레이아웃 단위가 아니라 슬라이드 단위다. 마스터에
// `p:hf`도 내지 않는다: 상속할 것이 없고, 내면 크롬을 쓰지 않는 덱의 골든까지 움직인다.
//
// **자동 갱신을 주는 것은 `a:fld`다.** 필드는 텍스트 바디 안에 살므로 플레이스홀더가 아니어도
// 뷰어가 계산한다 — 장을 더하거나 순서를 바꾸면 번호가 따라온다. `a:t`에 값 층이 센 숫자를
// 캐시로 싣는 이유는 필드를 계산하지 않는 소비자(썸네일·변환기)도 맞는 숫자를 보여야 한다는
// 것이다 (두 값이 갈라질 수 있는 유일한 자리는 저자가 PowerPoint에서 장을 더한 뒤이고, 그때
// 뷰어가 계산한 값이 이긴다).

// chromeFieldID는 `a:fld@id`의 고정 GUID다.
//
// 고정인 이유는 결정성이다: 필드 id는 문서 안에서 필드를 구별하는 이름일 뿐이고 의미를 갖지
// 않으므로(ST_Guid), 매번 새로 만들면 같은 덱을 두 번 조립한 바이트가 달라져 골든 비교가
// 성립하지 않는다 (zipEpoch를 고정한 것과 같은 근거). 슬라이드마다 같은 값이어도 된다 —
// 필드는 자기 텍스트 바디 안에서만 식별된다.
const chromeFieldID = "{B9A1C0E4-6F3D-4A5B-8C7E-2D1F0A9B8C7D}"

// writeChromeShapes는 슬라이드의 확정된 크롬을 p:sp 목록으로 쓴다 — 콘텐츠 도형 **다음**이다.
//
// 뒤에 오는 이유는 spTree 순서가 z-순서라는 것이다: 크롬은 여백 띠에 살지만 캔버스 슬라이드의
// 전면 도형이 띠까지 덮을 수 있으므로, 뒤에 두면 그때도 번호가 보인다 (웹에서 크롬이 스테이지
// 뒤 형제인 것과 같은 그림).
//
// 도형 id는 콘텐츠와 **같은 카운터**를 쓴다 (id는 슬라이드 안에서 유일해야 한다).
func writeChromeShapes(b *strings.Builder, deck *soml.Deck, slide *soml.Slide, id *int, g *geom.Geometry, rels *slideRels) error {
	for _, item := range slide.Chrome {
		rect, ok := g.ChromeRect(slide.Index, item.Kind)
		if !ok {
			// 조용히 건너뛰면 웹에는 있고 PPTX에만 없는 크롬이 되고, 그 갈라짐은 두 산출물을
			// 나란히 볼 때까지 드러나지 않는다.
			return fmt.Errorf("%s: chrome %s has no measured coordinates — the measured HTML was not rendered from this deck",
				slide.Source, item.Kind)
		}
		if item.Kind == soml.ChromeLogo {
			if err := writeChromeLogo(b, deck, item, *id, rect, rels); err != nil {
				return fmt.Errorf("%s: chrome %s: %w", slide.Source, item.Kind, err)
			}
			*id++
			continue
		}
		body, err := chromeBodyXML(item, deck.Theme)
		if err != nil {
			return fmt.Errorf("%s: chrome %s: %w", slide.Source, item.Kind, err)
		}
		// 이름은 종류다 — PowerPoint 선택 창에서 저자가 "이 글자는 어디서 왔는가"를 묻는다.
		spec := textBoxSp(*id, "chrome "+string(item.Kind), rect, body)
		if err := writeSp(b, spec, deck.Theme); err != nil {
			return err
		}
		*id++
	}
	return nil
}

// chromeBodyXML은 크롬 하나의 p:txBody다.
//
// 텍스트 박스(textBodyXML)와 갈라지는 자리는 둘이다.
//
//   - **앵커가 ctr이다.** 잰 rect는 글자 높이가 아니라 띠 전체(StagePadding)이고, 웹은 그 안에서
//     `align-items: center`로 글줄을 가운데 둔다 (render/web.chromeCSS). 같은 규칙을 PPTX가
//     `anchor="ctr"`로 갖는다 — `t`로 두면 같은 rect에서 글자만 위로 붙어 두 타깃이 갈라진다.
//   - **wrap이 none이다.** 웹의 크롬은 한 줄이라는 계약이고(`white-space: nowrap`), rect는 글자
//     내용 폭이다. `square`로 두면 다른 엔진이 몇 px 넓게 잰 순간 줄이 접혀 띠를 넘치고, 그
//     넘침은 본문 위로 올라간다. `<pre>`가 wrap="none"인 것과 같은 규칙이다. 접히지 않으므로
//     autofit도 끈다 (`noAutofit`).
func chromeBodyXML(item soml.ChromeItem, theme soml.Theme) (string, error) {
	algn, err := algnValue(item.Align)
	if err != nil {
		return "", err
	}
	if err := checkTextStyle(item.Style, theme); err != nil {
		return "", err
	}

	var b strings.Builder
	w := func(depth int, line string) {
		b.WriteString("\n" + shapeIndent + strings.Repeat("  ", depth) + line)
	}
	w(1, "<p:txBody>")
	w(2, `<a:bodyPr wrap="none" lIns="0" tIns="0" rIns="0" bIns="0" anchor="ctr">`)
	w(3, "<a:noAutofit/>")
	w(2, "</a:bodyPr>")
	w(2, "<a:lstStyle/>")
	w(2, "<a:p>")
	w(3, fmt.Sprintf(`<a:pPr algn="%s">`, algn))
	w(4, "<a:lnSpc>")
	w(5, fmt.Sprintf(`<a:spcPts val="%d"/>`, LineSpacing(item.Style.LineHeight, item.Style.Size)))
	w(4, "</a:lnSpc>")
	w(3, "</a:pPr>")
	if err := writeChromeRun(w, item, theme); err != nil {
		return "", err
	}
	w(2, "</a:p>")
	w(1, "</p:txBody>")
	return b.String(), nil
}

// writeChromeRun은 크롬의 글자다 — 번호는 필드(`a:fld`), 푸터는 보통 런(`a:r`)이다.
//
// 두 요소의 내용 모형은 같다 (rPr + t): 그래서 서식을 한 번만 쓰고 감싸는 태그만 갈라진다.
// 푸터를 필드로 낼 수는 없다 — 설정의 문자열에 대응하는 필드 종류가 DrawingML에 없다.
func writeChromeRun(w func(depth int, line string), item soml.ChromeItem, theme soml.Theme) error {
	style := item.Style
	latin, ea, err := typefaces(style.Font, theme)
	if err != nil {
		return err
	}
	tag, open := "a:r", "<a:r>"
	if item.Kind == soml.ChromePageNumber {
		tag = "a:fld"
		open = fmt.Sprintf(`<a:fld id="%s" type="slidenum">`, chromeFieldID)
	}
	w(3, open)
	w(4, fmt.Sprintf(`<a:rPr sz="%d" b="%s">`, FontSize(style.Size), boldValue(style.Weight)))
	if err := writeTextFill(w, 5, style.Color, theme); err != nil {
		return err
	}
	w(5, latin)
	w(5, ea)
	w(4, "</a:rPr>")
	// 필드의 `a:t`는 **캐시**다 (뷰어가 계산하면 덮어쓴다). 보통 런의 `a:t`는 내용 자체다 —
	// 같은 태그가 두 뜻을 갖는 자리이고, 어느 쪽이든 값 층이 정한 문자열 하나다.
	w(4, fmt.Sprintf("<a:t>%s</a:t>", escapeXML(item.Text)))
	w(3, "</"+tag+">")
	return nil
}

// writeChromeLogo는 그림 크롬 하나를 `p:pic`으로 쓴다 — 좌표는 잰 그림 rect 그대로이고(웹의 `<img>`가
// 크롬 클래스를 가져 그림 자신이 재진다), 대체 텍스트가 `cNvPr@descr`이다. 그림 외관은 `<image fit="contain">`과
// 같다: 잰 rect가 이미 그림의 비율이라 크롭이 없다.
func writeChromeLogo(b *strings.Builder, deck *soml.Deck, item soml.ChromeItem, id int, rect geom.Rect, rels *slideRels) error {
	n, asset, ok := mediaOf(deck, item.Image)
	if !ok {
		return fmt.Errorf("%q is not in the asset table — the deck did not go through core.ResolveAssets", item.Image)
	}
	ext := asset.Format.Ext()
	if ext == "" {
		return fmt.Errorf("unknown file extension for format %q", string(asset.Format))
	}
	style := soml.ImageStyle{Src: item.Image, Fit: soml.FitContain, Alt: item.Alt}
	spec, err := picSpec(id, "chrome "+string(item.Kind), rect, style, asset, rels.mediaID(mediaTarget(n, ext)))
	if err != nil {
		return err
	}
	return writePic(b, spec, deck.Theme)
}
