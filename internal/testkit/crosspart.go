package testkit

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"path"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// 파트 사이 규칙: 스키마는 파트 하나의 모양만 말한다. PowerPoint가 복구를 묻는 나머지 —
// 파트 사이의 참조가 맞는가 — 는 여기서 본다.
//
//   - 슬라이드 · 레이아웃 · 마스터 · 노트 파트 안에서 `cNvPr@id`가 겹치지 않는다.
//   - `sldId@id`는 256 이상 2147483648 미만이고 겹치지 않는다 · `sldMasterId`/`sldLayoutId`의 id는
//     2147483648 이상이고 둘을 합쳐 겹치지 않는다.
//   - `r:id`/`r:embed`가 그 파트의 관계를 가리키고, 그 관계의 종류가 요소에 맞는다 (`a:blip` → image …).
//   - 슬라이드는 레이아웃 하나를, 레이아웃은 마스터 하나를 가리킨다 · 마스터의 `sldLayoutIdLst`가 자기를
//     가리키는 레이아웃과 같다 · 프레젠테이션의 `sldIdLst`가 슬라이드 파트를 모두 담는다.
//   - 노트 파트는 노트 마스터와 슬라이드를 가리키고, 그 슬라이드가 노트 파트를 되가리킨다.
//   - 슬라이드의 `p:ph`는 레이아웃에 같은 자리(`idx`, 없으면 `type`)가 있다.

const (
	relNS      = "http://schemas.openxmlformats.org/officeDocument/2006/relationships"
	relTypeDir = "http://schemas.openxmlformats.org/officeDocument/2006/relationships/"
	presPart   = "ppt/presentation.xml"
	// 마스터 · 레이아웃 id의 하한 (ECMA-376 Part 1 §19.2.1.27 · §19.3.1.41).
	minMasterID = 2147483648
	minSlideID  = 256
)

// relRefTypes는 관계를 참조하는 요소마다 그 관계가 가져야 할 종류다. 렌더러가 새 참조를 내기
// 시작하면 이 표가 모른다고 실패한다 — 종류를 정해서 더한다.
var relRefTypes = map[string]string{
	"sldMasterId":   "slideMaster",
	"sldId":         "slide",
	"notesMasterId": "notesMaster",
	"sldLayoutId":   "slideLayout",
	"blip":          "image",
	"hlinkClick":    "hyperlink",
}

// partRel은 관계 하나다 — Type은 마지막 경로 조각(`slideLayout`), Target은 패키지 안 경로로 푼 것이다.
type partRel struct {
	Type, Target string
	External     bool
}

// partFacts는 XML 파트 하나에서 읽은 것이다.
type partFacts struct {
	shapeIDs []string
	refs     []relRef
	ids      map[string][]uint64 // sldId · sldMasterId · sldLayoutId → @id
	ph       []placeholder
}

type relRef struct{ element, id string }

type placeholder struct{ typ, idx string }

func (p placeholder) String() string {
	if p.idx != "" {
		return fmt.Sprintf("idx=%s(%s)", p.idx, p.typ)
	}
	return "type=" + p.typ
}

// AssertCrossParts는 CheckCrossParts가 낸 위반을 실패로 보고한다.
func AssertCrossParts(t testing.TB, parts map[string][]byte) {
	t.Helper()
	for _, problem := range CheckCrossParts(parts) {
		t.Error(problem)
	}
}

// CheckCrossParts는 파트 사이 규칙의 위반을 파트 이름 순으로 돌려준다.
func CheckCrossParts(parts map[string][]byte) []string {
	var problems []string
	fail := func(format string, args ...any) { problems = append(problems, fmt.Sprintf(format, args...)) }

	facts := map[string]*partFacts{}
	for _, name := range sortedParts(parts) {
		if !strings.HasSuffix(name, ".xml") || name == contentTypesPart {
			continue
		}
		f, err := readPartFacts(parts[name])
		if err != nil {
			fail("%s를 읽을 수 없다: %v", name, err)
			continue
		}
		facts[name] = f

		seen := map[string]bool{}
		for _, id := range f.shapeIDs {
			if seen[id] {
				fail("%s에 cNvPr@id=%s가 두 번 있다 — PowerPoint가 복구를 묻는다", name, id)
			}
			seen[id] = true
		}
		rels := readPartRels(parts, name)
		for _, ref := range f.refs {
			want, known := relRefTypes[ref.element]
			rel, ok := rels[ref.id]
			switch {
			case !known:
				fail("%s의 %s가 관계를 참조한다 — relRefTypes에 그 종류가 없다", name, ref.element)
			case !ok:
				fail("%s의 %s가 없는 관계 %s를 가리킨다", name, ref.element, ref.id)
			case rel.Type != want:
				fail("%s의 %s가 %s 관계 %s를 가리킨다 — %s여야 한다", name, ref.element, rel.Type, ref.id, want)
			}
		}
	}

	// 프레젠테이션의 id 공간.
	if pres := facts[presPart]; pres != nil {
		seen := map[uint64]bool{}
		for _, id := range pres.ids["sldId"] {
			if id < minSlideID || id >= minMasterID {
				fail("%s의 sldId@id=%d가 [%d, %d) 밖이다", presPart, id, minSlideID, uint64(minMasterID))
			}
			if seen[id] {
				fail("%s에 sldId@id=%d가 두 번 있다", presPart, id)
			}
			seen[id] = true
		}
		if n, want := len(pres.ids["sldId"]), len(partsWithPrefix(parts, "ppt/slides/slide")); n != want {
			fail("%s의 sldIdLst가 %d장, 슬라이드 파트는 %d개다", presPart, n, want)
		}
	} else {
		fail("%s가 없다", presPart)
	}
	masterIDs := map[uint64]string{}
	for _, name := range sortedFacts(facts) {
		for _, kind := range []string{"sldMasterId", "sldLayoutId"} {
			for _, id := range facts[name].ids[kind] {
				if id < minMasterID {
					fail("%s의 %s@id=%d가 %d보다 작다", name, kind, id, uint64(minMasterID))
				}
				if owner, dup := masterIDs[id]; dup {
					fail("%s의 %s@id=%d가 %s의 id와 겹친다 — 마스터와 레이아웃은 한 id 공간이다", name, kind, id, owner)
				}
				masterIDs[id] = name
			}
		}
	}

	// 슬라이드 → 레이아웃 → 마스터.
	layoutsOf := map[string][]string{}
	for _, layout := range partsWithPrefix(parts, "ppt/slideLayouts/slideLayout") {
		if master, ok := singleRel(parts, layout, "slideMaster", fail); ok {
			layoutsOf[master] = append(layoutsOf[master], layout)
		}
	}
	for _, master := range partsWithPrefix(parts, "ppt/slideMasters/slideMaster") {
		var listed []string
		rels := readPartRels(parts, master)
		for _, ref := range facts[master].refs {
			if ref.element == "sldLayoutId" {
				listed = append(listed, rels[ref.id].Target)
			}
		}
		sort.Strings(listed)
		if got, want := strings.Join(listed, ","), strings.Join(layoutsOf[master], ","); got != want {
			fail("%s의 sldLayoutIdLst가 [%s]인데 이 마스터를 가리키는 레이아웃은 [%s]다", master, got, want)
		}
	}
	for _, slide := range partsWithPrefix(parts, "ppt/slides/slide") {
		layout, ok := singleRel(parts, slide, "slideLayout", fail)
		if !ok || facts[layout] == nil {
			continue
		}
		have := map[string]bool{}
		for _, ph := range facts[layout].ph {
			have[ph.key()] = true
		}
		for _, ph := range facts[slide].ph {
			if !have[ph.key()] {
				fail("%s의 플레이스홀더 %s가 %s에 없다 — 상속할 자리가 없다", slide, ph, layout)
			}
		}
	}

	// 노트 ↔ 슬라이드.
	for _, notes := range partsWithPrefix(parts, "ppt/notesSlides/notesSlide") {
		singleRel(parts, notes, "notesMaster", fail)
		slide, ok := singleRel(parts, notes, "slide", fail)
		if !ok {
			continue
		}
		if back, ok := singleRel(parts, slide, "notesSlide", fail); ok && back != notes {
			fail("%s는 %s를 가리키는데 %s의 노트는 %s다", notes, slide, slide, back)
		}
	}
	return problems
}

// key는 슬라이드의 자리가 레이아웃의 자리와 짝지어지는 열쇠다 — idx가 있으면 idx, 없으면 type.
func (p placeholder) key() string {
	if p.idx != "" && p.idx != "0" {
		return "idx:" + p.idx
	}
	return "type:" + p.typ
}

// readPartFacts는 파트 하나를 훑어 도형 id · 관계 참조 · 목록 id · 플레이스홀더를 모은다.
func readPartFacts(data []byte) (*partFacts, error) {
	f := &partFacts{ids: map[string][]uint64{}}
	dec := xml.NewDecoder(bytes.NewReader(data))
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			return f, nil
		}
		if err != nil {
			return nil, err
		}
		el, ok := tok.(xml.StartElement)
		if !ok {
			continue
		}
		var id, typ, idx string
		for _, attr := range el.Attr {
			switch {
			case attr.Name.Space == relNS:
				f.refs = append(f.refs, relRef{element: el.Name.Local, id: attr.Value})
			case attr.Name.Space == "" && attr.Name.Local == "id":
				id = attr.Value
			case attr.Name.Space == "" && attr.Name.Local == "type":
				typ = attr.Value
			case attr.Name.Space == "" && attr.Name.Local == "idx":
				idx = attr.Value
			}
		}
		switch el.Name.Local {
		case "cNvPr":
			f.shapeIDs = append(f.shapeIDs, id)
		case "sldId", "sldMasterId", "sldLayoutId":
			n, err := strconv.ParseUint(id, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("%s@id=%q: %v", el.Name.Local, id, err)
			}
			f.ids[el.Name.Local] = append(f.ids[el.Name.Local], n)
		case "ph":
			if typ == "" {
				typ = "obj" // ST_PlaceholderType의 기본값
			}
			f.ph = append(f.ph, placeholder{typ: typ, idx: idx})
		}
	}
}

// readPartRels는 파트 하나의 관계 표다 (rels 파트가 없으면 빈 표). 모양은 assertRels가 본다.
func readPartRels(parts map[string][]byte, owner string) map[string]partRel {
	dir := path.Dir(owner)
	var rels struct {
		Rels []struct {
			ID         string `xml:"Id,attr"`
			Type       string `xml:"Type,attr"`
			Target     string `xml:"Target,attr"`
			TargetMode string `xml:"TargetMode,attr"`
		} `xml:"Relationship"`
	}
	out := map[string]partRel{}
	if err := xml.Unmarshal(parts[path.Join(dir, "_rels", path.Base(owner)+".rels")], &rels); err != nil {
		return out
	}
	for _, rel := range rels.Rels {
		r := partRel{Type: strings.TrimPrefix(rel.Type, relTypeDir), External: rel.TargetMode == "External", Target: rel.Target}
		if !r.External {
			r.Target = path.Join(dir, rel.Target)
			if strings.HasPrefix(rel.Target, "/") {
				r.Target = strings.TrimPrefix(rel.Target, "/")
			}
		}
		out[rel.ID] = r
	}
	return out
}

// singleRel은 파트가 typ 종류의 관계를 정확히 하나 갖는지 보고 그 대상을 돌려준다.
func singleRel(parts map[string][]byte, owner, typ string, fail func(string, ...any)) (string, bool) {
	var targets []string
	for _, rel := range readPartRels(parts, owner) {
		if rel.Type == typ {
			targets = append(targets, rel.Target)
		}
	}
	if len(targets) != 1 {
		sort.Strings(targets)
		fail("%s의 %s 관계가 %d개다 %v — 하나여야 한다", owner, typ, len(targets), targets)
		return "", false
	}
	return targets[0], true
}

// partsWithPrefix는 접두사로 시작하는 `.xml` 파트를 이름 순으로 돌려준다.
func partsWithPrefix(parts map[string][]byte, prefix string) []string {
	var names []string
	for _, name := range sortedParts(parts) {
		if strings.HasPrefix(name, prefix) && strings.HasSuffix(name, ".xml") {
			names = append(names, name)
		}
	}
	return names
}

func sortedFacts(facts map[string]*partFacts) []string {
	names := make([]string, 0, len(facts))
	for name := range facts {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
