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

// RGB8은 색을 8비트 RGB로 정규화한다. 알파는 흰색과 합성한다 — 스크린샷은 불투명이지만
// 알파가 있는 PNG를 받아도 결과가 흔들리지 않게 한다.
//
// 공개하는 이유: 픽셀을 읽는 테스트(internal/testkit · 래스터 결정성)가 같은 정규화를 쓴다.
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

// DecodePNG는 PNG 바이트를 이미지로 만든다.
func DecodePNG(data []byte) (image.Image, error) {
	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("cannot decode PNG: %w", err)
	}
	return img, nil
}

// EncodePNG는 이미지를 PNG 바이트로 만든다.
func EncodePNG(img image.Image) ([]byte, error) {
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, fmt.Errorf("cannot encode PNG: %w", err)
	}
	return buf.Bytes(), nil
}

// LoadPNG는 PNG 파일을 읽는다.
func LoadPNG(path string) (image.Image, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("cannot read PNG: %w", err)
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
		return fmt.Errorf("cannot create directory: %w", err)
	}
	data, err := EncodePNG(img)
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("cannot write PNG: %w", err)
	}
	return nil
}
