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
	"github.com/k2sebeom/balpyo/render/geom"
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
	// Geometry는 지오메트리 패스가 잰 좌표다 (측정하지 않은 빌드에서는 nil).
	//
	// 결과에 남기는 이유: 리뷰와 다음 iteration의 좌표 스냅샷이 **이 빌드가 실제로 무엇을
	// 쟀는지**를 알아야 하고, 그것을 다시 알아내려면 브라우저를 한 번 더 띄워야 한다.
	Geometry *geom.Geometry
}

// Renderers는 타깃 이름에서 렌더러를 찾는다. 새 타깃을 붙이는 지점이다.
//
// 지오메트리를 인자로 받는다 — 좌표를 쓰는 렌더러가 어느 것인지 이 표에 드러나고, 좌표 없이
// 만든 렌더러가 좌표가 필요한 덱을 만나면 렌더러 쪽에서 멈춘다 (pptx.Renderer.checkGeometry).
func Renderers(g *geom.Geometry) map[render.Target]render.Renderer {
	return map[render.Target]render.Renderer{
		render.TargetHTML: web.New(),
		render.TargetPPTX: pptx.NewWithGeometry(g),
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

	// 프로젝트 루트를 **한 fs.FS로** 본다: 슬라이드 수집과 에셋 해석의 경로 기준이 하나라는
	// 사실이 그 한 값으로 보인다 (`<image src>`는 프로젝트 루트 기준이다 — core.ResolveAssets).
	fsys := os.DirFS(projectDir)
	files, err := soml.Collect(fsys, cfg.SlidesDir)
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
		doc, fileDiags := soml.Parse(source, string(src))
		diags = append(diags, fileDiags...)
		// `<slide skip>`은 파일을 남기되 덱에서 뺀다 (PRD §7.1). Index를 남은 슬라이드로
		// 다시 세기 때문에 이 판단이 렌더러보다 앞에 있어야 한다 — 렌더러가 건너뛰면
		// 슬라이드 번호에 구멍이 생긴다.
		if doc.Skip {
			continue
		}
		id := doc.ID
		if id == "" {
			id = file.DerivedID()
		}
		deck.Slides = append(deck.Slides, &soml.Slide{
			Index:      len(deck.Slides),
			ID:         id,
			Source:     source,
			Chapter:    file.Chapter,
			Layout:     soml.SlideLayout(doc.Layout),
			LayoutSpan: doc.LayoutSpan,
			Notes:      doc.Notes,
			Content:    doc.Content,
			Tokens:     doc.Tokens,
		})
	}

	diags = append(diags, core.Resolve(deck)...)
	// 에셋 패스는 Resolve 다음, Validate 형제다 — 확정 값의 `src`를 읽고 **디스크를 봐야** 알
	// 수 있는 것(존재·포맷·고유 크기)을 본다. 파일 시스템이 여기서만 core로 넘어간다.
	diags = append(diags, core.ResolveAssets(deck, fsys)...)
	diags = append(diags, core.Validate(deck)...)
	if opts.Strict {
		diags = diags.Strict()
	}
	if err := diags.Err(); err != nil {
		return nil, diags, err
	}
	// 모두 skip이면 렌더할 것이 없다 — 산출물을 빈 덱으로 내는 대신 멈춘다. 파일이 없는
	// 경우와 같은 이유의 실패이므로 같은 자리에서 같은 모양으로 알린다.
	if len(deck.Slides) == 0 {
		return nil, diags, fmt.Errorf("렌더할 슬라이드가 없다: 수집된 %d개가 모두 skip 상태다", len(files))
	}
	return deck, diags, nil
}

// Render는 덱을 타깃별 산출 파일로 만든다 — 산출물을 디스크에 쓰지 않는다.
//
// 순서는 웹 렌더 → 지오메트리 → 나머지 렌더다 (PRD §9.3). 지오메트리 패스가 재는 것은 웹
// 타깃의 산출물이므로 웹 렌더가 앞에 와야 하고, PPTX 좌표가 그 측정에서 나오므로 PPTX가
// 뒤에 온다. 두 번째 반환값은 측정한 좌표이며, 재지 않았으면 nil이다.
func Render(deck *soml.Deck, targets []render.Target) (map[render.Target]render.Files, *geom.Geometry, error) {
	if len(targets) == 0 {
		targets = render.DefaultTargets()
	}
	// 타깃 이름을 먼저 검사한다 — 오타 하나로 실패할 빌드가 브라우저를 띄운 뒤에 죽으면
	// 사용자는 오타를 알기까지 기동 시간을 기다린다.
	for _, target := range targets {
		if _, ok := Renderers(nil)[target]; !ok {
			return nil, nil, fmt.Errorf("타깃 %q의 렌더러가 없다", string(target))
		}
	}

	var geometry *geom.Geometry
	if needsGeometry(deck, targets) {
		measured, err := measure(deck)
		if err != nil {
			return nil, nil, err
		}
		geometry = measured
	}

	available := Renderers(geometry)
	out := make(map[render.Target]render.Files, len(targets))
	for _, target := range targets {
		files, err := available[target].Render(deck)
		if err != nil {
			return nil, geometry, fmt.Errorf("%s 렌더 실패: %w", string(target), err)
		}
		out[target] = files
	}
	return out, geometry, nil
}

// needsGeometry는 이 빌드가 브라우저를 띄워야 하는지 정한다.
//
// 두 조건이 모두 맞아야 띄운다. **좌표를 쓰는 타깃**이 있어야 하고(웹은 자기 좌표를 브라우저가
// 직접 계산하므로 필요 없다), 잴 요소가 하나라도 있어야 한다. 브라우저 기동은 빌드의 가장 큰
// 고정 비용이므로, `--target html`만 쓰는 저작 루프가 그 비용을 내지 않는 것이 중요하다.
//
// 판정을 pptx 렌더러와 같은 기준(geom.Refs가 비었는가)으로 두는 이유: 여기서는 재지 않기로
// 했는데 렌더러는 좌표를 요구하면 빈 슬라이드 프로젝트가 브라우저 없이 못 돌아간다.
func needsGeometry(deck *soml.Deck, targets []render.Target) bool {
	if len(geom.Refs(deck)) == 0 {
		return false
	}
	for _, target := range targets {
		if target == render.TargetPPTX {
			return true
		}
	}
	return false
}

// measure는 웹 산출물을 임시 디렉토리에 쓰고 그것을 재서 좌표를 얻는다.
//
// 임시 디렉토리를 쓰는 이유는 두 가지다. (1) 출력 디렉토리에 쓴 파일을 재면 측정이 타깃 순서에
// 의존한다 — `--target pptx`만 준 빌드에는 그 파일이 없다. (2) 측정은 산출물을 쓰기 전에
// 끝나야 한다: 좌표가 어긋나 빌드가 실패했는데 index.html만 남으면 출력 디렉토리가 반쯤
// 갱신된 상태가 된다.
//
// 웹 렌더를 여기서 한 번 더 도는 것은 의도다. 렌더는 결정적이고(같은 덱 → 같은 바이트,
// pipeline_test가 못 박는다) 비용이 브라우저 기동의 수천분의 일이므로, 렌더 결과를 두 단계에
// 걸쳐 들고 다니는 대신 재렌더한다 — 재는 바이트와 사용자가 받는 바이트가 같다는 사실이
// 흐름을 읽는 것만으로 드러나는 쪽이 낫다.
func measure(deck *soml.Deck) (*geom.Geometry, error) {
	files, err := web.New().Render(deck)
	if err != nil {
		return nil, fmt.Errorf("지오메트리 패스의 입력을 만들 수 없다: %w", err)
	}
	dir, err := os.MkdirTemp("", "balpyo-geom-")
	if err != nil {
		return nil, fmt.Errorf("지오메트리 패스의 임시 디렉토리를 만들 수 없다: %w", err)
	}
	defer os.RemoveAll(dir)
	for _, name := range files.Paths() {
		if err := writeFile(dir, name, files[name]); err != nil {
			return nil, err
		}
	}
	g, err := geom.Measure(deck, filepath.Join(dir, web.IndexFile))
	if err != nil {
		return nil, fmt.Errorf("지오메트리 패스 실패: %w", err)
	}
	return g, nil
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
	rendered, geometry, err := Render(deck, targets)
	if err != nil {
		return &Result{Deck: deck, Diagnostics: diags, Geometry: geometry}, err
	}

	outDir, err := resolveOutDir(opts)
	if err != nil {
		return &Result{Deck: deck, Diagnostics: diags, Geometry: geometry}, err
	}
	result := &Result{Deck: deck, Diagnostics: diags, OutDir: outDir, Geometry: geometry}
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
