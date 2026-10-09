package soml

import (
	"strconv"
	"strings"
)

// 이 파일은 **문자 참조**(`&lt;`·`&#x27E8;`)의 닫힌 어휘와 문자 데이터 해석이다.
//
// 모델에 담기는 텍스트는 **해석된 문자**다 — `&lt;`는 `<`가 되고, 산출물 경계에서 렌더러가
// 다시 한 번 이스케이프한다 (웹 `html.EscapeString`, PPTX `escapeXML`). 그래서 왕복이
// 닫힌다: `&lt;`(원문) → `<`(모델) → `&lt;`(HTML·`a:t`). 해석을 렌더러에 맡기면 두 타깃이
// 각자 참조 표를 들게 되고, 같은 원문이 타깃마다 다른 문자열이 된다 (공백 접기를 파서가
// 하는 것과 같은 근거 — soml.Parse의 접기 계약).

// charRefName은 이름 있는 참조 하나다. 표를 **순서 있는 슬라이스**로 두는 이유는 카탈로그와
// 같다: 순서가 곧 진단 메시지와 문서의 순서다.
type charRefName struct {
	name string
	r    rune
}

// charRefNames는 이름 있는 참조의 **전부**다 — XML 1.0의 5종이다.
//
// HTML의 명명 엔티티 2000여 종을 열지 않는 이유는 닫힌 어휘의 태도와 같다: `&nbsp;`·`&mdash;`는
// 유니코드 문자로 직접 쓰면 되고(SOML 파일은 UTF-8이다), 표가 커지면 저자가 "이 이름이
// 있는가"를 매번 확인해야 한다. 반면 `<`·`&`는 문법 문자라 참조 없이는 **쓸 방법이 없다** —
// 그 다섯 개와 수치 참조가 필요의 전부다.
var charRefNames = []charRefName{
	{"amp", '&'},
	{"apos", '\''},
	{"gt", '>'},
	{"lt", '<'},
	{"quot", '"'},
}

// charRefs는 이름 → 문자 색인이다 (위 표에서 파생된다).
var charRefs = buildCharRefs()

func buildCharRefs() map[string]rune {
	refs := make(map[string]rune, len(charRefNames))
	for _, ref := range charRefNames {
		refs[ref.name] = ref.r
	}
	return refs
}

// charRefList는 진단에 실을 참조 어휘 목록이다.
func charRefList() string {
	names := make([]string, len(charRefNames))
	for i, ref := range charRefNames {
		names[i] = "&" + ref.name + ";"
	}
	return strings.Join(names, ", ")
}

// decodeCharData는 문자 데이터 한 조각을 모델에 담길 문자열로 만든다.
//
// collapse가 참이면 ASCII 공백 런을 한 칸으로 접은 **뒤에** 참조를 푼다: 접기의 대상은
// 원문에 있는 공백이고, `&#32;`는 저자가 명시로 넣은 문자다. 반대 순서에서는
// 푼 공백이 원문 공백과 구별되지 않아 파서가 저자의 문자를 지운다.
//
// span은 raw가 원문에서 차지하는 구간이다 — report에 넘기는 구간은 Span.Slice로 잘라
// **참조 조각을 정확히** 가리킨다. report가 nil이면 진단 없이 해석만 한다.
func decodeCharData(raw string, span Span, collapse bool, report func(Span, string)) string {
	// 빠른 길: 풀 것도 접을 것도 없으면 원문 그대로다 (문자 데이터의 대부분이 이쪽이다).
	if !strings.Contains(raw, "&") && !(collapse && strings.ContainsAny(raw, asciiSpace)) {
		return raw
	}
	var b strings.Builder
	b.Grow(len(raw))
	space := false
	// 바이트 단위로 훑어도 안전하다: UTF-8 연속 바이트는 ASCII 범위에 들어오지 않는다
	// (collapseSpace와 같은 근거).
	for i := 0; i < len(raw); {
		c := raw[i]
		if collapse && strings.IndexByte(asciiSpace, c) >= 0 {
			space = true
			i++
			continue
		}
		if space {
			b.WriteByte(' ')
			space = false
		}
		if c == '&' {
			if r, n, ok := scanCharRef(raw[i:]); n > 0 {
				if ok {
					b.WriteRune(r)
				} else {
					// 모양은 참조인데 어휘 밖이다. 진단하고 원문을 그대로 남긴다 — 빌드는
					// 이미 멈추고, 저자가 화면에서 보는 것과 진단이 말하는 것이 같아진다.
					if report != nil {
						report(span.Slice(raw, i, i+n), raw[i:i+n])
					}
					b.WriteString(raw[i : i+n])
				}
				i += n
				continue
			}
		}
		b.WriteByte(c)
		i++
	}
	if space {
		b.WriteByte(' ')
	}
	return b.String()
}

// scanCharRef는 s의 앞에서 문자 참조를 읽는다.
//
// length는 **참조 모양의 바이트 수**다 (`;`까지 포함). 0이면 참조 모양이 아니므로 `&`는
// 리터럴이다 — `R&D`·`?a=1&b=2`가 저자가 쓴 대로 남는 자리다. length > 0이고 ok가 거짓이면
// 모양은 참조인데 어휘 밖이거나 값이 XML 문자가 아니다 (`&nbsp;`·`&#0;`) — 진단 대상이다.
//
// **`;`가 판정의 축이다**: `;`를 요구하면 참조를 의도한 것(`&nbsp;`)과 리터럴
// `&`를 정확히 가를 수 있다.
func scanCharRef(s string) (rune, int, bool) {
	// 가장 짧은 참조 모양은 `&#;`·`&a;`의 3바이트다.
	if len(s) < 3 || s[0] != '&' {
		return 0, 0, false
	}
	if s[1] == '#' {
		i, base, isDigit := 2, 10, isDecDigit
		if s[i] == 'x' || s[i] == 'X' {
			i, base, isDigit = i+1, 16, isHexDigit
		}
		start := i
		for i < len(s) && isDigit(s[i]) {
			i++
		}
		if i >= len(s) || s[i] != ';' {
			// `&#x` 뒤에 16진이 아닌 것이 오면 참조 모양이 아니다 — 리터럴 `&`다.
			return 0, 0, false
		}
		length := i + 1
		if start == i {
			return 0, length, false // `&#;` — 모양은 맞고 수치가 없다.
		}
		n, err := strconv.ParseUint(s[start:i], base, 32)
		if err != nil || !isXMLChar(rune(n)) {
			return 0, length, false
		}
		return rune(n), length, true
	}
	if !isNameStart(s[1]) {
		return 0, 0, false
	}
	i := 2
	for i < len(s) && isRefNameByte(s[i]) {
		i++
	}
	if i >= len(s) || s[i] != ';' {
		return 0, 0, false
	}
	if r, ok := charRefs[s[1:i]]; ok {
		return r, i + 1, true
	}
	return 0, i + 1, false
}

// isRefNameByte는 참조 이름의 두 번째 이후 바이트다 — 영숫자만이다. 태그 이름(isNameByte)이
// 받는 `-`·`_`를 받지 않는 이유는 참조 어휘가 5종으로 닫혀 있어 그 문자가 이름에 올 일이
// 없고, 넓게 훑으면 `&a-b;` 같은 산문이 참조 모양으로 오인되기 때문이다.
func isRefNameByte(c byte) bool {
	return isNameStart(c) || isDecDigit(c)
}

func isDecDigit(c byte) bool { return c >= '0' && c <= '9' }

func isHexDigit(c byte) bool {
	return isDecDigit(c) || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')
}

// isXMLChar는 XML 1.0의 Char 생성 규칙이다 — 수치 참조가 가리킬 수 있는 코드포인트.
//
// 이 검사를 여기서 하는 이유: 금지된 코드포인트(NUL·대부분의 제어 문자·서로게이트)를 통과
// 시키면 PPTX의 `a:t`에 그 문자가 실려 **PowerPoint가 파일을 열지 못한다**. 진단이 없으면
// 저자는 산출물을 열어 봐야 안다.
func isXMLChar(r rune) bool {
	switch {
	case r == 0x9, r == 0xA, r == 0xD:
		return true
	case r >= 0x20 && r <= 0xD7FF:
		return true
	case r >= 0xE000 && r <= 0xFFFD:
		return true
	case r >= 0x10000 && r <= 0x10FFFF:
		return true
	}
	return false
}

// cdataBody는 CDATA 토큰에서 구분자를 벗긴 내용이다. 닫히지 않은 토큰(진단이 이미 나갔다)은
// 여는 구분자만 벗긴다 — 진단이 있는 문서는 렌더에 쓰이지 않지만, 값이 반쪽으로 남으면
// lsp가 보는 트리가 원문과 어긋난다.
func cdataBody(text string) string {
	body := strings.TrimPrefix(text, cdataOpen)
	return strings.TrimSuffix(body, cdataClose)
}
