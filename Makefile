SHELL := /bin/bash

.DEFAULT_GOAL := all

.PHONY: all
all: ## build pipeline
all: mod build lint test

.PHONY: precommit
precommit: ## validate the branch before commit
precommit: all

.PHONY: ci
ci: ## CI build pipeline
ci: precommit diff

.PHONY: help
help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

.PHONY: clean
clean: ## remove files created during build pipeline
	rm -rf dist
	rm -f coverage.*
	go clean -i -cache -testcache -fuzzcache -x

.PHONY: run
run: ## go run
	go run .

.PHONY: mod
mod: ## go mod tidy
	go mod tidy

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)

.PHONY: build
build: ## go build
	go build -ldflags "-X github.com/antopolskiy/kanban-md/cmd.version=$(VERSION)" -o dist/kanban-md .

.PHONY: lint
lint: ## golangci-lint
	golangci-lint run --fix

ifeq ($(CGO_ENABLED),0)
RACE_OPT =
else
RACE_OPT = -race
endif

.PHONY: test
test: ## go test
	go test $(RACE_OPT) -covermode=atomic -coverprofile=coverage.out -coverpkg=./... ./...
	go tool cover -html=coverage.out -o coverage.html

.PHONY: test-e2e
test-e2e: ## e2e tests
	go test $(RACE_OPT) -v ./e2e/

.PHONY: setup-hooks
setup-hooks: ## install git pre-commit hook
	git config core.hooksPath .githooks

.PHONY: diff
diff: ## git diff
	git diff --exit-code
	RES=$$(git status --porcelain) ; if [ -n "$$RES" ]; then echo $$RES && exit 1 ; fi
