PYTHON ?= python3
VENV   := .venv
MKDOCS := $(VENV)/bin/mkdocs

# 게이트는 호스트의 `go test ./...` 하나다 (ADR 0065) — PPTX의 유효성은 ECMA-376 스키마가
# 판정하므로 LibreOffice도, 그것을 고정할 컨테이너도 필요 없다. 빌드(지오메트리 패스)에 쓰는
# Chromium/Chrome만 있으면 된다 (없으면 그 테스트는 건너뛴다 — BALPYO_TEST_STRICT=1이면 실패).
GOFLAGS ?=

.PHONY: help test examples review golden dist docs docs-serve docs-clean clean

help: ## 사용 가능한 타깃 목록
	@grep -hE '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) \
		| awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-12s\033[0m %s\n", $$1, $$2}'

test: ## 전체 테스트 (CI의 게이트와 같은 명령)
	go test $(GOFLAGS) ./...

# 예제 게이트는 `make test`에도 들어 있다 (같은 테스트다). 이 타깃은 예제를 고치는 사람이
# 픽스처 스위트 전체를 기다리지 않게 하는 입구다 — 새 CI 잡은 없다 (ADR 0056).
examples: ## 예제 덱을 경고 0으로 빌드하는지 확인 (make test에도 포함)
	go test ./internal/testkit -run TestExamplesBuildClean -v -count=1

# 갤러리는 사람이 보는 도구다 — 테스트 스위트를 다시 돌리지 않고 `balpyo preview`를 픽스처마다
# 부른다. 한 픽스처가 실패해도 나머지는 만든다 — 실패한 픽스처는 첫 줄의 이유와 함께 알린다.
review: ## 픽스처마다 웹 · PPTX 나란히 보기 → review/<픽스처>/gallery/index.html
	@mkdir -p .make && go build -o .make/balpyo ./cmd/balpyo
	@for dir in testdata/fixtures/*/; do \
		name=$$(basename $$dir); \
		if out=$$(.make/balpyo preview --project $$dir --out review/$$name/build --gallery review/$$name/gallery 2>&1); \
			then echo "  $$name"; else echo "  $$name 실패: $$(echo "$$out" | head -n 1)"; fi; \
	done
	@echo "리뷰 갤러리: review/<픽스처>/gallery/index.html"

golden: ## 골든 갱신 (변경은 구현 커밋과 함께 검토한다)
	BALPYO_UPDATE_GOLDEN=1 go test ./... -run 'TestFixtures|TestGolden' -count=1
	@git status --short -- '*/golden/*' testdata || true

# 릴리스의 재현성(같은 태그 → 같은 바이트)은 툴체인이 지킨다 — go.mod의 Go 버전과 internal/release가
# 거는 CGO_ENABLED=0 · -trimpath (ADR 0061 · 0065). VERSION을 주지 않으면 HEAD에 붙은 태그를 읽는다.
dist: ## 릴리스 아카이브·SHA256SUMS·Homebrew 포뮬러를 dist/ 에 만든다 (VERSION=v0.3.0)
	go run ./internal/release/cmd -out dist $(if $(VERSION),-version $(VERSION),) $(if $(TARGETS),-targets $(TARGETS),)

docs: $(MKDOCS) ## 문서 사이트를 site/ 에 빌드
	$(MKDOCS) build --strict

docs-serve: $(MKDOCS) ## 문서 라이브 프리뷰 (http://127.0.0.1:8000)
	$(MKDOCS) serve

docs-clean: ## 빌드된 문서 사이트 삭제
	rm -rf site

clean: docs-clean ## 생성물 삭제
	rm -rf review dist .make

# 문서 빌드용 가상환경. docs/requirements.txt 가 바뀌면 다시 설치한다.
$(MKDOCS): docs/requirements.txt
	@test -x $(VENV)/bin/python || $(PYTHON) -m venv $(VENV)
	$(VENV)/bin/python -m pip install --quiet --upgrade pip
	$(VENV)/bin/python -m pip install --quiet -r docs/requirements.txt
	@touch $@
