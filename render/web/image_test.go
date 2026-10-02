package web

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/k2sebeom/balpyo/core"
	"github.com/k2sebeom/balpyo/soml"
)

// 이 파일은 **PRD §9.4 표의 이미지 행 회귀 장치**다 — 외관 표, `fit` 매핑, 에셋 산출 파일.

// imageFixture는 실제 에셋 바이트를 가진 유일한 픽스처다.
const imageFixture = "05_image"

// imageDeck은 subsetDeck에 **에셋 패스**를 더한다 — 에셋 표가 빈 덱에서 렌더러는 멈춘다.
func imageDeck(t *testing.T, body string) *soml.Deck {
	t.Helper()
	deck := subsetDeck(t, body)
	dir := filepath.Join("..", "..", "testdata", "fixtures", imageFixture)
	if err := core.ResolveAssets(deck, os.DirFS(dir)).Err(); err != nil {
		t.Fatalf("에셋 해석 실패: %v", err)
	}
	return deck
}

// imageBoxOf는 그림의 **박스** 여는 태그를 잘라낸다 (식별자를 갖지 않으므로 `<img>`에서
// 거꾸로 찾는다 — inkOf가 도형에서 하는 일의 반대 방향이다).
func imageBoxOf(t *testing.T, out, id string) string {
	t.Helper()
	img := openTagOf(t, out, id)
	at := strings.Index(out, img)
	if at < 0 {
		t.Fatalf("%s의 <img>를 찾을 수 없다", id)
	}
	head := out[:at]
	start := strings.LastIndex(head, "<div class=\""+imageClass)
	if start < 0 {
		t.Fatalf("%s의 박스 div를 찾을 수 없다:\n%s", id, head)
	}
	end := strings.Index(head[start:], ">")
	if end < 0 {
		t.Fatalf("%s의 박스 여는 태그를 자를 수 없다", id)
	}
	return head[start : start+end+1]
}

// wide는 고유 크기 16×8(2:1)인 픽스처 에셋이다 — 박스와 비율이 다른 그림이어야 `fit`이
// 무엇을 하는지 보인다.
const wideSrc = `src="assets/wide.png"`

// imageCase는 속성 하나가 CSS 선언 하나로 컴파일되는 것을 확인하는 행이다.
type imageCase struct {
	// attr은 어휘 목록의 속성 이름이다 (커버리지 대조의 열쇠).
	attr string
	name string
	// decl은 슬라이드 원문에 쓸 속성 표기다.
	decl string
	// want는 `<img>`에 나와야 하는 CSS 선언이다.
	want string
}

// imageCases는 PRD §9.4 표의 이미지 외관 행을 그대로 옮긴 것이다 (표 순서).
func imageCases() []imageCase {
	return []imageCase{
		// variant는 CSS 선언을 스스로 만들지 않는다 — 캐스케이드의 한 단이므로 **그 결과**가
		// CSS에 나오는지로 확인한다 (framed = 테두리+반경+그림자 — 테마의 image.variants).
		{"variant", "테마 번들이 CSS까지 온다", `variant="framed"`,
			"border-color: var(--balpyo-color-muted)"},

		// border — 축약 3조각이 선언 셋으로 갈라진다. 도형은 SVG `stroke`에 걸고 그림은 CSS
		// `border-*`에 건다 (`<img>`에 stroke는 아무 일도 하지 않는다).
		{"border", "실선 토큰", `border="1 solid muted"`, "border-color: var(--balpyo-color-muted)"},
		{"border", "굵기", `border="3 solid accent1"`, "border-width: 3px"},
		{"border", "리터럴 색", `border="2 solid #6B7280"`, "border-color: #6B7280"},
		{"border", "dashed", `border="2 dashed muted"`, "border-style: dashed"},
		{"border", "dotted", `border="2 dotted muted"`, "border-style: dotted"},
		{"border", "none은 선언을 내지 않는다", `border="none"`, ""},

		// radius — 그림의 모양은 직사각형과 반경으로 끝난다 (프리셋 기하가 없다).
		{"radius", "토큰", `radius="md"`, "border-radius: 12px"},
		{"radius", "수치", `radius="32"`, "border-radius: 32px"},

		// shadow — 토큰 참조다 (drop-shadow의 인자를 테마 층이 선언한다).
		{"shadow", "토큰", `shadow="soft"`, "filter: drop-shadow(var(--balpyo-shadow-soft))"},

		{"opacity", "0.4", `opacity="0.4"`, "opacity: 0.4"},
		{"opacity", "1", `opacity="1"`, "opacity: 1"},

		// rotate·flip은 **한 선언**을 커스텀 프로퍼티로 나눠 쓴다 (요소가 하나뿐이다).
		{"rotate", "각도", `rotate="15"`, "--balpyo-image-rotate: 15deg"},
		{"rotate", "음수", `rotate="-30"`, "--balpyo-image-rotate: -30deg"},
		{"flip", "h", `flip="h"`, "--balpyo-image-flip: scaleX(-1)"},
		{"flip", "v", `flip="v"`, "--balpyo-image-flip: scaleY(-1)"},
		{"flip", "hv", `flip="hv"`, "--balpyo-image-flip: scale(-1, -1)"},
	}
}

// 외관 속성 → CSS 선언 (PRD §9.4 표 그대로).
func TestImageAppearanceCompilesToCSS(t *testing.T) {
	for _, tc := range imageCases() {
		t.Run(tc.attr+"/"+tc.name, func(t *testing.T) {
			out := renderString(t, imageDeck(t, fmt.Sprintf("<image %s %s/>", wideSrc, tc.decl)))
			got := openTagOf(t, out, "s0.0")
			if tc.want == "" { // border="none"은 선언을 내지 않는다.
				if strings.Contains(got, "border-") {
					t.Errorf("<image %s>에 테두리 선언이 남았다: %s", tc.decl, got)
				}
				return
			}
			if !strings.Contains(got, tc.want) {
				t.Errorf("<image %s>에 %q가 없다: %s", tc.decl, tc.want, got)
			}
		})
	}
}

func TestImageAppearanceVocabularyIsFullyMapped(t *testing.T) {
	var covered []string
	for _, tc := range imageCases() {
		covered = append(covered, tc.attr)
	}
	assertCoversCatalog(t, "이미지 외관 속성", covered, soml.ImageAppearanceAttrs())
}

// `fit` → CSS (render/web/image.go의 fit 절): `contain`은 `object-fit`을 내지 않으므로 잰 rect가
// 여백을 포함한 박스가 아니라 그림이다. 인라인 style **전체**를 단정한다 (`max-width: 100%`는
// `width: 100%`를 품는다). 어휘의 모든 값이 여기 한 행 이상이다.
func TestImageFitCompilesToCSS(t *testing.T) {
	const contain = "max-width: 100%; max-height: 100%"
	for _, tc := range []struct{ name, decl, want string }{
		{"cover는 박스를 채우고 자른다", `fit="cover" w="fill" ratio="4:3"`, "width: 100%; height: 100%; object-fit: cover"},
		{"contain은 박스를 넘지 않는다", `fit="contain" w="fill" ratio="4:3"`, contain},
		{"생략하면 contain이다", `w="fill" ratio="4:3"`, contain},
		// 크기 속성이 없거나 한계만 있으면 `fit`은 아무것도 하지 않는다 (soml.ImageAttrs) — cover의
		// `width: 100%`가 나가면 고유 크기 그림이 부모 폭까지 늘어난다.
		{"크기가 없는 cover는 늘리지 않는다", `fit="cover"`, contain},
		{"max-w만 준 cover도 늘리지 않는다", `fit="cover" max-w="400"`, contain},
	} {
		t.Run(tc.name, func(t *testing.T) {
			out := renderString(t, imageDeck(t, fmt.Sprintf("<image %s %s/>", wideSrc, tc.decl)))
			if img := openTagOf(t, out, "s0.0"); !strings.Contains(img, `style="`+tc.want+`">`) {
				t.Errorf("<image %s>의 그림 선언 = %s, want style=%q", tc.decl, img, tc.want)
			}
		})
	}
}

// 요소 하나가 `<div>` 박스 + `<img>`이고, 식별자는 **`<img>`가** 갖는다 (측정 대상 계약).
func TestImageRendersBoxAndInk(t *testing.T) {
	out := renderString(t, imageDeck(t, fmt.Sprintf(`<image %s w="200" h="150" alt="구성도"/>`, wideSrc)))
	box := imageBoxOf(t, out, "s0.0")
	img := openTagOf(t, out, "s0.0")
	if !strings.Contains(box, `<div class="balpyo-image"`) {
		t.Errorf("박스가 엔진 클래스를 갖지 않는다: %s", box)
	}
	if strings.Contains(box, IDAttr) {
		t.Errorf("박스가 식별자를 가졌다 — 재는 것은 그림이다: %s", box)
	}
	// 박스는 저자가 준 크기, 그림은 `fit`과 외관이다.
	for _, want := range []string{"width: 200px", "height: 150px"} {
		if !strings.Contains(box, want) {
			t.Errorf("박스에 %q가 없다: %s", want, box)
		}
	}
	for _, want := range []string{
		`<img class="balpyo-image-ink"`,
		IDAttr + `="s0.0"`,
		ElAttr + `="image"`,
		`src="assets/wide.png"`,
		`alt="구성도"`,
	} {
		if !strings.Contains(img, want) {
			t.Errorf("그림에 %q가 없다: %s", want, img)
		}
	}
	// 고유 크기를 HTML 속성으로 한 번 더 말하지 않는다 (두 곳이 어긋날 수 있다).
	if strings.Contains(img, " width=") || strings.Contains(img, " height=") {
		t.Errorf("그림이 width/height HTML 속성을 냈다: %s", img)
	}
}

// `alt`을 생략하면 **`alt=""`**다 — 속성을 빼는 것과 다른 뜻이다 (장식 이미지의 표기이고,
// 속성이 없으면 스크린 리더가 파일 이름을 읽는다).
func TestImageAltIsAlwaysPresent(t *testing.T) {
	for _, tc := range []struct{ name, decl, want string }{
		{"생략", "", `alt=""`},
		{"빈 값", `alt=""`, `alt=""`},
		{"값", `alt="구성도"`, `alt="구성도"`},
		{"이스케이프", `alt="a & <b>"`, `alt="a &amp; &lt;b&gt;"`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			out := renderString(t, imageDeck(t, fmt.Sprintf("<image %s %s/>", wideSrc, tc.decl)))
			if img := openTagOf(t, out, "s0.0"); !strings.Contains(img, tc.want) {
				t.Errorf("%q가 없다: %s", tc.want, img)
			}
		})
	}
}

// 회전·대칭은 **엔진 CSS의 한 선언**을 나눠 쓴다: 표식 클래스는 둘 중 하나가 있을 때만
// 붙고(변형 없는 그림의 스택 문맥을 바꾸지 않는다), 측정 뷰가 그 선언을 무효화한다.
func TestImageTransformClassAndMeasureView(t *testing.T) {
	for _, tc := range []struct {
		name  string
		decl  string
		class bool
	}{
		{"변형 없음", "", false},
		{"회전", `rotate="15"`, true},
		{"대칭", `flip="h"`, true},
		{"둘 다", `rotate="15" flip="h"`, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			out := renderString(t, imageDeck(t, fmt.Sprintf("<image %s %s/>", wideSrc, tc.decl)))
			img := openTagOf(t, out, "s0.0")
			if got := strings.Contains(img, imageTransformClass); got != tc.class {
				t.Errorf("변형 표식 클래스 = %v, want %v: %s", got, tc.class, img)
			}
		})
	}
	out := renderString(t, imageDeck(t, fmt.Sprintf(`<image %s rotate="15"/>`, wideSrc)))
	// 선언이 엔진 CSS에 있어야 측정 뷰가 덮을 수 있다 (인라인 style은 층 밖에서 이긴다).
	if !strings.Contains(out, ".balpyo-image-transform {\n    transform: rotate(var(--balpyo-image-rotate, 0deg)) var(--balpyo-image-flip, scale(1));") {
		t.Error("회전·대칭 선언이 엔진 CSS에 없다")
	}
	if want := fmt.Sprintf("html[%s=%q] .balpyo-image-transform {\n    transform: none;", ViewAttr, MeasureView); !strings.Contains(out, want) {
		t.Error("측정 뷰가 변형을 무효화하지 않는다 — 잰 rect가 회전된 bbox가 된다")
	}
}

// 박스 속성은 레이아웃 컨테이너와 **같은 함수**에서 나온다 (boxSizeCSS·padValue) — 표기가
// 같은데 컴파일이 두 벌이면 크기 하나가 한쪽 요소에서만 먹는다.
func TestImageBoxCSSMatchesContainer(t *testing.T) {
	const box = `pad="md" w="240" h="180" min-w="100" max-w="400" ratio="4:3"`
	imageOut := renderString(t, imageDeck(t, fmt.Sprintf("<image %s %s/>", wideSrc, box)))
	containerOut := renderString(t, subsetDeck(t, fmt.Sprintf("<col %s><p>자식</p></col>", box)))
	imageBox := imageBoxOf(t, imageOut, "s0.0")
	container := openTagOf(t, containerOut, "s0.0")
	for _, want := range []string{
		"padding: var(--balpyo-space-md) var(--balpyo-space-md) var(--balpyo-space-md) var(--balpyo-space-md)",
		"width: 240px", "height: 180px", "min-width: 100px", "max-width: 400px", "aspect-ratio: 4 / 3",
	} {
		if !strings.Contains(imageBox, want) {
			t.Errorf("그림 박스에 %q가 없다: %s", want, imageBox)
		}
		if !strings.Contains(container, want) {
			t.Errorf("컨테이너에 %q가 없다 — 표기가 갈라졌다: %s", want, container)
		}
	}
}

// 자식 배치는 **박스**에 걸린다 — flex/grid의 항목이 그 박스이므로, `<img>`에 걸면 부모가
// 아는 항목과 선언이 붙은 요소가 어긋난다.
func TestImageChildPlacementLandsOnBox(t *testing.T) {
	out := renderString(t, imageDeck(t, fmt.Sprintf(
		`<grid cols="3"><image %s span="2" basis="360" self="center" order="2" w="fill"/></grid>`, wideSrc)))
	box := imageBoxOf(t, out, "s0.0.0")
	img := openTagOf(t, out, "s0.0.0")
	for _, want := range []string{"flex-basis: 360px", "grid-column: span 2", "align-self: center", "order: 2"} {
		if !strings.Contains(box, want) {
			t.Errorf("박스에 %q가 없다: %s", want, box)
		}
		if strings.Contains(img, want) {
			t.Errorf("그림에 배치 선언 %q가 실렸다: %s", want, img)
		}
	}
}

// 저자의 `style` 탈출구는 박스 선언의 **맨 끝**이다 (appendPlacement의 계약).
func TestImageStyleEscapeHatchIsLast(t *testing.T) {
	out := renderString(t, imageDeck(t, fmt.Sprintf(`<image %s w="200" style="outline: 1px solid red"/>`, wideSrc)))
	box := imageBoxOf(t, out, "s0.0")
	if !strings.HasSuffix(box, `outline: 1px solid red">`) {
		t.Errorf("탈출구가 맨 끝이 아니다: %s", box)
	}
}

// 그림자·공간 토큰 변수는 **참조하는 어휘**가 있는 덱에 나온다 — 도형 없이 그림만 있는 덱에서
// 선언이 빠지면 var()가 조용히 무시되어 그림자와 여백이 사라진다.
func TestImageDeckDeclaresShadowAndSpaceTokens(t *testing.T) {
	out := renderString(t, imageDeck(t, fmt.Sprintf(`<image %s shadow="soft" pad="md" w="200"/>`, wideSrc)))
	if strings.Contains(out, "balpyo-shape") {
		t.Fatal("도형이 없는 덱인데 도형 CSS가 나왔다 — 이 테스트의 전제가 깨졌다")
	}
	for _, want := range []string{shadowVarPrefix + "soft:", spaceVarPrefix + "md:"} {
		if !strings.Contains(out, want) {
			t.Errorf("토큰 선언 %q가 없다", want)
		}
	}
}

// 에셋은 **산출 파일**이다: 경로가 저자가 쓴 정규화 경로와 같고, 바이트가 커밋된 파일과 같다.
func TestRenderEmitsAssetFiles(t *testing.T) {
	deck := imageDeck(t, fmt.Sprintf(`<image %s w="200"/>`, wideSrc))
	files, err := New().Render(deck)
	if err != nil {
		t.Fatal(err)
	}
	want, err := os.ReadFile(filepath.Join("..", "..", "testdata", "fixtures", imageFixture, "assets", "wide.png"))
	if err != nil {
		t.Fatal(err)
	}
	got, ok := files["assets/wide.png"]
	if !ok {
		t.Fatalf("에셋이 산출 파일에 없다: %v", files.Paths())
	}
	if string(got) != string(want) {
		t.Errorf("에셋 바이트가 커밋된 파일과 다르다 (%d바이트, want %d바이트)", len(got), len(want))
	}
	// `<img src>`가 산출 경로와 **같은 문자열**이어야 문서가 그 파일을 가리킨다.
	if img := openTagOf(t, string(files[IndexFile]), "s0.0"); !strings.Contains(img, `src="assets/wide.png"`) {
		t.Errorf("문서가 산출 경로를 가리키지 않는다: %s", img)
	}
	if len(files) != 2 {
		t.Errorf("산출 파일 = %v, want index.html + 에셋 하나", files.Paths())
	}
}

// 같은 그림을 여러 번 쓴 덱에서도 에셋 항목은 **하나**이고, 두 번 렌더하면 에셋까지 바이트가 같다.
func TestRenderFixtureAssetsAreDeduplicatedAndDeterministic(t *testing.T) {
	deck := fixtureDeck(t, imageFixture)
	files, err := New().Render(deck)
	if err != nil {
		t.Fatal(err)
	}
	// 픽스처는 네 에셋을 일곱 슬라이드에서 스물네 번 쓴다 (wide.png 혼자 열다섯 번이다).
	wantPaths := []string{"assets/dot.png", "assets/photo.jpg", "assets/tall.png", "assets/wide.png", IndexFile}
	if got := files.Paths(); strings.Join(got, ",") != strings.Join(wantPaths, ",") {
		t.Errorf("산출 파일 = %v, want %v", got, wantPaths)
	}
	images := strings.Count(string(files[IndexFile]), `<img class="balpyo-image-ink`)
	if images < len(deck.Assets) {
		t.Errorf("그림 %d개, 에셋 %d개 — 픽스처가 같은 에셋을 여러 번 쓰지 않는다", images, len(deck.Assets))
	}
	again, err := New().Render(deck)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range files.Paths() {
		if string(files[name]) != string(again[name]) {
			t.Errorf("%s의 바이트가 실행마다 다르다", name)
		}
	}
}

// 에셋 패스를 거치지 않았거나 `src`가 빈 그림은 **멈춘다** — 깨진 그림은 0×0 rect로 재이고,
// `<img src="">`는 브라우저가 현재 문서를 다시 받는 요청이다.
func TestImageRequiresResolvedAssets(t *testing.T) {
	// 에셋 패스를 돌리지 않은 덱 (표가 비었다).
	deck := subsetDeck(t, fmt.Sprintf("<image %s/>", wideSrc))
	if _, err := New().Render(deck); err == nil {
		t.Fatal("에셋 표가 빈 덱을 렌더했다")
	} else if !strings.Contains(err.Error(), "core.ResolveAssets") {
		t.Errorf("진단이 무엇을 거치지 않았는지 말하지 않는다: %v", err)
	}
	deck = imageDeck(t, fmt.Sprintf("<image %s/>", wideSrc))
	deck.Slides[0].Content[0].(*soml.Element).Appearance.Image.Src = ""
	if _, err := New().Render(deck); err == nil {
		t.Fatal("src가 빈 그림을 렌더했다")
	}
}
