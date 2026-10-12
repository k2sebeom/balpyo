package geom

import (
	"fmt"
	"strings"

	"github.com/k2sebeom/balpyo/soml"
)

// 이 파일은 펼친 차트의 배치 진단이다.
//
// 값 층은 `<chart>`를 도형·텍스트의 하위 트리로 펼치고, 그 트리의 **틀 노드**(플롯 · 칸 · 기준선 ·
// 범주 라벨 줄 · 범례)는 어느 한 점·계열의 것이 아니라서 저자가 쓴 이름이 뿌리와 같은 `<chart>`다.
// 그래서 틀 노드의 넘침을 다른 요소처럼 말하면 `<chart> overflows its parent <chart>`가 되고, 저자는 어느
// 부분이 넘쳤는지도, 고칠 자리가 하나라는 것도 듣지 못한다. 틀이 넘치는 사고의 원인은 언제나 하나 —
// 차트가 내용에 비해 작다 — 이므로 차트 하나의 틀 넘침은 **경고 하나**로 모아 넘친 부분들의 이름으로
// 말한다.
//
// 같은 경고가 막대의 자리를 함께 본다. 막대 칸은 위에서 아래로 여백 · 값 라벨 · 막대이고, 라벨은
// 축소되지 않으므로 차트가 라벨들의 높이만큼도 안 되면 flex 축소가 여백과 막대를 0으로 거둔다 — 그림은
// 라벨만 남고 막대는 조용히 사라진다. 그 사실은 박스의 이탈로는 드러나지 않을 수 있으므로(라벨이 꼭 맞게
// 들어가도 막대는 0이다) 막대가 설 높이를 직접 잰다.

// chartEscape는 차트 하나의 틀 넘침을 모은 것이다.
type chartEscape struct {
	// at은 이 차트의 경고 자리다 (diags 안). 넘침이 없으면 -1이다.
	at int
	// parts는 넘친 부분의 이름이다 (처음 나온 순서).
	parts []string
	dir   string
	over  float64
}

func (c *chartEscape) add(part, dir string, over float64) {
	found := false
	for _, p := range c.parts {
		found = found || p == part
	}
	if !found {
		c.parts = append(c.parts, part)
	}
	if over > c.over {
		c.dir, c.over = dir, over
	}
}

// chartRoom은 차트 하나의 데이터가 설 높이다 — 막대 칸마다 여백 + 막대, 선 차트면 선 밴드의 높이.
type chartRoom struct {
	// data는 진단이 데이터를 부를 이름이다 ("bars" · "lines").
	data string
	// bands는 막대 칸(식별자) → 여백 + 막대 높이다.
	bands map[string]float64
	// height는 bands와 선 밴드 가운데 가장 높은 것이다 (finish가 채운다).
	height float64
	// seen은 데이터의 자리를 하나라도 쟀는지다 — 재지 못한 차트에는 말하지 않는다.
	seen bool
}

// chartDiagnosis는 Diagnose 한 번의 차트 판정 상태다.
type chartDiagnosis struct {
	// root는 요소(refs 인덱스) → 그 요소가 속한 차트 뿌리의 인덱스다 (뿌리 자신도 담는다).
	root    map[int]int
	escapes map[int]*chartEscape
	rooms   map[int]*chartRoom
	order   []int
}

func newChartDiagnosis() *chartDiagnosis {
	return &chartDiagnosis{root: map[int]int{}, escapes: map[int]*chartEscape{}, rooms: map[int]*chartRoom{}}
}

// visit은 요소 하나를 차트 문맥에 넣고, 그것이 데이터의 자리면 높이를 적는다.
func (c *chartDiagnosis) visit(refs []Ref, index map[string]int, i int, box Box) {
	ref := refs[i]
	if ref.Tag == soml.ChartElement && ref.Part == "" {
		c.root[i] = i
		c.order = append(c.order, i)
		return
	}
	parent, ok := index[ref.Parent]
	if !ok {
		return
	}
	root, ok := c.root[parent]
	if !ok {
		return
	}
	c.root[i] = root
	room := c.rooms[root]
	if room == nil {
		room = &chartRoom{bands: map[string]float64{}}
		c.rooms[root] = room
	}
	switch {
	case ref.Part == soml.ChartPartPlot && ref.El == soml.SpacerElement,
		ref.Tag == soml.ChartPointElement && ref.El == soml.ShapeElement:
		// 막대 칸의 여백과 막대다 — 둘의 합이 그 칸에서 데이터가 설 높이다 (값 0인 점은 막대가 없고
		// 여백이 전부다).
		room.data, room.seen = "bars", true
		room.bands[ref.Parent] += box.Rect.H
	case ref.Part == soml.ChartPartPlot && ref.El == soml.StackElement:
		// 선 차트의 밴드다 — 경로 도형들이 이 높이로 늘어난다.
		room.data, room.seen = "lines", true
		room.height = max(room.height, box.Rect.H)
	}
}

// frame은 이탈 경고를 받은 틀 노드 하나를 그 차트의 경고에 모은다. 돌려주는 값은 그 경고의 자리이고,
// 모을 차트가 없으면(뿌리를 재지 않은 손 입력) false다. 차트의 첫 틀 넘침이 경고 자리를 만든다 — 문장은
// 넘친 부분이 다 모인 뒤에 쓴다 (finish).
func (c *chartDiagnosis) frame(refs []Ref, i int, dir string, over float64, diags *soml.Diagnostics) (int, bool) {
	root, ok := c.root[i]
	if !ok {
		return 0, false
	}
	esc := c.escapes[root]
	if esc == nil {
		ref := refs[root]
		esc = &chartEscape{at: len(*diags)}
		c.escapes[root] = esc
		*diags = append(*diags, soml.Warningf(ref.Source, ref.Span, ""))
	}
	esc.add(refs[i].Part, dir, over)
	return esc.at, true
}

// isEscape는 diags의 자리 at이 차트 틀 넘침 경고인지다.
func (c *chartDiagnosis) isEscape(at int) bool {
	for _, esc := range c.escapes {
		if esc.at == at {
			return true
		}
	}
	return false
}

// finish는 차트마다의 경고 문장을 쓴다 — 틀 넘침이 있으면 그 자리의 문장을 채우고, 넘침 없이 데이터가
// 설 높이만 없으면 새 경고를 덧붙인다.
func (c *chartDiagnosis) finish(refs []Ref, diags soml.Diagnostics) soml.Diagnostics {
	for _, root := range c.order {
		esc := c.escapes[root]
		room := c.rooms[root]
		noRoom := ""
		if room != nil && room.seen {
			for _, h := range room.bands {
				room.height = max(room.height, h)
			}
			if room.height <= overflowTolerance {
				noRoom = "nothing is left for the " + room.data
			}
		}
		const fix = " (give the chart more room or shorten its labels)"
		switch {
		case esc != nil:
			verb := "overflow"
			if len(esc.parts) == 1 {
				verb = "overflows"
			}
			msg := fmt.Sprintf("<%s> is too small for its content — the %s %s the chart %s by %s",
				soml.ChartElement, joinParts(esc.parts), verb, esc.dir, px(esc.over))
			if noRoom != "" {
				msg += " and " + noRoom
			}
			diags[esc.at].Message = msg + fix
		case noRoom != "":
			ref := refs[root]
			diags = append(diags, soml.Warningf(ref.Source, ref.Span,
				"<%s> is too small for its content — %s%s", soml.ChartElement, noRoom, fix))
		}
	}
	return diags
}

// joinParts는 부분 이름들을 영어 나열로 잇는다 ("legend" · "plot and legend" · "plot, category labels and legend").
func joinParts(parts []string) string {
	if len(parts) <= 1 {
		return strings.Join(parts, "")
	}
	return strings.Join(parts[:len(parts)-1], ", ") + " and " + parts[len(parts)-1]
}
