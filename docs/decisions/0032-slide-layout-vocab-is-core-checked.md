# 0032 — 슬라이드 수준 어휘는 값과 Span으로 core가 검사한다

- 상태: 확정
- 날짜: 2026-09-02
- Iteration: 6 (항목 02 — NOTES.md 06/02가 근거 기록이다)
- 관련: PRD §7.3, §7.1, [`<slide>`의 `layout`](../reference/soml/slide.md),
  [아키텍처](../internals/architecture.md), ADR 0007 (역할 어휘는 테마가 자란다)

## 맥락

`<slide layout>`이 어휘에 들어온다. 지금 값은 2종(`content`·`canvas`)이고 나머지 9종은
표준 레이아웃 카탈로그와 함께 Iteration 10이다. 값 어휘를 **어느 층이 검사하는가**가 이
결정이다 — 파서는 `<slide>`의 `id`·`skip`을 값까지 본다 (구조 속성이라 어휘가 고정이다).
`layout`도 그렇게 할 것인가?

## 선택지

1. **파서가 값 어휘까지 본다** — `id`·`skip`처럼 `<slide>` 처리에서 `content`/`canvas`를
   검사한다.
2. **파서는 문법만 보고 어휘는 core가 본다** — 파서는 값 없는 `<slide layout>`·빈 값
   `layout=""`만 잡고, 2종 어휘 검사는 값 층으로 넘긴다.

## 결정

**선택지 2.** 파서는 원문 값과 구간을 그대로 나르고(`SlideDoc.Layout`), core가 2종 어휘를
확정한다(`core.slideLayout`). 어휘 밖 값은 `line:col`을 가진 진단이고, 생략된 슬라이드는
`DefaultSlideLayout`(content)으로 채워진다.

## 근거

- **레이아웃 어휘는 테마와 함께 자란다.** 지금은 2종이지만 Iteration 10의 표준 레이아웃
  카탈로그(11종)와 테마 레이아웃이 어휘를 늘린다 — `parse.go`의 기존 주석이 값 어휘를
  파서에서 보지 않는 이유로 이미 적어 둔 것이다 (역할·색 토큰이 그랬듯). `layout`은 그 기준을
  **슬라이드 속성에 처음 적용한** 자리다. `id`·`skip`은 구조 속성이라 어휘가 고정이지만
  `layout`은 그렇지 않다.
- **값과 함께 Span을 싣는 이유는 진단이 `line:col`을 가져야 하기 때문이다.** 덱 모델에는
  파서 트리가 없다 (해석이 끝난 값이다). core의 어휘 진단이 어휘 밖 `layout="titel"`의 값에
  밑줄을 그으려면 값의 소스 구간이 필요하고, 그래서 `Slide`가 `Layout`과 `LayoutSpan`을 함께
  나른다.
- **버린 선택지는 `Slide`에 루트 요소를 싣는 것이다.** 그러면 구간을 따로 나를 필요가 없지만
  덱 모델이 "해석이 끝난 값"이라는 성질이 깨지고, 렌더러가 원문 트리를 다시 읽을 수 있게 되어
  "렌더러는 확정 값만 본다"는 경계가 흐려진다. 값 하나에 구간 하나를 붙이는 편이 싸다.
- **어휘 밖 값은 폴백하지 않고 진단으로 막는다.** `layout="titel"` 오타가 조용히 기본
  레이아웃으로 렌더되면 닫힌 어휘가 막으려던 실패가 된다 — Iteration 10의 "미구현 테마
  레이아웃 → 폴백"과는 다른 규칙이다 (그쪽은 이식성, 이쪽은 오타).

## 결과

- `soml/deck.go`: `SlideLayout`(2종), `DefaultSlideLayout`, `SlideLayouts()`. `Slide.Layout` +
  `Slide.LayoutSpan`. `soml/parse.go`: `fillSlide`가 값 없는·빈 `layout`만 진단한다.
  `core/resolve.go`: `slideLayout`이 어휘를 확정하고 `알 수 없는 layout "titel" (쓸 수 있는
  layout: content, canvas)`를 낸다.
- 렌더러는 문자열이 아니라 `soml.SlideLayout` 타입을 소비한다 — `"canvas"` 리터럴이 세
  렌더러에 흩어지지 않게 한다 ([아키텍처](../internals/architecture.md)의 "슬라이드 수준
  속성이 렌더러에 닿는 경로").
- 픽스처 `010_content_explicit.slide`가 `layout="content"`를 명시하고, 그 산출물이 `layout`
  없는 같은 슬라이드와 바이트가 같은지가 `DefaultSlideLayout == LayoutContent`의 실물
  증거다.
