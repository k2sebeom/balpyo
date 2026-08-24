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

// 픽스처 이름. 골든이 붙어 픽스처 러너(internal/testkit)가 집는 활성 픽스처다.
const textFixture = "01_text"

// fixtureDeck는 픽스처 프로젝트를 파이프라인과 같은 순서로 읽어 덱을 만든다
// (수집 → 파싱 → 외관 해석).
//
// internal/pipeline을 쓰지 않는 이유는 임포트 순환이다 — 파이프라인이 이 패키지를
// 임포트한다. 픽스처를 전부 돌며 골든과 비교하는 일은 internal/testkit이 맡고, 여기서는
// **실제 SOML 원문**이 어떤 HTML/CSS가 되는지만 본다. 손으로 만든 덱만으로 테스트하면
// 파서·해석기가 실제로 채우는 모양과 렌더러가 기대하는 모양이 갈라져도 알 수 없다.
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
	if err := core.Resolve(deck).Err(); err != nil {
		t.Fatalf("외관 해석 실패: %v", err)
	}
	// 에셋 패스는 Resolve 다음이다 (pipeline.LoadDeck과 같은 순서) — `<image>`가 있는 픽스처는
	// 이 표 없이는 렌더되지 않는다 (render/web/image.go의 관문). 그림이 없는 픽스처에서는 빈
	// 표가 되므로 기존 골든은 움직이지 않는다.
	if err := core.ResolveAssets(deck, os.DirFS(dir)).Err(); err != nil {
		t.Fatalf("에셋 해석 실패: %v", err)
	}
	return deck
}

// 픽스처의 모든 `<text>`가 식별자를 가진 블록으로 나와야 한다 — 하나라도 빠지면 지오메트리
// 패스가 그 요소의 좌표를 잃고, PPTX에서 텍스트 하나가 조용히 사라진다.
func TestRenderFixtureRendersEveryTextElement(t *testing.T) {
	deck := fixtureDeck(t, textFixture)
	out := renderString(t, deck)

	want := 0
	for _, slide := range deck.Slides {
		want += len(slide.Content)
	}
	if want == 0 {
		t.Fatal("픽스처에 콘텐츠가 없다")
	}
	if got := strings.Count(out, "data-balpyo-id="); got != want {
		t.Errorf("렌더된 요소 = %d개, 덱 모델의 콘텐츠 = %d개", got, want)
	}
	if got := strings.Count(out, `data-balpyo-el="text"`); got != want {
		t.Errorf("텍스트 블록 = %d개, want %d", got, want)
	}
}

// 인라인 속성(size="64")과 역할 기본값(role="title"의 굵기·폰트)이 **해석된 뒤** CSS가 된다.
// 렌더러가 Attrs를 다시 읽으면 이 조합이 어긋난다.
func TestRenderFixtureCompilesResolvedAppearance(t *testing.T) {
	out := renderString(t, fixtureDeck(t, textFixture))
	for _, want := range []string{
		// 001_title.slide: <text role="title" size="64" color="fg" align="center">
		"font-size: 64px",
		"font-family: var(--balpyo-font-heading); font-size: 64px; font-weight: 700;",
		"text-align: center",
		// 002_overview.slide: <text role="heading" color="accent1"> — 역할 크기 + 인라인 색
		"font-size: 36px",
		"color: var(--balpyo-color-accent1)",
		// 002_overview.slide: <text role="caption" size="18" color="muted" align="end">
		"color: var(--balpyo-color-muted); text-align: end",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("픽스처 산출물에 %q가 없다", want)
		}
	}
}

// 토큰 색의 RGB는 문서 전체에서 **테마 선언 한 곳**에만 나온다. 요소 쪽에 한 번 더 나오면
// 그 요소는 테마 색 편집을 따라가지 못한다.
func TestRenderFixtureKeepsTokenColorsAsVariables(t *testing.T) {
	deck := fixtureDeck(t, textFixture)
	out := renderString(t, deck)
	for _, token := range deck.Theme.Colors {
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

// 테마 층이 문서에 실려야 요소의 var() 참조가 값을 얻는다 — 선언이 빠지면 var()는 조용히
// 무시되어 글자가 상속색·상속폰트로 그려지고, 그 상태로도 HTML은 열린다.
func TestRenderFixtureInlinesThemeLayer(t *testing.T) {
	deck := fixtureDeck(t, textFixture)
	out := renderString(t, deck)
	if !strings.Contains(out, "@layer balpyo.theme {") {
		t.Fatal("테마 층이 문서에 없다")
	}
	for _, token := range deck.Theme.Colors {
		if want := colorVarPrefix + token.Name + ": #" + token.RGB + ";"; !strings.Contains(out, want) {
			t.Errorf("색 토큰 선언 %q가 없다", want)
		}
	}
	for _, token := range deck.Theme.Fonts {
		if want := fontVarPrefix + string(token.Name) + ": " + fontStack(token) + ";"; !strings.Contains(out, want) {
			t.Errorf("폰트 토큰 선언 %q가 없다", want)
		}
	}
}

// 픽스처의 노트는 숨은 노트 요소로만 나온다 (PPTX 노트 슬라이드와 같은 내용).
func TestRenderFixtureNotes(t *testing.T) {
	deck := fixtureDeck(t, textFixture)
	out := renderString(t, deck)
	notes := 0
	for _, slide := range deck.Slides {
		if slide.Notes == "" {
			continue
		}
		notes++
		if !strings.Contains(out, `<div class="balpyo-notes" hidden>`+slide.Notes+"</div>") {
			t.Errorf("%s의 노트가 숨은 노트 요소로 나오지 않았다", slide.Source)
		}
		if got := strings.Count(out, slide.Notes); got != 1 {
			t.Errorf("%s의 노트가 %d번 나온다, want 1", slide.Source, got)
		}
	}
	if notes == 0 {
		t.Fatal("픽스처에 노트가 없다")
	}
}

func TestRenderFixtureIsDeterministic(t *testing.T) {
	deck := fixtureDeck(t, textFixture)
	if renderString(t, deck) != renderString(t, deck) {
		t.Error("같은 덱의 렌더 결과가 실행마다 다르다")
	}
}
