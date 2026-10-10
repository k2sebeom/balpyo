// Package pipeline은 수집부터 산출물 쓰기까지의 빌드 흐름을 조립한다.
//
// 왜 별도 패키지인가: core는 렌더러를 임포트하지 않고(해석·검증 전용), 렌더러는 core를
// 임포트하지 않는다(덱 모델만 본다). 둘을 잇는 자리가 필요하고, 그 자리는 cmd/balpyo가
// 될 수 없다 — main 패키지는 임포트할 수 없어서 테스트 하네스가 같은 흐름을 돌릴 수
// 없기 때문이다. 그래서 internal에 둔다: 공개 API 약속 없이 CLI와 테스트가 공유한다.
package pipeline

import (
	"fmt"
	"io/fs"
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
	// Theme가 비지 않으면 설정의 theme를 덮는다 (`--theme`).
	//
	// 플래그가 있는 이유는 **한 덱을 두 테마로 보는 일**이 테마를 고르는 방법이라는 것이다:
	// 설정을 고쳐 빌드하고 되돌리는 절차는 되돌리기를 잊는 순간 잘못된 테마를 커밋한다.
	// 어휘 검사는 설정과 같은 목록을 보되 진단이 **플래그를 가리킨다** (LoadDeck).
	Theme string
	// ThemeHome은 설치한 테마의 홈 디렉토리다 (`themes/<이름>/`을 담는다 — `balpyo theme install`이
	// 쓰는 자리). 비면 설치한 테마를 보지 않는다: 테스트가 개발자의 홈에 매이지 않고, 홈을 정하는
	// 일(`BALPYO_HOME` · `~/.balpyo`)은 CLI의 몫이다.
	ThemeHome string
	// Strict는 경고를 에러로 승격한다.
	Strict bool
	// CheckLayout은 배치 진단(geom.Diagnose)을 위해 **지오메트리 패스를 강제**한다.
	//
	// 좌표를 쓰는 타깃이 있으면(pptx) 이미 재므로 배치 진단은 값을 공짜로 얻는다 — 이
	// 플래그가 뜻을 갖는 자리는 `--target html`만 쓰는 저작 루프다. 기본을 끈
	// 이유는 브라우저 기동이 빌드의 가장 큰 고정 비용이고, 브라우저가 없는 환경에서도
	// 웹 빌드는 돌아가야 하기 때문이다.
	CheckLayout bool
	// Measurer는 지오메트리 패스를 도는 것이다. nil이면 이 빌드가 브라우저를 직접 띄우고
	// 닫는다 (`balpyo build`의 동작).
	//
	// 이 구멍이 있는 이유는 `balpyo watch`다: 기동이 빌드 고정 비용의 대부분이라
	// 저작 루프는 브라우저 하나를 웜으로 재사용해야 하는데, 그 수명은 루프의 것이지 빌드
	// 하나의 것이 아니다. 그래서 파이프라인은 **재는 일**만 받고 **띄우고 닫는 일**은 소유하지
	// 않는다.
	Measurer Measurer
}

// Measurer는 웹 산출물을 재서 좌표를 내는 것이다. *geom.Session이 이 인터페이스를 만족한다.
type Measurer interface {
	Measure(deck *soml.Deck, htmlPath string) (*geom.Geometry, error)
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
	// 결과에 남기는 이유: 호출자(검수 · 좌표 스냅샷)가 **이 빌드가 실제로 무엇을
	// 쟀는지**를 알아야 하고, 그것을 다시 알아내려면 브라우저를 한 번 더 띄워야 한다.
	Geometry *geom.Geometry
}

// Renderers는 타깃 이름에서 렌더러를 찾는다. 새 타깃을 붙이는 지점이다.
//
// 지오메트리를 인자로 받는다 — 좌표를 쓰는 렌더러가 어느 것인지 이 표에 드러나고, 좌표 없이
// 만든 렌더러가 좌표가 필요한 덱을 만나면 렌더러 쪽에서 멈춘다 (pptx.Renderer.checkGeometry).
func Renderers(g *geom.Geometry) map[render.Target]render.Renderer {
	html := web.New()
	if g != nil {
		// 잰 빌드의 웹은 연결선을 굽는다 — 끝점이 잰 rect에서 나온다 (연결선은 흐름에 끼지 않으므로
		// 측정한 HTML과 배치가 같다).
		html = web.NewWithBoxes(g.RouteBox)
	}
	return map[render.Target]render.Renderer{
		render.TargetHTML: html,
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
	home := themeHomeFS(opts.ThemeHome)
	cfg, err := core.LoadConfigWithThemes(projectDir, home)
	if err != nil {
		return nil, nil, err
	}
	stage, err := cfg.Aspect.Stage()
	if err != nil {
		return nil, nil, err
	}
	// 프로젝트 루트를 **한 fs.FS로** 본다: 슬라이드 수집과 에셋 해석의 경로 기준이 하나라는
	// 사실이 그 한 값으로 보인다 (`<image src>`는 프로젝트 루트 기준이다 — core.ResolveAssets).
	// 테마 이름의 어휘도 이 기준을 쓴다 (`themes/<이름>/`) — 설치한 테마의 홈은 이 값에 붙어서
	// 이름을 찾는 자리만 늘린다 (파일은 여전히 프로젝트 루트에서 읽는다).
	fsys := core.WithInstalledThemes(os.DirFS(projectDir), home)

	// 테마 옵션은 설정을 **읽은 뒤** 갈아 끼운다 (OutDir과 같은 자리): 아래 흐름은 cfg 하나만
	// 보므로 덮어쓰기가 여기서 끝나야 값 층·렌더러가 두 출처를 알 필요가 없다. 어휘 검사를
	// 여기서 다시 하는 이유는 진단이 가리킬 자리가 다르다는 것이다 — 설정의 오타는
	// `balpyo.json`이고 이 오타는 방금 친 명령이다 (core.LoadConfig와 같은 목록·문구).
	if opts.Theme != "" {
		if err := core.CheckThemeName(fsys, opts.Theme); err != nil {
			return nil, nil, fmt.Errorf("--theme: %w", err)
		}
		cfg.Theme = opts.Theme
	}
	files, err := soml.Collect(fsys, cfg.SlidesDir)
	if err != nil {
		return nil, nil, err
	}
	if len(files) == 0 {
		return nil, nil, fmt.Errorf("no slides: %s/ needs %s files", cfg.SlidesDir, soml.Ext)
	}

	deck := &soml.Deck{
		Title:  cfg.Title,
		Author: cfg.Author,
		Lang:   cfg.Lang,
		Aspect: cfg.Aspect,
		Stage:  stage,
		// 테마 요청도 크롬과 같은 태도로 옮긴다 — 이름을 패키지로 바꾸는 것은
		// core.Resolve다. 설정 로드가 이미 어휘를 검사했으므로 여기 도달한 이름은 내장 테마,
		// 이 프로젝트의 `themes/<이름>/`, 설치한 테마 중 하나다.
		ThemeName: cfg.Theme,
		// 크롬 요청은 설정에서 그대로 옮긴다 — 테마 선언과 합치는 것은 core.Resolve다.
		Chrome: soml.DeckChrome{Footer: cfg.Footer, PageNumber: cfg.PageNumber},
		Slides: make([]*soml.Slide, 0, len(files)),
	}
	var diags soml.Diagnostics
	for _, file := range files {
		source := path.Join(cfg.SlidesDir, file.Path)
		src, err := os.ReadFile(filepath.Join(projectDir, filepath.FromSlash(source)))
		if err != nil {
			return nil, nil, fmt.Errorf("cannot read %s: %w", source, err)
		}
		doc, fileDiags := soml.Parse(source, string(src))
		diags = append(diags, fileDiags...)
		// `<slide skip>`은 파일을 남기되 덱에서 뺀다. Index를 남은 슬라이드로
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
			IDSpan:     doc.IDSpan,
			Source:     source,
			Chapter:    file.Chapter,
			Layout:     soml.SlideLayout(doc.Layout),
			LayoutSpan: doc.LayoutSpan,
			// justify도 layout처럼 원문·구간을 함께 옮긴다 — 어휘 검사는 core다.
			Justify:     soml.BoxJustify(doc.Justify),
			JustifySpan: doc.JustifySpan,
			Enter:       doc.Enter,
			EnterSpan:   doc.EnterSpan,
			// 전환도 원문 그대로 옮긴다 — 어휘 · 테마의 기본값은 core가 본다.
			Invert:         doc.Invert,
			Transition:     soml.SlideTransition(doc.Transition),
			TransitionSpan: doc.TransitionSpan,
			Notes:          doc.Notes,
			Content:        doc.Content,
			Tokens:         doc.Tokens,
		})
	}

	diags = append(diags, core.Resolve(deck, fsys)...)
	// 에셋 패스는 Resolve 다음, Validate 형제다 — 확정 값의 `src`를 읽고 **디스크를 봐야** 알
	// 수 있는 것(존재·포맷·고유 크기)을 본다. 파일 시스템이 여기서만 core로 넘어간다.
	diags = append(diags, core.ResolveAssets(deck, fsys)...)
	diags = append(diags, core.Validate(deck)...)
	// 층 순서로 쌓인 진단을 소스 순서로 편다 — build · check · watch가 같은 순서를 본다.
	diags = diags.Sorted()
	if opts.Strict {
		diags = diags.Strict()
	}
	if err := diags.Err(); err != nil {
		return nil, diags, err
	}
	// 모두 skip이면 렌더할 것이 없다 — 산출물을 빈 덱으로 내는 대신 멈춘다. 파일이 없는
	// 경우와 같은 이유의 실패이므로 같은 자리에서 같은 모양으로 알린다.
	if len(deck.Slides) == 0 {
		return nil, diags, fmt.Errorf("no slides to render: all %d collected slides are skipped", len(files))
	}
	return deck, diags, nil
}

// Render는 덱을 타깃별 산출 파일로 만든다 — 산출물을 디스크에 쓰지 않는다.
//
// 순서는 웹 렌더 → 지오메트리 → 나머지 렌더다. 지오메트리 패스가 재는 것은 웹
// 타깃의 산출물이므로 웹 렌더가 앞에 와야 하고, PPTX 좌표가 그 측정에서 나오므로 PPTX가
// 뒤에 온다.
//
// 반환값이 셋인 이유: 좌표(재지 않았으면 nil)와 **배치 진단**이다. 배치 진단은 좌표에서만
// 나오므로 LoadDeck이 낼 수 없고(그때는 브라우저를 띄우기 전이다), 그래서 승격(`--strict`)도
// 여기서 한 번 더 한다 — 진단이 나오는 자리마다 같은 규칙을 적용하는 쪽이 한 곳에 모으는
// 것보다 안전하다 (LoadDeck의 승격과 짝이다).
//
// opts는 Targets·Strict·CheckLayout만 읽는다 (경로는 덱이 이미 읽혀 있으므로 뜻이 없다).
func Render(deck *soml.Deck, opts Options) (map[render.Target]render.Files, *geom.Geometry, soml.Diagnostics, error) {
	targets := opts.Targets
	if len(targets) == 0 {
		targets = render.DefaultTargets()
	}
	// 타깃 이름을 먼저 검사한다 — 오타 하나로 실패할 빌드가 브라우저를 띄운 뒤에 죽으면
	// 사용자는 오타를 알기까지 기동 시간을 기다린다.
	for _, target := range targets {
		if _, ok := Renderers(nil)[target]; !ok {
			return nil, nil, nil, fmt.Errorf("no renderer for target %q", string(target))
		}
	}

	var geometry *geom.Geometry
	var diags soml.Diagnostics
	if needsGeometry(deck, targets, opts.CheckLayout) {
		measured, err := measure(deck, opts.Measurer)
		if err != nil {
			return nil, nil, nil, err
		}
		geometry = measured
		// 잰 빌드는 배치를 **항상** 본다: 숫자가 이미 있는데 묻지 않을 이유가 없다.
		diags = geom.Diagnose(deck, geometry)
		if opts.Strict {
			diags = diags.Strict()
		}
		if err := diags.Err(); err != nil {
			// 승격된 에러는 산출물을 쓰기 **전에** 멈춘다 — 반쯤 갱신된 출력 디렉토리가
			// 남으면 다음 빌드가 무엇이 새 산출물인지 말해 주지 못한다.
			return nil, geometry, diags, err
		}
	}

	available := Renderers(geometry)
	out := make(map[render.Target]render.Files, len(targets))
	for _, target := range targets {
		files, err := available[target].Render(deck)
		if err != nil {
			return nil, geometry, diags, fmt.Errorf("%s render failed: %w", string(target), err)
		}
		out[target] = files
	}
	return out, geometry, diags, nil
}

// needsGeometry는 이 빌드가 브라우저를 띄워야 하는지 정한다.
//
// 잴 요소가 하나라도 있고, **좌표를 쓰는 타깃**이 있거나(웹은 자기 좌표를 브라우저가 직접
// 계산하므로 필요 없다) 저자가 배치 검사를 요청했어야 띄운다. 브라우저 기동은 빌드의 가장 큰
// 고정 비용이므로, `--target html`만 쓰는 저작 루프가 그 비용을 내지 않는 것이 중요하다 —
// 배치 진단을 그 루프에서도 받는 길이 `--check-layout`이다.
//
// 판정을 pptx 렌더러와 같은 기준(geom.Refs가 비었는가)으로 두는 이유: 여기서는 재지 않기로
// 했는데 렌더러는 좌표를 요구하면 빈 슬라이드 프로젝트가 브라우저 없이 못 돌아간다.
func needsGeometry(deck *soml.Deck, targets []render.Target, checkLayout bool) bool {
	if len(geom.Refs(deck)) == 0 && !deckHasChrome(deck) {
		return false
	}
	if checkLayout || deckHasConnectors(deck) || deckHasMeasuredInk(deck) {
		// 연결선의 끝점과 잰 박스로 그리는 도형(cube · 괄호)의 그림은 잰 rect에서 나오므로 HTML만 내는
		// 빌드도 잰다.
		return true
	}
	for _, target := range targets {
		if target == render.TargetPPTX {
			return true
		}
	}
	return false
}

// deckHasConnectors는 연결선이 있는 슬라이드가 있는지다 (연결선은 슬라이드 직속뿐이다).
func deckHasConnectors(deck *soml.Deck) bool {
	for _, slide := range deck.Slides {
		for _, node := range slide.Content {
			if elem, ok := node.(*soml.Element); ok && elem.Name == soml.ConnectorElement {
				return true
			}
		}
	}
	return false
}

// deckHasMeasuredInk는 잰 박스로 그림을 그리는 도형이 있는지다 (web.ShapeDrawnFromMeasuredBox).
func deckHasMeasuredInk(deck *soml.Deck) bool {
	var walk func(nodes []soml.Node) bool
	walk = func(nodes []soml.Node) bool {
		for _, node := range nodes {
			elem, ok := node.(*soml.Element)
			if !ok {
				continue
			}
			if s := elem.ShapeStyle(); s != nil && web.ShapeDrawnFromMeasuredBox(s.Type) {
				return true
			}
			if walk(elem.Children) {
				return true
			}
		}
		return false
	}
	for _, slide := range deck.Slides {
		if walk(slide.Content) {
			return true
		}
	}
	return false
}

// deckHasChrome은 확정된 덱 크롬이 실린 슬라이드가 있는지다.
//
// 크롬도 잰 좌표를 쓰므로(PPTX가 rect에 도형을 놓는다) 콘텐츠가 없는 슬라이드만 있는 덱도
// 크롬을 켜면 측정이 필요하다 — 판정을 pptx.checkGeometry와 같은 기준으로 둔다.
func deckHasChrome(deck *soml.Deck) bool {
	for _, slide := range deck.Slides {
		if len(slide.Chrome) > 0 {
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
// measurer가 nil이면 geom.Measure다 — 브라우저를 한 번 띄우고 닫는다.
func measure(deck *soml.Deck, measurer Measurer) (*geom.Geometry, error) {
	// 측정용 HTML에는 연결선이 없다 — 끝점이 이 측정의 결과에서 나온다. 연결선은 흐름에 끼지 않으므로
	// 사용자가 받는 HTML(연결선을 구운 것)과 배치가 같다.
	files, err := web.NewForMeasurement().Render(deck)
	if err != nil {
		return nil, fmt.Errorf("cannot prepare geometry pass input: %w", err)
	}
	dir, err := os.MkdirTemp("", "balpyo-geom-")
	if err != nil {
		return nil, fmt.Errorf("cannot create geometry pass temp directory: %w", err)
	}
	defer os.RemoveAll(dir)
	for _, name := range files.Paths() {
		if err := writeFile(dir, name, files[name]); err != nil {
			return nil, err
		}
	}
	htmlPath := filepath.Join(dir, web.IndexFile)
	var g *geom.Geometry
	if measurer != nil {
		g, err = measurer.Measure(deck, htmlPath)
	} else {
		g, err = geom.Measure(deck, htmlPath)
	}
	if err != nil {
		return nil, fmt.Errorf("geometry pass failed: %w", err)
	}
	return g, nil
}

// CheckResult는 판정만 한 결과다 (Check).
type CheckResult struct {
	Deck *soml.Deck
	// Diagnostics는 에러가 없어도 남는 경고를 담는다 (Build와 같은 규칙).
	Diagnostics soml.Diagnostics
	// Measured는 이 판정이 지오메트리 패스를 돌았는지다 — 배치 경고가 판정에 포함됐는지를
	// 호출자가 사용자에게 말할 수 있어야 한다 ("배치는 보지 않았다"와 "배치도 봤고 깨끗하다"는
	// 다른 문장이다).
	Measured bool
}

// Check는 산출물을 쓰지 않고 판정만 한다 — `balpyo check`다.
//
// 판정은 Build와 **같은 코드**다: 설정 → 수집 → 파싱 → 값 해석 → 에셋 → 검증, 그리고
// CheckLayout이면 측정 + 배치 진단. 빠른 길(파싱까지만)로 가지 않는 이유는 "check는 통과했는데
// build가 실패한다"가 가능해지는 순간 이 명령의 가치가 사라진다는 것이다.
//
// 디스크에는 아무것도 남지 않는다 — 출력 디렉토리를 만들지도 않는다 (Build는 resolveOutDir이
// 만든다). 측정에 필요한 웹 문서는 measure가 임시 디렉토리에 쓰고 지운다.
//
// 결과는 **언제나 non-nil**이다: 실패한 검사의 진단도 사람에게 보여야 한다.
func Check(opts Options) (*CheckResult, error) {
	deck, diags, err := LoadDeck(opts)
	if err != nil {
		return &CheckResult{Diagnostics: diags}, err
	}
	result := &CheckResult{Deck: deck, Diagnostics: diags}
	// 타깃을 보지 않는다: 진단은 타깃에 의존하지 않고(배치 경고까지 잰 박스에서 나온다),
	// 렌더 단계에서만 나는 실패는 Build의 판정이다.
	if !needsGeometry(deck, nil, opts.CheckLayout) {
		return result, nil
	}
	geometry, err := measure(deck, opts.Measurer)
	if err != nil {
		return result, err
	}
	result.Measured = true
	layoutDiags := geom.Diagnose(deck, geometry)
	if opts.Strict {
		layoutDiags = layoutDiags.Strict()
	}
	result.Diagnostics = append(result.Diagnostics, layoutDiags...).Sorted()
	return result, layoutDiags.Err()
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
	rendered, geometry, layoutDiags, err := Render(deck, opts)
	// 배치 진단은 렌더가 실패해도 사람에게 보여야 한다 — 승격된 경고가 곧 그 실패의 이유다.
	diags = append(diags, layoutDiags...).Sorted()
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

// OutDir은 이 옵션이 산출물을 쓸 디렉토리를 정하고 만든다.
//
// 빌드보다 **먼저** 물을 수 있어야 하는 호출자가 있어서 공개한다: `balpyo watch`는 그 디렉토리를
// 서브하고 감시 대상에서 빼야 하는데, 첫 빌드는 실패할 수 있다 (그때도 서버는 떠 있어야 한다).
func OutDir(opts Options) (string, error) {
	return resolveOutDir(opts)
}

// OutDirPath는 이 옵션이 산출물을 쓸 디렉토리의 절대 경로다 — OutDir과 같은 규칙이지만
// 디렉토리를 만들지 않는다. 빌드 전에 경로만 견주는 호출자(`balpyo preview`가 갤러리와 출력이
// 겹치는지 보는 자리)가 판정하다가 디스크를 건드리지 않게 한다.
func OutDirPath(opts Options) (string, error) {
	projectDir := opts.ProjectDir
	if projectDir == "" {
		projectDir = "."
	}
	outDir := opts.OutDir
	if outDir == "" {
		cfg, err := core.LoadConfigWithThemes(projectDir, themeHomeFS(opts.ThemeHome))
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
		return "", fmt.Errorf("cannot resolve output directory path: %w", err)
	}
	return abs, nil
}

// resolveOutDir은 출력 디렉토리 절대 경로를 정하고 만든다.
func resolveOutDir(opts Options) (string, error) {
	abs, err := OutDirPath(opts)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(abs, 0o755); err != nil {
		return "", fmt.Errorf("cannot create output directory %s: %w", abs, err)
	}
	return abs, nil
}

// writeFile은 산출 파일 하나를 쓴다. 렌더러가 준 경로에 하위 디렉토리가 있으면 만든다.
func writeFile(outDir, name string, data []byte) error {
	dest := filepath.Join(outDir, filepath.FromSlash(name))
	if dir := filepath.Dir(dest); dir != outDir {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("cannot create %s: %w", dir, err)
		}
	}
	if err := os.WriteFile(dest, data, 0o644); err != nil {
		return fmt.Errorf("cannot write %s: %w", dest, err)
	}
	return nil
}

// themeHomeFS는 설치한 테마의 홈을 파일 시스템으로 연다 (비면 nil — 설치한 테마를 보지 않는다).
func themeHomeFS(dir string) fs.FS {
	if dir == "" {
		return nil
	}
	return os.DirFS(dir)
}
