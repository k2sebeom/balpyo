# 외관 속성

외관은 **스키마에서 양 타깃으로 순방향 컴파일된다** (PRD §9.4). 저자가 쓴 속성은
`core.Resolve`가 한 번만 해석해 확정하고, 웹과 PPTX 렌더러는 그 확정 값만 읽는다 —
computed style 역추론은 없다. 이 문서는 지금 구현된 외관 속성의 살아 있는 매핑 표다.

## 배치는 외관이 아니다

SOML에는 선언 채널이 **둘**이다 (PRD §5.3): 이 페이지가 담는 **외관 채널**(색·폰트·크기·
정렬)과 [레이아웃 태그](soml/layout.md)가 담는 **배치 채널**(흐름·간격·크기·자리). 한
요소가 두 채널을 함께 쓰는 것은 정상이다 — `<p grow="1" size="30">`은 배치와 외관을 동시에
적은 것이고, 두 채널이 서로를 모르는 것이 이 언어의 경계다.

경계가 지켜지는 자리 세 곳:

- **`variant`는 배치를 담지 않는다.** variant는 외관 번들이므로 그 안에 `gap`이나 `grow`가
  들어오지 않는다 — 테마가 배치를 소유하는 자리는 1층 레이아웃(표준 레이아웃 카탈로그,
  Iteration 10)이다. 그래서 배치 속성에는 **캐스케이드가 없다**: 저자가 요소에 적은 값이
  곧 확정 값이다.
- **`style` 탈출구는 배치 전용이다.** 외관 선언(색·테두리)을 쓰면 웹 전용이 되고 경고가
  붙는다 ([레이아웃 태그](soml/layout.md)의 `style` 절,
  [ADR 0014](../decisions/0014-style-escape-hatch-boundary.md)).
- **배치는 좌표로 구워지고 외관은 참조로 남는다.** 색·폰트 토큰은 PPTX 테마 파트를 참조해
  편집을 따라가지만, 배치는 지오메트리 패스가 잰 절대 좌표가 되어 산출물에 남는다 (아래
  공간 토큰 절).

## 캐스케이드 — 해석 순서

외관은 4단 캐스케이드로 확정된다 (PRD §8.3):

```
인라인 속성 > variant > 컴포넌트 기본값 > 역할 기본값 · 테마 토큰
```

- **역할 기본값**이 완전한 외관을 바닥에 깔고, 위 세 단은 **부분 패치**로 낮은 단부터
  얹힌다 — 단이 정하지 않은 필드는 아래 단의 값이 남는다. `<text role="caption"
  size="20">`은 caption의 기본값(18px, muted, …) 위에 인라인 `size="20"`만 덮어쓴 것이다.
- **컴포넌트 기본값 단은 태그별**이다: `<h3>`은 크기 28, `<blockquote>`는 색 `muted`,
  나머지 태그와 `<text>`는 빈 패치다 — 그래서 `<p>` ≡ `<text role="body">` 등가가 깊은
  비교로 성립한다 ([텍스트 서브셋](soml/text-subset.md)의 등가 표,
  [ADR 0007](../decisions/0007-block-tag-role-ownership.md)).
- **병합은 파싱·해석 시점에 `core.Resolve`에서 한 번 일어나고 타깃 무관이다** — 웹과
  PPTX 렌더러는 확정 값만 읽으므로 캐스케이드 규칙이 렌더러마다 갈라질 수 없다. 인라인
  태그(strong/em …)의 런 스타일은 이 캐스케이드의 재적용이 아니라 블록 확정 스타일의
  상속-복사다 ([ADR 0008](../decisions/0008-run-style-inherit-copy.md)).

### `variant` — 테마가 이름 붙인 외관 번들

`variant` 속성은 `<text>`와 블록 태그(`h1`~`h3`, `p`, `blockquote`, `pre`), 그리고
[`<shape>`](soml/shape.md)·[`<image>`](soml/image.md)가 받는다. **어휘는 테마 소유이고
표준화하지 않는다** (PRD §13-6).

**어휘가 세 벌이다** — 텍스트·도형·이미지 variant는 이름 공간이 다르고 서로를 모른다. 표를
합치면 `<image variant="card">`가 도형 번들을 받아 `fill`을 정하려 들고(이미지에 없는 채널이다)
그 값이 조용히 버려진다. 내장 `minimal` 테마:

| 요소 | variant | 담는 것 |
|---|---|---|
| 텍스트 | `kicker` | 크기 18 + 굵기 700 + `accent1` |
| 텍스트 | `lead` | 크기 32 |
| 도형 | `card` | `fill: bg` · `radius: lg` |
| 도형 | `card-header` | `border: none` · `radius: sm` (채움은 정하지 않는 **부분 패치**) |
| 도형 | `card-body` | `fill: bg` |
| 도형 | `card-backdrop` | `fill: none` · `border: 3 solid accent1` · `radius: lg` |
| 도형 | `stat-card` | `border: none` · `shadow: soft` · `radius: lg` |
| 도형 | `badge` | `border: 1 solid accent1` · `radius: lg` |
| 이미지 | `framed` | `border: 1 solid muted` · `radius: md` · `shadow: soft` |

**이미지 어휘에 이름이 하나뿐인 것은 그 하나가 결정의 실물이기 때문이다.** PRD §7.5가 적었던
`frame` 속성을 열지 않고 액자를 번들로 옮긴 것이 Iteration 5의 결정이고
([ADR 0027](../decisions/0027-open-alt-close-frame.md)), 액자가 뜻하는 세 조각을 **함께** 정하는
것이 번들의 뜻이다 — 저자가 `border`만 쓰면 액자가 아니라 선 하나다.

**도형 variant는 자식 텍스트의 외관을 담지 않는다.** `variant="card-header"`가 그 안의
글자를 굵게 만들지 않는다 — 담게 하면 캐스케이드가 트리를 타고 내려가는 **상속**이 되고,
상속은 한 요소 안에서 끝나는 지금의 병합과 다른 두 번째 메커니즘이다
([ADR 0022](../decisions/0022-shape-variants-do-not-inherit.md)). 카드 제목의 타이포그래피는
자식에 직접 적는다 (`<h3>` 또는 `<text role="heading">`).

**테마에 없는 variant는 에러가 아니라 경고 + 기본 외관 폴백이다** (PRD §13-6) — variant
어휘는 테마의 사유물이라, 다른 테마로 만든 콘텐츠가 이 테마에서 빌드되지 않으면 콘텐츠의
테마 간 이식성이 깨진다. 엄격함이 필요한 곳(CI 등)은 `--strict`가 경고를 에러로 승격한다
([CLI](cli.md)):

```
slides/006_variant.slide:6:16: warning: variant "hero"가 테마에 없다 — 기본 외관으로 계속한다 (이 테마의 variant: kicker, lead)
```

문구는 텍스트와 도형이 **같고 나열된 목록만 다르다** — 어느 어휘인지는 목록이 말한다.

`testdata/fixtures/02_text_subset/slides/006_variant.slide`:

```html
<slide id="variant">
  <h2 variant="hero">모르는 variant는 기본 외관으로 폴백한다</h2>
  <text role="body" variant="kicker">KICKER — 작고 굵은 강조색</text>
  <p variant="lead">lead — 본문보다 큰 도입 문단이다.</p>
</slide>
```

값이 없거나 빈 `variant`(`<p variant>`·`variant=""`)는 다른 외관 속성과 같이 에러다 —
"값 없음"은 폴백할 뜻조차 정할 수 없는 형태다.

### 컴포넌트 기본값 — 도형 캐스케이드의 바닥

텍스트는 `role`이 캐스케이드의 바닥을 깔지만 **도형에는 역할이 없다.** 그 자리를 테마의
컴포넌트 기본값이 대신한다 (PRD §8.3):

| 요소 | 기본값 |
|---|---|
| `<shape>` | `fill: surface` · `border: 1 solid muted` · `radius: md` |
| `<image>` | **없음** (빈 패치) |

그래서 `<shape type="rect"/>`만 써도 채움과 테두리가 확정된다. 값이 비어 남으면 채우는 판단이
렌더러로 내려가고, 두 타깃이 서로 다른 값을 채우는 순간 웹과 PPTX가 갈라진다.

- **기본값의 `radius: md`는 `rounded-rect`가 아닌 프리셋에서 조용히 버려진다.** 저자가 인라인
  으로 적은 `radius`는 에러이지만(고칠 수 있으므로 진단이 고칠 방법을 말한다) 테마 기본값에
  같은 진단을 내면 `<shape type="rect"/>`가 무조건 에러가 된다
  ([ADR 0019](../decisions/0019-radius-is-rounded-rect-only.md)). **그 제약은 도형의 것이다** —
  그림에서 `radius`는 언제나 유효하다 (그림은 프리셋이 아니라 자기 클립을 갖는다, 아래 이미지
  외관 표).
- **이미지 행이 비어 있는 것은 도형과 정면으로 갈라지는 결정이다.** "값이 비면 렌더러가 채우는
  판단을 한다"는 위의 근거가 그림에는 걸리지 않는다 — 채움 없는 도형은 보이지 않지만 그림은
  자기 픽셀을 갖고, 두 타깃의 암묵 기본값도 갈라지지 않는다 (웹 `<img>`와 PPTX `pic`은 둘 다
  장식 없이 그린다). 기본으로 테두리를 두르면 스크린샷·도표에 이중 경계가 생기고 저자가 모든
  그림에 `border="none"`을 적어야 한다. 액자는 `variant="framed"`가 소유한다.
- **`fit`은 이 단에 없다.** 기본값 `contain`은 **언어**가 갖는다 (`soml.DefaultImageFit`) —
  `fit`은 외관 번들이 아니라 그림의 의미이고, 테마를 바꿨을 때 얼굴이 잘리기 시작하면 안 된다.
  `<p>` ≡ `<text role="body">` 등가가 테마가 아니라 언어에 있는 것과 같은 자리의 결정이다.

## 텍스트 역할 5종

역할은 캐스케이드의 바닥이다 — 크기·굵기·줄높이·색·폰트 기본값의 묶음. 값의 출처는
`core/theme.go`의 하드코딩 표다 (테마 패키지 포맷은 Iteration 10):

| 역할 | 크기(px) | 굵기 | 줄높이 | 색 | 폰트 토큰 | 대응 블록 태그 |
|---|---|---|---|---|---|---|
| `title` | 56 | 700 | 1.2 | `fg` | `heading` | `<h1>` |
| `heading` | 36 | 700 | 1.3 | `fg` | `heading` | `<h2>` `<h3>` |
| `body` | 24 | 400 | 1.5 | `fg` | `body` | `<p>` `<blockquote>` `<li>` |
| `caption` | 18 | 400 | 1.4 | `muted` | `body` | 없음 — `<text role="caption">`만 |
| `code` | 20 | 400 | 1.5 | `fg` | `mono` | `<pre>` |

수치의 근거는 수직 리듬이다: 본문 24px × 줄높이 1.5 = 36px이고, 스테이지 높이 720px은
정확히 본문 20줄이다.

## 토큰 색은 참조로 남는다 — 제품의 핵심 계약

`color="accent1"`·`fill="surface"` 같은 토큰 색은 해석 후에도 **토큰 이름으로** 남고,
렌더러가 RGB로 눕히지 않는다:

- **웹**: 요소는 `color: var(--balpyo-color-accent1)`을 받고, 토큰의 실제 RGB는 테마 층
  (`@layer balpyo.theme`)의 `:root` 선언 **한 곳**에만 나온다. 변수 하나를 바꾸면 그
  토큰을 쓴 모든 요소가 따라온다.
- **PPTX**: 도형은 `<a:schemeClr val="accent1"/>` 참조를 받고, 실제 RGB는 테마 파트
  (`ppt/theme/theme1.xml`의 clrScheme) 한 곳에만 나온다. 그래서 **내보낸 파일이
  PowerPoint의 테마 색 편집을 따라간다** — 사용자가 PowerPoint에서 강조색을 바꾸면
  덱의 모든 강조색 글자가 함께 바뀐다.

**리터럴 색은 반대다.** `fill="#0F65F9"`는 산출 PPTX에서 `<a:srgbClr val="0F65F9"/>`가 되고,
그것이 **정상이다** — 리터럴은 테마 참조가 아니라는 사실이 값의 형태에 그대로 보여야 한다.
차이가 나타나는 자리는 하나뿐이다: **PowerPoint에서 디자인 탭의 테마 색을 바꾸면 토큰으로
칠한 도형만 따라오고 리터럴로 칠한 도형은 그대로 남는다.** 같은 RGB를 토큰으로 쓴 도형과
리터럴로 쓴 도형이 그 순간 갈라지는 것이 **이 제품이 토큰을 권하는 공식적인 이유**다.

토큰을 어느 지점에서든 리터럴 RGB로 눕히면 이 참조를 복원할 방법이 없다. 그래서 해석된
색(`soml.Color`)은 토큰 참조와 리터럴 두 상태를 구분해 담고, 픽스처 테스트가 슬라이드 파트의
`srgbClr` 값을 **슬라이드별 리터럴 색 전수 표와 정확히 맞댄다** — 표에 없는 `srgbClr`는
토큰이 어딘가에서 눕은 것이다 ([테스트 환경](../internals/testing.md)). 폰트 토큰도 같은
구조다 — `heading`/`body`는 fontScheme의 major/minor 참조(`+mj-lt`/`+mn-lt`)로 나가 테마 폰트
변경을 따라가고, `mono`만 fontScheme에 세 번째 자리가 없어 리터럴 typeface로 나간다.

자동 검사가 볼 수 있는 것은 "XML에 참조가 남았다"까지다. **실제로 따라오는지는 사람이
PowerPoint에서 확인한다** — 각 iteration의 리뷰 단계가 그 자리다.

## 매핑 표

텍스트 외관 속성의 확정 값 기준 대응:

| 속성 | 확정 값 | 웹 (CSS) | PPTX (OOXML) |
|---|---|---|---|
| `role` | 역할 이름 | 별도 출력 없음 — 기본값이 아래 값들에 녹는다 | 별도 출력 없음 (동일) |
| `variant` | 테마 패치 | 별도 출력 없음 — 패치가 확정 값에 녹는다 | 별도 출력 없음 (동일) |
| `size` | 논리 px 수치 | `font-size: <px>px` | `a:rPr@sz` = px × 0.75 × 100 (1/100 pt) |
| (역할에서) 굵기 | CSS font-weight 수치 | `font-weight: <n>` | `a:rPr@b` — 600 이상이면 `1` |
| (역할에서) 줄높이 | 폰트 크기 배수 | `line-height: <배수>` | `a:lnSpc/a:spcPts@val` = 배수 × px × 0.75 × 100 — 절대값인 이유는 [ADR 0005](../decisions/0005-line-spacing-and-slack.md) |
| (역할에서) 폰트 | 폰트 토큰 이름 | `font-family: var(--balpyo-font-<토큰>)` | `a:latin`/`a:ea` 테마 참조 (`+mj-lt`·`+mj-ea` / `+mn-lt`·`+mn-ea`), mono는 리터럴 typeface |
| `color` (토큰) | 토큰 이름 | `color: var(--balpyo-color-<토큰>)` | `a:solidFill/a:schemeClr@val` = 토큰의 Ref |
| `color` (리터럴) | 대문자 6자리 hex | `color: #RRGGBB` | `a:solidFill/a:srgbClr@val="RRGGBB"` |
| `align` | 논리 이름 | `text-align: start\|center\|end\|justify` | `a:pPr@algn` = `l`\|`ctr`\|`r`\|`just` |
| `anchor` | `top` `center` `bottom` | (부모 도형의) `justify-content: start\|center\|end` | (부모 도형의) `a:bodyPr@anchor` = `t`\|`ctr`\|`b` |

- OOXML `algn`의 `l`/`r`은 물리 이름처럼 보이지만 문단 방향(RTL)에 따라 뜻이 뒤집히는
  **논리 값**이다 — SOML이 `start`/`end`를 쓰는 이유와 정확히 짝이 맞는다. `anchor`가
  반대로 **물리 이름**(`top`/`bottom`)인 것도 같은 기준이다: 세로축은 문단 방향으로 뒤집히지
  않으므로 논리 이름을 쓸 근거가 없다.
- **`anchor`는 뜻이 보이는 자리가 [도형 안](soml/shape.md)뿐이다.** 확정 값은 텍스트가 갖지만
  선언이 나가는 자리는 **부모 도형**이다 (웹은 도형 div의 flex 정렬, PPTX는 도형 `bodyPr`) —
  둘 다 박스에 하나뿐이라 도형의 앵커는 **첫 자식이 정한다.** 슬라이드 직속 텍스트 박스에서는
  선언이 아예 나가지 않는다: 잰 rect가 곧 내용 크기라 세로로 움직일 여백이 없다.
- 확정 값은 요소의 인라인 `style` 속성으로 나간다 — cascade layer 위에 있으므로 저자
  CSS가 외관을 덮지 못한다 ([ADR 0003](../decisions/0003-inline-style-appearance.md)).

### 도형 외관 (단색)

[`<shape>`](soml/shape.md)의 외관 8종이다. **정하지 않은 속성은 선언을 하나도 내지 않는다** —
`shadow`·`opacity`·`rotate`·`flip`의 미지정은 두 타깃의 암묵 기본값이 같아 렌더러가 채울
판단이 없다.

| 속성 | 확정 값 | 웹 (생성 CSS) | PPTX (OOXML) |
|---|---|---|---|
| `fill` (토큰) | 토큰 이름 | 그림 svg의 `fill: var(--balpyo-color-<토큰>)` | `a:solidFill/a:schemeClr@val` |
| `fill` (리터럴) | hex | `fill: #RRGGBB` | `a:solidFill/a:srgbClr@val` |
| `fill="none"` | `none` 상태 | `fill: none` | `a:noFill` |
| `border` (굵기) | 논리 px | `stroke-width: <px>px` | `a:ln@w` = px × 9525 (EMU) |
| `border` (색) | 색 | `stroke: var(…)` 또는 `#RRGGBB` | `a:ln/a:solidFill/…Clr` |
| `border` (스타일) | `solid` `dashed` `dotted` | `stroke-dasharray` — 4:3 / 1:3 (굵기 배수) | `a:prstDash@val` = `solid`·**`dash`**·**`dot`** |
| `border="none"` | `none` 상태 | `stroke: none` | `a:ln/a:noFill` |
| `radius` | 토큰 이름 + px | SVG `<rect rx="<px>">` (기하 속성) | `prstGeom/avLst`의 `adj` = 100000 × radius ÷ min(w,h) |
| `shadow` | 토큰 항목 그대로 | `filter: drop-shadow(var(--balpyo-shadow-<토큰>))` | `a:effectLst/a:outerShdw` (dist·dir·blurRad + 색 + `rotWithShape="1"`) |
| `opacity` | 0~1 실수 | 그림 svg의 `opacity` | 채움·선·그림자 **색 안의** `a:alpha@val` (100000분율) |
| `rotate` | 각도 | 박스 div의 `transform: rotate(<deg>deg)` | `a:xfrm@rot` = 각도 × 60000 (한 바퀴로 정규화) |
| `flip` | `h` `v` `hv` | 그림 svg의 `transform: scaleX(-1)` 등 | `a:xfrm@flipH`/`@flipV` |

- **선언이 박스(div)와 그림(svg)으로 갈라지는 기준은 OOXML의 자리다**: `rotate`는
  `xfrm@rot`이 텍스트까지 함께 돌리므로 박스, `flip`은 `flipH`가 글자를 뒤집지 않으므로
  그림, `opacity`는 알파가 `solidFill` 안에 있으므로 그림이다.
- **`border` 스타일 이름은 CSS 쪽이고 파선 수치가 두 타깃을 잇는다.** 어휘가
  `dashed`/`dotted`인 것은 축약 자체가 CSS `border`의 표기이기 때문이고, OOXML로는
  `dash`/`dot`으로 간다 — CSS `border-style: dashed`의 패턴은 UA가 정하므로 이름을 옮길 수
  없고 **수치(`prstDash`의 비율 4:3·1:3)를 웹이 `stroke-dasharray`로 내는 것**이 두 타깃이
  같은 간격을 갖는 유일한 길이다. 어휘를 셋으로 닫은 기준은 `prstDash`에 대응이 있는 것만이다
  (`double`/`groove`는 PPTX에서 조용히 실선이 된다).
- **`radius`는 웹에서도 확정 수치로 나간다** (토큰 변수 참조가 아니다). SVG 기하 속성(`rx`)은
  커스텀 프로퍼티를 이식성 있게 참조할 수 없고, 무엇보다 **OOXML에도 반경 테마 개념이 없어**
  두 타깃이 같은 방식으로 값을 굽는다 (아래 토큰 표).
- **`shadow`의 색만 참조로 남는다**: 웹은
  `color-mix(in srgb, var(--balpyo-color-fg) 15%, transparent)`, PPTX는 `outerShdw` 안의
  `schemeClr` + `alpha`다. rgba로 눕히면 그 도형의 그림자만 옛 색으로 남는다.
- **`opacity`는 합성 불투명도가 아니다.** 반투명 도형 안의 글자는 또렷하다 — 웹은 선언이 그림
  svg에 걸려 자식 텍스트를 건드리지 않고, PPTX는 알파를 색 안에만 싣는다
  ([ADR 0018](../decisions/0018-opacity-compiles-to-color-alpha.md)). 그림자 알파는 도형
  불투명도와 **곱한다** (웹에서 그림자가 그 svg의 필터 결과이므로 함께 흐려진다).

### 이미지 외관

[`<image>`](soml/image.md)의 외관 7종이다. **값 문법은 도형과 같은 함수가 해석하고 갈라지는 것은
값을 싣는 자리뿐이다** — 도형은 SVG 프로퍼티(`stroke`·`rx`)에, 그림은 CSS
(`border-*`·`border-radius`)에 건다. 그림에는 `fill`이 없다 (아래).

| 속성 | 확정 값 | 웹 (`<img>`의 CSS) | PPTX (`p:pic`) |
|---|---|---|---|
| `fit="cover"` | `cover` | `width: 100%` · `height: 100%` · `object-fit: cover` | `a:blipFill/a:srcRect` 크롭 (넘치는 축을 양쪽 절반씩) |
| `fit="contain"` | `contain` | `max-width: 100%` · `max-height: 100%` | 크롭 없음 — 잰 rect가 이미 맞춰진 그림이다 |
| `alt` | 문자열 (생략 ≡ `""`) | `alt` 속성 — **언제나 낸다** | `p:cNvPr@descr` — **값이 있을 때만** 낸다 |
| `border` (굵기) | 논리 px | `border-width: <px>px` | `a:ln@w` = px × 9525 (EMU) |
| `border` (색) | 색 | `border-color: var(…)` 또는 `#RRGGBB` | `a:ln/a:solidFill/…Clr` |
| `border` (스타일) | `solid` `dashed` `dotted` | `border-style: <이름>` — **간격은 UA가 정한다** | `a:prstDash@val` = `solid`·`dash`·`dot` |
| `border="none"` · 미지정 | `none` 상태 · nil | 선언 없음 | `a:ln/a:noFill` (둘이 같은 산출이다) |
| `radius` | 토큰 이름 + px | `border-radius: <px>px` | `prstGeom prst="roundRect"` + `avLst`의 `adj` (도형과 **같은 명세**) |
| `shadow` | 토큰 항목 그대로 | `filter: drop-shadow(var(--balpyo-shadow-<토큰>))` | `a:effectLst/a:outerShdw` |
| `opacity` | 0~1 실수 | `opacity` | **`a:blip/a:alphaModFix@amt`** + 선·그림자 색의 `a:alpha` |
| `rotate` | 각도 | 박스가 아니라 `<img>`의 `transform: rotate(…)` | `a:xfrm@rot` = 각도 × 60000 |
| `flip` | `h` `v` `hv` | 같은 `transform`의 `scale(…)` | `a:xfrm@flipH`/`@flipV` |

- **`opacity`가 그림에서는 `alphaModFix`다.** 도형에서 그 효과를 버린 근거는 그것이 도형 안
  **글자까지** 흐리게 한다는 것이었고, 그림에는 담을 글자가 없다 — 그래서 여기서는 CSS
  `opacity`와 정확한 대응이고, [ADR 0018](../decisions/0018-opacity-compiles-to-color-alpha.md)의
  "색 알파" 규칙이 색 없는 그림에서 가는 자리가 이것이다. 테두리·그림자 색에도 같은 알파가
  실린다 (CSS `opacity`는 테두리를 포함한 요소 전체를 흐리게 하고 `drop-shadow`의 결과도 함께
  흐려진다). 그림자 알파는 도형과 같이 토큰 알파와 **곱한다**.
- **`radius`에는 프리셋 제약이 없다** (도형과 갈라지는 자리다 — 위 컴포넌트 기본값 절). 값
  문법과 조정값 명세는 도형의 `rounded-rect`와 같은 곳에서 나온다: 같은 `radius="md"`가 도형과
  그림에서 다른 모서리로 그려지면 저자는 그 차이를 둘을 나란히 놓고서야 본다.
- **테두리의 위치가 두 타깃에서 다르다.** CSS 테두리는 박스 안쪽(`box-sizing: border-box`),
  OOXML `a:ln`은 경계 중앙이라 PPTX의 그림이 굵기만큼 크게 그려진다 — 보정하지 않는 이유는 잰
  rect가 그대로 `xfrm`이 된다는 계약이고, 1px에서 그 대가는 그림 면적의 4.9%다
  ([테스트 환경](../internals/testing.md)). 도형에는 이 문제가 없었다 (SVG `stroke`도 경로
  중앙이라 양쪽이 같은 규약이었다).
- **미지정 테두리와 `border="none"`이 같은 산출인 것도 도형과 다르다.** 도형은 컴포넌트
  기본값이 테두리를 채우므로 "해석되지 않았다"가 버그이지만, 이미지 기본값은 비어 있어 nil이
  정상 상태다 — 둘 다 "선 없음"이라는 같은 그림이므로 산출도 같아야 한다.
- **`fill`은 없다.** PRD §7.6 표의 `fill` 적용 대상에 image가 없고, 그림 뒤의 칠이 보이는 자리는
  `contain`의 여백뿐인데 그 여백은 PPTX에서 `pic`의 박스 밖이다 — 받아 놓고 한 타깃에서만 그리는
  속성이 된다.
- **회전과 대칭이 하나의 `transform`을 나눠 쓴다** (도형은 회전을 박스에, 대칭을 그림 svg에
  걸어 선언이 둘이다). 그림은 요소가 하나뿐이라 `transform`의 마지막 선언만 남으므로 커스텀
  프로퍼티 둘로 합성한다: `rotate(…) scale(…)` — 오른쪽이 먼저 적용되므로 "대칭 후 회전"이고
  OOXML `xfrm`(뒤집힌 기하를 `rot`으로 돌린다)과 같은 순서다.

### 인라인 서식 (텍스트 런)

인라인 태그의 확정 런 스타일은 블록 스타일의 상속-복사에 태그별 델타를 얹은 것이다
([ADR 0008](../decisions/0008-run-style-inherit-copy.md)). 웹은 런의 CSS를 **완전
명시**하고 (italic이 아니어도 `font-style: normal`), PPTX 인라인 런의 `a:rPr`도
`i`/`u`까지 전부 명시한다 — 양쪽 다 뷰어/UA 기본값에 기대는 값을 남기지 않는다.

| 태그 | 델타 | 웹 (CSS/태그) | PPTX (OOXML) |
|---|---|---|---|
| `strong` | 굵기 max(부모, 700) | `font-weight: 700` | `a:rPr@b="1"` |
| `em` | 이탤릭 | `font-style: italic` | `a:rPr@i="1"` |
| `code` | mono 폰트 (크기는 부모) | `font-family: var(--balpyo-font-mono)` | `a:latin`/`a:ea` 리터럴 typeface (mono는 fontScheme 슬롯이 없다) |
| `a` | 밑줄 + `accent1` 색 + 링크 | `text-decoration: underline; color: var(--balpyo-color-accent1)` + `<a href>` | `a:rPr@u="sng"` + `schemeClr accent1` + `a:hlinkClick r:id` + 슬라이드 rels의 External 관계 |
| `span` | 자기 `color`/`weight`만 | 해당 선언 | 해당 rPr 속성/채움 |
| `br` | 없음 (void) | `<br>` | `a:br` — 문단을 쪼개지 않는다 |

링크 색이 전용 토큰이 아니라 `accent1`인 이유는
[ADR 0010](../decisions/0010-link-color-accent1.md)이다 — 런의 `schemeClr` 참조와 테마
파트의 `hlink` 슬롯이 같은 토큰 표의 같은 값에서 나온다.

### 리스트

들여쓰기·마커 값은 **웹 리스트 CSS 상수(`render/web`)와 한 곳에서 나온다** — PPTX의
`marL`/`indent`가 같은 상수의 EMU 환산(1px = 9525 EMU)이라 두 타깃의 들여쓰기가 갈라질
수 없다. 값 36px은 PowerPoint 1수준 기본(342900 EMU)과 같다.

| 항목 | 값 | 웹 (CSS) | PPTX (OOXML) |
|---|---|---|---|
| 텍스트 들여쓰기 | 36px (`web.ListTextIndent`) | `li { padding-left: 36px }` | `a:pPr@marL="342900"` |
| 마커 내밈 (행잉 인덴트) | 36px (`web.ListMarkerHang`) | `li::before { left: 0 }` | `a:pPr@indent="-342900"` |
| 항목 사이 간격 | 0 (`web.ListItemGap`) | `li { margin: 0 }` — 수직 리듬은 줄높이 소유 | `spcBef`/`spcAft` 없음 |
| `ul` 마커 | `•` 글리프 (`web.ListBulletChar`) | `li::before { content: "•" }` | `a:buChar char="•"` |
| `ol` 마커 | `1.` 형식 | `counter(balpyo-item) "."` (엔진 카운터) | `a:buAutoNum type="arabicPeriod"` |
| 마커 폰트 | li의 확정 폰트 토큰 | `::before`가 li 폰트를 상속 | `a:buFont` = li의 latin typeface (body는 `+mn-lt` 테마 참조) |

마커를 UA `::marker`가 아니라 엔진이 그리는 이유: OOXML은 마커를 `marL+indent` 위치에
왼쪽 정렬로 그리고 뷰어가 그리는 방식이라 바꿀 수 없다 — CSS `::before`가 OOXML 모델을
표현할 수 있으므로 맞추는 쪽은 웹이다. 마커 글리프도 UA의 disc 도형이 아니라 폰트
글리프다 — PPTX `buChar`가 텍스트 폰트로 그리는 것과 같은 곳에서 나와야 마커 모양이
갈라지지 않는다.

## 내장 테마 `minimal`의 토큰

테마 패키지 포맷은 Iteration 10이다 — 지금은 하드코딩된 최소 표가 두 렌더러의 공급원이다.

색 토큰 (`--balpyo-color-*` ↔ clrScheme 슬롯):

| 토큰 | RGB | clrScheme 슬롯 | schemeClr 참조 |
|---|---|---|---|
| `fg` | `1A1A1A` | `dk1` | `tx1` |
| `bg` | `FFFFFF` | `lt1` | `bg1` |
| `muted` | `6B7280` | `dk2` | `tx2` |
| `surface` | `F3F4F6` | `lt2` | `bg2` |
| `accent1` | `0F65F9` | `accent1` | `accent1` |

슬라이드 바탕도 `bg` 토큰에서 나온다 (웹은 `.balpyo-slide { background: var(--balpyo-color-bg) }`,
PPTX는 마스터 배경의 `bg1` 참조).

폰트 토큰 (`--balpyo-font-*` ↔ fontScheme):

| 토큰 | 스택 | fontScheme 슬롯 |
|---|---|---|
| `heading` | Noto Sans, Noto Sans CJK KR, sans-serif | major |
| `body` | Noto Sans, Noto Sans CJK KR, sans-serif | minor |
| `mono` | Noto Sans Mono, Noto Sans CJK KR, monospace | 없음 (리터럴) |

공간 토큰 (`--balpyo-space-*`) — [레이아웃 태그](soml/layout.md)의 `gap`/`pad` 값 어휘다
(PRD §8.4의 `space` 스케일). 값은 **수직 리듬**에서 나온다: 본문 24px, 줄박스 36px을 기준
삼아 6px(리듬의 최소 눈금) 배수로 벌린 사다리다.

| 토큰 | 값(px) | 리듬에서의 자리 | PPTX 슬롯 |
|---|---|---|---|
| `xs` | 6 | ¼ md | **없음** |
| `sm` | 12 | ½ md | **없음** |
| `md` | 24 | 본문 한 글자 (= 웹 기본 스택 간격) | **없음** |
| `lg` | 36 | 본문 줄박스 한 줄 (24 × 1.5) | **없음** |
| `xl` | 48 | 2 md | **없음** |
| `2xl` | 72 | 줄박스 두 줄 | **없음** |

**공간 토큰은 PPTX 테마 파트에 대응 개념이 없는 유일한 토큰이다.** 색 토큰은 clrScheme
슬롯, 폰트 토큰은 fontScheme 슬롯을 갖는데(위 두 표) OOXML 테마에는 공간이 없고, 무엇보다
배치는 지오메트리 패스가 잰 **좌표로 구워져** 산출물에 남는다 — PowerPoint에서 `gap`을
고친다는 것은 도형을 옮긴다는 뜻이므로 참조로 남길 대상이 없다
([ADR 0012](../decisions/0012-space-tokens-have-no-theme-slot.md)).

웹에서는 색·폰트와 같은 규칙으로 참조가 남는다: `gap="md"`는
`gap: var(--balpyo-space-md)`, `gap="24"`는 `gap: 24px`다 — 같은 24px이지만 테마 참조인지
저자의 리터럴인지가 값에 남는다. 공간 변수는 레이아웃 태그가 있는 덱에만 선언된다 (참조하는
자리가 컨테이너의 `gap`/`pad`뿐이다).

4·8·16·32 같은 2의 거듭제곱 사다리를 쓰지 않은 이유는 36px 줄박스의 약수·배수가 아니어서
여백이 글줄 격자에서 매번 미끄러지기 때문이다. 스테이지 여백(64px)은 아직 이 표 밖에 있다 —
PRD §8.4의 `slidePadding` 토큰이 들어오는 Iteration 10이 합칠 자리다.

반경 토큰 (`radius` 값 어휘) — 공간 사다리의 최소 눈금(6px)에 얹혀 있다. 카드가 안쪽 여백과
모서리를 나란히 보이므로 두 값이 같은 눈금이어야 한다:

| 토큰 | 값(px) | 공간 사다리에서의 자리 | PPTX 슬롯 |
|---|---|---|---|
| `sm` | 6 | = `xs` | **없음** — OOXML 테마에 반경 개념이 아예 없다 |
| `md` | 12 | = `sm` | **없음** (동일) |
| `lg` | 24 | = `md` | **없음** (동일) |

그림자 토큰 (`shadow` 값 어휘) — 오프셋을 **극좌표**(거리·방향)로 담는 이유는 OOXML이
극좌표이고 방향이 60000분의 1도 정수라, 직교 → 극 변환을 산출 경계에 두면 반올림 손실이
그쪽에 생기기 때문이다 (변환하는 쪽이 웹이다):

| 토큰 | 거리 | 방향 | 번짐 | 색 | PPTX 슬롯 |
|---|---|---|---|---|---|
| `soft` | 4px | 90° (아래) | 16px | `fg` × 알파 0.15 | **v1은 쓰지 않는다** (아래) |

- **반경 토큰에는 PPTX 슬롯이 없다** — 공간 토큰과 같은 처지이지만 이유가 다르다. 공간은
  배치가 좌표로 구워져서 참조할 대상이 없는 것이고, 반경은 **OOXML 테마 파트에 반경이라는
  개념 자체가 없다**: `prstGeom/avLst`의 조정값은 도형마다 구워진다. 그래서 웹도 반경을
  변수 참조가 아니라 확정 수치로 낸다 (두 타깃이 같은 방식으로 굽는다).
- **그림자에는 슬롯이 있지만 v1은 쓰지 않는다.** `fmtScheme/effectStyleLst`가 그 자리인데
  스타일이 **셋뿐**이고 도형이 `style/effectRef`로 하나를 고르는 구조라, 토큰이 셋을 넘는
  순간 "어떤 도형의 그림자가 다른 토큰의 값으로 그려진다"로 무너진다. 그래서 도형마다
  `effectLst/outerShdw`를 직접 낸다 — **그림자의 색만은 참조로 남는다**
  ([ADR 0020](../decisions/0020-shadow-tokens-only.md), 슬롯 없음의 선례는
  [ADR 0012](../decisions/0012-space-tokens-have-no-theme-slot.md)).
- 그림자를 **수치로 쓸 수 없는** 이유도 여기 있다: CSS `box-shadow`의 4수치와 `outerShdw`의
  파라미터가 1:1이 아니라(번짐 정의가 다르고 `spread`에 대응이 없다) 같은 수치가 두 타깃에서
  다른 그림이 된다. 토큰은 웹에서 변수 참조로, PPTX에서 구운 파라미터로 각자 나간다.

## 값 문법

- 수치는 **단위 없는 논리 스테이지 px** 하나다 (PRD §13-5). `px`·`pt`·`em` 접미사는 에러다.
  외관 수치(`size`)에는 `%`도 에러다 — PPTX 폰트 크기가 절대값이라 상대 크기를 번역할 곳이
  없다. 배치 길이(`gap`·`pad`·`w`/`h`·`basis` …)는 `%`를 받는다: CSS 백분율이 그대로
  브라우저에서 풀리고 그 결과를 지오메트리 패스가 픽셀로 잰다
  ([레이아웃 태그](soml/layout.md)).
- 색은 토큰 이름 또는 `#RRGGBB`/`#RGB` 리터럴이다. `#RGB`는 CSS 규칙대로 자리를 두 배로
  늘려 정규화한다. `gradient(…)`·`shade(…)` 같은 색 함수는 문법에 있지만(PRD §7.6)
  Iteration 8에서 열린다 — 지금 쓰면 "아직 쓸 수 없다"는 진단이 나온다.

속성별 에러 조건과 진단 메시지는 [`<text>`](soml/text.md)와
[텍스트 서브셋](soml/text-subset.md), 배치 속성은
[레이아웃 태그](soml/layout.md)의 "빌드 에러가 나는 조건"에 있다.
