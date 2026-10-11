package pipeline_test

import (
	"regexp"
	"testing"

	"github.com/k2sebeom/balpyo/core"
	"github.com/k2sebeom/balpyo/internal/pipeline"
	"github.com/k2sebeom/balpyo/render"
	"github.com/k2sebeom/balpyo/render/web"
)

// spaceVarRef는 문서가 참조하는 공간 변수 이름을 잡는다.
var spaceVarRef = regexp.MustCompile(`var\((--balpyo-space-[A-Za-z0-9_-]+)\)`)

// 테마의 배치 CSS가 참조하는 공간 변수는 덱에 레이아웃 태그·도형·그림이 없어도 선언된다.
// 선언 없는 var()는 그 선언 전체를 무효로 만들어, 텍스트만 있는 표지에서 빈 간격 행이 0으로
// 무너진다 — 한 장의 모양이 덱의 다른 장에 무엇이 있는지에 따라 달라지면 안 된다.
func TestTextOnlyCoverDeclaresLayoutSpaceVars(t *testing.T) {
	noBrowser(t)
	cover := `<slide id="cover" layout="title">
  <text slot="kicker" variant="kicker">KICKER</text>
  <h1 slot="title">Title</h1>
  <text slot="subtitle" role="heading" color="muted">Subtitle</text>
  <text slot="author">Author</text>
  <text slot="date">2026-10-11</text>
</slide>`
	for _, theme := range core.BuiltinThemes() {
		t.Run(theme, func(t *testing.T) {
			dir := project(t, map[string]string{
				"balpyo.json":      `{"title": "cover", "theme": "` + theme + `"}`,
				"slides/001.slide": cover,
			})
			opts := pipeline.Options{ProjectDir: dir, Targets: []render.Target{render.TargetHTML}}
			deck, _, err := pipeline.LoadDeck(opts)
			if err != nil {
				t.Fatal(err)
			}
			out, _, _, err := pipeline.Render(deck, opts)
			if err != nil {
				t.Fatal(err)
			}
			html := string(out[render.TargetHTML][web.IndexFile])
			for _, m := range spaceVarRef.FindAllStringSubmatch(html, -1) {
				if !regexp.MustCompile(regexp.QuoteMeta(m[1]) + `\s*:`).MatchString(html) {
					t.Errorf("%s is referenced but never declared", m[1])
				}
			}
		})
	}
}
