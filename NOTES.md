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
