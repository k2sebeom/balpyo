// 이 파일만 `render_test` 외부 패키지인 이유: 여기서 두 렌더러를 **함께** 불러야 하고
// (web·pptx는 둘 다 render를 임포트한다) 패키지 안에서 임포트하면 순환이 된다. 대조가
// 성립하는 유일한 자리가 여기다 — 어느 한 렌더러의 테스트는 자기 목록만 볼 수 있고
// "어느 렌더러도 다루지 않는 종류"는 둘을 같이 봐야 보인다.
package render_test

import (
	"testing"

	"github.com/k2sebeom/balpyo/render/pptx"
	"github.com/k2sebeom/balpyo/render/web"
	"github.com/k2sebeom/balpyo/soml"
)

// 어휘 카탈로그와 렌더러 디스패치를 대조한다 (TASKS P0.2).
//
// 이 테스트가 존재하는 이유는 사고 하나다: `<table>`이 파서·값 층까지 열린 채 렌더러가 없어
// 저자가 슬라이드를 다 쓴 뒤 빌드에서 죽었다. 닫힌 어휘의 값은 "쓸 수 있는 것은 그려진다"에
// 있으므로, 카탈로그에 종류를 더하면서 렌더러 표를 늘리지 않는 것 자체가 실패여야 한다.
//
// 양방향으로 본다 — 빠뜨린 것과 적어 두고 구현한 것 둘 다 잡는다:
//
//	KindRendered(k) 참  ⇒ 두 렌더러 **모두** k를 다룬다      (빠뜨림: 렌더에서 죽는다)
//	KindRendered(k) 거짓 ⇒ 두 렌더러 **모두** k를 다루지 않는다 (구현했는데 파서가 막는다)
//
// 여기에 **셋째 갈래**가 있다 (ADR 0059): 값 층이 펼치는 종류(`<chart>`·`<point>`)는 카탈로그에
// 있고 파서가 통과시키지만 렌더러에는 닿지 않는다 — 확정 트리에 남는 것이 도형과 텍스트뿐이기
// 때문이다. 그래서 그 종류는 렌더러가 다루면 **실패**다: 다룰 수 있다는 것은 펼침을 지나치는
// 길이 열려 있다는 뜻이고, 그러면 같은 데이터에서 두 그림이 나올 수 있다.
//
//	KindDesugared(k) 참  ⇒ 두 렌더러 **모두** k를 다루지 않는다 (값 층이 펼친다)
func TestRenderersCoverCatalog(t *testing.T) {
	handlers := map[string]map[soml.ElementKind]bool{
		"web":  kindSet(web.RenderedKinds()),
		"pptx": kindSet(pptx.RenderedKinds()),
	}
	for _, kind := range soml.ElementKinds() {
		want := soml.KindRendered(kind) && !soml.KindDesugared(kind)
		for target, handled := range handlers {
			switch {
			case want && !handled[kind]:
				t.Errorf("%s 렌더러가 %v를 다루지 않는다 (%v) — 카탈로그에 있고 파서가 통과시키는 종류는 "+
					"전부 그려져야 한다. 렌더러 표를 늘리거나 soml.unrenderedKinds에 적어라",
					target, kind, soml.ElementNamesOf(kind))
			case !want && handled[kind] && soml.KindDesugared(kind):
				t.Errorf("%s 렌더러가 %v를 다룬다 (%v) — 이 종류는 값 층이 도형·텍스트로 펼치므로 "+
					"렌더러에 닿지 않는다 (ADR 0059). 렌더러 표에서 그 줄을 지워라",
					target, kind, soml.ElementNamesOf(kind))
			case !want && handled[kind]:
				t.Errorf("%s 렌더러가 %v를 다루는데 soml.unrenderedKinds에 남아 있다 (%v) — "+
					"파서가 막고 있으니 그 줄을 지워라", target, kind, soml.ElementNamesOf(kind))
			}
		}
	}
}

// 렌더러 목록에는 카탈로그 밖의 값이 없어야 한다 — 중복도 없다. 목록이 조용히 늘어나면
// 위의 대조가 통과해도 뜻이 흐려진다 (KindNone은 `<slide>`·`<text>`·`<note>`의 자리이고
// 각 렌더러가 이름으로 직접 다룬다).
func TestRenderedKindsAreCatalogKinds(t *testing.T) {
	catalog := kindSet(soml.ElementKinds())
	for target, kinds := range map[string][]soml.ElementKind{
		"web":  web.RenderedKinds(),
		"pptx": pptx.RenderedKinds(),
	} {
		seen := make(map[soml.ElementKind]bool, len(kinds))
		for _, kind := range kinds {
			if !catalog[kind] {
				t.Errorf("%s 렌더러 목록의 %v는 카탈로그 밖의 종류다", target, kind)
			}
			if seen[kind] {
				t.Errorf("%s 렌더러 목록에 %v가 두 번 있다", target, kind)
			}
			seen[kind] = true
		}
	}
}

func kindSet(kinds []soml.ElementKind) map[soml.ElementKind]bool {
	set := make(map[soml.ElementKind]bool, len(kinds))
	for _, kind := range kinds {
		set[kind] = true
	}
	return set
}
