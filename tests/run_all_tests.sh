#!/bin/bash
# Run all GitBak tests

TEST_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

echo "Running all GitBak tests..."
echo "============================"
echo ""

# Make sure all test scripts are executable
chmod +x "$TEST_DIR"/*.sh

# Run each test script one by one
for test in "$TEST_DIR"/basic_functionality.sh "$TEST_DIR"/lock_file.sh "$TEST_DIR"/continuation.sh "$TEST_DIR"/stress_test.sh "$TEST_DIR"/portability.sh; do
    echo "Running $(basename "$test")..."
    "$test"
    echo ""
    echo "----------------------------"
    echo ""
done

echo "All tests completed!"