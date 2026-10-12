package core

import (
	"strings"
	"testing"

	"github.com/k2sebeom/balpyo/soml"
)

// 수치 문법은 평범한 십진수 하나다 — Go의 수 문법(NaN·Inf·지수·16진·`+`·`_`)은 수가 아니다.
func TestParseNumberAcceptsOnlyPlainDecimals(t *testing.T) {
	good := map[string]float64{"0": 0, "12": 12, "-3": -3, "0.5": 0.5, ".5": 0.5, "-.25": -0.25, "007": 7, "1280.125": 1280.125}
	for raw, want := range good {
		got, ok := parseNumber(raw)
		if !ok || got != want {
			t.Errorf("parseNumber(%q) = %v, %v; want %v, true", raw, got, ok, want)
		}
	}
	bad := []string{"", "-", ".", "5.", "-.", "+5", "NaN", "nan", "Inf", "-Inf", "+Inf", "Infinity", "inf",
		"1_000", ".5e1", "1e3", "1E3", "0x1p4", "0x10", " 5", "5 ", "1.2.3", "--1", "1-", strings.Repeat("9", 400)}
	for _, raw := range bad {
		if got, ok := parseNumber(raw); ok {
			t.Errorf("parseNumber(%q) = %v, true; want not a number", raw, got)
		}
	}
}

func TestParseIntegerAcceptsOnlyPlainIntegers(t *testing.T) {
	good := map[string]int{"0": 0, "3": 3, "-2": -2, "010": 10}
	for raw, want := range good {
		got, ok := parseInteger(raw)
		if !ok || got != want {
			t.Errorf("parseInteger(%q) = %v, %v; want %v, true", raw, got, ok, want)
		}
	}
	for _, raw := range []string{"", "-", "+3", "1.5", "1_000", "1e3", "0x10", "NaN", "Inf", strings.Repeat("9", 40)} {
		if got, ok := parseInteger(raw); ok {
			t.Errorf("parseInteger(%q) = %v, true; want not an integer", raw, got)
		}
	}
}

// 수치를 받는 속성은 모두 같은 문법을 쓴다 — NaN·Inf·지수·`+`·`_`는 그 값을 짚는 에러 하나가 되고,
// 렌더러까지 가서 `NaNpx`나 스키마 밖의 정수가 되지 않는다.
func TestNumericAttributesRejectNonPlainNumbers(t *testing.T) {
	sites := []struct{ name, line string }{
		{"text size", `<text size="%s">내용</text>`},
		{"shape rotate", `<shape type="rect" w="100" h="100" rotate="%s"/>`},
		{"shape opacity", `<shape type="rect" w="100" h="100" opacity="%s"/>`},
		{"shape radius", `<shape type="rounded-rect" w="100" h="100" radius="%s"/>`},
		{"border width", `<shape type="rect" w="100" h="100" border="%s solid muted"/>`},
		{"notch", `<shape type="chevron" w="100" h="100" notch="%s%%"/>`},
		{"length w", `<row w="%s"><p>내용</p></row>`},
		{"length x", `<stack><p x="%s">내용</p></stack>`},
		{"gap", `<row gap="%s"><p>내용</p></row>`},
		{"pad", `<row pad="8 %s"><p>내용</p></row>`},
		{"track", `<grid cols="%s 1fr"><p>내용</p></grid>`},
		{"track percent", `<grid cols="%s%% 1fr"><p>내용</p></grid>`},
		{"order", `<row><p order="%s">내용</p></row>`},
		{"grow", `<row><p grow="%s">내용</p></row>`},
		{"ratio", `<stack ratio="%s:9"><p>내용</p></stack>`},
		{"reveal", `<ul><li reveal="%s">내용</li></ul>`},
		{"alpha", `<text color="alpha(fg, %s)">내용</text>`},
		{"shade", `<text color="shade(accent1, %s%%)">내용</text>`},
		{"gradient angle", `<shape type="rect" w="100" h="100" fill="gradient(accent1 accent2, %sdeg)"/>`},
		{"gradient stop", `<shape type="rect" w="100" h="100" fill="gradient(accent1 accent2/%s)"/>`},
		{"radial center", `<shape type="rect" w="100" h="100" fill="radial(accent1 accent2, at %s%% 50%%)"/>`},
		{"chart value", `<chart type="bar" h="300" alt="x"><point label="a" value="%s"/><point label="b" value="1"/></chart>`},
		{"chart max", `<chart type="bar" h="300" alt="x" max="%s"><point label="a" value="1"/></chart>`},
		{"line chart value", `<chart type="line" h="300" alt="x"><point label="a" value="%s"/><point label="b" value="1"/></chart>`},
	}
	values := []string{"NaN", "Inf", "-Inf", "Infinity", "1_000", "+5", ".5e1", "0x1p4"}
	for _, site := range sites {
		for _, value := range values {
			t.Run(site.name+"/"+value, func(t *testing.T) {
				line := strings.ReplaceAll(strings.ReplaceAll(site.line, "%s", value), "%%", "%")
				_, diags := resolveSlide(t, "<slide>\n  "+line+"\n</slide>\n")
				if len(diags) != 1 {
					t.Fatalf("진단 %d개, want 1개: %v", len(diags), diags)
				}
				diag := diags[0]
				if diag.Severity != soml.SeverityError || diag.Span.Start.Line != 2 {
					t.Errorf("진단 = %s %s %q, want 2행의 에러", diag.Severity, diag.Span.Start, diag.Message)
				}
				if !strings.Contains(diag.Message, value) {
					t.Errorf("진단 = %q, want 저자가 쓴 값 %q 포함", diag.Message, value)
				}
			})
		}
	}
}

// 이슈의 재현 그대로: 값 진단은 값의 원문을 담아 말한다.
func TestNonFiniteNumberDiagnosticWording(t *testing.T) {
	_, diag := oneDiag(t, chartSlide(`type="line" h="300"`, chartPoint(`label="a" value="NaN"`)+chartPoint(`label="b" value="1"`)))
	assertDiag(t, diag, 3, len(`    <point label="a" value="`)+1, `value is not a number ("NaN")`)
	_, diag = oneDiag(t, "<slide>\n  <text size=\"Inf\">hi</text>\n</slide>\n")
	assertDiag(t, diag, 2, 15, `size value is not a number ("Inf")`)
	_, diag = oneDiag(t, "<slide>\n  <shape type=\"rect\" w=\"100\" h=\"100\" rotate=\"NaN\"/>\n</slide>\n")
	assertDiag(t, diag, 0, 0, `rotate value is not a number ("NaN")`)
}
