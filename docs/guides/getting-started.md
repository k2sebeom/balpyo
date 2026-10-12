# 시작하기

빈 디렉토리에서 발표할 수 있는 덱까지. 저작 루프는 [`balpyo watch`](../reference/cli.md#balpyo-watch)
하나이고, 그 루프가 이 문서의 절반이다.

## 1. 바이너리

런타임 의존은 없다 — 바이너리 하나가 모든 서브커맨드를 담는다. Go 툴체인이 있으면 저장소에서
직접 만든다.

```bash
git clone https://github.com/k2sebeom/balpyo && cd balpyo
go build -o balpyo ./cmd/balpyo
./balpyo version
```

`go build`만으로도 그 바이너리는 자기 커밋을 안다 (툴체인이 박는다 — 위 `version`이 세 줄을
찍는다). 버그 리포트에는 그 세 줄을 그대로 붙인다:
[`balpyo version`](../reference/cli.md#balpyo-version).

### 설치 스크립트

Linux와 macOS에서는 한 줄로 설치한다:

```bash
curl -fsSL https://raw.githubusercontent.com/k2sebeom/balpyo/main/install.sh | sh
```

스크립트가 하는 일은 아래 "릴리스에서 받기"의 손 절차와 같다: 플랫폼(`uname`)에 맞는
아카이브와 `SHA256SUMS`를 GitHub Releases에서 임시 디렉토리로 받고, **체크섬을 맞춘 뒤에만**
압축을 풀어 설치한다. 체크섬이 다르면 아무것도 설치하지 않고 멈춘다. 끝에서 설치한 바이너리의
`balpyo version`을 찍는다. `curl`이 없으면 `wget`을 쓴다.

| 환경 변수 | 기본값 | 뜻 |
| --- | --- | --- |
| `BALPYO_VERSION` | 최신 릴리스 | 설치할 태그 (예: `v0.1.0`) |
| `BALPYO_INSTALL_DIR` | `$HOME/.local/bin` | 바이너리를 놓을 디렉토리 |
| `BALPYO_DOWNLOAD_URL` | `https://github.com/k2sebeom/balpyo/releases` | 릴리스를 받는 기준 URL (미러용) |

```bash
curl -fsSL https://raw.githubusercontent.com/k2sebeom/balpyo/main/install.sh \
  | BALPYO_VERSION=v0.1.0 BALPYO_INSTALL_DIR=/usr/local/bin sh
```

설치 디렉토리가 `PATH`에 없거나 Chromium/Chrome을 찾지 못하면 **경고만** 하고 설치는
끝낸다 (브라우저는 아래처럼 PPTX 타깃과 `--check-layout`에만 필요하다). 윈도우에서는
스크립트가 멈춘다 — 릴리스 페이지에서 아카이브를 직접 받는다.

### 태그된 릴리스에서 받기

`v*` 태그마다 네 플랫폼(`darwin/arm64`·`darwin/amd64`·`linux/amd64`·`linux/arm64`)의 아카이브와
체크섬 한 파일이 GitHub Releases에 올라간다. 받은 뒤 체크섬을 맞추고 압축을 풀면 그 안에는
바이너리 하나뿐이다:

```bash
tar xzf balpyo_v0.3.0_darwin_arm64.tar.gz     # → balpyo
sha256sum -c SHA256SUMS                        # 같은 디렉토리에 두고 돌린다
```

Homebrew tap(`brew install k2sebeom/balpyo/balpyo`)의 포뮬러도 릴리스마다 생성된다 — 아카이브를
받는 방법만 다르고 안의 바이너리는 같다.

!!! note "아직 첫 태그가 없다"

    이 저장소에는 아직 `v*` 태그가 없으므로 위 세 경로(설치 스크립트·아카이브·tap)는 **첫 릴리스부터**
    열린다. 그때까지는 위의 `go build`가 유일한 길이다. 윈도우 아카이브는 없다 — 그 플랫폼에서
    검증한 적이 없기 때문이다.

PPTX 타깃과 `--check-layout`은 시스템 브라우저(Chromium/Chrome)를 하나 띄운다 —
배치를 그 브라우저로 재서 PPTX 좌표로 옮기기 때문이다. 웹 타깃만 쓰면
브라우저는 필요 없다.

## 2. 프로젝트 하나

```bash
./balpyo init my-talk --author "이세범" --lang ko
```

```text
my-talk/
├── balpyo.json              # 덱 설정: 제목·저자·언어·종횡비·출력 디렉토리
├── .gitignore               # dist/ 와 preview/
└── slides/
    ├── 001_title.slide      # 파일 이름의 번호가 슬라이드 순서다
    ├── 002_content.slide
    └── 003_end.slide
```

세 장은 그대로 빌드된다 — `--check-layout --strict`까지 경고 0으로 통과하므로, 여기서
나오는 경고는 전부 방금 고친 것에서 온다. 각 장이 하나씩 가르친다: 표지는 자리를 **테마의
슬롯**이 정한다는 것(그래서 자식에 `align`이 없다), 두 번째 장은 `<row>`·`<col>`·`<shape>`로
칸을 나누는 방법, 세 번째 장은 `layout`이 정렬을 바꾼다는 것이다.

`balpyo.json`은 선택이다 (없으면 기본값을 쓰고 제목은 디렉토리 이름에서 유도한다). 슬라이드는
`slides/` 아래의 `.slide` 파일이고, **파일 이름으로 정렬한다** — 사이에 한 장을 끼우려면
`002_5_…`처럼 이름을 붙이는 대신 번호를 다시 매기는 쪽이 낫다. 설정 키는
[개요](../index.md)에, 슬라이드 어휘는 [레퍼런스](../reference/index.md)에 있다.

`--lang`은 덱 언어(BCP 47 언어 태그 — `ko`·`en`·`ja`…)이고 생략하면 `en`이다. 한국어로 쓰는
덱이면 `ko`로 둔다: 웹이 한국어를 **어절에서** 접는 규칙이 이 값에 걸려 있어서, `en`인 채로 두면
"인스턴스" 같은 낱말이 음절 사이에서 끊긴다. 나중에 바꾸려면 `balpyo.json`의 `lang`을 고친다.

`init`은 이미 프로젝트인 자리를 **덮어쓰지 않는다** (`--force`가 없다). 완성된 덱을 읽고
싶으면 저장소의 `examples/decks/`를 본다.

## 3. 감시하며 쓴다

```bash
./balpyo watch --project my-talk
```

```
watching my-talk → http://127.0.0.1:4600  (Ctrl+C to stop)
✓ 14:31:02  3 slides · 1 file (41ms)
```

주소를 브라우저에 붙이고 `slides/001_title.slide`의 제목을 고쳐 저장하면 그 탭이 스스로
새로고침한다. 이 루프에서 알아 둘 것은 셋이다.

- **틀리면 화면이 남는다.** 빌드가 실패하면 마지막 성공본이 그대로 있고 왼쪽 아래에 진단이
  뜬다 — 방금 고친 자리를 보면서 이유를 읽는다. 같은 진단이 터미널에도 한 줄씩 나온다.
- **경고는 배지로 뜬다.** 빌드는 성공했지만 할 말이 있는 경우다 (테마에 없는 `variant` 등).
  배지를 누르면 목록이 열린다.
- **산출물은 하나다.** `watch`는 `balpyo build`와 같은 파이프라인을 돌려 같은 `dist/`에 쓴다.
  리로드 스크립트는 응답에만 주입되므로 디스크의 `index.html`은 배포본 그대로다.

글이 박스를 넘치는지까지 보려면 배치 검사를 켠다 (브라우저를 하나 띄우고, 그 브라우저는 루프
내내 재사용된다).

```bash
./balpyo watch --project my-talk --check-layout
```

경고를 읽고 고치는 순서는 [배치 경고 읽기](layout-warnings.md)에 있다.

## 4. 슬라이드 한 장 추가

`slides/004_next.slide`를 만든다. 감시 중이면 저장하는 순간 덱에 들어온다.

```html
<slide id="next" layout="content">
  <text role="title">다음 분기</text>
  <row gap="lg">
    <col grow="1">
      <text role="heading">할 것</text>
      <ul>
        <li>가격 실험 2건</li>
        <li>온보딩 재설계</li>
      </ul>
    </col>
    <shape type="rect" fill="surface" w="360" h="200">
      <p align="center">여기에 도형 하나</p>
    </shape>
  </row>
  <note>이 줄은 발표자만 본다.</note>
</slide>
```

`layout` 11종과 각 레이아웃이 받는 것은 [레이아웃 카탈로그](../reference/layouts.md)에,
쓸 수 있는 태그는 [SOML 레퍼런스](../reference/index.md)에 있다. 어휘는 **닫혀 있다** —
없는 태그·속성은 오타로 보고 파일·행·열을 짚어 알려준다.

## 5. 내보낸다

```bash
./balpyo build --project my-talk --target html,pptx
```

```
4 slides → /path/to/my-talk/dist
  index.html
  my-talk.pptx
```

`index.html`은 자체 완결형이다 — 파일 하나(와 있으면 `assets/`)를 열면 오프라인에서 발표된다.
`.pptx`의 텍스트 박스는 진짜 텍스트 박스이고 도형은 진짜 도형이며 색은 테마 색 참조를
유지한다. 줄바꿈 위치처럼 엔진마다 달라도 되는 것은 [계약 밖의 차이](known-differences.md)에 있다.

CI에서는 경고를 통과시키지 않는다. 산출물이 필요 없는 검사라면 `check`가 같은 판정을
산출물 없이 한다 ([`balpyo check`](../reference/cli.md#balpyo-check)).

```bash
./balpyo build --project my-talk --target html,pptx --strict
./balpyo check --project my-talk --check-layout --strict   # 산출물 없이 판정만
```

두 산출물이 같은지 **눈으로** 보려면 검수 갤러리를 만든다 — 슬라이드마다 웹 스크린샷과 PPTX
렌더가 나란히 놓인다. PPTX를 그림으로 되돌리는 데 OnlyOffice Document Builder가, 없으면 LibreOffice와
`pdftoppm`이 필요하며, 둘 다 없으면 PPTX 열을 빼고 웹 열만 만든다
([`balpyo preview`](../reference/cli.md#balpyo-preview)).

```bash
./balpyo preview --project my-talk   # → my-talk/preview/index.html
```

## 6. 발표

`dist/index.html`을 브라우저로 연다.

| 조작 | 방법 |
|---|---|
| 슬라이드 이동 | 하단 바의 ‹ ›, 또는 ←/→·↑/↓·Space·PageUp/PageDown·Home/End |
| 순차 등장 | `<li reveal>`을 쓴 항목은 다음 키에 한 스텝씩 나타난다 ([자세히](../reference/soml/text-subset.md#순차-등장-reveal)) |
| 발표자 노트 | 하단 바의 **Notes** 또는 `n` |
| 전체 보기 | 하단 바의 **Grid** 또는 `o` — 타일을 클릭하면 그 슬라이드로, `Esc`는 제자리로 |
| 딥링크 | `index.html#/<슬라이드 id>` 또는 `#/<번호>` |
| PDF | 브라우저에서 인쇄(Cmd/Ctrl+P) — 한 장이 한 페이지로 나온다 ([자세히](../reference/cli.md#pdf로-인쇄)) |

## 다음

- [CLI](../reference/cli.md) — `init`·`build`·`check`·`watch`·`preview`의 옵션과 종료 코드
- [테마 고르기·만들기](theming.md) — 내장 테마 셋을 `--theme`로 견주고 고르는 방법
- [테마 패키지](../reference/theme.md) — 토큰·컴포넌트·레이아웃과 덱 크롬(페이지 번호·푸터)
- [외관 속성](../reference/appearance.md) — 색·크기·간격의 캐스케이드와 탈출구
