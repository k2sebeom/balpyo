package pptx

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/k2sebeom/balpyo/core"
	"github.com/k2sebeom/balpyo/render/geom"
	"github.com/k2sebeom/balpyo/render/web"
	"github.com/k2sebeom/balpyo/soml"
)

// `<image>` → `p:pic` + 미디어 파트 + 관계. 종류·rect·descr은 패리티가 보고, 여기서는 그 밖의
// 것(srcRect 크롭, 외관, 미디어 중복 제거와 관계 번호)을 본다.

// imageFixtureDir은 실제 에셋 바이트를 가진 프로젝트다 — 고유 크기가 커밋된 파일에서 온다.
const imageFixtureDir = "../../testdata/fixtures/05_image"

// 픽스처 에셋. 크롭 표는 비율만 쓰므로(2:1·1:2) 픽셀 수가 바뀌어도 기대값이 움직이지 않는다.
const (
	wideSrc  = `src="assets/wide.png"` // 1200×600 (2:1)
	tallSrc  = `src="assets/tall.png"` // 600×1200 (1:2)
	photoSrc = `src="assets/photo.jpg"`
)

// imageDeck은 그림이 든 덱을 파이프라인과 같은 순서로 만든다 — 파싱 → 해석 → 에셋 패스.
func imageDeck(t *testing.T, sources ...string) *soml.Deck {
	t.Helper()
	deck := resolvedDeck(t, sources...)
	if err := core.ResolveAssets(deck, os.DirFS(imageFixtureDir)).Err(); err != nil {
		t.Fatalf("에셋 해석 실패: %v", err)
	}
	return deck
}

// picOf는 슬라이드 파트에서 p:pic 조각 하나를 잘라낸다.
func picOf(t *testing.T, xml string) string {
	t.Helper()
	start := strings.Index(xml, "<p:pic>")
	if start < 0 {
		t.Fatalf("p:pic이 없다\n%s", xml)
	}
	end := strings.Index(xml[start:], "</p:pic>")
	if end < 0 {
		t.Fatalf("p:pic이 닫히지 않았다\n%s", xml)
	}
	return xml[start : start+end+len("</p:pic>")]
}

// 그림은 p:sp가 아니라 p:pic이다. 채움은 blipFill이 소유하므로 spPr에 채움을 내지 않고, 선
// 없음은 명시한다 (비워 두면 테마의 기본 선이 상속된다). picLocks도 넣지 않는다.
func TestRenderPicStructure(t *testing.T) {
	deck := imageDeck(t, fmt.Sprintf("<slide>\n  <image %s w=\"320\" h=\"160\"/>\n</slide>", wideSrc))
	xml := slideXML(t, deck, measuredGeometry(t, deck, geom.Rect{X: 64, Y: 64, W: 320, H: 160}))

	if strings.Contains(xml, "<p:sp>") {
		t.Errorf("그림이 p:sp를 냈다\n%s", xml)
	}
	pic := picOf(t, xml)
	for _, want := range []string{
		"<p:cNvPicPr/>",
		`<a:blip r:embed="rId3"/>`,
		"<a:stretch>",
		indentedXML(2, "<a:ln>") + indentedXML(3, "<a:noFill/>") + indentedXML(2, "</a:ln>"),
	} {
		if !strings.Contains(pic, want) {
			t.Errorf("p:pic에 %q가 없다\n%s", want, pic)
		}
	}
	if strings.Contains(pic, "<a:solidFill>") {
		t.Errorf("p:pic의 spPr에 채움이 들었다\n%s", pic)
	}
}

// `fit`의 기하 — `contain`은 계산이 없고 `cover`만 크롭이다 (image.go 머리 주석의 표).
func TestRenderPicFitGeometry(t *testing.T) {
	tests := []struct {
		name string
		src  string
		fit  string
		rect geom.Rect
		want string // 기대 srcRect 속성 (빈 문자열이면 요소가 없어야 한다)
	}{
		// contain: 잰 rect가 이미 맞춰진 그림이므로 크롭이 없다 — 박스 비율이 그림과 달라도
		// (웹이 그 박스를 그림에 맞춰 줄였다) 이쪽은 계산하지 않는다.
		{"contain/비율이 달라도 크롭 없음", wideSrc, `fit="contain"`, geom.Rect{X: 64, Y: 64, W: 200, H: 200}, ""},
		{"contain/기본값", wideSrc, "", geom.Rect{X: 64, Y: 64, W: 320, H: 160}, ""},
		// cover: 2:1 그림이 1:1 박스에 들어가면 좌우가 각각 25% 잘린다
		// ((1 − 1/2) / 2 = 0.25).
		{"cover/넓은 그림은 좌우가 잘린다", wideSrc, `fit="cover"`, geom.Rect{X: 64, Y: 64, W: 200, H: 200}, ` l="25000" r="25000"`},
		// 1:2 그림이 1:1 박스에 들어가면 위아래가 잘린다.
		{"cover/좁은 그림은 위아래가 잘린다", tallSrc, `fit="cover"`, geom.Rect{X: 64, Y: 64, W: 200, H: 200}, ` t="25000" b="25000"`},
		// 비율이 같으면 자를 것이 없다 — cover와 contain의 산출이 같아진다.
		{"cover/비율이 같으면 크롭 0", wideSrc, `fit="cover"`, geom.Rect{X: 64, Y: 64, W: 320, H: 160}, ""},
		// 3:2 박스에 2:1 그림 → 보이는 몫 1.5/2 = 0.75, 크롭 12.5%.
		{"cover/부분 크롭", wideSrc, `fit="cover"`, geom.Rect{X: 64, Y: 64, W: 300, H: 200}, ` l="12500" r="12500"`},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			deck := imageDeck(t, fmt.Sprintf("<slide>\n  <image %s %s w=\"320\" h=\"160\"/>\n</slide>", tc.src, tc.fit))
			xml := slideXML(t, deck, measuredGeometry(t, deck, tc.rect))
			pic := picOf(t, xml)
			if tc.want == "" {
				if strings.Contains(pic, "<a:srcRect") {
					t.Errorf("크롭이 없어야 하는데 srcRect가 있다\n%s", pic)
				}
				return
			}
			if !strings.Contains(pic, "<a:srcRect"+tc.want+"/>") {
				t.Errorf("srcRect%s가 없다\n%s", tc.want, pic)
			}
			// 스키마 순서: blip → srcRect → stretch.
			if !(strings.Index(pic, "<a:blip") < strings.Index(pic, "<a:srcRect") &&
				strings.Index(pic, "<a:srcRect") < strings.Index(pic, "<a:stretch>")) {
				t.Errorf("blip → srcRect → stretch 순서가 아니다\n%s", pic)
			}
		})
	}
}

// cover 크롭의 계산 — 고유 종횡비 × 박스 종횡비의 표.
//
// 이 함수가 두 타깃이 같은 영역을 보여준다는 계약의 전부다 — 수와 대칭성을 못 박는다.
func TestCoverCropTable(t *testing.T) {
	tests := []struct {
		name        string
		w, h        int
		rect        geom.Rect
		l, t_, r, b int64
	}{
		// 그림과 박스의 비율이 같으면 크롭이 0이다 (같은 비율의 다른 크기도 마찬가지 —
		// cover는 비율만 본다).
		{"같은 비율", 300, 200, geom.Rect{W: 600, H: 400}, 0, 0, 0, 0},
		{"1:1 그림, 1:1 박스", 64, 64, geom.Rect{W: 200, H: 200}, 0, 0, 0, 0},
		// 넓은 그림 → 좌우. 4:3 박스에 2:1 그림: 보이는 몫 (4/3)/2 = 0.6667, 크롭 16666.7 → 16667.
		{"넓은 그림", 400, 200, geom.Rect{W: 400, H: 300}, 16667, 0, 16667, 0},
		// 좁은 그림 → 위아래. 2:1 박스에 1:1 그림: 보이는 몫 1/2, 크롭 25%.
		{"좁은 그림", 64, 64, geom.Rect{W: 400, H: 200}, 0, 25000, 0, 25000},
		// 극단적인 비율에서도 크롭은 50000(짧은 변 전부)을 넘지 않는다 — 넘으면 보이는 몫이
		// 0이 되어 그림이 사라진다.
		{"극단적으로 긴 그림", 4000, 100, geom.Rect{W: 200, H: 200}, 48750, 0, 48750, 0},
		// 크기 0 박스는 크롭 0이다 (그릴 것이 없다).
		{"크기 0 박스", 400, 200, geom.Rect{}, 0, 0, 0, 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			asset := soml.Asset{Key: "assets/x.png", Format: soml.FormatPNG, Width: tc.w, Height: tc.h}
			got, err := coverCrop(asset, tc.rect)
			if err != nil {
				t.Fatal(err)
			}
			want := srcRect{l: tc.l, t: tc.t_, r: tc.r, b: tc.b}
			if got != want {
				t.Fatalf("coverCrop = %+v, want %+v", got, want)
			}
			// 대칭성: 잘리는 축의 양쪽이 같은 값이다 (중앙 정렬 = CSS의 기본
			// object-position 50%% 50%%). 잔차를 한 변에 몰면 그림이 반 픽셀 밀린다.
			if got.l != got.r || got.t != got.b {
				t.Errorf("크롭이 비대칭이다: %+v", got)
			}
			// 한 축만 잘린다 — 둘 다 자르면 cover가 아니라 확대 부족이다.
			if (got.l != 0 || got.r != 0) && (got.t != 0 || got.b != 0) {
				t.Errorf("두 축이 함께 잘렸다: %+v", got)
			}
			for _, v := range []int64{got.l, got.t, got.r, got.b} {
				if v < 0 || v >= PercentScale/2 {
					t.Errorf("크롭 %d가 0~50000 밖이다: %+v", v, got)
				}
			}
		})
	}

	// 고유 크기가 0인 에셋은 멈춘다 — 0으로 나눈 값이 srcRect에 실리면 파일이 열리지 않는다.
	if _, err := coverCrop(soml.Asset{Key: "assets/x.png"}, geom.Rect{W: 100, H: 100}); err == nil {
		t.Error("고유 크기 0에서 에러가 나지 않았다")
	}
}

// 외관 6종 → OOXML. **도형의 함수를 지나므로** 토큰 색이 `schemeClr`로 남는 것이 여기서도
// 계약이다 (제품 핵심 계약 — 그림에도 선이 있다).
func TestRenderPicAppearance(t *testing.T) {
	tests := []struct {
		attr string
		name string
		decl string
		want []string
		deny []string
	}{
		// variant는 캐스케이드의 한 단이므로 **그 결과**를 본다 (framed = 테두리+반경+그림자).
		{"variant", "테마 번들", `variant="framed"`, []string{
			`<a:ln w="9525">`,
			`<a:schemeClr val="tx2"/>`, // muted 토큰 → 테마 슬롯 (리터럴 RGB가 아니다)
			`<a:prstGeom prst="roundRect">`,
			"<a:outerShdw",
		}, nil},

		// border — 굵기는 EMU 속성, 색은 solidFill, 스타일은 prstDash (도형과 같은 writeLine).
		{"border", "실선 토큰", `border="2 solid muted"`, []string{
			`<a:ln w="19050">`, `<a:schemeClr val="tx2"/>`, `<a:prstDash val="solid"/>`,
		}, []string{"<a:srgbClr"}},
		{"border", "리터럴 색", `border="2 solid #6B7280"`, []string{`<a:srgbClr val="6B7280"/>`}, nil},
		{"border", "dashed", `border="2 dashed muted"`, []string{`<a:prstDash val="dash"/>`}, nil},
		{"border", "none은 선 없음을 명시한다", `border="none"`, []string{"<a:ln>", "<a:noFill/>"}, []string{"<a:prstDash"}},

		// radius — 그림에서는 **기하**가 된다 (프리셋 제약이 없다: 도형과 갈라지는 자리).
		// 320×160(ss=160)에 md(12px) → adj = 100000·12/160 = 7500.
		{"radius", "토큰", `radius="md"`, []string{
			`<a:prstGeom prst="roundRect">`, `<a:gd name="adj" fmla="val 7500"/>`,
		}, []string{`prst="rect"`}},
		{"radius", "수치", `radius="40"`, []string{`<a:gd name="adj" fmla="val 25000"/>`}, nil},
		// 반경이 짧은 변의 절반을 넘으면 프리셋 상한에서 잘린다 (완전한 반원).
		{"radius", "상한", `radius="200"`, []string{`<a:gd name="adj" fmla="val 50000"/>`}, nil},

		// shadow — 토큰의 극좌표를 그대로 낸다 (도형과 같은 writeEffect).
		{"shadow", "토큰", `shadow="soft"`, []string{"<a:outerShdw", `rotWithShape="1"`}, nil},

		// opacity — 그림에서는 `a:alphaModFix`다 (그림에는 흐려질 글자가 없다).
		{"opacity", "0.4", `opacity="0.4"`, []string{`<a:alphaModFix amt="40000"/>`}, nil},
		{"opacity", "1", `opacity="1"`, []string{`<a:alphaModFix amt="100000"/>`}, nil},

		// rotate·flip — xfrm의 속성이다 (도형과 같은 transformOf).
		{"rotate", "각도", `rotate="15"`, []string{`<a:xfrm rot="900000">`}, nil},
		{"rotate", "음수는 한 바퀴 안으로 접힌다", `rotate="-30"`, []string{`<a:xfrm rot="19800000">`}, nil},
		{"flip", "h", `flip="h"`, []string{`<a:xfrm flipH="1">`}, nil},
		{"flip", "v", `flip="v"`, []string{`<a:xfrm flipV="1">`}, nil},
		{"flip", "hv", `flip="hv"`, []string{`<a:xfrm flipH="1" flipV="1">`}, nil},

		// 정하지 않은 외관은 아무 요소도 내지 않는다 (soml.ImageStyle의 nil 계약).
		{"", "정하지 않음", "", nil, []string{"<a:effectLst>", "<a:alphaModFix", `prst="roundRect"`, "rot=", "flipH=", "flipV=", "<a:srcRect"}},
	}
	covered := map[string]bool{}
	for _, tc := range tests {
		covered[tc.attr] = true
		t.Run(tc.attr+"/"+tc.name, func(t *testing.T) {
			deck := imageDeck(t, fmt.Sprintf("<slide>\n  <image %s %s w=\"320\" h=\"160\"/>\n</slide>", wideSrc, tc.decl))
			xml := slideXML(t, deck, measuredGeometry(t, deck, geom.Rect{X: 64, Y: 64, W: 320, H: 160}))
			pic := picOf(t, xml)
			for _, want := range tc.want {
				if !strings.Contains(pic, want) {
					t.Errorf("%s의 산출에 %q가 없다\n%s", tc.decl, want, pic)
				}
			}
			for _, deny := range tc.deny {
				if strings.Contains(pic, deny) {
					t.Errorf("%s의 산출에 %q가 있다\n%s", tc.decl, deny, pic)
				}
			}
		})
	}
	// 어휘 목록과 대조한다 — 속성을 열고 OOXML 매핑을 잊으면 "파싱은 되는데 PPTX에만 없는"
	// 조용한 실패가 된다 (웹의 같은 대조와 짝이다).
	for _, attr := range soml.ImageAppearanceAttrs() {
		if !covered[attr] {
			t.Errorf("외관 속성 %q의 OOXML 매핑 단정이 없다", attr)
		}
	}
}

// 불투명도는 **그림·선·그림자에 함께** 실린다 — 웹의 CSS `opacity`가 요소 전체(테두리 포함)를
// 흐리게 하고 drop-shadow의 결과도 함께 흐려지므로, 그림만 흐려지면 두 타깃이 갈라진다.
func TestRenderPicOpacityReachesLineAndShadow(t *testing.T) {
	deck := imageDeck(t, fmt.Sprintf(`<slide>
  <image %s border="2 solid muted" shadow="soft" opacity="0.5" w="320" h="160"/>
</slide>`, wideSrc))
	pic := picOf(t, slideXML(t, deck, measuredGeometry(t, deck, geom.Rect{X: 64, Y: 64, W: 320, H: 160})))
	if got := strings.Count(pic, `<a:alpha val="50000"/>`); got != 1 {
		t.Errorf("선 색의 알파 50000이 %d개다 (선에 하나)\n%s", got, pic)
	}
	if !strings.Contains(pic, `<a:alphaModFix amt="50000"/>`) {
		t.Errorf("그림에 alphaModFix가 없다\n%s", pic)
	}
	// 그림자 토큰의 알파(soft = 0.15)에 불투명도가 곱해진다 (도형과 같은 shadowAlpha).
	if !strings.Contains(pic, `<a:alpha val="7500"/>`) {
		t.Errorf("그림자 알파에 불투명도가 곱해지지 않았다\n%s", pic)
	}
}

// 그림의 프리셋 이름은 도형 어휘가 가는 곳과 **같은 값**이다 — 두 곳에 문자열을 적었으므로
// (rectPreset·roundRectPreset ↔ web.ShapePreset) 갈라지지 않는지 맞대어 둔다. 이름이 틀려도
// 파일은 열리고 뷰어가 모르는 프리셋을 무시하므로 이 대조만이 그것을 잡는다.
func TestPicPresetsMatchShapeVocabulary(t *testing.T) {
	for _, tc := range []struct{ shapeType, preset string }{
		{"rect", rectPreset},
		{"rounded-rect", roundRectPreset},
	} {
		got, ok := web.ShapePreset(tc.shapeType)
		if !ok {
			t.Fatalf("도형 어휘에 %q가 없다", tc.shapeType)
		}
		if got != tc.preset {
			t.Errorf("%q의 프리셋 = %q, 그림이 쓰는 값 = %q", tc.shapeType, got, tc.preset)
		}
	}
}

// 미디어 파트는 에셋당 하나이고 관계는 슬라이드의 것이다 — 같은 그림을 한 슬라이드에서 두 번,
// 두 슬라이드에서 쓰고, 링크·노트가 함께 있어 관계 번호가 밀리는 자리를 본다. 관계 대상·콘텐츠
// 타입의 짝은 교차 파트 검증(testkit)이 본다.
func TestRenderMediaPartsAndRelationships(t *testing.T) {
	deck := imageDeck(t,
		fmt.Sprintf(`<slide>
  <p><a href="https://a.example">링크</a></p>
  <image %s w="320" h="160"/>
  <image %s w="160" h="80"/>
  <note>발표자 노트</note>
</slide>`, wideSrc, wideSrc),
		fmt.Sprintf("<slide>\n  <image %s w=\"160\" h=\"320\"/>\n  <image %s w=\"320\" h=\"160\"/>\n</slide>", tallSrc, wideSrc),
		fmt.Sprintf("<slide>\n  <image %s w=\"320\" h=\"160\"/>\n</slide>", photoSrc),
	)
	g := measuredGeometry(t, deck,
		geom.Rect{X: 64, Y: 64, W: 1152, H: 36},
		geom.Rect{X: 64, Y: 120, W: 320, H: 160},
		geom.Rect{X: 64, Y: 300, W: 160, H: 80},
		geom.Rect{X: 64, Y: 64, W: 160, H: 320},
		geom.Rect{X: 300, Y: 64, W: 320, H: 160},
		geom.Rect{X: 64, Y: 64, W: 320, H: 160},
	)
	parts, _ := renderPackage(t, NewWithGeometry(g), deck)

	// ① 미디어 파트 수 = 에셋 표 항목 수, 이름은 표 순서(첫 사용)이고 바이트는 표의 것이다.
	if len(deck.Assets) != 3 {
		t.Fatalf("에셋 %d개, want 3 (%v)", len(deck.Assets), deck.Assets.Keys())
	}
	wantParts := map[string]string{
		"ppt/media/image1.png": "assets/wide.png",
		"ppt/media/image2.png": "assets/tall.png",
		"ppt/media/image3.jpg": "assets/photo.jpg",
	}
	mediaCount := 0
	for name := range parts {
		if strings.HasPrefix(name, mediaPartDir) {
			mediaCount++
		}
	}
	if mediaCount != len(wantParts) {
		t.Errorf("미디어 파트 %d개, want %d개 — 같은 그림이 여러 파트가 되면 PowerPoint의 \"그림 바꾸기\"가 한 곳만 고친다", mediaCount, len(wantParts))
	}
	for name, key := range wantParts {
		asset, ok := deck.Assets.Get(key)
		if !ok {
			t.Fatalf("에셋 표에 %q가 없다", key)
		}
		got, ok := parts[name]
		if !ok {
			t.Errorf("미디어 파트 %s가 없다", name)
			continue
		}
		if got != string(asset.Bytes) {
			t.Errorf("%s의 바이트가 %s와 다르다 (엔진은 그림을 고치지 않는다)", name, key)
		}
	}

	if got := strings.Count(parts[contentTypesPart], `<Default Extension="png"`); got != 1 {
		t.Errorf("png Default가 %d개다 — 포맷당 하나여야 한다", got)
	}

	// 관계 번호: 링크가 rId3, 그림이 rId4 (문서 순서 = 한 카운터), 같은 그림 둘은 관계 하나.
	// 노트가 있는 슬라이드이므로 rId2는 노트가 쓴다 — 그 번호를 밟으면 노트나 그림이 사라진다.
	slide1, rels1 := parts[slidePart(1)], parts[slideRelsPart(1)]
	if !strings.Contains(slide1, `<a:hlinkClick r:id="rId3"/>`) {
		t.Errorf("링크가 rId3이 아니다\n%s", slide1)
	}
	if got := strings.Count(slide1, `<a:blip r:embed="rId4"/>`); got != 2 {
		t.Errorf("rId4를 가리키는 blip이 %d개다, want 2 (같은 그림은 관계를 재사용한다)\n%s", got, slide1)
	}
	if got := strings.Count(rels1, "<Relationship "); got != 4 {
		t.Errorf("슬라이드 1의 관계가 %d개다, want 4 (레이아웃·노트·링크·그림)\n%s", got, rels1)
	}
	if !strings.Contains(rels1, `Id="rId2" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/notesSlide"`) {
		t.Errorf("rId2가 노트 슬라이드가 아니다\n%s", rels1)
	}

	// 같은 그림을 쓴 두 번째 슬라이드는 **자기 관계**를 갖는다 (관계는 슬라이드의 것이고
	// 파트는 덱의 것이다) — 번호는 그 슬라이드의 문서 순서다.
	rels2 := parts[slideRelsPart(2)]
	for _, want := range []string{
		`Id="rId3" Type="` + imageRelType + `" Target="../media/image2.png"/>`,
		`Id="rId4" Type="` + imageRelType + `" Target="../media/image1.png"/>`,
	} {
		if !strings.Contains(rels2, want) {
			t.Errorf("슬라이드 2의 rels에 %q가 없다\n%s", want, rels2)
		}
	}
}

// 그림이 먼저 와도 번호는 문서 순서다 — 링크와 그림이 구간을 나눠 쓰지 않는다는 증거다
// (구간을 나누면 첫 그림을 쓰는 시점에 총 링크 수를 알아야 하고, 그것은 순회가 둘이라는 뜻이다).
func TestSlideRelIDsFollowDocumentOrder(t *testing.T) {
	deck := imageDeck(t, fmt.Sprintf(`<slide>
  <image %s w="320" h="160"/>
  <p><a href="https://a.example">링크</a></p>
</slide>`, wideSrc))
	g := measuredGeometry(t, deck,
		geom.Rect{X: 64, Y: 64, W: 320, H: 160},
		geom.Rect{X: 64, Y: 240, W: 1152, H: 36},
	)
	parts, _ := renderPackage(t, NewWithGeometry(g), deck)
	xml, rels := parts[slidePart(1)], parts[slideRelsPart(1)]
	if !strings.Contains(xml, `<a:blip r:embed="rId3"/>`) {
		t.Errorf("먼저 온 그림이 rId3이 아니다\n%s", xml)
	}
	if !strings.Contains(xml, `<a:hlinkClick r:id="rId4"/>`) {
		t.Errorf("나중에 온 링크가 rId4가 아니다\n%s", xml)
	}
	// 노트가 없어도 rId2는 예약된다 — 노트 한 줄을 더한 커밋이 관계 전부를 밀지 않게.
	if strings.Contains(rels, `Id="rId2"`) {
		t.Errorf("노트가 없는데 rId2가 쓰였다\n%s", rels)
	}
}

// `.jpg`와 `.jpeg`가 섞여도 Default는 **하나**다 — 파트 이름을 포맷에서 짓기 때문이다
// (mediaPart). 픽스처에는 `.jpeg` 파일이 없으므로 표를 손으로 만들어 그 사실을 못 박는다.
func TestMediaDefaultsGroupPerFormat(t *testing.T) {
	deck := &soml.Deck{Assets: soml.Assets{
		{Key: "assets/a.jpg", Format: soml.FormatJPEG, Width: 4, Height: 2, Bytes: []byte("a")},
		{Key: "assets/b.jpeg", Format: soml.FormatJPEG, Width: 4, Height: 2, Bytes: []byte("b")},
		{Key: "assets/c.png", Format: soml.FormatPNG, Width: 4, Height: 2, Bytes: []byte("c")},
	}}
	defaults, err := mediaDefaults(deck)
	if err != nil {
		t.Fatal(err)
	}
	want := []mediaDefault{{Ext: "jpg", ContentType: "image/jpeg"}, {Ext: "png", ContentType: "image/png"}}
	if len(defaults) != len(want) {
		t.Fatalf("Default %d개(%+v), want %d개 — 포맷당 하나로 모여야 한다", len(defaults), defaults, len(want))
	}
	for i, w := range want {
		if defaults[i] != w {
			t.Errorf("Default[%d] = %+v, want %+v", i, defaults[i], w)
		}
	}
	// 파트 이름도 포맷의 확장자를 쓴다 — `.jpeg` 원본이 `image2.jpg`가 된다.
	media, err := mediaParts(deck, nil)
	if err != nil {
		t.Fatal(err)
	}
	for i, wantName := range []string{"ppt/media/image1.jpg", "ppt/media/image2.jpg", "ppt/media/image3.png"} {
		if media[i].name != wantName {
			t.Errorf("미디어 파트[%d] = %s, want %s", i, media[i].name, wantName)
		}
	}
}

// 렌더러의 마지막 관문 — 값 층이 이미 막는 것들이지만, 어긴 덱이 조용히 그림 없는 파일이
// 되지 않는다 (도형의 같은 관문과 짝).
func TestRenderPicErrors(t *testing.T) {
	src := fmt.Sprintf("<slide>\n  <image %s w=\"320\" h=\"160\"/>\n</slide>", wideSrc)
	over := 1.5
	tests := []struct {
		name    string
		noAsset bool // core.ResolveAssets를 거치지 않는다
		breakIt func(deck *soml.Deck, style *soml.ImageStyle)
		want    string
	}{
		{"에셋 표가 없다", true, func(*soml.Deck, *soml.ImageStyle) {}, "is not in the asset table"},
		{"src가 해석되지 않았다", false, func(_ *soml.Deck, s *soml.ImageStyle) { s.Src = "" }, "src is unresolved"},
		{"fit이 어휘 밖이다", false, func(_ *soml.Deck, s *soml.ImageStyle) { s.Fit = soml.ImageFit("fill") }, "is not a known fit value"},
		{"테두리 굵기가 0이다", false, func(_ *soml.Deck, s *soml.ImageStyle) { s.Border = &soml.Border{Style: soml.BorderSolid} }, "border width is unresolved"},
		{"불투명도가 범위 밖이다", false, func(_ *soml.Deck, s *soml.ImageStyle) { s.Opacity = &over }, "outside 0–1"},
		{"flip이 어휘 밖이다", false, func(_ *soml.Deck, s *soml.ImageStyle) { s.Flip = soml.Flip("x") }, "is not a known flip value"},
		{"에셋의 바이트가 비었다", false, func(d *soml.Deck, _ *soml.ImageStyle) { d.Assets[0].Bytes = nil }, "has no bytes"},
		{"포맷의 콘텐츠 타입을 모른다", false, func(d *soml.Deck, _ *soml.ImageStyle) { d.Assets[0].Format = soml.ImageFormat("webp") }, "has no file extension"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			deck := resolvedDeck(t, src)
			if !tc.noAsset {
				deck = imageDeck(t, src)
			}
			tc.breakIt(deck, imageStyleOf(t, deck))
			if got := renderShapeError(t, deck, geom.Rect{X: 64, Y: 64, W: 320, H: 160}); !strings.Contains(got, tc.want) {
				t.Errorf("에러 = %q, want %q", got, tc.want)
			}
		})
	}
}

// imageStyleOf는 해석된 덱의 첫 콘텐츠 요소의 이미지 외관이다 (shapeStyleOf의 형제).
func imageStyleOf(t *testing.T, deck *soml.Deck) *soml.ImageStyle {
	t.Helper()
	elem, ok := deck.Slides[0].Content[0].(*soml.Element)
	if !ok {
		t.Fatalf("첫 콘텐츠가 요소가 아니다: %T", deck.Slides[0].Content[0])
	}
	style := elem.ImageStyle()
	if style == nil {
		t.Fatal("첫 콘텐츠에 이미지 외관이 없다")
	}
	return style
}
