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
dist/SHA256SUMS
dist/balpyo.rb
```

## 플랫폼 표는 넷이다

| `GOOS/GOARCH` | 아카이브 |
| --- | --- |
| `darwin/arm64` | `balpyo_<버전>_darwin_arm64.tar.gz` |
| `darwin/amd64` | `balpyo_<버전>_darwin_amd64.tar.gz` |
| `linux/amd64` | `balpyo_<버전>_linux_amd64.tar.gz` |
| `linux/arm64` | `balpyo_<버전>_linux_arm64.tar.gz` |

- **윈도우는 없다.** 근거는 없음이 아니라 **재보지 않았음**이다: 경로 구분자·CRLF·폰트 대체·
  `soffice` 위치를 한 번도 측정하지 않았고, 측정하지 않은 플랫폼의 아카이브를 놓는 것은
  "빌드된다"를 "된다"로 발표하는 일이다.
- 표는 `release.Targets()`에 있고 **테스트가 그 목록을 판정한다** — 타깃을 지우거나 더하는
  커밋은 테스트를 먼저 고쳐야 하고, 그 자리에서 "재봤는가"를 묻게 된다.
- 아카이브 안에는 **바이너리 하나**만 있다. 저자가 `tar xzf` 후 찾을 것이 하나다.

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

테스트가 한 타깃에서 이것을 판정한다 (두 번 빌드해 아카이브의 해시를 맞댄다). 그래서 "결정적
아카이브"는 주장이 아니라 게이트다.

## Homebrew 포뮬러는 생성물이다

`dist/balpyo.rb`는 체크섬에서 나온다 — OS × 아키텍처 네 칸이 각자의 URL과 `sha256`을 갖고,
`test` 스탠자가 설치된 바이너리를 **실제로 돌려** 버전을 맞춘다.

- **네 타깃을 모두 만들 때만 나온다.** 한 타깃만 만든 빌드에서 포뮬러를 내면 나머지 세 칸이
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

```
feature/* · fix/* · doc/*  ──PR──▶  develop  ──릴리스 PR (merge commit)──▶  main  ──태그 vX.Y.Z──▶  Release
                                       ▲                                     │
                                       └──────── fast-forward ◀──────────────┘
```

- `develop`이 통합 브랜치다. `feature/*` · `fix/*` · `doc/*`는 `develop`에서 갈라지고, PR도
  `develop`을 향한다. 작은 문서 수정은 `develop`에 바로 커밋해도 된다.
- 릴리스는 `develop`을 `main`에 머지하는 PR 하나다. 머지 커밋에 다음 버전 태그(`vX.Y.Z`)를
  붙이는 것이 버전을 올리는 일이다 — 버전은 태그에만 있으므로 올릴 파일이 따로 없다.
- 릴리스 PR은 **merge commit으로** 머지한다. squash · rebase 머지는 `main`에 SHA가 다른 커밋을
  만들어, `develop`의 커밋이 `main`에 들어간 적이 없는 것처럼 보이게 한다 — 다음 릴리스 PR에
  이미 나간 변경이 다시 나오고 충돌한다.
- 머지 뒤에는 `develop`을 `main`으로 **fast-forward**한다. 그러지 않으면 머지 커밋 하나만큼
  `main`이 `develop`보다 앞선다:
  `git checkout develop && git merge --ff-only origin/main && git push origin develop`
- **hotfix**(태그를 붙이기 전에 `main`에서 고쳐야 하는 것)는 `main`에서 `fix/*`를 갈라 `main`으로
  PR을 내고, 머지한 뒤 같은 fast-forward로 `develop`에 돌려준다.
- CI는 두 브랜치의 push와 모든 PR에서 돈다.

### 브랜치 보호

| 브랜치 | 규칙 |
| --- | --- |
| `main` | PR로만 들어온다 (승인 0 — 혼자 관리하면 자기 PR을 승인할 수 없다). `테스트` · `문서 빌드`가 초록이고 base가 최신이어야 머지된다. force push · 삭제 금지. 관리자는 예외로 둔다 (긴급 수정용). |
| `develop` | 직접 push는 된다. force push · 삭제 금지. |

저장소는 머지한 PR의 head 브랜치를 자동으로 지운다. 릴리스 PR의 head는 `develop`이므로, 보호가
없으면 릴리스 머지가 `develop`을 지우고 그것을 base로 하던 PR들은 `main`으로 옮겨진다. 보호된
브랜치는 자동 삭제되지 않는다.

## 태그가 릴리스를 만든다

`main`에 머지하는 순간 `.github/workflows/docs.yml`이 문서 사이트를 GitHub Pages에 올리고,
`install.sh`의 `main` URL이 새 스크립트를 내준다. 그 뒤 `.github/workflows/release.yml`이 `v*`
태그에서 돈다:

1. **게이트를 다시 돈다** (`make test` — CI와 같은 테마 글꼴을 깐다). 태그가 main의 초록 커밋을
   가리킨다고 믿지 않는다 — 발표한 아티팩트는 되돌릴 수 없으므로 3분을 다시 쓴다.
2. `make dist VERSION=<태그>`로 아카이브를 만들고 `sha256sum -c SHA256SUMS`로 확인한다.
3. **초안**(draft) 릴리스를 만들어 아카이브·`SHA256SUMS`·`balpyo.rb`를 첨부한다. 릴리스 노트는
   사람이 쓴다 (커밋 메시지는 릴리스 노트가 아니다).

릴리스를 내는 사람의 절차:

1. `develop` → `main` 릴리스 PR을 열고, CI가 초록이면 **merge commit**으로 머지한다.
2. `develop`을 `main`으로 fast-forward한다.
3. `main`의 머지 커밋에 태그를 붙여 민다: `git tag -a vX.Y.Z -m vX.Y.Z && git push origin vX.Y.Z`.
4. 워크플로가 만든 초안에 노트를 쓰고 **발행한다** — 발행해야 `releases/latest`가 그 릴리스를
   가리키고, 버전을 고르지 않은 설치 스크립트가 그것을 받는다.
5. 발행한 뒤 `curl -fsSL https://raw.githubusercontent.com/k2sebeom/balpyo/main/install.sh | sh`로
   설치해 `balpyo version`을 확인한다.

게이트가 실패하면 초안은 생기지 않는다 — 고친 뒤 태그를 지우고(`git push origin :vX.Y.Z`) 다시 민다.
Homebrew tap은 아직 없다: `balpyo.rb`는 생성되어 첨부되지만 올릴 곳이 없다.

## 릴리스 빌드는 호스트의 툴체인에서 돈다

`make dist`는 `make test`와 같은 Go(`go.mod`의 버전 — CI는 `setup-go`의 `go-version-file`)로
돈다. 재현성 계약(같은 태그 → 같은 바이트)을 지키는 것은 툴체인이다: `internal/release`가
`CGO_ENABLED=0` · `-trimpath`로 빌드하고 주입하는 값은 태그 하나다. 한때 이 빌드는 테스트
컨테이너 안에서 돌았지만, 컨테이너가 고정하던 것은 렌더 도구(LibreOffice · 폰트)였지 Go가
아니었다 ([ADR 0065](../decisions/0065-pptx-validity-is-judged-by-the-ecma-376-schema-not-by-libreoffice.md)).
