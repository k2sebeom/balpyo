# 항목 06 — `<note>` → PPTX 노트 슬라이드

`soml.Slide.Notes`는 이미 채워져 있고 웹은 `<div class="balpyo-notes" hidden>`으로 낸다
(`render/web/web.go`). PPTX 쪽 대응(PRD §9.3: `<note>`는 **노트 슬라이드**로 매핑)이
아직 없다 — 발표자 노트는 PowerPoint/Keynote의 발표자 뷰에서 읽히는 자리이므로, 텍스트
프레임에 얹는 것으로는 대체되지 않는다.

## 하는 일

1. 파트를 추가한다:
   - `ppt/notesMasters/notesMaster1.xml` (+ rels → 테마). 노트 마스터는 **하나**다.
   - 노트가 있는 슬라이드마다 `ppt/notesSlides/notesSlideN.xml` (+ rels → 해당 슬라이드,
     노트 마스터).
   - `[Content_Types].xml`의 Override, `presentation.xml`의 `p:notesMasterIdLst`,
     `presentation.xml.rels`의 노트 마스터 관계. `notesSz`는 이미 있다 (`NotesCX/NotesCY`).
   - 관계 ID 배정이 `presRelsTemplate`의 규칙(`rId1`=마스터, `rId2..`=슬라이드,
     마지막=테마)과 어긋나지 않게 한다. 노트 마스터를 어디에 끼웠는지 주석으로 남긴다 —
     rId가 밀리면 슬라이드 관계가 엉뚱한 파트를 가리키고, 그 파일은 열리기는 한다.

2. **번호 규칙을 정하고 근거를 적는다**: `notesSlideN`의 N을 슬라이드 번호로 갈지
   (노트 없는 슬라이드에서 번호가 비어 뛴다), 노트가 있는 것만 1부터 셀지. 어느 쪽이든
   rels가 가리키는 슬라이드와 반드시 일치해야 한다 — 이것이 이 항목에서 가장 조용히
   틀릴 수 있는 부분이다 (틀려도 파일은 열리고, 노트만 다른 슬라이드에 붙는다).

3. 노트 슬라이드 본문: `p:cSld/p:spTree`에 슬라이드 이미지 플레이스홀더(`p:ph type="sldImg"`)와
   본문 플레이스홀더(`type="body" idx="1"`). 본문에 노트 텍스트를 문단으로 넣는다.
   **좌표는 마스터에서 상속받는다 — 지오메트리 패스를 쓰지 않는다.** 노트는 화면에 렌더되지
   않으므로 잰 박스가 없고, 없는 것을 요구하면 노트만 있는 덱이 브라우저를 띄우게 된다.
   `checkGeometry`/`geom.Refs`가 노트를 좌표 대상으로 세지 않는지 확인한다.

4. 노트 원문의 개행 정책: 노트는 저자가 읽는 글이므로 웹의 `balpyo-notes`와 PPTX가 같은
   줄 구조를 내야 한다. 두 렌더러의 결정을 한곳에 두거나, 같은 테스트로 묶는다.

## 테스트

- 노트가 **없는** 슬라이드는 노트 파트도 관계도 갖지 않는다 (픽스처 002가 그 경우다).
- 노트 파트 ↔ 슬라이드 관계가 정확한 슬라이드를 가리킨다 (파트를 풀어 rels를 읽고 단정한다).
- `[Content_Types].xml`에 모든 노트 파트의 Override가 있다 — 빠지면 PowerPoint가 복구를 건다.
- 이스케이프와 개행.
- `make test`: LibreOffice가 경고 없이 여는지. 노트 파트는 관계·Content_Types가 조금만
  틀려도 복구 경고로 드러나므로 이 게이트가 여기서 특히 값을 한다.

## 범위 밖

- 발표자 뷰(웹) — P1이다.
- 픽스처 골든·픽셀 비교 (항목 07), 문서 (항목 08).
- 노트 마스터의 타이포그래피를 테마에서 컴파일하는 일 — 자리 표시자로 두고, 그 사실과
  이유를 주석과 NOTES.md에 남긴다 (테마 패키지 포맷은 Iteration 10이다).
