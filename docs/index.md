# balpyo

> **SOML**(Slide Oriented Markup Language)로 슬라이드를 쓰면,
> **balpyo**가 테마를 입혀 **웹 프레젠테이션**과 **편집 가능한 네이티브 PPTX**로 렌더링한다.

Hugo가 Markdown 콘텐츠를 테마로 렌더링하는 정적 사이트 생성기라면,
balpyo는 SOML 슬라이드를 테마로 렌더링하는 **프레젠테이션 생성기**다.

---

## 왜 balpyo인가

텍스트로 슬라이드를 만드는 도구는 이미 여럿 있지만, "**슬라이드 전용 언어로 선언하고,
흐름 레이아웃으로 배치하되, 산출물은 웹과 편집 가능한 네이티브 PPTX 둘 다**"인 도구는 없다.

| 도구 | 한계 |
|---|---|
| Marp / Slidev | Markdown은 문서 마크업이라 슬라이드 어휘가 없다. PPTX 출력은 이미지 기반 |
| reveal.js | 웹 전용. 테마·레이아웃 시스템 빈약 |
| python-pptx / pptxgenjs | 모든 요소를 EMU 절대 좌표로 수동 배치. 흐름 레이아웃 부재 |
| PowerPoint / Keynote | GUI 기반. 버전 관리·재사용·자동화 불가 |

balpyo에서 저자는 좌표를 만지지 않는다. `<grid cols="4">` 한 줄이 카드 네 장의 자리를 정하고,
PPTX에서도 그 자리에 진짜 도형과 텍스트 박스가 놓인다.

---

## 두 개의 이름

| 이름 | 정체 | 역할 |
|---|---|---|
| **SOML** (Slide Oriented Markup Language) | 마크업 언어 (`.slide` 파일) | 슬라이드의 구조·내용·배치·외관을 선언 |
| **balpyo** | 렌더 엔진 · 뷰어 · CLI | SOML을 파싱해 테마를 입히고 타깃별로 렌더링 |

---

## 빠른 시작

```bash
git clone https://github.com/k2sebeom/balpyo && cd balpyo
go build -o balpyo ./cmd/balpyo        # 바이너리 하나 (런타임 의존 없음)

./balpyo init my-talk --author "이세범"
./balpyo watch --project my-talk                     # http://127.0.0.1:4600 라이브 프리뷰
./balpyo build --project my-talk --target html,pptx  # → my-talk/dist/index.html + .pptx
```

PPTX 타깃은 시스템 브라우저(Chromium/Chrome)로 배치를 잰다. 설치부터 발표까지의 순서는
[시작하기](guides/getting-started.md)에 있다.

PowerPoint에서 연 `.pptx`의 텍스트 박스는 진짜 텍스트 박스이고, 도형은 진짜 도형이며,
**색은 테마 색 참조를 유지**해 PowerPoint 안에서의 테마 색 편집을 따라간다.

### 프로젝트 구조

```text
my-talk/
├── balpyo.json              # 덱 설정: 제목, 테마, 종횡비, 크롬
├── slides/                  # 슬라이드 = 파일. 파일명 접두 번호가 순서
│   ├── 001_title.slide
│   ├── 010_architecture.slide
│   └── 900_end.slide
├── assets/                  # 이미지
├── themes/                  # (선택) 자기 테마. 내장 테마를 쓰면 불필요
└── dist/                    # 산출물
```

- **슬라이드 하나 = 파일 하나.** 순서는 파일명 정렬(`001_`, `010_`, `900_`)로 결정된다.
  번호 사이를 띄워 두면 중간 삽입이 파일 하나 추가로 끝난다.
- **하위 디렉토리는 챕터**다 (`slides/02_deep-dive/010_...`).
- **`_`로 시작하는 파일은 무시**된다 (초안, 보류 슬라이드).
- 슬라이드 하나가 파일 하나이므로 Git diff·코드 리뷰·병합·재배치가 자연스럽다.

### `balpyo.json`

```json
{
  "title": "2026 아키텍처 리뷰",
  "author": "이세범",
  "theme": "minimal",
  "aspect": "16:9",
  "lang": "ko",
  "outDir": "dist",
  "footer": "2026 Q3 · 내부용",
  "pageNumber": true
}
```

우선순위: **CLI 플래그 > `balpyo.json` > 기본값.** `aspect`는 `16:9`(기본) 또는 `4:3`이다.
`lang`은 덱의 **자연어**다 (`index.html`의 `lang`, `ko`이면 어절 줄바꿈) — 코드 블록의 언어는
[`<pre lang>`](reference/soml/text-subset.md#코드-블록-lang과-탭)이 따로 정한다.

`slidesDir`(기본 `slides`)은 슬라이드를 모으는 디렉토리를 바꾼다. 프로젝트 루트 기준의 상대
경로여야 하고, 절대 경로나 `../`로 프로젝트 밖을 가리키면
`balpyo.json: slidesDir: "../shared/slides" must be a relative path inside the project`로 멈춘다.

`lang`은 덱 언어이고 기본은 `en`이다. 값은 BCP 47 언어 태그(`ko`·`ko-KR`·`en-US`·`ja`…)여야
한다 — 웹의 `<html lang>`이 되어 언어별 줄바꿈 규칙을 고르므로(한국어는
[어절에서 접는다](reference/soml/text.md)), `korean` 같은 이름은 빌드가
`balpyo.json: lang: not a BCP 47 language tag: "korean" (for example "en", "ko", "ja")`로
멈춘다. 대소문자·밑줄·세 글자 코드는 정규형으로
읽는다 (`ko_KR` → `ko-KR`, `kor` → `ko`).

`footer`·`pageNumber`는 [덱 크롬](reference/theme.md#덱-크롬-chrome)을 켠다 — 둘 다 기본은
꺼짐이고, 그 글자가 어디에 어떤 크기로 앉는지는 테마가 정한다.

---

## 어떻게 동작하는가

```text
slides/*.slide ─파싱─▶ 덱 모델 ─외관 해석·검증─▶ ┬─▶ 웹 렌더러  → index.html
                                    ▲          └─▶ PPTX 렌더러 → .pptx
                                  테마                ▲
                       (토큰 · 컴포넌트 외관 · 레이아웃)   브라우저가 잰 배치
```

선언은 두 채널로 나뉜다.

```text
배치(layout)      → 레이아웃 태그 (flex/grid 시맨틱)
                    … 브라우저가 계산하고, 그 박스가 그대로 PPTX 좌표가 된다.

외관(appearance)  → 외관 속성 + variant (타깃 중립)
                    … 편집 가능한 PPTX 속성으로 남아야 하므로 양 타깃으로 컴파일된다.
```

**번역할 필요가 없는 것(배치)은 브라우저에 맡기고, 번역되어 살아남아야 하는 것(외관)만
속성으로 선언한다.** 그래서 두 산출물은 같은 요소가 같은 종류의 객체로, 같은 자리에, 같은
외관으로, 같은 순서로 나온다. 다만 박스 **안에서** 줄이 어디서 접히는지는 각 엔진의 몫이다 —
PPTX 텍스트 박스는 `normAutofit`을 켜서 PowerPoint가 줄을 다르게 접어 넘치면 글자를 줄여
박스 안에 둔다 ([계약 밖의 차이](guides/known-differences.md)).

슬라이드는 두 가지로 쓴다.

- **레이아웃과 슬롯** — `<slide layout="title">`에 `slot="title"`처럼 자리 이름만 적는다.
  배치는 테마가 소유하므로 테마를 바꾸면 배치까지 달라진다. 표지·불릿·2단 비교·인용 같은
  대부분의 슬라이드가 여기 속한다.
- **배치와 컴포넌트** — `<row>`/`<col>`/`<grid>`/`<stack>`으로 저자가 배치를 소유하고,
  외관은 테마 기본값과 variant를 따른다. 다이어그램·자유 구성 슬라이드가 여기 속한다.

---

## SOML 맛보기

`.slide` 파일 하나는 슬라이드 하나이며, 루트 요소는 `<slide>`다. 별도 front matter는 없다 —
슬라이드 메타데이터는 루트 요소의 속성으로 선언한다.

### 레이아웃과 슬롯

```html
<!-- slides/001_title.slide -->
<slide layout="title">
  <text slot="kicker" variant="kicker">Platform Team</text>
  <h1 slot="title">2026 아키텍처 리뷰</h1>
  <text slot="subtitle" role="heading" color="muted">상반기 회고와 하반기 계획</text>
  <note>인사 30초. 청중에 인프라팀 포함되어 있음.</note>
</slide>
```

표준 레이아웃은 11종(`content`·`title`·`section-divider`·`two-column`·`image-full`·
`image-text`·`quote`·`code`·`stat`·`canvas`·`end`)이고, 각자 이름 있는 슬롯을 갖는다 —
슬롯은 웹에서 grid 영역이, PPTX에서 진짜 플레이스홀더가 된다. 목록은
[레이아웃과 슬롯](reference/layouts.md)에 있다.

### 배치와 컴포넌트

```html
<slide>
  <h2>서비스 아키텍처</h2>
  <row gap="xl" align="center" justify="center">
    <shape type="rounded-rect"><text>Web</text></shape>
    <shape type="arrow-right" />
    <shape type="rounded-rect" fill="accent1" color="bg"><text>API Gateway</text></shape>
    <shape type="arrow-right" />
    <shape type="cylinder"><text>Aurora</text></shape>
  </row>
  <note>이 다이어그램은 3분 정도 설명할 것</note>
</slide>
```

레이아웃 태그는 **CSS flex/grid 시맨틱의 이름 바꾸기 층**이다. 새 레이아웃 모델이 아니다.
`gap="xl"`처럼 **공간 값이 토큰 어휘에 들어온다** — 배치조차 테마의 리듬을 따라간다.

| 분류 | 태그 |
|---|---|
| 문서 | `<slide>`, `<note>` |
| 레이아웃 (배치) | `<row>`, `<col>`, `<grid>`, `<stack>`, `<spacer>` |
| 컴포넌트 | `<text>`, `<shape>`, `<image>`, `<table>`, `<chart>` |
| 텍스트 (HTML 서브셋) | `h1`~`h3`, `p`, `ul`/`ol`/`li`, `blockquote`, `pre`, `code`, `strong`, `em`, `a`, `br`, `span` |

!!! important "닫힌 어휘"
    미지의 태그와 속성은 무시되지 않고 `line:col`을 가진 **진단**이 된다. 오타를 조용한 시각
    버그가 아니라 빌드 실패로 만든다.

태그별 속성과 두 타깃의 렌더 결과는 [레퍼런스](reference/index.md)에 있다.

### 외관 속성

외관은 CSS가 아니라 **컴포넌트의 프레젠테이션 속성**으로 선언한다. SVG가
`<rect fill="…" stroke-width="…">`로 외관을 속성 선언하는 것과 같은 원리다.

- **색 값은 토큰명 우선.** 토큰명으로 선언된 색은 PPTX에서 `schemeClr` 참조로 보존되어,
  **내보낸 파일이 PowerPoint 안에서도 테마 색 편집을 따라간다.**
- **해석 순서**: 인라인 속성 > `variant` > 테마의 컴포넌트 기본값 > 텍스트 역할/토큰. 결과는
  **타깃 무관**이다.

전체 목록과 타깃별 매핑은 [외관 속성](reference/appearance.md)에 있다.

---

## 테마 — 디자인 시스템으로서의 테마

테마는 CSS 파일이 아니다. 다음을 소유하는 **디자인 시스템 패키지**다.

1. **디자인 토큰** — 색·폰트·공간·반경·그림자. CSS 변수와 PPTX 테마 파트로 컴파일된다.
2. **컴포넌트 외관** — 컴포넌트별 기본값 + 이름 붙은 variant + 텍스트 역할 5종.
3. **레이아웃 구현** — 표준 레이아웃 11종의 배치와 슬롯 영역 (**배치 전용 CSS**).
4. **크롬(chrome)** — 페이지 번호·푸터가 앉는 자리.

```text
my-theme/
├── theme.json          # 매니페스트: 이름, extends, 레이아웃 선언, 크롬 선언
├── tokens.json         # 디자인 토큰
├── components.json     # 컴포넌트 기본 외관 + variant + 텍스트 역할
└── layouts/            # 레이아웃별 배치와 슬롯 영역 — 배치 전용 CSS
```

**공식 테마는 바이너리에 내장된다** — 내장 셋(`minimal`·`editorial`·`ink`)이다. `balpyo init`
직후 아무 설치 없이 결과가 나온다. 자기 테마는 프로젝트의 `themes/<이름>/`에 두고 `extends`로
내장 테마를 물려받는다 ([테마 패키지](reference/theme.md) · [테마 고르기·만들기](guides/theming.md)).

---

## 두 산출물

### 웹 (기준 타깃)

웹은 **기준(reference) 타깃**이다. 산출물은 CSS와 뷰어 런타임을 품은 `index.html` 하나다 —
**파일로 열어도 동작해 오프라인 발표가 가능하다.**

- 키보드/터치 네비게이션, `reveal` 순차 등장, URL 해시 딥링크
- 화면 크기 무관 스케일링, 오버뷰 그리드, 발표자 뷰(노트 + 타이머 + 다음 슬라이드)
- 브라우저 인쇄로 PDF — 슬라이드 한 장이 한 페이지

### PPTX — 편집 가능한 네이티브 파일

- 텍스트는 텍스트 박스, 도형은 프리셋 도형, 표는 표, 그림은 그림이다 — 이미지가 아니다.
- 좌표는 브라우저가 잰 박스에서 오므로 두 타깃의 배치가 같다.
- 토큰 색은 `schemeClr`로, 슬롯은 `p:ph` 플레이스홀더로 남아 PowerPoint의 테마 편집·개요
  보기가 그대로 동작한다.

---

## CLI

```bash
balpyo init <dir>        # 경고 없이 빌드되는 세 장짜리 덱을 만든다
balpyo build             # --target html,pptx · --theme · --out · --strict · --check-layout
balpyo check             # 쓰지 않고 판정만 (CI 게이트)
balpyo watch             # 고칠 때마다 다시 빌드 + 브라우저 자동 새로고침
balpyo preview           # 웹/PPTX 렌더를 나란히 놓은 검수 갤러리
balpyo snap              # 브라우저 없이 슬라이드를 PNG/GIF로 (--slides · --scale · --gif · --outline)
balpyo version           # 버전 · 커밋 · 플랫폼
```

- 종료 코드: 에러가 있으면 비영. 경고는 0이지만 `--strict`가 에러로 승격한다.
- `--check-layout`은 잰 배치에서 넘침·스테이지 이탈·겹침을 경고한다
  ([배치 경고 읽기](guides/layout-warnings.md)).

서브커맨드별 플래그와 종료 코드는 [CLI 레퍼런스](reference/cli.md)가 갖는다.

---

## 예제

설명을 읽는 것보다 덱을 보는 것이 빠르다. 저장소의 `examples/decks/` 아래에 **사람이 쓴 덱
넷**이 있고, 넷 다 두 타깃으로 **경고 0건**에 빌드된다.

| 덱 | 장수 | 무엇을 보여 주는가 |
|---|---|---|
| [`examples/decks/balpyo-intro`](https://github.com/k2sebeom/balpyo/tree/main/examples/decks/balpyo-intro) | 12 | 이 도구 자체의 소개 덱 — 연결선으로 그린 빌드 경로, 색 토큰 견본, `<table>`, 뒤집힌 인용 장(`invert`), `lang="soml"` 강조 |
| [`examples/decks/incident-postmortem`](https://github.com/k2sebeom/balpyo/tree/main/examples/decks/incident-postmortem) | 10 | 장애 회고 (`ink` 테마) — 비례하는 시간 축, 한 결과로 모이는 곡선 연결선, 계열 둘의 `<chart>`, `lang="go"` 코드 창 |
| [`examples/decks/quarterly-review`](https://github.com/k2sebeom/balpyo/tree/main/examples/decks/quarterly-review) | 8 | 분기 리뷰 — 경로 · 막대 · 별로 그린 KPI 카드, 표와 차트, `image-text` 사진 장, 진행 막대 |
| [`examples/decks/ec2-intro`](https://github.com/k2sebeom/balpyo/tree/main/examples/decks/ec2-intro) | 13 | 손으로 짠 키노트 수준의 기술 소개 — 상태 기계 · 아키텍처 도식, `cube` 층, SVG 아이콘, 덱 안의 테마 `summit` |

```bash
./balpyo build --project examples/decks/quarterly-review --target html,pptx --strict
ls examples/decks/quarterly-review/dist/     # index.html + <덱 제목>.pptx
```

같은 덱을 다른 테마로 보려면 `--theme editorial`을 더한다 — **슬라이드는 한 줄도 고치지
않는다.**

`examples/themes/`에는 아무것도 물려받지 않는 **혼자 서는 테마** 둘이 있다 — 테마 하나를
처음부터 짜는 견본이다:

- [`harbor`](https://github.com/k2sebeom/balpyo/tree/main/examples/themes/harbor) — 차가운 회색
  바탕에 흰 카드, 굵은 제목, 위 왼쪽에서 시작하는 표지.
- [`aurora`](https://github.com/k2sebeom/balpyo/tree/main/examples/themes/aurora) — 한밤의 남색
  바탕에 Noto Sans Display 제목. 표지 · 구역 표지 · 마무리 장의 배경이 테마가 가진 오로라 그림이다
  — 덱은 그림을 한 번도 부르지 않는다.

덱의 `themes/<이름>/`으로 디렉토리를 복사하고 `balpyo.json`에 `"theme": "<이름>"`을 적으면 쓸 수
있다 (자세히는 [테마 고르기·만들기](guides/theming.md)).

---

## 비목표

- **WYSIWYG 편집기** — 텍스트 에디터 + `balpyo watch`가 UX다
- **PPTX 임포트(역방향)** — 렌더링은 단방향이다
- **픽셀 단위 일치** — 두 산출물은 요소와 배치가 같고, 줄바꿈 위치는 각 엔진이 정한다
- **Markdown 입력** — SOML만 읽는다
- **온라인 호스팅/공유 서비스** — 정적 산출물을 낼 뿐이다
- **타임라인 기반 애니메이션 저작** — `reveal` 순차 등장 수준만

---

## 더 읽을 것

| 문서 | 내용 |
|---|---|
| [시작하기](guides/getting-started.md) | 빈 디렉토리에서 발표까지 |
| [가이드](guides/index.md) | 사용자 시나리오별 안내 — 배치 경고, 테마, 계약 밖의 차이 |
| [레퍼런스](reference/index.md) | 태그·외관·레이아웃·테마·CLI — 지금 동작하는 것만 담는다 |
