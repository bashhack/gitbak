#!/usr/bin/env bash
set -e
set -o pipefail

# Run all tests for gitbak
echo "Running unit tests..."
go test -v -tags=test ./...

echo "Running integration tests..."
GITBAK_INTEGRATION_TESTS=1 go test -v -tags=integration,test ./test/integration

echo "All tests passed!"
