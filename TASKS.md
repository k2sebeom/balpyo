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
| D2 | `<table>`은 파싱·검증까지 통과한 뒤 **렌더에서 빌드가 죽었다**. 표 한 장을 `grid` + 도형 14개로 손으로 짰다 | **높음** | P0.2 · P2.1 | **닫힘** (P2.1) |
| D3 | 슬롯 카탈로그는 레이아웃 11종에 있는데 minimal 테마에는 `*.slots.css`가 3종뿐이다. `layout="quote"` + `slot=`이 **웹에서 조용히 무효**인데 **PPTX에는 `p:ph`가 그대로 나간다** — 같은 소스가 두 타깃에서 갈린다 | **높음** | P1.4 | **닫힘** (ADR 0044) |
| D4 | `title.slots.css`의 `justify-items: center`가 base 층 `.balpyo-text { width: 100% }`에 무력화된다. 표지가 왼쪽에 붙어서 슬롯 자식마다 `align="center"`를 손으로 달았다 | **높음** | P1.3 | **닫힘** (ADR 0043) |
| D5 | `chevron` 중앙 텍스트가 노치에 **잘린다**. `pad`를 키워도 해결 불가 → 라벨을 도형 밖으로 내렸다 | 중간 | P1.2 | 닫힘 |
| D6 | `<ul>`/`<ol>`에 외관 속성이 없다. 목록 글자 크기를 바꾸려면 레이아웃을 바꿔야 한다 | 중간 | P0.3 | 닫힘 |
| D7 | 도형 안에 레이아웃 컨테이너를 넣을 수 없어, 카드 헤더/본문을 **도형 2개 + `gap="0"` `col`**로 흉냈다 | 중간 | P2.3 | **닫힘** (ADR 0049) |
| D8 | 검수 루프가 전부 수동이다: 웹은 호스트, PPTX는 컨테이너, 그림 확인은 `soffice` + `pdftoppm`을 직접 조립했다 | 중간 | P3.1 · P3.2 | **닫힘** (P3.1 `balpyo watch` — 저장하면 탭이 새로고침된다 · P3.2 `balpyo preview` — 웹/PPTX/diff 갤러리가 한 명령이다) |
| D9 | **오버플로 진단이 없다.** 글이 박스를 넘쳐도 빌드는 조용하고 열어봐야 안다 | **높음** | P1.1 | 닫힘 |
| D10 | 페이지 번호·푸터·러닝 헤더가 없다 (`a:fld`/`slidenum` 미구현) | 중간 | P2.2 | **닫힘** (P2.2 — 번호·푸터. 러닝 헤더는 같은 `chrome` 어휘에 `kind`를 더하면 열린다) |
| D11 | LibreOffice 렌더에서 문장부호 앞뒤에 **간격이 벌어져 보인다** (`선언으로 , 색과`). 웹에는 없다 | 중간 | P1.5 | **닫힘** (알려진 차이 — 뷰어의 CJK↔서양 자동 간격, 경계당 1/4 em. DrawingML로 끌 수 없다) |
| D12 | accent 색이 **하나**라서 덱 전체가 한 가지 파랑이다. 강조의 위계를 만들 수 없다 | 중간 | P4.1 | **닫힘** (ADR 0050 — 강조색 여섯, 번호는 위계가 아니다) |
| D13 | 예제·문서가 썩는다. **예제는 CI에서 빌드되지 않는다** | 중간 | P5.1 · P5.2 | 열림 |
| D14 | **`<text>`에 `w`/`h`가 없다.** canvas 라벨은 자기 내용만큼만 넓어서 `align="center"`가 기댈 폭이 없다 — 라벨마다 보이지 않는 도형(`fill="none" border="none" w="140"`)으로 감쌌다. 라벨 10개가 태그 20개가 됐다 | **높음** | P1.3 · P2.5 | P1.3 닫힘 (ADR 0043) · `<chart>`는 P2.5 |
| D15 | `<pre>`에 **`lang`도 탭 폭도 없다.** gofmt 출력(탭)이 중첩 세 번에 x=350까지 밀려서 탭을 스페이스 두 칸으로 손으로 바꿨다 | 중간 | P2.4 | 열림 |
| D16 | **미구현과 오타의 진단이 구별되지 않는다.** `<table>`은 "아직 구현되지 않은 어휘다"라고 말해 주는데 `<pre lang="go">`는 그냥 "알 수 없는 속성"이다 | 중간 | P2.4 · P5.2 | 열림 |
| D17 | **레이아웃이 본문 높이를 소유하지 않는다.** `content`는 위에서부터 쌓기만 해서 10장 중 5장의 아래 40%가 비었다. 세로 리듬은 저자가 `<col h="fill" justify="between">` 랩퍼로 슬라이드마다 손수 만든다 (랩퍼 없음·`justify="center"`·`between` 셋 다 재봤다) | **높음** | P1.4 | **닫힘** (ADR 0044) |
| D18 | 같은 뜻의 이름이 갈린다: 컨테이너는 `justify`/`align`, 도형은 `anchor`다. 첫 시도가 `<col>에 알 수 없는 속성 anchor`로 죽었다 | 낮음 | P5.2 | 열림 |
| D19 | **저자가 처음 읽는 페이지가 틀렸다.** `slide.md`는 "`layout` 어휘는 2종"이라고 했고 `reference/index.md`는 `layouts.md`가 "Iteration 10에서 생긴다"고 했다 — 11종이 다 살아 있고 그 페이지도 이미 있다 | **높음** | P5.2 | 닫힘 |
| D20 | 도형으로 흉낸 표는 **머리행 아래 선이 없고** 본문 셀 테두리가 이웃과 2px로 겹친다. `border-collapse`를 흉낼 방법이 없다 | 중간 | P2.1 | **닫힘** (P2.1 — 표는 셀이 격자선을 공유한다) |

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
| P2.1 `<table>` 렌더러 (D2 · D20) | 웹은 진짜 `<table>`, PPTX는 `p:graphicFrame`+`a:tbl`(표 스타일 없음 + 명시 `tcPr`). 격자 수치는 전부 **잰 값**이고 그 합이 프레임 크기와 같다. 예제 덱 셋의 손으로 짠 표가 표로 돌아왔다 | [ADR 0036](docs/decisions/0036-table-is-html-ish-tr-td-with-explicit-a-tbl.md), `09_table`(5장) · `internal/testkit/table_fixture_test.go`(병합 격자 전수), LibreOffice flat ODF 왕복(`table:table-cell` 78개 · 덮인 셀 4개), `docs/reference/soml/table.md` |
| P2.2 덱 크롬: 페이지 번호 · 푸터 (D10) | 어휘를 **나눠 갖는다**: 내용·켜짐은 `balpyo.json`(`footer`·`pageNumber`, 둘 다 기본 꺼짐), 자리·정렬·역할·제외는 테마 `chrome` 선언. 제외 규칙은 값 층이 적용해 두 렌더러가 확정 모델 하나를 읽는다. 번호는 `a:fld type="slidenum"`(캐시 숫자 동봉)이라 뷰어가 다시 센다 — `p:ph`도 `p:hf`도 쓰지 않았다 | [ADR 0045](docs/decisions/0045-deck-chrome-is-config-text-in-a-theme-declared-band.md), `10_chrome`(4장) · `internal/testkit/chrome_fixture_test.go`(잰 rect ↔ 골든 EMU 0 오차 · 다른 픽스처 골든에 크롬 0조각), LibreOffice flat ODF 왕복(`<text:page-number>`가 포함된 두 장에만 · 장을 끼우면 1/3/5로 따라온다), `docs/reference/theme.md` |
| P2.3 `<shape>` 안에 레이아웃 컨테이너 (D7) | 카드 한 장이 **도형 하나**다: 도형이 컨테이너를 담으면 **배경**이 되고(빈 `<a:p/>` + 인셋·앵커는 그대로) 자식은 잰 좌표를 가진 **형제** 도형이 된다 — `p:grpSp`를 쓰지 않아 "잰 rect가 곧 xfrm"이 그대로다. 모드는 **도형마다** 정해져 배경 카드 안의 배지가 흡수 모드로 남고, 판정의 단일 소유자는 `soml.ShapeHoldsLayout`이다. 배경 + `rotate`는 에러다(두 타깃이 다른 그림이 된다). 흡수 모드로는 불가능했던 것이 열렸다: 푸터를 `justify="between"`으로 아래에 붙이는 것, 카드 전체의 둥근 모서리, 테두리를 한 번만 그리는 것 | [ADR 0049](docs/decisions/0049-a-shape-that-holds-layout-becomes-a-backdrop.md), `11_shape_layout`(3장) · `render/pptx/preset_test.go`(배경은 `<a:p/>` 하나 · 자식이 형제 · `<p:grpSp>`가 없다) · `internal/testkit`(좌표 대조로 자식의 xfrm 전수 · 도형 안 줄 세기를 두 픽스처가 공유) · `core`의 `rotate` 거부, `balpyo-intro` 06이 카드 하나로 돌아왔다, `docs/reference/soml/shape.md` |
| P3.1 `balpyo watch` — 핫 리로드 (D8) | 폴링(250ms) + "스캔이 같아진 뒤 빌드"로 저장 한 번 = 빌드 한 번. `build`와 **같은 파이프라인·같은 출력 디렉토리**를 쓰고 리로드 클라이언트는 **응답 바이트에만** 주입한다(디스크는 배포본 그대로). 실패한 빌드는 산출물을 건드리지 않아 마지막 성공본 위에 진단 오버레이가 뜨고, 터미널에는 `build`와 같은 규칙으로 요약 한 줄이 간다. 측정이 필요한 감시는 브라우저 **하나**를 계속 쓴다 | [ADR 0046](docs/decisions/0046-watch-polls-the-project-and-injects-its-reload-client.md), `internal/watch`(스캐너·서버·루프) · `cmd/balpyo/watch_test.go`(응답에 있고 디스크에 없다) · `TestRunnerReusesOneWarmBrowser`(세션 동일성 · 0.61초), `docs/reference/cli.md#balpyo-watch` · `docs/guides/getting-started.md` |
| P3.2 `balpyo preview` — 검수 갤러리 (D8) | 래스터화(PPTX → PDF → PNG)·픽셀 diff·갤러리 문서를 `internal/preview`가 소유하고 `make review`와 사용자 명령이 **그것을 함께 쓴다** (테스트가 보는 그림과 사람이 보는 그림이 달라지지 않는다). 하네스에 남는 것은 **테스트 정책**뿐이다 — 건너뛰기/실패, 픽스처별 오차 조정표("이 슬라이드는 달라야 정답이다"까지 든 표). 그 표가 사용자 덱에는 없으므로 `preview`는 **게이트가 아니다**: 오차를 넘어도 코드 0이고, 1은 검수할 것이 안 만들어졌을 때만이다. 도구가 없으면 빌드 전에 멈춘다 | [ADR 0047](docs/decisions/0047-review-gallery-lives-in-internal-preview.md), `internal/preview`(`Run`·래스터라이저·갤러리) · `raster_pptx_test.go`(장수 · 스테이지 크기 · 재실행 0 픽셀 차) · `cmd/balpyo/preview_test.go`(경로를 찍는다 · 한 열 갤러리 · 도구 없음은 빌드 전에), `04_shape` 실물 확인(슬라이드 4장이 기본 2.5% 초과, 종료 코드 0), `docs/reference/cli.md#balpyo-preview` |
| P3.3 `balpyo init` · `balpyo check` — 첫 덱과 CI 입구 | `init`은 자기 덱을 **소유한다**(`internal/scaffold`의 임베드 템플릿 3장) — `examples/`는 마찰점을 일부러 밟는 회귀 테스트라 출발점이 못 된다. 계약은 파일 목록이 아니라 **다음 명령의 성공**이다: 스캐폴딩 덱이 `build --target html,pptx --check-layout --strict`를 경고 0으로 통과한다. `--force`는 없고(덮어쓸 대상이 저자의 원고다) 내장 아닌 `--theme`은 거부한다(`balpyo.json`의 `theme`을 아직 아무도 읽지 않는다 — P4.2). `check`는 **쓰지 않는 `build`**다: 같은 코드로 판정하고, 종료 코드도 같고(경고는 0을 타고 나간다), 출력 디렉토리를 만들지도 않는다 | [ADR 0048](docs/decisions/0048-init-owns-its-deck-and-check-is-build-without-writing.md), `core.BuiltinThemes`(어휘 단일 출처) · `internal/scaffold` · `pipeline.Check` · `cmd/balpyo/init_test.go`(`init`→`--strict` 빌드) · `check_test.go`(디렉토리 목록이 그대로 · `dist/`가 생기지도 않는다), 스캐폴딩 덱 `preview` 실물 확인(3장 0.625%/1.772%/0.581%), `docs/reference/cli.md#balpyo-init` · `#balpyo-check` · `docs/guides/getting-started.md` |
| P4.1 accent 팔레트 확장 (D12) | 강조색이 **여섯**이고, 눈이 아니라 **세 수치**가 고른 값이다 — `bg` 대비 4.5:1↑(대비율이 대칭이라 글자색과 면색을 한 수치로 보장한다) · 서로 색상 25°↑ · 휘도 한 띠(≤1.25). 실측 4.76~5.38 · 26° · 1.18이고, 그 규칙을 주석이 아니라 **테스트가 소유한다**. `clrScheme`에서 accent1을 빌려 가던 다섯 자리가 자기 색을 갖고(빌리는 자리는 링크 둘뿐) 폴백은 강조색 하나짜리 **다른** 테마를 위해 남는다. 번호는 위계가 아니라 색상환의 이웃 순서이고, **위계는 덱이 만든다** | [ADR 0050](docs/decisions/0050-six-accents-are-a-measured-palette-not-a-ranking.md), `core/palette_test.go`(WCAG 상대 휘도 · 대비 · 색상 간격 전수) · `core/theme_package_test.go`(값 드리프트) · `render/pptx.TestClrSchemeMinimalFillsSixAccents`(로드된 테마의 여섯 슬롯이 서로 다르다) · `12_palette`(2장 — 견본 여섯 + 변형/그라디언트, `TestPaletteFixtureBindsSixAccentSlots`, 슬라이드 파트에 리터럴 `srgbClr` 0건), `examples/incident-postmortem`을 **세 색**으로 다시 칠했다(피해=`accent5` · 우리가 늦은 곳=`accent4` · 대응=`accent3`, 파랑 0건 · 두 래스터 실물 확인), `docs/reference/theme.md`(표준 토큰 어휘 + 팔레트 규칙) · `appearance.md` |
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

### P1.5 LibreOffice 문장부호 간격 조사 (D11) — **닫힘** (알려진 차이)
20장을 견줘 남은 **유일한** 두 타깃 차이다. 원인을 모르면 "두 타깃이 같다"를 말할 수 없다.
- [x] 조사: 런 분할이 아니다 — 같은 문장을 런 하나로 낸 덱과 지금처럼 분할한 덱의 간격이
      **같다**. 원인은 뷰어의 CJK↔서양 자동 간격이다(ODF `style:text-autospace`, editeng의
      `SvxScriptSpaceItem`, 기본 켜짐): ODP로 바꿔 `none`을 넣으면 간격이 정확히 사라지고
      `ideograph-alpha`로 되돌리면 다시 나타난다. **DrawingML에는 끌 속성이 없다** —
      `eaLnBrk`·`hangingPunct`·`latinLnBrk`, `kern`·`spc`, 폰트 교체·폰트 요소 없음,
      `lang`/`altLang` 네 조합까지 열 가지를 재봤고 하나도 움직이지 못했다.
- [x] RED/GREEN: `internal/testkit/script_spacing_test.go`가 뷰어를 잰다 — 경계 0곳인 줄과
      11곳인 줄을 한 슬라이드에 두어 폰트 메트릭 잔차(−1.2%)와 스크립트 간격을 분리한다.
      모델: **경계당 +0.245 em**(경계 11곳·372px 줄에서 +14.5%) — 오차는 줄 길이가 아니라
      경계 수에 비례하고, 그래서 여유 4%로는 덮을 수 없다 (키우면 반대 방향으로 줄이 줄어든다
      — ADR 0005 선택지 3). 뷰어 쪽 원인이라 렌더러는 그대로 두고, **잰 폭을 지키는** 수정만
      했다: 스테이지가 `text-autospace: no-autospace`를 못 박는다 (CSS Text 4는 그 간격을
      초기값에 둔다 — 브라우저가 켜는 날 잰 폭이 움직이고 잰 폭이 곧 PPTX 좌표다). 골든
      9개의 이동은 CSS 주석과 그 한 선언뿐이고, 픽셀은 움직이지 않았다.
- [x] DOC: `docs/guides/known-differences.md`(신설 — 증상·원인·측정표·다른 뷰어·저자가 할 일) ·
      `guides/index.md` · `mkdocs.yml` 내비. ADR 0005의 "+3.3%"가 상한이 아니었음을 그 ADR의
      결과 절과 `pptx.textBoxSlackRatio`·`loLineWidthMaxRatio` 주석에 되먹였다.

### P2.1 `<table>` 렌더러 (D2 · D20) — 어휘는 있고 렌더러만 없다
문법·값 층·테마 `table` 컴포넌트·ADR 0036까지 끝나 있었다. 한 것은 두 렌더러와 측정이다.
- [x] RED: 웹 골든(진짜 `<table>`) + PPTX 구조(`a:graphicFrame`+`a:tbl`, 명시 `tcPr`,
      `colspan`→`gridSpan` / `rowspan`→`rowSpan`+`vMerge`) + 픽셀 비교. 픽스처 `09_table`
      (기본·병합·줄무늬·슬롯·컨테이너 5장). 구조 어서션은 `internal/testkit/table_fixture_test.go`가
      픽스처 **전체 빌드**를 놓고 본다: 행마다 열 수만큼의 `a:tc`가 서 있는지(덮인 칸의
      `hMerge`/`vMerge`를 센다 — 이 실패는 골든 diff에도 픽셀에도 보이지 않고 PowerPoint의 복구
      대화상자로만 나타난다), 모든 셀이 네 변과 앵커를 명시하는지, 격자 수치가 잰 박스이고 그
      합이 프레임 크기와 정확히 같은지.
- [x] GREEN: `render/web`의 `writeTable`, `render/pptx`의 `writeGraphicFrameTbl`, 지오메트리
      패스의 셀 박스 수집(열 너비·행 높이). `unrenderedKinds`에서 표 3종을 지웠다. 배치 판정은
      **셀을 부모 박스 판정에서 뺀다** — `rowspan`이 든 셀은 자기 행보다 높은 것이 정상이라
      병합이 든 표마다 저자가 고칠 수 없는 경고가 났다 (`render/geom/layout.go`,
      `TestDiagnoseTableCellMayOutgrowItsRow`).
- [x] REVIEW: LibreOffice로 `09_table` 덱을 열어 flat ODF로 되돌렸다 — `table:table-row` 23개,
      `table:table-cell` 78개, `table:covered-table-cell` 4개,
      `table:number-columns-spanned="2"` 2개, `table:number-rows-spanned="2"` 2개, 셀 글자 그대로.
      도형으로 흉낸 표라면 여기에 표가 하나도 없다 — 프레임이 **진짜 표**로 임포트된다는 뜻이고,
      그래서 셀을 클릭해 고치고 행을 더할 수 있다.
- [x] DOC: `docs/reference/soml/table.md`(+ nav · `reference/index.md`) + 손으로 짠 표 **셋**을
      `<table>`로 되돌렸다: `balpyo-intro/007_pptx.slide`(도형 14개 → 표 1개, 마지막 행이 자기
      자신을 설명한다) · `incident-postmortem/004_impact.slide`(도형 16개 → 표 1개, 슬라이드
      도형이 18개에서 3개로) · `quarterly-review/004_table.slide`(P5.2의 주석 항목). **D20**의
      머리행 선 없음·2px 겹침은 `border-collapse`와 공유 격자선으로 사라졌고, 세 덱 모두
      `--check-layout`이 조용하다.

### P2.2 덱 크롬: 페이지 번호 · 푸터 (D10)
- [x] **선결정 ADR** 0045: 어휘는 **나눠 갖는다** — 내용·켜짐은 `balpyo.json`(`footer` 문자열 ·
      `pageNumber` 불리언, 둘 다 기본 꺼짐), 자리·정렬·역할·제외는 테마 `chrome` 선언
      (`kind` 2종 · `align` 3종 · `role` · `skipLayouts`). `<footer>` 태그는 열지 않는다
      (크롬은 덱의 사실이고, 태그로 열면 스테이지 흐름에 들어가 D17을 되돌린다).
      PPTX는 **플레이스홀더가 아니라** 슬라이드 파트의 평범한 `p:sp` + `a:fld type="slidenum"`
      — 제외가 슬라이드 단위인데 흐름 모드는 `slideLayout1` 하나를 공유하고, 자동 갱신을 주는
      것은 `p:ph`가 아니라 `a:fld`다. 제외 규칙은 **값 층**이 적용해 확정 모델 하나를 만든다
      (ADR 0044의 "테마가 못 받으면 경고 + 미탑재"를 그대로 쓴다).
- [x] RED: 네 층에 각각 냈다 — `core/chrome_test.go`(설정 기본값 꺼짐 · 테마 선언 어휘 8종 ·
      슬라이드별 확정 목록과 번호 2·4 · 테마 침묵 시 경고 + 미탑재), `render/web/chrome_test.go`
      (스테이지의 **형제** div · 정렬 클래스 3종 · 외관 인라인), `render/geom`(띠 측정이
      `Geometry.Chrome` 사이드 테이블로 — `Boxes`는 `Refs`와 1:1을 유지한다),
      `render/pptx/chrome_test.go`(`p:ph` 없는 `p:sp` + `a:fld type="slidenum"` + 캐시된 숫자).
- [x] GREEN: 값 층이 제외 규칙을 적용해 **확정 모델 하나**(`Slide.Chrome`)를 만들고 두 렌더러가
      그것만 읽는다. 마스터에 `p:hf`를 내지 않았다 — 크롬을 켜지 않은 덱의 바이트가 움직이지
      않는다(`TestChromeIsAbsentFromOtherFixtureGoldens`가 골든 전수로 지킨다). 외관은 테마의
      `caption` 역할 재사용이고, **정렬만 선언이 덮는다**: 확정 모델에 정렬이 둘이면
      웹(`text-align`)과 PPTX(`algn`)가 다른 쪽을 읽어 같은 입력에서 갈라진다(D3의 모양 —
      커밋 `dc5b5ad`가 그 잠복을 골든에서 찾아 고쳤다).
- [x] REVIEW: 번호가 **정적 텍스트가 아니라는 것**을 두 가지로 확인했다. (1) LibreOffice flat
      ODF 왕복에서 포함된 두 장에만 `<text:page-number>`가 있고 그 안에 `<number>` 자리표시자가
      들어 있다 — 뷰어가 다시 센다. (2) 첫 장을 끼워 넣고 다시 빌드하면 캐시 숫자가
      1/3/5로 따라오고 제외된 장은 여전히 크롬이 없다.
- [x] DOC: `docs/reference/theme.md`(신설 — 패키지 네 파일의 소유 경계 + `chrome` 선언 표 +
      로더가 멈추거나 경고하는 조건 11줄) · nav · `reference/index.md` · `docs/index.md`의
      `balpyo.json` 예제에 `footer`·`pageNumber`.
- [x] 픽스처 `10_chrome`(4장: 표지 · 흐름 본문 · 구역 표지 · canvas): 픽셀 허용 오차를
      3.6%로 둔 근거를 **측정과 함께** 남겼다 — 띠 안(y≥656)의 차이는 1381px(슬라이드의
      0.15%)이고 나머지는 본문 글리프 잔차다(D11). 크롬의 자리는 그 비율이 아니라 **0 오차**로
      따로 못 박힌다: `TestChromeFixtureShapesMatchGeometry`가 잰 rect의 EMU와 골든
      `a:off`/`a:ext`를 전수 비교한다.

### P3.1 `balpyo watch` — 핫 리로드 (D8)
- [x] RED: 파일 변경 → 재빌드 → 브라우저 갱신의 단위 테스트(감시기·디바운스·진단 오버레이).
      네 층을 따로 못 박았다 — 스캐너(`cdda61b`) · 서버와 클라이언트 주입(`8864997`) ·
      루프(`9b9fe8c`) · CLI 표면(`90d5008`), 그리고 파이프라인이 주입된 측정기를 쓴다는
      것(`3e82f45`).
- [x] GREEN: 웜 브라우저 세션 재사용(지오메트리 패스 시작 비용이 루프의 대부분이다) + 진단
      오버레이. `5ebceb1`(스캐너) · `1682d44`(서버+클라이언트) · `b427256`·`532d3b3`
      (`pipeline.Options.Measurer` + 루프) · `94daf06`(서브커맨드) · `00905ab`(실패 요약) ·
      `fd1928d`. 웜 세션은 주장이 아니라 **측정**이다 —
      `TestRunnerReusesOneWarmBrowser`가 두 번의 `--check-layout` 빌드에서 세션 포인터
      동일성을 확인하고 0.61초에 끝난다(기동 1회분).
- [x] DOC: `docs/reference/cli.md` + `docs/guides/getting-started.md`. `9f04176`.
      시작하기는 `balpyo init`이 아니라 **예제 복사**에서 출발한다 (스캐폴딩은 P3.3이다) —
      그 이유를 문서 안에 적었다. `docs/index.md`의 빠른 시작도 아직 없는 `balpyo server`
      대신 `balpyo watch`를 쓴다.

### P3.2 `balpyo preview --target pptx` (D8) — 손으로 하던 검수 루프를 제품으로
2회차에서도 PPTX를 보려고 임시 테스트 파일을 만들어 `soffice`+`pdftoppm`을 돌렸다.
`internal/testkit/soffice.go`가 이미 같은 일을 한다 — 테스트 전용으로 묶여 있을 뿐이다.
- [x] RED: PPTX → PNG 산출물 존재·장수·결정성 테스트(컨테이너 전용, 호스트는 skip). `51d61ec`.
      `internal/preview/raster_pptx_test.go`가 장수(= 슬라이드 수)·PNG 크기(= 논리 스테이지)와
      **두 번 래스터화한 결과의 픽셀 0 차이**를 단정한다. 엔진을 `internal/testkit`에서
      `internal/preview`로 먼저 옮겼다(`b4f7cca`) — 릴리스 바이너리가 `testing`을 링크하지
      않아야 하므로 CLI가 쓸 코드는 하네스 밖에 있어야 한다.
- [x] GREEN: 리뷰 갤러리 생성기를 CLI로 승격(웹/PPTX 나란히 + 차이 강조는 `make review` 재사용).
      `c572b33`(`internal/preview.Run`) + `ac35197`(`balpyo preview`). `make review`와 사용자
      명령이 **같은 래스터화·같은 diff·같은 갤러리 문서**를 쓴다. `--target` 하나만 주면
      갤러리가 한 열이 되고(오차 배지도 없다), 도구가 없으면 **빌드 전에** 멈춘다.
- [x] DOC: `docs/reference/cli.md` + "PPTX 검수에 무엇이 필요한가"를 명시. `4c4c051`.
      원래 문구는 "컨테이너가 필요하다"였는데, 필요한 것은 **도구 둘**(`soffice`·`pdftoppm`)이고
      컨테이너는 그 둘을 얻는 **세 길 중 하나**다(설치 · `BALPYO_SOFFICE`/`BALPYO_PDFTOPPM` ·
      `make shell`). 사용자에게 "컨테이너를 쓰라"고 말하면 빌드가 바이너리 하나로 끝난다는
      계약(PRD §10)까지 흐려지므로 셋을 나란히 적었다. `docs/internals/testing.md`의 시각 리뷰
      절도 엔진의 새 주인을 가리킨다.

### P3.3 `balpyo init` · `balpyo check`
- [x] `init`: 스캐폴딩(테마 선택 포함). 첫 덱까지 걸리는 시간이 채택을 정한다.
      **선결정 ADR** [0048](docs/decisions/0048-init-owns-its-deck-and-check-is-build-without-writing.md)
      (`788196b`) → `ee1a632`(`core.BuiltinThemes` — `--theme` 어휘의 단일 출처) ·
      `40dc7fa`(`internal/scaffold` + 임베드 템플릿) · `3606cea`(서브커맨드).
      계약을 **파일 목록이 아니라 다음 명령의 성공**으로 못 박았다:
      `TestInitScaffoldBuildsCleanly`이 `init` → `build --target html,pptx --check-layout
      --strict`를 경고 0으로 통과시킨다 — 어휘나 배치가 퇴행하면 첫 사용자가 아니라 이
      테스트가 먼저 깨진다. `examples/`를 복사하지 않은 이유는 그쪽이 마찰점을 일부러 밟는
      회귀 테스트라는 것이고, `--force`가 없는 이유는 덮어쓸 대상이 저자의 원고라는 것이다.
      내장 아닌 `--theme`는 거부한다 — `core.Resolve`가 아직 `balpyo.json`의 `theme`을 읽지
      않으므로(P4.2) 쓰면 조용한 거짓말이 된다.
- [x] `check`: 빌드 없이 진단만(P1.1의 경고를 CI에서 쓰는 입구). 종료 코드 규약을 문서화.
      `21d9a08`(`pipeline.Check`) · `3606cea`(서브커맨드). 판정은 `Build`와 **같은 코드**다
      (`LoadDeck` + `needsGeometry`/`measure` + `geom.Diagnose`) — 파싱까지만 보는 빠른 길로
      가면 "check는 통과했는데 build가 실패한다"가 가능해지고 그 순간 이 명령이 쓸모를 잃는다.
      "쓰지 않는다"도 단정한다: `TestCheckWritesNothing`이 프로젝트 디렉토리 목록이 그대로임을,
      배치 테스트가 `dist/`가 **생기지도** 않았음을 확인한다. 종료 코드는 `build`와 같다 —
      경고는 0을 타고 나가고 `--strict`가 1로 바꾼다.
- [x] DOC: `docs/reference/cli.md`. `f29869a`. 두 명령의 옵션·실제 출력·종료 코드 표를 싣고,
      `check`의 표는 `build`의 표와 **같다는 사실 자체**를 문장으로 적었다(CI가 한쪽을 다른
      쪽으로 바꿔 쓸 수 있는 근거다). `docs/guides/getting-started.md`의 출발점이 예제 복사에서
      `balpyo init my-talk`로 바뀌고 "`init`은 아직 없다" 안내가 사라졌다 — P3.1의 DOC 항목이
      남긴 부채를 여기서 갚는다.
- [x] REVIEW: 스캐폴딩 덱을 `balpyo preview`로 검수. 웹↔PPTX 차이는 #1 cover 0.625% ·
      #2 what 1.772% · #3 end 0.581%로 셋 다 기본 오차(2.5%) 아래이고, 남은 차이는
      LibreOffice의 문장부호 앞뒤 틈과 ~4px 베이스라인뿐이다(알려진 차이). 세 장이 각각
      가르치는 것을 눈으로 확인했다: 표지는 테마 슬롯이 자리를 정하고(자식에 `align`이 없다),
      2장은 `row`/`col`/`shape`로 칸을 나누고, 3장은 `layout`이 정렬을 바꾼다.

### P2.3 `<shape>` 안에 레이아웃 컨테이너 (D7) — **닫힘** (ADR 0049)
- [x] **선결정 ADR** [0049](docs/decisions/0049-a-shape-that-holds-layout-becomes-a-backdrop.md):
      도형은 **배경**, 내부 배치는 자식 좌표다 — 도형 `p:sp` 1개(빈 문단) + 자식은 **절대 좌표
      형제**이고 `p:grpSp`는 쓰지 않는다(그룹의 `chOff`/`chExt`가 둘째 좌표 공간을 만들어 "잰
      rect가 곧 xfrm"을 깬다 — PRD §9.3). 모드는 **도형마다** 정해지고, 판정의 단일 소유자는
      `soml.ShapeHoldsLayout`이다. 직속 자식으로 여는 것은 컨테이너 넷뿐이고(리스트·그림·표·
      중첩 도형은 컨테이너를 **거쳐** 들어온다), 배경 + `rotate`는 에러다(웹은 자식까지 돌리고
      PPTX는 형제라 돌지 않는다).
- [x] RED: `11_shape_layout` 픽스처 3장 — 카드 헤더/본문/푸터(`justify="between"`), 컨테이너를
      거쳐 들어오는 리스트 + 중첩 배지, 두 모드를 나란히 둔 슬라이드. 더해서 pptx 단위 테스트
      (배경은 `<a:p/>` 하나 + 자식이 형제 `p:sp` · `<p:grpSp>`가 없다)와 core의 `rotate` 거부.
- [x] GREEN: `soml`(자식 어휘 + `ShapeHoldsLayout`) → `core`(`rotate` 거부 · 배경에서 패치가
      회전을 버린다) → `render/web`(컨테이너 넷도 `position: relative`로 그림 위에) →
      `render/pptx`(도형을 낸 **뒤에** 자식으로 내려간다 · 배경의 `txBody`는 자식 없는 도형과
      같은 경로) → `internal/testkit`(줄 세기를 두 픽스처가 공유하고, 잉크 띠는 y로 묶는다).
      측정은 손대지 않았다 — 배경 모드의 자식은 이미 자기 rect를 갖는다.
- [x] REVIEW: `balpyo-intro` 06을 도형 2개 흉내에서 **카드 하나**로 되돌렸다 — 카드가
      `rounded-rect` 하나이고 헤더 띠는 컨테이너를 거쳐 들어온 중첩 도형(흡수 모드)이다.
      `build -target html,pptx -check-layout -strict`가 경고 0이고, 슬라이드 6을 웹/PPTX 두
      래스터로 견줘 카드 3장·띠·본문이 같은 자리에 떨어지는 것을 확인했다 (남은 차이는 D11).
- [x] DOC: `docs/reference/soml/shape.md`에 **콘텐츠 모델 — 두 가지 모드** 절(모드 표 · 예제
      둘 · 배경 모드의 PPTX 산출 · 도형 수 표에 배경 행) + `layout.md`(컨테이너가 올 수 있는
      자리) · `image.md`(도형 안 그림은 컨테이너를 거친다). `make docs --strict` 초록.

### P2.4 코드 블록: `lang`과 탭 (D15 · D16) ← **지금 여기**
소프트웨어를 설명하는 덱은 코드를 띄운다. P0.1이 끝나 이제 의미가 있다.
- [ ] **선결정 ADR**: 토크나이저를 어디에 둘지(Go 라이브러리 vs 내장 최소 문법), 색은 **테마
      토큰**으로만(P4.1이 닫혀 강조색 여섯이 있다 — 번호는 위계가 아니므로 토큰 번호와 코드
      토큰 종류의 대응을 이 ADR이 정한다), 지원 언어 1차 집합.
- [ ] RED: 토큰 → 웹 `span` + PPTX **런 분할**(`a:r` 여러 개, 색은 `schemeClr` 참조) 어서션.
- [ ] GREEN: `pre lang="…"` 속성 + 렌더.
- [ ] **D15**: 탭 처리도 같이 정한다 — 저자가 손으로 스페이스로 바꾸지 않아도 되게 `tab-size`
      (웹)와 PPTX 대응(런에 탭을 두는지 펴는지)을 **테마가** 정한다(어휘를 늘리지 않는다).
- [ ] **D16**: 계획된 속성의 진단을 오타와 구별한다 — `<table>`처럼 "아직 구현되지 않은 어휘다"로
      말하는 자리를 **속성**에도 만든다. 예정 이름은 닫힌 목록이고, 목록 밖은 지금처럼
      "알 수 없는 속성"이다. (`lang`을 구현하면 이 목록의 첫 줄이 지워진다.)
- [ ] DOC: `docs/reference/soml/text-subset.md`의 `pre` 절.

### P4.1 accent 팔레트 확장 (D12) — **닫힘** (ADR 0050)
**순서를 바꿔 P2.4보다 먼저 했다.** P2.4의 선결정이 "코드 색은 **테마 토큰**으로만"이고 그
토큰이 강조색이다 — 팔레트가 하나인 동안 하이라이팅을 정하면 키워드·문자열·주석이 같은 파랑의
농담이 되고, 그 결정을 팔레트가 여섯이 된 뒤 다시 해야 한다. 되돌리기 비싼 쪽을 먼저 둔다.
- [x] RED: `12_palette` 픽스처 2장(견본 여섯 + `shade`/`tint`/`alpha`·두 강조색 그라디언트·
      강조색 글자) + `core/palette_test.go`(팔레트 규칙의 소유자 — 대비·색상·휘도 전수) +
      `render/pptx`의 로드된 테마 `clrScheme` 여섯 슬롯. 새 accent에서도 참조가 보존되는지는
      기존 판정이 그대로 받는다(`TestFixtureGoldenSlideColorsKeepTokenReferences` — 슬라이드
      파트의 리터럴 `srgbClr` 전수 표).
- [x] GREEN: `tokens.json`에 `accent2..6`(각자 자기 `slot`/`ref`) + 고른 규칙을 파일에 적는
      `$comment-accent`. 테마 파트 컴파일은 **손대지 않았다** — 표가 여섯을 주면 여섯이 실린다
      (폴백은 강조색 하나짜리 다른 테마를 위해 남는다).
- [x] REVIEW: `examples/incident-postmortem`을 **세 색 위계**로 다시 칠했다 — `accent5`=장애가
      낸 값 · `accent4`=우리가 늦은 곳(탐지) · `accent3`=우리가 한 것과 할 것. 색마다 뜻이
      하나이고 덱의 규칙을 표지 주석이 적는다. 파랑은 이 덱에서 0건이 된다(강조색이 하나였을 때
      "중요함"을 뜻했던 흔적이다). `build -target html,pptx -check-layout -strict` 경고 0이고,
      네 슬라이드를 웹/PPTX 두 래스터로 견줘 위계가 같게 읽히는 것을 확인했다.
- [x] DOC: `docs/reference/theme.md`에 **`tokens.json` — 표준 토큰 어휘** 절(다섯 범주가 채워야
      하는 이름 · 색 토큰 네 필드와 슬롯 규칙 · **강조색 여섯의 규칙** 표) + `appearance.md`의
      색 표에 여섯과 대비 수치, `shape.md`·`text.md`의 진단 토큰 목록. `make docs --strict` 초록.

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
- [x] `examples/quarterly-review/slides/004_table.slide`의 "표는 어휘에 없다" 주석 (P2.1과 함께) —
      주석만 고치지 않고 슬라이드를 `<table>`로 되돌렸다 (우회가 사라지면 주석도 사라진다).
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
- [x] **한 명령으로 검수한다**: `balpyo watch`가 저장마다 탭을 새로고침하고(P3.1),
      `balpyo preview`가 웹·PPTX·diff를 나란히 놓은 갤러리를 한 줄로 만든다(P3.2). PPTX 렌더에는
      도구 둘(`soffice`·`pdftoppm`)이 필요하고 없으면 빌드 전에 그렇게 말한다
- [x] **덱이 덱처럼 생겼다**: 페이지 번호·푸터는 켤 수 있다 (P2.2 — 번호는 뷰어가 다시 세는
      `a:fld`다). 강조색은 재서 고른 여섯이고 번호가 위계를 만들지 않아, 덱이 색 둘~셋에 뜻을
      하나씩 주는 위계를 세울 수 있다 (P4.1 — 회고 덱이 그 예다)
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
| 덱 크롬 어휘와 `a:fld` 매핑 | P2.2 | ADR 0045 (설정이 내용을, 테마가 자리를, `a:fld`가 번호를) |
| 감시가 변경을 보는 법 · 리로드 클라이언트가 사는 곳 | P3.1 | ADR 0046 (폴링 + 스캔 안정화, 주입은 응답에만) |
| 검수 갤러리의 소유자 · `preview`가 게이트가 아닌 이유 | P3.2 | ADR 0047 (엔진은 `internal/preview`, 오차 표는 테스트 정책 — 그래서 코드 0) |
| `init`이 쓰는 덱의 소유자 · `check`가 얼마나 `build`여야 하는가 | P3.3 | ADR 0048 (덱은 `internal/scaffold`가 소유하고 3장, 덮어쓰지 않는다 · `check`는 쓰지 않는 `build`이고 종료 코드가 같다) |
| 도형이 컨테이너를 자식으로 받을 때의 좌표·매핑 | P2.3 | ADR 0049 (도형은 배경 `p:sp` 하나, 자식은 절대 좌표 **형제** — 그룹이 아니다 · 모드는 도형마다) |
| 강조색을 몇 개 · 어떻게 고르는가 · 번호가 위계인가 | P4.1 | ADR 0050 (여섯 · 세 수치가 고른다(테스트가 규칙을 소유) · 번호는 색상환의 이웃 순서이고 위계는 덱이 만든다) |
| 코드 하이라이팅 토크나이저와 색 토큰 · 탭 처리 · "예정된 속성" 진단 | P2.4 | 대기 |
| 차트: 네이티브 `c:chart` vs 도형 합성 | P2.5 | 대기 |
