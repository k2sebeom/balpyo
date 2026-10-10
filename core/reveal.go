package core

import (
	"strconv"

	"github.com/k2sebeom/balpyo/soml"
)

// resolveReveal은 리스트 항목의 **순차 등장 스텝**을 확정한다.
//
// 번호는 슬라이드 안에서 1부터다. 상태(revealTurn)가 resolver에 있는 것이 곧 "스텝은
// 슬라이드의 성질"이라는 뜻이다 — resolver는 슬라이드마다 새로 만들어지므로(Resolve) 다음
// 슬라이드의 첫 스텝은 저절로 1이 된다. 발표는 슬라이드 단위로 진행하고 뒤로 돌아온 장은
// 전부 전개된 상태이므로, 덱 전체를 관통하는 번호는 아무도 쓸 수 없다.
//
// 차례는 `reveal`을 **가진 항목마다** 올라간다 — 번호를 준 항목도 그렇다. 그래서
// `reveal="2"` 뒤의 값 없는 `reveal`은 2가 아니라 자기 자리의 번호를 받는다: 저자가 명시한
// 번호를 셈의 기준으로 삼으면 항목 하나의 번호를 고칠 때 아래 항목 전부의 번호가 따라
// 움직인다. 값이 틀린 항목에서도 차례는 올라간다 — 진단 하나가 나머지 항목의 번호를 바꾸면
// 저자가 고쳐야 할 자리가 둘로 보인다.
func (r *resolver) resolveReveal(item *soml.Element) {
	attr, ok := item.Attr(soml.RevealAttr)
	if !ok {
		return
	}
	r.revealTurn++
	step := r.revealTurn
	if attr.HasValue {
		n, ok := r.revealStep(attr)
		if !ok {
			// 틀린 값은 진단이고 그 항목은 순서를 잃는다 (전개된 상태로 남는다) — 뜻을 정할
			// 수 없는 값에 차례를 짐작해 주면 저자가 진단을 무시하고 넘어간다.
			return
		}
		step = n
	}
	elemAppearance(item).Reveal = step
}

// revealStep은 `reveal`의 값을 해석한다 — 1 이상 정수다.
//
// 값 없음은 여기 오지 않는다 (있음만으로 "자기 차례"라는 뜻이 되는 유일한 외관 밖 속성이다).
// 빈 값은 hasValue를 쓰지 않고 따로 말한다: `reveal=""`을 고치는 길은 "지우면 기본값"이
// 아니라 **값을 지워 자기 차례를 받는 것**이라 일반 문장이 틀린 안내가 된다.
//
// 0·음수를 막는 근거는 cellSpan과 같다: 스텝 0은 "처음부터 보인다"는 뜻이 되어 `reveal`을
// 쓰지 않은 것과 구별되지 않고, 음수는 되감을 자리가 없다. 정수만 받는 이유는 스텝이 세는
// 수라는 것이다 — 1.5번째 등장은 없다.
func (r *resolver) revealStep(attr soml.Attr) (int, bool) {
	if attr.Value == "" {
		r.errorf(attr.ValueSpan, "the %s attribute is empty (remove the value to take the next step in order)", attr.Name)
		return 0, false
	}
	n, err := strconv.Atoi(attr.Value)
	if err != nil || n < 1 {
		r.errorf(attr.ValueSpan, "%s is an integer of at least 1 (%q)", attr.Name, attr.Value)
		return 0, false
	}
	return n, true
}
