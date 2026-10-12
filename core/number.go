package core

import (
	"math"
	"strconv"
)

// parseNumber는 저자가 속성에 쓴 수치 하나를 읽는다 — 속성 값의 수치는 모두 이 함수를 지난다.
// (경로 `d`는 SVG 경로 문법이라 path.go가 따로 읽는다. 그 토큰은 글자에서 끊기므로 NaN·Inf가
// 수로 들어올 길이 없다. 그림으로 읽는 SVG 파일의 크기는 저자가 아니라 내보낸 도구가 쓴 값이라
// SVG의 수 문법을 따르는 svgNumber가 읽는다.)
//
// 받는 형태는 평범한 십진수 하나다: 선택적인 `-`, 숫자, 선택적인 소수부 (`12`, `-3`, `0.5`, `.5`).
// strconv.ParseFloat를 그대로 쓰지 않는 이유는 그것이 Go의 수 문법을 읽는다는 것이다 —
// `NaN`·`Inf`·`Infinity`·`1e3`·`0x1p4`·`+5`도 수가 된다. 무한과 NaN은 CSS에 `NaNpx`로,
// OOXML에 스키마 밖의 정수로 그대로 새어 나가고, 지수·16진·`+`는 두 타깃이 같은 수로 읽는다는
// 보장이 없는 표기다. 그래서 문법을 여기서 닫고, 넘쳐서 무한이 되는 값(`1` 뒤에 0 400개)도 버린다.
func parseNumber(raw string) (float64, bool) {
	if !isPlainDecimal(raw) {
		return 0, false
	}
	n, err := strconv.ParseFloat(raw, 64)
	if err != nil || math.IsInf(n, 0) || math.IsNaN(n) {
		return 0, false
	}
	return n, true
}

// parseInteger는 정수 하나를 읽는다 — parseNumber와 같은 문법에서 소수부만 뺀 것이다
// (`-`는 받고 `+`는 받지 않는다). int를 넘는 값도 수가 아니다.
func parseInteger(raw string) (int, bool) {
	digits := raw
	if len(digits) > 0 && digits[0] == '-' {
		digits = digits[1:]
	}
	if !isDigits(digits) {
		return 0, false
	}
	n, err := strconv.Atoi(raw)
	if err != nil {
		return 0, false
	}
	return n, true
}

// isPlainDecimal은 값이 `-?(숫자+(.숫자+)?|.숫자+)` 꼴인지다.
func isPlainDecimal(raw string) bool {
	if len(raw) > 0 && raw[0] == '-' {
		raw = raw[1:]
	}
	intPart, frac, hasDot := raw, "", false
	for i := 0; i < len(raw); i++ {
		if raw[i] == '.' {
			intPart, frac, hasDot = raw[:i], raw[i+1:], true
			break
		}
	}
	if !hasDot {
		return isDigits(intPart)
	}
	return (intPart == "" || isDigits(intPart)) && isDigits(frac)
}
