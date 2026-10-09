# 0034 — 그라디언트는 SVG def로 내고 objectBoundingBox로 늘린다

- 상태: 확정
- 날짜: 2026-09-02
- Iteration: 8 (그라디언트 + 색 변형 — NOTES.md 08이 근거 기록이다)
- 관련: PRD §7.6, §9.4, §13-5, [외관 속성](../../docs/reference/appearance.md),
  [testing.md](../internals/testing.md), ADR 0017 (도형 기하는 프리셋에서 유도한다)

## 맥락

`gradient(색… [, 각도deg])`가 채움·테두리·텍스트 색에 들어온다. 도형은 SVG로 그려지므로
(ADR 0017) 채움·테두리 그라디언트를 **어디에** 담고, CSS 각도를 두 타깃의 좌표계로 **어떻게**
옮길지가 이 결정이다. PRD §9.4는 `border=… gradient`를 "엔진 구현(SVG 오버레이/mask — CSS
`border-image`는 radius와 조합 불가)"으로 못 박아 두었다.

## 선택지

1. **채움은 CSS `background-image: linear-gradient`, 테두리는 `border-image`** — 도형이
   `<div>`라면 자연스럽지만 도형은 SVG다. `border-image`는 `border-radius`와 조합되지 않아
   둥근 도형의 테두리 그라디언트가 깨진다.
2. **SVG `<defs><linearGradient>` + `fill`/`stroke="url(#id)"`** — 도형 svg 안에 그라디언트
   정의를 만들고 채움·테두리가 가리킨다. 텍스트 색은 `background-clip: text`.

## 결정

**선택지 2.** 도형 svg가 요소 id(`ElementID`)로 이름 붙인 `<linearGradient>`를 `<defs>`에
담고 `fill: url(#id-fill-grad)`·`stroke: url(#id-stroke-grad)`로 가리킨다. 각도는
`objectBoundingBox`(기본 gradientUnits)의 시작·끝 벡터로 옮긴다. PPTX는 `a:gradFill` +
`a:lin@ang = (css−90) mod 360 × 60000`, `scaled="1"`이다. 텍스트 색 그라디언트는 웹
`background-clip: text`, PPTX `a:rPr/a:gradFill`이다.

## 근거

- **채움과 테두리가 이미 SVG에 있으니 그라디언트는 def 하나를 더할 뿐이다.** 12종 도형이 전부
  SVG인 이유가 "외관을 두 벌로 구현하지 않는다"였고(ADR 0017), 그 결정이 여기서 회수된다 —
  `fill`/`stroke`가 `url(#…)`을 받으므로 단색과 같은 자리에 그라디언트가 온다. `border-image`
  갈래는 둥근 도형에서 깨지고 도형별 오버레이 요소를 새로 만들어야 한다.
- **각도를 objectBoundingBox로 옮긴 이유**는 OOXML `a:lin@scaled="1"`과 같은 상대 좌표라
  세로·가로에서 두 타깃이 정확히 일치하기 때문이다. `objectBoundingBox`는 그라디언트 좌표를
  도형 박스에 상대화하고, `scaled="1"`도 그라디언트를 박스에 맞춰 늘린다 — 둘 다 "박스 기준"
  이라 180deg(위→아래)·90deg(왼→오)가 아귀가 맞는다.
- **대각 각도의 비정사각형 갈라짐은 알려진 v1 한계다.** `objectBoundingBox`는 45deg를
  박스 종횡비만큼 기울여 그리는데, LibreOffice의 `scaled` 해석이 그 기울임과 미세하게 달라
  비정사각형 박스에서 대각 그라디언트가 갈라진다 (측정: 220×120 45deg 카드가 3.7%, 정사각형
  140×140으로 바꾸니 3.3%로 떨어졌다 — 나머지는 3색 스톱 몫). 그래서 대각 그라디언트는
  정사각형 도형을 권하고, 세로·가로·정사각-대각은 종횡비와 무관해 일치한다. 웹이 도형의 픽셀
  크기를 모르는 것(브라우저가 배치를 계산하고 렌더러는 그 결과를 소비한다 — ADR 0017의 정규
  공간 늘림과 같은 구조)이 `userSpaceOnUse`로 정확한 벡터를 낼 수 없는 이유다.
- **다중 스톱은 뷰어 소관이다.** 3색 이상 그라디언트의 OOXML은 스펙대로 `a:gs`를 스톱 수만큼
  내지만(07_gradient/fill-gradient 골든에서 accent1 pos0 → tx2 pos50000 → bg2 pos100000 확인),
  테스트 래스터라이저 LibreOffice는 다중 스톱을 근사해 그린다. PowerPoint/Keynote는 모든
  스톱을 그대로 그리므로 이 갈라짐은 산출물 결함이 아니라 뷰어 소관이다 — 픽셀 게이트에서
  픽스처 한정 여유(`07_gradient/fill-gradient` MaxRatio 3.8%)로 흡수하고 사람이 PowerPoint로
  확인한다. 스톱 참조·위치·각도 자체는 구조 테스트가 0 오차로 못 박아 픽셀 여유가 갈라짐을
  덮어 감추지 않는다.
- **텍스트 색은 `background-clip: text`다** — 배경 그라디언트를 글자 실루엣으로 자르고 채움을
  투명하게 한다. PPTX는 `a:rPr` 안의 `a:gradFill`로, LibreOffice에서도 그대로 그려진다
  (07_gradient 슬라이드 2의 "그라디언트 제목"이 두 타깃에서 파랑→보라→분홍으로 일치).

## 결과

- `soml/appearance.go`: `Color.Gradient *Gradient`, `Gradient{Stops, Angle}`, `GradientColor`.
  스톱은 단색만 담는다 (중첩 불허).
- `core/shape.go`: `border()`가 `gradient(` 접두를 보고 `<굵기> gradient(…)` 2조각으로 간다.
- `render/web/shape.go`: `shapeInkCSS`가 `<defs>`를 함께 낸다, `linearGradientDef`,
  `gradientVector`(dir=(sinθ,−cosθ), 중심±0.5). `render/web/style.go`: `gradientCSS`,
  `textStyleCSS`의 `background-clip:text` 갈래.
- `render/pptx/shape.go`: `writeGradFill`(gsLst/gs pos/lin ang scaled=1), `writeFill`·
  `writeLine`·`writeTextFill`의 그라디언트 갈래. `render/pptx/units.go`: `GradientAngle`.
- 픽스처 `testdata/fixtures/07_gradient/`가 채움·테두리·텍스트 그라디언트와 각도 3종을
  덮는다. 대각 카드는 정사각형이다 (위 한계).
