package testkit_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/k2sebeom/balpyo/internal/testkit"
)

// exampleSlideCounts는 예제 덱의 장수다 — 예제를 고치는 사람이 함께 고치는 표다.
//
// 왜 숫자를 손으로 적는가: 이 게이트가 놓칠 수 있는 가장 조용한 사고가 "덱에서 장이 사라졌는데
// 남은 장이 깨끗하게 빌드되는 것"이다. 경로가 둘 있다 — 파일 이름을 `_`로 시작하게 고치는 것
// (작업 중 관례라 수집에서 빠진다)과 파일을 지우는 것. 숫자 하나가 그 둘을 빨간불로 만든다.
var exampleSlideCounts = map[string]int{
	"balpyo-intro":        12,
	"ec2-intro":           13,
	"incident-postmortem": 10,
	"quarterly-review":    7,
}

// TestExamplesBuildClean은 예제의 게이트다: 예제 덱이 **두 타깃으로 빌드되고 진단이
// 0건**이어야 한다.
//
// 이것이 CI에서 예제를 빌드한다는 요구의 실체다 — 잡의 이름이 아니라 실패의 유무가 그 사실이고,
// 이 테스트는 `go test ./...`에 있으므로 `make test`가 곧 게이트다.
func TestExamplesBuildClean(t *testing.T) {
	examples := testkit.Examples(t)
	if len(examples) != len(exampleSlideCounts) {
		t.Fatalf("예제 %d개를 찾았는데 장수 표에는 %d개가 있다 — 예제를 더했거나 지웠으면 exampleSlideCounts를 함께 고친다",
			len(examples), len(exampleSlideCounts))
	}
	for _, example := range examples {
		t.Run(example.Name, func(t *testing.T) {
			want, declared := exampleSlideCounts[example.Name]
			if !declared {
				t.Fatalf("예제 %q의 장수가 exampleSlideCounts에 없다 — 새 예제는 그 표에 한 줄을 더해야 게이트에 든다", example.Name)
			}
			result := example.Build(t)
			// 진단은 하나도 없어야 한다. Strict 빌드는 경고에서 이미 실패하므로 여기 남는 것은
			// 승격되지 않는 진단뿐이지만, 그 구멍이 생기면 이 줄이 잡는다.
			for _, d := range result.Diagnostics {
				t.Errorf("예제 %s에 진단이 남았다: %s", example.Name, d.String())
			}
			if got := len(result.Deck.Slides); got != want {
				t.Errorf("예제 %s의 슬라이드가 %d장이다 (선언: %d장) — 장이 빠졌거나 표가 낡았다",
					example.Name, got, want)
			}
			// 두 타깃의 산출물이 모두 나와야 한다: 예제가 PPTX에서만 막히는 어휘를 밟고도
			// 웹만 보고 통과하는 길을 막는다.
			if len(result.Written) < 2 {
				t.Errorf("예제 %s의 산출물이 %v다 — 웹과 PPTX 둘 다 나와야 한다", example.Name, result.Written)
			}
			// PPTX는 패키지 규칙과 ECMA-376 스키마를 지나야 한다 — 어휘가 겹친 실제 덱이
			// 픽스처 하나하나가 닿지 않는 조합을 밟는다.
			for _, name := range result.Written {
				if !strings.HasSuffix(name, ".pptx") {
					continue
				}
				data, err := os.ReadFile(filepath.Join(result.OutDir, filepath.FromSlash(name)))
				if err != nil {
					t.Fatal(err)
				}
				testkit.AssertValidPPTX(t, data)
			}
		})
	}
}

// TestExampleThemesBuildEveryDeck은 예제 테마의 게이트다: 테마마다 예제 덱 전부가 두 타깃으로
// **경고 없이** 빌드되어야 한다. 덱이 쓰는 variant · 슬롯 · 레이아웃을 테마가 다 갖추고, 테마의
// 크기로도 넘치는 자리가 없다는 뜻이다.
//
// 예제 테마는 혼자 선다(`extends`가 비어 있다): 사용자가 디렉토리 하나만 복사해 쓰는 견본이므로,
// 부모 테마에 기대면 복사한 쪽에서 무엇이 어디서 왔는지 보이지 않는다.
func TestExampleThemesBuildEveryDeck(t *testing.T) {
	examples := testkit.Examples(t)
	for _, theme := range testkit.ExampleThemes(t) {
		t.Run(theme.Name, func(t *testing.T) {
			data, err := os.ReadFile(filepath.Join(theme.Dir, "theme.json"))
			if err != nil {
				t.Fatal(err)
			}
			var manifest struct {
				Name    string `json:"name"`
				Extends string `json:"extends"`
			}
			if err := json.Unmarshal(data, &manifest); err != nil {
				t.Fatalf("theme.json을 읽을 수 없다: %v", err)
			}
			if manifest.Name != theme.Name {
				t.Errorf("theme.json의 name이 %q다 — 디렉토리 이름 %q와 같아야 한다", manifest.Name, theme.Name)
			}
			if manifest.Extends != "" {
				t.Errorf("예제 테마 %s가 %q를 물려받는다 — 예제 테마는 혼자 서야 한다 (extends는 비운다)", theme.Name, manifest.Extends)
			}
			for _, example := range examples {
				t.Run(example.Name, func(t *testing.T) {
					result := example.BuildWithTheme(t, theme)
					for _, d := range result.Diagnostics {
						t.Errorf("%s(테마 %s)에 진단이 남았다: %s", example.Name, theme.Name, d.String())
					}
					for _, name := range result.Written {
						if !strings.HasSuffix(name, ".pptx") {
							continue
						}
						data, err := os.ReadFile(filepath.Join(result.OutDir, filepath.FromSlash(name)))
						if err != nil {
							t.Fatal(err)
						}
						testkit.AssertValidPPTX(t, data)
					}
				})
			}
		})
	}
}
