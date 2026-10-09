# 0065 — PPTX의 유효성은 LibreOffice가 아니라 ECMA-376 스키마가 판정한다 (Docker를 걷어낸다)

- 상태: 확정
- 날짜: 2026-10-02
- 마일스톤: R4 (리팩터 — TASKS.md)
- 관련: [ADR 0064](0064-the-contract-is-component-and-layout-parity-not-pixel-parity.md)(컴포넌트·레이아웃 동일성 —
  "LibreOffice는 옵트인 스모크로 남는다"를 이 ADR이 대체한다),
  [ADR 0061](0061-release-is-built-by-make-and-the-go-toolchain-and-the-version-belongs-to-the-tag.md)(릴리스 —
  "모든 게이트는 컨테이너 하나를 지난다"는 근거를 이 ADR이 대체한다),
  [ADR 0047](0047-review-gallery-lives-in-internal-preview.md),
  [testing.md](../internals/testing.md)

## 맥락

ADR 0064로 픽셀 · 잉크 · 줄 수 판정이 없어지자 테스트 Docker 이미지(Chromium · LibreOffice · poppler ·
고정 폰트 · 힌팅 끔)에 남은 일은 하나였다: **산출 PPTX가 깨지지 않았다**를 LibreOffice가 경고 · 복구
없이 여는지로 확인하는 것. 폰트 고정과 poppler는 래스터 비교용이었고, Chromium은 빌드 자체(지오메트리
패스)의 의존성이라 개발자 호스트에 이미 있다.

그 하나의 판정도 약했다. LibreOffice는 관대한 리더다 — 자식 순서가 틀리거나 열거값 밖의 속성이 있어도
조용히 받아 그린다. 사용자가 보는 복구 대화상자는 PowerPoint의 것이고, PowerPoint가 먼저 보는 것은
ECMA-376 스키마다. 결국 이미지 하나(수백 MB · 버전 핀 · 핀 대조 테스트)를 유지해서 얻는 것이 "스키마보다
느슨한 검사"였다.

## 선택지

1. **Docker + LibreOffice 유지** (ADR 0064의 옵트인 스모크). 비용은 그대로, 판정은 느슨한 채로.
2. **OOXML 라이브러리로 읽는다** — `unidoc/unioffice`는 상용(라이선스 키 · 미터링 API), 그 전신
   `baliance/gooxml`과 포크들은 AGPL이고 2019년 무렵 멈췄다. 나머지 Go PPTX 패키지(`Go-pptx` 등)는
   읽고 쓰기만 하고 유효성을 보지 않는다. 읽기는 이미 `parity.go`가 한다.
3. **스키마 판독기를 직접 쓴다** — 우리가 내는 요소(~120개)에 대한 XSD 부분집합. 의존성은 없지만
   XSD의 `sequence`/`choice`/그룹/확장/패싯을 다시 구현하는 코드가 생긴다.
4. **범용 순수 Go XSD 검증기 + ECMA-376 스키마 원본.** 후보: `knroy/go-xml`(MIT · Go 1.25+ · W3C XSD
   테스트 99.9%), `jacoelho/xsd`(MIT · Go 1.27+ — 지금 툴체인 밖), `go-xsd-validate`(libxml2 · cgo —
   시스템 의존성이 돌아온다).

## 결정

**선택지 4 — `github.com/knroy/go-xml`로 ECMA-376 Transitional 스키마 원본을 읽어 모든 XML 파트를 검증한다.**

1. 스키마는 표준의 배포본을 **고치지 않고** `internal/testkit/ooxml/`에 둔다: ECMA-376 Part 4(Transitional
   — `pml` · `dml-*` · `shared-*`), Part 2(`opc-*`), Dublin Core(`dc` · `dcterms` · `dcmitype`), W3C
   `xml.xsd`. 원격 `schemaLocation`은 카탈로그가 로컬 파일로 답한다 — 네트워크를 타지 않는다.
2. `AssertPackage`가 패키지의 모든 XML 파트를 스키마로 검증한다 (픽스처 · 예제 · `render/pptx`의 열림
   테스트 덱). 스키마가 말하지 못하는 PowerPoint 규칙(슬라이드 안 도형 id 유일 · `sldId` 범위 · 관계 종류 ·
   레이아웃 연결 · 플레이스홀더 `idx`)은 같은 자리에서 Go 코드가 본다.
3. **테스트에서 LibreOffice를 지운다** — 옵트인 스모크도 남기지 않는다. `balpyo preview`의 PPTX 래스터는
   사람이 보는 도구로 남는다 (ADR 0047 — 게이트가 아니다).
4. **Docker 이미지를 지운다.** 게이트는 호스트의 `go test ./...`이고, CI는 `setup-go` + 러너의 Chrome으로
   같은 명령을 돈다. 릴리스(`make dist`)는 호스트에서 `CGO_ENABLED=0 -trimpath`로 — ADR 0061의 재현성
   계약(같은 태그 → 같은 바이트)은 툴체인이 지키는 것이지 컨테이너가 지키는 것이 아니었다.

## 근거

- **스키마는 PowerPoint가 먼저 보는 것이다.** 복구 대화상자의 흔한 원인 — 자식 순서 · 필수 속성 ·
  열거값 · 타입 · 범위 — 는 전부 스키마 위반이고, LibreOffice는 그것을 받아 준다.
- **스파이크가 확인했다** (2026-10-02): 스키마 18개(420KB) 적재 76ms, 덱당 검증 2–26ms. 픽스처 18 ·
  예제 3의 모든 파트가 유효. 변이 12종(자식 순서 · 필수 속성 · 열거값 · 모르는 요소 · 타입 · 범위 · 모르는
  속성 · 프리셋 이름 · 필수 자식 · rels `Target` · content types 속성)을 모두 잡았고, 메시지가 요소 경로와
  기대한 자식을 댄다 (`/sld/cSld/spTree/sp/spPr/prstGeom: … "rectt" is not one of the permitted values`).
- **규칙 표를 손으로 쓰지 않는다.** 렌더러를 보고 규칙을 쓰면 렌더러의 오해가 규칙에 그대로 옮겨진다
  (R1에서 웹 쪽 환산을 렌더러 함수 없이 따로 계산한 것과 같은 이유). 표준 원본은 우리가 쓴 것이 아니다.
- **의존성은 테스트 전용이다.** `internal/testkit`만 가져오고 `balpyo` 바이너리에는 링크되지 않는다.
  순수 Go라 cgo · 시스템 라이브러리가 없다. 별이 적은 젊은 프로젝트라는 위험은 변이 테스트가 덮는다:
  검증기가 거르기를 멈추면 그 테스트가 먼저 빨개진다.

## 결과

- `docker/`(Dockerfile · `fonts.conf` · `versions.env`) · `TestImageVersionsMatchPins` · Makefile의 컨테이너
  타깃 · `make test-host` · `make smoke` · `BALPYO_SMOKE`가 없어진다. `make test` = `go test ./...`.
- Chromium이 없는 호스트에서 빌드가 필요한 테스트는 메시지와 함께 건너뛰고, CI는 `BALPYO_TEST_STRICT=1`로
  건너뛰기를 실패로 바꾼다.
- 스키마가 받는다고 PowerPoint가 반드시 조용히 여는 것은 아니다 — 스키마 밖의 규칙은 파트 사이 검사가 아는
  만큼만 덮는다. 새로 알게 된 규칙은 그 검사에 더하고, 그 입력을 변이 테스트로 남긴다.
- 렌더러가 Transitional에 없는 확장(`p14:` 등 `mc:AlternateContent`)을 내기 시작하면 그 스키마를 더하거나
  `extLst` 안으로 넣어야 한다 — 지금은 하나도 내지 않는다.
