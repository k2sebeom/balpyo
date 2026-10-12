# `<chart>` — 막대 · 선 · 영역 · 계단 차트

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
| `<chart>` | `<point>`**만** 또는 `<series>`**만** | 배경 도형 1개 + 범주마다 도형 2개·텍스트 2개 (웹 `<div>`+`<svg>` · PPTX `p:sp`) |
| `<series>` | `<point>`**만** | 계열 하나 — 계열마다의 막대 · 선과 범례 항목 하나 |
| `<point>` | 없다 (**void**) | 막대 도형 1개 + 값 라벨·범주 라벨 2개 + 기준선 1개 |

- **`<point>`는 `<chart>` · `<series>` 밖에 올 수 없다** — `<li>`가 `<ul>` 밖에 못 오는 것과 같은 규칙이다.
  데이터가 자식 요소인 이유는 속성 CSV(`values="22,19,15"`)에는 값마다 색을 줄 자리가 없고
  진단이 "다섯 값 중 셋째"를 가리킬 방법이 없다는 것이다.
- `<chart>`는 슬라이드 직계 자식, [레이아웃 컨테이너](layout.md)의 자식(`<row><chart>`),
  그리고 [슬롯](../layouts.md)을 채우는 요소 셋 다 될 수 있다. [`<shape>`](shape.md)의 직속
  자식은 아니다 (표와 같은 규칙 — 컨테이너로 감싼다).
- 종류는 넷이다 — 점을 직접 담으면 **단일 계열**, `<series>`를 담으면 여러 계열이다 (아래 "여러 계열" 절): `bar`(세로 막대), `line`(점을 잇는 꺾은선), `area`(꺾은선 아래를 칠한
  영역), `step`(범주 칸마다 수평인 계단). 선 · 영역 · 계단은 [경로 도형](shape.md)(`type="path"`) 하나로 펼쳐진다
  (아래 "선 · 영역 · 계단" 절).

## 프레임 속성 — `<chart>`

| 속성 | 허용 값 | 하는 일 |
|---|---|---|
| `type` | `bar` · `line` · `area` · `step` (**필수**) | 차트 종류 — 없으면 에러다 |
| `unit` | 문자열 | 값 라벨의 꼬리 (`22` + `분`) — **표기**이므로 축 계산에 들어가지 않는다 |
| `max` | 0보다 큰 수치 | 값축의 최대. 생략하면 **데이터 최대 ÷ 0.9**다 (헤드룸 10%) |
| `series` | 색 토큰 · `#rrggbb` | **모든 막대**의 색. 생략하면 `accent1` — 라벨은 칠하지 않는다 |
| `alt` | 문자열 (빈 값은 **에러**) | 그림의 **접근성 이름** — 펼침의 뿌리 도형에 실린다 |
| 박스 | `pad`·`w`·`h`·`min-w`·`min-h`·`max-w`·`max-h`·`ratio` | 도형과 같은 한 벌 — **`h`나 `ratio`는 필수다** (아래) |
| `slot` | 레이아웃의 슬롯 이름 | 슬롯을 채운다 (웹 `grid-area` · PPTX는 슬롯 자리의 도형 — 글자가 없으므로 `p:ph`가 아니다) |
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

## 선 · 영역 · 계단

`line` · `area` · `step`은 막대 대신 **경로 하나**가 계열이다. 칸 나눔은 막대와 같다 — 범주 칸이 폭을 똑같이
나누고, 범주 라벨이 칸 가운데에 선다.

```html
<chart type="area" series="accent4" alt="Demand over the day" w="fill" h="320">
  <point label="6am" value="12"/>
  <point label="9am" value="40"/>
  <point label="noon" value="55"/>
  <point label="3pm" value="48"/>
  <point label="6pm" value="70"/>
</chart>
```

- **점의 자리**: 가로는 범주 칸의 가운데, 세로는 막대 높이와 같은 비율(값 ÷ 축)이다. 그 0~100 좌표로 쓴 경로
  도형이 플롯 밴드로 늘어난다 — 픽셀은 여전히 브라우저가 잰다.
- `line`은 점을 직선으로 잇는다. `step`은 칸마다 그 값의 높이로 칸 폭 전체를 수평으로 긋고 칸 경계에서 꺾는다.
  `area`는 `line`과 같은 선 아래를 기준선까지 계열색 20%로 칠하고 그 위에 선을 긋는다.
- 선의 굵기는 3px이고 색은 `series`다. 선 차트는 **값 라벨을 그리지 않는다** — 점 위의 수치가 필요하면 막대
  차트다.
- 점의 강조색(`<point fill>`)은 막대 하나를 칠하는 채널이라 선 · 영역 · 계단에서는 에러다.
- PPTX에서는 사용자 지오메트리 도형(`a:custGeom`)이다 — PowerPoint의 점 편집으로 고칠 수 있다.

## 여러 계열 — `<series>`

`<series>`는 계열 하나다. 점들을 담고 이름 · 색을 갖는다. 계열이 여럿이면 축 하나를 함께 쓰고, 범주 라벨 아래에
범례가 선다.

```html
<chart type="bar" unit="M" alt="Revenue by quarter" w="fill" h="360">
  <series name="2025">
    <point label="Q1" value="12"/>
    <point label="Q2" value="18"/>
  </series>
  <series name="2026" fill="accent4">
    <point label="Q1" value="16"/>
    <point label="Q2" value="21"/>
  </series>
</chart>
```

| 속성 | 허용 값 | 하는 일 |
|---|---|---|
| `name` | 문자열 (**필수**) | 범례에 실리는 계열 이름 |
| `fill` | 색 토큰 · `#rrggbb` | 계열색. 생략하면 순서대로 `accent1` … `accent6`이다 (일곱째 계열부터 되돌아간다) |
| `type` | `line` · `area` · `step` | 선 계열 차트에서 이 계열만의 종류 — 수요 선(`area`) 위에 용량 계단(`step`) |

- **한 차트는 점 아니면 계열이다.** 직계 `<point>`와 `<series>`를 섞으면 에러다. 계열 형식에서는 프레임의
  `series`도 에러다 — 색은 계열마다 있다.
- **범주는 모든 계열이 같다.** 첫 계열의 라벨이 범주이고, 다른 계열도 같은 라벨을 같은 순서로 적는다.
- **축**은 모든 계열의 값 중 최대에서 정한다 (`max`가 있으면 그 축).
- **막대 차트는 묶음 막대다** — 범주 칸마다 계열 수만큼의 막대가 칸의 70% 폭을 나눠 붙어 선다. 값 라벨은 막대마다
  있다. `<point fill>` 강조는 그 막대 하나와 그 값 라벨이다.
- **선 · 영역 · 계단 차트는 계열마다 경로 하나다** — 같은 플롯 밴드에 겹친다. 영역의 면을 먼저 다 칠하고 선을 그
  위에 긋는다. 막대 차트의 계열에는 `type`을 쓸 수 없다.
- **범례**는 계열마다 색 견본과 이름 캡션이고, 가운데 정렬된 한 줄이다. 견본은 막대 · 영역 계열이면 사각형, 선 · 계단
  계열이면 굵은 선이다.

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
- 테마는 차트 전용 팔레트를 갖지 않는다 — 계열색의 기본값은 테마의 **강조색 여섯**이다. 단일
  계열은 `accent1`이고, 여러 계열은 순서대로 `accent1` … `accent6`이다 (아래 "여러 계열" 절).
  그래서 테마의 강조색 순서가 곧 차트의 기본 계열 순서다.

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
  <col gap="0" w="fill" h="fill">
    <row gap="0" align="stretch" grow="1" basis="0" w="fill">
      <col grow="1" basis="0" gap="0" align="center">
        <col grow="1" basis="0" w="fill" align="center" gap="xs">
          <spacer grow="0" basis="10%"/>
          <text role="caption" align="center" w="fill" shrink="0">22분</text>
          <shape type="rect" fill="accent1" border="none" w="70%" basis="90%"/>
        </col>
        <shape type="rect" fill="muted" border="none" w="fill" h="1"/>
      </col>
      <col grow="1" basis="0" gap="0" align="center">
        <col grow="1" basis="0" w="fill" align="center" gap="xs">
          <spacer grow="0" basis="50.91%"/>
          <text role="caption" align="center" w="fill" color="accent4" shrink="0">12분</text>
          <shape type="rect" fill="accent4" border="none" w="70%" basis="49.09%"/>
        </col>
        <shape type="rect" fill="muted" border="none" w="fill" h="1"/>
      </col>
    </row>
    <row gap="0" w="fill">
      <text role="caption" align="center" w="fill" grow="1" basis="0">5월</text>
      <text role="caption" align="center" w="fill" color="accent4" grow="1" basis="0">9월</text>
    </row>
  </col>
</shape>
```

읽는 법:

- **범주 한 칸이 `<col grow="1" basis="0">`이다** — 막대 폭과 간격을 픽셀로 세지 않는다. 칸 안에서
  플롯 밴드가 `grow="1"`로 남은 높이를 전부 먹고, 기준선 높이는 거기서 빠진다.
- **범주 라벨은 칸 안이 아니라 아래의 한 줄이다.** 칸 안에 두면 두 줄로 접힌 라벨 하나가 그 칸의
  밴드만 줄여 그 막대가 이웃보다 짧게 그려진다. 한 줄로 모으면 모든 칸의 밴드가 같은 높이다 — 줄의
  높이는 가장 높은 라벨이 정한다. 라벨 줄의 칸도 `grow="1" basis="0"`이라 막대와 같은 열에 선다.
- **막대 높이가 `basis` 백분율이다** (`90%`·`49.09%`) — `22 ÷ (22÷0.9)`와 `12 ÷ (22÷0.9)`다. 위의
  `<spacer>`가 나머지(`10%`·`50.91%`)를 잡는다. 둘을 합치면 밴드를 라벨 한 줄 + 간격만큼 넘치고,
  flex 축소가 그만큼을 **basis에 비례해** 둘에게서 거둔다. 그래서 막대는 언제나 "밴드 − 라벨 −
  간격"의 그 비율이고 — 막대끼리의 비율은 데이터 그대로다 — 값 라벨은 낮은 차트에서도 밴드
  안에 남는다. 라벨은 축소되지 않는다 (`shrink="0"`). 픽셀은 브라우저가 잰다. 소수 둘째 자리까지
  자르는 이유는 이 트리가 사람이 쓸 수 있는 표기여야 한다는 것이다.
- **축 최대가 데이터 최대와 같지 않다** (÷0.9): 가장 높은 막대의 값 라벨 위에 숨 쉴 자리를 남긴다.
- **기준선은 칸마다 있는 1px 도형이다** — `gap="0"`으로 붙으면 선이 이어져 축이 된다. 축 도형
  하나를 따로 놓으면 막대의 기준선과 축선이 다른 상자에 살게 되어 두 타깃에서 어긋난다.
- **라벨은 캡션 역할이다** — 테마가 그 크기·색을 이미 갖는다 (차트 전용 타이포그래피를 만들면
  테마가 두 벌을 든다). 문서 순서는 값 라벨들 → 범주 라벨들이다. 그림의 뜻은 뿌리 도형의
  `alt`가 한 문장으로 전한다.

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

`examples/decks/quarterly-review/slides/004_table.slide` — 두 채널을 함께 쓴 실물. 계열색이 막대
다섯을 배경으로 가라앉히고, 점의 `fill`이 가장 싼 채널과 가장 비싼 채널만 앞에 세운다:

```html
<chart type="bar" series="muted" max="24" alt="채널별 CAC — 검색 12.0, 추천 3.2, 소셜 9.4, 제휴 7.1, 디스플레이 광고 21.5 (천 원)" w="fill" grow="1" basis="0" h="300">
  <point label="검색" value="12.0"/>
  <point label="추천" value="3.2" fill="accent3"/>
  <point label="소셜" value="9.4"/>
  <point label="제휴" value="7.1"/>
  <point label="광고" value="21.5" fill="accent5"/>
</chart>
```

`<chart>`가 생긴 이유는 `incident-postmortem`의 탐지 시간 슬라이드(`008_detection.slide`, 지금은 계열 둘의 차트)다 — 손으로 쓰던 판이 도형·텍스트 **16개**였고 좌표
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
| `<point>`를 차트 밖에 썼다 | `<point> can only be a child of <chart> or <series>` |
| `<series>`를 차트 밖에 썼다 | `<series> can only be a child of <chart>` |
| `<chart>`에 점이 아닌 자식을 썼다 | `<chart> cannot contain <p> (available tags: <point>, <series>)` |
| `<chart>` 안에 문자 데이터를 직접 썼다 | `character data cannot appear directly inside <chart> (wrap it in <point>)` |
| `<point>`에 내용을 담았다 | `void tag <point> cannot have content (write <point/>)` |
| `<point>`를 `/` 없이 닫았다 | `void tag <point> must be self-closed (write <point/>)` |
| `<chart>`를 도형의 직속 자식으로 썼다 | `<shape> cannot contain <chart> (available tags: <text>, <h1>, <h2>, <h3>, <p>, <blockquote>, <pre>, <row>, <col>, <grid>, <stack>)` |
| `<chart>`에 `fill`을 썼다 | `unknown attribute fill on <chart> (available attributes: type, unit, max, series, alt, pad, w, h, min-w, min-h, max-w, max-h, ratio, grow, shrink, basis, span, self, order, at, x, y, style, slot, enter)` |
| `<point>`에 `w`를 썼다 | `unknown attribute w on <point> (available attributes: label, value, fill)` |

값 해석(`core.Resolve`) — 데이터와 값 어휘. 자리는 **저자의 것**이다: 합성 노드가 아니라
`<chart>`/`<point>`의 그 속성에 밑줄이 그어지고, 한 오타에 진단은 하나다:

| 조건 | 진단 메시지 |
|---|---|
| `type`이 없다 | `<chart> needs a type (available types: bar, line, area, step) — without a type there is no way to guess what to draw` |
| 어휘 밖 `type` | `unknown chart type "pie" (available types: bar, line, area, step)` |
| 선 · 영역 · 계단 차트의 `<point fill>` | `<point fill> highlights one bar — a line chart draws its series as one line, so a point cannot take its own color` |
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
| 점과 계열을 섞었다 | `<chart> holds either <point>s or <series>, not both — put every point inside a <series>` |
| 계열 형식에 프레임 `series`를 썼다 | `<chart series> colors the only series — with <series> each series takes its own fill` |
| 계열에 `name`이 없다 | `<series> needs a name — it is the legend entry` |
| 계열에 점이 없다 | `<series> has no <point> — there is no data to draw (<point label="May" value="22"/>)` |
| 계열의 점 수가 첫 계열과 다르다 | `<series name="B"> has 1 points but the first series has 2 — every series lists the same categories` |
| 계열의 범주 라벨이 첫 계열과 다르다 | `<point label="Q3"> is category 2 — the first series has "Q2" there (every series lists the same categories in the same order)` |
| 막대 차트의 계열에 `type`을 썼다 | `<series type> is for line, area and step charts — every series of a bar chart is bars` |
| 어휘 밖 계열 `type` | `unknown series type "bar" (available series types: line, area, step)` |
| `alt=""` | `alt cannot be empty — no name is the default, so remove the attribute (…)` |

`series`·`fill`의 색 값 진단은 [외관 속성](../appearance.md)·[`<shape>`](shape.md)와 **같은
문구**다 (`series: color token "brand" is not in the theme`) — 목록이 한 벌이므로 진단도 한 벌이고,
진단은 저자가 쓴 속성 이름으로 말한다.

막대가 박스를 넘치는 사고는 [배치 경고](../../guides/layout-warnings.md)가 말한다 — 펼친 트리가
평범한 도형·텍스트이므로 넘침 판정도 평범하게 돈다 (차트 상자가 라벨 두 줄도 담지 못할 만큼
낮으면 그 텍스트가 경고에 나온다).
