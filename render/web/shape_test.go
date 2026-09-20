package web

import (
	"fmt"
	"math"
	"strings"
	"testing"

	"github.com/k2sebeom/balpyo/soml"
)

// 이 파일은 **PRD §9.4 표의 도형 행 회귀 장치**다 (layout_test.go가 §7.4 표에 대해 하는 일과
// 같은 자리). 외관 속성 하나가 어느 CSS 선언이 되는지를 표로 적고, 어휘 목록
// (soml.ShapeAppearanceAttrs)을 순회해 "표에 있고 코드에 없는 행"이 남지 않았는지 대조한다.
//
// 12종 기하는 **서로 다른 산출**을 내는지까지 본다 — 같은 모양이 두 프리셋에 붙는 실수는
// 그 테스트로만 잡히고, 산출물을 눈으로 보기 전까지 조용하다.

// shapeCase는 외관 속성 하나가 CSS 선언 하나로 컴파일되는 것을 확인하는 행이다.
type shapeCase struct {
	// attr은 어휘 목록의 속성 이름이다 (커버리지 대조의 열쇠).
	attr string
	name string
	// decl은 슬라이드 원문에 쓸 속성 표기다 (`type`은 행마다 다를 수 있어 함께 적는다).
	decl string
	// want는 나와야 하는 CSS 선언이다.
	want string
	// box면 박스 div의 인라인 style을, 아니면 그림 svg의 인라인 style을 본다.
	box bool
}

// shapeCases는 PRD §9.4 표의 도형 외관 행을 그대로 옮긴 것이다 (표 순서).
func shapeCases() []shapeCase {
	return []shapeCase{
		// variant는 CSS 선언을 스스로 만들지 않는다 — 캐스케이드의 한 단이므로 **그 결과**가
		// CSS에 나오는지로 확인한다 (card variant의 fill은 bg다 — core/theme.go).
		{"variant", "테마 번들이 CSS까지 온다", `variant="card" type="rounded-rect"`,
			"fill: var(--balpyo-color-bg)", false},

		// fill — 토큰은 변수 참조, 리터럴은 #RRGGBB, none은 SVG의 none이다.
		{"fill", "토큰", `type="rect" fill="accent1"`, "fill: var(--balpyo-color-accent1)", false},
		{"fill", "리터럴", `type="rect" fill="#0F65F9"`, "fill: #0F65F9", false},
		{"fill", "none", `type="rect" fill="none"`, "fill: none", false},

		// border — 축약 3조각이 선언 셋으로 갈라진다 (굵기·색·파선).
		{"border", "실선 토큰", `type="rect" border="1 solid muted"`, "stroke: var(--balpyo-color-muted)", false},
		{"border", "굵기", `type="rect" border="3 solid accent1"`, "stroke-width: 3px", false},
		{"border", "리터럴 색", `type="rect" border="2 solid #6B7280"`, "stroke: #6B7280", false},
		{"border", "dashed", `type="rect" border="2 dashed muted"`, "stroke-dasharray: 8px 6px", false},
		{"border", "dotted", `type="rect" border="2 dotted muted"`, "stroke-dasharray: 2px 6px", false},
		{"border", "none", `type="rect" border="none"`, "stroke: none", false},

		// radius — 기하이므로 선언이 아니라 `rx`로 나간다 (프리셋의 조정값 자리).
		{"radius", "토큰", `type="rounded-rect" radius="md"`, `rx="12"`, false},
		{"radius", "수치", `type="rounded-rect" radius="32"`, `rx="32"`, false},

		// shadow — 토큰 참조다 (drop-shadow의 인자를 테마 층이 선언한다).
		{"shadow", "토큰", `type="rect" shadow="soft"`,
			"filter: drop-shadow(var(--balpyo-shadow-soft))", false},

		{"opacity", "0.4", `type="rect" opacity="0.4"`, "opacity: 0.4", false},
		{"opacity", "1", `type="rect" opacity="1"`, "opacity: 1", false},

		// rotate는 **박스**에 걸린다 (자식 텍스트가 함께 돈다 — OOXML xfrm@rot과 같다).
		{"rotate", "각도", `type="rect" rotate="45"`, "--balpyo-shape-rotate: 45deg", true},
		{"rotate", "음수", `type="rect" rotate="-30"`, "--balpyo-shape-rotate: -30deg", true},

		// flip은 **그림**에만 걸린다 (글자는 거울이 되지 않는다 — PowerPoint와 같은 규약).
		{"flip", "h", `type="chevron" flip="h"`, "transform: scaleX(-1)", false},
		{"flip", "v", `type="chevron" flip="v"`, "transform: scaleY(-1)", false},
		{"flip", "hv", `type="chevron" flip="hv"`, "transform: scale(-1, -1)", false},
	}
}

// 외관 속성 → CSS 선언 (PRD §9.4 표 그대로).
func TestShapeAppearanceCompilesToCSS(t *testing.T) {
	for _, tc := range shapeCases() {
		t.Run(tc.attr+"/"+tc.name, func(t *testing.T) {
			out := renderString(t, subsetDeck(t, fmt.Sprintf("<shape %s/>", tc.decl)))
			got := openTagOf(t, out, "s0.0")
			where := "박스"
			if !tc.box {
				got, where = inkOf(t, out, "s0.0"), "그림"
			}
			if !strings.Contains(got, tc.want) {
				t.Errorf("<shape %s>의 %s에 %q가 없다: %s", tc.decl, where, tc.want, got)
			}
		})
	}
}

// **표에 있고 코드에 없는 행이 남지 않는다**: 어휘 목록(soml)을 순회해 위의 표가 도형 외관
// 속성을 전수로 덮는지 본다. 속성이 하나 늘면 이 테스트가 먼저 빨개진다 — 매핑을 잊은 속성은
// "파싱·해석은 되는데 그려지지 않는" 조용한 실패가 된다.
func TestShapeAppearanceVocabularyIsFullyMapped(t *testing.T) {
	covered := map[string]bool{}
	for _, tc := range shapeCases() {
		covered[tc.attr] = true
	}
	for _, attr := range soml.ShapeAppearanceAttrs() {
		if !covered[attr] {
			t.Errorf("도형 외관 속성 %q의 CSS 매핑 검증이 없다 (PRD §9.4 표의 행이 빠졌다)", attr)
		}
	}
	for attr := range covered {
		if !containsString(soml.ShapeAppearanceAttrs(), attr) {
			t.Errorf("도형 외관 속성 %q는 카탈로그에 없다", attr)
		}
	}
	// 박스 속성(pad·크기·비율)은 레이아웃 컨테이너와 **같은 매핑**을 쓴다 (boxSizeCSS) —
	// 그 공유를 아래 TestShapeBoxCSSMatchesContainer가 대조한다. 여기서는 어휘가 둘로
	// 갈라지지 않았는지만 본다.
	for _, attr := range soml.BoxAttrs() {
		if containsString(soml.ShapeAppearanceAttrs(), attr) {
			t.Errorf("박스 속성 %q가 외관 어휘에 섞였다", attr)
		}
	}
}

// 12종 전수: 프리셋마다 기하가 있고, **서로 다른 산출**을 낸다.
func TestShapeTypesAllHaveDistinctGeometry(t *testing.T) {
	seen := map[string]string{}
	for _, shapeType := range soml.ShapeTypes() {
		body := fmt.Sprintf("<shape type=%q w=\"220\" h=\"110\"/>", shapeType)
		out := renderString(t, subsetDeck(t, body))
		ink := inkOf(t, out, "s0.0")
		// 기하는 프리미티브(rect·ellipse) 아니면 경로다 — 둘 다 없으면 그림이 비었다.
		if !strings.Contains(ink, "<rect ") && !strings.Contains(ink, "<ellipse ") && !strings.Contains(ink, "<path ") {
			t.Errorf("type=%q의 그림에 기하가 없다: %s", shapeType, ink)
		}
		// 그림에서 외관 선언을 뺀 부분(기하)만 비교한다 — 모든 도형이 같은 외관을 갖고 있으니
		// 남는 차이가 곧 기하의 차이다.
		geometry := ink[strings.Index(ink, ">"):]
		if other, ok := seen[geometry]; ok {
			t.Errorf("type=%q의 기하가 type=%q와 같다 — 같은 모양이 두 프리셋에 붙었다", shapeType, other)
		}
		seen[geometry] = shapeType
	}
	if len(seen) != len(soml.ShapeTypes()) {
		t.Errorf("기하 %d종, 어휘 %d종", len(seen), len(soml.ShapeTypes()))
	}
}

// 기하 유도표는 **어휘와 같은 순서로 전수**여야 한다 — 프리셋 이름(prstGeom)이 이 표에서
// 나오므로(ShapePreset) 빠진 행은 항목 05에서 "PPTX에만 없는 도형"이 된다.
func TestShapeGeometryTableCoversVocabularyInOrder(t *testing.T) {
	geometries := shapeGeometries()
	types := soml.ShapeTypes()
	if len(geometries) != len(types) {
		t.Fatalf("기하 표 %d행, 어휘 %d종", len(geometries), len(types))
	}
	presets := map[string]string{}
	for i, geom := range geometries {
		if geom.Type != types[i] {
			t.Errorf("기하 표 %d행 = %q, 어휘 = %q (순서가 다르다)", i, geom.Type, types[i])
		}
		if geom.Preset == "" {
			t.Errorf("%q의 OOXML 프리셋 이름이 없다 — 기하의 출처가 표에 없다", geom.Type)
		}
		if other, ok := presets[geom.Preset]; ok {
			t.Errorf("프리셋 %q가 %q와 %q에 둘 다 붙었다", geom.Preset, other, geom.Type)
		}
		presets[geom.Preset] = geom.Type
		if got, ok := ShapePreset(geom.Type); !ok || got != geom.Preset {
			t.Errorf("ShapePreset(%q) = %q, %v — 표와 다르다", geom.Type, got, ok)
		}
		switch geom.Space {
		case spacePixel:
			if geom.Element == "" || geom.Attrs == "" {
				t.Errorf("%q는 px 공간인데 프리미티브가 없다", geom.Type)
			}
			if len(geom.Paths) != 0 {
				t.Errorf("%q는 px 공간인데 경로도 있다", geom.Type)
			}
		case spaceNormalized:
			if len(geom.Paths) == 0 {
				t.Errorf("%q는 정규 공간인데 경로가 없다", geom.Type)
			}
			for _, p := range geom.Paths {
				if !strings.HasPrefix(p.D, "M") {
					t.Errorf("%q의 경로가 moveTo로 시작하지 않는다: %s", geom.Type, p.D)
				}
			}
		}
	}
	if _, ok := ShapePreset("triangle"); ok {
		t.Error("어휘 밖 type에 프리셋이 있다")
	}
}

// 조정값 명세는 **늘림의 보정**이다 — 정규 공간을 늘린 프리셋 중 `ss` 의존이 있는 것만 갖고,
// 값은 유도 주석의 공식과 같아야 한다 (평가는 render/pptx.AdjustValue).
//
// 표를 두 벌로 적는 이유는 shapePresets와 같다: 공식이 통째로 바뀌어도 코드에서 값을 가져오는
// 테스트는 함께 따라가고, 조정값이 틀려도 **파일은 열린다** (프리셋이 조용히 다른 비율로
// 그려진다).
func TestShapeAdjustSpecsMatchStretchedGeometry(t *testing.T) {
	want := map[string][]ShapeAdjust{
		"rounded-rect":  {{Name: "adj", Base: 100000, Axis: AdjustAxisRadius, Max: 50000}},
		"cylinder":      {{Name: "adj", Base: 25000, Axis: AdjustAxisHeight}},
		"chevron":       {{Name: "adj", Base: 50000, Axis: AdjustAxisWidth}},
		"arrow-right":   {{Name: "adj2", Base: 50000, Axis: AdjustAxisWidth}},
		"hexagon":       {{Name: "adj", Base: 25000, Axis: AdjustAxisWidth}},
		"parallelogram": {{Name: "adj", Base: 25000, Axis: AdjustAxisWidth}},
	}
	for _, geom := range shapeGeometries() {
		adjusts, ok := ShapeAdjusts(geom.Type)
		if !ok {
			t.Fatalf("ShapeAdjusts(%q)가 표에 없다", geom.Type)
		}
		if len(adjusts) != len(want[geom.Type]) {
			t.Errorf("%q의 조정값 %d개, want %d개 (%v)", geom.Type, len(adjusts), len(want[geom.Type]), adjusts)
			continue
		}
		for i, adjust := range adjusts {
			if adjust != want[geom.Type][i] {
				t.Errorf("%q의 %d번째 조정값 = %+v, want %+v", geom.Type, i, adjust, want[geom.Type][i])
			}
		}
		// 반경 축을 쓰는 프리셋은 기하에서도 반경을 받는 프리셋이어야 한다 — 두 사실이
		// 갈라지면 웹은 `rx`를 그리고 PPTX는 조정값을 내지 않는(또는 반대의) 도형이 생긴다.
		radiusAxis := false
		for _, adjust := range adjusts {
			if adjust.Axis == AdjustAxisRadius {
				radiusAxis = true
			}
		}
		if radiusAxis != geom.Rounded {
			t.Errorf("%q: 반경 조정값 = %v인데 기하의 Rounded = %v", geom.Type, radiusAxis, geom.Rounded)
		}
	}
	if _, ok := ShapeAdjusts("triangle"); ok {
		// 빈 목록(조정값 없는 프리셋)과 구별되어야 PPTX가 "어휘에 있는데 표에 없는 프리셋"에서
		// 멈출 수 있다.
		t.Error("어휘 밖 type에 조정값 명세가 있다")
	}
}

// 텍스트 영역 표는 **프리셋 정의의 `<rect>`**이고 웹만 읽는다 (ADR 0040).
//
// 표를 두 벌로 적는 이유는 조정값 명세와 같다: 인셋이 틀려도 **파일은 열리고** 도형도 그려진다 —
// 글자가 실루엣을 가로지르거나 LibreOffice와 다른 자리에 앉는 것으로만 드러나고, 그 판정은
// 브라우저와 LibreOffice가 함께 필요한 픽셀 테스트에만 있다 (internal/testkit).
func TestShapeTextRectsMatchPresetDefinitions(t *testing.T) {
	roundRect := 1 - math.Sqrt2/2
	want := map[string]ShapeInsets{
		"rect":          {},
		"rounded-rect":  {Radius: roundRect},
		"ellipse":       {L: 14.6447, T: 14.6447, R: 14.6447, B: 14.6447},
		"diamond":       {L: 25, T: 25, R: 25, B: 25},
		"cylinder":      {T: 25, B: 12.5},
		"chevron":       {},
		"arrow-right":   {T: 25, R: 25, B: 25},
		"hexagon":       {L: 16.6667, T: 16.6667, R: 16.6667, B: 16.6667},
		"parallelogram": {L: 18.75, T: 18.75, R: 18.75, B: 18.75},
		"callout":       {},
		"star":          {L: 30.902, T: 38.1968, R: 30.902, B: 23.6073},
		"cloud":         {L: 13.7824, T: 15.1019, R: 20.8935, B: 19.7361},
	}
	for _, geom := range shapeGeometries() {
		insets, ok := ShapeTextRect(geom.Type)
		if !ok {
			t.Fatalf("ShapeTextRect(%q)가 표에 없다", geom.Type)
		}
		if insets != want[geom.Type] {
			t.Errorf("%q의 텍스트 영역 = %+v, want %+v", geom.Type, insets, want[geom.Type])
		}
		if insets != geom.TextRect {
			t.Errorf("%q: 내보낸 인셋이 표의 열과 다르다", geom.Type)
		}
		// 반경 비례 인셋은 반경을 받는 프리셋에만 있다 — 갈라지면 반경이 없는 도형에서 0이
		// 곱해져 조용히 사라지거나, 반경이 있는데 인셋이 없어 웹만 글자가 모서리에 붙는다.
		if (insets.Radius != 0) != geom.Rounded {
			t.Errorf("%q: 반경 비례 인셋 = %v인데 Rounded = %v", geom.Type, insets.Radius != 0, geom.Rounded)
		}
		if insets.HasPercent() != (insets.L != 0 || insets.T != 0 || insets.R != 0 || insets.B != 0) {
			t.Errorf("%q: HasPercent가 백분율 인셋과 어긋난다 (%+v)", geom.Type, insets)
		}
	}
	if _, ok := ShapeTextRect("triangle"); ok {
		t.Error("어휘 밖 type에 텍스트 영역이 있다")
	}
}

// 되돌림 열(TextCancel)은 **chevron 하나**다 (ADR 0041) — 그 프리셋의 rect가 우리가 내는
// 조정값에서 폭 0 열로 붕괴하기 때문이다. 값은 `x1 = w/2`·`x2 = r − x1`에서 오는 좌우 50%다.
//
// 두 열이 **동시에** 차 있을 수 없다는 것이 이 표의 계약이다: 프리셋의 rect를 쓰기로 했으면
// (TextRect) 되돌릴 것이 없고, 되돌리기로 했으면 그 rect를 안 쓰는 것이다. 둘이 겹치면 웹이
// 인셋을 넣고 PPTX가 같은 인셋을 빼는 상쇄가 생겨 어느 쪽이 참인지 알 수 없다.
func TestShapeTextCancelIsChevronOnly(t *testing.T) {
	for _, geom := range shapeGeometries() {
		cancel, ok := ShapeTextCancel(geom.Type)
		if !ok {
			t.Fatalf("ShapeTextCancel(%q)가 표에 없다", geom.Type)
		}
		want := ShapeInsets{}
		if geom.Type == "chevron" {
			want = ShapeInsets{L: 50, R: 50}
		}
		if cancel != want {
			t.Errorf("%q의 되돌림 = %+v, want %+v", geom.Type, cancel, want)
		}
		if cancel.HasPercent() && geom.TextRect.HasPercent() {
			t.Errorf("%q: 텍스트 영역과 되돌림이 함께 차 있다 (%+v / %+v) — 웹의 인셋과 PPTX의 "+
				"음수 인셋이 상쇄된다", geom.Type, geom.TextRect, cancel)
		}
		if cancel.Radius != 0 {
			t.Errorf("%q: 되돌림에 반경 비례가 있다 (%v) — 반경은 프리셋의 rect를 움직이지 않는다",
				geom.Type, cancel.Radius)
		}
	}
	if _, ok := ShapeTextCancel("triangle"); ok {
		t.Error("어휘 밖 type에 되돌림이 있다")
	}
}

// 어휘 카탈로그의 `soml.ShapeInsetText`(이름 8종)와 이 표의 `TextRect.HasPercent()`(수치)는
// **같은 사실의 두 얼굴**이다 — 값 층이 확정 높이를 요구하는 프리셋과 웹이 절대 배치 래퍼를
// 쓰는 프리셋이 갈라지면, 진단 없이 납작해지는 도형이나 이유 없이 높이를 요구하는 도형이
// 생긴다. 수치가 여기 있고 이름이 저기 있는 이유는 층의 소유다 (core는 렌더러를 볼 수 없다).
func TestShapeInsetTextVocabularyMatchesTheTable(t *testing.T) {
	for _, shapeType := range soml.ShapeTypes() {
		insets, ok := ShapeTextRect(shapeType)
		if !ok {
			t.Fatalf("ShapeTextRect(%q)가 표에 없다", shapeType)
		}
		if got, want := soml.ShapeInsetText(shapeType), insets.HasPercent(); got != want {
			t.Errorf("%q: soml.ShapeInsetText = %v인데 표의 백분율 인셋 = %v (%+v)",
				shapeType, got, want, insets)
		}
	}
	if soml.ShapeInsetText("triangle") {
		t.Error("어휘 밖 type이 백분율 인셋 프리셋이라고 한다")
	}
}

// Px는 백분율을 **잰 박스** 기준으로 푼다 — 그 기준이 CSS(절대 배치)와 테스트(잰 rect)에
// 한 번만 적혀야 두 소비자가 같은 사각형을 본다.
func TestShapeInsetsPxResolvesAgainstTheBox(t *testing.T) {
	insets := ShapeInsets{L: 25, T: 10, R: 50, B: 5, Radius: 0.5}
	left, top, right, bottom := insets.Px(200, 100, 8)
	// 가로는 폭, 세로는 높이 기준이고 반경 몫(0.5×8 = 4px)이 네 변에 같이 더해진다.
	if left != 54 || top != 14 || right != 104 || bottom != 9 {
		t.Errorf("Px = %v %v %v %v, want 54 14 104 9", left, top, right, bottom)
	}
	if (ShapeInsets{}).HasPercent() {
		t.Error("빈 인셋이 백분율을 가진다고 한다")
	}
}

// 텍스트 영역은 **박스 안**에 있다 (인셋이 음수가 아니고 마주 보는 두 인셋의 합이 100% 미만).
//
// 왜 이것만 단정하는가: "네 꼭짓점이 실루엣 안에 있다"는 더 강한 성질은 **프리셋 정의에서
// 성립하지 않는다** — parallelogram의 rect는 사선을 1.6%p 넘어가고(18.75% 대 접점 20.31%),
// diamond·arrow-right·chevron·callout의 변은 실루엣에 정확히 닿는다. 프리셋의 rect는 실루엣의
// 내접 사각형이 아니라 **PowerPoint가 정한 본문 영역**이고, 우리 일은 그것을 웹에서 같게
// 재현하는 것이다 (그래서 이 열은 유도가 아니라 정의를 옮긴 것이고, 실측이 그 짝을 확인한다).
func TestShapeTextRectsStayInsideTheBox(t *testing.T) {
	for _, geom := range shapeGeometries() {
		in := geom.TextRect
		for name, value := range map[string]float64{"l": in.L, "t": in.T, "r": in.R, "b": in.B} {
			if value < 0 {
				t.Errorf("%s: %s 인셋이 음수다 (%v) — 텍스트 영역이 박스 밖으로 나간다", geom.Type, name, value)
			}
		}
		if in.L+in.R >= 100 || in.T+in.B >= 100 {
			t.Errorf("%s: 마주 보는 인셋의 합이 100%%를 넘는다 (%+v) — 텍스트 영역이 뒤집힌다", geom.Type, in)
		}
		// 반경 비례가 0.5를 넘으면 반경이 짧은 변의 절반일 때(완전한 반원) 인셋이 변의 절반을
		// 넘어 영역이 사라진다.
		if in.Radius < 0 || in.Radius > 0.5 {
			t.Errorf("%s: 반경 비례 %v가 0~0.5 밖이다", geom.Type, in.Radius)
		}
	}
}

// 정규 공간의 도형만 늘어난다 — 늘린 공간에서는 선 굵기를 균일하게 유지해야 하고
// (vector-effect), px 공간에는 늘림이 없으므로 그 선언이 없어야 한다 (rounded-rect의 반경이
// 원형으로 남는 이유와 같은 자리다).
func TestShapeStretchedSpaceKeepsStrokeUniform(t *testing.T) {
	for _, tc := range []struct {
		shapeType string
		stretched bool
	}{
		{"rect", false}, {"rounded-rect", false}, {"ellipse", false},
		{"diamond", true}, {"cloud", true}, {"star", true},
	} {
		out := renderString(t, subsetDeck(t, fmt.Sprintf("<shape type=%q/>", tc.shapeType)))
		ink := inkOf(t, out, "s0.0")
		stretched := strings.Contains(ink, `preserveAspectRatio="none"`)
		if stretched != tc.stretched {
			t.Errorf("type=%q: 늘린 공간 = %v, want %v", tc.shapeType, stretched, tc.stretched)
		}
		if got := strings.Contains(ink, `vector-effect="non-scaling-stroke"`); got != tc.stretched {
			t.Errorf("type=%q: non-scaling-stroke = %v, want %v (늘린 공간에서만 필요하다)",
				tc.shapeType, got, tc.stretched)
		}
	}
}

// 토큰은 **참조와 선언이 한 표에서** 나온다 (색·폰트·공간과 같은 계약). 그림자도 같다 —
// 리터럴 수치가 요소 쪽에 남으면 테마를 바꿨을 때 그 도형만 옛 그림자로 남는다.
func TestShapeTokensStayReferences(t *testing.T) {
	// 글자를 담은 도형이다 — 텍스트 영역의 px 인셋은 담을 것이 있을 때만 여백에 더해지므로
	// (ADR 0040) 빈 도형으로는 `calc()` 안의 참조를 볼 수 없다.
	deck := subsetDeck(t, `<shape type="rounded-rect" fill="surface" border="1 solid muted" radius="md" shadow="soft" pad="md"><p>카드</p></shape>`)
	out := renderString(t, deck)
	box, ink := openTagOf(t, out, "s0.0"), inkOf(t, out, "s0.0")
	for _, want := range []string{
		"fill: var(--balpyo-color-surface)",
		"stroke: var(--balpyo-color-muted)",
		"filter: drop-shadow(var(--balpyo-shadow-soft))",
	} {
		if !strings.Contains(ink, want) {
			t.Errorf("그림에 %q가 없다: %s", want, ink)
		}
	}
	// rounded-rect는 프리셋 텍스트 영역의 px 인셋이 여백에 더해지므로 `calc()`다 (ADR 0040) —
	// 토큰은 그 안에서도 **참조로** 남아야 한다. 더해지는 수치는 TestShapeTextRects…가 본다.
	if !strings.Contains(box, "padding: calc(var(--balpyo-space-md) + ") {
		t.Errorf("박스에 공간 토큰 참조가 없다: %s", box)
	}
	// 해석된 값이 요소 쪽에 새면 참조가 아니라 값이 나간 것이다.
	for _, leaked := range []string{"#F3F4F6", "#6B7280", "rgba(", "24px 24px"} {
		if strings.Contains(box+ink, leaked) {
			t.Errorf("토큰이 리터럴로 나갔다 (%q):\n%s\n%s", leaked, box, ink)
		}
	}

	css := between(t, out, "<style>", "</style>")
	if len(deck.Theme.Shadows) == 0 {
		t.Fatal("테마에 그림자 토큰 표가 없다 — 참조와 선언이 같은 표에서 나오는지 볼 수 없다")
	}
	for _, token := range deck.Theme.Shadows {
		// 극좌표(dist·dir)가 직교로 풀리는 자리는 이 함수 하나다. soft = dist 4 · dir 90 →
		// 아래로 4px (y축이 아래로 향한다).
		want := fmt.Sprintf("%s%s: 0px %spx %spx color-mix(in srgb, var(%s%s) %s%%, transparent);",
			shadowVarPrefix, token.Name, formatNumber(token.Dist), formatNumber(token.Blur),
			colorVarPrefix, token.Color.Token, formatNumber(token.Alpha*100))
		if !strings.Contains(css, want) {
			t.Errorf("테마 층에 그림자 선언 %q가 없다:\n%s", want, css)
		}
	}
	// 반경은 **변수가 아니다** — SVG 기하 속성에 var()를 이식성 있게 실을 수 없고, OOXML에도
	// 반경 테마 개념이 없어 두 타깃이 같은 방식으로 값을 굽는다 (shadowVarPrefix의 주석).
	if strings.Contains(css, "--balpyo-radius-") {
		t.Error("반경 토큰이 CSS 변수로 나갔다 — 참조하는 자리가 없는 선언이다")
	}
	if !strings.Contains(ink, `rx="12"`) {
		t.Errorf("반경이 확정 수치로 나가지 않았다: %s", ink)
	}
}

// 회전은 **선언은 엔진 CSS, 각도는 인라인**이다 — 측정 뷰가 그 선언을 무효화할 수 있어야
// 회전 도형의 잰 rect가 회전 전 박스가 된다 (인라인 style은 층 밖에서 이긴다).
func TestShapeRotationIsOverridableInMeasureView(t *testing.T) {
	out := renderString(t, subsetDeck(t, `<shape type="rect" rotate="45"/>`+"\n  "+`<shape type="rect"/>`))
	rotated, plain := openTagOf(t, out, "s0.0"), openTagOf(t, out, "s0.1")
	if !strings.Contains(rotated, fmt.Sprintf("class=%q", shapeClass+" "+shapeRotateClass)) {
		t.Errorf("회전 도형에 %q 클래스가 없다: %s", shapeRotateClass, rotated)
	}
	if strings.Contains(rotated, "transform:") {
		t.Errorf("회전 선언이 인라인으로 새어 나왔다 — 측정 뷰가 덮을 수 없다: %s", rotated)
	}
	if strings.Contains(plain, shapeRotateClass) || strings.Contains(plain, shapeRotateVar) {
		t.Errorf("회전 없는 도형에 회전 표식이 붙었다: %s", plain)
	}
	css := between(t, out, "<style>", "</style>")
	for _, want := range []string{
		fmt.Sprintf(".%s {\n    transform: rotate(var(%s, 0deg));", shapeRotateClass, shapeRotateVar),
		fmt.Sprintf("html[%s=%q] .%s {\n    transform: none;", ViewAttr, MeasureView, shapeRotateClass),
	} {
		if !strings.Contains(css, want) {
			t.Errorf("엔진 CSS에 %q가 없다", want)
		}
	}
}

// 불투명도·대칭은 **그림에만** 걸린다: OOXML은 알파를 solidFill·ln 안에 싣고(도형 전체
// 투명도는 alphaModFix 효과다), flipH/flipV는 기하를 뒤집지만 글자는 읽히는 방향으로 남는다.
// 박스에 걸면 CSS가 자식 텍스트까지 흐리게·거울로 만들어 두 타깃이 갈라진다.
func TestShapeOpacityAndFlipDoNotTouchText(t *testing.T) {
	out := renderString(t, subsetDeck(t,
		`<shape type="chevron" opacity="0.4" flip="h"><p>글자</p></shape>`))
	box, ink, text := openTagOf(t, out, "s0.0"), inkOf(t, out, "s0.0"), openTagOf(t, out, "s0.0.0")
	for _, decl := range []string{"opacity:", "transform: scale"} {
		if strings.Contains(box, decl) {
			t.Errorf("박스에 %q가 걸렸다 — 자식 텍스트까지 따라간다: %s", decl, box)
		}
		if strings.Contains(text, decl) {
			t.Errorf("자식 텍스트에 %q가 걸렸다: %s", decl, text)
		}
		if !strings.Contains(ink, decl) {
			t.Errorf("그림에 %q가 없다: %s", decl, ink)
		}
	}
}

// 세로 앵커는 도형이 세로 flex 컨테이너라는 사실 위에서 주축 분배가 된다 (PRD §9.4의
// `anchor` 행). 값은 **첫 자식**이 정하고, 정한 자식이 없으면 선언이 없다.
func TestShapeAnchorComesFromFirstChild(t *testing.T) {
	for _, tc := range []struct {
		name string
		body string
		want string
	}{
		{"top", `<shape type="rect"><p anchor="top">글자</p></shape>`, "justify-content: start"},
		{"center", `<shape type="rect"><p anchor="center">글자</p></shape>`, "justify-content: center"},
		{"bottom", `<shape type="rect"><p anchor="bottom">글자</p></shape>`, "justify-content: end"},
		// 첫 자식이 정한 값이 이긴다 — 뒤 자식의 다른 앵커는 조용히 버려진다 (도형의
		// bodyPr@anchor가 하나뿐인 것과 같은 제약, ShapeAnchor의 주석).
		{"첫 자식이 이긴다", `<shape type="rect"><h3 anchor="bottom">머리</h3><p anchor="top">글자</p></shape>`,
			"justify-content: end"},
		// 앵커를 적지 않은 첫 자식은 건너뛴다 — 미지정은 빈 값이고 "위"라는 값이 아니다.
		{"미지정은 건너뛴다", `<shape type="rect"><h3>머리</h3><p anchor="center">글자</p></shape>`,
			"justify-content: center"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			box := openTagOf(t, renderString(t, subsetDeck(t, tc.body)), "s0.0")
			if !strings.Contains(box, tc.want) {
				t.Errorf("%s: 박스에 %q가 없다: %s", tc.body, tc.want, box)
			}
		})
	}
	// 앵커가 없는 도형은 선언이 없다 — flex의 기본값이 OOXML bodyPr의 기본 앵커(위)와 같다.
	plain := openTagOf(t, renderString(t, subsetDeck(t, `<shape type="rect"><p>글자</p></shape>`)), "s0.0")
	if strings.Contains(plain, "justify-content") {
		t.Errorf("앵커 없는 도형에 정렬 선언이 붙었다: %s", plain)
	}
}

// 박스 속성(pad·크기·비율)은 레이아웃 컨테이너와 **같은 매핑**에서 나온다 (soml.Box를 두
// 요소가 임베드로 공유하므로 컴파일도 하나여야 한다 — 두 벌이면 크기 하나가 한쪽 요소에서만
// 먹는 일이 생긴다).
func TestShapeBoxCSSMatchesContainer(t *testing.T) {
	const attrs = `w="320" h="180" min-w="100" min-h="80" max-w="400" max-h="300" ratio="16:9" pad="md sm"`
	shape := openTagOf(t, renderString(t, subsetDeck(t, "<shape type=\"rect\" "+attrs+"/>")), "s0.0")
	row := openTagOf(t, renderString(t, subsetDeck(t, "<row "+attrs+"><p>자식</p></row>")), "s0.0")
	for _, want := range []string{
		"padding: var(--balpyo-space-md) var(--balpyo-space-sm) var(--balpyo-space-md) var(--balpyo-space-sm)",
		"width: 320px", "height: 180px", "min-width: 100px", "min-height: 80px",
		"max-width: 400px", "max-height: 300px", "aspect-ratio: 16 / 9",
	} {
		if !strings.Contains(shape, want) {
			t.Errorf("도형 박스에 %q가 없다: %s", want, shape)
		}
		if !strings.Contains(row, want) {
			t.Errorf("컨테이너에 %q가 없다 (같은 매핑이어야 한다): %s", want, row)
		}
	}
}

// 도형은 잰다 — **재는 것과 도형이 되는 것이 처음 일치하는 종류**다. 도형 안 텍스트도 재지만
// (`<li>`처럼) PPTX가 그 rect를 소비하지 않는다. 개수 등식이 geom.Refs와 맞아야 자리
// 맞대기가 성립한다.
func TestShapeIsMeasuredWithItsText(t *testing.T) {
	out := renderString(t, subsetDeck(t, `
  <grid cols="2">
    <shape type="rect"><h3>머리</h3><p>본문</p></shape>
    <shape type="ellipse"/>
  </grid>`))
	for _, tc := range []struct{ id, el string }{
		{"s0.0", "grid"}, {"s0.0.0", "shape"}, {"s0.0.0.0", "h3"}, {"s0.0.0.1", "p"}, {"s0.0.1", "shape"},
	} {
		open := openTagOf(t, out, tc.id)
		if want := fmt.Sprintf("%s=%q", ElAttr, tc.el); !strings.Contains(open, want) {
			t.Errorf("%s의 여는 태그에 %s가 없다: %s", tc.id, want, open)
		}
	}
	// 식별자는 다섯 개다 (그리드 + 도형 둘 + 도형 안 텍스트 둘) — 그림 svg는 세지 않는다.
	if got, want := strings.Count(out, IDAttr+"="), 5; got != want {
		t.Errorf("식별자 개수 = %d, want %d", got, want)
	}
	if strings.Contains(out, fmt.Sprintf("<svg class=%q %s", shapeInkClass, IDAttr)) {
		t.Error("그림 svg에 식별자가 붙었다 — 측정 대상이 아니다")
	}
}

// 배치를 담은 도형은 **배경**이다 (ADR 0049): 레이아웃 컨테이너 자식이 박스 안에서 그대로
// 흐르고, PPTX에서는 그 자식들이 형제 도형이 된다. 웹 쪽 계약은 둘이다 —
//
//   - 구조: 컨테이너가 도형 박스 **안에**, 그림 svg **뒤에** 온다 (흡수 모드처럼 사라지지
//     않는다). 문서 순서가 칠 순서이므로 이 순서가 곧 "글자가 그림 위"다.
//   - 엔진 CSS: 그림이 절대 배치라 흐름 콘텐츠보다 위에 칠해지므로, 도형의 흐름 자식은
//     position: relative로 올라와야 한다. 그 규칙이 `.balpyo-text`에만 걸려 있으면 카드의
//     내용 **전부**가 배경 그림 아래로 사라지고, 그 실패는 산출물을 열어 보기 전까지 조용하다
//     (도형 안에 컨테이너를 넣을 수 있게 된 지금이 그 규칙이 좁아지는 순간이다).
func TestShapeHoldingLayoutLiftsItsChildrenAboveTheInk(t *testing.T) {
	out := renderString(t, subsetDeck(t, `
  <shape type="rounded-rect" pad="lg" w="320" h="240">
    <col gap="sm">
      <text role="heading">카드 제목</text>
      <p>본문 한 줄.</p>
      <text role="caption">푸터</text>
    </col>
  </shape>`))
	for _, tc := range []struct{ id, el string }{
		{"s0.0", "shape"}, {"s0.0.0", "col"},
		{"s0.0.0.0", "text"}, {"s0.0.0.1", "p"}, {"s0.0.0.2", "text"},
	} {
		open := openTagOf(t, out, tc.id)
		if want := fmt.Sprintf("%s=%q", ElAttr, tc.el); !strings.Contains(open, want) {
			t.Errorf("%s의 여는 태그에 %s가 없다: %s", tc.id, want, open)
		}
	}
	// 도형 → 그림 → 컨테이너 순서다 (컨테이너가 도형 밖으로 나가면 배경이 아니라 형제가 된다).
	shapeAt := strings.Index(out, fmt.Sprintf("%s=%q", IDAttr, "s0.0"))
	inkAt := strings.Index(out, fmt.Sprintf("<svg class=%q", shapeInkClass))
	colAt := strings.Index(out, fmt.Sprintf("%s=%q", IDAttr, "s0.0.0"))
	if shapeAt > inkAt || inkAt > colAt {
		t.Errorf("구조 순서가 도형 > 그림 > 컨테이너가 아니다 (%d %d %d)\n%s", shapeAt, inkAt, colAt, out)
	}
	if end := strings.Index(out[colAt:], "</div>"); end < 0 {
		t.Error("컨테이너가 도형 박스 안에서 닫히지 않았다")
	}
	// 엔진 CSS가 올리는 자식 목록에 컨테이너 넷이 있어야 한다 — 어휘를 열고 이 선택자를
	// 잊으면 "파싱은 되는데 보이지 않는" 카드가 된다.
	css := shapeCSS()
	for _, class := range []string{"balpyo-text", rowClass, colClass, gridClass, stackClass} {
		want := "." + shapeClass + " > ." + class
		if !strings.Contains(css, want) {
			t.Errorf("도형 엔진 CSS에 %q가 없다 — 그림 위로 올릴 자식 목록이 좁다", want)
		}
	}
	// 그림과 텍스트 영역 래퍼는 여전히 절대 배치다 (올리는 규칙이 그 둘을 덮으면 배경 그림이
	// 흐름에 끼어들어 잰 rect가 곧 xfrm이라는 계약이 깨진다).
	for _, want := range []string{
		"." + shapeInkClass + " {\n    position: absolute;",
		"." + shapeBodyClass + " {\n    position: absolute;",
	} {
		if !strings.Contains(css, want) {
			t.Errorf("도형 엔진 CSS에 %q가 없다", want)
		}
	}
}

// 도형을 쓰지 않은 덱에는 도형 CSS도, 그림자 토큰 변수도 나가지 않는다 — 골든 불변의 단위
// 테스트 절반이다 (나머지 절반은 픽스처 골든이 바이트로 확인한다).
func TestShapeCSSOnlyWhenShapesUsed(t *testing.T) {
	plain := renderString(t, subsetDeck(t, "<p>도형 없음</p>"))
	for _, absent := range []string{
		"." + shapeClass, "." + shapeInkClass, shapeRotateVar, shadowVarPrefix, MeasureView,
	} {
		if strings.Contains(plain, absent) {
			t.Errorf("도형이 없는 덱의 산출물에 %q가 있다", absent)
		}
	}
	shaped := renderString(t, subsetDeck(t, `<shape type="rect" pad="md"/>`))
	for _, want := range []string{
		"." + shapeClass + " {", "." + shapeInkClass + " {", shadowVarPrefix + "soft",
		// 도형의 `pad`도 공간 토큰을 참조한다 — 레이아웃 태그가 없어도 선언이 나가야 한다
		// (var()는 모르는 변수에 아무 소리도 내지 않으므로 여백이 조용히 0이 된다).
		spaceVarPrefix + "md",
	} {
		if !strings.Contains(shaped, want) {
			t.Errorf("도형을 쓴 덱의 산출물에 %q가 없다", want)
		}
	}
	// 레이아웃 CSS는 여전히 조건부다 — 도형이 배치 어휘를 켜지 않는다.
	if strings.Contains(shaped, ".balpyo-row {") {
		t.Error("도형만 있는 덱에 레이아웃 CSS가 나갔다")
	}
}

// 렌더러는 마지막 관문이다: 해석되지 않은 값이나 어휘 밖의 값이 CSS로 흘러가면 브라우저가
// 조용히 무시하고, 그 침묵은 산출물을 열어 보기 전까지 드러나지 않는다.
func TestShapeRenderRejectsUnresolvedOrUnknown(t *testing.T) {
	for _, tc := range []struct {
		name  string
		body  string
		patch func(*soml.Element)
	}{
		{"외관 없음", `<shape type="rect"/>`, func(shape *soml.Element) { shape.Appearance.Shape = nil }},
		{"모르는 type", `<shape type="rect"/>`, func(shape *soml.Element) {
			shape.Appearance.Shape.Type = "triangle"
		}},
		{"채움 없음", `<shape type="rect"/>`, func(shape *soml.Element) { shape.Appearance.Shape.Fill = nil }},
		{"테두리 없음", `<shape type="rect"/>`, func(shape *soml.Element) { shape.Appearance.Shape.Border = nil }},
		{"굵기 0", `<shape type="rect"/>`, func(shape *soml.Element) {
			shape.Appearance.Shape.Border = &soml.Border{Width: 0, Style: soml.BorderSolid,
				Color: soml.TokenColor("muted")}
		}},
		{"모르는 테두리 스타일", `<shape type="rect"/>`, func(shape *soml.Element) {
			shape.Appearance.Shape.Border = &soml.Border{Width: 1, Style: soml.BorderStyle("double"),
				Color: soml.TokenColor("muted")}
		}},
		{"모르는 그림자 토큰", `<shape type="rect"/>`, func(shape *soml.Element) {
			shape.Appearance.Shape.Shadow = &soml.ShadowToken{Name: "hard"}
		}},
		{"범위 밖 불투명도", `<shape type="rect"/>`, func(shape *soml.Element) {
			opacity := 1.5
			shape.Appearance.Shape.Opacity = &opacity
		}},
		{"모르는 대칭", `<shape type="rect"/>`, func(shape *soml.Element) {
			shape.Appearance.Shape.Flip = soml.Flip("d")
		}},
		{"모르는 앵커", `<shape type="rect"><p>글자</p></shape>`, func(shape *soml.Element) {
			child := shape.Children[0].(*soml.Element)
			child.Appearance.Text.Anchor = soml.TextAnchor("middle")
		}},
		// 반경이 없는 rounded-rect: CSS 기본값은 0이고 OOXML roundRect의 기본 조정값은
		// 16.67%다 — 비어 있으면 두 타깃이 서로 다른 모서리를 그린다.
		{"반경 없는 rounded-rect", `<shape type="rounded-rect"/>`, func(shape *soml.Element) {
			shape.Appearance.Shape.Radius = nil
		}},
		// 반경을 실을 자리가 없는 프리셋에 반경이 남으면 웹만 둥글어지고 PPTX는 각진 채 남는다.
		{"rect의 반경", `<shape type="rect"/>`, func(shape *soml.Element) {
			radius := soml.PxRadius(12)
			shape.Appearance.Shape.Radius = &radius
		}},
		{"정규 공간 프리셋의 반경", `<shape type="diamond"/>`, func(shape *soml.Element) {
			radius := soml.PxRadius(12)
			shape.Appearance.Shape.Radius = &radius
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			deck := subsetDeck(t, tc.body)
			tc.patch(deck.Slides[0].Content[0].(*soml.Element))
			if _, err := New().Render(deck); err == nil {
				t.Errorf("%s는 에러여야 한다", tc.name)
			}
		})
	}
}

func TestShapeRenderIsDeterministic(t *testing.T) {
	body := `<grid cols="2" gap="md">
    <shape type="cloud" fill="accent1" shadow="soft" rotate="15" span="2" h="160"><p anchor="center">구름</p></shape>
    <shape type="cylinder" fill="none" border="2 dashed muted" flip="v"/>
  </grid>`
	a, b := renderString(t, subsetDeck(t, body)), renderString(t, subsetDeck(t, body))
	if a != b {
		t.Error("도형 렌더 결과가 실행마다 다르다")
	}
}

// inkOf는 도형의 그림 svg 여는 태그와 기하를 잘라낸다 — 그림의 인라인 style과 기하를 한
// 문자열로 보고 단정한다.
func inkOf(t *testing.T, out, id string) string {
	t.Helper()
	box := openTagOf(t, out, id)
	at := strings.Index(out, box)
	rest := out[at+len(box):]
	start := strings.Index(rest, "<svg")
	end := strings.Index(rest, "</svg>")
	if start < 0 || end < 0 || start > end {
		t.Fatalf("%s의 그림 svg를 찾을 수 없다:\n%s", id, rest)
	}
	return rest[start : end+len("</svg>")]
}
