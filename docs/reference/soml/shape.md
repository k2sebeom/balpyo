# `<shape>` — 프리셋 도형 12종

도형 하나다. `type`이 **어떤 프리셋 지오메트리인가**를 정하고, 외관 속성이 채움·테두리·
모서리·그림자·투명도·변형을 정하며, 자식으로 텍스트나 **레이아웃 컨테이너**를 담을 수 있다.

이 페이지의 중심은 아래 **12종 표**다. 어휘를 12종으로 닫은 기준은 하나다: **OOXML 프리셋
지오메트리에 대응이 있는 것만** 넣는다. 그래서 도형은 PPTX에서 그림이 아니라
`prstGeom` 도형이 되고, PowerPoint에서 "도형 모양 변경"으로 다른 프리셋이 될 수 있다.

## `type` 12종

| SOML `type` | 웹에서 그려지는 것 | OOXML `prstGeom@prst` |
|---|---|---|
| `rect` | SVG `<rect>` (px 공간) | `rect` |
| `rounded-rect` | SVG `<rect rx>` (px 공간) | `roundRect` |
| `ellipse` | SVG `<ellipse>` (px 공간) | `ellipse` |
| `diamond` | SVG `<path>` (정규 공간) | `diamond` |
| `cylinder` | SVG `<path>` (정규 공간) | **`can`** |
| `chevron` | SVG `<path>` (정규 공간) | `chevron` |
| `arrow-right` | SVG `<path>` (정규 공간) | **`rightArrow`** |
| `hexagon` | SVG `<path>` (정규 공간) | `hexagon` |
| `parallelogram` | SVG `<path>` (정규 공간) | `parallelogram` |
| `callout` | SVG `<path>` (정규 공간) | **`wedgeRectCallout`** |
| `star` | SVG `<path>` (정규 공간) | **`star5`** |
| `cloud` | SVG `<path>` (정규 공간) | `cloud` |

굵게 표시한 넷은 **이름이 갈라지는 행**이다 — SOML 이름은 저자가 찾을 만한 말이고
(`cylinder`·`arrow-right`), OOXML 이름은 규격이 정한 것이다 (`can`·`rightArrow`).
직접 `prstGeom` 이름을 쓰게 하지 않는 이유는 어휘가 규격의 190여 종으로 열려 웹에서 그릴 수
없는 프리셋이 섞이기 때문이다.

- `type`은 **생략할 수 없다**. `<shape/>`가 조용히 사각형이 되면 저자의 오타
  (`type="elipse"`)와 구별되지 않는다. `<p>` ≡ `<text role="body">`처럼 언어가 정한 등가가
  도형에는 없다 ([텍스트 서브셋](text-subset.md)과 갈라지는 자리).
- **기하의 출처는 OOXML 프리셋 정의다** (ECMA-376의 `presetShapeDefinitions`). 웹의 SVG 경로도
  그 정의에서 유도하므로 두 타깃의 실루엣이 같은 출처에서 나온다.
- **정규 공간의 아홉 프리셋은 정사각형에서 평가한 경로를 도형 박스로 늘린다.** 그래서
  정사각형이 아닌 박스에서 노치·모서리 비율이 PowerPoint의 기본 모양과 다를 수 있고, **PPTX가
  `avLst` 조정값으로 웹을 맞춘다**.

## 속성

### 고유 속성

| 속성 | 값 | 기본값 | 뜻 |
|---|---|---|---|
| `type` | 위 표의 12종 | **필수** | 프리셋 지오메트리 |
| `id` | 임의 이름 | 없음 | 도형 이름 — **아직 아무것도 그리지 않는다** (아래) |
| `alt` | 문자열 (빈 값은 **에러**) | 없음 | 그림의 **접근성 이름** — 안의 글을 대체하지 않는다 (아래) |

`id`는 어휘에 있고 중복 검사도 받지만 **산출물에 나가지 않는다**. `id`를 참조하는
어휘가 아직 없고, 그 전에 `id`를 PPTX 도형 이름이나 웹 속성에 실으면 두 타깃의 식별자 규칙이
미리 갈라진다 — 그래서 지금은 "이 슬라이드에서 유일한
이름"이라는 계약만 지킨다. 같은 슬라이드에 같은 `id`가 둘이면 에러다 (범위가 슬라이드인
이유: 참조가 슬라이드를 넘지 않는다).

`alt`은 **그림에 이름을 더하고 안의 글을 대체하지 않는다** — 이름 붙은 도형의 `<text>`·`<p>`는
두 타깃 모두 읽히는 글로 남는다. 붙는 자리와 빈 값이 에러인 이유는
[외관 속성의 접근성 절](../appearance.md)이 소유한다 (표·그림의 `alt`과 한 규칙이다).
`alt`은 캐스케이드 밖이라 `variant`나 테마가 이름을 물려주지 않는다: 이름은 이 도형이 이
덱에서 무엇인가라 물려받을 수 있는 값이 아니다.

### 외관 속성 8종

값 어휘와 캐스케이드는 [외관 속성](../appearance.md)이 소유한다. 여기서는 도형에서의 뜻만
적는다.

| 속성 | 값 | 도형에서의 뜻 |
|---|---|---|
| `variant` | 테마가 정의한 이름 | 외관 번들 — **도형 어휘와 텍스트 어휘는 별개 표다** |
| `fill` | 색 토큰 · `#RRGGBB` · `none` | 면 채움 |
| `border` | `<굵기> <스타일> <색>` · `none` | 외곽선 — 경로 **중앙**에 그려진다 (아래) |
| `radius` | radius 토큰 · 수치 | 모서리 반경 — **`type="rounded-rect"` 전용** |
| `shadow` | shadow 토큰 | 그림자 — 수치는 받지 않는다 |
| `opacity` | 0~1 실수 | 채움·테두리의 투명도 — **글자는 흐려지지 않는다** |
| `rotate` | 각도 수치 (음수 허용) | 시계방향 회전 — 글자까지 함께 돈다 |
| `flip` | `h` `v` `hv` | 대칭 — **글자는 뒤집히지 않는다** |

도형은 `variant`가 없어도 **외관이 확정된다**: 테마의 컴포넌트 기본값이 캐스케이드의 바닥에
있어 `<shape type="rect"/>`도 `fill: surface` + `border: 1 solid muted` + `radius: md`를
받는다. 도형에는 텍스트의 `role`에 해당하는 단이 없으므로 그 자리를 컴포넌트
기본값이 대신한다.

- **`radius`는 `rounded-rect`에서만 쓴다.** 다른 프리셋에는 반경 조정값을 실을 자리가 없어
  **웹만 둥글어지고 PPTX는 각진 채** 남는다 — 그 조용한 갈라짐을 값 층이 에러로 막는다.
- **`shadow`는 토큰만 받는다.** CSS 축약 4수치와 OOXML `outerShdw`가 1:1이 아니라 같은
  수치가 두 타깃에서 다르게 그려진다.
- **`opacity`는 합성 불투명도가 아니라 색의 알파로 컴파일된다.** 그래서 반투명 도형 안의
  글자는 또렷하다.
- **테두리는 굵기의 절반이 도형 박스 밖으로 번진다** — 양 타깃 모두 외곽선 중앙 규약이므로
  (SVG `stroke`, OOXML `a:ln`) 같은 자리에 같은 만큼 번진다. 보정하지 않는 것이 두 타깃을
  맞추는 길이다.

### 자기 박스 — 여백과 크기

| 속성 | 값 | CSS 대응 |
|---|---|---|
| `pad` | 공간 토큰 · 수치 (1~4값) | `padding` — 도형 안 텍스트의 안쪽 여백 |
| `w` / `h` | 수치 · `%` · `fill` | `width` / `height` |
| `min-w` / `min-h` · `max-w` / `max-h` | 수치 · `%` · `fill` | `min-*` / `max-*` |
| `ratio` | `W:H` 정수 쌍 | `aspect-ratio` |

**이 목록은 [레이아웃 컨테이너](layout.md)의 크기·여백 속성과 같은 한 벌이다** — 크기는
컨테이너의 성질이 아니라 박스의 성질이므로 두 요소가 같은 표를 공유한다. 그래서
`<row w="fill">`와 `<shape w="fill">`의 값 어휘·진단이 한 글자도 다르지 않다.

- **도형에는 크기를 적는 것이 보통이다.** 도형은 그 자체가 그림이라 내용이 없으면 잰 크기가
  0이고, 0×0 도형은 산출물에서 아무것도 아니다 — 빈 `<text></text>`가 텍스트 박스를 내는
  것과 갈라지는 자리다 (그쪽은 저자가 자리를 확인해야 하는 콘텐츠다).
- `pad`에 **백분율은 쓸 수 없다.** CSS 백분율 padding은 부모의 **폭** 기준인데 OOXML 인셋은
  절대 EMU라, 잰 도형 rect만으로는 웹이 쓴 값을 복원할 수 없다 — 근사해서 내면 글자 위치가
  조용히 어긋난다. 진단은 PPTX 조립 단계에서 난다 (`도형의 pad는 백분율을 쓸 수 없다 (…%) —
  OOXML bodyPr 인셋은 절대 EMU이고 CSS 백분율 padding은 부모 폭 기준이다`) — 컨테이너의
  `pad`는 백분율을 받는다 (컨테이너의 여백은 좌표로 구워지고 인셋이 되지 않는다).

### 자식 배치 속성

도형도 컨테이너의 자식이 될 수 있으므로 `grow`/`shrink`/`basis`/`span`/`self`/`order`/`style`을
받는다 ([레이아웃 태그](layout.md)의 자식 속성 표와 같은 한 벌). `<grid cols="4">` 안의
`<shape span="2">`가 두 칸을 차지한다.

## 콘텐츠 모델 — 두 가지 모드

직속 자식 어휘는 **`<text>`와 블록 태그**(`h1`~`h3`, `p`, `blockquote`, `pre`) **그리고 레이아웃
컨테이너**(`row`, `col`, `grid`, `stack`)다. 개수 제한은 없다.

**컨테이너를 담았는지가 도형의 모드를 정한다**:

| 직속 자식 | 모드 | 도형이 되는 것 |
|---|---|---|
| `<text>`·블록 태그만 | **흡수** | 도형 1개 — 글자는 그 도형의 문단이 된다 |
| 컨테이너가 하나라도 있다 | **배경** | 도형 1개 + **자식마다 형제 도형** |

```html
<!-- 흡수 모드 — 자식 여럿은 문단 여럿이다. -->
<shape type="rounded-rect" fill="surface" border="1 solid muted" w="400" h="140" pad="lg">
  <h3>문단 둘</h3>
  <p>도형 하나에 자식 여럿은 문단 여럿이다.</p>
</shape>

<!-- 배경 모드 — 카드 한 장이 도형 하나다 (헤더 띠·본문·푸터가 그 안에서 배치된다). -->
<shape type="rounded-rect" fill="surface" border="1 solid muted" radius="md" w="fill" h="220" pad="md">
  <col gap="sm" h="fill">
    <shape type="rounded-rect" fill="accent1" w="fill" h="48" pad="xs">
      <text color="bg" align="center">선언</text>
    </shape>
    <p size="20">헤더 띠는 컨테이너를 거쳐 들어온 중첩 도형이고, 자식이 텍스트뿐이라 흡수
    모드로 남는다 — 모드는 도형마다 정해진다.</p>
  </col>
</shape>
```

- **모드는 도형마다 정해진다.** 배경 도형 안의 도형이 또 배경일 수도, 흡수일 수도 있다 —
  판정은 그 도형의 직속 자식만 본다 (손자 컨테이너는 모드를 뒤집지 않는다).
- **리스트(`<ul>`/`<ol>`)·[`<image>`](image.md)·[`<table>`](table.md)·중첩 도형은 컨테이너를
  거쳐 들어온다.** 직속 자식은 아니지만 `<col>` 안에서는 이미 올 수 있으므로 "도형 안에
  리스트를 넣는다"가 새 매핑 없이 성립한다: 리스트는 컨테이너의 자식으로서 오늘의 `p:sp`
  하나(불릿 문단들)가 되고, 중첩 도형은 자기 프리셋을 가진 형제 도형이 된다. 직속 자식으로
  막아 둔 이유는 어휘를 한 자리에 모으는 것이다 (컨테이너가 배치를 소유한다).
- **배경 모드 도형은 `rotate`를 쓸 수 없다** (아래 진단 표). PPTX에서 자식은 형제 도형이 되어
  도형의 회전을 물려받지 않는데 웹은 박스를 돌려 자식까지 돌리므로, 같은 SOML이 두 타깃에서
  다른 그림이 된다.
- **`anchor`가 뜻을 갖는 유일한 자리가 도형 안이다** — 세로 앵커는 [`<text>`](text.md)의
  속성이고, 슬라이드 직속 텍스트 박스는 잰 rect가 곧 내용 크기라 세로로 움직일 여백이 없다.
- 흡수 모드에서 도형의 앵커는 **첫 자식이 정한다.** 앵커는 자식마다 확정되지만 박스의 성질이라
  도형 하나에 하나뿐이다 (웹은 flex `justify-content` 하나, PPTX는 `bodyPr@anchor` 하나) — 뒤
  자식의 다른 앵커는 조용히 버려진다.
- `pad`와 `anchor`는 함께 배치를 만든다: `pad`가 안쪽 여백을 만들고 그 안에서 `anchor`가
  세로 자리를 정한다.
- **배경 모드에서 자식의 세로 자리는 컨테이너가 정한다** — 도형의 `pad` 안에서 `<col>`의
  `justify`/`gap`이 배치이고, 도형 자신의 `anchor`는 빈 본문에만 남는다 (담을 글자가 없다).
- 그래서 **세로 자리를 부르는 이름이 두 개다**: 도형·텍스트 박스 안은 `anchor`, 컨테이너는
  `justify`/`align`이다. `<shape justify="center">`와 `<col anchor="center">`는 둘 다 "알 수
  없는 속성" 에러이고, 어느 이름이 어디에 붙는지와 이름이 갈린 이유(CSS flex 대 OOXML
  `bodyPr@anchor`)는 [레이아웃](layout.md)에 한 번만 적혀 있다.

### 프리셋별 텍스트 영역

**글은 박스 전체가 아니라 프리셋이 정한 사각형 안에 흐른다.** 타원 안의 글이 곡면을 넘지
않는 것이 그 뜻이다. 출처는 실루엣과 같다 — 프리셋 정의의 `<rect>`이고, PowerPoint가 같은
도형에 글을 담을 때 쓰는 그 사각형이다.

| `type` | 텍스트 영역 (박스에 대한 인셋) |
|---|---|
| `rect` | 박스 전체 |
| `rounded-rect` | 네 변 **`0.2929 × radius` px** (모서리 원호에 내접) |
| `ellipse` | 네 변 14.64% |
| `diamond` | 네 변 25% |
| `cylinder` | 위 25% · 아래 12.5% (좌우는 박스) |
| `chevron` | 박스 전체 (아래) |
| `arrow-right` | 위·아래 25% · 오른쪽 25% (왼쪽은 박스) |
| `hexagon` | 네 변 16.67% |
| `parallelogram` | 네 변 18.75% |
| `callout` | 박스 전체 (꼬리는 박스 밖이다) |
| `star` | 좌우 30.90% · 위 38.20% · 아래 23.61% |
| `cloud` | 왼 13.78% · 위 15.10% · 오른 20.89% · 아래 19.74% |

`pad`는 이 사각형에 **더해진다** — 프리셋의 영역이 먼저이고 저자의 여백이 그 안이다.
`rounded-rect`만 인셋이 px인 것은 `radius`가 px이기 때문이고, 그 덕분에 이 프리셋은
인셋을 `padding`으로 받아 **내용이 높이를 정하는 카드**로 계속 쓸 수 있다.

- **백분율 인셋 프리셋(위 표의 8종)에 내용을 담으면 `h`나 `ratio`가 필요하다.** 인셋의 기준이
  자기 박스 높이인데 그 높이를 내용이 정하면 기준이 순환한다. 값 층이 에러로 막는다 (아래
  진단 표).
- **chevron 안의 단계 라벨은 도형 밖에 둔다.** 우리 유도에서 노치 깊이가 박스 폭의 절반이라
  (정규 공간을 늘린다) 실루엣 안에 라벨이 들어갈 자리가 없다: 가운데 정렬한 줄은 노치 끝을 가로지른다. 어휘가
  막지는 않으므로(짧은 기호 하나는 들어간다) 두 타깃은 같게 나오지만, 권하는 배치는 `<col>`
  안의 형제 `<text>`다.
  PPTX의 조정값은 노치를 폭의 절반보다 아주 조금(100000분의 1) 깊게 낸다 — 정확히 절반이면 프리셋의
  텍스트 영역 공식이 경계에 걸려 뷰어마다 박스 전체와 폭 0 열로 갈린다. 조금 넘기면 어느 뷰어에서든
  박스 전체다.

두 타깃은 같은 도형에 같은 글을 같은 정렬로 놓는다. 글자가 그 박스 안 어디에 그려지는지는 뷰어의 몫이다.

**흡수 모드에서는 구조가 두 타깃에서 비대칭이다**: 웹은 도형 div **안에 중첩된 텍스트 박스**이고,
PPTX는 **같은 도형의 본문**(`txBody`)이다. 그래서 도형 안 텍스트는 재기는 하지만 자기 좌표를 갖지
않는다.

**배경 모드는 그 비대칭을 걷어낸다**: 자식은 두 타깃 모두 자기 박스를 가지므로 잰 rect가 그대로
좌표가 된다 (웹은 도형 div 안의 흐름 요소, PPTX는 도형 **뒤에 오는 형제** `p:sp`다 — 문서 순서가
곧 z 순서이므로 배경이 먼저 그려진다). 그룹 도형(`p:grpSp`)을 쓰지 않은 이유는 그룹의
`chOff`/`chExt`가 두 번째 좌표 공간을 만들어 "잰 rect가 곧 좌표다"를 깨기 때문이다.

## 예제

`testdata/fixtures/04_shape/slides/002_token_colors.slide` — 토큰 색과 `fill="none"`:

```html
<slide id="shape-token-colors">
  <h2>토큰 색은 테마 참조로 보존된다</h2>
  <row gap="lg" align="stretch">
    <shape type="rounded-rect" fill="surface" border="1 solid muted" radius="md" w="300" h="160" pad="md">
      <h3>surface</h3>
      <p>배경은 토큰 surface, 테두리는 muted다.</p>
    </shape>
    <shape type="rounded-rect" fill="accent1" radius="md" w="300" h="160" pad="md">
      <h3 color="bg">accent1</h3>
      <p color="bg">글자색도 토큰(bg)이다.</p>
    </shape>
    <shape type="rect" fill="none" border="3 solid accent1" w="300" h="160" pad="md">
      <p>fill="none" — 채움 없는 테두리 도형.</p>
    </shape>
  </row>
</slide>
```

`004_appearance.slide` — 나머지 외관 속성 (각각 한 속성만 다르다):

```html
<row gap="md" align="center">
  <shape type="rounded-rect" fill="accent1" radius="sm" w="180" h="120" pad="sm">…</shape>
  <shape type="rounded-rect" fill="accent1" radius="lg" w="180" h="120" pad="sm">…</shape>
  <shape type="rounded-rect" fill="accent1" radius="32" w="180" h="120" pad="sm">…</shape>
</row>
<row gap="md" align="center">
  <shape type="rect" fill="surface" shadow="soft" w="180" h="120" pad="sm">…</shape>
  <shape type="rect" fill="accent1" opacity="0.4" w="180" h="120" pad="sm">…</shape>
  <shape type="rect" fill="accent1" rotate="45" w="120" h="120"/>
  <shape type="chevron" fill="muted" flip="h" w="180" h="120" pad="sm">…</shape>
</row>
```

`testdata/fixtures/11_shape_layout/slides/001_card.slide` — **카드 하나가 도형 하나다** (배경 모드):

```html
<grid cols="3" gap="lg">
  <shape type="rounded-rect" fill="surface" border="1 solid muted" radius="md" w="fill" h="260" pad="md">
    <col gap="sm" justify="between" h="fill">
      <text role="heading">측정</text>
      <p>브라우저가 배치를 계산하고 우리는 그 결과를 잰다.</p>
      <text role="caption">geom.Refs</text>
    </col>
  </shape>
  <!-- … 카드 둘 … -->
</grid>
```

푸터를 `justify="between"`으로 아래에 붙이는 것이 흡수 모드로는 불가능한 자리다 —
`bodyPr`의 앵커는 본문 전체에 하나뿐이다.

`04_shape/006_cards.slide` — 같은 카드를 **도형 둘**로 흉내 낸 옛 방법 (헤더 도형 + 본문 도형을
`<col gap="0">`으로 붙인다). 배경 모드와 견주면 이 방법의 값이 보인다 — 테두리가 두 번 그려지고
카드 전체의 둥근 모서리를 만들 수 없다:

```html
<grid cols="4" gap="lg">
  <col gap="0">
    <shape type="rounded-rect" fill="accent1" radius="md" w="fill" pad="sm">
      <text color="bg" align="center">공정성</text>
    </shape>
    <shape type="rect" fill="surface" border="1 solid muted" w="fill" h="160" pad="md">
      <p anchor="top">데이터와 모델의 편향을 측정한다.</p>
    </shape>
  </col>
  <!-- … 카드 셋 … -->
  <shape type="rounded-rect" span="2" fill="none" border="3 solid accent1" radius="lg" w="fill" h="120" pad="md">
    <p anchor="center">span="2" — 도형도 컨테이너의 자식이므로 배치 속성을 받는다.</p>
  </shape>
</grid>
```

## 렌더 결과

### 웹 — 박스 div + 그림 svg

도형은 **`<div>` 하나와 그 안의 `<svg>` 하나**다. 박스가 자리·크기·회전·앵커를 갖고, svg가
채움·테두리·그림자·투명도·대칭을 갖는다:

```html
<div class="balpyo-shape" data-balpyo-id="s0.1.0.0" data-balpyo-el="shape"
     style="width: 220px; height: 110px">
<svg class="balpyo-shape-ink" aria-hidden="true"
     style="fill: var(--balpyo-color-surface); stroke: var(--balpyo-color-muted); stroke-width: 1px"><rect width="100%" height="100%"/></svg>
</div>
```

**12종 전부 SVG 한 벌이다** — 사각형·둥근 사각형·타원까지 SVG로 그린다. CSS
`clip-path`로 세 종을 그리는 길을 버린 이유는 클리핑이 테두리와 그림자를 함께 자르고,
컴포넌트 기본값이 `border: 1 solid muted`라 테두리가 예외가 아니라 기본이기 때문이다.

- **박스와 그림의 분담이 OOXML의 자리에서 나온다**: `rotate`는 `xfrm@rot`이 텍스트까지 함께
  돌리므로 박스, `flip`은 `flipH`/`flipV`가 글자를 뒤집지 않으므로 그림, `opacity`는 알파가
  `solidFill` 안에 있으므로 그림이다.
- **그림자는 `box-shadow`가 아니라 `filter: drop-shadow`다.** box-shadow는 박스의 사각형을
  따르므로 열두 종 중 아홉이 엉뚱한 사각 그림자를 얻는다 — drop-shadow는 실루엣의 알파를
  따라가 `outerShdw`와 같은 그림이 된다.
- 회전은 엔진 CSS의 클래스(`.balpyo-shape-rotate`)와 인라인 커스텀 프로퍼티로 갈라져 있다 —
  측정 뷰가 변형을 끌 수 있어야 하기 때문이다.
- svg는 `overflow: visible`이다 — 테두리 번짐과 `callout`의 꼬리가 박스 밖에 그려진다.
- 자식 텍스트는 `position: relative`로 그림 위에 온다 (절대 배치된 svg보다 뒤에 칠해져야
  글자가 보인다). **배경 모드에서는 올릴 것이 글자가 아니라 컨테이너이므로** 엔진 CSS가
  컨테이너 넷(`.balpyo-row`·`.balpyo-col`·`.balpyo-grid`·`.balpyo-stack`)도 같은 규칙에
  넣는다 — 도형의 DOM은 그대로다 (`<div class="balpyo-shape">` 안의 흐름 콘텐츠).

### PPTX — 프리셋 도형 하나

**도형은 `sp` + `prstGeom` 하나가 되고, 도형 안 텍스트는 같은 도형의 본문이 된다.**

```xml
<p:sp>
  <p:nvSpPr>
    <p:cNvPr id="3" name="shape s1.1.0"/>
    <p:cNvSpPr/>                      <!-- txBox="1"이 없다 = 텍스트 박스가 아니라 도형 -->
    <p:nvPr/>
  </p:nvSpPr>
  <p:spPr>
    <a:xfrm><a:off x="609600" y="1283940"/><a:ext cx="2857500" cy="1524000"/></a:xfrm>
    <a:prstGeom prst="roundRect">
      <a:avLst><a:gd name="adj" fmla="val 7500"/></a:avLst>
    </a:prstGeom>
    <a:solidFill><a:schemeClr val="bg2"/></a:solidFill>
    <a:ln w="9525">
      <a:solidFill><a:schemeClr val="tx2"/></a:solidFill>
      <a:prstDash val="solid"/>
    </a:ln>
  </p:spPr>
  <p:txBody>
    <a:bodyPr wrap="square" lIns="228600" tIns="228600" rIns="228600" bIns="228600" anchor="t">
      <a:normAutofit/>
    </a:bodyPr>
    …문단들…
  </p:txBody>
</p:sp>
```

- **`txBox="1"`을 붙이지 않는다.** PowerPoint는 텍스트 박스와 도형에 다른 서식 창을 주고,
  도형이 텍스트 박스로 표시되면 "도형 모양 변경"이 사라진다.
- 좌표는 도형의 잰 rect 그대로다 (텍스트 박스와 같다). 글자가 넘치면 `a:normAutofit`이
  줄여 맞춘다.
- `pad` → `bodyPr`의 네 인셋(EMU), `anchor` → `bodyPr@anchor`(`t`/`ctr`/`b`),
  `wrap`은 언제나 `square`다 (본문에 `bodyPr`이 하나뿐이라 도형 안 `<pre>` 하나 때문에
  형제 문단의 줄바꿈을 끌 수 없다).

### PPTX — 배경 모드는 도형 + 형제들

배경 모드 도형은 **빈 본문(`<a:p/>`)을 가진 `p:sp` 하나**이고, 자식은 그 뒤에 오는 **형제**
도형이다 (그룹이 아니다 — 위의 비대칭 절):

```xml
<p:sp>…카드 배경 (prstGeom + solidFill + ln, txBody는 <a:p/> 하나)…</p:sp>
<p:sp>…헤더 띠: 자식 도형이 자기 prstGeom과 자기 xfrm을 갖는다…</p:sp>
<p:sp>…본문 문단: 텍스트 박스 (txBox="1", 잰 rect 그대로)…</p:sp>
```

- 배경의 본문이 비어도 `bodyPr`의 인셋과 `anchor`는 그대로 나간다 — 자식이 없는 도형과 **같은
  경로**다 (PowerPoint에서 그 카드에 글을 덧쓰면 저자가 준 `pad` 안에 들어간다).
- 자식 `<text>`는 텍스트 박스가 되고, 좌표는 자기 잰 rect 그대로다.
- 도형 번호(`p:cNvPr@id`)는 **도형을 실제로 낸 자리에서만** 오른다 — 배경 카드가 한 번,
  자식이 각각 한 번이다.

### 내 슬라이드는 PPTX에서 도형 몇 개가 되는가

| SOML | PPTX 도형 수 |
|---|---|
| 레이아웃 컨테이너 (`<row>` `<col>` `<grid>`) | **0개** — 자식이 절대 좌표 도형이 된다 |
| `<spacer/>` | **0개** — 배치 수단이라 남을 것이 없다 |
| 리스트 컨테이너 (`<ul>` `<ol>`) | **1개** — 불릿 문단이 한 텍스트 박스에 산다 |
| **`<shape>` (흡수 모드)** | **1개** (`p:sp`) — 그리고 **자식 텍스트를 흡수한다** |
| **`<shape>` (배경 모드)** | **1개 + 자식들** — 배경 `p:sp` 하나에 자식마다 형제 도형 |
| [`<image>`](image.md) | 1개 (`p:pic`) — 담을 것이 없다 (void) |
| `<text>`·블록 태그 (슬라이드 직속·컨테이너 안) | 1개씩 |

흡수 모드 행이 앞의 셋과 정반대인 자리다: **컨테이너는 사라지면서 자식이 도형이 되고, 도형은
남으면서 자식을 흡수한다.** 그래서 도형 안에 `<h3>` + `<p>`를 넣은 카드는 도형 1개다. 배경 모드
행은 그 둘을 합친 것이다 — 도형이 **남고** 자식도 도형이 된다: 헤더 띠 + 본문 + 푸터를 담은 카드
하나는 도형 4개다 (배경 1 + 띠 1 + 문단 2). 그림 행은 **태그가 다르다** — `p:sp`가 아니라
`p:pic`이다.

## 빌드 에러가 나는 조건

파서(`soml.Parse`) — 이름·자식 어휘:

| 조건 | 진단 메시지 |
|---|---|
| 도형 안에 문자 데이터를 직접 썼다 | `<shape> 안에 문자 데이터를 직접 쓸 수 없다 (<text>로 감싼다)` |
| 도형의 **직속** 자식으로 리스트·그림·표·중첩 도형을 썼다 | `<shape>의 자식으로 <ul>는 올 수 없다 (쓸 수 있는 태그: <text>, <h1>, <h2>, <h3>, <p>, <blockquote>, <pre>, <row>, <col>, <grid>, <stack>)` — 컨테이너로 감싸면 들어간다 |
| 어휘 밖의 속성 | `<shape>에 알 수 없는 속성 stroke (쓸 수 있는 속성: type, id, alt, variant, fill, border, radius, shadow, opacity, rotate, flip, pad, w, h, min-w, min-h, max-w, max-h, ratio, grow, shrink, basis, span, self, order, at, x, y, style, slot)` |
| `<arrow>`·`<line>`·`<group>`을 썼다 | `알 수 없는 태그 <arrow> (<slide> 안에 쓸 수 있는 태그: …)` — 목록은 그 자리에 쓸 수 있는 태그다 |

값 해석(`core.Resolve`) — 값 어휘는 이 층이 소유한다:

| 조건 | 진단 메시지 |
|---|---|
| `type`이 없다 | `<shape>에 type이 필요하다 (쓸 수 있는 종류: rect, rounded-rect, ellipse, diamond, cylinder, chevron, arrow-right, hexagon, parallelogram, callout, star, cloud) — 도형의 종류는 추측할 수 없다` |
| `type`이 12종 밖이다 | `알 수 없는 도형 type "elipse" (쓸 수 있는 종류: …)` |
| 같은 슬라이드에 `id`가 둘 | `id "db"가 이 슬라이드에 이미 있다 (…) — 참조가 어느 도형인지 정할 수 없다` |
| `alt`에 값이 없다 | `alt 속성에는 값이 필요하다 (alt="chart") — 이름이 필요 없으면 속성을 지운다` |
| `alt=""` | `alt은 비어 있을 수 없다 — 이름 없음이 기본값이므로 속성을 지운다 (그림의 alt=""와 다르다: <img>에서 빈 alt은 "장식 이미지"라는 뜻이지만 도형·표에는 그 표기가 없다)` |
| 테마에 없는 색 토큰 | `fill의 색 토큰 "brand"가 테마에 없다 (쓸 수 있는 토큰: fg, bg, muted, surface, accent1, accent2, accent3, accent4, accent5, accent6, 또는 #RRGGBB 리터럴)` |
| `border` 조각 수가 3이 아니다 | `border는 \`<굵기> <스타일> <색>\` 3조각이다 ("solid muted"는 2조각) — 예: "1 solid muted", 또는 "none"` |
| `border` 굵기가 0 이하 | `border의 굵기는 0보다 커야 한다 ("0") — 테두리를 지우려면 border="none"을 쓴다` |
| 어휘 밖의 `border` 스타일 | `알 수 없는 border 스타일 "double" (쓸 수 있는 값: solid, dashed, dotted) — 조각 순서는 \`<굵기> <스타일> <색>\`이다` |
| 백분율 인셋 프리셋에 내용을 담고 `h`·`ratio`를 안 썼다 | `type="ellipse" 도형이 내용을 담으면 h나 ratio가 필요하다 (예: h="120") — 이 프리셋의 텍스트 영역은 박스 높이의 백분율이라 내용이 높이를 정하면 기준이 순환한다` |
| `radius`를 `rounded-rect` 밖에 썼다 | `radius는 type="rounded-rect"에서만 쓴다 (지금은 "rect") — 다른 프리셋에는 반경 조정값이 없어 웹만 둥글어진다` |
| 모르는 `radius` 토큰 | `radius 값이 수치가 아니다 ("huge") — 토큰(sm, md, lg)이나 수치를 쓴다` |
| 모르는 `shadow` 토큰 | `알 수 없는 shadow 토큰 "hard" (쓸 수 있는 토큰: soft) — 그림자는 수치가 아니라 토큰으로 쓴다 (두 타깃의 번짐 정의가 달라 같은 수치가 다르게 그려진다)` |
| `opacity`에 `%`를 썼다 | `opacity에 %는 쓸 수 없다 ("40%") — 0~1 실수로 쓴다 (색 변형 alpha()와 표기를 하나로 유지한다)` |
| `opacity`가 0~1 밖이다 | `opacity는 0~1 사이다 ("1.5")` |
| `rotate`에 단위를 붙였다 | `rotate는 단위 없는 각도 수치다 ("45deg"의 "deg"를 뗀다) — 허용 접미사는 % 하나다` |
| **배경 모드 도형에 `rotate`를 썼다** | `배치를 담은 도형은 rotate를 쓸 수 없다 — PPTX에서 자식은 형제 도형이 되어 도형의 회전을 물려받지 않는데 웹은 박스를 돌려 자식까지 돌리므로, 같은 SOML이 두 타깃에서 다른 그림이 된다 (회전을 지우거나 컨테이너를 빼고 텍스트만 담는다)` |
| 어휘 밖의 `flip` | `알 수 없는 flip "both" (쓸 수 있는 값: h, v, hv — "hv"는 두 축 모두다)` |
| 모르는 `variant` | **경고** — `variant "card-heder"가 테마에 없다 — 기본 외관으로 계속한다 (이 테마의 variant: card, card-header, card-body, card-backdrop, stat-card, badge)` |

크기·여백 속성(`pad`·`w`/`h`·`ratio` …)의 진단은 [레이아웃 태그](layout.md)와 **같은
문구**다 — 목록이 한 벌이므로 진단도 한 벌이다.

모르는 `variant`만 경고인 이유: variant 어휘는 테마의 사유물이라 다른 테마로 만든 덱이
빌드되지 않으면 콘텐츠의 이식성이 깨진다. `--strict`가 경고를 에러로 승격한다
([CLI](../cli.md)).
