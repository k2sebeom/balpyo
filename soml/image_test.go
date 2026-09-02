package soml

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// 이미지의 정상 트리 — 태그·속성 넷(고유·외관·박스·자식 배치)·자기 닫힘·두 자리(슬라이드
// 직속·컨테이너 안)가 파싱을 그대로 통과한다. dumpNodes의 표현은 subset_test.go와 공유한다.
func TestParseImage(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want string
	}{
		{
			name: "고유 속성 — src와 fit",
			src:  `<slide><image src="assets/wide.png" fit="cover"/></slide>`,
			want: `image[src=assets/wide.png fit=cover]`,
		},
		{
			// alt은 PRD 표에 없지만 양 타깃에 실을 자리가 있어 연다 (imageOwnAttrs).
			name: "고유 속성 — alt",
			src:  `<slide><image src="assets/wide.png" alt="아키텍처 개요"/></slide>`,
			want: `image[src=assets/wide.png alt=아키텍처 개요]`,
		},
		{
			name: "외관 속성 7종",
			src: `<slide><image src="assets/wide.png" variant="framed" border="1 solid muted" ` +
				`radius="lg" shadow="soft" opacity="0.8" rotate="45" flip="h"/></slide>`,
			want: `image[src=assets/wide.png variant=framed border=1 solid muted ` +
				`radius=lg shadow=soft opacity=0.8 rotate=45 flip=h]`,
		},
		{
			// `fit`은 박스가 정해진 다음에 뜻을 갖는다 (ImageAttrs의 주석) — 박스 속성은
			// 도형·레이아웃 컨테이너와 같은 목록이다 (BoxAttrs).
			name: "박스 속성 — pad와 크기·비율",
			src:  `<slide><image src="assets/wide.png" pad="md" w="fill" h="220" min-w="120" min-h="80" max-w="480" max-h="320" ratio="16:9"/></slide>`,
			want: `image[src=assets/wide.png pad=md w=fill h=220 min-w=120 min-h=80 max-w=480 max-h=320 ratio=16:9]`,
		},
		{
			name: "자식 배치 속성 — 이미지는 컨테이너의 자식이 될 수 있다",
			src:  `<slide><grid cols="3"><image src="assets/wide.png" span="2" grow="1" self="center" order="2"/></grid></slide>`,
			want: `grid[cols=3](image[src=assets/wide.png span=2 grow=1 self=center order=2])`,
		},
		{
			name: "이미지도 style 탈출구를 받는다",
			src:  `<slide><row><image src="assets/wide.png" style="order: 2"/></row></slide>`,
			want: `row(image[src=assets/wide.png style=order: 2])`,
		},
		{
			name: "슬라이드 직속 이미지 — 전면 그림",
			src:  `<slide><h2>표제</h2><image src="assets/wide.png" fit="cover" w="fill" h="480"/></slide>`,
			want: `h2("표제") image[src=assets/wide.png fit=cover w=fill h=480]`,
		},
		{
			// 캡션은 자식이 아니라 형제 <text>다 (IsVoidElement의 주석).
			name: "그림+캡션 — 형제 text",
			src:  `<slide><col gap="xs"><image src="assets/wide.png" w="fill"/><text role="caption">그림 1. 구성도</text></col></slide>`,
			want: `col[gap=xs](image[src=assets/wide.png w=fill] text[role=caption]("그림 1. 구성도"))`,
		},
		{
			name: "그림+텍스트 2단",
			src:  `<slide><row gap="lg"><image src="assets/tall.png" fit="contain" basis="480"/><col><h3>제목</h3><p>본문</p></col></row></slide>`,
			want: `row[gap=lg](image[src=assets/tall.png fit=contain basis=480] col(h3("제목") p("본문")))`,
		},
		{
			// 파서는 값을 보지 않는다 — 없는 파일·어휘 밖의 fit도 문법은 통과하고 진단은
			// 값 층(항목 02)의 몫이다. 이 비대칭이 의도다 (Element.Attrs의 주석).
			name: "없는 파일도 파서는 통과시킨다",
			src:  `<slide><image src="assets/없는파일.png"/></slide>`,
			want: `image[src=assets/없는파일.png]`,
		},
		{
			name: "어휘 밖의 fit도 파서는 통과시킨다",
			src:  `<slide><image src="assets/wide.png" fit="fill"/></slide>`,
			want: `image[src=assets/wide.png fit=fill]`,
		},
		{
			// src 없는 이미지도 문법은 통과한다 — "속성이 있어야 한다"는 값의 요구이고,
			// href 없는 <a>·type 없는 <shape>를 파서가 통과시키는 것과 같은 근거다.
			name: "src 없는 이미지도 파서는 통과시킨다",
			src:  `<slide><image/></slide>`,
			want: `image`,
		},
		{
			// 값 없는 속성의 처리는 기존 규칙 그대로다 (Attr.HasValue) — 모델에는 이름만
			// 남고, 그것이 불리언 속성(`<slide skip>`)과 같은 모양이다.
			name: "값 없는 속성 — 이름만 남는다",
			src:  `<slide><image src/></slide>`,
			want: `image[src=]`,
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

// 값 없는 속성은 HasValue로 구별된다 — `<image src>`와 `<image src="">`가 모델에서 다른
// 것이어야 값 층이 "경로가 없다"와 "경로가 비었다"를 다른 진단으로 낼 수 있다.
func TestParseImageAttrHasValue(t *testing.T) {
	doc := parseOK(t, `<slide><image src/><image src=""/></slide>`)
	if attr, ok := elementAt(t, doc.Content, 0).Attr("src"); !ok || attr.HasValue {
		t.Errorf("<image src>의 HasValue = %v, want false", attr.HasValue)
	}
	if attr, ok := elementAt(t, doc.Content, 1).Attr("src"); !ok || !attr.HasValue || attr.Value != "" {
		t.Errorf(`<image src="">의 HasValue = %v Value = %q, want true ""`, attr.HasValue, attr.Value)
	}
}

// 요소의 Span은 여는 `<`부터 `/>`까지다 — 잰 rect가 곧 그림 좌표이므로 진단이 저자의
// 태그를 정확히 가리켜야 한다.
func TestParseImageSpans(t *testing.T) {
	doc := parseOK(t, "<slide>\n  <image src=\"assets/wide.png\"/>\n</slide>")
	image := elementAt(t, doc.Content, 0)
	if got, want := image.Span.Start.String()+"-"+image.Span.End.String(), "2:3-2:33"; got != want {
		t.Errorf("<image> Span = %s, want %s", got, want)
	}
	// 값 스팬은 인용 부호 안이다 — 항목 02의 `src` 진단이 경로 문자열을 가리킨다.
	attr, ok := image.Attr("src")
	if !ok {
		t.Fatal("src 속성이 없다")
	}
	if got, want := attr.ValueSpan.Start.String(), "2:15"; got != want {
		t.Errorf("src 값 스팬 시작 = %s, want %s", got, want)
	}
}

// 에러 케이스 전수 — 진단의 line:col과 문구까지 단정한다.
func TestParseImageErrors(t *testing.T) {
	tests := []struct {
		name string
		src  string
		// want의 각 원소는 "line:col 메시지조각"이다.
		want []string
	}{
		{
			// void 위반의 진단은 **여는 태그 하나**다 — 안쪽 내용까지 진단해 하나의 오용을
			// 여러 줄로 부풀리지 않는다 (parseDocument·charData의 modelVoid 갈래).
			name: "이미지 안의 문자 데이터",
			src:  `<slide><image src="assets/wide.png">캡션</image></slide>`,
			want: []string{`1:9 void 태그 <image>은 내용을 가질 수 없다 (<image/>로 쓴다)`},
		},
		{
			// 캡션은 형제 <text>다 — 그림 안 글자는 도형의 일이고 그림 위 글자는 겹침의 일이다.
			name: "이미지 안의 요소",
			src:  `<slide><image src="assets/wide.png"><text>캡션</text></image></slide>`,
			want: []string{"1:9 void 태그 <image>은 내용을 가질 수 없다"},
		},
		{
			name: "이미지 안의 note",
			src:  `<slide><image src="assets/wide.png"><note>메모</note></image></slide>`,
			want: []string{"1:9 void 태그 <image>은 내용을 가질 수 없다"},
		},
		{
			// 도형은 txBody 하나를 갖는다는 Iteration 4의 매핑과 정면으로 부딪힌다
			// (shapeChildren) — 도형 안 그림은 <stack>의 자리다.
			name: "도형 안의 이미지",
			src:  `<slide><shape type="rect"><image src="assets/wide.png"/></shape></slide>`,
			want: []string{"1:28 <shape>의 자식으로 <image>는 올 수 없다 (쓸 수 있는 태그: <text>, <h1>, <h2>, <h3>, <p>, <blockquote>, <pre>)"},
		},
		{
			name: "블록 안의 이미지",
			src:  `<slide><p><image src="assets/wide.png"/></p></slide>`,
			want: []string{"1:12 <p> 안에는 문자 데이터와 인라인 태그만 올 수 있다 (<image>는 쓸 수 없다)"},
		},
		{
			name: "li 안의 이미지",
			src:  `<slide><ul><li><image src="assets/wide.png"/></li></ul></slide>`,
			want: []string{"1:17 <li> 안에는 문자 데이터와 인라인 태그만 올 수 있다 (<image>는 쓸 수 없다)"},
		},
		{
			name: "리스트 컨테이너 안의 이미지",
			src:  `<slide><ul><image src="assets/wide.png"/></ul></slide>`,
			want: []string{"1:13 <ul>의 자식으로 <image>는 올 수 없다 (쓸 수 있는 태그: <li>)"},
		},
		{
			// `fill`은 PRD §7.6 표의 적용 대상에 image가 없다 — 그림 뒤의 칠이 보이는 자리는
			// contain의 여백뿐이고 그 여백은 PPTX에서 pic의 박스 밖이다 (imageAppearanceAttrs).
			// 진단이 이미지의 속성 표 전체를 순서대로 싣는다.
			name: "이미지에 fill은 없다",
			src:  `<slide><image src="assets/wide.png" fill="surface"/></slide>`,
			want: []string{`1:37 <image>에 알 수 없는 속성 fill (쓸 수 있는 속성: src, fit, alt, variant, border, radius, shadow, opacity, rotate, flip, pad, w, h, min-w, min-h, max-w, max-h, ratio, grow, shrink, basis, span, self, order, at, x, y, style, slot)`},
		},
		{
			// PRD §7.5 표의 frame은 뜻이 어디에도 정의되어 있지 않다 — 액자는 variant가
			// 소유하는 채널이다 (imageOwnAttrs의 주석).
			name: "frame은 열지 않았다",
			src:  `<slide><image src="assets/wide.png" frame="shadow"/></slide>`,
			want: []string{"1:37 <image>에 알 수 없는 속성 frame"},
		},
		{
			// 도형의 프리셋은 그림에 없다.
			name: "이미지에 type은 없다",
			src:  `<slide><image src="assets/wide.png" type="rect"/></slide>`,
			want: []string{"1:37 <image>에 알 수 없는 속성 type"},
		},
		{
			// 그림은 텍스트가 아니다 — 앵커·역할은 텍스트 요소가 소유한다.
			name: "이미지에 anchor는 없다",
			src:  `<slide><image src="assets/wide.png" anchor="center"/></slide>`,
			want: []string{"1:37 <image>에 알 수 없는 속성 anchor"},
		},
		{
			name: "이미지에 cols는 없다",
			src:  `<slide><image src="assets/wide.png" cols="2"/></slide>`,
			want: []string{"1:37 <image>에 알 수 없는 속성 cols"},
		},
		{
			name: "모르는 이미지 속성",
			src:  `<slide><image scr="assets/wide.png"/></slide>`,
			want: []string{"1:15 <image>에 알 수 없는 속성 scr"},
		},
		{
			// `src`·`fit`은 이미지의 것이다 — 열지 않은 자리에서 받으면 저자가 쓴 것이
			// 조용히 무시된다.
			name: "도형에는 src가 없다",
			src:  `<slide><shape type="rect" src="assets/wide.png"/></slide>`,
			want: []string{"1:27 <shape>에 알 수 없는 속성 src"},
		},
		{
			name: "도형에는 fit이 없다",
			src:  `<slide><shape type="rect" fit="cover"/></slide>`,
			want: []string{"1:27 <shape>에 알 수 없는 속성 fit"},
		},
		{
			name: "컨테이너에는 src가 없다",
			src:  `<slide><row src="assets/wide.png"></row></slide>`,
			want: []string{"1:13 <row>에 알 수 없는 속성 src"},
		},
		{
			name: "text에는 alt이 없다",
			src:  `<slide><text alt="설명">표제</text></slide>`,
			want: []string{"1:14 <text>에 알 수 없는 속성 alt"},
		},
		{
			// 닫는 태그를 쓴 void 요소는 여는 태그에서 한 번만 진단된다 — </image>가
			// "여는 태그가 없다"는 두 번째 진단을 만들지 않게 스택에는 올린다 (parseDocument).
			name: "짝 태그로 쓴 이미지",
			src:  `<slide><image src="assets/wide.png"></image></slide>`,
			want: []string{"1:9 void 태그 <image>은 내용을 가질 수 없다"},
		},
		{
			// `>` 없이 끝난 여는 태그는 세 진단이 같은 자리에 겹친다: 태그가 닫히지 않았다 →
			// void 위반(내용을 가질 수 없다) → 안쪽에 열린 채로 남았다. 셋 다 저자의
			// `<image>` 이름을 가리키는 것이 계약이다 (진단이 들여쓰기가 아니라 태그를 가리킨다).
			name: "닫히지 않은 이미지 태그",
			src:  "<slide>\n  <image src=\"assets/wide.png\"\n</slide>",
			want: []string{
				"2:4 <image> 태그가 `>`로 닫히지 않았다",
				"2:4 void 태그 <image>은 내용을 가질 수 없다",
				"2:4 <image> 태그가 </slide>보다 먼저 닫혀야 한다",
			},
		},
		{
			name: "속성이 중복됐다",
			src:  `<slide><image src="a.png" src="b.png"/></slide>`,
			want: []string{"1:27 속성 src가 중복됐다 (앞의 것만 쓴다)"},
		},
		{
			// 자기 닫힘 이미지는 정상이다 — 이 케이스는 진단 0건을 단정하는 자리다.
			name: "자기 닫힘 이미지는 진단이 없다",
			src:  `<slide><image src="assets/wide.png"/></slide>`,
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
// 좌표가 엉뚱한 요소에 붙는다. `<image>`가 **다른 종류**인 것이 계약이다 (KindImage).
func TestImageKind(t *testing.T) {
	if got := KindOf(ImageElement); got != KindImage {
		t.Errorf("KindOf(%q) = %v, want KindImage", ImageElement, got)
	}
	if got := KindOf(ShapeElement); got == KindOf(ImageElement) {
		t.Errorf("KindOf(shape) == KindOf(image) = %v — 네 층의 산출이 다르다", got)
	}
	// 종류만 있고 문법이 없는 이름은 "알 수 없는 태그"로 진단되면서 렌더러는 종류로
	// 분기하는 어긋난 상태가 된다.
	if _, known := elementSpecs[ImageElement]; !known {
		t.Errorf("<%s>가 문법 표(elementSpecs)에 없다", ImageElement)
	}
	// 이미지는 void다 — 담을 것이 없다 (캡션은 형제 <text>).
	if !IsVoidElement(ImageElement) {
		t.Errorf("IsVoidElement(%q) = false — 이미지는 담을 것이 없다", ImageElement)
	}
	if got := elementSpecs[ImageElement].model; got != modelVoid {
		t.Errorf("<%s>의 콘텐츠 모델 = %v, want modelVoid", ImageElement, got)
	}
	if children := elementSpecs[ImageElement].children; len(children) != 0 {
		t.Errorf("<%s>에 자식 어휘가 있다: %v", ImageElement, children)
	}
	// 이미지는 텍스트 역할을 갖지 않는다.
	if role, ok := TagTextRole(ImageElement); ok {
		t.Errorf("TagTextRole(%q) = %q — 이미지는 역할 표에 없어야 한다", ImageElement, role)
	}
	// Iteration 7·9와 P1·P2의 컴포넌트는 카탈로그에도 어휘에도 없어야 한다. `<stack>`은
	// Iteration 6에 열렸다 (layout.go의 TestLayoutKinds가 그 계약을 본다).
	for _, name := range []string{"arrow", "line", "group", "icon", "table", "chart"} {
		if KindOf(name) != KindNone {
			t.Errorf("<%s>가 카탈로그에 있다 — 뒤 iteration의 어휘다", name)
		}
		if _, known := elementSpecs[name]; known {
			t.Errorf("<%s>가 문법 표에 있다 — 뒤 iteration의 어휘다", name)
		}
	}
}

// `fit` 값 어휘 2종은 PRD §7.5의 순서 그대로다 — 목록은 soml에 있고 검사는 core다
// (색 토큰과 같은 구조). 순서가 흔들리면 모르는 fit 진단이 나열하는 목록이 비결정적이 된다.
func TestImageFits(t *testing.T) {
	want := []ImageFit{FitCover, FitContain}
	got := ImageFits()
	if len(got) != len(want) {
		t.Fatalf("ImageFits() = %v (%d개), want %d개", got, len(got), len(want))
	}
	for i, fit := range want {
		if got[i] != fit {
			t.Errorf("ImageFits()[%d] = %q, want %q", i, got[i], fit)
		}
	}
	if string(FitCover) != "cover" || string(FitContain) != "contain" {
		t.Errorf("fit 어휘의 문자열이 PRD와 다르다: %q %q", FitCover, FitContain)
	}
	// 호출자가 카탈로그를 흔들 수 없어야 한다 — 표를 한 곳에 둔 뜻이 사라진다.
	ImageFits()[0] = "짓밟기"
	if ImageFits()[0] != FitCover {
		t.Error("ImageFits()가 내부 슬라이스를 그대로 준다")
	}
	ImageAttrs()[0] = "짓밟기"
	if ImageAttrs()[0] != "src" {
		t.Error("ImageAttrs()가 내부 슬라이스를 그대로 준다")
	}
	ImageAppearanceAttrs()[0] = "짓밟기"
	if ImageAppearanceAttrs()[0] != "variant" {
		t.Error("ImageAppearanceAttrs()가 내부 슬라이스를 그대로 준다")
	}
}

// 이미지의 속성 표가 넷(고유·외관·박스·자식 배치)을 그 순서로 잇는지, 그리고 박스 속성
// 한 벌을 도형·컨테이너와 **공유**하는지 — 목록이 두 벌이 되면 PRD 표에 크기 속성이 하나
// 늘 때 어느 태그가 빠졌는지 사람이 세어야 한다.
func TestImageAttrs(t *testing.T) {
	want := strings.Join(append(append([]string{"src", "fit", "alt"}, ImageAppearanceAttrs()...),
		append(append(BoxAttrs(), ChildPlacementAttrs()...), SlotAttr)...), " ")
	if got := strings.Join(elementSpecs[ImageElement].attrs, " "); got != want {
		t.Errorf("<image> 속성 = %q\nwant %q", got, want)
	}
	for _, attr := range append(BoxAttrs(), ChildPlacementAttrs()...) {
		if !hasAttr(elementSpecs[ImageElement].attrs, attr) {
			t.Errorf("<%s>에 %s가 없다", ImageElement, attr)
		}
	}
	// 이미지에 열지 않은 것 전수 — 받아 놓고 한 타깃에서만 그리는 속성을 만들지 않는다.
	for _, attr := range []string{"fill", "frame", "type", "id", "role", "size", "color",
		"align", AnchorAttr, "stroke", "cap", "dash", "heads", "gap", "cols", "rows",
		"justify", "wrap", "href"} {
		if hasAttr(elementSpecs[ImageElement].attrs, attr) {
			t.Errorf("<%s>에 %s가 열려 있다", ImageElement, attr)
		}
	}
	// 이미지 고유 속성은 다른 태그에 새지 않았다.
	for name, spec := range elementSpecs {
		if name == ImageElement {
			continue
		}
		for _, attr := range []string{"src", "fit", "alt"} {
			if hasAttr(spec.attrs, attr) {
				t.Errorf("<%s>에 이미지 고유 속성 %s가 열려 있다", name, attr)
			}
		}
	}
	// 도형의 속성 표는 한 비트도 움직이지 않았다 — 진단 메시지가 그 순서를 싣는다.
	if got, want := strings.Join(ShapeAttrs(), " "),
		"type id variant fill border radius shadow opacity rotate flip pad w h min-w min-h max-w max-h ratio"; got != want {
		t.Errorf("ShapeAttrs() = %q\nwant %q", got, want)
	}
}

// 이미지가 놓일 수 있는 두 자리(슬라이드 직속·컨테이너 안)에 어휘가 열려 있고, 열지 않은
// 자리(도형 안)에는 없는지 — 자식 어휘의 **순서**는 진단 메시지에 실린다.
func TestImageChildVocabulary(t *testing.T) {
	for _, tc := range []struct {
		name     string
		children []string
	}{
		{"slide", slideChildren()},
		{"layout", layoutChildren()},
	} {
		if !hasAttr(tc.children, ImageElement) {
			t.Errorf("%s 자식 어휘 %v에 <%s>가 없다", tc.name, tc.children, ImageElement)
		}
		// 컴포넌트가 앞에 온다 — <text> <shape> <image> 순이다.
		if got := strings.Join(tc.children[:3], " "); got != "text shape image" {
			t.Errorf("%s 자식 어휘의 앞 세 개 = %q, want %q", tc.name, got, "text shape image")
		}
	}
	if hasAttr(shapeChildren(), ImageElement) {
		t.Errorf("도형의 자식 어휘에 <%s>가 있다 — 도형은 txBody 하나를 갖는다", ImageElement)
	}
}

// 픽스처 `05_image`의 슬라이드가 파서 어휘와 어긋나지 않는지 파싱만 검증한다 — 골든·렌더
// 검증은 픽스처 러너(internal/testkit)의 몫이다 (항목 05가 스위트에 편입했다).
// 여기가 파서 단으로 남아 있는 이유: 어휘 위반은 골든 diff가 아니라 진단 문구로 드러나야
// 한다. 없는 `src`·모르는 variant는 파서에 진단이 없다 — 값 검사는 값 층(항목 02)이 소유한다.
func TestParseImageFixture(t *testing.T) {
	root := filepath.Join("..", "testdata", "fixtures", "05_image")
	dir := filepath.Join(root, "slides")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("픽스처 디렉토리를 읽을 수 없다: %v", err)
	}
	parsed, images := 0, 0
	srcs := make(map[string]int)
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
		images += collectImages(doc.Content, srcs)
		parsed++
	}
	if parsed == 0 {
		t.Fatal("파싱한 슬라이드가 없다 — 픽스처가 비었다")
	}
	// 픽스처는 뒤 항목 전부의 시험대다 — fit 2종 × 넓은/좁은 그림의 4칸만으로도 4개다.
	if images < 4 {
		t.Errorf("픽스처의 이미지가 %d개다 — fit 4칸 표만으로도 4개여야 한다", images)
	}
	// 픽스처가 가리키는 에셋이 실제로 있어야 한다 (항목 02의 시험대이자 항목 04의 미디어
	// 파트 입력이다). 파서는 파일을 보지 않으므로 이 단정이 여기 있다.
	for src := range srcs {
		if _, err := os.Stat(filepath.Join(root, src)); err != nil {
			t.Errorf("픽스처 에셋이 없다: %s (%v)", src, err)
		}
	}
	// 미디어 파트 재사용(중복 제거) 계약의 유일한 회귀 자리 — 같은 src를 두 번 쓰는
	// 슬라이드가 픽스처에 있어야 한다 (항목 04).
	reused := false
	for _, count := range srcs {
		if count > 1 {
			reused = true
		}
	}
	if !reused {
		t.Error("같은 src를 두 번 쓰는 자리가 픽스처에 없다 — 미디어 파트 재사용의 회귀 자리다")
	}
}

// collectImages는 트리의 `<image>` 개수를 세고 `src` 값의 사용 횟수를 모은다.
func collectImages(nodes []Node, srcs map[string]int) int {
	count := 0
	for _, node := range nodes {
		elem, ok := node.(*Element)
		if !ok {
			continue
		}
		if elem.Name == ImageElement {
			count++
			if attr, ok := elem.Attr("src"); ok && attr.Value != "" {
				srcs[attr.Value]++
			}
		}
		count += collectImages(elem.Children, srcs)
	}
	return count
}
