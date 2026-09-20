PYTHON ?= python3
VENV   := .venv
MKDOCS := $(VENV)/bin/mkdocs

# 테스트는 항상 이 이미지 안에서 돈다 — 픽셀 비교와 텍스트 메트릭은 렌더 환경이
# 결정적일 때만 의미가 있다 (docs/internals/testing.md).
#
# 컨테이너 런타임은 자동으로 고른다: docker가 실제로 동작하면 docker, 아니면 finch
# (nerdctl 호환 CLI). `command -v`로 판단하지 않는 이유는 동작하지 않는 docker 심(shim)이
# PATH에 있는 환경이 흔하기 때문이다. `make test DOCKER=podman`으로 덮어쓸 수 있다.
DOCKER  ?= $(shell docker version >/dev/null 2>&1 && echo docker || \
	(finch version >/dev/null 2>&1 && echo finch || echo docker))
IMAGE   ?= balpyo-test
STAMP   := .make/image.stamp
GOFLAGS ?=

# 컨테이너는 네트워크 없이 돈다 (GOPROXY=off) — 의존성은 호스트 모듈 캐시를 마운트해
# 공급하고, 빌드 캐시는 명명 볼륨에 남겨 매 실행마다 컴파일을 반복하지 않는다.
GOMODCACHE ?= $(shell go env GOMODCACHE)

DOCKER_RUN = $(DOCKER) run --rm \
	-v "$(CURDIR)":/src \
	-v "$(GOMODCACHE)":/gomodcache \
	-v balpyo-gocache:/gocache \
	-e GOCACHE=/gocache \
	-e GOMODCACHE=/gomodcache \
	-e GOPROXY=off \
	-e GOFLAGS="$(GOFLAGS)" \
	-w /src $(IMAGE)

.PHONY: help image deps test test-host examples review golden shell versions docs docs-serve docs-clean clean

help: ## 사용 가능한 타깃 목록
	@grep -hE '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) \
		| awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-12s\033[0m %s\n", $$1, $$2}'

image: $(STAMP) ## 테스트 이미지를 빌드한다

deps: ## 의존성을 호스트 모듈 캐시로 내린다 (컨테이너가 이 캐시를 마운트한다)
	go mod download

test: image deps ## 컨테이너 안에서 전체 테스트
	$(DOCKER_RUN) go test ./...

test-host: ## 호스트에서 테스트 (LibreOffice·Chromium 테스트는 건너뛴다)
	go test ./...

# 예제 게이트는 `make test`에도 들어 있다 (같은 테스트다). 이 타깃은 예제를 고치는 사람이
# 130초짜리 픽스처 스위트를 기다리지 않게 하는 입구다 — 새 CI 잡은 없다 (ADR 0056).
examples: image deps ## 예제 덱을 경고 0으로 빌드하는지 확인 (make test에도 포함)
	$(DOCKER_RUN) go test ./internal/testkit -run TestExamplesBuildClean -v -count=1

review: image deps ## 리뷰 갤러리 생성 → review/index.html
	$(DOCKER_RUN) env BALPYO_REVIEW=1 go test ./internal/testkit -run TestFixtures -v -count=1
	@echo "리뷰 갤러리: review/index.html"

golden: image deps ## 골든 갱신 (변경은 구현 커밋과 함께 검토한다)
	$(DOCKER_RUN) env BALPYO_UPDATE_GOLDEN=1 go test ./... -run 'TestFixtures|TestGolden' -count=1
	@git status --short -- '*/golden/*' testdata || true

shell: image ## 컨테이너 안에서 셸 열기 (디버깅)
	$(DOCKER) run --rm -it \
		-v "$(CURDIR)":/src \
		-v balpyo-gocache:/gocache -v balpyo-gomodcache:/gomodcache \
		-e GOCACHE=/gocache -e GOMODCACHE=/gomodcache \
		-w /src $(IMAGE) bash

versions: image ## 이미지에 설치된 도구 버전 (docker/versions.env 갱신용)
	$(DOCKER_RUN) cat /etc/balpyo-versions.txt

# 이미지는 docker/ 아래 파일이 바뀔 때만 다시 빌드한다. 스탬프 파일이 없으면 캐시가
# 살아 있어도 docker build를 한 번 돌린다 (몇 초).
$(STAMP): docker/test.Dockerfile docker/fonts.conf
	@mkdir -p $(dir $@)
	$(DOCKER) build -f docker/test.Dockerfile -t $(IMAGE) .
	@touch $@

docs: $(MKDOCS) ## 문서 사이트를 site/ 에 빌드
	$(MKDOCS) build --strict

docs-serve: $(MKDOCS) ## 문서 라이브 프리뷰 (http://127.0.0.1:8000)
	$(MKDOCS) serve

docs-clean: ## 빌드된 문서 사이트 삭제
	rm -rf site

clean: docs-clean ## 생성물 삭제 (이미지는 남긴다)
	rm -rf review dist .make

# 문서 빌드용 가상환경. docs/requirements.txt 가 바뀌면 다시 설치한다.
$(MKDOCS): docs/requirements.txt
	@test -x $(VENV)/bin/python || $(PYTHON) -m venv $(VENV)
	$(VENV)/bin/python -m pip install --quiet --upgrade pip
	$(VENV)/bin/python -m pip install --quiet -r docs/requirements.txt
	@touch $@
