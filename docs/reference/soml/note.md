# `<note>`

발표자 노트다. 슬라이드 콘텐츠가 아니라 **슬라이드 메타데이터**다 — 콘텐츠 트리에 들어가지
않고, 화면에 그려지지 않으며, 외관 속성도 받지 않는다.

## 속성

없다. `<note>`는 문자 데이터만 담는다.

## 예제

`testdata/fixtures/01_text/slides/003_closing.slide`:

```html
<!-- 마무리. 노트만 있고 콘텐츠가 한 줄인 슬라이드도 도는지 본다. -->
<slide id="closing">
  <text role="title" align="center">질의응답 (Q and A)</text>
  <note>질문이 없으면 하반기 로드맵 슬라이드로 되돌아간다.</note>
</slide>
```

## 렌더 결과

**웹** — 숨은 요소 `<div class="balpyo-notes" hidden>…</div>`가 슬라이드 안에 남는다.
화면에는 나오지 않지만 문서에는 있다 — 발표자 뷰(Phase 2)가 이 요소를 읽는다.

**PPTX** — 노트 슬라이드가 된다:

- 노트가 있는 슬라이드마다 `ppt/notesSlides/notesSlideN.xml` 파트가 생긴다. **N은 슬라이드
  번호와 같다** — 노트 없는 슬라이드에서는 번호가 비어 뛴다 (노트가 2번에만 없으면
  `notesSlide1`·`notesSlide3`만 있다). 별도 카운터를 두면 관계 파일과 어긋나는 순간 노트가
  엉뚱한 슬라이드에 붙는 조용한 실패가 되므로, 파트 이름을 슬라이드 번호에서 직접 유도한다.
- 노트가 하나라도 있는 덱에만 노트 마스터(`ppt/notesMasters/notesMaster1.xml`)가 생긴다.
  빈 `<note></note>`는 노트 없음으로 친다 — 웹이 `balpyo-notes`를 내지 않는 판정과 같다.
- 노트 텍스트는 본문 플레이스홀더(`p:ph type="body"`)의 문단이 된다. PowerPoint의 노트
  창과 발표자 뷰에서 그대로 보인다.

원문 개행은 콘텐츠 텍스트와 같은 규칙으로 파서가 공백으로 접는다 — 웹의 `balpyo-notes`와
PPTX의 `a:t`가 같은 문자열을 받는다.

## 빌드 에러가 나는 조건

파서(`soml.Parse`):

| 조건 | 진단 메시지 |
|---|---|
| 슬라이드에 `<note>`가 둘 이상 | `only one <note> is allowed per slide (using the one at …)` |
| 자식으로 요소를 썼다 | `only character data is allowed inside <note> (<…> is not allowed)` |
| 속성을 썼다 | `<note> takes no attributes (…)` |
