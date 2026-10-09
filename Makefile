PYTHON ?= python3
VENV   := .venv
MKDOCS := $(VENV)/bin/mkdocs

# 게이트는 호스트의 `go test ./...` 하나다 — PPTX의 유효성은 ECMA-376 스키마가
# 판정하므로 LibreOffice도, 그것을 고정할 컨테이너도 필요 없다. 빌드(지오메트리 패스)에 쓰는
# Chromium/Chrome만 있으면 된다 (없으면 그 테스트는 건너뛴다 — BALPYO_TEST_STRICT=1이면 실패).
GOFLAGS ?=

.PHONY: help test examples qa showcase golden dist docs docs-serve docs-clean clean

help: ## 사용 가능한 타깃 목록
	@grep -hE '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) \
		| awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-12s\033[0m %s\n", $$1, $$2}'

test: ## 전체 테스트 (CI의 게이트와 같은 명령)
	go test $(GOFLAGS) ./...

# 예제 게이트는 `make test`에도 들어 있다 (같은 테스트다). 이 타깃은 예제를 고치는 사람이
# 픽스처 스위트 전체를 기다리지 않게 하는 입구다 — 새 CI 잡은 없다.
examples: ## 예제 덱 · 예제 테마를 경고 0으로 빌드하는지 확인 (make test에도 포함)
	go test ./internal/testkit -run 'TestExamplesBuildClean|TestExampleThemesBuildEveryDeck' -v -count=1

# 저장소 전체의 PPTX 검수: 예제 덱과 픽스처마다 `balpyo preview` 갤러리를 만든다 — 웹 스크린샷과
# OnlyOffice로 되돌린 PPTX가 나란히 놓인다 (없으면 LibreOffice). 판정은 하지 않는다. 렌더러를 고친
# 커밋은 이 갤러리를 보고 낸다. 한 프로젝트가 실패해도 나머지는 만들고, 실패는 첫 줄과 함께 알린다.
QA := preview/qa

qa: ## 예제 덱 · 픽스처마다 웹 · PPTX 검수 갤러리 → preview/qa/<이름>/index.html
	@set -e; tmp=$$(mktemp -d); trap 'rm -rf "$$tmp"' EXIT; \
	go build -o "$$tmp/balpyo" ./cmd/balpyo; rm -rf $(QA); mkdir -p $(QA); \
	for dir in examples/decks/*/ testdata/fixtures/*/; do \
		name=$$(basename $$dir); \
		if out=$$("$$tmp/balpyo" preview --project $$dir --target html,pptx --out "$$tmp/out-$$name" --gallery $(QA)/$$name 2>&1); \
			then echo "  $$name"; else echo "  $$name 실패: $$(echo "$$out" | head -n 1)"; fi; \
	done; \
	echo "검수 갤러리: $(QA)/<이름>/index.html"

# README의 쇼케이스: 예제 덱의 산출물(웹 · PPTX)과 슬라이드 스크린샷을 examples/showcase/에 만든다.
# 커밋하는 산출물이라 렌더 결과가 바뀌면 이 타깃을 다시 돌려 함께 커밋한다. 스크린샷은 웹 렌더다
# (balpyo preview --target html — 브라우저만 있으면 된다). 예제 테마는 덱마다 themes/에 넣어 둔다.
SHOWCASE        := examples/showcase
SHOWCASE_SLIDES := balpyo-intro:04 balpyo-intro:07 incident-postmortem:03 incident-postmortem:05 incident-postmortem:08 quarterly-review:05
SHOWCASE_THEMED := balpyo-intro:01 balpyo-intro:06
SHOWCASE_THEMES := minimal editorial ink harbor

showcase: ## README 쇼케이스(예제 덱 산출물 · 스크린샷)를 examples/showcase/에 다시 만든다
	@set -e; tmp=$$(mktemp -d); trap 'rm -rf "$$tmp"' EXIT; \
	go build -o "$$tmp/balpyo" ./cmd/balpyo; \
	rm -rf $(SHOWCASE)/decks $(SHOWCASE)/images; mkdir -p $(SHOWCASE)/images/themes; \
	for dir in examples/decks/*/; do \
		deck=$$(basename $$dir); proj="$$tmp/$$deck"; \
		mkdir -p "$$proj/themes"; cp -R $$dir. "$$proj/"; cp -R examples/themes/. "$$proj/themes/"; \
		"$$tmp/balpyo" build --project "$$proj" --target html,pptx --strict --out "$(CURDIR)/$(SHOWCASE)/decks/$$deck"; \
		"$$tmp/balpyo" preview --project "$$proj" --target html --out "$$tmp/out-$$deck" --gallery "$$tmp/gallery-$$deck" >/dev/null; \
	done; \
	for pick in $(SHOWCASE_SLIDES); do \
		deck=$${pick%%:*}; n=$${pick##*:}; \
		cp "$$tmp"/gallery-$$deck/*/$$n-web.png $(SHOWCASE)/images/$$deck-$$n.png; \
	done; \
	for theme in $(SHOWCASE_THEMES); do \
		for deck in $$(printf '%s\n' $(SHOWCASE_THEMED) | cut -d: -f1 | sort -u); do \
			"$$tmp/balpyo" preview --project "$$tmp/$$deck" --theme $$theme --target html --out "$$tmp/out-$$deck-$$theme" --gallery "$$tmp/gallery-$$deck-$$theme" >/dev/null; \
		done; \
		for pick in $(SHOWCASE_THEMED); do \
			deck=$${pick%%:*}; n=$${pick##*:}; \
			cp "$$tmp"/gallery-$$deck-$$theme/*/$$n-web.png $(SHOWCASE)/images/themes/$$deck-$$n-$$theme.png; \
		done; \
	done; \
	echo "쇼케이스: $(SHOWCASE)/"

golden: ## 골든 갱신 (변경은 구현 커밋과 함께 검토한다)
	BALPYO_UPDATE_GOLDEN=1 go test ./... -run 'TestFixtures|TestGolden' -count=1
	@git status --short -- '*/golden/*' testdata || true

# 릴리스의 재현성(같은 태그 → 같은 바이트)은 툴체인이 지킨다 — go.mod의 Go 버전과 internal/release가
# 거는 CGO_ENABLED=0 · -trimpath. VERSION을 주지 않으면 HEAD에 붙은 태그를 읽는다.
dist: ## 릴리스 아카이브·SHA256SUMS·Homebrew 포뮬러를 dist/ 에 만든다 (VERSION=v0.3.0)
	go run ./internal/release/cmd -out dist $(if $(VERSION),-version $(VERSION),) $(if $(TARGETS),-targets $(TARGETS),)

docs: $(MKDOCS) ## 문서 사이트를 site/ 에 빌드
	$(MKDOCS) build --strict

docs-serve: $(MKDOCS) ## 문서 라이브 프리뷰 (http://127.0.0.1:8000)
	$(MKDOCS) serve

docs-clean: ## 빌드된 문서 사이트 삭제
	rm -rf site

clean: docs-clean ## 생성물 삭제
	rm -rf dist

# 문서 빌드용 가상환경. docs/requirements.txt 가 바뀌면 다시 설치한다.
$(MKDOCS): docs/requirements.txt
	@test -x $(VENV)/bin/python || $(PYTHON) -m venv $(VENV)
	$(VENV)/bin/python -m pip install --quiet --upgrade pip
	$(VENV)/bin/python -m pip install --quiet -r docs/requirements.txt
	@touch $@
