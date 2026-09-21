package web

import (
	"fmt"
	"html"
)

// 이 파일은 저자의 `alt`이 **접근성 트리의 어디에 붙는가**다 (ADR 0058). 도형과 표가 한 규칙을
// 공유하므로(이름을 **더하고** 내용을 대체하지 않는다) 두 렌더러가 같은 함수를 지난다.

// inkAccessAttrs는 도형 잉크 svg의 접근성 속성이다 — 이름이 없으면 그림은 장식이고
// (`aria-hidden`), 있으면 이름을 가진 그래픽이 된다.
//
// **두 상태는 배타적이다**: `aria-hidden="true"`인 요소의 `aria-label`은 읽히지 않으므로 둘을
// 함께 내면 저자가 쓴 이름이 조용히 사라진다. 그래서 속성을 더하는 것이 아니라 **갈아** 낸다.
//
// 이름을 박스 `<div>`가 아니라 svg가 받는 이유: `aria-hidden`이 이미 "그림은 장식이고 내용은
// 안의 텍스트다"라는 판단을 적어 둔 자리이고, `alt`은 저자가 그 예외를 선언하는 채널이다.
// 박스는 역할이 `generic`이라 ARIA가 이름을 금지하며(무시해도 규격 준수다), 이름을 주려면
// `role="img"`가 함께 필요해져 **안의 글이 AT에서 장식이 된다** — PPTX의 `descr`은 도형 본문을
// 지우지 않으므로 그 길은 같은 SOML을 두 타깃에서 다르게 읽히게 만든다.
func inkAccessAttrs(alt string) string {
	if alt == "" {
		return ` aria-hidden="true"`
	}
	return fmt.Sprintf(` role="img" aria-label="%s"`, html.EscapeString(alt))
}

// labelAttr는 요소에 이름만 더한다 (역할을 바꾸지 않는다) — 표가 쓰는 자리다. 표의 내용은
// 그림이 아니라 격자 안의 글이고 셀은 이미 항해 가능하므로(`<th>`/`<td>`), 이름 위에 `role`을
// 얹으면 그 항해를 지운다.
func labelAttr(alt string) string {
	if alt == "" {
		return ""
	}
	return fmt.Sprintf(` aria-label="%s"`, html.EscapeString(alt))
}
