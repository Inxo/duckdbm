
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
	@go build -o $(BUILD_DIR)/$(BINARY_NAME) src/main.go
	@echo "Binary built at $(BUILD_DIR)/$(BINARY_NAME)"

# Clean the build directory
clean:
	@echo "Cleaning up..."
	@rm -rf $(BUILD_DIR)
	@echo "Cleaned."


lint:
	@golangci-lint run -v --disable-all -E errcheck