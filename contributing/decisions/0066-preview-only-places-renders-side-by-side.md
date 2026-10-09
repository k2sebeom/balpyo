# 0066 — `balpyo preview`는 두 렌더를 나란히 놓기만 한다 (diff · 허용 오차를 지운다)

- 상태: 확정
- 날짜: 2026-10-02
- 마일스톤: R6 (리팩터 — TASKS.md)
- 관련: [ADR 0047](0047-review-gallery-lives-in-internal-preview.md)(검수 갤러리 — 결정 4의 허용 오차
  부분 · 결정 5 · 결정 6을 이 ADR이 대체한다),
  [ADR 0064](0064-the-contract-is-component-and-layout-parity-not-pixel-parity.md)(컴포넌트·레이아웃 동일성),
  [ADR 0065](0065-pptx-validity-is-judged-by-the-ecma-376-schema-not-by-libreoffice.md),
  [CLI 레퍼런스](../../docs/reference/cli.md#balpyo-preview)

## 맥락

ADR 0047의 갤러리는 슬라이드마다 웹 · PPTX · **픽셀 diff** 세 칸을 놓고, 다른 픽셀 비율이 기본
허용 오차(2.5%)를 넘으면 "허용 오차 초과" 배지를 달았다. 그 숫자는 픽셀 동일성이 계약이던 때의
것이다. ADR 0064로 계약이 컴포넌트·레이아웃 동일성이 되자 그 숫자는 **계약 밖의 차이**(줄바꿈 ·
글리프 · 안티앨리어싱)를 재는 값이 되었다 — 3회차 현장 시험에서 이미 비율이 실제 불일치를 과장한다는
것이 확인됐고, `normAutofit`(R5)은 PowerPoint가 글자를 줄여 맞추게 하므로 LibreOffice 렌더와의
픽셀 차이는 더 커질 뿐 의미가 더 생기지 않는다.

또 하나: 기본 `--target html,pptx`에서 `soffice`가 없으면 빌드 전에 멈췄다 (결정 5 — "조용히 웹 열만
있는 갤러리를 내면 사람은 두 타깃이 같다고 읽는다"). Docker가 사라진 뒤(ADR 0065) 대부분의 개발자
호스트에는 LibreOffice가 없고, 그러면 `make review`와 기본 `balpyo preview`가 **아무것도 보여 주지
않는다**.

## 선택지

1. **그대로 둔다.** 숫자는 계약과 무관해졌고, 도구가 없는 호스트에서 기본 명령이 실패한다.
2. **diff는 지우고, 도구가 없으면 계속 멈춘다.** 숫자 문제는 풀리지만 기본 명령이 실패하는 문제는 남는다.
3. **diff를 지우고, 기본 렌더에서만 도구가 없으면 웹 열로 물러선다 — 그 사실을 로그와 갤러리 안내에
   말한다. 명시한 `pptx`는 여전히 빌드 전에 멈춘다.**

## 결정

**선택지 3.**

1. 갤러리는 슬라이드마다 렌더 칸(web · pptx)만 놓는다. **diff 이미지 · 비율 · 허용 오차 · 배지를 지운다**
   (`internal/preview/diff.go`의 `Diff`/`DiffOptions`/`DefaultDiffOptions`, `Verdict`, `Result.Over`).
   판정은 `go test`(동일성 · 스키마)가 하고, 갤러리는 눈으로 보는 자리다. ADR 0047 결정 4의 "게이트가
   아니다"는 그대로이고, 허용 오차를 보여 준다는 부분이 없어진다.
2. `--target`을 **주지 않으면** html과, 래스터 도구(`soffice` · `pdftoppm`)를 찾았을 때 pptx를 그린다.
   못 찾으면 웹 열만으로 갤러리를 만들고 — **조용히 빼지 않는다** — 로그에 "PPTX 열을 뺀다 — <이유>"를,
   갤러리 안내에 PPTX 열을 뺐다는 문장을 쓴다. 이 경우 PPTX를 빌드하지 않으므로 "직접 열어 보라"는
   안내도 내지 않는다. (결정 5를 대체한다.)
3. `--target`에 `pptx`를 **명시했는데** 도구가 없으면 지금처럼 빌드 전에 멈춘다 — 그 사람은 PPTX를
   보려고 명령을 쳤다. `--target ""`(빈 값을 준 것)은 주지 않은 것이 아니라 잘못된 입력이다.
4. `make review`는 픽스처마다 `balpyo preview`를 부르는 얇은 입구로 남는다 — 도구가 없는 호스트에서도
   웹 갤러리까지는 만든다.

ADR 0047 결정 7(PNG 크기 = 논리 스테이지 크기)은 그대로다 — 두 칸을 같은 크기로 나란히 놓는 데 쓰인다.

## 근거

- **계약 밖의 숫자를 보여 주면 계약처럼 읽힌다.** "허용 오차 초과" 배지가 붙은 슬라이드를 사람은 고칠
  자리로 읽는다. 지금 그 배지가 가리키는 것은 대부분 줄바꿈 · 글리프 차이고, 그것은 ADR 0064가 각 엔진의
  몫이라고 정한 것이다.
- **물러서기는 기본값에서만이다.** 결정 5가 막으려던 것 — PPTX를 보려 했는데 웹만 보고 같다고 읽는 것 —
  은 명시한 `pptx`에서 그대로 막는다. 기본값으로 친 사람에게는 안내 문장이 같은 일을 한다.
- **도구가 없는 호스트가 기본이다.** Docker가 없어진 뒤 LibreOffice는 선택 설치이고, 검수 명령이 그것
  없이 아무것도 내지 못하면 저작 루프에서 쓰이지 않는다.

## 결과

- `internal/preview/diff.go` · `diff_test.go`가 없어지고, 남는 PNG 도우미(`RGB8` · `DecodePNG` ·
  `EncodePNG` · `LoadPNG` · `SavePNG`)는 `png.go`로 옮긴다. 래스터 결정성 테스트는 0 오차 diff 대신 픽셀
  동등성을 직접 본다.
- `preview.Result`에 실제로 놓인 `Renders`와 `SkippedPPTX`(뺀 이유)가 생긴다. `SlideReport`는 자리 ·
  id만 갖는다.
- `balpyo preview`의 `--target` 기본값 표기가 `html,pptx`에서 "html과, 도구가 있으면 pptx"로 바뀐다.
- LibreOffice 렌더와 웹의 차이를 수치로 보고 싶어지면 그것은 저장소의 진단 도구로 따로 만든다 — 사용자
  명령의 출력으로 되돌리지 않는다.
