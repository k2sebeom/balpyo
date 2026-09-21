package pptx

import "fmt"

// 이 파일은 저자의 `alt`이 OOXML의 어디에 붙는가다 (ADR 0058): `p:cNvPr@descr` — PowerPoint의
// "대체 텍스트" 칸이다. 도형·그림·표 프레임 셋이 **같은 요소**를 쓰므로 방출도 한 자리다.

// cNvPrXML은 `p:cNvPr` 한 줄이다 — 이름은 지오메트리 식별자이고(PowerPoint 선택 창과 진단
// 메시지가 같은 이름으로 같은 요소를 가리킨다) `descr`은 저자의 `alt`이다.
//
// **빈 `descr`은 내지 않는다.** OOXML에는 "속성 없음"과 "빈 문자열"의 구별이 없어서
// (`descr=""`는 "대체 텍스트가 비었다"이고 장식이라는 뜻이 아니다 — PowerPoint의 "장식용으로
// 표시"는 확장 요소다) 빈 속성은 뜻을 더하지 않고 재저장 diff의 잡음만 된다. 웹이 `<img alt="">`를
// 언제나 내는 것과 갈라지는 자리이고, 그 비대칭은 두 포맷의 차이이지 두 타깃의 갈라짐이 아니다.
func cNvPrXML(id int, name, descr string) string {
	attrs := fmt.Sprintf(`id="%d" name="%s"`, id, escapeXML(name))
	if descr != "" {
		attrs += fmt.Sprintf(` descr="%s"`, escapeXML(descr))
	}
	return "<p:cNvPr " + attrs + "/>"
}
