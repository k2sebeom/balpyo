# 테스트 환경

balpyo의 테스트는 두 렌더러가 **같은 컴포넌트를 같은 자리에 낸다**는 것을 검사한다 — 픽셀이
아니라 컴포넌트 트리다 ([ADR 0064](../decisions/0064-the-contract-is-component-and-layout-parity-not-pixel-parity.md)).
그 판정은 같은 실행의 같은 측정값을 두 번 읽는 것이라 설치된 폰트나 렌더 엔진 버전에 기대지
않는다. 그래서 게이트는 컨테이너가 아니라 **호스트의 `go test ./...` 하나**다
([ADR 0065](../decisions/0065-pptx-validity-is-judged-by-the-ecma-376-schema-not-by-libreoffice.md)).

```
make test      # go test ./... — CI의 게이트와 같은 명령
make examples  # 예제 덱 셋을 경고 0으로 빌드하는지만 확인 (make test에도 포함)
make golden    # 골든 갱신
make docs      # mkdocs build --strict
```

## 호스트에서 돌리기

외부 도구는 **Chromium/Chrome 하나**다. 빌드 자체가 지오메트리 패스로 박스를 재기 때문이다 —
테스트만을 위한 도구가 아니라 `balpyo build --target pptx`의 의존성이다. PPTX의 유효성은
LibreOffice가 아니라 저장소에 둔 ECMA-376 스키마가 판정하므로([아래](#pptx-유효성)) 다른 것은
설치하지 않는다.

도구는 아니지만 **글꼴**도 필요하다. 잰 빌드는 테마 글꼴이 측정 브라우저에 없으면 경고를 내고
([글꼴 경고](../../docs/reference/cli.md#글꼴-경고)), 픽스처 골든과 예제 게이트(경고 0)는 내장 ·
예제 테마의 Noto 글꼴이 깔려 있다고 본다. Debian/Ubuntu에서는 `fonts-noto-core` ·
`fonts-noto-mono` · `fonts-noto-cjk`다 (CI가 설치한다).

Chromium이 없으면 빌드가 필요한 테스트는 메시지와 함께 **건너뛴다** — 픽스처 · 예제 스위트
전체가 여기에 든다 (`Fixture.Build`가 정책을 적용하는 자리다). 도구가 없는 호스트에서 초록이
판정하는 것은 파서 · 모델 · 렌더러의 단위 테스트와 스키마 변이 테스트까지다.

건너뛰기가 거짓 초록이 되지 않게 하는 장치가 `BALPYO_TEST_STRICT=1`이다. CI가 이 값을 켜므로
러너에 브라우저가 없으면 건너뛰지 않고 **실패**한다.

**브라우저는 테스트 바이너리마다 하나다.** `testkit.SharedSession`이 처음 부를 때 띄우고, 패키지의
`TestMain`이 끝에서 닫는다. 빌드마다 Chrome을 띄우던 때 전체 스위트가 59초였고, 세션을 나눠 쓰면서
17초가 되었다 — 실행 시간의 대부분이 브라우저 기동이었다. 빌드마다 산출 HTML을 새로 열고(`Session.Open`)
재므로 빌드 사이에 상태가 새지 않는다.

하네스가 읽는 환경 변수:

| 변수 | 뜻 |
|---|---|
| `BALPYO_TEST_STRICT` | 1이면 Chromium 없음을 실패로 본다 (CI가 켠다) |
| `BALPYO_UPDATE_GOLDEN` | 1이면 골든을 비교하지 않고 쓴다 (`make golden`) |
| `BALPYO_CHROMIUM` | 브라우저 실행 파일 경로 (없으면 `geom.Candidates`의 순서로 찾는다) |
| `BALPYO_CHROMIUM_NO_SANDBOX` | 1이면 `--no-sandbox` (CI 러너처럼 샌드박스를 못 쓰는 환경) |
| `BALPYO_ONLYOFFICE` | `balpyo preview`의 PPTX 래스터 엔진(OnlyOffice Document Builder) 설치 디렉토리 — 게이트와 무관하다 |
| `BALPYO_SOFFICE` / `BALPYO_PDFTOPPM` | OnlyOffice가 없을 때의 PPTX 래스터 도구 경로 — 게이트와 무관하다 |

## 픽스처

픽스처 하나(`testdata/fixtures/<NN>_<이름>/`)는 **완전한 balpyo 프로젝트**다. 그래서 사람이
그 디렉토리를 직접 빌드해 결과를 볼 수 있다.

```bash
go run ./cmd/balpyo build --project testdata/fixtures/00_empty --target html,pptx
```

```
00_empty/
├── balpyo.json          # 덱 설정 (없으면 기본값)
├── slides/*.slide       # 입력
├── assets/*.png         # 그림 입력 (그림이 있는 픽스처만 — 05_image)
├── themes/<이름>/       # 프로젝트 로컬 테마 입력 (15_local_theme)
└── golden/              # 구조 기록 (사람이 커밋한다 — 아래 골든)
    ├── layout.txt       #   컴포넌트 트리 (rect 없이)
    └── diagnostics.txt  #   경고가 있을 때만
```

- 접두 숫자는 iteration 순서이자 스위트 실행 순서다.
- `_`로 시작하는 디렉토리는 수집되지 않는다 (작업 중 픽스처). 슬라이드 수집 규칙과 같은 관례다.
- 빌드 산출물은 임시 디렉토리에 쓴다 — 픽스처 디렉토리에는 **입력과 골든만** 남는다.
- 픽스처는 **자기 프로젝트의 것만** 쓴다. `15_local_theme`이 그 성질의 증인이다: 바이너리에 없는
  테마 이름(`corp`)이 `themes/corp/`에서 풀리고, 그래서 이 픽스처는 `core/themes/`를 고치지 않고
  테마 상속을 끝까지 시험한다 (ADR 0054).

## 예제 게이트 — 도그푸딩 덱은 조용해야 한다

`examples/decks/` 아래 덱 셋(`balpyo-intro` 12장 · `incident-postmortem` 10장 ·
`quarterly-review` 6장)은 **사람이 실제로 쓴 덱**이고, 픽스처가 못 하는 일을 한다: 어휘 수십
개가 한 장에 겹쳐 있어 "이 도구로 덱을 쓸 수 있다"의 실물 증거가 된다. 그래서 게이트도 다르다.

```bash
make examples   # 예제만 (몇 초) — make test가 같은 테스트를 이미 돈다
```

`TestExamplesBuildClean`이 예제마다 판정하는 것은 넷이다 ([ADR 0056](../decisions/0056-examples-are-gated-by-a-strict-build-not-by-goldens.md)):

| 판정 | 어떻게 |
|---|---|
| 두 타깃으로 빌드된다 | `--target html,pptx` — PPTX가 지오메트리 패스를 돌리므로 좌표까지 난다 |
| **진단이 0건이다** | `Strict` — 경고도 실패다. 넘침·이탈·겹침(`geom.Diagnose`)과 테마에 없는 variant가 여기서 걸린다 |
| 슬라이드 수가 선언한 숫자와 같다 | `exampleSlideCounts` 표 (`12 · 10 · 6`) |
| PPTX가 유효하다 | `AssertValidPPTX` — 스키마 · 패키지 · 파트 사이 ([아래](#pptx-유효성)) |

**경고가 실패인 이유**는 예제가 사용자 덱이 아니라는 것이다. 경고는 "통과하되 보이는 것"으로
정의돼 있지만([ADR 0039](../decisions/0039-layout-warnings-are-box-containment.md)) 그 정의는
저자의 덱을 위한 것이고, 넘침 경고를 내는 예제는 읽는 사람에게 "경고는 무시해도 된다"를
가르친다. 일부러 경고를 내는 덱이 필요하면 그것은 예제가 아니라 **픽스처**다 — 픽스처는
`diagnostics.txt` 골든으로 경고를 증거로 남긴다.

**슬라이드 수를 손으로 적는 이유**는 이 게이트가 놓칠 수 있는 가장 조용한 사고를 잡는 것이다:
장이 사라졌는데 남은 장이 깨끗하게 빌드되는 것(파일 이름이 `_`로 시작하게 되거나 파일이
지워졌을 때). 목록 자체는 디렉토리를 걸어 얻으므로 새 예제는 자동으로 게이트에 들어오고, 그
표에 줄을 더할 때까지 **실패한다**.

**예제에는 골든이 없다.** 골든의 값은 사람이 diff를 읽는 데서 나오고, 픽스처가 그 조건을
만족하는 이유는 크기와 의도다(한 픽스처가 한 어휘, 2~4장). 예제 28장에 골든을 걸면 `accent1`
색을 한 번 고치는 일이 28장의 XML diff가 되고, 그 안에 섞인 진짜 회귀 한 줄은 보이지 않는
채로 승인된다. 그래서 예제는 "빌드되고 조용하다"만 지키고, **무엇이 어떻게 나오는가는
픽스처의 골든이** 지킨다. 다만 PPTX 유효성(`AssertValidPPTX`)은 골든이 아니라 규칙이라 예제도 받는다.

산출물은 임시 디렉토리에 쓴다 — 예제 디렉토리는 입력 전용이고 `examples/decks/*/dist`는 커밋하지
않는다. 사람이 손으로 보는 길은 그대로다:

```bash
go run ./cmd/balpyo build --project examples/decks/balpyo-intro --target html,pptx --strict
```

## 문서 게이트 — 닫힌 목록은 테스트가 문서에서 읽는다

문서에는 어휘의 **사본**이 있다 (도형 12종, 레이아웃 11종, 색 토큰 10종 …). 실제로 낡은 것이
그 사본이었으므로 — D19의 "`layout` 어휘는 2종", 개요의 "내장 테마는 `minimal`·`dark`·`paper`" —
`TestDocsVocabularyListsMatchCode`가 등록된 자리마다 문서의 목록을 코드의 목록과 맞댄다
([ADR 0057](../decisions/0057-docs-are-hand-written-but-closed-lists-are-read-from-them-by-a-test.md)).

```bash
go test ./core -run TestDocsVocabularyListsMatchCode   # 도구 없이 밀리초
```

| 판정 | 어떻게 |
|---|---|
| 문서의 목록이 코드의 목록과 **집합으로 같다** | `docVocabularies`의 (문서 · 어휘 · 패턴 · 구분자) 등록 줄 |
| 문서에서 목록을 **찾았다** | 패턴이 한 번도 맞지 않으면 `Fatal` — 생략(`…`)만 남은 문서도 실패다 |

순서와 표기는 문서의 것이다: 같은 목록이 진단 형식(`rect, rounded-rect, …`)으로도 산문 형식
(`` `minimal`·`editorial`·`ink` ``)으로도 적힐 수 있고, 판정하는 것은 이름의 집합뿐이다.
**패턴 불일치를 실패로 두는 이유**는 이 게이트가 죽는 가장 조용한 길이 문서를 고쳐 써서 패턴이
어긋나는 것이기 때문이다 — 그러면 테스트는 초록인데 아무것도 보지 않는다.

**진단 문구 전체는 대조하지 않는다.** 문서 표의 칸은 진단 + 사람을 위한 꼬리이고, 문구의 회귀는
각 패키지의 테스트가 기댓값으로 이미 잡는다. 문서에만 있는 위험은 목록이다. 문서에 실린 SOML
예제가 실제로 빌드되는지도 아직 아무것도 보지 않는다 (픽스처에서 가져오는 관례가 유일한 장치다).

테스트가 `core`에 있는 이유: 필수 토큰 목록이 그 패키지의 비공개 변수이고(테스트 때문에 공개
API를 늘리지 않는다), `internal/testkit`은 `internal/pipeline`을 임포트하므로 거기 두면 순환이다.
문서만 고치는 커밋은 이 판정만 돌리면 된다 — 브라우저도, 픽스처 스위트도 필요하지 않다.

## 골든

골든은 **두 종뿐이다** — 산출물 스냅샷이 아니라 사람이 diff로 읽는 구조 기록이다.

| 파일 | 담는 것 |
|---|---|
| `golden/layout.txt` | 슬라이드마다 컴포넌트 트리 — 종류 · 요소 ID · 플레이스홀더 · 채움/선 · 문단의 런 (rect는 뺀다) |
| `golden/diagnostics.txt` | 진단 (경고가 있을 때만) |

rect를 뺀 이유는 rect가 폰트 메트릭을 따라 움직이기 때문이다 — rect는 아래의 동일성 판정이 같은
실행 안에서 맞대고, 골든은 "무엇이 어떤 순서로 어떤 외관으로 나오는가"만 기록한다. 그래서 골든은
폰트가 다른 호스트에서도 한 바이트도 움직이지 않는다 (`TestParityHoldsUnderDifferentFonts`가 테마의
글꼴 토큰을 고정폭으로 갈아 끼워 그것을 확인한다).

골든은 `make golden`으로만 갱신하고, 손으로 쓰지 않는다. 골든 변경은 구현 변경과 **같은
커밋**에 담아 "무엇이 왜 움직였는지"를 기록에 남긴다. 갱신 실행은 내용이 같은 파일을
건드리지 않으므로, 나온 diff는 실제로 움직인 골든만 담는다. 골든 트리 비교는 **남은 골든도
잡는다** — 옛 이름의 파일이 아무도 보지 않는 채로 남아 통과하지 않게 한다
(`TestFixtureGoldenTreeShape`는 두 종 밖의 파일이 끼어드는 것도 실패로 본다).

예전의 파트별 PPTX XML 골든 · HTML 골든 · 바이너리 에셋 골든은 없다. CSS 한 줄에도 수십 파일이
움직였고 리뷰어가 그 diff를 읽지 않았다 — 그 자리를 아래의 규칙 판정이 맡는다.

## 판정 — 픽스처마다

`TestFixtures`가 픽스처마다 하는 일: 빌드 → 골든 비교 → 동일성 → 패키지 불변식 → 두 번째 빌드와
같은 바이트.

### 동일성 (ADR 0064)

`AssertParity`(`parity.go`)가 두 컴포넌트 트리를 줄 단위로 맞댄다.

- **웹 쪽 트리**는 덱 모델 + 지오메트리 side table에서 만든다 — 요소마다 측정 rect를 렌더러와
  **따로** EMU로 환산한다 (렌더러 함수를 부르면 렌더러의 오해가 판정에 그대로 옮겨진다).
- **PPTX 쪽 트리**는 산출된 슬라이드 XML의 `spTree`를 다시 읽어 만든다.

노드 한 줄은 종류(`textbox` · `shape:<prst>` · `pic` · `table`) · 요소 ID · rect · 플레이스홀더 ·
대체 텍스트 · 채움/선이고, 그 아래 문단마다 정렬 · 마커 · 런(글자 · 글꼴 참조 · 크기 · 굵기 · 기울임 ·
밑줄 · 색 토큰 · 링크)이 한 줄씩 붙는다. **줄이 어디서 접히는지는 트리에 없다** — 계약 밖이다.

**토큰은 눕지 않는다.** 런과 채움의 색은 토큰 이름(`accent1`) 또는 리터럴(`#0F65F9`)로 트리에 적힌다.
토큰으로 선언한 색이 PPTX에서 `srgbClr`로 나오면 PPTX 쪽 줄이 리터럴이 되어 트리가 갈라진다 — 같은
RGB여도 그렇다.

### PPTX 유효성

`AssertValidPPTX`(`package.go`)는 덱과 무관한 규칙을 본다 — 픽스처 · 예제 · `render/pptx`의 단위
테스트 덱이 같은 판정을 받는다.

| 판정 | 어디서 |
|---|---|
| 모든 XML 파트(`.xml` · `.rels`)가 **ECMA-376 Transitional 스키마**에 맞는다 | `schema.go` — `github.com/knroy/go-xml`, 스키마 원본은 `internal/testkit/ooxml/`에 고치지 않고 둔다 |
| 파트마다 content type이 있다 · 관계의 대상과 주인이 있다 · 아무도 가리키지 않는 파트가 없다 | `package.go` |
| 파트 사이의 참조가 맞는다 — 도형 id 유일 · `sldId` 범위 · 관계 종류 · 슬라이드→레이아웃→마스터 · 노트 ↔ 슬라이드 · 플레이스홀더 자리 | `crosspart.go` |
| zip 타임스탬프가 1980-01-01이다 | `package.go` |

덱에 기대는 규칙은 `AssertPackage`가 더 본다: 슬라이드 파트 수 = 장 수 · 노트 파트는 노트가 있는
장에만 · 마스터에 `p:hf`가 없다 · 미디어 파트의 이름이 에셋 표의 자리 + 포맷 확장자이고 **바이트가
입력 파일과 같다**(웹 산출 에셋도) — 엔진은 그림을 고치지 않는다.

스키마는 PowerPoint가 먼저 보는 것이다. 복구 대화상자의 흔한 원인 — 자식 순서 · 필수 속성 ·
열거값 · 타입 · 범위 — 는 전부 스키마 위반이고, 예전의 LibreOffice 열림 검사는 그것을 조용히 받아
주었다. 스키마 밖의 규칙은 파트 사이 검사가 아는 만큼만 덮는다 — 새로 알게 된 규칙은 거기에 더한다.

### 검증기를 검증한다 — 변이 테스트

판정이 조용히 느슨해지는 것(스키마 적재가 lax로 떨어지거나 의존성이 바뀌는 것)이 이 장치가 죽는
가장 조용한 길이다. 그래서 검증기가 **실제로 거르는지**를 테스트가 못 박는다.

- `TestSchemaRejectsMutatedParts` — 작은 덱의 파트를 하나씩 망가뜨린다: 자식 순서 · 필수 속성 ·
  열거값 · 프리셋 이름 · 모르는 요소 · 모르는 속성 · 범위 · 타입 · rels `Target` · content type 속성 ·
  모르는 이름공간. 변이마다 검증 에러가 나고, 메시지가 그 변이를 가리켜야 한다.
- `TestCrossPartsRejectsMutations` — 같은 방식으로 파트 사이 규칙을 하나씩 깬다.

변이 테스트는 브라우저 없이(합성 지오메트리로) 돈다 — 관심은 좌표가 아니라 XML의 모양이다.

### 결정성 (불변식 6)

`AssertSameBytes`가 같은 입력을 두 번 빌드해 산출 목록과 **모든 파일의 바이트**가 같은지 본다. zip
타임스탬프 고정, 맵이 아니라 순서 있는 목록 — 둘 중 하나가 깨지면 여기서 걸린다.

## 사람이 하는 확인

"두 타깃이 같은 것을 내는가"는 위의 판정이 매 실행 답한다. 다만 그 판정은 **우리가 쓴 XML을 우리가
읽는 것**이라, OOXML을 같은 방식으로 오해하면 두 쪽이 함께 틀린다. 그 종류의 결함은 같은 XML을
명세대로 읽는 다른 엔진이 그린 그림에서 보인다 ([ADR 0069](../decisions/0069-onlyoffice-is-the-pptx-review-renderer.md)).

```bash
make qa   # 예제 덱 · 픽스처마다 웹 | PPTX 갤러리 → preview/qa/<이름>/index.html
```

[`balpyo preview`](../../docs/reference/cli.md#balpyo-preview)가 웹 스크린샷과 PPTX 렌더를 나란히 놓는다
(판정은 없다 — [ADR 0066](../decisions/0066-preview-only-places-renders-side-by-side.md)). **렌더러를 고친
커밋은 `make qa`의 갤러리를 보고 낸다** — 동일성은 그대로 초록인데 다른 엔진에서 갈라지는 변경이
여기서 드러난다.

PPTX 열의 엔진은 **OnlyOffice가 먼저**다 (Document Builder — `x2t`). OOXML을 1차 형식으로 읽어 프리셋
공식 · 플레이스홀더 상속 · 색 변형을 명세대로 계산하고, 커닝을 켜지 않은 런을 PowerPoint처럼 커닝
없이 놓는다. 없으면 LibreOffice + `pdftoppm`으로 물러선다 (ODF로 옮긴 뒤 그리므로 자기 버릇을 더한다
— 한글↔라틴 간격). 갤러리 안내가 어느 엔진의 그림인지 밝힌다.

```bash
# Linux x86_64 — 설치 디렉토리 하나가 변환기(x2t) · 글꼴 색인 생성기(docbuilder) · 라이브러리를 담는다
curl -LO https://github.com/ONLYOFFICE/DocumentBuilder/releases/latest/download/onlyoffice-documentbuilder-linux-x86_64.tar.xz
sudo mkdir -p /opt/onlyoffice && sudo tar -xJf onlyoffice-documentbuilder-linux-x86_64.tar.xz \
  --strip-components=2 -C /opt/onlyoffice opt/onlyoffice/documentbuilder
# 다른 자리에 풀었으면 BALPYO_ONLYOFFICE=<설치 디렉토리>
```

글꼴 색인(`sdkjs/common/AllFonts.js`)은 변환 전마다 `docbuilder`가 설치된 글꼴과 대조해 갱신한다 —
색인이 낡으면 `x2t`는 한글만 빠진 그림을 성공으로 낸다. 그래서 설치 디렉토리는 쓸 수 있는 자리가
좋다 (쓸 수 없으면 있는 색인으로 간다).

**OnlyOffice의 한계로 확인된 차이** — 우리 XML이 명세대로인데 그림이 다른 자리다. 결함으로 읽지 않는다:

| 그림 | 실제 |
|---|---|
| 그라디언트 테두리가 첫 스톱 단색이다 | `a:ln/a:gradFill`이 스톱 둘로 나간다 (07_gradient 2장) |
| 부드러운 그림자가 아래 띠로만 보인다 | `outerShdw`의 `blurRad`가 토큰의 blur와 같다 (quarterly-review 3장) |

**편집 가능성은 파일을 열어 본다.** PPTX를 Keynote나 PowerPoint로 **직접 열어** 편집
가능성을 확인한다 — 텍스트 박스가 진짜 텍스트 박스인지, 테마 색을 바꾸면 도형 색이 따라오는지.
"열리는 파일"과 "편집 가능한 파일"의 차이가 이 제품의 핵심이고, 자동 판정은 그 앞(스키마 · 동일성)
까지다.

## CI

`.github/workflows/ci.yml`의 잡은 둘이다.

| 잡 | 하는 일 |
|---|---|
| `test` | `setup-go` + 러너의 Chrome + Noto 글꼴 → `make test` (`BALPYO_TEST_STRICT=1` · `BALPYO_CHROMIUM_NO_SANDBOX=1`) |
| `docs` | `make docs` (`mkdocs build --strict`) |

로컬과 CI는 **같은 명령**을 돈다. 둘의 차이는 설치된 폰트와 브라우저 버전뿐이고, 판정이 그 둘에
기대지 않으므로 로컬에서 초록이면 CI에서도 초록이어야 한다.

게이트를 늘리는 방법도 그 등식이 정한다: **잡을 더하지 않고 테스트를 더한다.** 예제 게이트가
그 예다 — `make test`가 이미 `TestExamplesBuildClean`을 돌기 때문에 CI에 잡이 따로 없다
([ADR 0056](../decisions/0056-examples-are-gated-by-a-strict-build-not-by-goldens.md)).
