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
	rm -f coverage.* coverage-unit.out coverage-e2e.out
	rm -rf coverage-e2e-raw
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
	go build -ldflags "-X github.com/antopolskiy/kanban-md/cmd.version=$(VERSION)" -o dist/kanban-md ./cmd/kanban-md

.PHONY: lint
lint: ## golangci-lint
	golangci-lint run --fix

ifeq ($(CGO_ENABLED),0)
RACE_OPT =
else
RACE_OPT = -race
endif

.PHONY: test
test: ## unit + e2e tests with merged coverage
	@# 1. Run unit tests (excludes e2e) with coverage profile.
	go test $(RACE_OPT) -covermode=atomic -coverprofile=coverage-unit.out -coverpkg=./... $$(go list ./... | grep -v /e2e)
	@# 2. Run e2e tests with GOCOVERDIR to capture binary coverage.
	@rm -rf coverage-e2e-raw && mkdir -p coverage-e2e-raw
	GOCOVERDIR=$$(pwd)/coverage-e2e-raw go test $(RACE_OPT) ./e2e/
	@# 3. Convert e2e raw coverage to textfmt.
	go tool covdata textfmt -i=coverage-e2e-raw -o=coverage-e2e.out
	@# 4. Merge both profiles (unit header + both bodies).
	@head -1 coverage-unit.out > coverage.out
	@tail -n +2 coverage-unit.out >> coverage.out
	@tail -n +2 coverage-e2e.out >> coverage.out
	go tool cover -html=coverage.out -o coverage.html
	@rm -rf coverage-e2e-raw coverage-unit.out coverage-e2e.out

.PHONY: test-unit
test-unit: ## unit tests only (no e2e)
	go test $(RACE_OPT) -covermode=atomic -coverprofile=coverage.out -coverpkg=./... $$(go list ./... | grep -v /e2e)
	go tool cover -html=coverage.out -o coverage.html

.PHONY: test-e2e
test-e2e: ## e2e tests only
	go test $(RACE_OPT) -v ./e2e/

.PHONY: setup-hooks
setup-hooks: ## install git pre-commit hook
	git config core.hooksPath .githooks

.PHONY: diff
diff: ## git diff
	git diff --exit-code
	RES=$$(git status --porcelain) ; if [ -n "$$RES" ]; then echo $$RES && exit 1 ; fi
