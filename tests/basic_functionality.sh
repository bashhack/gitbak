#!/bin/bash
# Test basic functionality
echo "Test 1: Basic functionality"
set -x  # Enable debugging
TEST_DIR=$(mktemp -d)
cd "$TEST_DIR"
git init
echo "Initial content" > test.txt
git add test.txt
git commit -m "Initial commit"

# Get the path to the gitbak script (one directory up from tests)
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/.."
GITBAK_SCRIPT="$SCRIPT_DIR/gitbak.sh"

# Run GitBak with short intervals for testing
INTERVAL_MINUTES=1 DEBUG=true "$GITBAK_SCRIPT" &
GITBAK_PID=$!

# Make changes at regular intervals
sleep 10
echo "Change 1" >> test.txt
sleep 30
echo "Change 2" >> test.txt
sleep 30
echo "Change 3" >> test.txt
sleep 10

# Stop GitBak
kill -TERM $GITBAK_PID
wait $GITBAK_PID

# Verify commits
COMMIT_COUNT=$(git log --grep "\[GitBak\]" | grep -c "commit")
if [ "$COMMIT_COUNT" -ge 2 ]; then
    echo "✅ Basic functionality test passed: $COMMIT_COUNT commits created"
else
    echo "❌ Basic functionality test failed: Only $COMMIT_COUNT commits created"
fi

cd - > /dev/null
rm -rf "$TEST_DIR"