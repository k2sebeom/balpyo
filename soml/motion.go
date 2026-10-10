package soml

// 이 파일은 등장(`enter`)의 어휘다.
//
// 등장은 외관이 아니라 **발표 순서**다 (Appearance.Reveal과 같은 쪽) — 캐스케이드 · variant가 없고, 박자
// (길이 · 간격 · 첫 지연)는 테마의 토큰(Theme.Motion)이 덱에 하나로 정한다. 효과는 두 타깃에 다 있는
// 넷으로 닫는다: 웹은 CSS 전환, PPTX는 같은 이름의 기본 등장 효과다.

// EnterAttr는 등장 속성 이름이다 — 콘텐츠 요소와 `<slide>`가 받는다.
const EnterAttr = "enter"

// EnterEffect는 등장 효과다.
type EnterEffect string

const (
	EnterFade EnterEffect = "fade"
	EnterRise EnterEffect = "rise"
	EnterZoom EnterEffect = "zoom"
	EnterWipe EnterEffect = "wipe"
	// EnterNone은 등장 없음이다 — `<slide enter>`가 준 등장을 자식이 끄는 값이다.
	EnterNone EnterEffect = "none"
)

// EnterEffects는 등장 어휘다 (순서 = 진단 · 문서 순서).
func EnterEffects() []string {
	return []string{string(EnterFade), string(EnterRise), string(EnterZoom), string(EnterWipe), string(EnterNone)}
}

// Entrance는 확정된 등장 하나다 — `enter`를 가진 요소에 실린다. 그 요소의 객체 전부(자손 포함 — 더 가까운
// 등장이 없는 것)가 함께 나타난다.
type Entrance struct {
	Effect EnterEffect
	// Order는 슬라이드 안의 등장 순서다 (0부터, 문서 순서).
	Order int
	// Delay는 슬라이드가 시작한 뒤 이 등장이 시작하는 시각(ms)이고, Duration은 길이(ms)다.
	Delay, Duration int
}

// MotionTokens는 테마의 등장 박자다 (tokens.json의 motion) — 단위는 ms다.
type MotionTokens struct {
	// Duration은 등장 하나의 길이, Stagger는 이웃 등장 사이의 간격, Delay는 첫 등장 앞의 기다림이다.
	Duration, Stagger, Delay int
	// Transition은 덱의 기본 슬라이드 전환이다 (빈 값은 없음). TransitionDuration은 그 길이(ms)다.
	Transition         SlideTransition
	TransitionDuration int
}

// TransitionAttr는 슬라이드 전환 속성 이름이다 (`<slide transition>`).
const TransitionAttr = "transition"

// SlideTransition은 슬라이드가 들어올 때의 전환이다. 빈 값은 전환 없음이다.
type SlideTransition string

const (
	TransitionFade SlideTransition = "fade"
	TransitionPush SlideTransition = "push"
	// TransitionNone은 테마의 기본 전환을 그 장에서 끄는 값이다 (확정 값으로는 빈 값이다).
	TransitionNone SlideTransition = "none"
)

// SlideTransitions는 전환 어휘다 (순서 = 진단 · 문서 순서).
func SlideTransitions() []string {
	return []string{string(TransitionFade), string(TransitionPush), string(TransitionNone)}
}

// EnterOf는 요소의 확정 등장이다 (없으면 nil).
func (e *Element) EnterOf() *Entrance {
	if e.Appearance == nil {
		return nil
	}
	return e.Appearance.Enter
}
