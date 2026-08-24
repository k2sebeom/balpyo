# 0027 — `alt`을 열고 `frame`을 닫는다 (속성을 여는 기준 두 가지)

- 상태: 확정
- 날짜: 2026-08-24
- Iteration: 5 (항목 01·02·04 — NOTES.md 05/01·05/02·05/04가 근거 기록이다)
- 관련: PRD §7.5(갱신), §7.6, [외관 속성](../reference/appearance.md),
  [`<image>`](../reference/soml/image.md)

## 맥락

PRD §7.5의 `<image>` 행은 주요 속성을 `src`, `fit(cover/contain)`, `frame`으로 적었다. 어휘를
열면서 두 속성이 함께 문제가 되었다.

- **`frame`**: 뜻이 PRD 어디에도 정의되어 있지 않다 — §7.6의 외관 속성 표에 `frame` 행이 없다.
  이름에서 읽히는 것은 "액자", 즉 테두리 + 모서리 + 그림자의 묶음이다.
- **`alt`**: PRD의 표에 없다. 그런데 웹 `<img>`와 OOXML `p:cNvPr@descr` 양쪽에 실을 자리가 있다.

한 iteration에서 속성 하나를 **닫고** 하나를 **연** 것이므로, 기준이 무엇이었는지가 남아야 한다.

## 선택지

`frame`에 대해:

1. **`frame`을 boolean으로 연다** (`<image frame/>` = 테마의 액자).
2. **`frame`을 닫고 `variant`가 그 뜻을 갖게 한다** (`<image variant="framed">`).
3. `frame`을 외관 속성으로 정식화한다 (§7.6에 행을 만들고 값 어휘를 정한다).

`alt`에 대해:

4. **열지 않는다** (PRD 표에 없다).
5. **연다.**

## 결정

**선택지 2 + 5.** `frame`은 어휘에서 빠지고 액자는 테마의 variant(`framed` = `border: 1 solid
muted` + `radius: md` + `shadow: soft`)가 소유한다. `alt`은 열고, 생략 시 웹이 `alt=""`를 낸다.
PRD §7.5의 표를 그대로 갱신했다.

## 근거

두 결정이 한 ADR에 있는 이유는 기준이 같은 쌍이기 때문이다 — **양 타깃에 실을 자리가 있는가**,
그리고 **같은 것을 두 이름으로 받지 않는가**.

- **`alt`은 양 타깃에 자리가 있다.** 웹 `<img alt>`, PPTX `p:cNvPr@descr`(PowerPoint의 "대체
  텍스트" 칸)이다. 자리가 있는데 열지 않으면 산출물이 접근성 없는 그림을 내고 **저자가 고칠 방법이
  아예 없다** — 표에 없다는 것보다 그 사실이 무겁다. PRD는 계약이고 문서는 살아 있는 판이므로,
  뒤집는 대신 PRD를 갱신했다.
- **`frame`은 `variant`와 같은 채널이다.** 액자가 정하는 세 조각은 이미 외관 채널에 있고
  `variant`가 그 묶음의 이름이다. 열면 캐스케이드에 단이 하나 늘고 `variant`와 `frame`이 충돌할 때
  누가 이기는지 정해야 하는데, 그 규칙은 저자가 외울 이유가 없다. 선택지 3(정식화)은 그 위에 값
  어휘까지 만드는 일이다 — 그러면 `frame="1 solid muted md soft"` 같은 표기가 생기고, 그것은
  `border`+`radius`+`shadow`를 한 문자열에 눌러 담은 것에 지나지 않는다.
- **번들이 테마의 것이라는 사실이 액자의 뜻에 맞는다.** 저자가 `border`만 쓰면 액자가 아니라 선
  하나다 — 세 조각을 **함께** 정하는 것이 액자이고, 그 조합을 정하는 것은 디자인 시스템의 일이다
  (PRD §13-6: variant 어휘는 테마의 사유물이다).
- **같은 기준이 `fill`도 닫았다.** PRD §7.6 표의 `fill` 적용 대상에 image가 없고, 그림 뒤의 칠이
  보이는 자리는 `contain`의 여백뿐인데 그 여백은 PPTX에서 `pic`의 박스 밖이다 — 받아 놓고 한
  타깃에서만 그리는 속성이 된다.

## 결과

- `soml/image.go`(`imageOwnAttrs` = `src fit alt`, `imageAppearanceAttrs`에 `fill` 없음),
  `core/theme.go`(`imageVariants`의 `framed`)가 구현이다. 픽스처
  `05_image/slides/007_variant.slide`가 이 결정의 실물 증거다 — 액자 없음 / `variant="framed"` /
  variant + 인라인 `border`의 세 칸이 캐스케이드 순서까지 보여준다.
- **`frame`을 쓴 덱은 진단을 받는다**: `<image>에 알 수 없는 속성 frame (쓸 수 있는 속성: src,
  fit, alt, …)`. 어휘를 나열하므로 저자가 `variant`를 찾을 수 있다.
- **빈 `alt`은 뜻이 있다.** `alt=""`는 에러가 아니라 "장식 이미지"의 표기이고, 생략과 같은 확정
  값이 된다 (둘 다 웹에서 `alt=""`로 나간다). 값 **없는** `alt`(`<image alt/>`)은 에러다 — 뜻을
  정할 수 없는 형태다.
- **두 타깃이 여기서 비대칭이다.** 웹은 `alt`을 언제나 내지만 PPTX는 `alt`이 있을 때만 `descr`을
  낸다. OOXML에는 "장식"의 표기가 없어 `descr=""`는 "대체 텍스트가 비었다"이고(PowerPoint의
  "장식용으로 표시"는 확장 요소다) 빈 속성은 재저장 diff의 잡음이 된다 — 이 비대칭은 두 **포맷**의
  차이이고 두 타깃의 갈라짐이 아니다.
- 열린 것 하나: PowerPoint에서 `descr`이 대체 텍스트 칸에 실제로 보이는지는 사람이 확인한다
  (TASKS의 Iteration 5 리뷰 항목). 자동 검사가 볼 수 있는 것은 "XML에 속성이 있다"까지다.
