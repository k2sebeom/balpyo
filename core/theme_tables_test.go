package core

import "github.com/k2sebeom/balpyo/soml"

// 캐스케이드 테스트가 아래 단의 값을 알기 위한 헬퍼다. 값을 다시 적지 않고 로드된 minimal에서
// 파생하므로 헬퍼와 파일이 어긋날 수 없고, 값 자체는 앵커 테스트(TestLoadBuiltinMinimal ·
// TestLoadBuiltinComponents)가 리터럴로 못 박는다.

// minimalComponents는 내장 minimal의 컴포넌트 표다.
func minimalComponents() components { return loadBuiltinTheme("minimal").components }

// minimalTheme는 내장 minimal의 토큰 표다.
func minimalTheme() soml.Theme { return loadBuiltinTheme("minimal").tokens }

func minimalShadows() []soml.ShadowToken { return minimalTheme().Shadows }

func textRoleDefaults(role soml.TextRole) soml.TextStyle {
	return minimalComponents().textRoles[role]
}

func textVariantNames() []string {
	var names []string
	for _, v := range minimalComponents().textVariants {
		names = append(names, v.name)
	}
	return names
}

func shapeComponentDefault(soml.Theme) shapePatch { return minimalComponents().shapeDefault }

func shapeVariantNames(soml.Theme) []string {
	var names []string
	for _, v := range minimalComponents().shapeVariants {
		names = append(names, v.name)
	}
	return names
}

func imageComponentDefault() imagePatch { return minimalComponents().imageDefault }
