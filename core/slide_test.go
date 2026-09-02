package core

import (
	"strings"
	"testing"

	"github.com/k2sebeom/balpyo/soml"
)

// resolveLayoutSlide는 슬라이드 원문을 파싱하고 **파이프라인이 하듯** layout 값·구간을
// Slide로 옮긴 뒤 해석한다. layout 어휘 검사는 core의 몫이므로(파서는 문법만 본다) 이
// 경로가 곧 검사 자리다 — 파서 진단이 있으면 해석기를 검사하는 것이 아니므로 멈춘다.
func resolveLayoutSlide(t *testing.T, src string) (*soml.Slide, soml.Diagnostics) {
	t.Helper()
	doc, parseDiags := soml.Parse("slides/001_layout.slide", src)
	if len(parseDiags) != 0 {
		t.Fatalf("파서 진단이 없어야 한다: %v", parseDiags)
	}
	slide := &soml.Slide{
		ID:         "layout",
		Source:     doc.File,
		Layout:     soml.SlideLayout(doc.Layout),
		LayoutSpan: doc.LayoutSpan,
		Content:    doc.Content,
		Tokens:     doc.Tokens,
	}
	deck := &soml.Deck{Slides: []*soml.Slide{slide}}
	return slide, Resolve(deck)
}

// 생략·명시한 content·canvas는 진단 없이 확정 값에 도착한다. 생략은 기본값(content)으로
// 채워진다 — 렌더러가 빈 값을 보지 않고, 명시한 content와 같은 값을 갖는다.
func TestResolveSlideLayoutConfirmed(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want soml.SlideLayout
	}{
		{"생략은 content로 채운다", `<slide></slide>`, soml.LayoutContent},
		{"명시한 content", `<slide layout="content"></slide>`, soml.LayoutContent},
		{"canvas", `<slide layout="canvas"></slide>`, soml.LayoutCanvas},
		{"title", `<slide layout="title"></slide>`, soml.LayoutTitle},
		{"two-column", `<slide layout="two-column"></slide>`, soml.LayoutTwoColumn},
		{"section-divider", `<slide layout="section-divider"></slide>`, soml.LayoutSectionDivider},
		{"end", `<slide layout="end"></slide>`, soml.LayoutEnd},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			slide, diags := resolveLayoutSlide(t, tc.src)
			if len(diags) != 0 {
				t.Fatalf("진단이 없어야 한다: %v", diags)
			}
			if slide.Layout != tc.want {
				t.Errorf("Layout = %q, want %q", slide.Layout, tc.want)
			}
		})
	}
}

// 어휘 밖 값은 **line:col을 가진** 진단이다 — 구간이 값을 가리키고 허용 목록이 어휘
// 순서(SlideLayouts)로 실린다. 폴백하지 않는 것이 계약이다: `layout="titel"` 오타가
// 조용히 기본 레이아웃으로 렌더되면 닫힌 어휘가 막으려던 실패가 된다. 표준 11종은 이제
// 전부 유효하므로 여기서 에러가 되는 것은 표준 카탈로그에도 테마에도 없는 이름뿐이다
// (테마가 구현하지 않은 표준 레이아웃 → 폴백+경고는 10/03의 다른 규칙이다 — 엔진이 이름은
// 아는 상태다).
func TestResolveSlideLayoutUnknown(t *testing.T) {
	tests := []struct {
		name string
		src  string
		// 값은 `layout="…"`의 여는 인용 부호 다음 칸에서 시작한다.
		col int
	}{
		{"오타", `<slide layout="titel"></slide>`, 16},
		{"카탈로그에도 테마에도 없는 이름", `<slide layout="corp-hero"></slide>`, 16},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			slide, diags := resolveLayoutSlide(t, tc.src)
			if len(diags) != 1 {
				t.Fatalf("진단 %d개, want 1개: %v", len(diags), diags)
			}
			diag := diags[0]
			if !diag.Span.Start.IsValid() || diag.Span.Start.Line != 1 || diag.Span.Start.Col != tc.col {
				t.Errorf("위치 = %s, want 1:%d (%s)", diag.Span.Start, tc.col, diag.Message)
			}
			if diag.Severity != soml.SeverityError || !strings.Contains(diag.Message, "알 수 없는 layout") {
				t.Errorf("진단 = %q, want '알 수 없는 layout' 에러", diag.String())
			}
			// 허용 목록이 어휘 정의(soml.SlideLayouts) 순서로 실린다 — 카탈로그의 앞부분.
			if !strings.Contains(diag.Message, "title, section-divider, content, two-column") {
				t.Errorf("메시지에 허용 목록이 순서대로 없다: %q", diag.Message)
			}
			// 진단 뒤 회복 기본값은 content다 (빌드는 렌더에 닿지 않지만 상태는 결정적이다).
			if slide.Layout != soml.LayoutContent {
				t.Errorf("어휘 밖 값 뒤 Layout = %q, want content(회복 기본값)", slide.Layout)
			}
		})
	}
}

// SlideLayouts 어휘는 PRD §7.3 카탈로그 11종이고 기본값은 content다. 순서가 진단 목록의
// 순서다.
func TestSlideLayoutVocabulary(t *testing.T) {
	got := soml.SlideLayouts()
	want := []soml.SlideLayout{
		soml.LayoutTitle, soml.LayoutSectionDivider, soml.LayoutContent, soml.LayoutTwoColumn,
		soml.LayoutImageFull, soml.LayoutImageText, soml.LayoutQuote, soml.LayoutCode,
		soml.LayoutStat, soml.LayoutCanvas, soml.LayoutEnd,
	}
	if len(got) != len(want) {
		t.Fatalf("SlideLayouts = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("SlideLayouts[%d] = %q, want %q", i, got[i], want[i])
		}
	}
	if soml.DefaultSlideLayout != soml.LayoutContent {
		t.Errorf("DefaultSlideLayout = %q, want content", soml.DefaultSlideLayout)
	}
}
