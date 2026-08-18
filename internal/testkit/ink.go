package testkit

import (
	"image"
	"image/color"
	"math"

	"github.com/k2sebeom/balpyo/render/geom"
)

// inkChannel은 잉크 판정의 채널 임계값이다. 안티앨리어싱 가장자리의 옅은 픽셀은 배경으로
// 치고 글자 본체만 잉크로 세도록 픽셀 diff의 채널 허용값(DiffOptions.Channel)보다 크게 둔다 —
// 여기서 재는 것은 "몇 줄인가"이지 "얼마나 다른가"가 아니다.
const inkChannel = 64

// InkBands는 이미지의 사각형 영역 안에서 글자가 차지한 가로 띠를 위에서 아래 순서로 돌려준다.
//
// 지오메트리 패스의 줄 박스(geom.Box.Lines)를 **다른 렌더러의 그림**과 대조하는 도구다:
// LibreOffice 렌더의 텍스트 박스 영역에서 띠를 세면 그 엔진이 실제로 나눈 줄 수가 나온다
// (PRD §13-1의 줄바꿈 판정 — 줄 사이에 빈 행이 있다는 사실에만 기대므로 글리프 모양이 달라도
// 성립한다). 띠의 X·W는 그 줄의 잉크가 걸친 가로 범위라 줄 폭 비교에도 쓴다.
//
// 배경은 영역에서 가장 흔한 색으로 정한다. 모서리 픽셀을 집으면 글자가 모서리에 닿는 순간
// 배경이 글자색이 되고, 호출자가 배경색을 넘기게 하면 테마가 바뀔 때마다 테스트가 따라
// 바뀌어야 한다 — 글자는 박스의 소수 픽셀이라는 사실이 더 오래 간다.
func InkBands(img image.Image, r geom.Rect) []geom.Rect {
	x0, y0 := int(math.Floor(r.X)), int(math.Floor(r.Y))
	x1, y1 := int(math.Ceil(r.Right())), int(math.Ceil(r.Bottom()))
	bounds := img.Bounds()
	x0, y0 = max(x0, 0), max(y0, 0)
	x1, y1 = min(x1, bounds.Dx()), min(y1, bounds.Dy())
	if x0 >= x1 || y0 >= y1 {
		return nil
	}

	background := modeColor(img, x0, y0, x1, y1)
	var bands []geom.Rect
	open := false
	for y := y0; y < y1; y++ {
		left, right := -1, -1
		for x := x0; x < x1; x++ {
			if maxChannelDelta(rgb8(img.At(bounds.Min.X+x, bounds.Min.Y+y)), background) <= inkChannel {
				continue
			}
			if left < 0 {
				left = x
			}
			right = x
		}
		if left < 0 {
			open = false
			continue
		}
		if !open {
			bands = append(bands, geom.Rect{X: float64(left), Y: float64(y), W: float64(right - left + 1), H: 1})
			open = true
			continue
		}
		band := &bands[len(bands)-1]
		if fl := float64(left); fl < band.X {
			band.W += band.X - fl
			band.X = fl
		}
		if fr := float64(right + 1); fr > band.Right() {
			band.W = fr - band.X
		}
		band.H = float64(y) - band.Y + 1
	}
	return bands
}

// modeColor는 영역에서 가장 흔한 색이다 (배경 추정).
func modeColor(img image.Image, x0, y0, x1, y1 int) color.RGBA {
	bounds := img.Bounds()
	counts := map[color.RGBA]int{}
	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			counts[rgb8(img.At(bounds.Min.X+x, bounds.Min.Y+y))]++
		}
	}
	var mode color.RGBA
	best := -1
	for c, n := range counts {
		// 동률은 어두운 색을 뒤로 미룬다 — 순회 순서로 결과가 흔들리지 않게 결정적으로 고른다.
		if n > best || (n == best && less(c, mode)) {
			mode, best = c, n
		}
	}
	return mode
}

// less는 색의 결정적 전순서다 (modeColor의 동률 판정용).
func less(a, b color.RGBA) bool {
	if a.R != b.R {
		return a.R < b.R
	}
	if a.G != b.G {
		return a.G < b.G
	}
	return a.B < b.B
}
