// 지오메트리 패스 배선 테스트. 브라우저가 **언제 뜨는지**가 검증 대상이므로 패키지 밖
// 테스트로 두고(testkit이 브라우저 정책을 소유한다), 브라우저가 필요 없는 경우는 실행 파일
// 경로를 없는 파일로 가리켜 "떴다면 실패했을" 상태로 만들어 확인한다.
package pipeline_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/k2sebeom/balpyo/internal/pipeline"
	"github.com/k2sebeom/balpyo/internal/testkit"
	"github.com/k2sebeom/balpyo/render"
	"github.com/k2sebeom/balpyo/render/geom"
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
