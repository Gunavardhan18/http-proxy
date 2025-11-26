# HTTP Proxy Server Makefile

# Build configuration
PROXY_BINARY=proxy
BACKEND_BINARY=backend  
TRAFFIC_GEN_BINARY=traffic-gen
CONFIG_GEN_BINARY=config-gen

# Go build flags
GO_BUILD_FLAGS=-ldflags="-w -s"
GO_TEST_FLAGS=-v -race -coverprofile=coverage.out

.PHONY: all build clean test coverage run-demo help

# Default target
all: build

# Build all binaries
build: build-proxy build-backend build-traffic-gen build-config-gen
	@echo "All binaries built successfully!"

# Build proxy server
build-proxy:
	@echo "Building proxy server..."
	go build $(GO_BUILD_FLAGS) -o $(PROXY_BINARY) cmd/proxy/main.go

# Build backend server
build-backend:
	@echo "Building backend server..."
	go build $(GO_BUILD_FLAGS) -o $(BACKEND_BINARY) cmd/backend/main.go

# Build traffic generator
build-traffic-gen:
	@echo "Building traffic generator..."
	go build $(GO_BUILD_FLAGS) -o $(TRAFFIC_GEN_BINARY) cmd/traffic-gen/main.go

# Build config generator
build-config-gen:
	@echo "Building config generator..."
	go build $(GO_BUILD_FLAGS) -o $(CONFIG_GEN_BINARY) cmd/config-gen/main.go

# Generate configuration files
config: build-config-gen
	@echo "Generating configuration files..."
	./$(CONFIG_GEN_BINARY)

# Run tests
test:
	@echo "Running tests..."
	go test $(GO_TEST_FLAGS) ./...

# Generate test coverage report
coverage: test
	@echo "Generating coverage report..."
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Run linter
lint:
	@echo "Running linter..."
	golangci-lint run ./...

# Format code
fmt:
	@echo "Formatting code..."
	go fmt ./...

# Download dependencies
deps:
	@echo "Downloading dependencies..."
	go mod download
	go mod tidy

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -f $(PROXY_BINARY) $(BACKEND_BINARY) $(TRAFFIC_GEN_BINARY) $(CONFIG_GEN_BINARY)
	rm -f coverage.out coverage.html
	rm -rf logs/

# Install binaries to GOPATH/bin
install: build
	@echo "Installing binaries..."
	cp $(PROXY_BINARY) $(GOPATH)/bin/
	cp $(BACKEND_BINARY) $(GOPATH)/bin/
	cp $(TRAFFIC_GEN_BINARY) $(GOPATH)/bin/

# Run demo (requires PowerShell on Windows)
run-demo: build config
	@echo "Running demo..."
	powershell -ExecutionPolicy Bypass -File demo.ps1

# Quick start - build, config, and run proxy with default settings
quick-start: build config
	@echo "Starting proxy server with default configuration..."
	./$(PROXY_BINARY) -config examples/proxy.yaml

# Development server - run with debug logging
dev: build config
	@echo "Starting development server with debug logging..."
	./$(PROXY_BINARY) -config examples/proxy.yaml -log-level debug

# Load test - run traffic generator against proxy
load-test: build
	@echo "Running load test..."
	./$(TRAFFIC_GEN_BINARY) -proxy http://localhost:8080 -c 20 -d 60s -rps 50

# Docker build
docker-build:
	@echo "Building Docker image..."
	docker build -t http-proxy:latest .

# Docker run
docker-run: docker-build
	@echo "Running Docker container..."
	docker run -p 8080:8080 -p 8090:8090 http-proxy:latest

# Help target
help:
	@echo "Available targets:"
	@echo "  build          - Build all binaries"
	@echo "  build-proxy    - Build only proxy server"
	@echo "  build-backend  - Build only backend server"
	@echo "  build-traffic-gen - Build only traffic generator"
	@echo "  config         - Generate configuration files"
	@echo "  test           - Run tests"
	@echo "  coverage       - Generate test coverage report"
	@echo "  lint           - Run linter"
	@echo "  fmt            - Format code"
	@echo "  deps           - Download and tidy dependencies"
	@echo "  clean          - Clean build artifacts"
	@echo "  install        - Install binaries to GOPATH/bin"
	@echo "  run-demo       - Run interactive demo"
	@echo "  quick-start    - Build and start proxy with defaults"
	@echo "  dev            - Start development server with debug logging"
	@echo "  load-test      - Run load test against proxy"
	@echo "  docker-build   - Build Docker image"
	@echo "  docker-run     - Run Docker container"
	@echo "  help           - Show this help message"
