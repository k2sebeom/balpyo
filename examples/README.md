# 예제

balpyo로 쓴 덱과 테마의 견본이다. 모두 두 타깃(웹 · PPTX)으로 **경고 0건**에 빌드된다.

## `decks/` — 덱

| 덱 | 장수 | 무엇을 보여 주는가 |
|---|---|---|
| [`balpyo-intro`](decks/balpyo-intro) | 12 | 이 도구 자체의 소개 — 파이프라인 도형, `<table>`, 인용 레이아웃, 장마다 `<note>` |
| [`incident-postmortem`](decks/incident-postmortem) | 10 | 장애 회고 — 타임라인, 영향 표, `lang="go"` 코드 블록, `layout="canvas"` |
| [`quarterly-review`](decks/quarterly-review) | 6 | 분기 리뷰 — KPI 카드 그리드, `<table>`, `<stack>` 콜아웃 |
| [`ec2-intro`](decks/ec2-intro) | 13 | 손으로 짠 키노트 수준의 기술 소개 — 연결선 도식(상태 기계 · 아키텍처), `cube` 층, 경로 차트, 형광펜 · 작은 단위, 등장 · 전환, 어두운 장(`invert`), 덱 안의 테마 `summit` |

```bash
balpyo build --project examples/decks/quarterly-review --target html,pptx
balpyo watch --project examples/decks/quarterly-review       # http://127.0.0.1:4600
```

## `themes/` — 테마

| 테마 | 성격 |
|---|---|
| [`harbor`](themes/harbor) | 차가운 회색 바탕에 흰 카드, 굵은 제목, 위 왼쪽에서 시작하는 표지. 아무것도 물려받지 않는다 |

테마는 덱 프로젝트의 `themes/<이름>/`에 두면 쓸 수 있다:

```bash
cp -R examples/themes/harbor my-deck/themes/harbor
# my-deck/balpyo.json에 "theme": "harbor"를 적거나, 한 번만 써 보려면:
balpyo build --project my-deck --theme harbor
```

## `showcase/` — 렌더 결과

저장소 [README](../README.md)의 쇼케이스다: 위의 덱을 렌더한 산출물(`decks/<덱>/index.html` ·
`.pptx`)과 슬라이드 스크린샷(`images/`). `index.html`은 브라우저로, `.pptx`는 PowerPoint나
Keynote로 열어 본다.

포맷은 [테마 패키지 레퍼런스](../docs/reference/theme.md)에, 만드는 법은
[테마 고르기·만들기](../docs/guides/theming.md)에 있다.
