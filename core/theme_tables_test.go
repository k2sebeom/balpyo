package core

import "github.com/k2sebeom/balpyo/soml"

// 이 파일은 옛 하드코딩 표(core/theme.go)의 **테스트 헬퍼 판**이다.
//
// P4.2가 그 파일을 지웠다 (ADR 0052의 결정 4): 엔진에 외관 값이 남아 있으면 그것이 폴백이 되고,
// 폴백은 테마의 누락을 조용히 메운다. 그런데 캐스케이드 테스트(cascade_test·resolve_test·
// table_test·inline_test·image_test·layout_test)는 "역할 기본값 위에 이 패치가 얹히면 무엇이
// 되는가"를 재므로 **아래 단의 값을 알 필요**가 있다.
//
// 그 값을 여기서 다시 적지 않고 **로드된 minimal에서 파생한다** (minimalFonts·minimalShadows가
// 먼저 쓴 방식이다): 헬퍼와 파일이 어긋날 수 없고, 값 자체를 못 박는 일은 기대값이 리터럴로
// 적힌 앵커 테스트가 맡는다 (TestLoadBuiltinMinimal · TestLoadBuiltinComponents). 그래서 이
// 헬퍼들이 도는 자리에서 값이 움직이면 앵커가 먼저 빨개진다.
//
// 테마를 인자로 받는 시그니처를 그대로 둔 이유는 호출 자리의 뜻이다: 도형·이미지 표는 반경·
// 그림자 토큰을 해석하므로 "어느 테마의 표인가"가 값의 일부다 (아래 함수들은 minimal만 안다).

// minimalComponents는 내장 minimal의 컴포넌트 표다.
func minimalComponents() components { return loadBuiltinTheme("minimal").components }

// minimalTheme는 내장 minimal의 토큰 표다.
func minimalTheme() soml.Theme { return loadBuiltinTheme("minimal").tokens }

func minimalFonts() []soml.FontToken     { return minimalTheme().Fonts }
func minimalShadows() []soml.ShadowToken { return minimalTheme().Shadows }

func textRoleDefaults(role soml.TextRole) soml.TextStyle {
	return minimalComponents().textRoles[role]
}

func textComponentDefault(tag string) textPatch {
	return minimalComponents().textComponentDefaults[tag]
}

func textVariants() []textVariant { return minimalComponents().textVariants }

func textVariantPatch(name string) (textPatch, bool) {
	for _, v := range textVariants() {
		if v.name == name {
			return v.patch, true
		}
	}
	return textPatch{}, false
}

func textVariantNames() []string {
	variants := textVariants()
	names := make([]string, len(variants))
	for i, v := range variants {
		names[i] = v.name
	}
	return names
}

func shapeComponentDefault(soml.Theme) shapePatch { return minimalComponents().shapeDefault }

func shapeVariants(soml.Theme) []shapeVariant { return minimalComponents().shapeVariants }

func shapeVariantPatch(theme soml.Theme, name string) (shapePatch, bool) {
	for _, v := range shapeVariants(theme) {
		if v.name == name {
			return v.patch, true
		}
	}
	return shapePatch{}, false
}

func shapeVariantNames(theme soml.Theme) []string {
	variants := shapeVariants(theme)
	names := make([]string, len(variants))
	for i, v := range variants {
		names[i] = v.name
	}
	return names
}

func imageComponentDefault() imagePatch { return minimalComponents().imageDefault }

func imageVariants(soml.Theme) []imageVariant { return minimalComponents().imageVariants }

func imageVariantPatch(theme soml.Theme, name string) (imagePatch, bool) {
	for _, v := range imageVariants(theme) {
		if v.name == name {
			return v.patch, true
		}
	}
	return imagePatch{}, false
}

func imageVariantNames(theme soml.Theme) []string {
	variants := imageVariants(theme)
	names := make([]string, len(variants))
	for i, v := range variants {
		names[i] = v.name
	}
	return names
}
