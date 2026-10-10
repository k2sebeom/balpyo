# 0088 — 무늬 채움은 도형에만 쓰는 프리셋 빗금이다

- 상태: 확정
- 날짜: 2026-10-10
- 관련: [ADR 0034](0034-gradient-svg-def-and-objectboundingbox.md)(그라디언트는 SVG def),
  [ADR 0084](0084-gradients-take-alpha-stops-and-a-radial-form-and-a-background-takes-an-overlay.md)(스톱의 불투명도),
  [ADR 0064](0064-the-contract-is-component-and-layout-parity-not-pixel-parity.md)(픽셀은 계약 밖), 백로그 Q15

## 맥락

참조 덱 3장은 수요 곡선과 계단식 용량 사이의 "낭비" 영역을 **빗금**으로 칠한다. 채움은 지금 단색 · 그라디언트뿐이다.
OOXML에는 원어가 있다: `a:pattFill prst="…"`는 앞색 · 뒷색 두 개와 프리셋 무늬 이름 하나로 된 채움이고, PowerPoint의
"채우기 → 패턴 채우기"에서 그대로 고칠 수 있다. 웹에는 SVG `<pattern>`이 있다.

무늬 프리셋은 OOXML에 54종이 있다. 대부분(`pct5`…`pct90`, `shingle`, `weave`, `sphere` …)은 8×8 비트맵이라 웹에서
같은 모양을 내려면 비트맵을 옮겨야 하고, 이름만 보고 모양을 떠올리기 어렵다.

## 선택지

1. 열지 않는다 — 빗금은 줄무늬 그라디언트로 흉내 낸다. 그라디언트 스톱은 균등 배치뿐이라 줄무늬가 나오지 않는다.
2. **54종 전부** — 웹 쪽에 비트맵 표가 54개 생기고, 어휘가 쓸모보다 크다.
3. **선 무늬 위주의 닫힌 부분집합** — 두 타깃에서 같은 모양(가로 · 세로 · 사선 · 교차 · 점)으로 읽히는 것만 연다.

## 결정

**선택지 3.**

1. **`pattern(<무늬> <앞색> <뒷색>)`** — 색 함수 자리에 쓰는 채움이다. 두 색은 그라디언트 스톱과 같은 문법이다
   (토큰 · `#리터럴`, `/퍼센트` 불투명도 — `bg/0`이면 뒷색이 비친다).
2. **무늬 어휘 9종** (SOML 이름 → OOXML `prst`):
   `horizontal`(`horz`) · `vertical`(`vert`) · `diagonal-up`(`upDiag`) · `diagonal-down`(`dnDiag`) ·
   `wide-diagonal-up`(`wdUpDiag`) · `wide-diagonal-down`(`wdDnDiag`) · `cross`(`cross`) ·
   `diagonal-cross`(`diagCross`) · `dots`(`pct20`).
3. **도형 채움에만 쓴다** — `<shape fill>`과 테마의 도형 기본값 · variant. 글자 색 · 테두리 · 연결선 · 표 칸 · 배경은
   진단이다. 글자와 선에 빗금을 칠하는 일은 참조 덱에 없고, 배경은 웹에서 SVG 무늬가 아니라 CSS 그림 층이라 길이
   하나 더 생긴다 — 쓸 자리가 생기면 연다.
4. **웹은 SVG `<pattern>`**(`patternUnits="userSpaceOnUse"`, 8×8 논리 px 타일 — 뒷색 사각형 위에 앞색 선 · 점)이다.
   **PPTX는 `a:pattFill`**이고 `fgClr` · `bgClr`가 토큰이면 `schemeClr`로 남는다. 도형 불투명도는 두 색에 `a:alpha`로
   나눠 싣는다 (그라디언트 스톱과 같은 방식).
5. 패리티 트리의 채움은 `pattern(<무늬>,<앞색>,<뒷색>)`이다.

## 근거

- 무늬는 선 굵기와 간격이 뜻이 아니라 질감이다 — PowerPoint는 무늬를 화면 픽셀 단위로 그려 확대해도 촘촘함이 그대로이고,
  웹은 무대와 함께 커진다. 이 차이는 ADR 0064의 "픽셀은 계약 밖"에 든다. 계약은 같은 종류의 채움, 같은 무늬 이름, 같은
  두 색이다.
- 퍼센트 무늬(`pct*`) 중 `pct20` 하나만 `dots`로 연다 — 점 무늬로 읽히는 대표 하나면 충분하고, 나머지는 비트맵 농도
  차이라 이름이 모양을 말하지 않는다.

## 결과

- `soml.Fill`에 `Pattern`이 생긴다 (단색 · 그라디언트와 나란한 셋째 갈래). 무늬를 모르는 렌더러 자리는 에러를 낸다.
- OnlyOffice와 PowerPoint의 무늬 밀도는 웹과 다르다 — 알려진 차이로 문서에 적는다.
