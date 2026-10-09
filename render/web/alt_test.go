package web

import (
	"strings"
	"testing"
)

// 이 파일은 **접근성 이름의 자리**를 못 박는다: `alt`은 그림에 이름을 **더하고**
// 안의 글을 대체하지 않는다. 회귀는 화면에 보이지 않고 스크린 리더만 달라진다.

// inkTagOf는 도형 박스 다음의 첫 잉크 `<svg` 여는 태그다 (svg에는 식별자가 없다).
func inkTagOf(t *testing.T, out, id string) string {
	t.Helper()
	box := openTagOf(t, out, id)
	at := strings.Index(out, box)
	rest := out[at+len(box):]
	start := strings.Index(rest, "<svg")
	if start < 0 {
		t.Fatalf("%s 뒤에 잉크 svg가 없다:\n%s", id, out)
	}
	end := strings.Index(rest[start:], ">")
	if end < 0 {
		t.Fatalf("잉크 svg의 여는 태그가 닫히지 않았다:\n%s", rest[start:])
	}
	return rest[start : start+end+1]
}

// `alt`은 **잉크 svg**에 이름을 붙인다 (`aria-hidden`이 적힌 자리). 박스 `<div>`는 역할이
// `generic`이라 ARIA가 이름을 금지하고, `role="img"`를 주면 안의 글이 AT에서 장식이 된다.
func TestShapeAltNamesTheInk(t *testing.T) {
	// 픽셀 공간(rect)과 정규 공간(path)은 svg를 쓰는 코드가 갈라져 있다.
	for _, shapeType := range []string{"rect", "arrow-right"} {
		t.Run(shapeType, func(t *testing.T) {
			out := renderString(t, subsetDeck(t, `<shape type="`+shapeType+
				`" alt="배포 흐름의 마지막 단계" w="200" h="100"><p>배포</p></shape>`))
			ink := inkTagOf(t, out, "s0.0")
			if !strings.Contains(ink, `role="img"`) {
				t.Errorf(`잉크 svg에 role="img"가 없다 — 이름만 붙은 그래픽은 역할이 없어 AT가 이름을 버린다: %s`, ink)
			}
			if !strings.Contains(ink, `aria-label="배포 흐름의 마지막 단계"`) {
				t.Errorf("잉크 svg에 저자의 이름이 없다: %s", ink)
			}
			if strings.Contains(ink, "aria-hidden") {
				t.Errorf("이름을 준 그림이 접근성 트리에서 감춰져 있다 — 저자가 쓴 이름이 읽히지 않는다: %s", ink)
			}
			// 박스는 한 글자도 달라지지 않는다.
			if box := openTagOf(t, out, "s0.0"); strings.Contains(box, "aria-") || strings.Contains(box, "role=") {
				t.Errorf("박스 div에 이름이 붙었다 — role=generic에 이름은 ARIA가 금지한다: %s", box)
			}
			// 안의 글은 **그대로 읽힌다** — `alt`은 더하는 이름이지 대체하는 이름이 아니다.
			if !strings.Contains(out, ">배포<") {
				t.Errorf("도형 안의 글이 사라졌다:\n%s", out)
			}
		})
	}
}

// `alt`이 없으면 도형 그림은 장식(aria-hidden)이고 표에는 이름이 없다.
func TestWithoutAltHasNoName(t *testing.T) {
	out := renderString(t, subsetDeck(t, `<shape type="rect" w="200" h="100"/><table><tr><td>Q3</td></tr></table>`))
	ink := inkTagOf(t, out, "s0.0")
	if !strings.Contains(ink, `aria-hidden="true"`) {
		t.Errorf("이름 없는 그림이 접근성 트리에 남았다: %s", ink)
	}
	if strings.Contains(ink, "aria-label") || strings.Contains(ink, "role=") {
		t.Errorf("이름이 없는데 역할·이름이 붙었다: %s", ink)
	}
	if tag := openTagOf(t, out, "s0.1"); strings.Contains(tag, "aria-") {
		t.Errorf("이름 없는 표에 aria 속성이 붙었다: %s", tag)
	}
}

// 표의 `alt`은 **표의 이름**이다 — `role="img"`는 셀 항해를 지우고, `<caption>`은 보이는
// 요소라 웹에만 줄이 생긴다.
func TestTableAltNamesTheTable(t *testing.T) {
	out := renderString(t, subsetDeck(t,
		`<table alt="분기별 매출 표"><tr><th>분기</th></tr><tr><td>Q3</td></tr></table>`))
	tag := openTagOf(t, out, "s0.0")
	if !strings.HasPrefix(tag, "<table") {
		t.Fatalf("식별자를 가진 요소가 표가 아니다: %s", tag)
	}
	if !strings.Contains(tag, `aria-label="분기별 매출 표"`) {
		t.Errorf("표에 이름이 없다: %s", tag)
	}
	if strings.Contains(tag, "role=") {
		t.Errorf(`표에 role이 붙었다 — role="img"는 셀 항해를 지운다: %s`, tag)
	}
	if strings.Contains(out, "<caption") {
		t.Errorf("표에 caption이 생겼다 — 보이는 줄이 웹에만 늘어 PPTX와 갈라진다:\n%s", out)
	}
	// 머리 셀과 본문 셀의 구별은 그대로다 (접근성의 본체는 이쪽이고 이름은 그 위다).
	if !strings.Contains(out, "<th") || !strings.Contains(out, "<td") {
		t.Errorf("표의 셀 구조가 달라졌다:\n%s", out)
	}
}

// 이름은 **이스케이프된다** — 속성 안에서 인용부호가 깨지면 DOM이 뒤틀린다 (`<img alt>`과 같은
// html.EscapeString을 지난다).
func TestAltIsEscapedInBothPlaces(t *testing.T) {
	out := renderString(t, subsetDeck(t,
		`<shape type="rect" alt="a &amp; b &lt; &quot;c&quot;" w="200" h="100"/>`+"\n"+
			`<table alt="a &amp; b &lt; &quot;c&quot;"><tr><td>x</td></tr></table>`))
	want := `a &amp; b &lt; &#34;c&#34;`
	if got := inkTagOf(t, out, "s0.0"); !strings.Contains(got, `aria-label="`+want+`"`) {
		t.Errorf("도형의 이름이 이스케이프되지 않았다: %s", got)
	}
	if got := openTagOf(t, out, "s0.1"); !strings.Contains(got, `aria-label="`+want+`"`) {
		t.Errorf("표의 이름이 이스케이프되지 않았다: %s", got)
	}
}
