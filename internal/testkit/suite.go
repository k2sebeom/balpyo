package testkit

import (
	"image"
	"path/filepath"
	"testing"

	"github.com/k2sebeom/balpyo/internal/pipeline"
	"github.com/k2sebeom/balpyo/render/geom"
	"github.com/k2sebeom/balpyo/render/web"
)

// Suite는 픽스처 스위트 한 번의 실행이다 — 브라우저 세션 하나와 리뷰 갤러리를 모든
// 픽스처가 공유한다.
type Suite struct {
	// Diff는 웹 vs PPTX 픽셀 비교 허용 오차다.
	Diff DiffOptions

	// root는 스위트를 시작한 테스트다. 브라우저 정리를 여기에 걸어 세션이 픽스처
	// 서브테스트보다 오래 살게 한다.
	root    testing.TB
	browser *Browser
	review  *Review
}

// NewSuite는 스위트를 준비한다.
func NewSuite(t testing.TB) *Suite {
	t.Helper()
	return &Suite{Diff: DefaultDiffOptions(), root: t, review: NewReview(t)}
}

// Run은 모든 픽스처를 돈다: 빌드 → 골든 비교 → 웹/PPTX 렌더 비교.
//
// 픽스처를 병렬로 돌리지 않는다. 브라우저 세션과 리뷰 갤러리가 공유 상태이고, 무엇보다
// 병렬 LibreOffice 프로세스는 CI에서 메모리로 서로를 죽인다.
func (s *Suite) Run(t *testing.T) {
	t.Helper()
	for _, fixture := range Fixtures(t) {
		t.Run(fixture.Name, func(t *testing.T) {
			result := fixture.Build(t)
			fixture.AssertGolden(t, result)
			// 렌더 비교는 외부 도구가 필요하다. 별도 서브테스트로 두면 도구가 없는 로컬에서
			// 이 부분만 건너뛰고 골든 비교는 그대로 돈다.
			t.Run("render", func(t *testing.T) { s.CompareRenders(t, fixture, result) })
		})
	}
	s.review.Write(t)
}

// CompareRenders는 픽스처의 웹 스크린샷과 PPTX 렌더를 슬라이드별로 비교한다.
func (s *Suite) CompareRenders(t *testing.T, fixture Fixture, result *pipeline.Result) {
	t.Helper()
	stage := result.Deck.Stage
	size := image.Pt(stage.Width, stage.Height)

	pptxPath := fixture.Output(t, result, ".pptx")
	// LibreOffice가 깨끗하게 열고 렌더하는지 — 여기서 도구가 없으면 건너뛴다(로컬) 또는
	// 실패한다(테스트 이미지).
	shots := RenderPPTX(t, pptxPath, size)
	if len(shots) != len(result.Deck.Slides) {
		t.Fatalf("PPTX 페이지 %d장, 덱 슬라이드 %d장", len(shots), len(result.Deck.Slides))
	}

	browser := s.Browser(t)
	browser.Load(t, filepath.Join(result.OutDir, web.IndexFile))

	for i, slide := range result.Deck.Slides {
		// 선택자는 렌더러가 소유한다 — 클래스·속성 이름을 여기에 손으로 적으면 이름을 바꾼
		// 순간 스크린샷이 "요소가 없다"로 죽거나, 더 나쁘게는 다른 요소를 찍는다.
		webImg := browser.ScreenshotElement(t, web.SlideIndexSelector(i))
		diff := AssertImagesMatch(t, webImg, shots[i], s.Diff)
		ok := diff.Image != nil && diff.Ratio <= s.Diff.MaxRatio
		t.Logf("슬라이드 %d(%s): %s", i+1, slide.ID, diff)
		if diff.Image != nil {
			s.review.AddSlide(t, fixture.Name, slide.ID, i, webImg, shots[i], diff.Image, diff.String(), ok)
		}
	}
	s.review.AddArtifact(t, fixture.Name, pptxPath)
	s.review.AddArtifact(t, fixture.Name, filepath.Join(result.OutDir, web.IndexFile))
}

// Browser는 스위트가 공유하는 브라우저 세션이다. 첫 호출에서 띄운다 — 골든 비교만 도는
// 실행(도구 없는 로컬)에서 브라우저를 띄우지 않기 위해서다.
func (s *Suite) Browser(t testing.TB) *Browser {
	t.Helper()
	bin := RequireChromium(t)
	if s.browser != nil {
		return s.browser
	}
	session, err := geom.Launch(bin)
	if err != nil {
		t.Fatalf("브라우저를 띄울 수 없다: %v", err)
	}
	s.root.Cleanup(session.Close)
	s.browser = &Browser{session: session}
	return s.browser
}
