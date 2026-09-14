# TASKS — production-grade 로드맵 (체크포인트 재계획)

> [PRD.md](PRD.md) 기준. **PDF 렌더는 out of scope.** 타깃은 웹(HTML)과 네이티브 PPTX 두 개.
>
> **2026-09-14 재계획.** 이전 계획은 "태그 세트 iteration" 순서로 어휘를 쌓는 것이었고,
> 그 대부분(I0–I6, I8)이 끝났다. 이 문서는 그 지점에서 **전체 체크포인트를 찍고**,
> 남은 일을 iteration 번호가 아니라 **"production-grade 덱을 만들 수 있는가"라는 목표에
> 대한 우선순위**로 다시 세운 것이다. 근거는 [`assessment.html`](assessment.html)의
> 실측 평가(`examples/quarterly-review/`를 직접 빌드해 얻은 마찰)다.

---

## 체크포인트 — 지금 동작하는 것 (baseline)

아래는 재계획 시점에 **구현·검증되어 있는** 능력이다. 새 작업은 이 위에 얹는다.

- **파이프라인**: `balpyo build`(웹+PPTX), 헤드리스 Chromium 지오메트리 패스 → EMU 변환,
  Docker 테스트 환경, 골든/리뷰 갤러리(`make test/review/golden`).
- **편집 가능한 네이티브 PPTX**: `<text>`→`p:sp`, `<shape>`→`p:sp`, `<image>`→`p:pic`+미디어
  파트, 슬라이드 마스터·레이아웃·테마·발표자 노트 포함. 이미지 기반 아님.
- **테마 색 보존**: 토큰 색 → `schemeClr` 참조(PowerPoint 테마 편집을 따라감). 검증됨.
- **어휘 (구현 완료)**: `<slide>` `<text>` `<note>`, 텍스트 서브셋(`h1–h3` `p` `ul/ol/li`
  `blockquote` `pre` `strong/em/code/a/br/span`), 레이아웃 `<row>` `<col>` `<grid>`
  `<stack>` `<spacer>` + 자식 배치(`grow/shrink/basis/span/self/order/at/x/y`),
  `<shape>` 12종 + 외관(`fill/border/radius/shadow/opacity/rotate/flip`), `<image>`,
  그라디언트 + 색 변형(`gradient()`/`shade/tint/alpha`), `layout="canvas"` 자유 배치.
- **테마 패키지 로딩(부분)**: `theme.json`/`tokens.json`/`components.json`을 `//go:embed`로
  읽어 토큰·컴포넌트를 캐스케이드에 연결. variant·역할·컴포넌트 기본값 동작.
- **발표 런타임(부분)**: 인라인 JS 발표자 뷰(한 장씩·스케일링·키보드·해시 딥링크·노트).
- **CLI**: `build`, `version`, `help`.

### 체크포인트에서 확인된 격차 (이 문서가 닫으려는 것)

| # | 격차 | 심각도 | 담는 마일스톤 |
|---|---|---|---|
| G1 | 슬라이드 레이아웃 CSS 11종이 **전부 빈 스텁** → `layout=`이 출력을 안 바꿈 | 높음 | M1 |
| G2 | 빌드 에러 진단이 **두 번 출력**됨 (`main.go` + `diags.Err()` 이중) | 낮음(쉬움) | M1 |
| G3 | **표(`<table>`)** 없음 — 실무 덱 최대 장벽 | 높음 | M2 |
| G4 | **차트(`<chart>`)** 없음 — 요소 자체가 없음 | 높음 | M2 |
| G5 | 테마 **1개 고정**(`minimal`), `theme` 필드 미사용, **accent 색 1개**뿐 | 높음 | M3 |
| G6 | `<shape>` 안에 레이아웃 컨테이너(`<col>` 등)를 못 넣음 → 카드 배치 우회 필요 | 중간 | M3 |
| G7 | **합성 컴포넌트(`<define>`/`<slot>`)** 없음 → 반복 카드 손복사 | 중간 | M4 |
| G8 | **아이콘(`<icon>`)** 없음 | 중간 | M4 |
| G9 | **커넥터/화살표(`<arrow>`/`<line>`)**, **`<group>`** 없음 | 중간 | M5 |
| G10 | 작성 워크플로 부재: `init`/`server`/`check`/`format`, 완성된 웹 런타임 | 중간 | M6 |
| G11 | `<raw>` 탈출구 + 래스터화 폴백 없음 | 낮음 | M5 |

---

## 반복 절차 (모든 action item 공통)

이전 계획의 작업 방식을 그대로 유지한다. 각 item은 아래 4단계를 밟고, 네 체크박스가
곧 완료 조건이다.

- **① RED — 테스트**: `testdata/fixtures/`에 정상 + **에러** 케이스 픽스처. 파서/외관해석/
  웹 골든/PPTX 구조(XML 어서션)/PPTX 열림(LibreOffice)/픽셀 비교 중 해당하는 것.
- **② GREEN — 구현**: `soml/` 어휘 → 덱 모델 → `render/web/` → `render/pptx/`. 컨테이너
  안에서 `make test` 전부 통과.
- **③ REVIEW — 시각 확인**: `make review` 갤러리 + PowerPoint/Keynote로 직접 열어
  편집 가능성·테마 추종 확인. 새 설계 결정은 ADR + 하단 로그.
- **④ DOC — 문서**: `docs/reference/…` 갱신(구현된 동작만). 예제는 실제 픽스처에서 인용.

> 문서 배치 규칙과 `docs/` 트리는 이전과 동일하다 (`docs/reference|guides|internals|decisions`).

---

## M1 — 지금 덱을 "제대로 보이게" 만들기 (기반 마감)

이미 있는 어휘로도 쓸만한 덱이 나오게 하는, 비용 대비 효과가 가장 큰 마일스톤.
새 태그가 거의 없어 빠르게 끝난다.

### M1.1 레이아웃 배치 CSS 실체 채우기 (G1) — **완료 (컨테이너 검증 끝)**
- 대상: `core/themes/minimal/layouts/*.css` 11종(현재 전부 주석 스텁).
- 플럼빙: `soml.Theme.Layouts`(정규화 배치 선언) 추가 → `core.Resolve`가 테마에서 실음 →
  `render/web`이 배치 CSS 있는 레이아웃에 `balpyo-layout-<이름>` 클래스 + 규칙(테마 층)을 냄.
  지오메트리 패스가 새 배치를 재므로 **PPTX 좌표는 자동으로 따라온다**.
- [x] RED: 웹 스테이지 클래스·배치 CSS 방출 테스트(`render/web/layout_test.go`), 배치 없는
      레이아웃엔 클래스 미부착 회귀, 테마 배치 정규화·경계 린트 통과(`core/theme_package_test.go`).
- [x] GREEN: `title`/`section-divider`/`quote`/`stat`/`end`(세로 가운데), `two-column`/
      `image-text`(2열 그리드), `image-full`(여백 0)을 배치 전용 속성으로 작성. `content`/
      `canvas`/`code`는 렌더러 기본(스택/자유 배치)을 쓰므로 의도적으로 빈 스텁 유지.
      린터는 이제 CSS 주석을 벗긴다(`stripCSSComments`). 호스트 테스트 전부 통과, 웹 골든 불변.
- [x] REVIEW: `examples/quarterly-review`를 재빌드 → 표지가 세로 가운데, 아젠다가 2열로
      나옴을 HTML에서 확인. **Finch 컨테이너에서 `make test` 전 패키지 통과(exit 0)** —
      `internal/testkit`(PPTX 지오메트리 + 픽셀 골든)·`render/geom` 포함. 호스트에서 어긋나던
      6개 픽스처(02–07)는 컨테이너에서 전부 초록 → 로컬 Chromium 폰트 메트릭 차이일 뿐이고
      M1.1 변경은 어떤 PPTX 골든도 깨지 않음을 확인. 웹 골든 불변.
- [x] DOC: `docs/reference/layouts.md` — 11종 카탈로그 + 배치/외관 경계 + 폴백 규칙.

### M1.2 슬롯 → PPTX 플레이스홀더 컴파일 (G1 후속)
- **설계 결정 (ADR 0035)**: 슬롯은 실제 OOXML `<p:ph>` 플레이스홀더가 되되 명시 xfrm/rPr을
  유지한다(선택지 3). 어휘·phType/idx 매핑은 soml이 소유(닫힌 카탈로그, layout ADR 0032와
  같은 태도), 시각 영역(grid-area)은 테마가 소유.
- **웹 쪽 결정 (추가형 슬롯 모드, 골든 안전)**: M1.1 문서 순서 흐름 배치는 그대로 두고, 슬롯
  배치는 슬라이드마다 켜지는 opt-in grid 모드로 얹는다. 스테이지가 `.balpyo-slots` 모디파이어를
  얻고, 테마는 `layouts/*.slots.css`에 `grid-template-areas`로 영역을 정의한다. 흐름 규칙은
  `:not(.balpyo-slots)`, 슬롯 규칙은 `.balpyo-slots`로 배타적으로 겨눈다.
- [x] RED/GREEN (soml 어휘 + core 검증): `soml/slot.go` 닫힌 카탈로그(레이아웃별 슬롯→phType/idx),
  `core/resolve.go` `resolveSlot`(직계 자식만·없는 슬롯·슬롯 없는 레이아웃 에러), `core/slot_test.go`.
- [x] RED/GREEN (웹 grid-area, 추가형 모드): `soml.Theme.SlotLayouts`, 테마 패키지 `loadSlotLayouts`
  (`*.slots.css`, 단방향 매니페스트 대조), 5개 writer의 `appendSlot`, `writeStage` 슬롯 모드 판정 +
  `balpyo-slots` 클래스, `features.slotLayouts` + `layoutStagesCSS` 영역 규칙 방출,
  `themes/minimal/layouts/{content,two-column,title}.slots.css`, `render/web/layout_test.go`의
  `TestSlotGridArea`/`TestSlotModeOffWithoutSlots`. **컨테이너 `make test` 전부 초록·골든 불변.**
- [x] RED/GREEN (PPTX 플레이스홀더): `render/pptx/shape.go`·`image.go`의 `slotPlaceholder`/`writeNvPr`로
  슬롯 단일 프레임 도형의 `<p:nvPr>`에 `<p:ph type/idx>` 주입(잰 xfrm·확정 rPr 유지), `layout_parts.go`의
  `deckSlotLayouts`가 슬롯 모드 레이아웃마다 `p:sldLayout` 파트 생성, 마스터 title/body 프로토타입 +
  `p:txStyles`, 슬라이드→레이아웃 관계 재배선. 전부 슬롯 사용 시에만 방출(골든 불변). `placeholder_test.go`
  4개 통과. 픽스처 `08_slots`(슬롯 content·two-column + 흐름 content 공존) + `make golden`.
- [x] REVIEW: **컨테이너 `make test` 전부 초록** — LibreOffice가 슬롯 덱을 열고 픽셀·골든 일치.
  슬라이드1(content)→slideLayout2, 2(two-column)→slideLayout3, 3(흐름)→slideLayout1, 마스터 3개
  sldLayoutId + txStyles 확인. **남은 것: PowerPoint에서 진짜 플레이스홀더로 잡히는지 수동 확인(사용자).**
- [x] DOC: `docs/reference/layouts.md`에 슬롯 표 추가(레이아웃별 슬롯→phType/idx + 추가형 모드 + 선택지 3 요지).

### M1.3 중복 진단 출력 버그 수정 (G2) — **완료**
- 원인: `cmd/balpyo/main.go`가 `result.Diagnostics`를 찍은 뒤, `pipeline.Build`가 돌려준
  `diags.Err()`(같은 진단을 합친 error)도 다시 찍는다.
- [x] RED: `cmd/balpyo/main_test.go`에 "각 진단이 정확히 1회" 어서션.
- [x] GREEN: 타입 있는 `soml.DiagnosticsError` + `errors.As`로 에러 경로에서 진단은 한 번만,
      돌려주는 에러는 개수 요약("빌드 실패: 에러 N건")으로 분리. `go test ./cmd/... ./soml/` 통과.
- [x] DOC: `docs/reference/cli.md`에 종료 코드·진단 출력 규약 명시(각 진단 1회 + `빌드 실패: 에러 N건` 요약).

---

## M2 — 데이터를 담는 어휘: 표 · 차트 (production 최대 장벽)

실무 덱의 필수. `examples/quarterly-review/`에서 표를 `grid`+`shape`로 손으로 그려야 했던
바로 그 문제. 둘 다 새 태그이므로 반복 절차를 정식으로 밟는다.

### M2.1 `<table>` (G3)
- 태그(안): `<table>` + `<tr>`/`<th>`/`<td>`(또는 `columns` 속성 스키마), 셀 병합
  (`colspan`/`rowspan`), 헤더 행, 셀 정렬·패딩, 테마 표 스타일(줄무늬/테두리 토큰).
- [x] **선결정 ADR (0036)**: 문법=HTML식 `<tr>/<th>/<td>`, 셀 콘텐츠=텍스트+인라인 전용(→`a:tc`
      1:1), PPTX=명시 `tcPr`+"스타일 없음" GUID(테마 표 스타일 상속 안 함, `tableStyles.xml` 불필요),
      프레임=도형·셀=잰 기하(열 너비/행 높이), 새 kind 3개(Table/Row/Cell), 셀 외관은 새 `table`
      컴포넌트 카테고리 캐스케이드. 병합 `colspan`→`gridSpan`/`rowspan`→`rowSpan`+`hMerge`/`vMerge`.
- [x] RED/GREEN (soml 어휘): `soml/table.go` 닫힌 카탈로그(`table`/`tr`/`th`/`td` 요소·kind 3개,
      프레임 `variant`, 셀 병합 `colspan`/`rowspan` + 외관 `fill`/`border`/`color`/`align`/`anchor`,
      배치 규칙 `tr∈table`·`th|td∈tr`), `soml/tablestyle.go` 확정 외관(`TableStyle`/`CellStyle`),
      `soml/table_test.go`(파스 트리·kind·속성 순서·배치 에러). 호스트 `go test ./soml/` 통과.
- [x] RED/GREEN (core 값 해석): 셀 캐스케이드(`core/cascade.go`의 `cellPatch`/`mergeCell`),
      테마 `table` 컴포넌트(`components.json` cell/header/stripe + `theme_package.go` 로더),
      `core/table.go` `resolveTable`/`resolveCell`(body 역할 바닥 → 테마 셀/헤더/줄무늬 → 인라인,
      프레임 `variant="striped"`가 홀수 본문 행 비헤더 셀에 줄무늬, 헤더/줄무늬 배타), 셀 안 인라인
      런 상속, `colspan`/`rowspan` ≥1 검증. `core/table_test.go`. 호스트 `go test ./core/` 통과.
- [ ] RED/GREEN (렌더): 웹 `writeTable`(진짜 `<table>`/CSS) + PPTX `writeGraphicFrameTbl`
      (`a:graphicFrame`+`a:tbl`, 명시 `tcPr`, 병합 → `gridSpan`/`rowSpan`+`hMerge`/`vMerge`) +
      지오메트리 패스 셀 박스 수집. 픽스처 `004_table` + 컨테이너 `make test`/`make golden`.
- [ ] REVIEW: PowerPoint에서 표 셀을 클릭해 편집·행 추가가 되는 진짜 표인지.
- [ ] DOC: `docs/reference/soml/table.md`.

### M2.2 `<chart>` (G4)
- 태그(안): `<chart type="bar|line|pie|…">` + 데이터(인라인 `<series>`/`<point>` 또는
  간단 데이터 속성), 축·범례·색(테마 accent 팔레트 참조).
- [ ] **선결정 ADR**: 지원 차트 종류 1차 집합, 데이터 표현 문법, **PPTX 네이티브 차트
      (`c:chart`+임베드 xlsx) vs 도형 합성** 중 택. 네이티브면 편집 가능성 최대(권장),
      대신 `xl/embeddings` 파트·`c:` 네임스페이스 구현 비용. 웹은 SVG/인라인.
- [ ] RED: 선택한 경로의 구조 어서션(네이티브면 `c:barChart` + 임베드 워크북, 합성이면
      도형 좌표), 데이터→기하 단위 테스트, 픽셀 비교.
- [ ] GREEN: 파서 + 웹 렌더 + PPTX. (데이터→좌표 계산은 웹/지오메트리 공유 스니펫.)
- [ ] REVIEW: PowerPoint에서 차트 데이터 편집이 되는지(네이티브 경로).
- [ ] DOC: `docs/reference/soml/chart.md` — 지원 종류·데이터 문법·편집 가능성 범위.

---

## M3 — 테마를 진짜 디자인 시스템으로 (브랜딩)

"테마는 CSS가 아니라 디자인 시스템"이라는 제품 약속을 실무 수준으로 끌어올린다.

### M3.1 테마 선택 + 다중 내장 테마 (G5)
- [ ] RED: `balpyo.json`의 `theme` 필드가 로더 선택을 바꾸는 테스트, 없는 테마 빌드 에러,
      내장 테마 2개 이상에서 같은 덱이 다르게 렌더되는 픽스처.
- [ ] GREEN: `core/resolve.go`의 하드코딩 `loadBuiltinTheme("minimal")` 제거 → `Config.Theme`
      로 선택. 내장 테마 최소 2~3종(예: `minimal` + 대비 강한 다크/코퍼릿) 추가.
- [ ] GREEN: 레거시 `core/theme.go`의 하드코딩 `minimalTheme()` 폴백 삭제.
- [ ] DOC: `docs/reference/theme.md`, `docs/guides/theming.md`(테마 만들기).

### M3.2 색 팔레트 확장 + 토큰 → PPTX 테마 파트 (G5 후속)
- 현재 색 토큰은 `fg/bg/muted/surface/accent1` 5개 — accent가 하나뿐.
- [ ] RED: 다색 accent(`accent1..6`) 픽스처, 토큰 → PPTX `clrScheme`/`fontScheme` 12슬롯
      매핑 XML 어서션, 웹 CSS 변수 대응.
- [ ] GREEN: `tokens.json` 팔레트 확장 + 토큰 → PPTX 테마 파트를 **패키지에서** 컴파일
      (지금은 부분). 색 변형/그라디언트가 새 accent에서도 참조 보존하는지 확인.
- [ ] DOC: `docs/reference/theme.md`의 표준 토큰 어휘 갱신.

### M3.3 외부 테마 패키지 + 상속 (`extends`)
- [ ] RED: 프로젝트 로컬 테마 디렉토리 로딩, `theme.json`의 `extends` 병합, `chrome`
      (마스터 고정 요소) 픽스처.
- [ ] GREEN: 외부 경로 테마 로더 + `extends` 병합 + `chrome` → 마스터 고정 요소.
- [ ] DOC: `docs/reference/theme.md`에 `extends`/`chrome` 절 추가.

### M3.4 `<shape>` 콘텐츠 모델에 레이아웃 컨테이너 허용 (G6)
- 지금 도형 안에는 텍스트류만 가능 → 카드 내부 정교한 배치가 어렵다.
- [ ] **선결정 ADR**: 도형이 레이아웃 컨테이너를 자식으로 받을 때의 좌표·측정·PPTX
      매핑(도형은 배경, 내부 배치는 좌표만) 확정.
- [ ] RED: `<shape><col>…</col></shape>` 픽스처, 도형 내부 그리드 좌표 스냅샷.
- [ ] GREEN: 콘텐츠 모델 확장 + 측정/렌더.
- [ ] DOC: `docs/reference/soml/shape.md` 갱신.

---

## M4 — 재사용과 밀도: 합성 컴포넌트 · 아이콘

### M4.1 `<define>` / `<slot>` 합성 컴포넌트 (G7)
- [x] **선결정 확정 (PRD §13-7)** — 파라미터는 속성 + `<slot>`까지, 제어 흐름 없음.
- [ ] RED: 파싱 시 코어 컴포넌트로 전개(덱 모델에 합성 흔적 없음), 이름 하이픈 강제,
      `components/*.slide` 로딩, 카드 그리드 픽스처(PRD §7.8).
- [ ] GREEN: define/slot 전개기.
- [ ] REVIEW: **8-카드 그리드를 `<grid>`+합성 컴포넌트로 재현 → `workspace/AI_FCD.pptx`
      원본과 비교** (수동 좌표 128회 → 0회 증명, PRD M0.5 목표 장면).
- [ ] DOC: `docs/reference/soml/composite.md`, `docs/guides/components.md`.

### M4.2 `<icon>` (G8)
- 태그(안): `<icon name="…">` — 아이콘 세트(SVG) → 웹 인라인 SVG / PPTX `custGeom`.
- [ ] **선결정 ADR**: 아이콘 소스(내장 세트 vs 사용자 SVG), 색 토큰 적용, PPTX 경로.
- [ ] RED: SVG → `custGeom` 어서션, 색 토큰 적용, 없는 아이콘 빌드 에러, 픽셀 비교.
- [ ] GREEN: 파서 + 웹 + PPTX.
- [ ] DOC: `docs/reference/soml/icon.md`.

---

## M5 — 다이어그램 어휘: 커넥터 · 그룹 · 탈출구

### M5.1 `<arrow>` / `<line>` (G9)
- [x] **선결정 확정 (PRD §13-4)** — 자동 앵커(최근접 변 중점) 기본, `from="db@top"`
      오버라이드, `via`는 논리 좌표점 목록.
- [ ] RED: `id` 참조 해석(없는 id 빌드 에러), 앵커 계산 단위 테스트, 배치 변경 시 연결
      유지, PPTX `cxnSp` 구조, 다이어그램 픽셀 비교.
- [ ] GREEN: **앵커 계산은 JS 한 벌** — 웹 런타임(SVG)과 지오메트리 패스(CDP)가 같은
      스니펫 공유. `label` → 선 중앙 텍스트.
- [ ] REVIEW: PowerPoint에서 커넥터를 끌면 도형에 붙어 있는지(진짜 `cxnSp`).
- [ ] DOC: `docs/reference/soml/arrow.md`, `docs/reference/soml/line.md`.

### M5.2 `<group>` (G9)
- [ ] RED: 그룹 → PPTX `grpSp`(자식 좌표계·오프셋), 그룹 단위 변형.
- [ ] GREEN: 파서 + 웹(래핑 div) + PPTX `grpSp`.
- [ ] DOC: `docs/reference/soml/group.md`.

### M5.3 `<raw>` 탈출구 + 래스터화 폴백 (G11)
- [ ] RED: `<raw>`가 웹은 그대로, PPTX는 CDP `captureScreenshot`으로 그림 파트가 되는지.
- [ ] GREEN: 래스터화 폴백(`captureScreenshot` 재사용). `targets.pptx.mode="image"`도 여기서.
- [ ] DOC: `docs/reference/soml/raw.md` — "raw는 PPTX에서 그림이 된다" 원칙.

---

## M6 — 작성 워크플로 (도구) (G10)

덱을 실무에서 반복 제작하려면 필요한, 어휘 밖 작업.

- [ ] `balpyo server` — 핫 리로드 + 검증 오버레이 + 웜 브라우저 세션. 시각 리뷰 루프를
      크게 가속하므로 **가장 먼저** 넣을 가치가 큼. → `docs/guides/getting-started.md`.
- [ ] `balpyo init` / `new` — 스캐폴딩(테마 선택 포함, M3 이후 자연스러움).
- [ ] 웹 런타임 완성 — 현재 인라인 JS(발표자 뷰·네비·딥링크·노트)에서 `reveal`·오버뷰
      추가 + vanilla TS(~10KB) 분리 → `go:embed`.
- [ ] `balpyo check` (CI 검증, 표면만) / `balpyo format` (트리비아 보존 파서 위 정규화).
- [ ] 릴리스 파이프라인: 크로스 컴파일 · GitHub Releases · Homebrew tap.
- [ ] CLI 서브커맨드를 추가할 때마다 `docs/reference/cli.md` 갱신.

---

## 우선순위 요약

1. **M1** (레이아웃 CSS·슬롯·진단 버그) — 있는 어휘로도 제대로 보이는 덱. 빠름, 최고 ROI.
2. **M2** (표·차트) — production 덱의 최대 장벽. 여기까지 오면 "실무 덱 가능" 판정이 바뀐다.
3. **M3** (테마 선택·팔레트·상속·도형 콘텐츠) — 브랜딩과 디자인 시스템 약속의 실현.
4. **M4** (합성 컴포넌트·아이콘) → **M5** (커넥터·그룹·raw) → **M6** (워크플로 도구).

> **production-grade 판정선**: M1 + M2 완료 시 "일반 비즈니스 덱 제작 가능", M3까지면
> "브랜드 덱 제작 가능", M4~M5까지면 "PRD 어휘 완성".

---

## 설계 결정 로그

**PRD §13의 열린 질문 12건은 2026-08-16에 전부 확정되어 PRD 본문에 반영되었다.**
아래 표는 구현 중 확정한 ADR(`docs/decisions/NNNN-*.md`)의 색인이다. 이번 재계획에서
새로 열리는 결정(표 문법·차트 경로·테마 선택·도형 콘텐츠 모델·아이콘 소스 등)은 해당
마일스톤 착수 시 ADR로 추가한다.

기존 ADR: 0001–0033 (I0–I8에서 확정). 전체 목록은 `docs/decisions/` 참조.

| 새로 열릴 결정 | 담는 마일스톤 |
|---|---|
| 표 문법 · 셀 캐스케이드 · `a:tbl` 매핑 | M2.1 |
| 차트: 네이티브 `c:chart`+임베드 xlsx vs 도형 합성 | M2.2 |
| 테마 선택 규약 · 다색 팔레트 표준 토큰 | M3.1 / M3.2 |
| 도형이 레이아웃 컨테이너를 자식으로 받을 때의 좌표·매핑 | M3.4 |
| 아이콘 소스(내장 세트 vs 사용자 SVG) · PPTX `custGeom` | M4.2 |
