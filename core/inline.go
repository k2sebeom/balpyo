package core

import (
	"strconv"

	"github.com/k2sebeom/balpyo/soml"
)

// 이 파일은 인라인 태그(strong/em/code/a/span/br)의 런 스타일 해석이다.
//
// 인라인 태그는 자기 박스를 갖지 않는다 — 감싼 블록의 텍스트 흐름 안에서 런 서식이 된다.
// 상속은 **확정 스타일의 복사**다: 부모의 확정 TextStyle을 복사해 태그의 델타를 얹는다.
// 캐스케이드(cascade.go)는 블록에서 끝났고 인라인은 그 결과의 상속이다 — variant·역할이
// 인라인에 다시 적용되지 않는다.
//
// 이 규칙이 core 한 곳에만 있어야 렌더러의 런 분할이 "잎에서 가장 가까운 요소의 스타일
// 읽기"로 끝난다 — 상속 규칙이 렌더러마다 따로 있으면 웹과 PPTX의 런 서식이 조용히
// 갈라지고, 그 종류의 버그는 두 산출물을 나란히 볼 때까지 드러나지 않는다.

// boldWeight는 strong의 굵기다 (CSS 700). 더 굵은 단계는 없다 — OOXML의 굵기는
// 켬/끔(rPr@b)뿐이라 700 위의 단계는 웹에서만 동작한다.
const boldWeight = 700

// resolveInlines는 parent(텍스트 역할 요소 또는 인라인 요소)의 인라인 자식들에 런 스타일을
// 확정한다. parentStyle은 parent의 **확정** 스타일이다.
func (r *resolver) resolveInlines(parent *soml.Element, parentStyle *soml.TextStyle) {
	for _, child := range parent.Children {
		inline, ok := child.(*soml.Element)
		if !ok {
			continue
		}
		if inline.Name == soml.BreakElement {
			// <br>은 void다 — 텍스트 흐름의 분절점일 뿐 서식이 아니므로 Appearance를
			// 채우지 않는다. 렌더러의 런 분할이 이 이름을 특별 취급한다 (soml.BreakElement).
			continue
		}
		style := *parentStyle // 상속 = 확정 스타일의 복사. 태그의 델타는 아래가 얹는다.
		appearance := &soml.Appearance{Text: &style}
		switch inline.Name {
		case "strong":
			// 부모가 이미 700 이상이면 그대로 둔다 — 더 굵은 단계는 없다 (boldWeight).
			if style.Weight < boldWeight {
				style.Weight = boldWeight
			}
		case "em":
			// 중첩 em의 토글(HTML의 시각 관례)은 하지 않는다 — v1은 "강조는 이탤릭"으로
			// 닫는다. 토글은 "몇 겹인가"를 저자가 세어야 뜻이 정해지는 규칙이라, 컴포넌트
			// 조합으로 강조가 겹칠 때 결과를 예측할 수 없게 된다.
			style.Italic = true
		case "code":
			// size는 부모를 따른다 — code 역할(20px)을 끌어오면 한 줄 안에서 글자 크기가
			// 섞여 웹 줄 박스가 커지고 PPTX 절대 줄간격(spcPts)과 갈라진다. <span size>를
			// 파서가 막은 것과 같은 근거다 (soml.buildElementSpecs).
			style.Font = soml.FontMono
		case "a":
			style.Underline = true
			// 링크 색은 accent1 토큰을 재사용한다. PPTX의 hlinkClick은 뷰어가 테마
			// clrScheme의 hlink 슬롯 색을 입히는데, 테마 파트가 hlink 자리를 accent1에서
			// 빌려 채우므로 (render/pptx/theme.go의 clrSchemeSlots fallback) 웹이 같은
			// 토큰을 쓰면 두 산출물의 링크 색이 같은 표의 같은 값에서 나온다. hlink 전용
			// 토큰을 버린 이유: 표준 토큰 어휘에 없는 이름이 저자의 color= 어휘와 진단 목록·
			// CSS 변수 산출에 들어간다.
			style.Color = soml.TokenColor("accent1")
			if href, ok := r.href(inline); ok {
				appearance.Link = href
			}
		case "span":
			// span은 자기 인라인 속성(color·weight — 파서 allowlist)만 적용한다. 속성 없는
			// span은 부모 그대로다 — 해석 흔적만 남는 무해한 요소. 값 어휘와 진단은
			// <text color>와 같은 해석 함수를 공유한다 — 두 벌이 되면 진단이 갈라진다.
			for _, attr := range inline.Attrs {
				switch attr.Name {
				case "color":
					if color, ok := r.color(attr); ok {
						style.Color = color
					}
				case "weight":
					if weight, ok := r.weight(attr); ok {
						style.Weight = weight
					}
				}
			}
		}
		inline.Appearance = appearance
		r.resolveInlines(inline, &style)
	}
}

// href는 <a>의 목적지를 해석한다. 없는/빈 href는 에러다 — 목적지 없는 링크는 뜻을 정할 수
// 없다 (hasValue와 같은 태도이되, "지우면 기본값"이라는 안내가 성립하지 않아 문구를 따로
// 쓴다). 값의 URL 문법은 검증하지 않는다 — 파일·앵커·상대 경로 전부 유효할 수 있는 열린
// 값이고, 잘못된 URL은 클릭이 실패할 뿐 빌드가 막을 수 있는 실패가 아니다.
func (r *resolver) href(elem *soml.Element) (string, bool) {
	attr, ok := elem.Attr("href")
	switch {
	case !ok:
		r.errorf(elem.NameSpan, "<a> needs an href — a link without a destination has no meaning")
	case !attr.HasValue:
		r.errorf(attr.NameSpan, "the href attribute needs a value (href=%q)", attrExample("href"))
	case attr.Value == "":
		r.errorf(attr.ValueSpan, "href is empty — a link without a destination has no meaning")
	default:
		return attr.Value, true
	}
	return "", false
}

// weight는 <span weight>의 굵기를 해석한다. 어휘는 normal(400)·bold(700) 둘로 닫는다 —
// OOXML의 굵기는 켬/끔(rPr@b)뿐이라 CSS의 9단계 수치 축을 열면 중간 굵기가 웹에서만
// 동작하는 속성이 된다 (size의 %를 막은 것과 같은 종류의 근거).
func (r *resolver) weight(attr soml.Attr) (int, bool) {
	if !r.hasValue(attr) {
		return 0, false
	}
	raw := attr.Value
	switch raw {
	case "normal":
		return 400, true
	case "bold":
		return boldWeight, true
	}
	// 저자는 CSS 습관대로 수치를 쓴다 — 대응하는 이름을 짚어 준다 (align의 left/right와
	// 같은 태도: 고치는 방법이 진단 안에 있어야 한 번에 끝난다).
	if n, err := strconv.Atoi(raw); err == nil {
		switch n {
		case 400:
			r.errorf(attr.ValueSpan, "weight takes a name — instead of %q, use %q", raw, "normal")
		case boldWeight:
			r.errorf(attr.ValueSpan, "weight takes a name — instead of %q, use %q", raw, "bold")
		default:
			r.errorf(attr.ValueSpan, "weight %q is not allowed — there are two weights, normal (400) and bold (700) (OOXML bold is on/off, so intermediate weights would only work on the web)", raw)
		}
		return 0, false
	}
	r.errorf(attr.ValueSpan, "unknown weight %q%s (available values: normal, bold)", raw, didYouMean(raw, "normal, bold"))
	return 0, false
}
