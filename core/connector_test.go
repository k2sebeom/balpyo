package core

import (
	"strings"
	"testing"

	"github.com/k2sebeom/balpyo/soml"
)

// connectorDeck는 연결선 슬라이드를 해석한다. 파서 진단 중 "아직 그리지 못하는 어휘"만 거른다 —
// 렌더러가 뒤따르는 동안에도 값 층은 그 어휘를 끝까지 해석해야 한다.
func connectorDeck(t *testing.T, body string) (*soml.Deck, soml.Diagnostics) {
	t.Helper()
	doc, parseDiags := soml.Parse("slides/001_flow.slide", "<slide>\n"+body+"\n</slide>\n")
	for _, d := range parseDiags {
		if !strings.Contains(d.Message, soml.UnrenderedVocabMessage) {
			t.Fatalf("파서 진단: %v", d)
		}
	}
	deck := &soml.Deck{Slides: []*soml.Slide{{ID: "flow", Source: doc.File, Content: doc.Content, Tokens: doc.Tokens}}}
	return deck, Resolve(deck, nil)
}

// 연결선은 슬라이드를 다 읽은 뒤에 풀린다 — 뒤에서 선언한 상자도 가리킬 수 있다. 외관은 테마의
// connector.default 위에 variant, 그 위에 인라인이다.
func TestResolveConnector(t *testing.T) {
	deck, diags := connectorDeck(t, `  <connector from="a" to="b" route="elbow" from-side="bottom" variant="dashed" end="dot"/>
  <shape type="rect" id="a" w="100" h="60"/>
  <image src="assets/x.png" id="b" w="40" h="40"/>`)
	for _, d := range diags {
		// 그림 파일은 이 테스트의 관심이 아니다 (ResolveAssets가 보지 않는다).
		t.Fatalf("진단: %v", d)
	}
	conn := findElement(t, deck, soml.ConnectorElement).Appearance.Connector
	if conn == nil {
		t.Fatal("연결선이 확정되지 않았다")
	}
	if conn.From != findElement(t, deck, soml.ShapeElement) || conn.To != findElement(t, deck, soml.ImageElement) {
		t.Errorf("끝 = %v → %v", conn.From, conn.To)
	}
	if conn.Route != "elbow" || conn.FromSide != "bottom" || conn.ToSide != "" {
		t.Errorf("경로 = %s %q→%q", conn.Route, conn.FromSide, conn.ToSide)
	}
	if conn.Line.Style != soml.BorderDashed || conn.Line.Width != 2 || conn.Line.Color.Token != "muted" {
		t.Errorf("선 = %+v, want variant dashed", conn.Line)
	}
	if conn.Start != soml.EndNone || conn.End != soml.EndDot {
		t.Errorf("표식 = %s → %s, want none → dot (인라인이 테마의 arrow를 덮는다)", conn.Start, conn.End)
	}
}

func TestResolveConnectorDiagnostics(t *testing.T) {
	shapes := "\n  <shape type=\"rect\" id=\"a\"/>\n  <shape type=\"rect\" id=\"b\"/>"
	for _, tc := range []struct {
		name, connector, want string
	}{
		{"없는 이름", `<connector from="a" to="c"/>`, `<connector> to="c" is not the id of a shape or image on this slide (ids on this slide: a, b)`},
		{"오타", `<connector from="aa" to="b"/>`, `did you mean "a"?`},
		{"같은 상자", `<connector from="a" to="a"/>`, `from and to are the same element "a"`},
		{"from 없음", `<connector to="b"/>`, "<connector> needs from"},
		{"모르는 경로", `<connector from="a" to="b" route="zigzag"/>`, `unknown route "zigzag" (available values: straight, elbow, curved)`},
		{"모르는 면", `<connector from="a" to="b" to-side="north"/>`, `unknown to-side "north" (available values: auto, top, right, bottom, left)`},
		{"모르는 표식", `<connector from="a" to="b" end="diamond"/>`, `unknown end "diamond" (available values: none, arrow, dot)`},
		{"선 없음", `<connector from="a" to="b" line="none"/>`, "a connector line cannot be none"},
		{"그라디언트 선", `<connector from="a" to="b" line="2 gradient(accent1 accent2)"/>`, "a connector line is one color"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, diags := connectorDeck(t, "  "+tc.connector+shapes)
			found := false
			for _, d := range diags {
				found = found || strings.Contains(d.Message, tc.want)
			}
			if !found {
				t.Errorf("진단 = %v, want %q", diags, tc.want)
			}
		})
	}
}

// 테마가 선을 정하지 않았고 저자도 쓰지 않았으면 에러다 — 엔진에는 선의 기본값이 없다.
func TestResolveConnectorNeedsALine(t *testing.T) {
	deck, _ := connectorDeck(t, "  <connector from=\"a\" to=\"b\"/>\n  <shape type=\"rect\" id=\"a\"/>\n  <shape type=\"rect\" id=\"b\"/>")
	r := &resolver{theme: soml.Theme{Name: "bare"}}
	r.idElems = map[string]*soml.Element{"a": findElement(t, deck, soml.ShapeElement), "b": {Name: soml.ShapeElement}}
	r.connectors = []*soml.Element{findElement(t, deck, soml.ConnectorElement)}
	r.resolveConnectors()
	if len(r.diags) != 1 || !strings.Contains(r.diags[0].Message, `theme "bare" defines no connector.default line`) {
		t.Errorf("진단 = %v", r.diags)
	}
}

// 흐르는 선은 파선이다 — 실선 위에 flow를 쓰면 같은 굵기 · 색의 파선이 된다 (PPTX의 멈춘 모습도 같다).
func TestResolveConnectorFlow(t *testing.T) {
	deck, diags := connectorDeck(t, "  <connector from=\"a\" to=\"b\" flow/>\n  <shape type=\"rect\" id=\"a\"/>\n  <shape type=\"rect\" id=\"b\"/>")
	if len(diags) != 0 {
		t.Fatalf("진단: %v", diags)
	}
	conn := findElement(t, deck, soml.ConnectorElement).Appearance.Connector
	if !conn.Flow || conn.Line.Style != soml.BorderDashed {
		t.Errorf("흐르는 선 = flow %v · %s, want 파선", conn.Flow, conn.Line.Style)
	}
}
