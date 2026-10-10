# `<chart>` — 막대 차트

`<chart>`는 데이터 한 줄을 그림으로 만든다. 점마다 `<point label value/>` 한 줄이고,
막대 높이는 **값 ÷ 축 최대**다 — 좌표를 적는 자리가 없다.

다른 태그와 갈라지는 자리가 하나 있다: **산출물에 차트라는 물건이 없다.** 값 층(`core.Resolve`)이
차트 하나를 배경 도형 + 막대·라벨의 하위 트리로 **펼치고**, 그 뒤의 파이프라인(지오메트리·웹·
PPTX)은 차트라는 것을 모른다 — PPTX에 `c:chart` 파트도, 임베드 워크북도, 그림(`p:pic`)도 없다.

네이티브 차트를 내지 않는 근거는 **같은 소스가 같은 그림이어야** 한다는 것이다: 네이티브 차트는
값축·눈금·범례·라벨 타이포그래피를 뷰어가 고르므로 웹과 PPTX가 다른 컴포넌트가 되고, 동일성
판정으로 회귀를 잡을 수 없다. 도형과 텍스트는 두 타깃이 이미 같게 그린다.

## 태그와 콘텐츠 모델

| 태그 | 담는 것 | 산출물 |
|---|---|---|
| `<chart>` | `<point>`**만** | 배경 도형 1개 + 범주마다 도형 2개·텍스트 2개 (웹 `<div>`+`<svg>` · PPTX `p:sp`) |
| `<point>` | 없다 (**void**) | 막대 도형 1개 + 값 라벨·범주 라벨 2개 + 기준선 1개 |

- **`<point>`는 `<chart>` 밖에 올 수 없다** — `<li>`가 `<ul>` 밖에 못 오는 것과 같은 규칙이다.
  데이터가 자식 요소인 이유는 속성 CSV(`values="22,19,15"`)에는 값마다 색을 줄 자리가 없고
  진단이 "다섯 값 중 셋째"를 가리킬 방법이 없다는 것이다.
- `<chart>`는 슬라이드 직계 자식, [레이아웃 컨테이너](layout.md)의 자식(`<row><chart>`),
  그리고 [슬롯](../layouts.md)을 채우는 요소 셋 다 될 수 있다. [`<shape>`](shape.md)의 직속
  자식은 아니다 (표와 같은 규칙 — 컨테이너로 감싼다).
- 종류는 하나다: `type="bar"` (세로 막대, 단일 계열). `line`·`pie`는 문법이 아니라 **값의
  추가**이므로 근거가 생기면 어휘에 이름이 늘어난다.

## 프레임 속성 — `<chart>`

| 속성 | 허용 값 | 하는 일 |
|---|---|---|
| `type` | `bar` (**필수**) | 차트 종류 — 없으면 에러다 (쓸 수 있는 종류: bar) |
| `unit` | 문자열 | 값 라벨의 꼬리 (`22` + `분`) — **표기**이므로 축 계산에 들어가지 않는다 |
| `max` | 0보다 큰 수치 | 값축의 최대. 생략하면 **데이터 최대 ÷ 0.9**다 (헤드룸 10%) |
| `series` | 색 토큰 · `#rrggbb` | **모든 막대**의 색. 생략하면 `accent1` — 라벨은 칠하지 않는다 |
| `alt` | 문자열 (빈 값은 **에러**) | 그림의 **접근성 이름** — 펼침의 뿌리 도형에 실린다 |
| 박스 | `pad`·`w`·`h`·`min-w`·`min-h`·`max-w`·`max-h`·`ratio` | 도형과 같은 한 벌 — **`h`나 `ratio`는 필수다** (아래) |
| `slot` | 레이아웃의 슬롯 이름 | 슬롯을 채운다 (웹 `grid-area` · PPTX `p:ph`) |
| 자식 배치 | `grow`·`shrink`·`basis`·`span`·`self`·`order`·`at`·`x`·`y`·`style` | 레이아웃 컨테이너·[`<stack>`](stack.md)의 자식으로서의 자리 |

- **높이는 필수다** (`h`나 `ratio` 중 하나). 막대 높이가 플롯 밴드에 대한 백분율이므로 밴드가
  먼저 정해져야 하는데, 내용이 높이를 정하는 차트에서는 밴드가 막대에 달린다 — 기준이 순환한다
  (백분율 인셋 프리셋이 확정 높이를 요구하는 것과 같은 종류다). `h="fill"`도 높이다 (부모가
  준다).
- **`type`이 필수인 이유**는 오타와의 구별이다: `<chart/>`가 조용히 막대가 되면 저자의
  `tpye="bar"`가 그림을 바꾸지 않고 지나간다.
- **`fill`·`border`·`radius`·`shadow`·`variant`·`rotate`는 없다.** 펼침의 뿌리는 계약상
  **배경**이므로(`fill="none" border="none"`)
  프레임에 면의 어휘를 열면 같은 이름이 "이 박스의 면"과 "막대의 면"을 동시에 뜻하게 된다.
  배경을 원하면 저자가 `<shape>`로 감싼다.
- `alt`은 그림에 **이름**을 준다 (웹 `<svg aria-label>` · PPTX `p:cNvPr@descr`) — 막대와 라벨은
  두 타깃 모두 읽히는 글로 남으므로 이름은 그림 **위에** 얹힌다. 붙는 자리와 빈 값이 에러인
  이유는 [외관 속성의 접근성 절](../appearance.md)이 소유한다 (도형·표의 `alt`과 한 규칙이다).

## 점 속성 — `<point>`

| 속성 | 허용 값 | 하는 일 |
|---|---|---|
| `label` | 문자열 (**필수**) | 범주 라벨 — 막대가 **무엇의** 값인지다 |
| `value` | 0 이상 수치 (**필수**) | 막대 높이 = `value ÷ 축 최대`. 단위 접미사는 받지 않는다 |
| `fill` | 색 토큰 · `#rrggbb` | **이 점의 강조** — 막대와 그 점의 **두 라벨**을 함께 칠한다 |

- **크기·배치·슬롯은 붙지 않는다** — 점은 flex 항목이 아니라 데이터 한 칸이고, 막대의 폭과
  자리는 차트가 정한다 (`<li>`·셀이 자식 배치를 받지 않는 것과 같은 근거).
- `value="22분"`은 **에러**다: 표기를 값에서 떼어 `unit`에 둔 것이 이 태그의 결정이고, 값마다
  단위를 적으면 축 계산이 저자가 적은 문자열에 달린다.
- `value="0"`은 **막대를 그리지 않는다** — 높이 0의 도형은 산출물에서 아무것도 아니므로 선언을
  내지 않는다. 라벨은 남는다: "0"이 데이터다.
- 음수도 받지 않는다 — 막대 차트의 축은 0에서 시작하고, 음수 막대는 축을 0 아래로 내리는 다른
  그림이다 (v1에 없다).

## 색 — 계열은 프레임이, 강조는 점이 갖는다

채널이 둘인 것이 이 태그의 색 규칙이다:

| 채널 | 쓰는 자리 | 칠하는 것 |
|---|---|---|
| 계열색 | `<chart series="surface">` | **모든 막대**의 면 (라벨은 캡션 색 그대로) |
| 강조 | `<point fill="accent4">` | 그 점의 막대 + **값 라벨·범주 라벨** |

- **강조가 계열을 이긴다** — 계열색이 바탕이고 강조를 준 점만 거기서 벗어난다.
- **계열색은 글에 오지 않는다.** 면의 색과 글의 색은 대비 요구가 다르기 때문이다: `surface`
  (#F3F4F6) 막대는 흰 배경에서 읽히는 면이지만 같은 색 **글자**는 대비 1.06:1로 사라진다.
  그래서 "넉 달의 추세를 배경에 가라앉히고 이번 달만 앞에 세우는" 그림을 채널 하나로는 쓸 수 없다.
- 이름이 `fill`이 아니라 `series`인 이유는 뿌리 도형이 배경이라는 계약이다 — 같은 이름을 쓰면
  저자가 막대를 칠하려다 회색 상자를 받는다.
- **`none`은 두 자리 모두 에러다** (도형의 `fill`과 갈라지는 유일한 자리다): 그리지 않는 막대는
  데이터가 아니라 빠뜨린 점이고, 빈 막대를 원하면 `value="0"`이 그 표기다.
- 테마는 계열색을 갖지 않는다 — 단일 계열이므로 기본값 하나로 충분하고, 다계열이 생기는 날
  테마에 팔레트가 생긴다.

## 펼침 — 같은 뜻을 손으로 쓴 덱과 **같은 트리**다

펼침은 **저자가 쓸 수 있는 SOML만** 쓴다. 그래서 차트가 비공개 채널을 몰래 열 수 없고, 그 사실이
테스트 하나로 판정된다 (`TestChartExpandsToHandWrittenDeck`). 이 차트는

```html
<chart type="bar" unit="분" alt="월별 평균 탐지 시간" w="fill" h="420">
  <point label="5월" value="22"/>
  <point label="9월" value="12" fill="accent4"/>
</chart>
```

해석 뒤 이 덱과 **정확히 같다**:

```html
<shape type="rect" fill="none" border="none" alt="월별 평균 탐지 시간" w="fill" h="420">
  <row gap="0" align="stretch" h="fill">
    <col grow="1" basis="0" gap="0" align="center" h="fill">
      <col grow="1" basis="0" w="fill" justify="end" align="center" gap="xs">
        <text role="caption" align="center" w="fill">22분</text>
        <shape type="rect" fill="accent1" border="none" w="70%" h="90%" shrink="0"/>
      </col>
      <shape type="rect" fill="muted" border="none" w="fill" h="1"/>
      <text role="caption" align="center" w="fill">5월</text>
    </col>
    <col grow="1" basis="0" gap="0" align="center" h="fill">
      <col grow="1" basis="0" w="fill" justify="end" align="center" gap="xs">
        <text role="caption" align="center" w="fill" color="accent4">12분</text>
        <shape type="rect" fill="accent4" border="none" w="70%" h="49.09%" shrink="0"/>
      </col>
      <shape type="rect" fill="muted" border="none" w="fill" h="1"/>
      <text role="caption" align="center" w="fill" color="accent4">9월</text>
    </col>
  </row>
</shape>
```

읽는 법:

- **범주 한 칸이 `<col grow="1" basis="0">`이다** — 막대 폭과 간격을 픽셀로 세지 않는다. 칸 안에서
  플롯 밴드가 `grow="1"`로 남은 높이를 전부 먹으므로 막대의 백분율 높이는 **밴드**에 대한
  비율이고, 범주 라벨·기준선 높이는 거기서 빠진다.
- **막대 높이가 백분율이다** (`90%`·`49.09%`) — `22 ÷ (22÷0.9)`와 `12 ÷ (22÷0.9)`다. 픽셀은
  브라우저가 잰다. 소수 둘째 자리까지 자르는 이유는 이 트리가 사람이 쓸 수 있는 표기여야
  한다는 것이다.
- **축 최대가 데이터 최대와 같지 않다** (÷0.9): 값 라벨이 막대 **위에** 올라타므로 헤드룸이
  없으면 가장 높은 막대의 라벨이 밴드를 넘어 잘린다.
- **기준선은 칸마다 있는 1px 도형이다** — `gap="0"`으로 붙으면 선이 이어져 축이 된다. 축 도형
  하나를 따로 놓으면 막대의 기준선과 축선이 다른 상자에 살게 되어 두 타깃에서 어긋난다.
- **라벨은 캡션 역할이다** — 테마가 그 크기·색을 이미 갖는다 (차트 전용 타이포그래피를 만들면
  테마가 두 벌을 든다). 읽기 순서는 값 → 범주다: 스크린 리더가 "22분, 5월"로 읽는다.
- **막대의 `shrink="0"`**이 없으면 브라우저의 flex 축소가 100%에 가까운 막대를 줄인다.

## 예제

`testdata/fixtures/17_chart/slides/001_bar.slide` — 기본 축 + 마지막 점만 강조:

```html
<slide id="chart-bar">
  <h2>차트는 도형과 텍스트로 펼쳐진다</h2>
  <chart type="bar" unit="분" alt="월별 평균 탐지 시간 — 5월 22분에서 9월 12분으로 줄었다" w="fill" h="380">
    <point label="5월" value="22"/>
    <point label="6월" value="19"/>
    <point label="7월" value="15"/>
    <point label="8월" value="13"/>
    <point label="9월" value="12" fill="accent4"/>
  </chart>
  <text role="caption">축 최대를 적지 않으면 가장 높은 막대가 플롯 밴드의 90%다 — 값 라벨이 올라탈 자리가 남는다.</text>
</slide>
```

`002_axis.slide` — 저자가 정한 축 + 계열색 + 값 0인 점, 그리고 컨테이너의 자식:

```html
<row gap="lg" align="stretch">
  <chart type="bar" unit="%" max="100" series="accent2" alt="지역별 도입률" grow="1" basis="0" h="300">
    <point label="서울" value="62"/>
    <point label="부산" value="41"/>
    <point label="대전" value="0"/>
  </chart>
  <col gap="sm" grow="1" basis="0" justify="center">
    <text role="heading">0은 막대가 없다</text>
    <p>값이 0인 점은 라벨만 남는다 — 그 자리가 비어 있는 것이 데이터다.</p>
  </col>
</row>
```

`examples/decks/incident-postmortem/slides/008_detection.slide` — 두 채널을 함께 쓴 실물:

```html
<chart type="bar" unit="분" series="surface" alt="월별 평균 탐지 시간 — 5월 22분에서 이번 장애 12분까지" w="fill" h="440">
  <point label="5월" value="22"/>
  <point label="6월" value="19"/>
  <point label="7월" value="15"/>
  <point label="8월" value="13"/>
  <point label="이번 장애" value="12" fill="accent4"/>
</chart>
```

이 슬라이드가 `<chart>`가 생긴 이유다 — 손으로 쓰던 판이 도형·텍스트 **16개**였고 좌표
스물넷(`x`·`y`)이 붙어 있어서 값 하나를 고치면 여섯 숫자를 다시 셌다. 이제 값 하나가 속성 하나다.

## 렌더 결과

### 웹 — 배경 `<div>` 하나와 그 안의 도형·텍스트

```html
<div class="balpyo-shape" data-balpyo-id="s0.1" data-balpyo-el="shape" style="width: 100%; height: 380px">
<svg class="balpyo-shape-ink" role="img" aria-label="월별 평균 탐지 시간 — …" style="fill: none; stroke: none"><rect width="100%" height="100%"/></svg>
<div class="balpyo-row" data-balpyo-id="s0.1.0" data-balpyo-el="row" style="gap: 0px; align-items: stretch; height: 100%">
<div class="balpyo-col" data-balpyo-id="s0.1.0.0" data-balpyo-el="col" style="gap: 0px; align-items: center; height: 100%; flex-grow: 1; flex-basis: 0px">
<div class="balpyo-col" data-balpyo-id="s0.1.0.0.0" data-balpyo-el="col" style="gap: var(--balpyo-space-xs); align-items: center; justify-content: end; width: 100%; flex-grow: 1; flex-basis: 0px">
<div class="balpyo-text" data-balpyo-id="s0.1.0.0.0.0" data-balpyo-el="text" style="… color: var(--balpyo-color-muted); text-align: center; width: 100%">22분</div>
<div class="balpyo-shape" data-balpyo-id="s0.1.0.0.0.1" data-balpyo-el="shape" style="width: 70%; height: 90%; flex-shrink: 0">
<svg class="balpyo-shape-ink" aria-hidden="true" style="fill: var(--balpyo-color-accent1); stroke: none"><rect width="100%" height="100%"/></svg>
</div>
</div>
<div class="balpyo-shape" data-balpyo-id="s0.1.0.0.1" data-balpyo-el="shape" style="width: 100%; height: 1px">
<svg class="balpyo-shape-ink" aria-hidden="true" style="fill: var(--balpyo-color-muted); stroke: none"><rect width="100%" height="100%"/></svg>
</div>
<div class="balpyo-text" data-balpyo-id="s0.1.0.0.2" data-balpyo-el="text" style="… color: var(--balpyo-color-muted); text-align: center; width: 100%">5월</div>
</div>
```

- **차트 전용 CSS도, 차트 전용 `data-balpyo-el`도 없다** — `el`은 `shape`·`row`·`col`·`text`뿐이고
  `chart`라는 값은 산출물에 나타나지 않는다. 엔진 CSS가 늘지 않는다는 뜻이기도 하다.
- **`data-balpyo-id`가 저자의 트리를 따르지 않는다** (`s0.1.0.0.0.1` 같은 깊은 경로) — 펼친 트리의
  자리이므로, 뷰어의 배치 경고가 가리키는 자리도 합성 노드다. 소스의 자리를 가리키는 것은
  진단이고, 진단은 저자의 `<point>` 속성에 밑줄을 긋는다 (아래).
- 막대는 `<svg><rect>`다 — 도형과 **같은 잉크**이므로 채움·테두리가 자리마다 다르게 읽히지
  않는다.

### PPTX — `p:sp`만 나온다 (`c:chart` 파트가 없다)

```xml
<p:sp>
  <p:nvSpPr>
    <p:cNvPr id="5" name="shape s0.1.0.0.0.1"/>
    <p:cNvSpPr/>
    <p:nvPr/>
  </p:nvSpPr>
  <p:spPr>
    <a:xfrm>
      <a:off x="938808" y="1621036"/>
      <a:ext cx="1536055" cy="3032968"/>
    </a:xfrm>
    <a:prstGeom prst="rect">
      <a:avLst/>
    </a:prstGeom>
    <a:solidFill>
      <a:schemeClr val="accent1"/>
    </a:solidFill>
    <a:ln>
      <a:noFill/>
    </a:ln>
  </p:spPr>
  …
</p:sp>
```

- **패키지에 차트 파트가 없다**: `ppt/charts/`도, `ppt/embeddings/`의 워크북도,
  `[Content_Types].xml`의 차트 타입도 없다.
- **막대 하나가 도형 하나다** — 점 다섯인 차트는 도형 21개다 (배경 1 + 범주마다 막대·기준선·값
  라벨·범주 라벨 4개). 도형 번호도 그만큼 쓴다. 대신 PowerPoint에서 막대 하나를 끌어 옮기거나
  색을 바꿀 수 있다: **편집 가능한 그림**이다.
- **`descr`은 배경 도형에 실린다** (`id="3" name="shape s0.1" descr="…"`) — 이름은 그림 위에
  얹히고 막대·라벨의 글은 그대로 남는다.
- 좌표는 전부 **잰 값**이다: 막대의 `a:ext cy`는 브라우저가 잰 백분율 높이의 픽셀이고, 코어는
  EMU를 세지 않는다.

## 빌드 에러가 나는 조건

파서(`soml.Parse`) — 이름·문법·콘텐츠 모델:

| 조건 | 진단 메시지 |
|---|---|
| `<point>`를 차트 밖에 썼다 | `<point> can only be a child of <chart>` |
| `<chart>`에 점이 아닌 자식을 썼다 | `<chart> cannot contain <p> (available tags: <point>)` |
| `<chart>` 안에 문자 데이터를 직접 썼다 | `character data cannot appear directly inside <chart> (wrap it in <point>)` |
| `<point>`에 내용을 담았다 | `void tag <point> cannot have content (write <point/>)` |
| `<chart>`를 도형의 직속 자식으로 썼다 | `<shape> cannot contain <chart> (available tags: <text>, <h1>, <h2>, <h3>, <p>, <blockquote>, <pre>, <row>, <col>, <grid>, <stack>)` |
| `<chart>`에 `fill`을 썼다 | `unknown attribute fill on <chart> (available attributes: type, unit, max, series, alt, pad, w, h, min-w, min-h, max-w, max-h, ratio, grow, shrink, basis, span, self, order, at, x, y, style, slot, enter)` |
| `<point>`에 `w`를 썼다 | `unknown attribute w on <point> (available attributes: label, value, fill)` |

값 해석(`core.Resolve`) — 데이터와 값 어휘. 자리는 **저자의 것**이다: 합성 노드가 아니라
`<chart>`/`<point>`의 그 속성에 밑줄이 그어지고, 한 오타에 진단은 하나다:

| 조건 | 진단 메시지 |
|---|---|
| `type`이 없다 | `<chart> needs a type (available types: bar) — without a type there is no way to guess what to draw` |
| 어휘 밖 `type` | `unknown chart type "line" (available types: bar)` |
| `h`도 `ratio`도 없다 | `<chart> needs h or ratio — bar heights are percentages of the axis, so letting the content set the height would be circular` |
| 점이 하나도 없다 | `<chart> has no <point> — there is no data to draw (<point label="May" value="22"/>)` |
| `label`이 없다 | `<point> needs a label — without saying what a bar measures, the chart cannot be read` |
| `value`가 없다 | `<point> needs a value — it is the bar height` |
| `value`에 단위를 붙였다 | `value is not a number ("22분") — put the unit in <chart unit> (value="22" unit="min")` |
| `value`가 음수다 | `value cannot be negative ("-3") — a bar chart axis starts at 0` |
| 값이 모두 0인데 `max`가 없다 | `all values of <chart> are 0 — the axis cannot be determined (set max to draw an empty chart)` |
| `value`가 `max`를 넘는다 | `value 22 is greater than max 20 — the bar exceeds the axis (raise or remove max)` |
| `max`가 수치가 아니다 | `max is not a number ("스물") — write the value axis maximum as a number` |
| `max`가 0 이하다 | `max must be greater than 0 ("0") — a zero axis leaves bar heights without a denominator` |
| `series`/`fill`이 `none`이다 | `series cannot be none — a bar that is not drawn is not data (an empty bar is value="0")` |
| `alt=""` | `alt cannot be empty — no name is the default, so remove the attribute (…)` |

`series`·`fill`의 색 값 진단은 [외관 속성](../appearance.md)·[`<shape>`](shape.md)와 **같은
문구**다 (`series: color token "brand" is not in the theme`) — 목록이 한 벌이므로 진단도 한 벌이고,
진단은 저자가 쓴 속성 이름으로 말한다.

막대가 박스를 넘치는 사고는 [배치 경고](../../guides/layout-warnings.md)가 말한다 — 펼친 트리가
평범한 도형·텍스트이므로 넘침 판정도 평범하게 돈다 (값 라벨이 밴드를 넘으면 그 텍스트가
경고에 나온다).
