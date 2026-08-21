// Package geom은 지오메트리 패스다 — 웹 렌더러가 만든 HTML을 헤드리스 브라우저에 올려
// 요소별 좌표를 읽어 온다.
//
// 이 패스가 존재하는 이유는 저자가 좌표를 만지지 않게 하려는 것이다 (PRD §9.3). flex/grid로
// 선언한 배치를 브라우저가 계산하고, 우리는 그 결과 **좌표만** 읽는다. computed-style 역추론은
// 없다 — 외관은 core가 해석한 스키마에서 양 타깃으로 순방향 컴파일된다.
//
// # 단위
//
// 오가는 수치는 전부 **논리 스테이지 px**이다 (PRD §13-5). 스테이지는 CSS px와 1:1이므로
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
		for i, node := range slide.Content {
			refs = collect(refs, slide, node, []int{i})
		}
	}
	return refs
}

// collect는 노드 하나와 그 자손을 모은다.
func collect(refs []Ref, slide *soml.Slide, node soml.Node, path []int) []Ref {
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
		return refs
	}
	refs = append(refs, Ref{
		ID:     web.ElementID(slide.Index, path),
		El:     elem.Name,
		Slide:  slide.Index,
		Source: slide.Source,
	})
	for i, child := range elem.Children {
		// 경로 규칙은 웹 렌더러가 소유한다 (web.ChildPath) — 식별자를 만드는 규칙과 같은
		// 곳에 있어야 순회 셋의 자리가 어긋날 수 없다.
		refs = collect(refs, slide, child, web.ChildPath(path, i))
	}
	return refs
}

// Box는 요소 하나의 측정 결과다. 좌표는 **슬라이드 상대 논리 px**다.
type Box struct {
	ID    string `json:"id"`
	El    string `json:"el"`
	Slide int    `json:"slide"`
	// Rect는 요소 박스다. PPTX 텍스트 박스는 이 크기로 고정되고 autofit은 꺼진다 (PRD §9.3).
	Rect Rect `json:"rect"`
	// Lines는 브라우저가 실제로 나눈 줄 박스다 (위에서 아래 순서).
	//
	// x/y/w/h 말고 이것까지 재는 이유는 Iteration 1의 리뷰 포인트 때문이다: 박스를 고정해
	// 내보낸 뒤 LibreOffice/PowerPoint가 **같은 자리에서 줄을 나누는가**를 수치로 판정해야
	// 한다 (PRD §13-1의 임계값). 줄 수는 판정의 좌변이고, Slack(줄 끝에 남은 여유 폭)은
	// 판정의 우변이다 — 다른 엔진의 글자 폭이 Slack을 넘어 넓어지는 순간 줄이 하나 늘어난다.
	// 그 이상(글리프 단위 메트릭, 폰트 테이블)은 재지 않는다: 임계값을 정하는 데 쓰이지 않는
	// 데이터는 골든만 키운다.
	Lines []Rect `json:"lines,omitempty"`
}

// LineCount는 브라우저가 나눈 줄 수다.
func (b Box) LineCount() int { return len(b.Lines) }

// MaxLineWidth는 가장 긴 줄의 폭이다.
func (b Box) MaxLineWidth() float64 {
	var max float64
	for _, line := range b.Lines {
		if line.W > max {
			max = line.W
		}
	}
	return max
}

// Slack은 가장 긴 줄과 박스 폭 사이에 남은 여유 폭이다 (px).
//
// 줄바꿈 허용 오차의 재료다: 다른 렌더 엔진이 같은 텍스트를 이보다 더 넓게 그리면 그 요소는
// 줄이 하나 늘어난다. 요소별 Slack/MaxLineWidth의 최소값이 곧 그 덱이 견디는 폭 오차 비율이다.
func (b Box) Slack() float64 {
	slack := b.Rect.W - b.MaxLineWidth()
	if slack < 0 {
		return 0
	}
	return slack
}

// LineHeight는 줄 박스 높이다 — **글자 상자**(ascent+descent)이고 CSS 줄높이가 아니다.
// 브라우저가 Range에 주는 사각형이 글자 상자이기 때문이다 (24px 본문에서 36px 줄높이의 줄이
// 29px로 돌아온다). 줄마다 다를 수 있으므로 첫 줄을 쓴다 — 이번 iteration의 어휘에서는 요소
// 하나가 한 가지 폰트 크기를 갖는다.
func (b Box) LineHeight() float64 {
	if len(b.Lines) == 0 {
		return 0
	}
	return b.Lines[0].H
}

// LineAdvance는 줄 사이의 세로 간격이다 (줄이 둘 미만이면 0).
//
// 줄바꿈 판정의 세로 절반이다. Slack이 "다른 엔진의 글자가 얼마나 넓어져도 줄이 안 늘어나는가"
// 라면, 이 값은 "줄이 하나 늘었을 때 고정 박스를 얼마나 넘치는가"의 단위다 — 박스 높이는
// 대체로 줄 수 × 이 값이므로, 줄이 하나 늘면 정확히 이만큼 넘친다. 첫 두 줄로 재는 이유는
// 평균이 값이 흔들리는 경우(요소 안에 크기가 다른 글자)를 감춰서다.
func (b Box) LineAdvance() float64 {
	if len(b.Lines) < 2 {
		return 0
	}
	return b.Lines[1].Y - b.Lines[0].Y
}

// TextRect는 줄 박스 전체를 감싸는 사각형이다 — 박스 안에서 글자가 실제로 차지한 영역.
func (b Box) TextRect() Rect {
	if len(b.Lines) == 0 {
		return Rect{X: b.Rect.X, Y: b.Rect.Y}
	}
	left, top := math.MaxFloat64, math.MaxFloat64
	right, bottom := -math.MaxFloat64, -math.MaxFloat64
	for _, line := range b.Lines {
		left = math.Min(left, line.X)
		top = math.Min(top, line.Y)
		right = math.Max(right, line.Right())
		bottom = math.Max(bottom, line.Bottom())
	}
	return Rect{X: left, Y: top, W: right - left, H: bottom - top}
}

// Geometry는 지오메트리 패스의 산출이다 — 식별자에서 슬라이드 상대 좌표로.
//
// 생성 시점에 덱과 대조가 끝나 있다 (newGeometry). 만든 뒤에는 읽기 전용이다.
type Geometry struct {
	// Stage는 측정에 쓰인 논리 스테이지다. 소비 측이 자기 덱과 대조한다 (Verify).
	Stage soml.Stage `json:"stage"`
	// Boxes는 문서 순서의 측정 결과다 (Refs와 같은 순서).
	Boxes []Box `json:"boxes"`

	// index는 식별자 → Boxes 위치다. 생성자가 채우고 그 뒤로는 읽기만 한다 — Box에서
	// 지연 생성하면 지오메트리를 여러 goroutine이 읽는 순간(픽스처 병렬 실행) 경쟁이 된다.
	index map[string]int
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

	g := &Geometry{Stage: deck.Stage, Boxes: boxes, index: newIndex(boxes)}
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
