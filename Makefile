.PHONY: all build clean test run install help lint fmt format gofmt kb deps

# Binary names and directories
KB_BUILDER_BINARY=pgedge-ai-kb-builder
BIN_DIR=bin
KB_BUILDER_CMD_DIR=cmd/kb-builder

# Build variables
GO=go
GOFLAGS=-v

# Default target - build the binary
all: build

# Build the kb-builder binary
build: kb-builder
kb-builder:
	@echo "Building $(KB_BUILDER_BINARY)..."
	@mkdir -p $(BIN_DIR)
	$(GO) build $(GOFLAGS) -o $(BIN_DIR)/$(KB_BUILDER_BINARY) ./$(KB_BUILDER_CMD_DIR)
	@echo "KB-builder build complete: $(BIN_DIR)/$(KB_BUILDER_BINARY)"

# Build/update the knowledgebase database
kb: kb-builder
	@echo "Building knowledgebase database..."
	$(BIN_DIR)/$(KB_BUILDER_BINARY) -c examples/pgedge-ai-kb-builder.yaml
	@echo "Knowledgebase build complete: $(BIN_DIR)/pgedge-ai-kb.db"

# Build for multiple platforms
build-all: build-linux build-darwin build-windows

build-linux:
	@echo "Building $(KB_BUILDER_BINARY) for Linux..."
	@mkdir -p $(BIN_DIR)
	GOOS=linux GOARCH=amd64 $(GO) build $(GOFLAGS) -o $(BIN_DIR)/$(KB_BUILDER_BINARY)-linux-amd64 ./$(KB_BUILDER_CMD_DIR)
	GOOS=linux GOARCH=arm64 $(GO) build $(GOFLAGS) -o $(BIN_DIR)/$(KB_BUILDER_BINARY)-linux-arm64 ./$(KB_BUILDER_CMD_DIR)
	@echo "Linux builds complete"

build-darwin:
	@echo "Building $(KB_BUILDER_BINARY) for macOS..."
	@mkdir -p $(BIN_DIR)
	GOOS=darwin GOARCH=amd64 $(GO) build $(GOFLAGS) -o $(BIN_DIR)/$(KB_BUILDER_BINARY)-darwin-amd64 ./$(KB_BUILDER_CMD_DIR)
	GOOS=darwin GOARCH=arm64 $(GO) build $(GOFLAGS) -o $(BIN_DIR)/$(KB_BUILDER_BINARY)-darwin-arm64 ./$(KB_BUILDER_CMD_DIR)
	@echo "macOS builds complete"

build-windows:
	@echo "Building $(KB_BUILDER_BINARY) for Windows..."
	@mkdir -p $(BIN_DIR)
	GOOS=windows GOARCH=amd64 $(GO) build $(GOFLAGS) -o $(BIN_DIR)/$(KB_BUILDER_BINARY)-windows-amd64.exe ./$(KB_BUILDER_CMD_DIR)
	@echo "Windows build complete"

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -f $(BIN_DIR)/$(KB_BUILDER_BINARY)
	rm -f $(BIN_DIR)/$(KB_BUILDER_BINARY)-linux-*
	rm -f $(BIN_DIR)/$(KB_BUILDER_BINARY)-darwin-*
	rm -f $(BIN_DIR)/$(KB_BUILDER_BINARY)-windows-*
	@echo "Clean complete"

# Run all tests
test:
	@echo "Running tests..."
	$(GO) test -v ./internal/... ./$(KB_BUILDER_CMD_DIR)/...

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	$(GO) test -v -race -coverprofile=coverage.out -covermode=atomic ./internal/... ./$(KB_BUILDER_CMD_DIR)/...
	$(GO) tool cover -func=coverage.out

# Install dependencies
deps:
	@echo "Installing dependencies..."
	$(GO) mod download
	$(GO) mod tidy
	@echo "Dependencies installed"

# Install binary to GOPATH/bin
install: build
	@echo "Installing $(KB_BUILDER_BINARY) to $$(go env GOPATH)/bin..."
	$(GO) install ./$(KB_BUILDER_CMD_DIR)
	@echo "Install complete: $(KB_BUILDER_BINARY)"

# Format code
fmt:
	@echo "Formatting code..."
	$(GO) fmt ./...
	@echo "Format complete"

# Alias for fmt
format: fmt

# Run gofmt directly
gofmt:
	@echo "Running gofmt..."
	@find . -name '*.go' -not -path './bin/*' -exec gofmt -l -w {} +
	@echo "gofmt complete"

# Run linter (requires golangci-lint)
lint:
	@echo "Running linter..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run ./cmd/... ./internal/...; \
	elif [ -f "$$(go env GOPATH)/bin/golangci-lint" ]; then \
		$$(go env GOPATH)/bin/golangci-lint run ./cmd/... ./internal/...; \
	else \
		echo "golangci-lint not found. Install it with:"; \
		echo "  go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest"; \
		echo "  or visit https://golangci-lint.run/usage/install/"; \
	fi

# Build the documentation site (requires mkdocs)
docs:
	@echo "Building documentation site..."
	@if [ ! -d venv ]; then \
		python3 -m venv venv && venv/bin/pip install -r requirements.txt; \
	fi
	venv/bin/mkdocs build -v
	@echo "Documentation built in site/"

# Show help
help:
	@echo "pgEdge AI Knowledgebase Builder - Makefile commands:"
	@echo ""
	@echo "Building:"
	@echo "  make                - Build the kb-builder binary (default)"
	@echo "  make build          - Build the kb-builder binary"
	@echo "  make kb-builder     - Build the kb-builder binary (alias)"
	@echo "  make build-all      - Build for all platforms"
	@echo "  make build-linux    - Build for Linux (amd64 and arm64)"
	@echo "  make build-darwin   - Build for macOS (amd64 and arm64)"
	@echo "  make build-windows  - Build for Windows (amd64)"
	@echo ""
	@echo "Testing:"
	@echo "  make test           - Run all tests"
	@echo "  make test-coverage  - Run tests with coverage report"
	@echo ""
	@echo "Formatting:"
	@echo "  make fmt            - Format Go code with go fmt"
	@echo "  make format         - Format Go code with go fmt (alias)"
	@echo "  make gofmt          - Format Go code with gofmt directly"
	@echo ""
	@echo "Linting:"
	@echo "  make lint           - Run linter on all code"
	@echo ""
	@echo "Cleaning:"
	@echo "  make clean          - Remove build artifacts"
	@echo ""
	@echo "Knowledgebase:"
	@echo "  make kb             - Build the knowledgebase database in bin/"
	@echo ""
	@echo "Documentation:"
	@echo "  make docs           - Build the documentation site with MkDocs"
	@echo ""
	@echo "Other:"
	@echo "  make deps           - Install/update dependencies"
	@echo "  make install        - Install kb-builder to GOPATH/bin"
	@echo "  make help           - Show this help message"
