# `<connector>` — 상자를 잇는 선

연결선 하나다. `from`·`to`가 같은 슬라이드의 [`<shape id>`](shape.md)·[`<image id>`](image.md)를 가리키고,
선은 두 상자의 **잰 자리**에서 그려진다 — 저자는 좌표를 쓰지 않는다. PPTX에서는 진짜 연결선(`p:cxnSp`)이
되어, PowerPoint에서 상자를 끌면 선이 따라온다.

```html
<slide id="lifecycle">
  <h2>Instance lifecycle</h2>
  <row gap="xl" align="center">
    <shape type="rounded-rect" id="pending" w="200" h="72"><p align="center">pending</p></shape>
    <shape type="rounded-rect" id="running" w="200" h="72" fill="accent3"><p align="center" color="bg">running</p></shape>
    <shape type="rounded-rect" id="stopped" w="200" h="72"><p align="center">stopped</p></shape>
  </row>
  <connector from="pending" to="running"/>
  <connector from="running" to="stopped" variant="accent"/>
  <connector from="stopped" to="pending" route="curved" from-side="top" to-side="top" variant="dashed"/>
</slide>
```

## 자리

- **슬라이드 직속이다.** 연결선은 흐름에 자리를 차지하지 않는다 — 문서의 어디에 써도 배치가 바뀌지 않고,
  [배치 경고](../../guides/layout-warnings.md)(넘침 · 겹침)의 대상도 아니다.
- **콘텐츠 위, 크롬 아래에 칠한다.** 두 타깃이 같은 자리다: 웹은 슬라이드마다 SVG 오버레이 하나,
  PPTX는 콘텐츠 도형들 뒤 · 크롬 앞의 `p:cxnSp`다. 연결선끼리는 문서 순서다.
- **끝점은 측정이 끝난 뒤 정해진다.** 그래서 연결선이 있는 덱은 HTML만 빌드해도 Chromium으로 잰다.
- 뒤에서 선언한 상자도 가리킬 수 있다 — 슬라이드를 다 읽은 뒤에 푼다.

## 속성

| 속성 | 값 | 기본값 | 뜻 |
|---|---|---|---|
| `from` | 같은 슬라이드의 `id` | **필수** | 선이 시작하는 상자 |
| `to` | 같은 슬라이드의 `id` | **필수** | 선이 끝나는 상자 |
| `route` | `straight` · `elbow` · `curved` | `straight` | 경로 종류 |
| `from-side` · `to-side` | `auto` · `top` · `right` · `bottom` · `left` | `auto` | 붙는 면 |
| `line` | `<굵기> <스타일> <색>` | 테마의 `connector.default` | 선 — 도형 [`border`](shape.md)와 같은 문법 |
| `start` · `end` | `none` · `arrow` · `dot` | 테마 (`minimal`: `none` · `arrow`) | 양 끝 표식 |
| `variant` | 테마가 정의한 이름 | 없음 | 선 번들 (`minimal`: `dashed` · `accent` · `plain`) |
| `label` | 짧은 글 | 없음 | 선 라벨 — 한 줄, 경로 옆 |
| `enter` | `draw` · `fade` · `none` | 없음 | 등장 — 아래 |
| `flow` | 없음 · `true` · `false` | `false` | 대시가 시작점에서 끝점 쪽으로 흐른다 — 아래 |

- **면은 가운데에 붙는다.** `auto`는 상대 상자를 향하는 면이다: 두 상자 중심의 가로 차가 세로 차
  이상이면 마주 보는 가로 면(오른쪽 ↔ 왼쪽), 아니면 세로 면(아래 ↔ 위)이다.
- **꺾인 경로는 두 면의 축이 정한다.** 축이 같으면(오른쪽 → 왼쪽, 위 → 아래) 가운데에서 꺾는 계단
  (`bentConnector3` · `curvedConnector3`), 축이 다르면(오른쪽 → 위) 한 번 꺾는 L자(`bentConnector2` ·
  `curvedConnector2`)다. **같은 면끼리**(위 → 위) 이으면 두 상자 바깥으로 나갔다 돌아오는 U자다 —
  꺾임은 32px 바깥, 곡선은 두 끝 거리의 1/4까지 더 부푼다.
- **면이 상대를 등지면 상자를 돌아간다.** 오른쪽 → 왼쪽인데 상대가 왼쪽에 있거나, 위 → 아래인데
  상대가 아래에 있으면 계단은 두 상자를 꿰뚫는다. 그때 선은 면에서 32px 바깥으로 나간 뒤 두 상자
  사이의 틈(틈이 없으면 두 상자 바깥)을 지나 끝 면에 바깥에서 들어오는 S자(`bentConnector5` ·
  `curvedConnector5`)다. 축이 다른 두 면에서 L자가 상자를 지나게 되면(오른쪽 → 위인데 상대가 왼쪽
  아래에 있으면) 한 번 더 꺾는다(`bentConnector4` · `curvedConnector4`).
- 웹의 SVG 경로는 같은 PPTX 프리셋을 같은 박스에서 평가한 것이다. 두 타깃의 선이 같은 출처에서 나온다.
- 표식은 선 굵기의 5배 크기다 (PPTX `w="lg" len="lg"`). 화살표 끝의 선은 화살촉 아래로 삐져나오지 않게
  줄여 그린다.
- `line`에는 그라디언트와 `none`을 쓸 수 없다 — 선 하나는 색 하나다.

## 라벨

`label`은 선 옆의 한 줄 글이다. **경로 길이의 가운데**에 앉는다 — 그 자리의 선이 가로에 가까우면 선
**위**, 세로에 가까우면 선 **오른쪽**에 6px 띄운다. 글자는 `caption` 역할에 테마의 `connector` 태그
기본값(`text.componentDefaults`의 `{ "tag": "connector", … }`)이 얹힌 것이다.

- 지오메트리 패스가 라벨의 크기를 재고, 두 타깃이 그 크기를 경로 옆 같은 자리에 놓는다. PPTX에서는
  모든 선 뒤의 한 줄 글상자(`wrap="none"`)다 — 선 위에 글자가 칠해진다.
- 라벨은 배치 경고의 대상이 아니다. 아치가 높은 U자 경로의 라벨은 슬라이드 위로 나갈 수 있다 — 상자를
  내려 자리를 만든다 (`<slide justify="center">`).

## 등장 — `enter`

연결선도 [등장](enter.md)한다 — 순서와 박자는 상자의 등장과 같은 줄에 선다 (문서 순서, 테마의 `motion`).
`<slide enter>`는 연결선에 닿지 않는다: 연결선은 자기 `enter`만 받는다.

| `enter` | 웹 (발표자 뷰) | PPTX |
|---|---|---|
| `draw` | 시작점에서 끝점으로 그려진다 — 실선은 선을 따라, 파선은 시작하는 쪽에서 닦아낸다. 끝 표식은 선이 닿을 무렵 나타난다 | 닦아내기 — 시작점 쪽 변에서 (선의 주된 방향) |
| `fade` | 선과 표식이 함께 나타난다 | 페이드 |

라벨은 그 선과 같은 시각에 페이드로 나타난다. PowerPoint에는 경로를 따라 그리는 등장이 없어 `draw`는
닦아내기가 가장 가까운 원어다 — 꺾인 선은 두 타깃에서 그려지는 순서가 다를 수 있다.

## 흐르는 선 — `flow`

`<connector flow>`는 선의 대시가 **시작점에서 끝점 쪽으로 계속 흐르게** 한다 — 요청이 지나는 경로처럼. 흐르는
선은 언제나 파선이다 (`line`이 실선이면 같은 굵기 · 색의 파선이 된다).

- 움직이는 곳은 **웹의 발표자 뷰**, 지금 장 하나다 (대시 한 주기를 1.2초에). PPTX · 인쇄 · 문서 뷰 · 오버뷰에서는 같은
  파선이 멈춰 있다 — PowerPoint에는 선의 대시를 흘리는 효과가 없고, 멈춘 파선이 그 경로를 그대로 말한다.
- 동작 줄이기(`prefers-reduced-motion`)를 켰으면 흐르지 않는다.
- 값 없는 `flow`는 참이다. 그 밖의 값은 `flow must be true or false ("yes")`다.

## 붙는 상자

PPTX에서 양 끝이 상자의 **연결점**에 붙는 것은 연결점 순서가 정해진 상자뿐이다: `rect` · `rounded-rect` ·
`diamond` · `ellipse` · 그림(`<image>`). 다른 프리셋(`chevron` · `star` …)을 잇는 끝은 같은 자리에
그려지지만 붙지 않는다 — PowerPoint에서 그 상자를 끌면 선은 그 자리에 남는다.

| 상자 | 위 · 왼쪽 · 아래 · 오른쪽의 연결점 |
|---|---|
| `rect` · `rounded-rect` · `diamond` · 그림 | 0 · 1 · 2 · 3 |
| `ellipse` | 0 · 2 · 4 · 6 |

## 테마

선의 기본값과 variant는 테마의 `components.json`이 정한다 — 캐스케이드는 도형과 같다: 인라인 > variant >
`connector.default`.

```json
"connector": {
  "default": { "line": "2 solid muted", "start": "none", "end": "arrow" },
  "variants": [
    { "name": "dashed", "line": "2 dashed muted" },
    { "name": "accent", "line": "2 solid accent1" },
    { "name": "plain",  "end": "none" }
  ]
}
```

테마가 선을 정하지 않았고 저자도 `line`을 쓰지 않았으면 에러다 — 엔진에는 선의 기본값이 없다.

## 진단

| 조건 | 진단 메시지 |
|---|---|
| `from`·`to`가 없다 | `<connector> needs from — the id of the shape or image the line starts at` |
| 없는 이름 | `<connector> to="db" is not the id of a shape or image on this slide (ids on this slide: app, web)` — 비슷한 이름이 있으면 `did you mean "…"?`가 붙는다 |
| 같은 상자를 잇는다 | `<connector> from and to are the same element "a" — a connector joins two boxes` |
| 어휘 밖의 경로 | `unknown route "zigzag" (available values: straight, elbow, curved)` |
| 어휘 밖의 면 | `unknown to-side "north" (available values: auto, top, right, bottom, left)` |
| 어휘 밖의 표식 | `unknown end "diamond" (available values: none, arrow, dot)` |
| 속성에 값이 없다 (`<connector … line/>`) | `the line attribute needs a value (line="2 solid muted")` — `route` · `from-side` · `to-side` · `start` · `end`도 쓸 수 있는 값 하나를 예시로 싣는다 |
| `line="none"` | `a connector line cannot be none — remove the connector instead` |
| 그라디언트 선 | `a connector line is one color — gradients are not supported on connectors` |
| 선이 정해지지 않았다 | `<connector> has no line — theme "corp" defines no connector.default line, so write line="2 solid muted"` |
| 슬라이드 직속이 아니다 | `<row> cannot contain <connector> (available tags: …)` |
