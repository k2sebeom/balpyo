package core

import (
	"strings"
	"testing"

	"github.com/k2sebeom/balpyo/soml"
)

// listItemStyles는 덱의 모든 `<li>` 확정 외관을 문서 순서로 모은다.
func listItemStyles(t *testing.T, deck *soml.Deck) []*soml.TextStyle {
	t.Helper()
	var styles []*soml.TextStyle
	var walk func(node soml.Node)
	walk = func(node soml.Node) {
		elem, ok := node.(*soml.Element)
		if !ok {
			return
		}
		if soml.KindOf(elem.Name) == soml.KindListItem {
			style := elem.TextStyle()
			if style == nil {
				t.Fatalf("<%s>에 해석된 외관이 없다 (%s)", elem.Name, elem.Span)
			}
			styles = append(styles, style)
		}
		for _, child := range elem.Children {
			walk(child)
		}
	}
	for _, slide := range deck.Slides {
		for _, node := range slide.Content {
			walk(node)
		}
	}
	return styles
}

// 컨테이너의 외관 속성은 **항목들이 함께 받는 패치**다: 목록 하나의
// 타이포그래피는 목록 단위의 성질이므로, 저자가 컨테이너에 한 번 쓰면 모든 항목이 같은
// 값에 도달해야 한다 — 항목마다 값이 갈리면 그것은 목록이 아니라 문단 여럿이다.
func TestResolveListAppearanceFlowsToItems(t *testing.T) {
	deck, diags := resolveSlide(t, `<slide>
  <ul size="20" color="muted" align="center" anchor="center">
    <li>하나</li>
    <li>둘</li>
  </ul>
</slide>
`)
	if len(diags) != 0 {
		t.Fatalf("진단이 없어야 한다: %v", diags)
	}
	styles := listItemStyles(t, deck)
	if len(styles) != 2 {
		t.Fatalf("<li> 외관 %d개, want 2개", len(styles))
	}
	body := *firstStyle(t, "<slide>\n  <text>내용</text>\n</slide>\n")
	want := body
	want.Size = 20
	want.Color = soml.TokenColor("muted")
	want.Align = soml.AlignCenter
	want.Anchor = soml.AnchorCenter
	for i, got := range styles {
		if *got != want {
			t.Errorf("<li>[%d] = %+v\nwant %+v (body 기본값에 컨테이너 패치)", i, *got, want)
		}
	}
}

// 컨테이너 자신은 **확정 텍스트 외관을 갖지 않는다**. 컨테이너는 텍스트를 담지
// 않으므로(자식은 `<li>`뿐) 소비자가 없고, 채워 두면 두 렌더러가 "컨테이너의 값"과 "항목의
// 값" 중 어느 쪽이 진실인지 각자 고르게 된다. 배치 채널은 그대로 붙는다.
func TestResolveListContainerHasNoTextStyle(t *testing.T) {
	deck, diags := resolveSlide(t, `<slide>
  <row>
    <ul size="20" basis="360"><li>하나</li></ul>
  </row>
</slide>
`)
	if len(diags) != 0 {
		t.Fatalf("진단이 없어야 한다: %v", diags)
	}
	ul := findElement(t, deck, "ul")
	if style := ul.TextStyle(); style != nil {
		t.Errorf("<ul>에 확정 텍스트 외관이 있다: %+v", *style)
	}
	placement := ul.Placement()
	if placement == nil || placement.Basis == nil || placement.Basis.Value != 360 {
		t.Errorf("<ul>의 배치 = %+v, want basis 360px (외관과 배치는 다른 채널이다)", placement)
	}
}

// 값 진단은 컨테이너의 **값을 가리킨다** — 블록과 같은 정밀도다 (값 층이 한 곳이므로
// 어휘·문구·위치 규칙이 갈라질 자리가 없다). 항목이 여럿이어도 진단은 하나다: 저자가 쓴
// 속성이 하나이기 때문이다.
func TestResolveListValueDiagnostics(t *testing.T) {
	_, diags := resolveSlide(t, "<slide>\n  <ul color=\"brand\"><li>하나</li><li>둘</li></ul>\n</slide>\n")
	if len(diags) != 1 {
		t.Fatalf("진단 %d개, want 1개: %v", len(diags), diags)
	}
	diag := diags[0]
	if diag.Severity != soml.SeverityError || !strings.Contains(diag.Message, "테마에 없다") {
		t.Errorf("진단 = %q, want 색 토큰 에러", diag.String())
	}
	if diag.Span.Start.Line != 2 || diag.Span.Start.Col != 14 {
		t.Errorf("위치 = %s, want 2:14 (값을 가리켜야 한다)", diag.Span.Start)
	}
}

// 두 목록은 서로 새지 않는다 — 패치는 컨테이너에서 자기 항목으로만 내려간다.
func TestResolveListPatchStaysInItsList(t *testing.T) {
	deck, diags := resolveSlide(t, `<slide>
  <ul size="20"><li>하나</li></ul>
  <ol><li>첫째</li></ol>
</slide>
`)
	if len(diags) != 0 {
		t.Fatalf("진단이 없어야 한다: %v", diags)
	}
	styles := listItemStyles(t, deck)
	if len(styles) != 2 {
		t.Fatalf("<li> 외관 %d개, want 2개", len(styles))
	}
	if styles[0].Size != 20 {
		t.Errorf("<ul>의 항목 Size = %v, want 20", styles[0].Size)
	}
	body := *firstStyle(t, "<slide>\n  <text>내용</text>\n</slide>\n")
	if *styles[1] != body {
		t.Errorf("<ol>의 항목 = %+v, want body 기본값 %+v", *styles[1], body)
	}
}
