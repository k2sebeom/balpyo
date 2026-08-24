package soml

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// 도형의 정상 트리 — 태그·속성 넷(고유·외관·박스·자식 배치)·자식 어휘·자기 닫힘이 파싱을
// 그대로 통과한다. dumpNodes의 표현은 subset_test.go와 공유한다.
func TestParseShape(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want string
	}{
		{
			name: "고유 속성 — type과 id",
			src:  `<slide><shape type="cylinder" id="db"/></slide>`,
			want: `shape[type=cylinder id=db]`,
		},
		{
			name: "외관 속성 8종",
			src: `<slide><shape type="rounded-rect" variant="card-header" fill="surface" ` +
				`border="1 solid muted" radius="lg" shadow="soft" opacity="0.8" rotate="45" flip="h"/></slide>`,
			want: `shape[type=rounded-rect variant=card-header fill=surface border=1 solid muted ` +
				`radius=lg shadow=soft opacity=0.8 rotate=45 flip=h]`,
		},
		{
			// 도형은 그 자체가 그림이라 크기가 필요하다 (ShapeAttrs의 주석) — 박스 속성은
			// 레이아웃 컨테이너와 같은 목록이다 (BoxAttrs).
			name: "박스 속성 — pad와 크기·비율",
			src:  `<slide><shape type="rect" pad="md" w="fill" h="220" min-w="120" min-h="80" max-w="480" max-h="320" ratio="1:1"/></slide>`,
			want: `shape[type=rect pad=md w=fill h=220 min-w=120 min-h=80 max-w=480 max-h=320 ratio=1:1]`,
		},
		{
			name: "자식 배치 속성 — 도형은 컨테이너의 자식이 될 수 있다",
			src:  `<slide><grid cols="4"><shape type="rect" span="2" grow="1" self="center" order="2"/></grid></slide>`,
			want: `grid[cols=4](shape[type=rect span=2 grow=1 self=center order=2])`,
		},
		{
			name: "도형도 style 탈출구를 받는다",
			src:  `<slide><row><shape type="rect" style="order: 2"/></row></slide>`,
			want: `row(shape[type=rect style=order: 2])`,
		},
		{
			name: "도형 안 text",
			src:  `<slide><shape type="rect"><text anchor="center">배지</text></shape></slide>`,
			want: `shape[type=rect](text[anchor=center]("배지"))`,
		},
		{
			// 개수를 문법으로 세지 않는다 — 자식 여럿은 PPTX에서 txBody의 문단 여럿이다.
			name: "도형 안 블록 여러 개",
			src:  `<slide><shape type="rounded-rect" pad="md"><h3>제목</h3><p>본문</p></shape></slide>`,
			want: `shape[type=rounded-rect pad=md](h3("제목") p("본문"))`,
		},
		{
			name: "도형 안 텍스트의 인라인 서식",
			src:  `<slide><shape type="rect"><p>배포 <strong>자동화</strong></p></shape></slide>`,
			want: `shape[type=rect](p("배포 " strong("자동화")))`,
		},
		{
			name: "도형 안 pre는 공백을 보존한다",
			src:  "<slide><shape type=\"rect\"><pre>a\n  b</pre></shape></slide>",
			want: "shape[type=rect](pre(\"a\\n  b\"))",
		},
		{
			name: "슬라이드 직속 도형 — 세로 스택의 한 칸",
			src:  `<slide><h2>표제</h2><shape type="rect" h="4" fill="muted"/><p>본문</p></slide>`,
			want: `h2("표제") shape[type=rect h=4 fill=muted] p("본문")`,
		},
		{
			name: "컨테이너 안 도형 — 카드 그리드",
			src:  `<slide><grid cols="2" gap="lg"><col gap="none"><shape type="rounded-rect" fill="accent1"><text>머리</text></shape><shape type="rect"><p>몸</p></shape></col></grid></slide>`,
			want: `grid[cols=2 gap=lg](col[gap=none](shape[type=rounded-rect fill=accent1](text("머리")) shape[type=rect](p("몸"))))`,
		},
		{
			// 자기 닫힘은 void가 아니어도 유효하다 (파서의 기존 동작) — PRD §7.6의 예제
			// 표기이고, 자식 없는 `<shape></shape>`도 같은 뜻이다.
			name: "빈 도형 — 자기 닫힘과 짝 태그가 같다",
			src:  `<slide><shape type="rect"/><shape type="ellipse"></shape></slide>`,
			want: `shape[type=rect] shape[type=ellipse]`,
		},
		{
			// 파서는 값을 보지 않는다 — 12종 밖의 type도 문법은 통과하고 진단은 값 층(항목 02)의
			// 몫이다. 이 비대칭이 의도다 (Element.Attrs의 주석).
			name: "12종 밖의 type도 파서는 통과시킨다",
			src:  `<slide><shape type="dodecagon"/></slide>`,
			want: `shape[type=dodecagon]`,
		},
		{
			// type 없는 도형도 문법은 통과한다 — "속성이 있어야 한다"는 값의 요구이고,
			// href 없는 <a>를 파서가 통과시키는 것과 같은 근거다.
			name: "type 없는 도형도 파서는 통과시킨다",
			src:  `<slide><shape/></slide>`,
			want: `shape`,
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

// `anchor`는 `<text>`와 블록 태그에만 열린다 — 앵커가 보이는 자리가 도형 안이고(AnchorAttr),
// 등가 표(`<p>` ≡ `<text role="body">`)의 대칭을 지켜 블록도 같이 받는다.
func TestParseAnchor(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want string
	}{
		{
			name: "text의 anchor",
			src:  `<slide><shape type="rect"><text anchor="end">아래</text></shape></slide>`,
			want: `shape[type=rect](text[anchor=end]("아래"))`,
		},
		{
			name: "블록의 anchor",
			src:  `<slide><shape type="rect"><h3 anchor="center">가운데</h3><p anchor="start">위</p></shape></slide>`,
			want: `shape[type=rect](h3[anchor=center]("가운데") p[anchor=start]("위"))`,
		},
		{
			// 도형 밖에서도 문법은 유효하다 — 슬라이드 직속 텍스트 박스에서 무해할 뿐이고,
			// 자리에 따라 속성 어휘가 갈라지면 저자가 태그를 옮길 때마다 문법이 바뀐다.
			name: "슬라이드 직속 text의 anchor도 문법은 유효하다",
			src:  `<slide><text anchor="center">표제</text></slide>`,
			want: `text[anchor=center]("표제")`,
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

// 요소의 Span은 여는 `<`부터 닫는 태그의 `>`까지다 — 도형은 잰 rect가 곧 도형 좌표이므로
// 진단이 저자의 태그를 정확히 가리켜야 한다.
func TestParseShapeSpans(t *testing.T) {
	doc := parseOK(t, "<slide>\n  <shape type=\"rect\">\n    <text>배지</text>\n  </shape>\n</slide>")
	shape := elementAt(t, doc.Content, 0)
	if got, want := shape.Span.Start.String()+"-"+shape.Span.End.String(), "2:3-4:11"; got != want {
		t.Errorf("<shape> Span = %s, want %s", got, want)
	}
	text := elementAt(t, shape.Children, 0)
	if got, want := text.Span.Start.String()+"-"+text.Span.End.String(), "3:5-3:20"; got != want {
		t.Errorf("<text> Span = %s, want %s", got, want)
	}
	// 자기 닫힘 도형도 같은 계약이다 (여는 `<`부터 `/>`까지).
	doc = parseOK(t, "<slide>\n  <shape type=\"rect\" id=\"db\"/>\n</slide>")
	shape = elementAt(t, doc.Content, 0)
	if got, want := shape.Span.Start.String()+"-"+shape.Span.End.String(), "2:3-2:31"; got != want {
		t.Errorf("자기 닫힘 <shape> Span = %s, want %s", got, want)
	}
}

// 에러 케이스 전수 — 진단의 line:col과 문구까지 단정한다.
func TestParseShapeErrors(t *testing.T) {
	tests := []struct {
		name string
		src  string
		// want의 각 원소는 "line:col 메시지조각"이다.
		want []string
	}{
		{
			// 도형의 콘텐츠 모델은 modelElements다 — 슬라이드에 그냥 쓴 글자가 어떤 역할의
			// 텍스트인지 정할 방법이 없다.
			name: "도형 안의 문자 데이터",
			src:  `<slide><shape type="rect">배지</shape></slide>`,
			want: []string{"1:27 <shape> 안에 문자 데이터를 직접 쓸 수 없다 (<text>로 감싼다)"},
		},
		{
			// 도형 안 리스트는 불릿 문단 조립과 도형 txBody 조립을 합치는 일이다 (shapeChildren).
			name: "도형 안의 리스트",
			src:  `<slide><shape type="rect"><ul><li>항목</li></ul></shape></slide>`,
			want: []string{"1:28 <shape>의 자식으로 <ul>는 올 수 없다 (쓸 수 있는 태그: <text>, <h1>, <h2>, <h3>, <p>, <blockquote>, <pre>)"},
		},
		{
			// 도형 안의 배치는 겹침(I6)·편집 단위(I9)의 자리다.
			name: "도형 안의 레이아웃 컨테이너",
			src:  `<slide><shape type="rect"><row><text>a</text></row></shape></slide>`,
			want: []string{"1:28 <shape>의 자식으로 <row>는 올 수 없다"},
		},
		{
			name: "도형 안의 spacer",
			src:  `<slide><shape type="rect"><spacer/></shape></slide>`,
			want: []string{"1:28 <shape>의 자식으로 <spacer>는 올 수 없다"},
		},
		{
			// 도형이 txBody 하나를 갖는다는 이번 매핑과 정면으로 부딪힌다.
			name: "도형 안의 도형",
			src:  `<slide><shape type="rect"><shape type="ellipse"/></shape></slide>`,
			want: []string{"1:28 <shape>의 자식으로 <shape>는 올 수 없다"},
		},
		{
			// 노트는 콘텐츠가 아니라 슬라이드 메타데이터다 (SlideDoc.Notes).
			name: "도형 안의 note",
			src:  `<slide><shape type="rect"><note>메모</note></shape></slide>`,
			want: []string{"1:28 <shape>의 자식으로 <note>는 올 수 없다"},
		},
		{
			name: "도형 안의 li",
			src:  `<slide><shape type="rect"><li>항목</li></shape></slide>`,
			want: []string{"1:28 <li>는 <ul>/<ol>의 자식으로만 쓸 수 있다"},
		},
		{
			name: "도형 안의 인라인 태그",
			src:  `<slide><shape type="rect"><strong>굵게</strong></shape></slide>`,
			want: []string{"1:28 <shape>의 자식으로 <strong>는 올 수 없다"},
		},
		{
			// stroke/cap/dash는 line·arrow의 것이다 (shapeAppearanceAttrs) — 도형에 받아
			// 놓고 버리지 않는다. 진단이 도형의 속성 표 전체를 순서대로 싣는다.
			name: "도형에 stroke는 없다",
			src:  `<slide><shape type="rect" stroke="2 accent1"/></slide>`,
			want: []string{`1:27 <shape>에 알 수 없는 속성 stroke (쓸 수 있는 속성: type, id, variant, fill, border, radius, shadow, opacity, rotate, flip, pad, w, h, min-w, min-h, max-w, max-h, ratio, grow, shrink, basis, span, self, order, style)`},
		},
		{
			name: "도형에 dash는 없다",
			src:  `<slide><shape type="rect" dash="dashed"/></slide>`,
			want: []string{"1:27 <shape>에 알 수 없는 속성 dash"},
		},
		{
			// 도형은 텍스트가 아니다 — 역할·글자 크기는 안쪽 텍스트 요소가 소유한다.
			name: "도형에 role은 없다",
			src:  `<slide><shape type="rect" role="body"/></slide>`,
			want: []string{"1:27 <shape>에 알 수 없는 속성 role"},
		},
		{
			name: "도형에 cols는 없다",
			src:  `<slide><shape type="rect" cols="2"/></slide>`,
			want: []string{"1:27 <shape>에 알 수 없는 속성 cols"},
		},
		{
			name: "모르는 도형 속성",
			src:  `<slide><shape type="rect" fil="surface"/></slide>`,
			want: []string{"1:27 <shape>에 알 수 없는 속성 fil"},
		},
		{
			// `<arrow>`·`<line>`·`<group>`은 Iteration 7·9다 — 렌더 매핑 없는 태그를 어휘에
			// 열면 저자가 쓴 것이 조용히 무시된다. `<image>`는 Iteration 5에 열렸다
			// (image_test.go).
			name: "arrow는 아직 어휘 밖이다",
			src:  `<slide><shape type="rect" id="a"/><arrow from="a" to="b"/></slide>`,
			want: []string{"1:36 알 수 없는 태그 <arrow>"},
		},
		{
			name: "line은 아직 어휘 밖이다",
			src:  `<slide><line dir="h"/></slide>`,
			want: []string{"1:9 알 수 없는 태그 <line>"},
		},
		{
			name: "group은 아직 어휘 밖이다",
			src:  `<slide><group><shape type="rect"/></group></slide>`,
			want: []string{"1:9 알 수 없는 태그 <group>"},
		},
		{
			// <li>는 리스트 컨테이너의 자식이지 도형 안 문단이 아니다 — 앵커를 열면
			// 웹에서만 뜻을 갖는 선언이 된다 (childPlacementAttrs와 같은 근거).
			name: "li에는 anchor가 없다",
			src:  `<slide><ul><li anchor="center">항목</li></ul></slide>`,
			want: []string{"1:16 <li>에는 속성을 쓸 수 없다 (anchor)"},
		},
		{
			name: "인라인에는 anchor가 없다",
			src:  `<slide><p><span anchor="center">글자</span></p></slide>`,
			want: []string{"1:17 <span>에 알 수 없는 속성 anchor (쓸 수 있는 속성: color, weight)"},
		},
		{
			name: "리스트 컨테이너에는 anchor가 없다",
			src:  `<slide><ul anchor="center"><li>항목</li></ul></slide>`,
			want: []string{"1:12 <ul>에 알 수 없는 속성 anchor (쓸 수 있는 속성: grow, shrink, basis, span, self, order, style)"},
		},
		{
			// 컨테이너는 자기 외관을 갖지 않는다 — 앵커도 텍스트의 것이다.
			name: "레이아웃 컨테이너에는 anchor가 없다",
			src:  `<slide><row anchor="center"><p>a</p></row></slide>`,
			want: []string{"1:13 <row>에 알 수 없는 속성 anchor"},
		},
		{
			// 도형의 세로 앵커는 안쪽 텍스트가 소유한다 — 도형 자신에게 열면 같은 뜻의
			// 선언이 두 자리에 생기고 어느 쪽이 이기는지 정할 근거가 없다.
			name: "도형에는 anchor가 없다",
			src:  `<slide><shape type="rect" anchor="center"/></slide>`,
			want: []string{"1:27 <shape>에 알 수 없는 속성 anchor"},
		},
		{
			name: "닫히지 않은 도형",
			src:  "<slide>\n  <shape type=\"rect\">\n    <text>배지</text>\n</slide>",
			want: []string{
				"2:4 <shape> 태그가 </slide>보다 먼저 닫혀야 한다",
			},
		},
		{
			name: "짝이 없는 닫는 도형 태그",
			src:  `<slide></shape></slide>`,
			want: []string{"1:8 </shape>와 짝이 맞는 여는 태그가 없다"},
		},
		{
			// 도형은 void가 아니다 — 자식을 가질 수 있고, 그래서 void 진단이 붙지 않는다.
			// 이 케이스는 진단 0건을 단정하는 자리다 (want가 비어 있다).
			name: "도형은 void가 아니다",
			src:  `<slide><shape type="rect"><text>a</text></shape></slide>`,
			want: nil,
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

// 종류 카탈로그는 다섯 층(파서·값 층·geom·web·pptx)이 공유하는 한 표다 — 표가 흔들리면
// 좌표가 엉뚱한 요소에 붙는다. `<shape>`가 **다른 종류**인 것이 계약이다 (KindShape).
func TestShapeKind(t *testing.T) {
	if got := KindOf(ShapeElement); got != KindShape {
		t.Errorf("KindOf(%q) = %v, want KindShape", ShapeElement, got)
	}
	// 종류만 있고 문법이 없는 이름은 "알 수 없는 태그"로 진단되면서 렌더러는 종류로
	// 분기하는 어긋난 상태가 된다.
	if _, known := elementSpecs[ShapeElement]; !known {
		t.Errorf("<%s>가 문법 표(elementSpecs)에 없다", ShapeElement)
	}
	// 도형은 void가 아니다 (텍스트를 담는다).
	if IsVoidElement(ShapeElement) {
		t.Errorf("IsVoidElement(%q) = true — 도형은 자식을 갖는다", ShapeElement)
	}
	// 도형은 텍스트 역할을 갖지 않는다 — 역할은 안쪽 텍스트 요소가 소유한다.
	if role, ok := TagTextRole(ShapeElement); ok {
		t.Errorf("TagTextRole(%q) = %q — 도형은 역할 표에 없어야 한다", ShapeElement, role)
	}
	// Iteration 6·7·9의 컴포넌트는 카탈로그에도 어휘에도 없어야 한다. `<image>`는
	// Iteration 5에 열렸다 (image.go — TestImageKind가 그 계약을 본다).
	for _, name := range []string{"stack", "arrow", "line", "group", "icon", "table", "chart"} {
		if KindOf(name) != KindNone {
			t.Errorf("<%s>가 카탈로그에 있다 — 뒤 iteration의 어휘다", name)
		}
		if _, known := elementSpecs[name]; known {
			t.Errorf("<%s>가 문법 표에 있다 — 뒤 iteration의 어휘다", name)
		}
	}
}

// `type` 값 어휘 12종은 PRD §7.5의 표 순서 그대로다 — 목록은 soml에 있고 검사는 core다
// (색 토큰과 같은 구조). 순서가 흔들리면 모르는 type 진단이 나열하는 목록이 비결정적이 된다.
func TestShapeTypes(t *testing.T) {
	want := []string{
		"rect", "rounded-rect", "ellipse", "diamond", "cylinder", "chevron",
		"arrow-right", "hexagon", "parallelogram", "callout", "star", "cloud",
	}
	got := ShapeTypes()
	if len(got) != len(want) {
		t.Fatalf("ShapeTypes() = %v (%d개), want %d개", got, len(got), len(want))
	}
	for i, name := range want {
		if got[i] != name {
			t.Errorf("ShapeTypes()[%d] = %q, want %q", i, got[i], name)
		}
	}
	// 호출자가 카탈로그를 흔들 수 없어야 한다 — 표를 한 곳에 둔 뜻이 사라진다.
	ShapeTypes()[0] = "짓밟기"
	if ShapeTypes()[0] != "rect" {
		t.Error("ShapeTypes()가 내부 슬라이스를 그대로 준다")
	}
	ShapeAttrs()[0] = "짓밟기"
	if ShapeAttrs()[0] != "type" {
		t.Error("ShapeAttrs()가 내부 슬라이스를 그대로 준다")
	}
	ShapeAppearanceAttrs()[0] = "짓밟기"
	if ShapeAppearanceAttrs()[0] != "variant" {
		t.Error("ShapeAppearanceAttrs()가 내부 슬라이스를 그대로 준다")
	}
	BoxAttrs()[0] = "짓밟기"
	if BoxAttrs()[0] != "pad" {
		t.Error("BoxAttrs()가 내부 슬라이스를 그대로 준다")
	}
}

// 박스 속성 카탈로그를 다시 나눈 것(pad를 flow에서 떼고 containerBoxAttrs를 boxSizeAttrs로)의
// 회귀: **레이아웃 컨테이너의 허용 속성 목록과 순서가 그대로여야 한다** — 진단 메시지가 그
// 순서를 싣고 기존 테스트가 문구를 단정하고 있다. 그리고 도형과 컨테이너가 **같은 목록**을
// 소비하는지도 여기서 본다 (목록이 두 벌이 되면 PRD 표에 크기 속성이 하나 늘 때 어느 태그가
// 빠졌는지 사람이 세어야 한다).
func TestBoxAttrsSharedWithContainers(t *testing.T) {
	wantRow := []string{"gap", "align", "justify", "wrap", "pad", "w", "h", "min-w", "min-h", "max-w", "max-h", "ratio"}
	wantGrid := []string{"gap", "align", "justify", "wrap", "pad", "cols", "rows", "w", "h", "min-w", "min-h", "max-w", "max-h", "ratio"}
	for _, tc := range []struct {
		name string
		want []string
	}{
		{RowElement, wantRow},
		{ColElement, wantRow},
		{GridElement, wantGrid},
	} {
		if got := strings.Join(ContainerAttrs(tc.name), " "); got != strings.Join(tc.want, " ") {
			t.Errorf("ContainerAttrs(%q) = %v\nwant %v", tc.name, ContainerAttrs(tc.name), tc.want)
		}
	}
	// 박스 속성은 도형의 속성 표에도 **같은 순서로** 들어 있다.
	if got, want := strings.Join(BoxAttrs(), " "), "pad w h min-w min-h max-w max-h ratio"; got != want {
		t.Errorf("BoxAttrs() = %q, want %q", got, want)
	}
	shape := elementSpecs[ShapeElement].attrs
	want := strings.Join(append(append([]string{"type", "id"}, ShapeAppearanceAttrs()...),
		append(BoxAttrs(), ChildPlacementAttrs()...)...), " ")
	if got := strings.Join(shape, " "); got != want {
		t.Errorf("<shape> 속성 = %q\nwant %q", got, want)
	}
	// 컨테이너와 도형이 크기 속성을 하나도 빠뜨리지 않는다.
	for _, attr := range BoxAttrs() {
		for _, name := range []string{RowElement, ColElement, GridElement, ShapeElement} {
			if !hasAttr(elementSpecs[name].attrs, attr) {
				t.Errorf("<%s>에 박스 속성 %s가 없다", name, attr)
			}
		}
	}
	// 크기 속성은 텍스트 요소·블록에 열리지 않는다 (이번 범위 밖 — 블록의 폭은 엔진 CSS가
	// 100%로 정하고 있다).
	for _, name := range append([]string{TextElement}, blockElements...) {
		for _, attr := range BoxAttrs() {
			if hasAttr(elementSpecs[name].attrs, attr) {
				t.Errorf("<%s>에 박스 속성 %s가 열려 있다 — 이번 iteration의 범위가 아니다", name, attr)
			}
		}
	}
}

// `anchor`가 열린 자리와 닫힌 자리 전수 — 속성 표가 흔들리면 저자가 쓴 앵커가 조용히
// 무시되거나 뜻 없는 자리에서 받아들여진다.
func TestAnchorAttrCoverage(t *testing.T) {
	for _, name := range append([]string{TextElement}, blockElements...) {
		if !hasAttr(elementSpecs[name].attrs, AnchorAttr) {
			t.Errorf("<%s>에 %s가 없다", name, AnchorAttr)
		}
	}
	closed := []string{ShapeElement, ListItemElement, RootElement, noteElement, SpacerElement}
	closed = append(closed, inlineElements...)
	closed = append(closed, listContainerElements...)
	closed = append(closed, layoutContainerElements...)
	for _, name := range closed {
		if hasAttr(elementSpecs[name].attrs, AnchorAttr) {
			t.Errorf("<%s>에 %s가 열려 있다", name, AnchorAttr)
		}
	}
}

// 도형도 자식 배치 속성 한 벌을 그대로 받는다 — 레이아웃 컨테이너의 자식이 될 수 있는
// 요소의 목록이 넓어졌다 (TestChildPlacementAttrsCoverage의 연장).
func TestShapeChildPlacementAttrs(t *testing.T) {
	for _, attr := range ChildPlacementAttrs() {
		if !hasAttr(elementSpecs[ShapeElement].attrs, attr) {
			t.Errorf("<%s>에 자식 배치 속성 %s가 없다", ShapeElement, attr)
		}
	}
	// 도형이 놓일 수 있는 두 자리(슬라이드 직속·컨테이너 안)에 어휘가 열려 있는지.
	for _, children := range [][]string{slideChildren(), layoutChildren()} {
		found := false
		for _, name := range children {
			if name == ShapeElement {
				found = true
			}
		}
		if !found {
			t.Errorf("자식 어휘 %v에 <%s>가 없다", children, ShapeElement)
		}
	}
}

func hasAttr(attrs []string, name string) bool {
	for _, attr := range attrs {
		if attr == name {
			return true
		}
	}
	return false
}

// 픽스처 `04_shape`의 슬라이드가 파서 어휘와 어긋나지 않는지 파싱만 검증한다 — 골든·렌더
// 검증은 픽스처 러너(internal/testkit)의 몫이다 (항목 06이 그 러너에 이 픽스처를 넣었다).
// 여기가 파서 단으로 남아 있는 이유: 어휘 위반은 골든 diff가 아니라 진단 문구로 드러나야
// 한다. 모르는 variant 슬라이드도 파서에는 진단이 없다: variant는 이름만 어휘에 있고 값
// 검사와 경고는 값 층(항목 03)이 소유한다.
func TestParseShapeFixture(t *testing.T) {
	dir := filepath.Join("..", "testdata", "fixtures", "04_shape", "slides")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("픽스처 디렉토리를 읽을 수 없다: %v", err)
	}
	parsed, shapes := 0, 0
	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), Ext) {
			continue
		}
		src, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			t.Fatalf("%s: %v", entry.Name(), err)
		}
		doc, diags := Parse(entry.Name(), string(src))
		if len(diags) != 0 {
			t.Errorf("%s: 진단이 없어야 한다:\n%s", entry.Name(), diags.Err())
		}
		shapes += countShapes(doc.Content)
		parsed++
	}
	if parsed == 0 {
		t.Fatal("파싱한 슬라이드가 없다 — 픽스처가 비었다")
	}
	// 픽스처는 뒤 항목 전부의 시험대다 — 도형이 없으면 시험대가 아니다. 12종 전수 슬라이드
	// 하나만으로도 12개이므로 이 하한은 "픽스처가 도형을 잃지 않았다"를 뜻한다.
	if shapes < 12 {
		t.Errorf("픽스처의 도형이 %d개다 — type 12종 전수만으로도 12개여야 한다", shapes)
	}
}

// countShapes는 트리의 `<shape>` 개수다.
func countShapes(nodes []Node) int {
	count := 0
	for _, node := range nodes {
		elem, ok := node.(*Element)
		if !ok {
			continue
		}
		if elem.Name == ShapeElement {
			count++
		}
		count += countShapes(elem.Children)
	}
	return count
}
