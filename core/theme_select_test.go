package core

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/k2sebeom/balpyo/soml"
)

// 테마 선택의 계약: 덱의 이름이 로더를 고르고, 어휘 밖 이름은 에러이며, 내장 테마는
// 같은 계약을 받는다 — 테마를 갈면 다르게 보여야 하지만 깨지면 버그다.

// themeDeck은 같은 원문을 주어진 내장 테마 이름으로 해석한다.
func themeDeck(t *testing.T, name, src string) (*soml.Deck, soml.Diagnostics) {
	t.Helper()
	return themeDeckIn(t, nil, name, src)
}

const themeProbe = `<slide id="probe"><h1>제목</h1><p>본문</p></slide>`

// 빈 이름은 기본 테마다 — 설정 없이 Deck을 만드는 길(테스트·프리뷰)이 있다.
func TestResolveEmptyThemeNameUsesDefault(t *testing.T) {
	deck, diags := themeDeck(t, "", themeProbe)
	if len(diags) != 0 {
		t.Fatalf("진단이 없어야 한다: %v", diags)
	}
	if deck.Theme.Name != DefaultTheme {
		t.Errorf("deck.Theme.Name = %q, want %q", deck.Theme.Name, DefaultTheme)
	}
}

// 테마 이름의 어휘는 엔진 소유라 목록 밖 이름은 에러 + 해석 중단이다 (variant와 달리 폴백 없음).
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
	if deck.Theme.Name != "" {
		t.Errorf("deck.Theme.Name = %q, want 빈 값 — 모르는 테마로 해석이 이어졌다", deck.Theme.Name)
	}
	if elem, ok := deck.Slides[0].Content[0].(*soml.Element); ok && elem.Appearance != nil {
		t.Error("모르는 테마인데 외관이 확정되었다 — 해석이 중단되지 않았다")
	}
}

// 요청한 테마가 결과에 닿고, 팔레트가 같은 테마 둘은 테마 둘이 아니다.
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

// 이름 판정은 프로젝트를 아는 LoadConfig의 일이다 (로컬 테마). Validate는 기본값만 채운다.
func TestLoadConfigChecksThemeVocabulary(t *testing.T) {
	write := func(t *testing.T, body string) string {
		t.Helper()
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, ConfigFile), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
		return dir
	}
	for _, name := range BuiltinThemes() {
		if _, err := LoadConfig(write(t, `{"theme":"`+name+`"}`)); err != nil {
			t.Errorf("내장 테마 %q가 거부됐다: %v", name, err)
		}
	}
	_, err := LoadConfig(write(t, `{"theme":"corp"}`))
	if err == nil {
		t.Fatal("없는 테마 이름이 통과했다")
	}
	for _, want := range append([]string{ConfigFile, "theme", "corp"}, BuiltinThemes()...) {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("에러가 %q를 말하지 않는다: %v", want, err)
		}
	}
	empty := DefaultConfig()
	empty.Theme = ""
	if err := empty.Validate(); err != nil {
		t.Fatalf("빈 theme가 거부됐다: %v", err)
	}
	if empty.Theme != DefaultTheme {
		t.Errorf("빈 theme = %q, want %q", empty.Theme, DefaultTheme)
	}
	local := DefaultConfig()
	local.Theme = "corp"
	if err := local.Validate(); err != nil {
		t.Errorf("Validate가 이름을 판정했다: %v", err)
	}
}

// 공식 테마는 같은 variant 어휘(순서까지)와 본문 리듬(24px/1.5)을 공유한다.
// 어긋나면 테마를 간 덱이 경고로 깨진다.
func TestBuiltinThemesShareContract(t *testing.T) {
	names := func(n int, at func(int) string) string {
		out := make([]string, n)
		for i := range out {
			out[i] = at(i)
		}
		return strings.Join(out, ",")
	}
	for _, name := range BuiltinThemes() {
		comps := loadBuiltinTheme(name).components
		if got := names(len(comps.textVariants), func(i int) string { return comps.textVariants[i].name }); got != "kicker,lead" {
			t.Errorf("%s의 텍스트 variant = %s", name, got)
		}
		if got := names(len(comps.shapeVariants), func(i int) string { return comps.shapeVariants[i].name }); got != "card,card-header,card-body,card-backdrop,stat-card,badge" {
			t.Errorf("%s의 도형 variant = %s", name, got)
		}
		if got := names(len(comps.imageVariants), func(i int) string { return comps.imageVariants[i].name }); got != "framed" {
			t.Errorf("%s의 이미지 variant = %s", name, got)
		}
		body, ok := comps.textRoles[soml.RoleBody]
		if !ok || body.Size != 24 || body.LineHeight != 1.5 {
			t.Errorf("%s의 body = %gpx/%g, want 24px/1.5", name, body.Size, body.LineHeight)
		}
	}
}

// ── 로더의 완전성 검사 ──
// 폴백 테마가 없으므로 필수 토큰·역할이 빠지면 로드가 실패한다 (theme.md의 "채워야 하는 이름" 표).

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

// fullComponents는 완전한 components.json이다. code 절만 겨누는 테스트(highlight_test)가
// componentsWithCode로 같은 역할 표를 재사용한다.
const fullComponents = "{" + fullRolesAndShape + `,
  "code": ` + fullCode + "\n}"

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

// memTheme는 메모리 위의 테마 패키지 하나다 — 로더는 fs.FS 위에서 돌므로 내장 테마와 같은 길이다.
func memTheme(tokens, comps string) fstest.MapFS {
	return fstest.MapFS{
		"t/theme.json":          {Data: []byte(`{"name":"t","layouts":["content"]}`)},
		"t/tokens.json":         {Data: []byte(tokens)},
		"t/components.json":     {Data: []byte(comps)},
		"t/layouts/content.css": {Data: []byte("/* content */")},
	}
}

// 완전한 패키지는 로드되고(기준선), 필수 토큰·역할·역할이 가리키는 토큰이 하나라도 빠지면
// 빠진 이름을 말하며 실패한다.
func TestLoadThemeRequiresCompleteness(t *testing.T) {
	if _, err := loadTheme(memTheme(fullTokens, fullComponents), "t"); err != nil {
		t.Fatalf("완전한 테마가 거부됐다: %v", err)
	}
	cases := []struct {
		name     string
		tokens   bool // true면 fullTokens, 아니면 fullComponents를 고친다
		from, to string
		want     string
	}{
		{"색 fg", true, `{"name":"fg","rgb":"000000","slot":"dk1","ref":"tx1"},`, "", "fg"},
		{"색 accent6", true, `,
    {"name":"accent6","rgb":"9333EA","slot":"accent6","ref":"accent6"}`, "", "accent6"},
		{"폰트 mono", true, `,
    {"name":"mono","latin":"Noto Sans Mono","ea":"Noto Sans Mono CJK KR","generic":"monospace"}`, "", "mono"},
		{"공간 md", true, `{"name":"md","px":24},`, "", "md"},
		{"반경 lg", true, `,{"name":"lg","px":24}`, "", "lg"},
		{"그림자 soft", true, `{"name":"soft","dist":4,"dir":90,"blur":16,"color":"fg","alpha":0.15}`, "", "soft"},
		{"역할 caption 없음", false,
			`{"name":"caption","font":"body","size":18,"weight":400,"lineHeight":1.4,"color":"muted","align":"start"},`,
			"", "caption"},
		{"역할의 폰트가 테마에 없다", false, `"font":"mono","size":20`, `"font":"display","size":20`, "display"},
		{"역할의 색이 테마에 없다", false, `"color":"muted","align":"start"},`, `"color":"dim","align":"start"},`, "dim"},
		{"역할의 크기가 0", false, `"size":24,"weight":400,"lineHeight":1.5,"color":"fg"`,
			`"size":0,"weight":400,"lineHeight":1.5,"color":"fg"`, "size"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tokens, comps := fullTokens, fullComponents
			target := &comps
			if tc.tokens {
				target = &tokens
			}
			before := *target
			*target = strings.Replace(before, tc.from, tc.to, 1)
			if *target == before {
				t.Fatalf("테스트 준비 오류 — %q를 찾지 못했다", tc.from)
			}
			err := second(loadTheme(memTheme(tokens, comps), "t"))
			if err == nil {
				t.Fatalf("%s인데 로드가 통과했다", tc.name)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("에러가 %q를 말하지 않는다: %v", tc.want, err)
			}
		})
	}
}

// second는 (값, 에러) 쌍에서 에러만 집는다.
func second(_ themePackage, err error) error { return err }
