SHELL := /usr/bin/env bash
.DEFAULT_GOAL := help

BINARY := sdkgen
OUT_DIR := ./sdk-example-output

.PHONY: help
help:
	@echo "Targets:"
	@echo "  make build        - build ./$(BINARY)"
	@echo "  make fmt          - gofmt all go files"
	@echo "  make tidy         - go mod tidy"
	@echo "  make test         - run unit tests"
	@echo "  make examples     - generate example SDKs into $(OUT_DIR)"
	@echo "  make golden       - (re)generate golden files from examples"
	@echo "  make ci           - fmt check + test (CI entrypoint)"

.PHONY: build
build:
	go build -o ./$(BINARY) ./cmd/sdkgen

.PHONY: fmt
fmt:
	gofmt -w .

.PHONY: fmt-check
fmt-check:
	@unformatted=$$(gofmt -l .); \
	if [ -n "$$unformatted" ]; then \
	  echo "gofmt needed on:"; echo "$$unformatted"; \
	  exit 1; \
	fi

.PHONY: tidy
tidy:
	go mod tidy

.PHONY: test
test:
	go test ./...

.PHONY: examples
examples: build
	./run_examples.sh

# Golden workflow:
#   make golden   -> overwrites testdata/golden/*
.PHONY: golden
golden: build
	@echo "Rebuilding golden files..."
	@rm -rf ./internal/generator/testdata/golden
	@mkdir -p ./internal/generator/testdata/golden
	@./$(BINARY) --input ./examples/swagger_jobs.json --out ./internal/generator/testdata/golden/jobs-ts --lang ts --name JobEngineSDK
	@./$(BINARY) --input ./examples/swagger_jobs.json --out ./internal/generator/testdata/golden/jobs-js --lang js --name JobEngineSDK
	@./$(BINARY) --input ./examples/swagger_misc.json --out ./internal/generator/testdata/golden/misc-ts --lang ts --name MiscSDK
	@./$(BINARY) --input ./examples/swagger_misc.json --out ./internal/generator/testdata/golden/misc-js --lang js --name MiscSDK
	@echo "✅ Golden files updated."

.PHONY: ci
ci: fmt-check test