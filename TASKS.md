# TASKS — 태그 세트 반복(iteration) 구현 플랜

> [PRD.md](PRD.md) 기준. **PDF 렌더는 out of scope.** 타깃은 웹(HTML)과 네이티브 PPTX 두 개.
>
> 진행 방식: 태그 세트를 하나씩 골라 **테스트 작성 → 구현 → 결과 확인·시각 리뷰**를
> 반복한다. 모든 반복은 같은 절차를 따르고, 테스트는 LibreOffice가 설치된 Docker
> 컨테이너 안에서 돌아 로컬과 CI가 같은 결과를 낸다.

---

## 문서 구조 (`docs/`)

문서는 코드와 같은 저장소에서 iteration을 따라 자란다. 배치 기준:

```
docs/
├── reference/              # "무엇이 있고 어떻게 쓰는가" — 태그가 추가될 때마다 갱신
│   ├── soml/               # 태그별 레퍼런스. 태그 1개(또는 밀접한 세트) = 파일 1개
│   │   ├── slide.md        #   속성 표 · 예제 · 웹/PPTX 렌더 결과 · 에러 조건
│   │   ├── text.md
│   │   └── …
│   ├── appearance.md       # 외관 속성 전체 + 타깃별 컴파일 매핑 표 (PRD §9.4의 살아있는 판)
│   ├── layouts.md          # 표준 레이아웃 카탈로그 11종 + 슬롯 목록
│   ├── theme.md            # 테마 패키지 포맷 (theme/tokens/components.json, layouts css)
│   └── cli.md              # 서브커맨드 레퍼런스
├── guides/                 # "어떻게 하는가" — 사용자 시나리오 단위
│   ├── getting-started.md
│   ├── theming.md
│   └── components.md       # 합성 컴포넌트 만들기
├── internals/              # 기여자용 — 아키텍처·파이프라인·테스트 환경
│   ├── architecture.md     # 파이프라인, 덱 모델, 패키지 경계
│   ├── geometry-pass.md    # CDP 지오메트리 패스, EMU 변환, 텍스트 메트릭 전략
│   └── testing.md          # Docker 이미지, make test/review/golden, 골든 갱신 규칙
└── decisions/              # ADR — 구현 중 확정하는 설계 결정 (번호 순, 불변)
    └── 0001-parser-trivia.md
```

**규칙**:
- 문서 갱신은 iteration **완료 조건**이다 — 구현 커밋과 같은 PR/커밋 흐름에 포함한다.
- **구현된 동작만 쓴다.** 미래 계획·미구현 속성은 레퍼런스에 넣지 않는다 (그건 PRD와
  이 파일의 몫).
- 태그 레퍼런스 페이지의 형식 통일: 속성 표 → SOML 예제 → 웹/PPTX 각각의 렌더 결과
  설명 → 빌드 에러가 나는 조건.
- **PRD §13의 열린 질문 12건은 전부 확정되어 PRD 본문에 반영되었다 (2026-08-16).**
  구현 중 새로 확정하는 설계 결정은 `docs/decisions/NNNN-*.md` ADR로 남긴다 (맥락 →
  선택지 → 결정 → 근거). PRD §13의 기존 결정을 뒤집는 경우 PRD도 함께 갱신한다.
  하단 결정 로그 표는 ADR의 색인이다.

---

## 반복 절차 (모든 Iteration 공통)

각 iteration은 아래 4단계를 순서대로 밟는다.

### ① 테스트 작성 (RED)

- `testdata/fixtures/<NN>_<이름>/` 에 이번 태그 세트를 사용하는 `.slide` 픽스처 작성
  - 정상 케이스 + **에러 케이스** (닫힌 어휘 위반, 잘못된 속성 — 위치 정보 포함 에러 검증)
- 테스트 종류:
  | 테스트 | 검증 대상 |
  |---|---|
  | 파서 테스트 | `.slide` → 덱 모델 (구조·속성·트리비아 보존·에러 위치) |
  | 외관 해석 테스트 | 캐스케이드 결과 (인라인 > variant > 기본값 > 역할/토큰) |
  | 웹 골든 테스트 | 생성 HTML/CSS 스냅샷 (`.golden` 파일 비교) |
  | PPTX 구조 테스트 | 산출 OOXML 파트의 XML 어서션 (도형 종류, `schemeClr` 참조, `xfrm` 존재) |
  | PPTX 열림 테스트 | LibreOffice headless가 에러/복구 없이 열고 렌더링하는지 |
  | 픽셀 비교 테스트 | 웹 스크린샷 vs LibreOffice 렌더 PNG — 허용 오차 내 일치 |

### ② 구현 (GREEN)

- `soml/` 파서에 태그·속성 추가 (닫힌 어휘 카탈로그 갱신)
- 덱 모델 노드 + 외관 해석기 규칙 추가
- `render/web/` 매핑 추가 (태그 → HTML/CSS)
- `render/pptx/` 매핑 추가 (태그 → OOXML)
- 컨테이너 안에서 `make test` 전부 통과

### ③ 결과 확인 · 시각 리뷰 (REVIEW)

- `make review` → `review/<fixture>/` 에 산출물 생성:
  - `web.png` (헤드리스 브라우저 스크린샷)
  - `pptx.png` (LibreOffice 렌더)
  - `diff.png` (픽셀 diff)
  - `index.html` (전체 픽스처 나란히 보기 갤러리)
- 사람이 갤러리를 눈으로 확인. macOS에서 `deck.pptx`를 **Keynote/PowerPoint로 직접
  열어** 편집 가능성(텍스트 박스가 진짜 텍스트 박스인지, 테마 색 편집을 따라가는지) 확인
- 이번 iteration에서 새로 확정한 설계 결정을 ADR + 하단 로그에 기록

### ④ 문서화 (DOC)

- 각 iteration에 명시된 `docs/` 파일을 작성·갱신한다 (아래 iteration별 "문서" 항목).
- 새 태그 → `docs/reference/soml/<태그>.md` 신규. 새 외관 속성 → `docs/reference/appearance.md`
  의 매핑 표에 행 추가. 새로 확정한 설계 결정 → `docs/decisions/` ADR.
- 문서의 예제는 실제 픽스처(`testdata/fixtures/`)에서 가져온다 — 문서 예제가 곧
  테스트된 코드가 되도록.
- 문서까지 커밋한 뒤 다음 iteration으로

---

## Iteration 0 — 기반: 프로젝트 골격 + Docker 테스트 환경

태그 없음. 이후 모든 반복이 올라탈 기반을 만든다.

- [ ] **Go 패키지 골격** (PRD §12.2): `soml/`(파서+덱 모델), `core/`(외관 해석·검증),
      `render/web/`, `render/pptx/`, `cmd/balpyo/`
- [ ] **선결정 확정** (코드 첫 줄 전에):
  - [x] 수치 단위: **확정 완료 (PRD §13-5)** — 논리 스테이지 px 단일 단위 (16:9 =
        1280×720). PPTX 변환 계수(`1px = 9525 EMU`, 폰트 `pt = px × 0.75`)는 상수로 구현
  - [ ] 파서 자료구조: **트리비아(공백·주석·인용부호) 보존 토크나이저** — build/check만
        쓰더라도 처음부터. 나중에 format/lsp에 재사용 (PRD §12.2 "나중에 붙일 수 없는 결정")
- [ ] **Docker 테스트 이미지** (`docker/test.Dockerfile`):
  - [ ] `golang` + **LibreOffice** (headless impress) + **Chromium** (지오메트리 패스·스크린샷)
        + `poppler-utils` (pdf → png)
  - [ ] **폰트 고정**: Noto Sans / Noto Sans CJK KR 등을 이미지에 설치·핀 — 텍스트 메트릭
        테스트가 재현 가능하려면 폰트가 결정적이어야 함
  - [ ] 브라우저·LibreOffice 버전 핀 (골든 좌표/픽셀 스냅샷 안정성)
- [ ] **테스트 하네스** (`internal/testkit/`):
  - [ ] 픽스처 러너: `testdata/fixtures/**` 순회 → 빌드 → 골든 비교
  - [ ] LibreOffice 헬퍼: `soffice --headless --convert-to pdf` → `pdftoppm` → PNG,
        stderr에 복구/경고 감지 시 실패
  - [ ] Chromium 헬퍼: HTML 로드 → 스크린샷 / `getBoundingClientRect` 수집
  - [ ] 픽셀 diff 헬퍼 (허용 오차 파라미터화 — 텍스트 안티앨리어싱 감안)
- [ ] **Make 타깃**: `make test` (컨테이너 안에서 go test), `make review` (갤러리 생성),
      `make golden` (골든 갱신)
- [ ] CI 워크플로우: 같은 이미지로 `make test`
- [ ] **문서**:
  - [ ] `docs/` 골격 생성 (위 문서 구조 트리대로 디렉토리 + 빈 index)
  - [ ] `docs/internals/testing.md` — Docker 이미지 구성, make 타깃, 골든 갱신 규칙
  - [ ] `docs/internals/architecture.md` — 파이프라인·패키지 경계 초안 (덱 모델 섹션은 I1에서)
  - [ ] `docs/decisions/0001-parser-trivia.md` — 트리비아 보존 파서 ADR

**완료 기준**: 빈 픽스처로 파이프라인이 끝에서 끝까지 돌고(내용은 더미), 컨테이너
안에서 LibreOffice·Chromium 헬퍼가 동작한다.

---

## Iteration 1 — 수직 절단: `<slide>` `<text>` `<note>` (지오메트리 패스 포함)

가장 작은 어휘로 **파싱 → 덱 모델 → 외관 해석 → 웹 + 네이티브 PPTX** 전 구간을 뚫는다.
PRD의 M0.5 스파이크(최대 리스크 검증)를 이 iteration이 흡수한다.

- 태그: `<slide>` (`layout` 제외한 속성: `id`, `skip`), `<text>` (일부 외관: `role`,
  `size`, `color`, `align`), `<note>`
- [ ] 테스트: 파서(정상/오타 태그 에러/위치), 슬라이드 수집·파일명 정렬, 웹 골든,
      PPTX 구조(`sp`+`txBody`, `xfrm`), LibreOffice 열림, 픽셀 비교
- [ ] 구현:
  - [ ] 엄격 파서 + 닫힌 어휘 + line/col 에러
  - [ ] 최소 테마 하드코딩 (토큰 몇 개 — 테마 패키지 포맷은 Iteration 10)
  - [ ] SOML → HTML 변환 + 생성 CSS
  - [ ] **지오메트리 패스**: chromedp로 렌더 → 요소별 rect → EMU. 텍스트 박스 크기
        명시 고정 + autofit off (PRD §9.3)
  - [ ] OOXML 조립기 최소판: presentation/master/layout/theme 파트 + 텍스트 박스,
        `archive/zip` 직조립
  - [ ] `<note>` → PPTX 노트 슬라이드
  - [ ] 얇은 CLI: `balpyo build --target html,pptx` (init/server는 Phase 2)
- [ ] 리뷰 포인트: **텍스트 메트릭** — 브라우저가 잰 박스가 LibreOffice/Keynote에서
      같은 줄바꿈을 유지하는가. 기본 정책은 확정 완료(PRD §13-1: 박스 고정 + autofit
      off + 오버플로 허용) — 여기서는 **수치 임계값**을 측정해 정의한다
- [ ] 문서:
  - [ ] `docs/reference/soml/slide.md`, `text.md`, `note.md`
  - [ ] `docs/reference/appearance.md` 생성 — `role`/`size`/`color`/`align`만 있는 초판 매핑 표
  - [ ] `docs/reference/cli.md` 생성 — `build` 항목만
  - [ ] `docs/internals/geometry-pass.md` — CDP 패스, EMU 변환, 텍스트 박스 고정 전략
  - [ ] `docs/internals/architecture.md` 덱 모델 섹션 완성
  - [ ] PRD §13-1 — 측정한 줄바꿈 허용 오차의 수치 임계값 기입

**이 iteration이 실패하면(텍스트 메트릭 오차가 수용 불가) 아키텍처를 재검토한다 —
그래서 가장 먼저 한다.**

---

## Iteration 2 — 텍스트 서브셋 + 역할/캐스케이드 완성

- 태그: `h1`~`h3`, `p`, `ul`/`ol`/`li`, `blockquote`, `pre`, `code`, `strong`, `em`,
  `a`, `br`, `span`
- [ ] 테스트: 블록 태그 ≡ 텍스트 역할 등가성 (`<p>` ≡ `<text role="body">`),
      인라인 서식 → PPTX 텍스트 런(`rPr`, `hlinkClick`), 리스트 → `buChar`/들여쓰기
- [ ] 구현:
  - [ ] 텍스트 역할 계약 (`title`/`heading`/`body`/`caption`/`code`)
  - [ ] **외관 해석기 4단 캐스케이드 완성**: 인라인 > `variant` > 컴포넌트 기본값 >
        역할/토큰 (variant 정의는 아직 하드코딩 테마 안)
  - [ ] 인라인 태그 → 웹 그대로 / PPTX 런 분할
- [ ] 리뷰: 불릿 리스트 슬라이드가 양 타깃에서 동일한 들여쓰기·마커로 보이는지
- [ ] 문서:
  - [ ] `docs/reference/soml/text-subset.md` — HTML 서브셋 태그 표 + 역할 등가 표
        (`<p>` ≡ `<text role="body">`)
  - [ ] `docs/reference/appearance.md` — **캐스케이드 규칙 섹션** (4단 해석 순서) +
        텍스트 역할 5종, 인라인 서식 매핑 행 추가

## Iteration 3 — 레이아웃 태그: `<row>` `<col>` `<grid>` `<spacer>`

- 태그·속성: 컨테이너(`gap` `align` `justify` `wrap` `pad` `cols`/`rows` `w`/`h`/`min-*`/
  `max-*` `ratio`) + 자식(`grow` `shrink` `basis` `span` `self` `order`) + `style` 탈출구
- [ ] 테스트: 속성 → CSS 매핑 단위 테스트 (PRD §7.4 표 그대로 — "CSS 이름 바꾸기 층"
      원칙의 회귀 테스트), 공간 토큰(`gap="xl"`) 해석, 중첩 레이아웃 지오메트리 패스
      좌표 스냅샷, `<grid cols="4">` 카드 배치 픽셀 비교
- [ ] 구현: 태그 → flex/grid CSS 컴파일. **레이아웃 태그는 PPTX 도형이 아니다** —
      지오메트리 패스에서 자식 좌표만 소비되고 컨테이너 자체는 사라짐
- [ ] 구현: `style` 탈출구 — 배치 속성 allowlist 검사, 외관 CSS 사용 시 경고 (§7.9)
- [ ] 리뷰: 좌표 계산 0회로 그리드 배치가 PPTX에 정확히 이식되는지
- [ ] 문서:
  - [ ] `docs/reference/soml/layout.md` — `row`/`col`/`grid`/`spacer` + 컨테이너/자식
        속성 전체와 **CSS 대응 표** (PRD §7.4 — "이름 바꾸기 층" 계약의 공식 문서)
  - [ ] `docs/reference/soml/layout.md` 안에 `style` 탈출구 규칙 (배치 전용, 경고 조건)

## Iteration 4 — `<shape>` 12종 + 외관 속성(단색)

- 태그: `<shape>` (`type` 12종: `rect` `rounded-rect` `ellipse` `diamond` `cylinder`
  `chevron` `arrow-right` `hexagon` `parallelogram` `callout` `star` `cloud`)
- 외관: `fill`(토큰/리터럴/`none`), `border`(단색), `radius`, `shadow`, `opacity`,
  `rotate`/`flip` — **그라디언트는 Iteration 8**
- [ ] 테스트: type → `prstGeom` 매핑 12종 전수, **토큰 색 → `schemeClr` 보존** (산출
      XML에 리터럴 RGB가 아닌 테마 참조가 있는지 — 제품 핵심 계약), `radius` →
      `prstGeom` 조정값, 도형 내 `<text>` 앵커링
- [ ] 구현: shape → 웹 div / PPTX `sp`, 외관 속성 → CSS / OOXML 양방향 컴파일 (§9.4 표)
- [ ] 리뷰: PowerPoint/Keynote에서 도형 선택 → 진짜 프리셋 도형인지, **테마 색을 바꾸면
      도형 색이 따라가는지** 직접 확인
- [ ] 문서:
  - [ ] `docs/reference/soml/shape.md` — `type` 12종 표 (각각 웹/PPTX `prstGeom` 대응)
  - [ ] `docs/reference/appearance.md` — `fill`/`border`/`radius`/`shadow`/`opacity`/
        `rotate`/`flip` 행 추가 + **토큰 색 → `schemeClr` 보존** 계약 명문화

## Iteration 5 — `<image>`

- 태그: `<image>` (`src`, `fit=cover/contain`, `frame`, `border`, `radius`)
- [ ] 테스트: 에셋 수집·PPTX 미디어 파트/관계 파일, `fit` → `object-fit` / `srcRect`
      크롭 등가성, 없는 `src` 빌드 에러
- [ ] 구현: 이미지 파이프라인 (assets/ 해석, PPTX `pic` + rels)
- [ ] 리뷰: cover 크롭이 양 타깃에서 같은 영역을 보여주는지
- [ ] 문서: `docs/reference/soml/image.md` (`src` 해석 규칙 — assets/ 상대 경로 포함)

## Iteration 6 — `<stack>` + `layout="canvas"` + 절대 배치

- 태그·속성: `<stack>`, 자식 `at`(9지점 앵커), `x`/`y`
- [x] **선결정 확정 완료 (PRD §13-3)** — 단일 셀 grid 겹침, `at` → `place-self`,
      `x`/`y`는 부모 상대 absolute, z-순서 = 문서 순서. 테스트는 이 계약대로 작성
- [ ] 테스트: 겹침 z-순서 → PPTX 도형 순서, 9지점 앵커 좌표 스냅샷, 배지/워터마크 픽스처
- [ ] 구현: stack 컴파일 + canvas 슬라이드 (테마는 배경/여백만 소유)
- [ ] 리뷰: 오버레이 배지가 양 타깃에서 같은 위치·같은 겹침 순서인지
- [ ] 문서:
  - [ ] `docs/reference/soml/stack.md` — `at` 9지점, `x`/`y` 좌표 기준, z-순서 규칙
        (PRD §13-3 계약대로)

## Iteration 7 — `<arrow>` `<line>` (가장 어려운 세트)

- 태그: `<arrow>` (`from`/`to` id 참조, `label`, `heads`, `dash`, `via`),
  `<line>` (`cap`, `dash`, `dir`, `stroke`)
- [x] **선결정 확정 완료 (PRD §13-4)** — 자동 앵커(최근접 변 중점)가 기본,
      `from="db@top"` 명시 오버라이드(4변+center), `via`는 공백 구분 논리 좌표점 목록
- [ ] 테스트: `id` 참조 해석(없는 id는 빌드 에러), 앵커 계산 단위 테스트, 배치 변경 시
      연결 유지, PPTX `cxnSp` 구조, 다이어그램 픽스처(PRD §7.4 예제) 픽셀 비교
- [ ] 구현:
  - [ ] **앵커 계산은 JS로 한 번만 작성** — 웹 런타임(SVG 패스)과 지오메트리 패스(CDP로
        같은 스니펫 평가) 양쪽에서 동일 코드 사용. 로직 이중 구현 금지
  - [ ] `label` → 선 중앙 텍스트 (웹/PPTX)
- [ ] 리뷰: 아키텍처 다이어그램 픽스처 — PowerPoint에서 커넥터를 잡아 끌면 도형에
      붙어 있는지(진짜 `cxnSp`인지)
- [ ] 문서:
  - [ ] `docs/reference/soml/arrow.md` — `from`/`to`/`via`/앵커 시맨틱 (PRD §13-4
        계약대로), `docs/reference/soml/line.md`

## Iteration 8 — 외관 스키마 v1 완성: 그라디언트 + 색 변형

PRD §9.4의 굵은 글씨 — "디자인된 느낌"의 정체. 실물 덱 역분석의 핵심 산출.

- 문법: `gradient(accent1 accent4 accent5, 45deg)`, `shade(accent5, 50%)`, `alpha(fg, 0.6)`
- [x] **선결정 확정 완료 (PRD §13-5, §7.6)** — `gradient(색… [, 각도deg])` 균등 스톱·
      기본 180deg, `shade`/`tint`/`alpha` 3종·중첩 불허
- [ ] 테스트: `fill=gradient` → `linear-gradient` / `gradFill`, **`border=gradient`** →
      SVG 오버레이(엔진 소유 구현) / `ln`+`gradFill`, **텍스트 `color=gradient`** →
      `background-clip:text` / `rPr`+`gradFill`, 색 변형 → `color-mix` /
      `lumMod`/`lumOff`/`alpha` (schemeClr 유지 확인)
- [ ] 리뷰: `workspace/AI_FCD.pptx`(역분석 원본)의 그라디언트 카드를 SOML로 재현 →
      원본과 나란히 비교
- [ ] 문서:
  - [ ] `docs/reference/appearance.md` — 그라디언트 표기·색 변형 함수 문법 섹션 +
        매핑 표 완성 (§9.4 전체가 문서에 반영된 상태가 이 iteration의 완료 기준)

## Iteration 9 — `<group>`

- [ ] 테스트: `<group>` → PPTX `grpSp` (자식 좌표가 그룹 상대 좌표로 변환되는지),
      `<row>`와의 차이(그룹은 편집 단위, row는 배치 후 소멸)
- [ ] 구현: 그룹 좌표 변환 + 웹 div 매핑
- [ ] 리뷰: PowerPoint에서 그룹 선택·이동이 한 단위로 되는지
- [ ] 문서: `docs/reference/soml/group.md` — `<row>`/`<col>`과의 사용 구분 기준 포함

## Iteration 10 — 1층: 표준 레이아웃 카탈로그 + 슬롯 + 테마 패키지 포맷

- 태그·속성: `<slide layout="…">`, `slot="…"`, 기본 슬롯(body) 규칙
- [ ] 테스트: 존재하지 않는 슬롯 빌드 에러, 레이아웃 11종(`title` `section-divider`
      `content` `two-column` `image-full` `image-text` `quote` `code` `stat` `canvas`
      `end`) 각각 픽스처 + 픽셀 비교, 미구현 레이아웃 → 기본 테마 폴백 + 경고
- [ ] 구현:
  - [ ] **테마 패키지 포맷** (하드코딩 테마 해체): `theme.json` + `tokens.json` +
        `components.json` + `layouts/*.css` (배치 전용)
  - [ ] 토큰 → CSS 변수 / PPTX 테마 파트(clrScheme/fontScheme) 컴파일
  - [ ] 텍스트 역할 → 마스터 텍스트 스타일, 슬롯 → **PPTX 플레이스홀더**, 크롬 →
        슬라이드 마스터 고정 요소
  - [ ] 내장 테마 `minimal` v1 (`//go:embed`)
- [ ] 리뷰: 같은 덱에 테마 토큰만 바꿔 재빌드 → 웹·PPTX 양쪽에서 룩이 따라가는지
- [ ] 문서:
  - [ ] `docs/reference/layouts.md` — 표준 레이아웃 11종 카탈로그 (레이아웃별 슬롯 표 +
        스크린샷), 폴백 규칙
  - [ ] `docs/reference/theme.md` — 테마 패키지 포맷 (`theme.json`/`tokens.json`/
        `components.json`/`layouts/*.css`), 토큰 표준 어휘, 배치·외관 경계 규칙
  - [ ] `docs/guides/theming.md` — 테마 만들기 가이드 (minimal을 예제로)

## Iteration 11 — 합성 컴포넌트: `<define>` `<slot>`

- [x] **선결정 확정 완료 (PRD §13-7)** — 파라미터는 속성 + `<slot>`까지만
      (조건·반복·표현식 없음, "SOML에는 제어 흐름이 없다")
- [ ] 테스트: 파싱 시 코어 컴포넌트로 전개(덱 모델에 합성 흔적 없음 — 렌더러 불변),
      이름 하이픈 강제, `components/*.slide` 로딩(루트 `<define>` 검증), 카드 그리드
      픽스처(PRD §7.8 예제)
- [ ] 구현: define/slot 전개기
- [ ] 리뷰: **8-카드 그리드 슬라이드를 `<grid>` + 합성 컴포넌트로 재현 → AI_FCD 원본과
      비교** (PRD M0.5의 최종 목표 장면. 수동 좌표 계산 128회 → 0회 증명)
- [ ] 문서:
  - [ ] `docs/reference/soml/composite.md` — `<define>`/`<slot>` 문법, 전개 규칙,
        네이밍(하이픈) 규칙
  - [ ] `docs/guides/components.md` — 합성 컴포넌트 만들기 가이드 (dim-card 예제)

---

## Phase 2 — 어휘 밖 작업 (iteration 사이 또는 이후에 배치)

태그 세트 반복이 아닌 것들. 필요해지는 시점에 끼워 넣는다. **CLI 서브커맨드를 추가할
때마다 `docs/reference/cli.md`를 갱신**하고, 그 밖의 문서 대상은 각 항목에 표기했다.

- [ ] `balpyo init` / `new` (스캐폴딩) — Iteration 10 이후가 자연스러움 (테마 필요)
      → 문서: `docs/guides/getting-started.md` 작성 (init → server → build 흐름)
- [ ] `balpyo server` — 핫 리로드 + 검증 오버레이 + 웜 브라우저 세션. 시각 리뷰 루프를
      크게 가속하므로 **이르면 Iteration 3~4쯤 최소판** 투입 권장
- [ ] 웹 런타임 (vanilla TS ~10KB): 네비게이션, `reveal`, 스케일링, 딥링크 →
      빌드 산출물 `go:embed`
- [ ] `balpyo check` (CI 검증) — 파서·검증기가 이미 있으므로 표면만
- [ ] `balpyo format` — 트리비아 보존 파서 위의 정규화 층 (Iteration 0의 결정이 여기서 회수됨)
- [ ] 릴리스 파이프라인: 크로스 컴파일, GitHub Releases, Homebrew tap
- [ ] 웹 런타임 문서 → `docs/reference/cli.md`의 build 옵션 + 발표 조작(키보드·딥링크)은
      `docs/guides/getting-started.md`에 통합
- [ ] `<table>` / `<icon>`(SVG→`custGeom`) / 코드 하이라이팅(Chroma) — P1 어휘,
      같은 반복 절차로 추가 → 문서: `docs/reference/soml/<태그>.md` 동일 규칙
- [ ] `<raw>` 탈출구 + 래스터화 폴백 (CDP `captureScreenshot` 재사용)
      → 문서: `docs/reference/soml/raw.md` — "raw는 PPTX에서 그림이 된다" 원칙 명시

---

## 설계 결정 로그

**PRD §13의 열린 질문 12건은 2026-08-16에 전부 확정되어 PRD 본문에 반영되었다**
(확장자 §7.1, stack §7.4, arrow 앵커 §7.5, 외관 값 문법·단위 §7.6, 합성 파라미터
§7.8, 테마 JS 금지 §8.2, variant 비표준화 §8.3, 텍스트 메트릭·폰트 §9.3, 종횡비
§6.2, 아이콘·인프로세스 백엔드 §11 — 색인은 PRD §13).

이 표는 구현 중 새로 확정하는 설계 결정(ADR)의 색인이다.

| 결정 | ADR | Iteration |
|---|---|---|
| 트리비아 보존 파서 | `docs/decisions/0001-parser-trivia.md` | I0 |
| 4:3 논리 스테이지는 960×720 (PRD §6.2·§13-9 갱신) | `docs/decisions/0002-logical-stage-size.md` | I0 |
| 텍스트 줄바꿈 허용 오차 — 수치 임계값 (PRD §13-1에 기입) | — | I1 |
