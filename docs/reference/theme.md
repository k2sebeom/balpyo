# 테마 패키지

테마는 **덱의 디자인 시스템 하나**다. 저자가 쓰는 SOML에는 색·크기·자리가 거의
없고(`role="heading"`처럼 **뜻**을 쓴다), 그 뜻을 실제 값으로 바꾸는 표가 테마다. 그래서
테마를 갈면 같은 슬라이드가 다른 덱처럼 보인다.

v1에 실려 있는 테마는 **내장 셋**이다 — `minimal`(기본값)·`editorial`·`ink`. 고르는 방법과 세
테마의 성격은 [테마 고르기·만들기](../guides/theming.md)에 있고, 이 문서는 **테마 하나의
포맷**을 적는다. 프로젝트 안에 자기 테마를 두는 방법(`themes/<이름>/`)과 내장 테마를 물려받는
`extends`는 아래 **외부 테마 — 프로젝트 안의 이름** 절에 있다.

엔진이 하드코딩한 외관 값은 **하나도 없다**: 토큰·컴포넌트 외관·레이아웃 CSS·크롬 선언이 전부
아래 파일에서 로드되고, 그것이 빠지면 빌드가 **멈춘다** (폴백 테마가 없다).
그래서 이 포맷은 "언젠가 외부 테마가 쓸 규격"이 아니라 **지금 세 테마가 실제로 읽히는 규격**이고,
외부 패키지가 열릴 때 같은 구조를 그대로 쓴다.

## 테마 선택

테마 이름은 **덱의 요청**이다. 덱이 요청하고(`balpyo.json`), 값 층이 그 이름으로 패키지를
로드해 결과(토큰 표·컴포넌트 외관·배치 CSS)를 확정 모델에 싣는다.

```json
{ "theme": "ink" }
```

| 어디서 | 무엇을 | 우선순위 |
|---|---|---|
| `balpyo build --theme ink` | 이 실행만 다른 테마로 낸다 (`check`·`watch`·`preview`도 같다) | 가장 높다 |
| `balpyo.json`의 `theme` | 이 덱의 테마 | 다음 |
| 없음 | `minimal` | 기본값 |

쓸 수 있는 이름은 **내장 셋 ∪ 이 프로젝트의 `themes/<이름>/` ∪ 이 기계에 설치한 테마**다 — 어휘가
프로젝트와 기계마다 다르고, 그 목록을 진단이 그대로 나열한다 (아래 **외부 테마**와 **설치한 테마** 절).
`balpyo init --theme <이름>`만 **내장 테마로 제한**된다: 그 명령은 프로젝트를 만드는 명령이라
`themes/`가 아직 없다.

**없는 이름은 에러다.** 두 자리가 같은 어휘로 같은 판정을 한다: 설정 로드가 `balpyo.json`을
가리키며 쓸 수 있는 이름을 나열하고(`theme: unknown theme "inkk" — not a built-in theme, a theme in the
project's themes/, or an installed theme (available themes: editorial, ink, minimal)`), 값 층도 같은
문구로 **그 덱의 해석을 중단한다**. `minimal`로
폴백하지 않는 이유는 오타의 값이다 — 조용히 기본 테마로 그리면 저자는 "내 테마가 왜 안 먹지"를
슬라이드에서 찾는다. 모르는 `variant`가 경고 + 폴백인 것과 갈라지는 자리이기도 하다: variant
어휘는 테마의 사유물이지만 테마 **이름**의 어휘는 프로젝트가 소유한다.

슬라이드마다 다른 테마, 시스템 다크 모드 자동 전환, 테마를 가리키는 **경로** 문법은 열지 않았다.

## 패키지 구조

```text
minimal/
├── theme.json            # 매니페스트 — 이름 · 레이아웃 목록 · 덱 크롬 선언
├── tokens.json           # 디자인 토큰 (색 · 폰트 · 공간 · 반경 · 그림자)
├── components.json       # 컴포넌트 외관 — 텍스트 역할 · 태그/슬롯 기본값 · variant
├── layouts/
│   ├── content.css       # 레이아웃 배치 CSS (흐름 모드)
│   ├── content.slots.css #   같은 레이아웃의 슬롯 영역 CSS (슬롯 모드 — 선택)
│   └── ...               #   theme.json의 layouts에 적힌 이름마다 .css 하나
├── assets/               # 테마가 소유한 그림 — 배경 · 로고 · 슬라이드가 theme:으로 쓰는 그림 (선택, 배경 · 로고는 PNG · JPEG)
└── README.md             # 덱 저자에게 쓰는 글 — 쓰임새 · 어울리는 덱 · variant와 그림의 의도 (선택)
```

네 파일의 역할이 겹치지 않는다는 것이 이 구조의 요점이다. `README.md`는 로더가 읽지 않는 문서다
(아래 [`README.md`](#readmemd-덱-저자에게-쓰는-글)).

| 파일 | 소유하는 것 | 자세히 |
|---|---|---|
| `theme.json` | 이 테마의 정체 — 이름, 구현하는 레이아웃 목록, 덱 크롬 선언 | 아래 |
| `tokens.json` | **값의 이름표** — `accent1`·`md`·`heading`이 무슨 값인가 | 아래 (값은 [외관 속성](appearance.md)) |
| `components.json` | **역할과 variant의 외관** — `role="body"`가 몇 px인가, 코드 토큰이 무슨 색인가 | [외관 속성](appearance.md) |
| `layouts/*.css` | **자리** — 레이아웃 하나가 자식을 어떻게 놓는가 | [레이아웃과 슬롯](layouts.md) |

로더는 파일 사이의 **교차 대조**를 빌드 시점에 한다: `layouts/`의 `*.css` 이름 집합이
`theme.json`의 `layouts` 선언과 **양방향으로** 같아야 하고(선언했는데 파일이 없어도, 파일이
있는데 선언에 없어도 에러다), 크롬 선언의 `skipLayouts`는 이 테마가 구현하는 레이아웃
이름이어야 한다. 슬롯 영역 파일(`<이름>.slots.css`)만 대조가 **단방향**이다 — 슬롯이 없는
레이아웃(`canvas`)은 파일이 없어도 되지만, 선언에 없는 레이아웃의 슬롯 파일은 에러다.
`tokens.json`·`components.json`의 모르는 키도 에러다 (닫힌 어휘). 예외는 `$comment`로 시작하는 키
(`$comment`, `$comment-accent` …)뿐이고, 두 파일의 어느 객체에 두어도 된다 — 값 옆에 그 값을 고른
이유를 적는 자리다.

`components.json`의 색 · 채움 · 테두리 값은 **저자가 인라인으로 쓰는 문법과 같다**. 그래서 테마가
그라디언트와 색 변형을 소유할 수 있다 — `"fill": "gradient(accent1 accent6, 135deg)"`,
`"border": "4 gradient(accent6 accent1)"`, `"color": "tint(accent1, 40%)"`. 원고에는 variant 이름만
남고(`<shape variant="hero">`), 그라디언트의 색은 테마가 바뀌면 함께 바뀐다. 테마가 쓸 수 있는 색은
**토큰 참조**뿐이다 — 토큰 이름, 변형된 토큰, 스톱이 전부 토큰인 그라디언트. PPTX에서 셋 다
`schemeClr`로 남아 PowerPoint의 테마 색 편집을 따라간다. 표 셀(`table`)은 단색 토큰만 받는다.
어긋난 테마는 **덱을 렌더하기 전에** 멈춘다: 테마의 구멍이 산출물에서 "글자가 없다"로
나타나면 원인을 찾는 자리가 테마에서 멀어지고, 테마는 슬라이드를 쓰는 저자가 고칠 수 없는
자리다.

같은 이유로 로더는 **완전성**도 요구한다: 아래 "채워야 하는 이름" 표의 토큰과 텍스트 역할 5종이
전부 있어야 하고, 역할이 가리키는 폰트·색은 **그 테마에 있는** 토큰이어야 한다. 폴백 테마를
지운 대가가 이 검사다 — 엔진에 대체 값이 없으므로 빠진 값은 `size: 0`의 제목이나
색이 없는 캡션으로 렌더된다.

## `theme.json` — 매니페스트

```json
{
  "name": "minimal",
  "extends": "",
  "layouts": [
    "title", "section-divider", "content", "two-column",
    "image-full", "image-text", "quote", "code", "stat", "canvas", "end"
  ],
  "chrome": [
    { "kind": "footer",     "align": "start", "role": "caption",
      "skipLayouts": ["title", "section-divider", "image-full", "end"] },
    { "kind": "pageNumber", "align": "end",   "role": "caption",
      "skipLayouts": ["title", "section-divider", "image-full", "end"] }
  ]
}
```

| 키 | 값 | 하는 일 |
|---|---|---|
| `name` | 문자열 | 테마의 이름 — 이름은 패키지가 소유한다 (비면 에러). `balpyo.json`의 `theme`와 `--theme`가 이 이름으로 가리킨다 |
| `extends` | 테마 이름 또는 빈 문자열 | 물려받을 부모 테마. 이름 어휘는 `theme`와 같다 (내장 ∪ 프로젝트 로컬). 내장 테마 셋은 비어 있다 — 아래 **외부 테마** 절 |
| `layouts` | 레이아웃 이름 목록 | 이 테마가 구현하는 레이아웃. [표준 카탈로그](layouts.md) **11종을 전부** 담아야 한다 (아래) |
| `chrome` | 크롬 선언 배열 | 아래 |
| `backgrounds` | 배경 선언 배열 (생략 가능) | 레이아웃별 슬라이드 배경 — 아래 **배경** 절 |

**빠뜨린 레이아웃은 다른 테마로 폴백하지 않는다.** 내장 테마 셋이 11종을 전부 구현하고, 그
사실을 전수 테스트(`TestBuiltinThemesImplementEveryStandardLayout`)가 지킨다 — 폴백이 섞인
테마는 디자인 시스템이 아니라 두 디자인의 평균이고, 그 섞임은 빌드가 조용한 채 슬라이드에서만
보인다. 상속하는 테마는 이 선언을 **부모에서 물려받으므로** 11종을 다시 적지 않아도 된다
(적으면 그 목록이 이긴다 — 그때는 그 목록이 다시 11종이어야 한다).

`layouts`에 **표준 카탈로그 밖의 이름**을 적는 것은 뜻이 없다: `<slide layout>`의 어휘는 11종으로
닫혀 있어(`layout="corp-hero"`는 진단이다) 어떤 슬라이드도 그 이름을 요청할 수 없고, 파일이
없으면 위의 양방향 대조가 멈춘다. 테마가 만드는 새 배치는 레이아웃 이름이 아니라 그 11종의
CSS와 [슬롯](layouts.md) 영역으로 짓는다.

## `tokens.json` — 표준 토큰 어휘

토큰은 **저자가 속성에 쓰는 이름의 집합**이다. `color="accent3"`·`gap="md"`·`radius="lg"`가
해석되는 자리가 이 파일이고, 이름이 여기 없으면 빌드가 멈춘다 — 쓸 수 있는 이름 목록을 진단이
함께 적는다 (`color token "brand" is not in the theme (available tokens: fg, bg, muted,
surface, accent1, …)`). 그래서 **토큰 이름은 테마가 소유하는 어휘**다: 새 테마는 값을 갈 수 있지만
이름을 빼면 그 이름을 쓴 덱이 빌드되지 않는다.

```json
{
  "color":  [{ "name": "accent3", "rgb": "15803D", "slot": "accent3", "ref": "accent3" }],
  "font":   [{ "name": "body", "latin": "Noto Sans", "ea": "Noto Sans CJK KR",
               "generic": "sans-serif", "scheme": "minor" }],
  "space":  [{ "name": "md", "px": 24 }],
  "radius": [{ "name": "md", "px": 12 }],
  "shadow": [{ "name": "soft", "dist": 4, "dir": 90, "blur": 16,
               "color": "fg", "alpha": 0.15 }]
}
```

다섯 범주는 모두 **순서 있는 배열**이다 (JSON 객체의 키 순서에 기대지 않는다) — 그 순서가 CSS
변수 선언 순서와 `clrScheme` 조립, 진단 메시지의 목록 순서를 결정적으로 만든다.

아래 "채워야 하는 이름"은 문서의 권고가 아니라 **로더의 검사**다 — 하나라도 없으면
테마 로드가 실패하고 덱은 렌더되지 않는다. 검사가 요구하는 것은 **이름**이지 값이 아니다:
이름이 있어야 저자가 속성에 쓸 수 있고(`color="accent6"`·`gap="md"`), 그 이름이 무엇으로
보이는지는 테마의 자유다.

| 범주 | 채워야 하는 이름 | 쓰이는 자리 | PPTX 테마 슬롯 |
|---|---|---|---|
| `color` | `fg` · `bg` · `muted` · `surface` · `accent1` · `accent2` · `accent3` · `accent4` · `accent5` · `accent6` | `color` · `fill` · `border` · 그라디언트 스톱 · 색 변형의 밑색 | `clrScheme` 12자리 (`slot`) |
| `font` | `heading` · `body` · `mono` | `font` · 코드 블록 | `fontScheme` major/minor (`scheme`) |
| `space` | `xs` · `sm` · `md` · `lg` · `xl` · `2xl` | `gap` · `pad` | **없음** (OOXML 테마에 간격 자리가 없다 — 값으로 구워진다) |
| `radius` | `sm` · `md` · `lg` | `radius` | **없음** (OOXML에 반경 개념이 없다) |
| `shadow` | `soft` | `shadow` | 없음 (도형마다 `outerShdw`를 굽고 **색만** 참조로 남는다) |

다섯 범주 밖에 **선택 절** 하나가 있다: `motion` — [등장](soml/enter.md)의 박자(ms)와 기본 슬라이드
전환이다 (`{ "duration": 600, "stagger": 90, "delay": 150, "transition": "fade", "transitionDuration": 400 }` —
`transition`은 생략하면 전환 없음). 배열이 아니라 객체 하나이고, 상속에서는 통째로
갈린다. 정하지 않은 테마도 로드되지만, 그 테마에서 `enter`를 쓴 덱은 에러다. PPTX 테마 파트에는 자리가
없다 — 효과마다 길이 · 지연으로 구워진다.

`minimal`의 실제 값과 각 수치의 근거(수직 리듬·대비·극좌표 그림자)는
[외관 속성](appearance.md)의 "내장 테마 `minimal`의 토큰" 절에 있다 — 그 문서의 값 표는
**`minimal`의 것**이고, 다른 테마의 값은 그 테마의 `tokens.json`·`components.json`이다
(제품이 테마마다 고정하는 수치는 본문 24px · 줄높이 1.5 하나다).

**색 토큰의 네 필드**가 토큰 색 보존 계약의 배선이다: `name`은 저자가 쓰는 이름,
`rgb`는 실제 색, `slot`은 이 색이 앉는 `clrScheme` 자리, `ref`는 도형·런이 그 자리를 가리킬 때
쓰는 `schemeClr` 값이다 (`dk1` 슬롯의 참조 이름이 `tx1`인 것처럼 둘이 다를 수 있다 — OOXML의
이름이 자리와 참조에서 갈린다). 슬롯이 **없는** 색 토큰이나 **겹치는** 슬롯은 에러다 (테마
로딩에서 — 진단이 `tokens.json`과 남은 자리를 나열한다): 슬롯이 없으면 그 색은 PowerPoint의 테마
색 편집을 따라가지 못하고, 겹치면 어느 색이 그 자리를 차지하는지 정할 수 없다. **그래서 한 테마의
색 이름은 최대 열둘이다** — 반경·공간·그림자·폰트에는 없는 상한이고, 상속하는 테마가 색을 더하는
대신 갈아 끼우는 이유다.
`clrScheme`의 12자리 중 표가 채우지 않은 자리는 가장 가까운 토큰에서
값을 **빌린다** (`accent2`~`accent6` → `accent1`, `hlink`/`folHlink` → `accent1`) — 자리를 비울
수 없고, Office 기본 색으로 채우면 사용자가 편집 UI에서 그 자리를 골랐을 때 테마와 무관한
색이 나온다. `minimal`은 강조색 여섯을 다 가지므로 빌리는 자리가 링크 둘뿐이다.

**폰트 토큰의 두 슬롯은 같은 가족을 가리켜야 한다.** `latin`과 `ea`는 글리프가 없을 때 엔진이
고르는 면인데, 그 선택을 하는 주체가 타깃마다 다르다 — 브라우저는 fontconfig에 물어보고
LibreOffice·PowerPoint는 `a:ea`에 적힌 이름을 그대로 쓴다. `mono`에서 이 규칙이 눈에 보인다:
**고정폭 라틴에는 고정폭 CJK `ea`가 필요하다**(`minimal`은 `Noto Sans Mono CJK KR`이다).
비례폭 면을 적어 두면 코드 블록의 한글 주석 한 줄이 두 엔진에서 다른 폭으로 그려진다 (측정
−7.9% → 짝을 맞춘 뒤 +0.6% — [계약 밖의 차이](../guides/known-differences.md)).

**폰트 토큰의 글꼴은 빌드하는 기계에 있어야 한다.** PPTX의 좌표는 브라우저가 그 기계의 글꼴로 잰
박스이고, PowerPoint는 토큰에 적힌 글꼴로 그린다. PowerPoint에만 있는 글꼴(`Malgun Gothic` 등)을
적으면 브라우저는 대체 글꼴로 재고 박스가 어긋난다 — 좌표를 재는 빌드가 없는 글꼴마다
[글꼴 경고](cli.md#글꼴-경고)를 낸다.

### 강조색 여섯의 규칙

강조색은 눈으로 고른 값이 아니라 **세 수치를 만족하는** 여섯이다.

| 규칙 | 수치 | 막는 실패 |
|---|---|---|
| `bg` 대비 | 4.5:1 이상 | 못 읽는 색. 대비율은 대칭이라 이 하나가 *바탕색 위의 강조색 글자*와 *강조색 면 위의 바탕색 글자*를 함께 보장한다 |
| 서로의 색상(hue) 간격 | 25° 이상 | 구별 안 되는 두 색 (프로젝터·색각 이상에서 먼저 무너진다) |
| 상대 휘도 띠 | 최대/최소 1.25 이하 | 한 색만 튀어서 **번호가 아닌 밝기가** 위계를 만드는 것 |

세 수치는 **내장 테마 전수**에 적용된다. 대비율이 두 색의 순서에 대해 대칭이므로 "`bg` 대비
4.5:1"이라는 한 문장이 밝은 테마와 어두운 테마에서 같은 뜻으로 성립한다.

| 테마 | `bg` 대비 | 휘도 띠 |
|---|---|---|
| `minimal` | 4.76 ~ 5.38 | 1.18 |
| `editorial` | 4.92 ~ 5.00 | 1.02 |
| `ink` | 5.93 ~ 6.00 | 1.02 |

내장 테마 셋은 이 수치를 WCAG 2.1 상대 휘도로 계산해 지킨다. 자기 테마에서는 로더가 이
수치를 검사하지 않으므로 직접 잰다. 노랑이
`minimal`의 팔레트에 없는 이유도 여기서 나온다: 흰 바탕에서 4.5:1을 만족하는 노랑은 이미
갈색이다 — 어두운 바탕에서는 그 천장이 반대로 움직여서 `ink`의 따뜻한 강조색(`accent4`
`CB8700`)이 `minimal`의 같은 자리(`B45309`)보다 밝다.

**번호는 위계가 아니다.** `accent2`는 `accent1`의 "두 번째로 중요한 색"이 아니라 색상환의
**옆 색**이고, 여섯이 한 휘도 띠에 있으므로 번호 사이에 강약이 없다. 위계는 **덱**이 만든다 —
한 덱에서 강조색을 둘이나 셋만 쓰고 각 색에 뜻을 하나씩 준다. 의미 이름 토큰(`danger`·`warn`)
을 어휘에 넣지 않은 이유는 뜻이 덱마다 다르기 때문이다 (회고 덱의 빨강은 "피해"지만 매출 덱의
빨강은 "경쟁사"다).

## 슬롯 영역의 가로 채널

슬롯 영역 CSS(`layouts/<이름>.slots.css`)는 grid 하나다 — 영역 이름이 슬롯 이름이고, 자식은 그
영역에 앉는다 ([레이아웃과 슬롯](layouts.md)). 그 파일에서 테마가 조심해야 하는 선언이
**`justify-items`** 하나다.

| 하려는 것 | 쓰는 것 | 왜 |
|---|---|---|
| 왼쪽에 세운다 | **아무것도 쓰지 않는다** | 글줄의 기본 정렬이 이미 왼쪽이다 — `justify-items: start`는 그림을 바꾸지 않고 **박스 폭만** 줄인다 |
| 가운데로 모은다 | `justify-items: center` | 이 방법만 측정에 남아 PPTX까지 간다 — 슬롯 CSS의 `text-align`은 배치가 아니라 외관이라 린터가 짚고(아래 표), 통과시켜도 PPTX에는 닿지 않는다 |

`justify-items`가 `stretch`(기본값)가 아니면 grid 항목이 **내용 폭**으로 줄어든다. 그러면 두
가지를 잃는다: 저자가 `align`으로 문단을 움직일 폭과, 잰 박스와
글줄 사이의 **여유 폭**이다. 여유가 0인 줄은 PowerPoint가 조금만 넓게 그려도 한 줄 더 접히고,
그러면 `normAutofit`이 글자를 줄여 박스에 맞춘다 ([계약 밖의 차이](../guides/known-differences.md)).
그래서 **가운데는 값을 내고 쓰고, 왼쪽은
선언하지 않는다**: 가운데가 곧 뜻인 자리(표지·`stat`의 수치 한 개·마무리)에서만 그 대가를
받아들인다.

## 배경 — `backgrounds`

테마는 레이아웃마다 **슬라이드 배경**을 정할 수 있다. 배경은 슬라이드의 내용이 아니라 테마의 정체라
원고에는 아무것도 적지 않는다 — 테마를 바꾸면 배경도 바뀐다.

```json
"backgrounds": [
  { "layouts": ["title"], "image": "assets/cover.png" },
  { "layouts": ["section-divider"], "fill": "gradient(accent1 accent6, 135deg)" },
  { "fill": "surface" }
]
```

| 키 | 값 | 하는 일 |
|---|---|---|
| `layouts` | 이 테마의 레이아웃 이름 목록 (생략 가능) | 이 배경을 받는 레이아웃. **생략하면 기본 배경**이다 (나머지 모든 레이아웃) |
| `fill` | 색 값 — 토큰 · 변형 · 토큰 그라디언트 | 바탕 채움 |
| `image` | `assets/` 아래 그림 경로 | 바탕 그림. 슬라이드를 **덮는다** — 비율을 지키고 넘치는 쪽을 가운데 기준으로 잘라 낸다 (`<image fit="cover">`와 같은 규칙) |
| `overlay` | 색 값 — 흔히 투명해지는 그라디언트 (생략 가능) | 바탕(채움 · 그림) **위**에 칠하는 덮개 — 사진 위의 어두운 띠, 모서리의 광채 |

- 배경 하나는 `fill` **또는** `image`다 — 둘을 함께 쓰면 에러다. PPTX의 슬라이드 배경은 채움 하나만
  담으므로, 그림 아래 깐 채움이 PPTX에서는 사라져 투명한 자리마다 두 산출물이 갈린다. 그라디언트 위의
  그림이 필요하면 둘을 그림 한 장으로 합친다.
- **덮개(`overlay`)는 바탕 위의 한 겹이다** — `{ "image": "assets/cover.jpg", "overlay": "gradient(fg fg/0, 90deg)" }`는
  사진 위에 왼쪽부터 투명해지는 어두운 띠를 칠한다. 웹은 슬라이드 배경의 윗겹이고, PPTX는 슬라이드 맨 뒤의
  **잠긴 전면 사각형**(`background overlay` — 선택 · 이동할 수 없다)이다: 슬라이드 배경은 채움 하나만 담는다.
- 레이아웃 하나는 **한 선언에만** 나온다. 기본 배경도 하나다. 겹치면 에러다 — 어느 것이 이기는지 외울
  순서 규칙을 두지 않는다.
- 배경을 정하지 않은 레이아웃은 지금처럼 `bg` 색이다. `backgrounds`를 쓰지 않는 테마의 산출물은 이
  기능이 생기기 전과 같다.
- 두 타깃의 자리: 웹은 슬라이드 요소의 배경(`background-color` · `background-image`), PPTX는 슬라이드의
  `p:bg`다 — PowerPoint의 "배경 서식"으로 고치는 자리이고, 끌어 옮길 수 있는 도형이 아니다. 색은
  `schemeClr`로 남는다.

그림은 테마 패키지의 `assets/` 아래에 둔다. `<image>`의 파일과 **같은 판정**을 받는다 — 배경 · 로고는 PNG · JPEG
(SVG는 슬라이드가 `theme:`으로 쓰는 그림에만 된다),
확장자와 내용이 같아야 하고, 고유 크기가 있어야 한다. 선언이 가리키는 그림만 산출물에 들어간다(웹은
`themes/<테마>/assets/…`, PPTX는 미디어 파트). `extends`로 물려받은 선언의 그림은 그 그림을 가진 테마의
것이고, 자식이 같은 경로에 파일을 두면 자식의 파일이 이긴다 (파일 단위 병합).

### 덱이 쓰는 그림 — `theme:`

`assets/`의 그림은 슬라이드도 쓸 수 있다: `<image src="theme:icons/a.png">`는 활성 테마 체인의
`assets/icons/a.png`다 (규칙은 [`<image>`](soml/image.md)의 `theme:` 절). 아이콘 · 일러스트 라이브러리를
테마에 담아 배포하는 자리다 — 설치한 테마로 빌드하는 덱도 프로젝트에 사본 없이 그 그림을 쓴다.

- 자식 테마는 같은 경로에 파일을 두어 **그림 하나만** 바꾼다 (예: 어두운 배경용 흰 아이콘). 원고의 `src`는
  그대로다.
- 테마가 로드할 때 `assets/`는 경로만 색인한다 — 바이트는 선언과 슬라이드가 실제로 가리킨 파일만 읽으므로
  수천 장짜리 라이브러리도 빌드를 느리게 하지 않는다.
- 파일 이름은 테마의 **공개 어휘**가 된다: 덱이 그 경로를 적으므로, 그림의 이름을 바꾸거나 지우면 그 그림을 쓰던
  덱이 빌드 에러가 난다. 다른 테마로 바꿔도 같은 경로를 쓰게 하려면 그 테마에도 같은 경로를 둔다.

## `README.md` — 덱 저자에게 쓰는 글

테마 파일은 값을 담지만 **의도**는 담지 않는다 — 이 테마가 어떤 덱에 맞는지, `variant="lead"`를 언제
쓰는지, `theme:icons/` 아래 그림 가운데 무엇을 무슨 뜻으로 넣었는지. 그것을 적는 자리가 테마
디렉토리의 `README.md`다. `balpyo theme info <이름>`이 그 글을 그대로 출력하고
([CLI](cli.md#balpyo-theme)), GitHub에서 저장소를 여는 사람도 같은 글을 본다.

- **파일 이름은 정확히 `README.md`다.** `readme.md`·`README.txt`는 읽지 않는다.
- **없어도 된다.** 로드·렌더·`--strict`와 상관이 없고 경고도 없다.
- **형식은 자유다.** 내장 테마의 README가 쓰는 절은 이렇다 — 참고할 뿐 요구하지 않는다:

| 절 | 담는 것 |
|---|---|
| 첫 문단 | 테마의 목소리 한두 문장 — 바탕 · 글꼴 · 강조색의 인상 |
| Works best for | 어울리는 덱 · 상황, 그리고 어울리지 않는 경우와 대신 쓸 테마 |
| Look | 원고를 쓸 때 알아야 할 배치의 특징 (예: 표지의 제목이 어디에 앉는가) |
| Variants | variant마다 **언제 쓰는가** — 값이 아니라 쓰임새 |
| Colors | 강조색을 덱에서 어떻게 나눠 쓰는가 |
| Assets | `assets/` 아래 그림마다 `theme:` 경로와 **넣은 의도** (무엇에, 무엇과 함께 쓰는가) |

`extends`로 물려받는 테마의 README는 자기가 바꾼 것만 적고 부모를 가리키면 된다 — `theme info`는
부모의 README를 합치지 않고, 부모를 볼 명령을 한 줄 적는다.

## 덱 크롬 — `chrome`

덱 크롬은 페이지 번호와 푸터, 그리고 로고다. 슬라이드마다 저자가 쓰는 것이 아니라 **덱의 사실**이고, 그래서
어휘를 두 곳이 나눠 갖는다.

| 누가 | 무엇을 | 어디에 |
|---|---|---|
| 저자(덱) | **내용과 켜짐** — 푸터 글자, 번호를 켤지 | `balpyo.json`의 `footer`·`pageNumber` |
| 테마 | **자리·정렬·역할·제외 규칙** | `theme.json`의 `chrome` |

둘 다 있어야 크롬이 나온다. 테마만 선언하면 아무것도 렌더되지 않고(내용이 없다), 설정만 켜면
**경고 + 미탑재**다 — 테마가 그 종류를 선언하지 않았다는 것은 "이 디자인에는 그 자리가 없다"는
뜻이므로, 엔진이 자리를 임의로 정해 넣는 대신 저자에게 알리고 넘어간다
(슬롯이 없는 레이아웃에 `slot=`을 쓴 것과 같은 규칙).

`<footer>` 같은 **태그는 없다**. 태그로 열면 크롬이 슬라이드의 세로 흐름에 들어가 본문 높이를
갉아먹고, 덱 전체에 같은 푸터를 넣는 일이 슬라이드 수만큼의 복사가 된다.

### 선언 하나의 키

| 키 | 허용 값 | 하는 일 |
|---|---|---|
| `kind` | `footer` · `pageNumber` · `logo` | 무엇의 선언인가. 같은 `kind`를 두 번 선언하면 에러다 |
| `align` | `start` · `center` · `end` | 아래 띠 안에서의 가로 자리 (웹은 `left`/`right`, PPTX는 `algn`) |
| `role` | [텍스트 역할](appearance.md) 5종 | 글자의 외관. 크롬은 자기 크기·색을 갖지 않는다 (`logo`에는 쓰지 않는다) |
| `skipLayouts` | 이 테마의 레이아웃 이름 목록 (생략 가능) | 이 크롬을 **싣지 않을** 레이아웃 |
| `image` · `height` · `alt` | `logo` 전용 — `assets/` 아래 그림 · 논리 px 높이 · 대체 텍스트 | 로고 그림. 폭은 그림의 비율이 정하고, 띠 가운데에 놓인다 |
| `format` | `pageNumber` 전용 — `{n}` 하나와 `{total}`을 담는 글 (생략 가능) | 쪽번호의 모양. `"{n} / {total}"`이면 `3 / 13`이다 |

**로고는 테마가 켠다.** 푸터 · 번호와 달리 그림도 자리도 테마의 것이라 `balpyo.json`이 정할 것이 없다 —
선언하면 모든 슬라이드(`skipLayouts`를 뺀)에 나온다. 웹은 띠 위의 `<img>`, PPTX는 그 자리의 그림(`p:pic`)이고
`alt`가 대체 텍스트(`cNvPr@descr`)다. 다른 크롬과 같은 자리(`align`)를 고르면 겹침 경고가 난다.

- `align`과 `role`은 **생략할 수 없다.** 기본값을 엔진에 두면 "테마가 자리와 타이포그래피를
  소유한다"는 결정이 반으로 쪼개진다 — 테마를 읽어도 크롬이 어디에 어떤 크기로 나올지 알 수 없다.
- `role`을 새로 만들지 않고 기존 역할(`caption`)을 **재사용**하는 이유는 같은 것이다: 크롬만의
  글자 크기를 테마에 하나 더 두면 "작은 부가 글자"의 값이 두 곳에 생기고 두 곳이 갈라진다.
- `skipLayouts`의 이름이 이 테마의 레이아웃이 아니면 **에러**다 (오타가 "제외되지 않았다"로
  나타나면 원인을 찾을 자리가 없다).
- 크롬은 콘텐츠 박스 **밖**, 스테이지 아래 여백 띠(64px)에 산다 — 본문과 겹치지 않는다.

### 번호는 덱 안의 위치다

번호는 1부터 세고, **제외된 장도 센다.** 표지가 번호를 달지 않아도 다음 장은 2다.

그 규칙의 근거는 PPTX다: 번호는 정적 글자가 아니라 `<a:fld type="slidenum">` 필드이고, 뷰어가
계산하는 값은 파일 안의 슬라이드 순서다 — 표지를 빼고 세는 방법이 뷰어에는 없다. 웹이 다른
규칙으로 찍으면 같은 덱의 두 산출물이 다른 번호를 보여 준다.

필드로 내는 이유는 **자동 갱신**이다. 저자가 PowerPoint에서 장을 더하거나 순서를 바꾸면 번호가
따라온다 — 정적 글자면 그 순간 번호가 거짓말을 시작한다. 파일에는 값 층이 센 숫자가 `a:t`에
**캐시**로 함께 들어간다: 필드를 계산하지 않는 소비자(썸네일 생성기·변환기)도 맞는 숫자를 보여야
하기 때문이다.

**모양(`format`)의 번호만 필드다.** `{n}` 앞뒤의 글자는 같은 서식의 보통 런이고, `{total}`은 **빌드할 때의 장 수**로
구워진다 — PowerPoint에서 장을 더하면 번호는 따라오지만 전체 쪽수는 다시 빌드할 때까지 옛 수다 (전체 쪽수의
필드가 DrawingML에 없다). 0을 채운 번호(`03`)는 없다: 필드는 자릿수를 정할 수 없고, 웹만 채우면 두 산출물이
다른 번호를 보여 준다. 어휘 밖이면 `pageNumber format "{page}" needs {n} exactly once and may use {total} — e.g. "{n} / {total}"`이다.

`<slide skip>`으로 뺀 장과 `skipLayouts`로 제외된 장은 다르다. skip은 덱에서 **아예 빠지므로**
뒤 장의 번호가 당겨지고, 제외는 자리를 지키고 크롬만 없다.

### 켜는 쪽 — `balpyo.json`

```json
{
  "title": "2026 Q3 리뷰",
  "footer": "2026 Q3 · 내부용",
  "pageNumber": true
}
```

| 키 | 기본값 | 켜지는 조건 |
|---|---|---|
| `footer` | `""` | 빈 문자열이 아니면 켜진다 (글자가 곧 켜짐이다) |
| `pageNumber` | `false` | `true`면 켜진다 (글자는 슬라이드가 정한다) |

둘 다 기본이 **꺼짐**인 이유는 크롬이 덱마다의 선택이라는 것이다 — 내부 공유 덱과 공개 발표
덱이 다르다. 그래서 이 필드를 쓰지 않는 덱의 산출물은 크롬 기능이 생기기 전과 **바이트가 같다**.

### 두 타깃에서의 산출물

| | 웹 | PPTX |
|---|---|---|
| 담는 것 | `.balpyo-slide` 안, `.balpyo-stage`의 **형제**인 `<div class="balpyo-chrome …">` | 슬라이드 파트의 평범한 `p:sp` (콘텐츠 도형 **뒤**) |
| 자리 | 아래 64px 띠에 절대 배치 + `align-items: center` | 브라우저가 잰 그 rect (`anchor="ctr"`) |
| 글자 | 번호도 글자다 (웹에는 필드가 없다) | 푸터는 `<a:r>`, 번호는 `<a:fld type="slidenum">` |
| 줄바꿈 | `white-space: nowrap` | `wrap="none"` (같은 규칙 — 웹이 접지 않기로 한 글자를 PPTX가 접으면 띠를 넘친다) |

PPTX에서 **플레이스홀더(`p:ph type="sldNum"`)를 쓰지 않는다.** 제외는 슬라이드 단위인데 흐름
모드 슬라이드는 레이아웃 파트 하나를 공유하므로, 플레이스홀더로는 "표지에만 번호가 없다"를
표현할 수 없다. 마스터에 `p:hf`도 내지 않는다: 상속할 것이 없고, 내면 크롬을 쓰지 않는 덱의
바이트까지 움직인다.

## 외부 테마 — 프로젝트 안의 이름

자기 테마는 **프로젝트 안에** 둔다. 디렉토리 이름이 곧 테마 이름이고, 그 이름을 `balpyo.json`이
가리킨다.

```text
my-deck/
├── balpyo.json           # { "theme": "corp" }
├── slides/
└── themes/
    └── corp/             # ← 디렉토리 이름이 테마 이름이다
        ├── theme.json    #   { "name": "corp", "extends": "minimal", … }
        ├── tokens.json   #   부분 선언이어도 된다 (상속할 때)
        ├── components.json
        └── layouts/
            └── title.slots.css
```

**경로 문법은 없다.** `"theme": "./themes/corp"`도 `"theme": "~/themes/corp"`도 쓸 수 없다 —
값은 언제나 **이름**이고, 그 이름의 어휘는 **내장 셋 ∪ `themes/` 아래 디렉토리**다. 이름 하나로
닫아 둔 이유는 둘이다: 진단이 어휘 하나만 나열하면 되고("available themes: corp, editorial, ink,
minimal"), 테마를 프로젝트 파일시스템 안에서만 읽으므로 `..`이나 절대 경로로 프로젝트 밖 파일을
읽는 길이 없다.

규칙 셋이 이름을 지킨다.

| 규칙 | 어기면 | 왜 |
|---|---|---|
| `themes/<이름>/theme.json`의 `name`이 디렉토리 이름과 같아야 한다 | 에러 — `themes/corp/theme.json has name "corp-dark" — it must match the directory name "corp"` | 요청은 이름으로 오고 파일은 경로로 찾는다. 둘이 갈리면 어느 이름을 써야 하는지 파일을 열어 봐야 안다 |
| 로컬 테마가 내장 이름을 가릴 수 없다 | 에러 — `themes/minimal/ has the same name as the built-in theme "minimal" — rename the local theme` | `extends: "minimal"`이 어느 minimal인지 답할 수 없다. 규칙으로 정할 수는 있지만 그 규칙을 외운 사람만 자기 테마의 출처를 안다 |
| 가림은 **요청한 테마와 무관하게** 멈춘다 | 에러 | `--theme ink`로 빌드하는 동안 `themes/minimal/`이 조용히 무시되면, 다음 사람이 그 디렉토리를 고쳐도 아무 일도 일어나지 않는다 |

`balpyo init --theme`만 이 어휘를 받지 않는다 (내장 셋뿐이다) — 프로젝트를 **만드는** 명령이라
`themes/`가 아직 없다.

### `extends` — 부모를 물려받는다

```json
{ "name": "corp", "extends": "minimal" }
```

부모 이름은 `theme`와 **같은 어휘**에서 풀린다 (내장이든 로컬이든). 체인은 길어도 된다
(`corp-dark` → `corp` → `minimal`) — 뿌리부터 잎까지 차례로 병합한다.

| 상황 | 결과 |
|---|---|
| 없는 이름을 가리킨다 | 에러 — `theme "corp" extends theme "minimalx", which does not exist (available themes: …)` (고칠 자리인 **자식**을 가리킨다) |
| 자기 자신을 가리킨다 | 에러 — `theme.json extends forms a cycle: corp → corp` |
| 돌아온다 (`a` → `b` → `a`) | 에러 — 체인을 그대로 적는다: `theme.json extends forms a cycle: a → b → a` |

상속하는 테마의 `tokens.json`·`components.json`·`layouts/`는 **부분이어도 되고 아예 없어도**
된다 (부모가 바닥을 깐다). 상속하지 않는 테마는 내장 테마와 **같은 조건**을 받는다: 세 파일이
모두 있어야 하고 위의 완전성 검사를 통과해야 한다.

### 병합 규칙 — 한 문장

**이름이 같으면 자식이 이기고, 없던 이름은 부모 순서 뒤에 붙는다.**

부모 순서를 지키는 것이 규칙의 일부다 — 순서가 산출물에 보이기 때문이다 (PPTX `clrScheme`의 슬롯
순서, 웹 CSS 변수 선언 순서, 모르는 `variant` 경고가 나열하는 목록).

| 병합 단위 | 무엇 | 자식이 적으면 |
|---|---|---|
| **이름** | 색·폰트·공간·반경·그림자 토큰, 텍스트 역할, 태그·슬롯 기본값, `variant`, 코드 토큰(`class`), 크롬 선언(`kind`) | 그 **한 벌이 통째로** 자식 것이다 (부모의 나머지 필드는 남지 않는다) |
| **필드** | 이름이 없는 절 — `shape.default`·`image.default`·`connector.default`, 표의 `cell`·`header`·`stripe`, `code.tabSize`, `text.mark` | 적은 필드만 갈리고 나머지는 물려받는다 (빈 문자열·0은 "정하지 않음"이다) |
| **파일** | `layouts/<이름>.css`, `layouts/<이름>.slots.css`, `assets/` 아래 그림 | 그 **파일 하나가** 자식 것이다. 두지 않은 파일은 부모 것이 그대로 실린다 |
| **목록** | `theme.json`의 `layouts` · `backgrounds` | 적으면 그 목록이 이기고, 적지 않으면 부모의 것을 물려받는다 |

역할과 `variant`가 필드 단위가 아닌 이유는 그 둘이 이미 이름을 가진 **한 벌**이라는 것이다:
자식이 `title`을 다시 적었다면 그 역할의 타이포그래피를 자기 것으로 정한 것이고, 부모의 크기가
반쯤 남으면 결과를 예측할 수 있는 사람이 없다. 레이아웃 CSS가 파일 단위인 이유도 같다 — 선언
단위로 섞으면 저자가 두 파일을 동시에 머리에 담아야 자기 배치를 안다.

**지우는 문법은 없다.** 부모가 준 이름을 없앨 수 없다 (`"color": null` 같은 것이 없다): 지움을
열면 자식을 읽어도 어휘가 무엇인지 알 수 없고, 값을 **덮는** 것으로 같은 목적을 이룬다.

색 토큰만 예외적인 제약을 하나 더 받는다: **새 색 이름을 더하려면 남은 `clrScheme` 슬롯이 있어야
한다** — 모든 색 토큰은 슬롯을 가져야 PPTX 테마 색 편집이 따라오고, 그 자리는 열둘로 닫혀 있다
(내장 테마가 열을 쓰므로 남는 자리는 링크 둘이다). 그래서 **자식 테마의 색 작업은 갈아 끼우기**다:
부모의 이름(`accent1`·`surface`)을 다시 적어 값을 바꾼다. 판정은 테마 로딩에서 나므로 진단이
`tokens.json`과 남은 자리를 가리킨다 (PPTX를 조립할 때가 아니다). 반경·공간·그림자·폰트에는 이 상한이 없어 자유롭게 **더한다**.

### 예제

`themes/corp/`가 `minimal`을 물려받아 파일 넷(`theme.json` · 부분 `tokens.json` · 부분
`components.json` · `layouts/title.slots.css` 하나)으로 표지 배치·제목 역할·강조색·크롬 자리를
갈아 끼운다고 하자. 나머지 열 개 레이아웃과 역할 넷, 코드 색 다섯은 **부모 것과 같다** — 적지
않은 것은 전부 물려받는다.

## 설치한 테마 — GitHub에서 받는다

여러 덱이 함께 쓰는 테마는 GitHub 저장소에 두고 **설치**한다. 설치는 명령이 하고, `balpyo.json`은
지금처럼 **이름만** 적는다.

```bash
balpyo theme install github.com/acme/slide-themes@v1.2.0
```

```json
{ "theme": "acme" }
```

- **이름은 테마가 정한다.** 설치는 받은 테마의 `theme.json`에서 `name`을 읽고 그 이름으로 둔다.
  저장소 이름이나 저장소 안의 디렉토리 이름은 상관없다.
- **자리는 `~/.balpyo/themes/<이름>/`이다** (`BALPYO_HOME`이 있으면 그 디렉토리 아래). 구조는 내장
  테마·프로젝트 `themes/`와 같다. 어디서 받았는지(저장소·하위 경로·ref·커밋)는
  `~/.balpyo/themes.json`에 적힌다.
- **빌드는 네트워크를 쓰지 않는다.** 설치한 테마는 이 기계의 디렉토리일 뿐이다. 다른 기계·CI에서
  그 이름이 없으면 빌드가 `unknown theme`로 멈추고, 같은 `balpyo theme install`을 그 기계에서
  친다.
- **`extends`도 같은 어휘다.** 설치한 테마가 내장 테마를, 프로젝트의 테마가 설치한 테마를 상속할
  수 있다.

### 로컬 디렉토리에서 설치한다

GitHub 주소 대신 **경로**를 주면 그 디렉토리의 테마를 홈으로 **복사**한다:

```bash
balpyo theme install ./brand
balpyo theme install ~/work/slide-themes acme-dark
```

- 설치한 테마는 원본과 떨어진 **사본**이다. 원본을 고쳐도 덱은 그대로이고, 고친 것은
  `balpyo theme update`가 기록한 디렉토리에서 다시 복사해 반영한다. 원본 디렉토리를 지워도 빌드는 된다.
- 찾고 고르는 규칙(아래 "한 저장소에 테마가 여럿일 때"), 설치를 막는 것, 전부 되거나 전부 안 되는 것은
  저장소와 같다. 링크는 따라가지 않는다 — 테마 디렉토리 안에 링크가 있으면 설치 에러다.
  `.`으로 시작하는 디렉토리(`.git` 등)는 복사하지 않는다.
- 출처는 `themes.json`에 디렉토리의 절대 경로로 적힌다. 같은 디렉토리에서 다시 설치하면 `--force`
  없이 갈아 끼우고, GitHub에서 설치한 같은 이름은 다른 출처다 (`--force`).
- 고치는 중의 테마를 곧바로 보고 싶다면 설치보다 프로젝트의 `themes/<이름>/`이 맞는 자리다.

### 한 저장소에 테마가 여럿일 때

설치는 받은 저장소(주소에 하위 경로가 있으면 그 아래)에서 **`theme.json`이 있는 디렉토리를 모두**
찾는다. `.`으로 시작하는 디렉토리(`.github` 등)는 보지 않고, 테마 디렉토리 안으로는 더 내려가지
않는다.

```text
acme/slide-themes
├── themes/light/theme.json     { "name": "acme", "extends": "minimal" }
├── themes/dark/theme.json      { "name": "acme-dark", "extends": "acme" }
└── print/theme.json            { "name": "acme-print", "extends": "minimal" }
```

| 명령 | 설치되는 것 |
|---|---|
| `balpyo theme install github.com/acme/slide-themes` | 셋 다 |
| `balpyo theme install github.com/acme/slide-themes acme-dark` | `acme-dark`와, 그것이 상속하는 같은 저장소의 `acme` |
| `balpyo theme install github.com/acme/slide-themes/print` | `acme-print` (하위 경로 아래만 본다) |

**설치는 전부 되거나 전부 안 된다.** 받은 테마를 빌드와 같은 로더로 로드해 보고(필수 토큰·
`extends` 체인·이름), 하나라도 실패하면 아무것도 바꾸지 않는다. 설치를 막는 것:

| 상황 | 문구(요지) |
|---|---|
| 내장 테마와 같은 이름 | `…: theme "minimal" has the same name as a built-in theme and cannot be installed — pick the themes to install by name and run again` |
| 저장소 안에 같은 이름이 둘 | `… has more than one theme "acme": v1, v2 — append a subpath to the address to pick one` |
| 다른 출처에서 이미 설치한 이름 | `theme "acme" is already installed from github.com/… — use --force to replace it` |
| 테마 디렉토리 안의 심볼릭 링크 | `… contains link … — themes contain regular files only` |
| 로드되지 않는 테마 | `…: theme "acme" cannot be loaded, so nothing was installed: …` |

설치한 테마의 레이아웃 CSS 경고(외관 프로퍼티)는 **설치할 때** 나오고 빌드에서는 나오지 않는다 —
덱 저자가 고칠 수 없는 파일이 `--strict` 빌드를 막지 않게 하기 위해서다. 명령의 전체 표면
(`list`·`info`·`remove`·`update`, 받는 주소 모양)은 [CLI](cli.md#balpyo-theme)에 있다.

### 이름이 겹칠 때

프로젝트의 `themes/acme/`와 설치한 `acme`가 함께 있어도 **그 이름을 요청하지 않는 덱은 빌드된다**.
요청하면(`theme`·`--theme`·`extends` 체인) 어느 쪽인지 답할 수 없으므로 에러다:

```text
theme "acme" exists both in the project's themes/acme/ and among installed themes — rename the local theme or remove the installation with balpyo theme remove acme
```

`balpyo theme list`가 겹치는 이름을 표시한다. 포맷이 데이터 전용(JSON + CSS + 그림)이라 받은
테마가 코드를 실행하는 길은 없다.

## 빌드가 멈추거나 경고하는 조건

| 상황 | 결과 | 문구(요지) |
|---|---|---|
| `theme` 이름이 어휘(내장 ∪ 로컬 ∪ 설치)에 없다 | **에러** | `unknown theme "inkk" — not a built-in theme, a theme in the project's themes/, or an installed theme (available themes: corp, editorial, ink, minimal)` — 설정 로드와 값 층 둘이 같은 판정을 하고, 목록에 **이 프로젝트의 로컬 테마와 설치한 테마**가 함께 들어간다 |
| 요청한 이름이 로컬 테마이면서 설치한 테마다 | **에러** (그 이름을 요청할 때만) | `theme "acme" exists both in the project's themes/acme/ and among installed themes — …` |
| 요청한 이름의 설치한 테마가 내장 이름이다 (손으로 넣은 디렉토리) | **에러** (그 이름을 요청할 때만) | `installed theme "ink" has the same name as a built-in theme — remove the installation with balpyo theme remove ink` |
| `theme.json`에 `name`이 없다 | **에러** | `theme.json has no name` |
| `themes/<이름>/theme.json`의 `name`이 디렉토리 이름과 다르다 | **에러** | `themes/corp/theme.json has name "corp-dark" — it must match the directory name "corp"` |
| 로컬 테마가 내장 이름을 가린다 | **에러** (요청한 테마와 무관하게) | `themes/minimal/ has the same name as the built-in theme "minimal" — rename the local theme` |
| `extends`가 없는 이름을 가리킨다 | **에러** | `theme "corp" extends theme "minimalx", which does not exist (available themes: …)` |
| `extends`가 순환한다 (자기 자신 포함) | **에러** | 체인을 그대로 적는다 — `theme.json extends forms a cycle: a → b → a` |
| `chrome`의 `kind`·`align`·`role`이 어휘 밖이거나 비었다 | **에러** | 쓸 수 있는 값을 함께 나열한다 |
| `chrome`에 같은 `kind`가 두 번 | **에러** | `kind footer is declared twice` |
| `logo`에 `image`·`height`가 없거나, 글자 크롬에 `image`·`height`·`alt`를 썼다 | **에러** | 빠지거나 잘못 쓴 키를 말한다 |
| `backgrounds`의 선언에 `fill`도 `image`도 없거나 둘 다 있다 · 레이아웃이 두 선언에 나온다 · 기본 배경이 둘이다 · 레이아웃이 이 테마의 것이 아니다 | **에러** | 선언의 자리(`backgrounds[1]`)와 이유를 말한다 |
| 배경 · 로고의 그림이 `assets/` 밖이거나 패키지에 없거나 그림이 아니다 | **에러** | `<image>`의 파일과 같은 판정 문구다 |
| `skipLayouts`의 이름이 이 테마의 레이아웃이 아니다 | **에러** | `skipLayouts: "cover" is not a layout of this theme` |
| `layouts`에 선언했는데 `layouts/<이름>.css`가 없다 | **에러** | `theme.json layouts lists … but layouts/….css is missing` |
| `layouts/<이름>.css`가 있는데 선언에 없다 | **에러** | `layouts/….css exists but is not declared in theme.json layouts` |
| `tokens.json`에 필수 토큰 이름이 없다 | **에러** | `tokens.json has no color token "accent6" (required: fg, bg, muted, surface, accent1, …)` |
| 색 토큰에 `clrScheme` 슬롯이 없다 | **에러** | 남은 자리를 나열한다 — `… has no clrScheme slot … (free slots: hlink, folHlink)`. 자리가 다 찼으면 `no free slots — replace the value of an existing name instead of adding a new color name` |
| 두 색 토큰이 같은 슬롯을 쓴다 | **에러** | `tokens.json clrScheme slot "accent1" is shared by color tokens "accent1" and "brand" — cannot tell which color owns the slot` |
| 색 토큰의 `ref`가 비었다 | **에러** | `tokens.json color token "accent1" has no schemeClr reference (ref) — …` |
| 폰트 토큰의 `scheme`이 `major`·`minor`·생략 밖이다 · 두 폰트가 같은 자리를 쓴다 · `major`나 `minor` 자리가 비었다 | **에러** | `… has an unknown fontScheme slot "middle" …` · `tokens.json font tokens "heading" and "body" both take the fontScheme slot "major" — each slot holds exactly one font` · `tokens.json has no font token in the fontScheme slot "minor" — …` |
| 그림자 토큰의 `color`가 이 테마의 색 토큰이 아니다 | **에러** | `tokens.json shadow token "soft" color "nope" is not a color token of this theme (this theme: …)` |
| `components.json`의 `text.roles`에 역할 5종 중 하나가 없다 | **에러** | `components.json text.roles has no role "code" (required: title, heading, body, caption, code)` |
| 역할의 `size`·`lineHeight`가 0 이하이거나 `align`이 없다 | **에러** | `roles are the bottom of the cascade and must be complete` |
| 역할의 `align`이 `start`·`center`·`end`·`justify` 밖이다 | **에러** | `role "title" has an unknown align "middle" (available aligns: start, center, end, justify)` |
| 역할의 `font`·`color`가 이 테마의 토큰이 아니다 | **에러** | 이 테마가 가진 이름을 함께 나열한다 |
| 컴포넌트 기본값·variant·이미지·표의 `color`·`fill`·`border` 색이 이 테마의 색 토큰 참조가 아니다 (`#RRGGBB` 리터럴, 리터럴 스톱이 섞인 그라디언트) | **에러** | 자리와 문제의 조각을 가리키고 이 테마의 색 토큰을 나열한다 — 그 variant를 쓴 슬라이드가 없어도 로딩에서 멈춘다 |
| 역할·컴포넌트 기본값·variant의 `tracking`이 −0.1 ~ 0.5 밖이거나 `case`가 `upper`·`none`이 아니다 | **에러** | `… tracking 0.8 is out of range …` · `… case "lower" is unknown (available values: upper, none)` |
| `text.mark.highlight`가 그라디언트다 | **에러** | `text.mark highlight "gradient(…)" cannot be a gradient — a highlight is one color (PPTX a:highlight holds a single color)` |
| `tokens.json`·`components.json`에 모르는 키가 있다 (`$comment*` 밖) | **에러** | 그 파일의 줄:칸과 JSON 안의 경로를 가리킨다 — `themes/corp/components.json:4:24: unknown key "lineHeight" at text.variants["stat"].lineHeight`. 이름이 있는 배열 원소는 이름으로, 없으면 순번(`componentDefaults[1]`)으로 부른다 |
| 테마 JSON 파일의 값 타입이 틀렸거나 문법이 깨졌다 | **에러** | 같은 방식으로 파일 · 줄:칸 · 경로를 가리킨다 — `shape.variants["card"].fill must be a string, not number` · `invalid JSON: …` |
| `code.tabSize`가 1 미만이다 | **에러** | `tabSize must be at least 1` |
| `code.tokens`에 모르는 종류가 있다 | **에러** | `unknown code token class "kw" (available types: keyword, name, attr, string, number, comment, punct)` |
| `code.tokens`에 같은 종류가 두 번 | **에러** | `code token class "keyword" is listed twice` |
| 코드 토큰 색이 리터럴·색 함수다 | **에러** | `the color of code token … must be a color token name … — a literal color breaks the reference` |
| 코드 토큰 색이 이 테마에 없는 토큰이다 | **에러** | 쓸 수 있는 색 토큰 이름을 함께 나열한다 |
| 로컬 테마의 레이아웃 CSS에 배치 아닌 프로퍼티(색·배경·글꼴 등)가 있다 | **경고** (내장 테마는 테스트가 전수로 막는다) | 그 파일(`themes/<이름>/layouts/….css`)을 가리킨다 — 외관은 `components.json`의 몫이고 그 프로퍼티는 PPTX에 도달하지 않는다. `extends` 체인에서는 병합 결과에 남은 파일만 본다 |
| 로컬 테마의 레이아웃 CSS 선언에 콜론·이름·값이 없다 | **에러** | `layout CSS declaration "…" has no colon` |
| 설정이 켠 크롬을 테마가 선언하지 않았다 | **경고** + 미탑재 | `balpyo.json`을 가리킨다 — 고칠 자리가 거기다 |

에러는 테마 로딩에서 나므로 **어느 슬라이드도 렌더되기 전에** 멈춘다. 크롬 선언은 토큰·컴포넌트
표보다 **먼저** 검사한다 — 선언만으로 판정할 수 있는 값 어휘이고, 먼저 멈추면 "테마의 오타"와
"토큰 표의 문제"가 한 메시지에 섞이지 않는다. 경고는 [`--strict`](cli.md)로 에러로 승격할 수 있다.

## 예제

같은 덱을 `--theme ink`로 내면 같은 어휘로 쓴 슬라이드가 다른 테마에서 어떻게 달라지는지 보인다 —
견주는 순서는 [테마 고르기·만들기](../guides/theming.md)에 있다.

크롬을 켠 4장 덱(표지 · 본문 · 구역 표지 · 본문)이면 표지(`title`)와 구역 표지(`section-divider`)는
제외되고, 흐름 모드 두 장이 푸터와 번호 2·4를 받는다.
