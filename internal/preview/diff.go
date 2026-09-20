package preview

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
)

// DiffOptions는 픽셀 비교 허용 오차다.
//
// 두 값으로 나눈 이유: 텍스트 안티앨리어싱은 글자 **경계 픽셀**의 채널을 크게 흔들지만
// 흔드는 픽셀 수는 적다. 채널 허용값만 키우면 색이 조금 틀린 큰 면적을 놓치고, 비율만
// 키우면 글자가 굵어진 것을 놓친다.
type DiffOptions struct {
	// Channel은 채널당 허용 차이다 (0-255). 이 값을 넘는 픽셀만 "다르다"로 센다.
	Channel int
	// MaxRatio는 허용 픽셀 비율이다 (0-1). 다른 픽셀이 이 비율을 넘으면 실패다.
	MaxRatio float64
}

// DefaultDiffOptions는 웹 스크린샷 vs LibreOffice 렌더 비교의 기본 오차다.
//
// 두 렌더러는 같은 폰트를 쓰지만 래스터라이저가 다르다 (Skia vs LibreOffice/poppler).
// 글리프 경계와 안티앨리어싱이 완전히 같을 수는 없으므로 0 오차는 달성 불가능한 목표다.
//
// MaxRatio 2.5%는 Iteration 1에서 01_text(한글·영문 혼합 텍스트 3장)로 측정한 값이다
// (PRD §13-1, NOTES.md 01/07): 글자가 큰 표지 슬라이드가 최대 2.1%였고, 그 구성은 전부
// 엔진 소관의 잔차다 — LibreOffice의 첫 기준선이 +2~5px 낮고, 스크립트 간격이 혼합 문자
// 줄 안의 글리프를 최대 +3.3% 오른쪽으로 흘린다. 이 잔차가 구조적 갈라짐(줄 수·줄 위치·
// 줄 폭)으로 번지는 것은 픽셀 비율이 아니라 잉크 띠 검사가 잡는다
// (TestFixtureLineBreaksMatchLibreOffice) — 그래서 비율을 그 이상 조이지 않는다.
//
// 이 값이 **사용자 덱의 게이트가 아닌** 이유는 ADR 0047에 있다: 픽스처별 조정표(2.5%~7%)에는
// "이 슬라이드는 두 렌더가 달라야 정답이다"까지 들어 있고, 그 표 없이는 임계값을 고를 근거가
// 없다. `balpyo preview`는 숫자를 보여 주고 판단은 사람에게 남긴다.
func DefaultDiffOptions() DiffOptions {
	return DiffOptions{Channel: 32, MaxRatio: 0.025}
}

// DiffResult는 픽셀 비교 결과다.
type DiffResult struct {
	Width, Height int
	// Changed는 Channel 허용값을 넘은 픽셀 수다.
	Changed int
	// Total은 전체 픽셀 수다.
	Total int
	// Ratio는 Changed / Total 이다.
	Ratio float64
	// MaxChannel은 관측된 최대 채널 차이다 — 허용값을 얼마나 여유 있게 통과했는지 보여 준다.
	MaxChannel int
	// Image는 사람이 보는 diff 이미지다 (다른 픽셀만 강조).
	Image *image.RGBA
}

// String은 로그와 갤러리가 함께 쓰는 한 줄 요약이다.
func (r DiffResult) String() string {
	return fmt.Sprintf("다른 픽셀 %d/%d (%.3f%%), 최대 채널 차 %d",
		r.Changed, r.Total, r.Ratio*100, r.MaxChannel)
}

// Diff는 두 이미지를 픽셀 단위로 비교한다. 크기가 다르면 에러다 — 크기를 맞추는 일은
// 렌더 단계의 책임이고, 여기서 리샘플하면 그 실수가 오차 안에 숨는다.
func Diff(want, got image.Image, opts DiffOptions) (DiffResult, error) {
	wb, gb := want.Bounds(), got.Bounds()
	if wb.Dx() != gb.Dx() || wb.Dy() != gb.Dy() {
		return DiffResult{}, fmt.Errorf("이미지 크기가 다르다: %dx%d vs %dx%d",
			wb.Dx(), wb.Dy(), gb.Dx(), gb.Dy())
	}

	w, h := wb.Dx(), wb.Dy()
	result := DiffResult{
		Width:  w,
		Height: h,
		Total:  w * h,
		Image:  image.NewRGBA(image.Rect(0, 0, w, h)),
	}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			wc := RGB8(want.At(wb.Min.X+x, wb.Min.Y+y))
			gc := RGB8(got.At(gb.Min.X+x, gb.Min.Y+y))
			delta := ChannelDelta(wc, gc)
			if delta > result.MaxChannel {
				result.MaxChannel = delta
			}
			if delta > opts.Channel {
				result.Changed++
				// 다른 픽셀은 마젠타로 — 슬라이드에 잘 쓰이지 않는 색이라 눈에 띈다.
				result.Image.SetRGBA(x, y, color.RGBA{R: 255, G: 0, B: 255, A: 255})
				continue
			}
			result.Image.SetRGBA(x, y, faded(wc))
		}
	}
	if result.Total > 0 {
		result.Ratio = float64(result.Changed) / float64(result.Total)
	}
	return result, nil
}

// faded는 배경으로 깔리는 흐린 픽셀이다. diff 이미지에서 원본 형태는 맥락으로만 보이고
// 강조색이 도드라지게 한다.
func faded(c color.RGBA) color.RGBA {
	mix := func(v uint8) uint8 { return uint8(int(v)/5 + 204) } // 20% 원본 + 80% 흰색
	return color.RGBA{R: mix(c.R), G: mix(c.G), B: mix(c.B), A: 255}
}

// RGB8은 색을 8비트 RGB로 정규화한다. 알파는 흰색과 합성한다 — 스크린샷은 불투명이지만
// 알파가 있는 PNG를 받아도 결과가 흔들리지 않게 한다.
//
// 공개하는 이유: 잉크 띠 검사(internal/testkit)도 같은 정규화로 픽셀을 읽어야 한다. 정규화가
// 두 벌이면 diff가 "같다"고 한 픽셀을 띠 검사가 "글자다"로 세는 상태가 생긴다.
func RGB8(c color.Color) color.RGBA {
	r, g, b, a := c.RGBA()
	if a == 0 {
		return color.RGBA{R: 255, G: 255, B: 255, A: 255}
	}
	// RGBA()는 알파 프리멀티플라이드 16비트다. 흰 배경 위에 합성한 뒤 8비트로 줄인다:
	// v + (1-alpha)×흰색 = v + (0xffff - a). v ≤ a 이므로 넘치지 않는다.
	over := func(v uint32) uint8 { return uint8((v + (0xffff - a)) >> 8) }
	return color.RGBA{R: over(r), G: over(g), B: over(b), A: 255}
}

// ChannelDelta는 세 채널 차이 중 최대값이다.
func ChannelDelta(a, b color.RGBA) int {
	delta := func(x, y uint8) int {
		if x > y {
			return int(x - y)
		}
		return int(y - x)
	}
	return max(delta(a.R, b.R), max(delta(a.G, b.G), delta(a.B, b.B)))
}

// DecodePNG는 PNG 바이트를 이미지로 만든다.
func DecodePNG(data []byte) (image.Image, error) {
	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("PNG를 디코드할 수 없다: %w", err)
	}
	return img, nil
}

// EncodePNG는 이미지를 PNG 바이트로 만든다.
func EncodePNG(img image.Image) ([]byte, error) {
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, fmt.Errorf("PNG를 인코드할 수 없다: %w", err)
	}
	return buf.Bytes(), nil
}

// LoadPNG는 PNG 파일을 읽는다.
func LoadPNG(path string) (image.Image, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("PNG를 읽을 수 없다: %w", err)
	}
	img, err := DecodePNG(data)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return img, nil
}

// SavePNG는 이미지를 PNG 파일로 쓴다 (상위 디렉토리를 만든다).
func SavePNG(path string, img image.Image) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("디렉토리를 만들 수 없다: %w", err)
	}
	data, err := EncodePNG(img)
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("PNG를 쓸 수 없다: %w", err)
	}
	return nil
}
