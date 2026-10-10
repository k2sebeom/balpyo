package core

import (
	"strings"
	"testing"

	"github.com/k2sebeom/balpyo/soml"
)

// `alt`은 도형과 표 프레임에서 **그림·표의 이름**이다 — 안의 글을 대체하지 않는다.
// 값 층이 하는 일은 한 가지뿐이다: 빈 값의 뜻을 정하는 것.
func TestShapeAndTableAltResolve(t *testing.T) {
	deck, diags := resolveSlide(t, `<slide id="a11y">
  <shape type="arrow-right" alt="배포 흐름의 마지막 단계"/>
  <table alt="분기별 매출 표"><tr><th>분기</th></tr><tr><td>Q3</td></tr></table>
</slide>
`)
	if len(diags) != 0 {
		t.Fatalf("진단이 없어야 한다: %v", diags)
	}
	shape := findElement(t, deck, soml.ShapeElement)
	if got := shape.ShapeStyle().Alt; got != "배포 흐름의 마지막 단계" {
		t.Errorf("도형의 alt = %q, want %q", got, "배포 흐름의 마지막 단계")
	}
	table := findElement(t, deck, soml.TableElement)
	if got := table.TableStyle().Alt; got != "분기별 매출 표" {
		t.Errorf("표의 alt = %q, want %q", got, "분기별 매출 표")
	}
}

// 빈 `alt`은 **에러**다 — 도형의 그림은 속성이 없을 때 이미 장식이고(웹의 `aria-hidden`), 표의
// 내용은 장식이 될 수 없다. `wrap="false"`를 막은 것과 같은 기준이다: 기본값을 명시하는 표기를
// 열면 저자가 두 곳을 봐야 한다. 진단은 지우는 법을 말해야 한다.
func TestAltRejectsEmptyAndValuelessOnShapeAndTable(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{"도형의 빈 alt", `<shape type="rect" alt=""/>`},
		{"도형의 값 없는 alt", `<shape type="rect" alt/>`},
		{"표의 빈 alt", `<table alt=""><tr><td>x</td></tr></table>`},
		{"표의 값 없는 alt", `<table alt><tr><td>x</td></tr></table>`},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, diags := resolveSlide(t, "<slide id=\"a\">\n  "+tc.body+"\n</slide>\n")
			if len(diags) != 1 {
				t.Fatalf("진단 %d건, want 1건: %v", len(diags), diags)
			}
			if !strings.Contains(diags[0].Message, "alt") || !strings.Contains(diags[0].Message, "remove the attribute") {
				t.Errorf("진단 = %q, want alt과 지우는 법", diags[0].Message)
			}
		})
	}
}

// 그림의 `alt=""`는 **남는다** — 그 표기의 뜻이 HTML `<img>`에 있고 우리가 정한 것이 아니다.
// 비대칭의 근거가 우리 밖에 있다는 것을 이 테스트가 못 박는다.
func TestImageAltStillAcceptsTheEmptyString(t *testing.T) {
	if got := resolvedImage(t, `src="assets/a.png" alt=""`).Alt; got != "" {
		t.Errorf("그림의 alt = %q, want 빈 문자열 (장식 그림)", got)
	}
}
