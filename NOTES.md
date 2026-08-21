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
