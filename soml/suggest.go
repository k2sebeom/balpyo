package soml

// Suggest는 닫힌 목록에서 word의 오타 후보를 하나 고른다 — 없으면 ok=false다.
//
// 어휘가 닫혀 있으므로(모르는 이름은 진단이다) 후보를 계산할 수 있다. 실제 오타는 편집 거리 1–2였다:
// `img`→`image` · `two-colum`→`two-column` · `heding`→`heading` · `elipse`→`ellipse`. 그래서 거리는
// 글자 하나의 넣기 · 빼기 · 바꾸기와 **이웃한 두 글자의 자리 바꿈**을 각각 1로 센다 (`haeding`도 1이다).
//
// 후보가 되는 조건은 셋이다:
//   - 거리가 2 이하다 — 네 글자 이하의 낱말은 1 이하다. 짧은 낱말에서 거리 2는 오타가 아니라 다른
//     낱말이다: `fill` → `fit`, `gap` → `at`은 철자가 아니라 개념을 잘못 고른 저자에게 엉뚱한 답이다.
//   - 거리가 word의 길이보다 작다. 그렇지 않으면 `p` → `a`처럼 아무 짧은 이름이나 후보가 된다.
//   - 대소문자만 다르면 거리 0으로 본다 (`<Text>` → `<text>`) — 어휘는 소문자뿐이다.
//
// 거리가 같은 후보가 여럿이면 목록의 앞쪽이 이긴다 — 목록의 순서가 이미 진단 순서(흔한 것이 앞)라서
// 결과가 실행마다 같다.
func Suggest(word string, candidates []string) (string, bool) {
	return suggestWithin(word, candidates, 2)
}

// suggestWithin은 Suggest에 거리 상한을 준다.
func suggestWithin(word string, candidates []string, maxDist int) (string, bool) {
	if word == "" {
		return "", false
	}
	lower := []rune(asciiLower(word))
	if len(lower) <= 4 && maxDist > 1 {
		maxDist = 1
	}
	best, bestDist := "", maxDist+1
	for _, candidate := range candidates {
		if candidate == word {
			// 같은 이름은 후보가 아니다 — 목록에 있는 이름이 진단됐다면 이유가 다른 데 있다.
			continue
		}
		d := editDistance(lower, []rune(asciiLower(candidate)))
		if d < bestDist && d < len(lower) {
			best, bestDist = candidate, d
		}
	}
	return best, best != ""
}

// htmlTagAliases는 오타가 아니라 **HTML 습관**인 태그 이름의 답이다 — 편집 거리로는 닿지 않거나
// (`<b>` → `<strong>`) 엉뚱한 이름에 닿는다 (`<bold>` → `<col>`).
var htmlTagAliases = map[string]string{
	"b":      "strong",
	"bold":   "strong",
	"i":      "em",
	"italic": "em",
	"img":    "image",
	"h4":     "h3",
	"h5":     "h3",
	"h6":     "h3",
}

// SuggestTag는 어휘 밖 태그 이름의 후보다. HTML 습관(htmlTagAliases)이 먼저이고, 다음이 그 자리에 쓸 수
// 있는 태그(거리 2까지), 마지막이 어휘 전체(거리 1까지)다. 자리 밖 후보의 문턱이 더 낮은 이유는
// 그 후보가 자리가 틀린 답이라 확신이 높을 때만 말할 가치가 있기 때문이다.
func SuggestTag(name string, here, all []string) (string, bool) {
	if alias, ok := htmlTagAliases[asciiLower(name)]; ok {
		return alias, true
	}
	if guess, ok := suggestWithin(name, here, 2); ok {
		return guess, true
	}
	return suggestWithin(name, all, 1)
}

// asciiLower는 ASCII 대문자만 내린다 — 한글 이름(테마 · 슬롯이 아닌 저자의 낱말)은 그대로 둔다.
func asciiLower(s string) string {
	b := []byte(s)
	for i, c := range b {
		if 'A' <= c && c <= 'Z' {
			b[i] = c + ('a' - 'A')
		}
	}
	return string(b)
}

// editDistance는 이웃 자리 바꿈을 1로 세는 편집 거리다 (최적 문자열 정렬 거리).
func editDistance(a, b []rune) int {
	prev2 := make([]int, len(b)+1)
	prev := make([]int, len(b)+1)
	cur := make([]int, len(b)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(a); i++ {
		cur[0] = i
		for j := 1; j <= len(b); j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			cur[j] = min(prev[j]+1, cur[j-1]+1, prev[j-1]+cost)
			if i > 1 && j > 1 && a[i-1] == b[j-2] && a[i-2] == b[j-1] {
				cur[j] = min(cur[j], prev2[j-2]+1)
			}
		}
		prev2, prev, cur = prev, cur, prev2
	}
	return prev[len(b)]
}
