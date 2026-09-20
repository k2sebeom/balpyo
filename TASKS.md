# TASKS — production-grade 로드맵 (도그푸딩 2회차 기준, 2026-09-20)

> **근거는 추측이 아니라 실측이다.** 이 문서는 덱을 직접 써 본 기록에서 **실제로 막힌 것만**
> 담는다 — 1회차 `examples/balpyo-intro`(12장), 2회차 `examples/incident-postmortem`(10장).
> 두 번 다 두 타깃으로 빌드하고 PPTX를 LibreOffice → PDF → PNG로 되돌려 전 장을 눈으로 봤다.
> 쓰다가 부딪히지 않은 기능은 여기 없다.
>
> 판정 기준은 하나다: **"XML로 슬라이드를 쓰는 사람이, 결과를 열어보지 않고도 production 품질을
> 낼 수 있는가."** 어휘가 몇 개인지가 아니라, 저자가 (1) 쓰고 싶은 것을 쓸 수 있고 (2) 잘못을
> 빌드가 잡아주고 (3) 두 타깃이 같은 그림을 내는가로 잰다.
>
> 타깃은 웹(HTML)과 **편집 가능한** 네이티브 PPTX 두 개다. PDF 렌더는 out of scope.

---

## 도그푸딩에서 막힌 것 (D1–D20)

심각도는 "저자가 우회할 수 있는가"로 매겼다. **상태**가 `닫힘`인 줄은 아래 「닫은 항목」에
증거가 있다.

| # | 증상 (실측) | 심각도 | 항목 | 상태 |
|---|---|---|---|---|
| D1 | 텍스트 안에 **리터럴 `<`를 쓸 수 없고** 문자 참조도 해석되지 않는다 (`&lt;`가 그대로 보인다). 마크업 언어를 설명하는 덱이 자기 문법을 화면에 띄울 수 없었다 | **높음** (우회 불가) | P0.1 | 닫힘 |
| D2 | `<table>`은 파싱·검증까지 통과한 뒤 **렌더에서 빌드가 죽었다**. 표 한 장을 `grid` + 도형 14개로 손으로 짰다 | **높음** | P0.2 · P2.1 | 진단만 닫힘 |
| D3 | 슬롯 카탈로그는 레이아웃 11종에 있는데 minimal 테마에는 `*.slots.css`가 3종뿐이다. `layout="quote"` + `slot=`이 **웹에서 조용히 무효**인데 **PPTX에는 `p:ph`가 그대로 나간다** — 같은 소스가 두 타깃에서 갈린다 | **높음** | P1.4 | **닫힘** (ADR 0044) |
| D4 | `title.slots.css`의 `justify-items: center`가 base 층 `.balpyo-text { width: 100% }`에 무력화된다. 표지가 왼쪽에 붙어서 슬롯 자식마다 `align="center"`를 손으로 달았다 | **높음** | P1.3 | **닫힘** (ADR 0043) |
| D5 | `chevron` 중앙 텍스트가 노치에 **잘린다**. `pad`를 키워도 해결 불가 → 라벨을 도형 밖으로 내렸다 | 중간 | P1.2 | 닫힘 |
| D6 | `<ul>`/`<ol>`에 외관 속성이 없다. 목록 글자 크기를 바꾸려면 레이아웃을 바꿔야 한다 | 중간 | P0.3 | 닫힘 |
| D7 | 도형 안에 레이아웃 컨테이너를 넣을 수 없어, 카드 헤더/본문을 **도형 2개 + `gap="0"` `col`**로 흉냈다 | 중간 | P2.3 | 열림 |
| D8 | 검수 루프가 전부 수동이다: 웹은 호스트, PPTX는 컨테이너, 그림 확인은 `soffice` + `pdftoppm`을 직접 조립했다 | 중간 | P3.1 · P3.2 | 열림 |
| D9 | **오버플로 진단이 없다.** 글이 박스를 넘쳐도 빌드는 조용하고 열어봐야 안다 | **높음** | P1.1 | 닫힘 |
| D10 | 페이지 번호·푸터·러닝 헤더가 없다 (`a:fld`/`slidenum` 미구현) | 중간 | P2.2 | 열림 |
| D11 | LibreOffice 렌더에서 문장부호 앞뒤에 **간격이 벌어져 보인다** (`선언으로 , 색과`). 웹에는 없다 | 중간 | P1.5 | 열림 |
| D12 | accent 색이 **하나**라서 덱 전체가 한 가지 파랑이다. 강조의 위계를 만들 수 없다 | 중간 | P4.1 | 열림 |
| D13 | 예제·문서가 썩는다. **예제는 CI에서 빌드되지 않는다** | 중간 | P5.1 · P5.2 | 열림 |
| D14 | **`<text>`에 `w`/`h`가 없다.** canvas 라벨은 자기 내용만큼만 넓어서 `align="center"`가 기댈 폭이 없다 — 라벨마다 보이지 않는 도형(`fill="none" border="none" w="140"`)으로 감쌌다. 라벨 10개가 태그 20개가 됐다 | **높음** | P1.3 · P2.5 | P1.3 닫힘 (ADR 0043) · `<chart>`는 P2.5 |
| D15 | `<pre>`에 **`lang`도 탭 폭도 없다.** gofmt 출력(탭)이 중첩 세 번에 x=350까지 밀려서 탭을 스페이스 두 칸으로 손으로 바꿨다 | 중간 | P2.4 | 열림 |
| D16 | **미구현과 오타의 진단이 구별되지 않는다.** `<table>`은 "아직 구현되지 않은 어휘다"라고 말해 주는데 `<pre lang="go">`는 그냥 "알 수 없는 속성"이다 | 중간 | P2.4 · P5.2 | 열림 |
| D17 | **레이아웃이 본문 높이를 소유하지 않는다.** `content`는 위에서부터 쌓기만 해서 10장 중 5장의 아래 40%가 비었다. 세로 리듬은 저자가 `<col h="fill" justify="between">` 랩퍼로 슬라이드마다 손수 만든다 (랩퍼 없음·`justify="center"`·`between` 셋 다 재봤다) | **높음** | P1.4 | **닫힘** (ADR 0044) |
| D18 | 같은 뜻의 이름이 갈린다: 컨테이너는 `justify`/`align`, 도형은 `anchor`다. 첫 시도가 `<col>에 알 수 없는 속성 anchor`로 죽었다 | 낮음 | P5.2 | 열림 |
| D19 | **저자가 처음 읽는 페이지가 틀렸다.** `slide.md`는 "`layout` 어휘는 2종"이라고 했고 `reference/index.md`는 `layouts.md`가 "Iteration 10에서 생긴다"고 했다 — 11종이 다 살아 있고 그 페이지도 이미 있다 | **높음** | P5.2 | 닫힘 |
| D20 | 도형으로 흉낸 표는 **머리행 아래 선이 없고** 본문 셀 테두리가 이웃과 2px로 겹친다. `border-collapse`를 흉낼 방법이 없다 | 중간 | P2.1 | 열림 |

**2회차에서 통과한 것** (이 로드맵이 건드리지 않는 baseline): 슬롯 모드 레이아웃 5종
(`title`·`code`·`two-column`·`quote`·`end`)이 웹·PPTX 양쪽에서 살아 있고, `cols`의 트랙
목록(`2fr 1fr 1fr 2fr`) · `grow`/`justify="between"` · canvas 좌표 · `fill="none"` 도형이 두
타깃에서 같은 자리에 떨어진다. 20장을 견줘 남은 차이는 D11(문장부호 간격)뿐이다.

---

## 닫은 항목

| 항목 | 닫은 것 | 증거 |
|---|---|---|
| P0.1 문자 참조와 리터럴 `<` (D1) | 표준 참조 5종 + 수치 참조 + `<pre>` CDATA. `;`로 닫힌 모양만 참조로 읽어 `R&D`가 그대로 남는다 | [ADR 0037](docs/decisions/0037-char-refs-require-semicolon-and-cdata.md), `soml/charref_test.go`, 소개 덱 7장이 자기 문법을 싣는다 |
| P0.2 미구현 어휘는 파싱 전에 막는다 (D2) | 어휘 카탈로그 ↔ 두 렌더러 디스패치를 **양방향** 대조. 미구현 kind는 파스 시점 진단 | `render/vocabulary_test.go`, `soml/unrendered_test.go`, `docs/internals/architecture.md` |
| P0.3 목록에 외관 속성 (D6) | `ul`/`ol`에 외관 4종, `li`는 상속만. 리뷰가 웹 마커 정렬 버그까지 잡았다 | [ADR 0038](docs/decisions/0038-list-containers-take-appearance.md), `02_text_subset/007` |
| P1.1 오버플로 · 이탈 진단 (D9) | 박스 포함 관계 + 줄 박스 가로 이탈 판정, `--check-layout` 경고 · `--strict` 승격. 클리핑 판정은 거짓 양성으로 버렸다 | [ADR 0039](docs/decisions/0039-overflow-warnings-use-measured-boxes.md), `render/geom/layout_test.go`, `docs/guides/layout-warnings.md` |
| P1.2 도형 프리셋의 텍스트 영역 (D5) | 프리셋 12종의 텍스트 영역을 `render/web` 기하표의 `TextRect` 열로. chevron은 음수 `bodyPr` 인셋으로 되돌린다 | ADR [0040](docs/decisions/0040-preset-text-rect-lives-in-the-web-geometry-table.md)·[0041](docs/decisions/0041-chevron-cancels-its-text-rect.md)·[0042](docs/decisions/0042-chevron-labels-stay-outside-the-shape.md), `04_shape/009`·`010` |
| P1.3 정렬과 텍스트 박스의 소유권 (D4 · D14) | 슬롯 모드가 세 번째 폭 리셋 문맥 + `<text>`에 가로 크기 3종. PPTX 렌더러는 손대지 않았다 | [ADR 0043](docs/decisions/0043-explicit-width-belongs-to-the-placement-context.md), `08_slots/004`(자식에 `align`이 없다), `06_stack/011` |
| P1.4 슬롯의 테마 구멍과 세로 리듬 (D3 · D17) | minimal이 슬롯 10종 전부에 `*.slots.css`를 갖고, 테마가 못 받는 슬롯은 **값 층에서 경고 + 미탑재**(두 렌더러가 한 입력을 읽는다). 세로 리듬은 `<slide justify>`가 소유한다 | [ADR 0044](docs/decisions/0044-slots-need-a-theme-and-vertical-rhythm-needs-an-owner.md), `render/pptx/placeholder_test.go`(두 타깃 모드 합의) · `core/theme_package_test.go`(카탈로그 ↔ 테마 대조), `03_layout/008` · `08_slots/005` |
| D19 문서가 어휘를 숨긴다 | `slide.md`(11종 표 + 실제 진단 문구) · `reference/index.md`(`layouts.md`는 구현됨) · `text.md`(속성 목록 + `<text>`에 `w`/`h`가 없다는 것) | `make docs --strict`, 커밋 `cc8239f` |

---

## 반복 절차 (모든 항목 공통)

항목마다 아래 네 체크박스가 곧 완료 조건이다.

- **① RED — 테스트**: 정상 + **에러/경고** 케이스. 파서 · 값 해석 · 웹 골든 · PPTX 구조(XML
  어서션) · PPTX 열림(LibreOffice) · 픽셀 비교 중 해당하는 것. 픽스처는 `testdata/fixtures/`.
- **② GREEN — 구현**: `soml/`(어휘) → `core/`(값) → `render/web/` · `render/pptx/`. 컨테이너에서
  `make test` 전부 통과.
- **③ REVIEW — 시각 확인**: `make review` 갤러리 + PPTX를 PNG로 되돌려 확인. 새 설계 결정은 ADR
  (`docs/decisions/`, 한국어, 맥락 → 선택지 → 결정 → 근거 → 결과. **쓴 뒤 고치지 않는다** —
  틀리면 다음 ADR이 고친다).
- **④ DOC — 문서**: `docs/reference/…`를 **구현된 동작만** 반영해 갱신. 새 페이지는
  `mkdocs.yml` nav에 등록한다 — CI가 `--strict`로 빌드해 nav에 없는 문서를 에러로 잡는다.

> 추가 규칙 둘:
> 1. **도그푸딩 덱이 회귀 테스트다.** 어휘를 건드리는 항목은 마지막에 `examples/*` 셋을 다시
>    빌드해 12 · 10 · 6장이 그대로 나오는지 본다 (P5.1이 이걸 자동화한다).
> 2. **저자 주석이 마찰 기록이다.** 우회로 짠 자리는 슬라이드 주석에 "마찰점"으로 남기고, 그
>    항목이 닫힐 때 주석과 우회를 함께 지운다 (그게 REVIEW의 증거다).

---

## 남은 항목 — **이 순서로 한다**

순서의 이유: **어휘를 더 늘려도 저자가 결과를 신뢰할 수 없으면 production이 아니다.** 2회차에서
잃은 시간도 없는 태그가 아니라 (1) 두 타깃이 갈리는 자리, (2) 세로 공간을 저자가 손으로 만드는
일, (3) 느린 검수 루프였다.

### P1.3 정렬과 텍스트 박스의 소유권 (D4 · D14) — **닫힘** (ADR 0043)
테마가 `justify-items`/`align-items`로 쓴 정렬이 base 층 `.balpyo-text { width: 100% }`에 먹힌다
— **테마가 배치를 소유한다는 약속이 깨져 있다.** 같은 규칙이 D14의 원인이기도 하다: 텍스트가
언제나 부모 폭이라 저자가 폭을 줄 수도, 내용 폭으로 둘 수도 없다.
- [x] **선결정 ADR** 0043: (나) 슬롯 모드가 **세 번째 리셋 문맥**이 된다 (`row`/`col`/`grid`·
      `stack`/canvas에 이어) + (B) 텍스트에 **가로 축만** 연다 (`w`·`min-w`·`max-w`) — 세로 축은
      `anchor`가 도형 안에서만 뜻을 가져 닿을 수 없는 공간이 생기기 때문이다.
- [x] RED: `soml/textbox_test.go`(어휘·부분집합) · `core/textbox_test.go`(값 → `Appearance.Box`) ·
      `render/web/textbox_test.go`(선언·순서·리셋 문맥 3종) + 픽스처 두 장
      (`08_slots/004_title_slots` — 자식에 `align`이 **하나도 없다**, `06_stack/011_canvas_labels`).
- [x] GREEN: 파서 표 3곳(`withTextSize`) + `core.resolveLayout`의 텍스트 분기 +
      `appendTextBox` + 슬롯 스테이지 폭 리셋 한 줄. **PPTX 렌더러는 손대지 않았다** — 잰 박스가
      바뀌면 좌표가 따라온다 (ADR 0043의 예측대로).
- [x] REVIEW (증거): 표지 두 장(`incident-postmortem`·`balpyo-intro` 001)에서 `align="center"`를
      **다 지웠고** 다섯 슬롯 자식의 중심이 전부 640px이다. `incident-postmortem` 008의 보이지
      않는 라벨 도형 10개를 지웠다 — 태그 20개 → 10개, 라벨 중심이 막대 중심과 같다.
- [x] DOC: `layouts.md` "정렬은 누가 소유하는가" 절(두 채널 + 문맥별 폭 표) + `text.md` 속성 표·
      진단 문자열 · `text-subset.md` · `index.md`.

### P1.4 슬롯: 카탈로그·테마의 구멍과 본문 높이 (D3 · D17) — **닫힘** (ADR 0044)
웹은 무효, PPTX는 플레이스홀더 발행. 두 타깃이 갈리는 것은 이 제품에서 가장 비싼 버그다.
- [x] **선결정 ADR** 0044: D3은 **(a)+(b) 둘 다** — minimal의 빈 7종을 채우고, 테마가 못 받는
      슬롯은 **값 층에서 경고 + 미탑재**(확정 모델에 싣지 않으면 두 렌더러가 `Appearance.Slot`
      하나를 읽는다). D17은 **(b)** — `<slide>`에 `justify`를 연다 (`gap`·`align`은 닫아 둔다).
- [x] RED: `render/pptx/placeholder_test.go`(웹 스테이지 클래스 ↔ `<p:ph>` 발행이 **같은 입력**을
      쓰는지 — 판정을 문서 전체 substring이 아니라 스테이지의 class 목록에서 읽는다) ·
      `core/theme_package_test.go`(슬롯 카탈로그 11종 ↔ 테마 `*.slots.css` 대조) ·
      `core/slot_test.go`(테마에 영역이 없으면 경고 + 미탑재).
- [x] GREEN (a): minimal에 `section-divider`·`quote`·`code`·`image-text`·`image-full`·`stat`·
      `end` 7종 `*.slots.css`. 표지 계열은 `justify-items: center`, 인용·구분선은 **가로 정렬을
      저자에게 남긴다**(`align="end"`가 뜻을 가져야 한다 — ADR 0043의 두 채널).
      `image-full.slots.css`는 `padding: 0`을 **다시 쓴다**: 흐름 규칙이 `:not(.balpyo-slots)`라
      물려받지 못하므로, 빠뜨리면 전면 그림이 조용히 64px 액자를 얻는다.
- [x] GREEN (b): `core.resolveSlot`이 테마의 슬롯 영역 유무를 보고 경고 + 미탑재.
      `render/web`은 `Theme.SlotLayouts` 조건을 **버렸고** `render/pptx`는 손대지 않았다 —
      두 렌더러가 읽는 입력이 하나가 됐다 (ADR 0044의 예측대로).
- [x] **D17**: `<slide justify>` — 파서 어휘 → 파이프라인 → 값 층(컨테이너와 **같은 6종**,
      슬롯 모드에서는 에러) → 웹 스테이지 인라인 `justify-content`. 픽스처 `03_layout/008`의
      잰 좌표가 64(위 끝)–656(아래 끝)이고 그 좌표가 그대로 PPTX `a:off`다.
- [x] REVIEW: 슬롯 10종을 한 장씩(슬롯 28개 전부 채움) 두 타깃으로 빌드해 견줬다 — 웹은 10장
      모두 `.balpyo-slots` + `grid-area` 28개, PPTX는 같은 28개가 `<p:ph>`로 나갔고 잰 좌표가
      영역과 맞는다 (`two-column` 564px 두 열 + 24px 간격 · `quote`·`stat`·`end` 블록 중심
      360px · `code` 캡션 620–656 · `image-full` 미디어 (0,0 1280×240) 전면). 이 리뷰가 **버그
      둘을 잡았다**: (1) `slot=`을 쓴 `<shape>`/`<image>`가 "엔진 버그" 진단으로 빌드를 멈췄다
      (`shapeAttrs`·`imageAttrs`의 exhaustive 검사가 슬롯을 면제하지 않았다 — media 슬롯을 가진
      두 레이아웃을 **저작할 수 없었다**), (2) 콘텐츠 박스 이탈 판정이 여백을 상수로 계산해
      전면 배치 레이아웃(`image-full`, 흐름 모드까지)에서 저자가 고칠 수 없는 경고를 냈다 —
      이제 스테이지의 패딩을 **재서** 판정한다 (`Geometry.Content`).
      D17은 `incident-postmortem` 002·003·004에서 `<col h="fill" justify="between">` 랩퍼를
      지우고 `<slide justify="between">` 한 선언으로 바꿨고, 세 장 모두 64–656 그대로다.
- [x] DOC: `layouts.md`("테마가 받아야 뜻이 있다" + "세로 리듬은 누가 소유하는가" 절 +
      minimal이 10종 전부를 갖는다) · `soml/slide.md`(`justify` 속성 절·진단 문자열 6줄) ·
      `reference/index.md` · `guides/layout-warnings.md`(콘텐츠 박스는 잰 값이다).

### P1.5 LibreOffice 문장부호 간격 조사 (D11) ← **지금 여기**
20장을 견줘 남은 **유일한** 두 타깃 차이다. 원인을 모르면 "두 타깃이 같다"를 말할 수 없다.
- [ ] 조사: 같은 문장을 (1) 런 하나, (2) 지금처럼 분할된 런으로 내보내 LibreOffice 렌더를 비교.
      원인이 런 분할이면 **인접 런 병합**(같은 `rPr`이면 하나로), CJK 텍스트 레이아웃이면
      `a:pPr`의 동아시아 속성(`eaLnBrk`/`hangingPunct`)을 명시.
- [ ] RED/GREEN: 원인에 맞는 테스트 + 수정. 골든 이동은 커밋에서 눈으로 검토.
- [ ] DOC: 원인이 렌더러 쪽이면 `docs/internals/`에, 뷰어 쪽이면 "알려진 차이"로 문서화.

### P2.1 `<table>` 렌더러 (D2 · D20) — 어휘는 있고 렌더러만 없다
문법·값 층·테마 `table` 컴포넌트·ADR 0036까지 끝나 있다. 남은 것은 두 렌더러와 측정이다.
- [ ] RED: 웹 골든(진짜 `<table>`) + PPTX 구조(`a:graphicFrame`+`a:tbl`, 명시 `tcPr`,
      `colspan`→`gridSpan` / `rowspan`→`rowSpan`+`vMerge`) + 픽셀 비교. 픽스처 `09_table`.
- [ ] GREEN: `render/web`의 `writeTable`, `render/pptx`의 `writeGraphicFrameTbl`, 지오메트리
      패스의 셀 박스 수집(열 너비·행 높이). `unrenderedKinds`에서 표 3종을 지운다.
- [ ] REVIEW: PowerPoint/LibreOffice에서 셀을 클릭해 편집하고 행을 추가할 수 있는 **진짜 표**인지.
- [ ] DOC: `docs/reference/soml/table.md` + 손으로 짠 표 둘을 `<table>`로 되돌린다:
      `balpyo-intro/007_pptx.slide`(도형 14개)와 `incident-postmortem/004_impact.slide`(도형 16개,
      **D20**의 머리행 선 없음·2px 겹침이 여기서 사라진다). 이게 이 항목의 증거다.

### P2.2 덱 크롬: 페이지 번호 · 푸터 (D10)
- [ ] **선결정 ADR**: 어휘(`balpyo.json`의 `footer`/`pageNumber` vs 테마 `chrome` 선언 vs
      `<footer>` 태그)와 매핑 — PPTX는 마스터/레이아웃의 `a:fld type="slidenum"`, 웹은 스테이지
      크롬 요소. "표지·섹션 구분은 제외" 규칙을 어디서 쓰는지도 같이 정한다.
- [ ] RED: `p:ph type="sldNum"` + `a:fld` 구조 어서션, 웹 골든, 제외 규칙 픽스처.
- [ ] GREEN: 마스터/레이아웃 파트 + 웹 렌더 + 테마 토큰(캡션 역할 재사용).
- [ ] REVIEW: 장을 추가하면 번호가 **자동으로** 따라오는지(정적 텍스트면 실패다).
- [ ] DOC: `docs/reference/theme.md`(신설).

### P3.1 `balpyo watch` — 핫 리로드 (D8)
- [ ] RED: 파일 변경 → 재빌드 → 브라우저 갱신의 단위 테스트(감시기·디바운스·진단 오버레이).
- [ ] GREEN: 웜 브라우저 세션 재사용(지오메트리 패스 시작 비용이 루프의 대부분이다) + 진단
      오버레이.
- [ ] DOC: `docs/reference/cli.md` + `docs/guides/getting-started.md`.

### P3.2 `balpyo preview --target pptx` (D8) — 손으로 하던 검수 루프를 제품으로
2회차에서도 PPTX를 보려고 임시 테스트 파일을 만들어 `soffice`+`pdftoppm`을 돌렸다.
`internal/testkit/soffice.go`가 이미 같은 일을 한다 — 테스트 전용으로 묶여 있을 뿐이다.
- [ ] RED: PPTX → PNG 산출물 존재·장수·결정성 테스트(컨테이너 전용, 호스트는 skip).
- [ ] GREEN: 리뷰 갤러리 생성기를 CLI로 승격(웹/PPTX 나란히 + 차이 강조는 `make review` 재사용).
- [ ] DOC: `docs/reference/cli.md` + "PPTX 검수는 컨테이너가 필요하다"를 명시.

### P3.3 `balpyo init` · `balpyo check`
- [ ] `init`: 스캐폴딩(테마 선택 포함). 첫 덱까지 걸리는 시간이 채택을 정한다.
- [ ] `check`: 빌드 없이 진단만(P1.1의 경고를 CI에서 쓰는 입구). 종료 코드 규약을 문서화.
- [ ] DOC: `docs/reference/cli.md`.

### P2.3 `<shape>` 안에 레이아웃 컨테이너 (D7)
- [ ] **선결정 ADR**: 도형은 배경, 내부 배치는 자식 좌표 — 도형이 `row`/`col`/`grid`/`stack`을
      자식으로 받을 때의 측정·PPTX 매핑(도형 `p:sp` 1개 + 자식은 절대 좌표 형제)을 확정.
- [ ] RED: `<shape><col>…</col></shape>` 픽스처(카드 헤더/본문/푸터), 내부 좌표 스냅샷, 픽셀 비교.
- [ ] GREEN: 콘텐츠 모델 확장 + 측정 + 두 렌더러.
- [ ] REVIEW: `balpyo-intro` 06을 도형 2개 흉내에서 카드 하나로 되돌린다.
- [ ] DOC: `docs/reference/soml/shape.md` 콘텐츠 모델.

### P2.4 코드 블록: `lang`과 탭 (D15 · D16)
소프트웨어를 설명하는 덱은 코드를 띄운다. P0.1이 끝나 이제 의미가 있다.
- [ ] **선결정 ADR**: 토크나이저를 어디에 둘지(Go 라이브러리 vs 내장 최소 문법), 색은 **테마
      토큰**으로만(P4.1과 물린다), 지원 언어 1차 집합.
- [ ] RED: 토큰 → 웹 `span` + PPTX **런 분할**(`a:r` 여러 개, 색은 `schemeClr` 참조) 어서션.
- [ ] GREEN: `pre lang="…"` 속성 + 렌더.
- [ ] **D15**: 탭 처리도 같이 정한다 — 저자가 손으로 스페이스로 바꾸지 않아도 되게 `tab-size`
      (웹)와 PPTX 대응(런에 탭을 두는지 펴는지)을 **테마가** 정한다(어휘를 늘리지 않는다).
- [ ] **D16**: 계획된 속성의 진단을 오타와 구별한다 — `<table>`처럼 "아직 구현되지 않은 어휘다"로
      말하는 자리를 **속성**에도 만든다. 예정 이름은 닫힌 목록이고, 목록 밖은 지금처럼
      "알 수 없는 속성"이다. (`lang`을 구현하면 이 목록의 첫 줄이 지워진다.)
- [ ] DOC: `docs/reference/soml/text-subset.md`의 `pre` 절.

### P4.1 accent 팔레트 확장 (D12)
- [ ] RED: `accent1..6` 픽스처 + 토큰 → PPTX `clrScheme` 12슬롯 매핑 + 색 변형·그라디언트가 새
      accent에서도 **참조를 보존**하는지.
- [ ] GREEN: `tokens.json` 팔레트 확장 + 테마 파트 컴파일.
- [ ] REVIEW: 예제 덱을 2~3색 위계로 다시 칠해 본다.
- [ ] DOC: `docs/reference/theme.md` 표준 토큰 어휘.

### P4.2 테마 선택 · 다중 내장 테마
- [ ] RED: `balpyo.json`의 `theme` 필드가 로더 선택을 바꾸는 테스트, 없는 테마 에러, 같은 덱이
      두 테마에서 다르게 렌더되는 픽스처.
- [ ] GREEN: `core`의 하드코딩 `minimal` 선택·레거시 폴백 제거 + 내장 테마 2~3종.
- [ ] DOC: `docs/reference/theme.md`, `docs/guides/theming.md`.

### P4.3 외부 테마 패키지 · `extends` · `chrome`
- [ ] RED: 프로젝트 로컬 테마 디렉토리 로딩, `extends` 병합, `chrome`(마스터 고정 요소) 픽스처.
- [ ] GREEN: 외부 경로 로더 + 병합 + 마스터 고정 요소(P2.2와 물린다).
- [ ] DOC: `docs/reference/theme.md`.

### P5.1 예제를 CI가 빌드한다 — **도그푸딩을 상시화한다** (D13)
- [ ] GREEN: `make examples`(모든 `examples/*`를 두 타깃으로, 진단 0건 + P1.1 경고 0건) → CI 잡.
      예제 산출물은 커밋하지 않는다(`dist/`는 ignore).
- [ ] GREEN: 예제를 **픽스처로도** 승격할지 판단 — 골든까지 걸면 어휘 변경마다 28장 diff를 봐야
      하므로, 우선은 "빌드 성공 + 경고 0"만 게이트로 둔다.
- [ ] DOC: `docs/internals/testing.md`에 예제 게이트 추가.

### P5.2 문서·예제 주석의 사실 정정 (D13 · D16 · D18)
- [ ] `examples/quarterly-review/slides/004_table.slide`의 "표는 어휘에 없다" 주석 (P2.1과 함께).
- [ ] `docs/index.md`의 "Iteration 5까지" 상태 문구 → 현재 상태 또는 상태 문구 제거.
- [ ] `docs/index.md`에 예제 덱 셋을 소개 링크로 넣는다(빌드해서 보는 방법 포함).
- [ ] **D18**: 컨테이너의 `justify`/`align`과 도형의 `anchor`가 같은 뜻의 다른 이름이라는 것을
      `layout.md`·`shape.md`가 서로 가리키게 한다 — 이름이 갈린 이유(CSS flex ↔ OOXML
      `bodyPr@anchor`)를 한 줄로 밝힌다. 어휘를 바꾸는 항목이 아니다.
- [ ] 문서가 코드와 갈리지 않게 **진단 문구를 테스트가 문서에서 읽게** 할지 판단한다 (지금은
      손으로 베껴 쓰고, D19처럼 조용히 썩는다). D16의 "예정된 속성" 목록도 같은 문제다.

### P5.3 접근성: 도형·텍스트의 대체 텍스트와 읽기 순서
- [ ] RED/GREEN: `alt`를 도형·표 프레임까지 확장(`cNvPr@descr`), 읽기 순서 = 문서 순서 어서션.
- [ ] DOC: `docs/reference/appearance.md`의 접근성 절.

### P2.5 `<chart>` (D14의 나머지)
2회차에서 막혔다: `incident-postmortem/008_detection.slide`의 막대 다섯 개가 좌표 손계산
**도형 6개 + 보이지 않는 라벨 도형 10개**다. 값 하나가 바뀌면 숫자 여섯 개를 다시 센다.
**표(P2.1)보다 뒤다.**
- [ ] **선결정 ADR**: 네이티브 `c:chart` + 임베드 xlsx(편집 가능성 최대) vs 도형 합성. 데이터 문법.
- [ ] RED / GREEN / REVIEW(뷰어에서 데이터 편집) / DOC.
- [ ] REVIEW의 증거: 008을 손계산 도형 16개에서 `<chart>` 하나로 되돌린다.

### P6 배포
- [ ] 크로스 컴파일 릴리스(`goreleaser` 또는 직접) + GitHub Releases + Homebrew tap.
- [ ] `balpyo version`이 빌드 정보(커밋·날짜)를 말하게.
- [ ] 웹 런타임 마감(오버뷰·`reveal`·인쇄) + 인라인 JS를 `go:embed` 자산으로 분리.

---

## production-grade 판정선

아래가 전부 `[x]`가 되면 "XML로 production 품질 덱을 쓴다"가 참이다.

- [x] **자기 설명**: SOML 문법을 화면에 띄우는 덱이 SOML로 쓰인다 (P0.1)
- [x] **조용한 실패가 없다**: 넘침·이탈이 빌드 진단으로 잡히고, 미구현 어휘는 쓰는 순간
      막힌다 (P1.1, P0.2)
- [ ] **손으로 짜는 것이 없다**: 표를 도형으로, 차트를 좌표로, 라벨을 빈 도형으로 흉내지
      않는다 (P2.1, P2.5, P1.3)
- [ ] **두 타깃이 갈리지 않는다**: 슬롯·정렬·문장부호가 웹과 PPTX에서 같은 그림 (P1.3–P1.5)
- [x] **세로 공간을 저자가 손으로 만들지 않는다**: 슬롯 모드는 테마가, 흐름 모드는
      `<slide justify>` 한 선언이 세로 리듬을 소유한다 (P1.4)
- [ ] **한 명령으로 검수한다**: `balpyo watch` + `balpyo preview --target pptx` (P3.1, P3.2)
- [ ] **덱이 덱처럼 생겼다**: 페이지 번호·푸터·색 위계 (P2.2, P4.1)
- [ ] **예제가 썩지 않는다**: CI가 모든 예제를 경고 0으로 빌드한다 (P5.1)

---

## 설계 결정 (ADR)

| 결정 | 항목 | 간 곳 |
|---|---|---|
| 문자 참조 · CDATA · `<`의 렉서 규칙 | P0.1 | ADR 0037 |
| 리스트 컨테이너의 외관 상속 | P0.3 | ADR 0038 |
| 어휘 카탈로그 ↔ 렌더러 능력의 단일 출처 | P0.2 | `internals/architecture.md` (ADR 없이 닫았다) |
| 오버플로 판정 기준과 경고/에러 승격 | P1.1 | ADR 0039 |
| 프리셋별 텍스트 영역 표(웹 인셋 ↔ `bodyPr`) | P1.2 | ADR 0040 (0041·0042가 chevron을 고쳤다) |
| 정렬 소유권 (base `width: 100%`) + `<text>`의 박스 속성 | P1.3 | ADR 0043 |
| 슬롯 구멍: 테마가 채우기 vs 경고 · 본문 높이의 소유자 | P1.4 | ADR 0044 (둘 다 — 테마를 채우고, 못 받는 슬롯은 값 층에서 경고 + 미탑재) |
| 덱 크롬 어휘와 `a:fld` 매핑 | P2.2 | 대기 |
| 도형이 컨테이너를 자식으로 받을 때의 좌표·매핑 | P2.3 | 대기 |
| 코드 하이라이팅 토크나이저와 색 토큰 · 탭 처리 · "예정된 속성" 진단 | P2.4 | 대기 |
| 차트: 네이티브 `c:chart` vs 도형 합성 | P2.5 | 대기 |
