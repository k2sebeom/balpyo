// Package scaffold는 `balpyo init`이 쓰는 프로젝트 하나를 소유한다.
//
// 왜 예제를 복사하지 않는가: `examples/decks/*`는 회귀 테스트다 (도그푸딩 덱 — 12·10·6장에 마찰
// 기록 주석까지 들어 있다). 스캐폴딩은 반대로 **작고 지루해야** 하고, 완료 조건이 "파일이
// 생긴다"가 아니라 **"그 덱이 경고 0으로 빌드된다"**이므로 따로 소유하고 따로 판정한다
// (scaffold_test.go · cmd/balpyo/init_test.go).
//
// 템플릿은 `//go:embed`로 바이너리에 들어간다 — `balpyo init`은 네트워크도 저장소 체크아웃도
// 요구하지 않는다 (CLI의 "바이너리 하나" 계약이 스캐폴딩에도 적용된다).
package scaffold

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"text/template"

	"github.com/k2sebeom/balpyo/core"
	"github.com/k2sebeom/balpyo/soml"
)

// files는 스캐폴딩이 쓰는 파일들이다. `balpyo.json`은 여기 없다 — 설정은 구조체에서
// 마샬해서 쓴다(configFile): 키 순서가 타입으로 고정되고, 값 이스케이프를 템플릿 문법으로
// 다시 발명하지 않는다.
//
//go:embed template
var files embed.FS

// templateDir은 embed 트리의 뿌리다.
const templateDir = "template"

// GitignoreFile은 스캐폴딩이 쓰는 무시 목록이다. embed는 `.`으로 시작하는 파일을 담지
// 못하므로 템플릿에서는 `gitignore`라는 이름으로 산다.
const GitignoreFile = ".gitignore"

// DefaultLang은 스캐폴딩 덱의 언어다. 템플릿 슬라이드의 글이 영어이므로 core의 기본값("en")과
// 같다 — 언어가 어긋나면 뷰어의 줄바꿈·하이픈 규칙이 글과 맞지 않는다.
const DefaultLang = "en"

// Options는 스캐폴딩 입력이다. 빈 값은 기본값으로 채운다.
type Options struct {
	// Dir은 만들 프로젝트 루트다. 없으면 만든다.
	Dir string
	// Title은 덱 제목이다. 비면 디렉토리 이름을 쓴다 (core.LoadConfig의 규칙과 같다).
	Title string
	// Author는 발표자다. 비면 템플릿의 자리표시자가 남는다 — 빈 텍스트를 슬라이드에 두면
	// 저자가 고칠 자리를 보지 못한다.
	Author string
	// Theme는 내장 테마 이름이다. 비면 core의 기본값(minimal)이다.
	Theme string
	// Aspect는 종횡비다. 비면 soml.DefaultAspect다.
	Aspect soml.Aspect
	// Lang은 덱 언어다. 비면 DefaultLang이다.
	Lang string
}

// Result는 스캐폴딩 산출이다.
type Result struct {
	// Dir은 만든 프로젝트 루트다 (절대 경로).
	Dir string
	// Written은 Dir 기준 상대 경로 목록이다 (쓴 순서, 슬래시 구분).
	Written []string
}

// configFile은 스캐폴딩이 쓰는 balpyo.json이다.
//
// core.Config를 그대로 마샬하지 않는 이유: 그 구조체에는 아직 뜻이 없는 필드까지 들어 있고
// (targets·footer·pageNumber), 처음 여는 설정 파일에 꺼진 기능의 키가 있으면 저자가 그것을
// 먼저 읽는다. 여기 있는 여섯 키가 "고쳐도 되는 것"의 목록이다. 키 이름이 틀리면
// core.LoadConfig이 알 수 없는 필드로 멈추므로, 그 왕복을 테스트가 단정한다.
type configFile struct {
	Title  string      `json:"title"`
	Author string      `json:"author"`
	Theme  string      `json:"theme"`
	Lang   string      `json:"lang"`
	Aspect soml.Aspect `json:"aspect"`
	OutDir string      `json:"outDir"`
}

// slideVars는 슬라이드 템플릿이 받는 값이다.
type slideVars struct {
	Title  string
	Author string
}

// Write는 프로젝트를 만든다.
//
// 기존 파일은 **덮어쓰지 않는다**: balpyo.json이 있거나 슬라이드 디렉토리에
// `.slide`가 있으면 아무것도 쓰지 않고 멈춘다. 되돌릴 수단이 저자의 git뿐인데 init은 git이
// 있다고 가정할 수 없다.
func Write(opts Options) (*Result, error) {
	dir := opts.Dir
	if dir == "" {
		dir = "."
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		return nil, fmt.Errorf("cannot resolve the project path: %w", err)
	}

	cfg := core.DefaultConfig()
	file := configFile{
		Title:  opts.Title,
		Author: opts.Author,
		Theme:  opts.Theme,
		Lang:   opts.Lang,
		Aspect: opts.Aspect,
		OutDir: cfg.OutDir,
	}
	if file.Title == "" {
		file.Title = filepath.Base(abs)
	}
	if file.Theme == "" {
		file.Theme = cfg.Theme
	}
	if file.Lang == "" {
		file.Lang = DefaultLang
	}
	if file.Aspect == "" {
		file.Aspect = soml.DefaultAspect
	}
	// 테마 이름은 **내장 목록**에 대조한다. 스캐폴딩이 만드는 프로젝트에는 로컬 테마(themes/)가
	// 없으므로, 내장 테마가 아닌 이름을 써 주면 첫 빌드가 그 테마를 찾지 못하고 실패한다.
	if !core.IsBuiltinTheme(file.Theme) {
		return nil, fmt.Errorf("not a built-in theme: %q (available themes: %s)",
			file.Theme, strings.Join(core.BuiltinThemes(), ", "))
	}
	// 종횡비는 스테이지를 만들 수 있어야 한다 — 빌드가 첫 명령에서 실패하지 않는 것이
	// 스캐폴딩의 계약이므로 여기서 판정한다 (core.Config.Validate와 같은 검사).
	if _, err := file.Aspect.Stage(); err != nil {
		return nil, fmt.Errorf("aspect: %w", err)
	}

	// 언어도 core.Config.Validate와 같은 판정이다 — 빌드가 거부할 값을 balpyo.json에 쓰지
	// 않고, 받아들인 값은 정규형으로 쓴다.
	lang, err := core.CanonicalLang(file.Lang)
	if err != nil {
		return nil, fmt.Errorf("lang: %w", err)
	}
	file.Lang = lang

	planned, err := plan(file)
	if err != nil {
		return nil, err
	}
	if err := checkEmpty(abs, cfg.SlidesDir); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(abs, 0o755); err != nil {
		return nil, fmt.Errorf("cannot create %s: %w", abs, err)
	}

	result := &Result{Dir: abs}
	for _, f := range planned {
		dest := filepath.Join(abs, filepath.FromSlash(f.name))
		// .gitignore는 **있으면 손대지 않는다** — 저자가 쓴 목록을 스캐폴딩이 지우지 않는다.
		if f.name == GitignoreFile {
			if _, err := os.Stat(dest); err == nil {
				continue
			}
		}
		if parent := filepath.Dir(dest); parent != abs {
			if err := os.MkdirAll(parent, 0o755); err != nil {
				return nil, fmt.Errorf("cannot create %s: %w", parent, err)
			}
		}
		if err := os.WriteFile(dest, f.data, 0o644); err != nil {
			return nil, fmt.Errorf("cannot write %s: %w", dest, err)
		}
		result.Written = append(result.Written, f.name)
	}
	return result, nil
}

// plannedFile은 쓸 파일 하나다 (프로젝트 기준 상대 경로 + 내용).
type plannedFile struct {
	name string
	data []byte
}

// plan은 쓸 파일 전부를 메모리에서 만든다 — 디스크를 건드리기 전에 템플릿 실패가 드러나야
// 반쯤 만들어진 프로젝트가 남지 않는다.
func plan(file configFile) ([]plannedFile, error) {
	data, err := json.MarshalIndent(file, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("cannot create %s: %w", core.ConfigFile, err)
	}
	out := []plannedFile{{name: core.ConfigFile, data: append(data, '\n')}}

	vars := slideVars{Title: file.Title, Author: file.Author}
	if vars.Author == "" {
		vars.Author = "Presenter"
	}
	slides, err := renderSlides(vars)
	if err != nil {
		return nil, err
	}
	out = append(out, slides...)

	ignore, err := files.ReadFile(path.Join(templateDir, "gitignore"))
	if err != nil {
		return nil, fmt.Errorf("cannot read the scaffold template: %w", err)
	}
	return append(out, plannedFile{name: GitignoreFile, data: ignore}), nil
}

// renderSlides는 템플릿 슬라이드를 치환해 돌려준다 (파일 이름 순 — 그것이 곧 덱 순서다).
func renderSlides(vars slideVars) ([]plannedFile, error) {
	dir := path.Join(templateDir, "slides")
	entries, err := fs.ReadDir(files, dir)
	if err != nil {
		return nil, fmt.Errorf("cannot read the scaffold template: %w", err)
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), soml.Ext) {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)

	out := make([]plannedFile, 0, len(names))
	for _, name := range names {
		raw, err := files.ReadFile(path.Join(dir, name))
		if err != nil {
			return nil, fmt.Errorf("cannot read the scaffold template: %w", err)
		}
		tmpl, err := template.New(name).Option("missingkey=error").Parse(string(raw))
		if err != nil {
			return nil, fmt.Errorf("parsing scaffold template %s failed: %w", name, err)
		}
		var buf bytes.Buffer
		if err := tmpl.Execute(&buf, vars); err != nil {
			return nil, fmt.Errorf("executing scaffold template %s failed: %w", name, err)
		}
		// 슬라이드는 core.DefaultConfig().SlidesDir 아래에 간다 — 설정의 slidesDir을
		// 스캐폴딩이 바꾸지 않으므로 기본값 하나가 두 자리(경로와 설정)를 함께 정한다.
		out = append(out, plannedFile{
			name: path.Join(core.DefaultConfig().SlidesDir, name),
			data: buf.Bytes(),
		})
	}
	return out, nil
}

// checkEmpty는 이 디렉토리가 아직 balpyo 프로젝트가 아님을 확인한다.
func checkEmpty(dir, slidesDir string) error {
	if _, err := os.Stat(filepath.Join(dir, core.ConfigFile)); err == nil {
		return fmt.Errorf("already a balpyo project: %s contains %s", dir, core.ConfigFile)
	}
	entries, err := os.ReadDir(filepath.Join(dir, slidesDir))
	if err != nil {
		// 디렉토리가 없는 것이 정상이다. 읽을 수 없는 다른 이유는 쓰기에서 다시 드러난다.
		return nil
	}
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), soml.Ext) {
			return fmt.Errorf("slides already exist: %s contains %s",
				filepath.Join(dir, slidesDir), e.Name())
		}
	}
	return nil
}
