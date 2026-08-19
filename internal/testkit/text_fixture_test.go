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

// 텍스트 픽스처들의 픽스처 수준 검증. PPTX 구조의 단위 단정은 render/pptx의 테스트가 이미
// 갖고 있다 — 여기서는 실제 픽스처의 산출물 전체를 놓고, 골든에 들어간 파트 목록·좌표·토큰
// 보존이 기대와 같은지, 그리고 두 렌더 엔진이 같은 자리에서 줄을 나누는지를 본다 (PRD §13-1).
const (
	// textFixtureName은 Iteration 1의 <text> 전용 픽스처다.
	textFixtureName = "01_text"
	// textSubsetFixtureName은 Iteration 2의 HTML 텍스트 서브셋 픽스처다 (블록·리스트·
	// 인라인 서식·pre — 항목 02/08이 활성화했다).
	textSubsetFixtureName = "02_text_subset"
)

// textFixtureNames는 픽스처 수준 검증(좌표 대조·잉크 띠·schemeClr)이 도는 텍스트 픽스처
// 전부다 — 새 텍스트 픽스처가 생기면 여기 더해 같은 계약을 받게 한다.
var textFixtureNames = []string{textFixtureName, textSubsetFixtureName}

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
			aligns := shapeAligns(t, result.Deck)
			if len(aligns) != len(result.Geometry.Boxes) {
				t.Fatalf("요소 %d개, 측정 박스 %d개", len(aligns), len(result.Geometry.Boxes))
			}
			for i, box := range result.Geometry.Boxes {
				align := aligns[i]
				if align == nil {
					// <li>는 측정되지만 자기 도형을 갖지 않는다 — 컨테이너의 도형 하나가
					// 좌표를 소유한다 (pptx.writeListShape).
					continue
				}
				part := fmt.Sprintf("ppt/slides/slide%d.xml", box.Slide+1)
				xml := string(parts[part])
				if xml == "" {
					t.Fatalf("파트 %s가 없다", part)
				}
				// 텍스트 박스는 잰 rect에 정렬 방향의 안전 여유를 더해 고정된다 (PRD §9.3,
				// pptx.TextBoxRect) — 같은 함수로 기대값을 만들어 조립기가 그 값을 썼는지 본다.
				rect := pptx.TextBoxRect(box.Rect, *align)
				x, y := pptx.Offset(rect)
				cx, cy := pptx.Extent(rect)
				xfrm := fmt.Sprintf("<a:off x=\"%d\" y=\"%d\"/>\n            <a:ext cx=\"%d\" cy=\"%d\"/>", x, y, cx, cy)
				if !strings.Contains(xml, xfrm) {
					t.Errorf("%s: %s의 xfrm이 측정값 %s(+여유)의 EMU와 다르다 (want %q)", part, box.ID, box.Rect, xfrm)
				}
			}
		})
	}
}

// shapeAligns는 지오메트리와 같은 문서 순서(geom.Refs — 인라인은 자손까지 건너뛴다)로,
// 측정된 요소마다 PPTX 조립기가 그 도형의 TextBoxRect에 쓰는 정렬을 모은다. 자기 도형을
// 갖지 않는 요소(<li> — 컨테이너가 도형 하나다)는 nil이다.
func shapeAligns(t *testing.T, deck *soml.Deck) []*soml.TextAlign {
	t.Helper()
	var aligns []*soml.TextAlign
	var walk func(slide *soml.Slide, node soml.Node)
	walk = func(slide *soml.Slide, node soml.Node) {
		elem, ok := node.(*soml.Element)
		if !ok {
			// 문자 데이터는 자기 박스를 갖지 않는다 (geom.collect와 같은 규칙).
			return
		}
		switch soml.KindOf(elem.Name) {
		case soml.KindInline:
			// 인라인은 측정 대상이 아니다 — 자손까지 건너뛴다 (geom.collect의 규칙 그대로.
			// 여기가 갈라지면 aligns와 Boxes의 자리가 어긋나 좌표가 엉뚱한 정렬과 만난다).
			return
		case soml.KindListContainer:
			// 리스트 도형의 안전 여유는 언제나 start 정렬로 적용된다 — li는 align 속성을
			// 받지 않아 확정 정렬이 항상 start다 (pptx.writeListShape의 결정).
			align := soml.AlignStart
			aligns = append(aligns, &align)
		case soml.KindListItem:
			aligns = append(aligns, nil)
		default:
			style := elem.TextStyle()
			if style == nil {
				t.Fatalf("%s: <%s>의 외관이 해석되지 않았다", slide.Source, elem.Name)
			}
			aligns = append(aligns, &style.Align)
		}
		for _, child := range elem.Children {
			walk(slide, child)
		}
	}
	for _, slide := range deck.Slides {
		for _, node := range slide.Content {
			walk(slide, node)
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
					// 폭 판정은 비율과 절대 하한 중 큰 쪽이다 — 비율은 스크립트 간격(줄
					// 길이에 비례하는 오차)을, 하한은 AA 잡음(줄 길이와 무관한 ±1px)을 잰다.
					if allowed := math.Max(widthMaxRatio, loLineWidthMinPx/wb.W); math.Abs(widthDelta) > allowed {
						t.Errorf("%s %s: 글줄 잉크 폭이 %+.1f%% 다르다 (허용 ±%.1f%%)",
							slide.Source, strings.Join(want.ids, "+"), widthDelta*100, allowed*100)
					}
				}
			}
		})
	}
}

// lineBand는 기대 잉크 띠 하나다 — 겹치는 줄 박스를 합친 것이라 요소가 여럿일 수 있다.
type lineBand struct {
	rect geom.Rect
	ids  []string
}

// expectedLineBands는 슬라이드의 모든 줄 박스를 위에서 아래로 합친 기대 잉크 띠다.
// 세로로 겹치는 줄 박스는 잉크에서도 한 띠가 되므로 합친다 — 인라인이 문자 데이터를 쪼갠
// 블록(`관측성 스택 도입 (<code>otel</code> + Grafana)`)은 같은 줄의 직계 문자 노드
// rect가 여럿이라, 합치는 규칙이 없으면 그 슬라이드에서 테스트가 방법 문제로 죽는다.
// 주의: 줄 박스는 **직계 문자 노드**에서만 나오므로 (geom.measureScript) 인라인이 낀 줄의
// 기대 띠 폭은 부분 폭이다 — 그래서 기대 띠는 줄 수와 요소 이름에만 쓴다.
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
				last.ids = appendOwner(last.ids, band.ids...)
				continue
			}
		}
		merged = append(merged, band)
	}
	return merged
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

// bandOwners는 진단 메시지용 요소 이름 목록이다.
func bandOwners(bands []lineBand) string {
	var ids []string
	for _, band := range bands {
		ids = appendOwner(ids, band.ids...)
	}
	return strings.Join(ids, ", ")
}
