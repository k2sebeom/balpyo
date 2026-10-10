# 0082 — 슬라이드 전환은 테마의 리듬이고, 슬라이드가 덮어쓸 수 있다

- 상태: 확정
- 날짜: 2026-10-10
- 관련: [ADR 0081](0081-entrance-is-a-native-effect-staggered-by-the-theme.md)(등장 · 테마의 박자),
  [ADR 0062](0062-the-viewer-is-embedded-assets-and-overview-and-print-are-view-states.md), 백로그 Q12

## 맥락

참조 덱은 장을 넘길 때 짧게 페이드하며 미끄러진다. 지금 balpyo의 뷰어는 장을 즉시 바꾸고, PPTX에는
전환이 없다. PowerPoint의 원어는 `p:transition` — 효과 요소 하나(`p:fade` · `p:push` …)와 속도
(`spd`: `fast` 0.5초 · `med` 0.75초 · `slow` 1초)다.

## 결정

1. **어휘는 `fade` · `push` · `none`이다** — 두 타깃에 다 있고 웹에서 같은 그림이 되는 것만 연다.
2. **기본값은 테마의 `motion.transition`**(선택 — 없으면 `none`)이고, `<slide transition="…">`가 그 장에
   들어올 때의 전환을 덮어쓴다. 전환의 길이는 `motion.transitionDuration`(ms, 기본은 등장의 `duration`)이다.
3. **웹은 발표자 뷰에서 다음 · 이전으로 넘길 때만** 움직인다: `fade`는 들어오는 장이 투명에서 나타나고,
   `push`는 들어오는 장이 오른쪽에서 밀고 들어오며 나가는 장이 왼쪽으로 밀려난다 (이전으로 가면 방향이
   반대다). 딥링크 · 오버뷰 · 인쇄 · 측정에는 전환이 없다.
4. **PPTX는 `p:transition`이다** — `p:fade` / `p:push dir="l"`(오른쪽에서 들어온다)이고, 길이는 가장 가까운
   속도 단계로 접힌다(≤ 600ms `fast`, ≤ 875ms `med`, 그보다 길면 `slow`). 정밀한 길이(`p14:dur`)는 확장
   네임스페이스가 필요해 이 결정 밖이다.
5. 패리티 트리에 슬라이드마다 전환 한 줄이 나온다 (`transition fade/fast`).

## 근거

- 전환을 테마에 두는 이유는 등장과 같다: 장마다 다른 전환은 의도가 아니라 사고로 읽힌다. 덮어쓰기는
  표지 · 장 구분처럼 리듬을 바꾸는 장을 위한 것이다.
- 속도를 세 단계로 접는 손실은 작다 — 0.5초와 0.6초는 발표에서 구별되지 않고, 확장 네임스페이스를 여는
  비용(검증 · 대체 콘텐츠)이 그 차이보다 크다.

## 결과

- `soml.Slide.Transition`이 확정 값(`fade` · `push` · 빈 값)을 싣는다. 슬라이드에 `p:transition`이 생긴다
  (전환이 없으면 없다 — 바이트가 그대로다).
