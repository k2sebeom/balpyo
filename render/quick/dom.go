// Package quick은 브라우저 없이 슬라이드를 그리는 내장 렌더러다.
//
// 입력은 덱 모델이 아니라 **웹 렌더러가 낸 HTML**이다. 웹이 기준 타깃이므로, 웹이 내는
// 마크업과 CSS를 그대로 읽어 배치하면 웹 렌더러의 해석(inline style, 슬롯 grid, 도형 ink)을
// 두 번 구현하지 않는다 — 웹이 바뀌면 이 렌더러도 같이 따라간다.
//
// 그래서 이 패키지는 balpyo가 내는 HTML에 한정된 작은 브라우저다: CSS cascade(@layer ·
// 선택자 · var() · calc()), block · inline · flex · grid · table 배치, 텍스트 셰이핑과 줄바꿈,
// SVG ink 래스터화. 일반 웹 페이지를 그리려는 것이 아니므로 웹 렌더러가 쓰지 않는 기능은
// 없다.
//
// 결과는 두 가지다. (1) geom.Geometry — 지오메트리 패스와 같은 측정 결과라 배치 진단과
// PPTX가 그대로 소비한다. (2) 슬라이드별 PNG. 브라우저와 줄바꿈 · 글리프가 같지는 않다 —
// 빠른 반복용 미리보기이고, 기준은 여전히 브라우저 측정이다.
package quick

import (
	"bytes"
	"fmt"
	"strings"

	"golang.org/x/net/html"
)

// node는 DOM 노드 하나다. tag가 비면 텍스트 노드다.
type node struct {
	tag      string
	text     string
	attrs    map[string]string
	classes  []string
	parent   *node
	children []*node

	order int // 문서 순번 (그리기 순서)

	style  *style // cascade가 채운다
	before *style // ::before (content가 있을 때만)
}

func (n *node) isText() bool { return n.tag == "" }

func (n *node) attr(name string) (string, bool) {
	v, ok := n.attrs[name]
	return v, ok
}

func (n *node) hasClass(class string) bool {
	for _, c := range n.classes {
		if c == class {
			return true
		}
	}
	return false
}

// elementIndex는 형제 요소 사이의 순번과 형제 요소 수다 (텍스트 노드는 세지 않는다).
func (n *node) elementIndex() (int, int) {
	if n.parent == nil {
		return 0, 1
	}
	idx, count := 0, 0
	for _, sib := range n.parent.children {
		if sib.isText() {
			continue
		}
		if sib == n {
			idx = count
		}
		count++
	}
	return idx, count
}

// find는 깊이 우선으로 조건을 만족하는 요소를 모은다.
func (n *node) find(match func(*node) bool) []*node {
	var out []*node
	var walk func(*node)
	walk = func(cur *node) {
		if !cur.isText() && match(cur) {
			out = append(out, cur)
		}
		for _, c := range cur.children {
			walk(c)
		}
	}
	walk(n)
	return out
}

// parseHTML은 문서를 노드 트리로 읽고 <style> 내용을 문서 순서로 모은다.
func parseHTML(src []byte) (*node, []string, error) {
	doc, err := html.Parse(bytes.NewReader(src))
	if err != nil {
		return nil, nil, fmt.Errorf("cannot parse HTML: %w", err)
	}
	var sheets []string
	var convert func(h *html.Node, parent *node) *node
	convert = func(h *html.Node, parent *node) *node {
		switch h.Type {
		case html.TextNode:
			return &node{text: h.Data, parent: parent}
		case html.ElementNode:
			tag := h.Data
			if tag == "style" || tag == "script" || tag == "head" {
				return nil
			}
			n := &node{tag: tag, attrs: make(map[string]string, len(h.Attr)), parent: parent}
			for _, a := range h.Attr {
				key := a.Key
				if a.Namespace != "" {
					key = a.Namespace + ":" + a.Key
				}
				n.attrs[key] = a.Val
			}
			n.classes = strings.Fields(n.attrs["class"])
			for c := h.FirstChild; c != nil; c = c.NextSibling {
				if child := convert(c, n); child != nil {
					n.children = append(n.children, child)
				}
			}
			return n
		}
		return nil
	}
	var root *node
	for c := doc.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode && c.Data == "html" {
			root = convert(c, nil)
		}
	}
	// <style>은 놓인 자리(head · body)와 상관없이 문서 순서가 cascade 순서다.
	var collect func(h *html.Node)
	collect = func(h *html.Node) {
		if h.Type == html.ElementNode && h.Data == "style" {
			var b strings.Builder
			for c := h.FirstChild; c != nil; c = c.NextSibling {
				b.WriteString(c.Data)
			}
			sheets = append(sheets, b.String())
		}
		for c := h.FirstChild; c != nil; c = c.NextSibling {
			collect(c)
		}
	}
	collect(doc)
	if root == nil {
		return nil, nil, fmt.Errorf("HTML has no <html> element")
	}
	numberNodes(root)
	return root, sheets, nil
}

// numberNodes는 문서 순번을 매긴다.
func numberNodes(root *node) {
	i := 0
	var walk func(n *node)
	walk = func(n *node) {
		n.order = i
		i++
		for _, c := range n.children {
			walk(c)
		}
	}
	walk(root)
}
