package web

import (
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"

	"github.com/k2sebeom/balpyo/core"
	"github.com/k2sebeom/balpyo/soml"
)

const textFixture = "01_text"

// fixtureDeck는 픽스처 프로젝트를 파이프라인과 같은 순서로 읽어 덱을 만든다
// (수집 → 파싱 → 외관 해석 → 에셋). internal/pipeline은 이 패키지를 임포트하므로 쓸 수 없다.
func fixtureDeck(t *testing.T, name string) *soml.Deck {
	t.Helper()
	dir := filepath.Join("..", "..", "testdata", "fixtures", name)
	cfg, err := core.LoadConfig(dir)
	if err != nil {
		t.Fatal(err)
	}
	stage, err := cfg.Aspect.Stage()
	if err != nil {
		t.Fatal(err)
	}
	files, err := soml.Collect(os.DirFS(dir), cfg.SlidesDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) == 0 {
		t.Fatalf("%s에 슬라이드가 없다", name)
	}
	deck := &soml.Deck{Title: cfg.Title, Author: cfg.Author, Lang: cfg.Lang, Aspect: cfg.Aspect, Stage: stage}
	for _, file := range files {
		source := path.Join(cfg.SlidesDir, file.Path)
		src, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(source)))
		if err != nil {
			t.Fatal(err)
		}
		doc, diags := soml.Parse(source, string(src))
		if err := diags.Err(); err != nil {
			t.Fatalf("%s 파싱 실패: %v", source, err)
		}
		if doc.Skip {
			continue
		}
		id := doc.ID
		if id == "" {
			id = file.DerivedID()
		}
		deck.Slides = append(deck.Slides, &soml.Slide{
			Index: len(deck.Slides), ID: id, Source: source, Chapter: file.Chapter,
			Notes: doc.Notes, Content: doc.Content, Tokens: doc.Tokens,
		})
	}
	if err := core.Resolve(deck, nil).Err(); err != nil {
		t.Fatalf("외관 해석 실패: %v", err)
	}
	if err := core.ResolveAssets(deck, os.DirFS(dir)).Err(); err != nil {
		t.Fatalf("에셋 해석 실패: %v", err)
	}
	return deck
}

// 토큰 색의 RGB는 문서 전체에서 **테마 선언 한 곳**에만 나온다. 요소 쪽에 한 번 더 나오면
// 그 요소는 테마 색 편집을 따라가지 못한다.
func TestRenderFixtureKeepsTokenColorsAsVariables(t *testing.T) {
	deck := fixtureDeck(t, textFixture)
	out := renderString(t, deck)
	if !strings.Contains(out, "@layer balpyo.theme {") {
		t.Fatal("테마 층이 문서에 없다")
	}
	for _, token := range deck.Theme.Colors {
		if want := colorVarPrefix + token.Name + ": #" + token.RGB + ";"; !strings.Contains(out, want) {
			t.Errorf("색 토큰 선언 %q가 없다", want)
		}
		if got := strings.Count(out, token.RGB); got > 1 {
			t.Errorf("색 토큰 %s의 RGB(%s)가 %d번 나온다 — 테마 선언 한 곳에만 있어야 한다",
				token.Name, token.RGB, got)
		}
	}
	// 폰트도 같다: 패밀리 이름은 테마 변수 선언에만 있고 요소는 var()로 참조한다.
	if got := strings.Count(out, `style="font-family: var(--balpyo-font`); got == 0 {
		t.Error("요소가 폰트 토큰을 var()로 참조하지 않는다")
	}
}
