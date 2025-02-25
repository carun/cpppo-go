.PHONY: build clean test lint run-example coverage

# Binary output
BINARY_NAME=cpppo-go
FANUC_BINARY=fanuc-example
PLC_BINARY=plc-example

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOLINT=golangci-lint
GOGET=$(GOCMD) get
GOFMT=$(GOCMD) fmt

# Build directory
BUILD_DIR=./build

# Build flags
BUILD_FLAGS=-v

# Default target
all: lint test build

# Build the application
build:
	mkdir -p $(BUILD_DIR)
	$(GOBUILD) $(BUILD_FLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/cpppo-go
	$(GOBUILD) $(BUILD_FLAGS) -o $(BUILD_DIR)/$(FANUC_BINARY) ./examples/fanuc
	$(GOBUILD) $(BUILD_FLAGS) -o $(BUILD_DIR)/$(PLC_BINARY) ./examples/plc

# Clean the build directory
clean:
	$(GOCLEAN)
	rm -rf $(BUILD_DIR)

# Run tests
test:
	$(GOTEST) -v ./...

# Run tests with coverage
coverage:
	$(GOTEST) -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out

# Format the code
fmt:
	$(GOFMT) ./...

# Lint the code
lint:
	$(GOLINT) run

# Install dependencies
deps:
	$(GOGET) -v ./...

# Run the example application
run-example:
	$(GOCMD) run ./examples/fanuc-example/main.go

# Generate the documentation
docs:
	$(GOCMD) doc -all -u ./...

# Install go tools
install-tools:
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(shell go env GOPATH)/bin v1.55.2
