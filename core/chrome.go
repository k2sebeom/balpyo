package core

import (
	"strconv"

	"github.com/k2sebeom/balpyo/soml"
)

// 이 파일은 덱 크롬(페이지 번호·푸터)의 값 층이다.
//
// 합치는 것은 셋이다: `balpyo.json`의 요청(내용·켜짐), 테마의 `chrome` 선언(자리·정렬·역할·
// 제외 규칙), 그리고 슬라이드의 확정 레이아웃(제외 판정의 입력). 결과는 슬라이드마다의
// []soml.ChromeItem 하나이고, **두 렌더러는 그 목록만 읽는다** — 제외 규칙도 역할 →
// 외관 캐스케이드도 번호 문자열도 여기서 끝난다. 그 조건을 렌더러마다 적었을 때 웹과 PPTX가
// 같은 소스에서 서로 다른 입력을 보고 갈라진다.

// resolveChrome은 덱의 크롬 요청과 테마 선언을 합쳐 **덱 수준의 계획**을 만든다 (선언 순서).
//
// 테마가 요청받은 kind를 선언하지 않았으면 경고 + 미탑재다 — 슬롯과 같은 규칙이고
// 진단이 슬라이드가 아니라 `balpyo.json`을 가리키는 이유는 고칠 자리가 거기라는 것이다.
// 경고는 **덱에 한 번** 낸다: 슬라이드마다 내면 12장 덱에서 같은 문장이 12번 나온다.
func resolveChrome(deck *soml.Deck, pkg themePackage) ([]soml.ChromeItem, soml.Diagnostics) {
	if deck == nil {
		return nil, nil
	}
	var (
		items []soml.ChromeItem
		diags soml.Diagnostics
	)
	// 순서는 **테마 선언 순서**가 아니라 어휘 순서다 (footer → pageNumber): 경고가 실행마다
	// 같은 순서로 나와야 하고, 테마가 선언 순서를 바꿨다고 산출물의 도형 순서가 흔들릴 이유가
	// 없다. 자리는 align이 정하므로 순서는 겹침에 영향을 주지 않는다.
	for _, kind := range soml.ChromeKinds() {
		if kind == soml.ChromeLogo {
			// 로고는 요청이 아니라 테마의 선언이 켠다 — 그림도 자리도 테마의 것이다. 선언하지 않은 테마는
			// 로고가 없을 뿐이다 (경고할 요청이 없다).
			if decl, ok := pkg.chromeDeclFor(kind); ok {
				items = append(items, soml.ChromeItem{
					Kind: kind, Align: decl.align, Image: decl.image, Height: decl.height, Alt: decl.alt,
				})
			}
			continue
		}
		text, on := chromeRequest(deck.Chrome, kind)
		if !on {
			continue
		}
		decl, ok := pkg.chromeDeclFor(kind)
		if !ok {
			diags = append(diags, soml.Warningf(ConfigFile, soml.Span{},
				"theme %q has no %s chrome declaration, so %s cannot be used (chrome in theme.json) — this deck renders without a %s",
				pkg.manifest.Name, kind, kind, chromeNoun(kind)))
			continue
		}
		// 역할이 가져온 **정렬은 선언이 덮는다**: 그러지 않으면 확정 모델 하나에 정렬이 둘이
		// 되고(선언의 align과 역할의 Align), 웹은 외관 쪽을 `text-align`으로 읽고 PPTX는 선언
		// 쪽을 `algn`으로 읽어 같은 입력에서 두 산출물이 갈라진다. 한 줄 박스라 지금은
		// 눈에 보이지 않을 뿐이고, 보이지 않는 갈라짐도 두 타깃의 불일치다.
		style := chromeStyle(pkg, decl.role)
		style.Align = decl.align
		items = append(items, soml.ChromeItem{
			Kind:  kind,
			Text:  text,
			Align: decl.align,
			Style: style,
		})
	}
	return items, diags
}

// slideChrome은 덱 계획에서 이 슬라이드에 실릴 항목을 고른다 — 제외 규칙을 적용하고
// 페이지 번호의 글자를 채운다.
//
// 번호는 **덱 안의 위치**다 (1-based, 제외된 장도 센다): PPTX의 `a:fld type="slidenum"`이
// 주는 값이 그것이라, 표지를 세지 않는 번호를 웹에 찍으면 두 타깃이 갈린다.
func slideChrome(plan []soml.ChromeItem, pkg themePackage, slide *soml.Slide) []soml.ChromeItem {
	if len(plan) == 0 {
		return nil
	}
	var items []soml.ChromeItem
	for _, item := range plan {
		decl, ok := pkg.chromeDeclFor(item.Kind)
		if ok && decl.skip[slide.Layout] {
			continue
		}
		if item.Kind == soml.ChromePageNumber {
			item.Text = strconv.Itoa(slide.Index + 1)
		}
		items = append(items, item)
	}
	return items
}

// chromeRequest는 kind의 요청 상태다 — 글자와 켜짐. 푸터는 글자가 곧 켜짐이고(빈 문자열 =
// 꺼짐), 번호의 글자는 슬라이드가 정하므로 요청은 불리언 하나다.
func chromeRequest(req soml.DeckChrome, kind soml.ChromeKind) (string, bool) {
	switch kind {
	case soml.ChromeFooter:
		return req.Footer, req.Footer != ""
	case soml.ChromePageNumber:
		return "", req.PageNumber
	}
	return "", false
}

// chromeStyle은 역할에서 확정된 텍스트 외관이다 — 로드된 컴포넌트 표에서 읽는다
// (resolver.textRoleDefaults와 같은 규칙이고, 폴백이 없는 것도 같다 — 빠진 값은 로드 시점에 잡힌다).
func chromeStyle(pkg themePackage, role soml.TextRole) soml.TextStyle {
	return pkg.components.textRoles[role]
}

// chromeNoun은 경고 문구에서 쓰는 우리말 이름이다 — 저자가 읽는 문장에 `pageNumber`가 두 번
// 나오면 그 문장은 설정 키를 말하는 것인지 결과를 말하는 것인지 모르게 된다.
func chromeNoun(kind soml.ChromeKind) string {
	if kind == soml.ChromePageNumber {
		return "page number"
	}
	return "footer"
}
