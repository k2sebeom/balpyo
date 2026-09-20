# 레이아웃 태그 — `row` · `col` · `grid` · `spacer`

레이아웃 태그는 **CSS flex/grid 시맨틱의 이름 바꾸기 층**이다 (PRD §7.4). 새 배치 모델이
아니라 아래 표의 CSS로 정확히 컴파일되고, 실제 배치는 브라우저가 한다 — 그 결과를
[지오메트리 패스](../../internals/geometry-pass.md)가 읽어 PPTX 좌표로 쓰므로 **저자는
좌표를 한 번도 계산하지 않는다**.

이 페이지의 중심은 CSS 대응 표다. 속성 이름에 뜻을 새로 붙이지 않겠다는 약속이 표의
형태로 있어야 저자가 아는 CSS 지식이 그대로 쓰인다.

## 태그 표

| 태그 | 컴파일 결과 (웹) | PPTX |
|---|---|---|
| `<row>` | `display: flex; flex-direction: row` | 도형 **0개** — 자식만 절대 좌표 도형이 된다 |
| `<col>` | `display: flex; flex-direction: column` | 도형 **0개** (동일) |
| `<grid>` | `display: grid` | 도형 **0개** (동일) |
| [`<stack>`](stack.md) | `display: grid` (**단일 셀** — 자식이 `grid-area: 1/1`에 겹친다) | 도형 **0개** — 자식이 문서 순서(= z-순서) 절대 좌표 도형이 된다 |
| `<spacer>` | `flex: 1`인 빈 `<div>` | 도형 **0개** — 밀어낸 결과가 이웃 도형의 좌표에 들어 있다 |

- 세 컨테이너의 자식 어휘는 같다: `<text>`, 블록 태그(`h1`~`h3`, `p`, `blockquote`,
  `pre`), 리스트 컨테이너(`ul`/`ol`), **레이아웃 컨테이너**(중첩), [`<shape>`](shape.md),
  [`<image>`](image.md), `<spacer>`. `<note>`는 올 수 없다 — 노트는 콘텐츠가 아니라 슬라이드
  메타데이터다. `<li>`도 올 수 없다 (리스트 컨테이너의 자식으로만 온다).
- 컨테이너가 올 수 있는 자리는 **슬라이드 직속·다른 컨테이너 안·[`<shape>`](shape.md) 안**이다.
  도형이 컨테이너를 담으면 그 도형은 **배경**이 되고 자식은 자기 좌표를 가진 형제 도형이 된다
  ([ADR 0049](../../decisions/0049-a-shape-that-holds-layout-becomes-a-backdrop.md)) — 컨테이너 자신이 도형
  0개라는 이 표의 규칙은 그 안에서도 같다.
- 컨테이너 안에 문자 데이터를 직접 쓸 수 없다 — `<text>`나 블록 태그로 감싼다. 배치
  컨테이너에 글을 직접 담으면 PPTX에서 그 글이 실릴 도형이 없다.
- `<spacer>`는 void 요소다 — `<spacer/>`로만 쓰고 자식도 문자 데이터도 갖지 않는다.
- **`<stack>`은 겹침 컨테이너다** (Iteration 6) — 단일 셀 grid로, 세 컨테이너와 자식 어휘가
  같지만 컨테이너 속성에서 `gap`·`wrap`·`cols`·`rows`가 빠진다 (단일 셀에는 셀 사이가 없다).
  자식 속성 `at`(9지점 앵커)·`x`/`y`(절대 좌표)도 이 iteration에 열렸다. 전용 페이지는
  [`<stack>`](stack.md)이다.

## 컨테이너 속성

컨테이너 자신의 배치와 크기다 — 받는 태그는 `<row>`/`<col>`/`<grid>` 셋이고 `<spacer>`는
컨테이너가 아니므로 받지 않는다 (아래 자식 속성만 받는다). **정하지 않은 속성은 CSS 선언을
하나도 내지 않는다** — 브라우저 기본값 그대로가 계약이다.

| 속성 | 허용 값 | CSS 대응 | 뜻이 있는 태그 |
|---|---|---|---|
| `gap` | 공간 토큰 · 수치 · `%` | `gap` | 셋 다 |
| `align` | `start` `center` `end` `stretch` `baseline` | `align-items` | 셋 다 |
| `justify` | `start` `center` `end` `between` `around` `evenly` | `justify-content` | 셋 다 |
| `wrap` | 값 없음 또는 `true` | `flex-wrap: wrap` | `<row>` `<col>` |
| `pad` | 공간 토큰 · 수치 · `%` (1~4값) | `padding` (항상 4값) | 셋 다 |
| `cols` | 트랙 개수 정수 또는 트랙 목록 | `grid-template-columns` | **`<grid>`만** |
| `rows` | 트랙 개수 정수 또는 트랙 목록 | `grid-template-rows` | **`<grid>`만** |
| `w` / `h` | 수치 · `%` · `fill` | `width` / `height` | 셋 다 |
| `min-w` / `min-h` | 수치 · `%` · `fill` | `min-width` / `min-height` | 셋 다 |
| `max-w` / `max-h` | 수치 · `%` · `fill` | `max-width` / `max-height` | 셋 다 |
| `ratio` | `W:H` 정수 쌍 (`16:9`) | `aspect-ratio: W / H` | 셋 다 |

**아래 다섯 행(`pad`·크기 6종·`ratio`)은 컨테이너 전용이 아니다** — [`<shape>`](shape.md)가
같은 목록을 받는다. 크기와 여백은 컨테이너의 성질이 아니라 **박스의 성질**이므로 두 요소가
한 벌을 공유하고, 그래서 `<row w="fill">`와 `<shape w="fill">`의 값 어휘·진단 문구가 한 글자도
다르지 않다. `gap`·`align`·`justify`·`wrap`·`cols`/`rows`는 흐름·트랙 속성이라 컨테이너
전용으로 남는다 (도형에는 배치할 자식 흐름이 없다).

- **`cols`/`rows`는 `<grid>`에만 열려 있다** — `grid-template-*`는 flex 컨테이너의 CSS에서
  무효이므로 `<row cols="3">`을 받아 주면 조용히 버려지는 선언이 되고, 그 침묵이 닫힌
  어휘가 막으려는 실패다. 고칠 방법이 "태그를 `<grid>`로 바꾸는 것"뿐이므로 진단이 이름
  층(파서)에 있다.
- **`cols="4"`는 트랙 개수다** (`repeat(4, 1fr)`) — 4px 트랙 하나가 아니다. 무단위 정수
  하나는 언제나 개수이므로 `cols="200"`은 트랙 200개다. 크기로 적으려면 형태를 붙이거나
  (`cols="1fr"`, `cols="100%"`, `cols="auto"`) 목록으로 두 개 이상 적는다
  (`cols="2fr 1fr"`).
- **트랙 목록의 어휘는 수치(px) · `Nfr` · `N%` · `auto` 넷이다.** `minmax()`·
  `fit-content()` 같은 CSS 함수는 열지 않았다 — 필요하면 `style` 탈출구가 그 자리다
  (아래).
- **`fill` = 해당 축의 100%다** — `flex-grow: 1`이 아니다. 남은 공간을 분배하는 것은
  `grow`이고, 두 표기가 같은 결과를 내지 않는 이유는 [ADR 0015](../../decisions/0015-fill-is-hundred-percent.md)에 있다.
- **`wrap`은 두 표기를 받는다**: 값 없는 `<row wrap>`과 `wrap="true"`. `wrap="false"`는
  에러다 — 줄바꿈 없음이 CSS 기본값이라 끄는 표기는 속성을 지우는 것이고, 기본값을 명시하는
  표기를 열면 저자가 두 곳을 봐야 한다. `<grid wrap>`은 지금 통과하고 `flex-wrap: wrap`이
  그대로 나가지만 grid 컨테이너에서 CSS가 무시한다 — 이름 층에서 막을 자리다 (알려진 구멍).
- **`pad`는 CSS `padding`과 같은 1~4값 축약이다** (상 · 상하/좌우 · 상/좌우/하 ·
  상/우/하/좌). 산출 CSS는 항상 네 변을 명시한다 — 축약 규칙이 엔진의 두 곳에 생기지
  않게 값 층이 한 번 푼다.
- 수치는 **단위 없는 논리 스테이지 px**다 (PRD §13-5). `px`·`pt`·`em` 접미사는 에러이고
  `%`만 예외다. 음수 길이도 에러다 — CSS가 무효로 버리는 선언은 조용히 사라지므로 값 층이
  막는다.

## 자식 속성

부모 컨테이너 안에서 **자기 자리**를 정하는 채널이므로 컨테이너가 아니라 자식에 붙는다.

| 속성 | 허용 값 | CSS 대응 |
|---|---|---|
| `grow` | 0 이상 정수 | `flex-grow` |
| `shrink` | 0 이상 정수 | `flex-shrink` |
| `basis` | 수치 · `%` | `flex-basis` |
| `span` | 1 이상 정수 | `grid-column: span N` |
| `self` | `start` `center` `end` `stretch` `baseline` | `align-self` |
| `order` | 정수 (음수 허용) | `order` |
| `at` | 9지점 앵커 (`top-left`…`bottom-right`) | `place-self` ([`<stack>`](stack.md)) |
| `x` / `y` | 수치 · `%` · **음수** | `position: absolute; left` / `top` ([`<stack>`](stack.md)) |
| `style` | 배치 CSS 선언 목록 | 인라인 `style`의 **맨 끝** (아래 절) |

**적용 대상** — 레이아웃 컨테이너의 자식이 될 수 있는 요소 전부다: `<text>`, 블록 태그,
리스트 컨테이너(`ul`/`ol`), 레이아웃 컨테이너(중첩), [`<shape>`](shape.md), `<spacer>`.

- **`<li>`와 인라인 태그(`strong`/`em`/`code`/`a`/`span`/`br`)에는 없다.** `<li>`는 리스트
  컨테이너의 자식이지 flex 항목이 아니고 (PPTX에서 `<ul>` 하나가 도형 하나이므로 항목의
  배치를 좌표로 옮길 자리가 없다), 인라인은 박스가 아니라 런이다. 열어 두면 `<li grow="1">`
  이 웹에서만 뜻을 갖는 선언이 된다.
- **`span`·`at`·`x`/`y`는 부모가 한정한다.** 셋 다 특정 컨테이너에서만 뜻이 있는 배치
  채널이라, 엉뚱한 부모에서 쓰면 CSS가 조용히 버리는 선언이 된다 — 그 침묵이 닫힌 어휘가
  막으려는 실패다. 부모를 아는 자리가 값 층(core)의 순회뿐이라 검증도 거기 있고, 진단은
  속성 **이름**을 가리키며 고칠 방법(태그를 바꾸거나 `<stack>`으로 감싸기)을 담는다:

  | 속성 | 뜻이 있는 부모 | 밖에서 쓰면 |
  |---|---|---|
  | `span` | `<grid>`의 자식 | `span은 <grid>의 자식에서만 쓴다 (부모는 <row>다) — flex 컨테이너에서 grid-column은 무효다` |
  | `at` | [`<stack>`](stack.md)의 자식 | `at은 <stack>의 자식에서만 쓴다 (…) — 겹침이 필요하면 <stack>으로 감싼다` |
  | `x` / `y` | `<stack>`의 자식 · `layout="canvas"` 슬라이드의 직속 자식 | `x는 <stack>의 자식이나 layout="canvas" 슬라이드의 직속 자식에서만 쓴다 (…) — <stack>으로 감싸거나 슬라이드에 layout="canvas"를 준다` |

  `at`과 `x`/`y`는 같은 자식에 **함께 쓸 수 없다** — 둘 다 자리를 정하는 채널이라 하나만
  쓴다 ([`<stack>`](stack.md)). 9지점 앵커 어휘·좌표 문법(음수·`%`·기준 박스)의 전모는
  겹침 페이지에 있다.
- `basis`는 `fill`을 받지 않는다 — `flex-basis: 100%`는 주축에서만 크기가 되어 교차축에서
  저자의 기대와 갈라진다. 백분율이 필요하면 `basis="100%"`로 적는다.
- `order`만 음수를 받는다 — CSS `order`의 음수는 유효하고 "먼저 보이게"의 관용 표기다.
  **문서 순서와 시각 순서가 갈린다는 뜻이고, PPTX에서 도형 순서(= 문서 순서)와 좌표(=
  시각 순서)가 그대로 갈라진다.**
- 배치 속성은 [캐스케이드](../appearance.md)를 타지 않는다 — `variant`는 외관 번들이고
  배치를 담지 않는다. 배치를 테마가 소유하는 자리는 1층 레이아웃(Iteration 10)이다.

## 공간 토큰

`gap`과 `pad`의 값 어휘다 (PRD §8.4의 `space` 스케일) — **배치조차 테마의 리듬을
따라간다**. 내장 `minimal` 테마의 사다리:

| 토큰 | 값(px) | 근거 |
|---|---|---|
| `xs` | 6 | ¼ md — 리듬의 최소 눈금 |
| `sm` | 12 | ½ md |
| `md` | 24 | 본문 한 글자 (= 웹 렌더러의 기본 스택 간격) |
| `lg` | 36 | 본문 줄박스 한 줄 (24 × 1.5) |
| `xl` | 48 | 2 md |
| `2xl` | 72 | 줄박스 두 줄 |

- **토큰과 수치는 산출물이 다르다.** `gap="md"`는 웹에서
  `gap: var(--balpyo-space-md)`가 되어 테마를 바꾸면 여백이 따라 움직이고, `gap="24"`는
  `gap: 24px`로 굳는다 — 같은 24px이지만 테마 참조인지 저자가 정한 리터럴인지가 값에
  남는다 (색 토큰이 `schemeClr` 참조로 남는 것과 같은 구조).
- **PPTX에는 토큰이 남지 않는다.** OOXML 테마 파트에 공간 개념이 없고 배치는 좌표로
  구워지므로, PowerPoint에서 `gap`을 고친다는 것은 도형을 옮긴다는 뜻이다 — 모든 색
  토큰이 clrScheme 슬롯을 갖는 계약의 **유일한 예외**다
  ([ADR 0012](../../decisions/0012-space-tokens-have-no-theme-slot.md)).
- 크기 속성(`w`/`h`/`min-*`/`max-*`)에는 공간 토큰을 쓸 수 없다 — 여백의 사다리로 박스
  크기를 정하면 리듬의 뜻이 흐려진다. 진단이 그 사실을 짚어 준다.

## 예제

`testdata/fixtures/03_layout/slides/001_row_col.slide` — 중첩 컨테이너:

```html
<slide id="row-col">
  <h2>두 열로 나눈 회고</h2>
  <row gap="xl" align="start" justify="between">
    <col gap="md">
      <h3>잘된 것</h3>
      <ul>
        <li>모놀리스 분리 완료</li>
        <li>배포 파이프라인 <strong>자동화</strong></li>
      </ul>
    </col>
    <col gap="md">
      <h3>아쉬운 것</h3>
      <ul>
        <li>관측성 도입이 늦었다</li>
        <li>온콜 문서가 흩어져 있다</li>
      </ul>
    </col>
  </row>
  <note>왼쪽 2분, 오른쪽 1분.</note>
</slide>
```

`002_grid_cards.slide` — 카드 그리드 (한 줄이 좌표 계산 전부를 대체한다):

```html
<slide id="grid-cards">
  <h2>플랫폼 지표</h2>
  <grid cols="4" gap="lg">
    <col gap="xs">
      <h3>8분</h3>
      <p>배포 소요 시간</p>
    </col>
    <!-- … 카드 셋 … -->
    <col gap="xs" span="2">
      <h3>두 칸 카드</h3>
      <p>span="2"는 grid-column: span 2다.</p>
    </col>
  </grid>
</slide>
```

`003_spacer.slide` — 밀어내기:

```html
<slide id="spacer">
  <h2>여백이 배치다</h2>
  <spacer/>
  <p>가운데 문단은 위아래 spacer가 밀어낸 자리에 놓인다.</p>
  <spacer grow="2"/>
  <row gap="sm" align="center">
    <p>왼쪽 꼬리글</p>
    <spacer/>
    <p>오른쪽 페이지 번호</p>
  </row>
</slide>
```

`005_sizing.slide` — 크기·비율·패딩·줄바꿈:

```html
<slide id="sizing">
  <h2>크기와 비율</h2>
  <row gap="md" pad="lg" w="fill" min-h="120" align="center">
    <col w="40%" gap="xs">
      <p>w=40% — 비율 폭</p>
    </col>
    <col w="320" max-w="360" gap="xs">
      <p>w=320 · max-w=360</p>
    </col>
    <col ratio="16:9" min-w="200" max-h="220" gap="xs">
      <p>ratio=16:9</p>
    </col>
  </row>
</slide>
```

## 렌더 결과

### 웹

컨테이너는 `<div>` 하나다 — 클래스는 흐름에서 나오고(`balpyo-row`/`balpyo-col`/
`balpyo-grid`/`balpyo-spacer`), 확정된 배치 값은 인라인 `style`로 나간다:

```html
<div class="balpyo-row" data-balpyo-id="s0.1" data-balpyo-el="row"
     style="gap: var(--balpyo-space-xl); align-items: start; justify-content: space-between">
```

- **엔진 CSS와 인라인 style의 경계**는 "태그의 성질인가, 요소마다 다른 값인가"다.
  `display`·`flex-direction`(태그의 정의 그 자체)과 `<spacer>`의 `flex: 1`만 클래스에
  있고, 나머지 전부(gap·정렬·wrap·pad·트랙·크기 6종·ratio + 자식 6종 + `style` 탈출구)는
  인라인 선언이다 — 인라인은 cascade layer 밖에서 이기므로 저자 CSS가 확정 배치를 덮지
  못한다 ([ADR 0003](../../decisions/0003-inline-style-appearance.md)의 연장).
- `justify="between"` → `justify-content: space-between`. `space-*` 접두를 붙이는 것이
  렌더러의 몫이고, `align`/`self`의 논리 이름(`start`/`end`)은 CSS에서 그대로 통한다.
- `cols="4"` → `grid-template-columns: repeat(4, 1fr)`, `w="fill"` → `width: 100%`,
  `pad="lg"` → 네 변 모두 `var(--balpyo-space-lg)`.
- **컨테이너 안의 텍스트·리스트는 브라우저 기본 폭으로 돌아간다.** 슬라이드의 세로
  스택은 텍스트에 `width: 100%`와 `flex: 0 0 auto`를 주어 줄바꿈을 결정적으로 만드는데,
  그 규칙이 가로 컨테이너 안까지 따라오면 자식마다 부모 폭 전체를 요구해 `<spacer>`가
  받을 남은 공간이 사라진다 (실측으로 드러난 사실이다). 컨테이너 안에서 폭을 정하는 것은
  컨테이너다 — 트랙, 교차축 정렬, 자식의 `basis`/`grow`.
- 컨테이너와 `<spacer>`도 지오메트리 식별자(`data-balpyo-id`)를 받는다 — 진짜 박스이므로
  rect에 뜻이 있다 ([ADR 0013](../../decisions/0013-measured-kinds-include-layout.md)).

### PPTX

**레이아웃 태그는 PPTX에서 사라진다.** 자식이 절대 좌표 도형이 되고 컨테이너 자체는
도형이 아니다 — 컨테이너의 rect는 측정되지만 어떤 파트도 그것을 소비하지 않는다
([ADR 0011](../../decisions/0011-layout-containers-are-not-shapes.md)).

| 태그 | PPTX 산출 |
|---|---|
| `<row>` `<col>` `<grid>` | 도형 0개 — 조립기가 자식으로 내려간다 |
| `<spacer/>` | 도형 0개 — 배치 수단이므로 산출물에 남을 것이 없다 |
| `<ul>` `<ol>` | 도형 **하나** (불릿 문단이 한 텍스트 박스에 산다) |
| [`<shape>`](shape.md) | 도형 **하나** — 그리고 **자식 텍스트를 흡수한다** (컨테이너를 담으면 배경이 되고 자식이 형제 도형이 된다) |
| `<group>` | `grpSp` — 편집 단위로 묶인 도형 (Iteration 9, 아직 어휘 밖) |

"컨테이너"라는 같은 단어가 정반대로 매핑되는 것이 이 표의 요점이다: 리스트는 하나의
텍스트 흐름이라 도형 하나가 편집 단위이고, 레이아웃은 배치 **수단**이라 좌표로 구워진 뒤
남을 것이 없다. `<shape>`는 셋 중 어느 쪽도 아니다 — **남으면서 자식을 흡수한다**
(컨테이너가 사라지면서 자식을 도형으로 만드는 것과 정확히 반대다).
`<group>`은 네 번째다 — 그룹은 사라지지 않고 묶는다.

- **자식 좌표는 컨테이너 상대가 아니다.** 브라우저가 잰 rect가 이미 슬라이드 상대이므로
  중첩 깊이가 좌표에 영향을 주지 않는다 — 조립기에 컨테이너 좌표를 더하는 코드가 없는
  것이 "좌표 계산 0회"의 렌더러 쪽 절반이다.
- 도형 id는 **도형을 실제로 낸 자리에서만** 오른다 — 슬라이드에 `<row>` 하나를 더한
  커밋이 도형 번호 전부를 밀지 않는다.
- `<spacer/>`는 도형 0개이지만 **빈 `<text></text>`는 도형 하나**다. 기준은 저자의
  콘텐츠인가 배치 수단인가다: 빈 텍스트 박스는 저자가 PowerPoint에서 자리를 확인할 수
  있어야 하고, spacer의 뜻("여기에 공간이 있다")은 이미 이웃 도형의 좌표에 들어가 있다.
- 그래서 도형 수는 측정 박스 수에서 배치·리스트 항목·**도형 안 텍스트**를 뺀 값이다.
  `03_layout` 픽스처 7장의 실측: **도형 47개 = 측정 박스 78 − 컨테이너 21 − spacer 4 −
  `<li>` 6**. 도형 이름에 `row`/`col`/`grid`/`spacer`가 하나도 없는 것을 픽스처 테스트가
  감시한다. `04_shape` 픽스처에서 항이 하나 늘었다 — 도형 안 텍스트는 재지만 부모 도형의
  본문이 된다 ([`<shape>`](shape.md)).
- `<grid cols="4" gap="lg">`의 카드 4개는 산출물에서 **같은 y(EMU가 정확히 같다)**와
  등간격 x를 갖는다 (트랙 261px + gap 36px = pitch 297px). 그리드가 그리드로 구워지는지를
  좌표로 단정하는 테스트가 있다.

## `style` 탈출구

레이아웃 속성이 못 덮는 배치는 `style`로 통과시킨다 (PRD §7.9). 배치는 어차피 좌표로
구워지므로 PPTX 충실도를 해치지 않는다 — 그래서 **배치 전용**이다.

계약은 "우리가 보지 않는다"이지 "안전하다"가 아니다
([ADR 0014](../../decisions/0014-style-escape-hatch-boundary.md)):

1. 선언을 `프로퍼티: 값`으로 쪼개 정규화한다 (이름은 소문자로, 마지막 `;`와 공백은
   흡수한다). **값은 검증하지 않는다** — 엔진이 CSS 값 문법을 재구현하면 탈출구가
   탈출구이기를 멈춘다.
2. 프로퍼티 이름만 배치 allowlist와 대조한다. 목록 밖은 **경고**이고 빌드는 계속된다.
3. 문법 위반(콜론 없음, 이름 없음, 값 없음)은 **에러**이고, 하나라도 있으면 `style`
   **전체**가 실리지 않는다 — 반쯤 통과한 선언 목록은 저자가 고친 뒤의 결과를 예측할 수
   없게 만든다.

**배치 allowlist** (구현의 목록 그대로 — `*`는 접두 가족이다):

```
display
flex, flex-*
grid, grid-*
gap, row-gap, column-gap
padding, padding-*
margin, margin-*
width, min-width, max-width
height, min-height, max-height
box-sizing
align-*, justify-*, place-*
order
aspect-ratio
position, inset, inset-*, top, right, bottom, left
translate
```

목록의 기준은 하나다: **지오메트리 패스가 재는 박스를 움직이는가.** `translate`가 있고
`rotate`/`scale`이 없는 것이 그 기준의 시험이다 — `getBoundingClientRect`는 이동한 박스를
재지만 회전·확대는 축에 정렬된 외접 사각형만 남겨 PPTX에서 다른 도형이 된다.

탈출구 선언은 인라인 `style`의 **맨 끝**에 붙는다 — 같은 프로퍼티면 뒤가 이기므로 저자가
엔진 선언을 덮는다. 그것이 탈출구의 뜻이다.

`006_style.slide` — 배치 선언만 쓴 슬라이드 (경고 없음):

```html
<row gap="md" style="align-content: space-evenly">
  <p style="margin-inline-start: 40px">배치 선언은 경고 없이 통과한다.</p>
  <col style="place-content: end">
    <p>레이아웃 속성으로 못 적는 배치가 여기로 온다.</p>
  </col>
</row>
<p style="translate: 0 12px">블록에도 배치 탈출구가 열려 있다.</p>
```

`007_style_appearance.slide` — 외관 선언을 쓰면 웹 전용이 된다:

```html
<row gap="md" style="background: #0F65F9; padding-inline: 24px">
  <p>배경색은 PPTX로 넘어가지 않는다 — 경고가 붙는다.</p>
  <p style="border: 2px solid red">테두리도 외관이다.</p>
</row>
```

```
slides/007_style_appearance.slide:8:24: warning: style의 "background" 선언은 배치가 아니다 — 웹에만 적용되고 PPTX에는 이식되지 않는다 (PRD §7.9). 색·테두리 같은 외관은 컴포넌트 속성으로 쓴다
```

경고인 이유는 에러로 두면 엔진이 CSS 프로퍼티 전체를 알아야 하고, 그것이 탈출구의 존재
이유와 모순되기 때문이다. 모르는 `variant` 경고와 같은 경로를 타므로 `--strict`가 에러로
승격한다 ([CLI](../cli.md)). 이렇게 웹 전용이 된 외관은 웹↔PPTX 픽셀 비교에서 당연히
갈라지므로, 픽스처 테스트가 그 슬라이드에만 오차 조정을 준다 (조정의 범위는 근거의 범위와
같다 — [테스트 환경](../../internals/testing.md)).

## 빌드 에러가 나는 조건

파서(`soml.Parse`) — 이름·배치·문법:

| 조건 | 진단 메시지 |
|---|---|
| `<spacer>`에 자식·문자 데이터를 썼다 | `void 태그 <spacer>은 내용을 가질 수 없다 (<spacer/>로 쓴다)` |
| 컨테이너 안에 문자 데이터를 직접 썼다 | `<row> 안에 문자 데이터를 직접 쓸 수 없다 (<text>로 감싼다)` |
| 어휘 밖의 자식 태그 | `<row>의 자식으로 <note>는 올 수 없다 (쓸 수 있는 태그: <text>, <shape>, <image>, <h1>, <h2>, <h3>, <p>, <blockquote>, <pre>, <ul>, <ol>, <row>, <col>, <grid>, <spacer>)` |
| flex 컨테이너에 트랙을 썼다 | `<row>에 알 수 없는 속성 cols (쓸 수 있는 속성: gap, align, justify, wrap, pad, w, h, min-w, min-h, max-w, max-h, ratio, grow, shrink, basis, span, self, order, style)` |
| `<spacer>`에 컨테이너 속성을 썼다 | `<spacer>에 알 수 없는 속성 gap (쓸 수 있는 속성: grow, shrink, basis, span, self, order, style)` |
| `<li>`에 배치 속성을 썼다 | `<li>에는 속성을 쓸 수 없다 (grow)` |
| 인라인 태그에 배치 속성을 썼다 | `<strong>에는 속성을 쓸 수 없다 (order)` |
| 블록·`<text>`·`<li>` 안에 컨테이너를 썼다 | `<p> 안에는 문자 데이터와 인라인 태그만 올 수 있다 (<row>는 쓸 수 없다)` |
| `<ul>`/`<ol>`의 자식에 컨테이너를 썼다 | `<ul>의 자식으로 <row>는 올 수 없다 (쓸 수 있는 태그: <li>)` |
| `<stack>`에 `gap`·`wrap`·`cols`·`rows`를 썼다 | `<stack>에 알 수 없는 속성 gap (쓸 수 있는 속성: align, justify, pad, w, h, min-w, min-h, max-w, max-h, ratio, grow, shrink, basis, span, self, order, at, x, y, style)` |

값 해석(`core.Resolve`) — 값 어휘는 이 층이 소유한다. 진단은 문제가 된 **조각**을
가리킨다 (`pad="sm huge"`의 진단은 `huge`를 가리킨다):

| 조건 | 진단 메시지 |
|---|---|
| 모르는 공간 토큰 | `알 수 없는 공간 토큰 "huge" (쓸 수 있는 토큰: xs, sm, md, lg, xl, 2xl, 또는 수치·백분율)` |
| 단위 접미사를 붙였다 | `gap는 단위 없는 논리 px 수치다 ("16px"의 "px"를 뗀다) — px·pt·em 접미사는 쓸 수 없다 (PRD §13-5)` |
| 음수 길이 | `gap는 0보다 작을 수 없다 ("-4")` |
| 크기에 공간 토큰을 썼다 | `공간 토큰 "md"는 gap·pad에만 쓴다 — w는 수치·백분율·fill이다` |
| 어휘 밖의 `align`/`self` | `알 수 없는 align "centre" (쓸 수 있는 값: start, center, end, stretch, baseline)` |
| CSS 접두를 붙였다 | `align는 접두 없는 이름을 쓴다 — "flex-start" 대신 "start"` |
| 어휘 밖의 `justify` | `알 수 없는 justify "middle" (쓸 수 있는 값: start, center, end, between, around, evenly)` |
| `justify`에 `space-` 접두를 붙였다 | `justify는 접두 없는 이름을 쓴다 — "space-between" 대신 "between"` |
| `wrap="false"` | `wrap은 값 없는 속성(<row wrap>)이거나 wrap="true"다 ("false") — 줄바꿈 없음이 기본값이므로 끄려면 속성을 지운다` |
| `pad`에 값이 5개 이상 | `pad는 1~4개 값이다 ("1 2 3 4 5"는 5개) — CSS padding의 상·우·하·좌 축약과 같은 규칙이다` |
| 트랙 개수가 0 이하 | `cols의 트랙 개수는 1 이상 정수다 ("0")` |
| 어휘 밖의 트랙 값 | `cols의 트랙 값 "넷"를 알 수 없다 (쓸 수 있는 형태: 수치·Nfr·N%·auto, 또는 트랙 개수 정수 하나)` |
| 잘못된 `ratio` | `ratio는 W:H 정수 쌍이다 ("16") — 예: 16:9` / `ratio의 W와 H는 0보다 커야 한다 ("0:9")` |
| `grow`/`shrink`가 정수가 아니다 | `grow는 정수다 ("1.5")` / `grow는 0보다 작을 수 없다 ("-1")` |
| `span`이 1보다 작다 | `span은 1 이상 정수다 ("0")` |
| `span`을 grid 밖에서 썼다 | `span은 <grid>의 자식에서만 쓴다 (부모는 <row>다) — flex 컨테이너에서 grid-column은 무효다` |
| `at`을 stack 밖에서 썼다 | `at은 <stack>의 자식에서만 쓴다 (부모는 <row>다) — 겹침이 필요하면 <stack>으로 감싼다` |
| `x`/`y`를 유효하지 않은 부모에서 썼다 | `x는 <stack>의 자식이나 layout="canvas" 슬라이드의 직속 자식에서만 쓴다 (…) — <stack>으로 감싸거나 슬라이드에 layout="canvas"를 준다` |
| `at`과 `x`/`y`를 함께 썼다 | `at과 x를 함께 쓸 수 없다 (x는 …에 있다) — 둘 다 자리를 정하는 채널이라 하나만 쓴다 (at은 9지점 앵커, x/y는 절대 오프셋)` |
| `style` 선언에 콜론이 없다 | `style 선언 "width 100px"에 콜론이 없다 — \`속성: 값\` 형태로 쓴다` |
| `style` 선언에 값이 없다 | `style 선언 "width:"에 값이 없다` |
| 배치 속성에 값이 없다 | `w 속성에는 값이 필요하다 (w="fill")` / `gap 속성이 비어 있다 (지우면 기본값을 쓴다)` |

`style`의 allowlist 밖 프로퍼티는 **경고**다 (위 탈출구 절) — 에러가 아니므로 기본 빌드는
통과하고 `--strict`가 승격한다.
