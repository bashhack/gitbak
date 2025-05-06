# Check for .env file and include it
ifneq (,$(wildcard .env))
	include .env
	export
endif

# Version information
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DATE    ?= $(shell date +%FT%T%z)

SCRIPT_NAME=gitbak.sh
TESTS_DIR=tests

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

## run: Run gitbak shell script
.PHONY: run
run:
	@echo "🚀 Running gitbak shell script..."
	@./$(SCRIPT_NAME)

## test: Run test suite
.PHONY: test
test:
	@echo 'Running all tests...'
	@./$(TESTS_DIR)/run_tests.sh

## test/specific: Run specific tests
.PHONY: test/specific
test/specific:
	@echo 'Running specific tests...'
	@echo 'Usage: make test/specific TESTS="test1.sh test2.sh"'
	@if [ -n "$(TESTS)" ]; then \
		./$(TESTS_DIR)/run_tests.sh $(TESTS); \
	else \
		echo "No tests specified. Use TESTS=\"test1.sh test2.sh\""; \
		exit 1; \
	fi

# ============================================================================= #
# QUALITY CONTROL
# ============================================================================= #

## check-shellcheck: Check if shellcheck is installed
.PHONY: check-shellcheck
check-shellcheck:
	@if command -v shellcheck > /dev/null; then \
		echo "✅ shellcheck is installed"; \
	else \
		echo "⚠️ shellcheck not found, static analysis will be skipped"; \
		echo "🔍 Install with: brew install shellcheck or apt-get install shellcheck"; \
		exit 1; \
	fi

## run-shellcheck: Run shellcheck on all shell scripts
.PHONY: run-shellcheck
run-shellcheck: check-shellcheck
	@echo 'Running shellcheck on all shell scripts...'
	@find . -type f -name "*.sh" -exec shellcheck {} \; || echo "Note: shellcheck found issues (exit code: $$?)"

## lint: Run linters
.PHONY: lint
lint:
	@echo 'Linting shell scripts...'
	@$(MAKE) run-shellcheck || echo "Skipping shellcheck"

## audit: Format and lint all code
.PHONY: audit
audit:
	@echo 'Auditing shell scripts...'
	@$(MAKE) lint

# ============================================================================= #
# INSTALLATION
# ============================================================================= #

## install: Install to ~/.local/bin
.PHONY: install
install:
	@echo "📦 Installing $(SCRIPT_NAME)..."
	@echo "Installing to ~/.local/bin (standard user location)"
	@mkdir -p $(HOME)/.local/bin
	@cp $(SCRIPT_NAME) $(HOME)/.local/bin/gitbak
	@chmod +x $(HOME)/.local/bin/gitbak
	@echo "✅ Installation complete!"
	@if [ "$(shell echo ":$$PATH:" | grep -c ":$(HOME)/.local/bin:")" -eq 0 ]; then \
		echo "⚠️  Please add ~/.local/bin to your PATH:"; \
		echo "   export PATH=\"$$HOME/.local/bin:\$$PATH\""; \
	fi

# ============================================================================= #
# CLEAN
# ============================================================================= #

## clean: Remove temporary files
.PHONY: clean
clean:
	@echo "Cleaning..."
	@find . -name "*.log" -type f -delete
	@find . -name "*.tmp" -type f -delete
	@echo "Clean complete."