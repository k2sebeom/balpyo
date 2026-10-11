package core

import (
	"strings"
	"testing"

	"github.com/k2sebeom/balpyo/soml"
)

func enterDeck(t *testing.T, slideAttrs, body string) (*soml.Deck, soml.Diagnostics) {
	t.Helper()
	src := "<slide" + slideAttrs + ">\n" + body + "\n</slide>\n"
	doc, parseDiags := soml.Parse("slides/001_enter.slide", src)
	if err := parseDiags.Err(); err != nil {
		t.Fatalf("파서 진단: %v", err)
	}
	deck := &soml.Deck{Slides: []*soml.Slide{{ID: "enter", Source: doc.File, Content: doc.Content,
		Tokens: doc.Tokens, Enter: doc.Enter, EnterSpan: doc.EnterSpan}}}
	return deck, Resolve(deck, nil)
}

// `<slide enter>`는 직속 자식마다 등장을 주고, 자식의 enter가 이긴다(none은 끈다). 순서는 문서 순서이고
// 지연은 테마의 박자에서 나온다 (minimal: 150 + 90·순서, 길이 600).
func TestResolveEntrances(t *testing.T) {
	deck, diags := enterDeck(t, ` enter="rise"`, `  <h2>title</h2>
  <row><shape type="rect" enter="zoom"/><shape type="rect"/></row>
  <p enter="none">quiet</p>
  <p enter="fade">last</p>`)
	if len(diags) != 0 {
		t.Fatalf("진단: %v", diags)
	}
	content := deck.Slides[0].Content
	var elems []*soml.Element
	for _, n := range content {
		if e, ok := n.(*soml.Element); ok {
			elems = append(elems, e)
		}
	}
	row := elems[1]
	var shapes []*soml.Element
	for _, n := range row.Children {
		if e, ok := n.(*soml.Element); ok {
			shapes = append(shapes, e)
		}
	}
	for _, tc := range []struct {
		name string
		elem *soml.Element
		want *soml.Entrance
	}{
		{"제목 (슬라이드 기본값)", elems[0], &soml.Entrance{Effect: soml.EnterRise, Order: 0, Delay: 150, Duration: 600}},
		{"행 (슬라이드 기본값)", row, &soml.Entrance{Effect: soml.EnterRise, Order: 1, Delay: 240, Duration: 600}},
		{"행 안의 자기 등장", shapes[0], &soml.Entrance{Effect: soml.EnterZoom, Order: 2, Delay: 330, Duration: 600}},
		{"행 안의 나머지 (행을 따른다)", shapes[1], nil},
		{"none", elems[2], nil},
		{"자식의 enter가 이긴다", elems[3], &soml.Entrance{Effect: soml.EnterFade, Order: 3, Delay: 420, Duration: 600}},
	} {
		got := tc.elem.EnterOf()
		if (got == nil) != (tc.want == nil) || (got != nil && *got != *tc.want) {
			t.Errorf("%s: 등장 = %+v, want %+v", tc.name, got, tc.want)
		}
	}
}

func TestResolveEntranceDiagnostics(t *testing.T) {
	_, diags := enterDeck(t, ` enter="slide-in"`, `  <p enter="spin">x</p>`)
	var msgs []string
	for _, d := range diags {
		msgs = append(msgs, d.Message)
	}
	all := strings.Join(msgs, "\n")
	for _, want := range []string{
		`unknown enter "slide-in" (available values: fade, rise, zoom, wipe, none)`,
		`unknown enter "spin"`,
	} {
		if !strings.Contains(all, want) {
			t.Errorf("진단 = %v, want %q", msgs, want)
		}
	}

	deck, _ := enterDeck(t, "", `  <p enter="rise">x</p>`)
	r := &resolver{theme: soml.Theme{Name: "still"}}
	r.resolveEntrances(deck.Slides[0])
	if len(r.diags) != 1 || !strings.Contains(r.diags[0].Message, `theme "still" defines no motion`) {
		t.Errorf("박자 없는 테마의 진단 = %v", r.diags)
	}
}

// 슬라이드의 transition이 테마의 기본값을 덮어쓰고(none은 끈다), 길이는 테마의 transitionDuration이다.
// 테마가 기본 전환을 정하지 않으면(내장 테마) 슬라이드가 고른 장만 전환한다.
func TestResolveTransition(t *testing.T) {
	for _, tc := range []struct {
		name, attr string
		theme      *soml.MotionTokens
		want       soml.SlideTransition
		ms         int
	}{
		{"테마에 기본 전환이 없다", "", &soml.MotionTokens{Duration: 600, TransitionDuration: 600}, "", 0},
		{"슬라이드가 고른다", "push", &soml.MotionTokens{Duration: 600, TransitionDuration: 600}, soml.TransitionPush, 600},
		{"테마의 기본 전환", "", &soml.MotionTokens{Transition: soml.TransitionFade, TransitionDuration: 400}, soml.TransitionFade, 400},
		{"none이 끈다", "none", &soml.MotionTokens{Transition: soml.TransitionFade, TransitionDuration: 400}, "", 0},
		{"박자 없는 테마에서도 산다", "fade", nil, soml.TransitionFade, 500},
	} {
		t.Run(tc.name, func(t *testing.T) {
			slide := &soml.Slide{Transition: soml.SlideTransition(tc.attr)}
			r := &resolver{theme: soml.Theme{Name: "t", Motion: tc.theme}}
			r.resolveTransition(slide)
			if len(r.diags) != 0 || slide.Transition != tc.want || slide.TransitionDuration != tc.ms {
				t.Errorf("전환 = %q %dms (진단 %v), want %q %dms", slide.Transition, slide.TransitionDuration, r.diags, tc.want, tc.ms)
			}
		})
	}
	r := &resolver{theme: soml.Theme{Name: "t"}}
	r.resolveTransition(&soml.Slide{Transition: "cube"})
	if len(r.diags) != 1 || !strings.Contains(r.diags[0].Message, `unknown transition "cube" (available values: fade, push, none)`) {
		t.Errorf("진단 = %v", r.diags)
	}
}

// 연결선은 자기 enter만 받는다 — `<slide enter>`의 상자 효과는 닿지 않고, 어휘는 draw · fade · none이다.
func TestResolveConnectorEntrance(t *testing.T) {
	deck, diags := enterDeck(t, ` enter="rise"`, `  <shape type="rect" id="a"/>
  <shape type="rect" id="b"/>
  <connector from="a" to="b" enter="draw"/>
  <connector from="b" to="a"/>`)
	if len(diags) != 0 {
		t.Fatalf("진단: %v", diags)
	}
	var conns []*soml.Element
	for _, n := range deck.Slides[0].Content {
		if e, ok := n.(*soml.Element); ok && e.Name == soml.ConnectorElement {
			conns = append(conns, e)
		}
	}
	if e := conns[0].EnterOf(); e == nil || e.Effect != soml.EnterDraw || e.Order != 2 {
		t.Errorf("그리는 연결선의 등장 = %+v, want draw · 순서 2", e)
	}
	if e := conns[1].EnterOf(); e != nil {
		t.Errorf("enter 없는 연결선이 슬라이드의 등장을 받았다: %+v", e)
	}
	_, diags = enterDeck(t, "", "  <shape type=\"rect\" id=\"a\"/>\n  <shape type=\"rect\" id=\"b\"/>\n  <connector from=\"a\" to=\"b\" enter=\"rise\"/>")
	if len(diags) != 1 || !strings.Contains(diags[0].Message, `unknown enter "rise" (available values: draw, fade, none)`) {
		t.Errorf("진단 = %v", diags)
	}
}
