package core

import (
	"path"
	"reflect"
	"sort"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/k2sebeom/balpyo/soml"
)

// TestLoadBuiltinMinimal은 내장 minimal 패키지가 로드되고, 그 토큰 표가 **값·순서까지** 기대와
// 같음을 못 박는다 — 테마 패키지 포맷의 앵커다 (theme_package.go). 값이 여기 그대로 적혀 있는
// 이유: tokens.json이 실수로 바뀌면(순서·수치) 이 테스트가 잡고, 기존 골든이 조용히 움직이는
// 것을 막는다. 엔진에 하드코딩 표가 있던 동안에는 그 표와 견주는 것이 앵커였지만(byte 동일성
// 계약), 표가 사라진 지금은 **리터럴 기대값**이 그 자리다 (ADR 0052).
func TestLoadBuiltinMinimal(t *testing.T) {
	pkg := loadBuiltinTheme("minimal")
	if pkg.manifest.Name != "minimal" {
		t.Fatalf("manifest.Name = %q, want minimal", pkg.manifest.Name)
	}
	// 매니페스트가 표준 레이아웃 11종을 전부 구현한다고 선언한다 (폴백 기준).
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
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("로드된 minimal 토큰이 기대와 다르다:\ngot  %+v\nwant %+v", got, want)
	}
}

// TestLoadBuiltinComponents는 내장 minimal의 components.json이 로드되고, 그 표가 **값·순서까지**
// 기대와 같음을 못 박는다 — 포맷의 컴포넌트 앵커다 (TestLoadBuiltinMinimal의 짝).
//
// 기대값이 리터럴인 이유도 같다: 캐스케이드 테스트들은 이 표에서 **파생한** 값을 쓰므로
// (theme_tables_test.go) 그 테스트들은 파일이 움직여도 함께 움직인다. 값을 못 박는 일은 여기
// 한 곳의 책임이고, 수치의 근거(수직 리듬·단마다 구별되는 값)는 components.json의 `$comment`가
// 소유한다 — 값과 근거가 같은 파일에 있다.
func TestLoadBuiltinComponents(t *testing.T) {
	comps := loadBuiltinTheme("minimal").components

	// 텍스트 역할 5종 — 본문 24px × 줄높이 1.5 = 36px이고 스테이지 720px이 정확히 20줄이다
	// (ADR 0002). 나머지 크기는 본문에서 약 1.5배씩 벌린 단계이고, 정렬은 전부 start다.
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

	// 컴포넌트 기본값 — h3(크기 한 단 아래)·blockquote(색만 muted) 둘만 정하고, 나머지 태그는
	// 표에 없어 빈 패치다. 그래서 `<p>` ≡ `<text role="body">`가 깊은 비교로 성립한다.
	wantDefaults := map[string]textPatch{
		"h3":         {size: ptr(28.0)},
		"blockquote": {color: ptr(soml.TokenColor("muted"))},
	}
	if !reflect.DeepEqual(comps.textComponentDefaults, wantDefaults) {
		t.Errorf("컴포넌트 기본값 표가 다르다:\ngot  %+v\nwant %+v", comps.textComponentDefaults, wantDefaults)
	}

	// 텍스트 variant — 값과 **순서**가 파일 그대로다 (모르는 variant 경고의 목록이 결정적이어야
	// 한다). 값은 역할 기본값과 구별되도록 골랐다 — 같은 값을 정하는 variant는 우선순위
	// 테스트가 아무것도 증명하지 못한다.
	wantTextVariants := []textVariant{
		{name: "kicker", patch: textPatch{size: ptr(18.0), weight: ptr(700),
			color: ptr(soml.TokenColor("accent1"))}},
		{name: "lead", patch: textPatch{size: ptr(32.0)}},
	}
	if !reflect.DeepEqual(comps.textVariants, wantTextVariants) {
		t.Errorf("텍스트 variant 표가 다르다:\ngot  %+v\nwant %+v", comps.textVariants, wantTextVariants)
	}

	// 도형 기본값 — 도형에는 역할이 없으므로 이 표가 캐스케이드의 **가장 아래 단**이고, 그래서
	// `<shape type="rect"/>` 하나로도 그려지는 완전한 외관이다 (radius가 여기 있는 이유는 두
	// 타깃의 암묵 기본값이 다르다는 것이다 — CSS는 0, OOXML roundRect는 16.67%).
	wantShapeDefault := shapePatch{fill: colorFill("surface"), border: solidBorder(1, "muted"),
		radius: ptr(soml.TokenRadius("md", 12))}
	if !reflect.DeepEqual(comps.shapeDefault, wantShapeDefault) {
		t.Errorf("도형 기본값이 다르다:\ngot  %+v\nwant %+v", comps.shapeDefault, wantShapeDefault)
	}

	// 도형 variant — PRD §13-6의 권장 어휘 6종 (값은 기본값과 구별된다).
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

	// 이미지 기본값은 **비어 있다** — 그림은 자기 픽셀을 갖고 있어 장식이 없어도 완전하다.
	// 액자는 저자가 variant로 말한다 (`framed` — 세 조각을 함께 정하는 것이 번들의 뜻이다).
	if !reflect.DeepEqual(comps.imageDefault, imagePatch{}) {
		t.Errorf("이미지 기본값 = %+v, want 빈 패치", comps.imageDefault)
	}
	wantImageVariants := []imageVariant{{name: "framed", patch: imagePatch{
		border: solidBorder(1, "muted"), radius: ptr(soml.TokenRadius("md", 12)), shadow: &soft}}}
	if !reflect.DeepEqual(comps.imageVariants, wantImageVariants) {
		t.Errorf("이미지 variant 표가 다르다:\ngot  %+v\nwant %+v", comps.imageVariants, wantImageVariants)
	}
}

// TestLoadBuiltinComponentsPreservesVariantOrder는 variant 배열이 파일 순서 그대로임을 못 박는다
// — 모르는 variant 경고가 나열하는 목록이 결정적이어야 하므로 (JSON 객체 키 순서에 기대지 않는다).
func TestLoadBuiltinComponentsPreservesVariantOrder(t *testing.T) {
	comps := loadBuiltinTheme("minimal").components
	wantText := []string{"kicker", "lead"}
	for i, v := range comps.textVariants {
		if v.name != wantText[i] {
			t.Errorf("텍스트 variant[%d] = %q, want %q", i, v.name, wantText[i])
		}
	}
	wantShape := []string{"card", "card-header", "card-body", "card-backdrop", "stat-card", "badge"}
	if len(comps.shapeVariants) != len(wantShape) {
		t.Fatalf("도형 variant 수 = %d, want %d", len(comps.shapeVariants), len(wantShape))
	}
	for i, v := range comps.shapeVariants {
		if v.name != wantShape[i] {
			t.Errorf("도형 variant[%d] = %q, want %q", i, v.name, wantShape[i])
		}
	}
}

// TestLoadComponentsRejectsUnknownField는 미지원/오타 키를 조용히 무시하지 않고 진단함을 못 박는다
// (닫힌 어휘). extends는 아직 구현하지 않은 상속 키의 예다.
func TestLoadComponentsRejectsUnknownField(t *testing.T) {
	fsys := fstest.MapFS{
		"t/components.json": {Data: []byte(`{"extends":"base","text":{"roles":[]}}`)},
	}
	if _, err := loadComponents(fsys, "t", soml.Theme{}); err == nil {
		t.Fatal("미지원 필드 extends가 진단되지 않았다 (조용히 무시됨)")
	}
}

// TestLoadLayoutsCoverage는 내장 minimal의 layouts/*.css가 매니페스트가 선언한 11종과 정확히
// 대응함을 못 박고, 선언/파일이 어긋나면 조용히 넘기지 않고 진단함을 검증한다 (닫힌 어휘).
func TestLoadLayoutsCoverage(t *testing.T) {
	pkg := loadBuiltinTheme("minimal")
	if pkg.layouts == nil {
		t.Fatal("pkg.layouts가 nil이다 — 로더가 layouts/를 읽지 않았다")
	}
	want := []string{"title", "section-divider", "content", "two-column",
		"image-full", "image-text", "quote", "code", "stat", "canvas", "end"}
	if len(pkg.layouts) != len(want) {
		t.Fatalf("레이아웃 수 = %d, want %d (%v)", len(pkg.layouts), len(want), keysOf(pkg.layouts))
	}
	for _, name := range want {
		if _, ok := pkg.layouts[name]; !ok {
			t.Errorf("레이아웃 %q에 대응하는 layouts/%s.css가 로드되지 않았다", name, name)
		}
	}
}

// keysOf는 진단 메시지에 맵 키를 결정적으로 찍기 위한 테스트 헬퍼다.
func keysOf(m map[string]string) []string {
	ks := make([]string, 0, len(m))
	for k := range m {
		ks = append(ks, k)
	}
	sort.Strings(ks)
	return ks
}

// TestLoadLayoutsMismatch는 선언과 파일의 양방향 불일치가 에러임을 못 박는다: 선언에 있는데
// 파일이 없거나, 파일이 있는데 선언에 없으면 로더가 진단한다.
func TestLoadLayoutsMismatch(t *testing.T) {
	// 선언에 있는데 파일이 없음.
	missing := fstest.MapFS{
		"t/layouts/a.css": {Data: []byte("/* a */")},
	}
	if _, err := loadLayouts(missing, "t", themeManifest{Layouts: []string{"a", "b"}}); err == nil {
		t.Error("선언한 b의 파일이 없는데 진단되지 않았다")
	}
	// 파일이 있는데 선언에 없음.
	extra := fstest.MapFS{
		"t/layouts/a.css":     {Data: []byte("/* a */")},
		"t/layouts/extra.css": {Data: []byte("/* extra */")},
	}
	if _, err := loadLayouts(extra, "t", themeManifest{Layouts: []string{"a"}}); err == nil {
		t.Error("선언에 없는 extra.css가 진단되지 않았다")
	}
	// 선언과 파일이 정확히 일치하면 에러 없이 맵을 준다.
	ok := fstest.MapFS{
		"t/layouts/a.css": {Data: []byte("/* a */")},
		"t/layouts/b.css": {Data: []byte("")},
	}
	got, err := loadLayouts(ok, "t", themeManifest{Layouts: []string{"a", "b"}})
	if err != nil {
		t.Fatalf("일치하는데 에러가 났다: %v", err)
	}
	if len(got) != 2 {
		t.Errorf("맵 크기 = %d, want 2", len(got))
	}
}

// TestMinimalLayoutStyles는 minimal 테마의 배치 CSS가 실체를 담고(빈 스텁이 아니고),
// 그 원문이 배치·외관 경계 린터를 에러·경고 없이 통과함을 못 박는다 (M1.1). layoutStyles는
// 주석을 벗기고 빈 레이아웃(content/canvas)을 제외한 정규화 선언만 돌려준다 — 이 맵이
// soml.Theme.Layouts로 실려 웹·지오메트리 패스가 배치를 얻는다.
func TestMinimalLayoutStyles(t *testing.T) {
	pkg := loadBuiltinTheme("minimal")

	// 배치·외관 경계: minimal의 모든 레이아웃 CSS는 배치 전용이어야 한다 (주석 포함해도
	// 린터가 주석을 벗기므로 조용해야 한다). 에러도 경고도 없다.
	if diags := lintLayouts(pkg.layouts); len(diags) != 0 {
		t.Errorf("minimal 레이아웃 CSS가 진단을 냈다:\n%v", diags)
	}

	styles := pkg.layoutStyles()
	// 실체가 채워진 레이아웃은 정규화 선언을 갖는다.
	if got := styles["title"]; got != "justify-content: center" {
		t.Errorf("title 배치 = %q, want %q", got, "justify-content: center")
	}
	if got := styles["two-column"]; !strings.Contains(got, "grid-template-columns: 1fr 1fr") {
		t.Errorf("two-column 배치에 2열 그리드가 없다: %q", got)
	}
	// content/canvas는 배치 CSS가 없다 (세로 스택·자유 배치는 렌더러가 소유) — 맵에 없어야
	// 스테이지에 레이아웃 클래스가 붙지 않는다.
	if _, ok := styles["content"]; ok {
		t.Error("content가 배치 맵에 있다 — 빈 레이아웃은 제외되어야 한다")
	}
	if _, ok := styles["canvas"]; ok {
		t.Error("canvas가 배치 맵에 있다 — 빈 레이아웃은 제외되어야 한다")
	}
}

// TestBuiltinThemesCoverEverySlotArea는 **슬롯 카탈로그와 테마의 대조**다 (ADR 0044 D3).
//
// soml이 슬롯 어휘를 소유하고(LayoutSlots) 영역의 실체는 테마가 소유하므로(ADR 0035), 두
// 목록이 어긋나면 슬롯 이름은 어휘에 있는데 **놓일 자리가 없다**. 그 구멍은 곧 제품의 구멍이다:
// `layout="quote"` + `slot="quote"`가 웹에서는 조용히 무효, PPTX에서는 플레이스홀더 발행이 됐다.
//
// 이름까지 대조하는 이유: `grid-template-areas`에 이름 하나가 빠지면 그 자식의 `grid-area`가
// 정의되지 않은 영역을 가리켜 **자동 배치로 조용히 떨어진다** (appendSlot의 주석과 같은 실패).
//
// **내장 테마 전수로 돈다** (P4.2): 미구현 레이아웃 폴백은 구현하지 않기로 했으므로(ADR 0052)
// 테마가 슬롯 하나를 빠뜨리면 그 테마를 고른 덱에서 그 슬롯이 조용히 무효가 된다.
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
			// canvas — 슬롯 어휘가 비어 있으니 영역 CSS도 있으면 안 된다 (쓸 수 없는 규칙이다).
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
		// 슬롯 영역 규칙은 흐름 규칙과 배타적이므로(layoutStagesCSS의 `:not(.balpyo-slots)`)
		// display부터 스스로 갖춰야 한다 — grid가 아니면 `grid-area`가 아무것도 하지 않는다.
		if !strings.Contains(decl, "display: grid") {
			t.Errorf("%s의 슬롯 영역 CSS가 display: grid를 스스로 갖지 않는다: %q", name, decl)
		}
		for _, slot := range slots {
			if !strings.Contains(decl, slot.Name) {
				t.Errorf("%s의 슬롯 영역 CSS에 슬롯 %q의 자리가 없다: %q", name, slot.Name, decl)
			}
		}
	}
	// 린터도 같이 본다 — 새 파일이 외관 프로퍼티를 들이면 여기서 잡힌다 (TestMinimalLayoutStyles의
	// 흐름 CSS와 같은 계약: 배치 CSS는 PPTX 마스터에 도달하지 못하는 외관을 담을 수 없다).
	if diags := lintLayouts(pkg.slotLayouts); len(diags) != 0 {
		t.Errorf("%s 슬롯 영역 CSS가 진단을 냈다:\n%v", pkg.manifest.Name, diags)
	}
}

// TestBuiltinThemesImplementEveryStandardLayout은 내장 테마 셋이 표준 카탈로그 11종을 **전부**
// 구현함을 못 박는다 (ADR 0052의 결정 7).
//
// 문서가 오래 적어 온 "테마가 빠뜨린 표준 레이아웃은 minimal로 폴백하고 경고한다"는 코드가 된
// 적이 없다. 폴백을 구현하는 대신 **계약을 테스트로** 세운 이유는 폴백이 섞인 테마가 디자인
// 시스템이 아니라는 것이다: editorial 덱의 stat 슬라이드만 minimal의 짜임으로 나오면 저자는
// 그 한 장이 왜 다른지 알 길이 없다. 외부 테마(P4.3)가 이 규칙을 다시 만난다.
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
		// 배치 CSS도 전 테마에서 배치 전용이어야 한다 — 외관은 components.json의 몫이고,
		// 레이아웃 CSS의 색·글꼴은 PPTX 마스터에 도달하지 못한다 (theme_lint.go).
		if diags := lintLayouts(pkg.layouts); len(diags) != 0 {
			t.Errorf("%s 레이아웃 CSS가 진단을 냈다:\n%v", name, diags)
		}
	}
}

// 내장 테마 목록은 embed 트리에서 나온다 — `balpyo init`의 `--theme` 어휘가 이 값이므로,
// 내장 테마를 늘리면 CLI의 어휘가 함께 열려야 한다 (ADR 0048).
func TestBuiltinThemesAreLoadable(t *testing.T) {
	names := BuiltinThemes()
	if len(names) == 0 {
		t.Fatal("내장 테마가 하나도 없다")
	}
	// 목록은 알파벳 정렬이므로 minimal이 첫 이름이라는 보장은 없다 (P4.2에서 editorial이 들어와
	// 첫 자리를 가져갔다). 못 박는 것은 **기본 테마가 목록에 있다**는 것이다 — 기본값이 어휘
	// 밖이면 설정을 쓰지 않는 프로젝트가 빌드되지 않는다.
	if !IsBuiltinTheme(DefaultTheme) {
		t.Errorf("기본 테마 %q가 내장 목록에 없다 (%v)", DefaultTheme, names)
	}
	// 목록에 있는 이름은 전부 **실제로 로드된다** — 디렉토리만 있고 theme.json이 없는 이름이
	// 어휘에 들어오면 init이 빌드되지 않는 프로젝트를 만든다.
	for _, name := range names {
		if _, err := loadTheme(builtinThemes, path.Join("themes", name)); err != nil {
			t.Errorf("내장 테마 %q 로드 실패: %v", name, err)
		}
		if !IsBuiltinTheme(name) {
			t.Errorf("IsBuiltinTheme(%q) = false", name)
		}
	}
	if IsBuiltinTheme("corp") {
		t.Error(`IsBuiltinTheme("corp") = true — 없는 테마를 받아들인다`)
	}
}
