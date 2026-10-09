package core

import (
	"errors"
	"fmt"
	"io/fs"
	"path"
	"strings"

	"github.com/k2sebeom/balpyo/soml"
)

// 이 파일은 **테마가 소유한 자산**이다 — 테마 패키지의 `assets/` 아래 그림과, 그것을 가리키는 두 선언
// (`theme.json`의 `backgrounds`와 `logo` 크롬)이다.
//
// 자산이 테마에 사는 이유: 회사 템플릿의 배경 그림과 로고는 원고의 내용이 아니라 테마의 정체다. 덱이
// 그림을 들고 있으면(`layout="canvas"` + `<image>`) 슬롯과 PPTX 플레이스홀더를 잃고, 저자가 좌표를 쓰고,
// 테마를 바꿔도 그림이 남는다.
//
// 테마 로딩이 바이트를 읽는다 (에셋 패스가 아니다): 테마 파일(JSON · CSS)을 읽는 자리가 이미 여기이고,
// 내장 테마의 자산은 프로젝트 파일 시스템에 없다. 읽은 그림은 soml.Theme.Assets에 실리고, 에셋 패스가
// 슬라이드가 실제로 쓴 것만 덱의 에셋 표로 옮긴다 — 렌더러는 지금처럼 덱의 표 하나만 읽는다.

// themeAssetDir는 테마 패키지에서 자산이 사는 디렉토리다.
const themeAssetDir = "assets"

// rawAsset은 테마 패키지 하나가 가진 자산 파일 하나다 — 상속 병합 전의 원문이다.
type rawAsset struct {
	// owner는 이 파일을 가진 테마 이름이다 — 에셋 키(`themes/<owner>/assets/…`)가 여기서 나온다.
	// 자식이 부모의 선언을 물려받아도 그림은 부모의 것이므로 키도 부모의 자리다.
	owner string
	data  []byte
}

// readThemeAssets는 dir/assets 아래 파일을 전부 읽는다 (없으면 빈 표). 키는 테마 루트 기준
// 상대 경로(`assets/bg.png`)다 — 선언이 그 이름으로 가리킨다.
func readThemeAssets(fsys fs.FS, dir, owner string) (map[string]rawAsset, error) {
	root := path.Join(dir, themeAssetDir)
	if _, err := fs.Stat(fsys, root); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("%s를 읽을 수 없다: %w", root, err)
	}
	assets := map[string]rawAsset{}
	err := fs.WalkDir(fsys, root, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		data, err := fs.ReadFile(fsys, p)
		if err != nil {
			return err
		}
		rel := strings.TrimPrefix(p, dir+"/")
		assets[rel] = rawAsset{owner: owner, data: data}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("%s를 읽을 수 없다: %w", root, err)
	}
	return assets, nil
}

// mergeAssetFiles는 자산 표를 **파일 단위로** 병합한다 — 같은 경로면 자식의 파일이 이긴다 (레이아웃
// CSS와 같은 단위다). 자식이 `assets/logo.png` 하나만 바꾸면 부모의 배경 그림은 그대로 남는다.
func mergeAssetFiles(parent, child map[string]rawAsset) map[string]rawAsset {
	if len(parent) == 0 {
		return child
	}
	out := make(map[string]rawAsset, len(parent)+len(child))
	for k, v := range parent {
		out[k] = v
	}
	for k, v := range child {
		out[k] = v
	}
	return out
}

// themeAssetKey는 테마 자산의 에셋 키다 — 웹 산출물에서 이 경로로 파일이 놓인다.
func themeAssetKey(owner, rel string) string {
	return path.Join("themes", owner, rel)
}

// assetResolver는 선언이 가리키는 자산을 판정하고 **가리킨 순서대로** 모은다.
type assetResolver struct {
	files map[string]rawAsset
	used  soml.Assets
}

// resolve는 선언의 경로(`assets/bg.png`)를 판정된 에셋 키로 바꾼다. where는 진단의 자리다.
func (a *assetResolver) resolve(where, rel string) (string, error) {
	clean := path.Clean(rel)
	if clean != rel || !strings.HasPrefix(rel, themeAssetDir+"/") {
		return "", fmt.Errorf("%s의 %q는 테마의 %s/ 아래 경로여야 한다 (예: %s/bg.png)", where, rel, themeAssetDir, themeAssetDir)
	}
	file, ok := a.files[rel]
	if !ok {
		return "", fmt.Errorf("%s가 가리키는 %q가 테마 패키지에 없다", where, rel)
	}
	key := themeAssetKey(file.owner, rel)
	for _, asset := range a.used {
		if asset.Key == key {
			return key, nil
		}
	}
	asset, msg := decodeImageAsset(key, file.data)
	if msg != "" {
		return "", fmt.Errorf("%s: %s", where, msg)
	}
	a.used = append(a.used, asset)
	return key, nil
}

// backgroundFile은 theme.json의 배경 선언 하나다 (원문).
//
// `layouts`를 생략한 선언은 **기본 배경**이다 (나머지 모든 레이아웃). 채움과 그림은 함께 쓸 수 있다 —
// 그림이 채움 위에 놓인다 (투명한 PNG 아래가 채움이다).
type backgroundFile struct {
	Layouts []string `json:"layouts"`
	Fill    string   `json:"fill"`
	Image   string   `json:"image"`
}

// backgroundDecls는 검사가 끝난 배경 표다 — 레이아웃 → 배경, 그리고 기본값.
type backgroundDecls struct {
	byLayout map[soml.SlideLayout]*soml.Background
	fallback *soml.Background
}

// For는 이 레이아웃의 배경이다 (선언이 없으면 nil — 테마의 bg 색 그대로).
func (b backgroundDecls) For(layout soml.SlideLayout) *soml.Background {
	if bg, ok := b.byLayout[layout]; ok {
		return bg
	}
	return b.fallback
}

// loadBackgrounds는 배경 선언을 검사한다.
//
// 규칙은 셋이다: 선언마다 채움이나 그림이 하나는 있다 · 레이아웃은 이 테마의 것이고 한 선언에만 나온다 ·
// 기본 선언(`layouts` 없음)은 하나다. 한 레이아웃이 두 선언에 나오면 어느 것이 이기는지 테마 작성자가
// 외워야 한다 — 순서 규칙을 두는 대신 겹침을 막는다.
func loadBackgrounds(manifest themeManifest, theme soml.Theme, assets *assetResolver) (backgroundDecls, error) {
	decls := backgroundDecls{byLayout: map[soml.SlideLayout]*soml.Background{}}
	layouts := make(map[string]bool, len(manifest.Layouts))
	for _, name := range manifest.Layouts {
		layouts[name] = true
	}
	for i, raw := range manifest.Backgrounds {
		where := fmt.Sprintf("theme.json의 backgrounds[%d]", i)
		if raw.Fill == "" && raw.Image == "" {
			return decls, fmt.Errorf("%s에 fill도 image도 없다 — 배경은 둘 중 하나는 있어야 한다", where)
		}
		bg := &soml.Background{}
		if raw.Fill != "" {
			fill, err := themeFill(theme, raw.Fill)
			if err != nil {
				return decls, fmt.Errorf("%s의 fill: %w", where, err)
			}
			if fill.None {
				return decls, fmt.Errorf("%s의 fill에 none은 쓸 수 없다 — 배경을 끄려면 선언을 지운다", where)
			}
			if bad, ok := notTokenRef(theme, fill.Color); ok {
				return decls, fmt.Errorf("%s의 fill %q가 이 테마의 색 토큰 참조가 아니다 — 테마는 색 토큰 이름(변형 · 토큰 그라디언트 포함)만 쓴다 (이 테마: %s)",
					where, bad, strings.Join(theme.ColorNames(), ", "))
			}
			bg.Fill = fill
		}
		if raw.Image != "" {
			key, err := assets.resolve(where+"의 image", raw.Image)
			if err != nil {
				return decls, err
			}
			bg.Image = key
		}
		if len(raw.Layouts) == 0 {
			if decls.fallback != nil {
				return decls, fmt.Errorf("%s: layouts 없는 기본 배경이 두 번 선언되었다", where)
			}
			decls.fallback = bg
			continue
		}
		for _, name := range raw.Layouts {
			if !layouts[name] {
				return decls, fmt.Errorf("%s의 layouts에 %q는 이 테마의 레이아웃이 아니다 (이 테마의 레이아웃: %s)",
					where, name, strings.Join(sortedKeys(layouts), ", "))
			}
			if _, dup := decls.byLayout[soml.SlideLayout(name)]; dup {
				return decls, fmt.Errorf("%s: 레이아웃 %q의 배경이 두 선언에 나온다 — 한 레이아웃의 배경은 하나다", where, name)
			}
			decls.byLayout[soml.SlideLayout(name)] = bg
		}
	}
	return decls, nil
}

// notTokenRef는 색이 토큰 참조가 아니면 그 조각(리터럴 · 모르는 이름)을 돌려준다 — 그라디언트면 문제의
// 스톱이다. 테마가 쓰는 색은 토큰 참조뿐이다: PPTX에서 `schemeClr`로 남아야 테마 색 편집을 따라간다.
func notTokenRef(theme soml.Theme, c soml.Color) (string, bool) {
	one := func(c soml.Color) (string, bool) {
		if !c.IsToken() {
			return "#" + c.RGB, true
		}
		if _, ok := theme.Color(c.Token); !ok {
			return c.Token, true
		}
		return "", false
	}
	if c.IsGradient() {
		for _, stop := range c.Gradient.Stops {
			if bad, ok := one(stop); ok {
				return bad, true
			}
		}
		return "", false
	}
	return one(c)
}
