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

.PHONY: smoke
smoke: golden
	cd tests/sdk-smoke && npm install
	@rm -rf internal/generator/testdata/golden/node_modules
	@ln -sfn "$$(pwd)/tests/sdk-smoke/node_modules" internal/generator/testdata/golden/node_modules
	cd tests/sdk-smoke && npm run test

# Golden workflow:
#   make golden   -> overwrites testdata/golden/*
.PHONY: golden
golden: build
	@echo "Rebuilding golden files..."
	@rm -rf ./internal/generator/testdata/golden
	@mkdir -p ./internal/generator/testdata/golden
	@./$(BINARY) --input ./examples/swagger_telephone.json --out ./internal/generator/testdata/golden/telephone-ts --lang ts --name TelephoneSDK
	@./$(BINARY) --input ./examples/swagger_telephone.json --out ./internal/generator/testdata/golden/telephone-js --lang js --name TelephoneSDK
	@./$(BINARY) --input ./examples/swagger_dog_parlor.json --out ./internal/generator/testdata/golden/dog-parlor-ts --lang ts --name DogParlorSDK
	@./$(BINARY) --input ./examples/swagger_dog_parlor.json --out ./internal/generator/testdata/golden/dog-parlor-js --lang js --name DogParlorSDK
	@./$(BINARY) --input ./examples/swagger_customer_booking.json --out ./internal/generator/testdata/golden/customer-booking-ts --lang ts --name CustomerBookingSDK
	@./$(BINARY) --input ./examples/swagger_customer_booking.json --out ./internal/generator/testdata/golden/customer-booking-js --lang js --name CustomerBookingSDK

	@echo "✅ Golden files updated."

.PHONY: ci
ci: fmt fmt-check golden test smoke