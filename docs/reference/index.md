# 레퍼런스

"무엇이 있고 어떻게 쓰는가". 이 섹션은 **구현된 것만** 담는다 — 계획된 태그와 속성은
[PRD](../index.md)와 `TASKS.md`의 몫이다. 그래서 여기 있는 것은 모두 픽스처로 테스트되어
있고, 예제는 `testdata/fixtures/`에서 가져온다.

구성:

- `soml/` — 태그별 레퍼런스. 태그 1개(또는 밀접한 세트)가 파일 1개다. 각 페이지는 같은
  형식을 따른다: 속성 표 → SOML 예제 → 웹/PPTX 각각의 렌더 결과 → 빌드 에러가 나는 조건.
- `appearance.md` — 외관 속성 전체와 타깃별 컴파일 매핑 표.
- `layouts.md` — 표준 레이아웃 카탈로그와 슬롯 목록.
- `theme.md` — 테마 패키지 포맷.
- `cli.md` — 서브커맨드 레퍼런스.

## 현재 상태

Iteration 1까지 구현된 어휘와 표면:

- 태그: [`<slide>`](soml/slide.md) (`id`·`skip`), [`<text>`](soml/text.md)
  (`role`·`size`·`color`·`align`), [`<note>`](soml/note.md)
- [외관 속성](appearance.md) — 매핑 표와 토큰 색 보존 계약
- [CLI](cli.md) — `build`와 `version`

`layouts.md`와 `theme.md`는 해당 iteration(3·10)에서 생긴다. 구조와 파이프라인은
[내부 구조](../internals/architecture.md)에 있다.
