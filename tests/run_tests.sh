#!/bin/bash
# Run GitBak tests one by one with timeouts

TEST_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

echo "Running GitBak tests..."
echo "============================"
echo ""

# Make sure all test scripts are executable
chmod +x "$TEST_DIR"/*.sh

# Run portability test (should be fast)
echo "Running portability.sh..."
"$TEST_DIR/portability.sh"
echo "----------------------------"
echo ""

# Run lock file test with timeout
echo "Running lock_file.sh..."
timeout 45s "$TEST_DIR/lock_file.sh"
echo "----------------------------"
echo ""

# Run basic functionality test with timeout
echo "Running basic_functionality.sh..."
timeout 90s "$TEST_DIR/basic_functionality.sh"
echo "----------------------------"
echo ""

# Run continuation test with timeout
echo "Running continuation.sh..."
timeout 90s "$TEST_DIR/continuation.sh"
echo "----------------------------"
echo ""

# Run stress test with timeout
echo "Running stress_test.sh..."
timeout 60s "$TEST_DIR/stress_test.sh"
echo "----------------------------"
echo ""

echo "All tests completed!"