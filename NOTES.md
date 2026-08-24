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

## Iteration 4 · 06-fixtures — 도형 픽스처 활성화 + 골든 + 프리셋 기하 픽셀 판정
<!-- item: 04/06-fixtures -->

- **한 것**:
  - `testdata/fixtures/04_shape/` — `_` 접두를 떼어 수집 대상으로 만들고 골든 43개 파일을
    만들었다 (슬라이드 8장 + 노트 파트 1 + `diagnostics.txt`). 슬라이드 둘을 고쳤다
    (아래 결정): `001_shape_types`는 이름표를 도형 **밖**으로, `005_text_anchor`는
    한 줄 → 두 줄, `007_variant`의 오른쪽 본문은 한 문장으로.
  - `internal/testkit/text_fixture_test.go` → **`fixture_level_test.go`**로 개명하고 표·검사
    이름에서 "text"를 뗐다 (`textFixtureNames` → `fixtureNames`,
    `TestTextFixture*` → `TestFixture*`). 검사가 재는 것은 텍스트가 아니라 픽스처 하나의
    산출물 전체다 — 이름이 남아 있으면 다음 사람이 같은 계약을 받을 픽스처를 두 번째 표에 적는다.
  - **`srgbClr` 계약 정밀화** (항목의 2번, 권한 길 (b)): `TestTextFixtureGoldenSlidesKeep
    SchemeClr`(슬라이드에 `srgbClr` 0건)를 **`TestFixtureGoldenSlideColorsKeepToken
    References`**로 바꿨다 — 슬라이드별 리터럴 색 전수 표(`fixtureLiteralColors`)와 산출
    XML을 정확히 맞댄다. 표의 열쇠는 파트 번호가 아니라 `cSld@name`의 슬라이드 id다.
  - 픽스처 수준 검증에 04_shape 편입: 골든 파트 목록 전수(`goldenPartLists`),
    슬라이드 수(`slideCounts`), 좌표 대조(`xfrm` == 측정 EMU), 도형 수 등식, 줄 수 대조 표면.
  - 새 검사 넷 (`internal/testkit/fixture_level_test.go`):
    - **`TestShapeFixturePresetGeometryPerShape`** — 12종을 격자 크롭으로 **도형마다** 픽셀
      비교한다 (`shapeCropMargin` 16px = 테두리 번짐 + callout 꼬리 12.5%, 격자 gap 24px보다
      작아 이웃이 들어오지 않는다). 이 iteration의 가장 큰 판정이고, 슬라이드 비율로는 12분의
      1로 희석되는 기하 갈라짐을 프리셋 이름과 함께 수치로 남긴다.
    - **`TestShapeFixtureInShapeTextLineCounts`** — 도형 **안쪽**에서 잉크 띠를 세어 두 엔진의
      줄 수를 맞댄다 (슬라이드 단위 판정이 도형에서 성립하지 않는 것의 대체 — 아래).
    - **`TestShapeFixtureSlideShapeCounts`** — 슬라이드별 도형 수 표. 덱 전체 등식은 한
      슬라이드에서 늘고 다른 곳에서 줄면 통과하므로, 도형 안 텍스트 흡수를 슬라이드마다 못 박는다.
    - **`TestShapeFixtureRotatedExtentIsPreRotation`** — 회전 도형의 `ext`가 회전 전 크기인가.
      좌표 대조로는 볼 수 없다 (측정이 bbox를 재기 시작하면 두 값이 함께 커지며 통과한다).
  - `internal/testkit/suite.go` — `fixtureDiffOptions["04_shape"] = 3.7%`,
    `slideDiffOptions["04_shape/shape-cards"] = 5.2%` (측정 수치와 근거는 주석에).
  - `inkBandSkipSlides`에 04_shape 8장 전부 (방법의 전제가 깨진다 — 아래).
  - `core/shape_test.go`·`soml/shape_test.go`·`core/cascade_test.go`·
    `render/geom/measure_browser_test.go`의 픽스처 경로·주석에서 `_04_shape` → `04_shape`,
    `docs/decisions/0005`의 테스트 이름 참조 갱신.

- **측정 — 프리셋 기하 12종 전수** (컨테이너, 도형별 크롭, 기본 오차 채널 32):

  | 프리셋 | 오차 | 프리셋 | 오차 |
  |---|---|---|---|
  | `ellipse` | 2.90% | `hexagon` | 3.74% |
  | `star` | 3.45% | `rect` | 4.71% |
  | `callout` | 3.63% | `cylinder` | 4.73% |
  | `chevron` | 3.66% | `diamond` | 4.90% |
  | `rounded-rect` | 4.71% | `arrow-right` | 5.74% |
  | | | `cloud` | 6.04% |
  | | | `parallelogram` | **7.95%** |

  **12종 모두 기하가 같다** — 오차의 순서가 그것을 말한다: 큰 쪽은 **긴 사선 테두리**를 가진
  프리셋(parallelogram·cloud·arrow-right)이고 작은 쪽은 매끄러운 곡선(ellipse)이다. 즉 남은
  오차는 굵기 1px 외곽선의 반 픽셀 반올림이 두 래스터라이저에서 다르게 떨어지는 것이고
  (경로 중앙 규약이라 둘레 전체가 다른 픽셀로 세어진다), 기하가 어긋났으면 오차는 이 폭이
  아니라 어긋난 **면적**으로 나타난다. 갤러리의 `01-diff.png`를 눈으로도 확인했다 — 12개
  외곽선이 전부 겹치고 마젠타는 선 위에만 있다. **오차를 키우지 않았다**: 임계값 10%는 측정
  최대(7.95%)에 앞 픽스처들과 같은 폭의 여유를 더한 값이다.

- **측정 — 도형 안 줄 수** (컨테이너, 도형 안 문단 20개):
  두 자리에서 **줄 수가 갈라진다** (나머지 18개는 같다):

  | 자리 | 여유 | 웹 | LibreOffice |
  |---|---|---|---|
  | `002` accent1 카드 본문 (`글자색도 토큰(bg)이다.`) | 8.0% | 2줄 | **3줄** |
  | `006` 첫 카드 본문 (`데이터와 모델의 편향을 측정한다.`) | 2.4% | 2줄 | **3줄** |

  **원인은 여유가 아니라 줄바꿈 규칙이다.** Chromium은 한글을 음절 사이에서 끊고
  (`데이터와 모델의 편향/을 측정한다`) LibreOffice는 낱말 경계(공백)에서만 끊는다
  (`데이터와 모델의/편향을 측정한다`) — 그래서 LibreOffice의 첫 줄이 **더 짧아지고** 밀린
  글자가 마지막 줄을 하나 더 만든다. 여유 8.0%인 자리에서 갈라진 것이 그 증거다: 폭을
  넓혀서는 닫히지 않는다. NOTES 04/05의 열린 질문 (a)("도형 안 텍스트의 줄바꿈 여유를 어떻게
  확보하는가")는 이 측정으로 **질문 자체가 바뀐다** — 확보할 여유의 문제가 아니라 규칙을
  맞추는 문제다 (아래 열린 질문).

- **설계 결정과 근거**:
  - **`srgbClr` 계약은 슬라이드별 리터럴 색 표로 정밀화했다** (길 (b)). 예외 표로 빼는 길
    (a)을 버린 이유: 리터럴이 있는 슬라이드에서 계약이 **아예 없어지고**, 같은 슬라이드의 토큰
    색(003의 `color="bg"`)이 눕는 것을 아무것도 보지 않는다. 픽스처에서 리터럴을 빼는 길 (c)은
    리터럴 경로의 회귀 테스트를 없앤다. 표에 값을 적으면 예상 밖의 리터럴은 "표에 없는 값"으로,
    토큰의 눕음은 "표에 없는 값이 토큰의 RGB와 같음"으로 함께 걸린다 — 003의 `#0F65F9`가
    accent1과 **같은 값**인 것이 그 구별을 사람 눈으로 할 수 없다는 사실을 못 박는다.
  - **잉크 띠 판정을 슬라이드 단위로 건너뛰고 도형 안쪽으로 옮겼다.** `InkBands`는 "영역에서
    가장 흔한 색이 배경"에 기대는데 칠한 도형은 그 전제를 깨뜨린다 (측정: 001의 첫 줄 도형
    넉 장이 1103×112px 띠 **하나**가 된다 — 글줄 넷이 아니다). **값 완화가 아니라 방법의
    전제가 깨진 것**이므로 03/05의 태도 그대로 표에 적었다. 다만 줄 수 계약을 버리지 않았다:
    도형 **안쪽**에서는 가장 흔한 색이 채움이고 잉크는 글자이므로 같은 방법이 성립한다.
    영역은 잰 rect를 테두리 굵기만큼(4px), 그리고 **모서리 반경만큼 더** 줄인 것이다 —
    둥근 모서리에서는 위·아래 몇 줄에 도형 밖 흰색이 남아 그 줄이 하나의 띠로 세어졌다
    (radius=32 도형이 글줄 하나인데 띠 셋으로 읽혔다). 채움이 잰 rect를 덮지 않는 프리셋
    (diamond·chevron…)은 이 방법이 성립하지 않아 목록으로 닫았다 (`inShapeBandPresets`) —
    도형 안 텍스트는 rect·rounded-rect에서만 쓰인다.
  - **줄 수 갈라짐은 표로 단정한다** (`inShapeLineDivergences`), 픽셀 오차로 덮지 않는다.
    갈라진 자리와 수를 적어 두면 (1) 나빠지면 깨지고 (2) **고쳐도 깨진다** — 표를 지우는 것이
    고쳤다는 증거다. 픽셀 비율로 덮으면 이 사실이 기록에서 사라진다.
  - **픽스처 슬라이드 셋을 고쳤다.** 세 수정의 기준은 같다 — *그 슬라이드가 시험하려는 것을
    시험하게 만드는가*:
    - `001`의 이름표를 도형 **밖으로**. 프리셋의 텍스트 사각형은 외곽선이 아니라 프리셋 정의가
      정하는 더 작은 사각형이라(diamond는 안쪽 절반), LibreOffice가 `diamond`를 `diamo/nd`로,
      `parallelogram`을 `parallelog/ram`으로 접었고 `adj=100000`인 `chevron`은 텍스트 사각형이
      붕괴해 글자가 **한 글자씩 세로로** 도형 밖까지 흘렀다 (NOTES 04/05의 갈라짐 (d)가 그림이
      된 것이다 — 그 세로 글자 기둥이 위 줄 도형의 잉크 띠를 20px 늘렸다). 기하 판정 슬라이드가
      글자 갈라짐에 묻히므로 이름표를 도형 밖에 뒀다. 항목 명세가 선택지로 적어 둔 수정이다.
      결과: 슬라이드 1의 픽셀 오차 2.70% → 2.45%, 도형별 오차의 정체가 테두리 잔차로 좁혀졌다.
    - `005`를 두 줄로. 다섯 도형을 한 `<row>`에 두면 선언한 240px가 들어갈 자리가 없어
      (5×240 + 4×24 > 1152) flex가 213px로 줄이고, 그러면 앵커 라벨이 박스를 **정확히** 채워
      여유가 0%가 된다 — 라벨마다 줄이 하나 늘었다. 앵커를 보이려는 슬라이드가 줄바꿈 갈라짐을
      시험하게 되므로 줄을 나눠 선언한 크기가 그대로 서게 했다.
    - `007`의 오른쪽 본문을 한 문장으로. 원래 문장은 웹 3줄 / LibreOffice 4줄로 갈라져 **넷째
      줄이 도형 밖으로 흘렀다**. 이 슬라이드가 시험하는 것은 캐스케이드 순서다.
    갈라짐 자체를 픽스처에서 **없애지는 않았다** — 002·006의 두 자리를 그대로 두어 위의 표가
    회귀 감시선으로 남는다.
  - **도형별 크롭 여유 16px**: 테두리는 경로 중앙이라 굵기의 절반이 잰 rect 밖으로 번지고
    `callout`의 꼬리는 프리셋 정의대로 아래로 12.5%(13.75px) 나간다. 격자 간격(24px)보다 작아야
    이웃 도형이 크롭에 들어오지 않는다 — 들어오면 오차의 주인이 흐려진다.
  - **회전 도형 검사의 기대값 출처는 저자가 선언한 크기**다 (골든 XML만 읽어 브라우저 없이
    돈다). 측정값과 맞대면 측정이 bbox를 재기 시작한 순간 두 값이 함께 커지며 통과한다 —
    실패 메시지가 √2배 값을 이름으로 짚는다.
  - **픽스처 diff 조정의 근거를 셋으로 나눴다**: 기하는 도형별 크롭이, 줄 수는 도형 안 띠가,
    나머지 잔차만 슬라이드 비율이 본다. 그래서 3.7%(픽스처)·5.2%(006)는 **잔차만 남은 수치**다 —
    도형이 스무 개면 외곽선 둘레만으로 1.4%가 쌓인다 (220×110 도형 하나의 둘레 660px =
    슬라이드의 0.07%). 006에 슬라이드 단위 조정을 준 이유는 도형이 가장 많고(10개) 알려진 줄
    갈라짐이 겹치는 유일한 슬라이드라서다 — 픽스처 전체에 같은 수치를 주면 나머지 슬라이드의
    실제 배치 갈라짐이 그 그늘에 숨는다 (03/05의 결정 그대로).

- **검증**: `gofmt -l .`(무출력), `go vet ./...`, **`make test DOCKER=finch`(컨테이너) 전부
  초록**. **기존 골든 4개(00_empty·01_text·02_text_subset·03_layout)는 한 파일도 움직이지
  않았다** (`git status -- testdata/fixtures/0{0,1,2,3}_*`가 비어 있다) — 항목 04·05의 불변
  계약이 픽스처 활성화에서도 성립한다. 새 골든의 diff는 전부 04_shape 아래이고, 그 내용은
  슬라이드 8장·노트 1장·경고 2건이다.
  `diagnostics.txt`가 경고 **둘**을 담는다 (도형 variant `card-heder` + 텍스트 variant `leed`) —
  문구는 텍스트 경고와 바이트가 같고 나열된 목록만 다르다 (항목 03의 결정이 골든으로 확인됐다).
  리터럴 `srgbClr`는 슬라이드 파트 전체에서 **slide3의 두 값뿐**이다 (`0F65F9`·`6B7280`).
  픽셀 비교(슬라이드 8장): 2.45 / 2.85 / 2.34 / 1.65 / 3.18 / 4.62 / 2.67 / 2.12%.
  `make review DOCKER=finch`로 `review/04_shape/` 갤러리를 만들고 24장을 **눈으로 봤다** —
  프리셋 12종의 외곽선이 겹치고, 앵커 3종·`pad` 유무·회전 45도·`flip=h`·`opacity=0.4`·
  `shadow=soft`·radius 3종·카드 그리드·variant 두 열이 두 타깃에서 같은 그림이다.

- **남긴 것**:
  - **사람이 확인할 것** (TASKS의 리뷰 체크박스 — 열어 두었다). macOS에서
    `review/04_shape/shape-deck.pptx`를 PowerPoint/Keynote로 열어:
    (a) 도형을 선택하면 **도형 서식**이 뜨고 "도형 모양 변경"으로 다른 프리셋이 되는지
    (텍스트 박스로 표시되면 항목 05의 `txBox` 결정이 틀린 것이다),
    (b) **디자인 탭에서 테마 색을 바꾸면** 002·006·007의 토큰 도형이 함께 바뀌고 003의 리터럴
    도형(`#0F65F9`)만 그대로인지 — 이 iteration의 제품 약속이고 자동 검사가 볼 수 없는 절반이다
    (자동 검사는 XML에 `schemeClr`가 있다는 것까지만 본다),
    (c) 005·006의 도형 안 글자를 클릭해 편집할 수 있고 도형을 끌면 글자가 따라오는지,
    (d) 004의 45도 도형의 조절점이 **회전 전** 박스에 있는지, 그림자·모서리 반지름이 서식 창에서
    **편집 가능한 값**으로 보이는지 (그림으로 구워졌으면 값이 없다),
    (e) 001의 12종 모양이 웹 스크린샷과 같은지.
  - **항목 07(문서)이 쓸 이 항목의 사실**: `srgbClr` 계약이 "0건"에서 "토큰은 눕지 않는다"로
    정밀화된 것, 도형 픽셀 오차의 성질이 글리프 잔차와 다르다는 것(모양이 다르면 오차를 키우지
    않는다 + 위의 12종 표), 잉크 띠 전제가 칠한 도형에서 깨지는 것과 판정을 도형 안쪽으로 옮긴
    것, 그리고 **프리셋의 텍스트 사각형은 외곽선보다 작다**는 사실 (`docs/internals/testing.md`
    와 `shape.md`의 도형 안 텍스트 절 양쪽에 자리가 있다 — 저자가 diamond 안에 긴 글자를 넣으면
    PowerPoint에서 접힌다).
  - **열린 질문 3건**:
    (a) **줄바꿈 규칙을 맞출 것인가** (NOTES 04/05의 (a)를 대체한다). 후보는 웹에
    `word-break: keep-all`을 걸어 Chromium도 낱말 경계에서만 끊게 하는 것이다 — 규칙이 같아지면
    두 엔진의 줄 수가 구조적으로 맞고 `TextBoxRect`의 4% 여유도 뜻이 좁아진다. 대가는 **모든
    덱의 타이포그래피가 움직이는 것**(기존 골든 4개가 전부 바뀐다)이고, 긴 한글 낱말이 좁은
    박스에서 넘칠 수 있다. 이 항목의 범위 밖이라 손대지 않았다 — ADR과 함께 별도 항목이 받을
    일이고, Iteration 12(품질 게이트)가 자연스러운 자리다.
    (b) **프리셋의 텍스트 사각형을 웹이 따라야 하는가.** 지금 웹은 도형 박스 전체에 글자를
    그리고 PowerPoint는 프리셋이 정한 더 작은 사각형에 그린다 — 001을 고쳐 픽스처에서는
    사라졌지만 저자가 `<shape type="diamond">` 안에 글자를 쓰면 그대로 만난다. 고치는 길은
    기하 유도표(`render/web.shapeGeometries`)에 프리셋의 `rect`를 한 열 더 유도해 웹의 자식
    텍스트 박스를 그 안으로 넣는 것이고, 그것은 항목 04의 표를 다시 유도하는 일이다.
    `chevron`의 붕괴(`adj=100000`에서 폭 0)는 그 길에서도 남는다 — 조정값을 자르면 모양이
    갈라지고, `custGeom`으로 내면 프리셋 도형이 아니게 된다.
    (c) **`inShapeBandPresets`가 둘뿐인 것.** 채움이 잰 rect를 덮지 않는 열 프리셋에서는 도형 안
    줄 수를 세는 방법이 없다 — 지금은 그 프리셋에 도형 안 텍스트를 쓰는 픽스처가 없어 문제가
    되지 않지만, (b)를 고치는 날 그 자리에도 판정이 필요해진다. 후보는 잉크 판정의 배경색을
    호출자가 넘기는 것이지만(그러면 테마가 바뀔 때 테스트가 따라 바뀐다) 지금 고를 근거가 없다.

## Iteration 4 · 07-docs — 도형 레퍼런스 + 외관 매핑 + ADR 7건
<!-- item: 04/07-docs -->

- **한 것**:
  - `docs/reference/soml/shape.md` **신규** — `type` 12종 표(SOML 이름 · 웹에서 그려지는 것 ·
    OOXML `prstGeom` 이름, 이름이 갈라지는 넷을 굵게), 속성 표 네 벌(고유·외관 8종·자기 박스·
    자식 배치), 도형 안 텍스트 절, 픽스처에서 가져온 예제 셋, 웹/PPTX 렌더 결과(실제 산출
    HTML·XML), **네 갈래 도형 수 대비 표**, 에러 조건 표 두 벌(파서 + 값 층).
  - `docs/reference/appearance.md` — `variant` 절을 **어휘 두 벌**(텍스트 2종 + 도형 6종)의 표로
    바꾸고 "도형 variant는 자식 텍스트를 담지 않는다"를 명시, **컴포넌트 기본값 절** 신규
    (도형 캐스케이드의 바닥), **도형 외관 매핑 표 11행** 신규, 텍스트 매핑 표에 `anchor` 행,
    토큰 색 보존 계약 절에 **리터럴은 `srgbClr`로 나가는 것이 정상**과 그 차이가
    PowerPoint에서 어떻게 나타나는지, 색 토큰 표에 `surface`, **반경·그림자 토큰 표** 신규
    (PPTX 슬롯 열에 "없음"과 "있으나 v1은 쓰지 않음"을 구별해 적었다).
  - `docs/reference/soml/text.md` — `anchor` 행 + 물리 이름인 이유 + **기본값이 없는 유일한
    속성**이라는 사실, 첫 자식이 도형의 앵커를 정하는 규칙, 색 토큰 진단 문구 갱신
    (속성 이름 접두 + `surface`).
  - `docs/reference/soml/text-subset.md` — 블록 태그 허용 속성에 `anchor`, 진단 문구 갱신.
  - `docs/reference/soml/layout.md` — 자식 어휘에 `<shape>`, **컨테이너 속성 표의 아래 다섯 행이
    도형과 공유된다**는 사실, PPTX 도형 수 표에 `<shape>` 행(정반대인 자리를 명시), 도형 수
    등식의 새 항, 진단 문구의 자식 어휘 목록 갱신.
  - `docs/reference/soml/slide.md` — 자식 어휘와 진단 목록에 `<shape>`.
  - `docs/reference/index.md` — 현재 상태를 Iteration 4로.
  - `docs/internals/geometry-pass.md` — 측정 대상에 도형과 도형 안 텍스트, **재는 것과 도형이
    되는 것의 비대칭**(도형은 일치하는 첫 종류, 그 자식은 반대), 도형 수 등식의 새 항,
    **"회전 도형은 회전 전 박스를 잰다"** 절 신규, 텍스트 박스 전략에 "도형에는 여유를
    적용하지 않는다"와 "도형의 `bodyPr`만 인셋을 갖는다".
  - `docs/internals/testing.md` — **"토큰이 눕지 않는다" 절 신규**(계약이 0건에서 전수 표로
    정밀화된 이유), 잉크 띠 전제가 도형에서 깨지는 것과 판정을 도형 안쪽으로 옮긴 것,
    도형 안 줄 수 갈라짐을 표로 단정하는 것, **"모양이 다르면 오차를 키우지 않는다"**와
    12종 오차 표.
  - `docs/index.md` — 현재 상태 Iteration 4, `<shape>` 웹 렌더 칸, `opacity`/`shadow` 행.
  - **ADR 7건** (`docs/decisions/0016`~`0022`) + `docs/decisions/index.md` + TASKS 하단 결정
    로그 표, `mkdocs.yml` nav에 새 페이지 8개.
  - **PRD 갱신 5곳**: §8.3의 `card-header`에서 `font`/`size`/`weight`를 뺐고 variant가 자신의
    외관만 담는다는 확정을 적었다, §7.6 표의 `border`(조각 3개·순서 고정·`none`)와
    `radius`(`type="rounded-rect"`인 shape), §7.6 아래에 확정 4건(border 축약·radius 전용·
    shadow 토큰만·opacity는 색 알파)과 리터럴 색의 뜻, §7.5의 `<shape>` 웹 렌더 칸
    (`div/SVG` → 박스 div + 그림 svg), §9.4의 `shadow`(`box-shadow` → `filter: drop-shadow`)와
    `opacity`(alpha → 색 안의 알파)와 `radius`(SVG는 `rx`).
  - `TASKS.md` — Iteration 4 **문서 체크박스 3개를 닫았다**. 리뷰 체크박스는 열어 두었다.

- **설계 결정과 근거**:
  - **ADR을 7건 올렸다.** 항목 명세의 후보 7개를 모두 승격했고, 국소 매핑 선택
    (`type` → `prstGeom` 이름 대응, `adj` 환산식, 토큰 수치)은 올리지 않았다 — 근거가 코드
    주석과 레퍼런스 표에 있으면 충분하다는 02/09·03/06의 기준 그대로다. 후보 6번(`radius`
    전용)과 7번(`shadow` 토큰만)은 "레퍼런스 표에 남길지"가 판단이었는데 올린 이유가 각각
    있다: `radius`는 **테마 값과 저자 값에 같은 규칙을 다르게 거는 첫 사례**이고(Iteration 10의
    테마 포맷이 다시 만난다), `shadow`는 **저자가 수치를 쓸 수 없게 만든 결정**이라 "왜 못
    쓰는가"의 답이 표의 한 칸보다 길다.
  - **ADR 0017에 조정값 소유권을 합쳤다.** 항목 05가 남긴 ADR 감("조정값 명세는 늘린 쪽이
    소유하고 평가는 재는 쪽이 한다")을 별도 ADR로 두지 않은 이유: 그 분업은 "정규 공간을 늘리고
    PPTX가 맞춘다"는 결정의 **구현 방식**이고, 따로 두면 두 문서가 같은 근거(웹은 도형의 픽셀
    크기를 모른다)를 두 번 적는다.
  - **12종 표의 열을 셋으로 두었다** (SOML · 웹 · `prstGeom`). TASKS가 요구한 것은 "각각 웹/PPTX
    `prstGeom` 대응"이고, 이름이 갈라지는 넷(`cylinder`→`can`, `arrow-right`→`rightArrow`,
    `callout`→`wedgeRectCallout`, `star`→`star5`)이 표의 존재 이유이므로 굵게 두었다. 웹 열에
    "px 공간 / 정규 공간"을 함께 적은 이유는 그 구별이 **비율이 갈라지는 프리셋이 어느 것인지**를
    말하기 때문이다.
  - **`shape.md`가 진단 문구를 실제 문자열로 싣는다** — 구현에서 그대로 가져와 확인했다
    (`go run`으로 파서·값 층을 돌려 문구를 대조했고, 그 과정에서 문서에 남아 있던 낡은 목록
    셋을 고쳤다: `<text>`·블록의 허용 속성에 `anchor`, `<row>`·`<slide>`의 자식 어휘에
    `<shape>`, 색 토큰 목록에 `surface`와 속성 이름 접두).
  - **네 갈래 도형 수 표를 `shape.md`와 `layout.md` 양쪽에 두었다.** 중복이지만 답을 찾는
    사람의 출발점이 둘이다 — 도형을 쓰는 저자는 `shape.md`에서, 컨테이너를 쓰는 저자는
    `layout.md`에서 "내 슬라이드가 PowerPoint에서 도형 몇 개가 되는가"를 묻는다. 두 표의
    문장이 갈라지지 않게 **정반대인 자리**(컨테이너는 사라지며 자식을 도형으로, 도형은 남으며
    자식을 흡수)를 같은 말로 적었다.
  - **크기·여백 속성을 `layout.md`에 남기고 `shape.md`가 가리킨다.** 목록이 두 곳에 전문으로
    있으면 PRD 표에 크기 속성이 하나 늘 때 한쪽만 갱신된다 — 코드에서 `BoxAttrs()` 한 벌을
    공유한 것과 같은 근거다. `shape.md`에는 도형에서 달라지는 것만 적었다 (백분율 `pad`는
    에러, 크기를 적는 것이 보통이다).
  - **`gradient(…)`가 지금 에러라는 사실을 레퍼런스에 적었다.** 미구현을 문서에 쓰지 않는
    규칙과 충돌하지 않는다 — 적은 것은 기능이 아니라 **진단**이고, 미래 iteration을 예고하는
    것은 진단 문구의 일이 아니라 문서의 일이다.
  - **PRD를 뒤집은 곳에는 ADR을 붙였다** (§8.3의 variant ↔ ADR 0022, §7.6의 radius ↔ ADR 0019,
    §9.4의 shadow ↔ ADR 0020, opacity ↔ ADR 0018). 구현이 PRD를 따르는 것이 기본이므로 뒤집은
    쪽에 근거가 없으면 다음 사람이 어느 쪽이 맞는지 알 수 없다.

- **검증**: `make docs`(strict) 통과 — nav에 없는 문서·깨진 내부 링크가 없다는 뜻이다
  (새 페이지 8개를 nav에 넣었고 ADR 링크 12개가 실제 파일을 가리킨다).
  `gofmt -l .`(무출력), `go vet ./...`, **`make test DOCKER=finch` 초록** (문서 항목이므로
  코드는 건드리지 않았고 골든은 한 파일도 움직이지 않았다).
  문서의 사실을 산출물로 대조했다: 웹 HTML의 도형 구조(`balpyo-shape` div + `balpyo-shape-ink`
  svg, `rx="12"`, `transform: scaleX(-1)`, `filter: drop-shadow(var(--balpyo-shadow-soft))`,
  `--balpyo-shadow-soft: 0px 4px 16px color-mix(…)`), PPTX의 `sp`(`prstGeom prst="roundRect"` +
  `avLst`, `schemeClr bg2`/`tx2`, `a:ln w="9525"` + `prstDash val="solid"`, `bodyPr`의 네 인셋과
  `anchor="t"`, `outerShdw blurRad="152400" dist="38100" dir="5400000" rotWithShape="1"`),
  그리고 진단 문구 9건을 `go run`으로 직접 찍어 문서의 표와 맞댔다.

- **남긴 것**:
  - **TASKS의 Iteration 4 리뷰 체크박스만 열려 있다** — 사람이 macOS에서
    `review/04_shape/shape-deck.pptx`를 열어 확인할 다섯 가지는 NOTES 04/06의 **남긴 것**에
    적혀 있다 (프리셋 도형 여부, 테마 색 추종과 리터럴의 비추종, 도형 안 글자 편집,
    회전 조절점·편집 가능한 그림자·반경, 12종 모양).
  - **문서를 쓰다 발견한 것** (코드 변경은 하지 않았다):
    (a) `soml.BorderStyle`의 주석이 파선 계열을 `sysDash`/`sysDot`으로 적고 있는데 실제 매핑은
    `dash`/`dot`이다 (항목 05가 이미 남긴 발견이고 문서는 실제 값을 적었다).
    (b) `soml.ShadowToken`의 주석에 "매핑의 실제 결정은 항목 05"라는 문장이 남아 있다 —
    그 결정은 끝났고 ADR 0020이 그 자리다.
    (c) PRD §7.8의 dim-card 예제가 `gap="none"`을 쓰는데 공간 토큰 어휘는 `xs`~`2xl`이라 그
    예제는 지금 에러다 (NOTES 04/01의 열린 질문 (d)). 이 iteration의 결정과 무관해 고치지
    않았다 — 토큰 `none`(=0)을 추가할지는 테마의 결정이고(Iteration 10), 추가하지 않으면 그
    예제를 `gap="0"`으로 고쳐야 한다.
  - **열린 질문 2건** (문서에 적지 못한 것):
    (a) **도형 안 리스트·중첩**을 열 날의 문서 자리. 지금 `shape.md`는 "아직 자식이 아니다"로
    닫아 두었지만 카드 안 불릿은 실물 덱의 흔한 단위다 — 열리면 도형 안 텍스트 절이 리스트의
    `marL`/`indent`와 도형 인셋의 관계를 함께 설명해야 한다.
    (b) **프리셋 텍스트 사각형의 문서 자리.** ADR 0017의 마지막 항에 "유도하지 않았다"로 적었고
    `shape.md`의 001 픽스처 주석이 같은 사실을 담지만, **저자를 위한 경고로는 약하다** —
    `<shape type="diamond">` 안에 긴 글자를 쓰면 PowerPoint에서 다르게 접힌다는 사실은
    도형 안 텍스트 절의 눈에 띄는 자리(admonition)에 있어야 할 수 있다. 지금 적지 않은 이유는
    그 갈라짐의 범위를 수치로 재지 않았기 때문이다 (픽스처에서 그 조합을 뺐다).

## Iteration 5 · 01-parser-image — 파서가 `<image>` 어휘를 연다
<!-- item: 05/01-parser-image -->

- **한 것**:
  - `soml/image.go` (신규) — 이미지 카탈로그. `ImageElement`, `ImageFit`(`FitCover`·
    `FitContain`)과 `ImageFits()`, 속성 표 넷(`imageOwnAttrs`=`src fit alt`,
    `imageAppearanceAttrs`=`variant border radius shadow opacity rotate flip`, 박스는
    `BoxAttrs()` 공유, 자식 배치는 `withChildPlacement`), `ImageAttrs()`·
    `ImageAppearanceAttrs()`. `shape.go`·`layout.go`·`subset.go`와 같은 자리·같은 태도다.
  - `soml/subset.go` — `ElementKind`에 `KindImage`(열거형은 여기 모은다는 03/01의 결정),
    `elementKinds`에 등록, `IsVoidElement`에 `<image>` 추가.
  - `soml/parse.go` — `elementSpecs`에 `<image>`(`modelVoid`, 자식 어휘 없음),
    `slideChildren()`에 `<image>`.
  - `soml/layout.go` — `layoutChildren()`에 `<image>` (컴포넌트 셋이 앞에 온다:
    `text shape image`).
  - `soml/image_test.go` (신규) — 정상 트리 13종·`HasValue`·Span·에러 21종(line:col과 문구
    단정)·종류 카탈로그·`fit` 어휘 순서와 복사본 계약·속성 표 넷의 순서·자식 어휘 두 자리.
  - `soml/shape_test.go`·`soml/layout_test.go` — `<image>`가 어휘 안으로 들어온 만큼만 고쳤다
    ("image는 아직 어휘 밖이다" 케이스 삭제, `<row>` 자식 어휘 진단 문구에 `<image>` 추가).
  - `testdata/fixtures/_05_image/` (비활성) — 슬라이드 6개 + **에셋 3개**(프로젝트에
    `assets/`가 처음 생긴다): `001_fit`(fit 2종 × 넓은/좁은 그림 4칸), `002_appearance`
    (외관 6종 각각 한 속성만), `003_formats_reuse`(PNG·JPEG + 같은 `src` 두 번),
    `004_image_text`(2단 + 링크 + 노트), `005_grid`(`<grid cols="3">`·`span`·`pad`),
    `006_full_bleed`(슬라이드 직속 + `<spacer/>`).
  - `PRD.md` §7.5 — `<image>` 행의 `frame`을 지우고 `alt`을 넣었고, 두 결정의 근거를 절의
    불릿으로 남겼다 (ADR은 항목 06).

- **설계 결정과 근거**:
  - **`frame`을 열지 않았다** (PRD 표에서 지웠다). 뜻이 PRD 어디에도 정의되어 있지 않고
    (§7.6 속성 표에 행이 없다), 액자가 뜻하는 것(테두리+반경+그림자의 묶음)은 이미
    `variant`가 소유하는 채널이다. 열면 캐스케이드에 5단이 생기고 `variant`와 `frame`이
    충돌할 때의 승자를 정해야 하는데, 그 규칙은 저자가 외울 이유가 없다. 항목 02가 테마에
    `framed` variant를 정의하면 그것이 이 결정의 실물 증거가 된다.
  - **`alt`을 열었다** (PRD 표에 넣었다). 기준은 이 저장소가 속성을 여는 기준 그 자체다 —
    양 타깃에 실을 자리가 있다: 웹 `<img alt>`, PPTX `p:cNvPr@descr`. 자리가 있는데 열지
    않으면 산출물이 접근성 없는 그림을 내고 저자가 고칠 방법이 아예 없다. 생략 시 웹이
    `alt=""`(장식 이미지)를 내는 것까지가 결정의 일부이고, 그 절반은 항목 03의 몫이다.
  - **`fill`을 열지 않았다.** PRD §7.6 표의 적용 대상에 image가 없고, 그림 뒤의 칠이 보이는
    자리는 `contain`의 여백뿐인데 그 여백은 PPTX에서 `pic`의 박스 밖이다 — 색을 받아 놓고 한
    타깃에서만 그리는 속성이 된다.
  - **`pad`는 `BoxAttrs()` 한 벌 그대로 받았다** (도형·컨테이너와 공유). 양 타깃에 자리가
    있다고 판단했다: 웹은 `<img>`의 padding이 콘텐츠 박스를 줄이고 `object-fit`이 그 박스에
    적용되며, PPTX는 `pic`의 `xfrm`을 여백만큼 줄여 내면 같은 결과다 (`bodyPr`의 인셋은
    텍스트의 것이라 쓰지 않는다). 목록을 갈라 두는 길과 저울질했고 공유를 골랐다 — 두 벌이
    되면 PRD 표에 크기 속성이 하나 늘 때 어느 태그가 빠졌는지 사람이 세어야 한다.
    **되돌릴 자리는 `005_grid`의 `pad="md"` 칸이다** (아래 열린 질문).
  - **`KindImage`를 별도 종류로 두었다.** `KindShape`와 합치지 않은 근거는 하나로 갈라진다:
    도형은 자식 텍스트를 `txBody`로 흡수하고 그림은 담을 것이 없으며, PPTX에서 `sp`와 `pic`은
    다른 요소다. 합쳐 두면 두 렌더러가 다시 이름으로 갈라야 한다. **재는 것과 도형이 되는
    것이 일치하는 두 번째 종류**다 (ADR 0009·0013의 표에 행이 하나 는다 — 항목 06).
  - **`modelVoid`.** `<image src="…" />` 하나가 전부다. 캡션은 형제 `<text>`이고
    (`004_image_text`·`001_fit`이 그 표기), 그림 안 글자는 도형의 일, 그림 위 글자는
    Iteration 6의 `<stack>`이다. 부작용 하나를 확인했다: void 위반의 진단은 여는 태그
    한 줄이고 안쪽 내용은 진단하지 않는다 (기존 `<spacer>`·`<br>`의 동작을 그대로 받는다) —
    `<image>...</image>`를 쓴 저자는 "void 태그 `<image>`은 내용을 가질 수 없다" 하나를 본다.
  - **`shapeChildren()`에는 넣지 않았다.** 도형 안 그림은 PPTX에서 `sp` 안의 `pic`이 아니라
    도형의 `blipFill`이거나 그룹이고, 둘 다 "도형은 `txBody` 하나를 갖는다"는 Iteration 4의
    매핑과 다른 것이다.
  - **`fit` 어휘를 2종으로 닫았다** (`cover`·`contain`, PRD 순서). CSS `object-fit`의
    `fill`/`scale-down`/`none`을 버린 근거는 `<shape type>`을 OOXML 프리셋으로 닫은 것과 같다 —
    `none`은 `pic`의 박스 밖으로 그림이 넘치는 상태라 PPTX에 대응이 아예 없다. 타입을 둔
    이유는 `BorderStyle`·`Flip`과 같다 (확정 값이 될 자리가 있다).
  - **에셋을 손으로 만들어 커밋했다** (생성 스크립트를 남기지 않았다 — 골든은 결정적이어야
    한다). `wide.png` 16×8(97B), `tall.png` 8×16(98B), `photo.jpg` 16×8(652B). 전부 2색이고
    **표식이 좌상단에만** 있다: `fit` 검증은 종횡비가 박스와 다를 때만 뜻이 있어 세로/가로
    각각을 만들었고, 대칭이면 크롭 축이 뒤바뀌어도 픽셀 비교가 통과한다. 픽스처의 박스는
    전부 `ratio`로 못 박고 그림의 고유 비율과 다르게 두었다 (같으면 cover와 contain이 같은
    그림을 낸다). 사진 한 장을 넣으면 저장소가 그 대가를 영구히 낸다 — 골든에 웹·PPTX 양쪽으로
    바이너리가 들어간다 (항목 05).

- **검증**: `gofmt -l .`(무출력), `go vet ./...`, **`make test DOCKER=finch` 초록**
  (컨테이너가 진짜 게이트다). 골든은 한 파일도 움직이지 않았다 — 픽스처가 `_` 접두로 꺼져
  있고 기존 어휘의 파싱 결과가 바뀌지 않는다는 계약의 실물 확인이다.
  호스트(macOS)에서 `go test ./...`을 돌리면 `01_text`~`04_shape` 골든의 텍스트 폭
  (`a:ext cx`)이 어긋나는데, `git stash`로 이 커밋의 변경을 전부 뺀 상태에서도 **같은 diff**가
  나오는 것을 확인했다 — 폰트 메트릭이 컨테이너 밖에서 다른 것이고 이 항목과 무관하다.
  픽스처의 `src`가 실제 파일을 가리키는지는 파서 테스트가 `os.Stat`으로 단정한다 (파서는
  파일을 보지 않으므로 그 단정이 여기 있어야 항목 02 전에 오타가 잡힌다).

- **남긴 것**:
  - 항목 02가 이어받을 표면: `ImageAttrs()`·`ImageAppearanceAttrs()`·`ImageFits()`(값 층이
    소비할 목록), `KindImage`(순회의 분기), `Attr.HasValue`(`<image src>`와 `src=""`의 구별 —
    "경로가 없다"와 "경로가 비었다"를 다른 진단으로 낼 수 있다). 두 렌더러는 여전히
    `<image>`에서 "매핑이 없다"로 멈춘다 (`render/web.writeNode`·`render/pptx.writeShapeNode`의
    기존 가드 — 이 항목이 손대지 않았다).
  - **문서가 아직 낡았다** (항목 06의 몫): `docs/reference/soml/slide.md`의 `<slide>` 자식
    어휘, `docs/reference/soml/layout.md`의 `<row>` 자식 어휘와 진단 표, `shape.md`의
    "`<image>`를 썼다 → 알 수 없는 태그" 행. 이 항목에서 고치지 않은 이유는 미구현 동작을
    레퍼런스에 쓰지 않는 규칙이다 — 지금 `<image>`는 파싱만 되고 그려지지 않는다.
  - **열린 질문 3건**:
    (a) **`pad`를 이미지에 열어 둔 것** (위 결정). 항목 03·04가 두 타깃의 계산을 실제로 맞추지
    못하면 `<image>`용 속성 목록을 따로 이어 `pad`만 떼고, `005_grid`의 `pad="md"` 칸을
    지운다. 그 판단의 기준을 정해 둔다: 웹의 잰 rect가 padding을 포함한 박스이므로 PPTX가
    `xfrm`을 줄일 때 **좌표 대조 테스트**(`xfrm` == 측정값 EMU, 0 오차)와 부딪히면 그것이
    갈라지는 증거다.
    (b) **`alt` 없는 그림의 웹 산출** — `alt=""`를 내기로 한 것은 이 항목의 결정이지만 실물은
    항목 03이 낸다. PPTX `descr`을 생략할지 빈 문자열로 낼지는 아직 정하지 않았다 (PowerPoint의
    대체 텍스트 칸이 빈 것과 속성이 없는 것의 차이를 확인하지 못했다 — 항목 04).
    (c) **픽스처에 `variant`를 쓰지 않았다.** 테마의 이미지 컴포넌트와 variant 이름은 항목 02의
    결정이므로 이름을 미리 굳히지 않았다 — 항목 02가 `framed`를 정의하면 `002_appearance`에
    그 칸을 더하는 것이 `frame`을 열지 않은 결정의 실물 증거가 된다.

## Iteration 5 · 02-core-assets — 값이 디스크를 가리킨다: 에셋 표 + 이미지 확정 외관
<!-- item: 05/02-core-assets -->

- **한 것**:
  - `soml/asset.go` (신규) — 에셋 표. `ImageFormat`(`FormatPNG`·`FormatJPEG`)·`ImageFormats()`,
    확장자 표(`imageExts` → `ImageExts()`·`ImageFormatForExt()`), `Asset`(키·포맷·고유 크기·
    바이트 + `Ext()`), `Assets []Asset`(`Get`·`Keys`). 표를 **덱에 두었다** — `soml/deck.go`의
    `Deck.Assets`(Theme의 형제).
  - `soml/imagestyle.go` (신규) — `ImageStyle`(`Src`·`Fit`·`Alt` + 외관 6종 + `Box`)과
    `Element.ImageStyle()`. `soml/appearance.go`에 `Appearance.Image`를 `Shape`의 형제로 달았다.
  - `soml/image.go` — `DefaultImageFit = FitContain`(언어의 기본값).
  - `core/image.go` (신규) — `<image>` 속성의 값 해석. `resolveImage`(4단 캐스케이드),
    `imageVariant`, `imageAttrs`, `imageSrc`(경로 문법·정규화·확장자 어휘), `srcScheme`,
    `imageExtReason`, `imageFit`(+`cssOnlyFits`), `imageAlt`, 진단용 목록 둘.
  - `core/asset.go` (신규) — 에셋 패스. `ResolveAssets(deck, fs.FS)` + `assetPass`(순회·`load`)와
    `extForFormat`. `image/png`·`image/jpeg`를 blank import로 등록한다.
  - `core/cascade.go` — `imagePatch`·`mergeImage` (세 번째 병합 복제).
  - `core/theme.go` — `imageComponentDefault()`(빈 패치), `imageVariants`/`imageVariantPatch`/
    `imageVariantNames` (`framed` = border 1 solid muted + radius md + shadow soft).
  - `core/shape.go` — `radius`에서 **값 문법만** `radiusValue`로 떼어 이미지와 공유한다
    (프리셋 제약은 도형에 남는다). `core/resolve.go` — `node`에 `<image>` 갈래,
    `attrExample`에 `src`·`fit`·`alt`.
  - `internal/pipeline/pipeline.go` — `os.DirFS(projectDir)`를 `fsys` 하나로 올려 `soml.Collect`와
    `core.ResolveAssets`가 **같은 fs.FS**를 쓴다. 호출 순서는 Resolve → ResolveAssets → Validate.
  - 테스트: `soml/asset_test.go`(신규), `core/image_test.go`(신규 — 표 전수·`src` 규칙·에러
    전수·도형과의 값 문법 대조·캐스케이드 단), `core/asset_test.go`(신규 — `fstest.MapFS` 표·
    에러 전수·`_05_image` 픽스처 통째 해석), `core/cascade_test.go`(이미지 캐스케이드 3건),
    `internal/pipeline/pipeline_test.go`(조립 2건).

- **설계 결정과 근거**:
  - **에셋 표는 `Deck.Assets`다** (곁 표가 아니다). ADR 0004(지오메트리는 곁 표)의 두 근거가
    여기서는 성립하지 않는다: 에셋은 파생이 아니라 **입력**이고, 렌더 전에 결정적으로 한 번
    채워지므로 `--target html,pptx`와 `pptx,html`이 같은 덱을 본다. 곁 표 + 생성자 주입은 두
    렌더러의 생성자를 모두 바꾸는데, 그 타입은 어차피 `soml`에 살아야 한다(core는 렌더러를,
    렌더러는 core를 임포트하지 않는다) — 즉 타입은 덱 패키지의 것이고, 그러면 덱에 필드로 두는
    것보다 얻는 것이 없다. `fs.FS` 핸들을 덱에 싣는 길은 버렸다(렌더러의 순수성과 테스트의
    디스크 무의존이 함께 깨진다). ADR은 항목 06이 쓴다.
  - **고유 크기는 표에만 있다** — `ImageStyle`에 담지 않았다 (항목 지시의 "확정 값에 고유 크기"와
    갈라지는 자리다). 근거: 같은 그림을 두 번 쓴 슬라이드에서 두 확정 값이 자기 크기를 다르게
    말할 수 있게 되고, 그것이 이 저장소가 가장 비싸다고 못 박은 실패(같은 사실의 두 벌)다.
    두 렌더러는 바이트를 얻으려고 어차피 표를 조회하므로(`deck.Assets.Get(style.Src)`) 조회가
    한 번 더 생기지도 않는다. 확정 값은 **어느 에셋인가**(`Src`)만 담는다.
  - **키는 `path.Clean`만 한다 — 대소문자를 눕히지 않는다.** 항목 지시의 "확장자를 소문자로
    정규화한다"는 요구는 **Format이 소유하게** 했다: 키로 파일을 다시 읽고(웹이 산출 디렉토리로
    복사한다) 대소문자를 구별하는 파일 시스템에서 `A.PNG`를 `a.png`로 눕히면 그 파일이 사라진다.
    콘텐츠 타입·미디어 파트 이름은 확장자가 아니라 `Asset.Format`에서 나오면 되고(그래서
    `.jpg`와 `.JPEG`가 한 `Default` 항목으로 모인다), 확장자 대소문자를 무시하는 것은
    `ImageFormatForExt`가 한다. `Asset.Ext()`가 소문자 확장자를 한 곳에서 준다 (항목 04가 쓸 표면).
  - **값 층을 둘로 나눈 선은 "디스크를 보는가"다.** `Resolve`는 경로 **문법**과 어휘(절대 경로·
    `..`·URL·역슬래시·확장자)를, `ResolveAssets`는 파일이 실제로 무엇인지(존재·디렉토리·헤더·
    고유 크기)를 본다. 그래야 에디터(lsp)가 프로젝트를 열지 않고도 문법 진단을 전부 낸다.
    `ResolveAssets`를 `Validate`의 형제로 둔 이유는 계약이 같기 때문이다: 파일 하나를 넘어야
    알 수 있는 것을 Resolve 다음에 본다.
  - **역슬래시를 막았다** (항목 지시에 없는 규칙 하나를 더했다). `src="assets\wide.png"`를
    조용히 `/`로 바꾸면 역슬래시를 파일 이름에 쓸 수 있는 파일 시스템에서 다른 파일을 가리키고,
    그 차이는 CI에서만 드러난다. 막으면 진단이 첫 번째 고칠 것을 정확히 말한다.
  - **`fit` 기본값은 언어에 두었다** (`soml.DefaultImageFit`, 테마의 컴포넌트 기본값이 아니다).
    `fit`은 외관 번들이 아니라 그림의 의미이고, 테마를 바꿨을 때 얼굴이 잘리기 시작하면 안 된다 —
    `<p>` ≡ `<text role="body">`가 테마가 아니라 언어에 있는 것과 같은 자리의 결정이다.
  - **이미지 컴포넌트 기본값은 비었다** (도형과 정면으로 갈라진다). "값이 비면 렌더러가 채우는
    판단을 한다"는 도형의 근거가 그림에는 걸리지 않는다: 채움 없는 도형은 보이지 않지만 그림은
    자기 픽셀을 갖고, 두 타깃의 암묵 기본값도 갈라지지 않는다(웹 `<img>`·PPTX `pic` 둘 다 장식
    없이 그린다). 기본으로 테두리를 두르면 스크린샷·도표에 이중 경계가 생기고 저자가 모든
    그림에 `border="none"`을 적어야 한다. 액자는 **variant `framed`**가 소유한다 — 항목 01이
    `frame` 속성을 열지 않은 결정의 실물이고, 세 조각(테두리·반경·그림자)을 함께 정하는 것이
    "번들"의 뜻이다.
  - **`imagePatch`를 새로 만들었다** (`shapePatch` 재사용을 버렸다). 이미지 패치가 도형 패치의
    부분집합(fill 하나 차이)이라 "도형 패치를 쓰고 fill만 버린다"가 가능해 보이지만, 그러면
    **버리는 코드**가 "테마가 정할 수 있는 것"의 두 번째 정의가 된다. 병합이 한눈에 보이는 대입
    목록인 것이 cascade.go의 값이고, 구조를 뽑는 것은 네 번째 채널(`<group>` — Iteration 9)이다.
    반면 **값 문법은 함수로 공유했다** (border·shadow·opacity·rotate·flip·박스는 도형의 함수를
    그대로 부르고, radius는 `radiusValue`로 문법만 떼었다) — 그 공유를 테스트가 값·진단 문구
    양쪽으로 대조한다.
  - **`radius`는 이미지에서 항상 유효하다.** ADR 0019의 근거는 프리셋 기하에 반경 조정값 자리가
    없다는 것이었고, 그림은 프리셋이 아니라 자기 클립을 갖는다(웹 `border-radius`, PPTX
    `pic/spPr/prstGeom prst="roundRect"`). 그 ADR의 적용 범위가 도형에 한정된다는 사실을 코드
    주석에 남겼고 문서는 항목 06이 고친다.
  - **`alt=""`를 받는다** (다른 외관 속성의 "비어 있다" 에러를 적용하지 않았다). `<img>`에서 속성
    없음과 빈 문자열은 스크린 리더에 다른 뜻이고 빈 문자열이 "장식 이미지"의 표기다. 생략과
    `alt=""`는 같은 확정 값이 된다(둘 다 웹에서 `alt=""`로 나간다 — 항목 01의 결정). 값 **없는**
    `alt`은 여전히 에러다. `src`도 `hasValue`를 그대로 쓰지 않았다: "지우면 기본값을 쓴다"가
    `src`에서는 틀린 안내다(기본값이 없다).
  - **진단은 이유와 대안을 말한다.** 지원하지 않는 확장자마다 이유가 다르므로
    (`imageExtReason`) svg는 `svgBlip`+폴백을, gif는 애니메이션을, webp는 버전별 지원을 말하고
    전부 "png나 jpeg로 내보낸다"로 끝난다. `fit="fill"`은 CSS 지식에서 온 값이므로 오타로 다루지
    않고 "PPTX에 대응이 없다 + `ratio`로 박스를 맞춘다"로 건너올 길을 준다.
  - **실패한 키도 `seen`에 넣는다** — 같은 오타를 슬라이드 수만큼 읽지 않게. 표는 매번 처음부터
    만든다(두 번 불러도 같은 표).
  - **PPTX/JPEG 디코더만 등록했다** (`image/gif`를 넣지 않았다). 어휘를 2종으로 닫은 결정이 그
    두 줄로도 강제된다: `.png` 이름의 gif는 `image.ErrFormat`으로 걸리고 진단이 "내용이 png가
    아니다"를 말한다.

- **검증**: `gofmt -l .`(무출력), `go vet ./...`, **`make test DOCKER=finch` 초록**(컨테이너가
  진짜 게이트다 — 픽스처 골든 테스트까지 통과했으므로 **골든은 한 바이트도 움직이지 않았다**).
  호스트(macOS) `go test ./...`에서는 `01_text`~`04_shape` 골든의 `a:ext cx`가 어긋나는데,
  `git worktree add /tmp/balpyo-head HEAD`로 만든 **이 커밋의 변경이 없는 트리에서 같은 diff**가
  나오는 것을 확인했다 (폰트 메트릭이 컨테이너 밖에서 다른 것이고 이 항목과 무관하다 —
  05/01의 관찰과 같다).
  손으로 확인한 것: `go run ./cmd/balpyo build --project testdata/fixtures/_05_image --target html`이
  `slides/001_fit.slide:12:7 <image>를 웹으로 렌더하는 매핑이 없다`로 멈춘다 — 값 층과 에셋
  패스가 픽스처 6장·에셋 3개를 **진단 없이** 통과하고 렌더 매핑에서만 멈춘다는 뜻이다
  (항목 03·04가 이어받을 정확한 지점).
  픽스처 에셋의 고유 크기가 05/01의 기록(wide 16×8, tall 8×16, photo 16×8)과 같고 표의 바이트가
  커밋된 파일과 동일하다는 것을 `TestResolveImageFixture`가 단정한다.

- **남긴 것**:
  - 항목 03·04가 이어받을 표면: `soml.ImageStyle`(확정 외관 — 렌더러는 이것만 읽는다),
    `deck.Assets`(`Get(style.Src)` → 바이트·포맷·고유 크기, `Asset.Ext()`, `Assets.Keys()`의
    결정적 순서 = 미디어 파트 번호의 근거), `soml.ImageFormats()`/`ImageExts()`(Content_Types의
    `Default` 항목), `core.ResolveAssets`(호출은 pipeline이 이미 한다).
  - 두 렌더러는 여전히 `<image>`에서 "매핑이 없다"로 멈춘다 (위 검증 참조). 픽스처는 `_` 접두다.
  - **문서가 아직 낡았다** (항목 06): `docs/reference/soml/image.md`(신규 — `src` 해석 규칙·포맷
    2종·`fit`·외관), ADR 두 건(에셋 표를 덱에 둔 결정, ADR 0019의 적용 범위가 도형이라는 사실),
    ADR 0009·0013의 측정 대상 표에 `<image>` 행, `appearance.md`의 적용 대상 열.
  - **열린 질문 3건**:
    (a) **`Asset.Bytes`를 덱에 싣는 비용.** 큰 사진 여러 장이 든 덱은 그 전부가 메모리에 남는다
    (렌더러가 바이트를 쓰는 시점까지). 지금은 픽스처가 1KB 미만이라 관찰할 것이 없지만, 실제 덱에서
    문제가 되면 표에 `Open() (io.ReadCloser, error)`를 얹는 길이 남아 있다 — 다만 그것은 렌더러의
    순수성을 다시 여는 결정이므로 값이 아니라 계약의 변경이다.
    (b) **EXIF 방향 태그.** 관찰하지 못했다 (픽스처의 JPEG에 EXIF가 없다). 엔진이 바이트를 고치지
    않기로 했으므로 방향 태그를 가진 JPEG는 브라우저(태그를 존중한다)와 PowerPoint(버전마다 갈린다)에서
    다르게 보일 수 있다 — 항목 03·04가 두 산출물을 나란히 볼 때 확인할 자리이고, 갈라지면 진단으로
    막는 것(바이트를 고치지 않는 결정과 일관)이 답이라고 본다.
    (c) **`Ext()`와 미디어 파트 이름의 소유.** `.jpg`/`.jpeg` 두 확장자가 한 포맷이므로 항목 04가
    파트 이름을 `Format`에서 만들 것(`image1.jpeg`)인지 원본 확장자에서 만들 것인지 정해야 한다.
    이 항목의 권고는 `Format`이다 — Content_Types의 `Default` 항목이 포맷당 하나로 모인다.

## Iteration 5 · 03-web-image — 잰 rect가 그려진 그림이다: `<img>` + 에셋 산출 + 디코드 대기
<!-- item: 05/03-web-image -->

- **한 것**:
  - `render/web/image.go` (신규) — `<image>` → **박스 div + `<img>`**. `writeImage`,
    `imageBoxCSS`(pad·크기·비율·자식 배치), `imageInkCSS`(fit + 외관 6종), `imageFitCSS`·
    `containFitDecls`·`boxHasSize`, `borderStyleValue`, `imageCSS`(엔진 CSS), `assetFiles`
    (에셋 → `render.Files`), 클래스 셋(`balpyo-image`·`balpyo-image-ink`·
    `balpyo-image-transform`)과 커스텀 프로퍼티 둘(`--balpyo-image-rotate`·`--balpyo-image-flip`).
  - `render/web/content.go` — `writeNode`에 `KindImage` 갈래, `features.images`,
    `contentCSS`에 `imageCSS()`, `spaceTokens()`에 이미지, **`shadowTokens()` 신설**
    (그림자 변수를 참조하는 어휘가 도형·그림 둘이 되었다). `style.go`의 조건이 그것을 쓴다.
  - `render/web/web.go` — `Render`가 `index.html` + 에셋을 낸다. 패키지 주석의 **측정 대상
    계약**에 `<image>` 행(재는 요소는 `<img>`이고 저자 박스는 식별자를 갖지 않는다).
  - `render/geom/browser.go` — `Session.Open`에 **이미지 디코드 대기 + 로드 실패 에러**
    (`imagesReadyScript`). `geom.go`의 `collect` 주석에 이미지 행.
  - 테스트: `render/web/image_test.go`(신규 — 외관 전수·`fit` 전수·`alt`·박스/배치·에셋
    `Files` 계약·중복 제거·결정성·도형과의 값 대조), `render/geom/measure_browser_test.go`
    (브라우저 6건: fit rect, `pad`, 회전 무효화, `ratio` 박스 회귀, 깨진 그림 에러, 디코드 대기),
    `render/web/fixture_test.go`·`measure_browser_test.go`의 헬퍼가 에셋 패스와 **파일 전부
    쓰기**를 하도록 고쳤다 (에셋 없이 재면 0×0을 잰다).

- **설계 결정과 근거**:
  - **박스 하나와 `<img>` 하나로 냈다 — 그리고 식별자는 `<img>`가 갖는다.** 항목 지시는
    `<img>` 하나를 말했지만 지시의 CSS 표(`100%`·`max-*: 100%`)는 **부모가 준 공간**을
    기준으로 하므로 저자의 `w`/`h`/`ratio`가 같은 요소에 실리면 성립하지 않는다: `w="fill"
    ratio="4:3"`가 `<img>`에 걸리면 그것이 요소 크기가 되고 `<img>`의 기본 `object-fit`은
    `fill`이라 **그림이 늘어난다**. 박스를 부모로 올리면 `100%`가 저자의 박스를 가리키고 잰
    rect는 언제나 그려진 그림이다. 도형과 층 수는 같고 **식별자가 붙는 층만 반대**다
    (도형은 겉 박스를 잰다 — 그림 svg가 박스를 덮으므로 둘이 같다).
  - **`fit` 매핑은 지시의 표 그대로다** (cover = `width/height:100%` + `object-fit:cover`,
    contain = `max-width/max-height:100%`, `object-fit` 없음). 실측으로 확인한 성질이
    결정의 일부다 — 아래 검증.
  - **크기 속성이 없는 그림은 `fit`이 아무것도 하지 않는다** (`boxHasSize`가 그 분기다).
    `cover`의 `width:100%`를 조건 없이 내면 고유 크기 그림이 부모 폭까지 늘어나
    (실측 16×8 → 1152×576) "크기 속성이 없으면 고유 크기"(soml.ImageAttrs의 계약)가 `fit`
    값에 따라 깨진다. min/max는 크기로 세지 않는다 — 한계는 박스가 아니고, `contain`의
    매핑이 이미 그 한계를 지킨다.
  - **외관은 값 함수를 공유하고 프로퍼티만 갈라진다.** 지시의 "도형의 CSS 조립을 재사용한다"를
    문자 그대로 할 수는 없다: 도형의 외관은 SVG 프로퍼티(`fill`·`stroke`·`rx`)이고 `<img>`에
    그것은 아무 일도 하지 않는다. 그래서 `colorValue`·`flipValue`·`formatNumber`·
    `shadowVarPrefix`·그림자 토큰 검사를 **함수째 공유**하고, 값을 싣는 프로퍼티만
    `border-width`/`border-style`/`border-color`/`border-radius`로 갈랐다. 그 공유를
    `TestImageAndShapeShareAppearanceValues`가 **값 단위로** 대조한다 (같은 `border="2 solid
    muted"`가 두 타깃에서 같은 `var(--balpyo-color-muted)`·`2px`를 내는지).
  - **파선 간격은 UA에 맡겼다** (알려진 갈라짐 하나). 도형은 `stroke-dasharray`로 OOXML
    `prstDash`의 비율을 정확히 내지만 CSS `border-style: dashed`의 간격은 UA가 정한다.
    맞추려면 그림을 svg 안에 넣어야 하고 그러면 `<img>`가 주는 것(고유 크기·`alt`·디코드
    신호)을 전부 잃는다 — 그 값이 파선 간격보다 크다고 판단했다.
  - **회전과 대칭이 한 `transform`을 나눠 쓴다** (커스텀 프로퍼티 둘 + 표식 클래스 하나).
    도형은 회전을 박스에, 대칭을 그림에 걸어 선언이 둘이지만 그림은 요소가 하나여서
    `transform`의 마지막 선언만 남는다. 합성은 `rotate(…) scale(…)`이고 오른쪽이 먼저
    적용되므로 "대칭 후 회전" — OOXML `xfrm`(flipH/flipV로 뒤집은 기하를 `rot`으로 돌린다)과
    같은 순서다. 선언을 엔진 CSS에 둔 이유는 도형과 같다: 인라인 style은 층 밖에서 이기므로
    측정 뷰가 덮을 수 없다.
  - **래퍼에 `min-width: 0; min-height: 0`을 넣었다** — 실측이 만든 결정이다 (아래 검증).
  - **`alt`은 언제나 낸다** (없으면 `alt=""`). 항목 01·02의 결정을 실물로 닫은 자리다.
  - **`src`가 빈 그림과 에셋 표에 없는 그림은 멈춘다.** `<img src="">`는 브라우저가 **현재
    문서를 다시 받는** 요청이고, 표에 없으면 산출물에 바이트가 없어 깨진 이미지가 된다 —
    둘 다 측정이 0×0을 재서 그 좌표를 PPTX `xfrm`으로 보내는 경로다.
  - **에셋 경로는 `Asset.Key` 그대로**이고 바이트는 표의 것을 그대로 넘긴다(복사 없음 —
    엔진은 그림을 고치지 않는다). `index.html`과 같은 이름의 에셋은 에러다 (산출 문서를
    덮어쓰는 길을 닫는다).
  - **`shadowTokens()`를 새로 뺐다** (`feat.shapes`를 그대로 쓰지 않았다): 도형 없이 그림만
    있는 덱에서 그림자 변수 선언이 빠지면 `var()`가 조용히 무시되어 **그림자가 사라진다**.
    `spaceTokens()`도 같은 이유로 이미지를 더했다 (`pad`가 공간 토큰을 받는다).

- **검증**: `gofmt -l .`(무출력), `go vet ./...`, **`make test DOCKER=finch` 전부 초록**
  (컨테이너가 진짜 게이트다 — 픽스처 골든 테스트까지 통과했으므로 **웹 골든 4개는 한 바이트도
  움직이지 않았다**). 호스트 `go test ./...`에서는 `01_text`~`04_shape` 골든의 `a:ext cx`만
  어긋나는데 05/01·05/02가 기록한 폰트 메트릭 차이와 같은 diff다 (텍스트 폭뿐이고 이 항목의
  산출물은 골든에 없다).

  브라우저로 **직접 잰 수치**(이 항목의 결정 근거다. `/tmp`에 최소 페이지를 띄워 재고,
  같은 결론을 `render/geom`의 브라우저 테스트가 산출물로 다시 확인한다):
  - **`max-*`는 위로 키우지 않는다.** 고유 크기 16×8 그림을 560×420 박스에 `contain`으로
    넣으면 잰 rect가 **16×8**로 남는다 (`object-fit: contain`이면 560×280이 된다). 그래서 이
    엔진의 `contain`은 "박스를 넘지 않는다"이고 "박스를 채운다"가 아니다.
  - **확대와 "박스 = 그림"을 CSS로 함께 얻을 수 없다.** `aspect-ratio: W/H` + `width:100%` +
    `max-height:100%`로 확대를 열어 보면 세로로 긴 그림에서 **비율이 깨진다**: 280×210 박스에
    8×16 그림이 280×210으로 늘어난다 (폭이 확정된 요소에서 `max-height`의 자름이 폭으로
    되돌지 않는다). `width:100%; height:auto; max-height:100%`도 같다. 좌표가 한 곳(브라우저)
    에서 오는 쪽을 골랐다.
  - **박스보다 큰 그림은 비율을 지키며 줄어든다** (`contain`이 실제로 맞추는 경로): 4:3
    560×420 박스에서 400×1200 그림은 140×420, 1200×400 그림은 560×186.66이다.
  - **래퍼의 자동 최소 크기가 박스를 키운다.** `min-*: 0`이 없으면 `fit="cover" w="fill"
    ratio="4:3"` 칸에 8×16 그림이 들어갔을 때 박스가 558×418.5가 아니라 **558×1116**이 되고
    (그림의 `height:100%`가 순환을 만나 auto로 풀리고 그 높이가 박스의 최소 높이가 된다) 잰
    rect가 슬라이드 밖으로 나간다. 픽스처 001_fit에서 실제로 드러났고
    `TestMeasureImageRatioBoxIsNotGrownByPicture`가 그 구조를 그대로 회귀 테스트로 받는다.
  - **디코드 전 rect는 0×0이다** (`naturalWidth === 0`). 대기의 근거가 계약만이 아니라
    **관측**이라는 뜻이다 — 지연 응답 서버로 재현했고, `contain`에서는 그 0이 그대로 잰 값이
    된다. 404는 `complete`가 true인 채 고유 크기가 0으로 남으므로 판정을 `decode()`의 거부가
    아니라 `naturalWidth === 0`에 두었다.
  - `pad="20"`인 240×120 cover 칸의 잰 rect는 **200×80**이고 위치가 20px 안쪽이다 —
    **PPTX가 `xfrm`에서 여백을 다시 뺄 필요가 없다** (항목 01의 열린 질문 (a)를 이 수치가 닫는
    쪽으로 기울인다: 여백은 웹에서 박스가 흡수하고 측정값은 이미 그림이다).

  손으로 확인한 것: `go run ./cmd/balpyo build --project testdata/fixtures/_05_image
  --target html`이 **`index.html` + `assets/{photo.jpg,tall.png,wide.png}`** 네 파일을 내고,
  `cmp`로 세 에셋이 커밋된 바이트와 동일하다. 산출물을 브라우저로 열어 슬라이드를 찍어 보면
  001_fit의 cover 두 칸이 각각 좌우·위아래가 잘린 그림으로 나온다 (크롭 축이 뒤바뀌지 않았다).

- **남긴 것**:
  - 항목 04가 이어받을 표면: 잰 rect가 **그려진 그림**이므로 `pic`의 `xfrm`은 측정값 그대로다
    (여백·letterbox 계산이 없다). `fit`은 `blipFill`에서 `cover`만 `srcRect` 크롭이 필요하고
    `contain`은 **크롭 없이 채우면 된다**(박스가 이미 그림 비율이다). 외관은
    `soml.ImageStyle` + `deck.Assets`에서 나오고, 웹이 쓴 값과 대조할 자리는
    `TestImageAndShapeShareAppearanceValues`의 표다.
  - 항목 05가 이어받을 것: 픽스처는 여전히 `_` 접두다. PPTX 렌더러는 아직 `<image>`에서
    "매핑이 없다"로 멈춘다 (`--target pptx` 빌드는 실패한다).
  - **열린 질문 4건**:
    (a) **`contain`이 확대하지 않는 것** (위 실측). 픽스처 에셋이 16×8이라 002_appearance의
    그림이 전부 16×8 점으로 나오고, 그 크기에서는 테두리·반경·그림자의 픽셀 비교가 뜻을 갖지
    못한다 (스크린샷으로 확인했다: 회전만 눈에 보인다). **항목 05가 에셋을 박스보다 크게 다시
    만드는 것**이 가장 값싼 해결이고(실제 덱의 사진은 언제나 박스보다 크다), 그것으로도 부족하다고
    판단되면 `contain`을 `object-fit: contain`으로 바꾸는 길이 남아 있다 — 다만 그것은 PPTX가
    letterbox를 다시 계산한다는 뜻이고(좌표 계산이 두 곳에 생긴다) 항목 03이 버린 쪽이다.
    ADR(항목 06)에 이 저울질을 그대로 적어야 한다: 지금의 `fit="contain"`은 CSS
    `object-fit: scale-down`에 가깝다.
    (b) **CSS 테두리는 안쪽에 그려진다.** `box-sizing: border-box`이므로 잰 rect는 테두리를
    포함하고 그림 영역은 그만큼 작다. PPTX `pic`의 `a:ln`은 도형 경계 **중앙**이라 그림이
    테두리 굵기만큼 크게 그려진다 — 1~3px 규모의 갈라짐이고, 항목 04가 `xfrm`을 테두리만큼
    줄일지 픽셀 비교의 임계값으로 흡수할지 정한다 (도형은 SVG 선이 경로 중앙이라 이 문제가
    없었다).
    (c) **파선 간격** (위 결정). 두 타깃의 파선이 갈라지는 것을 픽셀 비교가 어떻게 볼지는
    항목 05에서 처음 드러난다.
    (d) **`.balpyo-shape`에는 `flex: 0 0 auto`와 `min-*: 0`이 없다.** 그림에서 필요했던 이유
    (`ratio` 박스가 내용 때문에 커진다)가 도형에도 걸릴 수 있다 — 도형의 svg는 절대 배치라
    자동 최소 크기에 기여하지 않아 지금은 드러나지 않지만, 도형 안 텍스트가 박스보다 크면
    같은 모양의 실패가 된다. 이 항목의 범위 밖이라 손대지 않았다 (골든 4개가 움직인다).

## Iteration 5 · 04-pptx-image — PPTX: `p:pic` + 미디어 파트 + 관계 파일
<!-- item: 05/04-pptx-image -->

- **한 것**:
  - `render/pptx/image.go` (신규) — `<image>` → `p:pic`. `writePicShape`(관문 + 에셋 조회),
    `picInfo`·`picSpec`(확정 외관 → 명세, PRD §9.4 표 순서), `writePic`(골격: nvPicPr →
    blipFill → spPr), `srcRect`+`coverCrop`(cover 크롭), 미디어 파트의 이름·목표·조회
    (`mediaPart`·`mediaTarget`·`mediaOf`·`mediaParts`), 콘텐츠 타입(`mediaDefault`·
    `mediaDefaults`·`mediaContentType`), 관계 타입 상수 3개, `roundRectPreset`.
  - `render/pptx/shape.go` — **`slideLinks` → `slideRels`**: 하이퍼링크와 미디어가 한
    카운터(`slideRelBase`)를 문서 순서로 나눠 쓴다 (`relID`·`linkID`·`mediaID`, 목적지 중복은
    관계 하나). `writeShapeNode`에 `KindImage` 갈래(다섯 번째 분기), `xfrm` 타입 +
    `transformOf`(회전·대칭 조립을 도형과 공유 — `xfrmAttrs`가 메서드가 되었다), `writeGeom`
    (prstGeom/avLst 조립을 도형과 공유), `shapeSp`의 회전·대칭 블록이 `transformOf` 호출로 줄었다.
  - `render/pptx/parts.go` — `buildParts`가 미디어 파트를 붙이고(테마와 같은 층, 슬라이드
    앞), Content_Types 템플릿이 `MediaDefaults`를 돌고, `slideRelsTemplate`이 종류를 모르는
    관계 목록(`Rels`)을 돈다 (`slideLinkRel` 삭제 — `slideRel`이 그 자리다).
  - `render/pptx/pptx.go` — 패키지 주석에 `<image>` → `p:pic` + 미디어 파트 행.
  - `render/web/shape.go` — `RoundRectAdjust()` 내보냄. 기하 표의 `rounded-rect` 행이 그것을
    쓰고, 그림의 `radius`도 **같은 명세**로 조정값을 되돌린다.
  - `soml/asset.go` — `ImageFormat.Ext()` (포맷 → 첫 확장자). `core/asset.go`의
    `extForFormat`을 지우고 그 메서드를 부른다 — 같은 표가 두 벌이 되지 않게.
  - `internal/testkit/fixture_level_test.go` — 도형 수 등식이 `countedShapeTags`
    (`p:sp` + `p:pic`)로 센다(`countShapes`, 슬라이드별 표도 같은 함수), `shapeRoles`에
    `KindImage` 갈래(잰 rect 그대로).
  - `render/pptx/image_test.go` (신규) — `p:pic` 전문(스키마 순서)·`alt`/`descr`·`fit` 6종·
    `coverCrop` 표 6행 + 대칭·한 축·상한·에러·외관 16행(어휘 커버리지 대조)·정하지 않은 외관·
    불투명도가 선·그림자까지 가는지·프리셋 이름 대조·**삼각 대조**(파트/관계/콘텐츠 타입)·
    관계 번호(문서 순서, rId2 예약, 재사용)·그림 없는 덱·`.jpg`/`.jpeg` 한 Default·결정성·
    렌더러 관문 7종·픽스처 에셋 존재.
  - `TASKS.md` — Iteration 5의 **구현** 체크박스만 `[x]` (에셋 해석 = 항목 02, 웹 = 항목 03,
    `pic` + rels = 이 항목으로 "이미지 파이프라인"이 닫혔다). 테스트 체크박스는 건드리지
    않았다 — "`srcRect` 크롭 등가성"의 자동 판정은 항목 05의 픽셀 비교다.

- **설계 결정과 근거**:
  - **관계 번호를 링크와 그림이 한 카운터로 나눠 쓴다** (항목 지시와 갈라지는 자리). 지시는
    "이미지의 첫 id = `hyperlinkRelBase` + 그 슬라이드의 하이퍼링크 수"였는데, 그 값은 **첫
    그림을 쓰는 순간 그 슬라이드의 총 링크 수를 이미 알아야** 나온다 — `r:embed`는 XML을 쓰는
    중에 확정되므로 슬라이드를 두 번 걸어야 한다는 뜻이고, 그것이 지시가 막으려던 실패(번호를
    두 곳에서 계산한다)의 다른 얼굴이다. 한 카운터로 문서 순서대로 배정하면 그 어긋남을 만들
    방법 자체가 없고, **링크만 있는 덱의 산출 바이트는 그대로다**(rId3부터 같은 순서로
    배정된다 — 기존 골든이 한 바이트도 움직이지 않은 것이 그 확인이다). 대가는 링크 번호가
    그림 유무에 따라 밀린다는 것인데, rId는 슬라이드 안에서만 뜻이 있고 재저장하면 뷰어가
    다시 배정하는 값이라 저자가 보는 것이 아니다.
  - **미디어 파트 이름의 확장자는 포맷에서 나온다** (`image1.png`·`image3.jpg` — 저자가 쓴
    확장자가 아니다). 항목 02의 권고를 따랐다: `[Content_Types]`의 `Default`가 **확장자로**
    콘텐츠 타입을 정하므로, 저자 표기를 그대로 쓰면 `.jpg`·`.jpeg`·`.JPG`가 섞인 덱에서
    Default가 여러 개 필요해지고 **대소문자 매칭을 소비자의 관용에 맡기게 된다**. 포맷에서
    지으면 패키지 안의 확장자가 한 가지로 모이고 Default도 포맷당 하나다 (항목 지시의 "둘 다
    나올 수 있다"는 그래서 이 구현에서 성립하지 않고, 그 사실을 테스트가 손으로 만든 표
    (`a.jpg` + `b.jpeg`)로 못 박는다). `Asset.Key`가 대소문자를 눕히지 않는 것과 짝이다 —
    키는 디스크를 가리키고 파트 이름은 패키지 안의 이름이다.
  - **파트 번호는 덱 전체의 에셋 표 순서**이고 표 **전부**를 낸다 (어느 슬라이드가 무엇을 쓰는지
    보지 않는다). 표는 참조에서 만들어지므로 쓰이지 않는 항목이 없고, 번호가 표의 자리에서
    나오면 슬라이드를 지웠을 때 다른 슬라이드의 관계가 엉뚱한 파트를 가리키는 일이 없다.
  - **이름은 `Picture 1`이 아니라 `image s0.0`이다** (항목 지시와 갈라지는 두 번째 자리).
    지시의 요구는 결정성이고 저장소의 기존 규약(`cNvPr@name` = 지오메트리 식별자 —
    `writeSp`)도 결정적이면서 **진단 메시지와 같은 이름**을 준다. 규약을 그림에만 다르게 두면
    PowerPoint 선택 창에서 어느 요소가 어느 태그였는지 두 규칙으로 읽어야 한다.
  - **`descr`은 `alt`이 있을 때만 낸다.** 항목 01의 열린 질문 (b)를 닫았다: 웹은 `alt=""`를
    언제나 내지만(`<img>`에서 속성 없음 ≠ 빈 문자열, 빈 문자열이 "장식 이미지"의 표기) OOXML에는
    그 구별이 없다 — `descr=""`는 "대체 텍스트가 비었다"이고 장식이라는 뜻이 아니며(PowerPoint의
    "장식용으로 표시"는 확장 요소다), 빈 속성은 재저장 diff의 잡음이 된다. 그래서 이 비대칭은
    두 포맷의 차이이고 두 타깃의 갈라짐이 아니다.
  - **`picLocks noChangeAspect`를 넣지 않았다** (지시의 권고와 같다). 산출물은 편집 가능해야
    하고, 저자가 아니라 엔진이 잠근 제약은 파일을 열어 본 사람이 이유를 알 수 없다.
  - **`spPr`에 채움을 내지 않는다** — 도형과 정면으로 갈라지는 자리다. 도형은 `noFill`을
    명시해야 테마 `fmtScheme`의 기본 채움이 상속되지 않지만, 그림의 채움은 `p:blipFill`이
    이미 소유한다: spPr에 또 적으면 같은 요소의 채움이 두 곳에서 선언되고 어느 쪽이 이기는지는
    소비자마다 다르다 (PowerPoint가 내는 `p:pic`에도 그 자리가 비어 있다). **선은 반대로
    명시한다** (`a:ln`) — 선은 blipFill이 소유하지 않으므로 비워 두면 상속의 여지가 남는다.
  - **테두리를 정하지 않은 그림(nil)은 `border="none"`과 같은 산출이다.** 이미지 컴포넌트
    기본값이 비어 있으므로(05/02의 결정) 확정 외관의 `Border`가 nil일 수 있는데, 도형처럼
    "해석되지 않았다"로 멈추면 테두리 없는 그림을 아예 그릴 수 없다. 둘 다 "선 없음"이라는
    같은 그림이므로 산출도 같아야 한다.
  - **`opacity` = `a:alphaModFix`, 그리고 선·그림자 색에도 같은 알파를 싣는다.** ADR 0018의
    후속이다: 도형에서 `alphaModFix`를 버린 근거는 그것이 도형 안 **글자까지** 흐리게 한다는
    것이었고 그림에는 글자가 없다. 선·그림자에 함께 싣는 근거는 웹의 합성이다 — CSS `opacity`는
    요소 전체(테두리 포함)를 흐리게 하고 `filter: drop-shadow`의 결과도 함께 흐려지므로, 그림만
    흐려지면 두 타깃이 갈라진다 (그림자 토큰 알파와 곱하는 것은 도형의 `shadowAlpha` 그대로다).
  - **`radius` → `roundRect` + 조정값, 명세는 도형과 공유**(`web.RoundRectAdjust`). 값이
    두 곳에 적히면 같은 `radius="md"`가 도형과 그림에서 다른 모서리로 그려지고, 저자는 그 차이를
    둘을 나란히 놓고서야 본다. 프리셋 **제약**은 공유하지 않는다 — 그림에서 반경은 언제나 유효하다
    (ADR 0019의 적용 범위가 도형이라는 05/02의 결정).
  - **cover 크롭은 양변을 같은 값으로 반올림한다** (잔차를 한 변에 몰지 않는다). 1000분의 %는
    픽셀보다 거칠어 보이는 몫이 정확한 값과 최대 1단위(0.001% = 1280px에서 0.013px) 어긋나는데,
    한 변에 잔차를 몰면 크롭이 비대칭이 되어 "cover는 중앙을 보여준다"는 계약이 **눈에 보이지 않게**
    깨진다. 크기 0 박스는 크롭 0이고(그릴 것이 없다), 고유 크기 0은 에러다(0으로 나눈 값이
    `srcRect`에 실리면 파일이 열리지 않는다 — 값 층이 이미 막지만 마지막 관문이다).
  - **`cover`인데 박스 크기가 없는 그림에 특례를 두지 않았다.** 웹은 그때 `cover`의 CSS를 내지
    않는데(`boxHasSize`), 이쪽은 **비율 비교가 스스로 그 경우를 처리한다**: 크기 속성이 없으면
    잰 rect가 고유 비율을 그대로 유지하므로(`max-*`는 비율을 지키며 줄인다) 크롭이 0이 된다.
    조건을 한 번 더 적으면 웹의 판정과 이쪽의 판정이 갈라질 자리가 생긴다.
  - **`xfrm`은 잰 rect 그대로다 — 테두리만큼 줄이지 않았다** (05/03의 열린 질문 (b)를 이렇게
    닫는다). CSS 테두리는 박스 **안쪽**에 그려지고(`box-sizing: border-box`) OOXML `a:ln`은
    도형 경계 **중앙**이라, 보정하지 않으면 PPTX의 그림이 테두리 굵기만큼(1~3px) 크게 그려진다.
    보정하지 않은 근거는 이 엔진의 가장 강한 계약이다: **잰 rect가 그대로 `xfrm`이 된다**
    (`TestFixtureCoordinatesMatchGeometry`가 0 오차로 못 박는다). 보정을 넣으면 렌더러가 좌표
    계산을 갖기 시작하고, 그 계산은 웹의 박스 모델을 렌더러가 다시 아는 것이다. **필요해지면
    그 방법을 여기 적어 둔다**: rect를 굵기의 **절반**만큼 줄이면 선이 `[rect−w, rect]`를 덮어
    CSS의 테두리 띠와 정확히 겹치고(전부 줄이면 바깥에 빈 띠가 남는다) 오차가 절반으로 줄어든다.
    판정은 항목 05의 픽셀 비교가 낸다.
  - **조립 함수를 도형과 공유한 범위**: 좌표(Offset·Extent)·회전·대칭(`transformOf`)·프리셋
    기하(`writeGeom`)·선(`writeLine`)·그림자(`writeEffect`)·색(`writeColor`). 그림에만 있는
    것은 셋뿐이다 — `blipFill`·`srcRect`·`alphaModFix`. 공유를 위해 `spSpec`의 회전·대칭
    필드를 `xfrm` 임베드로 바꿨는데 승격 필드라 기존 대입(`spec.rot`)은 그대로 산다.

- **검증**: `gofmt -l .`(무출력), `go vet ./...`, **`make test DOCKER=finch` 전부 초록**
  (컨테이너가 진짜 게이트다 — 픽스처 골든 테스트까지 통과했으므로 **골든은 한 바이트도
  움직이지 않았다**: 관계 배정을 일반화하고 `xfrmAttrs`·`prstGeom` 조립을 함수로 뺐지만
  기존 덱의 산출 바이트가 같다는 실물 확인이다). 호스트 `go test ./...`에서는
  `01_text`~`04_shape` 골든의 텍스트 `a:ext`만 어긋나는데 05/01~05/03이 기록한 폰트 메트릭
  차이와 같은 diff다.

  **픽스처 하나짜리 임시 빌드**(컨테이너 안, `_05_image` → pptx + html):
  - 패키지에 `ppt/media/image1.png`(97B = wide.png), `image2.png`(98B = tall.png),
    `image3.jpg`(652B = photo.jpg)가 들었고 바이트가 커밋된 파일과 같다. Content_Types에
    `Default Extension="png"`·`"jpg"` 둘뿐이다 (덱에 든 포맷만).
  - 슬라이드 4(그림 + 링크 + 노트)의 rels가 `rId1` 레이아웃, `rId2` 노트, `rId3` 이미지,
    `rId4` 하이퍼링크(`TargetMode="External"`)다 — 링크와 그림이 같은 슬라이드에 있고 노트가
    `rId2`를 쓰는, 배정이 어긋날 유일한 자리가 실물에서 겹치지 않는다.
  - **LibreOffice가 경고 없이 열었다**: `soffice --convert-to pdf`의 stderr에 `javaldx`
    한 줄뿐이다 (testkit의 무해 목록에 있는 문구 — 복구·수리 메시지가 없다). pdftoppm으로 뽑은
    6장에 그림이 전부 그려져 있다.
  - **cover 크롭이 두 타깃에서 같은 영역을 보여준다** (TASKS의 리뷰 포인트를 눈으로 확인한
    것이고, 자동 판정은 항목 05다): 같은 슬라이드를 chromium 스크린샷과 LibreOffice 렌더로
    나란히 보면 `wide.png`의 좌상단 표식이 **같은 자리에서 끊긴다**(경계가 두 장에서 같은
    x·y에 있다). 차이는 스케일러뿐이다 — 웹은 부드럽게, LibreOffice는 최근접으로 늘린다.
  - 픽스처를 **임시로** 스위트에 넣어(`fixtureNames`에 한 줄 + 디렉토리 복사, 확인 후 되돌렸다)
    두 픽스처 수준 계약이 그림에서도 성립하는 것을 확인했다: `TestFixtureCoordinatesMatchGeometry`
    가 05_image에서 통과(그림의 `xfrm` = 잰 값의 EMU, 0 오차)하고, 도형 수 등식이
    `측정 박스 54 − 컨테이너 15 − spacer 1 = 도형 38`로 맞는다 (`p:sp` + `p:pic`을 함께 센다).

- **남긴 것**:
  - 항목 05가 이어받을 것: 픽스처는 여전히 `_` 접두다. 이제 `--target pptx`가 통하므로 켜는
    일은 골든 생성(**바이너리 파트가 골든에 처음 들어간다** — `ppt/media/*`)과 픽셀 비교
    임계값이다. 위 임시 확인이 좌표·도형 수 계약은 이미 통과한다는 것을 말해 준다.
    05/03의 열린 질문 (a)(`contain`이 확대하지 않아 002_appearance의 그림이 16×8 점으로 나온다)는
    그대로 남아 있다 — LibreOffice 렌더에서도 같은 크기이므로 **에셋을 박스보다 크게 다시 만드는**
    쪽이 여전히 가장 값싼 해결이다.
  - 항목 06이 이어받을 것: ADR 두 건의 후보가 이 항목에서 실물이 되었다 — `opacity`가 그림에서는
    `alphaModFix`라는 것(ADR 0018의 적용 범위)과 미디어 파트 이름·Content_Types Default가
    **포맷**에서 나온다는 것. `docs/reference/appearance.md`의 PPTX 열에 이미지 행 7개가 필요하고,
    `docs/internals/testing.md`의 도형 수 등식 설명에 `p:pic`이 들어간다.
  - **열린 질문 4건**:
    (a) **알파 PNG 뒤의 채움을 관찰하지 못했다.** `spPr`에 채움을 내지 않기로 했으므로, 소비자가
    `p:pic`에 테마 기본 채움을 상속시킨다면 투명 배경 PNG의 뒤에 색이 보인다 (웹의 `<img>`는
    배경이 없다). 픽스처 에셋이 전부 불투명해서 확인할 수 없었다 — 항목 05가 알파 PNG를 넣으면
    그 자리에서 드러나고, 드러나면 `a:noFill`을 명시하는 것이 답이다 (선을 명시한 것과 같은 근거).
    (b) **테두리 반칸 보정** (위 결정). 항목 05의 픽셀 비교가 임계값으로 흡수하지 못하면 위에 적은
    "굵기의 절반만큼 줄인다"를 넣는다 — 그때 `pptx.TextBoxRect`처럼 변환 함수를 내보내야
    `TestFixtureCoordinatesMatchGeometry`가 같은 기대값을 만든다.
    (c) **`.jpeg` 원본을 실물로 확인하지 못했다.** 픽스처에 `.jpeg` 파일이 없어 "두 확장자가 한
    Default로 모인다"를 손으로 만든 표로만 단정했다. 항목 05가 픽스처에 `.jpeg` 한 장을 더하면
    실물 확인이 되고, 그 자리에서 `Asset.Key`(원본 이름)와 파트 이름이 갈라지는 것이 처음 보인다.
    (d) **재저장 diff를 확인하지 못했다** (PowerPoint가 없다). 관계 번호·파트 이름·`descr` 생략을
    "PowerPoint가 내는 표기"에 맞췄지만 근거는 문서와 관례이고 실측이 아니다 — TASKS의 리뷰 항목
    (PowerPoint에서 열어 그림을 바꿔 보기)이 그것을 볼 자리다.
