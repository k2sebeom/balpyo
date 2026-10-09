# 0033 — 색 변형은 참조를 보존한다 (lumMod/lumOff/alpha vs color-mix)

- 상태: 확정
- 날짜: 2026-09-02
- Iteration: 8 (그라디언트 + 색 변형 — NOTES.md 08이 근거 기록이다)
- 관련: PRD §7.6, §9.4, §13-5, [외관 속성](../../docs/reference/appearance.md),
  ADR 0018 (opacity는 색 알파다), 토큰 색 보존 계약(soml.Color)

## 맥락

`shade(색, %)`·`tint(색, %)`·`alpha(색, 0~1)`가 어휘에 들어온다. 밑색이 토큰이면
(`shade(accent1, 40%)`) 두 타깃에서 **어떻게** 어둡게/밝게/투명하게 만들 것인가가 이
결정이다. 이 저장소의 핵심 계약은 "토큰 색을 RGB로 눕히지 않는다 — 눕히면 PowerPoint의
테마 색 편집을 따라가지 못한다"이다. 변형은 그 계약을 색을 바꾸면서도 지켜야 한다.

## 선택지

1. **변형을 리터럴 RGB로 계산해 낸다** — `shade(accent1, 40%)` = accent1의 RGB × 0.6을
   계산해 `srgbClr`/`#hex`로 낸다. 두 타깃이 픽셀 단위로 같아진다.
2. **참조를 유지하고 타깃별 변조기를 얹는다** — 웹은 `color-mix()`, PPTX는 색 요소의
   `lumMod`/`lumOff`/`alpha` 자식. 밑색의 `var()`/`schemeClr`가 살아남는다.

## 결정

**선택지 2.** 변형된 토큰 색은 `Token`을 그대로 두고 `Mod`만 얹는다 (soml.Color). 웹은
`color-mix(in srgb, <밑색>, black/white p%)`와 `color-mix(in srgb, transparent, <밑색> a%)`,
PPTX는 `schemeClr` + `a:lumMod val="(100−p)×1000"`(shade) · `+ a:lumOff val="p×1000"`(tint) ·
`a:alpha val="a×100000"`(alpha)다.

## 근거

- **참조 보존이 변형의 목적을 이긴다.** 리터럴로 계산하면 두 타깃이 완벽히 같지만, 사용자가
  PowerPoint에서 테마 accent1을 바꿔도 `shade(accent1, 40%)`로 칠한 도형만 옛 색에 붙박인다 —
  토큰 색을 눕히지 않는 계약이 막으려던 바로 그 실패다. `lumMod`/`lumOff`는 PowerPoint의 테마
  색 변형 UI("Darker 25%")가 쓰는 표기 그 자체라, 변형된 색도 편집 UI에서 같은 자리로 보인다.
- **완벽한 색 일치를 포기한 것은 의식적이다.** `color-mix(in srgb, …)`는 sRGB 채널을 섞고
  `lumMod`/`lumOff`는 광도 공간을 조절해 두 결과가 픽셀 단위로 같지 않다. 07_gradient 슬라이드
  3(shade·tint·alpha 4종)의 실측 차이는 2.0%로, 그 대부분이 CJK/라틴 텍스트 간격·가장자리
  안티앨리어싱 잔차이고 색 블록 자체의 차이는 눈으로 구별되지 않는다 (NOTES 08의 리뷰). 참조
  보존의 값이 그 잔차보다 크다.
- **alpha를 `lumMod` 계열이 아니라 `a:alpha`로 낸 이유**는 투명도가 광도가 아니기 때문이다 —
  `alpha()`는 밑색을 어둡게/밝게 하지 않고 배경이 비치게 한다. 웹도 `color-mix(transparent …)`로
  같은 뜻을 낸다. 도형 불투명도(`opacity`, ADR 0018)와 변형 `alpha()`가 함께 있으면 곱해
  `a:alpha` 하나로 낸다 — 웹의 합성 순서(svg opacity × 색 알파)와 같다.
- **중첩을 막은 이유**(v1)는 매핑을 1:1로 유지하는 것이다: 변형의 밑색은 토큰·리터럴 단색뿐이라
  웹은 `color-mix()` 한 겹, PPTX는 색 자식 한 벌이면 된다. `shade(tint(…))`는 두 겹의 변조기가
  필요해지고 그 조합의 타깃 대응이 v1에 없다.

## 결과

- `soml/appearance.go`: `Color.Mod *ColorMod`, `ColorMod{Kind, Amount}`,
  `ColorModKind`(shade/tint/alpha), `VariantColor`. 변형은 `Token`/`RGB`를 눕히지 않는다.
- `core/resolve.go`: `variantFunc`·`alphaFunc`가 밑색을 `simpleColor`로 받아 중첩을 막는다.
- `render/pptx/shape.go`: `colorTransforms`가 변형·외부 알파를 `lumMod`/`lumOff`/`alpha`
  자식으로 낸다. `render/pptx/units.go`: `LumPct`. `render/web/style.go`: `colorValue`가
  변형을 `color-mix()`로 감싼다.
- 스톱·변형이 `schemeClr`/`var()`를 유지하는지는 구조 테스트가 0 오차로 못 박는다
  (`render/pptx.TestRenderColorVariants`, `render/web.TestColorVariantsCompileToColorMix`).
