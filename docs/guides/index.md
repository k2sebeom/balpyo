# 가이드

"어떻게 하는가". 사용자 시나리오 하나가 문서 하나다.

| 가이드 | 다루는 것 |
|---|---|
| [시작하기](getting-started.md) | `init` → `watch`로 고치며 보기 → `build`로 내보내기 → 발표 |
| [배치 경고 읽기](layout-warnings.md) | 넘침·스테이지 이탈·겹침 경고를 켜고 읽고 고치는 순서 |
| [테마 고르기·만들기](theming.md) | `--theme`로 견주기 · 내장 테마 셋의 성격 · 새 테마가 지켜야 하는 계약 |
| [계약 밖의 차이](known-differences.md) | 엔진마다 다르게 그려도 되는 것 — 줄바꿈 위치 · 문장부호 앞뒤의 틈이 버그가 아닌 이유 |

[시작하기](getting-started.md)의 출발점은 `balpyo init`이다 — 세 장짜리 덱 하나가 경고 없이
빌드되는 상태로 나온다. 완성된 덱을 읽어 보고 싶다면 저장소의 `examples/decks/`가 있고, 혼자 서는
테마 하나가 `examples/themes/`에 있다.

```bash
balpyo build --project examples/decks/balpyo-intro --target html,pptx
```
