# TASKS — 도그푸딩에서 다시 세운 production-grade 로드맵

> **근거는 추측이 아니라 실측이다.** 이 문서는 `examples/balpyo-intro/`(SOML 12장을 balpyo로
> 직접 써서 웹 + PPTX로 빌드하고, PPTX를 LibreOffice → PDF → PNG로 되돌려 12장 전부 눈으로
> 검수한 기록)에서 **실제로 막힌 것만** 담는다. 쓰다가 부딪히지 않은 기능은 여기 없다.
>
> 판정 기준은 하나다: **"XML로 슬라이드를 쓰는 사람이, 결과를 열어보지 않고도 production 품질을
> 낼 수 있는가."** 어휘가 몇 개인지가 아니라, 저자가 (1) 쓰고 싶은 것을 쓸 수 있고 (2) 잘못을
> 빌드가 잡아주고 (3) 두 타깃이 같은 그림을 내는가로 잰다.
>
> 타깃은 웹(HTML)과 **편집 가능한** 네이티브 PPTX 두 개다. PDF 렌더는 out of scope.

---

## 도그푸딩 기록 (2026-09-20)

**한 것**: 12장 덱(표지 · 문제 · 스탯 · 파이프라인 · 프로젝트 구조 · 지오메트리 · 매핑 표 ·
테마 · 결정성 · 인용 · 현황 · 마무리)을 슬롯 모드와 흐름 모드를 섞어 쓰고, `grid`/`col`,
도형 프리셋, `pre`, 목록, 테마 토큰을 전부 동원해 작성 → 두 타깃 빌드(진단 0건) → PPTX를
PNG로 되돌려 12장 검수.

**통과한 것** (이 로드맵이 건드리지 않는 baseline):
빌드 파이프라인과 지오메트리 패스, 슬롯 → `p:ph` + `slideLayout` 컴파일, 도형·그림·겹침,
토큰 색의 `schemeClr` 보존, 11종 레이아웃 배치 CSS, 텍스트 서브셋, 진단이 가리키는 위치와
문구의 품질, 결정적 렌더 환경(픽스처 9종 · 골든 · 픽셀 비교).

**막힌 것** — 아래 표가 이 문서의 전부다. 심각도는 "저자가 우회할 수 있는가"로 매겼다.

| # | 증상 (실측) | 심각도 | 담는 항목 |
|---|---|---|---|
| D1 | 텍스트 안에 **리터럴 `<`를 쓸 수 없고** 문자 참조도 해석되지 않는다 (`&lt;`가 그대로 보인다). 마크업 언어를 설명하는 덱이 자기 문법을 화면에 띄울 수 없었다 | **높음** (우회 불가) | P0.1 |
| D2 | `<table>`은 파싱·검증까지 통과한 뒤 **렌더에서 빌드가 죽는다** (`<table>를 웹으로 렌더하는 매핑이 없다`). 표 한 장을 `grid` + 도형 14개로 손으로 짰다 | **높음** | P0.2 · P2.1 |
| D3 | 슬롯 카탈로그는 레이아웃 10종에 있는데 minimal 테마에는 `*.slots.css`가 3종(`content`·`title`·`two-column`)뿐이다. `layout="quote"` + `slot=`은 **웹에서 조용히 무효**(스테이지가 슬롯 모드로 안 가고 `grid-area`도 안 난다)인데 **PPTX에는 `p:ph`와 전용 `slideLayout`이 그대로 나간다** — 같은 소스가 두 타깃에서 갈린다 | **높음** | P1.4 |
| D4 | `title.slots.css`의 `justify-items: center`가 base 층 `.balpyo-text { width: 100% }`에 무력화된다. 표지가 왼쪽에 붙어서, 슬롯 자식마다 `align="center"`를 손으로 달아 고쳤다 | **높음** | P1.3 |
| D5 | `chevron` 중앙 텍스트가 왼쪽 노치에 **잘린다** (노치 꼭지가 도형 가로 중앙 = 텍스트 시작점). `pad`를 키워도 해결 불가 → 라벨을 도형 밖으로 내렸다 | 중간 | P1.2 |
| D6 | `<ul>`/`<ol>`에 외관 속성이 없다. 목록 글자 크기를 바꾸려면 레이아웃을 바꿔야 한다 (`<ul size="20">`은 파스 에러) | 중간 | P0.3 |
| D7 | 도형 안에 레이아웃 컨테이너를 넣을 수 없어, 카드 헤더/본문을 **도형 2개 + `gap="0"` `col`**로 흉냈다 | 중간 | P2.3 |
| D8 | 검수 루프가 전부 수동이다: 웹은 호스트, PPTX는 컨테이너, 그림 확인은 `soffice --convert-to pdf` + `pdftoppm`을 직접 조립했다. 한 장 고치고 다시 보기까지 네 단계 | 중간 | P3.1 · P3.2 |
| D9 | **오버플로 진단이 없다.** 글이 콘텐츠 박스를 넘치거나 열이 서로 붙어도 빌드는 조용하고, 열어보고 눈으로만 안다 (열 두 개가 붙어 보여 문장을 줄이는 작업을 세 번 반복했다) | **높음** | P1.1 |
| D10 | 페이지 번호·푸터·러닝 헤더가 없다 (`a:fld`/`slidenum` 미구현). 12장 덱에 장 번호를 넣을 방법이 없었다 | 중간 | P2.2 |
| D11 | LibreOffice 렌더에서 문장부호 앞뒤에 **간격이 벌어져 보인다** (`선언으로 , 색과`). 웹 렌더에는 없다. 런 분할/CJK 텍스트 레이아웃 중 무엇인지 미확인 | 중간 | P1.5 |
| D12 | accent 색이 **하나**라서 12장 전체가 한 가지 파랑이다. 강조의 위계를 만들 수 없었다 | 중간 | P4.1 |
| D13 | 예제·문서가 썩는다: `examples/quarterly-review/slides/004_table.slide` 주석은 "표는 어휘에 없다"라고 하고, `docs/index.md`는 "Iteration 5까지"라고 한다. **예제는 CI에서 빌드되지 않는다** | 중간 | P5.1 · P5.2 |

---

## 반복 절차 (모든 항목 공통)

항목마다 아래 네 체크박스가 곧 완료 조건이다. 기존 작업 방식을 유지한다.

- **① RED — 테스트**: 정상 + **에러/경고** 케이스. 파서 · 값 해석 · 웹 골든 · PPTX 구조(XML
  어서션) · PPTX 열림(LibreOffice) · 픽셀 비교 중 해당하는 것. 픽스처는 `testdata/fixtures/`.
- **② GREEN — 구현**: `soml/`(어휘) → `core/`(값) → `render/web/` · `render/pptx/`. 컨테이너에서
  `make test` 전부 통과.
- **③ REVIEW — 시각 확인**: `make review` 갤러리 + PowerPoint/Keynote로 직접 열어 편집 가능성과
  테마 추종 확인. 새 설계 결정은 ADR.
- **④ DOC — 문서**: `docs/reference/…`를 **구현된 동작만** 반영해 갱신. 예제는 픽스처에서 인용.
  새 페이지는 `mkdocs.yml` nav에 등록한다 — CI가 `--strict`로 빌드해 nav에 없는 문서를 에러로 잡는다.

> 추가 규칙 하나: **도그푸딩 덱이 회귀 테스트다.** 어휘를 건드리는 항목은 마지막에
> `examples/balpyo-intro`를 다시 빌드해 12장이 그대로 나오는지 본다 (P5.1이 이걸 자동화한다).

---

## P0 — 저자가 쓸 수 없는 것 (언어의 구멍)

우회가 불가능하거나, 우회하려면 산출물의 품질을 포기해야 하는 것들. 여기가 최우선이다.

### P0.1 문자 참조와 리터럴 `<` (D1)
마크업을 설명하는 덱이 마크업을 못 쓰는 것은 제품 모순이다. 코드·수식·비교 연산자·XML 예제가
전부 막힌다.
- [x] **선결정 ADR**: 세 선택지 — (a) 표준 문자 참조 5종(`&lt; &gt; &amp; &quot; &apos;`) +
      수치 참조 해석, (b) `<pre>`·`<code>`에 CDATA 섹션 허용, (c) 둘 다. 같이 확정할 것:
      **비-`<pre>` 텍스트에서 `<`가 태그 시작이 아닐 때의 렉서 규칙**(공백이 뒤따르면 문자로 볼지),
      그리고 **역방향 호환**(지금 `&lt;`를 리터럴로 쓰는 덱이 있는가 — 픽스처·예제 검사).
      → [ADR 0037](docs/decisions/0037-char-refs-require-semicolon-and-cdata.md): (c) 둘 다.
      참조는 `;`로 닫힌 모양일 때만 참조로 **읽고**(그래서 `R&D`·`?a=1&b=2`가 그대로 남아 역방향
      호환이 성립했다 — 픽스처·예제에 `&`가 하나도 없었다), 어휘 밖이면 에러다. `<`는 영문자·`/`·
      `<!--`·`<![CDATA[` 앞에서만 마크업이다. 접기 **다음**에 참조를 푼다(`&#32;`가 살아남는다).
- [x] RED: 렉서/파서 단위 테스트(참조 5종 + `&#x27E8;` + 잘못된 참조 진단), `<pre>` 안 CDATA에
      `<slide id="x">` 원문 보존, 웹 출력에서 **재이스케이프**되어 `&amp;lt;`가 아닌 `&lt;`가 나가는지,
      PPTX `a:t`에서 `<`가 XML 이스케이프되는지.
      → `soml/charref_test.go`(참조·CDATA·경계), `soml/lexer_test.go`(무손실 + 토큰 종류),
      `render/web/subset_test.go`·`render/pptx/shape_test.go`(한 번만 이스케이프).
- [x] GREEN: 렉서(`soml/lexer.go` `lexContent`) + 웹/PPTX 이스케이프 경로.
      → `soml/charref.go`(어휘 + 디코더), `lexer.go`의 `startsMarkup`·`textLen` + CDATA 토큰,
      `parse.go`의 `charDataValue`(문자 데이터·속성 값 공용). 이스케이프 경로는 이미 출력
      경계에 한 번만 있어(`html.EscapeString` · `escapeXML`) 손대지 않았다.
- [x] REVIEW: `examples/balpyo-intro`의 매핑 표(P2.1 이후 진짜 `<table>`)를 **실제 SOML 태그
      문법으로** 다시 쓴다 — 이 항목의 완료 증거는 "덱이 자기 문법을 보여준다"다.
      → 7장 매핑 표의 SOML 칸이 CDATA로 `<text>`·`<shape>`·`<row>`…를 원문 그대로 싣고, 3장이
      `&lt;grid cols="4"&gt;`를 쓴다. 두 타깃 빌드 진단 0건, PPTX를 PNG로 되돌려 확인했다.
      표 자체를 `<table>`로 바꾸는 것은 P2.1에 남는다 (표 렌더러가 없다).
- [x] DOC: `docs/reference/soml/text-subset.md`에 문자 참조 절(`pre`·`code` 포함) + `text.md` 갱신.
      → 「문자 참조」·「CDATA」 두 절 + 진단 표 4줄, `text.md`의 공백 규칙 옆 한 줄,
      `docs/reference/index.md` 현재 상태에 문법 한 줄. `--strict` 빌드 통과.

### P0.2 미구현 어휘는 **파싱 전에** 막는다 (D2)
표를 다 쓴 뒤 렌더 단계에서 죽는 것은 진단의 실패다. 어휘 카탈로그와 렌더러 능력이 갈라질 수
있는 구조 자체를 닫는다.
- [ ] RED: 두 렌더러가 다루는 kind 집합과 `soml`의 요소 카탈로그를 **대조하는 테스트**
      (`soml/subset.go`의 `ElementKind` 표 ↔ `render/web`·`render/pptx`의 디스패치 표). 카탈로그에
      있는데 어느 렌더러도 다루지 않는 kind가 있으면 실패 — `<table>`이 바로 이 상태였다.
- [ ] GREEN: 렌더러 디스패치를 표로 노출하고(맵 대신 명시 목록 — 카탈로그와 같은 태도),
      미구현 kind는 **파스 시점에** "아직 구현되지 않은 어휘다"로 진단. 위치는 태그가 있는 곳.
- [ ] DOC: `docs/internals/` — "어휘를 추가하면 렌더러 표를 먼저 늘린다" 규칙.

### P0.3 목록에 외관 속성 열기 (D6)
블록 5종(`h1–h3`/`p`/`blockquote`/`pre`)은 `size/color/align/anchor`를 받는데 `ul`/`ol`만 못 받는다.
저자가 이유를 알 수 없는 비대칭이다.
- [ ] **선결정**: `ul`/`ol`에 블록과 **같은** 외관 4종을 열고 `li`는 상속만 받는다(런 상속 규칙과
      동일). 불릿 글리프·들여쓰기 토큰화는 이 항목에서 다루지 않는다(별 항목).
- [ ] RED: 파서 속성 표 + core 캐스케이드(목록 → 항목 상속), 웹 CSS·PPTX `a:pPr`/`a:rPr` 어서션.
- [ ] GREEN: `soml/parse.go` 스펙 표 + `core` 해석 + 두 렌더러.
- [ ] DOC: `docs/reference/soml/text.md`의 속성 표.

---

## P1 — 화면이 거짓말하지 않게 (검수 가능성)

"열어보지 않아도 된다"를 만드는 마일스톤. **production-grade의 핵심은 어휘가 아니라 여기다.**

### P1.1 오버플로 · 스테이지 이탈 진단 (D9)
지오메트리 패스가 이미 모든 박스를 재고 있다 — 넘침을 아는 데 필요한 숫자가 전부 있는데도
아무 말을 하지 않는다.
- [ ] **선결정 ADR**: 무엇을 경고로 볼지 — (1) 콘텐츠 박스(1152×592) 이탈, (2) 스테이지
      (1280×720) 이탈, (3) 형제 박스 겹침(`stack`/`at`/`x,y`는 **의도된 겹침**이므로 제외),
      (4) 텍스트 클리핑(`scrollHeight > clientHeight`). 기본은 경고 + `--strict`로 에러 승격.
- [ ] RED: 넘치는 픽스처(긴 문장 · 작은 도형 안 텍스트 · 스테이지 밖 `x`) → 경고 진단 어서션,
      정상 픽스처 9종은 **경고 0건**(지금 덱과 예제가 조용해야 한다).
- [ ] GREEN: `render/geom`이 클리핑 여부까지 수집 → `pipeline`이 진단으로 승격. 메시지는 넘친
      양을 px로 말한다 ("본문이 콘텐츠 박스를 34px 넘친다").
- [ ] REVIEW: `examples/balpyo-intro`와 `quarterly-review`를 돌려 오검출 0인지 확인.
- [ ] DOC: `docs/reference/cli.md`(경고·`--strict`) + `docs/guides/` 저자 안내.

### P1.2 도형 프리셋의 텍스트 영역 (D5)
프리셋 12종은 저마다 글자가 들어갈 수 있는 영역이 다르다(`chevron` 노치, `arrow-right` 머리,
`callout` 꼬리, `cylinder` 상단 타원). 지금은 모든 프리셋에 균일 인셋을 쓴다.
- [ ] **선결정 ADR**: 프리셋별 텍스트 영역을 **soml의 닫힌 표**로 둔다(프리셋 목록과 같은 곳),
      OOXML 프리셋의 `rect` 정의를 기준으로 삼고 웹 인셋과 PPTX `bodyPr` 인셋이 **같은 표**를 읽는다.
- [ ] RED: 프리셋 12종 × 중앙 정렬 텍스트 픽스처 → 웹/LibreOffice 양쪽에서 글자가 도형 안에
      남는지 픽셀 비교, 표와 렌더러 인셋 일치 단위 테스트.
- [ ] GREEN: 표 + 웹(패딩/clip-path 보정) + PPTX(`bodyPr` 인셋).
- [ ] REVIEW: `examples/balpyo-intro` 04를 **라벨을 chevron 안으로 되돌려** 다시 검수 (이 항목의 증거).
- [ ] DOC: `docs/reference/soml/shape.md`에 프리셋별 텍스트 영역 표.

### P1.3 슬롯 정렬 계약 (D4)
테마가 `justify-items`/`align-items`로 쓴 정렬이 base 층 규칙에 먹힌다 — 즉 **테마가 배치를
소유한다는 약속이 깨져 있다.**
- [ ] **선결정 ADR**: `.balpyo-text { width: 100% }`를 슬롯/flow 모드에서 어떻게 다룰지 —
      (a) base에서 `width`를 빼고 흐름 모드 컨테이너가 `align-self: stretch`를 주기, (b) 슬롯
      모드에서만 `width: auto`, (c) 슬롯 영역 안에서 텍스트 정렬을 테마가 `text-align`으로 쓰기.
      **PPTX 쪽 결과가 갈리지 않는지**가 판정 기준이다(잰 박스가 바뀌면 `p:ph` 좌표도 바뀐다).
- [ ] RED: `title` 슬롯 덱의 잰 박스 스냅샷(중앙 정렬이면 좌우 여백이 대칭) + 웹 골든 + 픽셀 비교.
- [ ] GREEN: base/theme 층 규칙 조정 + 영향받는 `*.slots.css` 갱신.
- [ ] REVIEW: 표지에서 `align="center"`를 **다 지우고도** 가운데로 나오는지 (이 항목의 증거).
- [ ] DOC: `docs/reference/layouts.md`에 "정렬은 누가 소유하는가" 절.

### P1.4 슬롯: 카탈로그와 테마의 구멍을 닫는다 (D3) — **가장 조용히 위험한 것**
웹은 무효, PPTX는 플레이스홀더 발행. 두 타깃이 갈리는 것은 이 제품에서 가장 비싼 버그다.
- [ ] RED: 슬롯 카탈로그(10종)와 테마 `*.slots.css` 목록을 **대조하는 테스트** — 테마가 못 받는
      슬롯을 쓴 슬라이드는 경고, 그리고 **웹 슬롯 모드 판정과 PPTX 플레이스홀더 판정이 같은
      입력을 쓰는지** 어서션(지금은 각각 `Theme.SlotLayouts`와 `soml.LayoutSlots`를 본다).
- [ ] GREEN (택1을 ADR로): (a) minimal에 남은 7종 `*.slots.css`를 채운다, (b) 테마가 못 받는
      슬롯은 두 타깃 모두에서 무효 + 경고. **권장은 (a) + (b) 둘 다** — 카탈로그에 있는 슬롯은
      기본 테마가 다 받고, 외부 테마가 빠뜨린 경우는 경고로 잡는다.
- [ ] REVIEW: 10종 레이아웃 × 슬롯 덱 하나를 웹/PPTX 양쪽에서 열어 같은 자리인지.
- [ ] DOC: `docs/reference/layouts.md` 슬롯 표에 "테마가 받아야 뜻이 있다" 명시.

### P1.5 LibreOffice 문장부호 간격 조사 (D11)
- [ ] 조사: 같은 문장을 (1) 런 하나, (2) 지금처럼 분할된 런으로 내보내 LibreOffice 렌더를 비교.
      원인이 런 분할이면 **인접 런 병합**(같은 `rPr`이면 하나로), CJK 텍스트 레이아웃이면
      `a:pPr`의 동아시아 관련 속성(`eaLnBrk`/`hangingPunct`)을 명시.
- [ ] RED/GREEN: 원인에 맞는 테스트 + 수정. 골든 이동은 커밋에서 눈으로 검토.
- [ ] DOC: 원인이 렌더러 쪽이면 `docs/internals/`에 기록(PowerPoint와의 차이는 알려진 차이로 문서화).

---

## P2 — 실무 덱의 필수 어휘

### P2.1 `<table>` 렌더러 (D2) — 어휘는 있고 렌더러만 없다
문법·값 층·테마 `table` 컴포넌트·ADR 0036까지 끝나 있다. 남은 것은 두 렌더러와 측정이다.
- [ ] RED: 웹 골든(진짜 `<table>`) + PPTX 구조(`a:graphicFrame`+`a:tbl`, 명시 `tcPr`,
      `colspan`→`gridSpan` / `rowspan`→`rowSpan`+`hMerge`/`vMerge`) + 픽셀 비교. 픽스처 `09_table`.
- [ ] GREEN: `render/web`의 `writeTable`, `render/pptx`의 `writeGraphicFrameTbl`, 지오메트리
      패스의 셀 박스 수집(열 너비·행 높이).
- [ ] REVIEW: PowerPoint에서 셀을 클릭해 편집하고 행을 추가할 수 있는 **진짜 표**인지.
- [ ] DOC: `docs/reference/soml/table.md` + `examples/balpyo-intro/slides/007_pptx.slide`를
      손으로 짠 도형 표에서 `<table>`로 되돌린다 (이 항목의 증거).

### P2.2 덱 크롬: 페이지 번호 · 푸터 (D10)
- [ ] **선결정 ADR**: 어휘(`balpyo.json`의 `footer`/`pageNumber` vs 테마 `chrome` 선언 vs
      `<footer>` 태그)와 매핑 — PPTX는 마스터/레이아웃의 `a:fld type="slidenum"`, 웹은 스테이지
      크롬 요소. "표지·섹션 구분은 제외" 같은 규칙을 어디서 쓰는지도 같이 정한다.
- [ ] RED: `p:ph type="sldNum"` + `a:fld` 구조 어서션, 웹 골든, 제외 규칙 픽스처.
- [ ] GREEN: 마스터/레이아웃 파트 + 웹 렌더 + 테마 토큰(캡션 역할 재사용).
- [ ] REVIEW: PowerPoint에서 장을 추가하면 번호가 **자동으로** 따라오는지(정적 텍스트면 실패다).
- [ ] DOC: `docs/reference/theme.md`(신설).

### P2.3 `<shape>` 안에 레이아웃 컨테이너 (D7)
- [ ] **선결정 ADR**: 도형은 배경, 내부 배치는 자식 좌표 — 도형이 `row`/`col`/`grid`/`stack`을
      자식으로 받을 때의 측정·PPTX 매핑(도형 `p:sp` 1개 + 자식은 절대 좌표 형제)을 확정.
- [ ] RED: `<shape><col>…</col></shape>` 픽스처(카드 헤더/본문/푸터), 내부 좌표 스냅샷, 픽셀 비교.
- [ ] GREEN: 콘텐츠 모델 확장 + 측정 + 두 렌더러.
- [ ] REVIEW: `examples/balpyo-intro` 06을 도형 2개 흉내에서 카드 하나로 되돌린다.
- [ ] DOC: `docs/reference/soml/shape.md` 콘텐츠 모델.

### P2.4 코드 블록 하이라이팅 (`pre lang=`)
소프트웨어를 설명하는 덱은 코드를 띄운다. P0.1이 끝나면 비로소 의미가 생기는 항목.
- [ ] **선결정 ADR**: 토크나이저를 어디에 둘지(Go 라이브러리 vs 내장 최소 문법 집합), 색은
      **테마 토큰**으로만(accent 팔레트 확장 P4.1과 물린다), 지원 언어 1차 집합.
- [ ] RED: 토큰 → 웹 `span` + PPTX **런 분할**(`a:r` 여러 개, 색은 `schemeClr` 참조) 어서션.
- [ ] GREEN: `pre lang="…"` 속성 + 렌더.
- [ ] DOC: `docs/reference/soml/text-subset.md`의 `pre` 절.

### P2.5 `<chart>`
도그푸딩 덱에서 직접 막히지는 않았지만(숫자 한 개는 `stat`으로 충분했다), 실무 덱의 필수
어휘이므로 남겨둔다. **표(P2.1)보다 뒤다.**
- [ ] **선결정 ADR**: 네이티브 `c:chart` + 임베드 xlsx(편집 가능성 최대) vs 도형 합성. 데이터 문법.
- [ ] RED / GREEN / REVIEW(PowerPoint에서 데이터 편집) / DOC.

---

## P3 — 작성 루프 (도구) (D8)

12장을 쓰는 동안 가장 많은 시간을 먹은 것은 어휘가 아니라 **"고치고 다시 보기"**였다.

### P3.1 `balpyo watch` (또는 `server`) — 핫 리로드
- [ ] RED: 파일 변경 → 재빌드 → 브라우저 갱신의 단위 테스트(감시기·디바운스·진단 오버레이).
- [ ] GREEN: 웜 브라우저 세션 재사용(지오메트리 패스 시작 비용이 루프의 대부분이다) + 진단을
      화면에 띄우는 오버레이.
- [ ] DOC: `docs/reference/cli.md` + `docs/guides/getting-started.md`.

### P3.2 `balpyo preview --target pptx` — 손으로 하던 검수 루프를 제품으로
지금은 저자가 `soffice --convert-to pdf` + `pdftoppm`을 직접 조립해야 PPTX 결과를 본다.
`internal/testkit/soffice.go`가 이미 같은 일을 한다 — 테스트 전용으로 묶여 있을 뿐이다.
- [ ] RED: PPTX → PNG 산출물 존재·장수·결정성 테스트(컨테이너 전용, 호스트는 skip).
- [ ] GREEN: 리뷰 갤러리 생성기를 CLI로 승격(웹/PPTX 나란히 보기 + 차이 강조는 `make review` 재사용).
- [ ] DOC: `docs/reference/cli.md` + "PPTX 검수는 컨테이너가 필요하다"를 명시.

### P3.3 `balpyo init` · `balpyo check`
- [ ] `init`: 스캐폴딩(테마 선택 포함). 첫 덱까지 걸리는 시간이 채택을 정한다.
- [ ] `check`: 빌드 없이 진단만(P1.1의 경고를 CI에서 쓰는 입구). 종료 코드 규약을 문서화.
- [ ] DOC: `docs/reference/cli.md`.

---

## P4 — 테마를 디자인 시스템으로

### P4.1 accent 팔레트 확장 (D12)
- [ ] RED: `accent1..6` 픽스처 + 토큰 → PPTX `clrScheme` 12슬롯 매핑 어서션 + 색 변형
      (`shade/tint/alpha`)·그라디언트가 새 accent에서도 **참조를 보존**하는지.
- [ ] GREEN: `tokens.json` 팔레트 확장 + 테마 파트 컴파일.
- [ ] REVIEW: `examples/balpyo-intro`를 2~3색 위계로 다시 칠해 본다.
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

---

## P5 — 예제와 문서가 썩지 않게 (D13)

### P5.1 예제를 CI가 빌드한다 — **도그푸딩을 상시화한다**
- [ ] GREEN: `make examples`(모든 `examples/*`를 두 타깃으로 빌드, 진단 0건 + P1.1 경고 0건) →
      CI 잡에 추가. 예제 산출물은 커밋하지 않는다(`dist/`는 ignore).
- [ ] GREEN: `examples/balpyo-intro`를 **픽스처로도** 승격할지 판단 — 골든까지 걸면 어휘 변경마다
      12장 diff를 봐야 하므로, 우선은 "빌드 성공 + 경고 0"만 게이트로 둔다.
- [ ] DOC: `docs/internals/testing.md`에 예제 게이트 추가.

### P5.2 문서·예제 주석의 사실 정정
- [ ] `examples/quarterly-review/slides/004_table.slide`의 "표는 어휘에 없다" 주석 (P2.1 완료 시 함께).
- [ ] `docs/index.md`의 "Iteration 5까지" 상태 문구 → 현재 상태 또는 상태 문구 제거.
- [ ] `docs/index.md`에 도그푸딩 덱을 소개 링크로 넣는다(빌드해서 보는 방법 포함).

### P5.3 접근성: 도형·텍스트의 대체 텍스트와 읽기 순서
그림은 `alt` → `descr`이 되지만 도형·표는 대체 텍스트 경로가 없고, 읽기 순서(문서 순서)는
문서화되어 있지 않다.
- [ ] RED/GREEN: `alt`를 도형·표 프레임까지 확장(`cNvPr@descr`), 읽기 순서 = 문서 순서 어서션.
- [ ] DOC: `docs/reference/appearance.md` 또는 접근성 절.

---

## P6 — 배포

- [ ] 크로스 컴파일 릴리스(`goreleaser` 또는 직접) + GitHub Releases + Homebrew tap.
- [ ] `balpyo version`이 빌드 정보(커밋·날짜)를 말하게.
- [ ] 웹 런타임 마감(오버뷰·`reveal`·인쇄) + 인라인 JS를 `go:embed` 자산으로 분리.

---

## 우선순위와 판정선

**순서**: P0 → P1 → P2.1 · P2.2 → P3.1 · P3.2 → P2.3 · P2.4 → P4 → P5 → P2.5 → P6.

P0·P1이 먼저인 이유: 어휘를 더 늘려도 **저자가 결과를 신뢰할 수 없으면 production이 아니다.**
도그푸딩에서 잃은 시간의 대부분은 없는 태그가 아니라 (1) 쓸 수 없는 문자, (2) 조용한 오버플로,
(3) 두 타깃의 불일치, (4) 느린 검수 루프였다.

**production-grade 판정선** — 아래가 전부 `[x]`가 되면 "XML로 production 품질 덱을 쓴다"가 참이다.

- [ ] **자기 설명**: SOML 문법을 화면에 띄우는 덱이 SOML로 쓰인다 (P0.1)
- [ ] **손으로 짜는 것이 없다**: 표를 도형으로 흉내지 않는다 (P2.1)
- [ ] **조용한 실패가 없다**: 넘침·이탈·클리핑이 빌드 진단으로 잡히고, 미구현 어휘는 쓰는 순간
      막힌다 (P1.1, P0.2)
- [ ] **두 타깃이 갈리지 않는다**: 슬롯·정렬·도형 텍스트가 웹과 PPTX에서 같은 그림 (P1.2–P1.4)
- [ ] **한 명령으로 검수한다**: `balpyo watch` + `balpyo preview --target pptx` (P3.1, P3.2)
- [ ] **덱이 덱처럼 생겼다**: 페이지 번호·푸터·색 위계 (P2.2, P4.1)
- [ ] **예제가 썩지 않는다**: CI가 모든 예제를 경고 0으로 빌드한다 (P5.1)

---

## 열리는 설계 결정 (ADR 대기)

| 결정 | 항목 |
|---|---|
| 문자 참조 · CDATA · `<`의 렉서 규칙 | P0.1 |
| 어휘 카탈로그 ↔ 렌더러 능력의 단일 출처 | P0.2 |
| 오버플로 판정 기준과 경고/에러 승격 | P1.1 |
| 프리셋별 텍스트 영역 표(웹 인셋 ↔ `bodyPr`) | P1.2 |
| 정렬 소유권 (base `width: 100%` 처리) | P1.3 |
| 슬롯 구멍: 테마가 채우기 vs 경고 | P1.4 |
| 덱 크롬 어휘와 `a:fld` 매핑 | P2.2 |
| 도형이 컨테이너를 자식으로 받을 때의 좌표·매핑 | P2.3 |
| 코드 하이라이팅 토크나이저와 색 토큰 | P2.4 |
| 차트: 네이티브 `c:chart` vs 도형 합성 | P2.5 |
