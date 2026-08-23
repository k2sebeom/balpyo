package testkit

import (
	"fmt"
	"image"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/k2sebeom/balpyo/core"
	"github.com/k2sebeom/balpyo/render/geom"
	"github.com/k2sebeom/balpyo/render/pptx"
	"github.com/k2sebeom/balpyo/render/web"
	"github.com/k2sebeom/balpyo/soml"
)

// 텍스트 픽스처들의 픽스처 수준 검증. PPTX 구조의 단위 단정은 render/pptx의 테스트가 이미
// 갖고 있다 — 여기서는 실제 픽스처의 산출물 전체를 놓고, 골든에 들어간 파트 목록·좌표·토큰
// 보존이 기대와 같은지, 그리고 두 렌더 엔진이 같은 자리에서 줄을 나누는지를 본다 (PRD §13-1).
const (
	// textFixtureName은 Iteration 1의 <text> 전용 픽스처다.
	textFixtureName = "01_text"
	// textSubsetFixtureName은 Iteration 2의 HTML 텍스트 서브셋 픽스처다 (블록·리스트·
	// 인라인 서식·pre — 항목 02/08이 활성화했다).
	textSubsetFixtureName = "02_text_subset"
	// layoutFixtureName은 Iteration 3의 레이아웃 태그 픽스처다 (row/col/grid/spacer +
	// 자식 배치 속성 + `style` 탈출구 — 항목 03/05가 활성화했다). 도형은 여전히 텍스트
	// 박스뿐이라 아래 검증들(줄 수 대조·잉크 띠의 단색 배경 가정·schemeClr)이 그대로
	// 성립한다 — 이 픽스처가 새로 시험하는 것은 **한 줄에 여러 박스가 앉는 배치**다.
	layoutFixtureName = "03_layout"
)

// textFixtureNames는 픽스처 수준 검증(좌표 대조·잉크 띠·schemeClr)이 도는 텍스트 픽스처
// 전부다 — 새 텍스트 픽스처가 생기면 여기 더해 같은 계약을 받게 한다.
var textFixtureNames = []string{textFixtureName, textSubsetFixtureName, layoutFixtureName}

// fixtureByName은 픽스처 러너와 같은 수집 경로로 픽스처를 집는다 — 경로를 직접 조립하면
// 러너가 못 보는 픽스처를 여기만 보는 상태가 생길 수 있다.
func fixtureByName(t *testing.T, name string) Fixture {
	t.Helper()
	for _, fixture := range Fixtures(t) {
		if fixture.Name == name {
			return fixture
		}
	}
	t.Fatalf("픽스처 %s가 수집되지 않았다", name)
	return Fixture{}
}

// textFixture는 01_text다 (presenter_test 등 Iteration 1 전용 검증이 쓴다).
func textFixture(t *testing.T) Fixture {
	t.Helper()
	return fixtureByName(t, textFixtureName)
}

// goldenPartLists는 픽스처별 골든 파트 목록의 전수 기대값이다. 파트별 XML은 골든 비교가
// 보지만, "무엇이 있어야 하고 무엇이 없어야 하는가"는 목록이 말한다 — 특히 노트 파트는
// 노트가 있는 슬라이드에만 있어야 하고(01_text는 1·3, 02_text_subset은 5),
// diagnostics.txt는 경고가 있을 때만 생긴다: 01_text는 경고 없이 빌드되어야 하고,
// 02_text_subset은 모르는 variant 슬라이드가 경고를 내므로 **있어야** 한다 — 그 골든이
// 경고 경로(문구·위치)의 유일한 회귀 테스트다 (fixtures README).
var goldenPartLists = map[string][]string{
	textFixtureName: {
		"index.html",
		"pptx/[Content_Types].xml",
		"pptx/_rels/.rels",
		"pptx/docProps/app.xml",
		"pptx/docProps/core.xml",
		"pptx/ppt/_rels/presentation.xml.rels",
		"pptx/ppt/notesMasters/_rels/notesMaster1.xml.rels",
		"pptx/ppt/notesMasters/notesMaster1.xml",
		"pptx/ppt/notesSlides/_rels/notesSlide1.xml.rels",
		"pptx/ppt/notesSlides/_rels/notesSlide3.xml.rels",
		"pptx/ppt/notesSlides/notesSlide1.xml",
		"pptx/ppt/notesSlides/notesSlide3.xml",
		"pptx/ppt/presentation.xml",
		"pptx/ppt/slideLayouts/_rels/slideLayout1.xml.rels",
		"pptx/ppt/slideLayouts/slideLayout1.xml",
		"pptx/ppt/slideMasters/_rels/slideMaster1.xml.rels",
		"pptx/ppt/slideMasters/slideMaster1.xml",
		"pptx/ppt/slides/_rels/slide1.xml.rels",
		"pptx/ppt/slides/_rels/slide2.xml.rels",
		"pptx/ppt/slides/_rels/slide3.xml.rels",
		"pptx/ppt/slides/slide1.xml",
		"pptx/ppt/slides/slide2.xml",
		"pptx/ppt/slides/slide3.xml",
		"pptx/ppt/theme/theme1.xml",
	},
	textSubsetFixtureName: {
		"diagnostics.txt",
		"index.html",
		"pptx/[Content_Types].xml",
		"pptx/_rels/.rels",
		"pptx/docProps/app.xml",
		"pptx/docProps/core.xml",
		"pptx/ppt/_rels/presentation.xml.rels",
		"pptx/ppt/notesMasters/_rels/notesMaster1.xml.rels",
		"pptx/ppt/notesMasters/notesMaster1.xml",
		"pptx/ppt/notesSlides/_rels/notesSlide5.xml.rels",
		"pptx/ppt/notesSlides/notesSlide5.xml",
		"pptx/ppt/presentation.xml",
		"pptx/ppt/slideLayouts/_rels/slideLayout1.xml.rels",
		"pptx/ppt/slideLayouts/slideLayout1.xml",
		"pptx/ppt/slideMasters/_rels/slideMaster1.xml.rels",
		"pptx/ppt/slideMasters/slideMaster1.xml",
		"pptx/ppt/slides/_rels/slide1.xml.rels",
		"pptx/ppt/slides/_rels/slide2.xml.rels",
		"pptx/ppt/slides/_rels/slide3.xml.rels",
		"pptx/ppt/slides/_rels/slide4.xml.rels",
		"pptx/ppt/slides/_rels/slide5.xml.rels",
		"pptx/ppt/slides/_rels/slide6.xml.rels",
		"pptx/ppt/slides/slide1.xml",
		"pptx/ppt/slides/slide2.xml",
		"pptx/ppt/slides/slide3.xml",
		"pptx/ppt/slides/slide4.xml",
		"pptx/ppt/slides/slide5.xml",
		"pptx/ppt/slides/slide6.xml",
		"pptx/ppt/theme/theme1.xml",
	},
	layoutFixtureName: {
		// 007이 `style`에 외관을 써서 경고 2건을 내므로 diagnostics.txt가 **있어야** 한다 —
		// 그 골든이 경고 경로(문구·위치·순서)의 유일한 회귀 테스트다.
		"diagnostics.txt",
		"index.html",
		"pptx/[Content_Types].xml",
		"pptx/_rels/.rels",
		"pptx/docProps/app.xml",
		"pptx/docProps/core.xml",
		"pptx/ppt/_rels/presentation.xml.rels",
		"pptx/ppt/notesMasters/_rels/notesMaster1.xml.rels",
		"pptx/ppt/notesMasters/notesMaster1.xml",
		// 노트는 001에만 있다 — 레이아웃 컨테이너가 노트 파트를 만들지 않는다는 사실도
		// 이 목록이 지킨다.
		"pptx/ppt/notesSlides/_rels/notesSlide1.xml.rels",
		"pptx/ppt/notesSlides/notesSlide1.xml",
		"pptx/ppt/presentation.xml",
		"pptx/ppt/slideLayouts/_rels/slideLayout1.xml.rels",
		"pptx/ppt/slideLayouts/slideLayout1.xml",
		"pptx/ppt/slideMasters/_rels/slideMaster1.xml.rels",
		"pptx/ppt/slideMasters/slideMaster1.xml",
		"pptx/ppt/slides/_rels/slide1.xml.rels",
		"pptx/ppt/slides/_rels/slide2.xml.rels",
		"pptx/ppt/slides/_rels/slide3.xml.rels",
		"pptx/ppt/slides/_rels/slide4.xml.rels",
		"pptx/ppt/slides/_rels/slide5.xml.rels",
		"pptx/ppt/slides/_rels/slide6.xml.rels",
		"pptx/ppt/slides/_rels/slide7.xml.rels",
		"pptx/ppt/slides/slide1.xml",
		"pptx/ppt/slides/slide2.xml",
		"pptx/ppt/slides/slide3.xml",
		"pptx/ppt/slides/slide4.xml",
		"pptx/ppt/slides/slide5.xml",
		"pptx/ppt/slides/slide6.xml",
		"pptx/ppt/slides/slide7.xml",
		"pptx/ppt/theme/theme1.xml",
	},
}

// 골든에 들어간 파트 목록을 픽스처별로 통째로 단정한다.
func TestTextFixtureGoldenPartList(t *testing.T) {
	for _, name := range textFixtureNames {
		t.Run(name, func(t *testing.T) {
			want := goldenPartLists[name]
			var got []string
			goldenDir := fixtureByName(t, name).GoldenDir()
			err := filepath.Walk(goldenDir, func(path string, info os.FileInfo, err error) error {
				if err != nil || info.IsDir() {
					return err
				}
				rel, err := filepath.Rel(goldenDir, path)
				if err != nil {
					return err
				}
				got = append(got, filepath.ToSlash(rel))
				return nil
			})
			if err != nil {
				t.Fatalf("골든이 없다 (`make golden`으로 만든다): %v", err)
			}

			wantSet := map[string]bool{}
			for _, name := range want {
				wantSet[name] = true
			}
			for _, name := range got {
				if !wantSet[name] {
					t.Errorf("기대하지 않은 골든 파트: %s", name)
				}
				delete(wantSet, name)
			}
			for name := range wantSet {
				t.Errorf("골든에 파트가 없다: %s", name)
			}
		})
	}
}

// slideCounts는 픽스처별 슬라이드 수다 (골든 슬라이드 파트 순회용).
var slideCounts = map[string]int{
	textFixtureName:       3,
	textSubsetFixtureName: 6,
	layoutFixtureName:     7,
}

// 슬라이드 파트의 색은 전부 schemeClr 참조여야 한다. 픽스처의 색은 모두 토큰이므로 슬라이드에
// 리터럴 srgbClr가 하나라도 있으면 토큰이 어딘가에서 RGB로 눕혀진 것이다 — 그 도형은 테마 색
// 편집을 따라가지 못한다 (제품 핵심 계약, PRD §9.3 ②). 리터럴 RGB는 테마 파트의 토큰 선언
// 한 곳에만 있어야 한다. 02_text_subset은 인라인 서식의 런 색(링크 accent1, span 토큰 색)까지
// 이 계약을 받는다.
func TestTextFixtureGoldenSlidesKeepSchemeClr(t *testing.T) {
	for _, name := range textFixtureNames {
		t.Run(name, func(t *testing.T) {
			goldenDir := fixtureByName(t, name).GoldenDir()
			for i := 1; i <= slideCounts[name]; i++ {
				slide := fmt.Sprintf("slide%d.xml", i)
				data, err := os.ReadFile(filepath.Join(goldenDir, "pptx", "ppt", "slides", slide))
				if err != nil {
					t.Fatalf("골든이 없다 (`make golden`으로 만든다): %v", err)
				}
				xml := string(data)
				if strings.Contains(xml, "srgbClr") {
					t.Errorf("%s에 리터럴 srgbClr가 있다 — 토큰 색이 schemeClr 참조로 남지 않았다", slide)
				}
				if !strings.Contains(xml, "<a:schemeClr val=") {
					t.Errorf("%s에 schemeClr 참조가 없다", slide)
				}
			}
		})
	}
}

// 슬라이드 파트의 xfrm 좌표는 지오메트리 패스가 잰 값의 EMU 환산과 정확히 같아야 한다.
// 골든 XML만 보면 숫자가 그럴듯한지 알 수 없다 — 같은 빌드의 Result.Geometry와 맞대어
// "이 좌표가 그 측정에서 나왔다"를 못 박는다.
func TestTextFixtureCoordinatesMatchGeometry(t *testing.T) {
	RequireChromium(t) // 지오메트리 패스가 브라우저를 띄운다 — 없으면 건너뛴다(로컬).
	for _, name := range textFixtureNames {
		t.Run(name, func(t *testing.T) {
			fixture := fixtureByName(t, name)
			result := fixture.Build(t)
			if result.Geometry == nil {
				t.Fatal("텍스트가 있는 덱의 빌드에 지오메트리가 없다")
			}

			data, err := os.ReadFile(fixture.Output(t, result, ".pptx"))
			if err != nil {
				t.Fatal(err)
			}
			parts := PPTXParts(t, data)
			roles := shapeRoles(t, result.Deck)
			if len(roles) != len(result.Geometry.Boxes) {
				t.Fatalf("요소 %d개, 측정 박스 %d개", len(roles), len(result.Geometry.Boxes))
			}
			for i, box := range result.Geometry.Boxes {
				role := roles[i]
				if role.rect == nil {
					// 측정되지만 자기 도형을 갖지 않는 요소다: `<li>`는 컨테이너의 도형
					// 하나가 좌표를 소유하고 (pptx.writeListShape), **도형 안 텍스트**는
					// 도형의 txBody 문단이 되며 (pptx.shapeBodyXML), 레이아웃 컨테이너와
					// `<spacer>`는 도형이 아예 없다 (pptx.writeShapeNode) — 도형 수와의
					// 관계는 아래 TestTextFixtureShapeCountExcludesNonShapeElements가 본다.
					continue
				}
				part := fmt.Sprintf("ppt/slides/slide%d.xml", box.Slide+1)
				xml := string(parts[part])
				if xml == "" {
					t.Fatalf("파트 %s가 없다", part)
				}
				// 잰 rect → 도형 rect 변환은 종류가 정한다: 텍스트 박스는 정렬 방향의 안전
				// 여유를 더하고(PRD §9.3, pptx.TextBoxRect) 도형은 잰 값 그대로다 — 저자가
				// 정한 크기를 넓히면 그림이 커진다 (pptx.writeShapeSp). 같은 함수로 기대값을
				// 만들어 조립기가 그 값을 썼는지 본다.
				rect := role.rect(box.Rect)
				x, y := pptx.Offset(rect)
				cx, cy := pptx.Extent(rect)
				xfrm := fmt.Sprintf("<a:off x=\"%d\" y=\"%d\"/>\n            <a:ext cx=\"%d\" cy=\"%d\"/>", x, y, cx, cy)
				if !strings.Contains(xml, xfrm) {
					t.Errorf("%s: %s의 xfrm이 측정값 %s(→도형 rect)의 EMU와 다르다 (want %q)", part, box.ID, box.Rect, xfrm)
				}
			}
		})
	}
}

// 도형 수 = 측정 박스 수 − (레이아웃 컨테이너 + `<spacer>` + `<li>` + 도형 안 텍스트).
//
// 이 관계를 골든이 아니라 계약으로 두는 이유: "레이아웃 컨테이너는 도형이 되지 않는다"가
// 골든 XML의 모양으로만 남으면, 컨테이너가 도형을 내기 시작한 변경은 골든 diff에서 "도형이
// 하나 늘었다"로만 보이고 그것이 계약 위반인지 새 픽스처 슬라이드인지 사람이 판단해야 한다.
// 측정 목록(geom.Refs — 컨테이너도 잰다)과 실제 도형 수를 맞대면 그 판단이 필요 없다.
//
// 항이 넷인 이유는 도형이 들어왔기 때문이다 (Iteration 4): `<shape>`는 **재는 것과 도형이
// 되는 것이 처음 일치하는 종류**이지만 그 자식 텍스트는 재고도 도형이 되지 않는다 (부모
// 도형의 txBody 문단이 된다 — pptx.shapeBodyXML). 그래서 종류 판정이 태그 이름만으로는
// 부족하고 **부모가 도형인가**를 알아야 한다 — 판정을 shapeRoles의 순회 하나가 소유한다
// (좌표 대조와 같은 목록이어야 두 검사가 같은 계약을 본다).
func TestTextFixtureShapeCountExcludesNonShapeElements(t *testing.T) {
	RequireChromium(t) // 지오메트리 패스가 브라우저를 띄운다.
	for _, name := range textFixtureNames {
		t.Run(name, func(t *testing.T) {
			fixture := fixtureByName(t, name)
			result := fixture.Build(t)
			if result.Geometry == nil {
				t.Fatal("텍스트가 있는 덱의 빌드에 지오메트리가 없다")
			}
			data, err := os.ReadFile(fixture.Output(t, result, ".pptx"))
			if err != nil {
				t.Fatal(err)
			}
			parts := PPTXParts(t, data)

			roles := shapeRoles(t, result.Deck)
			if len(roles) != len(result.Geometry.Boxes) {
				t.Fatalf("요소 %d개, 측정 박스 %d개", len(roles), len(result.Geometry.Boxes))
			}
			want := 0
			excluded := map[string]int{}
			for _, role := range roles {
				if role.rect != nil {
					want++
					continue
				}
				excluded[role.excluded]++
			}
			got := 0
			for i := range result.Deck.Slides {
				got += strings.Count(string(parts[fmt.Sprintf("ppt/slides/slide%d.xml", i+1)]), "<p:sp>")
			}
			// 등식을 로그에 그대로 적는다 — 항 이름이 순회의 분류에서 오므로 항이 늘면
			// 로그도 함께 자란다 (손으로 적은 항 목록은 갱신을 잊는다).
			terms := make([]string, 0, len(shapeExcludedKinds))
			for _, kind := range shapeExcludedKinds {
				terms = append(terms, fmt.Sprintf("− %s %d", kind, excluded[kind]))
			}
			t.Logf("측정 박스 %d개 %s = 도형 %d개",
				len(roles), strings.Join(terms, " "), got)
			if got != want {
				t.Errorf("도형 %d개, want %d개 (측정 박스 %d %s)",
					got, want, len(roles), strings.Join(terms, " "))
			}
		})
	}
}

// 한 줄에 **여러 박스가 같은 y**에 앉는 배치에서 좌표 대조가 뜻을 유지하는가.
//
// 지금까지 모든 픽스처는 세로 스택이라 박스의 y가 단조 증가했고, "그리드가 그리드다"는 사실은
// 어느 검사도 보지 않았다 — 골든 xfrm은 숫자가 그럴듯한지 말해 주지 않고, 좌표 대조
// (TestTextFixtureCoordinatesMatchGeometry)는 측정값과 산출물이 같은지만 본다. 측정 자체가
// 어긋나면(트랙 계산이 한 칸씩 밀리면) 두 검사가 함께 통과한다.
//
// 그래서 배치의 뜻을 직접 단정한다: 같은 줄의 카드는 **같은 y**에 앉고(정확히 같은 EMU가
// 산출물에 있다), 4열 그리드의 x는 **등간격**이다. 이것이 TASKS의 리뷰 포인트("좌표 계산
// 0회로 그리드 배치가 PPTX에 정확히 이식되는지")의 자동화 가능한 절반이다 — 나머지 절반
// (편집 가능한 개별 도형인가)은 사람이 PowerPoint에서 본다.
func TestLayoutFixtureSameRowBoxesShareY(t *testing.T) {
	RequireChromium(t) // 지오메트리 패스가 브라우저를 띄운다.
	fixture := fixtureByName(t, layoutFixtureName)
	result := fixture.Build(t)
	if result.Geometry == nil {
		t.Fatal("빌드에 지오메트리가 없다")
	}
	data, err := os.ReadFile(fixture.Output(t, result, ".pptx"))
	if err != nil {
		t.Fatal(err)
	}
	parts := PPTXParts(t, data)

	// 슬라이드마다 "첫 줄에 나란히 앉아야 하는 요소"를 적는다. 001은 <row> 안의 두 <col>이고
	// 002는 <grid cols="4">의 카드 4개다 (다섯 번째 카드는 span="2"라 둘째 줄로 내려간다 —
	// 그 사실도 y 그룹이 갈리는 것으로 함께 확인된다).
	for _, tc := range []struct {
		slideID string
		el      string
		want    int
		pitch   bool // 등간격 x까지 보는가 (균등 트랙인 grid만)
	}{
		{slideID: "row-col", el: "h3", want: 2},
		{slideID: "grid-cards", el: "h3", want: 4, pitch: true},
	} {
		t.Run(tc.slideID, func(t *testing.T) {
			index := -1
			for i, slide := range result.Deck.Slides {
				if slide.ID == tc.slideID {
					index = i
				}
			}
			if index < 0 {
				t.Fatalf("슬라이드 %s가 없다", tc.slideID)
			}

			// 첫 줄 = 같은 요소 이름의 박스 중 y가 가장 작은 것들.
			var top []geom.Box
			for _, box := range result.Geometry.Boxes {
				if box.Slide != index || box.El != tc.el {
					continue
				}
				switch {
				case len(top) == 0 || box.Rect.Y < top[0].Rect.Y:
					top = []geom.Box{box}
				case box.Rect.Y == top[0].Rect.Y:
					top = append(top, box)
				}
			}
			if len(top) != tc.want {
				t.Fatalf("<%s>가 같은 y에 %d개, want %d개", tc.el, len(top), tc.want)
			}

			// PPTX의 y는 같은 EMU여야 한다. TextBoxRect의 안전 여유는 가로로만 붙으므로
			// (정렬이 정하는 방향 — pptx.TextBoxRect) y는 잰 값의 환산 그대로다.
			_, y := pptx.Offset(top[0].Rect)
			part := fmt.Sprintf("ppt/slides/slide%d.xml", index+1)
			xml := string(parts[part])
			if xml == "" {
				t.Fatalf("파트 %s가 없다", part)
			}
			if got := strings.Count(xml, fmt.Sprintf("y=\"%d\"", y)); got < tc.want {
				t.Errorf("%s: y=%d인 도형이 %d개, want %d개 이상 — 같은 줄의 박스가 같은 y로 굽히지 않았다",
					part, y, got, tc.want)
			}
			t.Logf("%s: <%s> %d개가 y=%.1fpx(%d EMU)를 공유한다", tc.slideID, tc.el, len(top), top[0].Rect.Y, y)

			if !tc.pitch {
				return
			}
			sort.Slice(top, func(i, j int) bool { return top[i].Rect.X < top[j].Rect.X })
			var pitches []float64
			for i := 1; i < len(top); i++ {
				pitches = append(pitches, top[i].Rect.X-top[i-1].Rect.X)
			}
			t.Logf("%s: 트랙 pitch %v (gap lg 36px 포함)", tc.slideID, pitches)
			for i, pitch := range pitches {
				// 트랙 폭은 (컨테이너 폭 − 3×gap)/4라 소수점이 남는다 — 브라우저가 트랙마다
				// 다른 방향으로 반올림하므로 1px까지는 등간격으로 본다. 한 칸이 밀리면
				// (트랙 계산이 틀리면) 차이는 트랙 폭 단위로 벌어진다.
				if math.Abs(pitch-pitches[0]) > 1.0 {
					t.Errorf("%s: %d번째 pitch %.1fpx가 첫 pitch %.1fpx와 다르다 — 트랙이 등간격이 아니다",
						tc.slideID, i+1, pitch, pitches[0])
				}
			}
		})
	}
}

// shapeRoles의 자리가 geom.Refs와 정확히 맞는가 — 좌표 대조와 도형 수 등식이 둘 다 이
// 자리 맞대기 위에 서 있다 (어긋나면 좌표가 엉뚱한 요소의 변환과 만나고, 그 실패는 산출물을
// 열어 보기 전까지 드러나지 않는다).
//
// 브라우저 없이 도는 검사다: geom.Refs는 덱 모델만 보고 만들어지므로(측정은 그 목록의 자리를
// 채우는 일이다) 순회 두 벌의 대조는 파싱·해석만으로 성립한다. 도형이 든 덱을 여기서
// 직접 만드는 이유는 도형 픽스처가 아직 비활성이기 때문이다 (항목 06이 켠다).
func TestShapeRolesLineUpWithGeometryRefs(t *testing.T) {
	stage, err := soml.Aspect16x9.Stage()
	if err != nil {
		t.Fatal(err)
	}
	src := `<slide>
  <ul>
    <li>첫째 <strong>강조</strong></li>
    <li>둘째</li>
  </ul>
  <col gap="md">
    <text>컨테이너 안 텍스트</text>
    <spacer/>
    <shape type="rect" fill="surface" w="240" h="120" pad="sm">
      <h3>도형 안 제목</h3>
      <p anchor="center">도형 안 본문</p>
    </shape>
  </col>
  <shape type="ellipse" fill="accent1" w="120" h="120"/>
</slide>`
	doc, diags := soml.Parse("slides/001.slide", src)
	if err := diags.Err(); err != nil {
		t.Fatalf("파싱 실패: %v", err)
	}
	deck := &soml.Deck{Title: "shape roles", Aspect: soml.Aspect16x9, Stage: stage}
	deck.Slides = append(deck.Slides, &soml.Slide{
		Index: 0, ID: "s1", Source: "slides/001.slide",
		Notes: doc.Notes, Content: doc.Content, Tokens: doc.Tokens,
	})
	if err := core.Resolve(deck).Err(); err != nil {
		t.Fatalf("외관 해석 실패: %v", err)
	}

	refs := geom.Refs(deck)
	roles := shapeRoles(t, deck)
	if len(roles) != len(refs) {
		t.Fatalf("역할 %d개, 측정 대상 %d개", len(roles), len(refs))
	}
	// 요소 이름 → 기대 분류. 도형 안 텍스트(h3·p)와 슬라이드 직속 텍스트(text)가 **같은
	// 종류인데 다른 분류**인 것이 이 순회의 존재 이유다.
	want := []struct {
		el       string
		excluded string
	}{
		{"ul", ""},
		{"li", "li"},
		{"li", "li"},
		{"col", "컨테이너"},
		{"text", ""},
		{"spacer", "spacer"},
		{"shape", ""},
		{"h3", "도형 안 텍스트"},
		{"p", "도형 안 텍스트"},
		{"shape", ""},
	}
	if len(want) != len(refs) {
		t.Fatalf("기대표 %d행, 측정 대상 %d개 (%v)", len(want), len(refs), refs)
	}
	for i, ref := range refs {
		if ref.El != want[i].el {
			t.Fatalf("%d번째 측정 대상이 <%s>다 (want <%s>) — 기대표가 순회와 어긋났다",
				i, ref.El, want[i].el)
		}
		role := roles[i]
		if want[i].excluded == "" {
			if role.rect == nil {
				t.Errorf("%d번째 <%s>가 도형이 되지 않는다고 표시되었다 (%s)", i, ref.El, role.excluded)
			}
			continue
		}
		if role.rect != nil {
			t.Errorf("%d번째 <%s>가 도형으로 표시되었다 — want %s", i, ref.El, want[i].excluded)
		}
		if role.excluded != want[i].excluded {
			t.Errorf("%d번째 <%s>의 분류 = %q, want %q", i, ref.El, role.excluded, want[i].excluded)
		}
	}
	// 분류 이름은 등식이 쓰는 어휘여야 한다 (로그의 항 이름이 순회에서 온다).
	for _, role := range roles {
		if role.excluded == "" {
			continue
		}
		found := false
		for _, kind := range shapeExcludedKinds {
			if kind == role.excluded {
				found = true
			}
		}
		if !found {
			t.Errorf("분류 %q가 shapeExcludedKinds에 없다 — 등식의 항이 빠진다", role.excluded)
		}
	}
	// 도형의 rect 변환은 잰 값 그대로이고, 텍스트 박스는 안전 여유를 더한다.
	measured := geom.Rect{X: 64, Y: 64, W: 240, H: 120}
	if got := roles[6].rect(measured); got != measured {
		t.Errorf("도형의 rect가 %s로 변했다 — 저자가 정한 크기를 넓혔다", got)
	}
	if got := roles[4].rect(measured); got == measured {
		t.Error("텍스트 박스에 안전 여유가 붙지 않았다")
	}
}

// shapeExcludedKinds는 **측정되지만 도형이 되지 않는** 요소의 분류다 (등식의 항 이름,
// 순서 = 등식에 적는 순서). shapeRoles가 붙이고 도형 수 등식이 로그에 그대로 쓴다.
var shapeExcludedKinds = []string{"컨테이너", "spacer", "li", "도형 안 텍스트"}

// shapeRole은 측정된 요소 하나가 PPTX에서 무엇이 되는지다.
type shapeRole struct {
	// rect는 잰 rect → 도형 rect 변환이다. nil이면 이 요소는 도형이 되지 않는다.
	rect func(geom.Rect) geom.Rect
	// excluded는 도형이 되지 않는 요소의 분류다 (shapeExcludedKinds의 값) — 도형이 되면
	// 빈 문자열이다.
	excluded string
}

// shapeRoles는 지오메트리와 같은 문서 순서(geom.Refs — 인라인은 자손까지 건너뛴다)로,
// 측정된 요소마다 PPTX 조립기가 그 도형의 좌표를 만드는 방법을 모은다.
//
// 자기 도형을 갖지 않는 요소는 rect가 nil이다 — `<li>`(컨테이너가 도형 하나다), **도형 안
// 텍스트**(부모 도형의 txBody 문단이다), 레이아웃 컨테이너와 `<spacer>`(도형이 아예 없다).
// 이 목록 하나를 좌표 대조와 도형 수 등식이 함께 쓰는 이유: 두 검사가 다른 판정을 들면
// 한쪽만 계약을 보게 되고, 어긋난 자리에서 좌표가 엉뚱한 요소와 만난다.
func shapeRoles(t *testing.T, deck *soml.Deck) []shapeRole {
	t.Helper()
	var roles []shapeRole
	textBox := func(align soml.TextAlign) func(geom.Rect) geom.Rect {
		return func(r geom.Rect) geom.Rect { return pptx.TextBoxRect(r, align) }
	}
	// 도형은 잰 rect **그대로**다 — 안전 여유는 글자를 잰 박스의 장치이고 도형의 박스는
	// 저자가 정한 크기다 (pptx.writeShapeSp).
	asMeasured := func(r geom.Rect) geom.Rect { return r }

	// inShape는 지금 도형 안을 걷고 있는지다 — 도형 안 텍스트는 태그 이름만으로는 슬라이드
	// 직속 텍스트와 구별되지 않는다 (같은 `<p>`다).
	var walk func(slide *soml.Slide, node soml.Node, inShape bool)
	walk = func(slide *soml.Slide, node soml.Node, inShape bool) {
		elem, ok := node.(*soml.Element)
		if !ok {
			// 문자 데이터는 자기 박스를 갖지 않는다 (geom.collect와 같은 규칙).
			return
		}
		kind := soml.KindOf(elem.Name)
		switch {
		case kind == soml.KindInline:
			// 인라인은 측정 대상이 아니다 — 자손까지 건너뛴다 (geom.collect의 규칙 그대로.
			// 여기가 갈라지면 목록과 Boxes의 자리가 어긋나 좌표가 엉뚱한 요소와 만난다).
			return
		case inShape:
			// 도형 안 텍스트는 **재지만 도형이 되지 않는다** (pptx.shapeBodyXML) — `<li>`와
			// 같은 처지다. 자리를 비워 두어야 뒤따르는 형제의 좌표가 제 변환과 만난다.
			roles = append(roles, shapeRole{excluded: "도형 안 텍스트"})
		case kind == soml.KindShape:
			roles = append(roles, shapeRole{rect: asMeasured})
		case kind == soml.KindListContainer:
			// 리스트 도형의 안전 여유는 언제나 start 정렬로 적용된다 — li는 align 속성을
			// 받지 않아 확정 정렬이 항상 start다 (pptx.writeListShape의 결정).
			roles = append(roles, shapeRole{rect: textBox(soml.AlignStart)})
		case kind == soml.KindListItem:
			roles = append(roles, shapeRole{excluded: "li"})
		case kind == soml.KindLayoutContainer:
			// 레이아웃 컨테이너와 `<spacer>`는 **측정되지만 도형이 되지 않는다**
			// (pptx.writeShapeNode — Iteration 3의 핵심 계약). 컨테이너는 자식으로
			// 내려가고(아래 순회), spacer는 void라 자식이 없다.
			roles = append(roles, shapeRole{excluded: "컨테이너"})
		case kind == soml.KindSpacer:
			roles = append(roles, shapeRole{excluded: "spacer"})
		default:
			style := elem.TextStyle()
			if style == nil {
				t.Fatalf("%s: <%s>의 외관이 해석되지 않았다", slide.Source, elem.Name)
			}
			roles = append(roles, shapeRole{rect: textBox(style.Align)})
		}
		for _, child := range elem.Children {
			// 도형의 자식은 도형 안이다 — 도형이 자식을 흡수하는 것과 컨테이너가 사라지는
			// 것이 여기서 정반대로 갈린다 (pptx.writeShapeNode의 네 분기).
			walk(slide, child, inShape || kind == soml.KindShape)
		}
	}
	for _, slide := range deck.Slides {
		for _, node := range slide.Content {
			walk(slide, node, false)
		}
	}
	return roles
}

// LibreOffice 렌더가 견뎌야 하는 글줄 오차의 임계값 (PRD §13-1의 수치 — 이 테스트가 잰
// 값에 여유를 더해 못 박은 회귀 감시선이다. 측정값은 NOTES.md 01/07의 표에 있다).
const (
	// loLineOffsetMaxPx는 글줄 잉크 세로 중심의 허용 오차다. spcPts(절대 줄간격)로 줄 전진
	// 폭은 맞췄지만 첫 기준선 배치는 엔진 소관이라 0이 되지 않는다 — LibreOffice가 일관되게
	// 아래로 +2~5px(측정 최대 +5.0px, 캡션 20px)이고, 6px은 거기에 픽셀 반올림 여유를 더한
	// 값이다. 이 선을 넘으면 줄간격 매핑이 다시 갈라진 것이다 (spcPct 시절에는 +8~13px였다).
	loLineOffsetMaxPx = 6.0
	// loLineWidthMaxRatio는 글줄 잉크 폭의 허용 오차 비율이다. LibreOffice는 한글↔라틴
	// 경계마다 스크립트 간격을 넣어 혼합 문자 줄을 최대 +3.3% 넓게 그린다 (측정값). 4%는
	// 텍스트 박스의 안전 여유(pptx.TextBoxRect)와 같은 값이다 — 폭 오차가 여유를 넘으면
	// 줄이 다시 접히므로, 이 검사가 깨지기 전에 줄 수 검사부터 깨지는 관계다.
	loLineWidthMaxRatio = 0.04
	// loLineWidthMinPx는 폭 오차의 절대 하한 허용값이다. 안티앨리어싱은 줄 길이와 무관하게
	// 글리프 경계에서 ±1px의 잉크를 만든다 — 비율 임계값은 긴 줄에서 측정한 값이라, 아주
	// 짧은 띠(pre의 `}` 한 글자, 잉크 폭 6px)에서는 그 1px이 16.7%로 계산되어 방법 문제로
	// 죽는다. 2px은 AA 잡음의 상한이고, 긴 줄에서는 비율 허용값이 항상 이보다 크므로
	// 게이트를 약화하지 않는다 (Iteration 2 항목 08의 측정).
	loLineWidthMinPx = 2.0
)

// loLineWidthMaxRatios는 픽스처별 글줄 잉크 폭 비율의 조정이다 — 없는 픽스처는
// loLineWidthMaxRatio(4%)를 쓴다. 조정은 측정 수치와 근거가 있을 때만 더한다.
//
// 02_text_subset: 측정 최대 +5.0% (003_inline의 code+링크 문단 — Iteration 2 항목 08).
// LibreOffice의 스크립트 간격은 한글↔라틴 **경계 수**에 비례하는데 비율 임계값은 줄
// 길이로 나눈 값이라, 짧은 줄에 경계가 몰리면(545px 줄에 mono 런 2회 전환 + 라틴
// 문장부호 경계 — 웹은 넣지 않는 간격 5곳) 01_text의 긴 본문(1125px, +3.3%)보다 비율이
// 커진다. 구조 갈라짐(줄 수·재접힘)은 0 오차의 줄 수 검사가 따로 잡고, 이 픽스처의 최소
// 여유 비율은 40.5%라 5%대 폭 오차로 줄이 늘 위험은 없다. 5.5%는 측정값에 01_text와
// 같은 폭의 여유를 더한 값이다 (3.3% 측정 → 4% 허용과 같은 태도).
var loLineWidthMaxRatios = map[string]float64{
	textSubsetFixtureName: 0.055,
	// 03_layout: 측정 최대 +5.8% (002_grid_cards의 두 칸 카드 문단 — Iteration 3 항목 05).
	// 02_text_subset과 같은 원인(스크립트 간격은 한글↔라틴 **경계 수**에 비례하고 비율은 줄
	// 길이로 나눈 값이다)이 배치에서 더 세게 나타난다: 트랙이 좁아 줄이 짧아지는데
	// (381px — 01_text 본문의 3분의 1) 그 짧은 줄에 `grid-column: span 2`의 라틴 런과
	// 인용 부호가 몰려 경계가 6곳이다. 이 줄의 박스는 두 칸 트랙(558px)이라 여유 비율이
	// 46%이므로 5.8%의 폭 오차로 줄이 접힐 위험은 없다 (구조 갈라짐은 0 오차의 줄 수 검사가
	// 따로 잡는다). 6.5%는 측정값에 앞 픽스처들과 같은 폭의 여유를 더한 값이다.
	layoutFixtureName: 0.065,
}

// inkBandSkipSlides는 잉크 띠 판정을 **적용할 수 없는** 슬라이드다 (`<픽스처>/<슬라이드 id>`).
//
// InkBands는 "영역에서 가장 흔한 색이 배경이고 잉크는 글자다"에 기대므로, 저자가 배경을 칠한
// 슬라이드에서는 그 사각형이 하나의 거대한 띠가 되어 글줄 판정 자체가 성립하지 않는다 —
// 값을 완화하는 문제가 아니라 **측정 방법의 전제가 깨지는** 문제다 (웹 띠 1152×40px = 칠한
// 띠, LibreOffice 띠 737×22px = 글자. 폭 차 −36%).
//
// 판정 범위를 박스별로 좁혀도 마찬가지다: 칠한 것은 부모 `<row>`이고 그 사각형이 자식 박스를
// 덮는다. 이 슬라이드의 배치는 좌표로 0 오차 대조된다
// (TestTextFixtureCoordinatesMatchGeometry — xfrm == 측정 EMU).
var inkBandSkipSlides = map[string]bool{
	// 007_style_appearance: `style="background: #0F65F9"`가 웹에만 적용된다 (§7.9의 계약).
	"03_layout/style-appearance-warning": true,
}

// 두 렌더 엔진이 같은 자리에서 줄을 나누는가 — Iteration 1의 리뷰 포인트를 수치로 판정한다
// (PRD §13-1). 지오메트리 패스의 줄 박스(geom.Box.Lines)를 슬라이드 단위로 합쳐 기대 띠를
// 만들고, 웹 스크린샷과 LibreOffice 렌더 각각의 잉크 띠와 맞댄다. Iteration 2부터는 인라인
// 서식이 낀 줄(굵은 런은 글자 폭이 다르다)과 리스트·pre까지 같은 계약을 받는다.
//
// 요소 박스 안에서 재지 않고 슬라이드 전체에서 재는 이유: 박스는 잰 줄 수만큼의 높이로
// 고정되므로, 다른 엔진이 줄을 하나 더 만들면 그 줄은 **박스 밖**에 그려진다 — 박스 안만
// 보면 늘어난 줄이 측정에서 빠져 갈라짐이 통과한다 (01_text의 본문이 처음에 정확히
// 그렇게 조용히 통과했다).
func TestTextFixtureLineBreaksMatchLibreOffice(t *testing.T) {
	RequireChromium(t)
	RequireSoffice(t)
	RequirePdftoppm(t)

	for _, name := range textFixtureNames {
		t.Run(name, func(t *testing.T) {
			widthMaxRatio := loLineWidthMaxRatio
			if override, ok := loLineWidthMaxRatios[name]; ok {
				widthMaxRatio = override
			}
			fixture := fixtureByName(t, name)
			result := fixture.Build(t)
			if result.Geometry == nil {
				t.Fatal("텍스트가 있는 덱의 빌드에 지오메트리가 없다")
			}
			stage := result.Deck.Stage
			stageRect := geom.Rect{W: float64(stage.Width), H: float64(stage.Height)}
			shots := RenderPPTX(t, fixture.Output(t, result, ".pptx"), image.Pt(stage.Width, stage.Height))

			browser := NewBrowser(t)
			browser.Load(t, filepath.Join(result.OutDir, web.IndexFile))

			minRatio := 1.0
			for _, box := range result.Geometry.Boxes {
				if box.LineCount() == 0 {
					continue
				}
				if ratio := box.Slack() / box.MaxLineWidth(); ratio < minRatio {
					minRatio = ratio
				}
			}
			t.Logf("최소 여유 비율 Slack/MaxLineWidth = %.2f%% — 이 덱이 견디는 글자 폭 오차", minRatio*100)

			for i, slide := range result.Deck.Slides {
				if inkBandSkipSlides[name+"/"+slide.ID] {
					t.Logf("%s: 잉크 띠 판정을 건너뛴다 — 저자가 칠한 배경이 단색 배경 가정을 깬다 (inkBandSkipSlides)", slide.Source)
					continue
				}
				lines := measuredLines(result.Geometry, i)
				if len(lines) == 0 {
					continue
				}

				// 웹 스크린샷은 잰 줄 박스와 같은 엔진의 그림이다 — 여기가 어긋나면 잉크 띠 측정
				// 자체가 틀린 것이므로, LibreOffice 판정보다 먼저 방법을 검증한다: 잰 줄마다
				// 웹 잉크가 있어야 하고(줄 → 띠), 잉크 띠마다 주인이 있어야 한다(띠 → 줄).
				webBands := InkBands(browser.ScreenshotElement(t, web.SlideIndexSelector(i)), stageRect)
				owners, ok := bandLineOwners(t, slide.Source, webBands, lines)
				if !ok {
					continue
				}

				loBands := InkBands(shots[i], stageRect)
				if len(loBands) != len(webBands) {
					t.Errorf("%s: 줄 수가 갈라진다 — 브라우저 %d줄, LibreOffice %d줄 (%s)",
						slide.Source, len(webBands), len(loBands), strings.Join(flatOwners(owners), ", "))
					continue
				}
				// 띠끼리의 비교는 웹 잉크 vs LibreOffice 잉크다 — 줄 박스(glyph box)는 잉크보다
				// 크므로(어센더·디센더의 빈 행) 줄 박스와 잉크의 중심을 맞대면 그 차이가 엔진
				// 오차처럼 보인다. 잰 줄은 띠의 주인 이름에만 쓴다.
				for j, wb := range webBands {
					lo := loBands[j]
					who := strings.Join(owners[j], "+")
					offset := (lo.Y + lo.H/2) - (wb.Y + wb.H/2)
					widthDelta := lo.W/wb.W - 1
					t.Logf("%s %s: 세로 중심 %+.1fpx, 잉크 폭 %+.1f%% (웹 %s, LibreOffice %s)",
						slide.Source, who, offset, widthDelta*100, wb, lo)
					if math.Abs(offset) > loLineOffsetMaxPx {
						t.Errorf("%s %s: 글줄이 %+.1fpx 밀렸다 (허용 ±%.0fpx)",
							slide.Source, who, offset, loLineOffsetMaxPx)
					}
					// 폭 판정은 비율과 절대 하한 중 큰 쪽이다 — 비율은 스크립트 간격(줄
					// 길이에 비례하는 오차)을, 하한은 AA 잡음(줄 길이와 무관한 ±1px)을 잰다.
					if allowed := math.Max(widthMaxRatio, loLineWidthMinPx/wb.W); math.Abs(widthDelta) > allowed {
						t.Errorf("%s %s: 글줄 잉크 폭이 %+.1f%% 다르다 (허용 ±%.1f%%)",
							slide.Source, who, widthDelta*100, allowed*100)
					}
				}
			}
		})
	}
}

// measuredLine은 지오메트리가 잰 글줄 하나와 그 주인 요소다.
type measuredLine struct {
	rect geom.Rect
	id   string
}

// measuredLines는 슬라이드의 모든 줄 박스를 위에서 아래 순서로 모은다.
func measuredLines(g *geom.Geometry, slide int) []measuredLine {
	var lines []measuredLine
	for _, box := range g.Boxes {
		if box.Slide != slide {
			continue
		}
		for _, line := range box.Lines {
			lines = append(lines, measuredLine{rect: line, id: box.ID})
		}
	}
	sort.Slice(lines, func(i, j int) bool { return lines[i].rect.Y < lines[j].rect.Y })
	return lines
}

// bandLineOwners는 잰 줄을 **웹 잉크 띠**에 배정하고, 띠마다 주인 요소 이름을 돌려준다.
// 배정이 성립하지 않으면(줄 자리에 잉크가 없거나, 주인 없는 잉크가 있으면) 측정 방법이 틀린
// 것이므로 실패로 보고하고 false를 돌려준다.
//
// 왜 줄 박스끼리 미리 합치지 않는가 (Iteration 3 항목 05에서 고친 것): 겹치는 줄 박스를 합치는
// 규칙은 세로 스택에서만 맞는다. 배치가 열리면 한 줄에 여러 박스가 앉고 그 박스들의 y가 조금씩
// 다른데(`<col gap="xs">` 안의 두 문단은 6px씩 엇갈린다), 줄 박스는 잉크보다 커서
// (어센더·디센더의 빈 행) 이웃한 두 줄의 박스가 서로 닿는다 — 합치기가 **전이적으로** 번져
// 004_child_attrs에서 세 줄이 한 띠가 되었고, 실제 그림은 세 띠였다 (방법 문제로 죽었다).
//
// 그래서 합치는 기준을 잰 rect가 아니라 **그림**에 둔다: 잉크 띠는 그 엔진이 실제로 그린
// 줄이고, 줄 박스는 세로로 가장 많이 겹치는 띠의 주인이 된다. 슬라이드 전체에서 재는 것은
// 그대로다 — LibreOffice가 줄을 하나 더 만들면 그 줄은 박스 밖에 그려지므로, 박스 안만 보면
// 갈라짐이 조용히 통과한다 (01_text의 본문이 처음에 정확히 그렇게 통과했다).
func bandLineOwners(t *testing.T, source string, bands []geom.Rect, lines []measuredLine) ([][]string, bool) {
	t.Helper()
	owners := make([][]string, len(bands))
	ok := true
	for _, line := range lines {
		best, overlap := -1, 0.0
		for i, band := range bands {
			top := math.Max(line.rect.Y, band.Y)
			bottom := math.Min(line.rect.Bottom(), band.Bottom())
			if bottom-top > overlap {
				best, overlap = i, bottom-top
			}
		}
		if best < 0 {
			t.Errorf("%s %s: 잰 줄 %s 자리에 웹 잉크가 없다 — 측정 방법이 틀렸다",
				source, line.id, line.rect)
			ok = false
			continue
		}
		owners[best] = appendOwner(owners[best], line.id)
	}
	for i, band := range bands {
		if len(owners[i]) == 0 {
			t.Errorf("%s: 웹 잉크 띠 %s에 주인이 없다 — 잰 줄이 없는 자리에 그림이 있다 (측정 방법이 틀렸다)",
				source, band)
			ok = false
		}
	}
	return owners, ok
}

// flatOwners는 진단 메시지용 요소 이름 목록이다 (띠 경계를 지운 평면 목록).
func flatOwners(owners [][]string) []string {
	var ids []string
	for _, band := range owners {
		ids = appendOwner(ids, band...)
	}
	return ids
}

// appendOwner는 중복 없이 요소 이름을 더한다 — 한 요소의 문자 노드 여러 개가 같은 줄에
// 있을 때(인라인이 쪼갠 경우) 진단 메시지에 같은 이름이 반복되지 않게 한다.
func appendOwner(ids []string, more ...string) []string {
	for _, id := range more {
		seen := false
		for _, have := range ids {
			if have == id {
				seen = true
				break
			}
		}
		if !seen {
			ids = append(ids, id)
		}
	}
	return ids
}
