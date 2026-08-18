# 항목 05 — OOXML 조립기 최소판: 텍스트 박스를 spTree에 넣는다

지금 `render/pptx`가 내는 슬라이드는 **spTree가 비어 있다** (`parts.go`의 `slideTemplate`
주석이 "도형은 Iteration 1부터 여기 들어간다"고 적어 둔 자리다). 이 항목이 그 자리를 채운다 —
잰 좌표(`render/geom`)와 해석된 외관(`soml.TextStyle`)을 OOXML로 **순방향 컴파일**한다.
computed-style 역추론은 없다 (PRD §9.3).

## 하는 일

1. `<text>` 하나 → `p:sp` 하나. 문서 순서가 spTree 순서다 (겹침 z-순서의 근거).
   - `p:nvSpPr`: 결정적인 도형 id와 이름. `p:cNvSpPr txBox="1"`.
   - `p:spPr`: `a:xfrm`(`a:off`/`a:ext`) + `a:prstGeom prst="rect"` + 채움/선 없음.
     좌표는 `pptx.Offset`/`pptx.Extent`로만 환산한다 — 단위 경계는 그 함수들뿐이다
     (`units.go`의 계약, PRD §13-5).
   - `p:txBody`: `a:bodyPr`에 **autofit 없음**(박스 고정, 오버플로 허용 — PRD §13-1)과
     인셋. 인셋은 웹의 박스 모델과 같은 값이어야 한다 — 웹이 잰 rect가 어느 박스(콘텐츠 박스냐
     테두리 박스냐)인지 `render/geom`에서 확인하고, 그 답에 맞춰 `lIns/tIns/rIns/bIns`를
     정한다. 여기서 틀리면 좌표는 맞는데 글자가 몇 px 밀리고, 그 오차는 픽셀 비교(항목 07)에서
     원인 불명으로 나타난다.
   - `a:pPr`: `algn`(`AlignStart/Center/End` → 논리 이름 그대로 `l/ctr/r`가 아니라
     OOXML의 논리 대응으로), `lnSpc`는 `spcPct`(TextStyle.LineHeight × 100000).
   - `a:r`/`a:rPr`: `sz`는 `pptx.FontSize`, 폰트는 토큰의 `Scheme`으로
     `+mj-lt`/`+mn-lt`(major/minor) 참조, 굵기는 `TextStyle.Weight` 임계값으로 `b="1"`.
   - 색: 토큰이면 **`a:schemeClr val="<Ref>"`**, 리터럴이면 `a:srgbClr`.
     `soml.ColorToken`의 `Slot`/`Ref`가 이미 이 매핑을 갖고 있다 — 새로 표를 만들지 않는다.

2. **테마 파트를 테마 표에서 컴파일한다.** 지금 `themeTemplate`의 `clrScheme`은
   하드코딩된 Office 기본값(`dk2=44546A`, `accent1=4472C4`)이고 `core.minimalColors`는
   다른 값(`muted=6B7280`, `accent1=0F65F9`)이다. 이대로면 `color="accent1"` 텍스트가
   **웹과 PPTX에서 다른 색으로 나온다** — 토큰 보존 계약이 겉으로만 지켜진다.
   `soml.Theme.Colors`의 `Slot`으로 clrScheme 자리를 채워 두 타깃이 같은 색을 쓰게 한다.
   테마 표에 없는 자리(`lt2`, `accent2~6`, `hlink`)는 어떻게 채웠는지 근거를 주석에 남긴다.

3. 슬라이드 배경: 웹은 `bg` 토큰으로 `.balpyo-slide` 배경을 낸다 (`render/web/style.go`).
   PPTX에서 같은 배경이 나오는지 확인하고, 아니면 `p:bg`로 맞춘다 (배경이 갈라지면 픽셀
   비교가 전면 불일치로 뜬다).

4. 빈 요소·크기 0 박스의 정책을 정한다: 도형을 내지 않을지, 낼지. `Extent`가 이 판단을
   호출자에게 남겨 두었으니(주석) 그 판단을 여기서 내리고 근거를 적는다.

## 테스트

- 단위 테스트로 XML을 단정한다 (구조 — `p:sp` 개수, `a:off/a:ext` 값, `schemeClr val`,
  리터럴 `srgbClr`가 **없는 것**). 골든 픽스처는 항목 07의 일이다.
- 좌표: 지오메트리가 잰 rect가 그대로 EMU로 나오는지 (`units_test.go`의 경계 계약 유지).
- 이스케이프: `&`·`<`가 든 텍스트가 유효한 XML로 나오는지. 픽스처 003의 "Q and A"처럼
  실제 원문이 무엇을 요구하는지 본다.
- 개행: 픽스처 002의 본문에는 **원문 개행이 있다.** 웹 렌더러가 그것을 어떻게 다루는지
  (`render/web/content.go`) 확인하고 PPTX가 **같은 결정**을 하게 한다 — 한쪽이 `a:br`를
  내고 다른 쪽이 공백으로 접으면 줄 수가 갈라진다. 두 렌더러의 일치를 못 박는 테스트를 남긴다.
- 결정성: 같은 덱을 두 번 조립하면 바이트가 같은지 (맵 순회가 XML에 새지 않는지).
- `make test`의 LibreOffice 열림 테스트가 통과해야 한다 — 여기서 깨지면 XML이 스키마에서
  벗어난 것이다. 경고 문자열도 본다 (복구 경고는 열림 성공으로 세지 않는다).

## 범위 밖

- `<note>` → 노트 슬라이드 (항목 06).
- `01_text` 픽스처 골든·픽셀 비교·임계값 측정 (항목 07). 이 항목은 `_01_text`를
  **그대로 둔다** — 픽스처를 수집 대상으로 바꾸는 것은 골든과 같은 커밋에서 일어난다.
- 문서 (항목 08). 다만 `render/pptx`의 패키지·함수 주석은 이 항목이 갱신한다
  ("Iteration 0은 뼈대만" 같은 이제 틀린 문장을 남기지 않는다).
- 레이아웃 카탈로그, 플레이스홀더 상속, 표준 레이아웃 11종 (Iteration 3 이후).
