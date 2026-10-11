# 백로그

아직 하지 않은 일의 목록이다. **순서는 정하지 않았다** — 집어 들 때 정한다. 구현된 동작은
[레퍼런스](../docs/reference/index.md)에, 계약을 바꾸는 결정은 [ADR](decisions/index.md)에 있다.
여기 있는 것은 어느 쪽에도 아직 없다.

항목을 끝내면 이 목록에서 지운다 — 끝난 일의 기록은 커밋 로그와 ADR이 맡는다.

첫 배포 전 도그푸딩(덱 작성 에이전트 + 테마 설계 에이전트가 skill과 공개 문서만 보고 덱 하나와
독립 테마 하나를 만든 시험)에서 나온 항목이다. 각 항목은 재현 입력과 함께 적었다.

## 진단

- **테마 JSON 오류의 자리.** `components.json`의 모르는 필드가 `balpyo.json: error: theme "x":
  cannot parse components.json: json: unknown field "lineHeight"`로 나온다 — 파일이 `balpyo.json`으로
  찍히고 JSON 안의 경로가 없어서 저자가 항목을 하나씩 지워 가며 찾았다. 실제 파일 경로와 JSON 경로
  (`variants.stat.lineHeight`)를 말한다.
- **`$comment-*` 키의 일관성.** `tokens.json`은 `$comment-accent` 같은 키를 받는데 `components.json`은
  `unknown field "$comment-defaults"`로 거부한다. 문서는 "`$comment` 예외"라고만 한다. 두 파일이 같은
  규칙을 갖게 한다.
- **차트 라벨 진단의 이름.** 펼친 차트의 라벨이 넘치면 진단이 `<text>`를 말한다 — 저자가 쓴 것은
  `<point>`다. 합성 노드의 진단은 저자가 쓴 태그 이름으로 말한다.
- **첫 경고가 원인이 아닌 상자를 가리킨다.** 너무 낮은 `<shape h="96">` 안의 `<col>`이 처음에는
  `<col> leaves the stage at the bottom by 11px`로 나왔고, 진짜 원인(`<col> overflows its parent
  <shape>`)은 배치를 옮긴 뒤에야 나왔다. 부모 넘침을 스테이지 이탈보다 먼저 말한다.
- **`build`는 배치 검사를 하지 않는다는 안내.** `balpyo build`(`--check-layout` 없이)는 배치 오류가
  있어도 조용히 성공한다. 문서에는 있지만 처음 쓰는 저자가 깨진 덱을 낸다 — 출력 끝에 한 줄 안내를
  고려한다.

## 테마

- **`components.json`의 필드 목록을 레퍼런스에.** variant·componentDefaults가 받는 필드가 문서에
  없어서 시도와 에러로 알아냈다.
- **variant·슬롯 기본값의 `font`/`lineHeight`.** 지금은 테마만으로 stat·quote 슬롯을 제목 서체로 바꿀
  수 없어 저자가 슬라이드마다 `role="heading"`을 적는다.
- **`minimal`의 `image-full` 크롬.** footer·쪽 번호가 전면 그림의 캡션 위에 앉는다 — `skipLayouts`에
  `image-full`이 없다. 또 `<image slot="media" fit="cover">`만으로는 가장자리까지 차지 않고
  `w="fill" h="fill"`이 필요한데 레이아웃 문서에 없다.
- **`tint()`가 채도를 유지한다는 경고를 테마 skill에.** 채도 높은 강조색의 tint가 형광색이 된다 —
  `appearance.md`에만 있다.

## 작성

- **챕터의 실체.** 하위 디렉토리는 지금 정렬 순서만 바꾼다 (PPTX `p14:sectionLst`도, 뷰어 오버뷰
  묶음도 없다). 기능을 붙이거나 문서에서 "정렬용"이라고 분명히 말한다.
- **차트 값의 표기.** `unit`은 꼬리뿐이라 `$1,850`을 쓸 수 없다 (`value="1,850"`은 에러). 앞붙이와
  천 단위 구분을 연다.
- **표 글자 크기.** `th`/`td`가 `size`를 받지 않아 표를 맞추는 길이 글을 줄이는 것뿐이다.
- **chevron의 기본 노치.** 노치 깊이가 상자 폭의 절반(`adj` 261113)이라 `<shape type="chevron"
  w="fill" h="88">` 안의 라벨이 실루엣 밖으로 나가고, 글은 위쪽에 붙는다. PowerPoint 기본처럼 얕은
  노치를 기본으로 하거나 skill에 경고를 넣는다.

## 검토

- **정적 슬라이드 뷰.** 딥링크 스크린숏은 `enter` 애니메이션 전의 반쯤 빈 화면을 찍는다. 모든 스텝이
  전개된 정적 모드(또는 skill에 print-to-PDF 절차)가 필요하다.
- **인쇄가 좌표 배치를 빠뜨린다.** headless Chrome `--print-to-pdf`에서 `x`/`y`로 놓은 canvas·stack
  자식이 사라진다 (화면에서는 그려진다). `ec2-intro` 2·9·10·11장이 거의 빈 장으로 인쇄된다.
- **`preview` 스크린숏이 애니메이션 중간을 찍는다.** `make showcase`를 두 번 돌리면 `ec2-intro-10.png`가
  한 번은 연결선 두 개(`terminate`, shutting-down → terminated)가 빠진 채로 나온다 — HTML은 같은 바이트다.
  `screenshot`이 `enter`/`draw` 전환이 끝나기를 기다리지 않는다. 찍기 전에 모든 스텝을 전개하고
  애니메이션을 끈다.
- **래스터라이저 없는 `preview`.** OnlyOffice·LibreOffice가 없으면 PPTX를 볼 길이 없다는 한 줄만 나오고
  산출물을 프로젝트 `dist/`에 쓴다. 무엇을 설치하면 되는지 말한다.
- **영어 레퍼런스.** skill이 가리키는 깊은 문서가 한국어뿐이라 영어 사용자는 `reveal`·chevron·차트
  주의사항에 닿기 어렵다.
