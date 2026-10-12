package pptx

import "strings"

// defaultRegions는 지역 없이 적은 언어 코드 → OOXML 문화권 이름의 기본 지역이다.
//
// PowerPoint의 맞춤법·글꼴 대체는 `lang`을 문화권(언어-지역)으로 읽는다 — 덱이 `ko`만 적었을 때
// 지역을 비우면 뷰어가 편집기 언어(대개 en-US)로 되돌아가 한글 낱말마다 맞춤법 밑줄이 그어진다.
// 표에 없는 언어는 적은 그대로 싣는다 (언어만 있는 이름도 OOXML에서 유효하다).
var defaultRegions = map[string]string{
	"ar": "SA", "cs": "CZ", "da": "DK", "de": "DE", "el": "GR", "en": "US", "es": "ES",
	"fi": "FI", "fr": "FR", "he": "IL", "hi": "IN", "hu": "HU", "id": "ID", "it": "IT",
	"ja": "JP", "ko": "KR", "nb": "NO", "nl": "NL", "pl": "PL", "pt": "BR", "ro": "RO",
	"ru": "RU", "sv": "SE", "th": "TH", "tr": "TR", "uk": "UA", "vi": "VN", "zh": "CN",
}

// cultureTag는 덱 언어(BCP 47 태그)를 OOXML `lang` 값으로 바꾼다.
//
// 지역이나 문자 체계까지 적은 태그(`ja-JP`, `zh-Hant-TW`)는 대소문자만 관례대로 고쳐 그대로
// 둔다 — 작성자가 고른 지역을 기본 지역으로 덮으면 안 된다. 빈 값은 빈 값이다 (파트가 `lang`을
// 아예 싣지 않는다).
func cultureTag(lang string) string {
	lang = strings.TrimSpace(strings.ReplaceAll(lang, "_", "-"))
	if lang == "" {
		return ""
	}
	subtags := strings.Split(lang, "-")
	subtags[0] = strings.ToLower(subtags[0])
	if len(subtags) == 1 {
		if region, ok := defaultRegions[subtags[0]]; ok {
			return subtags[0] + "-" + region
		}
		return subtags[0]
	}
	for i := 1; i < len(subtags); i++ {
		switch s := subtags[i]; {
		case len(s) == 2:
			subtags[i] = strings.ToUpper(s)
		case len(s) == 4:
			subtags[i] = strings.ToUpper(s[:1]) + strings.ToLower(s[1:])
		default:
			subtags[i] = strings.ToLower(s)
		}
	}
	return strings.Join(subtags, "-")
}
