#!/usr/bin/env bash
set -e  # Exit immediately if any command fails

echo "Test 1: Basic functionality"

ORIGINAL_DIR=$(cd "$(dirname "$0")/.." && pwd)

# Create timeout function to prevent test from hanging
test_timeout() {
  echo "TEST TIMEOUT REACHED - FORCE TERMINATING"
  # Kill any gitbak processes
  pgrep -f "gitbak.sh" | xargs kill -9 2>/dev/null || true
  exit 2
}

# Set 2-minute timeout
TIMEOUT_PID=0
(sleep 120; test_timeout) & 
TIMEOUT_PID=$!
trap 'kill $TIMEOUT_PID 2>/dev/null || true' EXIT

# Test execution
TEST_DIR=$(mktemp -d)
cd "$TEST_DIR" || exit 1
echo "Running test in $TEST_DIR"

echo "Initializing test git repository..."
git init
git config --local user.name "GitBak Test"
git config --local user.email "gitbak-test@example.com"

echo "Initial content" >test.txt
git add test.txt
git commit -m "Initial commit"

echo "Copying gitbak script..."
cp "$ORIGINAL_DIR/gitbak.sh" .
chmod +x gitbak.sh

echo "Starting gitbak..."
INTERVAL_MINUTES=1 DEBUG=true ./gitbak.sh >gitbak.log 2>&1 &
GITBAK_PID=$!
echo "Gitbak started with PID: $GITBAK_PID"

# Quick check to see if process is running
sleep 1
if ! ps -p $GITBAK_PID > /dev/null; then
    echo "ERROR: gitbak failed to start. Log output:"
    cat gitbak.log
    exit 1
fi

# Make changes to trigger commits
echo "Making first change..."
echo "Change 1" >>test.txt
sleep 15

echo "Making second change..."
echo "Change 2" >>test.txt
sleep 15

echo "Stopping gitbak..."
kill -TERM $GITBAK_PID 2>/dev/null || true

# Allow some time for clean termination
sleep 5

# Force kill if still running
if ps -p $GITBAK_PID > /dev/null 2>&1; then
    echo "Gitbak didn't terminate gracefully, force killing..."
    kill -9 $GITBAK_PID 2>/dev/null || true
fi

echo "Checking results..."
COMMIT_COUNT=$(git log --grep "\[gitbak\]" | grep -c "commit" || echo 0)
echo "Found $COMMIT_COUNT gitbak commits"

if [ "$COMMIT_COUNT" -ge 1 ]; then
    echo "✅ Basic functionality test passed: $COMMIT_COUNT commits created"
    cd - >/dev/null || exit 1
    rm -rf "$TEST_DIR"
    # Kill the timeout process
    kill $TIMEOUT_PID 2>/dev/null || true
    exit 0
else
    echo "❌ Basic functionality test failed: No commits created"
    echo "Gitbak log output:"
    cat gitbak.log
    cd - >/dev/null || exit 1
    rm -rf "$TEST_DIR"
    # Kill the timeout process
    kill $TIMEOUT_PID 2>/dev/null || true
    exit 1
fi