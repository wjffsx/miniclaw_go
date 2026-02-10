.PHONY: all build clean test run install deps fmt lint vet help

BINARY_NAME=miniclaw_go
BUILD_DIR=bin
CMD_DIR=cmd
GO=go
GOFLAGS=-v

all: clean deps fmt vet test build

deps:
	$(GO) mod download
	$(GO) mod tidy

build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	$(GO) build $(GOFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) $(CMD_DIR)/main.go
	@echo "Build complete: $(BUILD_DIR)/$(BINARY_NAME)"

clean:
	@echo "Cleaning..."
	@rm -rf $(BUILD_DIR)
	@rm -rf data/*
	@echo "Clean complete"

test:
	@echo "Running tests..."
	$(GO) test -v -race ./...

run: build
	@echo "Running $(BINARY_NAME)..."
	./$(BUILD_DIR)/$(BINARY_NAME)

install:
	@echo "Installing $(BINARY_NAME)..."
	$(GO) install $(CMD_DIR)/main.go

fmt:
	@echo "Formatting code..."
	$(GO) fmt ./...

vet:
	@echo "Vetting code..."
	$(GO) vet ./...

lint:
	@echo "Linting code..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not installed. Install with: curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b \$$(go env GOPATH)/bin"; \
	fi

docker-build:
	@echo "Building Docker image..."
	docker build -t miniclaw_go:latest .

docker-run:
	@echo "Running Docker container..."
	docker run -p 18789:18789 -v $(PWD)/data:/app/data -v $(PWD)/configs:/app/configs miniclaw_go:latest

help:
	@echo "Available targets:"
	@echo "  all       - Run clean, deps, fmt, vet, test, and build"
	@echo "  build     - Build the binary"
	@echo "  clean     - Remove build artifacts"
	@echo "  test      - Run tests"
	@echo "  run       - Build and run the binary"
	@echo "  install   - Install the binary"
	@echo "  deps      - Download dependencies"
	@echo "  fmt       - Format code"
	@echo "  vet       - Run go vet"
	@echo "  lint      - Run golangci-lint"
	@echo "  docker-build - Build Docker image"
	@echo "  docker-run  - Run Docker container"