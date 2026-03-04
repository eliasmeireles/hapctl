.PHONY: build clean install test lint fmt run version version-push help

BINARY_NAME=hapctl
BUILD_DIR=bin
GO=go
GOFLAGS=-v
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS=-ldflags "-s -w -X github.com/eliasmeireles/hapctl/internal/cmd.Version=$(VERSION)"

help:
	@echo "Available targets:"
	@echo "  build          - Build the binary"
	@echo "  clean          - Remove build artifacts"
	@echo "  install        - Install the binary to /usr/local/bin"
	@echo "  test           - Run tests"
	@echo "  lint           - Run linters"
	@echo "  fmt            - Format code"
	@echo "  run            - Run the agent with example config"
	@echo "  version        - Create a new version tag (usage: make version VERSION=v0.1.0)"
	@echo "  version-push   - Push version tag to trigger release"

build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	$(GO) build $(GOFLAGS) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/hapctl
	@mkdir -p .dev/multipass/.volumes
	@cp $(BUILD_DIR)/$(BINARY_NAME) .dev/multipass/.volumes/
	@cp ./.dev/multipass/instance-binary-update.sh .dev/multipass/.volumes/

clean:
	@echo "Cleaning..."
	@rm -rf $(BUILD_DIR)
	@$(GO) clean

install: build
	@echo "Installing $(BINARY_NAME) to /usr/local/bin..."
	@sudo cp $(BUILD_DIR)/$(BINARY_NAME) /usr/local/bin/
	@sudo chmod +x /usr/local/bin/$(BINARY_NAME)
	@echo "Installed successfully"

test:
	@echo "Running tests..."
	$(GO) test -v -race -coverprofile=coverage.out ./...
	$(GO) tool cover -html=coverage.out -o coverage.html

lint:
	@echo "Running linters..."
	@which golangci-lint > /dev/null || (echo "golangci-lint not found, install it from https://golangci-lint.run/usage/install/" && exit 1)
	golangci-lint run ./...

fmt:
	@echo "Formatting code..."
	$(GO) fmt ./...
	@which goimports > /dev/null && goimports -w . || echo "goimports not found, skipping"

run: build
	@echo "Running $(BINARY_NAME) agent..."
	@sudo $(BUILD_DIR)/$(BINARY_NAME) agent --config examples/config.yaml

validate-example:
	@echo "Validating example configurations..."
	$(BUILD_DIR)/$(BINARY_NAME) validate -f examples/tcp-bind.yaml
	$(BUILD_DIR)/$(BINARY_NAME) validate -f examples/http-bind.yaml

apply-example:
	@echo "Applying example TCP bind..."
	@sudo $(BUILD_DIR)/$(BINARY_NAME) apply -f examples/tcp-bind.yaml

build-webhook-test:
	@echo "Building webhook-test..."
	@mkdir -p $(BUILD_DIR)
	cd .dev/webhook-test && $(GO) build $(GOFLAGS) $(LDFLAGS) -o ../../$(BUILD_DIR)/webhook-test .
	@echo "✅ webhook-test built successfully"

version:
	@if [ -z "$(VERSION)" ]; then \
		echo "Error: VERSION is required"; \
		echo "Usage: make version VERSION=v0.1.0"; \
		exit 1; \
	fi
	@./scripts/version.sh $(VERSION)

version-push:
	@if [ -z "$(VERSION)" ]; then \
		echo "Error: VERSION is required"; \
		echo "Usage: make version-push VERSION=v0.1.0"; \
		exit 1; \
	fi
	@echo "Pushing tag $(VERSION)..."
	@git push origin $(VERSION)
	@echo "✅ Tag $(VERSION) pushed successfully!"
	@echo "GitHub Actions will build and publish the release"
	@echo "Check: https://github.com/eliasmeireles/hapctl/actions"
