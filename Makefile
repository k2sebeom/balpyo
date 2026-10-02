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
	$(GIT_SAFE_DIR) \
	-w /src $(IMAGE)

# 컨테이너는 root로 돌고 /src는 호스트 사용자의 것이므로, git이 이 저장소를 "의심스러운 소유"로
# 보고 거부한다. 그러면 `go build`의 vcs 스탬핑이 실패한다 (exit status 128) — 릴리스 테스트는
# 박힌 커밋을 보므로 이 설정이 없으면 컨테이너에서만 빨갛다. 설정 파일을 만들지 않고 환경으로
# 넣는다 (마운트한 작업 트리에 우리 설정을 남기지 않는다).
GIT_SAFE_DIR = -e GIT_CONFIG_COUNT=1 -e GIT_CONFIG_KEY_0=safe.directory -e GIT_CONFIG_VALUE_0=/src

# 릴리스 빌드는 **호스트 사용자로** 돈다 — dist/ 의 파일을 사람이 올리고 지워야 하므로 root가
# 소유하면 안 된다. 그래서 빌드 캐시도 명명 볼륨(root 소유) 대신 컨테이너 임시 디렉토리를 쓴다
# (찬 캐시로 네 타깃 40초 — 릴리스는 한 번이다).
DIST_RUN = $(DOCKER) run --rm \
	--user "$(shell id -u):$(shell id -g)" \
	-v "$(CURDIR)":/src \
	-v "$(GOMODCACHE)":/gomodcache \
	-e GOMODCACHE=/gomodcache \
	-e GOCACHE=/tmp/gocache \
	-e HOME=/tmp \
	-e GOPROXY=off \
	-w /src $(IMAGE)

.PHONY: help image deps test test-host examples review golden dist shell versions docs docs-serve docs-clean clean

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

golden: image deps ## 골든 갱신 (변경은 구현 커밋과 함께 검토한다)
	$(DOCKER_RUN) env BALPYO_UPDATE_GOLDEN=1 go test ./... -run 'TestFixtures|TestGolden' -count=1
	@git status --short -- '*/golden/*' testdata || true

# 릴리스는 테스트와 **같은 이미지**에서 나온다 — 재현성 계약이 툴체인 버전을 포함하므로,
# 게이트를 돈 Go와 아카이브를 만든 Go가 달라지면 "두 번 빌드하면 같은 바이트"가 저장소 밖에서
# 뜻을 잃는다 (ADR 0061). VERSION을 주지 않으면 HEAD에 붙은 태그를 읽는다.
dist: image deps ## 릴리스 아카이브·SHA256SUMS·Homebrew 포뮬러를 dist/ 에 만든다 (VERSION=v0.3.0)
	$(DIST_RUN) go run ./internal/release/cmd -out dist $(if $(VERSION),-version $(VERSION),) $(if $(TARGETS),-targets $(TARGETS),)

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
