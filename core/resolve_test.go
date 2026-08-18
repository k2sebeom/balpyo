package core

import (
	"os"
	"path/filepath"
	"sort"
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
		ID:      "text",
		Source:  doc.File,
		Content: doc.Content,
		Tokens:  doc.Tokens,
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
		{"색 함수", `color="gradient(fg accent1)"`, 2, 16, "아직 쓸 수 없다"},
		{"짧은 색 리터럴", `color="#12"`, 2, 16, "#RRGGBB"},
		{"값 없는 속성", `role`, 2, 9, "값이 필요하다"},
		{"빈 값", `role=""`, 2, 15, "비어 있다"},
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
	// 픽스처는 완전한 프로젝트다 (testdata/fixtures/README.md). 여기서는 파이프라인을 거치지
	// 않고 슬라이드 파일만 읽는다 — core는 internal/pipeline을 임포트할 수 없다 (임포트 순환).
	dir := filepath.Join("..", "testdata", "fixtures", "01_text", "slides")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), soml.Ext) {
			names = append(names, entry.Name())
		}
	}
	sort.Strings(names)
	if len(names) == 0 {
		t.Fatalf("%s에 슬라이드가 없다", dir)
	}

	deck := &soml.Deck{}
	for _, name := range names {
		src, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatal(err)
		}
		doc, parseDiags := soml.Parse("slides/"+name, string(src))
		if len(parseDiags) != 0 {
			t.Fatalf("%s 파서 진단: %v", name, parseDiags)
		}
		deck.Slides = append(deck.Slides, &soml.Slide{
			Index: len(deck.Slides), ID: doc.ID, Source: doc.File, Content: doc.Content, Tokens: doc.Tokens,
		})
	}
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
