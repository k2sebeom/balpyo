package core

import (
	"io/fs"
	"path"
	"reflect"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/k2sebeom/balpyo/soml"
)

// 내장 minimal의 매니페스트와 토큰 표를 값·순서까지 리터럴로 못 박는다 — tokens.json이 실수로
// 바뀌면 골든보다 여기가 먼저 잡는다.
func TestLoadBuiltinMinimal(t *testing.T) {
	pkg := loadBuiltinTheme("minimal")
	if pkg.manifest.Name != "minimal" {
		t.Fatalf("manifest.Name = %q, want minimal", pkg.manifest.Name)
	}
	wantLayouts := []string{"title", "section-divider", "content", "two-column",
		"image-full", "image-text", "quote", "code", "stat", "canvas", "end"}
	if !reflect.DeepEqual(pkg.manifest.Layouts, wantLayouts) {
		t.Errorf("manifest.Layouts = %v\nwant %v", pkg.manifest.Layouts, wantLayouts)
	}

	got := pkg.tokens
	want := soml.Theme{
		Name: "minimal",
		Colors: []soml.ColorToken{
			{Name: "fg", RGB: "1A1A1A", Slot: "dk1", Ref: "tx1"},
			{Name: "bg", RGB: "FFFFFF", Slot: "lt1", Ref: "bg1"},
			{Name: "muted", RGB: "6B7280", Slot: "dk2", Ref: "tx2"},
			{Name: "surface", RGB: "F3F4F6", Slot: "lt2", Ref: "bg2"},
			{Name: "accent1", RGB: "0F65F9", Slot: "accent1", Ref: "accent1"},
			{Name: "accent2", RGB: "0E8174", Slot: "accent2", Ref: "accent2"},
			{Name: "accent3", RGB: "15803D", Slot: "accent3", Ref: "accent3"},
			{Name: "accent4", RGB: "B45309", Slot: "accent4", Ref: "accent4"},
			{Name: "accent5", RGB: "DC2626", Slot: "accent5", Ref: "accent5"},
			{Name: "accent6", RGB: "9333EA", Slot: "accent6", Ref: "accent6"},
		},
		Fonts: []soml.FontToken{
			{Name: soml.FontHeading, Latin: "Noto Sans", EastAsian: "Noto Sans CJK KR", Generic: "sans-serif", Scheme: "major"},
			{Name: soml.FontBody, Latin: "Noto Sans", EastAsian: "Noto Sans CJK KR", Generic: "sans-serif", Scheme: "minor"},
			{Name: soml.FontMono, Latin: "Noto Sans Mono", EastAsian: "Noto Sans Mono CJK KR", Generic: "monospace"},
		},
		Spaces: []soml.SpaceToken{
			{Name: "xs", Px: 6}, {Name: "sm", Px: 12}, {Name: "md", Px: 24},
			{Name: "lg", Px: 36}, {Name: "xl", Px: 48}, {Name: "2xl", Px: 72},
		},
		Radii: []soml.RadiusToken{
			{Name: "sm", Px: 6}, {Name: "md", Px: 12}, {Name: "lg", Px: 24},
		},
		Shadows: []soml.ShadowToken{
			{Name: "soft", Dist: 4, Dir: 90, Blur: 16, Color: soml.TokenColor("fg"), Alpha: 0.15},
		},
		Motion: &soml.MotionTokens{Duration: 600, Stagger: 90, Delay: 150, TransitionDuration: 600},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("로드된 minimal 토큰이 기대와 다르다:\ngot  %+v\nwant %+v", got, want)
	}
}

// minimal의 components.json을 값·순서까지 리터럴로 못 박는다. 캐스케이드 테스트는 이 표에서
// 파생한 값을 쓰므로(theme_tables_test.go) 값을 고정하는 책임은 여기 한 곳이다. variant 순서는
// 모르는 variant 경고 목록의 결정성이다.
func TestLoadBuiltinComponents(t *testing.T) {
	comps := loadBuiltinTheme("minimal").components

	wantRoles := map[soml.TextRole]soml.TextStyle{
		soml.RoleTitle: {Role: soml.RoleTitle, Font: soml.FontHeading, Size: 56, Weight: 700,
			LineHeight: 1.2, Color: soml.TokenColor("fg"), Align: soml.AlignStart},
		soml.RoleHeading: {Role: soml.RoleHeading, Font: soml.FontHeading, Size: 36, Weight: 700,
			LineHeight: 1.3, Color: soml.TokenColor("fg"), Align: soml.AlignStart},
		soml.RoleBody: {Role: soml.RoleBody, Font: soml.FontBody, Size: 24, Weight: 400,
			LineHeight: 1.5, Color: soml.TokenColor("fg"), Align: soml.AlignStart},
		soml.RoleCaption: {Role: soml.RoleCaption, Font: soml.FontBody, Size: 18, Weight: 400,
			LineHeight: 1.4, Color: soml.TokenColor("muted"), Align: soml.AlignStart},
		soml.RoleCode: {Role: soml.RoleCode, Font: soml.FontMono, Size: 20, Weight: 400,
			LineHeight: 1.5, Color: soml.TokenColor("fg"), Align: soml.AlignStart},
	}
	if len(comps.textRoles) != len(wantRoles) {
		t.Errorf("텍스트 역할 수 = %d, want %d", len(comps.textRoles), len(wantRoles))
	}
	for _, role := range soml.TextRoles() {
		got, ok := comps.textRoles[role]
		if !ok {
			t.Fatalf("역할 %q가 로드된 표에 없다", role)
		}
		if !reflect.DeepEqual(got, wantRoles[role]) {
			t.Errorf("역할 %q: got %+v\nwant %+v", role, got, wantRoles[role])
		}
	}

	// 나머지 태그는 빈 패치라 `<p>` ≡ `<text role="body">`가 성립한다.
	wantDefaults := map[string]textPatch{
		"h3":         {size: ptr(28.0)},
		"blockquote": {color: ptr(soml.TokenColor("muted"))},
	}
	if !reflect.DeepEqual(comps.textComponentDefaults, wantDefaults) {
		t.Errorf("컴포넌트 기본값 표가 다르다:\ngot  %+v\nwant %+v", comps.textComponentDefaults, wantDefaults)
	}

	wantTextVariants := []textVariant{
		{name: "kicker", patch: textPatch{size: ptr(18.0), weight: ptr(700),
			color: ptr(soml.TokenColor("accent1")), tracking: ptr(0.1), textCase: ptr(soml.CaseUpper)}},
		{name: "lead", patch: textPatch{size: ptr(32.0)}},
	}
	if !reflect.DeepEqual(comps.textVariants, wantTextVariants) {
		t.Errorf("텍스트 variant 표가 다르다:\ngot  %+v\nwant %+v", comps.textVariants, wantTextVariants)
	}

	// 도형 기본값은 캐스케이드의 바닥이다. radius가 있는 이유: 두 타깃의 암묵 기본값이 다르다.
	wantShapeDefault := shapePatch{fill: colorFill("surface"), border: solidBorder(1, "muted"),
		radius: ptr(soml.TokenRadius("md", 12))}
	if !reflect.DeepEqual(comps.shapeDefault, wantShapeDefault) {
		t.Errorf("도형 기본값이 다르다:\ngot  %+v\nwant %+v", comps.shapeDefault, wantShapeDefault)
	}

	soft := minimalShadows()[0]
	wantShapeVariants := []shapeVariant{
		{name: "card", patch: shapePatch{fill: colorFill("bg"), radius: ptr(soml.TokenRadius("lg", 24))}},
		{name: "card-header", patch: shapePatch{border: ptr(soml.NoBorder()),
			radius: ptr(soml.TokenRadius("sm", 6))}},
		{name: "card-body", patch: shapePatch{fill: colorFill("bg")}},
		{name: "card-backdrop", patch: shapePatch{fill: ptr(soml.NoFill()),
			border: solidBorder(3, "accent1"), radius: ptr(soml.TokenRadius("lg", 24))}},
		{name: "stat-card", patch: shapePatch{border: ptr(soml.NoBorder()),
			radius: ptr(soml.TokenRadius("lg", 24)), shadow: &soft}},
		{name: "badge", patch: shapePatch{border: solidBorder(1, "accent1"),
			radius: ptr(soml.TokenRadius("lg", 24))}},
	}
	if !reflect.DeepEqual(comps.shapeVariants, wantShapeVariants) {
		t.Errorf("도형 variant 표가 다르다:\ngot  %+v\nwant %+v", comps.shapeVariants, wantShapeVariants)
	}

	// 이미지 기본값은 비어 있다 — 액자는 저자가 variant로 말한다.
	if !reflect.DeepEqual(comps.imageDefault, imagePatch{}) {
		t.Errorf("이미지 기본값 = %+v, want 빈 패치", comps.imageDefault)
	}
	wantImageVariants := []imageVariant{{name: "framed", patch: imagePatch{
		border: solidBorder(1, "muted"), radius: ptr(soml.TokenRadius("md", 12)), shadow: &soft}}}
	if !reflect.DeepEqual(comps.imageVariants, wantImageVariants) {
		t.Errorf("이미지 variant 표가 다르다:\ngot  %+v\nwant %+v", comps.imageVariants, wantImageVariants)
	}
}

// 닫힌 어휘: 모르는 키는 에러다. `extends`는 theme.json에서만 선언한다.
func TestLoadComponentsRejectsUnknownField(t *testing.T) {
	fsys := fstest.MapFS{
		"t/components.json": {Data: []byte(`{"extends":"base","text":{"roles":[]}}`)},
	}
	if _, err := readComponentsFile(fsys, "t", false); err == nil {
		t.Fatal("미지원 필드 extends가 진단되지 않았다 (조용히 무시됨)")
	}
}

// 닫힌 어휘의 유일한 예외는 범주마다 여는 `$comment`다 — 값의 근거가 값 옆에 있어야 한다.
func TestLoadComponentsAllowsCommentPerCategory(t *testing.T) {
	fsys := fstest.MapFS{
		"t/components.json": {Data: []byte(`{
		  "$comment": "테마 주석",
		  "text":  {"$comment": "역할 주석", "roles": []},
		  "shape": {"$comment": "도형 주석", "default": {}},
		  "image": {"$comment": "그림 주석", "default": {}},
		  "code":  {"$comment": "코드 주석", "tabSize": 2}
		}`)},
	}
	if _, err := readComponentsFile(fsys, "t", false); err != nil {
		t.Fatalf("범주 주석이 진단됐다: %v", err)
	}
}

// 레이아웃 선언과 layouts/*.css 파일은 양방향으로 맞아야 한다.
func TestLoadLayoutsMismatch(t *testing.T) {
	missing := fstest.MapFS{
		"t/layouts/a.css": {Data: []byte("/* a */")},
	}
	if _, err := loadLayoutsForTest(missing, "t", themeManifest{Layouts: []string{"a", "b"}}); err == nil {
		t.Error("선언한 b의 파일이 없는데 진단되지 않았다")
	}
	extra := fstest.MapFS{
		"t/layouts/a.css":     {Data: []byte("/* a */")},
		"t/layouts/extra.css": {Data: []byte("/* extra */")},
	}
	if _, err := loadLayoutsForTest(extra, "t", themeManifest{Layouts: []string{"a"}}); err == nil {
		t.Error("선언에 없는 extra.css가 진단되지 않았다")
	}
	ok := fstest.MapFS{
		"t/layouts/a.css": {Data: []byte("/* a */")},
		"t/layouts/b.css": {Data: []byte("")},
	}
	got, err := loadLayoutsForTest(ok, "t", themeManifest{Layouts: []string{"a", "b"}})
	if err != nil {
		t.Fatalf("일치하는데 에러가 났다: %v", err)
	}
	if len(got) != 2 {
		t.Errorf("맵 크기 = %d, want 2", len(got))
	}
}

// loadLayoutsForTest는 읽기(readLayoutCSS)와 검사(checkLayoutFiles)를 한 호출로 붙인다.
func loadLayoutsForTest(fsys fs.FS, dir string, manifest themeManifest) (map[string]string, error) {
	layouts, slots, err := readLayoutCSS(fsys, dir, false)
	if err != nil {
		return nil, err
	}
	if err := checkLayoutFiles(manifest, layouts, slots); err != nil {
		return nil, err
	}
	return layouts, nil
}

// layoutStyles는 주석을 벗긴 정규화 선언을 주고, 빈 레이아웃(content/canvas)은 빼서 스테이지에
// 레이아웃 클래스가 붙지 않게 한다.
func TestMinimalLayoutStyles(t *testing.T) {
	styles := loadBuiltinTheme("minimal").layoutStyles()
	if got := styles["title"]; got != "justify-content: center" {
		t.Errorf("title 배치 = %q, want %q", got, "justify-content: center")
	}
	if got := styles["two-column"]; !strings.Contains(got, "grid-template-columns: 1fr 1fr") {
		t.Errorf("two-column 배치에 2열 그리드가 없다: %q", got)
	}
	for _, empty := range []string{"content", "canvas"} {
		if _, ok := styles[empty]; ok {
			t.Errorf("%s가 배치 맵에 있다 — 빈 레이아웃은 제외되어야 한다", empty)
		}
	}
}

// 슬롯 카탈로그(soml)와 테마 영역 CSS의 대조. `grid-template-areas`에 이름이
// 빠지면 그 슬롯은 자동 배치로 조용히 떨어진다.
func TestBuiltinThemesCoverEverySlotArea(t *testing.T) {
	for _, name := range BuiltinThemes() {
		t.Run(name, func(t *testing.T) { checkSlotAreas(t, loadBuiltinTheme(name)) })
	}
}

func checkSlotAreas(t *testing.T, pkg themePackage) {
	t.Helper()
	styles := pkg.slotLayoutStyles()
	for _, layout := range soml.SlideLayouts() {
		name := string(layout)
		slots := soml.LayoutSlots(layout)
		if len(slots) == 0 {
			if _, ok := styles[name]; ok {
				t.Errorf("%s는 슬롯 어휘가 비어 있는데 슬롯 영역 CSS가 있다", name)
			}
			continue
		}
		decl, ok := styles[name]
		if !ok || decl == "" {
			t.Errorf("레이아웃 %q의 슬롯 %d종이 놓일 자리가 없다 — layouts/%s.slots.css가 필요하다",
				name, len(slots), name)
			continue
		}
		// 슬롯 영역 규칙은 흐름 규칙과 배타적이라 display부터 스스로 갖춰야 한다.
		if !strings.Contains(decl, "display: grid") {
			t.Errorf("%s의 슬롯 영역 CSS가 display: grid를 스스로 갖지 않는다: %q", name, decl)
		}
		for _, slot := range slots {
			if !strings.Contains(decl, slot.Name) {
				t.Errorf("%s의 슬롯 영역 CSS에 슬롯 %q의 자리가 없다: %q", name, slot.Name, decl)
			}
		}
	}
	if diags := lintLayouts("themes/"+pkg.manifest.Name, ".slots.css", pkg.slotLayouts); len(diags) != 0 {
		t.Errorf("%s 슬롯 영역 CSS가 진단을 냈다:\n%v", pkg.manifest.Name, diags)
	}
}

// 내장 테마는 표준 레이아웃 전부를 구현하고(폴백 없음 — 테마를 섞지 않는다), 레이아웃 CSS는
// 배치 전용이다 (외관은 PPTX 마스터에 도달하지 못한다).
func TestBuiltinThemesImplementEveryStandardLayout(t *testing.T) {
	for _, name := range BuiltinThemes() {
		pkg := loadBuiltinTheme(name)
		declared := map[string]bool{}
		for _, layout := range pkg.manifest.Layouts {
			declared[layout] = true
		}
		for _, layout := range soml.SlideLayouts() {
			if !declared[string(layout)] {
				t.Errorf("테마 %q가 표준 레이아웃 %q를 구현하지 않았다 (폴백은 없다)", name, layout)
			}
		}
		if len(pkg.manifest.Layouts) != len(soml.SlideLayouts()) {
			t.Errorf("테마 %q의 레이아웃 %d종, want %d종 (%v)",
				name, len(pkg.manifest.Layouts), len(soml.SlideLayouts()), pkg.manifest.Layouts)
		}
		if diags := lintLayouts("themes/"+name, ".css", pkg.layouts); len(diags) != 0 {
			t.Errorf("%s 레이아웃 CSS가 진단을 냈다:\n%v", name, diags)
		}
	}
}

// 내장 테마 목록(embed 트리 — `init --theme`의 어휘)은 기본 테마를 포함하고 전부 로드된다.
func TestBuiltinThemesAreLoadable(t *testing.T) {
	names := BuiltinThemes()
	if len(names) == 0 {
		t.Fatal("내장 테마가 하나도 없다")
	}
	if !IsBuiltinTheme(DefaultTheme) {
		t.Errorf("기본 테마 %q가 내장 목록에 없다 (%v)", DefaultTheme, names)
	}
	for _, name := range names {
		if _, err := loadTheme(builtinThemes, path.Join("themes", name)); err != nil {
			t.Errorf("내장 테마 %q 로드 실패: %v", name, err)
		}
		// 상속이 열려도 내장 테마는 각자 완전한 패키지다.
		if pkg, err := loadThemeNamed(nil, name); err != nil || pkg.manifest.Extends != "" {
			t.Errorf("내장 테마 %q: extends = %q, err = %v", name, pkg.manifest.Extends, err)
		}
		if !IsBuiltinTheme(name) {
			t.Errorf("IsBuiltinTheme(%q) = false", name)
		}
	}
	if IsBuiltinTheme("corp") {
		t.Error(`IsBuiltinTheme("corp") = true — 없는 테마를 받아들인다`)
	}
}

// 컴포넌트 기본값 한 줄의 주인은 하나다 — 태그이거나 레이아웃 슬롯이다. 슬롯은 레이아웃 어휘에 있어야
// 한다: 오타가 조용히 아무 데도 안 먹는 줄이 되면 테마 작성자는 왜 크기가 안 바뀌는지 모른다.
func TestLoadComponentsSlotDefaults(t *testing.T) {
	withDefaults := func(defaults string) string {
		return strings.Replace(fullComponents, `"text": { "roles": [`,
			`"text": { "componentDefaults": [`+defaults+`], "roles": [`, 1)
	}
	if _, err := loadTheme(memTheme(fullTokens, withDefaults(`{"layout":"quote","slot":"quote","size":40}`)), "t"); err != nil {
		t.Fatalf("유효한 슬롯 기본값이 실패했다: %v", err)
	}
	for _, tc := range []struct{ name, defaults, want string }{
		{"없는 슬롯", `{"layout":"quote","slot":"qoute","size":40}`, `"quote"/"qoute" is not in the layout vocabulary`},
		{"태그와 슬롯을 함께", `{"tag":"p","layout":"quote","slot":"quote","size":40}`, "tag \"p\" cannot be combined with slot"},
		{"주인 없음", `{"size":40}`, "has no owner"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := loadTheme(memTheme(fullTokens, withDefaults(tc.defaults)), "t")
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Errorf("에러 = %v, want %q", err, tc.want)
			}
		})
	}
}

// 자간·대소문자는 역할·컴포넌트 기본값·variant가 같은 진단으로 받는다. `none`은 아래 단의 upper를
// 끄는 패치이고 확정 값으로는 원문 그대로(빈 값)다.
func TestLoadComponentsTypography(t *testing.T) {
	withDefaults := func(defaults string) string {
		return strings.Replace(fullComponents, `"text": { "roles": [`,
			`"text": { "componentDefaults": [`+defaults+`], "roles": [`, 1)
	}
	theme, err := loadTheme(memTheme(fullTokens, withDefaults(`{"tag":"h3","tracking":0.2,"case":"upper"},{"tag":"p","case":"none"}`)), "t")
	if err != nil {
		t.Fatalf("유효한 자간·대소문자가 실패했다: %v", err)
	}
	comps := theme.components
	if got := comps.textComponentDefaults["h3"]; got.tracking == nil || *got.tracking != 0.2 || got.textCase == nil || *got.textCase != soml.CaseUpper {
		t.Errorf("h3 기본값 = %+v, want tracking 0.2 · upper", got)
	}
	if got := comps.textComponentDefaults["p"]; got.textCase == nil || *got.textCase != "" {
		t.Errorf("p 기본값 = %+v, want case none → 빈 값", got)
	}
	for _, tc := range []struct{ name, defaults, want string }{
		{"자간 범위 밖", `{"tag":"h3","tracking":0.8}`, "tracking 0.8 is out of range"},
		{"모르는 대소문자", `{"tag":"h3","case":"lower"}`, `case "lower" is unknown (available values: upper, none)`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := loadTheme(memTheme(fullTokens, withDefaults(tc.defaults)), "t")
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Errorf("에러 = %v, want %q", err, tc.want)
			}
		})
	}
}

// 형광펜은 단색 토큰 참조다 — a:highlight는 색 하나만 담으므로 그라디언트는 에러이고, 리터럴은
// 다른 컴포넌트 색과 같은 진단으로 멈춘다.
func TestLoadComponentsMark(t *testing.T) {
	withMark := func(mark string) string {
		return strings.Replace(fullComponents, `"text": { "roles": [`,
			`"text": { "mark": `+mark+`, "roles": [`, 1)
	}
	theme, err := loadTheme(memTheme(fullTokens, withMark(`{"highlight":"tint(accent1, 70%)"}`)), "t")
	if err != nil {
		t.Fatalf("유효한 형광펜이 실패했다: %v", err)
	}
	if hl := theme.components.markHighlight; hl == nil || hl.Token != "accent1" || hl.Mod == nil {
		t.Errorf("markHighlight = %+v, want tint(accent1, 70%%)", hl)
	}
	for _, tc := range []struct{ name, mark, want string }{
		{"그라디언트", `{"highlight":"gradient(accent1 fg)"}`, "cannot be a gradient"},
		{"리터럴", `{"highlight":"#FFEE00"}`, "text.mark highlight"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := loadTheme(memTheme(fullTokens, withMark(tc.mark)), "t")
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Errorf("에러 = %v, want %q", err, tc.want)
			}
		})
	}
}
