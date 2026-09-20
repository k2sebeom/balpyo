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

// 크롬 rect는 **Boxes가 아니라 곁 표**에 담긴다 (ADR 0045): Boxes는 덱 트리의 요소와 1:1이라
// (Refs ↔ Boxes 자리 맞대기) 크롬을 끼우면 그 대조가 무너지고, 크롬은 배치 진단의 대상이
// 저자의 요소와 다르다.
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

// 잰 HTML의 크롬이 덱의 확정 모델과 어긋나면 **빌드를 멈춘다** — 좌표가 엉뚱한 크롬에 붙으면
// 번호가 푸터 자리에 놓인 PPTX가 나오고, 그것은 열어 보기 전까지 정상으로 보인다.
func TestNewGeometryRejectsChromeMismatch(t *testing.T) {
	deck := testDeck([]soml.Node{textElem(chars("본문"))})
	deck.Slides[0].Chrome = chromeItems()
	tests := []struct {
		name    string
		mutate  func(m *measurement)
		wantMsg string
	}{
		{
			name:    "HTML에 크롬이 없다",
			mutate:  func(m *measurement) { m.Slides[0].Chrome = nil },
			wantMsg: "크롬",
		},
		{
			name: "kind가 어긋난다",
			mutate: func(m *measurement) {
				m.Slides[0].Chrome[1].Kind = string(soml.ChromeFooter)
			},
			wantMsg: "크롬",
		},
		{
			name: "rect가 유효하지 않다",
			mutate: func(m *measurement) {
				m.Slides[0].Chrome[0].Rect = Rect{W: math.Inf(1)}
			},
			wantMsg: "유효하지 않다",
		},
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

// 측정 스크립트는 웹 렌더러가 소유한 선택자·속성으로 크롬을 집는다 — 문자열이 두 벌이 되면
// 크롬이 "못 찾았다"가 아니라 "PPTX에만 없다"로 조용히 사라진다.
func TestMeasureExprReadsChrome(t *testing.T) {
	expr := measureExpr()
	for _, want := range []string{web.ChromeSelector(), web.ChromeKindAttr} {
		if !strings.Contains(expr, want) {
			t.Errorf("측정 스크립트에 %q가 없다", want)
		}
	}
}

// 크롬의 판정은 둘이다 (ADR 0045): 슬라이드 이탈과 **크롬끼리의** 겹침. 콘텐츠 박스 판정은
// 묻지 않는다 — 크롬은 여백 띠에 사는 것이 정의다.
func TestDiagnoseChromeEscapesSlide(t *testing.T) {
	deck := testDeck([]soml.Node{textElem(chars("본문"))})
	deck.Slides[0].Chrome = chromeItems()
	// 푸터가 오른쪽으로 흘러 슬라이드를 나갔다 (긴 문자열).
	g, err := newGeometry(deck, chromeMeasured(deck, chromeBand(64, 1300), chromeBand(1180, 36)))
	if err != nil {
		t.Fatal(err)
	}
	diags := Diagnose(deck, g)
	if len(diags) != 1 {
		t.Fatalf("진단 %d건, want 1 (%v)", len(diags), diags)
	}
	msg := diags[0].Message
	for _, want := range []string{"footer", "슬라이드", "오른쪽으로"} {
		if !strings.Contains(msg, want) {
			t.Errorf("진단 = %q, want substring %q", msg, want)
		}
	}
	if diags[0].Severity != soml.SeverityWarning {
		t.Errorf("심각도 = %v, want warning", diags[0].Severity)
	}
	if diags[0].File != deck.Slides[0].Source {
		t.Errorf("진단 파일 = %q, want %q", diags[0].File, deck.Slides[0].Source)
	}
}

// 푸터가 길어 번호를 덮으면 두 글자가 겹쳐 읽히지 않는다 — 열어 보지 않으면 모르는 사고다.
func TestDiagnoseChromeOverlap(t *testing.T) {
	deck := testDeck([]soml.Node{textElem(chars("본문"))})
	deck.Slides[0].Chrome = chromeItems()
	g, err := newGeometry(deck, chromeMeasured(deck, chromeBand(64, 1140), chromeBand(1180, 36)))
	if err != nil {
		t.Fatal(err)
	}
	diags := Diagnose(deck, g)
	if len(diags) != 1 {
		t.Fatalf("진단 %d건, want 1 (%v)", len(diags), diags)
	}
	for _, want := range []string{"footer", "pageNumber", "겹친다"} {
		if !strings.Contains(diags[0].Message, want) {
			t.Errorf("진단 = %q, want substring %q", diags[0].Message, want)
		}
	}
}

// 띠 안에 얌전히 있는 크롬은 아무 말도 듣지 않는다 — 크롬이 여백에 있다는 이유로 콘텐츠 박스
// 진단을 받으면 모든 덱이 고칠 수 없는 경고를 받는다.
func TestDiagnoseChromeIsQuietInTheBand(t *testing.T) {
	deck := testDeck([]soml.Node{textElem(chars("본문"))})
	deck.Slides[0].Chrome = chromeItems()
	g, err := newGeometry(deck, chromeMeasured(deck, chromeBand(64, 120), chromeBand(1180, 36)))
	if err != nil {
		t.Fatal(err)
	}
	if diags := Diagnose(deck, g); len(diags) != 0 {
		t.Errorf("띠 안의 크롬이 진단을 받았다: %v", diags)
	}
}

// Synthetic도 크롬을 채운다 — 소비 측(PPTX 조립)이 브라우저 없이 크롬 도형을 시험할 수 있어야
// 한다 (Boxes를 채우는 것과 같은 근거).
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
