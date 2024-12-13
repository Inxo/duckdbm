
# Makefile for building the DuckDB Migration Tool

BINARY_NAME=duckdbm
BUILD_DIR=build

.PHONY: all clean build

# Default target
all: build

# Build the binary
build:
	@echo "Building the binary..."
	@mkdir -p $(BUILD_DIR)
	@go build -ldflags="-extldflags=-static" -o $(BUILD_DIR)/$(BINARY_NAME) src/*.go
	@echo "Binary built at $(BUILD_DIR)/$(BINARY_NAME)"

# Clean the build directory
clean:
	@echo "Cleaning up..."
	@rm -rf $(BUILD_DIR)
	@echo "Cleaned."

test:
	@go test ./...

lint:
	@golangci-lint run -v --disable-all -E errcheck
