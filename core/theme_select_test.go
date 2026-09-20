package core

import (
	"strings"
	"testing"
	"testing/fstest"

	"github.com/k2sebeom/balpyo/soml"
)

// 이 파일은 **테마 선택**의 계약이다 (P4.2 · ADR 0052).
//
// 못 박는 것 셋:
//
//  1. 덱이 말한 이름이 값 층의 로더 선택을 바꾼다 (`Deck.ThemeName` → `deck.Theme`).
//  2. 어휘 밖 이름은 **에러**다 — 폴백 테마로 조용히 계속하지 않는다.
//  3. 내장 테마는 셋 다 **같은 계약**을 받는다 (팔레트·레이아웃·슬롯·variant 어휘·본문 리듬).
//
// 3번이 테스트인 이유는 이식성이다: 저자가 테마 이름 한 줄만 바꿨을 때 덱이 다르게 보이는 것은
// 제품이지만, 덱이 **깨지는** 것은 버그다.

// themeDeck은 같은 원문을 주어진 테마 이름으로 해석한다 — 테마 선택만 다른 두 결과를 견주는
// 자리라 슬라이드 원문은 호출자가 한 벌만 갖는다.
func themeDeck(t *testing.T, name, src string) (*soml.Deck, soml.Diagnostics) {
	t.Helper()
	doc, parseDiags := soml.Parse("slides/001_text.slide", src)
	if len(parseDiags) != 0 {
		t.Fatalf("파서 진단이 없어야 한다: %v", parseDiags)
	}
	deck := &soml.Deck{ThemeName: name, Slides: []*soml.Slide{{
		ID: "text", Source: doc.File,
		Layout: soml.SlideLayout(doc.Layout), LayoutSpan: doc.LayoutSpan,
		Content: doc.Content, Tokens: doc.Tokens,
	}}}
	return deck, Resolve(deck, nil)
}

const themeProbe = `<slide id="probe"><h1>제목</h1><p>본문</p></slide>`

// TestResolveUsesDeckThemeName은 덱의 테마 요청이 로더 선택을 바꾼다는 것을 못 박는다.
// 이것이 P4.2가 이은 배선이다 — 그전에는 `balpyo.json`의 theme 값이 검사만 받고 버려졌다.
func TestResolveUsesDeckThemeName(t *testing.T) {
	deck, diags := themeDeck(t, "ink", themeProbe)
	if len(diags) != 0 {
		t.Fatalf("내장 테마를 골랐는데 진단이 났다: %v", diags)
	}
	if deck.Theme.Name != "ink" {
		t.Errorf("deck.Theme.Name = %q, want ink — 요청한 테마가 결과에 닿지 않았다", deck.Theme.Name)
	}
}

// TestResolveEmptyThemeNameUsesDefault는 테마를 말하지 않은 덱이 기본 테마를 받음을 못 박는다.
// `Deck`을 설정 없이 만드는 길(테스트·프리뷰)이 있으므로 빈 값은 에러가 아니다.
func TestResolveEmptyThemeNameUsesDefault(t *testing.T) {
	deck, diags := themeDeck(t, "", themeProbe)
	if len(diags) != 0 {
		t.Fatalf("진단이 없어야 한다: %v", diags)
	}
	if deck.Theme.Name != DefaultTheme {
		t.Errorf("deck.Theme.Name = %q, want %q", deck.Theme.Name, DefaultTheme)
	}
}

// TestResolveUnknownThemeIsAnError는 어휘 밖 이름이 **에러 + 해석 중단**임을 못 박는다
// (ADR 0052).
//
// 모르는 variant와 규칙이 다른 자리다: variant 어휘는 테마의 사유물이라 경고 + 폴백이지만,
// 테마 **이름**의 어휘는 엔진이 소유하므로(BuiltinThemes) 목록 밖 이름은 오타이거나 설치되지
// 않은 테마다. 조용히 minimal로 그리면 저자는 "내 테마가 왜 안 먹지"를 슬라이드에서 찾는다.
func TestResolveUnknownThemeIsAnError(t *testing.T) {
	deck, diags := themeDeck(t, "corp", themeProbe)
	if len(diags) != 1 {
		t.Fatalf("진단 %d개, want 1: %v", len(diags), diags)
	}
	diag := diags[0]
	if diag.Severity != soml.SeverityError {
		t.Errorf("심각도 = %v, want error — 폴백 테마는 없다", diag.Severity)
	}
	if diag.File != ConfigFile {
		t.Errorf("진단 파일 = %q, want %q — 고칠 자리는 설정이다", diag.File, ConfigFile)
	}
	for _, want := range append([]string{"corp"}, BuiltinThemes()...) {
		if !strings.Contains(diag.Message, want) {
			t.Errorf("진단이 %q를 말하지 않는다: %s", want, diag.Message)
		}
	}
	// 해석은 **중단**된다 — 폴백 테마로 계속하면 산출물이 저자가 고르지 않은 테마로 나간다.
	if deck.Theme.Name != "" {
		t.Errorf("deck.Theme.Name = %q, want 빈 값 — 모르는 테마로 해석이 이어졌다", deck.Theme.Name)
	}
	if elem, ok := deck.Slides[0].Content[0].(*soml.Element); ok && elem.Appearance != nil {
		t.Error("모르는 테마인데 외관이 확정되었다 — 해석이 중단되지 않았다")
	}
}

// TestBuiltinThemesRenderTheSameDeckDifferently는 "테마를 갈면 같은 덱이 다르게 보인다"를
// 값으로 확인한다 — 팔레트가 같은 테마 둘은 테마 둘이 아니다.
func TestBuiltinThemesRenderTheSameDeckDifferently(t *testing.T) {
	seen := map[string]string{}
	for _, name := range BuiltinThemes() {
		deck, diags := themeDeck(t, name, themeProbe)
		if len(diags) != 0 {
			t.Fatalf("테마 %q에서 진단이 났다: %v", name, diags)
		}
		if deck.Theme.Name != name {
			t.Errorf("테마 %q를 골랐는데 deck.Theme.Name = %q", name, deck.Theme.Name)
		}
		key := colorTokenRGB(t, deck.Theme, "bg") + "/" + colorTokenRGB(t, deck.Theme, "fg") +
			"/" + colorTokenRGB(t, deck.Theme, "accent1")
		if other, dup := seen[key]; dup {
			t.Errorf("테마 %q와 %q의 bg/fg/accent1이 같다 (%s) — 두 테마가 아니다", name, other, key)
		}
		seen[key] = name
	}
}

// TestConfigValidateChecksThemeVocabulary는 `balpyo.json`의 theme가 내장 어휘와 대조됨을 못
// 박는다. 값 층도 같은 판정을 하지만(위 테스트) 설정 검사가 **더 좋은 에러**를 준다 — 파일을
// 가리키고 쓸 수 있는 이름을 나열한다.
func TestConfigValidateChecksThemeVocabulary(t *testing.T) {
	for _, name := range BuiltinThemes() {
		cfg := DefaultConfig()
		cfg.Theme = name
		if err := cfg.Validate(); err != nil {
			t.Errorf("내장 테마 %q가 거부됐다: %v", name, err)
		}
	}
	cfg := DefaultConfig()
	cfg.Theme = "corp"
	err := cfg.Validate()
	if err == nil {
		t.Fatal("없는 테마 이름이 통과했다 — 값이 조용히 버려지던 시절의 필드가 된다")
	}
	for _, want := range append([]string{"theme", "corp"}, BuiltinThemes()...) {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("에러가 %q를 말하지 않는다: %v", want, err)
		}
	}
	// 빈 값은 기본 테마로 채운다 (설정 파일 없이 빌드되는 프로젝트가 있다).
	empty := DefaultConfig()
	empty.Theme = ""
	if err := empty.Validate(); err != nil {
		t.Fatalf("빈 theme가 거부됐다: %v", err)
	}
	if empty.Theme != DefaultTheme {
		t.Errorf("빈 theme = %q, want %q", empty.Theme, DefaultTheme)
	}
}

// TestBuiltinThemesShareRecommendedVariants는 공식 테마 셋이 **같은 variant 어휘**를 갖는다는
// 계약이다 (PRD §13-6 · ADR 0052).
//
// variant 어휘는 테마의 사유물이지만, 공식 테마가 권장 어휘를 공유하지 않으면 테마를 간 덱의
// `variant="stat-card"`가 경고로 바뀐다 — "테마를 갈면 다르게 보인다"가 "테마를 갈면 깨진다"가
// 되는 자리다. 순서까지 같은 이유는 모르는 variant 경고가 나열하는 목록이 테마마다 다른 순서로
// 나올 이유가 없다는 것이다.
func TestBuiltinThemesShareRecommendedVariants(t *testing.T) {
	wantText := []string{"kicker", "lead"}
	wantShape := []string{"card", "card-header", "card-body", "card-backdrop", "stat-card", "badge"}
	wantImage := []string{"framed"}
	for _, name := range BuiltinThemes() {
		comps := loadBuiltinTheme(name).components
		gotText := make([]string, 0, len(comps.textVariants))
		for _, v := range comps.textVariants {
			gotText = append(gotText, v.name)
		}
		gotShape := make([]string, 0, len(comps.shapeVariants))
		for _, v := range comps.shapeVariants {
			gotShape = append(gotShape, v.name)
		}
		gotImage := make([]string, 0, len(comps.imageVariants))
		for _, v := range comps.imageVariants {
			gotImage = append(gotImage, v.name)
		}
		if strings.Join(gotText, ",") != strings.Join(wantText, ",") {
			t.Errorf("%s의 텍스트 variant = %v, want %v", name, gotText, wantText)
		}
		if strings.Join(gotShape, ",") != strings.Join(wantShape, ",") {
			t.Errorf("%s의 도형 variant = %v, want %v", name, gotShape, wantShape)
		}
		if strings.Join(gotImage, ",") != strings.Join(wantImage, ",") {
			t.Errorf("%s의 이미지 variant = %v, want %v", name, gotImage, wantImage)
		}
	}
}

// TestBuiltinThemesShareBodyRhythm은 본문이 세 테마에서 24px · 줄높이 1.5임을 못 박는다
// (ADR 0052의 결정 8). 스테이지 720px이 정확히 본문 20줄이라는 사실(ADR 0002)은 제품의 수직
// 리듬이고, 테마마다 갈리면 문서에서 그 문장이 사라진다. 테마가 가르는 것은 제목 위계·색·짜임이다.
func TestBuiltinThemesShareBodyRhythm(t *testing.T) {
	for _, name := range BuiltinThemes() {
		body, ok := loadBuiltinTheme(name).components.textRoles[soml.RoleBody]
		if !ok {
			t.Fatalf("%s에 body 역할이 없다", name)
		}
		if body.Size != 24 || body.LineHeight != 1.5 {
			t.Errorf("%s의 body = %gpx/%g, want 24px/1.5", name, body.Size, body.LineHeight)
		}
	}
}

// ── 로더의 완전성 검사 (ADR 0052의 결정 3) ──────────────────────────────────
//
// 폴백 테마를 지운 대가로 로더가 완전성을 요구한다: 필수 토큰·역할이 없으면 **테마 로드가
// 실패한다**. 그러지 않으면 테마 제작자의 누락이 산출물에서 "글자 크기가 0"으로 나타나고,
// 원인을 찾는 자리가 테마에서 멀어진다 (theme.md의 "채워야 하는 이름" 표가 이 검사다).

// fullTokens는 완전한 tokens.json이다 — 아래 테스트가 여기서 한 조각씩 뺀다.
const fullTokens = `{
  "color": [
    {"name":"fg","rgb":"000000","slot":"dk1","ref":"tx1"},
    {"name":"bg","rgb":"FFFFFF","slot":"lt1","ref":"bg1"},
    {"name":"muted","rgb":"666666","slot":"dk2","ref":"tx2"},
    {"name":"surface","rgb":"EEEEEE","slot":"lt2","ref":"bg2"},
    {"name":"accent1","rgb":"0F65F9","slot":"accent1","ref":"accent1"},
    {"name":"accent2","rgb":"0E8174","slot":"accent2","ref":"accent2"},
    {"name":"accent3","rgb":"15803D","slot":"accent3","ref":"accent3"},
    {"name":"accent4","rgb":"B45309","slot":"accent4","ref":"accent4"},
    {"name":"accent5","rgb":"DC2626","slot":"accent5","ref":"accent5"},
    {"name":"accent6","rgb":"9333EA","slot":"accent6","ref":"accent6"}
  ],
  "font": [
    {"name":"heading","latin":"Noto Sans","ea":"Noto Sans CJK KR","generic":"sans-serif","scheme":"major"},
    {"name":"body","latin":"Noto Sans","ea":"Noto Sans CJK KR","generic":"sans-serif","scheme":"minor"},
    {"name":"mono","latin":"Noto Sans Mono","ea":"Noto Sans Mono CJK KR","generic":"monospace"}
  ],
  "space": [
    {"name":"xs","px":6},{"name":"sm","px":12},{"name":"md","px":24},
    {"name":"lg","px":36},{"name":"xl","px":48},{"name":"2xl","px":72}
  ],
  "radius": [{"name":"sm","px":6},{"name":"md","px":12},{"name":"lg","px":24}],
  "shadow": [{"name":"soft","dist":4,"dir":90,"blur":16,"color":"fg","alpha":0.15}]
}`

// fullComponents는 완전한 components.json이다 (역할 5종 + 최소 도형 기본값 + code 절).
//
// 조각으로 쪼개 둔 이유는 **code 절만 겨누는 테스트**가 있다는 것이다 (highlight_test의
// themeFSWithCode): 완전성 검사를 통과해야 로더가 code 절까지 도달하므로, 그 테스트도 완전한
// 역할 표를 써야 하고 그 표를 두 번 적을 이유는 없다.
const fullComponents = "{" + fullRolesAndShape + `,
  "code": ` + fullCode + "\n}"

// componentsWithCode는 완전한 컴포넌트 표에서 code 절만 갈아 끼운다.
func componentsWithCode(code string) string {
	return "{" + fullRolesAndShape + `,
  "code": ` + code + "\n}"
}

const fullCode = `{"tabSize":2,"tokens":[{"class":"keyword","color":"accent6"}]}`

const fullRolesAndShape = `
  "text": { "roles": [
    {"name":"title","font":"heading","size":56,"weight":700,"lineHeight":1.2,"color":"fg","align":"start"},
    {"name":"heading","font":"heading","size":36,"weight":700,"lineHeight":1.3,"color":"fg","align":"start"},
    {"name":"body","font":"body","size":24,"weight":400,"lineHeight":1.5,"color":"fg","align":"start"},
    {"name":"caption","font":"body","size":18,"weight":400,"lineHeight":1.4,"color":"muted","align":"start"},
    {"name":"code","font":"mono","size":20,"weight":400,"lineHeight":1.5,"color":"fg","align":"start"}
  ]},
  "shape": { "default": {"fill":"surface","border":"1 solid muted","radius":"md"} }`

// memTheme는 메모리 위의 테마 패키지 하나다 — 로더는 fs.FS 위에서 도므로 내장 테마와 같은
// 코드를 지난다 (theme_package.go의 계약).
func memTheme(tokens, comps string) fstest.MapFS {
	return fstest.MapFS{
		"t/theme.json":          {Data: []byte(`{"name":"t","layouts":["content"]}`)},
		"t/tokens.json":         {Data: []byte(tokens)},
		"t/components.json":     {Data: []byte(comps)},
		"t/layouts/content.css": {Data: []byte("/* content */")},
	}
}

// TestLoadThemeAcceptsCompletePackage는 아래 테스트들이 무엇을 빼는지가 뜻을 갖도록 **완전한**
// 패키지가 로드됨을 먼저 못 박는다 (빼기 전의 기준선).
func TestLoadThemeAcceptsCompletePackage(t *testing.T) {
	if _, err := loadTheme(memTheme(fullTokens, fullComponents), "t"); err != nil {
		t.Fatalf("완전한 테마가 거부됐다: %v", err)
	}
}

// TestLoadThemeRequiresTokenVocabulary는 필수 토큰 이름이 빠지면 로드가 실패함을 못 박는다.
// 이름은 **저자가 속성에 쓰는 어휘**이므로(`color="accent6"`·`gap="md"`), 빠진 이름은 그 이름을
// 쓴 덱을 조용히 망가뜨린다.
func TestLoadThemeRequiresTokenVocabulary(t *testing.T) {
	cases := []struct{ name, drop, want string }{
		{"색 fg", `{"name":"fg","rgb":"000000","slot":"dk1","ref":"tx1"},`, "fg"},
		{"색 accent6", `,
    {"name":"accent6","rgb":"9333EA","slot":"accent6","ref":"accent6"}`, "accent6"},
		{"폰트 mono", `,
    {"name":"mono","latin":"Noto Sans Mono","ea":"Noto Sans Mono CJK KR","generic":"monospace"}`, "mono"},
		{"공간 md", `{"name":"md","px":24},`, "md"},
		{"반경 lg", `,{"name":"lg","px":24}`, "lg"},
		{"그림자 soft", `{"name":"soft","dist":4,"dir":90,"blur":16,"color":"fg","alpha":0.15}`, "soft"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tokens := strings.Replace(fullTokens, tc.drop, "", 1)
			if tokens == fullTokens {
				t.Fatalf("테스트 준비 오류 — %q를 fullTokens에서 찾지 못했다", tc.drop)
			}
			err := second(loadTheme(memTheme(tokens, fullComponents), "t"))
			if err == nil {
				t.Fatalf("%s가 없는데 로드가 통과했다", tc.name)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("에러가 빠진 이름 %q를 말하지 않는다: %v", tc.want, err)
			}
		})
	}
}

// TestLoadThemeRequiresTextRoles는 텍스트 역할 5종과 그 역할이 **가리키는 토큰**을 로더가
// 요구함을 못 박는다. 역할 하나가 없으면 그 역할을 쓴 블록이 크기 0·색 없음으로 렌더되고,
// 폰트·색 이름이 이 테마에 없으면 확정 외관이 해석되지 않는 참조를 담는다.
func TestLoadThemeRequiresTextRoles(t *testing.T) {
	cases := []struct{ name, from, to, want string }{
		{"역할 caption 없음",
			`{"name":"caption","font":"body","size":18,"weight":400,"lineHeight":1.4,"color":"muted","align":"start"},`,
			"", "caption"},
		{"역할의 폰트가 테마에 없다", `"font":"mono","size":20`, `"font":"display","size":20`, "display"},
		{"역할의 색이 테마에 없다", `"color":"muted","align":"start"},`, `"color":"dim","align":"start"},`, "dim"},
		{"역할의 크기가 0", `"size":24,"weight":400,"lineHeight":1.5,"color":"fg"`,
			`"size":0,"weight":400,"lineHeight":1.5,"color":"fg"`, "size"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			comps := strings.Replace(fullComponents, tc.from, tc.to, 1)
			if comps == fullComponents {
				t.Fatalf("테스트 준비 오류 — %q를 fullComponents에서 찾지 못했다", tc.from)
			}
			err := second(loadTheme(memTheme(fullTokens, comps), "t"))
			if err == nil {
				t.Fatalf("%s인데 로드가 통과했다", tc.name)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("에러가 %q를 말하지 않는다: %v", tc.want, err)
			}
		})
	}
}

// second는 (값, 에러) 쌍에서 에러만 집는 테스트 헬퍼다 — 위 표 주행이 값을 쓰지 않는다.
func second(_ themePackage, err error) error { return err }
