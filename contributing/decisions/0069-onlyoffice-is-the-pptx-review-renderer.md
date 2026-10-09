# 0069 — PPTX 검수 렌더는 OnlyOffice가 먼저다 (LibreOffice는 물러설 자리)

- 상태: 확정
- 날짜: 2026-10-09
- 관련: [ADR 0064](0064-the-contract-is-component-and-layout-parity-not-pixel-parity.md)(컴포넌트·레이아웃 동일성),
  [ADR 0065](0065-pptx-validity-is-judged-by-the-ecma-376-schema-not-by-libreoffice.md)(유효성은 스키마),
  [ADR 0066](0066-preview-only-places-renders-side-by-side.md)(preview는 나란히 놓기만 한다),
  [testing.md](../internals/testing.md)

## 맥락

자동 판정은 셋이다: 스키마(ECMA-376), 파트 사이 규칙, 동일성(웹 트리 = PPTX 트리). 셋 다 **우리가
쓴 XML을 우리가 읽는다**. 동일성의 PPTX 쪽 리더(`parity.go`)는 렌더러를 쓴 사람이 쓴 것이라, OOXML을
같은 방식으로 오해하면 두 쪽이 함께 틀리고 판정은 초록이다. 또 그 리더는 명시한 속성만 읽는다 —
엔진이 실제로 그리는 값(커닝 여부, 프리셋의 텍스트 영역 공식, 색 변형의 색 공간)은 보지 않는다.

`balpyo preview`의 PPTX 열은 LibreOffice였다. LibreOffice는 PPTX를 ODF로 옮긴 뒤 그리므로 자기
규칙(한글↔라틴 간격, 프리셋 공식의 경계값 해석)을 더하고, 깨진 파일도 조용히 받아 준다 —
ADR 0065가 유효성 판정에서 그것을 뺀 이유다. 검수 그림으로도 같은 문제가 있다: 그림의 차이가
우리 XML 때문인지 번역 때문인지 알 수 없다.

OnlyOffice(Document Builder의 `x2t`)로 예제 덱 셋과 픽스처 전부(111장)를 되돌려 보자 판정 셋이 모두
초록인 산출물에서 결함 넷이 나왔다 — 한 줄을 꽉 채운 텍스트 박스가 커닝 없는 엔진에서 접혀 겹치는
것(표지 셋), `tint`가 두 타깃에서 다른 색 공간을 쓰는 것, chevron의 텍스트 영역을 명세와 다르게 읽은
것. 모두 "같은 XML을 명세대로 읽는 다른 엔진"이 있어야 보이는 종류다.

## 선택지

1. **LibreOffice를 유지한다.** 번역 단계가 결함을 덮고 자기 버릇을 더한다.
2. **OnlyOffice로 바꾸고 LibreOffice를 지운다.** OnlyOffice가 없는 호스트(맥의 Homebrew 사용자 다수)에서
   PPTX 열이 사라진다.
3. **OnlyOffice가 먼저, 없으면 LibreOffice.** 엔진 이름을 갤러리와 로그에 밝힌다.
4. **OnlyOffice 렌더를 게이트로 만든다** (넘친 글자를 그림에서 찾는 판정). 잉크 판정은 폰트 ·
   힌팅 · 엔진 버전에 묶이고 ADR 0064가 버린 길이다.

## 결정

**선택지 3.**

1. `internal/preview`의 래스터라이저는 인터페이스이고 엔진이 둘이다: `OnlyOffice`(`x2t` → PNG 묶음)와
   `LibreOffice`(`soffice` → PDF → `pdftoppm`). `NewRasterizer`는 OnlyOffice를 먼저 찾는다
   (`BALPYO_ONLYOFFICE` → `/opt/onlyoffice/documentbuilder`).
2. OnlyOffice는 변환 전마다 `docbuilder`를 빈 스크립트로 돌려 글꼴 색인을 대조한다. `x2t`는 그 색인으로만
   글꼴을 찾고, 색인이 없거나 낡으면 **한글만 빠진 그림**을 성공으로 낸다.
3. 갤러리 안내와 로그는 PPTX 열을 그린 엔진의 이름을 밝힌다.
4. 게이트가 아니다 (ADR 0066 그대로). 대신 저장소 전체를 한 번에 보는 입구를 둔다 —
   `make qa`가 예제 덱과 픽스처마다 갤러리를 만든다. 렌더러를 고친 커밋은 그 갤러리를 보고 낸다.
5. 엔진 한계로 확인된 차이(그라디언트 선을 첫 스톱 단색으로, 부드러운 그림자를 아래 띠로 그리는 것)는
   testing.md에 적는다 — 그림을 보는 사람이 그것을 우리 결함으로 읽지 않게.

## 근거

- **독립 리더가 필요한 자리는 판정이 아니라 발견이다.** 우리 리더가 놓친 해석 차이는 규칙으로 쓸 수
  있게 되기 전까지는 그림으로만 보인다. 발견한 것은 렌더러의 규칙(한 줄 박스의 `wrap`, 색 변형의 색 공간)과
  단위 테스트로 굳힌다 — 그림은 다음 결함을 찾는 데 쓴다.
- **OnlyOffice가 PowerPoint에 더 가깝다.** OOXML이 1차 형식이고 프리셋 공식을 명세대로 계산하며,
  커닝을 켜지 않은 런을 커닝 없이 놓는다 — PowerPoint와 같은 기본값이다. LibreOffice의 렌더와 웹이
  같아 보이던 자리가 OnlyOffice에서 갈라진 것은 그 차이다.
- **물러설 자리를 남긴다.** 엔진이 하나도 없으면 PPTX 열이 빠진다는 ADR 0066의 동작은 그대로다.

## 결과

- 갤러리의 PPTX 열이 OnlyOffice 그림이 된다 (설치돼 있으면). LibreOffice 그림과 줄바꿈이 다를 수 있다.
- OnlyOffice가 없는 개발자는 이전과 같은 그림을 본다 — 결함 발견력은 OnlyOffice 쪽이 높다.
- `make qa`는 `/preview/qa/` 아래에 쓴다 (커밋하지 않는다).
