# Testing Scripts

This directory contains scripts for running Go tests in Ubuntu Docker containers to simulate the GitHub Actions environment locally.

## Why Test in Ubuntu Containers?

Because while developing on macOS, I'm not able to consistently catch or reproduce issues that only occur in a Linux environment. 
Running tests in an Ubuntu container helps identify these issues before they reach CI and before it drives me crazy.

## Available Scripts

### ubuntu-test.sh

The core script that runs tests in an Ubuntu Docker container.

```bash
# Run all tests
./scripts/ubuntu-test.sh ./...

# Run specific tests with race detection
./scripts/ubuntu-test.sh -race -v ./internal/lock/...

# Run tests matching a specific pattern
./scripts/ubuntu-test.sh -run=TestPattern ./...
```

To make this a bit more flexible, the script allows for customizing the environment with variables:
- `GO_VERSION` - Go version to use (default: "1.24")
- `DOCKER_IMG` - Docker image to use (default: "golang:1.24")
- `DOCKER_CMD` - Docker command to use (default: "docker")

Example of using non-defaults to test a specific Go version:
```bash
GO_VERSION=1.22 DOCKER_IMG=golang:1.22 ./scripts/ubuntu-test.sh ./...
```

### test-all.sh

Runs all tests in the project with race detection enabled.

```bash
# Run all tests with race detection
./scripts/test-all.sh

# Run all tests without race detection
./scripts/test-all.sh --no-race
```

## Performance Notes

The scripts create a Docker volume named `go-cache-</project/dir/of/pwd>` to cache Go modules between runs, significantly improving performance for later test runs.