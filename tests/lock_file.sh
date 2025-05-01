#!/bin/bash
# Test lock file functionality
echo "Test 2: Lock file functionality"
TEST_DIR=$(mktemp -d)
cd "$TEST_DIR"
git init
echo "Initial content" > test.txt
git add test.txt
git commit -m "Initial commit"

# Get the path to the gitbak script (one directory up from tests)
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/.."
GITBAK_SCRIPT="$SCRIPT_DIR/gitbak.sh"

# Start first instance
INTERVAL_MINUTES=60 DEBUG=true "$GITBAK_SCRIPT" &
GITBAK_PID1=$!
sleep 2

# Try to start second instance
INTERVAL_MINUTES=60 DEBUG=true "$GITBAK_SCRIPT" > output.txt 2>&1 &
GITBAK_PID2=$!
sleep 2

# Check if second instance detected the lock
if grep -q "Error: Another GitBak instance is already running" output.txt; then
    echo "✅ Lock file test passed: Second instance correctly detected lock"
else
    echo "❌ Lock file test failed: Second instance didn't detect lock"
    cat output.txt
fi

# Clean up
kill -TERM $GITBAK_PID1 2>/dev/null
kill -TERM $GITBAK_PID2 2>/dev/null
wait $GITBAK_PID1 2>/dev/null
wait $GITBAK_PID2 2>/dev/null

cd - > /dev/null
rm -rf "$TEST_DIR"