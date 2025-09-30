# reqo Makefile

# Variables
BINARY_NAME=reqo
BUILD_DIR=build
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS=-ldflags "-X main.version=$(VERSION) -X main.buildTime=$(shell date -u +%Y-%m-%dT%H:%M:%SZ)"
GO_FILES=$(shell find . -name "*.go" -type f)
GOOS=$(shell go env GOOS)
GOARCH=$(shell go env GOARCH)
GOPATH=$(shell go env GOPATH)
GOBIN=$(shell go env GOBIN)

# Default target
.PHONY: all
all: clean build

# Build the binary
.PHONY: build
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	@go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) cmd/reqo/main.go
	@echo "Built $(BUILD_DIR)/$(BINARY_NAME)"

# Build for current platform (development)
.PHONY: dev
dev:
	@echo "Building $(BINARY_NAME) for development..."
	@go build -o $(BINARY_NAME) cmd/reqo/main.go
	@echo "Built $(BINARY_NAME)"

# Cross-compilation targets
.PHONY: build-linux
build-linux:
	@echo "Building $(BINARY_NAME) for Linux..."
	@mkdir -p $(BUILD_DIR)
	@GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 cmd/reqo/main.go
	@echo "Built $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64"

.PHONY: build-darwin
build-darwin:
	@echo "Building $(BINARY_NAME) for macOS..."
	@mkdir -p $(BUILD_DIR)
	@GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 cmd/reqo/main.go
	@GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 cmd/reqo/main.go
	@echo "Built $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 and $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64"

.PHONY: build-windows
build-windows:
	@echo "Building $(BINARY_NAME) for Windows..."
	@mkdir -p $(BUILD_DIR)
	@GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe cmd/reqo/main.go
	@echo "Built $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe"

# Build all platforms
.PHONY: build-all
build-all: build-linux build-darwin build-windows
	@echo "Built binaries for all platforms"

# Install to GOBIN or GOPATH/bin
.PHONY: install
install: build
	@if [ -n "$(GOBIN)" ]; then \
		INSTALL_DIR="$(GOBIN)"; \
		echo "Installing $(BINARY_NAME) to $(GOBIN)..."; \
	else \
		INSTALL_DIR="$(GOPATH)/bin"; \
		echo "Installing $(BINARY_NAME) to $(GOPATH)/bin..."; \
	fi; \
	mkdir -p $$INSTALL_DIR; \
	cp $(BUILD_DIR)/$(BINARY_NAME) $$INSTALL_DIR/; \
	echo "Installed $(BINARY_NAME)"

# Install to system (requires sudo)
.PHONY: install-system
install-system: build
	@echo "Installing $(BINARY_NAME) to /usr/local/bin..."
	@sudo cp $(BUILD_DIR)/$(BINARY_NAME) /usr/local/bin/
	@echo "Installed $(BINARY_NAME) system-wide"

# Run tests
.PHONY: test
test:
	@echo "Running tests..."
	@go test -v ./...

# Run tests with coverage
.PHONY: test-coverage
test-coverage:
	@echo "Running tests with coverage..."
	@go test -v -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Run benchmarks
.PHONY: bench
bench:
	@echo "Running benchmarks..."
	@go test -bench=. ./...

# Lint code
.PHONY: lint
lint:
	@echo "Running linter..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not found. Install with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
		go vet ./...; \
	fi

# Format code
.PHONY: fmt
fmt:
	@echo "Formatting code..."
	@go fmt ./...
	@if command -v goimports >/dev/null 2>&1; then \
		goimports -w .; \
	else \
		echo "goimports not found. Install with: go install golang.org/x/tools/cmd/goimports@latest"; \
	fi

# Tidy dependencies
.PHONY: tidy
tidy:
	@echo "Tidying dependencies..."
	@go mod tidy
	@go mod verify

# Clean build artifacts
.PHONY: clean
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf $(BUILD_DIR)
	@rm -f $(BINARY_NAME)
	@rm -f coverage.out coverage.html
	@echo "Cleaned"

# Run the application (for testing)
.PHONY: run
run: dev
	@echo "Running $(BINARY_NAME)..."
	@./$(BINARY_NAME)

# Show help
.PHONY: help
help:
	@echo "Available targets:"
	@echo "  build          - Build binary for current platform"
	@echo "  dev            - Build binary for development (current dir)"
	@echo "  build-linux    - Build for Linux (amd64)"
	@echo "  build-darwin   - Build for macOS (amd64, arm64)"
	@echo "  build-windows  - Build for Windows (amd64)"
	@echo "  build-all      - Build for all platforms"
	@echo "  install        - Install to GOBIN or GOPATH/bin"
	@echo "  install-system - Install to /usr/local/bin (requires sudo)"
	@echo "  test           - Run tests"
	@echo "  test-coverage  - Run tests with coverage report"
	@echo "  bench          - Run benchmarks"
	@echo "  lint           - Run linter"
	@echo "  fmt            - Format code"
	@echo "  tidy           - Tidy dependencies"
	@echo "  clean          - Clean build artifacts"
	@echo "  run            - Build and run the application"
	@echo "  help           - Show this help message"
	@echo ""
	@echo "Variables:"
	@echo "  BINARY_NAME    - Name of the binary (default: reqo)"
	@echo "  BUILD_DIR      - Build directory (default: build)"
	@echo "  VERSION        - Version from git tags (default: dev)"

# Development workflow
.PHONY: dev-setup
dev-setup: tidy fmt lint
	@echo "Development setup complete"

# Release preparation
.PHONY: release-prep
release-prep: clean test lint build-all
	@echo "Release preparation complete"
	@echo "Built binaries:"
	@ls -la $(BUILD_DIR)/

# Check if required tools are installed
.PHONY: check-tools
check-tools:
	@echo "Checking required tools..."
	@command -v go >/dev/null 2>&1 || (echo "Go is not installed" && exit 1)
	@command -v git >/dev/null 2>&1 || (echo "Git is not installed" && exit 1)
	@echo "All required tools are installed"