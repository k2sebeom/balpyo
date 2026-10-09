package core

import (
	"path"
	"sort"
	"strings"

	"github.com/k2sebeom/balpyo/soml"
)

// 이 파일은 **레이아웃 CSS 배치·외관 경계 린터**다.
//
// `style` 탈출구의 배치 전용 경계를 레이아웃 CSS로 확장한다. 계약은 style.go와 같다:
// 선언을 `프로퍼티: 값`으로 쪼개 프로퍼티 이름만 배치 allowlist(isLayoutStyleProperty)와
// 대조한다. **값은 검증하지 않는다** — 엔진이 CSS 값 문법을 재구현하기 시작하면 경계가
// 경계이기를 멈춘다. 문법 위반(콜론·이름·값 없음)만 에러, allowlist 밖 프로퍼티는 경고다.
//
// 경고인 이유: 색·테두리·그림자·글꼴 같은 외관은 CSS에만 남아 PPTX 마스터에는 도달하지
// 않는다 (지오메트리 패스가 재는 박스를 움직이는 프로퍼티만 양 타깃에 남는다 — style.go의
// layoutStyleProperties 근거). 외관은 components.json에 선언해야 두 타깃이 같은 룩을 얻는다.
//
// 레이아웃 CSS는 소스 위치(line:col)가 없으므로 진단 Span은 zero다 — File(프로젝트 상대 경로
// `themes/<테마>/layouts/<이름>.css`)로 어느 파일인지만 가리킨다. --strict는
// soml.Diagnostics.Strict()가 경고를 에러로 승격한다.
//
// 린터가 도는 자리는 로컬 테마 로딩 하나다 (lintThemeChain). 내장 테마는 테스트가 전수로 막으므로
// 빌드마다 다시 보지 않는다.

// lintLayoutCSS는 레이아웃 파일 하나의 CSS 원문을 검사해 진단 목록을 돌려준다. 선언 순서 그대로
// 진단하므로 결과가 결정적이다. file은 진단의 File(프로젝트 상대 경로)이다.
func lintLayoutCSS(css, file string) soml.Diagnostics {
	var diags soml.Diagnostics
	// 주석을 먼저 벗긴다 — 레이아웃 CSS 파일은 관례대로 주석을 다는데(빈 스텁조차 주석이다),
	// 선언 분해는 주석을 모르므로 주석이 선언으로 새면 거짓 "콜론 없음" 에러가 난다.
	for _, part := range splitDeclarations(stripCSSComments(css)) {
		name, value, found := strings.Cut(part.text, ":")
		if !found {
			diags = append(diags, soml.Errorf(file, soml.Span{},
				"레이아웃 CSS 선언 %q에 콜론이 없다 — `속성: 값` 형태로 쓴다", part.text))
			continue
		}
		property := strings.ToLower(strings.TrimSpace(name))
		declaration := strings.TrimSpace(value)
		switch {
		case property == "":
			diags = append(diags, soml.Errorf(file, soml.Span{},
				"레이아웃 CSS 선언 %q에 속성 이름이 없다", part.text))
			continue
		case declaration == "":
			// 빈 값은 "검증하지 않는 값"이 아니라 완성되지 않은 선언이다 — CSS도 버린다.
			diags = append(diags, soml.Errorf(file, soml.Span{},
				"레이아웃 CSS 선언 %q에 값이 없다", part.text))
			continue
		}
		if !isLayoutStyleProperty(property) {
			diags = append(diags, soml.Warningf(file, soml.Span{},
				"레이아웃 CSS의 %q 선언은 배치가 아니다 — 웹에만 적용되고 PPTX 마스터에는 반영되지 않는다. 색·테두리 같은 외관은 components.json에 선언한다",
				property))
		}
	}
	return diags
}

// lintLayouts는 레이아웃 맵 하나(이름 → CSS 원문)를 검사한다. 파일 경로는
// `<dir>/layouts/<이름><suffix>`다 (suffix는 `.css` 또는 `.slots.css`). 맵 순회 순서는 정의되지
// 않으므로 이름을 정렬해 순회한다 — 진단 순서가 실행마다 안정이어야 한다 (결정성).
func lintLayouts(dir, suffix string, layouts map[string]string) soml.Diagnostics {
	names := make([]string, 0, len(layouts))
	for name := range layouts {
		names = append(names, name)
	}
	sort.Strings(names)
	var diags soml.Diagnostics
	for _, name := range names {
		diags = append(diags, lintLayoutCSS(layouts[name], path.Join(dir, "layouts", name+suffix))...)
	}
	return diags
}

// lintThemeChain은 extends 체인(뿌리 → 잎)에서 **병합 결과에 남은** 로컬 테마의 레이아웃 파일을
// 검사한다. 병합은 파일 단위로 자식이 이기므로, 잎에서 뿌리로 내려가며 이름을
// 처음 가진 테마가 그 파일의 주인이다 — 자식이 덮어 쓴 부모의 파일은 산출물에 닿지 않으므로
// 진단하지 않고, 진단은 고칠 파일(그 주인의 경로)을 가리킨다. 주인이 내장 테마인 파일은 건너뛴다.
func lintThemeChain(chain []rawTheme) soml.Diagnostics {
	var diags soml.Diagnostics
	seenLayouts, seenSlots := map[string]bool{}, map[string]bool{}
	for i := len(chain) - 1; i >= 0; i-- {
		raw := chain[i]
		layouts := unseenFiles(raw.layouts, seenLayouts)
		slots := unseenFiles(raw.slotLayouts, seenSlots)
		if IsBuiltinTheme(raw.manifest.Name) {
			continue
		}
		dir := path.Join(themeDir, raw.manifest.Name)
		diags = append(diags, lintLayouts(dir, ".css", layouts)...)
		diags = append(diags, lintLayouts(dir, ".slots.css", slots)...)
	}
	return diags
}

// unseenFiles는 files 중 seen에 없는 것만 돌려주고, 그 이름을 seen에 적는다.
func unseenFiles(files map[string]string, seen map[string]bool) map[string]string {
	out := map[string]string{}
	for name, css := range files {
		if !seen[name] {
			out[name] = css
			seen[name] = true
		}
	}
	return out
}
