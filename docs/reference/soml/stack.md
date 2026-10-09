# `<stack>` — 겹침 컨테이너

`<stack>`은 자식을 **같은 칸에 겹쳐 쌓는** 레이아웃 컨테이너다 ([ADR 0067](../../decisions/0067-prd-is-retired-and-its-standing-decisions-move-here.md) 결정 3). `<row>`가
자식을 가로로, `<col>`이 세로로 늘어놓는다면 `<stack>`은 **한 자리에 층으로** 놓는다 —
그림 위의 배지, 본문 아래의 워터마크, 카드 위의 라벨이 그 자리다.

구현은 **단일 셀 grid**다: 모든 자식이 `grid-area: 1/1`에 겹치되 in-flow로 남으므로
stack의 크기가 가장 큰 자식에 맞춰 자동 결정된다. 그래서 `<stack>`은
[레이아웃 태그](layout.md)의 하나이고 그 페이지의 계약(컨테이너는 CSS의 이름 바꾸기 층,
PPTX에서 사라진다, 자식만 절대 좌표 도형이 된다)을 그대로 진다. 이 페이지는 겹침에만 있는
두 가지 — **9지점 앵커(`at`)와 z-순서** — 를 담는다.

`<stack>`은 새 요소 **종류**가 아니라 새 **흐름**이다 — `<row>`와 파서·값 층·PPTX·지오메트리
패스에서 산출이 전부 같고, 다른 것은 웹 CSS 한 줄(단일 셀 grid)뿐이다
([ADR 0028](../../decisions/0028-stack-is-a-flow-not-a-kind.md)).

## 컨테이너 속성

`<stack>` 자신의 정렬·여백·크기다. `<row>`/`<col>`/`<grid>`의 컨테이너 속성에서 **`gap`·
`wrap`·`cols`·`rows`가 빠진** 목록이다.

| 속성 | 허용 값 | CSS 대응 |
|---|---|---|
| `align` | `start` `center` `end` `stretch` `baseline` | `align-items` — **`at`이 없는** 자식의 세로 기본 정렬 |
| `justify` | `start` `center` `end` `between` `around` `evenly` | `justify-content` — `at`이 없는 자식의 가로 기본 정렬 |
| `pad` | 공간 토큰 · 수치 · `%` (1~4값) | `padding` — 앵커·좌표의 기준 박스를 여백 안쪽으로 좁힌다 |
| `w` / `h` | 수치 · `%` · `fill` | `width` / `height` |
| `min-w` / `min-h` · `max-w` / `max-h` | 수치 · `%` · `fill` | `min-*` / `max-*` |
| `ratio` | `W:H` 정수 쌍 (`4:3`) | `aspect-ratio` |

- **`gap`·`wrap`이 없는 이유는 단일 셀이다.** `gap`은 셀 **사이**의 간격인데 stack의 자식은
  전부 같은 칸에 있어 사이가 없다 — 받아 놓고 산출 CSS에 내면 아무것도 하지 않는 선언이 되고,
  그 침묵이 닫힌 어휘가 막으려는 실패다. `wrap`은 flex 전용(`flex-wrap`)이라 grid 흐름인
  stack에서 CSS가 무시한다. `cols`/`rows`는 `<grid>` 전용이고([레이아웃 태그](layout.md)),
  stack의 격자는 언제나 암시 트랙 하나다.
- **크기·여백 7종은 [`<shape>`](shape.md)·컨테이너와 같은 한 벌이다** — 박스의 성질이므로
  값 어휘·진단이 한 글자도 다르지 않다. `<stack ratio="4:3">`은 카드 그리드의 흔한 칸이다.
- `align`/`justify`는 stack에서 **뜻이 있다**: `at`을 쓰지 않은 자식의 기본 정렬이고, `at`이
  그것을 자식 하나에서 덮는다.

## `at` — 9지점 앵커

`at`은 자식을 stack 셀의 **9지점** 중 하나에 붙인다. `place-self`로 컴파일되고, 값은 3×3
격자의 합성 이름이다 (`center`만 한 단어).

| `at` | `place-self` (세로 가로) | 붙는 자리 |
|---|---|---|
| `top-left` | `start start` | 좌상 |
| `top-center` | `start center` | 상단 중앙 |
| `top-right` | `start end` | 우상 |
| `center-left` | `center start` | 좌측 중앙 |
| `center` | `center center` | 정중앙 |
| `center-right` | `center end` | 우측 중앙 |
| `bottom-left` | `end start` | 좌하 |
| `bottom-center` | `end center` | 하단 중앙 |
| `bottom-right` | `end end` | 우하 |

- **축 순서는 `place-self: <align-self> <justify-self>`다** — 앞이 블록 축(세로), 뒤가
  인라인 축(가로)이다. 그래서 `top-right`는 `start end`(세로 start · 가로 end)이지
  `end start`가 아니다.
- **무엇에 대해 붙는가**: stack의 크기는 가장 큰 자식이 정하므로 앵커의 기준 격자도 그것이다.
  `w`/`h`를 주면 그 박스가 기준이고, `pad`를 주면 여백 **안쪽**이 기준이다.
- `center-center`는 어휘에 **없다** — 같은 자리를 두 이름으로 받지 않는다는 기준이라
  `center` 한 단어를 쓴다 (오타로 `center-center`를 쓰면 진단이 `center`를 권한다).
- `at`을 쓰지 않은 자식은 컨테이너의 `align`/`justify` 기본 정렬을 따른다.

## `x` / `y` — 절대 좌표

`at`의 9지점으로 부족할 때, `x`/`y`는 자식을 **부모 상대 절대 좌표**로 놓는다 (그 자식만
`position: absolute`가 되고 `left`/`top`으로 나간다).

- **기준은 부모의 패딩 박스다.** `<stack pad>`가 있으면 여백 **안쪽**이 원점이다 —
  `x="0" y="0"`은 여백 안쪽 좌상단이다.
- **음수를 받는다.** 크기 속성(`w`/`h`/`gap`)은 음수가 에러이지만 좌표는 음수가 정당하다 —
  박스 밖으로 반쯤 걸친 배지(전면 블리드)가 오버레이의 가장 흔한 장면이다. `x="-16"`은 박스
  왼쪽 밖 16px이고, 산출물까지 부호 그대로 흐른다 (자르지 않는다 — [ADR 0067](../../decisions/0067-prd-is-retired-and-its-standing-decisions-move-here.md) 결정 1의 오버플로 허용).
- **`%`는 부모 축 기준이다** — `x="50%"`는 부모 폭의 절반, `y="100%"`는 부모 높이의 전부다
  (박스 아래 모서리에 얹는 표기).
- **공간 토큰(`x="md"`)과 `fill`은 받지 않는다.** 토큰은 여백·간격의 어휘라 테마를 따라
  움직이는 것이 옳지만 좌표가 테마 교체로 움직이면 겹침이 깨지고, `fill`은 크기이지 위치가
  아니다.
- **한 축만 지정할 수 있다.** `x`만 준 자식은 세로가 정적 위치(컨테이너 정렬)로 남는다 —
  지정하지 않은 축은 선언을 내지 않는다 (`top: auto`).

`x`/`y`는 **`<stack>`의 자식**이거나 **`layout="canvas"` 슬라이드의 직속 자식**에서만 쓴다
([`<slide>`의 `layout`](slide.md)). `layout="content"`(기본) 슬라이드의 직속 자식에서는
에러다 — 그 스테이지는 세로 스택이라 좌표가 배치를 소유하는 자리가 아니다.

## z-순서

겹친 자식이 그리는 순서는 **문서 순서**다 — **먼저 쓴 것이 아래, 뒤에 쓴 것이 위**다.
별도 `z-index` 속성은 **없다**.

```html
<stack>
  <shape type="rect" fill="muted"/>   <!-- 맨 아래 -->
  <shape type="rect" fill="accent1"/> <!-- 가운데 -->
  <text>맨 위</text>                  <!-- 맨 위 -->
</stack>
```

- **`z-index`를 열지 않은 이유**는 순서가 이미 z축이기 때문이다 ([ADR 0067](../../decisions/0067-prd-is-retired-and-its-standing-decisions-move-here.md) 결정 3). 저자용 z-index를
  열면 문서 순서와 중복된 채널이 되고, 둘이 어긋날 때 누가 이기는지 저자가 외워야 한다 —
  한 축의 위치는 한 채널이 정한다는 기준의 적용이다
  ([ADR 0031](../../decisions/0031-position-is-one-channel.md)). `style` 탈출구의
  allowlist에도 `z-index`는 없다.
- **이 순서가 그대로 PPTX `spTree` 순서가 된다** — OOXML은 나중 도형을 위에 그리므로, 저자가
  SOML만 읽고 두 타깃의 겹침 순서를 예측할 수 있다. 웹은 이 계약을 지키려고 stack의 모든
  자식을 위치 지정 요소로 만든다 (아래 렌더 결과 절,
  [ADR 0029](../../decisions/0029-paint-order-follows-pptx.md)).

## `at`과 `x`/`y`는 함께 쓸 수 없다

둘 다 자식의 자리를 정하는 채널이라(`at`은 9지점 앵커, `x`/`y`는 절대 오프셋) 함께 쓰면
에러다:

```
slides/…:3:29: at과 x를 함께 쓸 수 없다 (x는 3:40에 있다) — 둘 다 자리를 정하는 채널이라 하나만 쓴다 (at은 9지점 앵커, x/y는 절대 오프셋)
```

진단은 두 속성의 자리를 모두 가리켜 저자가 어느 하나를 지울지 고르게 한다 — `variant`와
`frame`을 함께 받지 않은 것과 같은 기준이다
([ADR 0031](../../decisions/0031-position-is-one-channel.md)).

## 예제

`testdata/fixtures/06_stack/slides/002_badge_overlay.slide` — 그림 위 배지:

```html
<slide id="badge-overlay">
  <h2>배지 오버레이</h2>
  <stack w="360" h="240">
    <image src="assets/dot.png" fit="cover" w="fill" h="fill" alt="배경 그림"/>
    <shape type="rounded-rect" fill="muted" pad="sm" at="top-right">
      <text size="14" color="bg">NEW</text>
    </shape>
  </stack>
</slide>
```

`003_watermark.slide` — 본문 **아래** 워터마크 (문서 순서가 z-순서다):

```html
<slide id="watermark">
  <stack w="fill" h="fill">
    <shape type="rect" fill="muted" opacity="0.12" rotate="-20" w="fill" at="center">
      <text size="96" color="muted" align="center">DRAFT</text>
    </shape>
    <col gap="md" pad="xl">
      <h1>분기 실적 보고</h1>
      <p>이 슬라이드는 초안이다 — 워터마크가 본문 아래에 깔린다.</p>
    </col>
  </stack>
</slide>
```

`001_anchors.slide` — 9지점 전수 (같은 크기 배지 9개가 각 앵커에 붙는다):

```html
<stack w="fill" h="360">
  <shape type="rounded-rect" fill="surface" w="fill" h="fill"/>
  <shape type="rect" fill="accent1" w="64" h="40" at="top-left"/>
  <shape type="rect" fill="accent1" w="64" h="40" at="top-center"/>
  <!-- … center-* · bottom-* 여섯 개 … -->
  <shape type="rect" fill="accent1" w="64" h="40" at="bottom-right"/>
</stack>
```

`006_in_container.slide` — `<grid>` 안의 stack (겹침이 흐름 배치의 한 칸이 된다):

```html
<grid cols="3" gap="md">
  <stack ratio="4:3">
    <image src="assets/dot.png" fit="cover" w="fill" h="fill" alt="카드 1"/>
    <text size="14" color="bg" at="bottom-left">하나</text>
  </stack>
  <!-- … 카드 둘 더 … -->
</grid>
```

`005_absolute.slide` — `x`/`y`로 양수·음수·`%`를 섞는다:

```html
<stack w="480" h="300">
  <shape type="rounded-rect" fill="surface" w="fill" h="fill"/>
  <shape type="ellipse" fill="accent1" w="48" h="48" x="24" y="24"/>
  <shape type="star" fill="accent1" w="40" h="40" x="-16" y="-16"/>
  <shape type="rect" fill="muted" w="80" h="24" x="50%" y="100%"/>
  <text size="14" color="muted" x="12" y="12">좌상단에서 12px</text>
</stack>
```

## 렌더 결과

### 웹 — 단일 셀 grid

`<stack>`은 `<div class="balpyo-stack">` 하나가 되고, `at`은 자식의 `place-self`로, `x`/`y`는
`position: absolute; left/top`으로 나간다:

```html
<div class="balpyo-stack" data-balpyo-id="s0.1" data-balpyo-el="stack" style="width: 100%; height: 360px">
<div class="balpyo-shape" … style="width: 64px; height: 40px; place-self: start start">…</div>
<div class="balpyo-shape" … style="width: 64px; height: 40px; place-self: start end">…</div>
</div>
```

엔진 CSS는 `<stack>`이나 캔버스 스테이지를 실제로 쓴 덱에만 나간다 (안 쓴 골든이 이
iteration 때문에 움직이지 않게 — [레이아웃 태그](layout.md)의 `layoutCSS`와 별도 절):

```css
.balpyo-stack { display: grid; position: relative; }
.balpyo-stage-canvas { display: grid; }
:is(.balpyo-stack, .balpyo-stage-canvas) > * { grid-area: 1 / 1; position: relative; }
:is(.balpyo-stack, .balpyo-stage-canvas) > :is(.balpyo-text, .balpyo-list) { width: auto; }
```

- **단일 셀 grid + `grid-area: 1/1`**: 자식이 같은 칸에 겹치되 in-flow로 남아 stack 크기가
  가장 큰 자식에 맞춰 결정된다 (absolute 전면 방식은 컨테이너 크기가 소실되어 배제 — [ADR 0067](../../decisions/0067-prd-is-retired-and-its-standing-decisions-move-here.md) 결정 3).
- **stack 자신의 `position: relative`**: `x`/`y` 자식의 포함 블록을 stack의 패딩 박스로 만든다.
  오프셋이 없는 relative는 잰 rect를 움직이지 않는다.
- **자식의 `position: relative`가 칠 순서를 문서 순서로 되돌린다.** CSS는 같은 쌓임 문맥에서
  **위치 지정된 자손을 정적 자손보다 나중에** 칠하므로, `x`/`y` 자식(absolute)과 정적 자식이
  섞이면 겹침 순서가 PPTX(문서 순서)와 반대가 된다. 모든 자식을 위치 지정 요소로 만들면
  형제끼리 문서 순서로 칠해져 계약이 복원된다 — 엔진이 `z-index`를 내지 않는다
  ([ADR 0029](../../decisions/0029-paint-order-follows-pptx.md)).
- **`x`/`y`는 `left`/`top`이지 `transform: translate`가 아니다.** 측정 뷰가 변형을 끄므로
  ([ADR 0021](../../decisions/0021-measure-view-disables-transforms.md)), 변형으로 위치를
  내면 지오메트리 패스가 잰 rect에 위치가 남지 않고 PPTX가 겹치지 않은 좌표를 받는다. 음수·
  백분율도 그대로다: `left: -16px`, `left: 50%; top: 100%`.
- **캔버스 스테이지는 stack과 같은 규칙을 쓴다** — `:is(.balpyo-stack, .balpyo-stage-canvas)`
  한 선택자가 둘을 함께 겨눈다 ([`<slide>`의 `layout`](slide.md)).

### PPTX — stack은 사라지고 자식만 도형이 된다

`<stack>`은 `<row>`/`<col>`과 똑같이 **도형 0개**다. 자식이 절대 좌표 도형이 되고, **문서
순서가 곧 `spTree` 순서(= z-순서)**다. `x`/`y` 좌표 자식과 `at`·정적 자식을 섞어도 등장 순서가
문서 순서로 유지되고, `p:sp`(도형·텍스트)와 `p:pic`(그림)이 같은 순회의 문서 순서 자리에 낀다:

```xml
<p:cNvPr id="2" name="h2 s3.0"/>       <!-- 표제 -->
<p:cNvPr id="3" name="shape s3.1.0"/>  <!-- muted (x/y) — 문서상 먼저 = 맨 아래 -->
<p:cNvPr id="4" name="shape s3.1.1"/>  <!-- accent1 (at=center) -->
<p:cNvPr id="5" name="shape s3.1.2"/>  <!-- fg (x/y) — 문서상 뒤 = 위 -->
<p:cNvPr id="6" name="text s3.1.3"/>   <!-- 가운데 텍스트 (at=center) = 맨 위 -->
```

- **컨테이너는 도형 id를 소비하지 않는다** — id는 도형을 실제로 낸 자리에서만 오른다
  (`id="1"`은 spTree 그룹). `<stack>`이 끼어도 자식 번호가 건너뛰지 않는다.
- **음수 좌표는 부호 있는 `a:off`로 나간다** — `x="-24"` → `<a:off x="-228600"/>` (0쪽으로
  자르지 않는다).

**`<stack>`은 `<group>`과 여기서 갈라진다:**

| | 산출물 | 편집 단위 | 좌표 |
|---|---|---|---|
| `<stack>` | 도형 0개 — 배치 후 **소멸** | 없음 (자식이 낱개 도형) | 자식은 슬라이드 절대 좌표 |
| `<group>` (Iteration 9) | `grpSp` — **남는다** | 그룹 하나 | 그룹 상대 좌표 |

`<stack>`은 겹침을 **배치**로 표현하고, `<group>`은 겹침을 **편집 단위**로 묶는다. 겹친
도형이 PowerPoint에서 각각 선택·이동되어야 하면 `<stack>`이고, 한 덩어리로 움직여야 하면
`<group>`이다 ([ADR 0028](../../decisions/0028-stack-is-a-flow-not-a-kind.md),
[ADR 0011](../../decisions/0011-layout-containers-are-not-shapes.md)).

## 빌드 에러가 나는 조건

파서(`soml.Parse`) — 이름·문법. `at`·`x`/`y`는 배치 속성이므로 파서는 **어느 요소에서든**
받는다 (부모 한정은 값 층이 본다):

| 조건 | 진단 메시지 |
|---|---|
| stack에 `gap`·`wrap`을 썼다 | `<stack>에 알 수 없는 속성 gap (쓸 수 있는 속성: align, justify, pad, w, h, min-w, min-h, max-w, max-h, ratio, grow, shrink, basis, span, self, order, at, x, y, style)` |
| stack에 `cols`·`rows`를 썼다 | `<stack>에 알 수 없는 속성 cols (쓸 수 있는 속성: …)` |
| stack 안에 문자 데이터를 직접 썼다 | `<stack> 안에 문자 데이터를 직접 쓸 수 없다 (<text>로 감싼다)` |

값 해석(`core.Resolve`) — 값 어휘와 **부모 한정**은 이 층이 소유한다. 진단은 고칠 방법
(`<stack>`으로 감싸기·`layout="canvas"`)을 담는다:

| 조건 | 진단 메시지 |
|---|---|
| `at`을 stack 밖에서 썼다 | `at은 <stack>의 자식에서만 쓴다 (부모는 <row>다) — 겹침이 필요하면 <stack>으로 감싼다` |
| 어휘 밖의 `at` | `알 수 없는 at "middle" (쓸 수 있는 9지점 앵커: top-left, top-center, top-right, center-left, center, center-right, bottom-left, bottom-center, bottom-right)` |
| 오타·짧은 이름·뒤바뀐 축 | `알 수 없는 at "top" — "top-center"를 뜻한 것인가? (9지점 앵커: …)` |
| `x`/`y`를 유효하지 않은 부모에서 썼다 | `x는 <stack>의 자식이나 layout="canvas" 슬라이드의 직속 자식에서만 쓴다 (여기는 슬라이드 직속이다) — <stack>으로 감싸거나 슬라이드에 layout="canvas"를 준다` |
| `x`/`y`에 공간 토큰을 썼다 | `공간 토큰 "md"는 gap·pad에만 쓴다 — x는 수치·백분율이다` |
| `at`과 `x`/`y`를 함께 썼다 | `at과 x를 함께 쓸 수 없다 (x는 …에 있다) — 둘 다 자리를 정하는 채널이라 하나만 쓴다 (at은 9지점 앵커, x/y는 절대 오프셋)` |

크기·여백 속성(`pad`·`w`/`h`·`ratio` …)의 진단은 [레이아웃 태그](layout.md)·[`<shape>`](shape.md)와
**같은 문구**다 — 목록이 한 벌이므로 진단도 한 벌이다.
