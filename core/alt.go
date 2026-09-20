package core

import "github.com/k2sebeom/balpyo/soml"

// 이 파일은 도형·표의 `alt` 값 규칙이다 (ADR 0058) — 한 줄짜리 규칙이 두 요소에서 같아야
// 하므로 한 자리에 있다. 그림의 `alt`은 **다른 규칙**을 따르고 그 근거는 imageAlt에 있다.

// altName은 도형·표의 `alt`을 해석한다 — 검사할 값 문법이 없으므로 결정은 **빈 값의 뜻**
// 하나다.
//
// **빈 문자열을 받지 않는다.** 도형의 그림은 속성이 없을 때 이미 장식이고(웹 렌더러가 잉크
// `<svg>`에 `aria-hidden`을 붙인다) 표의 내용은 장식이 될 수 없으므로, `alt=""`은 "장식으로
// 해 달라"가 아니라 "이 속성을 지워 달라"의 어려운 표기다 — `wrap="false"`를 막은 것과 같은
// 기준이다 (기본값을 명시하는 표기를 열면 저자가 두 곳을 봐야 한다). 그래서 진단이 지우는
// 법을 말한다.
//
// 그림에서 `alt=""`가 살아 있는 이유는 그 표기의 뜻을 HTML `<img>`가 정의했기 때문이고
// (스크린 리더에게 "장식 이미지"다), 비대칭의 근거가 우리 밖에 있다는 것이 그 예외의 전부다.
func (r *resolver) altName(attr soml.Attr) (string, bool) {
	if !attr.HasValue {
		r.errorf(attr.NameSpan, "alt 속성에는 값이 필요하다 (alt=%q) — 이름이 필요 없으면 속성을 지운다",
			attrExample("alt"))
		return "", false
	}
	if attr.Value == "" {
		r.errorf(attr.ValueSpan, "alt은 비어 있을 수 없다 — 이름 없음이 기본값이므로 속성을 지운다 (그림의 alt=\"\"와 다르다: <img>에서 빈 alt은 \"장식 이미지\"라는 뜻이지만 도형·표에는 그 표기가 없다)")
		return "", false
	}
	return attr.Value, true
}
