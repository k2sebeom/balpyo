# balpyo

> **SOML**(Slide Oriented Markup Language)로 슬라이드를 쓰면,
> **balpyo**가 테마를 입혀 **웹 프레젠테이션**과 **편집 가능한 네이티브 PPTX**로 렌더링한다.

Hugo가 Markdown 콘텐츠를 테마로 렌더링하는 정적 사이트 생성기라면,
balpyo는 SOML 슬라이드를 테마로 렌더링하는 **프레젠테이션 생성기**다.

!!! note "현재 상태"
    Iteration 5까지 구현되었다 — `<slide>`/`<text>`/`<note>`와 텍스트 서브셋,
    레이아웃 태그(`<row>`/`<col>`/`<grid>`/`<spacer>`), `<shape>` 12종 + 외관 속성(단색),
    그리고 `<image>`(프로젝트 안의 PNG·JPEG → 웹 `<img>` + 산출 에셋 / PPTX `pic` + 미디어
    파트)까지 파싱 → 외관·배치 해석 → 웹 + 네이티브 PPTX 전 구간이 뚫려 있다.
    최대 리스크였던 텍스트 메트릭은 측정으로 허용 오차가 확정되었고 (PRD §13-1), 도형
    프리셋의 기하와 그림의 `fit` 크롭은 두 렌더의 요소별 픽셀 비교가 전수로 판정한다.
    **지금 동작하는 것**은 [레퍼런스](reference/index.md)에
    있다 — 이 문서의 나머지는 목표 명세이며 아직 전부 동작하지 않는다. 최신 진행 상황은
    [TASKS.md](https://github.com/k2sebeom/balpyo/blob/main/TASKS.md)를 참고한다.

---

## 왜 balpyo인가

텍스트로 슬라이드를 만드는 도구는 이미 여럿 있지만, "**슬라이드 전용 언어로 선언하고,
흐름 레이아웃으로 배치하되, 산출물은 웹과 편집 가능한 네이티브 PPTX 둘 다**"인 도구는 없다.

| 도구 | 한계 |
|---|---|
| Marp / Slidev | Markdown은 문서 마크업이라 슬라이드 어휘가 없다. PPTX 출력은 이미지 기반 |
| reveal.js | 웹 전용. 테마·레이아웃 시스템 빈약 |
| python-pptx / pptxgenjs | 모든 요소를 EMU 절대 좌표로 수동 배치. 흐름 레이아웃 부재 |
| react-pptx | JSX 문법은 제공하나 배치는 여전히 수동. 웹 렌더 타깃 없음 |
| PowerPoint / Keynote | GUI 기반. 버전 관리·재사용·자동화 불가 |

실제 기업 덱의 카드 그리드 슬라이드 한 장을 python-pptx로 재구성해 보니 **수동 좌표 계산
128회**와 **raw XML 수술 10회**가 필요했다. balpyo는 그 128회를 `<grid cols="4">` 한 줄로 대체한다.

### 핵심 통찰 세 가지

1. **PPTX의 도형 분류는 작다** — 텍스트 박스, 그림, 프리셋 도형, 커넥터, 테이블, 차트, 그룹.
   실물 덱을 전수 분석했을 때 콘텐츠의 **~90%가 이 어휘 안에** 있었다. 이 어휘를 1급 태그로
   만들면 표현력과 번역 가능성을 동시에 얻는다.
2. **브라우저를 레이아웃 엔진으로 빌려 쓸 수 있다** — flex/grid로 선언한 배치를 헤드리스
   브라우저가 계산하고, 그 결과 **지오메트리만** 읽어 PPTX 좌표로 굽는다. 저자는 좌표를
   만지지 않고, PPTX는 좌표만 받는다.
3. **OOXML은 이미 디자인 시스템이다** — 그러므로 balpyo의 테마를 MUI/Carbon 같은
   **디자인 시스템**(토큰 + 컴포넌트 외관 + variant)으로 정의하면 웹 CSS와 PPTX 테마
   양쪽으로 자연스럽게 컴파일된다.

---

## 두 개의 이름

| 이름 | 정체 | 역할 |
|---|---|---|
| **SOML** (Slide Oriented Markup Language) | 마크업 언어 (`.slide` 파일) | 슬라이드의 구조·내용·배치·외관을 선언 |
| **balpyo** | 렌더 엔진 · 뷰어 · CLI | SOML을 파싱해 테마를 입히고 타깃별로 렌더링 |

---

## 빠른 시작

```bash
brew install balpyo                    # 또는 바이너리 1개 내려받기 (런타임 의존 없음)

balpyo init my-talk && cd my-talk
balpyo server                          # localhost 라이브 프리뷰 (핫 리로드)
balpyo build --target html             # → dist/index.html   (발표용)
balpyo build --target pptx             # → dist/my-talk.pptx (제출용, 편집 가능)
```

`slides/*.slide`는 단 한 글자도 바뀌지 않는다. PowerPoint에서 연 `.pptx`의 텍스트 박스는
진짜 텍스트 박스이고, 도형은 진짜 도형이며, **색은 테마 색 참조를 유지**해 PowerPoint 안에서의
테마 색 편집을 따라간다.

### 프로젝트 구조

```text
my-talk/
├── balpyo.json              # 덱 설정: 제목, 테마, 종횡비, 타깃 옵션
├── slides/                  # 슬라이드 = 파일. 파일명 접두 번호가 순서
│   ├── 001_title.slide
│   ├── 002_agenda.slide
│   ├── 010_architecture.slide
│   └── 900_end.slide
├── components/              # 사용자 정의 합성 컴포넌트
│   └── dim-card.slide
├── assets/                  # 이미지, 폰트, 로고
├── themes/                  # (선택) 로컬·내려받은 테마. 내장 테마를 쓰면 불필요
└── dist/                    # 산출물 (.gitignore)
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
  "slidesDir": "slides",
  "outDir": "dist",
  "footer": "2026 Q3 · 내부용",
  "pageNumber": true,
  "targets": {
    "pptx": { "mode": "native" }
  }
}
```

우선순위: **CLI 플래그 > `balpyo.json` > 기본값.**

`footer`·`pageNumber`는 [덱 크롬](reference/theme.md#chrome)을 켠다 — 둘 다 기본은
꺼짐이고, 그 글자가 어디에 어떤 크기로 앉는지는 테마가 정한다.

---

## 아키텍처

```text
slides/*.slide ─파싱─▶ ┌────────────────────────────────┐ ◀─ theme (디자인 시스템)
   (SOML)              │  덱 모델 (AST)                  │    토큰 · 컴포넌트 외관/variant
                       │  + 합성 컴포넌트 전개            │    레이아웃 · 크롬
                       │  + 외관 해석 (스타일 resolve)    │
                       │  + 검증                         │
                       └───────────────┬────────────────┘
             ┌─────────────────────────┼─────────────────────────┐
             ▼                         ▼                         ▼
        웹 렌더러                 PPTX 렌더러                PDF 렌더러
   (SOML→HTML/CSS 변환,     (지오메트리 패스: 좌표만       (웹 렌더 결과를
    발표 런타임 포함)          + 외관→OOXML 직행)           헤드리스 브라우저로 인쇄)
```

**덱 모델(Deck Model)** 이 제품의 중심이다 — 슬라이드·레이아웃·슬롯·컴포넌트·해석된 외관·
노트·reveal 순서를 담은 정규화된 AST. SOML은 그 저작 문법이고, 각 타깃은 덱 모델을 소비하는
렌더러다. **새 타깃 추가 = 렌더러 추가.**

### 두 개의 선언 채널 — 이 제품의 핵심 경계

```text
배치(layout)      → 레이아웃 태그 (flex/grid 시맨틱)
                    … 지오메트리 패스가 좌표로 굽는다. 번역 불필요 — 결과만 이동한다.

외관(appearance)  → 외관 속성 + variant (타깃 중립 스키마)
                    … 편집 가능한 PPTX 속성으로 살아남아야 하므로 양 타깃으로 컴파일.
```

원칙: **번역할 필요가 없는 것(배치)은 브라우저에 맡기고, 번역되어 살아남아야 하는
것(외관)만 스키마를 만든다.** SOML에서는 이 경계가 린트 규칙이 아니라 **문법**이다 —
배치는 레이아웃 태그의 속성에, 외관은 컴포넌트의 외관 속성에 있다.

### 3층 계약

```text
┌──────────────────────────────────────────────────────────────┐
│ 1층 · 레이아웃 (Layouts)      — 테마가 배치를 소유              │
│   <slide layout="title"> + slot. 저자는 의도만 선언.            │
├──────────────────────────────────────────────────────────────┤
│ 2층 · 컴포넌트 (Components)   — 저자가 배치를, 테마가 외관을     │
│   <text> <shape> <arrow> … PPTX 1급 객체와 1:1 대응.           │
│   배치는 <row>/<col>/<grid>/<stack> 레이아웃 태그로 선언.        │
├──────────────────────────────────────────────────────────────┤
│ 3층 · 디자인 토큰 (Tokens)    — 모든 층을 관통하는 어휘          │
│   색·폰트·공간·형태. CSS 변수와 PPTX 테마 파트로 컴파일.         │
└──────────────────────────────────────────────────────────────┘
```

대부분의 슬라이드(표지, 불릿, 2단 비교, 인용)는 **1층**만으로 쓴다 — 테마를 바꾸면 배치까지
달라진다. 다이어그램·자유 구성 슬라이드는 **2층**으로 쓴다 — 저자가 배치를 소유하되 외관은
테마 기본값/variant를 따른다.

---

## SOML 맛보기

`.slide` 파일 하나는 슬라이드 하나이며, 루트 요소는 `<slide>`다. 별도 front matter는 없다 —
슬라이드 메타데이터는 루트 요소의 속성으로 선언한다.

### 1층 — 레이아웃과 슬롯

```html
<!-- slides/001_title.slide -->
<slide layout="title" transition="fade">
  <text slot="kicker">Platform Team</text>
  <text slot="title">2026 아키텍처 리뷰</text>
  <text slot="subtitle">상반기 회고와 하반기 계획</text>
  <text slot="author">이세범</text>
  <note>인사 30초. 청중에 인프라팀 포함되어 있음.</note>
</slide>
```

표준 레이아웃 카탈로그(11종)와 슬롯:

| 레이아웃 | 슬롯 | 용도 |
|---|---|---|
| `title` | kicker, title, subtitle, author, date | 표지 |
| `section-divider` | title, subtitle | 챕터 구분 |
| `content` (기본값) | title, body | 제목 + 본문 흐름 |
| `two-column` | title, left, right | 2단 비교/병렬 |
| `image-full` | media, caption | 전면 이미지 |
| `image-text` | title, media, body | 이미지+텍스트 조합 |
| `quote` | quote, attribution | 인용 |
| `code` | title, code, caption | 코드 중심 |
| `stat` | title, value, label, body | 수치 강조 콜아웃 |
| `canvas` | — | 자유 배치 (저자가 배치 소유) |
| `end` | title, contact | 마무리 |

존재하지 않는 슬롯 이름은 **빌드 에러**다.

### 2층 — 배치와 컴포넌트

```html
<slide layout="canvas">
  <text role="heading">서비스 아키텍처</text>

  <row gap="xl" align="center" justify="center" grow="1">
    <shape id="web" type="rounded-rect"><text>Web</text></shape>
    <shape id="api" type="rounded-rect" fill="accent" color="bg"><text>API Gateway</text></shape>
    <shape id="db"  type="cylinder"><text>Aurora</text></shape>
  </row>

  <arrow from="web" to="api" label="HTTPS" />
  <arrow from="api" to="db" dash="dashed" />

  <note>이 다이어그램은 3분 정도 설명할 것</note>
</slide>
```

레이아웃 태그는 **CSS flex/grid 시맨틱의 이름 바꾸기 층**이다. 새 레이아웃 모델이 아니다.

| 태그 | 컴파일 결과 |
|---|---|
| `<row>` | `display:flex; flex-direction:row` |
| `<col>` | `display:flex; flex-direction:column` |
| `<grid>` | `display:grid` |
| `<stack>` | 겹쳐 쌓기 (z축 배치 — 배지·워터마크·오버레이) |
| `<spacer>` | `flex:1` 빈 요소 (밀어내기) |

`gap="xl"`처럼 **공간 값이 토큰 어휘에 들어온다** — 배치조차 테마의 리듬을 따라간다.

### 태그 카탈로그

| 분류 | 태그 |
|---|---|
| 문서 | `<slide>`, `<note>` |
| 레이아웃 (배치) | `<row>`, `<col>`, `<grid>`, `<stack>`, `<spacer>` |
| 컴포넌트 (PPTX 1급) | `<text>`, `<shape>`, `<image>`, `<arrow>`, `<line>`, `<group>`, `<icon>`, `<table>`, `<chart>` |
| 합성 | `<define>`, `<slot>` |
| 탈출구 | `<raw>` |
| 텍스트 (HTML 서브셋) | `h1`~`h3`, `p`, `ul`/`ol`/`li`, `blockquote`, `pre`, `code`, `strong`, `em`, `a`, `br`, `span` |

!!! important "닫힌 어휘"
    미지의 태그는 무시되지 않고 **파싱 에러**가 된다. 오타를 조용한 시각 버그가 아니라
    빌드 실패로 만들고, AI 에이전트의 생성 품질도 끌어올린다.

### 컴포넌트 ↔ PPTX 대응

| 컴포넌트 | 웹 렌더 | PPTX 렌더 (OOXML) |
|---|---|---|
| `<text>` | 텍스트 요소 | 텍스트 박스 (`sp`+`txBody`) |
| `<shape>` | 박스 div + 그림 svg | 프리셋 도형 (`sp`+`prstGeom`) |
| `<image>` | img | 그림 (`pic`) |
| `<arrow>` | SVG 패스 (앵커 자동 계산) | 커넥터 (`cxnSp`) |
| `<line>` | SVG 라인 | 커넥터 (화살촉 없음) |
| `<group>` | div | 그룹 (`grpSp`) |
| `<icon>` | 인라인 SVG | SVG 패스 → `custGeom` 도형 (편집 가능) |
| `<table>` | table | 테이블 (`graphicFrame`+`tbl`) |
| `<chart>` | 차트 렌더 | 네이티브 차트 또는 이미지 폴백 |

`<shape type>` v1 어휘(12종)는 모두 OOXML 프리셋 지오메트리의 부분집합이다:
`rect`, `rounded-rect`, `ellipse`, `diamond`, `cylinder`, `chevron`, `arrow-right`,
`hexagon`, `parallelogram`, `callout`, `star`, `cloud`.

### 외관 속성

외관은 CSS가 아니라 **컴포넌트의 프레젠테이션 속성**으로 선언한다. SVG가
`<rect fill="…" stroke-width="…">`로 외관을 속성 선언하는 것과 같은 원리다.

```html
<shape type="rounded-rect"
       variant="card-backdrop"
       fill="none"
       border="3 gradient(accent1 accent4 accent5)"
       radius="lg" />
```

- **색 값은 토큰명 우선.** 토큰명으로 선언된 색은 PPTX에서 `schemeClr` 참조로 보존되어,
  **내보낸 파일이 PowerPoint 안에서도 테마 색 편집을 따라간다.**
- **해석 순서(캐스케이드)**: 인라인 속성 > `variant` > 테마의 컴포넌트 기본값 > 텍스트 역할/토큰.
  병합은 엔진이 파싱 시점에 수행하며 결과는 **타깃 무관**이다.

### 합성 컴포넌트

디자인된 슬라이드는 **동일한 도형 그룹의 반복**으로 만들어진다 (실물 덱 분석에서 카드
3도형 × 8회 반복 확인).

```html
<!-- components/dim-card.slide -->
<define name="dim-card">
  <col variant="card" gap="none">
    <shape type="rounded-rect" variant="card-header">
      <text><slot name="title" /></text>
    </shape>
    <shape type="rounded-rect" variant="card-body">
      <text><slot /></text>
    </shape>
  </col>
</define>
```

```html
<grid cols="4" gap="lg">
  <dim-card title="Controllability">시스템 행동을 감시·조종할 수단</dim-card>
  <dim-card title="Safety">유해 출력과 오용 방지</dim-card>
  <!-- … -->
</grid>
```

합성 컴포넌트는 **파싱 시 코어 컴포넌트로 전개되어** 덱 모델에 들어간다. 렌더러는 합성의
존재를 모르므로 PPTX 변환을 공짜로 얻는다.

---

## 테마 — 디자인 시스템으로서의 테마

테마는 CSS 파일이 아니다. 다음 네 가지를 소유하는 **디자인 시스템 패키지**다.

1. **디자인 토큰 값** — 색·폰트·스케일·공간·형태 (3층)
2. **컴포넌트 외관** — 컴포넌트별 기본값 + 이름 붙은 variant + 텍스트 역할 (2층)
3. **레이아웃 구현** — 표준 카탈로그 각 항목의 슬롯 배치 (1층, **배치 전용 CSS**)
4. **크롬(chrome)** — 로고, 브랜드 스트립, 푸터 등 전 슬라이드 장식. 선언적으로 정의해야
   웹과 PPTX 슬라이드 마스터 양쪽에 렌더링된다.

```text
my-theme/
├── theme.json          # 매니페스트: 이름, extends, 지원 레이아웃, 크롬 선언, 폰트
├── tokens.json         # 디자인 토큰 (단일 소스 → CSS 변수 + PPTX 테마 파트)
├── components.json     # 컴포넌트 기본 외관 + variant + 텍스트 역할
├── layouts/            # 레이아웃별 슬롯 배치 — 배치 전용 CSS
├── components/         # (선택) 테마 동반 합성 컴포넌트 (.slide)
├── web/                # 웹 전용 전환/등장 효과
├── assets/             # 폰트, 배경, 로고
└── preview.html        # (자동 생성) 갤러리 프리뷰
```

**공식 테마 3종은 바이너리에 내장된다** (`minimal`·`dark`·`paper`). `balpyo init` 직후
아무 설치 없이 예쁜 결과가 나오도록 기본 경로에서 네트워크를 제거한다.

```bash
balpyo theme new my-theme      # 기본 테마 베이스로 스캐폴딩
balpyo theme dev my-theme      # 갤러리 프리뷰 (웹 렌더와 PPTX 렌더 나란히), 핫 리로드
balpyo theme lint my-theme     # 토큰/레이아웃/variant 커버리지 + 배치·외관 경계 위반 검사
balpyo theme pack my-theme     # 배포용 tarball 빌드
balpyo theme get <git-url>     # 커뮤니티 테마를 themes/로 가져오기
```

---

## 렌더러

### 웹 (기준 타깃)

웹은 **기준(reference) 타깃**이다. 발표 경험의 1급 시민이며 다른 타깃의 시각적 정답지다.

- SOML → HTML 변환 + 외관 스키마 → 생성 CSS. 레이아웃 태그의 배치 CSS는 그대로 통과.
- 키보드/터치 네비게이션, `reveal` 순차 등장, URL 해시 딥링크(`#/5`, `#/id`),
  화면 크기 무관 스케일링, 오버뷰 모드, 발표자 뷰(노트+타이머+다음 슬라이드).
- 런타임은 프레임워크 무의존 vanilla TS(~10KB 목표)이고 빌드 산출물을 바이너리에 내장한다.
  산출물은 단일 `index.html`(+assets) — **파일로 열어도 동작해 오프라인 발표가 가능하다.**

### PPTX — 지오메트리 패스

핵심 기술 전략: **외관은 스키마에서 직행하고, 브라우저는 배치 계산에만 쓴다.**

```text
덱 모델 (외관 해석 완료)
   │
   ├── ① 지오메트리 패스: 헤드리스 브라우저에서 웹 렌더러와 동일한 HTML/CSS로 렌더링
   │      → 요소별 getBoundingClientRect → EMU 좌표 변환. 읽는 것은 좌표뿐.
   │
   ├── ② 외관 직행: 해석된 외관 스키마 → OOXML 속성 매핑
   │      토큰 참조는 schemeClr로 보존 → 산출물이 PPTX 테마 편집을 따라감
   │
   ▼
③ OOXML 조립: 마스터/레이아웃/테마 파트 + 슬라이드별 네이티브 도형 (xfrm=①, 외관=②)
   │
   ▼
deck.pptx (PowerPoint/Keynote에서 열어 편집 가능)
```

- computed-style **역추론은 없다.** 외관은 스키마에서 양 타깃으로 순방향 컴파일되므로
  손실 지점이 없고, 지오메트리 패스는 좌표 읽기라는 단일 책임만 갖는다.
- 같은 엔진(브라우저)이 웹과 PPTX의 배치를 계산하므로 **두 타깃의 배치 일치가 구조적으로
  보장된다.**
- 텍스트 재배치 대책: 텍스트 박스는 측정된 크기를 명시 고정하고 PowerPoint autofit을 끈다.
- **지오메트리 백엔드**는 시스템 브라우저(Chrome/Edge 자동 감지)를 CDP로 직접 제어한다.
  브라우저 다운로드 없음, 콜드 스타트 ~0.6초, 슬라이드당 ~1.5ms. PDF 인쇄와 `<raw>`
  래스터화가 같은 CDP 세션을 재사용하므로 세 기능이 하나의 의존성으로 해결된다.

### 외관 스키마 v1 — 타깃별 컴파일 매핑

| 외관 속성 | 웹 (생성 CSS) | PPTX (OOXML) |
|---|---|---|
| `fill` (단색/토큰) | `background-color: var(--balpyo-*)` | `solidFill` (+`schemeClr`) |
| `fill=gradient(…)` | `background-image: linear-gradient(…)` | `gradFill` |
| `border` (단색) | `border` | `ln` |
| **`border=… gradient(…)`** | 엔진 구현 (SVG 오버레이/mask) | `ln` + `gradFill` |
| **텍스트 `color=gradient(…)`** | `background-clip: text` | `rPr` + `gradFill` |
| `radius` | `border-radius` | `prstGeom` 조정값 |
| `font`/`size`/`weight`/`color`/`align` | `font-*`, `color`, `text-align` | `rPr` / `pPr` |
| **색 변형** (`shade()`, `alpha()`) | `color-mix()` / rgba | `schemeClr` + `lumMod`/`lumOff`/`alpha` |
| `cap` / `dash` | `stroke-linecap`, `stroke-dasharray` | `ln@cap`, `prstDash` |
| `rotate` / `flip` | `transform` | `xfrm@rot`, `flipH`/`flipV` |
| `opacity` / `shadow` | 색 알파 / `filter: drop-shadow` | 색 안의 `a:alpha` / `effectLst` outerShdw |
| 위치·크기 | (브라우저가 원본) | `xfrm` (EMU) |

실물 기업 덱 역분석 결과 "디자인된 느낌"은 굵은 글씨 항목(**그라디언트 3종, 토큰 색 변형**)에
집중되어 있었다. v1 스코프에 반드시 포함한다.

스키마 밖의 시각 효과(filter, clip-path, blend mode 등)는 저자 CSS로 쓸 수 있으나
**웹 전용**이며, dev 모드가 "PPTX에 번역되지 않음"을 경고한다.

---

## CLI

```bash
balpyo init <name>                     # 프로젝트 스캐폴딩
balpyo new <name> [--layout content]   # slides/에 다음 번호로 슬라이드 생성
balpyo server [--port 4321] [--open]   # 라이브 뷰어 + 핫 리로드
balpyo build [--target html|pptx|pdf] [--out dist] [--theme …] [--mode native|image]
balpyo theme new|dev|lint|pack|get …   # 테마 워크플로우
balpyo version                         # 버전 + 감지된 브라우저 경로
```

### 저작 품질 명령군

세 명령은 **하나의 파서·검증기 코어를 공유**한다. 표면만 다르므로 규칙이 서로 어긋날 수
없다 — 에디터에서 본 경고와 CI가 막는 경고가 항상 같다.

| 명령 | 형태 | 소비자 |
|---|---|---|
| `balpyo check [--strict]` | 단발 실행 → 진단 + 종료 코드 | CI, pre-commit, 터미널 |
| `balpyo format [--check]` | 파일 제자리 수정 (주석·텍스트 무손실 보존) | 저장 시 자동 실행, CI |
| `balpyo lsp` | 상주 서버 (LSP over stdio) | 에디터 — 입력 중 실시간 진단·완성·호버 |

`balpyo lsp`는 **해석된 테마를 알기 때문에** 스키마보다 더 안다 — `slot="`에는 선택된
레이아웃이 실제로 정의한 슬롯만, `variant="`에는 그 테마의 variant 이름만 제안한다.
즉 스키마는 **문법**을, LSP는 **테마 계약**을 발견 가능하게 만든다.

공통 규약:

- `--target` 다중 지정 가능: `balpyo build --target html,pptx`
- 모든 옵션은 `balpyo.json`으로 고정 가능 — `balpyo build` 한 번으로 재현 가능한 빌드
- 종료 코드: 검증 에러 시 비영. 경고는 0이지만 `--strict`로 승격 가능
- **바이너리 하나가 모든 서브커맨드**(뷰어·렌더러·포매터·언어 서버)를 담는다

---

## 구현

엔진은 **Go**로 구현한다. SOML이 자체 언어가 된 순간 JS 생태계에 남을 이유(HTML 파서
재사용)가 사라졌고, 남은 요구사항이 Go의 강점과 정확히 겹친다.

| 요구사항 | Go에서의 해법 |
|---|---|
| 단일 바이너리 배포 | 정적 링크 + 크로스 컴파일. 런타임·패키지 매니저 불필요 |
| 공식 테마·웹 런타임 내장 | `//go:embed` — 설치 없이 첫 결과물이 나온다 |
| 지오메트리 패스 · PDF · 래스터화 | `chromedp`/`go-rod`로 시스템 브라우저를 CDP 직접 제어 |
| OOXML 조립 | 표준 라이브러리 `archive/zip` + `text/template`로 XML 파트 직조립 |
| 코드 하이라이팅 | Chroma — 토큰 스트림 API라 웹 span과 PPTX 텍스트 런 양쪽에 쓸 수 있다 |
| dev 서버 + 핫 리로드 | `net/http` + WebSocket, 파일 워처 |
| CI 친화성 | 다운로드 1개. `node_modules`·락파일·가상환경 없음 |

### 패키지 경계

```text
soml/         파서 + 덱 모델   (외부 임포트 가능한 공개 API)
core/         외관 해석 · 검증
render/web/   SOML→HTML/CSS + 웹 런타임
render/pptx/  지오메트리 패스 + OOXML 조립
cmd/balpyo/   CLI
```

**SOML 파서는 `encoding/xml`에 의존하지 않고 직접 작성한다.** 닫힌 어휘 검증과 정확한
line/col 진단이 언어의 핵심 가치이므로, 파서는 처음부터 세 소비자를 전제로 설계한다 —
이것이 나중에 붙일 수 없는 유일한 설계 결정이다.

| 소비자 | 파서에 요구하는 것 |
|---|---|
| `build` / `check` | 첫 에러에서 멈춰도 무방. 정확한 위치 |
| `lsp` | **에러 복구** — 입력 중 문서는 항상 깨져 있다. 부분 AST + 진단 목록 |
| `format` | **무손실 파스** — 주석·공백·인용 부호 같은 트리비아 보존 |

---

## 로드맵

구현은 **태그 세트 단위 반복**으로 진행한다. 각 iteration은 `테스트 작성(RED) →
구현(GREEN) → 결과 확인·시각 리뷰(REVIEW)` 3단계를 밟고, 테스트는 LibreOffice가 설치된
Docker 컨테이너 안에서 돌아 로컬과 CI가 같은 결과를 낸다.

| Iteration | 범위 |
|---|---|
| 0 | 기반: Go 패키지 골격 + Docker 테스트 환경 + 테스트 하네스 |
| 1 | **수직 절단**: `<slide>` `<text>` `<note>` — 파싱부터 네이티브 PPTX까지 전 구간 |
| 2 | 텍스트 서브셋(`h1`~`h3`, `ul`, `strong` …) + 역할/캐스케이드 완성 |
| 3 | 레이아웃 태그: `<row>` `<col>` `<grid>` `<spacer>` |
| 4 | `<shape>` 12종 + 외관 속성(단색) |
| 5 | `<image>` |
| 6 | `<stack>` + `layout="canvas"` + 절대 배치 |
| 7 | `<arrow>` `<line>` (가장 어려운 세트 — 앵커 계산) |
| 8 | 외관 스키마 v1 완성: 그라디언트 + 색 변형 |
| 9 | `<group>` |
| 10 | 1층: 표준 레이아웃 카탈로그 + 슬롯 + 테마 패키지 포맷 |
| 11 | 합성 컴포넌트: `<define>` `<slot>` |

!!! warning "Iteration 1이 관문이다"
    최대 리스크는 **텍스트 메트릭** — 브라우저가 잰 박스가 LibreOffice/Keynote에서 같은
    줄바꿈을 유지하는가. Iteration 1이 실패하면(오차가 수용 불가) 아키텍처를 재검토한다.
    그래서 가장 먼저 한다. **PDF 렌더는 현재 구현 범위 밖이다** — 타깃은 웹과 PPTX 둘.

### 비목표

- **WYSIWYG 편집기** — 에디터 + `balpyo server`가 UX다
- **PPTX 임포트(역방향)** — 렌더링은 단방향이다
- **CSS/레이아웃 엔진의 재발명** — 레이아웃 태그는 CSS flex/grid의 이름 바꾸기 층이다
- **Markdown 지원** — v1에는 없다 (`md → SOML` 변환기는 별도 도구로 고려)
- **온라인 호스팅/공유 서비스** — 정적 산출물을 낼 뿐이다
- **타임라인 기반 애니메이션 저작** — 전환/순차 등장 수준만

---

## 성공 지표

| 지표 | 목표 (v1 출시 후 6개월) |
|---|---|
| "슬라이드 무수정 테마 교체" 성공률 | 100% (계약 준수 콘텐츠 기준) |
| **PPTX 네이티브 변환율** | 코어 어휘로 작성된 슬라이드의 100%가 편집 가능한 객체로 |
| **웹 ↔ PPTX 시각적 일치** | 픽셀 비교 기준 주요 편차 0건 (줄바꿈 허용 오차 내) |
| **PPTX 산출물의 테마 보존** | 토큰 참조 색이 `schemeClr`로 보존되어 테마 편집에 반응 |
| 첫 슬라이드까지 소요 시간 | < 1분 (`balpyo init` → 브라우저) |
| 첫 PPTX 산출까지 소요 시간 | < 5분 |
| 좌표 계산 제거율 | 카드 그리드 슬라이드 기준 128회 → **0회** |

---

## 더 읽을 것

| 문서 | 내용 |
|---|---|
| [레퍼런스](reference/index.md) | 태그·외관·CLI 레퍼런스 — 구현된 것만 담는다 |
| [가이드](guides/index.md) | 사용자 시나리오별 안내 |
| [아키텍처](internals/architecture.md) | 파이프라인, 패키지 경계, 덱 모델, 두 렌더러 |
| [테스트 환경](internals/testing.md) | Docker 이미지, make 타깃, 골든·픽셀 비교 규칙 |
| [설계 결정](decisions/index.md) | 구현 중 확정한 ADR 색인 |
| [PRD.md](https://github.com/k2sebeom/balpyo/blob/main/PRD.md) | 제품 정의 전문 — SOML 명세, 테마 시스템, 렌더러, 확정된 설계 결정, 리스크 |
| [TASKS.md](https://github.com/k2sebeom/balpyo/blob/main/TASKS.md) | 구현 플랜 — iteration별 테스트/구현/리뷰 체크리스트, 설계 결정 로그 |

### 이 문서 사이트 빌드하기

```bash
make docs          # → site/ 에 정적 사이트 생성
make docs-serve    # http://127.0.0.1:8000 라이브 프리뷰
```
