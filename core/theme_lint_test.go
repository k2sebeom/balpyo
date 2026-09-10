package core

import (
	"testing"

	"github.com/k2sebeom/balpyo/soml"
)

// TestLayoutCSSAppearanceWarning은 외관 프로퍼티가 배치가 아니라 경고임을, 배치 프로퍼티는
// 조용함을, --strict가 경고를 에러로 승격함을 못 박는다 (ADR 0014를 레이아웃 CSS로 확장).
func TestLayoutCSSAppearanceWarning(t *testing.T) {
	appearance := []string{"color", "background", "border", "box-shadow", "font-size"}
	for _, prop := range appearance {
		diags := lintLayoutCSS(prop+": x", "title")
		if len(diags) != 1 {
			t.Fatalf("%s: 진단 수 = %d, want 1 (%v)", prop, len(diags), diags)
		}
		if diags[0].Severity != soml.SeverityWarning {
			t.Errorf("%s: severity = %v, want warning", prop, diags[0].Severity)
		}
		if diags[0].File != "layouts/title.css" {
			t.Errorf("%s: file = %q, want layouts/title.css", prop, diags[0].File)
		}
	}
	// 배치 프로퍼티는 조용하다 (값은 보지 않는다 — 어떤 값이든 통과).
	if diags := lintLayoutCSS("display: grid; gap: 24px; padding: 10", "content"); len(diags) != 0 {
		t.Errorf("배치 프로퍼티가 경고를 냈다: %v", diags)
	}
	// --strict: 외관 경고가 에러로 승격한다 (soml.Diagnostics.Strict).
	strict := lintLayoutCSS("color: red", "title").Strict()
	if !strict.HasErrors() {
		t.Error("--strict가 외관 경고를 에러로 승격하지 않았다")
	}
}

// TestLayoutCSSSyntaxError는 문법 위반(콜론·이름·값 없음)이 경고가 아니라 에러임을 못 박는다 —
// allowlist 밖은 경고지만 문법은 값 층과 같은 정책으로 에러다 (style.go와 같은 규칙).
func TestLayoutCSSSyntaxError(t *testing.T) {
	for _, css := range []string{"display flex", ": 10px", "width:"} {
		diags := lintLayoutCSS(css, "title")
		if len(diags) == 0 {
			t.Fatalf("%q: 진단이 없다 — 문법 위반이 잡히지 않았다", css)
		}
		if diags[0].Severity != soml.SeverityError {
			t.Errorf("%q: severity = %v, want error", css, diags[0].Severity)
		}
	}
}

// TestLayoutCSSLintDeterminism은 맵 순회 순서가 진단 순서에 새지 않음을 못 박는다 — lintLayouts는
// 레이아웃 이름을 정렬해 순회하므로 File 순서가 실행마다 이름 정렬순으로 안정이어야 한다.
func TestLayoutCSSLintDeterminism(t *testing.T) {
	layouts := map[string]string{
		"zebra": "color: red",
		"alpha": "color: red",
		"mango": "color: red",
	}
	want := []string{"layouts/alpha.css", "layouts/mango.css", "layouts/zebra.css"}
	for run := 0; run < 5; run++ {
		diags := lintLayouts(layouts)
		if len(diags) != len(want) {
			t.Fatalf("진단 수 = %d, want %d", len(diags), len(want))
		}
		for i, w := range want {
			if diags[i].File != w {
				t.Errorf("run %d: 진단[%d].File = %q, want %q", run, i, diags[i].File, w)
			}
		}
	}
}
