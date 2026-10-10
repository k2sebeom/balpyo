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

- **면은 가운데에 붙는다.** `auto`는 상대 상자를 향하는 면이다: 두 상자 중심의 가로 차가 세로 차
  이상이면 마주 보는 가로 면(오른쪽 ↔ 왼쪽), 아니면 세로 면(아래 ↔ 위)이다.
- **꺾인 경로는 두 면의 축이 정한다.** 축이 같으면(오른쪽 → 왼쪽, 위 → 아래) 가운데에서 꺾는 계단
  (`bentConnector3` · `curvedConnector3`), 축이 다르면(오른쪽 → 위) 한 번 꺾는 L자(`bentConnector2` ·
  `curvedConnector2`)다. **같은 면끼리**(위 → 위) 이으면 두 상자 바깥으로 나갔다 돌아오는 U자다 —
  꺾임은 32px 바깥, 곡선은 두 끝 거리의 1/4까지 더 부푼다.
- 웹의 SVG 경로는 같은 PPTX 프리셋을 같은 박스에서 평가한 것이다. 두 타깃의 선이 같은 출처에서 나온다.
- 표식은 선 굵기의 5배 크기다 (PPTX `w="lg" len="lg"`). 화살표 끝의 선은 화살촉 아래로 삐져나오지 않게
  줄여 그린다.
- `line`에는 그라디언트와 `none`을 쓸 수 없다 — 선 하나는 색 하나다.

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
| `line="none"` | `a connector line cannot be none — remove the connector instead` |
| 그라디언트 선 | `a connector line is one color — gradients are not supported on connectors` |
| 선이 정해지지 않았다 | `<connector> has no line — theme "corp" defines no connector.default line, so write line="2 solid muted"` |
| 슬라이드 직속이 아니다 | `<row> cannot contain <connector> (available tags: …)` |
