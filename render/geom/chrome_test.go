package geom

import (
	"math"
	"strings"
	"testing"

	"github.com/k2sebeom/balpyo/render/web"
	"github.com/k2sebeom/balpyo/soml"
)

// chromeItems는 확정된 크롬 둘이다 (값 층이 이미 걸렀다 — ADR 0045).
func chromeItems() []soml.ChromeItem {
	style := soml.TextStyle{Role: soml.RoleCaption, Font: soml.FontBody, Size: 16, Weight: 400,
		LineHeight: 1.4, Color: soml.TokenColor("muted"), Align: soml.AlignStart}
	return []soml.ChromeItem{
		{Kind: soml.ChromeFooter, Text: "내부용", Align: soml.AlignStart, Style: style},
		{Kind: soml.ChromePageNumber, Text: "1", Align: soml.AlignEnd, Style: style},
	}
}

// chromeBand는 바닥 띠 안의 rect다 (웹 렌더러의 CSS가 내는 자리).
func chromeBand(x, w float64) Rect {
	pad := float64(web.StagePadding)
	return Rect{X: x, Y: 720 - pad, W: w, H: pad}
}

// chromeMeasured는 크롬까지 잰 측정 결과다.
func chromeMeasured(deck *soml.Deck, rects ...Rect) measurement {
	m := validMeasurement(deck)
	for i := range m.Slides {
		if len(deck.Slides[i].Chrome) == 0 {
			continue
		}
		for j, item := range deck.Slides[i].Chrome {
			rect := chromeBand(64, 120)
			if j < len(rects) {
				rect = rects[j]
			}
			m.Slides[i].Chrome = append(m.Slides[i].Chrome, chromeMeasurement{
				Kind: string(item.Kind), Rect: rect,
			})
		}
	}
	return m
}

// 크롬 rect는 Boxes가 아니라 곁 표에 담긴다 (ADR 0045) — Boxes는 덱 요소와 1:1이다.
func TestChromeIsMeasuredIntoSideTable(t *testing.T) {
	deck := testDeck([]soml.Node{textElem(chars("본문"))})
	deck.Slides[0].Chrome = chromeItems()
	footer := chromeBand(64, 120)
	number := chromeBand(1180, 36)

	g, err := newGeometry(deck, chromeMeasured(deck, footer, number))
	if err != nil {
		t.Fatalf("지오메트리를 만들 수 없다: %v", err)
	}
	if len(g.Boxes) != 1 {
		t.Errorf("Boxes %d개, want 1 (크롬은 Boxes에 들어가지 않는다)", len(g.Boxes))
	}
	if len(g.Chrome) != 2 {
		t.Fatalf("Chrome %d개, want 2", len(g.Chrome))
	}
	if got, ok := g.ChromeRect(0, soml.ChromeFooter); !ok || got != footer {
		t.Errorf("푸터 rect = %s (ok=%v), want %s", got, ok, footer)
	}
	if got, ok := g.ChromeRect(0, soml.ChromePageNumber); !ok || got != number {
		t.Errorf("번호 rect = %s (ok=%v), want %s", got, ok, number)
	}
	if _, ok := g.ChromeRect(0, "logo"); ok {
		t.Error("없는 kind의 rect가 나왔다")
	}
}

// 잰 HTML의 크롬이 덱의 확정 모델과 어긋나면 빌드를 멈춘다.
func TestNewGeometryRejectsChromeMismatch(t *testing.T) {
	deck := testDeck([]soml.Node{textElem(chars("본문"))})
	deck.Slides[0].Chrome = chromeItems()
	tests := []struct {
		name    string
		mutate  func(m *measurement)
		wantMsg string
	}{
		{"HTML에 크롬이 없다", func(m *measurement) { m.Slides[0].Chrome = nil }, "크롬"},
		{"kind가 어긋난다", func(m *measurement) { m.Slides[0].Chrome[1].Kind = string(soml.ChromeFooter) }, "크롬"},
		{"rect가 유효하지 않다", func(m *measurement) { m.Slides[0].Chrome[0].Rect = Rect{W: math.Inf(1)} }, "유효하지 않다"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := chromeMeasured(deck)
			tc.mutate(&m)
			if _, err := newGeometry(deck, m); err == nil {
				t.Fatal("어긋남이 진단되지 않았다")
			} else if !strings.Contains(err.Error(), tc.wantMsg) {
				t.Errorf("에러 = %q, want substring %q", err.Error(), tc.wantMsg)
			}
		})
	}
}

// 측정 스크립트는 웹 렌더러가 소유한 선택자·속성으로 크롬을 집는다 — 두 벌이면 조용히 사라진다.
func TestMeasureExprReadsChrome(t *testing.T) {
	expr := measureExpr()
	for _, want := range []string{web.ChromeSelector(), web.ChromeKindAttr} {
		if !strings.Contains(expr, want) {
			t.Errorf("측정 스크립트에 %q가 없다", want)
		}
	}
}

// 크롬의 판정은 둘이다 (ADR 0045): 슬라이드 이탈과 크롬끼리의 겹침. 크롬은 여백 띠에 사는
// 것이 정의이므로 콘텐츠 박스 판정은 묻지 않는다.
func TestDiagnoseChrome(t *testing.T) {
	tests := []struct {
		name  string
		items int // chromeItems()의 앞 몇 개를 켜는가
		rects []Rect
		want  []string
	}{
		{"띠 안의 크롬은 조용하다", 2, []Rect{chromeBand(64, 120), chromeBand(1180, 36)}, nil},
		{"슬라이드 이탈", 1, []Rect{chromeBand(64, 1300)}, []string{"footer", "슬라이드", "오른쪽으로"}},
		{"푸터가 번호를 덮는다", 2, []Rect{chromeBand(64, 1140), chromeBand(1180, 36)}, []string{"footer", "pageNumber", "겹친다"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			deck := testDeck([]soml.Node{textElem(chars("본문"))})
			deck.Slides[0].Chrome = chromeItems()[:tc.items]
			g, err := newGeometry(deck, chromeMeasured(deck, tc.rects...))
			if err != nil {
				t.Fatal(err)
			}
			diags := Diagnose(deck, g)
			if tc.want == nil {
				if len(diags) != 0 {
					t.Errorf("진단 %d건, want 0 (%v)", len(diags), diags)
				}
				return
			}
			if len(diags) != 1 {
				t.Fatalf("진단 %d건, want 1 (%v)", len(diags), diags)
			}
			for _, want := range tc.want {
				if !strings.Contains(diags[0].Message, want) {
					t.Errorf("진단 = %q, want substring %q", diags[0].Message, want)
				}
			}
			if diags[0].Severity != soml.SeverityWarning || diags[0].File != deck.Slides[0].Source {
				t.Errorf("진단 = %+v, want %s의 warning", diags[0], deck.Slides[0].Source)
			}
		})
	}
}

// Synthetic도 크롬을 채운다 — PPTX 조립이 브라우저 없이 크롬 도형을 시험한다.
func TestSyntheticFillsChrome(t *testing.T) {
	deck := testDeck([]soml.Node{textElem(chars("본문"))}, nil)
	deck.Slides[0].Chrome = chromeItems()
	rect := Rect{X: 64, Y: 64, W: 1152, H: 36}
	g := Synthetic(deck, rect)
	if len(g.Chrome) != 2 {
		t.Fatalf("Chrome %d개, want 2", len(g.Chrome))
	}
	if got, ok := g.ChromeRect(0, soml.ChromePageNumber); !ok || got != rect {
		t.Errorf("번호 rect = %s (ok=%v), want %s", got, ok, rect)
	}
	if _, ok := g.ChromeRect(1, soml.ChromeFooter); ok {
		t.Error("크롬이 없는 슬라이드에 rect가 있다")
	}
}
