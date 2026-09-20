# 가이드

"어떻게 하는가". 사용자 시나리오 하나가 문서 하나다.

| 가이드 | 다루는 것 |
|---|---|
| [시작하기](getting-started.md) | 예제 복사 → `watch`로 고치며 보기 → `build`로 내보내기 → 발표 |
| [배치 경고 읽기](layout-warnings.md) | 넘침·스테이지 이탈·겹침 경고를 켜고 읽고 고치는 순서 |
| [알려진 차이](known-differences.md) | 뷰어가 웹과 다르게 그리는 것 — 문장부호 앞뒤의 틈이 버그가 아닌 이유 |

예정된 가이드와 그것을 가능하게 하는 iteration:

| 가이드 | 다루는 것 | 생기는 시점 |
|---|---|---|
| `theming.md` | 테마 만들기 (`minimal`을 예제로) | 테마 패키지 포맷 — Iteration 10 |
| `components.md` | 합성 컴포넌트 만들기 (`dim-card` 예제) | `<define>`/`<slot>` — Iteration 11 |

[시작하기](getting-started.md)의 출발점이 `balpyo init`이 아니라 **예제 복사**인 이유는
스캐폴딩 명령이 아직 없다는 것이다 (Phase 2). 같은 이유로 실행 가능한 출발점이 하나 더
있다 — 저장소의 픽스처다. 하나가 곧 완전한 프로젝트이므로 그대로 빌드해 볼 수 있다.

```bash
go run ./cmd/balpyo build --project testdata/fixtures/00_empty --target html,pptx
```

픽스처의 구조와 규칙은 [테스트 환경](../internals/testing.md)에 있다.
