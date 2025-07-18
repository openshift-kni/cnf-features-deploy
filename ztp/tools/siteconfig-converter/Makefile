# Makefile for siteconfig-converter tool

# Variables
BINARY_NAME = siteconfig-converter
GO_FILES = main.go convert.go
TEST_FILES = convert_test.go
GOOS ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)
EXAMPLE_FILE = samples/example-sno1.yaml
OUTPUT_DIR = ./output

# Default target
.PHONY: all
all: build

# Build the binary
.PHONY: build
build:
	go mod vendor
	go build -o $(BINARY_NAME) $(GO_FILES)

# Build for specific OS/ARCH
.PHONY: build-linux
build-linux:
	GOOS=linux GOARCH=amd64 go build -o $(BINARY_NAME)-linux-amd64 $(GO_FILES)

# Run tests
.PHONY: test
test:
	@echo "=== Running Go tests ==="
	go test -v .

# Run tests with coverage
.PHONY: test-coverage
test-coverage:
	@echo "=== Running Go tests with coverage ==="
	go test -v -cover .

# Run the tool with help
.PHONY: help
help: build
	./$(BINARY_NAME) -h

.PHONY: demo-convert
demo-convert: build
	@echo "=== Converting SiteConfig to ClusterInstance ==="
	mkdir -p $(OUTPUT_DIR)
	./$(BINARY_NAME) -f $(EXAMPLE_FILE) -d $(OUTPUT_DIR) -t siteconfig-operator/ai-cluster-templates-v1
	@echo ""
	@echo "=== Generated ClusterInstance files ==="
	ls -la $(OUTPUT_DIR)/
	@echo ""
	@echo "=== ClusterInstance content preview ==="
	head -20 $(OUTPUT_DIR)/*.yaml

# Clean build artifacts and demo output
.PHONY: clean
clean:
	rm -f $(BINARY_NAME)
	rm -f $(BINARY_NAME)-*
	rm -rf $(OUTPUT_DIR)

# Format code
.PHONY: fmt
fmt:
	go fmt ./...

# Run linter
.PHONY: lint
lint:
	golangci-lint run

# Download dependencies
.PHONY: deps
deps:
	go mod download
	go mod tidy

