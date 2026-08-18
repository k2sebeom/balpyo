# 항목 08 — Iteration 1 문서

TASKS.md의 문서 규칙이 이 항목의 계약이다: **구현된 동작만 쓴다.** 미래 계획·미구현 속성은
레퍼런스에 넣지 않는다 (그건 PRD와 TASKS의 몫이다). 예제는 **실제 픽스처**
(`testdata/fixtures/01_text/`)에서 가져온다 — 문서 예제가 곧 테스트된 코드가 되도록.

## 쓰는 문서

- `docs/reference/soml/slide.md` — `<slide>`의 `id`·`skip`. `layout`은 아직 없으니 쓰지 않는다.
- `docs/reference/soml/text.md` — `<text>`와 `role`/`size`/`color`/`align`.
- `docs/reference/soml/note.md` — `<note>`와 PPTX 노트 슬라이드.
  세 페이지 모두 형식을 통일한다: **속성 표 → SOML 예제 → 웹/PPTX 각각의 렌더 결과 설명 →
  빌드 에러가 나는 조건**. 에러 조건은 `core/resolve.go`와 `soml/parse.go`의 실제 진단
  메시지에서 가져온다 (문서와 메시지가 갈라지면 문서가 틀린 것이 된다).
- `docs/reference/appearance.md` (신규) — `role`/`size`/`color`/`align`만 있는 초판 매핑 표.
  각 행은 **웹(CSS)과 PPTX(OOXML) 양쪽 대응**을 함께 적는다 (PRD §9.4의 살아있는 판).
  토큰 색이 `var(--balpyo-color-*)`와 `a:schemeClr`로 나가는 계약을 명문화한다 — 이것이
  제품의 핵심 계약이므로 표의 각주가 아니라 본문에 쓴다.
- `docs/reference/cli.md` (신규) — `build`만. `--target`·`--project`·`--out`, 종료 코드,
  진단 출력 형식. `init`/`server`는 아직 없으니 쓰지 않는다.
- `docs/internals/geometry-pass.md` (신규) — CDP 패스, 슬라이드 상대 좌표, EMU/pt 환산
  경계, 텍스트 박스 고정 전략. 근거는 이미 코드 주석에 있다 (`render/geom`의 패키지 주석,
  `measureScript`, `pptx.Offset`, `pipeline.needsGeometry`) — **옮기지 말고 정리한다**:
  코드 주석은 왜, 문서는 전체 그림과 순서.
- `docs/internals/architecture.md` — 덱 모델 섹션을 완성한다 (노드 종류, 외관이 붙는 자리,
  파서/core/렌더러의 소유 경계, 지오메트리가 덱이 아니라 곁 표인 이유).
- `mkdocs.yml` — 새 페이지를 nav에 넣는다. `make docs`(strict)가 통과해야 한다.

## ADR

이번 iteration에서 확정된 설계 결정을 `docs/decisions/NNNN-*.md`로 남긴다 (맥락 → 선택지 →
결정 → 근거). 최소 두 건은 확실히 ADR 감이다:

1. **요소 외관을 인라인 `style` 속성으로 낸다** — cascade layer 위에 놓이므로 저자 CSS가
   외관을 덮을 수 없다. 외관은 스키마에서 순방향 컴파일된다는 계약과 `style` 탈출구가
   배치 전용이라는 규칙(Iteration 3)이 근거다. 대안(`data-balpyo-id` 선택자로 생성 규칙)을
   왜 버렸는지 함께 적는다.
2. **지오메트리를 덱 모델이 아니라 곁 표로 두고 생성자로 주입한다** — 덱은 공개 계약이고,
   덱에 실으면 값이 채워지는 시점이 `--target` 순서에 의존한다.

`NOTES.md`의 앞선 항목들을 읽고 **혼자 정한 결정**으로 기록된 것을 훑어, ADR로 남길 것이
더 있는지 판단한다. 있으면 함께 쓴다. 결정 로그 표(TASKS.md 하단)는 ADR의 색인이므로
같이 갱신한다.

## 끝으로

`TASKS.md`의 Iteration 1 체크박스를 마무리한다 — 이 항목까지 끝나면 iteration 전체가
닫히므로, **실제로 끝난 것만** `[x]`로 바꾼다. 항목 07에서 측정이 실패로 결론 났다면
그 사실이 TASKS.md에 남아야 한다 (닫힌 것처럼 표시하지 않는다).

## 범위 밖

- 구현 변경. 문서를 쓰다 코드의 문제를 발견하면 고치지 말고 NOTES.md의 **남긴 것**에 적는다
  (문서 항목이 코드를 바꾸면 그 변경은 아무 테스트도 없이 들어간다).
