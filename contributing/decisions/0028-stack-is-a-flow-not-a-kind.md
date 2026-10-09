# 0028 — `<stack>`은 새 종류가 아니라 새 흐름이다

- 상태: 확정
- 날짜: 2026-09-02
- Iteration: 6 (항목 01·04·05 — NOTES.md 06/01·06/04·06/05이 근거 기록이다)
- 관련: PRD §7.4, §13-3, [`<stack>`](../../docs/reference/soml/stack.md),
  [레이아웃 태그](../../docs/reference/soml/layout.md), ADR 0009 (측정 대상 종류),
  ADR 0011 (컨테이너는 도형이 아니다), PRD §7.5 (`<group>` — Iteration 9)

## 맥락

`<stack>`(겹침 컨테이너)이 어휘에 들어온다. `<shape>`와 `<image>`는 각각 새 요소
**종류**(`KindShape`·`KindImage`)로 열렸다 — 그 기준은 "여러 층의 산출이 다르다"였다
(도형은 웹에서 svg·PPTX에서 `prstGeom`, 그림은 웹에서 `<img>`·PPTX에서 `p:pic`). `<stack>`도
같은 기준으로 `KindStack`을 만들 것인가가 이 iteration의 자료구조 결정이다.

## 선택지

1. **`KindStack`을 만든다** — `<stack>`이 새 요소 종류가 되고, 파서·core·렌더러가 그 종류를
   분기한다.
2. **`FlowStack`을 만든다** — `<stack>`은 여전히 `KindLayoutContainer`이고, 배치 **흐름**만
   `FlowRow`/`FlowColumn`/`FlowGrid` 옆에 하나 더한다 (`FlowStack`).

## 결정

**선택지 2 — `<stack>`은 새 흐름이다** (`soml.FlowStack`, `soml.FlowOf`). `KindStack`을 만들지
않는다.

## 근거

- **종류를 나눈 기준을 stack에 적용하면 반대가 나온다.** `KindShape`/`KindImage`는 여러 층의
  산출이 달라서 갈렸다. `<stack>`은 파서(콘텐츠 모델·자식 어휘), core(컨테이너 속성 해석),
  PPTX(도형 0개 + 자식 재귀), 지오메트리 패스(재지만 좌표 미소비)에서 `<row>`와 산출이 **전부
  같다**. 다른 것은 웹 CSS 한 줄(단일 셀 grid)뿐이고, 그 한 줄은 이미 종류가 아니라 **흐름**으로
  갈린다 (`<row>`=`flex-direction:row`, `<grid>`=`display:grid`, `<stack>`=단일 셀 grid).
- **`KindStack`은 `<row>`의 기계를 통째로 복제해 CSS 한 줄을 바꾼다.** 종류를 나누면
  파서·core·PPTX·지오메트리의 분기가 저마다 stack을 알아야 하고, 그 중 하나를 빠뜨리면
  "파싱은 되는데 그려지지 않는" 조용한 실패가 된다. `FlowOf`의 존재가 곧 확장 지점이다 —
  흐름을 하나 더하는 것으로 끝난다.
- **`<group>`(Iteration 9)이 이 판단을 다시 받는다 — 그리고 반대로 답한다.** 그룹은 배치 후
  소멸하지 않고 `grpSp`로 **남으며** 그룹 상대 좌표를 낸다. PPTX 산출이 실제로 다르므로
  `<group>`은 새 흐름이 아니라 새 종류(`KindGroup`)다. 두 태그가 갈라지는 자리가 이 ADR의
  기준을 증명한다: 산출이 다르면 종류, 같으면 흐름이다.

## 결과

- `soml/placement.go`의 `ContainerFlow`에 `FlowStack`이 더해지고 `FlowOf("stack")`이 그것을
  돌려준다. `soml.KindOf("stack")`은 `KindLayoutContainer`다 — `<row>`와 같다.
- **`TestRenderStackEmitsNoShapesLikeLayoutContainers`가 stack을 `<row>`·`<col>`과 나란히 세워**
  셋의 PPTX 산출(도형 0개 + 자식만)이 같은지 못 박는다. stack이 `grpSp`를 내기 시작하면 여기가
  먼저 빨개진다.
- 지오메트리 패스는 stack을 컨테이너로 잰다 (재고, 좌표 미소비 — 다른 컨테이너와 같은 칸,
  [지오메트리 패스](../internals/geometry-pass.md)).
- `<group>`(Iteration 9)이 들어올 때 이 ADR이 "stack은 여전히 소멸한다"의 근거로 읽힌다.
