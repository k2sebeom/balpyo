package testkit

import (
	"image"
	"testing"

	"github.com/k2sebeom/balpyo/internal/preview"
)

// OpenPPTX는 LibreOffice가 파일을 **깨끗하게** 여는지 검사하고 PDF 경로를 돌려준다.
//
// 변환 자체는 internal/preview가 한다 (ADR 0047) — 이 파일이 소유하는 것은 **테스트 정책**뿐이다:
// 도구가 없을 때의 건너뛰기/실패(RequireSoffice), 임시 디렉토리, 그리고 실패를 t.Fatal로 바꾸는
// 얇은 층.
//
// 경고를 실패로 보는 이유: PowerPoint와 LibreOffice는 깨진 패키지를 조용히 복구해서 열어 주기
// 때문에, 복구가 일어나면 사용자는 파일을 열 때마다 경고를 보게 된다.
func OpenPPTX(t testing.TB, pptxPath string) string {
	t.Helper()
	raster := &preview.Rasterizer{Soffice: RequireSoffice(t)}
	pdfPath, noise, err := raster.PDF(pptxPath, t.TempDir())
	if err != nil {
		t.Fatalf("%v", err)
	}
	if noise != "" {
		t.Errorf("LibreOffice가 경고·복구를 냈다 — 사용자도 파일을 열 때마다 본다:\n%s", noise)
	}
	return pdfPath
}

// RenderPPTX는 pptx를 슬라이드별 PNG로 렌더한다 (soffice → pdf → pdftoppm → png).
//
// size는 논리 스테이지 크기다 — 웹 스크린샷과 픽셀 비교하려면 같은 크기여야 한다.
func RenderPPTX(t testing.TB, pptxPath string, size image.Point) []image.Image {
	t.Helper()
	return RenderPDF(t, OpenPPTX(t, pptxPath), size)
}

// RenderPDF는 PDF를 페이지별 PNG로 렌더한다.
func RenderPDF(t testing.TB, pdfPath string, size image.Point) []image.Image {
	t.Helper()
	raster := &preview.Rasterizer{Pdftoppm: RequirePdftoppm(t)}
	pages, err := raster.Pages(pdfPath, size, t.TempDir())
	if err != nil {
		t.Fatalf("%v", err)
	}
	return pages
}

// RenderPDFPoints는 PDF를 **종이 크기 그대로** 페이지별 PNG로 렌더한다 — PNG의 픽셀 수가 곧
// 페이지의 pt 크기다.
//
// 크기를 주는 RenderPDF와 나눠 두는 이유는 판정이 다르기 때문이다: 인쇄 게이트가 보는 것은
// 그림이 아니라 종이다 (한 장이 한 페이지인지, ADR 0062). 목표 크기를 주고 렌더하면 종이가
// letter로 나왔어도 PNG는 원하는 크기가 되어 그 실패가 사라진다.
func RenderPDFPoints(t testing.TB, pdfPath string) []image.Image {
	t.Helper()
	raster := &preview.Rasterizer{Pdftoppm: RequirePdftoppm(t)}
	pages, err := raster.PagesAtPointScale(pdfPath, t.TempDir())
	if err != nil {
		t.Fatalf("%v", err)
	}
	return pages
}
