# 레퍼런스

"무엇이 있고 어떻게 쓰는가". 이 섹션은 **지금 동작하는 것만** 담는다 — 여기
없는 태그와 속성은 쓸 수 없고, 쓰면 진단이 난다.

구성:

- `soml/` — 태그별 레퍼런스. 태그 1개(또는 밀접한 세트)가 파일 1개다. 각 페이지는 같은
  형식을 따른다: 속성 표 → SOML 예제 → 웹/PPTX 각각의 렌더 결과 → 빌드 에러가 나는 조건.
- `appearance.md` — 외관 속성 전체와 타깃별 컴파일 매핑 표.
- `layouts.md` — 표준 레이아웃 카탈로그와 슬롯 목록.
- `theme.md` — 테마 선택, 테마 패키지 포맷, 덱 크롬(페이지 번호·푸터) 선언.
- `cli.md` — 서브커맨드 레퍼런스.

## 현재 상태

지금 구현된 어휘와 표면:

- 태그: [`<slide>`](soml/slide.md) (`layout`·`id`·`skip`·`justify`), [`<text>`](soml/text.md)
  (`role`·`variant`·`size`·`color`·`align`·`anchor` + 가로 크기 `w`·`min-w`·`max-w` +
  배치 속성), [`<note>`](soml/note.md),
  [텍스트 서브셋](soml/text-subset.md) (`h1`~`h3`, `p`, `ul`/`ol`/`li`, `blockquote`,
  `pre`, `strong`, `em`, `code`, `a`, `br`, `span`, `mark`),
  [레이아웃 태그](soml/layout.md) (`row`/`col`/`grid`/`spacer` + 컨테이너·자식 속성과
  `style` 탈출구), [`<stack>`](soml/stack.md) (겹침 컨테이너 + 자식 배치 `at`·`x`/`y`),
  [`<shape>`](soml/shape.md) (`type` 16종 + 외관 8종 + 박스 속성),
  [`<image>`](soml/image.md) (`src`·`id`·`fit`·`alt` + 외관 7종 + 박스 속성),
  [`<table>`](soml/table.md) (`table`/`tr`/`th`/`td` + 프레임 `variant` + 셀 병합 2종·외관 5종),
  [`<chart>`](soml/chart.md) (`chart`/`point` + `type`·`unit`·`max`·`series`·`alt` — 값 층이
  도형·텍스트로 펼친다), [`<connector>`](soml/connector.md) (`from`·`to` + 경로 · 면 · 선 · 끝 표식 —
  잰 두 상자를 잇는다), [`enter`](soml/enter.md) (등장 효과 넷 — 테마의 박자로 엇갈린다)
- 문법: [문자 참조와 CDATA](soml/text-subset.md) (닫힌 5종 + 수치 참조,
  원문 그대로 담는 섹션), 마크업이 아닌 `<`·`&`의 리터럴 규칙
- [외관 속성](appearance.md) — 4단 캐스케이드, 텍스트 역할 5종, 컴포넌트 기본값과
  variant 세 벌(텍스트·도형·이미지), 인라인 서식·리스트·도형·이미지 외관 매핑 표, 토큰 색
  보존 계약, 공간·반경·그림자 토큰
- [CLI](cli.md) — `init`(빌드되는 세 장짜리 덱)·`build`(`--strict` 포함)·`check`(산출물 없는
  판정)·`watch`·`preview`(웹/PPTX 검수 갤러리)와 `version`

- [레이아웃과 슬롯](layouts.md) — 표준 카탈로그 **11종**(`content`·`title`·`section-divider`·
  `two-column`·`image-full`·`image-text`·`quote`·`code`·`stat`·`canvas`·`end`)과 레이아웃별
  슬롯 목록, 배치 CSS의 경계. 배치의 실체는 테마의 `layouts/<이름>.css`가 소유하고, 슬롯은
  웹의 `grid-area`와 PPTX의 `p:ph`가 **같은 카탈로그**에서 나온다.

- [테마 패키지](theme.md) — `theme.json`·`tokens.json`·`components.json`·`layouts/*.css`가
  나눠 갖는 것, 그리고 **덱 크롬**(페이지 번호·푸터): 내용과 켜짐은 `balpyo.json`의
  `footer`·`pageNumber`가, 자리·정렬·역할·제외 규칙은 테마의 `chrome` 선언이 소유한다.

테마 이름의 어휘는 **내장 셋(`minimal`·`editorial`·`ink`) ∪ 이 프로젝트의 `themes/<이름>/` ∪
`balpyo theme install`로 설치한 테마**다
— 고르는 방법과 세 테마의 성격, 자기 테마를 `extends`로 물려받아 만드는 흐름은
[테마 고르기·만들기](../guides/theming.md)에 있다. 테마를 가리키는 경로·URL 문법은 없다 — GitHub의
테마는 [설치](theme.md#설치한-테마-github에서-받는다)한 뒤 이름으로 부른다.
