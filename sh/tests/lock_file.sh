#!/bin/bash
echo "Test 2: Lock file functionality"

ORIGINAL_DIR=$(cd "$(dirname "$0")/.." && pwd)

TEST_DIR=$(mktemp -d)
cd "$TEST_DIR" || exit 1
git init
# Set git identity for tests
git config --local user.name "GitBak Test"
git config --local user.email "gitbak-test@example.com"
echo "Initial content" >test.txt
git add test.txt
git commit -m "Initial commit"

cp "$ORIGINAL_DIR/gitbak.sh" .
chmod +x gitbak.sh

INTERVAL_MINUTES=60 DEBUG=true CREATE_BRANCH=false ./gitbak.sh &
GITBAK_PID1=$!
sleep 2

INTERVAL_MINUTES=60 DEBUG=true CREATE_BRANCH=false ./gitbak.sh >output.txt 2>&1

if grep -q "Error: Another gitbak instance is already running" output.txt; then
    echo "✅ Lock file test passed: Second instance correctly detected lock"
    success=true
else
    echo "❌ Lock file test failed: Second instance didn't detect lock"
    cat output.txt
    success=false
fi

kill -TERM $GITBAK_PID1 2>/dev/null
for i in $(seq 1 5); do
    if ! ps -p $GITBAK_PID1 > /dev/null 2>&1; then
        break
    fi
    echo "Waiting for gitbak process to terminate ($i/5)..."
    sleep 1
done

if ps -p $GITBAK_PID1 > /dev/null 2>&1; then
    echo "Process didn't terminate gracefully, force killing..."
    kill -9 $GITBAK_PID1 2>/dev/null
fi

cd - >/dev/null || exit 1
rm -rf "$TEST_DIR"

if [ "$success" = true ]; then
    exit 0
else
    exit 1
fi
