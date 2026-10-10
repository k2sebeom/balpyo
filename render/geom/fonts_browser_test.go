package geom_test

import (
	"strings"
	"testing"

	"github.com/k2sebeom/balpyo/internal/testkit"
	"github.com/k2sebeom/balpyo/render/geom"
	"github.com/k2sebeom/balpyo/soml"
)

// 없는 이름은 없다고, 있는 이름은 있다고 읽는다. "있는" 글꼴은 이 기계의 설치 목록을 가정하지
// 않고 브라우저에게서 얻는다 — 제네릭이 고른 면은 언제나 설치된 면이다. 같은 이름을 두 번
// 물어도 한 번 말하고, 탐침은 문서에 남지 않는다 (같은 세션이 뒤이어 스크린샷을 찍는다).
func TestMissingFontsTellsAbsentFromInstalled(t *testing.T) {
	testkit.RequireChromium(t)
	session := testkit.SharedSession(t)
	deck := layoutDeck(t, `<text>본문</text>`)
	if err := session.Open(renderHTML(t, deck, nil)); err != nil {
		t.Fatal(err)
	}
	for _, sample := range []string{geom.LatinSample, geom.EastAsianSample} {
		used, err := geom.UsedFamilies(session, "sans-serif", sample)
		if err != nil {
			t.Fatal(err)
		}
		if len(used) == 0 {
			t.Fatalf("제네릭으로 그린 면이 없다 (%q)", sample)
		}
		installed := used[0]
		const absent = "Balpyo No Such Font"
		missing, err := session.MissingFonts([]geom.FontProbe{
			{Family: installed, Sample: sample},
			{Family: absent, Sample: sample},
			{Family: absent, Sample: sample},
		})
		if err != nil {
			t.Fatal(err)
		}
		if len(missing) != 1 || missing[0] != absent {
			t.Errorf("설치된 %q · 없는 %q → missing = %v, want [%s]", installed, absent, missing, absent)
		}
	}
	var left int
	if err := session.Eval(`document.querySelectorAll(".balpyo-font-probe").length`, &left); err != nil {
		t.Fatal(err)
	}
	if left != 0 {
		t.Errorf("탐침 %d개가 문서에 남았다", left)
	}
}

// 측정이 테마의 글꼴을 묻고, 배치 진단이 없는 글꼴을 경고한다 — 빌드가 실제로 타는 길이다.
func TestMeasureWarnsAboutMissingThemeFont(t *testing.T) {
	deck := layoutDeck(t, `<text>본문</text>`)
	for i := range deck.Theme.Fonts {
		if deck.Theme.Fonts[i].Name == soml.FontBody {
			deck.Theme.Fonts[i].EastAsian = "Balpyo No Such Font"
		}
	}
	g := measureLayout(t, deck)
	found := false
	for _, f := range g.MissingFonts {
		found = found || f == "Balpyo No Such Font"
	}
	if !found {
		t.Fatalf("없는 글꼴을 기록하지 않았다: %v", g.MissingFonts)
	}
	var warned bool
	for _, d := range geom.Diagnose(deck, g) {
		warned = warned || (d.Severity == soml.SeverityWarning && strings.Contains(d.Message, `"Balpyo No Such Font" (body.ea)`))
	}
	if !warned {
		t.Errorf("배치 진단에 글꼴 경고가 없다: %v", geom.Diagnose(deck, g))
	}
}
