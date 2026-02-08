# cadre - AI Agent Orchestration Framework
# Makefile for building, testing, and development

BINARY_NAME := cadre
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
LDFLAGS := -ldflags "-X main.version=$(VERSION) -X main.buildTime=$(BUILD_TIME) -X github.com/cadre-oss/cadre/internal/cli.Version=$(VERSION) -X github.com/cadre-oss/cadre/internal/cli.BuildTime=$(BUILD_TIME) -X github.com/cadre-oss/cadre/internal/cli.GitCommit=$(GIT_COMMIT)"

# Go parameters
GOCMD := go
GOBUILD := $(GOCMD) build
GOTEST := $(GOCMD) test
GOVET := $(GOCMD) vet
GOFMT := gofmt
GOLINT := golangci-lint

# Directories
CMD_DIR := ./cmd/cadre
BUILD_DIR := ./build
DIST_DIR := ./dist

.PHONY: all build clean test test-coverage lint fmt vet install run help web-install web-build web-dev serve

# Default target
all: lint test build

# Build the binary
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) $(CMD_DIR)
	@echo "Built $(BUILD_DIR)/$(BINARY_NAME)"

# Build for multiple platforms
build-all: build-linux build-darwin build-windows

build-linux:
	@echo "Building for Linux..."
	GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-linux-amd64 $(CMD_DIR)
	GOOS=linux GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-linux-arm64 $(CMD_DIR)

build-darwin:
	@echo "Building for macOS..."
	GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-darwin-amd64 $(CMD_DIR)
	GOOS=darwin GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-darwin-arm64 $(CMD_DIR)

build-windows:
	@echo "Building for Windows..."
	GOOS=windows GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-windows-amd64.exe $(CMD_DIR)

# Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -rf $(BUILD_DIR) $(DIST_DIR)
	@go clean -cache -testcache

# Run tests
test:
	@echo "Running tests..."
	$(GOTEST) -v -race ./...

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	$(GOTEST) -v -race -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

# Run integration tests
test-integration:
	@echo "Running integration tests..."
	$(GOTEST) -v -race -tags=integration ./test/integration/...

# Run end-to-end tests
test-e2e:
	@echo "Running e2e tests..."
	$(GOTEST) -v -tags=e2e ./test/e2e/...

# Lint code
lint:
	@echo "Linting..."
	@if command -v $(GOLINT) > /dev/null; then \
		$(GOLINT) run ./...; \
	else \
		echo "golangci-lint not installed, running go vet only"; \
		$(GOVET) ./...; \
	fi

# Format code
fmt:
	@echo "Formatting..."
	$(GOFMT) -s -w .

# Check formatting
fmt-check:
	@echo "Checking formatting..."
	@test -z "$$($(GOFMT) -l .)" || (echo "Code not formatted. Run 'make fmt'" && exit 1)

# Run go vet
vet:
	@echo "Running go vet..."
	$(GOVET) ./...

# Install binary to GOPATH/bin
install: build
	@echo "Installing $(BINARY_NAME)..."
	@cp $(BUILD_DIR)/$(BINARY_NAME) $(GOPATH)/bin/

# Install to /usr/local/bin (requires sudo)
install-global: build
	@echo "Installing $(BINARY_NAME) to /usr/local/bin..."
	@sudo cp $(BUILD_DIR)/$(BINARY_NAME) /usr/local/bin/

# Run the binary
run: build
	$(BUILD_DIR)/$(BINARY_NAME)

# Generate mocks (if needed)
mocks:
	@echo "Generating mocks..."
	@go generate ./...

# Update dependencies
deps:
	@echo "Updating dependencies..."
	$(GOCMD) mod tidy
	$(GOCMD) mod verify

# Web UI - install frontend dependencies
web-install:
	@echo "Installing frontend dependencies..."
	cd web && npm install

# Web UI - build frontend (outputs to internal/server/dist/)
web-build: web-install
	@echo "Building frontend..."
	cd web && npm run build

# Web UI - start Vite dev server
web-dev:
	cd web && npm run dev

# Start cadre web UI server
serve: build
	$(BUILD_DIR)/$(BINARY_NAME) serve

# Show help
help:
	@echo "cadre - AI Agent Orchestration Framework"
	@echo ""
	@echo "Usage:"
	@echo "  make [target]"
	@echo ""
	@echo "Targets:"
	@echo "  all            Run lint, test, and build (default)"
	@echo "  build          Build the binary"
	@echo "  build-all      Build for all platforms"
	@echo "  clean          Clean build artifacts"
	@echo "  test           Run tests"
	@echo "  test-coverage  Run tests with coverage report"
	@echo "  test-integration Run integration tests"
	@echo "  test-e2e       Run end-to-end tests"
	@echo "  lint           Run linter"
	@echo "  fmt            Format code"
	@echo "  fmt-check      Check code formatting"
	@echo "  vet            Run go vet"
	@echo "  install        Install to GOPATH/bin"
	@echo "  install-global Install to /usr/local/bin (sudo)"
	@echo "  run            Build and run"
	@echo "  deps           Update dependencies"
	@echo "  web-install    Install frontend dependencies"
	@echo "  web-build      Build frontend for embedding"
	@echo "  web-dev        Start Vite dev server"
	@echo "  serve          Build and start web UI"
	@echo "  help           Show this help"
