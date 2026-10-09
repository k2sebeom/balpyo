package core

import (
	"os"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/k2sebeom/balpyo/soml"
)

// iconProject는 brand(minimal 상속 · 배경 · 로고) 위에 아이콘 두 장을 더한 프로젝트다.
func iconProject(t *testing.T) fstest.MapFS {
	t.Helper()
	fsys := brandProject(t, brandManifest)
	fsys["themes/brand/assets/icons/a.png"] = pngFile(t, "logo.png")
	fsys["themes/brand/assets/icons/b.png"] = pngFile(t, "cover.png")
	return fsys
}

// themeImageDeck은 slide를 theme으로 해석하고 에셋 패스까지 돌린다 — 두 단계의 진단을 합쳐 돌려준다.
func themeImageDeck(t *testing.T, project fstest.MapFS, theme, slide string) (*soml.Deck, soml.Diagnostics) {
	t.Helper()
	deck, diags := themeDeckIn(t, project, theme, slide)
	return deck, append(diags, ResolveAssets(deck, project)...)
}

func imageSrcOf(deck *soml.Deck, i int) string {
	var srcs []string
	for _, node := range deck.Slides[0].Content {
		if elem, ok := node.(*soml.Element); ok && elem.ImageStyle() != nil {
			srcs = append(srcs, elem.ImageStyle().Src)
		}
	}
	return srcs[i]
}

func assetKeys(assets soml.Assets) string {
	var keys []string
	for _, a := range assets {
		keys = append(keys, a.Key)
	}
	return strings.Join(keys, ",")
}

// `theme:` 경로는 테마의 assets/ 기준이고, 확정 값은 그 파일을 가진 테마의 에셋 키다. 쓴 그림만 덱의 에셋
// 표에 들어간다 (b.png는 쓰지 않았다).
func TestThemeImageSrcResolvesToTheThemeAssetKey(t *testing.T) {
	deck, diags := themeImageDeck(t, iconProject(t), "brand",
		`<slide><image src="theme:icons/a.png" w="64"/><image src="theme:./icons/a.png" w="32"/></slide>`)
	if len(diags) != 0 {
		t.Fatalf("진단이 없어야 한다: %v", diags)
	}
	for i := 0; i < 2; i++ {
		if got := imageSrcOf(deck, i); got != "themes/brand/assets/icons/a.png" {
			t.Errorf("그림 %d의 Src = %q, want 테마 자산 키", i, got)
		}
	}
	if got := assetKeys(deck.Assets); got != "themes/brand/assets/icons/a.png,themes/brand/assets/logo.png" {
		t.Errorf("덱 에셋 = %s", got)
	}
	if asset, ok := deck.Assets.Get("themes/brand/assets/icons/a.png"); !ok || asset.Width <= 0 || len(asset.Bytes) == 0 {
		t.Errorf("에셋이 판정되지 않았다: %+v", asset)
	}
}

// 설치한 테마의 그림도 같다 — 프로젝트에 themes/가 없어도 된다.
func TestThemeImageSrcFromInstalledTheme(t *testing.T) {
	home := iconProject(t)
	project := WithInstalledThemes(fstest.MapFS{}, home)
	deck, diags := themeDeckIn(t, project, "brand", `<slide><image src="theme:icons/a.png" w="64"/></slide>`)
	diags = append(diags, ResolveAssets(deck, project)...)
	if len(diags) != 0 {
		t.Fatalf("진단이 없어야 한다: %v", diags)
	}
	if _, ok := deck.Assets.Get("themes/brand/assets/icons/a.png"); !ok {
		t.Errorf("덱 에셋 = %s", assetKeys(deck.Assets))
	}
}

// 상속 체인에서 찾는다: 부모에만 있는 파일은 부모의 키, 둘 다 가진 파일은 자식의 바이트와 자식의 키다.
func TestThemeImageSrcFollowsTheExtendsChain(t *testing.T) {
	child := localTheme("brand-dark", `{"name":"brand-dark","extends":"brand"}`, "", "", nil)
	child["themes/brand-dark/assets/icons/b.png"] = pngFile(t, "logo.png")
	project := mergeFS(iconProject(t), child)
	deck, diags := themeImageDeck(t, project, "brand-dark",
		`<slide><image src="theme:icons/a.png" w="64"/><image src="theme:icons/b.png" w="64"/></slide>`)
	if len(diags) != 0 {
		t.Fatalf("진단이 없어야 한다: %v", diags)
	}
	if got := imageSrcOf(deck, 0); got != "themes/brand/assets/icons/a.png" {
		t.Errorf("부모에만 있는 그림 = %q, want 부모의 키", got)
	}
	if got := imageSrcOf(deck, 1); got != "themes/brand-dark/assets/icons/b.png" {
		t.Errorf("자식이 덮은 그림 = %q, want 자식의 키", got)
	}
	asset, _ := deck.Assets.Get("themes/brand-dark/assets/icons/b.png")
	if string(asset.Bytes) != string(child["themes/brand-dark/assets/icons/b.png"].Data) {
		t.Error("자식이 덮은 그림의 바이트가 자식의 파일이 아니다")
	}
}

// 같은 파일을 배경(테마 선언)과 `theme:` 그림이 함께 쓰면 덱의 에셋 표에 한 번 들어간다.
func TestThemeImageSrcSharesTheBackgroundAsset(t *testing.T) {
	deck, diags := themeImageDeck(t, iconProject(t), "brand",
		`<slide layout="title"><image slot="title" src="theme:cover.png" w="64"/></slide>`)
	if len(diags) != 0 {
		t.Fatalf("진단이 없어야 한다: %v", diags)
	}
	if got := assetKeys(deck.Assets); got != "themes/brand/assets/cover.png" {
		t.Errorf("덱 에셋 = %s, want 배경과 그림이 한 항목", got)
	}
	if got := assetKeys(deck.Theme.Assets); strings.Count(got, "cover.png") != 1 {
		t.Errorf("테마 에셋 = %s, want cover.png 한 번", got)
	}
}

// 체인에 없는 파일은 값 스팬의 에러이고, 본 테마들을 잎부터 나열한다. 다른 테마로 바꾼 덱도 같다 — 빈
// 그림으로 빌드되지 않는다.
func TestThemeImageSrcMissingFileListsTheChain(t *testing.T) {
	child := localTheme("brand-dark", `{"name":"brand-dark","extends":"brand"}`, "", "", nil)
	project := mergeFS(iconProject(t), child)
	for _, tc := range []struct{ theme, want string }{
		{"brand-dark", `theme:icons/x.png을 테마 "brand-dark"(→ "brand" → "minimal")의 assets/에서 찾을 수 없다`},
		{"minimal", `theme:icons/a.png을 테마 "minimal"의 assets/에서 찾을 수 없다`},
	} {
		src := "theme:icons/x.png"
		if tc.theme == "minimal" {
			src = "theme:icons/a.png"
		}
		deck, diags := themeImageDeck(t, project, tc.theme, `<slide><image src="`+src+`" w="64"/></slide>`)
		if len(diags) != 1 || diags[0].Severity != soml.SeverityError || !strings.Contains(diags[0].Message, tc.want) {
			t.Fatalf("%s: 진단 = %v, want %q", tc.theme, diags, tc.want)
		}
		if diags[0].Span.Start.Col != 20 {
			t.Errorf("%s: 진단의 열 = %d, want 값 스팬(20)", tc.theme, diags[0].Span.Start.Col)
		}
		if _, ok := deck.Assets.Get("themes/brand/assets/icons/x.png"); imageSrcOf(deck, 0) != "" || ok {
			t.Errorf("%s: 찾지 못한 그림이 확정 값에 실렸다", tc.theme)
		}
	}
}

// 경로 규칙은 프로젝트 경로와 같다 — 절대 경로 · assets/ 밖 · 역슬래시 · 열지 않은 포맷은 값 스팬의 에러다.
// 내용이 확장자와 다른 파일은 테마의 배경과 같은 판정에 걸린다.
func TestThemeImageSrcRejectsBadPaths(t *testing.T) {
	project := iconProject(t)
	jpeg, err := os.ReadFile("../testdata/fixtures/05_image/assets/photo.jpg")
	if err != nil {
		t.Fatal(err)
	}
	project["themes/brand/assets/icons/fake.png"] = &fstest.MapFile{Data: jpeg}
	project["themes/brand/assets/icons/a.gif"] = pngFile(t, "logo.png")
	for _, tc := range []struct{ src, want string }{
		{"theme:", "theme: 뒤에 경로가 없다"},
		{"theme:../x.png", "src가 테마의 assets/ 밖을 가리킨다"},
		{"theme:icons/../../x.png", "src가 테마의 assets/ 밖을 가리킨다"},
		{"theme:/x.png", "theme: 경로는 테마의 assets/ 기준 상대 경로다"},
		{`theme:icons\a.png`, "src의 경로 구분자는"},
		{"theme:icons/a.gif", `src의 ".gif"는 v1이 그리지 못한다`},
		{"theme:icons", "src에 확장자가 없다"},
		{"theme:icons/fake.png", "확장자는 png를 말하는데 내용은 jpeg다"},
	} {
		_, diags := themeImageDeck(t, project, "brand", `<slide><image src="`+tc.src+`" w="64"/></slide>`)
		if len(diags) != 1 || !strings.Contains(diags[0].Message, tc.want) {
			t.Errorf("%s: 진단 = %v, want %q", tc.src, diags, tc.want)
		}
	}
}

// 프로젝트 경로는 테마 자산 키와 모양이 같아도 프로젝트 파일 시스템에서 찾는다 — 설치한 테마의 파일이
// `themes/<이름>/…` 경로로 새어 나오지 않는다. 로컬 테마면 같은 파일이므로 표에 한 번 실린다.
func TestProjectSrcDoesNotReachInstalledThemes(t *testing.T) {
	project := WithInstalledThemes(fstest.MapFS{}, iconProject(t))
	deck, diags := themeDeckIn(t, project, "brand",
		`<slide><image src="theme:icons/a.png" w="64"/><image src="themes/brand/assets/icons/a.png" w="64"/></slide>`)
	diags = append(diags, ResolveAssets(deck, project)...)
	if len(diags) != 1 || !strings.Contains(diags[0].Message, "src가 가리키는 파일이 없다") {
		t.Errorf("진단 = %v, want 프로젝트에 파일이 없다 (같은 키가 theme:으로 먼저 실렸어도)", diags)
	}
	local := iconProject(t)
	deck, diags = themeImageDeck(t, local, "brand",
		`<slide><image src="theme:icons/a.png" w="64"/><image src="themes/brand/assets/icons/a.png" w="64"/></slide>`)
	if len(diags) != 0 || assetKeys(deck.Assets) != "themes/brand/assets/icons/a.png,themes/brand/assets/logo.png" {
		t.Errorf("로컬 테마: 진단 = %v, 덱 에셋 = %s", diags, assetKeys(deck.Assets))
	}
}
