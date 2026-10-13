// Package core는 덱 설정, 외관 해석, 검증을 담는다 — 타깃 렌더러가 소비할 수 있게
// 덱 모델을 완성시키는 층이다. 렌더러를 임포트하지 않으므로 파이프라인 조립은
// internal/pipeline이 맡는다.
package core

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"reflect"

	"golang.org/x/text/language"

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
	// Mode는 PPTX 렌더 모드다. 구현된 모드는 "native"(편집 가능한 도형) 하나뿐이다.
	Mode PPTXMode `json:"mode"`
}

// PPTXMode는 PPTX 렌더 모드다.
type PPTXMode string

const (
	// PPTXNative는 네이티브 도형으로 조립한다 — 제품의 기본값.
	PPTXNative PPTXMode = "native"
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
	cfg, err := ReadConfig(projectDir)
	if err != nil {
		return Config{}, err
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

// ReadConfig는 LoadConfigWithThemes에서 테마 이름 검사만 뺀 것이다 — 구조와 닫힌 어휘는 똑같이
// 검사하고 기본값을 채운다. 테마를 다른 출처(CLI 플래그)가 덮는 호출자가 쓴다: 버려질 이름을
// 검사하면 쓰지도 않을 설정의 오타나 이름 겹침이 다른 테마를 명시한 실행을 멈춘다. 이 함수를
// 쓰는 호출자는 실제로 쓸 이름을 CheckThemeName으로 직접 검사한다.
func ReadConfig(projectDir string) (Config, error) {
	cfg := DefaultConfig()
	cfg.Title = filepath.Base(projectDir)

	data, err := os.ReadFile(filepath.Join(projectDir, ConfigFile))
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return cfg, nil
	case err != nil:
		return Config{}, fmt.Errorf("cannot read %s: %w", ConfigFile, err)
	}

	// BOM은 키 검사보다 먼저 뗀다 — 진단의 줄:칸이 BOM을 보여 주지 않는 에디터와 맞아야 한다.
	data = trimBOM(data)
	if err := checkConfigRoot(data); err != nil {
		return Config{}, err
	}
	// 키 검사를 DisallowUnknownFields에 맡기지 않는 이유: 그 에러는 타입이 없고 자리도 말하지 않는다.
	// 스키마를 따라 걷는 검사는 틀린 키의 줄:칸과 경로(`targets.pptx.mdoe`)를 낸다.
	if err := checkJSONKeys(data, 0, reflect.TypeOf(cfg), "", data, false); err != nil {
		var fe *themeFileError
		if errors.As(err, &fe) {
			fe.file = ConfigFile
			return Config{}, fe
		}
		return Config{}, &themeFileError{file: ConfigFile, msg: err.Error()}
	}
	// 파일에 있는 필드만 덮어쓰도록 기본값이 담긴 cfg에 그대로 언마샬한다.
	dec := json.NewDecoder(bytes.NewReader(data))
	if err := dec.Decode(&cfg); err != nil {
		return Config{}, jsonDecodeError(data, ConfigFile, err)
	}
	// Decoder는 값 하나만 읽고 멈춘다 — 뒤에 남은 것을 보지 않으면 붙여 넣다 남은 조각
	// (`{…} {…}`)이 조용히 버려지고, 저자는 두 번째 객체의 값이 먹었다고 믿는다.
	end := int(dec.InputOffset())
	if rest := bytes.TrimLeft(data[end:], " \t\r\n"); len(rest) > 0 {
		return Config{}, &themeFileError{file: ConfigFile, pos: offsetPos(data, len(data)-len(rest)),
			msg: "unexpected content after the top-level object"}
	}
	if err := cfg.Validate(); err != nil {
		return Config{}, fmt.Errorf("%s: %w", ConfigFile, err)
	}
	return cfg, nil
}

// checkConfigRoot는 최상위 값이 객체인지 먼저 본다. 여는 `{`가 빠진 파일(`"title": "x", …}`)은
// 첫 키 문자열이 그대로 완결된 값으로 읽혀서, 이대로 디코딩하면 문법 오류 대신 "최상위가 객체가
// 아니다"만 남는다 — 저자가 고칠 자리는 그 뒤의 `:`다. 그래서 최상위가 객체가 아니면 파일 전체를
// 한 값으로 읽어 문법 오류를 먼저 말하고, 문법이 멀쩡할 때만(`"x"`, `[]`) 타입을 말한다.
func checkConfigRoot(data []byte) error {
	start := len(data) - len(bytes.TrimLeft(data, " \t\r\n"))
	if start == len(data) || data[start] == '{' {
		return nil
	}
	if err := json.Unmarshal(data, new(json.RawMessage)); err != nil {
		return jsonDecodeError(data, ConfigFile, err)
	}
	return &themeFileError{file: ConfigFile, pos: offsetPos(data, start),
		msg: "the top-level value must be an object, not " + jsonValueWord(data[start])}
}

// jsonValueWord는 JSON 값의 첫 바이트로 그 종류를 저자의 말로 이름 짓는다.
func jsonValueWord(b byte) string {
	switch b {
	case '"':
		return "string"
	case '[':
		return "array"
	case 't', 'f':
		return "true or false"
	case 'n':
		return "null"
	}
	return "number"
}

// CanonicalLang은 덱 언어를 BCP 47 언어 태그로 읽고 정규형을 돌려준다 ("EN-us" → "en-US",
// "kor" → "ko").
//
// 판정하는 이유는 이 값이 `<html lang>`에 그대로 실려 언어별 규칙을 고른다는 것이다 — 한국어의
// 어절 줄바꿈은 `:lang(ko)`에 걸려 있어서, "korean" 같은 이름은 빌드가 조용히 통과한 채 그
// 규칙만 꺼 버린다. 정규형으로 바꾸는 이유도 같다: "kor"·"ko_KR"도 뜻은 한국어인데 선택자는
// 정규형 태그에만 맞는다.
func CanonicalLang(s string) (string, error) {
	tag, err := language.Parse(s)
	if err != nil {
		return "", fmt.Errorf("not a BCP 47 language tag: %q (for example \"en\", \"ko\", \"ja\")", s)
	}
	return tag.String(), nil
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
	// 슬라이드는 프로젝트 파일시스템(os.DirFS)으로 읽으므로 slidesDir은 그 안의 경로여야
	// 한다. 여기서 막지 않으면 `../`나 절대 경로가 fs 층의 날것 에러("invalid argument")로
	// 터져 저자가 원인을 알 수 없다. `./talk/` 같은 표기는 정규화해서 받는다.
	slides := path.Clean(filepath.ToSlash(c.SlidesDir))
	if filepath.IsAbs(c.SlidesDir) || !fs.ValidPath(slides) {
		return fmt.Errorf("slidesDir: %q must be a relative path inside the project", c.SlidesDir)
	}
	c.SlidesDir = slides
	if c.OutDir == "" {
		c.OutDir = def.OutDir
	}
	if c.Targets.PPTX.Mode == "" {
		c.Targets.PPTX.Mode = def.Targets.PPTX.Mode
	}

	if _, err := c.Aspect.Stage(); err != nil {
		return fmt.Errorf("aspect: %w", err)
	}
	lang, err := CanonicalLang(c.Lang)
	if err != nil {
		return fmt.Errorf("lang: %w", err)
	}
	c.Lang = lang
	// 테마 **이름의 어휘**는 여기서 보지 않는다: 쓸 수 있는 이름이 내장 ∪
	// 프로젝트 로컬이므로 판정에 프로젝트 파일시스템이 필요하고, 이 함수는 그것을 모른다.
	// 판정하는 자리는 둘이다 — 프로젝트를 아는 LoadConfig와 값 층의 Resolve. 폴백하지 않는
	// 이유는 모르는 variant와 갈라지는 자리다: variant 어휘는 테마의 사유물이지만 테마 이름의
	// 어휘는 프로젝트가 소유하므로, 목록 밖 이름은 오타이거나 아직 만들지 않은 테마이고 둘 다
	// 저자가 즉시 고칠 수 있다.
	// 구현되지 않은 모드는 받지 않는다 — 받아 주면 네이티브와 같은 산출물이 조용히 나온다.
	if c.Targets.PPTX.Mode != PPTXNative {
		return fmt.Errorf("targets.pptx.mode: unknown mode %q (%q is the only mode)",
			string(c.Targets.PPTX.Mode), PPTXNative)
	}
	return nil
}
