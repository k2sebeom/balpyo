# 테마 고르기·만들기

테마는 **덱의 디자인 시스템 하나**다. 저자가 쓴 슬라이드에는 색·크기·자리가 거의 없고
(`role="title"`·`color="accent1"`처럼 **뜻**을 쓴다), 그 뜻을 값으로 바꾸는 표가 테마다. 그래서
테마를 갈면 **원고를 한 줄도 고치지 않고** 같은 덱이 다른 덱처럼 보인다.

이 가이드는 고르는 방법과 세 내장 테마의 성격, 그리고 새 테마를 만들 때 지켜야 하는 계약을
적는다. 파일 포맷 자체는 [테마 패키지](../reference/theme.md)에 있다.

## 고르는 법 — 같은 덱을 두 번 내서 견준다

테마는 스크린샷으로 고르는 것이 아니라 **자기 덱으로** 고른다. 글줄 길이·제목의 단어 수·표의
칸 수가 테마의 인상을 결정하기 때문이다.

```bash
# 한 덱을 두 테마로 검수 갤러리까지 낸다
balpyo preview --theme editorial --gallery preview-editorial
balpyo preview --theme ink       --gallery preview-ink
```

`--theme`는 `build`·`check`·`watch`·`preview`가 모두 받고 `balpyo.json`의 값을 **덮어쓴다**
(우선순위는 CLI 플래그 > `balpyo.json` > 기본값). 고른 뒤에 파일에 적는다:

```json
{ "theme": "ink" }
```

없는 이름은 렌더 전에 멈추고 쓸 수 있는 이름을 나열한다 — 조용히 기본 테마로 그리지 않는다
([ADR 0052](../decisions/0052-theme-selection-is-a-deck-request-and-there-is-no-fallback-theme.md)).

## 내장 테마 셋

| | `minimal` (기본값) | `editorial` | `ink` |
|---|---|---|---|
| 목소리 | 중립적인 업무 문서 | 인쇄 지면 | 어두운 방 |
| 바탕 / 글자 | 흰색 `FFFFFF` / 거의 검정 | 종이색 `FAF8F4` / 먹빛 `1C1A17` | 먹빛 `15171B` / 밝은 회백 `F2F4F7` |
| 제목 폰트 | 산세리프 (Noto Sans) | **세리프** (Noto Serif) | 산세리프 |
| 제목 크기·굵기 | 56px / 700 | 60px / 400 (가볍고 크다) | 52px / 600 |
| 표지 | 제목 묶음이 세로 **가운데**, 가로도 가운데 | 제목이 **아래**에 내려앉고 위를 비운다 | 세로 가운데, 가로는 **왼쪽** |
| 구역 표지 | 세로 가운데 · 왼쪽 | 아래로 내려앉는다 | 가운데 (유일하게 가로도 가운데) |
| 어울리는 자리 | 대부분 — 판단이 서지 않으면 이것이다 | 글이 많은 덱, 발표 후 읽히는 문서 | 무대·어두운 방·코드가 많은 덱 |

세 테마의 강조색은 서로 다르지만 **같은 세 수치**를 만족한다 (바탕 대비 4.5:1 이상 · 색상 간격
25° 이상 · 휘도 띠 1.25 이하 — [ADR 0050](../decisions/0050-six-accents-are-a-measured-palette-not-a-ranking.md)).
어두운 바탕에서 수치가 커지는 것(`ink`는 5.93~6.00)은 규칙이 뒤집히지 않았다는 뜻이다.

### 갈리는 것과 갈리지 않는 것

테마를 갈았을 때 덱이 **깨지지 않는다**는 것이 계약이고, 그 계약이 다음 목록이다. 내장 테마
전수로 도는 테스트가 지킨다.

| 갈리지 않는다 (어휘·리듬) | 갈린다 (값·짜임) |
|---|---|
| 토큰 **이름** — `fg`·`bg`·`muted`·`surface`·`accent1`~`6`, space 6종, radius 3종, `soft` | 그 이름의 색·수치 |
| 텍스트 역할 5종과 그 이름 | 제목·본문 외의 크기·굵기·폰트 |
| `variant` 어휘 — 텍스트 `kicker`·`lead`, 도형 `card`~`badge`, 이미지 `framed` | 그 번들이 담는 값 |
| 표준 레이아웃 11종 전부 (슬롯 이름까지) | 슬롯이 어디에 앉는가 (좌표) |
| **본문 24px · 줄높이 1.5** — 스테이지 720px이 본문 20줄이라는 제품의 사실 | 제목 위계 |
| 코드 토큰이 가리키는 색 토큰 이름 | 그래서 코드 색은 팔레트를 따라 자동으로 바뀐다 |

`variant` 어휘를 공유하는 것이 계약인 이유는 이식성이다: `variant="stat-card"`가 테마를 간
순간 경고로 바뀌면 "테마를 갈면 다르게 보인다"가 "테마를 갈면 덱이 깨진다"가 된다.

### 테마를 간 뒤에 볼 것

값이 갈리므로 **배치 판정도 갈린다** — 제목이 커지면 줄이 늘고, 늘어난 줄은 넘친다. 그래서
테마를 바꾼 덱은 한 번 검사한다:

```bash
balpyo check --theme ink --check-layout --strict
```

가운데로 모이는 슬롯(표지·`stat`·마무리)의 한 줄에 **한글 사이 문장부호**가 있으면 PPTX에서
접힐 수 있다 — 그 자리의 한 줄은 짧게 쓴다
([계약 밖의 차이](known-differences.md)의 "가운데 정렬 슬롯은 여유가 없다").

## 자기 테마 만들기 — 프로젝트 안에서

자기 테마는 **프로젝트 안에** 둔다. 이 저장소를 고치거나 다시 빌드하지 않고, 디렉토리 이름으로
`balpyo.json`이 가리킨다 ([ADR 0054](../decisions/0054-an-external-theme-is-a-name-inside-the-project-and-extends-overrides-by-name.md)).

```text
my-deck/
├── balpyo.json       # { "theme": "corp" }
├── slides/
└── themes/
    └── corp/         # ← 디렉토리 이름이 테마 이름이다 (theme.json의 name과 같아야 한다)
        ├── theme.json
        ├── tokens.json         # 부분이어도 된다 (상속할 때)
        ├── components.json     #   "
        └── layouts/            #   갈아 끼울 파일만 둔다
```

가장 빠른 길은 **가까운 내장 테마를 물려받는 것**이다. 파일 넷을 다 채우지 않고, 바꿀 것만 적는다.

`themes/corp/theme.json`:

```json
{
  "name": "corp",
  "extends": "minimal",
  "chrome": [
    { "kind": "footer", "align": "center", "role": "caption", "skipLayouts": ["title"] }
  ]
}
```

`themes/corp/tokens.json` — 색은 갈아 끼우고, 나머지는 더한다:

```json
{
  "color":  [{ "name": "accent1", "rgb": "7C3AED", "slot": "accent1", "ref": "accent1" }],
  "font":   [{ "name": "heading", "latin": "Noto Serif", "ea": "Noto Serif CJK KR",
               "generic": "serif", "scheme": "major" }],
  "radius": [{ "name": "xl", "px": 36 }]
}
```

병합 규칙은 한 문장이다 — **이름이 같으면 자식이 이기고, 없던 이름은 부모 순서 뒤에 붙는다**
(자세히: [테마 패키지](../reference/theme.md)). 세 줄만 외우면 된다.

- **색은 갈아 끼운다.** 새 색 **이름**을 더하려면 남은 `clrScheme` 슬롯이 있어야 하고, 내장
  테마가 열둘 중 열을 쓴다
  ([ADR 0055](../decisions/0055-a-child-theme-recolors-by-name-because-color-tokens-are-bounded-by-the-clrscheme.md)).
  반경·공간·그림자·폰트에는 그 상한이 없다.
- **역할과 `variant`는 한 벌 통째로** 갈린다 (`title`을 다시 적었으면 그 역할의 크기·색·정렬을
  전부 자기가 정한 것이다). `shape.default` 같은 이름 없는 절만 필드 단위다.
- **레이아웃 CSS는 파일 단위**다. `layouts/title.slots.css` 하나만 두면 표지 배치만 갈리고 나머지
  열은 부모 것이 그대로 실린다. `layouts` 선언도 물려받으므로 11종을 다시 적지 않는다.

끝까지 한 번 돌려 보는 증인이 저장소에 있다 — `testdata/fixtures/15_local_theme`가 `themes/corp/`
파일 넷으로 `minimal`을 물려받는 3장 덱이다.

### 내장 테마를 하나 더 만들 때

제품에 실리는 테마는 `core/themes/<이름>/`에 산다 — 디렉토리를 더하면 `BuiltinThemes()`가 그것을
돌려주므로 `--theme`과 `init --theme`의 어휘가 함께 열린다 (`init`은 프로젝트를 **만드는** 명령
이라 로컬 테마를 받지 않는다). 내장 테마는 `extends`를 쓰지 않는다: 셋이 서로 독립이어야 "무엇이
테마의 것인가"를 비교로 답할 수 있고, 아래 전수 테스트가 셋 모두에 같은 조건을 요구한다. 그래서
파일 넷이 전부 필요하다 (색 10 · 폰트 3 · space 6 · radius 3 · shadow 1 · 역할 5종 · 레이아웃
11개 CSS + 슬롯 영역 10개 CSS).

### 로더가 요구하는 것 (없으면 빌드가 멈춘다)

폴백 테마가 없으므로 빠진 값을 메워 줄 곳이 없다. 그래서 로드 시점에 멈춘다:

- 필수 **토큰 이름** 전부와 **텍스트 역할 5종** 전부. 역할의 `size`·`lineHeight`는 0보다 커야
  하고 `align`이 있어야 한다.
- 역할이 가리키는 `font`·`color`는 **그 테마에 있는** 토큰 이름이어야 한다 (리터럴 색은 거부다 —
  PPTX의 테마 색 편집을 끊는다).
- `theme.json`의 `layouts` 선언과 `layouts/*.css` 파일 집합이 **양방향으로** 같아야 한다.
- `components.json`의 모르는 키는 에러다 (닫힌 어휘).

문구와 전체 목록은 [테마 패키지](../reference/theme.md)의 "빌드가 멈추거나 경고하는 조건"에 있다.

### 테스트가 요구하는 것 (내장 테마 전수)

- 표준 레이아웃 **11종 전부** 구현 — 빠뜨린 레이아웃을 다른 테마로 폴백하지 않는다.
- 슬롯이 있는 레이아웃 10종의 **슬롯 영역 CSS 전수**, 그리고 각 파일이 `display: grid`와 그
  레이아웃의 슬롯 이름 전부를 스스로 가질 것.
- 레이아웃 CSS는 **배치 전용** — 색·글꼴 선언은 PPTX 마스터에 도달하지 않는다
  ([ADR 0014](../decisions/0014-style-escape-hatch-boundary.md)).
- 강조색 여섯이 세 수치를 만족할 것 (위 팔레트 규칙).
- `variant` 어휘가 다른 내장 테마와 **같을** 것 (이름과 순서까지).
- 본문 역할이 **24px · 1.5**일 것.

### 손으로 챙겨야 하는 것

로더도 테스트도 잡아 주지 않지만 산출물에서 드러나는 것들이다.

| 규칙 | 어기면 |
|---|---|
| `mono` 토큰의 `ea`는 **고정폭 CJK 면**이어야 한다 (`Noto Sans Mono CJK KR`) | 코드 블록의 한글 주석 한 줄이 두 엔진에서 다른 폭이 된다 (측정 −7.9%) |
| 슬롯 영역에서 **왼쪽 정렬은 선언하지 않는다** — 가운데만 `justify-items: center` | `justify-items: start`는 그림을 바꾸지 않고 여유 폭만 0으로 만들어 PPTX에서 줄이 접힌다 ([ADR 0053](../decisions/0053-a-theme-takes-the-horizontal-channel-only-to-center.md)) |
| 색 토큰 12자리 `clrScheme` 슬롯은 **밝기가 아니라 관례**다 — 어두운 테마도 `fg`를 `dk1`, `bg`를 `lt1`에 앉힌다 | PowerPoint의 테마 색 편집이 어두운 덱에서 뒤집힌다 |
| 어두운 바탕에서는 면과 선을 `muted`가 아니라 `surface`로 (`ink`의 표 셀 테두리가 그렇다) | 검은 바탕 위의 회색 선이 글자보다 튄다 |

세 파일을 다 채웠으면 픽스처로 끝까지 한 번 돌린다 — `testdata/fixtures/14_theme`가 `ink`로
렌더되는 3장 덱이고, 같은 방법으로 새 테마의 골든을 만들면 무엇이 테마의 것인지 그 차이가
곧 대답이다.

## 아직 아닌 것

원격 테마(`balpyo theme get`), 테마를 가리키는 **경로** 문법, 슬라이드마다 다른 테마, 시스템 다크
모드 자동 전환, 테마가 폰트 파일을 번들하는 것은 열지 않았다. 팀에서 테마를 나누려면
`themes/<이름>/`을 복사하거나 git 서브모듈로 둔다 — 포맷이 데이터 전용(JSON + CSS)이라 받은
테마가 코드를 실행하는 길은 없다. 스테이지 여백 64px과 기본 간격 24px은 테마가 아니라 **엔진**의
값이다.
