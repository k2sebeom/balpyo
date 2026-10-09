package core

import (
	"strings"

	"github.com/k2sebeom/balpyo/soml"
)

// 이 파일은 `style` **배치 CSS 탈출구**다 (PRD §7.9).
//
// 계약은 "우리가 보지 않는다"이지 "안전하다"가 아니다: 선언을 `프로퍼티: 값`으로 쪼개
// 정규화하고 프로퍼티 이름만 allowlist와 대조한다. **값은 검증하지 않는다** — 엔진이 CSS 값
// 문법을 재구현하기 시작하면 탈출구가 탈출구이기를 멈춘다 (그러면 그것은 그냥 또 하나의
// 속성 어휘다). 이 사실은 문서에도 남아야 한다 (항목 06).
//
// allowlist 밖의 프로퍼티는 **에러가 아니라 경고**다. 에러로 두려면 엔진이 CSS 프로퍼티
// 전체 목록을 알아야 하고, 그것은 탈출구의 존재 이유(엔진이 아직 모르는 배치를 저자가
// 통과시킨다)와 정면으로 모순된다. 경고는 모르는 variant 경고와 같은 경로를 탄다 —
// 기본 빌드를 통과시키고 `--strict`가 에러로 승격한다 (PRD §13-6, soml.Diagnostics.Strict).

// layoutStyleProperties는 **배치 CSS 프로퍼티 allowlist**다. 순서 있는 한 곳에 두고 문서가
// 같은 목록을 인용한다 (항목 06) — 목록이 두 벌이 되면 경고 조건과 문서가 갈라진다.
//
// `*`로 끝나는 항목은 접두 가족이다 (`flex-*`는 flex-grow·flex-basis·flex-direction …).
// 가족으로 적는 이유: CSS는 축약과 개별 프로퍼티가 짝을 이루고(예: `padding`과
// `padding-inline-start`), 개별 이름을 전수로 적으면 목록이 CSS 명세를 베끼기 시작한다.
//
// 목록의 기준은 하나다: **지오메트리 패스가 재는 박스를 움직이는 프로퍼티**. 배치는
// 브라우저가 잰 좌표로 구워져 PPTX에 남으므로(PRD §9.3) 이 프로퍼티들은 양 타깃에 도달한다.
// 색·테두리·그림자는 CSS에만 남는다 — 그것이 경고의 이유다.
//
// `translate`가 목록에 있고 `rotate`/`scale`은 없는 것이 이 기준의 좋은 시험이다:
// getBoundingClientRect는 변환된 박스를 재므로 이동은 좌표에 그대로 반영되지만, 회전·확대는
// 축에 정렬된 외접 박스만 남아 PPTX에서 다른 도형이 된다.
func layoutStyleProperties() []string {
	return []string{
		"display",
		"flex", "flex-*",
		"grid", "grid-*",
		"gap", "row-gap", "column-gap",
		"padding", "padding-*",
		"margin", "margin-*",
		"width", "min-width", "max-width",
		"height", "min-height", "max-height",
		"box-sizing",
		"align-*", "justify-*", "place-*",
		"order",
		"aspect-ratio",
		"position", "inset", "inset-*", "top", "right", "bottom", "left",
		"translate",
	}
}

// isLayoutStyleProperty는 프로퍼티가 배치 allowlist에 있는지다. 이름은 소문자로 정규화된
// 상태로 들어온다 (CSS 프로퍼티 이름은 대소문자를 구별하지 않는다).
func isLayoutStyleProperty(property string) bool {
	for _, pattern := range layoutStyleProperties() {
		if prefix, family := strings.CutSuffix(pattern, "*"); family {
			if strings.HasPrefix(property, prefix) {
				return true
			}
			continue
		}
		if property == pattern {
			return true
		}
	}
	return false
}

// placementStyle은 `style` 속성을 확정된 선언 문자열로 정규화한다.
//
// 정규화 규칙: `;`로 나누고 첫 `:`로 쪼개, 프로퍼티 이름은 소문자로 트림하고 값은 트림만
// 한다. 결과는 `프로퍼티: 값`을 `; `로 이은 문자열이다 — 공백과 마지막 `;`의 유무가 산출물을
// 흔들지 않아야 골든이 무엇이 변했는지 말해 준다.
//
// 문법 위반이 하나라도 있으면 **속성 전체를 싣지 않는다**: 값 층의 다른 속성과 같은 정책이고
// (틀린 값은 확정 배치에 실리지 않는다), 반쯤 통과한 선언 목록은 저자가 고친 뒤의 결과를
// 예측할 수 없게 만든다.
func (r *resolver) placementStyle(attr soml.Attr) (string, bool) {
	if !r.hasValue(attr) {
		return "", false
	}
	parts := splitDeclarations(attr.Value)
	if len(parts) == 0 {
		r.errorf(attr.ValueSpan, "style에 선언이 없다 (%q) — `속성: 값` 형태로 쓴다", attr.Value)
		return "", false
	}
	decls := make([]string, 0, len(parts))
	valid := true
	for _, part := range parts {
		name, value, found := strings.Cut(part.text, ":")
		if !found {
			r.errorf(part.span(attr), "style 선언 %q에 콜론이 없다 — `속성: 값` 형태로 쓴다", part.text)
			valid = false
			continue
		}
		property := strings.ToLower(strings.TrimSpace(name))
		declaration := strings.TrimSpace(value)
		switch {
		case property == "":
			r.errorf(part.span(attr), "style 선언 %q에 속성 이름이 없다", part.text)
			valid = false
			continue
		case declaration == "":
			// 빈 값은 "검증하지 않는 값"이 아니라 완성되지 않은 선언이다 — CSS도 버린다.
			r.errorf(part.span(attr), "style 선언 %q에 값이 없다", part.text)
			valid = false
			continue
		}
		if !isLayoutStyleProperty(property) {
			r.warnf(part.span(attr), "style의 %q 선언은 배치가 아니다 — 웹에만 적용되고 PPTX에는 이식되지 않는다. 색·테두리 같은 외관은 컴포넌트 속성으로 쓴다",
				property)
		}
		decls = append(decls, property+": "+declaration)
	}
	if !valid {
		return "", false
	}
	return strings.Join(decls, "; "), true
}

// stripCSSComments는 `/* … */` 블록 주석을 공백 하나로 지운다. 레이아웃 CSS 파일은 프로젝트
// 관례대로 주석을 다는데(빈 스텁조차 주석 하나다), 선언 분해는 주석을 모르므로 주석이 선언으로
// 새면 "콜론이 없다" 같은 거짓 진단이 난다. 여기서 먼저 벗겨 배치 린터와 정규화가 선언만 본다.
// 짝이 맞지 않는 `/*`는 문자열 끝까지를 주석으로 본다 (CSS 파서와 같은 관용).
func stripCSSComments(raw string) string {
	var b strings.Builder
	b.Grow(len(raw))
	for {
		open := strings.Index(raw, "/*")
		if open < 0 {
			b.WriteString(raw)
			break
		}
		b.WriteString(raw[:open])
		b.WriteByte(' ') // 주석은 토큰 경계다 — `a/* */b`가 `ab`로 붙지 않게 공백으로 바꾼다.
		rest := raw[open+2:]
		if shut := strings.Index(rest, "*/"); shut >= 0 {
			raw = rest[shut+2:]
			continue
		}
		break // 닫히지 않은 주석: 나머지 전부가 주석이다.
	}
	return b.String()
}

// normalizeDeclarations는 CSS 선언 원문을 `프로퍼티: 값`을 `; `로 이은 결정적 문자열로 눕힌다.
// 주석을 벗기고, 문법이 온전한 선언(콜론·이름·값이 다 있음)만 싣는다 — 문법 위반은 lintLayoutCSS가
// 에러로 잡으므로 여기서는 조용히 건너뛴다 (린트와 정규화가 같은 분해기를 공유해 갈라지지 않는다).
func normalizeDeclarations(raw string) string {
	var decls []string
	for _, part := range splitDeclarations(stripCSSComments(raw)) {
		name, value, found := strings.Cut(part.text, ":")
		property := strings.ToLower(strings.TrimSpace(name))
		declaration := strings.TrimSpace(value)
		if !found || property == "" || declaration == "" {
			continue
		}
		decls = append(decls, property+": "+declaration)
	}
	return strings.Join(decls, "; ")
}

// splitDeclarations는 `;`로 선언을 나눈다. 빈 조각(`a: 1;;`, 마지막 `;`)은 버린다 — CSS도
// 그렇게 읽고, 마지막 `;`는 저자의 관용 표기라 그것 때문에 진단이 나면 안 된다.
func splitDeclarations(raw string) []valuePart {
	var parts []valuePart
	start := 0
	for i := 0; i <= len(raw); i++ {
		if i < len(raw) && raw[i] != ';' {
			continue
		}
		if part, ok := trimmedPart(raw, start, i); ok {
			parts = append(parts, part)
		}
		start = i + 1
	}
	return parts
}

// trimmedPart는 raw[start:end)의 앞뒤 공백을 벗긴 조각이다. 오프셋을 함께 남기는 이유는
// 진단이 선언 하나를 가리켜야 하기 때문이다 (valuePart.span).
func trimmedPart(raw string, start, end int) (valuePart, bool) {
	for start < end && isASCIISpace(raw[start]) {
		start++
	}
	for end > start && isASCIISpace(raw[end-1]) {
		end--
	}
	if start == end {
		return valuePart{}, false
	}
	return valuePart{text: raw[start:end], start: start}, true
}
