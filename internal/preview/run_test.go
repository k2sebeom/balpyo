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
// 없다"를 알려 주면 사람의 시간을 버린다. 조용히 웹 열만 있는 갤러리를 내는 길도 두지 않는다:
// PPTX를 보려고 명령을 쳤는데 PPTX 열이 없는 갤러리가 나오면 사람은 **두 타깃이 같다고 읽는다**
// (ADR 0047 결정 5).
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

// TestRunBuildsGalleryWithBothRenders는 갤러리 한 장에 웹·PPTX·diff가 슬라이드마다 들어가는지
// 본다 — `make review`가 손으로 만들던 것과 같은 산출물이다.
func TestRunBuildsGalleryWithBothRenders(t *testing.T) {
	testkit.RequireChromium(t)
	testkit.RequireSoffice(t)
	testkit.RequirePdftoppm(t)

	project := twoSlideProject(t)
	var log bytes.Buffer
	result, err := preview.Run(preview.Options{
		Build:   pipeline.Options{ProjectDir: project, OutDir: t.TempDir()},
		Renders: preview.DefaultRenders(),
		Dir:     filepath.Join(project, "preview"),
		Log:     &log,
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
	for _, want := range []string{"web", "pptx", "diff", "raster-deck", "title", "body"} {
		if !strings.Contains(string(index), want) {
			t.Errorf("갤러리에 %q가 없다", want)
		}
	}
	// 슬라이드마다 세 칸이 파일로 남는다 — 문서만 있고 그림이 없으면 검수가 불가능하다.
	for i, slide := range result.Slides {
		if slide.Diff == nil {
			t.Errorf("#%d: 두 렌더를 봤는데 픽셀 비교가 없다", i+1)
			continue
		}
		for _, caption := range []string{"web", "pptx", "diff"} {
			path := filepath.Join(result.Dir, "raster-deck", fmtName(i, caption))
			if _, err := os.Stat(path); err != nil {
				t.Errorf("%s가 없다: %v", path, err)
			}
		}
		if !strings.Contains(log.String(), slide.ID) {
			t.Errorf("로그에 슬라이드 %s의 숫자가 없다:\n%s", slide.ID, log.String())
		}
	}
	// 산출물도 손에 잡히는 곳에 있어야 한다 (PowerPoint/Keynote로 직접 여는 확인).
	for _, name := range []string{"raster-deck.pptx", "index.html"} {
		if _, err := os.Stat(filepath.Join(result.Dir, "raster-deck", name)); err != nil {
			t.Errorf("검수 산출물 %s가 없다: %v", name, err)
		}
	}
}

// TestRunWithWebOnlyNeedsNoLibreOffice는 웹 열만 요청한 검수가 LibreOffice 없이 도는지 본다 —
// 그 갤러리는 한 열이고 판정(허용 오차 배지)이 없다.
func TestRunWithWebOnlyNeedsNoLibreOffice(t *testing.T) {
	testkit.RequireChromium(t)
	// soffice가 **있는** 환경에서도 찾지 않아야 한다: 웹만 보는 검수가 도구를 요구하면 저작 루프가
	// 컨테이너에 묶인다.
	t.Setenv(preview.EnvSoffice, filepath.Join(t.TempDir(), "없는-soffice"))

	project := twoSlideProject(t)
	result, err := preview.Run(preview.Options{
		Build:   pipeline.Options{ProjectDir: project, OutDir: t.TempDir()},
		Renders: []render.Target{render.TargetHTML},
		Dir:     filepath.Join(project, "preview"),
	})
	if err != nil {
		t.Fatalf("웹만 보는 검수가 실패했다: %v", err)
	}
	for i, slide := range result.Slides {
		if slide.Diff != nil {
			t.Errorf("#%d: 비교할 렌더가 하나인데 픽셀 비교가 있다", i+1)
		}
	}
	index, err := os.ReadFile(result.Index)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(index), "허용 오차") {
		t.Error("한 열 갤러리에 판정 배지가 있다 — 비교한 것이 없는데 판정이 보인다")
	}
	if _, err := os.Stat(filepath.Join(result.Dir, "raster-deck", fmtName(0, "pptx"))); err == nil {
		t.Error("요청하지 않은 pptx 칸이 남았다")
	}
}

// fmtName은 갤러리가 칸 하나에 쓰는 파일 이름이다.
func fmtName(index int, caption string) string {
	return fmt.Sprintf("%02d-%s.png", index+1, caption)
}
