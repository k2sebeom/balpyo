package core

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"path"

	"github.com/k2sebeom/balpyo/soml"
)

// 이 파일은 **테마 패키지 포맷**(PRD §8.2)의 로더다 — Iteration 10이 core/theme.go의 하드코딩
// 표를 데이터로 해체한 자리다. 순서를 이렇게 잡은 이유는 theme.go의 패키지 주석에 있다:
// 하드코딩 표가 두 렌더러의 실제 요구를 드러낸 다음에야 스키마를 파일로 고정한다.
//
// 포맷은 **데이터 전용**이다 (JSON + CSS + 에셋, JS 없음 — PRD §8.2·§13-11): `theme get`으로
// 받은 원격 테마가 임의 코드 실행 경로가 되어서는 안 된다. 로더는 fs.FS 위에서 도므로 내장
// 테마(go:embed)와 프로젝트 로컬 테마(os.DirFS)가 같은 코드를 지난다.
//
// **byte 동일성 계약**: 내장 minimal을 이 로더로 읽은 soml.Theme는 옛 하드코딩
// minimalTheme()과 값·순서가 정확히 같아야 한다. 그래야 기존 골든(themeCSS·clrScheme·픽스처)이
// 움직이지 않고, 해체가 동작을 바꾸지 않았음이 증명된다. 순서를 지키려고 tokens.json은 객체가
// 아니라 **순서 있는 배열**을 쓴다 (JSON 객체 키 순서에 기대지 않는다 — PRD §8.4의 객체
// 스케치와 갈라지는 자리이고 ADR이 근거를 남긴다).

// builtinThemes는 바이너리에 내장된 공식 테마다 (PRD §8.6). v1은 minimal 하나이며 기본값이자
// 폴백 기준이다 — `balpyo init` 직후 네트워크·설치 없이 예쁜 결과가 나오는 것이 첫인상을
// 지배하므로 기본 경로에서 네트워크를 제거한다.
//
//go:embed themes
var builtinThemes embed.FS

// themeManifest는 theme.json의 파싱 결과다 (PRD §8.2).
type themeManifest struct {
	Name string `json:"name"`
	// Extends는 상속할 테마 이름이다 (PRD §8.5) — v1 minimal에서는 비어 있다. 상속 병합은
	// 아직 구현하지 않으므로 비지 않은 값은 로더가 진단한다 (미구현을 조용히 무시하지 않는다).
	Extends string `json:"extends"`
	// Layouts는 이 테마가 구현하는 표준 레이아웃 이름 목록이다. 표준 카탈로그 중 여기 없는
	// 이름은 minimal 구현으로 폴백하고 경고한다 (core.slideLayout, 10/03). 테마 자체
	// 레이아웃(`corp-hero`)도 여기 실려 어휘가 된다.
	Layouts []string `json:"layouts"`
	// Chrome은 전 슬라이드 장식(로고·푸터) 선언이다 (PRD §8.1·§8.4) — v1 minimal은 비어 있다.
	Chrome []chromeDecl `json:"chrome"`
}

// chromeDecl은 크롬 요소 하나의 선언이다 (10/04·10/05가 소비한다). v1 minimal은 크롬이 없어
// 필드가 최소이며, 크롬 어휘가 실제로 필요한 iteration이 채운다.
type chromeDecl struct {
	Kind string `json:"kind"`
}

// themePackage는 로드된 테마 하나다 — 토큰 표(soml.Theme)와 매니페스트를 함께 담는다.
// 컴포넌트 외관(components.json)과 레이아웃 CSS(layouts/*.css)는 뒤 하위 단계가 이 구조체에
// 필드를 더한다.
type themePackage struct {
	manifest themeManifest
	tokens   soml.Theme
}

// ── tokens.json 스키마 ──────────────────────────────────────────────────────
// 필드는 소문자 JSON 키에 태그로 붙는다. 배열이라 순서가 파일 그대로 유지된다.

type tokensFile struct {
	Color  []colorTok  `json:"color"`
	Font   []fontTok   `json:"font"`
	Space  []spaceTok  `json:"space"`
	Radius []radiusTok `json:"radius"`
	Shadow []shadowTok `json:"shadow"`
}

type colorTok struct {
	Name string `json:"name"`
	RGB  string `json:"rgb"`
	Slot string `json:"slot"`
	Ref  string `json:"ref"`
}

type fontTok struct {
	Name    string `json:"name"`
	Latin   string `json:"latin"`
	EA      string `json:"ea"`
	Generic string `json:"generic"`
	Scheme  string `json:"scheme"`
}

type spaceTok struct {
	Name string  `json:"name"`
	Px   float64 `json:"px"`
}

type radiusTok struct {
	Name string  `json:"name"`
	Px   float64 `json:"px"`
}

type shadowTok struct {
	Name  string  `json:"name"`
	Dist  float64 `json:"dist"`
	Dir   float64 `json:"dir"`
	Blur  float64 `json:"blur"`
	Color string  `json:"color"`
	Alpha float64 `json:"alpha"`
}

// loadTheme는 fsys의 dir/ 아래 테마 패키지를 읽는다. dir은 fs 경로다 (내장은
// "themes/minimal", 로컬은 프로젝트 상대 경로). 이름은 매니페스트가 소유한다.
func loadTheme(fsys fs.FS, dir string) (themePackage, error) {
	var pkg themePackage

	manifestBytes, err := fs.ReadFile(fsys, path.Join(dir, "theme.json"))
	if err != nil {
		return pkg, fmt.Errorf("theme.json을 읽을 수 없다: %w", err)
	}
	if err := json.Unmarshal(manifestBytes, &pkg.manifest); err != nil {
		return pkg, fmt.Errorf("theme.json 파싱 실패: %w", err)
	}
	if pkg.manifest.Name == "" {
		return pkg, fmt.Errorf("theme.json에 name이 없다")
	}
	if pkg.manifest.Extends != "" {
		// 상속 병합은 아직 구현하지 않았다 — 조용히 무시하면 저자가 오버라이드가 먹었다고
		// 착각한다 (닫힌 어휘가 막으려던 조용한 실패의 변종).
		return pkg, fmt.Errorf("theme.json의 extends는 아직 지원하지 않는다 (%q)", pkg.manifest.Extends)
	}

	tokens, err := loadTokens(fsys, dir, pkg.manifest.Name)
	if err != nil {
		return pkg, err
	}
	pkg.tokens = tokens
	return pkg, nil
}

// loadTokens는 tokens.json을 soml.Theme로 읽는다. 배열 순서를 그대로 옮겨 byte 동일성을
// 지킨다.
func loadTokens(fsys fs.FS, dir, name string) (soml.Theme, error) {
	var theme soml.Theme
	raw, err := fs.ReadFile(fsys, path.Join(dir, "tokens.json"))
	if err != nil {
		return theme, fmt.Errorf("tokens.json을 읽을 수 없다: %w", err)
	}
	var file tokensFile
	if err := json.Unmarshal(raw, &file); err != nil {
		return theme, fmt.Errorf("tokens.json 파싱 실패: %w", err)
	}

	theme.Name = name
	for _, c := range file.Color {
		theme.Colors = append(theme.Colors, soml.ColorToken{Name: c.Name, RGB: c.RGB, Slot: c.Slot, Ref: c.Ref})
	}
	for _, f := range file.Font {
		theme.Fonts = append(theme.Fonts, soml.FontToken{
			Name: soml.FontRole(f.Name), Latin: f.Latin, EastAsian: f.EA, Generic: f.Generic, Scheme: f.Scheme,
		})
	}
	for _, s := range file.Space {
		theme.Spaces = append(theme.Spaces, soml.SpaceToken{Name: s.Name, Px: s.Px})
	}
	for _, r := range file.Radius {
		theme.Radii = append(theme.Radii, soml.RadiusToken{Name: r.Name, Px: r.Px})
	}
	for _, s := range file.Shadow {
		// color는 토큰 참조다 (표가 리터럴을 쓰면 PPTX에서 schemeClr 참조가 끊긴다 —
		// theme.go의 옛 minimalShadows 근거). soml.TokenColor로 감싼다.
		theme.Shadows = append(theme.Shadows, soml.ShadowToken{
			Name: s.Name, Dist: s.Dist, Dir: s.Dir, Blur: s.Blur,
			Color: soml.TokenColor(s.Color), Alpha: s.Alpha,
		})
	}
	return theme, nil
}

// loadBuiltinTheme는 내장 테마를 이름으로 읽는다. 내장 파일은 빌드 시점에 고정되므로 로드
// 실패는 프로그래머 오류다 (regexp.MustCompile과 같은 자리) — 호출자(minimalTheme)가 에러를
// 다룰 자리가 없고, TestLoadBuiltinMinimal이 이 불변을 못 박는다.
func loadBuiltinTheme(name string) themePackage {
	pkg, err := loadTheme(builtinThemes, path.Join("themes", name))
	if err != nil {
		panic(fmt.Sprintf("내장 테마 %q 로드 실패: %v", name, err))
	}
	return pkg
}
