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
| [0023](0023-assets-live-in-the-deck.md) | 에셋 표는 덱 모델이 소유한다 (지오메트리와 달리 곁 표가 아니다) | 5 | 확정 (0004의 경계를 다시 그린다) |
| [0024](0024-src-is-project-relative.md) | `src`는 프로젝트 루트 기준이고 프로젝트 밖·네트워크는 막는다 | 5 | 확정 |
| [0025](0025-image-formats-are-png-and-jpeg.md) | 이미지 포맷은 PNG·JPEG 2종이고 내용으로 판정한다 | 5 | 확정 |
| [0026](0026-contain-fits-the-box-to-the-picture.md) | `fit="contain"`은 박스를 그림에 맞춘다 (`object-fit`을 쓰지 않는다) | 5 | 확정 |
| [0027](0027-open-alt-close-frame.md) | `alt`을 열고 `frame`을 닫는다 (PRD §7.5 표 갱신) | 5 | 확정 |
| [0028](0028-stack-is-a-flow-not-a-kind.md) | `<stack>`은 새 종류가 아니라 새 흐름이다 (`KindStack`을 만들지 않는다) | 6 | 확정 |
| [0029](0029-paint-order-follows-pptx.md) | 겹침 칠 순서를 지키려고 stack 자식을 전부 위치 지정 요소로 만든다 | 6 | 확정 |
| [0030](0030-canvas-origin-is-stage-padding-box.md) | 캔버스 좌표는 스테이지 패딩 박스 기준이다 (PRD §13-3 갱신) | 6 | 확정 |
| [0031](0031-position-is-one-channel.md) | 한 축의 위치는 한 채널이 정한다 (`at`⊕`x`/`y`, z-index 없음, 좌표는 음수) | 6 | 확정 |
| [0032](0032-slide-layout-vocab-is-core-checked.md) | 슬라이드 수준 어휘는 값과 Span으로 core가 검사한다 (`layout`) | 6 | 확정 |
| [0033](0033-color-variants-preserve-scheme-ref.md) | 색 변형은 참조를 보존한다 (lumMod/lumOff/alpha vs color-mix) | 8 | 확정 |
| [0034](0034-gradient-svg-def-and-objectboundingbox.md) | 그라디언트는 SVG def로 내고 objectBoundingBox로 늘린다 (대각·비정사각형 한계) | 8 | 확정 |
| [0035](0035-slots-are-real-placeholders-with-explicit-override.md) | 슬롯은 진짜 PPTX 플레이스홀더가 되고 명시 xfrm/rPr로 패리티를 지킨다 | M1.2 | 확정 |
| [0036](0036-table-is-html-ish-tr-td-with-explicit-a-tbl.md) | `<table>`은 HTML식 `tr/td`, 셀은 텍스트 전용, 명시 `tcPr`로 `a:tbl`이 된다 | M2.1 | 확정 |
| [0037](0037-char-refs-require-semicolon-and-cdata.md) | 문자 참조는 `;`를 요구하는 닫힌 어휘이고, 마크업이 아닌 `<`·`&`는 리터럴이다 | P0.1 | 확정 |
| [0038](0038-list-appearance-lives-on-the-container.md) | 목록의 외관은 컨테이너에 쓰고 항목이 상속한다 (`variant`는 닫는다) | P0.3 | 확정 |
| [0039](0039-layout-warnings-are-box-containment.md) | 배치 경고는 박스 포함 관계로 판정한다 (`scrollHeight`를 쓰지 않는다) | P1.1 | 확정 |
| [0040](0040-preset-text-rect-is-a-geometry-column.md) | 프리셋의 텍스트 영역은 기하 표의 한 열이고, 그 표는 웹만 읽는다 | P1.2 | 확정 |
| [0041](0041-chevron-text-rect-is-cancelled-in-pptx.md) | chevron의 텍스트 영역은 폭 0이고, PPTX가 음수 인셋으로 되돌린다 | P1.2 | 확정 |
| [0042](0042-chevron-labels-stay-outside-the-shape.md) | chevron의 라벨은 도형 밖에 둔다 (0040 결과의 소개 덱 한 줄을 철회한다) | P1.2 | 확정 |
| [0043](0043-explicit-width-belongs-to-the-placement-context.md) | 확정 폭은 배치 문맥의 것이다 (슬롯 영역은 테마가, `<text>`의 가로 크기는 저자가) | P1.3 | 확정 |
| [0044](0044-slots-need-a-theme-and-vertical-rhythm-needs-an-owner.md) | 슬롯은 테마가 받아야 뜻이 있고, 흐름 모드의 세로 리듬은 저자의 `justify`다 | P1.4 | 확정 |
| [0045](0045-deck-chrome-is-config-text-in-a-theme-declared-band.md) | 덱 크롬은 설정이 내용을, 테마가 자리를, `a:fld`가 번호를 소유한다 | P2.2 | 확정 |

PRD §13의 열린 질문 12건은 2026-08-16에 전부 확정되어 PRD 본문에 반영되었다. 그 결정들의
색인은 PRD에 있고, 이 표는 **그 이후 구현 중에 새로 확정한 것**만 담는다.
