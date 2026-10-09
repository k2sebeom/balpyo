package core

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/k2sebeom/balpyo/soml"
)

// 프로젝트 로컬 테마: 이름은 내장 ∪ `themes/<이름>/`에서 찾고 한 프로젝트에서 하나를
// 뜻한다. extends 병합은 "이름이 같으면 자식이 이기고 없던 이름은 부모 순서 뒤에" 하나이고
// (싱글톤 절은 필드 단위, 레이아웃 CSS는 파일 단위), 완전성은 병합 결과의 성질이다.
// 로더는 fs.FS 위에서 돌므로 프로젝트는 MapFS 하나로 충분하다.

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

// childOfMinimal은 minimal을 상속한 부분 패치 테마다 — 항목마다 덮기 하나와 더하기 하나를 갖고,
// layouts 선언과 나머지는 부모가 깐다. 더하는 색 brand는 남은 clrScheme 자리(hlink)에 앉는다.
func childOfMinimal() fstest.MapFS {
	return localTheme("corp",
		`{"name":"corp","extends":"minimal",
		  "chrome":[{"kind":"footer","align":"center","role":"caption","skipLayouts":["title"]}]}`,
		`{"color":[{"name":"accent1","rgb":"112233","slot":"accent1","ref":"accent1"},
		           {"name":"brand","rgb":"445566","slot":"hlink","ref":"hlink"}]}`,
		`{"text":{"roles":[{"name":"title","font":"heading","size":72,"weight":700,
		                    "lineHeight":1.1,"color":"accent1","align":"center"}],
		          "variants":[{"name":"kicker","size":20},{"name":"eyebrow","color":"accent1"}]},
		  "shape":{"default":{"fill":"brand"}},
		  "code":{"tokens":[{"class":"keyword","color":"accent1"}]}}`,
		map[string]string{"title.css": "/* corp의 표지 */\ndisplay: grid;"})
}

// 프로젝트의 `themes/<이름>/`이 요청의 대상이 된다.
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

// fsys가 nil이면 내장 테마만 보인다 (설정 없이 Deck을 만드는 길).
func TestResolveWithoutFilesystemSeesBuiltinsOnly(t *testing.T) {
	if _, diags := themeDeck(t, "ink", themeProbe); len(diags) != 0 {
		t.Fatalf("fsys 없이 내장 테마가 거부됐다: %v", diags)
	}
	deck, diags := themeDeck(t, "corp", themeProbe)
	if len(diags) != 1 || deck.Theme.Name != "" {
		t.Fatalf("fsys 없이 로컬 이름이 통과했다: %v", diags)
	}
}

// 어휘는 두 출처의 합집합이다 — 자기 테마를 오타 냈을 때 목록에 자기 테마가 보여야 고칠 수 있다.
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

// 로컬 테마가 멈추는 자리: 디렉토리 이름이 요청 키라 매니페스트 이름과 갈리면 에러, 내장과 겹치는
// 이름은 요청이 무엇이든 에러(섀도잉하면 extends: "minimal"에 답이 없다), 상속하지 않은 테마는
// 내장 테마와 같은 완전성 조건을 받는다 (폴백 테마가 없다).
func TestLocalThemeLoadErrors(t *testing.T) {
	cases := []struct {
		name, request string
		project       fstest.MapFS
		want          []string
	}{
		{"디렉토리와 다른 이름", "corp", localTheme("corp", `{"name":"acme","layouts":["content"]}`,
			fullTokens, fullComponents, map[string]string{"content.css": ""}), []string{"corp", "acme"}},
		{"내장 섀도잉 (같은 이름 요청)", "minimal", standaloneLocal("minimal"), []string{"minimal"}},
		{"내장 섀도잉 (다른 이름 요청)", "ink", standaloneLocal("minimal"), []string{"minimal"}},
		{"상속 없는 불완전 테마", "corp", localTheme("corp", `{"name":"corp","layouts":["content"]}`,
			`{"color":[{"name":"fg","rgb":"000000","slot":"dk1","ref":"tx1"}]}`, fullComponents,
			map[string]string{"content.css": ""}), []string{"bg"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, diags := themeDeckIn(t, tc.project, tc.request, themeProbe)
			if len(diags) != 1 {
				t.Fatalf("진단 %d개, want 1: %v", len(diags), diags)
			}
			for _, want := range tc.want {
				if !strings.Contains(diags[0].Message, want) {
					t.Errorf("진단이 %q를 말하지 않는다: %s", want, diags[0].Message)
				}
			}
		})
	}
}

// 모든 색 토큰은 겹치지 않는 clrScheme 슬롯(열둘)을 가져야 한다. 판정은 렌더가 아니라
// 테마 로딩에 있어 진단이 고칠 파일(tokens.json)과 남은 자리를 말한다.
func TestLocalThemeColorNeedsAFreeClrSchemeSlot(t *testing.T) {
	cases := []struct {
		name  string
		color string
		want  []string
	}{
		{
			name:  "슬롯이 없다",
			color: `{"name":"brand","rgb":"445566"}`,
			want:  []string{"brand", "슬롯이 없다", "hlink", "folHlink"},
		},
		{
			name:  "없는 자리를 가리킨다",
			color: `{"name":"brand","rgb":"445566","slot":"accent7","ref":"accent7"}`,
			want:  []string{"brand", "accent7", "쓸 수 있는 슬롯"},
		},
		{
			name:  "부모가 쓰는 자리를 빼앗는다",
			color: `{"name":"brand","rgb":"445566","slot":"accent1","ref":"accent1"}`,
			want:  []string{"accent1", "accent1", "brand"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			project := localTheme("corp",
				`{"name":"corp","extends":"minimal"}`,
				`{"color":[`+tc.color+`]}`, "", nil)
			_, diags := themeDeckIn(t, project, "corp", themeProbe)
			if len(diags) != 1 {
				t.Fatalf("슬롯 없는/겹치는 색이 통과했다: %v", diags)
			}
			for _, want := range tc.want {
				if !strings.Contains(diags[0].Message, want) {
					t.Errorf("진단에 %q가 없다: %s", want, diags[0].Message)
				}
			}
			if !strings.Contains(diags[0].Message, "tokens.json") {
				t.Errorf("진단이 고칠 파일을 가리키지 않는다: %s", diags[0].Message)
			}
		})
	}
}

// 남은 자리가 없으면 나열할 것이 없으므로 "새 이름 대신 값을 갈아 끼운다"를 적는다.
func TestFullColorTableLeavesNoFreeSlot(t *testing.T) {
	owner := map[string]string{}
	for _, slot := range soml.ColorSlots() {
		owner[slot] = "쓰는 색"
	}
	if got := freeColorSlots(owner); !strings.Contains(got, "갈아 끼운다") {
		t.Errorf("자리가 다 찬 표의 문구 = %q — 고치는 방법을 말하지 않는다", got)
	}
	delete(owner, "folHlink")
	if got := freeColorSlots(owner); got != "남은 슬롯: folHlink" {
		t.Errorf("한 자리 남은 표의 문구 = %q, want 남은 슬롯: folHlink", got)
	}
}

// 이름이 같으면 자식이 이기고 없던 이름은 부모 순서 뒤에 붙는다 — 순서는
// clrScheme 슬롯·CSS 변수·경고 목록으로 산출물에 보인다.
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

// 이름 없는 절은 필드 단위로 갈린다 — shape.default.fill만 적으면 부모의 border·radius가 남는다.
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
	if want := loadBuiltinTheme("minimal").components.code.tabSize; pkg.components.code.tabSize != want {
		t.Errorf("code.tabSize = %d, want %d (부모의 값)", pkg.components.code.tabSize, want)
	}
	if got := pkg.components.code.colors[classKeyword]; got.Name != "accent1" {
		t.Errorf("keyword 색 = %q, want accent1 — 자식의 코드 토큰이 이기지 않았다", got.Name)
	}
	if _, ok := pkg.components.code.colors[classComment]; !ok {
		t.Error("comment 색이 사라졌다 — 부모의 코드 토큰을 물려받지 않았다")
	}
}

// 레이아웃 CSS는 파일 단위로 갈린다 — 선언 단위 병합은 저자가 두 파일을 함께
// 머리에 담아야 예측할 수 있다.
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

// 컴포넌트 표의 색은 이 테마의 색 토큰 **이름**이어야 한다 — 그라디언트·리터럴·오타는 그 variant를
// 쓴 슬라이드가 렌더될 때가 아니라 테마 로딩에서 멈춘다 (check가 build와 같은 판정을 하는 이유).
func TestComponentColorsMustBeThemeTokens(t *testing.T) {
	cases := []struct{ name, comps, want string }{
		{"shape variant 그라디언트", `{"shape":{"variants":[{"name":"chip","fill":"gradient(accent6 accent1, 90deg)"}]}}`,
			`shape variant "chip"의 fill "gradient(accent6 accent1, 90deg)"`},
		{"shape variant 테두리 리터럴", `{"shape":{"variants":[{"name":"badge","border":"2 solid #FF9900"}]}}`,
			`shape variant "badge"의 border "#FF9900"`},
		{"text variant 오타", `{"text":{"variants":[{"name":"kicker","color":"acent1"}]}}`,
			`text variant "kicker"의 color "acent1"`},
		{"표 줄무늬", `{"table":{"stripe":"zebra"}}`, `table.stripe의 fill "zebra"`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			project := localTheme("corp", `{"name":"corp","extends":"minimal"}`, "", tc.comps, nil)
			_, diags := themeDeckIn(t, project, "corp", themeProbe)
			if len(diags) != 1 || diags[0].Severity != soml.SeverityError {
				t.Fatalf("진단 = %v, want 에러 1개", diags)
			}
			if !strings.Contains(diags[0].Message, tc.want) {
				t.Errorf("진단이 %q를 말하지 않는다: %s", tc.want, diags[0].Message)
			}
		})
	}
}

// 로컬 테마의 레이아웃 CSS는 로딩에서 린트되어 덱 진단으로 나간다 — 경고는 병합 결과에 남은
// 파일의 주인(고칠 파일)을 가리키고, 자식이 덮어 쓴 부모의 파일과 내장 테마의 파일은 말하지 않는다.
func TestLocalThemeLayoutCSSIsLinted(t *testing.T) {
	parent := localTheme("corp", `{"name":"corp","extends":"minimal"}`, "", "", map[string]string{
		"title.css":       "background: red;",
		"end.css":         "color: red;",
		"title.slots.css": "box-shadow: none;",
	})
	child := localTheme("corp-dark", `{"name":"corp-dark","extends":"corp"}`, "", "", map[string]string{
		"title.css": "display: grid;",
	})
	_, diags := themeDeckIn(t, mergeFS(parent, child), "corp-dark", themeProbe)
	want := []string{"themes/corp/layouts/end.css", "themes/corp/layouts/title.slots.css"}
	if len(diags) != len(want) {
		t.Fatalf("진단 %d개, want %d: %v", len(diags), len(want), diags)
	}
	for i, file := range want {
		if diags[i].File != file || diags[i].Severity != soml.SeverityWarning {
			t.Errorf("진단[%d] = %s %q, want warning %q", i, diags[i].Severity, diags[i].File, file)
		}
	}
}

// layouts를 생략하면 부모의 목록이다. 적었다면 그것이 최종 목록이고 부모∪자식의 CSS 파일 집합과
// 양방향으로 맞아야 한다.
func TestExtendsInheritsTheLayoutDeclaration(t *testing.T) {
	pkg := mustLoadLocal(t, childOfMinimal(), "corp")
	want := loadBuiltinTheme("minimal").manifest.Layouts
	if strings.Join(pkg.manifest.Layouts, ",") != strings.Join(want, ",") {
		t.Errorf("layouts = %v, want %v", pkg.manifest.Layouts, want)
	}
	short := localTheme("corp", `{"name":"corp","extends":"minimal","layouts":["content"]}`, "", "", nil)
	if err := second(loadThemeNamed(short, "corp")); err == nil {
		t.Error("선언이 파일 집합보다 좁은데 통과했다 — 양방향 대조가 없다")
	}
}

// 크롬 선언도 같은 규칙이다 — 같은 kind면 자식이 이기고 나머지는 부모 것이 부모 순서로 남는다.
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

// 체인은 뿌리부터 잎으로 얹힌다 (corp-dark → corp → minimal).
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

// 순환과 없는 부모는 멈춘다 — 순환은 체인을 그대로 적어 고칠 자리를 보인다.
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

// 설정 로드는 디스크 위 프로젝트의 어휘를 본다 (여기서만 os.DirFS가 실제로 돈다).
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

// themeDeckIn은 원문을 주어진 프로젝트(nil이면 내장 테마만)와 테마 이름으로 해석한다.
func themeDeckIn(t *testing.T, project fs.FS, name, src string) (*soml.Deck, soml.Diagnostics) {
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
