# 외관 속성

외관은 **스키마에서 양 타깃으로 순방향 컴파일된다** (PRD §9.4). 저자가 쓴 속성은
`core.Resolve`가 한 번만 해석해 확정하고, 웹과 PPTX 렌더러는 그 확정 값만 읽는다 —
computed style 역추론은 없다. 이 문서는 지금 구현된 외관 속성의 살아 있는 매핑 표다.

## 해석 순서

캐스케이드 4단 중 이번 구현은 두 단이다 (가운데 두 단은 Iteration 2):

```
인라인 속성 > (variant) > (컴포넌트 기본값) > 역할 기본값 · 테마 토큰
```

`<text role="caption" size="20">`은 caption의 기본값(18px, muted, …) 위에 인라인
`size="20"`만 덮어쓴 것이다. 해석이 core에서 한 번만 일어나므로 두 렌더러가 항상 같은
확정 값을 본다.

## 토큰 색은 참조로 남는다 — 제품의 핵심 계약

`color="accent1"` 같은 토큰 색은 해석 후에도 **토큰 이름으로** 남고, 렌더러가 RGB로
눕히지 않는다:

- **웹**: 요소는 `color: var(--balpyo-color-accent1)`을 받고, 토큰의 실제 RGB는 테마 층
  (`@layer balpyo.theme`)의 `:root` 선언 **한 곳**에만 나온다. 변수 하나를 바꾸면 그
  토큰을 쓴 모든 요소가 따라온다.
- **PPTX**: 도형은 `<a:schemeClr val="accent1"/>` 참조를 받고, 실제 RGB는 테마 파트
  (`ppt/theme/theme1.xml`의 clrScheme) 한 곳에만 나온다. 그래서 **내보낸 파일이
  PowerPoint의 테마 색 편집을 따라간다** — 사용자가 PowerPoint에서 강조색을 바꾸면
  덱의 모든 강조색 글자가 함께 바뀐다.

토큰을 어느 지점에서든 리터럴 RGB로 눕히면 이 참조를 복원할 방법이 없다. 그래서 해석된
색(`soml.Color`)은 토큰 참조와 리터럴 두 상태를 구분해 담고, 픽스처 테스트가 슬라이드
파트에 `srgbClr`가 없는지(토큰만 쓴 덱 기준) 감시한다. 폰트 토큰도 같은 구조다 —
`heading`/`body`는 fontScheme의 major/minor 참조(`+mj-lt`/`+mn-lt`)로 나가 테마 폰트
변경을 따라가고, `mono`만 fontScheme에 세 번째 자리가 없어 리터럴 typeface로 나간다.

## 매핑 표

이번 iteration의 속성은 `<text>`의 4종이다. 확정 값 기준의 대응:

| 속성 | 확정 값 | 웹 (CSS) | PPTX (OOXML) |
|---|---|---|---|
| `role` | 역할 이름 | 별도 출력 없음 — 기본값이 아래 값들에 녹는다 | 별도 출력 없음 (동일) |
| `size` | 논리 px 수치 | `font-size: <px>px` | `a:rPr@sz` = px × 0.75 × 100 (1/100 pt) |
| (역할에서) 굵기 | CSS font-weight 수치 | `font-weight: <n>` | `a:rPr@b` — 600 이상이면 `1` |
| (역할에서) 줄높이 | 폰트 크기 배수 | `line-height: <배수>` | `a:lnSpc/a:spcPts@val` = 배수 × px × 0.75 × 100 — 절대값인 이유는 [ADR 0005](../decisions/0005-line-spacing-and-slack.md) |
| (역할에서) 폰트 | 폰트 토큰 이름 | `font-family: var(--balpyo-font-<토큰>)` | `a:latin`/`a:ea` 테마 참조 (`+mj-lt`·`+mj-ea` / `+mn-lt`·`+mn-ea`), mono는 리터럴 typeface |
| `color` (토큰) | 토큰 이름 | `color: var(--balpyo-color-<토큰>)` | `a:solidFill/a:schemeClr@val` = 토큰의 Ref |
| `color` (리터럴) | 대문자 6자리 hex | `color: #RRGGBB` | `a:solidFill/a:srgbClr@val="RRGGBB"` |
| `align` | 논리 이름 | `text-align: start\|center\|end\|justify` | `a:pPr@algn` = `l`\|`ctr`\|`r`\|`just` |

- OOXML `algn`의 `l`/`r`은 물리 이름처럼 보이지만 문단 방향(RTL)에 따라 뜻이 뒤집히는
  **논리 값**이다 — SOML이 `start`/`end`를 쓰는 이유와 정확히 짝이 맞는다.
- 확정 값은 요소의 인라인 `style` 속성으로 나간다 — cascade layer 위에 있으므로 저자
  CSS가 외관을 덮지 못한다 ([ADR 0003](../decisions/0003-inline-style-appearance.md)).

## 내장 테마 `minimal`의 토큰

테마 패키지 포맷은 Iteration 10이다 — 지금은 하드코딩된 최소 표가 두 렌더러의 공급원이다.

색 토큰 (`--balpyo-color-*` ↔ clrScheme 슬롯):

| 토큰 | RGB | clrScheme 슬롯 | schemeClr 참조 |
|---|---|---|---|
| `fg` | `1A1A1A` | `dk1` | `tx1` |
| `bg` | `FFFFFF` | `lt1` | `bg1` |
| `muted` | `6B7280` | `dk2` | `tx2` |
| `accent1` | `0F65F9` | `accent1` | `accent1` |

슬라이드 바탕도 `bg` 토큰에서 나온다 (웹은 `.balpyo-slide { background: var(--balpyo-color-bg) }`,
PPTX는 마스터 배경의 `bg1` 참조).

폰트 토큰 (`--balpyo-font-*` ↔ fontScheme):

| 토큰 | 스택 | fontScheme 슬롯 |
|---|---|---|
| `heading` | Noto Sans, Noto Sans CJK KR, sans-serif | major |
| `body` | Noto Sans, Noto Sans CJK KR, sans-serif | minor |
| `mono` | Noto Sans Mono, Noto Sans CJK KR, monospace | 없음 (리터럴) |

## 값 문법

- 수치는 **단위 없는 논리 스테이지 px** 하나다 (PRD §13-5). `px`·`pt`·`em` 접미사와 `%`는
  에러다.
- 색은 토큰 이름 또는 `#RRGGBB`/`#RGB` 리터럴이다. `#RGB`는 CSS 규칙대로 자리를 두 배로
  늘려 정규화한다. `gradient(…)`·`shade(…)` 같은 색 함수는 문법에 있지만(PRD §7.6)
  Iteration 8에서 열린다 — 지금 쓰면 "아직 쓸 수 없다"는 진단이 나온다.

속성별 에러 조건과 진단 메시지는 [`<text>`](soml/text.md)의 "빌드 에러가 나는 조건"에 있다.
