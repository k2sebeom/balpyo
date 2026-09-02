# 0029 — 겹침의 칠 순서를 지키려고 stack 자식을 전부 위치 지정 요소로 만든다

- 상태: 확정
- 날짜: 2026-09-02
- Iteration: 6 (항목 04 — NOTES.md 06/04가 근거 기록이다)
- 관련: PRD §13-3, [`<stack>`](../reference/soml/stack.md),
  [지오메트리 패스](../internals/geometry-pass.md), ADR 0028 (stack은 흐름이다),
  ADR 0031 (한 채널이 위치를 정한다)

## 맥락

겹침의 계약은 **문서 순서가 곧 z-순서**다 (PRD §13-3) — 먼저 쓴 것이 아래, 뒤에 쓴 것이 위.
PPTX는 이 계약을 그냥 지킨다: `spTree`의 나중 도형이 위에 그려지고, 자식 도형이 문서 순서로
나열되므로 순서가 곧 z-축이다.

**웹은 그렇지 않다.** CSS는 같은 쌓임 문맥에서 **위치 지정된(positioned) 자손을 위치 지정되지
않은(static) 자손보다 나중에** 칠한다. stack의 자식 중 `x`/`y`를 준 것은 `position: absolute`가
되고 나머지(`at`·기본 정렬)는 정적이므로, 둘이 섞이면 문서상 **앞**에 있는 x/y 자식이 정적
자식보다 **뒤에** 칠해진다 — 겹침 순서가 PPTX와 반대가 된다. 이 iteration은 **웹이 PPTX의
계약에 맞추는 첫 사례**다 (지금까지는 브라우저가 재고 PPTX가 그 좌표를 따랐다 — 방향이 처음
뒤집힌다).

`04_z_order` 픽스처가 그 배합을 정확히 만든다: muted(x/y, 문서상 먼저) · accent1(at=center,
정적, muted를 덮어야) · fg(x/y, 문서상 뒤, 맨 위)를 한 stack에 겹친다. 아무 장치가 없으면
겹친 자리의 색이 두 타깃에서 갈라진다.

## 선택지

1. **엔진이 자식마다 `z-index: N`을 낸다** — 문서 순서를 z-index 수치로 굳혀 칠 순서를 강제한다.
2. **stack의 모든 자식에 `position: relative`를 준다** — 형제가 전부 위치 지정 요소가 되면
   같은 쌓임 단계에서 문서 순서로 칠해진다.
3. **그냥 둔다** — 웹이 PPTX와 갈라지는 것을 받아들인다.

## 결정

**선택지 2** — `stackCSS`의 한 줄:

```css
:is(.balpyo-stack, .balpyo-stage-canvas) > * { grid-area: 1 / 1; position: relative; }
```

## 근거

- **위치 지정 형제는 `z-index: auto`일 때 문서 순서로 칠해진다.** 모든 자식을 위치 지정
  요소로 만들면 x/y 자식(`position: absolute`)과 정적 자식이 같은 칠 단계에 들어가고
  (absolute가 relative 선언을 덮어도 "위치 지정됨"은 그대로다), 형제끼리 문서 순서로 칠해져
  계약이 한 줄로 복원된다. 오프셋 없는 `relative`는 박스를 움직이지 않으므로 지오메트리 패스가
  잰 rect도 그대로다.
- **선택지 1(z-index)을 버린 대가는 둘이다.** (a) 골든에 문서 순서와 **중복된** 수치가 들어가
  diff가 무엇이 바뀌었는지 흐려진다. (b) `z-index`는 저자에게 **닫아 둔 채널**이다 (PRD §13-3,
  ADR 0031) — 엔진이 쓰면 저자 CSS가 이길 수 없는 값이 하나 늘고, 저자용 z-index를 열지 않은
  결정과 모순된다.
- **이번 iteration에서 가장 많은 것이 따라오는 결정이다.** 이 한 줄이 웹의 겹침 순서를 PPTX와
  일치시키고, 그 일치를 픽셀로 판정하는 검사(겹친 영역 크롭)와 XML 순서로 판정하는 검사가
  모두 이 결정 위에 선다.
- **캔버스 스테이지도 같은 규칙을 쓴다.** `:is(.balpyo-stack, .balpyo-stage-canvas)` 한
  선택자가 stack과 캔버스를 함께 겨눈다 — 캔버스를 `display:block`으로 따로 만들면 겹침·칠
  순서 규칙이 두 곳에 생기고, 갈라져도 픽셀 비교에서만 드러난다 (겹침은 한 벌이다).

## 결과

- `render/web/layout.go`의 `stackCSS()`가 이 규칙을 낸다 — **`layoutCSS()`와 별도 절**이라
  stack이나 canvas를 실제로 쓴 덱에만 나간다 (안 쓴 골든이 이 iteration 때문에 움직이지 않게).
- **`TestStackFixtureOverlapRegions`(항목 06)가 복원을 픽셀로 판정한다** — `04_z_order`의
  muted∩accent1 divergence 자리(136×156px)를 두 렌더에서 잘라 대조한다 (측정 2.627%, 허용 6%).
  `position: relative` 한 줄이 빠지면 그 자리가 두 타깃에서 갈라진다
  ([테스트 환경](../internals/testing.md)).
- **`TestRenderStackZOrderIsDocumentOrder`가 XML 순서를 못 박는다** — `spTree` 등장 순서 =
  문서 순서, `p:sp`와 `p:pic`이 한 순회의 문서 순서 자리에 낀다.
- **`x`/`y`는 `left`/`top`이지 `transform`이 아니다** (같은 함수 `appendPlacement`) — 측정
  뷰가 변형을 끄므로(ADR 0021) 위치를 변형으로 내면 잰 rect에 남지 않는다
  ([지오메트리 패스](../internals/geometry-pass.md)의 "위치는 변형으로 내지 않는다").
