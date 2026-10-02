# TASKS — 리팩터: 픽셀 동일성에서 컴포넌트·레이아웃 동일성으로 (2026-10-02)

> **계약을 바꾼다.** 지금까지 balpyo는 "웹과 PPTX가 픽셀 단위로 같다"를 목표로 삼았다. 그
> 목표는 글자를 서로 다른 두 엔진(Chromium · PowerPoint/LibreOffice)이 배치하는 한 닿지
> 않는다 — 줄바꿈 규칙(E1), 스크립트 경계 간격(P1.5), 폰트 해석(필드 테스트 4), 대각
> 그라디언트(ADR 0034)가 차례로 그 사실을 보였다. 게다가 픽셀 게이트가 비교한 상대는
> PowerPoint가 아니라 대리 렌더러 LibreOffice였다.
>
> **새 계약: 같은 컴포넌트가 같은 자리에 같은 외관으로 나온다.**
>
> - **컴포넌트**: SOML 요소 하나는 두 타깃에서 같은 종류의 객체다 — 텍스트 박스 · 프리셋
>   도형 · 그림 · 표 · 플레이스홀더. 문서 순서(= 칠 순서 · 읽기 순서)도 같다.
> - **레이아웃**: PPTX 객체의 `xfrm`은 브라우저가 잰 rect다 (EMU 반올림 안에서).
> - **외관**: 해석된 값이 같다 — 색은 토큰 이름(`schemeClr`), 글꼴은 테마 참조, 크기·굵기·정렬.
> - **글자 내용**은 런 단위로 같다. **줄이 어디서 접히는지는 계약 밖이다** — 각 엔진의 몫이다.
>
> 지키는 것은 그대로다: 저자는 좌표를 쓰지 않는다(지오메트리 패스), PPTX는 편집 가능한
> 네이티브 객체다, 테마는 디자인 시스템이다, 진단은 한 곳에서 나온다.
>
> 숫자로 본 출발점: 테스트 코드 34.6k줄 > 제품 코드 26.4k줄 · 골든 파일 473개 ·
> `internal/testkit/fixture_level_test.go` 한 파일 2,638줄 · 테스트에 Docker 이미지
> (Chromium + LibreOffice + poppler + 고정 폰트 + 힌팅 끔) 필요.

---

## 진행 규칙 (가볍게)

- 항목마다 **테스트 → 구현 → 문서**. 테스트가 먼저 바뀌는 항목(R1·R2)은 새 판정이 **옛 판정을
  대체하는 것**을 확인한 다음 옛 것을 지운다.
- ADR은 **계약을 바꾸는 결정에만** 쓴다. 이 리팩터 전체에 필요한 것은 R0의 하나다.
- 지우는 커밋은 지우는 것만 담는다 (`refactor:`/`test:`) — 되돌리기 쉽게.
- 끝나면 `go test ./...` 하나가 진짜 게이트다 (R4).

---

## R0. 계약을 문서로 바꾼다 — **먼저 한다**

- [x] **ADR 0064 — 컴포넌트·레이아웃 동일성**: 위의 네 줄을 결정으로 적는다. 대체하는 것을
      명시한다: PRD §13의 결정 1(autofit 배제 · 줄 수 차이 0 · 픽셀 임계값), §14의 "웹 ↔ PPTX
      시각적 일치 — 픽셀 비교" 지표, ADR 0005의 여유 근거(줄 수를 맞추기 위한 여유). 옛 ADR은
      고치지 않는다.
- [x] PRD §13·§14를 새 계약으로 바꾼다 (PRD는 ADR이 아니라 살아 있는 문서다).
- [x] `AGENTS.md`의 Testing 절과 불변식 목록을 새 계약에 맞춘다.

## R1. 동일성 판정 하나: 레이아웃 스냅샷

픽스처마다 **두 타깃에서 각각 뽑은 컴포넌트 트리가 같다**는 판정 하나로 픽스처 수준 검증을
대체한다.

- [ ] `internal/testkit`에 트리 추출기 둘 — 웹 쪽은 덱 모델 + 지오메트리 side table에서,
      PPTX 쪽은 산출된 슬라이드 XML(`spTree`)을 다시 읽어서. 노드 한 줄:
      `종류 · 요소 ID · rect · 외관 요약(색 토큰 · 글꼴 참조 · 크기 · 정렬) · 플레이스홀더 · 텍스트 런`.
- [ ] 판정 둘: (a) **웹 트리 == PPTX 트리** — 같은 실행 안에서 비교하므로 환경과 무관하다.
      (b) 트리를 `golden/layout.txt`로 남긴다 — 단, **rect는 빼고** 구조·외관·순서만 담는다
      (rect는 폰트에 따라 움직이므로 골든에 넣으면 다시 환경 고정이 필요해진다).
- [ ] 옛 판정이 지키던 것을 새 판정이 덮는지 대조한다:
      `TestFixtureCoordinatesMatchGeometry`, `…SlideColorsKeepTokenReferences`,
      `…ShapeCountExcludesNonShapeElements`, `TestStackFixtureZOrderIsDocumentOrder`,
      `TestTableFixture*`, `TestChromeFixture*`. 덮이면 지운다(R3).

## R2. 골든을 줄인다: 산출물 스냅샷 → 구조 스냅샷

`golden/index.html` 전체와 `golden/pptx/**`(파트별 XML) 스냅샷은 CSS 한 줄만 바꿔도 수십 파일이
움직이고, 리뷰어가 그 diff를 읽지 않는다.

- [ ] 픽스처 골든을 `layout.txt`(R1) + `diagnostics.txt` 둘로 줄인다. 웹 HTML · PPTX 파트 ·
      복사된 에셋 골든을 지운다.
- [ ] PPTX 패키지 구조(파트 목록 · content types · rels)는 한 테스트가 픽스처 전체에 대해
      **불변식으로** 단정한다 (스냅샷이 아니라): 모든 rel 대상이 존재한다 · 미디어 파트가
      에셋 표 순서다 · zip 타임스탬프 1980-01-01 · 같은 입력 두 번이 같은 바이트다.
- [ ] `TestStaleGoldensAreDetected` 계열은 새 골든 두 종에 맞춰 남긴다.

## R3. 픽셀 · 잉크 · LibreOffice 장치를 걷어낸다

- [ ] 지운다: `internal/testkit`의 `pixel.go` · `ink.go`(+ 테스트) · `script_spacing_test.go`,
      `fixture_level_test.go`의 픽셀/잉크/줄 수 판정
      (`TestFixtureLineBreaksMatchLibreOffice` · `…PresetGeometryPerShape` · `…InShapeTextLineCounts` ·
      `…InShapeTextRect` · `TestImageFixturePictureRegions` · `TestStackFixtureOverlapRegions` 등)과
      완화표 `inShapeLineDivergences` · `loLineWidthMaxRatios` · 픽스처별 `DiffOptions`.
- [ ] `fixture_level_test.go`에서 R1이 덮지 못하는 판정이 남으면 해당 패키지의 단위 테스트로
      옮긴다. 이 파일은 없어지는 것이 목표다.
- [ ] `render/pptx/soffice_test.go`: "LibreOffice가 연다" 하나만 **옵트인 스모크**로 남긴다
      (`soffice`가 없으면 건너뛰고, `make smoke`로만 돈다). 깨진 PPTX를 잡는 값은 진짜다.

## R4. 테스트 환경: 호스트에서 `go test ./...`

- [ ] 게이트를 호스트로 옮긴다: `make test` = `go test ./...`. Chromium이 필요한 테스트
      (지오메트리 패스)는 Chromium이 없으면 **명확한 메시지로 건너뛴다** — 빌드 자체가 Chromium을
      쓰므로 개발자는 이미 갖고 있다.
- [ ] 동일성 판정(R1-a)이 환경과 무관한지 확인한다 — 다른 폰트가 깔린 호스트에서도 초록이어야 한다.
- [ ] Docker 이미지는 CI 편의로만 남기고 줄인다: LibreOffice · poppler · `fonts.conf`(힌팅 고정) ·
      `versions.env` 버전 핀과 `TestImageVersionsMatchPins`를 지운다. `make test-host`는
      `make test`와 같아지므로 지운다.
- [ ] CI(`ci.yml`)의 "실패 시 리뷰 갤러리" 단계를 지운다.

## R5. 제품 코드에서 픽셀 시절의 타협을 푼다

계약이 바뀌어 **필요 없어진** 것만 푼다. 결과는 R1의 동일성 판정이 지킨다.

- [ ] PPTX 텍스트 몸체 `noAutofit` → `normAutofit` (`render/pptx/shape.go`의 세 자리). 글자가
      잰 박스를 넘으면 PowerPoint가 줄여 맞춘다 — 편집 가능한 파일에서 저자가 글을 고친 뒤에도
      박스 안에 머무는 쪽이 옳다. 크롬 띠와 `<pre>`(wrap="none")는 판단해서 그대로 둘 수 있다.
- [ ] `TextBoxRect`의 4% 여유(`textBoxSlackRatio`): 근거가 "LibreOffice 줄 수를 맞춘다"였다.
      autofit 아래에서 필요한지 다시 보고, 필요 없으면 지운다 (잰 rect == `xfrm`이 더 단순한 계약이다).
- [ ] 웹 `text-autospace: no-autospace` 고정(`render/web/content.go`)은 LibreOffice의 스크립트
      간격에 맞춘 것이다 — 지우고 브라우저 기본으로 돌린다.
- [ ] 한국어 줄바꿈: 웹 스테이지에 `word-break: keep-all` + `overflow-wrap: anywhere` (옛 E1).
      한국어 관행이 옳은 쪽이라 하는 것이지 PPTX와 맞추려는 것이 아니다.

## R6. `preview`: 나란히 보기만

- [ ] 갤러리는 웹 · PPTX를 나란히 놓는 사람용 도구로 남는다. **diff 이미지와 diff 비율을 지운다**
      (`internal/preview/diff.go` — 비율은 실제 불일치를 과장한다는 것이 3회차에서 이미 확인됐다).
- [ ] PPTX 래스터는 `soffice`가 있을 때만 — 없으면 웹만으로 갤러리를 만든다(지금
      `TestRunWithWebOnlyNeedsNoLibreOffice`의 동작을 기본으로).
- [ ] `make review`는 `balpyo preview`를 픽스처에 돌리는 얇은 입구가 된다 (테스트 스위트를
      `BALPYO_REVIEW=1`로 다시 돌리지 않는다).

## R7. 단위 테스트를 다듬는다

- [ ] 패키지별로 한 번씩 훑는다 (`soml` · `core` · `render/web` · `render/pptx` · `render/geom`):
      R1이 이미 덮는 단정 · 지운 상수(여유 비율, 스크립트 간격)를 못 박은 테스트 · 같은 것을 다른
      입력으로 반복하는 테스트를 지우거나 테이블 테스트로 합친다.
- [ ] 목표선: **테스트 코드 줄 수 < 제품 코드 줄 수**, `go test ./...`가 호스트에서 30초 안.
- [ ] `core/docs_test.go`(문서의 닫힌 목록 대조)는 남긴다 — 값이 크고 비용이 작다.

## R8. 문서를 새 계약에 맞춘다

- [ ] `docs/internals/testing.md`를 다시 쓴다: 동일성 판정 · 골든 두 종 · 호스트 실행 · 옵트인 스모크.
- [ ] `docs/guides/known-differences.md`를 "계약 밖의 차이"로 바꾼다 — 줄바꿈 위치 · 자간 ·
      안티에일리어싱은 결함이 아니라 엔진의 몫이라는 한 절.
- [ ] `docs/internals/geometry-pass.md`에서 픽셀 비교 파이프라인 그림을 지운다.
- [ ] `testdata/fixtures/README.md`의 골든 트리를 새 모양으로.
- [ ] `make docs`가 `--strict`로 통과.

---

## 완료 판정

- [ ] `go test ./...`가 호스트에서 초록이고, 그것이 CI의 게이트다
- [ ] 픽스처마다 "웹 트리 == PPTX 트리" 판정이 돌고, 골든은 `layout.txt` · `diagnostics.txt`뿐이다
- [ ] 저장소에 픽셀 허용 오차 상수가 하나도 없다
- [ ] `examples/*` 셋이 `check --check-layout --strict`로 경고 0건 (회귀 게이트는 그대로)
- [ ] 테스트 코드 < 제품 코드

---

## 리팩터 이후 (보류 — 순서 미정)

계약과 무관하게 여전히 유효한 것들. 리팩터가 끝난 뒤 다시 정렬한다.

- 3회차 사용성: `preview --out`이 갤러리와 같을 때 자기 PPTX를 지운다(옛 E2) · 알 수 없는 태그
  진단을 부모의 자식 목록으로 좁히고 오타 후보를 댄다 · 진단을 파일→줄→열로 정렬한다(E3–E5) ·
  사고 하나에 경고 하나(E6) · `quote`/`code`/`title`/`stat` 레이아웃 기본값(E7–E8) · ID 충돌 진단의
  `line:col`과 고칠 길(E9).
- 필드 테스트(`workspace/FIELD-TEST.md`): 테마 배경 채널 + 테마 소유 자산 · 크롬 로고 · 그라디언트
  토큰 · 측정 브라우저가 테마 글꼴을 못 찾을 때 경고.
