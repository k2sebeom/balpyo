package testkit

import (
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
