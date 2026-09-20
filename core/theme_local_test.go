package core

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/k2sebeom/balpyo/soml"
)

// 이 파일은 **외부 테마 패키지**의 계약이다 (P4.3 · ADR 0054).
//
// 못 박는 것 넷:
//
//  1. 테마는 여전히 **이름**이고, 그 이름을 찾는 자리가 둘이다 — 내장 테마와
//     `<프로젝트>/themes/<이름>/`. 어휘는 두 출처의 합집합이다.
//  2. 이름이 한 프로젝트에서 하나를 뜻한다 — 내장과 겹치는 로컬 테마도, 디렉토리와 다른
//     매니페스트 이름도 에러다.
//  3. `extends`의 병합 규칙은 하나다 — **이름이 같으면 자식이 이기고, 없던 이름은 부모 순서
//     뒤에 붙는다.** 이름이 없는 싱글톤 절은 필드 단위, 레이아웃 CSS는 파일 단위다.
//  4. 완전성은 **병합 결과**의 성질이다 — 상속하지 않은 로컬 테마는 내장 테마와 같은 조건을
//     받고, 상속한 테마는 부모가 바닥을 깔아 주므로 부분 파일이어도 된다.

// ── 프로젝트 하나를 메모리에 만드는 도구 ────────────────────────────────────
//
// 로더는 fs.FS 위에서 도므로 내장 테마와 프로젝트 로컬 테마가 **같은 코드**를 지난다
// (theme_package.go의 계약). 그래서 프로젝트도 MapFS 하나로 충분하다 — 디스크를 쓰는 테스트는
// `balpyo.json`을 읽는 자리(LoadConfig)에만 있다.

// localTheme는 `themes/<이름>/` 아래 파일 넷을 만든다 (빈 문자열은 파일을 만들지 않는다).
func localTheme(name, manifest, tokens, comps string, layouts map[string]string) fstest.MapFS {
	fsys := fstest.MapFS{}
	dir := "themes/" + name + "/"
	if manifest != "" {
		fsys[dir+"theme.json"] = &fstest.MapFile{Data: []byte(manifest)}
	}
	if tokens != "" {
		fsys[dir+"tokens.json"] = &fstest.MapFile{Data: []byte(tokens)}
	}
	if comps != "" {
		fsys[dir+"components.json"] = &fstest.MapFile{Data: []byte(comps)}
	}
	for file, css := range layouts {
		fsys[dir+"layouts/"+file] = &fstest.MapFile{Data: []byte(css)}
	}
	return fsys
}

// mergeFS는 여러 MapFS를 하나로 합친다 — 프로젝트에 테마가 둘 이상인 경우를 만든다.
func mergeFS(parts ...fstest.MapFS) fstest.MapFS {
	all := fstest.MapFS{}
	for _, part := range parts {
		for name, file := range part {
			all[name] = file
		}
	}
	return all
}

// standaloneLocal은 상속하지 않은 **완전한** 로컬 테마다 — 내장 테마와 같은 조건을 받는다.
func standaloneLocal(name string) fstest.MapFS {
	return localTheme(name,
		`{"name":"`+name+`","layouts":["content"]}`,
		strings.Replace(fullTokens, `"rgb":"0F65F9"`, `"rgb":"123456"`, 1),
		fullComponents,
		map[string]string{"content.css": "/* content */"})
}

// childOfMinimal은 `minimal`을 상속한 부분 패치 테마다 — 아래 병합 테스트들이 공유한다.
//
// 자식이 가진 것: 크롬 선언 하나(footer) · 색 둘(덮기 하나 + 더하기 하나) · 역할 하나(title) ·
// 텍스트 variant 둘(덮기 하나 + 더하기 하나) · 도형 기본값의 fill 하나 · 코드 토큰 하나 ·
// 레이아웃 CSS 하나. **layouts 선언과 tokens/components의 나머지는 없다** — 부모가 깐다.
func childOfMinimal() fstest.MapFS {
	return localTheme("corp",
		`{"name":"corp","extends":"minimal",
		  "chrome":[{"kind":"footer","align":"center","role":"caption","skipLayouts":["title"]}]}`,
		`{"color":[{"name":"accent1","rgb":"112233","slot":"accent1","ref":"accent1"},
		           {"name":"brand","rgb":"445566"}]}`,
		`{"text":{"roles":[{"name":"title","font":"heading","size":72,"weight":700,
		                    "lineHeight":1.1,"color":"accent1","align":"center"}],
		          "variants":[{"name":"kicker","size":20},{"name":"eyebrow","color":"accent1"}]},
		  "shape":{"default":{"fill":"brand"}},
		  "code":{"tokens":[{"class":"keyword","color":"accent1"}]}}`,
		map[string]string{"title.css": "/* corp의 표지 */\ndisplay: grid;"})
}

// ── 로컬 테마를 이름으로 찾는다 ─────────────────────────────────────────────

// TestResolveFindsProjectLocalTheme는 프로젝트의 `themes/<이름>/`이 요청의 대상이 됨을 못
// 박는다 — 이 저장소를 고쳐 다시 빌드하지 않고 자기 테마를 쓰는 길이다.
func TestResolveFindsProjectLocalTheme(t *testing.T) {
	deck, diags := themeDeckIn(t, standaloneLocal("corp"), "corp", themeProbe)
	if len(diags) != 0 {
		t.Fatalf("로컬 테마를 골랐는데 진단이 났다: %v", diags)
	}
	if deck.Theme.Name != "corp" {
		t.Errorf("deck.Theme.Name = %q, want corp — 요청이 로컬 테마에 닿지 않았다", deck.Theme.Name)
	}
	if got := colorTokenRGB(t, deck.Theme, "accent1"); got != "123456" {
		t.Errorf("accent1 = %q, want 123456 — 로컬 테마의 값이 아니다", got)
	}
}

// TestResolveWithoutFilesystemSeesBuiltinsOnly는 fsys가 nil인 길이 남아 있음을 못 박는다
// (설정 없이 Deck을 만드는 테스트·프리뷰·미래의 API).
func TestResolveWithoutFilesystemSeesBuiltinsOnly(t *testing.T) {
	if _, diags := themeDeck(t, "ink", themeProbe); len(diags) != 0 {
		t.Fatalf("fsys 없이 내장 테마가 거부됐다: %v", diags)
	}
	deck, diags := themeDeck(t, "corp", themeProbe)
	if len(diags) != 1 || deck.Theme.Name != "" {
		t.Fatalf("fsys 없이 로컬 이름이 통과했다: %v", diags)
	}
}

// TestUnknownThemeListsLocalAndBuiltinNames는 어휘가 **두 출처의 합집합**임을 진단 문구로 못
// 박는다 — 저자가 자기 테마를 오타 냈을 때 목록에 자기 테마가 보여야 고칠 수 있다.
func TestUnknownThemeListsLocalAndBuiltinNames(t *testing.T) {
	project := mergeFS(standaloneLocal("corp"), standaloneLocal("acme"))
	_, diags := themeDeckIn(t, project, "corpp", themeProbe)
	if len(diags) != 1 {
		t.Fatalf("진단 %d개, want 1: %v", len(diags), diags)
	}
	msg := diags[0].Message
	for _, want := range append([]string{"corpp", "acme", "corp"}, BuiltinThemes()...) {
		if !strings.Contains(msg, want) {
			t.Errorf("진단이 %q를 말하지 않는다: %s", want, msg)
		}
	}
	if names := strings.Join(ThemeNames(project), ","); names != "acme,corp,editorial,ink,minimal" {
		t.Errorf("ThemeNames = %s — 두 출처를 정렬해 합친 목록이어야 한다", names)
	}
}

// TestLocalThemeNameMustMatchItsDirectory는 디렉토리 이름이 요청 키임을 못 박는다.
// 둘이 갈리면 `themes/corp/`를 `theme: "acme"`로 요청해야 하고, 그 사실은 파일을 열어야 안다.
func TestLocalThemeNameMustMatchItsDirectory(t *testing.T) {
	project := localTheme("corp",
		`{"name":"acme","layouts":["content"]}`, fullTokens, fullComponents,
		map[string]string{"content.css": ""})
	_, diags := themeDeckIn(t, project, "corp", themeProbe)
	if len(diags) != 1 {
		t.Fatalf("진단 %d개, want 1: %v", len(diags), diags)
	}
	for _, want := range []string{"corp", "acme"} {
		if !strings.Contains(diags[0].Message, want) {
			t.Errorf("진단이 %q를 말하지 않는다: %s", want, diags[0].Message)
		}
	}
}

// TestLocalThemeCannotShadowABuiltin은 이름 충돌이 에러임을 못 박는다 — **요청한 테마가
// 무엇이든** 멈춘다. 섀도잉을 허용하면 `extends: "minimal"`에 답이 없다.
func TestLocalThemeCannotShadowABuiltin(t *testing.T) {
	project := standaloneLocal("minimal")
	for _, request := range []string{"minimal", "ink"} {
		_, diags := themeDeckIn(t, project, request, themeProbe)
		if len(diags) != 1 {
			t.Fatalf("%q를 요청했을 때 진단 %d개, want 1: %v", request, len(diags), diags)
		}
		if !strings.Contains(diags[0].Message, "minimal") {
			t.Errorf("진단이 충돌한 이름을 말하지 않는다: %s", diags[0].Message)
		}
	}
}

// TestLocalThemeWithoutExtendsMustBeComplete는 상속하지 않은 로컬 테마가 내장 테마와 **같은
// 조건**을 받음을 못 박는다 (ADR 0052의 완전성 검사 — 폴백 테마가 없으므로 메울 곳이 없다).
func TestLocalThemeWithoutExtendsMustBeComplete(t *testing.T) {
	project := localTheme("corp",
		`{"name":"corp","layouts":["content"]}`,
		`{"color":[{"name":"fg","rgb":"000000","slot":"dk1","ref":"tx1"}]}`,
		fullComponents,
		map[string]string{"content.css": ""})
	_, diags := themeDeckIn(t, project, "corp", themeProbe)
	if len(diags) != 1 {
		t.Fatalf("불완전한 로컬 테마가 통과했다: %v", diags)
	}
	if !strings.Contains(diags[0].Message, "bg") {
		t.Errorf("진단이 빠진 토큰을 말하지 않는다: %s", diags[0].Message)
	}
}

// ── extends: 이름 단위 덮어쓰기 ─────────────────────────────────────────────

// TestExtendsInheritsAndOverridesByName은 병합 규칙 한 줄을 표로 확인한다 (ADR 0054의 결정 7):
// 이름이 같으면 자식이 이기고, 없던 이름은 **부모 순서 뒤에** 붙는다.
//
// 순서가 규칙의 일부인 이유는 순서가 산출물에 보인다는 것이다 — `clrScheme` 슬롯, CSS 변수
// 선언, 모르는 variant 경고가 나열하는 목록.
func TestExtendsInheritsAndOverridesByName(t *testing.T) {
	pkg := mustLoadLocal(t, childOfMinimal(), "corp")
	if pkg.tokens.Name != "corp" {
		t.Errorf("테마 이름 = %q, want corp — 자식의 이름이 결과의 이름이다", pkg.tokens.Name)
	}

	// 색: 부모 10종의 순서가 남고 accent1만 값이 갈리고, 새 이름이 뒤에 붙는다.
	var names, accent1 string
	for _, token := range pkg.tokens.Colors {
		names += token.Name + " "
		if token.Name == "accent1" {
			accent1 = token.RGB
		}
	}
	wantNames := "fg bg muted surface accent1 accent2 accent3 accent4 accent5 accent6 brand "
	if names != wantNames {
		t.Errorf("색 토큰 = %q\nwant %q", names, wantNames)
	}
	if accent1 != "112233" {
		t.Errorf("accent1 = %q, want 112233 — 자식이 이기지 않았다", accent1)
	}

	// 역할: 자식이 적은 title만 갈리고 나머지 4종은 부모 것이다 (역할은 한 벌 통째로 덮는다).
	title, ok := pkg.components.textRoles[soml.RoleTitle]
	if !ok {
		t.Fatal("title 역할이 없다")
	}
	if title.Size != 72 || title.Align != soml.AlignCenter {
		t.Errorf("title = %gpx/%v, want 72px/center", title.Size, title.Align)
	}
	body, ok := pkg.components.textRoles[soml.RoleBody]
	if !ok {
		t.Fatal("body 역할이 없다 — 부모의 역할을 물려받지 않았다")
	}
	if body.Size != 24 || body.LineHeight != 1.5 {
		t.Errorf("body = %gpx/%g, want 24px/1.5 (부모의 값)", body.Size, body.LineHeight)
	}

	// variant: 이름이 같은 kicker는 자식 값, 부모의 lead는 그대로, 새 eyebrow는 뒤에 붙는다.
	var variants []string
	var kickerSize float64
	for _, v := range pkg.components.textVariants {
		variants = append(variants, v.name)
		if v.name == "kicker" && v.patch.size != nil {
			kickerSize = *v.patch.size
		}
	}
	if strings.Join(variants, ",") != "kicker,lead,eyebrow" {
		t.Errorf("텍스트 variant = %v, want [kicker lead eyebrow]", variants)
	}
	if kickerSize != 20 {
		t.Errorf("kicker size = %g, want 20", kickerSize)
	}
}

// TestExtendsMergesSingletonSectionsByField는 이름이 없는 절이 **필드 단위**로 갈림을 못 박는다:
// 자식이 `shape.default.fill`만 적으면 부모의 border·radius가 남는다. 빈 값은 "정하지 않음"이다.
func TestExtendsMergesSingletonSectionsByField(t *testing.T) {
	pkg := mustLoadLocal(t, childOfMinimal(), "corp")
	def := pkg.components.shapeDefault
	if def.fill == nil || def.fill.Color.Token != "brand" {
		t.Errorf("shape.default.fill = %v, want brand — 자식이 이기지 않았다", def.fill)
	}
	if def.border == nil || def.radius == nil {
		t.Fatalf("shape.default의 border/radius가 사라졌다 (%v/%v) — 절을 통째로 덮었다", def.border, def.radius)
	}
	parent := loadBuiltinTheme("minimal").components.shapeDefault
	if def.border.Width != parent.border.Width || def.radius.Px != parent.radius.Px {
		t.Errorf("border/radius = %v/%v, want 부모의 %v/%v", def.border, def.radius, parent.border, parent.radius)
	}
	// code 절도 같다 — 자식이 tabSize를 적지 않았으므로 부모의 값이다.
	if pkg.components.code.tabSize != parent2TabSize(t) {
		t.Errorf("code.tabSize = %d, want %d (부모의 값)", pkg.components.code.tabSize, parent2TabSize(t))
	}
	if got := pkg.components.code.colors[classKeyword]; got.Name != "accent1" {
		t.Errorf("keyword 색 = %q, want accent1 — 자식의 코드 토큰이 이기지 않았다", got.Name)
	}
	if _, ok := pkg.components.code.colors[classComment]; !ok {
		t.Error("comment 색이 사라졌다 — 부모의 코드 토큰을 물려받지 않았다")
	}
}

// parent2TabSize는 부모(minimal)의 탭 폭이다 — 기대값을 손으로 적지 않는다.
func parent2TabSize(t *testing.T) int {
	t.Helper()
	return loadBuiltinTheme("minimal").components.code.tabSize
}

// TestExtendsOverridesLayoutCSSByFile은 레이아웃 CSS가 **파일 단위**로 갈림을 못 박는다
// (ADR 0054의 결정 9): 파일을 두면 내 것, 두지 않으면 부모 것이다. 선언 단위 병합은 저자가 두
// 파일을 동시에 머리에 담아야 예측할 수 있고, 배치는 이 제품에서 가장 되돌리기 어려운 층이다.
func TestExtendsOverridesLayoutCSSByFile(t *testing.T) {
	pkg := mustLoadLocal(t, childOfMinimal(), "corp")
	styles := pkg.layoutStyles()
	if got := styles["title"]; !strings.Contains(got, "display: grid") {
		t.Errorf("title 배치 = %q — 자식의 파일이 이기지 않았다", got)
	}
	parent := loadBuiltinTheme("minimal")
	if len(styles) != len(parent.layoutStyles()) {
		t.Errorf("배치 CSS %d개, want %d — 부모의 파일을 물려받지 않았다", len(styles), len(parent.layoutStyles()))
	}
	for name, want := range parent.layoutStyles() {
		if name == "title" {
			continue
		}
		if styles[name] != want {
			t.Errorf("%s의 배치가 부모와 다르다: %q", name, styles[name])
		}
	}
	if len(pkg.slotLayoutStyles()) != len(parent.slotLayoutStyles()) {
		t.Errorf("슬롯 영역 %d개, want %d — 슬롯 CSS도 물려받아야 한다",
			len(pkg.slotLayoutStyles()), len(parent.slotLayoutStyles()))
	}
}

// TestExtendsInheritsTheLayoutDeclaration은 자식이 `layouts`를 생략할 수 있음을 못 박는다 —
// 생략하면 부모의 목록이고, 그래서 표준 11종 계약(ADR 0052)을 자식이 다시 적지 않아도 받는다.
func TestExtendsInheritsTheLayoutDeclaration(t *testing.T) {
	pkg := mustLoadLocal(t, childOfMinimal(), "corp")
	want := loadBuiltinTheme("minimal").manifest.Layouts
	if strings.Join(pkg.manifest.Layouts, ",") != strings.Join(want, ",") {
		t.Errorf("layouts = %v, want %v", pkg.manifest.Layouts, want)
	}
	// 적었다면 그것이 최종 목록이고, 부모∪자식의 CSS 파일 집합과 양방향으로 맞아야 한다.
	short := localTheme("corp", `{"name":"corp","extends":"minimal","layouts":["content"]}`, "", "", nil)
	if err := second(loadThemeNamed(short, "corp")); err == nil {
		t.Error("선언이 파일 집합보다 좁은데 통과했다 — 양방향 대조가 없다")
	}
}

// TestExtendsMergesChromeByKind는 크롬 선언도 같은 규칙을 받음을 못 박는다 (ADR 0054의 결정 7):
// 같은 `kind`면 자식이 이기고, 자식이 말하지 않은 kind는 부모 것이 남는다. 순서도 부모 순서다.
func TestExtendsMergesChromeByKind(t *testing.T) {
	pkg := mustLoadLocal(t, childOfMinimal(), "corp")
	decls := pkg.chromeDecls()
	if len(decls) != 2 {
		t.Fatalf("크롬 선언 %d개, want 2 (footer + 물려받은 pageNumber): %v", len(decls), decls)
	}
	if decls[0].kind != soml.ChromeFooter || decls[1].kind != soml.ChromePageNumber {
		t.Errorf("크롬 순서 = %v, %v — 부모 순서가 아니다", decls[0].kind, decls[1].kind)
	}
	if decls[0].align != soml.AlignCenter {
		t.Errorf("footer 정렬 = %v, want center — 자식이 이기지 않았다", decls[0].align)
	}
	if !decls[0].skip[soml.LayoutTitle] || decls[0].skip[soml.LayoutEnd] {
		t.Error("footer의 제외 규칙이 자식의 선언 그대로가 아니다 (이름 단위 덮어쓰기)")
	}
	if decls[1].align != soml.AlignEnd {
		t.Errorf("pageNumber 정렬 = %v, want end (부모의 값)", decls[1].align)
	}
}

// TestExtendsChainMergesRootToLeaf는 체인이 뿌리부터 잎으로 얹힘을 못 박는다 — 로컬이 로컬을
// 상속하는 길이 있다 (`corp-dark` → `corp` → `minimal`).
func TestExtendsChainMergesRootToLeaf(t *testing.T) {
	project := mergeFS(childOfMinimal(), localTheme("corp-dark",
		`{"name":"corp-dark","extends":"corp"}`,
		`{"color":[{"name":"bg","rgb":"000000","slot":"lt1","ref":"bg1"}]}`,
		"", nil))
	pkg := mustLoadLocal(t, project, "corp-dark")
	if got := colorTokenRGB(t, pkg.tokens, "bg"); got != "000000" {
		t.Errorf("bg = %q, want 000000 (잎의 값)", got)
	}
	if got := colorTokenRGB(t, pkg.tokens, "accent1"); got != "112233" {
		t.Errorf("accent1 = %q, want 112233 (중간 테마의 값)", got)
	}
	if got := colorTokenRGB(t, pkg.tokens, "brand"); got != "445566" {
		t.Errorf("brand = %q, want 445566 (중간 테마가 더한 이름)", got)
	}
	if _, ok := pkg.components.textRoles[soml.RoleCaption]; !ok {
		t.Error("caption 역할이 없다 — 뿌리의 역할이 잎까지 오지 않았다")
	}
}

// TestExtendsRejectsCyclesAndUnknownParents는 상속이 멈추는 자리를 못 박는다. 순환은 스택을
// 태우는 대신 체인을 그대로 적어 에러가 **고칠 자리**를 보여야 한다.
func TestExtendsRejectsCyclesAndUnknownParents(t *testing.T) {
	cases := []struct {
		name    string
		project fstest.MapFS
		request string
		want    []string
	}{
		{"자기 자신", localTheme("corp", `{"name":"corp","extends":"corp"}`, "", "", nil), "corp",
			[]string{"corp"}},
		{"둘이 서로", mergeFS(
			localTheme("corp", `{"name":"corp","extends":"corp-dark"}`, "", "", nil),
			localTheme("corp-dark", `{"name":"corp-dark","extends":"corp"}`, "", "", nil)),
			"corp", []string{"corp", "corp-dark"}},
		{"없는 부모", localTheme("corp", `{"name":"corp","extends":"nope"}`, "", "", nil), "corp",
			[]string{"nope"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := second(loadThemeNamed(tc.project, tc.request))
			if err == nil {
				t.Fatal("상속이 멈추지 않았다")
			}
			for _, want := range tc.want {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("에러가 %q를 말하지 않는다: %v", want, err)
				}
			}
		})
	}
}

// TestExtendsBuiltinFromBuiltinStaysLoadable은 내장 테마 셋이 여전히 상속 없이 완전함을 못
// 박는다 — 상속이 열려도 내장 테마는 각자 완전한 패키지다 (ADR 0052의 계약).
func TestExtendsBuiltinFromBuiltinStaysLoadable(t *testing.T) {
	for _, name := range BuiltinThemes() {
		pkg, err := loadThemeNamed(nil, name)
		if err != nil {
			t.Fatalf("내장 테마 %q가 로드되지 않는다: %v", name, err)
		}
		if pkg.manifest.Extends != "" {
			t.Errorf("내장 테마 %q가 상속을 쓴다 (%q) — 셋은 각자 완전해야 한다", name, pkg.manifest.Extends)
		}
	}
}

// TestLoadConfigAcceptsAProjectLocalTheme는 설정 로드가 **프로젝트의** 어휘를 본다는 것을
// 디스크 위에서 못 박는다 (위의 MapFS 테스트들과 달리 여기서는 `os.DirFS`가 실제로 돈다).
func TestLoadConfigAcceptsAProjectLocalTheme(t *testing.T) {
	dir := t.TempDir()
	writeTree(t, dir, standaloneLocal("corp"))
	if err := os.WriteFile(filepath.Join(dir, ConfigFile), []byte(`{"theme":"corp"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := LoadConfig(dir)
	if err != nil {
		t.Fatalf("로컬 테마를 적은 설정이 거부됐다: %v", err)
	}
	if cfg.Theme != "corp" {
		t.Errorf("cfg.Theme = %q, want corp", cfg.Theme)
	}
	// 같은 프로젝트에서 오타는 여전히 멈추고, 목록에 자기 테마가 보인다.
	if err := os.WriteFile(filepath.Join(dir, ConfigFile), []byte(`{"theme":"corpp"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err = LoadConfig(dir)
	if err == nil {
		t.Fatal("오타가 통과했다")
	}
	for _, want := range []string{ConfigFile, "corpp", "corp", "minimal"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("에러가 %q를 말하지 않는다: %v", want, err)
		}
	}
}

// ── 테스트 도구 ─────────────────────────────────────────────────────────────

// writeTree는 MapFS 하나를 디스크에 쓴다 — 메모리 픽스처를 실제 프로젝트로 옮기는 자리다.
func writeTree(t *testing.T, dir string, fsys fstest.MapFS) {
	t.Helper()
	for name, file := range fsys {
		dest := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(dest, file.Data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

// themeDeckIn은 themeDeck의 프로젝트 있는 판이다 — 같은 원문을 주어진 프로젝트에서 해석한다.
func themeDeckIn(t *testing.T, project fstest.MapFS, name, src string) (*soml.Deck, soml.Diagnostics) {
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
	return deck, Resolve(deck, project)
}

// mustLoadLocal은 프로젝트의 테마 하나를 로드한다 (병합 결과를 표로 보는 테스트들의 입구).
func mustLoadLocal(t *testing.T, project fstest.MapFS, name string) themePackage {
	t.Helper()
	pkg, err := loadThemeNamed(project, name)
	if err != nil {
		t.Fatalf("테마 %q 로드 실패: %v", name, err)
	}
	return pkg
}
