package core

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"path"
	"sort"
	"strconv"
	"strings"

	"github.com/k2sebeom/balpyo/soml"
)

// 이 파일은 **테마 패키지 포맷**(PRD §8.2)의 로더다 — Iteration 10이 core/theme.go의 하드코딩
// 표를 데이터로 해체한 자리다. 순서를 이렇게 잡은 이유는 theme.go의 패키지 주석에 있다:
// 하드코딩 표가 두 렌더러의 실제 요구를 드러낸 다음에야 스키마를 파일로 고정한다.
//
// 포맷은 **데이터 전용**이다 (JSON + CSS + 에셋, JS 없음 — PRD §8.2·§13-11): `theme get`으로
// 받은 원격 테마가 임의 코드 실행 경로가 되어서는 안 된다. 로더는 fs.FS 위에서 도므로 내장
// 테마(go:embed)와 프로젝트 로컬 테마(os.DirFS)가 같은 코드를 지난다.
//
// **byte 동일성 계약**: 내장 minimal을 이 로더로 읽은 soml.Theme는 옛 하드코딩
// minimalTheme()과 값·순서가 정확히 같아야 한다. 그래야 기존 골든(themeCSS·clrScheme·픽스처)이
// 움직이지 않고, 해체가 동작을 바꾸지 않았음이 증명된다. 순서를 지키려고 tokens.json은 객체가
// 아니라 **순서 있는 배열**을 쓴다 (JSON 객체 키 순서에 기대지 않는다 — PRD §8.4의 객체
// 스케치와 갈라지는 자리이고 ADR이 근거를 남긴다).

// builtinThemes는 바이너리에 내장된 공식 테마다 (PRD §8.6). v1은 minimal 하나이며 기본값이자
// 폴백 기준이다 — `balpyo init` 직후 네트워크·설치 없이 예쁜 결과가 나오는 것이 첫인상을
// 지배하므로 기본 경로에서 네트워크를 제거한다.
//
//go:embed themes
var builtinThemes embed.FS

// BuiltinThemes는 내장 테마 이름 목록이다 (정렬, 지금은 `minimal` 하나).
//
// 공개하는 이유는 **어휘의 단일 출처**다 (ADR 0048): `balpyo init --theme`이 받을 수 있는
// 이름이 곧 이 목록이어야 하고, 목록을 CLI에 적어 두면 내장 테마를 늘릴 때(P4.2) 두 곳이
// 갈라져 사용자가 값 층이 읽지 않는 이름을 `balpyo.json`에 갖게 된다.
func BuiltinThemes() []string {
	entries, err := fs.ReadDir(builtinThemes, "themes")
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

// themeManifest는 theme.json의 파싱 결과다 (PRD §8.2).
type themeManifest struct {
	Name string `json:"name"`
	// Extends는 상속할 테마 이름이다 (PRD §8.5) — v1 minimal에서는 비어 있다. 상속 병합은
	// 아직 구현하지 않으므로 비지 않은 값은 로더가 진단한다 (미구현을 조용히 무시하지 않는다).
	Extends string `json:"extends"`
	// Layouts는 이 테마가 구현하는 표준 레이아웃 이름 목록이다. 표준 카탈로그 중 여기 없는
	// 이름은 minimal 구현으로 폴백하고 경고한다 (core.slideLayout, 10/03). 테마 자체
	// 레이아웃(`corp-hero`)도 여기 실려 어휘가 된다.
	Layouts []string `json:"layouts"`
	// Chrome은 덱 크롬(페이지 번호·푸터) 선언이다 (PRD §8.1·§8.4 · ADR 0045).
	//
	// 테마가 소유하는 것은 **자리·정렬·역할·제외 규칙**이고, 내용과 켜짐은 `balpyo.json`의
	// footer·pageNumber다. 테마가 어떤 kind를 선언하지 않으면 그 크롬은 값 층에서 경고와
	// 함께 미탑재된다 (ADR 0044의 슬롯 규칙 그대로) — "테마가 받아야 뜻이 있다".
	Chrome []chromeDeclFile `json:"chrome"`
}

// chromeDeclFile은 theme.json의 크롬 선언 하나다 (원문 문자열). 값 어휘 검사는 loadChrome이
// 하고, 그 뒤로는 타입이 붙은 chromeDecl만 돈다 — 렌더러도 값 층도 원문 문자열을 다시 읽지
// 않는다 (컴포넌트 표가 값 문법을 미리 눕혀 담는 것과 같은 태도).
type chromeDeclFile struct {
	Kind        string   `json:"kind"`
	Align       string   `json:"align"`
	Role        string   `json:"role"`
	SkipLayouts []string `json:"skipLayouts"`
}

// chromeDecl은 검사가 끝난 크롬 선언 하나다.
type chromeDecl struct {
	kind  soml.ChromeKind
	align soml.TextAlign
	role  soml.TextRole
	// skip은 이 크롬을 싣지 않을 레이아웃 집합이다 (표지·섹션 구분 등). 값 층이 슬라이드마다
	// 이 집합을 보고 확정 모델에서 뺀다 — 렌더러는 규칙을 모른다.
	skip map[soml.SlideLayout]bool
}

// themePackage는 로드된 테마 하나다 — 토큰 표(soml.Theme)와 매니페스트를 함께 담는다.
// 컴포넌트 외관(components.json)과 레이아웃 CSS(layouts/*.css)는 뒤 하위 단계가 이 구조체에
// 필드를 더한다.
type themePackage struct {
	manifest themeManifest
	tokens   soml.Theme
	// components는 로드·변환된 컴포넌트 외관 표다 (components.json). tokens와 달리 값 문법을
	// 캐스케이드 내부 타입으로 이미 눕혀 담는다 — resolveText/Shape/Image가 재변환 없이 읽는다.
	components components
	// layouts는 레이아웃 이름→배치 CSS 원문 맵이다 (layouts/*.css). 값은 아직 컴파일하지 않은
	// 원문이다 — 슬롯 배치 컴파일은 10/03, 실체 CSS는 10/06이 채운다. 로더는 이 맵의 키 집합이
	// theme.json의 layouts 선언과 정확히 일치함을 대조한다.
	layouts map[string]string
	// slotLayouts는 레이아웃 이름→**슬롯 영역** CSS 원문 맵이다 (layouts/*.slots.css, ADR 0035).
	// layouts와 나란히 두지만 파일이 다르다 (`title.slots.css`): 흐름 배치(layouts)와 슬롯 영역은
	// 겨누는 선택자가 달라 한 선언 블록에 섞을 수 없다. 슬롯 영역 파일은 **선택**이라(슬롯이
	// 없는 canvas는 파일이 없다) 로더는 키가 선언된 레이아웃의 부분집합임만 확인한다.
	slotLayouts map[string]string
	// chrome은 검사가 끝난 크롬 선언들이다 (theme.json의 chrome — 선언 순서 그대로).
	// 선언 순서가 곧 슬라이드에 실리는 순서이므로(웹 DOM · PPTX 도형 순서) 맵이 아니라
	// 목록이다: 맵으로 두면 두 타깃의 산출물 순서가 실행마다 달라진다.
	chrome []chromeDecl
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

// codeComponentsFile은 코드 블록의 테마 절이다 (ADR 0051). 표가 **종류 → 색 토큰**이고,
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

type textComponentDefaultFile struct {
	Tag   string   `json:"tag"`
	Size  *float64 `json:"size"`
	Color *string  `json:"color"`
}

type textVariantFile struct {
	Name   string   `json:"name"`
	Size   *float64 `json:"size"`
	Weight *int     `json:"weight"`
	Color  *string  `json:"color"`
}

type shapeComponentsFile struct {
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

// tableComponentsFile은 표 셀 외관 원본 스키마다 (ADR 0036). shape/image와 달리 `variants`
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
// 캐스케이드가 이미 쓰는 타입 그대로 담는다 (theme.go의 옛 함수 반환형과 같다) — 그래야
// resolveText/Shape/Image가 재변환 없이 읽고, byte 동일성 테스트가 옛 함수와 직접 비교한다.

type components struct {
	textRoles             map[soml.TextRole]soml.TextStyle
	textComponentDefaults map[string]textPatch
	textVariants          []textVariant
	shapeDefault          shapePatch
	shapeVariants         []shapeVariant
	imageDefault          imagePatch
	imageVariants         []imageVariant
	table                 tableComponents
	code                  codeComponents
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

// loadTheme는 fsys의 dir/ 아래 테마 패키지를 읽는다. dir은 fs 경로다 (내장은
// "themes/minimal", 로컬은 프로젝트 상대 경로). 이름은 매니페스트가 소유한다.
func loadTheme(fsys fs.FS, dir string) (themePackage, error) {
	var pkg themePackage

	manifestBytes, err := fs.ReadFile(fsys, path.Join(dir, "theme.json"))
	if err != nil {
		return pkg, fmt.Errorf("theme.json을 읽을 수 없다: %w", err)
	}
	if err := json.Unmarshal(manifestBytes, &pkg.manifest); err != nil {
		return pkg, fmt.Errorf("theme.json 파싱 실패: %w", err)
	}
	if pkg.manifest.Name == "" {
		return pkg, fmt.Errorf("theme.json에 name이 없다")
	}
	if pkg.manifest.Extends != "" {
		// 상속 병합은 아직 구현하지 않았다 — 조용히 무시하면 저자가 오버라이드가 먹었다고
		// 착각한다 (닫힌 어휘가 막으려던 조용한 실패의 변종).
		return pkg, fmt.Errorf("theme.json의 extends는 아직 지원하지 않는다 (%q)", pkg.manifest.Extends)
	}

	// 크롬 선언은 토큰·컴포넌트보다 **먼저** 검사한다 — 선언만으로 판정할 수 있는 값 어휘이고,
	// 먼저 멈추면 "테마의 오타"와 "토큰 표의 문제"가 한 메시지에 섞이지 않는다.
	chrome, err := loadChrome(pkg.manifest)
	if err != nil {
		return pkg, err
	}
	pkg.chrome = chrome

	tokens, err := loadTokens(fsys, dir, pkg.manifest.Name)
	if err != nil {
		return pkg, err
	}
	// 완전성 검사는 파싱 **직후**다 (ADR 0052의 결정 3): 폴백 테마가 없으므로 빠진 토큰은
	// 산출물에서 영 값으로 나타난다 — 근거는 theme_require.go의 패키지 주석에 있다. 컴포넌트
	// 표보다 먼저 보는 이유는 역할 검사가 이 표를 참조로 쓴다는 것이다.
	if err := requireTokens(tokens); err != nil {
		return pkg, err
	}
	pkg.tokens = tokens

	comps, err := loadComponents(fsys, dir, tokens)
	if err != nil {
		return pkg, err
	}
	if err := requireComponents(tokens, comps); err != nil {
		return pkg, err
	}
	pkg.components = comps

	layouts, err := loadLayouts(fsys, dir, pkg.manifest)
	if err != nil {
		return pkg, err
	}
	pkg.layouts = layouts

	slotLayouts, err := loadSlotLayouts(fsys, dir, pkg.manifest)
	if err != nil {
		return pkg, err
	}
	pkg.slotLayouts = slotLayouts
	return pkg, nil
}

// loadChrome은 매니페스트의 크롬 선언을 검사해 타입 붙은 목록으로 만든다 (ADR 0045).
//
// 값 어휘가 어긋나면 **로드를 멈춘다** (경고가 아니다): 테마의 오타를 조용히 무시하면 저자는
// "표지에 번호가 왜 찍히는가"를 자기 슬라이드에서 찾게 되고, 테마는 저자가 고칠 수 없는
// 자리다. 같은 이유로 skipLayouts의 이름은 **이 테마가 구현하는 레이아웃**이어야 한다 —
// 오타 하나가 제외 규칙을 조용히 무효로 만든다.
func loadChrome(manifest themeManifest) ([]chromeDecl, error) {
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
		decl, err := chromeDeclOf(raw, layouts)
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
// 자리와 타이포그래피를 소유하기로 했으므로(ADR 0045) 기본값을 엔진에 두면 그 소유가 반으로
// 쪼개진다.
func chromeDeclOf(raw chromeDeclFile, layouts map[string]bool) (chromeDecl, error) {
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

	if len(raw.SkipLayouts) > 0 {
		decl.skip = make(map[soml.SlideLayout]bool, len(raw.SkipLayouts))
		for _, name := range raw.SkipLayouts {
			if !layouts[name] {
				return decl, fmt.Errorf("skipLayouts의 %q는 이 테마의 레이아웃이 아니다 (이 테마의 레이아웃: %s)",
					name, strings.Join(sortedKeys(layouts), ", "))
			}
			decl.skip[soml.SlideLayout(name)] = true
		}
	}
	return decl, nil
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

// loadTokens는 tokens.json을 soml.Theme로 읽는다. 배열 순서를 그대로 옮겨 byte 동일성을
// 지킨다.
func loadTokens(fsys fs.FS, dir, name string) (soml.Theme, error) {
	var theme soml.Theme
	raw, err := fs.ReadFile(fsys, path.Join(dir, "tokens.json"))
	if err != nil {
		return theme, fmt.Errorf("tokens.json을 읽을 수 없다: %w", err)
	}
	var file tokensFile
	if err := json.Unmarshal(raw, &file); err != nil {
		return theme, fmt.Errorf("tokens.json 파싱 실패: %w", err)
	}

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
		// theme.go의 옛 minimalShadows 근거). soml.TokenColor로 감싼다.
		theme.Shadows = append(theme.Shadows, soml.ShadowToken{
			Name: s.Name, Dist: s.Dist, Dir: s.Dir, Blur: s.Blur,
			Color: soml.TokenColor(s.Color), Alpha: s.Alpha,
		})
	}
	return theme, nil
}

// loadComponents는 components.json을 캐스케이드 내부 표로 읽는다. theme는 radius·shadow 토큰을
// 해석하려고 받는다 (loadTokens가 먼저 채운 표). 미지원 키는 DisallowUnknownFields가 진단한다
// — 조용히 무시하면 저자가 오버라이드가 먹었다고 착각한다 (닫힌 어휘, manifest.Extends와 같은 태도).
func loadComponents(fsys fs.FS, dir string, theme soml.Theme) (components, error) {
	var comps components
	raw, err := fs.ReadFile(fsys, path.Join(dir, "components.json"))
	if err != nil {
		return comps, fmt.Errorf("components.json을 읽을 수 없다: %w", err)
	}
	var file componentsFile
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&file); err != nil {
		return comps, fmt.Errorf("components.json 파싱 실패: %w", err)
	}

	comps.textRoles = map[soml.TextRole]soml.TextStyle{}
	for _, r := range file.Text.Roles {
		comps.textRoles[soml.TextRole(r.Name)] = soml.TextStyle{
			Role: soml.TextRole(r.Name), Font: soml.FontRole(r.Font), Size: r.Size,
			Weight: r.Weight, LineHeight: r.LineHeight,
			Color: soml.TokenColor(r.Color), Align: soml.TextAlign(r.Align),
		}
	}

	comps.textComponentDefaults = map[string]textPatch{}
	for _, d := range file.Text.ComponentDefaults {
		var patch textPatch
		if d.Size != nil {
			patch.size = d.Size
		}
		if d.Color != nil {
			c := soml.TokenColor(*d.Color)
			patch.color = &c
		}
		comps.textComponentDefaults[d.Tag] = patch
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
			c := soml.TokenColor(*v.Color)
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

// shapePatchFromFile은 shape default의 문자열 필드를 shapePatch로 눕힌다. 빈 문자열은 "정하지
// 않음"이다. fill/border 색·radius/shadow 토큰 해석은 theme.go의 헬퍼(colorFill·solidBorder·
// radiusToken·shadowToken)를 그대로 지나 byte 동일성을 지킨다.
func shapePatchFromFile(theme soml.Theme, fill, border, radius, shadow string) (shapePatch, error) {
	var patch shapePatch
	if fill != "" {
		f, err := fillFromString(fill)
		if err != nil {
			return patch, err
		}
		patch.fill = f
	}
	if border != "" {
		b, err := borderFromString(border)
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
		b, err := borderFromString(*v.Border)
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

// fillFromString은 `none` 또는 색 토큰명을 *soml.Fill로 눕힌다 (theme.go의 colorFill/NoFill과
// 같은 결과). 테마 데이터는 토큰만 쓴다 (리터럴 금지 — theme.go의 근거).
func fillFromString(s string) (*soml.Fill, error) {
	if s == noneValue {
		return ptr(soml.NoFill()), nil
	}
	return colorFill(s), nil
}

// borderFromString은 `none` 또는 `<굵기> <스타일> <색>` 축약을 *soml.Border로 눕힌다 —
// resolver.border의 값 문법과 같고, `1 solid muted`는 theme.go의 solidBorder(1,"muted")와 같다.
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

// loadLayouts는 dir/layouts/ 아래 *.css를 레이아웃 이름→CSS 원문 맵으로 읽고, 그 키 집합이
// manifest.Layouts 선언과 정확히 일치하는지 **양방향으로** 대조한다. 선언과 파일이 어긋나면
// 저자가 오버라이드/구현이 먹었다고 착각하므로 조용히 넘기지 않고 에러다 (닫힌 어휘,
// manifest.Extends·components.json과 같은 태도). CSS 내용은 여기서 보지 않는다 — 배치·외관
// 경계 검사는 theme_lint.go의 몫이다. 빈(주석만) CSS는 허용한다 (스텁, 실체는 10/06).
func loadLayouts(fsys fs.FS, dir string, manifest themeManifest) (map[string]string, error) {
	entries, err := fs.ReadDir(fsys, path.Join(dir, "layouts"))
	if err != nil {
		return nil, fmt.Errorf("layouts/를 읽을 수 없다: %w", err)
	}
	files := map[string]string{}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".css") {
			continue
		}
		// 슬롯 영역 파일(`<이름>.slots.css`)은 loadSlotLayouts의 몫이다 — 여기서 걸러내지
		// 않으면 `title.slots`라는 존재하지 않는 흐름 레이아웃으로 오인해 "선언에 없다"
		// 에러가 난다 (양방향 대조가 슬롯 파일을 미선언 흐름 파일로 읽는다).
		if strings.HasSuffix(e.Name(), ".slots.css") {
			continue
		}
		raw, err := fs.ReadFile(fsys, path.Join(dir, "layouts", e.Name()))
		if err != nil {
			return nil, fmt.Errorf("layouts/%s를 읽을 수 없다: %w", e.Name(), err)
		}
		files[strings.TrimSuffix(e.Name(), ".css")] = string(raw)
	}

	// 선언 → 파일: manifest.Layouts는 순서 있는 슬라이스라 순회가 이미 결정적이다.
	declared := map[string]bool{}
	for _, name := range manifest.Layouts {
		declared[name] = true
		if _, ok := files[name]; !ok {
			return nil, fmt.Errorf("theme.json의 layouts에 %q가 있지만 layouts/%s.css가 없다", name, name)
		}
	}
	// 파일 → 선언: files는 맵이라 순서가 없으므로 정렬해 진단이 결정적이게 한다.
	var undeclared []string
	for name := range files {
		if !declared[name] {
			undeclared = append(undeclared, name)
		}
	}
	sort.Strings(undeclared)
	if len(undeclared) > 0 {
		return nil, fmt.Errorf("layouts/%s.css가 있지만 theme.json의 layouts 선언에 없다", undeclared[0])
	}
	return files, nil
}

// loadSlotLayouts는 dir/layouts/ 아래 `*.slots.css`를 레이아웃 이름→슬롯 영역 CSS 원문 맵으로
// 읽는다 (ADR 0035). 키는 `.slots.css` 앞부분이다 (`title.slots.css` → `title`).
//
// loadLayouts와 달리 대조는 **단방향**이다 — 슬롯 영역 파일은 선택이므로(슬롯이 없는 canvas는
// 파일이 없어도 된다) "선언됐는데 파일이 없다"는 에러가 아니다. 반대로 선언되지 않은
// 레이아웃의 슬롯 파일(`corp-hero.slots.css`인데 layouts에 corp-hero가 없음)은 저자가 오타를
// 냈거나 흐름 레이아웃 선언을 빠뜨린 것이므로 에러다 (loadLayouts의 미선언 파일과 같은 태도).
// CSS 내용은 여기서 보지 않는다 — 배치·외관 경계 검사는 theme_lint.go의 몫이다.
func loadSlotLayouts(fsys fs.FS, dir string, manifest themeManifest) (map[string]string, error) {
	entries, err := fs.ReadDir(fsys, path.Join(dir, "layouts"))
	if err != nil {
		return nil, fmt.Errorf("layouts/를 읽을 수 없다: %w", err)
	}
	declared := map[string]bool{}
	for _, name := range manifest.Layouts {
		declared[name] = true
	}
	files := map[string]string{}
	var undeclared []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".slots.css") {
			continue
		}
		name := strings.TrimSuffix(e.Name(), ".slots.css")
		if !declared[name] {
			undeclared = append(undeclared, name)
			continue
		}
		raw, err := fs.ReadFile(fsys, path.Join(dir, "layouts", e.Name()))
		if err != nil {
			return nil, fmt.Errorf("layouts/%s를 읽을 수 없다: %w", e.Name(), err)
		}
		files[name] = string(raw)
	}
	// files는 맵이라 순서가 없으므로 정렬해 진단이 결정적이게 한다 (loadLayouts와 같은 태도).
	sort.Strings(undeclared)
	if len(undeclared) > 0 {
		return nil, fmt.Errorf("layouts/%s.slots.css가 있지만 theme.json의 layouts 선언에 %q가 없다", undeclared[0], undeclared[0])
	}
	return files, nil
}

// loadBuiltinTheme는 내장 테마를 이름으로 읽는다. 내장 파일은 빌드 시점에 고정되므로 로드
// 실패는 프로그래머 오류다 (regexp.MustCompile과 같은 자리) — 호출자(minimalTheme)가 에러를
// 다룰 자리가 없고, TestLoadBuiltinMinimal이 이 불변을 못 박는다.
func loadBuiltinTheme(name string) themePackage {
	pkg, err := loadTheme(builtinThemes, path.Join("themes", name))
	if err != nil {
		panic(fmt.Sprintf("내장 테마 %q 로드 실패: %v", name, err))
	}
	return pkg
}
