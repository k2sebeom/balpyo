package testkit

import (
	"image"
	"testing"

	"github.com/k2sebeom/balpyo/internal/preview"
)

// 픽셀 비교는 internal/preview가 소유한다 (ADR 0047) — `balpyo preview`가 사람에게 보여 주는
// 숫자와 테스트가 판정하는 숫자가 **같은 코드**에서 나와야 한다. 별칭으로 두는 이유는
// 하네스가 보는 이름을 움직이지 않기 위해서다.
type (
	// DiffOptions는 픽셀 비교 허용 오차다 (채널당 허용 차이 + 허용 픽셀 비율).
	DiffOptions = preview.DiffOptions
	// DiffResult는 픽셀 비교 결과다.
	DiffResult = preview.DiffResult
)

// DefaultDiffOptions는 웹 스크린샷 vs LibreOffice 렌더 비교의 기본 오차다 (2.5% · 채널 32).
func DefaultDiffOptions() DiffOptions { return preview.DefaultDiffOptions() }

// Diff는 두 이미지를 픽셀 단위로 비교한다.
func Diff(want, got image.Image, opts DiffOptions) (DiffResult, error) {
	return preview.Diff(want, got, opts)
}

// DecodePNG는 PNG 바이트를 이미지로 만든다.
func DecodePNG(data []byte) (image.Image, error) { return preview.DecodePNG(data) }

// AssertImagesMatch는 두 렌더 결과가 허용 오차 안에서 같은지 검사하고 결과를 돌려준다
// (리뷰 갤러리가 같은 결과를 쓴다).
func AssertImagesMatch(t testing.TB, want, got image.Image, opts DiffOptions) DiffResult {
	t.Helper()
	result, err := Diff(want, got, opts)
	if err != nil {
		t.Errorf("픽셀 비교 실패: %v", err)
		return result
	}
	if result.Ratio > opts.MaxRatio {
		t.Errorf("렌더 결과가 허용 오차를 넘는다: %s (허용 %.3f%%, 채널 %d)\n"+
			"diff 이미지를 보려면 `make review`", result, opts.MaxRatio*100, opts.Channel)
	}
	return result
}

// EncodePNG는 이미지를 PNG 바이트로 만든다.
func EncodePNG(t testing.TB, img image.Image) []byte {
	t.Helper()
	data, err := preview.EncodePNG(img)
	if err != nil {
		t.Fatalf("%v", err)
	}
	return data
}

// LoadPNG는 PNG 파일을 읽는다.
func LoadPNG(t testing.TB, path string) image.Image {
	t.Helper()
	img, err := preview.LoadPNG(path)
	if err != nil {
		t.Fatalf("%v", err)
	}
	return img
}

// SavePNG는 이미지를 PNG 파일로 쓴다 (상위 디렉토리를 만든다).
func SavePNG(t testing.TB, path string, img image.Image) {
	t.Helper()
	if err := preview.SavePNG(path, img); err != nil {
		t.Fatalf("%v", err)
	}
}
