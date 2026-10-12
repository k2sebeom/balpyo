# 0095 — 윈도우는 CI가 같은 게이트로 재고, `windows/amd64`를 zip으로 발표한다

- 상태: 확정 (0061의 결정 3 "플랫폼 표는 넷이다"와 결정 6의 아카이브 형식을 대체한다)
- 날짜: 2026-10-12
- 관련: [ADR 0061](0061-release-is-built-by-make-and-the-go-toolchain-and-the-version-belongs-to-the-tag.md)(릴리스 절차),
  [ADR 0094](0094-an-install-script-reads-the-release-assets-and-refuses-an-unverified-binary.md)(설치 스크립트),
  [릴리스](../internals/release.md), [테스트 전략](../internals/testing.md),
  [시작하기](../../docs/guides/getting-started.md)

## 맥락

0061은 윈도우를 표에서 뺐다. 근거는 "없음이 아니라 **재보지 않았음**"이었고, 재볼 목록으로 경로
구분자 · CRLF · 폰트 대체 · `soffice` 위치를 들었다. 크로스 컴파일은 처음부터 됐다
(`GOOS=windows go vet ./...`도 깨끗하다) — 막혀 있던 것은 "된다"를 말할 측정이었다.

그래서 이번에는 먼저 쟀다. GitHub의 `windows-latest` 러너에서 리눅스 잡과 **같은 게이트**
(`go test ./...`, `BALPYO_TEST_STRICT=1`)를 돌렸고, 처음 실행에서 깨진 것은 이랬다:

| 깨진 것 | 원인 | 고친 자리 |
| --- | --- | --- |
| 브라우저를 못 찾는다 (`BALPYO_CHROMIUM=…\chrome.exe`도 "not executable") | Chrome · Edge 설치 프로그램은 PATH에 실행 파일을 두지 않는다. Go는 윈도우 파일에 실행 비트를 주지 않는다 (0666/0444) | `render/geom`: 설치 폴더 환경 변수(`ProgramFiles` · `ProgramFiles(x86)` · `LocalAppData`)에서 후보를 만들고, 윈도우에서는 확장자(`.exe` · `.com` · `.bat` · `.cmd`)로 실행 파일을 판정한다. Edge는 Chrome 뒤의 후보다 |
| 모든 골든 · 스킬 골든 · `theme info` 출력이 다르다 | git의 `core.autocrlf=true`가 체크아웃에서 LF를 CRLF로 바꾼다 — 골든만이 아니라 `go:embed`로 바이너리에 묻는 내장 테마 · 스킬 파일까지 | `.gitattributes`: `* text=auto eol=lf` (벤더한 ECMA-376 XSD는 받은 그대로 `-text`) |
| `Noto Sans Display`가 "측정 브라우저에 없다" · ec2-intro의 도형이 무대 아래로 16~18px 넘친다 · 내장 렌더러의 드리프트가 84.6% | noto-fonts의 최신 커밋은 Display를 "Noto Sans"의 하위 가족(이름 표의 typographic family)으로 옮겼고, 가족을 그 이름으로 묶는 DirectWrite에는 그 가족이 없다. 글꼴 세대가 리눅스 패키지와 달랐다 | CI: 데비안 · 우분투 패키지와 같은 세대(noto-fonts `v20201206-phase3`, noto-cjk의 OTC)를 고정 커밋에서 받아 깐다 |
| OnlyOffice 탐색 · `x2t` 설정 문서 테스트 | 도구 이름에 `.exe`가 없었고, 테스트가 `/` 구분자를 박아 두었다 | `internal/preview`: 설치 디렉토리 안의 도구는 `.exe`로 부르고, Program Files 아래의 OnlyOffice · LibreOffice를 후보에 둔다 |
| 기동 재시도 테스트 · 도구 탐색 테스트 · 릴리스 실행 테스트 | 가짜 도구가 `#!/bin/sh` 스크립트였고 `sh`를 PATH에서 찾았다 | 테스트: 윈도우에서는 배치 파일 · `cmd`를 쓴다 |

이것들을 고친 뒤 윈도우 잡은 **건너뛰기 없이** 초록이다 — 동일성 · 골든 · 결정성 · PPTX 유효성 ·
예제의 경고 0 · 내장 렌더러의 드리프트까지 리눅스와 같은 판정이다. OnlyOffice · LibreOffice를
부르는 래스터 테스트는 리눅스에서와 같은 정책(도구가 없으면 엄격 모드에서도 건너뛴다 — 게이트가
아닌 도구다)으로 건너뛴다. 러너에 그 도구가 없으므로 `balpyo preview`의 PPTX 래스터는 윈도우에서
재지 않았다.

질문은 셋이다: **무엇을 표에 올리는가**, **어떤 형식으로 놓는가**, **CRLF를 어디서 받는가**.

## 선택지

**(1) 표**

- (가) `windows/amd64`와 `windows/arm64`를 함께 올린다 — 크로스 컴파일은 둘 다 된다.
- (나) **`windows/amd64`만 올린다** — 게이트를 도는 러너가 있는 것만.

**(2) 형식**

- (가) 다른 플랫폼처럼 `.tar.gz`.
- (나) **`.zip` 안에 `balpyo.exe` 하나.**

**(3) CRLF**

- (가) 골든 비교에서 CRLF를 LF로 접는다.
- (나) **저장소의 체크아웃을 LF로 고정하고, CRLF 입력은 테스트가 만들어서 판정한다.**

## 결정

1. **플랫폼 표는 다섯이다**: `darwin/arm64` · `darwin/amd64` · `linux/amd64` · `linux/arm64` ·
   `windows/amd64`. `windows/arm64`는 올리지 않는다 — 그것을 도는 러너가 없으므로 재보지 않은
   것이고, 0061의 기준(재본 것만 놓는다)은 그대로다.
2. **윈도우 아카이브는 `balpyo_<버전>_windows_amd64.zip`이고 안에는 `balpyo.exe` 하나다.** zip은
   tar.gz와 같은 재현성 계약을 진다: 헤더 시각은 1980-01-01(MS-DOS 시각이 담을 수 있는 가장 이른
   날 — PPTX 패키지와 같은 값), 모드는 0755, 압축은 Deflate. `SHA256SUMS`에 다른 아카이브와 함께
   한 줄로 들어간다.
3. **Homebrew 포뮬러는 macOS · 리눅스 네 칸만 덮는다.** 포뮬러가 요구하는 아티팩트도 그 넷이고,
   윈도우 아카이브는 있어도 쓰지 않는다. 포뮬러 파일은 지금처럼 표를 전부 만든 빌드에서 나온다.
4. **윈도우 잡이 CI의 게이트다.** 리눅스 잡과 같은 명령 · 같은 엄격 모드 · 같은 글꼴 가족과
   굵기다. 글꼴은 고정 커밋에서 받아 캐시하는 공용 액션(`.github/actions/windows-fonts`)이 깐다.
   **릴리스도 윈도우 게이트를 다시 돈다** — 아카이브 잡이 그 잡을 기다린다. 아카이브 자체는 지금처럼
   리눅스 잡이 크로스 컴파일한다.
5. **체크아웃은 어느 운영체제에서나 LF다** (`.gitattributes`). 저자가 CRLF로 쓴 `.slide`는
   파서가 받고, 그 사실은 픽스처를 CRLF로 바꿔 빌드한 구조 스냅샷 · 진단이 LF 골든과 바이트까지
   같다는 테스트가 판정한다.
6. **설치 스크립트는 윈도우를 다루지 않는다** (0094 그대로). 윈도우 사람의 절차는 zip과
   `SHA256SUMS`를 받아 `Get-FileHash`로 맞추고, 풀어서 PATH에 두는 것이다 — 시작하기 문서가 그
   절차를 적는다. PowerShell 설치기는 이번 결정의 범위가 아니다.

## 근거

- **"재봤다"의 뜻을 넓히지 않았다.** 0061이 윈도우를 미룬 이유는 측정이 없다는 것이었고, 이번에
  더한 것은 측정이다 — 게이트를 낮추지 않았다(건너뛰기를 늘리지 않았다). 그래서 arm64는 같은 이유로
  남는다.
- **zip인 이유는 받는 사람이다.** 윈도우 탐색기와 PowerShell(`Expand-Archive`)이 따로 도구 없이
  여는 형식이 zip이다. 실행 파일에 `.exe`가 붙는 이유도 같다 — 윈도우는 확장자로 실행 파일을
  알아본다.
- **CRLF를 비교에서 접지 않는 이유는 골든만의 일이 아니기 때문이다.** autocrlf는 바이너리에 묻는
  파일(내장 테마 · 스킬)도 바꾼다. 비교에서 접으면 테스트는 초록이 되지만 윈도우에서 빌드한
  바이너리는 다른 바이트를 담는다. 체크아웃을 고정하면 저장소의 바이트가 운영체제에서 떨어지고,
  저자 입력의 CRLF는 그것대로 테스트가 만든다.
- **글꼴을 고정 커밋에서 받는 이유는 같은 게이트이기 위해서다.** 예제 게이트는 경고 0을 요구하고
  측정은 글꼴에 매여 있다. 실측이 그것을 보였다 — 세대가 다른 Noto 하나가 가족 이름 하나와 도형
  하나의 넘침과 드리프트 15%를 낳았다. 리눅스 패키지와 같은 세대를 받으면 두 러너가 같은 것을 잰다.

## 결과

**열리는 것**

- `make dist`가 아카이브 다섯 + `SHA256SUMS` + `dist/balpyo.rb`를 낸다. 릴리스 초안에 `.zip`이
  함께 첨부된다.
- 윈도우에서 `balpyo build`가 Chrome을, 없으면 Edge를 설치 폴더에서 찾는다 — 윈도우 10 · 11에는
  Edge가 언제나 있다.
- CI의 모든 PR이 윈도우에서도 같은 판정을 받는다.

**닫히는 것 (근거가 생기면 다시 연다)**

- **`windows/arm64`** — 그 러너에서 같은 게이트를 돌 수 있게 되면 표에 한 줄 더한다.
- **PowerShell 설치기 · winget · scoop** — 설치 경로는 손 절차 하나다.
- **윈도우에서의 `balpyo preview` PPTX 래스터** — 후보 경로는 두었지만 러너에 OnlyOffice ·
  LibreOffice가 없어 재지 않았다.
