package testkit

import (
	"image/color"
	"os"
	"path/filepath"
	"testing"
)

// geometryProbe는 크기와 위치가 CSS로 못 박힌 문서다. 브라우저가 읽어 주는 rect가 우리가
// 쓴 px와 같아야 한다.
const geometryProbe = `<!DOCTYPE html>
<html><head><meta charset="utf-8"><style>
  * { box-sizing: border-box; }
  html, body { margin: 0; padding: 0; }
  .stage { position: relative; width: 1280px; height: 720px; background: #fff; }
  .box { position: absolute; left: 80px; top: 60px; width: 400px; height: 200px; background: #4472c4; }
</style></head>
<body><div class="stage"><div class="box"></div></div></body></html>
`

// 지오메트리 패스(Iteration 1)와 앵커 계산(Iteration 7)이 이 헬퍼 위에 올라간다. 좌표를
// 잘못 읽으면 PPTX의 모든 도형이 틀린 자리에 놓이므로, 하네스 단계에서 축을 검증해 둔다.
func TestBrowserReadsElementRects(t *testing.T) {
	dir := t.TempDir()
	page := filepath.Join(dir, "probe.html")
	if err := os.WriteFile(page, []byte(geometryProbe), 0o644); err != nil {
		t.Fatal(err)
	}

	browser := NewBrowser(t)
	browser.Load(t, page)

	rects := browser.Rects(t, ".box")
	if len(rects) != 1 {
		t.Fatalf("rect %d개, want 1개", len(rects))
	}
	want := Rect{X: 80, Y: 60, W: 400, H: 200}
	if rects[0] != want {
		t.Errorf("rect = %s, want %s — 논리 px가 그대로 읽혀야 한다", rects[0], want)
	}

	// 요소 스크린샷은 요소 크기 그대로여야 한다 (스케일 1). 여기가 어긋나면 픽셀 비교가
	// 전부 크기 불일치로 실패한다.
	shot := browser.ScreenshotElement(t, ".stage")
	if bounds := shot.Bounds(); bounds.Dx() != 1280 || bounds.Dy() != 720 {
		t.Fatalf("스크린샷 크기 = %dx%d, want 1280x720", bounds.Dx(), bounds.Dy())
	}

	// 크기만 보면 어긋난 오프셋을 놓친다 — 잘라낸 영역이 **그 요소**인지 내용으로 본다.
	// 파란 박스의 네 꼭짓점 안팎을 찍어 축과 원점을 함께 검증한다.
	blue := color.RGBA{R: 0x44, G: 0x72, B: 0xc4, A: 255}
	white := color.RGBA{R: 255, G: 255, B: 255, A: 255}
	for _, probe := range []struct {
		x, y int
		want color.RGBA
	}{
		{80, 60, blue}, {479, 60, blue}, {80, 259, blue}, {479, 259, blue},
		{79, 60, white}, {80, 59, white}, {480, 259, white}, {479, 260, white},
		{0, 0, white}, {1279, 719, white},
	} {
		if got := rgb8(shot.At(probe.x, probe.y)); got != probe.want {
			t.Errorf("(%d,%d) = %v, want %v — 요소 박스가 아닌 영역을 찍었다",
				probe.x, probe.y, got, probe.want)
		}
	}
}

// offsetProbe는 픽스처 덱과 같은 구조다: 어두운 페이지 배경 + 여백 + 뷰포트보다 큰
// 스테이지. 슬라이드가 페이지 원점에 없고 스크롤이 필요한 상황을 만든다.
const offsetProbe = `<!DOCTYPE html>
<html><head><meta charset="utf-8"><style>
  * { box-sizing: border-box; }
  html, body { margin: 0; padding: 0; background: #111; }
  .deck { display: flex; flex-direction: column; align-items: center; gap: 24px; padding: 24px; }
  .stage { position: relative; width: 1280px; height: 720px; background: #fff; }
</style></head>
<body><div class="deck"><div class="stage"></div><div class="stage"></div></div></body></html>
`

// 요소 스크린샷이 뷰포트 원점을 찍으면 페이지 배경(#111)이 섞여 들어와 픽셀 비교가
// "웹과 PPTX가 22% 다르다"고 거짓 보고한다. 흰 스테이지만 찍혔는지 전수로 본다.
func TestBrowserScreenshotIgnoresPageOffset(t *testing.T) {
	dir := t.TempDir()
	page := filepath.Join(dir, "offset.html")
	if err := os.WriteFile(page, []byte(offsetProbe), 0o644); err != nil {
		t.Fatal(err)
	}

	browser := NewBrowser(t)
	browser.Load(t, page)

	// 두 번째 스테이지는 y=768부터 시작한다 — 스크롤 없이는 뷰포트에 들어오지 않는다.
	for i, selector := range []string{".stage:first-child", ".stage:last-child"} {
		shot := browser.ScreenshotElement(t, selector)
		bounds := shot.Bounds()
		if bounds.Dx() != 1280 || bounds.Dy() != 720 {
			t.Fatalf("스테이지 %d 스크린샷 크기 = %dx%d, want 1280x720", i+1, bounds.Dx(), bounds.Dy())
		}
		white := color.RGBA{R: 255, G: 255, B: 255, A: 255}
		for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
			for x := bounds.Min.X; x < bounds.Max.X; x++ {
				if got := rgb8(shot.At(x, y)); got != white {
					t.Fatalf("스테이지 %d: (%d,%d) = %v, want 흰색 — 요소 밖(페이지 배경)이 찍혔다",
						i+1, x-bounds.Min.X, y-bounds.Min.Y, got)
				}
			}
		}
	}
}
