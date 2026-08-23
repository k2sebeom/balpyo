# 설계 결정 (ADR)

구현 중 확정한 설계 결정을 번호 순으로 남긴다. 형식은 **맥락 → 선택지 → 결정 → 근거 →
결과**이고, 한번 쓴 ADR은 고치지 않는다 — 결정을 뒤집을 때는 새 ADR을 쓰고 이 표에서
관계를 밝힌다. PRD §13의 기존 결정을 뒤집는 경우 PRD 본문도 함께 갱신한다.

| # | 결정 | Iteration | 상태 |
|---|---|---|---|
| [0001](0001-parser-trivia.md) | 파서는 트리비아를 보존한다 | 0 | 확정 |
| [0002](0002-logical-stage-size.md) | 4:3 논리 스테이지는 960×720이다 | 0 | 확정 |
| [0003](0003-inline-style-appearance.md) | 요소 외관은 인라인 `style` 속성으로 낸다 | 1 | 확정 |
| [0004](0004-geometry-side-table.md) | 지오메트리는 덱이 아니라 곁 표로 주입한다 | 1 | 확정 |
| [0005](0005-line-spacing-and-slack.md) | 줄간격은 spcPts, 박스 폭 안전 여유 4% | 1 | 확정 |
| [0006](0006-presenter-view-attribute.md) | 발표자 뷰는 <html> 속성으로 켜고 측정은 문서 뷰에서 한다 | 1.5 | 확정 |
| [0007](0007-block-tag-role-ownership.md) | 블록 태그의 역할은 언어가, 태그별 기본값은 테마가 소유한다 (h1 → title) | 2 | 확정 |
| [0008](0008-run-style-inherit-copy.md) | 인라인 런 스타일은 확정 스타일의 상속-복사다 | 2 | 확정 |
| [0009](0009-measured-kinds.md) | 측정 대상은 블록 종류뿐이다 (요소 종류 카탈로그 공유) | 2 | 확정 |
| [0010](0010-link-color-accent1.md) | 링크 색은 hlink 전용 토큰이 아니라 accent1을 재사용한다 | 2 | 확정 (I10 재검토) |
| [0011](0011-layout-containers-are-not-shapes.md) | 레이아웃 컨테이너는 PPTX 도형이 아니다 (자식 좌표만 남는다) | 3 | 확정 |
| [0012](0012-space-tokens-have-no-theme-slot.md) | 공간 토큰에는 PPTX 테마 슬롯이 없다 | 3 | 확정 |
| [0013](0013-measured-kinds-include-layout.md) | 측정 대상에 레이아웃 컨테이너와 `<spacer>`를 넣는다 | 3 | 확정 (0009의 후속) |
| [0014](0014-style-escape-hatch-boundary.md) | `style` 탈출구는 문법만 에러, allowlist 밖은 경고, 값은 보지 않는다 | 3 | 확정 |
| [0015](0015-fill-is-hundred-percent.md) | `fill`은 해당 축의 100%다 (`grow`와 겹치지 않는다) | 3 | 확정 |
| [0016](0016-shape-text-is-shape-body.md) | 도형 안 텍스트는 같은 도형의 본문이다 | 4 | 확정 |
| [0017](0017-shape-geometry-from-ooxml-presets.md) | 도형 기하의 출처는 OOXML 프리셋 정의다 (정규 공간을 늘리고 조정값이 맞춘다) | 4 | 확정 |
| [0018](0018-opacity-compiles-to-color-alpha.md) | `opacity`는 합성 불투명도가 아니라 색 알파로 컴파일된다 | 4 | 확정 |
| [0019](0019-radius-is-rounded-rect-only.md) | `radius`는 `rounded-rect` 전용이고 테마 기본값만 조용히 버려진다 | 4 | 확정 (I10 재검토) |
| [0020](0020-shadow-tokens-only.md) | `shadow`는 토큰만 받는다 (수치 축약을 열지 않는다) | 4 | 확정 |
| [0021](0021-measure-view-disables-transforms.md) | 측정 뷰에서 변형을 끈다 (회전은 잰 뒤에 얹힌다) | 4 | 확정 (0006의 후속) |
| [0022](0022-shape-variants-do-not-inherit.md) | 도형 variant는 자식 텍스트의 외관을 담지 않는다 | 4 | 확정 |

PRD §13의 열린 질문 12건은 2026-08-16에 전부 확정되어 PRD 본문에 반영되었다. 그 결정들의
색인은 PRD에 있고, 이 표는 **그 이후 구현 중에 새로 확정한 것**만 담는다.
