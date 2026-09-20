// LibreOffice의 **스크립트 간격**을 못 박는 특성 테스트다 (D11). 다른 테스트들이 "두
// 렌더러가 같은가"를 보는 것과 달리 이 테스트는 **뷰어가 무엇을 하는지**를 잰다: 웹과
// PPTX의 갈라짐 중에 우리 코드가 원인이 아닌 것이 하나 남아 있고, 그것의 크기가
// 픽스처 허용 오차(loLineWidthMaxRatios)를 정하기 때문이다.
//
// 재는 것은 한 문장이다 — **스크립트 경계가 없는 줄은 갈라지지 않고, 경계마다 1/4 em씩
// 갈라진다**. 이 두 줄이 한 슬라이드에 함께 있는 것이 실험 설계다: 순 한글 줄이 대조군이라
// 폰트 메트릭 차이(엔진이 글자를 조금 좁게 그리는 것)와 스크립트 간격이 섞이지 않는다.
//
// 이 테스트가 깨지면 **LibreOffice가 바뀐 것이다**(핀은 docker/versions.env가 잡는다) —
// 고칠 곳은 이 파일이 아니라 두 곳이다: 픽스처별 허용 비율(loLineWidthMaxRatios)을 다시
// 재서 좁히고, `docs/guides/known-differences.md`를 고친다.
package testkit

import (
	"fmt"
	"image"
	"os"
	"path/filepath"
	"testing"
	"unicode"

	"github.com/k2sebeom/balpyo/internal/pipeline"
	"github.com/k2sebeom/balpyo/render"
	"github.com/k2sebeom/balpyo/render/geom"
	"github.com/k2sebeom/balpyo/render/web"
)

// scriptSpacingSize는 탐침 문단의 글자 크기다 (px). 1/4 em 판정의 기준 단위다.
const scriptSpacingSize = 20

// scriptSpacingQuantumEm은 경계 하나가 벌어지는 폭의 허용 범위다 (em).
//
// 1/4 em은 CJK 조판의 관례값이고(JIS X 4051의 서양 문자 사이 간격), 측정값은 4.9px/20px =
// 0.245em이다. 범위를 두는 이유는 잉크로 재기 때문이다 — 안티앨리어싱이 띠 양 끝에서
// ±1px을 만들고, 그것이 경계 11곳으로 나뉘면 ±0.005em이다.
const (
	scriptSpacingQuantumMinEm = 0.18
	scriptSpacingQuantumMaxEm = 0.32
)

// scriptSpacingNoiseRatio는 **경계가 없는** 줄에서 허용하는 폭 차이다.
//
// 0이 아닌 이유: 두 엔진은 같은 폰트로도 글자 전진 폭을 완전히 같게 쓰지 않는다 (측정:
// 순 한글 줄에서 LibreOffice가 1.2% **좁다** — ADR 0005의 측정 범위 −1.5%와 같은 잔차다).
// 스크립트 간격은 이 잔차와 부호가 반대이고 크기가 열 배라 섞이지 않는다.
const scriptSpacingNoiseRatio = 0.02

// scriptSpacingProbe는 탐침 슬라이드의 문단들이다 — 순서가 잉크 띠 순서다.
//
// 1·3번 줄은 ASCII가 한 글자도 없다 (마침표도 스크립트 경계다). 2번 줄은 같은 한글 문장에
// 문장부호만 넣은 것이라 **줄 길이는 짧고 경계는 많다** — 허용 오차를 비율로 두면 이 줄이
// 가장 크게 나오는 이유이고(경계 수는 줄 길이와 무관하다), 픽스처 허용 비율이 픽스처마다
// 커진 이유이기도 하다.
var scriptSpacingProbe = []string{
	"한글만 쓴 문장은 스크립트 경계가 없으므로 두 엔진이 같은 폭으로 그린다",
	"쉼표, 마침표. 물음표? 느낌표! 콜론: 세미콜론;",
	"한글만 쓴 문장은 스크립트 경계가 없으므로 두 엔진이 같은 폭으로 그린다 쉼표 마침표 물음표",
}

func TestLibreOfficeScriptSpacingIsQuarterEmPerBoundary(t *testing.T) {
	RequireChromium(t)
	RequireSoffice(t)
	RequirePdftoppm(t)

	result := buildScriptSpacingProbe(t)
	stage := result.Deck.Stage
	stageRect := geom.Rect{W: float64(stage.Width), H: float64(stage.Height)}

	var pptxPath string
	for _, name := range result.Written {
		if filepath.Ext(name) == ".pptx" {
			pptxPath = filepath.Join(result.OutDir, name)
		}
	}
	if pptxPath == "" {
		t.Fatalf("PPTX 산출물이 없다: %v", result.Written)
	}
	shots := RenderPPTX(t, pptxPath, image.Pt(stage.Width, stage.Height))

	browser := NewBrowser(t)
	browser.Load(t, filepath.Join(result.OutDir, web.IndexFile))
	webBands := InkBands(browser.ScreenshotElement(t, web.SlideIndexSelector(0)), stageRect)
	loBands := InkBands(shots[0], stageRect)

	// 띠 수가 다르면 줄이 접힌 것이다 — 이 탐침의 문단은 전부 한 줄이어야 하므로 그때는
	// 폭 비교가 뜻을 잃는다 (다른 문단의 줄을 맞대게 된다).
	if len(webBands) != len(scriptSpacingProbe) || len(loBands) != len(scriptSpacingProbe) {
		t.Fatalf("띠 수가 문단 수와 다르다 — 웹 %d, LibreOffice %d, 문단 %d",
			len(webBands), len(loBands), len(scriptSpacingProbe))
	}

	for i, text := range scriptSpacingProbe {
		boundaries := scriptBoundaries(text)
		wb, lo := webBands[i], loBands[i]
		delta := lo.W - wb.W
		t.Logf("경계 %2d곳: 웹 %6.1fpx → LibreOffice %6.1fpx (%+.1fpx, %+.1f%%)",
			boundaries, wb.W, lo.W, delta, (lo.W/wb.W-1)*100)

		if boundaries == 0 {
			// 대조군: 경계가 없으면 스크립트 간격도 없다. 여기가 깨지면 두 엔진의 글자
			// 전진 폭 자체가 갈라진 것이므로 원인이 스크립트 간격이 아니다.
			if ratio := delta / wb.W; ratio > scriptSpacingNoiseRatio || ratio < -scriptSpacingNoiseRatio {
				t.Errorf("경계가 없는 줄의 폭이 %+.1f%% 다르다 (허용 ±%.0f%%) — 스크립트 간격이 아닌 원인이 있다",
					ratio*100, scriptSpacingNoiseRatio*100)
			}
			continue
		}

		// 경계가 있는 줄: 벌어진 폭을 경계 수로 나눈 값이 1/4 em이다. 비율(%)로 보지 않는
		// 이유가 이 나눗셈이다 — 오차는 줄 길이가 아니라 경계 수에 비례하므로, 같은 문장을
		// 좁은 트랙에 넣으면 비율만 커진다 (loLineWidthMaxRatios의 픽스처별 조정이 그 기록이다).
		quantum := delta / float64(boundaries) / scriptSpacingSize
		t.Logf("  경계당 %+.3f em (%+.1fpx) — 1/4 em 관례값", quantum, delta/float64(boundaries))
		if quantum < scriptSpacingQuantumMinEm || quantum > scriptSpacingQuantumMaxEm {
			t.Errorf("경계당 간격이 %+.3f em이다 (기대 %.2f~%.2f em) — LibreOffice의 동작이 바뀌었다면 "+
				"loLineWidthMaxRatios를 다시 재고 docs/guides/known-differences.md를 고친다",
				quantum, scriptSpacingQuantumMinEm, scriptSpacingQuantumMaxEm)
		}
		// 텍스트 박스의 안전 여유(pptx.TextBoxRect의 4%)로는 덮을 수 없다는 사실이 이 줄의
		// 뜻이다 — 여유를 이만큼 키우면 반대 방향으로 줄이 줄어든다 (ADR 0005).
		if ratio := delta / wb.W; ratio <= 0.04 {
			t.Errorf("문장부호가 몰린 줄의 폭 차이가 %+.1f%%다 — 4%% 안에 들어왔다면 "+
				"허용 오차를 다시 재서 좁힐 수 있다 (loLineWidthMaxRatios)", ratio*100)
		}
	}
}

// 웹이 스크립트 간격을 **끈 상태로** 잰다는 계약 (render/web의 `.balpyo-stage`).
//
// 선언이 산출물에 있는지는 단위 테스트가 보고(TestRenderStagePinsScriptSpacing), 여기서는
// **브라우저가 그것을 받았는지**를 본다: 오타나 뒤 층의 재정의로 값이 죽으면 CSS 문자열
// 검사는 통과하고 잰 폭만 조용히 움직인다. 지금 브라우저는 속성을 파싱하지만 간격을 넣지
// 않으므로(측정: `normal`과 `no-autospace`의 폭이 같다) 이 테스트는 오늘의 그림을 지키는
// 것이 아니라 **내일 브라우저가 기본값을 켤 때** 잰 폭이 움직이지 않게 못 박는다.
func TestBrowserHonorsTheScriptSpacingPin(t *testing.T) {
	RequireChromium(t)
	result := buildScriptSpacingProbe(t)

	browser := NewBrowser(t)
	browser.Load(t, filepath.Join(result.OutDir, web.IndexFile))
	var got struct {
		Supported bool   `json:"supported"`
		Value     string `json:"value"`
	}
	browser.Eval(t, fmt.Sprintf(`(() => {
  const stage = document.querySelector(%q);
  return {supported: CSS.supports("text-autospace", "no-autospace"),
          value: getComputedStyle(stage).textAutospace ?? ""};
})()`, web.StageSelector()), &got)

	if !got.Supported {
		// 속성을 모르는 브라우저는 간격도 넣지 않는다 — 고정할 것이 아직 없다는 뜻이고,
		// 속성이 생기는 버전에서 이 테스트가 판정을 시작한다.
		t.Logf("브라우저가 text-autospace를 모른다 — 고정이 아직 뜻을 갖지 않는다")
		return
	}
	if got.Value != "no-autospace" {
		t.Errorf("스테이지의 text-autospace = %q, want %q — 잰 폭이 브라우저 기본값에 실려 있다",
			got.Value, "no-autospace")
	}
}

// buildScriptSpacingProbe는 탐침 프로젝트를 임시 디렉토리에 만들어 두 타깃으로 빌드한다.
//
// 픽스처로 두지 않는 이유: 이 슬라이드는 저작 어휘를 보여 주지 않고 **뷰어를 재는 계측기**다
// (골든을 붙이면 뷰어의 동작이 골든에 박힌다).
func buildScriptSpacingProbe(t testing.TB) *pipeline.Result {
	t.Helper()
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "slides"), 0o755); err != nil {
		t.Fatal(err)
	}
	config := `{"title": "script spacing", "lang": "ko", "aspect": "16:9", "outDir": "dist"}`
	if err := os.WriteFile(filepath.Join(dir, "balpyo.json"), []byte(config), 0o644); err != nil {
		t.Fatal(err)
	}
	slide := "<slide id=\"script-spacing\">\n"
	for _, text := range scriptSpacingProbe {
		slide += fmt.Sprintf("  <p size=\"%d\">%s</p>\n", scriptSpacingSize, text)
	}
	slide += "</slide>\n"
	if err := os.WriteFile(filepath.Join(dir, "slides", "001_script_spacing.slide"), []byte(slide), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := pipeline.Build(pipeline.Options{
		ProjectDir: dir,
		Targets:    []render.Target{render.TargetHTML, render.TargetPPTX},
		OutDir:     t.TempDir(),
	})
	if err != nil {
		t.Fatalf("탐침 빌드 실패: %v", err)
	}
	return result
}

// scriptBoundaries는 한글과 그 밖의 문자가 맞닿은 자리의 수다 — 공백은 세지 않고 **건너뛴다**
// (LibreOffice는 이미 공백이 있는 자리에도 간격을 더한다: 측정에서 `쉼표 ,`의 간격이 벌어졌다).
func scriptBoundaries(text string) int {
	count, prev := 0, rune(0)
	for _, r := range text {
		if unicode.IsSpace(r) {
			continue
		}
		if prev != 0 && isHangul(prev) != isHangul(r) {
			count++
		}
		prev = r
	}
	return count
}

func isHangul(r rune) bool { return unicode.Is(unicode.Hangul, r) }
