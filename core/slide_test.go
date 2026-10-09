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
	return slide, Resolve(deck, nil)
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

// SlideLayouts 어휘는 레이아웃 카탈로그(docs/reference/layouts.md) 11종이고 기본값은
// content다. 순서가 진단 목록의 순서다.
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

// `<slide justify>`는 흐름 모드 슬라이드의 세로 분배를 저자에게 주는 채널이다 (ADR 0044 D17).
// 어휘는 컨테이너와 **같은 6종**이다 — 같은 뜻에 두 이름을 두지 않는다 (soml.BoxJustifies).
func TestResolveSlideJustify(t *testing.T) {
	for _, want := range soml.BoxJustifies() {
		deck, diags := resolveSlide(t, `<slide justify="`+string(want)+`"><p>x</p></slide>`)
		if len(diags) != 0 {
			t.Fatalf("justify=%q에 진단이 났다: %v", want, diags)
		}
		if got := deck.Slides[0].Justify; got != want {
			t.Errorf("Justify = %q, want %q", got, want)
		}
	}
	// 속성이 없으면 빈 값이다 — 선언이 없는 슬라이드의 산출물이 이 기능 때문에 움직이지 않는다.
	deck, diags := resolveSlide(t, `<slide><p>x</p></slide>`)
	if len(diags) != 0 {
		t.Fatalf("진단이 없어야 한다: %v", diags)
	}
	if got := deck.Slides[0].Justify; got != "" {
		t.Errorf("justify 없는 슬라이드의 Justify = %q, want \"\"", got)
	}
}

// 어휘 밖 값과 `space-` 접두는 컨테이너의 justify와 **같은 진단**을 받는다 — 두 자리가 같은
// 어휘를 쓰므로 진단도 같아야 저자가 한 번 배운 것을 다시 배우지 않는다.
func TestResolveSlideJustifyUnknown(t *testing.T) {
	_, diags := resolveSlide(t, `<slide justify="middle"><p>x</p></slide>`)
	if len(diags) != 1 {
		t.Fatalf("진단 %d개, want 1개: %v", len(diags), diags)
	}
	if diags[0].Severity != soml.SeverityError || !strings.Contains(diags[0].Message, "알 수 없는 justify") {
		t.Errorf("진단 = %q, want '알 수 없는 justify' 에러", diags[0].String())
	}
	if !strings.Contains(diags[0].Message, "between") {
		t.Errorf("메시지에 어휘 목록이 없다: %q", diags[0].Message)
	}
	// `space-` 접두는 접두 없는 이름을 짚어 준다.
	_, diags = resolveSlide(t, `<slide justify="space-between"><p>x</p></slide>`)
	if len(diags) != 1 || !strings.Contains(diags[0].Message, "접두 없는 이름") {
		t.Errorf("진단 = %v, want '접두 없는 이름'", diags)
	}
}

// **슬롯 모드 슬라이드의 justify는 에러다** (ADR 0044 D17). 두 근거가 겹친다: 그 문맥의 세로
// 리듬은 테마의 `*.slots.css`가 소유하고(align-content), grid에서 `justify-content`는 **가로
// 축**이라 같은 표기가 문맥마다 다른 축을 뜻하게 된다. 조용히 무시하면 저자는 왜 아무 일도
// 일어나지 않는지(또는 왜 가로로 움직이는지) 알 수 없다.
func TestResolveSlideJustifyInSlotMode(t *testing.T) {
	deck, diags := resolveSlide(t, `<slide layout="content" justify="between"><h1 slot="title">T</h1><p slot="body">x</p></slide>`)
	if len(diags) != 1 {
		t.Fatalf("진단 %d개, want 1개: %v", len(diags), diags)
	}
	diag := diags[0]
	if diag.Severity != soml.SeverityError || !strings.Contains(diag.Message, "슬롯") {
		t.Errorf("진단 = %q, want 슬롯 모드 에러", diag.String())
	}
	if !diag.Span.Start.IsValid() {
		t.Error("진단에 위치가 없다")
	}
	if got := deck.Slides[0].Justify; got != "" {
		t.Errorf("슬롯 모드의 Justify = %q, want \"\" (테마가 소유한다)", got)
	}
	// 슬롯 자체는 그대로 확정된다 — justify 하나 때문에 슬라이드 전체가 흐름으로 떨어지지 않는다.
	if got := slotOf(t, deck, "h1"); got != "title" {
		t.Errorf("슬롯 = %q, want \"title\"", got)
	}
}
