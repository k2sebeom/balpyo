// Package preview는 검수 산출물을 만든다 — PPTX를 그림으로 되돌리고, 웹 스크린샷과 나란히
// 놓은 갤러리 문서 하나로 묶는다.
//
// 왜 테스트 하네스가 아니라 여기인가: 이 흐름은 개발 내내 사람이 손으로 돌던
// 것이다 — 임시 테스트 파일을 쓰고, 컨테이너에 들어가고, 래스터 도구를 돌리고, PNG를
// 열어 본다. 그 루프가 제품이어야 저자도 같은 것을 볼 수 있다. 반대로 `cmd/balpyo`가
// internal/testkit을 임포트하면 릴리스 바이너리에 testing이 링크되고, "도구가 없으면
// 건너뛴다"는 **테스트 정책**이 사용자 행동이 된다.
//
// 그래서 소유가 뒤집혔다: 엔진(래스터화·갤러리)은 이 패키지가 갖고, 하네스는 그 위에
// 테스트 정책(건너뛰기·t.Fatal)만 얹는다. internal/testkit이 internal/preview를 임포트하며,
// 반대 방향은 없다.
//
// `balpyo preview`는 **판정하지 않는다**. 두 타깃의 계약은 픽셀이 아니라 컴포넌트·
// 레이아웃 동일성이고 그것은 `go test`가 본다 — 여기서 하는 일은 그림을 사람 앞에
// 나란히 놓는 것까지다. 픽셀 diff · 비율 · 허용 오차 배지는 없다.
package preview

import (
	"fmt"
	"image"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/k2sebeom/balpyo/internal/pipeline"
	"github.com/k2sebeom/balpyo/render"
	"github.com/k2sebeom/balpyo/render/geom"
	"github.com/k2sebeom/balpyo/render/pptx"
	"github.com/k2sebeom/balpyo/render/web"
	"github.com/k2sebeom/balpyo/soml"
)

// DefaultDirName은 갤러리가 떨어지는 디렉토리 이름이다 (프로젝트 루트 기준).
//
// 산출물 디렉토리(dist/) 안에 두지 않는 이유: 검수 산출물은 배포물이 아니다. dist/를 그대로
// 올리는 사람이 웹 스크린샷 수십 장을 함께 올리게 된다.
const DefaultDirName = "preview"

// DefaultRenders는 Renders를 주지 않았을 때 갤러리에 나란히 놓는 렌더다.
//
// 기본이 둘인 이유: 이 명령의 존재 이유가 **두 타깃을 나란히 눈으로 보는 것**이다. 다만 PPTX
// 열은 래스터 엔진(OnlyOffice, 또는 LibreOffice · poppler)이 있어야 그릴 수 있다 — 기본 렌더에서 도구가 없으면 웹 열만으로
// 갤러리를 만들고 그 사실을 로그와 갤러리 안내에 남긴다. 하나만 보려면 그 타깃을 준다.
func DefaultRenders() []render.Target { return []render.Target{render.TargetHTML, render.TargetPPTX} }

// Options는 검수 실행의 입력이다.
type Options struct {
	// Build는 빌드 옵션이다 — `balpyo build`가 받는 것과 같은 값이다. Targets는 Renders에서
	// 정해지므로 여기서 준 값은 무시한다.
	Build pipeline.Options
	// Renders는 갤러리에 나타날 렌더다. 비면 DefaultRenders이고, 그때는 래스터 도구가 없으면
	// PPTX 열을 뺀다. 명시한 pptx는 도구가 없으면 빌드 전에 멈춘다.
	Renders []render.Target
	// Dir가 비면 <프로젝트 루트>/preview다.
	Dir string
	// Log가 nil이면 로그를 버린다.
	Log io.Writer
}

// SlideReport는 슬라이드 한 장의 검수 결과다.
type SlideReport struct {
	Index int
	ID    string
}

// Result는 검수 실행의 산출이다.
type Result struct {
	// Index는 갤러리 문서 경로다.
	Index string
	// Dir는 갤러리 디렉토리다.
	Dir string
	// OutDir은 빌드 산출물 디렉토리다 — PPTX를 PowerPoint/Keynote로 직접 열어 보는 확인은
	// 갤러리가 대신할 수 없다.
	OutDir string
	// Renders는 실제로 갤러리에 놓인 렌더다 (기본 렌더에서 PPTX가 빠졌을 수 있다).
	Renders []render.Target
	// SkippedPPTX는 기본 렌더에서 PPTX 열을 뺀 이유다 (뺀 적이 없으면 빈 문자열).
	SkippedPPTX string
	// Slides는 슬라이드 순서대로의 결과다.
	Slides []SlideReport
	// Diagnostics는 빌드가 남긴 경고다.
	Diagnostics soml.Diagnostics
	// Engine은 PPTX 열을 그린 엔진 이름이다 (PPTX 열이 없으면 빈 문자열).
	Engine string
	// Noise는 래스터 엔진이 낸 무해하지 않은 출력이다 (있으면 산출물을 의심할 자리다).
	Noise string
}

// Run은 프로젝트를 빌드하고 검수 갤러리를 만든다.
//
// 순서가 중요하다: **도구를 먼저 찾는다.** 여섯 장을 빌드하고 브라우저까지 띄운 뒤에
// "래스터 엔진이 없다"를 알려 주는 것은 사람의 시간을 버리는 일이다.
func Run(opts Options) (*Result, error) {
	renders := opts.Renders
	optional := len(renders) == 0
	if optional {
		renders = DefaultRenders()
	}
	log := opts.Log
	if log == nil {
		log = io.Discard
	}
	var wantWeb, wantPPTX bool
	for _, target := range renders {
		switch target {
		case render.TargetHTML:
			wantWeb = true
		case render.TargetPPTX:
			wantPPTX = true
		default:
			return nil, fmt.Errorf("검수할 수 없는 타깃 %q", string(target))
		}
	}

	var raster Rasterizer
	skipped := ""
	if wantPPTX {
		found, err := NewRasterizer()
		switch {
		case err != nil && optional:
			// 기본 렌더의 PPTX 열은 도구가 있을 때만 그린다 — 웹 열만이라도 보는 것이 아무것도
			// 못 보는 것보다 낫다. 대신 조용히 빼지 않는다: 로그와 갤러리 안내가 그 사실을 말한다.
			skipped = err.Error()
			wantPPTX = false
			renders = []render.Target{render.TargetHTML}
			fmt.Fprintf(log, "PPTX 열을 뺀다 — %s\n", skipped)
		case err != nil:
			return nil, err
		default:
			raster = found
		}
	}

	projectDir := opts.Build.ProjectDir
	if projectDir == "" {
		projectDir = "."
	}
	dir := opts.Dir
	if dir == "" {
		dir = filepath.Join(projectDir, DefaultDirName)
	}
	if err := checkSeparateDirs(dir, opts.Build, projectDir); err != nil {
		return nil, err
	}

	// 웹 열이 있으면 세션을 우리가 띄우고 **지오메트리 패스에 넘긴다**: 스크린샷을 찍으려면
	// 어차피 브라우저가 필요하고, 빌드가 자기 브라우저를 따로 띄우면 이 명령이 기동 비용을 두 번
	// 낸다 (`balpyo watch`의 웜 세션과 같은 근거).
	build := opts.Build
	build.ProjectDir = projectDir
	build.Targets = renders
	var session *geom.Session
	if wantWeb {
		launched, err := geom.Launch("")
		if err != nil {
			return nil, err
		}
		defer launched.Close()
		session = launched
		build.Measurer = session
	}

	built, err := pipeline.Build(build)
	result := &Result{Dir: dir, Renders: renders, SkippedPPTX: skipped}
	if built != nil {
		result.Diagnostics = built.Diagnostics
		result.OutDir = built.OutDir
	}
	if err != nil {
		return result, err
	}
	deck := built.Deck
	size := image.Pt(deck.Stage.Width, deck.Stage.Height)

	// 래스터화는 임시 디렉토리에서 한다 — PDF와 중간 PNG는 검수 산출물이 아니다.
	var pptxPages []image.Image
	pptxPath := filepath.Join(built.OutDir, pptx.FileName(deck))
	if wantPPTX {
		workDir, err := os.MkdirTemp("", "balpyo-preview-")
		if err != nil {
			return result, fmt.Errorf("검수 임시 디렉토리를 만들 수 없다: %w", err)
		}
		defer os.RemoveAll(workDir)
		pages, noise, err := raster.Slides(pptxPath, size, workDir)
		if err != nil {
			return result, err
		}
		result.Noise = noise
		if noise != "" {
			// 경고를 실패로 바꾸지 않는다 — preview는 게이트가 아니다. 다만 **삼키지도** 않는다:
			// 엔진이 복구했다면 사용자도 파일을 열 때마다 그 경고를 본다.
			fmt.Fprintf(log, "%s가 경고를 냈다 — PPTX를 직접 열어 확인한다:\n%s\n", raster.Name(), noise)
		}
		if len(pages) != len(deck.Slides) {
			return result, fmt.Errorf("PPTX 페이지 %d장, 덱 슬라이드 %d장 — 산출물이 덱과 맞지 않는다",
				len(pages), len(deck.Slides))
		}
		pptxPages = pages
	}

	indexPath := filepath.Join(built.OutDir, web.IndexFile)
	if wantWeb {
		if err := session.Open(indexPath); err != nil {
			return result, err
		}
	}

	gallery, err := NewGallery(dir)
	if err != nil {
		return result, err
	}
	gallery.Title = galleryTitle(deck)
	engine := ""
	if raster != nil {
		engine = raster.Name()
		result.Engine = engine
	}
	gallery.Lede = galleryLede(wantWeb, wantPPTX, skipped != "", engine)
	// 묶음 이름은 디렉토리 이름이 되므로 슬러그를 쓴다 — 덱 제목에 `/`나 `:`가 있으면 갤러리가
	// 엉뚱한 곳에 파일을 쓴다 (산출 PPTX 이름을 정하는 것과 같은 규칙이다 — pptx.FileName).
	group := soml.Slug(deck.Title)
	if group == "" {
		group = soml.Slug(filepath.Base(mustAbs(projectDir)))
	}
	if group == "" {
		group = "deck"
	}

	for i, slide := range deck.Slides {
		var shots []Shot
		if wantWeb {
			shot, err := screenshot(session, i)
			if err != nil {
				return result, err
			}
			shots = append(shots, Shot{Caption: "web", Image: shot})
		}
		if wantPPTX {
			shots = append(shots, Shot{Caption: "pptx", Image: pptxPages[i]})
		}
		if err := gallery.AddSlide(group, Slide{ID: slide.ID, Index: i, Shots: shots}); err != nil {
			return result, err
		}
		result.Slides = append(result.Slides, SlideReport{Index: i, ID: slide.ID})
	}

	// 산출물도 갤러리 옆에 둔다 — 갤러리로 판정할 수 없는 확인이 있다: PPTX를 PowerPoint/
	// Keynote로 열어 텍스트 박스가 진짜 텍스트 박스인지, 테마 색 편집이 먹는지 보는 일.
	if wantPPTX {
		if err := gallery.AddArtifact(group, pptxPath); err != nil {
			return result, err
		}
	}
	if wantWeb {
		if err := gallery.AddArtifact(group, indexPath); err != nil {
			return result, err
		}
	}

	written, err := gallery.Write()
	if err != nil {
		return result, err
	}
	result.Index = written
	return result, nil
}

// checkSeparateDirs는 갤러리와 빌드 출력이 겹치지 않는지 빌드 **전에** 본다.
//
// 갤러리는 만들 때 디렉토리를 통째로 비운다 (NewGallery). 출력 디렉토리가 그와 같거나 그 안에
// 있으면 방금 쓴 PPTX와 웹 문서가 지워진 뒤 래스터화가 "파일이 없다"로 실패한다 — 원인이 경로
// 겹침이라는 사실은 어디에도 나오지 않는다. 갤러리가 출력 안에 있는 반대 방향도 막는다: 출력의
// 하위 디렉토리(그림 사본 같은)를 갤러리가 지울 수 있고, 두 산출물이 섞여 무엇이 무엇인지
// 읽기 어려워진다.
func checkSeparateDirs(galleryDir string, build pipeline.Options, projectDir string) error {
	build.ProjectDir = projectDir
	outDir, err := pipeline.OutDirPath(build)
	if err != nil {
		return err
	}
	gallery, err := filepath.Abs(galleryDir)
	if err != nil {
		return fmt.Errorf("갤러리 디렉토리 경로를 정할 수 없다: %w", err)
	}
	if within(outDir, gallery) || within(gallery, outDir) {
		return fmt.Errorf("검수 갤러리 디렉토리 %s가 빌드 출력 디렉토리 %s와 겹친다 — 갤러리는 만들 때 "+
			"디렉토리를 비우므로 빌드 산출물이 지워진다. --gallery나 --out을 서로 겹치지 않는 곳으로 준다",
			gallery, outDir)
	}
	return nil
}

// within은 path가 dir 자신이거나 그 아래인지 본다 (둘 다 절대 경로).
func within(path, dir string) bool {
	rel, err := filepath.Rel(dir, path)
	if err != nil {
		return false
	}
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)))
}

// screenshot은 슬라이드 하나를 찍는다.
//
// 선택자는 렌더러가 소유한다 (web.SlideIndexSelector) — 클래스 이름을 여기에 손으로 적으면 이름을
// 바꾼 순간 "요소가 없다"로 죽거나, 더 나쁘게는 다른 요소를 찍는다.
func screenshot(session *geom.Session, index int) (image.Image, error) {
	selector := web.SlideIndexSelector(index)
	rects, err := session.Rects(selector)
	if err != nil {
		return nil, err
	}
	if len(rects) == 0 {
		return nil, fmt.Errorf("%s에 맞는 요소가 없다 — 웹 산출물이 덱과 맞지 않는다", selector)
	}
	buf, err := session.Clip(rects[0])
	if err != nil {
		return nil, fmt.Errorf("%s: %w", selector, err)
	}
	return DecodePNG(buf)
}

// galleryTitle은 갤러리 문서 제목이다 — 덱 이름이 보이면 탭을 여러 개 열어 두고 비교할 수 있다.
func galleryTitle(deck *soml.Deck) string {
	if deck.Title == "" {
		return "balpyo 검수"
	}
	return "balpyo 검수 — " + deck.Title
}

// galleryLede는 열이 무엇인지 말하는 한 줄이다. 열이 하나뿐인 갤러리에 "왼쪽이 웹"이라고 쓰면
// 사람이 보고 있는 것을 잘못 읽는다. PPTX 열은 어느 엔진이 그렸는지 밝힌다 — 엔진마다 제 버릇이
// 있어 그림을 읽는 사람이 그것을 알아야 한다.
func galleryLede(hasWeb, hasPPTX, skippedPPTX bool, engine string) string {
	switch {
	case hasWeb && hasPPTX:
		return "왼쪽이 웹 스크린샷, 오른쪽이 " + engine + "로 되돌린 PPTX다. 두 타깃은 컴포넌트와 레이아웃이 " +
			"같고 줄바꿈 · 글리프는 각 엔진의 몫이다 — 판정은 `go test`가 하고, 여기는 눈으로 보는 자리다."
	case skippedPPTX:
		return "웹 스크린샷이다. PPTX를 그림으로 되돌릴 엔진(OnlyOffice, 또는 LibreOffice와 pdftoppm)이 없어 " +
			"PPTX 열을 뺐다 — 설치하면 나란히 보인다."
	case hasPPTX:
		return engine + "로 되돌린 PPTX다. 웹과 나란히 보려면 --target에 html을 넣는다."
	default:
		return "웹 스크린샷이다. PPTX와 나란히 보려면 --target에 pptx를 넣는다 (OnlyOffice나 LibreOffice가 필요하다)."
	}
}

// mustAbs는 절대 경로를 구한다. 실패하면 준 값을 그대로 쓴다 — 여기서 쓰이는 곳은 갤러리
// 묶음 이름뿐이고, 그 이름 때문에 검수가 멈추는 것은 균형이 맞지 않는다.
func mustAbs(path string) string {
	abs, err := filepath.Abs(path)
	if err != nil {
		return path
	}
	return abs
}
