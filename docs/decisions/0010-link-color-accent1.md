# 0010 — 링크 색은 hlink 전용 토큰이 아니라 accent1 토큰을 재사용한다

- 상태: 확정 (Iteration 10에서 재검토 예정 — 아래 "결과")
- 날짜: 2026-08-19
- Iteration: 2 (항목 04·06 — NOTES.md 02/04·02/06이 근거 기록이다)
- 관련: PRD §8.4 (토큰 표준 어휘), [외관 속성](../reference/appearance.md)의 인라인 서식 표

## 맥락

`<a>` 런은 색을 가져야 한다. OOXML에는 링크 색의 자리가 따로 있다 — 테마 clrScheme의
`hlink` 슬롯이고, 뷰어는 `hlinkClick`이 붙은 런에 그 색을 입힌다. 웹에는 대응하는 자리가
없어 엔진이 색을 정해야 하고, 두 산출물의 링크 색이 **같은 표의 같은 값**에서 나와야
한다는 계약(토큰 색 보존)과 만난다.

## 선택지

1. **`hlink` 전용 토큰 신설** — 테마 토큰 표에 `hlink`를 더하고 링크 런이 그것을 쓴다.
   OOXML 구조와 1:1이지만, 표준 토큰 어휘(PRD §8.4)에 없는 이름이 생긴다 — 저자의
   `color=` 어휘, "쓸 수 있는 토큰" 진단 목록, CSS 변수 산출(`--balpyo-color-*`)에 전부
   들어가 골든과 문서가 함께 흔들리고, 어휘 확장은 표준화 결정 없이 조용히 일어난다.
2. **`accent1` 재사용** — 링크 델타가 `accent1` 토큰을 쓴다. PPTX 테마 파트는 clrScheme의
   빈 슬롯을 가장 가까운 토큰에서 빌려 채우는 규칙(Iteration 1)에 따라 `hlink` 슬롯을
   이미 `accent1` 값으로 채우고 있다 — 웹이 같은 토큰을 쓰면 두 산출물의 링크 색이 같은
   표의 같은 값에서 나온다.

## 결정

**accent1 재사용(2번).** `core.resolveInlines`의 `<a>` 델타가
`soml.TokenColor("accent1")`을 확정하고, PPTX 런은 뷰어의 자동 색에 기대지 않고
`schemeClr val="accent1"`을 명시한다 (밑줄도 `u="sng"` 명시 — 같은 태도).

## 근거

- 토큰 어휘는 제품의 공개 표면이다 — 전용 토큰 하나가 저자 어휘·진단·CSS 변수·테마
  포맷(Iteration 10의 tokens.json)에 파급된다. 링크 색 하나 때문에 어휘를 열 값이 없다.
- core의 델타(`accent1`)와 PPTX 테마 파트의 hlink 채움(`accent1`에서 차용)이 같은 값이라는
  결합은 **컴파일러가 강제하지 못한다** (core↔render의 임포트 방향 때문에 한 곳에 둘 수
  없다). 대신 산출물 수준 테스트가 고정한다:
  `TestRunHyperlinkColorMatchesThemeHlinkSlot`(render/pptx/runs_test.go)이 런의
  `schemeClr val="accent1"`과 테마 파트 `<a:hlink>`의 RGB가 같은 표에서 나오는 것을
  단정한다 — 이 관례가 깨지는 커밋은 이 테스트가 잡는다.

## 결과

- 링크 런의 색은 저자가 `<span color>`로 감싸 덮을 수 없다 (인라인 델타가 `<a>` 자신에
  적용된다 — `<a>` **안의** `<span color>`는 동작한다).
- **Iteration 10(테마 패키지 포맷)에서 재검토한다**: 테마가 `hlink` 토큰을 명시할 수
  있게 되면 core의 링크 델타도 함께 옮겨야 한다 — 지금은 테마 파트의 차용 규칙과 이
  델타가 짝이고, 한쪽만 바꾸면 위 테스트가 빨간다.
