package core

import (
	"fmt"
	"io/fs"
	"path"
	"sort"
	"strings"
)

// 이 파일은 **외부 테마 패키지**의 자리다 — 두 가지를 담는다:
//
//  1. **이름의 어휘**: 테마는 이름이고, 그 이름을 찾는 자리가 둘이다 — 바이너리에 내장된 셋과
//     `<프로젝트>/themes/<이름>/`. 경로 문법은 없다: 값의 종류가 둘이 되면(이름이면 내장,
//     `/`가 있으면 경로) 진단이 두 어휘를 동시에 설명해야 한다. 테마가 `fs.FS` 안에서만 읽히는
//     것이 `..`·절대 경로를 막는 방법이기도 하다.
//  2. **`extends`의 병합**: 규칙은 하나다 — 이름이 같으면 자식이 이기고, 없던 이름은 부모 순서
//     뒤에 붙는다. 병합은 **원문 선언 층**(rawTheme)에서 일어나고 변환·완전성 검사는 병합
//     결과에 한 번 돈다. 이름이 없는 싱글톤 절은 필드 단위, 레이아웃 CSS는 파일 단위다.
//
// 병합을 원문 층에 둔 이유는 규칙이 한 문장으로 적힌다는 것이다: 변환된 패치(textPatch·
// shapePatch·cellPatch…)를 병합하면 "이 필드는 상속되는가"를 여덟 타입에서 따로 답해야 하고,
// 그만큼 흩어진 규칙은 문서가 정확히 적을 수 없다.

// ThemeNames는 이 프로젝트에서 쓸 수 있는 테마 이름이다 — **내장 ∪ 프로젝트 로컬**, 정렬.
//
// fsys는 프로젝트 루트다 (nil이면 내장 테마만). 공개하는 이유는 BuiltinThemes와 같다 (어휘
// 단일 출처): `--theme`의 진단과 설정 검사가 같은 목록을 말해야 한다.
func ThemeNames(fsys fs.FS) []string {
	names := BuiltinThemes()
	for _, name := range localThemeNames(fsys) {
		// 내장과 겹치는 이름은 여기서 두 번 실리지 않는다 — 그 겹침 자체는 CheckThemeName이
		// 에러로 잡는다 (목록을 만드는 자리가 판정하지 않는다).
		if !IsBuiltinTheme(name) {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	return names
}

// localThemeNames는 `<프로젝트>/themes/` 아래 디렉토리 이름이다 (정렬). fsys가 nil이거나
// `themes/`가 없으면 빈 목록이다 — 로컬 테마가 없는 프로젝트가 정상이고, 그 사실이 에러가
// 되어서는 안 된다.
func localThemeNames(fsys fs.FS) []string {
	if fsys == nil {
		return nil
	}
	entries, err := fs.ReadDir(fsys, themeDir)
	if err != nil {
		return nil
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	return names
}

// CheckThemeName은 이름이 이 프로젝트의 어휘 안인지 본다.
//
// 설정 로드(LoadConfig)와 CLI 플래그가 이 함수를 쓴다 — 값 층도 같은 판정을 하지만(Resolve)
// 여기서 먼저 멈추는 이유는 에러의 질이다: 고칠 자리(`balpyo.json` 또는 방금 친 명령)를
// 가리키고 쓸 수 있는 이름을 나열한다.
func CheckThemeName(fsys fs.FS, name string) error {
	if err := checkThemeShadowing(fsys); err != nil {
		return err
	}
	if _, _, ok := themeSource(fsys, name); !ok {
		return fmt.Errorf("알 수 없는 테마 %q (쓸 수 있는 테마: %s)",
			name, strings.Join(ThemeNames(fsys), ", "))
	}
	return nil
}

// checkThemeShadowing은 로컬 테마가 내장 이름을 가리지 않는지 본다.
//
// 요청한 테마가 무엇이든 멈추는 이유는 겹친 이름이 **답할 수 없는 질문**을 만든다는 것이다:
// `extends: "minimal"`이 프로젝트의 minimal인가 내장 minimal인가. 어느 쪽이 이기는지 규칙으로
// 적을 수는 있지만 그 규칙을 외운 사람만 자기 테마가 어디서 왔는지 안다 — 이름을 바꾸는 쪽이
// 싸다.
func checkThemeShadowing(fsys fs.FS) error {
	for _, name := range localThemeNames(fsys) {
		if IsBuiltinTheme(name) {
			return fmt.Errorf("%s/%s/가 내장 테마 %q와 이름이 같다 — 로컬 테마의 이름을 바꾼다",
				themeDir, name, name)
		}
	}
	return nil
}

// themeSource는 이름 하나의 파일 자리다 — 내장이면 임베드 FS, 로컬이면 프로젝트 FS다. 두
// 출처가 **같은 로더**를 지난다 — 로더가 fs.FS 위에서 도는 이유다.
func themeSource(fsys fs.FS, name string) (fs.FS, string, bool) {
	if IsBuiltinTheme(name) {
		return builtinThemes, path.Join(themeDir, name), true
	}
	for _, local := range localThemeNames(fsys) {
		if local == name {
			return fsys, path.Join(themeDir, name), true
		}
	}
	return nil, "", false
}

// loadThemeNamed는 이름 하나를 어휘에서 찾아 로드한다 — `extends` 체인을 원문 층에서 병합한 뒤
// 변환·검사를 병합 결과에 **한 번** 돌린다. fsys가 nil이면 내장 테마만 보인다.
func loadThemeNamed(fsys fs.FS, name string) (themePackage, error) {
	if err := checkThemeShadowing(fsys); err != nil {
		return themePackage{}, err
	}
	chain, err := readThemeChain(fsys, name)
	if err != nil {
		return themePackage{}, err
	}
	merged := chain[0]
	for _, child := range chain[1:] {
		merged = mergeRawTheme(merged, child)
	}
	pkg, err := buildThemePackage(merged)
	if err != nil {
		return pkg, err
	}
	pkg.lint = lintThemeChain(chain)
	return pkg, nil
}

// readThemeChain은 뿌리부터 잎까지의 원문 선언이다 (`minimal` → `corp` → `corp-dark`).
//
// 순환은 체인을 그대로 적어 멈춘다 — 스택을 태우는 대신 고칠 자리를 보여야 한다. 디렉토리
// 이름과 매니페스트의 `name`이 같아야 한다는 규칙도 여기서 지킨다: 요청은
// 이름으로 오고 파일은 경로로 찾으므로, 둘이 갈리면 어느 이름으로 요청해야 하는지를 파일을
// 열어 봐야 안다.
func readThemeChain(fsys fs.FS, name string) ([]rawTheme, error) {
	var chain []rawTheme
	var seen []string
	for {
		for _, prev := range seen {
			if prev == name {
				return nil, fmt.Errorf("theme.json의 extends가 순환한다: %s → %s",
					strings.Join(seen, " → "), name)
			}
		}
		seen = append(seen, name)

		src, dir, ok := themeSource(fsys, name)
		if !ok {
			if len(chain) == 0 {
				return nil, fmt.Errorf("알 수 없는 테마 %q (쓸 수 있는 테마: %s)",
					name, strings.Join(ThemeNames(fsys), ", "))
			}
			// 부모가 없는 경우는 **자식을 가리켜** 멈춘다 — 고칠 자리가 그 자식의 theme.json이다.
			return nil, fmt.Errorf("테마 %q의 extends가 없는 테마 %q를 가리킨다 (쓸 수 있는 테마: %s)",
				seen[len(seen)-2], name, strings.Join(ThemeNames(fsys), ", "))
		}
		raw, err := readRawTheme(src, dir)
		if err != nil {
			return nil, fmt.Errorf("테마 %q: %w", name, err)
		}
		if raw.manifest.Name != name {
			return nil, fmt.Errorf("%s/theme.json의 name이 %q다 — 디렉토리 이름 %q와 같아야 한다",
				dir, raw.manifest.Name, name)
		}
		chain = append(chain, raw)
		if raw.manifest.Extends == "" {
			break
		}
		name = raw.manifest.Extends
	}
	// 잎에서 뿌리로 모았으므로 뒤집는다 — 병합은 뿌리 위에 잎을 얹는 순서다.
	for i, j := 0, len(chain)-1; i < j; i, j = i+1, j-1 {
		chain[i], chain[j] = chain[j], chain[i]
	}
	return chain, nil
}

// ── 병합: 이름이 같으면 자식이 이기고, 없던 이름은 부모 순서 뒤에 붙는다 ─────

// mergeRawTheme는 부모 위에 자식을 얹는다.
func mergeRawTheme(parent, child rawTheme) rawTheme {
	return rawTheme{
		manifest:    mergeManifest(parent.manifest, child.manifest),
		tokens:      mergeTokensFile(parent.tokens, child.tokens),
		components:  mergeComponentsFile(parent.components, child.components),
		layouts:     mergeCSSFiles(parent.layouts, child.layouts),
		slotLayouts: mergeCSSFiles(parent.slotLayouts, child.slotLayouts),
	}
}

// mergeNamed는 이름 붙은 표 둘을 병합한다 — 이 파일의 규칙 한 문장이 사는 자리다.
//
// 부모 순서를 유지하는 것이 규칙의 일부인 이유는 **순서가 산출물에 보인다**는 것이다: PPTX
// clrScheme의 슬롯 순서, 웹 CSS 변수 선언 순서, 모르는 variant 경고가 나열하는 목록. 자식이
// 더한 이름을 뒤에 붙이면 부모 테마로 만든 산출물의 그 순서가 그대로 남는다.
func mergeNamed[T any](parent, child []T, nameOf func(T) string) []T {
	if len(child) == 0 {
		return parent
	}
	override := make(map[string]int, len(child))
	for i, item := range child {
		override[nameOf(item)] = i
	}
	out := make([]T, 0, len(parent)+len(child))
	taken := make(map[string]bool, len(child))
	for _, item := range parent {
		name := nameOf(item)
		if i, ok := override[name]; ok {
			out = append(out, child[i])
			taken[name] = true
			continue
		}
		out = append(out, item)
	}
	for _, item := range child {
		if !taken[nameOf(item)] {
			out = append(out, item)
		}
	}
	return out
}

// mergeCSSFiles는 레이아웃 CSS를 **파일 단위**로 병합한다: 자식이 그 파일을 두면 내 것,
// 두지 않으면 부모 것이다 (선언 단위 병합은 저자가 두 파일을 동시에
// 머리에 담아야 예측할 수 있고, 배치는 이 제품에서 가장 되돌리기 어려운 층이다).
func mergeCSSFiles(parent, child map[string]string) map[string]string {
	out := make(map[string]string, len(parent)+len(child))
	for name, css := range parent {
		out[name] = css
	}
	for name, css := range child {
		out[name] = css
	}
	return out
}

// mergeManifest는 매니페스트를 병합한다. 이름과 `extends`는 자식의 것이고(결과의 이름은 요청한
// 이름이다), 크롬 선언은 `kind`를 이름으로 삼아 같은 규칙을 받는다. `layouts`는 자식이
// 생략하면 부모의 목록이다 — 표준 11종 계약을 자식이 다시 적지 않아도
// 받는다.
func mergeManifest(parent, child themeManifest) themeManifest {
	out := child
	if len(child.Layouts) == 0 {
		out.Layouts = parent.Layouts
	}
	out.Chrome = mergeNamed(parent.Chrome, child.Chrome, func(d chromeDeclFile) string { return d.Kind })
	return out
}

// mergeTokensFile은 토큰 표 다섯을 이름 단위로 병합한다.
func mergeTokensFile(parent, child tokensFile) tokensFile {
	return tokensFile{
		Color:  mergeNamed(parent.Color, child.Color, func(t colorTok) string { return t.Name }),
		Font:   mergeNamed(parent.Font, child.Font, func(t fontTok) string { return t.Name }),
		Space:  mergeNamed(parent.Space, child.Space, func(t spaceTok) string { return t.Name }),
		Radius: mergeNamed(parent.Radius, child.Radius, func(t radiusTok) string { return t.Name }),
		Shadow: mergeNamed(parent.Shadow, child.Shadow, func(t shadowTok) string { return t.Name }),
	}
}

// mergeComponentsFile은 컴포넌트 선언을 병합한다. 이름 붙은 표(역할·컴포넌트 기본값·variant·
// 코드 토큰)는 이름 단위로 **한 벌 통째로** 갈리고, 이름이 없는 싱글톤 절(shape·image의
// default, 표의 cell·header·stripe, code의 tabSize)은 **필드 단위**로 갈린다.
//
// 역할과 variant가 필드 단위가 아닌 이유는 그 둘이 이미 이름을 가진 한 벌이라는 것이다: 자식이
// `title`을 다시 적었다면 그 역할의 타이포그래피를 자기 것으로 정한 것이고, 부모의 크기가
// 반쯤 남으면 그 결과를 예측할 수 있는 사람이 없다.
func mergeComponentsFile(parent, child componentsFile) componentsFile {
	var out componentsFile
	out.Comment = firstNonEmpty(child.Comment, parent.Comment)

	out.Text.Comment = firstNonEmpty(child.Text.Comment, parent.Text.Comment)
	out.Text.Roles = mergeNamed(parent.Text.Roles, child.Text.Roles,
		func(r textRoleFile) string { return r.Name })
	out.Text.ComponentDefaults = mergeNamed(parent.Text.ComponentDefaults, child.Text.ComponentDefaults,
		func(d textComponentDefaultFile) string { return d.key() })
	out.Text.Variants = mergeNamed(parent.Text.Variants, child.Text.Variants,
		func(v textVariantFile) string { return v.Name })

	out.Shape.Comment = firstNonEmpty(child.Shape.Comment, parent.Shape.Comment)
	out.Shape.Default = shapeStyleFile{
		Fill:   firstNonEmpty(child.Shape.Default.Fill, parent.Shape.Default.Fill),
		Border: firstNonEmpty(child.Shape.Default.Border, parent.Shape.Default.Border),
		Radius: firstNonEmpty(child.Shape.Default.Radius, parent.Shape.Default.Radius),
	}
	out.Shape.Variants = mergeNamed(parent.Shape.Variants, child.Shape.Variants,
		func(v shapeVariantFile) string { return v.Name })

	out.Image.Comment = firstNonEmpty(child.Image.Comment, parent.Image.Comment)
	out.Image.Variants = mergeNamed(parent.Image.Variants, child.Image.Variants,
		func(v imageVariantFile) string { return v.Name })

	out.Table = tableComponentsFile{
		Comment: firstNonEmpty(child.Table.Comment, parent.Table.Comment),
		Cell:    mergeCellStyleFile(parent.Table.Cell, child.Table.Cell),
		Header:  mergeCellStyleFile(parent.Table.Header, child.Table.Header),
		Stripe:  firstNonEmpty(child.Table.Stripe, parent.Table.Stripe),
	}

	out.Code = codeComponentsFile{
		Comment: firstNonEmpty(child.Code.Comment, parent.Code.Comment),
		TabSize: child.Code.TabSize,
		Tokens: mergeNamed(parent.Code.Tokens, child.Code.Tokens,
			func(t codeTokenFile) string { return t.Class }),
	}
	if out.Code.TabSize == 0 {
		out.Code.TabSize = parent.Code.TabSize
	}
	return out
}

// mergeCellStyleFile은 셀 외관 절을 필드 단위로 병합한다 (빈 문자열·nil이 "정하지 않음").
func mergeCellStyleFile(parent, child cellStyleFile) cellStyleFile {
	out := cellStyleFile{
		Fill:   firstNonEmpty(child.Fill, parent.Fill),
		Border: firstNonEmpty(child.Border, parent.Border),
		Color:  firstNonEmpty(child.Color, parent.Color),
		Weight: child.Weight,
	}
	if out.Weight == nil {
		out.Weight = parent.Weight
	}
	return out
}

// firstNonEmpty는 "빈 값은 정하지 않음"을 한 줄로 적는 도구다 — 싱글톤 절의 필드 단위 병합이
// 전부 이 모양이다.
func firstNonEmpty(child, parent string) string {
	if child != "" {
		return child
	}
	return parent
}
