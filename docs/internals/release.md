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
어떻게 조립되는지는 [`balpyo version`](../reference/cli.md#balpyo-version)에 있다.

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

## 태그가 릴리스를 만든다

`.github/workflows/release.yml`이 `v*` 태그에서 돈다:

1. **게이트를 다시 돈다** (`make test`). 태그가 main의 초록 커밋을 가리킨다고 믿지
   않는다 — 발표한 아티팩트는 되돌릴 수 없으므로 3분을 다시 쓴다.
2. `make dist VERSION=<태그>`로 아카이브를 만들고 `sha256sum -c SHA256SUMS`로 확인한다.
3. **초안**(draft) 릴리스를 만들어 아카이브·`SHA256SUMS`·`balpyo.rb`를 첨부한다. 릴리스 노트는
   사람이 쓴다 (커밋 메시지는 릴리스 노트가 아니다).

릴리스를 내는 사람의 절차는 네 걸음이다: 태그를 붙여 밀고 → 워크플로가 만든 초안에 노트를 쓰고
→ 발행하고 → tap 저장소에 `balpyo.rb`를 올린다.

## 릴리스 빌드는 호스트의 툴체인에서 돈다

`make dist`는 `make test`와 같은 Go(`go.mod`의 버전 — CI는 `setup-go`의 `go-version-file`)로
돈다. 재현성 계약(같은 태그 → 같은 바이트)을 지키는 것은 툴체인이다: `internal/release`가
`CGO_ENABLED=0` · `-trimpath`로 빌드하고 주입하는 값은 태그 하나다. 한때 이 빌드는 테스트
컨테이너 안에서 돌았지만, 컨테이너가 고정하던 것은 렌더 도구(LibreOffice · 폰트)였지 Go가
아니었다 ([ADR 0065](../decisions/0065-pptx-validity-is-judged-by-the-ecma-376-schema-not-by-libreoffice.md)).
