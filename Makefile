.PHONY: help build test fmt lint install clean test-telemetry

APP_NAME=devkit

VERSION_PKG=github.com/Layr-Labs/devkit-cli/internal/version
TELEMETRY_PKG=github.com/Layr-Labs/devkit-cli/pkg/telemetry
COMMON_PKG=github.com/Layr-Labs/devkit-cli/pkg/common

LD_FLAGS=\
  -X '$(VERSION_PKG).Version=$(shell cat VERSION)' \
  -X '$(VERSION_PKG).Commit=$(shell git rev-parse --short HEAD)' \
  -X '$(TELEMETRY_PKG).embeddedTelemetryApiKey=$${TELEMETRY_TOKEN}' \
  -X '$(COMMON_PKG).embeddedDevkitReleaseVersion=$(shell cat VERSION)'

GO_PACKAGES=./pkg/... ./cmd/...
ALL_FLAGS=
GO_FLAGS=-ldflags "$(LD_FLAGS)"
GO=$(shell which go)
BIN=./bin

help: ## Show available commands
	@grep -E '^[a-zA-Z0-9_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

build: ## Build the binary
	@go build $(GO_FLAGS) -o $(BIN)/$(APP_NAME) cmd/$(APP_NAME)/main.go

tests: ## Run tests
	$(GO) test -v ./... -p 1

tests-fast: ## Run fast tests (skip slow integration tests)
	$(GO) test -v ./... -p 1 -timeout 5m -short

fmt: ## Format code
	@go fmt $(GO_PACKAGES)

lint: ## Run linter
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@golangci-lint run $(GO_PACKAGES)

install: build ## Install binary to ~/bin
	@mkdir -p ~/bin
	@cp $(BIN)/$(APP_NAME) ~/bin/
	@if ! npm list -g @layr-labs/zeus@1.5.2 >/dev/null 2>&1; then \
		echo "Installing @layr-labs/zeus@1.5.2..."; \
		npm install -g @layr-labs/zeus@1.5.2; \
	fi

clean: ## Remove binary
	@rm -f $(APP_NAME) ~/bin/$(APP_NAME) 

build/darwin-arm64:
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 $(ALL_FLAGS) $(GO) build $(GO_FLAGS) -o release/darwin-arm64/devkit cmd/$(APP_NAME)/main.go

build/darwin-amd64:
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 $(ALL_FLAGS) $(GO) build $(GO_FLAGS) -o release/darwin-amd64/devkit cmd/$(APP_NAME)/main.go

build/linux-arm64:
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 $(ALL_FLAGS) $(GO) build $(GO_FLAGS) -o release/linux-arm64/devkit cmd/$(APP_NAME)/main.go

build/linux-amd64:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(ALL_FLAGS) $(GO) build $(GO_FLAGS) -o release/linux-amd64/devkit cmd/$(APP_NAME)/main.go


.PHONY: release
release:
	$(MAKE) build/darwin-arm64
	$(MAKE) build/darwin-amd64
	$(MAKE) build/linux-arm64
	$(MAKE) build/linux-amd64
