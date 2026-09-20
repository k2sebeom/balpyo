package main

import (
	"errors"
	"flag"
	"fmt"
	"io"

	"github.com/k2sebeom/balpyo/internal/pipeline"
	"github.com/k2sebeom/balpyo/internal/preview"
	"github.com/k2sebeom/balpyo/render"
	"github.com/k2sebeom/balpyo/soml"
)

// defaultPreviewTargets는 검수 갤러리의 기본 열이다 — 이 명령의 존재 이유가 두 타깃을 나란히
// 보는 것이므로 기본이 build(html)와 다르다.
const defaultPreviewTargets = "html,pptx"

// runPreview는 `balpyo preview`다: 빌드하고, PPTX를 그림으로 되돌려 웹과 나란히 놓는다.
//
// **게이트가 아니다** (ADR 0047 결정 4). 허용 오차를 넘은 슬라이드가 있어도 종료 코드는 0이다 —
// 두 렌더가 달라야 정답인 슬라이드가 있고(§7.9의 `style` 외관), 그 판단에 필요한 조정표는 저장소의
// 테스트 정책이지 사용자 덱의 기준이 아니다. 1로 끝나는 경우는 빌드가 실패했거나 도구가 없는
// 경우뿐이다 — 둘 다 "검수할 것이 만들어지지 않았다"는 뜻이다.
func runPreview(args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("preview", flag.ContinueOnError)
	fs.SetOutput(stderr)
	// 플래그 이름과 뜻은 build와 같다 — `--out`은 여기서도 **빌드 산출물**이 가는 곳이다.
	// 갤러리에 같은 이름을 쓰면 `--out dist`가 어느 디렉토리를 뜻하는지 사람이 매번 확인해야 한다.
	targetList := fs.String("target", defaultPreviewTargets, "검수할 렌더 (html, pptx — 쉼표로 여러 개)")
	outDir := fs.String("out", "", "빌드 출력 디렉토리 (기본: balpyo.json의 outDir)")
	projectDir := fs.String("project", ".", "프로젝트 루트")
	theme := fs.String("theme", "", "테마 (기본: balpyo.json의 theme — 내장 테마만 받는다)")
	strict := fs.Bool("strict", false, "경고를 에러로 승격한다")
	galleryDir := fs.String("gallery", "", "검수 갤러리 디렉토리 (기본: <프로젝트>/"+preview.DefaultDirName+")")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() > 0 {
		return fmt.Errorf("preview는 인자를 받지 않는다: %v", fs.Args())
	}
	targets, err := render.ParseTargets(*targetList)
	if err != nil {
		return err
	}

	result, err := preview.Run(preview.Options{
		Build: pipeline.Options{
			ProjectDir: *projectDir,
			OutDir:     *outDir,
			Theme:      *theme,
			Strict:     *strict,
			// 검수는 배치를 **항상** 본다: 눈으로 볼 자리를 찾으러 온 사람에게 이미 숫자로 아는
			// 넘침·이탈을 말하지 않을 이유가 없다 (브라우저는 어차피 떠 있다).
			CheckLayout: true,
		},
		Renders: targets,
		Dir:     *galleryDir,
		Log:     stdout,
	})
	// 경고는 검수가 실패해도 사람에게 보여야 한다 — 승격된 경고가 곧 그 실패의 이유다.
	// (빌드를 시작하기 전에 멈춘 경우 result는 nil이다 — 도구가 없다, 타깃 이름이 틀렸다.)
	if result != nil {
		printDiags(stderr, result.Diagnostics)
	}
	if err != nil {
		var de *soml.DiagnosticsError
		if errors.As(err, &de) {
			return fmt.Errorf("검수할 산출물을 만들지 못했다: 에러 %d건", de.Count())
		}
		return err
	}

	fmt.Fprintf(stdout, "슬라이드 %d장 → %s\n", len(result.Slides), result.Index)
	if over := result.Over(); over > 0 {
		// 실패가 아니라 **볼 자리**다. 문장으로 그렇게 말해야 사람이 종료 코드 0을 보고 혼란스럽지
		// 않다.
		fmt.Fprintf(stdout, "허용 오차(%.1f%%)를 넘은 슬라이드 %d장 — 눈으로 볼 자리라는 표시이고 실패가 아니다\n",
			preview.DefaultDiffOptions().MaxRatio*100, over)
	}
	// 갤러리로 판정할 수 없는 확인이 남는다 — PPTX를 PowerPoint/Keynote로 열어 텍스트 박스가
	// 진짜 텍스트 박스인지, 테마 색 편집이 먹는지 보는 일. LibreOffice 렌더는 그 확인을 대신하지
	// 못한다 (다중 스톱 그라디언트 같은 자리에서 뷰어끼리 갈라진다 — 07_gradient 픽스처의 기록).
	if hasTarget(targets, render.TargetPPTX) {
		fmt.Fprintf(stdout, "PPTX는 PowerPoint·Keynote로 직접 열어 확인한다: %s\n", result.OutDir)
	}
	return nil
}

// hasTarget은 목록에 타깃이 있는지다.
func hasTarget(targets []render.Target, want render.Target) bool {
	for _, target := range targets {
		if target == want {
			return true
		}
	}
	return false
}
