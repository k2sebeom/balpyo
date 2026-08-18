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
