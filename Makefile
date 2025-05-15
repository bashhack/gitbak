# Check for .env file and include it
ifneq (,$(wildcard .env))
	include .env
	export
endif

# Version information
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DATE    ?= $(shell date +%FT%T%z)
LDVARS  := -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)
LDFLAGS := -ldflags "$(LDVARS)"

BINARY_NAME=gitbak
BUILD_DIR=build
CMD_DIR=cmd/gitbak

# ============================================================================= #
# HELPERS
# ============================================================================= #

## help: Print this help message
.PHONY: help
help:
	@echo 'Usage:'
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' |  sed -e 's/^/ /'

.PHONY: confirm
confirm:
	@echo -n 'Are you sure? [y/N] ' && read ans && [ $${ans:-N} = y ]

# ============================================================================= #
# DEVELOPMENT
# ============================================================================= #

## run: Run gitbak in development mode
.PHONY: run
run:
	@echo "🚀 Running gitbak in development mode (not installed to PATH)..."
	@echo "ℹ️  For system-wide use, run 'make install' first."
	@go run $(LDFLAGS) ./$(CMD_DIR)

# Determine optimal number of parallel tests (min of CPU count and 8)
# This ensures we don't overwhelm systems with many cores
PARALLEL_COUNT := $(shell cores=$$(nproc 2>/dev/null || sysctl -n hw.ncpu 2>/dev/null || echo 4); \
                     if [ $$cores -gt 8 ]; then echo 8; else echo $$cores; fi)

## test: Run test suite
.PHONY: test
test:
	@echo 'Running tests with the "test" build tag using $(PARALLEL_COUNT) parallel tests...'
	@go test -v -tags=test -parallel=$(PARALLEL_COUNT) ./...

## test/short: Run only fast tests, skipping slow or external tests
.PHONY: test/short
test/short:
	@echo 'Running short tests only with the "test" build tag using $(PARALLEL_COUNT) parallel tests...'
	@go test -short -tags=test -parallel=$(PARALLEL_COUNT) ./...

## test/verbose: Run tests with verbose output
.PHONY: test/verbose
test/verbose:
	@echo 'Running tests with verbose output and the "test" build tag using $(PARALLEL_COUNT) parallel tests...'
	@go test -v -tags=test -parallel=$(PARALLEL_COUNT) ./...

## test/integration: Run integration tests
.PHONY: test/integration
test/integration:
	@echo 'Running integration tests with "integration,test" build tags using $(PARALLEL_COUNT) parallel tests...'
	@GITBAK_INTEGRATION_TESTS=1 go test -v -tags=integration,test -parallel=$(PARALLEL_COUNT) -timeout 5m ./test/integration

## coverage: Run test suite with coverage
.PHONY: coverage
coverage:
	@echo 'Running tests with coverage and the "test" build tag using $(PARALLEL_COUNT) parallel tests...'
	@go test -coverprofile=coverage.txt -tags=test -parallel=$(PARALLEL_COUNT) ./...
	@# Filter out test helpers and mock files from coverage.txt
	@grep -v "_test_helpers.go:" coverage.txt | \
		grep -v "mock_.*_test.go:" | \
		grep -v "mocks.go:" | \
		grep -v "test_helpers.go:" | \
		grep -v "setupTestRepo" | \
		grep -v "main" | \
		grep -v "setupTestGitbak" > coverage_filtered.txt
	@mv coverage_filtered.txt coverage.txt
	@go tool cover -html=coverage.txt -o coverage.html
	@echo "Coverage report generated at coverage.html"

## coverage/func: Show function-level coverage statistics
.PHONY: coverage/func
coverage/func:
	@echo 'Generating function-level coverage report with the "test" build tag using $(PARALLEL_COUNT) parallel tests...'
	@go test -coverprofile=coverage.txt -tags=test -parallel=$(PARALLEL_COUNT) ./...
	@# Filter out test helpers and mock files from coverage.txt
	@grep -v "_test_helpers.go:" coverage.txt | \
		grep -v "mock_.*_test.go:" | \
		grep -v "mocks.go:" | \
		grep -v "test_helpers.go:" | \
		grep -v "setupTestRepo" | \
		grep -v "main" | \
		grep -v "setupTestGitbak" > coverage_filtered.txt
	@mv coverage_filtered.txt coverage.txt
	@go tool cover -func=coverage.txt

# ============================================================================= #
# QUALITY CONTROL
# ============================================================================= #

## check-staticcheck: Check if staticcheck is installed
.PHONY: check-staticcheck
check-staticcheck:
	@if command -v staticcheck > /dev/null; then \
		echo "✅ staticcheck is installed"; \
	else \
		echo "⚠️ staticcheck not found, static analysis will be skipped"; \
		echo "🔍 Install with: go install honnef.co/go/tools/cmd/staticcheck@latest"; \
		exit 1; \
	fi

## run-staticcheck: Run staticcheck if it exists
.PHONY: run-staticcheck
run-staticcheck: check-staticcheck
	@echo 'Running staticcheck with test build tag...'
	@staticcheck -tags=test ./... || echo "Note: staticcheck found issues (exit code: $$?)"

## check-golangci-lint: Check if golangci-lint is installed
.PHONY: check-golangci-lint
check-golangci-lint:
	@if command -v golangci-lint > /dev/null; then \
		echo "✅ golangci-lint is installed"; \
	else \
		echo "⚠️ golangci-lint not found, external linting will be skipped"; \
		echo "🔍 Install with: brew install golangci-lint or go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
		exit 1; \
	fi

## run-golangci-lint: Run golangci-lint if it exists
.PHONY: run-golangci-lint
run-golangci-lint: check-golangci-lint
	@echo 'Running golangci-lint with test build tag...'
	@golangci-lint run --build-tags=test ./... || echo "Note: golangci-lint found issues (exit code: $$?)"

## lint: Run linters
.PHONY: lint
lint:
	@echo 'Linting...'
	@echo 'Running go vet with test build tag...'
	@go vet -tags=test ./...
	@$(MAKE) run-golangci-lint || echo "Skipping external linting"
	@$(MAKE) run-staticcheck || echo "Skipping staticcheck"

## audit: Tidy dependencies and format, vet and test all code
.PHONY: audit
audit:
	@echo 'Tidying and verifying module dependencies...'
	go mod tidy
	go mod verify
	@echo 'Formatting code...'
	go fmt ./...
	@echo 'Running lint (with test build tag)...'
	@$(MAKE) lint
	@echo 'Running tests with race detection and the "test" build tag using $(PARALLEL_COUNT) parallel tests...'
	go test -race -vet=off -tags=test -parallel=$(PARALLEL_COUNT) ./...
	@echo 'Generating coverage report (excluding test helpers)...'
	@$(MAKE) coverage/func

## vendor: Tidy and vendor dependencies
.PHONY: vendor
vendor:
	@echo 'Tidying and verifying module dependencies...'
	go mod tidy
	go mod verify
	@echo 'Vendoring dependencies...'
	go mod vendor

# ============================================================================= #
# BUILD
# ============================================================================= #

## version: Print the current version
.PHONY: version
version:
	@echo "Version: $(VERSION)"
	@echo "Commit: $(COMMIT)"
	@echo "Date: $(DATE)"

## build: Build for the current system
.PHONY: build
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	go mod tidy
	go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./$(CMD_DIR)

## build/optimize: Build optimized binary (smaller size)
.PHONY: build/optimize
build/optimize:
	@echo "Building optimized binary..."
	@mkdir -p $(BUILD_DIR)
	go build -ldflags "$(LDVARS) -s -w" -o $(BUILD_DIR)/$(BINARY_NAME) ./$(CMD_DIR)

## build/all: Build for all supported platforms
.PHONY: build/all
build/all:
	@echo "Building for all platforms..."
	@mkdir -p $(BUILD_DIR)/bin
	GOOS=darwin GOARCH=amd64 go build -ldflags "$(LDVARS) -s -w" -o $(BUILD_DIR)/bin/$(BINARY_NAME)-darwin-amd64 ./$(CMD_DIR)
	GOOS=darwin GOARCH=arm64 go build -ldflags "$(LDVARS) -s -w" -o $(BUILD_DIR)/bin/$(BINARY_NAME)-darwin-arm64 ./$(CMD_DIR)
	GOOS=linux GOARCH=amd64 go build -ldflags "$(LDVARS) -s -w" -o $(BUILD_DIR)/bin/$(BINARY_NAME)-linux-amd64 ./$(CMD_DIR)
	GOOS=linux GOARCH=arm64 go build -ldflags "$(LDVARS) -s -w" -o $(BUILD_DIR)/bin/$(BINARY_NAME)-linux-arm64 ./$(CMD_DIR)

## install: Install to ~/.local/bin
.PHONY: install
install: build/optimize
	@echo "📦 Installing $(BINARY_NAME)..."
	@echo "Installing to ~/.local/bin (standard user location)"
	@mkdir -p $(HOME)/.local/bin
	@cp $(BUILD_DIR)/$(BINARY_NAME) $(HOME)/.local/bin/
	@chmod +x $(HOME)/.local/bin/$(BINARY_NAME)
	@echo "✅ Installation complete!"
	@if [[ ":$$PATH:" != *":$(HOME)/.local/bin:"* ]]; then \
		echo "⚠️  Please add ~/.local/bin to your PATH:"; \
		echo "   export PATH=\"$$HOME/.local/bin:\$$PATH\""; \
	fi

# ============================================================================= #
# RELEASE
# ============================================================================= #

## release/snapshot: Run GoReleaser in snapshot mode (for testing)
.PHONY: release/snapshot
release/snapshot:
	@echo "Running GoReleaser in snapshot mode..."
	@if command -v goreleaser > /dev/null; then \
		goreleaser release --snapshot --clean; \
	else \
		echo "goreleaser not found, skipping"; \
		echo "Install with: brew install goreleaser"; \
	fi

## release: Run GoReleaser to create a production release (requires a git tag)
.PHONY: release
release: confirm
	@echo "Running GoReleaser to create a production release..."
	@if command -v goreleaser > /dev/null; then \
		goreleaser release --clean; \
	else \
		echo "goreleaser not found, skipping"; \
		echo "Install with: brew install goreleaser"; \
	fi

## release/check: Check if the GoReleaser config is valid
.PHONY: release/check
release/check:
	@echo "Checking GoReleaser configuration..."
	@if command -v goreleaser > /dev/null; then \
		goreleaser check; \
	else \
		echo "goreleaser not found, skipping"; \
		echo "Install with: brew install goreleaser"; \
	fi

## publish/gopkg: Publish to go.pkg.dev (triggers indexing)
.PHONY: publish/gopkg
publish/gopkg:
	@echo "Publishing to go.pkg.dev..."
	@if [ -z "$(VERSION)" ]; then \
		echo "Error: VERSION is not set. Please specify a version (e.g., make publish/gopkg VERSION=v1.2.3)"; \
		exit 1; \
	fi
	@echo "Triggering go.pkg.dev indexing for version $(VERSION)..."
	@GOPROXY=https://proxy.golang.org GO111MODULE=on go get github.com/bashhack/gitbak@$(VERSION)
	@echo "✅ Triggered indexing for go.pkg.dev"
	@echo "Note: It may take a few minutes for the package to appear on https://pkg.go.dev/github.com/bashhack/gitbak@$(VERSION)"

# ============================================================================= #
# DOCUMENTATION
# ============================================================================= #

## docs/serve: Run documentation server for the project
.PHONY: docs/serve
docs/serve:
	@echo 'Serving documentation...'
	go install golang.org/x/pkgsite/cmd/pkgsite@latest
	pkgsite -http localhost:3030

# ============================================================================= #
# CLEAN
# ============================================================================= #

## clean: Remove build artifacts
.PHONY: clean
clean:
	@echo "Cleaning..."
	@rm -f coverage.out coverage.txt coverage.html
	@rm -rf $(BUILD_DIR)
	@echo "Clean complete."