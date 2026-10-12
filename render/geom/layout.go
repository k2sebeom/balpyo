package geom

import (
	"fmt"
	"math"
	"strings"

	"github.com/k2sebeom/balpyo/render/web"
	"github.com/k2sebeom/balpyo/soml"
)

// 이 파일은 **배치 진단**이다 — 잰 좌표에서 "저자가 열어 보지 않으면 모를 사고"를
// 찾아 경고로 말한다. 지오메트리 패스는 이미 모든 박스와 줄을 재고 있으므로 필요한 숫자는 전부
// 여기 있다: 이 파일이 하는 일은 새 측정이 아니라 **판정**이다.
//
// 판정은 한 가지 문장의 반복이다 — **박스는 자기가 속한 박스 안에 있어야 한다**. 무엇을 "속한
// 박스"로 보는지에 따라 셋이 되고, 여기에 형제 관계를 보는 하나가 붙는다:
//
//  1. 스테이지 이탈 — 박스가 논리 스테이지(1280×720) 밖으로 나갔다. 웹은 잘라 내고
//     (`.balpyo-slide { overflow: hidden }`) PPTX는 슬라이드 밖에 둔다. 좌표를 직접 쓴
//     요소도 예외가 아니다: 화면 밖은 어떤 저작 의도로도 설명되지 않는다.
//  2. 부모 박스 이탈 — 박스가 부모 박스를 넘쳤다. 저자가 크기를 정한 박스(`<shape h="140">`)에
//     내용이 담기지 않은 사고가 이 모양으로 나타난다.
//  3. 콘텐츠 박스 이탈 — 슬라이드 직계 박스가 스테이지 여백(64px)을 침범했다.
//  4. 형제 겹침 — 같은 부모의 두 박스가 서로 겹쳤다.
//
// 2·3은 좌표를 직접 쓴 요소에게 묻지 않고(Ref.Free), 4는 겹치도록 놓인 자리를 묻지 않는다
// (Ref.Overlap — `<stack>`·canvas·`at`/`x`/`y`). 저자가 좌표를 쓴 것은 "여백과 부모 변을
// 넘어서는 선택"까지 포함한다. 2는 표의 셀에게도 묻지 않는다 — 셀의 박스는 저자가 정한 크기가
// 아니라 브라우저의 표 알고리즘이 짠 격자이고, `rowspan`이 든 셀은 자기 행보다 높은 것이 정상이다.
//
// # `scrollHeight`를 쓰지 않는 이유
//
// "내용이 박스를 넘쳤는가"를 브라우저에 직접 묻는 값은 `scrollHeight - clientHeight`다. 그
// 값은 **쓸 수 없다**: 글자 상자(ascent+descent ≈ 1.27em)가 CSS 줄 박스(line-height 1.2)보다
// 크기 때문에, 내용이 넉넉히 들어간 제목도 세로로 몇 px을 넘긴 것으로 나온다 (실측: 56px 제목
// 4px, 28px 제목 1px). 그 몇 px은 조상 박스로 전파되므로 고정 높이 도형까지 함께 경고를 받는다.
// 모든 제목이 경고를 받는 진단은 읽히지 않고, 읽히지 않는 경고는 없는 것과 같다. 그래서 세로
// 판정은 **CSS 박스끼리**(자식 rect ↔ 부모 rect) 본다 — 글자 메트릭이 섞이지 않는 유일한 비교다.
//
// 대가는 과소 보고다. 부모의 패딩 안에서 끝난 넘침은 말하지 않는다 (`pad="lg"` 도형에서
// 24px까지). 거짓 양성과 과소 보고 중에 과소 보고를 고른 이유는 진단의 신뢰다: 경고가 나오면
// 그것은 언제나 진짜 사고다.
//
// 판정을 여기 두는 이유는 값이 여기 있기 때문이다. core는 좌표를 모르고(브라우저를 띄우지
// 않는다), 렌더러는 자기 산출물만 본다. 진단 타입(soml.Diagnostics)은 파서·값 층과 공유하므로
// 저자가 보는 경고의 모양이 한 가지다.

// overflowTolerance는 이탈·겹침 판정의 허용 오차다 (px).
//
// 1px인 이유: 1px 미만의 이탈은 저자가 고칠 수 있는 사고가 아니다 — 반올림 경계에 걸린 박스마다
// 경고가 나면 경고를 읽지 않게 되고, 그것이 진단을 잃는 방식이다. 이 값 덕분에 메시지의 px는
// 항상 1 이상이다 ("0px 넘친다"는 문장이 나오지 않는다).
const overflowTolerance = 1.0

// ContentBox는 스테이지의 콘텐츠 박스다 — 스테이지에서 여백을 뺀 사각형(16:9에서 1152×592).
//
// 여백은 웹 렌더러가 소유한다 (web.StagePadding) — 스테이지 CSS를 쓰는 곳과 그것을 판정에
// 쓰는 곳이 같은 값을 읽어야 "여백을 침범했다"가 실제 여백을 가리킨다.
func ContentBox(stage soml.Stage) Rect {
	pad := float64(web.StagePadding)
	return Rect{X: pad, Y: pad, W: float64(stage.Width) - 2*pad, H: float64(stage.Height) - 2*pad}
}

// stageRect는 논리 스테이지 사각형이다 (슬라이드 상대 좌표이므로 원점은 0,0).
func stageRect(stage soml.Stage) Rect {
	return Rect{W: float64(stage.Width), H: float64(stage.Height)}
}

// Diagnose는 잰 좌표에서 배치 사고를 찾는다 (위 목록). 심각도는 전부 경고다 — 에러로
// 승격하는 것은 `--strict`의 몫이고, 그 판단은 파이프라인이 한 곳에서 한다.
//
// 요소 하나가 받는 이탈 진단은 하나다. 부모 → 스테이지 → 콘텐츠 박스 순으로 **원인에 가까운
// 사고부터** 보고 하나를 말한다: 같은 사고를 세 이름으로 부르면 저자는 세 줄을 읽고 한 곳을
// 고친다. 부모가 먼저인 이유는 스테이지 안의 부모를 넘친 박스가 스테이지까지 나갔다면 그것은
// 부모가 작은 결과라는 것이다 — 스테이지 이탈을 먼저 말하면 저자는 자식을 옮기고, 부모 경고를
// 그다음 빌드에서야 본다. 스테이지 이탈은 같은 문장에 덧붙인다. 같은 이유로 **사고 하나도 경고 하나다** — 이탈 경고를 받은 요소 안에서 함께 벗어난
// 자손은 따로 말하지 않고 그 경고에 세어 붙인다 (escapedAncestor). 글줄 넘침과 형제 겹침은
// 다른 사실이므로 따로 붙는다.
//
// 덱과 지오메트리가 어긋나면(잰 덱이 다른 덱이다) 아무 말도 하지 않는다 — 그 어긋남은 이미
// 빌드를 멈추는 에러이고(newGeometry·Verify), 엉뚱한 요소를 가리키는 경고를 더하면 진짜 원인이
// 묻힌다.
func Diagnose(deck *soml.Deck, g *Geometry) soml.Diagnostics {
	if deck == nil || g == nil || g.Verify(deck) != nil {
		return nil
	}
	refs := Refs(deck)
	stage := stageRect(deck.Stage)

	// 글꼴 경고가 먼저다: 대체 글꼴로 잰 덱이면 아래의 배치 경고도 그 글꼴의 숫자다.
	diags := FontDiagnostics(deck.Theme, g.MissingFonts)
	// 식별자 → refs 색인이다. 부모는 문서 순서에서 자식보다 앞에 오므로(Refs) 순회하며 채우면
	// 자식이 볼 때 부모가 이미 들어 있다.
	index := map[string]int{}
	// 같은 부모의 **이미 본** 형제들이다. 키에 슬라이드가 들어가는 이유: 슬라이드 직계 자식의
	// 부모 식별자는 전부 ""이고 좌표는 슬라이드 상대이므로, 슬라이드를 빼면 모든 장의 첫
	// 요소가 서로 겹친 것으로 보인다.
	siblings := map[sibGroup][]int{}
	// 이탈 경고를 받은 요소 → 그 경고의 자리(diags 안)다. 자손의 이탈은 가장 가까운 그런 조상의
	// 경고에 세어 붙인다 (foldInto) — 스테이지를 넘친 목록 하나가 항목마다 경고를 내지 않게.
	escaped := map[int]int{}
	folds := map[int]*escapeFold{}
	charts := newChartDiagnosis()
	for i, ref := range refs {
		index[ref.ID] = i
		box := g.Boxes[i]
		parent, hasParent := index[ref.Parent]
		charts.visit(refs, index, i, box)

		var warning *soml.Diagnostic
		// 이탈의 방향과 양이다 — 차트 틀의 넘침은 문장을 따로 쓰므로(render/geom/chart.go) 숫자만 넘긴다.
		warnDir, warnOver := "", 0.0
		// 부모 판정을 묻는 자리다 — 좌표를 직접 쓴 요소와 표의 셀은 부모 변을 넘는 것이 정상이다.
		//
		// 좌표를 직접 쓴 요소(와 그 자손)는 부모 변과 여백을 넘어서는 것이 저작 수단이다. 표의 셀은
		// 부모 `<tr>`를 **합법적으로** 넘친다: `rowspan`이 든 셀의 박스는 여러 행에 걸치므로 자기
		// 행보다 높다 (격자를 짜는 것은 브라우저의 표 알고리즘이다). 격자의 어느 박스도 저자가 크기를
		// 정한 것이 아니므로, "고정 크기 박스가 내용을 담지 못했다"는 이 판정의 사고가 셀에서는
		// 성립하지 않는다. 표가 담기지 않는 사고를 놓치지는 않는다 — 그것은 프레임(`<table>`)이 부모나
		// 콘텐츠 박스를 넘치는 모양으로 나타나고, 그 판정은 그대로 남는다. 행(`<tr>`)도 그대로 본다:
		// 행이 표를 넘치는 것은 격자가 설명하지 않는 일이다.
		askParent := hasParent && !ref.Free && soml.KindOf(ref.El) != soml.KindTableCell
		stageDir, stageOver := escape(box.Rect, stage)
		parentDir, parentOver := "", 0.0
		if askParent {
			parentDir, parentOver = escape(box.Rect, g.Boxes[parent].Rect)
		}
		switch {
		case parentOver > overflowTolerance:
			// 부모를 넘친 것이 먼저다: 부모가 스테이지 안에 있으면 스테이지 이탈은 너무 작은 부모의
			// 결과이고, 고칠 자리는 부모의 크기다. (부모까지 스테이지를 나갔다면 부모가 먼저 경고를
			// 받고 이 요소는 그 경고에 접힌다.)
			also := ""
			if stageOver > overflowTolerance {
				also = fmt.Sprintf(" and leaves the stage %s by %s", stageDir, px(stageOver))
			}
			d := soml.Warningf(ref.Source, ref.Span,
				"<%s> overflows its parent <%s> %s by %s%s — the parent box is too small for its content",
				ref.Tag, refs[parent].Tag, parentDir, px(parentOver), also)
			warning = &d
			warnDir, warnOver = parentDir, parentOver
		case stageOver > overflowTolerance:
			d := soml.Warningf(ref.Source, ref.Span,
				"<%s> leaves the stage %s by %s — off-stage content is clipped on screen and placed off the slide in PPTX",
				ref.Tag, stageDir, px(stageOver))
			warning = &d
			warnDir, warnOver = stageDir, stageOver
		case hasParent || ref.Free:
			// 부모 안에 있거나 좌표로 놓였다 — 콘텐츠 박스는 슬라이드 직계의 흐름 요소에게만 묻는다.
		default:
			// 콘텐츠 박스는 **이 슬라이드에서 잰** 것이다 — 여백은 테마가 레이아웃마다 정하므로
			// (`image-full`은 전면 배치라 여백이 없다) 상수로 판정하면 그 레이아웃의 전면
			// 그림마다 저자가 고칠 수 없는 경고가 나온다 (Geometry.Content).
			content := g.ContentBox(ref.Slide)
			if dir, over := escape(box.Rect, content); over > overflowTolerance {
				d := soml.Warningf(ref.Source, ref.Span,
					"<%s> overflows the content box %s by %s — this layout's content box is %s (a 64px stage margin unless the theme's layout CSS changes it)",
					ref.Tag, dir, px(over), content)
				warning = &d
			}
		}
		if warning != nil && ref.Part != "" {
			// 차트의 틀 노드다. 차트 뿌리나 그 바깥이 이미 벗어났다면 틀은 그 결과이고 저자가 쓴 요소가
			// 아니므로 세지도 않는다. 아니면 차트 하나의 경고에 모은다 — 틀의 자손(범례 항목 · 라벨)은
			// 그 경고에 접힌다.
			ancestor, folded := escapedAncestor(refs, index, escaped, ref)
			if !folded || charts.isEscape(ancestor) {
				if at, ok := charts.frame(refs, i, warnDir, warnOver, &diags); ok {
					if _, ok := folds[at]; !ok {
						folds[at] = &escapeFold{}
					}
					escaped[i] = at
				}
			}
		} else if warning != nil {
			if ancestor, ok := escapedAncestor(refs, index, escaped, ref); ok {
				folds[ancestor].add(ref.Tag)
			} else {
				escaped[i] = len(diags)
				folds[len(diags)] = &escapeFold{}
				diags = append(diags, *warning)
			}
		}

		if dir, over := textEscape(box); over > overflowTolerance {
			diags = append(diags, soml.Warningf(ref.Source, ref.Span,
				"<%s> text overflows its box %s by %s — text that cannot wrap (a long word, URL or `pre`)",
				ref.Tag, dir, px(over)))
		}

		group := sibGroup{slide: ref.Slide, parent: ref.Parent}
		if !ref.Overlap {
			for _, j := range siblings[group] {
				other := refs[j]
				if other.Overlap {
					continue
				}
				if w, h, ok := overlap(g.Boxes[j].Rect, box.Rect); ok {
					diags = append(diags, soml.Warningf(ref.Source, ref.Span,
						"<%s> overlaps its sibling <%s> by %s×%s — if the overlap is intended, use <%s> or x/y",
						ref.Tag, other.Tag, px(w), px(h), soml.StackElement))
				}
			}
		}
		siblings[group] = append(siblings[group], i)
	}
	diags = charts.finish(refs, diags)
	for at, fold := range folds {
		diags[at].Message += fold.suffix()
	}
	return append(diags, diagnoseChrome(deck, g)...)
}

// escapedAncestor는 ref의 조상 가운데 이미 이탈 경고를 받은 가장 가까운 것의 경고 자리다.
//
// 자손의 이탈을 그 경고에 접는 이유: 조상이 스테이지 · 부모 · 콘텐츠 박스를 벗어나면 그 안의
// 박스가 함께 나가는 것은 같은 사고의 결과다. 저자는 조상 하나를 고치고(목록을 줄이거나 박스를
// 키운다), 자손 경고는 그 사이에 저절로 사라진다. 판정의 종류가 달라도(목록은 스테이지, 항목은
// 부모) 접는다 — 고칠 자리가 하나라는 것이 기준이다.
func escapedAncestor(refs []Ref, index map[string]int, escaped map[int]int, ref Ref) (int, bool) {
	for id := ref.Parent; id != ""; {
		i, ok := index[id]
		if !ok {
			return 0, false
		}
		if at, ok := escaped[i]; ok {
			return at, true
		}
		id = refs[i].Parent
	}
	return 0, false
}

// escapeFold는 한 이탈 경고에 접힌 자손들이다 — 태그별 개수 (처음 나온 순서).
type escapeFold struct {
	tags   []string
	counts map[string]int
}

func (f *escapeFold) add(tag string) {
	if f.counts == nil {
		f.counts = map[string]int{}
	}
	if f.counts[tag] == 0 {
		f.tags = append(f.tags, tag)
	}
	f.counts[tag]++
}

// suffix는 접힌 자손을 경고 끝에 붙일 구절이다 (없으면 빈 문자열).
func (f *escapeFold) suffix() string {
	if len(f.tags) == 0 {
		return ""
	}
	parts := make([]string, 0, len(f.tags))
	for _, tag := range f.tags {
		parts = append(parts, fmt.Sprintf("%d <%s>", f.counts[tag], tag))
	}
	return " (" + strings.Join(parts, " · ") + " inside escape too)"
}

// diagnoseChrome은 덱 크롬의 배치 사고를 찾는다. 질문은 둘뿐이다:
//
//  1. 슬라이드 이탈 — 크롬이 논리 스테이지 밖으로 나갔다 (긴 푸터 문자열의 전형).
//  2. 크롬끼리 겹침 — 푸터가 길어 번호를 덮었다. 두 글자가 겹쳐 둘 다 읽히지 않는다.
//
// 요소의 판정(Diagnose의 목록) 중 **콘텐츠 박스 이탈은 묻지 않는다**: 크롬은 스테이지 여백
// 띠에 사는 것이 정의이므로(web.chromeCSS) 그 판정을 적용하면 크롬을 켠 모든 덱이 슬라이드마다
// 고칠 수 없는 경고를 받고, 그것이 진단 전부를 잃는 방식이다. 부모 이탈도 없다 —
// 크롬의 부모는 슬라이드이므로 1과 같은 질문이다. 형제 겹침은 **크롬끼리만** 본다: 크롬과
// 본문은 띠와 콘텐츠 박스로 갈라져 있고, 본문이 띠를 침범한 사고는 그 요소가 이미 콘텐츠 박스
// 이탈로 듣는다.
//
// 진단이 가리키는 파일은 슬라이드다 — 사고가 난 장을 알아야 고칠 수 있다. 무엇을 고치는지는
// 문장이 말한다 (`balpyo.json`의 문자열이거나 테마의 자리다): 크롬은 저자가 그 슬라이드에 쓴
// 것이 아니므로 구간(Span)은 없다.
func diagnoseChrome(deck *soml.Deck, g *Geometry) soml.Diagnostics {
	stage := stageRect(deck.Stage)
	var diags soml.Diagnostics
	for _, slide := range deck.Slides {
		var seen []ChromeBox
		for _, item := range slide.Chrome {
			rect, ok := g.ChromeRect(slide.Index, item.Kind)
			if !ok {
				// 잰 지오메트리에는 반드시 있다 (newGeometry가 자리를 맞댄다). 손으로 만든
				// 입력에는 없을 수 있으므로 조용히 넘긴다 — 여기서 경고를 내면 그 경고가
				// 가리키는 것은 덱이 아니라 시험 입력이다.
				continue
			}
			if dir, over := escape(rect, stage); over > overflowTolerance {
				diags = append(diags, soml.Warningf(slide.Source, soml.Span{},
					"deck chrome (%s) leaves the slide %s by %s — the text is too long for the bottom band (shorten the footer in %s, or have the theme use a smaller role)",
					item.Kind, dir, px(over), configFileName))
			}
			for _, other := range seen {
				if w, h, ok := overlap(other.Rect, rect); ok {
					diags = append(diags, soml.Warningf(slide.Source, soml.Span{},
						"deck chrome (%s) overlaps chrome (%s) by %s×%s — neither is readable (shorten the footer in %s)",
						item.Kind, other.Kind, px(w), px(h), configFileName))
				}
			}
			seen = append(seen, ChromeBox{Slide: slide.Index, Kind: item.Kind, Rect: rect})
		}
	}
	return diags
}

// configFileName은 진단 문구에서 가리키는 설정 파일 이름이다 (core.ConfigFile).
//
// core를 임포트하지 않는 이유: 렌더 층은 값 층을 임포트하지 않는다 (파이프라인이 둘을
// 조립한다 — internal/pipeline). 이름 하나를 위해 그 방향을 열면 렌더러가 설정 구조체를
// 읽을 수 있게 되고, 그러면 "확정 모델만 읽는다"는 계약이 문서상의 약속으로 내려간다.
const configFileName = "balpyo.json"

// sibGroup은 형제 집합의 열쇠다 — 슬라이드 하나 안의 부모 하나.
type sibGroup struct {
	slide  int
	parent string
}

// escape는 inner가 outer를 벗어난 가장 큰 방향과 그 양이다 (px). 벗어나지 않았으면 0이다.
//
// 가장 큰 한 방향만 말하는 이유: 요소 하나당 진단 하나다. 두 방향으로 넘친 박스는 큰 쪽을
// 고치는 동안 작은 쪽도 함께 움직이고, 남으면 다음 빌드가 다시 말한다.
func escape(inner, outer Rect) (string, float64) {
	// 순서는 흔한 사고 순이다 — 같은 양으로 벗어났을 때의 결정적 tie-break도 이 순서다.
	sides := [...]struct {
		dir  string
		over float64
	}{
		{"at the bottom", inner.Bottom() - outer.Bottom()},
		{"on the right", inner.Right() - outer.Right()},
		{"at the top", outer.Y - inner.Y},
		{"on the left", outer.X - inner.X},
	}
	dir, max := "", 0.0
	for _, side := range sides {
		if side.over > max {
			dir, max = side.dir, side.over
		}
	}
	return dir, max
}

// textEscape는 이 박스가 **직접 그린 글자**가 박스를 가로로 벗어난 양이다 (px).
//
// 가로만 보는 이유는 두 가지다. (1) 박스 폭은 배치가 정하므로(`.balpyo-text { width: 100% }`·
// 컨테이너 트랙) 내용이 폭을 넓힐 길이 없고, 줄바꿈이 불가능한 글자(긴 URL, `<pre>`의 한 줄)는
// 그대로 옆으로 흐른다 — 저자가 열어 보지 않으면 모르는 사고의 전형이다. (2) 세로는 글자
// 상자가 줄 박스보다 커서(파일 앞 주석) 줄 박스 합이 늘 박스보다 몇 px 크다 — 세로로는 줄
// 박스가 아무것도 말해 주지 않는다.
func textEscape(b Box) (string, float64) {
	dir, max := "", 0.0
	for _, line := range b.Lines {
		if over := line.Right() - b.Rect.Right(); over > max {
			dir, max = "on the right", over
		}
		if over := b.Rect.X - line.X; over > max {
			dir, max = "on the left", over
		}
	}
	return dir, max
}

// overlap은 두 사각형이 겹친 폭과 높이다. 두 축 모두 허용 오차를 넘겨야 겹침이다 — 맞닿은
// 변(겹침 0)과 크기 0인 박스(`<spacer>`)가 겹침으로 세어지면 안 된다.
func overlap(a, b Rect) (float64, float64, bool) {
	w := math.Min(a.Right(), b.Right()) - math.Max(a.X, b.X)
	h := math.Min(a.Bottom(), b.Bottom()) - math.Max(a.Y, b.Y)
	return w, h, w > overflowTolerance && h > overflowTolerance
}

// px는 진단 메시지의 픽셀 표기다. 소수점을 보여 주지 않는 이유: 저자가 고칠 단위는 px이고,
// 0.4px의 차이는 고칠 수 있는 것이 아니다 (허용 오차가 이미 그것을 걸러 낸다).
func px(v float64) string { return fmt.Sprintf("%.0fpx", v) }
