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
| D13 | 예제·문서가 썩는다. **예제는 CI에서 빌드되지 않는다** | 중간 | P5.1 · P5.2 | **닫힘** (P5.1 예제 게이트 — 두 타깃 · 진단 0건 · 장수 선언 · P5.2 문서 게이트 — 닫힌 목록을 테스트가 문서에서 읽는다) |
| D14 | **`<text>`에 `w`/`h`가 없다.** canvas 라벨은 자기 내용만큼만 넓어서 `align="center"`가 기댈 폭이 없다 — 라벨마다 보이지 않는 도형(`fill="none" border="none" w="140"`)으로 감쌌다. 라벨 10개가 태그 20개가 됐다 | **높음** | P1.3 · P2.5 | P1.3 닫힘 (ADR 0043) · `<chart>`는 P2.5 |
| D15 | `<pre>`에 **`lang`도 탭 폭도 없다.** gofmt 출력(탭)이 중첩 세 번에 x=350까지 밀려서 탭을 스페이스 두 칸으로 손으로 바꿨다 | 중간 | P2.4 | **닫힘** (ADR 0051 — `lang`은 닫힌 5값, 탭 폭은 테마의 `code.tabSize`) |
| D16 | **미구현과 오타의 진단이 구별되지 않는다.** `<table>`은 "아직 구현되지 않은 어휘다"라고 말해 주는데 `<pre lang="go">`는 그냥 "알 수 없는 속성"이다 | 중간 | P2.4 · P5.2 | **닫힘** (P2.4 — `soml.plannedAttrs`, 어휘 표와 양방향 대조 · P5.2 — 파서가 거절한 오타에 값 층이 "엔진 버그"를 겹쳐 말하던 것을 고쳤다) |
| D17 | **레이아웃이 본문 높이를 소유하지 않는다.** `content`는 위에서부터 쌓기만 해서 10장 중 5장의 아래 40%가 비었다. 세로 리듬은 저자가 `<col h="fill" justify="between">` 랩퍼로 슬라이드마다 손수 만든다 (랩퍼 없음·`justify="center"`·`between` 셋 다 재봤다) | **높음** | P1.4 | **닫힘** (ADR 0044) |
| D18 | 같은 뜻의 이름이 갈린다: 컨테이너는 `justify`/`align`, 도형은 `anchor`다. 첫 시도가 `<col>에 알 수 없는 속성 anchor`로 죽었다 | 낮음 | P5.2 | **닫힘** (P5.2 — 이름은 그대로 두고 갈린 이유를 문서가 소유한다: CSS flex는 태그 방향에 따라 세로가 바뀌고 OOXML `bodyPr@anchor`는 축이 하나다) |
| D19 | **저자가 처음 읽는 페이지가 틀렸다.** `slide.md`는 "`layout` 어휘는 2종"이라고 했고 `reference/index.md`는 `layouts.md`가 "Iteration 10에서 생긴다"고 했다 — 11종이 다 살아 있고 그 페이지도 이미 있다 | **높음** | P5.2 | **닫힘** (문구 정정 + 같은 일이 다시 생기지 않게 `TestDocsVocabularyListsMatchCode`가 목록을 대조한다) |
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
| P3.3 `balpyo init` · `balpyo check` — 첫 덱과 CI 입구 | `init`은 자기 덱을 **소유한다**(`internal/scaffold`의 임베드 템플릿 3장) — `examples/`는 마찰점을 일부러 밟는 회귀 테스트라 출발점이 못 된다. 계약은 파일 목록이 아니라 **다음 명령의 성공**이다: 스캐폴딩 덱이 `build --target html,pptx --check-layout --strict`를 경고 0으로 통과한다. `--force`는 없고(덮어쓸 대상이 저자의 원고다) 내장 아닌 `--theme`은 거부한다(`balpyo.json`의 `theme`을 아직 아무도 읽지 않는다 — P4.2에서 읽는다). `check`는 **쓰지 않는 `build`**다: 같은 코드로 판정하고, 종료 코드도 같고(경고는 0을 타고 나간다), 출력 디렉토리를 만들지도 않는다 | [ADR 0048](docs/decisions/0048-init-owns-its-deck-and-check-is-build-without-writing.md), `core.BuiltinThemes`(어휘 단일 출처) · `internal/scaffold` · `pipeline.Check` · `cmd/balpyo/init_test.go`(`init`→`--strict` 빌드) · `check_test.go`(디렉토리 목록이 그대로 · `dist/`가 생기지도 않는다), 스캐폴딩 덱 `preview` 실물 확인(3장 0.625%/1.772%/0.581%), `docs/reference/cli.md#balpyo-init` · `#balpyo-check` · `docs/guides/getting-started.md` |
| P4.1 accent 팔레트 확장 (D12) | 강조색이 **여섯**이고, 눈이 아니라 **세 수치**가 고른 값이다 — `bg` 대비 4.5:1↑(대비율이 대칭이라 글자색과 면색을 한 수치로 보장한다) · 서로 색상 25°↑ · 휘도 한 띠(≤1.25). 실측 4.76~5.38 · 26° · 1.18이고, 그 규칙을 주석이 아니라 **테스트가 소유한다**. `clrScheme`에서 accent1을 빌려 가던 다섯 자리가 자기 색을 갖고(빌리는 자리는 링크 둘뿐) 폴백은 강조색 하나짜리 **다른** 테마를 위해 남는다. 번호는 위계가 아니라 색상환의 이웃 순서이고, **위계는 덱이 만든다** | [ADR 0050](docs/decisions/0050-six-accents-are-a-measured-palette-not-a-ranking.md), `core/palette_test.go`(WCAG 상대 휘도 · 대비 · 색상 간격 전수) · `core/theme_package_test.go`(값 드리프트) · `render/pptx.TestClrSchemeMinimalFillsSixAccents`(로드된 테마의 여섯 슬롯이 서로 다르다) · `12_palette`(2장 — 견본 여섯 + 변형/그라디언트, `TestPaletteFixtureBindsSixAccentSlots`, 슬라이드 파트에 리터럴 `srgbClr` 0건), `examples/incident-postmortem`을 **세 색**으로 다시 칠했다(피해=`accent5` · 우리가 늦은 곳=`accent4` · 대응=`accent3`, 파랑 0건 · 두 래스터 실물 확인), `docs/reference/theme.md`(표준 토큰 어휘 + 팔레트 규칙) · `appearance.md` |
| P2.4 코드 블록: `lang`과 탭 (D15 · D16) | 하이라이팅은 **값 층의 토큰 스트림**이다 — Chroma의 토큰을 우리가 닫은 **7종**으로 축약하고 색이 하나인 조각마다 **합성 인라인 런**을 만든다. 두 렌더러는 한 줄도 고치지 않았고(인라인 런 경로가 이미 있었다) 색은 테마의 `code.tokens` 부분 패치가 갖는다(색 토큰 이름만 — 리터럴은 `schemeClr` 참조를 끊어 코드 블록만 테마 색 편집에서 빠지게 만든다). 런 경계는 종류가 아니라 **해결된 색**이다. 탭은 `lang`과 무관하게 `code.tabSize`로 펴 두 렌더러가 **같은 한 문자열**을 받는다(CSS `tab-size`와 `a:defTabSz`는 같은 규칙이 아니다). `lang` 어휘는 엔진이 닫고(`go`·`xml`/`soml`·`json`·`bash`) 목록 밖은 **에러**다 — 미구현과 오타를 가르는 자리는 `plannedAttrs`가 따로 갖는다 | [ADR 0051](docs/decisions/0051-highlighting-is-a-token-stream-compiled-to-runs.md), `core/highlight_test.go`(어휘 전수 · 탭 칼럼 · 원문 왕복 보존) · `soml/planned_test.go`(예정 목록 ↔ 어휘 표 양방향) · `13_code`(2장 — 골든에 `a:r` 분할, 리터럴 `srgbClr` 0건) · `render/geom`의 `TestMeasureLinesIncludeInlineRuns`(합성 런이 줄 수를 늘리지 않는다), `examples/incident-postmortem` 005가 gofmt 탭을 그대로 붙인다, `minimal`의 `mono` `ea`를 고정폭 CJK로(한글 주석 줄 −7.9% → +0.6%), `docs/reference/soml/text-subset.md`(코드 블록 절) · `appearance.md`(코드 하이라이팅) · `theme.md` · `guides/known-differences.md` |
| P4.2 테마 선택 · 다중 내장 테마 | 테마는 **덱의 요청**이다: `balpyo.json`의 `theme`(과 그것을 덮어쓰는 `--theme`)이 패키지를 고르고, 없는 이름은 렌더 전에 멈춰 쓸 수 있는 이름을 나열한다 — **폴백 테마가 없다**. 폴백이 없으므로 빠진 값을 메울 곳도 없어 로더가 토큰·역할 어휘의 **완전성**을 요구하고, 하드코딩 외관 표(`core/theme.go`)가 사라져 **엔진에 외관 값이 하나도 없다**. 내장은 손으로 쓴 셋(`minimal`·`editorial`·`ink`)이고 하나의 계약을 받는다: 레이아웃 11종 전부 · 슬롯 영역 전부 · `variant` 어휘 동일 · 본문 24px/1.5 · 팔레트 세 수치. 리뷰가 그림을 바꾸지 않는 선언 하나를 잡았다 — `justify-items: start`가 여유 폭을 0으로 만들어 PPTX에서 줄을 접었다(왼쪽은 선언하지 않는다) | [ADR 0052](docs/decisions/0052-theme-selection-is-a-deck-request-and-there-is-no-fallback-theme.md)·[0053](docs/decisions/0053-a-theme-takes-the-horizontal-channel-only-to-center.md), `core/theme_select_test.go`(선택·에러·두 테마가 다르다 + **갈리지 않는 것** 전수: variant 어휘·본문 리듬·레이아웃 11종·슬롯 영역) · `core/palette_test.go`(세 테마 실측 4.76~5.38 · 4.92~5.00 · 5.93~6.00, 띠 ≤1.18) · `cmd/balpyo/theme_test.go`(`--theme` 네 자리) · `14_theme`(`ink`로 렌더되는 3장 골든), `make review` 세 테마 갤러리 실물 확인, `docs/guides/theming.md`(새 가이드) · `docs/reference/theme.md` · `guides/known-differences.md` |
| P4.3 외부 테마 패키지 · `extends` | 자기 테마는 **프로젝트 안의 이름**이다 — `themes/<이름>/`이 `--theme`·`balpyo.json`의 어휘에 들어가고 **경로 문법은 없다**(값이 언제나 이름이라 진단이 어휘 하나만 나열하고, 프로젝트 밖 파일을 읽는 길이 없다). 이름을 규칙 셋이 지킨다: 디렉토리 = 매니페스트 `name` · 내장 이름 가림은 **요청한 테마와 무관하게** 에러 · 없는 부모와 순환은 체인을 적고 멈춘다. 병합은 한 문장이다 — **이름이 같으면 자식이 이기고, 없던 이름은 부모 순서 뒤에 붙는다**(순서가 `clrScheme`·CSS 변수·경고 목록에 보인다). 이름 없는 절만 필드 단위, 레이아웃 CSS는 파일 단위, 지우는 문법은 없다. 로더를 읽는 반·변환하는 반으로 쪼개 병합을 **원시 선언 층**에서 하고 완전성 검사는 결과에 한 번만 돈다. 리뷰가 상한을 드러냈다 — 색 토큰은 `clrScheme` 열두 자리를 넘을 수 없어 **자식은 색을 갈아 끼운다** | [ADR 0054](docs/decisions/0054-an-external-theme-is-a-name-inside-the-project-and-extends-overrides-by-name.md)·[0055](docs/decisions/0055-a-child-theme-recolors-by-name-because-color-tokens-are-bounded-by-the-clrscheme.md), `core/theme_local_test.go`(어휘·이름 규칙·체인 병합·슬롯 상한) · `core/theme_require.go`의 `requireColorSlots`(판정이 **테마 로딩**에 있고 남은 자리를 나열한다) · `soml.ColorSlots()`(12자리 단일 출처 — 값 층과 PPTX 렌더러가 같은 목록을 읽는다) · `15_local_theme`(`themes/corp/` 파일 넷으로 `minimal`을 물려받는 3장 골든 — 나머지 열 레이아웃·역할 넷이 부모와 **같다**는 것이 상속의 증인) · `internal/testkit`의 `chromeEnabledFixtures`(크롬 부재 게이트를 이름으로 옵트인), `make review` 실물 확인, `docs/reference/theme.md`(외부 테마 절 + 병합 표) · `guides/theming.md`(자기 테마 만들기) · `cli.md`·`layouts.md` |
| P5.1 예제를 게이트가 빌드한다 (D13) | 도그푸딩 덱 셋이 **테스트**다: 두 타깃으로 빌드되고 **진단 0건**이고 장수가 선언한 숫자(12·10·6)와 같아야 한다. 경고가 실패인 이유는 예제가 사용자 덱이 아니라 제품이 쓴 문서라는 것이다 — 넘침 경고를 내는 예제는 "경고는 무시해도 된다"를 가르친다. 목록은 디렉토리를 걸어 얻어 새 예제가 자동으로 들어오고, 장수 표에 줄을 더할 때까지 실패한다(조용히 빠진 장을 잡는 유일한 판정이다). **골든은 걸지 않는다**: 28장의 diff는 사람이 읽지 않은 채 승인되고, 승인되지 않는 골든은 게이트처럼 보이는 것일 뿐이다. 오늘 셋 다 경고 0으로 통과하므로 이 항목이 고친 것은 덱이 아니라 그 사실의 **상시화**다. CI 잡은 더하지 않았다 — 게이트는 잡이 아니라 테스트로 늘린다 | [ADR 0056](docs/decisions/0056-examples-are-gated-by-a-strict-build-not-by-goldens.md), `internal/testkit/example.go` · `TestExamplesBuildClean`(`make test`에 포함 · `make examples`가 같은 테스트만 돈다), 게이트가 무는지 확인(`005_callout.slide`를 `_`로 가리자 `슬라이드가 5장이다 (선언: 6장)`), `docs/internals/testing.md`(예제 게이트 절 + CI 절) |
| D19 문서가 어휘를 숨긴다 | `slide.md`(11종 표 + 실제 진단 문구) · `reference/index.md`(`layouts.md`는 구현됨) · `text.md`(속성 목록 + `<text>`에 `w`/`h`가 없다는 것) | `make docs --strict`, 커밋 `cc8239f` |
| P5.2 문서·예제 주석의 사실 정정 (D13 · D16 · D18 · D19) | 문서는 **손으로 쓰는 것으로 남기고** 닫힌 열거만 테스트가 읽는다 — 실제로 썩은 둘이 산문이 아니라 **목록**이었고(D19의 "`layout` 어휘는 2종", 개요의 `minimal`·`dark`·`paper`) 진단 문구의 회귀는 각 패키지 테스트가 이미 잡는다. 등록된 17자리에서 문서의 목록과 코드의 목록을 **집합으로** 맞대고, 순서·표기는 문서의 것이며, **패턴이 한 번도 맞지 않으면 실패**다(이 게이트가 죽는 가장 조용한 길이 문서를 고쳐 써서 패턴이 어긋나는 것이다). 개요는 이제 동작하는 어휘를 **세지 않고 가리킨다**(같은 목록 두 벌 중 한 벌이 낡았다). D18은 어휘를 바꾸지 않고 이름이 갈린 이유를 `layout.md`가 소유하게 했다. 문장을 실물로 확인하다 제품 결함 하나를 고쳤다 — 파서가 이미 거절한 오타에 값 층이 "엔진 버그"를 겹쳐 말했다 | [ADR 0057](docs/decisions/0057-docs-are-hand-written-but-closed-lists-are-read-from-them-by-a-test.md), `core/docs_test.go`(`docVocabularies` 17줄 · 패턴 불일치는 `Fatal` · 도구 없이 밀리초) · `core/catalog_gap_test.go`(**두 방향** — 거절된 이름에 침묵, 카탈로그가 연 이름에는 여전히 소리) · `soml.ElementAttrs`, 두 실패 모드를 실물로 확인(이름 하나 빼기 · 문장 고쳐 쓰기), `docs/index.md`(상태·테마 이름·CLI 표면·예제 덱 절) · `docs/internals/testing.md`(문서 게이트 절 — **판정하지 않는 것**까지) · `docs/reference/soml/layout.md`·`shape.md`(D18) · `examples/quarterly-review/003_kpis.slide`(거짓 주석) |
| P5.3 접근성: 이름과 읽기 순서 | `alt`이 도형·표까지 열렸고 **이름을 더하되 안의 글을 대체하지 않는다**: 도형은 잉크 `<svg role="img" aria-label>`(박스 `<div>`는 `generic`이라 ARIA가 이름을 금지한다 — 거기 적으면 저자의 접근성 텍스트가 **조용히 사라진다**), 표는 `<table aria-label>`(`role="img"`은 셀 항해를 지우고 `<caption>`은 보이는 줄을 웹에만 더한다), 양쪽 모두 PPTX는 `p:cNvPr@descr`이다. 이름 없는 도형의 산출은 **한 바이트도 바뀌지 않았다**(잉크는 여전히 `aria-hidden`). `alt=""`·값 없는 `alt`은 도형·표에서 **에러**이고 그림에서만 살아 있다 — 빈 alt의 뜻은 HTML `<img>`가 정의한 것이라 근거가 우리 밖에 있다. 읽기 순서는 **문서 순서**이고 그것을 바꾸는 문법은 없다(`order`·좌표는 **시각** 순서 채널이고, 갈라짐에 진단을 내면 `order`를 의도대로 쓴 덱이 `--strict`에서 멈춘다) | [ADR 0058](docs/decisions/0058-alt-names-the-drawing-and-reading-order-is-document-order.md), `core/alt.go`(빈 값 두 진단) · `render/web/alt.go`(`inkAccessAttrs`는 `aria-hidden`과 이름이 **배타적**임을 한 자리에 담는다) · `render/pptx/alt.go`(`cNvPrXML` — 도형·그림·표 프레임이 같은 한 함수를 지나 `descr` 규칙이 갈라질 수 없다), `render/pptx.TestReadingOrderIsDocumentOrderInBothTargets`(spTree 순서 **와** 웹 DOM 순서 + `order`가 여전히 CSS로 나가는지) · 픽스처 `16_a11y`(2장 — 세 태그의 이름 · `order`로 뒤집은 시각 순서), `docs/reference/appearance.md`의 접근성 절 · `docs/reference/soml/shape.md`·`table.md`(`alt` 행 + 두 진단) |

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
      않으므로(P4.2) 쓰면 조용한 거짓말이 된다. (P4.2에서 읽는다: 이제 `init --theme`이 적은
      값이 그대로 선택이 되고, `build`·`check`·`watch`·`preview`의 `--theme`이 그 값을 덮어쓴다.)
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

### P2.4 코드 블록: `lang`과 탭 (D15 · D16) — **닫힘** (ADR 0051)
소프트웨어를 설명하는 덱은 코드를 띄운다. P0.1이 끝나 이제 의미가 있다.
- [x] **선결정 ADR** ([0051](docs/decisions/0051-highlighting-is-a-token-stream-compiled-to-runs.md)):
      토크나이저는 **Chroma의 토큰 스트림만**(스타일·포매터는 쓰지 않는다), 우리 경계는 닫힌
      **토큰 종류 7종**(`keyword`·`name`·`attr`·`string`·`number`·`comment`·`punct`, 나머지는
      `plain` = 색 없음). 토큰은 **값 층이 합성 인라인 런**으로 컴파일해 두 렌더러를 손대지
      않는다(인접한 같은 종류는 합친다). 색은 `components.json`의 새 `code.tokens` 표가 갖고
      (부분 패치 · 색 토큰 이름만 · 리터럴 hex는 에러), `minimal`은 keyword→`accent6` ·
      name→`accent1` · attr→`accent2` · string→`accent3` · number→`accent4` · comment→`muted`.
      언어 1차 집합은 `go`·`xml`(별칭 `soml`)·`json`·`bash`이고 목록 밖은 **에러**다
      (엔진 어휘이므로 variant처럼 폴백하지 않는다). `<pre><code>`는 열지 않는다.
- [x] RED: `core/highlight_test.go`(토큰 → 색 있는 런 경계 · `lang` 어휘 전수 · 탭 칼럼 ·
      왕복 보존) + `core/chrome_test.go`의 합성 테마 `code` 절 + `13_code` 픽스처 2장
      (`go` 한 장 · `soml`/`json`/`bash` 한 장, 둘 다 원문이 탭 들여쓰기다). 런 분할과
      `schemeClr` 참조는 **기존 판정이 그대로 받는다** —
      `TestFixtureGoldenSlideColorsKeepTokenReferences`(슬라이드 파트의 리터럴 `srgbClr`
      전수 표)에 이 픽스처의 행이 없는 것이 계약이고, 골든 파트에 `a:r`가 여럿 보인다.
- [x] GREEN: `soml`이 `<pre>`에 `lang`을 열고(고유 속성 — 다른 블록에는 뜻이 없다) `core`가
      값 층에서 **탭을 펴고 토큰 스트림을 합성 런으로 컴파일한다**. 두 렌더러는 한 줄도
      고치지 않았다 — 인라인 런 경로가 이미 있었다. 런 경계는 종류가 아니라 **해결된 색**이다
      (테마가 적지 않은 종류는 plain과 같은 색이므로 끊을 이유가 없다).
- [x] **D15**: 탭은 `lang`과 무관하게 **값 층이 칼럼을 세어 편다** — 폭은 테마의
      `code.tabSize`(`minimal`은 2)이고 어휘는 늘지 않았다. `tab-size`(CSS)와 `a:defTabSz`
      (OOXML)가 같은 규칙이 아니므로 탭을 흘리면 두 산출물의 줄이 갈라진다: 두 렌더러가
      **같은 한 문자열**을 받는 쪽으로 풀었다. 탭 없는 덱의 산출물은 한 바이트도 안 움직인다.
- [x] **D16**: `soml`의 `plannedAttrs`가 그 자리다 — "이름은 맞는데 아직 구현되지 않은 속성"이
      `PlannedAttrMessage`로 말하고, 목록 밖은 지금처럼 "알 수 없는 속성"이다(오타와 예정을
      한 문장으로 뭉치지 않는다). 목록이 썩지 않게 어휘 표와 **양방향 대조**한다
      (`planned_test.go` — `unrenderedKinds` ↔ 렌더러 디스패치와 같은 장치). `lang`이 그
      규칙의 첫 증인이다: 이 항목이 줄을 적고 같은 항목이 지웠다.
- [x] REVIEW: `examples/incident-postmortem` 005를 **두 마찰점이 사라진 모습으로** 되돌렸다 —
      `lang="go"` + gofmt 출력의 탭을 그대로 붙이고, 머리 주석은 이제 마찰의 **기록**이다.
      두 래스터를 견줘 토큰 색·2칸 계단·줄 수가 같고, 남은 차이는 이미 문서화된 LibreOffice의
      스크립트 간격(한글 주석·캡션)뿐이다. 이 항목에서 실물로 잡은 것 둘: 잰 줄 수가 인라인
      런을 블록의 줄로 세지 않아 `13_code`가 "띠에 주인이 없다"로 죽었고(`render/geom`의
      `lineRects`가 식별자 없는 자손까지 읽고 같은 줄 조각을 합치게 고쳤다), `minimal`의
      `mono`가 고정폭 라틴에 **비례폭 CJK**를 짝지어 한글 주석 줄이 −7.9%였다
      (`ea: Noto Sans Mono CJK KR`로 +0.6%).
- [x] DOC: `text-subset.md`에 **코드 블록 — `lang`과 탭** 절(닫힌 값 표 · 강조는 토크나이저가
      만든다 · 탭은 `lang`과 무관하다) + 진단 표 네 줄, `appearance.md`에 **코드 하이라이팅
      (합성 런)** 절(7종 ↔ `minimal`의 색 · 색 토큰 이름만), `theme.md`에 `code` 절의 에러
      다섯 줄과 **`mono`의 `ea`는 고정폭 CJK여야 한다**는 규칙, `known-differences.md`에 그
      규칙의 측정. `make docs --strict` 초록.

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

### P4.2 테마 선택 · 다중 내장 테마 — **닫힘** (ADR 0052 · 0053)
- [x] RED: `balpyo.json`의 `theme` 필드가 로더 선택을 바꾸는 테스트, 없는 테마 에러, 같은 덱이
      두 테마에서 다르게 렌더되는 픽스처.
      **선결정 ADR** [0052](docs/decisions/0052-theme-selection-is-a-deck-request-and-there-is-no-fallback-theme.md)
      (`ef4634e`) → `b0efc69`(`core/theme_select_test.go` · 팔레트·컴포넌트 테스트를 내장 전수로)
      · `50f1573`(픽스처 `14_theme` — `ink`로 렌더되는 3장 덱, 웹·PPTX 골든까지).
      요청과 결과를 이름으로 갈랐다(`ThemeName` = 덱이 쓴 요청 · `Theme` = 로드된 결과):
      `TestResolveUsesDeckThemeName` · `TestResolveEmptyThemeNameUsesDefault` ·
      `TestResolveUnknownThemeIsAnError`(같은 판정을 `Config.Validate`도
      한다 — `TestConfigValidateChecksThemeVocabulary`, `TestLoadDeckRejectsUnknownTheme`).
      "같은 덱이 다르게 렌더된다"는 것을 픽스처 없이도 못 박는 테스트가
      `TestBuiltinThemesRenderTheSameDeckDifferently`이고, **갈리지 않아야 하는 것**도 같은
      파일이 잡는다: `TestBuiltinThemesShareRecommendedVariants`(이름과 순서까지) ·
      `TestBuiltinThemesShareBodyRhythm`(본문 24px · 1.5) ·
      `TestBuiltinThemesImplementEveryStandardLayout` · `TestBuiltinThemesCoverEverySlotArea` ·
      `TestBuiltinAccentPalettes`/`TestBuiltinBaseColorContrast`(P4.1의 세 수치를 세 테마에).
- [x] GREEN: `core`의 하드코딩 `minimal` 선택·레거시 폴백 제거 + 내장 테마 2~3종.
      `16ef5f7`(요청으로 고르고 폴백하지 않는다) · `b7f950f`(로더가 토큰·역할 어휘의 **완전성**을
      요구한다 — 폴백이 없으면 빠진 값을 메울 곳도 없다) · `ffa6d1b`(`core/theme.go` 삭제:
      엔진이 들고 있던 하드코딩 외관 표가 사라져 **엔진에 외관 값이 하나도 없다**) ·
      `1aa7168`(`editorial` — 종이 바탕 · 세리프 제목 · 표지가 아래로) · `b1bf77d`(`ink` —
      먹빛 바탕 · 밝은 강조색 · 왼쪽 기준선) · `56cad35`(`--theme`이 `build`·`check`·`watch`·
      `preview`에서 `balpyo.json`을 덮어쓴다 — 파일을 고치지 않고 두 테마를 견주는 자리다).
      내장 테마가 **셋 다 손으로 쓴 값**인 것은 의도다: `extends`가 없으므로 레이아웃 CSS가
      세 벌 존재하고, 무엇이 정말 갈라지는지 본 뒤에 병합 규칙을 정한다(P4.3).
- [x] REVIEW: `make review`로 세 테마 갤러리를 만들어 눈으로 확인했고, 거기서 **하나를 잡았다**:
      `ink` 표지의 부제가 웹에서 한 줄인데 PPTX에서 두 줄로 접혔다. 원인은 그림을 바꾸지 않는
      선언이었다 — 슬롯 영역의 `justify-items: start`가 텍스트 박스를 내용 폭으로 줄여
      여유 폭(`Slack`)을 0으로 만들고, LibreOffice의 문장부호 간격(경계마다 +0.245em)이 4%
      슬랙을 넘는다. `4ee50d7`에서 내장 테마의 모든 `justify-items: start`를 지웠다(왼쪽은
      기본값이라 선언할 필요가 없다). 가운데 정렬은 그 축소가 **수단**이라 남겨 두고 비용을
      문서로 넘겼다 — [ADR 0053](docs/decisions/0053-a-theme-takes-the-horizontal-channel-only-to-center.md)
      (`bd7115e`)이 그 결정과 열어 둔 것("한 줄짜리 박스를 슬롯 영역 폭으로 넓힌다")을 적는다.
      같은 커밋이 빠져 있던 ADR 0052 인덱스 줄도 채웠다.
- [x] DOC: `docs/reference/theme.md`, `docs/guides/theming.md`.
      `6771a0d`(theme.md에 **테마 선택** 절 — 우선순위 표 · 없는 이름의 실제 문구 · 폴백이 없는
      이유 · 로더 완전성 · 세 테마의 팔레트 실측 · **슬롯 영역의 가로 채널** 절) ·
      `86d20ca`(known-differences에 "가운데 정렬 슬롯은 여유가 없다" — 잰 값과 저자 조언) ·
      `037830f`(cli.md의 `--theme` 네 자리) · `1e650a8`(appearance.md의 값은 minimal의 것이고
      테마가 무엇을 갈아치우는지 표) · `a678075`(**새 가이드** `guides/theming.md` + nav) ·
      `0f26643`(layouts.md) · `0069072`(getting-started의 다음 목록). `make docs` 초록.
      문서를 코드와 대조하다 **거짓 두 줄을 잡았다**: theme.md의 "빠진 표준 레이아웃은 경고 +
      minimal 폴백"과 layouts.md의 "선언하지 않은 레이아웃은 `content`로 폴백하고 경고한다" —
      둘 다 그런 코드가 없다(`lintLayouts`는 **테스트만** 부르고, 레이아웃 맵 조회는 조용히
      빈 값을 준다). 실제 동작(아무 일도 일어나지 않는다)과 테스트가 지키는 계약으로 다시 썼고,
      `3eb45c7`에서 `core/themes/minimal/*.json`의 같은 거짓 주석과 죽은 `core/theme.go`
      포인터도 지웠다.

### P4.3 외부 테마 패키지 · `extends` · `chrome` — **닫힘** (ADR 0054 · 0055)
- [x] RED: 프로젝트 로컬 테마 디렉토리 로딩, `extends` 병합, `chrome` 선언까지 도는 픽스처.
      **선결정 ADR** [0054](docs/decisions/0054-an-external-theme-is-a-name-inside-the-project-and-extends-overrides-by-name.md)
      (`c9d24b6`) → `b72101b`(`core.Resolve`가 프로젝트 파일시스템을 받는다 — 테마를 거기서
      읽으므로 `..`이나 절대 경로로 프로젝트 밖을 읽는 길이 없다) → `088be37`
      (`core/theme_local_test.go`). 요구한 것이 어휘 하나다: `themes/<이름>/`이 `--theme`과
      `balpyo.json`의 어휘에 **들어가고**, 디렉토리 이름 ≠ 매니페스트 `name`·내장 이름 가림·
      없는 부모·순환(자기 자신 포함)이 전부 로드 시점 에러이고, 병합이 **이름 단위**로 돈다
      (`TestLocalThemeOverridesByName` · `TestExtendsChainMergesRootFirst` ·
      `TestLocalThemeShadowingABuiltinIsAnErrorForAnyRequest`).
- [x] GREEN: 이름 어휘 확장 + `extends` 병합. `b18d559`(로더를 **읽는 반**과 **변환하는 반**으로
      쪼갰다 — 병합은 원시 선언 층에서 일어나야 하고, 변환·완전성 검사는 병합 **결과에** 한 번만
      돈다) · `274378d`(로컬 테마 해석 + `extends` 체인 병합). 병합 규칙이 한 문장이다:
      **이름이 같으면 자식이 이기고, 없던 이름은 부모 순서 뒤에 붙는다** — 순서가 산출물에
      보이기 때문에(`clrScheme` 슬롯 순서 · CSS 변수 선언 순서 · variant 경고의 목록) 부모 순서를
      지키는 것이 규칙의 일부다. 이름 없는 절(`shape.default`·표의 `cell`)만 필드 단위,
      레이아웃 CSS는 **파일 단위**, 지우는 문법은 없다.
- [x] REVIEW: 픽스처 `15_local_theme`(`142aaa0` — `themes/corp/`가 `minimal`을 물려받는 3장 덱,
      웹·PPTX 골든까지)을 `make review`로 눈으로 봤고, 픽스처를 만들다 **하나를 잡았다**:
      자식이 색 이름 하나를 더하자 빌드가 `pptx 렌더 실패: 색 토큰 "brand"에 clrScheme 슬롯이
      없다`로 죽었다 — 고칠 자리는 `tokens.json`인데 진단이 PPTX 조립을 가리켰다. 원인이 상한
      이라는 것이 진짜 발견이다: 모든 색 토큰은 `clrScheme` 슬롯을 가져야 PowerPoint의 테마 색
      편집이 따라오고 그 자리는 **열둘로 닫혀** 있어, 내장 테마가 열을 쓰는 지금 자식이 더할 수
      있는 색 이름은 둘뿐이다. 그래서 **자식 테마의 색 작업은 갈아 끼우기**라고 정하고
      ([ADR 0055](docs/decisions/0055-a-child-theme-recolors-by-name-because-color-tokens-are-bounded-by-the-clrscheme.md),
      `63891f1`) 판정을 **테마 로딩으로** 옮겼다(`0c7ebe6` — `requireColorSlots`가 `requireTokens`
      옆에 서고, 진단이 남은 자리를 나열하거나 자리가 없으면 "기존 이름의 값을 갈아 끼운다"고
      말한다). 같은 커밋이 12자리 목록을 `soml.ColorSlots()` 하나로 모았다 — 값 층과 PPTX
      렌더러가 **같은 목록**을 읽어야 둘의 판정이 갈리지 않는다(렌더러의 검사는 층으로 남겼다).
      `chromeEnabledFixtures`를 둔 것도 리뷰의 산물이다: 크롬 부재 게이트를 픽스처 이름으로
      옵트인하게 해서 게이트가 헐거워지지 않게 했다.
- [x] DOC: `c746b73` — `docs/reference/theme.md`의 **외부 테마** 절(트리 · 경로 문법이 없는 이유 ·
      이름 규칙 셋 · `extends` 에러 표 · 병합 규칙 한 문장과 단위 표 · 색의 상한 · 예제) ·
      `guides/theming.md`의 **자기 테마 만들기**(물려받는 `theme.json`/`tokens.json`과 외울 세 줄) ·
      `cli.md`(`build`의 `--theme`는 내장 ∪ 로컬 이름이고 경로가 아니다 · `init --theme`만 내장
      제한인 이유) · `layouts.md`(로드 시점에 멈춘다 · **테마 자기만의 레이아웃 이름은 없다**) ·
      `reference/index.md`의 "외부 패키지는 아직 없다" 정정 · 두 픽스처 문서. `make docs` 초록.
      쓰면서 mkdocs의 제약을 하나 배웠다 — 기본 slugify가 비ASCII를 버려 한글 제목의 앵커가
      `_1`·`_2`가 되고 `pymdownx`가 없다. 한글 앵커 링크를 굵은 본문 참조로 바꿨다(미리 있던 셋도).

**`chrome`의 마스터 고정 요소는 열지 않았다.** 테마의 `chrome` 선언이 병합 어휘에 들어간 것까지가
이 항목이고(같은 `kind`면 자식이 통째로 이긴다), 로고·브랜드 스트립처럼 **모든 장에 도형을 얹는**
어휘는 없다. P2.2에서 정한 크롬의 경계가 "설정이 내용을, 테마가 자리를"인데(ADR 0045) 도형은 내용도
자리도 테마의 것이라 어휘가 한 겹 더 필요하고, 도그푸딩에서 그것을 요구한 기록이 D1–D20에 없다 —
근거 없는 어휘를 먼저 열지 않는다. 열 때 붙일 자리는 `theme.json`의 `chrome`이다.

### P5.1 예제를 CI가 빌드한다 — **도그푸딩을 상시화한다** (D13) — **닫힘** (ADR 0056)
- [x] RED: 예제 게이트를 테스트로 세웠다 — `87b0d23`(`internal/testkit/example.go` +
      `TestExamplesBuildClean`). **선결정 ADR**
      [0056](docs/decisions/0056-examples-are-gated-by-a-strict-build-not-by-goldens.md)(`5047d58`).
      예제마다 판정하는 것이 셋이다: 두 타깃(`html,pptx` — PPTX가 지오메트리 패스를 돌리므로
      좌표까지 난다) · **진단 0건**(`Strict` — 넘침·이탈·겹침과 테마에 없는 variant가 실패다) ·
      슬라이드 수가 표(`12 · 10 · 6`)와 같다. 목록은 디렉토리를 걸어 얻으므로 새 예제가 자동으로
      게이트에 들어오고, 장수 표에 줄을 더할 때까지 실패한다. 게이트가 무는지 실제로 확인했다 —
      `005_callout.slide`를 `_005_callout.slide`로 바꾸자 `슬라이드가 5장이다 (선언: 6장)`로
      빨간불이 났다(수집에서 조용히 빠지는 경로다).
- [x] GREEN: 게이트가 처음부터 초록이다 — 셋 모두 `--strict`를 **경고 0으로** 통과한다(12 · 10 ·
      6장, 컨테이너에서 실측 0.67s/0.53s/0.48s). 그래서 이 항목이 고친 것은 덱이 아니라 **사실의
      상시화**다: 다음 어휘 변경에서도 그 문장이 참인지 `make test`가 묻는다.
      `729ee40`이 `make examples`를 더했다 — 같은 테스트만 도는 입구다(예제를 고치는 사람이
      130초짜리 픽스처 스위트를 기다리지 않게).
- [x] GREEN: 예제를 픽스처로 승격하지 **않기로** 정했다 (ADR 0056 결정 3). 골든의 값은 사람이
      diff를 읽는 데서 나오고 픽스처가 그 조건을 만족하는 이유는 크기와 의도다(한 픽스처가 한
      어휘, 2~4장). 예제 28장에 골든을 걸면 `accent1` 색 한 번 고치는 일이 28장의 XML diff가 되고
      그 안의 진짜 회귀 한 줄은 읽히지 않은 채 `make golden`으로 승인된다 — 승인되지 않는 골든은
      게이트가 아니라 **게이트처럼 보이는 것**이다. 픽셀 비교와 LibreOffice 열림 검사도 같은
      이유로 픽스처의 몫으로 남겼다.
- [x] DOC: `7fbc1e9` — `docs/internals/testing.md`에 **예제 게이트** 절(판정 셋의 표 · 경고가
      실패인 이유 · 장수를 손으로 적는 이유 · 골든이 없는 이유 · 손으로 보는 명령)과 CI 절의 한
      단락(게이트는 **잡이 아니라 테스트로** 늘린다 — 잡을 쪼개면 가장 무거운 이미지 빌드가 한 번
      더 붙는다). `make docs` 초록.

**CI 잡은 더하지 않았다.** D13의 문구는 "예제가 CI에서 빌드되지 않는다"이고 그 요구는 게이트에
들어간 것으로 충족된다 — `make test`가 이 테스트를 돌고, 이 저장소의 CI는 **로컬과 같은 make
타깃**을 도는 것이 원칙이다(`ci.yml`의 첫 주석). 잡을 따로 두면 예제 셋을 빌드하는 2초를 위해
컨테이너 이미지 빌드(가장 무거운 단계)를 한 번 더 낸다. 예제가 열 개가 되어 `make test`가 눈에
띄게 느려지면 그때 분리하고, **분리의 근거는 실측**이다.

### P5.2 문서·예제 주석의 사실 정정 (D13 · D16 · D18 · D19) — **닫힘** (ADR 0057)
- [x] **선결정**: 문서가 코드와 갈리지 않게 **테스트가 문서를 읽게** 할지 판단했다 →
      [0057](docs/decisions/0057-docs-are-hand-written-but-closed-lists-are-read-from-them-by-a-test.md)(`10b7d32`).
      근거는 실측이다: 실제로 썩은 둘(D19의 "`layout` 어휘는 2종", 개요의
      "`minimal`·`dark`·`paper`")이 **산문이 아니라 목록**이었고, 진단 문구의 회귀는 각 패키지의
      테스트가 이미 기댓값으로 잡는다. 그래서 문서는 손으로 쓰는 것으로 남기고 **닫힌 열거만**
      집합으로 대조한다 — 문서 생성도, 문구 전체 비교도 열지 않았다.
- [x] `examples/quarterly-review/slides/004_table.slide`의 "표는 어휘에 없다" 주석 (P2.1과 함께) —
      주석만 고치지 않고 슬라이드를 `<table>`로 되돌렸다 (우회가 사라지면 주석도 사라진다).
      같은 성질의 거짓 주석을 하나 더 잡았다: `003_kpis.slide`의 "도형은 `<col>`을 못 받아"는
      ADR 0049 이후 거짓이다 (`3da9dad`).
- [x] `docs/index.md`의 "Iteration 5까지" 상태 문구 → **세지 않고 가리킨다** (`3da9dad`).
      같은 목록을 두 곳에 두었더니 낡았으므로 "지금 동작하는 것"의 단일 출처는 레퍼런스의
      "현재 상태"다. 확인하면서 세 가지가 더 틀려 있었다 — 내장 테마 이름(`minimal`·`dark`·
      `paper` → `minimal`·`editorial`·`ink`) · CLI 표면(있는 여섯 개와 `--mode`가 없다는 것) ·
      `build`의 타깃이 둘이라는 것.
- [x] `docs/index.md`에 **예제 덱** 절 (`3da9dad`) — 셋의 장수·주제 표 + clone부터
      `ls dist/`까지의 명령. 산출 PPTX 이름이 덱 제목에서 나온다는 것까지 실물로 확인해 적었다
      (`deck.pptx`라고 쓸 뻔했다).
- [x] **D18**: `layout.md`가 이름이 둘인 이유를 소유하고 `shape.md`가 가리킨다 (`c470b0d`) —
      컨테이너 속성은 CSS flex/grid 이름의 얇은 층이라 **어느 속성이 세로인지가 태그 방향에 따라
      바뀌고**(`<col>`은 `justify`가 세로), `anchor`는 축이 하나뿐인 OOXML `bodyPr@anchor`의
      이름이라 태그와 무관하게 언제나 세로다. 어휘는 바꾸지 않았다.
- [x] GREEN: `TestDocsVocabularyListsMatchCode`가 등록된 자리 17곳에서 문서의 목록을 코드의
      목록과 **집합으로** 맞댄다 (`607258b`). 순서·표기는 문서의 것이고(진단 형식과 산문 형식이
      같은 목록을 적을 수 있다), **패턴이 한 번도 맞지 않으면 `Fatal`**이다 — 이 게이트가 죽는
      가장 조용한 길이 문서를 고쳐 써서 패턴이 어긋나는 것이기 때문이다(ADR 0056의 "승인되지
      않는 골든은 게이트가 아니다"와 같은 기준). 두 실패 모드를 이름 하나를 빼고 문장 하나를
      고쳐 써서 실물로 확인했다. 테스트가 `core`에 있는 이유는 필수 토큰 목록이 그 패키지의
      비공개 변수이고 `internal/testkit`에 두면 순환이라는 것이다 — 문서만 고치는 커밋이
      브라우저도 130초짜리 스위트도 기다리지 않는다.
- [x] DOC: `docs/internals/testing.md`에 **문서 게이트** 절 (`9eaa138`) — 명령 · 판정 두 줄의 표 ·
      순서와 표기가 문서의 것인 이유 · 패턴 불일치가 실패인 이유 · **판정하지 않는 것**(진단 문구
      전체, 문서에 실린 SOML 예제가 빌드되는지) · `core`에 있는 이유.
- [x] 덤으로 **제품 결함 하나**를 고쳤다 (`f71c2c3`): D18의 문장을 실물로 확인하다 `<shape>`·
      `<image>`·`<td>`가 파서가 이미 거절한 속성에 `"값 해석이 없다 … (엔진 버그)"`를 한 번 더
      내는 것을 봤다 — 저자가 오타 하나에 진단 둘을 받고 둘째가 자기 오타를 제품의 결함이라고
      말한다(D16이 가르려던 그 경계다). `catalogGap`이 이제 태그의 카탈로그(`soml.ElementAttrs`)로
      두 경우를 가르고, 테스트가 **두 방향**을 함께 못 박는다(거절된 이름에 침묵 · 카탈로그가 연
      이름에는 여전히 소리 — 후자가 없으면 가드를 지운 변경이 초록으로 통과한다).

**문서에 남은 위험은 예제다.** 이 게이트는 목록만 본다 — 문서에 실린 SOML 블록이 실제로
빌드되는지는 아직 아무것도 보지 않고, 픽스처에서 베껴 오는 관례가 유일한 장치다. 열려면
"문서의 코드 블록을 픽스처 파일에서 포함(include)한다"가 다음 후보이고, 근거는 그 블록이
실제로 썩은 사례가 나올 때 생긴다 (지금은 없다).

### P5.3 접근성: 도형·텍스트의 대체 텍스트와 읽기 순서 — **닫힘** (ADR 0058)
- [x] **선결정**: `alt`이 도형의 **안의 글을 대체하는가**, 이름이 어느 요소에 붙는가, 읽기 순서를
      바꾸는 문법을 열 것인가를 먼저 판단했다 →
      [0058](docs/decisions/0058-alt-names-the-drawing-and-reading-order-is-document-order.md)(`21cf65b`).
      셋 다 "더 적게 여는" 쪽이다: 이름은 **더하고** 대체하지 않으며(대체하면 도형 안 글이 접근성
      트리에서 사라져 같은 SOML이 두 채널에서 다른 내용이 된다), 읽기 순서를 바꾸는 속성은 열지
      않았고(그 순서는 두 타깃에서 문서 순서로 **이미 같다** — 바꾸는 문법을 열면 웹은 `aria-owns`,
      PPTX는 `spTree` 재배열이라 z-순서까지 끌려온다), OOXML `adec:decorative`도 쓰지 않았다.
- [x] GREEN 1 — 값 층 (`660592a`): `<shape alt>`·`<table alt>`이 열렸고 `core/alt.go`가 **빈 값의
      뜻** 하나만 결정한다. 도형·표에서 `alt=""`과 값 없는 `alt`은 **에러**이고 진단이 "속성을
      지운다"고 말한다 — 이름 없음이 이미 기본값이라 `alt=""`은 기본값을 어렵게 적는 표기다
      (`wrap="false"`를 막은 것과 같은 기준). 그림의 `alt=""`은 살아 있다: 그 표기의 뜻을 HTML
      `<img>`가 정의했으므로 비대칭의 근거가 우리 밖에 있다. `soml.plannedAttrs`가 이 커밋으로
      **비었고**, 목록이 비어도 관문이 사는지를 `TestPlannedAttrDiagnostic`이 가상 항목을 세워
      함께 지킨다.
- [x] GREEN 2 — 렌더러 (`6a5935b`): 웹은 도형의 잉크 `<svg>`를 `aria-hidden="true"`에서
      `role="img" aria-label="…"`으로 **뒤집고**(박스 `<div>`가 아니다 — 역할이 `generic`이라
      ARIA가 이름을 금지하고, 거기 적으면 저자의 접근성 텍스트가 조용히 사라진다) 표에는
      `aria-label`만 얹는다(`role="img"`은 셀 항해를 지우고 `<caption>`은 보이는 줄을 웹에만
      더한다). PPTX는 셋 다 `p:cNvPr@descr`이고 **한 함수**(`cNvPrXML`)를 지나므로 세 자리가
      `descr` 규칙에서 갈라질 수 없다 — 빈 `descr`은 내지 않는다(OOXML에 "없음"과 "빈 문자열"의
      구별이 없어 재저장 diff의 잡음만 된다).
- [x] GREEN 3 — 읽기 순서 어서션 (`6a5935b`): `TestReadingOrderIsDocumentOrderInBothTargets`가
      `spTree`의 도형 순서 **와** 웹 DOM 순서를 문서 순서와 맞대고, `order`가 여전히 CSS 선언으로
      나가는지까지 본다(뒤 조건이 없으면 "시각 순서 채널을 지워서" 초록이 되는 길이 열린다).
      이 회귀가 **조용하기** 때문에 어서션이 유일한 감시자다 — 언젠가 "도형을 z-순서로 정렬하자"는
      최적화가 순회를 건드리면 읽기 순서가 뒤집히는데 **그림은 똑같다**.
- [x] 게이트 — 픽스처 `16_a11y` (`8603677`): 1장은 세 태그(`<shape>`·`<table>`·`<image>`)의 이름을
      한 장에 모아 **붙는 자리 셋**과 "이름 없는 도형은 오늘과 한 바이트도 다르지 않다"를 골든에
      담고, 2장은 `order`로 시각 순서를 뒤집어 **먼저 쓴 카드가 화면에서는 오른쪽이면서 먼저
      읽히는** 그림을 담는다. 픽셀 비교는 이 주제를 판정할 수 없으므로(접근성 이름은 픽셀을 한
      개도 바꾸지 않는다) 골든의 한 줄이 회귀를 잡는 유일한 자리다. 두 슬라이드의 완화
      (3.0% · 4.4%)는 측정값 + 앞 픽스처와 같은 폭이고 잔차의 출처는 주제가 아니라 테두리 둘레
      반올림과 스크립트 간격이다 (`internal/testkit/suite.go`의 주석).
- [x] DOC (`3947764`): `docs/reference/appearance.md`에 **접근성** 절 — 세 태그 × 두 타깃의 이름
      표 · 도형의 이름이 박스가 아니라 잉크 svg인 이유 · 표가 `role="img"`도 `<caption>`도 아닌
      이유 · 빈 값이 그림에서만 열린 이유 · `alt`이 캐스케이드 밖인 이유 · **읽기 순서는 문서
      순서**이고 갈라짐에 진단이 없는 이유. `shape.md`·`table.md`에 `alt` 행과 두 진단을 더했다.
      **덤으로 낡음 하나를 잡았다**: 세 문서의 "쓸 수 있는 속성" 사본에 `alt`은 물론 그 전
      이터레이션의 `at`·`x`·`y`·`slot`도 빠져 있었다(문서를 읽은 저자는 쓸 수 있는 속성을 못
      쓴다고 믿는다). 다섯 자리를 `docVocabularies`에 등록해 같은 낡음이 다시 초록으로 지나가지
      못하게 했다.

**남긴 것**: `<text>`에는 `alt`을 열지 않았다 — 글에 이름을 더하는 일은 글을 **대체**하는 일이
되고(스크린 리더는 이름이 있으면 내용을 읽지 않는다) 그것이 이 항목이 거절한 바로 그 모양이다.
시각 순서와 읽기 순서가 갈라진 덱을 **알려 주는** 장치도 없다: `order`의 존재 이유가 그
갈라짐이라 경고로 두면 예제 게이트(경고 0 — ADR 0056)가 우리 스스로 그 채널을 못 쓰게 만든다.
근거가 생기는 자리는 진단이 아니라 `balpyo check`의 별도 리포트일 것이다.

### P2.5 `<chart>` (D14의 나머지) ← **지금 여기**
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
- [x] **접근성이 산출물에 실린다**: 그림·도형·표에 이름을 줄 수 있고(웹은 ARIA, PPTX는
      `p:cNvPr@descr`) 읽기 순서는 두 타깃에서 **문서 순서**다 — 이름은 안의 글을 대체하지
      않으므로 도형에 이름을 준 대가로 그 안의 글이 사라지지 않는다 (P5.3)
- [x] **예제가 썩지 않는다**: 게이트가 예제 셋(28장)을 두 타깃으로 빌드하고 **진단 0건**을
      요구한다 — 경고도 실패다. 장수(12·10·6)까지 선언해서 조용히 빠진 장을 잡는다 (P5.1 —
      골든은 걸지 않는다: 픽스처가 그 일을 한다)

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
| 하이라이팅의 토크나이저 · 토큰이 런이 되는 자리 · 코드 색의 주인 · 탭 | P2.4 | ADR 0051 (Chroma 토큰 스트림만 · 값 층이 합성 인라인 런으로 컴파일(렌더러 무변경) · 색은 테마의 `code.tokens` · 탭은 칼럼 세어 펴고 폭은 테마) |
| 외부 테마가 이름인가 경로인가 · `extends`의 병합 단위 | P4.3 | ADR 0054 (프로젝트 안의 **이름**이고 경로 문법이 없다 · 이름이 같으면 자식이 이기고 없던 이름은 부모 순서 뒤에 붙는다 · 병합은 원시 선언 층에서) |
| 색 토큰의 상한 — 자식이 색을 더할 수 있는가 | P4.3 | ADR 0055 (모든 색 토큰이 `clrScheme` 슬롯을 갖고 자리는 열둘이다 → 자식은 **갈아 끼운다** · 판정은 테마 로딩에서) |
| 예제 게이트가 무엇을 판정하는가 · 골든을 걸 것인가 · CI 잡 | P5.1 | ADR 0056 (두 타깃 · 진단 0건(경고도 실패) · 장수 선언 · **골든 없음** · 게이트는 잡이 아니라 테스트로 늘린다) |
| 문서와 코드의 어휘가 갈리는 것을 무엇이 막는가 | P5.2 | ADR 0057 (문서는 산문을 지키고 **닫힌 목록만** 테스트가 문서에서 읽어 집합으로 대조 · 패턴 불일치도 실패 · 문구 전체 비교와 문서 생성은 열지 않았다) |
| `alt`이 안의 글을 대체하는가 · 이름이 붙는 요소 · 읽기 순서를 바꿀 수 있는가 | P5.3 | ADR 0058 (이름을 **더한다** · 도형은 잉크 svg·표는 `<table aria-label>`·양쪽 PPTX는 `p:cNvPr@descr` · 읽기 순서는 문서 순서이고 바꾸는 문법도 갈라짐의 진단도 없다) |
| 차트: 네이티브 `c:chart` vs 도형 합성 | P2.5 | 대기 |
