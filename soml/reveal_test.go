package soml

import (
	"strings"
	"testing"
)

// `reveal`은 `<li>`가 처음 여는 속성이다 (ADR 0063 결정 1).
//
// 외관은 여전히 컨테이너의 것이고(ADR 0038) 이 속성은 외관이 아니라 **순서**다 — 항목이
// 언제 나타나는지는 항목의 성질이므로 컨테이너로 올릴 수 없다.
func TestListItemOpensRevealOnly(t *testing.T) {
	attrs := ElementAttrs(ListItemElement)
	if len(attrs) != 1 || attrs[0] != "reveal" {
		t.Fatalf("<%s>의 속성 = %v, want [reveal]", ListItemElement, attrs)
	}
}

// reveal은 `<li>` **하나에만** 열린다 (ADR 0063 결정 1의 근거: 짐작으로 여는 속성은
// 진단 문구·문서 목록·PPTX 평탄화 계약을 태그마다 한 벌씩 늘린다).
func TestRevealIsClosedOutsideListItems(t *testing.T) {
	for _, name := range allElementNames() {
		if name == ListItemElement {
			continue
		}
		for _, attr := range ElementAttrs(name) {
			if attr == "reveal" {
				t.Errorf("<%s>가 reveal을 연다 — reveal은 <li>의 속성이다", name)
			}
		}
	}
}

// 값 없는 reveal과 번호를 준 reveal이 모두 파싱된다 — 값 어휘(정수 ≥ 1)는 값 층의 것이고
// 파서는 이름만 본다.
func TestParseRevealAttr(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want string
	}{
		{
			name: "값 없는 reveal",
			src:  `<slide><ul><li>하나</li><li reveal>둘</li></ul></slide>`,
			want: `ul(li("하나") li[reveal=]("둘"))`,
		},
		{
			name: "번호를 준 reveal",
			src:  `<slide><ul><li reveal="2">둘째로</li><li reveal="1">첫째로</li></ul></slide>`,
			want: `ul(li[reveal=2]("둘째로") li[reveal=1]("첫째로"))`,
		},
		{
			name: "ol도 같다",
			src:  `<slide><ol><li reveal>단계</li></ol></slide>`,
			want: `ol(li[reveal=]("단계"))`,
		},
		{
			name: "인라인과 함께",
			src:  `<slide><ul><li reveal>배포 <strong>자동화</strong></li></ul></slide>`,
			want: `ul(li[reveal=]("배포 " strong("자동화")))`,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			doc, diags := Parse("slides/001_reveal.slide", tc.src)
			if len(diags) != 0 {
				t.Fatalf("진단이 나왔다:\n%s", diags.Err())
			}
			if got := dumpNodes(doc.Content); got != tc.want {
				t.Errorf("트리 = %s, want %s", got, tc.want)
			}
		})
	}
}

// reveal을 다른 태그에 쓰면 그 태그의 속성 목록이 나오고, `<li>`의 다른 속성은 이제
// `<li>`의 목록(reveal 하나)을 보여준다 — 속성이 하나라도 열린 태그의 진단 문구다.
func TestParseRevealErrors(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want string
	}{
		{
			name: "블록에 reveal",
			src:  `<slide><p reveal>본문</p></slide>`,
			want: `1:11 <p>에 알 수 없는 속성 reveal (쓸 수 있는 속성: variant, size,`,
		},
		{
			name: "리스트 컨테이너에 reveal",
			src:  `<slide><ul reveal><li>항목</li></ul></slide>`,
			want: `1:12 <ul>에 알 수 없는 속성 reveal (쓸 수 있는 속성: size,`,
		},
		{
			name: "li의 외관 속성은 여전히 닫혀 있다",
			src:  `<slide><ul><li size="20">항목</li></ul></slide>`,
			want: `1:16 <li>에 알 수 없는 속성 size (쓸 수 있는 속성: reveal)`,
		},
		{
			name: "li의 variant도 닫혀 있다",
			src:  `<slide><ul><li variant="lead">항목</li></ul></slide>`,
			want: `1:16 <li>에 알 수 없는 속성 variant (쓸 수 있는 속성: reveal)`,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, diags := Parse("slides/001_reveal.slide", tc.src)
			if len(diags) != 1 {
				t.Fatalf("진단 %d개, want 1:\n%s", len(diags), diags.Err())
			}
			pos, message, _ := strings.Cut(tc.want, " ")
			if got := diags[0].Span.Start.String(); got != pos {
				t.Errorf("위치 = %s, want %s (%s)", got, pos, diags[0].Message)
			}
			if !strings.Contains(diags[0].Message, message) {
				t.Errorf("메시지 = %q, want %q 포함", diags[0].Message, message)
			}
		})
	}
}
