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

	"github.com/k2sebeom/balpyo/render/geom"
	"github.com/k2sebeom/balpyo/render/pptx"
	"github.com/k2sebeom/balpyo/render/web"
	"github.com/k2sebeom/balpyo/soml"
)

// 01_text 픽스처의 픽스처 수준 검증 (Iteration 1). PPTX 구조의 단위 단정은 render/pptx의
// 테스트가 이미 갖고 있다 — 여기서는 실제 픽스처의 산출물 전체를 놓고, 골든에 들어간 파트
// 목록·좌표·토큰 보존이 기대와 같은지, 그리고 두 렌더 엔진이 같은 자리에서 줄을 나누는지를
// 본다 (PRD §13-1).
const textFixtureName = "01_text"

// textFixture는 픽스처 러너와 같은 수집 경로로 01_text를 집는다 — 경로를 직접 조립하면
// 러너가 못 보는 픽스처를 여기만 보는 상태가 생길 수 있다.
func textFixture(t *testing.T) Fixture {
	t.Helper()
	for _, fixture := range Fixtures(t) {
		if fixture.Name == textFixtureName {
			return fixture
		}
	}
	t.Fatalf("픽스처 %s가 수집되지 않았다", textFixtureName)
	return Fixture{}
}

// 골든에 들어간 파트 목록을 통째로 단정한다. 파트별 XML은 골든 비교가 보지만, "무엇이
// 있어야 하고 무엇이 없어야 하는가"는 목록이 말한다 — 특히 노트 파트는 노트가 있는
// 슬라이드(1·3)에만 있어야 하고, 노트 없는 슬라이드 2의 notesSlide2가 생기면 안 된다.
func TestTextFixtureGoldenPartList(t *testing.T) {
	want := []string{
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
	}

	var got []string
	goldenDir := textFixture(t).GoldenDir()
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
		t.Fatal(err)
	}

	// diagnostics.txt는 경고가 있을 때만 생긴다 — 이 픽스처는 경고 없이 빌드되어야 한다.
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
}

// 슬라이드 파트의 색은 전부 schemeClr 참조여야 한다. 픽스처의 색은 모두 토큰이므로 슬라이드에
// 리터럴 srgbClr가 하나라도 있으면 토큰이 어딘가에서 RGB로 눕혀진 것이다 — 그 도형은 테마 색
// 편집을 따라가지 못한다 (제품 핵심 계약, PRD §9.3 ②). 리터럴 RGB는 테마 파트의 토큰 선언
// 한 곳에만 있어야 한다.
func TestTextFixtureGoldenSlidesKeepSchemeClr(t *testing.T) {
	goldenDir := textFixture(t).GoldenDir()
	for _, slide := range []string{"slide1.xml", "slide2.xml", "slide3.xml"} {
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
}

// 슬라이드 파트의 xfrm 좌표는 지오메트리 패스가 잰 값의 EMU 환산과 정확히 같아야 한다.
// 골든 XML만 보면 숫자가 그럴듯한지 알 수 없다 — 같은 빌드의 Result.Geometry와 맞대어
// "이 좌표가 그 측정에서 나왔다"를 못 박는다.
func TestTextFixtureCoordinatesMatchGeometry(t *testing.T) {
	RequireChromium(t) // 지오메트리 패스가 브라우저를 띄운다 — 없으면 건너뛴다(로컬).
	fixture := textFixture(t)
	result := fixture.Build(t)
	if result.Geometry == nil {
		t.Fatal("텍스트가 있는 덱의 빌드에 지오메트리가 없다")
	}

	data, err := os.ReadFile(fixture.Output(t, result, ".pptx"))
	if err != nil {
		t.Fatal(err)
	}
	parts := PPTXParts(t, data)
	aligns := textAligns(t, result.Deck)
	if len(aligns) != len(result.Geometry.Boxes) {
		t.Fatalf("요소 %d개, 측정 박스 %d개", len(aligns), len(result.Geometry.Boxes))
	}
	for i, box := range result.Geometry.Boxes {
		part := fmt.Sprintf("ppt/slides/slide%d.xml", box.Slide+1)
		xml := string(parts[part])
		if xml == "" {
			t.Fatalf("파트 %s가 없다", part)
		}
		// 텍스트 박스는 잰 rect에 정렬 방향의 안전 여유를 더해 고정된다 (PRD §9.3,
		// pptx.TextBoxRect) — 같은 함수로 기대값을 만들어 조립기가 그 값을 썼는지 본다.
		rect := pptx.TextBoxRect(box.Rect, aligns[i])
		x, y := pptx.Offset(rect)
		cx, cy := pptx.Extent(rect)
		xfrm := fmt.Sprintf("<a:off x=\"%d\" y=\"%d\"/>\n            <a:ext cx=\"%d\" cy=\"%d\"/>", x, y, cx, cy)
		if !strings.Contains(xml, xfrm) {
			t.Errorf("%s: %s의 xfrm이 측정값 %s(+여유)의 EMU와 다르다 (want %q)", part, box.ID, box.Rect, xfrm)
		}
	}
}

// textAligns는 덱의 텍스트 요소별 해석된 정렬을 지오메트리와 같은 문서 순서로 모은다
// (geom.Refs의 순회 규칙 — 이 픽스처의 어휘에서 요소는 슬라이드 직속 <text>뿐이다).
func textAligns(t *testing.T, deck *soml.Deck) []soml.TextAlign {
	t.Helper()
	var aligns []soml.TextAlign
	for _, slide := range deck.Slides {
		for _, node := range slide.Content {
			elem, ok := node.(*soml.Element)
			if !ok {
				continue
			}
			style := elem.TextStyle()
			if style == nil {
				t.Fatalf("%s: <%s>의 외관이 해석되지 않았다", slide.Source, elem.Name)
			}
			aligns = append(aligns, style.Align)
			if len(elem.Children) > 0 {
				for _, child := range elem.Children {
					if _, ok := child.(*soml.Element); ok {
						t.Fatalf("%s: 중첩 요소는 이 순회가 모른다 — geom.Refs와 같은 재귀로 바꿔야 한다", slide.Source)
					}
				}
			}
		}
	}
	return aligns
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
)

// 두 렌더 엔진이 같은 자리에서 줄을 나누는가 — Iteration 1의 리뷰 포인트를 수치로 판정한다
// (PRD §13-1). 지오메트리 패스의 줄 박스(geom.Box.Lines)를 슬라이드 단위로 합쳐 기대 띠를
// 만들고, 웹 스크린샷과 LibreOffice 렌더 각각의 잉크 띠와 맞댄다.
//
// 요소 박스 안에서 재지 않고 슬라이드 전체에서 재는 이유: 박스는 잰 줄 수만큼의 높이로
// 고정되므로, 다른 엔진이 줄을 하나 더 만들면 그 줄은 **박스 밖**에 그려진다 — 박스 안만
// 보면 늘어난 줄이 측정에서 빠져 갈라짐이 통과한다 (이 픽스처의 본문이 처음에 정확히
// 그렇게 조용히 통과했다).
func TestTextFixtureLineBreaksMatchLibreOffice(t *testing.T) {
	RequireChromium(t)
	RequireSoffice(t)
	RequirePdftoppm(t)

	fixture := textFixture(t)
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
		expected := expectedLineBands(result.Geometry, i)
		if len(expected) == 0 {
			continue
		}

		// 웹 스크린샷은 잰 줄 박스와 같은 엔진의 그림이다 — 여기가 어긋나면 잉크 띠 측정
		// 자체가 틀린 것이므로, LibreOffice 판정보다 먼저 방법을 검증한다.
		webBands := InkBands(browser.ScreenshotElement(t, web.SlideIndexSelector(i)), stageRect)
		if len(webBands) != len(expected) {
			t.Errorf("%s: 웹 잉크 띠 %d개, 줄 박스 %d개 — 측정 방법이 틀렸다",
				slide.Source, len(webBands), len(expected))
			continue
		}

		loBands := InkBands(shots[i], stageRect)
		if len(loBands) != len(expected) {
			t.Errorf("%s: 줄 수가 갈라진다 — 브라우저 %d줄, LibreOffice %d줄 (%s)",
				slide.Source, len(expected), len(loBands), bandOwners(expected))
			continue
		}
		// 띠끼리의 비교는 웹 잉크 vs LibreOffice 잉크다 — 줄 박스(glyph box)는 잉크보다
		// 크므로(어센더·디센더의 빈 행) 줄 박스 중심과 잉크 중심을 맞대면 그 차이가 엔진
		// 오차처럼 보인다. 기대 띠는 줄 수와 요소 이름에만 쓴다.
		for j, want := range expected {
			lo, wb := loBands[j], webBands[j]
			offset := (lo.Y + lo.H/2) - (wb.Y + wb.H/2)
			widthDelta := lo.W/wb.W - 1
			t.Logf("%s %s: 세로 중심 %+.1fpx, 잉크 폭 %+.1f%% (웹 %s, LibreOffice %s)",
				slide.Source, strings.Join(want.ids, "+"), offset, widthDelta*100, wb, lo)
			if math.Abs(offset) > loLineOffsetMaxPx {
				t.Errorf("%s %s: 글줄이 %+.1fpx 밀렸다 (허용 ±%.0fpx)",
					slide.Source, strings.Join(want.ids, "+"), offset, loLineOffsetMaxPx)
			}
			if math.Abs(widthDelta) > loLineWidthMaxRatio {
				t.Errorf("%s %s: 글줄 잉크 폭이 %+.1f%% 다르다 (허용 ±%.1f%%)",
					slide.Source, strings.Join(want.ids, "+"), widthDelta*100, loLineWidthMaxRatio*100)
			}
		}
	}
}

// lineBand는 기대 잉크 띠 하나다 — 겹치는 줄 박스를 합친 것이라 요소가 여럿일 수 있다.
type lineBand struct {
	rect geom.Rect
	ids  []string
}

// expectedLineBands는 슬라이드의 모든 줄 박스를 위에서 아래로 합친 기대 잉크 띠다.
// 세로로 겹치는 줄 박스는 잉크에서도 한 띠가 되므로 합친다 (이 픽스처에서는 일어나지 않지만,
// 합치는 규칙이 없으면 겹침이 생기는 순간 테스트가 방법 문제로 죽는다).
func expectedLineBands(g *geom.Geometry, slide int) []lineBand {
	var bands []lineBand
	for _, box := range g.Boxes {
		if box.Slide != slide {
			continue
		}
		for _, line := range box.Lines {
			bands = append(bands, lineBand{rect: line, ids: []string{box.ID}})
		}
	}
	sort.Slice(bands, func(i, j int) bool { return bands[i].rect.Y < bands[j].rect.Y })
	var merged []lineBand
	for _, band := range bands {
		if len(merged) > 0 {
			last := &merged[len(merged)-1]
			if band.rect.Y < last.rect.Bottom() {
				right := math.Max(last.rect.Right(), band.rect.Right())
				bottom := math.Max(last.rect.Bottom(), band.rect.Bottom())
				last.rect.X = math.Min(last.rect.X, band.rect.X)
				last.rect.W = right - last.rect.X
				last.rect.H = bottom - last.rect.Y
				last.ids = append(last.ids, band.ids...)
				continue
			}
		}
		merged = append(merged, band)
	}
	return merged
}

// bandOwners는 진단 메시지용 요소 이름 목록이다.
func bandOwners(bands []lineBand) string {
	var ids []string
	for _, band := range bands {
		ids = append(ids, band.ids...)
	}
	return strings.Join(ids, ", ")
}
