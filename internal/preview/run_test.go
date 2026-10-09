package preview_test

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/k2sebeom/balpyo/internal/pipeline"
	"github.com/k2sebeom/balpyo/internal/preview"
	"github.com/k2sebeom/balpyo/internal/testkit"
	"github.com/k2sebeom/balpyo/render"
)

// TestRunStopsBeforeBuildingWhenToolsAreMissing은 도구가 없으면 **빌드 전에** 멈추는지 본다.
//
// 도구가 있는 환경에서도 도는 테스트다 (환경 변수로 없는 경로를 준다) — 이 판단은 도구 유무가
// 아니라 순서에 대한 것이기 때문이다. 여섯 장을 빌드하고 브라우저까지 띄운 뒤에 "soffice가
// 없다"를 알려 주면 사람의 시간을 버린다. pptx를 **명시한** 검수는 웹 열만으로 물러서지 않는다 —
// 그 사람은 PPTX를 보려고 명령을 쳤다. 기본 렌더의 물러서기는
// TestRunDefaultRendersFallBackToWebWithoutTools가 본다.
func TestRunStopsBeforeBuildingWhenToolsAreMissing(t *testing.T) {
	project := twoSlideProject(t)
	t.Setenv(preview.EnvSoffice, filepath.Join(t.TempDir(), "없는-soffice"))

	dir := filepath.Join(project, "preview")
	_, err := preview.Run(preview.Options{
		Build:   pipeline.Options{ProjectDir: project},
		Renders: []render.Target{render.TargetPPTX},
		Dir:     dir,
	})
	if err == nil {
		t.Fatal("soffice가 없으면 에러여야 한다")
	}
	if !strings.Contains(err.Error(), preview.EnvSoffice) {
		t.Errorf("경로를 주는 방법이 메시지에 없다: %v", err)
	}
	// 빌드도, 갤러리도 시작하지 않았다.
	if _, statErr := os.Stat(dir); !os.IsNotExist(statErr) {
		t.Errorf("도구가 없는데 갤러리 디렉토리를 만들었다: %v", statErr)
	}
	if _, statErr := os.Stat(filepath.Join(project, "dist")); !os.IsNotExist(statErr) {
		t.Error("도구가 없는데 빌드를 시작했다")
	}
}

// TestRunRejectsUnknownRenderTarget은 검수할 수 없는 타깃을 먼저 거른다.
func TestRunRejectsUnknownRenderTarget(t *testing.T) {
	_, err := preview.Run(preview.Options{
		Build:   pipeline.Options{ProjectDir: twoSlideProject(t)},
		Renders: []render.Target{render.Target("pdf")},
	})
	if err == nil || !strings.Contains(err.Error(), "pdf") {
		t.Fatalf("알 수 없는 타깃은 에러여야 한다: %v", err)
	}
}

// TestRunBuildsGalleryWithBothRenders는 갤러리 한 장에 웹·PPTX가 슬라이드마다 들어가는지
// 본다.
func TestRunBuildsGalleryWithBothRenders(t *testing.T) {
	testkit.RequireChromium(t)
	testkit.RequireSoffice(t)
	testkit.RequirePdftoppm(t)

	project := twoSlideProject(t)
	result, err := preview.Run(preview.Options{
		Build:   pipeline.Options{ProjectDir: project, OutDir: t.TempDir()},
		Renders: preview.DefaultRenders(),
		Dir:     filepath.Join(project, "preview"),
	})
	if err != nil {
		t.Fatalf("검수 실패: %v", err)
	}
	if len(result.Slides) != 2 {
		t.Fatalf("슬라이드 %d장, want 2", len(result.Slides))
	}
	if result.Noise != "" {
		t.Errorf("LibreOffice가 경고·복구를 냈다:\n%s", result.Noise)
	}

	index, err := os.ReadFile(result.Index)
	if err != nil {
		t.Fatalf("갤러리를 읽을 수 없다: %v", err)
	}
	for _, want := range []string{"web", "pptx", "raster-deck", "title", "body"} {
		if !strings.Contains(string(index), want) {
			t.Errorf("갤러리에 %q가 없다", want)
		}
	}
	// 슬라이드마다 두 칸이 파일로 남는다 — 문서만 있고 그림이 없으면 검수가 불가능하다.
	for i := range result.Slides {
		for _, caption := range []string{"web", "pptx"} {
			path := filepath.Join(result.Dir, "raster-deck", fmtName(i, caption))
			if _, err := os.Stat(path); err != nil {
				t.Errorf("%s가 없다: %v", path, err)
			}
		}
	}
	if _, err := os.Stat(filepath.Join(result.Dir, "raster-deck", fmtName(0, "diff"))); err == nil {
		t.Error("diff 칸이 남았다 — 검수는 판정하지 않는다")
	}
	// 산출물도 손에 잡히는 곳에 있어야 한다 (PowerPoint/Keynote로 직접 여는 확인).
	for _, name := range []string{"raster-deck.pptx", "index.html"} {
		if _, err := os.Stat(filepath.Join(result.Dir, "raster-deck", name)); err != nil {
			t.Errorf("검수 산출물 %s가 없다: %v", name, err)
		}
	}
}

// TestRunDefaultRendersFallBackToWebWithoutTools는 Renders를 주지 않은 검수가 래스터 도구 없이
// 웹 열만으로 갤러리를 만들고, 그 사실을 로그 · 결과 · 갤러리 안내에 남기는지 본다.
//
// soffice가 **있는** 환경에서도 도는 테스트다 (환경 변수로 없는 경로를 준다): 도구가 없는 호스트에서도
// 저작 루프가 웹 갤러리까지는 가야 한다.
func TestRunDefaultRendersFallBackToWebWithoutTools(t *testing.T) {
	testkit.RequireChromium(t)
	t.Setenv(preview.EnvSoffice, filepath.Join(t.TempDir(), "없는-soffice"))

	project := twoSlideProject(t)
	var log bytes.Buffer
	result, err := preview.Run(preview.Options{
		Build: pipeline.Options{ProjectDir: project, OutDir: t.TempDir()},
		Dir:   filepath.Join(project, "preview"),
		Log:   &log,
	})
	if err != nil {
		t.Fatalf("기본 렌더의 검수가 도구 없이 실패했다: %v", err)
	}
	if len(result.Renders) != 1 || result.Renders[0] != render.TargetHTML {
		t.Errorf("렌더 = %v, want [html]", result.Renders)
	}
	if !strings.Contains(result.SkippedPPTX, preview.EnvSoffice) {
		t.Errorf("PPTX를 뺀 이유에 경로를 주는 방법이 없다: %q", result.SkippedPPTX)
	}
	if !strings.Contains(log.String(), "PPTX 열을 뺀다") {
		t.Errorf("PPTX 열을 조용히 뺐다 — 로그:\n%s", log.String())
	}
	index, err := os.ReadFile(result.Index)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(index), "PPTX 열을 뺐다") {
		t.Error("갤러리 안내가 PPTX 열이 빠졌다고 말하지 않는다 — 사람은 두 타깃을 다 봤다고 읽는다")
	}
	if _, err := os.Stat(filepath.Join(result.Dir, "raster-deck", fmtName(0, "web"))); err != nil {
		t.Errorf("웹 칸이 없다: %v", err)
	}
	if _, err := os.Stat(filepath.Join(result.Dir, "raster-deck", fmtName(0, "pptx"))); err == nil {
		t.Error("그릴 수 없는 pptx 칸이 남았다")
	}
}

// fmtName은 갤러리가 칸 하나에 쓰는 파일 이름이다.
func fmtName(index int, caption string) string {
	return fmt.Sprintf("%02d-%s.png", index+1, caption)
}
