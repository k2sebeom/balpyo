package core

import (
	"strings"
	"testing"

	"github.com/k2sebeom/balpyo/soml"
)

func TestValidateAcceptsDistinctIDs(t *testing.T) {
	deck := &soml.Deck{Slides: []*soml.Slide{
		{ID: "title", Source: "slides/001_title.slide"},
		{ID: "agenda", Source: "slides/002_agenda.slide"},
	}}
	if diags := Validate(deck); len(diags) != 0 {
		t.Errorf("진단이 없어야 한다: %v", diags)
	}
}

func TestValidateRejectsDuplicateID(t *testing.T) {
	deck := &soml.Deck{Slides: []*soml.Slide{
		{ID: "intro", Source: "slides/001_intro.slide"},
		{ID: "intro", Source: "slides/02_deep/001_intro.slide"},
	}}
	diags := Validate(deck)
	if !diags.HasErrors() {
		t.Fatal("ID 충돌은 에러여야 한다")
	}
	msg := diags[0].String()
	// 진단은 충돌한 두 파일을 모두 가리켜야 사람이 고칠 수 있다.
	if !strings.Contains(msg, "slides/02_deep/001_intro.slide") ||
		!strings.Contains(msg, "slides/001_intro.slide") {
		t.Errorf("두 파일을 모두 가리켜야 한다: %s", msg)
	}
}

// 충돌 진단은 두 번째 슬라이드의 ID 자리에 서고, 메시지 안의 첫 슬라이드도 `file:line:col`로
// 가리킨다 — 둘 다 에디터에서 클릭해 갈 수 있어야 어느 쪽을 고칠지 바로 고를 수 있다.
func TestValidateDuplicateIDPointsAtBothSlides(t *testing.T) {
	at := func(line, col int) soml.Span {
		return soml.Span{Start: soml.Pos{Line: line, Col: col}, End: soml.Pos{Line: line, Col: col + 5}}
	}
	deck := &soml.Deck{Slides: []*soml.Slide{
		{ID: "cover", Source: "slides/001_title.slide", IDSpan: at(1, 12)},
		{ID: "cover", Source: "slides/ch/001_cover.slide", IDSpan: at(2, 2)},
	}}
	diags := Validate(deck)
	if len(diags) != 1 {
		t.Fatalf("진단 한 건이어야 한다: %v", diags)
	}
	want := `slides/ch/001_cover.slide:2:2: error: slide ID "cover" collides with slides/001_title.slide:1:12 (give one of them a different id, or remove the slide you do not need)`
	if got := diags[0].String(); got != want {
		t.Errorf("진단\n got: %s\nwant: %s", got, want)
	}
}

func TestValidateRejectsEmptyID(t *testing.T) {
	deck := &soml.Deck{Slides: []*soml.Slide{{Source: "slides/001.slide"}}}
	if !Validate(deck).HasErrors() {
		t.Error("빈 ID는 에러여야 한다")
	}
}

// linkDeck은 앞 슬라이드부터 `<a href>`를 하나씩 가진 세 장짜리 덱이다 — 링크 검증은 해석이
// 채운 Link를 본다.
func linkDeck(hrefs ...string) *soml.Deck {
	deck := &soml.Deck{}
	for i, id := range []string{"cover", "second", "third"} {
		slide := &soml.Slide{Index: i, ID: id, Source: "slides/" + id + ".slide"}
		if i < len(hrefs) {
			span := soml.Span{Start: soml.Pos{Line: 2, Col: 12}, End: soml.Pos{Line: 2, Col: 12 + len(hrefs[i])}}
			link := &soml.Element{
				Name:       "a",
				Attrs:      []soml.Attr{{Name: "href", Value: hrefs[i], ValueSpan: span, HasValue: true}},
				Appearance: &soml.Appearance{Link: hrefs[i]},
			}
			slide.Content = []soml.Node{&soml.Element{Name: "p", Children: []soml.Node{link}}}
		}
		deck.Slides = append(deck.Slides, slide)
	}
	return deck
}

// 덱 안 링크는 웹 뷰어의 해시 딥링크와 같은 규칙으로 목적지를 찾는다 — id, `#/id`, 1부터 세는
// 번호, 빈 조각(첫 슬라이드). 외부 URL은 이 층이 보지 않는다.
func TestValidateAcceptsSlideLinks(t *testing.T) {
	if diags := Validate(linkDeck("#third", "#/cover", "#/2")); len(diags) != 0 {
		t.Errorf("진단이 없어야 한다: %v", diags)
	}
	if diags := Validate(linkDeck("https://example.com/#nowhere", "#", "#3")); len(diags) != 0 {
		t.Errorf("외부 URL · 빈 조각 · 번호는 진단이 없어야 한다: %v", diags)
	}
}

// 덱에 없는 목적지는 href 값 자리에 서는 경고다 — 오타면 가까운 id를 권한다.
func TestValidateWarnsUnknownSlideLink(t *testing.T) {
	diags := Validate(linkDeck("#secnd", "#/4"))
	if len(diags) != 2 {
		t.Fatalf("진단 두 건이어야 한다: %v", diags)
	}
	if diags.HasErrors() {
		t.Errorf("목적지 없는 링크는 경고다: %v", diags)
	}
	want := `slides/cover.slide:2:12: warning: href "#secnd" points to no slide in this deck (did you mean "#second"?) — link to a slide id ("#cover") or a 1-based slide number ("#/2"); skipped slides are not in the deck`
	if got := diags[0].String(); got != want {
		t.Errorf("진단\n got: %s\nwant: %s", got, want)
	}
	if got := diags[1].String(); !strings.Contains(got, `href "#/4" points to no slide in this deck —`) {
		t.Errorf("번호가 덱 밖이면 경고다: %s", got)
	}
}
