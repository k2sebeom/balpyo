package pptx

import (
	"fmt"
	"math"
	"strings"
	"testing"

	"github.com/k2sebeom/balpyo/render/geom"
	"github.com/k2sebeom/balpyo/soml"
)

// 프리셋 도형(`<shape>`)의 OOXML 조립 중 패리티가 보지 않는 것: prstGeom 12종, avLst 조정값,
// 그림자·불투명도·회전·대칭, 인셋 보정.

// shapeStyleOf는 해석된 덱의 첫 콘텐츠 요소의 도형 외관이다 — 에러 경로 테스트가 확정 값을
// 망가뜨려 렌더러의 마지막 관문을 두드릴 때 쓴다.
func shapeStyleOf(t *testing.T, deck *soml.Deck) *soml.ShapeStyle {
	t.Helper()
	elem, ok := deck.Slides[0].Content[0].(*soml.Element)
	if !ok {
		t.Fatalf("첫 콘텐츠가 요소가 아니다: %T", deck.Slides[0].Content[0])
	}
	style := elem.ShapeStyle()
	if style == nil {
		t.Fatal("첫 콘텐츠에 도형 외관이 없다")
	}
	return style
}

// renderShapeError는 도형이 든 덱을 렌더해 에러를 돌려준다 (성공하면 실패).
func renderShapeError(t *testing.T, deck *soml.Deck, rects ...geom.Rect) string {
	t.Helper()
	g := measuredGeometry(t, deck, rects...)
	if _, err := NewWithGeometry(g).Render(deck); err != nil {
		return err.Error()
	}
	t.Fatal("에러가 나야 하는 덱이 렌더되었다")
	return ""
}

// indentedXML은 조립기와 같은 규칙으로 들여쓴 한 줄이다 — 인접한 요소들의 **순서**를 단정하는
// 테스트가 공백을 손으로 세지 않게 한다 (세면 shapeIndent가 바뀔 때 테스트가 이유 없이 깨진다).
func indentedXML(depth int, line string) string {
	return "\n" + shapeIndent + strings.Repeat("  ", depth) + line
}

// shapePresets는 `type` → `prstGeom@prst`의 **전수 기대표**다.
//
// 이 표가 web.ShapePreset의 두 번째 사본인 것은 의도다: SOML 이름과 OOXML 이름이 다른 것이
// 표의 존재 이유이므로(cylinder → can, arrow-right → rightArrow, callout →
// wedgeRectCallout, star → star5) 문자열을 테스트가 직접 적어야 회귀 감시가 된다. 코드에서
// 값을 가져와 비교하면 매핑이 통째로 바뀌어도 테스트가 함께 따라간다 — 그리고 `prst` 이름이
// 틀려도 **파일은 열린다** (뷰어가 모르는 프리셋을 무시한다). 골든 어서션만이 그것을 잡는다.
var shapePresets = map[string]string{
	"rect":          "rect",
	"rounded-rect":  "roundRect",
	"ellipse":       "ellipse",
	"diamond":       "diamond",
	"cylinder":      "can",
	"chevron":       "chevron",
	"arrow-right":   "rightArrow",
	"hexagon":       "hexagon",
	"parallelogram": "parallelogram",
	"callout":       "wedgeRectCallout",
	"star":          "star5",
	"cloud":         "cloud",
}

// 12종 전수 — 어휘의 모든 `type`이 프리셋 지오메트리를 갖고, 도형은 **텍스트 박스로 표시되지
// 않는다** (`txBox="1"`이 붙으면 PowerPoint가 서식 창을 텍스트 박스로 주고 "모양 변경"이
// 사라진다 — 리뷰 항목의 절반이 이 한 속성이다).
func TestRenderShapePresetsCoverVocabulary(t *testing.T) {
	types := soml.ShapeTypes()
	if len(types) != len(shapePresets) {
		t.Fatalf("어휘 %d종, 기대표 %d행 — 프리셋을 더하고 표를 잊었다", len(types), len(shapePresets))
	}
	var src strings.Builder
	src.WriteString("<slide>\n")
	rects := make([]geom.Rect, 0, len(types))
	for i, name := range types {
		fmt.Fprintf(&src, "  <shape type=%q w=\"220\" h=\"110\"/>\n", name)
		rects = append(rects, geom.Rect{X: float64(64 + i*4), Y: 64, W: 220, H: 110})
	}
	src.WriteString("</slide>")
	deck := resolvedDeck(t, src.String())
	xml := slideXML(t, deck, measuredGeometry(t, deck, rects...))

	if got := strings.Count(xml, "<p:sp>"); got != len(types) {
		t.Fatalf("p:sp %d개, want %d개\n%s", got, len(types), xml)
	}
	if strings.Contains(xml, `txBox="1"`) {
		t.Error("도형에 txBox=\"1\"이 붙었다 — PowerPoint가 도형을 텍스트 박스로 준다")
	}
	for _, name := range types {
		want := fmt.Sprintf(`<a:prstGeom prst="%s">`, shapePresets[name])
		if !strings.Contains(xml, want) {
			t.Errorf("type=%q의 산출에 %q가 없다", name, want)
		}
	}
	// 프리셋 이름이 12종 서로 다른지 — 같은 이름이 두 종류에 붙는 실수는 이 검사로만 잡힌다.
	seen := make(map[string]string, len(types))
	for _, name := range types {
		if other, dup := seen[shapePresets[name]]; dup {
			t.Errorf("prst %q가 %s와 %s에 함께 붙었다", shapePresets[name], other, name)
		}
		seen[shapePresets[name]] = name
	}
}

// 늘린 정규 공간을 되돌리는 `avLst` 조정값 — 웹의 기하 표가 명세를 갖고(web.ShapeAdjusts)
// 여기서 잰 박스로 평가된다. 220×110(ss=110)에서 값이 프리셋 기본값과 **다른 것**이 보정이
// 실제로 일어났다는 증거다 (`w`가 ss의 2배이므로 w 기준 조정값은 두 배가 된다).
func TestRenderShapeAdjustments(t *testing.T) {
	tests := []struct {
		shapeType string
		want      []string
	}{
		// radius md(12px) → adj = 100000·12/110 = 10909.09 → 10909.
		{"rounded-rect", []string{`<a:gd name="adj" fmla="val 10909"/>`}},
		// h가 곧 ss이므로 프리셋 기본값 그대로다 (25000).
		{"cylinder", []string{`<a:gd name="adj" fmla="val 25000"/>`}},
		// 노치 = 폭/2가 정확히 100000이라 한 단위 넘긴다 — 텍스트 영역 분기(dx = 0)를 어느
		// 엔진이든 박스 전체로 읽게 한다.
		{"chevron", []string{`<a:gd name="adj" fmla="val 100001"/>`}},
		// adj1은 h 기준이라 보정이 없지만 기본값으로 함께 적는다 — adj2만 적힌 avLst는
		// PowerPoint가 복구하며 도형을 지운다. hexagon의 vf도 같다.
		{"arrow-right", []string{
			`<a:gd name="adj1" fmla="val 50000"/>`,
			`<a:gd name="adj2" fmla="val 100000"/>`,
		}},
		{"hexagon", []string{
			`<a:gd name="adj" fmla="val 50000"/>`,
			`<a:gd name="vf" fmla="val 115470"/>`,
		}},
		{"parallelogram", []string{`<a:gd name="adj" fmla="val 50000"/>`}},
	}
	for _, tc := range tests {
		t.Run(tc.shapeType, func(t *testing.T) {
			deck := resolvedDeck(t, fmt.Sprintf(`<slide>
  <shape type=%q w="220" h="110"/>
</slide>`, tc.shapeType))
			xml := slideXML(t, deck, measuredGeometry(t, deck, geom.Rect{X: 64, Y: 64, W: 220, H: 110}))
			for _, want := range tc.want {
				if !strings.Contains(xml, want) {
					t.Errorf("%s의 avLst에 %q가 없다\n%s", tc.shapeType, want, xml)
				}
			}
			if strings.Contains(xml, "<a:avLst/>") {
				t.Errorf("%s가 빈 avLst를 냈다 — 조정값이 실리지 않았다", tc.shapeType)
			}
		})
	}

	// 조정값이 없는 프리셋은 빈 avLst다 — 프리셋 기본값이 그대로 사는 것이 정상이고
	// (callout의 꼬리·star5의 비율은 w/h 기준이라 늘림과 일치한다), 빈 요소를 내는 것이
	// "조정값을 정하지 않았다"의 표기다.
	for _, shapeType := range []string{"rect", "ellipse", "diamond", "callout", "star", "cloud"} {
		t.Run(shapeType+"/기본값", func(t *testing.T) {
			deck := resolvedDeck(t, fmt.Sprintf(`<slide>
  <shape type=%q w="220" h="110"/>
</slide>`, shapeType))
			xml := slideXML(t, deck, measuredGeometry(t, deck, geom.Rect{X: 64, Y: 64, W: 220, H: 110}))
			if !strings.Contains(xml, "<a:avLst/>") {
				t.Errorf("%s가 조정값을 냈다 — 이 프리셋에는 ss 의존이 없다\n%s", shapeType, xml)
			}
		})
	}
}

// 토큰 색은 도형에서도 `schemeClr` 참조로 남는다 — 채움·테두리·**그림자**까지다 (제품 핵심
// 계약: 내보낸 파일이 PowerPoint의 테마 색 편집을 따라간다).
func TestRenderShapeKeepsTokenColorsAsSchemeClr(t *testing.T) {
	deck := resolvedDeck(t, `<slide>
  <shape type="rect" fill="surface" border="1 solid muted" shadow="soft" w="300" h="160" pad="md">
    <p color="accent1">토큰</p>
  </shape>
</slide>`)
	xml := slideXML(t, deck, measuredGeometry(t, deck,
		geom.Rect{X: 64, Y: 64, W: 300, H: 160},
		geom.Rect{X: 88, Y: 88, W: 252, H: 24},
	))

	for _, want := range []string{
		`<a:schemeClr val="bg2"/>`,     // fill surface
		`<a:schemeClr val="tx2"/>`,     // border muted
		`<a:schemeClr val="tx1">`,      // shadow fg (알파 자식을 갖는다)
		`<a:schemeClr val="accent1"/>`, // 도형 안 글자
		`<a:alpha val="15000"/>`,       // 그림자 토큰의 알파
	} {
		if !strings.Contains(xml, want) {
			t.Errorf("slide1.xml에 %q가 없다\n%s", want, xml)
		}
	}
	if strings.Contains(xml, "<a:srgbClr") {
		t.Error("토큰만 쓴 슬라이드에 srgbClr가 있다 — 색이 리터럴로 눕었다")
	}
	for _, token := range deck.Theme.Colors {
		if strings.Contains(xml, token.RGB) {
			t.Errorf("토큰 %s의 RGB(%s)가 슬라이드 파트에 있다 — 테마 파트에만 있어야 한다",
				token.Name, token.RGB)
		}
	}
}

// 반대 방향: 리터럴 색은 `srgbClr`로 **나오는 것이 정상**이다. 두 방향이 함께 있어야
// "토큰이 눕지 않는다"가 계약이 된다 (한 방향만 보면 색을 전부 리터럴로 내는 구현도 통과한다).
func TestRenderShapeLiteralColorsStayLiteral(t *testing.T) {
	deck := resolvedDeck(t, `<slide>
  <shape type="rect" fill="#0F65F9" border="2 solid #6B7280" w="300" h="140"/>
</slide>`)
	xml := slideXML(t, deck, measuredGeometry(t, deck, geom.Rect{X: 64, Y: 64, W: 300, H: 140}))

	for _, want := range []string{`<a:srgbClr val="0F65F9"/>`, `<a:srgbClr val="6B7280"/>`} {
		if !strings.Contains(xml, want) {
			t.Errorf("slide1.xml에 %q가 없다\n%s", want, xml)
		}
	}
	// accent1과 같은 값이지만 테마 참조가 아니다 — 리터럴이 schemeClr로 올라가면 저자가
	// 고정한 색이 테마 편집에 끌려간다.
	if strings.Contains(xml, "<a:schemeClr") {
		t.Error("리터럴만 쓴 도형에 schemeClr가 있다")
	}
}

// 그림자 → `effectLst > outerShdw`. 극좌표(dist·dir)를 그대로 내고(변환은 웹 쪽이다),
// 색은 토큰 참조 + 알파다. fmtScheme의 effectStyleLst 참조로 내지 않는다 (writeEffect의 근거).
func TestRenderShapeShadow(t *testing.T) {
	deck := resolvedDeck(t, `<slide>
  <shape type="rect" fill="surface" shadow="soft" w="180" h="120"/>
  <shape type="rect" fill="surface" w="180" h="120"/>
</slide>`)
	xml := slideXML(t, deck, measuredGeometry(t, deck,
		geom.Rect{X: 64, Y: 64, W: 180, H: 120},
		geom.Rect{X: 300, Y: 64, W: 180, H: 120},
	))

	// soft = dist 4px(38100 EMU) · dir 90도(5400000) · blur 16px(152400) · fg 알파 0.15.
	want := `<a:outerShdw blurRad="152400" dist="38100" dir="5400000" rotWithShape="1">`
	if !strings.Contains(xml, want) {
		t.Errorf("slide1.xml에 %q가 없다\n%s", want, xml)
	}
	if got := strings.Count(xml, "<a:effectLst>"); got != 1 {
		t.Errorf("effectLst %d개, want 1개 — 그림자를 정하지 않은 도형은 효과를 내지 않는다", got)
	}
	if strings.Contains(xml, "effectRef") {
		t.Error("그림자가 fmtScheme 참조(effectRef)로 나갔다 — 스타일 셋으로는 도형마다 다른 그림자를 실을 수 없다")
	}
}

// 불투명도는 **채움·선·그림자 색 안의 알파**가 된다 (OOXML에는 도형 단위 불투명도가 없다).
// 도형 안 **글자에는 실리지 않는다** — 웹이 `opacity`를 그림 svg에만 걸어 글자를 흐리게 하지
// 않으므로, 글자에 알파를 실으면 두 타깃이 갈라진다 (spSpec.alpha의 결정).
func TestRenderShapeOpacityRidesColorsNotText(t *testing.T) {
	deck := resolvedDeck(t, `<slide>
  <shape type="rect" fill="accent1" border="2 solid muted" shadow="soft" opacity="0.4" w="200" h="120" pad="sm">
    <p color="bg">흐린 도형의 또렷한 글자</p>
  </shape>
</slide>`)
	xml := slideXML(t, deck, measuredGeometry(t, deck,
		geom.Rect{X: 64, Y: 64, W: 200, H: 120},
		geom.Rect{X: 76, Y: 76, W: 176, H: 24},
	))

	spPr, body, ok := strings.Cut(xml, "<p:txBody>")
	if !ok {
		t.Fatalf("txBody가 없다\n%s", xml)
	}
	// 채움·선의 알파는 저자가 준 0.4 그대로, 그림자는 토큰 알파 0.15 × 0.4 = 0.06이다
	// (웹에서 opacity가 filter 결과까지 흐리게 하는 합성 순서와 같다).
	if got := strings.Count(spPr, `<a:alpha val="40000"/>`); got != 2 {
		t.Errorf("채움·선의 알파가 %d개, want 2개\n%s", got, spPr)
	}
	if !strings.Contains(spPr, `<a:alpha val="6000"/>`) {
		t.Errorf("그림자 알파가 도형 불투명도와 곱해지지 않았다\n%s", spPr)
	}
	if strings.Contains(body, "<a:alpha") {
		t.Errorf("도형 안 글자에 알파가 실렸다 — 웹은 글자를 흐리게 하지 않는다\n%s", body)
	}
	if strings.Contains(xml, "alphaModFix") {
		t.Error("도형 전체 투명도(alphaModFix)가 나갔다 — 그쪽은 글자까지 흐려진다")
	}
}

// 회전은 `xfrm@rot`(60000분의 1도)이고 좌표를 건드리지 않는다 — 잰 rect는 회전 **전** 박스이고
// (측정 뷰가 변형을 끈다) 회전 중심이 양 타깃 모두 박스 중심이다.
func TestRenderShapeRotateAndFlip(t *testing.T) {
	tests := []struct {
		attrs string
		want  string
	}{
		{`rotate="45"`, `<a:xfrm rot="2700000">`},
		// 음수·한 바퀴 넘는 각도는 한 바퀴 안으로 접힌다 (Rotation의 근거).
		{`rotate="-90"`, `<a:xfrm rot="16200000">`},
		{`rotate="405"`, `<a:xfrm rot="2700000">`},
		{`rotate="360"`, `<a:xfrm rot="0">`},
		{`flip="h"`, `<a:xfrm flipH="1">`},
		{`flip="v"`, `<a:xfrm flipV="1">`},
		{`flip="hv"`, `<a:xfrm flipH="1" flipV="1">`},
		{`rotate="45" flip="h"`, `<a:xfrm rot="2700000" flipH="1">`},
		// 정하지 않은 변형은 속성을 내지 않는다 — 텍스트 박스와 같은 xfrm이다.
		{`fill="accent1"`, "<a:xfrm>"},
	}
	for _, tc := range tests {
		t.Run(tc.attrs, func(t *testing.T) {
			deck := resolvedDeck(t, fmt.Sprintf(`<slide>
  <shape type="rect" %s w="120" h="120"/>
</slide>`, tc.attrs))
			xml := slideXML(t, deck, measuredGeometry(t, deck, geom.Rect{X: 64, Y: 64, W: 120, H: 120}))
			if !strings.Contains(xml, tc.want) {
				t.Errorf("%s의 산출에 %q가 없다\n%s", tc.attrs, tc.want, xml)
			}
			// 좌표는 잰 값 그대로다 (회전이 bbox를 키우지 않는다).
			if !strings.Contains(xml, `<a:ext cx="1143000" cy="1143000"/>`) {
				t.Errorf("%s가 잰 크기를 바꿨다\n%s", tc.attrs, xml)
			}
		})
	}
}

// chevron의 텍스트 영역은 인셋으로 되돌리지 않는다 — 조정값이 노치를 폭의 절반보다 한 단위 넘겨
// 프리셋 공식(`il = ?: dx x1 l`)의 dx가 음수가 되고, 명세대로 읽든 LibreOffice처럼 읽든 영역이
// 박스 전체다. 음수 인셋으로 폭 0 열을 되돌리던 때는 명세대로 읽는 엔진(OnlyOffice)에서 글자가
// 박스 왼쪽 밖으로 반 폭만큼 밀렸다.
func TestRenderChevronTextIsTheWholeBox(t *testing.T) {
	shape := geom.Rect{X: 64, Y: 64, W: 280, H: 120}
	for _, tc := range []struct {
		name  string
		attrs string
		want  string
	}{
		{"pad 없는 chevron", `type="chevron"`, `lIns="0" tIns="0" rIns="0" bIns="0"`},
		{"pad 있는 chevron", `type="chevron" pad="16"`,
			fmt.Sprintf(`lIns="%d" tIns="%d" rIns="%d" bIns="%d"`, EMU(16), EMU(16), EMU(16), EMU(16))},
	} {
		t.Run(tc.name, func(t *testing.T) {
			deck := resolvedDeck(t, fmt.Sprintf(`<slide>
  <shape %s fill="surface" w="280" h="120">
    <p>단계</p>
  </shape>
</slide>`, tc.attrs))
			xml := slideXML(t, deck, measuredGeometry(t, deck, shape,
				geom.Rect{X: 64, Y: 100, W: 280, H: 30}))
			if !strings.Contains(xml, tc.want) {
				t.Errorf("slide1.xml에 %q가 없다\n%s", tc.want, xml)
			}
			// 프리셋 공식으로 dx = w − 2·x1 = w − 2·ss·adj/100000 < 0이어야 한다.
			var adj int64
			at := strings.Index(xml, `<a:gd name="adj" fmla="val `)
			if at < 0 {
				t.Fatalf("chevron 조정값이 없다\n%s", xml)
			}
			fmt.Sscanf(xml[at+len(`<a:gd name="adj" fmla="val `):], "%d", &adj)
			ss := math.Min(shape.W, shape.H)
			if dx := shape.W - 2*ss*float64(adj)/100000; dx >= 0 {
				t.Errorf("adj %d에서 dx = %v — 텍스트 영역이 엔진마다 갈린다 (음수여야 한다)", adj, dx)
			}
		})
	}
}

// 렌더러는 마지막 관문이다 — 값 층이 이미 막는 조합이라도 여기서 조용히 그럴듯한 산출을
// 내면 웹과 갈라진 파일이 만들어진다.
func TestRenderShapeGuards(t *testing.T) {
	rect := geom.Rect{X: 64, Y: 64, W: 200, H: 100}
	src := `<slide>
  <shape type="rect" fill="accent1" w="200" h="100"/>
</slide>`

	t.Run("외관 미해석", func(t *testing.T) {
		deck := resolvedDeck(t, src)
		deck.Slides[0].Content[0].(*soml.Element).Appearance = nil
		if got := renderShapeError(t, deck, rect); !strings.Contains(got, "core.Resolve") {
			t.Errorf("에러가 해석 누락을 가리키지 않는다: %s", got)
		}
	})
	t.Run("어휘 밖 type", func(t *testing.T) {
		deck := resolvedDeck(t, src)
		shapeStyleOf(t, deck).Type = "triangle"
		if got := renderShapeError(t, deck, rect); !strings.Contains(got, "prstGeom 매핑이 없다") {
			t.Errorf("에러가 프리셋 매핑 부재를 가리키지 않는다: %s", got)
		}
	})
	t.Run("반경을 실을 자리 없음", func(t *testing.T) {
		deck := resolvedDeck(t, src)
		radius := soml.PxRadius(8)
		shapeStyleOf(t, deck).Radius = &radius
		if got := renderShapeError(t, deck, rect); !strings.Contains(got, "반경을 실을 자리가 없다") {
			t.Errorf("에러가 반경 불일치를 가리키지 않는다: %s", got)
		}
	})
	t.Run("rounded-rect의 반경 미해석", func(t *testing.T) {
		deck := resolvedDeck(t, `<slide>
  <shape type="rounded-rect" fill="accent1" w="200" h="100"/>
</slide>`)
		shapeStyleOf(t, deck).Radius = nil
		if got := renderShapeError(t, deck, rect); !strings.Contains(got, "반경이 해석되지 않았다") {
			t.Errorf("에러가 반경 누락을 가리키지 않는다: %s", got)
		}
	})
	t.Run("백분율 pad", func(t *testing.T) {
		deck := resolvedDeck(t, src)
		shapeStyleOf(t, deck).Pad = &soml.Pad{
			Top: soml.Percent(10), Right: soml.Percent(10),
			Bottom: soml.Percent(10), Left: soml.Percent(10),
		}
		if got := renderShapeError(t, deck, rect); !strings.Contains(got, "백분율") {
			t.Errorf("에러가 백분율 pad를 가리키지 않는다: %s", got)
		}
	})
	t.Run("어휘 밖 flip", func(t *testing.T) {
		deck := resolvedDeck(t, src)
		shapeStyleOf(t, deck).Flip = "d"
		if got := renderShapeError(t, deck, rect); !strings.Contains(got, "대칭 어휘 밖") {
			t.Errorf("에러가 대칭 어휘를 가리키지 않는다: %s", got)
		}
	})
	t.Run("어휘 밖 테두리 스타일", func(t *testing.T) {
		deck := resolvedDeck(t, `<slide>
  <shape type="rect" border="2 solid muted" w="200" h="100"/>
</slide>`)
		shapeStyleOf(t, deck).Border.Style = "double"
		if got := renderShapeError(t, deck, rect); !strings.Contains(got, "테두리 스타일") {
			t.Errorf("에러가 테두리 스타일 어휘를 가리키지 않는다: %s", got)
		}
	})
	t.Run("범위 밖 불투명도", func(t *testing.T) {
		deck := resolvedDeck(t, src)
		opacity := 1.5
		shapeStyleOf(t, deck).Opacity = &opacity
		if got := renderShapeError(t, deck, rect); !strings.Contains(got, "0~1 밖") {
			t.Errorf("에러가 불투명도 범위를 가리키지 않는다: %s", got)
		}
	})
	t.Run("도형 안 매핑 없는 자식", func(t *testing.T) {
		deck := resolvedDeck(t, src)
		shape := deck.Slides[0].Content[0].(*soml.Element)
		shape.Children = append(shape.Children, &soml.Element{Name: "ul"})
		// 자식이 늘면 측정 대상도 늘어난다 (리스트 컨테이너는 잰다 — geom.collect).
		if got := renderShapeError(t, deck, rect, rect); !strings.Contains(got, "매핑이 없다") {
			t.Errorf("에러가 매핑 부재를 가리키지 않는다: %s", got)
		}
	})
}

// 앵커 어휘 밖의 값은 조용히 위 정렬이 되는 대신 멈춘다 (algnValue와 같은 태도).
func TestBodyAnchorValue(t *testing.T) {
	tests := []struct {
		anchor soml.TextAnchor
		want   string
	}{
		{"", "t"},
		{soml.AnchorTop, "t"},
		{soml.AnchorCenter, "ctr"},
		{soml.AnchorBottom, "b"},
	}
	for _, tc := range tests {
		got, err := bodyAnchorValue(tc.anchor)
		if err != nil {
			t.Fatalf("bodyAnchorValue(%q): %v", tc.anchor, err)
		}
		if got != tc.want {
			t.Errorf("bodyAnchorValue(%q) = %q, want %q", tc.anchor, got, tc.want)
		}
	}
	if _, err := bodyAnchorValue("middle"); err == nil {
		t.Error("어휘 밖 앵커는 에러여야 한다")
	}
}

// 파선 계열은 웹의 수치와 짝이다 — 어휘 3종 전수 + 어휘 밖.
func TestPrstDashValue(t *testing.T) {
	want := map[soml.BorderStyle]string{
		soml.BorderSolid:  "solid",
		soml.BorderDashed: "dash",
		soml.BorderDotted: "dot",
	}
	for _, style := range soml.BorderStyles() {
		got, err := prstDashValue(style)
		if err != nil {
			t.Fatalf("prstDashValue(%q): %v", style, err)
		}
		if got != want[style] {
			t.Errorf("prstDashValue(%q) = %q, want %q", style, got, want[style])
		}
	}
	if _, err := prstDashValue("double"); err == nil {
		t.Error("어휘 밖 스타일은 에러여야 한다")
	}
}
