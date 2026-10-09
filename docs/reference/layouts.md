# `layout=` — 슬라이드 레이아웃 11종

슬라이드의 `layout` 속성은 그 슬라이드의 **스테이지 배치**를 고른다. 값 어휘는 표준 카탈로그
11종으로 닫혀 있고, 어휘 밖 값은 `line:col`을 가진 빌드 에러다 (`core.slideLayout`).
생략하면 `content`다.

배치의 **실체는 테마가 소유한다**: 각 레이아웃은 테마 패키지의 `layouts/<이름>.css`에
**배치 전용 CSS 선언**으로 정의되고, 렌더러는 그 선언을 슬라이드 스테이지에 얹는다. 웹의
스테이지 CSS와 PPTX 좌표가 **같은 한 선언**에서 나오므로 두 타깃의
배치가 갈라지지 않는다 — 브라우저가 잰 박스가 그대로 PPTX 좌표가 된다.

## 11종 카탈로그 (minimal 테마)

아래 표는 기본 테마 `minimal`의 배치다. **모든 테마가 11종을 전부 구현한다** — 빠뜨린
레이아웃을 다른 테마로 폴백하지 않으므로, 같은 `layout=` 값이 테마마다 다른 자리를
뜻한다 (`editorial`의 표지는 아래로 내려앉고, `ink`의 구역 표지는 가로도 가운데다 —
[테마 고르기·만들기](../guides/theming.md)). 배치 CSS가 **없는** 레이아웃(아래 "배치"가 비어
있는 것)은 스테이지에 레이아웃 클래스를 붙이지 않는다 — 기본 스택/자유 배치를 렌더러가
소유하기 때문이다.

| `layout` | 쓰임 | minimal의 배치 (`layouts/*.css`) |
|---|---|---|
| `content` | 제목 + 본문 흐름 (**기본값**) | *(없음 — 세로 스택. 렌더러 기본)* |
| `title` | 표지 | `justify-content: center` (세로 가운데) |
| `section-divider` | 섹션 표지 | `justify-content: center` |
| `quote` | 인용 | `justify-content: center` |
| `stat` | 큰 수치 강조 | `justify-content: center` |
| `end` | 마무리 | `justify-content: center` |
| `two-column` | 2단 | `display: grid; grid-template-columns: 1fr 1fr; align-content: start` |
| `image-text` | 그림 + 글 | `display: grid; grid-template-columns: 1fr 1fr; align-content: center` |
| `image-full` | 전면 그림 | `padding: 0` (가장자리까지) |
| `code` | 코드 | *(없음 — 세로 스택. 코드 블록은 위에서 쌓는다)* |
| `canvas` | 자유 배치 | *(없음 — 스테이지가 stack이 되어 자식이 `x`/`y`·`at`으로 자리를 정한다)* |

2단 계열(`two-column`·`image-text`)은 자식을 **문서 순서대로** 두 열에 채운다 — 좌우 배치는
저자가 쓴 자식 순서가 정한다. 세로 가운데 계열은 스테이지가 flex column이므로
`justify-content`가 곧 세로축 정렬이다.

## 배치 CSS의 경계 (배치 vs 외관)

`layouts/*.css`에는 **배치 프로퍼티만** 쓸 수 있다 — 지오메트리 패스가 재는 박스를 움직이는
것들이다 (`display`·`flex-*`·`grid-*`·`gap`·`padding`·`margin`·`width`/`height`·`align-*`·
`justify-*`·`place-*`·`order`·`aspect-ratio`·`position`·`inset`·`top/right/bottom/left`·
`translate`). 색·테두리·그림자·글꼴 같은 **외관은 CSS에만 남고 PPTX 마스터에는 도달하지
않으므로**, 여기 쓰면 경고이고 `--strict`가 에러로 승격한다. 외관은
`components.json`에 선언해야 두 타깃이 같은 룩을 얻는다. 주석(`/* … */`)은 허용된다.

## 선언하지 않은 레이아웃

**테마는 11종을 전부 선언해야 한다** — 다른 테마의 구현으로 폴백하지 않는다.
내장 테마 셋은 11종을 모두 갖췄으므로, 이 조건은 자기 테마에서만 걸린다.

걸렸을 때 일어나는 일은 **아무 일도 일어나지 않는 것**이다: 배치 CSS가 없으면 스테이지에
레이아웃 클래스가 붙지 않아 렌더러 기본 세로 스택(`content`와 같은 그림)이 남는다. 그 조용함이
폴백을 막은 이유다 — 그래서 프로젝트 로컬 테마도 **로드 시점에** 멈춘다: `layouts` 선언과
`layouts/*.css` 파일 집합의 양방향 대조가 어느 슬라이드도 렌더되기 전에 돈다. 상속하는 테마는
그 선언과 파일을 부모에서 물려받으므로(`extends` — [테마 패키지](theme.md))
갈아 끼울 파일만 두면 된다.

**테마 자기만의 레이아웃 이름은 없다.** 이 카탈로그 11종이 `layout=`의 어휘 전부이고
(`layout="corp-hero"`는 진단이다), 테마가 만드는 새 배치는 이름이 아니라 이 11종의 CSS와 슬롯
영역으로 짓는다.

## 슬롯 (`slot=`) — 이름 있는 영역

레이아웃은 **이름 있는 영역**(슬롯)을 선언하고, 슬라이드의 **직계 자식**이 `slot="이름"`으로
그 영역을 채운다. 슬롯을 하나라도 쓴 슬라이드는 **슬롯 모드**가 되고, 안 쓰면 위의 문서 순서
흐름 배치를 그대로 쓴다 (추가형 — 한 레이아웃을 두 슬라이드가 슬롯·흐름으로 나눠 쓸 수 있다).

슬롯은 두 타깃에서 서로 다른 두 가지가 된다 (같은 한 선언에서):

- **웹**: 스테이지가 `.balpyo-slots` 모디파이어를 얻고, 테마의 `layouts/<이름>.slots.css`가
  `grid-template-areas`로 정의한 그리드 영역에 자식이 `grid-area`로 배치된다. 흐름 규칙은
  `:not(.balpyo-slots)`, 슬롯 규칙은 `.balpyo-slots`로 배타적으로 겨눈다.
- **PPTX**: 슬롯을 채운 **단일 프레임** 자식(`<text>`·블록·리스트·`<shape>`·`<image>`)이 진짜
  `<p:ph>` 플레이스홀더가 된다 — PowerPoint의 개요 보기·서식 초기화·"클릭해 입력"이 인식하는
  플레이스홀더다. 좌표·서식은 여전히 잰 박스와 확정 rPr이 소유한다 (`<p:ph>`는
  표식일 뿐이다). 슬롯을 쓴 슬라이드는 그 레이아웃의 플레이스홀더를 선언한 전용
  `slideLayout` 파트를 가리키고, 마스터가 title/body 프로토타입과 `txStyles`를 얻는다.
  **컨테이너에 실린 슬롯**(`<col slot="left">` 등)은 웹 영역일 뿐이고 `<p:ph>`가 되지 않는다
  (단일 프레임이 아니라 자식으로 내려가기 때문).

슬롯 어휘와 phType/idx 매핑은 **엔진이 정한 닫힌 목록**이다(layout 어휘와 같은
태도 — 목록 밖 슬롯은 `line:col`을 가진 빌드 에러). 시각 영역(`grid-area`)은 테마가
소유한다. **내장 테마 셋 모두** 슬롯을 가진 **10종 전부**에 `layouts/<이름>.slots.css`를 둔다
(`canvas`만 슬롯이 없다).

**테마가 받아야 슬롯에 뜻이 있다.** 어휘에 있는 이름이라도 테마에 그 레이아웃의 슬롯
영역(`layouts/<이름>.slots.css`)이 없으면 콘텐츠가 앉을 자리가 없다. 그때는 **경고 한 줄과
함께 슬롯을 버리고** 그 슬라이드를 흐름 배치로 렌더한다 — 두 타깃 모두에서다.
에러가 아니라 경고인 이유는 오타가 아니라
**테마가 부족한 것**이고, 테마를 바꾸면 같은 소스가 그대로 뜻을 얻기 때문이다 (`--strict`가
승격한다).

| `layout` | 슬롯 (이름 → `p:ph` type[/idx]) |
|---|---|
| `title` | `kicker`→body/2, `title`→ctrTitle, `subtitle`→subTitle/1, `author`→body/3, `date`→body/4 |
| `section-divider` | `title`→title, `subtitle`→subTitle/1 |
| `content` | `title`→title, `body`→body/1 |
| `two-column` | `title`→title, `left`→body/1, `right`→body/2 |
| `image-full` | `media`→pic/1, `caption`→body/2 |
| `image-text` | `title`→title, `media`→pic/1, `body`→body/2 |
| `quote` | `quote`→body/1, `attribution`→body/2 |
| `code` | `title`→title, `code`→body/1, `caption`→body/2 |
| `stat` | `title`→title, `value`→body/1, `label`→body/2, `body`→body/3 |
| `end` | `title`→title, `contact`→body/1 |
| `canvas` | *(없음 — 자유 배치, 슬롯 어휘가 비어 있다)* |

`idx`가 없는 행(`title`·`ctrTitle`)은 `p:ph`에 `idx` 속성을 내지 않는다 (제목류의 관례).
`layout="canvas"`처럼 슬롯이 없는 레이아웃에 `slot=`을 쓰면 빌드 에러다.

## 세로 리듬은 누가 소유하는가

가로 정렬처럼 세로 분배에도 소유자가 둘 있고, **모드가 그것을 가른다**:

| 모드 | 세로 리듬의 소유자 |
|---|---|
| 흐름 모드 | 저자 — [`<slide justify="between">`](soml/slide.md) 한 선언이 스테이지의 세로 분배다 |
| 슬롯 모드 | 테마 — `layouts/<이름>.slots.css`의 `grid-template-rows`·`align-content` |

흐름 스테이지는 세로 flex이므로 `justify`가 곧 세로축 분배이고, 값 어휘는 컨테이너
(`<row>`·`<col>`)의 `justify`와 **같은 6종**이다: `start` · `center` · `end` · `between` ·
`around` · `evenly`. 슬롯 모드 슬라이드에 `justify`를 쓰면 빌드 에러다 — 그 세로 리듬은
테마의 몫이고, 두 소유자가 같은 축을 움직이면 어느 쪽이 이기는지 저자가 외워야 한다.

이 채널이 없던 동안 저자는 슬라이드마다 `<col h="fill" justify="between">` 랩퍼를 손으로
만들었고, **그 랩퍼 안의 자식은 슬라이드 직계 자식이 아니게 되어 `slot=`을 쓸 수 없었다** —
세로 리듬과 슬롯이 서로를 배제했다.

## 정렬은 누가 소유하는가

가로 정렬에는 채널이 둘 있고, **어느 쪽이 이기는지는 폭을 누가 정했는지가 결정한다**:

- `align`(저자·테마의 외관)은 **박스 안에서 문단을 움직인다** (`text-align`/`a:pPr@algn`).
- `justify-items`·`justify-content`·`place-self`(테마의 배치 CSS)는 **박스 자체를 움직인다** —
  박스가 영역보다 좁을 때만 움직일 여지가 있다.

그래서 텍스트 박스의 폭이 문맥마다 다르다:

| 문맥 | 텍스트 박스 폭 | 가운데로 두는 방법 |
|---|---|---|
| 흐름 모드 슬라이드 직계 | 콘텐츠 박스 전체 (`width: 100%`) | 저자가 `align="center"` |
| 슬롯 영역 안 | `auto` (내용만큼) | **테마**가 `justify-items: center` |
| `<row>`·`<col>`·`<grid>` 자식 | `auto` | 컨테이너의 `justify`·`align` |
| `<stack>`·canvas 자식 | `auto` | 자식의 `at`, 또는 `w=` + `align=` |

세 문맥이 `width: 100%`를 `auto`로 되돌린다 — 그 리셋이 없으면 테마가 `*.slots.css`에 쓴
`justify-items`가 아무것도 하지 못하고, 저자가 슬롯 자식마다 `align="center"`를 손으로 달게
된다. 표지 예제(`examples/balpyo-intro/slides/001_cover.slide`)에 `align`이 한 줄도 없는
이유가 그것이다.

폭이 `auto`인 문맥에서 특정 폭 안에 글을 가운데 두려면 그 폭을 텍스트에 직접 준다
([`<text w="140" align="center">`](soml/text.md)) — 두 채널을 함께 쓰는 자리다.
