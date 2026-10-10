package core

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/k2sebeom/balpyo/soml"
)

// 이 파일은 **문서 게이트**다: 레퍼런스와 개요에 손으로 적힌 어휘의 닫힌 목록이
// 코드의 목록과 같은지 판정한다.
//
// 왜 진단 문구 전체가 아니라 목록인가: 문서에서 낡는 것은 목록이다 (어휘의 종류 수,
// 내장 테마 이름). 문구의 산문은 사람의
// 것이라 기계가 맞추게 하면 문서가 덤프가 되고, 문구의 회귀는 각 패키지의 테스트가 기댓값으로
// 이미 잡는다. 낡는 것은 **사본**이고, 사본에서 어휘가 틀리면 "어휘는 닫혀 있다"는 이 언어의
// 주장 자체가 거짓이 된다.
//
// 왜 core인가: 필수 토큰 목록이 이 패키지의 비공개 변수이고(공개 API를 테스트 때문에 늘리지
// 않는다), internal/testkit은 internal/pipeline을 임포트하므로 여기서 그것을 쓰면 순환이다.
// 그리고 문서만 고치는 커밋이 브라우저와 130초짜리 스위트를 기다릴 이유가 없다 — 이 판정은
// 도구 없이 밀리초에 끝난다.

// docVocabulary는 등록 한 줄이다: "이 문서의 이 패턴 안에 이 어휘의 목록이 있다."
//
// 표에 없는 자리는 지켜지지 않는다 — 그것을 표 자체가 말한다. 새 문서·새 어휘는 줄을 더해야
// 게이트에 든다.
type docVocabulary struct {
	// Doc은 docs/ 기준 경로다.
	Doc string
	// Vocab은 실패 메시지에 실리는 어휘 이름이다.
	Vocab string
	// Pattern은 목록을 담는 **캡처 그룹 하나**를 가진 정규식이다. 문서를 공백 정규화한
	// 뒤 맞추므로 줄바꿈으로 접힌 목록도 한 줄로 읽힌다.
	Pattern string
	// Sep은 목록의 구분자다 — 진단 형식(`, `)과 산문 형식(`·`)을 같은 표가 함께 받는다.
	Sep string
	// Want는 정본이다 (코드).
	Want []string
}

// docVocabularies는 문서에 사본이 있는 어휘 일곱 벌이다.
var docVocabularies = []docVocabulary{
	{
		Doc: "reference/soml/enter.md", Vocab: "등장 효과",
		Pattern: `unknown enter "spin" \(available values: ([^)]+)\)`, Sep: ", ", Want: soml.EnterEffects(),
	},
	{
		Doc: "reference/soml/connector.md", Vocab: "연결선 경로",
		Pattern: `unknown route "zigzag" \(available values: ([^)]+)\)`, Sep: ", ", Want: soml.ConnectorRoutes(),
	},
	{
		Doc: "reference/soml/connector.md", Vocab: "연결선 면",
		Pattern: `unknown to-side "north" \(available values: ([^)]+)\)`, Sep: ", ", Want: soml.ConnectorSides(),
	},
	{
		Doc: "reference/soml/connector.md", Vocab: "연결선 표식",
		Pattern: `unknown end "diamond" \(available values: ([^)]+)\)`, Sep: ", ", Want: soml.ConnectorEnds(),
	},
	{
		Doc: "reference/soml/shape.md", Vocab: "도형 프리셋",
		Pattern: `available types: ([^)]+)\)`, Sep: ", ", Want: soml.ShapeTypes(),
	},
	{
		Doc: "reference/soml/slide.md", Vocab: "표준 레이아웃",
		Pattern: `available layouts: ([^)]+)\)`, Sep: ", ", Want: docSlideLayouts(),
	},
	{
		Doc: "reference/soml/text.md", Vocab: "색 토큰",
		Pattern: `available tokens: ([^)]+), or #RRGGBB`, Sep: ", ", Want: requiredColors,
	},
	{
		Doc: "reference/soml/shape.md", Vocab: "색 토큰",
		Pattern: `available tokens: ([^)]+), or #RRGGBB`, Sep: ", ", Want: requiredColors,
	},
	// 테마 문서의 "채워야 하는 이름" 표는 로더의 완전성 검사 그 자체다 (theme_require.go의
	// 주석이 그 표를 가리킨다) — 범주마다 한 줄이므로 범주 이름으로 칸을 집는다.
	{
		Doc: "reference/theme.md", Vocab: "필수 색 토큰",
		Pattern: "\\| `color` \\| ([^|]+) \\|", Sep: "·", Want: requiredColors,
	},
	{
		Doc: "reference/theme.md", Vocab: "필수 폰트 토큰",
		Pattern: "\\| `font` \\| ([^|]+) \\|", Sep: "·", Want: docFontRoles(),
	},
	{
		Doc: "reference/theme.md", Vocab: "필수 공간 토큰",
		Pattern: "\\| `space` \\| ([^|]+) \\|", Sep: "·", Want: requiredSpaces,
	},
	{
		Doc: "reference/theme.md", Vocab: "필수 반경 토큰",
		Pattern: "\\| `radius` \\| ([^|]+) \\|", Sep: "·", Want: requiredRadii,
	},
	{
		Doc: "reference/theme.md", Vocab: "필수 그림자 토큰",
		Pattern: "\\| `shadow` \\| ([^|]+) \\|", Sep: "·", Want: requiredShadows,
	},
	{
		Doc: "reference/theme.md", Vocab: "텍스트 역할",
		Pattern: `has no role "[^"]*" \(required: ([^)]+)\)`, Sep: ", ", Want: docTextRoles(),
	},
	{
		Doc: "reference/soml/layout.md", Vocab: "공간 토큰",
		Pattern: `available tokens: ([^)]+), or a number or percentage`, Sep: ", ", Want: requiredSpaces,
	},
	{
		Doc: "reference/soml/shape.md", Vocab: "반경 토큰",
		Pattern: `use a token \(([^)]+)\) or a number`, Sep: ", ", Want: requiredRadii,
	},
	{
		Doc: "reference/soml/shape.md", Vocab: "그림자 토큰",
		Pattern: `shadow token[^(]*\(available tokens: ([^)]+)\)`, Sep: ", ", Want: requiredShadows,
	},
	{
		Doc: "reference/index.md", Vocab: "내장 테마",
		Pattern: `내장 셋\(([^)]+)\)`, Sep: "·", Want: BuiltinThemes(),
	},
	{
		Doc: "reference/cli.md", Vocab: "내장 테마",
		Pattern: `내장 셋\(([^)]+)\)`, Sep: "·", Want: BuiltinThemes(),
	},
	{
		Doc: "index.md", Vocab: "내장 테마",
		Pattern: `내장 셋\(([^)]+)\)`, Sep: "·", Want: BuiltinThemes(),
	},
	// **태그가 여는 속성 목록**이다 ("unknown attribute" 진단의 사본). 이 자리를 뒤늦게 등록하는
	// 이유는 속성을 새로 열 때 문서의 사본이 따라오지 않기 쉽기 때문이다
	// (문서를 읽은 저자는 쓸 수 있는 속성을 못 쓴다고 믿는다 — 닫힌 목록의 낡음 중 가장 비싼
	// 종류다). 태그 이름을 패턴에 넣어 한 문서의 여러 목록(`<table>`과 `<td>`)이 섞이지 않게 한다.
	{
		Doc: "reference/soml/shape.md", Vocab: "<shape>의 속성",
		Pattern: `unknown attribute [a-z-]+ on <shape>[^(]*\(available attributes: ([^)]+)\)`,
		Sep:     ", ", Want: soml.ElementAttrs(soml.ShapeElement),
	},
	{
		Doc: "reference/soml/image.md", Vocab: "<image>의 속성",
		Pattern: `unknown attribute [a-z-]+ on <image>[^(]*\(available attributes: ([^)]+)\)`,
		Sep:     ", ", Want: soml.ElementAttrs(soml.ImageElement),
	},
	{
		Doc: "reference/soml/table.md", Vocab: "<table>의 속성",
		Pattern: `unknown attribute [a-z-]+ on <table>[^(]*\(available attributes: ([^)]+)\)`,
		Sep:     ", ", Want: soml.ElementAttrs(soml.TableElement),
	},
	{
		Doc: "reference/soml/table.md", Vocab: "<td>의 속성",
		Pattern: `unknown attribute [a-z-]+ on <td>[^(]*\(available attributes: ([^)]+)\)`,
		Sep:     ", ", Want: soml.ElementAttrs(soml.TableDataCellElement),
	},
	{
		Doc: "reference/soml/chart.md", Vocab: "<chart>의 속성",
		Pattern: `unknown attribute [a-z-]+ on <chart>[^(]*\(available attributes: ([^)]+)\)`,
		Sep:     ", ", Want: soml.ElementAttrs(soml.ChartElement),
	},
	{
		Doc: "reference/soml/chart.md", Vocab: "<point>의 속성",
		Pattern: `unknown attribute [a-z-]+ on <point>[^(]*\(available attributes: ([^)]+)\)`,
		Sep:     ", ", Want: soml.ElementAttrs(soml.ChartPointElement),
	},
	// 차트 종류는 도형 프리셋과 같은 종류의 닫힌 목록이다 — 진단 문구가 목록을 싣고 문서가
	// 그 문장을 그대로 인용하므로, 종류가 늘면 두 자리가 함께 움직여야 한다.
	{
		Doc: "reference/soml/chart.md", Vocab: "차트 종류",
		Pattern: `available types: ([^)]+)\)`, Sep: ", ", Want: soml.ChartTypes(),
	},
	{
		Doc: "reference/soml/text-subset.md", Vocab: "<li>의 속성",
		Pattern: `unknown attribute [a-z-]+ on <li>[^(]*\(available attributes: ([^)]+)\)`,
		Sep:     ", ", Want: soml.ElementAttrs(soml.ListItemElement),
	},
	{
		Doc: "reference/soml/text-subset.md", Vocab: "<h1>의 속성",
		Pattern: `unknown attribute [a-z-]+ on <h1>[^(]*\(available attributes: ([^)]+)\)`,
		Sep:     ", ", Want: soml.ElementAttrs("h1"),
	},
}

// TestDocsVocabularyListsMatchCode는 등록된 자리마다 문서의 목록과 코드의 목록을 **집합으로**
// 맞댄다. 순서와 표기는 문서의 것이다 (같은 목록이 한 쪽에서는 `, `로, 다른 쪽에서는 `·`로
// 적혀도 둘 다 초록이다).
func TestDocsVocabularyListsMatchCode(t *testing.T) {
	for _, entry := range docVocabularies {
		t.Run(entry.Doc+"/"+entry.Vocab, func(t *testing.T) {
			content := docFile(t, entry.Doc)
			matches := regexp.MustCompile(entry.Pattern).FindAllStringSubmatch(content, -1)
			want := sortedCopy(entry.Want)
			checked := 0
			for _, match := range matches {
				// 생략한 자리(`(available types: …)`)는 사람을 위한 축약이므로 건너뛴다 —
				// 다만 한 문서에 생략만 남으면 아래에서 실패한다.
				if strings.Contains(match[1], "…") {
					continue
				}
				got := docList(match[1], entry.Sep)
				if !equalLists(got, want) {
					t.Errorf("%s의 %s 목록이 코드와 다르다\n문서: %v\n코드: %v\n문서에만: %v\n코드에만: %v",
						entry.Doc, entry.Vocab, got, want, missing(got, want), missing(want, got))
				}
				checked++
			}
			// 패턴이 맞지 않는 것을 통과로 두면 이 게이트가 죽는 가장 조용한 길이 열린다:
			// 문서를 고쳐 써서 패턴이 어긋나면 테스트는 초록인데 아무것도 보지 않는다.
			if checked == 0 {
				t.Fatalf("%s에서 %s 목록을 찾지 못했다 (패턴: %s) — 문서의 문장이 바뀌었으면 docVocabularies의 패턴을 함께 고친다",
					entry.Doc, entry.Vocab, entry.Pattern)
			}
		})
	}
}

// docFile은 문서 하나를 **공백 정규화해서** 돌려준다 — 목록이 줄바꿈으로 접혀 있어도 한 줄로
// 읽힌다 (문서의 줄 폭은 저자의 것이다).
func docFile(t *testing.T, rel string) string {
	t.Helper()
	path := filepath.Join("..", "docs", filepath.FromSlash(rel))
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("문서를 읽을 수 없다 (%s): %v", rel, err)
	}
	return strings.Join(strings.Fields(string(data)), " ")
}

// docList는 문서에 적힌 목록을 이름들로 쪼갠다. 마크다운의 장식(백틱·강조)은 표기이므로 걷어낸다.
func docList(raw, sep string) []string {
	var names []string
	for _, item := range strings.Split(raw, sep) {
		name := strings.Trim(strings.TrimSpace(item), "`*")
		if name != "" {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	return names
}

func docSlideLayouts() []string {
	layouts := soml.SlideLayouts()
	names := make([]string, 0, len(layouts))
	for _, layout := range layouts {
		names = append(names, string(layout))
	}
	return names
}

func docFontRoles() []string {
	names := make([]string, 0, len(requiredFonts))
	for _, role := range requiredFonts {
		names = append(names, string(role))
	}
	return names
}

func docTextRoles() []string {
	roles := soml.TextRoles()
	names := make([]string, 0, len(roles))
	for _, role := range roles {
		names = append(names, string(role))
	}
	return names
}

func sortedCopy(list []string) []string {
	out := append([]string(nil), list...)
	sort.Strings(out)
	return out
}

func equalLists(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// missing은 a에는 있고 b에는 없는 이름들이다 — 실패 메시지가 "어느 이름이 문제인지"까지 적게
// 한다 (목록을 눈으로 대조하게 두지 않는다).
func missing(a, b []string) []string {
	have := make(map[string]bool, len(b))
	for _, name := range b {
		have[name] = true
	}
	var out []string
	for _, name := range a {
		if !have[name] {
			out = append(out, name)
		}
	}
	return out
}
