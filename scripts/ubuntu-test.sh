#!/usr/bin/env bash
# ubuntu-test.sh - Run Go tests in Ubuntu Docker container
#
# This script runs Go tests in an Ubuntu Docker container to simulate
# the GitHub Actions environment locally, catching platform-specific issues
# before they reach CI.
#
# Usage: ./scripts/ubuntu-test.sh [test-flags]
# Examples:
#   ./scripts/ubuntu-test.sh ./...                         # Run all tests
#   ./scripts/ubuntu-test.sh -v -race ./...                # Run all tests with race detector
#   ./scripts/ubuntu-test.sh -run=TestSpecific ./pkg/...   # Run specific tests in a package

set -e
set -o pipefail

# Configuration (can be overridden with environment variables)
GO_VERSION=${GO_VERSION:-"1.24.0"}
DOCKER_IMG=${DOCKER_IMG:-"golang:1.24"}  # Using official Go 1.24 image
DOCKER_CMD=${DOCKER_CMD:-"docker"}
PROJECT_DIR=$(pwd)
PROJECT_NAME=$(basename "$PROJECT_DIR")

# Pretty output helpers
GREEN="\033[0;32m"
RED="\033[0;31m"
YELLOW="\033[0;33m"
BLUE="\033[0;34m"
NC="\033[0m" # No Color

echo -e "${BLUE}🔍 Running tests for ${PROJECT_NAME} with Go ${GO_VERSION}...${NC}"

# Create or use module cache volume
CACHE_VOLUME="go-cache-${PROJECT_NAME}"
if ! $DOCKER_CMD volume inspect $CACHE_VOLUME >/dev/null 2>&1; then
  echo -e "${YELLOW}📦 Creating Go module cache volume: ${CACHE_VOLUME}${NC}"
  $DOCKER_CMD volume create $CACHE_VOLUME >/dev/null
fi

# Define common Docker options
DOCKER_OPTS=(
  --rm
  -v "${PROJECT_DIR}:/app"
  -v "${CACHE_VOLUME}:/go/pkg"
  -w /app
  -e CGO_ENABLED=1
  -e GOPATH=/go
  -e GO111MODULE=on
  -e HOME=/tmp
)

# Start time for duration calculation
start_time=$(date +%s)

# Run tests
echo -e "${YELLOW}🧪 Executing: go test $*${NC}"
if $DOCKER_CMD run "${DOCKER_OPTS[@]}" "${DOCKER_IMG}" sh -c "go test $*"; then
  exit_code=0
  status="${GREEN}✅ Tests passed!${NC}"
else
  exit_code=$?
  status="${RED}❌ Tests failed with exit code ${exit_code}${NC}"
fi

# Calculate duration
end_time=$(date +%s)
duration=$((end_time - start_time))
duration_text="Test duration: $duration seconds"

echo -e "$status (${duration_text})"

# Return the original exit code
exit $exit_code