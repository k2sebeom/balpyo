# NOTES — 항목별 리뷰 노트

## Iteration 01 · 05-pptx-text — 텍스트 박스를 spTree에 넣고 테마 파트를 토큰 표에서 컴파일
<!-- item: 01/05-pptx-text -->

- **한 것**:
  - `render/pptx/shape.go` 신규 — `<text>` 하나 → `p:sp` 하나 (문서 순서 = spTree 순서 =
    z-순서). `p:nvSpPr`(결정적 id 2부터, 이름 = `text s0.0` 지오메트리 식별자),
    `p:spPr`(`a:xfrm` = 잰 rect의 EMU, `prstGeom rect`, 채움/선 없음), `p:txBody`
    (`bodyPr` 인셋 0 + `noAutofit`, `pPr` `algn`+`lnSpc spcPct`, `rPr` `sz`/`b`/색/폰트).
  - `render/pptx/theme.go` 신규 — clrScheme 12자리와 fontScheme major/minor를
    `soml.Theme` 토큰 표에서 컴파일. 하드코딩 Office 기본값(`dk2=44546A`,
    `accent1=4472C4`) 제거 — `color="accent1"`이 웹과 PPTX에서 같은 색이 된다.
  - `render/pptx/parts.go` — `buildParts(deck, g)`로 지오메트리 수용, 슬라이드 템플릿에
    `{{.Shapes}}`, 테마 템플릿에 `{{.ClrScheme}}`/`{{.FontScheme}}`, 낡은 "Iteration 0"
    주석 갱신. `render/pptx/pptx.go` — 패키지 주석 갱신, `r.geometry` 전달.
  - `render/pptx/units.go` — `LineSpacing`(배수 → spcPct 1/1000 %) 추가.
  - 테스트: `shape_test.go`(구조·외관·schemeClr 보존·이스케이프·개행 일치·빈 요소·
    미매핑 태그·결정성), `theme_test.go`(12자리·폴백·필수 슬롯·중복/슬롯 없는 토큰),
    `soffice_test.go`(LibreOffice 열림 — 외부 테스트 패키지, testkit 순환 회피),
    `units_test.go`(LineSpacing), `pptx_test.go`(testTheme 도입 — core 최소 테마와
    다른 값으로 "표에서 나왔다"를 증명, 배경 = bg 토큰 테스트, 테마 없는 덱 거부).
  - 골든: `00_empty`의 `theme1.xml`만 변경 (전 줄이 토큰 표 컴파일로 설명된다 —
    name balpyo→minimal, dk2/accent1이 muted/accent1 토큰 값, 빈 자리는 폴백).

- **설계 결정과 근거**:
  - **인셋 0**: 웹이 잰 rect는 border box이고 `.balpyo-text`는 padding·border 0이라 곧
    콘텐츠 박스다 (`render/web.contentCSS`). OOXML 기본 인셋(91440/45720)을 두면 좌표는
    맞는데 글자만 밀린다. 대안(기본 인셋 유지 + rect 보정)은 보정값이 두 곳으로 갈라져 버렸다.
  - **굵기 임계값 600**: regular/bold 두 벌 폰트에서 CSS 폰트 매칭이 600부터 bold 면을
    고른다 — 웹이 굵게 그리는 지점과 PPTX가 굵게 내는 지점을 맞춘다.
  - **빈 요소·크기 0 박스도 도형을 낸다**: 건너뛰면 spTree 순서·도형 id가 내용에 따라
    흔들리고, 저자가 빈 요소의 존재를 산출물에서 확인할 수 없다. 크기 0 도형은 OOXML에서
    유효하다. 빈 텍스트는 런 없는 문단으로 남는다.
  - **clrScheme 빈 자리는 가장 가까운 토큰에서 빌린다** (lt2←lt1, dk2←dk1,
    accent2~6·hlink·folHlink←accent1; dk1/lt1/accent1은 필수): 표의 모든 토큰이 자기
    슬롯을 가지므로 빌린 자리를 참조하는 도형은 없다 — 채움은 PowerPoint 테마 색 편집
    UI를 위한 것이고, Office 기본값을 남기면 팔레트 밖 색이 나온다.
  - **개행**: 파서가 원문 개행을 공백으로 접으므로 (soml.Parse) 두 렌더러가 같은 문자열을
    받는다 — `a:br`을 내지 않고, 웹·PPTX 산출물을 한 테스트에서 대조해 못 박았다.
  - **정렬 매핑** start/center/end/justify → `l`/`ctr`/`r`/`just`: OOXML `algn`의
    `l`/`r`은 rtl에서 뜻이 뒤집히는 논리 값이라 논리 이름과 정확히 짝이 맞는다.
  - **테마 없는 덱은 조립 거부**: 기본값으로 채우면 웹과 다른 색을 내는 파일이 조용히
    만들어진다 (`core.Resolve`를 거치라는 에러).
  - **mono 폰트는 리터럴 typeface**: fontScheme에는 major/minor 두 자리뿐 —
    `soml.FontToken.Scheme`이 이미 정한 계약을 따랐다.

- **검증**: `gofmt -l .`(무출력), `go vet ./...`, `go test ./...`, `make golden`
  (theme1.xml만 변경), `make test`(컨테이너) 전부 초록.
  `TestLibreOfficeOpensTextShapes`를 컨테이너에서 -v로 단독 실행해 PASS(0.49s, 복구·경고
  없음 — sofficeNoise 검사 통과)를 확인했다. 한글·`&`·`>`·빈 요소·리터럴 색이 든 덱이다.

- **남긴 것**:
  - `rPr@lang`을 내지 않는다. 덱에 `Lang`이 있으므로 (`balpyo.json`) 붙일 수 있지만,
    맞춤법 검사 언어 외 렌더에는 영향이 없어 뺐다 — 필요해지면 여기서 정한 결정을 뒤집는다.
  - **CSS `line-height` 배수와 OOXML `spcPct`는 기준이 다를 수 있다**: CSS는 폰트 크기의
    배수, PowerPoint 계열은 "단일 줄 간격"의 배수(폰트 고유 줄 높이 기준)로 해석하는
    구현이 있다. 항목 지시대로 `LineHeight × 100000`으로 냈고, 실제 오차는 항목 07의
    픽셀 비교·줄바꿈 임계값 측정에서 드러난다 — 어긋나면 spcPts(절대값) 전환이 대안이다.
  - `fmtScheme`은 하드코딩으로 남는다 (도형 기본 스타일 — Iteration 10의 테마 패키지가
    해체). `<note>` → 노트 슬라이드는 항목 06, `_01_text` 픽스처 활성화·골든은 항목 07.

## Iteration 01 · 06-pptx-notes — `<note>`를 PPTX 노트 슬라이드로 매핑
<!-- item: 01/06-pptx-notes -->

- **한 것**:
  - `render/pptx/parts.go` — 노트가 있는 덱에만 `ppt/notesMasters/notesMaster1.xml`
    (+rels → theme1.xml)과, 노트가 있는 슬라이드마다 `ppt/notesSlides/notesSlideN.xml`
    (+rels → 해당 슬라이드·노트 마스터)을 낸다. `[Content_Types].xml` Override,
    `presentation.xml`의 `p:notesMasterIdLst`, `presentation.xml.rels`의 노트 마스터
    관계(테마 뒤 덧붙임), 슬라이드 rels의 `rId2` → notesSlide. `slideRelsXML` 상수를
    `slideRelsTemplate`로 바꿨다 (노트 유무 조건). `notesMasterLayout`이 노트 페이지
    안 sldImg/body 플레이스홀더 좌표를 계산한다 (이미지 높이만 슬라이드 종횡비 따라 계산).
  - 테스트: `notes_test.go` 신규 — 파트·관계·Content_Types 존재/부재, rels가 정확한
    슬라이드를 가리키는지(파트를 풀어 단정), rId 규칙 불변(테마 rId5 유지·노트 마스터
    rId6), xfrm 상속(노트 슬라이드에 그룹 xfrm 하나뿐), 이스케이프, 웹 `balpyo-notes`와의
    개행 일치, `geom.Refs`가 노트를 세지 않음 + 지오메트리 없는 렌더 성공.
    `soffice_test.go` — 2번 슬라이드에만 노트를 넣어 번호가 비어 뛰는 경우(notesSlide2만
    존재)를 LibreOffice 게이트에 태웠다.

- **설계 결정과 근거**:
  - **notesSlideN의 N = 슬라이드 번호** (slidePart와 같은 n, 노트 없는 슬라이드에서 번호가
    비어 뛴다): 노트 있는 것만 1부터 세면 별도 카운터가 생기고, 카운터와 rels의 슬라이드
    번호가 어긋나는 순간 노트가 엉뚱한 슬라이드에 붙는다 — 파일은 열리므로 조용한 실패다.
    파트 이름을 슬라이드 번호에서 직접 유도하면 그 어긋남을 만들 방법 자체가 없다. OPC
    파트 이름은 임의 문자열이고 소비자는 관계만 따라간다 — 빈 번호(notesSlide2만 있는 덱)를
    LibreOffice가 경고 없이 여는 것을 확인했다.
  - **노트 마스터 관계는 테마 뒤에 덧붙인다** (rId = 슬라이드 수 + 3): presRelsTemplate의
    기존 규칙(rId1=마스터, rId2..=슬라이드, 그다음=테마)의 어느 번호도 밀지 않는 유일한
    자리다. 중간에 끼우면 슬라이드 관계가 한 칸씩 밀려 엉뚱한 파트를 가리킨다.
  - **노트가 하나도 없는 덱은 노트 마스터 자체를 내지 않는다**: 빈 `<note></note>`도
    Notes == ""이라 노트 파트를 내지 않는다 (웹이 balpyo-notes div를 내지 않는 판정과
    같다). 덕분에 00_empty 골든이 이 항목 전후로 바이트 단위로 같다.
  - **좌표는 마스터에서 상속** — 노트 슬라이드 플레이스홀더에 xfrm을 내지 않는다. 노트는
    화면에 렌더되지 않으므로 잰 박스가 없고, 지오메트리를 요구하면 노트만 있는 덱이
    브라우저를 띄우게 된다. 마스터 좌표는 PowerPoint 관례(여백 0.75in, 이미지 y=1.25in)를
    따르되 이미지 높이는 슬라이드 종횡비로 계산한다 — 기본값을 그대로 박으면 4:3 덱에서
    이미지와 본문이 겹친다.
  - **노트 마스터의 테마 = theme1.xml 공유**: 전용 theme2.xml을 내는 대안(PowerPoint가
    저장할 때 하는 방식)은 파트가 하나 늘 뿐 지금은 값이 없다 — 한 파트가 여러 관계의
    대상이 되는 것은 OPC에서 유효하고, LibreOffice 게이트가 통과한다.
  - **개행 정책은 파서 한 곳**: soml.Parse가 원문 개행을 공백으로 접으므로 웹 balpyo-notes와
    PPTX `a:t`가 같은 문자열을 받는다 — 항목 05의 텍스트와 같은 방식으로, 두 산출물을 한
    테스트에서 대조해 못 박았다 (`TestRenderNotesNewlinesAgreeWithWebRenderer`).

- **검증**: `gofmt -l .`(무출력), `go vet ./...`, `go test ./...`, `make test`(컨테이너)
  전부 초록. 00_empty 골든 무변경 (TestFixtures 통과 — 노트 없는 덱은 산출이 같다).
  `TestLibreOfficeOpensTextShapes`를 컨테이너에서 -v 단독 실행해 PASS(0.51s, 복구·경고
  없음) — notesSlide2만 있고 notesSlide1이 없는 덱이다.

- **남긴 것**:
  - **notesStyle 타이포그래피(12pt)는 자리 표시자다** — 테마 토큰 표에서 컴파일하는 일은
    Iteration 10의 테마 패키지 포맷이 맡는다 (notesMasterTemplate 주석에도 남겼다). 노트는
    발표자 화면 전용이라 웹·PPTX 픽셀 일치 계약 밖이고, 지금 하드코딩해도 두 타깃이
    갈라지지 않는다.
  - `docProps/app.xml`에 `<Notes>` 수를 넣지 않았다 — PowerPoint가 채우는 통계 필드지만
    렌더에 영향이 없고, 넣으면 노트 추가마다 app.xml 골든이 흔들린다. 필요해지면 여기서
    정한 결정을 뒤집는다.
  - 발표자 뷰(웹)는 P1, `_01_text` 픽스처 활성화·골든·픽셀 비교는 항목 07, note.md 문서는
    항목 08이다.

## Iteration 01 · 07-fixtures — 01_text 골든 + 픽셀 비교 + 줄바꿈 임계값 측정
<!-- item: 01/07-fixtures -->

- **한 것**:
  - `testdata/fixtures/_01_text` → `01_text` 개명 — 픽스처 러너가 집는다. 참조 갱신:
    `render/web/fixture_test.go`(상수), `core/resolve_test.go`(경로),
    `render/pptx/notes_test.go`(주석).
  - 골든 신규: `01_text/golden/` (index.html + PPTX 파트 24개 — 전 파트를 눈으로 훑었다).
  - `internal/testkit/ink.go` 신규 — `InkBands`: 렌더 PNG의 영역에서 글줄 잉크 띠를 센다
    (배경은 최빈색, 잉크 판정 채널 64). `ink_test.go`에 합성 이미지 단위 테스트.
  - `internal/testkit/text_fixture_test.go` 신규 — 픽스처 수준 검증 4종: 골든 파트 목록
    전수(노트 파트는 슬라이드 1·3에만), 슬라이드 파트에 `srgbClr` 없음(전부 `schemeClr`),
    xfrm == 지오메트리 측정값(+여유)의 EMU, **잉크 띠 대조**(웹/LibreOffice 줄 수·위치·폭).
  - **측정이 드러낸 구현 결함 2건을 고쳤다** (둘 다 이 항목이 측정하기로 한 계약의 구멍):
    - `render/pptx/units.go` `LineSpacing` — `spcPct`(배수) → **`spcPts`(절대 pt)**.
      항목 05가 "어긋나면 spcPts 전환"으로 남겨 둔 대안 그대로다.
    - `render/pptx/shape.go` `TextBoxRect` 신규 — PRD §9.3의 "측정 크기 + 안전 여유"가
      미구현이었다. 폭 +4%, 정렬이 닻을 내린 변은 고정(start→오른쪽, center→양쪽,
      end→왼쪽, justify→여유 없음).
  - `internal/testkit/pixel.go` `DefaultDiffOptions` — MaxRatio 2%(가안) → **2.5%(측정값)**.
  - `PRD.md` §13-1에 측정 수치 기입, `TASKS.md` 테스트·PRD 기입 체크박스 [x].

- **설계 결정과 근거**:
  - **줄 수 판정은 요소 박스 안이 아니라 슬라이드 전체 잉크 띠로 한다**: 박스는 잰 줄
    수만큼의 높이로 고정되므로 다른 엔진이 만든 추가 줄은 박스 **밖**에 그려진다 — 박스
    안만 재면 갈라짐이 측정에서 빠진다 (첫 구현이 실제로 그렇게 조용히 통과했다).
  - **띠 비교는 웹 잉크 vs LibreOffice 잉크**: 지오메트리의 줄 박스(glyph box)는 잉크보다
    커서 잉크 중심과 맞대면 그 차이가 엔진 오차처럼 보인다. 줄 박스는 줄 수와 요소
    이름에만 쓴다.
  - **spcPts 전환**: spcPct는 CSS(폰트 크기 배수)와 LibreOffice(폰트 고유 줄 높이 배수)의
    기준이 달라 같은 150%가 글줄을 +8~13px 밀었다. 절대값 전환 후 +2~5px (첫 기준선
    배치의 엔진 차만 남는다). 실험은 컨테이너 빌드 pptx를 XML만 바꿔 LibreOffice로
    다시 렌더해 확인한 뒤 구현했다.
  - **안전 여유 4%**: LibreOffice는 한글↔라틴 경계에 스크립트 간격을 넣어 혼합 문자 줄을
    최대 +3.3% 넓게 그린다 (OOXML 속성으로 끌 수 없다). 여유 2.4%였던 002 본문이 정확히
    이것 때문에 1줄→2줄로 갈라졌고, 4%는 측정 최대치 + 여유다. 더 키우지 않는 이유:
    반대 방향 위험(웹이 접은 단어가 넓힌 박스에 들어가 줄이 줄어드는 것)이 커진다.
  - **픽셀 허용 오차 2.5% (채널 32 유지)**: 측정 최대 2.1%(표지 — 글자가 클수록 잔차
    픽셀이 많다)이고 구성이 전부 엔진 소관(기준선 +2~5px, 스크립트 간격 글리프 흘림,
    래스터라이저 AA)이다. 구조적 갈라짐은 잉크 띠 검사가 따로 잡으므로 비율을 더 조이지
    않았다 — 헐렁하게 여는 것도, 잔차를 비율로 잡는 척하는 것도 피했다.
  - 회귀 임계값: 줄 수 차 **0**, 글줄 세로 중심 **±6px**, 글줄 잉크 폭 **±4%**
    (`text_fixture_test.go` 상수 — 근거 주석 포함).

- **검증**: `gofmt -l .`(무출력), `go vet ./...`, 호스트 `go test ./...`, `make test`
  (컨테이너) 전부 초록. `make golden` 재실행 시 무변경(결정성). `make review` 갤러리와
  diff 이미지를 눈으로 확인 — 남은 diff는 2~5px 이동 잔상뿐, 줄 배치 갈라짐 없음.
  CLI 경로: `go run ./cmd/balpyo build --project testdata/fixtures/01_text --target
  html,pptx` 종료 0, index.html + text-deck.pptx 산출. 측정표 (컨테이너, 요소 = 지오메트리
  식별자):

  | 요소 | 텍스트 | 웹 줄수 | 최장 줄 px | 여유 px(%) | LO 줄수 | 세로 중심 Δpx | 잉크 폭 Δ% |
  |---|---|---|---|---|---|---|---|
  | s0.0 | Platform Team | 1 | 138 | 1014 (736.2%) | 1 | +2.0 | +0.0 |
  | s0.1 | 2026 아키텍처 리뷰 | 1 | 533 | 619 (116.1%) | 1 | +2.0 | +2.7 |
  | s0.2 | 상반기 회고와… | 1 | 262 | 890 (340.4%) | 1 | +5.0 | −1.2 |
  | s1.0 | 올해의 세 가지 변화 | 1 | 293 | 859 (293.1%) | 1 | +3.0 | −1.0 |
  | s1.1 | 모놀리스 분리를… | 1 | 1125 | 27 (**2.4%**) | 1 | +4.0 | **+3.3** |
  | s1.2 | 2026 H1 | 1 | 70 | 1082 (1557.6%) | 1 | +3.0 | −1.5 |
  | s2.0 | 질의응답 (Q and A) | 1 | 477 | 675 (141.5%) | 1 | +2.0 | +2.5 |

  최소 여유 비율 2.44% (s1.1). 픽셀 diff: 슬라이드별 2.097% / 1.962% / 1.203%
  (채널 32 초과 픽셀 비율). 수정 전에는 s1.1이 LibreOffice에서 2줄로 갈라졌고
  (스크립트 간격 +2.5% > 여유 2.4%), spcPct 시절 세로 오프셋은 +8~13px였다.

- **남긴 것**:
  - **사람 확인 남음 (TASKS ③)**: `make review` 갤러리와 `review/01_text/text-deck.pptx`를
    Keynote/PowerPoint로 직접 연다. 스크립트 간격은 LibreOffice 렌더러의 동작이라
    PowerPoint/Keynote에서는 웹에 더 가깝게 나올 가능성이 높다 — 확인되면 좋고, 아니면
    그 관찰을 여기 남긴다. 리뷰 포인트 체크박스는 그래서 아직 [ ]다 (항목 08이 마무리).
  - 호스트 `go test`가 01_text 골든 비교를 통과하는 이유는 박스 rect가 폰트 독립이어서다
    (폭 = 컨테이너 폭, 높이 = 줄 수 × line-height). 호스트 폰트가 줄 수를 다르게 접으면
    호스트만 빨개질 수 있다 — 게이트는 컨테이너이므로 수용했다.
  - `textAligns`(text_fixture_test.go)는 슬라이드 직속 `<text>`만 안다 — 중첩 요소가
    생기면 geom.Refs와 같은 재귀로 바꿔야 한다 (지금은 t.Fatalf 가드).
  - `InkBands`는 단색 배경 가정(최빈색)이다 — 도형/이미지가 생기는 iteration에서는 잰
    영역을 좁히거나 배경 판정을 바꿔야 할 수 있다.
  - justify 정렬은 안전 여유가 없어 스크립트 간격 위험이 남는다 (`TextBoxRect` 주석) —
    justify를 쓰는 픽스처가 생기면 그때 측정한다.

## Iteration 01 · 08-docs — Iteration 1 레퍼런스·내부 문서와 ADR 3건
<!-- item: 01/08-docs -->

- **한 것**:
  - 태그 레퍼런스 신규: `docs/reference/soml/slide.md`·`text.md`·`note.md` — 통일 형식
    (속성 표 → SOML 예제 → 웹/PPTX 렌더 결과 → 빌드 에러 조건). 예제는 전부
    `testdata/fixtures/01_text`의 실제 슬라이드이고, 에러 조건 표의 메시지는
    `soml/parse.go`·`core/resolve.go`·`core/validate.go`의 실제 진단 문자열이다.
  - `docs/reference/appearance.md` 신규 — role/size/color/align의 웹(CSS)/PPTX(OOXML)
    매핑 표, 캐스케이드 순서, **토큰 색 `var(--balpyo-color-*)` ↔ `a:schemeClr` 보존
    계약을 본문으로** 명문화. minimal 테마의 토큰 표(색 4·폰트 3) 포함.
  - `docs/reference/cli.md` 신규 — `build`의 옵션·우선순위·종료 코드·진단 출력 형식과
    `version`/`help`. init/server는 없으므로 쓰지 않았다.
  - `docs/internals/geometry-pass.md` 신규 — 흐름도, 슬라이드 상대 좌표, 덱 대조, 단위
    경계(EMU·pt는 render/pptx만), 텍스트 박스 고정 전략(+4% 여유·인셋 0·spcPts). 코드
    주석을 옮기지 않고 전체 그림과 순서만 잇고 각 결정은 주석·ADR로 링크했다.
  - `docs/internals/architecture.md` — 덱 모델 절 완성 (노드 종류, 소유 경계 3층, 외관이
    붙는 자리, 지오메트리 곁 표 이유). 파이프라인 그림·패키지 표에 `render/geom`을 반영
    (이전 판은 지오메트리 패스를 render/pptx 소유로 적어 실제 패키지 구조와 어긋났다).
  - ADR 3건: `0003-inline-style-appearance.md`(인라인 style — data-balpyo-id 생성 규칙
    대안을 버린 이유 포함), `0004-geometry-side-table.md`(곁 표 + 생성자 주입),
    `0005-line-spacing-and-slack.md`(spcPts + 안전 여유 4% — 항목 07의 측정이 근거).
    `docs/decisions/index.md`와 TASKS.md 결정 로그 표 갱신 (I1 임계값 행이 0005를 가리킨다).
  - `mkdocs.yml` nav에 새 페이지 6건 추가, `docs/reference/index.md`·`docs/index.md`의
    "현재 상태"를 구현 완료 상태로 갱신, TASKS.md Iteration 1의 구현·문서 체크박스 [x].

- **설계 결정과 근거**:
  - **ADR은 3건으로 판단했다.** 항목 지시의 2건에 더해, 항목 07의 spcPts + 안전 여유는
    모든 텍스트 박스에 걸리는 횡단 결정이고 측정 데이터가 근거라 0005로 남겼다. NOTES의
    다른 단독 결정들(인셋 0, 굵기 임계 600, notesSlideN=슬라이드 번호, clrScheme 빈 자리
    차용, 개행 접기)은 **국소적 매핑 선택**이라 ADR로 올리지 않았다 — 각 결정의 왜는
    코드 주석과 NOTES에 이미 있고, 전부 ADR로 올리면 색인이 결정의 무게를 구별하지
    못하게 된다.
  - 에러 조건 표는 구현의 진단 문자열을 그대로 옮겼다 — "문서와 메시지가 갈라지면 문서가
    틀린 것"이라는 항목 규칙대로, 문서가 메시지의 사본임을 명시적으로 남긴다.
  - 렌더 결과 설명의 OOXML 구조(cSld@name, ph type="body", bgRef 등)는 골든 파트에서
    직접 확인하고 적었다.

- **검증**: `make docs`(mkdocs --strict) 통과. 코드 변경이 없는 항목이라 Go 게이트는
  항목 07 커밋의 초록이 유효하고, 커밋 전에 `gofmt -l .`(무출력)·`go vet ./...`·
  `go test ./...`·`make test`를 다시 돌려 확인했다.

- **남긴 것**:
  - **낡은 주석 2곳** (문서 항목이라 고치지 않았다 — 코드 변경 금지 규칙): 항목 07이
    spcPct → spcPts로 바꾼 뒤에도 `soml/appearance.go`의 `TextStyle.LineHeight` 주석과
    `render/web/style.go`의 줄높이 주석이 "OOXML lnSpc@spcPct가 배수를 받는다"고 말한다.
    다음 코드 항목이 고친다.
  - **Iteration 1의 남은 칸은 리뷰 포인트 체크박스 하나다** — LibreOffice 쪽 수치는
    확정됐지만 Keynote/PowerPoint에서 같은 줄바꿈을 유지하는지는 사람이
    `review/01_text/text-deck.pptx`를 직접 열어 확인하는 단계(TASKS ③)가 남았다.
    끝나지 않은 것을 [x]로 만들지 않았다.
  - `docs/index.md` 본문의 목표 명세(CLI 전체 표면, 태그 카탈로그)는 그대로 두었다 —
    개요 문서는 비전을 말하고, "지금 동작하는 것"의 경계는 상태 노트와 레퍼런스가 긋는다.

## Iteration 02 · 01-parser-subset — 파서가 HTML 텍스트 서브셋 어휘를 연다
<!-- item: 02/01-parser-subset -->

- **한 것**:
  - `soml/subset.go` 신규 — 요소 종류 카탈로그. `ElementKind`(블록/리스트 컨테이너/리스트
    항목/인라인) + `KindOf`, void 판정 `IsVoidElement`, `ListItemElement`/`BreakElement`
    상수. 종류별 이름은 순서 있는 슬라이스(진단·문서 순서)이고 색인 맵은 거기서 파생된다.
  - `soml/parse.go` — `elementSpecs`를 카탈로그에서 `buildElementSpecs`로 파생 (태그를
    카탈로그에 더하면 문법이 따라온다). 콘텐츠 모델 3종 추가: `modelInline`(문자 데이터 +
    인라인 태그 — `<text>`·블록·`<li>`), `modelPre`(공백 원문 보존), `modelVoid`(`<br>`).
    `checkPlacement`가 부모 대신 **열린 조상 스택**을 받는다 (`<a>` 안의 `<a>`는 사이에
    인라인이 끼어도 에러). 배치 진단: li 자리, 중첩 리스트("아직"), 블록 안 블록, a 안 a,
    인라인 전용 모델 위반. `<ul>` 안 문자 데이터의 힌트는 `<li>로 감싼다`. void 위반은
    여는 태그에서 한 번만 진단하고 내용은 조용히 지나간다. `trimPreEdges`가 pre 양 끝
    개행을 하나씩만 벗긴다 (CRLF 포함).
  - `soml/subset_test.go` 신규 — 정상 트리 표(트리 전체를 `dumpNodes` 한 줄로 단정),
    에러 표(line:col + 문구 전수), 인라인 경계 공백 계약 표, pre 공백 보존 표, 비활성
    픽스처를 읽는 파싱 테스트.
  - `soml/parse_test.go` — 어휘 확장으로 계약이 바뀐 두 단정만 갱신: 어휘 진단 테스트가
    `<h1>`(이제 어휘 안) 대신 `<h6>`을 쓰고, text 안의 text 메시지가 "문자 데이터와
    인라인 태그만"이 됐다. 그 외 기존 테스트는 전부 그대로 통과한다.
  - `testdata/fixtures/_02_text_subset/` 신규 (비활성 — `_` 접두): 제목 위계·리스트·
    인라인 서식·pre·역할 등가 확인 5장. 활성화(골든)는 항목 08의 몫이다.
- **설계 결정과 근거**:
  - **인라인 경계 공백 계약**: 요소(블록이든 인라인이든) 양 끝의 공백은 잘리고, 형제
    사이의 공백은 내용이다. `<strong>배포 </strong>자동화`의 태그 안쪽 끝 공백은 잘린다 —
    공백은 태그 밖에 쓴다. HTML처럼 노드 경계를 넘는 접기(인접 공백을 하나로)를 하지
    않은 이유: 닫힌 자식 목록을 형제 상황에 따라 되돌아가 고쳐야 해서 접기 결과가 문맥
    의존이 되고, 균일한 "양 끝 트리밍"이 런 분할(항목 06)에 더 예측 가능한 입력을 준다.
    이 규칙으로 웹(브라우저 재접기)과 PPTX(접기 없음)가 같은 문자열을 본다.
  - **`<br>`은 자기 닫힘만 받는다**: 내용이 없어도 `<br></br>`는 에러다 — "내용이 있을
    때만 에러"보다 계약이 단순하고, 진단이 여는 태그 한 곳에서 한 번만 난다. 진단 후에도
    스택에 올려 `</br>`가 "짝 없는 닫는 태그" 진단을 덧내지 않게 했다.
  - **href 없는 `<a>`의 진단은 core(항목 04)의 몫**이다 — 파서는 이름 allowlist만 보고,
    "속성이 있어야 한다"는 값 층의 요구다. `buildElementSpecs`의 주석에 남겼다.
  - **`<span size>`는 allowlist가 막는다** (color·weight만 연다): 런 하나가 커지면 웹의
    줄 박스 높이가 변하는데 PPTX 줄간격은 문단 절대값(spcPts)이라 글줄이 갈라진다 —
    core가 `size="%"`를 막은 것과 같은 근거.
  - **pre는 양 끝 개행을 하나씩만 벗긴다**. `code\n  </pre>`처럼 개행 뒤 들여쓰기가 붙으면
    "직전"이 아니므로 벗기지 않는다 — pre 안 공백은 내용이라는 원칙이 우선한다. v1의
    pre는 문자 데이터만 담는다 (`<pre><code>` 관용은 코드 하이라이팅 결정과 함께 연다).
  - **중첩 리스트는 에러**이고 메시지가 "아직 지원하지 않는다"고 말한다 — PPTX `lvl` 매핑
    없이 열면 한 타깃만 동작한다.
  - **`reveal`은 열지 않았다** (PRD §7.7 예제에 있지만): 웹 런타임 reveal은 Phase 2이고,
    렌더 없는 속성을 어휘에 열면 닫힌 어휘가 막으려던 조용한 무시가 된다.
  - **인라인은 슬라이드 직속이 될 수 없다** — 문단 없는 런은 두 타깃 어디에도 대응하는
    구조가 없다. `<slide>`의 자식은 기존 2종 + 블록 + 리스트 컨테이너뿐이다.
- **검증**: `gofmt -l .`(무출력), `go vet ./...`, `go test ./...`, `make test`(컨테이너)
  전부 초록. 새 픽스처 5장은 `TestParseTextSubsetFixture`가 진단 0건으로 파싱함을 확인.
  골든 변경 없음 — 렌더 매핑이 없는 항목이라 기존 픽스처 산출물은 그대로다.
- **남긴 것**:
  - 항목 02~04가 `KindOf`로 역할 등가·캐스케이드·인라인 해석을 얹고, 항목 05~07이 같은
    카탈로그로 렌더 매핑(블록 vs 런)을 만든다. 항목 08이 `_02_text_subset`의 `_`를 떼고
    골든을 만든다.
  - **열린 질문 — 블록 태그의 속성**: `<h1 align="center">`처럼 블록에 `<text>`의 외관
    속성을 열지 PRD가 정하지 않아 **일단 전부 닫았다** (속성 없음). 역할 등가(항목 03)가
    "블록 = `<text role>`의 축약"을 어디까지 밀지 정할 때 함께 정해야 한다.
  - macOS Finder가 만든 `.DS_Store`가 골든 디렉토리에 들어와 testkit 대조를 깨뜨렸다
    (추적되지 않은 파일이라 삭제로 해결). `.gitignore` 추가는 항목 범위 밖이라 하지 않았다.

## Iteration 02 · 02-cascade — 외관 해석기 4단 캐스케이드 완성 + `variant`
<!-- item: 02/02-cascade -->

- **한 것**:
  - `core/cascade.go` 신규 — 캐스케이드 병합 기계. `textPatch`(포인터 필드로 "정하지
    않음"과 0값을 구별하는 부분 외관)와 유일한 병합 지점 `mergeText`, 패치 리터럴용
    `ptr[T]`. Iteration 4의 도형 외관이 같은 기계를 재사용할 자리다.
  - `core/resolve.go` — `resolveText`를 4단 캐스케이드로 재구성: 역할 기본값(완전한 외관)
    위에 `textComponentDefault()` ← `variantPatch` ← `inlinePatch`를 낮은 단부터 얹는다.
    우선순위는 `mergeText` 호출 세 줄의 순서가 전부다. 모르는 variant는 `warnf`(경고) +
    빈 패치 폴백. 기존 인라인 속성 해석(size/color/align)의 값 어휘 검사·진단은 문구와
    위치 그대로 `inlinePatch`로 이사했다.
  - `core/theme.go` — minimal 테마의 텍스트 variant 표 `textVariants()`:
    `kicker`(size 18 + weight 700 + accent1), `lead`(size 32, 부분 패치). 조회
    `textVariantPatch`, 경고 메시지용 `textVariantNames`(순서 있는 슬라이스 — 결정적
    출력). `textComponentDefault()`는 빈 패치다 (아래 "설계 결정").
  - `soml/parse.go` — `<text>`에 `variant` 속성 개방 (`role, variant, size, color, align`
    순서), 블록 태그(h1~h3, p, blockquote, pre)에 `variant`만 개방. 인라인·리스트
    컨테이너·`<li>`에는 열지 않았다 — 근거는 buildElementSpecs 주석에.
  - `soml/appearance.go` — `TextStyle.LineHeight`의 낡은 주석 수정 (01/08의 "남긴 것"):
    "spcPct가 배수를 받는다" → PPTX는 spcPts 절대값으로 환산한다 (01/07의 측정 결과 반영).
  - 테스트: `core/cascade_test.go`(우선순위 표 — 단마다 하나씩 빼 가며, 부분성, 0값 구별,
    variant 표의 구별성), `core/resolve_test.go`(variant > 역할, 인라인 > variant, 부분성,
    모르는 variant의 경고 severity·문구·ValueSpan 위치·폴백 동등성, 값 없는/빈 variant
    에러), `internal/pipeline/pipeline_test.go`(경고는 빌드 통과 + `--strict` 승격 시 실패),
    `soml/subset_test.go`(variant 파싱 정상 케이스 + 속성 어휘 에러 4건 갱신·추가).

- **설계 결정과 근거**:
  - **패치 = 포인터 필드 구조체**. 대안이던 "완성 TextStyle + 필드 마스크"는 마스크와
    값이 따로 놀 수 있어 버렸다 — 포인터는 "정함"과 "값"이 한 몸이다.
  - **Role은 패치에 없다**: 역할은 외관 값이 아니라 캐스케이드의 바닥을 고르는 축이다.
    variant가 역할을 바꿀 수 있으면 병합 도중 바닥이 바뀌어 순서가 정의되지 않는다.
  - **minimal 테마의 컴포넌트 기본값(3단)은 빈 패치다.** 역할 기본값이 이미 완전한
    외관이라, 여기서 필드를 정하면 역할과 다른 값 → 모든 `<text>`의 외관이 바뀌어
    Iteration 1 골든이 움직이고, 같은 값 → 아무것도 증명하지 못하는 중복이다. 단은
    구조로 존재하고(`resolveText`가 항상 병합), 우선순위는 cascade_test가 합성 패치로
    증명한다. 실제 값은 Iteration 10 테마 패키지가 처음 넣는다.
  - **variant 이름·값**: 항목 지시의 예시를 따라 `kicker`/`lead`. 값은 캐스케이드 각 단이
    구별되도록 골랐고(테스트가 강제), `lead`는 의도적으로 size만 정해 부분성의 실물
    예시가 되게 했다.
  - **값 없는/빈 `variant`는 에러다** (모르는 variant의 경고와 다르다): 다른 외관 속성과
    같은 `hasValue` 정책 — "값 없음"은 폴백할 뜻조차 정할 수 없는 형태이기 때문이다.
  - **경고 문구**: `variant "hero"가 테마에 없다 — 기본 외관으로 계속한다 (이 테마의
    variant: kicker, lead)` — 색 토큰 진단과 같은 태도로 테마의 어휘 목록을 싣고,
    폴백했다는 사실을 문구에 남겼다 (경고를 무시한 저자가 결과를 예측할 수 있게).
  - **블록의 variant 개방으로 기존 파서 진단 문구가 바뀌었다**: `<h1 align>`이
    "속성을 쓸 수 없다 (align)"에서 "알 수 없는 속성 align (쓸 수 있는 속성: variant)"로.
    허용 어휘가 비어 있지 않게 되면서 진단 분기가 바뀐 것이고, 테스트를 그에 맞춰 갱신했다
    (01의 "열린 질문 — 블록 태그의 속성"에 대한 부분 답이기도 하다: variant만 연다).

- **검증**: `gofmt -l .` 무출력, `go vet ./...`, `go test ./...` 로컬 전부 통과,
  `make test`(컨테이너, LibreOffice·Chromium 포함) 전부 통과. `git status`에
  testdata/골든 변화 없음 — 캐스케이드 재구성이 Iteration 1 덱의 해석 결과를 바이트
  단위로 보존한다는 증거다. RED를 먼저 확인했다 (구현 전 core 빌드 실패 + soml/pipeline
  테스트 실패).

- **남긴 것**:
  - 블록 태그의 `variant`는 **파싱만** 된다 — core는 `<text>`만 해석하므로, 항목 03이
    블록을 역할 등가로 해석할 때 같은 `resolveText`(또는 그 캐스케이드)를 태워야 한다.
    렌더 매핑 없는 블록은 렌더러가 명시적으로 멈추므로 조용한 무시는 없다.
  - `render/web/style.go`의 같은 낡은 spcPct 주석은 항목 05가 고친다 (지시대로 남겨 둠).
  - **열린 질문 — attrExample의 "kicker"**: 값 없는 `variant`의 에러 예시가 minimal
    테마의 사유 어휘를 인용한다. 테마가 파일로 나가는 Iteration 10에서 예시를 테마
    어휘에서 뽑을지, 중립 예시로 바꿀지 정해야 한다.
  - Iteration 10이 variant 표·컴포넌트 기본값을 테마 패키지 파일로 옮길 때 `ptr` 헬퍼와
    하드코딩 표(`textVariants`)는 로더로 대체된다 — 병합 기계(`textPatch`/`mergeText`)는
    그대로 남는 경계로 설계했다.

## Iteration 02 · 03-block-roles — 블록 태그 ≡ 텍스트 역할 등가
<!-- item: 02/03-block-roles -->

- **한 것**:
  - `soml/subset.go` — `TagTextRole` 신규: 태그 → 역할 등가 표 (h1→title, h2/h3→heading,
    p/blockquote/li→body, pre→code). 언어 계약이므로 테마가 아니라 soml에 둔다.
  - `soml/parse.go` — 블록 태그의 속성 allowlist를 `variant` 하나에서
    `variant, size, color, align`으로 확장 (`<text>`의 어휘에서 role만 뺀 것).
  - `core/theme.go` — `textComponentDefault()`가 태그를 받는 `textComponentDefault(tag)`가
    됐다: h3는 size 28, blockquote는 muted 색, 나머지 태그와 `<text>`는 빈 패치.
  - `core/resolve.go` — `resolver.node`의 해석 대상이 `<text>`에서 텍스트 역할을 갖는 요소
    전부(블록 6종 + `<li>`)로 넓어졌다. 역할 출처를 고르는 `textRole`(속성 vs 등가 표)을
    분리하고 `resolveText(elem, role)`로 역할을 주입한다. `<ul>`/`<ol>`의 Appearance를
    채우지 않는 근거를 `node` 주석에 남겼다.
  - 테스트: `core/resolve_test.go`(등가 깊은 비교 4쌍, h3·blockquote 태그 기본값과 부분성,
    인라인 > 태그 기본값, 블록의 variant 캐스케이드 순서 3단 확인, li=body + 컨테이너
    무외관, 블록 인라인 속성의 값 진단 line:col), `soml/subset_test.go`(`TagTextRole` 표
    전수 + 표 밖 이름 + 블록 전수 커버리지, 블록 인라인 속성 파싱 정상 케이스,
    `<h1 role>` 거부 진단 — 기존 `<h1 align>` 거부 테스트를 대체).
- **설계 결정과 근거**:
  - **h1 → title** (항목 지시의 표를 따르되 근거를 적었다 — TagTextRole 주석): 슬라이드는
    문서가 아니라 화면 하나이고 v1의 블록은 문단이지 문서 섹션이 아니므로, 슬라이드에서
    가장 큰 표제 = 그 슬라이드의 제목이다. h1을 heading으로 두면 title 역할에 대응하는
    블록 표기가 없어져 서브셋만 쓰는 저자가 표지를 만들 수 없다. ADR 승격은 항목 09가
    한다 (09 지시가 이 결정을 ADR 목록에 올려 두었다).
  - **역할 대응은 언어(soml), 태그별 기본값은 테마(core/theme.go)**: 등가는 PRD §7.7의
    계약이라 테마가 바꿀 수 있으면 문서의 뜻이 테마마다 달라진다. 기본값은 외관이므로
    테마 소유 — Iteration 10의 components.json이 기본값 표만 파일로 옮기고 등가 표는
    soml에 남는 경계다.
  - **h3의 크기는 28**: heading(36)과 body(24) 사이의 반 단계. heading의 lineHeight
    1.3을 그대로 받으므로 줄박스가 28×1.3 ≈ 36.4px로 본문 한 줄(36px)의 리듬에 얹힌다 —
    36/1.3 ≈ 27.7의 최근접 정수다. 기하 평균 √(24×36) ≈ 29.4는 줄박스 38.2px로 리듬에서
    더 멀어 버렸다. size 외 필드는 패치하지 않는다 (부분성).
  - **blockquote는 색(muted)만**: 왼쪽 괘선 같은 시각 장식은 도형 외관이 없는 이번
    iteration에서 열지 않는다 (항목 지시대로) — Iteration 4의 도형 외관이 들어온 뒤의 몫.
  - **블록에 size/color/align을 열었다** (01의 열린 질문에 대한 답): 항목 지시가
    "`size`/`color`/`align`/`variant` 인라인 속성은 `<text>`와 같은 규칙으로 얹힌다"고
    명시한다. role만 닫는다 — `<p role="title">`은 두 표기의 등가를 깬다. 이로써 기존
    파서 진단 테스트(`<h1 align>` 거부)가 무의미해져 `<h1 role>` 거부로 교체했다 —
    등가 계약 자체를 단정하는 더 나은 테스트다. `<li>`와 인라인·리스트 컨테이너는
    이전 결정대로 속성을 열지 않았다.
  - **컨테이너(<ul>/<ol>)의 Appearance는 nil**: 마커·들여쓰기를 렌더러가 컨테이너에서
    무엇으로 읽을지는 항목 05·07이 정한다 — 여기서 미리 채우면 그 결정 전에 두 렌더러가
    각자 추측한다. 테스트가 nil임을 단정한다.
- **검증**: RED 먼저 확인 (구현 전 soml 빌드 실패 + core 테스트 8건 실패 — 블록에 외관
  없음·파서가 size/color 거부). 구현 후 `gofmt -l .` 무출력, `go vet ./...`,
  `go test ./...` 로컬 전부 통과, `make test`(컨테이너, LibreOffice·Chromium) 전부 통과.
  `git status`에 testdata/골든 변화 없음 — Iteration 1 픽스처에는 블록이 없고 렌더
  매핑도 이 항목이 건드리지 않으므로 산출물이 바이트 단위로 그대로다.
- **남긴 것**:
  - 블록·li의 Appearance가 이제 채워지지만 **렌더 매핑은 아직 없다** — 렌더러는 모르는
    태그에서 명시적으로 멈추므로(닫힌 어휘) 조용한 무시는 없다. 항목 05(웹)·06(PPTX 런)·
    07(리스트)이 이 확정 외관을 소비한다. 인라인 태그의 Appearance는 여전히 nil이다
    (항목 04).
  - caption 역할에는 대응 블록 태그가 없다 (figcaption은 어휘 밖) — 문서(항목 09)의
    등가 표에 이 비대칭을 적어야 한다.
  - TASKS의 "텍스트 역할 계약" 체크박스를 [x]로 바꿨다 — 역할 5종의 해석 계약(`<text
    role>` + 블록 등가)이 이 항목으로 완성됐다. "블록 태그 ≡ 역할 등가성" 테스트
    체크박스는 인라인 런·리스트 렌더까지 묶여 있어 건드리지 않았다 (항목 06·07 몫).
  - ADR 승격 대기: 태그→역할 표의 소유 경계 + h1→title (항목 09의 ADR 목록 1번).

## Iteration 02 · 04-inline-resolve — 인라인 태그의 런 스타일 해석
<!-- item: 02/04-inline-resolve -->

- **한 것**:
  - `soml/appearance.go` — `TextStyle`에 `Italic`(em)·`Underline`(a) 추가. 웹 CSS
    (`font-style`/`text-decoration`)와 OOXML(`rPr@i`/`rPr@u`)이 둘 다 불리언으로 직접
    받는다. `Appearance`에 `Link`(하이퍼링크 목적지, href 원문) 추가 — 스타일이 아니라
    행동이므로 `TextStyle` 밖이고, "렌더러가 Attrs를 다시 읽지 않는다" 계약의 연장이다.
  - `core/inline.go` 신규 — `resolveInlines`: 부모의 **확정** `TextStyle`을 복사해 태그별
    델타를 얹는다 (strong=굵기 700, em=이탤릭, code=mono 폰트, a=밑줄+링크 색+Link,
    span=자기 color/weight, br=서식 없음). `href`(없음/값 없음/빈 값 3형 에러, URL 문법은
    열린 값), `weight`(normal/bold 어휘 + CSS 수치 힌트) 해석 함수.
  - `core/resolve.go` — `node`가 텍스트 역할 요소를 해석한 뒤 `resolveInlines`로 인라인
    자식을 마저 확정하고 순회를 끝낸다 (modelInline이라 자식은 문자 데이터+인라인뿐).
    `attrExample`에 `href`/`weight` 예시 추가.
  - `core/inline_test.go` 신규 — 델타 표(부모 스타일의 변형 함수로 깊은 비교 — 델타 밖
    필드 변화를 잡는다), 상속 표(중첩·역순 중첩·span 아래·블록 인라인 속성 아래·중첩 em·
    li 안), strong의 굵은 부모 유지, title 안 code의 56px mono, a의 색·밑줄·Link,
    href 열린 값 4종, href 에러 3형(line:col·문구), span↔text 색 진단 문구 동일성,
    weight 어휘·진단 6종, 인라인 전부 Appearance non-nil(+br은 nil) 커버리지.
- **설계 결정과 근거**:
  - **링크 색 = accent1 토큰 재사용** (hlink 전용 토큰과 저울질 — ADR 후보): PPTX 테마
    파트가 hlink 슬롯을 accent1에서 빌려 채우므로(`render/pptx/theme.go`의 fallback,
    `TestClrSchemeFallbacks`가 고정) 웹이 같은 토큰을 쓰면 두 산출물의 링크 색이 같은
    표의 같은 값에서 나온다. 전용 토큰을 버린 이유: 표준 토큰 어휘(PRD §8.4)에 없는
    이름이 저자의 `color=` 어휘·진단 목록·CSS 변수 산출에 들어가 골든과 문서가 전부
    흔들린다. 테마가 파일로 나가는 Iteration 10에서 재검토한다 — 그때 테마가 hlink
    토큰을 명시하면 core의 링크 델타도 함께 옮겨야 한다 (지금은 fallback 규칙과 이
    델타가 짝이다).
  - **`<span weight>` 어휘는 normal(400)·bold(700) 둘로 닫았다** (PRD가 값 어휘를
    명시하지 않아 정한 것): OOXML 굵기는 켬/끔(`rPr@b`)뿐이라 CSS 9단계 수치 축을 열면
    중간 굵기(600 등)가 웹에서만 동작한다 — `<span size>`·`size="%"`를 막은 것과 같은
    근거. 수치 400/700은 align의 left/right처럼 대응 이름을 짚어 주고, 그 외 수치는
    "두 단계"임을 설명한다.
  - **strong은 `max(부모, 700)`**: 더 굵은 단계가 없으므로 700 이상인 부모(h1 등)
    아래에서는 그대로다 — 깊은 비교 테스트가 "h1 안의 strong ≡ h1 스타일"을 고정한다.
  - **중첩 em은 토글하지 않는다**: 토글은 "몇 겹인가"를 세어야 뜻이 정해지는 규칙이라
    컴포넌트 조합(Iteration 11)에서 결과를 예측할 수 없게 된다 — v1은 "강조는 이탤릭".
  - **br의 Appearance는 nil**: void — 텍스트 흐름의 분절점일 뿐 서식이 아니다. 렌더러의
    런 분할(항목 05·06)이 `soml.BreakElement`를 이름으로 특별 취급한다. "인라인 전부
    non-nil" 테스트의 유일한 예외로 명시했다.
  - **href 에러는 hasValue를 타지 않는다**: 공용 문구의 "지우면 기본값을 쓴다"가 href에는
    성립하지 않는다 (지워도 여전히 에러). 3형(속성 없음→NameSpan, 값 없음→속성 이름,
    빈 값→값 위치)을 따로 쓰되 태도는 같다. href가 틀려도 밑줄·색 스타일은 확정된다 —
    진단이 한 번에 다 나오는 기존 정책.
  - **span의 color는 `<text color>`와 해석 함수(`resolver.color`)를 공유**하고 테스트가
    두 진단의 문구 동일성을 단정한다 — 값 해석이 두 벌이 되는 순간을 잡는 회귀 장치.
- **검증**: RED 먼저 확인 (구현 전 `go vet ./core`가 Italic/Underline/Link 부재로 빌드
  실패). 구현 후 `gofmt -l .` 무출력, `go vet ./...`, `go test ./...` 로컬 전부 통과,
  `make test`(컨테이너, finch) 전부 통과. `git status`에 testdata/골든 변화 없음 —
  활성 픽스처(00/01)에 인라인 태그가 없고 렌더 매핑도 건드리지 않아 산출물이 바이트
  단위로 그대로다. TASKS 체크박스는 바꾸지 않았다 (남은 박스들은 렌더 매핑까지 묶여
  있어 항목 05~07 몫).
- **남긴 것**:
  - 항목 05(웹)·06(PPTX 런 분할·hlinkClick·`a:br`)이 이 확정 스타일과 `Appearance.Link`를
    소비한다. `TextStyle.Italic`/`Underline`/`Link`의 렌더 매핑은 아직 없다 — 렌더러는
    모르는 태그에서 명시적으로 멈추므로 조용한 무시는 없다.
  - **열린 질문 — 링크 색의 표 결합이 관례 수준이다**: core의 `TokenColor("accent1")`
    델타와 pptx의 hlink fallback(accent1)이 같은 값이라는 것을 컴파일러가 강제하지
    않는다 (core↔render 임포트 방향 때문에 한 테스트로 못 묶는다). 항목 06이 hlinkClick을
    낼 때 픽스처/골든으로 두 산출물의 링크 색 일치를 고정해야 한다.
  - ADR 승격 대기 (항목 09): 링크 색 accent1 재사용 vs hlink 전용 토큰, weight 어휘
    2값 폐쇄.

## Iteration 02 · 05-web — 웹 렌더러: 서브셋 태그 + 리스트 + 측정 대상 계약
<!-- item: 02/05-web -->

- **한 것**:
  - `render/web/content.go` — `writeNode`가 `soml.KindOf` 카탈로그로 분기한다: 블록·`<li>`는
    `writeBlock`(구 writeText의 일반화 — `<text>`는 `<div>`, 서브셋 블록은 저자가 쓴 이름
    그대로 시맨틱 태그), `<ul>`/`<ol>`은 `writeList`(class `balpyo-list`, 외관 없음, 식별자
    있음), 인라인은 `writeInline`(식별자 없음, 앞뒤 공백 없음, `<br/>`→`<br>`, `<a>`는
    href를 HTML 이스케이프해 싣는다). `<pre>`는 인라인 style에 `white-space: pre`를 더하고,
    선두 개행이 남아 있으면 개행 하나를 덧대 HTML 파서의 "여는 태그 직후 개행 벗기기"에서
    내용을 지킨다. 리스트 배치 상수 export: `ListTextIndent`(36)·`ListMarkerHang`(36)·
    `ListItemGap`(0)·`ListBulletChar`("•") — 항목 07의 `marL`/`indent`/`buChar`가 import한다.
    `listCSS`(마커는 `list-style: none` + `li::before`, ol은 엔진 카운터 + `"." ` 접미)는
    덱에 리스트가 있을 때만 낸다(`hasLists`) — 골든 불변의 근거다 (빈 스테이지와 같은 태도).
  - `render/web/style.go` — `runStyleCSS` 신규: textStyleCSS + `font-style`(Italic) +
    `text-decoration`(Underline)을 **항상** 명시 (UA의 `em{font-style}`·`a{text-decoration}`
    이 개입할 자리를 없앤다). 낡은 spcPct 줄높이 주석을 spcPts(ADR 0005)로 고쳤다 (NOTES
    01/08·02/02의 이월).
  - `render/web/web.go` — 패키지 주석의 식별자 계약을 "블록 종류에만"으로 갱신,
    `baseCSS(stage, lists)`.
  - `render/geom/geom.go` — `collect`가 `KindInline`을 자손까지 건너뛴다 (웹 렌더러와 같은
    카탈로그 — Refs ↔ querySelectorAll 자리 맞대기의 대칭).
  - 테스트: `render/web/subset_test.go`(파서+core를 거친 원문 기반 — 태그 매핑·크기 위계,
    리스트 마크업·CSS 상수, 인라인 완전 명시 7선언, 식별자 부재, href 이스케이프, br,
    인라인 경계 공백 보존, pre 공백 보존·선두 개행 보호, 이스케이프, 미해석 인라인·Link
    없는 a 에러), `render/geom/geom_test.go`(Refs가 인라인을 건너뛰는 표),
    `render/geom/measure_browser_test.go`(`TestMeasureSubsetSlide` — 실측: 자리 맞대기 통과,
    측정 목록에 인라인 없음, li rect 문서 순서·겹침 없음·간격 0, 첫 li 텍스트가
    ul.x+ListTextIndent에서 시작).
- **설계 결정과 근거**:
  - **시맨틱 태그를 골랐다** (전부 `<div>`와 저울질, ADR 후보 — 항목 09): 접근성(제목
    위계·리스트 낭독)과 저자 CSS의 표적. 조건인 "UA 기본값 무의존"은 메트릭=인라인 style
    전부 명시 + 배치=엔진 CSS(margin 0)로 충족되므로 메트릭은 태그 선택과 무관하다.
    `<text>`는 대응 시맨틱 태그가 없는 중립 요소라 `<div>`로 남는다 — 역할별 태그로 바꾸면
    `<text role>` ≡ 블록 태그 등가가 산출물에서 두 갈래가 된다.
  - **마커 배치는 웹이 PPTX 관례를 따른다** (ADR 후보): OOXML은 마커를 `marL+indent`에
    왼쪽 정렬로 그리고 뷰어가 그리는 방식이라 바꿀 수 없다. CSS 기본 마커(outside)는
    오른쪽 정렬이라 어긋나지만, `::before`로 OOXML 모델을 정확히 표현할 수 있으므로
    맞추는 쪽은 웹이다. 마커-텍스트 "간격"은 상수가 아니라 (Hang − 글리프 폭)의 파생값 —
    OOXML도 간격을 직접 갖지 않으므로 간격을 상수로 두면 PPTX가 표현할 수 없다.
  - **값 36/36은 PowerPoint 1수준 기본**(marL=indent=342900 EMU) 그대로다 — 342900÷9525 =
    36px로 환산이 정수로 떨어지고, 익숙한 기본을 벗어날 이유가 없다.
  - **마커는 UA disc가 아니라 폰트 글리프 "•"**: PPTX `buChar`가 텍스트 폰트로 그리는
    글리프와 같은 곳에서 나와야 마커 모양이 갈라지지 않는다. ol은 엔진 카운터
    (`counter(balpyo-item) "."`)로 — UA list-item 카운터는 display에 묶여 있어 의존하지
    않는다. "1." 형식은 `buAutoNum@type="arabicPeriod"`와 짝 (항목 07).
  - **블록의 인라인 style은 그대로 6선언, 런만 8선언**: 서브셋 블록 태그에는 이탤릭·밑줄
    UA 규칙이 없고 지금 어휘에서 블록 스타일은 Italic/Underline을 가질 수 없으므로,
    textStyleCSS에 두 선언을 더하면 Iteration 1 골든만 이유 없이 움직인다. 블록이 Italic을
    가질 수 있게 되는 날 textStyleCSS가 함께 얻어야 한다 (runStyleCSS 주석에 명시).
  - **listCSS는 조건부다**: 리스트 없는 덱(00_empty, 01_text)의 산출물이 리스트 렌더가
    붙었다는 이유로 달라지면 골든 diff가 무엇이 움직였는지 말해 주지 못한다 — themeCSS가
    빈 테마에 층을 만들지 않는 것과 같은 태도.
  - **`<pre>` 선두 개행 보호**: 파서가 양 끝 개행 하나씩을 벗긴 뒤 남은 선두 개행은
    내용인데, HTML 파서도 `<pre>` 직후 개행 하나를 벗기므로 렌더러가 하나를 덧댄다 —
    덧대지 않으면 첫 빈 줄이 조용히 사라져 PPTX(원문 그대로)와 줄 수가 갈라진다.
  - **Link 없는 `<a>`·미해석 인라인은 렌더 에러**: core의 href 검증을 지나지 않은 값이
    산출물에 실릴 길을 막는다 (writeText의 미해석 외관 가드와 같은 태도).
- **검증**: RED 먼저 확인 (상수 부재로 web·geom 테스트 빌드 실패). 구현 후 `gofmt -l .`
  무출력, `go vet ./...`, `go test ./...` 로컬 전부 통과, `make test`(컨테이너, finch —
  LibreOffice·Chromium) 전부 통과. 브라우저 실측(`TestMeasureSubsetSlide`): li 5개 rect가
  문서 순서로 내려가고 같은 리스트 안 간격 0, 첫 li의 첫 줄 x = ul.x+36 확인. `git status`에
  testdata/골든 변화 없음 — 중간에 CSS 주석 한 줄을 고쳤다가 골든이 2줄 움직이는 것을
  픽스처 테스트가 잡아 되돌렸다 (골든 불변을 지키는 데 실제로 쓰인 회귀 장치).
- **남긴 것**:
  - 항목 06·07이 소비할 표면: `runStyleCSS`가 명시하는 런 서식 = PPTX `rPr@b/@i/@u`+색+
    폰트의 짝, `web.ListTextIndent`/`ListMarkerHang`/`ListItemGap`/`ListBulletChar` →
    `marL`/`indent`/(spcBef 0)/`buChar`. pre의 줄바꿈 없음(`white-space: pre`)의 짝은
    `bodyPr@wrap`(항목 06). 마커가 li의 확정 스타일을 상속하므로 PPTX 마커(buFont·색)도
    런 서식을 따라야 한다.
  - 지오메트리의 줄 박스(Lines)는 **직계 문자 노드**에서만 나온다 (measureScript) —
    인라인이 섞인 블록은 줄 일부만 잡힌다. 줄바꿈 임계값 판정(Slack)이 서브셋 픽스처로
    넓어지는 항목 08에서 이 한계가 문제가 되는지 봐야 한다.
  - `_02_text_subset` 픽스처의 `<span weight="700">`(003_inline.slide)은 core의 weight
    어휘(normal/bold)에서 에러다 — 항목 08이 픽스처를 활성화할 때 `bold`로 고쳐야 한다.
  - **열린 질문 — 항목 사이 간격 0의 시각 검증**: ListItemGap=0은 수직 리듬을 줄높이에
    맡긴 엔진 결정이다 (PRD가 값을 정하지 않는다). 리뷰 포인트(불릿 들여쓰기·마커 육안
    확인, TASKS I2)에서 답답해 보이면 항목 07 전에 상수 하나만 바꾸면 된다.
  - ADR 승격 대기 (항목 09): 측정 대상 = 블록 종류 카탈로그(09 지시의 3번), 시맨틱 태그
    선택, 마커의 PPTX 관례 채택 + 36px 상수.

## Iteration 02 · 06-pptx-runs — PPTX 텍스트 런 분할 + hlinkClick + a:br
<!-- item: 02/06-pptx-runs -->

- **한 것**:
  - `render/pptx/shape.go` — `textBodyXML`이 평탄화된 문자열 대신 **블록 요소를 받아
    자식을 순회**한다: 문자 데이터 잎마다 `a:r`(스타일은 잎을 감싼 가장 안쪽 요소의 확정
    `TextStyle`), `<br/>` → `a:br`, `<pre>`의 보존 개행 → `a:br`(빈 줄은 연속 a:br, 빈
    a:t는 내지 않는다) + `bodyPr@wrap="none"`. 인라인 런의 rPr에 `i="0/1"`·`u="none/sng"`
    (`italicValue`/`underlineValue`), `<a>` 잎에는 latin/ea **뒤에** `<a:hlinkClick r:id>`
    (CT_TextCharacterProperties의 자식 순서 — 어기면 LibreOffice가 복구를 묻는다).
    `slideLinks`가 순회 중 외부 URL에 rel id를 배정한다. `writeShapeNode`의 대상을
    `<text>` + `soml.KindBlock` 카탈로그로 넓혔다 (`writeTextShape` 이름·좌표 경로는 그대로,
    `<ul>`/`<ol>`은 기존 가드 에러 유지 — 항목 07).
  - `render/pptx/parts.go` — `shapesXML`이 돌려준 링크 목록을 `slideRelsTemplate`의
    `{{range .Links}}` 로 실었다: `Type=…/hyperlink`, `Target`은 XML 이스케이프,
    `TargetMode="External"`. r 네임스페이스는 슬라이드 루트에 이미 선언되어 있다 (확인).
  - `render/pptx/runs_test.go` 신규 — 런 분할(경계 공백이 앞 런 끝에 남음), 중첩
    `<strong><em>` 누적, span 토큰 색 schemeClr, code 런의 부모 크기 + mono 리터럴
    typeface, hlinkClick↔rels 짝·URL `&` 이스케이프, rel id 결정성(재사용·노트 공존·
    바이트 동일성), 웹·PPTX 링크 색의 표 일치(런 accent1 참조 = 테마 hlink 슬롯 값),
    `<br/>`·pre 개행의 a:br(문단 불변), 블록 태그 p:sp, 리스트 가드, 미해석 인라인·Link
    없는 `<a>` 에러, 서식 없는 런의 XML 불변.
  - `render/pptx/soffice_test.go` — 열림 테스트에 슬라이드 3 추가: h2+strong, span 색,
    같은 URL 링크 2회(관계 재사용 경로), br, em, code, pre.
- **설계 결정과 근거**:
  - **rel id 배정: rId1=레이아웃, rId2=노트(없어도 예약), rId3부터 외부 URL이 문서 순서**.
    노트 자리를 예약하지 않으면 노트 유무가 링크 번호를 밀어, 노트 한 줄을 더한 커밋의
    diff에 링크 관계 전부가 잡음으로 실린다. OPC 관계 id는 임의 문자열이라 빈 번호는
    유효하다 (notesSlidePart 번호 규칙과 같은 태도).
  - **같은 URL은 관계 하나를 재사용**한다 (매번 내는 대안을 버림): OPC 관계의 정체는
    (Type, Target, TargetMode)라 중복은 정보가 없고, PowerPoint도 재저장 시 하나로
    합치므로 매번 내면 재저장 diff에 잡음이 남는다. 테스트가 같은 URL 3회 → 관계 1개
    (rId3 재사용)를 고정한다.
  - **i/u 명시는 인라인 런만** (`runContext.inline` — 웹의 textStyleCSS 6선언/runStyleCSS
    8선언 분리와 같은 결정): 지금 어휘에서 블록 스타일은 Italic/Underline을 가질 수
    없으므로 블록 직속 런에 `i="0" u="none"`을 더하면 Iteration 1 골든만 이유 없이
    움직인다. 그 덕에 01_text 골든이 바이트 단위로 그대로다 (`TestRunPlainTextUnchanged`가
    XML 모양을 단위 테스트로도 고정). 블록이 Italic을 얻는 날 이 구분이 함께 사라져야
    한다 (runStyleCSS 주석과 같은 이월). hlinkClick 런은 core가 Underline=true를 확정하므로
    `u="sng"`·solidFill이 항상 나간다 — 뷰어의 자동 밑줄·자동 색에 기대지 않는다.
  - **인접 같은 스타일 잎은 합치지 않는다** (항목 지시의 확인 사항): 렉서가 연속 문자
    데이터를 다음 `<`까지 토큰 하나로 담으므로 잎 경계는 대개 요소이고, **주석이 문자
    데이터를 쪼갠 경우에만** 같은 스타일의 이웃 런이 생긴다 — rPr가 같은 이웃 런은
    뷰어에서 구별되지 않는 유효한 XML이라 합치는 코드는 값어치가 없다 (writeRuns 주석).
  - **`<br/>`·pre 개행은 문단을 쪼개지 않고 a:br**: 문단 사이 간격은 줄간격과 다른 축이라
    쪼개면 웹(같은 블록 안 줄바꿈)과 갈라질 수 있다. pre의 `wrap="none"`은 웹의
    `white-space: pre`(항목 05)의 짝이다. CRLF 원문의 `\r`은 a:t에 남기지 않는다.
  - **href의 &는 원문 그대로, rels에서만 XML 이스케이프**: SOML은 엔티티를 해석하지
    않으므로 (파서 계약) 저자가 쓴 `?a=1&b=2`가 웹 href와 PPTX Target에 같은 문자열로
    실린다.
  - **링크 색의 표 결합을 산출물로 고정** (02/04의 열린 질문에 대한 답):
    `TestRunHyperlinkColorMatchesThemeHlinkSlot`이 런의 `schemeClr val="accent1"`과 테마
    파트 `<a:hlink>`의 RGB(accent1 값)가 같은 표에서 나오는 것을 단정한다 — core↔render
    임포트 방향 때문에 컴파일러로 못 묶는 관례를 테스트가 잡는다.
- **검증**: RED 먼저 확인 (구현 전 13건 실패 — 전부 "매핑이 없다"). 구현 후 `gofmt -l .`
  무출력, `go vet ./...`, `go test ./...` 로컬 전부 통과, `make test`(컨테이너, finch)
  전부 통과. 컨테이너에서 `TestLibreOfficeOpensTextShapes -v -count=1` 단독 실행 PASS —
  hlinkClick·External 관계·i/u·wrap="none"이 든 슬라이드를 LibreOffice가 복구 경고 없이
  열었다 (OpenPPTX는 stderr의 복구 흔적을 실패로 본다). `git status`에 testdata/골든 변화
  없음 — 서식 없는 텍스트의 런 XML이 바이트 단위로 그대로다.
- **남긴 것**:
  - 항목 07(리스트)이 이 런 분할기(`writeRuns`)를 그대로 쓴다 — li 하나가 `a:p` 하나가
    되는 구조라 문단 분리가 필요하고, `textBodyXML`의 단일 `a:p` 가정을 그때 풀어야 한다.
  - **열린 질문 — 내부 앵커(`href="#/id"`)**: v1은 외부 URL만이고, 지금은 모든 href가
    `TargetMode="External"`로 나간다 — 내부 앵커도 문법상 유효한 열린 값이라(core의 href
    검증은 URL 문법을 보지 않는다) PPTX에서는 죽은 링크가 된다. 슬라이드 점프 액션
    (`ppaction://hlinksldjump` + 슬라이드 파트 관계)으로 나누려면 "무엇이 내부 앵커인가"의
    판별(`#/` 접두?)을 core와 웹 런타임(딥링크 해시 규칙)과 한 곳에서 정해야 한다 —
    렌더러가 각자 판별하면 같은 href가 타깃마다 다른 종류의 링크가 된다.
  - TASKS의 "인라인 태그 → 웹 그대로 / PPTX 런 분할" 체크박스를 [x]로 바꿨다 (웹은 항목
    05, PPTX 런은 이 항목). "테스트"·"리뷰" 박스는 리스트(항목 07)까지 묶여 있어 두지
    않았다.

## Iteration 02 · 07-pptx-lists — PPTX 리스트: 불릿 문단 + 들여쓰기
<!-- item: 02/07-pptx-lists -->

- **한 것**:
  - `render/pptx/shape.go` — `writeShapeNode`에 리스트 분기: `<ul>`/`<ol>` 하나 =
    `p:sp` **하나**(`writeListShape`), `<li>` 하나 = `a:p` 하나(`writeListItem`). 도형
    골격은 `writeSp`로 추출해 블록과 공유한다 (출력 바이트 동일 — 골든 불변 확인).
    문단 pPr: `marL="342900"`(=`EMU(web.ListTextIndent)`) + `indent="-342900"`
    (=`-EMU(web.ListMarkerHang)`, 행잉 인덴트) + li의 확정 정렬 + `lnSpc`/`spcPts`
    (블록과 같은 절대 줄간격 규칙). 마커: ul은 `buChar`(=`web.ListBulletChar`), ol은
    `buAutoNum type="arabicPeriod"`(웹의 `counter() "."` 짝), `buFont`는 li의 확정
    폰트 토큰(`typefaceValues`의 latin — body는 `+mn-lt` 테마 참조). `spcBef`/`spcAft`는
    내지 않는다 (`web.ListItemGap=0` — 수직 리듬은 줄높이 소유). li 안의 인라인·`<br/>`은
    항목 06의 `writeRuns`를 그대로 부른다 — 리스트용 순회를 만들지 않았다. 블록 검증은
    `checkTextStyle`로, `typefaces`는 값(`typefaceValues`)과 요소 조립로 나눴다.
  - `render/pptx/lists_test.go` 신규 — li 3개 → 한 sp 안 `a:p` 3개, marL/indent가
    웹 상수 × `EMUPerPixel`과 일치·행잉 인덴트 부호, spcPts 2700(body 24px×1.5),
    buFont `+mn-lt`, spcBef/spcAft 부재 + `ListItemGap==0` 단정(상수가 움직이는 날
    PPTX 대응을 요구), ul↔ol이 마커 치환만으로 동일 XML, li 안 strong/a 런·경계 공백·
    rel id 문서 순서(블록→리스트 경유 rId3/rId4 + 같은 URL 재사용), `<br/>` = 같은 문단
    안 `a:br`, 빈 li = 런 없는 문단, 빈 ul = `<a:p/>` 하나 든 도형 + 도형 id 연속,
    바이트 동일성. `runs_test.go`의 가드 테스트(`TestRenderListStillFails`)는 제거.
  - `render/pptx/soffice_test.go` — 슬라이드 4 추가: ul(인라인 strong/a·br·빈 li) + ol.
- **설계 결정과 근거**:
  - **안전 여유는 `TextBoxRect`를 start 정렬로 그대로 적용** (항목 지시의 정책 결정):
    li는 align 속성을 받지 않아 (파서 allowlist) 확정 정렬이 언제나 start이므로, 닻 내린
    왼쪽 변(=마커 위치)을 움직이지 않는 방향은 오른쪽 확장뿐이다. 4%가 마커 자리(36px)를
    포함한 컨테이너 폭에 걸려 절대 여유가 텍스트만일 때보다 약간 크지만, 그 오차는 줄
    늘어남을 막는 안전한 방향이다 — 판정은 항목 08의 픽셀 비교에 넘긴다.
  - **buFont는 li의 확정 폰트 토큰(`+mn-lt` 참조)** — 명시하지 않으면 뷰어 기본 폰트가
    마커를 그려 크기·모양이 흔들리고, 웹 마커(`li::before`)는 li의 폰트를 상속하므로
    같은 폰트여야 짝이 맞는다. 테마 참조로 남겨 PowerPoint 테마 폰트 편집을 따라간다
    (rPr latin과 같은 값 — `typefaceValues` 한 곳). 색·크기는 내지 않는다: OOXML 기본
    (buClrTx/buSzTx)이 이미 "텍스트를 따른다"로 `::before` 상속과 같은 뜻이다.
  - **빈 `<li>` = 런 없는 문단, `<li>` 없는 `<ul>`(파서가 안 막는다) = 도형은 낸다** —
    빈 텍스트 정책과 같은 태도 (자리·순서·도형 id 보존). txBody는 `p+`를 요구하므로
    빈 리스트는 서식 없는 `<a:p/>` 하나가 스키마를 채운다 (li가 없으니 실을 스타일도
    마커도 없다).
  - **`<li>` 안 `<br/>`은 같은 문단 안 `a:br`** — 문단(=항목)을 쪼개면 마커가 하나 더
    생긴다. 웹의 `<br>`(li 블록 안 줄바꿈)과 같은 뜻의 "마커 없는 이어짐 줄"이다.
  - pPr 자식 순서는 스키마(CT_TextParagraphProperties) 그대로: lnSpc → buFont →
    buChar/buAutoNum. 어기면 LibreOffice가 복구를 묻는다 (hlinkClick 순서와 같은 종류).
- **검증**: RED 먼저 확인 (lists_test 7건 전부 "매핑이 없다"로 실패). 구현 후
  `gofmt -l .` 무출력, `go vet ./...`, `go test ./...` 로컬 전부 통과, `make test`
  (컨테이너, finch) 전부 통과. 컨테이너에서 `TestLibreOfficeOpensTextShapes -v -count=1`
  단독 PASS — buChar/buAutoNum·행잉 인덴트·`buFont typeface="+mn-lt"`가 든 리스트
  슬라이드를 LibreOffice가 복구 경고 없이 열었다. `git status`에 testdata/골든 변화 없음 —
  `writeSp` 추출이 기존 도형 XML을 바이트 단위로 보존했다.
- **남긴 것**:
  - **픽셀 비교(항목 08)가 판정할 것**: 리스트 안전 여유 정책(start 확장, 마커 자리 포함
    4%), 마커 글리프의 실제 x 위치(웹 `left: ListTextIndent-ListMarkerHang` = 0 대비
    PPTX의 marL+indent = 0 — 뷰어가 마커를 왼쪽 정렬로 그린다는 가정), 빈 li의 마커
    유무(웹 `::before`는 빈 li에도 마커를 그리지만 PPTX 뷰어는 빈 문단에 마커를 안
    그릴 수 있다 — 구조만 이 항목의 책임이다).
  - `lvl` 매핑(중첩 리스트)은 파서 가드를 풀 때 함께 온다 (항목 01의 결정 그대로).
  - TASKS의 Iteration 2 "테스트" 박스(등가성·런·리스트)를 [x]로 바꿨다 — 등가성은 02/02,
    런은 02/06, 리스트는 이 항목이 마감했다. "리뷰"(양 타깃 육안 대조)·"문서" 박스는
    항목 08·09 몫으로 남긴다.

## Iteration 02 · 08-fixtures — 02_text_subset 활성화 + 골든 + 픽셀 비교
<!-- item: 02/08-fixtures -->

- **한 것**:
  - 픽스처 활성화 전 정비: `003_inline.slide`의 `<span weight="700">` → `weight="bold"`
    (02/05가 남긴 이월 — core의 weight 어휘는 normal/bold다). **경고 슬라이드
    `006_variant.slide` 신규** (항목 지시의 권고대로): 모르는 variant(`hero`) + 알려진
    variant(kicker·lead) 대조. 골든 `diagnostics.txt`는 "경고가 있을 때만" 존재하는
    계약이라 이 슬라이드가 그 경로(경고 문구·위치·테마 어휘 목록)의 유일한 회귀 테스트다.
  - `testdata/fixtures/_02_text_subset` → `02_text_subset` 개명 (git mv), 골든 신규
    (index.html + diagnostics.txt + PPTX 파트 27개 — 전 파트를 훑었다).
    `soml/subset_test.go`의 픽스처 경로·주석 갱신. **00_empty·01_text 골든은 바이트
    단위로 무변경** — diff의 모든 줄이 새 픽스처 것이다.
  - **환경 결함 수정 — 테스트 이미지에 Noto Sans Mono가 없었다**: 테마의 mono 토큰은
    "Noto Sans Mono"인데 `fonts-noto-core`에 그 폰트가 없어 fc-match가 Noto Sans로
    폴백하고, Chromium과 LibreOffice가 **서로 다른** 대체 폰트를 골라 code 런의 글줄
    폭이 +5%대로 갈라졌다 (첫 픽셀 비교가 잡았다 — 이 게이트가 만들어진 목적 그대로).
    `docker/test.Dockerfile`에 `fonts-noto-mono` 설치 + `/etc/balpyo-versions.txt`에
    `BALPYO_FONT_MONO` 기록, `docker/versions.env`에 핀 추가 (대조 테스트가 폴백 재발을
    골든보다 먼저 잡는다).
  - **하네스 이월 처리 (NOTES 01/07의 "남긴 것")**: `textAligns` →
    `shapeAligns`(internal/testkit/text_fixture_test.go) — geom.Refs와 같은 재귀
    (인라인은 자손까지 건너뜀)로 바꾸고, PPTX 도형이 되지 않는 `<li>`는 nil로 표시해
    좌표 대조가 건너뛴다 (리스트는 컨테이너가 도형 하나 — pptx.writeListShape).
    컨테이너는 start 정렬 고정.
  - **픽스처 수준 검증을 두 텍스트 픽스처(01_text·02_text_subset)로 확장**: 골든 파트
    목록 전수(diagnostics.txt·notesSlide5 포함), 슬라이드 파트 srgbClr 부재(인라인 런
    색까지), xfrm == 측정값(+여유) EMU, 잉크 띠 대조. `textFixtureNames` 슬라이스에 새
    텍스트 픽스처를 더하면 같은 계약을 받는다.
  - **허용 오차 조정 2건 (측정 수치·근거 포함)**:
    - 픽셀 비교: `02_text_subset`에 한해 MaxRatio 2.5% → **3.5%**
      (testkit/suite.go `fixtureDiffOptions`). 측정 최대 3.04%(슬라이드 1 — 제목 위계).
      구성은 LibreOffice 첫 기준선 +2~5px의 글리프 경계 이중 계수이고, 큰 글리프
      (h1 48px + h2/h3 + 본문 + 인용)의 잉크 면적이 01_text 표지보다 많다. 같은
      슬라이드의 잉크 띠 검사(줄 수 0차·세로 중심 ±6px·폭)는 전부 초록 — 구조가 아니라
      잔차다. 여유 폭은 01_text와 같은 태도 (2.1% 측정 → 2.5% 허용).
    - 잉크 폭 판정: (a) 절대 하한 `loLineWidthMinPx = 2px` — AA는 줄 길이와 무관하게
      ±1px을 만들므로 아주 짧은 띠(pre의 `}` 한 글자, 잉크 6px)에서 1px이 16.7%로
      계산되어 방법 문제로 죽는다. 긴 줄에서는 비율 허용값이 항상 2px보다 커 게이트
      약화가 아니다. (b) `02_text_subset`에 한해 비율 4% → **5.5%** — 측정 최대
      +5.0%(003_inline의 code+링크 문단). LibreOffice 스크립트 간격은 **경계 수**에
      비례하는데(라틴 문장부호 앞·mono 런 전환 포함 — 리뷰 렌더에서 육안 확인) 비율은
      줄 길이로 나눈 값이라, 짧은 줄(545px)에 경계가 몰리면 01_text의 긴 본문(1125px,
      +3.3%)보다 비율이 커진다. 이 픽스처의 최소 여유 비율은 40.5%라 재접힘 위험 없음.
  - `make review` → `review/02_text_subset/` 갤러리 (6장 × web/pptx/diff +
    text-subset-deck.pptx). TASKS Iteration 2 "구현" 부모 박스 [x] (자식 3개는 02/02·
    05·06·07이 이미 [x]).

- **설계 결정과 근거**:
  - **경고 슬라이드는 알려진 variant와 한 슬라이드에 뒀다**: 폴백(hero = 기본 외관)이
    kicker·lead와 나란히 보여 갤러리에서 "폴백이 무엇인가"가 시각으로 남는다. 부수
    효과로 variant 캐스케이드의 렌더 결과(웹 CSS + PPTX rPr)가 처음으로 골든에 들어갔다.
  - **폰트 설치는 구현 변경이 아니라 환경 수정으로 판단했다** (항목의 "범위 밖" 검토):
    렌더러 코드는 처음부터 "Noto Sans Mono"를 냈고 두 산출물 모두 그 이름을 실었다 —
    틀린 것은 이미지의 폰트 집합이다. 골든과 같은 커밋에 담아 diff가 설명된다
    (mono가 낀 슬라이드 2·3의 xfrm이 폰트 교체로 움직였다).
  - **오차 조정은 전부 픽스처 한정 + 구조 검사 초록 전제**: 기본값(2.5%/4%)은 그대로
    두고 조정 지점(`fixtureDiffOptions`·`loLineWidthMaxRatios`)에 측정값과 근거를
    주석으로 남겼다 — 다음 픽스처가 기본값으로 시작해 다시 재게 한다.

- **검증**: `gofmt -l .` 무출력, `go vet ./...`, `make test`(컨테이너, finch —
  LibreOffice·Chromium 포함) 전부 초록. `make golden` 재실행 시 무변경(결정성).
  `make review` PASS + 갤러리 6장 육안 확인 — 리스트 마커·들여쓰기·굵기·code·링크 색이
  양 타깃에서 같고, diff는 기준선 이동 잔상뿐. 잉크 띠 측정: 02_text_subset 줄 수 차 0
  (전 슬라이드), 세로 중심 최대 +6px(경계값 — s5.2 lead 32px), 잉크 폭 최대 +5.0%
  (s2.2), 최소 여유 비율 40.5%. 픽셀 diff: 3.04 / 1.78 / 2.18 / 0.79 / 1.36 / 2.24%.

- **남긴 것**:
  - **사람 확인 남음 (TASKS I2 리뷰 박스 — 자동화하지 않는다)**: macOS에서
    `review/02_text_subset/text-subset-deck.pptx`를 Keynote/PowerPoint로 직접 열어
    (a) 리스트가 진짜 불릿 문단인지 (Tab/Shift+Tab이 동작하는 편집 가능한 리스트),
    (b) 링크가 클릭되는지, (c) 인라인 서식(굵기·이탤릭·밑줄·mono·색)이 런으로 살아
    있는지, (d) 불릿 들여쓰기·마커가 갤러리와 같아 보이는지 확인한다. 리뷰 박스는 그
    뒤에만 닫는다. 참고 관찰: 스크립트 간격(라틴 문장부호·mono 경계 포함)은 LibreOffice
    렌더러의 동작이라 PowerPoint/Keynote에서는 웹에 더 가깝게 나올 가능성이 높다.
  - **호스트 `go test`는 02_text_subset 골든 비교에서 빨갛다** (컨테이너는 초록 —
    게이트는 컨테이너다): macOS에 Noto Sans Mono가 없어 code가 낀 블록의 줄 박스
    높이가 1px 다르게 측정되고(37px vs 36px), 박스 높이 = 줄 수 × 줄높이라 xfrm의
    cy가 움직인다. 01/07이 "호스트 폰트가 줄 수를 다르게 접으면 호스트만 빨개질 수
    있다"고 남긴 그 경우의 실물이다 — mono 픽스처가 생기며 실제가 됐다. 호스트에서
    Noto Sans Mono를 설치하면 사라진다.
  - `InkBands`의 단색 배경 가정은 텍스트뿐인 이 픽스처에서 여전히 유효했다 — 깨짐 없음
    (도형 iteration에서 다시 본다).
  - 지오메트리 줄 박스가 직계 문자 노드에서만 나오는 한계(02/05의 이월)는 이 픽스처에서
    문제가 되지 않았다: 인라인이 낀 모든 줄에 직계 문자가 하나 이상 있어 줄 수가 다
    잡혔고, 기대 띠의 부분 폭은 판정에 쓰지 않는다 (expectedLineBands 주석에 명시).
    **인라인만으로 이루어진 줄**(예: `<p><strong>…긴 굵은 문장…</strong></p>`이 접힌
    둘째 줄)이 생기면 그 줄이 기대 띠에서 빠져 줄 수 대조가 방법 문제로 죽는다 — 그런
    픽스처를 만들 때 measureScript를 확장해야 한다.
  - 항목 07이 픽셀 비교에 넘긴 판정들: 리스트 안전 여유(start 확장, 마커 자리 포함
    4%)와 마커 x 위치는 허용 오차 안 (슬라이드 2 diff 1.78%, 잉크 띠 초록). 빈 li의
    마커 유무는 이 픽스처에 빈 li가 없어 판정하지 못했다 — 여전히 열린 관찰.

## Iteration 02 · 09-docs — Iteration 2 레퍼런스·캐스케이드 문서와 ADR 4건
<!-- item: 02/09-docs -->

- **한 것**:
  - `docs/reference/soml/text-subset.md` 신규 — 통일 형식(속성/태그 표 → 예제 → 웹/PPTX
    렌더 결과 → 에러 조건). HTML 서브셋 태그 표(종류·허용 속성·배치 규칙과 각각의 왜),
    역할 등가 표(h1→title, h3=28, blockquote=muted, caption 비대칭, `<li>`=body),
    공백 규칙(인라인 경계·pre 보존), `<br/>`·`<a href>` 사용법. 예제는 전부
    `testdata/fixtures/02_text_subset`의 실제 슬라이드, 에러 표의 진단 문구는
    `soml/parse.go`·`core/resolve.go`·`core/inline.go`의 실제 문자열이다. `reveal`은
    쓰지 않았다 (어휘에 없다 — 미구현 동작 금지 규칙).
  - `docs/reference/appearance.md` 갱신 — "해석 순서" 잠정 절을 **캐스케이드 절 전면**으로
    교체 (4단 순서, 부분 패치 병합, 병합이 core에서 한 번·타깃 무관이라는 계약, variant
    하위 절: 테마 소유 어휘·모르는 variant의 경고+폴백+`--strict` 승격 — 경고 예시는
    006_variant 골든 diagnostics.txt의 실제 문구). 텍스트 역할 5종 표(+대응 블록 태그 열),
    매핑 표에 `variant` 행, **인라인 서식 표**(strong/em/code/a/span/br의 CSS ↔
    rPr@b/@i/@u·hlinkClick·a:br 대응), **리스트 표**(marL/indent/buChar/buAutoNum/buFont —
    웹 CSS 상수와 한 곳에서 나온다는 계약 명문화).
  - `docs/reference/cli.md` — `--strict`는 build 옵션 표에 이미 있었고, 진단 출력 절에
    variant 경고(첫 사용자 가시 경고)의 실제 출력 예시와 `--strict` 승격을 이었다.
  - **낡아진 기존 문서 수정** (문서와 코드가 갈라지면 문서가 틀린 것): `text.md`
    (콘텐츠 모델이 "문자 데이터만" → 인라인 태그 포함, `variant` 속성 행, 바뀐 진단
    문구 2건), `slide.md`(자식 어휘 문장과 에러 표 행 — 실제 메시지를 `go run`으로 출력해
    확인하고 옮겼다), `reference/index.md`의 "현재 상태".
  - ADR 4건: `0007-block-tag-role-ownership.md`(등가 표는 언어, 기본값은 테마 + h1→title),
    `0008-run-style-inherit-copy.md`(상속-복사 + size 상속 차단 근거),
    `0009-measured-kinds.md`(측정 대상 = 블록 종류, 카탈로그 4층 공유),
    `0010-link-color-accent1.md`(hlink 전용 토큰 대신 accent1 재사용, I10 재검토).
    `docs/decisions/index.md`·TASKS.md 결정 로그 갱신 — 결정 로그에 빠져 있던 0006 행도
    함께 채웠다 (색인 완결성). `mkdocs.yml` nav에 5페이지 추가.
  - TASKS.md Iteration 2 문서 체크박스 3개 `[x]`. 리뷰 박스는 사람 확인 대기라 열어 뒀다.

- **설계 결정과 근거**:
  - **ADR은 4건으로 판단했다.** 항목 지시의 3건에 더해 링크 색 accent1 재사용을 0010으로
    승격 — 02/04와 02/06 두 항목이 승격 대기로 표시했고, core↔render를 가로지르는 관례를
    산출물 테스트가 고정하는 횡단 결정이며 I10 재검토 예정이 명시적이라 색인에 있어야
    한다. 나머지 후보(시맨틱 태그 선택, 마커의 PPTX 관례 + 36px 상수, weight 2값 폐쇄)는
    01/08과 같은 태도로 **국소 매핑 선택**이라 올리지 않았다 — 각 근거는 코드 주석과
    레퍼런스 표에 이미 있고, 마커·상수 계약은 appearance.md 리스트 표가 본문으로 담는다.
  - 문서의 진단 문구는 전부 구현의 실제 문자열 사본이다 — slide.md의 자식 어휘 목록처럼
    확신이 없는 것은 파서를 직접 돌려(`go run` 일회용 스니펫) 출력을 확인하고 옮겼다.

- **검증**: `make docs`(mkdocs --strict) 통과. 코드 변경이 없는 항목이지만 커밋 전
  게이트를 다시 돌렸다: `gofmt -l .` 무출력, `go vet ./...`, `make test`(컨테이너, finch)
  전부 초록. 호스트 `go test ./...`는 02_text_subset 골든 비교에서 빨갛다 — 02/08이
  기록한 macOS의 Noto Sans Mono 부재 그 경우다 (mono가 낀 두 슬라이드의 cy가 1px(9525
  EMU) 차이. 게이트는 컨테이너).

- **남긴 것**:
  - **항목 07·08이 커밋을 남기지 않은 채 끝나 있었다** (NOTES 기록과 검증은 완료, 작업
    트리만 미커밋). 항목별 한 커밋 흐름을 지키기 위해 두 항목의 변경을 NOTES 기록과
    대조해 파일 단위로 나눠 **항목별 커밋 2건으로 대신 커밋**하고, 그 위에 이 항목의
    문서 커밋을 얹었다. 커밋 본문에 대리 커밋임을 밝혔다.
  - Iteration 2의 남은 칸은 리뷰 박스 하나다 — 02/08의 "남긴 것"에 있는 (a)~(d) 확인
    목록을 사람이 Keynote/PowerPoint로 직접 연다. 끝나지 않은 것을 [x]로 만들지 않았다.
  - `attrExample`의 "kicker"(값 없는 variant 에러 예시가 테마 사유 어휘를 인용)는 02/02의
    열린 질문 그대로 Iteration 10 몫이다. 0010의 재검토(테마가 hlink 토큰을 명시하는
    경우)도 같은 시점이다.
