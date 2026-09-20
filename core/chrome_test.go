package core

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/k2sebeom/balpyo/soml"
)

// 덱 크롬은 `balpyo.json`이 내용과 켜짐을 소유한다 (ADR 0045). 기본값은 **둘 다 꺼짐**이다 —
// 켜짐을 기본으로 두면 골든 9개와 예제 셋이 이 기능 하나로 움직인다.
func TestConfigChromeFields(t *testing.T) {
	dir := t.TempDir()
	const src = `{"title":"t","footer":"2026 Q3 · 내부용","pageNumber":true}`
	if err := os.WriteFile(filepath.Join(dir, ConfigFile), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := LoadConfig(dir)
	if err != nil {
		t.Fatalf("설정 로드 실패: %v", err)
	}
	if cfg.Footer != "2026 Q3 · 내부용" {
		t.Errorf("footer = %q, want %q", cfg.Footer, "2026 Q3 · 내부용")
	}
	if !cfg.PageNumber {
		t.Error("pageNumber = false, want true")
	}

	def := DefaultConfig()
	if def.Footer != "" || def.PageNumber {
		t.Errorf("기본 설정의 크롬 = (%q, %v), want (\"\", false)", def.Footer, def.PageNumber)
	}
}

// minimal은 두 크롬을 **다 선언한다** — 기본 테마가 못 받으면 `balpyo.json`의 두 필드가
// 경고 뒤에 버려진다. 자리·역할은 테마의 몫이고(ADR 0045), 제외 목록은 표지 계열이다.
func TestLoadBuiltinChrome(t *testing.T) {
	pkg := loadBuiltinTheme("minimal")
	decls := pkg.chromeDecls()
	if len(decls) != 2 {
		t.Fatalf("크롬 선언 %d개, want 2 (%+v)", len(decls), decls)
	}
	wantKind := []soml.ChromeKind{soml.ChromeFooter, soml.ChromePageNumber}
	wantAlign := []soml.TextAlign{soml.AlignStart, soml.AlignEnd}
	for i, decl := range decls {
		if decl.kind != wantKind[i] {
			t.Errorf("chrome[%d].kind = %q, want %q", i, decl.kind, wantKind[i])
		}
		if decl.align != wantAlign[i] {
			t.Errorf("chrome[%d].align = %q, want %q", i, decl.align, wantAlign[i])
		}
		if decl.role != soml.RoleCaption {
			t.Errorf("chrome[%d].role = %q, want %q", i, decl.role, soml.RoleCaption)
		}
		// 표지·섹션 구분·끝 장에는 번호도 푸터도 찍지 않는다.
		for _, layout := range []soml.SlideLayout{"title", "section-divider", "end"} {
			if !decl.skip[layout] {
				t.Errorf("chrome[%d]의 skipLayouts에 %q가 없다", i, layout)
			}
		}
		if decl.skip["content"] {
			t.Errorf("chrome[%d]의 skipLayouts가 content를 빼고 있다", i)
		}
	}
}

// 테마 선언의 값 어휘는 닫혀 있고, 어긋나면 **로드가 멈춘다** — 테마의 오타가 조용히 무시되면
// 저자는 "표지에 번호가 왜 찍히는가"를 자기 슬라이드에서 찾게 된다.
func TestLoadChromeRejectsBadDecls(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want string
	}{
		{
			name: "알 수 없는 kind",
			src:  `[{"kind":"logo","align":"start","role":"caption"}]`,
			want: `chrome[0]: 알 수 없는 kind "logo" (쓸 수 있는 kind: footer, pageNumber)`,
		},
		{
			name: "알 수 없는 align",
			src:  `[{"kind":"footer","align":"middle","role":"caption"}]`,
			want: `chrome[0]: 알 수 없는 align "middle" (쓸 수 있는 align: start, center, end)`,
		},
		{
			name: "justify는 크롬의 align이 아니다",
			src:  `[{"kind":"footer","align":"justify","role":"caption"}]`,
			want: `chrome[0]: 알 수 없는 align "justify" (쓸 수 있는 align: start, center, end)`,
		},
		{
			name: "align이 없다",
			src:  `[{"kind":"footer","role":"caption"}]`,
			want: `chrome[0]: align이 없다 (쓸 수 있는 align: start, center, end)`,
		},
		{
			name: "알 수 없는 role",
			src:  `[{"kind":"footer","align":"start","role":"hero"}]`,
			want: `chrome[0]: 알 수 없는 role "hero" (쓸 수 있는 role: title, heading, body, caption, code)`,
		},
		{
			name: "role이 없다",
			src:  `[{"kind":"footer","align":"start"}]`,
			want: `chrome[0]: role이 없다`,
		},
		{
			name: "제외 목록의 레이아웃이 이 테마에 없다",
			src:  `[{"kind":"footer","align":"start","role":"caption","skipLayouts":["cover"]}]`,
			want: `chrome[0]: skipLayouts의 "cover"는 이 테마의 레이아웃이 아니다`,
		},
		{
			name: "같은 kind를 두 번 선언한다",
			src: `[{"kind":"footer","align":"start","role":"caption"},` +
				`{"kind":"footer","align":"end","role":"caption"}]`,
			want: `chrome[1]: kind footer가 두 번 선언되었다`,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			fsys := themeFSWithChrome(tc.src)
			_, err := loadTheme(fsys, "t")
			if err == nil {
				t.Fatalf("잘못된 크롬 선언이 진단되지 않았다 (%s)", tc.src)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("에러 = %q, want substring %q", err.Error(), tc.want)
			}
		})
	}
}

// themeFSWithChrome은 크롬 선언만 다른 최소 테마 패키지를 만든다 — 로더가 보는 것은
// theme.json이므로 토큰·컴포넌트·레이아웃은 최소로 둔다.
func themeFSWithChrome(chrome string) fstest.MapFS {
	manifest := `{"name":"t","layouts":["content"],"chrome":` + chrome + `}`
	return fstest.MapFS{
		"t/theme.json":          {Data: []byte(manifest)},
		"t/tokens.json":         {Data: []byte(`{"color":[],"font":[],"space":[],"radius":[],"shadow":[]}`)},
		// code 절은 모든 테마가 적어야 한다 (탭 폭이 테마의 것이다 — ADR 0051).
		"t/components.json":     {Data: []byte(`{"text":{"roles":[]},"code":{"tabSize":2,"tokens":[]}}`)},
		"t/layouts/content.css": {Data: []byte("")},
	}
}

// 확정 모델에는 **걸러진 뒤의** 항목만 실린다 (ADR 0045) — 두 렌더러가 제외 규칙을 각자
// 적지 않는다. 번호는 덱 안의 위치이고 **제외된 장도 센다**: `a:fld type="slidenum"`이
// 주는 값이 그것이고, 웹이 다른 수를 찍으면 두 타깃이 갈린다.
func TestResolveChromePerSlide(t *testing.T) {
	deck := chromeDeck("title", "content", "section-divider", "content")
	deck.Chrome = soml.DeckChrome{Footer: "내부용", PageNumber: true}
	diags := Resolve(deck)
	if len(diags) != 0 {
		t.Fatalf("진단이 없어야 한다: %v", diags)
	}

	if got := len(deck.Slides[0].Chrome); got != 0 {
		t.Errorf("표지(layout=title)의 크롬 %d개, want 0", got)
	}
	if got := len(deck.Slides[2].Chrome); got != 0 {
		t.Errorf("섹션 구분의 크롬 %d개, want 0", got)
	}

	for _, tc := range []struct {
		index  int
		number string
	}{{1, "2"}, {3, "4"}} {
		items := deck.Slides[tc.index].Chrome
		if len(items) != 2 {
			t.Fatalf("슬라이드 %d의 크롬 %d개, want 2 (%+v)", tc.index, len(items), items)
		}
		// 순서는 테마 선언 순서다 (footer → pageNumber).
		if items[0].Kind != soml.ChromeFooter || items[0].Text != "내부용" {
			t.Errorf("슬라이드 %d의 crome[0] = (%q, %q), want (footer, 내부용)",
				tc.index, items[0].Kind, items[0].Text)
		}
		if items[0].Align != soml.AlignStart {
			t.Errorf("푸터 align = %q, want start", items[0].Align)
		}
		if items[1].Kind != soml.ChromePageNumber || items[1].Text != tc.number {
			t.Errorf("슬라이드 %d의 chrome[1] = (%q, %q), want (pageNumber, %q)",
				tc.index, items[1].Kind, items[1].Text, tc.number)
		}
		if items[1].Align != soml.AlignEnd {
			t.Errorf("번호 align = %q, want end", items[1].Align)
		}
		// 외관은 테마 역할에서 온다 — 크롬이 자기 크기·색을 갖지 않는다. 정렬 **하나만** 선언이
		// 덮는다: 확정 모델에 정렬이 둘이면 웹(`text-align`)과 PPTX(`algn`)가 다른 쪽을 읽는다.
		caption := loadBuiltinTheme("minimal").components.textRoles[soml.RoleCaption]
		for i, item := range items {
			want := caption
			want.Align = item.Align
			if item.Style != want {
				t.Errorf("chrome[%d].Style = %+v, want caption 역할 + align %q %+v",
					i, item.Style, item.Align, want)
			}
			if item.Style.Align != item.Align {
				t.Errorf("chrome[%d]의 정렬이 둘이다: Style.Align=%q, Align=%q",
					i, item.Style.Align, item.Align)
			}
		}
	}
}

// 켜지 않은 크롬은 확정 모델에 없다 — 이 필드 때문에 기존 덱의 산출물이 움직이지 않는다.
func TestResolveChromeOffByDefault(t *testing.T) {
	deck := chromeDeck("content", "content")
	diags := Resolve(deck)
	if len(diags) != 0 {
		t.Fatalf("진단이 없어야 한다: %v", diags)
	}
	for _, slide := range deck.Slides {
		if len(slide.Chrome) != 0 {
			t.Errorf("슬라이드 %d의 크롬 %d개, want 0", slide.Index, len(slide.Chrome))
		}
	}
}

// 푸터만 켠 덱은 번호를 얻지 않는다 (두 필드가 서로 독립이다).
func TestResolveChromeFooterOnly(t *testing.T) {
	deck := chromeDeck("content")
	deck.Chrome = soml.DeckChrome{Footer: "내부용"}
	if diags := Resolve(deck); len(diags) != 0 {
		t.Fatalf("진단이 없어야 한다: %v", diags)
	}
	items := deck.Slides[0].Chrome
	if len(items) != 1 || items[0].Kind != soml.ChromeFooter {
		t.Fatalf("크롬 = %+v, want footer 하나", items)
	}
}

// 테마가 그 kind를 선언하지 않았으면 **경고 + 미탑재**다 (ADR 0044의 규칙 그대로) — 확정
// 모델에 싣지 않으므로 두 렌더러가 테마 조건을 각자 보지 않는다. 진단은 슬라이드가 아니라
// `balpyo.json`을 가리킨다: 고칠 자리가 거기다.
func TestResolveChromeWarnsWhenThemeIsSilent(t *testing.T) {
	// 선언이 하나도 없는 테마다 — 파일을 읽지 않고 그 상태만 만든다.
	pkg := themePackage{manifest: themeManifest{Name: "t"}}
	deck := chromeDeck("content")
	deck.Chrome = soml.DeckChrome{Footer: "내부용", PageNumber: true}

	items, diags := resolveChrome(deck, pkg)
	if len(items) != 0 {
		t.Errorf("미탑재여야 하는데 항목이 실렸다: %+v", items)
	}
	if len(diags) != 2 {
		t.Fatalf("진단 %d건, want 2 (%v)", len(diags), diags)
	}
	for _, diag := range diags {
		if diag.Severity != soml.SeverityWarning {
			t.Errorf("심각도 = %v, want warning (%s)", diag.Severity, diag.Message)
		}
		if diag.File != ConfigFile {
			t.Errorf("진단 파일 = %q, want %q", diag.File, ConfigFile)
		}
	}
	if want := `테마 "t"에 footer 크롬 선언이 없어`; !strings.Contains(diags[0].Message, want) {
		t.Errorf("푸터 경고 = %q, want substring %q", diags[0].Message, want)
	}
	if want := `테마 "t"에 pageNumber 크롬 선언이 없어`; !strings.Contains(diags[1].Message, want) {
		t.Errorf("번호 경고 = %q, want substring %q", diags[1].Message, want)
	}
}

// chromeDeck은 레이아웃만 다른 빈 슬라이드 덱이다 — 크롬은 슬라이드 내용과 무관하다.
func chromeDeck(layouts ...string) *soml.Deck {
	deck := &soml.Deck{Stage: soml.Stage{Width: 1280, Height: 720}}
	for i, layout := range layouts {
		deck.Slides = append(deck.Slides, &soml.Slide{
			Index:  i,
			ID:     layout,
			Source: "slides/00" + string(rune('1'+i)) + "_x.slide",
			Layout: soml.SlideLayout(layout),
		})
	}
	return deck
}
