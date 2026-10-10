# `enter` — 등장

슬라이드에 들어갈 때 요소가 나타나는 방식이다. 콘텐츠 요소(`<text>` · 블록 · 리스트 · 레이아웃 컨테이너 ·
`<shape>` · `<image>` · `<table>` · `<chart>`)와 `<slide>`가 `enter` 속성을 받는다. 등장은 **자동**이다 —
슬라이드가 시작되면 첫 등장이, 그다음부터 한 박자씩 엇갈려 시작한다.

```html
<slide id="families" enter="rise">
  <h2>Pick a family by its bottleneck</h2>
  <grid cols="3" gap="lg">
    <shape type="rounded-rect" variant="card"><h3>General purpose</h3></shape>
    <shape type="rounded-rect" variant="card" enter="zoom"><h3>Compute optimized</h3></shape>
    <shape type="rounded-rect" variant="card"><h3>Memory optimized</h3></shape>
  </grid>
  <p enter="none">Sizes double as you go up.</p>
</slide>
```

## 효과

| `enter` | 웹 (발표자 뷰) | PPTX |
|---|---|---|
| `fade` | 투명에서 불투명으로 | 페이드 |
| `rise` | 페이드 + 아래 24px에서 제자리로 | 떠오르기 (Float In) |
| `zoom` | 페이드 + 90% 크기에서 제자리로 | 확대 (Zoom) |
| `wipe` | 왼쪽에서 오른쪽으로 드러난다 | 닦아내기 (Wipe — 왼쪽에서) |
| `none` | 등장 없음 | 등장 없음 |

PPTX에서는 PowerPoint의 기본 등장 효과다 — 애니메이션 창에 그대로 나열되고, 효과 · 지연 · 길이를 거기서
고칠 수 있다.

## 무엇이 함께 나타나는가

- **`enter`를 가진 요소 하나가 한 번의 등장이다.** 그 요소 안의 모든 것(글 · 도형 · 그림)이 함께 나타난다 —
  `<grid enter="rise">`는 격자 전체가 한 번에 떠오른다.
- **안쪽의 `enter`는 따로 등장한다.** 위 예제의 가운데 카드는 격자와 별개로 확대된다. 객체는 **가장 가까운**
  `enter` 조상의 등장에 속한다.
- **`<slide enter>`는 직속 자식마다** 같은 값을 준 것과 같다 — 제목, 격자, 문단이 하나씩 엇갈린다. 자식이
  자기 `enter`를 쓰면 그것이 이기고, `enter="none"`으로 끈다.
- 등장의 순서는 그 요소들의 **문서 순서**다.
- 연결선은 등장하지 않는다 (콘텐츠 위의 선은 처음부터 보인다).

## 박자 — 테마의 `motion`

길이 · 간격 · 첫 지연은 테마의 `tokens.json`이 덱에 하나로 정한다 (ms):

```json
"motion": { "duration": 600, "stagger": 90, "delay": 150 }
```

첫 등장은 `delay` 뒤에, n번째 등장은 `delay + n × stagger` 뒤에 시작하고, 각각 `duration` 동안 움직인다.
내장 테마 셋은 위 값이다. 테마가 `motion`을 정하지 않았는데 덱이 `enter`를 쓰면 에러다.

## 어디서 움직이는가

- **웹은 발표자 뷰에서만 움직인다.** 문서 뷰 · 오버뷰 · 인쇄는 끝 상태(전부 보이는 상태)이고, 지오메트리
  패스도 끝 상태를 잰다 — `enter`를 써도 배치와 PPTX 좌표는 한 픽셀도 바뀌지 않는다.
- 이미 지나온 장으로 **뒤로** 돌아가면 움직이지 않고 끝 상태로 보인다.
- 운영체제가 동작 줄이기(`prefers-reduced-motion`)를 켰으면 웹은 움직이지 않는다.
- JS가 꺼져 있으면 끝 상태로 열린다.

## 진단

| 조건 | 진단 메시지 |
|---|---|
| 어휘 밖의 효과 | `unknown enter "spin" (available values: fade, rise, zoom, wipe, none)` |
| `enter`에 값이 없다 (`<slide enter>`) | `the enter attribute needs a value (enter="rise")` |
| 테마에 박자가 없다 | `<p> has enter="rise", but theme "corp" defines no motion — add "motion": { "duration": 600, "stagger": 90, "delay": 150 } to its tokens.json` |
| `motion`의 수가 틀렸다 | `tokens.json motion needs duration > 0 and stagger, delay ≥ 0 (ms) — …` |
