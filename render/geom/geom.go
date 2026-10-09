// Package geom은 지오메트리 패스다 — 웹 렌더러가 만든 HTML을 헤드리스 브라우저에 올려
// 요소별 좌표를 읽어 온다.
//
// 이 패스가 존재하는 이유는 저자가 좌표를 만지지 않게 하려는 것이다 (docs/internals/geometry-pass.md). flex/grid로
// 선언한 배치를 브라우저가 계산하고, 우리는 그 결과 **좌표만** 읽는다. computed-style 역추론은
// 없다 — 외관은 core가 해석한 스키마에서 양 타깃으로 순방향 컴파일된다.
//
// # 단위
//
// 오가는 수치는 전부 **논리 스테이지 px**이다 (ADR 0067 결정 5). 스테이지는 CSS px와 1:1이므로
// 브라우저가 준 값을 스케일하지 않는다 — 다만 그것은 **확인하는 사실**이지 가정이 아니다
// (measure.go의 검증: 슬라이드 rect가 논리 스테이지와 다르면 실패한다). EMU·pt 환산은
// render/pptx의 경계에서만 일어나고, 이 패키지는 그 단위를 알지 못한다.
//
// # 좌표계
//
// 모든 rect는 **슬라이드 상대**다 (슬라이드 좌상단 = 0,0). 문서/뷰포트 좌표를 그대로 넘기면
// 덱 여백·슬라이드 간격·스크롤 위치가 좌표에 섞여 들어가고, PPTX에서는 그것이 "모든 도형이
// 아래로 밀렸다"로 나타난다 (Iteration 0에서 요소 스크린샷이 같은 방식으로 어긋났다 —
// TASKS.md Iteration 0의 리뷰 기록).
//
// # 덱 모델과의 대조
//
// 요소 식별자(`data-balpyo-id`)는 **덱 모델에서 계산한다** (web.ElementID). 렌더 결과를
// 파싱해 이름을 얻으면 두 쪽이 어긋난 순간을 알 수 없다. 그래서 Refs가 덱을 걸어 기대 목록을
// 만들고, 브라우저가 읽어 온 목록과 개수·순서·식별자·태그 이름을 대조한 뒤에만 Geometry가
// 만들어진다. 소비 측(render/pptx)은 Verify로 한 번 더 대조한다 — 좌표가 엉뚱한 요소에 붙은
// PPTX는 열어 보기 전까지 정상으로 보이기 때문에, 이 종류의 실패는 조용해서는 안 된다.
//
// # 덱에 붙이지 않고 별도 표로 넘기는 이유
//
// Geometry는 덱 모델의 필드가 아니라 **곁 표**다. (1) 덱 모델은 공개 계약이고(제3자 도구가
// 임포트한다) 브라우저 측정값은 저작 내용이 아니다. (2) 무엇보다 덱에 실으면 값이 채워지는
// 시점이 타깃 순서에 따라 달라진다 — `--target html,pptx`와 `pptx,html`이 웹 렌더러가 보는
// 덱을 다르게 만든다. 별도 표를 pptx 렌더러에 명시적으로 넘기면 그 의존이 타입에 드러난다.
package geom

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/k2sebeom/balpyo/render/web"
	"github.com/k2sebeom/balpyo/soml"
)

// Rect는 사각형이다 (논리 스테이지 px). 좌표 기준은 쓰는 곳이 정한다 — Box.Rect와 Box.Lines는
// 슬라이드 상대, Session.Rects는 문서 상대다.
type Rect struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	W float64 `json:"w"`
	H float64 `json:"h"`
}

// String은 진단·로그 표기다.
func (r Rect) String() string {
	return fmt.Sprintf("(%.1f,%.1f %.1f×%.1f)", r.X, r.Y, r.W, r.H)
}

// Right는 오른쪽 끝 x다.
func (r Rect) Right() float64 { return r.X + r.W }

// Bottom은 아래쪽 끝 y다.
func (r Rect) Bottom() float64 { return r.Y + r.H }

// valid는 수치가 유한한지 본다. NaN·Inf가 좌표로 흘러가면 EMU 변환에서 정수 쓰레기값이 되고,
// 산출물은 열리지만 도형이 사라진 상태가 된다.
func (r Rect) valid() bool {
	for _, v := range []float64{r.X, r.Y, r.W, r.H} {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return false
		}
	}
	return r.W >= 0 && r.H >= 0
}

// Ref는 좌표가 필요한 덱 요소 하나의 참조다 — 덱 모델만 보고 만든다.
type Ref struct {
	// ID는 `data-balpyo-id` 값이다 (web.ElementID).
	ID string
	// El은 이 요소를 만든 SOML 태그 이름이다 (`data-balpyo-el`과 대조한다).
	El string
	// Slide는 0-based 슬라이드 인덱스다.
	Slide int
	// Source는 슬라이드 파일 경로다 — 대조가 실패했을 때 진단이 가리킬 곳.
	Source string
	// Parent는 부모 요소의 ID다 (슬라이드의 직계 자식은 "").
	Parent string
	// Span은 태그 이름 구간이다 (soml.Element.NameSpan) — 이 요소를 가리키는 진단이 쓴다.
	Span soml.Span
	// Free는 저자가 이 요소의 자리를 **좌표로** 정했는지다 (`at`/`x`/`y`를 썼거나, 그런
	// 요소의 자손이거나, canvas 슬라이드 안이다). 배치 진단이 콘텐츠 박스 이탈을 이 요소에
	// 대해 묻지 않는다 — 좌표를 직접 쓴 것은 여백을 벗어나는 선택까지 포함한다 (ADR 0039).
	Free bool
	// Overlap은 이 요소가 **겹치도록** 놓인 자리에 있는지다 (`<stack>`·canvas의 직계이거나
	// 자기 좌표를 가졌다). 형제 겹침 진단에서 제외된다.
	Overlap bool
}

// Refs는 덱에서 좌표가 필요한 요소를 문서 순서로 모은다.
//
// 순서와 인덱스 규칙은 웹 렌더러의 순회와 같아야 한다: 자식 인덱스는 **문자 데이터 노드까지
// 포함해** 세고(그 노드는 요소가 아니어서 식별자를 갖지 않는다), 부모가 자식보다 앞에 온다.
// 브라우저의 querySelectorAll도 문서 순서를 주므로 두 목록을 자리 하나씩 맞대어 볼 수 있다.
func Refs(deck *soml.Deck) []Ref {
	if deck == nil {
		return nil
	}
	var refs []Ref
	for _, slide := range deck.Slides {
		// canvas 슬라이드의 스테이지는 단일 셀이다 (ADR 0067 결정 3) — 직계 자식이 좌표로 자기
		// 자리를 정하고 서로 겹칠 수 있다. 그 사실은 슬라이드에서 시작하는 문맥이다.
		canvas := slide.Layout == soml.LayoutCanvas
		ctx := placeContext{free: canvas, overlap: canvas}
		for i, node := range slide.Content {
			refs = collect(refs, slide, node, []int{i}, ctx)
		}
	}
	return refs
}

// placeContext는 순회 중의 배치 문맥이다 — 부모가 정하고 자식이 물려받는 사실이다.
// 배치 진단(layout.go)이 "이 이탈·겹침이 저자의 선택인가"를 이 문맥으로 판정한다.
type placeContext struct {
	// parent는 부모 요소의 식별자다 (슬라이드 직계는 "").
	parent string
	// free는 조상 중에 좌표로 놓인 요소가 있는지다 (canvas 슬라이드 포함).
	free bool
	// overlap은 부모가 겹침 컨테이너인지다 (`<stack>`·canvas 스테이지).
	overlap bool
}

// collect는 노드 하나와 그 자손을 모은다.
func collect(refs []Ref, slide *soml.Slide, node soml.Node, path []int, ctx placeContext) []Ref {
	elem, ok := node.(*soml.Element)
	if !ok {
		// 문자 데이터는 자기 박스를 갖지 않는다 (줄 박스로 부모 요소 안에서 측정된다).
		return refs
	}
	if soml.KindOf(elem.Name) == soml.KindInline {
		// 측정 대상은 **박스를 갖는 요소**다 (render/web 패키지 주석의 측정 대상 계약):
		// 블록 종류와 레이아웃 컨테이너·`<spacer>`는 재고, 인라인은 재지 않는다 — 여러 줄에
		// 걸친 인라인의 rect는 뜻이 없고, PPTX에서 인라인은 박스가 아니라 런이다. 웹
		// 렌더러가 같은 카탈로그로 식별자를 붙이지 않으므로 여기서도 건너뛰어야 Refs ↔
		// querySelectorAll 자리 맞대기가 성립한다. 자손까지 건너뛴다 — 인라인의 자식은
		// 문자 데이터와 인라인뿐이고(콘텐츠 모델), 웹 렌더러의 순회(writeInline)와 같은
		// 모양이어야 두 쪽이 어긋난 순간이 대조에서 드러난다.
		//
		// 레이아웃 컨테이너를 재는 것은 ADR 0009("측정 대상은 블록 종류뿐")의 후속 결정이다
		// (Iteration 3): 컨테이너는 진짜 박스이므로 rect에 뜻이 있고, PPTX가 그것을
		// 소비하지 않아도(컨테이너는 도형이 되지 않는다) 픽셀 비교가 갈라질 때 "어느
		// 칼럼부터 밀렸는가"를 이 좌표가 말해 준다 — `<li>`를 재 둔 것과 같은 근거다.
		//
		// **`<shape>`도 잰다** (Iteration 4). 새로운 것은 재는 것과 도형이 되는 것이 처음
		// 일치한다는 사실이다: 도형의 잰 rect는 곧 `xfrm`이므로 소비자가 있다. 반면 도형
		// **안** 텍스트는 재지만 소비되지 않는다 (PPTX에서는 같은 도형의 `txBody` 문단이다) —
		// `<li>`와 같은 처지이고, 그래서 도형 수 등식에 항이 하나 늘었다 (render/web의
		// 측정 대상 계약).
		//
		// **`<image>`도 잰다** (Iteration 5) — 재는 것과 도형이 되는 것이 일치하는 두 번째
		// 종류다: 잰 rect가 `pic`의 `xfrm`이 된다. 웹에서 재는 요소는 `<img>`이고 저자가 준
		// 박스는 식별자를 갖지 않으므로(render/web/image.go), 덱 요소 하나 ↔ 식별자 하나가
		// 유지되고 이 순회는 그림을 **다른 종류와 똑같이** 담으면 된다. 자손은 없다 (void).
		return refs
	}
	id := web.ElementID(slide.Index, path)
	// 좌표를 직접 쓴 요소는 그 자리에 대한 판단을 저자가 이미 했다 — `at`은 셀 안 앵커,
	// `x`/`y`는 축 오프셋이다 (soml.Placement). 확정 값을 읽는 이유는 렌더러와 같다:
	// 속성을 다시 해석하면 값 층을 지나지 않은 판정이 생긴다.
	positioned := isPositioned(elem)
	refs = append(refs, Ref{
		ID:      id,
		El:      elem.Name,
		Slide:   slide.Index,
		Source:  slide.Source,
		Parent:  ctx.parent,
		Span:    elem.NameSpan,
		Free:    ctx.free || positioned,
		Overlap: ctx.overlap || positioned,
	})
	child := placeContext{
		parent: id,
		free:   ctx.free || positioned,
		// 겹침은 **부모가** 정한다: `<stack>`의 자식은 한 칸에 쌓이고, 다른 컨테이너의
		// 자식은 서로 자리를 나눈다. 좌표를 쓴 요소의 **자식**은 그 안에서 다시 흐름이므로
		// 겹침 문맥을 물려받지 않는다.
		overlap: elem.Name == soml.StackElement,
	}
	for i, node := range elem.Children {
		// 경로 규칙은 웹 렌더러가 소유한다 (web.ChildPath) — 식별자를 만드는 규칙과 같은
		// 곳에 있어야 순회 셋의 자리가 어긋날 수 없다.
		refs = collect(refs, slide, node, web.ChildPath(path, i), child)
	}
	return refs
}

// isPositioned는 저자가 이 요소의 자리를 좌표로 정했는지 본다 (`at`·`x`·`y`).
func isPositioned(elem *soml.Element) bool {
	if elem.Appearance == nil || elem.Appearance.Placement == nil {
		return false
	}
	p := elem.Appearance.Placement
	return p.At != "" || p.X != nil || p.Y != nil
}

// Box는 요소 하나의 측정 결과다. 좌표는 **슬라이드 상대 논리 px**다.
type Box struct {
	ID    string `json:"id"`
	El    string `json:"el"`
	Slide int    `json:"slide"`
	// Rect는 요소 박스다. PPTX 도형의 xfrm이 이 rect다 — 텍스트 몸체는 normAutofit이라 넘치면
	// PowerPoint가 글자를 줄여 맞춘다 (ADR 0064).
	Rect Rect `json:"rect"`
	// Lines는 브라우저가 실제로 나눈 줄 박스다 (위에서 아래 순서).
	//
	// 쓰는 곳은 배치 검사의 가로 넘침 하나다 (layout.go의 textEscape — 줄바꿈이 불가능한 글자가
	// 박스 옆으로 흐른 것). 줄 수 · 줄 폭은 두 타깃의 계약이 아니다 (ADR 0064).
	Lines []Rect `json:"lines,omitempty"`
}

// Geometry는 지오메트리 패스의 산출이다 — 식별자에서 슬라이드 상대 좌표로.
//
// 생성 시점에 덱과 대조가 끝나 있다 (newGeometry). 만든 뒤에는 읽기 전용이다.
type Geometry struct {
	// Stage는 측정에 쓰인 논리 스테이지다. 소비 측이 자기 덱과 대조한다 (Verify).
	Stage soml.Stage `json:"stage"`
	// Boxes는 문서 순서의 측정 결과다 (Refs와 같은 순서).
	Boxes []Box `json:"boxes"`
	// Content는 슬라이드마다의 콘텐츠 박스다 (슬라이드 순서, 슬라이드 상대 좌표) — 스테이지의
	// 패딩 안쪽을 **잰** 값이다.
	//
	// 왜 슬라이드마다인가: 여백은 테마가 레이아웃마다 정한다 (`image-full`은 전면 배치라
	// `padding: 0`이다). 덱 하나에 한 값이 아니므로 판정 기준도 슬라이드마다다.
	//
	// 비어 있을 수 있다 (JSON에서 되읽은 값·Synthetic·손으로 만든 시험 입력) — 그때는
	// ContentBox가 기본 여백으로 물러선다.
	Content []Rect `json:"content,omitempty"`
	// Chrome은 덱 크롬의 측정 결과다 (슬라이드 순서 · 슬라이드 안에서는 확정 모델 순서).
	//
	// Boxes가 아니라 **곁 표**인 이유는 Boxes의 계약이다 (ADR 0045): Boxes는 덱 트리의 요소와
	// 1:1이고 그 자리 맞대기가 "잰 HTML이 이 덱의 산출물인가"를 판정한다 (Refs ↔ Boxes,
	// Verify). 크롬은 덱 트리에 없으므로(Slide.Chrome은 Content의 형제다) 끼우면 그 대조가
	// 무너지고, 배치 진단도 크롬에는 다른 질문을 한다 (여백 띠에 사는 것이 정의다).
	// Content가 곁 표인 것과 같은 자리다.
	Chrome []ChromeBox `json:"chrome,omitempty"`

	// index는 식별자 → Boxes 위치다. 생성자가 채우고 그 뒤로는 읽기만 한다 — Box에서
	// 지연 생성하면 지오메트리를 여러 goroutine이 읽는 순간(픽스처 병렬 실행) 경쟁이 된다.
	index map[string]int
}

// ChromeBox는 크롬 하나의 측정 결과다 — 어느 슬라이드의 어느 kind가 어디에 놓였는가.
//
// kind를 함께 나르는 이유: PPTX가 rect의 주인을 알아야 같은 자리에 같은 도형을 놓는다.
// 문서 순서만으로 짝을 지으면 제외 규칙이 한쪽을 뺀 슬라이드에서 번호가 푸터 자리에 놓인다.
type ChromeBox struct {
	// Slide는 0-based 슬라이드 순서다 (soml.Slide.Index).
	Slide int `json:"slide"`
	// Kind는 크롬 종류다 (soml.ChromeKind).
	Kind soml.ChromeKind `json:"kind"`
	// Rect는 슬라이드 상대 좌표의 사각형이다 — 바닥 띠 안이다 (web.chromeCSS).
	Rect Rect `json:"rect"`
}

// ChromeRect는 슬라이드 하나의 kind 하나가 놓인 사각형이다.
//
// 훑는 이유는 Box와 다르다: 크롬은 슬라이드마다 최대 2개이므로 색인을 만들면 자료구조가
// 목록보다 커진다. 없으면 false다 — 소비 측은 그때 도형을 만들지 않고 에러를 낸다
// (조용히 건너뛰면 크롬이 PPTX에서만 사라진다).
func (g *Geometry) ChromeRect(slide int, kind soml.ChromeKind) (Rect, bool) {
	if g == nil {
		return Rect{}, false
	}
	for _, box := range g.Chrome {
		if box.Slide == slide && box.Kind == kind {
			return box.Rect, true
		}
	}
	return Rect{}, false
}

// Box는 식별자로 측정 결과를 찾는다.
func (g *Geometry) Box(id string) (Box, bool) {
	if g == nil {
		return Box{}, false
	}
	if g.index == nil {
		// JSON에서 되읽은 값은 색인이 없다. 여기서 만들지 않고 훑는다 — 읽기 메서드가
		// 상태를 바꾸지 않으면 동시 읽기가 안전하고, 목록은 슬라이드 수준의 크기다.
		for _, box := range g.Boxes {
			if box.ID == id {
				return box, true
			}
		}
		return Box{}, false
	}
	i, ok := g.index[id]
	if !ok {
		return Box{}, false
	}
	return g.Boxes[i], true
}

// ContentBox는 슬라이드 하나의 콘텐츠 박스다 — 잰 값이 있으면 그것, 없으면 기본 여백으로
// 계산한 사각형이다 (ContentBox 함수).
//
// 물러서는 길을 두는 이유: 좌표를 소비하는 쪽은 잰 지오메트리만 받는 것이 아니다 (Synthetic·
// JSON에서 되읽은 값). 그 입력에서 콘텐츠 박스가 0×0이 되면 모든 박스가 여백을 침범한 것으로
// 보이고, 그 경고는 진단 전부를 믿지 못하게 만든다.
func (g *Geometry) ContentBox(slide int) Rect {
	if g == nil {
		return Rect{}
	}
	if slide >= 0 && slide < len(g.Content) {
		if rect := g.Content[slide]; rect.valid() && rect.W > 0 && rect.H > 0 {
			return rect
		}
	}
	return ContentBox(g.Stage)
}

// newIndex는 식별자 색인이다.
func newIndex(boxes []Box) map[string]int {
	index := make(map[string]int, len(boxes))
	for i, box := range boxes {
		index[box.ID] = i
	}
	return index
}

// Verify는 이 지오메트리가 주어진 덱을 잰 것인지 대조한다.
//
// 측정 시점에 이미 대조했지만 소비 시점에 한 번 더 본다: 파이프라인이 덱을 다시 읽거나
// (dev 서버의 핫 리로드) 타깃 순서가 바뀌어도, 잰 덱과 조립하는 덱이 다르면 좌표는 전부
// 엉뚱한 요소로 간다. 그 실패는 산출물의 모양으로만 드러나므로 여기서 멈춘다.
func (g *Geometry) Verify(deck *soml.Deck) error {
	if g == nil {
		return fmt.Errorf("지오메트리가 없다")
	}
	if deck == nil {
		return fmt.Errorf("덱이 nil이다")
	}
	if g.Stage != deck.Stage {
		return fmt.Errorf("지오메트리는 논리 스테이지 %dx%d에서 측정되었다 — 덱의 스테이지는 %dx%d다",
			g.Stage.Width, g.Stage.Height, deck.Stage.Width, deck.Stage.Height)
	}
	refs := Refs(deck)
	if len(refs) != len(g.Boxes) {
		return fmt.Errorf("측정된 요소 %d개, 덱의 요소 %d개 — 잰 덱과 다른 덱이다",
			len(g.Boxes), len(refs))
	}
	for i, ref := range refs {
		box := g.Boxes[i]
		if box.ID != ref.ID || box.El != ref.El {
			return fmt.Errorf("%s: %d번째 요소가 어긋난다 — 측정 %s <%s>, 덱 %s <%s>",
				ref.Source, i, box.ID, box.El, ref.ID, ref.El)
		}
	}
	return nil
}

// newGeometry는 브라우저가 읽어 온 값을 덱과 대조해 Geometry로 만든다.
//
// 여기가 "두 쪽이 어긋난 순간"을 잡는 유일한 자리다. 순서를 자리 하나씩 맞대어 보는 이유:
// 식별자를 집합으로만 비교하면 렌더러가 요소 순서를 바꾼 변경(같은 식별자, 다른 위치)을
// 놓치고, 그때 좌표는 조용히 뒤바뀐다.
func newGeometry(deck *soml.Deck, m measurement) (*Geometry, error) {
	if deck == nil {
		return nil, fmt.Errorf("덱이 nil이다")
	}
	// devicePixelRatio가 1이 아니면 브라우저가 준 CSS px와 스크린샷 픽셀이 갈린다 — 좌표는
	// 맞는데 리뷰의 픽셀 비교만 틀리는, 원인을 찾기 어려운 상태가 된다.
	if m.DPR != 1 {
		return nil, fmt.Errorf("브라우저 devicePixelRatio가 %s다 (1이어야 한다) — 논리 px와 실제 픽셀이 갈린다",
			formatPx(m.DPR))
	}
	if len(m.Slides) != len(deck.Slides) {
		return nil, fmt.Errorf("HTML의 슬라이드 수 %d개, 덱의 슬라이드 수 %d개 — 잰 HTML이 이 덱의 산출물이 아니다",
			len(m.Slides), len(deck.Slides))
	}

	refs := Refs(deck)
	boxes := make([]Box, 0, len(refs))
	content := make([]Rect, len(m.Slides))
	var chrome []ChromeBox
	next := 0
	for i, sm := range m.Slides {
		slide := deck.Slides[i]
		if sm.Index != slide.Index {
			return nil, fmt.Errorf("%s: %d번째 슬라이드의 %s가 %d다 (want %d)",
				slide.Source, i, web.SlideIndexAttr, sm.Index, slide.Index)
		}
		// 스테이지가 CSS px와 1:1이라는 전제를 여기서 확인한다. 페이지가 스케일되면(zoom,
		// transform, 뷰포트 맞춤) 모든 좌표가 같은 비율로 틀린 채 산출물까지 가고, 산출물은
		// "열리지만 조금씩 어긋난" 상태가 되어 원인을 찾는 데 가장 오래 걸린다.
		if !sameSize(sm.W, float64(deck.Stage.Width)) || !sameSize(sm.H, float64(deck.Stage.Height)) {
			return nil, fmt.Errorf("%s: 브라우저가 잰 슬라이드 크기가 %s×%s다 — 논리 스테이지 %dx%d와 달라 페이지가 스케일된 상태다",
				slide.Source, formatPx(sm.W), formatPx(sm.H), deck.Stage.Width, deck.Stage.Height)
		}

		// 콘텐츠 박스는 판정에만 쓰이므로(배치 진단) 어긋나도 빌드를 멈추지 않는다 — 유효하지
		// 않으면 담지 않고 판정 쪽이 기본 여백으로 물러선다 (Geometry.ContentBox). 좌표를
		// 소비하는 쪽은 이 값을 읽지 않는다.
		if sm.Content.valid() {
			content[i] = sm.Content
		}

		// 크롬은 확정 모델과 **자리 하나씩** 맞댄다 (ADR 0045) — 수가 맞아도 kind가 뒤바뀌면
		// 번호가 푸터 자리에 놓인 PPTX가 나오고, 그것은 열어 보기 전까지 정상으로 보인다.
		if len(sm.Chrome) != len(slide.Chrome) {
			return nil, fmt.Errorf("%s: HTML의 크롬 %d개, 덱의 크롬 %d개 — 잰 HTML이 이 덱의 산출물이 아니다",
				slide.Source, len(sm.Chrome), len(slide.Chrome))
		}
		for j, cm := range sm.Chrome {
			item := slide.Chrome[j]
			if cm.Kind != string(item.Kind) {
				return nil, fmt.Errorf("%s: %d번째 크롬이 어긋난다 — HTML %s, 덱 %s",
					slide.Source, j, cm.Kind, item.Kind)
			}
			if !cm.Rect.valid() {
				return nil, fmt.Errorf("%s: 크롬 %s의 rect %s가 유효하지 않다", slide.Source, cm.Kind, cm.Rect)
			}
			chrome = append(chrome, ChromeBox{Slide: slide.Index, Kind: item.Kind, Rect: cm.Rect})
		}

		for _, bm := range sm.Boxes {
			if next >= len(refs) {
				return nil, fmt.Errorf("%s: HTML에 덱보다 많은 요소가 있다 — %s <%s>는 덱에 없다",
					slide.Source, bm.ID, bm.El)
			}
			ref := refs[next]
			if bm.ID != ref.ID || bm.El != ref.El {
				return nil, fmt.Errorf("%s: %d번째 요소가 어긋난다 — HTML %s <%s>, 덱 %s <%s>",
					slide.Source, next, bm.ID, bm.El, ref.ID, ref.El)
			}
			if !bm.Rect.valid() {
				return nil, fmt.Errorf("%s: %s의 rect %s가 유효하지 않다", slide.Source, bm.ID, bm.Rect)
			}
			for j, line := range bm.Lines {
				if !line.valid() {
					return nil, fmt.Errorf("%s: %s의 %d번째 줄 박스 %s가 유효하지 않다",
						slide.Source, bm.ID, j, line)
				}
			}
			boxes = append(boxes, Box{
				ID: bm.ID, El: bm.El, Slide: slide.Index, Rect: bm.Rect, Lines: bm.Lines,
			})
			next++
		}
	}
	if next != len(refs) {
		missing := make([]string, 0, len(refs)-next)
		for _, ref := range refs[next:] {
			missing = append(missing, ref.ID)
		}
		return nil, fmt.Errorf("HTML에 없는 덱 요소가 있다: %s — 웹 렌더러가 낸 요소와 덱이 어긋났다",
			strings.Join(missing, ", "))
	}

	g := &Geometry{Stage: deck.Stage, Boxes: boxes, Content: content, Chrome: chrome, index: newIndex(boxes)}
	// 같은 대조를 소비 측 계약(Verify)으로 한 번 더 돈다 — 두 경로가 같은 판정을 하는지
	// 여기서 확인해 두면, 나중에 한쪽만 느슨해지는 일을 막는다.
	if err := g.Verify(deck); err != nil {
		return nil, err
	}
	return g, nil
}

// sizeEpsilon은 크기 비교의 허용 오차다. 브라우저는 레이아웃을 1/64px 단위로 계산하므로
// `width: 1280px`은 정확히 1280으로 돌아온다 — 이 폭을 넘는 차이는 반올림이 아니라 스케일이다.
const sizeEpsilon = 0.01

// sameSize는 두 크기가 같은지 본다.
func sameSize(got, want float64) bool { return math.Abs(got-want) < sizeEpsilon }

// formatPx는 수치를 자릿수를 늘리지 않고 쓴다 (진단 메시지용).
func formatPx(v float64) string { return strconv.FormatFloat(v, 'f', -1, 64) }
