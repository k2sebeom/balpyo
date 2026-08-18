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

값 우선순위는 **CLI 플래그 > `balpyo.json` > 기본값**이다. `balpyo.json`은 선택이다 —
없으면 기본값을 쓰고 덱 제목은 디렉토리 이름에서 유도한다.

`--target html,pptx`처럼 PPTX가 포함되고 덱에 좌표가 필요한 요소가 있으면 빌드가
지오메트리 패스를 위해 시스템 브라우저(Chromium/Chrome)를 띄운다 —
[지오메트리 패스](../internals/geometry-pass.md) 참조. `--target html`만 쓰는 저작
루프는 브라우저 비용을 내지 않는다.

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

## `balpyo version`

버전을 출력한다. 릴리스 빌드는 주입된 버전을, 개발 빌드는 Go 모듈 정보를 쓴다
(`balpyo (devel)`).

## `balpyo help`

위 사용법을 출력한다. 알 수 없는 서브커맨드는 사용법을 stderr에 찍고 코드 1로 끝난다.
