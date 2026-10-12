# 릴리스

저자가 이 도구를 **받는** 방법을 만드는 쪽이다. 절차는 goreleaser 같은 별도 도구가 아니라
`internal/release` 패키지가 소유하고, `make dist`는 그것을 부르는 한 줄이다. 근거는
[ADR 0061](../decisions/0061-release-is-built-by-make-and-the-go-toolchain-and-the-version-belongs-to-the-tag.md)에
있다 — 요지는 "쉘에 든 절차는 테스트가 읽지 않는다"와 "환경은 하나"다.

```
make dist                             # HEAD에 붙은 태그로 전부 만든다
make dist VERSION=v0.3.0              # 태그를 직접 준다
make dist TARGETS=linux/amd64         # 한 타깃만 (포뮬러는 나오지 않는다)
```

산출물은 `dist/`에 놓인다 (`.gitignore`에 있다 — 생성물은 커밋하지 않는다):

```
dist/balpyo_v0.3.0_darwin_arm64.tar.gz
dist/balpyo_v0.3.0_darwin_amd64.tar.gz
dist/balpyo_v0.3.0_linux_amd64.tar.gz
dist/balpyo_v0.3.0_linux_arm64.tar.gz
dist/balpyo_v0.3.0_windows_amd64.zip
dist/SHA256SUMS
dist/balpyo.rb
```

## 플랫폼 표는 다섯이다

| `GOOS/GOARCH` | 아카이브 | 안의 파일 |
| --- | --- | --- |
| `darwin/arm64` | `balpyo_<버전>_darwin_arm64.tar.gz` | `balpyo` |
| `darwin/amd64` | `balpyo_<버전>_darwin_amd64.tar.gz` | `balpyo` |
| `linux/amd64` | `balpyo_<버전>_linux_amd64.tar.gz` | `balpyo` |
| `linux/arm64` | `balpyo_<버전>_linux_arm64.tar.gz` | `balpyo` |
| `windows/amd64` | `balpyo_<버전>_windows_amd64.zip` | `balpyo.exe` |

- 표에 오르는 조건은 "빌드된다"가 아니라 **재봤다**다 — CI가 그 플랫폼에서 같은 게이트를 돈다.
  윈도우는 `windows-latest` 러너가 리눅스와 같은 `go test ./...`(엄격 모드, 같은 Noto 글꼴)를 돌므로
  표에 있다. `windows/arm64`는 크로스 컴파일은 되지만 그것을 도는 러너가 없으므로 없다. 근거는
  [ADR 0095](../decisions/0095-windows-is-measured-in-ci-and-released-as-a-zip.md)에 있다 (0061의
  "넷"을 대체한다).
- 표는 `release.Targets()`에 있고 **테스트가 그 목록을 판정한다** — 타깃을 지우거나 더하는
  커밋은 테스트를 먼저 고쳐야 하고, 그 자리에서 "재봤는가"를 묻게 된다.
- 아카이브 안에는 **바이너리 하나**만 있다. 저자가 풀고 나서 찾을 것이 하나다. 윈도우가 zip인
  이유는 탐색기와 `Expand-Archive`가 따로 도구 없이 여는 형식이기 때문이고, `.exe`가 붙는 이유는
  윈도우가 확장자로 실행 파일을 알아보기 때문이다.

## 버전은 태그가 소유한다

릴리스 빌드가 주입하는 값은 **하나**다:

```
go build -trimpath -ldflags "-X main.version=v0.3.0" ./cmd/balpyo
```

커밋·시각·더러움은 주입하지 않는다 — Go 툴체인이 `vcs.revision`·`vcs.time`·`vcs.modified`에
박고 `-trimpath`도 그것을 지우지 않는다. 같은 사실의 두 번째 출처를 만들면, 릴리스 절차를
거치지 않은 빌드(예: `go install`)에서 리포트에 가장 필요한 줄이 가장 조용히 빈다. 세 줄이
어떻게 조립되는지는 [`balpyo version`](../../docs/reference/cli.md#balpyo-version)에 있다.

## 재현성은 계약이다

같은 태그에서 두 번 빌드하면 **같은 바이트**가 나와야 한다. 그것을 만드는 것 셋:

- `CGO_ENABLED=0` (이 모듈은 cgo를 쓰지 않는다) · `-trimpath` (빌드한 사람의 경로를 지운다)
- 주입 값은 태그 하나 (시각·호스트 이름 같은 "빌드할 때마다 다른 것"을 넣지 않는다)
- **tar 헤더를 고정한다**: 시각은 epoch 0, uid/gid는 0, 사용자 이름은 없다. gzip 헤더의 시각도
  0이다.
- **zip 헤더를 고정한다**: 시각은 1980-01-01(MS-DOS 시각이 담을 수 있는 가장 이른 날 — PPTX
  패키지와 같은 값), 모드는 0755, 압축은 Deflate.

테스트가 호스트 타깃과 `windows/amd64`에서 이것을 판정한다 (두 번 빌드해 아카이브의 해시를 맞댄다). 그래서 "결정적
아카이브"는 주장이 아니라 게이트다.

## Homebrew 포뮬러는 생성물이다

`dist/balpyo.rb`는 체크섬에서 나온다 — macOS · 리눅스의 OS × 아키텍처 네 칸이 각자의 URL과
`sha256`을 갖고 (Homebrew가 설치하지 않는 윈도우 아카이브는 들어가지 않는다),
`test` 스탠자가 설치된 바이너리를 **실제로 돌려** 버전을 맞춘다.

- **표의 타깃을 모두 만들 때만 나온다.** 한 타깃만 만든 빌드에서 포뮬러를 내면 나머지 세 칸이
  빈 `url`·빈 `sha256`인 `.rb`가 놓이고, 그 파일은 tap에 올라간 뒤 **받는 사람의 기계에서만**
  실패한다. `make dist TARGETS=…`는 포뮬러 대신 "나오지 않았다"를 출력한다.
- **tap 저장소에 올리는 것은 사람의 일이다.** 우리 저장소가 아닌 곳에 쓰는 토큰을 CI에 두지
  않는다. 릴리스 워크플로는 `.rb`를 릴리스에 첨부만 하고, `homebrew-balpyo`에 커밋하는 것은
  사람이 한다.
- 포뮬러에 `license` 스탠자가 없다 — 이 저장소에 `LICENSE` 파일이 아직 없기 때문이다
  (Homebrew의 audit은 그것을 기대한다). 라이선스를 정하면 포뮬러 생성기에 한 줄이 늘어난다.

## 설치 스크립트는 릴리스 자산을 읽는다

저장소 루트의 `install.sh`가 `curl -fsSL https://raw.githubusercontent.com/k2sebeom/balpyo/main/install.sh | sh`의
그 스크립트다. 근거는 [ADR 0094](../decisions/0094-an-install-script-reads-the-release-assets-and-refuses-an-unverified-binary.md)에
있고, 사용자 쪽 설명은 [시작하기](../../docs/guides/getting-started.md#설치-스크립트)에 있다.

- **POSIX `sh`다** (dash · macOS의 sh에서 돈다). `curl`이 없으면 `wget`, 해시는 `sha256sum` →
  `shasum -a 256` → `openssl`. `jq`도 GitHub API도 쓰지 않는다.
- 최신 버전은 `releases/latest/download/SHA256SUMS` 안의 아카이브 이름에서 읽고, 아카이브는 그 태그의
  URL(`releases/download/<태그>/…`)에서 받는다. `releases/latest`는 초안·사전 릴리스를 가리키지 않으므로
  **초안 릴리스를 발행해야** 스크립트가 그것을 본다.
- 체크섬이 다르거나 해시 도구가 없으면 **아무것도 설치하지 않고** 멈춘다. `PATH`·브라우저(빌드와 같은
  `BALPYO_CHROMIUM` → `chromium` · `chromium-browser` · `google-chrome` → macOS 설치 경로)는 경고만 한다.
- 받히는 것은 `main`의 스크립트다 — `develop`에서 고친 것은 다음 릴리스 머지에서 나간다.
- **아카이브 이름과 `SHA256SUMS` 형식은 이 스크립트의 입력이다.** `internal/release/install_test.go`가
  `release.Build`로 호스트 타깃의 실물을 만들고 httptest가 GitHub Releases의 경로로 내준 뒤,
  스크립트를 표준 입력으로 `sh`에 흘려 넣는다 (`BALPYO_DOWNLOAD_URL`이 기준 URL을 바꾼다). 판정은
  태그 고정 설치 · 최신 해석 · 체크섬 변조 거부 · 없는 태그다. `sh`·`curl`/`wget`이 없으면 건너뛴다.
- 스크립트를 고치면 `shellcheck install.sh`도 돌린다 (게이트에는 없다 — CI에 설치하지 않는다).

## 브랜치는 git flow를 따른다

- `develop`이 통합 브랜치다. `feature/*` · `fix/*` · `doc/*`는 `develop`에서 갈라지고, PR도
  `develop`을 향한다.
- 릴리스는 `develop`을 `main`에 머지하는 PR 하나다. 머지 커밋에 다음 버전 태그(`vX.Y.Z`)를
  붙이는 것이 버전을 올리는 일이다 — 버전은 태그에만 있으므로 올릴 파일이 따로 없다.
- CI는 두 브랜치의 push와 모든 PR에서 돈다.

## 태그가 릴리스를 만든다

`.github/workflows/release.yml`이 `v*` 태그에서 돈다:

1. **게이트를 다시 돈다** — 리눅스(`make test`)와 윈도우(`go test ./...`) 둘 다. 태그가 main의
   초록 커밋을 가리킨다고 믿지 않는다 — 발표한 아티팩트는 되돌릴 수 없으므로 몇 분을 다시 쓴다.
   아카이브 잡은 윈도우 게이트를 기다리고, 다섯 아카이브는 리눅스 잡이 크로스 컴파일한다.
2. `make dist VERSION=<태그>`로 아카이브를 만들고 `sha256sum -c SHA256SUMS`로 확인한다.
3. **초안**(draft) 릴리스를 만들어 아카이브(`.tar.gz` · `.zip`)·`SHA256SUMS`·`balpyo.rb`를 첨부한다. 릴리스 노트는
   사람이 쓴다 (커밋 메시지는 릴리스 노트가 아니다).

릴리스를 내는 사람의 절차는 네 걸음이다: 태그를 붙여 밀고 → 워크플로가 만든 초안에 노트를 쓰고
→ 발행하고 → tap 저장소에 `balpyo.rb`를 올린다.

## 릴리스 빌드는 호스트의 툴체인에서 돈다

`make dist`는 `make test`와 같은 Go(`go.mod`의 버전 — CI는 `setup-go`의 `go-version-file`)로
돈다. 재현성 계약(같은 태그 → 같은 바이트)을 지키는 것은 툴체인이다: `internal/release`가
`CGO_ENABLED=0` · `-trimpath`로 빌드하고 주입하는 값은 태그 하나다. 한때 이 빌드는 테스트
컨테이너 안에서 돌았지만, 컨테이너가 고정하던 것은 렌더 도구(LibreOffice · 폰트)였지 Go가
아니었다 ([ADR 0065](../decisions/0065-pptx-validity-is-judged-by-the-ecma-376-schema-not-by-libreoffice.md)).

## 윈도우 게이트

CI의 `test-windows` 잡과 릴리스의 `gate-windows` 잡이 같은 준비를 한다:

- **글꼴**: 공용 액션 `.github/actions/windows-fonts`가 리눅스 패키지(`fonts-noto-core` ·
  `-mono` · `-cjk`)와 같은 가족 · 같은 굵기를 고정 커밋에서 받아(캐시한다) 시스템 글꼴 폴더와
  레지스트리에 넣는다. 라틴은 noto-fonts `v20201206-phase3` — 데비안 패키지와 같은 세대다. 그 뒤의
  커밋은 Display를 "Noto Sans"의 하위 가족으로 옮겨서, DirectWrite에서는 `Noto Sans Display`
  가족이 사라지고 예제 게이트가 글꼴 경고와 넘침으로 깨진다.
- **브라우저**: 러너의 Chrome은 PATH에 없다. `render/geom`이 `ProgramFiles` · `ProgramFiles(x86)` ·
  `LocalAppData` 아래의 Chrome, 그다음 Edge를 찾는다. 윈도우에서는 샌드박스가 뜨므로
  `BALPYO_CHROMIUM_NO_SANDBOX`를 두지 않는다.
- **줄 끝**: `.gitattributes`가 체크아웃을 LF로 고정한다 (윈도우 git의 `core.autocrlf=true`가 골든과
  `go:embed` 파일을 CRLF로 바꾸기 때문이다). 저자가 쓴 CRLF `.slide`는
  `TestCRLFSlidesBuildTheSameDeck`이 따로 판정한다.
- 윈도우에서 실행 파일은 실행 비트가 아니라 확장자(`.exe` · `.com` · `.bat` · `.cmd`)로 판정한다
  (`geom.Executable`). 가짜 도구를 쓰는 테스트는 윈도우에서 배치 파일을 쓴다.
- 래스터 도구(OnlyOffice · LibreOffice)는 러너에 없으므로 그 테스트는 리눅스와 같은 정책으로
  건너뛴다. 후보 경로(Program Files 아래)는 있지만 윈도우에서 잰 적은 없다.
