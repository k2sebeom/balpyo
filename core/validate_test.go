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
	want := `slides/ch/001_cover.slide:2:2: error: 슬라이드 ID "cover"가 slides/001_title.slide:1:12와 충돌한다 (한쪽에 다른 id를 주거나, 필요 없는 슬라이드를 지운다)`
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
