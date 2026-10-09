# 0036 — `<table>`은 HTML식 `tr/td`이고, 셀은 텍스트 전용이며, 명시 좌표로 `a:tbl`이 된다

- 상태: 확정
- 날짜: 2026-09-14
- 마일스톤: M2.1 (`<table>` — TASKS.md)
- 관련: PRD §7(어휘), §9.3(computed-style 역추론 없음), [layouts.md](../../docs/reference/layouts.md),
  ADR 0007(블록 role은 core 소유), ADR 0011(레이아웃 컨테이너는 도형이 아니다),
  ADR 0016(도형의 텍스트는 도형 본문), ADR 0032(어휘는 core가 검사한다),
  ADR 0035(슬롯은 진짜 플레이스홀더 + 명시 오버라이드)

## 맥락

실무 덱의 필수 요소인 표가 어휘에 없다. `examples/quarterly-review/slides/004_table.slide`는
`<grid cols="3">` 안에 9개의 `<shape type="rect">`를 손으로 깔고 줄무늬·테두리를 셀마다
직접 칠해 표를 흉내낸다 — 행/열 모델이 없고, 헤더 행·병합·정렬이 전부 수작업이다. 이
보일러플레이트를 접는 것이 `<table>`의 목적이다.

OOXML의 표는 도형(`p:sp`)이 아니라 `<p:graphicFrame>` 안의 `<a:tbl>`이다 —
`<a:tblGrid>`(열 너비) + `<a:tr h="..">`(행) + `<a:tc>`(셀, `gridSpan`/`rowSpan`/`hMerge`/
`vMerge`로 병합) + `<a:tcPr>`(셀 채움·테두리·여백·정렬) + `<a:txBody>`(셀 텍스트). 지금
렌더러는 `p:sp`와 `p:pic`만 낸다 — `graphicFrame`/`a:tbl`은 처음이다. **핵심 긴장**은
balpyo의 계약(§9.3, ADR 0035)과 같다: 모든 좌표·외관이 명시적이고 브라우저가 잰 값이라
웹과 PPTX가 픽셀 단위로 같아야 하는데, OOXML 표는 `tableStyleId`로 **테마 표 스타일에서
줄무늬·테두리·글꼴을 상속**한다 — 그대로 두면 PowerPoint가 열자마자 갈라진다.

## 선택지

**문법** (셀 저작):
1. **HTML식 `<tr>/<th>/<td>`** — 기존 `<ul>/<li>`·`<h1>`·`<p>`·`<col>` HTML-서브셋 어휘와
   같은 결. 저자가 이미 안다. 병합은 `colspan`/`rowspan`. `a:tr`/`a:tc`에 1:1로 맞는다.
2. **데이터 우선 `columns=` 스키마** — `<table columns="지역,Q1,Q2">` + 행 데이터. 순수
   데이터엔 간결하지만 나머지 어휘와 이질적이고 병합·혼합 콘텐츠에 어색하다.

**셀 콘텐츠 모델**:
1. **텍스트 + 인라인 런** — 셀은 `<li>`처럼 텍스트와 인라인 런(굵게/강조)만 담는다.
   OOXML `a:tc`→하나의 `a:txBody`에 1:1.
2. **블록 콘텐츠 허용** — 셀에 `<p>`·리스트·이미지까지. 풍부하지만 `a:tc`는 프레임을
   중첩할 수 없어(이미지·다중 블록) 손실 합성이 필요하고 깔끔한 `a:tbl` 매핑·패리티가 깨진다.

**PPTX 표 스타일**:
1. **테마 표 스타일 상속** (`tableStyleId`→`tableStyles.xml`의 GUID) — OOXML 관례지만
   외관이 마스터/스타일에서 상속돼 명시-전부 계약이 깨진다 (ADR 0035의 선택지 2와 같은 병).
2. **"스타일 없음" + 명시 `tcPr`** — 내장 null 스타일 GUID를 가리키고(또는 생략) 모든
   채움·테두리를 셀마다 `a:tcPr`에 명시한다. 잰 값·해석된 외관이 그대로 이긴다.

## 결정

**문법은 HTML식 `<tr>/<th>/<td>`, 셀은 텍스트+인라인 전용, PPTX는 명시 `tcPr` + 스타일 없음.**

- **새 종류 3개** (ADR 0007·0011의 "kind가 네 층의 행동을 정한다" 교리 확장):
  `KindTable`(→`p:graphicFrame`+`a:tbl` 도형 하나), `KindTableRow`(→`a:tr`, 도형 아님),
  `KindTableCell`(→`a:tc`, 도형 아님, 텍스트 담는 단일 프레임). `<table>`은 슬라이드 직계
  자식이자 레이아웃 컨테이너 자식이 될 수 있는 도형-경계 요소다. `<tr>`은 `<table>`의,
  `<th>/<td>`는 `<tr>`의 직계 자식으로만 유효(중첩 위반 → `line:col` 진단, `<li>` 선례와 같다).
- **표 프레임이 도형, 셀은 잰 박스로 내부 기하를 채운다** (리스트·레이아웃 컨테이너 선례의
  변주): 웹은 진짜 `<table>`(또는 CSS 그리드)을 그리고, 지오메트리 패스가 `<table>` 박스로
  프레임 `xfrm`을, 셀 박스로 `<a:tblGrid>` 열 너비와 `<a:tr h>` 행 높이를 EMU로 낸다.
  셀은 `KindInline`이 아니므로 `geom.Refs`가 자동 수집하지만 **독립 `p:sp`가 되지 않는다** —
  프레임의 내부 격자를 채우는 용도로만 소비된다 (`<li>`가 재어지되 도형이 아닌 것과 같다).
- **병합**: `colspan`→`gridSpan`, `rowspan`→`rowSpan`. 병합에 덮이는 격자 셀은 `hMerge`/
  `vMerge` 빈 `a:tc`로 낸다 (OOXML 요구사항). v1은 직사각형 병합만.
- **헤더**: `<th>`는 `<td>`와 같은 `a:tc`이되 헤더 role의 외관(테마 표 컴포넌트의 header
  스타일)을 얻고, `a:tr`의 첫 행이면 `a:tbl`의 `firstRow="1"` 힌트를 켠다. 줄무늬는 테마가
  아니라 우리가 셀 `a:tcPr` 채움으로 명시한다.
- **셀 외관 캐스케이드** (ADR 0007·components.json 선례): 새 최상위 `"table"` 컴포넌트
  카테고리(shape과 평행) — header/body/stripe/border 기본값 + variants. 개별 색·테두리는
  기존 토큰(`accent1`·`surface`·`muted`)을 **참조**한다(새 색 토큰 없음). 셀별 오버라이드는
  `<td fill=.. align=..>`. 어휘(무엇이 유효한가)는 core, 룩(값)은 테마.
- **표 스타일은 명시 `tcPr` + 스타일 없음**: `<a:tableStyleId>`는 내장 "No Style, No Grid"
  GUID(`{5940675A-B579-460E-94D1-54222C63F5DA}`)를 가리키거나 생략하고, 모든 채움·테두리·
  여백·정렬을 셀마다 `a:tcPr`에 명시한다. 이러면 `tableStyles.xml` 파트가 필요 없다.

## 근거

- **HTML식 문법**은 어휘 일관성이다. balpyo는 이미 HTML 서브셋(`<ul>/<li>`·`<h1>`·`<p>`)이라
  `<tr>/<td>`가 가장 예측 가능하고, 병합(`colspan`/`rowspan`)·혼합 정렬이 자연스럽다.
  데이터 우선은 순수 숫자표엔 짧지만 나머지 어휘와 이질적이고, 표는 종종 헤더·병합·문장
  셀을 섞으므로 데이터 스키마가 금세 빠듯해진다.
- **셀 텍스트 전용은 정직한 매핑**이다 (ADR 0016·0035와 같은 논리): PowerPoint 셀은
  하나의 텍스트 흐름(`a:txBody`)만 담는다. 블록·이미지를 허용하면 `a:tc`가 프레임을 중첩
  못 하므로 손실 합성하거나 조용히 떨어뜨려야 하고, 그건 "웹=PPTX 픽셀 패리티" 계약을
  깬다. 텍스트+인라인은 셀↔`a:tc`를 1:1로 유지해 진짜 편집 가능한 표를 준다.
- **명시 `tcPr` + 스타일 없음**은 ADR 0035의 선택지 3과 같은 결정의 표 판이다. 표를
  `tableStyleId`로 테마 스타일에 맡기면(선택지 1) 파일을 열자마자 PowerPoint의 줄무늬가
  우리가 잰 외관을 덮어 갈라진다. 모든 셀 외관을 명시하면 잰 값이 이기고, 사용자가
  PowerPoint에서 표 스타일을 직접 바꾸는 것만이 유일한 "갈라짐"인데 그건 산출물이 아니라
  사용자 행동이다. 덤으로 `tableStyles.xml` 파트·콘텐츠 타입·관계를 안 만들어도 된다.
- **프레임=도형, 셀=잰 기하**는 리스트(하나의 텍스트 박스)와 레이아웃 컨테이너(도형 아님,
  자식만 좌표로 남음)의 선례를 잇는다. 표는 그 둘의 교차다 — 프레임은 도형이되 내부
  격자는 셀 박스에서 유도한다. `geom.Refs`가 셀을 이미 수집하므로 새 측정 기계가 필요 없고,
  소비 방식(좌표가 아니라 열 너비·행 높이로)만 새롭다.

## 결과

- `soml`: 새 `soml/table.go`(`TableElement`/`RowElement...`? — `<table>/<tr>/<th>/<td>` 이름
  상수, 셀 attr 목록 `colspan`/`rowspan`/`align`/`fill`/…). `soml/subset.go`에 `KindTable`/
  `KindTableRow`/`KindTableCell` + `buildElementKinds` 등록. `soml/parse.go`
  `buildElementSpecs`에 세 스펙(콘텐츠 모델: table=modelElements children=[tr], tr=
  modelElements children=[th,td], td/th=modelInline) + `slideChildren()`/`layoutChildren()`에
  `table` 추가 + `checkPlacement`에 tr/td 중첩 규칙.
- `core`: 셀 attr 값 → `Appearance`(정렬·채움·테두리, 테마 `table` 컴포넌트 캐스케이드),
  병합 검증(직사각형·격자 초과 금지 → 진단).
- `core/themes/minimal/components.json`: 새 `"table"` 카테고리(header/body/stripe/border
  기본값, 기존 토큰 참조). `core/theme_package.go`에 `tableComponentsFile` 구조체 +
  로드 경로, `core/theme.go`에 `tableComponentDefault` 평행 해석.
- `render/web`: `writeTable`(진짜 `<table>` 또는 그리드, `data-balpyo-id` 부여, 셀 CSS).
- `render/pptx`: `writeShapeNode`에 `KindTable` 분기 → 새 `writeGraphicFrameTbl` 이미터
  (`p:graphicFrame`+`a:graphic`+`a:graphicData uri` + `a:tbl`(`a:tblGrid`/`a:tr`/`a:tc`/
  `a:tcPr`/`a:txBody`) + 병합 `gridSpan`/`rowSpan`/`hMerge`/`vMerge` + 스타일 없음 GUID).
  프레임 `xfrm`은 잰 `<table>` 박스, 열 너비·행 높이는 잰 셀 박스.
- `render/geom`: 별도 코드 변경 없음 — 셀이 비인라인이라 `Refs`가 자동 수집, 표 이미터가
  셀 박스를 열 너비·행 높이로 소비.
- 픽스처: 헤더+본문+병합 정상 케이스, 없는 셀 위치(중첩 위반) 빌드 에러, `a:tbl`/`a:tr`/
  `a:tc`/`gridSpan`/`rowSpan` 구조 어서션, 웹 `<table>` 골든, LibreOffice 열림, 픽셀 비교.
  `examples/quarterly-review/slides/004_table.slide`를 `<table>`로 교체(보일러플레이트 소멸 증명).
- `docs/reference/soml/table.md`.
