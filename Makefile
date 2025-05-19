.PHONY: help build test fmt lint install clean test-telemetry

APP_NAME=devkit

GO_PACKAGES=./pkg/... ./cmd/...
ALL_FLAGS=
GO_FLAGS=-ldflags "-X 'github.com/Layr-Labs/devkit-cli/internal/version.Version=$(shell cat VERSION)' -X 'github.com/Layr-Labs/devkit-cli/internal/version.Commit=$(shell git rev-parse --short HEAD)'"
GO=$(shell which go)

help: ## Show available commands
	@grep -E '^[a-zA-Z0-9_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

build: ## Build the binary
	@go build $(GO_FLAGS) -o $(APP_NAME) cmd/$(APP_NAME)/main.go

tests: ## Run tests
	@go test $(GO_PACKAGES)

test-telemetry: ## Run telemetry tests
	@go test ./pkg/telemetry/...

fmt: ## Format code
	@go fmt $(GO_PACKAGES)

lint: ## Run linter
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@golangci-lint run $(GO_PACKAGES)

install: build ## Install binary to ~/bin
	@mkdir -p ~/bin
	@mv $(APP_NAME) ~/bin/

clean: ## Remove binary
	@rm -f $(APP_NAME) ~/bin/$(APP_NAME) 

build/darwin-arm64:
	GOOS=darwin GOARCH=arm64 $(ALL_FLAGS) $(GO) build $(GO_FLAGS) -o release/darwin-arm64/devkit cmd/$(APP_NAME)/main.go

build/darwin-amd64:
	GOOS=darwin GOARCH=amd64 $(ALL_FLAGS) $(GO) build $(GO_FLAGS) -o release/darwin-amd64/devkit cmd/$(APP_NAME)/main.go

build/linux-arm64:
	GOOS=linux GOARCH=arm64 $(ALL_FLAGS) $(GO) build $(GO_FLAGS) -o release/linux-arm64/devkit cmd/$(APP_NAME)/main.go

build/linux-amd64:
	GOOS=linux GOARCH=amd64 $(ALL_FLAGS) $(GO) build $(GO_FLAGS) -o release/linux-amd64/devkit cmd/$(APP_NAME)/main.go


.PHONY: release
release:
	$(MAKE) build/darwin-arm64
	$(MAKE) build/darwin-amd64
	$(MAKE) build/linux-arm64
	$(MAKE) build/linux-amd64
