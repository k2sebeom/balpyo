package core

import (
	"os"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/k2sebeom/balpyo/soml"
)

// pngFile은 시험용 그림 파일 하나다 (testdata/theme_assets — 2색 팔레트 PNG).
func pngFile(t *testing.T, name string) *fstest.MapFile {
	t.Helper()
	data, err := os.ReadFile("testdata/theme_assets/" + name)
	if err != nil {
		t.Fatal(err)
	}
	return &fstest.MapFile{Data: data}
}

// brandProject는 minimal을 물려받아 배경과 로고를 더한 로컬 테마 하나다.
func brandProject(t *testing.T, manifest string) fstest.MapFS {
	t.Helper()
	fsys := localTheme("brand", manifest, "", "", nil)
	fsys["themes/brand/assets/cover.png"] = pngFile(t, "cover.png")
	fsys["themes/brand/assets/logo.png"] = pngFile(t, "logo.png")
	fsys["themes/brand/assets/unused.png"] = pngFile(t, "logo.png")
	return fsys
}

const brandManifest = `{"name":"brand","extends":"minimal",
  "backgrounds":[
    {"layouts":["title","section-divider"],"fill":"gradient(accent1 accent6, 135deg)","image":"assets/cover.png"},
    {"fill":"surface"}
  ],
  "chrome":[{"kind":"logo","align":"center","image":"assets/logo.png","height":24,"alt":"브랜드","skipLayouts":["title"]}]}`

// 배경은 레이아웃이 고른다: 선언한 레이아웃은 그 배경, 나머지는 기본 배경이다. 로고는 요청 없이 켜지고
// (`balpyo.json`이 정할 것이 없다) 제외 규칙을 따른다. 쓴 그림만 덱의 에셋 표에 들어간다 — 패키지 안의 쓰지
// 않는 파일은 산출물에 나가지 않는다.
func TestThemeBackgroundsAndLogo(t *testing.T) {
	project := brandProject(t, brandManifest)
	for _, tc := range []struct {
		src      string
		fill     string
		image    string
		logo     bool
		assetKey []string
	}{
		{`<slide layout="title"></slide>`, "gradient", "themes/brand/assets/cover.png", false, []string{"themes/brand/assets/cover.png"}},
		{`<slide><p>본문</p></slide>`, "surface", "", true, []string{"themes/brand/assets/logo.png"}},
	} {
		deck, diags := themeDeckIn(t, project, "brand", tc.src)
		if len(diags) != 0 {
			t.Fatalf("진단이 없어야 한다: %v", diags)
		}
		if d := ResolveAssets(deck, project); len(d) != 0 {
			t.Fatalf("에셋 진단: %v", d)
		}
		bg := deck.Slides[0].Background
		if bg == nil || bg.Fill == nil {
			t.Fatalf("%s: 배경이 없다", tc.src)
		}
		gotFill := bg.Fill.Color.Token
		if bg.Fill.Color.IsGradient() {
			gotFill = "gradient"
		}
		if gotFill != tc.fill || bg.Image != tc.image {
			t.Errorf("%s: 배경 = %s + %q, want %s + %q", tc.src, gotFill, bg.Image, tc.fill, tc.image)
		}
		var logo *soml.ChromeItem
		for i := range deck.Slides[0].Chrome {
			if deck.Slides[0].Chrome[i].Kind == soml.ChromeLogo {
				logo = &deck.Slides[0].Chrome[i]
			}
		}
		if (logo != nil) != tc.logo {
			t.Errorf("%s: 로고 = %v, want %v", tc.src, logo != nil, tc.logo)
		}
		if logo != nil && (logo.Image != "themes/brand/assets/logo.png" || logo.Height != 24 || logo.Alt != "브랜드") {
			t.Errorf("로고 = %+v", *logo)
		}
		var keys []string
		for _, a := range deck.Assets {
			keys = append(keys, a.Key)
		}
		if strings.Join(keys, ",") != strings.Join(tc.assetKey, ",") {
			t.Errorf("%s: 덱 에셋 = %v, want %v", tc.src, keys, tc.assetKey)
		}
	}
}

// 자식 테마가 부모의 배경 선언을 물려받으면 그림도 부모의 것이다 — 에셋 키가 그림을 가진 테마의 자리다.
// 자식이 같은 경로의 파일을 두면 그 파일이 이긴다 (파일 단위 병합).
func TestThemeAssetsFollowTheOwner(t *testing.T) {
	parent := brandProject(t, brandManifest)
	child := localTheme("brand-dark", `{"name":"brand-dark","extends":"brand"}`, "", "", nil)
	pkg := mustLoadLocal(t, mergeFS(parent, child), "brand-dark")
	if bg := pkg.backgrounds.For(soml.LayoutTitle); bg == nil || bg.Image != "themes/brand/assets/cover.png" {
		t.Errorf("물려받은 배경 그림 = %+v, want 부모의 키", bg)
	}
	child["themes/brand-dark/assets/cover.png"] = pngFile(t, "logo.png")
	pkg = mustLoadLocal(t, mergeFS(parent, child), "brand-dark")
	if bg := pkg.backgrounds.For(soml.LayoutTitle); bg == nil || bg.Image != "themes/brand-dark/assets/cover.png" {
		t.Errorf("자식이 덮은 배경 그림 = %+v, want 자식의 키", bg)
	}
}

// 배경 선언의 검사 — 테마의 실수는 로딩에서 멈춘다 (그 레이아웃을 쓴 슬라이드가 없어도).
func TestThemeBackgroundsRejectBadDecls(t *testing.T) {
	for _, tc := range []struct{ name, backgrounds, want string }{
		{"빈 선언", `[{"layouts":["title"]}]`, "fill도 image도 없다"},
		{"리터럴 색", `[{"fill":"#FF0000"}]`, `fill "#FF0000"가 이 테마의 색 토큰 참조가 아니다`},
		{"none", `[{"fill":"none"}]`, "fill에 none은 쓸 수 없다"},
		{"모르는 레이아웃", `[{"layouts":["cover"],"fill":"bg"}]`, `layouts에 "cover"는 이 테마의 레이아웃이 아니다`},
		{"겹친 레이아웃", `[{"layouts":["title"],"fill":"bg"},{"layouts":["title"],"fill":"surface"}]`, `레이아웃 "title"의 배경이 두 선언에 나온다`},
		{"기본 배경 둘", `[{"fill":"bg"},{"fill":"surface"}]`, "기본 배경이 두 번 선언되었다"},
		{"없는 그림", `[{"image":"assets/none.png"}]`, `"assets/none.png"가 테마 패키지에 없다`},
		{"그림이 아닌 파일", `[{"image":"assets/notes.txt"}]`, "그림 포맷이 아니다"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			project := brandProject(t, `{"name":"brand","extends":"minimal","backgrounds":`+tc.backgrounds+`}`)
			project["themes/brand/assets/notes.txt"] = &fstest.MapFile{Data: []byte("메모")}
			_, err := loadThemeNamed(project, "brand")
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Errorf("에러 = %v, want %q", err, tc.want)
			}
		})
	}
}
