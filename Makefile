.PHONY: sample build serve
.DEFAULT_GOAL := help

TEST_FORMAT ?= pkgname

.PHONY: install
install: ## Install dependencies
	@go install golang.org/x/lint/golint@latest
	@go install gotest.tools/gotestsum@latest

.PHONY: lint
lint: ## Run linters
	@golint -set_exit_status ./vm/...

.PHONY: test
test: ## Run tests
	@gotestsum --format $(TEST_FORMAT) --format-hide-empty-pkg --hide-summary skipped --raw-command -- go test -tags native -json -race -cover ./...

.PHONY: serve
serve: ## Serve the sample
	python3 -m http.server 8222 -d sample

.PHONY: build
build: ## Build the project
	@CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o build/wasmpack ./cmd

.PHONY: help
help: ## Shows this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'