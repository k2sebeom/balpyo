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
