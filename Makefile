.PHONY: help build test fmt lint install clean test-telemetry setup-hourglass

GIT_HOURGLASS_REPO=https://github.com/Layr-Labs/hourglass-monorepo.git
HOURGLASS_DIR=./temp_external/hourglass-monorepo
APP_NAME=devkit
GO_PACKAGES=./pkg/... ./cmd/...

help: ## Show available commands
	@grep -E '^[a-zA-Z0-9_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

setup-hourglass: ## Clone and wire up hourglass-monorepo locally
	@echo "📦 Cloning hourglass-monorepo to $(HOURGLASS_DIR)..."
	@rm -rf $(HOURGLASS_DIR)
	@git clone --depth=1 $(GIT_HOURGLASS_REPO) $(HOURGLASS_DIR)
	@echo "🔧 Replacing Go module path to use local clone..."
	@go mod edit -replace=github.com/Layr-Labs/hourglass-monorepo=$(HOURGLASS_DIR)
	@go mod tidy
	@echo "✅ hourglass-monorepo linked successfully"

build: ## Build the binary
	@go build -o $(APP_NAME) cmd/$(APP_NAME)/main.go

tests: ## Run tests
	@go test $(GO_PACKAGES)

test-telemetry: ## Run telemetry tests
	@go test ./pkg/telemetry/...

fmt: ## Format code
	@go fmt $(GO_PACKAGES)

lint: ## Run linter
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@golangci-lint run $(GO_PACKAGES)

install: setup-hourglass build ## Install binary to ~/bin
	@mkdir -p ~/bin
	@mv $(APP_NAME) ~/bin/

clean: ## Remove binary
	@rm -f $(APP_NAME) ~/bin/$(APP_NAME) 

dump-state:
	./contracts/anvil/dump-state.sh
