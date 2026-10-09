// 이 파일만 외부 테스트 패키지(preview_test)인 이유: 도구가 없을 때 건너뛸지 실패할지는
// **테스트 정책**이고 그 정책의 주인은 internal/testkit이다 (BALPYO_TEST_STRICT). 같은 판단을
// 여기에 다시 적으면 도구가 없을 때의 약속이 두 벌이 된다.
// 외부 테스트 패키지는 testkit을 임포트할 수 있다 — testkit이 preview를 임포트해도 순환이
// 아니다.
package preview_test

import (
	"image"
	"os"
	"path/filepath"
	"testing"

	"github.com/k2sebeom/balpyo/internal/pipeline"
	"github.com/k2sebeom/balpyo/internal/preview"
	"github.com/k2sebeom/balpyo/internal/testkit"
	"github.com/k2sebeom/balpyo/render"
	"github.com/k2sebeom/balpyo/render/pptx"
)

// twoSlideProject는 슬라이드 두 장인 최소 프로젝트다.
//
// 픽스처 코퍼스를 쓰지 않는 이유: 여기서 보는 것은 래스터화(장수·크기·결정성)이고, 그 판정은
// 픽스처의 내용이 바뀌어도 그대로여야 한다. 장수를 두 장으로 두면 "한 장만 나왔다"와 "순서가
// 뒤집혔다"가 모두 잡힌다.
func twoSlideProject(t testing.TB) string {
	t.Helper()
	dir := t.TempDir()
	write := func(rel, body string) {
		t.Helper()
		path := filepath.Join(dir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("balpyo.json", `{"title": "raster deck", "lang": "ko", "aspect": "16:9"}`+"\n")
	write("slides/001_title.slide", `<slide id="title">
  <text role="title" align="center">래스터 검수</text>
  <text role="body" align="center">첫 장</text>
</slide>
`)
	write("slides/002_body.slide", `<slide id="body">
  <text role="heading">두 번째 장</text>
  <text role="body">PPTX를 그림으로 되돌려 웹과 나란히 본다.</text>
</slide>
`)
	return dir
}

// buildPPTX는 프로젝트를 빌드하고 pptx 경로와 스테이지 크기를 돌려준다.
func buildPPTX(t testing.TB, projectDir string) (pptxPath string, size image.Point, slides int) {
	t.Helper()
	testkit.RequireChromium(t)
	result, err := pipeline.Build(pipeline.Options{
		ProjectDir: projectDir,
		Targets:    []render.Target{render.TargetHTML, render.TargetPPTX},
		OutDir:     t.TempDir(),
	})
	if err != nil {
		t.Fatalf("빌드 실패: %v", err)
	}
	stage := result.Deck.Stage
	return filepath.Join(result.OutDir, pptx.FileName(result.Deck)),
		image.Pt(stage.Width, stage.Height), len(result.Deck.Slides)
}

// TestRasterizerSlidesRenderEveryPageAtStageSize는 PPTX → PNG의 **산출물·장수·크기**를 못 박는다.
//
// 크기를 따로 보는 이유: 갤러리는 웹 스크린샷과 같은 크기로 나란히 놓고, DPI로 맞추면 페이지 크기
// 반올림 때문에 한두 px씩 어긋난다.
// engines는 래스터 엔진마다의 생성자다 — 같은 판정을 두 엔진이 받는다. 생성자가 도구 정책
// (없으면 건너뛰기)을 지므로 한 엔진만 깔린 호스트에서도 다른 엔진의 판정은 돈다.
var engines = []struct {
	name string
	make func(t testing.TB) preview.Rasterizer
}{
	{"OnlyOffice", func(t testing.TB) preview.Rasterizer {
		return &preview.OnlyOffice{Dir: testkit.RequireOnlyOffice(t)}
	}},
	{"LibreOffice", func(t testing.TB) preview.Rasterizer {
		return &preview.LibreOffice{
			Soffice:  testkit.RequireSoffice(t),
			Pdftoppm: testkit.RequirePdftoppm(t),
		}
	}},
}

func TestRasterizerSlidesRenderEveryPageAtStageSize(t *testing.T) {
	for _, engine := range engines {
		t.Run(engine.name, func(t *testing.T) {
			raster := engine.make(t)
			pptxPath, size, slides := buildPPTX(t, twoSlideProject(t))
			slidesRenderEveryPageAtStageSize(t, raster, pptxPath, size, slides)
		})
	}
}

func slidesRenderEveryPageAtStageSize(t *testing.T, raster preview.Rasterizer, pptxPath string, size image.Point, slides int) {

	pages, noise, err := raster.Slides(pptxPath, size, t.TempDir())
	if err != nil {
		t.Fatalf("래스터화 실패: %v", err)
	}
	if noise != "" {
		t.Errorf("%s가 경고·복구를 냈다 — 사용자도 파일을 열 때마다 본다:\n%s", raster.Name(), noise)
	}
	if len(pages) != slides {
		t.Fatalf("PNG %d장, 덱 슬라이드 %d장", len(pages), slides)
	}
	for i, page := range pages {
		if got := page.Bounds().Size(); got != size {
			t.Errorf("%d번째 페이지 크기 = %v, want %v (논리 스테이지)", i+1, got, size)
		}
	}
}

// TestRasterizerSlidesAreDeterministic은 같은 PPTX를 두 번 래스터화하면 **같은 픽셀**이 나오는지
// 본다.
//
// 이것을 못 박아야 하는 이유: 래스터화 자체가 실행마다 흔들리면 사람은 고친 것이 없는데 갤러리가
// 바뀌는 것을 보게 된다. LibreOffice가 실행마다 새 사용자 프로필을 쓰는 것(LibreOffice.PDF)과
// OnlyOffice가 실행마다 글꼴 색인을 대조하는 것(OnlyOffice.Slides)이 지켜야 하는 성질이 바로 이것이다.
//
// PDF 바이트가 아니라 PNG 픽셀을 비교한다 — PDF에는 생성 시각이 들어가 실행마다 다르다.
func TestRasterizerSlidesAreDeterministic(t *testing.T) {
	for _, engine := range engines {
		t.Run(engine.name, func(t *testing.T) {
			raster := engine.make(t)
			pptxPath, size, _ := buildPPTX(t, twoSlideProject(t))
			slidesAreDeterministic(t, raster, pptxPath, size)
		})
	}
}

func slidesAreDeterministic(t *testing.T, raster preview.Rasterizer, pptxPath string, size image.Point) {

	first, _, err := raster.Slides(pptxPath, size, t.TempDir())
	if err != nil {
		t.Fatalf("첫 래스터화 실패: %v", err)
	}
	second, _, err := raster.Slides(pptxPath, size, t.TempDir())
	if err != nil {
		t.Fatalf("두 번째 래스터화 실패: %v", err)
	}
	if len(first) != len(second) {
		t.Fatalf("장수가 다르다: %d vs %d", len(first), len(second))
	}
	// 0 오차로 본다 — 여기서는 렌더러 둘을 비교하는 것이 아니라 같은 렌더러를 두 번 돌린 것이다.
	for i := range first {
		if a, b := first[i].Bounds(), second[i].Bounds(); a != b {
			t.Fatalf("%d번째 페이지 크기가 다르다: %v vs %v", i+1, a, b)
		}
		if x, y, ok := firstPixelDifference(first[i], second[i]); !ok {
			t.Errorf("%d번째 페이지가 실행마다 다르다 — (%d, %d)부터", i+1, x, y)
		}
	}
}

// firstPixelDifference는 두 이미지가 같으면 ok, 다르면 처음 다른 픽셀의 자리를 돌려준다.
func firstPixelDifference(a, b image.Image) (x, y int, ok bool) {
	bounds := a.Bounds()
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			if preview.RGB8(a.At(x, y)) != preview.RGB8(b.At(x, y)) {
				return x, y, false
			}
		}
	}
	return 0, 0, true
}
