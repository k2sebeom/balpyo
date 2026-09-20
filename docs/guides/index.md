# 가이드

"어떻게 하는가". 사용자 시나리오 하나가 문서 하나다.

| 가이드 | 다루는 것 |
|---|---|
| [시작하기](getting-started.md) | `init` → `watch`로 고치며 보기 → `build`로 내보내기 → 발표 |
| [배치 경고 읽기](layout-warnings.md) | 넘침·스테이지 이탈·겹침 경고를 켜고 읽고 고치는 순서 |
| [테마 고르기·만들기](theming.md) | `--theme`로 견주기 · 내장 테마 셋의 성격 · 새 테마가 지켜야 하는 계약 |
| [알려진 차이](known-differences.md) | 뷰어가 웹과 다르게 그리는 것 — 문장부호 앞뒤의 틈이 버그가 아닌 이유 |

예정된 가이드와 그것을 가능하게 하는 iteration:

| 가이드 | 다루는 것 | 생기는 시점 |
|---|---|---|
| `components.md` | 합성 컴포넌트 만들기 (`dim-card` 예제) | `<define>`/`<slot>` — Iteration 11 |

[시작하기](getting-started.md)의 출발점은 `balpyo init`이다 — 세 장짜리 덱 하나가 경고 없이
빌드되는 상태로 나온다. 저장소 안에서 읽을 수 있는 출발점도 둘 있다: `examples/`(완성된 덱,
다만 어휘의 마찰점을 일부러 밟는다)와 픽스처다. 픽스처 하나가 곧 완전한 프로젝트이므로
그대로 빌드해 볼 수 있다.

```bash
go run ./cmd/balpyo build --project testdata/fixtures/00_empty --target html,pptx
```

픽스처의 구조와 규칙은 [테스트 환경](../internals/testing.md)에 있다.
