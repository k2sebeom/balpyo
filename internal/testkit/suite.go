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

// fixtureDiffOptions는 픽스처별 픽셀 비교 허용 오차 조정이다 — 없는 픽스처는 Suite.Diff
// (DefaultDiffOptions)를 쓴다. 기본값을 넘는 조정은 **구조 검사(잉크 띠 — 줄 수·위치·폭)가
// 초록일 때만, 측정 수치와 함께** 남긴다: 수치 없는 완화는 게이트 약화다.
var fixtureDiffOptions = map[string]DiffOptions{
	// 측정 최대 3.04% (슬라이드 1 — Iteration 2 항목 08). 제목 위계 슬라이드(h1 48px +
	// h2/h3 + 본문 2줄 + 인용)는 큰 글리프의 잉크 면적이 01_text의 표지보다 많고,
	// LibreOffice의 첫 기준선 +2~5px(엔진 소관 — NOTES 01/07)가 그 글리프 경계 픽셀을
	// 전부 이중으로 세게 한다. 같은 슬라이드의 잉크 띠 검사(세로 중심 ±6px, 줄 수 0차)는
	// 전부 허용 안이므로 구성은 구조 갈라짐이 아니라 잔차다. 3.5%는 측정값에 01_text와
	// 같은 폭의 여유를 더한 값이다 (2.1% 측정 → 2.5% 허용과 같은 태도).
	"02_text_subset": {Channel: 32, MaxRatio: 0.035},
	// 측정 최대 2.68% (슬라이드 4 004_child_attrs — Iteration 3 항목 05). 배치가 열리면서
	// 한 슬라이드에 텍스트 박스가 9개까지 앉고(세로 스택 시절은 3~4개), LibreOffice의 첫
	// 기준선 +2~5px가 그 박스마다 글리프 경계를 이중으로 세게 한다 — 잔차가 박스 수에
	// 비례하는 것이다. 같은 슬라이드의 잉크 띠 검사(줄 수 0차, 세로 중심 +2~5px)는 전부
	// 허용 안이므로 구조 갈라짐이 아니다. 3.2%는 측정값에 앞 두 픽스처와 같은 폭의 여유를
	// 더한 값이다 (2.1% → 2.5%, 3.04% → 3.5%와 같은 태도).
	"03_layout": {Channel: 32, MaxRatio: 0.032},
	// 측정 최대 3.18% (슬라이드 5 005_text_anchor — Iteration 4 항목 06). 도형이 들어오면
	// 잔차의 성질이 하나 늘어난다: 모든 도형에 **외곽선이 하나씩** 생기고(컴포넌트 기본값이
	// `border: 1 solid muted`다) 그 선은 양 타깃 모두 경로 중앙에 그려지므로, 굵기 1px 선의
	// 반 픽셀 반올림이 두 래스터라이저에서 다르게 떨어지면 **둘레 전체**가 다른 픽셀로 세어진다
	// (220×110 도형 하나의 둘레가 660px = 슬라이드의 0.07%이고, 도형이 스무 개면 1.4%다).
	// 기하 자체는 이 비율이 아니라 도형별 크롭 비교가 판정한다
	// (TestShapeFixturePresetGeometryPerShape — 12종 전수, 측정 최대 7.95%), 줄 수 갈라짐은
	// 도형 안 줄 세기가 따로 잡는다 (TestShapeFixtureInShapeTextLineCounts). 그래서 이 비율은
	// 잔차만 남은 수치다. 3.7%는 측정값에 앞 픽스처들과 같은 폭의 여유를 더한 값이다.
	"04_shape": {Channel: 32, MaxRatio: 0.037},
}

// slideDiffOptions는 **슬라이드 한 장**의 픽셀 비교 허용 오차다 (`<픽스처>/<슬라이드 id>`).
// 픽스처 전체를 완화하는 것과 갈라 두는 이유: 아래 슬라이드의 갈라짐은 잔차가 아니라 **저자가
// 선언한 것**이라, 같은 수치를 픽스처 전체에 주면 나머지 슬라이드의 실제 배치 갈라짐이 그
// 그늘에 숨는다.
var slideDiffOptions = map[string]DiffOptions{
	// 007_style_appearance: `style`에 외관을 쓴 슬라이드다. `background: #0F65F9`가 웹에만
	// 적용되고 PPTX에는 이식되지 않는 것이 §7.9의 계약 그 자체이므로, 두 렌더가 이 슬라이드
	// 에서 다른 것이 **정답**이다 — 저자가 칠한 띠(1152×40px = 슬라이드의 5.0%)가 픽셀
	// 차이의 대부분이고 남은 1.1%가 글리프 잔차다 (측정 6.11%, Iteration 3 항목 05).
	// 7%는 측정값에 여유를 더한 값이다. 배치가 깨지면(박스가 다른 자리에 앉으면) 이 수치도
	// 넘으므로 검사가 무력해지지는 않고, 좌표 자체는 0 오차로 따로 못 박혀 있다
	// (TestFixtureCoordinatesMatchGeometry).
	"03_layout/style-appearance-warning": {Channel: 32, MaxRatio: 0.07},
	// 006_cards: 도형이 가장 많은 슬라이드다 (제목 + 카드 8 + 배너 1 = 도형 10개, 그중 아홉이
	// 외곽선을 갖는다) — 위의 둘레 잔차가 여기서 가장 크게 쌓인다. 더해서 첫 카드 본문은
	// **알려진 줄바꿈 규칙 갈라짐**의 자리라(inShapeLineDivergences — 웹 2줄, LibreOffice 3줄)
	// 그 문단의 글자가 통째로 다른 자리에 있다. 측정 4.62%(Iteration 4 항목 06)이고 5.2%는
	// 거기에 여유를 더한 값이다. 갈라짐의 **범위**는 픽셀 비율이 아니라 줄 수 표가 단정하므로
	// (표가 바뀌면 그 검사가 깨진다) 이 수치가 갈라짐을 덮어 감추지는 않는다.
	"04_shape/shape-cards": {Channel: 32, MaxRatio: 0.052},
	// 06_stack/in-container: 겹침 픽스처에서 잔차가 가장 많이 쌓이는 슬라이드다 — 그림 카드
	// 3장(dot.png `cover` 리샘플링 잔차)에 흰(bg) 라벨 글자가 각 카드 위에 얹히고, 아래
	// `<row>`에는 외곽선 있는 도형 + 문단이 있다. 겹침이 새 잔차원을 만들지는 않지만(그림·
	// 글자·테두리의 잔차가 겹친 자리에 겹쳐 나타날 뿐이다) 그 세 종류가 한 슬라이드에 모두
	// 있어 합이 커진다. 측정 3.128%(항목 06)이고 3.7%는 거기에 앞 픽스처들과 같은 폭의
	// 여유를 더한 값이다 (04_shape 3.18%→3.7%와 같은 태도). 겹침 순서 자체는 이 비율이 아니라
	// 겹친 영역 크롭이 판정하므로(TestStackFixtureOverlapRegions) 이 수치가 순서 갈라짐을
	// 덮어 감추지 않는다 — 위아래가 뒤집히면 겹친 면적만큼(수십 %) 이 수치도 함께 넘는다.
	"06_stack/in-container": {Channel: 32, MaxRatio: 0.037},
	// 06_stack/content-explicit: 겹침이 **없는** 순수 텍스트 슬라이드다 (h1 + 문단 + 리스트).
	// 큰 제목 글리프(h1)의 잉크 면적이 많고 LibreOffice의 첫 기준선 +2~5px(엔진 소관 —
	// NOTES 01/07)가 그 글리프 경계 픽셀을 이중으로 세게 한다 — 02_text_subset의 제목 위계
	// 슬라이드(측정 3.04%)와 같은 잔차다. 측정 2.514%(항목 06, 기본 2.5%를 근소하게 넘는다)
	// 이고 3.0%는 거기에 여유를 더한 값이다. 이 슬라이드는 잉크 띠 줄바꿈 판정에도 남아 있어
	// (inkBandSkipSlides에 없다) 줄 수·위치는 0 오차로 따로 못 박힌다 — 이 픽셀 여유가 배치
	// 갈라짐을 감추지 않는다.
	"06_stack/content-explicit": {Channel: 32, MaxRatio: 0.030},
	// 07_gradient/fill-gradient: 그라디언트 채움 슬라이드다 (Iteration 8). 잔차의 대부분은
	// **3색 스톱 카드**에서 온다 — 산출 OOXML은 스펙대로 3개 `a:gs`(accent1 pos0 → tx2
	// pos50000 → bg2 pos100000, 위 골든 slide1.xml에서 확인)이지만, 테스트 래스터라이저인
	// LibreOffice는 다중 스톱 gradFill을 근사해 그려 중간·끝 스톱이 웹(SVG linearGradient)과
	// 갈라진다. PowerPoint/Keynote는 세 스톱을 그대로 그리므로 이 갈라짐은 산출물 결함이
	// 아니라 **뷰어 소관**이다 (04_shape의 "알려진 줄바꿈 규칙 갈라짐"과 같은 성질). 나머지는
	// 그라디언트 경계의 색 보간·안티앨리어싱 잔차다. 측정 3.265%(대각 그라디언트는 정사각형
	// 도형으로 두어 objectBoundingBox↔OOXML 각도 왜곡을 이미 제거했다)이고 3.8%는 거기에
	// 앞 픽스처들과 같은 폭의 여유를 더한 값이다. **사람 확인 필수**: deck.pptx를 PowerPoint/
	// Keynote로 열어 3색 스톱과 각도가 웹과 같은지 본다 (LibreOffice 렌더로는 판정할 수 없는
	// 유일한 자리다). 채움 색·스톱 참조 자체는 구조 테스트(render/pptx.TestRenderShapeFillGradient,
	// render/web.TestShapeFillGradientEmitsSVGDef)가 0 오차로 못 박으므로 이 픽셀 여유가
	// 스톱 갈라짐을 덮어 감추지 않는다.
	"07_gradient/fill-gradient": {Channel: 32, MaxRatio: 0.038},
}

// CompareRenders는 픽스처의 웹 스크린샷과 PPTX 렌더를 슬라이드별로 비교한다.
func (s *Suite) CompareRenders(t *testing.T, fixture Fixture, result *pipeline.Result) {
	t.Helper()
	diff := s.Diff
	if override, ok := fixtureDiffOptions[fixture.Name]; ok {
		diff = override
	}
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
		// 슬라이드 한 장의 조정이 픽스처 조정을 덮는다 (더 좁은 범위가 이긴다).
		slideDiff := diff
		if override, ok := slideDiffOptions[fixture.Name+"/"+slide.ID]; ok {
			slideDiff = override
		}
		// 선택자는 렌더러가 소유한다 — 클래스·속성 이름을 여기에 손으로 적으면 이름을 바꾼
		// 순간 스크린샷이 "요소가 없다"로 죽거나, 더 나쁘게는 다른 요소를 찍는다.
		webImg := browser.ScreenshotElement(t, web.SlideIndexSelector(i))
		cmp := AssertImagesMatch(t, webImg, shots[i], slideDiff)
		ok := cmp.Image != nil && cmp.Ratio <= slideDiff.MaxRatio
		t.Logf("슬라이드 %d(%s): %s", i+1, slide.ID, cmp)
		if cmp.Image != nil {
			s.review.AddSlide(t, fixture.Name, slide.ID, i, webImg, shots[i], cmp.Image, cmp.String(), ok)
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
