package core

import (
	"strings"
	"testing"

	"github.com/k2sebeom/balpyo/soml"
)

// 인라인 태그는 자기 박스를 갖지 않는다 — 감싼 블록의 확정 스타일을 복사해 태그의 델타를
// 얹은 런 서식이 된다 (PRD §7.7, §9.4). 이 표는 태그별 델타가 **그 필드만** 바꾸는지
// 확인한다: want가 부모 스타일의 변형 함수라서, 델타 밖의 필드가 바뀌면 깊은 비교가 잡는다.
func TestResolveInlineDeltas(t *testing.T) {
	cases := []struct {
		name string
		src  string
		tag  string
		want func(parent soml.TextStyle) soml.TextStyle
	}{
		{"strong은 굵기만", "<slide>\n  <p><strong>굵게</strong></p>\n</slide>\n", "strong",
			func(s soml.TextStyle) soml.TextStyle { s.Weight = 700; return s }},
		{"em은 이탤릭만", "<slide>\n  <p><em>기울임</em></p>\n</slide>\n", "em",
			func(s soml.TextStyle) soml.TextStyle { s.Italic = true; return s }},
		// code는 폰트만 mono로 — size는 부모를 따른다 (code 역할의 20px를 끌어오지 않는다).
		{"code는 폰트만", "<slide>\n  <p><code>코드</code></p>\n</slide>\n", "code",
			func(s soml.TextStyle) soml.TextStyle { s.Font = soml.FontMono; return s }},
		// a의 링크 색은 accent1 토큰 재사용이다 — PPTX 테마 파트의 hlink 슬롯이 accent1에서
		// 빌려 채워지므로 (render/pptx/theme.go) 두 산출물의 링크 색이 같은 표에서 나온다.
		{"a는 밑줄과 링크 색", "<slide>\n  <p><a href=\"https://example.com\">링크</a></p>\n</slide>\n", "a",
			func(s soml.TextStyle) soml.TextStyle {
				s.Underline = true
				s.Color = soml.TokenColor("accent1")
				return s
			}},
		// 속성 없는 span은 부모 그대로다 — 해석 흔적만 남는 무해한 요소.
		{"빈 span은 부모 그대로", "<slide>\n  <p><span>스팬</span></p>\n</slide>\n", "span",
			func(s soml.TextStyle) soml.TextStyle { return s }},
		{"span color", "<slide>\n  <p><span color=\"#0f65f9\">색</span></p>\n</slide>\n", "span",
			func(s soml.TextStyle) soml.TextStyle { s.Color = soml.LiteralColor("0F65F9"); return s }},
		{"span weight", "<slide>\n  <p><span weight=\"bold\">굵게</span></p>\n</slide>\n", "span",
			func(s soml.TextStyle) soml.TextStyle { s.Weight = 700; return s }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			deck, diags := resolveSlide(t, tc.src)
			if len(diags) != 0 {
				t.Fatalf("진단이 없어야 한다: %v", diags)
			}
			parent := findElement(t, deck, "p").TextStyle()
			if parent == nil {
				t.Fatal("<p>에 확정 외관이 없다")
			}
			got := findElement(t, deck, tc.tag).TextStyle()
			if got == nil {
				t.Fatalf("<%s>에 확정 외관이 없다", tc.tag)
			}
			if want := tc.want(*parent); *got != want {
				t.Errorf("<%s> = %+v, want %+v", tc.tag, *got, want)
			}
		})
	}
}

// 상속은 확정 스타일의 복사다 — 중첩된 인라인은 바깥 인라인의 확정 스타일 위에 자기 델타를
// 얹는다. 캐스케이드는 블록에서 끝났고 variant·역할이 인라인에 다시 적용되지 않는다.
func TestResolveInlineInheritance(t *testing.T) {
	body := textRoleDefaults(soml.RoleBody)
	cases := []struct {
		name string
		src  string
		tag  string // 확정 스타일을 검사할 가장 안쪽 요소
		want func(s soml.TextStyle) soml.TextStyle
	}{
		{"strong 안의 em", "<slide>\n  <p><strong><em>x</em></strong></p>\n</slide>\n", "em",
			func(s soml.TextStyle) soml.TextStyle { s.Weight = 700; s.Italic = true; return s }},
		// 역순 중첩 — 순서와 무관하게 같은 확정 스타일에 도달한다.
		{"em 안의 strong", "<slide>\n  <p><em><strong>x</strong></em></p>\n</slide>\n", "strong",
			func(s soml.TextStyle) soml.TextStyle { s.Weight = 700; s.Italic = true; return s }},
		{"span color 안의 strong", "<slide>\n  <p><span color=\"accent1\"><strong>x</strong></span></p>\n</slide>\n", "strong",
			func(s soml.TextStyle) soml.TextStyle { s.Color = soml.TokenColor("accent1"); s.Weight = 700; return s }},
		// 블록의 인라인 속성(캐스케이드 최상단)이 인라인 상속의 출발점이 된다.
		{"블록 인라인 속성 아래", "<slide>\n  <p color=\"muted\"><strong>x</strong></p>\n</slide>\n", "strong",
			func(s soml.TextStyle) soml.TextStyle { s.Color = soml.TokenColor("muted"); s.Weight = 700; return s }},
		// 중첩 em의 토글(HTML 시각 관례)은 하지 않는다 — v1의 강조는 이탤릭으로 닫는다.
		{"em 안의 em", "<slide>\n  <p><em>a<em>b</em></em></p>\n</slide>\n", "em",
			func(s soml.TextStyle) soml.TextStyle { s.Italic = true; return s }},
		{"li 안의 em", "<slide>\n  <ul><li><em>x</em></li></ul>\n</slide>\n", "em",
			func(s soml.TextStyle) soml.TextStyle { s.Italic = true; return s }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := tagStyle(t, tc.src, tc.tag)
			if want := tc.want(body); *got != want {
				t.Errorf("<%s> = %+v, want %+v", tc.tag, *got, want)
			}
		})
	}
}

// 부모가 이미 700 이상이면 strong은 그대로 둔다 — 더 굵은 단계는 없다.
func TestResolveStrongKeepsBolderParent(t *testing.T) {
	got := *tagStyle(t, "<slide>\n  <h1><strong>x</strong></h1>\n</slide>\n", "strong")
	want := *tagStyle(t, "<slide>\n  <h1>제목</h1>\n</slide>\n", "h1")
	if got != want {
		t.Errorf("<h1> 안의 <strong> = %+v, want h1 확정 스타일 그대로 %+v", got, want)
	}
}

// code의 size는 부모를 따른다 — title 안의 code는 56px mono다. code 역할(20px)을 끌어오면
// 한 줄 안에서 글자 크기가 섞여 웹 줄 박스와 PPTX 절대 줄간격(spcPts)이 갈라진다.
func TestResolveCodeInheritsSize(t *testing.T) {
	got := tagStyle(t, "<slide>\n  <text role=\"title\"><code>x</code></text>\n</slide>\n", "code")
	if got.Size != 56 || got.Font != soml.FontMono {
		t.Errorf("title 안의 code = %+v, want Size 56 + mono", *got)
	}
	if got.Role != soml.RoleTitle {
		t.Errorf("Role = %s, want 부모의 역할 유지 (역할은 인라인에 다시 적용되지 않는다)", got.Role)
	}
}

// 하이퍼링크 목적지는 스타일이 아니다 — Appearance.Link에 원문 그대로 실린다.
func TestResolveAnchorLink(t *testing.T) {
	deck, diags := resolveSlide(t, "<slide>\n  <p><a href=\"https://example.com/x\">링크</a></p>\n</slide>\n")
	if len(diags) != 0 {
		t.Fatalf("진단이 없어야 한다: %v", diags)
	}
	a := findElement(t, deck, "a")
	if a.Appearance == nil || a.Appearance.Link != "https://example.com/x" {
		t.Fatalf("Appearance = %+v, want Link 원문 유지", a.Appearance)
	}
	style := a.TextStyle()
	if style == nil || !style.Underline || style.Color != soml.TokenColor("accent1") {
		t.Errorf("<a> 스타일 = %+v, want 밑줄 + accent1 토큰", style)
	}
	// 링크가 아닌 인라인에는 Link가 실리지 않는다.
	deck, _ = resolveSlide(t, "<slide>\n  <p><strong>x</strong></p>\n</slide>\n")
	if strong := findElement(t, deck, "strong"); strong.Appearance.Link != "" {
		t.Errorf("<strong>에 Link가 실렸다: %q", strong.Appearance.Link)
	}
}

// href 값의 URL 문법은 검증하지 않는다 — 파일·앵커·상대 경로 전부 유효할 수 있고, 잘못된
// URL은 클릭이 실패할 뿐 빌드가 막을 수 있는 실패가 아니다.
func TestResolveAnchorHrefIsOpenValue(t *testing.T) {
	for _, href := range []string{"assets/intro.pdf", "#section-2", "mailto:x@y.z", "not a url"} {
		t.Run(href, func(t *testing.T) {
			deck, diags := resolveSlide(t, "<slide>\n  <p><a href=\""+href+"\">링크</a></p>\n</slide>\n")
			if len(diags) != 0 {
				t.Fatalf("진단이 없어야 한다: %v", diags)
			}
			if got := findElement(t, deck, "a").Appearance.Link; got != href {
				t.Errorf("Link = %q, want %q", got, href)
			}
		})
	}
}

// 없는/빈 href는 에러다 — 목적지 없는 링크는 뜻을 정할 수 없다 (hasValue와 같은 태도).
func TestResolveAnchorHrefErrors(t *testing.T) {
	cases := []struct {
		name     string
		src      string
		line     int
		col      int
		contains string
	}{
		{"href 없음", "<slide>\n  <p><a>링크</a></p>\n</slide>\n", 2, 7, "href가 필요하다"},
		{"값 없는 href", "<slide>\n  <p><a href>링크</a></p>\n</slide>\n", 2, 9, "값이 필요하다"},
		{"빈 href", "<slide>\n  <p><a href=\"\">링크</a></p>\n</slide>\n", 2, 15, "비어 있다"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			deck, diags := resolveSlide(t, tc.src)
			if len(diags) != 1 {
				t.Fatalf("진단 %d개, want 1개: %v", len(diags), diags)
			}
			diag := diags[0]
			if diag.Severity != soml.SeverityError || !strings.Contains(diag.Message, tc.contains) {
				t.Errorf("진단 = %q, want %q 포함", diag.String(), tc.contains)
			}
			if diag.Span.Start.Line != tc.line || diag.Span.Start.Col != tc.col {
				t.Errorf("위치 = %s, want %d:%d", diag.Span.Start, tc.line, tc.col)
			}
			// href가 틀려도 런 스타일은 확정된다 — 진단은 한 번에 다 나와야 하고, 밑줄·색은
			// 목적지와 무관하게 정해진다.
			if style := findElement(t, deck, "a").TextStyle(); style == nil || !style.Underline {
				t.Errorf("href 에러 후에도 <a>의 스타일은 확정되어야 한다: %+v", style)
			}
		})
	}
}

// span의 color 값 어휘 진단은 <text color>와 문구가 같아야 한다 — 값 해석 코드를 공유한다는
// 계약의 회귀 테스트다 (두 벌이 되면 진단이 갈라진다).
func TestResolveSpanColorSharesTextDiagnostics(t *testing.T) {
	_, spanDiags := resolveSlide(t, "<slide>\n  <p><span color=\"brand\">x</span></p>\n</slide>\n")
	_, textDiags := resolveSlide(t, "<slide>\n  <text color=\"brand\">x</text>\n</slide>\n")
	if len(spanDiags) != 1 || len(textDiags) != 1 {
		t.Fatalf("진단이 각 1개여야 한다: span=%v text=%v", spanDiags, textDiags)
	}
	if spanDiags[0].Message != textDiags[0].Message {
		t.Errorf("span 진단 %q ≠ text 진단 %q — 값 해석 코드가 갈라졌다",
			spanDiags[0].Message, textDiags[0].Message)
	}
	// 위치는 span의 값을 가리킨다.
	if spanDiags[0].Span.Start.Line != 2 || spanDiags[0].Span.Start.Col != 19 {
		t.Errorf("위치 = %s, want 2:19", spanDiags[0].Span.Start)
	}
}

// weight의 어휘는 normal(400)·bold(700) 둘이다 — OOXML의 굵기는 켬/끔(rPr@b)뿐이라
// CSS의 수치 축을 열면 중간 굵기가 웹에서만 동작한다.
func TestResolveSpanWeightVocabulary(t *testing.T) {
	// normal은 상속된 굵기를 되돌릴 수 있다 — strong 안에서 400.
	got := tagStyle(t, "<slide>\n  <p><strong><span weight=\"normal\">x</span></strong></p>\n</slide>\n", "span")
	if got.Weight != 400 {
		t.Errorf("strong 안의 <span weight=normal> = %d, want 400", got.Weight)
	}
	bold := tagStyle(t, "<slide>\n  <p><span weight=\"bold\">x</span></p>\n</slide>\n", "span")
	if bold.Weight != 700 {
		t.Errorf(`<span weight="bold"> = %d, want 700`, bold.Weight)
	}
}

func TestResolveSpanWeightDiagnostics(t *testing.T) {
	cases := []struct {
		name     string
		attrs    string
		line     int
		col      int
		contains string
	}{
		// CSS 습관의 수치는 대응하는 이름을 짚어 준다 (align의 left/right와 같은 태도).
		{"수치 400", `weight="400"`, 2, 20, `"normal"`},
		{"수치 700", `weight="700"`, 2, 20, `"bold"`},
		{"중간 굵기", `weight="600"`, 2, 20, "두 단계"},
		{"알 수 없는 이름", `weight="semibold"`, 2, 20, "알 수 없는 weight"},
		{"값 없는 weight", `weight`, 2, 12, "값이 필요하다"},
		{"빈 weight", `weight=""`, 2, 20, "비어 있다"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, diags := resolveSlide(t, "<slide>\n  <p><span "+tc.attrs+">x</span></p>\n</slide>\n")
			if len(diags) != 1 {
				t.Fatalf("진단 %d개, want 1개: %v", len(diags), diags)
			}
			diag := diags[0]
			if diag.Severity != soml.SeverityError || !strings.Contains(diag.Message, tc.contains) {
				t.Errorf("진단 = %q, want %q 포함", diag.String(), tc.contains)
			}
			if diag.Span.Start.Line != tc.line || diag.Span.Start.Col != tc.col {
				t.Errorf("위치 = %s, want %d:%d", diag.Span.Start, tc.line, tc.col)
			}
		})
	}
}

// 해석 후 인라인 요소 전부의 Appearance가 non-nil이어야 렌더러의 nil 가드가 실제로
// "core를 거치지 않은 덱"만 잡는다. 유일한 예외는 <br> — void이고 텍스트 흐름의 분절점일
// 뿐이라 서식을 갖지 않는다.
func TestResolveInlineAppearanceNonNil(t *testing.T) {
	src := "<slide>\n  <p>본문 <strong>굵게 <em>기울임</em></strong> <code>코드</code>" +
		" <a href=\"#x\">링크</a> <span>스팬</span>줄<br/>끝</p>\n</slide>\n"
	deck, diags := resolveSlide(t, src)
	if len(diags) != 0 {
		t.Fatalf("진단이 없어야 한다: %v", diags)
	}
	seen := map[string]bool{}
	var walk func(node soml.Node)
	walk = func(node soml.Node) {
		elem, ok := node.(*soml.Element)
		if !ok {
			return
		}
		if soml.KindOf(elem.Name) == soml.KindInline {
			seen[elem.Name] = true
			if elem.Name == soml.BreakElement {
				if elem.Appearance != nil {
					t.Errorf("<br>은 서식이 아니다 — Appearance가 비어 있어야 한다: %+v", elem.Appearance)
				}
			} else if elem.Appearance == nil || elem.Appearance.Text == nil {
				t.Errorf("<%s>의 Appearance가 채워지지 않았다", elem.Name)
			}
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
	for _, name := range []string{"strong", "em", "code", "a", "span", "br"} {
		if !seen[name] {
			t.Errorf("테스트 원문에 <%s>가 없다 — 커버리지가 줄었다", name)
		}
	}
}
