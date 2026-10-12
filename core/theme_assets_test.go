package core

import (
	"bytes"
	"image/jpeg"
	"io/fs"
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
    {"layouts":["title"],"image":"assets/cover.png"},
    {"layouts":["section-divider"],"fill":"gradient(accent1 accent6, 135deg)"},
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
		{`<slide layout="title"></slide>`, "", "themes/brand/assets/cover.png", false, []string{"themes/brand/assets/cover.png"}},
		{`<slide layout="section-divider"></slide>`, "gradient", "", true, []string{"themes/brand/assets/logo.png"}},
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
		if bg == nil {
			t.Fatalf("%s: 배경이 없다", tc.src)
		}
		gotFill := ""
		if bg.Fill != nil {
			gotFill = bg.Fill.Color.Token
			if bg.Fill.Color.IsGradient() {
				gotFill = "gradient"
			}
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

// openLog는 열린 파일 이름을 적는 파일 시스템이다 (디렉토리 목록은 적지 않는다).
type openLog struct {
	fs.FS
	opened map[string]bool
}

func (o *openLog) Open(name string) (fs.File, error) {
	o.opened[name] = true
	return o.FS.Open(name)
}

func (o *openLog) ReadDir(name string) ([]fs.DirEntry, error) { return fs.ReadDir(o.FS, name) }

// 테마 로드는 `assets/`를 색인만 한다 — 선언이 가리키지 않은 파일은 열지 않는다. 아이콘 수천 장을 가진
// 테마도 로드 비용이 덱이 쓰는 몇 장만큼이어야 한다.
func TestThemeLoadReadsOnlyDeclaredAssets(t *testing.T) {
	log := &openLog{FS: brandProject(t, brandManifest), opened: map[string]bool{}}
	if _, err := loadThemeNamed(log, "brand"); err != nil {
		t.Fatal(err)
	}
	if !log.opened["themes/brand/assets/cover.png"] {
		t.Error("선언이 가리킨 cover.png를 읽지 않았다")
	}
	if log.opened["themes/brand/assets/unused.png"] {
		t.Error("선언이 가리키지 않은 unused.png를 열었다")
	}
}

// 배경 선언의 검사 — 테마의 실수는 로딩에서 멈춘다 (그 레이아웃을 쓴 슬라이드가 없어도).
func TestThemeBackgroundsRejectBadDecls(t *testing.T) {
	for _, tc := range []struct{ name, backgrounds, want string }{
		{"빈 선언", `[{"layouts":["title"]}]`, "has neither fill nor image"},
		{"채움과 그림을 함께", `[{"fill":"bg","image":"assets/cover.png"}]`, "cannot have both fill and image"},
		{"리터럴 색", `[{"fill":"#FF0000"}]`, `fill "#FF0000" is not a color token reference of this theme`},
		{"none", `[{"fill":"none"}]`, "fill cannot be none"},
		{"모르는 레이아웃", `[{"layouts":["cover"],"fill":"bg"}]`, `layouts: "cover" is not a layout of this theme`},
		{"겹친 레이아웃", `[{"layouts":["title"],"fill":"bg"},{"layouts":["title"],"fill":"surface"}]`, `the background of layout "title" appears in two declarations`},
		{"기본 배경 둘", `[{"fill":"bg"},{"fill":"surface"}]`, "default background without layouts is declared twice"},
		{"없는 그림", `[{"image":"assets/none.png"}]`, `"assets/none.png", which is not in the theme package`},
		{"그림이 아닌 파일", `[{"image":"assets/notes.txt"}]`, "is not an image format"},
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

// EXIF 방향이 적힌 JPEG 배경은 **세운 픽셀**로 다시 인코딩한 그림을 쓴다 — PPTX의 슬라이드 배경
// (`p:bg`의 `blipFill`)에는 회전을 적을 자리가 없어 PowerPoint가 저장된 픽셀을 그대로 펴기 때문이다. 같은
// 파일을 가리키는 로고는 원래 바이트와 방향을 그대로 쓰고, 다시 인코딩한 바이트는 로드마다 같다.
func TestThemeBackgroundUprightsEXIFOrientation(t *testing.T) {
	// 240×120으로 저장되고 방향 6(90°)이 적힌 사진이다.
	rotated, err := os.ReadFile("../testdata/fixtures/05_image/assets/rotated.jpg")
	if err != nil {
		t.Fatal(err)
	}
	project := localTheme("brand", `{"name":"brand","extends":"minimal",
  "backgrounds":[{"layouts":["title"],"image":"assets/rotated.jpg"}],
  "chrome":[{"kind":"logo","align":"center","image":"assets/rotated.jpg","height":24,"alt":"브랜드"}]}`, "", "", nil)
	project["themes/brand/assets/rotated.jpg"] = &fstest.MapFile{Data: rotated}
	build := func() *soml.Deck {
		deck, diags := themeDeckIn(t, project, "brand", `<slide layout="title"></slide>`)
		if len(diags) != 0 {
			t.Fatalf("진단이 없어야 한다: %v", diags)
		}
		if d := ResolveAssets(deck, project); len(d) != 0 {
			t.Fatalf("에셋 진단: %v", d)
		}
		return deck
	}
	deck := build()
	const uprightKey = "themes/brand/upright/assets/rotated.jpg"
	if bg := deck.Slides[0].Background; bg == nil || bg.Image != uprightKey {
		t.Fatalf("배경 = %+v, want 그림 %q", bg, uprightKey)
	}
	upright, ok := deck.Assets.Get(uprightKey)
	if !ok {
		t.Fatalf("세운 배경 그림이 에셋 표에 없다: %v", showAssets(deck.Assets))
	}
	if upright.Format != soml.FormatJPEG || upright.Width != 120 || upright.Height != 240 || upright.Orientation != 0 {
		t.Errorf("세운 배경 그림 = %s, want jpeg 120x240 o0", showAssets(soml.Assets{upright}))
	}
	if o := soml.JPEGOrientation(upright.Bytes); o != 0 {
		t.Errorf("세운 배경 그림의 바이트에 방향 %d가 남아 있다", o)
	}
	config, err := jpeg.DecodeConfig(bytes.NewReader(upright.Bytes))
	if err != nil || config.Width != 120 || config.Height != 240 {
		t.Errorf("세운 배경 그림의 저장된 크기 = %d×%d (%v), want 120×240", config.Width, config.Height, err)
	}
	logo, ok := deck.Assets.Get("themes/brand/assets/rotated.jpg")
	if !ok || logo.Orientation != 6 || !bytes.Equal(logo.Bytes, rotated) {
		t.Errorf("로고 그림은 원래 바이트와 방향 6이어야 한다: %v", showAssets(deck.Assets))
	}
	again, _ := build().Assets.Get(uprightKey)
	if !bytes.Equal(again.Bytes, upright.Bytes) {
		t.Error("같은 입력의 세운 배경 그림이 로드마다 다른 바이트다")
	}
}
