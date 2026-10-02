package testkit

import (
	"fmt"
	"image"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/k2sebeom/balpyo/internal/preview"
	"github.com/k2sebeom/balpyo/render/geom"
	"github.com/k2sebeom/balpyo/render/pptx"
	"github.com/k2sebeom/balpyo/render/web"
	"github.com/k2sebeom/balpyo/soml"
)

// 픽스처 수준 검증. PPTX 구조의 단위 단정은 render/pptx의 테스트가 이미 갖고 있다 —
// 여기서는 실제 픽스처의 산출물 전체를 놓고, 골든에 들어간 파트 목록·좌표·토큰 보존이
// 기대와 같은지, 그리고 두 렌더 엔진이 같은 자리에서 줄을 나누는지를 본다 (PRD §13-1).
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
	// shapeFixtureName은 Iteration 4의 도형 픽스처다 (프리셋 12종 + 외관 + 도형 안 텍스트
	// — 항목 04/06이 활성화했다). 여기서 처음으로 **면이 그려지므로** 위 셋에서 성립했던
	// 두 전제가 깨진다: 슬라이드 파트에 리터럴 `srgbClr`가 0건이라는 것(리터럴 색이 어휘에
	// 들어왔다 — fixtureLiteralColors), 그리고 "가장 흔한 색이 배경"이라는 잉크 띠의
	// 전제(칠한 도형이 큰 면을 덮는다 — inkBandSkipSlides).
	shapeFixtureName = "04_shape"
	// imageFixtureName은 Iteration 5의 그림 픽스처다 (`fit` 2종 + 외관 + 포맷·파트 재사용 +
	// variant — 항목 05/05가 활성화했다). 여기서 처음으로 **골든에 바이너리가 들어간다**:
	// 웹 산출 에셋(`assets/*`)과 PPTX 미디어 파트(`ppt/media/*`)가 같은 그림을 두 번 담는다
	// (두 산출물이 각자 온전해야 하므로 정상이다 — 에셋 파일이 작아야 하는 이유가 여기다).
	// 도형 픽스처가 깨뜨린 두 전제(리터럴 색·잉크 띠의 단색 배경)에서 이 픽스처는 앞쪽만
	// 되돌린다: 그림의 테두리도 토큰 색이라 슬라이드 파트에 `srgbClr`가 0건이지만
	// (fixtureLiteralColors에 행이 없다), 큰 색면은 도형보다 오히려 넓다 — inkBandSkipSlides가
	// 그 사실을 슬라이드마다 적는다.
	imageFixtureName = "05_image"
	// stackFixtureName은 Iteration 6의 겹침 픽스처다 (`<stack>` + 자식 배치 `at`·`x`·`y` +
	// `<slide layout="canvas">` — 항목 06이 활성화했다). 이 픽스처가 새로 시험하는 것은
	// **겹친 것이 두 타깃에서 같은 순서로 겹치는가**다: 문서 순서가 곧 z-순서라는 계약을
	// (PRD §13-3) spTree 순서 전수(TestStackFixtureZOrderIsDocumentOrder)와 겹친 영역 크롭
	// 비교(TestStackFixtureOverlapRegions)가 판정한다.
	//
	// 도형·그림 픽스처가 깬 두 전제(리터럴 색·잉크 띠의 단색 배경)에서 이 픽스처는 앞쪽을
	// 되돌린다: 모든 색이 토큰이라 슬라이드 파트에 리터럴 `srgbClr`가 0건이다
	// (fixtureLiteralColors에 행이 없다). 잉크 띠는 04·05와 같은 이유로 깨지지만 — 겹침
	// 슬라이드에는 **띠의 주인이 없다** (겹친 자리에 여러 도형이 포개진다) — 이번에는 겹치지
	// 않은 텍스트 슬라이드(010 content-explicit)가 하나 있어 그 슬라이드는 판정에 남긴다
	// (inkBandSkipSlides는 001~009만 적는다).
	//
	// 캔버스 슬라이드(008·009)가 파트를 늘리지 않는다는 것은 패키지 불변식이 본다 (AssertPackage —
	// 관계가 가리키지 않는 파트가 없다).
	stackFixtureName = "06_stack"
	// shapeLayoutFixtureName은 P2.3의 "도형 안 레이아웃 컨테이너" 픽스처다 (ADR 0049).
	// 새로 시험하는 것은 **도형 하나가 배경이 되고 자식이 형제 도형이 되는가**다: 04_shape이
	// 도형 안 텍스트가 `txBody`로 흡수되는 것을 못 박았고(ADR 0016), 이 픽스처는 그 흡수가
	// 도형마다 꺼질 수 있다는 것과 그때 자식의 좌표가 **잰 값 그대로** 나오는 것을 본다
	// (좌표 대조가 그 절반이다 — shapeRoles가 모드를 판정한다).
	//
	// 앞 픽스처들이 깨뜨린 전제를 그대로 물려받는다: 카드 배경이 큰 면을 덮으므로 잉크 띠의
	// 단색 배경 가정이 깨지고(inkBandSkipSlides), 색은 전부 토큰이라 슬라이드 파트에 리터럴
	// `srgbClr`는 0건이다 (fixtureLiteralColors에 행이 없다).
	shapeLayoutFixtureName = "11_shape_layout"
	// paletteFixtureName은 P4.1의 강조색 팔레트 픽스처다 (D12 — ADR 0050). 새로 시험하는
	// 것은 **여섯 토큰이 여섯 슬롯으로 간다**는 것이다: 팔레트가 하나였을 때 clrScheme의
	// accent2..6은 accent1에서 빌려 채운 자리였고(render/pptx.clrSchemeSlots), 그래서
	// `fill="accent3"`을 쓸 어휘가 있어도 색이 파랑이었다.
	//
	// 아래 검사들이 이 픽스처에서 보는 것은 **참조**다: 슬라이드 파트에 리터럴 `srgbClr`가
	// 0건이고(fixtureLiteralColors에 행이 없다) 여섯 accent 참조가 그대로 있다. 색 변형과
	// 그라디언트까지 토큰으로만 쓴 이유가 그것이다 — 변형 코드가 토큰 이름을 보고 분기하지
	// 않는다는 것은 accent1 밖에서도 같은 XML이 나올 때 사실이 된다 (ADR 0033).
	//
	// 앞 픽스처들이 깬 전제 하나를 물려받는다: 색 견본이 큰 면을 덮으므로 잉크 띠의 단색
	// 배경 가정이 깨진다 (inkBandSkipSlides에 두 슬라이드가 있다).
	paletteFixtureName = "12_palette"
	// codeFixtureName은 P2.4의 코드 하이라이팅 픽스처다 (D15·D16 — ADR 0051). 새로 시험하는
	// 것은 **토큰 스트림이 런이 된다**는 것이다: `lang`이 붙은 `<pre>`의 문자 데이터가 값 층에서
	// 합성 인라인 런으로 컴파일되므로 슬라이드 파트의 `a:r`가 여럿으로 늘고, 그 색은 전부
	// `schemeClr` 참조다 (fixtureLiteralColors에 행이 없다 — 하이라이팅 색이 리터럴 RGB로
	// 나가면 PowerPoint의 테마 색 편집이 코드 블록만 두고 간다).
	//
	// 탭도 여기서 지난다 (D15): 001은 gofmt 출력 그대로라 들여쓰기가 탭인데, 산출물에는 탭이
	// 없다 — 값 층이 테마 폭으로 칼럼을 세어 편다. 앞 픽스처의 전제는 그대로 물려받는다
	// (코드 블록은 텍스트이므로 잉크 띠의 단색 배경 가정을 깨지 않는다).
	codeFixtureName = "13_code"
	// themeFixtureName은 P4.2의 테마 선택 픽스처다 (ADR 0052). 새로 시험하는 것은 **덱이
	// minimal이 아닌 테마로 끝까지 간다**는 것이다: 앞 픽스처 열셋은 모두 기본 테마로 빌드되므로,
	// 값 층이 테마를 하드코딩으로 되돌리거나(삭제한 core/theme.go) 렌더러가 특정 테마의 수치를
	// 가정하면 그 열셋은 전부 초록인 채 남는다. `balpyo.json`의 `"theme": "ink"` 한 줄이
	// 산출물의 모든 색·서체·레이아웃 CSS를 바꾸는 것을 골든이 붙든다.
	//
	// 어두운 테마를 고른 이유: 밝은 테마끼리는 골든이 갈라져도 눈으로 틀렸음을 알기 어렵지만,
	// 바탕이 #15171B인 산출물에 minimal의 흰 바탕이 한 군데라도 남으면 리뷰에서 즉시 보인다.
	//
	// 앞 픽스처가 깬 전제를 **하나도** 물려받지 않는다: 텍스트만 쓰고 칠한 도형·그림이 없으므로
	// 잉크 띠의 단색 배경 가정이 성립하고(inkBandSkipSlides에 행이 없다 — 어두운 바탕도
	// "영역에서 가장 흔한 색"이라는 점에서 흰 바탕과 다르지 않다), 색은 전부 토큰이라 슬라이드
	// 파트에 리터럴 `srgbClr`가 0건이다 (fixtureLiteralColors에 행이 없다). 즉 테마를 바꿔도
	// 계약이 약해지지 않는다는 것까지 이 픽스처가 말한다.
	themeFixtureName = "14_theme"
)

// fixtureNames는 픽스처 수준 검증(파트 목록·색 계약·좌표 대조·도형 수 등식·잉크 띠)이 도는
// 픽스처 전부다 — 새 픽스처가 생기면 여기 더해 같은 계약을 받게 한다.
//
// 이 표와 아래 검사들의 이름에 "text"가 없는 이유(Iteration 4 항목 06의 판단): 검사가 재는
// 것은 텍스트가 아니라 **픽스처 하나의 산출물 전체**다. 이름이 `textFixtureNames`로 남아
// 있으면 도형 픽스처를 넣는 것이 표의 뜻을 어기는 일처럼 보이고, 그러면 다음 사람이 같은
// 계약을 받을 픽스처를 두 번째 표에 적는다.
var fixtureNames = []string{textFixtureName, textSubsetFixtureName, layoutFixtureName, shapeFixtureName,
	imageFixtureName, stackFixtureName, shapeLayoutFixtureName, paletteFixtureName, codeFixtureName,
	themeFixtureName}

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

// goldenSlideIDPattern은 조립기가 슬라이드 id를 남기는 자리다 (`<p:cSld name="…">`).
var goldenSlideIDPattern = regexp.MustCompile(`<p:cSld name="([^"]*)"`)

// goldenSlideID는 골든 파트에서 슬라이드 id를 읽는다 — 표의 열쇠를 파트 번호가 아니라
// 저자가 붙인 id로 두기 위해서다 (슬라이드를 끼워 넣어도 표가 따라 움직인다).
func goldenSlideID(t *testing.T, xml string) string {
	t.Helper()
	m := goldenSlideIDPattern.FindStringSubmatch(xml)
	if m == nil {
		t.Fatal("골든 슬라이드 파트에 cSld@name이 없다 — 표의 열쇠를 만들 수 없다")
	}
	return m[1]
}

// cropImage는 이미지에서 사각형 영역을 새 이미지로 잘라 온다 (이미지 경계로 자른다).
//
// 두 렌더의 같은 영역을 잘라야 하므로 좌표는 반올림 없이 바닥·천장으로 맞춘다 — 한쪽만
// 반올림하면 크롭이 1px 어긋나 그 차이가 전부 오차로 계산된다.
func cropImage(img image.Image, r geom.Rect) image.Image {
	b := img.Bounds()
	x0, y0 := max(int(math.Floor(r.X)), 0), max(int(math.Floor(r.Y)), 0)
	x1, y1 := min(int(math.Ceil(r.Right())), b.Dx()), min(int(math.Ceil(r.Bottom())), b.Dy())
	out := image.NewRGBA(image.Rect(0, 0, max(x1-x0, 0), max(y1-y0, 0)))
	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			out.SetRGBA(x-x0, y-y0, preview.RGB8(img.At(b.Min.X+x, b.Min.Y+y)))
		}
	}
	return out
}

// inShapeBandInset은 도형 안 줄 세기에서 잰 rect를 안으로 줄이는 폭이다 (px).
//
// 테두리를 영역에서 빼기 위한 것이다: 테두리는 경로 중앙에 그려져 잰 rect의 변에 걸치므로
// (최대 굵기 3px), 영역에 들어오면 위아래 변이 각각 하나의 띠가 되어 줄 수에 더해진다.
// 4px은 가장 굵은 테두리(3px)에 안티앨리어싱 1px을 더한 값이다.
const inShapeBandInset = 4.0

// inShapeBandRegion은 도형 안 줄 세기를 판정할 영역이다 — 잰 rect가 아니라 **프리셋의 텍스트
// 영역**(ADR 0040)을 테두리·모서리만큼 더 줄인 사각형이고, 그 영역이 성립하지 않으면 false다.
//
// 텍스트 영역에서 재는 이유는 InkBands의 전제다: "영역에서 가장 흔한 색이 배경"인데, 채움이
// 잰 rect를 덮지 않는 프리셋(diamond·star·cloud…)에서는 배경 추정이 도형 밖 흰색을 골라
// **도형 자체가 하나의 거대한 띠**가 된다. 프리셋의 텍스트 영역은 실루엣 안이라(그래서 글이
// 거기 흐른다) 그 영역에서는 채움이 다시 최빈색이다 — 전에는 이 검사를 rect·rounded-rect로
// 닫아 두었는데, 영역을 옮기면 12종 전수로 열린다.
//
// 성립하지 않는 자리는 chevron 하나다: 그 프리셋의 rect는 폭 0으로 붕괴해 우리가 **박스
// 전체**를 쓰므로(PPTX가 되돌린다 — ADR 0041) 영역 안에 도형 밖 흰색이 크게 남는다. 판정을
// 이름이 아니라 `ShapeTextCancel`로 하는 이유는 목록이 두 벌이 되지 않게 하는 것이다.
func inShapeBandRegion(entry inShapeText) (geom.Rect, bool) {
	if cancel, ok := web.ShapeTextCancel(entry.preset); ok && cancel.HasPercent() {
		return geom.Rect{}, false
	}
	// 위아래는 **모서리 반경만큼 더** 줄인다: 둥근 모서리에서는 잰 rect의 위·아래 몇 줄에
	// 도형 밖 흰색이 남아, 그 줄이 배경(채움)과 달라 하나의 띠로 세어진다 (측정: radius=32
	// 도형이 글줄 하나인데 띠 셋으로 읽혔다 — 위 모서리·글자·아래 모서리). 반경만큼
	// 내려가면 그 행부터는 폭 전체가 채움이다 — 텍스트 영역의 px 인셋(0.29·radius)은 원호에
	// **닿는** 사각형이라 그만큼으로는 모자란다.
	insetY := inShapeBandInset
	if entry.radius+2 > insetY {
		insetY = entry.radius + 2
	}
	text := presetTextRect(entry)
	x0 := math.Max(text.X, entry.rect.X+inShapeBandInset)
	y0 := math.Max(text.Y, entry.rect.Y+insetY)
	x1 := math.Min(text.Right(), entry.rect.Right()-inShapeBandInset)
	y1 := math.Min(text.Bottom(), entry.rect.Bottom()-insetY)
	if x1-x0 <= 0 || y1-y0 <= 0 {
		return geom.Rect{}, false
	}
	return geom.Rect{X: x0, Y: y0, W: x1 - x0, H: y1 - y0}, true
}

// inShapeLineDivergences는 도형 안 문단의 줄 수가 **실제로 갈라지는** 자리다
// (`<픽스처>/<요소 id>` → 브라우저 줄 수 / LibreOffice 줄 수).
//
// 갈라짐을 표로 **단정**하는 이유(값을 완화하지 않는 이유): 이것은 오차가 아니라 두 엔진의
// **줄바꿈 규칙 차이**다 — Chromium은 한글을 음절 사이에서 끊고(`데이터와 모델의 편향/을`),
// LibreOffice는 낱말 경계(공백)에서만 끊는다(`데이터와 모델의/편향을 측정한다`). 그래서 같은
// 폭에서 LibreOffice의 첫 줄이 더 짧아지고, 밀린 글자가 마지막 줄을 하나 더 만든다.
// 픽셀 비율로 덮으면 이 사실이 기록에서 사라지므로, 갈라진 자리와 수를 여기 적어 **바뀌면
// 테스트가 깨지게** 둔다 (고쳐도 깨진다 — 그때 표를 지우는 것이 고쳤다는 증거다).
//
// 왜 도형에서 처음 드러났는가: 텍스트 박스는 잰 글자 폭보다 4% 넓게 내보내 흡수하지만
// (pptx.TextBoxRect) **도형은 넓힐 수 없다** — 박스가 저자가 정한 크기다. 다만 원인은 여유가
// 아니라 규칙이므로 여유가 8.0%인 자리에서도 갈라진다 (아래 s1.1.1) — 넓히는 것으로는 닫히지
// 않는 문제이고, 후보 수단은 웹에 `word-break: keep-all`을 걸어 **규칙 자체를 맞추는 것**이다
// (모든 덱의 타이포그래피가 움직이므로 이 항목의 범위 밖이다 — NOTES 04/06의 열린 질문).
var inShapeLineDivergences = map[string][2]int{
	// 002_token_colors의 accent1 카드 본문 `글자색도 토큰(bg)이다.` — 여유 8.0%.
	// LibreOffice가 `(bg)` 뒤에서 끊어 `이다.`가 셋째 줄로 내려간다.
	shapeFixtureName + "/s1.1.1": {2, 3},
	// 006_cards의 첫 카드 본문 `데이터와 모델의 편향을 측정한다.` — 여유 2.4%.
	// LibreOffice는 `데이터와 모델의`까지만 첫 줄에 두고(낱말 경계) 마침표가 셋째 줄이 된다.
	shapeFixtureName + "/s6.1.0.1": {2, 3},
}

// 도형 안 문단이 두 엔진에서 **같은 줄 수**로 접히는가.
//
// 슬라이드 단위 잉크 띠 판정(TestFixtureLineBreaksMatchLibreOffice)이 도형 픽스처에서
// 성립하지 않으므로(칠한 도형이 큰 면을 덮어 띠가 글줄이 아니라 도형이 된다 —
// inkBandSkipSlides) 줄 수 계약을 **도형 안쪽**으로 옮긴다: 도형의 잰 rect 안에서는 가장 흔한
// 색이 채움이고 잉크는 글자다.
//
// 이 검사가 이 iteration에 필요한 이유(항목 05의 5번): 텍스트 박스는 정렬 방향으로 4% 넓혀
// LibreOffice의 스크립트 간격을 흡수하지만(pptx.TextBoxRect) **도형은 넓힐 수 없다** — 박스가
// 저자가 정한 크기이므로 넓히면 그림이 커진다. 그래서 도형 안 문단에는 여유를 만드는 장치가
// 없고, 여유가 0에 가까운 문단은 LibreOffice에서 줄이 하나 늘어난다. 늘어난 줄은 도형 밖으로
// 흐르므로 픽셀 비율로는 잔차와 구별되지 않는다 — 그것이 이 검사가 따로 있는 이유다.
func TestShapeFixtureInShapeTextLineCounts(t *testing.T) {
	assertInShapeLineCounts(t, shapeFixtureName)
}

// **배경 모드**에서도 같은 계약이다 (11_shape_layout — ADR 0049).
//
// 같은 방법이 그대로 성립한다: 카드의 채움이 영역의 최빈색이고 잉크는 글자다. 갈라지는 것은
// 위험의 크기다 — 배경 모드의 자식은 텍스트 박스가 되어 정렬 방향으로 4% 여유를 받으므로
// (pptx.TextBoxRect) 줄이 늘 위험이 흡수 모드보다 구조적으로 작다.
//
// 그래도 이 검사를 따로 두는 이유는 **이 픽스처에서 줄바꿈을 보는 검사가 사라지기 때문**이다:
// 칠한 카드가 슬라이드 단위 잉크 띠의 전제를 깨 세 슬라이드가 전부 빠져 있고
// (inkBandSkipSlides), 픽셀 비율 완화(fixtureDiffOptions)를 정당하게 만드는 구조 검사가
// 좌표 대조와 이 검사 둘이다.
func TestShapeLayoutFixtureInShapeTextLineCounts(t *testing.T) {
	assertInShapeLineCounts(t, shapeLayoutFixtureName)
}

// assertInShapeLineCounts는 픽스처 하나의 도형 안 줄 수를 두 엔진에서 견준다 — 위 두 검사가
// 공유한다. 모드로 갈라지지 않는 이유: 방법이 보는 것은 "도형의 텍스트 영역 안에서 띠가 몇
// 개인가"이고, 그 물음은 글자가 도형의 문단이든 형제 텍스트 박스든 같다 (inShapeTexts도 도형의
// 자손을 전부 훑으므로 컨테이너를 거친 글자가 그대로 세어진다).
func assertInShapeLineCounts(t *testing.T, fixtureName string) {
	t.Helper()
	RequireChromium(t)
	RequireSoffice(t)
	RequirePdftoppm(t)

	fixture := fixtureByName(t, fixtureName)
	result := fixture.Build(t)
	if result.Geometry == nil {
		t.Fatal("빌드에 지오메트리가 없다")
	}
	stage := result.Deck.Stage
	shots := RenderPPTX(t, fixture.Output(t, result, ".pptx"), image.Pt(stage.Width, stage.Height))
	browser := NewBrowser(t)
	browser.Load(t, filepath.Join(result.OutDir, web.IndexFile))

	webShots := make([]image.Image, len(result.Deck.Slides))
	for i := range result.Deck.Slides {
		webShots[i] = browser.ScreenshotElement(t, web.SlideIndexSelector(i))
	}

	entries := inShapeTexts(t, result.Deck, result.Geometry)
	if len(entries) == 0 {
		t.Fatal("도형 안 텍스트가 하나도 없다 — 픽스처가 이 계약을 시험하지 않는다")
	}
	worst := 1.0
	seen := map[string]bool{}
	for _, entry := range entries {
		inner, ok := inShapeBandRegion(entry)
		if !ok {
			t.Logf("%s %s: 줄 세기를 건너뛴다 — 프리셋의 텍스트 영역이 실루엣 안이 아니다 (inShapeBandRegion)",
				entry.id, entry.preset)
			continue
		}
		webBands := InkBands(webShots[entry.slide], inner)
		loBands := InkBands(shots[entry.slide], inner)
		if entry.slack < worst {
			worst = entry.slack
		}
		rows := bandRows(entry.lineRects)
		t.Logf("%s %s %s: 웹 잰 줄 %d(띠 %d) / 웹 잉크 띠 %d / LibreOffice 잉크 띠 %d, 여유 %.1f%%",
			entry.id, entry.preset, entry.rect, len(entry.lineRects), rows, len(webBands), len(loBands), entry.slack*100)
		if len(webBands) != rows {
			// 웹 그림과 잰 줄이 어긋나면 측정 방법이 틀린 것이다 (LibreOffice 판정보다 먼저
			// 본다 — bandLineOwners와 같은 태도).
			t.Errorf("%s: 웹 잉크 띠 %d개가 잰 줄 묶음 %d개와 다르다 — 도형 안 줄 세기 방법이 틀렸다\n  띠 %v\n  잰 줄 %v",
				entry.id, len(webBands), rows, webBands, entry.lineRects)
			continue
		}
		if want, known := inShapeLineDivergences[fixtureName+"/"+entry.id]; known {
			seen[entry.id] = true
			if len(webBands) != want[0] || len(loBands) != want[1] {
				t.Errorf("%s: 알려진 갈라짐이 %d/%d줄인데 %d/%d줄이다 (브라우저/LibreOffice) — "+
					"고쳤으면 inShapeLineDivergences에서 지우고, 나빠졌으면 원인을 찾는다\n  웹 %v\n  LibreOffice %v",
					entry.id, want[0], want[1], len(webBands), len(loBands), webBands, loBands)
			}
			continue
		}
		if len(loBands) != len(webBands) {
			t.Errorf("%s: 도형 안 줄 수가 갈라진다 — 브라우저 %d줄, LibreOffice %d줄 (여유 %.1f%%). "+
				"도형은 넓힐 수 없으므로(pptx.writeShapeSp) 오차를 키울 자리가 아니다\n  웹 %v\n  LibreOffice %v",
				entry.id, len(webBands), len(loBands), entry.slack*100, webBands, loBands)
		}
	}
	t.Logf("도형 안 문단의 최소 여유 비율 = %.1f%% — 이 값이 0에 가까울수록 LibreOffice에서 줄이 늘 위험이 크다", worst*100)
	// 표에 있는데 픽스처에서 사라진 자리는 지운다 — 남아 있으면 갈라짐이 고쳐졌는지
	// 없어졌는지 구별되지 않는다.
	// **이 픽스처의 행만** 본다 (다른 픽스처의 행은 그쪽 검사가 소유한다).
	for key := range inShapeLineDivergences {
		id, mine := strings.CutPrefix(key, fixtureName+"/")
		if !mine || seen[id] {
			continue
		}
		t.Errorf("inShapeLineDivergences의 %s가 픽스처에 없다 — 표에서 지운다", key)
	}
}

// inShapeTextRectSlides는 프리셋별 텍스트 영역을 시험하는 슬라이드다 (id → 가운데 정렬인가).
//
// 두 장인 이유는 판정이 둘이기 때문이다: 가운데 정렬은 영역의 **중심**을, 왼쪽·위 정렬은
// **왼쪽·위 변**을 드러낸다 (대칭 인셋 프리셋은 가운데 정렬로는 어긋남이 상쇄된다 — 픽스처
// 009·010의 주석).
var inShapeTextRectSlides = map[string]bool{
	"shape-text-rect":        true,
	"shape-text-rect-corner": false,
}

// inShapeTextRectSlack은 도형 안 글자 자리를 두 타깃에서 견줄 때의 허용 오차다 (px).
//
// 두 엔진의 글리프 래스터라이즈가 다르므로 잉크의 바깥 경계는 완전히 같을 수 없다 (좌우
// 사이드 베어링, 첫 기준선의 반올림 — 실측 4px). 그 상수는 `rect` 행이 대조군으로
// **빼 주므로**(inShapeTextRectControl) 여기 남는 것은 프리셋마다 다른 몫뿐이고, 그래서 이
// 값이 표에서 가장 작은 인셋(rounded-rect의 `0.29289 × radius` = 반경 12에서 3.5px)보다
// 작을 수 있다 — 어긋남의 정체를 가리지 않는다는 뜻이다.
const inShapeTextRectSlack = 2.0

// inShapeTextRectControl은 대조군 프리셋이다 — 텍스트 영역이 박스 전체인 `rect`다.
//
// 왜 필요한가: 두 엔진의 첫 줄 기준선이 같은 자리가 아니어서 같은 텍스트 영역에서도 잉크의
// 위 경계가 상수만큼 (실측 4px) 어긋난다. 그 상수를 허용 오차에 흡수시키면 오차가 표의 가장
// 작은 인셋만큼 커져 시험이 무뎌지므로, **영역이 박스와 같은 프리셋**에서 그 상수를 재고
// 나머지 프리셋의 어긋남에서 뺀다 — 남는 것이 순수하게 텍스트 영역의 몫이다.
const inShapeTextRectControl = "rect"

// inShapeTextRectDelta는 도형 하나에서 잰 두 타깃의 어긋남이다 — 대조군을 빼려면 한 슬라이드의
// 잰 값이 다 모인 뒤에 판정해야 하므로 잰 것과 판정하는 것이 두 단계로 나뉜다.
type inShapeTextRectDelta struct {
	entry inShapeText
	// dx·dy는 웹 잉크 − LibreOffice 잉크다 (what이 무엇을 견줬는지 말한다: 중심 또는 왼쪽 위).
	dx, dy float64
	what   string
	// text는 표가 뜻하는 텍스트 영역, loInk는 LibreOffice가 실제로 글자를 놓은 자리다 —
	// 진단 로그와 "영역 안에 있는가" 단정이 쓴다.
	text  geom.Rect
	loInk geom.Rect
}

// 프리셋의 텍스트 영역이 두 타깃에서 **같은 자리**인가 (ADR 0040의 회귀 테스트).
//
// 표(render/web.ShapeTextRect)가 맞는지를 우리 코드 두 벌을 견주는 것으로는 알 수 없다 —
// 수치의 출처가 PowerPoint·LibreOffice가 들고 있는 프리셋 정의이고, PPTX 쪽은 그 값을 **쓰지
// 않는다**(ADR 0040: `bodyPr` 인셋은 저자의 `pad`다). 그래서 보장을 픽셀로 옮긴다: 같은
// SOML에서 웹 잉크와 LibreOffice 잉크가 같은 자리에 오면 웹이 읽은 표가 프리셋과 같다는
// 뜻이다. 표가 틀리면 이 검사가 깨지고, 표를 두 벌로 공유하는 것보다 강하다.
//
// 잉크를 **박스 전체**에서 재는 이유: 고치기 전의 웹 글자는 텍스트 영역 밖(박스 모서리)에
// 있으므로 영역으로 크롭하면 증거가 잘려 나간다. 박스 안에서 글자만 잉크가 되는 것은 픽스처의
// `fill="surface"` 덕이다 (009의 주석 — 잉크 임계값보다 칠과 배경의 차가 작다).
func TestShapeFixtureInShapeTextRect(t *testing.T) {
	RequireChromium(t)
	RequireSoffice(t)
	RequirePdftoppm(t)

	fixture := fixtureByName(t, shapeFixtureName)
	result := fixture.Build(t)
	if result.Geometry == nil {
		t.Fatal("빌드에 지오메트리가 없다")
	}
	centered := map[int]bool{}
	for i, slide := range result.Deck.Slides {
		if mode, ok := inShapeTextRectSlides[slide.ID]; ok {
			centered[i] = mode
		}
	}
	if len(centered) != len(inShapeTextRectSlides) {
		t.Fatalf("텍스트 영역 슬라이드를 %d장만 찾았다 — %v", len(centered), inShapeTextRectSlides)
	}

	stage := result.Deck.Stage
	shots := RenderPPTX(t, fixture.Output(t, result, ".pptx"), image.Pt(stage.Width, stage.Height))
	browser := NewBrowser(t)
	browser.Load(t, filepath.Join(result.OutDir, web.IndexFile))
	webShots := map[int]image.Image{}
	for index := range centered {
		webShots[index] = browser.ScreenshotElement(t, web.SlideIndexSelector(index))
	}

	seen := map[string]int{}
	var measured []inShapeTextRectDelta
	control := map[int]inShapeTextRectDelta{}
	for _, entry := range inShapeTexts(t, result.Deck, result.Geometry) {
		mode, ok := centered[entry.slide]
		if !ok {
			continue
		}
		seen[entry.preset]++
		text := presetTextRect(entry)
		webInk, webOK := inkBounds(webShots[entry.slide], entry.rect)
		loInk, loOK := inkBounds(shots[entry.slide], entry.rect)
		if !webOK || !loOK {
			t.Errorf("%s %s: 박스 안에 잉크가 없다 (웹 %t, LibreOffice %t) — 글자가 박스를 "+
				"벗어났거나 칠이 잉크로 읽혔다", entry.id, entry.preset, webOK, loOK)
			continue
		}
		delta := inShapeTextRectDelta{entry: entry, text: text, loInk: loInk, what: "왼쪽 위"}
		if mode {
			delta.dx = (webInk.X + webInk.W/2) - (loInk.X + loInk.W/2)
			delta.dy = (webInk.Y + webInk.H/2) - (loInk.Y + loInk.H/2)
			delta.what = "중심"
		} else {
			delta.dx, delta.dy = webInk.X-loInk.X, webInk.Y-loInk.Y
		}
		measured = append(measured, delta)
		if entry.preset == inShapeTextRectControl {
			control[entry.slide] = delta
		}
	}
	if len(control) != len(centered) {
		t.Fatalf("대조군 %q를 %d장에서만 찾았다, want %d장 — 두 엔진의 기준선 차를 뺄 수 없다",
			inShapeTextRectControl, len(control), len(centered))
	}
	for _, d := range measured {
		base := control[d.entry.slide]
		dx, dy := d.dx-base.dx, d.dy-base.dy
		// 표가 뜻하는 인셋과 LibreOffice가 실제로 준 인셋을 나란히 남긴다 — 깨졌을 때
		// 고칠 곳이 웹인지 표인지 이 줄이 가른다.
		t.Logf("%-14s %s: 잉크 %s Δ = (%+5.1f, %+5.1f)px (대조군 보정 전 %+.1f, %+.1f) | "+
			"표 l %.1f t %.1f r %.1f b %.1f | LibreOffice 잉크 − 박스 l %+.1f t %+.1f",
			d.entry.preset, d.entry.rect, d.what, dx, dy, d.dx, d.dy,
			d.text.X-d.entry.rect.X, d.text.Y-d.entry.rect.Y,
			d.entry.rect.Right()-d.text.Right(), d.entry.rect.Bottom()-d.text.Bottom(),
			d.loInk.X-d.entry.rect.X, d.loInk.Y-d.entry.rect.Y)
		if math.Abs(dx) > inShapeTextRectSlack || math.Abs(dy) > inShapeTextRectSlack {
			t.Errorf("%s %s: 도형 안 글자의 %s가 두 타깃에서 (%+.1f, %+.1f)px 어긋난다 (허용 %.1fpx) — "+
				"웹이 프리셋의 텍스트 영역을 읽지 않으면 이 값은 표의 인셋과 같다 (ADR 0040)",
				d.entry.id, d.entry.preset, d.what, dx, dy, inShapeTextRectSlack)
		}
		// LibreOffice의 글자는 프리셋의 텍스트 영역 안에 있어야 한다 — 표가 프리셋과 같다는
		// 직접 단정이다 (`pad`는 영역 안쪽에 더 들어가므로 한 방향으로만 본다).
		if d.loInk.X < d.text.X-inShapeTextRectSlack || d.loInk.Y < d.text.Y-inShapeTextRectSlack ||
			d.loInk.Right() > d.text.Right()+inShapeTextRectSlack || d.loInk.Bottom() > d.text.Bottom()+inShapeTextRectSlack {
			t.Errorf("%s %s: LibreOffice 잉크 %s가 표의 텍스트 영역 %s을 벗어난다 — 표의 행이 "+
				"프리셋 정의와 다르다 (render/web.ShapeTextRect)", d.entry.id, d.entry.preset, d.loInk, d.text)
		}
	}
	for _, preset := range soml.ShapeTypes() {
		if seen[preset] != len(inShapeTextRectSlides) {
			t.Errorf("%s: 텍스트 영역 슬라이드에서 %d번 잡혔다, want %d번 — 프리셋 전수여야 한다",
				preset, seen[preset], len(inShapeTextRectSlides))
		}
	}
}

// presetTextRect는 프리셋의 텍스트 영역을 잰 박스 위에 얹는다 (ADR 0040의 표).
func presetTextRect(entry inShapeText) geom.Rect {
	insets, ok := web.ShapeTextRect(entry.preset)
	if !ok {
		return entry.rect
	}
	l, t, r, b := insets.Px(entry.rect.W, entry.rect.H, entry.radius)
	return geom.Rect{X: entry.rect.X + l, Y: entry.rect.Y + t, W: entry.rect.W - l - r, H: entry.rect.H - t - b}
}

// inkBounds는 영역 안 잉크 전체를 감싸는 사각형이다 (띠의 합집합).
func inkBounds(img image.Image, r geom.Rect) (geom.Rect, bool) {
	bands := InkBands(img, r)
	if len(bands) == 0 {
		return geom.Rect{}, false
	}
	out := bands[0]
	for _, band := range bands[1:] {
		if band.X < out.X {
			out.W += out.X - band.X
			out.X = band.X
		}
		if band.Right() > out.Right() {
			out.W = band.Right() - out.X
		}
		out.H = band.Bottom() - out.Y
	}
	return out, true
}

// inShapeText는 텍스트를 담은 도형 하나다 (도형 안 줄 수 대조의 단위).
type inShapeText struct {
	id     string
	preset string
	slide  int
	rect   geom.Rect
	// radius는 모서리 반경 px다 (rounded-rect가 아니면 0) — 줄 세기 영역이 둥근 모서리를
	// 피해야 한다.
	radius float64
	// lineRects는 브라우저가 이 도형 안에서 나눈 줄의 rect 전부다 (자식 문단을 이어 붙인
	// 것). 개수가 아니라 rect를 드는 이유는 **나란히 앉은 글자**다: 도형 안에 `<row>`가
	// 있으면 서로 다른 문단의 줄이 같은 y에 앉아 잉크 띠 **하나**가 되므로, 기대값은 줄
	// 개수가 아니라 y로 겹치는 묶음의 수다 (bandRows).
	lineRects []geom.Rect
	// slack은 자식 문단 중 **가장 좁은** 여유 비율이다 (Slack/MaxLineWidth).
	slack float64
}

// bandRows는 잰 줄 rect들을 **y로 겹치는 묶음**의 수로 센다 — 잉크 띠 하나가 몇 개의 잰 줄에
// 대응하는지의 답이고, 줄 세기의 기대값이다.
//
// 세로 스택에서는 줄 개수와 같다 (줄이 y로 겹치지 않는다). 갈라지는 자리는 도형 안 `<row>`다:
// 배지와 소제목이 나란히 앉으면 두 문단의 줄이 같은 y에 있어 잉크는 띠 **하나**가 된다 —
// 개수로 기대하면 그 배치에서 "방법이 틀렸다"가 나오는데, 틀린 것은 배치가 아니라 기대값이다.
func bandRows(lines []geom.Rect) int {
	if len(lines) == 0 {
		return 0
	}
	sorted := append([]geom.Rect(nil), lines...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Y < sorted[j].Y })
	rows, bottom := 1, sorted[0].Bottom()
	for _, line := range sorted[1:] {
		// 위 묶음의 아래 변보다 아래에서 시작하면 새 띠다. bottom을 누적 최대로 두는 이유는
		// 키가 다른 글자가 나란히 앉는 경우다 (배지 36px + 소제목 24px).
		if line.Y >= bottom {
			rows++
		}
		bottom = math.Max(bottom, line.Bottom())
	}
	return rows
}

// inShapeTexts는 텍스트를 담은 도형을 지오메트리와 같은 순서로 훑어 모은다.
//
// 순회 규칙은 geom.Refs와 같아야 한다 (인라인은 자손까지 건너뛴다) — 어긋나면 자리가 밀려
// 엉뚱한 요소의 rect를 도형의 rect로 쓴다. 그 계약을 마지막에 개수로 확인한다.
func inShapeTexts(t *testing.T, deck *soml.Deck, g *geom.Geometry) []inShapeText {
	t.Helper()
	var out []inShapeText
	index := 0
	var walk func(node soml.Node, shape *inShapeText)
	walk = func(node soml.Node, shape *inShapeText) {
		elem, ok := node.(*soml.Element)
		if !ok {
			return
		}
		kind := soml.KindOf(elem.Name)
		if kind == soml.KindInline {
			return
		}
		if index >= len(g.Boxes) {
			t.Fatalf("순회가 측정 박스(%d개)보다 길다", len(g.Boxes))
		}
		box := g.Boxes[index]
		index++
		if kind == soml.KindShape {
			style := elem.ShapeStyle()
			if style == nil {
				t.Fatalf("<shape>의 외관이 해석되지 않았다")
			}
			entry := inShapeText{id: box.ID, preset: style.Type, slide: box.Slide, rect: box.Rect, slack: 1.0}
			if style.Radius != nil {
				entry.radius = style.Radius.Px
			}
			for _, child := range elem.Children {
				walk(child, &entry)
			}
			if len(entry.lineRects) > 0 {
				out = append(out, entry)
			}
			return
		}
		if shape != nil {
			shape.lineRects = append(shape.lineRects, box.Lines...)
			if width := box.MaxLineWidth(); width > 0 {
				if ratio := box.Slack() / width; ratio < shape.slack {
					shape.slack = ratio
				}
			}
		}
		for _, child := range elem.Children {
			walk(child, shape)
		}
	}
	for _, slide := range deck.Slides {
		for _, node := range slide.Content {
			walk(node, nil)
		}
	}
	if index != len(g.Boxes) {
		t.Fatalf("순회 %d개, 측정 박스 %d개 — 순회 규칙이 geom.Refs와 어긋났다", index, len(g.Boxes))
	}
	return out
}

// 한 줄에 **여러 박스가 같은 y**에 앉는 배치에서 좌표 대조가 뜻을 유지하는가.
//
// 지금까지 모든 픽스처는 세로 스택이라 박스의 y가 단조 증가했고, "그리드가 그리드다"는 사실은
// 어느 검사도 보지 않았다 — 골든 xfrm은 숫자가 그럴듯한지 말해 주지 않고, 좌표 대조
// (TestFixtureCoordinatesMatchGeometry)는 측정값과 산출물이 같은지만 본다. 측정 자체가
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

// imageRegionDiffMaxRatio는 **그림 하나**(잰 rect)의 픽셀 오차 허용 비율이다.
//
// 슬라이드 전체 비율(fixtureDiffOptions)과 갈라 두는 이유가 이 iteration의 핵심 계약이다:
// 슬라이드는 대부분 배경이므로 크롭 축이 뒤바뀐 그림 하나는 전체 비율에서 희석된다 (006의
// 전면 그림처럼 큰 그림 하나가 슬라이드의 45%인 슬라이드도 있지만, 001·005는 그림 하나가
// 3~5%다 — 축이 뒤바뀌어도 픽스처 임계값 안에 든다). 그림만 잘라 보면 `cover`가 **다른 부분**을
// 보여주는 것이 곧바로 수치가 된다: 잘린 쪽이 바뀌면 표식(좌상단 흰 사각형)이 크롭 안에서
// 사라지거나 나타나므로 오차가 그림 면적의 수십 %가 된다.
//
// 여유를 넣지 않고 **잰 rect 그대로** 자르는 이유: 그림자·회전 외접 사각형·경계 중앙에
// 그려지는 `a:ln`은 rect 밖으로 24px까지 퍼지는데, 픽스처의 가장 좁은 칸 간격이 `gap="md"`
// = 24px이라 여유를 그만큼 주면 **이웃 그림이 크롭에 들어온다** (도형은 여유 16px < 간격
// 24px으로 그 문제를 피했다 — shapeCropMargin). 밖으로 나간 잉크의 판정은 슬라이드 전체
// 비교가 갖고, 이 검사는 "그림의 어느 부분이 어디에 그려졌는가"만 본다.
//
// 6%는 측정값에 여유를 더한 값이다 (항목 05/05의 측정 — 아래 로그의 표): 22장 중 최대는
// 004_image_text의 `cover` 그림 4.05%이고 그 잔차의 정체는 **리샘플링 커널**이다. 두 렌더러가
// 같은 배율로 줄이지 않으므로(브라우저는 부드럽게, LibreOffice는 거칠게) 표식 경계의 한 줄이
// 서로 다른 픽셀로 떨어진다 — 그림 안이 단색인 2색 에셋을 고른 이유가 이것이고(잔차가 경계에만
// 남는다), 그래서 오차는 그림 크기가 아니라 **경계의 길이**에 비례한다.
const imageRegionDiffMaxRatio = 0.06

// imageRegionDiffMinPixels는 그림 오차의 **절대 하한 허용값**이다 (다른 픽셀 수).
//
// 잔차는 표식 경계에 남으므로 픽셀 수가 그림 **크기와 무의존**한데(리샘플링 커널의 차이는
// 경계 한 줄에서 난다) 비율은 면적으로 나눈 값이다 — 아주 작은 그림에서는 그 한 줄이 큰
// 비율로 계산되어 **방법 문제로 죽는다**: 001_fit의 `dot.png`(16×8 = 128픽셀)에서 경계
// 20픽셀이 13.9%다. 32는 그 측정값에 여유를 더한 값이고, 게이트를 약화하지 않는다 — 크롭
// 축이 갈라지는 `cover` 그림은 전부 17,000픽셀 이상이라 6%가 1,000픽셀을 넘고, 그 그림들에서
// 이 하한은 판정에 닿지 않는다. loLineWidthMinPx가 짧은 잉크 띠에 둔 하한과 같은 종류의
// 장치다 (그쪽은 AA 잡음 1px, 이쪽은 경계 한 줄).
const imageRegionDiffMinPixels = 32

// 그림이 두 타깃에서 **같은 영역을 같은 자리에** 보여주는가 — 그림마다 잰다.
//
// 이 iteration의 가장 큰 판정이다 (TASKS Iteration 5의 리뷰 포인트 "cover 크롭이 양 타깃에서
// 같은 영역을 보여주는지"의 자동 절반). 웹은 `object-fit: cover`가 중앙을 보여주고 PPTX는
// `srcRect`로 양쪽에서 같은 몫을 자르는데(pptx.coverCrop), 두 계산이 어긋나면 **같은 그림의
// 다른 부분**이 같은 자리에 그려진다 — 좌표 대조(xfrm == 측정 EMU)는 그것을 볼 수 없다.
func TestImageFixturePictureRegions(t *testing.T) {
	RequireChromium(t)
	RequireSoffice(t)
	RequirePdftoppm(t)

	fixture := fixtureByName(t, imageFixtureName)
	result := fixture.Build(t)
	if result.Geometry == nil {
		t.Fatal("빌드에 지오메트리가 없다")
	}
	stage := result.Deck.Stage
	shots := RenderPPTX(t, fixture.Output(t, result, ".pptx"), image.Pt(stage.Width, stage.Height))
	browser := NewBrowser(t)
	browser.Load(t, filepath.Join(result.OutDir, web.IndexFile))

	worst, worstAt := 0.0, ""
	pictures := 0
	for i, slide := range result.Deck.Slides {
		var boxes []geom.Box
		for _, box := range result.Geometry.Boxes {
			if box.Slide == i && box.El == soml.ImageElement {
				boxes = append(boxes, box)
			}
		}
		if len(boxes) == 0 {
			continue
		}
		webImg := browser.ScreenshotElement(t, web.SlideIndexSelector(i))
		loImg := shots[i]
		for _, box := range boxes {
			cmp, err := Diff(cropImage(webImg, box.Rect), cropImage(loImg, box.Rect), DefaultDiffOptions())
			if err != nil {
				t.Fatalf("%s/%s: %v", slide.ID, box.ID, err)
			}
			pictures++
			t.Logf("%-16s %-10s %s: %s", slide.ID, box.ID, box.Rect, cmp)
			if cmp.Ratio > worst {
				worst, worstAt = cmp.Ratio, slide.ID+"/"+box.ID
			}
			if cmp.Ratio > imageRegionDiffMaxRatio && cmp.Changed > imageRegionDiffMinPixels {
				t.Errorf("%s/%s: 그림 안 픽셀 오차 %.2f%%가 허용 %.2f%%를 넘는다 — 크롭이 갈라졌다면 "+
					"고칠 곳은 오차가 아니라 웹의 `fit` CSS(render/web.imageFitCSS)와 크롭 계산(pptx.coverCrop)이다",
					slide.ID, box.ID, cmp.Ratio*100, imageRegionDiffMaxRatio*100)
			}
		}
	}
	if pictures != imageFixturePictures {
		t.Errorf("그림 %d개를 쟀다, want %d개 — 픽스처가 바뀌면 이 수를 함께 고친다", pictures, imageFixturePictures)
	}
	t.Logf("가장 큰 그림 오차: %s %.2f%% (허용 %.2f%%, 그림 %d개)",
		worstAt, worst*100, imageRegionDiffMaxRatio*100, pictures)
}

// imageFixturePictures는 그림 픽스처의 `<image>` 수다 (문서 전체).
//
// 수를 못 박는 이유: 위 검사는 측정 목록을 걷다가 그림을 찾으므로, 순회가 그림을 놓치기
// 시작하면 **아무것도 비교하지 않고 통과한다** (측정 목록에서 `<image>`가 빠지는 실패가
// 정확히 그 모양이다). 슬라이드별 내역은 001 5 + 002 6 + 003 3 + 004 1 + 005 5 + 006 1 +
// 007 3 = 24다.
const imageFixturePictures = 24

// LibreOffice 렌더가 견뎌야 하는 글줄 오차의 임계값 (PRD §13-1의 수치 — 이 테스트가 잰
// 값에 여유를 더해 못 박은 회귀 감시선이다. 측정값은 NOTES.md 01/07의 표에 있다).
const (
	// loLineOffsetMaxPx는 글줄 잉크 세로 중심의 허용 오차다. spcPts(절대 줄간격)로 줄 전진
	// 폭은 맞췄지만 첫 기준선 배치는 엔진 소관이라 0이 되지 않는다 — LibreOffice가 일관되게
	// 아래로 +2~5px(측정 최대 +5.0px, 캡션 20px)이고, 6px은 거기에 픽셀 반올림 여유를 더한
	// 값이다. 이 선을 넘으면 줄간격 매핑이 다시 갈라진 것이다 (spcPct 시절에는 +8~13px였다).
	loLineOffsetMaxPx = 6.0
	// loLineWidthMaxRatio는 글줄 잉크 폭의 허용 오차 비율이다. LibreOffice는 한글↔라틴
	// 경계마다 스크립트 간격을 넣어 혼합 문자 줄을 넓게 그린다 — 01_text의 긴 본문에서 +3.3%
	// (측정값). 4%는 텍스트 박스의 안전 여유(pptx.TextBoxRect)와 같은 값이다.
	//
	// **비율은 이 간격의 자연스러운 단위가 아니다**: 간격은 줄 길이가 아니라 **경계 수**에
	// 비례하므로(경계당 +0.245 em — script_spacing_test.go가 잰다) 같은 문장을 좁은 트랙에
	// 넣으면 비율만 커진다. 그것이 아래 loLineWidthMaxRatios가 픽스처마다 조금씩 커진 이유이고,
	// 이 상수를 그만큼 키울 수 없는 이유는 여유를 키우면 반대 방향으로 줄이 줄어들기
	// 때문이다 (docs/guides/known-differences.md).
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
	// 06_stack: 측정 최대 +5.5% (010 content-explicit의 첫 문단 — 항목 06). 겹침이 아니라
	// **겹치지 않은 텍스트 슬라이드**의 잔차다 (이 픽스처에서 잉크 띠 판정에 남는 유일한
	// 슬라이드다 — inkBandSkipSlides가 001~009만 뺀다). 원인은 02·03과 같다: LibreOffice의
	// 스크립트 간격은 한글↔라틴 **경계 수**에 비례하는데 비율은 줄 길이로 나눈 값이라, 짧은
	// 문단(`layout="content"는 기본 스테이지의 이름이다.` — 라틴 런 `layout="content"` +
	// 인용부호가 한글 문장에 낀다)에서 비율이 커진다. 줄 수 갈라짐은 0 오차의 줄 수 검사가
	// 따로 잡고(이 슬라이드는 줄 수가 같다), 6.0%는 측정값에 앞 픽스처들과 같은 폭의 여유를
	// 더한 값이다 (02 5.0%→5.5%, 03 5.8%→6.5%와 같은 태도).
	stackFixtureName: 0.06,
	// 14_theme: 측정 최대 +5.5% (002_roles의 캡션 줄 — P4.2). 원인은 02·03·06과 같다
	// (스크립트 간격은 한글↔라틴 **경계 수**에 비례하고 비율은 줄 길이로 나눈 값이다): 이
	// 픽스처의 글줄은 역할 이름과 수치를 말하느라 `24px`·`muted`·`variant`처럼 라틴 낱말이
	// 한글 문장에 박혀 있고, 캡션 줄(`캡션은 muted로 물러난다.` — 200px, 18px 글자)은 그 중
	// 가장 짧아 경계 둘의 간격이 비율로 가장 크게 나온다.
	//
	// 테마가 이 수치를 만든 것이 아니다: ink의 본문 리듬은 세 내장 테마가 공유하는 24px/1.5이고
	// (TestBuiltinThemesShareBodyRhythm), 어두운 바탕은 잉크 띠 측정의 전제를 바꾸지 않는다
	// (가장 흔한 색이 배경이다 — InkBands). 줄 수 갈라짐은 0 오차의 줄 수 검사가 따로 잡고
	// (이 픽스처는 세 장 모두 줄 수가 같다), 6.0%는 측정값에 앞 픽스처들과 같은 폭의 여유를
	// 더한 값이다 (06_stack의 5.5%→6.0%와 같은 태도).
	themeFixtureName: 0.06,
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
// (TestFixtureCoordinatesMatchGeometry — xfrm == 측정 EMU).
var inkBandSkipSlides = map[string]bool{
	// 007_style_appearance: `style="background: #0F65F9"`가 웹에만 적용된다 (§7.9의 계약).
	"03_layout/style-appearance-warning": true,
	// 04_shape 전 슬라이드: 도형이 들어오면 이 방법의 전제가 슬라이드 단위로 깨진다.
	// 칠한 도형은 03_layout/007(저자가 `style`로 칠한 띠)이 이미 깨뜨린 전제이고, 도형
	// 픽스처는 **대부분의 슬라이드가 그렇다** — 잰 줄 하나마다 띠가 하나 있어야 하는 전제
	// 대신, 도형 하나가 채움으로 하나의 큰 띠가 되고 그 안의 글줄은 띠 안에 묻힌다
	// (측정: 001의 첫 줄 도형 넉 장이 1103×112px 띠 **하나**가 된다 — 글줄 넷이 아니다).
	// 값 완화가 아니라 **방법의 전제가 깨진 것**이므로 슬라이드 단위로 건너뛴다.
	//
	// 대신 줄 수 계약을 잃지 않는다: 도형 **안쪽**에서는 가장 흔한 색이 채움이므로 같은
	// 방법이 성립한다 — TestShapeFixtureInShapeTextLineCounts가 도형마다 그것을 본다
	// (그 검사가 이 픽스처에서 줄바꿈 갈라짐의 판정자다). 슬라이드 직속 텍스트(제목)의
	// 줄바꿈은 01~03 픽스처가 같은 계약으로 이미 지킨다.
	"04_shape/shape-types":           true,
	"04_shape/shape-token-colors":    true,
	"04_shape/shape-literal-colors":  true,
	"04_shape/shape-appearance":      true,
	"04_shape/shape-text-anchor":     true,
	"04_shape/shape-cards":           true,
	"04_shape/shape-variant":         true,
	"04_shape/shape-variant-unknown": true,
	// 05_image 전 슬라이드: 그림이 들어오면 전제가 **도형보다 더 크게** 깨진다. 도형의 채움은
	// 잰 rect 하나를 덮지만 그림은 그 자체가 큰 색면이고, 무엇보다 **잰 줄이 없는 자리에
	// 잉크가 있다** — 그래서 판정이 "허용 오차를 넘었다"가 아니라 `띠에 주인이 없다`로
	// 죽는다 (측정: 001의 첫 줄 그림 네 장이 1068×202px 띠 **하나**가 되고, 006의 전면 그림은
	// 1152×360px 띠 하나다. 004처럼 오른쪽에 글이 있는 슬라이드에서는 그림 띠가 그 글줄과
	// 겹쳐 **주인을 찾아 통과하는데**, 그때 재는 것은 글자가 아니라 그림이다 — 값 완화보다
	// 나쁜, 뜻이 뒤바뀐 통과다).
	//
	// 이 픽스처는 일곱 슬라이드 전부에 그림이 있어 "그림이 없는 슬라이드 한정"으로 좁힐
	// 자리가 없다. 잃는 것과 남는 것: 슬라이드 직속 텍스트(제목·캡션)의 줄바꿈 계약은
	// 01~03 픽스처가 같은 검사로 이미 지키고, 그림의 자리·크기는 좌표 대조가 0 오차로
	// (TestFixtureCoordinatesMatchGeometry), 그림의 **내용**은 그림별 크롭 비교가
	// (TestImageFixturePictureRegions) 판정한다.
	"05_image/image-fit":        true,
	"05_image/image-appearance": true,
	"05_image/image-formats":    true,
	"05_image/image-text":       true,
	"05_image/image-grid":       true,
	"05_image/image-full":       true,
	"05_image/image-variant":    true,
	// 06_stack의 겹침 슬라이드 001~009: 겹침은 04·05가 깬 전제(칠한 도형·그림이 단색 배경
	// 가정을 깬다)를 **한 단계 더** 깬다 — 겹친 자리에는 도형이 여러 층 포개져 **띠에 주인이
	// 하나가 아니다**. 잰 줄 하나 ↔ 잉크 띠 하나의 대응이 성립하지 않으므로(포갠 도형들이
	// 하나의 큰 띠로 뭉치거나, 위 도형이 아래 도형의 글줄을 덮는다) 값 완화가 아니라 방법의
	// 전제가 깨진 것이다. 슬라이드 단위로 건너뛴다 (04가 8장, 05가 7장을 같은 이유로 뺐다).
	//
	// 대신 잃는 것과 남는 것: 겹침 순서는 spTree 순서 전수와 겹친 영역 크롭 비교가
	// (TestStackFixtureZOrderIsDocumentOrder·TestStackFixtureOverlapRegions), 좌표는 0 오차
	// 대조가(TestFixtureCoordinatesMatchGeometry) 판정한다. 무엇보다 **010
	// content-explicit는 겹침이 없는 순수 텍스트 슬라이드**라 단색 배경 가정이 성립하므로
	// 잉크 띠 판정에 **남긴다** (아래 표에 없다) — 픽스처 전체를 통째로 빼지 않는다는 것이
	// 이 슬라이드가 픽스처에 있는 이유다 (제목·본문·리스트의 줄바꿈 계약을 이 픽스처에서도
	// 지킨다).
	"06_stack/anchors":       true, // surface 배경 + accent1 배지 9장이 면을 덮는다
	"06_stack/badge-overlay": true, // 그림 위 배지 — 그림이 큰 색면이다
	"06_stack/watermark":     true, // 반투명 워터마크 도형이 본문 아래에 깔린다
	"06_stack/z-order":       true, // 세 도형이 계단식으로 포개진다 (띠에 주인이 여럿)
	"06_stack/absolute":      true, // surface 배경 + x/y 도형들
	"06_stack/in-container":  true, // 그림 카드 3장 + 가로 열 안 도형
	"06_stack/nested":        true, // 중첩 stack — 도형 안 도형이 면을 덮는다
	"06_stack/canvas-free":   true, // 캔버스 자유 배치 도형들
	"06_stack/canvas-stack":  true, // 캔버스 위 stack + 도형
	// 11_shape_layout 전 슬라이드: 04_shape과 **같은 이유**다 (칠한 카드가 큰 띠 하나가 되고
	// 그 안의 글줄이 묻힌다). 이 픽스처의 카드는 배경 하나가 헤더·본문·푸터 셋을 덮으므로
	// 전제가 04보다 더 크게 깨진다 — 잰 줄 셋이 카드 띠 하나 안에 있다.
	//
	// 잃는 것과 남는 것: 자식의 자리는 좌표 대조가 0 오차로 판정하고
	// (TestFixtureCoordinatesMatchGeometry — 배경 모드의 자식은 자기 xfrm을 갖는다), 슬라이드
	// 직속 텍스트(제목)의 줄바꿈은 01~03 픽스처가 같은 검사로 지킨다.
	"11_shape_layout/shape-layout-card":   true,
	"11_shape_layout/shape-layout-nested": true,
	"11_shape_layout/shape-layout-modes":  true,
	// 12_palette 두 장: 색 견본이 슬라이드의 큰 면을 덮는다 — 색을 보는 픽스처이므로 04와
	// 같은 이유로 "가장 흔한 색이 배경"이라는 전제가 깨진다 (여섯 견본 합이 슬라이드의 12%,
	// 흰 라벨 글자는 그 면 **안**에 있어 띠로 뭉친다).
	//
	// 잃는 것과 남는 것: 이 픽스처가 시험하는 것은 줄바꿈이 아니라 색 참조이고, 그 참조는
	// 0 오차로 따로 못 박힌다 (아래 리터럴 색 검사 + render/pptx.TestRenderColorVariants).
	// 좌표는 좌표 대조가 판정한다.
	"12_palette/palette-accents":  true,
	"12_palette/palette-variants": true,
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
func TestFixtureLineBreaksMatchLibreOffice(t *testing.T) {
	RequireChromium(t)
	RequireSoffice(t)
	RequirePdftoppm(t)

	for _, name := range fixtureNames {
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

// stackOverlapRegionDiffMaxRatio는 겹친 영역 크롭의 픽셀 오차 허용 비율이다.
//
// 슬라이드 전체 비율(fixtureDiffOptions·slideDiffOptions)과 갈라 두는 것이 이 항목의 핵심
// 계약이다: 슬라이드는 대부분 배경이라 위아래가 뒤바뀐 겹침 하나는 전체 비율에서 희석된다
// (겹친 면적만큼만 튀므로 z-order 슬라이드의 전체 오차는 0.54%다). 겹친 사각형만 잘라 보면
// 위 도형이 아래 도형으로 바뀐 것이 곧바로 수치가 된다 — 겹친 영역 안쪽은 위 도형의 단색
// 채움이라 두 타깃이 같으면 오차가 잔차(AA·리샘플링)뿐이고, 순서가 뒤집히면 **영역 전체**가
// 다른 색이 되어 수십 %로 튄다.
//
// 값은 측정값에 여유를 더한 것이다 (항목 06의 측정). z-order 슬라이드의 겹친 쌍 셋 중
// **핵심은 muted∩accent1** — 위치 지정(x/y) 자식 위에 정적(at=center) 자식이 문서상 뒤로 얹힌
// 자리다 (항목 04가 position:relative로 문서 순서를 되돌린 그 divergence). 그 영역(136×156px)의
// 측정 오차는 2.63%이고, accent1∩fg(112×102px)는 1.42%다 — 겹친 영역 안쪽은 위 도형의 단색
// 채움이라 잔차는 채움 위 AA와 그림 리샘플링뿐이다. 6%는 거기에 여유를 더한 값이고
// (imageRegionDiffMaxRatio와 같은 값·같은 성질의 잔차다), 순서가 뒤집히면 영역 전체가 다른
// 색이 되어 100%에 가깝게 튀므로(2.63% → ~100%) 이 임계값은 여전히 뒤집힘을 잡는다.
const stackOverlapRegionDiffMaxRatio = 0.06

// stackOverlapRegionMinPixels는 겹친 영역 오차의 **절대 하한 허용값**이다 (다른 픽셀 수) —
// 아주 작은 겹침에서 잔차 몇 픽셀이 큰 비율로 계산되어 방법 문제로 죽는 것을 막는다
// (imageRegionDiffMinPixels·loLineWidthMinPx와 같은 종류의 장치). 순서가 뒤집힌 겹침은
// 영역 전체가 바뀌므로 이 하한에 걸리지 않는다.
const stackOverlapRegionMinPixels = 32

const (
	// stackOverlapInset은 겹친 사각형을 안으로 물리는 폭이다 (px) — 위 도형의 테두리
	// (경로 중앙 1px, w=9525 EMU)와 AA가 크롭을 가로지르지 않게 안쪽 채움만 남긴다.
	stackOverlapInset = 4.0
	// stackOverlapMinSide는 크롭이 뜻을 갖는 최소 변 길이다 (px). 이보다 얇은 겹침은 두 도형이
	// 맞닿는 **색 경계**(예: muted|fg 세로 경계)가 영역을 세로로 가로지르고, 그 경계 한 줄을 두
	// 래스터라이저가 서로 다르게 리샘플링해(그림 리샘플링과 같은 잔차) 얇은 띠에서 큰 비율이 된다
	// — 측정: muted∩fg의 16×66px 슬리버가 그 경계 한 줄(≈129px)로 12%였다. 그 쌍의 z-순서
	// (fg가 위)는 accent1∩fg(112×102px)가 이미 같은 관계로 판정하므로 얇은 슬리버는 신호를 더하지
	// 않는다 — 채움 대비 경계 비중이 작은 굵은 겹침만 남긴다 (핵심 divergence인 muted∩accent1은
	// 136×156px라 남는다). imageRegion이 잰 rect 그대로 자르되 작은 그림을 픽셀 하한으로 거른
	// 것과 같은 자리의 장치다.
	stackOverlapMinSide = 24.0
)

// intersectRect는 두 사각형의 겹친 사각형이다 (겹치지 않으면 빈 사각형).
func intersectRect(a, b geom.Rect) geom.Rect {
	x0, y0 := math.Max(a.X, b.X), math.Max(a.Y, b.Y)
	x1, y1 := math.Min(a.Right(), b.Right()), math.Min(a.Bottom(), b.Bottom())
	if x1 <= x0 || y1 <= y0 {
		return geom.Rect{}
	}
	return geom.Rect{X: x0, Y: y0, W: x1 - x0, H: y1 - y0}
}

// 겹친 도형이 두 타깃에서 **같은 순서로** 겹치는가 — 겹친 사각형만 잘라 잰다.
//
// 이 iteration의 핵심 증거다 (TASKS의 리뷰 포인트 "겹침 순서가 웹과 같은지"의 자동 절반).
// z-order 슬라이드는 `at="center"`인 정적 도형과 `x`/`y`로 놓인 도형을 한 stack에 섞었다 —
// 웹은 positioned를 non-positioned 위로 올리고(칠 순서 규칙) PPTX는 문서 순서로 그리는데,
// 항목 04가 그 둘을 **같은 순서**(문서 순서)로 맞췄다. 두 유도가 갈라지면 겹친 자리의 색이
// 두 타깃에서 달라지고, 슬라이드 전체 비율로는 겹친 면적만큼만 튀어 희석되므로(0.54%) 겹친
// 사각형만 잘라 본다. 절대 하한 픽셀 수를 함께 둔다.
func TestStackFixtureOverlapRegions(t *testing.T) {
	RequireChromium(t)
	RequireSoffice(t)
	RequirePdftoppm(t)

	fixture := fixtureByName(t, stackFixtureName)
	result := fixture.Build(t)
	if result.Geometry == nil {
		t.Fatal("빌드에 지오메트리가 없다")
	}
	const slideID = "z-order"
	index := -1
	for i, slide := range result.Deck.Slides {
		if slide.ID == slideID {
			index = i
		}
	}
	if index < 0 {
		t.Fatalf("슬라이드 %s가 없다", slideID)
	}
	stage := result.Deck.Stage
	shots := RenderPPTX(t, fixture.Output(t, result, ".pptx"), image.Pt(stage.Width, stage.Height))
	browser := NewBrowser(t)
	browser.Load(t, filepath.Join(result.OutDir, web.IndexFile))
	webImg := browser.ScreenshotElement(t, web.SlideIndexSelector(index))
	loImg := shots[index]

	// 겹침 도형을 문서 순서로 모은다 (z-order 슬라이드의 세 rect — 텍스트는 뺀다).
	var boxes []geom.Box
	for _, box := range result.Geometry.Boxes {
		if box.Slide == index && box.El == soml.ShapeElement {
			boxes = append(boxes, box)
		}
	}
	if len(boxes) < 2 {
		t.Fatalf("겹침 도형이 %d개다 — z-order 슬라이드는 겹친 도형이 여럿이어야 한다", len(boxes))
	}

	worst, worstAt, pairs := 0.0, "", 0
	for a := 0; a < len(boxes); a++ {
		for b := a + 1; b < len(boxes); b++ {
			region := intersectRect(boxes[a].Rect, boxes[b].Rect)
			if region.W == 0 {
				continue
			}
			region = geom.Rect{
				X: region.X + stackOverlapInset, Y: region.Y + stackOverlapInset,
				W: region.W - 2*stackOverlapInset, H: region.H - 2*stackOverlapInset,
			}
			if region.W < stackOverlapMinSide || region.H < stackOverlapMinSide {
				continue
			}
			pairs++
			cmp, err := Diff(cropImage(webImg, region), cropImage(loImg, region), DefaultDiffOptions())
			if err != nil {
				t.Fatalf("%s %s∩%s: %v", slideID, boxes[a].ID, boxes[b].ID, err)
			}
			t.Logf("%s %s∩%s %s: %s", slideID, boxes[a].ID, boxes[b].ID, region, cmp)
			if cmp.Ratio > worst {
				worst, worstAt = cmp.Ratio, fmt.Sprintf("%s∩%s", boxes[a].ID, boxes[b].ID)
			}
			if cmp.Ratio > stackOverlapRegionDiffMaxRatio && cmp.Changed > stackOverlapRegionMinPixels {
				t.Errorf("%s %s∩%s: 겹친 영역 픽셀 오차 %.2f%%가 허용 %.2f%%를 넘는다 — 위아래가 뒤바뀌었다면 "+
					"고칠 곳은 오차가 아니라 항목 04의 칠 순서 결정(render/web)과 문서 순서 조립(render/pptx)이다",
					slideID, boxes[a].ID, boxes[b].ID, cmp.Ratio*100, stackOverlapRegionDiffMaxRatio*100)
			}
		}
	}
	if pairs == 0 {
		t.Fatal("겹친 영역이 하나도 없다 — z-order 슬라이드가 겹치지 않는다 (픽스처가 바뀌었다)")
	}
	t.Logf("가장 큰 겹침 영역 오차: %s %.2f%% (허용 %.2f%%, 겹친 쌍 %d개)",
		worstAt, worst*100, stackOverlapRegionDiffMaxRatio*100, pairs)
}
