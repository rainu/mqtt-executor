SHELL:=/bin/bash

ifneq ("$(wildcard ./.env)","")
  $(info Environment variable file .env found)
  include .env
endif

# Directories
BUILD_DIR 			:= build
BIN_DIR 			:= ${BUILD_DIR}/bin
DIST_DIR			:= ${BUILD_DIR}/dist

GO := go
GO_VERSION=`go version`
MAIN_GO=main.go

# Project parameters
BINARY_NAME=mqtt-executor
DOCKER_REGISTRY=eloo

.PHONY: init
init:
	git config core.hooksPath .githooks

.PHONY: clean
clean:
	@rm -rf ${BUILD_DIR} || true

.PHONY: test
test: 
	$(info Running all Go unit tests...)
	go test -parallel 1 -count 1 -cpu 1 -tags slow -timeout 20m ./... -v

.PHONY: lint
lint:
	golangci-lint run ./...

.PHONY: update-deps
update-deps:
	$(GO) get -u -t ./...
	make tidy

.PHONY: download-tools
download-tools:
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(shell go env GOPATH)/bin
	curl -sfL https://install.goreleaser.com/github.com/goreleaser/goreleaser.sh | sh -s -- -b $(shell go env GOPATH)/bin
	curl -sfL https://install.goreleaser.com/github.com/caarlos0/svu.sh | sh -s -- -b $(shell go env GOPATH)/bin

	make tidy

.PHONY: build
build: 
	goreleaser build --snapshot --rm-dist

.PHONY: build-snapshot
build-snapshot:
	goreleaser release --snapshot --rm-dist

.PHONY: build-release
build-release:
	goreleaser release --rm-dist

.PHONY: fmt
fmt:
	$(GO) fmt ./...

.PHONY: tidy
tidy:
	$(GO) mod tidy

help: ## Prints usage help for this makefile
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = "(:|: .*?## )"}; {printf "\033[36m%-30s\033[0m %s\n", $$(NF-1), $$(NF)}'

.PHONY: bump-patch
bump-patch:
	svu next --force-patch-increment | tee > VERSION
	git tag $$(cat VERSION)
	git push --tags

.PHONY: bump-minor
bump-minor:
	svu minor | tee > VERSION
	git tag $$(cat VERSION)
	git push origin $$(cat VERSION)