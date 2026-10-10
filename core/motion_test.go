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
