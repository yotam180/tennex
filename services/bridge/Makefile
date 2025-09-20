# Tennex Bridge Service Makefile

# Build variables
APP_NAME := tennex-bridge
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "v0.1.0")
BUILD_TIME := $(shell date -u '+%Y-%m-%d %H:%M:%S UTC')
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# Go variables
GOOS := $(shell go env GOOS)
GOARCH := $(shell go env GOARCH)
BUILD_DIR := ./build
BINARY := $(BUILD_DIR)/$(APP_NAME)

# Linker flags
LDFLAGS := -ldflags "\
	-X 'main.Version=$(VERSION)' \
	-X 'main.BuildTime=$(BUILD_TIME)' \
	-X 'main.GitCommit=$(GIT_COMMIT)' \
	-s -w"

.PHONY: all build run dev clean test deps check fmt vet lint

# Default target
all: clean deps check build

# Build the binary
build:
	@echo "Building $(APP_NAME) $(VERSION)..."
	@mkdir -p $(BUILD_DIR)
	@go build $(LDFLAGS) -o $(BINARY) ./cmd/bridge
	@echo "Binary built: $(BINARY)"

# Run the service in development mode
run: build
	@echo "Starting $(APP_NAME) in development mode..."
	@TENNEX_ENV=development \
	 TENNEX_BRIDGE_LOG_LEVEL=debug \
	 TENNEX_BRIDGE_DEV_QR_IN_TERMINAL=true \
	 TENNEX_BRIDGE_DEV_ENABLE_PPROF=true \
	 $(BINARY)

# Development mode with live reload (requires air: go install github.com/cosmtrek/air@latest)
dev:
	@echo "Starting development server with live reload..."
	@air -c .air.toml || (echo "Air not found. Install with: go install github.com/cosmtrek/air@latest" && exit 1)

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf $(BUILD_DIR)
	@rm -rf ./session
	@go clean -cache

# Run tests
test:
	@echo "Running tests..."
	@go test -v ./...

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	@go test -v -race -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

# Install dependencies
deps:
	@echo "Installing dependencies..."
	@go mod tidy
	@go mod download

# Check for common issues
check: fmt vet lint

# Format code
fmt:
	@echo "Formatting code..."
	@go fmt ./...

# Run go vet
vet:
	@echo "Running go vet..."
	@go vet ./...

# Run golangci-lint (install with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest)
lint:
	@echo "Running golangci-lint..."
	@golangci-lint run || echo "golangci-lint not found. Install with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"

# Install development tools
dev-tools:
	@echo "Installing development tools..."
	@go install github.com/cosmtrek/air@latest
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Build for multiple platforms
build-all:
	@echo "Building for multiple platforms..."
	@mkdir -p $(BUILD_DIR)
	@for os in linux darwin windows; do \
		for arch in amd64 arm64; do \
			if [ "$$os" != "windows" ] || [ "$$arch" = "amd64" ]; then \
				echo "Building $$os/$$arch..."; \
				GOOS=$$os GOARCH=$$arch go build $(LDFLAGS) -o $(BUILD_DIR)/$(APP_NAME)-$$os-$$arch ./cmd/bridge; \
				if [ "$$os" = "windows" ]; then \
					mv $(BUILD_DIR)/$(APP_NAME)-$$os-$$arch $(BUILD_DIR)/$(APP_NAME)-$$os-$$arch.exe; \
				fi; \
			fi; \
		done; \
	done

# Docker build (for future use)
docker-build:
	@echo "Building Docker image..."
	@docker build -t $(APP_NAME):$(VERSION) -t $(APP_NAME):latest .

# Show help
help:
	@echo "Tennex Bridge Service - Available commands:"
	@echo ""
	@echo "  build        - Build the binary"
	@echo "  run          - Build and run in development mode"
	@echo "  dev          - Run with live reload (requires air)"
	@echo "  clean        - Clean build artifacts and session data"
	@echo "  test         - Run tests"
	@echo "  test-coverage- Run tests with coverage report"
	@echo "  deps         - Install/update dependencies"
	@echo "  check        - Run fmt, vet, and lint"
	@echo "  fmt          - Format code"
	@echo "  vet          - Run go vet"
	@echo "  lint         - Run golangci-lint"
	@echo "  dev-tools    - Install development tools"
	@echo "  build-all    - Build for multiple platforms"
	@echo "  docker-build - Build Docker image"
	@echo "  help         - Show this help"
	@echo ""
	@echo "Environment variables:"
	@echo "  TENNEX_BRIDGE_LOG_LEVEL - Set log level (debug, info, warn, error, fatal)"
	@echo "  TENNEX_BRIDGE_HTTP_PORT - Set HTTP port (default: 8080)"
	@echo "  TENNEX_ENV              - Set environment (development, staging, production)"
