.PHONY: all build test clean release

BINARY_NAME := neoncast
CMD_PATH := ./cmd/neoncast
BUILD_DIR := build

VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

LDFLAGS := -ldflags "-X neoncast/internal/version.Version=$(VERSION) -X neoncast/internal/version.Commit=$(COMMIT) -X neoncast/internal/version.BuildTime=$(BUILD_TIME) -s -w"

all: build

build:
	go build $(LDFLAGS) -o $(BINARY_NAME) $(CMD_PATH)

test:
	go test -v ./...

clean:
	rm -rf $(BINARY_NAME) $(BUILD_DIR)

release: clean
	mkdir -p $(BUILD_DIR)
	# Linux AMD64
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 $(CMD_PATH)
	# Linux ARM64
	GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 $(CMD_PATH)
	# Darwin AMD64
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 $(CMD_PATH)
	# Darwin ARM64
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 $(CMD_PATH)
	# Windows AMD64
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe $(CMD_PATH)
	@echo "Release binaries built in $(BUILD_DIR)/"
	@ls -la $(BUILD_DIR)/

run: build
	./$(BINARY_NAME)
