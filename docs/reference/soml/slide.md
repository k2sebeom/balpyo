# `<slide>`

`.slide` 파일 하나의 루트 요소다. 슬라이드 하나가 파일 하나이고, 파일의 역할은 확장자가
아니라 루트 요소가 정한다 (PRD §13-2).

## 속성

| 속성 | 값 | 기본값 | 뜻 |
|---|---|---|---|
| `id` | 문자열 | 파일명에서 유도 | 딥링크 앵커. 덱 전체에서 유일해야 한다 |
| `skip` | 없음 · `true` · `false` | `false` | 참이면 이 슬라이드를 렌더에서 뺀다 |

- `id`를 생략하면 파일명에서 유도한다: 순서용 숫자 접두(`001_`, `010-`)를 떼고 확장자를
  뗀 나머지가 앵커가 된다 — `001_title.slide`는 `title`. 순서 접두가 앵커에 들어가지
  않으므로 슬라이드를 재배치해도 링크가 살아남는다.
- `skip`은 HTML 불리언 속성 관례를 따른다 — 값 없는 `<slide skip>`은 참이다. 건너뛴
  슬라이드는 덱 모델에 아예 들어가지 않으므로 슬라이드 번호에 구멍이 생기지 않는다.

## 예제

`testdata/fixtures/01_text/slides/001_title.slide`:

```html
<!-- 표지. size·align을 명시해 파서가 원문 값을 그대로 보존하는지 본다. -->
<slide id="title">
  <text role="caption" size="20" color="muted" align="center">Platform Team</text>
  <text role="title" size="64" color="fg" align="center">2026 아키텍처 리뷰</text>
  <text role="body" align="center">상반기 회고와 하반기 계획</text>
  <note>인사 30초. 청중에 인프라팀 포함되어 있음.</note>
</slide>
```

## 렌더 결과

**웹** — `<section class="balpyo-slide">` 하나가 된다. `id` 속성이 그대로 HTML `id`
(딥링크 앵커)가 되고, `data-slide-index`(0-based 렌더 순서)와 `data-slide-source`
(원본 파일 경로)가 붙는다. 슬라이드는 논리 스테이지 크기(16:9는 1280×720px)를 그대로
갖고, 콘텐츠 자식은 `.balpyo-stage` 안에 문서 순서의 세로 스택으로 쌓인다 (여백 64px,
간격 24px).

**PPTX** — `ppt/slides/slideN.xml` 파트 하나가 된다 (N은 1-based 렌더 순서). `id`는
`p:cSld@name`으로 나가 PowerPoint의 슬라이드 이름이 된다. 콘텐츠 자식은 `p:spTree`의
도형이 되고, 문서 순서가 곧 도형 순서(겹침 z-순서)다.

지금 어휘에서 `<slide>`의 자식으로 올 수 있는 것은 `<text>`·`<note>`,
[텍스트 서브셋](text-subset.md)의 블록·리스트 태그(`h1`~`h3`, `p`, `blockquote`, `pre`,
`ul`/`ol`), [레이아웃 태그](layout.md)(`row`/`col`/`grid`/`spacer`), 그리고
[`<shape>`](shape.md)·[`<image>`](image.md)다. 인라인 태그는
슬라이드 직속이 될 수 없다 — 블록 안에 쓴다. 콘텐츠 자식은 [자식 배치 속성](layout.md)
(`grow`·`self`·`order` …)도 받는다 — 슬라이드의 세로 스택 자신이 flex 컨테이너다.

## 빌드 에러가 나는 조건

파서(`soml.Parse`)가 내는 진단 — 모두 `file:line:col` 위치를 갖는다:

| 조건 | 진단 메시지 |
|---|---|
| 루트가 `<slide>`가 아니다 | `루트 요소는 <slide>여야 한다 (<text>는 그 안에 쓴다)` |
| 파일에 `<slide>`가 없다 | `루트 요소 <slide>가 없다` |
| 파일에 `<slide>`가 둘 이상 | `루트 요소 <slide>는 파일에 하나여야 한다 (슬라이드 하나가 파일 하나다)` |
| 어휘 밖의 자식 태그 | `<slide>의 자식으로 <…>는 올 수 없다 (쓸 수 있는 태그: <text>, <shape>, <image>, <note>, <h1>, <h2>, <h3>, <p>, <blockquote>, <pre>, <ul>, <ol>, <row>, <col>, <grid>, <spacer>)` |
| 문자 데이터를 직접 썼다 | `<slide> 안에 문자 데이터를 직접 쓸 수 없다 (<text>로 감싼다)` |
| 어휘 밖의 속성 | `<slide>에 알 수 없는 속성 … (쓸 수 있는 속성: id, skip)` |
| `id`에 값이 없다 (`<slide id>`) | `id 속성에는 값이 필요하다 (id="architecture")` |
| `id=""` | `id 속성이 비어 있다 (지우면 파일명에서 유도한다)` |
| `skip` 값이 `true`/`false`가 아니다 | `skip 값은 true 또는 false다 (…)` |

검증기(`core.Validate`)가 내는 진단 — 파일을 넘어야 알 수 있는 것:

| 조건 | 진단 메시지 |
|---|---|
| 두 슬라이드의 ID가 같다 | `슬라이드 ID "…"가 …와 충돌한다 (id 속성으로 구분한다)` |
