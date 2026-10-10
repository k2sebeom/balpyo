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
//
// 로딩이 `assets/`에서 미리 하는 일은 **경로 색인**뿐이다 — 바이트는 선언이 가리킨 파일만 그때 읽는다.
// 아이콘 수천 장을 담은 테마도 덱이 쓰는 몇 장만큼만 읽혀야 테마 로드 비용이 라이브러리 크기와
// 무관해진다 (watch는 저장할 때마다 테마를 다시 로드한다).

// themeAssetDir는 테마 패키지에서 자산이 사는 디렉토리다.
const themeAssetDir = "assets"

// rawAsset은 테마 패키지 하나가 가진 자산 파일 하나다 — 상속 병합 전의 원문이다. 바이트가 아니라
// **자리**를 담는다 (read가 읽는다).
type rawAsset struct {
	// owner는 이 파일을 가진 테마 이름이다 — 에셋 키(`themes/<owner>/assets/…`)가 여기서 나온다.
	// 자식이 부모의 선언을 물려받아도 그림은 부모의 것이므로 키도 부모의 자리다.
	owner string
	// fsys · path는 파일의 자리다 — 주인 테마의 FS다 (내장은 임베드 FS, 로컬은 프로젝트, 설치는 홈).
	fsys fs.FS
	path string
}

// read는 파일의 바이트를 읽는다.
func (a rawAsset) read() ([]byte, error) { return fs.ReadFile(a.fsys, a.path) }

// readThemeAssets는 dir/assets 아래 파일을 색인한다 (없으면 빈 표). 키는 테마 루트 기준
// 상대 경로(`assets/bg.png`)다 — 선언이 그 이름으로 가리킨다.
func readThemeAssets(fsys fs.FS, dir, owner string) (map[string]rawAsset, error) {
	root := path.Join(dir, themeAssetDir)
	if _, err := fs.Stat(fsys, root); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("cannot read %s: %w", root, err)
	}
	assets := map[string]rawAsset{}
	err := fs.WalkDir(fsys, root, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel := strings.TrimPrefix(p, dir+"/")
		assets[rel] = rawAsset{owner: owner, fsys: fsys, path: p}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("cannot read %s: %w", root, err)
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
		return "", fmt.Errorf("%s %q must be a path under the theme's %s/ (e.g. %s/bg.png)", where, rel, themeAssetDir, themeAssetDir)
	}
	key, found, msg := a.use(rel)
	if !found {
		return "", fmt.Errorf("%s points to %q, which is not in the theme package", where, rel)
	}
	if msg != "" {
		return "", fmt.Errorf("%s: %s", where, msg)
	}
	return key, nil
}

// use는 색인의 파일 하나(`assets/…`, 정규화된 경로)를 읽고 판정해 표에 싣는다 — 이미 실었으면 그 키를
// 그대로 돌려준다. found가 false면 색인에 없고, msg는 읽기·판정의 실패다.
//
// 테마 선언(배경 · 로고)과 슬라이드의 `theme:` 그림이 이 한 자리를 지난다: 같은 파일이 두 곳에서 쓰여도
// 키가 하나라 덱의 에셋 표에 한 번, PPTX 미디어 파트로 한 번 들어간다.
func (a *assetResolver) use(rel string) (key string, found bool, msg string) {
	file, ok := a.files[rel]
	if !ok {
		return "", false, ""
	}
	key = themeAssetKey(file.owner, rel)
	if _, ok := a.used.Get(key); ok {
		return key, true, ""
	}
	data, err := file.read()
	if err != nil {
		return "", true, fmt.Sprintf("cannot read %s: %v", key, err)
	}
	asset, msg := decodeImageAsset(key, data)
	if msg != "" {
		return "", true, msg
	}
	a.used = append(a.used, asset)
	return key, true, ""
}

// backgroundFile은 theme.json의 배경 선언 하나다 (원문).
//
// `layouts`를 생략한 선언은 **기본 배경**이다 (나머지 모든 레이아웃). 배경은 채움 **또는** 그림이다 —
// 둘을 겹치지 않는다: PPTX의 슬라이드 배경(`p:bgPr`)은 채움 요소 하나만 담으므로, 웹에서 그림 아래 깐
// 채움이 PPTX에서는 사라져 투명한 자리마다 두 타깃이 갈린다. 그라디언트 위 그림이 필요하면 그 둘을 그림
// 한 장으로 합친다.
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
		where := fmt.Sprintf("theme.json backgrounds[%d]", i)
		if raw.Fill == "" && raw.Image == "" {
			return decls, fmt.Errorf("%s has neither fill nor image — a background is one of the two", where)
		}
		if raw.Fill != "" && raw.Image != "" {
			return decls, fmt.Errorf("%s cannot have both fill and image — a PPTX slide background is a single fill, so the fill under the picture would be lost (merge the two into one picture)", where)
		}
		bg := &soml.Background{}
		if raw.Fill != "" {
			fill, err := themeFill(theme, raw.Fill)
			if err != nil {
				return decls, fmt.Errorf("%s fill: %w", where, err)
			}
			if fill.None {
				return decls, fmt.Errorf("%s fill cannot be none — to turn the background off, remove the declaration", where)
			}
			if bad, ok := notTokenRef(theme, fill.Color); ok {
				return decls, fmt.Errorf("%s fill %q is not a color token reference of this theme — themes use only color token names (including modifiers and token gradients) (this theme: %s)",
					where, bad, strings.Join(theme.ColorNames(), ", "))
			}
			bg.Fill = fill
		}
		if raw.Image != "" {
			key, err := assets.resolve(where+" image", raw.Image)
			if err != nil {
				return decls, err
			}
			bg.Image = key
		}
		if len(raw.Layouts) == 0 {
			if decls.fallback != nil {
				return decls, fmt.Errorf("%s: the default background without layouts is declared twice", where)
			}
			decls.fallback = bg
			continue
		}
		for _, name := range raw.Layouts {
			if !layouts[name] {
				return decls, fmt.Errorf("%s layouts: %q is not a layout of this theme (layouts in this theme: %s)",
					where, name, strings.Join(sortedKeys(layouts), ", "))
			}
			if _, dup := decls.byLayout[soml.SlideLayout(name)]; dup {
				return decls, fmt.Errorf("%s: the background of layout %q appears in two declarations — a layout has one background", where, name)
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
