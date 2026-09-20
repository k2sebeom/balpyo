# 가이드

"어떻게 하는가". 사용자 시나리오 하나가 문서 하나다.

| 가이드 | 다루는 것 |
|---|---|
| [배치 경고 읽기](layout-warnings.md) | 넘침·스테이지 이탈·겹침 경고를 켜고 읽고 고치는 순서 |

예정된 가이드와 그것을 가능하게 하는 iteration:

| 가이드 | 다루는 것 | 생기는 시점 |
|---|---|---|
| `getting-started.md` | `init` → `server` → `build` 흐름, 발표 조작 | `balpyo init`이 생기는 Phase 2 |
| `theming.md` | 테마 만들기 (`minimal`을 예제로) | 테마 패키지 포맷 — Iteration 10 |
| `components.md` | 합성 컴포넌트 만들기 (`dim-card` 예제) | `<define>`/`<slot>` — Iteration 11 |

가이드는 스캐폴딩 명령과 테마가 있어야 정직하게 쓸 수 있다. 그때까지 실행 가능한 출발점은
저장소의 픽스처다 — 하나가 곧 완전한 프로젝트이므로 그대로 빌드해 볼 수 있다.

```bash
go run ./cmd/balpyo build --project testdata/fixtures/00_empty --target html,pptx
```

픽스처의 구조와 규칙은 [테스트 환경](../internals/testing.md)에 있다.
