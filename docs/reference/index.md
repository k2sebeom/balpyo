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

Iteration 6까지 구현된 어휘와 표면:

- 태그: [`<slide>`](soml/slide.md) (`layout`·`id`·`skip`), [`<text>`](soml/text.md)
  (`role`·`variant`·`size`·`color`·`align`·`anchor`), [`<note>`](soml/note.md),
  [텍스트 서브셋](soml/text-subset.md) (`h1`~`h3`, `p`, `ul`/`ol`/`li`, `blockquote`,
  `pre`, `strong`, `em`, `code`, `a`, `br`, `span`),
  [레이아웃 태그](soml/layout.md) (`row`/`col`/`grid`/`spacer` + 컨테이너·자식 속성과
  `style` 탈출구), [`<stack>`](soml/stack.md) (겹침 컨테이너 + 자식 배치 `at`·`x`/`y`),
  [`<shape>`](soml/shape.md) (`type` 12종 + 외관 8종 + 박스 속성),
  [`<image>`](soml/image.md) (`src`·`fit`·`alt` + 외관 7종 + 박스 속성)
- 문법: [문자 참조와 CDATA](soml/text-subset.md) (닫힌 5종 + 수치 참조,
  원문 그대로 담는 섹션), 마크업이 아닌 `<`·`&`의 리터럴 규칙
- [외관 속성](appearance.md) — 4단 캐스케이드, 텍스트 역할 5종, 컴포넌트 기본값과
  variant 세 벌(텍스트·도형·이미지), 인라인 서식·리스트·도형·이미지 외관 매핑 표, 토큰 색
  보존 계약, 공간·반경·그림자 토큰
- [CLI](cli.md) — `build`(`--strict` 포함)와 `version`

`layouts.md`(표준 레이아웃 카탈로그)와 `theme.md`(테마 패키지 포맷)는 Iteration 10에서
생긴다 — 지금의 레이아웃은 2층 배치 태그이고, 1층 레이아웃과 슬롯은 아직 없다. 슬라이드의
`layout` 속성(`content`·`canvas` 2종)은 그때까지 [`<slide>`](soml/slide.md)가 레퍼런스를
갖고, Iteration 10이 카탈로그 페이지를 만들 때 링크를 그리로 옮긴다. 구조와 파이프라인은
[내부 구조](../internals/architecture.md)에 있다.
