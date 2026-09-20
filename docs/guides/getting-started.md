# 시작하기

빈 디렉토리에서 발표할 수 있는 덱까지. 저작 루프는 [`balpyo watch`](../reference/cli.md#balpyo-watch)
하나이고, 그 루프가 이 문서의 절반이다.

!!! note "`balpyo init`은 아직 없다"
    스캐폴딩 명령이 Phase 2에 생기면 아래 2단계가 `balpyo init my-talk` 한 줄로 바뀐다.
    그때까지 출발점은 저장소의 **예제를 복사하는 것**이다 — 예제 하나가 곧 완전한
    프로젝트이므로 그대로 빌드되고, 고칠 자리도 이미 있다.

## 1. 바이너리

런타임 의존은 없다. Go 툴체인이 있으면 저장소에서 직접 만든다.

```bash
git clone https://github.com/k2sebeom/balpyo && cd balpyo
go build -o balpyo ./cmd/balpyo
./balpyo version
```

PPTX 타깃과 `--check-layout`은 시스템 브라우저(Chromium/Chrome)를 하나 띄운다 —
[지오메트리 패스](../internals/geometry-pass.md)가 배치를 그 브라우저로 잰다. 웹 타깃만 쓰면
브라우저는 필요 없다.

## 2. 프로젝트 하나

```bash
cp -r examples/quarterly-review my-talk
```

```text
my-talk/
├── balpyo.json              # 덱 설정: 제목·저자·종횡비·출력 디렉토리
└── slides/
    ├── 001_title.slide      # 파일 이름의 번호가 슬라이드 순서다
    ├── 002_agenda.slide
    └── …
```

`balpyo.json`은 선택이다 (없으면 기본값을 쓰고 제목은 디렉토리 이름에서 유도한다). 슬라이드는
`slides/` 아래의 `.slide` 파일이고, **파일 이름으로 정렬한다** — 사이에 한 장을 끼우려면
`002_5_…`처럼 이름을 붙이는 대신 번호를 다시 매기는 쪽이 낫다. 설정 키는
[개요](../index.md)에, 슬라이드 어휘는 [레퍼런스](../reference/index.md)에 있다.

## 3. 감시하며 쓴다

```bash
./balpyo watch --project my-talk
```

```
감시 중: my-talk → http://127.0.0.1:4600  (Ctrl+C로 끝낸다)
✓ 14:31:02  슬라이드 6개 · 파일 1개 (41ms)
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

`slides/007_next.slide`를 만든다. 감시 중이면 저장하는 순간 덱에 들어온다.

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
슬라이드 7장 → /path/to/my-talk/dist
  index.html
  q3-비즈니스-리뷰.pptx
```

`index.html`은 자체 완결형이다 — 파일 하나(와 있으면 `assets/`)를 열면 오프라인에서 발표된다.
`.pptx`의 텍스트 박스는 진짜 텍스트 박스이고 도형은 진짜 도형이며 색은 테마 색 참조를
유지한다. 뷰어가 웹과 다르게 그리는 몇 가지는 [알려진 차이](known-differences.md)에 있다.

CI에서는 경고를 통과시키지 않는다.

```bash
./balpyo build --project my-talk --target html,pptx --strict
```

## 6. 발표

`dist/index.html`을 브라우저로 연다.

| 조작 | 방법 |
|---|---|
| 슬라이드 이동 | 하단 바의 ‹ ›, 또는 ←/→·↑/↓·Space·PageUp/PageDown·Home/End |
| 발표자 노트 | 하단 바의 **Notes** 또는 `n` |
| 딥링크 | `index.html#/<슬라이드 id>` 또는 `#/<번호>` |

## 다음

- [CLI](../reference/cli.md) — `build`·`watch`의 옵션과 종료 코드
- [테마 패키지](../reference/theme.md) — 토큰·컴포넌트·레이아웃과 덱 크롬(페이지 번호·푸터)
- [외관 속성](../reference/appearance.md) — 색·크기·간격의 캐스케이드와 탈출구
