package soml

import (
	"sort"
	"strings"
)

// RootElement는 `slides/` 아래 파일의 루트 요소 이름이다. 파일의 역할은 확장자가 아니라
// 루트 요소가 정한다 — `components/`의 `<define>`은 아직 열리지 않았다.
const RootElement = "slide"

// noteElement는 발표자 노트 태그다. 콘텐츠가 아니라 슬라이드 메타데이터이므로 콘텐츠
// 트리에서 빠지고 SlideDoc.Notes로 간다.
const noteElement = "note"

// elementSpecs는 **닫힌 어휘**다. 여기 없는 태그 이름은 무시되지 않고 에러가 된다.
//
// 어휘는 렌더 매핑과 함께 열린다 — 텍스트 서브셋은 subset.go, 레이아웃 태그는 layout.go,
// `<shape>`는 shape.go, `<image>`는 image.go의 카탈로그가 소유한다.
// 어휘를 미리 열어 두면 "파싱은 되는데 렌더는 안 되는" 상태가 생기고, 그것이 닫힌 어휘로
// 막으려던 바로 그 조용한 실패다. 렌더 매핑이 아직 없는 태그는 렌더러가 명시적으로 멈춘다
// (render/web.writeNode, render/pptx.writeShapeNode).
var elementSpecs = buildElementSpecs()

// buildElementSpecs는 문법 표를 만든다. 서브셋·레이아웃 태그의 규칙은 카탈로그
// (subset.go·layout.go)의 종류에서 파생된다 — 태그를 카탈로그에 추가하면 문법이 따라오고,
// 두 표가 어긋날 수 없다.
func buildElementSpecs() map[string]elementSpec {
	specs := map[string]elementSpec{
		RootElement: {
			// 순서는 속성 표 순서다 — `layout` `id` `skip`
			// (진단이 이 순서를 싣는다).
			// `layout`은 슬라이드 수준 속성 중 **렌더러의 배치를 바꾸는 첫 속성**이다
			// (id는 딥링크, skip은 덱에서 빠진다). 값 어휘(content·canvas)는 core가 본다 —
			// fillSlide가 문법만 보는 이유와 같다.
			// `justify`는 **흐름 모드 슬라이드의 세로 분배**다 — 저자에게
			// 세로 리듬 채널이 없어서 `<col h="fill" justify="between">` 랩퍼가 그 자리를
			// 대신하고 있었다. 값 어휘는 컨테이너와 같은 6종이고 검사는 core다.
			// `enter`는 직속 자식마다 같은 등장을 준다 (자식의 `enter`가 이긴다 — core).
			// `invert`는 그 장의 색 매핑을 뒤집는다 (배경 · 글자 네 자리 — 어두운 장).
			attrs: []string{"layout", "id", "skip", "justify", EnterAttr, TransitionAttr, InvertAttr},
			model: modelElements,
			// `<slide>`에는 `reveal` 속성을 열지 않는다 — 웹 런타임의 reveal은 Phase 2이고,
			// 렌더가 없는 속성을 어휘에 열면 닫힌 어휘가 막으려던 조용한 무시가 된다.
			// 자식 배치 속성도 없다: `<slide>`는 컨테이너의 자식이 될 수 없고(루트다),
			// 슬라이드 자신의 배치는 레이아웃과 테마가 소유한다.
			children: slideChildren(),
		},
		TextElement: {
			// 외관 → 박스(가로 크기) → 자식 배치 → 슬롯 순이다 (withTextSize +
			// withChildPlacement). `<text>`는 레이아웃 컨테이너의 자식이 될 수 있으므로
			// `<row><text grow="1">`이 문법상 유효해야 하고, 가로 크기도 받는다 —
			// 특정 폭 안에 라벨을 가운데 두려면 그 폭을 가진 박스가 필요하다.
			attrs: withChildPlacement(withTextSize([]string{"role", "variant", "size", "color", "align", AnchorAttr})),
			model: modelInline,
		},
		ShapeElement: {
			// 고유 → 외관 → 박스(pad·크기) → 자식 배치 순이다 (ShapeAttrs). 도형은 레이아웃
			// 컨테이너의 자식이 될 수 있으므로 `<grid><shape span="2">`이 유효하다.
			attrs:    withChildPlacement(ShapeAttrs()),
			model:    modelElements,
			children: shapeChildren(),
		},
		ImageElement: {
			// 고유 → 외관 → 박스(pad·크기) → 자식 배치 순이다 (ImageAttrs). 이미지도
			// 레이아웃 컨테이너의 자식이 될 수 있으므로 `<grid><image span="2">`이 유효하고,
			// 그림+텍스트 2단·그림 그리드가 흔한 장면이다.
			//
			// void다 — `<image src="…" />` 하나가 전부이고 담을 것이 없다 (IsVoidElement의
			// 주석에 근거가 있다). 자식을 넣은 `<image>`는 파서가 멈춘다.
			attrs: withChildPlacement(ImageAttrs()),
			model: modelVoid,
		},
		TableElement: {
			// 고유(variant) → 자식 배치 → 슬롯 순이다 (TableAttrs + withChildPlacement).
			// 표는 슬라이드 직속이자 레이아웃 컨테이너의 자식이 될 수 있고(`<col><table>`),
			// 슬롯도 채울 수 있다. 자식은 행뿐이다 (`<ul>`의 자식이 `<li>`뿐인 것과 같다).
			attrs:    withChildPlacement(TableAttrs()),
			model:    modelElements,
			children: tableChildren(),
		},
		TableRowElement: {
			// 행은 셀만 담는다 (modelElements, children=[th,td]). 속성을 갖지 않는다 —
			// 행 단위 외관은 v1에 없고, 줄무늬는 셀 `fill`을 우리가 명시로 칠한다.
			// 자식 배치·슬롯도 없다: `<tr>`은 <table>의 자식으로만 오지 flex 항목이 아니다.
			model:    modelElements,
			children: tableRowChildren(),
		},
		ChartElement: {
			// 고유(type·unit·max·alt) → 박스 → 자식 배치 → 슬롯 순이다 (ChartAttrs +
			// withChildPlacement). 차트는 슬라이드 직속이자 레이아웃 컨테이너의 자식이 될 수
			// 있고(`<row><chart grow="1">`), 슬롯도 채운다 — 펼침의 뿌리가 도형이므로 배치
			// 어휘가 `<shape>`와 같은 한 벌이다.
			//
			// 자식은 점뿐이다 (`<table>`의 자식이 행뿐인 것과 같다). 문법은 여기서 끝이고
			// 값(종류 이름·수치·축)은 core가 본다 — `<chart>`의 특이한 점은 그 값 층이
			// 검사만 하지 않고 **트리를 도형·텍스트로 바꿔 놓는다**는 것이다 (core/chart.go).
			attrs:    withChildPlacement(ChartAttrs()),
			model:    modelElements,
			children: chartChildren(),
		},
		ChartPointElement: {
			// 점은 void다 — 라벨·값·강조색이 전부 속성이고 담을 것이 없다 (`<image>`와 같은
			// 근거). 자식 배치·슬롯도 없다: 점은 flex 항목이 아니라 데이터 한 칸이고, 막대의
			// 폭과 자리는 차트가 정한다 (`<tr>`·셀과 같은 분담).
			attrs: ChartPointAttrs(),
			model: modelVoid,
		},
		noteElement: {model: modelText},
		ConnectorElement: {
			// 연결선은 void이고 박스·배치 속성이 없다 — 자리는 잇는 두 상자가 정한다. 슬라이드
			// 직속뿐이다 (slideChildren): 렌더러가 슬라이드마다 오버레이 하나에 문서 순서로 그린다.
			attrs: ConnectorAttrs(),
			model: modelVoid,
		},
	}
	for _, name := range tableCellElements {
		// 셀은 텍스트+인라인을 담는다 (modelInline — `<li>`와 같은 콘텐츠 모델. 셀은 텍스트
		// 전용이다: OOXML `a:tc`가 하나의 `a:txBody`에 1:1로 맞는다). 속성은 병합
		// (colspan/rowspan)과 외관(fill/border/color/align/anchor)이다 —
		// 자식 배치·슬롯은 없다 (`<li>`가 childPlacement를 받지 않는 것과 같은 근거).
		specs[name] = elementSpec{attrs: TableCellAttrs(), model: modelInline}
	}
	for _, name := range blockElements {
		model := modelInline
		var own []string
		if name == "pre" {
			// v1의 pre는 문자 데이터만 담는다 — <pre><code> 관용은 열지 않는다: 런은 저자가
			// 아니라 토크나이저가 만든다. 하이라이팅이 인라인 서식과
			// 만나는 문제를 여는 대신 **지우는** 쪽으로 풀었다.
			model = modelPre
			// lang은 `<pre>` 고유 속성이다 — 다른 블록에는 뜻이 없다. 어휘의
			// 순서가 진단의 순서이므로 고유 속성이 외관·박스보다 앞이다. 값 어휘(닫힌
			// 네 이름 + 별칭 하나)와 진단은 core가 소유한다 — 언어 이름은 값이다.
			own = []string{LangAttr}
		}
		// 블록의 속성은 `<text>`의 외관 어휘에서 role만 뺀 것이다. 블록은 역할의 축약이라
		// (`<p>` ≡ `<text role="body">`, TagTextRole) 역할은 등가 표에서만 오고, role
		// 속성을 열면 `<p role="title">`이 두 표기의 등가를 깬다. 나머지 외관 속성
		// (variant/size/color/align/anchor)은 `<text>`와 같은 캐스케이드 규칙으로 얹힌다 —
		// 값 어휘와 진단은 core가 소유한다 (값 층). `anchor`를 `<text>`에만 열지 않는
		// 이유는 등가 표의 대칭이다: 블록이 받는 외관은 `<text>`가 받는 것과 같아야
		// `<p>` ≡ `<text role="body">`가 속성 표에서도 성립한다. variant를 열지 **않는**
		// 곳의 근거: 인라인 태그의 외관은 런 규칙이 소유하고, 리스트
		// 컨테이너와 <li>의 위계는 역할이 소유한다 — 항목마다 다른 외관 번들이 붙으면
		// 리스트의 타이포그래피가 리스트이기를 멈춘다.
		// 가로 크기 속성도 같은 대칭으로 받는다 (withTextSize).
		specs[name] = elementSpec{
			attrs: withChildPlacement(withTextSize(append(own,
				"variant", "size", "color", "align", AnchorAttr))),
			model: model,
		}
	}
	for _, name := range listContainerElements {
		// 리스트 컨테이너는 블록과 **같은 외관 4종**을 받는다 (variant만 빠진다):
		// 목록 글자 크기를 바꾸려고 레이아웃을 바꿀 이유가 없어야 한다. 속성이 컨테이너에
		// 붙고 `<li>`는 상속만 받는 이유는 외관이 **목록 단위의 성질**이라는 것이다 —
		// 항목마다 크기가 갈리면 그것은 목록이 아니라 문단 여럿이다.
		//
		// variant를 열지 않는 이유는 위 블록 주석과 같다: variant는 외관 **번들**이라
		// 상속의 뜻이 "글자 하나"가 아니라 "역할 바꾸기"가 된다.
		//
		// 배치의 대상도 된다 — `<row><ul basis="360">`은 두 열에 리스트를 놓는 흔한 배치다.
		// 가로 크기도 컨테이너의 것이다 (withTextSize): `<ul w="480">`은 목록
		// 전체의 폭이고, `<li>`는 여기서도 상속만 받는다.
		specs[name] = elementSpec{
			attrs:    withChildPlacement(withTextSize([]string{"size", "color", "align", AnchorAttr})),
			model:    modelElements,
			children: []string{ListItemElement},
		}
	}
	// `<li>`가 갖는 속성은 `reveal` **하나**다 — 외관은 컨테이너에서 상속하고,
	// 자식 배치 속성도 없다 (리스트 항목은 flex 항목이 아니라 한 텍스트 흐름의 문단이다).
	// `reveal`이 예외인 이유는 그것이 외관이 아니라 **순서**라는 것이다: 항목이
	// 언제 나타나는지는 항목마다 갈리는 성질이라 컨테이너로 올릴 수 없다.
	specs[ListItemElement] = elementSpec{attrs: []string{RevealAttr}, model: modelInline}
	for _, name := range layoutContainerElements {
		// 빈 컨테이너(`<row></row>`)는 유효하다 — 자식 0개의 flex 컨테이너는 CSS에서
		// 유효하고, 레이아웃을 짜는 중간 상태에서 저자가 늘 만든다. 빈 컨테이너가
		// 산출물에 무엇을 남기는지는 렌더러의 결정이다.
		specs[name] = elementSpec{
			attrs:    withChildPlacement(ContainerAttrs(name)),
			model:    modelElements,
			children: layoutChildren(),
		}
	}
	// <spacer>는 void다 — 밀어내기 자체가 내용이고, 담을 것이 있으면 그것은 컨테이너다.
	specs[SpacerElement] = elementSpec{attrs: withChildPlacement(nil), model: modelVoid}
	for _, name := range inlineElements {
		spec := elementSpec{model: modelInline}
		switch name {
		case "a":
			// 속성은 href 하나다. href 없는 <a>의 진단은 core(값 층)의 몫이다 — 파서는
			// 이름 allowlist만 보고, "속성이 있어야 한다"는 값의 요구이기 때문이다.
			spec.attrs = []string{"href"}
		case "span":
			// color·weight·size를 연다. size는 **감싼 텍스트보다 작거나 같은 값만** 뜻이 있고
			// 그 판정은 값 층(core)의 것이다: 런 하나가 커지면 웹의 줄 박스가 커지는데 PPTX
			// 줄간격은 문단 절대값(spcPts)이라 두 타깃의 글줄이 갈라진다. 작은 런은 블록의 줄
			// 박스 안에 앉으므로 그 근거가 성립하지 않는다 (`750+`의 작은 `+`).
			spec.attrs = []string{"color", "weight", "size"}
		case BreakElement:
			spec.model = modelVoid
		}
		specs[name] = spec
	}
	return specs
}

// slideChildren은 <slide>의 자식 어휘다. 순서가 진단 메시지의 순서다.
//
// 컴포넌트(`<text>`·`<shape>`·`<image>`)가 앞에 오고 `<note>`·텍스트 서브셋·레이아웃이
// 뒤따른다. 슬라이드 직속 도형은 세로 스택의 한 칸이다 — 표제 아래 구분면·배지가 그
// 자리다. 슬라이드 직속 이미지도 같은 자리이고 그것이 전면 그림(`image-full` 레이아웃)의
// 표기다.
//
// 레이아웃 컨테이너와 `<spacer>`가 들어 있다: 슬라이드 직속 `<spacer>`는 세로 스택에서
// 밀어내기가 되므로 뜻이 있다 (머리글 아래 여백을 늘려 꼬리글을 바닥에 붙이는 배치).
func slideChildren() []string {
	children := []string{TextElement, ShapeElement, ImageElement, TableElement, ChartElement, ConnectorElement, noteElement}
	children = append(children, blockElements...)
	children = append(children, listContainerElements...)
	children = append(children, layoutContainerElements...)
	return append(children, SpacerElement)
}

// elementSpec은 태그 하나의 문법 규칙이다. 값 어휘는 여기 없다 — 값은 core가 본다.
type elementSpec struct {
	// attrs는 허용 속성 이름이다. 진단 메시지에 그대로 실리므로 순서가 곧 문서 순서다.
	attrs []string
	model contentModel
	// children은 modelElements일 때 허용되는 자식 태그다.
	children []string
}

// contentModel은 요소가 담을 수 있는 것이다.
type contentModel int

const (
	// modelElements는 요소만 담는다. 공백 아닌 문자 데이터는 에러다 — 슬라이드에 그냥 쓴
	// 글자가 어떤 역할의 텍스트인지는 정할 방법이 없고, 닫힌 어휘는 그런 추측을 하지 않는다.
	modelElements contentModel = iota
	// modelText는 문자 데이터만 담는다 (<note>).
	modelText
	// modelInline은 문자 데이터와 인라인 태그를 담는다 (<text>·블록·<li>).
	modelInline
	// modelPre는 문자 데이터만 담되 **공백을 원문 그대로 보존**한다 (<pre>).
	// 여는 태그 직후·닫는 태그 직전의 개행 하나만 벗긴다 — HTML의 관례로, 저자가 <pre>
	// 다음 줄부터 쓰게 하기 위한 것이지 내용이 아니다 (trimPreEdges).
	modelPre
	// modelVoid는 아무것도 담지 않는다 (<br>). 자기 닫힘(<br/>)만 허용된다.
	modelVoid
)

// SlideDoc은 `.slide` 파일 하나의 파싱 결과다.
//
// Slide(덱 모델)와 나누는 이유: Index·Chapter·파일명 유도 ID는 파일 하나를 봐서는 알 수 없고,
// Skip은 덱에 들어가기 전에 소비되어 사라진다. 파일 안에서 알 수 있는 것만 여기 담고,
// 덱 조립은 파이프라인이 한다.
type SlideDoc struct {
	// File은 진단에 찍히는 경로다 (프로젝트 루트 기준 상대 경로).
	File string
	// Root는 루트 `<slide>` 요소다. 루트를 찾지 못하면 nil이다 (진단이 남는다).
	Root *Element
	// ID는 `<slide id>`의 값이다. 속성이 없으면 빈 문자열 — 파일명 유도는 호출자 몫이다
	// (SlideFile.DerivedID).
	ID string
	// IDSpan은 ID가 정해진 자리다 — `id` 값이 있으면 그 값, 없으면(파일명 유도) 루트
	// `<slide>`의 태그 이름이다. 파일을 넘는 ID 충돌 진단이 `line:col`을 가져야 에디터에서
	// 바로 그 자리로 갈 수 있다.
	IDSpan Span
	// Skip은 `<slide skip>`이다. true면 렌더에서 제외된다. 제외 판정은 파서가
	// 노출하고 덱에서 빼는 것은 파이프라인이 한다 — Index가 남은 슬라이드 기준이어야 하므로.
	Skip bool
	// Layout은 `<slide layout>`의 값이다 (원문 그대로). 속성이 없으면 빈 문자열이다 —
	// 어휘 검사(2종)는 core가 한다: 레이아웃 어휘는 테마와 함께 자라므로
	// 파서가 알 수 있는 것이 아니다 (fillSlide의 주석과 같은 경계). 파서는 문법만 본다
	// (값 없는 `<slide layout>`·빈 값 `layout=""`).
	Layout string
	// LayoutSpan은 layout 값 구간이다 — core의 어휘 진단이 값에 밑줄을 긋게 한다.
	LayoutSpan Span
	// Justify는 `<slide justify>`의 값이다 (원문 그대로, 속성이 없으면 빈 문자열). 어휘
	// 6종과 "슬롯 모드에서는 에러"는 core가 본다 — layout과 같은 경계다.
	Justify string
	// JustifySpan은 justify 값 구간이다 (LayoutSpan과 같은 근거).
	JustifySpan Span
	// Enter는 `<slide enter>`의 값이다 (원문 그대로). 어휘는 core가 본다 — justify와 같은 경계.
	Enter string
	// EnterSpan은 enter 값 구간이다.
	EnterSpan Span
	// Invert는 `<slide invert>`다 — 그 장의 색 매핑(배경 · 글자 네 자리)을 뒤집는다.
	Invert bool
	// Transition은 `<slide transition>`의 값이다 (원문 그대로 — 어휘는 core가 본다).
	Transition string
	// TransitionSpan은 transition 값 구간이다.
	TransitionSpan Span
	// Notes는 `<note>`의 텍스트다.
	Notes string
	// Content는 `<slide>`의 콘텐츠 자식이다 (`<note>` 제외).
	Content []Node
	// Tokens는 이 파일의 원문 토큰 열이다 — 무손실 계약(Tokens.Source)이 여기까지 유지된다.
	Tokens Tokens
}

// Parse는 `.slide` 파일 하나를 파싱한다. file은 진단에 찍히는 경로다.
//
// 엄격 파서다: 닫힌 어휘 밖의 태그·속성 이름, 짝이 맞지 않는 태그, 인용 부호 미닫힘은 모두
// **line:col을 가진 진단**이 된다. 대신 첫 에러에서 멈추지 않고 가능한 만큼 계속 읽어 한
// 파일의 진단을 한 번에 모아 낸다 — 빌드를 돌릴 때마다 오타 하나씩 알려주는 파서는 저자의
// 시간을 오타 수만큼 쓴다. 편집 중이라 늘 깨져 있는 문서를 다루는 lsp의 요구와도 같다.
//
// 공백 접기: 문자 데이터의 ASCII 공백(스페이스·탭·CR·LF) 런은 한 칸으로 접고, 요소 양 끝의
// 공백은 잘라낸다 (HTML `white-space: normal` 관례). 이 접기를 **파서가** 하는 이유는 타깃
// 때문이다 — 웹은 브라우저가 접고 PPTX는 접는 개념이 아예 없어서, 렌더러에 맡기면 같은 원문이
// 타깃마다 다른 문자열이 된다. U+00A0 같은 비ASCII 공백은 저자가 의도해 넣은 문자이므로
// 건드리지 않는다.
func Parse(file, src string) (*SlideDoc, Diagnostics) {
	p := &parser{file: file, toks: Tokenize(src)}
	doc := &SlideDoc{File: file, Tokens: p.toks}
	doc.Root = p.parseDocument()
	if doc.Root == nil {
		p.errorf(SpanOf(StartPos(), StartPos()), "missing root element <%s>", RootElement)
	} else {
		p.fillSlide(doc)
	}
	// 진단은 소스 순서로 낸다 — 저자가 위에서 아래로 고칠 수 있게. 닫히지 않은 태그처럼
	// 나중에 발견되는 진단이 앞을 가리키기 때문에 정렬이 필요하다.
	sort.SliceStable(p.diags, func(i, j int) bool {
		return p.diags[i].Span.Start.Offset < p.diags[j].Span.Start.Offset
	})
	return doc, p.diags
}

// parser는 토큰 열을 훑는 커서다. 트리비아는 건너뛰며 읽는다 (토큰 열은 무손실이라 원문을
// 그대로 복원할 수 있다).
type parser struct {
	file  string
	toks  Tokens
	i     int
	diags Diagnostics
}

func (p *parser) errorf(span Span, format string, args ...any) {
	p.diags = append(p.diags, Errorf(p.file, span, format, args...))
}

// peek은 트리비아를 건너뛴 다음 토큰이다. 닫히지 않은 주석은 지나면서 한 번 진단한다 —
// 렉서는 실패하지 않으므로 그 진단은 이 층의 몫이다.
func (p *parser) peek() Token {
	for p.i < len(p.toks) {
		tok := p.toks[p.i]
		if !tok.Kind.IsTrivia() {
			return tok
		}
		if tok.Kind == TokenCommentUnterminated {
			p.errorf(tok.Span, "comment is not closed with `-->`")
		}
		p.i++
	}
	return Token{Kind: TokenEOF}
}

// take는 다음 의미 토큰을 소비한다.
func (p *parser) take() Token {
	tok := p.peek()
	if tok.Kind != TokenEOF {
		p.i++
	}
	return tok
}

// parseDocument는 파일 전체를 읽고 루트 요소를 돌려준다.
func (p *parser) parseDocument() *Element {
	var root *Element
	var stack []*Element
	for {
		switch tok := p.peek(); tok.Kind {
		case TokenEOF:
			// 남은 여는 태그는 전부 닫히지 않았다. 문서 순서(바깥부터)로 보고한다.
			for _, open := range stack {
				p.errorf(open.NameSpan, "tag <%s> is not closed", open.Name)
			}
			return root

		case TokenText, TokenCData, TokenCDataUnterminated:
			p.i++
			if tok.Kind == TokenCDataUnterminated {
				p.errorf(tok.Span, "CDATA section is not closed with `%s`", cdataClose)
			}
			p.charData(tok, parentOf(stack))

		case TokenTagStart:
			elem, stillOpen := p.parseStartTag()
			if elem == nil {
				continue
			}
			parent := parentOf(stack)
			p.checkKnown(elem, parent)
			p.checkPlacement(elem, stack, root)
			p.checkRendered(elem, stack)
			switch {
			case parent != nil:
				parent.Children = append(parent.Children, elem)
			case root == nil:
				root = elem
			}
			if stillOpen {
				if _, known := elementSpecs[elem.Name]; known && specOf(elem).model == modelVoid {
					// void 요소는 자기 닫힘이 표준이다. 진단하되 스택에는 올린다 — 짝이 되는
					// </br>가 "여는 태그가 없다"는 두 번째 진단을 만들지 않게.
					p.errorf(elem.NameSpan, "void tag <%s> cannot have content (write <%s/>)", elem.Name, elem.Name)
				}
				stack = append(stack, elem)
			} else {
				p.closeElement(elem)
			}

		case TokenTagStartClose:
			name, tagSpan := p.parseEndTag()
			if name == "" {
				continue
			}
			depth := -1
			for i := len(stack) - 1; i >= 0; i-- {
				if stack[i].Name == name {
					depth = i
					break
				}
			}
			if depth < 0 {
				p.errorf(tagSpan, "no matching opening tag for </%s>", name)
				continue
			}
			// 안쪽에 열린 채 남은 요소는 닫히지 않았다. 각각 진단하고 함께 닫아 뒤따르는
			// 문서를 계속 읽는다.
			for i := len(stack) - 1; i > depth; i-- {
				p.errorf(stack[i].NameSpan, "tag <%s> must be closed before </%s>", stack[i].Name, name)
				p.closeElement(stack[i])
			}
			stack[depth].Span.End = tagSpan.End
			p.closeElement(stack[depth])
			stack = stack[:depth]

		default:
			// 렉서는 문자 데이터 위치에서 다른 종류를 내지 않는다. 방어적으로 건너뛴다.
			p.i++
		}
	}
}

// charData는 문자 데이터 토큰을 부모의 콘텐츠 모델에 맞게 처리한다.
func (p *parser) charData(tok Token, parent *Element) {
	if parent != nil {
		spec, known := elementSpecs[parent.Name]
		if !known {
			// 미지의 태그는 이미 진단됐다. 그 안의 내용까지 진단해 하나의 오타를 여러 줄로
			// 부풀리지 않는다.
			return
		}
		switch spec.model {
		case modelText, modelInline:
			parent.Children = append(parent.Children, &Text{Value: p.charDataValue(tok, true), Span: tok.Span})
			return
		case modelPre:
			// pre 안에서는 접지 않는다 — 개행·연속 공백이 곧 내용이다. 양 끝 개행 규칙은
			// 닫힐 때 적용된다 (trimPreEdges).
			parent.Children = append(parent.Children, &Text{Value: p.charDataValue(tok, false), Span: tok.Span})
			return
		case modelVoid:
			// void 위반은 여는 태그에서 이미 진단됐다 (parseDocument). 내용까지 진단해
			// 하나의 오용을 여러 줄로 부풀리지 않는다.
			return
		}
	}
	if strings.Trim(tok.Text, asciiSpace) == "" {
		// 요소만 담는 모델에서 들여쓰기·줄바꿈은 내용이 아니다.
		return
	}
	if parent == nil {
		p.errorf(dataSpan(tok), "character data outside any element (write content inside <%s>)", RootElement)
		return
	}
	// 고칠 방법은 **부모가** 정한다: 글자를 담을 수 있는 자식이 부모마다 다르기 때문이다.
	// `<text>로 감싼다`를 모든 부모에게 말하면 표 안에서는 따를 수 없는 조언이 된다
	// (`<table>`의 자식은 행뿐이고 `<tr>`의 자식은 셀뿐이다 — tableChildren·tableRowChildren).
	switch KindOf(parent.Name) {
	case KindListContainer:
		p.errorf(dataSpan(tok), "character data cannot appear directly inside <%s> (wrap it in <%s>)", parent.Name, ListItemElement)
	case KindTable:
		p.errorf(dataSpan(tok), "character data cannot appear directly inside <%s> (wrap it in <%s>)", parent.Name, TableRowElement)
	case KindTableRow:
		p.errorf(dataSpan(tok), "character data cannot appear directly inside <%s> (wrap it in <%s>)", parent.Name, TableDataCellElement)
	case KindChart:
		// 차트 안의 글자는 라벨이고 라벨은 점의 속성이다 — `<text>로 감싼다`고 말하면 저자가
		// 차트 안에 텍스트를 놓을 수 있다고 듣는다 (자식은 점뿐이다).
		p.errorf(dataSpan(tok), "character data cannot appear directly inside <%s> (wrap it in <%s>)", parent.Name, ChartPointElement)
	default:
		p.errorf(dataSpan(tok), "character data cannot appear directly inside <%s> (wrap it in <text>)", parent.Name)
	}
}

// charDataValue는 문자 데이터 토큰이 모델에 싣는 값이다.
//
// CDATA는 **원문 그대로**다 — 그 안에서 참조를 풀지 않는 것이 CDATA의 뜻이다. 공백 접기만
// 부모의 콘텐츠 모델을 따른다: `<pre>` 안에서는 보존되고 `<p>` 안에서는 접힌다. 같은 자리에
// 쓴 두 표기(참조와 CDATA)가 접기에서 갈라지면 저자가 표기마다 다른 규칙을 외워야 한다.
func (p *parser) charDataValue(tok Token, collapse bool) string {
	if tok.Kind == TokenText {
		return decodeCharData(tok.Text, tok.Span, collapse, p.reportCharRef)
	}
	body := cdataBody(tok.Text)
	if collapse {
		return collapseSpace(body)
	}
	return body
}

// reportCharRef는 참조 모양인데 어휘 밖인 조각을 진단한다. 구간은 그 조각을 가리킨다.
func (p *parser) reportCharRef(span Span, ref string) {
	p.errorf(span, "unsupported character reference %s (available: %s, `&#decimal;`, `&#xhex;`)",
		ref, charRefList())
}

// parseStartTag는 여는 태그를 읽는다. stillOpen은 요소가 열린 채로 남는지다 —
// 자기 닫는 태그와 이미 진단된 깨진 태그는 스택에 올리지 않는다.
func (p *parser) parseStartTag() (elem *Element, stillOpen bool) {
	open := p.take() // `<`
	nameTok := p.peek()
	if nameTok.Kind != TokenName {
		// 렉서가 `<`를 여는 태그로 내는 것은 글자가 뒤따를 때뿐이므로(startsMarkup) 이
		// 자리는 지금 닿지 않는다 — 두 층이 어긋나는 날을 위한 방어막으로 남긴다.
		// 마크업이 아닌 `<`는 여기까지 오지 않고 문자 데이터가 된다.
		p.errorf(open.Span, "missing tag name after `<`")
		p.skipToTagEnd()
		return nil, false
	}
	p.i++
	elem = &Element{
		Name:     nameTok.Text,
		NameSpan: nameTok.Span,
		Span:     SpanOf(open.Span.Start, nameTok.Span.End),
	}
	for {
		switch tok := p.peek(); tok.Kind {
		case TokenTagEnd:
			p.i++
			elem.Span.End = tok.Span.End
			return elem, true
		case TokenTagEndSelfClose:
			p.i++
			elem.Span.End = tok.Span.End
			return elem, false
		case TokenName:
			p.parseAttr(elem)
		case TokenEquals:
			p.errorf(tok.Span, "missing attribute name before `=`")
			p.i++
		case TokenString, TokenStringUnterminated:
			p.errorf(tok.Span, "attribute value without an attribute name")
			p.i++
		case TokenInvalid:
			p.errorf(tok.Span, "invalid character %q inside a tag", tok.Text)
			p.i++
		case TokenEOF:
			p.errorf(elem.NameSpan, "file ended inside tag <%s> before `>`", elem.Name)
			return elem, false
		default:
			// `<`나 `</`가 태그 안에 나왔다 — 이 태그의 `>`가 없다는 뜻이다. 여기서 태그를
			// 끝내고 그 토큰은 문서 흐름으로 넘긴다 (렉서가 `<`를 넘겨 준 이유와 같다).
			p.errorf(elem.NameSpan, "tag <%s> is not closed with `>`", elem.Name)
			return elem, true
		}
	}
}

// parseAttr는 속성 하나를 읽어 elem에 붙인다.
func (p *parser) parseAttr(elem *Element) {
	nameTok := p.take()
	attr := Attr{Name: nameTok.Text, NameSpan: nameTok.Span}
	if p.peek().Kind == TokenEquals {
		p.i++
		attr.HasValue = true
		switch value := p.peek(); value.Kind {
		case TokenString:
			p.i++
			attr.Value, attr.ValueSpan = unquote(value)
		case TokenStringUnterminated:
			p.i++
			p.errorf(value.Span, "value of attribute %s has no closing quote", attr.Name)
			attr.Value, attr.ValueSpan = unquote(value)
		case TokenName:
			p.i++
			p.errorf(value.Span, "attribute values must be quoted (%s=%q)", attr.Name, value.Text)
			attr.Value, attr.ValueSpan = value.Text, value.Span
		default:
			// 값이 아예 없는 속성은 모델에 넣지 않는다 — 진단은 이미 나갔고, 뒤 층이
			// `<slide id>`(값 없는 불리언)와 구별할 수 없는 반쪽 속성을 보게 하지 않는다.
			p.errorf(nameTok.Span, "attribute %s has `=` but no value", attr.Name)
			return
		}
	}

	if attr.HasValue {
		// 속성 값도 문자 데이터와 같은 참조 규칙을 따른다 — `alt="a &lt; b"`가
		// 동작하고 `href="?a=1&b=2"`는 그대로 남는다. 접지는 않는다: 값 안의 공백은 저자가
		// 쓴 그대로가 뜻이고(`pad="sm md"`), 렉서가 줄 끝에서 멈추므로 접을 개행도 없다.
		attr.Value = decodeCharData(attr.Value, attr.ValueSpan, false, p.reportCharRef)
	}
	if _, dup := elem.Attr(attr.Name); dup {
		p.errorf(attr.NameSpan, "duplicate attribute %s (only the first one is used)", attr.Name)
		return
	}
	p.checkAttrName(elem, attr)
	elem.Attrs = append(elem.Attrs, attr)
}

// parseEndTag는 닫는 태그를 읽는다. 이름을 찾지 못하면 빈 이름을 돌려준다.
func (p *parser) parseEndTag() (name string, tagSpan Span) {
	open := p.take() // `</`
	nameTok := p.peek()
	if nameTok.Kind != TokenName {
		p.errorf(open.Span, "missing tag name after `</`")
		p.skipToTagEnd()
		return "", open.Span
	}
	p.i++
	name = nameTok.Text
	extra := false
	for {
		switch tok := p.peek(); tok.Kind {
		case TokenTagEnd, TokenTagEndSelfClose:
			p.i++
			if extra {
				p.errorf(nameTok.Span, "closing tag </%s> cannot have attributes", name)
			}
			return name, SpanOf(open.Span.Start, tok.Span.End)
		case TokenEOF, TokenTagStart, TokenTagStartClose:
			p.errorf(nameTok.Span, "closing tag </%s> is not closed with `>`", name)
			return name, SpanOf(open.Span.Start, nameTok.Span.End)
		default:
			extra = true
			p.i++
		}
	}
}

// skipToTagEnd는 회복용으로 태그의 나머지를 버린다.
func (p *parser) skipToTagEnd() {
	for {
		switch p.peek().Kind {
		case TokenTagEnd, TokenTagEndSelfClose:
			p.i++
			return
		case TokenEOF, TokenTagStart, TokenTagStartClose:
			return
		default:
			p.i++
		}
	}
}

// checkPlacement는 요소가 놓인 자리를 검증한다. stack은 아직 열려 있는 조상들이다 —
// <a> 안의 <a>처럼 부모만 봐서는 잡을 수 없는 규칙이 있어 부모가 아니라 스택을 받는다.
func (p *parser) checkPlacement(elem *Element, stack []*Element, root *Element) {
	if _, known := elementSpecs[elem.Name]; !known {
		return // 미지의 태그는 이미 진단됐다 — 같은 오타로 진단을 두 번 내지 않는다.
	}
	parent := parentOf(stack)
	if parent == nil {
		switch {
		case elem.Name != RootElement:
			p.errorf(elem.NameSpan, "the root element must be <%s> (write <%s> inside it)", RootElement, elem.Name)
		case root != nil:
			p.errorf(elem.NameSpan, "a file must have exactly one root element <%s> (one slide per file)", RootElement)
		}
		return
	}
	spec, known := elementSpecs[parent.Name]
	if !known {
		return
	}
	switch spec.model {
	case modelText, modelPre:
		p.errorf(elem.NameSpan, "only character data is allowed inside <%s> (<%s> is not allowed)", parent.Name, elem.Name)
		return
	case modelVoid:
		return // void 위반은 여는 태그에서 이미 진단됐다 (parseDocument).
	}
	// <li>는 어디에 있든 리스트 컨테이너의 자식이어야 한다 — 자리별 일반 메시지보다
	// "리스트 안에 쓴다"가 고칠 방법을 곧장 말한다.
	if KindOf(elem.Name) == KindListItem && KindOf(parent.Name) != KindListContainer {
		p.errorf(elem.NameSpan, "<%s> can only be a child of <%s>/<%s>",
			ListItemElement, listContainerElements[0], listContainerElements[1])
		return
	}
	// <tr>/<th>/<td>도 <li>와 같다 — 어디에 있든 각자의 부모(표·행)의 자식이어야 하고,
	// 자리별 일반 메시지보다 "표/행 안에 쓴다"가 고칠 방법을 곧장 말한다.
	if KindOf(elem.Name) == KindTableRow && KindOf(parent.Name) != KindTable {
		p.errorf(elem.NameSpan, "<%s> can only be a child of <%s>", TableRowElement, TableElement)
		return
	}
	// <point>도 같은 규칙이다 — 어디에 있든 차트의 자식이어야 한다. 자리별 일반 메시지
	// ("<slide>의 자식으로 <point>는 올 수 없다")는 저자가 붙여 놓을 곳을 말하지 않는다.
	if KindOf(elem.Name) == KindChartPoint && KindOf(parent.Name) != KindChart {
		p.errorf(elem.NameSpan, "<%s> can only be a child of <%s>", ChartPointElement, ChartElement)
		return
	}
	if KindOf(elem.Name) == KindTableCell && KindOf(parent.Name) != KindTableRow {
		p.errorf(elem.NameSpan, "<%s>/<%s> can only be children of <%s>",
			TableHeaderCellElement, TableDataCellElement, TableRowElement)
		return
	}
	if spec.model == modelInline {
		switch KindOf(elem.Name) {
		case KindInline:
			if elem.Name != "a" {
				return
			}
			// <a> 안의 <a>는 에러다 — 겹친 링크는 "무엇을 클릭했는가"가 정의되지 않고,
			// PPTX hlinkClick은 런 하나에 링크 하나다. 사이에 다른 인라인이 끼어도 잡히게
			// 부모가 아니라 열린 조상 전체를 본다.
			for _, open := range stack {
				if open.Name == "a" {
					p.errorf(elem.NameSpan, "<a> cannot be nested inside <a>")
					return
				}
			}
			return
		case KindListContainer:
			if KindOf(parent.Name) == KindListItem {
				// PPTX의 들여쓰기 단계(lvl) 매핑 없이 열면 웹에서만 동작한다 — 한 타깃만
				// 그리는 어휘는 열지 않는다.
				p.errorf(elem.NameSpan, "<%s> cannot be nested inside <%s> — nested lists are not supported yet",
					elem.Name, parent.Name)
				return
			}
			p.errorf(elem.NameSpan, "block tag <%s> is not allowed inside <%s> (blocks do not nest)", elem.Name, parent.Name)
			return
		case KindBlock:
			// 블록은 슬라이드 **또는 레이아웃 컨테이너** 직속이다 (layoutChildren).
			// 컨테이너는 블록을 담지만 블록 안의 블록은 에러다: "문단 안의 문단"은
			// 두 타깃 어디에도 대응하는 구조가 없다. 이 비대칭이 의도다.
			p.errorf(elem.NameSpan, "block tag <%s> is not allowed inside <%s> (blocks do not nest)", elem.Name, parent.Name)
			return
		default:
			p.errorf(elem.NameSpan, "only character data and inline tags are allowed inside <%s> (<%s> is not allowed)",
				parent.Name, elem.Name)
			return
		}
	}
	for _, allowed := range spec.children {
		if allowed == elem.Name {
			return
		}
	}
	p.errorf(elem.NameSpan, "<%s> cannot contain <%s> (available tags: %s)",
		parent.Name, elem.Name, tagList(spec.children))
}

// UnrenderedVocabMessage는 "렌더러가 아직 그리지 못하는 어휘" 진단의 핵 문구다. 상수로
// 두는 이유는 진단을 내는 곳과 걸러 보는 곳(파서·값 층의 테스트 — 표 어휘는 이 두 층까지
// 열려 있다)이 같은 문자열을 쓰게 하는 것이다.
const UnrenderedVocabMessage = "is vocabulary that is not implemented yet"

// checkRendered는 렌더러가 아직 그리지 못하는 어휘를 **파스 시점에** 막는다.
//
// 표를 다 쓴 뒤 렌더 단계에서 빌드가 죽으면 진단의 실패다: 저자는 자기 문서가 유효한 줄
// 알고 계속 쓰고, 실패는 파일 이름도 줄 번호도 없는 렌더 에러로 온다. 어휘
// 카탈로그(KindRendered)가 그 사실을 이미 알고 있으므로 태그가 있는 자리에서 말한다 — 같은
// 목록을 렌더러 디스패치와 대조하는 테스트가 카탈로그와 렌더러가 갈라지는 것을 막는다
// (render 패키지).
func (p *parser) checkRendered(elem *Element, stack []*Element) {
	if KindRendered(KindOf(elem.Name)) {
		return
	}
	// 표 하나가 진단 여럿이 되지 않게 **가장 바깥 요소에서만** 말한다 — `<table>` 안의
	// `<tr>`·`<td>`는 같은 사실의 반복이고, 저자가 고칠 것은 하나다.
	for _, open := range stack {
		if !KindRendered(KindOf(open.Name)) {
			return
		}
	}
	p.errorf(elem.NameSpan, "<%s> %s — the parser and value layer know it, but the web and PPTX renderers cannot draw it yet (rejected here so the render does not crash)",
		elem.Name, UnrenderedVocabMessage)
}

// checkAttrName은 속성 이름이 태그의 닫힌 어휘 안에 있는지 본다. 값은 보지 않는다.
func (p *parser) checkAttrName(elem *Element, attr Attr) {
	spec, known := elementSpecs[elem.Name]
	if !known {
		return
	}
	for _, allowed := range spec.attrs {
		if allowed == attr.Name {
			return
		}
	}
	// 예정된 속성은 오타와 **다른 문장으로** 말한다: 이름은 맞는데 구현이
	// 아직인 자리를 "알 수 없는 속성"으로 말하면 저자가 이름을 의심하며 헤맨다 (plannedAttrs).
	for _, planned := range plannedAttrs {
		if planned.Tag == elem.Name && planned.Name == attr.Name {
			p.errorf(attr.NameSpan, "%s on <%s> %s — %s opens that attribute",
				attr.Name, elem.Name, PlannedAttrMessage, planned.Opens)
			return
		}
	}
	if len(spec.attrs) == 0 {
		p.errorf(attr.NameSpan, "<%s> takes no attributes (%s)", elem.Name, attr.Name)
		return
	}
	hint := ""
	if guess, ok := Suggest(attr.Name, spec.attrs); ok {
		hint = " — did you mean " + guess + "?"
	}
	p.errorf(attr.NameSpan, "unknown attribute %s on <%s>%s (available attributes: %s)",
		attr.Name, elem.Name, hint, strings.Join(spec.attrs, ", "))
}

// closeElement는 요소를 닫으며 콘텐츠 모델에 따른 정규화를 마친다.
func (p *parser) closeElement(elem *Element) {
	switch specOf(elem).model {
	case modelText, modelInline:
		elem.Children = trimEdgeSpace(elem.Children)
	case modelPre:
		elem.Children = trimPreEdges(elem.Children)
	}
}

// fillSlide는 루트 `<slide>`에서 슬라이드 메타데이터를 뽑는다.
//
// `id`와 `skip`은 외관이 아니라 **구조** 속성이다 — 어떤 슬라이드가 덱에 들어가고 어떤 앵커를
// 갖는지를 정하므로 값 해석까지 파서가 한다. `<text>`의 role/size/color/align은 원문 그대로
// 남겨 core가 해석한다: 외관 값의 어휘는 테마와 함께 자라기 때문에 파서가 알 수
// 있는 것이 아니다.
func (p *parser) fillSlide(doc *SlideDoc) {
	root := doc.Root
	if root.Name != RootElement {
		return
	}
	doc.IDSpan = root.NameSpan
	if attr, ok := root.Attr("id"); ok {
		switch {
		case !attr.HasValue:
			p.errorf(attr.NameSpan, "the id attribute needs a value (id=\"architecture\")")
		case attr.Value == "":
			p.errorf(attr.ValueSpan, "the id attribute is empty (remove it to derive the id from the file name)")
		default:
			doc.ID = attr.Value
			doc.IDSpan = attr.ValueSpan
		}
	}
	if attr, ok := root.Attr("skip"); ok {
		switch {
		case !attr.HasValue, attr.Value == "true":
			// 값 없는 `<slide skip>`은 참이다 — HTML 불리언 속성 관례.
			doc.Skip = true
		case attr.Value == "false":
		default:
			p.errorf(attr.ValueSpan, "skip must be true or false (%q)", attr.Value)
		}
	}
	if attr, ok := root.Attr(InvertAttr); ok {
		switch {
		case !attr.HasValue, attr.Value == "true":
			// skip과 같은 HTML 불리언 관례다.
			doc.Invert = true
		case attr.Value == "false":
		default:
			p.errorf(attr.ValueSpan, "invert must be true or false (%q)", attr.Value)
		}
	}
	if attr, ok := root.Attr("layout"); ok {
		// 파서는 문법만 본다 — 값 없는 `<slide layout>`과 빈 값 `layout=""`은 뜻을 정할 수
		// 없는 형태이므로 여기서 진단하고, **어휘(content·canvas) 검사는 core**다 (레이아웃
		// 어휘는 테마와 함께 자라므로 파서가 알 수 없다). 원문 값과 구간을 그대로 나른다.
		switch {
		case !attr.HasValue:
			p.errorf(attr.NameSpan, "the layout attribute needs a value (layout=\"canvas\")")
		case attr.Value == "":
			p.errorf(attr.ValueSpan, "the layout attribute is empty (remove it to continue with content)")
		default:
			doc.Layout = attr.Value
			doc.LayoutSpan = attr.ValueSpan
		}
	}
	if attr, ok := root.Attr("justify"); ok {
		// layout과 같은 경계다 — 문법(값 있음·비지 않음)만 보고 어휘 6종은 core가 본다.
		switch {
		case !attr.HasValue:
			p.errorf(attr.NameSpan, "the justify attribute needs a value (justify=\"between\")")
		case attr.Value == "":
			p.errorf(attr.ValueSpan, "the justify attribute is empty (remove it to stack from the top)")
		default:
			doc.Justify = attr.Value
			doc.JustifySpan = attr.ValueSpan
		}
	}
	if attr, ok := root.Attr(EnterAttr); ok {
		switch {
		case !attr.HasValue:
			p.errorf(attr.NameSpan, "the enter attribute needs a value (enter=\"rise\")")
		case attr.Value == "":
			p.errorf(attr.ValueSpan, "the enter attribute is empty (remove it for no entrance)")
		default:
			doc.Enter = attr.Value
			doc.EnterSpan = attr.ValueSpan
		}
	}
	if attr, ok := root.Attr(TransitionAttr); ok {
		switch {
		case !attr.HasValue:
			p.errorf(attr.NameSpan, "the transition attribute needs a value (transition=\"fade\")")
		case attr.Value == "":
			p.errorf(attr.ValueSpan, "the transition attribute is empty (remove it to use the theme's transition)")
		default:
			doc.Transition = attr.Value
			doc.TransitionSpan = attr.ValueSpan
		}
	}

	var note *Element
	for _, child := range root.Children {
		elem, ok := child.(*Element)
		if !ok {
			continue
		}
		if _, known := elementSpecs[elem.Name]; !known {
			continue // 이미 진단됐다. 진단이 있는 문서는 렌더에 쓰이지 않는다.
		}
		if elem.Name != noteElement {
			doc.Content = append(doc.Content, elem)
			continue
		}
		if note != nil {
			p.errorf(elem.NameSpan, "only one <%s> is allowed per slide (using the one at %s)",
				noteElement, note.NameSpan.Start)
			continue
		}
		note = elem
		doc.Notes = elem.TextContent()
	}
}

// specOf는 요소의 문법 규칙이다. 미지의 태그는 요소만 담는 모델로 취급한다 — 그 태그의
// 진단은 이미 나갔고, 안쪽 내용으로 진단을 부풀리지 않는 쪽을 고른다.
func specOf(elem *Element) elementSpec {
	return elementSpecs[elem.Name]
}

// ElementAttrs는 태그가 여는 속성 이름 전체다 (진단·문서 순서). 미지의 태그는 nil이다.
//
// 내보내는 이유는 값 층의 **엔진 버그 가드**다 (core.catalogGap): "카탈로그가 열었는데 값 층이
// 모르는 속성"은 소리를 내야 하지만, 파서가 이미 "알 수 없는 속성"으로 거절한 이름은 값 층의
// 몫이 아니다 — 그것까지 가드에 닿으면 저자의 오타 하나가 진단 둘이 되고 둘째가 엔진을
// 탓한다. 두 경우를 가르는 판정이 이 목록이므로, 값 층은 여기서 읽는다 (ContainerAttrs·
// ShapeAttrs와 같은 이유로 목록은 한 벌이다).
func ElementAttrs(name string) []string {
	spec, known := elementSpecs[name]
	if !known {
		return nil
	}
	attrs := make([]string, len(spec.attrs))
	copy(attrs, spec.attrs)
	return attrs
}

func parentOf(stack []*Element) *Element {
	if len(stack) == 0 {
		return nil
	}
	return stack[len(stack)-1]
}

// knownElementNames는 태그 어휘 전체다. 정렬해서 메시지가 결정적이게 한다.
func knownElementNames() []string {
	names := make([]string, 0, len(elementSpecs))
	for name := range elementSpecs {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// checkKnown은 어휘 밖의 태그를 진단한다 — **그 자리에 쓸 수 있는 태그**를 나열하고, 오타로 보이면
// 후보를 하나 말한다.
//
// 목록을 부모가 정하는 이유: 어휘 전체(50개 남짓)를 나열하면 저자는 그 안에서 지금 자리에 맞는 것을
// 다시 골라야 한다 — `<ul>` 안의 오타에 답은 `<li>` 하나다. 슬롯 · 속성 진단이 이미 부모의 목록을
// 말하는 것과 같은 태도다. 후보는 HTML 습관 · 그 자리의 목록 · 어휘 전체 순서로 찾는다 (SuggestTag).
func (p *parser) checkKnown(elem *Element, parent *Element) {
	if _, known := elementSpecs[elem.Name]; known {
		return
	}
	where, allowed := allowedHere(parent)
	if len(allowed) == 0 {
		// 태그가 올 수 없는 자리다 — 어떤 태그 이름을 권해도 다음 진단이 될 뿐이다.
		p.errorf(elem.NameSpan, "unknown tag <%s> (only character data is allowed %s)", elem.Name, where)
		return
	}
	hint := ""
	if guess, ok := SuggestTag(elem.Name, allowed, knownElementNames()); ok {
		hint = " — did you mean <" + guess + ">?"
	}
	p.errorf(elem.NameSpan, "unknown tag <%s>%s (tags available %s: %s)", elem.Name, hint, where, tagList(allowed))
}

// allowedHere는 부모 아래에 쓸 수 있는 태그와 그 자리를 말하는 구절이다. 문자 데이터만 담는 부모는
// 빈 목록이다.
func allowedHere(parent *Element) (where string, allowed []string) {
	if parent == nil {
		return "at the root", []string{RootElement}
	}
	spec, known := elementSpecs[parent.Name]
	if !known {
		// 부모도 어휘 밖이다 (이미 진단됐다) — 자리를 모르므로 어휘 전체를 말한다.
		return "anywhere", knownElementNames()
	}
	where = "inside <" + parent.Name + ">"
	switch spec.model {
	case modelElements:
		return where, spec.children
	case modelInline:
		return where, inlineElements
	}
	return where, nil
}

func tagList(names []string) string {
	quoted := make([]string, len(names))
	for i, name := range names {
		quoted[i] = "<" + name + ">"
	}
	return strings.Join(quoted, ", ")
}

// asciiSpace는 접기 대상 공백이다 — 렉서가 태그 안에서 구분자로 보는 집합과 같다.
const asciiSpace = " \t\n\r"

// collapseSpace는 공백 런을 한 칸으로 접는다. 바이트 단위로 훑어도 안전하다: UTF-8
// 연속 바이트는 ASCII 범위에 들어오지 않는다.
func collapseSpace(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	space := false
	for i := 0; i < len(s); i++ {
		if strings.IndexByte(asciiSpace, s[i]) >= 0 {
			space = true
			continue
		}
		if space {
			b.WriteByte(' ')
			space = false
		}
		b.WriteByte(s[i])
	}
	if space {
		b.WriteByte(' ')
	}
	return b.String()
}

// trimEdgeSpace는 자식 목록 **양 끝**의 공백만 잘라낸다.
//
// 요소 경계의 공백은 저자가 들여쓰기로 넣은 것이지 내용이 아니다 — `<text>\n  제목\n</text>`는
// "제목"이다. 반면 자식 사이의 공백은 남긴다: 인라인 태그에서
// `<strong>배포</strong> 자동화`의 한 칸은 내용이다.
func trimEdgeSpace(children []Node) []Node {
	if len(children) == 0 {
		return children
	}
	if first, ok := children[0].(*Text); ok {
		first.Value = strings.TrimLeft(first.Value, " ")
	}
	if last, ok := children[len(children)-1].(*Text); ok {
		last.Value = strings.TrimRight(last.Value, " ")
	}
	out := children[:0]
	for _, child := range children {
		if text, ok := child.(*Text); ok && text.Value == "" {
			continue
		}
		out = append(out, child)
	}
	return out
}

// trimPreEdges는 <pre> 양 끝의 개행을 **하나씩만** 벗긴다 (HTML 관례).
//
// `<pre>` 직후의 개행은 저자가 다음 줄부터 쓰기 위한 것이지 내용이 아니고, 닫는 태그
// 직전의 개행도 같다. 딱 하나만 벗기므로 빈 줄로 시작하는 코드(`\n\ncode`)는 표현할 수
// 있다. 개행 뒤에 들여쓰기가 붙은 `code\n  </pre>`의 개행은 "직전"이 아니므로 벗기지
// 않는다 — pre 안의 공백은 전부 내용이라는 원칙이 우선한다.
func trimPreEdges(children []Node) []Node {
	if len(children) == 0 {
		return children
	}
	if first, ok := children[0].(*Text); ok {
		first.Value = cutEdgeNewline(first.Value, strings.CutPrefix)
	}
	if last, ok := children[len(children)-1].(*Text); ok {
		last.Value = cutEdgeNewline(last.Value, strings.CutSuffix)
	}
	out := children[:0]
	for _, child := range children {
		if text, ok := child.(*Text); ok && text.Value == "" {
			continue
		}
		out = append(out, child)
	}
	return out
}

// cutEdgeNewline은 한쪽 끝의 개행 하나를 벗긴다. CRLF도 개행 하나다.
func cutEdgeNewline(s string, cut func(string, string) (string, bool)) string {
	for _, newline := range []string{"\r\n", "\n"} {
		if rest, ok := cut(s, newline); ok {
			return rest
		}
	}
	return s
}

// unquote는 값 토큰에서 값과 인용 부호를 제외한 구간을 뽑는다. 값 안에 줄바꿈은 있을 수
// 없으므로(렉서가 줄 끝에서 멈춘다) 끝 지점은 같은 줄에서 한 칸 물러난 곳이다.
func unquote(tok Token) (string, Span) {
	start, end := tok.Span.Start, tok.Span.End
	start.Offset++
	start.Col++
	if tok.Kind == TokenString {
		end.Offset--
		end.Col--
		return tok.Text[1 : len(tok.Text)-1], SpanOf(start, end)
	}
	// 닫히지 않은 값은 여는 부호만 벗긴다.
	return tok.Text[1:], SpanOf(start, end)
}

// dataSpan은 문자 데이터에서 공백을 뺀 구간이다 — 진단이 들여쓰기가 아니라 글자를 가리키게 한다.
func dataSpan(tok Token) Span {
	body := strings.TrimLeft(tok.Text, asciiSpace)
	start := advance(tok.Span.Start, tok.Text[:len(tok.Text)-len(body)])
	body = strings.TrimRight(body, asciiSpace)
	return SpanOf(start, advance(start, body))
}
