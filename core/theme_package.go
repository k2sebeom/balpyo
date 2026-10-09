package core

import (
	"bytes"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"sort"
	"strconv"
	"strings"

	"github.com/k2sebeom/balpyo/soml"
)

// 이 파일은 **테마 패키지 포맷**의 로더다. **엔진에는 어떤 외관 값도 없다** — 값의 유일한
// 출처는 테마 패키지이고, 그 대가로 로더가 완전성을 요구한다 (theme_require.go).
//
// 포맷은 **데이터 전용**이다 (JSON + CSS + 에셋, JS 없음): `balpyo theme install`로
// 받은 원격 테마가 임의 코드 실행 경로가 되어서는 안 된다. 로더는 fs.FS 위에서 도므로 내장
// 테마(go:embed)와 프로젝트 로컬 테마(os.DirFS)가 같은 코드를 지난다.
//
// tokens.json·components.json이 객체가 아니라 **순서 있는 배열**을 쓰는 이유는 결정성이다
// (JSON 객체 키 순서에 기대지 않는다): 모르는 variant 경고가 나열하는 목록과 PPTX clrScheme의
// 슬롯 순서가 실행마다 같아야 한다. 값·순서를 파일에 못 박는 앵커는
// TestLoadBuiltinMinimal·TestLoadBuiltinComponents다
// (기대값이 테스트에 리터럴로 적혀 있어 tokens.json이 실수로 움직이면 골든보다 먼저 잡힌다).

// builtinThemes는 바이너리에 내장된 공식 테마다 — `balpyo init` 직후 네트워크·설치
// 없이 예쁜 결과가 나오는 것이 첫인상을 지배하므로 기본 경로에서 네트워크를 제거한다.
// 셋 다 같은 계약을 받고(11 레이아웃·모든 슬롯·팔레트 규칙·권장 variant 어휘) 그중 minimal이
// 기본값이다 — **폴백 기준은 아니다** (테마를 섞지 않는다).
//
//go:embed themes
var builtinThemes embed.FS

// themeDir는 테마 패키지들이 사는 디렉토리 이름이다 — 임베드 FS와 **프로젝트 루트에서 같다**:
// 구조가 같아야 외부 테마의 설명이 "이 디렉토리를 프로젝트로 옮긴다"로 끝나고,
// 로더가 두 출처에서 같은 코드를 지난다. (`//go:embed`는 리터럴만 받으므로 위 지시문은 이
// 상수를 쓸 수 없다 — 둘이 갈리면 BuiltinThemes가 즉시 panic한다.)
const themeDir = "themes"

// BuiltinThemes는 내장 테마 이름 목록이다 (알파벳 정렬: editorial·ink·minimal).
//
// 공개하는 이유는 **어휘의 단일 출처**다: `balpyo init --theme`이 받을 수 있는
// 이름이 곧 이 목록이어야 하고, 목록을 CLI에 적어 두면 내장 테마를 늘릴 때 두 곳이
// 갈라져 사용자가 값 층이 읽지 않는 이름을 `balpyo.json`에 갖게 된다.
func BuiltinThemes() []string {
	entries, err := fs.ReadDir(builtinThemes, themeDir)
	if err != nil {
		// 내장 파일은 빌드 시점에 고정이다 — 읽히지 않으면 프로그래머 오류다
		// (loadBuiltinTheme의 panic과 같은 자리).
		panic(fmt.Sprintf("내장 테마 목록을 읽을 수 없다: %v", err))
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	return names
}

// IsBuiltinTheme는 이름이 내장 테마인지다.
func IsBuiltinTheme(name string) bool {
	for _, n := range BuiltinThemes() {
		if n == name {
			return true
		}
	}
	return false
}

// themeManifest는 theme.json의 파싱 결과다.
type themeManifest struct {
	Name string `json:"name"`
	// Extends는 상속할 테마 이름이다 — 내장 테마에서는 비어 있다.
	// 이름은 **프로젝트의 어휘**(내장 ∪ `themes/<이름>/`)에서 풀리므로 그 어휘를 모르는 loadTheme은
	// 비지 않은 값을 진단한다: 상속을 푸는 자리는 loadThemeNamed 하나다.
	Extends string `json:"extends"`
	// Layouts는 이 테마가 구현하는 표준 레이아웃 이름 목록이다. 테마는 표준 카탈로그 11종을
	// **전부** 선언해야 한다 — 빠뜨린 레이아웃을 다른 테마로 폴백하면 한 덱에 두 테마가 섞이고,
	// 그 계약은 TestBuiltinThemesImplementEveryStandardLayout이 지킨다. 테마 자체
	// 레이아웃은 **없다**: `<slide layout>`의 어휘는 soml.SlideLayouts() 11종으로 닫혀 있어
	// (resolver.slideLayout) 목록에 더한 이름은 어떤 슬라이드도 요청할 수 없다.
	Layouts []string `json:"layouts"`
	// Chrome은 덱 크롬(페이지 번호·푸터) 선언이다.
	//
	// 테마가 소유하는 것은 **자리·정렬·역할·제외 규칙**이고, 내용과 켜짐은 `balpyo.json`의
	// footer·pageNumber다. 테마가 어떤 kind를 선언하지 않으면 그 크롬은 값 층에서 경고와
	// 함께 미탑재된다 (슬롯 규칙 그대로) — "테마가 받아야 뜻이 있다".
	Chrome []chromeDeclFile `json:"chrome"`
	// Backgrounds는 레이아웃별 슬라이드 배경이다 (theme_assets.go). 비면 모든 슬라이드가 `bg` 색이다.
	Backgrounds []backgroundFile `json:"backgrounds"`
}

// chromeDeclFile은 theme.json의 크롬 선언 하나다 (원문 문자열). 값 어휘 검사는 loadChrome이
// 하고, 그 뒤로는 타입이 붙은 chromeDecl만 돈다 — 렌더러도 값 층도 원문 문자열을 다시 읽지
// 않는다 (컴포넌트 표가 값 문법을 미리 눕혀 담는 것과 같은 태도).
type chromeDeclFile struct {
	Kind        string   `json:"kind"`
	Align       string   `json:"align"`
	Role        string   `json:"role"`
	SkipLayouts []string `json:"skipLayouts"`
	// Image · Height · Alt는 그림 크롬(logo)의 자리다 — 글자 크롬에는 쓸 수 없다.
	Image  string  `json:"image"`
	Height float64 `json:"height"`
	Alt    string  `json:"alt"`
}

// chromeDecl은 검사가 끝난 크롬 선언 하나다.
type chromeDecl struct {
	kind  soml.ChromeKind
	align soml.TextAlign
	role  soml.TextRole
	// skip은 이 크롬을 싣지 않을 레이아웃 집합이다 (표지·섹션 구분 등). 값 층이 슬라이드마다
	// 이 집합을 보고 확정 모델에서 뺀다 — 렌더러는 규칙을 모른다.
	skip map[soml.SlideLayout]bool
	// image · height · alt는 그림 크롬의 판정된 값이다 (image는 에셋 키).
	image  string
	height float64
	alt    string
}

// themePackage는 로드된 테마 하나다 — 토큰 표(soml.Theme)와 매니페스트를 함께 담는다.
// 컴포넌트 외관(components.json)과 레이아웃 CSS(layouts/*.css)는 뒤 하위 단계가 이 구조체에
// 필드를 더한다.
type themePackage struct {
	manifest themeManifest
	// backgrounds는 레이아웃별 슬라이드 배경이다 (theme.json의 backgrounds).
	backgrounds backgroundDecls
	tokens      soml.Theme
	// components는 로드·변환된 컴포넌트 외관 표다 (components.json). tokens와 달리 값 문법을
	// 캐스케이드 내부 타입으로 이미 눕혀 담는다 — resolveText/Shape/Image가 재변환 없이 읽는다.
	components components
	// layouts는 레이아웃 이름→배치 CSS 원문 맵이다 (layouts/*.css). 값은 아직 컴파일하지 않은
	// 원문이다. 로더는 이 맵의 키 집합이
	// theme.json의 layouts 선언과 정확히 일치함을 대조한다.
	layouts map[string]string
	// slotLayouts는 레이아웃 이름→**슬롯 영역** CSS 원문 맵이다 (layouts/*.slots.css).
	// layouts와 나란히 두지만 파일이 다르다 (`title.slots.css`): 흐름 배치(layouts)와 슬롯 영역은
	// 겨누는 선택자가 달라 한 선언 블록에 섞을 수 없다. 슬롯 영역 파일은 **선택**이라(슬롯이
	// 없는 canvas는 파일이 없다) 로더는 키가 선언된 레이아웃의 부분집합임만 확인한다.
	slotLayouts map[string]string
	// chrome은 검사가 끝난 크롬 선언들이다 (theme.json의 chrome — 선언 순서 그대로).
	// 선언 순서가 곧 슬라이드에 실리는 순서이므로(웹 DOM · PPTX 도형 순서) 맵이 아니라
	// 목록이다: 맵으로 두면 두 타깃의 산출물 순서가 실행마다 달라진다.
	chrome []chromeDecl
	// assets는 상속 체인 전체를 병합한 `assets/` 색인과, 지금까지 쓴 그림의 표다. 테마 선언(배경 · 로고)이
	// 로딩에서 먼저 쓰고, 슬라이드의 `theme:` 그림이 값 층에서 이어 쓴다.
	assets *assetResolver
	// chain은 상속 체인의 테마 이름이다 — **잎부터** (`corp-dark`, `corp`, `minimal`). `theme:` 그림을
	// 찾지 못한 진단이 어느 테마들을 보았는지 말한다.
	chain []string
	// lint는 로컬 테마의 레이아웃 CSS 진단이다 (theme_lint.go). 로드를 멈추는 에러와 달리 경고가
	// 섞이므로 error가 아니라 진단 목록으로 나르고, 값 층(Resolve)이 덱 진단의 맨 앞에 싣는다.
	lint soml.Diagnostics
}

// chromeDecls는 이 테마의 크롬 선언이다 (선언 순서).
func (p themePackage) chromeDecls() []chromeDecl { return p.chrome }

// chromeDecl은 kind로 선언 하나를 찾는다.
func (p themePackage) chromeDeclFor(kind soml.ChromeKind) (chromeDecl, bool) {
	for _, decl := range p.chrome {
		if decl.kind == kind {
			return decl, true
		}
	}
	return chromeDecl{}, false
}

// layoutStyles는 레이아웃 이름 → **정규화된 배치 선언**이다 (soml.Theme.Layouts로 실린다).
// 원문(주석·공백)을 눕히고, 배치 CSS가 없는 레이아웃(주석만 있는 content·canvas 스텁)은
// 제외한다 — 맵에 없다는 것이 곧 "스테이지에 레이아웃 클래스를 붙이지 않는다"는 신호다
// (render/web writeStage). 배치·외관 경계 검사는 lintLayouts의 몫이고, 여기서는 문법이
// 온전한 선언만 정규화해 싣는다 (normalizeDeclarations).
func (p themePackage) layoutStyles() map[string]string {
	out := make(map[string]string, len(p.layouts))
	for name, raw := range p.layouts {
		if decls := normalizeDeclarations(raw); decls != "" {
			out[name] = decls
		}
	}
	return out
}

// slotLayoutStyles는 레이아웃 이름 → **정규화된 슬롯 영역 선언**이다 (soml.Theme.SlotLayouts로
// 실린다). layoutStyles와 같은 태도다 — 원문을 눕히고, 영역이 없는(주석만 있는) 파일은
// 제외한다. 맵에 없다는 것이 곧 "슬롯 모디파이어 CSS를 내지 않는다"는 신호다 (render/web).
func (p themePackage) slotLayoutStyles() map[string]string {
	out := make(map[string]string, len(p.slotLayouts))
	for name, raw := range p.slotLayouts {
		if decls := normalizeDeclarations(raw); decls != "" {
			out[name] = decls
		}
	}
	return out
}

// ── tokens.json 스키마 ──────────────────────────────────────────────────────
// 필드는 소문자 JSON 키에 태그로 붙는다. 배열이라 순서가 파일 그대로 유지된다.

type tokensFile struct {
	Color  []colorTok  `json:"color"`
	Font   []fontTok   `json:"font"`
	Space  []spaceTok  `json:"space"`
	Radius []radiusTok `json:"radius"`
	Shadow []shadowTok `json:"shadow"`
}

type colorTok struct {
	Name string `json:"name"`
	RGB  string `json:"rgb"`
	Slot string `json:"slot"`
	Ref  string `json:"ref"`
}

type fontTok struct {
	Name    string `json:"name"`
	Latin   string `json:"latin"`
	EA      string `json:"ea"`
	Generic string `json:"generic"`
	Scheme  string `json:"scheme"`
}

type spaceTok struct {
	Name string  `json:"name"`
	Px   float64 `json:"px"`
}

type radiusTok struct {
	Name string  `json:"name"`
	Px   float64 `json:"px"`
}

type shadowTok struct {
	Name  string  `json:"name"`
	Dist  float64 `json:"dist"`
	Dir   float64 `json:"dir"`
	Blur  float64 `json:"blur"`
	Color string  `json:"color"`
	Alpha float64 `json:"alpha"`
}

// ── components.json 스키마 (원본 JSON) ──────────────────────────────────────
// 배열이라 순서가 파일 그대로 유지된다 (tokens.json과 같은 이유). optional 필드는 포인터라
// "정하지 않음"(nil)과 0값을 구별한다 — 부분 패치가 아래 단을 지우지 않게 한다.

type componentsFile struct {
	// Comment는 문서용 $comment 키를 허용하기 위한 자리다 — DisallowUnknownFields가 그 외
	// 모든 미지원 키(예: extends·오타)를 진단하되 문서 주석은 통과시킨다 (닫힌 어휘).
	Comment string              `json:"$comment"`
	Text    textComponentsFile  `json:"text"`
	Shape   shapeComponentsFile `json:"shape"`
	Image   imageComponentsFile `json:"image"`
	Table   tableComponentsFile `json:"table"`
	Code    codeComponentsFile  `json:"code"`
}

// codeComponentsFile은 코드 블록의 테마 절이다. 표가 **종류 → 색 토큰**이고,
// `tabSize`는 값 층이 탭을 펼 때 쓰는 칼럼 간격이다 — 둘 다 테마의 것이다: 코드의 색은
// 팔레트의 일부이고, 탭 폭은 코드 슬라이드의 리듬이다.
type codeComponentsFile struct {
	// Comment는 문서용 $comment 자리다 (componentsFile.Comment와 같은 이유).
	Comment string          `json:"$comment"`
	TabSize int             `json:"tabSize"`
	Tokens  []codeTokenFile `json:"tokens"`
}

// codeTokenFile은 종류 하나의 색이다. 색은 **토큰 이름**만 받는다 (리터럴 hex는 에러 —
// 참조가 끊기면 PowerPoint의 테마 색 편집이 코드 블록만 두고 간다).
type codeTokenFile struct {
	Class string `json:"class"`
	Color string `json:"color"`
}

type textComponentsFile struct {
	// Comment는 문서용 $comment 자리다 (componentsFile.Comment와 같은 이유) — 범주마다 여는
	// 이유는 값의 근거가 값 옆에 있어야 한다는 것이다: 역할 표의 수치를 고른 근거를 파일 맨
	// 위 한 덩어리에 몰아 두면 테마를 고치는 사람이 그 문장을 다시 찾지 않는다.
	Comment           string                     `json:"$comment"`
	Roles             []textRoleFile             `json:"roles"`
	ComponentDefaults []textComponentDefaultFile `json:"componentDefaults"`
	Variants          []textVariantFile          `json:"variants"`
}

type textRoleFile struct {
	Name       string  `json:"name"`
	Font       string  `json:"font"`
	Size       float64 `json:"size"`
	Weight     int     `json:"weight"`
	LineHeight float64 `json:"lineHeight"`
	Color      string  `json:"color"`
	Align      string  `json:"align"`
}

// textComponentDefaultFile은 컴포넌트 기본값 한 줄이다 — 주인은 **태그**(`tag`)이거나 레이아웃의
// **슬롯**(`layout` + `slot`)이다. 둘 중 하나만 쓴다.
//
// 슬롯이 주인이 될 수 있는 이유: 크기를 정하는 것이 역할이 아니라 자리인 경우가 있다. 인용
// 레이아웃의 `quote` 슬롯은 한 문장을 크게 세우는 자리이지만, 흐름 속 `<blockquote>`는 문단 사이의
// 인용이라 본문 크기여야 한다 — 태그 기본값으로는 둘을 가를 수 없다. 키에 레이아웃이 들어가는
// 이유는 슬롯 이름이 레이아웃마다 겹치기 때문이다 (`body`는 content · image-text · stat에 있다).
type textComponentDefaultFile struct {
	Tag    string   `json:"tag"`
	Layout string   `json:"layout"`
	Slot   string   `json:"slot"`
	Size   *float64 `json:"size"`
	Weight *int     `json:"weight"`
	Color  *string  `json:"color"`
}

// key는 기본값 한 줄의 주인 이름이다 — 상속 병합과 로드가 같은 열쇠를 쓴다.
func (d textComponentDefaultFile) key() string {
	if d.Slot != "" || d.Layout != "" {
		return slotDefaultKey(soml.SlideLayout(d.Layout), d.Slot)
	}
	return d.Tag
}

// slotDefaultKey는 슬롯 기본값 표의 열쇠다.
func slotDefaultKey(layout soml.SlideLayout, slot string) string {
	return string(layout) + "/" + slot
}

type textVariantFile struct {
	Name   string   `json:"name"`
	Size   *float64 `json:"size"`
	Weight *int     `json:"weight"`
	Color  *string  `json:"color"`
}

type shapeComponentsFile struct {
	Comment  string             `json:"$comment"`
	Default  shapeStyleFile     `json:"default"`
	Variants []shapeVariantFile `json:"variants"`
}

type shapeStyleFile struct {
	Fill   string `json:"fill"`
	Border string `json:"border"`
	Radius string `json:"radius"`
}

type shapeVariantFile struct {
	Name   string  `json:"name"`
	Fill   *string `json:"fill"`
	Border *string `json:"border"`
	Radius *string `json:"radius"`
	Shadow *string `json:"shadow"`
}

type imageComponentsFile struct {
	Comment  string             `json:"$comment"`
	Default  imageStyleFile     `json:"default"`
	Variants []imageVariantFile `json:"variants"`
}

type imageStyleFile struct{}

type imageVariantFile struct {
	Name   string  `json:"name"`
	Border *string `json:"border"`
	Radius *string `json:"radius"`
	Shadow *string `json:"shadow"`
}

// tableComponentsFile은 표 셀 외관 원본 스키마다. shape/image와 달리 `variants`
// 배열이 없다: 표의 유일한 변형(줄무늬)은 프레임 `<table variant="striped">`가 켜는 셀
// 채움이고(stripe), 셀 단위 variant는 v1에 없다 (soml/table.go의 근거). 그래서 표는 세 자리만
// 갖는다 — 모든 셀의 기본값(cell), `<th>`에 얹는 델타(header), 줄무늬 채움(stripe).
type tableComponentsFile struct {
	Comment string        `json:"$comment"`
	Cell    cellStyleFile `json:"cell"`
	Header  cellStyleFile `json:"header"`
	// Stripe는 줄무늬 행의 셀 채움 토큰이다 (`none` 또는 색 토큰). 빈 문자열이면 줄무늬가
	// 없는 테마다 — variant="striped"가 아무 채움도 얹지 않는다 (모르는 variant가 폴백하는
	// 것과 같은 태도).
	Stripe string `json:"stripe"`
}

// cellStyleFile은 셀 외관 한 벌의 원본이다 — 박스(fill·border)와 텍스트(color·weight)를 함께
// 담는다 (soml.CellStyle과 같은 모양). 빈 문자열/nil이 "정하지 않음"이다 (shapeStyleFile과
// 같은 계약). align·anchor를 열지 않는 이유: 테마가 정하는 것은 헤더/본문의 **성질**(강조색·
// 굵기)이고, 정렬은 저자가 셀마다 `<td align>`으로 말하는 배치 결정이다.
type cellStyleFile struct {
	Fill   string `json:"fill"`
	Border string `json:"border"`
	Color  string `json:"color"`
	Weight *int   `json:"weight"`
}

// ── 변환된 컴포넌트 표 (캐스케이드 내부 타입) ───────────────────────────────
// 캐스케이드가 이미 쓰는 타입 그대로 담는다 — 그래야 resolveText/Shape/Image가 재변환 없이
// 읽는다 (값 문법을 렌더 때마다 다시 읽는 자리가 생기지 않는다).

// textVariant·shapeVariant·imageVariant는 테마가 이름 붙인 외관 번들 하나다 (요소 종류별로
// 표가 셋이다 — 같은 이름이 세 표에 있어도 무관하고, 요소가 어느 표를 볼지 정한다:
// resolver.textVariantPatch · shapeVariantPatch · imageVariantPatch).
type textVariant struct {
	name  string
	patch textPatch
}

type shapeVariant struct {
	name  string
	patch shapePatch
}

type imageVariant struct {
	name  string
	patch imagePatch
}

type components struct {
	textRoles             map[soml.TextRole]soml.TextStyle
	textComponentDefaults map[string]textPatch
	// textSlotDefaults는 레이아웃 슬롯의 기본값이다 (열쇠 slotDefaultKey) — 태그 기본값 위,
	// variant 아래 단이다.
	textSlotDefaults map[string]textPatch
	textVariants     []textVariant
	shapeDefault     shapePatch
	shapeVariants    []shapeVariant
	imageDefault     imagePatch
	imageVariants    []imageVariant
	table            tableComponents
	code             codeComponents
}

// tableComponents는 변환된 표 셀 외관 표다 (캐스케이드 내부 타입). shape/image의 default·
// variant와 나란한 자리이고, 담는 것이 cellPatch인 이유는 셀 캐스케이드가 그 패치를 병합하기
// 때문이다 (resolveCell → mergeCell). variant 목록이 없는 이유는 tableComponentsFile과 같다:
// 표의 변형은 프레임이 켜는 stripe 하나뿐이다.
type tableComponents struct {
	// cell은 모든 셀의 기본 외관 패치다 (테두리 등). header는 `<th>`에 얹는 델타(강조색·굵기).
	cell   cellPatch
	header cellPatch
	// stripe는 줄무늬 행 셀의 채움이다 — variant="striped"의 홀수 본문 행에만 얹힌다. 테마가
	// 줄무늬를 정하지 않으면 nil이다 (resolveTable이 nil을 건너뛴다).
	stripe *soml.Fill
}

// rawTheme는 테마 하나의 **원문 선언**이다 — 변환도 검사도 하기 전 상태다. 상속 병합이 이
// 층에서 일어나기 때문에 있는 타입이다: 파일 스키마의 모양(이름 붙은 배열
// 몇 개와 싱글톤 절 몇 개) 그대로 병합하면 규칙이 한 문장이고, 변환·완전성 검사는 병합
// **결과**에 한 번만 돈다 — 부모를 따로 완전하게 만들지 않아도 되고 자식의 부분 파일이 홀로
// requireTokens를 통과해야 하는 이상한 계약이 생기지 않는다.
type rawTheme struct {
	manifest   themeManifest
	tokens     tokensFile
	components componentsFile
	// layouts·slotLayouts는 레이아웃 이름 → CSS 원문이다 (파일 단위 병합의 단위). 파싱하지
	// 않고 문자열로 나르는 이유는 병합이 파일을 통째로 갈아 끼우기 때문이다.
	layouts     map[string]string
	slotLayouts map[string]string
	// assets는 `assets/` 아래 파일의 색인이다 (테마 루트 기준 경로 → 자리 · 주인). 파일 단위로 병합된다.
	assets map[string]rawAsset
	// origin은 이 선언을 읽은 자리다 (내장 · 로컬 · 설치) — 병합에는 쓰이지 않는다.
	origin themeOrigin
}

// loadTheme는 fsys의 dir/ 아래 테마 패키지를 **상속 없이** 읽는다. dir은 fs 경로다 (내장은
// "themes/minimal", 로컬은 프로젝트 상대 경로). 이름은 매니페스트가 소유한다.
//
// 상속을 여기서 풀지 않는 이유는 부모를 찾을 **이름 어휘**가 이 함수에 없다는 것이다 (어휘는
// 내장 ∪ 프로젝트 로컬이고 그것을 아는 자리는 loadThemeNamed다).
func loadTheme(fsys fs.FS, dir string) (themePackage, error) {
	raw, err := readRawTheme(fsys, dir)
	if err != nil {
		return themePackage{}, err
	}
	if raw.manifest.Extends != "" {
		return themePackage{}, fmt.Errorf(
			"theme.json의 extends(%q)는 이름 어휘가 있는 자리에서만 풀린다", raw.manifest.Extends)
	}
	return buildThemePackage(raw)
}

// readRawTheme는 dir/ 아래 파일을 **그대로** 읽는다 — 변환도 검사도 하지 않는다.
//
// 상속하는 테마의 파일은 **부분 파일**이어도 되고 아예 없어도 된다 (부모가 바닥을 깐다).
// 상속하지 않는 테마에서는 tokens.json·components.json·layouts/가 모두 있어야 한다 — 내장
// 테마가 받는 조건 그대로이고, 없는 파일을 빈 선언으로 읽으면 진단이 "fg 토큰이 없다"로
// 멀어진다.
func readRawTheme(fsys fs.FS, dir string) (rawTheme, error) {
	var raw rawTheme

	manifestBytes, err := fs.ReadFile(fsys, path.Join(dir, "theme.json"))
	if err != nil {
		return raw, fmt.Errorf("theme.json을 읽을 수 없다: %w", err)
	}
	if err := json.Unmarshal(manifestBytes, &raw.manifest); err != nil {
		return raw, fmt.Errorf("theme.json 파싱 실패: %w", err)
	}
	if raw.manifest.Name == "" {
		return raw, fmt.Errorf("theme.json에 name이 없다")
	}
	optional := raw.manifest.Extends != ""

	if raw.tokens, err = readTokensFile(fsys, dir, optional); err != nil {
		return raw, err
	}
	if raw.components, err = readComponentsFile(fsys, dir, optional); err != nil {
		return raw, err
	}
	if raw.layouts, raw.slotLayouts, err = readLayoutCSS(fsys, dir, optional); err != nil {
		return raw, err
	}
	if raw.assets, err = readThemeAssets(fsys, dir, raw.manifest.Name); err != nil {
		return raw, err
	}
	return raw, nil
}

// buildThemePackage는 원문 선언 한 벌을 **변환·검사**해 패키지로 만든다 (상속 병합이 끝난
// 뒤의 한 번).
func buildThemePackage(raw rawTheme) (themePackage, error) {
	var pkg themePackage
	pkg.manifest = raw.manifest

	// 크롬 선언은 토큰·컴포넌트보다 **먼저** 검사한다 — 선언만으로 판정할 수 있는 값 어휘이고,
	// 먼저 멈추면 "테마의 오타"와 "토큰 표의 문제"가 한 메시지에 섞이지 않는다.
	assets := &assetResolver{files: raw.assets}
	chrome, err := loadChrome(pkg.manifest, assets)
	if err != nil {
		return pkg, err
	}
	pkg.chrome = chrome

	tokens := tokensFromFile(raw.tokens, pkg.manifest.Name)
	// 완전성 검사는 변환 **직후**다: 폴백 테마가 없으므로 빠진 토큰은
	// 산출물에서 영 값으로 나타난다 — 근거는 theme_require.go의 패키지 주석에 있다. 컴포넌트
	// 표보다 먼저 보는 이유는 역할 검사가 이 표를 참조로 쓴다는 것이다.
	if err := requireTokens(tokens); err != nil {
		return pkg, err
	}
	// 색 토큰의 clrScheme 자리도 여기서 본다 — 필수 이름이 다 있는지 다음에 보는
	// 이유는 순서가 진단의 품질이라는 것이다: 슬롯 상한("남은 자리가 없다")보다 빠진 이름이
	// 먼저 읽혀야 테마 제작자가 표를 두 번 고치지 않는다.
	if err := requireColorSlots(tokens); err != nil {
		return pkg, err
	}
	pkg.tokens = tokens

	comps, err := componentsFromFile(raw.components, tokens)
	if err != nil {
		return pkg, err
	}
	if err := requireComponents(tokens, comps); err != nil {
		return pkg, err
	}
	if err := requireComponentColors(tokens, comps); err != nil {
		return pkg, err
	}
	pkg.components = comps

	if err := checkLayoutFiles(pkg.manifest, raw.layouts, raw.slotLayouts); err != nil {
		return pkg, err
	}
	pkg.layouts = raw.layouts
	pkg.slotLayouts = raw.slotLayouts

	// 배경은 토큰 표 **뒤**다 — 채움이 색 토큰을 가리킨다.
	backgrounds, err := loadBackgrounds(pkg.manifest, tokens, assets)
	if err != nil {
		return pkg, err
	}
	pkg.backgrounds = backgrounds
	// 선언이 가리킨 그림만 싣는다 (가리킨 순서) — `assets/` 안의 쓰지 않는 파일은 산출물에 나가지 않는다.
	pkg.tokens.Assets = assets.used
	pkg.assets = assets
	pkg.chain = []string{pkg.manifest.Name}
	return pkg, nil
}

// loadChrome은 매니페스트의 크롬 선언을 검사해 타입 붙은 목록으로 만든다.
//
// 값 어휘가 어긋나면 **로드를 멈춘다** (경고가 아니다): 테마의 오타를 조용히 무시하면 저자는
// "표지에 번호가 왜 찍히는가"를 자기 슬라이드에서 찾게 되고, 테마는 저자가 고칠 수 없는
// 자리다. 같은 이유로 skipLayouts의 이름은 **이 테마가 구현하는 레이아웃**이어야 한다 —
// 오타 하나가 제외 규칙을 조용히 무효로 만든다.
func loadChrome(manifest themeManifest, assets *assetResolver) ([]chromeDecl, error) {
	if len(manifest.Chrome) == 0 {
		return nil, nil
	}
	layouts := make(map[string]bool, len(manifest.Layouts))
	for _, name := range manifest.Layouts {
		layouts[name] = true
	}
	seen := map[soml.ChromeKind]bool{}
	decls := make([]chromeDecl, 0, len(manifest.Chrome))
	for i, raw := range manifest.Chrome {
		decl, err := chromeDeclOf(raw, layouts, assets)
		if err != nil {
			return nil, fmt.Errorf("theme.json의 chrome[%d]: %w", i, err)
		}
		if seen[decl.kind] {
			return nil, fmt.Errorf("theme.json의 chrome[%d]: kind %s가 두 번 선언되었다", i, decl.kind)
		}
		seen[decl.kind] = true
		decls = append(decls, decl)
	}
	return decls, nil
}

// chromeDeclOf는 선언 하나의 값 어휘를 검사한다. align·role은 **생략할 수 없다** — 테마가
// 자리와 타이포그래피를 소유하므로 기본값을 엔진에 두면 그 소유가 반으로
// 쪼개진다.
func chromeDeclOf(raw chromeDeclFile, layouts map[string]bool, assets *assetResolver) (chromeDecl, error) {
	var decl chromeDecl
	kinds := make([]string, 0, len(soml.ChromeKinds()))
	for _, kind := range soml.ChromeKinds() {
		kinds = append(kinds, string(kind))
		if string(kind) == raw.Kind {
			decl.kind = kind
		}
	}
	if decl.kind == "" {
		return decl, fmt.Errorf("알 수 없는 kind %q (쓸 수 있는 kind: %s)", raw.Kind, strings.Join(kinds, ", "))
	}

	aligns := make([]string, 0, len(soml.ChromeAligns()))
	for _, align := range soml.ChromeAligns() {
		aligns = append(aligns, string(align))
		if string(align) == raw.Align {
			decl.align = align
		}
	}
	if decl.align == "" {
		if raw.Align == "" {
			return decl, fmt.Errorf("align이 없다 (쓸 수 있는 align: %s)", strings.Join(aligns, ", "))
		}
		return decl, fmt.Errorf("알 수 없는 align %q (쓸 수 있는 align: %s)", raw.Align, strings.Join(aligns, ", "))
	}

	if decl.kind == soml.ChromeLogo {
		// 그림 크롬은 역할이 아니라 그림과 높이를 갖는다 — 글자 외관이 없으므로 role은 뜻이 없다.
		if raw.Role != "" {
			return decl, fmt.Errorf("logo에는 role을 쓸 수 없다 (%q) — 그림 크롬은 글자 외관이 없다. 크기는 height가 정한다", raw.Role)
		}
		if raw.Image == "" {
			return decl, fmt.Errorf("logo에 image가 없다 — 테마 패키지의 그림을 가리킨다 (예: %s/logo.png)", themeAssetDir)
		}
		if raw.Height <= 0 {
			return decl, fmt.Errorf("logo의 height가 없거나 0 이하다 (%g) — 논리 px 높이를 준다. 폭은 그림의 비율이 정한다", raw.Height)
		}
		key, err := assets.resolve("logo의 image", raw.Image)
		if err != nil {
			return decl, err
		}
		decl.image, decl.height, decl.alt = key, raw.Height, raw.Alt
		return decl, chromeSkips(&decl, raw, layouts)
	}
	if raw.Image != "" || raw.Height != 0 || raw.Alt != "" {
		return decl, fmt.Errorf("%s에는 image · height · alt를 쓸 수 없다 — 그림 크롬(logo)의 자리다", decl.kind)
	}

	roles := make([]string, 0, len(soml.TextRoles()))
	for _, role := range soml.TextRoles() {
		roles = append(roles, string(role))
		if string(role) == raw.Role {
			decl.role = role
		}
	}
	if decl.role == "" {
		if raw.Role == "" {
			return decl, fmt.Errorf("role이 없다 (쓸 수 있는 role: %s)", strings.Join(roles, ", "))
		}
		return decl, fmt.Errorf("알 수 없는 role %q (쓸 수 있는 role: %s)", raw.Role, strings.Join(roles, ", "))
	}
	return decl, chromeSkips(&decl, raw, layouts)
}

// chromeSkips는 제외 레이아웃 목록을 검사해 선언에 싣는다 — 이 테마의 레이아웃 이름만 받는다.
func chromeSkips(decl *chromeDecl, raw chromeDeclFile, layouts map[string]bool) error {
	if len(raw.SkipLayouts) == 0 {
		return nil
	}
	decl.skip = make(map[soml.SlideLayout]bool, len(raw.SkipLayouts))
	for _, name := range raw.SkipLayouts {
		if !layouts[name] {
			return fmt.Errorf("skipLayouts의 %q는 이 테마의 레이아웃이 아니다 (이 테마의 레이아웃: %s)",
				name, strings.Join(sortedKeys(layouts), ", "))
		}
		decl.skip[soml.SlideLayout(name)] = true
	}
	return nil
}

// sortedKeys는 진단 문구가 실행마다 같도록 이름 집합을 정렬해 돌려준다.
func sortedKeys(set map[string]bool) []string {
	names := make([]string, 0, len(set))
	for name := range set {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// readTokensFile은 tokens.json을 원문 선언으로 읽는다. optional이면 파일이 없어도 된다
// (상속하는 테마 — 부모의 표가 바닥이다).
func readTokensFile(fsys fs.FS, dir string, optional bool) (tokensFile, error) {
	var file tokensFile
	raw, err := fs.ReadFile(fsys, path.Join(dir, "tokens.json"))
	if err != nil {
		if optional && errors.Is(err, fs.ErrNotExist) {
			return file, nil
		}
		return file, fmt.Errorf("tokens.json을 읽을 수 없다: %w", err)
	}
	if err := json.Unmarshal(raw, &file); err != nil {
		return file, fmt.Errorf("tokens.json 파싱 실패: %w", err)
	}
	return file, nil
}

// tokensFromFile은 원문 토큰 선언을 soml.Theme로 눕힌다. 배열 순서를 그대로 옮겨 byte
// 동일성을 지킨다.
func tokensFromFile(file tokensFile, name string) soml.Theme {
	var theme soml.Theme
	theme.Name = name
	for _, c := range file.Color {
		theme.Colors = append(theme.Colors, soml.ColorToken{Name: c.Name, RGB: c.RGB, Slot: c.Slot, Ref: c.Ref})
	}
	for _, f := range file.Font {
		theme.Fonts = append(theme.Fonts, soml.FontToken{
			Name: soml.FontRole(f.Name), Latin: f.Latin, EastAsian: f.EA, Generic: f.Generic, Scheme: f.Scheme,
		})
	}
	for _, s := range file.Space {
		theme.Spaces = append(theme.Spaces, soml.SpaceToken{Name: s.Name, Px: s.Px})
	}
	for _, r := range file.Radius {
		theme.Radii = append(theme.Radii, soml.RadiusToken{Name: r.Name, Px: r.Px})
	}
	for _, s := range file.Shadow {
		// color는 토큰 참조다 (표가 리터럴을 쓰면 PPTX에서 schemeClr 참조가 끊긴다 —
		// 제품 핵심 계약). soml.TokenColor로 감싼다.
		theme.Shadows = append(theme.Shadows, soml.ShadowToken{
			Name: s.Name, Dist: s.Dist, Dir: s.Dir, Blur: s.Blur,
			Color: soml.TokenColor(s.Color), Alpha: s.Alpha,
		})
	}
	return theme
}

// readComponentsFile은 components.json을 원문 선언으로 읽는다. 미지원 키는
// DisallowUnknownFields가 진단한다 — 조용히 무시하면 저자가 오버라이드가 먹었다고 착각한다
// (닫힌 어휘). optional이면 파일이 없어도 된다 (readTokensFile과 같은 계약).
func readComponentsFile(fsys fs.FS, dir string, optional bool) (componentsFile, error) {
	var file componentsFile
	raw, err := fs.ReadFile(fsys, path.Join(dir, "components.json"))
	if err != nil {
		if optional && errors.Is(err, fs.ErrNotExist) {
			return file, nil
		}
		return file, fmt.Errorf("components.json을 읽을 수 없다: %w", err)
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&file); err != nil {
		return file, fmt.Errorf("components.json 파싱 실패: %w", err)
	}
	return file, nil
}

// componentsFromFile은 원문 컴포넌트 선언을 캐스케이드 내부 표로 눕힌다. theme는 radius·shadow
// 토큰을 해석하려고 받는다 (tokensFromFile이 먼저 채운 표).
func componentsFromFile(file componentsFile, theme soml.Theme) (components, error) {
	var comps components
	comps.textRoles = map[soml.TextRole]soml.TextStyle{}
	for _, r := range file.Text.Roles {
		comps.textRoles[soml.TextRole(r.Name)] = soml.TextStyle{
			Role: soml.TextRole(r.Name), Font: soml.FontRole(r.Font), Size: r.Size,
			Weight: r.Weight, LineHeight: r.LineHeight,
			Color: soml.TokenColor(r.Color), Align: soml.TextAlign(r.Align),
		}
	}

	comps.textComponentDefaults = map[string]textPatch{}
	comps.textSlotDefaults = map[string]textPatch{}
	for _, d := range file.Text.ComponentDefaults {
		var patch textPatch
		if d.Size != nil {
			patch.size = d.Size
		}
		if d.Weight != nil {
			patch.weight = d.Weight
		}
		if d.Color != nil {
			c, err := themeColor(theme, fmt.Sprintf("text.componentDefaults %q의 color", d.key()), *d.Color)
			if err != nil {
				return comps, err
			}
			patch.color = &c
		}
		switch {
		case d.Slot == "" && d.Layout == "":
			if d.Tag == "" {
				return comps, fmt.Errorf("text.componentDefaults의 한 줄에 주인이 없다 — tag나 layout + slot을 쓴다")
			}
			comps.textComponentDefaults[d.Tag] = patch
		case d.Tag != "":
			return comps, fmt.Errorf("text.componentDefaults의 tag %q와 slot을 함께 쓸 수 없다 — 한 줄의 주인은 하나다", d.Tag)
		default:
			layout := soml.SlideLayout(d.Layout)
			if _, ok := soml.LayoutSlot(layout, d.Slot); !ok {
				return comps, fmt.Errorf("text.componentDefaults의 슬롯 %q/%q가 레이아웃 어휘에 없다 (레이아웃 %q의 슬롯: %s)",
					d.Layout, d.Slot, d.Layout, strings.Join(soml.SlotNames(layout), ", "))
			}
			comps.textSlotDefaults[d.key()] = patch
		}
	}

	for _, v := range file.Text.Variants {
		var patch textPatch
		if v.Size != nil {
			patch.size = v.Size
		}
		if v.Weight != nil {
			patch.weight = v.Weight
		}
		if v.Color != nil {
			c, err := themeColor(theme, fmt.Sprintf("text variant %q의 color", v.Name), *v.Color)
			if err != nil {
				return comps, err
			}
			patch.color = &c
		}
		comps.textVariants = append(comps.textVariants, textVariant{name: v.Name, patch: patch})
	}

	shapeDefault, err := shapePatchFromFile(theme, file.Shape.Default.Fill,
		file.Shape.Default.Border, file.Shape.Default.Radius, "")
	if err != nil {
		return comps, fmt.Errorf("shape.default: %w", err)
	}
	comps.shapeDefault = shapeDefault
	for _, v := range file.Shape.Variants {
		patch, err := shapeVariantPatchFromFile(theme, v)
		if err != nil {
			return comps, fmt.Errorf("shape variant %q: %w", v.Name, err)
		}
		comps.shapeVariants = append(comps.shapeVariants, shapeVariant{name: v.Name, patch: patch})
	}

	for _, v := range file.Image.Variants {
		patch, err := imageVariantPatchFromFile(theme, v)
		if err != nil {
			return comps, fmt.Errorf("image variant %q: %w", v.Name, err)
		}
		comps.imageVariants = append(comps.imageVariants, imageVariant{name: v.Name, patch: patch})
	}

	table, err := tableComponentsFromFile(file.Table)
	if err != nil {
		return comps, fmt.Errorf("table: %w", err)
	}
	comps.table = table

	code, err := codeComponentsFromFile(file.Code, theme)
	if err != nil {
		return comps, fmt.Errorf("code: %w", err)
	}
	comps.code = code

	return comps, nil
}

// tableComponentsFromFile은 표 셀 외관 원본을 캐스케이드 내부 표로 눕힌다. cell·header는
// cellPatch로(셀 캐스케이드가 병합할 형태), stripe는 *soml.Fill로 변환한다. 색·테두리 문법은
// shape default과 같은 헬퍼(fillFromString·borderFromString)를 지나 자리마다 다르게 읽히지
// 않는다.
func tableComponentsFromFile(file tableComponentsFile) (tableComponents, error) {
	var table tableComponents
	cell, err := cellPatchFromFile(file.Cell)
	if err != nil {
		return table, fmt.Errorf("cell: %w", err)
	}
	table.cell = cell
	header, err := cellPatchFromFile(file.Header)
	if err != nil {
		return table, fmt.Errorf("header: %w", err)
	}
	table.header = header
	if file.Stripe != "" {
		fill, err := fillFromString(file.Stripe)
		if err != nil {
			return table, fmt.Errorf("stripe: %w", err)
		}
		table.stripe = fill
	}
	return table, nil
}

// cellPatchFromFile은 셀 외관 원본 한 벌을 cellPatch로 눕힌다 — shapePatchFromFile의 셀 판이고
// 규약이 같다: 빈 문자열/nil은 "정하지 않음". color는 토큰 참조로 남긴다 (RGB로 눕히면
// schemeClr 참조가 끊긴다 — soml.Color의 계약). align·anchor는 원본이 열지 않으므로 여기 없다.
func cellPatchFromFile(file cellStyleFile) (cellPatch, error) {
	var patch cellPatch
	if file.Fill != "" {
		fill, err := fillFromString(file.Fill)
		if err != nil {
			return patch, err
		}
		patch.fill = fill
	}
	if file.Border != "" {
		border, err := borderFromString(file.Border)
		if err != nil {
			return patch, err
		}
		patch.border = border
	}
	if file.Color != "" {
		c := soml.TokenColor(file.Color)
		patch.color = &c
	}
	if file.Weight != nil {
		patch.weight = file.Weight
	}
	return patch, nil
}

// ── 값 헬퍼 (토큰 참조를 한 줄로 적는 도구) ─────────────────────────────────
//
// 토큰 이름을 받는 이유는 테마의 색·반경·그림자가 전부 토큰 참조여야 하기 때문이다 — 테마가
// 리터럴을 쓰면 PPTX에서 `schemeClr` 참조가 끊겨 그 도형만 테마 색 편집을 따라가지 못한다
// (제품 핵심 계약). 하드코딩 표(옛 theme.go)가 쓰던 도구였고, 그 표가 데이터로 나간 뒤에는
// **파일의 문자열을 패치로 눕히는** 아래 변환기들이 쓴다.
func colorFill(token string) *soml.Fill { return ptr(soml.ColorFill(soml.TokenColor(token))) }

func solidBorder(width float64, token string) *soml.Border {
	return &soml.Border{Width: width, Style: soml.BorderSolid, Color: soml.TokenColor(token)}
}

// radiusToken·shadowToken은 표에 없는 이름이면 nil을 준다 — 그 단이 그 필드를 정하지 않은
// 것이 된다. 필수 토큰(sm·md·lg · soft)은 로더가 요구하므로(theme_require.go) 내장 테마에서는
// nil이 나오지 않지만, 테마가 **자기 이름**을 쓴 variant는 그 이름이 자기 tokens.json에 있어야
// 한다 — 없으면 빌드를 멈추는 대신 그 필드가 빈다 (모르는 variant를 경고로 두는 것과 같은 태도).
func radiusToken(theme soml.Theme, name string) *soml.Radius {
	token, ok := theme.Radius(name)
	if !ok {
		return nil
	}
	return ptr(soml.TokenRadius(name, token.Px))
}

func shadowToken(theme soml.Theme, name string) *soml.ShadowToken {
	token, ok := theme.Shadow(name)
	if !ok {
		return nil
	}
	return &token
}

// shapePatchFromFile은 shape default의 문자열 필드를 shapePatch로 눕힌다. 빈 문자열은 "정하지
// 않음"이다. fill/border 색·radius/shadow 토큰 해석은 위 헬퍼(colorFill·solidBorder·
// radiusToken·shadowToken)를 지난다.
func shapePatchFromFile(theme soml.Theme, fill, border, radius, shadow string) (shapePatch, error) {
	var patch shapePatch
	if fill != "" {
		f, err := themeFill(theme, fill)
		if err != nil {
			return patch, err
		}
		patch.fill = f
	}
	if border != "" {
		b, err := themeBorder(theme, border)
		if err != nil {
			return patch, err
		}
		patch.border = b
	}
	if radius != "" {
		patch.radius = radiusToken(theme, radius)
	}
	if shadow != "" {
		patch.shadow = shadowToken(theme, shadow)
	}
	return patch, nil
}

func shapeVariantPatchFromFile(theme soml.Theme, v shapeVariantFile) (shapePatch, error) {
	fill, border, radius, shadow := "", "", "", ""
	if v.Fill != nil {
		fill = *v.Fill
	}
	if v.Border != nil {
		border = *v.Border
	}
	if v.Radius != nil {
		radius = *v.Radius
	}
	if v.Shadow != nil {
		shadow = *v.Shadow
	}
	return shapePatchFromFile(theme, fill, border, radius, shadow)
}

func imageVariantPatchFromFile(theme soml.Theme, v imageVariantFile) (imagePatch, error) {
	var patch imagePatch
	if v.Border != nil {
		b, err := themeBorder(theme, *v.Border)
		if err != nil {
			return patch, err
		}
		patch.border = b
	}
	if v.Radius != nil {
		patch.radius = radiusToken(theme, *v.Radius)
	}
	if v.Shadow != nil {
		patch.shadow = shadowToken(theme, *v.Shadow)
	}
	return patch, nil
}

// themeFill · themeBorder · themeColor는 테마 파일의 외관 문자열을 **저자의 값 문법 그대로** 읽는다 —
// 인라인 `fill`·`border`·`color`를 해석하는 같은 함수(resolver)를 지난다.
//
// 같은 함수인 이유: 테마가 `gradient(accent1 accent6)`·`shade(accent1, 20%)`·`4 gradient(accent1 muted)`를
// 쓰려면 그 문법이 테마에서도 같은 뜻이어야 한다. 테마용 축약 파서를 따로 두면 그 파서가 아는 것만
// 테마에 열리고(그라디언트가 그렇게 막혀 있었다), 같은 값이 자리에 따라 다르게 읽힌다. 진단은 파일
// 위치가 없으므로 메시지만 에러로 올린다 — 위치는 호출자가 감싸는 자리 이름이 말한다. 토큰 참조만
// 쓰는지(리터럴 금지)는 이 문법의 몫이 아니라 requireComponentColors의 몫이다.
func themeFill(theme soml.Theme, raw string) (*soml.Fill, error) {
	r := &resolver{theme: theme}
	f, ok := r.fill(soml.Attr{Name: "fill", Value: raw, HasValue: true})
	if err := themeValueErr(r, ok); err != nil {
		return nil, err
	}
	return &f, nil
}

func themeBorder(theme soml.Theme, raw string) (*soml.Border, error) {
	r := &resolver{theme: theme}
	b, ok := r.border(soml.Attr{Name: "border", Value: raw, HasValue: true})
	if err := themeValueErr(r, ok); err != nil {
		return nil, err
	}
	return &b, nil
}

func themeColor(theme soml.Theme, where, raw string) (soml.Color, error) {
	r := &resolver{theme: theme}
	c, ok := r.colorValue(where, raw, soml.Span{})
	if err := themeValueErr(r, ok); err != nil {
		return soml.Color{}, err
	}
	return c, nil
}

// themeValueErr는 테마 값 하나를 읽은 resolver의 진단을 에러 하나로 모은다.
func themeValueErr(r *resolver, ok bool) error {
	if len(r.diags) == 0 && ok {
		return nil
	}
	msgs := make([]string, 0, len(r.diags))
	for _, d := range r.diags {
		msgs = append(msgs, d.Message)
	}
	if len(msgs) == 0 {
		msgs = append(msgs, "값을 해석할 수 없다")
	}
	return errors.New(strings.Join(msgs, "; "))
}

// fillFromString은 `none` 또는 색 토큰명을 *soml.Fill로 눕힌다 (colorFill/soml.NoFill과 같은
// 결과). 테마 데이터는 색 **토큰만** 쓴다 (리터럴 금지 — colorFill의 근거).
func fillFromString(s string) (*soml.Fill, error) {
	if s == noneValue {
		return ptr(soml.NoFill()), nil
	}
	return colorFill(s), nil
}

// borderFromString은 `none` 또는 `<굵기> <스타일> <색>` 축약을 *soml.Border로 눕힌다 —
// resolver.border의 값 문법과 같고, `1 solid muted`는 solidBorder(1, "muted")와 같다.
func borderFromString(s string) (*soml.Border, error) {
	if s == noneValue {
		return ptr(soml.NoBorder()), nil
	}
	fields := strings.Fields(s)
	if len(fields) != borderParts {
		return nil, fmt.Errorf("border는 %q 또는 `<굵기> <스타일> <색>` 3조각이다 (%q)", noneValue, s)
	}
	width, err := strconv.ParseFloat(fields[0], 64)
	if err != nil {
		return nil, fmt.Errorf("border 굵기가 수치가 아니다 (%q)", fields[0])
	}
	style := soml.BorderStyle(fields[1])
	ok := false
	for _, s := range soml.BorderStyles() {
		if s == style {
			ok = true
		}
	}
	if !ok {
		return nil, fmt.Errorf("알 수 없는 border 스타일 %q", fields[1])
	}
	return &soml.Border{Width: width, Style: style, Color: soml.TokenColor(fields[2])}, nil
}

// readLayoutCSS는 dir/layouts/ 아래 CSS를 두 맵으로 읽는다 — 흐름 배치(`<이름>.css`)와 슬롯
// 영역(`<이름>.slots.css`). 키는 접미를 뗀 레이아웃 이름이다.
//
// 두 맵을 한 번에 읽는 이유는 디렉토리를 한 번만 훑기 때문이고, 파일 내용을 문자열로 나르는
// 이유는 병합의 단위가 **파일**이라는 것이다. 선언과의 대조는 병합 뒤에
// 도는 checkLayoutFiles의 몫이다 — 자식의 `layouts/`만으로는 대조가 성립하지 않는다.
// optional이면 `layouts/`가 없어도 된다 (상속하는 테마 — 부모의 파일이 바닥이다).
func readLayoutCSS(fsys fs.FS, dir string, optional bool) (layouts, slots map[string]string, err error) {
	entries, err := fs.ReadDir(fsys, path.Join(dir, "layouts"))
	if err != nil {
		if optional && errors.Is(err, fs.ErrNotExist) {
			return map[string]string{}, map[string]string{}, nil
		}
		return nil, nil, fmt.Errorf("layouts/를 읽을 수 없다: %w", err)
	}
	layouts, slots = map[string]string{}, map[string]string{}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".css") {
			continue
		}
		raw, err := fs.ReadFile(fsys, path.Join(dir, "layouts", e.Name()))
		if err != nil {
			return nil, nil, fmt.Errorf("layouts/%s를 읽을 수 없다: %w", e.Name(), err)
		}
		// 슬롯 영역 파일을 먼저 가른다 — 흐름 맵에 실으면 `title.slots`라는 존재하지 않는
		// 레이아웃으로 오인해 "선언에 없다" 에러가 난다.
		if name, ok := strings.CutSuffix(e.Name(), ".slots.css"); ok {
			slots[name] = string(raw)
			continue
		}
		layouts[strings.TrimSuffix(e.Name(), ".css")] = string(raw)
	}
	return layouts, slots, nil
}

// checkLayoutFiles는 CSS 파일 집합이 manifest.Layouts 선언과 맞는지 대조한다. 선언과 파일이
// 어긋나면 저자가 오버라이드/구현이 먹었다고 착각하므로 조용히 넘기지 않고 에러다 (닫힌 어휘,
// components.json과 같은 태도). CSS 내용은 여기서 보지 않는다 — 배치·외관 경계 검사는
// theme_lint.go의 몫이고, 빈(주석만) CSS는 허용한다.
//
// 흐름 배치는 **양방향** 대조다 (선언에 있으면 파일이 있어야 하고, 파일이 있으면 선언에
// 있어야 한다). 슬롯 영역은 **단방향**이다 — 슬롯 영역 파일은 선택이므로(슬롯이 없는 canvas는
// 파일이 없어도 된다) "선언됐는데 파일이 없다"는 에러가 아니다. 반대로 선언되지 않은
// 레이아웃의 슬롯 파일(`corp-hero.slots.css`인데 layouts에 corp-hero가 없음)은 오타이므로
// 에러다.
func checkLayoutFiles(manifest themeManifest, layouts, slots map[string]string) error {
	// 선언 → 파일: manifest.Layouts는 순서 있는 슬라이스라 순회가 이미 결정적이다.
	declared := map[string]bool{}
	for _, name := range manifest.Layouts {
		declared[name] = true
		if _, ok := layouts[name]; !ok {
			return fmt.Errorf("theme.json의 layouts에 %q가 있지만 layouts/%s.css가 없다", name, name)
		}
	}
	// 파일 → 선언: 맵은 순서가 없으므로 정렬해 진단이 결정적이게 한다.
	var undeclared []string
	for name := range layouts {
		if !declared[name] {
			undeclared = append(undeclared, name)
		}
	}
	sort.Strings(undeclared)
	if len(undeclared) > 0 {
		return fmt.Errorf("layouts/%s.css가 있지만 theme.json의 layouts 선언에 없다", undeclared[0])
	}
	undeclared = nil
	for name := range slots {
		if !declared[name] {
			undeclared = append(undeclared, name)
		}
	}
	sort.Strings(undeclared)
	if len(undeclared) > 0 {
		return fmt.Errorf("layouts/%s.slots.css가 있지만 theme.json의 layouts 선언에 %q가 없다",
			undeclared[0], undeclared[0])
	}
	return nil
}

// loadBuiltinTheme는 내장 테마를 이름으로 읽는다. 내장 파일은 빌드 시점에 고정되므로 로드
// 실패는 프로그래머 오류다 (regexp.MustCompile과 같은 자리) — 호출자가 에러를 다룰 자리가 없고,
// TestBuiltinThemesAreLoadable이 이 불변을 전 테마에서 못 박는다. **이름이 어휘 안인지는
// 호출자가 먼저 본다** (IsBuiltinTheme): 저자가 쓴 이름은 진단이 되어야 하고 panic이 되면 안 된다.
func loadBuiltinTheme(name string) themePackage {
	pkg, err := loadTheme(builtinThemes, path.Join(themeDir, name))
	if err != nil {
		panic(fmt.Sprintf("내장 테마 %q 로드 실패: %v", name, err))
	}
	return pkg
}
