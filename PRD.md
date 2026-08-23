# PRD: balpyo — 슬라이드 렌더 엔진 & SOML

> **SOML**(Slide Oriented Markup Language)로 슬라이드를 쓰면,
> **balpyo**가 테마를 입혀 웹 프레젠테이션과 **편집 가능한 네이티브 PPTX**로 렌더링한다.
>
> Hugo가 Markdown 콘텐츠를 테마로 렌더링하는 정적 사이트 생성기라면,
> balpyo는 SOML 슬라이드를 테마로 렌더링하는 **프레젠테이션 생성기**다.

- **문서 버전**: 1.1
- **작성일**: 2026-08-13 (최종 수정 2026-08-16)
- **상태**: 제품 정의 확정. 구현 언어 **Go** 확정(§12.1). **열린 질문 12건 확정(§13)**.
  구현 착수 전 마지막 스파이크(M0.5) 대기
- **참고**: 이 정의에 도달하기까지의 아이데이션 과정·프로토타입·역분석 실험 전문은
  [`PRELIMINARY.md`](PRELIMINARY.md)에 보존되어 있다. 본 문서는 히스토리 없이
  **최종 통합 제품 정의**만 담는다.

---

## 1. 배경 및 문제 정의

### 1.1 배경

**(a) Markdown 프레젠테이션 도구의 표현력 한계** — Marp, Slidev 등은 "텍스트로
슬라이드를 만든다"는 점에서 매력적이지만, Markdown은 문서 마크업이지 **슬라이드
마크업이 아니다**. 다이어그램, 텍스트 박스, 2단 배치, 도형 연결 같은 슬라이드의
기본 어휘가 언어에 없어 비표준 확장이나 인라인 HTML 탈출구에 의존한다.

**(b) 프로그래매틱 PPTX 생성의 고통** — 조직에서 "PowerPoint 파일로 주세요"는
피할 수 없는 요구이고, 자동 리포트 덱 수요도 크다. 그러나 python-pptx, pptxgenjs
같은 라이브러리는 **모든 요소를 EMU 절대 좌표로 배치**해야 한다. 실제 기업 덱의
카드 그리드 슬라이드 한 장을 재구성해 보면 수동 좌표 계산 128회가 필요했고,
그라디언트 테두리·그라디언트 텍스트처럼 "디자인된 느낌"을 만드는 요소에는 API조차
없어 raw XML을 직접 조립해야 했다.

**(c) HTML 기반 도구의 미완성** — reveal.js 등은 웹의 표현력을 쓰지만 산출물이
웹뿐이고, 테마는 CSS 파일 수준이며, PPTX 내보내기는 슬라이드를 이미지로 찍는
수준에 머문다 (편집 불가).

### 1.2 문제 정의

> "슬라이드를 **슬라이드 전용 언어로 선언하고, 흐름 레이아웃(flex/grid)으로 배치**하되,
> 산출물은 **웹 프레젠테이션과 편집 가능한 네이티브 PPTX 둘 다**인 도구가 없다."

| 도구 | 한계 |
|---|---|
| Marp / Slidev | Markdown 표현력 한계. PPTX 출력은 이미지 기반 |
| reveal.js | 웹 전용. 테마·레이아웃 시스템 빈약 |
| python-pptx / pptxgenjs | 절대 좌표 수동 배치. 흐름 레이아웃 부재. 시그니처 디자인 요소 API 부재 |
| react-pptx | JSX 문법은 제공하나 배치는 여전히 수동. 웹 렌더 타깃 없음 |
| PowerPoint / Keynote | GUI 기반. 버전 관리·재사용·자동화 불가 |

### 1.3 핵심 통찰

1. **PPTX의 도형 분류는 작다** — 텍스트 박스, 그림, 프리셋 도형, 커넥터, 테이블,
   차트, 그룹. 실제 기업 덱을 전수 분석했을 때 콘텐츠의 ~90%가 이 어휘 안에 있었다.
   이 어휘를 1급 태그로 만들면 표현력과 번역 가능성을 동시에 얻는다.
2. **브라우저를 레이아웃 엔진으로 빌려 쓸 수 있다** — flex/grid로 선언한 배치를
   헤드리스 브라우저가 계산하고, 그 결과 지오메트리만 읽어 PPTX 좌표로 굽는다.
   저자는 좌표를 만지지 않고, PPTX는 좌표만 받는다.
3. **OOXML은 이미 디자인 시스템이다** — 테마가 색/폰트/스타일 매트릭스를 정의하고
   도형이 이를 참조하는 구조. 그러므로 balpyo의 테마를 MUI/Carbon 같은 **디자인
   시스템**(토큰 + 컴포넌트 기본 외관 + variant)으로 정의하면 웹 CSS와 PPTX 테마
   양쪽으로 자연스럽게 컴파일된다.

---

## 2. 제품 비전

> **"슬라이드는 SOML로 한 번 쓰고, balpyo가 웹·PDF·편집 가능한 PPTX로 렌더링한다.
> 테마는 CSS 파일이 아니라 디자인 시스템이다."**

제품은 두 개의 이름으로 구성된다.

| 이름 | 정체 | 역할 |
|---|---|---|
| **SOML** (Slide Oriented Markup Language) | 마크업 언어 (`.slide` 파일) | 슬라이드의 구조·내용·배치·외관을 선언 |
| **balpyo** | 렌더 엔진 · 뷰어 · CLI | SOML을 파싱해 테마를 입히고 타깃별로 렌더링 |

- 저자는 `slides/001_title.slide`에 SOML로 **내용과 구조**를 쓴다.
  배치는 `<row>`/`<col>`/`<grid>`/`<stack>` 태그로, 외관은 외관 속성과 variant로 선언한다.
- balpyo는 슬라이드들을 **덱 모델(AST)** 로 정규화하고 외관을 해석한 뒤, 타깃별
  렌더러가 이를 소비한다: 웹(발표 런타임 포함), PDF, **네이티브 PPTX**.
- 테마는 **토큰 + 컴포넌트 외관/variant + 레이아웃 구현 + 크롬**을 담은 디자인 시스템
  패키지다. 같은 슬라이드에 테마만 바꿔 끼우면 웹에서도 PPTX에서도 다른 브랜드가 나온다.
- OOXML의 모든 잡무(마스터, 관계 파일, EMU 좌표, z-순서, autofit)는 엔진이 소유한다.
  저자는 XML도, 좌표도 보지 않는다.

### 한 줄 데모 시나리오

```bash
brew install balpyo                    # 또는 바이너리 1개 내려받기 (런타임 의존 없음)

balpyo init my-talk && cd my-talk
balpyo server                          # localhost 라이브 프리뷰 (핫 리로드)
balpyo build --target html             # → dist/index.html   (발표용)
balpyo build --target pptx             # → dist/my-talk.pptx (제출용, 편집 가능)
```

`slides/*.slide`는 단 한 글자도 바뀌지 않는다. PowerPoint에서 연 `.pptx`의 텍스트
박스는 진짜 텍스트 박스이고, 도형은 진짜 도형이며, **색은 테마 색 참조를 유지**해
PowerPoint 안에서의 테마 색 편집을 따라간다.

---

## 3. 목표 / 비목표

### 3.1 목표 (Goals)

1. **G1 — 슬라이드 전용 언어**: SOML은 슬라이드의 어휘(슬라이드·도형·화살표·노트·행·열)를
   1급 태그로 갖는다. HTML/XML 계열 문법이라 기존 에디터·포매터·하이라이터를 그대로
   쓰고, 컴파일 도구 없이 텍스트 에디터만으로 저작한다. 파서가 문법·구조 오류를
   경고가 아닌 **에러로** 잡는다.
2. **G2 — 흐름 레이아웃으로 좌표 해방**: 저자는 flex/grid 시맨틱으로 배치를 선언하고,
   절대 좌표 계산은 엔진(지오메트리 패스)이 대신한다.
3. **G3 — 멀티 타깃 렌더링**: 하나의 덱 모델에서 웹·PDF·네이티브 PPTX를 렌더링한다.
   SOML 어휘로 작성된 콘텐츠는 **PPTX에서 편집 가능한 네이티브 객체**가 된다.
4. **G4 — 외관의 무손실 이식**: 외관 스키마로 선언된 속성은 웹과 PPTX에서 동일한
   시각 결과를 낸다. 역추론 없음 — 스키마에서 양 타깃으로 순방향 컴파일.
5. **G5 — 계층적 관심사 분리**: 레이아웃 슬라이드에서는 테마가 배치까지 소유하고,
   캔버스 슬라이드에서는 저자가 배치를, 테마가 외관을 소유한다. 어느 쪽이든
   색·폰트·장식은 테마 교체를 따라간다.
6. **G6 — Hugo급 프로젝트 경험**: `init`으로 스캐폴딩, `server`로 라이브 프리뷰,
   `build`로 재현 가능한 산출물. 슬라이드 하나 = 파일 하나라 Git diff·리뷰·병합이 자연스럽다.
7. **G7 — 단일 바이너리 배포**: balpyo는 런타임 의존 없는 정적 바이너리 하나다.
   공식 테마·웹 런타임·템플릿을 바이너리에 내장하므로, 내려받아 실행하면 즉시
   슬라이드가 나온다 (`node_modules`도, 가상환경도, 브라우저 드라이버 설치도 없다).
8. **G8 — 매끄러운 테마 제작 경험**: 스캐폴딩, 갤러리 프리뷰, 린트, 패키징.

### 3.2 비목표 (Non-Goals)

- **WYSIWYG 편집기**: GUI 편집 도구는 만들지 않는다. (에디터 + `balpyo server`가 UX)
- **PPTX 임포트 (역방향)**: `.pptx → SOML` 변환은 지원하지 않는다. 렌더링은 단방향이다.
- **CSS/레이아웃 엔진의 재발명**: SOML의 레이아웃 태그는 CSS flex/grid 시맨틱의
  **이름 바꾸기 층**이다. 새 레이아웃 모델을 정의하지 않는다.
- **CSS 전체의 외관 스키마화**: 외관 스키마는 정의된 속성 집합(§9.4)만 보장한다.
  그 밖의 시각 효과는 웹 전용이며 PPTX로 번역되지 않는다 (빌드 경고).
- **Markdown 지원**: v1에서는 없다. `md → SOML` 변환기는 별도 도구로 고려 (P2).
- **온라인 호스팅/공유 서비스**: 정적 산출물을 낼 뿐, 호스팅은 사용자 몫이다.
- **타임라인 기반 애니메이션 저작**: 전환/순차 등장(reveal) 수준만 제공한다.

---

## 4. 사용자 페르소나

| 페르소나 | 설명 | 핵심 니즈 |
|---|---|---|
| **발표하는 개발자** (주 타깃) | 기술 발표 자료를 코드처럼 만들고 싶은 엔지니어 | Git 버전 관리, 다이어그램, 예쁜 기본 테마, "PPT로 달라"는 요청 대응 |
| **문서 자동화 엔지니어** | 주간 리포트·대시보드 덱을 프로그램으로 생성해야 하는 개발자 | 좌표 지옥 탈출. 템플릿+데이터 → PPTX 파이프라인 |
| **테마 제작자** | 팀/회사 브랜드 테마를 만들어 배포하는 개발자 | 명확한 디자인 시스템 API, 프리뷰 도구, 웹·PPTX 동시 대응 |
| **조직/팀** | 전 팀원이 같은 브랜드 룩으로 자료를 만들어야 하는 조직 | 테마+컴포넌트 라이브러리 중앙 배포, PowerPoint 워크플로우와 호환 |
| **AI 에이전트** (부상 중) | "이 내용으로 발표자료 만들어줘"를 수행하는 코딩 에이전트 | 닫힌 어휘 + 스키마 검증. 좌표 없이 텍스트만 생성해도 디자인된 결과 |

---

## 5. 아키텍처

### 5.1 파이프라인

```
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

- **덱 모델(Deck Model)** 이 제품의 중심이다 — 슬라이드·레이아웃·슬롯·컴포넌트·
  해석된 외관·노트·reveal 순서를 담은 정규화된 AST. SOML은 그 저작 문법이고,
  각 타깃은 덱 모델을 소비하는 렌더러다. 새 타깃 추가 = 렌더러 추가.
- **외관 해석은 엔진이 한다** (타깃 무관): 인라인 속성 > `variant` > 컴포넌트 기본값 >
  텍스트 역할/토큰 순으로 병합해 각 노드에 해석된 외관을 부착한다. 렌더러는 이를
  각 타깃의 표현(CSS 선언 / OOXML 속성)으로 기계적으로 변환만 한다.
- 덱 모델은 JSON 직렬화 가능하며 공식 스키마를 갖는다.
- **SOML → HTML 변환은 엔진 내부 구현 세부다.** 웹 렌더러와 지오메트리 패스가
  같은 변환 결과를 공유하므로, 두 타깃의 배치 일치가 구조적으로 보장된다.

### 5.2 3층 계약

```
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

- 대부분의 슬라이드(표지, 불릿, 2단 비교, 인용)는 **1층**만으로 작성한다.
  테마를 바꾸면 배치까지 완전히 달라진다.
- 다이어그램·자유 구성 슬라이드는 **2층**으로 작성한다. 저자가 배치를 소유하되
  외관은 테마 기본값/variant를 따르고, 필요할 때만 인라인 속성으로 오버라이드한다.

### 5.3 두 개의 선언 채널 — 이 제품의 핵심 경계

```
배치(layout)      → 레이아웃 태그 (flex/grid 시맨틱)
                    … 지오메트리 패스가 좌표로 굽는다. 번역 불필요 — 결과만 이동한다.

외관(appearance)  → 외관 속성 + variant (타깃 중립 스키마)
                    … 편집 가능한 PPTX 속성으로 살아남아야 하므로 양 타깃으로 컴파일.
```

원칙: **번역할 필요가 없는 것(배치)은 브라우저에 맡기고, 번역되어 살아남아야 하는
것(외관)만 스키마를 만든다.** SOML에서는 이 경계가 린트 규칙이 아니라 **문법**이다 —
배치는 레이아웃 태그의 속성에, 외관은 컴포넌트의 외관 속성에 있다.

---

## 6. 프로젝트 구조와 워크플로우

### 6.1 `balpyo init`이 만드는 구조

```
my-talk/
├── balpyo.json              # 덱 설정: 제목, 테마, 종횡비, 타깃 옵션
├── slides/                  # 슬라이드 = 파일. 파일명 접두 번호가 순서
│   ├── 001_title.slide
│   ├── 002_agenda.slide
│   ├── 010_architecture.slide
│   └── 900_end.slide
├── components/              # 사용자 정의 합성 컴포넌트 (§7.8)
│   └── dim-card.slide
├── assets/                  # 이미지, 폰트, 로고
├── themes/                  # (선택) 로컬·내려받은 테마. 내장 테마를 쓰면 불필요
└── dist/                    # 산출물 (.gitignore)
```

- **슬라이드 하나 = 파일 하나.** 순서는 파일명 정렬(`001_`, `010_`, `900_`)로 결정된다.
  번호 사이를 띄워 두면 중간 삽입이 파일 하나 추가로 끝난다.
- **하위 디렉토리는 챕터**다. `slides/02_deep-dive/010_...` 처럼 나누면 경로 정렬
  순서대로 이어지고, 오버뷰/발표자 뷰에서 챕터로 묶여 표시된다.
- **`_`로 시작하는 파일은 무시**된다 (작업 중 초안, 보류 슬라이드).
- 슬라이드 하나가 파일 하나이므로 **Git diff·코드 리뷰·병합·재배치가 자연스럽다.**
  여러 사람이 같은 덱의 다른 슬라이드를 동시에 작업해도 충돌하지 않는다.

### 6.2 `balpyo.json`

```jsonc
{
  "title": "2026 아키텍처 리뷰",
  "author": "이세범",
  "theme": "minimal",             // 내장 테마 이름 · "./themes/corp" · git URL (§8.6)
  "aspect": "16:9",               // 논리 스테이지 1280×720 (4:3 → 960×720)
  "lang": "ko",
  "slidesDir": "slides",
  "outDir": "dist",
  "targets": {
    "pptx": { "mode": "native" }  // "native" | "image"
  }
}
```

우선순위: CLI 플래그 > `balpyo.json` > 기본값.

`aspect`는 v1에서 **`16:9`와 `4:3` 2종만** 받는다 (§13-9). 논리 스테이지는 각각
1280×720 / 960×720이고, PPTX 페이지 크기(13.333×7.5in / 10×7.5in)로 연동된다.
두 비율이 **높이를 공유**하므로 폰트 크기와 수직 리듬은 그대로 두고 폭만 다루면 된다.
4:3 스테이지가 1024×768이 아닌 이유는 `docs/decisions/0002-logical-stage-size.md`에 있다.
테마의 표준 레이아웃은 두 비율을 모두 지원해야 하며 `balpyo theme lint`가 검사한다.
임의 비율은 v2에서 검토한다.

### 6.3 저작 워크플로우

```bash
balpyo init my-talk            # 프로젝트 스캐폴딩 (샘플 슬라이드 3장 포함)
balpyo new agenda              # slides/에 다음 번호로 슬라이드 파일 생성
balpyo server                  # localhost 뷰어 + 핫 리로드 + 검증 오버레이
balpyo format                  # SOML 정규 형식으로 정렬
balpyo check                   # CI용 검증 (문법·슬롯·미지원 외관)
balpyo build --target html     # 자체 완결형 정적 프레젠테이션
balpyo build --target pptx     # 네이티브 PPTX
```

---

## 7. SOML 명세 (Slide Oriented Markup Language)

### 7.1 파일 형태

`.slide` 파일 하나는 슬라이드 하나이며, 루트 요소는 `<slide>`다. 별도 front matter는
없다 — 슬라이드 메타데이터는 루트 요소의 속성으로 선언한다 (단일 메커니즘).

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

**`<slide>` 속성**

| 속성 | 의미 |
|---|---|
| `layout` | 레이아웃 카탈로그에서 선택 (생략 시 `content`) |
| `id` | 딥링크 앵커 이름 (`#/architecture`). 생략 시 파일명에서 유도 |
| `transition` | 전환 효과 (웹 전용) |
| `skip` | `true`면 렌더에서 제외 (파일은 유지) |

**확장자 규칙 (확정, §13-2)** — SOML 파일의 확장자는 **`.slide` 하나**다. 합성
컴포넌트도 `components/dim-card.slide`처럼 같은 확장자를 쓴다. 파일의 역할은 확장자가
아니라 **루트 요소**(`<slide>` vs `<define>`)와 디렉토리(`slides/` vs `components/`)가
결정하며, 파서가 위치에 맞는 루트 요소를 검증한다 (`slides/` 안의 `<define>`은 빌드
에러, 역도 같다). 에디터·스키마·포매터·LSP는 파일 타입 하나만 등록하면 된다.

### 7.2 태그 카탈로그

| 분류 | 태그 |
|---|---|
| 문서 | `<slide>`, `<note>` |
| 레이아웃 (배치) | `<row>`, `<col>`, `<grid>`, `<stack>`, `<spacer>` |
| 컴포넌트 (PPTX 1급) | `<text>`, `<shape>`, `<image>`, `<arrow>`, `<line>`, `<group>`, `<icon>`, `<table>`, `<chart>` |
| 합성 | `<define>`, `<slot>` |
| 탈출구 | `<raw>` |
| 텍스트 (HTML 서브셋) | `h1`~`h3`, `p`, `ul`/`ol`/`li`, `blockquote`, `pre`, `code`, `strong`, `em`, `a`, `br`, `span` |

설계 원칙: **닫힌 어휘**. 미지의 태그는 무시되지 않고 파싱 에러가 된다. 이것이 오타를
조용한 시각 버그가 아니라 빌드 실패로 만들고, AI 에이전트의 생성 품질도 끌어올린다.

### 7.3 1층 — 레이아웃과 슬롯

레이아웃 슬라이드에서는 **테마가 배치를 소유**한다. 저자는 슬롯에 콘텐츠만 넣는다.

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
| `canvas` | — | 자유 배치 (저자가 배치 소유, 테마는 배경/여백/외관 기본값만) |
| `end` | title, contact | 마무리 |

```html
<slide layout="two-column">
  <text slot="title">Before / After</text>
  <col slot="left" gap="md">
    <text role="heading">Before</text>
    <shape type="rect" variant="stat-card"><text>배포에 3시간</text></shape>
  </col>
  <col slot="right" gap="md">
    <text role="heading">After</text>
    <shape type="rect" variant="stat-card" fill="accent"><text>배포에 8분</text></shape>
  </col>
</slide>
```

- 슬롯 지정은 `slot="이름"` 속성. 슬롯 없는 직계 자식은 기본 슬롯(`body`)에 들어간다.
- 테마는 자체 레이아웃을 추가할 수 있다 (`layout="corp-hero"`).
- 테마가 구현하지 않은 표준 레이아웃은 기본 테마 구현으로 폴백하고 dev 모드에서 경고한다.
- 존재하지 않는 슬롯 이름은 **빌드 에러**다.

### 7.4 2층 — 레이아웃 태그 (배치 채널)

레이아웃 태그는 **CSS flex/grid 시맨틱의 이름 바꾸기 층**이다. 새 레이아웃 모델이
아니며, 아래 표의 CSS로 정확히 컴파일된다 (브라우저 기본값 그대로).

| 태그 | 컴파일 결과 |
|---|---|
| `<row>` | `display:flex; flex-direction:row` |
| `<col>` | `display:flex; flex-direction:column` |
| `<grid>` | `display:grid` |
| `<stack>` | 겹쳐 쌓기 (단일 셀 grid + 자식 레이어링) — z축 배치, 배지·워터마크·오버레이 |
| `<spacer>` | `flex:1` 빈 요소 (밀어내기) |

**컨테이너 속성**

| 속성 | 값 | CSS |
|---|---|---|
| `gap` | 공간 토큰(`xs`~`2xl`) 또는 수치 | `gap` |
| `align` | `start·center·end·stretch·baseline` | `align-items` |
| `justify` | `start·center·end·between·around·evenly` | `justify-content` |
| `wrap` | `true` (값 없는 `wrap` 표기도 같은 뜻) | `flex-wrap: wrap` |
| `pad` | 공간 토큰 또는 수치 (1~4값) | `padding` |
| `cols` / `rows` | 수치(트랙 **개수**) 또는 트랙 목록 (`"2fr 1fr"`) — **`<grid>` 전용** | `grid-template-columns/rows` |
| `w` / `h` / `min-w` / `min-h` / `max-w` / `max-h` | 수치·`%`·`fill` (= 해당 축의 100%) | `width`/`height`/… |
| `ratio` | `16:9`, `1:1` | `aspect-ratio` |

**자식 속성** (컨테이너의 자식이 될 수 있는 컴포넌트/레이아웃 태그에 적용 가능 — `<li>`와
인라인 태그는 제외한다: 리스트는 PPTX에서 도형 하나이고 인라인은 박스가 아니라 런이다)

| 속성 | CSS |
|---|---|
| `grow` / `shrink` / `basis` | `flex-grow` / `flex-shrink` / `flex-basis` |
| `span` (grid 자식) | `grid-column: span N` |
| `self` | `align-self` |
| `order` | `order` |
| `at` (stack 자식) | 9지점 앵커(`top-left`…`bottom-right`) → `place-self` 조합 |
| `x` / `y` (stack·canvas 자식) | **부모 상대** 논리 좌표 절대 배치 (`position:absolute; left/top`) — 정말 필요할 때만 |
| `style` | **배치 전용 CSS 탈출구** (§7.9) |

**`<stack>` 시맨틱 (확정, §13-3)** — 단일 셀 grid로 구현한다: 모든 자식이
`grid-area: 1/1`에 겹치되 in-flow로 남으므로, stack의 크기가 가장 큰 자식에 맞춰
자동 결정된다 (absolute 전면 방식은 컨테이너 크기가 소실되어 배제). `at`은 해당 자식의
`place-self`로 컴파일되고, `x`/`y`는 그 자식만 absolute로 전환하되 좌표 기준은 항상
**부모 상대**다 — canvas 슬라이드의 직계 자식은 부모가 곧 스테이지이므로 스테이지 절대
좌표와 자연히 일치한다. z-순서는 **문서 순서**(뒤에 선언된 것이 위)이며 별도 z-index
속성은 없다. 이 순서가 그대로 PPTX 도형 순서가 된다.

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

- `gap="xl"`처럼 **공간 값이 토큰 어휘에 들어온다** — 원시 CSS 문자열로는 불가능했던
  일관성이다. 단, 공간 토큰은 PPTX 테마 파트에 대응 슬롯이 없다 (배치는 좌표로 구워진다 —
  `docs/decisions/0012-space-tokens-have-no-theme-slot.md`).
- `<grid cols="4" gap="lg">` 한 줄이 카드 그리드의 좌표 계산 전부를 대체한다.
- **레이아웃 태그는 PPTX 도형이 되지 않는다** — 자식만 절대 좌표 도형으로 남고 컨테이너는
  사라진다 (`<group>`의 `grpSp`와 갈라지는 지점 —
  `docs/decisions/0011-layout-containers-are-not-shapes.md`).

### 7.5 2층 — 컴포넌트 어휘와 PPTX 대응

| 컴포넌트 | 주요 속성 | 웹 렌더 | PPTX 렌더 (OOXML) |
|---|---|---|---|
| `<text>` | role, align, anchor | 텍스트 요소 | 텍스트 박스 (`sp`+`txBody`) |
| `<shape>` | type, id | 박스 div + 그림 svg | 프리셋 도형 (`sp`+`prstGeom`) |
| `<image>` | src, fit(cover/contain), frame | img | 그림 (`pic`) |
| `<arrow>` | from, to, label, heads, via | SVG 패스 (앵커 자동 계산) | 커넥터 (`cxnSp`) |
| `<line>` | cap, dash, dir | SVG 라인 | 커넥터 (화살촉 없음) — 장식 구분선 |
| `<group>` | (레이아웃 속성) | div | 그룹 (`grpSp`) |
| `<icon>` (P1) | src(svg), size | 인라인 SVG | **SVG 패스 → `custGeom` 도형 (편집 가능)**, 실패 시 이미지 폴백 |
| `<table>` (P1) | header-row, banding, cols | table | 테이블 (`graphicFrame`+`tbl`) |
| `<chart>` (P2) | type, data | 차트 렌더 | 네이티브 차트 또는 이미지 폴백 |

- **`<shape type>` v1 어휘 (12종)**: `rect`, `rounded-rect`, `ellipse`, `diamond`,
  `cylinder`, `chevron`, `arrow-right`, `hexagon`, `parallelogram`, `callout`,
  `star`, `cloud`. 모두 OOXML 프리셋 지오메트리의 부분집합이며, 확장 시에도 OOXML에
  존재하는 것만 추가해 번역 가능성을 지킨다.
- **`<arrow>`는 GUI PowerPoint보다 나은 지점이다**: `from`/`to`로 도형 `id`를 참조하면
  앵커를 자동 계산하고, 배치가 바뀌어도 연결이 유지된다.
- **`<arrow>` 앵커 시맨틱 (확정, §13-4)**: 기본은 **자동 앵커** — 두 도형의 중심을 잇는
  선이 지나는 **최근접 변의 중점**에 스냅한다. 명시 앵커 `from="db@top"`이 자동을
  오버라이드한다 (`@top`/`@right`/`@bottom`/`@left`/`@center`, v1은 이 5종). 경유점은
  `via="640,180 800,180"` — 공백 구분 논리 좌표점 목록이다. 자동/명시 앵커는 PPTX
  `cxnSp`의 연결 사이트로 매핑되어 PowerPoint에서 도형을 끌면 커넥터가 붙어 다니고,
  `via`가 있으면 꺾인 커넥터로 렌더하되 양 끝의 도형 연결은 유지한다.
- `<group>`은 레이아웃 태그와 달리 **PPTX 그룹 객체로 묶이는** 의미를 갖는다.
  단순 배치에는 `<row>`/`<col>`을, 하나의 편집 단위로 묶으려면 `<group>`을 쓴다.

### 7.6 외관 속성 (외관 채널)

외관은 CSS가 아니라 **컴포넌트의 프레젠테이션 속성**으로 선언한다. SVG가
`<rect fill="…" stroke-width="…">`로 외관을 속성 선언하는 것과 같은 원리다.

```html
<shape type="rounded-rect"
       variant="card-backdrop"
       fill="none"
       border="3 gradient(accent1 accent4 accent5)"
       radius="lg" />
```

| 속성 | 적용 대상 | 값 예시 |
|---|---|---|
| `variant` | 모든 컴포넌트 | 테마가 정의한 외관 번들 이름 (`card-header`) |
| `fill` | shape, group, table, slide | 토큰명(`surface`) · 리터럴(`#0F65F9`) · `gradient(…)` · `none` |
| `border` | shape, image | `<굵기> <스타일\|gradient(…)> <색>` 축약 (조각 3개·순서 고정) 또는 `none` |
| `radius` | shape(`type="rounded-rect"`), image | 토큰(`sm/md/lg`) 또는 수치 |
| `stroke` | line, arrow | `<굵기> <색\|gradient(…)>` |
| `cap` / `dash` / `heads` | line, arrow | `round`, `dashed`, `both` |
| `role` | text | 텍스트 역할 (`title`/`heading`/`body`/`caption`/`code`) |
| `font` / `size` / `weight` / `color` / `align` / `anchor` | text | `font="heading" size="24" weight="bold" color="fg"` |
| `shadow` / `opacity` | 모든 컴포넌트 | 토큰(`soft`) / `0.8` |
| `rotate` / `flip` | shape, image, group | `45`, `h` |

- **색 값은 토큰명 우선.** 토큰명으로 선언된 색은 PPTX에서 `schemeClr` 참조로 보존되어,
  **내보낸 파일이 PowerPoint 안에서도 테마 색 편집을 따라간다.** 리터럴로 쓴 색은
  `srgbClr`로 나가 테마 편집을 따라가지 않는다 — 그 차이가 토큰을 권하는 이유다.
- **해석 순서 (캐스케이드)**: 인라인 속성 > `variant` > 테마의 컴포넌트 기본값 >
  텍스트 역할/토큰. 병합은 엔진이 파싱 시점에 수행하며 결과는 타깃 무관이다.
- **`border` 축약은 조각 3개·순서 고정이다 (확정, I4)**: `<굵기> <스타일> <색>`. 순서를
  자유롭게 두면 `border="solid 3"`의 진단이 무엇이 빠졌는지 말할 수 없다. 스타일 어휘는
  `prstDash`에 대응이 있는 셋(`solid`/`dashed`/`dotted`)이고, "테두리 없음"의 표기는
  `none` **하나**다 (굵기 0은 에러 — 표기가 둘이면 렌더러마다 `w="0"`과 `noFill` 중 무엇을
  낼지 판단이 생긴다).
- **`radius`는 `type="rounded-rect"`에서만 쓴다 (확정, ADR 0019)**: 다른 프리셋에는 반경
  조정값을 실을 자리가 없어 웹만 둥글어진다. 테마 컴포넌트 기본값의 `radius`는 그 프리셋에서
  조용히 버려진다 (저자가 고칠 수 없는 진단을 내지 않는다).
- **`shadow`는 토큰만 받는다 (확정, ADR 0020)**: CSS 4수치와 `outerShdw` 파라미터가 1:1이
  아니라 같은 수치가 두 타깃에서 다르게 그려진다.
- **`opacity`는 채움·선·그림자 색의 알파다 (확정, ADR 0018)** — 합성 불투명도가 아니므로
  반투명 도형 안의 **글자는 흐려지지 않는다**.

**외관 값 문법 (확정, §13-5)**

- **수치 단위**: 무단위 수치는 전부 **논리 스테이지 px**다 (16:9 = 1280×720 기준,
  폰트 `size` 포함). 허용 접미사는 `%`(부모 기준) 하나이며, 그 외 단위(`pt`/`em` …)는
  파스 에러다. 타깃 변환은 엔진이 소유한다 — PPTX 좌표 `1px = 9525 EMU`, 폰트
  `pt = px × 0.75`. 저자와 지오메트리 패스가 같은 좌표계를 쓰므로 환산 분기가 없다.
- **그라디언트**: `gradient(색1 색2 [색3 …] [, <각도>deg])`. 색은 공백 구분·균등 스톱
  배치이고, 인자는 토큰명/리터럴만 받는다 (스톱 위치 지정·함수 중첩은 v2). 각도는 CSS
  `linear-gradient` 시맨틱(0deg = 아래→위, 시계방향)이며 생략 시 `180deg`(위→아래).
  OOXML `gradFill@ang`으로의 변환은 엔진 몫이다.
- **색 변형 함수**: `shade(색, %)` 어둡게 · `tint(색, %)` 밝게 · `alpha(색, 0~1)` 투명도.
  중첩은 불허한다 (v1). 토큰에 적용해도 `schemeClr` + `lumMod`/`lumOff`/`alpha`로
  테마 참조가 보존된다 (§9.4).

### 7.7 텍스트 — HTML 서브셋

리치 텍스트는 재발명하지 않는다. 블록 태그(`h1`~`h3`, `p`, `ul`/`ol`/`li`,
`blockquote`, `pre`)는 텍스트 역할(§8.3)의 축약이고, 인라인 태그(`strong`, `em`,
`code`, `a`, `br`, `span`)는 텍스트 런(run) 서식으로 PPTX까지 번역된다.

```html
<slide>
  <text slot="title">올해의 세 가지 변화</text>
  <ul>
    <li>모놀리스 분리 완료</li>
    <li reveal>배포 파이프라인 <strong>자동화</strong></li>
    <li reveal>관측성 스택 도입</li>
  </ul>
</slide>
```

- `<p>` ≡ `<text role="body">`, `<h2>` ≡ `<text role="heading">` — 같은 계약의 두 표기.
- `reveal` 속성: 클릭 시 순차 등장. `reveal="2"`로 순서 지정. PPTX에서는 모두 표시된
  정적 상태로 평탄화된다 (진입 애니메이션 매핑은 P2).

### 7.8 합성 컴포넌트

디자인된 슬라이드는 **동일한 도형 그룹의 반복**으로 만들어진다 (실물 덱 분석에서
카드 3도형 × 8회 반복 확인). `components/*.slide`에 재사용 단위를 정의한다.

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
<slide layout="canvas">
  <text slot="title">Responsible AI 8대 차원</text>
  <grid cols="4" gap="lg">
    <dim-card title="Controllability">시스템 행동을 감시·조종할 수단</dim-card>
    <dim-card title="Safety">유해 출력과 오용 방지</dim-card>
    <!-- … -->
  </grid>
</slide>
```

- 합성 컴포넌트는 **파싱 시 코어 컴포넌트로 전개되어** 덱 모델에 들어간다.
  렌더러는 합성의 존재를 모르므로 PPTX 변환을 공짜로 얻는다.
- 이름에 하이픈을 포함해야 한다 (코어 어휘 확장과의 충돌 방지).
- 팀이 브랜드 테마와 함께 컴포넌트 라이브러리를 배포하는 생태계의 기반이며,
  테마도 컴포넌트를 함께 제공할 수 있다.
- **파라미터 계약 (확정, §13-7)**: 명명 `<slot>` + **속성 파라미터**까지만 지원한다.
  사용처의 속성 중 정의의 슬롯 이름과 일치하는 것(`title="…"`)은 해당 슬롯을 텍스트로
  채우고, 배치 자식 속성(`grow`/`span`/`self`/`order` …)은 전개 결과의 루트 요소에
  전달되며, 그 밖의 속성은 빌드 에러다. **조건·반복·표현식은 지원하지 않는다 —
  "SOML에는 제어 흐름이 없다"가 템플릿 언어화를 끊는 선이다.** 반복 생성이 필요하면
  SOML을 생성하는 프로그램을 쓴다 (P2 데이터 주도 덱).

### 7.9 탈출구

**배치 CSS 탈출구** — 레이아웃 속성이 못 덮는 배치는 `style`로 통과시킬 수 있다.
배치는 어차피 좌표로 구워지므로 PPTX 충실도를 해치지 않는다. 단, `style`에 외관
속성(색·테두리 등)을 쓰면 **웹 전용**이 되고 dev 모드가 경고한다.

**최후 탈출구 `<raw>`** — `canvas` 슬라이드 안에서 임의의 HTML/SVG/CSS를 쓸 수 있다.
웹에서는 그대로 렌더링되지만, **PPTX에서는 고해상도 이미지로 래스터화**되어 삽입된다
(편집 불가) + 빌드 경고. 문서화 원칙: "`<raw>`를 쓰는 순간 그 블록은 PPTX에서
그림이 된다. 코어 어휘로 표현 가능한지 먼저 검토하라."

---

## 8. 테마 시스템 — 디자인 시스템으로서의 테마

### 8.1 테마가 소유하는 것

1. **디자인 토큰 값** — 색·폰트·스케일·공간·형태 (3층)
2. **컴포넌트 외관** — 컴포넌트별 기본값 + 이름 붙은 variant + 텍스트 역할 (2층)
3. **레이아웃 구현** — 표준 카탈로그 각 항목의 슬롯 배치 (1층, 배치 전용 CSS)
4. **크롬(chrome)** — 로고, 브랜드 스트립, 저작권 푸터 등 전 슬라이드 장식.
   `theme.json`에 **선언적으로** 정의한다. 선언적이어야 웹과 PPTX 슬라이드 마스터
   양쪽에 렌더링된다.

### 8.2 패키지 구조

```
my-theme/
├── theme.json          # 매니페스트: 이름, extends, 지원 레이아웃, 크롬 선언, 폰트
├── tokens.json         # 디자인 토큰 (단일 소스 → CSS 변수 + PPTX 테마 파트)
├── components.json     # 컴포넌트 기본 외관 + variant + 텍스트 역할 (외관 스키마)
├── layouts/            # 레이아웃별 슬롯 배치 — 배치 전용 CSS
│   ├── title.css
│   ├── two-column.css
│   └── corp-hero.css   # 테마 자체 레이아웃
├── components/         # (선택) 테마 동반 합성 컴포넌트 (.slide)
├── web/
│   └── transitions.css # 웹 전용 전환/등장 효과
├── assets/             # 폰트, 배경, 로고
└── preview.html        # (자동 생성) 갤러리 프리뷰
```

**`layouts/*.css`는 배치 전용이다.** 외관 속성이 레이아웃 CSS에 들어가면 웹 전용으로만
동작하고 PPTX 마스터에 반영되지 않으므로 `balpyo theme lint`가 검출해 경고한다.
슬롯 안 텍스트의 외관은 텍스트 역할로 선언한다.

**테마는 JS를 포함할 수 없다 (v1 확정, §13-11).** 테마 패키지는 JSON + CSS + 에셋의
**데이터 전용 포맷**이다 — `balpyo theme get`으로 받은 원격 테마가 임의 코드 실행
경로가 되어서는 안 되고(보안), "컴파일 대상 없음" 서사(§8.6)도 지켜진다. 전환/등장
효과는 `web/transitions.css`의 선언적 CSS로 한정하며, 웹 전용 고급 효과의 수요가
실증되면 v2에서 샌드박스 훅을 검토한다.

### 8.3 `components.json` — 외관 스키마

```jsonc
{
  "shape": {
    "default":  { "fill": "surface", "border": "1 solid muted", "radius": "md" },
    "variants": {
      "card-backdrop": { "fill": "none", "border": "3 gradient(accent1 accent4 accent5)", "radius": "lg" },
      "card-header":   { "fill": "none", "border": "none", "radius": "sm" },
      "stat-card":     { "fill": "surface", "radius": "lg", "shadow": "soft" }
    }
  },
  "line":  { "default": { "stroke": "2 solid muted", "cap": "round" } },
  "arrow": { "default": { "stroke": "2 solid fg", "heads": "tail" } },
  "text": {
    // 텍스트 역할: HTML 서브셋(1층)과 <text role>(2층)이 공유하는 타이포그래피 계약
    "title":   { "font": "heading", "size": "h1", "color": "fg" },
    "heading": { "font": "heading", "size": "h2", "color": "fg" },
    "body":    { "font": "body",    "size": "body", "color": "fg" },
    "caption": { "font": "body",    "size": "caption", "color": "muted" },
    "code":    { "font": "mono",    "size": "body" }
  }
}
```

- **웹 컴파일**: 엔진이 `tokens.json` + `components.json`에서 CSS를 생성한다.
  그라디언트 테두리처럼 CSS로 어색한 항목은 **엔진이 구현을 소유**한다 (SVG 오버레이
  등) — 테마 제작자가 핵을 쓸 일이 없다.
- **PPTX 컴파일**: 토큰 → 테마 파트(clrScheme/fontScheme), 텍스트 역할 → 마스터
  텍스트 스타일, 컴포넌트 외관 → 도형 속성 (토큰 참조는 `schemeClr`로 보존).
- **variant 어휘는 표준화하지 않는다 (확정, §13-6)**: 계약은 캐스케이드 메커니즘까지다.
  대신 공식 테마 3종이 같은 **권장 어휘**(`card`, `card-header`, `card-body`,
  `card-backdrop`, `stat-card`, `badge`)를 공유해 사실상의 관례를 만든다. 테마에 없는
  variant는 빌드 에러가 아니라 **컴포넌트 기본 외관으로 폴백 + 경고**다 (`--strict`로
  에러 승격 가능) — 콘텐츠의 테마 간 이식성을 지키기 위해서다.
- **컴포넌트 variant는 그 컴포넌트 자신의 외관만 담는다 (확정, ADR 0022)**: 도형 variant에
  `font`/`size`/`weight` 같은 텍스트 속성을 넣지 않는다. 넣으면 캐스케이드가 트리를 타고
  내려가는 **상속**이 되고, 상속은 한 요소 안에서 끝나는 병합과 다른 두 번째 메커니즘이다
  ("우선순위는 병합 함수 한 곳에 있다"는 계약의 예외가 된다). 카드 제목의 타이포그래피는
  자식 텍스트가 소유한다 (`<h3>`의 태그 기본값 — §7.7). variant는 `type`도 정하지 않는다:
  테마가 프리셋을 바꾸면 저자가 적은 도형이 테마에 따라 다른 그림이 된다.

### 8.4 토큰 어휘 (표준)

```jsonc
// tokens.json — 웹: --balpyo-* CSS 변수 / PPTX: 테마 파트
{
  "color":  { "bg": "…", "fg": "…", "accent": "…", "accent2": "…", "muted": "…", "surface": "…" },
  "font":   { "heading": "…", "body": "…", "mono": "…" },
  "scale":  { "h1": "…", "h2": "…", "h3": "…", "body": "…", "caption": "…" },
  "space":  { "xs": "…", "sm": "…", "md": "…", "lg": "…", "xl": "…", "2xl": "…", "slidePadding": "…" },
  "shape":  { "radius-sm": "…", "radius-md": "…", "radius-lg": "…", "border": "…", "shadow-soft": "…" }
}
```

`space` 스케일은 레이아웃 태그의 `gap`/`pad` 값 어휘와 직결된다 — 배치조차 테마의
리듬을 따라간다.

### 8.5 테마 워크플로우

```bash
balpyo theme new my-theme      # 기본 테마 베이스로 스캐폴딩 (내장 테마를 풀어씀)
balpyo theme dev my-theme      # 갤러리 프리뷰: 레이아웃 + 컴포넌트/variant 시트, 핫 리로드
balpyo theme lint my-theme     # 토큰/레이아웃/variant 커버리지 + 배치·외관 경계 위반 검사
balpyo theme pack my-theme     # 배포용 tarball 빌드
balpyo theme get <git-url>     # 커뮤니티 테마를 themes/로 가져오기
```

- 상속: `theme.json`의 `extends`로 기존 테마의 토큰/variant/레이아웃 일부만 오버라이드.
  내장 테마도 상속 대상이 된다 (`"extends": "minimal"`).
- 갤러리 프리뷰는 표준 레이아웃 전체 + 컴포넌트 어휘 전체(도형 12종, 선/화살표 변형,
  텍스트 역할, 정의된 variant)를 한 화면에서 **웹 렌더와 PPTX 렌더 나란히** 검증한다.

### 8.6 테마 배포와 공식 테마

테마는 JSON + CSS + 에셋뿐이고 **컴파일 대상이 없으므로 패키지 매니저가 필요 없다.**
`theme` 필드는 세 형태를 받는다:

| 형태 | 예 | 해석 |
|---|---|---|
| 내장 테마 이름 | `"minimal"` | 바이너리에 내장 — 설치·네트워크 불필요 |
| 로컬 경로 | `"./themes/corp"` | 프로젝트 내 테마 (사내 브랜드 테마의 기본형) |
| git URL / tarball | `"github.com/user/balpyo-theme-corp"` | `balpyo theme get`이 `themes/`에 캐시 |

**공식 테마 3종은 바이너리에 내장된다** (`minimal`·`dark`·`paper`). `balpyo init` 직후
아무 설치 없이 예쁜 결과가 나오는 것이 첫인상을 지배하므로, 기본 경로에서 네트워크를
제거한다. `minimal`이 기본값이자 폴백 기준이다.

- `minimal` — 여백 중심 미니멀 (기본 테마)
- `dark` — 다크 모드 기술 발표용
- `paper` — 문서 느낌의 밝은 테마

"balpyo로 뽑은 PPTX는 기본 테마만 써도 예쁘다"가 마케팅 자산이다.

---

## 9. 렌더러

### 9.1 웹 렌더러 (기준 타깃)

웹은 **기준(reference) 타깃**이다. 발표 경험의 1급 시민이며 다른 타깃의 시각적
정답지 역할을 한다.

- SOML → HTML 변환 + 외관 스키마 → 생성 CSS. 레이아웃 태그의 배치 CSS는 그대로 통과.
- **P0**: 키보드/터치 네비게이션, `reveal` 순차 등장, URL 해시 딥링크(`#/5`, `#/id`)
  + `hashchange` 구독, 화면 크기 무관 스케일링, 오버뷰 모드
- **P1**: 발표자 뷰(노트+타이머+다음 슬라이드), 프로그레스 바, 챕터 내비게이션
- **P2**: 원격 컨트롤, 청중 팔로우 모드
- 런타임은 프레임워크 무의존 vanilla TS (~10KB 목표)로 작성하고 **빌드 산출물을
  바이너리에 내장**한다 (엔진 언어와 무관한 에셋). 산출물은 단일 `index.html`
  (+assets) — 파일로 열어도 동작해 오프라인 발표가 가능하다.

### 9.2 `balpyo server`

웹 렌더러 위에 개발 편의를 얹은 뷰어.

- `slides/`·`components/`·테마 파일 변경 감시 → WebSocket 핫 리로드. **현재 보고 있는
  슬라이드 위치를 유지**한다.
- 검증 경고 오버레이: 미지원 외관 CSS, 폴백된 레이아웃, `<raw>` 사용, 오버플로 감지.
- 발표자 뷰(`/presenter`), 오버뷰, 슬라이드별 딥링크.

### 9.3 PPTX 렌더러 — 지오메트리 패스

핵심 기술 전략: **외관은 스키마에서 직행하고, 브라우저는 배치 계산에만 쓴다.**

```
덱 모델 (외관 해석 완료)
   │
   ├── ① 지오메트리 패스: 헤드리스 브라우저에서 웹 렌더러와 동일한 HTML/CSS로 렌더링
   │      → 요소별 getBoundingClientRect → EMU 좌표 변환. 읽는 것은 좌표뿐.
   │
   ├── ② 외관 직행: 해석된 외관 스키마 → OOXML 속성 매핑 (§9.4)
   │      토큰 참조는 schemeClr로 보존 → 산출물이 PPTX 테마 편집을 따라감
   │
   ▼
③ OOXML 조립: 마스터/레이아웃/테마 파트 + 슬라이드별 네이티브 도형 (xfrm=①, 외관=②)
   │
   ▼
deck.pptx (PowerPoint/Keynote에서 열어 편집 가능)
```

- computed-style 역추론은 없다. 외관은 스키마에서 양 타깃으로 순방향 컴파일되므로
  **손실 지점이 없고**, 지오메트리 패스는 좌표 읽기라는 단일 책임만 갖는다.
- 같은 엔진(브라우저)이 웹과 PPTX의 배치를 계산하므로 두 타깃의 배치 일치가
  구조적으로 보장된다.
- 텍스트 재배치 대책 — **기본값 확정 (§13-1): "박스 크기 고정 + 오버플로 허용"**.
  텍스트 박스는 측정된 크기에 약간의 안전 여유를 더해 명시 고정하고 PowerPoint
  autofit을 끈다. autofit 위임은 폰트 크기를 바꿔 웹과의 시각 일치를 깨므로 배제한다.
  허용 오차의 수치 임계값은 M0.5 스파이크에서 측정해 확정한다.
- 폰트 — **v1은 PPTX에 폰트를 임베드하지 않는다 (§13-10)**. 테마가 폰트를
  이름 + 폴백 스택으로 명시해 메트릭 차이를 최소화하고, 웹 타깃은 테마 에셋의 woff2를
  쓸 수 있다. 임베드는 라이선스 문제 때문에 P2의 opt-in 옵션(`embedFonts`)으로 미루며,
  라이선스 책임은 사용자에게 고지한다. 공식 테마는 광범위하게 설치된 폰트를 기준으로
  설계한다.
- 1층 레이아웃 슬라이드는 슬롯 → **PPTX 플레이스홀더**로, 테마 크롬은 **슬라이드
  마스터 고정 요소**로, `<note>`는 **노트 슬라이드**로 매핑한다.
- `--mode image`는 충실도 100%·편집 불가 폴백 모드다.

**지오메트리 백엔드**: 시스템 브라우저(Chrome/Edge 자동 감지)를 **CDP로 직접 제어**한다
(Go: `chromedp`/`go-rod`). 브라우저 다운로드 없음, 콜드 스타트 ~0.6초, 슬라이드당
~1.5ms — 빌드 성능 병목이 아니다. 브라우저 의존은 지오메트리만의 문제가 아니라
PDF 인쇄(`Page.printToPDF`)와 `<raw>`/`<chart>` 래스터화(`Page.captureScreenshot`)가
같은 CDP 세션을 재사용하므로, 세 기능이 하나의 의존성으로 해결된다. 시스템 브라우저가
없는 환경에는 Chromium 내려받기를 opt-in으로 안내한다. 백엔드는
`(스타일된 슬라이드 트리) → 좌표` 인터페이스로 **플러그블**하게 설계한다 (§13-8).

### 9.4 외관 스키마 v1 — 타깃별 컴파일 매핑

이 표가 곧 외관 스키마의 v1 스코프다.

| 외관 속성 | 웹 (생성 CSS) | PPTX (OOXML) |
|---|---|---|
| `fill` (단색/토큰) | `background-color: var(--balpyo-*)` | `solidFill` (+`schemeClr`) |
| `fill=gradient(…)` | `background-image: linear-gradient(…)` | `gradFill` |
| `border` (단색) | `border` | `ln` |
| **`border=… gradient(…)`** | 엔진 구현 (SVG 오버레이/mask — CSS `border-image`는 radius와 조합 불가) | `ln` + `gradFill` |
| **텍스트 `color=gradient(…)`** | `background-clip: text` | `rPr` + `gradFill` |
| `radius` | `border-radius` (SVG 기하는 `rx`) | `prstGeom` 조정값 |
| `font`/`size`/`weight`/`color`/`align` | `font-*`, `color`, `text-align` | `rPr` / `pPr` |
| **색 변형** (`shade()`, `tint()`, `alpha()`) | `color-mix()` / rgba | `schemeClr` + `lumMod`/`lumOff`/`alpha` |
| `cap` / `dash` | `stroke-linecap`, `stroke-dasharray` | `ln@cap`, `prstDash` |
| `rotate` / `flip` | `transform` | `xfrm@rot`, `flipH`/`flipV` |
| `anchor` / wrap / autofit | flex 정렬 / `white-space` | `bodyPr` anchor/wrap/autofit |
| `opacity` | 그림 svg의 `opacity` (색 알파와 등가 — 글자는 흐려지지 않는다) | 채움·선·그림자 색 안의 `a:alpha` |
| `shadow` | `filter: drop-shadow` (실루엣을 따라간다 — `box-shadow`는 박스 사각형을 따른다) | `effectLst` outerShdw |
| 인라인 서식 (`strong`/`em`/`code`/`a`) | `<strong>` 등 | 텍스트 런 `rPr` / `hlinkClick` |
| 위치·크기 (지오메트리 패스 결과) | (브라우저가 원본) | `xfrm` (EMU) |

실물 기업 덱 역분석 결과 "디자인된 느낌"은 굵은 글씨 항목(그라디언트 3종, 토큰 색
변형)에 집중되어 있었다. v1 스코프에 반드시 포함한다.

스키마 밖의 시각 효과(filter, clip-path, blend mode 등)는 저자 CSS로 쓸 수 있으나
**웹 전용**이며, dev 모드가 "PPTX에 번역되지 않음"을 경고한다.

### 9.5 PDF 렌더러

웹 렌더 결과를 헤드리스 브라우저로 인쇄한다 (슬라이드 1장 = 1페이지, `reveal`은
전개 상태로 평탄화, 노트 포함 여부 옵션). 지오메트리 패스의 브라우저 의존성을 재사용한다.

---

## 10. CLI 명세

```bash
balpyo init <name>                     # 프로젝트 스캐폴딩
balpyo new <name> [--layout content]   # slides/에 다음 번호로 슬라이드 생성
balpyo server [--port 4321] [--open]   # 라이브 뷰어 + 핫 리로드
balpyo build [--target html|pptx|pdf] [--out dist] [--theme …] [--mode native|image]
balpyo theme new|dev|lint|pack|get …   # 테마 워크플로우 (§8.5)
balpyo version                         # 버전 + 감지된 브라우저 경로
```

### 10.1 저작 품질 명령군 — `check` / `format` / `lsp`

세 명령은 **하나의 파서·검증기 코어를 공유**한다. 표면만 다르므로 규칙이 서로
어긋날 수 없다 — 에디터에서 본 경고와 CI가 막는 경고가 항상 같다.

```bash
balpyo check [--strict]                # 검증: 문법·슬롯·미지원 외관·오버플로
balpyo format [paths…] [--check]       # 정규 형식으로 정렬 (--check: 검사만)
balpyo lsp                             # 에디터용 언어 서버 (stdio)
```

| 명령 | 형태 | 소비자 |
|---|---|---|
| `check` | 단발 실행 → 진단 출력 + 종료 코드 | CI, pre-commit, 터미널 |
| `format` | 파일 제자리 수정 (또는 `--check`로 검사) | 저장 시 자동 실행, CI |
| `lsp` | 상주 서버 (LSP over stdio) | 에디터 — **입력 중 실시간** 진단·완성·호버 |

**`balpyo format`의 정규 형식**: 들여쓰기, 속성 순서(구조 `id`/`layout`/`slot` →
배치 → 외관), 빈 요소 자기 종료(`<arrow … />`), 속성값 인용 부호 통일.
**주석과 텍스트 내용은 손실 없이 보존**한다. 포매터가 있으면 팀 저작에서 diff 노이즈가
사라지고, 생성기·AI 에이전트가 뱉은 SOML도 즉시 사람이 읽는 형태로 정규화된다.

**`balpyo lsp`가 스키마(XSD/JSON)보다 더 아는 것** — 해석된 테마를 알기 때문에:

- `layout="` → 테마 자체 레이아웃까지 포함한 카탈로그
- `slot="` → **선택된 레이아웃이 실제로 정의한** 슬롯만
- `variant="` → 그 테마 `components.json`의 variant 이름
- `fill=`/`color=` → 테마 색 토큰, `gap=` → 공간 스케일 (호버에 해석된 값 표시)
- 정의로 이동: `<dim-card>` → `components/dim-card.slide`, `<arrow from="web">` → `<shape id="web">`
- 문법으로는 못 잡는 경고: "이 `style` 선언은 외관 CSS라 PPTX에 반영되지 않음", 오버플로

즉 스키마는 **문법**을, LSP는 **테마 계약**을 발견 가능하게 만든다. 빌드 전에
`variant="stat-card"`가 이 테마에 없다는 걸 알게 되는 것이 목표다.

### 10.2 공통 규약

- `--target` 다중 지정 가능: `balpyo build --target html,pptx`.
- 모든 옵션은 `balpyo.json`으로 고정 가능 — `balpyo build` 한 번으로 재현 가능한
  빌드 (CI 친화적).
- 종료 코드: 검증 에러 시 비영(non-zero). 경고는 0이지만 `--strict`로 승격 가능.
  `format --check`도 미정규 파일이 있으면 비영을 반환한다.
- 바이너리 하나가 모든 서브커맨드(뷰어·렌더러·포매터·언어 서버)를 담는다 —
  별도 설치물이 없다.

---

## 11. 기능 요구사항 우선순위

### P0 — MVP

- [ ] **SOML 파서**: 태그·속성 카탈로그, 닫힌 어휘 검증, 위치 정보 포함 에러 리포팅
- [ ] 덱 모델 스키마 v1 + 정규화 (슬라이드 수집·정렬·챕터, 슬롯, reveal, 노트)
- [ ] **외관 스키마 v1 + 스타일 해석기** (인라인 > variant > 기본값 > 역할/토큰)
- [ ] **레이아웃 태그**: `row`, `col`, `grid`, `stack`, `spacer` + 속성→CSS 매핑
- [ ] **컴포넌트 코어**: `text`, `shape`(12종), `image`, `arrow`, `line`, `group`
- [ ] HTML 텍스트 서브셋 → 텍스트 역할/런 매핑
- [ ] 표준 레이아웃 카탈로그 11종 (`canvas` 포함) + 폴백 메커니즘
- [ ] 테마 패키지 포맷: `tokens.json` + `components.json` + 배치 전용 레이아웃 CSS
- [ ] 외관 스키마 → 웹 CSS 컴파일러 (그라디언트 테두리 등 엔진 소유 구현 포함)
- [ ] `balpyo init` / `new` / `server`(핫 리로드) / `build --target html` / `check` / `format`
- [ ] 에디터 지원 1단계 (거의 무료): `.slide` → XML 파일 연결 안내 +
  SOML 스키마(XSD/JSON) 배포 → 하이라이팅·태그 매칭·기본 속성 완성
- [ ] 웹 런타임: 네비게이션, reveal, 스케일링, 딥링크
- [ ] 내장 기본 테마 `minimal` (variant 세트 포함)
- [ ] 릴리스 파이프라인: 크로스 컴파일(darwin/linux/windows × amd64/arm64),
  GitHub Releases, Homebrew tap

### P1 — 렌더 엔진의 증명

- [ ] **PPTX 렌더러**: 이미지 모드 → 지오메트리 패스 기반 네이티브 모드
  - [ ] `tokens.json` → PPTX 테마 파트, 텍스트 역할 → 마스터 텍스트 스타일
  - [ ] 외관 스키마 → OOXML 매핑 (§9.4 전체, `schemeClr` 보존 포함)
  - [ ] 1층 레이아웃 → 슬라이드 마스터/플레이스홀더 매핑
  - [ ] `<raw>`/미지원 요소 이미지 폴백 + 경고
- [ ] PDF 내보내기
- [ ] **합성 컴포넌트** (`<define>`/`<slot>`) — 컴포넌트 라이브러리 생태계의 기반
- [ ] `balpyo theme new/dev/lint/pack`, 테마 상속
- [ ] 발표자 뷰, 오버뷰 모드, 챕터 내비게이션
- [ ] `<icon>`: SVG 패스 → `custGeom` 네이티브 변환 파이프라인 (아이콘 라이브러리
  번들 없음 — `src` 로컬 SVG만, §13-12)
- [ ] `<table>` (헤더 행·밴딩 스타일 훅), 코드 하이라이팅 (Chroma 토큰 → 웹 span +
  PPTX 텍스트 런, 테마 토큰 연동)
- [ ] 내장 테마 `dark`, `paper` + `balpyo theme get`(git 테마 가져오기)
- [ ] 에디터 지원 2단계: **`balpyo lsp`** (단일 바이너리 서브커맨드) — 진단·완성·호버·
  정의 이동 4종만 구현하고 나머지 LSP 스펙은 미광고. Neovim/Zed/Helix는 설정 몇 줄로
  즉시 연결되므로 배포물이 없다
- [ ] VS Code 확장: LSP 얇은 클라이언트 (바이너리는 PATH에서 찾음) + 마켓플레이스 배포

### P2 — 생태계

- [ ] `<chart>` (웹: 차트 렌더 / PPTX: 네이티브 차트 검토, 초기엔 이미지)
- [ ] `reveal` → PPTX 진입 애니메이션 매핑
- [ ] 데이터 주도 덱 생성 (`slides = f(data)`) 공식 지원 + 템플릿 문법 검토
- [ ] **언어 래퍼**: npm/pip 패키지가 바이너리를 내려받아 감싸는 얇은 층
  (`npx balpyo` 유지, Python 파이프라인에 SOML 생성 헬퍼 제공) — 엔진 언어와 무관하게
  각 생태계에서 자연스럽게 쓰이도록
- [ ] Markdown → SOML 변환기 (goldmark)
- [ ] 테마/컴포넌트 라이브러리 레지스트리·갤러리 웹사이트
- [ ] Mermaid 등 다이어그램 → 컴포넌트 어휘 변환 플러그인
- [ ] 원격 컨트롤 / 청중 모드
- [ ] 인프로세스 flex 레이아웃 백엔드 (무브라우저 CI/고속 경로) — **도입 게이트
  3조건 (§13-8)**: ①텍스트 측정이 골든 코퍼스에서 §13-1 허용 오차 내 ②flex/grid
  웹 기본값 시맨틱의 골든 좌표 테스트 100% 일치 ③무브라우저 CI 수요 실증.
  셋을 만족하기 전에는 착수하지 않는다 (플러그블 인터페이스만 유지)

---

## 12. 기술 아키텍처

### 12.1 언어 선택: Go

**엔진은 Go로 구현한다.** SOML이 자체 언어가 된 순간 JS 생태계에 남을 이유(HTML 파서
재사용)가 사라졌고, 남은 요구사항이 Go의 강점과 정확히 겹친다.

| 요구사항 | Go에서의 해법 |
|---|---|
| 단일 바이너리 배포 (G7) | 정적 링크 + 크로스 컴파일. 런타임·패키지 매니저 불필요 |
| 공식 테마·웹 런타임·템플릿 내장 | `//go:embed` — 설치 없이 첫 결과물이 나온다 |
| 지오메트리 패스 · PDF · 래스터화 | `chromedp`/`go-rod`로 시스템 브라우저를 CDP 직접 제어 (네이티브, 드라이버 설치 없음) |
| OOXML 조립 | 표준 라이브러리 `archive/zip` + `text/template`로 XML 파트 직조립 |
| 코드 하이라이팅 | Chroma — **토큰 스트림 API**라 웹 span과 PPTX 텍스트 런 양쪽에 쓸 수 있다 |
| dev 서버 + 핫 리로드 | `net/http` + WebSocket, 파일 워처 |
| CI 친화성 | 다운로드 1개. `node_modules`·락파일·가상환경 없음 |

- **엔진 언어 ≠ 생태계 언어.** 웹 런타임은 여전히 TS로 쓰고 빌드 산출물을 내장한다.
  각 생태계 사용자는 바이너리를 감싸는 얇은 npm/pip 래퍼로 흡수한다 (P2).
- 검토 후 배제: **Node**(엔진·런타임 언어 통일은 매력적이나 런타임 의존과 브라우저
  드라이버 설치가 배포 서사를 해친다), **Python**(CLI 배포가 가장 약하고, Playwright
  파이썬 바인딩이 Node 드라이버를 요구해 지오메트리 패스의 의존성 서사가 무너진다).

### 12.2 구성

- **파이프라인**: `slides/**/*.slide` 수집·정렬 → SOML 파싱(관용적 HTML 파서가 아닌
  **엄격 파서**, 위치 정보 보존) → 덱 모델 정규화 → 합성 컴포넌트 전개 →
  외관 해석 → 검증 → 타깃 렌더러 실행
- **SOML 파서**: `encoding/xml`에 의존하지 않고 **직접 작성한 토크나이저**를 쓴다.
  닫힌 어휘 검증과 정확한 line/col 진단이 언어의 핵심 가치이므로 파서가 위치 정보를
  1급으로 보존해야 한다 (~수백 줄 규모). 파서는 **처음부터 세 소비자를 전제로**
  설계한다 — 이것이 나중에 붙일 수 없는 유일한 설계 결정이다:
  | 소비자 | 파서에 요구하는 것 |
  |---|---|
  | `build` / `check` | 첫 에러에서 멈춰도 무방. 정확한 위치 |
  | `lsp` | **에러 복구** — 입력 중 문서는 항상 깨져 있다(`<shape type="round`). 치명적 실패 대신 **부분 AST + 진단 목록**을 내고, 닫히지 않은 노드의 위치도 유지 |
  | `format` | **무손실 파스** — 주석·공백·인용 부호 같은 트리비아를 보존해야 원문을 되돌려 쓸 수 있다 |
  → 결론: AST가 아니라 **트리비아를 보존하는 관용 파서 + 그 위의 정규화 층**으로
  구성한다. 빌드만 생각해 관용성 없는 파서를 먼저 만들면 `format`/`lsp` 단계에서
  재작성이 된다.
- **웹 렌더러**: SOML→HTML 변환, 외관 스키마→CSS 생성, 런타임 JS 인라인 → 자체 완결형
  정적 번들. CSS cascade layers(`@layer balpyo.base, balpyo.theme, balpyo.author`)로
  기본 < 테마 < 저자 우선순위를 보장한다.
- **PPTX 렌더러**: CDP로 지오메트리 패스(좌표 전용) → 해석된 외관과 병합 →
  **OOXML 직접 생성**(템플릿 기반 XML 조립). 기존 pptx 라이브러리는 그라디언트
  선/텍스트 등 시그니처 요소의 API가 없어 어느 언어에서도 채택하지 않는다.
- **`style` 탈출구 검증**: CSS 파서(`tdewolff/parse/css` 등)로 선언을 훑어
  배치 속성 allowlist 밖의 외관 속성을 경고한다 (§7.9의 경계를 기계적으로 집행).
- **패키지 경계** (Go 모듈 내부 패키지): `soml`(파서+덱 모델) / `core`(해석·검증) /
  `render/web` / `render/pptx` / `render/pdf` / `cmd/balpyo`. `soml`과 덱 모델은
  외부에서 임포트 가능한 공개 API로 유지해 제3자 도구(생성기, 에이전트)가 재사용한다.
  덱 모델의 언어 중립 계약은 JSON Schema로 배포한다.
- **dev 서버**: `net/http` + WebSocket 핫 리로드 + 웜 브라우저 세션 유지.

---

## 13. 확정된 설계 결정 (구 "열린 질문")

v1.0에서 열어 두었던 질문 12건을 모두 확정했다 (2026-08-16). 결정의 본문은 각 섹션에
녹아 있고, 이 절은 **질문 → 결정의 색인**이다.

1. **텍스트 줄바꿈 허용 오차** → **"박스 크기 고정 + autofit off + 오버플로 허용"이
   기본값** (§9.3). autofit 위임은 폰트 크기를 바꿔 웹↔PPTX 시각 일치를 깨므로 배제.
   측정 크기에 안전 여유를 더해 고정하며, 허용 오차의 **수치 임계값만** M0.5 스파이크에서
   측정해 이 항목에 기입한다.

   **측정한 수치 (Iteration 1, `01_text` 한글·영문 혼합 3장 — 2026-08-18, NOTES.md 01/07):**
   - **안전 여유 = 박스 폭 +4%** (정렬이 닻을 내린 변은 고정 — `render/pptx.TextBoxRect`).
     근거: LibreOffice는 한글↔라틴 스크립트 경계에 간격을 넣어 혼합 문자 줄을 최대
     **+3.3%** 넓게 그린다 (OOXML로 끌 수 없는 엔진 동작). 여유가 이보다 작으면 줄이
     하나 늘어난다 — 여유 2.4%였던 본문이 실제로 갈라졌다.
   - **줄간격은 `spcPct`(배수)가 아니라 `spcPts`(절대값)** — 배수는 CSS(폰트 크기 기준)와
     LibreOffice(폰트 고유 줄 높이 기준)의 기준이 달라 같은 150%가 글줄을 8~13px 밀었다.
   - **회귀 감시 임계값** (`internal/testkit`의 잉크 띠·픽셀 비교): 줄 수 차이 **0**,
     글줄 잉크 세로 중심 **±6px** (측정 +2~5px — 첫 기준선 배치는 엔진 소관), 글줄 잉크 폭
     **±4%** (측정 −1.5~+3.3%), 픽셀 diff **채널 32 / 비율 2.5%** (측정 최대 2.1%).
2. **`.slide` / `.soml` 두 확장자** → **`.slide` 단일 확장자로 통일** (§7.1). 합성
   컴포넌트도 `.slide`를 쓴다. 파일의 역할은 루트 요소(`<slide>` vs `<define>`)와
   디렉토리가 결정하고 파서가 검증한다. 에디터·스키마·LSP가 파일 타입 하나만 다루면
   되고, 사용자가 배울 규칙도 하나 준다.
3. **`<stack>` 시맨틱** → **단일 셀 grid 겹침** (§7.4). `at`은 `place-self`, `x`/`y`는
   해당 자식만 absolute이며 좌표 기준은 항상 **부모 상대** (canvas 직계 자식은 부모가
   스테이지라 스테이지 절대와 일치). z-순서는 문서 순서(뒤 = 위), z-index 속성 없음 —
   그대로 PPTX 도형 순서가 된다.
4. **`<arrow>` 앵커 시맨틱** → **자동 앵커(중심 연결선의 최근접 변 중점)가 기본**,
   `from="db@top"` 명시 앵커(4변 + center)가 오버라이드, `via`는 공백 구분 논리
   좌표점 목록 (§7.5). 앵커는 PPTX `cxnSp` 연결 사이트로 매핑된다.
5. **외관 값 문법** → 확정 (§7.6 "외관 값 문법"). `gradient(색… [, 각도deg])` —
   균등 스톱, 기본 180deg, CSS 각도 시맨틱. 색 변형은 `shade`/`tint`/`alpha` 3종,
   중첩 불허. 수치는 **논리 스테이지 px 단일 단위** (`%`만 예외, 그 외 단위는 파스
   에러). PPTX 변환 계수: `1px = 9525 EMU`, 폰트 `pt = px × 0.75`.
6. **variant 어휘의 표준화** → **계약에 포함하지 않는다** (§8.3). 공식 테마 3종이
   권장 어휘(`card`, `card-header`, `card-body`, `card-backdrop`, `stat-card`,
   `badge`)를 공유해 관례를 만들고, 미정의 variant는 기본 외관 폴백 + 경고(에러 아님,
   `--strict` 승격)로 이식성을 지킨다.
7. **합성 컴포넌트의 파라미터 전달** → **속성 파라미터 + 명명 슬롯까지만** (§7.8).
   조건·반복·표현식 없음 — "SOML에는 제어 흐름이 없다"가 템플릿 언어화를 끊는 선이다.
   반복은 SOML을 생성하는 프로그램의 몫 (P2 데이터 주도 덱).
8. **인프로세스 레이아웃 백엔드 도입 조건** → P2 유지 + **도입 게이트 3조건** (§11):
   ①텍스트 측정이 골든 코퍼스에서 §13-1 허용 오차 내 ②flex/grid 골든 좌표 100% 일치
   ③무브라우저 CI 수요 실증. 셋을 만족하기 전에는 착수하지 않는다. (텍스트 측정이
   필요해지면 `x/image/font/sfnt`로 폰트 메트릭을 읽는 경로가 있다.)
9. **슬라이드 종횡비** → **현재 안 확정** (§6.2). 덱 설정 `aspect`가 결정하고 테마가
   대응한다. v1은 `16:9`(1280×720)·`4:3`(960×720) 2종만, PPTX 페이지 크기 연동,
   테마는 두 비율 지원 의무 (`theme lint` 검사).
10. **폰트 임베딩** → **v1은 임베드하지 않는다** (§9.3). 테마는 폰트를 이름 + 폴백
    스택으로 명시하고, 웹 타깃은 테마 에셋 woff2 허용. PPTX 임베드는 P2의 opt-in
    (`embedFonts`)으로 미루고 라이선스 책임은 사용자에게 고지한다. 공식 테마는
    광범위하게 설치된 폰트 기준으로 설계한다.
11. **테마의 JS 포함 허용 범위** → **v1 테마는 JS를 포함할 수 없다** (§8.2). 테마는
    JSON + CSS + 에셋의 데이터 전용 포맷 — `theme get` 원격 테마의 임의 코드 실행을
    차단하고 "컴파일 없음" 서사를 지킨다. 전환 효과는 선언적 CSS로 한정, 샌드박스
    훅은 수요 실증 시 v2 검토.
12. **`<icon>` 아이콘 라이브러리 번들** → **번들하지 않는다** (§11 P1). `<icon src>`는
    로컬 SVG 파일만 받는다. 단일 바이너리 크기(G7)와 라이선스 고지 관리 부담이 이유이며,
    Lucide 등 공식 번들은 P2에서 재검토한다.

---

## 14. 성공 지표

| 지표 | 목표 (v1 출시 후 6개월) |
|---|---|
| "슬라이드 무수정 테마 교체" 성공률 | 100% (계약 준수 콘텐츠 기준) |
| **PPTX 네이티브 변환율** | SOML 코어 어휘로 작성된 슬라이드의 100%가 편집 가능한 객체로 |
| **웹 ↔ PPTX 시각적 일치** | 픽셀 비교 기준 주요 편차 0건 (텍스트 줄바꿈 허용 오차 내) |
| **PPTX 산출물의 테마 보존** | 토큰 참조 색이 `schemeClr`로 보존되어 PowerPoint 테마 편집에 반응 |
| 첫 슬라이드까지 소요 시간 (`balpyo init` → 브라우저) | < 1분 |
| 첫 PPTX 산출까지 소요 시간 | < 5분 |
| 테마 스캐폴딩 → 첫 커스텀 테마 완성 | < 30분 |
| 좌표 계산 제거율 | 카드 그리드 슬라이드 기준 수동 좌표 계산 128회 → 0회 |
| 커뮤니티 테마 수 | 10개 이상 |

---

## 15. 리스크와 완화

| 리스크 | 완화 |
|---|---|
| **텍스트 메트릭 차이** — 브라우저가 측정한 텍스트 박스가 PowerPoint에서 다르게 줄바꿈 | 박스 크기 명시 고정 + autofit 비활성 + 테마 폰트 명시(이름+폴백 스택, 임베드는 P2 opt-in — §13-10). "한 줄 차이" 수준의 허용 오차를 명세로 정의. **M0.5 스파이크로 최우선 검증** |
| **외관 스키마의 표현력 부족** — 스키마가 못 담는 디자인을 웹 전용 CSS로 도망가면 PPTX 충실도가 무너짐 | 스키마 v1을 실물 덱 역분석에 기반해 설계 — "디자인된 느낌"의 요소(그라디언트 3종+색 변형)를 처음부터 포함. dev 모드가 번역 불가 CSS를 즉시 경고. 스키마 확장 프로세스 공개 |
| **새 언어의 학습·생태계 비용** — SOML은 배워야 하고 에디터 지원이 없다 | 단계적으로 값싸게 해소한다: ①XML 파일 연결(설정 몇 줄, P0)로 하이라이팅·폴딩 확보 ②스키마 배포(P0)로 기본 완성 ③`balpyo lsp`(P1)로 테마 인식 완성·진단. 레이아웃 태그는 CSS flex 시맨틱 그대로라 "이미 아는 것"이고, 닫힌 어휘라 문서·자동완성·LLM 생성이 모두 유리. `format`이 팀 저작의 diff 노이즈를 제거 |
| **웹 자산은 JS/CSS인데 엔진은 Go** — 저장소에 두 개의 툴체인이 생긴다 | 웹 런타임은 ~10KB 단일 vanilla TS 파일이라 빌드가 사소하고, 결과물을 `go:embed`로 넣으므로 **런타임 빌드는 릴리스 시 1회**다. 테마는 JSON+CSS로 컴파일 대상이 아니다. 개발 중에는 dev 서버가 소스를 직접 서빙해 재빌드를 우회한다 |
| **시스템 브라우저 부재/버전 편차** — CI·컨테이너에 Chrome이 없거나 버전이 다르면 좌표가 흔들릴 수 있다 | Chrome/Edge 자동 감지 + 미탐지 시 Chromium 내려받기 opt-in 안내. 지오메트리는 flex/grid 박스 산수라 버전 간 편차가 사실상 없는 영역이지만, CI에 브라우저 버전을 고정하고 골든 좌표 스냅샷 테스트를 둔다 |
| **레이아웃 태그가 CSS를 재발명할 유혹** — 속성이 늘어나며 미묘한 시맨틱 차이 발생 | "레이아웃 태그 = CSS 이름 바꾸기 층"을 명세에 못 박고 각 속성의 CSS 대응을 문서화(§7.4). 못 덮는 배치는 `style` 탈출구로 넘긴다 (배치는 어차피 좌표로 구워지므로 안전) |
| OOXML의 복잡성 — 마스터/관계/콘텐츠 타입 엣지 케이스 | 지원 범위를 코어 어휘 + 외관 스키마로 명시 한정. PowerPoint/Keynote/LibreOffice 3종 오픈 테스트를 CI에 포함 |
| `<raw>` 남용 — 탈출구를 쓰면 PPTX 충실도와 테마 추종이 동시에 무너짐 | 코어 어휘를 실물 덱 분석 기반으로 설계해 탈출 수요 최소화 (커버리지 ~90% 확인). raw 블록당 dev 경고 |
| 캔버스 슬라이드에서 테마 교체 효과 약화 (배치는 저자 소유) | variant/토큰이 테마를 따라감을 데모로 증명 (같은 다이어그램 × 테마 3종). "배치는 유지, 룩은 교체"가 캔버스의 계약임을 문서화 |
| 테마 품질 편차 → "balpyo = 못생김" 인식 | 공식 테마 3종 품질에 집중 투자. `theme lint`로 커버리지 강제 |
| 상표/브랜드 자산 — 기업 공식 테마의 로고·법적 문구 재현 | 데모 수준은 문제없으나 공개 배포 시 브랜드 가이드라인 승인 필요. 레지스트리 운영 정책에 포함 |

---

## 16. 마일스톤

| 단계 | 범위 | 기간(안) |
|---|---|---|
| **M0.5 — 지오메트리 스파이크 (Go)** | 실물 기업 덱의 8-카드 그리드 슬라이드를 SOML(`<grid>` + 합성 컴포넌트)로 작성 → **chromedp로 배치 측정** → 외관 순방향 컴파일로 네이티브 PPTX 생성 → 원본과 비교. **최대 리스크 2건(텍스트 메트릭, 외관 컴파일) + Go 툴체인(CDP·zip·XML 조립) 동시 검증 — 정답지가 이미 존재** | 1주 |
| M1 — SOML + 웹 | P0 전체: 파서 + 덱 모델 + 외관 해석기 + 레이아웃/컴포넌트 어휘 + 웹 렌더러 + `init/new/server/build --target html` + 내장 `minimal` 테마 + 릴리스 파이프라인 | 6주 |
| M2 — PPTX 렌더러 | 이미지 모드 → 네이티브 모드, 토큰→테마 파트, 3종 오피스 호환 테스트, PDF | 5주 |
| M3 — 테마·저작 도구 | 테마 워크플로우 + 공식 테마 3종(웹·PPTX 동시 검증) + 합성 컴포넌트 + 에디터 지원 | 4주 |
| M4 — 공개 | 문서 사이트, SOML 언어 레퍼런스, 데모 갤러리("같은 덱 → 웹 발표 + PPTX 제출"), v1.0 | 2주 |

---

## 17. 설계 근거 (검증 완료)

본 정의는 세 차례의 실측 실험 위에 서 있다. 상세 기록은 [`PRELIMINARY.md`](PRELIMINARY.md).

| 실험 | 확인된 사실 | 본 문서에 반영된 지점 |
|---|---|---|
| **테마 교체 프로토타입** (3종 테마 × 동일 콘텐츠) | 콘텐츠 무수정 테마 교체 성립. 테마가 레이아웃까지 소유 가능. 토큰이 자유 그래픽까지 관통. CSS cascade layers 필요. 의사 요소로는 크롬 장식이 부족 | 1층 레이아웃 계약(§7.3), 선언적 크롬(§8.1), `@layer` 전략(§12) |
| **실물 기업 덱 역분석 + 재구성** (AWS 템플릿 기반 실제 덱) | 도형 인벤토리의 ~90%가 코어 어휘로 커버. "덱 = 얇은 콘텐츠 + 거대한 디자인 레이어". 카드는 테마 스타일 매트릭스를 참조. "디자인된 느낌"의 정체는 그라디언트 3종 + 색 변형. 동일 도형 그룹의 8회 반복. python-pptx 재구성 비용 = 좌표 계산 128회 + raw XML 수술 10회 | 컴포넌트 어휘(§7.5), 외관 스키마 v1 스코프(§9.4), 합성 컴포넌트(§7.8), 테마=디자인 시스템(§8) |
| **지오메트리 백엔드 벤치마크** (Chromium vs Yoga) | 시스템 브라우저 제어 방식은 다운로드 0·슬라이드당 1.5ms·콜드 스타트 0.6초. Yoga는 156배 빠르지만 텍스트 엔진이 없어 최대 34px 오차, `flex-shrink` 기본값 차이로 조용한 편차, CSS grid 미지원. 브라우저는 PDF·래스터화에 어차피 필요 | 기본 백엔드 결정(§9.3), 플러그블 인터페이스, 인프로세스 백엔드의 P2 강등(§11, §13-8) |

**언어 결정에 대한 주석**: 벤치마크는 `playwright-core`(Node)로 측정했으나, 결론은
"**시스템 브라우저를 CDP로 제어한다**"는 전략에 관한 것이므로 언어 변경과 무관하게
유효하다. Go의 `chromedp`는 같은 프로토콜을 네이티브로 말하며, npm 의존성 13MB가
바이너리 내장 의존으로 대체되는 것이 유일한 차이다 (§12.1).
