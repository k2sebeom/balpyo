package geom

// UsedFamilies는 stack(CSS font-family 값)으로 sample을 그렸을 때 브라우저가 쓴 면의 가족
// 이름이다 — 글꼴 탐침의 "있다" 쪽을 기계마다 다른 설치 목록에 기대지 않고 시험하는 입구다
// (제네릭이 고른 면은 언제나 설치된 면이다).
func UsedFamilies(s *Session, stack, sample string) ([]string, error) {
	usages, err := s.platformFonts([]fontSpec{{Stack: stack, Sample: sample}})
	if err != nil {
		return nil, err
	}
	var names []string
	for _, u := range usages[0] {
		if u.GlyphCount > 0 {
			names = append(names, u.FamilyName)
		}
	}
	return names, nil
}
