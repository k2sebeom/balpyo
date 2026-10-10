# `<slide>`

`.slide` 파일 하나의 루트 요소다. 슬라이드 하나가 파일 하나이고, 파일의 역할은 확장자가
아니라 루트 요소가 정한다.

## 속성

| 속성 | 값 | 기본값 | 뜻 |
|---|---|---|---|
| `layout` | 표준 카탈로그 11종 | `content` | 슬라이드의 배치 레이아웃 (아래) |
| `id` | 문자열 | 파일명에서 유도 | 딥링크 앵커. 덱 전체에서 유일해야 한다 |
| `skip` | 없음 · `true` · `false` | `false` | 참이면 이 슬라이드를 렌더에서 뺀다 |
| `justify` | `start` · `center` · `end` · `between` · `around` · `evenly` | 없음 (위에서부터 쌓는다) | 흐름 모드 슬라이드의 **세로 분배** (아래) |
| `enter` | `fade` · `rise` · `zoom` · `wipe` · `none` | 없음 | 직속 자식마다의 [등장](enter.md) |

- `id`를 생략하면 파일명에서 유도한다: 순서용 숫자 접두(`001_`, `010-`)를 떼고 확장자를
  뗀 나머지가 앵커가 된다 — `001_title.slide`는 `title`. 순서 접두가 앵커에 들어가지
  않으므로 슬라이드를 재배치해도 링크가 살아남는다.
- `skip`은 HTML 불리언 속성 관례를 따른다 — 값 없는 `<slide skip>`은 참이다. 건너뛴
  슬라이드는 덱 모델에 아예 들어가지 않으므로 슬라이드 번호에 구멍이 생기지 않는다.

## `layout` 속성

`layout`은 슬라이드의 **배치 레이아웃**이다. 어휘는 표준 카탈로그 11종으로 닫혀
있고, **카탈로그와 슬롯의 레퍼런스는 [`layout=` 페이지](../layouts.md)에 있다** — 여기서는
`<slide>`가 그 값을 어떻게 받는지만 본다.

| 값 | 뜻 |
|---|---|
| `content` (기본값) | 콘텐츠 자식을 **문서 순서의 세로 스택**으로 쌓는다 — 여백 64px, 간격 24px |
| `canvas` | 자식을 쌓지 않고 자식이 `x`/`y`·`at`으로 **자기 자리를 정한다** (자유 배치) |
| 나머지 9종 | `title` · `section-divider` · `two-column` · `image-full` · `image-text` · `quote` · `code` · `stat` · `end` — 배치는 테마의 `layouts/<이름>.css`가 소유한다 ([카탈로그](../layouts.md)) |

- **`content`는 `layout`을 생략한 슬라이드의 기본값이자 정당한 명시 표기다.** 구현된
  동작(세로 스택)에 이름이 있으므로 `layout="content"`가 에러가 아니고, 그 산출물은
  `layout` 없는 같은 슬라이드와 **바이트가 같다** (기본값과 명시가 같은 이름이다).
- **어휘 밖 값은 빌드 에러다** — 오타가 조용히 기본 레이아웃으로 렌더되지 않게 막는다:
  `unknown layout "titel" (available layouts: title, section-divider, content, two-column, image-full, image-text, quote, code, stat, canvas, end)`. 나열 순서는 카탈로그 순서
  (표지에서 마무리로)이고, 진단·문서·`soml.SlideLayouts()`가 같은 순서를 쓴다.
- **슬롯은 레이아웃이 정한다** — 직계 자식의 `slot="title"` 같은 배정은 레이아웃마다 허용
  이름이 다르고, 그 표도 [`layout=` 페이지](../layouts.md)에 있다. 슬롯을 하나도 쓰지 않으면
  슬라이드는 흐름 모드로 남는다 (슬롯 모드는 슬라이드마다, 덧붙는 방식이다).
- **렌더러는 문자열이 아니라 확정된 레이아웃 값을 소비한다** — `"canvas"`가 세 렌더러에
  흩어지지 않게 core가 어휘를 확정해 덱 모델(`Slide.Layout`)에 싣는다. 값 어휘는 파서가 아니라
  값 층이 보고, 값과 함께 구간(`LayoutSpan`)을 실어 진단이 `line:col`을 가리킨다.

### 캔버스에서 테마와 저자의 경계

`layout="canvas"`에서도 **배경·여백은 테마가 소유하고, 배치만 저자가 소유한다**.
스테이지는 여전히 테마의 64px 여백을 갖고, 자식의 `x`/`y`·`at`이 그 안에서 자리를 정한다.

- **캔버스 좌표의 기준은 스테이지의 패딩 박스**(여백 안쪽)다. `x="0" y="0"`은 슬라이드
  가장자리가 아니라 **여백 안쪽 좌상단**이다 — 캔버스 직속 자식의 좌표는 스테이지 절대
  좌표와 **일치하지 않는다**.
- **슬라이드 가장자리에 닿으려면 음수 좌표를 쓴다.** 여백이 64px이므로 `x="-64"`가 왼쪽
  가장자리이고, 그보다 작은 음수는 슬라이드 밖으로 걸친 전면 블리드다 (자르지 않는다 —
  오버플로는 허용이다).
- 캔버스 스테이지는 [`<stack>`](stack.md)과 **같은 겹침 규칙**을 쓴다 (단일 셀 grid, 문서
  순서 z-순서) — 자식으로 `<stack>`을 두면 그 stack이 자기 셀 좌표계를 다시 연다.

`008_canvas_free.slide` — 캔버스 직속 자식이 `x`/`y`로 자리를 정한다 (음수·`%` 포함):

```html
<slide id="canvas-free" layout="canvas">
  <h2>자유 배치</h2>
  <shape type="rounded-rect" fill="surface" w="240" h="160" x="80" y="120"/>
  <shape type="ellipse" fill="accent1" w="96" h="96" x="50%" y="40%"/>
  <text size="18" color="bg" x="120" y="160">캔버스</text>
  <shape type="star" fill="accent1" w="64" h="64" x="-24" y="24"/>
</slide>
```

## `justify` 속성 — 세로 리듬

흐름 모드 스테이지는 **세로 flex 컨테이너**이므로 `justify` 한 선언이 곧 세로 분배다. 값
어휘는 컨테이너([`<row>`·`<col>`](layout.md))의 `justify`와 **같은 6종**이고, `flex-start`·
`space-between` 같은 CSS 접두형은 받지 않는다 (진단이 접두 없는 이름을 알려 준다).

```xml
<slide id="summary" justify="between">
  <h2>요약</h2>
  <p>가운데 블록은 남은 공간이 나눠 준 자리에 앉는다.</p>
  <p role="caption">아래 끝에 닿는다</p>
</slide>
```

- **랩퍼를 지우는 것이 이 속성의 목적이다.** 이전에는 세로 리듬을 저자가 슬라이드마다
  `<col h="fill" justify="between">`으로 만들었고, 그 랩퍼 안의 자식은 슬라이드 직계 자식이
  아니게 되어 **`slot=`을 쓸 수 없었다** (슬롯은 직계 자식에만 실린다). 스테이지가 직접
  분배하면 세로 리듬과 슬롯이 서로를 배제하지 않는다.
- **슬롯 모드 슬라이드에서는 빌드 에러다** — 그 세로 리듬은 테마의 `*.slots.css`가 소유하고,
  한 축에 소유자가 둘이면 어느 쪽이 이기는지 저자가 외워야 한다. 슬롯을 하나도 쓰지 않은
  슬라이드에서 쓴다 ([`layout=` 페이지](../layouts.md)의 "세로 리듬은 누가 소유하는가").
- `gap`·`align`은 `<slide>`에 열려 있지 않다. 스테이지의 간격(24px)과 가로 정렬은 각각 테마의
  리듬과 [두 채널의 규칙](../layouts.md)이 소유한다 — 세로 분배만 저자에게 열린 이유는 그것이
  랩퍼로 흉낼 수밖에 없던 유일한 채널이었다는 것이다.
- **판정은 잰 좌표다**: `between`이면 첫 박스의 y가 콘텐츠 박스 위(64), 마지막 박스의 아래가
  콘텐츠 박스 아래(656)에 닿고, 그 좌표가 그대로 PPTX `a:off`로 실린다.

## 예제

`testdata/fixtures/01_text/slides/001_title.slide`:

```html
<!-- 표지. size·align을 명시해 파서가 원문 값을 그대로 보존하는지 본다. -->
<slide id="title">
  <text role="caption" size="20" color="muted" align="center">Platform Team</text>
  <text role="title" size="64" color="fg" align="center">2026 아키텍처 리뷰</text>
  <text role="body" align="center">상반기 회고와 하반기 계획</text>
  <note>인사 30초. 청중에 인프라팀 포함되어 있음.</note>
</slide>
```

## 렌더 결과

**웹** — `<section class="balpyo-slide">` 하나가 된다. `id` 속성이 그대로 HTML `id`
(딥링크 앵커)가 되고, `data-slide-index`(0-based 렌더 순서)와 `data-slide-source`
(원본 파일 경로)가 붙는다. 슬라이드는 논리 스테이지 크기(16:9는 1280×720px)를 그대로
갖고, 콘텐츠 자식은 `.balpyo-stage` 안에 문서 순서의 세로 스택으로 쌓인다 (여백 64px,
간격 24px).

**PPTX** — `ppt/slides/slideN.xml` 파트 하나가 된다 (N은 1-based 렌더 순서). `id`는
`p:cSld@name`으로 나가 PowerPoint의 슬라이드 이름이 된다. 콘텐츠 자식은 `p:spTree`의
도형이 되고, 문서 순서가 곧 도형 순서(겹침 z-순서)다.

`layout="canvas"` 슬라이드의 웹 스테이지는 `balpyo-stage-canvas` 클래스가 더해져 세로
스택 대신 [`<stack>`](stack.md)과 같은 단일 셀 grid 겹침이 된다 — PPTX는 어느 레이아웃이든
자식이 절대 좌표 도형이므로 산출 구조가 같다 (레이아웃은 자식 좌표를 정할 뿐 도형이 되지
않는다).

지금 어휘에서 `<slide>`의 자식으로 올 수 있는 것은 `<text>`·`<note>`,
[텍스트 서브셋](text-subset.md)의 블록·리스트 태그(`h1`~`h3`, `p`, `blockquote`, `pre`,
`ul`/`ol`), [레이아웃 태그](layout.md)(`row`/`col`/`grid`/[`stack`](stack.md)/`spacer`), 그리고
[`<shape>`](shape.md)·[`<image>`](image.md)다. 인라인 태그는
슬라이드 직속이 될 수 없다 — 블록 안에 쓴다. 콘텐츠 자식은 [자식 배치 속성](layout.md)
(`grow`·`self`·`order` …)도 받는다 — 슬라이드의 세로 스택 자신이 flex 컨테이너다.

## 빌드 에러가 나는 조건

파서(`soml.Parse`)가 내는 진단 — 모두 `file:line:col` 위치를 갖는다:

| 조건 | 진단 메시지 |
|---|---|
| 루트가 `<slide>`가 아니다 | `the root element must be <slide> (write <text> inside it)` |
| 파일에 `<slide>`가 없다 | `missing root element <slide>` |
| 파일에 `<slide>`가 둘 이상 | `a file must have exactly one root element <slide> (one slide per file)` |
| 어휘 밖의 자식 태그 | `<slide> cannot contain <…> (available tags: <text>, <shape>, <image>, <note>, <h1>, <h2>, <h3>, <p>, <blockquote>, <pre>, <ul>, <ol>, <row>, <col>, <grid>, <stack>, <spacer>)` |
| 문자 데이터를 직접 썼다 | `character data cannot appear directly inside <slide> (wrap it in <text>)` |
| 어휘 밖의 속성 | `unknown attribute … on <slide> (available attributes: layout, id, skip, justify, enter)` |
| `id`에 값이 없다 (`<slide id>`) | `the id attribute needs a value (id="architecture")` |
| `id=""` | `the id attribute is empty (remove it to derive the id from the file name)` |
| `skip` 값이 `true`/`false`가 아니다 | `skip must be true or false (…)` |
| `layout`에 값이 없다 (`<slide layout>`) | `the layout attribute needs a value (layout="canvas")` |
| `layout=""` | `the layout attribute is empty (remove it to continue with content)` |
| `justify`에 값이 없다 (`<slide justify>`) | `the justify attribute needs a value (justify="between")` |
| `justify=""` | `the justify attribute is empty (remove it to stack from the top)` |

값 층·검증기(`core`)가 내는 진단 — 어휘와 파일을 넘어야 알 수 있는 것. `layout`의 값
어휘(테마와 함께 자란다)는 파서가 아니라 이 층이 본다:

| 조건 | 진단 메시지 |
|---|---|
| 어휘 밖의 `layout` 값 | `unknown layout "titel" (available layouts: title, section-divider, content, two-column, image-full, image-text, quote, code, stat, canvas, end)` |
| 레이아웃에 없는 `slot` 이름 | 레이아웃마다 허용 슬롯이 다르다 — [`layout=` 페이지](../layouts.md)의 슬롯 표 |
| 어휘 밖의 `justify` 값 | `unknown justify "middle" (available values: start, center, end, between, around, evenly)` |
| CSS 접두형 `justify` 값 | `justify takes unprefixed names — instead of "space-between", use "between"` |
| 슬롯 모드 슬라이드의 `justify` | `justify cannot be used on a slot-mode slide (the theme's content.slots.css owns the vertical rhythm) — use it on slides without slots` |
| 두 슬라이드의 ID가 같다 | `slide ID "…" collides with … (give one of them a different id, or remove the slide you do not need)` — 진단은 뒤 슬라이드의 `id` 값(파일명에서 유도했으면 `<slide>` 태그)에 서고, 메시지는 앞 슬라이드를 `파일:줄:열`로 가리킨다 |
