#!/bin/bash
#
# gitbak test runner - Sequential only
#
# Usage:
#   ./run_tests.sh           # Run all tests
#   ./run_tests.sh [test1] [test2] ...  # Run specific tests
#

TEST_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

ALL_TESTS=(
    "shell_compatibility.sh"
    "lock_file.sh"
    "basic_functionality.sh"
    "continuation.sh"
    "stress_test.sh"
)

if [ $# -gt 0 ]; then
    TESTS=("$@")
else
    TESTS=("${ALL_TESTS[@]}")
fi

cleanup_processes() {
    echo "Checking for lingering gitbak processes..."
    pgrep -f "gitbak.sh" | while read -r pid; do
        echo "Killing gitbak process: $pid"
        kill -TERM "$pid" 2>/dev/null || true
    done
    
    sleep 1
}

cleanup_and_exit() {
    echo ""
    echo "Interrupt received, stopping tests..."
    cleanup_processes
    echo "Tests aborted."
    exit 1
}

trap cleanup_and_exit INT TERM

echo "Running gitbak tests..."
echo "======================="
echo ""

cleanup_processes

chmod +x "$TEST_DIR"/*.sh

passed=0
failed=0
total=${#TESTS[@]}
total_start_time=$(date +%s)

for test in "${TESTS[@]}"; do
    if [[ "$test" != /* ]]; then
        test_path="$TEST_DIR/$test"
    else
        test_path="$test"
    fi
    
    test_name=$(basename "$test_path")
    
    echo "Running $test_name..."
    
    log_file=$(mktemp)
    
    start_time=$(date +%s)
    echo "  Started at $(date +"%H:%M:%S")"
    
    "$test_path" > "$log_file" 2>&1
    result=$?
    
    end_time=$(date +%s)
    runtime=$((end_time - start_time))
    echo "  Finished at $(date +"%H:%M:%S") (runtime: ${runtime}s)"
    
    if [ $result -eq 0 ]; then
        echo "✅ $test_name passed"
        
        # Look for success message in the output
        if grep -q "✅.*passed" "$log_file"; then
            grep "✅.*passed" "$log_file" | tail -n 1
        else
            # If no explicit success message found, show the last few lines...
            echo "  Last output lines:"
            tail -n 3 "$log_file" | sed 's/^/  /'
        fi
        
        passed=$((passed + 1))
    else
        echo "❌ $test_name failed with exit code $result"
        echo "  Last 10 lines of output:"
        tail -n 10 "$log_file" | sed 's/^/  /'
        failed=$((failed + 1))
    fi
    
    rm -f "$log_file"
    
    echo "----------------------------"
    echo ""
    
    cleanup_processes
done

total_end_time=$(date +%s)
total_runtime=$((total_end_time - total_start_time))
minutes=$((total_runtime / 60))
seconds=$((total_runtime % 60))

echo ""
echo "=============================="
echo "       TEST SUMMARY           "
echo "=============================="
echo "  Total tests:  $total"
echo "  Passed:       $passed"
echo "  Failed:       $failed"
echo "  Total time:   ${minutes}m ${seconds}s"
echo "=============================="

if [ $failed -gt 0 ]; then
    echo "❌ Some tests failed"
    exit 1
else
    echo "✅ All tests passed"
    exit 0
fi