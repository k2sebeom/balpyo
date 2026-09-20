package testkit

import (
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/k2sebeom/balpyo/render/pptx"
	"github.com/k2sebeom/balpyo/render/web"
	"github.com/k2sebeom/balpyo/soml"
)

// 덱 크롬 픽스처(10_chrome) 수준의 검증이다 (ADR 0045). render/pptx·render/web·core의 단위
// 테스트가 조립기 하나의 산출을 보는 것과 달리, 여기서는 **픽스처 전체를 빌드한 결과**를 놓고
// 크롬의 세 계약이 실제 덱에서도 성립하는지 본다.
//
//  1. 제외는 **슬라이드 단위**다 — 표지(title)와 구역 표지(section-divider)에는 크롬 도형이
//     0개이고, 흐름 모드 두 장에는 푸터·번호가 각각 하나다.
//  2. 번호는 **덱 안의 위치**다 (제외된 장도 센다) — 그래서 2·4가 나오고, 그 숫자를 담은 것이
//     정적 런이 아니라 `a:fld type="slidenum"`이다 (뷰어가 장을 더하면 따라온다).
//  3. 크롬의 자리는 **잰 값**이다 — 골든 EMU가 Geometry.Chrome의 rect와 정확히 같다.
//
// 10_chrome을 fixtureNames에 넣지 않는 이유는 09_table과 같다: 그 표가 도는 검사들은 "측정
// 박스 하나 ↔ 도형 하나"의 등식(TestFixtureShapeCountExcludesNonShapeElements)을 전제하는데,
// 크롬 도형은 덱 트리의 요소가 아니라 설정·테마에서 생긴 것이라 대응하는 geom.Ref가 없다
// (그래서 좌표도 Boxes가 아니라 side table에 산다 — geom.Geometry.Chrome). 그 등식에 항을
// 더하는 것은 별개의 변경이고, 여기서는 크롬 고유의 계약을 크롬 고유의 자리에서 본다. 골든
// 비교와 픽셀 비교는 일반 러너(TestFixtures)가 이미 돈다.
const chromeFixtureName = "10_chrome"

// chromeEnabledFixtures는 크롬을 **켜는** 픽스처 전부다 — 아래 골든 불변 검사
// (TestChromeIsAbsentFromOtherFixtureGoldens)가 건너뛸 목록이고, 그래서 이 표에 이름을 더하는
// 것이 "이 덱의 골든에 크롬이 있어도 된다"는 선언이다 (목록을 손으로 적어 두면 크롬이 엉뚱한
// 픽스처에 새어 나간 것을 그 검사가 계속 잡는다).
//
//   - 10_chrome: 크롬 그 자체를 보는 픽스처 (ADR 0045) — 위 표들이 전수로 판정한다.
//   - 15_local_theme: 프로젝트 로컬 테마가 부모의 크롬 선언을 **병합**하는 증인 (ADR 0054) —
//     corp가 footer의 skipLayouts를 줄이고 pageNumber는 minimal의 선언을 물려받는다.
var chromeEnabledFixtures = map[string]bool{
	chromeFixtureName: true,
	"15_local_theme":  true,
}

// chromeFixtureFooter는 픽스처 `balpyo.json`의 `footer` 값이다 — 모든 장이 같은 값을 받는다.
const chromeFixtureFooter = "2026 Q3 · 내부용"

// chromeFixtureSlides는 골든 슬라이드 파트 순서의 기대값이다 (파트 번호 = 덱 위치).
//
// 슬라이드를 **id로** 기억하는 이유는 다른 픽스처 검사와 같다: 파트 번호로만 표를 짜면
// 슬라이드를 하나 끼워 넣는 순간 표가 엉뚱한 슬라이드를 가리킨다. 번호 기대값을 손으로 적는
// 이유는 그것이 이 항목의 핵심 사실이라는 것이다 — 3번 장이 크롬을 달지 않아도 4번 장은 4다.
var chromeFixtureSlides = []struct {
	id    string
	items []chromeFixtureItem
}{
	{id: "chrome-cover"},
	{id: "chrome-body", items: []chromeFixtureItem{
		{kind: "footer", algn: "l", text: chromeFixtureFooter},
		{kind: "pageNumber", algn: "r", text: "2", field: true},
	}},
	{id: "chrome-divider"},
	{id: "chrome-after-divider", items: []chromeFixtureItem{
		{kind: "footer", algn: "l", text: chromeFixtureFooter},
		{kind: "pageNumber", algn: "r", text: "4", field: true},
	}},
}

// chromeFixtureItem은 크롬 도형 하나의 기대값이다.
type chromeFixtureItem struct {
	kind  string
	algn  string
	text  string
	field bool // true면 `a:fld type="slidenum"`, false면 보통 런
}

// 크롬은 제외 규칙이 고른 슬라이드에만, 골든에 적힌 그 모양으로 나간다.
//
// 골든 XML만 읽으므로 브라우저 없이 돈다 — 크롬의 **구성**(어느 장에 무엇이 몇 개)은 좌표를
// 몰라도 판정할 수 있고, 그래야 이 계약이 브라우저가 없는 환경에서도 회귀 테스트로 남는다.
func TestChromeFixtureIsOnTheIncludedSlidesOnly(t *testing.T) {
	for i, want := range chromeFixtureSlides {
		part := fmt.Sprintf("slide%d.xml", i+1)
		data := chromeFixtureGoldenSlide(t, part)
		if id := goldenSlideID(t, string(data)); id != want.id {
			t.Fatalf("%s의 슬라이드 id = %q, want %q — 표가 엉뚱한 장을 가리킨다", part, id, want.id)
		}
		got := decodeChromeShapes(t, part, data)
		if len(got) != len(want.items) {
			t.Fatalf("%s(%s)의 크롬 도형 %d개, want %d개", part, want.id, len(got), len(want.items))
		}
		for j, item := range want.items {
			sp := got[j]
			// 이름이 순서다 — 푸터가 먼저, 번호가 다음이다 (core.resolveChrome의 어휘 순서).
			if name := "chrome " + item.kind; sp.Name.Value != name {
				t.Errorf("%s의 크롬[%d] 이름 = %q, want %q", part, j, sp.Name.Value, name)
			}
			// 플레이스홀더가 아니다 — 제외가 슬라이드 단위인데 흐름 모드는 레이아웃 파트
			// 하나를 공유한다 (ADR 0045).
			if sp.Ph != nil {
				t.Errorf("%s의 크롬[%d]이 플레이스홀더로 나갔다", part, j)
			}
			if sp.Body.Paragraphs[0].Properties.Align != item.algn {
				t.Errorf("%s의 크롬[%d] 문단 정렬 = %q, want %q",
					part, j, sp.Body.Paragraphs[0].Properties.Align, item.algn)
			}
			// 웹이 한 줄로 두기로 한 글자를 PPTX가 접으면 띠를 넘쳐 본문 위로 올라간다.
			if sp.Body.Properties.Wrap != "none" || sp.Body.Properties.Anchor != "ctr" {
				t.Errorf("%s의 크롬[%d] bodyPr = (wrap=%q, anchor=%q), want (none, ctr)",
					part, j, sp.Body.Properties.Wrap, sp.Body.Properties.Anchor)
			}
		}
	}
}

// 번호는 **필드**이고 그 값은 덱 안의 위치다 — 정적 런으로 내면 뷰어에서 장을 더하거나 순서를
// 바꾼 순간 번호가 거짓말을 한다 (P2.2의 REVIEW 항목).
//
// 두 가지를 함께 본다. (1) 담는 요소가 `a:fld type="slidenum"`이라는 것 — 자동 갱신을 주는 것이
// 이것뿐이다. (2) 캐시된 숫자(`a:t`)가 덱 위치라는 것 — 필드를 계산하지 않는 소비자(썸네일·
// 변환기)도 맞는 숫자를 보여야 하고, 뷰어가 계산한 값과 갈라지지 않으려면 그 값이 **뷰어가
// 세는 방식과 같은 규칙**(제외된 장도 센다)으로 나와야 한다. 3번 장에 번호가 없는데도 4번 장이
// 4인 것이 그 증거다.
func TestChromeFixtureNumberIsASlidenumFieldOfDeckPosition(t *testing.T) {
	for i, want := range chromeFixtureSlides {
		part := fmt.Sprintf("slide%d.xml", i+1)
		shapes := decodeChromeShapes(t, part, chromeFixtureGoldenSlide(t, part))
		for j, item := range want.items {
			para := shapes[j].Body.Paragraphs[0]
			if !item.field {
				if len(para.Fields) != 0 {
					t.Errorf("%s의 크롬[%d](%s)이 필드로 나갔다", part, j, item.kind)
				}
				if len(para.Runs) != 1 || para.Runs[0].Text != item.text {
					t.Errorf("%s의 크롬[%d] 런 = %+v, want 글자 %q", part, j, para.Runs, item.text)
				}
				continue
			}
			if len(para.Runs) != 0 {
				t.Errorf("%s의 번호가 보통 런으로 나갔다 — 뷰어가 갱신하지 않는다", part)
			}
			if len(para.Fields) != 1 {
				t.Fatalf("%s의 번호 필드 %d개, want 1개", part, len(para.Fields))
			}
			field := para.Fields[0]
			if field.Type != "slidenum" {
				t.Errorf("%s의 번호 필드 종류 = %q, want slidenum", part, field.Type)
			}
			if field.Text != item.text {
				t.Errorf("%s의 캐시된 번호 = %q, want %q (덱 위치 — 제외된 장도 센다)",
					part, field.Text, item.text)
			}
			// 필드 id는 결정적이어야 한다 (매번 새 GUID면 같은 덱의 바이트가 달라진다).
			if !strings.HasPrefix(field.ID, "{") || !strings.HasSuffix(field.ID, "}") {
				t.Errorf("%s의 필드 id가 GUID 모양이 아니다: %q", part, field.ID)
			}
		}
	}
	// 웹도 같은 숫자를 찍는다 — 두 타깃이 한 확정 모델을 읽는다는 사실의 나머지 절반이다.
	// (웹은 필드를 가질 수 없으므로 글자가 곧 값이고, 그래서 값 층이 센 숫자가 맞아야 한다.)
	html, err := os.ReadFile(filepath.Join(fixtureByName(t, chromeFixtureName).GoldenDir(), web.IndexFile))
	if err != nil {
		t.Fatalf("골든이 없다 (`make golden`으로 만든다): %v", err)
	}
	numbers := chromeDivs(string(html), string(soml.ChromePageNumber))
	want := []string{"2", "4"}
	if !equalStrings(numbers, want) {
		t.Errorf("웹 골든의 번호 = %v, want %v", numbers, want)
	}
}

// 크롬의 자리는 **잰 값**이다: 골든 EMU가 Geometry.Chrome의 rect와 정확히 같아야 한다.
//
// 이 검사가 필요한 이유는 픽셀 비교가 크롬의 자리를 판정하지 못한다는 것이다 — 띠 안의 차이는
// 슬라이드의 0.15%라 본문 글리프 잔차에 묻힌다 (fixtureDiffOptions의 10_chrome 주석). 그래서
// 자리는 0 오차로 여기서 못 박는다. 텍스트 박스와 달리 안전 여유(pptx.TextBoxRect)를 더하지
// 않는다: 크롬은 `wrap="none"`이라 접힐 줄이 없고, 막을 줄바꿈이 없으면 여유도 뜻이 없다.
func TestChromeFixtureShapesMatchGeometry(t *testing.T) {
	RequireChromium(t) // 지오메트리 패스가 브라우저를 띄운다 — 없으면 건너뛴다(로컬).
	fixture := fixtureByName(t, chromeFixtureName)
	result := fixture.Build(t)
	if result.Geometry == nil {
		t.Fatal("크롬이 켜진 덱의 빌드에 지오메트리가 없다 — 크롬도 좌표를 쓴다")
	}
	data, err := os.ReadFile(fixture.Output(t, result, ".pptx"))
	if err != nil {
		t.Fatal(err)
	}
	parts := PPTXParts(t, data)

	measured := 0
	for _, slide := range result.Deck.Slides {
		part := fmt.Sprintf("ppt/slides/slide%d.xml", slide.Index+1)
		shapes := decodeChromeShapes(t, part, parts[part])
		if len(shapes) != len(slide.Chrome) {
			t.Fatalf("%s의 크롬 도형 %d개, 확정 모델 %d개", part, len(shapes), len(slide.Chrome))
		}
		for j, item := range slide.Chrome {
			rect, ok := result.Geometry.ChromeRect(slide.Index, item.Kind)
			if !ok {
				t.Fatalf("%s: 크롬 %s를 재지 않았다", part, item.Kind)
			}
			measured++
			x, y := pptx.Offset(rect)
			cx, cy := pptx.Extent(rect)
			if shapes[j].Transform.Offset.X != x || shapes[j].Transform.Offset.Y != y {
				t.Errorf("%s의 크롬 %s off = (%d, %d), want 측정값 %s의 EMU (%d, %d)",
					part, item.Kind, shapes[j].Transform.Offset.X, shapes[j].Transform.Offset.Y, rect, x, y)
			}
			if shapes[j].Transform.Extent.CX != cx || shapes[j].Transform.Extent.CY != cy {
				t.Errorf("%s의 크롬 %s ext = (%d, %d), want 측정값 %s의 EMU (%d, %d)",
					part, item.Kind, shapes[j].Transform.Extent.CX, shapes[j].Transform.Extent.CY, rect, cx, cy)
			}
			// 크롬은 아래 띠에 산다 — 콘텐츠 박스를 침범하면 본문과 겹친다 (geom.Diagnose가
			// 경고를 내는 자리이고, 픽스처는 경고 없이 빌드되어야 한다).
			if rect.Y < float64(result.Deck.Stage.Height-web.StagePadding) {
				t.Errorf("%s의 크롬 %s가 띠 위로 올라왔다: %s", part, item.Kind, rect)
			}
		}
	}
	if measured != len(result.Geometry.Chrome) {
		t.Errorf("도형과 대조한 크롬 %d개, 잰 크롬 %d개 — 재고 쓰지 않은 것이 있다",
			measured, len(result.Geometry.Chrome))
	}
	if len(result.Diagnostics) != 0 {
		t.Errorf("크롬 픽스처가 진단을 냈다: %v", result.Diagnostics)
	}
	// 재는 자리와 쓰는 자리가 같은 좌표계라는 사실을 한 번 더 못 박는다 (geom.Rect → EMU).
	if _, ok := result.Geometry.ChromeRect(0, soml.ChromePageNumber); ok {
		t.Error("표지(layout=title)의 번호를 쟀다 — 제외된 장에는 크롬 요소가 없어야 한다")
	}
}

// chromeFixtureGoldenSlide는 골든 슬라이드 파트 하나를 읽는다.
func chromeFixtureGoldenSlide(t *testing.T, part string) []byte {
	t.Helper()
	path := filepath.Join(fixtureByName(t, chromeFixtureName).GoldenDir(), "pptx", "ppt", "slides", part)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("골든이 없다 (`make golden`으로 만든다): %v", err)
	}
	return data
}

// decodeChromeShapes는 슬라이드 파트에서 크롬 도형을 문서 순서로 읽는다.
//
// 정규식이 아니라 XML 디코더를 쓰는 이유: 보는 것이 **중첩된 구조**다 (도형 안의 문단, 문단
// 안의 런과 필드). 정규식으로는 "이 도형의 필드"와 "다음 도형의 필드"를 가를 수 없다.
//
// 크롬을 고르는 열쇠는 도형 이름이다 (`chrome <kind>`) — 이름은 조립기가 PowerPoint 선택
// 창을 위해 붙이는 값이고(pptx.writeChromeShapes), 그래서 이 검사가 이름 규칙도 함께 지킨다.
func decodeChromeShapes(t *testing.T, part string, data []byte) []goldenChromeSp {
	t.Helper()
	if len(data) == 0 {
		t.Fatalf("파트 %s가 없다", part)
	}
	var tree struct {
		Shapes []goldenChromeSp `xml:"cSld>spTree>sp"`
	}
	if err := xml.Unmarshal(data, &tree); err != nil {
		t.Fatalf("%s를 읽을 수 없다: %v", part, err)
	}
	var chrome []goldenChromeSp
	for _, sp := range tree.Shapes {
		if strings.HasPrefix(sp.Name.Value, "chrome ") {
			chrome = append(chrome, sp)
		}
	}
	// 크롬은 콘텐츠 **뒤**다 (spTree 순서 = z-순서): 앞선 도형 중에 크롬이 있으면 순서가 깨진 것이다.
	for i, sp := range tree.Shapes {
		if !strings.HasPrefix(sp.Name.Value, "chrome ") {
			continue
		}
		if i+len(chrome) != len(tree.Shapes) {
			t.Errorf("%s: 크롬이 콘텐츠 사이에 있다 (도형 %d/%d) — 본문에 가려질 수 있다",
				part, i+1, len(tree.Shapes))
		}
		break
	}
	return chrome
}

// goldenChromeSp는 슬라이드 파트의 `p:sp` 하나에서 크롬 검사가 읽는 것만 담는다.
type goldenChromeSp struct {
	Name      goldenSpName `xml:"nvSpPr>cNvPr"`
	Ph        *struct{}    `xml:"nvSpPr>nvPr>ph"`
	Transform goldenXfrm   `xml:"spPr>xfrm"`
	Body      goldenTxBody `xml:"txBody"`
}

type goldenSpName struct {
	Value string `xml:"name,attr"`
}

type goldenXfrm struct {
	Offset struct {
		X int64 `xml:"x,attr"`
		Y int64 `xml:"y,attr"`
	} `xml:"off"`
	Extent struct {
		CX int64 `xml:"cx,attr"`
		CY int64 `xml:"cy,attr"`
	} `xml:"ext"`
}

type goldenTxBody struct {
	Properties struct {
		Wrap   string `xml:"wrap,attr"`
		Anchor string `xml:"anchor,attr"`
	} `xml:"bodyPr"`
	Paragraphs []goldenParagraph `xml:"p"`
}

type goldenParagraph struct {
	Properties struct {
		Align string `xml:"algn,attr"`
	} `xml:"pPr"`
	Runs []struct {
		Text string `xml:"t"`
	} `xml:"r"`
	Fields []struct {
		ID   string `xml:"id,attr"`
		Type string `xml:"type,attr"`
		Text string `xml:"t"`
	} `xml:"fld"`
}

// chromeDivs는 웹 골든에서 한 종류의 크롬 글자를 문서 순서로 뽑는다.
func chromeDivs(html, kind string) []string {
	var texts []string
	marker := fmt.Sprintf("%s=%q", web.ChromeKindAttr, kind)
	for rest := html; ; {
		i := strings.Index(rest, marker)
		if i < 0 {
			return texts
		}
		rest = rest[i:]
		open := strings.Index(rest, ">")
		end := strings.Index(rest, "</div>")
		if open < 0 || end < open {
			return texts
		}
		texts = append(texts, rest[open+1:end])
		rest = rest[end:]
	}
}

// 크롬을 쓰지 않는 픽스처의 산출물에는 크롬이 **한 조각도** 없어야 한다 (골든 불변).
//
// 이 검사가 있는 이유: 크롬을 마스터의 `p:hf`나 레이아웃 파트로 냈다면 크롬을 켜지 않은 덱의
// 바이트까지 움직였을 것이고, 그 변화는 골든 diff에서 "왜 움직였는가"를 설명하지 못한다
// (ADR 0045가 `p:hf`를 내지 않기로 한 근거). 기능 게이트가 실제로 닫혀 있는지는 픽스처
// 골든에서만 확인할 수 있다.
func TestChromeIsAbsentFromOtherFixtureGoldens(t *testing.T) {
	for _, fixture := range Fixtures(t) {
		if chromeEnabledFixtures[fixture.Name] {
			continue
		}
		err := filepath.Walk(fixture.GoldenDir(), func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() || filepath.Ext(path) == ".png" || filepath.Ext(path) == ".jpg" {
				return err
			}
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			for _, needle := range []string{"slidenum", web.ChromeClass, web.ChromeKindAttr} {
				if strings.Contains(string(data), needle) {
					t.Errorf("%s의 %s에 %q가 있다 — 크롬을 켜지 않은 덱의 바이트가 움직였다",
						fixture.Name, filepath.Base(path), needle)
				}
			}
			return nil
		})
		if err != nil && !os.IsNotExist(err) {
			t.Fatalf("%s의 골든을 훑을 수 없다: %v", fixture.Name, err)
		}
	}
}
