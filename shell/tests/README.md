# gitbak Test Suite

This directory contains tests for the gitbak script.

## Running Tests

Use `run_tests.sh` to run all the tests:

```bash
# Run all tests
./run_tests.sh

# Run specific tests
./run_tests.sh basic_functionality.sh stress_test.sh
```

## Available Tests

The test suite includes the following tests (run in this order):

1. **shell_compatibility.sh** - Tests syntax compatibility across multiple shell environments (bash, dash, sh, zsh)
2. **lock_file.sh** - Tests the lock file mechanism that prevents multiple instances
3. **basic_functionality.sh** - Tests the core functionality of gitbak
4. **continuation.sh** - Tests continuing a gitbak session with existing commits
5. **stress_test.sh** - Tests gitbak under rapid commit conditions

## Adding New Tests

When adding new tests, follow these guidelines:

1. Create a new test script in the tests directory
2. Use `mktemp -d` to create a temporary directory for test isolation
3. Clean up temporary directories and processes at the end of the test
4. Add the test to the `ALL_TESTS` array in `run_tests.sh`
5. Make sure to follow the pattern of reporting clear success/failure messages

## Test Structure

Each test script should:

1. Create its own isolated test environment using a temporary directory
2. Clean up after itself (remove temp directories, kill processes)
3. Use clear "✅ Test passed" or "❌ Test failed" messages
4. Return 0 for success, non-zero for failure