# `layout=` — 슬라이드 레이아웃 11종

슬라이드의 `layout` 속성은 그 슬라이드의 **스테이지 배치**를 고른다. 값 어휘는 표준 카탈로그
11종으로 닫혀 있고(PRD §7.3), 어휘 밖 값은 `line:col`을 가진 빌드 에러다 (`core.slideLayout`).
생략하면 `content`다.

배치의 **실체는 테마가 소유한다**: 각 레이아웃은 테마 패키지의 `layouts/<이름>.css`에
**배치 전용 CSS 선언**으로 정의되고, 렌더러는 그 선언을 슬라이드 스테이지에 얹는다. 웹의
스테이지 CSS와 PPTX 좌표가 **같은 한 표**(`soml.Theme.Layouts`)에서 나오므로 두 타깃의
배치가 갈라지지 않는다 — 브라우저가 잰 박스가 그대로 PPTX 좌표가 된다 (PRD §9.3).

## 11종 카탈로그 (minimal 테마)

`minimal`은 폴백 기준이라 11종을 전부 구현한다. 배치 CSS가 **없는** 레이아웃(아래 "배치"가
비어 있는 것)은 스테이지에 레이아웃 클래스를 붙이지 않는다 — 기본 스택/자유 배치를 렌더러가
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
| `canvas` | 자유 배치 | *(없음 — 스테이지가 stack이 되어 자식이 `x`/`y`·`at`으로 자리를 정한다, PRD §13-3)* |

2단 계열(`two-column`·`image-text`)은 자식을 **문서 순서대로** 두 열에 채운다 — 좌우 배치는
저자가 쓴 자식 순서가 정한다. 세로 가운데 계열은 스테이지가 flex column이므로
`justify-content`가 곧 세로축 정렬이다.

## 배치 CSS의 경계 (배치 vs 외관)

`layouts/*.css`에는 **배치 프로퍼티만** 쓸 수 있다 — 지오메트리 패스가 재는 박스를 움직이는
것들이다 (`display`·`flex-*`·`grid-*`·`gap`·`padding`·`margin`·`width`/`height`·`align-*`·
`justify-*`·`place-*`·`order`·`aspect-ratio`·`position`·`inset`·`top/right/bottom/left`·
`translate`). 색·테두리·그림자·글꼴 같은 **외관은 CSS에만 남고 PPTX 마스터에는 도달하지
않으므로**(ADR 0014), 여기 쓰면 경고이고 `--strict`가 에러로 승격한다. 외관은
`components.json`에 선언해야 두 타깃이 같은 룩을 얻는다. 주석(`/* … */`)은 허용된다.

## 폴백

표준 카탈로그(`soml.SlideLayouts`) 중 테마가 `theme.json`의 `layouts`에 선언하지 않은
이름은 그 테마의 `content` 구현으로 폴백하고 경고한다. `minimal`은 11종을 전부 구현하므로
폴백이 없다.

## 슬롯 (`slot=`) — 이름 있는 영역 (ADR 0035)

레이아웃은 **이름 있는 영역**(슬롯)을 선언하고, 슬라이드의 **직계 자식**이 `slot="이름"`으로
그 영역을 채운다. 슬롯을 하나라도 쓴 슬라이드는 **슬롯 모드**가 되고, 안 쓰면 위의 문서 순서
흐름 배치를 그대로 쓴다 (추가형 — 한 레이아웃을 두 슬라이드가 슬롯·흐름으로 나눠 쓸 수 있다).

슬롯은 두 타깃에서 서로 다른 두 가지가 된다 (같은 한 선언에서):

- **웹**: 스테이지가 `.balpyo-slots` 모디파이어를 얻고, 테마의 `layouts/<이름>.slots.css`가
  `grid-template-areas`로 정의한 그리드 영역에 자식이 `grid-area`로 배치된다. 흐름 규칙은
  `:not(.balpyo-slots)`, 슬롯 규칙은 `.balpyo-slots`로 배타적으로 겨눈다.
- **PPTX**: 슬롯을 채운 **단일 프레임** 자식(`<text>`·블록·리스트·`<shape>`·`<image>`)이 진짜
  `<p:ph>` 플레이스홀더가 된다 — PowerPoint의 개요 보기·서식 초기화·"클릭해 입력"이 인식하는
  플레이스홀더다. 좌표·서식은 여전히 잰 박스와 확정 rPr이 소유한다 (선택지 3 — `<p:ph>`는
  표식일 뿐이다). 슬롯을 쓴 슬라이드는 그 레이아웃의 플레이스홀더를 선언한 전용
  `slideLayout` 파트를 가리키고, 마스터가 title/body 프로토타입과 `txStyles`를 얻는다.
  **컨테이너에 실린 슬롯**(`<col slot="left">` 등)은 웹 영역일 뿐이고 `<p:ph>`가 되지 않는다
  (단일 프레임이 아니라 자식으로 내려가기 때문).

슬롯 어휘와 phType/idx 매핑은 **soml이 소유하는 닫힌 카탈로그**다(`soml.LayoutSlots` — layout
어휘와 같은 태도, 어휘 밖 슬롯은 `line:col`을 가진 빌드 에러). 시각 영역(`grid-area`)은 테마가
소유한다. `minimal`은 `content`·`two-column`·`title`에 `*.slots.css`를 둔다.

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
