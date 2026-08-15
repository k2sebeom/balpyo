PYTHON ?= python3
VENV   := .venv
MKDOCS := $(VENV)/bin/mkdocs

.PHONY: help docs docs-serve docs-clean

help: ## 사용 가능한 타깃 목록
	@grep -hE '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) \
		| awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-12s\033[0m %s\n", $$1, $$2}'

docs: $(MKDOCS) ## 문서 사이트를 site/ 에 빌드
	$(MKDOCS) build --strict

docs-serve: $(MKDOCS) ## 문서 라이브 프리뷰 (http://127.0.0.1:8000)
	$(MKDOCS) serve

docs-clean: ## 빌드된 문서 사이트 삭제
	rm -rf site

# 문서 빌드용 가상환경. docs/requirements.txt 가 바뀌면 다시 설치한다.
$(MKDOCS): docs/requirements.txt
	@test -x $(VENV)/bin/python || $(PYTHON) -m venv $(VENV)
	$(VENV)/bin/python -m pip install --quiet --upgrade pip
	$(VENV)/bin/python -m pip install --quiet -r docs/requirements.txt
	@touch $@
