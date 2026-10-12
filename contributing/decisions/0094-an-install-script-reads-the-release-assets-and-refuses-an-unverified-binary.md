# 0094 — 설치 스크립트는 릴리스 자산을 읽고, 체크섬이 맞지 않는 바이너리는 설치하지 않는다

- 상태: 확정
- 날짜: 2026-10-12
- 관련: [ADR 0061](0061-release-is-built-by-make-and-the-go-toolchain-and-the-version-belongs-to-the-tag.md)(릴리스 절차),
  [릴리스](../internals/release.md), [시작하기](../../docs/guides/getting-started.md)

## 맥락

릴리스는 플랫폼별 아카이브(`balpyo_<태그>_<os>_<arch>.tar.gz`)와 `SHA256SUMS`를 GitHub Releases에
놓는다(0061). 받는 사람의 절차는 "내 플랫폼 이름을 고르고, 둘을 받고, 체크섬을 맞추고, 풀고, `PATH`에
놓는다"이고, 그 가운데 사람이 가장 자주 건너뛰는 것이 체크섬이다. Homebrew tap은 macOS·Linux의 한
길이지만 tap에 포뮬러를 올리는 일은 사람의 일이고 아직 열리지 않았다.

질문은 셋이다: **스크립트가 어디에 사는가**, **최신 버전을 어떻게 아는가**, **검증할 수 없을 때 무엇을 하는가**.

## 선택지

**(1) 자리**

- (가) 릴리스 자산으로 첨부한다.
- (나) **저장소 루트의 `install.sh`** — `raw.githubusercontent.com/…/main/install.sh`로 받는다.

**(2) 최신 버전**

- (가) GitHub API(`/repos/…/releases/latest`)의 JSON에서 `tag_name`을 읽는다.
- (나) `releases/latest`의 리다이렉트를 따라가 태그를 읽는다.
- (다) **`releases/latest/download/SHA256SUMS`를 받아 그 안의 아카이브 이름에서 태그를 읽는다.**

**(3) 검증 도구가 없을 때**

- (가) 경고하고 설치한다.
- (나) **멈춘다.**

## 결정

1. **(1)은 (나)다.** 스크립트는 `main`의 것이 받힌다 — `develop`에서 고친 스크립트는 릴리스와 함께 나간다.
2. **(2)는 (다)다.** 아카이브는 그 뒤에 **태그가 박힌 URL**(`download/<태그>/…`)에서 받는다. `BALPYO_VERSION`이
   있으면 처음부터 그 태그의 경로만 쓴다.
3. **(3)은 (나)다.** `sha256sum` → `shasum -a 256` → `openssl` 순으로 찾고, 없으면 설치하지 않는다. 체크섬이
   다르면 아무것도 놓지 않고 멈춘다.
4. 기준 URL은 `BALPYO_DOWNLOAD_URL`로 바꿀 수 있다. 그 아래의 경로 모양(`download/<태그>/<파일>`,
   `latest/download/<파일>`)은 GitHub Releases의 것이다.
5. 스크립트는 linux·darwin × amd64·arm64만 다룬다. 다른 플랫폼에서는 무엇이 안 되는지 말하고 멈춘다.

## 근거

**왜 API가 아닌가.** 비인증 API 호출은 시간당 상한이 있고(CI 러너 여럿이 한 IP를 나눠 쓰면 금방 닿는다),
JSON을 `jq` 없이 읽으려면 `sed`로 긁어야 한다. 체크섬 파일은 어차피 받아야 하므로 버전을 거기서 읽으면
요청이 하나 줄고 파싱도 "공백으로 나뉜 두 칸"이다. 리다이렉트를 따라가는 (나)는 `curl`과 `wget`이
최종 URL을 알려 주는 방법이 달라 두 갈래 코드가 된다.

**왜 아카이브를 태그 URL에서 받는가.** `latest`를 두 번 부르면 그 사이에 새 릴리스가 발행될 수 있고, 그러면
체크섬과 아카이브가 서로 다른 릴리스에서 온다 — 정직한 불일치지만 사람에게는 변조와 구별되지 않는다.

**왜 멈추는가.** 확인하지 않은 바이너리를 설치하는 것이 스크립트가 막으려는 바로 그 일이다.

## 결과

- **아카이브 이름과 `SHA256SUMS`의 형식은 이제 설치 스크립트의 입력이다.** 둘을 바꾸는 커밋은
  `internal/release`의 설치 테스트를 깬다 — 테스트가 `release.Build`로 호스트 타깃의 실물을 만들고,
  httptest가 GitHub Releases의 경로로 내주고, 스크립트를 `sh`로 돌려 설치된 바이너리를 실행한다.
- `releases/latest`는 초안과 사전 릴리스를 가리키지 않는다. 사전 릴리스는 `BALPYO_VERSION`으로만 설치된다.
- 첫 태그가 발행되기 전에는 스크립트가 "최신 릴리스의 SHA256SUMS를 받을 수 없다"로 멈춘다.
