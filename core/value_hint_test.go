package core

import (
	"strings"
	"testing"

	"github.com/k2sebeom/balpyo/soml"
)

// 값 없는 속성의 진단은 "적을 값"의 예시를 싣는다 (`the enter attribute needs a value
// (enter="rise")`). 예시가 빈 값이면 진단을 그대로 따른 저자가 곧바로 "비었다" 진단을 받는다 —
// 고치는 법을 말해야 할 자리가 다음 에러로 가는 길을 가리키는 셈이다. 그래서 카탈로그가 연
// **모든 태그의 모든 속성**을 값 없이 써 보고, 어느 진단도 빈 값을 권하지 않는지 본다. 새
// 속성이 들어오며 예시를 빠뜨리면 여기서 걸린다.
func TestValuelessAttrHintsNeverSuggestEmpty(t *testing.T) {
	type tagged struct{ name, attr, src string }
	var cases []tagged
	names := []string{soml.TextElement}
	for _, kind := range soml.ElementKinds() {
		names = append(names, soml.ElementNamesOf(kind)...)
	}
	for _, name := range names {
		for _, attr := range soml.ElementAttrs(name) {
			cases = append(cases, tagged{name, attr, valuelessAttrSlide(name, attr)})
		}
	}
	for _, attr := range soml.ElementAttrs(soml.RootElement) {
		cases = append(cases, tagged{soml.RootElement, attr, "<slide " + attr + ">\n  <p>x</p>\n</slide>\n"})
	}
	hints := 0
	for _, tc := range cases {
		doc, diags := soml.Parse("slides/001_hint.slide", tc.src)
		deck := &soml.Deck{Slides: []*soml.Slide{{
			ID: "hint", Source: doc.File, Content: doc.Content, Tokens: doc.Tokens,
		}}}
		for _, d := range append(diags, Resolve(deck, nil)...) {
			if !strings.Contains(d.Message, "the "+tc.attr+" attribute needs a value") {
				continue
			}
			hints++
			if strings.Contains(d.Message, tc.attr+`="")`) {
				t.Errorf("<%s %s>: 진단이 빈 값을 권한다: %s", tc.name, tc.attr, d.Message)
			}
		}
	}
	// 자리를 잘못 맞춰 진단이 하나도 나지 않으면 이 테스트는 초록인 채 아무것도 지키지 않는다.
	if hints < len(cases)/2 {
		t.Errorf("값 없음 진단 %d건 / 속성 %d개 — 슬라이드 원문이 시험할 진단에 닿지 않는다", hints, len(cases))
	}
}

// valuelessAttrSlide는 태그 하나를 그 태그가 놓일 수 있는 자리에 두고, 시험할 속성만 값 없이
// 붙인 슬라이드다. 자리를 맞추는 이유는 부모를 틀린 진단이 시험할 진단을 가리지 않게 하려는 것이다.
func valuelessAttrSlide(name, attr string) string {
	open := "<" + name + " " + attr
	var body string
	switch name {
	case soml.ListItemElement:
		body = "<ul>" + open + ">x</li></ul>"
	case "ul", "ol":
		body = open + "><li>x</li></" + name + ">"
	case soml.TableRowElement:
		body = "<table>" + open + "><td>x</td></tr></table>"
	case soml.TableHeaderCellElement, soml.TableDataCellElement:
		body = "<table><tr>" + open + ">x</" + name + "></tr></table>"
	case soml.TableElement:
		body = open + "><tr><td>x</td></tr></table>"
	case soml.ChartElement:
		body = open + `><point label="a" value="1"/></chart>`
	case soml.ChartPointElement:
		body = `<chart type="bar">` + open + "/></chart>"
	case soml.ChartSeriesElement:
		body = `<chart type="line">` + open + `><point label="a" value="1"/></series></chart>`
	case soml.ConnectorElement:
		ends := ""
		for _, end := range []string{"from", "to"} {
			if end != attr {
				ends += " " + end + `="` + map[string]string{"from": "a", "to": "b"}[end] + `"`
			}
		}
		body = `<shape type="rect" id="a"/><shape type="rect" id="b"/>` + open + ends + "/>"
	default:
		switch {
		case soml.IsVoidElement(name):
			body = "<row>" + open + "/></row>"
		case soml.KindOf(name) == soml.KindInline:
			body = "<p>" + open + ">x</" + name + "></p>"
		case name == soml.TextElement || soml.KindOf(name) == soml.KindBlock:
			body = open + ">x</" + name + ">"
		default:
			body = open + "><p>x</p></" + name + ">"
		}
	}
	return "<slide>\n  " + body + "\n</slide>\n"
}
