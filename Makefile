# ═══════════════════════════════════════════════════════════
# 🐉🔥 DrogonClaw — Go Build System
# ═══════════════════════════════════════════════════════════

BINARY_NAME := drogonclaw
MODULE := github.com/0xP4X/drogonclaw-go
BUILD_DIR := ./bin
CMD_DIR := ./cmd/drogonclaw
LDFLAGS := -s -w
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME := $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')
LDFLAGS += -X main.version=$(VERSION) -X main.buildTime=$(BUILD_TIME)

.PHONY: help build build-linux build-all test test-cover lint format clean setup \
        run daemon docker docker-compose skills doctor

help: ## Show this help message
	@echo "🐉 DrogonClaw — Available Commands"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-18s\033[0m %s\n", $$1, $$2}'

# ─── Build ────────────────────────────────────────────────

build: ## Build DrogonClaw for current OS
	CGO_ENABLED=1 go build -ldflags="$(LDFLAGS)" -o $(BINARY_NAME) $(CMD_DIR)/

build-linux: ## Cross-compile for Linux amd64
	CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 $(CMD_DIR)/

build-all: build-linux ## Build for all supported platforms
	@echo "[+] All binaries built in $(BUILD_DIR)/"

# ─── Development ──────────────────────────────────────────

run: build ## Build and run DrogonClaw
	./$(BINARY_NAME)

daemon: build ## Build and run in daemon mode (Telegram only)
	./$(BINARY_NAME) daemon

setup: build ## Run the setup wizard
	./$(BINARY_NAME) setup

# ─── Testing ──────────────────────────────────────────────

test: ## Run all Go tests
	go test -v -race -count=1 ./internal/...

test-cover: ## Run tests with coverage report
	go test -v -race -coverprofile=coverage.out -covermode=atomic ./internal/...
	go tool cover -html=coverage.out -o coverage.html
	@echo "[+] Coverage report: coverage.html"

test-short: ## Run tests without long-running tests
	go test -v -short -race ./internal/...

# ─── Code Quality ─────────────────────────────────────────

lint: ## Run golangci-lint
	golangci-lint run ./...

format: ## Format all Go source files
	gofmt -s -w .
	goimports -w .

vet: ## Run go vet
	go vet ./...

# ─── Utilities ────────────────────────────────────────────

skills: ## Regenerate skill manifest
	node scripts/gen_skill_manifest.mjs

doctor: build ## Run system diagnostics (alias for the health command)
	./$(BINARY_NAME) health

deps: ## Download and tidy dependencies
	go mod download
	go mod tidy

clean: ## Remove build artifacts
	rm -f $(BINARY_NAME) $(BINARY_NAME).exe
	rm -rf $(BUILD_DIR)/ coverage.out coverage.html

# ─── Docker ───────────────────────────────────────────────

docker: ## Build Docker image
	docker build -t drogonclaw:$(VERSION) -t drogonclaw:latest .

docker-compose: ## Run with Docker Compose
	docker compose up -d

docker-down: ## Stop Docker Compose services
	docker compose down

.DEFAULT_GOAL := help
