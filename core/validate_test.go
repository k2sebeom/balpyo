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

func TestValidateRejectsEmptyID(t *testing.T) {
	deck := &soml.Deck{Slides: []*soml.Slide{{Source: "slides/001.slide"}}}
	if !Validate(deck).HasErrors() {
		t.Error("빈 ID는 에러여야 한다")
	}
}
