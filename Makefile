# MCP Manager - Unified Build System
# Treats the entire project as a single application regardless of language

.PHONY: all dev build clean install test lint help

# Variables
SHELL := /bin/bash
ROOT_DIR := $(shell pwd)
BUILD_DIR := $(ROOT_DIR)/build
DIST_DIR := $(ROOT_DIR)/dist
GO_BIN := $(ROOT_DIR)/services/manager/bin/mcp-manager
FRONTEND_DIST := $(ROOT_DIR)/apps/desktop/dist

# Platform detection
UNAME := $(shell uname -s)
ARCH := $(shell uname -m)

ifeq ($(UNAME),Darwin)
	PLATFORM := mac
	GO_OS := darwin
else ifeq ($(UNAME),Linux)
	PLATFORM := linux
	GO_OS := linux
else
	PLATFORM := win
	GO_OS := windows
	GO_BIN := $(ROOT_DIR)/services/manager/bin/mcp-manager.exe
endif

ifeq ($(ARCH),x86_64)
	GO_ARCH := amd64
else ifeq ($(ARCH),arm64)
	GO_ARCH := arm64
else
	GO_ARCH := $(ARCH)
endif

# Colors for output
RED := \033[0;31m
GREEN := \033[0;32m
YELLOW := \033[0;33m
BLUE := \033[0;34m
CYAN := \033[0;36m
NC := \033[0m # No Color

# Default target
all: build

# Help target
help:
	@echo "$(CYAN)╔══════════════════════════════════════════════════════╗$(NC)"
	@echo "$(CYAN)║          MCP Manager - Unified Build System         ║$(NC)"
	@echo "$(CYAN)╚══════════════════════════════════════════════════════╝$(NC)"
	@echo ""
	@echo "$(GREEN)Available targets:$(NC)"
	@echo "  $(YELLOW)make dev$(NC)          - Start development mode (all services)"
	@echo "  $(YELLOW)make dev-electron$(NC) - Start development with Electron"
	@echo "  $(YELLOW)make build$(NC)        - Build entire application"
	@echo "  $(YELLOW)make build-backend$(NC) - Build Go backend only"
	@echo "  $(YELLOW)make build-frontend$(NC)- Build TypeScript frontend only"
	@echo "  $(YELLOW)make package$(NC)      - Build and package for distribution"
	@echo "  $(YELLOW)make package-mac$(NC)  - Package for macOS"
	@echo "  $(YELLOW)make package-win$(NC)  - Package for Windows"
	@echo "  $(YELLOW)make package-linux$(NC)- Package for Linux"
	@echo "  $(YELLOW)make install$(NC)      - Install dependencies"
	@echo "  $(YELLOW)make test$(NC)         - Run all tests"
	@echo "  $(YELLOW)make lint$(NC)         - Run linters"
	@echo "  $(YELLOW)make clean$(NC)        - Clean build artifacts"
	@echo "  $(YELLOW)make run$(NC)          - Run built application"
	@echo "  $(YELLOW)make help$(NC)         - Show this help message"
	@echo ""
	@echo "$(BLUE)Platform detected: $(PLATFORM) ($(GO_OS)/$(GO_ARCH))$(NC)"

# Install dependencies
install:
	@echo "$(CYAN)[Installing dependencies]$(NC)"
	@echo "$(YELLOW)→ Installing Node.js dependencies...$(NC)"
	@npm install
	@echo "$(YELLOW)→ Installing Go dependencies...$(NC)"
	@cd services/manager && go mod download
	@echo "$(GREEN)✓ Dependencies installed$(NC)"

# Development mode
dev:
	@echo "$(CYAN)[Starting development mode]$(NC)"
	@node scripts/dev-unified.mjs

dev-electron:
	@echo "$(CYAN)[Starting development mode with Electron]$(NC)"
	@node scripts/dev-unified.mjs --electron

# Build targets
build: clean build-backend build-frontend
	@echo "$(GREEN)╔══════════════════════════════════════════════════════╗$(NC)"
	@echo "$(GREEN)║           Build completed successfully!              ║$(NC)"
	@echo "$(GREEN)╚══════════════════════════════════════════════════════╝$(NC)"

build-backend:
	@echo "$(CYAN)[Building Go backend]$(NC)"
	@mkdir -p services/manager/bin
	@cd services/manager && \
		CGO_ENABLED=0 GOOS=$(GO_OS) GOARCH=$(GO_ARCH) \
		go build -ldflags="-s -w" -o bin/mcp-manager ./cmd/manager
	@echo "$(GREEN)✓ Backend built: $(GO_BIN)$(NC)"

build-frontend:
	@echo "$(CYAN)[Building TypeScript frontend]$(NC)"
	@cd apps/desktop && npm run build
	@echo "$(GREEN)✓ Frontend built$(NC)"

# Quick build (no clean)
quick-build: build-backend build-frontend
	@echo "$(GREEN)✓ Quick build completed$(NC)"

# Package for distribution
package: build
	@echo "$(CYAN)[Packaging application]$(NC)"
	@node scripts/build.mjs --electron --$(PLATFORM)

package-mac:
	@echo "$(CYAN)[Packaging for macOS]$(NC)"
	@node scripts/build.mjs --electron --mac

package-win:
	@echo "$(CYAN)[Packaging for Windows]$(NC)"
	@node scripts/build.mjs --electron --win

package-linux:
	@echo "$(CYAN)[Packaging for Linux]$(NC)"
	@node scripts/build.mjs --electron --linux

package-all: package-mac package-win package-linux
	@echo "$(GREEN)✓ All platform packages created$(NC)"

# Run the built application
run: build
	@echo "$(CYAN)[Running application]$(NC)"
	@if [ -f "$(GO_BIN)" ]; then \
		$(GO_BIN) & \
		PID=$$!; \
		sleep 2; \
		open http://localhost:5173 || xdg-open http://localhost:5173 || start http://localhost:5173; \
		wait $$PID; \
	else \
		echo "$(RED)✗ Binary not found. Run 'make build' first$(NC)"; \
		exit 1; \
	fi

# Testing
test:
	@echo "$(CYAN)[Running tests]$(NC)"
	@echo "$(YELLOW)→ Testing Go backend...$(NC)"
	@cd services/manager && go test -v ./...
	@echo "$(YELLOW)→ Testing TypeScript frontend...$(NC)"
	@cd apps/desktop && npm test
	@echo "$(GREEN)✓ All tests passed$(NC)"

test-backend:
	@cd services/manager && go test -v ./...

test-frontend:
	@cd apps/desktop && npm test

# Linting
lint:
	@echo "$(CYAN)[Running linters]$(NC)"
	@echo "$(YELLOW)→ Linting Go code...$(NC)"
	@cd services/manager && go vet ./...
	@echo "$(YELLOW)→ Linting TypeScript code...$(NC)"
	@npx biome check .
	@echo "$(GREEN)✓ Linting completed$(NC)"

format:
	@echo "$(CYAN)[Formatting code]$(NC)"
	@echo "$(YELLOW)→ Formatting Go code...$(NC)"
	@cd services/manager && go fmt ./...
	@echo "$(YELLOW)→ Formatting TypeScript code...$(NC)"
	@npx biome format --write .
	@echo "$(GREEN)✓ Formatting completed$(NC)"

# Clean build artifacts
clean:
	@echo "$(CYAN)[Cleaning build artifacts]$(NC)"
	@rm -rf $(BUILD_DIR)
	@rm -rf $(DIST_DIR)
	@rm -rf services/manager/bin
	@rm -rf apps/desktop/dist
	@rm -rf apps/desktop/out
	@echo "$(GREEN)✓ Clean completed$(NC)"

# Deep clean (including node_modules)
deep-clean: clean
	@echo "$(YELLOW)→ Removing node_modules...$(NC)"
	@rm -rf node_modules
	@rm -rf apps/*/node_modules
	@rm -rf packages/*/node_modules
	@echo "$(YELLOW)→ Removing Go cache...$(NC)"
	@go clean -cache
	@echo "$(GREEN)✓ Deep clean completed$(NC)"

# Check prerequisites
check-deps:
	@echo "$(CYAN)[Checking prerequisites]$(NC)"
	@command -v node >/dev/null 2>&1 || { echo "$(RED)✗ Node.js is required but not installed$(NC)"; exit 1; }
	@command -v npm >/dev/null 2>&1 || { echo "$(RED)✗ npm is required but not installed$(NC)"; exit 1; }
	@command -v go >/dev/null 2>&1 || { echo "$(RED)✗ Go is required but not installed$(NC)"; exit 1; }
	@echo "$(GREEN)✓ All prerequisites installed$(NC)"

# Watch mode for development
watch:
	@echo "$(CYAN)[Starting watch mode]$(NC)"
	@make -j2 watch-backend watch-frontend

watch-backend:
	@cd services/manager && go run ./cmd/manager

watch-frontend:
	@cd apps/desktop && npm run dev

# Docker targets (if needed in future)
docker-build:
	@echo "$(CYAN)[Building Docker image]$(NC)"
	@docker build -t mcp-manager:latest .

docker-run:
	@docker run -p 38018:38018 -p 5173:5173 mcp-manager:latest

# Version information
version:
	@echo "$(CYAN)MCP Manager Version Information$(NC)"
	@echo "Node.js: $$(node --version)"
	@echo "npm: $$(npm --version)"
	@echo "Go: $$(go version)"
	@echo "Platform: $(PLATFORM) ($(GO_OS)/$(GO_ARCH))"

# Performance profiling
profile-backend:
	@cd services/manager && go test -cpuprofile cpu.prof -memprofile mem.prof -bench .

# Security scanning
security-scan:
	@echo "$(CYAN)[Running security scan]$(NC)"
	@npm audit
	@cd services/manager && go list -json -deps ./... | nancy sleuth

.DEFAULT_GOAL := help