# CLI

바이너리 하나가 모든 서브커맨드를 담는다 (PRD §10). 지금 구현된 표면은 `build`와
`version`이다 — `init`/`server`/`check`/`format`/`lsp`는 Phase 2에서 열리며, 그때 이
문서에 함께 실린다.

## `balpyo build`

프로젝트를 읽어 타깃별 산출물을 쓴다.

```bash
balpyo build --project testdata/fixtures/01_text --target html,pptx
```

```
슬라이드 3장 → /path/to/dist
  index.html
  text-deck.pptx
```

### 옵션

| 옵션 | 기본값 | 뜻 |
|---|---|---|
| `--project <dir>` | `.` | 프로젝트 루트 — `balpyo.json`과 `slides/`가 있는 곳 |
| `--target <목록>` | `html` | 렌더 타깃. 쉼표로 여러 개: `html,pptx` |
| `--out <dir>` | `balpyo.json`의 `outDir` (기본 `dist`) | 출력 디렉토리. 상대 경로는 프로젝트 루트 기준 |
| `--strict` | 꺼짐 | 경고를 에러로 승격한다 |
| `--check-layout` | 꺼짐 | 배치를 검사한다 (넘침·스테이지 이탈·겹침). 브라우저를 띄운다 |

값 우선순위는 **CLI 플래그 > `balpyo.json` > 기본값**이다. `balpyo.json`은 선택이다 —
없으면 기본값을 쓰고 덱 제목은 디렉토리 이름에서 유도한다.

`--target html,pptx`처럼 PPTX가 포함되고 덱에 좌표가 필요한 요소가 있으면 빌드가
지오메트리 패스를 위해 시스템 브라우저(Chromium/Chrome)를 띄운다 —
[지오메트리 패스](../internals/geometry-pass.md) 참조. `--target html`만 쓰는 저작
루프는 브라우저 비용을 내지 않는다. `--check-layout`은 그 루프에서도 측정을 켜는
플래그다 (아래 배치 경고).

### 웹 산출물에서의 발표 조작

`index.html`은 자체 완결형 발표자 뷰다 — 파일을 브라우저로 열면 슬라이드 한 장이 화면에
맞춰 나온다 (오프라인 발표 가능, PRD §9.1).

| 조작 | 방법 |
|---|---|
| 슬라이드 이동 | 하단 컨트롤 바의 ‹ › 버튼, 또는 ←/→·↑/↓·Space·PageUp/PageDown·Home/End |
| 발표자 노트 | 컨트롤 바의 **Notes** 버튼 또는 `n` — 현재 슬라이드의 `<notes>`를 패널로 연다 |
| 딥링크 | `index.html#/<슬라이드 id>` 또는 `#/<1-based 번호>` — 이동하면 주소가 따라온다 |

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
slides/002_overview.slide:3:9: error: 알 수 없는 role "heding" (쓸 수 있는 역할: title, heading, body, caption, code)
```

형태는 `파일:행:열: 심각도: 메시지`다. 행·열은 1-based이고 열은 룬(문자) 기준이다 —
한글 한 글자가 세 칸으로 세어지지 않는다. 심각도는 `error`(빌드 실패)와 `warning`
(통과하되 출력, `--strict`가 에러로 승격) 두 가지다. 경고는 빌드가 실패한 경우에도
출력된다.

각 진단은 **정확히 한 번** 찍힌다. 빌드가 진단 에러로 실패하면 stderr에 진단들이
한 줄씩 나온 뒤 마지막에 개수 요약 한 줄(`빌드 실패: 에러 N건`)이 붙는다 — 같은 진단을
통째로 다시 찍지 않는다. 설정·파일 같은 진단 아닌 에러는 그 에러 메시지가 그대로 나온다.

지금 어휘에서 경고가 나오는 곳은 테마에 없는 `variant`와 **배치**다. variant 폴백 규칙은
[외관 속성](appearance.md)의 캐스케이드 절에 있다:

```
slides/006_variant.slide:6:16: warning: variant "hero"가 테마에 없다 — 기본 외관으로 계속한다 (이 테마의 variant: kicker, lead)
```

이 빌드는 코드 0으로 통과하고 산출물이 쓰인다. `--strict`를 켜면 같은 진단이 에러가
되어 빌드가 실패한다 — 테마 어휘 밖의 variant를 잡고 싶은 CI에서 쓴다.

### 배치 경고

빌드가 좌표를 재면(`--target`에 `pptx`가 있거나 `--check-layout`) 잰 박스에서 배치 사고를
찾아 경고한다. 네 가지를 보고, 넘친 양을 논리 px로 말한다:

| 경고 | 판정 |
|---|---|
| 스테이지 이탈 | 박스가 논리 스테이지(1280×720) 밖으로 나갔다 |
| 부모 박스 이탈 | 박스가 부모 박스를 넘쳤다 — 크기를 정한 도형에 내용이 담기지 않은 모양이다 |
| 콘텐츠 박스 이탈 | 슬라이드 직계 박스가 스테이지 여백(64px)을 침범했다 |
| 형제 겹침 | 같은 부모의 두 박스가 서로 겹쳤다 |

```
slides/003_kpis.slide:7:6: warning: <p>가 부모 <shape>를 아래로 34px 넘친다 — 부모 박스 크기가 내용을 담기에 모자란다
```

의도한 겹침과 의도한 이탈은 묻지 않는다: `at`·`x`·`y`를 쓴 요소(와 그 자손), `layout="canvas"`
슬라이드의 요소는 부모 변과 여백을 넘어도 조용하고, `<stack>`·canvas의 겹침도 조용하다.
스테이지 이탈만은 예외가 없다 — 화면 밖은 웹에서 잘리고 PPTX에서는 슬라이드 밖에 놓인다.
판정의 근거와 한계는 [ADR 0039](../decisions/0039-layout-warnings-are-box-containment.md)에
있고, 경고를 읽고 고치는 순서는 [배치 경고 읽기](../guides/layout-warnings.md)에 있다.

```bash
# 저작 루프에서 한 번 확인한다 (브라우저를 띄운다)
balpyo build --check-layout

# CI: 넘친 덱을 통과시키지 않는다
balpyo build --target html,pptx --strict
```

## `balpyo version`

버전을 출력한다. 릴리스 빌드는 주입된 버전을, 개발 빌드는 Go 모듈 정보를 쓴다
(`balpyo (devel)`).

## `balpyo help`

위 사용법을 출력한다. 알 수 없는 서브커맨드는 사용법을 stderr에 찍고 코드 1로 끝난다.
