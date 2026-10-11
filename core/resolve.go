package core

import (
	"errors"
	"io/fs"
	"sort"
	"strconv"
	"strings"

	"github.com/k2sebeom/balpyo/soml"
)

// Resolve는 덱 모델의 외관을 해석한다 — 렌더러가 보기 전에 모든 외관 값이 확정되어야
// 한다는 계약을 이 함수가 진다. 결과는 요소의 Appearance와 덱의 Theme(토큰 표)에 담긴다.
//
// 해석 순서는 4단 캐스케이드다:
//
//	인라인 속성 > variant > 컴포넌트 기본값 > 역할/토큰
//
// 역할 기본값이 완전한 외관을 깔고 위 세 단이 부분 패치로 얹힌다 — 병합은 mergeText
// (cascade.go) 한 곳에서만 일어난다. 도형 외관도 **같은 기계의 복제**를 쓴다 (mergeShape) —
// 도형에는 역할이 없어 컴포넌트 기본값이 가장 아래 단이라는 점만 갈라진다 (resolveShape).
// 역할 기본값·컴포넌트 기본값·variant 표는 전부 **활성 테마의 components.json**에서 온다
// (theme_package.go) — 엔진에는 어떤 외관 값도 하드코딩되어 있지 않다.
//
// 이 순서를 렌더러가 아니라 여기서 한 번만 적용하기 때문에 웹과 PPTX가 같은 외관을 본다.
// computed-style 역추론은 없다 — 외관은 스키마에서 양 타깃으로 순방향 컴파일된다.
//
// 값 어휘도 이 층이 소유한다. 파서는 태그·속성 **이름**만 보고 값은 원문으로 남기므로
// (soml/node.go), 역할 5종·정렬 4종·색 토큰 같은 **값 층의 닫힌 어휘**를 여기서 검사한다.
// 진단은 Attr.ValueSpan을 근거로 파서와 같은 정밀도의 line:col을 갖는다.
//
// fsys는 **프로젝트 루트**다 (`ResolveAssets`와 같은 인자, 같은 기준). 테마 이름을
// 찾는 자리가 둘이기 때문에 받는다: 내장 테마와 `themes/<이름>/`. nil이면 내장 테마만 보인다 —
// 설정 없이 `Deck`을 만드는 길이 그대로 남는다 (테스트·프리뷰·미래의 API).
func Resolve(deck *soml.Deck, fsys fs.FS) soml.Diagnostics {
	if deck == nil {
		return nil
	}
	// 테마 선택은 **덱의 요청**이다: `balpyo.json`의 theme(또는 CLI의 -theme)가
	// deck.ThemeName으로 실려 오고, 이 줄이 그 이름으로 패키지를 로드한다. 빈 이름은 기본
	// 테마다 — 설정 없이 Deck을 만드는 길이 있다.
	//
	// 어휘 밖 이름은 **에러 + 해석 중단**이다. 폴백 테마로 계속하면 산출물이 저자가 고르지 않은
	// 테마로 나가고, 그 사실은 슬라이드를 열어야 드러난다. 진단이 `balpyo.json`을 가리키는
	// 이유는 고칠 자리가 거기라는 것이다 (테마를 선언하지 않은 크롬 경고와 같은 태도).
	//
	// 이름을 찾는 자리는 둘이다 — 내장 테마와 `themes/<이름>/`. fsys가 그래서 여기
	// 있고, nil이면 내장 테마만 보인다.
	name := deck.ThemeName
	if name == "" {
		name = DefaultTheme
	}
	// 토큰과 컴포넌트 외관 표를 한 번에 로드해 slide 루프에 나른다. 어휘 밖 이름·이름 충돌·
	// 순환 `extends`·불완전한 로컬 테마가 모두 이 한 에러로 온다 — 로컬 테마는
	// 저자가 방금 쓴 파일이므로 내장 테마처럼 panic할 수 없다.
	pkg, err := loadThemeNamed(fsys, name)
	if err != nil {
		// 테마 파일 안의 틀린 자리는 그 파일을 가리킨다 — 이름을 요청한 `balpyo.json`이 아니다.
		var fe *themeFileError
		if errors.As(err, &fe) {
			return soml.Diagnostics{fe.diagnostic()}
		}
		return soml.Diagnostics{soml.Errorf(ConfigFile, soml.Span{}, "%s", err)}
	}
	deck.Theme = pkg.tokens
	// 레이아웃 배치 CSS도 토큰 표에 싣는다 — 웹 스테이지 CSS와 지오메트리 패스가 재는 박스가
	// 같은 한 표에서 나와야 두 타깃의 배치가 갈라지지 않는다 (soml.Theme.Layouts).
	deck.Theme.Layouts = pkg.layoutStyles()
	// 슬롯 영역 배치도 같은 표에 싣는다 — 슬롯을 쓴 슬라이드의 스테이지가
	// `.balpyo-slots` 모디파이어로 이 영역 규칙을 받는다 (render/web). 흐름 배치(Layouts)와
	// 겨누는 선택자만 다르고 계약은 같다: 한 표에서 웹 CSS와 지오메트리 패스가 나온다.
	deck.Theme.SlotLayouts = pkg.slotLayoutStyles()

	// 덱 크롬 계획은 슬라이드 루프 **앞**이다: 설정과 테마 선언만 보므로 덱에 한
	// 번 결정되고, 테마가 침묵했을 때의 경고도 그래서 덱에 한 번 나간다.
	chromePlan, chromeDiags := resolveChrome(deck, pkg)
	diags := append(pkg.lint, chromeDiags...)
	for _, slide := range deck.Slides {
		r := resolver{theme: deck.Theme, comps: pkg.components, themeAssets: pkg.assets, themeChain: pkg.chain, file: slide.Source}
		// 레이아웃 어휘는 슬라이드마다 도는 이 루프 머리에서 확정한다 — 파일 하나로
		// 판정되므로 Validate(파일을 넘어야 아는 것 — 슬라이드 ID 충돌)가 아니라 여기다.
		r.slideLayout(slide)
		// 확정된 layout을 resolver에 싣는다 — canvas 슬라이드의 직속 자식에서 `x`/`y`가
		// 유효한지 판정하려면 배치 해석이 슬라이드의 layout을 알아야 한다 (coordParentOK).
		r.layout = slide.Layout
		// 크롬 확정도 layout이 정해진 **뒤**다 — 제외 규칙의 입력이 확정된 레이아웃이라서,
		// 어휘 밖 이름이 content로 폴백한 슬라이드도 폴백된 레이아웃으로 판정된다.
		slide.Chrome = slideChrome(chromePlan, pkg, slide, len(deck.Slides))
		// 배경도 확정 레이아웃의 것이다. 값을 복사해 싣는 이유는 확정 모델이 슬라이드마다 독립이어야
		// 한다는 것이다 — 테마의 선언 값을 슬라이드들이 나눠 가리키면 한 슬라이드를 고친 코드가 전부를 고친다.
		if bg := pkg.backgrounds.For(slide.Layout); bg != nil {
			copied := *bg
			slide.Background = &copied
		}
		for _, node := range slide.Content {
			// 슬라이드 직속 노드의 부모는 없다 — 배치 해석이 부모를 보는 이유는
			// resolveLayout의 주석에 있다 (span은 <grid>의 자식에서만 뜻이 있다).
			r.node(node, nil)
		}
		// 슬라이드 수준 justify는 콘텐츠 **다음**이다 — 슬롯 모드인지는 자식의 확정 슬롯이
		// 정하고(slideUsesSlots), 테마가 못 받는 슬롯은 실리지 않으므로(resolveSlot) 그
		// 슬라이드는 흐름 모드로 판정되어 justify가 유효해진다.
		r.slideJustify(slide)
		r.resolveConnectors()
		// 등장은 콘텐츠가 다 확정된 뒤에 센다 — 차트가 펼쳐진 트리에서 문서 순서가 정해진다.
		r.resolveEntrances(slide)
		r.resolveTransition(slide)
		// 진단은 파일 안에서 소스 순서로 낸다 — 저자가 위에서 아래로 고칠 수 있게. 역할을
		// 먼저 해석하고 나머지 속성을 훑기 때문에 발견 순서가 소스 순서와 어긋난다.
		sort.SliceStable(r.diags, func(i, j int) bool {
			return r.diags[i].Span.Start.Offset < r.diags[j].Span.Start.Offset
		})
		diags = append(diags, r.diags...)
	}
	// 테마의 그림 표는 슬라이드 **뒤**에 확정된다 — `theme:` 그림이 선언(배경 · 로고) 뒤에 이어 실린다.
	// 복사하는 이유는 확정 모델이 패키지의 표를 나눠 가리키지 않게 하는 것이다.
	deck.Theme.Assets = append(soml.Assets(nil), pkg.assets.used...)
	return diags
}

// resolver는 슬라이드 하나의 해석 상태다.
type resolver struct {
	theme soml.Theme
	// comps는 로드된 컴포넌트 외관 표다 (components.json) — 캐스케이드의 컴포넌트 기본값·
	// variant 단이 여기서 나온다 — 폴백은 없다 (엔진에 외관 값이 없다).
	comps components
	// themeAssets · themeChain은 활성 테마의 병합된 `assets/` 색인과 상속 체인이다 — `<image
	// src="theme:…">`가 여기서 풀린다 (resolver.themeImageSrc). 슬라이드마다 resolver가 새로 만들어져도
	// 색인은 덱에 하나다: 같은 그림을 여러 슬라이드가 써도 표에 한 번 실린다.
	themeAssets *assetResolver
	themeChain  []string
	file        string
	// layout은 이 슬라이드의 확정 레이아웃이다 (slideLayout이 채운다). 배치 검증이 슬라이드를
	// 알아야 하는 유일한 자리가 canvas다: `x`/`y`는 <stack>의 자식이거나 canvas 슬라이드의
	// 직속 자식에서만 쓸 수 있는데, 후자는 부모가 요소가 아니므로(parent == nil) 부모만으로는
	// 판정할 수 없다 (coordParentOK).
	layout soml.SlideLayout
	diags  soml.Diagnostics
	// ids는 이 슬라이드에서 이미 쓴 도형 `id`와 그 자리다 (중복 진단이 처음 자리를 가리켜야
	// 한다). 상태가 resolver에 있는 것이 곧 검사 범위가 슬라이드라는 뜻이다 — 근거는
	// resolver.shapeID의 주석에 있다.
	ids map[string]soml.Span
	// idElems는 같은 이름의 요소다 — 연결선이 `from`·`to`를 푸는 표다 (resolveConnectors).
	idElems map[string]*soml.Element
	// connectors는 이 슬라이드의 연결선이다 (문서 순서). 슬라이드를 다 읽은 **뒤**에 푼다 — 뒤에서
	// 선언한 상자도 가리킬 수 있어야 한다.
	connectors []*soml.Element
	// revealTurn은 이 슬라이드에서 지금까지 `reveal`을 가진 리스트 항목의 수다 — 값 없는
	// `reveal`이 받을 차례다 (resolveReveal). 상태가 resolver에 있는 것이 곧 스텝 번호의
	// 범위가 슬라이드라는 뜻이다 (ids와 같은 태도).
	revealTurn int
}

func (r *resolver) errorf(span soml.Span, format string, args ...any) {
	r.diags = append(r.diags, soml.Errorf(r.file, span, format, args...))
}

func (r *resolver) warnf(span soml.Span, format string, args ...any) {
	r.diags = append(r.diags, soml.Warningf(r.file, span, format, args...))
}

// slideLayout은 슬라이드 레이아웃 어휘(2종)를 확정한다. 파서는 layout 값을 원문으로 남기고
// (레이아웃 어휘는 테마와 함께 자란다) 문법만 보므로, 어휘 검사는 값 층인
// 여기다. 어휘 밖 값은 **line:col을 가진** 진단이다 — Slide.LayoutSpan이 값 구간을 가리켜
// 진단이 부호가 아니라 값에 밑줄을 긋는다.
//
// 생략된 슬라이드(빈 문자열)는 DefaultSlideLayout(content)으로 채운다 — 렌더러가 빈 값을
// 보지 않고, 속성을 명시한 슬라이드와 생략한 슬라이드가 같은 값을 갖는다. 어휘 밖 값도
// 폴백하지 않고 진단으로 막는 것이 계약이다 (`layout="titel"` 오타가 조용히 기본 레이아웃으로
// 렌더되면 닫힌 어휘가 막으려던 실패가 된다). 진단 뒤 기본값을 넣는 것은 회복용일 뿐이다:
// 진단이 있으면 빌드는 렌더에 닿지 않는다.
func (r *resolver) slideLayout(slide *soml.Slide) {
	if slide.Layout == "" {
		slide.Layout = soml.DefaultSlideLayout
		return
	}
	for _, layout := range soml.SlideLayouts() {
		if slide.Layout == layout {
			return
		}
	}
	r.errorf(slide.LayoutSpan, "unknown layout %q%s (available layouts: %s)",
		string(slide.Layout), didYouMean(string(slide.Layout), layoutList()), layoutList())
	slide.Layout = soml.DefaultSlideLayout
}

// slideJustify는 슬라이드 스테이지의 세로 분배를 확정한다 (`<slide justify>`).
//
// 어휘는 컨테이너의 `justify`와 **같은 6종**이고 진단도 같은 함수(boxJustify)에서 나온다 —
// 같은 뜻에 두 이름·두 메시지를 두지 않는다. 그래서 값 하나를 Attr로 감싸 넘긴다: 슬라이드
// 모델에는 요소 참조가 없으므로(해석이 끝난 값이다) 값과 구간을 Attr 모양으로 되돌린다.
//
// **슬롯 모드에서는 에러다.** 두 근거가 겹친다: 그 문맥의 세로 리듬은 테마의 `*.slots.css`가
// 소유하고(`align-content`), grid에서 `justify-content`는 **가로 축**이라 같은 표기가 문맥마다
// 다른 축을 뜻하게 된다. 조용히 무시하면 저자는 아무 일도 일어나지 않는 이유를 알 수 없다.
//
// 슬롯 판정이 콘텐츠 순회 **다음**이어야 하는 이유: 테마가 못 받는 슬롯은 확정 모델에 실리지
// 않으므로(resolveSlot) 그 슬라이드는 실제로 흐름 모드가 되고, 거기서는 justify가 유효하다.
func (r *resolver) slideJustify(slide *soml.Slide) {
	if slide.Justify == "" {
		return
	}
	if slideUsesSlots(slide) {
		r.errorf(slide.JustifySpan,
			"justify cannot be used on a slot-mode slide (the theme's %s.slots.css owns the vertical rhythm) — use it on slides without slots",
			string(slide.Layout))
		slide.Justify = ""
		return
	}
	justify, ok := r.boxJustify(soml.Attr{
		Name:      "justify",
		Value:     string(slide.Justify),
		ValueSpan: slide.JustifySpan,
		HasValue:  true,
	})
	if !ok {
		slide.Justify = ""
		return
	}
	slide.Justify = justify
}

// slideUsesSlots는 이 슬라이드의 직계 자식 중 슬롯이 **확정된** 요소가 있는지다 — 슬롯 모드의
// 정의이고, render/web의 같은 이름 함수와 같은 한 줄을 본다 (판정 입력은 하나다).
func slideUsesSlots(slide *soml.Slide) bool {
	for _, node := range slide.Content {
		if elem, ok := node.(*soml.Element); ok && elem.Appearance != nil && elem.Appearance.Slot != "" {
			return true
		}
	}
	return false
}

// node는 콘텐츠 트리를 훑는다. 외관을 갖는 요소만 외관을 해석하고, **배치는 요소 종류와
// 무관하게 전부 지난다** (resolveLayout — 자식 배치 속성은 한 벌이므로 종류마다 표를
// 복제하지 않는다). 한 요소가 두 채널을 함께 쓰는 것은 정상이다 (`<p grow="1" size="30">`).
//
// 리스트 컨테이너(<ul>/<ol>)의 Appearance는 **텍스트 외관을 채우지 않는다**: 컨테이너는
// 항목을 담을 뿐이고 타이포그래피는 <li>(body 역할)에 붙는다. 컨테이너가 받는 외관 속성은
// 확정 값이 아니라 **항목들이 함께 받는 패치**다 (resolveList). 반면 배치는
// 컨테이너에도 붙는다 (`<row><ul basis="360">`은 두 열에 리스트를 놓는 흔한 배치다).
func (r *resolver) node(node soml.Node, parent *soml.Element) {
	elem, ok := node.(*soml.Element)
	if !ok {
		return
	}
	r.resolveLayout(elem, parent)
	r.resolveSlot(elem, parent)
	if elem.Name == soml.ConnectorElement {
		// 연결선은 슬라이드를 다 읽은 뒤에 푼다 (resolveConnectors) — 앞에서 아직 보지 않은 `id`를
		// 가리킬 수 있다. void이므로 아래 순회가 데려갈 자식이 없다.
		r.connectors = append(r.connectors, elem)
		return
	}
	if elem.Name == soml.ChartElement {
		// 여덟 번째 갈래이고 **유일하게 트리를 바꾸는** 갈래다: `<chart>`는 확정
		// 값을 얻는 대신 도형·텍스트의 하위 트리로 **펼쳐진다**. 끝나면 elem은 `<shape>`이므로
		// 바로 아래 도형 갈래가 뿌리를 평범한 배경 도형으로 해석하고, 합성된 자식은 이 함수의
		// 끝에 있는 순회가 저자가 쓴 것과 같은 경로로 데려간다.
		//
		// 배치·슬롯 **뒤**에 있는 이유는 진단의 이름이다: 그 두 해석이 내는 진단은 저자가 쓴
		// `<chart>`를 말해야 한다 (펼친 뒤에 지나면 저자가 쓰지 않은 `<shape>`를 듣는다).
		// 박스 속성은 도형과 같은 자리에서(resolveShape의 shapeAttrs) 해석되므로 순서가
		// 바뀌지 않는다 — resolveLayout은 도형에게도 배치 속성만 본다.
		r.resolveChart(elem)
	}
	if elem.Name == soml.ShapeElement {
		// 도형은 세 번째 갈래다 (외관·배치에 이어) — 자식은 텍스트 요소이므로 아래 순회가
		// 기존 텍스트 경로로 데려간다. 부모가 도형이라는 사실이 텍스트 해석을 바꾸지 않는다:
		// 바꾸는 것은 상속이고, 상속은 지금 구조에 없다.
		r.resolveShape(elem)
	}
	if elem.Name == soml.ImageElement {
		// 네 번째 갈래다. 도형과 나란히 두고 같은 기계를 쓰지만(캐스케이드·값 문법) 확정 값이
		// 형제인 이유는 담을 것이 다르다는 것이다 (soml.Appearance.Image). 그림은 void이므로
		// 아래 순회가 데려갈 자식이 없다.
		r.resolveImage(elem)
	}
	if elem.Name == soml.TableElement {
		// 다섯 번째 갈래다. 프레임의 배치·슬롯은 위에서(resolveLayout·resolveSlot) 이미 지났고,
		// resolveTable이 프레임 아래 행·셀·셀 안 인라인 런까지 **자기 서브트리를 통째로** 해석한다.
		// return하는 이유는 텍스트 역할 갈래와 같다: <tr>/<th>/<td>는 텍스트 역할이 없어 아래
		// 순회가 셀 안 인라인 런을 해석하지 못하므로(resolveInlines는 resolveText·resolveCell에서만
		// 불린다), 여기서 끝내고 표 해석에 순회를 넘긴다.
		r.resolveTable(elem)
		return
	}
	if soml.KindOf(elem.Name) == soml.KindListContainer {
		// 여섯 번째 갈래다. 표와 같은 이유로 return한다 — 컨테이너의 외관 속성이 항목의
		// 캐스케이드에 들어가므로(resolveList) 항목을 아래 순회에 맡기면 그 패치가 사라진다.
		r.resolveList(elem)
		return
	}
	if role, ok := r.textRole(elem); ok {
		r.resolveText(elem, role, textPatch{}, placedSlot(elem))
		if elem.Name == soml.PreElement {
			// 일곱 번째 갈래다. `<pre>`의 문자 데이터는 탭이 펴지고, `lang`이 있으면
			// **합성 인라인 런**이 된다 — 확정 스타일이 채워진 뒤이므로 합성 런은 그것을 부모로
			// 상속하고(아래 resolveInlines), 색만 다른 조각이 된다. 렌더러는 손대지 않는다.
			r.resolveCode(elem)
		}
		// 텍스트 역할 요소의 자식은 문자 데이터와 인라인 태그뿐이다 (파서의 modelInline) —
		// 인라인 런 스타일은 블록의 확정 스타일에서 상속하므로 여기서 트리 순회가 끝난다.
		r.resolveInlines(elem, elem.Appearance.Text)
		return
	}
	for _, child := range elem.Children {
		r.node(child, elem)
	}
}

// resolveSlot은 요소의 슬롯 배정을 검증하고 확정한다.
//
// 슬롯은 **슬라이드 직계 자식에만** 유효하다 (parent == nil). 파서는 slot 속성을 슬라이드
// 직계 자식이 될 수 있는 요소 전부에 열어 두지만(SlotAttr — 태그별 allowlist는 요소가 어디에
// 놓였는지 모른다), "직계 자식인가"와 "그 레이아웃에 그 슬롯이 있는가"는 부모와 슬라이드
// 레이아웃을 아는 이 값 층이 판정한다. 중첩 요소의 slot은 뜻이 없으므로 진단이다 — 조용히
// 무시하면 저자는 콘텐츠가 왜 그 영역에 안 놓이는지 알 수 없다.
//
// 어휘 밖 슬롯 이름은 **line:col을 가진** 빌드 에러다 (layout과 같은 태도 —
// slideLayout). 폴백하지 않는 것이 계약이다: `slot="titel"` 오타가 조용히 기본 슬롯으로 떨어지면
// 콘텐츠가 엉뚱한 자리에 놓이고 그 실패는 렌더를 볼 때까지 안 보인다. 슬롯이 없는 레이아웃
// (canvas)의 slot도 같은 경로로 잡힌다 (LayoutSlot이 false).
//
// slot을 쓰지 않은 직계 자식은 손대지 않는다 — 빈 Slot이 곧 "슬롯 배치 없음"이라(기본 슬롯은
// 렌더 매핑의 몫), 슬롯을 쓰지 않은 덱의 산출물이 이 검증 때문에 달라지지 않는다.
//
// # 테마가 못 받는 슬롯은 경고 + 미탑재다
//
// 어휘에 있는 슬롯 이름이라도 **테마에 그 레이아웃의 슬롯 영역이 없으면**(`*.slots.css`) 놓일
// 자리가 없다. 그 판정을 여기서 한 번만 하고 슬롯을 **싣지 않는** 것이 계약이다: 렌더러 두
// 곳이 각자 조건을 적었을 때 웹은 조용히 무효, PPTX는 플레이스홀더 발행이 되어 한 소스가 두
// 산출물에서 다른 그림이 됐다. 확정 모델에 싣지 않으면 두 렌더러가 보는 입력이
// `Appearance.Slot` 하나로 합쳐진다.
//
// 에러가 아니라 경고인 이유: 슬롯 이름은 어휘에 있고 오타도 아니다 — 부족한 것은 테마이고,
// 테마를 바꾸면 같은 소스가 그대로 뜻을 얻는다 (모르는 variant와 같은 규칙).
// `--strict`가 승격한다.
func (r *resolver) resolveSlot(elem *soml.Element, parent *soml.Element) {
	attr, ok := elem.Attr(soml.SlotAttr)
	if !ok {
		return
	}
	if parent != nil {
		r.errorf(attr.ValueSpan, "slot is only allowed on direct children of the slide (<%s> is nested)", elem.Name)
		return
	}
	if _, ok := soml.LayoutSlot(r.layout, attr.Value); !ok {
		r.errorf(attr.ValueSpan, "layout %q has no slot %q%s (available slots: %s)",
			string(r.layout), attr.Value, didYouMean(attr.Value, slotList(r.layout)), slotList(r.layout))
		return
	}
	if _, ok := r.theme.SlotLayouts[string(r.layout)]; !ok {
		r.warnf(attr.ValueSpan, "the theme has no slot areas for layout %q, so slot=%q cannot be used (layouts/%s.slots.css) — this slide renders with flow layout",
			string(r.layout), attr.Value, string(r.layout))
		return
	}
	elemAppearance(elem).Slot = attr.Value
}

// textRole은 요소의 텍스트 역할이다. `<text>`는 role 속성(생략하면 body)이, 블록·<li>는
// 언어의 등가 표(soml.TagTextRole — `<p>` ≡ `<text role="body">`)가 역할을 정한다.
// 블록은 role 속성을 받지 않으므로 (파서 allowlist — `<p role="title">`은 두 표기의
// 등가를 깬다) 두 출처가 겹치는 요소는 없다.
func (r *resolver) textRole(elem *soml.Element) (soml.TextRole, bool) {
	if elem.Name == soml.TextElement {
		return r.role(elem), true
	}
	return soml.TagTextRole(elem.Name)
}

// resolveText는 텍스트 요소(`<text>`·블록·<li>)의 외관을 4단 캐스케이드로 확정한다:
//
//	인라인 속성 > variant > 컴포넌트 기본값(슬롯 > 태그) > 역할/토큰
//
// 역할 기본값(완전한 외관)을 바닥에 깔고, 위 세 단을 낮은 단부터 부분 패치로 얹는다.
// 컴포넌트 기본값 단은 둘로 겹친다: 태그별(textComponentDefault — h3의 크기, blockquote의 색)
// 위에 **놓인 자리**(textSlotDefault — 인용 레이아웃의 quote 슬롯)가 온다. 자리가 태그보다 구체적이다:
// 같은 `<blockquote>`가 흐름에서는 본문 크기, 인용 슬롯에서는 인용 크기다.
// 병합은 mergeText(cascade.go) 한 곳에서만 일어난다 — 우선순위는 아래 네 줄의 순서가
// 전부이고, 다른 어디에도 없다.
//
// inherited는 **부모가 물려준 패치**다 (리스트 항목만 쓴다 — resolveList). 자기 인라인
// 속성보다 **먼저** 얹는 이유는 우선순위의 방향이다: 가까운 선언이 이긴다. `<li>`는 지금
// 외관 속성을 받지 않으므로(여는 속성은 순서인 `reveal` 하나다) 두 단이 겹치는
// 경우가 아직 없지만, 순서를 여기서 못 박아 두면 `<li>`에 외관 속성을 여는 날 우선순위를
// 다시 논의할 필요가 없다.
//
// slot은 요소가 놓인 슬롯이다 (없으면 빈 문자열) — 리스트 항목은 컨테이너의 슬롯을 받는다.
func (r *resolver) resolveText(elem *soml.Element, role soml.TextRole, inherited textPatch, slot string) {
	style := r.textRoleDefaults(role)
	mergeText(&style, r.textComponentDefault(elem.Name))
	mergeText(&style, r.textSlotDefault(slot))
	mergeText(&style, r.variantPatch(elem))
	mergeText(&style, inherited)
	mergeText(&style, r.inlinePatch(elem))
	elemAppearance(elem).Text = &style
}

// resolveList는 리스트 컨테이너와 그 항목을 해석한다.
//
// 컨테이너의 외관 속성(size/color/align/anchor)은 컨테이너의 확정 값이 되지 않고 **모든
// 항목이 함께 받는 패치**가 된다. 근거는 외관이 목록 단위의 성질이라는 것이다: 목록 하나의
// 글자 크기는 목록의 성질이고, 항목마다 갈리면 그것은 목록이 아니라 문단 여럿이다. 컨테이너
// 자신에게 확정 스타일을 채우지 않는 이유는 소비자가 없다는 것이다 — 컨테이너는 텍스트를
// 담지 않으므로(자식은 `<li>`뿐) 값을 채우면 두 렌더러가 "컨테이너의 값"과 "항목의 값"
// 중 어느 쪽이 진실인지 각자 고르게 된다.
//
// 패치는 여기서 **한 번** 만들어 모든 항목에 같은 것을 준다: 항목마다 다시 읽으면 값 진단이
// 항목 수만큼 나고(저자가 쓴 속성은 하나다), 같은 속성을 두 번 해석할 이유도 없다.
func (r *resolver) resolveList(list *soml.Element) {
	patch := r.inlinePatch(list)
	for _, child := range list.Children {
		item, ok := child.(*soml.Element)
		if !ok {
			// 컨테이너 직속 문자 데이터는 파서가 막는다 (modelElements) — 여기 오는 노드는
			// 없지만, 파서를 거치지 않은 덱을 조용히 통과시키지 않고 건너뛴다.
			continue
		}
		role, ok := r.textRole(item)
		if !ok {
			// 컨테이너의 자식은 `<li>`뿐이다 (파서의 children allowlist) — 역할 없는 자식은
			// 파서를 거치지 않은 덱이고, 렌더러가 매핑 없음으로 멈춘다.
			continue
		}
		r.resolveText(item, role, patch, placedSlot(list))
		r.resolveInlines(item, item.Appearance.Text)
		// 순서는 외관 뒤에 본다 — 항목의 유일한 자기 속성이고 외관 패치와
		// 겹치지 않는다.
		r.resolveReveal(item)
	}
}

// elemAppearance는 요소의 확정 외관·배치 자리다 (없으면 만든다).
//
// 구조체를 통째로 대입하지 않는 이유: 한 요소가 외관과 배치를 함께 갖는다
// (`<p grow="1" size="30">`). 대입하는 코드가 둘이면 나중에 해석되는 쪽이 앞의 결과를
// 조용히 지운다 — 그 실패는 산출물에서 배치나 외관 하나가 사라지는 것으로만 드러난다.
func elemAppearance(elem *soml.Element) *soml.Appearance {
	if elem.Appearance == nil {
		elem.Appearance = &soml.Appearance{}
	}
	return elem.Appearance
}

// variantPatch는 `variant` 속성을 테마의 **텍스트** variant 표에서 찾는다.
//
// 모르는 variant는 에러가 아니라 **경고 + 빈 패치(컴포넌트 기본 외관 폴백)**다:
// variant 어휘는 테마의 사유물이라, 다른 테마로 만든 콘텐츠가 이 테마에서 빌드되지 않으면
// 콘텐츠의 테마 간 이식성이 깨진다. 엄격함이 필요한 곳은 --strict가 에러로 승격한다.
func (r *resolver) variantPatch(elem *soml.Element) textPatch {
	attr, ok := r.variantAttr(elem)
	if !ok {
		return textPatch{}
	}
	patch, known := r.textVariantPatch(attr.Value)
	if !known {
		r.warnUnknownVariant(attr, r.textVariantNames())
		return textPatch{}
	}
	return patch
}

// variantAttr는 쓸 값이 있는 `variant` 속성이다 — 텍스트와 도형이 공유한다. 값 없음·빈 값의
// 진단은 hasValue가 소유한다 (모르는 variant는 경고지만 값 없는 variant는 에러다: 뜻을 정할
// 수 없는 형태이기 때문이다).
func (r *resolver) variantAttr(elem *soml.Element) (soml.Attr, bool) {
	attr, ok := elem.Attr("variant")
	if !ok || !r.hasValue(attr) {
		return soml.Attr{}, false
	}
	return attr, true
}

// warnUnknownVariant는 모르는 variant 경고다 — **요소 종류에 맞는 어휘를 나열한다.**
//
// variant 어휘가 텍스트용·도형용 두 벌이라 생기는 요구다:
// `<shape variant="kicker">`에 텍스트 목록을 나열하면 저자를 그 테마에 없는 방향으로 보낸다.
// 두 표가 같은 이름을 가져도 무관하고(요소 종류가 어느 표를 볼지 정한다), 그래서 목록은
// 호출자가 준다.
//
// 문구는 여기 한 곳에만 있다: 폴백 규칙이 두 채널에서 같으므로 문구가 갈라질 이유가 없고,
// 갈라지면 저자가 두 경고를 다른 규칙으로 읽는다. 어휘 종류를 문구에 적지 않은 이유는
// 나열된 목록이 이미 그것을 말하기 때문이다 (`variant: kicker, lead`).
func (r *resolver) warnUnknownVariant(attr soml.Attr, names []string) {
	list := strings.Join(names, ", ")
	// 후보는 괄호로 단다 — 이 문장은 이미 대시로 "그래도 계속한다"를 잇고 있다.
	hint := ""
	if guess, ok := soml.Suggest(attr.Value, names); ok {
		hint = " (did you mean " + strconv.Quote(guess) + "?)"
	}
	r.warnf(attr.ValueSpan, "variant %q is not in the theme%s — continuing with the default appearance (variants in this theme: %s)",
		attr.Value, hint, list)
}

// 아래 넷은 텍스트 캐스케이드의 컴포넌트 기본값·variant 단을 **로드된 표**(r.comps)에서 읽는다.
//
// 폴백이 없다: 역할 5종은 로더가 요구하므로(theme_require.go) 표에 없을 수 없고,
// 하드코딩 표를 폴백으로 남겨 두면 그것이 곧 엔진이 외관 값을 아는 자리가 된다 — 테마의 누락이
// 조용히 minimal의 값으로 메워지고, 저자는 자기 테마가 반만 먹는 것을 슬라이드에서 본다.
func (r *resolver) textRoleDefaults(role soml.TextRole) soml.TextStyle {
	return r.comps.textRoles[role]
}

func (r *resolver) textComponentDefault(tag string) textPatch {
	if patch, ok := r.comps.textComponentDefaults[tag]; ok {
		return patch
	}
	return textPatch{}
}

// textSlotDefault는 이 슬라이드 레이아웃의 슬롯 기본값이다 (슬롯이 없으면 빈 패치).
func (r *resolver) textSlotDefault(slot string) textPatch {
	if slot == "" {
		return textPatch{}
	}
	return r.comps.textSlotDefaults[slotDefaultKey(r.layout, slot)]
}

// placedSlot은 요소에 **실린** 슬롯이다 — resolveSlot이 검증을 지나 확정 모델에 실은 값만 본다. 테마가
// 그 레이아웃의 슬롯 영역을 갖지 않아 흐름 배치로 렌더되는 슬라이드에서는 비어 있고, 그때는 슬롯
// 기본값도 얹지 않는다 (자리가 없는데 자리의 크기를 받을 이유가 없다).
func placedSlot(elem *soml.Element) string {
	if elem.Appearance == nil {
		return ""
	}
	return elem.Appearance.Slot
}

func (r *resolver) textVariantPatch(name string) (textPatch, bool) {
	for _, v := range r.comps.textVariants {
		if v.name == name {
			return v.patch, true
		}
	}
	return textPatch{}, false
}

func (r *resolver) textVariantNames() []string {
	names := make([]string, len(r.comps.textVariants))
	for i, v := range r.comps.textVariants {
		names[i] = v.name
	}
	return names
}

// inlinePatch는 인라인 속성을 캐스케이드의 마지막 단으로 만든다. 값 어휘 검사와 진단은
// 속성별 해석 함수(size/color/align)가 그대로 소유한다 — 틀린 값은 패치에 실리지 않으므로
// 아래 단의 값이 남는다 (틀린 role이 기본 역할로 계속하는 것과 같은 정책).
func (r *resolver) inlinePatch(elem *soml.Element) textPatch {
	var patch textPatch
	for _, attr := range elem.Attrs {
		switch attr.Name {
		case "size":
			if size, ok := r.size(attr); ok {
				patch.size = &size
			}
		case "color":
			if color, ok := r.color(attr); ok {
				patch.color = &color
			}
		case "align":
			if align, ok := r.align(attr); ok {
				patch.align = &align
			}
		case soml.AnchorAttr:
			if anchor, ok := r.anchor(attr); ok {
				patch.anchor = &anchor
			}
		}
	}
	return patch
}

// role은 텍스트 역할을 해석한다. 값이 없거나 어휘 밖이면 진단하고 기본 역할로 계속한다 —
// 역할 하나가 틀렸다고 그 요소의 다른 속성 진단을 삼키면 저자가 빌드를 여러 번 돌게 된다.
func (r *resolver) role(elem *soml.Element) soml.TextRole {
	attr, ok := elem.Attr("role")
	if !ok {
		return soml.DefaultTextRole
	}
	if !r.hasValue(attr) {
		return soml.DefaultTextRole
	}
	for _, role := range soml.TextRoles() {
		if string(role) == attr.Value {
			return role
		}
	}
	r.errorf(attr.ValueSpan, "unknown role %q%s (available roles: %s)", attr.Value, didYouMean(attr.Value, roleList()), roleList())
	return soml.DefaultTextRole
}

// size는 폰트 크기를 해석한다. 값은 **무단위 수치 = 논리 스테이지 px**다.
func (r *resolver) size(attr soml.Attr) (float64, bool) {
	if !r.hasValue(attr) {
		return 0, false
	}
	raw := attr.Value
	if strings.HasSuffix(raw, "%") {
		// %를 막는 이유: 상대 폰트 크기를 PPTX로 번역할 곳이 없다. OOXML
		// `rPr@sz`는 절대값(1/100 pt)이고 "부모 텍스트 크기"라는 상속 기준도 없어서, 웹에서만
		// 동작하고 PPTX에서는 조용히 다른 크기가 되는 속성이 된다. 상대적 위계는 역할과
		// variant로 표현한다 — 그 둘은 양 타깃에 모두 대응이 있다.
		r.errorf(attr.ValueSpan, "size cannot use %% (%q) — PPTX font sizes are absolute, so a relative size has nothing to translate to; use a number in logical px or a role", raw)
		return 0, false
	}
	if unit := numberSuffix(raw); unit != "" {
		// 수치 뒤에 붙은 것만 "단위"라고 부른다. 수치가 아예 없으면 아래의 수치 진단으로
		// 내려보낸다 — `size="큼"`에 단위를 떼라고 하면 저자가 무엇을 고칠지 알 수 없다
		// (numberSuffix가 그 판정을 소유한다).
		r.errorf(attr.ValueSpan, "size is a unitless number in logical px (drop the %[2]q from %[1]q) — px, pt and em suffixes are not allowed", raw, unit)
		return 0, false
	}
	px, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		r.errorf(attr.ValueSpan, "size value is not a number (%q)", raw)
		return 0, false
	}
	if px <= 0 {
		r.errorf(attr.ValueSpan, "size must be greater than 0 (%q)", raw)
		return 0, false
	}
	return px, true
}

// color는 색을 해석한다. 토큰 이름은 **이름으로 남는다** — RGB로 눕히면 PPTX의 schemeClr
// 참조를 만들 수 없다 (soml.Color의 주석).
func (r *resolver) color(attr soml.Attr) (soml.Color, bool) {
	if !r.hasValue(attr) {
		return soml.Color{}, false
	}
	return r.colorValue(attr.Name, attr.Value, attr.ValueSpan)
}

// colorValue는 색 값 하나를 해석한다 — 속성이 아니라 (이름, 값, 구간)을 받는 이유는
// lengthValue와 같다: 값 하나가 색을 조각으로 담는다 (`border="1 solid muted"`). 진단은
// 문제가 된 **조각**을 가리킨다.
//
// 도형의 `fill`·`border` 색이 텍스트 `color`와 이 함수를 공유하는 것이 계약이다 — 색 문법이
// 속성마다 따로 있으면 같은 값이 자리에 따라 다르게 읽히고, 그 차이는 두 산출물을 나란히
// 볼 때까지 보이지 않는다.
func (r *resolver) colorValue(name, raw string, span soml.Span) (soml.Color, bool) {
	if strings.IndexByte(raw, '(') >= 0 {
		// 색 함수(gradient·shade·tint·alpha)는 여기서 갈린다.
		// 단색과 코드가 갈라져도 진단은 같은 함수(colorValue)를 지난다 — 그래야 `fill`·
		// `border` 색·텍스트 `color`가 같은 함수 어휘를 쓴다.
		return r.colorFunc(name, raw, span)
	}
	return r.simpleColor(name, raw, span)
}

// simpleColor는 단색 하나를 해석한다 — 토큰명 또는 #리터럴이다 (함수·그라디언트 아님).
//
// colorValue와 나눈 이유: 그라디언트 스톱과 변형 함수의 밑색은 **단색만** 받는다 (중첩
// 불허). 그 자리에서 다시 함수를 허용하면 `gradient(shade(a,10%) b)` 같은
// 표기가 문법으로 새어 들어오고, 그 표기의 타깃 매핑은 v1에 없다.
func (r *resolver) simpleColor(name, raw string, span soml.Span) (soml.Color, bool) {
	if strings.HasPrefix(raw, "#") {
		rgb, ok := hexRGB(raw[1:])
		if !ok {
			r.errorf(span, "a color literal has the form #RRGGBB or #RGB (%q)", raw)
			return soml.Color{}, false
		}
		return soml.LiteralColor(rgb), true
	}
	if _, known := r.theme.Color(raw); !known {
		names := strings.Join(r.theme.ColorNames(), ", ")
		r.errorf(span, "%s: color token %q is not in the theme%s (available tokens: %s, or #RRGGBB)",
			name, raw, didYouMean(raw, names), names)
		return soml.Color{}, false
	}
	return soml.TokenColor(raw), true
}

// colorFunc는 색 함수 하나를 해석한다 (`gradient`/`shade`/`tint`/`alpha`).
//
// 닫는 괄호·함수 이름 어휘·중첩 금지를 여기서 한 번에 검사한 뒤 종류별 파서로 넘긴다 —
// 중첩 금지(인자 안에 다시 `(`가 없음)를 위에서 막으므로 종류별 파서는 인자에 단색만 온다고
// 믿을 수 있다.
func (r *resolver) colorFunc(name, raw string, span soml.Span) (soml.Color, bool) {
	open := strings.IndexByte(raw, '(')
	fn := raw[:open]
	if !strings.HasSuffix(raw, ")") {
		r.errorf(span, "color function %s(…) is missing the closing %q (%q)", fn, ")", raw)
		return soml.Color{}, false
	}
	inner := strings.TrimSpace(raw[open+1 : len(raw)-1])
	if strings.IndexByte(inner, '(') >= 0 {
		// 중첩은 v1에서 불허한다 — `shade(gradient(…),…)`·
		// `gradient(shade(…) …)` 둘 다 여기서 걸린다. 매핑을 1:1로 유지하려는 결정의 관문이다.
		r.errorf(span, "color functions cannot be nested (%q) — arguments are token names or # literal solid colors (v1)", raw)
		return soml.Color{}, false
	}
	switch fn {
	case "gradient":
		return r.gradientFunc(name, inner, span, raw)
	case "radial":
		return r.radialFunc(name, inner, span, raw)
	case string(soml.ModShade), string(soml.ModTint):
		return r.variantFunc(soml.ColorModKind(fn), name, inner, span, raw)
	case string(soml.ModAlpha):
		return r.alphaFunc(name, inner, span, raw)
	case patternFunc:
		// 무늬는 색이 아니라 두 색을 가진 채움이다 — 도형 fill이 colorValue보다 먼저 가로챈다 (shapeFill).
		// 여기까지 왔다면 무늬를 받지 않는 자리다.
		r.errorf(span, "%s cannot be a pattern (%q) — pattern(…) is only a shape fill", name, raw)
		return soml.Color{}, false
	}
	r.errorf(span, "unknown color function %q%s (available functions: gradient, radial, shade, tint, alpha) — %q",
		fn, didYouMean(fn, "gradient, radial, shade, tint, alpha"), raw)
	return soml.Color{}, false
}

// gradientFunc는 `gradient(색… [, 각도deg])`를 해석한다 — 색은 공백 구분·균등 스톱이고
// 각도는 생략 시 180(위→아래)이다.
func (r *resolver) gradientFunc(name, inner string, span soml.Span, raw string) (soml.Color, bool) {
	colorPart := inner
	anglePart := ""
	if c := strings.IndexByte(inner, ','); c >= 0 {
		colorPart = inner[:c]
		anglePart = strings.TrimSpace(inner[c+1:])
	}
	fields := strings.Fields(colorPart)
	if len(fields) < 2 {
		r.errorf(span, "gradient needs at least two colors (%q) — e.g. gradient(accent1 surface), or with an angle gradient(accent1 surface, 45deg)", raw)
		return soml.Color{}, false
	}
	stops, ok := r.gradientStops(name, fields, span)
	angle := gradientDefaultAngle
	if anglePart != "" {
		a, aok := r.gradientAngle(anglePart, span, raw)
		if !aok {
			ok = false
		} else {
			angle = a
		}
	}
	if !ok {
		return soml.Color{}, false
	}
	return soml.GradientColor(stops, angle), true
}

// gradientStops는 그라디언트의 스톱들이다 — 스톱은 단색이고, `/퍼센트`로 불투명도를 받는다 (`bg/0`은 투명한 bg).
// 불투명도는 alpha 변형과 같은 확정 값이 된다 (PPTX 스톱의 `a:alpha`).
func (r *resolver) gradientStops(name string, fields []string, span soml.Span) ([]soml.Color, bool) {
	ok := true
	stops := make([]soml.Color, 0, len(fields))
	for _, f := range fields {
		base, pct, hasPct := strings.Cut(f, "/")
		c, cok := r.simpleColor(name, base, span)
		if !cok {
			ok = false
			continue
		}
		if hasPct {
			n, err := strconv.ParseFloat(pct, 64)
			if err != nil || n < 0 || n > 100 || strings.HasSuffix(pct, "%") {
				r.errorf(span, "a gradient stop opacity is a percentage from 0 to 100 without %% (%q) — e.g. bg/0 or fg/40", f)
				ok = false
				continue
			}
			c = soml.VariantColor(c, soml.ModAlpha, n/100)
		}
		stops = append(stops, c)
	}
	return stops, ok
}

// radialFunc는 `radial(색… [, at X% Y%])`다 — 첫 색이 중심, 마지막 색이 가장 먼 모서리다.
func (r *resolver) radialFunc(name, inner string, span soml.Span, raw string) (soml.Color, bool) {
	colorPart, atPart, hasAt := strings.Cut(inner, ",")
	fields := strings.Fields(colorPart)
	if len(fields) < 2 {
		r.errorf(span, "radial needs at least two colors (%q) — e.g. radial(accent1 bg/0), or with a center radial(accent1 bg/0, at 20%% 80%%)", raw)
		return soml.Color{}, false
	}
	stops, ok := r.gradientStops(name, fields, span)
	cx, cy := 50.0, 50.0
	if hasAt {
		parts := strings.Fields(strings.TrimSpace(atPart))
		valid := len(parts) == 3 && parts[0] == "at"
		var x, y float64
		if valid {
			var errX, errY error
			x, errX = strconv.ParseFloat(strings.TrimSuffix(parts[1], "%"), 64)
			y, errY = strconv.ParseFloat(strings.TrimSuffix(parts[2], "%"), 64)
			valid = errX == nil && errY == nil && strings.HasSuffix(parts[1], "%") && strings.HasSuffix(parts[2], "%") &&
				x >= 0 && x <= 100 && y >= 0 && y <= 100
		}
		if !valid {
			r.errorf(span, "a radial center has the form `at X%% Y%%` with 0–100%% (%q) — e.g. radial(accent1 bg/0, at 20%% 80%%)", raw)
			ok = false
		} else {
			cx, cy = x, y
		}
	}
	if !ok {
		return soml.Color{}, false
	}
	return soml.RadialColor(stops, cx, cy), true
}

// gradientDefaultAngle은 각도를 생략한 그라디언트의 CSS 각도다 (위→아래).
const gradientDefaultAngle = 180.0

// gradientAngle은 그라디언트의 각도 인자를 해석한다 — `<수치>deg` 형태다.
func (r *resolver) gradientAngle(part string, span soml.Span, raw string) (float64, bool) {
	if !strings.HasSuffix(part, "deg") {
		r.errorf(span, "a gradient angle has the form `<number>deg` (%[2]q in %[1]q) — e.g. 45deg", raw, part)
		return 0, false
	}
	deg, err := strconv.ParseFloat(strings.TrimSpace(strings.TrimSuffix(part, "deg")), 64)
	if err != nil {
		r.errorf(span, "gradient angle is not a number (%[2]q in %[1]q)", raw, part)
		return 0, false
	}
	return deg, true
}

// variantFunc는 `shade(색, %)`·`tint(색, %)`를 해석한다 — 밑색 단색에 백분율 세기를 얹는다.
func (r *resolver) variantFunc(kind soml.ColorModKind, name, inner string, span soml.Span, raw string) (soml.Color, bool) {
	comma := strings.IndexByte(inner, ',')
	if comma < 0 {
		r.errorf(span, "%s takes two arguments, `%s(color, %%)` (%q) — e.g. %s(accent1, 40%%)", kind, kind, raw, kind)
		return soml.Color{}, false
	}
	base, bok := r.simpleColor(name, strings.TrimSpace(inner[:comma]), span)
	pctStr := strings.TrimSpace(inner[comma+1:])
	pct, pok := r.variantPercent(kind, pctStr, span, raw)
	if !bok || !pok {
		return soml.Color{}, false
	}
	return soml.VariantColor(base, kind, pct), true
}

// variantPercent는 shade/tint의 백분율 인자를 해석한다 (0~100, `%` 필수).
func (r *resolver) variantPercent(kind soml.ColorModKind, part string, span soml.Span, raw string) (float64, bool) {
	if !strings.HasSuffix(part, "%") {
		r.errorf(span, "the second argument of %s is a percentage (`%%`) (%[3]q in %[2]q) — e.g. %[4]s(accent1, 40%%)", kind, raw, part, kind)
		return 0, false
	}
	pct, err := strconv.ParseFloat(strings.TrimSpace(strings.TrimSuffix(part, "%")), 64)
	if err != nil {
		r.errorf(span, "the percentage of %s is not a number (%[3]q in %[2]q)", kind, raw, part)
		return 0, false
	}
	if pct <= 0 || pct > 100 {
		r.errorf(span, "the percentage of %s must be greater than 0 and at most 100 (%q) — 0%% leaves the base color unchanged", kind, part)
		return 0, false
	}
	return pct, true
}

// alphaFunc는 `alpha(색, 0~1)`를 해석한다 — 밑색 단색에 불투명도를 얹는다.
func (r *resolver) alphaFunc(name, inner string, span soml.Span, raw string) (soml.Color, bool) {
	comma := strings.IndexByte(inner, ',')
	if comma < 0 {
		r.errorf(span, "alpha takes two arguments, `alpha(color, 0–1)` (%q) — e.g. alpha(fg, 0.6)", raw)
		return soml.Color{}, false
	}
	base, bok := r.simpleColor(name, strings.TrimSpace(inner[:comma]), span)
	aStr := strings.TrimSpace(inner[comma+1:])
	a, err := strconv.ParseFloat(aStr, 64)
	if err != nil {
		r.errorf(span, "the alpha opacity is not a number (%[2]q in %[1]q) — it is a real number from 0 to 1", raw, aStr)
		return soml.Color{}, false
	}
	if a < 0 || a > 1 {
		r.errorf(span, "the alpha opacity is between 0 and 1 (%q) — a real number, not a percentage (e.g. 0.6)", aStr)
		return soml.Color{}, false
	}
	if !bok {
		return soml.Color{}, false
	}
	return soml.VariantColor(base, soml.ModAlpha, a), true
}

// align은 문단 정렬을 해석한다. 어휘는 논리 이름 4종이다 (soml.TextAlign의 주석).
func (r *resolver) align(attr soml.Attr) (soml.TextAlign, bool) {
	if !r.hasValue(attr) {
		return "", false
	}
	raw := attr.Value
	for _, align := range soml.TextAligns() {
		if string(align) == raw {
			return align, true
		}
	}
	// 저자는 CSS 습관대로 left/right를 쓴다. 어휘 목록만 던지는 대신 대응하는 논리 이름을
	// 짚어 준다 — 고치는 방법이 진단 안에 있어야 한 번에 끝난다.
	if physical, ok := map[string]soml.TextAlign{"left": soml.AlignStart, "right": soml.AlignEnd}[raw]; ok {
		r.errorf(attr.ValueSpan, "align takes logical names — instead of %q, use %q (the meaning holds in RTL too)", raw, string(physical))
		return "", false
	}
	r.errorf(attr.ValueSpan, "unknown align %q%s (available values: %s)", raw, didYouMean(raw, alignList()), alignList())
	return "", false
}

// hasValue는 속성에 쓸 값이 있는지 본다. 값 없는 외관 속성은 에러다 — `<slide skip>`처럼
// 있음만으로 뜻이 되는 불리언 속성과 달리, 외관 속성의 "값 없음"은 뜻을 정할 수 없다.
func (r *resolver) hasValue(attr soml.Attr) bool {
	switch {
	case !attr.HasValue:
		r.errorf(attr.NameSpan, "the %s attribute needs a value (%s=%q)", attr.Name, attr.Name, attrExample(attr.Name))
		return false
	case attr.Value == "":
		r.errorf(attr.ValueSpan, "the %s attribute is empty (remove it to use the default)", attr.Name)
		return false
	}
	return true
}

// attrExample은 진단에 실을 값 예시다.
func attrExample(name string) string {
	switch name {
	case "role":
		return "body"
	case "variant":
		return "kicker"
	case "size":
		return "24"
	case "color":
		return "accent1"
	case "align":
		return "center"
	case soml.AnchorAttr:
		return "center"
	// `<pre>` 고유 속성. 예시가 어휘의 첫 이름인 이유: 값 없음 진단은 "쓸 수 있는
	// 값"을 나열하지 않으므로(그 목록은 어휘 밖 이름을 받은 자리의 것이다) 예시가 저자가 적을
	// 것을 보여 주는 유일한 자리다.
	case soml.LangAttr:
		return "go"
	// 도형 외관 속성.
	case "type":
		return "rect"
	case "id":
		return "db"
	case "fill":
		return "surface"
	case "border":
		return "1 solid muted"
	case "radius":
		return "md"
	case "shadow":
		return "soft"
	case "opacity":
		return "0.8"
	case "rotate":
		return "45"
	case "flip":
		return "h"
	// 이미지 속성.
	case "src":
		return "assets/photo.png"
	case "fit":
		return "contain"
	case "alt":
		return "chart"
	case "weight":
		return "bold"
	case "href":
		return "https://example.com"
	// 배치 속성. `wrap`은 값 없이도 뜻이 되므로 여기 없다 —
	// 그 판정은 resolver.wrap이 소유한다.
	case "gap":
		return "md"
	case "justify":
		return "between"
	case "pad":
		return "lg"
	case "cols", "rows":
		return "4"
	case "w", "h":
		return "fill"
	case "min-w", "min-h", "max-w", "max-h":
		return "240"
	case "ratio":
		return "16:9"
	case "grow":
		return "1"
	case "shrink":
		return "0"
	case "basis":
		return "240"
	case "span":
		return "2"
	case "self":
		return "center"
	case "order":
		return "-1"
	case "at":
		return "center"
	case "x", "y":
		return "40"
	case soml.StyleAttr:
		return "align-content: space-evenly"
	// 차트 속성. `type`은 도형 행과 이름을 공유하지만 예시가 갈라질 일이 없다 —
	// 차트의 `type`은 값 없음도 자기 문구로 말한다 (chartType).
	case "unit":
		return "%"
	case "max":
		return "100"
	case "series":
		return "accent1"
	case "label":
		return "May"
	case "value":
		return "22"
	}
	return ""
}

// numberSuffix는 수치 뒤에 붙은 꼬리(단위 접미사)다. 수치가 아예 없으면 빈 문자열이다 —
// `size="큼"`·`w="어디"`는 "단위를 떼라"가 아니라 "수치가 아니다"로 진단해야 저자가 무엇을
// 고칠지 안다. 외관(size)과 배치(길이)가 같은 판정을 공유하는 이유는 단위 계약이 하나이기
// 때문이다 — 두 벌이 되면 한쪽만 새 단위를 막게 된다.
func numberSuffix(raw string) string {
	end := 0
	for end < len(raw) {
		c := raw[end]
		if c >= '0' && c <= '9' || c == '.' || (end == 0 && (c == '-' || c == '+')) {
			end++
			continue
		}
		break
	}
	if end == 0 {
		return ""
	}
	return raw[end:]
}

// hexRGB는 `#` 뒤의 16진수를 대문자 6자리로 정규화한다. #RGB는 각 자리를 두 배로 늘린다
// (CSS와 같은 규칙 — 저자가 두 표기에서 다른 색을 보면 안 된다).
func hexRGB(digits string) (string, bool) {
	if len(digits) != 3 && len(digits) != 6 {
		return "", false
	}
	var b strings.Builder
	b.Grow(6)
	for i := 0; i < len(digits); i++ {
		c := digits[i]
		switch {
		case c >= '0' && c <= '9', c >= 'A' && c <= 'F':
		case c >= 'a' && c <= 'f':
			c -= 'a' - 'A'
		default:
			return "", false
		}
		b.WriteByte(c)
		if len(digits) == 3 {
			b.WriteByte(c)
		}
	}
	return b.String(), true
}

// roleList와 alignList는 진단에 실을 어휘 목록이다 — 어휘 정의(soml)와 메시지가 갈라지지
// 않도록 목록을 손으로 쓰지 않는다.
func roleList() string {
	names := make([]string, 0, len(soml.TextRoles()))
	for _, role := range soml.TextRoles() {
		names = append(names, string(role))
	}
	return strings.Join(names, ", ")
}

func alignList() string {
	names := make([]string, 0, len(soml.TextAligns()))
	for _, align := range soml.TextAligns() {
		names = append(names, string(align))
	}
	return strings.Join(names, ", ")
}

// didYouMean은 닫힌 목록(진단에 싣는 쉼표 목록 그대로)에서 raw의 오타 후보를 골라 진단 문구의 꼬리로
// 만든다 — 없으면 빈 문자열이다.
//
// 목록 **문자열**을 받는 이유: 진단은 이미 그 목록을 싣고 있고, 후보를 같은 문자열에서 고르면 "목록에
// 없는 이름을 권하는" 일이 구조적으로 생기지 않는다. 실제 오타는 편집 거리 1–2였다
// (`two-colum`→`two-column` · `heding`→`heading` · `elipse`→`ellipse`).
func didYouMean(raw, list string) string {
	if list == "" {
		return ""
	}
	if guess, ok := soml.Suggest(raw, strings.Split(list, ", ")); ok {
		return " — did you mean " + strconv.Quote(guess) + "?"
	}
	return ""
}

// layoutList는 진단에 실을 슬라이드 레이아웃 어휘 목록이다 — 어휘 정의(soml.SlideLayouts)와
// 메시지가 갈라지지 않도록 목록을 손으로 쓰지 않는다.
func layoutList() string {
	names := make([]string, 0, len(soml.SlideLayouts()))
	for _, layout := range soml.SlideLayouts() {
		names = append(names, string(layout))
	}
	return strings.Join(names, ", ")
}

// slotList는 레이아웃의 허용 슬롯 이름을 어휘 순서로 잇는다 — 없는 슬롯 진단이 허용 어휘를
// 나열하는 근거다 (layoutList와 같은 태도). 슬롯이 없는 레이아웃(canvas)은 빈 문자열이다.
func slotList(layout soml.SlideLayout) string {
	return strings.Join(soml.SlotNames(layout), ", ")
}
