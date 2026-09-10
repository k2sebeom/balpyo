package core

import (
	"sort"
	"strings"

	"github.com/k2sebeom/balpyo/soml"
)

// 이 파일은 **레이아웃 CSS 배치·외관 경계 린터**다 (SPECS.md 10/02, PRD §8.2).
//
// ADR 0014(`style` 탈출구 경계)를 레이아웃 CSS로 확장한다. 계약은 style.go와 같다:
// 선언을 `프로퍼티: 값`으로 쪼개 프로퍼티 이름만 배치 allowlist(isLayoutStyleProperty)와
// 대조한다. **값은 검증하지 않는다** — 엔진이 CSS 값 문법을 재구현하기 시작하면 경계가
// 경계이기를 멈춘다. 문법 위반(콜론·이름·값 없음)만 에러, allowlist 밖 프로퍼티는 경고다.
//
// 경고인 이유: 색·테두리·그림자·글꼴 같은 외관은 CSS에만 남아 PPTX 마스터에는 도달하지
// 않는다 (지오메트리 패스가 재는 박스를 움직이는 프로퍼티만 양 타깃에 남는다 — style.go의
// layoutStyleProperties 근거). 외관은 components.json에 선언해야 두 타깃이 같은 룩을 얻는다.
//
// 레이아웃 CSS는 소스 위치(line:col)가 없으므로 진단 Span은 zero다 — File(layouts/<이름>.css)로
// 어느 파일인지만 가리킨다. --strict는 soml.Diagnostics.Strict()가 경고를 에러로 승격한다.

// lintLayoutCSS는 레이아웃 하나의 CSS 원문을 검사해 진단 목록을 돌려준다. 선언 순서 그대로
// 진단하므로 결과가 결정적이다. layoutName은 진단의 File(layouts/<이름>.css)에 쓴다.
func lintLayoutCSS(css, layoutName string) soml.Diagnostics {
	file := "layouts/" + layoutName + ".css"
	var diags soml.Diagnostics
	for _, part := range splitDeclarations(css) {
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
				"레이아웃 CSS의 %q 선언은 배치가 아니다 — 웹에만 적용되고 PPTX 마스터에는 반영되지 않는다 (ADR 0014). 색·테두리 같은 외관은 components.json에 선언한다",
				property))
		}
	}
	return diags
}

// lintLayouts는 로드된 레이아웃 맵 전체를 검사한다. 맵 순회 순서는 정의되지 않으므로 이름을
// 정렬해 순회한다 — 진단 순서가 실행마다 안정이어야 한다 (Task의 결정성 요구).
func lintLayouts(layouts map[string]string) soml.Diagnostics {
	names := make([]string, 0, len(layouts))
	for name := range layouts {
		names = append(names, name)
	}
	sort.Strings(names)
	var diags soml.Diagnostics
	for _, name := range names {
		diags = append(diags, lintLayoutCSS(layouts[name], name)...)
	}
	return diags
}
