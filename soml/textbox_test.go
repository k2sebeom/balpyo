package soml

import (
	"strings"
	"testing"
)

// 텍스트 요소의 **가로 크기 속성** (P1.3 · ADR 0043 — 도그푸딩 D14).
//
// 어휘를 연 근거: 캔버스에서 특정 폭 안에 라벨을 가운데 두려면 그 폭을 가진 박스가 필요한데,
// `<text>`가 폭을 못 받아 저자가 보이지 않는 도형으로 감쌌다
// (`examples/incident-postmortem/slides/008_detection.slide`의 라벨 10개 → 태그 20개).
func TestParseTextInlineSize(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want string
	}{
		{
			name: "text의 가로 크기 3종",
			src:  `<slide><text w="140" min-w="80" max-w="50%">라벨</text></slide>`,
			want: `text[w=140 min-w=80 max-w=50%]("라벨")`,
		},
		{
			// 블록도 같은 속성을 받는다 — 등가 표(`<p>` ≡ `<text role="body">`)가 속성 표에서도
			// 성립해야 한다 (블록이 외관 5종을 받는 것과 같은 근거).
			name: "블록의 가로 크기",
			src:  `<slide><h2 w="fill">제목</h2><p max-w="720">본문</p></slide>`,
			want: `h2[w=fill]("제목") p[max-w=720]("본문")`,
		},
		{
			// 리스트 컨테이너도 받는다 (ADR 0038의 대칭 — 외관이 컨테이너에 붙는 것과 같은 자리).
			name: "리스트 컨테이너의 가로 크기",
			src:  `<slide><ul w="480"><li>항목</li></ul></slide>`,
			want: `ul[w=480](li("항목"))`,
		},
		{
			// 외관·크기·자식 배치·슬롯이 한 요소에 함께 온다 — 채널이 서로를 모르는 것이
			// 이 언어의 경계다.
			name: "외관과 크기와 배치가 함께",
			src:  `<slide><text size="16" w="140" align="center" grow="1">x</text></slide>`,
			want: `text[size=16 w=140 align=center grow=1]("x")`,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			doc := parseOK(t, tc.src)
			if got := dumpNodes(doc.Content); got != tc.want {
				t.Errorf("트리 = %s\nwant %s", got, tc.want)
			}
		})
	}
}

// 세로 축(`h` 계열)과 `pad`·`ratio`는 **닫혀 있다** (ADR 0043).
//
//   - `h`: 박스 안 세로 정렬 채널(`anchor`)이 도형 안에서만 뜻을 가지므로, 높이만 열면 저자가
//     닿을 수 없는 공간이 생긴다. 레이아웃이 본문 높이를 소유하는가(D17)가 P1.4에서 정해질
//     값이고 그 결정이 `h`의 뜻을 바꾼다 — 블록 축은 그때 함께 연다.
//   - `pad`: PPTX 텍스트 박스의 인셋 0 계약이 `.balpyo-text`의 padding 0에 달려 있다.
//   - `ratio`: 텍스트 박스의 높이는 글줄 수가 정하므로 종횡비에 대응이 없다.
//
// 진단이 **속성 목록을 실어** 저자가 무엇을 쓸 수 있는지 그 자리에서 읽게 한다.
func TestParseTextBlockAxisStaysClosed(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want []string
	}{
		{
			name: "text에는 h가 없다",
			src:  `<slide><text h="56">x</text></slide>`,
			want: []string{`1:14 <text>에 알 수 없는 속성 h (쓸 수 있는 속성: role, variant, size, color, align, anchor, w, min-w, max-w, grow, shrink, basis, span, self, order, at, x, y, style, slot)`},
		},
		{
			name: "text에는 min-h·max-h가 없다",
			src:  `<slide><text min-h="20">x</text></slide>`,
			want: []string{"1:14 <text>에 알 수 없는 속성 min-h"},
		},
		{
			name: "text에는 pad가 없다",
			src:  `<slide><text pad="md">x</text></slide>`,
			want: []string{"1:14 <text>에 알 수 없는 속성 pad"},
		},
		{
			name: "text에는 ratio가 없다",
			src:  `<slide><text ratio="16:9">x</text></slide>`,
			want: []string{"1:14 <text>에 알 수 없는 속성 ratio"},
		},
		{
			name: "블록도 같다",
			src:  `<slide><p h="120">x</p></slide>`,
			want: []string{`1:11 <p>에 알 수 없는 속성 h (쓸 수 있는 속성: variant, size, color, align, anchor, w, min-w, max-w, grow, shrink, basis, span, self, order, at, x, y, style, slot)`},
		},
		{
			name: "리스트 컨테이너도 같다",
			src:  `<slide><ul h="200"><li>x</li></ul></slide>`,
			want: []string{`1:12 <ul>에 알 수 없는 속성 h (쓸 수 있는 속성: size, color, align, anchor, w, min-w, max-w, grow, shrink, basis, span, self, order, at, x, y, style, slot)`},
		},
		{
			// `<li>`는 속성을 받지 않는다 (ADR 0038) — 크기도 컨테이너의 것이다.
			name: "li에는 크기도 없다",
			src:  `<slide><ul><li w="100">x</li></ul></slide>`,
			want: []string{"1:16 <li>에는 속성을 쓸 수 없다 (w)"},
		},
		{
			// 인라인은 박스가 아니라 런이다.
			name: "인라인에는 크기가 없다",
			src:  `<slide><p><span w="100">x</span></p></slide>`,
			want: []string{"1:17 <span>에 알 수 없는 속성 w (쓸 수 있는 속성: color, weight)"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, diags := Parse("slides/001_title.slide", tc.src)
			for i, want := range tc.want {
				pos, message, _ := strings.Cut(want, " ")
				if i >= len(diags) {
					t.Errorf("진단 %d개뿐이다, want %q", len(diags), want)
					break
				}
				if got := diags[i].Span.Start.String(); got != pos {
					t.Errorf("진단[%d] 위치 = %s, want %s (%s)", i, got, pos, diags[i].Message)
				}
				if !strings.Contains(diags[i].Message, message) {
					t.Errorf("진단[%d] 메시지 = %q, want %q 포함", i, diags[i].Message, message)
				}
			}
			if len(diags) != len(tc.want) {
				t.Errorf("진단 %d개, want %d개:\n%s", len(diags), len(tc.want), diags.Err())
			}
		})
	}
}

// 텍스트의 크기 목록은 박스 카탈로그의 **부분집합**이고 순서도 같다 — 목록이 두 벌이 되면
// 박스 속성이 하나 늘 때 같은 표기가 두 뜻을 갖는다 (컨테이너와 도형이 BoxAttrs를 공유하는
// 것과 같은 근거).
func TestTextSizeAttrsAreBoxSubset(t *testing.T) {
	box := BoxAttrs()
	text := TextSizeAttrs()
	at := 0
	for _, name := range text {
		found := false
		for ; at < len(box); at++ {
			if box[at] == name {
				found = true
				at++
				break
			}
		}
		if !found {
			t.Fatalf("TextSizeAttrs의 %q가 BoxAttrs에 없거나 순서가 다르다 (text=%v box=%v)", name, text, box)
		}
	}
	// 세로 축과 pad·ratio는 부분집합에 들지 않는다 (ADR 0043).
	for _, closed := range []string{"pad", "h", "min-h", "max-h", "ratio"} {
		for _, name := range text {
			if name == closed {
				t.Errorf("%q는 텍스트에 닫혀 있어야 한다", closed)
			}
		}
	}
}
