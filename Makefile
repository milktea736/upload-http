# upload-http Makefile

.PHONY: all build clean test run-server run-client help install

# Variables
BINARY_DIR := bin
SERVER_BINARY := $(BINARY_DIR)/server
CLIENT_BINARY := $(BINARY_DIR)/client
GO_VERSION := $(shell go version | cut -d' ' -f3)

# Default target
all: build

# Build both server and client binaries
build: $(SERVER_BINARY) $(CLIENT_BINARY)

# Build server binary
$(SERVER_BINARY): cmd/server/main.go pkg/server/server.go pkg/config/config.go
	@echo "Building server..."
	@mkdir -p $(BINARY_DIR)
	go build -o $(SERVER_BINARY) ./cmd/server

# Build client binary
$(CLIENT_BINARY): cmd/client/main.go pkg/client/client.go pkg/config/config.go
	@echo "Building client..."
	@mkdir -p $(BINARY_DIR)
	go build -o $(CLIENT_BINARY) ./cmd/client

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -rf $(BINARY_DIR)
	rm -rf uploads/
	rm -rf storage/

# Run tests
test:
	@echo "Running tests..."
	go test -v ./...

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Format code
fmt:
	@echo "Formatting code..."
	go fmt ./...

# Lint code
lint:
	@echo "Linting code..."
	golangci-lint run

# Install dependencies
deps:
	@echo "Installing dependencies..."
	go mod download
	go mod tidy

# Run server with default settings
run-server: $(SERVER_BINARY)
	@echo "Starting server..."
	./$(SERVER_BINARY) --log-level debug

# Run server with config file
run-server-config: $(SERVER_BINARY)
	@echo "Starting server with config..."
	./$(SERVER_BINARY) --config configs/server.json

# Generate server config
generate-server-config: $(SERVER_BINARY)
	@echo "Generating server configuration..."
	./$(SERVER_BINARY) --generate-config server-config.json

# Generate client config
generate-client-config: $(CLIENT_BINARY)
	@echo "Generating client configuration..."
	./$(CLIENT_BINARY) config --generate client-config.json

# Test upload functionality
test-upload: $(CLIENT_BINARY)
	@echo "Testing upload functionality..."
	mkdir -p test-data
	echo "Hello World" > test-data/file1.txt
	echo "Hello Go" > test-data/file2.txt
	mkdir -p test-data/subdir
	echo "Subdirectory file" > test-data/subdir/file3.txt
	./$(CLIENT_BINARY) upload test-data --verbose

# Test download functionality
test-download: $(CLIENT_BINARY)
	@echo "Testing download functionality..."
	./$(CLIENT_BINARY) download test-data ./downloaded --verbose

# Test listing functionality
test-list: $(CLIENT_BINARY)
	@echo "Testing list functionality..."
	./$(CLIENT_BINARY) list --verbose

# Health check
health-check: $(CLIENT_BINARY)
	@echo "Checking server health..."
	./$(CLIENT_BINARY) health

# Install binaries to system PATH (optional)
install: build
	@echo "Installing binaries to /usr/local/bin..."
	sudo cp $(SERVER_BINARY) /usr/local/bin/upload-http-server
	sudo cp $(CLIENT_BINARY) /usr/local/bin/upload-http-client

# Show build information
info:
	@echo "Go version: $(GO_VERSION)"
	@echo "Project: upload-http"
	@echo "Binaries will be built in: $(BINARY_DIR)/"

# Full test suite
test-all: build test-upload test-download test-list health-check
	@echo "All tests completed successfully!"

# Help
help:
	@echo "Available targets:"
	@echo "  build              - Build both server and client binaries"
	@echo "  clean              - Clean build artifacts"
	@echo "  test               - Run unit tests"
	@echo "  test-coverage      - Run tests with coverage report"
	@echo "  fmt                - Format code"
	@echo "  lint               - Lint code (requires golangci-lint)"
	@echo "  deps               - Install dependencies"
	@echo "  run-server         - Run server with default settings"
	@echo "  run-server-config  - Run server with config file"
	@echo "  generate-server-config - Generate server configuration file"
	@echo "  generate-client-config - Generate client configuration file"
	@echo "  test-upload        - Test upload functionality"
	@echo "  test-download      - Test download functionality"
	@echo "  test-list          - Test listing functionality"
	@echo "  health-check       - Check server health"
	@echo "  test-all           - Run complete test suite"
	@echo "  install            - Install binaries to system PATH"
	@echo "  info               - Show build information"
	@echo "  help               - Show this help message"