## Iteration 04 · 01-parser-shape — 파서가 `<shape>` 어휘를 연다

<!-- item: 04/01-parser-shape -->

- **한 것**:
  - `soml/shape.go` 신규 — 도형 카탈로그. `ShapeElement` 상수, `type` 12종의 **순서 있는**
  목록(`shapeTypes` + `ShapeTypes()`), 속성 목록 2벌(`shapeOwnAttrs` = `type`·`id`,
  `shapeAppearanceAttrs` = `variant` `fill` `border` `radius` `shadow` `opacity` `rotate`
  `flip`), 파생 함수 `ShapeAttrs()`·`ShapeAppearanceAttrs()`·`shapeChildren()`,
  그리고 `AnchorAttr` 상수.
  - `soml/subset.go` — `ElementKind`에 `KindShape` 추가(열거형은 여기 모은다는 03/01의
  결정 그대로), `buildElementKinds`가 `<shape>`를 색인, 타입·색인 주석에 `shape.go`를 넣었다.
  - `soml/layout.go` — **박스 속성 카탈로그 재분할**: `containerFlowAttrs`에서 `pad`를 떼어
  `boxPadAttrs`로, `containerBoxAttrs`를 `boxSizeAttrs`로 개명하고 둘을 잇는 `BoxAttrs()`를
  내보냈다. `ContainerAttrs`는 `flow + pad + (track) + size` 순으로 이어 **기존 순서를 그대로**
  유지한다. `layoutChildren()`에 `<shape>` 추가.
  - `soml/parse.go` — `buildElementSpecs`에 `<shape>`(속성 = `withChildPlacement(ShapeAttrs())`,
  모델 = `modelElements`, 자식 = `shapeChildren()`), `<text>`·블록 속성 표에 `anchor`,
  `slideChildren()`에 `<shape>`. 어휘 개방 주석에 Iteration 4를 적었다.
  - `soml/shape_test.go` 신규 — 정상 트리 14건(고유/외관 8종/박스/자식 배치/style/도형 안
  text·블록 여럿·인라인·pre/슬라이드 직속/컨테이너 안 카드/자기 닫힘/12종 밖 `type`/`type`
  없음), `anchor` 3건, Span 계약(짝 태그·자기 닫힘), 에러 24건(line:col + 문구),
  종류 카탈로그 계약(`KindShape`·void 아님·역할 없음·I5·7·9 태그 부재),
  `type` 12종 순서와 카탈로그 복사 방어, **박스 속성 공유 회귀**(`ContainerAttrs` 3태그의
  목록·순서 동일 + 도형이 같은 목록을 받는지 + `<text>`·블록에는 열리지 않았는지),
  `anchor` 커버리지 양방향, 도형의 자식 배치 속성, 픽스처 파싱(도형 12개 하한).
  - `testdata/fixtures/_04_shape/`(비활성) — `balpyo.json` + 슬라이드 8개:
  `001_shape_types`(`<grid cols="4">`에 12종 전수 — 항목 04·05의 프리셋 기하 시험대),
  `002_token_colors`(토큰 색 + `border="1 solid muted"` + `fill="none"` — `schemeClr` 보존 계약),
  `003_literal_colors`(리터럴 색만 — `srgbClr` 계약 충돌을 항목 06이 표로 다루도록 분리),
  `004_appearance`(radius 토큰·수치, shadow, opacity, rotate, flip — 나머지 조건 고정),
  `005_text_anchor`(`anchor` 3종 × `pad` + 자식 여럿), `006_cards`(4열 카드 그리드 —
  PRD §7.8 dim-card를 손으로 펼친 것 + `span="2"` 배너), `007_variant`(도형 variant +
  인라인이 덮는 장면), `008_variant_unknown`(모르는 variant 경고 경로).
  - `soml/subset_test.go`·`soml/layout_test.go` — 속성·자식 어휘가 넓어져 진단 문구가 바뀐
  기존 단정 2건 갱신(`<h1>`의 허용 속성에 `anchor`, `<row>`의 자식 어휘에 `<shape>`).
  트리·배치 규칙 단정은 하나도 손대지 않았다.
- **설계 결정과 근거**:
  - `**<shape>`는 별도 종류(`KindShape`)다.** `KindSpacer`를 나눈 것과 같은 근거이지만 이유가
  한 겹 더 있다: 도형은 **재는 것과 도형이 되는 것이 처음 일치하는 종류**다. 컨테이너는
  재고 사라지고, `<li>`는 재지 않고 부모 도형에 담기는데, `<shape>`는 잰 rect가 곧 `xfrm`이
  되면서 자신도 텍스트를 담는다. 종류를 나누지 않으면 geom·web·pptx가 각자
  `name == "shape"` 비교를 든다.
  - **크기·여백 목록을 컨테이너와 공유한다** (`BoxAttrs`). 크기는 컨테이너의 성질이 아니라
  박스의 성질이므로 목록이 두 벌이 되면 PRD 표에 크기 속성이 하나 늘 때 어느 태그가 빠졌는지
  사람이 세어야 한다 — `withChildPlacement`와 같은 근거다. **순서는 그대로 두었다**:
  `pad`를 flow에서 뗐지만 `ContainerAttrs`가 `flow + pad + track + size`로 이어 붙여
  기존 진단 문구가 한 글자도 움직이지 않는다 (그 사실을 새 테스트가 세 태그로 못 박는다).
  도형에서는 트랙이 없으므로 `BoxAttrs()` 한 벌을 그대로 받는다.
  - **도형에 `w`/`h`/`ratio`를 열었다.** 도형은 그 자체가 그림이라 내용이 없으면 잰 크기가 0이고
  0×0 도형은 산출물에서 아무것도 아니다 — 빈 `<text>`가 텍스트 박스를 내는 정책과 갈라지는
  자리다(그쪽은 저자가 자리를 확인해야 하는 콘텐츠다). 크기 없이는 배지·구분면·카드 배경이
  전부 닫힌다. 반대로 `<text>`·블록에는 열지 **않았다**: 블록의 폭은 엔진 CSS가 100%로
  정하고 있어(`.balpyo-text`) 여기서 열면 기존 골든 4개가 이 iteration과 무관한 이유로 움직인다.
  - `**anchor`는 `<text>`와 블록 둘 다에 열었다** (항목 지시가 남긴 결정). 등가 표의
  대칭이 근거다: 블록은 역할의 축약이므로(`<p>` ≡ `<text role="body">`) 블록이 받는 외관은
  `<text>`가 받는 것과 같아야 등가가 속성 표에서도 성립한다. `<text>`에만 열면 카드 본문을
  `<p>`로 쓰던 저자가 앵커를 쓰려고 태그를 바꿔야 하고, 그 순간 등가는 문서상의 약속으로만
  남는다. `<li>`·인라인·컨테이너·**도형 자신**에는 열지 않았다 — 도형에 열면 같은 뜻의 선언이
  두 자리에 생기고 어느 쪽이 이기는지 정할 근거가 없다.
  - **도형의 자식은 텍스트 요소(`<text>` + 블록)뿐, 개수 제한 없음.** 리스트는 불릿 문단
  조립과 도형 `txBody` 조립을 합치는 일이라 어휘를 여는 것과 다른 크기의 작업이고,
  컨테이너·도형 중첩은 "도형은 `txBody` 하나"라는 이번 매핑과 부딪힌다 (겹침은 I6,
  편집 단위는 I9). 개수는 문법으로 세지 않았다 — 자식 여럿은 `txBody`의 문단 여럿으로
  이미 뜻이 있다(`writeListShape`의 선례).
  - `**id`는 어휘만 열었다.** 산출물에 아무것도 그리지 않는 속성이고 `<slide id>`의 선례가
  그렇다. 중복 검사도 하지 않았다 — 참조(`<arrow from>`)는 Iteration 7이고, 지금 할 수 있는
  검사가 무엇인지는 값 층(항목 02)이 판단한다.
  - `**stroke`/`cap`/`dash`는 열지 않았다** (line·arrow의 것). 도형에 받아 놓고 버리는 선언이
  되고, 그 침묵이 닫힌 어휘가 막으려던 실패다. `<image>`·`<arrow>`·`<line>`·`<group>`·
  `<icon>`·`<table>`·`<chart>`가 어휘 밖임을 에러 테스트가 못 박아, 뒤 iteration 전에 누가
  픽스처에 쓰면 빨갛게 된다.
  - **속성 목록을 내보냈다**(`ShapeAttrs`·`ShapeAppearanceAttrs`·`BoxAttrs`·`ShapeTypes`).
  03/01은 "core가 값 어휘를 어떤 모양으로 요구할지 정해지기 전에 표면을 열지 않는다"며
  닫아 두었지만, 항목 02가 실제로 `ContainerAttrs`를 소비하게 된 뒤라 이제는 두 벌이 되는
  쪽이 더 비싸다. 네 함수 전부 복사본을 주고, 호출자가 카탈로그를 흔들 수 없다는 것을
  테스트가 확인한다.
  - **픽스처의 리터럴 색을 슬라이드로 분리했다.** `fill="#0F65F9"`는 정상적으로 `srgbClr`를
  내는데 지금 픽스처 계약은 "슬라이드 파트에 `srgbClr` 0건"이다
  (`internal/testkit.TestTextFixtureGoldenSlidesKeepSchemeClr`). 섞여 있으면 항목 06이
  계약을 슬라이드 단위로 나눌 수가 없다.
- **검증**: `gofmt -l .` 무출력, `go vet ./...`, `**make test`(컨테이너, finch) 전부 초록** —
골든은 하나도 움직이지 않았다 (`git status`에 `testdata/*/golden/**` 변경 없음; 새 픽스처는
`_` 접두로 수집되지 않고 기존 픽스처의 파싱 결과는 그대로다). 호스트 `go test ./...`는
03_layout·02_text_subset 골든의 텍스트 폭(cx)이 다른 **기존** 그 차이만 빨갛고, 같은 실패가
변경 전(`git stash`) 상태에서도 그대로 난다는 것을 확인했다 (macOS 폰트 메트릭 — 게이트는
컨테이너).
게이트 밖 확인: `go run ./cmd/balpyo build --project testdata/fixtures/_04_shape --target html,pptx`가 값 층 진단 0건(경고 1건 = 008의 모르는 텍스트 variant)으로 지나
`slides/001_shape_types.slide:9:5 <shape>를 웹으로 렌더하는 매핑이 없다`로 **명시적으로**
멈춘다 — 어휘는 열렸고 렌더 매핑 부재는 침묵이 아니라 에러라는 사실을 이 항목 상태에서
눈으로 확인했다 (항목 04·05가 이 에러를 없앤다). 이 확인 과정에서 `gap="none"`이 공간 토큰
어휘 밖이라는 것이 드러나 픽스처를 `gap="0"`으로 고쳤다 (아래 열린 질문).
- **남긴 것**:
  - 항목 02·03(core)이 이어받을 표면: `ShapeTypes()`(12종 검사와 모르는 `type` 진단),
  `ShapeAppearanceAttrs()`(캐스케이드가 해석할 속성), `BoxAttrs()`(도형의 크기·여백),
  `AnchorAttr`(값 어휘 — 아래 열린 질문), 그리고 `Appearance`에 도형 외관 형제 필드를 붙이는
  일(`soml/appearance.go`의 주석이 그 자리를 이미 예고한다). 파서는 이름만 봤으므로
  `<shape type="dodecagon">`·`fill="어쩌구">`는 지금 통과한다.
  - 항목 04·05가 이어받을 표면: `KindShape` 분기. 웹은 모양을 그려야 하고 PPTX는 `prstGeom`을
  내야 하며, 컨테이너와 달리 **잰 rect가 그대로 도형 좌표**다.
  - 항목 06: `_04_shape`의 `_` 제거 + 골든. 003의 `srgbClr`와 픽스처 계약 충돌, 008의 경고
  문구가 처음 골든 `diagnostics.txt`에 실린다.
  - 항목 07(문서): `docs/reference/soml/shape.md` 신규와 `appearance.md`의 도형 행. 갱신해야
  하는 낡은 문서는 `soml/slide.md`·`layout.md`의 자식 어휘 목록(`<shape>`가 빠져 있다)과
  `text.md`·`text-subset.md`의 속성 표·진단 예시(`anchor`가 넓어졌다)다.
  - **열린 질문** (PRD/TASKS로 정할 수 없어 임의로 고른 것):
  (a) `**anchor`의 값 어휘.** 픽스처는 `start`/`center`/`end`를 쓴다 — `align`이 논리 이름
  3+1종을 쓰고(`soml.TextAligns`) OOXML `bodyPr@anchor`가 `t`/`ctr`/`b` 셋이라 1:1로 붙기
  때문이다. `top`/`middle`/`bottom`을 쓰는 길도 있지만 그러면 같은 축의 두 속성이 다른
  어휘를 쓴다. 값 층(항목 02)이 정하고, 다르게 정하면 픽스처 005·006을 함께 고친다.
  (b) **도형 variant 이름.** 픽스처 007은 PRD §7.8의 `card-header`/`card-body`를 쓴다 —
  항목 03이 테마에 정의하는 이름이 다르면 그 항목이 픽스처를 고쳐야 한다. 008의 오타
  (`card-heder`)도 같이 움직인다.
  (c) `**surface` 색 토큰**이 픽스처 002·004·005·006에 쓰인다 — `core/theme.go`의 주석이
  "Iteration 4에서 함께 들어온다"고 적고 있으므로 항목 03의 일이고, 지금은 값 층이 도형
  속성을 아예 보지 않아 조용히 지나간다.
  (d) **공간 토큰 `none`.** PRD §7.8의 dim-card 예제는 `gap="none"`을 쓰는데 minimal 테마의
  공간 토큰은 `xs`~`2xl`이라 에러가 난다. 픽스처는` gap="0"`으로 우회했다 — 토큰` none`(=0)을 추가할지는 테마의 결정(항목 03 또는 Iteration 10)이고, 추가하지 않으면 PRD §7.8 예제를 고쳐야 한다. (e) **도형에` align`을 열지 않았다.** PRD §7.6에서` align`은 text 행의 속성이고 도형의 가로 정렬은 안쪽 텍스트가 소유한다고 읽었다. 다만` anchor`(세로)를 텍스트가 갖는 것과
  대칭이므로 이 해석이 맞다고 보았고, 도형 안 텍스트의 가로 정렬이 부모 도형 폭을 기준으로
  도는지는 항목 04가 CSS를 짜 보며 확인할 것.

## Iteration 04 · 02-core-appearance — 도형 토큰 + 외관 값 문법 (단색)

<!-- item: 04/02-core-appearance -->

- **한 것**:
  - `soml/shapestyle.go` 신규 — **확정 도형 외관**. `ShapeStyle`(Type·ID·Fill·Border·Radius·
    Shadow·Opacity·Rotate·Flip + 임베드 `Box`), `Fill`(색 또는 `none`), `Border`(굵기·스타일·
    색 또는 `none`), `Radius`(토큰 이름 + px), 그리고 **`Box`**(`Pad`·크기 6종·`Ratio`) —
    `ContainerLayout`에서 떼어 낸 것을 두 요소가 임베드로 공유한다. `Element.ShapeStyle()`
    접근자도 여기 있다.
  - `soml/appearance.go` — `Appearance.Shape *ShapeStyle`(Text의 형제), `TextStyle.Anchor`,
    `TextAnchor` 어휘(`top`/`center`/`bottom` + `TextAnchors()`), 테마 표에
    `Radii []RadiusToken`·`Shadows []ShadowToken`과 조회·이름 목록 4종.
  - `soml/shape.go` — 값 어휘 두 벌: `BorderStyle`(`solid`/`dashed`/`dotted` + `BorderStyles()`),
    `Flip`(`h`/`v`/`hv` + `Flips()`).
  - `soml/placement.go` — `ContainerLayout`에서 박스 필드(`Pad`·크기·`Ratio`)를 떼고
    `Box`를 임베드했다. **필드 선택자는 그대로다**(`container.Pad`) — 렌더러 3곳(web/pptx/geom)이
    한 줄도 바뀌지 않았고, 그래서 배치 CSS 선언 순서도 그대로다 (03_layout 골든 불변).
  - `core/theme.go` — `surface` 색 토큰(#F3F4F6, 슬롯 lt2 → 참조 bg2), `minimalRadii()`
    (sm 6 · md 12 · lg 24), `minimalShadows()`(soft: dist 4 · dir 90 · blur 16 · fg · alpha 0.15).
  - `core/shape.go` 신규 — 값 해석. `resolveShape`와 속성별 함수(`shapeType`·`shapeID`·`fill`·
    `border`(+`borderWidth`·`borderStyle`)·`radius`·`shadow`·`opacity`·`rotate`·`flip`·`anchor`),
    진단용 어휘 목록 4종.
  - `core/layout.go` — `containerAttr`에서 박스 속성을 떼어 **`boxAttr`**로 만들고 도형과
    공유한다. `splitValues`를 **괄호 인식**으로 바꿨다 (아래 결정).
  - `core/resolve.go` — `resolver.ids`(슬라이드 범위 id 중복 검사 상태), `node`에 도형 갈래,
    `color`를 `colorValue(name, raw, span)`로 쪼개 border 조각이 같은 색 문법을 지나게,
    `inlinePatch`에 `anchor`, `attrExample`에 도형 속성 9종 + `anchor`.
  - `core/cascade.go` — `textPatch.anchor` + `mergeText`의 한 줄 (캐스케이드의 유일한 병합
    지점을 유지).
  - `core/shape_test.go` 신규 — 값 표 30건(속성 전수 × 값 형태), `type` 12종 전수와 필수·어휘
    진단, 색 에러 5건(모르는 토큰·`gradient(…)`·`shade(…)`·잘못된 리터럴), border 축약 7건,
    radius의 형제 속성 검증, 나머지 값 위반 13건, **박스 문법 공유 대조**(도형 Box ==
    컨테이너 Box, `BoxAttrs()` 커버리지), 빈 외관 불변, id 중복(슬라이드 범위), `anchor`
    3종 + 논리 이름 진단 + 미지정 시 빈 값, 도형 안 텍스트 외관 불변, 외관+배치 공존,
    픽스처 `_04_shape` 해석(경고 1건 = 008의 텍스트 variant, 프리셋 12종 도달, 토큰 색 보존).
  - `core/layout_test.go` — `ContainerLayout` 리터럴 10건을 `Box:`로 감쌌다 (단정 내용 불변).
  - `testdata/fixtures/_04_shape/slides/005,006` — `anchor="start"/"end"` →
    `anchor="top"/"bottom"` (어휘 결정에 맞춘 것 — 아래).
  - 골든 8줄: 픽스처 4개의 `index.html`에 `--balpyo-color-surface` 선언 1줄, `theme1.xml`의
    `lt2`가 빌린 값(FFFFFF)에서 surface(F3F4F6)로. **그 8줄이 diff의 전부다.**

- **설계 결정과 근거**:
  - **`Box`를 임베드로 공유한다.** 이름 있는 필드(`container.Box.Pad`)로 두면 기존 렌더러
    전부가 한 겹 더 들어가야 하고, 그 변경이 배치 CSS를 건드릴 위험을 이 항목과 무관한
    이유로 만든다. 임베드는 선택자를 그대로 두므로 골든이 움직이지 않는 것이 구조로 보장된다
    (실제로 렌더러 파일은 하나도 바뀌지 않았다).
  - **`type` 없음·오타 모두 에러.** `role`의 결(진단하고 기본값으로 계속)과 갈리는 근거는
    **기본값의 존재**다: `<p>` ≡ `<text role="body">`는 언어가 정한 등가라 생략에 뜻이
    있지만(PRD §7.7) 도형에는 그런 등가가 없고, `<shape/>`가 조용히 사각형이 되면 저자의
    오타(`type="elipse"`)와 구별되지 않는다.
  - **border 스타일 이름은 CSS 쪽**(`dashed`/`dotted`, OOXML의 `dash`/`dot`이 아니다):
    축약 자체가 CSS `border`의 표기이므로 그 안에 OOXML 어휘를 섞으면 둘 다 아닌 제3의
    표기가 된다. PRD §7.6 표의 `dash` 행 예시도 `dashed`다. 어휘를 셋으로 닫은 기준은
    `prstDash`에 대응이 있는 것만이다 (`double`/`groove`는 PPTX에서 조용히 실선이 된다).
  - **border는 조각 3개 고정 순서**다 (`<굵기> <스타일> <색>`). 순서를 자유롭게 두면
    `border="solid 3"`의 진단이 "무엇이 빠졌는가"를 말할 수 없다. 굵기 0도 에러다 —
    "테두리 없음"의 표기가 둘이 되면(`0 solid muted` vs `none`) 렌더러마다 `w="0"`과
    `noFill` 중 무엇을 낼지 판단이 생긴다.
  - **`splitValues`를 괄호 인식으로 바꿨다.** PRD §7.6의 그라디언트 테두리 표기는
    `border="3 gradient(accent1 accent4 accent5)"`인데, 공백을 모두 조각 경계로 세면 그
    표기가 "5조각"(또는 괄호 인식 후 "2조각") 에러로 끝나 저자가 **아직 없는 기능**이라는
    사실에 도달하지 못한다. 그래서 (1) 괄호 안 공백은 조각이 아니고 (2) 조각에 `(`가 있으면
    조각 수보다 **먼저** 색 문법이 진단한다 ("색 함수 gradient(…)는 아직 쓸 수 없다 —
    Iteration 8"). `pad`·`cols`도 같은 split을 쓰지만 그쪽 값에 괄호가 오면 지금도 에러이고,
    괄호 인식은 그 진단을 조각 수에서 값 문법으로 옮길 뿐이다.
  - **radius는 `type="rounded-rect"`에서만 허용**하고 아니면 에러다 (진단이 고칠 방법을
    말한다). 다른 프리셋에는 조정값을 실을 곳이 없어 **웹만 둥글어지고 PPTX는 각진 채**
    남는다 — 이 테스트가 그 조용한 갈라짐의 회귀 장치다. `type`이 아예 해석되지 않은 도형에는
    두 번째 진단을 얹지 않는다 (고칠 곳이 하나다).
  - **shadow는 토큰만.** CSS 4수치와 OOXML `outerShdw`가 1:1이 아니다 (번짐 정의가 다르고
    `spread`에 대응이 없다). 확정 값은 **토큰 항목 그대로**다(`*soml.ShadowToken`) — 이름은
    웹의 변수 참조, 파라미터는 PPTX가 굽는 값이고, 이는 `Length`가 토큰 이름과 px를 함께
    담는 것과 같은 계약이다. 오프셋을 **극좌표(dist·dir)** 로 담은 이유: OOXML이 극좌표이고
    dir가 60000분의 1도 정수라, 직교 → 극 변환은 산출 경계에서 반올림 손실을 만든다.
  - **radius·shadow 토큰의 clrScheme 슬롯 계약**: radius는 **없다**(OOXML에 반경 테마 개념이
    아예 없다 — `prstGeom/avLst`의 조정값은 도형마다 구워진다). shadow는 **있지만 v1은 쓰지
    않는다**(`fmtScheme/effectStyleLst`가 그 자리인데 스타일이 셋뿐이고 `effectRef`로 고르는
    구조라 도형마다 다른 그림자를 실을 수 없다). 두 문장이 다르므로 주석에 따로 적었다.
    그림자 **색**에는 계약이 걸린다 — `soml.Color`가 토큰 참조를 유지하므로 `outerShdw`가
    `schemeClr`를 낼 수 있다.
  - **`anchor` 어휘는 물리 이름 `top`/`center`/`bottom`**이다. `TextAlign`이 논리 이름을 쓴
    근거는 문단 방향이 가로축의 뜻을 뒤집는다는 것이고 세로축에는 그 근거가 없다 (v1은
    세로쓰기를 지원하지 않는다). 그래서 항목 01이 픽스처에 쓴 `start`/`end`를 `top`/`bottom`으로
    고쳤다. `start`/`end`를 쓰면 대응 이름을 짚는 진단이 난다 (`align`의 left/right와 같은 태도).
  - **`TextStyle.Anchor`는 역할 기본값이 채우지 않는 유일한 필드**다 (미지정 = 빈 문자열).
    기본값을 정하면 뜻이 없는 자리(슬라이드 직속 텍스트 박스 — 잰 rect가 곧 내용 크기라 앵커가
    움직일 여백이 없다)까지 선언이 실려 기존 골든 4개가 이 항목과 무관한 이유로 움직인다.
  - **id 중복 검사 범위는 슬라이드**다 (`resolver`가 슬라이드 하나의 상태라 자리가 곧 범위다).
    참조는 슬라이드를 넘지 않으므로(I7) 덱 전역 유일은 저자가 슬라이드마다 `db`를 쓰는 것을
    막을 뿐이다. `id`를 `ShapeStyle`에 담은 이유: 식별자는 외관이 아니지만(`Appearance.Link`의
    구별) 렌더러가 Attrs를 다시 읽지 않아야 하고, PPTX 도형 이름에 저자의 id를 실을지는 항목
    05의 결정이다. **id의 문자 문법(공백·중복 외 형식)은 검사하지 않았다** — 참조 문법이
    정해지는 I7의 일이다.
  - **`surface` 슬롯은 lt2(참조 bg2)**: surface는 배경이므로 clrScheme의 두 번째 밝은 자리가
    뜻이 맞고, dk2는 이미 muted가 쓴다. 그 결과 `theme1.xml`의 lt2가 "빌린 값"에서 실제
    토큰으로 바뀌었다 (골든 4줄). **`accent2`는 넣지 않았다** — 픽스처가 쓰지 않는다.
  - **variant는 손대지 않았다** (도형이든 텍스트든 도형 쪽 표는 항목 03의 것이다). 그래서
    `_04_shape/007`·`008`의 도형 variant는 지금 조용하고, 커버리지 테스트가 `variant`를
    `pending`으로 명시해 그 항목이 오면 지울 줄을 남겼다.
  - **radius 사다리 6/12/24**: 공간 사다리의 최소 눈금(6px)에 얹혀 xs·sm·md와 같은 값이다 —
    카드가 안쪽 여백과 모서리를 나란히 보이므로 두 값이 같은 눈금이어야 한다. 공간 토큰을
    그대로 쓰게 하는 길은 버렸다 (여백은 리듬, 반경은 형태 — 한쪽을 조율하면 다른 쪽이 끌려간다).

- **검증**: `gofmt -l .`(무출력), `go vet ./...`, `make golden`(8줄 — 위에 전부 적었다),
  **`make test`(컨테이너) 초록**. 컨테이너에서 좌표·픽셀 골든은 **하나도 움직이지 않았다**
  (박스 구조체를 임베드로 나눈 것이 산출물에 영향이 없다는 증거다).
  주의: **호스트에서 `go test ./...`는 픽스처 골든 2건이 빨갛다** —
  `02_text_subset/slide2·3`, `03_layout/slide2·4·6·7`의 `a:ext`가 다르다. 이 항목 전
  (HEAD의 별도 워크트리)에서도 같은 실패가 나므로 호스트 폰트 차이이고 이 커밋과 무관하다.
  게이트는 컨테이너다 (`make test`).

- **남긴 것**:
  - `ShapeStyle`·`TextStyle.Anchor`·테마의 radius/shadow 표가 **아직 어느 렌더러도 읽지
    않는다** — 항목 04(웹 CSS)·05(OOXML)가 소비한다. 웹은 반경·그림자 토큰을
    `--balpyo-radius-*`/`--balpyo-shadow-*` 변수로 낼지 정해야 한다 (색·공간은 변수로 나가고
    있다). 지금 내지 않는 이유는 소비자가 없는 선언이 골든을 키우기 때문이다.
  - **캐스케이드가 없다** (항목 03): 지금은 `<shape type="rect"/>`의 fill/border가 nil로
    남아 PRD §8.3의 컴포넌트 기본값(`fill: surface`, `border: 1 solid muted`, `radius: md`)이
    적용되지 않는다. 도형 variant 표 이름(`card-header`·`card-body`)은 `_04_shape/007`이
    가정하고 있다.
  - **문서가 낡았다** (항목 07이 고친다): `docs/reference/soml/text.md`가 색 토큰 진단 문구를
    그대로 싣고 있는데, 토큰 목록에 `surface`가 늘고 문구에 속성 이름 접두가 붙었다
    (`fill의 색 토큰 "brand"가 테마에 없다`).
  - **열린 질문 3건**:
    (a) **`anchor`를 도형이 어떻게 읽는가.** 자식이 여럿일 때(`<h3>` + `<p>`) 도형의
    `bodyPr@anchor`는 하나뿐인데 자식마다 다른 앵커를 쓸 수 있다. 값 층은 자식마다 확정
    값을 남기므로 "첫 자식의 앵커를 쓴다/충돌은 경고다"의 선택은 항목 04·05의 것이다.
    (b) **`opacity`가 도형 전체인가 채움만인가.** CSS `opacity`는 자식 텍스트까지 흐리게
    하지만 OOXML은 알파를 `solidFill` 안에 넣는 것이 자연스럽다 (도형 전체 투명도는
    `a:alphaModFix` 효과다). `_04_shape/004`의 `opacity="0.4"` 도형이 그 차이를 그림으로
    보일 것이다 — 두 타깃이 갈라지면 ADR 감이다.
    (c) **`rotate`와 지오메트리 패스.** `getBoundingClientRect`는 회전된 박스의 외접
    사각형을 재므로(`core/style.go`의 allowlist 주석이 같은 이유로 `rotate`를 뺐다)
    회전 도형의 rect가 실제 도형보다 크다. 웹에서 `transform: rotate()`를 쓰면 PPTX
    `xfrm@rot`과 좌표가 어긋날 수 있고, 그 보정은 항목 04·05가 재 보며 정해야 한다.

## Iteration 4 · 03-core-cascade — 도형 캐스케이드: 컴포넌트 기본값 + variant 표
<!-- item: 04/03-core-cascade -->

- **한 것**:
  - `core/cascade.go`: `shapePatch`(외관 7필드, 전부 포인터) + `mergeShape`를 `textPatch`/
    `mergeText` 옆에 뒀다. 파일 머리말에 "채널이 둘이고 기계는 복제한다"를 적었다.
  - `core/theme.go`: 도형 컴포넌트 기본값(`shapeComponentDefault`)과 도형 variant 표
    (`shapeVariants`·`shapeVariantPatch`·`shapeVariantNames`), 표를 한 줄로 적는 도구 넷
    (`colorFill`·`solidBorder`·`radiusToken`·`shadowToken`)을 텍스트 표 아래에 붙였다.
  - `core/shape.go`: `resolveShape`가 **병합 세 줄**이 되었다 (인라인 패치 → 컴포넌트
    기본값 → variant → 인라인). 속성 훑기는 `shapeAttrs`로 떼고, 테마 단의 radius를
    프리셋에 맞춰 다듬는 `themeShapePatch`, 도형 variant 조회 `resolver.shapeVariant`를
    더했다.
  - `core/resolve.go`: `variant` 속성 획득(`variantAttr`)과 모르는 variant 경고
    (`warnUnknownVariant`)를 텍스트·도형 공용으로 뽑았다. 텍스트 경로의 동작·문구는 그대로다.
  - `core/cascade_test.go`: 도형 캐스케이드 테스트 8개(우선순위 표 + 단별 출처 표, 부분성·
    0값 구별·패치 필드 누락 감시, 컴포넌트 기본값의 완전성, radius의 프리셋 규칙, variant 표의
    구별성·순서, 모르는 variant 경고·폴백·`--strict`, 두 어휘의 분리, 자식 텍스트 불변).
  - `core/shape_test.go`: 표의 `want`를 **컴포넌트 기본값 위의 델타**로 바꿨고(`wantShapeStyle`
    — 얹는 데 운영 코드의 `mergeShape`를 쓴다), `variant` 행을 더해 커버리지 예외(`pending`)를
    지웠다. 픽스처 경고 기대치는 1 → 2 (008의 도형 variant가 이제 경고한다).

- **설계 결정과 근거**:
  - **컴포넌트 기본값이 도형 캐스케이드의 가장 아래 단**이다 (`textRoleDefaults`의 자리를
    대신한다) — 도형에는 역할이 없다. 그래서 `<shape type="rect"/>`만 써도 fill·border가
    확정된다: 값이 비어 남으면 채우는 판단이 렌더러로 내려가고, 두 타깃이 서로 다른 값을
    채우는 순간 웹과 PPTX가 갈라진다.
  - **radius 충돌은 "검사를 인라인 단에만"으로 풀었다.** 기본값에 `radius: md`를 두되
    `rounded-rect`가 아닌 프리셋에서는 테마 단의 radius를 조용히 버린다 (`themeShapePatch`).
    기준은 **저자가 고칠 수 있는가**다: 인라인 radius는 지울 수 있으니 진단이 고칠 방법을
    말하지만, 테마 기본값에 같은 진단을 내면 `<shape type="rect"/>`가 무조건 에러가 된다.
    기본값에서 radius를 빼는 길도 있었지만 그러면 `type="rounded-rect"`만 쓴 도형에서 두
    타깃의 **암묵 기본값 차이**가 되살아난다 (CSS는 0, OOXML `roundRect`는 조정값 기본
    16.67%). 이 자리가 "테마 값과 저자 값에 같은 규칙을 걸 수 있는가"의 첫 사례이고,
    Iteration 10에서는 진단의 수신자가 또 달라져(테마 제작자) `theme lint`가 받는 편이
    자연스럽다는 관측을 주석에 남겼다.
  - **도형 variant는 자식 텍스트의 외관을 담지 않는다** (PRD §8.3의 `card-header`가 담는
    `font`/`size`/`weight`를 버렸다). 담으면 캐스케이드가 트리를 타고 내려가는 **상속**이
    되고, 상속은 한 요소 안에서 끝나는 지금의 병합과 다른 두 번째 메커니즘이다 — 그것이
    "우선순위는 병합 함수 한 곳에 있다"는 계약의 첫 예외가 된다. `TestResolveShapeVariant
    DoesNotTouchChildText`가 6종 전수로 그 상속이 몰래 들어오는 것을 막는다.
    **PRD §8.3과 어긋나므로 항목 07이 PRD를 함께 고쳐야 한다.**
  - **variant 표 값** (컴포넌트 기본값 `surface`/`1 solid muted`/`md`와 구별되게):
    `card`(fill bg, radius lg) · `card-header`(border none, radius sm — fill은 정하지 않는
    부분 패치) · `card-body`(fill bg) · `card-backdrop`(fill none, border 3 solid accent1,
    radius lg) · `stat-card`(border none, shadow soft, radius lg) · `badge`(border 1 solid
    accent1, radius lg). 순서 있는 슬라이스이고 이름은 PRD §13-6의 권장 어휘다.
    - `card-backdrop`의 그라디언트 테두리는 **단색 accent1로 대체**했다 — 색 함수는
      Iteration 8이고 그 항목이 이 자리를 되돌린다 (주석에 적었다).
    - **채움에 accent1/muted를 쓰는 variant를 만들지 않았다**: 어두운 채움은 그 위의 글자색이
      `bg`여야 하고, 그 요구는 "variant가 자식 타이포그래피를 담지 않는다"는 결정과 부딪힌다
      (저자가 색을 손으로 맞춰야 하는 번들은 번들이 아니다). `badge`를 선만 강조색으로 둔 것도
      같은 이유다.
    - **어휘 6종을 픽스처가 다 쓰지 않는다** — 토큰 표의 기준("픽스처가 실제로 쓴다")과
      의도적으로 다르게 골랐다. 토큰은 값이 두 타깃에서 검증되어야 하지만 variant는 **이미
      검증된 토큰의 조합**일 뿐 새 값을 만들지 않으므로 산출물 위험이 없고, 권장 어휘를 한 벌로
      갖춰 두면 다른 테마로 만든 덱이 폴백 경고 없이 빌드된다 (PRD §13-6이 지키려는 이식성).
  - **경고는 요소 종류에 맞는 목록을 나열한다** (`warnUnknownVariant`가 목록을 인자로 받는다).
    문구 템플릿은 한 곳뿐이고 **기존 텍스트 경고와 바이트가 같다** — "도형 variant"처럼 종류를
    문구에 넣으면 `02_text_subset/golden/diagnostics.txt`가 이 항목의 산출과 무관한 이유로
    움직이고, 그것은 "도형 없는 덱은 아무것도 달라지지 않는다"는 이 항목의 계약과 어긋난다.
    나열된 목록이 이미 어느 어휘인지 말한다.
  - **`Type`·`ID`·`Box`는 패치에 없다**: `Type`은 어느 프리셋을 그리는가라 variant가 바꾸면
    저자의 `type`이 테마에 따라 다른 그림이 되고(textPatch에 `Role`이 없는 것과 같은 자리),
    `ID`는 참조 이름이라 테마가 붙일 수 없고, `Box`는 배치 채널이다(테마가 배치를 담지 않는다 —
    Iteration 3이 닫은 질문). 그래서 `shapeAttrs`는 그 둘을 style에 직접 채운다.
  - **두 패치 타입을 제네릭으로 합치지 않았다**: 필드가 하나도 겹치지 않아 뽑아낼 공통이
    "nil이 아니면 대입한다" 한 줄뿐이고, 그 한 줄을 위해 접근자나 리플렉션을 끼우면 병합이
    **한눈에 보이는 대입 목록**이라는 성질을 잃는다.
  - **`mergeShape`는 포인터를 대입하지 않고 값을 복사한다.** 지금은 테마 표가 호출마다 새로
    만들어져 차이가 없지만, 표가 파일에서 한 번 읽혀 캐시되는 날(Iteration 10) 여러 도형이 같은
    `Fill`을 가리키게 되고 한 도형을 고치는 코드가 테마를 고친다.
  - **테마 표가 `soml.Theme`를 인자로 받는다** (텍스트 표는 받지 않는다): 도형의 확정 값은 토큰
    이름과 **해석된 수치를 함께** 담는다 (`soml.Radius`·`ShadowToken` — PPTX가 `prstGeom`
    조정값·`outerShdw` 파라미터로 굽는다). 표에 없는 토큰이면 그 필드를 비운다 (모르는 variant를
    경고로 두는 것과 같은 태도 — 테마 교체가 덱을 멈추지 않는다).
  - **`shape_test.go`의 표를 델타로 바꾼 이유**: 기본값 세 필드를 30줄에 반복하면 각 줄이 무엇을
    시험하는지 묻히고 기본값을 고칠 때 표 전체를 고쳐야 한다. 얹는 데 운영 코드의 `mergeShape`를
    쓰므로 테스트가 병합 규칙을 다시 쓰지 않는다.

- **검증**: `gofmt -l .`(무출력), `go vet ./...`, **`make test`(컨테이너) 초록**.
  **골든은 한 파일도 움직이지 않았다** (`git status`에 `*/golden/*`이 없다) — 아직 어느
  렌더러도 `ShapeStyle`을 읽지 않고, 텍스트 경고 문구를 그대로 둔 결과다. 이것이 항목 04·05의
  "도형 없는 덱 골든 불변" 계약의 앞단이다. 호스트 `go test ./...`는 여전히 픽스처 골든이
  빨갛지만 **차이는 전부 `a:ext` 14줄**(텍스트 폭)이고 항목 02 노트에 적힌 호스트 폰트 차이와
  같다 — 게이트는 컨테이너다.

- **남긴 것**:
  - 확정된 도형 외관이 이제 **완전하다** — 항목 04(웹 CSS)·05(OOXML)는 nil 자리를 만나지
    않는다. 단 `Radius`는 `rounded-rect`가 아니면 nil이고(각진 모서리라는 뜻), `Shadow`·
    `Opacity`·`Rotate`·`Flip`의 nil은 "선언 없음"이다 (두 타깃의 암묵 기본값이 같아 렌더러가
    채울 판단이 없다).
  - `_04_shape/007`은 이제 경고 없이 해석되고 `008`은 경고 **두 개**를 낸다 (도형 + 텍스트) —
    항목 06이 `diagnostics.txt` 골든에 그 둘을 담는다.
  - **항목 07이 고칠 문서**: (a) PRD §8.3의 `card-header`가 타이포그래피를 담는 부분 —
    위의 상속 결정과 어긋난다. (b) `docs/reference/appearance.md`는 `variant`가 `<text>`와
    블록 태그의 것이라고만 적고 있다 (이제 `<shape>`도 받고, **어휘가 두 벌**이다). (c) 항목
    02가 남긴 색 토큰 진단 문구 갱신도 그대로 남아 있다.
  - **열린 질문 2건**:
    (a) **variant가 `type`을 정할 수 있어야 하는가.** 지금은 못 정한다(위 근거). 그래서
    `variant="badge"`를 쓰는 저자도 `type`을 매번 적어야 하고, `card-header`가 `rounded-rect`가
    아니면 그 variant의 radius가 조용히 버려진다. 테마가 "이 variant는 rounded-rect를
    가정한다"고 말할 방법이 없다 — Iteration 10의 테마 포맷이 다시 만날 질문이다.
    (b) **컴포넌트 기본 채움이 모든 도형에 적용되는 것이 맞는가.** `_04_shape/004`의
    `rotate="45"` 도형처럼 장식용 도형도 surface 배경을 받는다. PRD §8.3의 `default`가 그
    뜻이므로 그대로 따랐지만, 그림으로 봐야 판단이 되는 자리다 (항목 04·05의 갤러리).

## Iteration 4 · 04-web-shape — 웹 도형: SVG 프리셋 기하 + 외관 CSS
<!-- item: 04/04-web-shape -->

- **한 것**:
  - `render/web/shape.go` 신규 — 도형 렌더 전부. `writeShape`(박스 div + 그림 svg + 자식
    텍스트), **12종 기하 유도표 `shapeGeometries()`**(OOXML 프리셋 이름 + 정규 공간 경로 +
    항목 05용 조정값 공식), `shapeBoxCSS`(pad·크기·비율·회전 각도·앵커), `shapeInkCSS`
    (fill·border·shadow·opacity·flip), `dashArray`·`flipValue`·`anchorValue`,
    `shadowVarsCSS`(그림자 토큰 → CSS 변수), `shapeCSS`(엔진 CSS), 그리고 내보낸 계약 셋
    `ShapePreset`(type → `prstGeom@prst`)·`ShapeAnchor`(도형이 쓸 앵커).
  - `render/web/layout.go` — 크기·비율 매핑을 `boxSizeCSS(soml.Box)`로 떼어 컨테이너와 도형이
    **같은 함수**를 쓴다. `pad`는 컨테이너에서 트랙 속성보다 앞이라 호출자에 남겼다 (선언
    순서를 옮기면 03_layout 골든이 이 항목과 무관한 이유로 움직인다 — 실제로 불변이다).
  - `render/web/content.go` — `writeNode`의 `KindShape` 갈래(지금까지 의도된 가드였던 자리),
    `features.shapes`와 `features.spaceTokens()`(공간 토큰 참조 조건이 레이아웃 **또는** 도형:
    `pad`가 두 요소의 공유 속성이다), `contentCSS`에 `shapeCSS`.
  - `render/web/style.go` — `themeCSS(theme, features)`로 바꾸고 그림자 토큰 변수를 도형이
    있는 덱에만 낸다.
  - `render/web/runtime.go` — **`MeasureView`("measure")와 `MeasureViewExpr()`**. 뷰가 셋이
    되었다 (문서·발표자·측정).
  - `render/web/web.go` — 패키지 주석의 **측정 대상 계약**에 `<shape>`와 도형 수 등식의 새 항
    (`− 도형 안 텍스트`), 그리고 "뷰는 셋이다" 절.
  - `render/geom/measure.go` — 측정 스크립트가 측정 뷰를 켜고 재고 `finally`에서 되돌린다.
    표현식은 `web`이 소유한 문자열을 채운다 (선택자 계약과 같은 태도).
  - `render/geom/geom.go` — `collect`의 측정 대상 주석에 도형과 도형 안 텍스트의 비대칭.
  - 테스트: `render/web/shape_test.go` 신규(PRD §9.4 표 전수 + 어휘 커버리지, 12종 기하
    구별·표 순서·프리셋 이름, 늘린 공간의 `vector-effect`, 토큰 참조/반경 수치, 회전의
    측정 뷰 무효화, opacity·flip이 텍스트를 건드리지 않음, 앵커 첫 자식 규칙, 박스 매핑
    공유 대조, 측정 대상 개수, 도형 없는 덱 불변, 에러 13건, 결정성 — 38 서브테스트),
    `render/geom/geom_test.go`(도형·도형 안 텍스트 Refs ↔ HTML 자리 맞대기, 측정 스크립트의
    뷰 전환 순서), `render/geom/measure_browser_test.go` 실측 4건(회전 도형의 잰 rect가
    회전 전 박스 + 되돌린 뷰에서는 1.41배, pad·anchor 3종, 그리드 도형 카드, 그림 외관이
    브라우저에서 실제로 값이 되는지), `render/web/runtime_test.go`(뷰 셋의 값 분리).

- **설계 결정과 근거**:
  - **12종 전부 SVG 한 벌**(항목의 선택지 (b)). `clip-path`는 테두리·그림자를 함께 자르고
    컴포넌트 기본값이 `border: 1 solid muted`이므로 테두리는 예외가 아니라 기본이다. 셋만
    CSS로 두는 (c)는 그림자를 `box-shadow`와 SVG 필터로 두 벌 구현하게 되고, 두 벌이
    갈라지는 것은 "도형 종류에 따라 외관이 다르다"로만 드러난다. SVG는 어차피 들어온다
    (`border=… gradient(…)`가 SVG 오버레이로 못 박혀 있다 — PRD §9.4, Iteration 8).
    외관 선언은 **svg 하나에** 걸리고 기하만 두 갈래다: 백분율 프리미티브로 정확히 되는
    셋(`<rect>`·`<rect rx>`·`<ellipse>`)은 px 공간, 나머지 아홉은 정규 경로다.
  - **기하의 출처는 ECMA-376 `presetShapeDefinitions.xml`**이다 (apache/poi가 배포하는 사본을
    읽었다). 각 프리셋을 **기본 조정값으로** 평가하고 유도 근거(`gd` 이름과 공식)를 표 행의
    주석에 남겼다. `cloud`는 43200 경로 공간의 `arcTo` 11개를 SVG `A`로 변환했고(중심 =
    현재점 − (wR·cos stAng, hR·sin stAng)), `star5`의 좌표에 0.0001이 남은 것은 프리셋
    계수(hf·vf)가 정확한 별 비율의 근사값이기 때문이다 — 그 잔차가 수치의 출처를 증명한다.
    `wedgeRectCallout`은 16점 중 겹치는 9점이 사라져 7점이 되고, 꼬리 끝이 박스 아래로
    12.5% 나간다.
  - **정규 100×100 공간을 늘린다** (`preserveAspectRatio="none"` + `vector-effect:
    non-scaling-stroke`). 프리셋 공식의 `ss`는 `min(w,h)`인데 **웹은 도형의 픽셀 크기를 모른다**
    (브라우저가 배치를 계산하고 우리가 그 결과를 잰다). CSS/SVG에는 자기 박스의 `min(w,h)`를
    경로 데이터에 넣을 수단이 없다 (`cqmin`은 `clip-path`에서만 쓸 수 있고 그 길은 테두리를
    자른다). 그래서 프리셋을 **정사각형에서 평가**해 정규 공간에 두고 박스에 늘린다.
    - 대가: `ss` 의존이 있는 다섯 프리셋(rounded-rect·cylinder·chevron·arrow-right·hexagon·
      parallelogram)은 정사각형이 아닌 박스에서 PowerPoint의 기본 모양과 비율이 다르다.
      220×110 chevron의 노치가 `ss/2`(55)가 아니라 `w/2`(110)가 된다 — 그림으로 보면 훨씬
      깊은 화살표다.
    - 그래서 **PPTX가 맞춘다**: 표의 각 행에 측정된 박스로 계산할 `avLst` 조정값 공식을 적어
      두었다 (`chevron: adj = 50000·w/ss`, `hexagon`·`parallelogram`: `25000·w/ss`,
      `arrow-right`: `adj2 = 50000·w/ss`, `cylinder`: `adj = 25000·h/ss`, `rounded-rect`:
      `adj = 100000·radius/ss`). 웹이 기준 타깃이므로(PRD §9.1) 맞추는 쪽이 PPTX이고,
      조정값을 실은 프리셋도 PowerPoint에서 편집 가능한 진짜 프리셋 도형이다
      (`radius` → `prstGeom` 조정값이 이미 같은 수단이다).
    - 버린 길: (1) 런타임 JS가 잰 박스로 경로를 계산하는 길 — 정확하지만 도형이 스크립트
      없이는 그려지지 않고, 골든이 도형의 모양을 보지 못하며, 리사이즈마다 다시 계산하는
      **살아 있는 기하 엔진**이 브라우저에 생긴다. (2) `w`/`h`가 px로 확정된 도형만 정확히
      그리는 길 — 같은 프리셋이 두 방식으로 그려지고, 그 갈라짐은 같은 덱 안에서만 드러난다.
  - **테두리는 경로 중앙**이다 (`stroke`) — OOXML `a:ln`과 같은 규약이므로 굵기의 절반이 잰
    박스 밖으로 번지고 **두 타깃이 같은 자리에 번진다**. 항목의 8번이 남긴 질문("웹에서
    보정할지, PPTX에서 rect를 줄일지")은 이 결정으로 닫힌다: 보정할 수치가 없다. CSS
    `border`(박스 안쪽)를 골랐으면 그 수치가 항목 05로 넘어갔을 것이다. svg는 그래서
    `overflow: visible`이다 (callout 꼬리도 같은 이유).
  - **박스와 그림의 경계**: 박스(div)는 `pad`·크기·비율·**회전**·앵커, 그림(svg)은 채움·
    테두리·**그림자**·**불투명도**·**대칭**. 각 자리의 근거가 OOXML이다 —
    - `rotate`는 박스: `xfrm@rot`이 텍스트까지 함께 돌린다. 중심 회전이므로
      `transform-origin` 기본값(50% 50%)이 그대로 맞다.
    - `flip`은 그림: PowerPoint의 `flipH`/`flipV`는 기하를 뒤집지만 **글자는 읽히는 방향으로
      남는다**. 박스에 걸면 CSS가 글자까지 거울로 만든다.
    - `opacity`는 그림: OOXML은 알파를 `solidFill`·`ln` 안에 싣는다 (도형 전체 투명도는
      `alphaModFix` 효과다) — NOTES 04/02의 열린 질문 (b)를 이렇게 닫았다. 항목 05는 알파를
      색 안에 넣어야 두 타깃이 같은 그림이 된다.
    - `shadow`는 그림이고 **`box-shadow`가 아니라 `filter: drop-shadow`**다: box-shadow는
      박스의 사각형을 따르므로 열두 종 중 아홉이 엉뚱한 사각 그림자를 얻는다. drop-shadow는
      실루엣의 알파를 따라가 `outerShdw`와 같은 그림이 된다. **PRD §9.4 표의 `box-shadow`
      칸이 이 결정으로 갱신되어야 한다 (항목 07).**
  - **회전 선언은 엔진 CSS, 각도는 인라인 커스텀 프로퍼티**(`.balpyo-shape-rotate`
    + `--balpyo-shape-rotate`). 측정 뷰가 `transform: none`으로 덮어야 하는데 인라인 style은
    층 밖에서 이기므로, 선언이 요소 쪽에 있으면 `!important` 없이는 무효화할 수 없다.
    클래스를 회전 있는 도형에만 붙이는 이유: 모든 도형에 `transform: rotate(0deg)`를 걸면
    회전 없는 도형까지 스택 문맥과 래스터화 경로가 바뀌고, 그 변화는 픽셀 비교에서만 보인다.
  - **측정 뷰는 세 번째 뷰**다 (ADR 0006의 수단 재사용). 측정 스크립트가 켜고 `finally`에서
    되돌리는 이유: 같은 세션이 재고 찍으므로(internal/testkit) 측정 상태가 남으면 스크린샷이
    회전 없는 그림을 찍는다. `Session.Open`에 넣지 않은 것도 같은 이유다. `offsetWidth`
    체인으로 재는 길은 두 번째 측정 경로를 만들므로 버렸다.
  - **앵커는 첫 자식이 정한다** (`ShapeAnchor`, 04/02의 열린 질문 (a)). 앵커는 자식마다 확정
    값으로 있지만 박스의 성질이다 (flex `justify-content` 하나, `bodyPr@anchor` 하나).
    뒤 자식의 다른 앵커는 **조용히 버려진다** — 경고는 값 층의 몫이고 렌더러에는 진단 통로가
    없다. 함수를 내보내 항목 05가 같은 규칙을 쓰게 했다 (규칙이 두 벌이면 두 타깃의 앵커가
    갈라진다).
  - **그림자 토큰은 CSS 변수, 반경 토큰은 확정 수치**. 갈리는 기준은 참조하는 자리의 성질이다:
    그림자는 CSS 프로퍼티 값(`filter`)이라 `var()`가 통하지만, 반경은 SVG 기하 속성(`rx`)이라
    커스텀 프로퍼티를 이식성 있게 참조할 수 없다. 반경을 수치로 내도 웹만 뒤처지지 않는
    이유는 **OOXML에도 반경 테마 개념이 없기 때문**이다 (조정값은 도형마다 구워진다) — 두
    타깃이 같은 방식으로 값을 굽는다. 그림자 색은 `color-mix(in srgb, var(--balpyo-color-fg)
    15%, transparent)`로 **토큰 참조를 유지**한다 (PRD §9.4의 색 변형 행이 `color-mix()`를
    가리킨다). rgba로 눕히면 그 도형만 옛 색으로 남는다. 극좌표(dist·dir) → 직교 변환은
    `shadowVarsCSS` 한 곳이고, 잔차 반올림(소수 넷째 자리)도 거기 하나다.
  - **파선 수치는 OOXML `prstDash` 비율**이다 (`dashed` 4:3, `dotted` 1:3, 단위는 선 굵기).
    CSS `border-style: dashed`의 패턴은 UA가 정하므로 이름을 옮길 수 없고, 수치를 여기서
    정하는 것이 두 타깃이 같은 간격을 갖는 유일한 길이다.
  - **그림 svg는 `aria-hidden`이고 자식 텍스트는 `position: relative`**다. 절대 배치된 svg가
    흐름 콘텐츠보다 위에 칠해지므로 자식을 올려야 글자가 그림 위에 온다. `z-index: -1`로
    그림을 내리는 길은 스택 문맥에 따라 슬라이드 바탕 아래로 사라진다.
  - **`radius`가 프리셋과 어긋나면 에러**다 (rounded-rect에 없거나, 다른 프리셋에 있으면).
    비어 있으면 CSS 기본(0)과 OOXML `roundRect` 기본(16.67%)이 달라 두 타깃이 다른 모서리를
    그리고, 실을 자리가 없는 프리셋에 남아 있으면 웹만 둥글어진다. core가 이미 막지만 렌더러가
    마지막 관문이다 (해석되지 않은 값 전반을 에러로 두는 기존 태도).
  - **TASKS.md 체크박스는 건드리지 않았다** — iteration 4의 "테스트"·"구현" 항목이 웹과 PPTX에
    걸쳐 있고 PPTX는 항목 05다 (부분적으로 걸친 체크박스는 손대지 않는다는 규약).

- **검증**: `gofmt -l .`(무출력), `go vet ./...`, **`make test`(컨테이너) 초록**
  (`internal/testkit`을 `-count=1`로 다시 돌려 캐시 없이 확인했다). **골든은 한 파일도
  움직이지 않았다** (`git status`에 `*/golden/*`이 없다) — 도형 CSS·그림자 변수가 도형 있는
  덱에만 나가고, 박스 크기 매핑을 함수로 뺀 것이 선언 순서를 바꾸지 않았다는 증거다.
  브라우저 실측으로 확인한 사실: 회전 45도 도형의 잰 rect가 **120×120**(회전된 bbox는 169.7),
  측정 뒤 뷰 속성이 지워져 같은 요소가 다시 169.7로 읽힘, `pad`가 잰 박스를 키우지 않고
  텍스트를 24px 안으로 밀어냄, 앵커 3종이 서로 다른 y를 만들고 center의 위아래 여백이 같음,
  `<grid cols="4">` 안 도형 카드가 같은 y·등간격 x·트랙 폭, 그림의 computed style이 실제 값이
  됨(`fill: rgb(15,101,249)`, `filter: drop-shadow(... 0px 4px 16px ...)`, `opacity: 0.4`).
  `_04_shape` 픽스처 8장을 실제로 렌더해 눈으로 봤다 — 12종이 모두 프리셋 모양으로 보이고
  (chevron만 위에 적은 깊은 노치), 회전·대칭·투명도·그림자·앵커·카드 그리드가 그림에서 맞다.
  호스트 `go test ./...`는 여전히 픽스처 골든 7건이 빨갛지만 **차이는 전부 `a:ext`**(텍스트 폭)이고
  항목 02·03 노트의 호스트 폰트 차이와 같다 — 게이트는 컨테이너다.

- **남긴 것**:
  - **항목 05(PPTX)가 이어받을 표면**: `web.ShapePreset`(12종 → `prstGeom@prst`),
    `web.ShapeAnchor`(`bodyPr@anchor`의 같은 규칙), 그리고 `shapeGeometries()` 각 행의
    **`Adjust` 공식** — 늘린 웹 그림과 같은 모양을 내려면 측정된 `w`/`h`/`ss`로 `avLst`를
    계산해야 한다. 잰 rect는 **회전 전 박스**이므로 `xfrm@rot`을 그대로 얹으면 된다(보정 없음),
    테두리는 양 타깃 모두 외곽선 중앙이라 rect를 줄이지 않는다, 알파는 `solidFill`·`ln` 안에,
    그림자는 도형 실루엣에(텍스트 아님), `flipH`/`flipV`는 텍스트를 뒤집지 않는다.
    도형 **안** 텍스트의 잰 rect는 소비하지 않는다 (`txBody` 문단이 된다).
  - **알려진 갈라짐 3건** (항목 06의 픽셀 비교가 수치로 볼 것):
    (a) `cylinder`의 뚜껑에 OOXML은 `fill="lighten"`이 걸려 있는데 CSS에 대응이 없어(색 변형은
    Iteration 8) 웹은 같은 채움에 선만 얹는다. (b) `cylinder`·`cloud`의 프리셋 세부는
    `fill="none"` 경로(선)이므로 `border="none"` 도형에서는 사라진다. (c) `chevron` 등 `ss`
    의존 프리셋의 비율 (위의 결정 — PPTX가 조정값으로 맞추면 두 타깃은 같아지고, PowerPoint의
    기본 모양과는 다르다).
  - **항목 07이 고칠 문서**: PRD §9.4의 `shadow` 행(`box-shadow` → `filter: drop-shadow`),
    §7.5의 `<shape>` 웹 렌더 칸("div/SVG" → 박스 div + 그림 svg), 그리고 ADR 감이 셋 —
    "도형 기하는 SVG 한 벌이고 정규 공간을 늘린다(조정값은 PPTX가 맞춘다)", "측정 뷰에서
    변형을 끈다", "박스/그림의 외관 분담(회전만 박스)".
  - **열린 질문 3건**:
    (a) **정규 공간의 늘림을 언제 다시 볼 것인가.** 항목 06의 픽셀 비교가 `ss` 의존 프리셋의
    차이를 수치로 보여줄 것이고, 저자가 조정값을 쓸 수 있게 되는 날(어휘 확장) 이 결정이
    다시 열린다. 기하 표가 한 곳이므로 되돌리기는 표 하나를 고치는 일이다.
    (b) **도형 안 리스트·중첩 도형이 없다** (파서가 닫아 두었다). 카드 안 불릿은 실물 덱의
    흔한 단위이므로 어휘를 열 날이 온다 — 그때 도형의 flex 컨테이너 규칙(`.balpyo-shape >
    .balpyo-text`)이 리스트까지 덮어야 한다.
    (c) **`<image>`의 `fit`/`frame`(Iteration 5)이 같은 구조를 재사용할 수 있는가.** 지금
    구조에서 `<image>`는 그림 svg 자리에 `<img>`가 오고 `frame`이 도형의 기하가 되는 모양이
    자연스럽지만, `fit="cover"`는 클리핑이 필요하고 클리핑은 이 항목이 피한 `clip-path`의
    문제를 다시 만난다 (SVG `clipPath` + `<image preserveAspectRatio>`가 그 자리일 것이다).

## Iteration 4 · 05-pptx-shape — PPTX: 프리셋 도형 12종 + 외관 OOXML
<!-- item: 04/05-pptx-shape -->

- **한 것**:
  - `render/pptx/shape.go` — `writeShapeNode`의 **네 번째 분기**(`KindShape` — 도형 하나 +
    자식 흡수), `writeShapeSp`, 조립 명세 `spSpec`·`spAdjust`, `textBoxSp`(텍스트 박스의
    명세)·`shapeSp`(외관 → 명세)·`shapeAdjusts`, 일반화된 `writeSp`(+`xfrmAttrs`),
    `writeFill`·`writeLine`·`writeEffect`·`shadowAlpha`·`prstDashValue`,
    도형 본문 `shapeBodyXML`(+`bodyAnchorValue`·`bodyInsets`), 문단 조립을 떼어낸
    `writeTextParagraph`(블록과 도형 안 텍스트가 **같은 함수**를 지난다),
    색 요소를 알파와 함께 낼 수 있게 나눈 `colorElem`·`writeColor`.
  - `render/pptx/units.go` — `Rotation`(60000분의 1도 + 한 바퀴 정규화)·`Alpha`(100000분율)·
    `AdjustValue`(조정값 평가 + 상한 자르기), 상수 `UnitsPerDegree`·`FullRotation`·`PercentScale`.
  - `render/web/shape.go` — 기하 표의 `Adjust`를 **기계가 읽는 명세**로 바꿨다
    (`ShapeAdjust{Name, Base, Axis, Max}` + `ShapeAdjustAxis` 3종) 그리고 `ShapeAdjusts`를
    내보냈다. `dashArray`의 프리셋 이름 오기 정정(1:3은 `sysDot`이 아니라 `dot`이다).
  - 테스트: `render/pptx/preset_test.go` 신규(12종 전수 + 프리셋 이름 문자열·중복 없음,
    조정값 6종·조정값 없는 6종, 토큰 색 → `schemeClr` + 리터럴 → `srgbClr` 양방향,
    `noFill` 쌍·테두리 3조각·파선 계열, 그림자, 불투명도가 색에 실리고 **글자에는 실리지
    않음**, 회전 정규화 4건·대칭 3종·변형 없음, 도형 안 텍스트(도형 1개·문단 여럿·인셋·
    여유 미적용)·앵커 4건·빈 도형, **네 분기 대비 한 슬라이드**(도형 수·이름 있음/없음·id
    연속성·문서 순서), 결정성, 관문 9건 — 서브테스트 47개),
    `render/pptx/units_test.go`(회전 10건·알파 6건·조정값 경계 12건 + 모르는 축),
    `render/web/shape_test.go`(조정값 명세 전수 + 반경 축 ↔ 기하의 `Rounded` 일치),
    `render/pptx/soffice_test.go`(프리셋 12종 슬라이드 + 외관·도형 안 텍스트·도형 본문의
    하이퍼링크 슬라이드).
  - `internal/testkit/text_fixture_test.go` — 하네스 이월: `shapeAligns`(정렬 목록) →
    **`shapeRoles`**(잰 rect → 도형 rect **변환** + 도형이 되지 않는 요소의 분류).
    도형 수 등식에 항이 하나 늘었다 (`− 도형 안 텍스트`), 항 이름을 `shapeExcludedKinds`가
    소유해 로그·에러가 순회의 분류에서 나온다. 자리 맞대기 단위 테스트 신규
    (`TestShapeRolesLineUpWithGeometryRefs` — 브라우저 없이 돈다).

- **설계 결정과 근거**:
  - **`writeSp`는 조립된 조각이 아니라 값 명세(`spSpec`)를 받는다.** 스키마 순서
    (CT_ShapeProperties: xfrm → 지오메트리 → 채움 → ln → effectLst)가 함수 하나에서만
    지켜져야 하고, 어긴 순서는 LibreOffice의 복구 경고로만 드러난다. 텍스트 박스는
    `fill=none`·`border=none`·`preset=rect`·`textBox=true`를 준 명세이고 — 그래서 **산출
    바이트가 그대로다** (골든 4개 불변이 리팩터링의 정확성 증거다).
  - **`txBox="1"`은 도형에 붙이지 않는다** (`<p:cNvSpPr/>`). PowerPoint가 텍스트 박스와
    도형에 다른 서식 창을 주고, 도형이 텍스트 박스로 표시되면 "모양 변경"이 사라진다.
  - **조정값 명세는 웹의 기하 표가, 평가는 PPTX가 소유한다.** 항목 04가 남긴 `Adjust` 메모
    문자열을 구조체로 올리고(`Base × Axis ÷ ss`) `ShapeAdjusts`로 내보냈다 — 공식이 PPTX에
    손으로 한 번 더 적히면 늘림(웹)과 보정(PPTX)이 갈라지고, 그 어긋남은 같은 프리셋이 두
    타깃에서 **다른 비율**로 그려지는 것으로만 드러난다. 평가가 여기인 이유는 웹이 도형의
    픽셀 크기를 모른다는 것이다 (잰 박스는 지오메트리 패스를 지나 이 렌더러에만 온다).
    상한은 `roundRect`의 50000 하나다 — 나머지 다섯의 `maxAdj`가 같은 축의 비율로 커져
    공식이 구조적으로 그 안에 있다 (각 행 주석에 근거를 적었다). 넘긴 조정값은 뷰어가
    조용히 자르므로(파일은 열린다) 자르는 자리를 우리가 갖는 편이 골든이 실제로 그려질 값을
    담는다.
  - **파선 계열은 `dash`(4:3)·`dot`(1:3)이다** — `sysDash`(3:1)·`sysDot`(1:1)이 아니다.
    `soml.BorderStyle`이 "계열 선택은 렌더 매핑의 몫(항목 05)"으로 넘긴 결정을 여기서 닫는다:
    웹이 이미 그 비율을 `stroke-dasharray`로 냈으므로(ECMA-376 프리셋 파선 표) 이름이 아니라
    **수치**가 두 타깃을 잇는다. 실선도 명시한다 (`prstDash val="solid"` — 선 없음을 명시하는
    것과 같은 근거).
  - **불투명도는 채움·선·그림자 색 안의 알파이고, 텍스트 런에는 싣지 않는다.** 이 항목의
    명세는 "같은 알파를 텍스트 런 색에도 싣기"를 권했지만 항목 04가 `opacity`를 **그림
    svg**에 걸어 자식 텍스트를 건드리지 않기로 했다 (NOTES 04/04) — 웹이 글자를 흐리게 하지
    않으므로 PPTX가 흐리게 하면 그때 두 타깃이 갈라진다. 명세의 권고를 뒤집은 근거가 그것이고,
    도형 전체 투명도(`a:alphaModFix`)를 안 쓰는 이유도 같다 (글자까지 흐려진다).
    **그림자 알파는 도형 불투명도와 곱한다**: 웹에서 `opacity`가 svg에 걸리고 그림자는 그
    svg의 `filter` 결과이므로 함께 흐려진다 (합성 순서). 곱하지 않으면 반투명 도형의 그림자만
    PPTX에서 진하다.
  - **그림자는 `effectLst > outerShdw`이고 `fmtScheme/effectStyleLst` 참조를 버렸다.** 그
    목록은 스타일이 셋뿐이고 도형이 `style/effectRef`로 하나를 고르는 구조라 도형마다 다른
    그림자를 실을 수 없다 — 토큰이 셋을 넘는 순간 "어떤 도형의 그림자가 다른 토큰의 값으로
    그려진다"로 무너진다 (`soml.ShadowToken`이 남긴 결정을 닫았다). 극좌표(dist·dir)는 **그대로**
    낸다 — 직교로 푸는 쪽이 웹인 것은 손실 있는 방향(atan2 → 정수 반올림)을 산출 경계에 두지
    않기로 한 결과다. `rotWithShape="1"`: 웹에서 회전은 박스에 걸리고 그림자는 그 안 svg의
    필터이므로 그림자가 도형과 함께 돈다.
  - **회전은 한 바퀴 안으로 정규화하고 좌표는 건드리지 않는다.** 방향 규약이 두 타깃에서
    같아 부호를 뒤집지 않고(둘 다 시계방향·y축 아래), 회전 중심이 둘 다 박스 중심이며, 잰
    rect가 **회전 전** 박스다 (측정 뷰가 변형을 끈다 — 항목 04) — 보정할 값이 하나도 없다.
    정규화하는 이유는 뷰어의 관용에 기대지 않는 것이다: `rot="-2700000"`도 열리지만 재저장하면
    PowerPoint가 정규화해서 쓰므로, 먼저 접어 두면 저자가 열어 저장한 diff가 이 속성에서 나지
    않는다 (zipEpoch·rel id를 결정적으로 두는 것과 같은 태도).
  - **테두리는 rect를 줄이지 않는다.** `a:ln`이 외곽선 중앙에 그려지는데 웹의 SVG `stroke`도
    경로 중앙이므로 굵기의 절반이 **두 타깃 모두** 잰 박스 밖으로 번진다 — 항목 04·05가 함께
    정하기로 한 "보정할지/오차로 둘지"는 보정할 **수치가 없다**로 닫힌다 (항목 04의 결정이
    이쪽 절반을 미리 결정했다). CSS `border`를 골랐으면 여기서 rect를 굵기 절반만큼 줄여야
    했다.
  - **도형 안 텍스트는 같은 도형의 `txBody` 문단**이고 자기 도형·도형 id·잰 rect를 갖지
    않는다. 대안(도형 sp + 텍스트 sp)을 버린 근거는 편집 단위다 — 도형을 끌면 글자가 따라와야
    한다 (`writeListShape`가 리스트를 도형 하나로 낸 것과 같은 근거). 그래서 `writeShapeNode`는
    도형의 자식으로 **내려가지 않는다**: 컨테이너가 사라지고 자식이 도형이 되는 것과 정확히
    반대다.
    - **안전 여유(`TextBoxRect`)를 적용하지 않는다**: 여유는 "브라우저가 잰 글자 폭보다
      LibreOffice가 넓게 잴 수 있다"를 흡수하는 장치인데 도형의 박스는 글자를 잰 결과가 아니라
      **저자가 정한 크기**다 (`w`/`h`/`ratio`). 넓히면 그림이 커지고 글자가 도형 밖으로 나간다.
    - `pad` → `bodyPr`의 인셋 EMU (웹에서 같은 값이 도형의 padding이고 `box-sizing:
      border-box`라 잰 박스를 키우지 않는다), `anchor` → `bodyPr@anchor`이며 **규칙을 웹과
      공유한다** (`web.ShapeAnchor` — 첫 자식이 정한다).
    - **`wrap`은 언제나 `square`다.** `<pre>` 자식이 있어도 바꾸지 않는다: `bodyPr`은 본문에
      하나뿐이라 `wrap="none"`은 형제 문단까지 줄바꿈하지 않게 만들고, 그러면 그려진 도형
      밖으로 글자가 나가는 일이 **pre가 아닌 자식에서** 일어난다. 대가는 도형 안 `<pre>`의 긴
      줄이 웹에서는 넘치고 PPTX에서는 접히는 것이고, 그 갈라짐은 그 자식 하나에 갇힌다.
  - **백분율 `pad`는 에러다.** CSS 백분율 padding은 **부모의 폭** 기준인데 OOXML 인셋은 절대
    EMU이므로 잰 도형 rect만으로는 웹이 쓴 값을 복원할 수 없다 — 근사해서 내면 글자 위치가
    조용히 어긋난다 (해석되지 않은 값을 에러로 두는 태도의 연장).
  - **저자의 `id`는 산출물에 나가지 않는다** (도형 이름은 지오메트리 식별자다). 그것은
    `<arrow from="db">`의 참조 대상 이름이고(Iteration 7) 웹도 쓰지 않는다 — 한쪽만 쓰기
    시작하면 두 타깃의 식별자 규칙이 갈라진다.
  - **하네스는 정렬 목록이 아니라 rect 변환 목록을 든다** (`shapeRoles`). 텍스트 박스는 여유를
    더하고 도형은 잰 값 그대로이므로 "정렬"로는 두 규칙을 담을 수 없다. 좌표 대조와 도형 수
    등식이 **같은 순회 하나**를 쓰는 이유는 두 검사가 다른 판정을 들면 한쪽만 계약을 보게
    되기 때문이다. 도형 안 텍스트는 태그 이름만으로 슬라이드 직속 텍스트와 구별되지 않아
    (같은 `<p>`다) 순회가 "도형 안인가"를 끌고 내려간다.

- **검증**: `gofmt -l .`(무출력), `go vet ./...`, **`make test DOCKER=finch`(컨테이너) 초록**.
  LibreOffice 열림 테스트가 새 슬라이드 둘(프리셋 12종 · 외관+도형 안 텍스트)을 **복구 경고
  없이** 연다 — `prstGeom` 이름과 `avLst` 문법, `outerShdw`·색 안의 `a:alpha`, `bodyPr`의
  인셋·앵커가 스키마를 벗어나지 않는다. **골든은 한 파일도 움직이지 않았다**
  (`git status -- '*/golden/*' testdata`가 비어 있다) — `writeSp` 일반화가 텍스트 박스 산출을
  바이트 단위로 보존했다는 증거다. 호스트 `go test ./...`는 픽스처 골든이 빨갛지만 차이는
  전부 `a:ext`(글자 폭·높이)이고 항목 02·03·04 노트의 호스트 폰트 차이와 같다 — 게이트는
  컨테이너다.
  `_04_shape` 픽스처 8장을 컨테이너에서 실제로 빌드해 PPTX를 PDF·PNG로 렌더해 **눈으로 봤다**:
  12종이 모두 프리셋 도형으로 그려지고(조정값이 실린 chevron의 깊은 노치·cylinder의 뚜껑·
  hexagon의 비율이 웹과 같은 모양), 45도 회전 사각형이 다이아몬드로, `flip="h"` chevron이
  거울로, `opacity=0.4`가 연한 파랑으로, `shadow=soft`가 카드 아래 그림자로, `radius` 3종이
  서로 다른 모서리로 나온다.

- **측정 — 도형 안 텍스트의 여유** (컨테이너, `_04_shape` 8장의 도형 안 텍스트 47개):
  항목 명세는 "도형은 `pad`를 뺀 안쪽이 글자보다 큰 경우가 많아 여유가 구조적으로 더 크다"고
  적었지만 **측정은 그 반대를 말한다**. 여유(박스 폭 − 가장 긴 줄)/글자 폭:

  | 자리 | 여유 |
  |---|---|
  | 짧은 라벨(12종 슬라이드의 프리셋 이름) | 24 ~ 354% |
  | 카드 본문(`006_cards`의 네 카드) | **0.9%** |
  | 카드 배너·`008`의 본문 | 1.5 ~ 2.5% |
  | `005_text_anchor`의 앵커 라벨 둘 | **0.0%** (글자가 박스를 정확히 채운다) |

  즉 **글자가 박스 폭을 채우는 문단은 여유가 텍스트 박스의 4%보다 훨씬 작고 0까지 간다.**
  줄바꿈 임계값 정책(PRD §13-1, ADR 0005)의 결함이 도형 안에서 이렇게 나타난다: 텍스트 박스는
  박스를 넓혀 스크립트 간격(최대 +2.7%)을 흡수하지만, 도형은 넓힐 수 없어(위의 결정) 흡수할
  장치가 없다 — 한글↔라틴이 섞인 도형 안 문단은 LibreOffice에서 줄이 하나 늘어날 수 있다.
  **정책 자체는 고치지 않았다** (이 항목의 범위 밖 — PRD §13-1과 ADR 0005를 함께 고쳐야 한다).
  실제로 줄이 늘어나는지는 항목 06의 픽셀 비교와 줄 수 대조가 판정한다.

- **남긴 것**:
  - **알려진 갈라짐 4건** (셋은 항목 04가 남긴 것 + 새로 하나):
    (a) `cylinder` 뚜껑의 `lighten`, (b) `cylinder`·`cloud`의 `fill="none"` 세부가
    `border="none"`에서 사라지는 것, (c) `ss` 의존 프리셋의 비율(조정값으로 맞췄으므로 두
    타깃은 같고 **PowerPoint의 기본 모양과** 다르다), 그리고 **(d) `chevron`의 텍스트 사각형이
    붕괴한다**: 220×110 chevron(`adj=100000`)의 라벨이 LibreOffice에서 한 글자씩 세로로 접힌다.
    같은 라벨을 220×220 chevron(`adj=50000`, 프리셋 기본값)에 넣으면 한 줄로 나오므로 원인은
    **조정값**이다 — 프리셋의 텍스트 사각형이 노치 좌표에서 파생되고 `adj`가 100000이면
    좌우 경계가 만나 폭이 0이 된다 (실험으로 확인했다). 웹은 도형 박스 전체에 글자를 그리므로
    이 자리에서 두 타깃이 갈라진다. 고칠 수 있는 길은 조정값을 자르는 것(→ 모양이 웹과
    갈라진다)이나 `custGeom`으로 내는 것(→ 프리셋 도형이 아니게 된다)뿐이어서, **지금은
    갈라짐으로 남긴다** — 항목 06의 픽셀 비교가 수치로 볼 것이고, 그때 `chevron` 라벨을
    도형 밖으로 옮기는 픽스처 수정도 선택지다.
  - **항목 06이 볼 것**: 위 측정의 줄 수 대조(도형 안 문단이 LibreOffice에서 늘어나는가),
    갈라짐 (a)~(d)의 수치, `003_literal_colors`가 "슬라이드 파트에 `srgbClr` 0건" 계약과
    부딪히는 자리(리터럴 색은 `srgbClr`로 **나오는 것이 정상**임을 이 항목의 단위 테스트가
    양방향으로 단정했으므로, 픽스처 수준 계약은 슬라이드 단위 예외 표가 필요하다).
  - **항목 07이 고칠 문서**: 항목 04가 남긴 셋(PRD §9.4의 `shadow` 행, §7.5의 `<shape>` 웹
    렌더 칸, ADR 감 셋)에 더해 — `soml.BorderStyle`의 주석이 `sysDash`/`sysDot`을 가리키는
    것(실제 매핑은 `dash`/`dot`), `soml.ShadowToken`의 "매핑의 실제 결정은 항목 05" 문장,
    그리고 ADR 감 둘: **"불투명도가 없는 타깃에서 무엇으로 번역하는가"**(색 안의 알파 + 글자에
    싣지 않기 + 그림자 알파 곱하기)와 **"조정값 명세는 늘린 쪽이 소유하고 평가는 재는 쪽이
    한다"**.
  - **열린 질문 3건**:
    (a) **도형 안 텍스트의 줄바꿈 여유를 어떻게 확보하는가.** 위 측정이 0%를 보였고 넓히는
    수단이 없다. 후보는 `bodyPr@lIns` 등을 조금 줄이는 것(글자 위치가 웹과 어긋난다), 폰트
    크기를 자동 축소하는 것(`normAutofit` — 시각 일치가 깨진다, PRD §13-1이 이미 버렸다),
    그리고 스크립트 간격 자체를 없애는 것(OOXML에 수단이 없다). Iteration 12(품질 게이트)가
    다시 만날 질문이다.
    (b) **도형에 `pre`를 담는 것이 뜻이 있는가.** 파서는 열어 두었지만 `wrap` 결정 때문에
    도형 안 `pre`는 두 타깃에서 다르게 접힌다. 어휘를 닫는 편이 나을 수 있고, 그 판단은
    코드 블록 카드가 실물 덱에 있는지를 봐야 한다.
    (c) **백분율 `pad`를 언제 지원할 것인가.** 부모의 잰 rect가 지오메트리에 이미 있으므로
    (컨테이너도 잰다) 부모를 순회에 끌고 내려가면 계산할 수 있다 — 하지만 그 순회는 "자식의
    좌표는 곁 표에서 그대로 온다"는 지금의 단순함을 깬다. 저자가 실제로 쓰기 전에는 열지
    않는다.
