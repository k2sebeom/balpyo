package pptx

import (
	"regexp"
	"strings"
	"testing"
)

// 하이라이팅은 **런 분할**로 도착한다: 값 층이 심은 합성 `<span>`을 이미 있는
// writeRuns가 `a:r` 여러 개로 낸다 — 렌더러는 손대지 않는다. 이 파일이 지키는 것은 그
// 도착 지점의 두 성질이다: 색은 `schemeClr` 참조이고(테마 색 편집이 코드에도 닿는다),
// 개행은 여전히 `a:br`이다(웹의 white-space: pre와 줄 수가 같아야 한다).

var textRE = regexp.MustCompile(`<a:t>([^<]*)</a:t>`)

// runText는 `a:t`를 이어 붙인 글자다 — a:br로 나간 개행은 여기 없으므로 줄 단위로 본다.
func runText(xml string) []string {
	var out []string
	for _, m := range textRE.FindAllStringSubmatch(xml, -1) {
		out = append(out, m[1])
	}
	return out
}

// 토큰 종류 경계마다 런이 끊기고, 색은 스키마 참조로 나간다.
func TestHighlightSplitsRunsWithSchemeColors(t *testing.T) {
	xml := singleShapeXML(t, `<slide><pre lang="go">func main() {
  n := 42 // 예열
}</pre></slide>`)

	if got := strings.Count(xml, "<a:r>"); got < 5 {
		t.Fatalf("런 %d개 — 종류 경계마다 끊기지 않았다\n%s", got, xml)
	}
	for _, want := range []string{
		`<a:schemeClr val="accent6"/>`, // keyword → accent6
		`<a:schemeClr val="accent1"/>`, // name → accent1
		`<a:schemeClr val="tx2"/>`,     // comment → muted(tx2)
	} {
		if !strings.Contains(xml, want) {
			t.Errorf("색 %q가 없다\n%s", want, xml)
		}
	}
	// 토큰 보존 계약: 슬라이드 파트에 리터럴 RGB가 있으면 PowerPoint의 테마 색 편집이
	// 코드 블록만 두고 간다.
	if strings.Contains(xml, "<a:srgbClr") {
		t.Errorf("코드 런이 리터럴 srgbClr로 나갔다\n%s", xml)
	}
	// 줄바꿈은 여전히 a:br다 — 런을 쪼갠 것이 문단을 쪼개지 않았다.
	if got := strings.Count(xml, "<a:br/>"); got != 2 {
		t.Errorf("a:br %d개, want 2개\n%s", got, xml)
	}
	// 글자는 하나도 잃지 않는다.
	if joined := strings.Join(runText(xml), ""); !strings.Contains(joined, "func main() {") ||
		!strings.Contains(joined, "// 예열") {
		t.Errorf("런을 이어 붙인 글자가 원문이 아니다: %q", joined)
	}
	// 인라인 런이므로 i/u까지 명시된다 (기존 인라인 계약과 같은 모양).
	if !strings.Contains(xml, `i="0" u="none"`) {
		t.Errorf("합성 런의 rPr가 완전 명시가 아니다\n%s", xml)
	}
}

// `lang`이 없는 `<pre>`는 런 하나짜리 옛 모양이다 — 켜지 않은 덱의 골든은 움직이지 않는다.
func TestHighlightWithoutLangKeepsOneRunPerLine(t *testing.T) {
	xml := singleShapeXML(t, "<slide><pre>func main() {\n  println(1)\n}</pre></slide>")
	if got := strings.Count(xml, "<a:r>"); got != 3 {
		t.Errorf("런 %d개, want 3개 (줄마다 하나)\n%s", got, xml)
	}
}
