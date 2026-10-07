// 지오메트리 패스 배선 테스트. 브라우저가 **언제 뜨는지**가 검증 대상이므로 패키지 밖
// 테스트로 두고(testkit이 브라우저 정책을 소유한다), 브라우저가 필요 없는 경우는 실행 파일
// 경로를 없는 파일로 가리켜 "떴다면 실패했을" 상태로 만들어 확인한다.
package pipeline_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/k2sebeom/balpyo/internal/pipeline"
	"github.com/k2sebeom/balpyo/internal/testkit"
	"github.com/k2sebeom/balpyo/render"
	"github.com/k2sebeom/balpyo/render/geom"
	"github.com/k2sebeom/balpyo/render/web"
	"github.com/k2sebeom/balpyo/soml"
)

// project는 파일 맵으로 프로젝트 디렉토리를 만든다.
func project(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for name, body := range files {
		dest := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(dest, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

const textSlide = `<slide id="hero">
  <text role="title">2026 아키텍처 리뷰</text>
  <text>상반기 회고와 하반기 계획</text>
</slide>`

// noBrowser는 브라우저를 띄우려는 순간 실패하게 만든다.
func noBrowser(t *testing.T) {
	t.Helper()
	t.Setenv(geom.EnvChromium, filepath.Join(t.TempDir(), "없는-크로미움"))
}

// html 타깃만 빌드할 때는 브라우저를 띄우지 않는다 — 웹 렌더에 좌표가 필요하지 않고,
// 브라우저 기동은 빌드 시간의 가장 큰 고정 비용이다 (~0.6초, PRD §9.3).
func TestBuildHTMLOnlySkipsGeometryPass(t *testing.T) {
	noBrowser(t)
	dir := project(t, map[string]string{"slides/001_title.slide": textSlide})
	result, err := pipeline.Build(pipeline.Options{
		ProjectDir: dir,
		Targets:    []render.Target{render.TargetHTML},
	})
	if err != nil {
		t.Fatalf("html 전용 빌드가 브라우저를 요구했다: %v", err)
	}
	if result.Geometry != nil {
		t.Errorf("측정하지 않은 빌드의 Geometry = %+v, want nil", result.Geometry)
	}
}

// 반대 방향: pptx 타깃은 좌표 없이 조립할 수 없으므로 브라우저가 없으면 멈춘다. 좌표가 없는
// 채로 도형 없는 파일을 내면 사용자는 PowerPoint에서 열어 보고서야 안다.
func TestBuildPPTXFailsWithoutBrowser(t *testing.T) {
	noBrowser(t)
	dir := project(t, map[string]string{"slides/001_title.slide": textSlide})
	_, err := pipeline.Build(pipeline.Options{
		ProjectDir: dir,
		Targets:    []render.Target{render.TargetHTML, render.TargetPPTX},
	})
	if err == nil {
		t.Fatal("브라우저 없이 pptx가 빌드되었다")
	}
	if !strings.Contains(err.Error(), "지오메트리") {
		t.Errorf("에러가 지오메트리 패스를 가리키지 않는다: %v", err)
	}
}

// 콘텐츠가 없는 덱은 잴 것이 없다 — 빈 슬라이드만 있는 프로젝트는 브라우저 없이도 pptx가 나온다.
func TestBuildEmptyDeckNeedsNoBrowser(t *testing.T) {
	noBrowser(t)
	dir := project(t, map[string]string{"slides/001_blank.slide": "<slide></slide>"})
	result, err := pipeline.Build(pipeline.Options{
		ProjectDir: dir,
		Targets:    []render.Target{render.TargetHTML, render.TargetPPTX},
	})
	if err != nil {
		t.Fatalf("빈 덱이 브라우저를 요구했다: %v", err)
	}
	if result.Geometry != nil {
		t.Errorf("잴 것이 없는 빌드의 Geometry = %+v, want nil", result.Geometry)
	}
}

// recordingMeasurer는 지오메트리 패스를 대신 받는다 — 브라우저 없이 배선만 본다.
type recordingMeasurer struct {
	calls int
	path  string
	body  []byte
	err   error
}

func (m *recordingMeasurer) Measure(deck *soml.Deck, htmlPath string) (*geom.Geometry, error) {
	m.calls++
	m.path = htmlPath
	m.body, _ = os.ReadFile(htmlPath)
	return nil, m.err
}

// 호출자가 준 측정기가 있으면 빌드는 그것을 쓴다 — `balpyo watch`가 브라우저 하나를 **웜으로**
// 재사용하는 길이다 (ADR 0046). 기동이 저작 루프의 가장 큰 비용이므로 루프마다 다시 띄우지
// 않는 것이 이 옵션의 전부이고, 측정기를 만들고 닫는 일은 루프가 소유한다.
//
// 측정기에 넘어가는 HTML은 **이 빌드가 낼 산출물 그대로**여야 한다 (그러지 않으면 잰 좌표가
// 다른 페이지의 것이다) — 그 사실도 여기서 함께 못 박는다.
func TestBuildUsesTheInjectedMeasurer(t *testing.T) {
	noBrowser(t)
	measurer := &recordingMeasurer{err: errors.New("잰 척했다")}
	dir := project(t, map[string]string{"slides/001_title.slide": textSlide})
	_, err := pipeline.Build(pipeline.Options{
		ProjectDir: dir,
		Targets:    []render.Target{render.TargetHTML, render.TargetPPTX},
		Measurer:   measurer,
	})
	if err == nil || !strings.Contains(err.Error(), "잰 척했다") {
		t.Fatalf("주입한 측정기가 쓰이지 않았다: %v", err)
	}
	if measurer.calls != 1 {
		t.Errorf("측정기 호출 %d회, want 1회", measurer.calls)
	}
	if filepath.Base(measurer.path) != web.IndexFile {
		t.Errorf("측정 대상 = %s, want %s", measurer.path, web.IndexFile)
	}
	if !strings.Contains(string(measurer.body), "2026 아키텍처 리뷰") {
		t.Errorf("측정기가 받은 HTML이 이 덱의 산출물이 아니다:\n%s", measurer.body)
	}
}

// overflowProject는 배치 사고 3종을 담은 프로젝트다 (TASKS.md P1.1의 RED 목록).
//
// 세 사고를 슬라이드 하나에 섞지 않는 이유: 요소 하나가 받는 이탈 진단은 하나이므로(ADR 0039)
// 섞으면 어느 규칙이 말했는지가 흐려진다. 넘친 양을 **선언한 px로** 정할 수 있게 도형 크기를
// 쓴 것도 같은 이유다 — 폰트 줄바꿈에 기대면 판정이 아니라 폰트를 시험하게 된다.
func overflowProject(t *testing.T) string {
	t.Helper()
	return project(t, map[string]string{
		"balpyo.json": `{"title": "overflow"}`,
		// (1) 콘텐츠 박스 이탈: 1200px 도형은 1152px 콘텐츠 폭을 48px 넘친다.
		"slides/001_wide.slide": `<slide id="wide">
  <shape type="rect" fill="surface" w="1200" h="100"></shape>
</slide>`,
		// (2) 작은 도형 안 텍스트: 300×80 도형에 다섯 줄을 담는다. 줄은 `<br/>`가 정한다 —
		// 문장이 몇 줄로 접히는지는 글꼴의 몫이라(ADR 0064) 그것에 기대면 측정 호스트의
		// 글꼴에 따라 경고가 나오기도 하고 안 나오기도 한다 (CJK 글꼴이 없는 CI 러너).
		"slides/002_small_shape.slide": `<slide id="small-shape">
  <shape type="rect" fill="surface" w="300" h="80">
    <p>하나<br/>둘<br/>셋<br/>넷<br/>다섯</p>
  </shape>
</slide>`,
		// (3) 스테이지 이탈: canvas 좌표로 스테이지 오른쪽 밖에 둔다.
		"slides/003_offstage.slide": `<slide id="offstage" layout="canvas">
  <shape type="rect" fill="surface" x="1200" y="64" w="200" h="100"></shape>
</slide>`,
	})
}

// `--check-layout`은 좌표를 쓰는 타깃이 없어도 지오메트리 패스를 켠다 — 저작 루프(html만)가
// 배치 경고를 받는 길이다 (ADR 0039). 경고이므로 빌드는 성공하고 산출물도 나온다.
func TestBuildCheckLayoutWarnsOnOverflow(t *testing.T) {
	testkit.RequireChromium(t)
	result, err := pipeline.Build(pipeline.Options{
		ProjectDir:  overflowProject(t),
		Targets:     []render.Target{render.TargetHTML},
		CheckLayout: true,
	})
	if err != nil {
		t.Fatalf("경고가 빌드를 멈췄다: %v", err)
	}
	if result.Geometry == nil {
		t.Fatal("--check-layout 빌드에 지오메트리가 없다")
	}
	if len(result.Written) == 0 {
		t.Error("경고만 있는 빌드가 산출물을 쓰지 않았다")
	}
	want := []string{
		"<shape>가 콘텐츠 박스를 오른쪽으로 48px 넘친다",
		"<p>가 부모 <shape>를 아래로",
		"<shape>가 스테이지를 오른쪽으로",
	}
	if len(result.Diagnostics) != len(want) {
		t.Fatalf("진단 %d건, want %d건:\n%s", len(result.Diagnostics), len(want), result.Diagnostics)
	}
	for i, fragment := range want {
		if !strings.Contains(result.Diagnostics[i].Message, fragment) {
			t.Errorf("%d번째 진단 = %q, want %q 포함", i, result.Diagnostics[i].Message, fragment)
		}
	}
}

// `--strict`는 배치 경고도 에러로 승격한다 — CI가 넘친 덱을 통과시키지 않는 길이다. 승격된
// 에러는 산출물을 **쓰기 전에** 멈춘다: 반쯤 갱신된 출력 디렉토리가 남으면 다음 빌드가
// 무엇이 새 산출물인지 말해 주지 못한다.
func TestBuildStrictPromotesLayoutWarnings(t *testing.T) {
	testkit.RequireChromium(t)
	dir := overflowProject(t)
	outDir := filepath.Join(t.TempDir(), "dist")
	result, err := pipeline.Build(pipeline.Options{
		ProjectDir:  dir,
		Targets:     []render.Target{render.TargetHTML},
		OutDir:      outDir,
		CheckLayout: true,
		Strict:      true,
	})
	if err == nil {
		t.Fatal("--strict가 배치 경고를 통과시켰다")
	}
	if !result.Diagnostics.HasErrors() {
		t.Errorf("승격되지 않은 진단: %s", result.Diagnostics)
	}
	if len(result.Written) != 0 {
		t.Errorf("승격된 에러에도 산출물을 썼다: %v", result.Written)
	}
	if entries, err := os.ReadDir(outDir); err == nil && len(entries) > 0 {
		t.Errorf("출력 디렉토리에 파일이 남았다: %d개", len(entries))
	}
}

// 반대 방향: 제자리에 있는 덱은 `--check-layout`에서도 조용하다. 정상 덱이 경고를 받으면
// 저자는 경고를 읽지 않게 되고, 그것이 진단을 잃는 방식이다.
func TestBuildCheckLayoutIsQuietOnAFittingDeck(t *testing.T) {
	testkit.RequireChromium(t)
	dir := project(t, map[string]string{"slides/001_title.slide": textSlide})
	result, err := pipeline.Build(pipeline.Options{
		ProjectDir:  dir,
		Targets:     []render.Target{render.TargetHTML},
		CheckLayout: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Diagnostics) > 0 {
		t.Errorf("정상 덱의 진단 %d건, want 0건:\n%s", len(result.Diagnostics), result.Diagnostics)
	}
}

// 두 타깃을 함께 빌드하면 웹 렌더 → 지오메트리 → PPTX 순으로 돌고, 측정 결과가 결과에 남는다
// (Iteration 1 리뷰와 다음 iteration의 좌표 스냅샷이 이 값을 본다).
func TestBuildPPTXRunsGeometryPass(t *testing.T) {
	testkit.RequireChromium(t)
	dir := project(t, map[string]string{
		"balpyo.json":            `{"title": "geometry"}`,
		"slides/001_title.slide": textSlide,
	})
	result, err := pipeline.Build(pipeline.Options{
		ProjectDir: dir,
		Targets:    []render.Target{render.TargetHTML, render.TargetPPTX},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Geometry == nil {
		t.Fatal("pptx 빌드에 지오메트리가 없다")
	}
	if got, want := len(result.Geometry.Boxes), len(geom.Refs(result.Deck)); got != want {
		t.Errorf("박스 %d개, 덱 요소 %d개", got, want)
	}
	if err := result.Geometry.Verify(result.Deck); err != nil {
		t.Errorf("빌드가 낸 지오메트리가 덱과 어긋난다: %v", err)
	}
	if result.Geometry.Stage != result.Deck.Stage {
		t.Errorf("Stage = %+v, want %+v", result.Geometry.Stage, result.Deck.Stage)
	}
}
