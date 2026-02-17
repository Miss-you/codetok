BINARY_NAME := codetok
MODULE := github.com/Miss-you/codetok

VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
BUILD_DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

LDFLAGS := -s -w \
	-X main.version=$(VERSION) \
	-X main.commit=$(COMMIT) \
	-X main.date=$(BUILD_DATE)

GO := go
GOFLAGS := -trimpath
GOLANGCI_LINT_VERSION := v1.64.8

.PHONY: all build run clean test lint fmt vet tidy help

all: lint test build ## Run lint, test, and build

build: ## Build the binary
	$(GO) build $(GOFLAGS) -ldflags '$(LDFLAGS)' -o bin/$(BINARY_NAME) .

run: build ## Build and run
	./bin/$(BINARY_NAME)

clean: ## Remove build artifacts
	rm -rf bin/
	$(GO) clean

test: ## Run tests
	$(GO) test -race -cover ./...

lint: ## Run golangci-lint
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run ./...; \
	else \
		echo "golangci-lint not installed, skipping (install: https://golangci-lint.run/welcome/install/)"; \
	fi

fmt: ## Format code
	$(GO) fmt ./...
	@if command -v goimports >/dev/null 2>&1; then \
		goimports -w .; \
	fi

vet: ## Run go vet
	$(GO) vet ./...

tidy: ## Tidy and verify dependencies
	$(GO) mod tidy
	$(GO) mod verify

install: build ## Install binary to GOPATH/bin
	cp bin/$(BINARY_NAME) $(shell $(GO) env GOPATH)/bin/

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-15s\033[0m %s\n", $$1, $$2}'
