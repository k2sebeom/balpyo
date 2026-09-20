# `<table>` — 표

`<table>`은 격자 데이터다 (PRD §7.6). 태그는 HTML을 그대로 빌린다 — 프레임이 `<table>`, 행이
`<tr>`, 셀이 `<th>`/`<td>`이고, 병합도 HTML의 이름(`colspan`·`rowspan`)을 쓴다. 두 타깃 모두
**진짜 표**가 된다: 웹은 `<table>` 요소이고 PPTX는 `p:graphicFrame` + `a:tbl`이다 — 도형을
손으로 쌓은 그림이 아니라 PowerPoint에서 셀을 클릭해 고치고 행을 더할 수 있는 표다
([ADR 0036](../../decisions/0036-table-is-html-ish-tr-td-with-explicit-a-tbl.md)).

표는 **세 번째 골격**이다. `<shape>`·`<text>`는 `p:sp`가 되고 [`<image>`](image.md)는 `p:pic`가
되지만, 표는 `p:graphicFrame`이다 (기하가 없고, 채움·선이 프레임에 없다 — 셀이 소유한다).
그래도 **도형 하나**를 소비한다: 프레임이 도형 번호 하나와 `spTree` 자리 하나를 쓰고, 행·셀은
그 안에 접혀 번호를 쓰지 않는다.

## 태그와 콘텐츠 모델

| 태그 | 담는 것 | 산출물 |
|---|---|---|
| `<table>` | `<tr>`**만** | 웹 `<table>` · PPTX `p:graphicFrame` + `a:tbl` (도형 1개) |
| `<tr>` | `<th>`/`<td>`**만** | 웹 `<tr>` · PPTX `a:tr` (도형 0개) |
| `<th>` | 문자 데이터 + [인라인 서식](text-subset.md) | 웹 `<th>` · PPTX `a:tc` (머리 외관) |
| `<td>` | 문자 데이터 + [인라인 서식](text-subset.md) | 웹 `<td>` · PPTX `a:tc` (본문 외관) |

- **셀은 텍스트 전용이다** — `<p>`·`<ul>`·`<shape>` 같은 블록은 셀 안에 올 수 없다. OOXML
  `a:tc`가 `a:txBody` 하나에 1:1로 맞기 때문이다: 블록을 열면 웹에서만 되는 표가 생긴다.
  인라인(`strong`·`em`·`code`·`a`·`span`·`br`)은 [`<li>`](text-subset.md)와 똑같이 쓴다.
- **`<tr>`/`<th>`/`<td>`는 표 밖에 올 수 없다** — `<li>`가 `<ul>`/`<ol>` 밖에 못 오는 것과
  같은 규칙이고, 진단 문구도 같은 모양이다 (아래 에러 표).
- `<table>`은 슬라이드 직계 자식, [레이아웃 컨테이너](layout.md)의 자식(`<col><table>`),
  그리고 [슬롯](../layouts.md)을 채우는 요소 셋 다 될 수 있다.

## 프레임 속성 — `<table>`

| 속성 | 허용 값 | 하는 일 |
|---|---|---|
| `variant` | `striped` | 테마 `table` 컴포넌트의 변형 — 홀수 본문 행에 줄무늬 채움 |
| `slot` | 레이아웃의 슬롯 이름 | 슬롯을 채운다 (웹 `grid-area` · PPTX `p:ph`) |
| 자식 배치 | `grow`·`shrink`·`basis`·`span`·`self`·`order`·`at`·`x`·`y`·`style` | 레이아웃 컨테이너·[`<stack>`](stack.md)의 자식으로서의 자리 |

- **크기 속성(`w`/`h`)이 없다.** 표는 **잰 내용이 곧 크기**다 ([`<ul>`](text-subset.md)과 같은
  판단): 프레임 `xfrm`은 잰 `<table>` 박스에서, 열 너비는 잰 셀 박스에서, 행 높이는 잰 `<tr>`
  박스에서 온다. 저자가 열 너비를 EMU로 적을 자리가 아니다 — 폭을 정하고 싶으면 표를 담은
  컨테이너의 폭(`<col w="480">`)이 그 일을 한다.
- **외관 속성도 없다** (`fill`·`border`·`shadow`·`radius`). 표의 외관은 **셀**이 소유한다 —
  프레임에 채움을 주면 셀 채움과 겹쳐 어느 쪽이 보이는지가 뷰어마다 갈린다.
- `variant`가 어휘 밖이면 **경고**이고 기본 외관으로 계속한다 (테마를 바꿔도 콘텐츠가 남는다는
  정책 — PRD §13-6). 셀에는 `variant`가 **없다**: 줄무늬는 프레임 하나가 켜는 행 채움이다.

## 셀 속성 — `<th>`/`<td>`

`<tr>`은 속성을 갖지 않는다 (행 단위 외관은 v1에 없다 — 줄무늬조차 셀 채움이다).

| 속성 | 허용 값 | 하는 일 |
|---|---|---|
| `colspan` | 1 이상 정수 | 가로 병합 — 웹 `colspan` · PPTX `gridSpan` |
| `rowspan` | 1 이상 정수 | 세로 병합 — 웹 `rowspan` · PPTX `rowSpan` |
| `fill` | 색 토큰 · `#rrggbb` · `none` | 셀 배경 — 웹 `background-color` · PPTX `a:tcPr > a:solidFill` |
| `border` | `<굵기> <스타일> <색>` · `none` | 셀 테두리 네 변 — 웹 `border-*` · PPTX `a:lnL/R/T/B` |
| `color` | 색 토큰 · `#rrggbb` | 셀 글자색 |
| `align` | `start` `center` `end` `justify` | 셀 글자의 가로 정렬 |
| `anchor` | `top` `center` `bottom` | 셀 글자의 세로 앵커 — 웹 `vertical-align` · PPTX `a:tcPr@anchor` |

값 어휘와 진단은 [외관 속성](../appearance.md)과 **한 벌**이다 — 도형의 `fill`/`border`와 같은
함수를 지나므로 같은 값이 자리마다 다르게 읽히지 않는다. 변별 테두리(`border-top`만)는 어휘에
없다: `border` 하나가 셀 전체를 정하고, 격자선은 이웃 셀이 같은 값으로 함께 그린다.

## 셀 외관은 캐스케이드다

셀은 **본문 텍스트**가 바닥이고 그 위에 테마 `table` 컴포넌트의 세 단이 얹힌다:

```
인라인 셀 속성 > (머리 셀이면 header 델타 | 줄무늬 행이면 stripe 채움) > 셀 기본값 > body 역할
```

내장 `minimal` 테마의 값이다:

| 단 | 값 | 받는 셀 |
|---|---|---|
| `cell` | `border: 1 solid muted` | 모든 셀 |
| `header` | `fill: accent1` · `color: bg` · `weight: 700` | `<th>` |
| `stripe` | `fill: surface` | `variant="striped"`의 **홀수 본문 행**의 비머리 셀 |

- **머리 셀과 줄무늬는 배타다** — `<th>`는 header 델타를 받고 줄무늬를 받지 않는다 (겹치면
  머리 강조색이 행마다 흔들린다).
- **머리행은 "모든 셀이 `<th>`인 행"이다.** 그래서 열 머리(첫 행 전체가 `<th>`)는 줄무늬에서
  빠지고, 행 머리(각 행의 첫 칸만 `<th>`)인 표는 그 행을 본문 행으로 세어 나머지 데이터 셀이
  줄무늬를 받는다.
- 줄무늬는 **0-index 홀수 본문 행**(1·3·5…)이다 — 첫 본문 행이 맨바탕이고 그다음이 얼룩이다.
- **세로 앵커의 기본값은 `top`이다.** 두 타깃 모두 값을 요구하는 채널이라(CSS `vertical-align`의
  UA 기본값은 `middle`, OOXML `bodyPr@anchor`도 값을 갖는다) 값 층이 정한다 — 렌더러가 각자
  기본값을 고르면 그 순간 두 타깃이 갈라진다.
- `none`은 **선언 없음이 아니라 명시**다: `border="none"`은 웹 `border-style: none`과 PPTX 네 변
  `a:noFill`로 나간다. 그래야 테마가 얹은 테두리를 지운 것이 조용히 되살아나지 않는다.

## 병합 — 첫 행이 열 수를 정한다

규칙은 HTML 표와 같다: **첫 행의 `colspan` 합이 표의 열 수**이고, 이후 행은 위에서 내려온
`rowspan`이 덮은 칸을 건너뛰며 왼쪽부터 채운다. 첫 행이 기준인 이유는 저자가 읽는 순서이기
때문이다 — 최대 행에서 역산하면 어느 행이 틀렸는지 말해 줄 수 없다.

격자가 직사각형을 정확히 덮지 않으면 **에러**다 (아래 에러 표). 웹 `<table>`은 깨진 격자를
조용히 복구하지만 PPTX `a:tbl`은 열 수와 행마다의 칸 수가 정확히 맞아야 열리므로, 막지 않으면
웹만 보고 넘긴 표가 PowerPoint에서 복구 대화상자를 띄운다.

## 예제

`testdata/fixtures/09_table/slides/001_basic.slide` — 머리행 하나 + 본문 세 행:

```html
<slide id="table-basic">
  <h2>지역별 분기 매출</h2>
  <table>
    <tr><th>지역</th><th>Q1</th><th>Q2</th><th>전분기 대비</th></tr>
    <tr><td>서울</td><td>12.4억</td><td>14.1억</td><td>+13.7%</td></tr>
    <tr><td>부산</td><td>6.8억</td><td>7.2억</td><td>+5.9%</td></tr>
    <tr><td>남미</td><td>3.1억</td><td>5.6억</td><td>+80.6%</td></tr>
  </table>
</slide>
```

`002_merge.slide` — 병합 머리행 (5열 6행 격자):

```html
<table>
  <tr><th rowspan="2">구분</th><th colspan="2">상반기</th><th colspan="2">하반기</th></tr>
  <tr><th>Q1</th><th>Q2</th><th>Q3</th><th>Q4</th></tr>
  <tr><td>서울</td><td>12.4</td><td>14.1</td><td>15.0</td><td>16.2</td></tr>
  <tr><td rowspan="2">기타</td><td>1.0</td><td>1.1</td><td>1.3</td><td>1.4</td></tr>
  <tr><td>0.9</td><td>1.0</td><td>1.1</td><td>1.2</td></tr>
</table>
```

`003_striped.slide` — 줄무늬 변형 + 셀 오버라이드 (테마를 이기는 자리):

```html
<table variant="striped">
  <tr><th>항목</th><th align="center">상태</th><th align="end">예산</th></tr>
  <tr><td>서버 증설</td><td align="center" fill="accent1" color="bg">진행</td><td align="end">2.4억</td></tr>
  <tr><td>보안 감사</td><td align="center">대기</td><td align="end">0.8억</td></tr>
  <tr><td>계약 갱신</td><td align="center" border="none" anchor="bottom">보류</td><td align="end">1.2억</td></tr>
</table>
```

`004_slots.slide` — 슬롯을 채우는 표 (표가 도형 하나이므로 슬롯의 주인이 될 수 있다):

```html
<slide id="table-slots" layout="content">
  <h1 slot="title">SLA 달성률</h1>
  <table slot="body">
    <tr><th>서비스</th><th>목표</th><th>실적</th></tr>
    <tr><td>API</td><td>99.9%</td><td>99.95%</td></tr>
    <tr><td>웹</td><td>99.5%</td><td>99.70%</td></tr>
  </table>
</slide>
```

슬롯을 채운 표는 **그 영역을 채운다** — 크기가 전부 잰 값이므로 그리드 영역이 늘려 놓은
폭·높이가 그 값이고, CSS 표는 늘어난 높이를 행들이 나눠 갖는다. 그 높이가 `a:tr@h`로 그대로
실리므로 두 타깃의 격자는 여전히 같은 격자다.

`005_in_container.slide` — 컨테이너 안의 표 둘 (각자 다른 폭을 잰다):

```html
<row gap="lg" align="start">
  <col gap="sm">
    <h3>등급별 건수</h3>
    <table>
      <tr><th>등급</th><th>건수</th></tr>
      <tr><td>P1</td><td>2</td></tr>
    </table>
  </col>
  <col gap="sm">
    <h3>평균 복구 시간</h3>
    <table variant="striped">…</table>
  </col>
</row>
```

## 렌더 결과

### 웹 — 진짜 `<table>`, 셀 외관은 전부 인라인

```html
<table class="balpyo-table" data-balpyo-id="s0.1" data-balpyo-el="table">
<tr data-balpyo-id="s0.1.0" data-balpyo-el="tr">
<th data-balpyo-id="s0.1.0.0" data-balpyo-el="th" style="font-family: var(--balpyo-font-body); font-size: 24px; font-weight: 700; line-height: 1.5; color: var(--balpyo-color-bg); text-align: start; background-color: var(--balpyo-color-accent1); border-width: 1px; border-style: solid; border-color: var(--balpyo-color-muted); vertical-align: top">지역</th>
…
</tr>
</table>
```

엔진 CSS는 표를 실제로 쓴 덱에만 나가고(안 쓴 골든이 움직이지 않게), **셀 하나로 표현할 수
없는 값**만 담는다:

```css
.balpyo-table { width: 100%; flex: 0 0 auto; border-collapse: collapse; table-layout: auto; }
.balpyo-table th, .balpyo-table td { padding: 8px 12px; }
```

- **CSS 그리드로 흉내지 않은 이유**는 둘이다: 스크린 리더가 행·열 관계를 읽고(시맨틱 태그를
  고른 것과 같은 근거), 브라우저의 표 알고리즘이 정한 **열 너비와 행 높이를 그대로 잰다**.
  격자 기하를 엔진이 계산하지 않는다 — 줄바꿈을 브라우저에 맡긴 것과 같은 선택이다.
- **`border-collapse: collapse`**는 이웃 셀의 테두리를 한 줄로 합친다. PPTX의 셀 테두리도
  격자선 하나를 공유하므로, 합치지 않으면 웹에서만 선이 2px로 보인다.
- **`table-layout: auto`**가 열 너비를 정한다 — 글자가 긴 열이 넓어진다 (`001_basic`의
  `전분기 대비` 열이 다른 열보다 넓은 것이 그 증거다).
- 셀 외관은 UA 기본값(`th { font-weight: bold; text-align: center }`)에 기대는 값이 하나도
  없다 — 굵기·정렬·색·세로 정렬까지 인라인 style이 전부 명시한다. `<th>`/`<td>`를 가르는
  이유는 **접근성**뿐이다 (머리 셀 인식).
- `variant`는 CSS 훅(`balpyo-table-striped`)으로만 나간다 — 줄무늬 **색**은 값 층이 셀 채움에
  이미 구워 놓았다.

### PPTX — `p:graphicFrame` + `a:tbl`

```xml
<p:graphicFrame>
  <p:nvGraphicFramePr>
    <p:cNvPr id="3" name="table s0.1"/>
    <p:cNvGraphicFramePr/>
    <p:nvPr/>
  </p:nvGraphicFramePr>
  <p:xfrm>
    <a:off x="609600" y="1283940"/>
    <a:ext cx="10972800" cy="2028825"/>
  </p:xfrm>
  <a:graphic>
    <a:graphicData uri="http://schemas.openxmlformats.org/drawingml/2006/table">
      <a:tbl>
        <a:tblPr firstRow="1">
          <a:tableStyleId>{5940675A-B579-460E-94D1-54222C63F5DA}</a:tableStyleId>
        </a:tblPr>
        <a:tblGrid>
          <a:gridCol w="1899493"/><a:gridCol w="2594075"/>
          <a:gridCol w="2594074"/><a:gridCol w="3885158"/>
        </a:tblGrid>
        <a:tr h="509588">
          <a:tc>
            <a:txBody>…<a:t>지역</a:t>…</a:txBody>
            <a:tcPr marL="114300" marR="114300" marT="76200" marB="76200" anchor="t">
              <a:lnL w="9525">…</a:lnL><a:lnR w="9525">…</a:lnR>
              <a:lnT w="9525">…</a:lnT><a:lnB w="9525">…</a:lnB>
              <a:solidFill><a:schemeClr val="accent1"/></a:solidFill>
            </a:tcPr>
          </a:tc>
          …
        </a:tr>
      </a:tbl>
    </a:graphicData>
  </a:graphic>
</p:graphicFrame>
```

- **격자 수치는 전부 잰 값이다**: `p:xfrm` ← 잰 `<table>` 박스, `a:gridCol@w` ← 잰 셀 박스의
  좌우 경계, `a:tr@h` ← 잰 `<tr>` 박스의 위아래 경계. 열 너비를 셀 **폭**이 아니라 **경계**에서
  뽑는 이유는 정확한 타일링이다 — 경계를 먼저 EMU로 환산하고 차분하면 열 너비의 합이 프레임
  폭과 **정확히** 같다 (칸마다 반올림하면 마지막 열이 프레임을 삐져나온다).
- **표 스타일을 가리키는 것이 "스타일 없음"의 선언이다.** `{5940675A-…}`는 내장 "No Style,
  No Grid"다 — 참조를 아예 빼면 뷰어가 `tableStyles.xml`의 기본 스타일을 얹고 그 색이 우리가
  명시한 셀 외관 위에 덮인다.
- **`firstRow="1"`은 첫 행이 전부 `<th>`일 때만 켠다.** 스타일이 없으므로 그림을 바꾸지 않고,
  PowerPoint의 "머리글 행" 체크와 스크린 리더의 머리 셀 인식만 켜는 값이다. **`bandRow`는
  켜지 않는다** — 줄무늬 색이 셀 채움에 이미 구워져 있어 힌트까지 켜면 뷰어에서 표 스타일을
  바꾼 순간 줄무늬가 두 벌로 겹친다.
- **`a:tcPr`는 명시다**: 여백 네 개(웹 셀 padding과 **같은 상수** — 12px = 114300 EMU,
  8px = 76200 EMU) + 세로 앵커 + 테두리 네 변 + 채움. 정하지 않은 채널도 명시한다 (채움 없음은
  `a:noFill`) — 비워 두면 뷰어가 그 자리를 채운다. 스키마 순서가 도형과 **반대**다: `a:tcPr`는
  선이 먼저고 채움이 나중이다 (`a:spPr`은 채움이 먼저다).
- **병합에 덮인 칸도 `a:tc`로 낸다** — `a:tbl`은 행마다 열 수만큼의 칸을 요구한다. 덮인 칸은
  빈 본문과 빈 `a:tcPr`에 표시만 싣는다 (그 자리의 외관·글자는 좌상단 셀이 소유한다):

```xml
<a:tc hMerge="1"><a:txBody><a:bodyPr/><a:lstStyle/><a:p/></a:txBody><a:tcPr/></a:tc>
<a:tc vMerge="1"><a:txBody><a:bodyPr/><a:lstStyle/><a:p/></a:txBody><a:tcPr/></a:tc>
```

- **표 하나가 도형 번호 하나를 쓴다** — 행·셀은 번호를 소비하지 않는다 (`005_in_container`의
  두 표 번호가 연달아 붙는다). `slot`을 쓴 표는 `p:nvPr`에 `p:ph`가 실리고, 좌표는 여전히 잰
  박스가 소유한다 (`p:xfrm`이 그대로 실린다).
- 셀 안 문단·런은 블록 텍스트와 **같은 조립기**를 지난다 — 표 안에서만 글자가 다르게 나갈 길이
  없다. 셀의 `a:bodyPr`는 비어 있다: 여백과 앵커가 `a:tcPr`에 사므로 같은 값을 두 곳에 적으면
  뷰어가 둘을 더한다.

## 빌드 에러가 나는 조건

파서(`soml.Parse`) — 이름·문법·콘텐츠 모델:

| 조건 | 진단 메시지 |
|---|---|
| `<tr>`을 표 밖에 썼다 | `<tr>는 <table>의 자식으로만 쓸 수 있다` |
| `<th>`/`<td>`를 행 밖에 썼다 | `<th>/<td>는 <tr>의 자식으로만 쓸 수 있다` |
| `<table>`에 행이 아닌 자식을 썼다 | `<table>의 자식으로 <p>는 올 수 없다 (쓸 수 있는 태그: <tr>)` |
| 셀 안에 블록을 썼다 | `<td> 안에 블록 태그 <p>는 올 수 없다` |
| `<table>` 안에 문자 데이터를 직접 썼다 | `<table> 안에 문자 데이터를 직접 쓸 수 없다 (<tr>로 감싼다)` |
| `<tr>` 안에 문자 데이터를 직접 썼다 | `<tr> 안에 문자 데이터를 직접 쓸 수 없다 (<td>로 감싼다)` |
| `<table>`에 `w`/`h`를 썼다 | `<table>에 알 수 없는 속성 w (쓸 수 있는 속성: variant, grow, shrink, basis, span, self, order, at, x, y, style, slot)` |
| `<tr>`에 속성을 썼다 | `<tr>에는 속성을 쓸 수 없다 (fill)` |
| 셀에 `variant`를 썼다 | `<td>에 알 수 없는 속성 variant (쓸 수 있는 속성: colspan, rowspan, fill, border, color, align, anchor)` |

값 해석(`core.Resolve`) — 격자와 값 어휘. 격자 진단은 **행 하나에 하나**다 (어느 셀이 틀렸는지는
값 층이 알 수 없다 — 칸 수만 안다):

| 조건 | 진단 메시지 |
|---|---|
| 첫 행이 비었다 | `표의 첫 행에 셀이 없다 — 첫 행이 표의 열 수를 정한다` |
| 행의 칸 수가 열 수와 다르다 | `이 행은 4칸인데 표의 열은 5칸이다 — 첫 행이 열 수를 정한다 (셀이나 colspan을 맞춘다)` |
| `colspan`이 위 행의 `rowspan`이 덮은 칸을 지난다 | `병합이 겹친다 — 2번째 열은 위 행의 rowspan이 이미 덮었다 (colspan 2이 그 칸을 지난다)` |
| `rowspan`이 마지막 행을 넘는다 | `rowspan 3은 표의 마지막 행을 넘는다 — 이 셀 아래 남은 행은 1개다` |
| `colspan`/`rowspan`이 0·음수·정수가 아니다 | `colspan는 1 이상 정수다 ("0")` |
| 어휘 밖 `variant` (**경고**) | `variant "zebra"가 테마에 없다 — 기본 외관으로 계속한다 (이 테마의 variant: striped)` |

`fill`·`border`·`color`·`align`·`anchor`의 값 진단은 [외관 속성](../appearance.md)·
[`<shape>`](shape.md)와 **같은 문구**다 — 목록이 한 벌이므로 진단도 한 벌이다.

병합이 든 표에서 셀 박스는 자기 행보다 높은 것이 정상이므로(`rowspan`) 셀은
[배치 경고](../../guides/layout-warnings.md)의 부모 박스 판정에서 빠진다. 표가 담기지 않는
사고는 그대로 말한다 — 그것은 프레임(`<table>`)이 부모나 콘텐츠 박스를 넘치는 모양으로 나타난다.
