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

`variant` 속성은 `<text>`와 블록 태그(`h1`~`h3`, `p`, `blockquote`, `pre`)가 받는다.
**어휘는 테마 소유다 — 표준화하지 않는다** (PRD §13-6): 내장 `minimal` 테마는 `kicker`
(크기 18 + 굵기 700 + `accent1`)와 `lead`(크기 32)를 정의한다.

**테마에 없는 variant는 에러가 아니라 경고 + 기본 외관 폴백이다** (PRD §13-6) — variant
어휘는 테마의 사유물이라, 다른 테마로 만든 콘텐츠가 이 테마에서 빌드되지 않으면 콘텐츠의
테마 간 이식성이 깨진다. 엄격함이 필요한 곳(CI 등)은 `--strict`가 경고를 에러로 승격한다
([CLI](cli.md)):

```
slides/006_variant.slide:6:16: warning: variant "hero"가 테마에 없다 — 기본 외관으로 계속한다 (이 테마의 variant: kicker, lead)
```

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

- OOXML `algn`의 `l`/`r`은 물리 이름처럼 보이지만 문단 방향(RTL)에 따라 뜻이 뒤집히는
  **논리 값**이다 — SOML이 `start`/`end`를 쓰는 이유와 정확히 짝이 맞는다.
- 확정 값은 요소의 인라인 `style` 속성으로 나간다 — cascade layer 위에 있으므로 저자
  CSS가 외관을 덮지 못한다 ([ADR 0003](../decisions/0003-inline-style-appearance.md)).

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
