// Package core는 덱 설정, 외관 해석, 검증을 담는다 — 타깃 렌더러가 소비할 수 있게
// 덱 모델을 완성시키는 층이다. 렌더러를 임포트하지 않으므로 파이프라인 조립은
// internal/pipeline이 맡는다.
package core

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/k2sebeom/balpyo/soml"
)

// ConfigFile은 덱 설정 파일 이름이다.
const ConfigFile = "balpyo.json"

// DefaultTheme는 테마를 말하지 않은 덱이 받는 내장 테마다.
//
// 상수로 두는 이유는 기본값을 아는 자리가 셋이라는 것이다: 설정 기본값(DefaultConfig), 값
// 층(Resolve — ThemeName이 빈 덱), `balpyo init`의 스캐폴드. 세 곳이 각자 문자열을 적으면
// 기본 테마를 갈 때 한 곳이 남는다.
const DefaultTheme = "minimal"

// Config는 덱 설정이다 (`balpyo.json`). 우선순위는 CLI 플래그 > balpyo.json > 기본값이며,
// 플래그 적용은 CLI가 이 구조체를 덮어써서 수행한다.
//
// 알 수 없는 필드는 에러다 — 오타가 조용히 무시되는 대신 빌드를 멈춘다 (닫힌 어휘 원칙).
type Config struct {
	Title     string      `json:"title"`
	Author    string      `json:"author"`
	Theme     string      `json:"theme"`
	Aspect    soml.Aspect `json:"aspect"`
	Lang      string      `json:"lang"`
	SlidesDir string      `json:"slidesDir"`
	OutDir    string      `json:"outDir"`
	Targets   Targets     `json:"targets"`
	// Footer는 덱 크롬의 푸터 한 줄이다. 빈 문자열이면 꺼져 있다.
	//
	// 크롬의 **내용과 켜짐만** 여기 있다 — 자리·정렬·역할·제외 규칙은 테마의 `chrome`
	// 선언이 소유한다. 기본값이 꺼짐인 이유는 크롬이 덱마다의 선택이라는 것이다(내부 공유
	// 덱과 공개 발표 덱이 다르다).
	Footer string `json:"footer"`
	// PageNumber는 페이지 번호를 켤지다. PPTX에서 `a:fld type="slidenum"`이 되므로 뷰어에서
	// 장을 더하면 번호가 따라온다.
	PageNumber bool `json:"pageNumber"`
}

// Targets는 타깃별 옵션이다.
type Targets struct {
	PPTX PPTXTarget `json:"pptx"`
}

// PPTXTarget은 PPTX 타깃 옵션이다.
type PPTXTarget struct {
	// Mode는 "native"(편집 가능한 도형) 또는 "image"(충실도 100%·편집 불가 폴백)다.
	Mode PPTXMode `json:"mode"`
}

// PPTXMode는 PPTX 렌더 모드다.
type PPTXMode string

const (
	// PPTXNative는 네이티브 도형으로 조립한다 — 제품의 기본값.
	PPTXNative PPTXMode = "native"
	// PPTXImage는 슬라이드를 그림 한 장으로 굽는 폴백이다. Phase 2에서 구현한다.
	PPTXImage PPTXMode = "image"
)

// DefaultConfig는 balpyo.json이 없거나 필드가 비었을 때 쓰는 기본값이다.
func DefaultConfig() Config {
	return Config{
		Theme:     DefaultTheme,
		Aspect:    soml.DefaultAspect,
		Lang:      "en",
		SlidesDir: "slides",
		OutDir:    "dist",
		Targets:   Targets{PPTX: PPTXTarget{Mode: PPTXNative}},
	}
}

// LoadConfig는 프로젝트 디렉토리에서 설정을 읽는다.
//
// balpyo.json은 **선택**이다 — 없으면 기본값을 쓰고 제목을 디렉토리 이름에서 유도한다.
// `slides/`에 파일 하나만 두고도 빌드가 되어야 첫 결과까지의 거리가 짧다.
func LoadConfig(projectDir string) (Config, error) {
	return LoadConfigWithThemes(projectDir, nil)
}

// LoadConfigWithThemes는 LoadConfig와 같고, 테마 이름을 검사할 때 home(설치한 테마의 홈 —
// `themes/<이름>/`을 담은 디렉토리)도 어휘에 넣는다. home이 nil이면 LoadConfig다.
func LoadConfigWithThemes(projectDir string, home fs.FS) (Config, error) {
	cfg := DefaultConfig()
	cfg.Title = filepath.Base(projectDir)

	data, err := os.ReadFile(filepath.Join(projectDir, ConfigFile))
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return cfg, nil
	case err != nil:
		return Config{}, fmt.Errorf("cannot read %s: %w", ConfigFile, err)
	}

	// 파일에 있는 필드만 덮어쓰도록 기본값이 담긴 cfg에 그대로 언마샬한다.
	dec := json.NewDecoder(strings.NewReader(string(trimBOM(data))))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&cfg); err != nil {
		return Config{}, fmt.Errorf("cannot parse %s: %w", ConfigFile, err)
	}
	if err := cfg.Validate(); err != nil {
		return Config{}, fmt.Errorf("%s: %w", ConfigFile, err)
	}
	// 테마 이름은 **프로젝트의 어휘**와 대조한다: 내장 ∪ `themes/<이름>/` ∪ 설치한 테마.
	// Validate가 아니라 여기인 이유는 어휘가 파일시스템에 달렸다는 것이다 — 설정 구조체만으로는
	// 로컬 테마를 알 수 없고, 프로젝트를 아는 자리는 이 함수다. 값 층이 같은 판정을 한 번 더
	// 하는 이유는 설정 없이 Deck을 만드는 길이 있다는 것이다 (Resolve).
	if err := CheckThemeName(WithInstalledThemes(os.DirFS(projectDir), home), cfg.Theme); err != nil {
		return Config{}, fmt.Errorf("%s: theme: %w", ConfigFile, err)
	}
	return cfg, nil
}

// Validate는 설정 값이 닫힌 어휘 안에 있는지 검사하고, 빈 필드를 기본값으로 채운다.
func (c *Config) Validate() error {
	def := DefaultConfig()
	if c.Theme == "" {
		c.Theme = def.Theme
	}
	if c.Aspect == "" {
		c.Aspect = def.Aspect
	}
	if c.Lang == "" {
		c.Lang = def.Lang
	}
	if c.SlidesDir == "" {
		c.SlidesDir = def.SlidesDir
	}
	if c.OutDir == "" {
		c.OutDir = def.OutDir
	}
	if c.Targets.PPTX.Mode == "" {
		c.Targets.PPTX.Mode = def.Targets.PPTX.Mode
	}

	if _, err := c.Aspect.Stage(); err != nil {
		return fmt.Errorf("aspect: %w", err)
	}
	// 테마 **이름의 어휘**는 여기서 보지 않는다: 쓸 수 있는 이름이 내장 ∪
	// 프로젝트 로컬이므로 판정에 프로젝트 파일시스템이 필요하고, 이 함수는 그것을 모른다.
	// 판정하는 자리는 둘이다 — 프로젝트를 아는 LoadConfig와 값 층의 Resolve. 폴백하지 않는
	// 이유는 모르는 variant와 갈라지는 자리다: variant 어휘는 테마의 사유물이지만 테마 이름의
	// 어휘는 프로젝트가 소유하므로, 목록 밖 이름은 오타이거나 아직 만들지 않은 테마이고 둘 다
	// 저자가 즉시 고칠 수 있다.
	switch c.Targets.PPTX.Mode {
	case PPTXNative, PPTXImage:
	default:
		return fmt.Errorf("targets.pptx.mode: unknown mode %q (%q or %q)",
			string(c.Targets.PPTX.Mode), PPTXNative, PPTXImage)
	}
	return nil
}
