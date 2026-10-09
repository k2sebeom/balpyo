package core

import (
	"strings"
	"testing"

	"github.com/k2sebeom/balpyo/soml"
)

// resolveSlide는 슬라이드 원문 하나를 파싱해 해석한다. 파서 진단이 있으면 해석기 검사가
// 아니므로 바로 멈춘다.
func resolveSlide(t *testing.T, src string) (*soml.Deck, soml.Diagnostics) {
	t.Helper()
	doc, parseDiags := soml.Parse("slides/001_text.slide", src)
	if len(parseDiags) != 0 {
		t.Fatalf("파서 진단이 없어야 한다: %v", parseDiags)
	}
	deck := &soml.Deck{Slides: []*soml.Slide{{
		ID:          "text",
		Source:      doc.File,
		Layout:      soml.SlideLayout(doc.Layout),
		LayoutSpan:  doc.LayoutSpan,
		Justify:     soml.BoxJustify(doc.Justify),
		JustifySpan: doc.JustifySpan,
		Content:     doc.Content,
		Tokens:      doc.Tokens,
	}}}
	return deck, Resolve(deck, nil)
}

// oneDiag는 진단이 정확히 하나임을 요구하고 그 진단을 낸다.
func oneDiag(t *testing.T, src string) (*soml.Deck, soml.Diagnostic) {
	t.Helper()
	deck, diags := resolveSlide(t, src)
	if len(diags) != 1 {
		t.Fatalf("진단 %d개, want 1개: %v", len(diags), diags)
	}
	return deck, diags[0]
}

// assertDiag는 에러 진단이 line:col(값의 자리)을 가리키고 문구 조각을 모두 담는지 본다.
// line이 0이면 위치는 보지 않는다.
func assertDiag(t *testing.T, diag soml.Diagnostic, line, col int, contains ...string) {
	t.Helper()
	if diag.Severity != soml.SeverityError {
		t.Errorf("severity = %s, want error: %s", diag.Severity, diag.Message)
	}
	if line != 0 && (!diag.Span.Start.IsValid() || diag.Span.Start.Line != line || diag.Span.Start.Col != col) {
		t.Errorf("위치 = %s, want %d:%d (%s)", diag.Span.Start, line, col, diag.Message)
	}
	for _, want := range contains {
		if !strings.Contains(diag.Message, want) {
			t.Errorf("진단 = %q, want %q 포함", diag.Message, want)
		}
	}
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

// tagStyle은 이름이 tag인 첫 요소의 확정 외관이다 (진단 0건을 전제한다).
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

// 역할 등가 계약: 블록 태그는 텍스트 역할의 축약이라 정확히 같은 외관에 도달한다.
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

// 텍스트 캐스케이드 우선순위: 인라인 > variant > 태그 기본값 > 역할 기본값. 각 단은 부분 패치라
// 기대값은 역할 기본값에 그 단이 정한 필드만 얹은 전체 외관이다 (말하지 않은 필드는 아래 단).
// 값은 단마다 구별된다: kicker(18/700/accent1)·lead(32)·h3(28)·blockquote(muted).
func TestResolveTextCascadePrecedence(t *testing.T) {
	cases := []struct {
		name, src, tag string
		role           soml.TextRole
		patch          func(*soml.TextStyle)
	}{
		{"role 없는 text는 body", `<text>x</text>`, "text", soml.RoleBody, func(*soml.TextStyle) {}},
		{"태그 기본값 h3", `<h3>x</h3>`, "h3", soml.RoleHeading, func(s *soml.TextStyle) { s.Size = 28 }},
		{"태그 기본값 blockquote", `<blockquote>x</blockquote>`, "blockquote", soml.RoleBody,
			func(s *soml.TextStyle) { s.Color = soml.TokenColor("muted") }},
		{"variant > 역할", `<text variant="kicker">x</text>`, "text", soml.RoleBody, func(s *soml.TextStyle) {
			s.Size, s.Weight, s.Color = 18, 700, soml.TokenColor("accent1")
		}},
		{"variant는 부분 패치", `<text role="title" variant="lead">x</text>`, "text", soml.RoleTitle,
			func(s *soml.TextStyle) { s.Size = 32 }},
		{"variant > 역할 (블록)", `<p variant="lead">x</p>`, "p", soml.RoleBody, func(s *soml.TextStyle) { s.Size = 32 }},
		{"variant > 태그 기본값", `<h3 variant="lead">x</h3>`, "h3", soml.RoleHeading, func(s *soml.TextStyle) { s.Size = 32 }},
		{"인라인 > variant", `<text variant="kicker" size="40" color="fg">x</text>`, "text", soml.RoleBody,
			func(s *soml.TextStyle) { s.Size, s.Weight = 40, 700 }},
		{"인라인 > variant > 태그 기본값", `<h3 variant="lead" size="40">x</h3>`, "h3", soml.RoleHeading,
			func(s *soml.TextStyle) { s.Size = 40 }},
		{"인라인 > 역할", `<text role="caption" size="40" color="accent1" align="center">x</text>`, "text", soml.RoleCaption,
			func(s *soml.TextStyle) { s.Size, s.Color, s.Align = 40, soml.TokenColor("accent1"), soml.AlignCenter }},
		{"인라인 > 태그 기본값", `<h3 size="40">x</h3>`, "h3", soml.RoleHeading, func(s *soml.TextStyle) { s.Size = 40 }},
		{"인라인 > 태그 기본값 (색·정렬)", `<blockquote color="accent1" align="center">x</blockquote>`, "blockquote", soml.RoleBody,
			func(s *soml.TextStyle) { s.Color, s.Align = soml.TokenColor("accent1"), soml.AlignCenter }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := *tagStyle(t, "<slide>\n  "+tc.src+"\n</slide>\n", tc.tag)
			want := textRoleDefaults(tc.role)
			tc.patch(&want)
			if got != want {
				t.Errorf("외관 = %+v\nwant %+v", got, want)
			}
		})
	}
}

// 역할 기본값은 속성 없이도 그릴 수 있는 완전한 외관이다 (값은 TestLoadBuiltinComponents가 고정).
func TestResolveAppliesRoleDefaults(t *testing.T) {
	for _, role := range soml.TextRoles() {
		got := *firstStyle(t, "<slide>\n  <text role=\""+string(role)+"\">내용</text>\n</slide>\n")
		if want := textRoleDefaults(role); got != want || got.Size <= 0 {
			t.Errorf("%s 외관 = %+v, want %+v", role, got, want)
		}
	}
}

// li는 body 역할로 확정되고, 컨테이너(<ul>/<ol>)는 외관을 갖지 않는다.
func TestResolveListItemRole(t *testing.T) {
	deck, diags := resolveSlide(t, "<slide>\n  <ul><li>하나</li><li>둘</li></ul>\n</slide>\n")
	if len(diags) != 0 {
		t.Fatalf("진단이 없어야 한다: %v", diags)
	}
	li := findElement(t, deck, "li").TextStyle()
	if li == nil || *li != textRoleDefaults(soml.RoleBody) {
		t.Errorf("<li> = %+v, want body 기본값", li)
	}
	if ul := findElement(t, deck, "ul"); ul.Appearance != nil {
		t.Errorf("<ul>은 외관을 갖지 않아야 한다: %+v", ul.Appearance)
	}
}

// 모르는 variant는 경고 + 컴포넌트 기본 외관 폴백이다 (테마 간 이식성). 인라인
// 속성은 계속 해석한다.
func TestResolveUnknownVariantWarnsAndFallsBack(t *testing.T) {
	deck, diags := resolveSlide(t, "<slide>\n  <text variant=\"hero\" align=\"center\">내용</text>\n</slide>\n")
	if len(diags) != 1 {
		t.Fatalf("진단 %d개, want 1개: %v", len(diags), diags)
	}
	diag := diags[0]
	if diag.Severity != soml.SeverityWarning || diags.HasErrors() {
		t.Errorf("severity = %s, want warning — 에러면 테마 간 이식성이 깨진다", diag.Severity)
	}
	if diag.Span.Start.Line != 2 || diag.Span.Start.Col != 18 {
		t.Errorf("위치 = %s, want 2:18 (값을 가리켜야 한다)", diag.Span.Start)
	}
	if !strings.Contains(diag.Message, `variant "hero"`) || !strings.Contains(diag.Message, "kicker, lead") {
		t.Errorf("경고 문구 = %q, want variant 이름과 테마의 variant 목록 포함", diag.Message)
	}
	want := textRoleDefaults(soml.RoleBody)
	want.Align = soml.AlignCenter
	if got := *textStyles(t, deck)[0]; got != want {
		t.Errorf("폴백 외관 = %+v, want %+v", got, want)
	}
}

// 토큰 색은 참조로 남는다 — RGB로 눕히면 PPTX schemeClr를 복원할 수 없다.
func TestResolveKeepsTokenColorAsReference(t *testing.T) {
	deck, diags := resolveSlide(t, "<slide>\n  <text color=\"accent1\">내용</text>\n</slide>\n")
	if len(diags) != 0 {
		t.Fatalf("진단이 없어야 한다: %v", diags)
	}
	got := textStyles(t, deck)[0].Color
	if !got.IsToken() || got.Token != "accent1" || got.RGB != "" {
		t.Errorf("색이 토큰 참조로 남아야 한다: %+v", got)
	}
	token, ok := deck.Theme.Color(got.Token)
	if !ok || token.RGB == "" || token.Ref == "" {
		t.Errorf("토큰 %q를 덱의 테마 표에서 찾을 수 없다: %+v", got.Token, token)
	}
}

func TestResolveLiteralColor(t *testing.T) {
	cases := []struct{ value, want string }{
		{"#0F65F9", "0F65F9"},
		{"#0f65f9", "0F65F9"}, // 대문자로 정규화
		{"#abc", "AABBCC"},    // #RGB는 CSS와 같은 규칙으로 늘린다
	}
	for _, tc := range cases {
		got := firstStyle(t, "<slide>\n  <text color=\""+tc.value+"\">내용</text>\n</slide>\n").Color
		if got.IsToken() || got.RGB != tc.want {
			t.Errorf("%s: 색 = %+v, want RGB %s", tc.value, got, tc.want)
		}
	}
}

// 값 층의 진단은 파서와 같은 정밀도로 문제가 된 값을 line:col로 가리킨다.
func TestResolveValueDiagnostics(t *testing.T) {
	cases := []struct {
		name, el, attrs string
		line, col       int
		contains        string
	}{
		{"오타 role", "text", `role="titel"`, 2, 15, "알 수 없는 role"},
		{"오타 align", "text", `align="centre"`, 2, 16, "알 수 없는 align"},
		{"물리 정렬 이름", "text", `align="left"`, 2, 16, "논리 이름"},
		{"size 단위 접미사", "text", `size="12pt"`, 2, 15, "단위 없는 논리 px"},
		{"size 백분율", "text", `size="50%"`, 2, 15, "PPTX 폰트 크기는 절대값"},
		{"size 0", "text", `size="0"`, 2, 15, "0보다 커야"},
		{"size 비수치", "text", `size="큼"`, 2, 15, "수치가 아니다"},
		{"알 수 없는 색 토큰", "text", `color="brand"`, 2, 16, "테마에 없다"},
		{"알 수 없는 색 함수", "text", `color="glow(fg accent1)"`, 2, 16, "알 수 없는 색 함수"},
		{"짧은 색 리터럴", "text", `color="#12"`, 2, 16, "#RRGGBB"},
		{"값 없는 속성", "text", `role`, 2, 9, "값이 필요하다"},
		{"빈 값", "text", `role=""`, 2, 15, "비어 있다"},
		// variant의 값 없음은 모르는 variant(경고)와 달리 뜻을 정할 수 없어 에러다.
		{"값 없는 variant", "text", `variant`, 2, 9, "값이 필요하다"},
		{"빈 variant", "text", `variant=""`, 2, 18, "비어 있다"},
		{"블록 태그도 같은 값 층", "h2", `color="brand"`, 2, 14, "테마에 없다"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, diag := oneDiag(t, "<slide>\n  <"+tc.el+" "+tc.attrs+">내용</"+tc.el+">\n</slide>\n")
			assertDiag(t, diag, tc.line, tc.col, tc.contains)
		})
	}
}

// 값 하나가 틀려도 멈추지 않는다 — 진단은 소스 순서로 한 번에 나오고, 틀린 역할 옆의 인라인
// 속성도 해석된다.
func TestResolveReportsEveryBadValueInSourceOrder(t *testing.T) {
	_, diags := resolveSlide(t, "<slide>\n  <text role=\"titel\" size=\"12pt\" color=\"brand\">내용</text>\n</slide>\n")
	if len(diags) != 3 {
		t.Fatalf("진단 %d개, want 3개: %v", len(diags), diags)
	}
	for i := 1; i < len(diags); i++ {
		if diags[i-1].Span.Start.Offset > diags[i].Span.Start.Offset {
			t.Errorf("진단이 소스 순서가 아니다: %v", diags)
		}
	}

	deck, _ := oneDiag(t, "<slide>\n  <text role=\"titel\" align=\"center\">내용</text>\n</slide>\n")
	if got := textStyles(t, deck)[0]; got.Role != soml.RoleBody || got.Align != soml.AlignCenter {
		t.Errorf("외관 = %+v, want 기본 역할 + 인라인 정렬", *got)
	}
}

// 렌더러는 덱에 실린 표에서만 토큰 값을 얻는다.
func TestResolveAttachesThemeToDeck(t *testing.T) {
	deck := &soml.Deck{}
	if diags := Resolve(deck, nil); len(diags) != 0 {
		t.Errorf("진단이 없어야 한다: %v", diags)
	}
	if deck.Theme.Name == "" || len(deck.Theme.Colors) == 0 || len(deck.Theme.Fonts) == 0 {
		t.Errorf("테마 표가 덱에 실리지 않았다: %+v", deck.Theme)
	}
}

// 모든 색 토큰은 고유한 clrScheme 슬롯을 갖는다 — 슬롯 없는 색은 PowerPoint 테마 색 편집을
// 따라가지 못한다.
func TestThemeColorTokensMapToSchemeSlots(t *testing.T) {
	clrSchemeSlots := map[string]bool{
		"dk1": true, "lt1": true, "dk2": true, "lt2": true,
		"accent1": true, "accent2": true, "accent3": true, "accent4": true, "accent5": true, "accent6": true,
		"hlink": true, "folHlink": true,
	}
	theme := minimalTheme()
	if len(theme.Colors) == 0 {
		t.Fatal("색 토큰이 하나도 없다")
	}
	slots := map[string]string{}
	names := map[string]bool{}
	for _, token := range theme.Colors {
		if !clrSchemeSlots[token.Slot] || token.Ref == "" {
			t.Errorf("%s: 슬롯 %q · 참조 %q", token.Name, token.Slot, token.Ref)
		}
		if len(token.RGB) != 6 || strings.TrimLeft(token.RGB, "0123456789ABCDEF") != "" {
			t.Errorf("%s: RGB는 대문자 6자리 16진수여야 한다: %q", token.Name, token.RGB)
		}
		if first, dup := slots[token.Slot]; dup {
			t.Errorf("슬롯 %q를 %q와 %q가 나눠 쓴다", token.Slot, first, token.Name)
		}
		slots[token.Slot] = token.Name
		if names[token.Name] {
			t.Errorf("토큰 이름이 중복됐다: %q", token.Name)
		}
		names[token.Name] = true
	}
}

// fontScheme에는 major/minor 두 자리뿐이다 — 나눠 쓰면 한쪽이 지워지고, 자리 없는 폰트(mono)는
// 렌더러가 리터럴 typeface로 낸다.
func TestThemeFontTokensMapToFontScheme(t *testing.T) {
	seen := map[string]soml.FontRole{}
	for _, token := range minimalTheme().Fonts {
		switch token.Scheme {
		case "major", "minor":
			if first, dup := seen[token.Scheme]; dup {
				t.Errorf("fontScheme 슬롯 %q를 %q와 %q가 나눠 쓴다", token.Scheme, first, token.Name)
			}
			seen[token.Scheme] = token.Name
		case "":
		default:
			t.Errorf("%q는 fontScheme 슬롯이 아니다", token.Scheme)
		}
		if token.Latin == "" || token.Generic == "" {
			t.Errorf("%s 폰트 토큰에 패밀리나 제네릭 폴백이 없다: %+v", token.Name, token)
		}
	}
	if len(seen) != 2 {
		t.Errorf("major·minor 슬롯이 모두 채워져야 한다: %v", seen)
	}
}
