# Scripts

This directory contains utility scripts for the gitbak project.

- `ubuntu-test.sh` - Runs tests in an Ubuntu Docker container to simulate the GitHub Actions environment locally

## Ubuntu Test Script

### Usage

```bash
./scripts/ubuntu-test.sh [test-flags]
```

### Examples

```bash
# Run all tests
./scripts/ubuntu-test.sh ./...

# Run all tests with race detector
./scripts/ubuntu-test.sh -v -race ./...

# Run specific tests in a package
./scripts/ubuntu-test.sh -run=TestSpecific ./pkg/...
```

### Configuration

The script can be configured with environment variables:
- `GO_VERSION` - Go version to use (default: 1.24.0)
- `DOCKER_IMG` - Docker image to use (default: golang:1.24)
- `DOCKER_CMD` - Docker command to use (default: docker)

## Testing Your Code

For regular testing, please use the Makefile targets in the project root:

```bash
# Run unit tests
make test

# Run integration tests
make test/integration

# Run the full test suite with linting
make audit
```
