# `<image>` — 프로젝트 안의 그림 하나

그림 하나다. `src`가 **어느 파일인가**를 정하고, `fit`이 박스와 그림이 어긋날 때 무엇을
보여줄지 정하며, 외관 속성이 테두리·모서리·그림자·투명도·변형을 정한다.

```html
<image src="assets/photo.jpg" fit="cover" w="fill" h="360" alt="전면 그림"/>
```

**void 태그다** — 자식이 없다. 캡션은 자식이 아니라 **형제 `<text>`**이고(아래 예제), 그림 안
글자는 [도형](shape.md)의 일, 그림 위 글자는 겹침([`<stack>`](stack.md))의 일이다.

이 페이지에는 다른 태그 페이지에 없는 절이 하나 있다 — **`src` 해석 규칙**이다. `<image>`는
값이 파일을 가리키는 첫 태그이고, 그래서 빌드가 프로젝트 디렉토리 밖의 무언가에 의존할 수
있는 첫 자리다.

## `src` — 프로젝트 루트 기준 상대 경로

기준은 **프로젝트 루트**다 (`balpyo.json`이 있는 디렉토리). 슬라이드 파일 기준이 아니다:
하위 디렉토리가 챕터가 되므로 슬라이드 기준이면 슬라이드를 챕터로 옮길 때 같은
그림의 경로가 파일마다 달라지고, 슬라이드 이동은 이 제품이 값싸다고 약속한 조작이다.

`assets/` 아래일 것은 **강제하지 않는다.** [개요](../../index.md)의 프로젝트 구조는 관례이고, 프로젝트
안이면 어디든 좋다 (`img/ch1/photo.jpg`도 유효하다).

값은 **정규화되어 에셋 표의 키**가 된다 (`path.Clean`):

| 저자가 쓴 것 | 표의 키 |
|---|---|
| `assets/wide.png` | `assets/wide.png` |
| `./assets/wide.png` | `assets/wide.png` |
| `assets//wide.png` | `assets/wide.png` |
| `assets/deep/../wide.png` | `assets/wide.png` |
| `img/ch1/Photo.JPG` | `img/ch1/Photo.JPG` — **대소문자는 눕히지 않는다** |

정규화가 필요한 이유는 산출물이다: `./assets/a.png`와 `assets/a.png`가 두 항목이 되면 같은
바이트가 두 번 나가고 PPTX 미디어 파트가 둘이 된다. 반대로 **대소문자를 눕히지 않는** 이유도
산출물이다 — 이 키로 파일을 다시 읽으므로(웹이 산출 디렉토리로 복사한다) `A.PNG`를 `a.png`로
바꾸면 대소문자를 구별하는 파일 시스템에서 그 파일이 사라진다. 확장자 대소문자에 걸린 판단
(PPTX 콘텐츠 타입·미디어 파트 이름)은 경로가 아니라 **포맷**이 소유한다 (아래 렌더 결과).

### 막는 것과 그 이유

| 표기 | 왜 막는가 |
|---|---|
| `/Users/me/a.png` (절대 경로) | 빌드를 이 기계에 묶는다 — 같은 덱이 CI에서 다르게 빌드된다 |
| `../shared/a.png` (프로젝트 밖) | 같은 이유. 에셋은 프로젝트 안에 둔다 (덱 하나가 자기 그림을 전부 갖는다) |
| `https://…/a.png` (URL) | 빌드가 네트워크에 의존하면 **오프라인 발표 약속**이 깨진다 |
| `data:image/png;base64,…` | 원문에 바이트를 실으면 슬라이드를 사람이 읽을 수도, 진단이 자리를 가리킬 수도 없다 |
| `assets\wide.png` (역슬래시) | 조용히 `/`로 바꾸면 역슬래시를 파일 이름에 쓸 수 있는 파일 시스템에서 **다른 파일**을 가리킨다 |
| `assets/` (디렉토리) | 그림 파일 하나를 가리켜야 한다 |

넷을 한 문장으로 묶으면 이렇다: **빌드는 프로젝트 디렉토리와 커밋된 바이트만 보고 결정된다.**
그것이 같은 덱이 어느 기계에서나 같은 산출물을 내는 근거이고, 발표장에서 네트워크가 없어도
덱이 온전한 근거다.

### `theme:` — 테마가 가진 그림

`theme:`으로 시작하는 값은 프로젝트의 파일이 아니라 **활성 테마의 그림**이다:

```xml
<image src="theme:icons/compute/lambda.png" w="64" alt="Lambda"/>
```

- 경로는 테마 패키지의 `assets/` 기준이다 — `theme:icons/a.png`는 `<테마>/assets/icons/a.png`다.
- 찾는 자리는 활성 테마와 그 테마가 `extends`로 물려받은 테마들이다. 같은 경로의 파일이 둘 다에 있으면
  **자식의 파일**이 이긴다 ([테마](../theme.md)의 `assets/`와 같은 파일 단위 병합).
- 테마가 어디서 왔는지는 상관없다 — 내장 테마, 프로젝트의 `themes/<이름>/`, `balpyo theme install`로
  설치한 테마가 모두 같다. 설치한 테마의 그림을 쓰려고 프로젝트에 `themes/`를 둘 필요가 없다.

슬라이드가 테마의 이름도 자리도 적지 않으므로, `balpyo.json`의 `theme`만 바꿔 같은 덱을 다른 테마로 빌드할
수 있다. 그 테마의 체인에 파일이 없으면 **빌드 에러**다 (빈 그림으로 빌드되지 않는다) — 진단이 찾아 본 테마를
나열한다.

경로 규칙은 프로젝트 경로와 같다: 절대 경로 · `assets/` 밖(`..`) · 역슬래시 · 열지 않은 확장자를 막고, 값은
정규화된다 (`theme:./icons/a.png` = `theme:icons/a.png`). 내용 판정도 같다 — 확장자와 내용이 다른 파일은
에러다. 산출물에서 테마의 그림은 그 파일을 가진 테마의 자리(`themes/<테마>/assets/…`)에 놓이고, 쓴 그림만
나간다. 테마의 배경 · 로고와 같은 파일이면 산출물에 한 번 들어간다 (PPTX 미디어 파트도 하나다).

### 포맷 2종 — PNG·JPEG

쓸 수 있는 확장자는 `.png` · `.jpg` · `.jpeg` 셋이고 **포맷은 둘**이다 (PNG·JPEG).

어휘를 둘로 닫은 기준은 [`<shape type>`](shape.md)을 OOXML 프리셋으로 닫은 것과 같다 —
**양 타깃이 그리는 것만 연다.** 진단이 확장자마다 그 이유를 말한다:

| 확장자 | 왜 없는가 |
|---|---|
| `.svg` | PPTX는 벡터를 확장 요소(`svgBlip`) + 래스터 폴백으로 실어 소비자마다 다르게 열린다 — 벡터 그림은 PNG로 내보내 쓴다 |
| `.gif` · `.apng` | 애니메이션에 양 타깃의 대응이 없다 |
| `.webp` · `.avif` · `.heic` | PowerPoint 버전마다 지원이 갈려 **열리지 않는 기계**가 생긴다 |
| `.bmp` · `.tif` | 브라우저의 지원이 갈린다 |

**확장자는 선언이고 내용이 판정이다.** 엔진은 파일 헤더를 읽어 실제 포맷을 확인하고, 확장자만
바꾼 파일을 빌드 에러로 막는다 — 통과시키면 산출물은 만들어지고 PowerPoint만 열지 못한다
(`[Content_Types].xml`이 **확장자로** 콘텐츠 타입을 정하기 때문이다).

## 속성

### 고유 속성

| 속성 | 값 | 기본값 | 뜻 |
|---|---|---|---|
| `src` | 프로젝트 루트 기준 경로, 또는 `theme:<경로>` | **필수** | 그림 파일 |
| `id` | 임의 이름 | 없음 | [연결선](connector.md)이 가리키는 이름 — 도형의 `id`와 같은 표에서 슬라이드 안 유일하다 |
| `fit` | `cover` `contain` | `contain` | 박스와 그림이 어긋날 때 (아래) |
| `alt` | 임의 문자열 (빈 값 허용) | 없음 ≡ `alt=""` | 대체 텍스트 |

- **`src`는 생략할 수 없다.** 그림의 경로는 추측할 수 없고, 빈 그림을 그리는 것은 저자의
  오타를 산출물에 통과시키는 일이다.
- **`alt`은 양 타깃에 자리가 있다** — 웹 `<img alt>`, PPTX `p:cNvPr@descr`(PowerPoint의
  "대체 텍스트" 칸). 자리가 있는데 열지 않으면 접근성 없는 그림이 나가고 저자가 고칠 방법이
  없다. `alt=""`은 **에러가 아니다**: `<img>`에서 속성 없음과 빈 문자열은 스크린 리더에 다른
  뜻이고 빈 문자열이 "장식 이미지"의 표기다 (생략과 `alt=""`는 같은 확정 값이 된다). 값
  **없는** `alt`은 여전히 에러다 — 뜻을 정할 수 없는 형태다.
- **`frame` 속성은 없다.** 액자가 뜻하는 묶음(테두리+반경+그림자)은 이미
  `variant`가 소유하는 채널이므로 같은 것을 두 이름으로 받지 않는다 —
  `<image variant="framed">`가 그 자리다.

### 외관 속성 7종

값 어휘와 캐스케이드는 [외관 속성](../appearance.md)이 소유한다. 여기서는 그림에서의 뜻만
적는다.

| 속성 | 값 | 그림에서의 뜻 |
|---|---|---|
| `variant` | 테마가 정의한 이름 | 외관 번들 — **이미지 어휘는 텍스트·도형과 별개 표다** (`minimal`은 `framed` 하나) |
| `border` | `<굵기> <스타일> <색>` · `none` | 외곽선 — 웹은 박스 **안쪽**, OOXML은 경계 **중앙**이다 (아래) |
| `radius` | radius 토큰 · 수치 | 모서리 반경 — 그림에서는 **언제나 유효하다** |
| `shadow` | shadow 토큰 | 그림자 — 그림의 **알파 실루엣**을 따라간다 |
| `opacity` | 0~1 실수 | 그림·테두리·그림자의 투명도 |
| `rotate` | 각도 수치 (음수 허용) | 시계방향 회전 |
| `flip` | `h` `v` `hv` | 대칭 |

**`fill`은 없다.** 그림 뒤의 칠이 보이는 자리는 `fit="contain"`의 여백뿐인데 그 여백은 PPTX
에서 `pic`의 박스 밖이다 — 받아 놓고 한 타깃에서만 그리는 속성이 된다. 그림 뒤에 칠이 필요하면
그것은 [도형](shape.md)의 일이다 (`<shape fill>` 위에 그림을 겹치려면 [`<stack>`](stack.md)을 쓴다).

**도형과 정반대인 자리가 둘 있다.**

- **테마가 그림에 아무것도 얹지 않는다.** 도형은 컴포넌트 기본값이 캐스케이드의 바닥에 있어
  `<shape type="rect"/>`도 채움·테두리·반경을 받지만, `<image src="…"/>`는 장식이 **하나도**
  없다. 그림은 자기 픽셀을 갖고 있어 장식 없이도 완전하고, 기본으로 테두리를 두르면
  스크린샷·도표에 이중 경계가 생겨 저자가 모든 그림에 `border="none"`을 적어야 한다. 액자가
  필요한 자리에는 `variant="framed"`가 있다.
- **`radius`에 프리셋 제약이 없다.** 도형에서 `radius`가 `rounded-rect` 전용인 이유는
  프리셋 기하에 반경 조정값 자리가 없다는 것이고, 그림은 프리셋이 아니라 **자기 클립**을
  갖는다 (웹 `border-radius`, PPTX `pic/spPr/prstGeom prst="roundRect"`).

두 가지는 도형과 같은 값이지만 컴파일되는 자리가 다르다.

- **`opacity`는 `a:alphaModFix`가 된다.** 도형에서 그것을 버린 근거는 `alphaModFix`가 도형 안
  **글자까지** 흐리게 한다는 것이었고, 그림에는 담을 글자가 없다 — 그래서 그림에서는 CSS
  `opacity`와 정확한 대응이다. 테두리·그림자 색에도 같은 알파가 실린다 (CSS `opacity`가 요소
  전체를 흐리게 하므로 그림만 흐려지면 두 타깃이 갈라진다).
- **테두리의 위치가 갈라진다.** CSS 테두리는 박스 **안쪽**에 그려지고(`box-sizing:
  border-box`) OOXML `a:ln`은 경계 **중앙**이라, PPTX의 그림이 테두리 굵기만큼 크게 그려진다.
  보정하지 않는 이유는 이 엔진의 가장 강한 계약이다 — **잰 rect가 그대로 `xfrm`이 된다.**
  1px 테두리에서 그 대가는 그림 면적의 4.9%다 — 획이 그려지는 모습이라 계약(자리 · 크기) 밖이다
  ([계약 밖의 차이](../../guides/known-differences.md)).
- **파선 간격은 UA가 정한다.** 도형은 `stroke-dasharray`로 OOXML `prstDash`의 비율을 정확히
  내지만 CSS `border-style: dashed`의 패턴은 브라우저가 정한다. 맞추려면 그림을 svg 안에 넣어야
  하고 그러면 `<img>`가 주는 것(고유 크기·`alt`·디코드 신호)을 전부 잃는다.

### 자기 박스 — 여백과 크기

| 속성 | 값 | CSS 대응 |
|---|---|---|
| `pad` | 공간 토큰 · 수치 (1~4값) | `padding` — **그림 영역이 여백만큼 줄어든다** |
| `w` / `h` | 수치 · `%` · `fill` | `width` / `height` |
| `min-w` / `min-h` · `max-w` / `max-h` | 수치 · `%` · `fill` | `min-*` / `max-*` |
| `ratio` | `W:H` 정수 쌍 | `aspect-ratio` |

**이 목록은 [레이아웃 컨테이너](layout.md)·[도형](shape.md)과 같은 한 벌이다** — 크기는
컨테이너나 도형의 성질이 아니라 박스의 성질이므로 세 요소가 같은 표를 공유한다. 값 어휘와
진단이 한 글자도 다르지 않다.

- **`ratio`가 그림에서 가장 흔한 표기다.** `fit`은 박스가 정해진 다음에만 뜻을 갖고(아래),
  `ratio="16:9"`가 박스의 비율을 그림의 고유 비율과 무관하게 못 박는 표기다.
- `pad`는 박스에 걸리고 그림 영역이 그만큼 준다. **PPTX가 여백을 다시 빼지 않는다** — 웹에서
  여백은 저자 박스가 흡수하고 잰 rect는 이미 그 안쪽의 그림이다 (`pad="20"`인 240×120 칸의 잰
  rect는 200×80이다).

### 자식 배치 속성

그림도 컨테이너의 자식이 될 수 있으므로
`grow`/`shrink`/`basis`/`span`/`self`/`order`/`style`을 받는다 ([레이아웃 태그](layout.md)의
자식 속성 표와 같은 한 벌). `<grid cols="4">` 안의 `<image span="2">`가 두 칸을 차지한다.
선언은 **박스**에 걸린다 — flex/grid의 항목은 저자가 준 박스이고 그 안의 그림이 아니다.

## `fit` — 박스와 그림이 어긋날 때

값은 둘이다. CSS `object-fit`의 `fill`·`none`·`scale-down`은 없다 — `none`은 `pic`의 박스 밖으로
그림이 넘치는 상태라 PPTX에 대응이 아예 없다. 늘리고 싶으면 `ratio`로 박스를 그림 비율에 맞춘다.

| `fit` | 웹 (CSS) | PPTX | 무엇이 잘리고 무엇이 남는가 |
|---|---|---|---|
| `cover` | `width: 100%` · `height: 100%` · `object-fit: cover` | `xfrm` = 잰 rect, `a:blipFill/a:srcRect`로 크롭 | **박스가 남고 그림이 잘린다** — 넘치는 축의 양쪽을 절반씩 (중앙이 보인다) |
| `contain` | `max-width: 100%` · `max-height: 100%` | `xfrm` = 잰 rect, 크롭 없음 | **그림 전체가 남고 박스가 그림에 맞춰진다** |

두 번째 행의 오른쪽 칸이 이 태그에서 가장 많은 것이 따라오는 사실이다:
**`fit="contain"`은 박스를 그림에 맞춘다.** `object-fit: contain`처럼 저자가 준 박스를 그대로
두고 그 안에 letterbox 여백을 두지 않는다. 저자가 배치를 짤 때 이것이 보여야 한다:

- **`<row>` 안의 `contain` 그림은 남는 폭을 형제에게 주지 않는다.** 박스가 그림 크기로
  줄어들지만 그 자리는 여전히 이 항목의 것이고, 여백은 flex 정렬이 만든 항목 **밖의** 공간이다.
  칸을 나누는 것은 `basis`·`grow`·`gap`이 하는 일이지 `fit`이 하는 일이 아니다.
- **테두리·모서리·그림자가 그림에 딱 붙는다.** 박스와 그림이 같으므로 웹의 `border`와 PPTX
  `pic`의 `a:ln`이 같은 자리에 온다 — `object-fit`을 쓰면 웹의 테두리는 여백을 포함한 박스에
  붙고 PPTX의 선은 그림에 붙어 두 타깃의 외관이 갈라진다.
- **PPTX가 letterbox를 다시 계산하지 않는다.** 좌표는 브라우저가 한 번만 계산한다는 이 엔진의
  전제가 그림에서도 그대로 성립한다.

**`contain`은 확대하지 않는다.** `max-*`는 위로 키우지 않으므로 고유 크기 16×8 그림은 240×120
박스 안에서 **16×8로 남는다** — 이 엔진의 `contain`은 CSS `object-fit: contain`이 아니라
`scale-down`에 가깝다. 확대를 열려면 CSS에 그림의 고유 비율을 실어야 하는데 그러면 세로로 긴
그림에서 비율이 깨진다.

**크기 속성이 없는 그림은 `fit`이 아무것도 하지 않는다.** 박스가 고유 크기이므로 자를 것도
맞출 것도 없고, 그때는 `cover`도 `contain`과 같은 선언을 낸다 — 그러지 않으면 `cover`의
`width: 100%`가 그림을 부모 폭까지 늘려 "크기 속성이 없으면 고유 크기"라는 계약이 `fit` 값에
따라 깨진다. `min-*`/`max-*`는 크기로 세지 않는다 (한계는 박스가 아니다).

## 고유 크기는 논리 스테이지 px다

파일의 픽셀 크기가 **그대로 논리 스테이지 px**다 (단위는 하나다). 2배 해상도 에셋은
두 배 크게 놓이고, 엔진은 DPR을 추측하지 않는다 — 저자가 확인할 수 없는 배율이 산출물에
섞이면 "좌표계는 하나"라는 계약이 무너진다.

그러므로 **배율을 정하는 표기는 `w`/`h`/`ratio`다.** 1200×600 사진을 절반 크기로 놓으려면
`w="600"`을 쓰고, 슬라이드 폭을 채우려면 `w="fill"`을 쓴다.

엔진은 **바이트를 고치지 않는다** — 리사이즈·재압축·EXIF 회전이 없다. 커밋한 파일과 산출물의
그림이 다르면 저자가 그 차이를 확인할 방법이 없기 때문이다. 그래서 큰 사진은 저자가 미리
줄여 커밋하는 것이 산출물 크기를 줄이는 유일한 길이다.

## 예제

`testdata/fixtures/05_image/slides/001_fit.slide` — `fit` 2종 × 넓은/좁은 그림. 박스는 전부
`ratio`로 못 박고 그 비율을 그림의 고유 비율과 다르게 둔다 (같으면 두 값이 같은 그림을 낸다):

```html
<slide id="image-fit">
  <h2>fit — cover와 contain</h2>
  <grid cols="4" gap="md">
    <col gap="xs">
      <image src="assets/wide.png" fit="cover" w="fill" ratio="4:3" alt="넓은 그림을 박스에 채운 것"/>
      <text role="caption">cover · 넓은 그림</text>
    </col>
    <col gap="xs">
      <image src="assets/tall.png" fit="cover" w="fill" ratio="4:3" alt="좁은 그림을 박스에 채운 것"/>
      <text role="caption">cover · 좁은 그림</text>
    </col>
    <col gap="xs">
      <image src="assets/wide.png" fit="contain" w="fill" ratio="4:3" alt="넓은 그림을 박스에 맞춘 것"/>
      <text role="caption">contain · 넓은 그림</text>
    </col>
    <col gap="xs">
      <image src="assets/tall.png" fit="contain" w="fill" ratio="4:3" alt="좁은 그림을 박스에 맞춘 것"/>
      <text role="caption">contain · 좁은 그림</text>
    </col>
  </grid>
  <row gap="sm" align="center">
    <image src="assets/dot.png" fit="contain" w="240" h="120" alt="고유 크기가 박스보다 작은 그림"/>
    <text role="caption">contain은 확대하지 않는다 — 고유 크기 16×8 그림은 240×120 박스 안에서 16×8로 남는다</text>
  </row>
</slide>
```

`002_appearance.slide` — 외관 6종 (각각 **한 속성만** 다르다):

```html
<row gap="md" align="center">
  <image src="assets/wide.png" fit="contain" border="1 solid muted" w="200" h="150" alt="테두리"/>
  <image src="assets/wide.png" fit="contain" radius="lg" w="200" h="150" alt="둥근 모서리"/>
  <image src="assets/wide.png" fit="contain" shadow="soft" w="200" h="150" alt="그림자"/>
</row>
<row gap="md" align="center">
  <image src="assets/wide.png" fit="contain" opacity="0.4" w="200" h="150" alt="반투명"/>
  <image src="assets/wide.png" fit="contain" rotate="15" w="200" h="150" alt="회전"/>
  <image src="assets/wide.png" fit="contain" flip="h" w="200" h="150" alt="좌우 대칭"/>
</row>
```

`007_variant.slide` — 액자는 속성이 아니라 테마의 번들이고, 인라인 `border`가 그것을 덮는다:

```html
<row gap="lg" align="center">
  <col gap="xs">
    <image src="assets/wide.png" fit="contain" w="240" h="180" alt="장식 없는 그림"/>
    <text role="caption">기본 — 테마가 얹지 않는다</text>
  </col>
  <col gap="xs">
    <image src="assets/wide.png" fit="contain" variant="framed" w="240" h="180" alt="액자를 씌운 그림"/>
    <text role="caption">framed — 액자 한 벌</text>
  </col>
  <col gap="xs">
    <image src="assets/wide.png" fit="contain" variant="framed" border="2 solid accent1" w="240" h="180" alt="액자에 테두리를 덮은 그림"/>
    <text role="caption">인라인 border가 이긴다</text>
  </col>
</row>
```

`004_image_text.slide` — 캡션이 **형제**라는 것과 그림이 컨테이너의 자식이라는 것:

```html
<row gap="xl" align="center">
  <col gap="xs" basis="520">
    <image src="assets/tall.png" fit="cover" w="fill" ratio="4:3" alt="구성도"/>
    <text role="caption">그림 1. 구성도 — 캡션은 그림의 자식이 아니라 형제다</text>
  </col>
  <col gap="sm">
    <h3>담을 것이 없는 태그</h3>
    <p>그림은 void다. 그림 안 글자는 도형의 일이고, 그림 위 글자는 겹침의 일이다.</p>
  </col>
</row>
```

`006_full_bleed.slide` — 슬라이드 직속 전면 그림 (`<spacer/>`가 꼬리글을 바닥으로 밀어낸다):

```html
<image src="assets/photo.jpg" fit="cover" w="fill" h="360" alt="전면 그림"/>
<spacer/>
<text role="caption">배율은 고유 크기가 아니라 w·h·ratio가 정한다 — 이 그림은 1152×360으로 놓였다.</text>
```

## 렌더 결과

### 웹 — 박스 div + `<img>` + 산출 파일

그림은 **`<div>` 하나와 그 안의 `<img>` 하나**다. 박스가 저자가 준 크기·여백·비율·배치를 갖고,
`<img>`가 `fit`과 외관을 갖는다:

```html
<div class="balpyo-image" style="width: 100%; aspect-ratio: 4 / 3">
<img class="balpyo-image-ink" data-balpyo-id="s0.1.0.0" data-balpyo-el="image"
     src="assets/wide.png" alt="넓은 그림을 박스에 채운 것"
     style="width: 100%; height: 100%; object-fit: cover">
</div>
```

- **식별자가 붙는 층이 도형과 반대다.** 도형은 겉 박스를 재고(그림 svg가 박스를 덮으므로 둘이
  같다) 그림은 `<img>`를 잰다 — `contain`에서 박스와 그림이 갈라지고, `pic`의 `xfrm`이 되어야
  하는 것은 **그림**이다.
- **`alt`은 언제나 나간다** (없으면 `alt=""`). 속성 자체가 없으면 스크린 리더가 파일 이름을
  읽는다.
- **에셋이 산출 파일이다.** `index.html` 외에 `assets/wide.png` 같은 파일이 함께 나오고, 경로는
  저자가 쓴 정규화 경로 **그대로**다 — `.slide`에 적힌 것과 산출 디렉토리의 이름이 같다.
  파이프라인이 나중에 복사하지 않고 렌더러가 내는 이유는 측정이다: 지오메트리 패스가 재는
  것이 이 산출물이고, 그림 바이트가 없으면 브라우저가 **깨진 이미지**(rect 0×0)를 잰다.
- `width`/`height` **HTML 속성은 내지 않는다** — 고유 크기를 속성으로 한 번 더 말하면 박스를
  정하는 곳이 둘이 된다.
- 회전·대칭은 하나의 `transform`을 나눠 쓴다 (엔진 CSS의 클래스 + 인라인 커스텀 프로퍼티).
  합성은 `rotate(…) scale(…)` — 오른쪽이 먼저 적용되므로 "대칭 후 회전"이고, OOXML
  `xfrm`(뒤집힌 기하를 `rot`으로 돌린다)과 같은 순서다. 선언이 엔진 CSS에 있는 이유는 측정
  뷰가 그것을 끌 수 있어야 하기 때문이다.
- 그림자는 `box-shadow`가 아니라 `filter: drop-shadow`다 — 그림의 **알파 실루엣**을 따라가므로
  투명 배경 PNG의 그림자가 사각형이 되지 않고 `border-radius`로 자른 모서리도 따라온다.

### PPTX — `p:pic` + 미디어 파트 + 관계

**세 자리가 맞아야 그림이 나온다.** 하나가 어긋나면 파일이 열리지 않거나 그림 자리가 빈 채로
열린다:

| 자리 | 무엇인가 |
|---|---|
| `ppt/media/image<N>.<ext>` | 바이트 — 파트가 처음으로 XML이 아니다 |
| `[Content_Types].xml`의 `Default` | 확장자 → 콘텐츠 타입 |
| `ppt/slides/_rels/slideN.xml.rels` | 관계 — `a:blip@r:embed`가 가리키는 목적지 |

```xml
<p:pic>
  <p:nvPicPr>
    <p:cNvPr id="3" name="image s0.1.0.0" descr="넓은 그림을 박스에 채운 것"/>
    <p:cNvPicPr/>
    <p:nvPr/>
  </p:nvPicPr>
  <p:blipFill>
    <a:blip r:embed="rId3"/>
    <a:srcRect l="16667" r="16667"/>   <!-- cover — 2:1 그림을 4:3 박스에 채우고 좌우를 잘랐다 -->
    <a:stretch><a:fillRect/></a:stretch>
  </p:blipFill>
  <p:spPr>
    <a:xfrm><a:off x="609600" y="1283940"/><a:ext cx="2571750" cy="1928813"/></a:xfrm>
    <a:prstGeom prst="rect"><a:avLst/></a:prstGeom>
    <a:ln><a:noFill/></a:ln>
  </p:spPr>
</p:pic>
```

- **미디어 파트는 덱 전체에서 에셋 하나당 하나다.** 번호는 에셋 표의 자리(첫 사용의 문서
  순서)이고, 같은 `src`를 열다섯 번 써도 파트는 하나다 — 그러지 않으면 파일이 부풀고,
  PowerPoint에서 "그림 바꾸기"가 한 슬라이드만 고친다.
- **파트의 확장자는 저자가 쓴 것이 아니라 포맷에서 나온다** (`.jpg`·`.JPEG`가 모두
  `image4.jpg`). `Default`가 확장자로 콘텐츠 타입을 정하므로 이름을 포맷에서 지으면 항목이
  포맷당 하나로 모이고, 확장자 대소문자 매칭을 소비자의 관용에 맡기지 않는다.
- **관계 번호는 하이퍼링크와 한 카운터를 나눠 쓴다** — 문서 순서대로 배정한다. 그림과 링크와
  노트가 한 슬라이드에 있는 자리에서 어긋나기 쉬운 배정이다 (예: `rId1` 레이아웃 · `rId2` 노트 · `rId3` 이미지 · `rId4` 하이퍼링크).
- **`name`은 지오메트리 식별자다** (`image s0.1.0.0`) — 도형과 같은 규약이고, PowerPoint의
  선택 창에 나오는 이름이 진단 메시지의 이름과 같다.
- **`descr`은 `alt`이 있을 때만 낸다.** 웹은 `alt=""`를 언제나 내지만 OOXML에는 "장식"의
  표기가 없어 `descr=""`는 "대체 텍스트가 비었다"이고, 빈 속성은 재저장 diff의 잡음이 된다.
- **`spPr`에 채움을 내지 않는다** — 그림의 채움은 `p:blipFill`이 이미 소유한다 (도형은
  `a:noFill`을 명시해야 테마 채움이 상속되지 않지만 여기는 반대다). **선은 명시한다**
  (`a:ln/a:noFill`) — 선은 `blipFill`이 소유하지 않으므로 비워 두면 상속의 여지가 남는다.
- `radius`가 있으면 `prstGeom`이 `roundRect`가 되고 조정값은 도형의 `rounded-rect`와 **같은
  명세**에서 나온다 (같은 `radius="md"`가 도형과 그림에서 다른 모서리로 그려지지 않게).
- `picLocks noChangeAspect`를 넣지 않는다 — 산출물은 편집 가능해야 하고, 엔진이 잠근 제약은
  파일을 열어 본 사람이 이유를 알 수 없다.

### 내 슬라이드는 PPTX에서 도형 몇 개가 되는가

| SOML | PPTX 도형 수 |
|---|---|
| 레이아웃 컨테이너 (`<row>` `<col>` `<grid>`) · `<spacer/>` | **0개** |
| 리스트 컨테이너 (`<ul>` `<ol>`) | 1개 — 불릿 문단이 한 텍스트 박스에 산다 |
| `<shape>` | 1개 (`p:sp`) — 자식 텍스트를 흡수한다 (컨테이너를 담으면 1개 + 자식들) |
| **`<image>`** | **1개 (`p:pic`)** — 담을 것이 없다 |
| `<text>`·블록 태그 | 1개씩 |

`<image>`가 `p:sp`가 아니라 `p:pic`인 것이 도형과 갈라지는 자리다.

## 빌드 에러가 나는 조건

파서(`soml.Parse`) — 이름과 콘텐츠 모델:

| 조건 | 진단 메시지 |
|---|---|
| 내용을 담았다 | `void tag <image> cannot have content (write <image/>)` |
| 문단·리스트 항목 안에 썼다 | `only character data and inline tags are allowed inside <p> (<image> is not allowed)` |
| 도형의 **직속** 자식으로 썼다 | `<shape> cannot contain <image> (available tags: <text>, <h1>, …, <row>, <col>, <grid>, <stack>)` — 도형 안 그림은 컨테이너를 거쳐 들어온다 (`<shape><col><image/></col></shape>`) |
| 어휘 밖의 속성 (`fill`·`frame`·`type`·`anchor` …) | `unknown attribute fill on <image> (available attributes: src, id, fit, alt, variant, border, radius, shadow, opacity, rotate, flip, pad, w, h, min-w, min-h, max-w, max-h, ratio, grow, shrink, basis, span, self, order, at, x, y, style, slot)` |

값 해석(`core.Resolve`) — 경로 **문법**과 값 어휘. 디스크를 보지 않으므로 에디터가 프로젝트를
열지 않고도 같은 진단을 낸다. `theme:` 그림만은 여기서 테마 패키지의 파일까지 판정한다 — 그 파일이
없거나 내용이 확장자와 다르면 아래 에셋 패스와 같은 문구의 진단이 여기서 난다:

| 조건 | 진단 메시지 |
|---|---|
| `src`가 없다 | `<image> needs a src (src="assets/photo.png") — the image path cannot be guessed` |
| `src`에 값이 없다 | `the src attribute needs a value (src="assets/photo.png")` |
| `src=""` | `src is empty — write a path relative to the project root (e.g. "assets/photo.png")` |
| 절대 경로 | `src is a path relative to the project root ("/Users/me/a.png" is absolute) — an absolute path ties the build to this machine (the same deck builds differently in CI)` |
| 프로젝트 밖 | `src points outside the project ("../shared/a.png") — keep assets inside the project (pointing outside makes the build depend on this machine)` |
| URL | `src cannot be a URL ("https://…") — a build that depends on the network builds the same deck differently in CI and breaks the offline-presentation promise; put the file in the project and point to it with a relative path` |
| `data:` URL | `src cannot be a data: URL — commit the image as a file (bytes inlined in the source make the slide unreadable to people and leave diagnostics nothing to point at)` |
| 역슬래시 | `the path separator in src is "/" ("assets\wide.png" contains a backslash) — so the same deck builds the same on every machine` |
| `theme:` 뒤가 비었다 | `no path after theme: — write a path relative to the theme's assets/ (e.g. "theme:icons/logo.png")` |
| `theme:` 절대 경로 | `a theme: path is relative to the theme's assets/ ("theme:/x.png" is absolute)` |
| `theme:` 경로가 `assets/` 밖 | `src points outside the theme's assets/ ("theme:../x.png") — a theme: image is a file inside the theme's assets/` |
| `theme:` 그림이 테마 체인에 없다 | `cannot find theme:icons/x.png in theme "corp-dark"(→ "corp" → "minimal") (under assets/)` |
| 확장자가 없다 | `src has no extension ("assets/photo") — the extension sets the PPTX content type (available: .png, .jpg, .jpeg)` |
| 지원하지 않는 확장자 | `v1 cannot render ".svg" in src (available: .png, .jpg, .jpeg) — PPTX carries vectors as an extension element (svgBlip) with a raster fallback, so each consumer opens it differently — export as png or jpeg` |
| CSS `object-fit` 값을 썼다 | `CSS object-fit "fill" has no PPTX equivalent (available values: cover, contain) — instead of stretching past the aspect ratio, match the box to the image with ratio` |
| 어휘 밖의 `fit` | `unknown fit "crop" (available values: cover, contain)` |
| `alt`에 값이 없다 | `the alt attribute needs a value (alt="chart") — write alt="" for a decorative image` |
| 모르는 `variant` | **경고** — `variant "frame" is not in the theme (did you mean "framed"?) — continuing with the default appearance (variants in this theme: framed)` |

외관 값 어휘(`border`·`radius`·`shadow`·`opacity`·`rotate`·`flip`)와 박스 속성의 진단은
[도형](shape.md)·[레이아웃 태그](layout.md)와 **같은 문구**다 — 같은 함수가 해석하므로 진단도
한 벌이다. 다른 자리는 하나뿐이다: `radius`에 프리셋 제약 진단이 없다.

에셋 패스(`core.ResolveAssets`) — 파일이 **실제로** 무엇인지. 같은 키는 몇 번 참조해도 진단이
한 번이다:

| 조건 | 진단 메시지 |
|---|---|
| 파일이 없다 | `the file src points to does not exist (assets/missing.png) — paths are relative to the project root (not to the slide file)` |
| 디렉토리다 | `src points to a directory (assets/dir.png) — point it at a single image file` |
| 0바이트 | `assets/empty.png is empty (0 bytes) — not an image file (a download or LFS checkout may not have finished)` |
| 헤더가 잘렸다 | `cannot read the image header of assets/cut.png (the file is truncated) — check the committed bytes` |
| 확장자와 내용이 다르다 | `the extension of assets/liar.png says png but the content is jpeg — [Content_Types] derives the content type from the extension, so PowerPoint cannot open the file as is (change the extension to ".jpg")` |
| 그림이 아니다 | `the content of assets/text.png is not png — a file with only its extension changed builds fine but PowerPoint cannot open it (re-export as png or jpeg)` |
| 고유 크기가 0 | `assets/zero.png has an intrinsic size of 0×0 — nothing to draw` |

렌더 단계에는 진단이 없다 — 그림을 그릴 수 없는 상태는 값 층과 에셋 패스가 이미 전부 막는다.
브라우저 단계에 하나 남는데, 그것은 저자의 실수가 아니라 산출물의 이상이다: **산출물의 그림이
로드되지 않으면 측정이 실패한다** (고유 크기 0인 그림의 rect는 0×0이고, 그 좌표가 PPTX
`xfrm`으로 나가면 아무것도 없는 슬라이드가 나온다).
