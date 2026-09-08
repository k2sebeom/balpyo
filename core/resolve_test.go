package core

import (
	"strings"
	"testing"

	"github.com/k2sebeom/balpyo/soml"
)

// resolveSlide는 슬라이드 원문 하나를 파싱해 해석한다. 파서 진단이 있으면 그 테스트는
// 해석기를 검사하는 것이 아니므로 바로 멈춘다.
func resolveSlide(t *testing.T, src string) (*soml.Deck, soml.Diagnostics) {
	t.Helper()
	doc, parseDiags := soml.Parse("slides/001_text.slide", src)
	if len(parseDiags) != 0 {
		t.Fatalf("파서 진단이 없어야 한다: %v", parseDiags)
	}
	deck := &soml.Deck{Slides: []*soml.Slide{{
		ID:         "text",
		Source:     doc.File,
		Layout:     soml.SlideLayout(doc.Layout),
		LayoutSpan: doc.LayoutSpan,
		Content:    doc.Content,
		Tokens:     doc.Tokens,
	}}}
	return deck, Resolve(deck)
}

// textStyles는 덱의 모든 `<text>` 외관을 문서 순서로 모은다.
func textStyles(t *testing.T, deck *soml.Deck) []*soml.TextStyle {
	t.Helper()
	var styles []*soml.TextStyle
	var walk func(node soml.Node)
	walk = func(node soml.Node) {
		elem, ok := node.(*soml.Element)
		if !ok {
			return
		}
		if elem.Name == soml.TextElement {
			style := elem.TextStyle()
			if style == nil {
				t.Fatalf("<text>에 해석된 외관이 없다 (%s)", elem.Span)
			}
			styles = append(styles, style)
		}
		for _, child := range elem.Children {
			walk(child)
		}
	}
	for _, slide := range deck.Slides {
		for _, node := range slide.Content {
			walk(node)
		}
	}
	return styles
}

// firstStyle은 슬라이드의 첫 `<text>` 외관이다.
func firstStyle(t *testing.T, src string) *soml.TextStyle {
	t.Helper()
	deck, diags := resolveSlide(t, src)
	if len(diags) != 0 {
		t.Fatalf("진단이 없어야 한다: %v", diags)
	}
	styles := textStyles(t, deck)
	if len(styles) == 0 {
		t.Fatal("<text>가 없다")
	}
	return styles[0]
}

// findElement는 덱에서 이름이 tag인 첫 요소를 문서 순서로 찾는다.
func findElement(t *testing.T, deck *soml.Deck, tag string) *soml.Element {
	t.Helper()
	var found *soml.Element
	var walk func(node soml.Node)
	walk = func(node soml.Node) {
		elem, ok := node.(*soml.Element)
		if !ok || found != nil {
			return
		}
		if elem.Name == tag {
			found = elem
			return
		}
		for _, child := range elem.Children {
			walk(child)
		}
	}
	for _, slide := range deck.Slides {
		for _, node := range slide.Content {
			walk(node)
		}
	}
	if found == nil {
		t.Fatalf("<%s>가 없다", tag)
	}
	return found
}

// tagStyle은 슬라이드 원문에서 이름이 tag인 첫 요소의 확정 외관이다 (진단 0건을 전제한다).
func tagStyle(t *testing.T, src, tag string) *soml.TextStyle {
	t.Helper()
	deck, diags := resolveSlide(t, src)
	if len(diags) != 0 {
		t.Fatalf("진단이 없어야 한다: %v", diags)
	}
	style := findElement(t, deck, tag).TextStyle()
	if style == nil {
		t.Fatalf("<%s>에 해석된 외관이 없다", tag)
	}
	return style
}

// PRD §7.7의 등가 계약: 블록 태그는 텍스트 역할의 축약이다 — 같은 계약의 두 표기가
// **정확히** 같은 확정 외관에 도달해야 한다 (깊은 비교). 태그별 컴포넌트 기본값이 있는
// h3·blockquote는 별도 테스트(TestResolveTagComponentDefaults)가 다룬다.
func TestResolveBlockRoleEquivalence(t *testing.T) {
	cases := []struct {
		tag  string
		role soml.TextRole
	}{
		{"h1", soml.RoleTitle},
		{"h2", soml.RoleHeading},
		{"p", soml.RoleBody},
		{"pre", soml.RoleCode},
	}
	for _, tc := range cases {
		t.Run(tc.tag, func(t *testing.T) {
			got := *tagStyle(t, "<slide>\n  <"+tc.tag+">내용</"+tc.tag+">\n</slide>\n", tc.tag)
			want := *firstStyle(t, "<slide>\n  <text role=\""+string(tc.role)+"\">내용</text>\n</slide>\n")
			if got != want {
				t.Errorf("<%s> = %+v, want <text role=%q>와 동일 %+v", tc.tag, got, tc.role, want)
			}
		})
	}
}

// 태그별 컴포넌트 기본값 — 역할만으로 부족한 자리는 테마의 태그 기본값이 채운다 (부분 패치).
func TestResolveTagComponentDefaults(t *testing.T) {
	// h3: 역할은 heading이되 크기 한 단 아래 (28 — 근거는 core/theme.go).
	h3 := tagStyle(t, "<slide>\n  <h3>소제</h3>\n</slide>\n", "h3")
	heading := textRoleDefaults(soml.RoleHeading)
	if h3.Role != soml.RoleHeading || h3.Size != 28 {
		t.Errorf("h3 = %+v, want heading 역할 + size 28", *h3)
	}
	if h3.Size >= heading.Size || h3.Size <= textRoleDefaults(soml.RoleBody).Size {
		t.Errorf("h3 크기 %v는 body(24)와 heading(%v) 사이여야 한다", h3.Size, heading.Size)
	}
	// 크기 외의 필드는 heading 기본값 그대로다 — 태그 기본값은 부분 패치다.
	if h3.Font != heading.Font || h3.Weight != heading.Weight || h3.LineHeight != heading.LineHeight ||
		h3.Color != heading.Color || h3.Align != heading.Align {
		t.Errorf("h3가 정하지 않은 필드가 heading 기본값을 잃었다: %+v", *h3)
	}

	// blockquote: 역할은 body, 색만 muted — 도형 외관(왼쪽 괘선)이 없는 지금은 색으로 구별한다.
	bq := tagStyle(t, "<slide>\n  <blockquote>인용</blockquote>\n</slide>\n", "blockquote")
	body := textRoleDefaults(soml.RoleBody)
	if bq.Role != soml.RoleBody || bq.Color != soml.TokenColor("muted") {
		t.Errorf("blockquote = %+v, want body 역할 + muted 색", *bq)
	}
	if bq.Size != body.Size || bq.Font != body.Font || bq.Weight != body.Weight ||
		bq.LineHeight != body.LineHeight || bq.Align != body.Align {
		t.Errorf("blockquote가 정하지 않은 필드가 body 기본값을 잃었다: %+v", *bq)
	}
}

// 인라인 속성은 블록에서도 캐스케이드의 최상단이다 — 태그 기본값을 이긴다.
func TestResolveInlineBeatsTagDefault(t *testing.T) {
	got := tagStyle(t, "<slide>\n  <h3 size=\"40\">소제</h3>\n</slide>\n", "h3")
	if got.Size != 40 {
		t.Errorf("Size = %v, want 40 (인라인 > 태그 기본값)", got.Size)
	}
	// color·align도 <text>와 같은 규칙으로 얹힌다.
	bq := tagStyle(t, "<slide>\n  <blockquote color=\"accent1\" align=\"center\">인용</blockquote>\n</slide>\n", "blockquote")
	if bq.Color != soml.TokenColor("accent1") || bq.Align != soml.AlignCenter {
		t.Errorf("blockquote = %+v, want 인라인 색·정렬", *bq)
	}
}

// variant는 블록 태그에서도 캐스케이드 둘째 단으로 동작한다 —
// 인라인 > variant > 태그 기본값 > 역할 순서를 h3(태그 기본값 28)에서 확인한다.
func TestResolveVariantOnBlockTags(t *testing.T) {
	// lead(size 32)가 body 역할 기본값(24)을 이긴다.
	p := tagStyle(t, "<slide>\n  <p variant=\"lead\">본문</p>\n</slide>\n", "p")
	if p.Size != 32 {
		t.Errorf("<p variant=lead> Size = %v, want 32", p.Size)
	}
	// variant(32) > 태그 기본값(28).
	h3 := tagStyle(t, "<slide>\n  <h3 variant=\"lead\">소제</h3>\n</slide>\n", "h3")
	if h3.Size != 32 {
		t.Errorf("<h3 variant=lead> Size = %v, want 32 (variant > 태그 기본값)", h3.Size)
	}
	// 인라인(40) > variant(32) > 태그 기본값(28).
	h3in := tagStyle(t, "<slide>\n  <h3 variant=\"lead\" size=\"40\">소제</h3>\n</slide>\n", "h3")
	if h3in.Size != 40 {
		t.Errorf("<h3 variant=lead size=40> Size = %v, want 40 (인라인 > variant)", h3in.Size)
	}
}

// li는 body 역할로 확정된다 — 리스트 렌더 매핑(항목 05·07)이 없어도 해석은 여기서 끝난다.
// 컨테이너(<ul>/<ol>)는 외관을 갖지 않는다: 마커·들여쓰기 같은 리스트 표현은 렌더 매핑의
// 몫이고, 타이포그래피는 <li>에 붙는다.
func TestResolveListItemRole(t *testing.T) {
	deck, diags := resolveSlide(t, "<slide>\n  <ul><li>하나</li><li>둘</li></ul>\n</slide>\n")
	if len(diags) != 0 {
		t.Fatalf("진단이 없어야 한다: %v", diags)
	}
	li := findElement(t, deck, "li").TextStyle()
	if li == nil {
		t.Fatal("<li>에 해석된 외관이 없다")
	}
	want := *firstStyle(t, "<slide>\n  <text>내용</text>\n</slide>\n")
	if *li != want {
		t.Errorf("<li> = %+v, want <text>(body 기본값)와 동일 %+v", *li, want)
	}
	if ul := findElement(t, deck, "ul"); ul.Appearance != nil {
		t.Errorf("<ul>은 외관을 갖지 않아야 한다: %+v", ul.Appearance)
	}
}

// 블록의 인라인 속성은 <text>와 같은 값 층을 지난다 — 진단도 같은 정밀도(값의 line:col)다.
func TestResolveBlockValueDiagnostics(t *testing.T) {
	_, diags := resolveSlide(t, "<slide>\n  <h2 color=\"brand\">제목</h2>\n</slide>\n")
	if len(diags) != 1 {
		t.Fatalf("진단 %d개, want 1개: %v", len(diags), diags)
	}
	diag := diags[0]
	if diag.Severity != soml.SeverityError || !strings.Contains(diag.Message, "테마에 없다") {
		t.Errorf("진단 = %q, want 색 토큰 에러", diag.String())
	}
	if diag.Span.Start.Line != 2 || diag.Span.Start.Col != 14 {
		t.Errorf("위치 = %s, want 2:14 (값을 가리켜야 한다)", diag.Span.Start)
	}
}

func TestResolveVariantBeatsRoleDefault(t *testing.T) {
	// kicker는 기본 역할(body: size 24, weight 400, fg)과 구별되는 값 세 개를 정한다.
	got := firstStyle(t, "<slide>\n  <text variant=\"kicker\">내용</text>\n</slide>\n")
	if got.Size != 18 || got.Weight != 700 || got.Color != soml.TokenColor("accent1") {
		t.Errorf("variant가 역할 기본값을 이기지 못했다: %+v", *got)
	}
	// variant가 말하지 않은 것은 역할 기본값이 남는다 — 부분성.
	if got.Role != soml.RoleBody || got.Font != soml.FontBody || got.LineHeight != 1.5 || got.Align != soml.AlignStart {
		t.Errorf("variant가 정하지 않은 필드가 역할 기본값을 잃었다: %+v", *got)
	}
}

func TestResolveVariantIsPartial(t *testing.T) {
	// lead는 size만 정한다 — color·align·weight는 아래 단(역할 기본값)에서 온다.
	got := firstStyle(t, "<slide>\n  <text role=\"title\" variant=\"lead\">내용</text>\n</slide>\n")
	if got.Size != 32 {
		t.Errorf("Size = %v, want 32 (variant lead)", got.Size)
	}
	if got.Role != soml.RoleTitle || got.Font != soml.FontHeading || got.Weight != 700 ||
		got.Color != soml.TokenColor("fg") || got.Align != soml.AlignStart {
		t.Errorf("variant가 정하지 않은 필드는 title 기본값이어야 한다: %+v", *got)
	}
}

func TestResolveInlineBeatsVariant(t *testing.T) {
	got := firstStyle(t, "<slide>\n  <text variant=\"kicker\" size=\"40\" color=\"fg\">내용</text>\n</slide>\n")
	// 인라인이 정한 필드는 variant를 이긴다.
	if got.Size != 40 || got.Color != soml.TokenColor("fg") {
		t.Errorf("인라인 속성이 variant를 이기지 못했다: %+v", *got)
	}
	// 인라인이 말하지 않은 필드는 variant가 남는다.
	if got.Weight != 700 {
		t.Errorf("인라인이 정하지 않은 필드가 variant 값을 잃었다: %+v", *got)
	}
}

func TestResolveUnknownVariantWarnsAndFallsBack(t *testing.T) {
	// 모르는 variant는 에러가 아니라 경고 + 컴포넌트 기본 외관 폴백이다 (PRD §13-6) —
	// 다른 테마로 만든 콘텐츠가 이 테마에서도 빌드되어야 한다.
	deck, diags := resolveSlide(t, "<slide>\n  <text variant=\"hero\">내용</text>\n</slide>\n")
	if len(diags) != 1 {
		t.Fatalf("진단 %d개, want 1개: %v", len(diags), diags)
	}
	diag := diags[0]
	if diag.Severity != soml.SeverityWarning {
		t.Errorf("severity = %s, want warning — 에러면 테마 간 이식성이 깨진다", diag.Severity)
	}
	if diag.Span.Start.Line != 2 || diag.Span.Start.Col != 18 {
		t.Errorf("위치 = %s, want 2:18 (값을 가리켜야 한다)", diag.Span.Start)
	}
	// 경고 문구에는 테마가 가진 variant 목록이 실린다 (색 토큰 진단과 같은 태도).
	if !strings.Contains(diag.Message, `variant "hero"`) || !strings.Contains(diag.Message, "kicker, lead") {
		t.Errorf("경고 문구 = %q, want variant 이름과 테마의 variant 목록 포함", diag.Message)
	}
	if diags.HasErrors() {
		t.Error("경고만 있는 빌드는 실패하면 안 된다")
	}
	// 폴백: variant 없는 해석과 같은 결과여야 한다.
	got := *textStyles(t, deck)[0]
	want := *firstStyle(t, "<slide>\n  <text>내용</text>\n</slide>\n")
	if got != want {
		t.Errorf("폴백 외관 = %+v, want variant 없는 해석과 동일 %+v", got, want)
	}
}

func TestResolveUnknownVariantKeepsInlineAttrs(t *testing.T) {
	// variant가 틀려도 인라인 속성은 계속 해석한다 — 역할이 틀렸을 때와 같은 정책.
	deck, diags := resolveSlide(t, "<slide>\n  <text variant=\"hero\" align=\"center\">내용</text>\n</slide>\n")
	if len(diags) != 1 || diags[0].Severity != soml.SeverityWarning {
		t.Fatalf("경고 1개여야 한다: %v", diags)
	}
	if got := textStyles(t, deck)[0]; got.Align != soml.AlignCenter {
		t.Errorf("외관 = %+v, want 인라인 정렬 유지", *got)
	}
}

func TestResolveAppliesRoleDefaults(t *testing.T) {
	// 역할 기본값은 렌더러가 아무 속성 없이도 그릴 수 있는 완전한 외관이어야 한다.
	cases := []struct {
		role soml.TextRole
		want soml.TextStyle
	}{
		{soml.RoleTitle, soml.TextStyle{Role: soml.RoleTitle, Font: soml.FontHeading, Size: 56, Weight: 700,
			LineHeight: 1.2, Color: soml.TokenColor("fg"), Align: soml.AlignStart}},
		{soml.RoleHeading, soml.TextStyle{Role: soml.RoleHeading, Font: soml.FontHeading, Size: 36, Weight: 700,
			LineHeight: 1.3, Color: soml.TokenColor("fg"), Align: soml.AlignStart}},
		{soml.RoleBody, soml.TextStyle{Role: soml.RoleBody, Font: soml.FontBody, Size: 24, Weight: 400,
			LineHeight: 1.5, Color: soml.TokenColor("fg"), Align: soml.AlignStart}},
		{soml.RoleCaption, soml.TextStyle{Role: soml.RoleCaption, Font: soml.FontBody, Size: 18, Weight: 400,
			LineHeight: 1.4, Color: soml.TokenColor("muted"), Align: soml.AlignStart}},
		{soml.RoleCode, soml.TextStyle{Role: soml.RoleCode, Font: soml.FontMono, Size: 20, Weight: 400,
			LineHeight: 1.5, Color: soml.TokenColor("fg"), Align: soml.AlignStart}},
	}
	for _, tc := range cases {
		t.Run(string(tc.role), func(t *testing.T) {
			got := firstStyle(t, "<slide>\n  <text role=\""+string(tc.role)+"\">내용</text>\n</slide>\n")
			if *got != tc.want {
				t.Errorf("외관 = %+v, want %+v", *got, tc.want)
			}
		})
	}
}

func TestResolveDefaultsToBodyRole(t *testing.T) {
	// `<p>` ≡ `<text role="body">` 등가(PRD §7.7)의 다른 쪽 끝 — role 없는 <text>는 본문이다.
	got := firstStyle(t, "<slide>\n  <text>내용</text>\n</slide>\n")
	if got.Role != soml.RoleBody || got.Size != 24 {
		t.Errorf("외관 = %+v, want body 기본값", *got)
	}
}

func TestResolveInlineBeatsRoleDefault(t *testing.T) {
	got := firstStyle(t, "<slide>\n  <text role=\"caption\" size=\"40\" color=\"accent1\" align=\"center\">내용</text>\n</slide>\n")
	if got.Size != 40 || got.Color != soml.TokenColor("accent1") || got.Align != soml.AlignCenter {
		t.Errorf("인라인 속성이 역할 기본값을 이기지 못했다: %+v", *got)
	}
	// 인라인이 말하지 않은 것은 역할 기본값이 남는다 — 캐스케이드는 속성 단위로 병합된다.
	if got.Role != soml.RoleCaption || got.Weight != 400 || got.LineHeight != 1.4 || got.Font != soml.FontBody {
		t.Errorf("지정하지 않은 속성이 역할 기본값을 잃었다: %+v", *got)
	}
}

func TestResolveKeepsTokenColorAsReference(t *testing.T) {
	deck, diags := resolveSlide(t, "<slide>\n  <text color=\"accent1\">내용</text>\n</slide>\n")
	if len(diags) != 0 {
		t.Fatalf("진단이 없어야 한다: %v", diags)
	}
	got := textStyles(t, deck)[0].Color
	if !got.IsToken() || got.Token != "accent1" {
		t.Errorf("색이 토큰 참조로 남아야 한다: %+v", got)
	}
	// 토큰 색을 RGB로 눕히면 PPTX의 schemeClr 참조를 복원할 수 없다 — 내보낸 파일이
	// PowerPoint의 테마 색 편집을 따라가지 못한다 (PRD §9.4).
	if got.RGB != "" {
		t.Errorf("토큰 색에 리터럴 RGB가 들어갔다: %+v", got)
	}
	// 값은 덱에 실린 토큰 표에서 찾는다 — 웹 CSS 변수와 PPTX clrScheme의 공통 출처다.
	token, ok := deck.Theme.Color(got.Token)
	if !ok || token.RGB == "" || token.Ref == "" {
		t.Errorf("토큰 %q를 덱의 테마 표에서 찾을 수 없다: %+v", got.Token, token)
	}
}

func TestResolveLiteralColor(t *testing.T) {
	cases := []struct{ value, want string }{
		{"#0F65F9", "0F65F9"},
		{"#0f65f9", "0F65F9"}, // 대문자로 정규화 — OOXML val과 골든 비교가 표기에 흔들리지 않는다
		{"#abc", "AABBCC"},    // #RGB는 CSS와 같은 규칙으로 늘린다
	}
	for _, tc := range cases {
		t.Run(tc.value, func(t *testing.T) {
			got := firstStyle(t, "<slide>\n  <text color=\""+tc.value+"\">내용</text>\n</slide>\n").Color
			if got.IsToken() || got.RGB != tc.want {
				t.Errorf("색 = %+v, want RGB %s", got, tc.want)
			}
		})
	}
}

func TestResolveValueDiagnostics(t *testing.T) {
	// 값 층의 진단은 파서와 같은 정밀도를 가져야 한다 — 문제가 된 **값**을 line:col로 가리킨다.
	cases := []struct {
		name     string
		attrs    string
		line     int
		col      int
		contains string
	}{
		{"오타 role", `role="titel"`, 2, 15, "알 수 없는 role"},
		{"오타 align", `align="centre"`, 2, 16, "알 수 없는 align"},
		{"물리 정렬 이름", `align="left"`, 2, 16, "논리 이름"},
		{"size 단위 접미사", `size="12pt"`, 2, 15, "단위 없는 논리 px"},
		{"size 백분율", `size="50%"`, 2, 15, "PPTX 폰트 크기는 절대값"},
		{"size 0", `size="0"`, 2, 15, "0보다 커야"},
		{"size 비수치", `size="큼"`, 2, 15, "수치가 아니다"},
		{"알 수 없는 색 토큰", `color="brand"`, 2, 16, "테마에 없다"},
		{"알 수 없는 색 함수", `color="glow(fg accent1)"`, 2, 16, "알 수 없는 색 함수"},
		{"짧은 색 리터럴", `color="#12"`, 2, 16, "#RRGGBB"},
		{"값 없는 속성", `role`, 2, 9, "값이 필요하다"},
		{"빈 값", `role=""`, 2, 15, "비어 있다"},
		// variant의 "값 없음"은 다른 외관 속성과 같은 에러다 — 모르는 variant(경고)와 달리
		// 뜻을 정할 수 없는 형태이기 때문이다.
		{"값 없는 variant", `variant`, 2, 9, "값이 필요하다"},
		{"빈 variant", `variant=""`, 2, 18, "비어 있다"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, diags := resolveSlide(t, "<slide>\n  <text "+tc.attrs+">내용</text>\n</slide>\n")
			if len(diags) != 1 {
				t.Fatalf("진단 %d개, want 1개: %v", len(diags), diags)
			}
			diag := diags[0]
			if !diag.Span.Start.IsValid() || diag.Span.Start.Line != tc.line || diag.Span.Start.Col != tc.col {
				t.Errorf("위치 = %s, want %d:%d (%s)", diag.Span.Start, tc.line, tc.col, diag.Message)
			}
			if diag.Severity != soml.SeverityError || !strings.Contains(diag.Message, tc.contains) {
				t.Errorf("진단 = %q, want %q 포함", diag.String(), tc.contains)
			}
		})
	}
}

func TestResolveReportsEveryBadValueInSourceOrder(t *testing.T) {
	// 값 하나가 틀렸다고 멈추면 저자가 빌드를 오타 수만큼 돌게 된다. 파서와 같은 정책이다.
	_, diags := resolveSlide(t, "<slide>\n  <text role=\"titel\" size=\"12pt\" color=\"brand\">내용</text>\n</slide>\n")
	if len(diags) != 3 {
		t.Fatalf("진단 %d개, want 3개: %v", len(diags), diags)
	}
	for i := 1; i < len(diags); i++ {
		if diags[i-1].Span.Start.Offset > diags[i].Span.Start.Offset {
			t.Errorf("진단이 소스 순서가 아니다: %v", diags)
		}
	}
}

func TestResolveContinuesAfterBadRole(t *testing.T) {
	// 역할이 틀려도 나머지 속성은 계속 해석한다 — 진단은 한 번에 다 나와야 한다.
	deck, diags := resolveSlide(t, "<slide>\n  <text role=\"titel\" align=\"center\">내용</text>\n</slide>\n")
	if len(diags) != 1 {
		t.Fatalf("진단 %d개, want 1개: %v", len(diags), diags)
	}
	got := textStyles(t, deck)[0]
	if got.Role != soml.RoleBody || got.Align != soml.AlignCenter {
		t.Errorf("외관 = %+v, want 기본 역할 + 인라인 정렬", *got)
	}
}

func TestResolveAttachesThemeToDeck(t *testing.T) {
	deck := &soml.Deck{}
	if diags := Resolve(deck); len(diags) != 0 {
		t.Errorf("진단이 없어야 한다: %v", diags)
	}
	// 렌더러는 덱에 실린 표에서만 토큰 값을 얻는다 — 표가 비어 있으면 색을 찾을 곳이 없다.
	if deck.Theme.Name == "" || len(deck.Theme.Colors) == 0 || len(deck.Theme.Fonts) == 0 {
		t.Errorf("테마 표가 덱에 실리지 않았다: %+v", deck.Theme)
	}
}

// clrSchemeSlots는 OOXML 테마 파트의 clrScheme가 갖는 자리 전체다 (ECMA-376).
var clrSchemeSlots = map[string]bool{
	"dk1": true, "lt1": true, "dk2": true, "lt2": true,
	"accent1": true, "accent2": true, "accent3": true, "accent4": true, "accent5": true, "accent6": true,
	"hlink": true, "folHlink": true,
}

func TestThemeColorTokensMapToSchemeSlots(t *testing.T) {
	// 이 테스트가 못 박는 계약: **모든** 색 토큰이 clrScheme 슬롯을 갖는다. 슬롯 없는 토큰을
	// 만들면 그 색만 PowerPoint의 테마 색 편집을 따라가지 못해 덱 안에서 색 하나가 어긋난다.
	theme := minimalTheme()
	slots := map[string]string{}
	names := map[string]bool{}
	for _, token := range theme.Colors {
		t.Run(token.Name, func(t *testing.T) {
			if !clrSchemeSlots[token.Slot] {
				t.Errorf("clrScheme 슬롯이 아니다: %q", token.Slot)
			}
			if token.Ref == "" {
				t.Errorf("schemeClr 참조 이름이 없다: %+v", token)
			}
			if len(token.RGB) != 6 || strings.ToUpper(token.RGB) != token.RGB ||
				strings.TrimLeft(token.RGB, "0123456789ABCDEF") != "" {
				t.Errorf("RGB는 대문자 6자리 16진수여야 한다: %q", token.RGB)
			}
			if first, dup := slots[token.Slot]; dup {
				t.Errorf("슬롯 %q를 %q와 나눠 쓴다 — 한 슬롯에 두 토큰이면 하나가 지워진다", token.Slot, first)
			}
			slots[token.Slot] = token.Name
			if names[token.Name] {
				t.Errorf("토큰 이름이 중복됐다: %q", token.Name)
			}
			names[token.Name] = true
		})
	}
	if len(theme.Colors) == 0 {
		t.Fatal("색 토큰이 하나도 없다")
	}
}

func TestThemeCoversRoleDefaults(t *testing.T) {
	// 역할 기본값이 테마에 없는 토큰을 가리키면 렌더러가 색·폰트를 찾지 못한다. 해석은
	// 성공하고 렌더만 비어 보이는 종류의 실패라 테스트로 막는다.
	theme := minimalTheme()
	for _, role := range soml.TextRoles() {
		style := textRoleDefaults(role)
		if !style.Color.IsToken() {
			t.Errorf("%s의 기본 색은 토큰이어야 한다: %+v", role, style.Color)
			continue
		}
		if _, ok := theme.Color(style.Color.Token); !ok {
			t.Errorf("%s가 테마에 없는 색 토큰 %q를 가리킨다", role, style.Color.Token)
		}
		font, ok := theme.Font(style.Font)
		if !ok {
			t.Errorf("%s가 테마에 없는 폰트 토큰 %q를 가리킨다", role, style.Font)
			continue
		}
		if len(font.Family()) < 2 {
			t.Errorf("%s의 폰트 스택에 폴백이 없다: %v", role, font.Family())
		}
		if style.Size <= 0 || style.Weight <= 0 || style.LineHeight <= 0 || style.Align == "" {
			t.Errorf("%s의 기본값이 비어 있다: %+v", role, style)
		}
	}
}

func TestThemeFontTokensMapToFontScheme(t *testing.T) {
	// fontScheme에는 major/minor 두 자리뿐이다. 슬롯을 나눠 쓰면 한쪽이 지워지고, mono처럼
	// 자리가 없는 폰트는 렌더러가 리터럴 typeface로 낸다 (soml.FontToken.Scheme).
	seen := map[string]soml.FontRole{}
	for _, token := range minimalFonts() {
		switch token.Scheme {
		case "major", "minor":
			if first, dup := seen[token.Scheme]; dup {
				t.Errorf("fontScheme 슬롯 %q를 %q와 %q가 나눠 쓴다", token.Scheme, first, token.Name)
			}
			seen[token.Scheme] = token.Name
		case "":
		default:
			t.Errorf("%q는 fontScheme 슬롯이 아니다 (major/minor 또는 빈 문자열)", token.Scheme)
		}
		if token.Latin == "" || token.Generic == "" {
			t.Errorf("%s 폰트 토큰에 패밀리나 제네릭 폴백이 없다: %+v", token.Name, token)
		}
	}
	if len(seen) != 2 {
		t.Errorf("major·minor 슬롯이 모두 채워져야 한다: %v", seen)
	}
}

func TestResolveTextFixture(t *testing.T) {
	// 픽스처 읽기는 readFixtureDeck 한 곳이다 (layout_test.go) — 같은 일을 하는 코드가
	// 두 벌이 되면 한쪽만 고쳐진 채 남는다.
	deck := readFixtureDeck(t, "01_text")
	if diags := Resolve(deck); len(diags) != 0 {
		t.Fatalf("픽스처는 진단 없이 해석되어야 한다: %v", diags)
	}

	styles := textStyles(t, deck)
	if len(styles) != 7 {
		t.Fatalf("<text> %d개, want 7개", len(styles))
	}
	// 002_overview의 캡션 — 논리 정렬 이름 end와 muted 토큰이 그대로 도달한다.
	caption := styles[5]
	if caption.Role != soml.RoleCaption || caption.Size != 18 ||
		caption.Align != soml.AlignEnd || caption.Color != soml.TokenColor("muted") {
		t.Errorf("캡션 외관 = %+v", *caption)
	}
	// 001_title의 표지 제목 — 인라인 size가 역할 기본값(56)을 이긴다.
	if title := styles[1]; title.Role != soml.RoleTitle || title.Size != 64 || title.Align != soml.AlignCenter {
		t.Errorf("표지 제목 외관 = %+v", *title)
	}
}
