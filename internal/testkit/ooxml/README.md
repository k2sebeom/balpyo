# ECMA-376 스키마 (테스트 전용)

`internal/testkit`의 PPTX 스키마 검증이 읽는 XSD 원본이다. **고치지 않는다** — 렌더러를 보고
규칙을 쓰면 렌더러의 오해가 규칙에 옮겨지므로, 판정의 근거는 표준 배포본 그대로여야 한다. 원격
`schemaLocation`은 고치는 대신 `schema.go`의 카탈로그가 이름공간으로 답한다.

| 파일 | 출처 |
|---|---|
| `pml.xsd` · `dml-*.xsd` · `shared-*.xsd` | ECMA-376 5판 Part 4 (2016-12) `OfficeOpenXML-XMLSchema-Transitional.zip` — https://ecma-international.org/publications-and-standards/standards/ecma-376/ |
| `opc-*.xsd` | ECMA-376 5판 Part 2 (2021-12) `OpenPackagingConventions-XMLSchema.zip` — 같은 페이지 |
| `dc.xsd` · `dcterms.xsd` · `dcmitype.xsd` | Dublin Core 2008-02-11 — https://www.dublincore.org/schemas/xmls/qdc/2008/02/11/ |
| `xml.xsd` | W3C — https://www.w3.org/2001/xml.xsd |

`dml-*`는 `pml` · `dml-main`이 가져오는 닫힘이다 (차트 · 다이어그램 등은 렌더러가 내지 않지만
`dml-main`의 import가 요구한다). 새 파일이 필요해지면 같은 배포본에서 가져오고 이 표에 더한다.

SHA-256:

```
cb06610457a2d840e8ad5ff6c312ef319e95f0d969791c852ed069b1ee8b50a8  dc.xsd
04d8303aca4246d73dea2122467bef648c89bc6e43160990dc903e8bd0db19cc  dcmitype.xsd
d23102baf91c8df732f52a230f5fbf5307313ac9112d1374a038c297b2d42e96  dcterms.xsd
4de4390a26e7e44e682ed98b39872cce2620f807f5b0de421567a43f74526388  dml-chart.xsd
3fd0586f2637b98bb9886f0e0b67d89e1cc987c2d158cc7deb5f5b9890ced412  dml-chartDrawing.xsd
809f77f658f71e10f5cfbf99e25c895a23a4e62c81210b59a2fe8c7fb24849d2  dml-diagram.xsd
5cb76dabd8b97d1e9308a1700b90c20139be4d50792d21a7f09789f5cccd6026  dml-lockedCanvas.xsd
6978ba7e889070b0c3cb5b546b23e5a6c3516134afc53b87a21f482ca33f3858  dml-main.xsd
5d389d42befbebd91945d620242347caecd3367f9a3a7cf8d97949507ae1f53c  dml-picture.xsd
49ec1c03440a9b5590e69acdf92b200fe133697de0b05f43a129fb94d1885026  opc-contentTypes.xsd
69688ea15f82a870053b9a60c79359063dbcfeec9cdabafabad51a53934bc35b  opc-coreProperties.xsd
607ffb6879cf5d5d765bfdfb2548f3f427ea964ddee6d08a15f0dc781749866d  opc-relationships.xsd
39c6396b7b1f096885c3ceba5dab30e3ee14291c45a1d6fa6038ec4db69d88d0  pml.xsd
48675c4f82f6b097434d4b7e313635b790257cca58fa90dd0e8f19c9affa18ed  shared-commonSimpleTypes.xsd
bc92e36ccd233722d4c5869bec71ddc7b12e2df56059942cce5a39065cc9c368  shared-documentPropertiesExtended.xsd
7b5b7413e2c895b1e148e82e292a117d53c7ec65b0696c992edca57b61b4a74b  shared-documentPropertiesVariantTypes.xsd
12264f3c03d738311cd9237d212f1c07479e70f0cbe1ae725d29b36539aef637  shared-relationshipReference.xsd
61960fb3131e38022caad5360e2f33a3382578ab3c80cd58bd74320ede61b20c  xml.xsd
```
