# picoHWMon Makefile

.PHONY: all build clean test fmt vet run linux windows deps help

# Variables
APP_NAME := picoHWMon
MAIN_FILE := main.go
BUILD_DIR := build
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-X main.version=$(VERSION) -s -w"

# Default target
all: clean deps fmt vet test build

# Install dependencies
deps:
	@echo "Installing dependencies..."
	go mod download
	go mod tidy

# Format code
fmt:
	@echo "Formatting code..."
	go fmt ./...

# Vet code
vet:
	@echo "Vetting code..."
	go vet ./...

# Run tests
test:
	@echo "Running tests..."
	go test -v -race ./...

# Build for current platform
build:
	@echo "Building for current platform..."
	@mkdir -p $(BUILD_DIR)
	go build $(LDFLAGS) -o $(BUILD_DIR)/$(APP_NAME) $(MAIN_FILE)

# Build for Linux
linux:
	@echo "Building for Linux..."
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(APP_NAME)-linux-amd64 $(MAIN_FILE)

# Build for Windows
windows:
	@echo "Building for Windows..."
	@mkdir -p $(BUILD_DIR)
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(APP_NAME)-windows-amd64.exe $(MAIN_FILE)

# Build for both platforms
cross: linux windows

# Run the application
run:
	@echo "Running application..."
	go run $(MAIN_FILE) --port 8080

# Run with race detection
run-race:
	@echo "Running with race detection..."
	go run -race $(MAIN_FILE) --port 8080

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -rf $(BUILD_DIR)
	go clean

# Development build with debug info
dev:
	@echo "Building development version..."
	@mkdir -p $(BUILD_DIR)
	go build -gcflags="all=-N -l" -o $(BUILD_DIR)/$(APP_NAME)-dev $(MAIN_FILE)

# Install tools
tools:
	@echo "Installing development tools..."
	go install golang.org/x/tools/cmd/goimports@latest
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Lint code
lint:
	@echo "Running linter..."
	golangci-lint run

# Docker build
docker:
	@echo "Building Docker image..."
	docker build -t $(APP_NAME):$(VERSION) .

# Show help
help:
	@echo "Available targets:"
	@echo "  all      - Run clean, deps, fmt, vet, test, and build"
	@echo "  deps     - Install Go dependencies"
	@echo "  fmt      - Format Go code"
	@echo "  vet      - Run Go vet"
	@echo "  test     - Run tests with race detection"
	@echo "  build    - Build for current platform"
	@echo "  linux    - Build for Linux"
	@echo "  windows  - Build for Windows"
	@echo "  cross    - Build for Linux and Windows"
	@echo "  run      - Run the application"
	@echo "  run-race - Run with race detection"
	@echo "  clean    - Clean build artifacts"
	@echo "  dev      - Build development version with debug info"
	@echo "  tools    - Install development tools"
	@echo "  lint     - Run golangci-lint"
	@echo "  docker   - Build Docker image"
	@echo "  help     - Show this help message"
