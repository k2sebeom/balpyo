# 지오메트리 패스

PPTX 렌더러는 좌표 계산을 하지 않는다. 웹 렌더러가 만든 HTML을 헤드리스 브라우저에 올려
**요소별 좌표를 읽고**, 그 값을 그대로 OOXML `xfrm`으로 눕힌다. 이 문서는 그
패스의 전체 그림과 순서를 담는다 — 개별 결정의 *왜*는 코드 주석(`render/geom`의 패키지
주석, `measureScript`, `pptx.Offset`, `pipeline.needsGeometry`)이 소유하고, 여기서는
어디에 무엇이 있는지를 잇는다.

## 왜 브라우저인가

저자가 좌표를 만지지 않게 하려는 것이 이 제품의 핵심이다. flex/grid로 선언한 배치를
계산할 엔진이 필요하고, 웹 타깃은 어차피 브라우저가 그린다 — 같은 엔진이 PPTX의 배치도
정하면 두 타깃의 배치가 **구조적으로** 일치한다. 레이아웃 엔진을 하나 더 구현하면(예:
Yoga) 텍스트 메트릭과 flex 기본값의 차이가 조용한 편차로 쌓인다. 설계 단계의 벤치마크에서
Yoga는 156배 빨랐지만 텍스트 엔진이 없어 최대 34px 어긋났고, `flex-shrink` 기본값이 달라
경고 없이 어긋났으며, CSS grid를 지원하지 않았다. 시스템 브라우저를 CDP로 제어하는 쪽은
슬라이드당 약 1.5ms, 콜드 스타트 0.6초였다. 인프로세스 백엔드는 세 조건이 모두 갖춰지기
전에는 착수하지 않는다 ([ADR 0067](../decisions/0067-prd-is-retired-and-its-standing-decisions-move-here.md) 결정 8).

브라우저는 다운로드하지 않고 시스템에 있는 것을 CDP로 직접 제어한다 (`render/geom/browser.go`
의 `Candidates` — 환경 변수 `BALPYO_CHROMIUM` > PATH > 알려진 설치 경로).

## 흐름

```
pipeline.Render
  │ needsGeometry?  ← PPTX 타깃이나 연결선이 있고, 잴 요소(geom.Refs)가 있을 때만
  ↓
web.NewForMeasurement()         웹 렌더 (임시 디렉토리 — 산출물 쓰기 전에 측정을 끝낸다).
  .Render(deck)                 연결선은 내지 않는다 — 흐름에 끼지 않으므로 배치가 같다
  ↓
geom.Launch → Session.Open      브라우저 기동, 폰트 로드·이미지 디코드 완료까지 대기
  ↓
measureScript 평가 (1회)         페이지 전체를 스크립트 한 번으로 읽는다
  ↓
geom.newGeometry                덱과 대조 → geom.Geometry (곁 표)
  ↓
pptx.NewWithGeometry(g).Render  xfrm = TextBoxRect(측정 rect, 정렬) → EMU
web.NewWithBoxes(g.RouteBox)    사용자가 받는 HTML — 연결선의 끝점이 잰 rect에서 나온다
  .Render(deck)                 (render/route가 두 렌더러의 경로를 한 번 계산한다)
```

- **연결선은 측정의 출력이다** ([ADR 0080](../decisions/0080-a-connector-joins-two-measured-boxes-and-is-drawn-after-measurement.md)).
  측정용 HTML과 사용자가 받는 HTML은 연결선 오버레이 하나만 다르다 — 그래서 연결선이 있는 덱은
  `--target html`만 써도 잰다.

- `Session.Open`은 페이지를 연 직후 **문서 뷰로 되돌린다** (`web.DocumentViewExpr`).
  산출물의 뷰어 런타임은 로드 즉시 발표자 뷰 — 슬라이드를 겹치고 화면에 맞춰 스케일한
  배치 — 를 켜는데, 측정은 스케일 없는 세로 스택에서 해야 슬라이드 rect가 논리 스테이지와
  1:1이다. 뷰 전환은 `<html>`의 속성 하나다
  ([ADR 0006](../decisions/0006-presenter-view-attribute.md)).
- `--target html`만 쓰는 저작 루프는 브라우저를 띄우지 않는다 — 기동이 빌드의 가장 큰
  고정 비용이라서다 (`pipeline.needsGeometry`).
- 노트만 있는 슬라이드처럼 잴 요소가 없는 덱도 브라우저 없이 PPTX가 나온다 — 판정 기준이
  PPTX 렌더러의 요구(`geom.Refs`가 비었는가)와 같다.

## 무엇을 재는가

측정은 스크립트 한 번의 평가다 (`measureScript`) — 요소마다 CDP 왕복을 하면 왕복 사이에
페이지가 바뀔 수 있고, 그러면 요소들의 좌표가 서로 다른 레이아웃에서 나온 값이 된다.

읽는 값:

| 값 | 용도 |
|---|---|
| `devicePixelRatio` | 1이 아니면 실패 — 논리 px와 픽셀이 갈린 상태다 |
| 슬라이드 rect | 논리 스테이지와 다르면 실패 — 페이지가 스케일된 상태다 |
| 요소 rect (`getBoundingClientRect`) | PPTX 텍스트 박스의 위치·크기 |
| 줄 박스 (`Range.getClientRects`) | 줄바꿈 검증 — 브라우저가 어디서 줄을 나눴는지 |

### 누구를 재는가 — 박스인 것 전부

측정 대상은 **박스인 요소**다: `<text>`·블록·리스트 컨테이너·`<li>`, 레이아웃
컨테이너(`<row>`/`<col>`/`<grid>`/`<stack>`)와 `<spacer>`, `<shape>`와 **도형 안 텍스트**, 그리고
`<image>`. 인라인 요소는 재지 않는다 — 여러 줄에 걸친 런의 합집합 사각형은 PPTX에서 받을 자리가
없다 ([ADR 0009](../decisions/0009-measured-kinds.md),
[ADR 0013](../decisions/0013-measured-kinds-include-layout.md)).

| 요소 | 재는가 | 좌표가 소비되는가 |
|---|---|---|
| `<text>`·블록 태그 | 재고 | **소비된다** — 텍스트 박스의 `xfrm` |
| `<ul>`/`<ol>` | 재고 | 소비된다 (`<ul>` 하나가 도형 하나다) |
| `<li>` | 재고 | 소비되지 않는다 — 문단이 부모 박스에 산다 |
| 레이아웃 컨테이너 · `<spacer>` | 재고 | 소비되지 않는다 — 컨테이너는 도형이 아니다 ([ADR 0011](../decisions/0011-layout-containers-are-not-shapes.md)) |
| `<stack>` | 재고 | 소비되지 않는다 — 겹침 컨테이너도 도형이 아니다 (`<row>`와 같은 칸 — [ADR 0028](../decisions/0028-stack-is-a-flow-not-a-kind.md)) |
| `<shape>` | 재고 | **소비된다** — `sp`의 `xfrm` |
| 도형 안 텍스트 | 재고 | 소비되지 않는다 — 부모 도형의 `txBody`가 된다 ([ADR 0016](../decisions/0016-shape-text-is-shape-body.md)) |
| **`<image>`** | **재고** | **소비된다** — `pic`의 `xfrm` (여백 보정이 없다) |

**`<shape>`와 `<image>`는 재는 것과 도형이 되는 것이 일치하는 두 종류다** — 잰 rect가 그대로
`xfrm`이 된다 (그 박스는 글자를 잰 결과가 아니라 저자가 정한 크기다). 도형의 **자식 텍스트는 반대**다: 재지만 소비되지 않는다. 그래서 종류 판정이
태그 이름만으로는 부족하고 **부모가 도형인가**를 알아야 한다 (도형 안 `<p>`와 슬라이드 직속
`<p>`는 같은 태그다).

**그림에서 재는 요소는 `<img>`이고 저자가 준 박스가 아니다.** 두 층으로 나온 것 중 식별자가
붙는 쪽이 `<img>`이기 때문이고, 그 이유는 `fit="contain"`에서 둘이 갈라지는데 `pic`의 `xfrm`이
되어야 하는 것은 **그려진 그림**이라서다 ([ADR 0026](../decisions/0026-contain-fits-the-box-to-the-picture.md)).
그 결정 덕분에 이쪽에는 letterbox·여백 계산이 하나도 없다 — `pad`가 붙은 그림도 잰 rect가 이미
여백 안쪽의 그림이다.

컨테이너를 재 두는 값은 진단이다 — 배치가 갈라졌을 때 "어느 칼럼부터 밀렸는가"를 답하는
것은 컨테이너 좌표이고, 그리드 트랙의 등간격이나 중첩 컨테이너의 좌표 정규화를 단정하는
테스트의 입력도 그 rect다. 그래서 표의 항목 수는 도형 수보다 많고, 그 차이가 픽스처 수준의
계약이다: **도형 = 측정 박스 − 레이아웃 컨테이너 − spacer − `<li>` − 도형 안 텍스트**
(왼쪽의 "도형"은 `p:sp`와 `p:pic`을 함께 센다).

### 측정 전에 기다리는 것 — 폰트와 그림

`Session.Open`은 페이지를 연 뒤 **두 가지가 끝날 때까지 기다린다**: 웹폰트 로드
(`document.fonts.ready`)와 **이미지 디코드**. 근거가 같다 — 둘 다 끝나기 전에 재면 브라우저가
아직 없는 것을 기준으로 배치를 계산하고, 그 좌표가 그대로 PPTX `xfrm`이 된다.

- 폰트가 없으면 폴백 폰트의 메트릭으로 줄이 나뉘고, 그림이 없으면 **rect가 0×0이다** (실측:
  디코드 전 `naturalWidth === 0`이고 `contain`에서는 그 0이 그대로 잰 값이 된다).
- **로드 실패는 에러다.** 판정은 `decode()`의 거부가 아니라 `naturalWidth === 0`에 둔다 — 404는
  `complete`가 true인 채 고유 크기만 0으로 남는다. 그림 없는 PPTX는 **정상으로 열리므로**
  (빈 자리가 있는 슬라이드) 조용히 통과시키면 저자가 발표 직전에 발견한다.
- 산출물에 에셋 바이트가 있다는 것이 이 대기의 전제다. 그림은 파이프라인이 나중에 복사하는
  것이 아니라 웹 렌더러의 반환값에 들어 있고, 측정은 그 파일들을 임시 디렉토리에 전부 쓴 뒤에
  시작한다 ([아키텍처](architecture.md)의 에셋 표 절).

### 회전 도형은 회전 전 박스를 잰다

`getBoundingClientRect`는 변형이 걸린 요소에서 **회전된 도형의 외접 사각형**을 돌려준다
(45도 회전한 120×120 도형은 169.7×169.7로 읽힌다). 그 값을 `xfrm`에 쓰면 PowerPoint에서
도형이 실제보다 크고 그 위에 회전이 또 얹혀 그림이 웹과 갈라진다.

그래서 측정 스크립트는 **측정 뷰를 켜고 재고 되돌린다** — 뷰는 `<html>`의 속성 하나이고
(ADR 0006의 수단을 세 번째 뷰로 재사용한다) 그 뷰의 엔진 CSS가 도형의 변형을 끈다. 회전
선언이 요소의 인라인 `style`이 아니라 엔진 CSS의 클래스 + 커스텀 프로퍼티로 갈라져 있는
이유가 이것이다: 인라인 선언은 cascade layer 밖에서 이기므로 `!important` 없이는 무효화할 수
없다 ([ADR 0021](../decisions/0021-measure-view-disables-transforms.md)).

**그림도 같은 장치를 쓴다** (실측: 200×100 그림에 `rotate(15deg)`를 걸면
`getBoundingClientRect`가 219.07×148.36을 준다). 다른 점은 하나뿐이다 — 도형은 대칭이 그림 svg에
걸려 측정 뷰가 건드리지 않지만 그림은 회전과 대칭이 **한 `transform`**을 나눠 쓰므로 함께 꺼진다.
대칭은 외접 사각형을 바꾸지 않으므로 잰 rect가 같다.

되돌리는 것(`finally`)도 계약이다 — 같은 세션이 재고 스크린샷을 찍으므로 측정 상태가 남으면
그림에서 회전이 사라진다. 산출물 `xfrm@rot`이 회전 전 크기인지는 산출 XML만 읽는 테스트가
따로 단정한다 (좌표 대조로는 볼 수 없다: 측정이 bbox를 재기 시작하면 두 값이 함께 커지며
통과한다).

### 위치는 변형으로 내지 않는다

같은 이유가 겹침의 `x`/`y` 좌표에도 걸린다. 자식의 절대 위치는 `left`/`top`으로 내고
**`transform: translate()`로 내지 않는다** ([`<stack>`](../../docs/reference/soml/stack.md),
`render/web/layout.go`의 `appendPlacement`). 측정 뷰가 변형을 끄므로(위와 같은 장치), 위치를
`translate`로 내면 지오메트리 패스가 잰 rect에 그 이동이 **남지 않고** PPTX는 겹치지 않은
좌표를 받아 웹과 갈라진다. `translate`가 더 짧다고 바꾸는 것을 막는 것이 이 문장이다 —
회전(변형이라 끈다)과 위치(좌표라 남겨야 한다)가 측정 뷰에서 갈라지는 자리다.

## 좌표계 — 슬라이드 상대

모든 rect는 **슬라이드 좌상단이 원점**이다. 문서/뷰포트 좌표를 그대로 넘기면 덱 여백·
슬라이드 간격·스크롤량이 좌표에 섞이고, PPTX에서 "모든 도형이 아래로 밀렸다"로 나타난다
(Iteration 0에서 요소 스크린샷이 같은 방식으로 어긋났다). 슬라이드
자신의 rect를 빼면 그 항들이 전부 상쇄된다 — 두 번째 슬라이드의 첫 요소가 첫 번째
슬라이드의 첫 요소와 같은 좌표로 나오는 것이 검증 방법이다.

**중첩 컨테이너 안의 요소도 같은 원점을 쓴다.** `<col pad="lg">` 안의 `<row>` 안의
`<text>`가 슬라이드 상대 좌표로 나오므로, 중첩 깊이가 좌표에 영향을 주지 않는다 — PPTX
조립기에 부모 좌표를 더하는 코드가 없는 것이 "좌표 계산 0회"의 나머지 절반이다 (컨테이너
상대 좌표를 만들었다면 조립기가 트리를 따라 누적 오프셋을 관리해야 했다).

## 덱과의 대조 — 좌표가 엉뚱한 요소에 붙지 않게

요소 식별자(`data-balpyo-id`)의 **구현은 하나다**: `web.ElementID`가 덱 모델의 슬라이스
인덱스에서 `s<슬라이드>.<자식>…`을 만들고, 렌더러가 그 값을 속성으로 내고, 지오메트리
패스가 같은 함수로 기대 목록을 만들어 브라우저가 읽어 온 목록과 **자리 하나씩** 대조한다
(개수·순서·식별자·태그 이름). 대조가 끝나야 `geom.Geometry`가 만들어지고, 소비 측
(render/pptx)은 `Verify`로 한 번 더 대조한다 — 좌표가 엉뚱한 요소에 붙은 PPTX는 열어
보기 전까지 정상으로 보이기 때문에, 이 실패는 조용해서는 안 된다.

## 지오메트리는 덱이 아니라 곁 표다

측정값은 덱 모델의 필드가 아니라 별도 타입(`geom.Geometry`)으로 PPTX 렌더러 생성자에
주입된다 — 덱은 공개 계약이고, 덱에 실으면 값이 채워지는 시점이 `--target` 순서에
의존한다. [ADR 0004](../decisions/0004-geometry-side-table.md)에 결정 전체가 있다.

## 단위 경계 — EMU·pt는 render/pptx만 안다

패스가 주고받는 수치는 전부 **논리 스테이지 px**다 ([ADR 0067](../decisions/0067-prd-is-retired-and-its-standing-decisions-move-here.md) 결정 5). EMU·pt 환산은
`render/pptx/units.go`의 `Offset`/`Extent`/`FontSize`/`LineSpacing`에서만 일어난다
(`1px = 9525 EMU`, `pt = px × 0.75`). `render/geom`의 코드에 EMU가 나오지 않는 것을
테스트가 감시한다 (`TestGeometryPassKnowsNoEMU`) — 환산이 흩어지면 두 번 환산된 좌표나
환산되지 않은 좌표가 생기고, 어느 값이 어느 단위인지 타입으로 드러나지 않는다.

## 텍스트 박스 고정 전략

측정된 rect는 **그대로** 박스가 된다 — 정책은 **박스 고정 + 넘치면 뷰어가 줄여 맞춤**이다
([ADR 0064](../decisions/0064-the-contract-is-component-and-layout-parity-not-pixel-parity.md)):

- 폭에 여유를 더하지 않는다. 예전의 안전 여유 +4%([ADR 0005](../decisions/0005-line-spacing-and-slack.md))는
  다른 엔진이 같은 줄을 같은 자리에서 접게 하려는 장치였고, 줄이 어디서 접히는지는 이제 계약 밖이다.
  잰 rect가 그대로 `xfrm`이어야 동일성 판정이 렌더러의 보정 없이 두 트리를 맞댈 수 있다.
- `a:bodyPr`은 인셋 0 + `wrap="square"` + `a:normAutofit`이다. 웹이 잰 rect는 border box이고
  `.balpyo-text`는 padding·border가 0이라 곧 콘텐츠 박스다 — OOXML 기본 인셋을 두면 좌표는 맞는데
  글자만 밀린다. 뷰어가 한 줄 더 접어 박스를 넘기면 `normAutofit`이 글자를 줄여 박스에 맞춘다 —
  박스는 움직이지 않으므로 이웃이 밀리지 않는다. `<pre>`만 `wrap="none"` + `a:noAutofit`이다 (줄 수가
  원문의 줄 수로 정해져 있다).
- **도형의 `bodyPr`만 인셋을 갖는다** — 저자의 `pad`가 그 값이고, 웹에서 같은 값이 도형 div의
  padding이며 `box-sizing: border-box`라 잰 박스를 키우지 않는다. 그림에는 여백 보정이 없다 (`pad`는
  웹에서 박스가 흡수하고 잰 rect는 이미 그림이다).
- 줄간격은 절대값(`a:spcPts`)이다 — 배수(`spcPct`)는 엔진마다 기준이 달라 글줄이 밀린다.
- 스테이지를 넘어간 박스도 그대로 간다 — 음수 좌표·넘치는 크기를 자르지 않는다 (오버플로
  허용 — 넘침은 `--check-layout`이 경고로 알린다).

측정이 남기는 줄 상자(`Box.Lines`)는 텍스트 이스케이프 판정에만 쓰인다 — 줄 수 · 여유 폭 같은 줄
메트릭은 계약 밖이라 지오메트리에 도우미가 없다. 엔진마다 남는 차이는
[계약 밖의 차이](../../docs/guides/known-differences.md)에 있다.

## 테스트 하네스와의 관계

테스트가 쓰는 브라우저 헬퍼(`internal/testkit.Browser`)는 이 패스의 세션(`geom.Session`)을
감싼 것이다 — CDP 클라이언트·기동 플래그·좌표 스니펫이 두 벌이 되면 테스트가 재는 좌표와
빌드가 재는 좌표가 다른 조건에서 나오고, 그 차이는 원인 없는 rect 차이로만 보인다. 테스트는
바이너리마다 세션 하나를 나눠 쓴다 (`testkit.SharedSession` — [테스트 환경](testing.md)). `balpyo preview`의
웹 스크린샷도 같은 세션의 좌표에서 잘라낸다 (`Session.Clip`).
