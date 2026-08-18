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
