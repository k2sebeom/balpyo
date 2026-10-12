package snapshot

import (
	"bufio"
	"compress/lzw"
	"fmt"
	"image"
	"image/color"
	"io"
	"time"
)

// gifWriter는 GIF를 프레임 하나씩 흘려 쓴다. image/gif의 EncodeAll은 모든 프레임을 한 번에
// 받으므로 덱 전체를 메모리에 쥐어야 하지만, 여기서는 프레임을 받는 대로 팔레트 · LZW로 써
// 버리고 놓는다. 프레임마다 지역 색표를 두고 전역 색표는 쓰지 않으며, 무한 반복한다.
type gifWriter struct {
	w      *bufio.Writer
	width  int
	height int
	frames int
	err    error
}

func newGIFWriter(w io.Writer) *gifWriter {
	return &gifWriter{w: bufio.NewWriter(w)}
}

func (g *gifWriter) bytes(b ...byte) {
	if g.err == nil {
		_, g.err = g.w.Write(b)
	}
}

func (g *gifWriter) uint16(v int) { g.bytes(byte(v), byte(v>>8)) }

// header는 첫 프레임의 크기로 논리 화면과 반복 확장을 쓴다.
func (g *gifWriter) header(width, height int) {
	g.width, g.height = width, height
	g.bytes([]byte("GIF89a")...)
	g.uint16(width)
	g.uint16(height)
	g.bytes(0, 0, 0) // 전역 색표 없음 · 배경 색 0 · 픽셀 비율 미지정
	// NETSCAPE2.0 확장: 반복 횟수 0은 무한 반복이다.
	g.bytes(0x21, 0xff, 0x0b)
	g.bytes([]byte("NETSCAPE2.0")...)
	g.bytes(0x03, 0x01, 0, 0, 0)
}

// Add는 프레임 하나를 쓴다. 모든 프레임은 첫 프레임과 크기가 같아야 한다.
func (g *gifWriter) Add(p *image.Paletted, delay time.Duration) error {
	if g.err != nil {
		return g.err
	}
	w, h := p.Rect.Dx(), p.Rect.Dy()
	if w < 1 || h < 1 || w > 0xffff || h > 0xffff {
		return fmt.Errorf("a %dx%d frame does not fit in a GIF", w, h)
	}
	if len(p.Palette) == 0 || len(p.Palette) > 256 {
		return fmt.Errorf("a GIF frame needs 1 to 256 colors, got %d", len(p.Palette))
	}
	if g.frames == 0 {
		g.header(w, h)
	} else if w != g.width || h != g.height {
		return fmt.Errorf("frame %d is %dx%d, the GIF is %dx%d", g.frames+1, w, h, g.width, g.height)
	}
	// 그래픽 제어 확장: 지연(1/100초) · 투명색 없음.
	g.bytes(0x21, 0xf9, 0x04, 0)
	g.uint16(int(delay / (10 * time.Millisecond)))
	g.bytes(0, 0)
	// 이미지 서술자와 지역 색표 (색 수는 2의 거듭제곱으로 채운다).
	bits := 1
	for 1<<bits < len(p.Palette) {
		bits++
	}
	g.bytes(0x2c)
	g.uint16(0)
	g.uint16(0)
	g.uint16(w)
	g.uint16(h)
	g.bytes(0x80 | byte(bits-1))
	for i := 0; i < 1<<bits; i++ {
		var r, gr, b byte
		if i < len(p.Palette) {
			c := color.RGBAModel.Convert(p.Palette[i]).(color.RGBA)
			r, gr, b = c.R, c.G, c.B
		}
		g.bytes(r, gr, b)
	}
	// LZW 최소 코드 크기는 2 이상이다.
	lit := max(bits, 2)
	g.bytes(byte(lit))
	if g.err != nil {
		return g.err
	}
	bw := &blockWriter{w: g.w}
	lz := lzw.NewWriter(bw, lzw.LSB, lit)
	for y := 0; y < h && g.err == nil; y++ {
		row := p.Pix[y*p.Stride : y*p.Stride+w]
		for _, v := range row {
			if int(v) >= len(p.Palette) {
				lz.Close()
				return fmt.Errorf("color index %d is past the %d-color palette", v, len(p.Palette))
			}
		}
		_, g.err = lz.Write(row)
	}
	if err := lz.Close(); g.err == nil {
		g.err = err
	}
	if g.err == nil {
		g.err = bw.close()
	}
	g.frames++
	return g.err
}

// Close는 끝 표시를 쓰고 버퍼를 비운다. 프레임이 하나도 없으면 에러다.
func (g *gifWriter) Close() error {
	if g.err != nil {
		return g.err
	}
	if g.frames == 0 {
		return fmt.Errorf("no slide to put in the GIF")
	}
	g.bytes(0x3b)
	if g.err != nil {
		return g.err
	}
	return g.w.Flush()
}

// blockWriter는 LZW 출력을 GIF의 255바이트 하위 블록으로 자른다.
type blockWriter struct {
	w   io.Writer
	buf [255]byte
	n   int
}

func (b *blockWriter) Write(p []byte) (int, error) {
	total := len(p)
	for len(p) > 0 {
		k := copy(b.buf[b.n:], p)
		b.n += k
		p = p[k:]
		if b.n == len(b.buf) {
			if err := b.flush(); err != nil {
				return total - len(p), err
			}
		}
	}
	return total, nil
}

func (b *blockWriter) flush() error {
	if b.n == 0 {
		return nil
	}
	if _, err := b.w.Write([]byte{byte(b.n)}); err != nil {
		return err
	}
	_, err := b.w.Write(b.buf[:b.n])
	b.n = 0
	return err
}

// close는 남은 블록과 블록 끝(길이 0) 표시를 쓴다.
func (b *blockWriter) close() error {
	if err := b.flush(); err != nil {
		return err
	}
	_, err := b.w.Write([]byte{0})
	return err
}
