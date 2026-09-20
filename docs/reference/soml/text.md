# `<text>`

텍스트 블록 하나다. 문자 데이터와 [인라인 태그](text-subset.md)(strong/em/code/a/br/span)를
담을 수 있고, 외관은 `role` 기본값 위에 `variant`와 인라인 속성이 얹혀 확정된다
(캐스케이드는 [외관 속성](../appearance.md) 참조).

## 속성

| 속성 | 값 | 기본값 | 뜻 |
|---|---|---|---|
| `role` | `title` `heading` `body` `caption` `code` | `body` | 타이포그래피 역할 — 크기·굵기·줄높이·색·폰트 기본값의 묶음 |
| `variant` | 테마가 정의한 이름 | 없음 | 테마의 외관 번들 — [캐스케이드](../appearance.md) 참조 |
| `size` | 단위 없는 수치 (논리 px) | 역할 기본값 | 폰트 크기 |
| `color` | 색 토큰 이름 · `#RRGGBB` · `#RGB` | 역할 기본값 | 글자색 |
| `align` | `start` `center` `end` `justify` | `start` | 문단 정렬 — 논리 이름이다 (`left`/`right` 아님) |
| `anchor` | `top` `center` `bottom` | 없음 | 박스 안 **세로** 정렬 — 뜻이 보이는 자리는 [도형 안](shape.md)뿐이다 |

역할 5종의 기본값 (내장 `minimal` 테마):

| 역할 | 크기(px) | 굵기 | 줄높이 | 색 | 폰트 토큰 |
|---|---|---|---|---|---|
| `title` | 56 | 700 | 1.2 | `fg` | `heading` |
| `heading` | 36 | 700 | 1.3 | `fg` | `heading` |
| `body` | 24 | 400 | 1.5 | `fg` | `body` |
| `caption` | 18 | 400 | 1.4 | `muted` | `body` |
| `code` | 20 | 400 | 1.5 | `fg` | `mono` |

- `size`는 **단위 없는 논리 스테이지 px**다 (PRD §13-5). `px`·`pt`·`em` 접미사와 `%`는
  파스 에러다 — 상대 크기는 PPTX의 절대 폰트 크기(`rPr@sz`)로 번역할 곳이 없다.
- `align`이 논리 이름인 이유: CSS `text-align: start|end`와 OOXML `a:pPr@algn`이 둘 다
  문단 방향(RTL)에 따라 뜻이 유지되는 논리 값을 받는다. `left`를 쓰면 진단이 대응하는
  논리 이름(`start`)을 짚어 준다.
- **`anchor`는 반대로 물리 이름이다** (`top`/`center`/`bottom`) — 세로축은 문단 방향으로
  뒤집히지 않으므로 논리 이름을 쓸 근거가 없다 (v1은 세로쓰기를 지원하지 않는다).
  `anchor="start"`를 쓰면 진단이 `top`을 짚어 준다.
- **`anchor`는 기본값이 없는 유일한 속성이다.** 뜻이 보이는 자리가
  [도형 안](shape.md)뿐이기 때문이다: 슬라이드 직속 텍스트 박스는 잰 rect가 곧 내용
  크기라 세로로 움직일 여백이 없다 — 기본값을 정하면 뜻이 없는 자리에까지 선언이 실린다.
  블록 태그(`<p anchor="center">`)도 같은 속성을 받는다 (`<p>` ≡ `<text role="body">`
  등가가 속성 표에서도 성립해야 한다).
- 도형 하나에 자식이 여럿이면 **첫 자식의 앵커**가 도형의 앵커다 — 웹의 flex 정렬과 OOXML
  `bodyPr@anchor`가 둘 다 박스에 하나뿐이라, 뒤 자식의 다른 앵커는 조용히 버려진다.
- 원문 개행과 들여쓰기는 파서가 접는다 — ASCII 공백 런은 한 칸이 되고 양 끝은 잘린다.
  두 렌더러가 **같은 문자열**을 받으므로 타깃 간 줄바꿈이 원문 서식에 의존하지 않는다.
- `<`와 `&`는 [문자 참조](text-subset.md)(`&lt;`·`&amp;` + 수치 참조)나 CDATA로 쓴다 —
  둘 다 [텍스트 서브셋](text-subset.md)이 레퍼런스를 갖는다. 다만 참조로 **읽는** 자리는
  `;`로 닫힌 참조 모양일 때뿐이고 태그를 여는 `<`는 영문자·`/` 앞의 `<`뿐이므로 `R&D`와
  `a < b`는 그냥 쓴다. 참조는 접기 **다음**에 풀리므로 `&#32;`로 쓴 공백은 남는다.

## 예제

`testdata/fixtures/01_text/slides/002_overview.slide`:

```html
<!-- 한글과 영문이 섞인 본문. 텍스트 메트릭(줄바꿈)이 두 타깃에서 같은지 보는 슬라이드다. -->
<slide>
  <text role="heading" color="accent1">올해의 세 가지 변화</text>
  <text role="body">모놀리스 분리를 마쳤고, 배포 파이프라인을 자동화했으며,
  관측성 스택(OpenTelemetry + Grafana)을 도입했다.</text>
  <text role="caption" size="18" color="muted" align="end">2026 H1</text>
</slide>
```

## 렌더 결과

**웹** — `<div class="balpyo-text">` 하나가 된다. 확정된 외관은 인라인 `style` 속성의
CSS 선언(`font-family` `font-size` `font-weight` `line-height` `color` `text-align`)으로
나간다 ([ADR 0003](../../decisions/0003-inline-style-appearance.md)). 색·폰트 값은
리터럴이 아니라 테마 변수 참조다 — `color="accent1"`은
`color: var(--balpyo-color-accent1)`이 된다. 요소에는 `data-balpyo-id`(지오메트리
식별자)와 `data-balpyo-el="text"`가 붙는다.

**PPTX** — `p:sp` 텍스트 박스 하나가 된다 (`p:cNvSpPr@txBox="1"`, 프리셋 `rect`,
채움·선 없음). 좌표(`a:xfrm`)는 지오메트리 패스가 브라우저에서 잰 박스에 안전 여유
(폭 +4%)를 더한 값이고, autofit은 꺼진다 (`a:noAutofit` — 박스 고정 + 오버플로 허용,
PRD §13-1). 외관 매핑:

| 확정 값 | OOXML |
|---|---|
| 크기 | `a:rPr@sz` (px × 0.75 × 100 = 1/100 pt) |
| 굵기 | `a:rPr@b` (600 이상이면 `1`) |
| 줄높이 | `a:lnSpc/a:spcPts` (절대 pt — [ADR 0005](../../decisions/0005-line-spacing-and-slack.md)) |
| 토큰 색 | `a:solidFill/a:schemeClr` — **리터럴 RGB로 눕히지 않는다** |
| 리터럴 색 | `a:solidFill/a:srgbClr` |
| 폰트 토큰 | `a:latin`/`a:ea`의 테마 참조 (`+mj-lt`/`+mn-lt` 등, mono만 리터럴) |
| 정렬 | `a:pPr@algn` (`l`/`ctr`/`r`/`just`) |

빈 `<text></text>`도 도형을 낸다 (런 없는 문단) — 건너뛰면 도형 순서와 id가 내용에 따라
흔들린다.

## 빌드 에러가 나는 조건

파서(`soml.Parse`):

| 조건 | 진단 메시지 |
|---|---|
| 자식으로 블록·리스트 요소를 썼다 | `<text> 안에는 문자 데이터와 인라인 태그만 올 수 있다 (<…>는 쓸 수 없다)` |
| 어휘 밖의 속성 | `<text>에 알 수 없는 속성 … (쓸 수 있는 속성: role, variant, size, color, align, anchor, grow, shrink, basis, span, self, order, style)` |

속성 목록의 뒤쪽 일곱 개는 [자식 배치 속성](layout.md)이다 — `<text>`는 레이아웃
컨테이너의 자식이 될 수 있으므로 `<row><text grow="1">`이 유효하다.

외관 해석(`core.Resolve`) — 값 어휘는 이 층이 소유한다:

| 조건 | 진단 메시지 |
|---|---|
| 어휘 밖의 역할 | `알 수 없는 role "…" (쓸 수 있는 역할: title, heading, body, caption, code)` |
| `size`에 단위가 붙었다 | `size는 단위 없는 논리 px 수치다 (…의 …를 뗀다) — px·pt·em 접미사는 쓸 수 없다 (PRD §13-5)` |
| `size`에 `%`를 썼다 | `size에 %는 쓸 수 없다 (…) — PPTX 폰트 크기는 절대값이라 상대 크기를 번역할 곳이 없다. 논리 px 수치나 role을 쓴다` |
| `size`가 수치가 아니다 | `size 값이 수치가 아니다 (…)` |
| `size`가 0 이하 | `size는 0보다 커야 한다 (…)` |
| 테마에 없는 색 토큰 | `color의 색 토큰 "…"가 테마에 없다 (쓸 수 있는 토큰: fg, bg, muted, surface, accent1, 또는 #RRGGBB 리터럴)` |
| 색 리터럴 형식 오류 | `색 리터럴은 #RRGGBB 또는 #RGB 형태다 (…)` |
| 색 함수를 썼다 | `색 함수 …(…)는 아직 쓸 수 없다 (gradient·shade·tint·alpha는 Iteration 8) — 토큰 이름이나 #RRGGBB 리터럴을 쓴다` |
| 물리 정렬 이름 | `align은 논리 이름을 쓴다 — "left" 대신 "start" (RTL에서도 뜻이 유지된다)` |
| 어휘 밖의 정렬 | `알 수 없는 align "…" (쓸 수 있는 값: start, center, end, justify)` |
| 외관 속성에 값이 없다 | `… 속성에는 값이 필요하다 (…="…")` / `… 속성이 비어 있다 (지우면 기본값을 쓴다)` |
