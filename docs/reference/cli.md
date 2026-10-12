# CLI

바이너리 하나가 모든 서브커맨드를 담는다. 지금 구현된 표면은 `init`·`build`·
`check`·`watch`·`preview`·`snap`·`theme`·`version`이다 — `format`/`lsp`는 아직 열리지 않았고, 열리면 이
문서에 함께 실린다.

## `balpyo init`

빌드되는 프로젝트 하나를 만든다 — 빈 디렉토리에서 시작하는 명령이다.

```bash
balpyo init my-talk --author "이세범"
```

```
created project: /path/to/my-talk
  balpyo.json
  slides/001_title.slide
  slides/002_content.slide
  slides/003_end.slide
  .gitignore

Next:
  balpyo watch --project my-talk
  balpyo build --project my-talk --target html,pptx
```

만들어진 덱은 **그 자리에서 빌드된다** — `balpyo build --target html,pptx --check-layout
--strict`를 경고 0으로 통과한다. 첫 명령이 경고를 내면 저자는 경고를 읽지 않는 습관부터
배우므로, 이것이 스캐폴딩의 계약이다. 조건이 하나 있다: 테마의 글꼴(기본 `minimal`은 Noto
Sans · Noto Sans Mono · Noto Sans CJK KR · Noto Sans Mono CJK KR)이 빌드하는 기계에 설치되어
있어야 한다. 없으면 [글꼴 경고](#글꼴-경고)가 그 사실을 알린다.

세 장이 각각 가르치는 것이 있다. 표지는 자리를 **테마의 슬롯**이 정한다는 것(자식에 `align`이
하나도 없다 — [슬롯](layouts.md)), 두 번째 장은 `<row>`·`<col>`·`<shape>`로 칸을 나누는
방법과 파일 이름의 번호가 슬라이드 순서라는 것, 세 번째 장은 `layout`이 정렬을 바꾼다는
것이다. `<note>`(발표자 노트)도 표지에 한 줄 들어
있다.

### 옵션

디렉토리는 위치 인자로도 `--project`로도 준다 (`balpyo init my-talk` = `balpyo init
--project my-talk`). 둘을 함께 주면 멈춘다 — 어느 쪽이 이겼는지 추측하게 두지 않는다.
위치 인자는 플래그 앞에도, 사이에도, 뒤에도 올 수 있다 (`balpyo init --author 나 my-talk --theme ink`).

| 옵션 | 기본값 | 뜻 |
|---|---|---|
| `--project <dir>` | `.` | 만들 프로젝트 루트 (위치 인자와 같은 뜻) |
| `--title <제목>` | 디렉토리 이름 | 덱 제목. PPTX 파일 이름이 여기서 나온다 |
| `--author <이름>` | `Presenter` (자리표시자) | 발표자 |
| `--theme <이름>` | `minimal` | 테마. **내장 테마만** 받는다 (`minimal`·`editorial`·`ink`) |
| `--aspect <비율>` | `16:9` | 종횡비 (`16:9`·`4:3`) |
| `--lang <코드>` | `en` | 덱 언어 (`index.html`의 `lang`) |

`init`의 `--theme`만 내장 테마로 **제한**된다 (`build`·`check`·`watch`·`preview`는 로컬 테마
이름도 받는다): 이 명령은 프로젝트를 **만드는** 명령이라 `themes/`가 아직 없고, 읽히지 않는 이름을
`balpyo.json`에 써 두면 그 파일이 조용한 거짓말이 된다. 자기 테마는 프로젝트를 만든 뒤
`themes/<이름>/`에 두고 `balpyo.json`의 `theme`를 고친다
([자기 테마 만들기](../guides/theming.md)).

### 덮어쓰지 않는다

`balpyo.json`이나 `.slide` 파일이 이미 있는 디렉토리에서는 **아무것도 쓰지 않고** 멈춘다.
`--force`는 없다: 덮어쓸 대상이 저자가 쓴 원고이므로, 되돌릴 수 없는 손실 쪽이 한 번 더
타이핑하는 불편보다 크다. 이미 있는 `.gitignore`는 건드리지 않고 나머지만 쓴다.

| 코드 | 뜻 |
|---|---|
| 0 | 프로젝트를 만들었다 — 찍힌 파일 전부가 새로 쓰였다 |
| 1 | 아무것도 쓰지 않았다 — 이미 프로젝트인 자리, 내장 아닌 테마, 없는 종횡비, 디렉토리를 두 번 준 경우 |

### 아직 아닌 것

템플릿 선택(`--template`), 예제 덱 복사, 슬라이드 추가 명령(`balpyo new slide`)은 열지 않았다.
저장소의 `examples/decks/`는 읽어 보는 완성 덱이고, `init`이 만드는 세 장이 출발점이다.

## `balpyo build`

프로젝트를 읽어 타깃별 산출물을 쓴다.

```bash
balpyo build --project my-talk --target html,pptx
```

```
3 slides → /path/to/dist
  index.html
  text-deck.pptx
```

### 옵션

| 옵션 | 기본값 | 뜻 |
|---|---|---|
| `--project <dir>` | `.` | 프로젝트 루트 — `balpyo.json`과 `slides/`가 있는 곳 |
| `--target <목록>` | `html` | 렌더 타깃. 쉼표로 여러 개: `html,pptx` |
| `--out <dir>` | `balpyo.json`의 `outDir` (기본 `dist`) | 출력 디렉토리. 상대 경로는 프로젝트 루트 기준 |
| `--theme <이름>` | `balpyo.json`의 `theme` (기본 `minimal`) | 테마 **이름**. 내장 셋(`minimal`·`editorial`·`ink`) ∪ 이 프로젝트의 `themes/<이름>/` ∪ 설치한 테마 ([`balpyo theme`](#balpyo-theme)) |
| `--strict` | 꺼짐 | 경고를 에러로 승격한다 |
| `--check-layout` | 꺼짐 | 배치를 검사한다 (넘침·스테이지 이탈·겹침). 브라우저를 띄운다 |

값 우선순위는 **CLI 플래그 > `balpyo.json` > 기본값**이다. `balpyo.json`은 선택이다 —
없으면 기본값을 쓰고 덱 제목은 디렉토리 이름에서 유도한다.

`--theme`가 플래그인 이유는 테마를 고르는 방법이 **같은 덱을 두 테마로 내서 견주는 것**이라는
것이다 (자세히는 [테마 고르기·만들기](../guides/theming.md)). `balpyo.json`을 고쳐서 하면
되돌리기를 잊는다. 없는 이름은 렌더 전에 멈춘다 — 쓸 수 있는 이름을 함께 나열하고, 그 목록에는
이 프로젝트의 로컬 테마와 설치한 테마가 함께 들어간다 (`themes/<이름>/` —
[외부 테마](theme.md)). **경로는 받지 않는다**: 값은 언제나 이름이다.

`--target html,pptx`처럼 PPTX가 포함되고 덱에 좌표가 필요한 요소가 있으면 빌드가
지오메트리 패스를 위해 시스템 브라우저(Chromium/Chrome)를 띄운다 —
브라우저가 잰 박스가 그대로 PPTX 좌표가 된다. `--target html`만 쓰는 저작
루프는 브라우저 비용을 내지 않는다. `--check-layout`은 그 루프에서도 측정을 켜는
플래그다 (아래 배치 경고).

### 웹 산출물에서의 발표 조작

`index.html`은 자체 완결형 발표자 뷰다 — 파일을 브라우저로 열면 슬라이드 한 장이 화면에
맞춰 나온다 (오프라인 발표 가능). 브라우저 탭에는 balpyo 로고가 파비콘으로 뜬다 — 아이콘도
파일 안에 들어 있어 외부 참조가 없다.

| 조작 | 방법 |
|---|---|
| 슬라이드 이동 | 하단 컨트롤 바의 ‹ › 버튼, 또는 ←/→·↑/↓·Space·PageUp/PageDown·Home/End |
| 순차 등장([`reveal`](soml/text-subset.md#순차-등장-reveal)) | **다음**은 남은 스텝을 먼저 먹고 그 다음에 슬라이드를 넘긴다. **이전**은 스텝을 하나 되감고, 스텝이 없으면 앞 슬라이드로 — 뒤로 들어온 슬라이드는 전부 전개된 상태다. 오버뷰에서는 스텝을 건드리지 않고 타일 사이를 옮긴다 |
| 발표자 노트 | 컨트롤 바의 **Notes** 버튼 또는 `n` — 현재 슬라이드의 `<note>`를 패널로 연다 |
| 전체 보기(오버뷰) | 컨트롤 바의 **Grid** 버튼 또는 `o` — 덱 전체가 한 화면의 격자로 줄어든다 |
| 오버뷰에서 나가기 | 타일을 클릭하면 **그 슬라이드로** 들어가고(타일 위의 링크를 눌러도 링크를 따라가지 않는다), `Esc`·`o`·**Grid**는 **보고 있던** 슬라이드로 돌아온다 |
| 딥링크 | `index.html#/<슬라이드 id>` 또는 `#/<1-based 번호>` — 이동하면 주소가 따라온다 |

오버뷰는 보기이지 이동이 아니다 — 들어갔다 그냥 나오면 자리가 그대로다. 열 수와 축소 배율은
창 크기와 슬라이드 수에서 계산하므로 덱 전체가 스크롤 없이 한 화면에 들어온다.

### PDF로 인쇄

PDF 산출 명령은 아직 없다. 브라우저에서 **그대로 인쇄하면** 된다 (`index.html`을 열고 Cmd/Ctrl+P →
"PDF로 저장"): 어느 뷰에서 눌러도 **슬라이드 한 장이 한 페이지**이고, 종이 크기는 덱의 논리
스테이지다 (16:9 기본값은 960×540pt). 배경·도형·이미지가 함께 나오도록 브라우저의 "배경 그래픽"
옵션은 켜 둔다 — 여백은 0으로 맞춰져 있으므로 페이지를 슬라이드가 꽉 채운다. 컨트롤 바와 노트
패널은 인쇄되지 않는다.

발표자 노트가 함께 필요한 유인물은 PPTX를 PowerPoint·Keynote에서 "노트 포함"으로 내보내는 쪽이
낫다 — 웹 산출물의 인쇄는 슬라이드 그림이다.

### 종료 코드

| 코드 | 뜻 |
|---|---|
| 0 | 성공 — 요청한 산출물이 전부 쓰였다 |
| 1 | 실패 — 진단 에러, 설정 오류, 렌더 실패 등. 실패 이유는 stderr에 나온다 |

산출물은 빌드가 끝까지 성공했을 때만 쓰인다 — 진단 에러가 있는 빌드가 출력 디렉토리를
반쯤 갱신한 채 끝나지 않는다.

### 진단 출력

진단은 stderr에 한 줄씩, 에디터가 클릭할 수 있는 관례적 형태로 나온다:

```
slides/002_overview.slide:3:9: error: unknown role "heding" — did you mean "heading"? (available roles: title, heading, body, caption, code)
```

형태는 `파일:행:열: 심각도: 메시지`다. 행·열은 1-based이고 열은 룬(문자) 기준이다 —
한글 한 글자가 세 칸으로 세어지지 않는다. 심각도는 `error`(빌드 실패)와 `warning`
(통과하되 출력, `--strict`가 에러로 승격) 두 가지다. 경고는 빌드가 실패한 경우에도
출력된다.

어휘는 닫혀 있으므로 이름이 틀린 진단은 **그 자리에 쓸 수 있는 목록**을 함께 싣는다 — 태그는 부모가
받는 자식(`<ul>` 안이면 `<li>`), 속성은 그 태그의 속성, 값은 그 속성의 어휘다. 철자가 목록의 이름과
한두 글자 다르면(네 글자 이하의 이름은 한 글자) `— did you mean "heading"?`처럼 후보를 하나 단다. `<b>`·`<i>`·
`<img>`·`<h4>`–`<h6>`처럼 HTML에서 온 태그 이름은 철자 대신 SOML의 같은 뜻(`<strong>`·`<em>`·
`<image>`·`<h3>`)을 권한다.

진단은 **소스 순서**로 나온다 — 파일은 슬라이드 순서(파일명 순, 장 디렉토리 포함)이고, 한
파일 안에서는 행 → 열 순이다. 위치가 없는 진단(파일 전체에 대한 것)은 그 파일의 앞에 선다.
위에서 아래로 고쳐 내려가면 된다.

각 진단은 **정확히 한 번** 찍힌다. 빌드가 진단 에러로 실패하면 stderr에 진단들이
한 줄씩 나온 뒤 마지막에 개수 요약 한 줄(`build failed: N errors`)이 붙는다 — 같은 진단을
통째로 다시 찍지 않는다. 설정·파일 같은 진단 아닌 에러는 그 에러 메시지가 그대로 나온다.

지금 어휘에서 경고가 나오는 곳은 테마에 없는 `variant`, **배치**, 그리고 측정 브라우저에 없는
**글꼴**이다. variant 폴백 규칙은
[외관 속성](appearance.md)의 캐스케이드 절에 있다:

```
slides/006_variant.slide:6:16: warning: variant "hero" is not in the theme — continuing with the default appearance (variants in this theme: kicker, lead)
```

이 빌드는 코드 0으로 통과하고 산출물이 쓰인다. `--strict`를 켜면 같은 진단이 에러가
되어 빌드가 실패한다 — 테마 어휘 밖의 variant를 잡고 싶은 CI에서 쓴다.

### 배치 경고

빌드가 좌표를 재면(`--target`에 `pptx`가 있거나 `--check-layout`) 잰 박스에서 배치 사고를
찾아 경고한다. 다섯 가지를 보고, 넘친 양을 논리 px로 말한다:

| 경고 | 판정 |
|---|---|
| 스테이지 이탈 | 박스가 논리 스테이지(1280×720) 밖으로 나갔다 |
| 부모 박스 이탈 | 박스가 부모 박스를 넘쳤다 — 크기를 정한 도형에 내용이 담기지 않은 모양이다 |
| 콘텐츠 박스 이탈 | 슬라이드 직계 박스가 스테이지 여백(64px)을 침범했다 |
| 글자 넘침 | 줄바꿈할 수 없는 글자(긴 낱말 · URL · `pre`의 긴 줄)가 제 박스 밖으로 흘렀다 |
| 형제 겹침 | 같은 부모의 두 박스가 서로 겹쳤다 |

```
slides/003_kpis.slide:7:6: warning: <p> overflows its parent <shape> at the bottom by 34px — the parent box is too small for its content
```

의도한 겹침과 의도한 이탈은 묻지 않는다: `at`·`x`·`y`를 쓴 요소(와 그 자손), `layout="canvas"`
슬라이드의 요소는 부모 변과 여백을 넘어도 조용하고, `<stack>`·canvas의 겹침도 조용하다.
스테이지 이탈만은 예외가 없다 — 화면 밖은 웹에서 잘리고 PPTX에서는 슬라이드 밖에 놓인다.
판정의 한계와 경고를 읽고 고치는 순서는 [배치 경고 읽기](../guides/layout-warnings.md)에 있다.

### 글꼴 경고

좌표를 재는 빌드는 테마의 글꼴 토큰(`latin`·`ea` 슬롯)마다 측정 브라우저가 **그 글꼴로
실제로 그리는지** 묻는다. 없는 글꼴이 있으면 브라우저는 대체 글꼴로 재고, PowerPoint는 테마에
적힌 글꼴로 그린다 — PPTX의 텍스트 박스가 그 차이만큼 넘치거나 남는다. 없는 글꼴마다 경고
하나가 나온다 (위치 없음 — 원인은 슬라이드가 아니라 테마와 이 기계의 조합이다):

```
warning: theme "corp" font "Malgun Gothic" (heading.ea, body.ea) is missing from the measuring browser — the layout was measured with a fallback font, so text boxes will be off when PowerPoint draws with this font. Install the font on the build machine, or change the theme's font token to an installed font
```

- 테마의 토큰을 **전부** 묻는다 — 덱이 그 역할을 지금 쓰는지는 보지 않는다. 코드 블록이 없는
  덱도 `mono` 글꼴이 없으면 경고를 받는다: 테마의 글꼴은 PPTX 테마에 통째로 실린다.
- 재지 않는 빌드(`--target html`만, `--check-layout` 없이)는 묻지 않는다. 웹 산출물은 보는
  사람의 브라우저가 자기 글꼴로 그린다.
- `--strict`는 이 경고도 에러로 바꾼다. CI에서 `--strict`를 쓴다면 러너에 테마의 글꼴을
  설치한다 (Debian/Ubuntu에서 내장 테마는 `fonts-noto-core` · `fonts-noto-mono` · `fonts-noto-cjk`).

```bash
# 저작 루프에서 한 번 확인한다 (브라우저를 띄운다)
balpyo build --check-layout

# CI: 넘친 덱을 통과시키지 않는다
balpyo build --target html,pptx --strict
```

## `balpyo check`

판정만 한다 — 산출물을 쓰지 않는다. CI에서 진단을 게이트로 쓰는 입구다.

```bash
balpyo check --project my-talk
```

```
3 slides · no diagnostics · layout not checked (--check-layout)
```

판정은 `build`와 **같은 코드**다: 설정 → 수집 → 파싱 → 값 해석 → 에셋 → 검증, 그리고
`--check-layout`이면 측정과 배치 진단까지. 파싱까지만 보는 빠른 길로 가지 않은 이유는
"`check`는 통과했는데 `build`가 실패한다"가 가능해지는 순간 이 명령의 가치가 사라진다는
것이다.

디스크에는 아무것도 남지 않는다 — 출력 디렉토리를 만들지도 않는다 (`build`는 만든다).
`--check-layout`이 재는 웹 문서는 임시 디렉토리에 쓰고 지운다.

### 옵션

| 옵션 | 기본값 | 뜻 |
|---|---|---|
| `--project <dir>` | `.` | 프로젝트 루트 |
| `--theme <이름>` | `balpyo.json`의 `theme` | `build`와 같다 — 테마가 배치를 바꾸므로 배치 판정도 바뀐다 |
| `--strict` | 꺼짐 | 경고를 에러로 승격한다 |
| `--check-layout` | 꺼짐 | 배치도 검사한다 (넘침·스테이지 이탈·겹침). 브라우저를 띄운다 |

`--target`과 `--out`은 없다. 진단은 타깃에 의존하지 않고(배치 경고까지 잰 박스에서 나온다),
렌더 단계에서만 나는 실패는 `build`의 판정이다. 배치를 기본으로 켜지 않은 이유는 `build`와
같다 — 브라우저 기동이 가장 큰 고정 비용이고, 브라우저가 없는 CI에서도 어휘 검사는 돌아야
한다.

요약 한 줄은 **무엇을 봤는지** 말한다: `layout not checked (--check-layout)` 또는
`layout checked`. "깨끗하다"와 "안 봤다"는 다른 문장이다.

### 종료 코드

`build`와 **같은 규약**이다. 같은 덱에 대해 두 명령의 판정이 갈리지 않는다.

| 코드 | 뜻 |
|---|---|
| 0 | 에러가 없다. **경고는 코드 0을 타고 나간다** — stderr에 나오고 요약에 개수가 실린다 |
| 1 | 진단 에러, 또는 `--strict`로 승격된 경고. 설정·파일 오류도 1이다 |

```
$ balpyo check
slides/002_content.slide:1:30: warning: variant "hero" is not in the theme — continuing with the default appearance (variants in this theme: kicker, lead)
3 slides · 1 warning · layout not checked (--check-layout)
$ echo $?
0

$ balpyo check --strict
slides/002_content.slide:1:30: error: variant "hero" is not in the theme — continuing with the default appearance (variants in this theme: kicker, lead)
check failed: 1 error
$ echo $?
1
```

진단의 형태와 중복 규칙은 위 `balpyo build`의 **진단 출력** 절과 같다 — 각 진단은 stderr에
정확히 한 번 나오고, 마지막 한 줄이 개수를 요약한다 (`check failed: N errors`). 실패한 검사는 stdout에
아무것도 쓰지 않는다.

```bash
# CI 레시피: 어휘와 배치를 모두 게이트로 쓴다
balpyo check --check-layout --strict
```

### 아직 아닌 것

기계가 읽는 출력(JSON·SARIF), 슬라이드 범위 지정, 진단 규칙 끄기는 열지 않았다.

## `balpyo watch`

고치면 다시 빌드하고 브라우저를 새로고침한다 — 저작 루프다.

```bash
balpyo watch --project examples/decks/quarterly-review
```

```
watching examples/decks/quarterly-review → http://127.0.0.1:4600  (Ctrl+C to stop)
✓ 14:31:02  6 slides · 1 file (41ms)
✓ 14:31:18  6 slides · 1 file (38ms)
✗ 14:31:33  build failed: 1 error (12ms)
  slides/003_kpis.slide:7:6: error: unknown tag <txt> — did you mean <text>? (tags available inside <col>: …)
```

주소 줄을 브라우저에 붙이면 그 탭이 빌드마다 스스로 새로고침한다. 빌드 하나가 터미널에
한 줄이고, 진단은 그 아래에 `build`와 같은 형태로 붙는다.

### 옵션

`build`의 옵션을 전부 같은 뜻으로 받고(`--project`·`--target`·`--out`·`--theme`·`--strict`·
`--check-layout`) 하나가 더 있다.

| 옵션 | 기본값 | 뜻 |
|---|---|---|
| `--addr <host:port>` | `127.0.0.1:4600` | 감시 서버 주소 (루프백만) |

`--target`에는 `html`이 **있어야 한다** — 없으면 브라우저에 보여 줄 페이지가 없으므로 시작하기
전에 멈춘다. `--target html,pptx`도 되고, 그때는 저장마다 PPTX까지 다시 나온다 (지오메트리
패스를 도므로 한 번이 느리다).

주소를 루프백에 고정한 이유는 이 서버가 저자 한 사람의 도구라는 것이다 — 아직 공개하지 않은
발표 자료를 같은 와이파이의 누구나 열 수 있으면 안 된다. 공유는 빌드한 산출물을 올리는
일이다. 그래서 `--addr`로 바꿀 수 있는 것은 포트와 루프백 표기뿐이다: 호스트는 `127.0.0.0/8`·
`::1`(`[::1]:4600`)·`localhost`만 받고, `0.0.0.0`·`[::]`·호스트를 비운 `:4600`·다른 IP나 이름은
서버를 띄우기 전에 에러로 멈춘다.

### 산출물은 하나다

`watch`는 `build`와 **같은 파이프라인**을 돌려 **같은 출력 디렉토리**에 쓴다. 리로드
클라이언트는 응답 바이트에만 주입되고(`</body>` 앞) 디스크의 `index.html`은 `build`가 쓴
그대로다 — 저자가 보며 쓴 페이지와 배포하는 페이지가 같아야 한다.

| 경로 | 무엇 |
|---|---|
| `/` · `/index.html` | 산출물 문서 + 리로드 클라이언트 (주입은 응답에만) |
| 그 밖의 경로 | 출력 디렉토리의 파일 그대로 (바이트 동일). 밖은 서브하지 않는다 |
| `/_balpyo/status` | 마지막 빌드 상태 JSON |
| `/_balpyo/events` | 같은 JSON을 빌드마다 미는 SSE 스트림 |

상태 JSON의 모양은 다음과 같다. 터미널 한 줄과 화면의 오버레이가 이 **한 값**에서 나온다.

```json
{
  "generation": 3,
  "ok": false,
  "error": "build failed: 1 error",
  "diagnostics": [
    {
      "severity": "error",
      "file": "slides/003_kpis.slide",
      "line": 7,
      "column": 6,
      "message": "unknown tag <txt> — did you mean <text>? (tags available inside <col>: …)"
    }
  ],
  "slides": 0,
  "builtAt": "14:31:33"
}
```

### 실패한 빌드는 화면을 지우지 않는다

빌드가 실패하면 산출물을 건드리지 않는다 — 마지막 성공본이 그대로 남고, 그 위에 진단
오버레이가 뜬다 (왼쪽 아래). 방금 고친 자리를 계속 보면서 이유를 읽는 것이 목적이므로
슬라이드를 밀지 않고 겹쳐 뜬다. 경고만 있는 성공 빌드는 접힌 배지로 뜨고, 배지를 누르면
목록이 열린다.

첫 빌드가 실패하면(산출물이 아직 없다) 자리표시자 페이지가 나온다 — 그 페이지에도
클라이언트가 실려 있어서 빌드가 성공하는 순간 스스로 바뀐다.

### 변경을 어떻게 보는가

프로젝트를 250ms마다 걷고, **직전 스캔과 같아진 뒤에** 한 번 빌드한다. 에디터는 저장 한 번에
여러 파일을 쓰므로(포맷터·임시 파일) 변경을 보자마자 빌드하면 저장 한 번이 빌드 두세 번이
되고, 앞의 것들은 반쯤 쓰인 파일을 파싱해 가짜 진단을 낸다. 파일 서명은 크기와 mtime이다 —
둘이 모두 같은 편집은 놓치며(드물다), 그때는 저장을 한 번 더 하면 된다.

출력 디렉토리와 `.`으로 시작하는 디렉토리(`.git`)는 걷지 않는다. 출력 디렉토리를 보면 빌드가
쓴 파일이 다음 빌드를 부르고 루프가 멈추지 않는다.

걷는 것은 프로젝트뿐이다 — 설치한 테마(`~/.balpyo/themes/`)를 `balpyo theme update`로 바꿨으면
프로젝트 파일을 한 번 저장하거나 감시를 다시 띄운다.

측정이 필요한 감시(`--check-layout`, `--target html,pptx`)는 브라우저 **하나**를 계속 쓴다.
기동이 지오메트리 패스 비용의 대부분이므로(~0.6초) 저장마다 다시 띄우면 검사를 켜고 쓸 수
없다. `--target html`만 쓰는 감시는 브라우저를 아예 띄우지 않는다.

Ctrl+C로 끝낸다 — 웜 브라우저를 닫고 서버를 내린 뒤 코드 0으로 돌아온다.

### 아직 아닌 것

브라우저를 자동으로 열지 않고(주소를 찍는다), 폴링 간격은 플래그가 아니며, 성공한 빌드는
페이지를 **통째로** 새로고침한다 (부분 갱신 HMR이 아니다 — 화면이 파일과 같다는 보장을 잃지
않는 쪽을 골랐다).

## `balpyo preview`

웹과 PPTX를 나란히 놓은 **검수 갤러리**를 만든다 — 두 타깃을 눈으로 견주는 명령이다. 판정은
하지 않는다: 두 타깃의 계약(컴포넌트·레이아웃 동일성)은 저장소의 테스트가 보고, 이 명령은 그림을
사람 앞에 놓는 것까지다.

```bash
balpyo preview --project my-talk
```

```
10 slides → /path/to/preview/index.html
open the PPTX in PowerPoint or Keynote to review it: /path/to/dist
```

`preview/index.html`을 브라우저로 열면 슬라이드마다 두 칸이 나란히 있다: 웹 스크린샷과
OnlyOffice(없으면 LibreOffice)로 되돌린 PPTX — 갤러리 안내가 어느 엔진의 그림인지 밝힌다. 줄바꿈 · 글리프가 두 칸에서 다른 것은 계약 밖의 차이다
([계약 밖의 차이](../guides/known-differences.md)). 산출물(`*.pptx`·`index.html`)도 갤러리 옆에
복사되므로 PPTX를 그 자리에서 PowerPoint·Keynote로 열 수 있다.

### 옵션

`build`의 옵션을 같은 뜻으로 받고(`--project`·`--target`·`--out`·`--theme`·`--strict`) 하나가
더 있다. `--out`은 여기서도 **빌드 산출물**이 가는 곳이고, 갤러리는 `--gallery`가 정한다.
`--theme`로 갤러리를 두 번 만들면(`--gallery preview-ink` 등) 한 덱의 두 테마를 눈으로 견줄 수
있다.

| 옵션 | 기본값 | 뜻 |
|---|---|---|
| `--target <목록>` | html과, 도구가 있으면 pptx | 갤러리에 나란히 놓을 렌더. 하나만 주면 갤러리가 한 열이 된다 |
| `--gallery <dir>` | `<프로젝트>/preview` | 검수 갤러리 디렉토리. 실행마다 비우고 다시 쓴다 |

기본 `--target`이 `build`와 다른 이유는 이 명령의 존재 이유가 **두 타깃을 나란히 보는 것**이라는
것이다. 배치 검사는 항상 켜져 있다 — 눈으로 볼 자리를 찾으러 온 사람에게 이미 숫자로 아는
넘침·이탈을 말하지 않을 이유가 없다 (브라우저는 어차피 떠 있다).

갤러리 디렉토리는 실행마다 **비운다**: 앞선 실행이 남긴 슬라이드가 섞이면 사람이 지금 없는
결과를 보고 판단한다. 산출물 디렉토리(`dist/`) 안에 두지 않는 이유는 검수 산출물이 배포물이
아니라는 것이다 — `dist/`를 그대로 올리는 사람이 스크린샷 수십 장을 함께 올리게 된다.
`.gitignore`에 `preview/`를 넣어 두면 좋다.

그래서 갤러리와 빌드 출력은 **겹칠 수 없다** — 같은 디렉토리이거나 한쪽이 다른 쪽 안에 있으면
(`--gallery dist`, `balpyo.json`의 `outDir`가 `preview` 등) 빌드를 시작하기 전에 멈춘다:
`preview gallery directory … overlaps build output directory … — the gallery clears its directory
when created, which would delete the build output; point --gallery and --out at non-overlapping locations`.

### PPTX 칸은 래스터 엔진이 필요하다

PPTX를 그림으로 되돌리는 엔진은 둘 중 하나다. 앞의 것이 있으면 그것을 쓴다:

1. **OnlyOffice Document Builder** — 설치 디렉토리(`x2t`와 `docbuilder`가 있는 곳)를 찾는다:
   `BALPYO_ONLYOFFICE` → `/opt/onlyoffice/documentbuilder`. OOXML을 직접 읽어 PowerPoint와 같은 기본값으로
   그리므로 PPTX에서만 생기는 접힘 · 겹침이 더 잘 보인다.
2. **LibreOffice(`soffice`)**와 **`pdftoppm`(poppler-utils)**.

이 도구들은 balpyo가 산출물을 만드는 데 쓰는 것이 아니라 **만든 산출물을 다시 그림으로 보는** 데만
쓴다 — 빌드는 여전히 바이너리 하나로 끝난다.

- `--target`을 **주지 않았는데** 도구를 못 찾으면 웹 칸만으로 갤러리를 만든다. 조용히 빼지 않는다 —
  출력에 `dropping the PPTX column — <reason>`가, 갤러리 안내에 PPTX 열을 뺐다는 문장이 남는다.
- `--target`에 `pptx`를 **명시했는데** 도구를 못 찾으면 **빌드를 시작하기 전에** 멈춘다 — PPTX를
  보려고 친 명령이 웹만 내면 사람은 두 타깃을 다 봤다고 읽는다.

```bash
# OnlyOffice Document Builder를 다른 자리에 풀었으면 그 디렉토리를 준다
BALPYO_ONLYOFFICE=/path/to/documentbuilder balpyo preview
# 또는 LibreOffice를 설치하거나(macOS: brew install --cask libreoffice; brew install poppler) 경로를 준다
BALPYO_SOFFICE=/path/to/soffice BALPYO_PDFTOPPM=/path/to/pdftoppm balpyo preview
```

웹만 보는 검수(`--target html`)는 이 도구들을 찾지 않는다. 브라우저(Chromium/Chrome)는
어느 조합에서든 필요하다 — 웹 스크린샷과 PPTX 좌표가 모두 거기서 나온다.

### 게이트가 아니다

| 코드 | 뜻 |
|---|---|
| 0 | 갤러리를 만들었다 |
| 1 | 검수할 것이 만들어지지 않았다 — 빌드 실패, 명시한 pptx의 도구 없음, 산출물이 덱과 맞지 않음 |

픽셀 diff · 비율 · 허용 오차는 없다. 두 렌더는 줄바꿈과 글리프가 다를 수 있고(PPTX 텍스트 몸체는
`normAutofit`이라 PowerPoint가 넘치는 글자를 줄여 맞춘다), `style`에 쓴 외관은 웹에만 적용된다
([외관 속성](appearance.md)) — 그 차이를 숫자로 내면 고칠 자리처럼 읽힌다.

엔진이 파일을 열며 경고나 복구를 냈다면 그 출력도 그대로 보여 준다 — 복구가
일어났다면 사용자도 파일을 열 때마다 같은 경고를 볼 수 있다.

### 아직 아닌 것

갤러리를 감시하지 않고(고치면 `preview`를 다시 돈다), PDF 타깃과 슬라이드 범위 지정, 브라우저
자동 열기는 열지 않았다.

## `balpyo snap`

슬라이드를 **브라우저 없이** 그림으로 그린다 — 쓰는 동안 결과를 빨리 보는 명령이다. 내장 렌더러가
웹 산출물을 직접 배치하고 그리므로 Chromium도, OnlyOffice · LibreOffice도 필요 없다.

```bash
balpyo snap --project my-talk
```

```
12 images → /path/to/my-talk/dist/snap (620ms, builtin renderer)
  001.png
  002.png
  …
```

그림은 슬라이드 번호 순서로 `001.png`, `002.png` …이고, 크기는 논리 스테이지(16:9는 1280×720,
4:3은 960×720)에 `--scale`을 곱한 것이다. 모든 `reveal` 스텝이 펼쳐진 상태이고, 등장 효과(`enter`)는
끝난 자리에 그린다 — 웹의 문서 보기와 같은 장면이다.

### 옵션

`--project`·`--theme`·`--strict`는 `build`와 같은 뜻이다.

| 옵션 | 기본값 | 뜻 |
|---|---|---|
| `--out <dir>` | `<출력 디렉토리>/snap` | 그림을 쓸 디렉토리. 기본은 `balpyo.json`의 `outDir` 아래다 |
| `--slides <목록>` | 전부 | 그릴 슬라이드 번호 (1부터). `1,3-5`처럼 쉼표와 범위를 쓴다 |
| `--scale <배율>` | `1` | 스테이지 px 하나에 드는 픽셀 수. `2`면 1280×720 슬라이드가 2560×1440이 된다 |
| `--gif` | 끔 | 덱 전체를 `deck.gif` 하나로도 쓴다 — 슬라이드마다, 그리고 `reveal` 스텝마다 한 프레임(1.5초)이다 |
| `--outline` | 끔 | 요소마다 잰 박스의 외곽선과 요소 식별자(`s3.1.0` …)를 덧그린다 — 배치가 왜 그렇게 나왔는지 볼 때 쓴다 |

`--outline`의 식별자는 [배치 경고](#배치-경고)와 산출 HTML의 `data-balpyo-id`가 쓰는 것과 같다. 색은
중첩 깊이를 따른다.

### 배치 경고도 함께 나온다

내장 렌더러가 잰 박스로 `--check-layout`과 **같은 판정**(넘침 · 스테이지 이탈 · 겹침)을 하고 경고를
stderr에 낸다. `--strict`면 그 경고가 에러가 되어 종료 코드가 1이 된다 — 그래도 그림은 쓴 뒤에 멈춘다.
이 명령의 목적이 무엇이 넘쳤는지 **보는** 것이기 때문이다.

| 코드 | 뜻 |
|---|---|
| 0 | 그림을 썼다 (경고가 있어도) |
| 1 | 덱에 진단 에러가 있거나, 없는 슬라이드 번호를 줬거나, `--strict`에서 배치 경고가 있었다 |

### 빌드 산출물과 같은 그림은 아니다

배치는 웹 산출물의 HTML · CSS를 그대로 읽어 브라우저와 같은 규칙(flex · grid · 표 · 줄바꿈)으로
계산하고, 예제 덱과 테스트 덱에서 요소의 자리가 브라우저와 1px 안에서 맞는다. 다만 글꼴을 고르고
그리는 엔진이 브라우저와 달라서 **줄이 접히는 자리와 글리프 모양까지 같다고 약속하지 않는다.**
산출물(`index.html` · PPTX)과 `build --check-layout`의 판정은 여전히 브라우저가 잰 값에서 나온다 —
이 명령은 그 앞의 빠른 확인이다.

글꼴은 시스템에 설치된 것(fontconfig 목록)에서 고른다. 테마 글꼴이 없으면 조용히 비슷한 글꼴로
물러선다 — 글꼴이 빠졌는지 확인하는 것은 `build`의 [글꼴 경고](#글꼴-경고)다.

### 아직 아닌 것

등장 효과의 중간 프레임(GIF는 슬라이드와 `reveal` 스텝 단위다), 컬러 이모지, SVG 그림 안의 글자
(`<text>`)는 그리지 않는다. 고칠 때마다 다시 그리는 감시 모드는 없다 (고치면 `snap`을 다시 돈다).

## `balpyo theme`

GitHub 저장소나 이 기계의 디렉토리에서 테마를 설치하고, 설치한 테마를 다룬다. 설치한 테마는
`~/.balpyo/themes/<이름>/`에 놓이고(`BALPYO_HOME`이 있으면 그 디렉토리 아래), `balpyo.json`의
`theme`와 `--theme`가 그 **이름**을 받는다. 이름은 테마의 `theme.json`이 정한다. 규칙 전체는
[설치한 테마](theme.md#설치한-테마-github에서-받는다)에 있다.

```bash
balpyo theme install <GitHub URL | path> [name...] [--force]
balpyo theme list [--project .]
balpyo theme info <name> [--project .]
balpyo theme remove <name...>
balpyo theme update [name...]
```

| 명령 | 하는 일 |
|---|---|
| `install` | 저장소를 받아(경로면 그 디렉토리를 복사해) 테마를 설치한다. 이름을 주지 않으면 찾은 테마 전부, 주면 그 테마와 같은 저장소 안의 `extends` 부모. `--force`는 다른 출처에서 설치한 같은 이름을 갈아 끼운다 |
| `list` | 쓸 수 있는 테마를 자리별로 나열한다 — 내장, 설치한 테마(출처·커밋), `--project`의 `themes/`. 겹치는 이름을 표시한다 |
| `info` | 테마 하나가 어디서 왔는지(내장 · `--project`의 `themes/` · 설치한 테마)와 부모를 적고, 테마의 `README.md`를 그대로 출력한다 |
| `remove` | 설치한 테마를 지운다. 남는 설치한 테마가 상속하는 테마는 지우지 않는다 |
| `update` | 설치한 테마를 기록한 출처·ref에서 다시 받는다 (경로에서 설치한 테마는 그 디렉토리에서 다시 복사한다) (이름을 주지 않으면 출처를 아는 테마 전부). 같은 저장소의 테마는 한 번에 받는다 |

받는 주소 모양:

| 주소 | 뜻 |
|---|---|
| `github.com/acme/themes` | 저장소 루트, 기본 브랜치 |
| `https://github.com/acme/themes.git` · `git@github.com:acme/themes.git` | 같다 (스킴·`.git`은 떼어 낸다) |
| `github.com/acme/themes@v1.2.0` | 브랜치·태그·커밋 |
| `github.com/acme/themes/brand/dark@v1.2.0` | 하위 경로 아래만 찾는다 |
| `https://github.com/acme/themes/tree/main/brand` | 브라우저 주소창의 모양 (ref에 `/`가 있으면 `@ref`를 쓴다) |

`git`은 필요 없다 — 저장소를 GitHub의 tarball 하나로 받는다. 공개 저장소만 받는다. 설치한 뒤
빌드는 네트워크를 쓰지 않는다.

받는 경로 모양 — 이 기계의 디렉토리에서 파일을 **복사**해 설치한다 (원본을 가리키는 링크가 아니다):

| 경로 | 뜻 |
|---|---|
| `./brand` · `../themes` · `/srv/themes` · `~/themes` | 언제나 경로다 |
| `my-themes` | GitHub 주소가 아니고 그런 디렉토리가 있으면 경로다 |
| `./github.com/acme/themes` | 주소처럼 생긴 경로는 `./`를 붙인다 (`github.com/…`는 언제나 주소로 읽는다) |

경로는 테마 디렉토리 하나(`theme.json`이 있는 곳)를 가리켜도, 테마 여럿을 담은 디렉토리를 가리켜도 된다 —
찾는 규칙은 저장소와 같다. 비공개 저장소나 GitHub 밖의 호스트에 있는 테마는 체크아웃한 디렉토리를
경로로 설치한다.

```
$ balpyo theme install github.com/acme/slide-themes@v1.2.0
installed 2 themes from github.com/acme/slide-themes@v1.2.0 (3f2a91c) → /home/me/.balpyo/themes
  acme       themes/light
  acme-dark  themes/dark
use the name in balpyo.json, e.g. "theme": "acme"
see: balpyo theme info acme
```

```
$ balpyo theme install ./brand
copied and installed 1 theme from /home/me/work/brand → /home/me/.balpyo/themes
  acme  .
use the name in balpyo.json, e.g. "theme": "acme"
see: balpyo theme info acme
```

`info`는 이름을 빌드와 같은 규칙으로 찾는다 — `balpyo.json`에 적을 수 있는 이름이면 무엇이든 받는다.
첫 줄은 자리, 부모가 있으면 둘째 줄은 부모를 볼 명령이고, 빈 줄 뒤가 `README.md`의 내용 그대로다.
부모의 README는 합치지 않는다. README가 없는 테마는 `this theme has no README.md`를 적고 성공한다.
무엇을 적는지는 [테마 패키지](theme.md#readmemd-덱-저자에게-쓰는-글)에 있다.

```
$ balpyo theme info acme-dark
acme-dark — installed theme (/home/me/.balpyo/themes/acme-dark)
extends acme — see: balpyo theme info acme

# acme-dark

The dark variant of acme for stages and demos. …
```

종료 코드는 성공 0, 실패 1이다. 실패한 설치는 홈을 바꾸지 않는다.

## `balpyo version`

버그 리포트에 그대로 붙일 수 있는 **세 줄**을 출력한다 — 어느 버전인지, 어느 커밋인지(그리고
그 나무가 깨끗했는지), 어디서 돈 바이너리인지:

```
$ balpyo version
balpyo v0.3.0
commit feda13c (2026-09-21T01:27:36Z)
linux/amd64 · go1.26
```

| 줄 | 사실 | 출처 |
| --- | --- | --- |
| `balpyo v0.3.0` | 릴리스 태그 | 릴리스 빌드가 주입한다 (`-X main.version=`) |
| `commit feda13c (…)` | 커밋과 그 커밋의 시각 | Go 툴체인이 박는 `vcs.revision`·`vcs.time` |
| `linux/amd64 · go1.26` | OS/아키텍처와 툴체인 | 실행 중인 런타임 |

- **커밋은 주입하지 않는다.** 툴체인이 박고 `-trimpath`도 그것을 지우지 않으므로,
  `go install`로 받은 바이너리도 자기 커밋을 안다.
- **커밋되지 않은 트리에서 빌드하면 그렇게 말한다**: `commit feda13c+dirty (…)`.
- **아는 것만 적는다.** 태그도 모듈 버전도 없으면 첫 줄이 `balpyo (devel)`이고, 커밋을 모르는
  빌드(예: `go build`로 만든 저장소 밖의 사본)는 **그 줄이 없다** — `알 수 없음`을 찍으면
  리포트에 쓸모없는 줄이 하나 늘 뿐이다. 마지막 줄은 언제나 있다.

## `balpyo help`

위 사용법을 출력한다. 알 수 없는 서브커맨드는 사용법을 stderr에 찍고 코드 1로 끝난다.
