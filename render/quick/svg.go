package quick

import (
	"bytes"
	"encoding/xml"
	"strings"
)

// parseSVGDoc는 SVG 문서(에셋)를 노드 트리로 읽고 계산값까지 붙인다 (SVG가 아니면 nil).
// 문서 안의 <style>은 HTML과 같은 cascade로 푼다.
func parseSVGDoc(data []byte) *node {
	head := bytes.TrimSpace(data)
	if len(head) == 0 || head[0] != '<' || !bytes.Contains(head[:min(len(head), 2048)], []byte("<svg")) {
		return nil
	}
	dec := xml.NewDecoder(bytes.NewReader(data))
	dec.Strict = false
	dec.AutoClose = xml.HTMLAutoClose
	dec.Entity = xml.HTMLEntity
	var root *node
	var stack []*node
	var sheets []string
	inStyle := false
	for {
		tok, err := dec.Token()
		if err != nil {
			break
		}
		switch t := tok.(type) {
		case xml.StartElement:
			n := &node{tag: t.Name.Local, attrs: map[string]string{}}
			for _, a := range t.Attr {
				key := a.Name.Local
				if a.Name.Space != "" && a.Name.Space != "http://www.w3.org/2000/svg" && key != "href" {
					if strings.HasSuffix(a.Name.Space, "xlink") {
						key = "href"
					} else if a.Name.Space == "xmlns" {
						continue
					}
				}
				n.attrs[key] = a.Value
			}
			n.classes = strings.Fields(n.attrs["class"])
			if len(stack) > 0 {
				parent := stack[len(stack)-1]
				n.parent = parent
				parent.children = append(parent.children, n)
			} else if root == nil {
				root = n
			}
			stack = append(stack, n)
			inStyle = n.tag == "style"
		case xml.EndElement:
			if len(stack) > 0 {
				stack = stack[:len(stack)-1]
			}
			inStyle = false
		case xml.CharData:
			if inStyle {
				sheets = append(sheets, string(t))
			} else if len(stack) > 0 {
				parent := stack[len(stack)-1]
				if parent.tag == "text" || parent.tag == "tspan" {
					parent.children = append(parent.children, &node{text: string(t), parent: parent})
				}
			}
		}
	}
	if root == nil || root.tag != "svg" {
		return nil
	}
	numberNodes(root)
	newCascader(sheets).apply(root)
	return root
}
