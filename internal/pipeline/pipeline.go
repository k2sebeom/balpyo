// Package pipeline은 수집부터 산출물 쓰기까지의 빌드 흐름을 조립한다.
//
// 왜 별도 패키지인가: core는 렌더러를 임포트하지 않고(해석·검증 전용), 렌더러는 core를
// 임포트하지 않는다(덱 모델만 본다). 둘을 잇는 자리가 필요하고, 그 자리는 cmd/balpyo가
// 될 수 없다 — main 패키지는 임포트할 수 없어서 테스트 하네스가 같은 흐름을 돌릴 수
// 없기 때문이다. 그래서 internal에 둔다: 공개 API 약속 없이 CLI와 테스트가 공유한다.
package pipeline

import (
	"fmt"
	"os"
	"path"
	"path/filepath"

	"github.com/k2sebeom/balpyo/core"
	"github.com/k2sebeom/balpyo/render"
	"github.com/k2sebeom/balpyo/render/pptx"
	"github.com/k2sebeom/balpyo/render/web"
	"github.com/k2sebeom/balpyo/soml"
)

// Options는 빌드 입력이다. 값은 이미 CLI 플래그 > balpyo.json > 기본값 순으로 정해진
// 상태로 들어온다.
type Options struct {
	// ProjectDir는 프로젝트 루트다 (balpyo.json과 slides/가 있는 곳).
	ProjectDir string
	// Targets가 비면 render.DefaultTargets()를 쓴다.
	Targets []render.Target
	// OutDir이 비면 설정의 outDir을 쓴다. 상대 경로는 ProjectDir 기준이다.
	OutDir string
	// Strict는 경고를 에러로 승격한다.
	Strict bool
}

// Result는 빌드 산출이다.
type Result struct {
	Deck *soml.Deck
	// Diagnostics는 에러가 없어 빌드가 성공한 경우에도 남는 경고를 담는다.
	Diagnostics soml.Diagnostics
	// OutDir은 산출물을 쓴 디렉토리다 (절대 경로).
	OutDir string
	// Written은 OutDir 기준 상대 경로 목록이다 (타깃 순서, 타깃 안에서는 경로 정렬순).
	Written []string
}

// Renderers는 타깃 이름에서 렌더러를 찾는다. 새 타깃을 붙이는 지점이다.
func Renderers() map[render.Target]render.Renderer {
	return map[render.Target]render.Renderer{
		render.TargetHTML: web.New(),
		render.TargetPPTX: pptx.New(),
	}
}

// LoadDeck은 프로젝트를 읽어 렌더 가능한 덱 모델을 만든다 — 디스크에 쓰지 않는다.
//
// 흐름: 설정 읽기 → 슬라이드 수집·정렬 → 파싱 → 외관 해석 → 검증.
func LoadDeck(opts Options) (*soml.Deck, soml.Diagnostics, error) {
	projectDir := opts.ProjectDir
	if projectDir == "" {
		projectDir = "."
	}
	cfg, err := core.LoadConfig(projectDir)
	if err != nil {
		return nil, nil, err
	}
	stage, err := cfg.Aspect.Stage()
	if err != nil {
		return nil, nil, err
	}

	files, err := soml.Collect(os.DirFS(projectDir), cfg.SlidesDir)
	if err != nil {
		return nil, nil, err
	}
	if len(files) == 0 {
		return nil, nil, fmt.Errorf("슬라이드가 없다: %s/ 아래에 %s 파일이 필요하다", cfg.SlidesDir, soml.Ext)
	}

	deck := &soml.Deck{
		Title:  cfg.Title,
		Author: cfg.Author,
		Lang:   cfg.Lang,
		Aspect: cfg.Aspect,
		Stage:  stage,
		Slides: make([]*soml.Slide, 0, len(files)),
	}
	var diags soml.Diagnostics
	for _, file := range files {
		source := path.Join(cfg.SlidesDir, file.Path)
		src, err := os.ReadFile(filepath.Join(projectDir, filepath.FromSlash(source)))
		if err != nil {
			return nil, nil, fmt.Errorf("%s를 읽을 수 없다: %w", source, err)
		}
		// Iteration 1에서 soml.Parse로 바뀐다. 지금은 토큰만 뜨고 빈 슬라이드를 만든다 —
		// 파이프라인이 끝에서 끝까지 도는지 확인하는 것이 Iteration 0의 목표다.
		deck.Slides = append(deck.Slides, &soml.Slide{
			Index:   len(deck.Slides),
			ID:      file.DerivedID(),
			Source:  source,
			Chapter: file.Chapter,
			Tokens:  soml.Tokenize(string(src)),
		})
	}

	diags = append(diags, core.Resolve(deck)...)
	diags = append(diags, core.Validate(deck)...)
	if opts.Strict {
		diags = diags.Strict()
	}
	if err := diags.Err(); err != nil {
		return nil, diags, err
	}
	return deck, diags, nil
}

// Render는 덱을 타깃별 산출 파일로 만든다 — 디스크에 쓰지 않는다.
func Render(deck *soml.Deck, targets []render.Target) (map[render.Target]render.Files, error) {
	if len(targets) == 0 {
		targets = render.DefaultTargets()
	}
	available := Renderers()
	out := make(map[render.Target]render.Files, len(targets))
	for _, target := range targets {
		renderer, ok := available[target]
		if !ok {
			return nil, fmt.Errorf("타깃 %q의 렌더러가 없다", string(target))
		}
		files, err := renderer.Render(deck)
		if err != nil {
			return nil, fmt.Errorf("%s 렌더 실패: %w", string(target), err)
		}
		out[target] = files
	}
	return out, nil
}

// Build는 프로젝트를 읽어 산출물을 디스크에 쓴다.
func Build(opts Options) (*Result, error) {
	deck, diags, err := LoadDeck(opts)
	if err != nil {
		return &Result{Diagnostics: diags}, err
	}

	targets := opts.Targets
	if len(targets) == 0 {
		targets = render.DefaultTargets()
	}
	rendered, err := Render(deck, targets)
	if err != nil {
		return &Result{Deck: deck, Diagnostics: diags}, err
	}

	outDir, err := resolveOutDir(opts)
	if err != nil {
		return &Result{Deck: deck, Diagnostics: diags}, err
	}
	result := &Result{Deck: deck, Diagnostics: diags, OutDir: outDir}
	// 타깃 순서를 유지해 로그가 사용자가 쓴 --target 순서와 같게 보이게 한다.
	for _, target := range targets {
		files := rendered[target]
		for _, name := range files.Paths() {
			if err := writeFile(outDir, name, files[name]); err != nil {
				return result, err
			}
			result.Written = append(result.Written, name)
		}
	}
	return result, nil
}

// resolveOutDir은 출력 디렉토리 절대 경로를 정하고 만든다.
func resolveOutDir(opts Options) (string, error) {
	projectDir := opts.ProjectDir
	if projectDir == "" {
		projectDir = "."
	}
	outDir := opts.OutDir
	if outDir == "" {
		cfg, err := core.LoadConfig(projectDir)
		if err != nil {
			return "", err
		}
		outDir = cfg.OutDir
	}
	if !filepath.IsAbs(outDir) {
		outDir = filepath.Join(projectDir, outDir)
	}
	abs, err := filepath.Abs(outDir)
	if err != nil {
		return "", fmt.Errorf("출력 디렉토리 경로를 정할 수 없다: %w", err)
	}
	if err := os.MkdirAll(abs, 0o755); err != nil {
		return "", fmt.Errorf("출력 디렉토리 %s를 만들 수 없다: %w", abs, err)
	}
	return abs, nil
}

// writeFile은 산출 파일 하나를 쓴다. 렌더러가 준 경로에 하위 디렉토리가 있으면 만든다.
func writeFile(outDir, name string, data []byte) error {
	dest := filepath.Join(outDir, filepath.FromSlash(name))
	if dir := filepath.Dir(dest); dir != outDir {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("%s를 만들 수 없다: %w", dir, err)
		}
	}
	if err := os.WriteFile(dest, data, 0o644); err != nil {
		return fmt.Errorf("%s를 쓸 수 없다: %w", dest, err)
	}
	return nil
}
