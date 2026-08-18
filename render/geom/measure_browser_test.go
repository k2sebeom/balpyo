// 이 파일만 브라우저를 띄운다 — 외부 도구 정책(없으면 로컬은 건너뛰고 테스트 이미지에서는
// 실패)은 internal/testkit이 소유하므로, 패키지 밖 테스트로 두어 testkit을 임포트한다
// (testkit → pipeline → render/pptx → render/geom 이므로 같은 패키지 안에서는 순환이 된다).
package geom_test

import (
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/k2sebeom/balpyo/core"
	"github.com/k2sebeom/balpyo/internal/testkit"
	"github.com/k2sebeom/balpyo/render/geom"
	"github.com/k2sebeom/balpyo/render/web"
	"github.com/k2sebeom/balpyo/soml"
)

// longBody는 어떤 폰트에서도 1152px 안에서 여러 줄이 되는 길이다 — 줄 박스 수집을 검증하려면
// 실제로 줄이 나뉘어야 한다.
const longBody = "모놀리스 분리를 마쳤고, 배포 파이프라인을 자동화했으며, 관측성 스택(OpenTelemetry + Grafana)을 " +
	"도입했다. 하반기에는 멀티 리전 배포와 비용 최적화를 함께 진행하며, 팀별 서비스 경계를 다시 정리한다."

// browserDeck은 같은 내용의 슬라이드 두 장을 가진 덱이다. 두 장이 같아야 좌표가 슬라이드
// 상대인지 문서 상대인지가 드러난다 — 문서 상대라면 두 번째 슬라이드의 y가 크게 다르다.
func browserDeck(t *testing.T) *soml.Deck {
	t.Helper()
	stage, err := soml.Aspect16x9.Stage()
	if err != nil {
		t.Fatal(err)
	}
	content := func() []soml.Node {
		return []soml.Node{
			&soml.Element{
				Name:     soml.TextElement,
				Attrs:    []soml.Attr{{Name: "role", Value: "title", HasValue: true}},
				Children: []soml.Node{&soml.Text{Value: "2026 아키텍처 리뷰"}},
			},
			&soml.Element{
				Name:     soml.TextElement,
				Children: []soml.Node{&soml.Text{Value: longBody}},
			},
		}
	}
	deck := &soml.Deck{
		Title: "geometry", Lang: "ko", Aspect: soml.Aspect16x9, Stage: stage,
		Slides: []*soml.Slide{
			{Index: 0, ID: "first", Source: "slides/001_first.slide", Content: content()},
			{Index: 1, ID: "second", Source: "slides/002_second.slide", Content: content()},
		},
	}
	if diags := core.Resolve(deck); len(diags) > 0 {
		t.Fatalf("외관 해석 진단: %v", diags)
	}
	return deck
}

// renderHTML은 덱을 임시 디렉토리의 index.html로 쓴다 — 측정 대상은 사용자가 받는 것과 같은
// 바이트여야 한다.
func renderHTML(t *testing.T, deck *soml.Deck, transform func(string) string) string {
	t.Helper()
	files, err := web.New().Render(deck)
	if err != nil {
		t.Fatal(err)
	}
	html := string(files[web.IndexFile])
	if transform != nil {
		html = transform(html)
	}
	path := filepath.Join(t.TempDir(), web.IndexFile)
	if err := os.WriteFile(path, []byte(html), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestMeasureReadsSlideRelativeBoxes(t *testing.T) {
	testkit.RequireChromium(t)
	deck := browserDeck(t)
	path := renderHTML(t, deck, nil)

	g, err := geom.Measure(deck, path)
	if err != nil {
		t.Fatalf("지오메트리 패스 실패: %v", err)
	}
	if len(g.Boxes) != len(geom.Refs(deck)) {
		t.Fatalf("박스 %d개, 덱 요소 %d개", len(g.Boxes), len(geom.Refs(deck)))
	}

	title, ok := g.Box("s0.0")
	if !ok {
		t.Fatalf("s0.0이 없다: %+v", g.Boxes)
	}
	// 스테이지 여백 64px, 콘텐츠 폭 1280-128 = 1152px (render/web contentCSS). 논리 px가
	// 그대로 읽혀야 한다 — 여기가 어긋나면 PPTX의 모든 좌표가 틀린다.
	if title.Rect.X != 64 || title.Rect.Y != 64 || title.Rect.W != 1152 {
		t.Errorf("제목 박스 = %s, want (64,64 1152×…)", title.Rect)
	}

	// 두 번째 슬라이드는 문서에서 768px 아래에 있다. 좌표가 슬라이드 상대라면 두 슬라이드의
	// 박스가 완전히 같다 (같은 내용이므로).
	for _, id := range []string{"0", "1"} {
		first, ok1 := g.Box("s0." + id)
		second, ok2 := g.Box("s1." + id)
		if !ok1 || !ok2 {
			t.Fatalf("s0.%s/s1.%s를 찾을 수 없다", id, id)
		}
		if first.Rect != second.Rect {
			t.Errorf("자식 %s: 슬라이드 0 = %s, 슬라이드 1 = %s — 좌표가 슬라이드 상대가 아니다",
				id, first.Rect, second.Rect)
		}
		if second.Rect.Y >= float64(deck.Stage.Height) {
			t.Errorf("s1.%s의 y = %v — 스테이지 높이를 넘었다 (문서 좌표가 섞였다)", id, second.Rect.Y)
		}
	}

	// 세로 스택: 본문은 제목 아래 간격(24px)만큼 떨어진다.
	body, ok := g.Box("s0.1")
	if !ok {
		t.Fatal("s0.1이 없다")
	}
	if want := title.Rect.Y + title.Rect.H + 24; math.Abs(body.Rect.Y-want) > 0.01 {
		t.Errorf("본문 y = %v, want %v (제목 아래 24px)", body.Rect.Y, want)
	}
}

// 텍스트 메트릭: 줄 박스는 Iteration 1 리뷰 포인트(브라우저가 잰 박스가 다른 엔진에서 같은
// 줄바꿈을 유지하는가, PRD §13-1)의 수치 재료다.
func TestMeasureCollectsLineBoxes(t *testing.T) {
	testkit.RequireChromium(t)
	deck := browserDeck(t)
	g, err := geom.Measure(deck, renderHTML(t, deck, nil))
	if err != nil {
		t.Fatal(err)
	}

	body, ok := g.Box("s0.1")
	if !ok {
		t.Fatal("s0.1이 없다")
	}
	if body.LineCount() < 2 {
		t.Fatalf("줄 %d개 — 긴 본문은 여러 줄이어야 한다 (%s)", body.LineCount(), body.Rect)
	}
	// 줄 박스는 위에서 아래로, 박스 안에 들어 있어야 한다. 벗어나면 좌표계가 섞인 것이다.
	prev := -math.MaxFloat64
	for i, line := range body.Lines {
		if line.Y < prev {
			t.Errorf("줄 %d의 y = %v가 앞 줄(%v)보다 위다", i, line.Y, prev)
		}
		prev = line.Y
		if line.X < body.Rect.X-0.5 || line.X+line.W > body.Rect.X+body.Rect.W+0.5 {
			t.Errorf("줄 %d(%s)이 박스(%s) 밖이다", i, line, body.Rect)
		}
	}
	// 박스 높이 ≈ 줄 수 × 줄 간격. 이 관계가 성립해야 "박스 고정 + autofit off"로 내보낸
	// 박스가 텍스트를 담고, 줄이 하나 늘었을 때 얼마나 넘치는지가 이 간격으로 계산된다.
	advance := body.LineAdvance()
	if want := float64(body.LineCount()) * advance; math.Abs(body.Rect.H-want) > 1 {
		t.Errorf("박스 높이 = %v, 줄 %d개 × 간격 %v = %v", body.Rect.H, body.LineCount(), advance, want)
	}
	// 줄 박스는 글자 상자이므로 줄 간격보다 낮다 — 이 둘이 같아지면 잰 것이 글자 상자가
	// 아니라 줄 슬롯이라는 뜻이고, 그러면 겹치는 줄을 구분할 수 없다.
	if body.LineHeight() <= 0 || body.LineHeight() > advance {
		t.Errorf("줄 박스 높이 = %v, 줄 간격 = %v", body.LineHeight(), advance)
	}
	if slack := body.Slack(); slack < 0 || slack > body.Rect.W {
		t.Errorf("Slack = %v, 박스 폭 %v 안이어야 한다", slack, body.Rect.W)
	}
}

// 스테이지가 1:1이라는 전제는 확인해야 하는 사실이다 — 페이지가 조용히 스케일되면 모든
// 좌표가 틀린 채로 산출물까지 간다.
func TestMeasureRejectsScaledPage(t *testing.T) {
	testkit.RequireChromium(t)
	deck := browserDeck(t)
	path := renderHTML(t, deck, func(html string) string {
		return strings.Replace(html, "</head>", "<style>.balpyo-slide { zoom: 0.5; }</style>\n</head>", 1)
	})
	_, err := geom.Measure(deck, path)
	if err == nil {
		t.Fatal("스케일된 페이지는 에러여야 한다")
	}
	if !strings.Contains(err.Error(), "스케일") {
		t.Errorf("에러에 스케일 진단이 없다: %v", err)
	}
}

// 렌더된 HTML과 덱이 어긋나면 rect를 붙일 곳이 없다. 조용히 넘기지 않는다.
func TestMeasureRejectsDeckMismatch(t *testing.T) {
	testkit.RequireChromium(t)
	deck := browserDeck(t)
	path := renderHTML(t, deck, nil)

	extra := *deck
	extra.Slides = append(append([]*soml.Slide{}, deck.Slides...), &soml.Slide{
		Index: 2, ID: "third", Source: "slides/003_third.slide",
		Content: []soml.Node{&soml.Element{Name: soml.TextElement}},
	})
	if _, err := geom.Measure(&extra, path); err == nil {
		t.Error("HTML에 없는 슬라이드가 있는 덱은 에러여야 한다")
	}
}
