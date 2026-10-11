package quick

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"

	"github.com/k2sebeom/balpyo/render/geom"
	"github.com/k2sebeom/balpyo/render/web"
	"github.com/k2sebeom/balpyo/soml"
)

// Document는 배치를 마친 문서다.
type Document struct {
	e      *engine
	root   *node
	Slides []*Slide
	// Width · Height는 논리 스테이지 크기다.
	Width, Height float64
}

// Slide는 배치를 마친 슬라이드 하나다.
type Slide struct {
	Index int
	node  *node
	box   *box
	boxes map[*node]*box
}

// Layout은 웹 렌더러가 낸 index.html을 배치한다. asset은 src의 바이트를 준다 (없으면 false).
func Layout(index []byte, asset func(name string) ([]byte, bool)) (*Document, error) {
	fonts, err := sharedFonts()
	if err != nil {
		return nil, fmt.Errorf("cannot load system fonts: %w", err)
	}
	root, sheets, err := parseHTML(index)
	if err != nil {
		return nil, err
	}
	newCascader(sheets).apply(root)

	e := &engine{fonts: fonts, images: map[string]*decodedImage{}, assets: assetMap{asset}}
	e.lang = root.attrs["lang"]
	doc := &Document{e: e, root: root}
	sections := root.find(func(n *node) bool { return n.hasClass(web.SlideClass) })
	for _, sec := range sections {
		idx, err := strconv.Atoi(sec.attrs[web.SlideIndexAttr])
		if err != nil {
			idx = -1
		}
		fonts.mu.Lock()
		b := e.build(sec, "")
		if b == nil {
			fonts.mu.Unlock()
			continue
		}
		w := lengthOr(sec.style.get("width"), sec.style.fontSize, length{px: 1280}).px
		h := lengthOr(sec.style.get("height"), sec.style.fontSize, length{px: 720}).px
		e.stageW = w
		e.layout(b, w, &h, w, h, true)
		b.x, b.y = 0, 0
		finalize(b, 0, 0)
		fonts.mu.Unlock()
		doc.Width, doc.Height = w, h
		e.stageW = w
		s := &Slide{Index: idx, node: sec, box: b, boxes: map[*node]*box{}}
		indexBoxes(b, s.boxes)
		doc.Slides = append(doc.Slides, s)
	}
	return doc, nil
}

// assetMap은 에셋 조회를 map처럼 쓰게 한다.
type assetMap struct{ get func(string) ([]byte, bool) }

func indexBoxes(b *box, out map[*node]*box) {
	if !b.pseudo && b.n != nil {
		if _, seen := out[b.n]; !seen {
			out[b.n] = b
		}
	}
	for _, k := range b.kids {
		indexBoxes(k, out)
	}
	for _, k := range b.abs {
		indexBoxes(k, out)
	}
	for _, it := range b.inline {
		if it.atomic != nil {
			indexBoxes(it.atomic, out)
		}
	}
}

// ── 측정 ──

type jsonRect struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	W float64 `json:"w"`
	H float64 `json:"h"`
}

type jsonBox struct {
	ID    string     `json:"id"`
	El    string     `json:"el"`
	Rect  jsonRect   `json:"rect"`
	Lines []jsonRect `json:"lines"`
}

type jsonChrome struct {
	Kind string   `json:"kind"`
	Rect jsonRect `json:"rect"`
}

type jsonSlide struct {
	Index   int          `json:"index"`
	W       float64      `json:"w"`
	H       float64      `json:"h"`
	Content jsonRect     `json:"content"`
	Boxes   []jsonBox    `json:"boxes"`
	Chrome  []jsonChrome `json:"chrome"`
}

type jsonMeasurement struct {
	DPR    float64     `json:"dpr"`
	Slides []jsonSlide `json:"slides"`
}

// MeasurementJSON은 지오메트리 패스의 측정 스크립트와 같은 꼴의 JSON이다 (geom.FromMeasurement).
func (d *Document) MeasurementJSON() ([]byte, error) {
	m := jsonMeasurement{DPR: 1}
	for _, s := range d.Slides {
		js := jsonSlide{Index: s.Index, W: s.box.w, H: s.box.h}
		if stages := s.node.find(func(n *node) bool { return n.hasClass(web.StageClass) }); len(stages) > 0 {
			if b := s.boxes[stages[0]]; b != nil {
				f := b.frame(s.box.w)
				js.Content = jsonRect{b.ax + f.l, b.ay + f.t, b.w - f.h(), b.h - f.v()}
			}
		}
		for _, n := range s.node.find(func(n *node) bool { _, ok := n.attrs[web.IDAttr]; return ok }) {
			jb := jsonBox{ID: n.attrs[web.IDAttr], El: n.attrs[web.ElAttr], Lines: []jsonRect{}}
			if b := s.boxes[n]; b != nil {
				jb.Rect = jsonRect{b.ax, b.ay, b.w, b.h}
				jb.Lines = s.lineRects(b)
			}
			js.Boxes = append(js.Boxes, jb)
		}
		for _, n := range s.node.find(func(n *node) bool { return n.hasClass(web.ChromeClass) }) {
			jc := jsonChrome{Kind: n.attrs[web.ChromeKindAttr]}
			if b := s.boxes[n]; b != nil {
				jc.Rect = jsonRect{b.ax, b.ay, b.w, b.h}
			}
			js.Chrome = append(js.Chrome, jc)
		}
		if js.Boxes == nil {
			js.Boxes = []jsonBox{}
		}
		if js.Chrome == nil {
			js.Chrome = []jsonChrome{}
		}
		m.Slides = append(m.Slides, js)
	}
	return json.Marshal(m)
}

// lineRects는 지오메트리 패스의 lineRects와 같은 정의다: 요소의 텍스트 노드 rect(식별자를 가진
// 자손은 건너뛴다)를 모아 세로로 겹치는 것끼리 한 줄로 합친다.
func (s *Slide) lineRects(b *box) []jsonRect {
	var rects []rect
	var walk func(x *box, top bool)
	walk = func(x *box, top bool) {
		if !top && !x.pseudo && x.n != nil && x.n != b.n {
			if _, ok := x.n.attrs[web.IDAttr]; ok {
				return
			}
		}
		if x.pseudo {
			return
		}
		for _, l := range x.lines {
			for _, f := range l.frags {
				if f.atomic != nil {
					walk(f.atomic, false)
				}
			}
			if l.hasText {
				r := l.textRect
				r.x += x.ax
				r.y += x.ay
				rects = append(rects, r)
			}
		}
		for _, k := range x.kids {
			walk(k, false)
		}
		for _, k := range x.abs {
			walk(k, false)
		}
	}
	walk(b, true)
	sort.SliceStable(rects, func(i, j int) bool {
		if rects[i].y != rects[j].y {
			return rects[i].y < rects[j].y
		}
		return rects[i].x < rects[j].x
	})
	var lines []rect
	for _, r := range rects {
		if n := len(lines); n > 0 {
			last := &lines[n-1]
			overlap := math.Min(last.bottom(), r.bottom()) - math.Max(last.y, r.y)
			if overlap > math.Min(last.h, r.h)/2 {
				*last = union(*last, r, true)
				continue
			}
		}
		lines = append(lines, r)
	}
	out := make([]jsonRect, 0, len(lines))
	for _, l := range lines {
		if l.w > 0 && l.h > 0 {
			out = append(out, jsonRect{l.x, l.y, l.w, l.h})
		}
	}
	return out
}

// Measurer는 pipeline.Measurer다 — 브라우저 대신 이 엔진으로 잰다.
type Measurer struct{}

// Measure는 htmlPath의 문서를 배치해 지오메트리를 만든다. 에셋은 같은 디렉토리에서 읽는다.
func (Measurer) Measure(deck *soml.Deck, htmlPath string) (*geom.Geometry, error) {
	data, err := os.ReadFile(htmlPath)
	if err != nil {
		return nil, err
	}
	doc, err := Layout(data, DirAssets(filepath.Dir(htmlPath)))
	if err != nil {
		return nil, err
	}
	raw, err := doc.MeasurementJSON()
	if err != nil {
		return nil, err
	}
	return geom.FromMeasurement(deck, raw)
}

// DirAssets는 디렉토리 아래 상대 경로로 에셋을 읽는다.
func DirAssets(dir string) func(string) ([]byte, bool) {
	return func(name string) ([]byte, bool) {
		data, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(name)))
		return data, err == nil
	}
}
